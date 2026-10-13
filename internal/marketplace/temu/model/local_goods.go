package model

// LocalGoodsV2 follows E17's current explicit creation contract. These fields
// are outer business parameters, not a nested request object. There is no
// client-selected mallId or siteId: the original regional token owns the mall.
// Optional facts stay absent; required quantity distinguishes missing from zero.
type LocalGoodsV2 struct {
	Language                string                `json:"language,omitempty"`
	Basic                   LocalGoodsBasic       `json:"goodsBasic"`
	ServicePromise          LocalServicePromise   `json:"goodsServicePromise"`
	Properties              []LocalProperty       `json:"goodsProperty,omitempty"`
	Origin                  *LocalOrigin          `json:"goodsOriginInfo,omitempty"`
	Size                    *LocalSize            `json:"goodsSize,omitempty"`
	SKUs                    []LocalSKU            `json:"skuList"`
	VehiclePropertyRelation *LocalVehicleRelation `json:"goodsVehiclePropertyRelation,omitempty"`
	ReferenceGoods          *LocalReferenceGoods  `json:"referenceGoods,omitempty"`
	Certification           *LocalCertification   `json:"certificationInfo,omitempty"`
}

type LocalGoodsBasic struct {
	Name            string           `json:"goodsName"`
	CatID           int64            `json:"catId"`
	Description     string           `json:"goodsDesc,omitempty"`
	Brand           *LocalBrand      `json:"brand,omitempty"`
	BulletPoints    []string         `json:"bulletPoints,omitempty"`
	Gallery         *LocalGallery    `json:"goodsGallery,omitempty"`
	ExternalGoodsID string           `json:"externalGoodsId,omitempty"`
	ProductType     *int64           `json:"productType,omitempty"`
	ItemTaxCode     string           `json:"itemTaxCode,omitempty"`
	SecondHand      *LocalSecondHand `json:"secondHand,omitempty"`
}

type LocalBrand struct {
	TrademarkID *int64 `json:"trademarkId,omitempty"`
	NoTrademark *bool  `json:"noTrademark,omitempty"`
}
type LocalGallery struct {
	DetailVideoID   string   `json:"detailVideoId,omitempty"`
	DetailImages    []string `json:"detailImage,omitempty"`
	CarouselVideoID string   `json:"carouselVideoId,omitempty"`
	CarouselImages  []string `json:"goodsCarouselImage,omitempty"`
}

// E17 uses 0/1/2 for businessScope while E29 describes 1/2/3. Do not
// infer a conversion or select a default until official evidence resolves it.
type LocalSecondHand struct {
	BusinessScope *int64 `json:"businessScope,omitempty"`
	Level         *int64 `json:"level,omitempty"`
	AgencyName    string `json:"insName,omitempty"`
	Grade         string `json:"grade,omitempty"`
}
type LocalServicePromise struct {
	ShipmentLimitDay *int64 `json:"shipmentLimitDay"`
	FulfillmentType  *int64 `json:"fulfillmentType,omitempty"`
	CostTemplateID   string `json:"costTemplateId"`
}
type LocalProperty struct {
	RefPID           int64  `json:"refPid"`
	VID              *int64 `json:"vid,omitempty"`
	Value            string `json:"value,omitempty"`
	NumberInputValue string `json:"numberInputValue,omitempty"`
	ValueUnitID      *int64 `json:"valueUnitId,omitempty"`
}
type LocalOrigin struct {
	Region1                string   `json:"originRegion1,omitempty"`
	Region2                string   `json:"originRegion2,omitempty"`
	ManufacturingLocations []string `json:"manufacturingLocations,omitempty"`
}
type LocalSize struct {
	Images  []string `json:"goodsSizeImage,omitempty"`
	GroupID *int64   `json:"groupId,omitempty"`
}
type LocalSKU struct {
	Images         []string                  `json:"images"`
	Price          LocalSKUPrice             `json:"price"`
	Quantity       *int64                    `json:"quantity"`
	ExternalSKUID  string                    `json:"externalSkuId,omitempty"`
	PackageInfo    LocalPackageInfo          `json:"packageInfo"`
	Specifications []LocalSpecificationValue `json:"specDetails"`
	BarCodeType    *int64                    `json:"barCodeType,omitempty"`
	BarCodeID      string                    `json:"barCodeId,omitempty"`
	BarCodeIDs     []string                  `json:"barCodeIdList,omitempty"`
	ReferenceLink  string                    `json:"referenceLink,omitempty"`
	ReferencePrice *LocalMoney               `json:"referencePrice,omitempty"`
	References     *LocalReferences          `json:"references,omitempty"`
}
type LocalSKUPrice struct {
	BasePrice LocalMoney  `json:"basePrice"`
	ListPrice *LocalMoney `json:"listPrice,omitempty"`
}
type LocalMoney struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}
type LocalPackageInfo struct {
	Weight string `json:"weight"`
	Length string `json:"length"`
	Width  string `json:"width"`
	Height string `json:"height"`
}
type LocalSpecificationValue struct {
	ParentSpecID *int64 `json:"parentSpecId,omitempty"`
	SpecID       *int64 `json:"specId,omitempty"`
	Name         string `json:"specName,omitempty"`
}
type LocalReferences struct {
	Code *LocalReferenceCode `json:"code,omitempty"`
	Note []string            `json:"note,omitempty"`
}
type LocalReferenceCode struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}
type LocalVehicleRelation struct {
	KTypes []int64 `json:"ktype,omitempty"`
}
type LocalReferenceGoods struct {
	Channel  string `json:"referenceChannel,omitempty"`
	MallLink string `json:"referenceMallLink,omitempty"`
}
type LocalCertification struct {
	Properties         []LocalGovernanceProperty `json:"governProperty,omitempty"`
	ResponsiblePersons []int64                   `json:"responsiblePerson,omitempty"`
	ResponsibleNames   []string                  `json:"respNameList,omitempty"`
}
type LocalGovernanceProperty struct {
	TemplateID int64  `json:"templateId"`
	RefPID     int64  `json:"refPid"`
	VID        *int64 `json:"vid,omitempty"`
	Value      string `json:"value,omitempty"`
	InputValue string `json:"inputValue,omitempty"`
}

// Creation is a distinct fact from review or on-sale status. Missing identifiers
// remain observable, so the Submission owner cannot infer success from HTTP 200.
type LocalGoodsCreated struct {
	GoodsID     *int64             `json:"goodsId"`
	ProductType *int64             `json:"productType"`
	SKUs        []LocalCreatedSKU  `json:"skuInfoList"`
	Warnings    []LocalRuleWarning `json:"warnings"`
}
type LocalCreatedSKU struct {
	SKUID          *int64                    `json:"skuId"`
	ExternalSKUSN  string                    `json:"outSkuSn"`
	Specifications []LocalSpecificationValue `json:"specList"`
}
