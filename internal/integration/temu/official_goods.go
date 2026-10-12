package temu

import (
	"context"
	"encoding/json"
	"net/url"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/storecenter"
	"time"
)

const localGoodsAddAPI = "temu.local.goods.v2.add"
const localImageUploadAPI = "temu.local.goods.image.v2.upload"
const supplierGoodsAddAPI = "bg.glo.goods.add"

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
