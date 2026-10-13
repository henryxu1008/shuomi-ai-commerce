// Package tiktokshop owns pure TikTok Shop publication rules and result values.
// A parsed result is not a Submission receipt or permission to call a provider;
// the owning Listing transaction binds it to its original intent and stages.
package tiktokshop

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"unicode/utf8"

	sigjson "sigs.k8s.io/json"
)

const MaxOfficialResultBytes = 2 << 20

var (
	ErrListingRules       = errors.New("TikTok Shop listing rules not verified")
	ErrListingMethod      = errors.New("TikTok Shop listing method not supported by current official rules")
	ErrPublicationInvalid = errors.New("invalid TikTok Shop publication expectation")
	ErrPublicationUnknown = errors.New("TikTok Shop publication result not verified")
)

type ListingMethod string

const (
	GlobalPublishing ListingMethod = "GLOBAL_PUBLISHING"
	LocalReplication ListingMethod = "LOCAL_REPLICATION"
)

type ListingRules struct {
	Methods     []ListingMethod
	Fingerprint string
}

// DecodeListingRules verifies the official method list. Warehouse/category
// rules remain separate required inputs; this result never signifies Ready.
func DecodeListingRules(raw []byte) (ListingRules, error) {
	var response struct {
		Code *int `json:"code"`
		Data *struct {
			Methods []ListingMethod `json:"listing_methods"`
		} `json:"data"`
	}
	if !decode(raw, &response) || response.Code == nil || *response.Code != 0 || response.Data == nil || len(response.Data.Methods) == 0 || len(response.Data.Methods) > 2 {
		return ListingRules{}, ErrListingRules
	}
	seen := map[ListingMethod]bool{}
	for _, method := range response.Data.Methods {
		if method != GlobalPublishing && method != LocalReplication || seen[method] {
			return ListingRules{}, ErrListingRules
		}
		seen[method] = true
	}
	return ListingRules{Methods: append([]ListingMethod(nil), response.Data.Methods...), Fingerprint: fingerprint(raw)}, nil
}

func (r ListingRules) RequireMethod(method ListingMethod) error {
	if r.Fingerprint == "" || method != GlobalPublishing && method != LocalReplication {
		return ErrListingMethod
	}
	for _, allowed := range r.Methods {
		if method == allowed {
			return nil
		}
	}
	return ErrListingMethod
}

// ChannelState describes provider business state, not kernel execution state.
// SUCCESS is under review; it is never projected as an approved/live product.
type ChannelState string

const (
	SubmittedForReview ChannelState = "submitted_for_review"
	RemoteDraft        ChannelState = "remote_draft"
	PublicationFailed  ChannelState = "publication_failed"
)

type GlobalPublishExpectation struct {
	ShopID       string
	GlobalSKUIDs []string
}

type PublishedSKU struct{ LocalSKUID, GlobalSKUID string }

type GlobalPublishResult struct {
	State             ChannelState
	ShopID, ProductID string
	SKUs              []PublishedSKU
	FailureReason     string
	FailureCount      int
	Fingerprint       string
}

// DecodeGlobalPublishResult correlates the sole approved US target with the
// actual Shop and complete global/local SKU set. DRAFT retains the created
// product; FAILED is a business observation and grants no retry permission.
func DecodeGlobalPublishResult(raw []byte, expected GlobalPublishExpectation) (GlobalPublishResult, error) {
	if !identifier(expected.ShopID) || len(expected.GlobalSKUIDs) == 0 || len(expected.GlobalSKUIDs) > 300 {
		return GlobalPublishResult{}, ErrPublicationInvalid
	}
	approved := make(map[string]bool, len(expected.GlobalSKUIDs))
	for _, id := range expected.GlobalSKUIDs {
		if !identifier(id) || approved[id] {
			return GlobalPublishResult{}, ErrPublicationInvalid
		}
		approved[id] = true
	}
	var response struct {
		Code *int `json:"code"`
		Data *struct {
			Products []struct {
				ID     string `json:"id"`
				ShopID string `json:"shop_id"`
				Region string `json:"region"`
				SKUs   []struct {
					ID       string `json:"id"`
					GlobalID string `json:"related_global_sku_id"`
				} `json:"skus"`
			} `json:"products"`
			Results []struct {
				Region      string `json:"region"`
				Status      string `json:"status"`
				FailReasons []*struct {
					Message string `json:"message"`
				} `json:"fail_reasons"`
			} `json:"publish_result"`
		} `json:"data"`
	}
	if !decode(raw, &response) || response.Code == nil || *response.Code != 0 || response.Data == nil || len(response.Data.Results) != 1 {
		return GlobalPublishResult{}, ErrPublicationUnknown
	}
	result := response.Data.Results[0]
	if result.Region != "US" {
		return GlobalPublishResult{}, ErrPublicationUnknown
	}
	for _, reason := range result.FailReasons {
		if reason == nil {
			return GlobalPublishResult{}, ErrPublicationUnknown
		}
	}
	out := GlobalPublishResult{ShopID: expected.ShopID, Fingerprint: fingerprint(raw), FailureCount: len(result.FailReasons)}
	switch result.Status {
	case "SUCCESS":
		if len(result.FailReasons) != 0 {
			return GlobalPublishResult{}, ErrPublicationUnknown
		}
		out.State = SubmittedForReview
	case "DRAFT":
		out.State, out.FailureReason = RemoteDraft, "official_validation_failed_saved_as_draft"
	case "FAILED":
		if len(response.Data.Products) != 0 {
			return GlobalPublishResult{}, ErrPublicationUnknown
		}
		out.State, out.FailureReason = PublicationFailed, "official_synchronization_failed"
		return out, nil
	default:
		return GlobalPublishResult{}, ErrPublicationUnknown
	}
	if len(response.Data.Products) != 1 {
		return GlobalPublishResult{}, ErrPublicationUnknown
	}
	product := response.Data.Products[0]
	if product.Region != "US" || product.ShopID != expected.ShopID || !identifier(product.ID) || len(product.SKUs) != len(approved) {
		return GlobalPublishResult{}, ErrPublicationUnknown
	}
	byGlobal := make(map[string]PublishedSKU, len(product.SKUs))
	localIDs := make(map[string]bool, len(product.SKUs))
	for _, sku := range product.SKUs {
		if !identifier(sku.ID) || !approved[sku.GlobalID] || localIDs[sku.ID] {
			return GlobalPublishResult{}, ErrPublicationUnknown
		}
		if _, duplicate := byGlobal[sku.GlobalID]; duplicate {
			return GlobalPublishResult{}, ErrPublicationUnknown
		}
		byGlobal[sku.GlobalID] = PublishedSKU{LocalSKUID: sku.ID, GlobalSKUID: sku.GlobalID}
		localIDs[sku.ID] = true
	}
	out.ProductID = product.ID
	for _, id := range expected.GlobalSKUIDs {
		out.SKUs = append(out.SKUs, byGlobal[id])
	}
	return out, nil
}

func identifier(value string) bool {
	if value == "" || len(value) > 128 || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}

func decode(raw []byte, target any) bool {
	if len(raw) == 0 || len(raw) > MaxOfficialResultBytes || !utf8.Valid(raw) {
		return false
	}
	strict, err := sigjson.UnmarshalStrict(raw, target, sigjson.DisallowDuplicateFields)
	return err == nil && len(strict) == 0
}

func fingerprint(raw []byte) string { hash := sha256.Sum256(raw); return hex.EncodeToString(hash[:]) }
