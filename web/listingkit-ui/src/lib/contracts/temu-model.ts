import { z } from "zod";

// Fixed typed TEMU protocol models on the browser boundary. Native integer
// values are decimal strings here and numeric values only at the official API.
const temuText=z.string().max(2*1024*1024).refine(v=>!v.includes("\0"));
export const temuIntegerSchema=z.string().max(20).regex(/^(0|-?[1-9][0-9]*)$/).refine(v=>{try{const n=BigInt(v);return n>=BigInt("-9223372036854775808")&&n<=BigInt("9223372036854775807")}catch{return false}});
export const temuModeSchema=z.enum(["local","fully_managed","semi_managed"]);
const supplierCategorySchema=z.object({
 catId:temuIntegerSchema,
 catName:temuText,
 parentCatId:temuIntegerSchema.nullable(),
 catType:temuIntegerSchema.nullable(),
 isLeaf:z.boolean().nullable(),
 catLevel:temuIntegerSchema.nullable(),
 isHidden:z.boolean().nullable(),
 hiddenType:temuIntegerSchema.nullable()
}).strict();
const localCategorySchema=z.object({
 catId:temuIntegerSchema,
 catName:temuText,
 parentId:temuIntegerSchema.nullable(),
 catType:temuIntegerSchema.nullable(),
 leaf:z.boolean().nullable(),
 level:temuIntegerSchema.nullable(),
 availableStatus:temuIntegerSchema.nullable(),
 expandCatType:temuIntegerSchema.nullable(),
 secondHandCategory:z.boolean().nullable()
}).strict();
const localAttributeValueRulesSchema=z.object({
 minValue:temuText.nullable(),
 maxValue:temuText.nullable(),
 valuePrecision:temuIntegerSchema.nullable(),
 valueRule:temuIntegerSchema.nullable(),
 propertyChooseTitle:temuText,
 numberInputTitle:temuText
}).strict();
const localAttributeRulesSchema=z.object({
 controlType:temuIntegerSchema.nullable(),
 chooseMaxNum:temuIntegerSchema.nullable(),
 transnationalAttribute:z.boolean().nullable(),
 attributeValueRules:localAttributeValueRulesSchema.nullable()
}).strict();
const localChildAttributeSchema=z.object({
 childAttributeId:temuIntegerSchema.nullable(),
 childAttributeValueGroupId:temuText
}).strict();
const localAttributeValueSchema=z.object({
 vid:temuIntegerSchema.nullable(),
 value:temuText,
 childAttribute:z.array(localChildAttributeSchema).max(10000).nullable()
}).strict();
const localAttributeValueGroupSchema=z.object({
 groupId:temuText,
 attributeValueList:z.array(localAttributeValueSchema).max(10000).nullable()
}).strict();
const localAttributeUnitSchema=z.object({
 valueUnitName:temuText,
 valueUnitId:temuIntegerSchema.nullable()
}).strict();
const localAttributeSchema=z.object({
 refPid:temuIntegerSchema.nullable(),
 attributeName:temuText,
 required:z.boolean().nullable(),
 attributeRules:localAttributeRulesSchema.nullable(),
 attributeValueDetail:z.array(localAttributeValueGroupSchema).max(10000).nullable(),
 attributeValueUnitList:z.array(localAttributeUnitSchema).max(10000).nullable()
}).strict();
const localAttributeTemplateSchema=z.object({
 catId:temuIntegerSchema.nullable(),
 language:temuText,
 attributeList:z.array(localAttributeSchema).max(10000).nullable()
}).strict();
const localFreightTemplateSchema=z.object({
 templateName:temuText,
 templateId:temuText
}).strict();
const localTrademarkSchema=z.object({
 brandId:temuIntegerSchema.nullable(),
 brandName:temuText,
 trademarkId:temuIntegerSchema.nullable(),
 trademarkName:temuText
}).strict();
const localTrademarkPageSchema=z.object({
 pageNo:temuIntegerSchema.nullable(),
 totalNum:temuIntegerSchema.nullable(),
 trademarkList:z.array(localTrademarkSchema).max(10000).nullable()
}).strict();
const localVariationValueSchema=z.object({
 specId:temuIntegerSchema.nullable(),
 specName:temuText,
 groupId:temuIntegerSchema.nullable(),
 groupName:temuText,
 subGroupId:temuIntegerSchema.nullable(),
 subGroupName:temuText
}).strict();
const localVariationSchema=z.object({
 parentSpecId:temuIntegerSchema.nullable(),
 parentSpecName:temuText,
 required:z.boolean().nullable(),
 variationType:temuIntegerSchema.nullable(),
 specList:z.array(localVariationValueSchema).max(10000).nullable()
}).strict();
const localVariationTemplateSchema=z.object({
 catId:temuIntegerSchema.nullable(),
 language:temuText,
 variations:z.array(localVariationSchema).max(10000).nullable()
}).strict();
const localFieldRuleSchema=z.object({
 filedName:temuText,
 requirementType:temuIntegerSchema.nullable()
}).strict();
const localSecondHandLevelSchema=z.object({
 number:temuIntegerSchema.nullable(),
 name:temuText
}).strict();
const localSecondHandAgencySchema=z.object({
 insName:temuText,
 grade:z.array(temuText).max(10000).nullable()
}).strict();
const localSecondHandConditionSchema=z.object({
 type:temuIntegerSchema.nullable(),
 level:z.array(localSecondHandLevelSchema).max(10000).nullable(),
 agency:z.array(localSecondHandAgencySchema).max(10000).nullable()
}).strict();
const localSecondHandScopeSchema=z.object({
 businessScope:temuIntegerSchema.nullable(),
 condition:z.array(localSecondHandConditionSchema).max(10000).nullable()
}).strict();
const localRuleWarningSchema=z.object({
 message:temuText
}).strict();
const localCategoryRulesSchema=z.object({
 fieldRules:z.array(localFieldRuleSchema).max(10000).nullable(),
 expandCatType:temuIntegerSchema.nullable(),
 secondHand:z.array(localSecondHandScopeSchema).max(10000).nullable(),
 warnings:z.array(localRuleWarningSchema).max(10000).nullable()
}).strict();
const supplierPropertyGroupSchema=z.object({
 id:temuIntegerSchema.nullable(),
 name:temuText
}).strict();
const supplierPropertyValueSchema=z.object({
 vid:temuIntegerSchema.nullable(),
 specId:temuIntegerSchema.nullable(),
 value:temuText,
 lang2Value:z.record(z.string(),temuText).nullable(),
 parentVidList:z.array(temuIntegerSchema).max(10000).nullable(),
 extendInfo:temuText
}).strict();
const supplierPropertyValueParentsSchema=z.object({
 vidList:z.array(temuIntegerSchema).max(10000).nullable(),
 parentVidList:z.array(temuIntegerSchema).max(10000).nullable()
}).strict();
const supplierPropertyShowConditionSchema=z.object({
 parentRefPid:temuIntegerSchema.nullable(),
 parentVids:z.array(temuIntegerSchema).max(10000).nullable()
}).strict();
const supplierPropertySchema=z.object({
 refPid:temuIntegerSchema.nullable(),
 pid:temuIntegerSchema.nullable(),
 templatePid:temuIntegerSchema.nullable(),
 parentTemplatePid:temuIntegerSchema.nullable(),
 parentSpecId:temuIntegerSchema.nullable(),
 name:temuText,
 lang2Name:z.record(z.string(),temuText).nullable(),
 required:z.boolean().nullable(),
 isSale:z.boolean().nullable(),
 mainSale:z.boolean().nullable(),
 controlType:temuIntegerSchema.nullable(),
 referenceType:temuIntegerSchema.nullable(),
 propertyValueType:temuIntegerSchema.nullable(),
 valueRule:temuIntegerSchema.nullable(),
 feature:temuIntegerSchema.nullable(),
 showType:temuIntegerSchema.nullable(),
 inputMaxNum:temuIntegerSchema.nullable(),
 chooseMaxNum:temuIntegerSchema.nullable(),
 valuePrecision:temuIntegerSchema.nullable(),
 minValue:temuText.nullable(),
 maxValue:temuText.nullable(),
 valueUnit:z.array(temuText).max(10000).nullable(),
 numberInputTitle:temuText,
 propertyChooseTitle:temuText,
 group:supplierPropertyGroupSchema.nullable(),
 values:z.array(supplierPropertyValueSchema).max(10000).nullable(),
 templatePropertyValueParentList:z.array(supplierPropertyValueParentsSchema).max(10000).nullable(),
 showCondition:z.array(supplierPropertyShowConditionSchema).max(10000).nullable()
}).strict();
const supplierAttributeTemplateSchema=z.object({
 inputMaxSpecNum:temuIntegerSchema.nullable(),
 chooseAllQualifySpec:z.boolean().nullable(),
 singleSpecValueNum:temuIntegerSchema.nullable(),
 properties:z.array(supplierPropertySchema).max(10000).nullable()
}).strict();
const localBrandSchema=z.object({
 trademarkId:temuIntegerSchema.nullable().optional(),
 noTrademark:z.boolean().nullable().optional()
}).strict();
const localGallerySchema=z.object({
 detailVideoId:temuText.optional(),
 detailImage:z.array(temuText).max(10000).nullable().optional(),
 carouselVideoId:temuText.optional(),
 goodsCarouselImage:z.array(temuText).max(10000).nullable().optional()
}).strict();
const localSecondHandSchema=z.object({
 businessScope:temuIntegerSchema.nullable().optional(),
 level:temuIntegerSchema.nullable().optional(),
 insName:temuText.optional(),
 grade:temuText.optional()
}).strict();
const localGoodsBasicSchema=z.object({
 goodsName:temuText,
 catId:temuIntegerSchema,
 goodsDesc:temuText.optional(),
 brand:localBrandSchema.nullable().optional(),
 bulletPoints:z.array(temuText).max(10000).nullable().optional(),
 goodsGallery:localGallerySchema.nullable().optional(),
 externalGoodsId:temuText.optional(),
 productType:temuIntegerSchema.nullable().optional(),
 itemTaxCode:temuText.optional(),
 secondHand:localSecondHandSchema.nullable().optional()
}).strict();
const localServicePromiseSchema=z.object({
 shipmentLimitDay:temuIntegerSchema.nullable(),
 fulfillmentType:temuIntegerSchema.nullable().optional(),
 costTemplateId:temuText
}).strict();
const localPropertySchema=z.object({
 refPid:temuIntegerSchema,
 vid:temuIntegerSchema.nullable().optional(),
 value:temuText.optional(),
 numberInputValue:temuText.optional(),
 valueUnitId:temuIntegerSchema.nullable().optional()
}).strict();
const localOriginSchema=z.object({
 originRegion1:temuText.optional(),
 originRegion2:temuText.optional(),
 manufacturingLocations:z.array(temuText).max(10000).nullable().optional()
}).strict();
const localSizeSchema=z.object({
 goodsSizeImage:z.array(temuText).max(10000).nullable().optional(),
 groupId:temuIntegerSchema.nullable().optional()
}).strict();
const localMoneySchema=z.object({
 amount:temuText,
 currency:temuText
}).strict();
const localSKUPriceSchema=z.object({
 basePrice:localMoneySchema,
 listPrice:localMoneySchema.nullable().optional()
}).strict();
const localPackageInfoSchema=z.object({
 weight:temuText,
 length:temuText,
 width:temuText,
 height:temuText
}).strict();
const localSpecificationValueSchema=z.object({
 parentSpecId:temuIntegerSchema.nullable().optional(),
 specId:temuIntegerSchema.nullable().optional(),
 specName:temuText.optional()
}).strict();
const localReferenceCodeSchema=z.object({
 name:temuText.optional(),
 id:temuText.optional()
}).strict();
const localReferencesSchema=z.object({
 code:localReferenceCodeSchema.nullable().optional(),
 note:z.array(temuText).max(10000).nullable().optional()
}).strict();
const localSKUSchema=z.object({
 images:z.array(temuText).max(10000).nullable(),
 price:localSKUPriceSchema,
 quantity:temuIntegerSchema.nullable(),
 externalSkuId:temuText.optional(),
 packageInfo:localPackageInfoSchema,
 specDetails:z.array(localSpecificationValueSchema).max(10000).nullable(),
 barCodeType:temuIntegerSchema.nullable().optional(),
 barCodeId:temuText.optional(),
 barCodeIdList:z.array(temuText).max(10000).nullable().optional(),
 referenceLink:temuText.optional(),
 referencePrice:localMoneySchema.nullable().optional(),
 references:localReferencesSchema.nullable().optional()
}).strict();
const localVehicleRelationSchema=z.object({
 ktype:z.array(temuIntegerSchema).max(10000).nullable().optional()
}).strict();
const localReferenceGoodsSchema=z.object({
 referenceChannel:temuText.optional(),
 referenceMallLink:temuText.optional()
}).strict();
const localGovernancePropertySchema=z.object({
 templateId:temuIntegerSchema,
 refPid:temuIntegerSchema,
 vid:temuIntegerSchema.nullable().optional(),
 value:temuText.optional(),
 inputValue:temuText.optional()
}).strict();
const localCertificationSchema=z.object({
 governProperty:z.array(localGovernancePropertySchema).max(10000).nullable().optional(),
 responsiblePerson:z.array(temuIntegerSchema).max(10000).nullable().optional(),
 respNameList:z.array(temuText).max(10000).nullable().optional()
}).strict();
const localGoodsV2Schema=z.object({
 language:temuText.optional(),
 goodsBasic:localGoodsBasicSchema,
 goodsServicePromise:localServicePromiseSchema,
 goodsProperty:z.array(localPropertySchema).max(10000).nullable().optional(),
 goodsOriginInfo:localOriginSchema.nullable().optional(),
 goodsSize:localSizeSchema.nullable().optional(),
 skuList:z.array(localSKUSchema).max(10000).nullable(),
 goodsVehiclePropertyRelation:localVehicleRelationSchema.nullable().optional(),
 referenceGoods:localReferenceGoodsSchema.nullable().optional(),
 certificationInfo:localCertificationSchema.nullable().optional()
}).strict();
export const ruleSelectionSchema=z.object({
 parentCatId:temuIntegerSchema,
 catId:temuIntegerSchema,
 language:temuText.optional(),
 costTemplateId:temuText.optional(),
 supplierSiteId:temuIntegerSchema.optional(),
 supplierSizeOffset:temuIntegerSchema.optional(),
	 supplierCategoryPath:z.array(temuIntegerSchema).length(10).optional(),
 trademarkPage:temuIntegerSchema.optional(),
 trademarkSize:temuIntegerSchema.optional()
}).strict();
const supplierSemiManagedSchema=z.object({
 semiLanguageStrategy:temuIntegerSchema.nullable().optional(),
 bindSiteIds:z.array(temuIntegerSchema).max(10000).nullable(),
 semiManagedSiteMode:temuIntegerSchema.nullable().optional()
}).strict();
const supplierNameSchema=z.object({
 language:temuText,
 productName:temuText
}).strict();
const supplierBasicPropertySchema=z.object({
 vid:temuIntegerSchema.nullable(),
 valueUnit:temuText,
 pid:temuIntegerSchema.nullable(),
 templatePid:temuIntegerSchema.nullable(),
 numberInputValue:temuText.optional(),
 propValue:temuText,
 propName:temuText,
 refPid:temuIntegerSchema.nullable()
}).strict();
const supplierSpecPropertySchema=supplierBasicPropertySchema.extend({
 specId:temuIntegerSchema.nullable(),
 parentSpecId:temuIntegerSchema.nullable(),
 valueGroupId:temuIntegerSchema.nullable(),
 valueGroupName:temuText
}).strict();
const supplierImageSchema=z.object({
 imageUrl:temuText
}).strict();
const supplierGuideSchema=z.object({
 fileName:temuText,
 pdfMaterialId:temuIntegerSchema.nullable(),
 languages:z.array(temuText).max(10000).nullable()
}).strict();
const supplierShipmentSchema=z.object({
 freightTemplateId:temuText,
 shipmentLimitSecond:temuIntegerSchema.nullable()
}).strict();
const supplierWarehouseSitesSchema=z.object({
 siteIdList:z.array(temuIntegerSchema).max(10000).nullable(),
 warehouseId:temuText
}).strict();
const supplierWarehouseRouteSchema=z.object({
 targetRouteList:z.array(supplierWarehouseSitesSchema).max(10000).nullable(),
 currentRouteList:z.array(supplierWarehouseSitesSchema).max(10000).nullable().optional()
}).strict();
const supplierFileSchema=z.object({
 fileName:temuText,
 fileUrl:temuText
}).strict();
const supplierOriginSchema=z.object({
 region1ShortName:temuText,
 region2Id:temuIntegerSchema.nullable().optional()
}).strict();
const supplierWarehouseFactsSchema=z.object({
 productOriginCertFiles:z.array(supplierFileSchema).max(10000).nullable().optional(),
 outerGoodsUrl:temuText,
 productOrigin:supplierOriginSchema
}).strict();
const supplierSitePriceSchema=z.object({
 siteId:temuIntegerSchema.nullable(),
 supplierPrice:temuIntegerSchema.nullable()
}).strict();
const supplierWarehouseStockSchema=z.object({
 targetStockAvailable:temuIntegerSchema.nullable(),
 warehouseId:temuText,
 currentStockAvailable:temuIntegerSchema.nullable().optional()
}).strict();
const supplierStockSchema=z.object({
 warehouseStockQuantityReqs:z.array(supplierWarehouseStockSchema).max(10000).nullable()
}).strict();
const supplierWeightSchema=z.object({
 inputUnit:temuText.optional(),
 inputValue:temuText.optional(),
 value:temuIntegerSchema.nullable()
}).strict();
const supplierVolumeSchema=z.object({
 inputUnit:temuText.optional(),
 len:temuIntegerSchema.nullable(),
 width:temuIntegerSchema.nullable(),
 height:temuIntegerSchema.nullable(),
 inputLen:temuText.optional(),
 inputWidth:temuText.optional(),
 inputHeight:temuText.optional()
}).strict();
const supplierKnifeTipSchema=z.object({
 degrees:temuIntegerSchema.nullable()
}).strict();
const supplierSensitiveLimitsSchema=z.object({
 maxBatteryCapacityHp:temuIntegerSchema.nullable().optional(),
 maxBatteryCapacity:temuIntegerSchema.nullable().optional(),
 maxLiquidCapacity:temuIntegerSchema.nullable().optional(),
 maxLiquidCapacityHp:temuIntegerSchema.nullable().optional(),
 maxKnifeLength:temuIntegerSchema.nullable().optional(),
 maxKnifeLengthHp:temuIntegerSchema.nullable().optional(),
 knifeTipAngle:supplierKnifeTipSchema.nullable().optional()
}).strict();
const supplierSensitiveSchema=z.object({
 isSensitive:temuIntegerSchema.nullable().optional(),
 sensitiveTypes:z.array(temuIntegerSchema).max(10000).nullable().optional(),
 sensitiveList:z.array(temuIntegerSchema).max(10000).nullable().optional()
}).strict();
const supplierSKUPackagingSchema=z.object({
 productSkuWeightReq:supplierWeightSchema.nullable(),
 productSkuVolumeReq:supplierVolumeSchema.nullable(),
 productSkuSensitiveLimitReq:supplierSensitiveLimitsSchema.nullable(),
 productSkuSensitiveAttrReq:supplierSensitiveSchema.nullable(),
 productSkuNetWeightReq:supplierWeightSchema.nullable().optional()
}).strict();
const supplierSpecificationSchema=z.object({
 specId:temuIntegerSchema.nullable(),
 parentSpecId:temuIntegerSchema.nullable(),
 parentSpecName:temuText,
 specName:temuText
}).strict();
const supplierMultiPackSchema=z.object({
 numberOfPieces:temuIntegerSchema.nullable().optional(),
 individuallyPacked:temuIntegerSchema.nullable().optional()
}).strict();
const supplierNetContentSumSchema=z.object({
 netContentUnitCode:temuIntegerSchema.nullable().optional(),
 netContentNumber:temuIntegerSchema.nullable().optional()
}).strict();
const supplierNetContentSchema=z.object({
 netContentUnitCode:temuIntegerSchema.nullable().optional(),
 netContentNumber:temuIntegerSchema.nullable().optional(),
 mixedType:temuIntegerSchema.nullable().optional(),
 totalNetContent:supplierNetContentSumSchema.nullable().optional(),
 pieceNewUnitCode:temuIntegerSchema.nullable().optional(),
 skuClassification:temuIntegerSchema.nullable().optional(),
 numberOfPiecesNew:temuIntegerSchema.nullable().optional(),
 pieceUnitCode:temuIntegerSchema.nullable().optional()
}).strict();
const supplierBarCodeSchema=z.object({
 code:temuText.optional(),
 codeType:temuIntegerSchema.nullable().optional()
}).strict();
const supplierSKUSchema=z.object({
 currencyType:temuText,
 supplierPrice:temuIntegerSchema.nullable().optional(),
 siteSupplierPrices:z.array(supplierSitePriceSchema).max(10000).nullable().optional(),
 productSkuStockQuantityReq:supplierStockSchema.nullable().optional(),
 extCode:temuText,
 thumbUrl:temuText,
 productSkuWhExtAttrReq:supplierSKUPackagingSchema.nullable(),
 productSkuSpecReqs:z.array(supplierSpecificationSchema).max(10000).nullable(),
 productSkuMultiPackReq:supplierMultiPackSchema.nullable().optional(),
 productSkuNetContentReq:supplierNetContentSchema.nullable().optional(),
 productSkuBarCodeReqs:z.array(supplierBarCodeSchema).max(10000).nullable().optional()
}).strict();
const supplierSKCSchema=z.object({
 extCode:temuText,
 productSkuReqs:z.array(supplierSKUSchema).max(10000).nullable(),
 mainProductSkuSpecReqs:z.array(supplierSpecificationSchema).max(10000).nullable(),
 previewImgUrls:z.array(temuText).max(10000).nullable(),
 colorImageUrl:temuText.optional()
}).strict();
const supplierSecondHandSchema=z.object({
 isSecondHand:z.boolean().nullable().optional(),
 secondHandLevel:temuIntegerSchema.nullable().optional()
}).strict();
const supplierSaleFactsSchema=z.object({
 inventoryRegion:temuIntegerSchema.nullable().optional(),
 productSecondHandReq:supplierSecondHandSchema.nullable().optional(),
 discreetShipping:z.boolean().nullable().optional()
}).strict();
const supplierManufacturingSchema=z.object({
 region1ShortNames:z.array(temuText).max(10000).nullable().optional()
}).strict();
const supplierComplianceSchema=z.object({
 protocolVersion:temuText,
 protocolUrl:temuText
}).strict();
const supplierTextStyleSchema=z.object({
 backgroundColor:temuText,
 fontFamily:temuIntegerSchema.nullable().optional(),
 fontSize:temuIntegerSchema.nullable(),
 align:temuText,
 fontColor:temuText
}).strict();
const supplierDetailContentSchema=z.object({
 imgUrl:temuText.optional(),
 text:temuText.optional(),
 width:temuIntegerSchema.nullable().optional(),
 height:temuIntegerSchema.nullable().optional(),
 textModuleDetails:supplierTextStyleSchema.nullable().optional()
}).strict();
const supplierDetailDecorationSchema=z.object({
 lang:temuText,
 type:temuText,
 priority:temuIntegerSchema.nullable(),
 contentList:z.array(supplierDetailContentSchema).max(10000).nullable(),
 key:temuText
}).strict();
const supplierGoodsSchema=z.object({
 productSemiManagedReq:supplierSemiManagedSchema.nullable().optional(),
 productName:temuText,
 productI18nReqs:z.array(supplierNameSchema).max(10000).nullable().optional(),
 cat1Id:temuIntegerSchema.nullable(),
 cat2Id:temuIntegerSchema.nullable(),
 cat3Id:temuIntegerSchema.nullable(),
 cat4Id:temuIntegerSchema.nullable(),
 cat5Id:temuIntegerSchema.nullable(),
 cat6Id:temuIntegerSchema.nullable(),
 cat7Id:temuIntegerSchema.nullable(),
 cat8Id:temuIntegerSchema.nullable(),
 cat9Id:temuIntegerSchema.nullable(),
 cat10Id:temuIntegerSchema.nullable(),
 productSpecPropertyReqs:z.array(supplierSpecPropertySchema).max(10000).nullable(),
 productPropertyReqs:z.array(supplierBasicPropertySchema).max(10000).nullable(),
 carouselImageUrls:z.array(temuText).max(10000).nullable().optional(),
 materialImgUrl:temuText,
 productOuterPackageImageReqs:z.array(supplierImageSchema).max(10000).nullable().optional(),
 sizeTemplateIds:z.array(temuIntegerSchema).max(10000).nullable().optional(),
 showSizeTemplateIds:z.array(temuIntegerSchema).max(10000).nullable().optional(),
 productGuideFileReqs:z.array(supplierGuideSchema).max(10000).nullable().optional(),
 materialMultiLanguages:z.array(temuText).max(10000).nullable().optional(),
 productShipmentReq:supplierShipmentSchema.nullable().optional(),
 productWarehouseRouteReq:supplierWarehouseRouteSchema.nullable().optional(),
 productWhExtAttrReq:supplierWarehouseFactsSchema.nullable().optional(),
 productSkcReqs:z.array(supplierSKCSchema).max(10000).nullable(),
 productSaleExtAttrReq:supplierSaleFactsSchema.nullable().optional(),
 manufacturingLocation:supplierManufacturingSchema.nullable().optional(),
 ipCodes:z.array(temuText).max(10000).nullable().optional(),
 productComplianceStatementReq:supplierComplianceSchema.nullable().optional(),
 goodsLayerDecorationReqs:z.array(supplierDetailDecorationSchema).max(10000).nullable().optional()
}).strict();
const sourceSchema=z.object({
 evidenceId:temuText,
 url:temuText,
 updatedAt:temuText,
 contentHash:temuText,
 totalRows:temuIntegerSchema,
 includedRows:temuIntegerSchema
}).strict();
const regionSchema=z.object({
 regionId:temuIntegerSchema,
 siteId:temuIntegerSchema,
 label:temuText,
 deploymentPartition:temuText,
 semi:z.enum(["unknown","open","closed","coming_soon"]),
 local:z.enum(["unknown","open","closed","coming_soon"])
}).strict();
const numberRuleSchema=z.object({
 precision:temuIntegerSchema,
 minimum:temuText,
 maximum:temuText,
 minimumExclusive:z.boolean(),
 maximumExclusive:z.boolean(),
 rawRange:temuText
}).strict();
const unitRuleSchema=z.object({
 unit:temuText,
 number:numberRuleSchema
}).strict();
const localSpecificationSchema=z.object({
 siteId:temuIntegerSchema,
 label:temuText,
 documentPartition:temuText,
 rawLanguages:temuText,
 languages:z.array(temuText).max(10000).nullable(),
 weight:unitRuleSchema,
 dimension:unitRuleSchema,
 baseCurrency:temuText,
 retailCurrency:temuText,
 basePrice:numberRuleSchema,
 retailPrice:numberRuleSchema
}).strict();
const localTargetSchema=z.object({
 applicationPartition:temuText,
 region:regionSchema,
 specification:localSpecificationSchema,
 snapshotRevision:temuText,
 issues:z.array(temuText).max(10000).nullable()
}).strict();
const semiSiteSchema=z.object({
 siteId:temuIntegerSchema,
 label:temuText,
 opening:z.enum(["unknown","open","closed","coming_soon"]),
 association:temuText,
 associatedSiteIds:z.array(temuIntegerSchema).max(10000).nullable()
}).strict();
const temuImageSlotSchema=z.object({field:z.enum(["detail","gallery","sku","size","material","outer","skc","thumb"]),skc:temuIntegerSchema,sku:temuIntegerSchema,sort:temuIntegerSchema,assetId:temuText.min(1),usage:temuIntegerSchema}).strict();
export const temuDraftInputSchema=z.object({mode:temuModeSchema,selection:ruleSelectionSchema,local:localGoodsV2Schema.optional(),supplier:supplierGoodsSchema.optional(),packageUnits:z.object({weight:temuText,dimension:temuText}).strict(),images:z.array(temuImageSlotSchema).max(10000)}).strict().refine(v=>v.mode==="local"?!!v.local&&!v.supplier:!!v.supplier&&!v.local);
const temuIssueSchema=z.object({field:temuText,code:temuText,message:temuText}).strict();
export const temuDraftSchema=z.object({mode:temuModeSchema,local:localGoodsV2Schema.optional(),supplier:supplierGoodsSchema.optional(),images:z.array(temuImageSlotSchema).max(10000).nullable(),issues:z.array(temuIssueSchema).max(10000),readyForUpload:z.boolean(),submissionPayload:z.union([localGoodsV2Schema,supplierGoodsSchema]).optional()});
const supplierSizeClassSchema=z.object({catId:temuIntegerSchema.nullable(),classId:temuIntegerSchema.nullable(),parentClassId:temuIntegerSchema.nullable(),relatedClassIds:z.array(temuIntegerSchema).max(256).nullable(),classType:temuIntegerSchema.nullable()}).strict();
const supplierSizeRequirementSchema=z.object({catId:temuIntegerSchema,observed:z.boolean(),class:supplierSizeClassSchema.optional()}).strict();
const supplierSizeSettingsSchema=z.object({classId:temuIntegerSchema,settings:z.object({code:temuIntegerSchema.nullable(),sizeList:z.array(temuText).max(10000).nullable(),mappingContent:z.object({records:z.array(z.object({values:z.record(z.string(),z.string()).nullable()}).strict()).max(10000).nullable(),meta:z.object({groupList:z.array(z.object({id:temuIntegerSchema.nullable(),name:temuText,unnecessary:z.boolean().nullable()}).strict()).max(10000).nullable()}).strict().nullable()}).strict().nullable()}).strict()}).strict();
const supplierSizePageSchema=z.object({offset:temuIntegerSchema.nullable(),pageSize:temuIntegerSchema.nullable(),totalCount:temuIntegerSchema.nullable(),sizeSpecDataList:z.array(z.object({classId:temuIntegerSchema.nullable(),businessId:temuIntegerSchema.nullable(),name:temuText,reusable:z.boolean().nullable(),updatedAt:temuIntegerSchema.nullable(),contentDTO:z.object({records:z.array(z.object({values:z.record(z.string(),z.string()).nullable()}).strict()).max(10000).nullable()}).strict().nullable()}).strict()).max(50).nullable()}).strict();
export const temuRuleSnapshotSchema=z.object({selection:ruleSelectionSchema,mode:temuModeSchema,localTarget:localTargetSchema.optional(),semiSites:z.array(semiSiteSchema).max(256).optional(),specificationRevision:temuText,specificationSources:z.array(sourceSchema).max(32),localCategories:z.array(localCategorySchema).max(10000).nullable(),supplierCategories:z.array(supplierCategorySchema).max(10000).nullable(),supplierCategoryPath:z.array(supplierCategorySchema).max(10).optional(),selectableLeaf:z.boolean(),supplierAttributes:supplierAttributeTemplateSchema.optional(),supplierSizeRequirement:supplierSizeRequirementSchema.optional(),supplierSizeCharts:supplierSizePageSchema.optional(),supplierSizeSettings:z.array(supplierSizeSettingsSchema).max(32).optional(),localFieldRules:localCategoryRulesSchema.optional(),localAttributes:localAttributeTemplateSchema.optional(),localVariations:localVariationTemplateSchema.optional(),freightTemplates:z.array(localFreightTemplateSchema).max(10000).nullable(),trademarks:localTrademarkPageSchema.optional(),observations:z.array(z.object({evidenceId:temuText,api:temuText,inputHash:temuText,decodedHash:temuText})).max(32),issues:z.array(temuText).max(10000),observedAt:z.iso.datetime({offset:true})});

export type TemuDraftInput=z.infer<typeof temuDraftInputSchema>;
export type TemuRuleSnapshot=z.infer<typeof temuRuleSnapshotSchema>;

const createdID=temuIntegerSchema.refine(v=>BigInt(v)>0);
const createdSpecification=z.object({parentSpecId:createdID,specId:createdID,specName:temuText}).strict();
export const temuCreatedGoodsSchema=z.object({
 local:z.object({goodsId:createdID,productType:createdID,skuInfoList:z.array(z.object({skuId:createdID,outSkuSn:temuText,specList:z.array(createdSpecification).min(1).max(256)}).strict()).min(1).max(500),warnings:z.array(z.object({message:temuText}).strict()).max(4096).nullable()}).strict().optional(),
 supplier:z.object({productId:createdID,productSkcList:z.array(z.object({productSkcId:createdID}).strict()).min(1).max(40),productSkuList:z.array(z.object({productSkuId:createdID,extCode:temuText,productSkcId:createdID,skuSpecList:z.array(supplierSpecificationSchema).max(256).nullable()}).strict()).min(1).max(1000)}).strict().optional(),
 responseHash:z.string().regex(/^[0-9a-f]{64}$/)
}).strict().refine(v=>!!v.local!==!!v.supplier);
export type TemuCreatedGoods=z.infer<typeof temuCreatedGoodsSchema>;
