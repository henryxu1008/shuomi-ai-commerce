package alibaba1688

import (
	"context"
	"errors"
)

const (
	AccountProfileUnavailable = "source_account_unavailable"
	AccountProfileDisabled    = "source_account_disabled"
)

// AccountProfile is the safe browser runtime configuration for one 1688 login account.
type AccountProfile struct {
	ID          int64
	TenantID    int64
	Label       string
	ProfileDir  string
	ProxyServer string
	LoginURL    string
}

// AccountProfileResolver resolves a tenant-owned 1688 login account.
type AccountProfileResolver interface {
	ResolveAlibaba1688Account(context.Context, int64, int64) (AccountProfile, error)
}

type accountProfileError struct {
	code    string
	message string
}

func (e *accountProfileError) Error() string {
	if e == nil {
		return ""
	}
	return e.message
}

func newAccountProfileError(code, message string) error {
	return &accountProfileError{code: code, message: message}
}

// AccountProfileErrorCode returns a stable account-profile resolution error code.
func AccountProfileErrorCode(err error) string {
	var profileErr *accountProfileError
	if errors.As(err, &profileErr) && profileErr != nil {
		return profileErr.code
	}
	return sourceAccessErrorCode(err)
}
