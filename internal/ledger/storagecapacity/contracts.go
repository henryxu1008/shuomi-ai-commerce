// Package storagecapacity owns enterprise capacity admission and retained-byte
// accounting. Business owners retain ownership of their objects and lifecycles.
package storagecapacity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

const (
	GiB           int64 = 1 << 30
	BaseBytes           = 5 * GiB
	PackBytes           = 50 * GiB
	PeriodSeconds int64 = 30 * 24 * 60 * 60
)

var (
	ErrInvalid     = errors.New("storage capacity input is invalid")
	ErrExceeded    = errors.New("enterprise storage capacity exceeded")
	ErrConflict    = errors.New("storage capacity identity conflict")
	ErrUnknown     = errors.New("storage write result is unknown")
	ErrNotFound    = errors.New("storage capacity identity not found")
	ErrUnavailable = errors.New("storage capacity is unavailable")
)

func Digest(s string) string         { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func ValidID(s string, max int) bool { return s != "" && len(s) <= max && s == strings.TrimSpace(s) }
func ValidDigest(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 32 && strings.ToLower(s) == s
}

type WriteIntent struct {
	OrganizationID string
	Owner          string
	OperationID    string
	Fingerprint    string
	MaxBytes       int64
	MaxObjects     int
}

func (i WriteIntent) Validate() error {
	if !ValidID(i.OrganizationID, 128) || !ValidID(i.Owner, 64) || !ValidID(i.OperationID, 256) || !ValidDigest(i.Fingerprint) || i.MaxBytes < 1 || i.MaxBytes > 10*PackBytes || i.MaxObjects < 1 || i.MaxObjects > 16 {
		return ErrInvalid
	}
	return nil
}

type Reservation struct {
	ID     string
	Intent WriteIntent
	State  string
}
type Object struct {
	Namespace string
	Key       string
	SHA256    string
	SizeBytes int64
	Category  string
}

func (o Object) Validate() error {
	if !ValidID(o.Namespace, 256) || !ValidID(o.Key, 1024) || !ValidDigest(o.SHA256) || o.SizeBytes < 1 {
		return ErrInvalid
	}
	switch o.Category {
	case "product_media", "ai_image", "knowledge_document", "business_attachment":
		return nil
	}
	return ErrInvalid
}

type PurchaseLot struct {
	OrganizationID, OrderID, ItemID, TermsFingerprint string
	Bytes, DurationSeconds                            int64
}

func (l PurchaseLot) Validate() error {
	if !ValidID(l.OrganizationID, 128) || !ValidID(l.OrderID, 128) || !ValidID(l.ItemID, 128) || !ValidDigest(l.TermsFingerprint) || l.Bytes < PackBytes || l.Bytes > 10*PackBytes || l.Bytes%PackBytes != 0 || l.DurationSeconds != PeriodSeconds {
		return ErrInvalid
	}
	return nil
}

type Lot struct {
	OrderID     string    `json:"order_id"`
	Bytes       string    `json:"bytes"`
	ActivatedAt time.Time `json:"activated_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}
type Summary struct {
	MoreLots       bool              `json:"more_lots"`
	BaseBytes      string            `json:"base_bytes"`
	QuotaBytes     string            `json:"quota_bytes"`
	OccupiedBytes  string            `json:"occupied_bytes"`
	ReservedBytes  string            `json:"reserved_bytes"`
	RemainingBytes string            `json:"remaining_bytes"`
	OverageBytes   string            `json:"overage_bytes"`
	Categories     map[string]string `json:"categories"`
	ActiveLots     []Lot             `json:"active_lots"`
	ObservedAt     time.Time         `json:"observed_at"`
}

// CloseProof is supplied only by the original native owner after it has durably
// fenced all further output. A timeout, missing object or failed task is no proof.
type CloseProof struct{ Owner, OperationID, Fingerprint, TerminalReceipt string }
type Writer interface {
	FindObjectWrite(context.Context, string, string, Object) (Reservation, error)
	FindWrite(context.Context, string, string, string) (Reservation, error)
	ReserveWrite(context.Context, WriteIntent) (Reservation, error)
	BindObjects(context.Context, string, string, []Object) error
	ClaimDispatch(context.Context, string, string) error
	AbortUnstarted(context.Context, string, string) error
	ConfirmStored(context.Context, string, string, string, string, string, int64) error
	CloseWrite(context.Context, string, string, CloseProof) error
}
type Reader interface {
	ReadSummary(context.Context, string) (Summary, error)
}
