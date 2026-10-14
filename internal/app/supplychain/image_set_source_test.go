package supplychainapp

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/imageagent"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"testing"
	"time"
)

type selectedImageSourceFixture struct{ source preparation.AuthorizedSource }

type revisedCollectionFixture struct {
	collection.Repository
	item collection.Item
}

func (f revisedCollectionFixture) ReadItem(context.Context, collection.Scope, string) (collection.Item, error) {
	return f.item, nil
}

type retainedVersionFixture struct{ original, current catalog.PublishedSnapshot }

func (f retainedVersionFixture) GetSnapshot(_ context.Context, _ catalog.SnapshotIdentity, version uint64) (catalog.PublishedSnapshot, error) {
	if version == f.original.Version {
		return f.original, nil
	}
	if version == f.current.Version {
		return f.current, nil
	}
	return catalog.PublishedSnapshot{}, preparation.ErrNotFound
}

func TestRetainedConsumerKeepsOriginalAfterOwnCollectionRevision(t *testing.T) {
	_, scope, records, _, _, _, auth := uploadFixture(t)
	source := records.saved.Source
	original := catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: source.Source.ProductKey}, PublicationID: source.Source.PublicationID, Version: 1, Snapshot: catalog.ProductSnapshot{Title: "original draft source"}}
	current := original
	current.Version = 2
	current.PublicationID = collection.StableID(source.ID, "edit")
	current.Snapshot = catalog.ProductSnapshot{Title: "edited library source"}
	item := collection.Item{ID: source.CollectionItemID, Revision: source.CollectionRevision + 1, Source: collection.Source{Kind: "own", ProductKey: current.Identity.ProductKey, PublicationID: current.PublicationID, Version: 2}}
	collections, err := collection.NewService(revisedCollectionFixture{item: item}, auth, uploadUnusedSources{})
	require.NoError(t, err)
	versions := retainedVersionFixture{original, current}
	collections.WithSnapshots(versions)
	sources := uploadSources{source: source}
	preparations, err := preparation.NewService(sources, collections, auth)
	require.NoError(t, err)
	selector, err := preparation.NewSourceSelector(preparations, collections, sources, versions)
	require.NoError(t, err)
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute)})
	detail, err := collections.ReadItem(ctx, item.ID)
	require.NoError(t, err)
	require.Equal(t, "edited library source", detail.Product.Title)
	selected, err := selector.Select(ctx, source.ID)
	require.NoError(t, err)
	_, retained, snapshot, err := selected.Read(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, retained.Source.Version)
	require.EqualValues(t, 1, snapshot.Version)
	require.Equal(t, "original draft source", snapshot.Snapshot.Title)
}

func (f selectedImageSourceFixture) Select(context.Context, string) (preparation.AuthorizedSource, error) {
	return f.source, nil
}

func TestSupplySetSourceReusesRetainedSourceOwnerAndLiveExecutionAuthorization(t *testing.T) {
	_, scope, records, _, _, _, auth := uploadFixture(t)
	original := catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: records.saved.Source.Source.ProductKey}, PublicationID: records.saved.Source.Source.PublicationID, Version: 1, Snapshot: catalog.ProductSnapshot{Title: "Original", Images: []catalog.Image{{URL: "https://images.example/source.png"}}}}
	collections, err := collection.NewService(uploadCollections{}, auth, uploadUnusedSources{})
	require.NoError(t, err)
	sources := uploadSources{source: records.saved.Source}
	preparations, err := preparation.NewService(sources, collections, auth)
	require.NoError(t, err)
	selector, err := preparation.NewSourceSelector(preparations, collections, sources, uploadCatalog{original})
	require.NoError(t, err)
	selector, err = selector.WithExecution(auth, collection.ExecutionOwnerAuthority{Authorization: auth})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	ctx = authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute)})
	reader := ImageSetSources{ExecutionSources: selector, ExecutionAuthorization: auth, Products: EffectiveProductReader{}}
	identity := imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, TenantID: scope.OrganizationID, UserID: scope.ActorID, MemberID: scope.MemberID, BusinessTaskID: records.saved.Source.ID}
	input := imageagent.PrepareImageSetInput{ContextID: records.saved.Source.ID, Target: imageagent.ImageTargetSelection{Platform: "product"}}
	result, err := reader.ReadImageSetSource(ctx, identity, input)
	require.NoError(t, err)
	require.Equal(t, records.saved.Source.Source.ProductKey, result.Source.ProductID)
	require.Equal(t, records.saved.Source.ID, result.Source.OperationID)
	require.Equal(t, records.saved.Source.Source.PublicationID, result.Source.OriginalPublicationID)
	require.Len(t, result.Catalog.Assets, 1)
	require.Equal(t, collection.StableID(scope.OrganizationID, records.saved.Source.Source.PublicationID, "source-image", result.Catalog.Assets[0].URL), result.Catalog.Assets[0].ID)
	selected, err := selector.SelectForExecution(ctx, scope, input.ContextID)
	require.NoError(t, err)
	selectionReader := SourceImageReader{Sources: selectedImageSourceFixture{selected}, Products: EffectiveProductReader{}, Authorization: auth}
	selection, err := selectionReader.ReadSourceSelection(ctx, asset.SourceSelectionRequest{ItemID: input.ContextID, OriginalPublicationID: original.PublicationID, OriginalSnapshotVersion: 1, EffectiveCatalogVersion: 1, TargetPlatform: "product"})
	require.NoError(t, err, "generic asset approval uses the same explicit source owner")
	require.Equal(t, "product", selection.TargetPlatform)
	require.Equal(t, result.Catalog.Assets[0].ID, selection.Images[0].ID)
	temuSelection, err := selectionReader.ReadSourceSelection(ctx, asset.SourceSelectionRequest{ItemID: input.ContextID, OriginalPublicationID: original.PublicationID, OriginalSnapshotVersion: 1, EffectiveCatalogVersion: 1, TargetPlatform: "temu"})
	require.NoError(t, err, "TEMU source approval must use the current source and original live authority")
	require.Equal(t, "temu", temuSelection.TargetPlatform)
	require.Equal(t, selection.Images, temuSelection.Images)
	auth.denied = true
	_, err = reader.ReadImageSetSource(ctx, identity, input)
	require.Error(t, err, "per-dispatch source permissions must be live")
	auth.denied = false
	identity.MemberID = "replaced-member"
	_, err = reader.ReadImageSetSource(ctx, identity, input)
	require.ErrorIs(t, err, imageagent.ErrIdentityRequired)
}
