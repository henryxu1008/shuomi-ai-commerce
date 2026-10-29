package productsourcing

import (
	"context"
	"database/sql"
	"errors"
	"gorm.io/gorm"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	sourcestore "task-processor/internal/integration/persistence/product/sourcing"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
	"time"
)

// AmazonCaptureAuthorizer is assembled from current native read/live-write
// authority. Neither payload nor stored metadata grants access.
type AmazonCaptureAuthorizer interface {
	Authorize(context.Context, bool) (collection.Scope, error)
}
type AmazonBrowserCaptureService struct {
	db   *gorm.DB
	auth AmazonCaptureAuthorizer
}
type AmazonCaptureResult struct {
	State          string                  `json:"state"`
	CaptureID      string                  `json:"captureID"`
	PublicationID  string                  `json:"publicationID"`
	Key            string                  `json:"key"`
	PayloadSHA256  string                  `json:"payloadSHA256"`
	ProductKey     string                  `json:"productKey"`
	CatalogVersion uint64                  `json:"catalogVersion,string"`
	BatchID        string                  `json:"batchID"`
	CurrentBatchID string                  `json:"currentBatchID"`
	ItemID         string                  `json:"itemID"`
	Archived       bool                    `json:"archived"`
	Replayed       bool                    `json:"replayed"`
	Snapshot       catalog.ProductSnapshot `json:"snapshot"`
}

func NewAmazonBrowserCapture(db *gorm.DB, auth AmazonCaptureAuthorizer) (*AmazonBrowserCaptureService, error) {
	if db == nil || db.Dialector.Name() != "postgres" || auth == nil {
		return nil, sourcing.ErrSourcePublicationUnavailable
	}
	if _, err := sourcestore.NewRepository(db, newCatalogBridge); err != nil {
		return nil, err
	}
	return &AmazonBrowserCaptureService{db: db, auth: auth}, nil
}
func amazonCaptureID(scope collection.Scope, key string) string {
	return collection.StableID(scope.OrganizationID, scope.ActorID, "amazon-browser-capture:v1", key)
}
func amazonProductKey(scope collection.Scope, site, asin string) string {
	return "amazon-" + collection.StableID(scope.OrganizationID, scope.ActorID, "amazon-product", site, asin)
}
func amazonBatch(scope collection.Scope, id, site string) (collection.PublicationBatch, error) {
	return collection.NewPublicationBatch(scope, id, "amazon_data", "Amazon "+site+" · 插件采集")
}

func (s *AmazonBrowserCaptureService) Capture(ctx context.Context, key string, body []byte) (AmazonCaptureResult, error) {
	return s.execute(ctx, key, "", body, true)
}
func (s *AmazonBrowserCaptureService) Verify(ctx context.Context, key string, body []byte) (AmazonCaptureResult, error) {
	return s.execute(ctx, key, "", body, false)
}
func (s *AmazonBrowserCaptureService) ByKey(ctx context.Context, key string) (AmazonCaptureResult, error) {
	return s.execute(ctx, key, "", nil, false)
}
func (s *AmazonBrowserCaptureService) Read(ctx context.Context, id string) (AmazonCaptureResult, error) {
	return s.execute(ctx, "", id, nil, false)
}
func (s *AmazonBrowserCaptureService) Published(ctx context.Context, id string) (AmazonCaptureResult, error) {
	return s.Read(ctx, id)
}

func (s *AmazonBrowserCaptureService) execute(ctx context.Context, key, id string, body []byte, write bool) (out AmazonCaptureResult, err error) {
	if ctx == nil || s == nil || s.auth == nil {
		return out, sourcing.ErrPublicationForbidden
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if key != "" && !collection.ValidID(key) || id != "" && !collection.ValidID(id) || key == "" && id == "" {
		return out, sourcing.ErrInvalidBrowserCapture
	}
	var capture sourcing.AmazonBrowserCapture
	if body != nil {
		capture, err = sourcing.ParseAmazonBrowserCapture(body)
		if err != nil {
			return out, err
		}
	}
	if write && body == nil {
		return out, sourcing.ErrInvalidBrowserCapture
	}
	scope, err := s.auth.Authorize(ctx, write)
	if err != nil {
		return out, err
	}
	if scope.Validate() != nil {
		return out, sourcing.ErrPublicationForbidden
	}
	if id == "" {
		id = amazonCaptureID(scope, key)
	}
	txCtx, txCancel := context.WithTimeout(ctx, sourcing.PublicationTimeout)
	defer txCancel()
	tx := s.db.WithContext(txCtx).Begin(&sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: !write})
	if tx.Error != nil {
		return out, sourcing.ErrSourcePublicationUnavailable
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback().Error
		}
		if err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || txCtx.Err() != nil) {
			err = sourcing.ErrSourcePublicationOutcomeUnknown
		}
	}()
	if err = sourcestore.LockPublication(txCtx, tx, scope.OrganizationID, id); err != nil {
		return out, sourcing.ErrSourcePublicationOutcomeUnknown
	}
	persisted, reference, found, err := s.inspect(txCtx, tx, scope, id)
	if err != nil {
		return out, err
	}
	if found {
		meta := persisted.Envelope.RawReference.Metadata
		if key != "" && key != meta["capture_key"] {
			return out, sourcing.ErrSourcePublicationConflict
		}
		if body != nil && (meta["capture_sha256"] != capture.PayloadSHA256 || meta["site"] != capture.Site || meta["asin"] != capture.ASIN) {
			return out, sourcing.ErrSourcePublicationConflict
		}
		return amazonResult(scope, persisted, reference, true)
	}
	if !write {
		return out, sourcing.ErrSourcePublicationNotFound
	}
	envelope, err := capture.Envelope(id)
	if err != nil {
		return out, err
	}
	envelope.RawReference.Metadata["capture_key"] = key
	envelope.RawReference.Metadata["member_id"] = scope.MemberID
	envelope.RawReference.Metadata["organization_id"] = scope.OrganizationID
	envelope.RawReference.Metadata["actor_id"] = scope.ActorID
	productKey := amazonProductKey(scope, capture.Site, capture.ASIN)
	reader, err := catalogstore.NewBoundedSnapshotReader(tx, sourcing.MaxEncodedSnapshotBytes)
	if err != nil {
		return out, err
	}
	base := uint64(0)
	current, readErr := reader.GetCurrentSnapshot(txCtx, catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: productKey})
	if readErr == nil {
		base = current.Version
	} else if !errors.Is(readErr, catalog.ErrSnapshotNotReady) {
		return out, readErr
	}
	currentScope, err := s.auth.Authorize(txCtx, true)
	if err != nil {
		return out, err
	}
	if currentScope != scope {
		return out, sourcing.ErrPublicationForbidden
	}
	store, err := sourcestore.NewTransactionWriter(tx, newCatalogBridge)
	if err != nil {
		return out, err
	}
	authority := amazonPublicationAuthority{s.auth, scope}
	descriptor := sourcing.ProducerDescriptor{Kind: sourcing.AmazonBrowserProducerKind, Version: "v1"}
	producer, err := sourcing.NewInternalProducer(authority, store, descriptor)
	if err != nil {
		return out, err
	}
	receipt, err := producer.Publish(txCtx, sourcing.PublicationCommand{PublicationID: id, Producer: descriptor, ProductKey: productKey, ExpectedBaseVersion: &base, Envelope: envelope})
	if err != nil {
		return out, err
	}
	batch, err := amazonBatch(scope, id, capture.Site)
	if err != nil {
		return out, err
	}
	_, err = collectionstore.AppendDataPublication(txCtx, tx, scope, batch, collection.Source{ProductKey: receipt.ProductKey, PublicationID: id, Version: receipt.CatalogVersion, OperationID: id, Kind: "amazon_data"}, receipt.PublishedAt)
	if err != nil {
		return out, err
	}
	persisted, reference, found, err = s.inspect(txCtx, tx, scope, id)
	if err != nil {
		return out, err
	}
	if !found {
		return out, sourcing.ErrSourcePublicationStateInvalid
	}
	out, err = amazonResult(scope, persisted, reference, false)
	if err != nil {
		return out, err
	}
	if commitErr := tx.Commit().Error; commitErr != nil {
		return AmazonCaptureResult{}, sourcing.ErrSourcePublicationOutcomeUnknown
	}
	committed = true
	return out, nil
}

type amazonPublicationAuthority struct {
	auth  AmazonCaptureAuthorizer
	scope collection.Scope
}

func (a amazonPublicationAuthority) Authorize(ctx context.Context) (sourcing.PublicationScope, error) {
	scope, err := a.auth.Authorize(ctx, true)
	if err != nil {
		return sourcing.PublicationScope{}, err
	}
	if scope != a.scope {
		return sourcing.PublicationScope{}, sourcing.ErrPublicationForbidden
	}
	return sourcing.PublicationScope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}, nil
}

func (s *AmazonBrowserCaptureService) inspect(ctx context.Context, tx *gorm.DB, scope collection.Scope, id string) (sourcing.PersistedPublication, collectionstore.CaptureReference, bool, error) {
	var empty sourcing.PersistedPublication
	reference, refFound, err := collectionstore.ReadCaptureReference(ctx, tx, scope, id)
	if err != nil {
		return empty, reference, false, err
	}
	count, err := catalogstore.PublicationPresence(ctx, tx, scope.OrganizationID, id)
	if err != nil {
		return empty, reference, false, err
	}
	reader, err := sourcestore.NewTransactionReader(tx, newCatalogBridge)
	if err != nil {
		return empty, reference, false, err
	}
	persisted, err := reader.Read(ctx, scope.OrganizationID, id)
	if errors.Is(err, sourcing.ErrSourcePublicationNotFound) {
		if refFound || count != 0 {
			return empty, reference, false, sourcing.ErrSourcePublicationStateInvalid
		}
		return empty, reference, false, nil
	}
	if err != nil {
		return empty, reference, false, err
	}
	r, m := persisted.Receipt, persisted.Envelope.RawReference.Metadata
	if r.ActorID != scope.ActorID || m["actor_id"] != scope.ActorID || m["organization_id"] != scope.OrganizationID || m["member_id"] != scope.MemberID {
		return empty, reference, false, sourcing.ErrPublicationForbidden
	}
	expectedProduct := amazonProductKey(scope, m["site"], m["asin"])
	if !refFound || count != 1 || r.Producer != (sourcing.ProducerDescriptor{Kind: sourcing.AmazonBrowserProducerKind, Version: "v1"}) || !collection.ValidID(m["capture_key"]) || amazonCaptureID(scope, m["capture_key"]) != id || m["contract_version"] != sourcing.AmazonBrowserContract || m["funding"] != "plugin_free" || r.ProductKey != expectedProduct || persisted.Envelope.Identity.SourcePlatform != "amazon" || persisted.Envelope.Identity.SourceID != m["site"]+":"+m["asin"] || persisted.Envelope.Trace.SourceRunID != "amazon-browser-capture:"+id || reference.MemberID != scope.MemberID || reference.PublicationID != id || reference.ProductKey != r.ProductKey || reference.OriginalVersion != r.CatalogVersion || reference.SourceKind != "amazon_data" || reference.SourceOperationID != id {
		return empty, reference, false, sourcing.ErrSourcePublicationStateInvalid
	}
	return persisted, reference, true, nil
}
func amazonResult(scope collection.Scope, p sourcing.PersistedPublication, ref collectionstore.CaptureReference, replayed bool) (AmazonCaptureResult, error) {
	r, m := p.Receipt, p.Envelope.RawReference.Metadata
	batch, err := amazonBatch(scope, r.PublicationID, m["site"])
	if err != nil {
		return AmazonCaptureResult{}, err
	}
	return AmazonCaptureResult{State: "saved", CaptureID: r.PublicationID, PublicationID: r.PublicationID, Key: m["capture_key"], PayloadSHA256: m["capture_sha256"], ProductKey: r.ProductKey, CatalogVersion: r.CatalogVersion, BatchID: batch.ID, CurrentBatchID: ref.BatchID, ItemID: ref.ID, Archived: ref.ArchivedAt != nil || ref.BatchArchivedAt != nil, Replayed: replayed, Snapshot: p.Snapshot}, nil
}
