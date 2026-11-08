package httpapi

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http/httptest"
	"strings"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/commercial/billing"
	store "task-processor/internal/integration/persistence/commercialbilling"
	"task-processor/internal/ledger/orgresource"
	"testing"
	"time"
)

func TestDataRowPricingRequiresPlatformAuthorityAndPreservesOriginalReceipt(t *testing.T) {
	db, e := gorm.Open(sqlite.Open("file:price-http?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, e)
	require.NoError(t, store.AutoMigrate(db))
	repo, e := store.New(db)
	require.NoError(t, e)
	require.NoError(t, repo.CreateOffer(context.Background(), billing.Offer{OfferID: "initial", ProductKind: billing.ProductDataRow, ResourceType: orgresource.ResourceDataRow, Currency: "CNY", UnitPriceMinor: 5, PricingVersion: "initial", MinQuantity: 1, MaxQuantity: 100000, Status: billing.OfferActive}))
	auth, e := authz.NewListingKitAuthorizer([]string{"maintainer"}, nil)
	require.NoError(t, e)
	id := authidentity.AuthenticatedIdentity{UserID: "maintainer", Roles: []string{"listingkit_admin"}, TenantID: "org-a", EffectiveOrganizationID: "org-a", EffectiveMemberID: "member", TokenExpiresAt: time.Now().Add(time.Hour)}
	m := dataRowPricingModule{store: repo, authorizer: auth}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(authidentity.WithAuthenticatedIdentity(c.Request.Context(), id))
	})
	for _, r := range m.routes() {
		router.Handle(r.Method, r.Path, r.Handler)
	}
	send := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, dataRowPriceBase+path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "original-key")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	body := `{"expected_version":"initial","unit_price_minor":"0","reason":"isolated debugging"}`
	id.UserID = "enterprise-admin"
	require.Equal(t, 403, send("POST", "/publish", body).Code)
	id.UserID = "maintainer"
	require.Equal(t, 400, send("POST", "/publish", strings.TrimSuffix(body, "}")+`,"organization_id":"victim"}`).Code)
	for _, invalid := range []string{
		`{"expected_version":"initial","reason":"missing"}`,
		`{"expected_version":"initial","unit_price_minor":"-1","reason":"negative"}`,
		`{"expected_version":"initial","unit_price_minor":"9223372036854775808","reason":"overflow"}`,
		`{"expected_version":"initial","unit_price_minor":"0","unit_price_minor":"1","reason":"duplicate"}`,
	} {
		require.Equal(t, 400, send("POST", "/publish", invalid).Code)
	}
	w := send("POST", "/publish", body)
	require.Equal(t, 201, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), `"unit_price_minor":"0"`)
	require.Equal(t, 200, send("GET", "/receipts/original-key", "").Code)
	require.Equal(t, 409, send("POST", "/publish", strings.Replace(body, `"0"`, `"7"`, 1)).Code)
	id.UserID = "enterprise-admin"
	require.Equal(t, 403, send("GET", "/receipts/original-key", "").Code)
}
