package publishing

import "testing"

func TestDefaultSiteListByRegionPreservesSites(t *testing.T) {
	for _, tc := range []struct{ region, sub string }{
		{"US", "shein-us"}, {"FR", "shein-fr"}, {"DE", "shein-de"}, {"IT", "shein-it"}, {"ES", "shein-es"}, {"UK", "shein-uk"},
		{"AU", "shein-au"}, {"JP", "shein-jp"}, {"MX", "shein-mx"}, {"SA", "shein-sa"}, {"AE", "shein-ae"}, {"CA", "shein-ca"},
		{"UNKNOWN", "shein-us"}, {"", "shein-us"},
	} {
		t.Run(tc.region, func(t *testing.T) {
			got := DefaultSiteListByRegion(tc.region)
			if len(got) != 1 || got[0].MainSite != "shein" || len(got[0].SubSiteList) != 1 || got[0].SubSiteList[0] != tc.sub {
				t.Fatalf("sites=%+v, want %s", got, tc.sub)
			}
		})
	}
}
func TestDefaultSiteListByRegionReturnsIndependentSlices(t *testing.T) {
	first := DefaultSiteListByRegion("FR")
	first[0].MainSite = "changed"
	first[0].SubSiteList[0] = "changed"
	next := DefaultSiteListByRegion("FR")
	if next[0].MainSite != "shein" || next[0].SubSiteList[0] != "shein-fr" {
		t.Fatalf("default sites share mutable state: %+v", next)
	}
}
