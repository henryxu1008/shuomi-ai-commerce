package supplymarketpersistence

import (
	"context"
	"errors"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/supplymarket"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type verifierFixture struct{ fail error }

func (v verifierFixture) Verify(context.Context, supplymarket.SourceReference, supplymarket.PublicProduct) error {
	return v.fail
}

type filesFixture struct{}

func (filesFixture) Verify(context.Context, collection.Scope, []string) error { return nil }

type receiverFixture struct {
	tx   *gorm.DB
	fail bool
}

type restrictedReceiverFixture struct {
	receiverFixture
	readError error
}

func (r restrictedReceiverFixture) Visible(ctx context.Context, scope collection.Scope, refs []collection.Receipt, lock bool) (map[string]collection.Receipt, error) {
	if !lock && r.readError != nil {
		return nil, r.readError
	}
	return r.receiverFixture.Visible(ctx, scope, refs, lock)
}

func (r receiverFixture) Visible(_ context.Context, scope collection.Scope, refs []collection.Receipt, _ bool) (map[string]collection.Receipt, error) {
	result := map[string]collection.Receipt{}
	for _, ref := range refs {
		var count int64
		if e := r.tx.Raw("SELECT count(*) FROM receiver_test WHERE operation_id=? AND organization_id=? AND actor_id=?", ref.OperationID, scope.OrganizationID, scope.ActorID).Scan(&count).Error; e != nil {
			return nil, e
		}
		if count == 1 {
			result[ref.ItemID] = ref
		}
	}
	return result, nil
}

func (r receiverFixture) Receive(_ context.Context, scope collection.Scope, operation string, release supplymarket.Release) (collection.Receipt, error) {
	if err := r.tx.Exec("INSERT INTO receiver_test(operation_id,organization_id,actor_id) VALUES(?,?,?)", operation, scope.OrganizationID, scope.ActorID).Error; err != nil {
		return collection.Receipt{}, err
	}
	if r.fail {
		return collection.Receipt{}, errors.New("receiver failed after first SQL write")
	}
	return collection.Receipt{OperationID: operation, BatchID: collection.StableID(operation, "batch"), ItemID: collection.StableID(operation, "item"), Revision: 2}, nil
}
func command(scope collection.Scope, actor string, i supplymarket.Mutation) supplymarket.Command {
	key := uuid.NewString()
	parts := []string{scope.OrganizationID, scope.ActorID, key}
	if actor != "" {
		parts = []string{"supply-platform", actor, key}
	}
	return supplymarket.Command{Scope: scope, PlatformActor: actor, Key: key, OperationID: collection.StableID(parts...), InputHash: collection.Digest(i), Input: i}
}
func TestPostgresManualPublicationIsolationAtomicSelectionReplayAndUnknown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("supply622"), tcpostgres.WithUsername("supply_owner"), tcpostgres.WithPassword("isolated-supply-test"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	require.NoError(t, InstallSchema(db))
	require.NoError(t, db.Exec("CREATE TABLE receiver_test(operation_id uuid PRIMARY KEY,organization_id text NOT NULL,actor_id text NOT NULL)").Error)
	receiverFails := false
	verifierFails := false
	var receiverReadError error
	r, err := NewRepository(ctx, db, Dependencies{
		Disclosure: func(*gorm.DB) (DisclosureVerifier, error) {
			if verifierFails {
				return verifierFixture{fail: supplymarket.ErrForbidden}, nil
			}
			return verifierFixture{}, nil
		},
		Files: func(*gorm.DB) (FileVerifier, error) { return filesFixture{}, nil },
		Receiver: func(tx *gorm.DB) (Receiver, error) {
			return restrictedReceiverFixture{receiverFixture{tx, receiverFails}, receiverReadError}, nil
		},
	})
	require.NoError(t, err)
	scope := collection.Scope{"org-a", "actor-a", "member-a"}
	guard := func(context.Context) error { return nil }
	categoryCommand := func(action, id, parent, name string, revision int64) supplymarket.Command {
		return command(collection.Scope{}, "platform-a", supplymarket.Mutation{Action: action, ID: id, ParentCategoryID: parent, CategoryName: name, ExpectedRevision: revision})
	}
	rootCmd := categoryCommand("category_create", "", "", "家居日用", 0)
	root, err := r.Execute(ctx, rootCmd, guard)
	require.NoError(t, err)
	rootID := root.Category.After.ID
	same, err := r.Execute(ctx, rootCmd, guard)
	require.NoError(t, err)
	require.True(t, same.Replayed)
	require.Equal(t, root.Category, same.Category)
	_, err = r.Execute(ctx, categoryCommand("category_create", "", "", "家居日用", 0), guard)
	require.ErrorIs(t, err, supplymarket.ErrCategoryNameTaken)
	first, err := r.Execute(ctx, categoryCommand("category_create", "", rootID, "收纳用品", 0), guard)
	require.NoError(t, err)
	second, err := r.Execute(ctx, categoryCommand("category_create", "", rootID, "家用纺织品", 0), guard)
	require.NoError(t, err)
	secondID := second.Category.After.ID
	pick := &supplymarket.CategorySelection{ID: first.Category.After.ID, Revision: 1, ParentRevision: 1}
	submit := command(scope, "", supplymarket.Mutation{Action: "submit_selected", Selection: &supplymarket.SelectionInput{}, Supply: &supplymarket.SupplyDeclaration{Stock: 10, MinimumQuantity: 1, Province: "浙江", City: "杭州"}, FileIDs: []string{uuid.NewString()}, Disclosure: true})
	submit.Select = func(context.Context) (supplymarket.SelectedProduct, error) {
		return supplymarket.SelectedProduct{Source: supplymarket.SourceReference{Scope: scope, ItemID: uuid.NewString(), ItemRevision: 1, ProductKey: "own-product", OriginalPublicationID: "original", OriginalVersion: 1, PublicationID: "review:actual", Version: 2, ApplyID: uuid.NewString()}, Product: supplymarket.PublicProduct{Title: "优化商品", Images: []string{"https://images.example.org/source.png"}}}, nil
	}
	created, err := r.Execute(ctx, submit, guard)
	require.NoError(t, err)
	// A replay does not rebuild mutable source selection, but still checks authority.
	submit.Select = func(context.Context) (supplymarket.SelectedProduct, error) {
		return supplymarket.SelectedProduct{}, supplymarket.ErrConflict
	}
	replay, err := r.Execute(ctx, submit, guard)
	require.NoError(t, err)
	require.True(t, replay.Replayed)
	require.Equal(t, created.RecordID, replay.RecordID)
	changed := submit
	changed.InputHash = collection.Digest("different")
	_, err = r.Execute(ctx, changed, guard)
	require.ErrorIs(t, err, supplymarket.ErrConflict)
	_, err = r.ReadRecord(ctx, collection.Scope{"org-a", "other", "other-member"}, "", created.RecordID)
	require.ErrorIs(t, err, supplymarket.ErrNotFound)
	current, err := r.ReadRecord(ctx, scope, "", created.RecordID)
	require.NoError(t, err)
	require.Equal(t, scope, current.Source.Scope)
	require.Equal(t, supplymarket.Submitted, current.Stage)
	platform := func(action string, revision int64, cooperation bool) supplymarket.Command {
		mutation := supplymarket.Mutation{Action: action, ID: created.RecordID, ExpectedRevision: revision, Note: "人工核实", CooperationConfirmed: cooperation}
		if action == "approve" || action == "publish" || action == "confirm_category" {
			mutation.Category = pick
		}
		return command(collection.Scope{}, "platform-a", mutation)
	}
	_, err = r.Execute(ctx, platform("publish", 1, false), guard)
	require.ErrorIs(t, err, supplymarket.ErrConflict)
	evaluated, err := r.Execute(ctx, platform("evaluate", 1, false), guard)
	require.NoError(t, err)
	_, err = r.Execute(ctx, platform("approve", evaluated.Revision, false), guard)
	require.ErrorIs(t, err, supplymarket.ErrInvalid)
	approved, err := r.Execute(ctx, platform("approve", evaluated.Revision, true), guard)
	require.NoError(t, err)
	market, err := r.ListReleases(ctx, supplymarket.Query{Limit: 10})
	require.NoError(t, err)
	require.Empty(t, market.Items, "approval is not publication")
	_, err = r.Execute(ctx, categoryCommand("category_disable", pick.ID, "", "", 1), guard)
	require.NoError(t, err)
	_, err = r.Execute(ctx, platform("publish", approved.Revision, false), guard)
	require.ErrorIs(t, err, supplymarket.ErrCategoryChanged)
	pick = &supplymarket.CategorySelection{ID: secondID, Revision: 1, ParentRevision: 1}
	reconfirmed, err := r.Execute(ctx, platform("confirm_category", approved.Revision, false), guard)
	require.NoError(t, err)
	app, err := r.ReadRecord(ctx, collection.Scope{}, "platform-a", created.RecordID)
	require.NoError(t, err)
	require.Equal(t, supplymarket.Approved, app.Stage)
	require.Equal(t, secondID, app.Category.Child.ID)
	published, err := r.Execute(ctx, platform("publish", reconfirmed.Revision, false), guard)
	require.NoError(t, err)
	_, err = r.Execute(ctx, categoryCommand("category_disable", rootID, "", "", 1), guard)
	require.NoError(t, err)
	for _, categoryID := range []string{rootID, secondID} {
		page, err := r.ListReleases(ctx, supplymarket.Query{Kind: "selected", Keyword: "优化", Limit: 1, Filters: supplymarket.MarketFilters{CategoryID: categoryID}})
		require.NoError(t, err)
		require.EqualValues(t, 1, page.Total)
		require.Equal(t, published.ReleaseID, page.Items[0].ID)
	}
	publicCategories, err := r.ListCategories(ctx, supplymarket.CategoryQuery{Kind: "selected", ParentID: rootID, Limit: 10})
	require.NoError(t, err)
	require.Len(t, publicCategories.Items, 1)
	require.Equal(t, secondID, publicCategories.Items[0].ID)
	otherChannel, err := r.ListCategories(ctx, supplymarket.CategoryQuery{Kind: "official", ParentID: rootID, Limit: 10})
	require.NoError(t, err)
	require.Empty(t, otherChannel.Items)
	_, err = r.Execute(ctx, categoryCommand("category_enable", rootID, "", "", 2), guard)
	require.NoError(t, err)
	pick = &supplymarket.CategorySelection{ID: secondID, Revision: 1, ParentRevision: 3}
	consumer := collection.Scope{"org-b", "actor-b", "member-b"}
	selectCmd := command(consumer, "", supplymarket.Mutation{Action: "select_release", ID: published.ReleaseID, ExpectedRevision: 1})
	receiverFails = true
	_, err = r.Execute(ctx, selectCmd, guard)
	require.Error(t, err)
	var count int64
	require.NoError(t, db.Raw("SELECT count(*) FROM receiver_test").Scan(&count).Error)
	require.Zero(t, count, "a partial receiver save rolls back with the command")
	receiverFails = false
	_, err = r.Execute(ctx, selectCmd, func(context.Context) error { return supplymarket.ErrForbidden })
	require.ErrorIs(t, err, supplymarket.ErrForbidden)
	require.NoError(t, db.Raw("SELECT count(*) FROM receiver_test").Scan(&count).Error)
	require.Zero(t, count, "commit guard rollback includes receiver writes")
	r.afterCommit = func() error { return errors.New("lost commit response") }
	_, err = r.Execute(ctx, selectCmd, guard)
	require.ErrorIs(t, err, supplymarket.ErrUnknown)
	r.afterCommit = nil
	selected, err := r.Execute(ctx, selectCmd, guard)
	require.NoError(t, err)
	require.True(t, selected.Replayed)
	require.NotNil(t, selected.Collection)
	require.NoError(t, db.Raw("SELECT count(*) FROM receiver_test").Scan(&count).Error)
	require.Equal(t, int64(1), count)
	// Different browser keys for the same exact release must reuse the owned item.
	type concurrentResult struct {
		receipt supplymarket.Receipt
		err     error
	}
	done := make(chan concurrentResult, 2)
	for range 2 {
		go func() {
			value, e := r.Execute(ctx, command(consumer, "", supplymarket.Mutation{Action: "select_release", ID: published.ReleaseID, ExpectedRevision: 1}), guard)
			done <- concurrentResult{value, e}
		}()
	}
	for range 2 {
		value := <-done
		require.NoError(t, value.err)
		require.Equal(t, selected.Collection.ItemID, value.receipt.Collection.ItemID)
	}
	require.NoError(t, db.Raw("SELECT count(*) FROM receiver_test").Scan(&count).Error)
	require.EqualValues(t, 1, count)
	data, e := r.ReadMarketData(ctx, consumer, []string{published.ReleaseID})
	require.NoError(t, e)
	require.NotNil(t, data[published.ReleaseID].Added)
	require.True(t, *data[published.ReleaseID].Added)
	require.Equal(t, "家用纺织品", data[published.ReleaseID].Category.Child.Name)
	_, err = r.Execute(ctx, categoryCommand("category_rename", secondID, "", "家庭纺织品", 1), guard)
	require.NoError(t, err)
	updated, err := r.ReadMarketData(ctx, consumer, []string{published.ReleaseID})
	require.NoError(t, err)
	require.Equal(t, "家庭纺织品", updated[published.ReleaseID].Category.Child.Name)
	require.True(t, *updated[published.ReleaseID].Added)
	pick = &supplymarket.CategorySelection{ID: secondID, Revision: 2, ParentRevision: 3}
	foreign, e := r.ReadMarketData(ctx, collection.Scope{consumer.OrganizationID, consumer.ActorID, "another-member"}, []string{published.ReleaseID})
	require.NoError(t, e)
	require.False(t, *foreign[published.ReleaseID].Added)
	// A denied status read is unknown. It does not become a new permission
	// prerequisite for the existing manage-authorized selection command.
	receiverReadError = collection.ErrForbidden
	unknown, e := r.ReadMarketData(ctx, consumer, []string{published.ReleaseID})
	require.NoError(t, e)
	require.Nil(t, unknown[published.ReleaseID].Added)
	reused, e := r.Execute(ctx, command(consumer, "", supplymarket.Mutation{Action: "select_release", ID: published.ReleaseID, ExpectedRevision: 1}), guard)
	require.NoError(t, e)
	require.Equal(t, selected.Collection.ItemID, reused.Collection.ItemID)
	require.NoError(t, db.Raw("SELECT count(*) FROM receiver_test").Scan(&count).Error)
	require.EqualValues(t, 1, count)
	receiverReadError = errors.New("collection store unavailable")
	_, e = r.ReadMarketData(ctx, consumer, []string{published.ReleaseID})
	require.ErrorIs(t, e, receiverReadError, "dependency failure must not be disguised as an unselected item")
	receiverReadError = nil
	revoke := command(collection.Scope{}, "platform-a", supplymarket.Mutation{Action: "revoke", ID: published.ReleaseID, ExpectedRevision: 1})
	_, err = r.Execute(ctx, revoke, guard)
	require.NoError(t, err)
	_, err = r.ReadRelease(ctx, published.ReleaseID)
	require.ErrorIs(t, err, supplymarket.ErrNotFound)
	_, err = r.Execute(ctx, command(consumer, "", supplymarket.Mutation{Action: "select_release", ID: published.ReleaseID, ExpectedRevision: 1}), guard)
	require.ErrorIs(t, err, supplymarket.ErrNotFound)
	selected, err = r.Execute(ctx, selectCmd, guard)
	require.NoError(t, err)
	require.True(t, selected.Replayed, "retained selection receipt survives source revocation")
	events, err := r.ListEvents(ctx, scope, "", created.RecordID, supplymarket.Query{Limit: 50})
	require.NoError(t, err)
	require.Equal(t, int64(6), events.Total, "publication revocation belongs in the original application's progress")
	// The platform must be the original drafter for official supply.
	official := command(scope, "", supplymarket.Mutation{Action: "create_official_draft", Supply: submit.Input.Supply, Disclosure: true})
	official.Select = func(context.Context) (supplymarket.SelectedProduct, error) {
		return supplymarket.SelectedProduct{Source: *current.Source, Product: *current.Product}, nil
	}
	draft, err := r.Execute(ctx, official, guard)
	require.NoError(t, err)
	publishDraft := command(collection.Scope{}, "platform-a", supplymarket.Mutation{Action: "publish", ID: draft.RecordID, ExpectedRevision: 1, Category: pick})
	_, err = r.Execute(ctx, publishDraft, guard)
	require.ErrorIs(t, err, supplymarket.ErrForbidden)
	publishDraft = command(collection.Scope{}, scope.ActorID, publishDraft.Input)
	verifierFails = true
	_, err = r.Execute(ctx, publishDraft, guard)
	require.ErrorIs(t, err, supplymarket.ErrForbidden)
	verifierFails = false
	officialPublished, err := r.Execute(ctx, publishDraft, guard)
	require.NoError(t, err)
	officialPage, err := r.ListReleases(ctx, supplymarket.Query{Kind: "official", Limit: 10, Filters: supplymarket.MarketFilters{CategoryID: secondID}})
	require.NoError(t, err)
	require.Len(t, officialPage.Items, 1)
	require.Equal(t, officialPublished.ReleaseID, officialPage.Items[0].ID)
	t.Run("qualification files retain exact private owner and attachment", func(t *testing.T) {
		file := supplymarket.PrivateFile{ID: submit.Input.FileIDs[0], Owner: scope, ContentType: "application/pdf", SHA256: collection.Digest("qualification"), Size: 40, CreatedAt: time.Now().UTC()}
		file.ObjectKey = supplymarket.PrivateObjectKey(scope, file.ID, file.SHA256)
		_, err := r.SaveUpload(ctx, file, func(context.Context) error { return supplymarket.ErrForbidden })
		require.ErrorIs(t, err, supplymarket.ErrForbidden)
		_, err = r.ReadUpload(ctx, scope, file.ID)
		require.ErrorIs(t, err, supplymarket.ErrNotFound, "revoked commit rolls back file ownership")
		_, err = r.SaveUpload(ctx, file, guard)
		require.NoError(t, err)
		_, err = r.SaveUpload(ctx, file, guard)
		require.NoError(t, err, "same immutable upload can replay")
		for _, denied := range []collection.Scope{consumer, {"org-a", "actor-a", "replacement-member"}, {"org-a", "other", "other-member"}} {
			_, err = r.ReadUpload(ctx, denied, file.ID)
			require.ErrorIs(t, err, supplymarket.ErrNotFound)
			_, err = r.ReadAttachedFile(ctx, denied, "", created.RecordID, file.ID)
			require.ErrorIs(t, err, supplymarket.ErrNotFound)
		}
		for _, actor := range []string{"", "platform-a"} {
			readScope := scope
			if actor != "" {
				readScope = collection.Scope{}
			}
			read, err := r.ReadAttachedFile(ctx, readScope, actor, created.RecordID, file.ID)
			require.NoError(t, err)
			require.Equal(t, file.Owner, read.Owner)
			require.Equal(t, file.SHA256, read.SHA256)
		}
		unattached := file
		unattached.ID = uuid.NewString()
		unattached.ObjectKey = supplymarket.PrivateObjectKey(scope, unattached.ID, unattached.SHA256)
		_, err = r.SaveUpload(ctx, unattached, guard)
		require.NoError(t, err)
		_, err = r.ReadAttachedFile(ctx, collection.Scope{}, "platform-a", created.RecordID, unattached.ID)
		require.ErrorIs(t, err, supplymarket.ErrNotFound, "platform record access does not expose unattached uploads")
		changed := file
		changed.Owner = consumer
		changed.ObjectKey = supplymarket.PrivateObjectKey(consumer, changed.ID, changed.SHA256)
		_, err = r.SaveUpload(ctx, changed, guard)
		require.ErrorIs(t, err, supplymarket.ErrConflict, "an immutable upload ID cannot be reassigned")
	})
}
