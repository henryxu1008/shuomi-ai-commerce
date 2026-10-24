package attribute

// AttributeValueGroup 属性值组
type AttributeValueGroup struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	MultiName string `json:"multi_name"`
}

// AttributeValue 属性值信息
type AttributeValue struct {
	AttributeValueID           int                   `json:"attribute_value_id"`
	AttributeValue             string                `json:"attribute_value"`
	AttributeValueEn           string                `json:"attribute_value_en"`
	IsCustomAttributeValue     bool                  `json:"is_custom_attribute_value"`
	IsShow                     int                   `json:"is_show"`
	FromCateMis                *string               `json:"from_cate_mis"`
	SupplierID                 int                   `json:"supplier_id"`
	AttributeValueDoc          *string               `json:"attribute_value_doc"`
	AttributeValueDocImageList []any                 `json:"attribute_value_doc_image_list"`
	AttributeValueGroupList    []AttributeValueGroup `json:"attribute_value_group_list"`
}

// AttributeInfo 属性信息
type AttributeInfo struct {
	AttributeID                 int              `json:"attribute_id"`
	AttributeName               string           `json:"attribute_name"`
	AttributeNameEn             string           `json:"attribute_name_en"`
	AttributeRemarkList         []any            `json:"attribute_remark_list"`
	AttributeIsShow             int              `json:"attribute_is_show"`
	AttributeSource             int              `json:"attribute_source"`
	AttributeLabel              int              `json:"attribute_label"`
	AttributeMode               int              `json:"attribute_mode"`
	AttributeInputNum           int              `json:"attribute_input_num"`
	DataDimension               int              `json:"data_dimension"`
	AttributeStatus             int              `json:"attribute_status"`
	AttributeStatusGpc          *string          `json:"attribute_status_gpc"`
	AttributeType               int              `json:"attribute_type"`
	BusinessMode                int              `json:"business_mode"`
	IsSample                    int              `json:"is_sample"`
	SupplierID                  int              `json:"supplier_id"`
	AttributeDoc                *string          `json:"attribute_doc"`
	RuleInfoList                any              `json:"rule_info_list"`
	AttributeDocImageList       []any            `json:"attribute_doc_image_list"`
	AttributeValueInfoList      []AttributeValue `json:"attribute_value_info_list"`
	SiteTitle                   *string          `json:"site_title"`
	SiteURL                     *string          `json:"site_url"`
	CascadeAttributeID          int              `json:"cascade_attribute_id"`
	CascadeAttributeValueIDList *string          `json:"cascade_attribute_value_id_list"`
	SKCScope                    *bool            `json:"skc_scope"`
	SortOrder                   int              `json:"sort_order"`
	SourceSystemIDList          []int            `json:"source_system_id_list"`
}

// CustomAttributeRelation 自定义属性关系
type CustomAttributeRelation struct {
	PreAttributeValueID int64 `json:"pre_attribute_value_id"`
	AttributeValueID    int64 `json:"attribute_value_id"`
}
