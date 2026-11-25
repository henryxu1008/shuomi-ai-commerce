package googlemerchant

import (
	"errors"
	"time"
)

// These outcomes contain no provider response or credentials.
var ErrRefreshNotSent = errors.New("Google Merchant renewal request was not sent")

type RefreshRetry struct {
	After time.Duration
}

func (RefreshRetry) Error() string { return "Google Merchant renewal was declined before exchange" }
