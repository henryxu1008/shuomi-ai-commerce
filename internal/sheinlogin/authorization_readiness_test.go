package sheinlogin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"task-processor/internal/authruntime/zitadel"
)

func TestReadinessWithNativeProviderClientsUsesOnlyServiceIdentity(t *testing.T) {
	var revoked, unavailable atomic.Bool
	var introspections, reads atomic.Int32
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]string{"introspection_endpoint": server.URL + "/introspect"})
		case "/introspect":
			introspections.Add(1)
			user, password, ok := r.BasicAuth()
			if !ok || user != "client" || password != "synthetic-client-secret" || r.ParseForm() != nil || r.Form.Get("token") != "synthetic-service-token" {
				t.Error("native introspection input changed")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"active": !revoked.Load(), "sub": "service-subject", "urn:zitadel:iam:user:resourceowner:id": "service-home"})
		case "/zitadel.authorization.v2.AuthorizationService/ListAuthorizations":
			reads.Add(1)
			data, _ := io.ReadAll(r.Body)
			for _, value := range []string{"service-subject", "project", "org-a"} {
				if !strings.Contains(string(data), value) {
					t.Errorf("missing exact readiness scope %s", value)
				}
			}
			if r.Header.Get("Authorization") != "Bearer synthetic-service-token" {
				t.Error("wrong readonly service credential")
			}
			if unavailable.Load() {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			_, _ = io.WriteString(w, `{"pagination":{"totalResult":"0"},"authorizations":[]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	a := &Authorization{ProjectID: "project", Verifier: zitadel.NewVerifier(zitadel.Config{IssuerURL: server.URL, ClientID: "client", ClientSecret: "synthetic-client-secret", HTTPClient: server.Client()}), Grants: zitadel.NewAuthorizationClient(server.URL, server.Client()), ServiceToken: func(context.Context) (string, error) { return "synthetic-service-token", nil }}
	if !a.Ready(context.Background(), "org-a") {
		t.Fatal("native empty-grant dependency probe failed")
	}
	revoked.Store(true)
	if a.Ready(context.Background(), "org-a") {
		t.Fatal("revoked service token remained ready")
	}
	revoked.Store(false)
	unavailable.Store(true)
	if a.Ready(context.Background(), "org-a") {
		t.Fatal("authorization outage remained ready")
	}
	unavailable.Store(false)
	if !a.Ready(context.Background(), "org-a") || introspections.Load() != 4 || reads.Load() != 3 {
		t.Fatal("fresh native recovery checks missing")
	}
}
