package researchpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sort"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/research"
	"time"
)

type Repository struct{ db *gorm.DB }

func New(ctx context.Context, db *gorm.DB) (*Repository, error) {
	if e := VerifySchema(ctx, db); e != nil {
		return nil, e
	}
	return &Repository{db}, nil
}

type candidateRow struct {
	OrganizationID, ActorID, ProjectID, ID, Site, ASIN string
	SelectedLinkID                                     *string
	State, Reason                                      string
	Revision                                           int64
}

func (candidateRow) TableName() string { return "research_opportunities" }

type linkRow struct {
	OrganizationID, ActorID, ProjectID, ID, CandidateID string
	ReferenceJSON                                       []byte
	Removed                                             bool
	CreatedAt                                           time.Time
}

func (linkRow) TableName() string { return "research_evidence_links" }

type commandRow struct {
	OrganizationID, ActorID, ProjectID, Key, InputHash string
	CandidateID                                        *string
	IntentJSON, ReceiptJSON, DecisionJSON              []byte
	CreatedAt                                          time.Time
}

func (commandRow) TableName() string { return "research_commands" }
func scoped(db *gorm.DB, s collection.Scope, p string) *gorm.DB {
	return db.Where("organization_id=? AND actor_id=? AND project_id=?", s.OrganizationID, s.ActorID, p)
}
func lookup(db *gorm.DB, s collection.Scope, p, key string) (commandRow, error) {
	var row commandRow
	e := scoped(db, s, p).Where("key=?", key).Take(&row).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return row, research.ErrNotFound
	}
	if e != nil {
		return row, research.ErrUnavailable
	}
	return row, nil
}
func decodeReceipt(row commandRow) (research.Receipt, error) {
	var out research.Receipt
	if json.Unmarshal(row.ReceiptJSON, &out) != nil || out.Key != row.Key {
		return out, research.ErrUnavailable
	}
	out.Replayed = true
	return out, nil
}
func (r *Repository) Lookup(ctx context.Context, s collection.Scope, p, key string) (research.Receipt, error) {
	row, e := lookup(r.db.WithContext(ctx), s, p, key)
	if e != nil {
		return research.Receipt{}, e
	}
	return decodeReceipt(row)
}
func (r *Repository) Replay(ctx context.Context, s collection.Scope, p, key string, c research.Command) (research.Receipt, bool, error) {
	row, e := lookup(r.db.WithContext(ctx), s, p, key)
	if errors.Is(e, research.ErrNotFound) {
		return research.Receipt{}, false, nil
	}
	if e != nil {
		return research.Receipt{}, false, e
	}
	if row.InputHash != collection.Digest(c) {
		return research.Receipt{}, true, research.ErrConflict
	}
	out, e := decodeReceipt(row)
	return out, true, e
}
func load(db *gorm.DB, s collection.Scope, p, id string, lock bool) (candidateRow, error) {
	var row candidateRow
	q := scoped(db, s, p).Where("id=?", id)
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	e := q.Take(&row).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return row, research.ErrNotFound
	}
	if e != nil {
		return row, research.ErrUnavailable
	}
	return row, nil
}
func (r *Repository) Load(ctx context.Context, s collection.Scope, p, id string) (research.Candidate, error) {
	row, e := load(r.db.WithContext(ctx), s, p, id, false)
	if e != nil {
		return research.Candidate{}, e
	}
	return details(r.db.WithContext(ctx), s, p, row)
}
func details(db *gorm.DB, s collection.Scope, p string, row candidateRow) (research.Candidate, error) {
	out := research.Candidate{ID: row.ID, ProjectID: p, Site: row.Site, ASIN: row.ASIN, State: row.State, Reason: row.Reason, Revision: row.Revision, Links: []research.Link{}, History: []research.Decision{}}
	if row.SelectedLinkID != nil {
		out.SelectedLinkID = *row.SelectedLinkID
	}
	var links []linkRow
	// Include the exact selected observation even when it is older than the recent window.
	q := scoped(db, s, p).Where("candidate_id=?", row.ID).Where("removed=false")
	if e := q.Order("created_at DESC,id DESC").Limit(20).Find(&links).Error; e != nil {
		return out, research.ErrUnavailable
	}
	found := false
	for _, l := range links {
		found = found || l.ID == out.SelectedLinkID
	}
	if out.SelectedLinkID != "" && !found {
		var l linkRow
		if e := scoped(db, s, p).Where("candidate_id=? AND id=? AND removed=false", row.ID, out.SelectedLinkID).Take(&l).Error; e == nil {
			links = append(links, l)
		} else if !errors.Is(e, gorm.ErrRecordNotFound) {
			return out, research.ErrUnavailable
		}
	}
	for _, l := range links {
		var ref research.EvidenceRef
		if json.Unmarshal(l.ReferenceJSON, &ref) != nil {
			return out, research.ErrUnavailable
		}
		out.Links = append(out.Links, research.Link{ID: l.ID, Ref: ref})
	}
	var commands []commandRow
	if e := scoped(db, s, p).Where("candidate_id=? AND decision_json IS NOT NULL", row.ID).Order("created_at DESC,key DESC").Limit(20).Find(&commands).Error; e != nil {
		return out, research.ErrUnavailable
	}
	for _, c := range commands {
		var d research.Decision
		if json.Unmarshal(c.DecisionJSON, &d) != nil {
			return out, research.ErrUnavailable
		}
		out.History = append(out.History, d)
	}
	return out, nil
}
func (r *Repository) List(ctx context.Context, s collection.Scope, p, after string) ([]research.Candidate, string, error) {
	var rows []candidateRow
	q := scoped(r.db.WithContext(ctx), s, p)
	if after != "" {
		q = q.Where("id>?::uuid", after)
	}
	if e := q.Order("id").Limit(51).Find(&rows).Error; e != nil {
		return nil, "", research.ErrUnavailable
	}
	next := ""
	if len(rows) > 50 {
		rows = rows[:50]
		next = rows[49].ID
	}
	out := []research.Candidate{}
	for _, row := range rows {
		v, e := details(r.db.WithContext(ctx), s, p, row)
		if e != nil {
			return nil, "", e
		}
		out = append(out, v)
	}
	return out, next, nil
}
func (r *Repository) Commit(ctx context.Context, s collection.Scope, p, key string, c research.Command, observations []research.Observation, validation error) (research.Receipt, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return research.Receipt{}, research.ErrUnavailable
	}
	defer tx.Rollback()
	if e := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", collection.Digest([]string{s.OrganizationID, s.ActorID, p, key})).Error; e != nil {
		return research.Receipt{}, research.ErrUnavailable
	}
	existing, e := lookup(tx, s, p, key)
	if e == nil {
		if existing.InputHash != collection.Digest(c) {
			return research.Receipt{}, research.ErrConflict
		}
		return decodeReceipt(existing)
	}
	if !errors.Is(e, research.ErrNotFound) {
		return research.Receipt{}, e
	}
	if validation != nil {
		return research.Receipt{}, validation
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	out := research.Receipt{Key: key, CandidateIDs: []string{}, Revisions: map[string]int64{}}
	var decision *research.Decision
	if c.Action == "link" {
		if len(observations) != len(c.Selections) {
			return out, research.ErrInvalid
		}
		observations = append([]research.Observation(nil), observations...)
		sort.Slice(observations, func(i, j int) bool {
			return collection.StableID(s.OrganizationID, s.ActorID, p, "opportunity", observations[i].Site, observations[i].ASIN) < collection.StableID(s.OrganizationID, s.ActorID, p, "opportunity", observations[j].Site, observations[j].ASIN)
		})
		for _, v := range observations {
			id := collection.StableID(s.OrganizationID, s.ActorID, p, "opportunity", v.Site, v.ASIN)
			row := candidateRow{OrganizationID: s.OrganizationID, ActorID: s.ActorID, ProjectID: p, ID: id, Site: v.Site, ASIN: v.ASIN, State: "TO_RESEARCH", Revision: 1}
			if e := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; e != nil {
				return out, research.ErrUnavailable
			}
			row, e = load(tx, s, p, id, true)
			if e != nil {
				return out, e
			}
			refJSON, _ := json.Marshal(v.Ref)
			linkID := collection.StableID(s.OrganizationID, s.ActorID, p, "observation", v.Ref.Digest())
			link := linkRow{OrganizationID: s.OrganizationID, ActorID: s.ActorID, ProjectID: p, ID: linkID, CandidateID: id, ReferenceJSON: refJSON, CreatedAt: now}
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&link)
			if result.Error != nil {
				return out, research.ErrUnavailable
			}
			if result.RowsAffected == 0 {
				result = scoped(tx, s, p).Model(&linkRow{}).Where("id=? AND candidate_id=? AND removed=true", linkID, id).Update("removed", false)
				if result.Error != nil {
					return out, research.ErrUnavailable
				}
			}
			if result.RowsAffected == 1 {
				updates := map[string]any{"revision": row.Revision + 1}
				if row.SelectedLinkID == nil {
					updates["selected_link_id"] = linkID
					updates["state"] = "TO_RESEARCH"
					updates["reason"] = ""
				}
				if e := scoped(tx, s, p).Model(&candidateRow{}).Where("id=?", id).Updates(updates).Error; e != nil {
					return out, research.ErrUnavailable
				}
			}
			out.CandidateIDs = append(out.CandidateIDs, id)
			out.Revisions[id] = row.Revision
			if result.RowsAffected == 1 {
				out.Revisions[id]++
			}
		}
	} else {
		row, e := load(tx, s, p, c.CandidateID, true)
		if e != nil {
			return out, e
		}
		if row.Revision != c.ExpectedRevision {
			return out, research.ErrConflict
		}
		var ref research.EvidenceRef
		linkID := ""
		if row.SelectedLinkID != nil {
			linkID = *row.SelectedLinkID
		}
		if c.Action == "select" || c.Action == "unlink" {
			linkID = c.LinkID
		}
		var link linkRow
		if e := scoped(tx, s, p).Where("id=? AND candidate_id=? AND removed=false", linkID, row.ID).Take(&link).Error; e != nil {
			return out, research.ErrConflict
		}
		if json.Unmarshal(link.ReferenceJSON, &ref) != nil {
			return out, research.ErrUnavailable
		}
		if c.Action != "select" && c.Action != "unlink" && c.EvidenceDigest != ref.Digest() {
			return out, research.ErrConflict
		}
		switch c.Action {
		case "select":
			row.SelectedLinkID = &linkID
			row.State = "TO_RESEARCH"
			row.Reason = ""
		case "unlink":
			if e := scoped(tx, s, p).Model(&linkRow{}).Where("id=?", linkID).Update("removed", true).Error; e != nil {
				return out, research.ErrUnavailable
			}
			if row.SelectedLinkID != nil && *row.SelectedLinkID == linkID {
				row.SelectedLinkID = nil
				row.State = "TO_RESEARCH"
				row.Reason = ""
			}
		case "review":
			if row.State != "TO_RESEARCH" {
				return out, research.ErrConflict
			}
			row.State = "PENDING_CONFIRMATION"
		case "decide":
			if row.State != "PENDING_CONFIRMATION" {
				return out, research.ErrConflict
			}
			row.State = c.State
			row.Reason = c.Reason
		case "reset":
			row.State = "TO_RESEARCH"
			row.Reason = ""
		default:
			return out, research.ErrInvalid
		}
		row.Revision++
		if e := scoped(tx, s, p).Model(&candidateRow{}).Where("id=?", row.ID).Updates(map[string]any{"selected_link_id": row.SelectedLinkID, "state": row.State, "reason": row.Reason, "revision": row.Revision}).Error; e != nil {
			return out, research.ErrUnavailable
		}
		decision = &research.Decision{State: row.State, Reason: row.Reason, EvidenceDigest: ref.Digest(), Ref: ref, Revision: row.Revision, At: now}
		out.CandidateIDs = append(out.CandidateIDs, row.ID)
		out.Revisions[row.ID] = row.Revision
	}
	raw, _ := json.Marshal(c)
	receiptJSON, _ := json.Marshal(out)
	command := commandRow{OrganizationID: s.OrganizationID, ActorID: s.ActorID, ProjectID: p, Key: key, InputHash: collection.Digest(c), IntentJSON: raw, ReceiptJSON: receiptJSON, CreatedAt: now}
	if c.CandidateID != "" {
		command.CandidateID = &c.CandidateID
	}
	if decision != nil {
		command.DecisionJSON, _ = json.Marshal(decision)
	}
	if e := tx.Create(&command).Error; e != nil {
		return research.Receipt{}, research.ErrUnavailable
	}
	if tx.Commit().Error != nil {
		return research.Receipt{}, research.ErrUnknown
	}
	return out, nil
}
