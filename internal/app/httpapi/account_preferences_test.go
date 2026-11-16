package httpapi

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http/httptest"
	"strings"
	"task-processor/internal/authidentity"
	"task-processor/internal/geography"
	store "task-processor/internal/integration/persistence/accountprofile"
	"testing"
)

func TestAccountPreferencesRequireCompleteSelfScopedInput(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE account_user_preferences(user_id text PRIMARY KEY,country text,province text,city text,updated_at datetime)`).Error)
	repo, err := store.New(db)
	require.NoError(t, err)
	m := accountProfileModule{repository: repo}
	request := func(method, subject, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		r := httptest.NewRequest(method, "/api/v1/account/preferences", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		c.Request = r.WithContext(authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{UserID: subject}))
		m.preferences(c)
		return w
	}
	require.Equal(t, 400, request("PUT", "self", `{"country":"中国","province":"浙江","city":"杭州"}`).Code, "old free text must not reach storage")
	_, err = repo.SavePreferences(context.Background(), store.UserPreferences{UserID: "self", Country: "旧地区", Province: "旧值", City: "旧城市"})
	require.NoError(t, err)
	catalog, err := geography.Bundled()
	require.NoError(t, err)
	input := func(country, province, city string) string {
		return fmt.Sprintf(`{"catalogVersion":%q,"country":%q,"province":%q,"city":%q}`, catalog.Version(), country, province, city)
	}
	for _, body := range []string{`{"country":"other","country":"China","province":"","city":""}`, `{"country":null,"province":"","city":""}`, `{"country":"China"}`, `{"country":"China","province":"","city":"","userId":"foreign"}`} {
		require.Equal(t, 400, request("PUT", "self", body).Code, body)
	}
	for _, body := range []string{
		input("US", "5332921", "1880252"),
		input("US", "浙江", "杭州"),
		input("SG", "direct", "1880252"),
		strings.Replace(input("SG", "", "1880252"), catalog.Version(), "gn-old", 1),
	} {
		require.Equal(t, 400, request("PUT", "self", body).Code, body)
	}
	require.Contains(t, request("GET", "self", "").Body.String(), "旧城市")
	require.Equal(t, 200, request("PUT", "self", input("SG", "", "1880252")).Code)
	require.Contains(t, request("GET", "self", "").Body.String(), "Singapore")
	require.NotContains(t, request("GET", "foreign", "").Body.String(), "Singapore")
	require.Equal(t, 200, request("PUT", "self", input("", "", "")).Code)
	require.NotContains(t, request("GET", "self", "").Body.String(), "Singapore")
}

func TestAccountRegionCatalogReadScope(t *testing.T) {
	request := func(query, subject string) *httptest.ResponseRecorder {
		writer := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(writer)
		r := httptest.NewRequest("GET", accountRegionCatalogPath+query, nil)
		c.Request = r.WithContext(authidentity.WithAuthenticatedIdentity(r.Context(), authidentity.AuthenticatedIdentity{UserID: subject}))
		accountProfileModule{}.regionCatalog(c)
		return writer
	}
	require.Equal(t, 401, request("", "").Code)
	require.Equal(t, 400, request("?country=SG&country=CN", "self").Code)
	require.Equal(t, 400, request("?unknown=SG", "self").Code)
	index := request("", "self")
	require.Equal(t, 200, index.Code)
	require.Contains(t, index.Body.String(), `"countries"`)
	require.NotContains(t, index.Body.String(), `"cities"`)
	country := request("?country=SG", "self")
	require.Equal(t, 200, country.Code)
	require.Contains(t, country.Body.String(), `"provinces":[]`)
	require.Contains(t, country.Body.String(), `"1880252"`)
	require.Equal(t, 400, request("?country=SG&view=shipping", "self").Code)
	require.Equal(t, 400, request("?country=CN&view=shipping&view=shipping", "self").Code)
	require.Equal(t, 400, request("?country=CN&view=shipping&extra=1", "self").Code)
	shipping := request("?country=CN&view=shipping", "self")
	require.Equal(t, 200, shipping.Code)
	require.Contains(t, shipping.Body.String(), "浙江省")
	require.Contains(t, shipping.Body.String(), "杭州市")
	require.Contains(t, shipping.Body.String(), "仙桃市")
}
