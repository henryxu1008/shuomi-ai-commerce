package s3

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"

	"task-processor/internal/knowledge"
)

type KnowledgeOriginalDeleter struct {
	config  ClientConfig
	bucket  string
	client  *sdk.Client
	objects *KnowledgeStore
}

func NewKnowledgeOriginalDeleter(config ClientConfig, bucket string) (*KnowledgeOriginalDeleter, error) {
	if config.CapacityIdentityProvider != MinIOCapacityIdentityProvider || config.ValidateCapacityIdentity(bucket, "aws") != nil {
		return nil, knowledge.ErrUnavailable
	}
	client, err := NewKnowledgeClient(config)
	if err != nil {
		return nil, err
	}
	uploader, err := NewUploaderWithOptions(client, UploaderOptions{Bucket: bucket, ArtifactCapabilities: ArtifactStorageCapabilities{Mode: ArtifactStorageModeAWS}})
	if err != nil {
		return nil, err
	}
	objects, err := NewKnowledgeStore(uploader)
	if err != nil {
		return nil, err
	}
	return &KnowledgeOriginalDeleter{config: config, bucket: bucket, client: client, objects: objects}, nil
}

func configAbsent(err error, code string) bool {
	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && apiErr.ErrorCode() == code
}

func (s *KnowledgeOriginalDeleter) VerifyNamespace(ctx context.Context, namespace string) error {
	ns, err := ResolveCapacityNamespace(ctx, s.config, s.bucket, "aws")
	if err != nil || ns != namespace {
		return knowledge.ErrUnavailable
	}
	bucket := aws.String(s.bucket)
	version, err := s.client.GetBucketVersioning(ctx, &sdk.GetBucketVersioningInput{Bucket: bucket})
	if err != nil || version == nil || version.Status != "" || version.MFADelete != "" {
		return knowledge.ErrUnavailable
	}
	if _, err = s.client.GetObjectLockConfiguration(ctx, &sdk.GetObjectLockConfigurationInput{Bucket: bucket}); !configAbsent(err, "ObjectLockConfigurationNotFoundError") {
		return knowledge.ErrUnavailable
	}
	if _, err = s.client.GetBucketLifecycleConfiguration(ctx, &sdk.GetBucketLifecycleConfigurationInput{Bucket: bucket}); !configAbsent(err, "NoSuchLifecycleConfiguration") {
		return knowledge.ErrUnavailable
	}
	if _, err = s.client.GetBucketReplication(ctx, &sdk.GetBucketReplicationInput{Bucket: bucket}); !configAbsent(err, "ReplicationConfigurationNotFoundError") {
		return knowledge.ErrUnavailable
	}
	return nil
}

func originalKnowledgeObject(o knowledge.Object) bool {
	parts := strings.Split(o.Key, "/")
	return len(parts) == 3 && parts[0] == "knowledge" && knowledge.ValidID(parts[1]) && parts[2] == o.SHA256 && o.SizeBytes > 0 && o.SizeBytes <= knowledge.MaxUploadBytes
}

func (s *KnowledgeOriginalDeleter) InspectOriginal(ctx context.Context, namespace string, o knowledge.Object) (knowledge.Inspection, error) {
	if !originalKnowledgeObject(o) {
		return knowledge.Inspection{}, knowledge.ErrInvalid
	}
	if err := s.VerifyNamespace(ctx, namespace); err != nil {
		return knowledge.Inspection{}, err
	}
	observed, err := s.objects.Inspect(ctx, o)
	if err != nil || observed.Exists {
		return observed, err
	}
	// A missing HEAD can be a delete marker. Require an authorized, complete
	// version-list response on this exact key in the original bucket.
	versions, err := s.client.ListObjectVersions(ctx, &sdk.ListObjectVersionsInput{Bucket: aws.String(s.bucket), Prefix: aws.String(o.Key), MaxKeys: aws.Int32(1)})
	if err != nil || versions == nil || aws.ToBool(versions.IsTruncated) || len(versions.Versions) != 0 || len(versions.DeleteMarkers) != 0 {
		return knowledge.Inspection{}, knowledge.ErrUnavailable
	}
	if err = s.VerifyNamespace(ctx, namespace); err != nil {
		return knowledge.Inspection{}, err
	}
	return observed, nil
}

func (s *KnowledgeOriginalDeleter) DeleteOriginal(ctx context.Context, namespace string, o knowledge.Object) error {
	if !originalKnowledgeObject(o) {
		return knowledge.ErrInvalid
	}
	if err := s.VerifyNamespace(ctx, namespace); err != nil {
		return err
	}
	_, err := s.client.DeleteObject(ctx, &sdk.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(o.Key)}, func(opt *sdk.Options) { opt.Retryer = aws.NopRetryer{} })
	if err != nil {
		return knowledge.ErrUnavailable
	}
	return nil
}
