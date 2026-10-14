package tiktokshop

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	marketplace "task-processor/internal/marketplace/tiktokshop"
)

// All mutations use a fresh HTTP/1 connection, one transport invocation and
// sanitized errors. Submission, not this adapter, owns dispatch permission.
func (c *OfficialClient) resourceMutation(ctx context.Context, path, token, cipher, contentType string, body []byte) ([]byte, error) {
	if c == nil || ctx == nil || ctx.Err() != nil || !bounded(token, 8192) || len(body) == 0 || len(body) > 12<<20 || cipher != "" && !bounded(cipher, 4096) {
		return nil, marketplace.ErrPublicationInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	query := url.Values{"app_key": {c.application.AppKey}, "timestamp": {strconv.FormatInt(c.now().Unix(), 10)}}
	if cipher != "" {
		query.Set("shop_cipher", cipher)
	}
	signed, err := signature(path, query, body, contentType, c.appSecret)
	if err != nil {
		return nil, marketplace.ErrPublicationInvalid
	}
	query.Set("sign", signed)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, apiOrigin+path+"?"+query.Encode(), bytes.NewReader(body))
	if err != nil {
		return nil, marketplace.ErrPublicationInvalid
	}
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("x-tts-access-token", token)
	request.Close = true
	response, err := c.authHTTP.Do(request)
	if err != nil {
		return nil, marketplace.ErrPublicationUnknown
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, marketplace.ErrPublicationUnknown
	}
	media, params, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || media != "application/json" || params["charset"] != "" && !strings.EqualFold(params["charset"], "utf-8") {
		return nil, marketplace.ErrPublicationUnknown
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, marketplace.MaxOfficialResultBytes+1))
	if err != nil || len(raw) == 0 || len(raw) > marketplace.MaxOfficialResultBytes || !utf8.Valid(raw) {
		return nil, marketplace.ErrPublicationUnknown
	}
	return raw, nil
}

func (c *OfficialClient) UploadProductImage(ctx context.Context, token, useCase, mediaType string, content []byte) (marketplace.UploadedImage, error) {
	if len(content) == 0 || len(content) > 10<<20 || mediaType != "image/jpeg" && mediaType != "image/png" {
		return marketplace.UploadedImage{}, marketplace.ErrPublicationInvalid
	}
	switch useCase {
	case "MAIN_IMAGE", "ATTRIBUTE_IMAGE", "CERTIFICATION_IMAGE", "SIZE_CHART_IMAGE":
	default:
		return marketplace.UploadedImage{}, marketplace.ErrPublicationInvalid
	}
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	filename := "image.jpg"
	if mediaType == "image/png" {
		filename = "image.png"
	}
	file, err := writer.CreateFormFile("data", filename)
	if err != nil {
		return marketplace.UploadedImage{}, marketplace.ErrPublicationInvalid
	}
	if _, err = file.Write(content); err != nil {
		return marketplace.UploadedImage{}, marketplace.ErrPublicationInvalid
	}
	if err = writer.WriteField("use_case", useCase); err != nil {
		return marketplace.UploadedImage{}, marketplace.ErrPublicationInvalid
	}
	if err = writer.Close(); err != nil {
		return marketplace.UploadedImage{}, marketplace.ErrPublicationInvalid
	}
	raw, err := c.resourceMutation(ctx, "/product/202309/images/upload", token, "", writer.FormDataContentType(), buffer.Bytes())
	if err != nil {
		return marketplace.UploadedImage{}, err
	}
	return marketplace.DecodeUploadedImage(raw, useCase)
}
func marshalMutation(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil || len(raw) == 0 || len(raw) > 2<<20 {
		return nil, marketplace.ErrPublicationInvalid
	}
	return raw, nil
}
func (c *OfficialClient) CreateGlobalProduct(ctx context.Context, token string, input marketplace.CreateGlobalRequest) (marketplace.CreatedGlobalProduct, error) {
	if len(input.SKUs) < 1 || len(input.SKUs) > 300 || input.CategoryVersion != "v2" || !bounded(input.ExternalID, 128) {
		return marketplace.CreatedGlobalProduct{}, marketplace.ErrPublicationInvalid
	}
	body, err := marshalMutation(input)
	if err != nil {
		return marketplace.CreatedGlobalProduct{}, err
	}
	raw, err := c.resourceMutation(ctx, "/product/202309/global_products", token, "", "application/json", body)
	if err != nil {
		return marketplace.CreatedGlobalProduct{}, err
	}
	sellers := make([]string, 0, len(input.SKUs))
	for _, sku := range input.SKUs {
		sellers = append(sellers, sku.SellerSKU)
	}
	return marketplace.DecodeCreatedGlobal(raw, sellers)
}
func (c *OfficialClient) CreateLocalProduct(ctx context.Context, token, cipher string, input marketplace.CreateLocalRequest) (marketplace.CreatedLocalProduct, error) {
	if !bounded(cipher, 4096) || len(input.SKUs) < 1 || len(input.SKUs) > 300 || input.CategoryVersion != "v2" || !bounded(input.IdempotencyKey, 128) || input.SaveMode != "LISTING" && input.SaveMode != "AS_DRAFT" {
		return marketplace.CreatedLocalProduct{}, marketplace.ErrPublicationInvalid
	}
	body, err := marshalMutation(input)
	if err != nil {
		return marketplace.CreatedLocalProduct{}, err
	}
	raw, err := c.resourceMutation(ctx, "/product/202309/products", token, cipher, "application/json", body)
	if err != nil {
		return marketplace.CreatedLocalProduct{}, err
	}
	sellers := make([]string, 0, len(input.SKUs))
	for _, sku := range input.SKUs {
		sellers = append(sellers, sku.SellerSKU)
	}
	return marketplace.DecodeCreatedLocal(raw, sellers)
}
func (c *OfficialClient) PublishGlobalProduct(ctx context.Context, token, globalID string, input marketplace.PublishGlobalRequest, expected marketplace.GlobalPublishExpectation) (marketplace.GlobalPublishResult, error) {
	if !bounded(globalID, 128) || len(input.Targets) != 1 || input.Targets[0].Region != "US" || len(input.Targets[0].SKUs) != len(expected.GlobalSKUIDs) || len(expected.GlobalSKUIDs) < 1 {
		return marketplace.GlobalPublishResult{}, marketplace.ErrPublicationInvalid
	}
	wanted := map[string]bool{}
	for _, id := range expected.GlobalSKUIDs {
		if !bounded(id, 128) || wanted[id] {
			return marketplace.GlobalPublishResult{}, marketplace.ErrPublicationInvalid
		}
		wanted[id] = true
	}
	for _, sku := range input.Targets[0].SKUs {
		if !wanted[sku.RelatedGlobalSKUID] {
			return marketplace.GlobalPublishResult{}, marketplace.ErrPublicationInvalid
		}
		delete(wanted, sku.RelatedGlobalSKUID)
	}
	body, err := marshalMutation(input)
	if err != nil {
		return marketplace.GlobalPublishResult{}, err
	}
	raw, err := c.resourceMutation(ctx, "/product/202309/global_products/"+url.PathEscape(globalID)+"/publish", token, "", "application/json", body)
	if err != nil {
		return marketplace.GlobalPublishResult{}, err
	}
	return marketplace.DecodeGlobalPublishResult(raw, expected)
}
