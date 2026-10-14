package tiktokshop

import (
	"encoding/json"
	"sort"
)

// These values contain only the listing fields needed by the US consumer.
// No warehouse contact/address details or arbitrary provider body is retained.
type RuleSelection struct {
	Method     ListingMethod `json:"method"`
	CategoryID string        `json:"categoryId"`
}

type Category struct {
	ID          string   `json:"id"`
	ParentID    string   `json:"parent_id"`
	Name        string   `json:"local_name"`
	Leaf        *bool    `json:"is_leaf"`
	Permissions []string `json:"permission_statuses"`
}

type Condition struct {
	Region      string `json:"region,omitempty"`
	Type        string `json:"condition_type"`
	AttributeID string `json:"attribute_id"`
	ValueID     string `json:"attribute_value_id"`
}

type AttributeValue struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type AttributeRule struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Type            string           `json:"type"`
	Required        *bool            `json:"is_requried"`
	Values          []AttributeValue `json:"values"`
	Multiple        *bool            `json:"is_multiple_selection"`
	Customizable    *bool            `json:"is_customizable"`
	ValueDataFormat string           `json:"value_data_format,omitempty"`
	RequiredRegions []string         `json:"required_regions,omitempty"`
	OptionalRegions []string         `json:"optional_regions,omitempty"`
	Conditions      []Condition      `json:"requirement_conditions,omitempty"`
}

type RequiredRule struct {
	Required *bool `json:"is_required"`
}
type SizeChartRule struct {
	Required  *bool `json:"is_required"`
	Supported *bool `json:"is_supported"`
}
type CertificationRule struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	DocumentDetails string        `json:"document_details,omitempty"`
	Required        *bool         `json:"is_required"`
	RequiredRegions []string      `json:"required_regions,omitempty"`
	OptionalRegions []string      `json:"optional_regions,omitempty"`
	Conditions      []Condition   `json:"requirement_conditions,omitempty"`
	Expiration      *RequiredRule `json:"expiration_date,omitempty"`
}
type CategoryRules struct {
	Certifications   []CertificationRule `json:"product_certifications"`
	SizeChart        *SizeChartRule      `json:"size_chart"`
	PackageDimension *RequiredRule       `json:"package_dimension,omitempty"`
}

type Warehouse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	EffectStatus string `json:"effect_status"`
	Type         string `json:"type"`
	SubType      string `json:"sub_type"`
}
type GlobalWarehouse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Ownership string `json:"ownership"`
}

type RuleSnapshot struct {
	Selection        RuleSelection     `json:"selection"`
	Listing          ListingRules      `json:"listing"`
	Category         Category          `json:"category"`
	Rules            CategoryRules     `json:"rules"`
	Attributes       []AttributeRule   `json:"attributes"`
	Warehouses       []Warehouse       `json:"warehouses"`
	GlobalWarehouses []GlobalWarehouse `json:"globalWarehouses,omitempty"`
}

func RuleHash(value RuleSnapshot) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	var canonical RuleSnapshot
	if json.Unmarshal(raw, &canonical) != nil {
		return ""
	}
	sort.Slice(canonical.Attributes, func(i, j int) bool { return canonical.Attributes[i].ID < canonical.Attributes[j].ID })
	for _, attribute := range canonical.Attributes {
		sort.Slice(attribute.Values, func(i, j int) bool { return attribute.Values[i].ID < attribute.Values[j].ID })
	}
	sort.Slice(canonical.Warehouses, func(i, j int) bool { return canonical.Warehouses[i].ID < canonical.Warehouses[j].ID })
	sort.Slice(canonical.GlobalWarehouses, func(i, j int) bool { return canonical.GlobalWarehouses[i].ID < canonical.GlobalWarehouses[j].ID })
	sort.Slice(canonical.Rules.Certifications, func(i, j int) bool {
		return canonical.Rules.Certifications[i].ID < canonical.Rules.Certifications[j].ID
	})
	raw, _ = json.Marshal(canonical)
	return fingerprint(raw)
}

func ValidSelection(value RuleSelection) bool {
	return (value.Method == GlobalPublishing || value.Method == LocalReplication) && identifier(value.CategoryID)
}

func Categories(raw []byte) ([]Category, error) {
	var response struct {
		Code *int `json:"code"`
		Data *struct {
			Categories []Category `json:"categories"`
		} `json:"data"`
	}
	if !decode(raw, &response) || response.Code == nil || *response.Code != 0 || response.Data == nil || len(response.Data.Categories) > 30000 {
		return nil, ErrListingRules
	}
	seen := map[string]bool{}
	for _, category := range response.Data.Categories {
		if !identifier(category.ID) || !identifier(category.ParentID) || category.Leaf == nil || seen[category.ID] {
			return nil, ErrListingRules
		}
		seen[category.ID] = true
	}
	return response.Data.Categories, nil
}

func DecodeCategoryRules(raw []byte) (CategoryRules, error) {
	var response struct {
		Code *int           `json:"code"`
		Data *CategoryRules `json:"data"`
	}
	if !decode(raw, &response) || response.Code == nil || *response.Code != 0 || response.Data == nil || len(response.Data.Certifications) > 100 {
		return CategoryRules{}, ErrListingRules
	}
	return *response.Data, nil
}

func DecodeAttributes(raw []byte) ([]AttributeRule, error) {
	var response struct {
		Code *int `json:"code"`
		Data *struct {
			Attributes []AttributeRule `json:"attributes"`
		} `json:"data"`
	}
	if !decode(raw, &response) || response.Code == nil || *response.Code != 0 || response.Data == nil || len(response.Data.Attributes) > 1000 {
		return nil, ErrListingRules
	}
	seen := map[string]bool{}
	for _, attribute := range response.Data.Attributes {
		if !identifier(attribute.ID) || seen[attribute.ID] || attribute.Type != "SALES_PROPERTY" && attribute.Type != "PRODUCT_PROPERTY" || attribute.Type == "PRODUCT_PROPERTY" && attribute.Required == nil || len(attribute.Values) > 10000 {
			return nil, ErrListingRules
		}
		seen[attribute.ID] = true
	}
	return response.Data.Attributes, nil
}
