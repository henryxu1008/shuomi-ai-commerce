package shopify

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
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

func TestReadDraftMediaEvidenceRequiresOriginalsForEachGalleryImage(t *testing.T) {
	const first = "https://shopify-shop-assets.storage.googleapis.com/shop/first?signature=fixture"
	const second = "https://shopify-shop-assets.storage.googleapis.com/shop/second?signature=fixture"
	original := existingDraftMediaFixture(first)
	video := `{"id":"gid://shopify/Video/11","status":"READY","mediaContentType":"VIDEO"}`
	image := `{"id":"gid://shopify/MediaImage/11","status":"READY","mediaContentType":"IMAGE","originalSource":{"url":"` + second + `","fileSize":"11"}}`
	for _, test := range []struct {
		name, body string
		reads      int32
		secondType string
		complete   bool
		err        error
	}{
		{name: "complete originals", body: strings.Replace(original, video, image, 1), reads: 2, complete: true},
		{name: "processing is still incomplete", body: strings.Replace(original, video, strings.Replace(image, "READY", "PROCESSING", 1), 1), reads: 1},
		{name: "unsafe second original is not downloaded", body: strings.Replace(original, video, strings.Replace(image, second, "https://elsewhere.example/second", 1), 1), reads: 1},
		{name: "non-image is visible and has no original hash", body: original, reads: 1, secondType: "VIDEO"},
		{name: "incomplete gallery page fails before any download", body: strings.Replace(original, "false", "true", 1), err: ErrDraftUnverified},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			admin := NewDevelopmentAdminClient(&http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return fixtureResponse(200, strings.Replace(installationFixture, `"currencyCode":"USD"`, `"currencyCode":"USD","plan":{"partnerDevelopment":true}`, 1)), nil
				}
				return fixtureResponse(200, test.body), nil
			})})
			var requests atomic.Int32
			images := &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
				requests.Add(1)
				if req.Method != http.MethodGet || req.URL.Host != draftOriginalHost || req.Header.Get("X-Shopify-Access-Token") != "" {
					t.Error("unexpected request for Shopify originals")
				}
				body := "image bytes"
				if req.URL.String() == second {
					body = "other bytes"
				}
				return fixtureResponse(200, body), nil
			})}
			got, err := admin.readDraftMediaEvidence(context.Background(), productInstallation(), "fixture-token", "gid://shopify/Product/55", images)
			if err != test.err || requests.Load() != test.reads || calls != 2 {
				t.Fatalf("unexpected draft evidence: reads %d, Admin queries %d, error %v", requests.Load(), calls, err)
			}
			if err != nil {
				return
			}
			secondType, secondID := test.secondType, "gid://shopify/MediaImage/11"
			if secondType == "" {
				secondType = "IMAGE"
			} else {
				secondID = "gid://shopify/Video/11"
			}
			if len(got.Gallery) != 2 || len(got.Variants) != 1 || got.Variants[0].SKU != "BLACK-S" ||
				got.Gallery[0].ID != "gid://shopify/MediaImage/9" || got.Gallery[0].Hash != sha256.Sum256([]byte("image bytes")) ||
				got.Gallery[1].ID != secondID || got.Gallery[1].Type != secondType {
				t.Fatalf("incorrect gallery or variant evidence: %+v", got)
			}
			if test.complete && got.Gallery[1].Hash != sha256.Sum256([]byte("other bytes")) ||
				!test.complete && got.Gallery[1].Hash != ([32]byte{}) {
				t.Fatalf("incorrect evidence for second image: %+v", got.Gallery[1])
			}
		})
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
