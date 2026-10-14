package temu

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/storecenter"
	"time"
)

const supplierSizeClassAPI = "bg.glo.goods.sizecharts.class.get"
const supplierSizeChartsAPI = "bg.glo.goods.sizecharts.get"
const supplierSizeSettingsAPI = "bg.glo.goods.sizecharts.settings.get"

func (c *OfficialClient) QuerySupplierSizeSettings(ctx context.Context, credential storecenter.OfficialMerchantCredential, cat, class int64) (model.SupplierSizeSettings, error) {
	var result model.SupplierSizeSettings
	if ctx == nil || cat < 0 || class < 1 {
		return result, storecenter.ErrOfficialConnectionUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := c.requireSupplierRead(ctx, credential, supplierSizeSettingsAPI); err != nil {
		return result, err
	}
	fields := map[string]json.RawMessage{"classId": json.RawMessage(strconv.FormatInt(class, 10))}
	if cat > 0 {
		fields["catId"] = json.RawMessage(strconv.FormatInt(cat, 10))
	}
	if err := c.readOfficialResponse(ctx, supplierAPIOrigin, supplierSizeSettingsAPI, credential.Temu.AccessToken, fields, &result); err != nil {
		return model.SupplierSizeSettings{}, err
	}
	if result.Sizes == nil || result.Mapping == nil || result.Mapping.Meta == nil || result.Mapping.Meta.Groups == nil || result.Mapping.Records == nil {
		return model.SupplierSizeSettings{}, storecenter.ErrOfficialConnectionUnavailable
	}
	for _, group := range result.Mapping.Meta.Groups {
		if group.ID == nil || *group.ID < 1 {
			return model.SupplierSizeSettings{}, storecenter.ErrOfficialConnectionUnavailable
		}
	}
	return result, nil
}

func (c *OfficialClient) requireSupplierRead(ctx context.Context, credential storecenter.OfficialMerchantCredential, api string) error {
	if c.application.IdentityPartition != storecenter.PartitionCN || !c.validCredential(credential) {
		return storecenter.ErrOfficialConnectionUnavailable
	}
	info, err := c.QueryStore(ctx, credential)
	if err != nil || info.Verification == nil {
		return storecenter.ErrOfficialConnectionUnavailable
	}
	if info.Verification.Mode == storecenter.MerchantLocal || !hasScope(info.Verification.Scopes, api) {
		return storecenter.ErrOfficialAuthorizationRejected
	}
	return nil
}
func (c *OfficialClient) QuerySupplierSizeClass(ctx context.Context, credential storecenter.OfficialMerchantCredential, cat int64) (model.SupplierSizeRequirement, error) {
	var zero model.SupplierSizeRequirement
	if ctx == nil || cat < 1 {
		return zero, storecenter.ErrOfficialConnectionUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := c.requireSupplierRead(ctx, credential, supplierSizeClassAPI); err != nil {
		return zero, err
	}
	var data json.RawMessage
	if err := c.readOfficialResponse(ctx, supplierAPIOrigin, supplierSizeClassAPI, credential.Temu.AccessToken, map[string]json.RawMessage{"catId": json.RawMessage(strconv.FormatInt(cat, 10))}, &data); err != nil {
		return zero, err
	}
	if bytes.Equal(data, []byte("null")) {
		return model.SupplierSizeRequirement{CatID: cat, Observed: true}, nil
	}
	var raw struct {
		Mapping json.RawMessage `json:"sizeSpecClassCat"`
	}
	if json.Unmarshal(data, &raw) != nil {
		return zero, storecenter.ErrOfficialConnectionUnavailable
	}
	if len(raw.Mapping) == 0 {
		return zero, storecenter.ErrOfficialConnectionUnavailable
	}
	result := model.SupplierSizeRequirement{CatID: cat, Observed: true}
	if bytes.Equal(raw.Mapping, []byte("null")) {
		return result, nil
	}
	var class model.SupplierSizeClass
	if json.Unmarshal(raw.Mapping, &class) != nil || class.CatID == nil || *class.CatID != cat || class.ClassID == nil || *class.ClassID < 1 || class.ClassType == nil || *class.ClassType < 0 || *class.ClassType > 1 || *class.ClassType == 1 && len(class.RelatedClassIDs) < 2 {
		return zero, storecenter.ErrOfficialConnectionUnavailable
	}
	result.Class = &class
	return result, nil
}
func (c *OfficialClient) QuerySupplierSizeCharts(ctx context.Context, credential storecenter.OfficialMerchantCredential, cat, offset int64) (model.SupplierSizePage, error) {
	var result model.SupplierSizePage
	if ctx == nil || cat < 1 || offset < 0 || offset > 1000000 {
		return result, storecenter.ErrOfficialConnectionUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := c.requireSupplierRead(ctx, credential, supplierSizeChartsAPI); err != nil {
		return result, err
	}
	fields := map[string]json.RawMessage{"catId": json.RawMessage(strconv.FormatInt(cat, 10)), "offset": json.RawMessage(strconv.FormatInt(offset, 10)), "pageSize": json.RawMessage("50")}
	if err := c.readOfficialResponse(ctx, supplierAPIOrigin, supplierSizeChartsAPI, credential.Temu.AccessToken, fields, &result); err != nil {
		return model.SupplierSizePage{}, err
	}
	if result.Offset == nil || *result.Offset != offset || result.PageSize == nil || *result.PageSize != 50 || result.Total == nil || *result.Total < 0 || result.Charts == nil || len(result.Charts) > 50 || int64(len(result.Charts))+offset > *result.Total {
		return model.SupplierSizePage{}, storecenter.ErrOfficialConnectionUnavailable
	}
	seen := map[int64]bool{}
	for _, chart := range result.Charts {
		if chart.ID == nil || *chart.ID < 1 || seen[*chart.ID] || chart.ClassID == nil || *chart.ClassID < 1 || chart.Reusable == nil || chart.UpdatedAt == nil || chart.Content == nil || chart.Content.Records == nil {
			return model.SupplierSizePage{}, storecenter.ErrOfficialConnectionUnavailable
		}
		seen[*chart.ID] = true
	}
	return result, nil
}
