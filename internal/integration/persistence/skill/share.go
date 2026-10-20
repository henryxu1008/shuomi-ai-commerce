package skillpersistence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	d "task-processor/internal/skill"
	"time"
)

type shareRow struct {
	ID, SourceSkillID, SourceDigest, SubmittedBy, OrganizationID, State string
	SourceVersion                                                       uint64
	Consent                                                             bool
	ResultSkillID                                                       *string
	CreatedAt, UpdatedAt                                                time.Time
}

func (shareRow) TableName() string { return "skill_capability.share_requests" }
func (r shareRow) view() d.ShareRequest {
	result := ""
	if r.ResultSkillID != nil {
		result = *r.ResultSkillID
	}
	return d.ShareRequest{ID: r.ID, Source: agent.SkillVersionRef{Kind: "skill-version-v1", ID: r.SourceSkillID, OwnerKind: d.UserOwner, OwnerID: r.SubmittedBy, Version: r.SourceVersion, Digest: r.SourceDigest, PolicyVersion: d.MethodPolicyVersion}, SubmittedBy: r.SubmittedBy, OrganizationID: r.OrganizationID, Consent: r.Consent, State: r.State, ResultID: result, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}
func readShare(db *gorm.DB, id string, lock bool) (shareRow, error) {
	var row shareRow
	sql := "SELECT * FROM skill_capability.share_requests WHERE id=?"
	if lock {
		sql += " FOR UPDATE"
	}
	q := db.Raw(sql, id).Scan(&row)
	if q.Error != nil {
		return row, q.Error
	}
	if q.RowsAffected != 1 {
		return row, d.ErrNotFound
	}
	if !row.view().Source.Valid() || !row.Consent {
		return row, d.ErrUnavailable
	}
	return row, nil
}
func (s *Store) shareCommand(ctx context.Context, p d.Principal, o d.Owner, key string, input any, resolve d.ResolvePrincipal, apply func(*gorm.DB, string) (d.ShareRequest, error)) (d.ShareRequest, error) {
	var result d.ShareRequest
	if !idOK(key) || resolve == nil || !agent.ValidID(p.UserID) || !o.Valid() {
		return result, d.ErrInvalid
	}
	raw, _ := json.Marshal(input)
	sum := sha256.Sum256(raw)
	fingerprint := hex.EncodeToString(sum[:])
	identity, _ := json.Marshal([]string{o.Kind, o.ID, p.UserID, key})
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, e := resolve(ctx)
		if e != nil {
			return e
		}
		if current != p {
			return d.ErrForbidden
		}
		if e = tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", string(identity)).Error; e != nil {
			return e
		}
		var old struct {
			Fingerprint string
			Receipt     []byte
		}
		q := tx.Raw("SELECT fingerprint,receipt FROM skill_capability.commands WHERE owner_kind=? AND owner_id=? AND actor_id=? AND key=?", o.Kind, o.ID, p.UserID, key).Scan(&old)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected == 1 {
			if old.Fingerprint != fingerprint {
				return d.ErrConflict
			}
			if json.Unmarshal(old.Receipt, &result) != nil {
				return d.ErrUnavailable
			}
			return nil
		}
		id := uuid.NewSHA1(uuid.NameSpaceOID, append([]byte("skill-share:"), identity...)).String()
		result, e = apply(tx, id)
		if e != nil {
			return e
		}
		receipt, _ := json.Marshal(result)
		return tx.Exec("INSERT INTO skill_capability.commands(owner_kind,owner_id,actor_id,key,fingerprint,receipt) VALUES(?,?,?,?,?,?)", o.Kind, o.ID, p.UserID, key, fingerprint, receipt).Error
	})
	return result, err
}
func (s *Store) Share(ctx context.Context, p d.Principal, key, id string, version uint64, consent bool, resolve d.ResolvePrincipal) (d.ShareRequest, error) {
	if !consent || !idOK(id) || version == 0 || version >= 1<<63 || !agent.ValidID(p.OrganizationID) || !p.UseOrganization {
		return d.ShareRequest{}, d.ErrInvalid
	}
	return s.shareCommand(ctx, p, d.Owner{Kind: d.UserOwner, ID: p.UserID}, key, []any{"share", id, version, p.OrganizationID, consent}, resolve, func(tx *gorm.DB, requestID string) (d.ShareRequest, error) {
		source, e := readVersion(tx, id, version, true)
		if e != nil {
			return d.ShareRequest{}, e
		}
		if source.Metadata.Owner != (d.Owner{Kind: d.UserOwner, ID: p.UserID}) || !d.Authorized(p, source, d.Copy) {
			return d.ShareRequest{}, d.ErrForbidden
		}
		now := time.Now().UTC()
		row := shareRow{ID: requestID, SourceSkillID: id, SourceVersion: version, SourceDigest: source.Package.Digest, SubmittedBy: p.UserID, OrganizationID: p.OrganizationID, Consent: true, State: "SUBMITTED", CreatedAt: now, UpdatedAt: now}
		return row.view(), tx.Create(&row).Error
	})
}
func (s *Store) ReadShared(ctx context.Context, p d.Principal, id string) (d.Version, error) {
	if !idOK(id) || !p.ConfigureOrganization || !agent.ValidID(p.OrganizationID) {
		return d.Version{}, d.ErrForbidden
	}
	var out d.Version
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, e := readShare(tx, id, false)
		if e != nil {
			return e
		}
		if row.OrganizationID != p.OrganizationID {
			return d.ErrNotFound
		}
		// Use the same resource-first ordering as approve/withdraw/revoke.
		var lockID string
		if e = tx.Raw("SELECT id FROM skill_capability.skills WHERE id=? FOR SHARE", row.SourceSkillID).Scan(&lockID).Error; e != nil {
			return e
		}
		row, e = readShare(tx, id, true)
		if e != nil {
			return e
		}
		if row.State != "SUBMITTED" && row.State != "APPROVED" {
			return d.ErrForbidden
		}
		out, e = readVersion(tx, row.SourceSkillID, row.SourceVersion, false)
		if e != nil {
			return e
		}
		if out.Ref() != row.view().Source || out.State == d.Revoked {
			return d.ErrForbidden
		}
		return nil
	})
	return out, err
}
func (s *Store) DecideShare(ctx context.Context, p d.Principal, key, id, action string, resolve d.ResolvePrincipal) (d.ShareRequest, error) {
	if !idOK(id) {
		return d.ShareRequest{}, d.ErrInvalid
	}
	owner := d.Owner{Kind: d.UserOwner, ID: p.UserID}
	if action == "approve" || action == "reject" {
		if !p.ConfigureOrganization || !agent.ValidID(p.OrganizationID) {
			return d.ShareRequest{}, d.ErrForbidden
		}
		owner = d.Owner{Kind: d.OrganizationOwner, ID: p.OrganizationID}
	} else if action != "withdraw" {
		return d.ShareRequest{}, d.ErrInvalid
	}
	return s.shareCommand(ctx, p, owner, key, []any{action, id}, resolve, func(tx *gorm.DB, _ string) (d.ShareRequest, error) {
		row, e := readShare(tx, id, false)
		if e != nil {
			return d.ShareRequest{}, e
		}
		if action == "withdraw" && row.SubmittedBy != p.UserID || action != "withdraw" && row.OrganizationID != p.OrganizationID {
			return d.ShareRequest{}, d.ErrNotFound
		}
		source, e := readVersion(tx, row.SourceSkillID, row.SourceVersion, true)
		if e != nil {
			return d.ShareRequest{}, e
		}
		row, e = readShare(tx, id, true)
		if e != nil {
			return d.ShareRequest{}, e
		}
		if action == "approve" && row.State == "APPROVED" {
			return row.view(), nil
		}
		if row.State != "SUBMITTED" {
			return d.ShareRequest{}, d.ErrConflict
		}
		now := time.Now().UTC()
		var resultID *string
		state := "WITHDRAWN"
		if action == "reject" {
			state = "REJECTED"
		}
		if action == "approve" {
			if source.Ref() != row.view().Source || source.State == d.Revoked || !source.AllowCopy || !source.AllowExport {
				return d.ShareRequest{}, d.ErrForbidden
			}
			copyID := uuid.NewSHA1(uuid.NameSpaceOID, []byte("skill-approved-copy:"+row.ID)).String()
			resultID = &copyID
			state = "APPROVED"
			ref := source.Ref()
			copy := d.Version{Metadata: d.Metadata{ID: copyID, Owner: owner, Creator: p.UserID, HeadVersion: 1, Revision: 1, CreatedAt: now, UpdatedAt: now}, Number: 1, Package: source.Package, State: d.Draft, Visibility: d.Private, AllowCopy: source.AllowCopy, AllowExport: source.AllowExport, Source: &ref, CreatedAt: now}
			skill := skillRow{ID: copyID, OwnerKind: owner.Kind, OwnerID: owner.ID, Creator: p.UserID, HeadVersion: 1, Revision: 1, CreatedAt: now, UpdatedAt: now}
			if e = tx.Create(&skill).Error; e != nil {
				return d.ShareRequest{}, e
			}
			if e = insertVersion(tx, copy); e != nil {
				return d.ShareRequest{}, e
			}
		}
		if e = tx.Exec("UPDATE skill_capability.share_requests SET state=?,result_skill_id=?,updated_at=? WHERE id=?", state, resultID, now, id).Error; e != nil {
			return d.ShareRequest{}, e
		}
		row.State, row.ResultSkillID, row.UpdatedAt = state, resultID, now
		return row.view(), nil
	})
}
func (s *Store) Shares(ctx context.Context, p d.Principal, organization bool) ([]d.ShareRequest, error) {
	if !agent.ValidID(p.UserID) {
		return nil, d.ErrForbidden
	}
	var rows []shareRow
	query := s.db.WithContext(ctx)
	if organization {
		if !p.ConfigureOrganization || !agent.ValidID(p.OrganizationID) {
			return nil, d.ErrForbidden
		}
		query = query.Where("organization_id=?", p.OrganizationID)
	} else {
		query = query.Where("submitted_by=?", p.UserID)
	}
	if e := query.Order("created_at DESC,id").Limit(50).Find(&rows).Error; e != nil {
		return nil, e
	}
	out := make([]d.ShareRequest, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.view())
	}
	return out, nil
}
