package browserclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	sigjson "sigs.k8s.io/json"
	"strings"
	"task-processor/internal/integration/acquisition/browsercollector"
	"task-processor/internal/product/sourcing"
)

func (c *Client) Discover(ctx context.Context, input sourcing.DiscoveryInput) (sourcing.DiscoveryEvidence, error) {
	var result sourcing.DiscoveryEvidence
	if c == nil || c.httpClient == nil {
		return result, sourcing.ErrAcquisitionUnavailable
	}
	body, _ := json.Marshal(input)
	probe, _ := http.NewRequest(http.MethodPost, "http://localhost", bytes.NewReader(body))
	if _, e := browsercollector.ReadDiscoveryRequest(probe); e != nil {
		return result, e
	}
	endpoint := strings.TrimSuffix(c.endpoint, browsercollector.AcquirePath) + browsercollector.DiscoverPath
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if e != nil {
		return result, sourcing.ErrAcquisitionUnavailable
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")
	if c.admission != nil {
		if e = c.admission(req); e != nil {
			return result, sourcing.ErrAcquisitionUnavailable
		}
	}
	response, e := c.httpClient.Do(req)
	if e != nil {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		return result, sourcing.ErrAcquisitionUnavailable
	}
	defer response.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(response.Body, browsercollector.MaxResponseBytes+1))
	if e != nil || len(raw) > browsercollector.MaxResponseBytes {
		return result, sourcing.ErrAcquisitionUnavailable
	}
	if response.StatusCode != http.StatusOK {
		return result, decodeErrorStatus(response.StatusCode, raw)
	}
	strict, e := sigjson.UnmarshalStrict(raw, &result, sigjson.DisallowUnknownFields, sigjson.DisallowDuplicateFields)
	if e != nil || len(strict) > 0 || result.SourceURL != input.URL || len(result.Products) > sourcing.MaxPlanProducts || len(result.Products) == 0 {
		return result, sourcing.ErrAcquisitionUnavailable
	}
	seen := map[string]bool{}
	for _, raw := range result.Products {
		source, e := sourcing.Canonical1688Source(raw)
		if e != nil || source.URL != raw || seen[raw] {
			return result, sourcing.ErrAcquisitionUnavailable
		}
		seen[raw] = true
	}
	return result, nil
}
