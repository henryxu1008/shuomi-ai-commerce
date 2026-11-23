package amazon

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
)

type ListingInput struct {
	ProductType string
	Attributes  json.RawMessage
}

type Issue struct {
	Code, Severity string
	AttributeNames []string
}

type PreviewResult struct {
	Valid  bool
	Issues []Issue
}

type SubmissionReceipt struct {
	Accepted     bool
	SubmissionID string
	Issues       []Issue
}

type ListingObservation struct {
	ASIN                  string
	Buyable, Discoverable bool
	Issues                []Issue
}

type putResponse struct {
	SKU, Status, SubmissionID string
	Issues                    []Issue
}

func (c *Client) Preview(ctx context.Context, accessToken, sellerID, sku string, input ListingInput) (PreviewResult, error) {
	result, err := c.put(ctx, accessToken, sellerID, sku, input, "VALIDATION_PREVIEW")
	if err != nil {
		return PreviewResult{}, err
	}
	switch result.Status {
	case "VALID", "INVALID":
		return PreviewResult{Valid: result.Status == "VALID", Issues: result.Issues}, nil
	default:
		return PreviewResult{}, ErrUnavailable
	}
}

func (c *Client) Submit(ctx context.Context, accessToken, sellerID, sku string, input ListingInput) (SubmissionReceipt, error) {
	result, err := c.put(ctx, accessToken, sellerID, sku, input, "")
	if err != nil {
		if err == ErrInvalid {
			return SubmissionReceipt{}, err
		}
		return SubmissionReceipt{}, ErrSubmissionUnknown
	}
	switch result.Status {
	case "ACCEPTED", "INVALID":
		if !visibleASCII(result.SubmissionID, 256) {
			return SubmissionReceipt{}, ErrSubmissionUnknown
		}
		return SubmissionReceipt{Accepted: result.Status == "ACCEPTED", SubmissionID: result.SubmissionID, Issues: result.Issues}, nil
	default:
		return SubmissionReceipt{}, ErrSubmissionUnknown
	}
}

func (c *Client) put(ctx context.Context, accessToken, sellerID, sku string, input ListingInput, mode string) (putResponse, error) {
	attrs := bytes.TrimSpace(input.Attributes)
	if !validIdentifier(sellerID, 64) || !validIdentifier(sku, 40) || !validIdentifier(input.ProductType, 128) ||
		len(attrs) < 2 || len(attrs) > maxPayload || attrs[0] != '{' || !json.Valid(attrs) {
		return putResponse{}, ErrInvalid
	}
	body, err := json.Marshal(struct {
		ProductType  string          `json:"productType"`
		Requirements string          `json:"requirements"`
		Attributes   json.RawMessage `json:"attributes"`
	}{input.ProductType, "LISTING", attrs})
	if err != nil {
		return putResponse{}, ErrInvalid
	}
	query := url.Values{"marketplaceIds": {USMarketplace}, "issueLocale": {"en_US"}}
	if mode != "" {
		query.Set("mode", mode)
	}
	raw, metadata, err := c.api(ctx, http.MethodPut, listingsPath+sellerID+"/"+sku, query, accessToken, body)
	if err != nil {
		return putResponse{}, err
	}
	if err := responseStatus(metadata.status); err != nil {
		return putResponse{}, err
	}
	var result putResponse
	if json.Unmarshal(raw, &result) != nil || result.SKU != sku {
		return putResponse{}, ErrUnavailable
	}
	result.Issues = safeIssues(result.Issues)
	return result, nil
}

func (c *Client) GetListing(ctx context.Context, accessToken, sellerID, sku string) (ListingObservation, error) {
	if !validIdentifier(sellerID, 64) || !validIdentifier(sku, 40) {
		return ListingObservation{}, ErrInvalid
	}
	query := url.Values{"marketplaceIds": {USMarketplace}, "includedData": {"summaries,issues"}, "issueLocale": {"en_US"}}
	raw, metadata, err := c.api(ctx, http.MethodGet, listingsPath+sellerID+"/"+sku, query, accessToken, nil)
	if err != nil {
		return ListingObservation{}, err
	}
	if metadata.status == http.StatusNotFound && visibleASCII(metadata.requestID, 128) {
		var envelope struct {
			Errors []struct{ Code string }
		}
		if json.Unmarshal(raw, &envelope) == nil && len(envelope.Errors) == 1 && validIdentifier(envelope.Errors[0].Code, 64) {
			return ListingObservation{}, ErrNotFound
		}
	}
	if err := responseStatus(metadata.status); err != nil {
		return ListingObservation{}, err
	}
	var result struct {
		SKU       string
		Summaries []struct {
			MarketplaceID, ASIN string
			Status              []string
		}
		Issues []Issue
	}
	if json.Unmarshal(raw, &result) != nil || result.SKU != sku || len(result.Summaries) != 1 {
		return ListingObservation{}, ErrUnavailable
	}
	summary := result.Summaries[0]
	if summary.MarketplaceID != USMarketplace || summary.Status == nil || (summary.ASIN != "" && (len(summary.ASIN) != 10 || !validIdentifier(summary.ASIN, 10))) {
		return ListingObservation{}, ErrUnavailable
	}
	return ListingObservation{
		ASIN: summary.ASIN, Buyable: slices.Contains(summary.Status, "BUYABLE"),
		Discoverable: slices.Contains(summary.Status, "DISCOVERABLE"), Issues: safeIssues(result.Issues),
	}, nil
}

// Leave the provider's message out of persisted or browser-facing data. Keep
// bounded issue codes and field names for the consumer to render in its own language.
func safeIssues(issues []Issue) []Issue {
	output := make([]Issue, 0, min(len(issues), 64))
	for _, issue := range issues[:min(len(issues), 64)] {
		if !validIdentifier(issue.Code, 64) {
			continue
		}
		switch issue.Severity {
		case "ERROR", "WARNING", "INFO":
		default:
			continue
		}
		fields := make([]string, 0, min(len(issue.AttributeNames), 20))
		for _, field := range issue.AttributeNames[:min(len(issue.AttributeNames), 20)] {
			if validIdentifier(field, 128) {
				fields = append(fields, field)
			}
		}
		output = append(output, Issue{issue.Code, issue.Severity, fields})
	}
	return output
}
