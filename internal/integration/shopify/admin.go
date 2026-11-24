// Package shopify implements the bounded Shopify Admin API protocol.
// Store Center owns merchant consent and authorization. Listing owns sending permits and receipts.
package shopify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	sigjson "sigs.k8s.io/json"
)

const (
	adminPath         = "/admin/api/2026-10/graphql.json"
	adminTimeout      = 5 * time.Second
	maxAdminResponse  = 1 << 20
	productWriteScope = "write_products"
)

var (
	ErrInvalidRequest = errors.New("invalid Shopify request")
	errAdminResponse  = errors.New("Shopify Admin API response unavailable")
	shopDomainPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.myshopify\.com$`)
	currencyPattern   = regexp.MustCompile(`^[A-Z]{3}$`)
)

type AdminClient struct {
	http *http.Client
}

func NewAdminClient(transport *http.Client) *AdminClient {
	return &AdminClient{http: boundedClient(transport)}
}

func boundedClient(transport *http.Client) *http.Client {
	client := http.Client{Timeout: adminTimeout}
	if transport != nil {
		client = *transport
	}
	if client.Timeout <= 0 || client.Timeout > adminTimeout {
		client.Timeout = adminTimeout
	}
	client.Jar = nil
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &client
}

func (c *AdminClient) query(ctx context.Context, shop, token, document string, variables any, output any) error {
	if c == nil || !validToken(token) {
		return ErrInvalidRequest
	}
	wire, err := json.Marshal(struct {
		Query     string `json:"query"`
		Variables any    `json:"variables,omitempty"`
	}{document, variables})
	if err != nil {
		return ErrInvalidRequest
	}
	raw, err := postShopify(ctx, c.http, shop, adminPath, token, "application/json", wire, maxAdminResponse)
	if err != nil {
		return err
	}
	var reply struct {
		Data   json.RawMessage   `json:"data"`
		Errors []json.RawMessage `json:"errors"`
	}
	if !decodeAdminJSON(raw, &reply) || len(reply.Errors) != 0 || len(reply.Data) == 0 || string(reply.Data) == "null" ||
		!decodeAdminJSON(reply.Data, output) {
		return errAdminResponse
	}
	return nil
}

func postShopify(ctx context.Context, client *http.Client, shop, path, token, contentType string, wire []byte, limit int64) ([]byte, error) {
	if client == nil || ctx == nil || ctx.Err() != nil || !shopDomainPattern.MatchString(shop) {
		return nil, ErrInvalidRequest
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+shop+path, bytes.NewReader(wire))
	if err != nil {
		return nil, ErrInvalidRequest
	}
	if token != "" {
		req.Header.Set("X-Shopify-Access-Token", token)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, errAdminResponse
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errAdminResponse
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || int64(len(raw)) > limit {
		return nil, errAdminResponse
	}
	return raw, nil
}

func decodeAdminJSON(raw []byte, output any) bool {
	mismatches, err := sigjson.UnmarshalStrict(raw, output, sigjson.DisallowDuplicateFields)
	return err == nil && len(mismatches) == 0
}

func validToken(token string) bool {
	return token != "" && len(token) <= 4096 && strings.IndexFunc(token, func(r rune) bool {
		return r < '!' || r > '~'
	}) == -1
}

func validGID(gid, kind string) bool {
	number, ok := strings.CutPrefix(gid, "gid://shopify/"+kind+"/")
	if !ok || number == "" || len(number) > 20 || number[0] < '1' || number[0] > '9' {
		return false
	}
	_, err := strconv.ParseUint(number, 10, 64)
	return err == nil
}
