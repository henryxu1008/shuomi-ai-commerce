package s3

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	"task-processor/internal/ledger/storagecapacity"
)

func CapacityNamespace(mode, endpoint, region, bucket string) string {
	raw, _ := json.Marshal([]string{mode, endpoint, region, bucket})
	return storagecapacity.Digest(string(raw))
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
