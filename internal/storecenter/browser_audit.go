package storecenter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type BrowserAuditQuery struct {
	Limit                                    int
	Cursor, Actor, Action, Outcome, From, To string
}
type BrowserAuditItem struct {
	EventID          string     `json:"eventId"`
	OperationID      string     `json:"operationId"`
	ActorSubject     string     `json:"actorSubject"`
	MemberRef        string     `json:"memberRef"`
	Action           string     `json:"action"`
	ServerTime       time.Time  `json:"serverTime"`
	ClientObservedAt *time.Time `json:"clientObservedAt,omitempty"`
	Outcome          string     `json:"outcome"`
	ReasonCode       string     `json:"reasonCode"`
	ProfileRef       string     `json:"profileRef"`
	BindingRef       string     `json:"bindingRef"`
	FactSource       string     `json:"factSource"`
}
type BrowserAuditPage struct {
	SchemaVersion  string             `json:"schemaVersion"`
	UserID         string             `json:"userId"`
	OrganizationID string             `json:"organizationId"`
	StoreID        string             `json:"storeId"`
	Items          []BrowserAuditItem `json:"items"`
	NextCursor     *string            `json:"nextCursor"`
	Coverage       []string           `json:"coverage"`
	CanFilterActor bool               `json:"canFilterActor"`
}
type browserAuditRow struct {
	RecordedAt       time.Time
	SourceRank       int
	SourceKey        string
	OperationID      string
	ActorSubject     string
	MemberRef        string
	Action           string
	ServerTime       time.Time
	ClientObservedAt *time.Time
	Outcome          string
	ReasonCode       string
	ProfileRef       string
	BindingRef       string
	FactSource       string
}
type browserAuditCursor struct {
	Scope string    `json:"scope"`
	From  time.Time `json:"from"`
	To    time.Time `json:"to"`
	At    time.Time `json:"at"`
	Rank  int       `json:"rank"`
	Key   string    `json:"key"`
}

func browserAuditActions() []string {
	return []string{string(AuditActionBrowserAdmitted), string(AuditActionBrowserConfigured), string(AuditActionBrowserOpenReported), string(AuditActionBrowserClosed), "store_access_granted", "store_access_revoked"}
}
func (s *BrowserOperations) Audit(ctx context.Context, org, store string, q BrowserAuditQuery) (BrowserAuditPage, error) {
	if _, err := canonicalUUID(store); err != nil {
		return BrowserAuditPage{}, ErrBrowserInvalid
	}
	scoped, _ := NewMemberScopedStoreRepository(s.db, s.stores)
	access, err := scoped.authorize(ctx, org, false)
	if err != nil {
		return BrowserAuditPage{}, err
	}
	if access.Administrator {
		var record workbenchStoreRecord
		if err = s.db.WithContext(ctx).Where("organization_id=? AND id=?", org, store).Take(&record).Error; err != nil {
			return BrowserAuditPage{}, ErrNotFound
		}
	} else {
		if _, err = scoped.Get(ctx, org, store); err != nil {
			return BrowserAuditPage{}, err
		}
		if q.Actor != "" && q.Actor != access.ActorID {
			return BrowserAuditPage{}, ErrBrowserDenied
		}
		q.Actor = access.ActorID
	}
	if q.Limit == 0 {
		q.Limit = 20
	}
	if q.Limit < 1 || q.Limit > 100 || len(q.Actor) > MaxSubjectBytes {
		return BrowserAuditPage{}, ErrBrowserInvalid
	}
	if q.Action != "" {
		found := false
		for _, a := range browserAuditActions() {
			found = found || a == q.Action
		}
		if !found {
			return BrowserAuditPage{}, ErrBrowserInvalid
		}
	}
	if q.Outcome != "" && !validAuditOutcome(AuditOutcome(q.Outcome)) {
		return BrowserAuditPage{}, ErrBrowserInvalid
	}
	var after *browserAuditCursor
	if q.Cursor != "" {
		if len(q.Cursor) > 2048 {
			return BrowserAuditPage{}, ErrBrowserInvalid
		}
		raw, e := base64.RawURLEncoding.Strict().DecodeString(q.Cursor)
		if e != nil {
			return BrowserAuditPage{}, ErrBrowserInvalid
		}
		var pos browserAuditCursor
		decoder := json.NewDecoder(strings.NewReader(string(raw)))
		decoder.DisallowUnknownFields()
		if e = decoder.Decode(&pos); e != nil || pos.At.IsZero() || len(pos.Key) != 36 || (pos.Rank != 1 && pos.Rank != 2) {
			return BrowserAuditPage{}, ErrBrowserInvalid
		}
		encoded, _ := json.Marshal(pos)
		if base64.RawURLEncoding.EncodeToString(encoded) != q.Cursor {
			return BrowserAuditPage{}, ErrBrowserInvalid
		}
		if _, e = canonicalUUID(pos.Key); e != nil {
			return BrowserAuditPage{}, ErrBrowserInvalid
		}
		after = &pos
	}
	to := s.now().UTC()
	if after != nil {
		to = after.To
	}
	if q.To != "" {
		to, err = time.Parse(time.RFC3339Nano, q.To)
		if err != nil {
			return BrowserAuditPage{}, ErrBrowserInvalid
		}
		to = to.UTC()
	}
	from := to.Add(-30 * 24 * time.Hour)
	if after != nil {
		from = after.From
	}
	if q.From != "" {
		from, err = time.Parse(time.RFC3339Nano, q.From)
		if err != nil {
			return BrowserAuditPage{}, ErrBrowserInvalid
		}
		from = from.UTC()
	}
	if !from.Before(to) || to.Sub(from) > 90*24*time.Hour {
		return BrowserAuditPage{}, ErrBrowserInvalid
	}
	scope := hashTuple("store-browser-audit-v1", s.issuer, access.ActorID, org, store, q.Actor, q.Action, q.Outcome, from.Format(time.RFC3339Nano), to.Format(time.RFC3339Nano))
	if after != nil && (after.Scope != scope || !after.From.Equal(from) || !after.To.Equal(to) || after.At.Before(from) || !after.At.Before(to)) {
		return BrowserAuditPage{}, ErrBrowserInvalid
	}
	var all []browserAuditRow
	for _, rank := range []int{1, 2} {
		var rows []browserAuditRow
		var timeColumn, keyColumn string
		var sourceQuery = s.db.WithContext(ctx)
		if rank == 1 {
			timeColumn, keyColumn = "l.created_at", "l.event_id"
			sourceQuery = sourceQuery.Table("workbench_store_audit_logs l").Select(`l.created_at AS recorded_at, 1 AS source_rank,l.event_id AS source_key,l.request_key AS operation_id,l.actor_subject,coalesce(o.member_id,'') AS member_ref,l.action,l.occurred_at AS server_time,p.observed_at AS client_observed_at,l.outcome,coalesce(p.reason_code,'') AS reason_code,coalesce(o.profile_id,'') AS profile_ref,coalesce(o.binding_id,'') AS binding_ref,CASE WHEN l.action='browser_operation_admitted' THEN 'server_admission' ELSE 'client_report' END AS fact_source`).Joins("LEFT JOIN workbench_store_browser_operations o ON o.organization_id=l.organization_id AND o.operation_id=l.request_key").Joins("LEFT JOIN workbench_store_browser_reports p ON p.organization_id=o.organization_id AND p.operation_id=o.operation_id AND p.kind=CASE l.action WHEN 'browser_configuration_reported' THEN 'configure_result' WHEN 'browser_open_reported' THEN 'open_result' WHEN 'browser_close_reported' THEN 'close_result' ELSE '' END").Where("l.organization_id=? AND l.store_id=? AND l.action IN ?", org, store, browserAuditActions()[:4])
			if q.Actor != "" {
				sourceQuery = sourceQuery.Where("l.actor_subject=?", q.Actor)
			}
			if q.Action != "" {
				sourceQuery = sourceQuery.Where("l.action=?", q.Action)
			}
			if q.Outcome != "" {
				sourceQuery = sourceQuery.Where("l.outcome=?", q.Outcome)
			}
		} else {
			timeColumn, keyColumn = "g.created_at", "g.operation_id"
			sourceQuery = sourceQuery.Table("workbench_store_member_grant_operations g").Select(`g.created_at AS recorded_at,2 AS source_rank,g.operation_id AS source_key,g.operation_id,g.actor_id AS actor_subject,g.member_id AS member_ref,CASE WHEN g.active THEN 'store_access_granted' ELSE 'store_access_revoked' END AS action,g.created_at AS server_time,NULL AS client_observed_at,'succeeded' AS outcome,'' AS reason_code,'' AS profile_ref,'' AS binding_ref,'store_grant_receipt' AS fact_source`).Where("g.organization_id=? AND g.store_id=?", org, store)
			if q.Actor != "" {
				sourceQuery = sourceQuery.Where("g.actor_id=?", q.Actor)
			}
			if q.Action != "" {
				if q.Action != "store_access_granted" && q.Action != "store_access_revoked" {
					continue
				}
				sourceQuery = sourceQuery.Where("g.active=?", q.Action == "store_access_granted")
			}
			if q.Outcome != "" && q.Outcome != "succeeded" {
				continue
			}
		}
		sourceQuery = sourceQuery.Where(timeColumn+">=? AND "+timeColumn+"<?", from, to)
		if after != nil {
			if rank < after.Rank {
				sourceQuery = sourceQuery.Where(timeColumn+"<=?", after.At)
			} else if rank == after.Rank {
				sourceQuery = sourceQuery.Where(timeColumn+"<? OR ("+timeColumn+"=? AND "+keyColumn+"<?)", after.At, after.At, after.Key)
			} else {
				sourceQuery = sourceQuery.Where(timeColumn+"<?", after.At)
			}
		}
		if err = sourceQuery.Order(timeColumn + " DESC," + keyColumn + " DESC").Limit(q.Limit + 1).Scan(&rows).Error; err != nil {
			return BrowserAuditPage{}, err
		}
		all = append(all, rows...)
	}
	sort.Slice(all, func(i, j int) bool {
		if !all[i].RecordedAt.Equal(all[j].RecordedAt) {
			return all[i].RecordedAt.After(all[j].RecordedAt)
		}
		if all[i].SourceRank != all[j].SourceRank {
			return all[i].SourceRank > all[j].SourceRank
		}
		return all[i].SourceKey > all[j].SourceKey
	})
	result := BrowserAuditPage{SchemaVersion: "store-browser-v1", UserID: access.ActorID, OrganizationID: org, StoreID: store, Items: []BrowserAuditItem{}, Coverage: []string{"store_member_grant_receipts", "store_browser_application_commands", "excludes_merchant_page_activity"}}
	result.CanFilterActor = access.Administrator
	if len(all) > q.Limit {
		all = all[:q.Limit]
		last := all[len(all)-1]
		pos := browserAuditCursor{Scope: scope, From: from, To: to, At: last.RecordedAt, Rank: last.SourceRank, Key: last.SourceKey}
		raw, _ := json.Marshal(pos)
		cursor := base64.RawURLEncoding.EncodeToString(raw)
		result.NextCursor = &cursor
	}
	for _, r := range all {
		result.Items = append(result.Items, BrowserAuditItem{EventID: r.SourceKey, OperationID: r.OperationID, ActorSubject: r.ActorSubject, MemberRef: r.MemberRef, Action: r.Action, ServerTime: r.ServerTime, ClientObservedAt: r.ClientObservedAt, Outcome: r.Outcome, ReasonCode: r.ReasonCode, ProfileRef: r.ProfileRef, BindingRef: r.BindingRef, FactSource: r.FactSource})
	}
	return result, nil
}
