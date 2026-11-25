package skillpersistence

import (
	"context"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agentconfig"
	d "github.com/henryxu1008/shuomi-ai-commerce/internal/skill"
	"gorm.io/gorm"
	"sort"
	"sync"
	"time"
)

func (s *Store) UsesPool(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	a, e := s.db.DB()
	b, be := db.DB()
	return e == nil && be == nil && a == b
}

// CheckUseInTransaction joins the existing configuration/Run Claim transaction.
// All callers resolve live identity/grants before supplying this trusted principal.
func (s *Store) CheckUseInTransaction(ctx context.Context, tx *gorm.DB, p d.Principal, ref agent.SkillVersionRef) (d.Package, error) {
	return s.checkUseForTarget(ctx, tx, p, ref, d.TitleTarget, false)
}

// CheckImageUseInTransaction joins the configuration owner's snapshot write
// after the application has resolved the trusted principal.
func (s *Store) CheckImageUseInTransaction(ctx context.Context, tx *gorm.DB, p d.Principal, ref agent.SkillVersionRef) error {
	material, err := s.checkUseForTarget(ctx, tx, p, ref, d.ImageTarget, false)
	if err != nil {
		return err
	}
	_, err = d.ImageVisualBrief(material)
	return err
}

func (s *Store) checkUseForTarget(ctx context.Context, tx *gorm.DB, p d.Principal, ref agent.SkillVersionRef, target string, completedReview bool) (d.Package, error) {
	agentID := agentconfig.ContentAgentID
	if target == d.ImageTarget {
		agentID = agentconfig.ImageAgentID
	} else if target != d.TitleTarget {
		return d.Package{}, d.ErrInvalid
	}
	if !ref.Valid() || !agent.ValidID(p.UserID) || !agent.ValidID(p.OrganizationID) || !agent.ValidID(p.MemberID) || !p.UseOrganization {
		return d.Package{}, d.ErrForbidden
	}
	var activation string
	q := tx.WithContext(ctx).Raw("SELECT activation FROM agent_configuration.organization_agents WHERE organization_id=? AND agent_id=? FOR SHARE", p.OrganizationID, agentID).Scan(&activation)
	if q.Error != nil {
		return d.Package{}, d.ErrUnavailable
	}
	if q.RowsAffected != 1 || (activation != "ENABLED" && !(completedReview && activation == "DISABLED")) {
		return d.Package{}, d.ErrNotEnabled
	}
	if ref.OwnerKind == d.UserOwner {
		if target == d.ImageTarget {
			return d.Package{}, d.ErrForbidden
		}
		var allow bool
		if e := tx.WithContext(ctx).Raw("SELECT allow_personal FROM agent_configuration.skill_admissions WHERE organization_id=? AND agent_id='product.content.agent'", p.OrganizationID).Scan(&allow).Error; e != nil {
			return d.Package{}, d.ErrUnavailable
		}
		if !allow {
			return d.Package{}, d.ErrForbidden
		}
	}
	var locked string
	if e := tx.WithContext(ctx).Raw("SELECT id FROM skill_capability.skills WHERE id=? FOR SHARE", ref.ID).Scan(&locked).Error; e != nil {
		return d.Package{}, d.ErrUnavailable
	}
	v, e := readVersion(tx.WithContext(ctx), ref.ID, ref.Version, false)
	if e != nil {
		return d.Package{}, e
	}
	if v.Ref() != ref {
		return d.Package{}, d.ErrConflict
	}
	if v.State != d.Enabled {
		return d.Package{}, d.ErrNotEnabled
	}
	if !d.Authorized(p, v, d.Use) {
		return d.Package{}, d.ErrForbidden
	}
	if !v.Package.Supports(target) {
		return d.Package{}, d.ErrForbidden
	}
	return v.Package, nil
}

// AcquireUse holds the availability fence until the caller's existing effect
// owner returns. Rollback is idempotent; cancellation also releases the PG tx.
func (s *Store) AcquireUse(ctx context.Context, p d.Principal, ref agent.SkillVersionRef) (d.Package, func(), error) {
	return s.AcquireUseForTarget(ctx, p, ref, d.TitleTarget)
}

// AcquireChatText grants use of an exact private version by its owner. The
// Workbench consumer independently checks live Chat and conversation access.
func (s *Store) AcquireChatText(ctx context.Context, actor, id string, version uint64, digest string) (d.Version, func(), error) {
	if !agent.ValidID(actor) || !idOK(id) || version == 0 || version >= 1<<63 || len(digest) != 64 {
		return d.Version{}, nil, d.ErrInvalid
	}
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return d.Version{}, nil, d.ErrUnavailable
	}
	var locked string
	q := tx.Raw("SELECT id FROM skill_capability.skills WHERE id=? AND owner_kind='USER' AND owner_id=? FOR SHARE", id, actor).Scan(&locked)
	if q.Error != nil || q.RowsAffected != 1 {
		_ = tx.Rollback().Error
		if q.Error != nil {
			return d.Version{}, nil, d.ErrUnavailable
		}
		return d.Version{}, nil, d.ErrNotFound
	}
	v, err := readVersion(tx, id, version, false)
	if err != nil || v.Package.Digest != digest || v.Visibility != d.Private || !v.Package.Supports(d.ChatTextTarget) || !d.Authorized(d.Principal{UserID: actor}, v, d.Use) {
		_ = tx.Rollback().Error
		return d.Version{}, nil, d.ErrNotEnabled
	}
	var once sync.Once
	release := func() { once.Do(func() { _ = tx.Rollback().Error }) }
	return v, release, nil
}

// AcquireAuthoring fences the exact official revision while the separately
// authorized Chat consumer sends. Title and image grants do not confer it.
func (s *Store) AcquireAuthoring(ctx context.Context, ref agent.SkillVersionRef) (func(), error) {
	if !ref.Valid() || ref.ID != d.OfficialReleaseID(d.AuthoringReleaseKey) || ref.OwnerKind != d.PlatformOwner || ref.OwnerID != d.PlatformID {
		return nil, d.ErrForbidden
	}
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, d.ErrUnavailable
	}
	var locked string
	result := tx.Raw("SELECT id FROM skill_capability.skills WHERE id=? FOR SHARE", ref.ID).Scan(&locked)
	if result.Error != nil || result.RowsAffected != 1 {
		_ = tx.Rollback().Error
		return nil, d.ErrUnavailable
	}
	v, err := readVersion(tx, ref.ID, ref.Version, false)
	if err != nil || v.Ref() != ref || v.State != d.Enabled || v.Visibility != d.Shared || !v.Package.Supports(d.AuthoringTarget) {
		_ = tx.Rollback().Error
		return nil, d.ErrNotEnabled
	}
	var once sync.Once
	return func() { once.Do(func() { _ = tx.Rollback().Error }) }, nil
}

func (s *Store) AcquireUseForTarget(ctx context.Context, p d.Principal, ref agent.SkillVersionRef, target string) (d.Package, func(), error) {
	if target == d.ImageTarget {
		packages, release, err := s.acquireImageUses(ctx, p, []agent.SkillVersionRef{ref})
		if err != nil {
			return d.Package{}, nil, err
		}
		return packages[0], release, nil
	}
	return s.acquireUseForTarget(ctx, p, ref, target, false)
}

func (s *Store) AcquireImageUses(ctx context.Context, p d.Principal, refs []agent.SkillVersionRef) (func(), error) {
	if len(refs) == 0 || len(refs) > 40 {
		return nil, d.ErrInvalid
	}
	ordered := append([]agent.SkillVersionRef(nil), refs...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].ID == ordered[j].ID {
			return ordered[i].Version < ordered[j].Version
		}
		return ordered[i].ID < ordered[j].ID
	})
	packages, release, err := s.acquireImageUses(ctx, p, ordered)
	if err != nil {
		return nil, err
	}
	for _, material := range packages {
		if _, err := d.ImageVisualBrief(material); err != nil {
			release()
			return nil, err
		}
	}
	return release, nil
}

func (s *Store) acquireImageUses(ctx context.Context, p d.Principal, refs []agent.SkillVersionRef) ([]d.Package, func(), error) {
	pool, err := s.db.DB()
	if err != nil {
		return nil, nil, d.ErrUnavailable
	}
	wait, cancelWait := context.WithTimeout(ctx, time.Second)
	conn, err := pool.Conn(wait)
	cancelWait()
	if err != nil {
		return nil, nil, d.ErrUnavailable
	}
	bound := s.db.WithContext(ctx)
	bound.Statement.ConnPool = conn
	tx := bound.Begin()
	if tx.Error != nil {
		_ = conn.Close()
		return nil, nil, d.ErrUnavailable
	}
	closeConnection := sync.OnceFunc(func() {
		_ = tx.Rollback().Error
		_ = conn.Close()
	})
	stopClose := context.AfterFunc(ctx, closeConnection)
	release := func() {
		stopClose()
		closeConnection()
	}
	// Only acquisition is short. Begin uses the operation context so a slow
	// authorized supplier call cannot expire an otherwise valid fence.
	lock, cancelLock := context.WithTimeout(ctx, time.Second)
	defer cancelLock()
	if err := tx.WithContext(lock).Exec("SET LOCAL lock_timeout = '1s'").Error; err != nil {
		release()
		return nil, nil, d.ErrUnavailable
	}
	packages := make([]d.Package, 0, len(refs))
	for _, ref := range refs {
		material, err := s.checkUseForTarget(lock, tx, p, ref, d.ImageTarget, false)
		if err != nil {
			release()
			return nil, nil, err
		}
		packages = append(packages, material)
	}
	return packages, release, nil
}

// AcquireCompletedTitleScriptReview holds the live Skill grant for an already
// completed, verified own-product title script. A pause only prevents execution.
func (s *Store) AcquireCompletedTitleScriptReview(ctx context.Context, p d.Principal, ref agent.SkillVersionRef) (d.Package, func(), error) {
	if ref.PolicyVersion != d.ScriptPolicyVersion {
		return d.Package{}, nil, d.ErrInvalid
	}
	return s.acquireUseForTarget(ctx, p, ref, d.TitleTarget, true)
}

func (s *Store) acquireUseForTarget(ctx context.Context, p d.Principal, ref agent.SkillVersionRef, target string, completedReview bool) (d.Package, func(), error) {
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return d.Package{}, nil, d.ErrUnavailable
	}
	material, e := s.checkUseForTarget(ctx, tx, p, ref, target, completedReview)
	if e != nil {
		_ = tx.Rollback().Error
		return d.Package{}, nil, e
	}
	var once sync.Once
	release := func() { once.Do(func() { _ = tx.Rollback().Error }) }
	return material, release, nil
}
func (s *Store) ReadForUse(ctx context.Context, p d.Principal, ref agent.SkillVersionRef) (d.Package, error) {
	material, release, e := s.AcquireUse(ctx, p, ref)
	if e != nil {
		return d.Package{}, e
	}
	release()
	return material, nil
}
