package temu

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/storecenter"
	"time"
)

const localGoodsAddAPI = "temu.local.goods.v2.add"
const localImageUploadAPI = "temu.local.goods.image.v2.upload"
const supplierGoodsAddAPI = "bg.glo.goods.add"
const supplierImageUploadAPI = "bg.goods.image.upload.global"
const supplierAPIOrigin = "https://openapi-b-partner.temu.com"

func (c *OfficialClient) CreateSupplierGoods(ctx context.Context, credential storecenter.OfficialMerchantCredential, input model.SupplierGoods) (temugoods.CreatedGoods, error) {
	var result temugoods.CreatedGoods
	if ctx == nil || c.application.IdentityPartition != storecenter.PartitionCN || !c.validCredential(credential) {
		return result, storecenter.ErrOfficialConnectionUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	info, err := c.QueryStore(ctx, credential)
	if err != nil || info.Verification == nil {
		return result, storecenter.ErrOfficialConnectionUnavailable
	}
	facts := info.Verification
	if facts.Mode != storecenter.MerchantFullyManaged && facts.Mode != storecenter.MerchantSemiManaged || !hasScope(facts.Scopes, supplierGoodsAddAPI) || facts.Mode == storecenter.MerchantSemiManaged != (input.SemiManaged != nil) {
		return result, storecenter.ErrOfficialAuthorizationRejected
	}
	fields, err := fixedGoodsFields(input)
	if err != nil {
		return result, err
	}
	var created model.SupplierGoodsCreated
	hash, err := c.callOfficialResponse(ctx, supplierAPIOrigin, supplierGoodsAddAPI, credential.Temu.AccessToken, fields, &created)
	if err != nil {
		return result, err
	}
	return temugoods.CreatedGoods{Supplier: &created, ResponseHash: hash}, nil
}
func (c *OfficialClient) UploadSupplierImage(ctx context.Context, credential storecenter.OfficialMerchantCredential, input model.SupplierImageUpload) (model.SupplierImageUploaded, string, error) {
	var result model.SupplierImageUploaded
	if ctx == nil || c.application.IdentityPartition != storecenter.PartitionCN || !c.validCredential(credential) || input.Options.CategoryID < 1 || input.BusinessType < 0 || input.BusinessType > 1 || input.Options.IntelligenceCrop || input.Options.Boost || input.Options.SizeMode != 0 || len(input.Image) > base64.StdEncoding.EncodedLen(2<<20) {
		return result, "", storecenter.ErrOfficialConnectionUnavailable
	}
	content, err := base64.StdEncoding.Strict().DecodeString(input.Image)
	if err != nil || len(content) < 1 || len(content) > 2<<20 || base64.StdEncoding.EncodeToString(content) != input.Image {
		return result, "", storecenter.ErrOfficialConnectionUnavailable
	}
	defer clear(content)
	config, format, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil || format != "png" && format != "jpeg" || config.Width < 1 || config.Height < 1 || config.Width > 10000 || config.Height > 10000 || int64(config.Width)*int64(config.Height) > 20_000_000 {
		return result, "", storecenter.ErrOfficialConnectionUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	info, err := c.QueryStore(ctx, credential)
	if err != nil || info.Verification == nil {
		return result, "", storecenter.ErrOfficialConnectionUnavailable
	}
	facts := info.Verification
	if facts.Mode != storecenter.MerchantFullyManaged && facts.Mode != storecenter.MerchantSemiManaged || !hasScope(facts.Scopes, supplierImageUploadAPI) {
		return result, "", storecenter.ErrOfficialAuthorizationRejected
	}
	raw, err := json.Marshal(input)
	if err != nil || len(raw) > officialRequestLimit(supplierImageUploadAPI) {
		return result, "", storecenter.ErrOfficialConnectionUnavailable
	}
	fields := map[string]json.RawMessage{}
	if json.Unmarshal(raw, &fields) != nil {
		return result, "", storecenter.ErrOfficialConnectionUnavailable
	}
	clear(raw)
	hash, err := c.callOfficialResponse(ctx, supplierAPIOrigin, supplierImageUploadAPI, credential.Temu.AccessToken, fields, &result)
	if err != nil || len(result.URLs) > 1000 {
		return model.SupplierImageUploaded{}, "", storecenter.ErrOfficialConnectionUnavailable
	}
	urls := append([]string{}, result.URLs...)
	if result.ImageURL != "" {
		urls = append(urls, result.ImageURL)
	}
	if result.URL != "" {
		urls = append(urls, result.URL)
	}
	if len(urls) == 0 {
		return model.SupplierImageUploaded{}, "", storecenter.ErrOfficialConnectionUnavailable
	}
	for _, value := range urls {
		u, e := url.Parse(value)
		if e != nil || len(value) > 2048 || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
			return model.SupplierImageUploaded{}, "", storecenter.ErrOfficialConnectionUnavailable
		}
	}
	return result, hash, nil
}

// These methods retain one fixed typed official protocol and never retry a
// mutation. The original Store handle and committed Submission permit own use.
func (c *OfficialClient) CreateLocalGoods(ctx context.Context, credential storecenter.OfficialMerchantCredential, input model.LocalGoodsV2) (temugoods.CreatedGoods, error) {
	var result temugoods.CreatedGoods
	origin := localAPIOrigin(c.application.IdentityPartition)
	if ctx == nil || origin == "" || !c.validCredential(credential) {
		return result, storecenter.ErrOfficialConnectionUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := c.requireLocalScope(ctx, credential.Temu.AccessToken, localGoodsAddAPI); err != nil {
		return result, err
	}
	fields, err := fixedGoodsFields(input)
	if err != nil {
		return result, err
	}
	created := model.LocalGoodsCreated{}
	hash, err := c.callOfficialResponse(ctx, origin, localGoodsAddAPI, credential.Temu.AccessToken, fields, &created)
	if err != nil {
		return result, err
	}
	return temugoods.CreatedGoods{Local: &created, ResponseHash: hash}, nil
}

func (c *OfficialClient) UploadLocalImage(ctx context.Context, credential storecenter.OfficialMerchantCredential, input model.LocalImageUpload) (model.LocalImageUploaded, string, error) {
	var result model.LocalImageUploaded
	origin := localAPIOrigin(c.application.IdentityPartition)
	source, err := url.Parse(input.FileURL)
	if ctx == nil || origin == "" || !c.validCredential(credential) || err != nil || source.Scheme != "https" || source.Hostname() == "" || source.User != nil || source.Fragment != "" || input.CatID < 1 || input.Usage < 1 || input.Usage > 7 {
		return result, "", storecenter.ErrOfficialConnectionUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := c.requireLocalScope(ctx, credential.Temu.AccessToken, localImageUploadAPI); err != nil {
		return result, "", err
	}
	fields, err := fixedGoodsFields(input)
	if err != nil {
		return result, "", err
	}
	hash, err := c.callOfficialResponse(ctx, origin, localImageUploadAPI, credential.Temu.AccessToken, fields, &result)
	if err != nil || len(result.Images) < 1 || len(result.Images) > 1000 {
		return model.LocalImageUploaded{}, "", storecenter.ErrOfficialConnectionUnavailable
	}
	for _, image := range result.Images {
		u, err := url.Parse(image.URL)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
			return model.LocalImageUploaded{}, "", storecenter.ErrOfficialConnectionUnavailable
		}
	}
	return result, hash, nil
}

func fixedGoodsFields(input any) (map[string]json.RawMessage, error) {
	raw, err := json.Marshal(input)
	if err != nil || len(raw) > maxOfficialBody {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	return fields, nil
}
