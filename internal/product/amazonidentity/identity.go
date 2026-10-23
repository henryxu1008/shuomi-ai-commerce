// Package amazonidentity owns pure source site/ASIN normalization shared by producers.
package amazonidentity

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var ErrInvalid = errors.New("invalid amazon identity")
var asinPattern = regexp.MustCompile(`^[A-Z0-9]{10}$`)

type Site struct {
	Code   string `json:"code"`
	Domain string `json:"domain"`
	Name   string `json:"name"`
}

var siteList = []Site{
	{"us", "www.amazon.com", "美国"}, {"uk", "www.amazon.co.uk", "英国"}, {"de", "www.amazon.de", "德国"},
	{"fr", "www.amazon.fr", "法国"}, {"it", "www.amazon.it", "意大利"}, {"es", "www.amazon.es", "西班牙"},
	{"ca", "www.amazon.ca", "加拿大"}, {"jp", "www.amazon.co.jp", "日本"}, {"au", "www.amazon.com.au", "澳大利亚"},
	{"mx", "www.amazon.com.mx", "墨西哥"}, {"br", "www.amazon.com.br", "巴西"}, {"in", "www.amazon.in", "印度"},
	{"ae", "www.amazon.ae", "阿联酋"}, {"sa", "www.amazon.sa", "沙特阿拉伯"},
}

func Sites() []Site { return append([]Site(nil), siteList...) }
func ResolveSite(code string) (Site, error) {
	for _, s := range siteList {
		if s.Code == code {
			return s, nil
		}
	}
	return Site{}, ErrInvalid
}

func NormalizeASIN(site Site, value string) (string, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "https://") {
		u, err := url.Parse(value)
		if err != nil || u.User != nil || u.Port() != "" || u.Fragment != "" || (u.Hostname() != site.Domain && u.Hostname() != strings.TrimPrefix(site.Domain, "www.")) {
			return "", ErrInvalid
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) != 2 || parts[0] != "dp" {
			return "", ErrInvalid
		}
		value = parts[1]
	}
	value = strings.ToUpper(value)
	if !asinPattern.MatchString(value) {
		return "", ErrInvalid
	}
	return value, nil
}
