package googlemerchant

import (
	"context"
	"net/http"
	"strconv"

	domain "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
	sourceapi "google.golang.org/api/merchantapi/datasources/v1"
)

const pageSize = 100

// ListAccounts returns accounts visible to the merchant's OAuth grant.
func (c *Client) ListAccounts(ctx context.Context, token, page string) (domain.AccountPage, error) {
	if c == nil || c.accounts == nil || ctx == nil || !validToken(token) || len(page) > 4096 {
		return domain.AccountPage{}, domain.ErrInvalid
	}
	call := c.accounts.Accounts.List().PageSize(pageSize).PageToken(page).Context(ctx)
	call.Header().Set("Authorization", "Bearer "+token)
	response, err := call.Do()
	if err != nil {
		return domain.AccountPage{}, readError(err)
	}
	result := domain.AccountPage{NextPageToken: response.NextPageToken}
	for _, row := range response.Accounts {
		if row != nil && row.AccountId > 0 && row.Name == "accounts/"+strconv.FormatInt(row.AccountId, 10) {
			result.Accounts = append(result.Accounts, domain.Account{
				ID: strconv.FormatInt(row.AccountId, 10), DisplayName: row.AccountName, TestAccount: row.TestAccount,
			})
		}
	}
	return result, nil
}

// ListSources offers primary online API sources. Source ownership confirmation is
// required in Store Center even if its current platform settings are usable.
func (c *Client) ListSources(ctx context.Context, token, accountID, page string) (domain.SourcePage, error) {
	if c == nil || c.sources == nil || ctx == nil || !validToken(token) || !domain.ValidResourceID(accountID) || len(page) > 4096 {
		return domain.SourcePage{}, domain.ErrInvalid
	}
	call := c.sources.Accounts.DataSources.List("accounts/" + accountID).PageSize(pageSize).PageToken(page).Context(ctx)
	call.Header().Set("Authorization", "Bearer "+token)
	response, err := call.Do()
	if err != nil {
		return domain.SourcePage{}, readError(err)
	}
	result := domain.SourcePage{NextPageToken: response.NextPageToken}
	for _, row := range response.DataSources {
		if !isPrimaryAPI(row, accountID) {
			continue
		}
		primary := row.PrimaryProductDataSource
		result.Sources = append(result.Sources, domain.DataSource{
			ID: strconv.FormatInt(row.DataSourceId, 10), DisplayName: row.DisplayName,
			ContentLanguage: primary.ContentLanguage, FeedLabel: primary.FeedLabel,
		})
	}
	return result, nil
}

func isPrimaryAPI(source *sourceapi.DataSource, accountID string) bool {
	return source != nil && source.Input == "API" && source.PrimaryProductDataSource != nil &&
		!source.PrimaryProductDataSource.LegacyLocal && source.DataSourceId > 0 &&
		source.Name == "accounts/"+accountID+"/dataSources/"+strconv.FormatInt(source.DataSourceId, 10)
}

func (c *Client) VerifySource(ctx context.Context, token string, target domain.Target) error {
	if c == nil || c.sources == nil || ctx == nil || !validToken(token) || !target.Valid() {
		return domain.ErrInvalid
	}
	name := dataSource(target)
	call := c.sources.Accounts.DataSources.Get(name).Context(ctx)
	call.Header().Set("Authorization", "Bearer "+token)
	source, err := call.Do()
	if err != nil {
		if responseCode(err) == http.StatusNotFound {
			return domain.ErrSourceMismatch
		}
		return readError(err)
	}
	if !isPrimaryAPI(source, target.AccountID) || source.Name != name {
		return domain.ErrSourceMismatch
	}
	primary := source.PrimaryProductDataSource
	if (primary.ContentLanguage != "" || primary.FeedLabel != "") &&
		(primary.ContentLanguage != target.ContentLanguage || primary.FeedLabel != target.FeedLabel) {
		return domain.ErrSourceMismatch
	}
	freeEnabled := false
	for _, destination := range primary.Destinations {
		if destination == nil || destination.State == "DISABLED" {
			continue
		}
		if destination.Destination != freeListings || destination.State != "ENABLED" {
			return domain.ErrSourceMismatch
		}
		freeEnabled = true
	}
	if !freeEnabled {
		return domain.ErrSourceMismatch
	}
	return nil
}

func readError(err error) error {
	switch responseCode(err) {
	case http.StatusUnauthorized, http.StatusForbidden:
		return domain.ErrUnauthorized
	default:
		return domain.ErrUnavailable
	}
}
