package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	"task-processor/internal/product/catalog"
	d "task-processor/internal/skill"
	"testing"
	"time"
)

func TestSkillConsoleClientsUseActualBFFAndOwners(t *testing.T) {
	if os.Getenv("ISSUE688_FRONTEND_TEST") != "1" {
		t.Skip("requires installed UI dependencies and task-owned HTTP/PostgreSQL fixture")
	}
	testProductAgentOwners(t, "content flow skill bff")
}
func testSkillBFFClient(t *testing.T, f *acquisitionHTTPFixture, server *httptest.Server, operation, modeRevision string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	args := []string{"exec", "vitest", "run", "--config", "e2e/issue688-skill.config.ts"}
	command := exec.CommandContext(ctx, "pnpm", args...)
	if runtime.GOOS == "windows" {
		command = exec.CommandContext(ctx, "cmd", "/c", "pnpm.cmd")
		command.Args = append(command.Args, args...)
	}
	web, e := filepath.Abs(filepath.Join("..", "..", "..", "web", "listingkit-ui"))
	require.NoError(t, e)
	command.Dir = web
	command.Env = append(os.Environ(), "SKILL_INTEGRATION_ORIGIN="+server.URL, "SKILL_INTEGRATION_OPERATION="+operation, "SKILL_INTEGRATION_MODE_REVISION="+modeRevision)
	output, e := command.CombinedOutput()
	t.Log(string(output))
	require.NoError(t, e)
	var adopted int64
	require.NoError(t, f.owner.Table("product_title_proposals").Where("state='applied'").Count(&adopted).Error)
	require.EqualValues(t, 1, adopted)
}

func TestSkillHTTPToGovernedContentReviewAndCatalog(t *testing.T) {
	for _, mode := range []string{"default", "auto_approve", "read_only", "full_access", "cross-admin healthy", "cross-admin revoked before apply", "cross-admin version revoked before apply"} {
		t.Run(mode, func(t *testing.T) { testProductAgentOwners(t, "content flow skill "+mode) })
	}
}
func skillHTTP(t *testing.T, server *httptest.Server, method, path, actor, org, key, match string, payload any, status int) []byte {
	t.Helper()
	var body io.Reader
	if payload != nil {
		raw, e := json.Marshal(payload)
		require.NoError(t, e)
		body = strings.NewReader(string(raw))
	}
	r, e := http.NewRequest(method, server.URL+path, body)
	require.NoError(t, e)
	r.Header.Set("Authorization", "Bearer "+actor)
	if org != "" {
		r.Header.Set("X-Requested-Organization-ID", org)
	}
	if key != "" {
		r.Header.Set("Idempotency-Key", key)
	}
	if match != "" {
		r.Header.Set("If-Match", `"`+match+`"`)
	}
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	response, e := server.Client().Do(r)
	require.NoError(t, e)
	defer response.Body.Close()
	raw, e := io.ReadAll(response.Body)
	require.NoError(t, e)
	require.Equal(t, status, response.StatusCode, string(raw))
	return raw
}
func testSkillContentFlow(t *testing.T, f *acquisitionHTTPFixture, server *httptest.Server, cfg *configstore.Store, app *productAgentApplication, operationID, mode, modeRevision string, iam *contentReviewIAM, replaced *atomic.Bool, calls *atomic.Int32, crossChange string) {
	t.Helper()
	require.NotNil(t, app.skills)
	input := map[string]any{"files": []d.File{{Path: "SKILL.md", Text: "---\nname: product-title-normalization\ndescription: Normalize exact saved facts.\n---\nNormalize exact saved facts, using [style](references/style.md).\n"}, {Path: "references/style.md", Text: "Keep supported attributes and never invent facts.\n"}}, "allowCopy": true, "allowExport": true}
	var official d.Receipt
	raw := skillHTTP(t, server, "POST", "/api/v1/workbench/skills/official", "admin", "", uuid.NewString(), "", input, 200)
	require.NoError(t, json.Unmarshal(raw, &official))
	lifecycle := func(space string, r d.Receipt, action string) d.Receipt {
		raw := skillHTTP(t, server, "POST", fmt.Sprintf("/api/v1/workbench/skills/%s/%s/versions/%d/%s", space, r.ID, r.Version, action), "admin", "", uuid.NewString(), fmt.Sprint(r.Revision), map[string]any{}, 200)
		var out d.Receipt
		require.NoError(t, json.Unmarshal(raw, &out))
		return out
	}
	official = lifecycle("official", official, "publish")
	official = lifecycle("official", official, "enable")
	var publication struct {
		Version d.Version `json:"version"`
	}
	raw = skillHTTP(t, server, "GET", fmt.Sprintf("/api/v1/workbench/skills/official/%s/versions/1", official.ID), "admin", "", "", "", nil, 200)
	require.NoError(t, json.Unmarshal(raw, &publication))
	var personal d.Receipt
	raw = skillHTTP(t, server, "POST", "/api/v1/workbench/skills/personal/copy", "admin", "", uuid.NewString(), "", map[string]any{"source": publication.Version.Ref(), "allowCopy": true, "allowExport": true}, 200)
	require.NoError(t, json.Unmarshal(raw, &personal))
	personal = lifecycle("personal", personal, "publish")
	personal = lifecycle("personal", personal, "enable")
	var chosen struct {
		Version d.Version `json:"version"`
	}
	raw = skillHTTP(t, server, "GET", fmt.Sprintf("/api/v1/workbench/skills/personal/%s/versions/1", personal.ID), "admin", "", "", "", nil, 200)
	require.NoError(t, json.Unmarshal(raw, &chosen))
	ref := chosen.Version.Ref()
	// A lawful reviewer still cannot browse another person's unshared package.
	skillHTTP(t, server, "GET", fmt.Sprintf("/api/v1/workbench/skills/personal/%s/versions/1", personal.ID), "reviewer", "", "", "", nil, 404)
	request := map[string]any{"expectedModeRevision": modeRevision, "targetPlatform": "product", "contentSelection": map[string]any{"targetLanguage": "en-US", "title": true, "description": false, "sellingPoints": false, "seoKeywords": false}, "skillSelection": map[string]any{"skillId": ref.ID, "version": ref.Version, "digest": ref.Digest}}
	path := productAcquisitionBase + "/" + operationID + "/content-agent/runs"
	skillHTTP(t, server, "POST", path, "admin", "B", uuid.NewString(), "", request, 403)
	require.Zero(t, calls.Load(), "default personal denial must occur before any model dispatch")
	_, e := cfg.SetSkillAdmission(context.Background(), agentconfig.SkillAdmissionCommand{Scope: agent.Scope{OrganizationID: "B", ActorID: "admin"}, Key: uuid.NewString(), AllowPersonal: true}, func(context.Context) error { return nil })
	require.NoError(t, e)
	key := uuid.NewString()
	raw = skillHTTP(t, server, "POST", path, "admin", "B", key, "", request, 200)
	var result productAgentResultDTO
	require.NoError(t, json.Unmarshal(raw, &result))
	require.Equal(t, ref, result.SkillVersionRef)
	require.Len(t, result.Candidate.Changes, 1)
	require.Equal(t, agent.HumanReviewRequired, result.Phase)
	before := calls.Load()
	skillHTTP(t, server, "POST", path, "admin", "B", key, "", request, 200)
	require.Equal(t, before, calls.Load(), "exact replay must not resend")
	altered := map[string]any{}
	for k, v := range request {
		altered[k] = v
	}
	altered["skillSelection"] = map[string]any{"skillId": ref.ID, "version": ref.Version, "digest": strings.Repeat("0", 64)}
	skillHTTP(t, server, "POST", path, "admin", "B", key, "", altered, 409)
	require.Equal(t, before, calls.Load())
	raw = skillHTTP(t, server, "POST", path+"/"+key+"/review", "admin", "B", "", "", map[string]any{}, 200)
	var link struct {
		ProposalID string `json:"proposalId"`
	}
	require.NoError(t, json.Unmarshal(raw, &link))
	view := titleCall(t, server, "GET", titleBasePath+"/"+link.ProposalID, "admin", "B", "", "", 200)
	require.Equal(t, ref, view.Execution.SkillVersionRef)
	require.Len(t, view.Content.After, 1)
	if iam != nil {
		if crossChange == "version revoked before apply" {
			accepted := titleCall(t, server, "POST", titleBasePath+"/"+view.ID+"/decisions", "reviewer", "B", uuid.NewString(), `{"action":"accept","expected_revision":1}`, 200)
			lifecycle("personal", personal, "revoke")
			titleCall(t, server, "POST", titleBasePath+"/"+view.ID+"/apply", "reviewer", "B", uuid.NewString(), fmt.Sprintf(`{"expected_revision":%d}`, accepted.Revision), 409)
			require.Equal(t, before, calls.Load())
			return
		}
		testContentCrossAdministratorReview(t, crossChange, iam, replaced, view, server, calls, f)
		return
	}
	if mode == "read_only" {
		titleDecision(t, server, view, "accept", "admin", "", 403)
		require.Equal(t, "pending", view.State)
		return
	}
	if result.PolicyAdoption != nil {
		require.Equal(t, "applied", view.State)
		require.Equal(t, ref, view.PolicyAdoption.Execution.SkillVersionRef)
		require.False(t, result.HumanReviewRequired)
	} else {
		require.Equal(t, "default", mode, "title-only auto approval preserves the existing programmatic gate")
		accepted := titleDecision(t, server, view, "accept", "admin", "", 200)
		view = titleApply(t, server, accepted, uuid.NewString(), 200)
		require.Equal(t, "applied", view.State)
	}
	reader, e := catalogstore.NewBoundedSnapshotReader(f.owner, 2<<20)
	require.NoError(t, e)
	snapshot, e := reader.GetCurrentSnapshot(context.Background(), catalog.SnapshotIdentity{TenantID: "B", ProductKey: result.ProductKey})
	require.NoError(t, e)
	require.EqualValues(t, 2, snapshot.Version)
	// The material owner can revoke new effects. Existing exact run/receipt
	// remains readable without another invocation or a version substitution.
	personal = lifecycle("personal", personal, "revoke")
	_ = personal
	skillHTTP(t, server, "POST", path, "admin", "B", key, "", request, 200)
	require.Equal(t, before, calls.Load())
	skillHTTP(t, server, "POST", path, "admin", "B", uuid.NewString(), "", request, 409)
	require.Equal(t, before, calls.Load())
}
