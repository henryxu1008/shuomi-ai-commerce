package setupcontrol

import (
	"context"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testCatalog struct{}

func (testCatalog) Snapshot(context.Context) (Snapshot, error) {
	return Snapshot{DockerReady: true}, nil
}
func (testCatalog) Inspect(context.Context, string) (Instance, error) { return Instance{}, nil }
func TestPairingRequiresExactOriginAndDoesNotAcceptURLOrCrossSiteAuthority(t *testing.T) {
	s := NewHTTP("127.0.0.1:39222", "one-time-code", New(t.TempDir(), &testDriver{}), testCatalog{}, http.NotFoundHandler())
	call := func(host, origin, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "http://"+host+path, strings.NewReader(body))
		r.Host = host
		r.Header.Set("Origin", origin)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		return w
	}
	require.Equal(t, 403, call("evil.test", "http://evil.test", "/api/pair", `{"code":"one-time-code"}`).Code)
	require.Equal(t, 403, call("127.0.0.1:39222", "http://evil.test", "/api/pair", `{"code":"one-time-code"}`).Code)
	require.Equal(t, 400, call("127.0.0.1:39222", "http://127.0.0.1:39222", "/api/pair?code=one-time-code", `{"code":"one-time-code"}`).Code)
	w := call("127.0.0.1:39222", "http://127.0.0.1:39222", "/api/pair", `{"code":"one-time-code"}`)
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Header().Get("Set-Cookie"), "HttpOnly")
	require.Contains(t, w.Header().Get("Set-Cookie"), "SameSite=Strict")
	require.Equal(t, 403, call("127.0.0.1:39222", "http://127.0.0.1:39222", "/api/pair", `{"code":"one-time-code"}`).Code)
}
func TestMutationRequiresSessionAndCSRFAndRejectsAmbiguousJSON(t *testing.T) {
	d := &testDriver{revision: "before"}
	s := NewHTTP("127.0.0.1:39222", "code", New(t.TempDir(), d), testCatalog{}, http.NotFoundHandler())
	pair := httptest.NewRequest("POST", "http://127.0.0.1:39222/api/pair", strings.NewReader(`{"code":"code"}`))
	pair.Header.Set("Origin", "http://127.0.0.1:39222")
	pair.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, pair)
	require.Equal(t, 200, w.Code)
	call := func(body, csrf string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "http://127.0.0.1:39222/api/plans", strings.NewReader(body))
		r.Header.Set("Origin", "http://127.0.0.1:39222")
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Setup-CSRF", csrf)
		r.AddCookie(w.Result().Cookies()[0])
		out := httptest.NewRecorder()
		s.ServeHTTP(out, r)
		return out
	}
	require.Equal(t, 403, call(`{"mode":"existing"}`, "").Code)
	require.Equal(t, 400, call(`{"mode":"existing","mode":"fresh"}`, s.csrf).Code)
	require.Equal(t, 400, call(`{"mode":"existing","command":"rm"}`, s.csrf).Code)
	require.Zero(t, d.calls)
}
