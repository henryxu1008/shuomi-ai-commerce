package listinghttp

import (
	"context"
	"net/http"
	listingapp "task-processor/internal/app/listing"
	"task-processor/internal/httproute"
)

func IsControlledLocalDraftRoute(method, path string, enabled bool) bool {
	if method == http.MethodGet {
		return path == ListingBasePath+"/sources/:source_id/targets/:store_id" || path == ListingBasePath+"/records/:record_id" || path == ListingBasePath+"/target-commands/:key"
	}
	if !enabled || method != http.MethodPost {
		return false
	}
	return path == ListingBasePath+"/target-rules" || path == ListingBasePath+"/shein-completion-preview" || path == ListingBasePath+"/shein-sales-mapping-preview" || path == ListingBasePath+"/targets"
}
func ControlledLocalDraftRoutes(app *listingapp.Application, bind func(context.Context, string) (context.Context, error), enabled bool) []httproute.Descriptor {
	var out []httproute.Descriptor
	for _, r := range ListingRoutes(app, bind) {
		if IsControlledLocalDraftRoute(r.Method, r.Path, enabled) {
			out = append(out, r)
		}
	}
	return out
}
