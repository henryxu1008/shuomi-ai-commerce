package tiktokshop

import (
	"encoding/json"
	"html"
	"math/big"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"task-processor/internal/product/asset"
)

type Weight struct {
	Value string `json:"value"`
	Unit  string `json:"unit"`
}
type Dimensions struct {
	Length string `json:"length"`
	Width  string `json:"width"`
	Height string `json:"height"`
	Unit   string `json:"unit"`
}
type InventoryInput struct {
	WarehouseID string `json:"warehouse_id"`
	Quantity    int64  `json:"quantity"`
}
type GlobalInventoryInput struct {
	WarehouseID string `json:"global_warehouse_id"`
	Quantity    int64  `json:"quantity"`
}
type ProductAttributeInput struct {
	ID     string           `json:"id"`
	Values []AttributeValue `json:"values"`
}
type SalesAttributeInput struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	ValueID      string `json:"value_id,omitempty"`
	ValueName    string `json:"value_name,omitempty"`
	ImageAssetID string `json:"imageAssetId,omitempty"`
}
type SKUInput struct {
	SellerSKU       string                 `json:"sellerSku"`
	SalePrice       string                 `json:"salePrice"`
	GlobalBasePrice string                 `json:"globalBasePrice,omitempty"`
	Inventory       []InventoryInput       `json:"inventory"`
	GlobalInventory []GlobalInventoryInput `json:"globalInventory,omitempty"`
	SalesAttributes []SalesAttributeInput  `json:"salesAttributes,omitempty"`
}
type CertificationInput struct {
	ID             string   `json:"id"`
	Images         []string `json:"images"`
	ExpirationDate int64    `json:"expirationDate,omitempty"`
}
type DraftInput struct {
	Selection         RuleSelection           `json:"selection"`
	Title             string                  `json:"title"`
	Description       string                  `json:"description"`
	BrandID           string                  `json:"brandId,omitempty"`
	SaveMode          string                  `json:"saveMode"`
	MainImages        []string                `json:"mainImages"`
	SKUs              []SKUInput              `json:"skus"`
	PackageWeight     Weight                  `json:"packageWeight"`
	PackageDimensions *Dimensions             `json:"packageDimensions,omitempty"`
	Attributes        []ProductAttributeInput `json:"attributes,omitempty"`
	Certifications    []CertificationInput    `json:"certifications,omitempty"`
	SizeChartAssetID  string                  `json:"sizeChartAssetId,omitempty"`
}
type ImageObservation struct {
	AssetID     string `json:"assetId"`
	UseCase     string `json:"useCase"`
	SourceURL   string `json:"sourceUrl"`
	ContentHash string `json:"contentHash"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Bytes       int64  `json:"bytes"`
	MediaType   string `json:"mediaType"`
}
type DraftIssue struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}
type Draft struct {
	ReadyForUpload bool         `json:"readyForUpload"`
	Issues         []DraftIssue `json:"issues"`
}

var positiveDecimal = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,3})?$`)
var positiveWhole = regexp.MustCompile(`^[0-9]+$`)
var numericAttribute = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`)

func text(value string, maximum int) bool {
	if !utf8.ValidString(value) || strings.TrimSpace(value) == "" || utf8.RuneCountInString(value) > maximum {
		return false
	}
	for _, r := range value {
		if unicode.Is(unicode.Han, r) || unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' || r >= 0x1f000 && r <= 0x1ffff {
			return false
		}
	}
	return true
}
func positive(value string, whole bool) bool {
	pattern := positiveDecimal
	if whole {
		pattern = positiveWhole
	}
	if len(value) > 14 || !pattern.MatchString(value) {
		return false
	}
	number, ok := new(big.Rat).SetString(value)
	return ok && number.Sign() > 0
}
func positiveAttribute(value string) bool {
	if len(value) > 2000 || !numericAttribute.MatchString(value) {
		return false
	}
	number, ok := new(big.Rat).SetString(value)
	return ok && number.Sign() > 0
}

func RequiredInUS(required *bool, regions []string, conditions []Condition, selected []ProductAttributeInput, method ListingMethod) (bool, error) {
	if required == nil {
		return false, ErrListingRules
	}
	if *required {
		return true, nil
	}
	for _, region := range regions {
		if region == "US" {
			return true, nil
		}
	}
	for _, condition := range conditions {
		if method == GlobalPublishing && condition.Region == "" {
			return false, ErrListingRules
		}
		if condition.Region != "" && condition.Region != "US" {
			continue
		}
		if condition.Type != "VALUE_ID_MATCH" || !identifier(condition.AttributeID) || !identifier(condition.ValueID) {
			return false, ErrListingRules
		}
		for _, attribute := range selected {
			if attribute.ID == condition.AttributeID {
				for _, value := range attribute.Values {
					if value.ID == condition.ValueID {
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}

func RequiredImages(input DraftInput) []ImageObservation {
	var images []ImageObservation
	for _, id := range input.MainImages {
		images = append(images, ImageObservation{AssetID: id, UseCase: "MAIN_IMAGE"})
	}
	for _, sku := range input.SKUs {
		for _, attribute := range sku.SalesAttributes {
			if attribute.ImageAssetID != "" {
				images = append(images, ImageObservation{AssetID: attribute.ImageAssetID, UseCase: "ATTRIBUTE_IMAGE"})
			}
		}
	}
	for _, certification := range input.Certifications {
		for _, id := range certification.Images {
			images = append(images, ImageObservation{AssetID: id, UseCase: "CERTIFICATION_IMAGE"})
		}
	}
	if input.SizeChartAssetID != "" {
		images = append(images, ImageObservation{AssetID: input.SizeChartAssetID, UseCase: "SIZE_CHART_IMAGE"})
	}
	unique := map[string]bool{}
	result := make([]ImageObservation, 0, len(images))
	for _, image := range images {
		key := image.AssetID + "\x00" + image.UseCase
		if !unique[key] {
			unique[key] = true
			result = append(result, image)
		}
	}
	return result
}

func ValidImageObservation(image ImageObservation) bool {
	if !identifier(image.AssetID) || len(image.ContentHash) != 64 || image.SourceURL == "" || image.Bytes < 1 || image.Bytes > 10<<20 || image.Width < 100 || image.Height < 100 || image.Width > 20000 || image.Height > 20000 || image.MediaType != "image/jpeg" && image.MediaType != "image/png" {
		return false
	}
	return ValidImageDimensions(image.UseCase, image.Width, image.Height)
}

func ValidImageDimensions(useCase string, width, height int) bool {
	if width < 100 || height < 100 || width > 20000 || height > 20000 {
		return false
	}
	switch useCase {
	case "MAIN_IMAGE":
		return width >= 300 && height >= 300 && width <= 4000 && height <= 4000
	case "SIZE_CHART_IMAGE":
		return width >= 1024 && height >= 1024
	case "ATTRIBUTE_IMAGE", "CERTIFICATION_IMAGE":
		return true
	default:
		return false
	}
}

// BuildDraft validates local, immutable references. Platform URIs are minted
// only by the later approved upload stages; they are never an input field.
func BuildDraft(input DraftInput, rules RuleSnapshot, inventory asset.ApprovedAssetInventory, images []ImageObservation) Draft {
	result := Draft{Issues: []DraftIssue{}}
	issue := func(field, code string) { result.Issues = append(result.Issues, DraftIssue{field, code}) }
	if raw, err := json.Marshal(input); err != nil || len(raw) > 512<<10 || len(input.SKUs) > 300 || len(input.Attributes) > 1000 || len(input.Certifications) > 10 || len(images) > 1000 || len(inventory.Assets) > 1000 {
		issue("input", "too_large")
		return result
	}
	if !ValidSelection(input.Selection) || input.Selection != rules.Selection || rules.Listing.RequireMethod(input.Selection.Method) != nil || rules.Category.ID != input.Selection.CategoryID || rules.Category.Leaf == nil || !*rules.Category.Leaf || len(rules.Category.Permissions) != 1 || rules.Category.Permissions[0] != "AVAILABLE" {
		issue("selection", "official_category_not_available")
	}
	if !text(input.Title, 255) {
		issue("title", "english_title_required")
	}
	if !text(input.Description, 10000) || utf8.RuneCountInString("<p>"+html.EscapeString(input.Description)+"</p>") > 10000 {
		issue("description", "english_description_required")
	}
	if input.SaveMode != "LISTING" && input.SaveMode != "AS_DRAFT" || input.Selection.Method == GlobalPublishing && input.SaveMode != "LISTING" {
		issue("saveMode", "unsupported_mode")
	}
	if input.BrandID != "" && !identifier(input.BrandID) {
		issue("brandId", "invalid")
	}
	if len(input.MainImages) < 1 || len(input.MainImages) > 9 {
		issue("mainImages", "one_to_nine_required")
	}
	if input.PackageWeight.Unit != "KILOGRAM" || !positive(input.PackageWeight.Value, false) {
		issue("packageWeight", "positive_metric_weight_required")
	}
	if input.PackageDimensions != nil {
		d := input.PackageDimensions
		if d.Unit != "CENTIMETER" || !positive(d.Length, true) || !positive(d.Width, true) || !positive(d.Height, true) {
			issue("packageDimensions", "positive_metric_dimensions_required")
		}
	}
	if input.Selection.Method == LocalReplication && (rules.Rules.PackageDimension == nil || rules.Rules.PackageDimension.Required == nil) {
		issue("packageDimensions", "official_dimension_rule_missing")
	}
	if rules.Rules.PackageDimension != nil && rules.Rules.PackageDimension.Required != nil && *rules.Rules.PackageDimension.Required && input.PackageDimensions == nil {
		issue("packageDimensions", "required")
	}
	if rules.Rules.SizeChart == nil || rules.Rules.SizeChart.Required == nil || rules.Rules.SizeChart.Supported == nil {
		issue("sizeChart", "official_size_rule_missing")
	} else if *rules.Rules.SizeChart.Required && input.SizeChartAssetID == "" || !*rules.Rules.SizeChart.Supported && input.SizeChartAssetID != "" {
		issue("sizeChart", "required_or_unsupported")
	}
	if len(input.SKUs) < 1 || len(input.SKUs) > 300 {
		issue("skus", "one_to_three_hundred_required")
	}
	validateAttributes(input, rules, issue)
	validateSKUs(input, rules, issue)
	validateCertifications(input, rules, issue)
	approved := map[string]asset.ApprovedAsset{}
	for _, image := range inventory.Assets {
		approved[image.ID] = image
	}
	observed := map[string]ImageObservation{}
	for _, image := range images {
		key := image.AssetID + "\x00" + image.UseCase
		if _, duplicate := observed[key]; duplicate {
			issue("images", "duplicate_observation")
		}
		observed[key] = image
	}
	for _, required := range RequiredImages(input) {
		image, found := approved[required.AssetID]
		observation, probed := observed[required.AssetID+"\x00"+required.UseCase]
		if inventory.Scope.TargetPlatform != "tiktokshop" || !found || image.Role == asset.RoleDesign || !probed || observation.SourceURL != image.URL || !ValidImageObservation(observation) {
			issue("images", "approved_image_not_verified")
		}
	}
	if raw, err := json.Marshal(input); err != nil || len(raw) > 512<<10 {
		issue("input", "too_large")
	}
	result.ReadyForUpload = len(result.Issues) == 0
	return result
}

func validateAttributes(input DraftInput, rules RuleSnapshot, issue func(string, string)) {
	provided := map[string]ProductAttributeInput{}
	definitions := map[string]AttributeRule{}
	if len(input.Attributes) > 1000 {
		issue("attributes", "too_many")
		return
	}
	for _, rule := range rules.Attributes {
		if rule.Type == "PRODUCT_PROPERTY" {
			definitions[rule.ID] = rule
			required, err := RequiredInUS(rule.Required, rule.RequiredRegions, rule.Conditions, input.Attributes, input.Selection.Method)
			if err != nil {
				issue("attributes", "unknown_requirement")
			}
			if required {
				found := false
				for _, value := range input.Attributes {
					if value.ID == rule.ID && len(value.Values) > 0 {
						found = true
					}
				}
				if !found {
					issue("attributes", "required")
				}
			}
		}
	}
	for _, attribute := range input.Attributes {
		rule, known := definitions[attribute.ID]
		if _, duplicate := provided[attribute.ID]; duplicate {
			issue("attributes", "duplicate")
		}
		provided[attribute.ID] = attribute
		if !known || len(attribute.Values) == 0 || len(attribute.Values) > 100 || len(attribute.Values) > 1 && (rule.Multiple == nil || !*rule.Multiple) {
			issue("attributes", "invalid_selection")
			continue
		}
		seen := map[string]bool{}
		for _, value := range attribute.Values {
			key := value.ID + "\x00" + value.Name
			if seen[key] {
				issue("attributes", "duplicate_value")
			}
			seen[key] = true
			if value.ID != "" {
				found := false
				for _, option := range rule.Values {
					if value.ID == option.ID {
						found = true
					}
				}
				if !found || value.Name != "" {
					issue("attributes", "unknown_value")
				}
			} else if rule.Customizable == nil || !*rule.Customizable || !text(value.Name, 2000) || rule.ValueDataFormat != "" && (rule.ValueDataFormat != "POSITIVE_INT_OR_DECIMAL" || !positiveAttribute(value.Name)) {
				issue("attributes", "invalid_custom_value")
			}
		}
	}
}

func validateSKUs(input DraftInput, rules RuleSnapshot, issue func(string, string)) {
	warehouses := map[string]Warehouse{}
	for _, warehouse := range rules.Warehouses {
		warehouses[warehouse.ID] = warehouse
	}
	globalWarehouses := map[string]bool{}
	for _, warehouse := range rules.GlobalWarehouses {
		if warehouse.Ownership == "SELLER" || warehouse.Ownership == "PLATFORM_COOPERATION" {
			globalWarehouses[warehouse.ID] = true
		}
	}
	sellers, variants := map[string]bool{}, map[string]bool{}
	var attributeTypes []string
	for index, sku := range input.SKUs {
		if !identifier(sku.SellerSKU) || utf8.RuneCountInString(sku.SellerSKU) > 50 || strings.IndexFunc(sku.SellerSKU, unicode.IsSpace) >= 0 || sellers[sku.SellerSKU] {
			issue("skus", "invalid_seller_sku")
		}
		sellers[sku.SellerSKU] = true
		var selected []Warehouse
		localIDs := map[string]bool{}
		localStock := int64(0)
		if len(sku.Inventory) < 1 || len(sku.Inventory) > 50 {
			issue("inventory", "explicit_warehouse_quantity_required")
		}
		for _, stock := range sku.Inventory {
			warehouse, known := warehouses[stock.WarehouseID]
			if !known || localIDs[stock.WarehouseID] || stock.Quantity < 1 || stock.Quantity > 99999 {
				issue("inventory", "invalid_warehouse_or_quantity")
				continue
			}
			localIDs[stock.WarehouseID] = true
			selected = append(selected, warehouse)
			localStock += stock.Quantity
		}
		if ValidateUSSalePrice(sku.SalePrice, selected) != nil {
			issue("salePrice", "outside_actual_warehouse_price_range")
		}
		if input.Selection.Method == GlobalPublishing {
			if len(sku.Inventory) != 1 {
				issue("inventory", "one_us_publication_warehouse_required")
			}
			if ValidateGlobalBasePrice(sku.GlobalBasePrice) != nil {
				issue("globalBasePrice", "required")
			}
			globalIDs := map[string]bool{}
			globalStock := int64(0)
			if len(sku.GlobalInventory) < 1 || len(sku.GlobalInventory) > 50 {
				issue("globalInventory", "explicit_global_budget_required")
			}
			for _, stock := range sku.GlobalInventory {
				if !globalWarehouses[stock.WarehouseID] || globalIDs[stock.WarehouseID] || stock.Quantity < 1 || stock.Quantity > 99999 {
					issue("globalInventory", "invalid_warehouse_or_quantity")
					continue
				}
				globalIDs[stock.WarehouseID] = true
				globalStock += stock.Quantity
			}
			if localStock > globalStock {
				issue("inventory", "exceeds_approved_global_budget")
			}
		} else if sku.GlobalBasePrice != "" || len(sku.GlobalInventory) > 0 {
			issue("skus", "global_fields_in_local_method")
		}
		if len(sku.SalesAttributes) > 3 || len(input.SKUs) > 1 && len(sku.SalesAttributes) == 0 {
			issue("salesAttributes", "variant_attributes_required")
		}
		types := make([]string, 0, len(sku.SalesAttributes))
		values := make([]string, 0, len(sku.SalesAttributes))
		seen := map[string]bool{}
		for _, attribute := range sku.SalesAttributes {
			name := attribute.ID
			if name == "" {
				name = attribute.Name
			}
			if name == "" || attribute.ID != "" && attribute.Name != "" || seen[name] {
				issue("salesAttributes", "invalid_or_duplicate_attribute")
			}
			seen[name] = true
			if attribute.ID != "" {
				known := false
				valueKnown := false
				for _, rule := range rules.Attributes {
					if rule.Type == "SALES_PROPERTY" && rule.ID == attribute.ID {
						known = true
						for _, value := range rule.Values {
							if value.ID == attribute.ValueID {
								valueKnown = true
							}
						}
					}
				}
				if !known || attribute.ValueID != "" && !valueKnown {
					issue("salesAttributes", "unknown_attribute_or_value")
				}
			} else if !text(attribute.Name, 20) {
				issue("salesAttributes", "invalid_custom_name")
			}
			if (attribute.ValueID == "") == (attribute.ValueName == "") || attribute.ValueName != "" && !text(attribute.ValueName, 50) {
				issue("salesAttributes", "invalid_value")
			}
			types = append(types, name)
			values = append(values, name+"\x00"+attribute.ValueID+"\x00"+attribute.ValueName)
		}
		sort.Strings(types)
		sort.Strings(values)
		if index == 0 {
			attributeTypes = types
		} else if strings.Join(types, "\x00") != strings.Join(attributeTypes, "\x00") {
			issue("salesAttributes", "inconsistent_variant_axes")
		}
		variant := strings.Join(values, "\x01")
		if len(input.SKUs) > 1 && variants[variant] {
			issue("skus", "duplicate_variant")
		}
		variants[variant] = true
	}
}

func validateCertifications(input DraftInput, rules RuleSnapshot, issue func(string, string)) {
	if len(input.Certifications) > 10 {
		issue("certifications", "too_many")
	}
	provided := map[string]CertificationInput{}
	known := map[string]bool{}
	for _, certification := range input.Certifications {
		if provided[certification.ID].ID != "" || !identifier(certification.ID) || len(certification.Images) < 1 || len(certification.Images) > 10 {
			issue("certifications", "invalid_or_duplicate")
		}
		provided[certification.ID] = certification
	}
	for _, rule := range rules.Rules.Certifications {
		known[rule.ID] = true
		required, err := RequiredInUS(rule.Required, rule.RequiredRegions, rule.Conditions, input.Attributes, input.Selection.Method)
		if err != nil {
			issue("certifications", "unknown_requirement")
		}
		value, exists := provided[rule.ID]
		if required && !exists {
			issue("certifications", "required")
		}
		if exists && rule.Expiration != nil && (rule.Expiration.Required == nil || *rule.Expiration.Required && value.ExpirationDate <= time.Now().Unix()) {
			issue("certifications", "expiration_required")
		}
		if exists && value.ExpirationDate != 0 && value.ExpirationDate <= time.Now().Unix() {
			issue("certifications", "expired")
		}
	}
	for id := range provided {
		if !known[id] {
			issue("certifications", "unknown_type")
		}
	}
}
