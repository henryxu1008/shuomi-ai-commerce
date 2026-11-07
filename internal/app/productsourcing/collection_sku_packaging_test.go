package productsourcing

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"task-processor/internal/listing/record"
	"task-processor/internal/marketplace/shein/draft"
	"task-processor/internal/marketplace/shein/offlinepackage"
	"task-processor/internal/marketplace/shein/workspace"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	collectionhttp "task-processor/internal/product/collection/httpapi"
	"task-processor/internal/product/sourcing"
)

// Reuse the existing disposable PostgreSQL fixture and its runtime grants. Only
// identity and live roles are synthetic; HTTP, Source publication/read integrity,
// Collection history, Catalog and SHEIN construction/diagnosis are production.
func testOwnSKUPackagingHTTPChain(t *testing.T, db *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	permissions, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	revoked := false
	live := liveRolesFunc(func(context.Context, string, string) ([]string, error) {
		if revoked { return nil, nil }
		return []string{"listingkit_admin"}, nil
	})
	authority, err := collection.NewContextAuthorizer(live, permissions)
	require.NoError(t, err)
	repository, err := collectionstore.NewRepository(ctx, db, func(tx *gorm.DB) (collectionstore.OwnPublisher, error) { return NewOwnProductWriter(tx, authority) })
	require.NoError(t, err)
	sources, err := NewPublishedAcquisitionReader(ctx, db, live, permissions)
	require.NoError(t, err)
	service, err := collection.NewService(repository, authority, sources)
	require.NoError(t, err)
	snapshots, err := catalogstore.NewBoundedSnapshotReader(db, sourcing.MaxEncodedSnapshotBytes)
	require.NoError(t, err)
	history, err := NewOwnHistoryReader(db)
	require.NoError(t, err)
	service.WithSnapshots(snapshots).WithOwnHistory(history)
	actor := acquisitionIdentity("packaging-org", "packaging-actor")
	foreign := acquisitionIdentity("packaging-org", "foreign-actor")
	otherOrg := acquisitionIdentity("foreign-org", "packaging-actor")
	router := gin.New()
	for _, route := range collectionhttp.Routes(service, func(ctx context.Context, token string) (context.Context, error) {
		selected := actor
		switch token {
		case "Bearer actor":
		case "Bearer foreign": selected = foreign
		case "Bearer other-org": selected = otherOrg
		default: return nil, collection.ErrForbidden
		}
		identity, ok := authidentity.AuthenticatedIdentityFromContext(selected)
		if !ok { return nil, collection.ErrForbidden }
		return authidentity.WithAuthenticatedIdentity(ctx, identity), nil
	}) { router.Handle(route.Method, route.Path, route.Handler) }
	token := "Bearer actor"
	request := func(method, path, key string, value any, status int) []byte {
		t.Helper()
		var raw []byte
		if value != nil { raw, err = json.Marshal(value); require.NoError(t, err) }
		req := httptest.NewRequest(method, collectionhttp.BasePath+path, bytes.NewReader(raw))
		req.Header.Set("Authorization", token)
		if method == http.MethodPost { req.Header.Set("Content-Type", "application/json"); req.Header.Set("Idempotency-Key", key) }
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		require.Equal(t, status, response.Code, "%s %s: %s", method, path, response.Body.String())
		return append([]byte(nil), response.Body.Bytes()...)
	}
	number := func(value float64) *float64 { return &value }
	product := collection.OwnProduct{Title: "HTTP measured package", Variants: []collection.OwnVariant{
		{SourceID: "red", SKU: "RED", Currency: "CNY", Stock: 0, Packaging: &catalog.SKUPackaging{WeightGrams: number(125.5), LengthCm: number(10), WidthCm: number(8), HeightCm: number(2)}},
		{SourceID: "blue", SKU: "BLUE", Currency: "CNY", Stock: 7, Packaging: &catalog.SKUPackaging{LengthCm: number(20)}},
	}}
	key := uuid.NewString()
	input := collection.Mutation{Action: "create_product", Product: &product}
	var created collection.Receipt
	require.NoError(t, json.Unmarshal(request("POST", "/commands", key, input, 200), &created))
	itemPath := "/items/"+created.ItemID
	var original collection.ItemDetail
	require.NoError(t, json.Unmarshal(request("GET", itemPath, "", nil, 200), &original))
	require.Len(t, original.Product.Variants, 2)
	require.Equal(t, product.Variants[0].Packaging, original.Product.Variants[0].Packaging)
	require.Equal(t, product.Variants[1].Packaging, original.Product.Variants[1].Packaging)
	require.True(t, original.Product.Variants[0].OwnStockDeclared)
	require.NotNil(t, original.Product.Variants[0].Stock)
	require.Zero(t, *original.Product.Variants[0].Stock)
	published, err := history.Read(actor, "packaging-org", original.Item.Source.PublicationID)
	require.NoError(t, err)
	require.Equal(t, original.Product, published.Snapshot)
	require.Equal(t, product.Variants[0].Packaging, published.Envelope.ProductCandidate.Variants[0].Packaging)
	var evidence struct { EnvelopeJSON, SnapshotJSON []byte; EnvelopeHash, SnapshotHash string }
	require.NoError(t, db.Table("product_source_publications").Where("organization_id=? AND publication_id=?", "packaging-org", original.Item.Source.PublicationID).Take(&evidence).Error)
	require.Equal(t, fmt.Sprintf("%x", sha256.Sum256(evidence.EnvelopeJSON)), evidence.EnvelopeHash)
	require.Equal(t, fmt.Sprintf("%x", sha256.Sum256(evidence.SnapshotJSON)), evidence.SnapshotHash)
	build := func(snapshot catalog.ProductSnapshot) (offlinepackage.Package, []byte) {
		t.Helper()
		raw, err := (draft.Builder{}).Build(actor, snapshot, asset.ApprovedAssetInventory{}, record.Input{Country: "US", Language: "en"})
		require.NoError(t, err)
		var pkg offlinepackage.Package
		require.NoError(t, json.Unmarshal(raw, &pkg))
		return pkg, raw
	}
	oldPackage, oldBytes := build(original.Product)
	oldChecks := workspace.BuildOriginalSKUReadinessChecks(&oldPackage)
	require.Len(t, oldChecks, 1)
	require.False(t, oldChecks[0].OK)
	next := product
	next.Variants = []collection.OwnVariant{product.Variants[1], product.Variants[0]}
	next.Variants[1].SKU = "RENAMED-RED"
	next.Variants[1].Packaging = &catalog.SKUPackaging{WeightGrams: number(250), LengthCm: number(10), WidthCm: number(8), HeightCm: number(2)}
	revision := collection.Mutation{Action: "revise_product", ItemID: created.ItemID, ExpectedRevision: original.Item.Revision, ExpectedSourceVersion: original.Item.Source.Version, ExpectedPublicationID: original.Item.Source.PublicationID, Product: &next}
	revisionKey := uuid.NewString()
	var saved, replay collection.Receipt
	require.NoError(t, json.Unmarshal(request("POST", "/commands", revisionKey, revision, 200), &saved))
	require.NoError(t, json.Unmarshal(request("POST", "/commands", revisionKey, revision, 200), &replay))
	require.True(t, replay.Replayed)
	require.Equal(t, saved.ItemID, replay.ItemID)
	conflict := next
	conflict.Variants = append([]collection.OwnVariant(nil), next.Variants...)
	conflict.Variants[1].Packaging = catalog.CloneSKUPackaging(next.Variants[1].Packaging)
	*conflict.Variants[1].Packaging.WeightGrams = 300
	conflicting := revision
	conflicting.Product = &conflict
	request("POST", "/commands", revisionKey, conflicting, 409)
	var current collection.ItemDetail
	require.NoError(t, json.Unmarshal(request("GET", itemPath, "", nil, 200), &current))
	require.Greater(t, current.Item.Source.Version, original.Item.Source.Version)
	require.Equal(t, "blue", current.Product.Variants[0].SourceID)
	require.Equal(t, product.Variants[1].Packaging, current.Product.Variants[0].Packaging)
	require.Equal(t, "red", current.Product.Variants[1].SourceID)
	require.Equal(t, next.Variants[1].Packaging, current.Product.Variants[1].Packaging)
	latest, err := history.Read(actor, "packaging-org", current.Item.Source.PublicationID)
	require.NoError(t, err)
	require.NotEqual(t, published.Receipt.EnvelopeHash, latest.Receipt.EnvelopeHash)
	require.NotEqual(t, published.Receipt.SnapshotHash, latest.Receipt.SnapshotHash)
	var versions collection.OwnHistoryPage
	require.NoError(t, json.Unmarshal(request("GET", itemPath+"/history", "", nil, 200), &versions))
	require.Len(t, versions.Versions, 2)
	require.Equal(t, current.Item.Source.Version, versions.Versions[0].Version)
	var old collection.OwnVersionDetail
	require.NoError(t, json.Unmarshal(request("GET", fmt.Sprintf("%s/history/%d", itemPath, original.Item.Source.Version), "", nil, 200), &old))
	require.Equal(t, original.Product, old.Product)
	retained, err := history.Read(actor, "packaging-org", original.Item.Source.PublicationID)
	require.NoError(t, err)
	require.Equal(t, published, retained)
	_, rebuiltOld := build(old.Product)
	require.JSONEq(t, string(oldBytes), string(rebuiltOld))
	var oldAgain offlinepackage.Package
	require.NoError(t, json.Unmarshal(rebuiltOld, &oldAgain))
	require.Equal(t, oldChecks, workspace.BuildOriginalSKUReadinessChecks(&oldAgain))
	newPackage, _ := build(current.Product)
	skus := newPackage.DraftPayload.SKCList[0].SKUList
	require.Len(t, skus, 2)
	require.Equal(t, "BLUE", skus[0].SupplierSKU)
	require.Zero(t, skus[0].Weight)
	require.Empty(t, skus[0].WeightUnit)
	require.Equal(t, "20", skus[0].Length)
	require.Empty(t, skus[0].Width)
	require.Empty(t, skus[0].Height)
	require.Equal(t, "RENAMED-RED", skus[1].SupplierSKU)
	require.Equal(t, 250.0, skus[1].Weight)
	require.Equal(t, "g", skus[1].WeightUnit)
	require.Equal(t, "10", skus[1].Length)
	require.Equal(t, "8", skus[1].Width)
	require.Equal(t, "2", skus[1].Height)
	require.Equal(t, "cm", skus[1].LengthUnit)
	require.Zero(t, skus[1].StockCount)
	require.True(t, skus[1].OwnStockDeclared)
	checks := workspace.BuildOriginalSKUReadinessChecks(&newPackage)
	require.Len(t, checks, 1)
	require.False(t, checks[0].OK)
	checkJSON, err := json.Marshal(checks)
	require.NoError(t, err)
	for _, text := range []string{"BLUE", "重量待补", "宽待补", "高待补", "库存 0"} { require.Contains(t, string(checkJSON), text) }
	for _, foreignToken := range []string{"Bearer foreign", "Bearer other-org"} {
		token = foreignToken
		request("GET", itemPath, "", nil, 404)
		request("GET", itemPath+"/history", "", nil, 404)
	}
	token = "Bearer actor"
	revoked = true
	request("GET", itemPath, "", nil, 403)
	request("GET", itemPath+"/history", "", nil, 403)
	request("POST", "/commands", uuid.NewString(), revision, 403)
	revoked = false
	request("GET", itemPath, "", nil, 200)
	var verified collection.Receipt
	require.NoError(t, json.Unmarshal(request("GET", "/by-key/"+revisionKey, "", nil, 200), &verified))
	require.Equal(t, saved.OperationID, verified.OperationID)
}
