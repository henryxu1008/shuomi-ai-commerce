package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/core/config"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/review"
)

type contentReviewIAM struct {
	server                        *httptest.Server
	revoked, failed, rolesRevoked atomic.Bool
	calls                         atomic.Int32
}

func newContentReviewIAM(t *testing.T, replaced *atomic.Bool) *contentReviewIAM {
	t.Helper()
	iam := &contentReviewIAM{}
	iam.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		iam.calls.Add(1)
		require.Equal(t, "Bearer controlled-service-token", r.Header.Get("Authorization"))
		if iam.failed.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if iam.revoked.Load() {
			fmt.Fprint(w, `{"pagination":{"totalResult":"0"},"authorizations":[]}`)
			return
		}
		member, role := "member-B-admin", "listingkit_admin"
		if replaced.Load() {
			member += "-replacement"
		}
		if iam.rolesRevoked.Load() {
			role = "listingkit_viewer"
		}
		fmt.Fprintf(w, `{"pagination":{"totalResult":"1"},"authorizations":[{"id":%q,"project":{"id":"project"},"organization":{"id":"B"},"user":{"id":"admin"},"state":"STATE_ACTIVE","roles":[{"key":%q}]}]}`, member, role)
	}))
	t.Cleanup(iam.server.Close)
	return iam
}

func (iam *contentReviewIAM) config() *config.Config {
	if iam == nil {
		return nil
	}
	cfg := &config.Config{}
	cfg.ListingKit.Zitadel.AuthorizationAPIURL = iam.server.URL
	cfg.ListingKit.Zitadel.TenantDirectoryToken = "controlled-service-token"
	cfg.ListingKit.Zitadel.ProjectID = "project"
	return cfg
}

func TestContentCrossAdministratorReviewRevalidatesOriginalExecution(t *testing.T) {
	for _, change := range []string{"healthy", "member replaced", "revoked before apply", "roles revoked", "IAM unavailable"} {
		t.Run(change, func(t *testing.T) { testProductAgentOwners(t, "content flow cross-admin "+change) })
	}
}

func testContentCrossAdministratorReview(t *testing.T, change string, iam *contentReviewIAM, replaced *atomic.Bool, view review.View, server *httptest.Server, calls *atomic.Int32, f *acquisitionHTTPFixture) {
	t.Helper()
	before := calls.Load()
	if change == "member replaced" {
		replaced.Store(true)
	}
	if change == "roles revoked" {
		iam.rolesRevoked.Store(true)
	}
	if change == "IAM unavailable" {
		iam.failed.Store(true)
	}
	status := http.StatusOK
	if change == "member replaced" || change == "roles revoked" {
		status = http.StatusForbidden
	}
	if change == "IAM unavailable" {
		status = http.StatusServiceUnavailable
	}
	accepted := titleCall(t, server, "POST", titleBasePath+"/"+view.ID+"/decisions", "reviewer", "B", uuid.NewString(), `{"action":"accept","expected_revision":1}`, status)
	if status == http.StatusOK {
		if change == "revoked before apply" {
			iam.revoked.Store(true)
			status = http.StatusForbidden
		}
		titleCall(t, server, "POST", titleBasePath+"/"+view.ID+"/apply", "reviewer", "B", uuid.NewString(), fmt.Sprintf(`{"expected_revision":%d}`, accepted.Revision), status)
	}
	require.Equal(t, before, calls.Load(), "human review must never resend the model")
	require.Positive(t, iam.calls.Load(), "cross-admin writes must independently read original IAM membership")
	var applied int64
	require.NoError(t, f.owner.Table("product_title_proposals").Where("state = ?", "applied").Count(&applied).Error)
	if change == "healthy" {
		require.EqualValues(t, 1, applied)
	} else {
		require.Zero(t, applied)
	}
	if change != "healthy" {
		reader, err := catalogstore.NewBoundedSnapshotReader(f.owner, 2<<20)
		require.NoError(t, err)
		current, err := reader.GetCurrentSnapshot(context.Background(), catalog.SnapshotIdentity{TenantID: "B", ProductKey: view.Input.ProductKey})
		require.NoError(t, err)
		require.EqualValues(t, 1, current.Version)
	}
}
