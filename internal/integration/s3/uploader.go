package s3

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Uploader uploads objects through an S3-compatible API.
type Uploader struct {
	s3Client             S3ObjectAPI
	bucket               string
	publicBase           string
	endpoint             string
	usePathStyle         bool
	artifactCapabilities ArtifactStorageCapabilities
}

// S3ObjectAPI is the narrow AWS SDK v2 request boundary used by Uploader.
type S3ObjectAPI interface {
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	HeadObject(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	CopyObject(context.Context, *s3.CopyObjectInput, ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
}

type ObjectInspection struct {
	Exists               bool
	ContentLength        int64
	ContentType          string
	Metadata             map[string]string
	ServerChecksumSHA256 string
	ETag                 string
}

type ImmutableObjectPut struct {
	Key         string
	Data        []byte
	ContentType string
	SHA256      string
	SizeBytes   int64
}

type ImmutableObjectCopy struct {
	SourceKey   string
	Destination ImmutableObjectPut
}

type ArtifactStorageMode string

const (
	ArtifactStorageModeAWS ArtifactStorageMode = "aws"
	ArtifactStorageModeCOS ArtifactStorageMode = "cos"
)

// ArtifactStorageCapabilities must be explicit for the new durable artifact
// path. COS requires an operator-confirmed immutable policy on a non-versioned
// bucket because x-cos-forbid-overwrite is not sufficient for versioned buckets.
type ArtifactStorageCapabilities struct {
	Mode                                 ArtifactStorageMode
	COSImmutableNonVersionedBucketPolicy bool
}

type UploaderOptions struct {
	Bucket               string
	PublicBase           string
	Endpoint             string
	UsePathStyle         bool
	ArtifactCapabilities ArtifactStorageCapabilities
}

// NewUploaderWithOptions constructs the sole uploader implementation. The
// narrow client interface keeps SDK fakes in this integration package.
func NewUploaderWithOptions(s3Client S3ObjectAPI, opts UploaderOptions) (*Uploader, error) {
	if s3Client == nil || isNilReflectValue(reflect.ValueOf(s3Client)) {
		return nil, fmt.Errorf("s3 client cannot be nil")
	}
	bucket := strings.TrimSpace(opts.Bucket)
	if bucket == "" {
		return nil, fmt.Errorf("s3 bucket cannot be empty")
	}
	return &Uploader{
		s3Client:             s3Client,
		bucket:               bucket,
		publicBase:           opts.PublicBase,
		endpoint:             opts.Endpoint,
		usePathStyle:         opts.UsePathStyle,
		artifactCapabilities: opts.ArtifactCapabilities,
	}, nil
}

func isNilReflectValue(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// PutImmutable writes an object only when its deterministic key is unused. It
// records application metadata as a fallback for S3-compatible endpoints that
// do not expose a server checksum through HeadObject.
func (u *Uploader) PutImmutable(ctx context.Context, object ImmutableObjectPut) error {
	if err := validateImmutableObjectIdentity(object); err != nil {
		return err
	}
	if int64(len(object.Data)) != object.SizeBytes {
		return fmt.Errorf("immutable S3 object body length does not match metadata")
	}
	checksumEnabled, options, err := u.immutableWriteOptions()
	if err != nil {
		return err
	}
	input := &s3.PutObjectInput{
		Bucket:        aws.String(u.bucket),
		Key:           aws.String(object.Key),
		Body:          bytes.NewReader(object.Data),
		ContentLength: aws.Int64(object.SizeBytes),
		ContentType:   aws.String(object.ContentType),
		IfNoneMatch:   aws.String("*"),
		Metadata: map[string]string{
			"sha256":     strings.ToLower(object.SHA256),
			"size-bytes": strconv.FormatInt(object.SizeBytes, 10),
		},
	}
	if checksumEnabled {
		checksum, err := sha256Base64(object.SHA256)
		if err != nil {
			return err
		}
		input.ChecksumAlgorithm = types.ChecksumAlgorithmSha256
		input.ChecksumSHA256 = aws.String(checksum)
	}
	_, err = u.s3Client.PutObject(ctx, input, options...)
	if err != nil {
		return fmt.Errorf("put immutable S3 object %q: %w", object.Key, err)
	}
	return nil
}

// InspectObject returns typed object metadata. Only explicit AWS/Smithy
// not-found errors become Exists=false; every other HeadObject error remains
// observable to recovery callers.
func (u *Uploader) InspectObject(ctx context.Context, key string) (ObjectInspection, error) {
	checksumEnabled, err := u.strictArtifactCapability()
	if err != nil {
		return ObjectInspection{}, err
	}
	input := &s3.HeadObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
	}
	if checksumEnabled {
		input.ChecksumMode = types.ChecksumModeEnabled
	}
	output, err := u.s3Client.HeadObject(ctx, input)
	if err != nil {
		if isS3NotFound(err) {
			return ObjectInspection{}, nil
		}
		return ObjectInspection{}, fmt.Errorf("head S3 object %q: %w", key, err)
	}
	metadata := make(map[string]string, len(output.Metadata))
	for name, value := range output.Metadata {
		metadata[strings.ToLower(name)] = value
	}
	return ObjectInspection{
		Exists:               true,
		ContentLength:        aws.ToInt64(output.ContentLength),
		ContentType:          aws.ToString(output.ContentType),
		Metadata:             metadata,
		ServerChecksumSHA256: aws.ToString(output.ChecksumSHA256),
		ETag:                 aws.ToString(output.ETag),
	}, nil
}

// ReadObject returns a bounded object body together with the same integrity
// metadata exposed by InspectObject. Durable recovery callers validate that
// metadata against their content-addressed identity before trusting the bytes.
func (u *Uploader) ReadObject(ctx context.Context, key string, maxBytes int64) ([]byte, ObjectInspection, error) {
	if strings.TrimSpace(key) == "" || key != strings.TrimSpace(key) || maxBytes <= 0 {
		return nil, ObjectInspection{}, fmt.Errorf("invalid bounded S3 object read")
	}
	checksumEnabled, err := u.strictArtifactCapability()
	if err != nil {
		return nil, ObjectInspection{}, err
	}
	input := &s3.GetObjectInput{Bucket: aws.String(u.bucket), Key: aws.String(key)}
	if checksumEnabled {
		input.ChecksumMode = types.ChecksumModeEnabled
	}
	output, err := u.s3Client.GetObject(ctx, input)
	if err != nil {
		if isS3NotFound(err) {
			return nil, ObjectInspection{}, nil
		}
		return nil, ObjectInspection{}, fmt.Errorf("get S3 object %q: %w", key, err)
	}
	if output.Body == nil {
		return nil, ObjectInspection{}, fmt.Errorf("get S3 object %q returned an empty body", key)
	}
	defer output.Body.Close()
	contentLength := aws.ToInt64(output.ContentLength)
	if contentLength <= 0 || contentLength > maxBytes {
		return nil, ObjectInspection{}, fmt.Errorf("S3 object %q exceeds the bounded read contract", key)
	}
	data, err := io.ReadAll(io.LimitReader(output.Body, maxBytes+1))
	if err != nil {
		return nil, ObjectInspection{}, fmt.Errorf("read S3 object %q: %w", key, err)
	}
	if int64(len(data)) != contentLength || int64(len(data)) > maxBytes {
		return nil, ObjectInspection{}, fmt.Errorf("S3 object %q body length does not match metadata", key)
	}
	metadata := make(map[string]string, len(output.Metadata))
	for name, value := range output.Metadata {
		metadata[strings.ToLower(name)] = value
	}
	return data, ObjectInspection{
		Exists:               true,
		ContentLength:        contentLength,
		ContentType:          aws.ToString(output.ContentType),
		Metadata:             metadata,
		ServerChecksumSHA256: aws.ToString(output.ChecksumSHA256),
		ETag:                 aws.ToString(output.ETag),
	}, nil
}

// CopyImmutable copies a verified staged object to an unused deterministic
// destination. Destination preconditions prevent it from overwriting a key
// created by a concurrent or external writer.
func (u *Uploader) CopyImmutable(ctx context.Context, object ImmutableObjectCopy) error {
	if err := validateImmutableObjectIdentity(object.Destination); err != nil {
		return err
	}
	checksumEnabled, options, err := u.immutableWriteOptions()
	if err != nil {
		return err
	}
	input := &s3.CopyObjectInput{
		Bucket:            aws.String(u.bucket),
		Key:               aws.String(object.Destination.Key),
		CopySource:        aws.String(copySource(u.bucket, object.SourceKey)),
		ContentType:       aws.String(object.Destination.ContentType),
		IfNoneMatch:       aws.String("*"),
		MetadataDirective: types.MetadataDirectiveReplace,
		Metadata: map[string]string{
			"sha256":     strings.ToLower(object.Destination.SHA256),
			"size-bytes": strconv.FormatInt(object.Destination.SizeBytes, 10),
		},
	}
	if checksumEnabled {
		input.ChecksumAlgorithm = types.ChecksumAlgorithmSha256
	}
	_, err = u.s3Client.CopyObject(ctx, input, options...)
	if err != nil {
		return fmt.Errorf("copy immutable S3 object %q to %q: %w", object.SourceKey, object.Destination.Key, err)
	}
	return nil
}

// PublicURL returns the externally reachable URL for an object key using the
// configured public-base and endpoint rules.
func (u *Uploader) PublicURL(key string) string {
	return u.resolveObjectURL(key)
}

func (u *Uploader) resolveObjectURL(key string) string {
	fallbackBase := BuildS3PublicBase(u.endpoint, u.bucket, u.usePathStyle)
	fallbackURL := ""
	if fallbackBase != "" {
		fallbackURL = fallbackBase + "/" + key
	} else {
		fallbackURL = fmt.Sprintf("https://%s.s3.amazonaws.com/%s", u.bucket, key)
	}
	return ResolveObjectURL(u.publicBase, key, fallbackURL)
}

func validateImmutableObjectIdentity(object ImmutableObjectPut) error {
	if strings.TrimSpace(object.Key) == "" || object.Key != strings.TrimSpace(object.Key) || object.SizeBytes <= 0 || strings.TrimSpace(object.ContentType) == "" || strings.TrimSpace(object.SHA256) == "" {
		return fmt.Errorf("invalid immutable S3 object")
	}
	return nil
}

func sha256Base64(value string) (string, error) {
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != 32 {
		return "", fmt.Errorf("invalid SHA-256")
	}
	return base64.StdEncoding.EncodeToString(decoded), nil
}

func copySource(bucket, key string) string {
	segments := strings.Split(key, "/")
	for index, segment := range segments {
		segments[index] = strictPathEscape(segment)
	}
	return strictPathEscape(bucket) + "/" + strings.Join(segments, "/")
}

func strictPathEscape(value string) string {
	return strings.ReplaceAll(url.PathEscape(value), "+", "%2B")
}

func (u *Uploader) strictArtifactCapability() (bool, error) {
	switch u.artifactCapabilities.Mode {
	case ArtifactStorageModeAWS:
		return true, nil
	case ArtifactStorageModeCOS:
		if !u.artifactCapabilities.COSImmutableNonVersionedBucketPolicy {
			return false, fmt.Errorf("COS durable artifacts require an explicit immutable non-versioned bucket policy")
		}
		return false, nil
	default:
		return false, fmt.Errorf("durable artifacts require explicit S3/COS storage capabilities")
	}
}

func (u *Uploader) immutableWriteOptions() (bool, []func(*s3.Options), error) {
	checksumEnabled, err := u.strictArtifactCapability()
	if err != nil || u.artifactCapabilities.Mode != ArtifactStorageModeCOS {
		return checksumEnabled, nil, err
	}
	return false, []func(*s3.Options){s3.WithAPIOptions(cosForbidOverwriteMiddleware)}, nil
}

func cosForbidOverwriteMiddleware(stack *middleware.Stack) error {
	return stack.Build.Add(middleware.BuildMiddlewareFunc("cosForbidOverwrite", func(ctx context.Context, input middleware.BuildInput, next middleware.BuildHandler) (middleware.BuildOutput, middleware.Metadata, error) {
		request, ok := input.Request.(*smithyhttp.Request)
		if !ok {
			return middleware.BuildOutput{}, middleware.Metadata{}, fmt.Errorf("COS forbid-overwrite requires HTTP request")
		}
		request.Header.Set("x-cos-forbid-overwrite", "true")
		return next.HandleBuild(ctx, input)
	}), middleware.After)
}

func isS3NotFound(err error) bool {
	var noSuchKey *types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return true
	}
	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return true
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NoSuchKey", "NotFound":
			return true
		}
	}
	return false
}
