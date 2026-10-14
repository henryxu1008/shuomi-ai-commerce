package model

// SupplierGoods is E13's single supplier-product creation body. The PA route
// belongs to the adapter; no app/token, wrapper or per-country create is here.
// Required numbers are pointers so omission cannot manufacture zero facts.
type SupplierGoods struct {
	SemiManaged       *SupplierSemiManaged       `json:"productSemiManagedReq,omitempty"`
	Name              string                     `json:"productName"`
	Names             []SupplierName             `json:"productI18nReqs,omitempty"`
	Cat1ID            *int64                     `json:"cat1Id"`
	Cat2ID            *int64                     `json:"cat2Id"`
	Cat3ID            *int64                     `json:"cat3Id"`
	Cat4ID            *int64                     `json:"cat4Id"`
	Cat5ID            *int64                     `json:"cat5Id"`
	Cat6ID            *int64                     `json:"cat6Id"`
	Cat7ID            *int64                     `json:"cat7Id"`
	Cat8ID            *int64                     `json:"cat8Id"`
	Cat9ID            *int64                     `json:"cat9Id"`
	Cat10ID           *int64                     `json:"cat10Id"`
	SpecProperties    []SupplierSpecProperty     `json:"productSpecPropertyReqs"`
	Properties        []SupplierBasicProperty    `json:"productPropertyReqs"`
	CarouselImages    []string                   `json:"carouselImageUrls,omitempty"`
	MaterialImage     string                     `json:"materialImgUrl"`
	OuterImages       []SupplierImage            `json:"productOuterPackageImageReqs,omitempty"`
	SizeTemplateIDs   []int64                    `json:"sizeTemplateIds,omitempty"`
	ShowSizeIDs       []int64                    `json:"showSizeTemplateIds,omitempty"`
	Guides            []SupplierGuide            `json:"productGuideFileReqs,omitempty"`
	MaterialLanguages []string                   `json:"materialMultiLanguages,omitempty"`
	Shipment          *SupplierShipment          `json:"productShipmentReq,omitempty"`
	WarehouseRoute    *SupplierWarehouseRoute    `json:"productWarehouseRouteReq,omitempty"`
	WarehouseFacts    *SupplierWarehouseFacts    `json:"productWhExtAttrReq,omitempty"`
	SKCs              []SupplierSKC              `json:"productSkcReqs"`
	SaleFacts         *SupplierSaleFacts         `json:"productSaleExtAttrReq,omitempty"`
	Manufacturing     *SupplierManufacturing     `json:"manufacturingLocation,omitempty"`
	IPCodes           []string                   `json:"ipCodes,omitempty"`
	Compliance        *SupplierCompliance        `json:"productComplianceStatementReq,omitempty"`
	Details           []SupplierDetailDecoration `json:"goodsLayerDecorationReqs,omitempty"`
}

type SupplierSemiManaged struct {
	LanguageStrategy *int64  `json:"semiLanguageStrategy,omitempty"`
	SiteIDs          []int64 `json:"bindSiteIds"`
	SiteMode         *int64  `json:"semiManagedSiteMode,omitempty"`
}
type SupplierName struct {
	Language string `json:"language"`
	Name     string `json:"productName"`
}
type SupplierBasicProperty struct {
	VID              *int64 `json:"vid"`
	Unit             string `json:"valueUnit"`
	PID              *int64 `json:"pid"`
	TemplatePID      *int64 `json:"templatePid"`
	NumberInputValue string `json:"numberInputValue,omitempty"`
	Value            string `json:"propValue"`
	Name             string `json:"propName"`
	RefPID           *int64 `json:"refPid"`
}
type SupplierSpecProperty struct {
	SupplierBasicProperty
	SpecID         *int64 `json:"specId"`
	ParentSpecID   *int64 `json:"parentSpecId"`
	ValueGroupID   *int64 `json:"valueGroupId"`
	ValueGroupName string `json:"valueGroupName"`
}
type SupplierShipment struct {
	FreightTemplateID string `json:"freightTemplateId"`
	LimitSecond       *int64 `json:"shipmentLimitSecond"`
}
type SupplierWarehouseRoute struct {
	Target  []SupplierWarehouseSites `json:"targetRouteList"`
	Current []SupplierWarehouseSites `json:"currentRouteList,omitempty"`
}
type SupplierWarehouseSites struct {
	SiteIDs     []int64 `json:"siteIdList"`
	WarehouseID string  `json:"warehouseId"`
}
type SupplierWarehouseFacts struct {
	OriginCertificates []SupplierFile `json:"productOriginCertFiles,omitempty"`
	ExternalGoodsURL   string         `json:"outerGoodsUrl"`
	Origin             SupplierOrigin `json:"productOrigin"`
}
type SupplierOrigin struct {
	Region1ShortName string `json:"region1ShortName"`
	Region2ID        *int64 `json:"region2Id,omitempty"`
}
type SupplierManufacturing struct {
	Region1ShortNames []string `json:"region1ShortNames,omitempty"`
}
type SupplierFile struct {
	Name string `json:"fileName"`
	URL  string `json:"fileUrl"`
}
type SupplierGuide struct {
	Name          string   `json:"fileName"`
	PDFMaterialID *int64   `json:"pdfMaterialId"`
	Languages     []string `json:"languages"`
}
type SupplierImage struct {
	URL string `json:"imageUrl"`
}
type SupplierSKC struct {
	ExternalCode string                  `json:"extCode"`
	SKUs         []SupplierSKU           `json:"productSkuReqs"`
	MainSpecs    []SupplierSpecification `json:"mainProductSkuSpecReqs"`
	PreviewURLs  []string                `json:"previewImgUrls"`
	ColorURL     string                  `json:"colorImageUrl,omitempty"`
}
type SupplierSKU struct {
	Currency      string                  `json:"currencyType"`
	SupplierPrice *int64                  `json:"supplierPrice,omitempty"`
	SitePrices    []SupplierSitePrice     `json:"siteSupplierPrices,omitempty"`
	Stock         *SupplierStock          `json:"productSkuStockQuantityReq,omitempty"`
	ExternalCode  string                  `json:"extCode"`
	ThumbURL      string                  `json:"thumbUrl"`
	Packaging     *SupplierSKUPackaging   `json:"productSkuWhExtAttrReq"`
	Specs         []SupplierSpecification `json:"productSkuSpecReqs"`
	MultiPack     *SupplierMultiPack      `json:"productSkuMultiPackReq,omitempty"`
	NetContent    *SupplierNetContent     `json:"productSkuNetContentReq,omitempty"`
	BarCodes      []SupplierBarCode       `json:"productSkuBarCodeReqs,omitempty"`
}

// E13 prices are already in fen or cents; never marshal a float amount or
// silently convert the local mall's currency into this supplier contract.
type SupplierSitePrice struct {
	SiteID *int64 `json:"siteId"`
	Price  *int64 `json:"supplierPrice"`
}
type SupplierStock struct {
	Warehouses []SupplierWarehouseStock `json:"warehouseStockQuantityReqs"`
}
type SupplierWarehouseStock struct {
	Target      *int64 `json:"targetStockAvailable"`
	WarehouseID string `json:"warehouseId"`
	Current     *int64 `json:"currentStockAvailable,omitempty"`
}
type SupplierSpecification struct {
	ID         *int64 `json:"specId"`
	ParentID   *int64 `json:"parentSpecId"`
	ParentName string `json:"parentSpecName"`
	Name       string `json:"specName"`
}
type SupplierSKUPackaging struct {
	Weight          *SupplierWeight          `json:"productSkuWeightReq"`
	Volume          *SupplierVolume          `json:"productSkuVolumeReq"`
	SensitiveLimits *SupplierSensitiveLimits `json:"productSkuSensitiveLimitReq"`
	Sensitive       *SupplierSensitive       `json:"productSkuSensitiveAttrReq"`
	NetWeight       *SupplierWeight          `json:"productSkuNetWeightReq,omitempty"`
}
type SupplierWeight struct {
	InputUnit  string `json:"inputUnit,omitempty"`
	InputValue string `json:"inputValue,omitempty"`
	ValueMG    *int64 `json:"value"`
}

// Keep E13's exact len/width/height. Its len description says "Shortest"
// while inputLen says "Longest"; no local sorting or inferred conversion.
type SupplierVolume struct {
	InputUnit string `json:"inputUnit,omitempty"`
	LengthMM  *int64 `json:"len"`
	WidthMM   *int64 `json:"width"`
	HeightMM  *int64 `json:"height"`
	InputLen  string `json:"inputLen,omitempty"`
	InputWide string `json:"inputWidth,omitempty"`
	InputHigh string `json:"inputHeight,omitempty"`
}
type SupplierSensitive struct {
	IsSensitive *int64  `json:"isSensitive,omitempty"`
	Types       []int64 `json:"sensitiveTypes,omitempty"`
	List        []int64 `json:"sensitiveList,omitempty"`
}
type SupplierSensitiveLimits struct {
	BatteryMWh *int64            `json:"maxBatteryCapacityHp,omitempty"`
	BatteryWh  *int64            `json:"maxBatteryCapacity,omitempty"`
	LiquidML   *int64            `json:"maxLiquidCapacity,omitempty"`
	LiquidUL   *int64            `json:"maxLiquidCapacityHp,omitempty"`
	KnifeMM    *int64            `json:"maxKnifeLength,omitempty"`
	KnifeUM    *int64            `json:"maxKnifeLengthHp,omitempty"`
	KnifeAngle *SupplierKnifeTip `json:"knifeTipAngle,omitempty"`
}
type SupplierKnifeTip struct {
	Degrees *int64 `json:"degrees"`
}
type SupplierMultiPack struct {
	Pieces             *int64 `json:"numberOfPieces,omitempty"`
	IndividuallyPacked *int64 `json:"individuallyPacked,omitempty"`
}
type SupplierNetContent struct {
	UnitCode       *int64                 `json:"netContentUnitCode,omitempty"`
	Number         *int64                 `json:"netContentNumber,omitempty"`
	MixedType      *int64                 `json:"mixedType,omitempty"`
	Total          *SupplierNetContentSum `json:"totalNetContent,omitempty"`
	PieceNewUnit   *int64                 `json:"pieceNewUnitCode,omitempty"`
	Classification *int64                 `json:"skuClassification,omitempty"`
	PiecesNew      *int64                 `json:"numberOfPiecesNew,omitempty"`
	PieceUnit      *int64                 `json:"pieceUnitCode,omitempty"`
}
type SupplierNetContentSum struct {
	UnitCode *int64 `json:"netContentUnitCode,omitempty"`
	Number   *int64 `json:"netContentNumber,omitempty"`
}
type SupplierBarCode struct {
	Code string `json:"code,omitempty"`
	Type *int64 `json:"codeType,omitempty"`
}
type SupplierSaleFacts struct {
	InventoryRegion  *int64              `json:"inventoryRegion,omitempty"`
	SecondHand       *SupplierSecondHand `json:"productSecondHandReq,omitempty"`
	DiscreetShipping *bool               `json:"discreetShipping,omitempty"`
}
type SupplierSecondHand struct {
	IsSecondHand *bool  `json:"isSecondHand,omitempty"`
	Level        *int64 `json:"secondHandLevel,omitempty"`
}

// These are explicit previously approved legal facts, never an acceptance API.
type SupplierCompliance struct {
	Version string `json:"protocolVersion"`
	URL     string `json:"protocolUrl"`
}
type SupplierDetailDecoration struct {
	Language string                  `json:"lang"`
	Type     string                  `json:"type"`
	Priority *int64                  `json:"priority"`
	Content  []SupplierDetailContent `json:"contentList"`
	Key      string                  `json:"key"`
}
type SupplierDetailContent struct {
	ImageURL string             `json:"imgUrl,omitempty"`
	Text     string             `json:"text,omitempty"`
	Width    *int64             `json:"width,omitempty"`
	Height   *int64             `json:"height,omitempty"`
	Style    *SupplierTextStyle `json:"textModuleDetails,omitempty"`
}
type SupplierTextStyle struct {
	Background string `json:"backgroundColor"`
	Font       *int64 `json:"fontFamily,omitempty"`
	Size       *int64 `json:"fontSize"`
	Align      string `json:"align"`
	Color      string `json:"fontColor"`
}

// Creation IDs are separate from review/on-sale facts. Missing IDs stay unknown.
type SupplierGoodsCreated struct {
	ProductID *int64               `json:"productId"`
	SKCs      []SupplierCreatedSKC `json:"productSkcList"`
	SKUs      []SupplierCreatedSKU `json:"productSkuList"`
}
type SupplierCreatedSKC struct {
	ID *int64 `json:"productSkcId"`
}
type SupplierCreatedSKU struct {
	ID           *int64                  `json:"productSkuId"`
	ExternalCode string                  `json:"extCode"`
	SKCID        *int64                  `json:"productSkcId"`
	Specs        []SupplierSpecification `json:"skuSpecList"`
}
