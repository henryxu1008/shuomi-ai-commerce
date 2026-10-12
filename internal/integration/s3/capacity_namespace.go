package s3

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	"strings"
	"task-processor/internal/ledger/storagecapacity"
)

// ResolveCapacityNamespace binds the physical bucket owner, not the caller's
// rotating credentials. Resolve before consulting a prior namespace admission.
func ResolveCapacityNamespace(ctx context.Context, c ClientConfig, bucket, mode string) (string, error) {
	if (mode != "aws" && mode != "cos") || strings.TrimSpace(bucket) == "" {
		return "", storagecapacity.ErrInvalid
	}
	client, err := NewKnowledgeClient(c)
	if err != nil {
		return "", err
	}
	acl, err := client.GetBucketAcl(ctx, &sdk.GetBucketAclInput{Bucket: aws.String(bucket)})
	if err != nil {
		return "", storagecapacity.ErrUnavailable
	}
	if acl == nil || acl.Owner == nil {
		return "", storagecapacity.ErrUnavailable
	}
	owner := aws.ToString(acl.Owner.ID)
	if !storagecapacity.ValidID(owner, 256) || strings.ContainsAny(owner, "\r\n\x00") {
		return "", storagecapacity.ErrUnavailable
	}
	raw, _ := json.Marshal([]string{mode, strings.TrimRight(strings.TrimSpace(c.Endpoint), "/"), strings.TrimSpace(c.Region), strings.TrimSpace(bucket), owner})
	return storagecapacity.Digest(string(raw)), nil
}

// VerifyEmptyCapacityPrefix uses one bounded listing only at first admission.
// It is not a bucket inventory, migration or usage polling implementation.
func VerifyEmptyCapacityPrefix(ctx context.Context, c ClientConfig, bucket, prefix, mode string) error {
	client, e := NewKnowledgeClient(c)
	if e != nil {
		return e
	}
	page, e := client.ListObjectsV2(ctx, &sdk.ListObjectsV2Input{Bucket: aws.String(bucket), Prefix: aws.String(prefix), MaxKeys: aws.Int32(1)})
	if e != nil {
		return e
	}
	if page == nil || len(page.Contents) > 0 || aws.ToBool(page.IsTruncated) {
		return storagecapacity.ErrConflict
	}
	if mode == "aws" {
		// Live-key listing hides noncurrent versions and delete markers. Those
		// are retained bytes too; a reused versioned prefix is not a new namespace.
		versioning, e := client.GetBucketVersioning(ctx, &sdk.GetBucketVersioningInput{Bucket: aws.String(bucket)})
		if e != nil {
			return e
		}
		if versioning == nil {
			return storagecapacity.ErrUnavailable
		}
		if string(versioning.Status) != "" {
			versions, e := client.ListObjectVersions(ctx, &sdk.ListObjectVersionsInput{Bucket: aws.String(bucket), Prefix: aws.String(prefix), MaxKeys: aws.Int32(1)})
			if e != nil {
				return e
			}
			if versions == nil || len(versions.Versions) > 0 || len(versions.DeleteMarkers) > 0 || aws.ToBool(versions.IsTruncated) {
				return storagecapacity.ErrConflict
			}
		}
	}
	return nil
}
