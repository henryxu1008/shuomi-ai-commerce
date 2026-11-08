package listingapp

import (
	"context"
	"errors"
	"reflect"
	"task-processor/internal/app/workbenchmetrics"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"time"
)

// CockpitSubmissionStore is read from the authorized local Store directory,
// never supplied as platform metadata by the browser.
type CockpitSubmissionStore struct{ ID, Platform, Region string }
type submissionOverviewCounts struct {
	Confirmed int  `json:"confirmed"`
	Rejected  *int `json:"rejected,omitempty"`
	Unknown   int  `json:"unknown"`
}
type submissionOverviewRow struct {
	Platform string                    `json:"platform"`
	Status   string                    `json:"status"`
	Stores   int                       `json:"stores"`
	Counts   *submissionOverviewCounts `json:"counts,omitempty"`
}
type submissionOverview struct {
	Status         string                  `json:"status"`
	CapturedAt     time.Time               `json:"capturedAt"`
	Rows           []submissionOverviewRow `json:"rows"`
	ExcludedStores int                     `json:"excludedStores"`
	UnknownTotal   *int                    `json:"unknownTotal,omitempty"`
}

func (a *Application) SubmissionOverview(ctx context.Context, filter workbenchmetrics.Filter, now time.Time) (submissionOverview, error) {
	result := submissionOverview{Status: "incomplete", CapturedAt: now.UTC(), Rows: []submissionOverviewRow{
		{Platform: "walmart", Status: "empty"}, {Platform: "tiktokshop", Status: "empty"},
		{Platform: "shein", Status: "empty"}, {Platform: "temu", Status: "empty"},
	}}
	if len(filter.Stores) == 0 || len(filter.Stores) > 500 || !filter.Start.Before(filter.End) {
		return result, record.ErrInvalid
	}
	if a.Authorization == nil || a.CockpitSubmissionStores == nil || a.CockpitStoreAccess == nil {
		return result, record.ErrUnavailable
	}
	scope, err := a.Authorization.Authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return result, err
	}
	readStores := func() (map[string]CockpitSubmissionStore, error) {
		if err := a.CockpitStoreAccess(ctx, scope, filter.Stores); err != nil {
			return nil, err
		}
		stores, err := a.CockpitSubmissionStores(ctx, scope, filter.Stores)
		if err != nil {
			return nil, err
		}
		byID := make(map[string]CockpitSubmissionStore, len(stores))
		for _, store := range stores {
			if _, ok := byID[store.ID]; ok {
				return nil, record.ErrUnavailable
			}
			byID[store.ID] = store
		}
		if len(byID) != len(filter.Stores) {
			return nil, record.ErrUnavailable
		}
		for _, id := range filter.Stores {
			if _, ok := byID[id]; !ok {
				return nil, record.ErrUnavailable
			}
		}
		return byID, nil
	}
	stores, err := readStores()
	if err != nil {
		return result, err
	}
	for _, id := range filter.Stores {
		store := stores[id]
		rowIndex := -1
		for i := range result.Rows {
			if store.Platform == result.Rows[i].Platform && (store.Region == "US" || store.Platform == "temu") {
				rowIndex = i
				break
			}
		}
		if rowIndex < 0 {
			result.ExcludedStores++
			continue
		}
		row := &result.Rows[rowIndex]
		row.Stores++
		if row.Status == "limited" || row.Status == "unavailable" {
			continue
		}
		if err := ctx.Err(); err != nil {
			return submissionOverview{}, record.ErrUnavailable
		}
		single := filter
		single.Stores = []string{id}
		var status string
		var counts submissionOverviewCounts
		var readErr error
		switch store.Platform {
		case "walmart", "tiktokshop":
			var value storeSubmissionMetrics
			if store.Platform == "walmart" {
				value, readErr = a.WalmartFeedMetrics(ctx, single, now)
			} else {
				value, readErr = a.TikTokPublishMetrics(ctx, single, now)
			}
			status = value.Status
			if value.Counts != nil {
				rejected := value.Counts.Rejected
				counts = submissionOverviewCounts{Confirmed: value.Counts.Accepted, Rejected: &rejected, Unknown: value.Counts.Unknown}
			}
		case "shein", "temu":
			var value publicationMetrics
			if store.Platform == "shein" {
				value, readErr = a.SheinPublishMetrics(ctx, single, now)
			} else {
				value, readErr = a.TemuCreateMetrics(ctx, single, now)
			}
			status = value.Status
			if value.Counts != nil {
				counts = submissionOverviewCounts{Confirmed: value.Counts.Confirmed, Unknown: value.Counts.Unknown}
			}
		}
		if errors.Is(readErr, record.ErrForbidden) || errors.Is(readErr, preparation.ErrForbidden) {
			return submissionOverview{}, readErr
		}
		if readErr != nil || status != "complete" && status != "limited" {
			row.Status = "unavailable"
			row.Counts = nil
			continue
		}
		if status == "limited" {
			row.Status = "limited"
			row.Counts = nil
			continue
		}
		if row.Counts == nil {
			row.Counts = &submissionOverviewCounts{Rejected: counts.Rejected}
			if row.Counts.Rejected != nil {
				zero := 0
				row.Counts.Rejected = &zero
			}
		}
		row.Counts.Confirmed += counts.Confirmed
		row.Counts.Unknown += counts.Unknown
		total := row.Counts.Confirmed + row.Counts.Unknown
		if counts.Rejected != nil {
			*row.Counts.Rejected += *counts.Rejected
			total += *row.Counts.Rejected
		}
		row.Status = "complete"
		if total > workbenchmetrics.MaxCandidates {
			row.Status = "limited"
			row.Counts = nil
		}
	}
	if ctx.Err() != nil {
		return submissionOverview{}, record.ErrUnavailable
	}
	fresh, err := a.Authorization.Authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return submissionOverview{}, err
	}
	if fresh != scope {
		return submissionOverview{}, record.ErrForbidden
	}
	current, err := readStores()
	if err != nil {
		return submissionOverview{}, err
	}
	if !reflect.DeepEqual(stores, current) {
		return submissionOverview{}, record.ErrForbidden
	}
	complete := result.ExcludedStores == 0
	unknown := 0
	for _, row := range result.Rows {
		if row.Status != "empty" && row.Status != "complete" {
			complete = false
		}
		if row.Counts != nil {
			unknown += row.Counts.Unknown
		}
	}
	if complete {
		result.Status = "complete"
		result.UnknownTotal = &unknown
	}
	return result, nil
}
