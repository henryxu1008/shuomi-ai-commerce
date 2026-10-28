package sourcing

import (
	"context"
	"errors"
	"testing"

	"task-processor/internal/product/catalog"
)

type sourcingSnapshotWriter struct {
	request catalog.PublishRequest
}

func (w *sourcingSnapshotWriter) PublishSnapshot(_ context.Context, request catalog.PublishRequest) (catalog.PublishedSnapshot, error) {
	w.request = request
	return catalog.PublishedSnapshot{
		Identity: request.Identity, Version: 1, PublicationID: request.PublicationID, Snapshot: request.Snapshot,
	}, nil
}

func TestToSnapshotAndCatalogPublisherConvertsStructuredEnvelopeAndPublishesExactIdentity(t *testing.T) {
	t.Parallel()

	writer := &sourcingSnapshotWriter{}
	catalogPublisher, err := catalog.NewPublisher(writer)
	if err != nil {
		t.Fatalf("catalog.NewPublisher() error = %v", err)
	}

	snapshot, err := ToSnapshot(SourceEnvelope{
		Identity:         SourceIdentity{SourceType: "crawler", SourcePlatform: "amazon", SourceID: "B001"},
		RawReference:     RawSourceReference{ReferenceID: "raw-1"},
		ProductCandidate: ProductCandidate{Title: "Bottle", Attributes: map[string]string{"material": "steel", "color": "black"}},
		Warnings:         []SourceWarning{{Code: "missing_brand", Field: "brand", Message: "brand unavailable"}},
	})
	if err != nil {
		t.Fatalf("ToSnapshot() error = %v", err)
	}
	got, err := catalogPublisher.Publish(context.Background(), catalog.PublishRequest{
		Identity: catalog.SnapshotIdentity{TenantID: "tenant-a", ProductKey: "product-1"}, PublicationID: "source-run-1", Snapshot: snapshot,
	})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if got.Version != 1 || got.Snapshot.Title != "Bottle" {
		t.Fatalf("Publish() = %+v, want versioned Bottle snapshot", got)
	}
	if writer.request.Identity != (catalog.SnapshotIdentity{TenantID: "tenant-a", ProductKey: "product-1"}) || writer.request.PublicationID != "source-run-1" {
		t.Fatalf("catalog publish request identity = %+v", writer.request)
	}
	if len(writer.request.Snapshot.Attributes) != 2 || writer.request.Snapshot.Attributes[0].Name != "color" || writer.request.Snapshot.Attributes[1].Name != "material" {
		t.Fatalf("snapshot attributes = %+v, want deterministic order", writer.request.Snapshot.Attributes)
	}
}

func TestToSnapshotFailsClosedBeforeCatalogWhenSourceIdentityIsIncomplete(t *testing.T) {
	t.Parallel()

	writer := &sourcingSnapshotWriter{}
	_, err := catalog.NewPublisher(writer)
	if err != nil {
		t.Fatalf("catalog.NewPublisher() error = %v", err)
	}

	_, err = ToSnapshot(SourceEnvelope{Identity: SourceIdentity{SourceType: "crawler", SourcePlatform: "amazon"}})
	if !errors.Is(err, ErrSourceIdentityRequired) {
		t.Fatalf("Publish() error = %v, want ErrSourceIdentityRequired", err)
	}
	if writer.request.PublicationID != "" {
		t.Fatalf("catalog writer called for incomplete source identity: %+v", writer.request)
	}
}
