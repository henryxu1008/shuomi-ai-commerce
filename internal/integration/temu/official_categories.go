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
	cnAPIOrigin           = "https://openapi.kuajingmaihuo.com"
	supplierCategoriesAPI = "bg.goods.cats.get"
	localCategoriesAPI    = "bg.local.goods.cats.get"
)

func localAPIOrigin(partition storecenter.OfficialIdentityPartition) string {
	switch partition {
	case storecenter.PartitionUS:
		return "https://openapi-b-us.temu.com"
	case storecenter.PartitionEU:
		return "https://openapi-b-eu.temu.com"
	case storecenter.PartitionGlobal:
		return "https://openapi-b-global.temu.com"
	default:
		return ""
	}
}

// QuerySupplierCategories implements E33 at CN, where the interface has not
// migrated. PA publishing scopes are not evidence for this CN permission.
// Credential access and Store/actor fencing belong to the server-side caller.
func (c *OfficialClient) QuerySupplierCategories(ctx context.Context, credential storecenter.OfficialMerchantCredential, query model.SupplierCategoryQuery) ([]model.SupplierCategory, error) {
	if c.application.IdentityPartition != storecenter.PartitionCN || !c.validCredential(credential) || query.ParentCatID < 0 || query.SiteID < 0 {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	token := credential.Temu.AccessToken
	if err := c.requireCNScope(ctx, token, supplierCategoriesAPI); err != nil {
		return nil, err
	}
	fields := map[string]json.RawMessage{"showHidden": json.RawMessage("false")}
	if query.ParentCatID > 0 {
		fields["parentCatId"] = json.RawMessage(strconv.FormatInt(query.ParentCatID, 10))
	}
	if query.SiteID > 0 {
		fields["siteId"] = json.RawMessage(strconv.FormatInt(query.SiteID, 10))
	}
	var result struct {
		Items []model.SupplierCategory `json:"categoryDTOList"`
	}
	if err := c.readOfficialResponse(ctx, cnAPIOrigin, supplierCategoriesAPI, token, fields, &result); err != nil {
		return nil, err
	}
	if result.Items == nil {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	seen := make(map[int64]struct{}, len(result.Items))
	for _, item := range result.Items {
		if !validCategory(item.CatID, item.Name, item.ParentCatID, query.ParentCatID, seen) {
			return nil, storecenter.ErrOfficialConnectionUnavailable
		}
	}
	return result.Items, nil
}

// QueryLocalCategories retains the original application route. The local
// response uses goodsCatsList/availableStatus, rather than supplier flags.
func (c *OfficialClient) QueryLocalCategories(ctx context.Context, credential storecenter.OfficialMerchantCredential, query model.LocalCategoryQuery) ([]model.LocalCategory, error) {
	origin := localAPIOrigin(c.application.IdentityPartition)
	if origin == "" || !c.validCredential(credential) || query.ParentCatID < 0 || query.Language != "" && !boundedParameter(query.Language, 32) {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	token := credential.Temu.AccessToken
	identity, err := c.queryLocal(ctx, token)
	if err != nil {
		return nil, err
	}
	if !hasScope(identity.Verification.Scopes, localCategoriesAPI) {
		return nil, storecenter.ErrOfficialAuthorizationRejected
	}
	fields := map[string]json.RawMessage{"parentCatId": json.RawMessage(strconv.FormatInt(query.ParentCatID, 10))}
	if query.Language != "" {
		fields["language"], err = exactJSON(query.Language)
		if err != nil {
			return nil, storecenter.ErrOfficialConnectionUnavailable
		}
	}
	var result struct {
		Items []model.LocalCategory `json:"goodsCatsList"`
	}
	if err := c.readOfficialResponse(ctx, origin, localCategoriesAPI, token, fields, &result); err != nil {
		return nil, err
	}
	if result.Items == nil {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	seen := make(map[int64]struct{}, len(result.Items))
	for _, item := range result.Items {
		if !validCategory(item.CatID, item.Name, item.ParentID, query.ParentCatID, seen) {
			return nil, storecenter.ErrOfficialConnectionUnavailable
		}
	}
	return result.Items, nil
}

func (c *OfficialClient) requireCNScope(ctx context.Context, token, api string) error {
	var result struct {
		MallID      *int64   `json:"mallId"`
		ExpiredTime *int64   `json:"expiredTime"`
		Scopes      []string `json:"apiScopeList"`
	}
	if err := c.readIdentity(ctx, cnAPIOrigin, localInfoAPI, token, &result); err != nil {
		return err
	}
	if result.MallID == nil || *result.MallID < 1 {
		return storecenter.ErrOfficialConnectionUnavailable
	}
	if _, _, err := c.validScopeExpiry(result.ExpiredTime, result.Scopes); err != nil {
		return err
	}
	if !hasScope(result.Scopes, api) {
		return storecenter.ErrOfficialAuthorizationRejected
	}
	return nil
}

func hasScope(scopes []string, api string) bool {
	for _, scope := range scopes {
		if scope == api {
			return true
		}
	}
	return false
}

func validCategory(id int64, name string, parent *int64, requestedParent int64, seen map[int64]struct{}) bool {
	if id < 1 || !boundedParameter(name, 1000) || parent != nil && *parent != requestedParent {
		return false
	}
	if _, exists := seen[id]; exists {
		return false
	}
	seen[id] = struct{}{}
	return true
}
