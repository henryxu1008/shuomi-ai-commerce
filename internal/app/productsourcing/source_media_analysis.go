package productsourcing

import (
	"context"
	"net/url"
	"strings"
	"task-processor/internal/authidentity"
	"task-processor/internal/product/collection"
)

// ReadOwnIntake reads an upload owned by the current authorized actor before
// the first product publication. The existing URL and byte checks stay shared.
func (m SourceMedia) ReadOwnIntake(ctx context.Context, rawURL string, maximum int64) ([]byte, bool, error) {
	scope, err := m.scope(ctx)
	if err != nil {
		return nil, false, err
	}
	return m.ReadAuthorizedPublished(ctx, scope.ActorID, rawURL, maximum)
}

// ReadAuthorizedPublished extracts the existing immutable media read for
// an already-authorized exact Catalog/ApprovedAsset fact. The caller derives
// originalActor from that fact, never HTTP input. No upload/manage grant,
// storage accounting or public-URL fallback is acquired by this read port.
func (m SourceMedia) ReadAuthorizedPublished(ctx context.Context, originalActor, rawURL string, maximum int64) ([]byte, bool, error) {
	storage := m.Storage
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, true, collection.ErrInvalid
	}
	if !strings.Contains(u.Path, "/product-source/") {
		return nil, false, nil
	}
	if ctx == nil || storage == nil || maximum < 1 || maximum > collection.MaxMediaBytes {
		return nil, true, collection.ErrUnavailable
	}
	i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || i.TenantID != i.EffectiveOrganizationID || i.EffectiveMemberID == "" {
		return nil, true, collection.ErrForbidden
	}
	owner := collection.Scope{OrganizationID: i.TenantID, ActorID: originalActor, MemberID: i.EffectiveMemberID}
	if owner.Validate() != nil {
		return nil, true, collection.ErrForbidden
	}
	parts := strings.Split(u.Path, "/")
	hash := parts[len(parts)-1]
	if !mediaHash.MatchString(hash) {
		return nil, true, collection.ErrInvalid
	}
	key := mediaKey(owner, collection.MediaIdentity{Hash: hash})
	if storage.PublicURL(key) != rawURL {
		return nil, true, collection.ErrForbidden
	}
	raw, meta, err := storage.ReadObject(ctx, key, maximum)
	if err != nil {
		return nil, true, collection.ErrUnavailable
	}
	if !meta.Exists || meta.ContentLength != int64(len(raw)) || len(raw) == 0 || int64(len(raw)) > maximum {
		return nil, true, collection.ErrConflict
	}
	image, err := inspectSourceMedia(collection.MediaIdentity{Hash: hash, Bytes: meta.ContentLength}, raw)
	if err != nil || image.MediaType != meta.ContentType {
		return nil, true, collection.ErrConflict
	}
	return append([]byte(nil), raw...), true, nil
}
