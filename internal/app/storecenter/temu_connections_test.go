package storecenterapp_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	storeapp "task-processor/internal/app/storecenter"
	"task-processor/internal/integration/temu"
	"task-processor/internal/storecenter"
)

type temuProvider struct {
	app        storecenter.OfficialApplication
	queries    int
	queryErr   error
	merchantID string
	mode       storecenter.OfficialMerchantMode
	tokens     []string
}

func (p *temuProvider) Application() storecenter.OfficialApplication { return p.app }
func (*temuProvider) AuthorizationURL(string) (string, error)        { return "", nil }
func (*temuProvider) Exchange(context.Context, string, string) (storecenter.OfficialMerchantCredential, error) {
	panic("manual token install must not exchange a code")
}
func (p *temuProvider) QueryStore(_ context.Context, credential storecenter.OfficialMerchantCredential) (storecenter.OfficialStoreInformation, error) {
	p.queries++
	p.tokens = append(p.tokens, credential.Temu.AccessToken)
	if p.queryErr != nil {
		return storecenter.OfficialStoreInformation{}, p.queryErr
	}
	return storecenter.OfficialStoreInformation{Verification: &storecenter.OfficialVerificationFacts{Platform: storecenter.PlatformTemu, IdentityPartition: p.app.IdentityPartition, Namespace: storecenter.MerchantSupplierNamespace, MerchantID: p.merchantID, Mode: p.mode, ExpiresAt: time.Now().UTC().Add(time.Hour), Scopes: []string{"bg.glo.goods.add"}, ContractRevision: "official-identity-v1", VerifiedAt: time.Now().UTC()}}, nil
}

func TestTemuInstallIsBoundToOriginalAttemptAndPrivateInputBeforeVerification(t *testing.T) {
	db, repo, access, _ := serviceFixture(t)
	s, err := storecenter.NewStore(storecenter.CreateStoreInput{ID: uuid.NewString(), OrganizationID: "org-a", ActorSubject: "operator", Name: "TEMU", Platform: "temu", Region: "CN", ExternalStoreID: "metadata-only", CreateIdempotencyKey: uuid.NewString(), OccurredAt: time.Now().UTC()})
	require.NoError(t, err)
	s, _, err = repo.CreateOrReplay(context.Background(), "org-a", s)
	require.NoError(t, err)
	p := &temuProvider{app: storecenter.OfficialApplication{AppID: "same-id", Version: "v1:temu:CN:manual_token:6:r1", Platform: storecenter.PlatformTemu, IdentityPartition: storecenter.PartitionCN, AuthorizationMethod: storecenter.AuthorizationManualToken, SupportedModes: storecenter.ModesFullyManaged | storecenter.ModesSemiManaged}, merchantID: "pa-123", mode: storecenter.MerchantFullyManaged, queryErr: storecenter.ErrOfficialConnectionUnavailable}
	protection, err := temu.NewCredentialProtection("test-key", make([]byte, 32))
	require.NoError(t, err)
	registry, err := storeapp.NewOfficialApplicationRegistry([]storeapp.OfficialApplicationRegistration{{Provider: p, Protection: protection}})
	require.NoError(t, err)
	app, err := storeapp.NewOfficialConnections(repo, registry)
	require.NoError(t, err)
	cmd := storecenter.OfficialConnectionCommand{OrganizationID: "org-a", StoreID: s.ID(), AttemptID: uuid.NewString(), ExpectedStoreVersion: s.Version(), ApplicationID: p.app.AppID, IdentityPartition: p.app.IdentityPartition}
	begin, err := app.Begin(context.Background(), cmd)
	require.NoError(t, err)
	require.Empty(t, begin.AuthorizationURL)
	input := storecenter.InstallOfficialToken{OrganizationID: "org-a", StoreID: s.ID(), AttemptID: cmd.AttemptID, AccessToken: "synthetic-token"}
	_, err = app.InstallToken(context.Background(), input)
	require.Error(t, err)
	require.Equal(t, 1, p.queries)
	sealed, err := repo.ReadOfficialInstallAttempt(context.Background(), "org-a", s.ID(), cmd.AttemptID)
	require.NoError(t, err)
	for _, mutate := range []func(*storecenter.OfficialConnectionAttempt){
		func(a *storecenter.OfficialConnectionAttempt) { a.StoreID = uuid.NewString() },
		func(a *storecenter.OfficialConnectionAttempt) { a.ActorID = "another-actor" },
		func(a *storecenter.OfficialConnectionAttempt) { a.MemberID = "another-membership" },
		func(a *storecenter.OfficialConnectionAttempt) {
			a.AuthorizationMethod = storecenter.AuthorizationCallback
		},
		func(a *storecenter.OfficialConnectionAttempt) { a.StateHash = "changed-binding" },
		func(a *storecenter.OfficialConnectionAttempt) { a.StoreVersion++ },
	} {
		changed := sealed
		mutate(&changed)
		_, err := repo.CompleteOfficialConnection(context.Background(), changed, storecenter.ConnectionStatusUnavailable, storecenter.OfficialStoreInformation{}, time.Now().UTC())
		require.ErrorIs(t, err, storecenter.ErrNotFound, "completion cannot alter the original immutable attempt binding")
	}
	// A failed read may safely resume the original sealed token, never install
	// a different input into the same idempotent operation.
	changed := input
	changed.AccessToken = "different-token"
	_, err = app.InstallToken(context.Background(), changed)
	require.ErrorIs(t, err, storecenter.ErrVersionConflict)
	require.Equal(t, 1, p.queries)
	p.queryErr = nil
	view, err := app.ResumeQuery(context.Background(), "org-a", s.ID(), cmd.AttemptID)
	require.NoError(t, err)
	require.Equal(t, storecenter.ConnectionStatusConnected, view.Status)
	require.Equal(t, 2, p.queries)
	require.Equal(t, []string{"synthetic-token", "synthetic-token"}, p.tokens)
	require.Equal(t, storecenter.MerchantFullyManaged, view.Mode)
	require.NotEmpty(t, view.MerchantIdentityHash)
	verified, err := repo.ReadOfficialInstallAttempt(context.Background(), "org-a", s.ID(), cmd.AttemptID)
	require.NoError(t, err)
	verified.MemberID = "another-membership"
	require.ErrorIs(t, repo.ObserveOfficialConnection(context.Background(), verified, storecenter.ConnectionStatusUnavailable, storecenter.OfficialStoreInformation{}, time.Now().UTC()), storecenter.ErrNotFound)
	public, err := json.Marshal(view)
	require.NoError(t, err)
	require.NotContains(t, string(public), "synthetic-token")
	var ciphertext string
	require.NoError(t, db.Table("workbench_store_connection_attempts").Select("ciphertext").Where("attempt_id = ?", cmd.AttemptID).Scan(&ciphertext).Error)
	require.NotEmpty(t, ciphertext)
	require.NotContains(t, ciphertext, "synthetic-token")
	access.member = "rejoined-member"
	_, err = app.InstallToken(context.Background(), input)
	require.Error(t, err)
	require.Equal(t, 2, p.queries)
	access.member = "member-a"
	p.merchantID = "different-pa-id"
	_, err = app.ResumeQuery(context.Background(), "org-a", s.ID(), cmd.AttemptID)
	require.Error(t, err, "revalidation cannot replace official merchant identity")
	var count int64
	require.NoError(t, db.Table("workbench_store_merchant_bindings").Where("organization_id = ? AND store_id = ?", "org-a", s.ID()).Count(&count).Error)
	require.Equal(t, int64(1), count)
	view, err = app.Read(context.Background(), "org-a", s.ID())
	require.NoError(t, err)
	require.Equal(t, storecenter.ConnectionStatusUnavailable, view.Status)
}

func TestTemuMerchantClaimSurvivesDisconnectApplicationTokenAndModeChanges(t *testing.T) {
	_, repo, access, _ := serviceFixture(t)
	newStore := func(org string) *storecenter.Store {
		s, err := storecenter.NewStore(storecenter.CreateStoreInput{ID: uuid.NewString(), OrganizationID: org, ActorSubject: "operator", Name: "TEMU", Platform: "temu", Region: "CN", ExternalStoreID: uuid.NewString(), CreateIdempotencyKey: uuid.NewString(), OccurredAt: time.Now().UTC()})
		require.NoError(t, err)
		s, _, err = repo.CreateOrReplay(context.Background(), org, s)
		require.NoError(t, err)
		return s
	}
	registrations := []storeapp.OfficialApplicationRegistration{}
	for _, id := range []string{"app-first", "app-another"} {
		p := &temuProvider{app: storecenter.OfficialApplication{AppID: id, Version: storeapp.BoundTemuOfficialRevision("v1", storecenter.PartitionCN), Platform: storecenter.PlatformTemu, IdentityPartition: storecenter.PartitionCN, AuthorizationMethod: storecenter.AuthorizationManualToken, SupportedModes: storecenter.ModesFullyManaged | storecenter.ModesSemiManaged}, merchantID: "same-pa-identity", mode: storecenter.MerchantSemiManaged}
		protection, err := temu.NewCredentialProtection(id, make([]byte, 32))
		require.NoError(t, err)
		registrations = append(registrations, storeapp.OfficialApplicationRegistration{Provider: p, Protection: protection})
	}
	registry, err := storeapp.NewOfficialApplicationRegistry(registrations)
	require.NoError(t, err)
	app, err := storeapp.NewOfficialConnections(repo, registry)
	require.NoError(t, err)
	connect := func(s *storecenter.Store, appid, token string) error {
		cmd := storecenter.OfficialConnectionCommand{OrganizationID: s.OrganizationID(), StoreID: s.ID(), AttemptID: uuid.NewString(), ExpectedStoreVersion: s.Version(), ApplicationID: appid, IdentityPartition: storecenter.PartitionCN}
		_, err := app.Begin(context.Background(), cmd)
		if err != nil {
			return err
		}
		_, err = app.InstallToken(context.Background(), storecenter.InstallOfficialToken{OrganizationID: s.OrganizationID(), StoreID: s.ID(), AttemptID: cmd.AttemptID, AccessToken: token})
		return err
	}
	original := newStore("org-a")
	require.NoError(t, connect(original, "app-first", "first-token"))
	original, err = repo.Get(context.Background(), "org-a", original.ID())
	require.NoError(t, err)
	disconnected, err := app.Disconnect(context.Background(), storecenter.OfficialConnectionCommand{OrganizationID: "org-a", StoreID: original.ID(), AttemptID: uuid.NewString(), ExpectedStoreVersion: original.Version()})
	require.NoError(t, err)
	require.Equal(t, storecenter.PlatformTemu, disconnected.Platform)
	registrations[1].Provider.(*temuProvider).mode = storecenter.MerchantFullyManaged
	access.member = "member-b"
	other := newStore("org-b")
	require.ErrorIs(t, connect(other, "app-another", "rotated-token"), storecenter.ErrAlreadyExists)
	view, err := app.Read(context.Background(), "org-b", other.ID())
	require.NoError(t, err)
	require.NotEqual(t, storecenter.ConnectionStatusConnected, view.Status)
	access.member = "member-a"
	original, err = repo.Get(context.Background(), "org-a", original.ID())
	require.NoError(t, err)
	require.NoError(t, connect(original, "app-another", "original-rotated-token"), "the original native Store retains ownership across applications and modes")
}
