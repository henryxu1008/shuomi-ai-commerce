package productsourcing

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
	"testing"
)

func TestCollectionQuoteProjectionBindsEveryPrivatePublicationIdentity(t *testing.T) {
	scope := collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "current-member"}
	source := collection.Source{ProductKey: "product", PublicationID: "publication", Version: 2}
	publication := sourcing.PersistedPublication{Receipt: sourcing.PublicationReceipt{OrganizationID: "org", ActorID: "actor", ProductKey: "product", PublicationID: "publication", CatalogPublicationID: "publication", CatalogVersion: 2, Producer: sourcing.ProducerDescriptor{Kind: sourcing.BrowserAcquisitionProducerKind, Version: "v2"}}}
	quotes, err := projectCollectionQuotes(scope, source, publication)
	require.NoError(t, err)
	require.Equal(t, "available", quotes.State)
	require.NotNil(t, quotes.Prices)
	for _, mutate := range []func(*sourcing.PublicationReceipt){func(r *sourcing.PublicationReceipt) { r.OrganizationID = "other" }, func(r *sourcing.PublicationReceipt) { r.ActorID = "other" }, func(r *sourcing.PublicationReceipt) { r.ProductKey = "other" }, func(r *sourcing.PublicationReceipt) { r.PublicationID = "other" }, func(r *sourcing.PublicationReceipt) { r.CatalogPublicationID = "other" }, func(r *sourcing.PublicationReceipt) { r.CatalogVersion++ }} {
		changed := publication
		mutate(&changed.Receipt)
		_, err := projectCollectionQuotes(scope, source, changed)
		require.ErrorIs(t, err, collection.ErrUnavailable)
	}
	publication.Receipt.Producer = sourcing.ProducerDescriptor{Kind: sourcing.AmazonBrowserProducerKind, Version: "v1"}
	_, err = projectCollectionQuotes(scope, source, publication)
	require.NoError(t, err)
	publication.Receipt.Producer.Version = "future"
	quotes, err = projectCollectionQuotes(scope, source, publication)
	require.NoError(t, err)
	require.Equal(t, "not_applicable", quotes.State)
}
