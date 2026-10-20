package publishing

import (
	"strings"

	sheinproduct "task-processor/internal/shein/api/product"
)

// DefaultSiteListByRegion returns a fresh SHEIN site list for the configured region.
func DefaultSiteListByRegion(region string) []sheinproduct.SiteInfo {
	subSite := "shein-us"
	switch region {
	case "US", "FR", "DE", "IT", "ES", "UK", "AU", "JP", "MX", "SA", "AE", "CA":
		subSite = "shein-" + strings.ToLower(region)
	}
	return []sheinproduct.SiteInfo{{MainSite: "shein", SubSiteList: []string{subSite}}}
}
