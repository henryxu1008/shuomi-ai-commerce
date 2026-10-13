package tiktokshop

import (
	"errors"
	"strings"
	"testing"
)

func TestListingMethodComesFromExactOfficialRules(t *testing.T) {
	for _, method := range []ListingMethod{GlobalPublishing, LocalReplication} {
		rules, err := DecodeListingRules([]byte(`{"code":0,"data":{"listing_methods":["` + string(method) + `"],"inventory_rules":[]}}`))
		if err != nil || rules.Fingerprint == "" {
			t.Fatalf("rules: %+v, %v", rules, err)
		}
		if err := rules.RequireMethod(method); err != nil {
			t.Fatal(err)
		}
		other := GlobalPublishing
		if method == GlobalPublishing {
			other = LocalReplication
		}
		if err := rules.RequireMethod(other); !errors.Is(err, ErrListingMethod) {
			t.Fatal("unsupported method accepted")
		}
	}
	for _, raw := range []string{
		`{"data":{"listing_methods":["GLOBAL_PUBLISHING"]}}`,
		`{"code":0,"data":{"listing_methods":[]}}`,
		`{"code":0,"data":{"listing_methods":["FUTURE_METHOD"]}}`,
		`{"code":0,"data":{"listing_methods":["GLOBAL_PUBLISHING","GLOBAL_PUBLISHING"]}}`,
		`{"code":7,"code":0,"data":{"listing_methods":["GLOBAL_PUBLISHING"]}}`,
	} {
		if _, err := DecodeListingRules([]byte(raw)); err == nil {
			t.Fatalf("unqualified rule accepted: %s", raw)
		}
	}
}

func TestListingRuleHashIgnoresRequestTraceAndTracksInventorySemantics(t *testing.T) {
	first, err := DecodeListingRules([]byte(`{"code":0,"request_id":"trace-a","data":{"listing_methods":["LOCAL_REPLICATION"],"inventory_rules":[{"local_warehouse_id":"wh","allocation_mode":"MANUAL","associated_warehouses":[]}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	second, err := DecodeListingRules([]byte(`{"code":0,"request_id":"trace-b","data":{"listing_methods":["LOCAL_REPLICATION"],"inventory_rules":[{"local_warehouse_id":"wh","allocation_mode":"MANUAL","associated_warehouses":[]}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if first.Fingerprint != second.Fingerprint {
		t.Fatal("request trace changes invalidate unchanged listing rules")
	}
	changed, err := DecodeListingRules([]byte(`{"code":0,"data":{"listing_methods":["LOCAL_REPLICATION"],"inventory_rules":[{"local_warehouse_id":"wh","allocation_mode":"SHARED","associated_warehouses":[{"warehouse_id":"linked","region":"US"}]}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if first.Fingerprint == changed.Fingerprint {
		t.Fatal("inventory semantics changed without invalidating rules")
	}
}

func publishFixture(status, product string) []byte {
	reasons := `[{"message":"untrusted provider text"}]`
	if status == "SUCCESS" {
		reasons = "[]"
	}
	return []byte(`{"code":0,"request_id":"fixture-request","data":{"products":[` + product + `],"publish_result":[{"region":"US","status":"` + status + `","fail_reasons":` + reasons + `}]}}`)
}

const fixtureLocalProduct = `{"region":"US","shop_id":"approved-us-shop","id":"local-product","skus":[{"id":"local-sku-b","related_global_sku_id":"global-sku-b"},{"id":"local-sku-a","related_global_sku_id":"global-sku-a"}]}`

func TestGlobalPublicationDistinguishesReviewDraftAndFailedAndRetainsAssets(t *testing.T) {
	expectation := GlobalPublishExpectation{ShopID: "approved-us-shop", GlobalSKUIDs: []string{"global-sku-a", "global-sku-b"}}
	for _, state := range []struct {
		wire     string
		expected ChannelState
		product  string
	}{
		{"SUCCESS", SubmittedForReview, fixtureLocalProduct},
		{"DRAFT", RemoteDraft, fixtureLocalProduct},
		{"FAILED", PublicationFailed, ""},
	} {
		got, err := DecodeGlobalPublishResult(publishFixture(state.wire, state.product), expectation)
		if err != nil || got.State != state.expected || got.Fingerprint == "" {
			t.Fatalf("%s: %+v, %v", state.wire, got, err)
		}
		if state.product != "" && (got.ProductID != "local-product" || len(got.SKUs) != 2 || got.SKUs[0].GlobalSKUID != "global-sku-a") {
			t.Fatal("created asset/SKU linkage lost")
		}
		if strings.Contains(got.FailureReason, "untrusted") {
			t.Fatal("raw provider failure text exposed")
		}
		if got.State == RemoteDraft && got.ProductID == "" {
			t.Fatal("DRAFT treated as no side effect")
		}
	}
}

func TestGlobalPublishRejectsMissingDuplicateWrongShopAndSKUAssociations(t *testing.T) {
	expectation := GlobalPublishExpectation{ShopID: "approved-us-shop", GlobalSKUIDs: []string{"global-sku-a", "global-sku-b"}}
	valid := string(publishFixture("SUCCESS", fixtureLocalProduct))
	for _, raw := range []string{
		strings.Replace(valid, `"shop_id":"approved-us-shop"`, `"shop_id":"other-shop"`, 1),
		strings.Replace(valid, `"related_global_sku_id":"global-sku-b"`, `"related_global_sku_id":"global-sku-a"`, 1),
		strings.Replace(valid, `"id":"local-sku-b"`, `"id":"local-sku-a"`, 1),
		strings.Replace(valid, `"global-sku-b"`, `"unknown-global-sku"`, 1),
		strings.Replace(valid, `"publish_result":[{`, `"publish_result":[{"region":"US","status":"SUCCESS"},{`, 1),
		strings.Replace(valid, `"region":"US","status"`, `"region":"GB","status"`, 1),
		strings.Replace(valid, `"code":0`, `"code":12`, 1),
		strings.Replace(valid, `"code":0`, `"code":9,"code":0`, 1),
		string(publishFixture("SUCCESS", "")),
		string(publishFixture("DRAFT", "")),
		string(publishFixture("FAILED", fixtureLocalProduct)),
		string(publishFixture("ACTIVE", fixtureLocalProduct)),
	} {
		if _, err := DecodeGlobalPublishResult([]byte(raw), expectation); !errors.Is(err, ErrPublicationUnknown) {
			t.Fatalf("unqualified result accepted: %v", err)
		}
	}
	expectation.GlobalSKUIDs = []string{"global-sku-a", "global-sku-a"}
	if _, err := DecodeGlobalPublishResult([]byte(valid), expectation); !errors.Is(err, ErrPublicationInvalid) {
		t.Fatal("ambiguous approved SKU set accepted")
	}
}
