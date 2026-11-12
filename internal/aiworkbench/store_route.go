package aiworkbench

import (
	strictjson "sigs.k8s.io/json"
	"time"
)

type StoreRoute struct {
	Kind   string `json:"kind"`
	Status string `json:"status"`
}

// Call only while holding the routing command row lock, using a database
// realtime value obtained after acquiring that lock.
func RouteCommitEligible(databaseNow, commitDeadline time.Time) bool {
	return !databaseNow.IsZero() && !commitDeadline.IsZero() && !databaseNow.After(commitDeadline)
}

// settled must come from the matching authoritative invocation receipt and
// mean successful usage settlement, not merely a provider success response.
func RouteLostEligible(databaseNow, commitDeadline time.Time, settled bool) bool {
	return settled && !databaseNow.IsZero() && !commitDeadline.IsZero() && databaseNow.After(commitDeadline)
}

// ParseStoreRoute accepts only a closed decision, never identity or commands.
func ParseStoreRoute(raw []byte) (StoreRoute, error) {
	var out StoreRoute
	if len(raw) == 0 || len(raw) > 4096 {
		return out, ErrInvalid
	}
	var wire struct {
		Kind   *string `json:"kind"`
		Status *string `json:"status"`
	}
	violations, err := strictjson.UnmarshalStrict(raw, &wire, strictjson.DisallowDuplicateFields, strictjson.DisallowUnknownFields)
	if err != nil || len(violations) > 0 {
		return StoreRoute{}, ErrInvalid
	}
	if wire.Kind == nil || wire.Status == nil {
		return StoreRoute{}, ErrInvalid
	}
	out = StoreRoute{Kind: *wire.Kind, Status: *wire.Status}
	switch out.Kind {
	case "STORE_OVERVIEW":
		if out.Status != "ALL_NON_DELETED" && out.Status != "ACTIVE" {
			return StoreRoute{}, ErrInvalid
		}
	case "PRODUCT_TASK", "CLARIFY":
		if out.Status != "" {
			return StoreRoute{}, ErrInvalid
		}
	default:
		return StoreRoute{}, ErrInvalid
	}
	return out, nil
}
