package dataacquisition

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/collection"
)

func TestReaderKeepsNonterminalFactsWithoutExecutionDependencies(t *testing.T) {
	for _, state := range []string{"ADMITTED", "RUNNING", "SUCCEEDED"} {
		for _, external := range []bool{false, true} {
			t.Run(state+map[bool]string{false: "/console", true: "/DataKey"}[external], func(t *testing.T) {
				scope := collection.Scope{OrganizationID: "org", ActorID: "creator", MemberID: "original"}
				repo := &resultRepository{job: Job{ID: uuid.NewString(), Scope: scope, CredentialID: uuid.NewString(), State: state}, item: Item{ID: uuid.NewString(), State: "PENDING"}}
				access := &revokedAccess{}
				reader, err := NewReader(repo, access)
				require.NoError(t, err)
				p := Principal{Scope: scope}
				if external {
					p.CredentialID, p.CredentialRevision = repo.job.CredentialID, 1
				}
				job, err := reader.Read(context.Background(), p, repo.job.ID)
				require.NoError(t, err)
				require.Equal(t, state, job.State)
				page, err := reader.Results(context.Background(), p, job.ID, "", 10, resultDependencies{})
				require.NoError(t, err)
				require.Len(t, page.Items, 1)
				require.Equal(t, "PENDING", page.Items[0].State)
				if external {
					require.Equal(t, 2, repo.guardCalls)
					p.CredentialID = uuid.NewString()
					_, err = reader.Read(context.Background(), p, job.ID)
					require.ErrorIs(t, err, ErrNotFound)
				}
				access.denied = true
				_, err = reader.Read(context.Background(), p, job.ID)
				require.ErrorIs(t, err, ErrForbidden)
			})
		}
	}
}

func TestReaderDiscardsResultsWhenCredentialGuardFails(t *testing.T) {
	scope := collection.Scope{OrganizationID: "org", ActorID: "creator", MemberID: "original"}
	p := Principal{Scope: scope, CredentialID: uuid.NewString(), CredentialRevision: 1}
	repo := &resultRepository{job: Job{ID: uuid.NewString(), Scope: scope, CredentialID: p.CredentialID, State: "RUNNING"}, guardCommitError: ErrForbidden}
	reader, err := NewReader(repo, &revokedAccess{})
	require.NoError(t, err)
	job, err := reader.Read(context.Background(), p, repo.job.ID)
	require.ErrorIs(t, err, ErrForbidden)
	require.Empty(t, job.ID)
	page, err := reader.Results(context.Background(), p, repo.job.ID, "", 10, resultDependencies{})
	require.ErrorIs(t, err, ErrForbidden)
	require.Empty(t, page.Items)
}
