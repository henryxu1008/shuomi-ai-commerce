package googlemerchant

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	ErrInvalid        = errors.New("invalid Google Merchant product or target")
	ErrUnauthorized   = errors.New("Google Merchant authorization rejected")
	ErrUnavailable    = errors.New("Google Merchant read unavailable")
	ErrOfferExists    = errors.New("Google Merchant offer already exists")
	ErrSourceMismatch = errors.New("Google Merchant offer belongs to a different data source")
	ErrNotObserved    = errors.New("Google Merchant processed product not yet observed")
	ErrRejected       = errors.New("Google Merchant rejected the product input")
	ErrOutcomeUnknown = errors.New("Google Merchant product submission outcome unknown")
)

var (
	numericID = regexp.MustCompile(`^[1-9][0-9]{0,18}$`)
	language  = regexp.MustCompile(`^[a-z]{2}$`)
	feedLabel = regexp.MustCompile(`^[A-Z0-9_-]{1,20}$`)
	prefix    = regexp.MustCompile(`^sm-[a-f0-9]{20}-$`)
	offerPart = regexp.MustCompile(`^[a-f0-9]{24}$`)
	currency  = regexp.MustCompile(`^[A-Z]{3}$`)
)

// Target is derived from the current, verified Store Center binding. OfferPrefix
// is persistently reserved for that account, independent of its grant revision.
type Target struct {
	AccountID, DataSourceID, ContentLanguage, FeedLabel, OfferPrefix string
}

func (t Target) Valid() bool {
	return numericID.MatchString(t.AccountID) && numericID.MatchString(t.DataSourceID) &&
		language.MatchString(t.ContentLanguage) && feedLabel.MatchString(t.FeedLabel) && prefix.MatchString(t.OfferPrefix)
}

func (t Target) OwnsOfferID(id string) bool {
	suffix, found := strings.CutPrefix(id, t.OfferPrefix)
	return t.Valid() && found && offerPart.MatchString(suffix)
}

// OfferID derives an immutable identity from an already qualified product variant.
// The caller uses the scoped product and variant identity, not display labels.
func (t Target) OfferID(variantIdentity string) (string, error) {
	if !t.Valid() || strings.TrimSpace(variantIdentity) == "" || len(variantIdentity) > 512 {
		return "", ErrInvalid
	}
	digest := sha256.Sum256([]byte(variantIdentity))
	return t.OfferPrefix + hex.EncodeToString(digest[:12]), nil
}

type Offer struct {
	ID, Title, Description, Link, ImageLink string
	Currency, Availability, Condition       string
	AmountMicros                            int64
	Brand, MPN, ItemGroupID, Color, Size    string
	GTINs                                   []string
}

func (o Offer) ValidFor(t Target) bool {
	if !t.OwnsOfferID(o.ID) || !validText(o.Title, 150) || !validText(o.Description, 5000) ||
		!validURL(o.Link) || !validURL(o.ImageLink) || o.AmountMicros <= 0 || !currency.MatchString(o.Currency) {
		return false
	}
	switch o.Availability {
	case "IN_STOCK", "OUT_OF_STOCK":
	default:
		return false
	}
	switch o.Condition {
	case "NEW", "REFURBISHED", "USED":
		return true
	default:
		return false
	}
}

func validText(value string, max int) bool {
	return strings.TrimSpace(value) != "" && utf8.ValidString(value) && utf8.RuneCountInString(value) <= max
}

func validURL(value string) bool {
	u, err := url.Parse(value)
	return len(value) <= 2000 && err == nil && u.Hostname() != "" && u.User == nil &&
		(u.Scheme == "https" || u.Scheme == "http")
}

type Receipt struct {
	Input, Product string
}

type ListingCountries struct {
	Approved, Pending, Disapproved []string
}

type ProductIssue struct {
	Code, Attribute, Severity string
}

type Observation struct {
	Product, DataSource   string
	FreeListings          ListingCountries
	UnexpectedDestination bool
	Issues                []ProductIssue
}

func ValidResourceID(id string) bool {
	return numericID.MatchString(id)
}

type Account struct {
	ID, DisplayName string
	TestAccount     bool
}

type AccountPage struct {
	Accounts      []Account
	NextPageToken string
}

type DataSource struct {
	ID, DisplayName, ContentLanguage, FeedLabel string
}

type SourcePage struct {
	Sources       []DataSource
	NextPageToken string
}
