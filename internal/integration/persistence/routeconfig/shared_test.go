package routeconfigstore

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
)

type sharedCatalog struct{ unready bool }

func (*sharedCatalog) ValidateDefaultChannel(_ context.Context, _ string, channel string) error {
	if channel != "synthetic-channel" {
		return d.ErrNeedsConfiguration
	}
	return nil
}

func (c *sharedCatalog) QualifyModel(_ context.Context, org string, category d.ModelCategory, ref d.ModelBindingReference) (d.ModelQualification, error) {
	return syntheticQualifiedModel(d.ModelQualification{OrganizationID: org, Category: category, Reference: ref, Channel: "synthetic-channel", Model: ref.ID,
		Ready: !c.unready, Reason: "BUDGET_BOUND_MISSING", ProfileVersion: "profile-v1", BoundEvidenceVersion: "synthetic-bound",
		Price: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "official-price-v1", FXVersion: "synthetic-fx-v1", Currency: "USD", CNYMicrosPerCurrencyUnit: 7_000_000, Model: ref.ID, Channel: "synthetic-channel", UsageVersion: "synthetic", Rates: aicapability.OfficialModelRates{InputText: 750_000, OutputText: 4_500_000}}}), nil
}

func sharedInput() d.SharedModelInput {
	return d.SharedModelInput{ExpectedRevision: d.NoRevision, Selection: d.SharedModelSelection{SchemaVersion: 1, UnitVersion: aicapability.AIPointMicrounitV1, DefaultChannel: "synthetic-channel",
		Text: d.ModelBindingReference{ID: "synthetic-mini", Version: "binding-v1"}, Image: d.ModelBindingReference{ID: "synthetic-image", Version: "binding-v1"}, Multimodal: d.ModelBindingReference{ID: "synthetic-mini", Version: "binding-v1"}}}
}

func TestSharedSelectionAtomicPublicationReplayAndRevocation(t *testing.T) {
	db, repo := fixture(t)
	ctx, catalog := context.Background(), &sharedCatalog{}
	service := &d.SharedModelService{Repository: repo, Catalog: catalog}
	input := sharedInput()
	first, err := service.Publish(ctx, "org-a", "actor", "original-key", input)
	require.NoError(t, err)
	require.Equal(t, input.Selection, *first.Payload.SharedModels)
	require.Equal(t, d.Scope{OrganizationID: "org-a", Capability: d.SharedModelCapability}, first.Scope)
	var count int64
	require.NoError(t, db.Table("ai_route_config_current").Count(&count).Error)
	require.EqualValues(t, 1, count, "all categories have one canonical current")
	_, _, err = service.ReadCategory(ctx, "org-b", d.ModelCategoryText)
	require.ErrorIs(t, err, d.ErrNotFound)

	catalog.unready = true
	_, _, err = service.ReadCategory(ctx, "org-a", d.ModelCategoryText)
	require.ErrorIs(t, err, d.ErrNeedsConfiguration)
	replay, err := service.Publish(ctx, "org-a", "actor", "original-key", input)
	require.NoError(t, err, "original receipt lookup precedes current mutable qualification")
	require.Equal(t, first.ID, replay.ID)
	input.Selection.Image.Version = "binding-v2"
	_, err = service.Publish(ctx, "org-a", "actor", "original-key", input)
	require.ErrorIs(t, err, d.ErrConflict)
	_, err = service.Publish(ctx, "org-a", "actor", "new-unready-key", input)
	require.ErrorIs(t, err, d.ErrNeedsConfiguration)
	require.NoError(t, db.Table("ai_route_config_revisions").Count(&count).Error)
	require.EqualValues(t, 1, count, "no partial revision can survive failed qualification")
}

func TestSharedSelectionConcurrentCASAndCommitAckRecovery(t *testing.T) {
	db, repo := fixture(t)
	ctx, catalog := context.Background(), &sharedCatalog{}
	service := &d.SharedModelService{Repository: repo, Catalog: catalog}
	input := sharedInput()
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, key := range []string{"first-writer", "second-writer"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			_, err := service.Publish(ctx, "org-a", "actor", key, input)
			errs <- err
		}(key)
	}
	wg.Wait()
	close(errs)
	success, conflict := 0, 0
	for err := range errs {
		if err == nil {
			success++
		} else {
			require.ErrorIs(t, err, d.ErrConflict)
			conflict++
		}
	}
	require.Equal(t, 1, success)
	require.Equal(t, 1, conflict)
	current, err := repo.Current(ctx, d.Scope{OrganizationID: "org-a", Capability: d.SharedModelCapability})
	require.NoError(t, err)
	input.ExpectedRevision = current.ID
	repo.commitTransaction = func(tx *gorm.DB) error {
		require.NoError(t, tx.Commit().Error)
		return errors.New("synthetic commit acknowledgment lost")
	}
	_, err = service.Publish(ctx, "org-a", "actor", "ack-lost-key", input)
	require.ErrorIs(t, err, d.ErrOutcomeUnknown)
	repo.commitTransaction = nil
	catalog.unready = true
	recovered, err := service.Publish(ctx, "org-a", "actor", "ack-lost-key", input)
	require.NoError(t, err)
	require.NotEqual(t, current.ID, recovered.ID)
	var count int64
	require.NoError(t, db.Table("ai_route_config_revisions").Count(&count).Error)
	require.EqualValues(t, 2, count, "ack recovery never creates a second revision")
}
