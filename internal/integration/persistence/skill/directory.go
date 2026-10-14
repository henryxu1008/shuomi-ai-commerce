package skillpersistence

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"task-processor/internal/agent"
	d "task-processor/internal/skill"
)

type directoryCursor struct {
	Owner                     d.Owner
	User, Organization, After string
}

func (s *Store) List(ctx context.Context, p d.Principal, o d.Owner, cursor string, size int) ([]d.Version, string, error) {
	if !o.Valid() || !agent.ValidID(p.UserID) || size < 1 || size > 50 {
		return nil, "", d.ErrInvalid
	}
	if o.Kind == d.UserOwner && o.ID != p.UserID || o.Kind == d.OrganizationOwner && o.ID != p.OrganizationID {
		return nil, "", d.ErrNotFound
	}
	after := ""
	if cursor != "" {
		raw, e := base64.RawURLEncoding.DecodeString(cursor)
		var c directoryCursor
		if e != nil || len(raw) > 1024 || json.Unmarshal(raw, &c) != nil || c.Owner != o || c.User != p.UserID || c.Organization != p.OrganizationID || !idOK(c.After) {
			return nil, "", d.ErrInvalid
		}
		after = c.After
	}
	where := "s.owner_kind=? AND s.owner_id=?"
	args := []any{o.Kind, o.ID}
	if after != "" {
		where += " AND s.id>?"
		args = append(args, after)
	}
	switch o.Kind {
	case d.OrganizationOwner:
		if !p.ConfigureOrganization {
			where += " AND ((s.creator=? AND ?) OR (v.visibility='SHARED' AND v.state NOT IN ('DRAFT','REVOKED') AND ?))"
			args = append(args, p.UserID, p.UseOrganization, p.ReadOrganization)
		}
	case d.PlatformOwner:
		if !p.PlatformMaintainer {
			where += " AND v.visibility='SHARED' AND v.state NOT IN ('DRAFT','REVOKED')"
		}
	}
	args = append(args, size+1)
	var rows []struct {
		ID      string
		Version uint64
	}
	e := s.db.WithContext(ctx).Raw("SELECT s.id,max(v.version) AS version FROM skill_capability.skills s JOIN skill_capability.versions v ON v.skill_id=s.id WHERE "+where+" GROUP BY s.id ORDER BY s.id LIMIT ?", args...).Scan(&rows).Error
	if e != nil {
		return nil, "", e
	}
	next := ""
	if len(rows) > size {
		rows = rows[:size]
		raw, _ := json.Marshal(directoryCursor{Owner: o, User: p.UserID, Organization: p.OrganizationID, After: rows[len(rows)-1].ID})
		next = base64.RawURLEncoding.EncodeToString(raw)
	}
	out := make([]d.Version, 0, len(rows))
	for _, row := range rows {
		v, e := s.Read(ctx, p, o, row.ID, row.Version)
		if e != nil {
			return nil, "", e
		}
		v.Package.Files = nil
		out = append(out, v)
	}
	return out, next, nil
}

func (s *Store) Versions(ctx context.Context, p d.Principal, o d.Owner, id string) ([]d.Version, error) {
	if !idOK(id) {
		return nil, d.ErrInvalid
	}
	current, e := readVersion(s.db.WithContext(ctx), id, 0, false)
	if e != nil {
		return nil, e
	}
	if current.Metadata.Owner != o {
		return nil, d.ErrNotFound
	}
	if o.Kind == d.UserOwner && o.ID != p.UserID || o.Kind == d.OrganizationOwner && o.ID != p.OrganizationID {
		return nil, d.ErrNotFound
	}
	if !agent.ValidID(p.UserID) {
		return nil, d.ErrForbidden
	}
	if !o.Valid() {
		return nil, d.ErrInvalid
	}
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	var rows []struct{ Version uint64 }
	if e := s.db.WithContext(ctx).Raw("SELECT version FROM skill_capability.versions WHERE skill_id=? ORDER BY version DESC LIMIT 50", id).Scan(&rows).Error; e != nil {
		return nil, e
	}
	out := []d.Version{}
	for _, row := range rows {
		v, e := readVersion(s.db.WithContext(ctx), id, row.Version, false)
		if e != nil {
			return nil, e
		}
		if !d.Authorized(p, v, d.Read) {
			continue
		}
		v.Package.Files = nil
		out = append(out, v)
	}
	return out, nil
}

// ResolveSelection derives the publisher/owner from storage, never a browser field.
func (s *Store) ResolveSelection(ctx context.Context, p d.Principal, id string, version uint64, digest string) (agent.SkillVersionRef, error) {
	if !idOK(id) || version == 0 || version >= 1<<63 {
		return agent.SkillVersionRef{}, d.ErrInvalid
	}
	v, e := readVersion(s.db.WithContext(ctx), id, version, false)
	if e != nil {
		return agent.SkillVersionRef{}, e
	}
	if v.Package.Digest != digest {
		return agent.SkillVersionRef{}, d.ErrConflict
	}
	ref := v.Ref()
	if _, e = s.ReadForUse(ctx, p, ref); e != nil {
		return agent.SkillVersionRef{}, e
	}
	return ref, nil
}

// Candidates contains only metadata and already-enabled exact versions usable
// in this organization. It is never supplied as a model/tool authority catalog.
func (s *Store) Candidates(ctx context.Context, p d.Principal) ([]d.Version, error) {
	if !agent.ValidID(p.OrganizationID) || !agent.ValidID(p.MemberID) || !p.UseOrganization {
		return nil, d.ErrForbidden
	}
	var allow bool
	if e := s.db.WithContext(ctx).Raw("SELECT allow_personal FROM agent_configuration.skill_admissions WHERE organization_id=? AND agent_id='product.content.agent'", p.OrganizationID).Scan(&allow).Error; e != nil {
		return nil, d.ErrUnavailable
	}
	var rows []struct {
		ID      string
		Version uint64
	}
	e := s.db.WithContext(ctx).Raw(`SELECT s.id,v.version FROM skill_capability.skills s JOIN skill_capability.versions v ON v.skill_id=s.id WHERE v.state='ENABLED' AND ((s.owner_kind='USER' AND s.owner_id=? AND ?) OR (s.owner_kind='ORGANIZATION' AND s.owner_id=? AND v.visibility='SHARED') OR (s.owner_kind='PLATFORM' AND s.owner_id='official' AND v.visibility='SHARED')) ORDER BY s.id,v.version DESC LIMIT 50`, p.UserID, allow, p.OrganizationID).Scan(&rows).Error
	if e != nil {
		return nil, e
	}
	out := []d.Version{}
	for _, row := range rows {
		v, e := readVersion(s.db.WithContext(ctx), row.ID, row.Version, false)
		if e != nil {
			return nil, e
		}
		if !d.Authorized(p, v, d.Use) {
			continue
		}
		v.Package.Files = nil
		out = append(out, v)
	}
	return out, nil
}
