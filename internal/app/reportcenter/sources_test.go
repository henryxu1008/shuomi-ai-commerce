package reportcenterapp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/listing/record"
	sheindraft "task-processor/internal/marketplace/shein/draft"
	sheinvalidator "task-processor/internal/marketplace/shein/validator"
	validator "task-processor/internal/marketplace/validator"
	productasset "task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/review"
	rc "task-processor/internal/reportcenter"
	"testing"
	"time"
)

type allowPolicy bool

func (p allowPolicy) Authorize(string, []string, string) bool { return bool(p) }

type reviewFunc func(context.Context, string) (review.View, error)

func (f reviewFunc) Get(c context.Context, id string) (review.View, error) { return f(c, id) }

type recordFunc func(context.Context, record.Actor, string) (record.Record, error)

func (f recordFunc) ReadOfflinePackage(c context.Context, a record.Actor, id string) (record.Record, error) {
	return f(c, a, id)
}
func TestSourceAuthorizationUsesTheVerifiedUserSubject(t *testing.T) {
	policy, err := authz.NewListingKitAuthorizer([]string{"configured-user"}, nil)
	require.NoError(t, err)
	calls := 0
	s := &Sources{Policy: policy, Reviews: reviewFunc(func(context.Context, string) (review.View, error) { calls++; return review.View{}, review.ErrNotFound }), Records: recordFunc(func(context.Context, record.Actor, string) (record.Record, error) {
		calls++
		return record.Record{}, record.ErrNotFound
	})}
	for _, user := range []string{"configured-user", "unconfigured-user"} {
		ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{UserID: user, TenantID: "org-a", EffectiveOrganizationID: "org-a", TokenExpiresAt: time.Now().Add(time.Hour)})
		for _, kind := range []string{"TITLE_REVIEW", "SHEIN_RECORD"} {
			_, err := s.Read(ctx, rc.Scope{OrganizationID: "org-a", ActorID: user}, kind, uuid.NewString())
			if user == "configured-user" {
				require.ErrorIs(t, err, rc.ErrNotFound)
			} else {
				require.ErrorIs(t, err, rc.ErrForbidden)
			}
		}
	}
	require.Equal(t, 2, calls)
}

func TestReviewSourceCapturesApplyAtSameRevisionAndEmptyBefore(t *testing.T) {
	scope := rc.Scope{OrganizationID: "org-a", ActorID: "actor-a"}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{UserID: scope.ActorID, TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, Roles: []string{"member"}, TokenExpiresAt: time.Now().Add(time.Hour)})
	id := uuid.NewString()
	view := review.View{ID: id, Owner: scope.ActorID, Input: review.CreateInput{ProductKey: "product-a", BaseVersion: 1}, Before: "", Title: "suggested title", Policy: "title-review-v1", State: "accepted", Revision: 2}
	require.NoError(t, review.ValidateView(view))
	s := Sources{Policy: allowPolicy(true), Reviews: reviewFunc(func(context.Context, string) (review.View, error) { return view, nil })}
	first, e := s.Read(ctx, scope, "TITLE_REVIEW", id)
	require.NoError(t, e)
	_, _, e = first.Bytes()
	require.NoError(t, e)
	require.Equal(t, "2:accepted", first.Ref.Version)
	view.State = "applied"
	view.Receipt = &review.Receipt{ProposalID: id, Revision: 2, ProductVersion: 2, PublicationID: uuid.NewString(), Actor: scope.ActorID, At: time.Now().UTC()}
	applied, e := s.Read(ctx, scope, "TITLE_REVIEW", id)
	require.NoError(t, e)
	_, _, e = applied.Bytes()
	require.NoError(t, e)
	require.Equal(t, "2:applied", applied.Ref.Version)
	require.Len(t, applied.Content.Sections, 2)
}

// Real domain creation, not equal fabricated package/diagnostic digests.
type offlineRecordFixture struct {
	saved record.Record
	at    time.Time
}

func (f *offlineRecordFixture) GetSnapshot(_ context.Context, id catalog.SnapshotIdentity, version uint64) (catalog.PublishedSnapshot, error) {
	return catalog.PublishedSnapshot{Identity: id, Version: version, Snapshot: catalog.ProductSnapshot{Title: "Synthetic offline bottle"}}, nil
}
func (f *offlineRecordFixture) GetApprovedInventory(_ context.Context, scope productasset.InventoryScope) (productasset.ApprovedAssetInventory, error) {
	return productasset.ApprovedAssetInventory{Scope: scope, Assets: []productasset.ApprovedAsset{{ID: "approved-main", RunID: "synthetic-run", PlanRevision: 1, SlotID: "main", Attempt: 1, Role: productasset.RoleMain, URL: "https://controlled.invalid/main.jpg"}}}, nil
}
func (f *offlineRecordFixture) GetStoreReference(_ context.Context, org, id string) (record.StoreReference, error) {
	return record.StoreReference{OrganizationID: org, StoreID: id, Platform: "shein"}, nil
}
func (f *offlineRecordFixture) FindOperation(context.Context, record.Actor, string) (record.Record, error) {
	return record.Record{}, record.ErrNotFound
}
func (f *offlineRecordFixture) Insert(_ context.Context, p record.Prepared) (record.Record, error) {
	f.saved = p.Record()
	f.saved.CreatedAt = f.at
	return f.saved.Clone(), nil
}
func (allowPolicy) IsTenantAdmin(string, []string) bool { return false }
func TestRecordSourceUsesSavedDiagnosticFromActualCreation(t *testing.T) {
	scope := rc.Scope{OrganizationID: "org-a", ActorID: "actor-a"}
	at := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{UserID: scope.ActorID, TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, Roles: []string{"member"}, TokenExpiresAt: time.Now().Add(time.Hour)})
	f := &offlineRecordFixture{at: at}
	service, err := record.NewService(record.ServiceDependencies{Products: f, Assets: f, Stores: f, Records: f, Builder: sheindraft.Builder{}, Evaluator: sheinvalidator.ExactApprovedAssetValidator{}, Authorizer: allowPolicy(true), Now: func() time.Time { return at }, RuleRevision: sheinvalidator.DiagnosticRuleVersion, PolicyRevision: sheinvalidator.BindingVersion})
	require.NoError(t, err)
	_, err = service.Create(ctx, "offline-source", record.Input{ProductKey: "product-a", SnapshotVersion: 1, StoreID: uuid.NewString(), Country: "US", Language: "en", Action: validator.SaveDraft})
	require.NoError(t, err)
	r := f.saved.Clone()
	var diagnostic validator.DiagnosticResult
	require.NoError(t, json.Unmarshal(r.Diagnostic, &diagnostic))
	require.NotEqual(t, r.PackageHash, diagnostic.Input.Digest)
	s := Sources{Policy: allowPolicy(true), Records: recordFunc(func(context.Context, record.Actor, string) (record.Record, error) {
		r.ReadAt = time.Now()
		return r.Clone(), nil
	})}
	first, err := s.Read(ctx, scope, "SHEIN_RECORD", r.ID)
	require.NoError(t, err)
	body, digest, err := first.Bytes()
	require.NoError(t, err)
	require.NotContains(t, string(body), "https://controlled.invalid/main.jpg", "full package must not leak into display snapshot")
	require.Contains(t, string(body), "not_evaluated")
	require.Equal(t, r.InputHash, first.Ref.Version)
	second, err := s.Read(ctx, scope, "SHEIN_RECORD", r.ID)
	require.NoError(t, err)
	_, nextDigest, err := second.Bytes()
	require.NoError(t, err)
	require.Equal(t, digest, nextDigest, "read time must not re-diagnose the saved result")
	original := r.Clone()
	r.RuleRevision = sheinvalidator.PreviousDiagnosticRuleVersion
	diagnostic.RuleVersion = r.RuleRevision
	diagnostic.Input.Digest, err = sheinvalidator.DiagnosticInputDigest(validator.BoundRequest[[]byte]{
		Input: r.Payload, Target: diagnostic.Target, Action: diagnostic.Action, RuleVersion: r.RuleRevision, BindingVersion: r.PolicyRevision,
	})
	require.NoError(t, err)
	r.Diagnostic, err = json.Marshal(diagnostic)
	require.NoError(t, err)
	r.DiagnosticHash = fmt.Sprintf("sha256:%x", sha256.Sum256(r.Diagnostic))
	historic, err := s.Read(ctx, scope, "SHEIN_RECORD", r.ID)
	require.NoError(t, err)
	historicBody, _, err := historic.Bytes()
	require.NoError(t, err)
	require.Contains(t, string(historicBody), sheinvalidator.PreviousDiagnosticRuleVersion)
	for _, mutate := range []func(){
		func() { r.Payload = append(r.Payload, ' ') },
		func() { r.Input.Action = validator.Publish },
		func() { r.PolicyRevision = "unsupported" },
		func() { r.RuleRevision = "unsupported" },
		func() { r.Diagnostic = append(r.Diagnostic, ' ') },
	} {
		r = original.Clone()
		mutate()
		_, err = s.Read(ctx, scope, "SHEIN_RECORD", r.ID)
		require.ErrorIs(t, err, rc.ErrUnavailable)
	}
}
func TestSourcesNeverInheritAdminOwnershipBypass(t *testing.T) {
	scope := rc.Scope{OrganizationID: "org-a", ActorID: "actor-a"}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{UserID: scope.ActorID, TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, Roles: []string{"listingkit_admin"}, TokenExpiresAt: time.Now().Add(time.Hour)})
	id := uuid.NewString()
	s := Sources{Policy: allowPolicy(true), Reviews: reviewFunc(func(context.Context, string) (review.View, error) { return review.View{ID: id, Owner: "other"}, nil }), Records: recordFunc(func(_ context.Context, a record.Actor, _ string) (record.Record, error) {
		require.Equal(t, scope.ActorID, a.UserID)
		return record.Record{ID: id, OrganizationID: scope.OrganizationID, OwnerUserID: "other"}, nil
	})}
	for _, kind := range []string{"TITLE_REVIEW", "SHEIN_RECORD"} {
		_, e := s.Read(ctx, scope, kind, id)
		require.ErrorIs(t, e, rc.ErrNotFound)
	}
	s.Policy = allowPolicy(false)
	_, e := s.Read(ctx, scope, "TITLE_REVIEW", id)
	require.ErrorIs(t, e, rc.ErrForbidden)
	s.Policy = allowPolicy(true)
	_, e = s.Read(ctx, rc.Scope{OrganizationID: "org-b", ActorID: scope.ActorID}, "TITLE_REVIEW", id)
	require.ErrorIs(t, e, rc.ErrForbidden)
	var _ authz.StaticAuthorizer = allowPolicy(true)
}
