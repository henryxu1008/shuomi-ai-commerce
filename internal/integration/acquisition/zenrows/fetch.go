// Package zenrows adapts the fixed SHEIN US page trial to ZenRows Fetch and
// normalizes evidence for the original request. Serving runtime wiring is separate.
package zenrows

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
)

const (
	fetchURL         = "https://api.zenrows.com/v1/"
	maxResponseBytes = 5_000_000
)

// ResolveKey resolves the approved non-secret account and key references.
type ResolveKey func(ctx context.Context, accountRef, keyRef string) (string, error)

// NewDedicatedKeyResolver opens only the reference frozen in the approval.
// Construction does not read the secret or dispatch a Fetch request.
func NewDedicatedKeyResolver(a dataacquisition.SHEINUSTrialApproval, read ResolveKey) (ResolveKey, error) {
	if read == nil || !collection.ValidID(a.AccountRef) || !collection.ValidID(a.KeyRef) || a.ExpiresAt.IsZero() {
		return nil, dataacquisition.ErrUnavailable
	}
	return func(ctx context.Context, accountRef, keyRef string) (string, error) {
		if ctx == nil || ctx.Err() != nil || accountRef != a.AccountRef || keyRef != a.KeyRef || !time.Now().Before(a.ExpiresAt) {
			return "", dataacquisition.ErrUnavailable
		}
		key, err := read(ctx, a.AccountRef, a.KeyRef)
		if err != nil || ctx.Err() != nil || !time.Now().Before(a.ExpiresAt) || len(key) == 0 || len(key) > 512 || strings.TrimSpace(key) != key || strings.ContainsAny(key, "\x00\r\n\t") {
			return "", dataacquisition.ErrUnavailable
		}
		return key, nil
	}, nil
}

type Fetch struct {
	http    *http.Client
	resolve ResolveKey
}

func NewFetch(resolve ResolveKey) (*Fetch, error) {
	if resolve == nil {
		return nil, dataacquisition.ErrUnavailable
	}
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	return &Fetch{
		resolve: resolve,
		http: &http.Client{
			Transport: &http.Transport{
				// HTTP/1 on a fresh connection avoids transport replay on a
				// reused connection. A Fetch GET may consume paid credits.
				Protocols:              protocols,
				DisableKeepAlives:      true,
				MaxResponseHeaderBytes: 64 << 10,
			},
			Timeout:       dataacquisition.SHEINUSTrialTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

// Send is the adapter for the already claimed, single-use trial slot. A
// complete response may confirm billing even when there are no usable cards.
func (f *Fetch) Send(ctx context.Context, a dataacquisition.SHEINUSTrialApproval, request dataacquisition.SHEINUSTrialRequest) (dataacquisition.SHEINUSTrialSubmission, error) {
	usage := dataacquisition.SHEINUSTrialSubmission{SHEINUSTrialUsage: dataacquisition.SHEINUSTrialUsage{ApprovalID: a.ID, Binding: a.Binding()}}
	if !request.Valid() {
		return usage, dataacquisition.ErrInvalid
	}
	ctx, cancel := context.WithDeadline(ctx, a.ExpiresAt)
	defer cancel()
	if ctx.Err() != nil {
		return usage, dataacquisition.ErrUnknown
	}
	key, err := f.resolve(ctx, a.AccountRef, a.KeyRef)
	if err != nil || key == "" {
		return usage, dataacquisition.ErrUnavailable
	}
	query := url.Values{
		"apikey":        {key},
		"url":           {request.URL},
		"mode":          {request.Mode},
		"extract":       {request.Extract},
		"proxy_country": {request.ProxyCountry},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL+"?"+query.Encode(), nil)
	if err != nil {
		return usage, dataacquisition.ErrUnknown
	}
	resp, err := f.http.Do(req)
	if err != nil {
		// The transport error can include the URL with its query credential.
		return usage, dataacquisition.ErrUnknown
	}
	defer resp.Body.Close()

	id := singleHeader(resp.Header, "X-Request-Id")
	if dataacquisition.ValidSHEINUSTrialRequestID(id) && !strings.Contains(id, key) {
		usage.RequestID = id
	}
	if resp.StatusCode < 200 || (resp.StatusCode >= 300 && resp.StatusCode < 400) {
		return usage, dataacquisition.ErrUnknown
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil || len(body) > maxResponseBytes || ctx.Err() != nil {
		return usage, dataacquisition.ErrUnknown
	}
	if usage.RequestID != "" {
		// X-Request-Cost is a USD amount. Only the explicit integer credits
		// header, including a valid zero, is suitable for trial accounting.
		if n, err := strconv.ParseUint(singleHeader(resp.Header, "X-Request-Credits"), 10, 31); err == nil {
			actual := int(n)
			usage.Credits = &actual
		}
	}
	if usage.Credits != nil && resp.StatusCode == http.StatusOK && singleHeader(resp.Header, "Zr-Final-Url") == request.URL {
		usage.Cards = pageCards(body, request.CardLimit)
	}
	return usage, nil
}

func singleHeader(headers http.Header, name string) string {
	values := headers.Values(name)
	if len(values) != 1 {
		return ""
	}
	return strings.Trim(values[0], " \t")
}
