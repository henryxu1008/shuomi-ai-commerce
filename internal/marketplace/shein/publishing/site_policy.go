package publishing

import (
	"strings"
	"task-processor/internal/marketplace/shein/offlinepackage/previewdata"
)

// DefaultSiteListByRegion returns a fresh SHEIN site list for the configured region.
func DefaultSiteListByRegion(region string) []previewdata.SiteInfo {
	subSite := "shein-us"
	switch region {
	case "US", "FR", "DE", "IT", "ES", "UK", "AU", "JP", "MX", "SA", "AE", "CA":
		subSite = "shein-" + strings.ToLower(region)
	}
	return []previewdata.SiteInfo{{MainSite: "shein", SubSiteList: []string{subSite}}}
}
