package supplychainapp

import (
	"context"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"task-processor/internal/authruntime/zitadel"
	"task-processor/internal/authz"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

func TestPublicationPurposeExplicitlySeparatesReadAndWrite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"pagination":{"totalResult":"1"},"authorizations":[{"id":"member-a","project":{"id":"project-a"},"organization":{"id":"org-a"},"user":{"id":"actor-a"},"state":"STATE_ACTIVE","roles":[{"key":"listingkit_admin"}]}]}`))
	}))
	defer server.Close()
	policy, e := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, e)
	auth := OrganizationExecutionAuthorizer{Client: zitadel.NewAuthorizationClient(server.URL, server.Client()), ServiceToken: func(context.Context) (string, error) { return "fixture", nil }, ProjectID: "project-a", Permissions: policy}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	for _, tc := range []struct {
		purpose string
		write   bool
	}{{storecenter.ProductPurposeStatus, false}, {storecenter.ProductPurposeShelfOn, true}, {storecenter.ProductPurposeRules, false}} {
		v, e := auth.AuthorizeProductExecution(ctx, storecenter.ProductExecutionSubject{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a", Purpose: tc.purpose})
		require.NoError(t, e)
		require.True(t, v.Allowed)
		require.Equal(t, tc.write, v.Access.CanWrite)
	}
	_, e = auth.AuthorizeProductExecution(ctx, storecenter.ProductExecutionSubject{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a", Purpose: "unknown-purpose"})
	require.Error(t, e)
}
