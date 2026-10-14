package tiktokshop

import (
	"html"
)

type ImageReference struct {
	AssetID string        `json:"assetId"`
	Image   UploadedImage `json:"image"`
}
type URIReference struct {
	URI string `json:"uri"`
}
type ProductContent struct {
	Title           string                  `json:"title"`
	Description     string                  `json:"description"`
	CategoryID      string                  `json:"category_id"`
	CategoryVersion string                  `json:"category_version"`
	BrandID         string                  `json:"brand_id,omitempty"`
	MainImages      []URIReference          `json:"main_images"`
	Weight          Weight                  `json:"package_weight"`
	Dimensions      *Dimensions             `json:"package_dimensions,omitempty"`
	Attributes      []ProductAttributeInput `json:"product_attributes,omitempty"`
	Certifications  []CertificationPayload  `json:"certifications,omitempty"`
	SizeChart       *SizeChartPayload       `json:"size_chart,omitempty"`
}
type CertificationPayload struct {
	ID             string         `json:"id"`
	Images         []URIReference `json:"images"`
	ExpirationDate int64          `json:"expiration_date,omitempty"`
}
type SizeChartPayload struct {
	Image URIReference `json:"image"`
}
type SalesAttributePayload struct {
	ID        string        `json:"id,omitempty"`
	Name      string        `json:"name,omitempty"`
	ValueID   string        `json:"value_id,omitempty"`
	ValueName string        `json:"value_name,omitempty"`
	Image     *URIReference `json:"sku_img,omitempty"`
}
type GlobalPrice struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}
type SalePrice struct {
	SalePrice string `json:"sale_price"`
	Currency  string `json:"currency"`
}
type GlobalSKU struct {
	SellerSKU       string                  `json:"seller_sku"`
	SalesAttributes []SalesAttributePayload `json:"sales_attributes,omitempty"`
	Price           GlobalPrice             `json:"price"`
	Inventory       []GlobalInventoryInput  `json:"inventory"`
}
type LocalSKU struct {
	SellerSKU       string                  `json:"seller_sku"`
	SalesAttributes []SalesAttributePayload `json:"sales_attributes,omitempty"`
	Price           SalePrice               `json:"price"`
	Inventory       []InventoryInput        `json:"inventory"`
}
type CreateGlobalRequest struct {
	ProductContent
	SKUs       []GlobalSKU `json:"skus"`
	ExternalID string      `json:"external_global_product_id"`
}
type CreateLocalRequest struct {
	ProductContent
	SKUs           []LocalSKU `json:"skus"`
	ExternalID     string     `json:"external_product_id"`
	SaveMode       string     `json:"save_mode"`
	IdempotencyKey string     `json:"idempotency_key"`
}
type GlobalPublishSKU struct {
	RelatedGlobalSKUID string         `json:"related_global_sku_id"`
	Price              SalePrice      `json:"price"`
	Inventory          InventoryInput `json:"inventory"`
}
type GlobalPublishTarget struct {
	Region string             `json:"region"`
	SKUs   []GlobalPublishSKU `json:"skus"`
}
type PublishGlobalRequest struct {
	Targets []GlobalPublishTarget `json:"publish_target"`
}

func imageMap(input DraftInput, references []ImageReference) (map[string]UploadedImage, error) {
	required := RequiredImages(input)
	if len(required) != len(references) {
		return nil, ErrPublicationInvalid
	}
	result := map[string]UploadedImage{}
	for _, ref := range references {
		key := ref.AssetID + "\x00" + ref.Image.UseCase
		if !identifier(ref.AssetID) || !ValidUploadedImage(ref.Image) {
			return nil, ErrPublicationInvalid
		}
		if _, duplicate := result[key]; duplicate {
			return nil, ErrPublicationInvalid
		}
		result[key] = ref.Image
	}
	for _, ref := range required {
		if _, found := result[ref.AssetID+"\x00"+ref.UseCase]; !found {
			return nil, ErrPublicationInvalid
		}
	}
	return result, nil
}
func productContent(input DraftInput, images map[string]UploadedImage) ProductContent {
	content := ProductContent{Title: input.Title, Description: "<p>" + html.EscapeString(input.Description) + "</p>", CategoryID: input.Selection.CategoryID, CategoryVersion: "v2", BrandID: input.BrandID, Weight: input.PackageWeight, Dimensions: input.PackageDimensions, Attributes: input.Attributes}
	for _, id := range input.MainImages {
		content.MainImages = append(content.MainImages, URIReference{images[id+"\x00MAIN_IMAGE"].URI})
	}
	for _, certification := range input.Certifications {
		value := CertificationPayload{ID: certification.ID, ExpirationDate: certification.ExpirationDate}
		for _, id := range certification.Images {
			value.Images = append(value.Images, URIReference{images[id+"\x00CERTIFICATION_IMAGE"].URI})
		}
		content.Certifications = append(content.Certifications, value)
	}
	if input.SizeChartAssetID != "" {
		content.SizeChart = &SizeChartPayload{Image: URIReference{images[input.SizeChartAssetID+"\x00SIZE_CHART_IMAGE"].URI}}
	}
	return content
}
func salesAttributes(input SKUInput, images map[string]UploadedImage) []SalesAttributePayload {
	var values []SalesAttributePayload
	for _, attribute := range input.SalesAttributes {
		value := SalesAttributePayload{ID: attribute.ID, Name: attribute.Name, ValueID: attribute.ValueID, ValueName: attribute.ValueName}
		if attribute.ImageAssetID != "" {
			value.Image = &URIReference{images[attribute.ImageAssetID+"\x00ATTRIBUTE_IMAGE"].URI}
		}
		values = append(values, value)
	}
	return values
}
func BuildGlobalCreate(input DraftInput, references []ImageReference, externalID string) (CreateGlobalRequest, error) {
	if input.Selection.Method != GlobalPublishing || !identifier(externalID) {
		return CreateGlobalRequest{}, ErrPublicationInvalid
	}
	images, err := imageMap(input, references)
	if err != nil {
		return CreateGlobalRequest{}, err
	}
	result := CreateGlobalRequest{ProductContent: productContent(input, images), ExternalID: externalID}
	for _, sku := range input.SKUs {
		result.SKUs = append(result.SKUs, GlobalSKU{SellerSKU: sku.SellerSKU, SalesAttributes: salesAttributes(sku, images), Price: GlobalPrice{Amount: sku.GlobalBasePrice, Currency: "USD"}, Inventory: sku.GlobalInventory})
	}
	return result, nil
}
func BuildLocalCreate(input DraftInput, references []ImageReference, externalID, key string) (CreateLocalRequest, error) {
	if input.Selection.Method != LocalReplication || !identifier(externalID) || !identifier(key) {
		return CreateLocalRequest{}, ErrPublicationInvalid
	}
	images, err := imageMap(input, references)
	if err != nil {
		return CreateLocalRequest{}, err
	}
	result := CreateLocalRequest{ProductContent: productContent(input, images), ExternalID: externalID, SaveMode: input.SaveMode, IdempotencyKey: key}
	for _, sku := range input.SKUs {
		result.SKUs = append(result.SKUs, LocalSKU{SellerSKU: sku.SellerSKU, SalesAttributes: salesAttributes(sku, images), Price: SalePrice{SalePrice: sku.SalePrice, Currency: "USD"}, Inventory: sku.Inventory})
	}
	return result, nil
}
func BuildGlobalPublish(input DraftInput, created CreatedGlobalProduct) (PublishGlobalRequest, error) {
	if input.Selection.Method != GlobalPublishing || !ValidCreatedGlobal(created) || len(input.SKUs) != len(created.SKUs) {
		return PublishGlobalRequest{}, ErrPublicationInvalid
	}
	ids := map[string]string{}
	for _, sku := range created.SKUs {
		ids[sku.SellerSKU] = sku.ID
	}
	target := GlobalPublishTarget{Region: "US"}
	seen := map[string]bool{}
	for _, sku := range input.SKUs {
		id, found := ids[sku.SellerSKU]
		if !found || seen[id] || len(sku.Inventory) != 1 {
			return PublishGlobalRequest{}, ErrPublicationInvalid
		}
		seen[id] = true
		target.SKUs = append(target.SKUs, GlobalPublishSKU{RelatedGlobalSKUID: id, Price: SalePrice{SalePrice: sku.SalePrice, Currency: "USD"}, Inventory: sku.Inventory[0]})
	}
	return PublishGlobalRequest{Targets: []GlobalPublishTarget{target}}, nil
}
