package shopify

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
)

func TestInspectDraftMediaReadsExactProductAndOriginal(t *testing.T) {
	const original = "https://shopify-shop-assets.storage.googleapis.com/shop/file?signature=secret"
	calls := 0
	admin := NewDevelopmentAdminClient(&http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			development := strings.Replace(installationFixture, `"currencyCode":"USD"`, `"currencyCode":"USD","plan":{"partnerDevelopment":true}`, 1)
			return fixtureResponse(200, development), nil
		}
		var document struct {
			Query     string
			Variables struct{ ID string }
		}
		if json.NewDecoder(req.Body).Decode(&document) != nil ||
			!strings.HasPrefix(document.Query, "query InspectDraftMedia") ||
			document.Variables.ID != "gid://shopify/Product/55" {
			t.Fatal("expected the fixed read-only query for the selected ID")
		}
		return fixtureResponse(200, existingDraftMediaFixture(original)), nil
	})})
	reads := 0
	images := &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		reads++
		if req.Method != http.MethodGet || req.URL.String() != original || req.Header.Get("X-Shopify-Access-Token") != "" {
			t.Fatal("unexpected original file request")
		}
		return fixtureResponse(200, "image bytes"), nil
	})}
	proof := DraftMediaFingerprint{Size: 11, Hash: sha256.Sum256([]byte("image bytes"))}
	got, err := admin.inspectDraftMedia(context.Background(), productInstallation(), "fixture-token", "gid://shopify/Product/55", []DraftMediaFingerprint{proof}, images)
	if err != nil || calls != 2 || reads != 1 || got.Title != "Existing" || got.GalleryCount != 2 ||
		len(got.Variants) != 1 || got.Variants[0].SKU != "BLACK-S" ||
		!slices.Equal(got.Variants[0].MediaIDs, []string{"gid://shopify/MediaImage/9"}) ||
		len(got.Images) != 1 || got.Images[0].MediaID != "gid://shopify/MediaImage/9" || got.Images[0].State != DraftMediaPresent {
		t.Fatalf("incorrect evidence (queries=%d reads=%d): %+v, %v", calls, reads, got, err)
	}
}

func TestInspectDraftMediaPreservesUnverifiedOriginals(t *testing.T) {
	for _, original := range []string{"https://example.com/non-Shopify", "https://shopify-shop-assets.storage.googleapis.com/shop/file"} {
		t.Run(original, func(t *testing.T) {
			calls := 0
			admin := makeProductClient(t, &calls, existingDraftMediaFixture(original), 200)
			images := &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Host != "shopify-shop-assets.storage.googleapis.com" {
					t.Fatal("request made to an unapproved host")
				}
				return fixtureResponse(403, "unavailable"), nil
			})}
			photo := DraftMediaFingerprint{Size: 11, Hash: sha256.Sum256([]byte("image bytes"))}
			got, err := admin.inspectDraftMedia(context.Background(), productInstallation(), "fixture-token", "gid://shopify/Product/55", []DraftMediaFingerprint{photo}, images)
			if err != nil || len(got.Images) != 1 || got.Images[0].MediaID != "" || got.Images[0].State != DraftMediaUnverified {
				t.Fatalf("uncertain upload evidence: %+v, %v", got, err)
			}
		})
	}
}

func TestInspectDraftMediaRejectsIncompleteAndNonDraftProducts(t *testing.T) {
	for _, body := range []string{
		strings.Replace(existingDraftMediaFixture(""), "DRAFT", "ACTIVE", 1),
		strings.Replace(existingDraftMediaFixture(""), "false", "true", 1),
		strings.Replace(existingDraftMediaFixture(""), "gid://shopify/Product/55", "gid://shopify/Product/77", 1),
	} {
		calls := 0
		admin := makeProductClient(t, &calls, body, 200)
		_, err := admin.inspectDraftMedia(context.Background(), productInstallation(), "fixture-token", "gid://shopify/Product/55", nil, nil)
		if err != ErrDraftUnverified || calls != 2 {
			t.Fatalf("accepted a different or incomplete product: %v", err)
		}
	}
}

func TestMatchDraftOriginalDistinguishesMissingFromAmbiguousEvidence(t *testing.T) {
	proof := DraftMediaFingerprint{Size: 11, Hash: sha256.Sum256([]byte("image bytes"))}
	found := draftOriginal{size: 11, hash: proof.Hash, mediaID: "media-1"}
	other := draftOriginal{size: 11, hash: sha256.Sum256([]byte("other bytes")), mediaID: "media-2"}
	duplicate := draftOriginal{size: 11, hash: proof.Hash, mediaID: "media-2"}
	unavailable := draftOriginal{size: 11, mediaID: "media-2"}
	for _, scenario := range []struct {
		media []draftOriginal
		want  DraftMediaState
	}{
		{[]draftOriginal{found, other}, DraftMediaPresent},
		{[]draftOriginal{other}, DraftMediaMissing},
		{[]draftOriginal{found, duplicate}, DraftMediaUnverified},
		{[]draftOriginal{found, unavailable}, DraftMediaUnverified},
	} {
		if got := matchDraftOriginal(proof, scenario.media); got.State != scenario.want {
			t.Fatalf("got %s, want %s", got.State, scenario.want)
		}
	}
}

func TestParseDraftMediaProductChecksTheSelectedShop(t *testing.T) {
	const domain = "demo-shop.myshopify.com"
	for _, candidate := range []string{
		"55", "https://demo-shop.myshopify.com/admin/products/55", "https://admin.shopify.com/store/demo-shop/products/55?ref=1",
	} {
		if id, err := ParseDraftMediaProduct(candidate, domain); err != nil || id != "gid://shopify/Product/55" {
			t.Fatalf("correct draft reference not parsed: %s, %v", id, err)
		}
	}
	for _, candidate := range []string{
		"https://other-shop.myshopify.com/admin/products/55", "https://admin.shopify.com/store/other-shop/products/55",
		"https://demo-shop.myshopify.com:443/admin/products/55", "https://demo-shop.myshopify.com/admin/products/5%35",
	} {
		if _, err := ParseDraftMediaProduct(candidate, domain); err != ErrInvalidRequest {
			t.Fatal("accepted a link outside the selected shop or with an invalid path")
		}
	}
}

func existingDraftMediaFixture(original string) string {
	return "{\"data\":{\"product\":{\"id\":\"gid://shopify/Product/55\",\"status\":\"DRAFT\",\"title\":\"Existing\"," +
		"\"media\":{\"pageInfo\":{\"hasNextPage\":false},\"nodes\":[" +
		"{\"id\":\"gid://shopify/MediaImage/9\",\"status\":\"READY\",\"mediaContentType\":\"IMAGE\",\"originalSource\":{\"url\":\"" + original + "\",\"fileSize\":\"11\"}}," +
		"{\"id\":\"gid://shopify/Video/11\",\"status\":\"READY\",\"mediaContentType\":\"VIDEO\"}]}," +
		"\"variants\":{\"pageInfo\":{\"hasNextPage\":false},\"nodes\":[" +
		"{\"id\":\"gid://shopify/ProductVariant/15\",\"sku\":\"BLACK-S\",\"selectedOptions\":[{\"name\":\"Color\",\"value\":\"Black\"}]," +
		"\"media\":{\"pageInfo\":{\"hasNextPage\":false},\"nodes\":[{\"id\":\"gid://shopify/MediaImage/9\"}]}}]}}}}"
}
