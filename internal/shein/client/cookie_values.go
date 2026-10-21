package client

import (
	"net/http"
	"strings"
	"time"
)

// CookieData is the browser session Cookie value consumed by SessionTransport.
type CookieData struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	HttpOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	SameSite string  `json:"sameSite"`
}

func buildCookiesFromJSON(cookieDataList []CookieData) []*http.Cookie {
	cookies := make([]*http.Cookie, 0, len(cookieDataList))
	for _, cookieData := range cookieDataList {
		cookie := &http.Cookie{
			Name:     cookieData.Name,
			Value:    cookieData.Value,
			Domain:   cookieData.Domain,
			Path:     cookieData.Path,
			HttpOnly: cookieData.HttpOnly,
			Secure:   cookieData.Secure,
		}
		if cookieData.Expires > 0 {
			cookie.Expires = time.Unix(int64(cookieData.Expires), 0)
		}

		switch strings.ToLower(cookieData.SameSite) {
		case "strict":
			cookie.SameSite = http.SameSiteStrictMode
		case "lax":
			cookie.SameSite = http.SameSiteLaxMode
		case "none":
			cookie.SameSite = http.SameSiteNoneMode
		default:
			cookie.SameSite = http.SameSiteDefaultMode
		}

		cookies = append(cookies, cookie)
	}
	return cookies
}
