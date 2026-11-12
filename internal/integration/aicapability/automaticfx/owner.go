package automaticfx

import (
	"context"
	"errors"
	"strconv"
	"time"

	d "task-processor/internal/aicapability/routeconfig"
)

const automaticActor = "system:fx:frankfurter-cfets-v1"

type platformOwner interface {
	Receipt(context.Context, d.Scope, string) (d.Revision, error)
	Publish(context.Context, string, string, d.PlatformPricesInput, time.Time) (d.Revision, error)
}

// ServiceOwner uses only the existing platform price owner and its receipt.
type ServiceOwner struct{ Service *d.PlatformPricesService }

func (o ServiceOwner) Receipt(ctx context.Context, scope d.Scope, key string) (d.Revision, error) {
	if o.Service == nil || o.Service.Repository == nil {
		return d.Revision{}, d.ErrUnavailable
	}
	return o.Service.Repository.Receipt(ctx, scope, key)
}
func (o ServiceOwner) Publish(ctx context.Context, actor, key string, in d.PlatformPricesInput, notAfter time.Time) (d.Revision, error) {
	if o.Service == nil {
		return d.Revision{}, d.ErrUnavailable
	}
	return o.Service.PublishBefore(ctx, actor, key, in, notAfter)
}

func publicationFingerprint(actor string, in d.PlatformPricesInput, notAfter time.Time) (string, error) {
	return d.PlatformPricesDeadlineFingerprint(actor, in, notAfter)
}

// publicationDeadline is derived solely from the frozen original payload.
func publicationDeadline(day string, in d.PlatformPricesInput) (time.Time, error) {
	var deadline time.Time
	for _, r := range in.FX {
		if r.Currency != "USD" || r.RateDate != day {
			continue
		}
		if !deadline.IsZero() || r.SourceURL != officialCFETSSource || r.EffectiveFrom.IsZero() || !r.VerifiedAt.Equal(r.EffectiveFrom) || !r.EffectiveUntil.After(r.EffectiveFrom) {
			return time.Time{}, ErrPublicationConflict
		}
		deadline = r.EffectiveFrom.Add(30 * time.Second)
		if r.EffectiveUntil.Before(deadline) {
			deadline = r.EffectiveUntil
		}
	}
	if deadline.IsZero() {
		return time.Time{}, ErrPublicationConflict
	}
	return deadline.UTC(), nil
}
func publicationKey(day string, attempt uint32) (string, error) {
	x, err := time.Parse("2006-01-02", day)
	if err != nil || x.Format("2006-01-02") != day || attempt == 0 || attempt > 100 {
		return "", ErrPublicationConflict
	}
	return "fx-cfets-usd-v1:" + day + ":" + strconv.FormatUint(uint64(attempt), 10), nil
}
func inspectExact(ctx context.Context, owner platformOwner, day string, attempt uint32, in d.PlatformPricesInput) (d.Revision, error) {
	if ctx == nil || owner == nil {
		return d.Revision{}, d.ErrUnavailable
	}
	key, err := publicationKey(day, attempt)
	if err != nil {
		return d.Revision{}, err
	}
	deadline, err := publicationDeadline(day, in)
	if err != nil {
		return d.Revision{}, err
	}
	hash, err := publicationFingerprint(automaticActor, in, deadline)
	if err != nil {
		return d.Revision{}, err
	}
	receipt, err := owner.Receipt(ctx, d.PlatformPricesScope(), key)
	if err != nil {
		return d.Revision{}, err
	}
	if receipt.Scope != d.PlatformPricesScope() || receipt.Key != key || receipt.ActorID != automaticActor || receipt.Fingerprint != hash {
		return d.Revision{}, ErrPublicationConflict
	}
	return receipt, nil
}

// PublishFirstAttempt must only be called once for the exact durable intent.
// Any uncertain response is reconciled by InspectPublication, never resent.
func PublishFirstAttempt(ctx context.Context, owner platformOwner, day string, attempt uint32, in d.PlatformPricesInput) (d.Revision, error) {
	if ctx == nil || owner == nil {
		return d.Revision{}, d.ErrUnavailable
	}
	key, err := publicationKey(day, attempt)
	if err != nil {
		return d.Revision{}, err
	}
	old, err := inspectExact(ctx, owner, day, attempt, in)
	if err == nil {
		return old, nil
	}
	if !errors.Is(err, d.ErrNotFound) {
		return d.Revision{}, err
	}
	deadline, err := publicationDeadline(day, in)
	if err != nil {
		return d.Revision{}, err
	}
	r, err := owner.Publish(ctx, automaticActor, key, in, deadline)
	if err != nil {
		return d.Revision{}, err
	}
	hash, err := publicationFingerprint(automaticActor, in, deadline)
	if err != nil {
		return d.Revision{}, err
	}
	if r.Scope != d.PlatformPricesScope() || r.ActorID != automaticActor || r.Key != key || r.Fingerprint != hash {
		return d.Revision{}, ErrPublicationConflict
	}
	return r, nil
}

func InspectPublication(ctx context.Context, owner platformOwner, day string, attempt uint32, in d.PlatformPricesInput) (d.Revision, error) {
	r, err := inspectExact(ctx, owner, day, attempt, in)
	if errors.Is(err, d.ErrNotFound) {
		return d.Revision{}, d.ErrOutcomeUnknown
	}
	return r, err
}
