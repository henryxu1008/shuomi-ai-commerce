package shein

func cloneResolvedSaleAttributeMap(input map[string]ResolvedSaleAttribute) map[string]ResolvedSaleAttribute {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]ResolvedSaleAttribute, len(input))
	for key, value := range input {
		out[key] = cloneResolvedSaleAttribute(value)
	}
	return out
}

func cloneResolvedSaleAttributeSliceMap(input map[string][]ResolvedSaleAttribute) map[string][]ResolvedSaleAttribute {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string][]ResolvedSaleAttribute, len(input))
	for key, value := range input {
		cloned := make([]ResolvedSaleAttribute, len(value))
		for index := range value {
			cloned[index] = cloneResolvedSaleAttribute(value[index])
		}
		out[key] = cloned
	}
	return out
}

func cloneResolvedSaleAttribute(value ResolvedSaleAttribute) ResolvedSaleAttribute {
	if value.AttributeValueID != nil {
		attributeValueID := *value.AttributeValueID
		value.AttributeValueID = &attributeValueID
	}
	return value
}
