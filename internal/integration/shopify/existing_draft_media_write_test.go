package shopify

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExistingDraftMediaWritesUseSeparateBoundedRequests(t *testing.T) {
	ctx := context.Background()
	installation := Installation{ShopID: "gid://shopify/Shop/10", Domain: "demo-shop.myshopify.com", Currency: "USD",
		ID: "gid://shopify/AppInstallation/21", AppID: "gid://shopify/App/32"}
	const product = "gid://shopify/Product/44"
	const cover = "gid://shopify/MediaImage/87"
	const variant = "gid://shopify/ProductVariant/98"
	const token = "fixture-app-token"
	var calls []string
	var associationReply string
	admin := NewDevelopmentAdminClient(&http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, token, req.Header.Get("X-Shopify-Access-Token"))
		var body struct {
			Query     string                     `json:"query"`
			Variables map[string]json.RawMessage `json:"variables"`
		}
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		switch {
		case strings.Contains(body.Query, "ConfirmInstallation"):
			reply := strings.Replace(installationFixture, `"currencyCode":"USD"`, `"currencyCode":"USD","plan":{"partnerDevelopment":true}`, 1)
			return fixtureResponse(http.StatusOK, reply), nil
		case strings.Contains(body.Query, "stagedUploadsCreate"):
			calls = append(calls, "prepare")
			var inputs []stagedImageInput
			require.NoError(t, json.Unmarshal(body.Variables["input"], &inputs))
			require.Len(t, inputs, 1)
			require.Equal(t, "POST", inputs[0].Method)
			target := strings.ReplaceAll(stagedFixture, "image-1.png", inputs[0].Filename)
			return fixtureResponse(http.StatusOK, `{"data":{"stagedUploadsCreate":{"userErrors":[],"stagedTargets":[`+target+`]}}}`), nil
		case strings.Contains(body.Query, "productUpdate("):
			calls = append(calls, "append")
			require.JSONEq(t, fmt.Sprintf(`{"id":%q}`, product), string(body.Variables["product"]))
			var payload []struct{ MediaContentType, OriginalSource string }
			require.NoError(t, json.Unmarshal(body.Variables["media"], &payload))
			require.Len(t, payload, 1)
			require.Equal(t, "IMAGE", payload[0].MediaContentType)
			require.Contains(t, payload[0].OriginalSource, "/tmp/123/products/abc/")
			return fixtureResponse(http.StatusOK, fmt.Sprintf(`{"data":{"productUpdate":{"product":{"id":%q,"status":"DRAFT","media":{"pageInfo":{"hasNextPage":false},"nodes":[{"id":%q,"mediaContentType":"IMAGE"}]}},"userErrors":[]}}}`, product, cover)), nil
		case strings.Contains(body.Query, "productVariantsBulkUpdate("):
			calls = append(calls, "associate")
			require.Contains(t, body.Query, "allowPartialUpdates: false")
			require.JSONEq(t, fmt.Sprintf(`[{"id":%q,"mediaId":%q}]`, variant, cover), string(body.Variables["variants"]))
			require.JSONEq(t, fmt.Sprintf(`%q`, product), string(body.Variables["productId"]))
			if associationReply != "" {
				return fixtureResponse(http.StatusOK, associationReply), nil
			}
			return fixtureResponse(http.StatusOK, fmt.Sprintf(`{"data":{"productVariantsBulkUpdate":{"product":{"id":%q,"status":"DRAFT"},"productVariants":[{"id":%q}],"userErrors":[]}}}`, product, variant)), nil
		default:
			t.Fatalf("unexpected Admin query: %s", body.Query)
			return nil, nil
		}
	})})
	verify := func() *DraftMediaWrite {
		capability, err := admin.VerifyDraftMediaWrite(ctx, installation, token, product)
		require.NoError(t, err)
		return capability
	}
	contents := []byte("\x89PNG\r\n\x1a\nfixture")
	prepare := verify()
	targets, err := prepare.Prepare(ctx, [][32]byte{sha256.Sum256(contents)})
	require.NoError(t, err)
	require.Equal(t, []string{"prepare"}, calls)
	_, err = prepare.Prepare(ctx, [][32]byte{sha256.Sum256(contents)})
	require.ErrorIs(t, err, ErrInvalidRequest)
	uploadClient := &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		calls = append(calls, "upload")
		require.Equal(t, stagedPostURL, req.URL.String())
		require.Empty(t, req.Header.Get("Authorization"))
		require.Empty(t, req.Header.Get("X-Shopify-Access-Token"))
		return fixtureResponse(http.StatusCreated, fmt.Sprintf(`<PostResponse><Location>%s</Location><Bucket>%s</Bucket><Key>%s</Key><ETag>&quot;fixture&quot;</ETag></PostResponse>`,
			targets[0].target.ResourceURL, stagedBucket, strings.TrimPrefix(targets[0].target.ResourceURL, stagedPostURL))), nil
	})}
	_, err = verify().Append(ctx, targets)
	require.ErrorIs(t, err, ErrInvalidRequest, "unverified upload must not reach append")
	require.Equal(t, []string{"prepare"}, calls)
	require.ErrorIs(t, verify().upload(ctx, uploadClient, targets[0], append(contents, 1)), ErrInvalidRequest)
	truncatedXML := &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		return fixtureResponse(http.StatusCreated, "<PostResponse><Key>incomplete"), nil
	})}
	require.ErrorIs(t, verify().upload(ctx, truncatedXML, targets[0], contents), ErrDraftOutcomeUnknown)
	_, err = verify().Append(ctx, targets)
	require.ErrorIs(t, err, ErrInvalidRequest)
	require.ErrorIs(t, verify().upload(ctx, uploadClient, targets[0], contents), ErrInvalidRequest, "possibly sent target cannot send twice")
	targets, err = verify().Prepare(ctx, [][32]byte{sha256.Sum256(contents)})
	require.NoError(t, err)
	require.NoError(t, verify().upload(ctx, uploadClient, targets[0], contents))
	candidates, err := verify().Append(ctx, targets)
	require.NoError(t, err)
	require.Equal(t, []string{cover}, candidates)
	require.NoError(t, verify().Associate(ctx, []DraftMediaAssociation{{VariantID: variant, MediaID: cover}}))
	require.Equal(t, []string{"prepare", "prepare", "upload", "append", "associate"}, calls)
	changes := []DraftMediaAssociation{{VariantID: variant, MediaID: cover}}
	associationReply = `{"data":{"productVariantsBulkUpdate":{"product":null,"productVariants":null,"userErrors":[{"field":["variants","0","mediaId"],"message":"Invalid media"}]}}}`
	require.ErrorIs(t, verify().Associate(ctx, changes), ErrDraftMediaRejected)
	associationReply = fmt.Sprintf(`{"data":{"productVariantsBulkUpdate":{"product":null,"productVariants":[{"id":%q}],"userErrors":[{"field":[],"message":"Invalid media"}]}}}`, variant)
	require.ErrorIs(t, verify().Associate(ctx, changes), ErrDraftOutcomeUnknown)
}
