package completiondraft

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
)

type historyRecords struct {
	*testRecords
	history   HistoryPage
	revision  Record
	afterRead func()
}

func (r *historyRecords) ListHistory(context.Context, collection.Scope, string, HistoryQuery) (HistoryPage, error) {
	if r.afterRead != nil {
		r.afterRead()
	}
	return r.history, nil
}
func (r *historyRecords) ReadRevision(context.Context, collection.Scope, string, int64) (Record, error) {
	if r.afterRead != nil {
		r.afterRead()
	}
	return r.revision, nil
}

func TestHistoryReadsValidateExactSavedIdentityAndLiveAuthority(t *testing.T) {
	for _, change := range []string{"none", "source", "draft", "future", "order", "cursor", "revoked", "member", "cancelled"} {
		t.Run(change, func(t *testing.T) {
			s, a, r, ctx, source := fixture(t)
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			draft := collection.StableID(a.scope.OrganizationID, a.scope.ActorID, source.ID, "completion-draft")
			old := Record{ID: uuid.NewString(), DraftID: draft, Revision: 1, Source: source, Content: Content{Title: "old", Description: "\n"}, Status: StatusPending, CreatedAt: time.Now()}
			latest := old
			latest.ID = uuid.NewString()
			latest.Revision = 2
			latest.Content.Title = "new"
			h := &historyRecords{testRecords: r, revision: old, history: HistoryPage{Source: source, DraftID: draft, CurrentRevision: 2, Items: []Record{latest, old}}}
			s.repo = h
			h.afterRead = func() {
				switch change {
				case "source":
					h.history.Items[1].Source.ID = uuid.NewString()
					h.revision.Source.ID = uuid.NewString()
				case "draft":
					h.history.Items[1].DraftID = uuid.NewString()
					h.revision.DraftID = uuid.NewString()
				case "future":
					h.history.Items[0].Revision = 3
					h.revision.Revision = 3
				case "order":
					h.history.Items[1].Revision = 2
				case "cursor":
					h.history.NextBefore = 2
				case "revoked":
					a.denied = true
				case "member":
					a.scope.MemberID = "other-member"
				case "cancelled":
					cancel()
				}
			}
			page, err := s.ListHistory(ctx, source.ID, HistoryQuery{Limit: 20})
			if change != "none" {
				require.Error(t, err)
				require.Empty(t, page.Items)
				return
			}
			require.NoError(t, err)
			require.Equal(t, h.history, page)
			got, err := s.ReadRevision(ctx, source.ID, 1)
			require.NoError(t, err)
			require.Equal(t, old, got)
		})
	}
}

func TestExactHistoryRevisionCannotReturnAnotherRevisionOrDraft(t *testing.T) {
	for _, change := range []string{"revision", "draft", "source", "revoked"} {
		t.Run(change, func(t *testing.T) {
			s, a, r, ctx, source := fixture(t)
			h := &historyRecords{testRecords: r, revision: Record{ID: uuid.NewString(), DraftID: collection.StableID(a.scope.OrganizationID, a.scope.ActorID, source.ID, "completion-draft"), Revision: 1, Source: source}}
			s.repo = h
			h.afterRead = func() {
				switch change {
				case "revision":
					h.revision.Revision = 2
				case "draft":
					h.revision.DraftID = uuid.NewString()
				case "source":
					h.revision.Source.Source.Version = 2
				case "revoked":
					a.denied = true
				}
			}
			got, err := s.ReadRevision(ctx, source.ID, 1)
			require.Error(t, err)
			require.Empty(t, got)
		})
	}
}

func TestHistoryRevisionAndQueryAreCanonicalAndBounded(t *testing.T) {
	for _, value := range []string{"0", "01", "+1", "1e2", "-1", "9007199254740992", ""} {
		_, err := ParseRevision(value)
		require.ErrorIs(t, err, preparation.ErrInvalid)
	}
	for _, value := range []string{"1", "9007199254740991"} {
		_, err := ParseRevision(value)
		require.NoError(t, err)
	}
	s, _, _, ctx, source := fixture(t)
	for _, q := range []HistoryQuery{{Limit: 101}, {Limit: -1}, {Limit: 20, Before: -1}, {Limit: 20, Before: MaxHistoryRevision + 1}} {
		_, err := s.ListHistory(ctx, source.ID, q)
		require.ErrorIs(t, err, preparation.ErrInvalid)
	}
	_, err := s.ReadRevision(ctx, source.ID, 0)
	require.ErrorIs(t, err, preparation.ErrInvalid)
}
