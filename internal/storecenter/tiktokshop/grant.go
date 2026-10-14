package tiktokshop

import (
	"encoding/hex"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"

	"task-processor/internal/authidentity"
)

var ErrGrantConflict = errors.New("TikTok Shop grant revision or operation conflict")

type GrantStatus string

const (
	GrantActive            GrantStatus = "active"
	GrantRefreshDispatched GrantStatus = "refresh_dispatched"
	GrantRefreshUnknown    GrantStatus = "refresh_unknown"
	GrantRevoked           GrantStatus = "revoked"
	GrantExpired           GrantStatus = "expired"
)

// GrantSnapshot is a trusted repository rehydration boundary, not an HTTP DTO.
// A single Store transaction must CAS the old revision and persist the changed
// grant together with the original operation/attempt receipt.
type GrantSnapshot struct {
	OrganizationID, ID, AppKey, AppRevision, SellerIdentityHash string
	Generation, Revision                                        int64
	Status                                                      GrantStatus
	KeyID, Ciphertext                                           string `json:"-"`
	AccessExpiresAt, RefreshExpiresAt                           time.Time
	OperationID                                                 string
	ClaimExpiresAt                                              time.Time
	CreatedAt, UpdatedAt                                        time.Time
}

func (GrantSnapshot) String() string   { return "TikTok Shop grant [protected]" }
func (GrantSnapshot) GoString() string { return "TikTok Shop grant [protected]" }

type Grant struct{ snapshot GrantSnapshot }
type GrantFence struct {
	OrganizationID, GrantID, OperationID string
	Generation, Revision                 int64
}

func RestoreGrant(snapshot GrantSnapshot) (*Grant, error) {
	hash, err := hex.DecodeString(snapshot.SellerIdentityHash)
	_, aadErr := (CredentialBinding{OrganizationID: snapshot.OrganizationID, AppKey: snapshot.AppKey, AppRevision: snapshot.AppRevision, ID: snapshot.ID, Purpose: "grant", Generation: snapshot.Generation}).AssociatedData()
	if err != nil || len(hash) != 32 || hex.EncodeToString(hash) != snapshot.SellerIdentityHash || aadErr != nil || snapshot.Revision < snapshot.Generation || !authidentity.IsBoundedIdentifier(snapshot.KeyID) || len(snapshot.Ciphertext) == 0 || len(snapshot.Ciphertext) > 16384 ||
		snapshot.AccessExpiresAt.IsZero() || snapshot.RefreshExpiresAt.IsZero() || snapshot.CreatedAt.IsZero() || snapshot.UpdatedAt.Before(snapshot.CreatedAt) {
		return nil, ErrInvalid
	}
	switch snapshot.Status {
	case GrantActive, GrantRevoked, GrantExpired:
		if snapshot.OperationID != "" || !snapshot.ClaimExpiresAt.IsZero() {
			return nil, ErrInvalid
		}
	case GrantRefreshDispatched, GrantRefreshUnknown:
		if uuid.Validate(snapshot.OperationID) != nil || !snapshot.ClaimExpiresAt.After(snapshot.CreatedAt) {
			return nil, ErrInvalid
		}
	default:
		return nil, ErrInvalid
	}
	return &Grant{snapshot: snapshot}, nil
}

func (g *Grant) Snapshot() GrantSnapshot { return g.snapshot }

// CanSend is only the token-state prerequisite. The consumer still needs live
// Store/member/service access and an exact fresh official Shop/scopes proof.
func (g *Grant) CanSend(now time.Time) bool {
	return g != nil && !now.IsZero() && g.snapshot.Status == GrantActive && now.Before(g.snapshot.AccessExpiresAt) && now.Before(g.snapshot.RefreshExpiresAt)
}

func (g *Grant) ClaimRefresh(expectedRevision int64, operationID string, now, expiresAt time.Time) (GrantFence, error) {
	if !g.canChange(expectedRevision, now) || g.snapshot.Status != GrantActive || !now.Before(g.snapshot.RefreshExpiresAt) || uuid.Validate(operationID) != nil || !expiresAt.After(now) || expiresAt.After(now.Add(5*time.Minute)) {
		return GrantFence{}, ErrGrantConflict
	}
	g.snapshot.Status, g.snapshot.OperationID, g.snapshot.ClaimExpiresAt = GrantRefreshDispatched, operationID, expiresAt.UTC()
	g.advance(now)
	return GrantFence{OrganizationID: g.snapshot.OrganizationID, GrantID: g.snapshot.ID, OperationID: operationID, Generation: g.snapshot.Generation, Revision: g.snapshot.Revision}, nil
}

func (g *Grant) CompleteRefresh(fence GrantFence, keyID, ciphertext string, accessExpiry, refreshExpiry, now time.Time) error {
	if !g.matches(fence, now) || !now.Before(g.snapshot.ClaimExpiresAt) {
		return ErrGrantConflict
	}
	return g.replace(keyID, ciphertext, accessExpiry, refreshExpiry, now)
}

func (g *Grant) MarkRefreshUnknown(fence GrantFence, now time.Time) error {
	if !g.matches(fence, now) {
		return ErrGrantConflict
	}
	g.snapshot.Status = GrantRefreshUnknown
	g.advance(now)
	return nil
}

// Reauthorize is called only after the original scoped OAuth attempt proves
// current consent and seller/Shop identity. It fences any earlier refresh.
func (g *Grant) Reauthorize(expectedRevision int64, keyID, ciphertext string, accessExpiry, refreshExpiry, now time.Time) error {
	if !g.canChange(expectedRevision, now) {
		return ErrGrantConflict
	}
	return g.replace(keyID, ciphertext, accessExpiry, refreshExpiry, now)
}

// Revoke concerns the seller grant. A single Shop lifecycle event must instead
// close that Shop binding and fence its access proof, not revoke other Shops.
func (g *Grant) Revoke(expectedRevision int64, now time.Time) error {
	if !g.canChange(expectedRevision, now) {
		return ErrGrantConflict
	}
	g.snapshot.Status, g.snapshot.OperationID, g.snapshot.ClaimExpiresAt = GrantRevoked, "", time.Time{}
	g.advance(now)
	return nil
}

func (g *Grant) FenceAccess(expectedRevision int64, now time.Time) error {
	if !g.canChange(expectedRevision, now) {
		return ErrGrantConflict
	}
	if g.snapshot.Status == GrantRefreshDispatched {
		g.snapshot.Status = GrantRefreshUnknown
	}
	g.advance(now)
	return nil
}

func (g *Grant) matches(fence GrantFence, now time.Time) bool {
	return g.canChange(fence.Revision, now) && g.snapshot.Status == GrantRefreshDispatched && fence.OrganizationID == g.snapshot.OrganizationID && fence.GrantID == g.snapshot.ID && fence.OperationID == g.snapshot.OperationID && fence.Generation == g.snapshot.Generation
}
func (g *Grant) canChange(expectedRevision int64, now time.Time) bool {
	return g != nil && expectedRevision == g.snapshot.Revision && g.snapshot.Revision < math.MaxInt64 && !now.IsZero() && !now.Before(g.snapshot.UpdatedAt)
}
func (g *Grant) advance(now time.Time) { g.snapshot.Revision++; g.snapshot.UpdatedAt = now.UTC() }
func (g *Grant) replace(keyID, ciphertext string, accessExpiry, refreshExpiry, now time.Time) error {
	if g.snapshot.Generation == math.MaxInt64 || !authidentity.IsBoundedIdentifier(keyID) || len(ciphertext) == 0 || len(ciphertext) > 16384 || !accessExpiry.After(now) || !refreshExpiry.After(now) {
		return ErrGrantConflict
	}
	g.snapshot.Generation++
	g.snapshot.KeyID, g.snapshot.Ciphertext = keyID, ciphertext
	g.snapshot.AccessExpiresAt, g.snapshot.RefreshExpiresAt = accessExpiry.UTC(), refreshExpiry.UTC()
	g.snapshot.Status, g.snapshot.OperationID, g.snapshot.ClaimExpiresAt = GrantActive, "", time.Time{}
	g.advance(now)
	return nil
}
