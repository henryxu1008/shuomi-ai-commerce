package rules_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/marketplace/temu/rules"
)

func TestCatalogPreservesOfficialSetsAndMissingLocalSpecifications(t *testing.T) {
	c, err := rules.NewOfficialCatalog()
	require.NoError(t, err)
	semi := c.SemiSites()
	require.Len(t, semi, 96)
	open := 0
	for _, site := range semi {
		if site.Opening == rules.Open {
			open++
		} else {
			require.Equal(t, rules.Closed, site.Opening)
		}
	}
	require.Equal(t, 70, open)
	require.Len(t, c.LocalSpecifications(), 39)
	regions := c.Regions()
	require.Len(t, regions, 87)
	sites, missing := map[int64]bool{}, map[int64]bool{}
	localOpen, specified := 0, 0
	for _, region := range regions {
		sites[region.SiteID] = true
		if region.Local == rules.Open {
			localOpen++
			_, ok := c.LocalSpecification(region.SiteID)
			if ok {
				specified++
			} else {
				missing[region.SiteID] = true
			}
		}
	}
	require.Len(t, sites, 83)
	require.Equal(t, 39, localOpen)
	require.Equal(t, 37, specified)
	require.Equal(t, map[int64]bool{165: true, 166: true}, missing)
	require.Len(t, c.Revision(), 64)
	require.Len(t, c.Sources(), 3)
}

func TestLocalTargetUsesExactRegionAndOriginalApplicationPartition(t *testing.T) {
	c, err := rules.NewOfficialCatalog()
	require.NoError(t, err)
	for _, fixture := range []struct {
		partition    string
		region, site int64
		issue        rules.IssueCode
	}{
		{"US", 211, 100, ""},
		{"US", 83, 100, rules.LocalOpeningUnverified},
		{"GLOBAL", 100, 118, ""},
		{"US", 100, 118, rules.PartitionMismatch},
		{"EU", 20, 142, ""},
		{"GLOBAL", 20, 142, rules.PartitionMismatch},
		{"EU", 75, 165, rules.SpecificationsMissing},
		{"EU", 10, 166, rules.SpecificationsMissing},
		{"EU", 174, 120, rules.LocalOpeningUnverified},
		{"EU", 114, 152, rules.LocalOpeningUnverified},
		{"GLOBAL", 999999, 0, rules.RegionMappingMissing},
		{"CN", 211, 100, rules.PartitionMismatch},
	} {
		t.Run(fmt.Sprintf("%s/%d", fixture.partition, fixture.region), func(t *testing.T) {
			view := c.LocalTarget(fixture.partition, fixture.region)
			require.Equal(t, fixture.site, view.Region.SiteID)
			require.Equal(t, fixture.partition, view.ApplicationPartition, "a specification label must not reroute an application")
			if fixture.issue == "" {
				require.Empty(t, view.Issues)
			} else {
				require.Contains(t, view.Issues, fixture.issue)
			}
		})
	}
	belgium := c.LocalTarget("EU", 20)
	require.Equal(t, "GLO", belgium.Specification.DocumentPartition)
	require.Equal(t, "EU", belgium.Region.DeploymentPartition)
	guam := c.LocalTarget("US", 83)
	require.Equal(t, rules.Unknown, guam.Region.Local, "blank is not authorization or false")
}

func TestSemiSelectionKeepsItsOwnDictionaryAndAssociationFacts(t *testing.T) {
	c, err := rules.NewOfficialCatalog()
	require.NoError(t, err)
	korea, ok := c.SemiSite(119)
	require.True(t, ok)
	require.Equal(t, rules.Closed, korea.Opening)
	local := c.LocalTarget("GLOBAL", 185)
	require.Equal(t, rules.Open, local.Region.Local)
	selection := c.SemiSelection([]int64{119})
	require.Contains(t, selection.Issues, rules.SemiSiteClosed)
	require.Contains(t, c.SemiSelection([]int64{160}).Issues, rules.SemiSiteUnknown)
	require.Empty(t, c.SemiSelection([]int64{192}).Issues)
	require.Contains(t, c.SemiSelection([]int64{103}).Issues, rules.AssociatedSitesMissing)
	require.Empty(t, c.SemiSelection([]int64{104, 103}).Issues)
	require.Equal(t, []int64{103, 104}, c.SemiSelection([]int64{104, 103}).SiteIDs)
	pan := c.SemiSelection([]int64{105})
	require.True(t, pan.RequiresPanEuropeanEvidence, "do not fabricate the current merchant's Pan-European eligibility or group")
	require.Contains(t, pan.Issues, rules.PanEuropeanEligibilityMissing)
	require.Contains(t, c.SemiSelection([]int64{100, 100}).Issues, rules.DuplicateSite)
}

func TestOfficialRulesRejectMissingRoundedConvertedAndOutOfRangeFacts(t *testing.T) {
	c, err := rules.NewOfficialCatalog()
	require.NoError(t, err)
	us, _ := c.LocalSpecification(100)
	ca, _ := c.LocalSpecification(101)
	jp, _ := c.LocalSpecification(118)
	bg, _ := c.LocalSpecification(141)
	for _, fixture := range []struct {
		name  string
		rule  rules.NumberRule
		value string
		valid bool
	}{
		{"US base lower", us.BasePrice, "0.02", true},
		{"US base below", us.BasePrice, "0.01", false},
		{"CA base below", ca.BasePrice, "1.99", false},
		{"CA base lower", ca.BasePrice, "2.00", true},
		{"US base maximum", us.BasePrice, "99999999.99", true},
		{"no rounding", us.BasePrice, "0.019", false},
		{"integer", jp.BasePrice, "2", true},
		{"integer fractional", jp.BasePrice, "2.01", false},
		{"weight lower", us.Weight.Number, "0.1", true},
		{"old default is not rule", us.Weight.Number, "0.22", false},
		{"weight maximum", us.Weight.Number, "9999.9", true},
		{"weight over", us.Weight.Number, "10000", false},
		{"open bound", bg.Weight.Number, "9999999.9", true},
		{"open bound endpoint", bg.Weight.Number, "10000000", false},
		{"empty", us.Weight.Number, "", false},
		{"unit in number", us.Weight.Number, "1 lb", false},
		{"exponent", us.BasePrice, "2e0", false},
		{"fraction", us.BasePrice, "2/1", false},
		{"negative", us.BasePrice, "-2", false},
		{"unknown rule", rules.NumberRule{}, "2", false},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			require.Equal(t, fixture.valid, fixture.rule.Allows(fixture.value))
		})
	}
	require.Equal(t, "lb", us.Weight.Unit)
	require.Equal(t, "g", ca.Weight.Unit)
	for _, site := range []int64{103, 108, 140, 142} {
		spec, ok := c.LocalSpecification(site)
		require.True(t, ok)
		require.True(t, spec.Dimension.Number.Allows("9999999.9"), "retain actual site-specific dimension range")
	}
	require.False(t, ca.Dimension.Number.Allows("10000"))
}

func TestCatalogCopiesCannotAlterSubsequentRuleSnapshots(t *testing.T) {
	c, err := rules.NewOfficialCatalog()
	require.NoError(t, err)
	before := c.Revision()
	sites := c.SemiSites()
	sites[0].Opening = rules.Closed
	site, ok := c.SemiSite(103)
	require.True(t, ok)
	site.AssociatedSiteIDs[0] = 999
	spec, ok := c.LocalSpecification(100)
	require.True(t, ok)
	spec.Languages[0] = "unrecognized"
	spec2, _ := c.LocalSpecification(100)
	require.Equal(t, "en", spec2.Languages[0])
	site2, _ := c.SemiSite(103)
	require.Equal(t, []int64{103, 104}, site2.AssociatedSiteIDs)
	require.Equal(t, before, c.Revision())
}
