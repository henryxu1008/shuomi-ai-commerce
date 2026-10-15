package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
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

const MinIOCapacityIdentityProvider = "minio-deployment"

// ValidateCapacityIdentity is configuration-only: cloud storage never falls
// back to a local identity when its canonical owner cannot be established.
func (c ClientConfig) ValidateCapacityIdentity(bucket, mode string) error {
	provider, pin := c.CapacityIdentityProvider, c.CapacityMinIODeploymentID
	if provider == "" && pin == "" {
		return nil
	}
	id, err := uuid.Parse(pin)
	u, parseErr := url.Parse(c.Endpoint)
	if provider != MinIOCapacityIdentityProvider || mode != "aws" || !c.UsePathStyle ||
		err != nil || id == uuid.Nil || id.String() != pin || !storagecapacity.ValidID(bucket, 128) ||
		strings.TrimSpace(c.Region) == "" || strings.TrimSpace(c.AccessKeyID) == "" || strings.TrimSpace(c.SecretAccessKey) == "" ||
		parseErr != nil || u.Scheme != "http" && u.Scheme != "https" || u.User != nil || u.Path != "" || u.RawPath != "" ||
		u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || u.String() != c.Endpoint {
		return storagecapacity.ErrInvalid
	}
	port, err := strconv.Atoi(u.Port())
	ip := net.ParseIP(u.Hostname())
	local := ip != nil && ip.IsLoopback() && ip.String() == u.Hostname()
	if !local && u.Hostname() != "knowledge-objects" && u.Hostname() != "ecoservices-objects" ||
		err != nil || port < 1 || port > 65535 || strconv.Itoa(port) != u.Port() {
		return storagecapacity.ErrInvalid
	}
	return nil
}

func resolveMinIOCapacityNamespace(ctx context.Context, c ClientConfig, bucket, mode string) (string, error) {
	if err := c.ValidateCapacityIdentity(bucket, mode); err != nil {
		return "", err
	}
	pin := c.CapacityMinIODeploymentID
	if err := verifyMinIODeployment(ctx, c, pin); err != nil {
		return "", err
	}
	raw, _ := json.Marshal([]string{"local-minio-deployment-v1", c.Endpoint, c.Region, bucket, pin})
	return storagecapacity.Digest(string(raw)), nil
}

// Shared authenticated read, also used by the original source-only trial.
// Each caller keeps its own explicit admission and namespace identity rules.
func verifyMinIODeployment(ctx context.Context, c ClientConfig, pin string) error {
	if ctx == nil {
		return storagecapacity.ErrUnavailable
	}
	u, err := url.Parse(c.Endpoint)
	if err != nil {
		return storagecapacity.ErrInvalid
	}
	transport := awshttp.NewBuildableClient().GetTransport()
	transport.Proxy = nil
	client, err := madmin.NewWithOptions(u.Host, &madmin.Options{
		Creds: credentials.NewStaticV4(c.AccessKeyID, c.SecretAccessKey, ""), Secure: u.Scheme == "https",
		Transport: minIOInfoTransport{base: transport, origin: u.Scheme + "://" + u.Host},
	})
	if err != nil {
		return storagecapacity.ErrUnavailable
	}
	readContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	info, err := client.ServerInfo(readContext)
	if err != nil || info.DeploymentID != pin {
		return storagecapacity.ErrUnavailable
	}
	return nil
}

type minIOInfoTransport struct {
	base   http.RoundTripper
	origin string
}

func (t minIOInfoTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Method != http.MethodGet || r.URL.Scheme+"://"+r.URL.Host != t.origin ||
		r.URL.Path != "/minio/admin/v3/info" || r.URL.RawPath != "" || r.URL.User != nil {
		return nil, storagecapacity.ErrUnavailable
	}
	r.Header.Del("Cookie")
	response, err := t.base.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.ContentLength > 1<<20 {
		return nil, storagecapacity.ErrUnavailable
	}
	// The SDK decodes only its first JSON value. Require a complete, single
	// document after bounding the entire response, including chunked bodies.
	body, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 || !json.Valid(body) {
		return nil, storagecapacity.ErrUnavailable
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	return response, nil
}
