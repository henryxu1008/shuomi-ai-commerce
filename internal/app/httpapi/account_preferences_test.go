package httpapi

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/geography"
	store "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/accountprofile"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http/httptest"
	"strings"
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
	_, err = repo.SavePreferences(context.Background(), store.UserPreferences{UserID: "self", Country: "China", Province: "浙江", City: "乌镇"})
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
		input("CN", "1809935", "1795565"),
		input("CN", "440000", "330100"),
		strings.Replace(input("SG", "", "1880252"), catalog.Version(), "gn-old", 1),
	} {
		require.Equal(t, 400, request("PUT", "self", body).Code, body)
	}
	stored := request("GET", "self", "").Body.String()
	require.Contains(t, stored, `"province":"浙江"`)
	require.Contains(t, stored, `"city":"乌镇"`)
	require.Equal(t, 400, request("PUT", "self", strings.Replace(input("CN", "330000", "330100"), catalog.Version(), "gn-e6eec5af36e9aae5", 1)).Code)
	require.Contains(t, request("GET", "self", "").Body.String(), "乌镇")
	require.Equal(t, 200, request("PUT", "self", input("CN", "330000", "330100")).Code)
	stored = request("GET", "self", "").Body.String()
	require.Contains(t, stored, `"province":"浙江省"`)
	require.Contains(t, stored, `"city":"杭州市"`)
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
	china := request("?country=CN", "self")
	require.Equal(t, 200, china.Code)
	require.True(t, china.Body.String() == shipping.Body.String(), "China catalog query forms must return the same response")
	require.Contains(t, shipping.Body.String(), "浙江省")
	require.Contains(t, shipping.Body.String(), "杭州市")
	require.Contains(t, shipping.Body.String(), "仙桃市")
}
