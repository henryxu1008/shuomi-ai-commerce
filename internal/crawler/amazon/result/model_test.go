package result_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	amazonresult "task-processor/internal/crawler/amazon/result"
)

func TestResultWireContract(t *testing.T) {
	zero := `{"title":"","brand":"","description":"","initial_price":0,"final_price":0,"currency":"","prices_breakdown":{},"availability":"","is_available":false,"reviews_count":0,"rating":0,"categories":null,"parent_asin":"","asin":"","seller_name":"","seller_id":"","url":"","image_url":"","images":null,"images_count":0,"features":null,"variations":null,"variations_values":null,"product_details":null,"zipcode":"","timestamp":null}`
	price, unit, text := 12.5, "unit", "text"
	empty := []string{}
	cases := []struct {
		name  string
		value any
		want  string
	}{
		{"product zero", amazonresult.Product{}, zero},
		{"product empty arrays", amazonresult.Product{Categories: []string{}, Images: []string{}, Features: []string{}, Variations: []amazonresult.Variation{}, VariationsValues: []amazonresult.VariationValue{}, ProductDetails: []amazonresult.ProductDetail{}}, `{"title":"","brand":"","description":"","initial_price":0,"final_price":0,"currency":"","prices_breakdown":{},"availability":"","is_available":false,"reviews_count":0,"rating":0,"categories":[],"parent_asin":"","asin":"","seller_name":"","seller_id":"","url":"","image_url":"","images":[],"images_count":0,"features":[],"variations":[],"variations_values":[],"product_details":[],"zipcode":"","timestamp":null}`},
		{"description", amazonresult.Description{Text: text, Type: "image", URL: "https://example.invalid/a"}, `{"text":"text","type":"image","url":"https://example.invalid/a"}`},
		{"price", amazonresult.PriceBreakdown{TypicalPrice: &price, ListPrice: &price, DealType: &text}, `{"typical_price":12.5,"list_price":12.5,"deal_type":"text"}`},
		{"buybox", amazonresult.BuyboxPrices{FinalPrice: price, UnitPrice: &unit}, `{"final_price":12.5,"unit_price":"unit"}`},
		{"rank", amazonresult.Subcategory{SubcategoryName: text, SubcategoryRank: 3}, `{"subcategory_name":"text","subcategory_rank":3}`},
		{"keywords nil", amazonresult.CustomersSay{}, `{"keywords":{}}`},
		{"keywords empty", amazonresult.CustomersSay{Text: &text, Keywords: amazonresult.CustomersKeywords{Positive: &empty, Negative: &empty, Mixed: &empty}}, `{"text":"text","keywords":{"positive":[],"negative":[],"mixed":[]}}`},
		{"inactive", amazonresult.InactiveBuyBox{Price: &price}, `{"price":12.5}`},
		{"variation", amazonresult.Variation{Name: "blue", Asin: "fixture", Image: "image", Attributes: map[string]any{"Size": "M", "Color": "blue"}}, `{"name":"blue","asin":"fixture","image":"image","attributes":{"Color":"blue","Size":"M"}}`},
		{"variation value", amazonresult.VariationValue{VariantName: "Color", Values: []string{"blue", "red"}}, `{"variant_name":"Color","values":["blue","red"]}`},
		{"detail", amazonresult.ProductDetail{Type: "material", Value: text}, `{"type":"material","value":"text"}`},
		{"size", amazonresult.SizeChart{Title: "size", Subtitle: text, Headers: []string{"cm"}, Rows: [][]string{{"2"}}, RawText: "raw"}, `{"title":"size","subtitle":"text","headers":["cm"],"rows":[["2"]],"raw_text":"raw"}`},
		{"video", amazonresult.VideoInfo{VideoID: "id", VideoURL: "url", ThumbnailURL: "thumbnail", Duration: 2, Title: text}, `{"video_id":"id","video_url":"url","thumbnail_url":"thumbnail","duration":2,"title":"text"}`},
		{"request", amazonresult.ProductRequest{URL: "url", Zipcode: "zip"}, `{"URL":"url","Zipcode":"zip"}`},
		{"result", amazonresult.ProductResult{}, `{"Product":null,"Error":null}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.value)
			if err != nil || string(got) != tc.want {
				t.Fatalf("wire = %s, error = %v; want %s", got, err, tc.want)
			}
		})
	}
	_, err := json.Marshal(amazonresult.Product{FinalPrice: math.NaN()})
	var unsupported *json.UnsupportedValueError
	if !errors.As(err, &unsupported) {
		t.Fatalf("non-finite price must retain JSON rejection: %v", err)
	}
}

func TestVariationValueDecoderContract(t *testing.T) {
	for _, tc := range []struct{ wire, want string }{
		{`{"variant_name":"Color","variant name":"Size","values":["blue"]}`, "Color"},
		{`{"variant_name":"","variant name":"Size","values":["blue"]}`, "Size"},
		{`{"variant name":"Size","values":["blue"]}`, "Size"},
	} {
		var got amazonresult.VariationValue
		if err := json.Unmarshal([]byte(tc.wire), &got); err != nil || got.VariantName != tc.want || !reflect.DeepEqual(got.Values, []string{"blue"}) {
			t.Fatalf("decode %s: %#v, %v", tc.wire, got, err)
		}
	}
	got := amazonresult.VariationValue{VariantName: "retained", Values: []string{"old"}}
	if err := got.UnmarshalJSON([]byte(`{"variant name":"ignored"}`)); err != nil || got.VariantName != "retained" || !reflect.DeepEqual(got.Values, []string{"old"}) {
		t.Fatalf("omitted fields must preserve receiver: %#v, %v", got, err)
	}
	for _, wire := range []string{`{`, `{"values":1}`} {
		if err := got.UnmarshalJSON([]byte(wire)); err == nil {
			t.Fatalf("invalid variation accepted: %s", wire)
		}
	}
}

func TestNullableTimeCodecContract(t *testing.T) {
	fixed := time.Date(2026, 1, 2, 3, 4, 5, 123000000, time.FixedZone("fixture", 8*3600))
	for _, wire := range []string{"", `null`, `""`} {
		got := amazonresult.NullableTime{Time: &fixed}
		if err := got.UnmarshalJSON([]byte(wire)); err != nil || got.Time != nil {
			t.Fatalf("nullable %q: %#v, %v", wire, got, err)
		}
		encoded, err := json.Marshal(got)
		if err != nil || string(encoded) != "null" {
			t.Fatalf("nil time wire %s: %v", encoded, err)
		}
	}
	const wire = `"2026-01-02T03:04:05.123+08:00"`
	var got amazonresult.NullableTime
	if err := json.Unmarshal([]byte(wire), &got); err != nil || got.Time == nil || !got.Equal(fixed) {
		t.Fatalf("time decode: %#v, %v", got, err)
	}
	encoded, err := json.Marshal(got)
	if err != nil || string(encoded) != wire {
		t.Fatalf("time wire %s: %v", encoded, err)
	}
	for _, bad := range []string{`"bad"`, `123`, `"2026-99-02T03:04:05Z"`} {
		before := got.Time
		if err := got.UnmarshalJSON([]byte(bad)); err == nil || got.Time != before {
			t.Fatalf("invalid time must fail without replacing value: %q, %v", bad, err)
		}
	}
}

func TestProductNotFoundErrorChainContract(t *testing.T) {
	cause := errors.New("synthetic cause")
	got := &amazonresult.ProductNotFoundError{ProductID: "fixture", Message: "missing", Cause: cause}
	if got.Error() != "产品不存在: missing (ProductID: fixture): synthetic cause" || got.Unwrap() != cause {
		t.Fatalf("not-found contract: %v", got)
	}
	wrapped := fmt.Errorf("outer: %w", got)
	var typed *amazonresult.ProductNotFoundError
	if !errors.Is(wrapped, cause) || !errors.As(wrapped, &typed) || typed != got {
		t.Fatal("not-found identity or cause lost")
	}
	got.Cause = nil
	if got.Error() != "产品不存在: missing (ProductID: fixture)" || got.Unwrap() != nil {
		t.Fatalf("not-found nil cause: %v", got)
	}
}
