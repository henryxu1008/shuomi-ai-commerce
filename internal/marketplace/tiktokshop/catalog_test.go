package tiktokshop

import "testing"

func TestCategoriesRetainRootNodesWithoutGrantingPublicationPermission(t *testing.T) {
	values, err := Categories([]byte(`{"code":0,"data":{"categories":[{"id":"100","parent_id":"0","local_name":"Root","is_leaf":false},{"id":"101","parent_id":"100","local_name":"Leaf","is_leaf":true,"permission_statuses":["AVAILABLE"]}]}}`))
	if err != nil || len(values) != 2 {
		t.Fatalf("category tree rejected: %v", err)
	}
	if *values[0].Leaf || len(values[0].Permissions) != 0 {
		t.Fatal("root acquired publication permission")
	}
}

func TestAttributesPreserveOfficialRequiredSpellingAndRejectMissingFact(t *testing.T) {
	valid := []byte(`{"code":0,"data":{"attributes":[{"id":"1","name":"Material","type":"PRODUCT_PROPERTY","is_requried":true,"is_multiple_selection":false,"is_customizable":false,"values":[{"id":"v","name":"Cotton"}]}]}}`)
	values, err := DecodeAttributes(valid)
	if err != nil || len(values) != 1 || values[0].Required == nil || !*values[0].Required {
		t.Fatalf("official required flag not retained: %v", err)
	}
	if _, err := DecodeAttributes([]byte(`{"code":0,"data":{"attributes":[{"id":"1","type":"PRODUCT_PROPERTY","is_required":true}]}}`)); err == nil {
		t.Fatal("invented protocol spelling silently disabled a required attribute")
	}
}
