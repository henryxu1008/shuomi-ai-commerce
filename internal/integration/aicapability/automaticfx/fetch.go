package automaticfx

import (
	"context"
	"errors"
	"net/http"
	"time"
)

const cfetsEndpoint = "https://api.frankfurter.dev/v2/providers/cfets/rate/usd/cny"

var ErrSourceUnavailable = errors.New("CFETS observation unavailable")

// FetchCFETS contacts only the pinned HTTPS endpoint. transport is a trusted
// server dependency; a synthetic RoundTripper permits offline verification.
func FetchCFETS(ctx context.Context, transport http.RoundTripper, now time.Time) (Observation, error) {
	if ctx == nil {
		return Observation{}, ErrSourceUnavailable
	}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfetsEndpoint, nil)
	if err != nil {
		return Observation{}, ErrSourceUnavailable
	}
	resp, err := client.Do(req)
	if err != nil {
		return Observation{}, ErrSourceUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Observation{}, ErrSourceUnavailable
	}
	return ParseCFETSObservation(resp.Body, now)
}
