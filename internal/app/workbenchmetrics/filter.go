// Package workbenchmetrics parses the shared read filters used by the workbench homepage.
package workbenchmetrics

import (
	"errors"
	"net/url"
	"time"

	"task-processor/internal/product/collection"
)

var ErrInvalid = errors.New("invalid workbench metrics filter")

const MaxCandidates = 250

type Filter struct {
	Start, End time.Time
	Stores     []string
}

func Parse(raw string, now time.Time) (Filter, error) {
	var result Filter
	if len(raw) >= 16000 {
		return result, ErrInvalid
	}
	query, err := url.ParseQuery(raw)
	if err != nil || len(query) != 3 || len(query["startDate"]) != 1 || len(query["endDate"]) != 1 ||
		len(query["storeId"]) == 0 || len(query["storeId"]) > 500 {
		return result, ErrInvalid
	}
	zone := time.FixedZone("UTC+8", 8*60*60)
	start, startErr := time.ParseInLocation(time.DateOnly, query.Get("startDate"), zone)
	end, endErr := time.ParseInLocation(time.DateOnly, query.Get("endDate"), zone)
	if startErr != nil || endErr != nil || start.Year() < 1 || start.Format(time.DateOnly) != query.Get("startDate") ||
		end.Format(time.DateOnly) != query.Get("endDate") || end.Before(start) ||
		end.Sub(start) >= 30*24*time.Hour || query.Get("endDate") >= now.In(zone).Format(time.DateOnly) {
		return result, ErrInvalid
	}
	ids := query["storeId"]
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !collection.ValidID(id) || seen[id] {
			return result, ErrInvalid
		}
		seen[id] = true
	}
	return Filter{Start: start.UTC(), End: end.AddDate(0, 0, 1).UTC(), Stores: ids}, nil
}
