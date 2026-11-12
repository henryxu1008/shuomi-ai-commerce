package automaticfx

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFetchUsesOnlyPinnedProviderAndRejectsRedirect(t *testing.T) {
	now := time.Date(2026, 11, 12, 2, 0, 0, 0, time.UTC)
	requests := 0
	transport := transportFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		if r.URL.String() != cfetsEndpoint || r.Method != http.MethodGet {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"date":"2026-11-12","base":"USD","quote":"CNY","rate":6.7174}`)), Request: r}, nil
	})
	got, err := FetchCFETS(context.Background(), transport, now)
	if err != nil || got.CNYMicrosPerUSD != 6717400 || requests != 1 {
		t.Fatalf("got %+v, err %v, requests %d", got, err, requests)
	}
	redirect := transportFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": []string{"https://elsewhere.invalid/"}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	})
	if _, err = FetchCFETS(context.Background(), redirect, now); err == nil {
		t.Fatal("redirect must fail")
	}
}
