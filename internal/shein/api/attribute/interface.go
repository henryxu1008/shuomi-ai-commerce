package attribute

// CustomAttributeRelation 自定义属性关系
type CustomAttributeRelation struct {
	PreAttributeValueID int64 `json:"pre_attribute_value_id"`
	AttributeValueID    int64 `json:"attribute_value_id"`
}
