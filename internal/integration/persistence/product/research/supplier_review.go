package researchpersistence

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"task-processor/internal/product/collection"
	"task-processor/internal/product/research"
)

type supplierReviewRow struct {
	OrganizationID, ActorID, ProjectID, RequestKey, CandidateID string
	CandidateRevision                                           int64
	RecordJSON                                                  []byte
	CreatedAt                                                   time.Time
}

func (supplierReviewRow) TableName() string { return "research_supplier_reviews" }

func (r *Repository) CommitSupplier(ctx context.Context, scope collection.Scope, project, key string, cmd research.Command, proof research.SupplierReviewProof, validation error) (research.Receipt, error) {
	if cmd.Action != research.RecordSupplierReview || cmd.Supplier == nil {
		return research.Receipt{}, research.ErrInvalid
	}
	if validation == nil {
		if proof.Expires.IsZero() {
			return research.Receipt{}, research.ErrUnavailable
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, proof.Expires)
		defer cancel()
	}
	return r.commit(ctx, scope, project, key, cmd, nil, validation, &proof, nil)
}

func insertSupplierReview(tx *gorm.DB, scope collection.Scope, project, key string, cmd research.Command, revision int64, now time.Time, proof research.SupplierReviewProof) error {
	rec := proof.Record
	if !time.Now().Before(proof.Expires) || rec.MemberID != scope.MemberID || rec.LinkID != cmd.LinkID ||
		rec.EvidenceDigest != cmd.EvidenceDigest || rec.Market.Digest() != cmd.EvidenceDigest ||
		collection.Digest(rec.Input) != collection.Digest(cmd.Supplier) {
		return research.ErrConflict
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return research.ErrUnavailable
	}
	row := supplierReviewRow{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, ProjectID: project,
		RequestKey: key, CandidateID: cmd.CandidateID, CandidateRevision: revision, RecordJSON: raw, CreatedAt: now}
	if tx.Session(&gorm.Session{Logger: logger.Discard}).Create(&row).Error != nil {
		return research.ErrUnavailable
	}
	return nil
}

func (r *Repository) ListSupplierReviews(ctx context.Context, scope collection.Scope, project, candidate string, before int64) ([]research.SupplierReviewRow, error) {
	q := scoped(r.db.WithContext(ctx), scope, project).Where("candidate_id=?", candidate)
	if before > 0 {
		q = q.Where("candidate_revision < ?", before)
	}
	var rows []supplierReviewRow
	if err := q.Order("candidate_revision DESC").Limit(6).Find(&rows).Error; err != nil {
		return nil, research.ErrUnavailable
	}
	out := make([]research.SupplierReviewRow, 0, len(rows))
	for _, row := range rows {
		var record research.SupplierReviewRecord
		if json.Unmarshal(row.RecordJSON, &record) != nil {
			return nil, research.ErrUnavailable
		}
		out = append(out, research.SupplierReviewRow{RequestKey: row.RequestKey, Revision: row.CandidateRevision, At: row.CreatedAt, Record: record})
	}
	return out, nil
}

func (r *Repository) SupplierReviewLink(ctx context.Context, scope collection.Scope, project, candidate, id string) (research.EvidenceRef, error) {
	var row linkRow
	if err := scoped(r.db.WithContext(ctx), scope, project).Where("id=? AND candidate_id=? AND removed=false", id, candidate).Take(&row).Error; err != nil {
		return research.EvidenceRef{}, research.ErrNotFound
	}
	var ref research.EvidenceRef
	if json.Unmarshal(row.ReferenceJSON, &ref) != nil {
		return research.EvidenceRef{}, research.ErrUnavailable
	}
	return ref, nil
}

const supplierTable = "public.research_supplier_reviews"

// InstallSupplierReviewSchema is called by Product's locked empty-database
// initialization after base Research schema and grants have been installed.
func InstallSupplierReviewSchema(tx *gorm.DB) error {
	if tx == nil || tx.Dialector.Name() != "postgres" {
		return research.ErrUnavailable
	}
	return tx.Exec(`CREATE TABLE public.research_supplier_reviews (
 organization_id varchar(128) NOT NULL,
 actor_id varchar(128) NOT NULL,
 project_id uuid NOT NULL,
 request_key uuid NOT NULL,
 candidate_id uuid NOT NULL,
 candidate_revision bigint NOT NULL CHECK(candidate_revision > 0),
 record_json jsonb NOT NULL CHECK(octet_length(record_json::text) <= 16384),
 created_at timestamptz NOT NULL,
 CONSTRAINT supplier_review_key PRIMARY KEY(organization_id,actor_id,project_id,request_key),
 CONSTRAINT supplier_review_revision UNIQUE(organization_id,actor_id,project_id,candidate_id,candidate_revision),
 CONSTRAINT supplier_review_candidate FOREIGN KEY(organization_id,actor_id,project_id,candidate_id)
   REFERENCES public.research_opportunities(organization_id,actor_id,project_id,id),
 CONSTRAINT supplier_review_command FOREIGN KEY(organization_id,actor_id,project_id,request_key)
   REFERENCES public.research_commands(organization_id,actor_id,project_id,key)
 );
 REVOKE ALL ON TABLE public.research_supplier_reviews FROM PUBLIC, product_research_runtime;
 GRANT SELECT, INSERT ON TABLE public.research_supplier_reviews TO product_research_runtime;`).Error
}

func VerifySupplierReviewSchema(ctx context.Context, db *gorm.DB) error {
	return verifySupplierReviewSchema(ctx, db, true)
}

// VerifySupplierReviewOwnerSchema is used only by explicit owner installation.
// The serving check continues to reject a table-owner connection.
func VerifySupplierReviewOwnerSchema(ctx context.Context, db *gorm.DB) error {
	return verifySupplierReviewSchema(ctx, db, false)
}

func verifySupplierReviewSchema(ctx context.Context, db *gorm.DB, serving bool) error {
	var verified bool
	err := db.WithContext(ctx).Raw(`WITH expected(name, kind, cols, ref, refcols) AS (VALUES
 ('supplier_review_key','p',ARRAY['organization_id','actor_id','project_id','request_key'],NULL::text,NULL::text[]),
 ('supplier_review_revision','u',ARRAY['organization_id','actor_id','project_id','candidate_id','candidate_revision'],NULL,NULL),
 ('supplier_review_candidate','f',ARRAY['organization_id','actor_id','project_id','candidate_id'],'public.research_opportunities',ARRAY['organization_id','actor_id','project_id','id']),
 ('supplier_review_command','f',ARRAY['organization_id','actor_id','project_id','request_key'],'public.research_commands',ARRAY['organization_id','actor_id','project_id','key'])
 ) SELECT NOT EXISTS(SELECT 1 FROM expected e WHERE NOT EXISTS(
 SELECT 1 FROM pg_constraint c WHERE c.conrelid=to_regclass(?) AND c.conname=e.name AND c.contype::text=e.kind
 AND c.convalidated AND NOT c.condeferrable
 AND ARRAY(SELECT a.attname::text FROM unnest(c.conkey) WITH ORDINALITY AS k(n,i)
 JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=k.n ORDER BY k.i)=e.cols
 AND (e.ref IS NULL OR (c.confrelid=to_regclass(e.ref) AND c.confdeltype='a' AND c.confupdtype='a'
 AND ARRAY(SELECT a.attname::text FROM unnest(c.confkey) WITH ORDINALITY AS k(n,i)
 JOIN pg_attribute a ON a.attrelid=c.confrelid AND a.attnum=k.n ORDER BY k.i)=e.refcols))))
 AND EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass(?) AND convalidated AND pg_get_constraintdef(oid)='CHECK ((candidate_revision > 0))')
 AND EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass(?) AND convalidated AND pg_get_constraintdef(oid)='CHECK ((octet_length((record_json)::text) <= 16384))')
 AND EXISTS(SELECT 1 FROM pg_class WHERE oid=to_regclass(?) AND relkind='r' AND (? OR relowner<>(SELECT oid FROM pg_roles WHERE rolname=current_user)))`,
		supplierTable, supplierTable, supplierTable, supplierTable, !serving).Scan(&verified).Error
	if err != nil || !verified {
		return research.ErrUnavailable
	}
	if db.WithContext(ctx).Exec("SELECT organization_id,actor_id,project_id,request_key,candidate_id,candidate_revision,record_json,created_at FROM "+supplierTable+" LIMIT 0").Error != nil {
		return research.ErrUnavailable
	}
	return nil
}
