package supplychainapp

import (
	"context"
	"testing"
	"time"

	"task-processor/internal/authidentity"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	sheinmodel "task-processor/internal/marketplace/shein/model"
	temumodel "task-processor/internal/marketplace/temu/model"
	temurules "task-processor/internal/marketplace/temu/rules"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type temuRulesFixture struct {
	TemuRulesMerchant
	binding         storecenter.ProductMerchantBinding
	selection       temumodel.RuleSelection
	changed, hidden bool
	changeAfterRead bool
	reads           []string
}

func (f *temuRulesFixture) TemuMerchant(_ context.Context, scope collection.Scope, storeID string, expected *storecenter.ProductMerchantBinding) (TemuRulesMerchant, error) {
	if scope.OrganizationID != f.binding.OrganizationID || storeID != f.binding.StoreID || expected != nil && (f.changed || *expected != f.binding) {
		return nil, record.ErrForbidden
	}
	f.reads = append(f.reads, "authorize")
	return f, nil
}
func (f *temuRulesFixture) Binding() storecenter.ProductMerchantBinding { return f.binding }
func (f *temuRulesFixture) LocalCategories(_ context.Context, query temumodel.LocalCategoryQuery) ([]temumodel.LocalCategory, error) {
	f.reads = append(f.reads, "local-categories")
	if query.ParentCatID != f.selection.ParentCatID || query.Language != f.selection.Language {
		return nil, record.ErrInvalid
	}
	leaf, level, catType, status := true, int64(2), int64(0), int64(0)
	if f.hidden {
		status = 1
	}
	return []temumodel.LocalCategory{{CatID: f.selection.CatID, ParentID: &query.ParentCatID, Leaf: &leaf, Level: &level, CatType: &catType, AvailableStatus: &status}}, nil
}
func (f *temuRulesFixture) SupplierCategories(_ context.Context, query temumodel.SupplierCategoryQuery) ([]temumodel.SupplierCategory, error) {
	f.reads = append(f.reads, "supplier-categories")
	if query.ParentCatID != f.selection.ParentCatID || query.SiteID != f.selection.SupplierSiteID {
		return nil, record.ErrInvalid
	}
	leaf, level, catType, hidden, hiddenType := true, int64(2), int64(0), false, int64(0)
	return []temumodel.SupplierCategory{{CatID: f.selection.CatID, ParentCatID: &query.ParentCatID, Leaf: &leaf, Level: &level, CatType: &catType, Hidden: &hidden, HiddenType: &hiddenType}}, nil
}
func (f *temuRulesFixture) SupplierAttributes(_ context.Context, catID int64) (temumodel.SupplierAttributeTemplate, error) {
	f.reads = append(f.reads, "supplier-attributes")
	if catID != f.selection.CatID {
		return temumodel.SupplierAttributeTemplate{}, record.ErrInvalid
	}
	return temumodel.SupplierAttributeTemplate{Properties: []temumodel.SupplierProperty{}}, nil
}
func (f *temuRulesFixture) LocalFreightTemplates(context.Context) ([]temumodel.LocalFreightTemplate, error) {
	f.reads = append(f.reads, "freight")
	return []temumodel.LocalFreightTemplate{{ID: "current-freight", Name: "Current template"}}, nil
}
func (f *temuRulesFixture) LocalCategoryRules(_ context.Context, query temumodel.CategoryRuleQuery) (temumodel.LocalCategoryRules, error) {
	f.reads = append(f.reads, "field-rules")
	if query.CatID != f.selection.CatID || query.Language != f.selection.Language {
		return temumodel.LocalCategoryRules{}, record.ErrInvalid
	}
	return temumodel.LocalCategoryRules{FieldRules: []temumodel.LocalFieldRule{}}, nil
}
func (f *temuRulesFixture) LocalAttributes(_ context.Context, query temumodel.LocalAttributeQuery) (temumodel.LocalAttributeTemplate, error) {
	f.reads = append(f.reads, "attributes:"+query.CostTemplateID)
	if query.CatID != f.selection.CatID || query.Language != f.selection.Language || query.CostTemplateID != f.selection.CostTemplateID {
		return temumodel.LocalAttributeTemplate{}, record.ErrInvalid
	}
	return temumodel.LocalAttributeTemplate{CatID: &query.CatID, Language: query.Language, Attributes: []temumodel.LocalAttribute{}}, nil
}
func (f *temuRulesFixture) LocalVariations(_ context.Context, query temumodel.CategoryRuleQuery) (temumodel.LocalVariationTemplate, error) {
	f.reads = append(f.reads, "variations")
	if f.changeAfterRead {
		f.binding.ConnectionRevision++
	}
	return temumodel.LocalVariationTemplate{CatID: &query.CatID, Language: query.Language, Variations: []temumodel.LocalVariation{}}, nil
}

func TestTemuRuleReaderDiscardsMetadataWhenOriginalConnectionChanges(t *testing.T) {
	scope, fixture, reader := newTemuRuleFixture(t, storecenter.MerchantLocal)
	fixture.changeAfterRead = true
	_, snapshot, err := reader.ReadTemuTargetRules(context.Background(), scope, fixture.binding.StoreID, fixture.selection)
	require.ErrorIs(t, err, record.ErrNotReady)
	require.Nil(t, snapshot.LocalAttributes)
	require.Empty(t, snapshot.Observations, "a partial snapshot cannot combine metadata from another connection revision")
}

type temuQueryRevocableAuth struct {
	scope   collection.Scope
	revoked bool
}

func (a *temuQueryRevocableAuth) Authorize(context.Context, string) (collection.Scope, error) {
	if a.revoked {
		return collection.Scope{}, preparation.ErrForbidden
	}
	return a.scope, nil
}

type temuQueryRevokingReader struct {
	TemuTargetRuleReader
	authority *temuQueryRevocableAuth
}

func (r temuQueryRevokingReader) ReadTemuTargetRules(ctx context.Context, scope collection.Scope, storeID string, input temumodel.RuleSelection) (storecenter.ProductMerchantBinding, TemuTargetRules, error) {
	merchant, rules, err := r.TemuTargetRuleReader.ReadTemuTargetRules(ctx, scope, storeID, input)
	r.authority.revoked = true
	return merchant, rules, err
}

func TestSharedSupplyTemuRulesRejectsRevocationDuringOfficialRead(t *testing.T) {
	uploader, scope, records, _, _, _, _ := uploadFixture(t)
	_, fixture, reader := newTemuRuleFixture(t, storecenter.MerchantLocal)
	authority := &temuQueryRevocableAuth{scope: scope}
	app := Application{Sources: uploader.dependencies.Sources.(*preparation.SourceSelector), Products: uploader.dependencies.Products, Authorization: authority, TemuRules: temuQueryRevokingReader{TemuTargetRuleReader: reader, authority: authority}}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute)})
	input := record.TargetInput{SourceID: records.saved.Source.ID, StoreID: fixture.binding.StoreID, EffectiveVersion: records.saved.EffectiveVersion, TemuRules: &fixture.selection}
	view, err := app.QueryRules(ctx, input)
	require.ErrorIs(t, err, preparation.ErrForbidden)
	require.Nil(t, view.Temu)
	require.Empty(t, view.Merchant.StoreID)
}

func newTemuRuleFixture(t *testing.T, mode storecenter.OfficialMerchantMode) (collection.Scope, *temuRulesFixture, TemuRuleReader) {
	t.Helper()
	scope := collection.Scope{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}
	partition, site, region, revision := storecenter.PartitionEU, "temu-local", int64(20), "v1:temu:EU:manual_token:1:r1"
	if mode != storecenter.MerchantLocal {
		partition, site, region, revision = storecenter.PartitionCN, "temu-supplier", 0, "v1:temu:CN:manual_token:6:r1"
	}
	f := &temuRulesFixture{binding: storecenter.ProductMerchantBinding{Platform: storecenter.PlatformTemu, IdentityPartition: partition, MerchantMode: mode, RegionID: region, CredentialExpiresAt: time.Now().Add(time.Hour), VerificationRevision: "official-v1", ScopesHash: collection.Digest("scopes"), OrganizationID: scope.OrganizationID, StoreID: uuid.NewString(), Site: site, StoreVersion: 1, ConnectionRevision: 2, ApplicationID: "app-a", ApplicationRevision: revision, SupplierIdentityHash: collection.Digest("merchant"), ServiceExpiresAt: time.Now().Add(time.Hour)}, selection: temumodel.RuleSelection{ParentCatID: 12, CatID: 34, Language: "en", CostTemplateID: "current-freight"}}
	catalog, err := temurules.NewOfficialCatalog()
	require.NoError(t, err)
	return scope, f, TemuRuleReader{Stores: f, Catalog: catalog}
}

func TestTemuRuleReaderUsesVerifiedModeOriginalPartitionAndSelectedFreight(t *testing.T) {
	scope, f, reader := newTemuRuleFixture(t, storecenter.MerchantLocal)
	binding, snapshot, err := reader.ReadTemuTargetRules(context.Background(), scope, f.binding.StoreID, f.selection)
	require.NoError(t, err)
	require.Equal(t, f.binding, binding)
	require.True(t, snapshot.SelectableLeaf)
	require.NotNil(t, snapshot.LocalTarget)
	require.EqualValues(t, 142, snapshot.LocalTarget.Region.SiteID)
	require.Equal(t, "EU", snapshot.LocalTarget.ApplicationPartition, "Belgium's old GLO spec tag must not change original credential route")
	require.NotNil(t, snapshot.LocalAttributes)
	require.Equal(t, []string{"authorize", "local-categories", "freight", "field-rules", "attributes:current-freight", "variations", "authorize"}, f.reads)
	require.Len(t, snapshot.SpecificationSources, 3)
	require.NotZero(t, snapshot.ObservedAt)
}

func TestTemuRuleReaderKeepsMissingSiteSpecificationsWithoutClaimingReadiness(t *testing.T) {
	scope, f, reader := newTemuRuleFixture(t, storecenter.MerchantLocal)
	f.binding.RegionID = 75
	_, snapshot, err := reader.ReadTemuTargetRules(context.Background(), scope, f.binding.StoreID, f.selection)
	require.NoError(t, err)
	require.Contains(t, snapshot.LocalTarget.Issues, temurules.SpecificationsMissing)
	require.Empty(t, snapshot.LocalTarget.Specification.BaseCurrency)
	// Returning observable rule facts is not approval or publishing readiness.
}

func TestTemuRuleReaderDoesNotFetchDependentRulesForUnselectableCategoryOrUnverifiedFreight(t *testing.T) {
	scope, f, reader := newTemuRuleFixture(t, storecenter.MerchantLocal)
	f.hidden = true
	_, snapshot, err := reader.ReadTemuTargetRules(context.Background(), scope, f.binding.StoreID, f.selection)
	require.NoError(t, err)
	require.False(t, snapshot.SelectableLeaf)
	require.Nil(t, snapshot.LocalAttributes)
	require.NotContains(t, f.reads, "field-rules")
	f.hidden = false
	f.selection.CostTemplateID = "not-in-current-mall"
	_, snapshot, err = reader.ReadTemuTargetRules(context.Background(), scope, f.binding.StoreID, f.selection)
	require.NoError(t, err)
	require.Nil(t, snapshot.LocalAttributes)
	require.Contains(t, snapshot.Issues, "selected_freight_unverified")
	require.NotContains(t, f.reads, "attributes:not-in-current-mall")
}

func TestTemuRuleReaderSupplierModesDoNotInheritLocalCategoryOrFreightRules(t *testing.T) {
	for _, mode := range []storecenter.OfficialMerchantMode{storecenter.MerchantFullyManaged, storecenter.MerchantSemiManaged} {
		t.Run(string(mode), func(t *testing.T) {
			scope, f, reader := newTemuRuleFixture(t, mode)
			f.selection.Language, f.selection.CostTemplateID = "", ""
			if mode == storecenter.MerchantSemiManaged {
				f.selection.SupplierSiteID = 100
			}
			_, snapshot, err := reader.ReadTemuTargetRules(context.Background(), scope, f.binding.StoreID, f.selection)
			require.NoError(t, err)
			require.True(t, snapshot.SelectableLeaf)
			require.NotNil(t, snapshot.SupplierAttributes)
			require.Nil(t, snapshot.LocalTarget)
			require.Len(t, snapshot.SemiSites, 96)
			require.Equal(t, []string{"authorize", "supplier-categories", "supplier-attributes", "authorize"}, f.reads)
		})
	}
}

func TestTemuRuleReaderRejectsForeignStoreAndLocalSiteSelection(t *testing.T) {
	scope, f, reader := newTemuRuleFixture(t, storecenter.MerchantLocal)
	_, _, err := reader.ReadTemuTargetRules(context.Background(), scope, uuid.NewString(), f.selection)
	require.ErrorIs(t, err, record.ErrNotReady)
	require.Empty(t, f.reads)
	f.selection.SupplierSiteID = 100
	_, _, err = reader.ReadTemuTargetRules(context.Background(), scope, f.binding.StoreID, f.selection)
	require.ErrorIs(t, err, record.ErrInvalid)
	require.Equal(t, []string{"authorize"}, f.reads, "a client site must not switch the original local mall")
}

func TestSharedSupplyQueryRulesConsumesTemuFactsFromOriginalAuthorizedSource(t *testing.T) {
	uploader, scope, records, _, _, _, auth := uploadFixture(t)
	_, fixture, reader := newTemuRuleFixture(t, storecenter.MerchantLocal)
	app := Application{Sources: uploader.dependencies.Sources.(*preparation.SourceSelector), Products: uploader.dependencies.Products, Authorization: auth, TemuRules: reader}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute)})
	input := record.TargetInput{SourceID: records.saved.Source.ID, StoreID: fixture.binding.StoreID, EffectiveVersion: records.saved.EffectiveVersion, TemuRules: &fixture.selection}
	view, err := app.QueryRules(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, view.Temu)
	require.Equal(t, fixture.binding, view.Merchant)
	require.EqualValues(t, 142, view.Temu.LocalTarget.Region.SiteID)
	require.Empty(t, view.Rules.ApplicationMode)
	fixture.reads = nil
	input.SourceID = uuid.NewString()
	_, err = app.QueryRules(ctx, input)
	require.ErrorIs(t, err, preparation.ErrNotFound)
	require.Empty(t, fixture.reads, "no merchant metadata may be obtained for an unauthorized source")
	input.SourceID = records.saved.Source.ID
	input.Draft = goods.OfficialDraftInput{Product: sheinmodel.PublishProduct{CategoryID: 123}}
	_, err = app.QueryRules(ctx, input)
	require.ErrorIs(t, err, record.ErrInvalid, "explicit platform variants cannot be mixed")
	require.Empty(t, fixture.reads)
}
