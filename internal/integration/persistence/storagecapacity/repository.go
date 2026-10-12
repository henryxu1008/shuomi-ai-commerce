package storagecapacitystore

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"task-processor/internal/ledger/storagecapacity"
)

type accountRow struct {
	OrganizationID string `gorm:"primaryKey;size:128"`
	Occupied       int64
	Reserved       int64
}

func (accountRow) TableName() string { return "storage_capacity_accounts" }

type reservationRow struct {
	ID              string `gorm:"primaryKey;size:36"`
	OrganizationID  string `gorm:"size:128;uniqueIndex:storage_write_identity"`
	Owner           string `gorm:"size:64;uniqueIndex:storage_write_identity"`
	OperationID     string `gorm:"size:256;uniqueIndex:storage_write_identity"`
	Fingerprint     string `gorm:"size:64"`
	MaxBytes        int64
	MaxObjects      int
	Remaining       int64
	State           string `gorm:"size:16"`
	TerminalReceipt string `gorm:"size:256"`
}

func (reservationRow) TableName() string { return "storage_capacity_reservations" }

type objectRow struct {
	Identity       string `gorm:"primaryKey;size:64"`
	OrganizationID string `gorm:"size:128;index"`
	Namespace      string `gorm:"size:256"`
	Key            string `gorm:"size:1024"`
	SHA256         string `gorm:"size:64"`
	SizeBytes      int64
	Category       string `gorm:"size:32"`
	Stored         bool
}

func (objectRow) TableName() string { return "storage_capacity_objects" }

type bindingRow struct {
	ReservationID  string `gorm:"primaryKey;size:36"`
	ObjectIdentity string `gorm:"primaryKey;size:64"`
	Confirmed      bool
}

func (bindingRow) TableName() string { return "storage_capacity_bindings" }

type lotRow struct {
	OrderID          string `gorm:"primaryKey;size:128"`
	ItemID           string `gorm:"size:128;uniqueIndex"`
	OrganizationID   string `gorm:"size:128;index"`
	TermsFingerprint string `gorm:"size:64"`
	Bytes            int64
	DurationSeconds  int64
	MoneyReceipt     string `gorm:"size:256"`
	ActivatedAt      *time.Time
	ExpiresAt        *time.Time
}

func (lotRow) TableName() string { return "storage_capacity_lots" }

type Repository struct {
	db  *gorm.DB
	now func() time.Time
}

func New(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, storagecapacity.ErrUnavailable
	}
	return &Repository{db: db, now: func() time.Time { return time.Now().UTC() }}, nil
}
func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return storagecapacity.ErrUnavailable
	}
	return db.AutoMigrate(&accountRow{}, &reservationRow{}, &objectRow{}, &bindingRow{}, &lotRow{}, &namespaceRow{})
}
func (r *Repository) clock(tx *gorm.DB) (time.Time, error) {
	if tx.Dialector.Name() == "postgres" {
		var t time.Time
		e := tx.Raw("SELECT clock_timestamp()").Scan(&t).Error
		return t.UTC(), e
	}
	return r.now().UTC(), nil
}
func (r *Repository) locked(ctx context.Context, org string, fn func(*gorm.DB, *accountRow, time.Time) error) error {
	if !storagecapacity.ValidID(org, 128) {
		return storagecapacity.ErrInvalid
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if tx.Dialector.Name() == "postgres" {
			if e := tx.Exec("SET LOCAL lock_timeout = '2s'").Error; e != nil {
				return e
			}
			if e := tx.Exec("SET LOCAL statement_timeout = '5s'").Error; e != nil {
				return e
			}
		}
		if e := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&accountRow{OrganizationID: org}).Error; e != nil {
			return e
		}
		var a accountRow
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, "organization_id = ?", org).Error; e != nil {
			return e
		}
		now, e := r.clock(tx)
		if e != nil {
			return e
		}
		return fn(tx, &a, now)
	})
}
func quota(tx *gorm.DB, org string, now time.Time) (int64, error) {
	var total int64
	if e := tx.Model(&lotRow{}).Select("COALESCE(SUM(bytes),0)").Where("organization_id = ? AND activated_at IS NOT NULL AND expires_at > ?", org, now).Scan(&total).Error; e != nil {
		return 0, e
	}
	if total < 0 || total > math.MaxInt64-storagecapacity.BaseBytes {
		return 0, storagecapacity.ErrConflict
	}
	return storagecapacity.BaseBytes + total, nil
}
func saveAccount(tx *gorm.DB, a *accountRow) error {
	if a.Occupied < 0 || a.Reserved < 0 {
		return storagecapacity.ErrConflict
	}
	return tx.Save(a).Error
}
func findReservation(tx *gorm.DB, org, id string) (reservationRow, error) {
	var row reservationRow
	e := tx.First(&row, "organization_id = ? AND id = ?", org, id).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		e = storagecapacity.ErrNotFound
	}
	return row, e
}
func toReservation(row reservationRow) storagecapacity.Reservation {
	return storagecapacity.Reservation{ID: row.ID, State: row.State, Intent: storagecapacity.WriteIntent{OrganizationID: row.OrganizationID, Owner: row.Owner, OperationID: row.OperationID, Fingerprint: row.Fingerprint, MaxBytes: row.MaxBytes, MaxObjects: row.MaxObjects}}
}

func (r *Repository) FindWrite(ctx context.Context, org, owner, operation string) (storagecapacity.Reservation, error) {
	var row reservationRow
	e := r.db.WithContext(ctx).First(&row, "organization_id = ? AND owner = ? AND operation_id = ?", org, owner, operation).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		e = storagecapacity.ErrNotFound
	}
	return toReservation(row), e
}

func (r *Repository) FindObjectWrite(ctx context.Context, org, owner string, o storagecapacity.Object) (storagecapacity.Reservation, error) {
	if e := o.Validate(); e != nil {
		return storagecapacity.Reservation{}, e
	}
	var object objectRow
	if e := r.db.WithContext(ctx).First(&object, "identity = ? AND organization_id = ?", objectIdentity(o.Namespace, o.Key), org).Error; e != nil {
		return storagecapacity.Reservation{}, storagecapacity.ErrNotFound
	}
	if object.Namespace != o.Namespace || object.Key != o.Key || object.SHA256 != o.SHA256 || object.SizeBytes != o.SizeBytes || object.Category != o.Category {
		return storagecapacity.Reservation{}, storagecapacity.ErrConflict
	}
	var rows []reservationRow
	e := r.db.WithContext(ctx).Model(&reservationRow{}).Select("storage_capacity_reservations.*").Joins("JOIN storage_capacity_bindings b ON b.reservation_id = storage_capacity_reservations.id").Where("storage_capacity_reservations.organization_id = ? AND storage_capacity_reservations.owner = ? AND b.object_identity = ?", org, owner, object.Identity).Limit(2).Find(&rows).Error
	if e != nil {
		return storagecapacity.Reservation{}, e
	}
	if len(rows) == 0 {
		return storagecapacity.Reservation{}, storagecapacity.ErrNotFound
	}
	if len(rows) != 1 {
		return storagecapacity.Reservation{}, storagecapacity.ErrConflict
	}
	return toReservation(rows[0]), nil
}
func (r *Repository) ReserveWrite(ctx context.Context, i storagecapacity.WriteIntent) (result storagecapacity.Reservation, err error) {
	if err = i.Validate(); err != nil {
		return
	}
	err = r.locked(ctx, i.OrganizationID, func(tx *gorm.DB, a *accountRow, now time.Time) error {
		var row reservationRow
		e := tx.First(&row, "organization_id = ? AND owner = ? AND operation_id = ?", i.OrganizationID, i.Owner, i.OperationID).Error
		if e == nil {
			if row.Fingerprint != i.Fingerprint || row.MaxBytes != i.MaxBytes || row.MaxObjects != i.MaxObjects {
				return storagecapacity.ErrConflict
			}
			if row.State == "ABORTED" {
				return storagecapacity.ErrConflict
			}
			result = toReservation(row)
			return nil
		}
		if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		q, e := quota(tx, i.OrganizationID, now)
		if e != nil {
			return e
		}
		if a.Occupied > q || a.Reserved > q-a.Occupied || i.MaxBytes > q-a.Occupied-a.Reserved {
			return storagecapacity.ErrExceeded
		}
		row = reservationRow{ID: uuid.NewString(), OrganizationID: i.OrganizationID, Owner: i.Owner, OperationID: i.OperationID, Fingerprint: i.Fingerprint, MaxBytes: i.MaxBytes, MaxObjects: i.MaxObjects, Remaining: i.MaxBytes, State: "RESERVED"}
		if e = tx.Create(&row).Error; e != nil {
			return e
		}
		a.Reserved += i.MaxBytes
		result = toReservation(row)
		return saveAccount(tx, a)
	})
	return
}
func objectIdentity(namespace, key string) string {
	return storagecapacity.Digest(namespace + "\x00" + key)
}
func (r *Repository) BindObjects(ctx context.Context, org, id string, objects []storagecapacity.Object) error {
	if len(objects) < 1 || len(objects) > 16 {
		return storagecapacity.ErrInvalid
	}
	for _, o := range objects {
		if e := o.Validate(); e != nil {
			return e
		}
	}
	return r.locked(ctx, org, func(tx *gorm.DB, a *accountRow, now time.Time) error {
		row, e := findReservation(tx, org, id)
		if e != nil {
			return e
		}
		if row.State == "ABORTED" {
			return storagecapacity.ErrConflict
		}
		var bindings []bindingRow
		if e = tx.Where("reservation_id = ?", id).Find(&bindings).Error; e != nil {
			return e
		}
		var total int64
		for _, b := range bindings {
			var o objectRow
			if e = tx.First(&o, "identity = ?", b.ObjectIdentity).Error; e != nil {
				return e
			}
			total += o.SizeBytes
		}
		for _, o := range objects {
			identity := objectIdentity(o.Namespace, o.Key)
			var existing objectRow
			e = tx.First(&existing, "identity = ?", identity).Error
			if e == nil && (existing.OrganizationID != org || existing.Namespace != o.Namespace || existing.Key != o.Key || existing.SHA256 != o.SHA256 || existing.SizeBytes != o.SizeBytes || existing.Category != o.Category) {
				return storagecapacity.ErrConflict
			}
			if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
				return e
			}
			var b bindingRow
			be := tx.First(&b, "reservation_id = ? AND object_identity = ?", id, identity).Error
			if be == nil {
				continue
			}
			if row.State == "CLOSED" {
				return storagecapacity.ErrConflict
			}
			if !errors.Is(be, gorm.ErrRecordNotFound) {
				return be
			}
			if len(bindings) >= row.MaxObjects || o.SizeBytes > row.MaxBytes-total {
				return storagecapacity.ErrConflict
			}
			total += o.SizeBytes
			if errors.Is(e, gorm.ErrRecordNotFound) {
				if e = tx.Create(&objectRow{Identity: identity, OrganizationID: org, Namespace: o.Namespace, Key: o.Key, SHA256: o.SHA256, SizeBytes: o.SizeBytes, Category: o.Category}).Error; e != nil {
					return e
				}
			}
			b = bindingRow{ReservationID: id, ObjectIdentity: identity}
			if e = tx.Create(&b).Error; e != nil {
				return e
			}
			bindings = append(bindings, b)
		}
		return nil
	})
}
func (r *Repository) ClaimDispatch(ctx context.Context, org, id string) error {
	return r.locked(ctx, org, func(tx *gorm.DB, a *accountRow, now time.Time) error {
		row, e := findReservation(tx, org, id)
		if e != nil {
			return e
		}
		if row.State == "DISPATCHED" || row.State == "CLOSED" {
			return nil
		}
		if row.State != "RESERVED" {
			return storagecapacity.ErrConflict
		}
		return tx.Model(&row).Update("state", "DISPATCHED").Error
	})
}
func (r *Repository) AbortUnstarted(ctx context.Context, org, id string) error {
	return r.locked(ctx, org, func(tx *gorm.DB, a *accountRow, now time.Time) error {
		row, e := findReservation(tx, org, id)
		if e != nil {
			return e
		}
		if row.State == "ABORTED" {
			return nil
		}
		if row.State != "RESERVED" {
			return storagecapacity.ErrUnknown
		}
		a.Reserved -= row.Remaining
		row.Remaining = 0
		row.State = "ABORTED"
		if e = tx.Save(&row).Error; e != nil {
			return e
		}
		return saveAccount(tx, a)
	})
}
func (r *Repository) ConfirmStored(ctx context.Context, org, id, namespace, key, hash string, size int64) error {
	return r.locked(ctx, org, func(tx *gorm.DB, a *accountRow, now time.Time) error {
		row, e := findReservation(tx, org, id)
		if e != nil {
			return e
		}
		identity := objectIdentity(namespace, key)
		var o objectRow
		if e = tx.First(&o, "identity = ? AND organization_id = ?", identity, org).Error; e != nil {
			return storagecapacity.ErrNotFound
		}
		if o.Namespace != namespace || o.Key != key || o.SHA256 != hash || o.SizeBytes != size {
			return storagecapacity.ErrConflict
		}
		var b bindingRow
		if e = tx.First(&b, "reservation_id = ? AND object_identity = ?", id, identity).Error; e != nil {
			return storagecapacity.ErrNotFound
		}
		if b.Confirmed {
			return nil
		}
		if row.State != "DISPATCHED" || row.Remaining < size {
			return storagecapacity.ErrConflict
		}
		if !o.Stored {
			a.Occupied += size
			o.Stored = true
			if e = tx.Save(&o).Error; e != nil {
				return e
			}
		}
		a.Reserved -= size
		row.Remaining -= size
		b.Confirmed = true
		if e = tx.Save(&b).Error; e != nil {
			return e
		}
		if e = tx.Save(&row).Error; e != nil {
			return e
		}
		return saveAccount(tx, a)
	})
}
func (r *Repository) CloseWrite(ctx context.Context, org, id string, p storagecapacity.CloseProof) error {
	if !storagecapacity.ValidID(p.TerminalReceipt, 256) {
		return storagecapacity.ErrInvalid
	}
	return r.locked(ctx, org, func(tx *gorm.DB, a *accountRow, now time.Time) error {
		row, e := findReservation(tx, org, id)
		if e != nil {
			return e
		}
		if p.Owner != row.Owner || p.OperationID != row.OperationID || p.Fingerprint != row.Fingerprint {
			return storagecapacity.ErrConflict
		}
		if row.State == "CLOSED" {
			if row.TerminalReceipt != p.TerminalReceipt {
				return storagecapacity.ErrConflict
			}
			return nil
		}
		if row.State != "DISPATCHED" {
			return storagecapacity.ErrConflict
		}
		var unconfirmed int64
		if e = tx.Model(&bindingRow{}).Where("reservation_id = ? AND confirmed = ?", id, false).Count(&unconfirmed).Error; e != nil {
			return e
		}
		if unconfirmed > 0 {
			return storagecapacity.ErrUnknown
		}
		a.Reserved -= row.Remaining
		row.Remaining = 0
		row.State = "CLOSED"
		row.TerminalReceipt = p.TerminalReceipt
		if e = tx.Save(&row).Error; e != nil {
			return e
		}
		return saveAccount(tx, a)
	})
}
func (r *Repository) PrepareLot(ctx context.Context, l storagecapacity.PurchaseLot) error {
	if e := l.Validate(); e != nil {
		return e
	}
	return r.locked(ctx, l.OrganizationID, func(tx *gorm.DB, a *accountRow, now time.Time) error { _, e := prepareLot(tx, l); return e })
}
func prepareLot(tx *gorm.DB, l storagecapacity.PurchaseLot) (lotRow, error) {
	var row lotRow
	e := tx.First(&row, "order_id = ?", l.OrderID).Error
	if e == nil {
		if row.OrganizationID != l.OrganizationID || row.ItemID != l.ItemID || row.Bytes != l.Bytes || row.DurationSeconds != l.DurationSeconds || row.TermsFingerprint != l.TermsFingerprint {
			return row, storagecapacity.ErrConflict
		}
		return row, nil
	}
	if !errors.Is(e, gorm.ErrRecordNotFound) {
		return row, e
	}
	row = lotRow{OrderID: l.OrderID, OrganizationID: l.OrganizationID, ItemID: l.ItemID, Bytes: l.Bytes, DurationSeconds: l.DurationSeconds, TermsFingerprint: l.TermsFingerprint}
	return row, tx.Create(&row).Error
}

// ActivateLotInTransaction is called by the commercial order UoW only after
// Money's durable COMMITTED receipt has been verified. It never commits the UoW.
func (r *Repository) ActivateLotInTransaction(ctx context.Context, tx *gorm.DB, l storagecapacity.PurchaseLot, receipt string) error {
	if e := l.Validate(); e != nil {
		return e
	}
	if tx == nil || !storagecapacity.ValidID(receipt, 256) {
		return storagecapacity.ErrInvalid
	}
	tx = tx.WithContext(ctx)
	if e := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&accountRow{OrganizationID: l.OrganizationID}).Error; e != nil {
		return e
	}
	var a accountRow
	if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, "organization_id = ?", l.OrganizationID).Error; e != nil {
		return e
	}
	var row lotRow
	if e := tx.First(&row, "order_id = ?", l.OrderID).Error; e != nil {
		return storagecapacity.ErrNotFound
	}
	if row.OrganizationID != l.OrganizationID || row.ItemID != l.ItemID || row.Bytes != l.Bytes || row.DurationSeconds != l.DurationSeconds || row.TermsFingerprint != l.TermsFingerprint {
		return storagecapacity.ErrConflict
	}
	if row.ActivatedAt != nil {
		if row.MoneyReceipt != receipt {
			return storagecapacity.ErrConflict
		}
		return nil
	}
	now, e := r.clock(tx)
	if e != nil {
		return e
	}
	expiry := now.Add(time.Duration(l.DurationSeconds) * time.Second)
	row.ActivatedAt = &now
	row.ExpiresAt = &expiry
	row.MoneyReceipt = receipt
	return tx.Save(&row).Error
}
func (r *Repository) ReadSummary(ctx context.Context, org string) (s storagecapacity.Summary, err error) {
	if !storagecapacity.ValidID(org, 128) {
		return s, storagecapacity.ErrInvalid
	}
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var account accountRow
		e := tx.First(&account, "organization_id = ?", org).Error
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		// A new metered namespace has no account until its first write. Reads
		// derive the permanent base allowance and zero counters without inserting.
		a := &account
		now, e := r.clock(tx)
		if e != nil {
			return e
		}
		q, e := quota(tx, org, now)
		if e != nil {
			return e
		}
		remaining := q - a.Occupied - a.Reserved
		if remaining < 0 {
			remaining = 0
		}
		overage := a.Occupied - q
		if overage < 0 {
			overage = 0
		}
		s = storagecapacity.Summary{BaseBytes: strconv.FormatInt(storagecapacity.BaseBytes, 10), QuotaBytes: strconv.FormatInt(q, 10), OccupiedBytes: strconv.FormatInt(a.Occupied, 10), ReservedBytes: strconv.FormatInt(a.Reserved, 10), RemainingBytes: strconv.FormatInt(remaining, 10), OverageBytes: strconv.FormatInt(overage, 10), ObservedAt: now, Categories: map[string]string{}, ActiveLots: []storagecapacity.Lot{}}
		var categories []struct {
			Category string
			Bytes    int64
		}
		if e = tx.Model(&objectRow{}).Select("category, SUM(size_bytes) AS bytes").Where("organization_id = ? AND stored = ?", org, true).Group("category").Scan(&categories).Error; e != nil {
			return e
		}
		for _, c := range categories {
			s.Categories[c.Category] = strconv.FormatInt(c.Bytes, 10)
		}
		var lots []lotRow
		if e = tx.Where("organization_id = ? AND activated_at IS NOT NULL AND expires_at > ?", org, now).Order("expires_at, order_id").Limit(51).Find(&lots).Error; e != nil {
			return e
		}
		if len(lots) > 50 {
			s.MoreLots = true
			lots = lots[:50]
		}
		for _, l := range lots {
			s.ActiveLots = append(s.ActiveLots, storagecapacity.Lot{OrderID: l.OrderID, Bytes: strconv.FormatInt(l.Bytes, 10), ActivatedAt: *l.ActivatedAt, ExpiresAt: *l.ExpiresAt})
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return
}
