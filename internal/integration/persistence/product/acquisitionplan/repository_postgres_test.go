package acquisitionplan

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
	"strings"
	"task-processor/internal/product/sourcing"
	"testing"
	"time"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("ISSUE398_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP: task-owned PostgreSQL is required")
	}
	require.Contains(t, dsn, "host=127.0.0.1")
	require.Contains(t, dsn, "user=issue398_owner")
	root, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	name := "acqplan_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, root.Exec("CREATE DATABASE "+name).Error)
	db, e := gorm.Open(postgres.Open(dsn+" dbname="+name), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	t.Cleanup(func() {
		pool, _ := db.DB()
		_ = pool.Close()
		require.NoError(t, root.Exec("DROP DATABASE "+name+" WITH (FORCE)").Error)
		p, _ := root.DB()
		_ = p.Close()
	})
	return db
}

func TestSchemaRejectsUnscopedForeignKey(t *testing.T) {
	db := testDB(t)
	require.NoError(t, InstallSchema(db))
	require.NoError(t, db.Exec(`ALTER TABLE public.product_acquisition_plans ADD CONSTRAINT wrong_unique UNIQUE(id)`).Error)
	var fk string
	require.NoError(t, db.Raw(`SELECT conname FROM pg_constraint WHERE conrelid='public.product_acquisition_plan_items'::regclass AND contype='f'`).Scan(&fk).Error)
	require.NoError(t, db.Exec(`ALTER TABLE public.product_acquisition_plan_items DROP CONSTRAINT "`+fk+`"`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE public.product_acquisition_plan_items ADD CONSTRAINT wrong_fk FOREIGN KEY(plan_id) REFERENCES public.product_acquisition_plans(id)`).Error)
	// Keep the plan constraint count unchanged, so counting alone is insufficient.
	var unique string
	require.NoError(t, db.Raw(`SELECT conname FROM pg_constraint WHERE conrelid='public.product_acquisition_plans'::regclass AND contype='u' AND conname<>'wrong_unique'`).Scan(&unique).Error)
	require.NoError(t, db.Exec(`ALTER TABLE public.product_acquisition_plans DROP CONSTRAINT "`+unique+`"`).Error)
	_, e := NewRepository(context.Background(), db)
	require.Error(t, e)
}

func TestPlanScopeFenceAndFrozenIntent(t *testing.T) {
	db := testDB(t)
	require.NoError(t, InstallSchema(db))
	repo, e := NewRepository(context.Background(), db)
	require.NoError(t, e)
	ctx := context.Background()
	check := func(context.Context) error { return nil }
	scope := sourcing.AcquisitionPlanScope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	input, e := sourcing.NewAcquisitionPlanInput("product", []string{"123"}, sourcing.AcquisitionRule{MaxPrice: "10"}, true)
	require.NoError(t, e)
	p := sourcing.AcquisitionPlan{ID: uuid.NewString(), Key: uuid.NewString(), Scope: scope, Input: input, Revision: 1, State: "pending", DiscoveryIndex: 1, CreatedAt: time.Now().UTC(), Items: []sourcing.AcquisitionPlanItem{{Source: input.Inputs[0], ChildKey: uuid.NewString(), State: "pending"}}}
	p, e = repo.Create(ctx, p, check)
	require.NoError(t, e)
	_, e = repo.Read(ctx, sourcing.AcquisitionPlanScope{OrganizationID: "org", ActorID: "actor", MemberID: "different"}, p.ID)
	require.ErrorIs(t, e, sourcing.ErrAcquisitionNotFound)
	first, e := repo.Claim(ctx, scope, p.ID, 1, check)
	require.NoError(t, e)
	_, e = repo.Claim(ctx, scope, p.ID, 1, check)
	require.ErrorIs(t, e, sourcing.ErrAcquisitionConflict)
	first.TransferKey = uuid.NewString()
	first.TransferInputJSON = []byte(`[{"key":"fixture-key","input":{"batchId":"fixture"}}]`)
	first.PreparationIDs = []string{uuid.NewString()}
	first.Items[0].CollectionBatchID = uuid.NewString()
	first.Items[0].CollectionRevision = 4
	saved, e := repo.Save(ctx, first, check)
	require.NoError(t, e)
	require.Equal(t, int64(2), saved.Revision)
	replay, e := repo.ByKey(ctx, scope, p.Key)
	require.NoError(t, e)
	require.Equal(t, first.TransferKey, replay.TransferKey)
	require.Equal(t, first.TransferInputJSON, replay.TransferInputJSON)
	require.Equal(t, first.Items[0].ChildKey, replay.Items[0].ChildKey)
	require.Equal(t, int64(4), replay.Items[0].CollectionRevision)
	_, e = repo.Save(ctx, first, check)
	require.ErrorIs(t, e, sourcing.ErrAcquisitionConflict)
	saved, e = repo.Claim(ctx, scope, p.ID, 2, check)
	require.NoError(t, e)
	changed := saved
	changed.TransferInputJSON = []byte(`[]`)
	_, e = repo.Save(ctx, changed, check)
	require.ErrorIs(t, e, sourcing.ErrAcquisitionConflict)
	changed = saved
	changed.PreparationIDs = []string{uuid.NewString()}
	_, e = repo.Save(ctx, changed, check)
	require.ErrorIs(t, e, sourcing.ErrAcquisitionConflict)
	_, e = repo.Save(ctx, saved, func(context.Context) error { return sourcing.ErrPublicationForbidden })
	require.ErrorIs(t, e, sourcing.ErrPublicationForbidden)
	replay, e = repo.Read(ctx, scope, p.ID)
	require.NoError(t, e)
	require.Equal(t, int64(2), replay.Revision)
}

func TestRuleMutationReplayAndRevision(t *testing.T) {
	db := testDB(t)
	require.NoError(t, InstallSchema(db))
	repo, e := NewRepository(context.Background(), db)
	require.NoError(t, e)
	ctx := context.Background()
	scope := sourcing.AcquisitionPlanScope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	check := func(context.Context) error { return nil }
	key := uuid.NewString()
	input := sourcing.RuleTemplateMutation{Action: "create", Name: "采购规则", Rule: sourcing.AcquisitionRule{MaxPrice: "10"}}
	rule, e := repo.MutateRule(ctx, scope, key, input, check)
	require.NoError(t, e)
	update := sourcing.RuleTemplateMutation{Action: "update", ID: rule.ID, ExpectedRevision: 1, Name: "调整后", Rule: sourcing.AcquisitionRule{MaxPrice: "20"}}
	changed, e := repo.MutateRule(ctx, scope, uuid.NewString(), update, check)
	require.NoError(t, e)
	require.Equal(t, int64(2), changed.Revision)
	replay, e := repo.MutateRule(ctx, scope, key, input, check)
	require.NoError(t, e)
	require.Equal(t, int64(1), replay.Revision)
	require.Equal(t, "10", replay.Rule.MaxPrice)
	input.Name = "不同输入"
	_, e = repo.MutateRule(ctx, scope, key, input, check)
	require.ErrorIs(t, e, sourcing.ErrAcquisitionConflict)
	_, e = repo.MutateRule(ctx, scope, uuid.NewString(), update, check)
	require.ErrorIs(t, e, sourcing.ErrAcquisitionConflict)
}

func TestRuleTemplateQuotaFollowsCurrentMembership(t *testing.T) {
	db := testDB(t)
	require.NoError(t, InstallSchema(db))
	repo, err := NewRepository(context.Background(), db)
	require.NoError(t, err)
	ctx := context.Background()
	check := func(context.Context) error { return nil }
	oldScope := sourcing.AcquisitionPlanScope{OrganizationID: "org", ActorID: "actor", MemberID: "old-member"}
	currentScope := oldScope
	currentScope.MemberID = "re-added-member"
	input := sourcing.RuleTemplateMutation{Action: "create", Name: "采购规则", Rule: sourcing.AcquisitionRule{MaxPrice: "10"}}
	var oldRule sourcing.AcquisitionRuleTemplate
	for range 100 {
		oldRule, err = repo.MutateRule(ctx, oldScope, uuid.NewString(), input, check)
		require.NoError(t, err)
	}
	visible, err := repo.ListRules(ctx, currentScope)
	require.NoError(t, err)
	require.Empty(t, visible)
	_, err = repo.ReadRule(ctx, currentScope, oldRule.ID)
	require.ErrorIs(t, err, sourcing.ErrAcquisitionNotFound)
	_, err = repo.MutateRule(ctx, currentScope, uuid.NewString(), sourcing.RuleTemplateMutation{Action: "archive", ID: oldRule.ID, ExpectedRevision: 1}, check)
	require.ErrorIs(t, err, sourcing.ErrAcquisitionNotFound)

	firstKey := uuid.NewString()
	first, err := repo.MutateRule(ctx, currentScope, firstKey, input, check)
	require.NoError(t, err, "old membership templates must not consume the current scope quota")
	for range 99 {
		_, err = repo.MutateRule(ctx, currentScope, uuid.NewString(), input, check)
		require.NoError(t, err)
	}
	visible, err = repo.ListRules(ctx, currentScope)
	require.NoError(t, err)
	require.Len(t, visible, 100)
	_, err = repo.MutateRule(ctx, currentScope, uuid.NewString(), input, check)
	require.ErrorIs(t, err, sourcing.ErrAcquisitionCapacity)
	replay, err := repo.MutateRule(ctx, currentScope, firstKey, input, check)
	require.NoError(t, err)
	require.Equal(t, first.ID, replay.ID)
	_, err = repo.MutateRule(ctx, currentScope, uuid.NewString(), sourcing.RuleTemplateMutation{Action: "archive", ID: first.ID, ExpectedRevision: 1}, check)
	require.NoError(t, err)
	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := repo.MutateRule(ctx, currentScope, uuid.NewString(), input, check)
			results <- err
		}()
	}
	created := 0
	for range 2 {
		if err := <-results; err == nil {
			created++
		} else {
			require.ErrorIs(t, err, sourcing.ErrAcquisitionCapacity)
		}
	}
	require.Equal(t, 1, created, "the existing transaction lock must admit only one final slot")
	oldVisible, err := repo.ListRules(ctx, oldScope)
	require.NoError(t, err)
	require.Len(t, oldVisible, 100, "the prior membership facts remain untouched")
}
