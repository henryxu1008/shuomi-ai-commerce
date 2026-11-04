package dataacquisition

import (
	"context"
	"encoding/json"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
)

// Provenance is an internal, live-authorized read of the original saved evidence.
// It does not change the external API's requested-field projection or start work.
type Provenance struct {
	JobID    string
	Source   collection.Source
	Query    Query
	Evidence Evidence
	Checksum string
}
type CapturedEvidenceRepository interface {
	ReadCaptured(context.Context, collection.Scope, string) (Job, Item, error)
}

func (s *Service) Provenance(ctx context.Context, p Principal, source collection.Source, reader CapturedResultReader) (Provenance, error) {
	r := Reader{repo: s.repo, live: s.live}
	return r.Provenance(ctx, p, source, reader)
}

func (s *Reader) Provenance(ctx context.Context, p Principal, source collection.Source, reader CapturedResultReader) (Provenance, error) {
	if ctx == nil || p.Scope.Validate() != nil || p.CredentialID != "" || source.Kind != "amazon_data" || !collection.ValidID(source.PublicationID) || source.OperationID != source.PublicationID || source.ProductKey == "" || source.Version == 0 || reader == nil {
		return Provenance{}, ErrInvalid
	}
	if err := s.live.CheckRead(ctx, p); err != nil {
		return Provenance{}, err
	}
	repo, ok := s.repo.(CapturedEvidenceRepository)
	if !ok {
		return Provenance{}, ErrUnavailable
	}
	job, item, err := repo.ReadCaptured(ctx, p.Scope, source.PublicationID)
	if err != nil {
		return Provenance{}, err
	}
	if job.Scope != p.Scope || !collection.ValidID(job.ID) || item.ID != source.OperationID || item.State != "SAVED" || item.Source == nil || *item.Source != source || item.Evidence == nil || item.Evidence.Validate() != nil || item.ASIN != item.Evidence.ASIN {
		return Provenance{}, ErrNotFound
	}
	if err = reader.Verify(ctx, p.Scope, source, *item.Evidence); err != nil {
		return Provenance{}, err
	}
	raw, err := json.Marshal(item.Evidence)
	if err != nil {
		return Provenance{}, ErrUnavailable
	}
	var evidence Evidence
	if json.Unmarshal(raw, &evidence) != nil {
		return Provenance{}, ErrUnavailable
	}
	if err = s.live.CheckRead(ctx, p); err != nil {
		return Provenance{}, err
	}
	return Provenance{JobID: job.ID, Source: source, Query: job.Query, Evidence: evidence, Checksum: sourcing.RawSnapshotChecksum(string(raw))}, nil
}
