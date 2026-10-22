package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type stubSDSCatalogRouteHandler struct{}

func (stubSDSCatalogRouteHandler) ListSDSProducts(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": []any{}})
}

func (stubSDSCatalogRouteHandler) GetSDSProduct(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"id": c.Param("product_id")})
}

func (stubSDSCatalogRouteHandler) ListSDSCategories(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": []any{}})
}

func (stubSDSCatalogRouteHandler) ListSDSShipmentAreas(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": []any{}})
}
