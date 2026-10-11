package temu

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/storecenter"
)

const (
	supplierAttributesAPI = "bg.goods.attrs.get"
	localCategoryRulesAPI = "temu.local.goods.cats.rules.get"
	localAttributesAPI    = "temu.local.product.attributes.get"
	localFreightAPI       = "bg.freight.template.list.query"
	localTrademarksAPI    = "temu.local.goods.brand.trademark.V2.get"
)

func (c *OfficialClient) QueryLocalFreightTemplates(ctx context.Context, credential storecenter.OfficialMerchantCredential) ([]model.LocalFreightTemplate, error) {
	origin := localAPIOrigin(c.application.IdentityPartition)
	if origin == "" || !c.validCredential(credential) {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := c.requireLocalScope(ctx, credential.Temu.AccessToken, localFreightAPI); err != nil {
		return nil, err
	}
	var result struct {
		Templates []model.LocalFreightTemplate `json:"templateList"`
	}
	if err := c.readOfficialResponse(ctx, origin, localFreightAPI, credential.Temu.AccessToken, nil, &result); err != nil {
		return nil, err
	}
	if result.Templates == nil {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	seen := make(map[string]struct{}, len(result.Templates))
	for _, template := range result.Templates {
		if !boundedParameter(template.ID, 256) || !boundedParameter(template.Name, 1000) {
			return nil, storecenter.ErrOfficialConnectionUnavailable
		}
		if _, duplicate := seen[template.ID]; duplicate {
			return nil, storecenter.ErrOfficialConnectionUnavailable
		}
		seen[template.ID] = struct{}{}
	}
	return result.Templates, nil
}

func (c *OfficialClient) QueryLocalTrademarks(ctx context.Context, credential storecenter.OfficialMerchantCredential, query model.LocalTrademarkQuery) (model.LocalTrademarkPage, error) {
	var result model.LocalTrademarkPage
	origin := localAPIOrigin(c.application.IdentityPartition)
	if origin == "" || !c.validCredential(credential) || query.Page < 1 || query.Size < 1 || query.Size > 100 {
		return result, storecenter.ErrOfficialConnectionUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := c.requireLocalScope(ctx, credential.Temu.AccessToken, localTrademarksAPI); err != nil {
		return result, err
	}
	fields := map[string]json.RawMessage{"page": json.RawMessage(strconv.FormatInt(query.Page, 10)), "size": json.RawMessage(strconv.FormatInt(query.Size, 10))}
	if err := c.readOfficialResponse(ctx, origin, localTrademarksAPI, credential.Temu.AccessToken, fields, &result); err != nil {
		return model.LocalTrademarkPage{}, err
	}
	if result.Page == nil || *result.Page != query.Page || result.Total == nil || *result.Total < 0 || result.Trademarks == nil || int64(len(result.Trademarks)) > query.Size || int64(len(result.Trademarks)) > *result.Total {
		return model.LocalTrademarkPage{}, storecenter.ErrOfficialConnectionUnavailable
	}
	seen := make(map[int64]struct{}, len(result.Trademarks))
	for _, trademark := range result.Trademarks {
		if trademark.TrademarkID == nil || *trademark.TrademarkID < 1 || trademark.BrandID == nil || *trademark.BrandID < 1 || !boundedParameter(trademark.BrandName, 1000) || !boundedParameter(trademark.TrademarkName, 1000) {
			return model.LocalTrademarkPage{}, storecenter.ErrOfficialConnectionUnavailable
		}
		if _, duplicate := seen[*trademark.TrademarkID]; duplicate {
			return model.LocalTrademarkPage{}, storecenter.ErrOfficialConnectionUnavailable
		}
		seen[*trademark.TrademarkID] = struct{}{}
	}
	return result, nil
}

func (c *OfficialClient) QueryLocalAttributes(ctx context.Context, credential storecenter.OfficialMerchantCredential, query model.LocalAttributeQuery) (model.LocalAttributeTemplate, error) {
	var result model.LocalAttributeTemplate
	origin := localAPIOrigin(c.application.IdentityPartition)
	if origin == "" || !c.validCredential(credential) || query.CatID < 1 || query.Language != "" && !boundedParameter(query.Language, 32) || query.CostTemplateID != "" && !boundedParameter(query.CostTemplateID, 256) {
		return result, storecenter.ErrOfficialConnectionUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := c.requireLocalScope(ctx, credential.Temu.AccessToken, localAttributesAPI); err != nil {
		return result, err
	}
	fields := map[string]json.RawMessage{"catId": json.RawMessage(strconv.FormatInt(query.CatID, 10))}
	for _, parameter := range []struct{ name, value string }{{"language", query.Language}, {"costTemplateId", query.CostTemplateID}} {
		if parameter.value != "" {
			encoded, err := exactJSON(parameter.value)
			if err != nil {
				return result, storecenter.ErrOfficialConnectionUnavailable
			}
			fields[parameter.name] = encoded
		}
	}
	if err := c.readOfficialResponse(ctx, origin, localAttributesAPI, credential.Temu.AccessToken, fields, &result); err != nil {
		return model.LocalAttributeTemplate{}, err
	}
	if result.CatID == nil || *result.CatID != query.CatID || result.Attributes == nil || query.Language != "" && result.Language != query.Language {
		return model.LocalAttributeTemplate{}, storecenter.ErrOfficialConnectionUnavailable
	}
	seen := make(map[int64]struct{}, len(result.Attributes))
	for _, attribute := range result.Attributes {
		if attribute.RefPID == nil || *attribute.RefPID < 1 || !boundedParameter(attribute.Name, 1000) {
			return model.LocalAttributeTemplate{}, storecenter.ErrOfficialConnectionUnavailable
		}
		if _, duplicate := seen[*attribute.RefPID]; duplicate {
			return model.LocalAttributeTemplate{}, storecenter.ErrOfficialConnectionUnavailable
		}
		seen[*attribute.RefPID] = struct{}{}
	}
	// Preserve transnational, parent-value groups and unknown constraints. Only
	// the current marketplace completion check can interpret this rule evidence.
	return result, nil
}

func (c *OfficialClient) QueryLocalCategoryRules(ctx context.Context, credential storecenter.OfficialMerchantCredential, query model.CategoryRuleQuery) (model.LocalCategoryRules, error) {
	var result model.LocalCategoryRules
	origin := localAPIOrigin(c.application.IdentityPartition)
	if origin == "" || !c.validCredential(credential) || query.CatID < 1 || query.Language != "" && !boundedParameter(query.Language, 32) {
		return result, storecenter.ErrOfficialConnectionUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := c.requireLocalScope(ctx, credential.Temu.AccessToken, localCategoryRulesAPI); err != nil {
		return result, err
	}
	fields := map[string]json.RawMessage{"catId": json.RawMessage(strconv.FormatInt(query.CatID, 10))}
	if query.Language != "" {
		language, err := exactJSON(query.Language)
		if err != nil {
			return result, storecenter.ErrOfficialConnectionUnavailable
		}
		fields["language"] = language
	}
	if err := c.readOfficialResponse(ctx, origin, localCategoryRulesAPI, credential.Temu.AccessToken, fields, &result); err != nil {
		return model.LocalCategoryRules{}, err
	}
	if result.FieldRules == nil {
		return model.LocalCategoryRules{}, storecenter.ErrOfficialConnectionUnavailable
	}
	seen := make(map[string]struct{}, len(result.FieldRules))
	for _, field := range result.FieldRules {
		if !boundedParameter(field.FieldName, 256) {
			return model.LocalCategoryRules{}, storecenter.ErrOfficialConnectionUnavailable
		}
		if _, duplicate := seen[field.FieldName]; duplicate {
			return model.LocalCategoryRules{}, storecenter.ErrOfficialConnectionUnavailable
		}
		seen[field.FieldName] = struct{}{}
	}
	// Unknown requirement/category enums and warnings are retained. This is
	// evidence for a typed rule consumer, never a ready-to-publish verdict.
	return result, nil
}

func (c *OfficialClient) QuerySupplierAttributes(ctx context.Context, credential storecenter.OfficialMerchantCredential, catID int64) (model.SupplierAttributeTemplate, error) {
	var result model.SupplierAttributeTemplate
	if c.application.IdentityPartition != storecenter.PartitionCN || !c.validCredential(credential) || catID < 1 {
		return result, storecenter.ErrOfficialConnectionUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := c.requireCNScope(ctx, credential.Temu.AccessToken, supplierAttributesAPI); err != nil {
		return result, err
	}
	fields := map[string]json.RawMessage{"catId": json.RawMessage(strconv.FormatInt(catID, 10))}
	// supportedType would override the authorized mall channel. This named
	// current-template query exposes neither that override nor old create time.
	if err := c.readOfficialResponse(ctx, cnAPIOrigin, supplierAttributesAPI, credential.Temu.AccessToken, fields, &result); err != nil {
		return model.SupplierAttributeTemplate{}, err
	}
	if result.Properties == nil {
		return model.SupplierAttributeTemplate{}, storecenter.ErrOfficialConnectionUnavailable
	}
	seen := make(map[int64]struct{}, len(result.Properties))
	for _, property := range result.Properties {
		if property.TemplatePID == nil || *property.TemplatePID < 1 || property.RefPID == nil || *property.RefPID < 1 || !boundedParameter(property.Name, 1000) {
			return model.SupplierAttributeTemplate{}, storecenter.ErrOfficialConnectionUnavailable
		}
		if _, duplicate := seen[*property.TemplatePID]; duplicate {
			return model.SupplierAttributeTemplate{}, storecenter.ErrOfficialConnectionUnavailable
		}
		seen[*property.TemplatePID] = struct{}{}
	}
	return result, nil
}
