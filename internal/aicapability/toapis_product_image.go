package aicapability

import (
	"net/url"
	"path"
	"strings"
)

const (
	ToAPIsProductImageProvider = "toapis"
	ToAPIsProductImageModel    = "gpt-image-2.6-flare"
	ToAPIsProductImageProtocol = "toapis-product-image-async-v1"
	ToAPIsProductImageBaseURL  = "https://api.toapis.cn"
)

func OwnedProductImageQuality(provider, model, protocol string) string {
	switch {
	case provider == "grsai" && model == GRSaiProductImageModel && protocol == GRSaiProductImageProtocol:
		return "auto"
	case provider == ToAPIsProductImageProvider && model == ToAPIsProductImageModel && protocol == ToAPIsProductImageProtocol:
		return "high"
	default:
		return ""
	}
}

// ToAPIsProductImageSource is limited to public 1688 product photographs.
func ToAPIsProductImageSource(raw string) bool {
	const directory = "/img/ibank/"
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != "cbu01.alicdn.com" ||
		u.User != nil || strings.ContainsAny(raw, "?#%\\") || path.Clean(u.Path) != u.Path {
		return false
	}
	name, ok := strings.CutPrefix(u.Path, directory)
	if !ok || strings.Contains(name, "/") {
		return false
	}
	ext := path.Ext(name)
	return len(name) > len(ext) && (ext == ".jpg" || ext == ".jpeg" || ext == ".png")
}
