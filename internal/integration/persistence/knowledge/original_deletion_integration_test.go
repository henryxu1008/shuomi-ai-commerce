//go:build integration

package knowledge_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	sdktypes "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	madmin "github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	knowledgedb "task-processor/internal/integration/persistence/knowledge"
	ledgerdb "task-processor/internal/integration/persistence/storagecapacity"
	objects3 "task-processor/internal/integration/s3"
	k "task-processor/internal/knowledge"
	"task-processor/internal/ledger/storagecapacity"
)

const fixtureBucket = "knowledge"

type putGate struct {
	next            http.RoundTripper
	count           atomic.Int64
	loss            atomic.Bool
	hold            atomic.Bool
	started, resume chan struct{}
}

func (g *putGate) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodPut || !strings.HasPrefix(req.URL.Path, "/knowledge/knowledge/") {
		return g.next.RoundTrip(req)
	}
	g.count.Add(1)
	if g.hold.Swap(false) {
		close(g.started)
		select {
		case <-g.resume:
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	}
	response, err := g.next.RoundTrip(req)
	if err == nil && response.StatusCode == http.StatusOK && g.loss.Swap(false) {
		response.Body.Close()
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Status: "503 Service Unavailable",
			Header: http.Header{"Content-Type": {"application/xml"}}, Body: io.NopCloser(strings.NewReader("<Error><Code>SlowDown</Code></Error>")), Request: req}, nil
	}
	return response, err
}

type deleteFault struct {
	*objects3.KnowledgeOriginalDeleter
	failReadback bool
	deleteCalls  int
}

func (f *deleteFault) DeleteOriginal(ctx context.Context, ns string, o k.Object) error {
	f.deleteCalls++
	if err := f.KnowledgeOriginalDeleter.DeleteOriginal(ctx, ns, o); err != nil {
		return err
	}
	f.failReadback = true
	return errors.New("fixture lost DELETE response")
}

func (f *deleteFault) InspectOriginal(ctx context.Context, ns string, o k.Object) (k.Inspection, error) {
	if f.failReadback {
		f.failReadback = false
		return k.Inspection{}, errors.New("fixture readback timeout")
	}
	return f.KnowledgeOriginalDeleter.InspectOriginal(ctx, ns, o)
}

type lostRelease struct {
	*ledgerdb.Repository
	lost    bool
	proof   storagecapacity.StoredDeletion
	receipt storagecapacity.ReleaseReceipt
}

func (l *lostRelease) ReleaseStored(ctx context.Context, proof storagecapacity.StoredDeletion) (storagecapacity.ReleaseReceipt, error) {
	receipt, err := l.Repository.ReleaseStored(ctx, proof)
	if err == nil && !l.lost {
		l.lost, l.proof, l.receipt = true, proof, receipt
		return storagecapacity.ReleaseReceipt{}, errors.New("fixture lost the committed release response")
	}
	return receipt, err
}

func originalFixtureAddress(t *testing.T, key string) string {
	t.Helper()
	addr := os.Getenv(key)
	if addr == "" {
		t.Skip("requires the disposable loopback fixture")
	}
	host, port, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", host)
	n, err := strconv.Atoi(port)
	require.NoError(t, err)
	require.Greater(t, n, 1024)
	return addr
}

func originalFixtureDB(t *testing.T, addr, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.Open("postgres://capacity_fixture:fixture-db-password@"+addr+"/"+name+"?sslmode=disable"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })
	return db
}

func fixturePolicy(t *testing.T, ctx context.Context, admin *madmin.AdminClient, id string, policies ...string) {
	t.Helper()
	require.NoError(t, admin.AddUser(ctx, id, "fixture-"+id+"-password"))
	for _, name := range policies {
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "deployments", "docker", "account-compose", name+"-policy.json"))
		require.NoError(t, err)
		require.NoError(t, admin.AddCannedPolicy(ctx, name, data))
	}
	_, err := admin.AttachPolicy(ctx, madmin.PolicyAssociationReq{User: id, Policies: policies})
	require.NoError(t, err)
}

func fixtureRuntimeDB(t *testing.T, pg, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.Open("postgres://knowledge_runtime:fixture-runtime-password@"+pg+"/"+name+"?sslmode=disable"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, knowledgedb.VerifyRuntime(context.Background(), db))
	return db
}

func verifyOriginalDeletionOff(t *testing.T, ctx context.Context, pg, namespace string, owner *gorm.DB, objects *objects3.KnowledgeStore) {
	t.Helper()
	baseline := originalFixtureDB(t, pg, "knowledge_baseline_fixture")
	require.NoError(t, knowledgedb.Install(ctx, baseline))
	require.NoError(t, knowledgedb.GrantRuntime(ctx, baseline))
	// Remove only the V1 additions in the two explicitly named disposable databases.
	require.NoError(t, baseline.WithContext(ctx).Exec("DROP TABLE public.knowledge_original_delete_keys,public.knowledge_original_deletions,public.knowledge_original_uploads,public.knowledge_original_generation").Error)
	commercial := originalFixtureDB(t, pg, "commercial_baseline_fixture")
	require.NoError(t, ledgerdb.AutoMigrate(commercial))
	require.NoError(t, commercial.WithContext(ctx).Exec(`ALTER TABLE public.storage_capacity_objects
		DROP COLUMN exclusive_binding, DROP COLUMN release_operation_id, DROP COLUMN release_proof_digest, DROP COLUMN released_at`).Error)
	baselineRuntime := fixtureRuntimeDB(t, pg, "knowledge_baseline_fixture")
	repo, err := knowledgedb.NewRepository(ctx, baselineRuntime)
	require.NoError(t, err)
	require.NoError(t, repo.ConfigureOriginalGeneration(ctx, ""))
	ledger, err := ledgerdb.New(commercial)
	require.NoError(t, err)
	service, err := k.NewService(repo, objects)
	require.NoError(t, err)
	service.ConfigureStorageCapacity(&storagecapacity.Meter{Writer: ledger, Namespace: namespace})
	scope := k.Scope{OrganizationID: "baseline-org", ActorID: "baseline-admin"}
	base, err := service.Mutate(ctx, k.Command{Scope: scope, Key: uuid.NewString(), Kind: "base_create", Name: "Baseline fixture"})
	require.NoError(t, err)
	uploaded, err := service.Upload(ctx, k.Command{Scope: scope, Key: uuid.NewString(), Kind: "source_create", Name: "Baseline source", BaseID: base.Base.ID}, "sample.txt", []byte("baseline"))
	require.NoError(t, err)
	require.Equal(t, k.ObjectStored, uploaded.Revision.State)
	summary, err := ledger.ReadSummary(ctx, scope.OrganizationID)
	require.NoError(t, err)
	require.Equal(t, "8", summary.OccupiedBytes)
	_, err = service.OriginalFile(ctx, scope, uploaded.Source.ID)
	require.ErrorIs(t, err, k.ErrNotFound)
	pool, err := baselineRuntime.DB()
	require.NoError(t, err)
	require.NoError(t, pool.Close())
	require.NoError(t, owner.WithContext(ctx).Exec("REVOKE CONNECT ON DATABASE knowledge_baseline_fixture FROM knowledge_runtime").Error)
	t.Log("feature off: baseline schema admits the narrow Knowledge runtime and meters a new upload")
}

func TestOriginalDeletionDisposablePostgresMinIO(t *testing.T) {
	pg := originalFixtureAddress(t, "LK1103_ISOLATED_PG")
	endpoint := originalFixtureAddress(t, "LK1103_ISOLATED_S3")
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
	defer cancel()
	db := originalFixtureDB(t, pg, "knowledge")
	require.NoError(t, knowledgedb.Install(ctx, db))
	for _, stmt := range []string{
		"CREATE DATABASE commercial_fixture", "CREATE DATABASE knowledge_baseline_fixture", "CREATE DATABASE commercial_baseline_fixture",
		"CREATE ROLE knowledge_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD 'fixture-runtime-password'",
		"REVOKE ALL ON DATABASE knowledge,commercial_fixture,knowledge_baseline_fixture,commercial_baseline_fixture,postgres,template1 FROM PUBLIC",
	} {
		require.NoError(t, db.WithContext(ctx).Exec(stmt).Error)
	}
	commercial := originalFixtureDB(t, pg, "commercial_fixture")
	require.NoError(t, ledgerdb.AutoMigrate(commercial))
	ledger, err := ledgerdb.New(commercial)
	require.NoError(t, err)

	rootCfg := objects3.ClientConfig{Region: "us-east-1", Endpoint: "http://" + endpoint, UsePathStyle: true, AccessKeyID: "fixture-root", SecretAccessKey: "fixture-minio-root-password"}
	root, err := objects3.NewKnowledgeClient(rootCfg)
	require.NoError(t, err)
	_, err = root.CreateBucket(ctx, &sdk.CreateBucketInput{Bucket: aws.String(fixtureBucket)})
	require.NoError(t, err)
	admin, err := madmin.NewWithOptions(endpoint, &madmin.Options{Creds: credentials.NewStaticV4(rootCfg.AccessKeyID, rootCfg.SecretAccessKey, "")})
	require.NoError(t, err)
	info, err := admin.ServerInfo(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, info.DeploymentID)

	writerCfg := rootCfg
	writerCfg.CapacityIdentityProvider, writerCfg.CapacityMinIODeploymentID = objects3.MinIOCapacityIdentityProvider, info.DeploymentID
	writerCfg.AccessKeyID, writerCfg.SecretAccessKey = "writer-fixture", "fixture-writer-fixture-password"
	oldWriter := writerCfg
	oldWriter.AccessKeyID, oldWriter.SecretAccessKey = "old-writer-fixture", "fixture-old-writer-fixture-password"
	fixturePolicy(t, ctx, admin, oldWriter.AccessKeyID, "knowledge-storage", "knowledge-capacity-read")
	fixturePolicy(t, ctx, admin, writerCfg.AccessKeyID, "knowledge-storage", "knowledge-capacity-read")
	deleteCfg := writerCfg
	deleteCfg.AccessKeyID, deleteCfg.SecretAccessKey = "delete-fixture", "fixture-delete-fixture-password"
	fixturePolicy(t, ctx, admin, deleteCfg.AccessKeyID, "knowledge-original-delete")
	ns, err := objects3.ResolveCapacityNamespace(ctx, writerCfg, fixtureBucket, "aws")
	require.NoError(t, err)
	require.NoError(t, ledger.EnsureNamespace(ctx, ns, "knowledge/", func(ctx context.Context) error {
		return objects3.VerifyEmptyCapacityPrefix(ctx, writerCfg, fixtureBucket, "knowledge/", "aws")
	}))
	deleter, err := objects3.NewKnowledgeOriginalDeleter(deleteCfg, fixtureBucket)
	require.NoError(t, err)
	require.NoError(t, deleter.VerifyNamespace(ctx, ns))
	wrongDeployment := deleteCfg
	wrongDeployment.CapacityMinIODeploymentID = uuid.NewString()
	unpinned, err := objects3.NewKnowledgeOriginalDeleter(wrongDeployment, fixtureBucket)
	require.NoError(t, err)
	require.Error(t, unpinned.VerifyNamespace(ctx, ns))
	missingBucket, err := objects3.NewKnowledgeOriginalDeleter(deleteCfg, "missing-bucket")
	require.NoError(t, err)
	require.Error(t, missingBucket.VerifyNamespace(ctx, ns))
	insufficient, err := objects3.NewKnowledgeOriginalDeleter(writerCfg, fixtureBucket)
	require.NoError(t, err)
	require.Error(t, insufficient.VerifyNamespace(ctx, ns))

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	t.Cleanup(transport.CloseIdleConnections)
	gate := &putGate{next: transport, started: make(chan struct{}), resume: make(chan struct{})}
	writerCfg.HTTPClient = &http.Client{Transport: gate, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	writer, err := objects3.NewClient(writerCfg)
	require.NoError(t, err)
	uploader, err := objects3.NewUploaderWithOptions(writer, objects3.UploaderOptions{Bucket: fixtureBucket, ArtifactCapabilities: objects3.ArtifactStorageCapabilities{Mode: objects3.ArtifactStorageModeAWS}})
	require.NoError(t, err)
	objects, err := objects3.NewKnowledgeStore(uploader)
	require.NoError(t, err)

	oldClient, err := objects3.NewKnowledgeClient(oldWriter)
	require.NoError(t, err)
	oldUploader, err := objects3.NewUploaderWithOptions(oldClient, objects3.UploaderOptions{Bucket: fixtureBucket, ArtifactCapabilities: objects3.ArtifactStorageCapabilities{Mode: objects3.ArtifactStorageModeAWS}})
	require.NoError(t, err)
	oldObjects, err := objects3.NewKnowledgeStore(oldUploader)
	require.NoError(t, err)
	verifyOriginalDeletionOff(t, ctx, pg, ns, db, oldObjects)
	require.NoError(t, knowledgedb.GrantRuntime(ctx, db))
	runtimeDB := fixtureRuntimeDB(t, pg, "knowledge")

	org := k.Scope{OrganizationID: "org-a", ActorID: "actor-a"}
	other := k.Scope{OrganizationID: "org-b", ActorID: "actor-b"}
	repo, err := knowledgedb.NewRepository(ctx, runtimeDB)
	require.NoError(t, err)
	require.NoError(t, repo.ConfigureOriginalGeneration(ctx, ""))
	meter := &storagecapacity.Meter{Writer: ledger, Namespace: ns}
	service, err := k.NewService(repo, oldObjects)
	require.NoError(t, err)
	service.ConfigureStorageCapacity(meter)
	processor, err := k.NewProcessor(repo, oldObjects, unusedKnowledgeParser{})
	require.NoError(t, err)
	processor.ConfigureStorageCapacity(meter)
	makeBase := func(scope k.Scope) string {
		result, e := service.Mutate(ctx, k.Command{Scope: scope, Key: uuid.NewString(), Kind: "base_create", Name: "Original deletion fixture"})
		require.NoError(t, e)
		return result.Base.ID
	}
	base, otherBase := makeBase(org), makeBase(other)
	command := func(scope k.Scope, base string) k.Command {
		return k.Command{Scope: scope, Key: uuid.NewString(), Kind: "source_create", BaseID: base, Name: "Fixture document"}
	}
	upload := func(cmd k.Command, data string) k.Result {
		result, e := service.Upload(ctx, cmd, "fixture.txt", []byte(data))
		require.NoError(t, e)
		require.Equal(t, k.ObjectStored, result.Revision.State)
		return result
	}
	disable := func(scope k.Scope, uploaded k.Result) k.Source {
		require.NoError(t, processor.Sweep(ctx))
		source, e := service.GetSource(ctx, scope, uploaded.Source.ID)
		require.NoError(t, e)
		require.Equal(t, k.Available, source.LatestRevision.State)
		result, e := service.Mutate(ctx, k.Command{Scope: scope, Kind: "source_disable", Key: uuid.NewString(), SourceID: source.ID, Version: source.Version})
		require.NoError(t, e)
		return *result.Source
	}
	assertCapacity := func(scope k.Scope, occupied int) {
		summary, e := ledger.ReadSummary(ctx, scope.OrganizationID)
		require.NoError(t, e)
		require.Equal(t, strconv.Itoa(occupied), summary.OccupiedBytes)
		require.Equal(t, "0", summary.ReservedBytes)
		require.Equal(t, strconv.FormatInt(5*1024*1024*1024-int64(occupied), 10), summary.RemainingBytes)
		if occupied == 0 {
			require.Empty(t, summary.Categories)
		} else {
			require.Equal(t, map[string]string{"knowledge_document": strconv.Itoa(occupied)}, summary.Categories)
		}
	}
	old := disable(org, upload(command(org, base), "before activation"))
	assertCapacity(org, 17)

	require.NoError(t, admin.SetUserStatus(ctx, oldWriter.AccessKeyID, madmin.AccountDisabled))
	retiredData := []byte("retired writer")
	retiredDigest := k.Digest(retiredData)
	retiredTarget := k.Object{Key: "knowledge/" + uuid.NewString() + "/" + retiredDigest, SHA256: retiredDigest, Data: retiredData, SizeBytes: int64(len(retiredData)), ContentType: "text/plain"}
	require.Error(t, oldObjects.PutImmutableOnce(ctx, retiredTarget), "old writer must lose its ability to PUT before activation")
	service, err = k.NewService(repo, objects)
	require.NoError(t, err)
	service.ConfigureStorageCapacity(meter)
	processor, err = k.NewProcessor(repo, objects, unusedKnowledgeParser{})
	require.NoError(t, err)
	processor.ConfigureStorageCapacity(meter)
	generation := uuid.NewString()
	require.NoError(t, db.WithContext(ctx).Exec("INSERT INTO public.knowledge_original_generation(singleton,generation) VALUES (true,?)", generation).Error)
	require.Error(t, repo.ConfigureOriginalGeneration(ctx, ""))
	require.NoError(t, repo.ConfigureOriginalGeneration(ctx, generation))
	require.NoError(t, service.ConfigureOriginalDeletion(deleter, true))
	_, err = service.OriginalFile(ctx, org, old.ID)
	require.ErrorIs(t, err, k.ErrNotFound)

	lost := command(org, base)
	gate.loss.Store(true)
	before := gate.count.Load()
	lostResult := upload(lost, "lost PUT")
	require.Equal(t, before+1, gate.count.Load(), "the SDK must not retry a committed PUT with a lost response")
	lostSource := disable(org, lostResult)
	_, err = service.OriginalFile(ctx, org, lostSource.ID)
	require.ErrorIs(t, err, k.ErrNotFound)

	late := command(org, base)
	gate.hold.Store(true)
	firstDone := make(chan error, 1)
	go func() {
		_, err := service.Upload(ctx, late, "fixture.txt", []byte("late PUT"))
		firstDone <- err
	}()
	select {
	case <-gate.started:
	case <-ctx.Done():
		t.Fatal("first PUT did not reach the wire gate")
	}
	require.NoError(t, db.WithContext(ctx).Exec(`UPDATE public.knowledge_revisions SET lease_until=clock_timestamp()-interval '1 second'
		WHERE organization_id=? AND id=(SELECT revision_id FROM public.knowledge_ingest_operations WHERE organization_id=? AND key=?)`, org.OrganizationID, org.OrganizationID, late.Key).Error)
	second := upload(late, "late PUT")
	lateSource := disable(org, second)
	_, err = service.OriginalFile(ctx, org, lateSource.ID)
	require.ErrorIs(t, err, k.ErrNotFound)
	close(gate.resume)
	require.NoError(t, <-firstDone)
	_, err = service.OriginalFile(ctx, org, lateSource.ID)
	require.ErrorIs(t, err, k.ErrNotFound)
	require.Equal(t, before+3, gate.count.Load(), "one lost PUT and exactly two racing PUT requests")
	t.Log("late PUT and lost PUT stayed ineligible, with the SDK wire count verified")

	_ = disable(other, upload(command(other, otherBase), "other org"))
	assertCapacity(other, 9)
	mainCommand := command(org, base)
	fileBytes := "permanently delete this sample file"
	mainResult := upload(mainCommand, fileBytes)
	source := disable(org, mainResult)
	physical := k.Object{SizeBytes: int64(len(fileBytes)), SHA256: k.Digest([]byte(fileBytes))}
	require.NoError(t, db.WithContext(ctx).Raw("SELECT object_key FROM public.knowledge_revisions WHERE organization_id=? AND id=?", org.OrganizationID, mainResult.Revision.ID).Scan(&physical.Key).Error)
	view, err := service.OriginalFile(ctx, org, source.ID)
	require.NoError(t, err)
	require.True(t, view.CanDelete)
	_, err = service.OriginalFile(ctx, other, source.ID)
	require.ErrorIs(t, err, k.ErrNotFound)
	_, err = writer.DeleteObject(ctx, &sdk.DeleteObjectInput{Bucket: aws.String(fixtureBucket), Key: aws.String(physical.Key)})
	require.Error(t, err, "writer cannot delete")
	deleteClient, err := objects3.NewKnowledgeClient(deleteCfg)
	require.NoError(t, err)
	_, err = deleteClient.PutObject(ctx, &sdk.PutObjectInput{Bucket: aws.String(fixtureBucket), Key: aws.String(physical.Key), Body: strings.NewReader(fileBytes)})
	require.Error(t, err, "deleter cannot write")
	assertCapacity(org, 33+len(fileBytes))
	deletionKey := uuid.NewString()
	authorize := func(context.Context) error { return nil }
	_, err = service.DeleteOriginal(ctx, org, source.ID, mainResult.Revision.ID, deletionKey, source.Version, func(context.Context) error { return k.ErrForbidden })
	require.ErrorIs(t, err, k.ErrForbidden)
	observed, err := deleter.InspectOriginal(ctx, ns, physical)
	require.NoError(t, err)
	require.True(t, observed.Exists, "revoked authority cannot dispatch DeleteObject")
	view, err = service.OriginalFile(ctx, org, source.ID)
	require.NoError(t, err)
	require.Equal(t, k.OriginalFenced, view.State)
	operationID := view.OperationID

	fault := &deleteFault{KnowledgeOriginalDeleter: deleter}
	require.NoError(t, service.ConfigureOriginalDeletion(fault, true))
	view, err = service.DeleteOriginal(ctx, org, source.ID, mainResult.Revision.ID, deletionKey, source.Version, authorize)
	require.NoError(t, err)
	require.Equal(t, operationID, view.OperationID)
	require.Equal(t, k.OriginalDispatched, view.State)
	require.Equal(t, 1, fault.deleteCalls)
	assertCapacity(org, 33+len(fileBytes))
	observed, err = deleter.InspectOriginal(ctx, ns, physical)
	require.NoError(t, err)
	require.False(t, observed.Exists, "authorized HEAD and version listing agree on physical absence")

	loss := &lostRelease{Repository: ledger}
	service.ConfigureStorageCapacity(&storagecapacity.Meter{Writer: loss, Namespace: ns})
	require.NoError(t, service.ConfigureOriginalDeletion(fault, true))
	view, err = service.ContinueOriginalDeletion(ctx, org, source.ID, operationID, authorize)
	require.NoError(t, err)
	require.Equal(t, k.OriginalAbsent, view.State)
	require.Equal(t, 1, fault.deleteCalls, "readback of absence prevents another DeleteObject")
	assertCapacity(org, 33)
	require.True(t, loss.lost)
	require.Equal(t, int64(len(fileBytes)), loss.receipt.Bytes)

	// Discard both Go repositories and reconnect after the response was lost.
	repo, err = knowledgedb.NewRepository(ctx, fixtureRuntimeDB(t, pg, "knowledge"))
	require.NoError(t, err)
	require.NoError(t, repo.ConfigureOriginalGeneration(ctx, generation))
	ledger, err = ledgerdb.New(originalFixtureDB(t, pg, "commercial_fixture"))
	require.NoError(t, err)
	service, err = k.NewService(repo, objects)
	require.NoError(t, err)
	service.ConfigureStorageCapacity(&storagecapacity.Meter{Writer: ledger, Namespace: ns})
	require.NoError(t, service.ConfigureOriginalDeletion(deleter, false))
	var retry sync.WaitGroup
	for range 6 {
		retry.Go(func() {
			_, err := service.ContinueOriginalDeletion(ctx, org, source.ID, operationID, authorize)
			if err != nil {
				t.Errorf("retry failed: %v", err)
			}
		})
	}
	retry.Wait()
	view, err = service.OriginalFile(ctx, org, source.ID)
	require.NoError(t, err)
	require.Equal(t, k.OriginalReleased, view.State)
	require.Equal(t, operationID, view.OperationID)
	sameReceipt, err := ledger.ReleaseStored(ctx, loss.proof)
	require.NoError(t, err)
	require.Equal(t, loss.receipt, sameReceipt)
	_, err = service.ContinueOriginalDeletion(ctx, other, source.ID, operationID, authorize)
	require.ErrorIs(t, err, k.ErrNotFound)
	_, err = service.DeleteOriginal(ctx, org, source.ID, mainResult.Revision.ID, deletionKey, source.Version+1, authorize)
	require.ErrorIs(t, err, k.ErrConflict)
	before = gate.count.Load()
	_, err = service.Upload(ctx, mainCommand, "fixture.txt", []byte(fileBytes))
	require.NoError(t, err)
	require.Equal(t, before, gate.count.Load(), "the original upload cannot recreate the released object")
	_, err = service.Preview(ctx, org, source.ID, mainResult.Revision.ID)
	require.Error(t, err)
	assertCapacity(org, 33)
	assertCapacity(other, 9)
	t.Logf("one physical DELETE; %d bytes released once, with identical receipt on concurrent retry and reconnect", len(fileBytes))

	// Objects absent before dispatch and changing namespace configuration cannot
	// create a deletion proof. These admin mutations are isolated fixture faults.
	require.NoError(t, service.ConfigureOriginalDeletion(deleter, true))
	missing := upload(command(org, base), "missing before delete")
	processor, err = k.NewProcessor(repo, objects, unusedKnowledgeParser{})
	require.NoError(t, err)
	processor.ConfigureStorageCapacity(&storagecapacity.Meter{Writer: ledger, Namespace: ns})
	missingSource := disable(org, missing)
	var missingKey string
	require.NoError(t, db.WithContext(ctx).Raw("SELECT object_key FROM public.knowledge_revisions WHERE organization_id=? AND id=?", org.OrganizationID, missing.Revision.ID).Scan(&missingKey).Error)
	_, err = root.DeleteObject(ctx, &sdk.DeleteObjectInput{Bucket: aws.String(fixtureBucket), Key: aws.String(missingKey)})
	require.NoError(t, err)
	view, err = service.DeleteOriginal(ctx, org, missingSource.ID, missing.Revision.ID, uuid.NewString(), missingSource.Version, authorize)
	require.NoError(t, err)
	require.Equal(t, k.OriginalFenced, view.State)
	assertCapacity(org, 54)

	_, err = root.PutBucketVersioning(ctx, &sdk.PutBucketVersioningInput{Bucket: aws.String(fixtureBucket), VersioningConfiguration: &sdktypes.VersioningConfiguration{Status: sdktypes.BucketVersioningStatusEnabled}})
	require.NoError(t, err)
	key := "knowledge/" + uuid.NewString() + "/" + k.Digest([]byte("versioned"))
	_, err = root.PutObject(ctx, &sdk.PutObjectInput{Bucket: aws.String(fixtureBucket), Key: aws.String(key), Body: strings.NewReader("versioned")})
	require.NoError(t, err)
	_, err = root.DeleteObject(ctx, &sdk.DeleteObjectInput{Bucket: aws.String(fixtureBucket), Key: aws.String(key)})
	require.NoError(t, err)
	_, err = deleter.InspectOriginal(ctx, ns, k.Object{Key: key, SHA256: k.Digest([]byte("versioned")), SizeBytes: 9})
	require.Error(t, err, "a delete marker cannot be absence proof")
	_, err = root.PutBucketVersioning(ctx, &sdk.PutBucketVersioningInput{Bucket: aws.String(fixtureBucket), VersioningConfiguration: &sdktypes.VersioningConfiguration{Status: sdktypes.BucketVersioningStatusSuspended}})
	require.NoError(t, err)
	require.Error(t, deleter.VerifyNamespace(ctx, ns))
	assertCapacity(org, 54)
	t.Log("proof rejects wrong namespace, denied authority, missing bucket, preexisting absence, and version markers")
}
