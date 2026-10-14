package s3

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/google/uuid"
	madmin "github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"task-processor/internal/ledger/storagecapacity"
)

// ResolveSourceMediaCapacityNamespace is the explicit source-only trial edge.
// Normal providers still require their original canonical bucket owner proof.
func ResolveSourceMediaCapacityNamespace(ctx context.Context, c ClientConfig, bucket, mode string, trial bool, deploymentID string) (string, error) {
	if !trial {
		if deploymentID != "" {
			return "", storagecapacity.ErrInvalid
		}
		return ResolveCapacityNamespace(ctx, c, bucket, mode)
	}
	id, err := uuid.Parse(deploymentID)
	u, parseErr := url.Parse(c.Endpoint)
	if err != nil || id == uuid.Nil || id.String() != deploymentID || parseErr != nil ||
		mode != "aws" || !c.UsePathStyle || !storagecapacity.ValidID(bucket, 128) ||
		strings.TrimSpace(c.AccessKeyID) == "" || strings.TrimSpace(c.SecretAccessKey) == "" || strings.TrimSpace(c.Region) == "" ||
		u.Hostname() != "127.0.0.1" || u.Scheme != "http" && u.Scheme != "https" ||
		u.Port() == "" || u.User != nil || u.Path != "" || u.RawPath != "" ||
		u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || u.String() != c.Endpoint {
		return "", storagecapacity.ErrInvalid
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 || strconv.Itoa(port) != u.Port() {
		return "", storagecapacity.ErrInvalid
	}
	transport := awshttp.NewBuildableClient().GetTransport()
	transport.Proxy = nil
	client, err := madmin.NewWithOptions(u.Host, &madmin.Options{
		Creds: credentials.NewStaticV4(c.AccessKeyID, c.SecretAccessKey, ""), Secure: u.Scheme == "https",
		Transport: sourceMinIOInfoTransport{base: transport, origin: u.Scheme + "://" + u.Host},
	})
	if err != nil {
		return "", storagecapacity.ErrUnavailable
	}
	readContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	info, err := client.ServerInfo(readContext)
	if err != nil || info.DeploymentID != deploymentID {
		return "", storagecapacity.ErrUnavailable
	}
	raw, _ := json.Marshal([]string{"isolated-source-minio", c.Endpoint, c.Region, bucket, deploymentID})
	return storagecapacity.Digest(string(raw)), nil
}

// Keep the SDK's authentication/decoding, with one bounded local read surface.
// Reject before the SDK's http.Client can follow a redirect or replay cookies.
type sourceMinIOInfoTransport struct {
	base   http.RoundTripper
	origin string
}

func (t sourceMinIOInfoTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Method != http.MethodGet || r.URL.Scheme+"://"+r.URL.Host != t.origin ||
		r.URL.Path != "/minio/admin/v3/info" || r.URL.RawPath != "" || r.URL.User != nil {
		return nil, storagecapacity.ErrUnavailable
	}
	r.Header.Del("Cookie")
	response, err := t.base.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK || response.ContentLength > 1<<20 {
		_ = response.Body.Close()
		return nil, storagecapacity.ErrUnavailable
	}
	response.Body = sourceMinIOInfoBody{Reader: io.LimitReader(response.Body, 1<<20), Closer: response.Body}
	return response, nil
}

type sourceMinIOInfoBody struct {
	io.Reader
	io.Closer
}
