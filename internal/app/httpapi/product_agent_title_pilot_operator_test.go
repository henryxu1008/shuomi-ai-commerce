package httpapi

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTitlePilotOperatorLoopbackBoundary(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	host := listener.Addr().String()
	server := &http.Server{Handler: titlePilotLoopback(host, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))}
	t.Cleanup(func() { _ = server.Close(); _ = listener.Close() })
	go func() { _ = server.Serve(listener) }()
	client := &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{}}
	t.Cleanup(client.CloseIdleConnections)
	for _, tc := range []struct {
		name, query, header, value string
		want                       int
	}{
		{"direct", "", "", "", 204},
		{"proxy", "", "Forwarded", "for=127.0.0.1", 400},
		{"legacy proxy", "", "X-Forwarded-For", "127.0.0.1", 400},
		{"host", "", "Host", "localhost:5555", 400},
		{"overrides", "?batchId=other", "", "", 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "http://"+host+"/title-pilot/preflight"+tc.query, nil)
			require.NoError(t, err)
			if tc.header == "Host" {
				req.Host = tc.value
			} else if tc.header != "" {
				req.Header.Set(tc.header, tc.value)
			}
			res, err := client.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()
			require.Equal(t, tc.want, res.StatusCode)
		})
	}
	for _, remote := range []string{"[::1]:33000", "192.0.2.1:33000"} {
		req := httptest.NewRequest(http.MethodPost, "http://"+host+"/title-pilot/start", nil)
		req.RemoteAddr = remote
		out := httptest.NewRecorder()
		server.Handler.ServeHTTP(out, req)
		require.Equal(t, http.StatusBadRequest, out.Code)
	}
}

func TestTitlePilotOperatorAuthorizationExpiry(t *testing.T) {
	at := time.Now().UTC()
	clock := at
	op := &TitlePilotOperator{ExecutionApprovalID: "fixture-approval", NotBefore: at.Add(-time.Minute), NotAfter: at.Add(time.Minute), now: func() time.Time { return clock }}
	ctx := context.WithValue(context.Background(), productReviewCapabilityContextKey{}, productReviewRequestCapability{tokenExpiresAt: at.Add(time.Second)})
	require.NoError(t, op.authorizeTime(ctx))
	clock = at.Add(time.Second)
	require.ErrorIs(t, op.authorizeTime(ctx), errTitlePilotExpired)
	ctx = context.WithValue(context.Background(), productReviewCapabilityContextKey{}, productReviewRequestCapability{tokenExpiresAt: at.Add(time.Hour)})
	clock = op.NotAfter
	require.ErrorIs(t, op.authorizeTime(ctx), errTitlePilotExpired)
	clock = at
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	require.ErrorIs(t, op.authorizeTime(cancelled), context.Canceled)
	op.ExecutionApprovalID = ""
	require.ErrorIs(t, op.authorizeTime(ctx), errTitlePilotExpired, "timestamps alone are not an approval")
}
