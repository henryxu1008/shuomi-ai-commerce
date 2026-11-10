package target

import (
	"context"
	"encoding/hex"
	"reflect"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

// LocalTrial is durable provenance, independent of one-shot import metadata.
// It is not a permission token and cannot be upgraded into official authority.
type LocalTrial struct {
	ContractVersion int                      `json:"contractVersion"`
	TrialID         string                   `json:"trialId"`
	SourceID        string                   `json:"sourceId"`
	CompletionDraft CompletionDraftSelection `json:"completionDraft"`
	StoreID         string                   `json:"storeId"`
	FixtureVersion  string                   `json:"fixtureVersion"`
	FixtureHash     string                   `json:"fixtureHash"`
}

func (v LocalTrial) Valid() bool {
	if raw, err := hex.DecodeString(v.FixtureHash); err != nil || len(raw) != 32 {
		return false
	}
	return v.ContractVersion == 1 && v.TrialID == "issue1158-local-draft-v1" && v.FixtureVersion == "issue1158-local-draft-v1" && len(v.FixtureHash) == 64 && collection.ValidID(v.SourceID) && collection.ValidID(v.StoreID) && collection.ValidID(v.CompletionDraft.RecordID) && v.CompletionDraft.Revision == 1
}

// LocalTrialPolicy is supplied only by the trusted composition root. Copies
// prevent mutation of the binding after construction. No provider is involved.
type LocalTrialPolicy struct {
	Scope          collection.Scope
	Binding        LocalTrial
	ProductKey     string
	ProductVersion uint64
	Enabled        bool
}

func (p LocalTrialPolicy) valid() bool {
	return p.Scope.Validate() == nil && p.Binding.Valid() && p.ProductKey != "" && p.ProductVersion == 1
}

func NewLocalTrialTargetService(d TargetDependencies, p LocalTrialPolicy) (*TargetService, error) {
	if !p.valid() || !p.Enabled || d.CompletionVersions == nil || d.ExecutionSources != nil || d.ExecutionAuthorization != nil || d.TemuRules != nil || d.TikTokRules != nil || d.WalmartRules != nil {
		return nil, ErrForbidden
	}
	s, err := NewTargetService(d)
	if err != nil {
		return nil, err
	}
	s.localTrial = &p
	return s, nil
}

// InitialSheinDraft is the existing source-to-draft projection. It copies no
// prices, stocks or image approvals; those require explicit owner workflows.
func InitialSheinDraft(product catalog.ProductSnapshot) goods.OfficialDraftInput {
	skc := model.ProductSKC{SKUs: []model.ProductSKU{}}
	for _, v := range product.Variants {
		skc.SKUs = append(skc.SKUs, model.ProductSKU{SupplierSKU: v.SKU, SaleAttributes: []model.AttributeValue{}})
	}
	if len(skc.SKUs) == 0 {
		skc.SKUs = []model.ProductSKU{{SaleAttributes: []model.AttributeValue{}}}
	}
	return goods.OfficialDraftInput{Product: model.PublishProduct{Names: []model.LanguageContent{{Language: "en", Name: product.Title}}, Descriptions: []model.LanguageContent{{Language: "en", Name: product.PlainDescription()}}, SKCs: []model.ProductSKC{skc}, Attributes: []model.AttributeValue{}}}
}

func (p LocalTrialPolicy) bootstrap(source preparation.SourceItem, product catalog.PublishedSnapshot) (TargetInput, error) {
	if source.ID != p.Binding.SourceID || source.Source.ProductKey != p.ProductKey || source.Source.Version != p.ProductVersion || product.Version != p.ProductVersion || product.Identity.TenantID != p.Scope.OrganizationID || product.Identity.ProductKey != p.ProductKey || len(product.Snapshot.Variants) != 3 {
		return TargetInput{}, ErrConflict
	}
	b := p.Binding
	draft := InitialSheinDraft(product.Snapshot)
	draft.Product.CategoryID = 123
	draft.Product.ProductTypeID = 456
	return TargetInput{LocalTrial: &b, SourceID: source.ID, StoreID: b.StoreID, EffectiveVersion: product.Version, Draft: draft}, nil
}

func (s *TargetService) LocalTrialBootstrap(ctx context.Context, sourceID, storeID string) (TargetInput, error) {
	if s == nil || s.localTrial == nil {
		return TargetInput{}, ErrForbidden
	}
	scope, err := s.authorizeTarget(ctx, preparation.PermissionManage)
	if err != nil || scope != s.localTrial.Scope || sourceID != s.localTrial.Binding.SourceID || storeID != s.localTrial.Binding.StoreID {
		return TargetInput{}, ErrForbidden
	}
	selected, err := s.dependencies.Sources.Select(ctx, sourceID)
	if err != nil {
		return TargetInput{}, err
	}
	owner, source, _, err := selected.Read(ctx)
	if err != nil || owner != scope {
		return TargetInput{}, ErrForbidden
	}
	if err = s.validateLocalCandidate(ctx, scope, source); err != nil {
		return TargetInput{}, err
	}
	product, err := s.dependencies.Products.ReadEffectiveTargetProduct(ctx, selected, s.localTrial.ProductVersion, "")
	if err != nil {
		return TargetInput{}, err
	}
	input, err := s.localTrial.bootstrap(source, product)
	if err != nil {
		return input, err
	}
	_, rules, err := s.dependencies.Rules.ReadTargetRules(ctx, scope, storeID, input.Draft)
	if err != nil || rules.Authority != "local_trial" || collection.Digest(rules) != s.localTrial.Binding.FixtureHash {
		return TargetInput{}, ErrNotReady
	}
	return input, nil
}

func (s *TargetService) validLocalMerchant(scope collection.Scope, input TargetInput, m storecenter.ProductMerchantBinding, r goods.OfficialRuleSnapshot) bool {
	p := s.localTrial
	return p != nil && scope == p.Scope && input.LocalTrial != nil && *input.LocalTrial == p.Binding && input.SourceID == p.Binding.SourceID && input.StoreID == p.Binding.StoreID && r.Authority == "local_trial" && collection.Digest(r) == p.Binding.FixtureHash && m.OrganizationID == scope.OrganizationID && m.StoreID == input.StoreID && m.Site == "shein-us" && m.StoreVersion > 0 && m.Platform == storecenter.PlatformShein && m.ConnectionRevision == 0 && m.ApplicationID == "" && m.ApplicationRevision == "" && m.ApplicationType == storecenter.ApplicationSelfOperated && m.ServiceExpiresAt.IsZero() && m.CredentialExpiresAt.IsZero() && !m.ValidApplication() && len(m.SupplierIdentityHash) == 64 && r.ApplicationMode == model.ModeSelfOperated
}

func (s *TargetService) validPreviewMerchant(scope collection.Scope, base TargetRecord, m storecenter.ProductMerchantBinding, r goods.OfficialRuleSnapshot) bool {
	if base.LocalTrial != nil || base.Input.LocalTrial != nil {
		return s.validLocalMerchant(scope, base.Input, m, r)
	}
	return s.localTrial == nil && validImportMerchant(scope, base.Input.StoreID, m, r) && r.Authority != "local_trial"
}

func (s *TargetService) validateLocalInput(scope collection.Scope, input TargetInput, source preparation.SourceItem, product catalog.PublishedSnapshot) error {
	if s.localTrial == nil {
		if input.LocalTrial != nil {
			return ErrForbidden
		}
		return nil
	}
	p := s.localTrial
	if !p.Enabled || scope != p.Scope || input.LocalTrial == nil || *input.LocalTrial != p.Binding || input.ColorPartition != nil || input.Temu != nil || input.TikTok != nil || input.Walmart != nil || input.CompletionDraft != nil || input.SourceID != p.Binding.SourceID || input.StoreID != p.Binding.StoreID {
		return ErrForbidden
	}
	bootstrap, err := p.bootstrap(source, product)
	if err != nil {
		return err
	}
	if input.ExpectedRevision == 0 {
		// Compare every input field, with only the existing empty-slice JSON
		// normalization used by the original strict BFF contract.
		expectedDraft, actualDraft := bootstrap.Draft, input.Draft
		bootstrap.Draft = goods.OfficialDraftInput{}
		input.Draft = goods.OfficialDraftInput{}
		if !reflect.DeepEqual(bootstrap, input) || !equalSheinDraft(expectedDraft, actualDraft) {
			return ErrConflict
		}
	} else if input.SheinImport == nil && input.SheinSalesMapping == nil {
		return ErrForbidden
	}
	if input.SheinImport != nil && input.SheinImport.CompletionDraft != p.Binding.CompletionDraft || input.SheinSalesMapping != nil && input.SheinSalesMapping.CompletionDraft != p.Binding.CompletionDraft {
		return ErrConflict
	}
	return nil
}

// SameLocalTrial guards immutable lineage under the existing repository lock.
func SameLocalTrial(a, b *LocalTrial) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

func IsLocalTrial(v TargetRecord) bool { return v.LocalTrial != nil || v.Input.LocalTrial != nil }

func (s *TargetService) validateLocalCandidate(ctx context.Context, scope collection.Scope, source preparation.SourceItem) error {
	if s.localTrial == nil {
		return nil
	}
	if s.dependencies.CompletionVersions == nil {
		return ErrUnavailable
	}
	v, err := s.dependencies.CompletionVersions.ReadVersion(ctx, scope, s.localTrial.Binding.CompletionDraft.RecordID)
	if err != nil {
		return err
	}
	if scope != s.localTrial.Scope || v.ID != s.localTrial.Binding.CompletionDraft.RecordID || v.Revision != s.localTrial.Binding.CompletionDraft.Revision || v.Source != source {
		return ErrConflict
	}
	return nil
}

func validLocalRecord(v TargetRecord) bool {
	p := v.LocalTrial
	return p != nil && p.Valid() && SameLocalTrial(p, v.Input.LocalTrial) && v.Source.ID == p.SourceID && v.Input.SourceID == p.SourceID && v.Input.StoreID == p.StoreID && v.Input.ColorPartition == nil && v.Input.Temu == nil && v.Input.TikTok == nil && v.Input.Walmart == nil && v.Input.TemuRules == nil && v.Input.TikTokRules == nil && v.Input.WalmartRules == nil && v.Temu == nil && v.TikTok == nil && v.Walmart == nil && v.TemuRuleSnapshot == nil && v.TikTokRuleSnapshot == nil && v.WalmartRuleSnapshot == nil && len(v.ImageObservations) == 0 && len(v.ProofObservations) == 0 && v.RulesHash == p.FixtureHash && v.Merchant.Platform == storecenter.PlatformShein && v.Merchant.Site == "shein-us" && v.Merchant.StoreID == p.StoreID && v.Merchant.StoreVersion > 0 && v.Merchant.ConnectionRevision == 0 && v.Merchant.ApplicationID == "" && v.Merchant.ApplicationRevision == "" && v.Merchant.ServiceExpiresAt.IsZero() && v.Merchant.CredentialExpiresAt.IsZero() && !v.Merchant.ValidApplication() && !v.Result.ReadyForUpload && len(v.Result.SubmissionPayload) == 0 && len(v.Input.Draft.Images) == 0
}
