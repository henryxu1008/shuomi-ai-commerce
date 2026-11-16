package geography

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

func TestBundledCatalogOptionsAndHierarchy(t *testing.T) {
	c, err := Bundled()
	if err != nil {
		t.Fatal(err)
	}
	// Round-trip names must be unique within their respective parent.
	check := func(options []Option) {
		t.Helper()
		ids, names := map[string]bool{}, map[string]bool{}
		for _, option := range options {
			if ids[option.ID] || names[option.Name] || option.ID == "" ||
				option.Name == "" || strings.TrimSpace(option.Name) != option.Name ||
				len(option.Name) > 128 || !utf8.ValidString(option.Name) ||
				strings.IndexFunc(option.Name, unicode.IsControl) >= 0 {
				t.Fatalf("invalid catalog option: %+v", option)
			}
			ids[option.ID], names[option.Name] = true, true
		}
	}
	check(c.Countries())
	var directCase bool
	for _, entry := range c.Countries() {
		country := c.Country(entry.ID)
		check(country.Cities)
		provinceOptions := make([]Option, 0, len(country.Provinces))
		for _, province := range country.Provinces {
			provinceOptions = append(provinceOptions, province.Option)
			check(province.Cities)
		}
		check(provinceOptions)
		if len(country.Cities) > 0 && len(country.Provinces) > 0 {
			directCase = true
			city := country.Cities[0]
			if labels, err := c.Resolve(c.Version(), entry.ID, Direct, city.ID); err != nil || labels.Province != "" || labels.City != city.Name {
				t.Fatalf("direct city %+v: %+v, %v", city, labels, err)
			}
			if _, err := c.Resolve(c.Version(), entry.ID, "", city.ID); err == nil {
				t.Fatalf("direct city must use the explicit group: %s", entry.ID)
			}
		}
	}
	if !directCase {
		t.Fatal("expected a direct-city group in the published snapshot")
	}
	for _, selection := range [][3]string{
		{"SG", "", "1880252"},
		{"US", "5332921", "5368361"},
		{"CN", "1809935", "1795565"},
	} {
		t.Run(fmt.Sprint(selection), func(t *testing.T) {
			labels, err := c.Resolve(c.Version(), selection[0], selection[1], selection[2])
			if err != nil || labels.City == "" {
				t.Fatalf("%+v: %+v, %v", selection, labels, err)
			}
		})
	}
}
