package s3

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfiguredMinIOCapacityBindsAuthenticatedDeployment(t *testing.T) {
	for _, scenario := range []string{"correct", "different", "empty", "forbidden", "redirect", "malformed", "truncated", "oversized"} {
		t.Run(scenario, func(t *testing.T) {
			redirects := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/minio/admin/v3/info" {
					redirects++
					w.WriteHeader(500)
					return
				}
				require.Equal(t, http.MethodGet, r.Method)
				require.Empty(t, r.Header.Get("Cookie"))
				require.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 "))
				w.Header().Set("Content-Type", "application/json")
				switch scenario {
				case "different":
					_, _ = w.Write([]byte(`{"deploymentID":"91cac44d-7030-4f4a-a76a-4474af1e73bb"}`))
				case "empty":
					_, _ = w.Write([]byte(`{}`))
				case "forbidden":
					w.WriteHeader(403)
				case "redirect":
					w.Header().Set("Location", "/redirected")
					w.WriteHeader(302)
				case "malformed":
					_, _ = w.Write([]byte(`not-json`))
				case "truncated":
					_, _ = w.Write([]byte(`{"deploymentID":`))
				case "oversized":
					w.WriteHeader(200)
					w.(http.Flusher).Flush()
					_, _ = w.Write([]byte(`{"deploymentID":"` + trialDeploymentID + `"}` + strings.Repeat(" ", 1<<20)))
				default:
					_, _ = w.Write([]byte(`{"deploymentID":"` + trialDeploymentID + `"}`))
				}
			}))
			defer server.Close()
			c := ClientConfig{Endpoint: server.URL, Region: "us-east-1", UsePathStyle: true, AccessKeyID: "runtime-key", SecretAccessKey: "synthetic"}
			ns, err := ResolveConfiguredCapacityNamespace(context.Background(), c, "knowledge", "aws", MinIOCapacityIdentityProvider, trialDeploymentID)
			if scenario != "correct" {
				require.Error(t, err)
				require.Empty(t, ns)
				require.Zero(t, redirects)
				return
			}
			require.NoError(t, err)
			require.Len(t, ns, 64)
			c.AccessKeyID, c.SecretAccessKey = "rotated", "rotated-secret"
			again, err := ResolveConfiguredCapacityNamespace(context.Background(), c, "knowledge", "aws", MinIOCapacityIdentityProvider, trialDeploymentID)
			require.NoError(t, err)
			require.Equal(t, ns, again)
			trial, err := ResolveSourceMediaCapacityNamespace(context.Background(), c, "knowledge", "aws", true, trialDeploymentID)
			require.NoError(t, err)
			require.NotEqual(t, trial, ns, "the source-only trial identity is unchanged and distinct")
		})
	}
}

func TestConfiguredMinIOCapacityRejectsInvalidOrNonLocalInputsBeforeIO(t *testing.T) {
	base := ClientConfig{Endpoint: "http://127.0.0.1:9000", Region: "us-east-1", UsePathStyle: true, AccessKeyID: "runtime", SecretAccessKey: "synthetic"}
	for _, endpoint := range []string{"http://localhost:9000", "http://10.0.0.2:9000", "https://example.com:443", "http://knowledge-objects.example.com:9000", "http://knowledge-objects", "http://127.0.0.1:09000", "http://127.0.0.1:9000/", "http://127.0.0.1:9000?", "http://127.0.0.1:9000?x=1", "http://user@127.0.0.1:9000", "http://127.0.0.1:9000#x"} {
		c := base
		c.Endpoint = endpoint
		require.Error(t, ValidateCapacityIdentity(c, "knowledge", "aws", MinIOCapacityIdentityProvider, trialDeploymentID), endpoint)
	}
	for _, endpoint := range []string{base.Endpoint, "https://[::1]:9000", "http://knowledge-objects:9000", "http://ecoservices-objects:9101"} {
		c := base
		c.Endpoint = endpoint
		require.NoError(t, ValidateCapacityIdentity(c, "knowledge", "aws", MinIOCapacityIdentityProvider, trialDeploymentID))
	}
	for _, pin := range []string{"", "invalid", "00000000-0000-0000-0000-000000000000", strings.ToUpper(trialDeploymentID)} {
		require.Error(t, ValidateCapacityIdentity(base, "knowledge", "aws", MinIOCapacityIdentityProvider, pin))
	}
	require.Error(t, ValidateCapacityIdentity(base, "knowledge", "aws", "", trialDeploymentID))
	require.Error(t, ValidateCapacityIdentity(base, "knowledge", "aws", "unknown", trialDeploymentID))
	require.Error(t, ValidateCapacityIdentity(base, "knowledge", "cos", MinIOCapacityIdentityProvider, trialDeploymentID))
	base.AccessKeyID = ""
	require.Error(t, ValidateCapacityIdentity(base, "knowledge", "aws", MinIOCapacityIdentityProvider, trialDeploymentID))
}

func TestConfiguredCapacityCloudOwnerNeverFallsBackToMinIO(t *testing.T) {
	infoCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/minio/admin/v3/info" {
			infoCalls++
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<AccessControlPolicy><Owner/></AccessControlPolicy>`))
	}))
	defer server.Close()
	_, err := ResolveConfiguredCapacityNamespace(context.Background(), ClientConfig{Endpoint: server.URL, Region: "us-east-1", AccessKeyID: "runtime", SecretAccessKey: "synthetic", UsePathStyle: true}, "knowledge", "aws", "", "")
	require.Error(t, err)
	require.Zero(t, infoCalls)
}

func TestConfiguredMinIOCapacityHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ResolveConfiguredCapacityNamespace(ctx, ClientConfig{Endpoint: "http://127.0.0.1:9000", Region: "us-east-1", AccessKeyID: "runtime", SecretAccessKey: "synthetic", UsePathStyle: true}, "knowledge", "aws", MinIOCapacityIdentityProvider, trialDeploymentID)
	require.Error(t, err)
}
