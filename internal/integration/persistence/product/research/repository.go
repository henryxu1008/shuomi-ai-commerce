package researchpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"sort"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/research"
	"time"
)

type Repository struct {
	db          *gorm.DB
	assessments bool
}

func New(ctx context.Context, db *gorm.DB) (*Repository, error) {
	if e := VerifySchema(ctx, db); e != nil {
		return nil, e
	}
	var present bool
	if db.WithContext(ctx).Raw("SELECT to_regclass(?) IS NOT NULL", assessmentTable).Scan(&present).Error != nil {
		return nil, research.ErrUnavailable
	}
	if present && VerifyAssessmentSchema(ctx, db) != nil {
		return nil, research.ErrUnavailable
	}
	return &Repository{db: db, assessments: present}, nil
}

type candidateRow struct {
	OrganizationID, ActorID, ProjectID, ID, Platform, Market, NativeProductID string
	SelectedLinkID                                                            *string
	State, Reason                                                             string
	Revision                                                                  int64
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
	return scopedReceipt(r.db.WithContext(ctx), s, p, row)
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
	out, e := scopedReceipt(r.db.WithContext(ctx), s, p, row)
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

// ReadAnalysisEvidence uses one scoped read for candidate state and the exact
// attached evidence, independent of the display window.
func (r *Repository) ReadAnalysisEvidence(ctx context.Context, scope collection.Scope, project, candidate, link string) (research.AnalysisEvidence, error) {
	var row struct {
		Revision       int64
		SelectedLinkID *string
		ReferenceJSON  []byte
	}
	err := r.db.WithContext(ctx).Table("research_opportunities AS candidate").
		Select("candidate.revision, candidate.selected_link_id, evidence.reference_json").
		Joins(`LEFT JOIN research_evidence_links AS evidence ON
			evidence.organization_id=candidate.organization_id AND evidence.actor_id=candidate.actor_id
			AND evidence.project_id=candidate.project_id AND evidence.candidate_id=candidate.id
			AND evidence.id=? AND evidence.removed=false`, link).
		Where("candidate.organization_id=? AND candidate.actor_id=? AND candidate.project_id=? AND candidate.id=?",
			scope.OrganizationID, scope.ActorID, project, candidate).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return research.AnalysisEvidence{}, research.ErrNotFound
	}
	if err != nil {
		return research.AnalysisEvidence{}, research.ErrUnavailable
	}
	out := research.AnalysisEvidence{CandidateRevision: row.Revision}
	if row.SelectedLinkID != nil {
		out.SelectedLinkID = *row.SelectedLinkID
	}
	if row.ReferenceJSON != nil {
		var ref research.EvidenceRef
		if json.Unmarshal(row.ReferenceJSON, &ref) != nil {
			return research.AnalysisEvidence{}, research.ErrUnavailable
		}
		out.Ref = &ref
	}
	return out, nil
}

func details(db *gorm.DB, s collection.Scope, p string, row candidateRow) (research.Candidate, error) {
	out := research.Candidate{ID: row.ID, ProjectID: p, Identity: research.ProductIdentity{Platform: row.Platform, Market: row.Market, NativeProductID: row.NativeProductID}, State: row.State, Reason: row.Reason, Revision: row.Revision, Links: []research.Link{}, History: []research.Decision{}}
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
		d, err := commandDecision(c, row.ID)
		if err != nil {
			return out, err
		}
		out.History = append(out.History, d)
	}
	if row.Reason != "" {
		// The current reason is owned by a decide command, not by the bounded
		// display history. Nonselected unlinks must not sever its authorization.
		var owners []commandRow
		if err := scoped(db, s, p).Where("candidate_id=? AND decision_json IS NOT NULL AND intent_json->>'action'='decide'", row.ID).
			Order("(decision_json->>'Revision')::bigint DESC").Limit(1).Find(&owners).Error; err != nil {
			return out, research.ErrUnavailable
		}
		if len(owners) == 1 {
			d, err := commandDecision(owners[0], row.ID)
			if err != nil {
				return out, err
			}
			if d.Reason == row.Reason && d.State == row.State && d.Revision <= row.Revision {
				out.ReasonDecision = &d
			}
		}
	}
	return out, nil
}

func commandDecision(c commandRow, candidate string) (research.Decision, error) {
	var d research.Decision
	if json.Unmarshal(c.DecisionJSON, &d) != nil {
		return d, research.ErrUnavailable
	}
	if d.Reason != "" {
		var intent research.Command
		if json.Unmarshal(c.IntentJSON, &intent) != nil || intent.Action != "decide" || intent.CandidateID != candidate || intent.Reason != d.Reason || intent.EvidenceDigest != d.Ref.Digest() {
			d.Reason = ""
			d.AssessmentRef = nil
		}
	}
	return d, nil
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
	return r.commit(ctx, s, p, key, c, observations, validation, nil, nil)
}

func (r *Repository) commit(ctx context.Context, s collection.Scope, p, key string, c research.Command, observations []research.Observation, validation error, supplier *research.SupplierReviewProof, assessment *research.AssessmentProof) (research.Receipt, error) {
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
		return scopedReceipt(tx, s, p, existing)
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
			return observations[i].Identity().CandidateID(s, p) < observations[j].Identity().CandidateID(s, p)
		})
		for _, v := range observations {
			identity := v.Identity()
			id := identity.CandidateID(s, p)
			row := candidateRow{OrganizationID: s.OrganizationID, ActorID: s.ActorID, ProjectID: p, ID: id, Platform: identity.Platform, Market: identity.Market, NativeProductID: identity.NativeProductID, State: "TO_RESEARCH", Revision: 1}
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
		if assessment != nil {
			if row.Platform != "amazon" || row.Market != "us" || assessment.Record.MemberID != s.MemberID || assessment.Record.Market != ref || assessment.Record.LinkID != linkID || assessment.Recheck == nil || !time.Now().Before(assessment.Expires) {
				return out, research.ErrConflict
			}
			if c.Action == research.SaveAssessment {
				if assessment.Record.CandidateRevision != row.Revision || assessment.Record.RequestHash != collection.Digest(c) || linkID != c.LinkID {
					return out, research.ErrConflict
				}
				v := assessment.Record.Reference(key)
				out.AssessmentRef = &v
			} else {
				stored, e := loadAssessment(tx, s, p, c.CandidateID, c.AssessmentRef.ID)
				if e != nil {
					return out, e
				}
				if stored.Record.MemberID != s.MemberID || stored.Record.Reference(stored.ID) != *c.AssessmentRef || collection.Digest(stored.Record) != collection.Digest(assessment.Record) {
					return out, research.ErrConflict
				}
				out.AssessmentRef = c.AssessmentRef
			}
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
		case research.RecordSupplierReview:
			if supplier == nil || row.Platform != "amazon" || row.Market != "us" ||
				linkID != c.LinkID || ref.Source.Kind != "amazon_data" || !time.Now().Before(supplier.Expires) {
				return out, research.ErrConflict
			}
		case research.SaveAssessment:
			if assessment == nil || c.Assessment == nil {
				return out, research.ErrInvalid
			}
		case "reset":
			row.State = "TO_RESEARCH"
			row.Reason = ""
		default:
			return out, research.ErrInvalid
		}
		row.Revision++
		updates := map[string]any{"revision": row.Revision}
		if supplier == nil && c.Action != research.SaveAssessment {
			updates["selected_link_id"] = row.SelectedLinkID
			updates["state"] = row.State
			updates["reason"] = row.Reason
		}
		if e := scoped(tx, s, p).Model(&candidateRow{}).Where("id=?", row.ID).Updates(updates).Error; e != nil {
			return out, research.ErrUnavailable
		}
		if supplier == nil && c.Action != research.SaveAssessment {
			decision = &research.Decision{State: row.State, EvidenceDigest: ref.Digest(), Ref: ref, Revision: row.Revision, At: now}
		}
		if c.Action == "decide" {
			decision.Reason = c.Reason
			decision.AssessmentRef = c.AssessmentRef
		}
		out.CandidateIDs = append(out.CandidateIDs, row.ID)
		out.Revisions[row.ID] = row.Revision
	}
	raw, _ := json.Marshal(c)
	if supplier != nil || c.Action == research.SaveAssessment {
		raw, _ = json.Marshal(research.Command{Action: c.Action, CandidateID: c.CandidateID, ExpectedRevision: c.ExpectedRevision})
	}
	receiptJSON, _ := json.Marshal(out)
	command := commandRow{OrganizationID: s.OrganizationID, ActorID: s.ActorID, ProjectID: p, Key: key, InputHash: collection.Digest(c), IntentJSON: raw, ReceiptJSON: receiptJSON, CreatedAt: now}
	if c.CandidateID != "" {
		command.CandidateID = &c.CandidateID
	}
	if decision != nil {
		command.DecisionJSON, _ = json.Marshal(decision)
	}
	if e := tx.Session(&gorm.Session{Logger: logger.Discard}).Create(&command).Error; e != nil {
		return research.Receipt{}, research.ErrUnavailable
	}
	if supplier != nil {
		if err := insertSupplierReview(tx, s, p, key, c, out.Revisions[c.CandidateID], now, *supplier); err != nil {
			return research.Receipt{}, err
		}
	}
	if assessment != nil {
		if c.Action == research.SaveAssessment {
			if err := insertAssessment(tx, s, p, key, c, out.Revisions[c.CandidateID], now, *assessment); err != nil {
				return research.Receipt{}, err
			}
		}
		if !time.Now().Before(assessment.Expires) {
			return research.Receipt{}, research.ErrConflict
		}
		if err := assessment.Recheck(ctx); err != nil {
			return research.Receipt{}, err
		}
		if !time.Now().Before(assessment.Expires) {
			return research.Receipt{}, research.ErrConflict
		}
	}
	if tx.Commit().Error != nil {
		return research.Receipt{}, research.ErrUnknown
	}
	return out, nil
}
