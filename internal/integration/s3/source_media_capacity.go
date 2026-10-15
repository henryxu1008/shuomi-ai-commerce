package s3

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"net/url"
	"strconv"
	"strings"
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
		strings.TrimSpace(c.AccessKeyID) == "" || strings.TrimSpace(c.SecretAccessKey) == "" || strings.TrimSpace(c.Region) == "" || c.Region != strings.TrimSpace(c.Region) ||
		u.Hostname() != "127.0.0.1" || u.Scheme != "http" && u.Scheme != "https" ||
		u.Port() == "" || u.User != nil || u.Path != "" || u.RawPath != "" ||
		u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || u.String() != c.Endpoint {
		return "", storagecapacity.ErrInvalid
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 || strconv.Itoa(port) != u.Port() {
		return "", storagecapacity.ErrInvalid
	}
	if err := verifyMinIODeployment(ctx, c, deploymentID); err != nil {
		return "", err
	}
	raw, _ := json.Marshal([]string{"isolated-source-minio", c.Endpoint, c.Region, bucket, deploymentID})
	return storagecapacity.Digest(string(raw)), nil
}
