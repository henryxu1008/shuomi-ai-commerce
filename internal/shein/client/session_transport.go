package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// SessionTransport makes bounded platform reads with an explicit current
// session. It has no account repository, numeric identity or login refresher.
type SessionTransport struct {
	base      string
	client    *http.Client
	transport *http.Transport
}

func NewSessionTransport(base, proxy string, cookies []*http.Cookie) (*SessionTransport, error) {
	u, e := url.Parse(base)
	if e != nil || u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Host != "sellerhub.shein.com" && u.Host != "sso.geiwohuo.com") {
		return nil, errors.New("invalid SHEIN session endpoint")
	}
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	t.Proxy = nil
	if proxy != "" {
		p, e := url.Parse(proxy)
		if e != nil || p.Host == "" {
			return nil, errors.New("invalid SHEIN session proxy")
		}
		t.Proxy = http.ProxyURL(p)
	}
	// Redirects must never carry the explicit Cookie to another origin or launch
	// a hidden login. Authentication expiry is a safe failed read.
	c := &http.Client{Transport: t, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	rt := &SessionTransport{base: strings.TrimRight(base, "/"), client: c, transport: t}
	jar, _ := cookiejar.New(nil)
	jar.SetCookies(u, cookies)
	rt.client.Jar = jar
	return rt, nil
}

func (s *SessionTransport) Close() { s.transport.CloseIdleConnections() }
func (s *SessionTransport) Read(ctx context.Context, path string, body any, out any) error {
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return errors.New("invalid SHEIN endpoint")
	}
	b, e := json.Marshal(body)
	if e != nil {
		return e
	}
	r, e := http.NewRequestWithContext(ctx, http.MethodPost, s.base+path, bytes.NewReader(b))
	if e != nil {
		return errors.New("SHEIN request unavailable")
	}
	headers := sessionHeaders(s.base, path)
	headers["content-type"] = "application/json"
	if strings.Contains(path, "/sso/public/account/supplier/getSupplierOperateInfo") {
		headers["gmpsso-language"] = "CN"
		headers["path"] = "/sso/public/account/supplier/getSupplierOperateInfo"
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	response, e := s.client.Do(r)
	if e != nil {
		return errors.New("SHEIN session read unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errors.New("SHEIN session read rejected")
	}
	data, e := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if e != nil || len(data) > 1<<20 {
		return errors.New("SHEIN session response invalid")
	}
	if json.Unmarshal(data, out) != nil {
		return errors.New("SHEIN session response invalid")
	}
	return nil
}

// Shared value behavior extracted from the original API transport; callers
// retain the same platform headers without constructing a numeric Store owner.
func sessionHeaders(base, path string) map[string]string {
	h := map[string]string{"priority": "u=1, i", "referer": base + "/", "sec-ch-ua": `"Google Chrome";v="138", "Chromium";v="138", "Not/A)Brand";v="24"`, "sec-ch-ua-mobile": "?0", "sec-ch-ua-platform": `"Windows"`, "sec-fetch-dest": "empty", "sec-fetch-mode": "cors", "sec-fetch-site": "same-origin", "user-agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36", "front-version": "20250207", "x-lt-language": "CN", "x-req-zone-id": "America/Los_Angeles"}

	return h
}

// CookiesFromBrowserState reuses the existing platform Cookie value conversion,
// including expiry/domain/SameSite, without a numeric CookieManager instance.
func CookiesFromBrowserState(state map[string]any) ([]*http.Cookie, error) {
	data, err := json.Marshal(state["cookies"])
	if err != nil {
		return nil, errors.New("invalid browser cookies")
	}
	var values []CookieData
	if json.Unmarshal(data, &values) != nil || len(values) == 0 || len(values) > 500 {
		return nil, errors.New("invalid browser cookies")
	}
	for _, c := range values {
		if c.Name == "" || len(c.Name)+len(c.Value) > 16384 {
			return nil, errors.New("invalid browser cookies")
		}
	}
	return buildCookiesFromJSON(values), nil
}
