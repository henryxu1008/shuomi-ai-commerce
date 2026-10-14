package tiktokshop

import (
	"net/url"
	"sort"
	"strings"
)

type UploadedImage struct {
	URI          string `json:"uri"`
	URL          string `json:"url"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	UseCase      string `json:"use_case"`
	ResponseHash string `json:"responseHash,omitempty"`
}
type CreatedSKU struct {
	ID        string `json:"id"`
	SellerSKU string `json:"seller_sku"`
}
type CreatedGlobalProduct struct {
	ID           string       `json:"id"`
	SKUs         []CreatedSKU `json:"skus"`
	ResponseHash string       `json:"responseHash"`
}
type CreatedLocalProduct struct {
	ID           string       `json:"id"`
	SKUs         []CreatedSKU `json:"skus"`
	ResponseHash string       `json:"responseHash"`
}

func validURI(value string) bool {
	return value != "" && len(value) <= 4096 && strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\x00\r\n")
}
func ValidUploadedImage(value UploadedImage) bool {
	parsed, err := url.Parse(value.URL)
	if !validURI(value.URI) || !validURI(value.URL) || err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return false
	}
	return ValidImageDimensions(value.UseCase, value.Width, value.Height)
}
func DecodeUploadedImage(raw []byte, useCase string) (UploadedImage, error) {
	var response struct {
		Code *int           `json:"code"`
		Data *UploadedImage `json:"data"`
	}
	if !decode(raw, &response) || response.Code == nil || *response.Code != 0 || response.Data == nil || response.Data.UseCase != useCase || !ValidUploadedImage(*response.Data) {
		return UploadedImage{}, ErrPublicationUnknown
	}
	response.Data.ResponseHash = fingerprint(raw)
	return *response.Data, nil
}

func validCreatedSKUs(skus []CreatedSKU, expected []string) bool {
	if len(expected) < 1 || len(expected) > 300 || len(skus) != len(expected) {
		return false
	}
	wanted := map[string]bool{}
	ids := map[string]bool{}
	sellers := map[string]bool{}
	for _, seller := range expected {
		if !identifier(seller) || wanted[seller] {
			return false
		}
		wanted[seller] = true
	}
	for _, sku := range skus {
		if !identifier(sku.ID) || !wanted[sku.SellerSKU] || ids[sku.ID] || sellers[sku.SellerSKU] {
			return false
		}
		ids[sku.ID] = true
		sellers[sku.SellerSKU] = true
	}
	return true
}
func DecodeCreatedGlobal(raw []byte, expected []string) (CreatedGlobalProduct, error) {
	var response struct {
		Code *int `json:"code"`
		Data *struct {
			ID   string       `json:"global_product_id"`
			SKUs []CreatedSKU `json:"global_skus"`
		} `json:"data"`
	}
	if !decode(raw, &response) || response.Code == nil || *response.Code != 0 || response.Data == nil || !identifier(response.Data.ID) || !validCreatedSKUs(response.Data.SKUs, expected) {
		return CreatedGlobalProduct{}, ErrPublicationUnknown
	}
	sort.Slice(response.Data.SKUs, func(i, j int) bool { return response.Data.SKUs[i].SellerSKU < response.Data.SKUs[j].SellerSKU })
	return CreatedGlobalProduct{ID: response.Data.ID, SKUs: response.Data.SKUs, ResponseHash: fingerprint(raw)}, nil
}
func DecodeCreatedLocal(raw []byte, expected []string) (CreatedLocalProduct, error) {
	var response struct {
		Code *int `json:"code"`
		Data *struct {
			ID   string       `json:"product_id"`
			SKUs []CreatedSKU `json:"skus"`
		} `json:"data"`
	}
	if !decode(raw, &response) || response.Code == nil || *response.Code != 0 || response.Data == nil || !identifier(response.Data.ID) || !validCreatedSKUs(response.Data.SKUs, expected) {
		return CreatedLocalProduct{}, ErrPublicationUnknown
	}
	sort.Slice(response.Data.SKUs, func(i, j int) bool { return response.Data.SKUs[i].SellerSKU < response.Data.SKUs[j].SellerSKU })
	return CreatedLocalProduct{ID: response.Data.ID, SKUs: response.Data.SKUs, ResponseHash: fingerprint(raw)}, nil
}

func ValidCreatedGlobal(value CreatedGlobalProduct) bool {
	sellers := make([]string, 0, len(value.SKUs))
	for _, sku := range value.SKUs {
		sellers = append(sellers, sku.SellerSKU)
	}
	return identifier(value.ID) && len(value.ResponseHash) == 64 && validCreatedSKUs(value.SKUs, sellers)
}
func ValidCreatedLocal(value CreatedLocalProduct) bool {
	sellers := make([]string, 0, len(value.SKUs))
	for _, sku := range value.SKUs {
		sellers = append(sellers, sku.SellerSKU)
	}
	return identifier(value.ID) && len(value.ResponseHash) == 64 && validCreatedSKUs(value.SKUs, sellers)
}

func CorrelatesCreatedSKUs(skus []CreatedSKU, input DraftInput) bool {
	if len(input.SKUs) == 0 || len(input.SKUs) > 300 {
		return false
	}
	expected := make([]string, 0, len(input.SKUs))
	for _, sku := range input.SKUs {
		expected = append(expected, sku.SellerSKU)
	}
	return validCreatedSKUs(skus, expected)
}

func ValidGlobalPublishResult(value GlobalPublishResult, shopID string) bool {
	if value.ShopID != shopID || !identifier(shopID) || len(value.Fingerprint) != 64 || value.FailureCount < 0 {
		return false
	}
	if value.State == PublicationFailed {
		return value.ProductID == "" && len(value.SKUs) == 0 && value.FailureReason == "official_synchronization_failed"
	}
	if value.State != RemoteDraft && value.State != SubmittedForReview || !identifier(value.ProductID) || len(value.SKUs) < 1 || len(value.SKUs) > 300 {
		return false
	}
	if value.State == SubmittedForReview && (value.FailureReason != "" || value.FailureCount != 0) || value.State == RemoteDraft && value.FailureReason != "official_validation_failed_saved_as_draft" {
		return false
	}
	local, global := map[string]bool{}, map[string]bool{}
	for _, sku := range value.SKUs {
		if !identifier(sku.LocalSKUID) || !identifier(sku.GlobalSKUID) || local[sku.LocalSKUID] || global[sku.GlobalSKUID] {
			return false
		}
		local[sku.LocalSKUID] = true
		global[sku.GlobalSKUID] = true
	}
	return true
}
func CorrelatesGlobalPublication(value GlobalPublishResult, input PublishGlobalRequest, shopID string) bool {
	if !ValidGlobalPublishResult(value, shopID) || len(input.Targets) != 1 || input.Targets[0].Region != "US" || len(input.Targets[0].SKUs) < 1 {
		return false
	}
	wanted := map[string]bool{}
	for _, sku := range input.Targets[0].SKUs {
		if !identifier(sku.RelatedGlobalSKUID) || wanted[sku.RelatedGlobalSKUID] {
			return false
		}
		wanted[sku.RelatedGlobalSKUID] = true
	}
	if value.State == PublicationFailed {
		return true
	}
	if len(value.SKUs) != len(wanted) {
		return false
	}
	for _, sku := range value.SKUs {
		if !wanted[sku.GlobalSKUID] {
			return false
		}
	}
	return true
}
