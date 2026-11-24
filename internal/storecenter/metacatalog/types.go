package metacatalog

import (
	"context"
	"errors"
	"regexp"
	"time"
)

var (
	ErrUnavailable  = errors.New("meta catalog is unavailable")
	ErrRejected     = errors.New("meta catalog declined this product")
	ErrUnauthorized = errors.New("meta catalog authorization is unavailable")
	validID         = regexp.MustCompile("^[1-9][0-9]{0,25}$")
)

func ValidID(id string) bool { return validID.MatchString(id) }

type Product struct {
	RetailerID  string `json:"retailerId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ProductURL  string `json:"productUrl"`
	ImageURL    string `json:"imageUrl"`
	Brand       string `json:"brand"`
	PriceCents  int64  `json:"priceCents"`
}

type Choice struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	BusinessID string `json:"businessId"`
}

type Credentials struct {
	Token     string
	Subject   string
	WhatsApp  bool
	ExpiresAt time.Time
}

type Provider interface {
	ConsentURL(callback, state string) (string, error)
	Exchange(ctx context.Context, callback, code string) (Credentials, error)
	Choices(context.Context, Credentials) ([]Choice, []Choice, error)
	Create(context.Context, string, string, Product) (string, error)
	ReadProduct(context.Context, string, string, string, string) (bool, error)
	WhatsAppLinked(context.Context, string, string, string) (bool, error)
}
