package googlemerchant

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	domain "github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/googlemerchant"
	"google.golang.org/api/googleapi"
	accountapi "google.golang.org/api/merchantapi/accounts/v1"
	sourceapi "google.golang.org/api/merchantapi/datasources/v1"
	merchant "google.golang.org/api/merchantapi/products/v1"
	"google.golang.org/api/option"
)

const (
	productTimeout = 10 * time.Second
	maxResponse    = 2 << 20
	freeListings   = "FREE_LISTINGS"
)

type Client struct {
	products *merchant.Service
	accounts *accountapi.APIService
	sources  *sourceapi.Service
}

func NewClient(httpClient *http.Client) (*Client, error) {
	return newClient(httpClient, "")
}

func newClient(httpClient *http.Client, endpoint string) (*Client, error) {
	client := boundedHTTPClient(httpClient)
	options := []option.ClientOption{option.WithHTTPClient(client), option.WithLogger(slog.New(slog.DiscardHandler))}
	if endpoint != "" {
		options = append(options, option.WithEndpoint(endpoint))
	}
	products, err := merchant.NewService(context.Background(), options...)
	if err != nil {
		return nil, domain.ErrUnavailable
	}
	accounts, err := accountapi.NewService(context.Background(), options...)
	if err != nil {
		return nil, domain.ErrUnavailable
	}
	sources, err := sourceapi.NewService(context.Background(), options...)
	if err != nil {
		return nil, domain.ErrUnavailable
	}
	return &Client{products: products, accounts: accounts, sources: sources}, nil
}

type boundedTransport struct {
	base http.RoundTripper
}

func (t boundedTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.Method == http.MethodPost {
		// The submission owner decides whether any further write is authorized.
		request = request.Clone(request.Context())
		request.GetBody = nil
	}
	response, err := t.base.RoundTrip(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponse+1))
	if err != nil || len(data) > maxResponse {
		return nil, domain.ErrUnavailable
	}
	response.Body = io.NopCloser(bytes.NewReader(data))
	return response, nil
}

// InsertNew makes one insert attempt. Its caller holds the durable send permit
// and supplies a current grant for the reserved source and offer ID namespace.
// A preflight 404 cannot make the upstream write conditional.
func (c *Client) InsertNew(ctx context.Context, token string, target domain.Target, offer domain.Offer) (domain.Receipt, error) {
	if ctx == nil || !offer.ValidFor(target) {
		return domain.Receipt{}, domain.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, productTimeout)
	defer cancel()
	if err := c.VerifySource(ctx, token, target); err != nil {
		return domain.Receipt{}, err
	}
	_, err := c.get(ctx, token, target, offer.ID)
	switch {
	case err == nil:
		return domain.Receipt{}, domain.ErrOfferExists
	case !errors.Is(err, domain.ErrNotObserved):
		return domain.Receipt{}, err
	}
	call := c.products.Accounts.ProductInputs.Insert(account(target), productInput(target, offer)).DataSource(dataSource(target)).Context(ctx)
	call.Header().Set("Authorization", "Bearer "+token)
	input, err := call.Do()
	if err != nil {
		return domain.Receipt{}, writeError(err)
	}
	if !matchesResource(input.Name, target, "productInputs", offer.ID) ||
		!matchesResource(input.Product, target, "products", offer.ID) {
		return domain.Receipt{}, domain.ErrOutcomeUnknown
	}
	return domain.Receipt{Input: input.Name, Product: input.Product}, nil
}

// ReadProduct returns an observation, which is not proof of a particular write.
func (c *Client) ReadProduct(ctx context.Context, token string, target domain.Target, offerID string) (domain.Observation, error) {
	product, err := c.get(ctx, token, target, offerID)
	if err != nil {
		return domain.Observation{}, err
	}
	if product.DataSource != dataSource(target) {
		return domain.Observation{}, domain.ErrSourceMismatch
	}
	result := domain.Observation{Product: product.Name, DataSource: product.DataSource}
	if product.ProductStatus == nil {
		return result, nil
	}
	for _, status := range product.ProductStatus.DestinationStatuses {
		if status == nil {
			continue
		}
		if status.ReportingContext != freeListings {
			result.UnexpectedDestination = true
			continue
		}
		result.FreeListings.Approved = append(result.FreeListings.Approved, status.ApprovedCountries...)
		result.FreeListings.Pending = append(result.FreeListings.Pending, status.PendingCountries...)
		result.FreeListings.Disapproved = append(result.FreeListings.Disapproved, status.DisapprovedCountries...)
	}
	for _, issue := range product.ProductStatus.ItemLevelIssues {
		if issue != nil {
			result.Issues = append(result.Issues, domain.ProductIssue{Code: issue.Code, Attribute: issue.Attribute, Severity: issue.Severity})
		}
	}
	return result, nil
}

func (c *Client) get(ctx context.Context, token string, target domain.Target, offerID string) (*merchant.Product, error) {
	if c == nil || c.products == nil || ctx == nil || !target.OwnsOfferID(offerID) ||
		!validToken(token) {
		return nil, domain.ErrInvalid
	}
	name := account(target) + "/products/" + encodedID(target, offerID)
	call := c.products.Accounts.Products.Get(name).Context(ctx)
	call.Header().Set("Authorization", "Bearer "+token)
	product, err := call.Do()
	if err != nil {
		switch responseCode(err) {
		case http.StatusNotFound:
			return nil, domain.ErrNotObserved
		case http.StatusUnauthorized, http.StatusForbidden:
			return nil, domain.ErrUnauthorized
		default:
			return nil, domain.ErrUnavailable
		}
	}
	if !matchesResource(product.Name, target, "products", offerID) {
		return nil, domain.ErrUnavailable
	}
	return product, nil
}

func account(target domain.Target) string {
	return "accounts/" + target.AccountID
}

func dataSource(target domain.Target) string {
	return account(target) + "/dataSources/" + target.DataSourceID
}

func encodedID(target domain.Target, offerID string) string {
	id := target.ContentLanguage + "~" + target.FeedLabel + "~" + offerID
	return base64.RawURLEncoding.EncodeToString([]byte(id))
}

func matchesResource(name string, target domain.Target, collection, offerID string) bool {
	prefix := account(target) + "/" + collection + "/"
	id := target.ContentLanguage + "~" + target.FeedLabel + "~" + offerID
	return name == prefix+id || name == prefix+encodedID(target, offerID)
}

func productInput(target domain.Target, offer domain.Offer) *merchant.ProductInput {
	return &merchant.ProductInput{
		ContentLanguage: target.ContentLanguage,
		FeedLabel:       target.FeedLabel,
		OfferId:         offer.ID,
		ProductAttributes: &merchant.ProductAttributes{
			Title: offer.Title, Description: offer.Description, Link: offer.Link, ImageLink: offer.ImageLink,
			Availability: offer.Availability, Condition: offer.Condition, Brand: offer.Brand, Mpn: offer.MPN,
			ItemGroupId: offer.ItemGroupID, Color: offer.Color, Size: offer.Size, Gtins: offer.GTINs,
			Price:                &merchant.Price{AmountMicros: offer.AmountMicros, CurrencyCode: offer.Currency},
			IncludedDestinations: []string{freeListings},
			ExcludedDestinations: []string{
				"SHOPPING_ADS", "DISPLAY_ADS", "LOCAL_INVENTORY_ADS", "FREE_LOCAL_LISTINGS", "YOUTUBE_SHOPPING",
				"YOUTUBE_SHOPPING_CHECKOUT", "YOUTUBE_AFFILIATE", "FREE_VEHICLE_LISTINGS", "VEHICLE_ADS",
				"CLOUD_RETAIL", "LOCAL_CLOUD_RETAIL", "RENTAL_ADS",
			},
		},
	}
}

func responseCode(err error) int {
	var upstream *googleapi.Error
	if errors.As(err, &upstream) {
		return upstream.Code
	}
	return 0
}

func writeError(err error) error {
	switch responseCode(err) {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return domain.ErrRejected
	case http.StatusForbidden, http.StatusUnauthorized:
		return domain.ErrUnauthorized
	default:
		return domain.ErrOutcomeUnknown
	}
}

func validToken(token string) bool {
	return token != "" && len(token) <= 8192 && !strings.ContainsAny(token, " \t\r\n\x00")
}

func boundedHTTPClient(httpClient *http.Client) *http.Client {
	client := http.Client{}
	if httpClient != nil {
		client = *httpClient
	}
	if client.Timeout <= 0 || client.Timeout > productTimeout {
		client.Timeout = productTimeout
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	client.Transport = boundedTransport{base: base}
	return &client
}
