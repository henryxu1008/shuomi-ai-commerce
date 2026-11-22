package httpapi

import (
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"net/url"
)

func urlQuery(raw string) (map[string]string, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return nil, collection.ErrInvalid
	}
	result := map[string]string{}
	for key, value := range values {
		if key != "after" && key != "keyword" && key != "limit" || len(value) != 1 {
			return nil, collection.ErrInvalid
		}
		result[key] = value[0]
	}
	return result, nil
}
