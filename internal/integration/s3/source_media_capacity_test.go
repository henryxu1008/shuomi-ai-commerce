package s3

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const trialDeploymentID = "d7b806bb-6f4f-47be-999c-99f027252244"

func TestSourceMediaMinIORequiresAuthenticatedPinnedDeployment(t *testing.T) {
	for _, scenario := range []string{"correct", "empty", "different", "forbidden", "redirect"} {
		t.Run(scenario, func(t *testing.T) {
			redirected := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/redirected" {
					redirected++
					w.WriteHeader(500)
					return
				}
				require.Equal(t, http.MethodGet, r.Method)
				require.Equal(t, "/minio/admin/v3/info", r.URL.Path)
				require.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 "))
				w.Header().Set("Content-Type", "application/json")
				switch scenario {
				case "empty":
					_, _ = w.Write([]byte(`{"mode":"online"}`))
				case "different":
					_, _ = w.Write([]byte(`{"deploymentID":"91cac44d-7030-4f4a-a76a-4474af1e73bb"}`))
				case "forbidden":
					w.WriteHeader(403)
				case "redirect":
					w.Header().Set("Location", "/redirected")
					w.WriteHeader(302)
				default:
					_, _ = w.Write([]byte(`{"mode":"online","deploymentID":"` + trialDeploymentID + `"}`))
				}
			}))
			defer server.Close()
			c := ClientConfig{Region: "us-east-1", Endpoint: server.URL, UsePathStyle: true, AccessKeyID: "first-key", SecretAccessKey: "synthetic-secret"}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			ns, err := ResolveSourceMediaCapacityNamespace(ctx, c, "source-media-trial", "aws", true, trialDeploymentID)
			if scenario != "correct" {
				require.Error(t, err)
				require.Empty(t, ns)
				require.Zero(t, redirected)
				return
			}
			require.NoError(t, err)
			require.Len(t, ns, 64)
			c.AccessKeyID, c.SecretAccessKey = "rotated-key", "rotated-secret"
			again, err := ResolveSourceMediaCapacityNamespace(ctx, c, "source-media-trial", "aws", true, trialDeploymentID)
			require.NoError(t, err)
			require.Equal(t, ns, again, "physical namespace must not depend on rotating credentials")
		})
	}
}

func TestSourceMediaMinIORejectsIncompleteOrNonLocalConfiguration(t *testing.T) {
	base := ClientConfig{Region: "us-east-1", Endpoint: "http://127.0.0.1:9103", UsePathStyle: true, AccessKeyID: "fixture", SecretAccessKey: "fixture"}
	for _, pin := range []string{"", "not-an-id", "00000000-0000-0000-0000-000000000000", strings.ToUpper(trialDeploymentID)} {
		_, err := ResolveSourceMediaCapacityNamespace(context.Background(), base, "source-media-trial", "aws", true, pin)
		require.Error(t, err)
	}
	for _, endpoint := range []string{"http://localhost:9103", "http://127.0.0.1", "http://127.0.0.1:9103/", "http://127.0.0.1:9103?x=1", "https://provider.example.org:443"} {
		c := base
		c.Endpoint = endpoint
		_, err := ResolveSourceMediaCapacityNamespace(context.Background(), c, "source-media-trial", "aws", true, trialDeploymentID)
		require.Error(t, err)
	}
	_, err := ResolveSourceMediaCapacityNamespace(context.Background(), base, "source-media-trial", "aws", false, trialDeploymentID)
	require.Error(t, err, "normal S3 cannot consume the trial pin")
}

func TestSourceMediaMinIOCannotUseAnAnonymousIdentityResponse(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write([]byte(`{"deploymentID":"` + trialDeploymentID + `"}`))
	}))
	defer server.Close()
	_, err := ResolveSourceMediaCapacityNamespace(context.Background(), ClientConfig{Endpoint: server.URL, Region: "us-east-1", UsePathStyle: true}, "source-media-trial", "aws", true, trialDeploymentID)
	require.Error(t, err)
	require.Zero(t, requests)
}

func TestMinIOInfoRejectsChunkedValidJSONWithOversizedTail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush() // No Content-Length: a single JSON Decode is insufficient.
		_, _ = w.Write([]byte(`{"deploymentID":"` + trialDeploymentID + `"}`))
		_, _ = w.Write([]byte(strings.Repeat(" ", (1<<20)+1)))
	}))
	defer server.Close()
	_, err := ResolveSourceMediaCapacityNamespace(context.Background(), ClientConfig{Endpoint: server.URL, Region: "us-east-1", UsePathStyle: true, AccessKeyID: "fixture", SecretAccessKey: "fixture"}, "source-media-trial", "aws", true, trialDeploymentID)
	require.Error(t, err, "oversized full response must fail even when its first JSON value is valid")
}
