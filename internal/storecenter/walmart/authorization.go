// Package walmart defines the Store-owned US OAuth port. Credentials never
// enter public DTOs, workflow history or Listing persistence.
package walmart

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrInvalid = errors.New("invalid Walmart authorization material")

type Application struct{ ClientID, Revision, Environment, CallbackURL string }
type TokenSet struct {
	AccessToken, RefreshToken         string `json:"-"`
	SellerID                          string
	AccessExpiresAt, RefreshExpiresAt time.Time
}

func (TokenSet) String() string   { return "Walmart credential [redacted]" }
func (TokenSet) GoString() string { return "Walmart credential [redacted]" }

type TokenDetail struct {
	Valid     bool
	ExpiresAt time.Time
	Scopes    map[string]string
}

func (d TokenDetail) Allows(purpose string, now time.Time) bool {
	if !d.Valid || !now.Before(d.ExpiresAt) {
		return false
	}
	level := d.Scopes["item"]
	if purpose == "product_publish" {
		return level == "full_access"
	}
	if purpose == "feed_result" {
		level = d.Scopes["feeds"]
	}
	return level == "full_access" || level == "view_only"
}

type Merchant struct {
	PartnerID string `json:"partnerId"`
	Name      string `json:"name"`
}
type Authorization interface {
	Application() Application
	AuthorizationURL(state, nonce string) (string, error)
	Exchange(context.Context, string, string) (TokenSet, error)
	Refresh(context.Context, TokenSet) (TokenSet, error)
	TokenDetail(context.Context, TokenSet) (TokenDetail, error)
	Merchant(context.Context, TokenSet) (Merchant, error)
}
type CredentialBinding struct {
	OrganizationID, AppID, AppRevision, Environment, ID, Purpose string
	Generation                                                   int64
}

func (b CredentialBinding) AssociatedData() ([]byte, error) {
	if b.OrganizationID == "" || b.AppID == "" || b.AppRevision == "" || b.ID == "" || b.Environment != "production" && b.Environment != "sandbox" || b.Purpose != "attempt" && b.Purpose != "grant" || b.Purpose == "grant" && b.Generation < 1 || b.Purpose == "attempt" && b.Generation != 0 {
		return nil, ErrInvalid
	}
	return json.Marshal([]any{"task-processor/walmart-us/credential/v1", b.Environment, b.OrganizationID, b.AppID, b.AppRevision, b.Purpose, b.ID, b.Generation})
}

type CredentialProtection interface {
	Seal(CredentialBinding, TokenSet) (string, string, error)
	Open(CredentialBinding, string, string) (TokenSet, error)
}
