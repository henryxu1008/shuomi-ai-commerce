package commercialbilling

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"task-processor/internal/commercial/billing"
)

type dataRowPricePublicationRow struct {
	ID             string    `gorm:"primaryKey;size:128"`
	ActorID        string    `gorm:"size:128;not null;uniqueIndex:uq_data_row_price_request"`
	IdempotencyKey string    `gorm:"size:128;not null;uniqueIndex:uq_data_row_price_request"`
	Fingerprint    string    `gorm:"size:64;not null"`
	ResultJSON     string    `gorm:"not null"`
	CreatedAt      time.Time `gorm:"not null;index"`
}

func (dataRowPricePublicationRow) TableName() string { return "commercial_data_row_price_publications" }

func (r *Repository) CurrentDataRowOffer(ctx context.Context) (billing.Offer, error) {
	values, err := r.ListDataRowOffers(ctx)
	if err != nil {
		return billing.Offer{}, err
	}
	if len(values) != 1 {
		return billing.Offer{}, billing.ErrOfferUnavailable
	}
	return values[0], nil
}
func validPriceToken(s string) bool {
	return s != "" && len(s) <= 128 && strings.TrimSpace(s) == s && !strings.ContainsAny(s, "\x00\r\n\t")
}
func validPriceCommand(c billing.DataRowPriceCommand) bool {
	return validPriceToken(c.ActorID) && validPriceToken(c.OrganizationID) && validPriceToken(c.ExpectedVersion) && validPriceToken(c.IdempotencyKey) && c.UnitPriceMinor >= 0 && utf8.ValidString(c.Reason) && len(c.Reason) <= 500 && strings.TrimSpace(c.Reason) != "" && !strings.ContainsRune(c.Reason, 0)
}
func readPriceReceipt(db *gorm.DB, actor, key string) (dataRowPricePublicationRow, error) {
	var row dataRowPricePublicationRow
	err := db.Where("actor_id=? AND idempotency_key=?", actor, key).Take(&row).Error
	return row, err
}
func decodePriceReceipt(row dataRowPricePublicationRow) (billing.DataRowPriceReceipt, error) {
	var out billing.DataRowPriceReceipt
	if json.Unmarshal([]byte(row.ResultJSON), &out) != nil || out.ID != row.ID || out.ActorID != row.ActorID || out.Offer.Validate() != nil || out.Offer.ProductKind != billing.ProductDataRow {
		return out, billing.ErrFeatureUnavailable
	}
	return out, nil
}

func (r *Repository) PublishDataRowPrice(ctx context.Context, c billing.DataRowPriceCommand) (billing.DataRowPriceReceipt, error) {
	if r == nil || r.db == nil || !validPriceCommand(c) {
		return billing.DataRowPriceReceipt{}, billing.ErrInvalid
	}
	digest := fingerprint(struct {
		ActorID, OrganizationID string
		Command                 billing.DataRowPriceCommand
	}{c.ActorID, c.OrganizationID, c})
	var out billing.DataRowPriceReceipt
	replay := func(tx *gorm.DB) (bool, error) {
		row, e := readPriceReceipt(tx, c.ActorID, c.IdempotencyKey)
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return false, nil
		}
		if e != nil {
			return false, e
		}
		if row.Fingerprint != digest {
			return false, billing.ErrConflict
		}
		out, e = decodePriceReceipt(row)
		return true, e
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if ok, e := replay(tx); ok || e != nil {
			return e
		}
		// Lock the expected immutable version, then check its mutable sale status.
		// A waiter must replay again after the lock to resolve the same request.
		var old offerRow
		e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("product_kind=? AND pricing_version=?", billing.ProductDataRow, c.ExpectedVersion).Take(&old).Error
		if errors.Is(e, gorm.ErrRecordNotFound) {
			var count int64
			if tx.Model(&offerRow{}).Where("product_kind=?", billing.ProductDataRow).Count(&count).Error != nil {
				return billing.ErrFeatureUnavailable
			}
			if count == 0 {
				return billing.ErrOfferUnavailable
			}
			return billing.ErrConflict
		}
		if e != nil {
			return e
		}
		if ok, e := replay(tx); ok || e != nil {
			return e
		}
		current := offerFromRow(old)
		if current.Status != billing.OfferActive {
			return billing.ErrConflict
		}
		now := r.now().UTC()
		if current.Validate() != nil {
			return billing.ErrOfferUnavailable
		}
		if (current.StartsAt != nil && now.Before(*current.StartsAt)) || (current.ExpiresAt != nil && !now.Before(*current.ExpiresAt)) {
			return billing.ErrOfferUnavailable
		}
		next := current
		next.OfferID = uuid.NewString()
		next.PricingVersion = uuid.NewString()
		next.UnitPriceMinor = c.UnitPriceMinor
		next.SettlementMode = billing.SettlementWallet
		if c.UnitPriceMinor == 0 {
			next.SettlementMode = billing.SettlementZeroPrice
		}
		next.StartsAt = nil
		next.ExpiresAt = nil
		if next.Validate() != nil {
			return billing.ErrInvalid
		}
		changed := tx.Model(&offerRow{}).Where("offer_id=? AND status=?", old.OfferID, billing.OfferActive).Update("status", billing.OfferDisabled)
		if changed.Error != nil {
			return changed.Error
		}
		if changed.RowsAffected != 1 {
			return billing.ErrConflict
		}
		row := offerToRow(next)
		if e := tx.Create(&row).Error; e != nil {
			return e
		}
		out = billing.DataRowPriceReceipt{ID: uuid.NewString(), ActorID: c.ActorID, OrganizationID: c.OrganizationID, PreviousOfferID: current.OfferID, PreviousVersion: current.PricingVersion, Offer: next, Reason: c.Reason, CreatedAt: now}
		raw, e := json.Marshal(out)
		if e != nil {
			return e
		}
		return tx.Create(&dataRowPricePublicationRow{ID: out.ID, ActorID: c.ActorID, IdempotencyKey: c.IdempotencyKey, Fingerprint: digest, ResultJSON: string(raw), CreatedAt: now}).Error
	})
	return out, err
}
func (r *Repository) ReadDataRowPriceReceipt(ctx context.Context, actor, key string) (billing.DataRowPriceReceipt, error) {
	if r == nil || r.db == nil || !validPriceToken(actor) || !validPriceToken(key) {
		return billing.DataRowPriceReceipt{}, billing.ErrInvalid
	}
	row, e := readPriceReceipt(r.db.WithContext(ctx), actor, key)
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return billing.DataRowPriceReceipt{}, billing.ErrNotFound
	}
	if e != nil {
		return billing.DataRowPriceReceipt{}, e
	}
	return decodePriceReceipt(row)
}
func (r *Repository) DataRowPriceHistory(ctx context.Context, cursor string, limit int) (billing.DataRowPriceHistory, error) {
	out := billing.DataRowPriceHistory{Items: []billing.DataRowPriceReceipt{}}
	if r == nil || r.db == nil || limit < 1 || limit > 50 || len(cursor) > 512 {
		return out, billing.ErrInvalid
	}
	q := r.db.WithContext(ctx).Model(&dataRowPricePublicationRow{})
	if cursor != "" {
		var pos struct {
			CreatedAt time.Time
			ID        string
		}
		raw, e := base64.RawURLEncoding.DecodeString(cursor)
		if e != nil || json.Unmarshal(raw, &pos) != nil || pos.CreatedAt.IsZero() || !validPriceToken(pos.ID) {
			return out, billing.ErrInvalid
		}
		q = q.Where("created_at < ? OR (created_at=? AND id < ?)", pos.CreatedAt, pos.CreatedAt, pos.ID)
	}
	var rows []dataRowPricePublicationRow
	if e := q.Order("created_at DESC,id DESC").Limit(limit + 1).Find(&rows).Error; e != nil {
		return out, e
	}
	more := len(rows) > limit
	if more {
		rows = rows[:limit]
	}
	for _, row := range rows {
		v, e := decodePriceReceipt(row)
		if e != nil {
			return out, e
		}
		out.Items = append(out.Items, v)
	}
	if more {
		last := rows[len(rows)-1]
		raw, _ := json.Marshal(struct {
			CreatedAt time.Time
			ID        string
		}{last.CreatedAt, last.ID})
		out.NextCursor = base64.RawURLEncoding.EncodeToString(raw)
	}
	return out, nil
}
