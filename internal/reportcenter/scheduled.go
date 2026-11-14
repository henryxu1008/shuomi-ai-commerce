package reportcenter

import (
	"context"
	"github.com/google/uuid"
	"time"
)

const StoreDailyReportKind = "STORE_DAILY_REPORT"

// ScheduledOrigin is frozen by Workbench, never accepted as browser claims.
// The source content and the capture clock are deliberately absent from intent.
type ScheduledOrigin struct {
	OrganizationID  string    `json:"organizationId"`
	ActorID         string    `json:"actorId"`
	MemberID        string    `json:"memberId"`
	PlanID          string    `json:"planId"`
	PlanRevision    int64     `json:"planRevision"`
	OccurrenceID    string    `json:"occurrenceId"`
	ScheduledAt     time.Time `json:"scheduledAt"`
	StoreID         string    `json:"storeId"`
	Day             string    `json:"day"`
	RendererVersion int       `json:"rendererVersion"`
}

func (o ScheduledOrigin) Scope() Scope { return Scope{o.OrganizationID, o.ActorID} }
func (o ScheduledOrigin) Valid() bool {
	day, err := time.Parse(time.DateOnly, o.Day)
	expected := o.ScheduledAt.In(time.FixedZone("UTC+8", 8*3600)).AddDate(0, 0, -1).Format(time.DateOnly)
	return o.Scope().Valid() && textValid(o.MemberID, 128, false) && UUID(o.PlanID) && UUID(o.OccurrenceID) && o.PlanRevision > 0 && !o.ScheduledAt.IsZero() && UUID(o.StoreID) && err == nil && day.Format(time.DateOnly) == o.Day && expected == o.Day && o.RendererVersion == 1
}
func (o ScheduledOrigin) Key() string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("scheduled-store-report\x1f"+o.OrganizationID+"\x1f"+o.ActorID+"\x1f"+o.OccurrenceID)).String()
}
func (o ScheduledOrigin) Fingerprint() string { return Fingerprint("capture-store-daily-v1", o) }

type CaptureReceipt struct {
	ReportID string
	Digest   string
	Origin   ScheduledOrigin
}

func (r CaptureReceipt) Proof() string { return Fingerprint("saved-store-daily-v1", r) }
func (r CaptureReceipt) ValidFor(o ScheduledOrigin) bool {
	return o.Valid() && r.Origin == o && UUID(r.ReportID) && len(r.Digest) == 64
}

type ScheduledCaptureRepository interface {
	ReadScheduledCaptureReceipt(context.Context, ScheduledOrigin) (CaptureReceipt, bool, error)
	SaveScheduledCapture(context.Context, ScheduledOrigin, Snapshot, func(context.Context, *Snapshot) error) (CaptureReceipt, error)
}
type ScheduledCapture struct {
	Repository ScheduledCaptureRepository
	Authorize  func(context.Context, ScheduledOrigin) error
	Read       func(context.Context, ScheduledOrigin) (ScheduledSnapshot, error)
}

func (c *ScheduledCapture) ReadScheduledCaptureReceipt(ctx context.Context, o ScheduledOrigin) (CaptureReceipt, bool, error) {
	if c == nil || c.Repository == nil || !o.Valid() {
		return CaptureReceipt{}, false, ErrInvalid
	}
	return c.Repository.ReadScheduledCaptureReceipt(ctx, o)
}
func (c *ScheduledCapture) CaptureScheduledDaily(ctx context.Context, o ScheduledOrigin) (CaptureReceipt, error) {
	if c == nil || c.Repository == nil || c.Authorize == nil || c.Read == nil || !o.Valid() {
		return CaptureReceipt{}, ErrUnavailable
	}
	if r, found, e := c.ReadScheduledCaptureReceipt(ctx, o); e != nil || found {
		return r, e
	}
	if e := c.Authorize(ctx, o); e != nil {
		return CaptureReceipt{}, e
	}
	captured, e := c.Read(ctx, o)
	snap := captured.Snapshot
	if e != nil {
		return CaptureReceipt{}, e
	}
	if snap.ScheduledOrigin == nil || *snap.ScheduledOrigin != o || snap.Ref != (SourceRef{StoreDailyReportKind, o.OccurrenceID, "1"}) {
		return CaptureReceipt{}, ErrUnavailable
	}
	if _, _, e = snap.Bytes(); e != nil {
		return CaptureReceipt{}, e
	}
	return c.Repository.SaveScheduledCapture(ctx, o, snap, func(ctx context.Context, snapshot *Snapshot) error {
		if captured.Revalidate != nil {
			if e := captured.Revalidate(ctx, snapshot); e != nil {
				return e
			}
		}
		return c.Authorize(ctx, o)
	})
}

// Revalidate may remove an optional source before the final bytes/digest are built.
type ScheduledSnapshot struct {
	Snapshot
	Revalidate func(context.Context, *Snapshot) error
}
