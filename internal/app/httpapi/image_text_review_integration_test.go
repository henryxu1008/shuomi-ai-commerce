package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/authidentity"
	"task-processor/internal/imageagent"
)

// Runs inside the existing isolated PostgreSQL content-owner fixture. Its model
// is an httptest server; this exercises real owner records, never a real provider.
func testConfirmedImageTextOwner(t *testing.T, app *productAgentApplication, server *httptest.Server, path, key string, calls *atomic.Int32) {
	t.Helper()
	before := calls.Load()
	code, raw, err := acquisitionHTTPRequest(server, "POST", path+"/"+key+"/review", "admin", "B", "", `{}`)
	require.NoError(t, err)
	require.Equal(t, 200, code, string(raw))
	var link struct {
		ProposalID string `json:"proposalId"`
	}
	require.NoError(t, json.Unmarshal(raw, &link))
	view := titleCall(t, server, "GET", titleBasePath+"/"+link.ProposalID, "admin", "B", "", "", 200)
	i := authidentity.AuthenticatedIdentity{TenantID: "B", EffectiveOrganizationID: "B", HomeOrganizationID: "A", UserID: "admin", EffectiveMemberID: "member-B-admin", TokenExpiresAt: time.Now().Add(time.Hour), Roles: []string{"listingkit_admin"}}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), i)
	ctx, err = (productReviewCapabilityBinder{now: time.Now}).Bind(ctx, "Bearer admin")
	require.NoError(t, err)
	identity := imageagent.ExecutionIdentity{TenantID: "B", UserID: "admin", MemberID: "member-B-admin"}
	reader := imageTextReviewReader{content: app}
	ref := imageagent.ImageTextReviewReference{ID: view.ID, Revision: view.Revision}
	_, err = reader.ReadConfirmedImageText(ctx, identity, ref)
	require.ErrorIs(t, err, imageagent.ErrRevisionConflict, "pending copy is not consent")
	accepted := titleDecision(t, server, view, "accept", "admin", "", 200)
	_, err = reader.ReadConfirmedImageText(ctx, identity, ref)
	require.ErrorIs(t, err, imageagent.ErrRevisionConflict, "old revision cannot borrow new consent")
	ref.Revision = accepted.Revision
	copy, err := reader.ReadConfirmedImageText(ctx, identity, ref)
	require.NoError(t, err)
	require.Equal(t, "en", copy.Language)
	require.Equal(t, view.Input.ProductKey, copy.ProductID)
	require.Equal(t, view.Input.BaseVersion, copy.Version)
	require.Equal(t, view.Execution.Binding.PublicationID, copy.PublicationID)
	require.Len(t, copy.Lines, 2)
	require.Equal(t, ref, copy.Review)
	_, err = reader.ReadConfirmedImageText(ctx, imageagent.ExecutionIdentity{TenantID: "A", UserID: "admin", MemberID: "member-A-admin"}, ref)
	require.ErrorIs(t, err, imageagent.ErrIdentityRequired)
	i.TenantID, i.EffectiveOrganizationID, i.EffectiveMemberID = "A", "A", "member-A-admin"
	other := authidentity.WithAuthenticatedIdentity(context.Background(), i)
	other, err = (productReviewCapabilityBinder{now: time.Now}).Bind(other, "Bearer admin")
	require.NoError(t, err)
	_, err = reader.ReadConfirmedImageText(other, imageagent.ExecutionIdentity{TenantID: "A", UserID: "admin", MemberID: "member-A-admin"}, ref)
	require.Error(t, err, "another enterprise cannot resolve this review")
	_, err = app.configuration.ExecuteMode(context.Background(), agentconfig.ModeCommand{Identity: agent.ExecutionIdentity{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"}, Subject: app.modeSubject(), Key: uuid.NewString(), Mode: agent.ModeAutoApprove, ExpectedRevision: 1})
	require.NoError(t, err)
	restored, err := reader.ReadConfirmedImageText(ctx, identity, ref)
	require.NoError(t, err, "current mode must not replace the original default snapshot")
	require.Equal(t, copy, restored)
	current := titleCall(t, server, "GET", titleBasePath+"/"+view.ID, "admin", "B", "", "", 200)
	require.Equal(t, "accepted", current.State)
	require.Nil(t, current.Receipt, "image preparation must never apply Catalog")
	require.Nil(t, current.PolicyAdoption)
	require.Equal(t, before, calls.Load(), "read, review and mode change cannot translate again")
}
