package bravesearch

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/websearch"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/aicapability/einomodel"
)

const maxBody = 256 << 10

type Client struct{ transport http.RoundTripper }

func New(transport http.RoundTripper) Client {
	if transport == nil {
		production := einomodel.NewSingleSendHTTPTransport()
		production.Proxy = nil
		transport = production
	}
	return Client{transport: transport}
}

func (b Client) Search(ctx context.Context, key, query, country, language string) ([]websearch.Result, error) {
	if !websearch.ValidQuery(query) || key == "" || b.transport == nil {
		return nil, websearch.ErrInvalid
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, websearch.BaseURL+"/res/v1/web/search", nil)
	if err != nil {
		return nil, websearch.ErrUnavailable
	}
	params := url.Values{"q": {query}, "count": {"5"}, "result_filter": {"web"}, "country": {country}, "search_lang": {language}}
	req.URL.RawQuery = params.Encode()
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", key)
	resp, err := b.transport.RoundTrip(req)
	if err != nil {
		return nil, websearch.ErrUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, websearch.ErrUnavailable
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil || len(raw) > maxBody {
		return nil, websearch.ErrUnavailable
	}
	var response struct {
		Web *struct {
			Results []struct{ Title, URL, Description string }
		}
		Type string
	}
	if json.Unmarshal(raw, &response) != nil || response.Web == nil && response.Type != "search" {
		return nil, websearch.ErrUnavailable
	}
	items := []websearch.Result{}
	if response.Web == nil {
		return items, nil
	}
	for _, item := range response.Web.Results {
		if len(items) == 5 {
			break
		}
		if !publicLink(item.URL) || strings.TrimSpace(item.Title) == "" {
			continue
		}
		items = append(items, websearch.Result{Title: clip(item.Title, 200), URL: item.URL, Snippet: clip(item.Description, 500)})
	}
	return items, nil
}

func publicLink(s string) bool {
	if len(s) > 2048 || strings.ContainsAny(s, " \r\n\t\\") {
		return false
	}
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.Opaque != "" {
		return false
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if ip, err := netip.ParseAddr(host); err == nil {
		if ip.Zone() != "" {
			return false
		}
		ip = ip.Unmap()
		return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast()
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return false
	}
	// A browser can normalize legacy octal/hex forms into an IP address even
	// when Go's strict IP parser rejects the string.
	last := labels[len(labels)-1]
	if _, err := strconv.ParseUint(last, 10, 64); err == nil {
		return false
	}
	if strings.HasPrefix(last, "0x") {
		if _, err := strconv.ParseUint(last[2:], 16, 64); err == nil {
			return false
		}
	}
	for _, label := range labels {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		for _, ch := range label {
			if ch != '-' && (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') {
				return false
			}
		}
	}
	return true
}

func clip(s string, limit int) string {
	runes := []rune(s)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return s
}
