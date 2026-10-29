package draft

import (
	"sort"
	"strconv"
	"strings"

	"task-processor/internal/marketplace/shein/offlinepackage"
	productasset "task-processor/internal/product/asset"
	"task-processor/internal/product/catalog/canonical"
)

func buildSourceImages(product *canonical.Product) *offlinepackage.ImageSet {
	set := &offlinepackage.ImageSet{}
	if product == nil {
		return nil
	}
	for _, image := range product.Images {
		url := strings.TrimSpace(image.URL)
		if url == "" {
			continue
		}
		switch productasset.Role(image.Role) {
		case productasset.RoleMain:
			if set.MainImage == "" {
				set.MainImage = url
			}
		case productasset.RoleWhiteBackground:
			if set.WhiteBgImage == "" {
				set.WhiteBgImage = url
			}
		case productasset.RoleDesign, productasset.RoleGallery:
			set.Gallery = append(set.Gallery, url)
		}
	}
	set.Gallery = uniqueSourceStrings(set.Gallery)
	if set.MainImage == "" && set.WhiteBgImage == "" && len(set.Gallery) == 0 {
		return nil
	}
	return set
}

func flattenSourceAttributes(attributes map[string]canonical.Attribute) map[string]string {
	if len(attributes) == 0 {
		return nil
	}
	result := make(map[string]string, len(attributes))
	for key, value := range attributes {
		result[key] = value.Value
	}
	return result
}

func buildSourceAttributes(attributes map[string]canonical.Attribute) []offlinepackage.Attribute {
	if len(attributes) == 0 {
		return nil
	}
	result := make([]offlinepackage.Attribute, 0, len(attributes))
	keys := make([]string, 0, len(attributes))
	for key := range attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		result = append(result, offlinepackage.Attribute{Name: key, Value: attributes[key].Value})
	}
	return result
}

func defaultSourceSites(country string) []offlinepackage.Site {
	country = strings.ToUpper(strings.TrimSpace(country))
	if country == "" {
		country = "US"
	}
	return []offlinepackage.Site{{MainSite: country, SubSites: []string{country}}}
}

func formatSourceFloat(value float64) string {
	if value == 0 {
		return ""
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func cloneSourceMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func parseSourceFloatDefault(value string) float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func lastSourceCategory(path []string) string {
	if len(path) == 0 {
		return ""
	}
	return path[len(path)-1]
}

func uniqueSourceStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
