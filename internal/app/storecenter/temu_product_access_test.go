package storecenterapp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/integration/temu"
	temumodel "task-processor/internal/marketplace/temu/model"
	"task-processor/internal/storecenter"
)

type temuRulesProvider struct {
	storecenter.OfficialConnectionProvider
	TemuRulesProvider
	application                  storecenter.OfficialApplication
	facts                        storecenter.OfficialVerificationFacts
	identityReads, metadataReads int
	deadline                     time.Time
	afterIdentity                func()
	afterMetadata                func()
}

func (p *temuRulesProvider) Application() storecenter.OfficialApplication { return p.application }
func (p *temuRulesProvider) QueryStore(context.Context, storecenter.OfficialMerchantCredential) (storecenter.OfficialStoreInformation, error) {
	p.identityReads++
	if p.afterIdentity != nil {
		p.afterIdentity()
	}
	return storecenter.OfficialStoreInformation{Verification: &p.facts}, nil
}
func (p *temuRulesProvider) QuerySupplierCategories(ctx context.Context, c storecenter.OfficialMerchantCredential, q temumodel.SupplierCategoryQuery) ([]temumodel.SupplierCategory, error) {
	p.metadataReads++
	p.deadline, _ = ctx.Deadline()
	if p.afterMetadata != nil {
		p.afterMetadata()
	}
	return []temumodel.SupplierCategory{{CatID: 34, Name: "Category"}}, nil
}
func (p *temuRulesProvider) QueryLocalCategories(ctx context.Context, c storecenter.OfficialMerchantCredential, q temumodel.LocalCategoryQuery) ([]temumodel.LocalCategory, error) {
	p.metadataReads++
	p.deadline, _ = ctx.Deadline()
	if p.afterMetadata != nil {
		p.afterMetadata()
	}
	return []temumodel.LocalCategory{{CatID: 34, Name: "Category"}}, nil
}

func TestTemuMetadataReadCannotReleaseFactsAfterAuthorityChangesDuringHTTP(t *testing.T) {
	for _, partition := range []storecenter.OfficialIdentityPartition{storecenter.PartitionCN, storecenter.PartitionGlobal} {
		for _, change := range []string{"grant", "connection", "credential", "service", "proof", "cancel"} {
			t.Run(string(partition)+"/"+change, func(t *testing.T) {
				mode := storecenter.MerchantLocal
				if partition == storecenter.PartitionCN {
					mode = storecenter.MerchantSemiManaged
				}
				a, reader, p, subject := newTemuRulesAccess(t, partition, mode)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				h, err := a.AuthorizeTemuRules(ctx, subject, "store-a", nil)
				require.NoError(t, err)
				p.afterMetadata = func() {
					switch change {
					case "grant":
						reader.err = storecenter.ErrNotFound
					case "connection":
						reader.material.Connection.Version++
					case "credential":
						reader.material.Attempt.Ciphertext = "changed"
					case "service":
						reader.material.ServiceExpiresAt = a.now()
					case "proof":
						expired := a.now().Add(6 * time.Second)
						a.now = func() time.Time { return expired }
					case "cancel":
						cancel()
					}
				}
				if partition == storecenter.PartitionCN {
					value, err := h.SupplierCategories(ctx, temumodel.SupplierCategoryQuery{})
					require.ErrorIs(t, err, ErrProductAccessChanged)
					require.Empty(t, value)
				} else {
					value, err := h.LocalCategories(ctx, temumodel.LocalCategoryQuery{})
					require.ErrorIs(t, err, ErrProductAccessChanged)
					require.Empty(t, value)
				}
			})
		}
	}
}

func newTemuRulesAccess(t *testing.T, partition storecenter.OfficialIdentityPartition, mode storecenter.OfficialMerchantMode) (*OfficialProductAccess, *productMaterialReader, *temuRulesProvider, storecenter.ProductExecutionSubject) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	app := storecenter.OfficialApplication{Platform: storecenter.PlatformTemu, IdentityPartition: partition, AuthorizationMethod: storecenter.AuthorizationManualToken, AppID: "app-a", Version: BoundTemuOfficialRevision("v1", partition), SupportedModes: storecenter.ModesLocal}
	facts := storecenter.OfficialVerificationFacts{Platform: storecenter.PlatformTemu, IdentityPartition: partition, Mode: mode, Namespace: storecenter.MerchantLocalNamespace, MerchantID: "9007199254740993", RegionID: 211, ExpiresAt: now.Add(time.Hour), Scopes: []string{"bg.local.goods.cats.get", "temu.local.goods.cats.rules.get"}, ContractRevision: "temu-identity-E08-E10-E11-E26-v1", VerifiedAt: now.Add(-time.Second)}
	if partition == storecenter.PartitionCN {
		app.SupportedModes = storecenter.ModesFullyManaged | storecenter.ModesSemiManaged
		facts.Namespace = storecenter.MerchantSupplierNamespace
		facts.MerchantID = "pa-mall-1"
		facts.RegionID = 0
		facts.Scopes = []string{"bg.goods.cats.get", "bg.goods.attrs.get"}
	}
	p := &temuRulesProvider{application: app, facts: facts}
	protection, err := temu.NewCredentialProtection("temu-key", make([]byte, 32))
	require.NoError(t, err)
	registry, err := NewOfficialApplicationRegistry([]OfficialApplicationRegistration{{Provider: p, Protection: protection}})
	require.NoError(t, err)
	encoded, err := json.Marshal(facts)
	require.NoError(t, err)
	attempt := storecenter.OfficialConnectionAttempt{OrganizationID: "org-a", StoreID: "store-a", AttemptID: "connection-a", ActorID: "actor-a", MemberID: "member-a", AppID: app.AppID, AppVersion: app.Version, StoreVersion: 1, ConnectionVersion: 1, Platform: app.Platform, IdentityPartition: app.IdentityPartition, AuthorizationMethod: app.AuthorizationMethod, StateHash: "original-state", State: "verified", VerificationJSON: string(encoded)}
	attempt.KeyID, attempt.Ciphertext, err = protection.Seal(attempt, storecenter.OfficialMerchantCredential{AppID: app.AppID, Temu: &storecenter.TemuMerchantCredential{AccessToken: strings.Repeat("synthetic-token+part/=", 20)}})
	require.NoError(t, err)
	hash, err := facts.MerchantIdentityHash()
	require.NoError(t, err)
	reader := &productMaterialReader{material: storecenter.ProductExecutionMaterial{Platform: app.Platform, StoreVersion: 1, ServiceExpiresAt: now.Add(time.Hour), Attempt: attempt, Connection: storecenter.OfficialConnectionView{Platform: app.Platform, IdentityPartition: partition, AuthorizationMethod: app.AuthorizationMethod, Mode: mode, RegionID: facts.RegionID, MerchantIdentityHash: hash, CredentialExpiresAt: &facts.ExpiresAt, AppID: app.AppID, AppRevision: app.Version, AttemptID: attempt.AttemptID, Version: 1, Status: storecenter.ConnectionStatusConnected, State: "verified"}}}
	access, err := NewOfficialProductAccess(reader, productLiveAccess{}, registry)
	require.NoError(t, err)
	access.now = func() time.Time { return now }
	return access, reader, p, storecenter.ProductExecutionSubject{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a", Purpose: storecenter.ProductPurposeRules}
}

func TestTemuRuleHandleVerifiesOriginalMerchantAndHasNoSerializableCredential(t *testing.T) {
	for _, tc := range []struct {
		partition storecenter.OfficialIdentityPartition
		mode      storecenter.OfficialMerchantMode
	}{
		{storecenter.PartitionCN, storecenter.MerchantFullyManaged}, {storecenter.PartitionCN, storecenter.MerchantSemiManaged}, {storecenter.PartitionUS, storecenter.MerchantLocal}, {storecenter.PartitionEU, storecenter.MerchantLocal}, {storecenter.PartitionGlobal, storecenter.MerchantLocal},
	} {
		t.Run(string(tc.partition)+string(tc.mode), func(t *testing.T) {
			a, _, p, subject := newTemuRulesAccess(t, tc.partition, tc.mode)
			h, err := a.AuthorizeTemuRules(context.Background(), subject, "store-a", nil)
			require.NoError(t, err)
			require.Equal(t, 1, p.identityReads)
			binding := h.Binding()
			require.Equal(t, storecenter.PlatformTemu, binding.Platform)
			require.Equal(t, tc.partition, binding.IdentityPartition)
			require.Equal(t, tc.mode, binding.MerchantMode)
			require.Len(t, binding.SupplierIdentityHash, 64)
			raw, err := json.Marshal(h)
			require.NoError(t, err)
			require.JSONEq(t, `{}`, string(raw))
			if tc.partition == storecenter.PartitionCN {
				_, err = h.SupplierCategories(context.Background(), temumodel.SupplierCategoryQuery{})
			} else {
				_, err = h.LocalCategories(context.Background(), temumodel.LocalCategoryQuery{})
			}
			require.NoError(t, err)
			require.Equal(t, 1, p.metadataReads)
			require.LessOrEqual(t, p.deadline.Sub(a.now()), 5*time.Second)
		})
	}
}

func TestTargetMerchantBindingSelectsVerifiedPlatformAndCannotFallback(t *testing.T) {
	for _, mode := range []storecenter.OfficialMerchantMode{storecenter.MerchantFullyManaged, storecenter.MerchantSemiManaged, storecenter.MerchantLocal} {
		partition := storecenter.PartitionCN
		if mode == storecenter.MerchantLocal {
			partition = storecenter.PartitionGlobal
		}
		a, reader, p, subject := newTemuRulesAccess(t, partition, mode)
		binding, err := a.ReadTargetMerchantBinding(context.Background(), subject, "store-a")
		require.NoError(t, err)
		require.Equal(t, mode, binding.MerchantMode)
		require.True(t, binding.ValidTemuApplication())
		require.Equal(t, 1, p.identityReads)
		reader.material.Platform = "unsupported"
		_, err = a.ReadTargetMerchantBinding(context.Background(), subject, "store-a")
		require.ErrorIs(t, err, ErrProductAccessChanged)
		require.Equal(t, 1, p.identityReads)
		reader.material.Platform = storecenter.PlatformTemu
		subject.Purpose = storecenter.ProductPurposePublish
		_, err = a.ReadTargetMerchantBinding(context.Background(), subject, "store-a")
		require.ErrorIs(t, err, ErrProductAccessChanged)
		require.Equal(t, 1, p.identityReads)
	}
}

func TestTemuRuleHandleRejectsChangedMembershipConnectionMerchantAndPurpose(t *testing.T) {
	a, reader, p, subject := newTemuRulesAccess(t, storecenter.PartitionCN, storecenter.MerchantSemiManaged)
	subject.Purpose = storecenter.ProductPurposePublish
	_, err := a.AuthorizeTemuRules(context.Background(), subject, "store-a", nil)
	require.ErrorIs(t, err, ErrProductAccessChanged)
	require.Zero(t, p.identityReads, "rules handle does not authorize submission")
	subject.Purpose = storecenter.ProductPurposeRules
	original := reader.material
	for _, change := range []func(){
		func() { reader.err = storecenter.ErrNotFound },
		func() { reader.material.Connection.Version++ },
		func() { reader.material.Attempt.Ciphertext = "different" },
		func() { reader.material.ServiceExpiresAt = a.now() },
		func() { reader.material.Connection.Mode = storecenter.MerchantFullyManaged },
		func() { reader.material.Attempt.IdentityPartition = storecenter.PartitionUS },
	} {
		reader.material = original
		reader.err = nil
		h, err := a.AuthorizeTemuRules(context.Background(), subject, "store-a", nil)
		require.NoError(t, err)
		change()
		_, err = h.SupplierCategories(context.Background(), temumodel.SupplierCategoryQuery{})
		require.ErrorIs(t, err, ErrProductAccessChanged)
		require.Zero(t, p.metadataReads)
	}
	reader.material = original
	reader.err = nil
	p.facts.MerchantID = "different-pa-mall"
	_, err = a.AuthorizeTemuRules(context.Background(), subject, "store-a", nil)
	require.ErrorIs(t, err, ErrProductAccessChanged)
	require.Zero(t, p.metadataReads)
}

func TestTemuRuleHandleRejectsPlatformMethodAndExpiredProofWithoutProviderCall(t *testing.T) {
	a, _, p, subject := newTemuRulesAccess(t, storecenter.PartitionCN, storecenter.MerchantFullyManaged)
	h, err := a.AuthorizeTemuRules(context.Background(), subject, "store-a", nil)
	require.NoError(t, err)
	_, err = h.LocalCategories(context.Background(), temumodel.LocalCategoryQuery{})
	require.ErrorIs(t, err, ErrProductAccessChanged)
	binding := h.Binding()
	binding.RegionID++
	_, err = a.AuthorizeTemuRules(context.Background(), subject, "store-a", &binding)
	require.ErrorIs(t, err, ErrProductAccessChanged)
	now := a.now().Add(5 * time.Second)
	a.now = func() time.Time { return now }
	_, err = h.SupplierCategories(context.Background(), temumodel.SupplierCategoryQuery{})
	require.ErrorIs(t, err, ErrProductAccessChanged)
	require.Zero(t, p.metadataReads)
}

func TestTemuRuleHandleCannotUseGrantRevokedDuringOfficialVerification(t *testing.T) {
	a, reader, p, subject := newTemuRulesAccess(t, storecenter.PartitionGlobal, storecenter.MerchantLocal)
	p.afterIdentity = func() { reader.err = storecenter.ErrNotFound }
	_, err := a.AuthorizeTemuRules(context.Background(), subject, "store-a", nil)
	require.ErrorIs(t, err, ErrProductAccessChanged)
	require.Zero(t, p.metadataReads)
}
