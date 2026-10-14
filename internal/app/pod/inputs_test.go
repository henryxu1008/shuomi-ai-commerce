package podapp

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"io"
	"net/http"
	"strings"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/pod"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGeneratedArtworkLayoutUsesApprovedOwnerBytesWithoutURLDownload(t *testing.T) {
	scope := collection.Scope{"org", "actor", "member"}
	pub, action := uuid.NewString(), uuid.NewString()
	url := "https://localhost:33644/image-agent-assets/images/original.png"
	ref := pod.ArtworkReference{Input: pod.InputReference{Source: collection.Source{ProductKey: "generated-product", PublicationID: pub, Version: 1}}, Version: 1, ActionID: action, AssetID: "approved"}
	approved := asset.ApprovedAsset{ID: ref.AssetID, Role: asset.RoleDesign, URL: url, SourceApproval: &asset.SourceApprovalProvenance{OriginalPublicationID: pub, OriginalSnapshotVersion: 1, ActorID: scope.ActorID, MemberID: scope.MemberID, ReferenceHash: asset.ReferenceHash("source", url)}}
	approved.SourceAssetID = "source"
	commit := asset.ApprovalCommit{TenantID: scope.OrganizationID, ProductKey: ref.Input.Source.ProductKey, TargetPlatform: "sds", ActionID: action, SourceSnapshotVersion: 1, Assets: []asset.ApprovedAsset{approved}}
	var content bytes.Buffer
	require.NoError(t, png.Encode(&content, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	reads := 0
	inputs := OriginalInputs{Approvals: approvalReader{commit}, Snapshots: snapshotReader{catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: ref.Input.Source.ProductKey}, PublicationID: pub, Version: 1}}, ImageHTTP: &http.Client{Transport: imageTransport(func(*http.Request) (*http.Response, error) {
		t.Fatal("generated artwork must not be downloaded by URL")
		return nil, nil
	})}, GeneratedArtworkBytes: func(_ context.Context, actual collection.Scope, original pod.ArtworkReference, selected asset.ApprovedAsset, maximum int64) ([]byte, bool, error) {
		require.Equal(t, scope, actual)
		require.Equal(t, ref.Input, original.Input)
		require.Equal(t, approved, selected)
		require.EqualValues(t, pod.MaxArtworkBytes, maximum)
		reads++
		return content.Bytes(), true, nil
	}}
	_, frozen, err := inputs.bytes(context.Background(), scope, ref)
	require.NoError(t, err)
	require.Equal(t, 1, reads)
	require.NotEqual(t, strings.Repeat("0", 64), frozen.Hash)
	inputs.GeneratedArtworkBytes = func(context.Context, collection.Scope, pod.ArtworkReference, asset.ApprovedAsset, int64) ([]byte, bool, error) {
		return nil, true, asset.ErrApprovalConflict
	}
	_, _, err = inputs.bytes(context.Background(), scope, ref)
	require.ErrorIs(t, err, asset.ErrApprovalConflict, "a rejected generated object cannot fall through to a URL download")
}

type approvalReader struct{ commit asset.ApprovalCommit }

func (r approvalReader) ReadApprovalCommit(context.Context, string, string) (asset.ApprovalCommit, error) {
	return r.commit, nil
}

type snapshotReader struct{ snapshot catalog.PublishedSnapshot }

func (r snapshotReader) GetSnapshot(context.Context, catalog.SnapshotIdentity, uint64) (catalog.PublishedSnapshot, error) {
	return r.snapshot, nil
}

type imageTransport func(*http.Request) (*http.Response, error)

func (f imageTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestApprovedArtworkRequiresCompleteImageBeforeAnyProviderUpload(t *testing.T) {
	scope := collection.Scope{"org", "actor", "member"}
	pub := uuid.NewString()
	source := uuid.NewString()
	action := uuid.NewString()
	url := "https://images.example.org/approved.png"
	ref := pod.ArtworkReference{Input: pod.InputReference{ItemID: uuid.NewString(), Revision: 1, Source: collection.Source{ProductKey: "own-product", PublicationID: pub, Version: 1, Kind: "own"}}, Version: 1, ActionID: action, AssetID: "approved"}
	commit := asset.ApprovalCommit{TenantID: scope.OrganizationID, ProductKey: ref.Input.Source.ProductKey, TargetPlatform: "sds", ActionID: action, SourceSnapshotVersion: 1, Assets: []asset.ApprovedAsset{{ID: ref.AssetID, Role: asset.RoleDesign, URL: url, SourceAssetID: source, SourceApproval: &asset.SourceApprovalProvenance{OriginalPublicationID: pub, OriginalSnapshotVersion: 1, ActorID: scope.ActorID, MemberID: scope.MemberID, ReferenceHash: asset.ReferenceHash(source, url)}}}}
	var content bytes.Buffer
	require.NoError(t, png.Encode(&content, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	raw := content.Bytes()[:33]
	inputs := OriginalInputs{Approvals: approvalReader{commit}, Snapshots: snapshotReader{catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: ref.Input.Source.ProductKey}, PublicationID: pub, Version: 1}}, ImageHTTP: &http.Client{Transport: imageTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(raw))}, nil
	})}}
	_, _, e := inputs.bytes(context.Background(), scope, ref)
	require.ErrorIs(t, e, pod.ErrInvalid)
	raw = content.Bytes()
	_, frozen, e := inputs.bytes(context.Background(), scope, ref)
	require.NoError(t, e)
	require.Len(t, frozen.Hash, 64)
	raw = append(append([]byte(nil), raw...), 1)
	_, _, e = inputs.bytes(context.Background(), scope, frozen)
	require.ErrorIs(t, e, pod.ErrConflict)
	commit.Assets[0].SourceApproval.MemberID = "another-member"
	_, _, e = inputs.bytes(context.Background(), scope, ref)
	require.ErrorIs(t, e, pod.ErrForbidden)
}
