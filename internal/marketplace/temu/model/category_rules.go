package model

type CategoryRuleQuery struct {
	CatID    int64
	Language string
}

// E36 varies required attributes with the original mall's shipping template.
// A snapshot without that exact selection cannot prove cross-border completion.
type LocalAttributeQuery struct {
	CatID          int64
	Language       string
	CostTemplateID string
}
type LocalAttributeTemplate struct {
	CatID      *int64           `json:"catId"`
	Language   string           `json:"language"`
	Attributes []LocalAttribute `json:"attributeList"`
}
type LocalAttribute struct {
	RefPID      *int64                     `json:"refPid"`
	Name        string                     `json:"attributeName"`
	Required    *bool                      `json:"required"`
	Rules       *LocalAttributeRules       `json:"attributeRules"`
	ValueGroups []LocalAttributeValueGroup `json:"attributeValueDetail"`
	Units       []LocalAttributeUnit       `json:"attributeValueUnitList"`
}
type LocalAttributeRules struct {
	ControlType   *int64                    `json:"controlType"`
	ChooseMaxNum  *int64                    `json:"chooseMaxNum"`
	Transnational *bool                     `json:"transnationalAttribute"`
	Values        *LocalAttributeValueRules `json:"attributeValueRules"`
}
type LocalAttributeValueRules struct {
	Minimum     *string `json:"minValue"`
	Maximum     *string `json:"maxValue"`
	Precision   *int64  `json:"valuePrecision"`
	ValueRule   *int64  `json:"valueRule"`
	ChooseTitle string  `json:"propertyChooseTitle"`
	NumberTitle string  `json:"numberInputTitle"`
}
type LocalAttributeValueGroup struct {
	ID     string                `json:"groupId"`
	Values []LocalAttributeValue `json:"attributeValueList"`
}
type LocalAttributeValue struct {
	VID             *int64                `json:"vid"`
	Value           string                `json:"value"`
	ChildAttributes []LocalChildAttribute `json:"childAttribute"`
}
type LocalChildAttribute struct {
	RefPID       *int64 `json:"childAttributeId"`
	ValueGroupID string `json:"childAttributeValueGroupId"`
}
type LocalAttributeUnit struct {
	Name string `json:"valueUnitName"`
	ID   *int64 `json:"valueUnitId"`
}

type LocalFreightTemplate struct {
	Name string `json:"templateName"`
	ID   string `json:"templateId"`
}
type LocalTrademarkQuery struct {
	Page int64
	Size int64
}

// E38 is one bounded page of current mall metadata. It exposes no license
// status, global eligibility or completeness assertion for unqueried pages.
type LocalTrademarkPage struct {
	Page       *int64           `json:"pageNo"`
	Total      *int64           `json:"totalNum"`
	Trademarks []LocalTrademark `json:"trademarkList"`
}
type LocalTrademark struct {
	BrandID       *int64 `json:"brandId"`
	BrandName     string `json:"brandName"`
	TrademarkID   *int64 `json:"trademarkId"`
	TrademarkName string `json:"trademarkName"`
}

type LocalVariationTemplate struct {
	CatID      *int64           `json:"catId"`
	Language   string           `json:"language"`
	Variations []LocalVariation `json:"variations"`
}
type LocalVariation struct {
	ParentSpecID *int64                `json:"parentSpecId"`
	Name         string                `json:"parentSpecName"`
	Required     *bool                 `json:"required"`
	Type         *int64                `json:"variationType"`
	Values       []LocalVariationValue `json:"specList"`
}
type LocalVariationValue struct {
	SpecID       *int64 `json:"specId"`
	Name         string `json:"specName"`
	GroupID      *int64 `json:"groupId"`
	GroupName    string `json:"groupName"`
	SubGroupID   *int64 `json:"subGroupId"`
	SubGroupName string `json:"subGroupName"`
}

// RequirementType is optional in a decoded response: absent is not disabled
// or optional. The official protocol spells the field key filedName.
type LocalFieldRule struct {
	FieldName       string `json:"filedName"`
	RequirementType *int64 `json:"requirementType"`
}

type LocalCategoryRules struct {
	FieldRules    []LocalFieldRule       `json:"fieldRules"`
	ExpandCatType *int64                 `json:"expandCatType"`
	SecondHand    []LocalSecondHandScope `json:"secondHand"`
	Warnings      []LocalRuleWarning     `json:"warnings"`
}

type LocalRuleWarning struct {
	Message string `json:"message"`
}
type LocalSecondHandScope struct {
	BusinessScope *int64                     `json:"businessScope"`
	Conditions    []LocalSecondHandCondition `json:"condition"`
}
type LocalSecondHandCondition struct {
	Type     *int64                  `json:"type"`
	Levels   []LocalSecondHandLevel  `json:"level"`
	Agencies []LocalSecondHandAgency `json:"agency"`
}
type LocalSecondHandLevel struct {
	Number *int64 `json:"number"`
	Name   string `json:"name"`
}
type LocalSecondHandAgency struct {
	Name   string   `json:"insName"`
	Grades []string `json:"grade"`
}

type SupplierAttributeTemplate struct {
	InputMaxSpecNum      *int64             `json:"inputMaxSpecNum"`
	ChooseAllQualifySpec *bool              `json:"chooseAllQualifySpec"`
	SingleSpecValueNum   *int64             `json:"singleSpecValueNum"`
	Properties           []SupplierProperty `json:"properties"`
}

type SupplierProperty struct {
	RefPID              *int64                          `json:"refPid"`
	PID                 *int64                          `json:"pid"`
	TemplatePID         *int64                          `json:"templatePid"`
	ParentTemplatePID   *int64                          `json:"parentTemplatePid"`
	ParentSpecID        *int64                          `json:"parentSpecId"`
	Name                string                          `json:"name"`
	NamesByLanguage     map[string]string               `json:"lang2Name"`
	Required            *bool                           `json:"required"`
	IsSale              *bool                           `json:"isSale"`
	MainSale            *bool                           `json:"mainSale"`
	ControlType         *int64                          `json:"controlType"`
	ReferenceType       *int64                          `json:"referenceType"`
	PropertyValueType   *int64                          `json:"propertyValueType"`
	ValueRule           *int64                          `json:"valueRule"`
	Feature             *int64                          `json:"feature"`
	ShowType            *int64                          `json:"showType"`
	InputMaxNum         *int64                          `json:"inputMaxNum"`
	ChooseMaxNum        *int64                          `json:"chooseMaxNum"`
	ValuePrecision      *int64                          `json:"valuePrecision"`
	MinValue            *string                         `json:"minValue"`
	MaxValue            *string                         `json:"maxValue"`
	ValueUnits          []string                        `json:"valueUnit"`
	NumberInputTitle    string                          `json:"numberInputTitle"`
	PropertyChooseTitle string                          `json:"propertyChooseTitle"`
	Group               *SupplierPropertyGroup          `json:"group"`
	Values              []SupplierPropertyValue         `json:"values"`
	ValueParents        []SupplierPropertyValueParents  `json:"templatePropertyValueParentList"`
	ShowConditions      []SupplierPropertyShowCondition `json:"showCondition"`
}

type SupplierPropertyGroup struct {
	ID   *int64 `json:"id"`
	Name string `json:"name"`
}
type SupplierPropertyValue struct {
	VID              *int64            `json:"vid"`
	SpecID           *int64            `json:"specId"`
	Value            string            `json:"value"`
	ValuesByLanguage map[string]string `json:"lang2Value"`
	ParentVIDs       []int64           `json:"parentVidList"`
	ExtendInfo       string            `json:"extendInfo"`
}
type SupplierPropertyValueParents struct {
	VIDs       []int64 `json:"vidList"`
	ParentVIDs []int64 `json:"parentVidList"`
}
type SupplierPropertyShowCondition struct {
	ParentRefPID *int64  `json:"parentRefPid"`
	ParentVIDs   []int64 `json:"parentVids"`
}
