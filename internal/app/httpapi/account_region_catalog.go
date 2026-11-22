package httpapi

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/geography"
)

const accountRegionCatalogPath = "/api/v1/account/region-catalog"

func (m accountProfileModule) regionCatalog(c *gin.Context) {
	identity, ok := authidentity.AuthenticatedIdentityFromContext(c.Request.Context())
	if !ok || !authidentity.IsBoundedIdentifier(identity.UserID) {
		writeAccountProfileError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED")
		return
	}
	if c.Request.URL.ForceQuery || len(c.Request.URL.RawQuery) > 32 ||
		c.Request.ContentLength != 0 || len(c.Request.TransferEncoding) != 0 {
		writeAccountProfileError(c, http.StatusBadRequest, "INVALID_REQUEST")
		return
	}
	query, err := url.ParseQuery(c.Request.URL.RawQuery)
	country := query.Get("country")
	countryOnly := len(query) == 1 && len(query["country"]) == 1 &&
		len(country) == 2 && strings.Trim(country, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") == ""
	shipping := len(query) == 2 && len(query["country"]) == 1 && country == "CN" &&
		len(query["view"]) == 1 && query.Get("view") == "shipping"
	if err != nil || (c.Request.URL.RawQuery != "" && !countryOnly && !shipping) {
		writeAccountProfileError(c, http.StatusBadRequest, "INVALID_REQUEST")
		return
	}
	catalog, err := geography.Bundled()
	if err != nil {
		writeAccountProfileError(c, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE")
		return
	}
	c.Header("Cache-Control", "private, no-store")
	if country == "" {
		c.JSON(http.StatusOK, gin.H{"version": catalog.Version(), "countries": catalog.Countries()})
		return
	}
	detail := catalog.Country(country)
	if detail == nil {
		writeAccountProfileError(c, http.StatusBadRequest, "INVALID_REQUEST")
		return
	}
	c.JSON(http.StatusOK, gin.H{"version": catalog.Version(), "country": detail})
}
