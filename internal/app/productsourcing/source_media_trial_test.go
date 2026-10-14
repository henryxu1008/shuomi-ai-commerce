package productsourcing

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	s3 "task-processor/internal/integration/s3"
	"task-processor/internal/ledger/storagecapacity"
	"task-processor/internal/product/collection"
)

const sourceMediaTrialBase = "https://localhost:39744/source-media-assets/source-media-trial"

type trialSourceMediaStorage struct {
	*mediaStorage
	url string
}

func (s *trialSourceMediaStorage) PublicURL(key string) string {
	if s.url != "" {
		return s.url
	}
	return sourceMediaTrialBase + "/" + key
}

func TestSourceMediaTrialPolicyRejectsAnyOtherOriginEndpointOrObject(t *testing.T) {
	policy, err := NewIsolatedTrialSourceMediaURLPolicy(sourceMediaTrialBase, "source-media-trial", "http://127.0.0.1:9103")
	require.NoError(t, err)
	key := "product-source/" + strings.Repeat("a", 64) + "/" + strings.Repeat("b", 64)
	require.NoError(t, policy.ValidateURL(key, sourceMediaTrialBase+"/"+key))
	for _, raw := range []string{
		strings.Replace(sourceMediaTrialBase, "39744", "39745", 1) + "/" + key,
		sourceMediaTrialBase + "/" + key + "?download=1", sourceMediaTrialBase + "/" + key + "#x",
		sourceMediaTrialBase + "/" + strings.Replace(key, strings.Repeat("a", 64), strings.Repeat("c", 64), 1),
		sourceMediaTrialBase + "/" + key + "%2fextra",
	} {
		require.Error(t, policy.ValidateURL(key, raw))
	}
	require.Error(t, policy.ValidateURL("product-source/../secret", sourceMediaTrialBase+"/product-source/../secret"))
	for _, tc := range []struct{ base, bucket, endpoint string }{
		{"https://127.0.0.1:39744/source-media-assets/source-media-trial", "source-media-trial", "http://127.0.0.1:9103"},
		{sourceMediaTrialBase + "/", "source-media-trial", "http://127.0.0.1:9103"},
		{sourceMediaTrialBase, "other-bucket", "http://127.0.0.1:9103"},
		{sourceMediaTrialBase, "source-media-trial", "https://storage.example.org"},
		{sourceMediaTrialBase, "source-media-trial", "http://127.0.0.1:9103/path"},
		{sourceMediaTrialBase, "source-media-trial", "http://127.0.0.1:9103?x=1"},
		{sourceMediaTrialBase, "source-media-trial", "http://user:secret@127.0.0.1:9103"},
	} {
		_, err = NewIsolatedTrialSourceMediaURLPolicy(tc.base, tc.bucket, tc.endpoint)
		require.Error(t, err)
	}
}

func TestSourceMediaTrialUploadUsesVerifiedExactObjectAndFreshScope(t *testing.T) {
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	raw := b.Bytes()
	id := collection.MediaIdentity{Hash: storagecapacity.Digest(string(raw)), Bytes: int64(len(raw))}
	storage := &trialSourceMediaStorage{mediaStorage: &mediaStorage{values: map[string]s3.ImmutableObjectPut{}}}
	auth := &mediaAuth{scope: collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}}
	service := SourceMedia{Storage: storage, Authorization: auth}
	_, err := service.Upload(context.Background(), id, raw)
	require.ErrorIs(t, err, collection.ErrUnavailable)
	require.Zero(t, storage.puts, "default remains public HTTPS")
	service.TrialURLs, err = NewIsolatedTrialSourceMediaURLPolicy(sourceMediaTrialBase, "source-media-trial", "http://127.0.0.1:9103")
	require.NoError(t, err)
	result, err := service.Upload(context.Background(), id, raw)
	require.NoError(t, err)
	require.Equal(t, sourceMediaTrialBase+"/"+mediaKey(auth.scope, id), result.URL)
	_, err = service.Read(context.Background(), id)
	require.NoError(t, err)
	storage.url = sourceMediaTrialBase + "/product-source/" + strings.Repeat("f", 64) + "/" + id.Hash
	_, err = service.Upload(context.Background(), id, raw)
	require.ErrorIs(t, err, collection.ErrUnavailable)
	require.Equal(t, 1, storage.puts, "wrong scope URL cannot dispatch")
	storage.url = ""
	auth.scope.ActorID = "other"
	_, err = service.Read(context.Background(), id)
	require.ErrorIs(t, err, collection.ErrNotFound)
}
