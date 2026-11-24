package amazon

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

const (
	fixtureDefinitionSchema = `{"type":"object","required":["brand"],"properties":{"brand":{"type":"array"}}}`
	fixtureSignature        = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	fixtureDefinitionURL    = "https://selling-partner-definitions-prod-iad.s3.amazonaws.com/schema/LUGGAGE.json/synthetic?X-Amz-Signature=" + fixtureSignature
)

func TestAmazonProductTypeDefinitionUsesUSSellerAndVerifiesSchema(t *testing.T) {
	var requests int
	client := clientWithTransport(t, func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodGet {
			t.Fatalf("unexpected request method %s", req.Method)
		}
		switch requests {
		case 1:
			if req.URL.Host != "sellingpartnerapi-na.amazon.com" ||
				req.URL.Path != "/definitions/2020-09-01/productTypes/LUGGAGE" ||
				req.URL.Query().Get("marketplaceIds") != USMarketplace ||
				req.URL.Query().Get("sellerId") != "FIXTURESELLER1" ||
				req.URL.Query().Get("locale") != "en_US" ||
				req.URL.Query().Get("parentageLevel") != "NONE" ||
				req.URL.Query().Get("requirements") != "LISTING" ||
				req.URL.Query().Get("requirementsEnforced") != "ENFORCED" ||
				req.URL.Query().Get("productTypeVersion") != "LATEST" ||
				req.Header.Get("X-Amz-Access-Token") != "fixture-access" {
				t.Fatal("unexpected definition request")
			}
			return response(200, fixtureProductTypeDefinition(t, fixtureDefinitionURL, USMarketplace, fixtureDefinitionSchema)), nil
		case 2:
			if req.URL.String() != fixtureDefinitionURL || req.Header.Get("X-Amz-Access-Token") != "" ||
				req.Header.Get("X-Amz-Date") != "" || req.Header.Get("Authorization") != "" {
				t.Fatal("schema GET must use the signed address without seller credentials")
			}
			return response(200, fixtureDefinitionSchema), nil
		default:
			t.Fatal("unexpected extra request")
			return nil, nil
		}
	})
	definition, err := client.GetProductTypeDefinition(context.Background(), "fixture-access", "FIXTURESELLER1", "LUGGAGE")
	if err != nil || requests != 2 || string(definition.Schema) != fixtureDefinitionSchema ||
		definition.Fingerprint != sha256.Sum256([]byte(fixtureDefinitionSchema)) {
		t.Fatalf("definition requests=%d, error=%v", requests, err)
	}
}

func TestAmazonProductTypeDefinitionRejectsWrongScopeAndUnsafeDownloads(t *testing.T) {
	const bucketHost = "selling-partner-definitions-prod-iad.s3.amazonaws.com"
	for _, tc := range []struct {
		name, link, market, body string
		expectedCalls            int
		redirect                 bool
	}{
		{"wrong market", fixtureDefinitionURL, "A2EUQ1WTGCTBG2", "", 1, false},
		{"unknown bucket", "https://other-bucket.s3.amazonaws.com/schema/x?X-Amz-Signature=" + fixtureSignature, USMarketplace, "", 1, false},
		{"lookalike host", "https://" + bucketHost + ".example.org/schema/x?X-Amz-Signature=" + fixtureSignature, USMarketplace, "", 1, false},
		{"private host", "https://127.0.0.1/schema/x?X-Amz-Signature=" + fixtureSignature, USMarketplace, "", 1, false},
		{"user information", "https://x@" + bucketHost + "/schema/x?X-Amz-Signature=" + fixtureSignature, USMarketplace, "", 1, false},
		{"different port", "https://" + bucketHost + ":8080/schema/x?X-Amz-Signature=" + fixtureSignature, USMarketplace, "", 1, false},
		{"wrong object prefix", "https://" + bucketHost + "/x?X-Amz-Signature=" + fixtureSignature, USMarketplace, "", 1, false},
		{"encoded dot segment", "https://" + bucketHost + "/schema/%2e%2e/x?X-Amz-Signature=" + fixtureSignature, USMarketplace, "", 1, false},
		{"unsigned address", "https://" + bucketHost + "/schema/x", USMarketplace, "", 1, false},
		{"unsigned query", "https://" + bucketHost + "/schema/x?random=1", USMarketplace, "", 1, false},
		{"duplicate signature", fixtureDefinitionURL + "&X-Amz-Signature=" + fixtureSignature, USMarketplace, "", 1, false},
		{"checksum mismatch", fixtureDefinitionURL, USMarketplace, `{"type":"string"}`, 2, false},
		{"redirect", fixtureDefinitionURL, USMarketplace, "", 2, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls int
			client := clientWithTransport(t, func(_ *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return response(200, fixtureProductTypeDefinition(t, tc.link, tc.market, fixtureDefinitionSchema)), nil
				}
				if tc.redirect {
					out := response(302, "")
					out.Header.Set("Location", "https://other-bucket.s3.amazonaws.com/redirect")
					return out, nil
				}
				return response(200, tc.body), nil
			})
			_, err := client.GetProductTypeDefinition(context.Background(), "fixture-access", "FIXTURESELLER1", "LUGGAGE")
			if !errors.Is(err, ErrUnavailable) || calls != tc.expectedCalls {
				t.Fatalf("requests=%d, error=%v", calls, err)
			}
		})
	}
}

func fixtureProductTypeDefinition(t *testing.T, link, market, schema string) string {
	t.Helper()
	checksum := md5.Sum([]byte(schema))
	body, err := json.Marshal(map[string]any{
		"requirements": "LISTING", "requirementsEnforced": "ENFORCED",
		"productType": "LUGGAGE", "locale": "en_US", "marketplaceIds": []string{market},
		"productTypeVersion": map[string]any{"version": "synthetic-v1", "latest": true, "releaseCandidate": false},
		"schema": map[string]any{
			"link":     map[string]string{"verb": "GET", "resource": link},
			"checksum": base64.StdEncoding.EncodeToString(checksum[:]),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
