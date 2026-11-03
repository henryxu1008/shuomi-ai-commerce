package productsourcing

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"image"
	"image/png"
	"task-processor/internal/authidentity"
	s3 "task-processor/internal/integration/s3"
	"task-processor/internal/product/collection"
	"testing"
)

func TestAnalysisReadsAuthorizedPublishedMediaWithoutWriteGrantOrPublicFallback(t *testing.T) {
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	raw := b.Bytes()
	h := sha256.Sum256(raw)
	hash := hex.EncodeToString(h[:])
	owner := collection.Scope{OrganizationID: "org", ActorID: "original-owner", MemberID: "original-member"}
	key := mediaKey(owner, collection.MediaIdentity{Hash: hash})
	storage := &mediaStorage{values: map[string]s3.ImmutableObjectPut{key: {Key: key, Data: raw, SizeBytes: int64(len(raw)), ContentType: "image/png"}}}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "reader", EffectiveMemberID: "reader-member"})
	content, handled, err := (SourceMedia{Storage: storage}).ReadAuthorizedPublished(ctx, "original-owner", storage.PublicURL(key), 1024)
	require.NoError(t, err)
	require.True(t, handled)
	require.Equal(t, raw, content)
	require.Zero(t, storage.puts)
	_, handled, err = (SourceMedia{Storage: storage}).ReadAuthorizedPublished(ctx, "wrong-owner", storage.PublicURL(key), 1024)
	require.True(t, handled)
	require.Error(t, err)
	_, handled, err = (SourceMedia{}).ReadAuthorizedPublished(ctx, "original-owner", storage.PublicURL(key), 1024)
	require.True(t, handled)
	require.Error(t, err)
	_, handled, err = (SourceMedia{Storage: storage}).ReadAuthorizedPublished(ctx, "original-owner", "https://external.example/image.png", 1024)
	require.False(t, handled)
	require.NoError(t, err)
}
