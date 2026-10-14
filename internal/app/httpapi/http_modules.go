package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"task-processor/internal/httproute"
	kernelmodule "task-processor/internal/kernel/module"
)

func newCoreHTTPModule() httpModule {
	return httpModule{
		name: "system",
		register: func(reg *kernelmodule.Registry) error {
			reg.AddRoutes(httproute.Descriptor{
				Method: http.MethodGet,
				Path:   "/health",
				Module: "system",
				Handler: func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"status": "ok"})
				},
			})
			return nil
		},
	}
}
