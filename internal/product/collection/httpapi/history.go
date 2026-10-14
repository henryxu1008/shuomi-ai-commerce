package httpapi

import (
	"net/http"
	"net/url"
	"strconv"
	"task-processor/internal/product/collection"
)

func parseHistoryQuery(request *http.Request) (uint64, int, error) {
	q, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		return 0, 0, collection.ErrInvalid
	}
	limit := 20
	before := uint64(0)
	for key, values := range q {
		if len(values) != 1 || values[0] == "" {
			return 0, 0, collection.ErrInvalid
		}
		switch key {
		case "limit":
			limit, err = strconv.Atoi(values[0])
			if err != nil || limit < 1 || limit > 20 || strconv.Itoa(limit) != values[0] {
				return 0, 0, collection.ErrInvalid
			}
		case "before":
			before, err = collection.ParseHistoryVersion(values[0])
			if err != nil {
				return 0, 0, err
			}
		default:
			return 0, 0, collection.ErrInvalid
		}
	}
	return before, limit, nil
}
