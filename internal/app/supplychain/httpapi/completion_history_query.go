package supplychainhttp

import (
	"net/http"
	"net/url"
	"strconv"
	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/listing/preparation"
)

func completionHistoryQuery(request *http.Request) (completiondraft.HistoryQuery, error) {
	q := completiondraft.HistoryQuery{Limit: 20}
	values, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		return q, preparation.ErrInvalid
	}
	for key, v := range values {
		if len(v) != 1 || v[0] == "" {
			return q, preparation.ErrInvalid
		}
		switch key {
		case "before":
			q.Before, err = completiondraft.ParseRevision(v[0])
		case "limit":
			q.Limit, err = strconv.Atoi(v[0])
			if err == nil && (q.Limit < 1 || strconv.Itoa(q.Limit) != v[0]) {
				err = preparation.ErrInvalid
			}
		default:
			return q, preparation.ErrInvalid
		}
		if err != nil {
			return q, preparation.ErrInvalid
		}
	}
	return q.Normalize()
}
