package target

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	sigjson "sigs.k8s.io/json"
	"strconv"
	"task-processor/internal/product/description"
	"time"

	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

// These are selections, never replacement business values. Values come only
// from the immutable candidate and baseline owned by the existing repositories.
type SheinAttributeMapping struct {
	SourceIndex int    `json:"sourceIndex"`
	AttributeID int64  `json:"attributeId"`
	OptionID    *int64 `json:"optionId,omitempty"`
}
type SheinSKUMapping struct {
	SourceIndex int    `json:"sourceIndex"`
	SKCIndex    int    `json:"skcIndex"`
	SKUIndex    int    `json:"skuIndex"`
	Price       bool   `json:"price"`
	Stock       bool   `json:"stock"`
	WarehouseID string `json:"warehouseId"`
}
type SheinImportSelection struct {
	Title       bool                    `json:"title"`
	Description bool                    `json:"description"`
	Attributes  []SheinAttributeMapping `json:"attributes"`
	SKUs        []SheinSKUMapping       `json:"skus"`
}

// Missing/null integer positions must never silently become source/SKC/SKU 0.
// Keep strict decoding when this type is nested in either Preview or Target.
func (s *SheinImportSelection) UnmarshalJSON(raw []byte) error {
	type plain SheinImportSelection
	var value plain
	strict, err := sigjson.UnmarshalStrict(raw, &value, sigjson.DisallowDuplicateFields, sigjson.DisallowUnknownFields)
	if err != nil || len(strict) > 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return ErrInvalid
	}
	var fields struct {
		Attributes []map[string]json.RawMessage `json:"attributes"`
		SKUs       []map[string]json.RawMessage `json:"skus"`
	}
	if json.Unmarshal(raw, &fields) != nil {
		return ErrInvalid
	}
	required := func(rows []map[string]json.RawMessage, keys ...string) bool {
		for _, row := range rows {
			for _, key := range keys {
				v := bytes.TrimSpace(row[key])
				if len(v) == 0 || bytes.Equal(v, []byte("null")) {
					return false
				}
			}
		}
		return true
	}
	if !required(fields.Attributes, "sourceIndex", "attributeId") || !required(fields.SKUs, "sourceIndex", "skcIndex", "skuIndex", "price", "stock", "warehouseId") {
		return ErrInvalid
	}
	*s = SheinImportSelection(value)
	return nil
}

type SheinImportRequest struct {
	SourceID        string                   `json:"sourceId"`
	BaseRecordID    string                   `json:"baseRecordId"`
	CompletionDraft CompletionDraftSelection `json:"completionDraft"`
	Selection       SheinImportSelection     `json:"selection"`
}
type SheinImport struct {
	TargetID        string                   `json:"targetId"`
	BaseRecordID    string                   `json:"baseRecordId"`
	CompletionDraft CompletionDraftSelection `json:"completionDraft"`
	Selection       SheinImportSelection     `json:"selection"`
	RulesHash       string                   `json:"rulesHash"`
	MerchantHash    string                   `json:"merchantHash"`
}
type SheinImportPreview struct {
	Input TargetInput `json:"input"`
}

func deriveSheinDraft(base goods.OfficialDraftInput, candidate completiondraft.Content, selection SheinImportSelection, rules goods.OfficialRuleSnapshot) (goods.OfficialDraftInput, error) {
	if len(selection.Attributes) > completiondraft.MaxStructuredItems || len(selection.SKUs) > completiondraft.MaxStructuredItems || !selection.Title && !selection.Description && len(selection.Attributes) == 0 && len(selection.SKUs) == 0 {
		return goods.OfficialDraftInput{}, ErrInvalid
	}
	raw, err := json.Marshal(base)
	if err != nil {
		return goods.OfficialDraftInput{}, ErrInvalid
	}
	var result goods.OfficialDraftInput
	if json.Unmarshal(raw, &result) != nil {
		return result, ErrInvalid
	}
	replaceEnglish := func(list []model.LanguageContent, value string) []model.LanguageContent {
		for i := range list {
			if list[i].Language == "en" {
				list[i].Name = value
				return list
			}
		}
		return append(list, model.LanguageContent{Language: "en", Name: value})
	}
	if selection.Title {
		result.Product.Names = replaceEnglish(result.Product.Names, candidate.Title)
	}
	if selection.Description {
		result.Product.Descriptions = replaceEnglish(result.Product.Descriptions, description.PlainText(candidate.Description, candidate.DescriptionFormat))
	}
	if len(selection.Attributes)+len(selection.SKUs) > 0 && candidate.Structured == nil {
		return result, ErrConflict
	}
	sources, targets := map[int]bool{}, map[int64]bool{}
	for _, mapping := range selection.Attributes {
		if mapping.AttributeID < 1 || mapping.OptionID != nil && *mapping.OptionID < 1 || mapping.SourceIndex < 0 || mapping.SourceIndex >= len(candidate.Structured.Attributes) || sources[mapping.SourceIndex] || targets[mapping.AttributeID] {
			return result, ErrInvalid
		}
		sources[mapping.SourceIndex], targets[mapping.AttributeID] = true, true
		var selected *model.Attribute
		for i := range rules.Attributes.Attributes {
			if rules.Attributes.Attributes[i].ID == mapping.AttributeID {
				selected = &rules.Attributes.Attributes[i]
				break
			}
		}
		if selected == nil || selected.Show == nil || *selected.Show != 1 || selected.Status == nil || *selected.Status == 1 || selected.Type == nil || *selected.Type < 3 || *selected.Type > 4 || selected.Dimension == nil || *selected.Dimension != 1 || selected.Mode == nil || *selected.Mode < 0 || *selected.Mode > 4 {
			return result, ErrNotReady
		}
		value := model.AttributeValue{AttributeID: mapping.AttributeID}
		if *selected.Mode == 0 {
			if mapping.OptionID != nil {
				return result, ErrInvalid
			}
			value.ExtraValue = candidate.Structured.Attributes[mapping.SourceIndex].Value
		} else {
			found := false
			for _, option := range selected.Options {
				if mapping.OptionID != nil && option.ID == *mapping.OptionID && option.Show != nil && *option.Show == 1 {
					found = true
				}
			}
			if !found {
				return result, ErrNotReady
			}
			value.AttributeValueID = mapping.OptionID
			if *selected.Mode == 4 {
				value.ExtraValue = candidate.Structured.Attributes[mapping.SourceIndex].Value
			}
		}
		kept := make([]model.AttributeValue, 0, len(result.Product.Attributes)+1)
		for _, old := range result.Product.Attributes {
			if old.AttributeID != mapping.AttributeID {
				kept = append(kept, old)
			}
		}
		result.Product.Attributes = append(kept, value)
	}
	skuSources, skuTargets := map[int]bool{}, map[[2]int]bool{}
	for _, mapping := range selection.SKUs {
		dest := [2]int{mapping.SKCIndex, mapping.SKUIndex}
		if mapping.SourceIndex < 0 || mapping.SourceIndex >= len(candidate.Structured.Variants) || mapping.SKCIndex < 0 || mapping.SKCIndex >= len(result.Product.SKCs) || mapping.SKUIndex < 0 || mapping.SKUIndex >= len(result.Product.SKCs[mapping.SKCIndex].SKUs) || skuSources[mapping.SourceIndex] || skuTargets[dest] || !mapping.Price && !mapping.Stock || !mapping.Stock && mapping.WarehouseID != "" {
			return result, ErrInvalid
		}
		skuSources[mapping.SourceIndex], skuTargets[dest] = true, true
		source := candidate.Structured.Variants[mapping.SourceIndex]
		if source.SourceIndex == nil || *source.SourceIndex != mapping.SourceIndex {
			return result, ErrConflict
		}
		sku := &result.Product.SKCs[mapping.SKCIndex].SKUs[mapping.SKUIndex]
		if mapping.Price {
			if source.Price == nil || source.Price.Amount == nil {
				return result, ErrInvalid
			}
			switch rules.ApplicationMode {
			case model.ModeSelfOperated:
				if source.Price.Currency != "USD" {
					return result, ErrNotReady
				}
				sku.Prices = []model.ProductPrice{{BasePrice: *source.Price.Amount, Currency: "USD", SubSite: "shein-us"}}
			case model.ModeSemiManaged, model.ModeFullyManaged:
				if rules.Fill.Currency == nil || source.Price.Currency != *rules.Fill.Currency {
					return result, ErrNotReady
				}
				sku.Cost = &model.CostPrice{Price: strconv.FormatFloat(*source.Price.Amount, 'f', -1, 64), Currency: source.Price.Currency}
			default:
				return result, ErrNotReady
			}
		}
		if mapping.Stock {
			if source.Stock == nil || *source.Stock < 0 || *source.Stock > 99999 {
				return result, ErrInvalid
			}
			stock := model.ProductStock{Quantity: int(*source.Stock), WarehouseID: mapping.WarehouseID}
			if rules.ApplicationMode == model.ModeFullyManaged || len(rules.Warehouses) == 0 {
				if mapping.WarehouseID != "" {
					return result, ErrNotReady
				}
			} else {
				found := false
				for _, w := range rules.Warehouses {
					if w.Code == mapping.WarehouseID && w.Code != "" && w.Type == 1 {
						for _, country := range w.SaleCountries {
							if country == "US" {
								found = true
								stock.WarehouseName = w.Name
							}
						}
					}
				}
				if !found {
					return result, ErrNotReady
				}
			}
			at := -1
			for i, old := range sku.Stock {
				if old.WarehouseID == stock.WarehouseID {
					if at != -1 {
						return result, ErrConflict
					}
					at = i
				} else if old.WarehouseID == "" || stock.WarehouseID == "" {
					return result, ErrConflict
				}
			}
			if at < 0 {
				sku.Stock = append(sku.Stock, stock)
			} else {
				sku.Stock[at] = stock
			}
		}
	}
	return result, nil
}

func validImportRequest(in SheinImportRequest) bool {
	return collection.ValidID(in.SourceID) && collection.ValidID(in.BaseRecordID) && collection.ValidID(in.CompletionDraft.RecordID) && in.CompletionDraft.Revision > 0
}

func (s *TargetService) readImport(ctx context.Context, scope collection.Scope, in SheinImportRequest, source preparation.SourceItem) (TargetRecord, completiondraft.Record, error) {
	if !validImportRequest(in) {
		return TargetRecord{}, completiondraft.Record{}, ErrInvalid
	}
	if s.dependencies.CompletionVersions == nil {
		return TargetRecord{}, completiondraft.Record{}, ErrUnavailable
	}
	base, err := s.dependencies.Records.ReadTargetRecord(ctx, scope, in.BaseRecordID)
	if err != nil {
		return base, completiondraft.Record{}, err
	}
	candidate, err := s.dependencies.CompletionVersions.ReadVersion(ctx, scope, in.CompletionDraft.RecordID)
	if err != nil {
		return base, candidate, err
	}
	if base.ID != in.BaseRecordID || base.Revision < 1 || base.Source != source || source.ID != in.SourceID || base.TargetID != TargetIdentity(scope, source.ID, base.Input.StoreID) || base.Input.Temu != nil || base.Input.TikTok != nil || base.Input.TemuRules != nil || base.Input.TikTokRules != nil || base.Input.SourceID != source.ID || base.Merchant.Site != "shein-us" || candidate.Source != source || candidate.ID != in.CompletionDraft.RecordID || candidate.Revision != in.CompletionDraft.Revision {
		return base, candidate, ErrConflict
	}
	return base, candidate, nil
}

// Preview is read-only. The returned input must still pass all current Save
// gates; it is not a capability token, readiness receipt or a publication.
func (s *TargetService) PreviewSheinImport(ctx context.Context, in SheinImportRequest) (SheinImportPreview, error) {
	if ctx == nil || s == nil || !validImportRequest(in) {
		return SheinImportPreview{}, ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	scope, err := s.authorizeTarget(ctx, preparation.PermissionManage)
	if err != nil {
		return SheinImportPreview{}, err
	}
	selected, err := s.dependencies.Sources.Select(ctx, in.SourceID)
	if err != nil {
		return SheinImportPreview{}, err
	}
	owner, source, _, err := selected.Read(ctx)
	if err != nil || owner != scope {
		return SheinImportPreview{}, ErrForbidden
	}
	base, candidate, err := s.readImport(ctx, scope, in, source)
	if err != nil {
		return SheinImportPreview{}, err
	}
	head, err := s.dependencies.Records.ReadTargetHead(ctx, scope, base.TargetID)
	if err != nil {
		return SheinImportPreview{}, err
	}
	if head.ID != base.ID || head.Revision != base.Revision {
		return SheinImportPreview{}, ErrConflict
	}
	if _, err = s.dependencies.Products.ReadEffectiveTargetProduct(ctx, selected, base.EffectiveVersion, base.ApplyReceiptID); err != nil {
		return SheinImportPreview{}, err
	}
	_, rules, err := s.dependencies.Rules.ReadTargetRules(ctx, scope, base.Input.StoreID, base.Input.Draft)
	if err != nil {
		return SheinImportPreview{}, err
	}
	draft, err := deriveSheinDraft(base.Input.Draft, candidate.Content, in.Selection, rules)
	if err != nil {
		return SheinImportPreview{}, err
	}
	merchant, rules, err := s.dependencies.Rules.ReadTargetRules(ctx, scope, base.Input.StoreID, draft)
	if err != nil {
		return SheinImportPreview{}, err
	}
	if !validImportMerchant(scope, base.Input.StoreID, merchant, rules) {
		return SheinImportPreview{}, ErrNotReady
	}
	recomputed, err := deriveSheinDraft(base.Input.Draft, candidate.Content, in.Selection, rules)
	if err != nil {
		return SheinImportPreview{}, err
	}
	if !reflect.DeepEqual(draft, recomputed) {
		return SheinImportPreview{}, ErrConflict
	}
	input := TargetInput{SourceID: source.ID, StoreID: base.Input.StoreID, ExpectedRevision: base.Revision, EffectiveVersion: base.EffectiveVersion, ApplyReceiptID: base.ApplyReceiptID, Draft: draft, SheinImport: &SheinImport{TargetID: base.TargetID, BaseRecordID: base.ID, CompletionDraft: in.CompletionDraft, Selection: in.Selection, RulesHash: collection.Digest(rules), MerchantHash: collection.Digest(merchant)}}
	if _, _, _, err = selected.Read(ctx); err != nil {
		return SheinImportPreview{}, err
	}
	if _, err = s.authorizeTarget(ctx, preparation.PermissionManage); err != nil {
		return SheinImportPreview{}, err
	}
	return SheinImportPreview{Input: input}, nil
}

func validImportMerchant(scope collection.Scope, storeID string, m storecenter.ProductMerchantBinding, r goods.OfficialRuleSnapshot) bool {
	return m.OrganizationID == scope.OrganizationID && m.StoreID == storeID && m.Site == "shein-us" && m.StoreVersion > 0 && m.ConnectionRevision > 0 && m.ValidApplication() && string(r.ApplicationMode) == string(m.ApplicationType) && len(m.SupplierIdentityHash) == 64 && time.Now().Before(m.ServiceExpiresAt)
}

func (s *TargetService) validateSheinImport(ctx context.Context, scope collection.Scope, input TargetInput, source preparation.SourceItem, platform preparedPlatform) error {
	ref := input.SheinImport
	if ref == nil {
		return nil
	}
	request := SheinImportRequest{SourceID: input.SourceID, BaseRecordID: ref.BaseRecordID, CompletionDraft: ref.CompletionDraft, Selection: ref.Selection}
	base, candidate, err := s.readImport(ctx, scope, request, source)
	if err != nil {
		return err
	}
	if ref.TargetID != base.TargetID || input.StoreID != base.Input.StoreID || input.ExpectedRevision != base.Revision || input.EffectiveVersion != base.EffectiveVersion || input.ApplyReceiptID != base.ApplyReceiptID || ref.RulesHash != platform.rulesHash || ref.MerchantHash != collection.Digest(platform.merchant) {
		return ErrConflict
	}
	derived, err := deriveSheinDraft(base.Input.Draft, candidate.Content, ref.Selection, platform.sheinRules)
	if err != nil {
		return err
	}
	if !equalSheinDraft(derived, input.Draft) {
		return ErrConflict
	}
	return nil
}

// The Console's existing list DTO treats null and [] as the same empty list.
// Normalize only empty slice representation; pointers and all business values
// (including zero, omitted optional values, and ordering) remain distinct.
func equalSheinDraft(a, b goods.OfficialDraftInput) bool {
	canonical := func(value goods.OfficialDraftInput) goods.OfficialDraftInput {
		raw, _ := json.Marshal(value)
		var copy goods.OfficialDraftInput
		_ = json.Unmarshal(raw, &copy)
		var visit func(reflect.Value)
		visit = func(v reflect.Value) {
			switch v.Kind() {
			case reflect.Struct:
				for i := 0; i < v.NumField(); i++ {
					visit(v.Field(i))
				}
			case reflect.Pointer:
				if !v.IsNil() {
					visit(v.Elem())
				}
			case reflect.Slice:
				if v.Len() == 0 {
					v.Set(reflect.Zero(v.Type()))
				} else {
					for i := 0; i < v.Len(); i++ {
						visit(v.Index(i))
					}
				}
			}
		}
		visit(reflect.ValueOf(&copy).Elem())
		return copy
	}
	return reflect.DeepEqual(canonical(a), canonical(b))
}
