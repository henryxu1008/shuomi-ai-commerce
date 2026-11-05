package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	"task-processor/internal/aicapability"
	governed "task-processor/internal/integration/aicapability/einomodel"
	resourceadapter "task-processor/internal/integration/orgresource"
	"task-processor/internal/listing/readiness/tools/readinessinspect"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
	"task-processor/internal/product/sourcing"
	"task-processor/internal/product/sourcing/tools/sourceevidenceinspect"
	"task-processor/internal/workbenchcontext"
)

func TestIssue132TitlePilotThroughOwners(t *testing.T) {
	dsn := os.Getenv("ISSUE132_TEST_DSN")
	if dsn == "" {
		t.Skip("requires a task-owned ISSUE132_TEST_DSN")
	}
	require.Contains(t, dsn, "dbname=issue132_rehearsal")
	t.Setenv("ISSUE398_TEST_DSN", dsn)
	batch := uuid.MustParse("13200000-0000-4000-8000-000000000001")
	if id := os.Getenv("ISSUE132_TEST_BATCH_ID"); id != "" {
		var err error
		batch, err = uuid.Parse(id)
		require.NoError(t, err)
	}
	testProductAgentOwnersWithTitleRehearsal(t, "", &titleRehearsal{pilot: true, batchID: batch})
}

func configureTitlePilotFixture(t *testing.T, settings *ProductAgentDependencies, rehearsal *titleRehearsal) {
	t.Helper()
	settings.TextTransport = acquisitionHTTPTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.Scheme != "https" || req.URL.Host != "api.openai.com" || req.URL.Path != "/v1/chat/completions" {
			return nil, errors.New("unexpected title pilot destination")
		}
		w := httptest.NewRecorder()
		rehearsal.reply(w, req)
		return w.Result(), nil
	})
	p := settings.TextPolicies["B"]
	p.AdmittedEndpointIdentityDigest = governed.EndpointIdentityDigest(governed.TitlePilotOpenAIEndpoint)
	p.Profile.ProviderID, p.Profile.ModelID = "openai", aicapability.SingleImageMiniModel
	p.Profile.AdapterPolicyVersion = governed.TitlePilotOpenAIAdapterPolicy
	p.Profile.UsageMappingVersion = "openai-text-v1"
	p.Profile.MaximumPromptTokens = governed.TitlePilotPromptTokens
	p.Profile.MaximumCompletionTokens = governed.TitlePilotCompletionTokens
	p.Profile.InputMicrosPerMillion, p.Profile.OutputMicrosPerMillion = 6_000_000, 36_000_000
	tariff := &p.Profile.OfficialTariff
	tariff.Model, tariff.Channel = p.Profile.ModelID, "openai-direct"
	tariff.Rates = aicapability.OfficialModelRates{InputText: 6_000_000, CachedInputText: 600_000, OutputText: 36_000_000}
	settings.TextPolicies["B"] = p
	digest, err := p.ShapeProfile().Digest()
	require.NoError(t, err)
	settings.titlePilotApproval = titlePilotApproval{
		batchID: rehearsal.batchID, identity: agent.ExecutionIdentity{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"},
		profileDigest: digest,
	}
}

func (r *titleRehearsal) runPilot(t *testing.T, db *gorm.DB, app *productAgentApplication) {
	resolver, ok := app.resolver.(*workbenchcontext.Resolver)
	require.True(t, ok)
	ctx, err := candidateReviewFixtureContext(context.Background(), titleVerifier{}, resolver, "admin", "B")
	require.NoError(t, err)
	batch := r.batchID
	_, err = app.prepareTitlePilot(ctx, uuid.New())
	require.ErrorIs(t, err, agent.ErrUnavailable, "a different batch ID must be rejected before any publication")
	selector := app.selectTitleProfile
	app.selectTitleProfile = func(ctx context.Context, org string) (aicapability.ModelProfile, error) {
		profile, err := selector(ctx, org)
		profile.OfficialTariff.FXVersion = "different-approval-version"
		return profile, err
	}
	_, err = app.prepareTitlePilot(ctx, batch)
	app.selectTitleProfile = selector
	require.ErrorIs(t, err, agent.ErrUnavailable, "the approved batch cannot choose a different tariff or route")
	plan, err := app.prepareTitlePilot(ctx, batch)
	require.NoError(t, err)
	require.Len(t, plan.slots, 60)
	var costMicros int64
	for _, slot := range plan.slots {
		costMicros += slot.limits.CostMicros
		require.Equal(t, time.Minute, slot.limits.Runtime)
	}
	require.EqualValues(t, 37_800_000, costMicros)
	r.script()
	overBudget := *plan
	overBudget.slots = slices.Clone(plan.slots)
	overBudget.slots[0].limits.CostMicros += titlePilotBatchCost - costMicros + 1
	_, err = app.runTitlePilot(ctx, &overBudget)
	require.ErrorIs(t, err, errTitlePilotStopped, "¥38 plus one micro must fail before any claim or send")
	require.Empty(t, r.captured(t))
	_, found, err := app.readTitlePilotRun(ctx, &plan.slots[0])
	require.NoError(t, err)
	require.False(t, found)
	underquoted := plan.profile
	underquoted.InputMicrosPerMillion = 1
	require.False(t, titlePilotProfileQualified(underquoted), "runtime price must cover frozen tariff")
	oversize := plan.profile
	oversize.OutputMicrosPerMillion = 100_000_000
	require.False(t, titlePilotProfileQualified(oversize), "maximum quote must fit the slot before publishing")
	require.False(t, governed.ValidRouteProfile(plan.profile, aicapability.OperationAIWorkbenchChatPlan))

	var publications, runs []string
	var emptyTitles int
	var actions []agent.Action
	for n := 0; n < len(plan.slots); n += 2 {
		fixed, dynamic := &plan.slots[n], &plan.slots[n+1]
		require.True(t, fixed.fixed)
		require.False(t, dynamic.fixed)
		require.Equal(t, fixed.binding, dynamic.binding)
		require.EqualValues(t, 180_000, fixed.limits.CostMicros)
		require.EqualValues(t, 1_080_000, dynamic.limits.CostMicros)
		require.NotEqual(t, fixed.key, dynamic.key)
		publications = append(publications, fixed.binding.PublicationID)
		runs = append(runs, fixed.key, dynamic.key)
		published, err := app.titlePilotPublicationForBinding(fixed.context(ctx), fixed.binding)
		require.NoError(t, err)
		require.Equal(t, "admin", published.Receipt.ActorID)
		require.Equal(t, "local-synthetic", published.Envelope.Identity.SourcePlatform)
		require.Equal(t, sourcing.SourceTypeManualImport, published.Envelope.Identity.SourceType)
		require.Empty(t, published.Envelope.RawReference.URL)
		require.Empty(t, published.Snapshot.Images)
		if published.Snapshot.Title == "" {
			emptyTitles++
		}
		title := strings.TrimSpace(strings.SplitN(published.Snapshot.Description, ".", 2)[0])
		candidate := rehearsalTitle(title, published.Envelope.RawReference.ReferenceID)
		actions = append(actions, candidate,
			agent.Action{Kind: "tool", Tool: canonicalinspect.Definition().Ref},
			agent.Action{Kind: "tool", Tool: sourceevidenceinspect.Definition().Ref})
		if n == 0 {
			invalid := rehearsalTitle(title, "unobserved-evidence")
			actions = append(actions, agent.Action{Kind: "tool", Tool: readinessinspect.Definition().Ref}, invalid, invalid)
		}
		actions = append(actions, candidate)
	}
	require.Len(t, uniqueTitlePilotValues(publications), 30)
	require.Len(t, uniqueTitlePilotValues(runs), 60)
	require.Equal(t, 8, emptyTitles)
	_, err = app.Authorize(ctx, plan.slots[0].binding)
	require.Error(t, err, "normal title authorizer cannot accept an evaluation binding")
	_, err = app.startRequest(ctx, plan.slots[0].binding, plan.slots[0].key, "", nil, "0")
	require.Error(t, err, "normal title preparation cannot create an evaluation run")
	identity, err := app.freshIdentity(ctx)
	require.NoError(t, err)
	_, err = app.currentAcquisitionBinding(ctx, identity, plan.slots[0].binding.ContextID, "shein")
	require.Error(t, err, "source publication is not a public acquisition receipt")
	first := &plan.slots[0]
	foreign := first.binding
	foreign.CatalogVersion = "2"
	_, err = app.Authorize(first.context(ctx), foreign)
	require.Error(t, err)
	_, err = app.Authorize(first.context(ctx), plan.slots[2].binding)
	require.Error(t, err)
	other, err := candidateReviewFixtureContext(ctx, titleVerifier{}, resolver, "admin", "A")
	require.NoError(t, err)
	_, err = app.Authorize(first.context(other), first.binding)
	require.Error(t, err)

	r.script(actions...)
	type outcome struct {
		records []agent.Record
		err     error
	}
	start := make(chan struct{})
	out := make(chan outcome, 2)
	for range 2 {
		go func() {
			<-start
			records, err := app.startApprovedTitlePilot(ctx)
			out <- outcome{records, err}
		}()
	}
	close(start)
	left, right := <-out, <-out
	if left.err != nil {
		left, right = right, left
	}
	require.NoError(t, left.err)
	require.ErrorIs(t, right.err, agent.ErrConflict)
	require.Empty(t, right.records)
	require.Len(t, left.records, 60)
	for i, record := range left.records {
		slot := &plan.slots[i]
		require.Equal(t, slot.binding, record.State.Request.Binding)
		require.Equal(t, slot.key, record.State.Request.Key)
		require.Equal(t, agent.HumanReviewRequired, record.State.Phase, "%s stopped at %s", slot.command.Envelope.Identity.SourceID, record.State.StopReason)
		require.NotNil(t, record.State.Validation)
		require.True(t, record.State.Validation.Valid)
		require.NoError(t, app.confirmTitlePilotCosts(slot.context(ctx), record))
		if slot.fixed {
			require.Equal(t, 1, record.State.Usage.ModelCalls)
		} else if i == 1 {
			require.Equal(t, 6, record.State.Usage.ModelCalls)
			require.Equal(t, 2, record.State.Repairs)
		} else {
			require.Equal(t, 3, record.State.Usage.ModelCalls)
		}
		for _, h := range record.State.History {
			if h.InvocationID != "" {
				require.Error(t, app.points.ConfirmModelInvocationSettlement(slot.context(ctx), "B", h.InvocationID, "different-run"))
			}
		}
	}
	calls := r.captured(t)
	require.Len(t, calls, 123)
	for i, call := range calls {
		require.JSONEq(t, `"gpt-5.4-mini-2026-03-17"`, string(call.options["model"]))
		require.JSONEq(t, "2048", string(call.options["max_completion_tokens"]))
		require.JSONEq(t, `"none"`, string(call.options["reasoning_effort"]))
		require.JSONEq(t, "false", string(call.options["store"]))
		require.JSONEq(t, `"default"`, string(call.options["service_tier"]))
		require.NotContains(t, call.options, "max_tokens")
		require.NotContains(t, string(call.evidence), ".test")
		if i == 6 {
			require.JSONEq(t, string(calls[0].evidence), string(call.evidence), "fixed and repaired Agent see the same facts")
		} else if i >= 7 && (i-7)%4 == 3 {
			require.JSONEq(t, string(calls[i-3].evidence), string(call.evidence), "both groups see the same canonical product facts")
		}
	}
	_, err = app.prepareTitlePilot(ctx, batch)
	require.ErrorIs(t, err, agent.ErrConflict)
	_, err = app.prepareTitlePilot(ctx, uuid.New())
	require.ErrorIs(t, err, agent.ErrUnavailable)

	// Each fault case simulates its own isolated admission. The batch ID of
	// the running application cannot be changed by its caller.
	prepareFaultCase := func(t *testing.T) *titlePilotPlan {
		t.Helper()
		prior := app.config.titlePilotApproval
		app.config.titlePilotApproval.batchID = uuid.New()
		t.Cleanup(func() { app.config.titlePilotApproval = prior })
		fault, err := app.prepareTitlePilot(ctx, app.config.titlePilotApproval.batchID)
		require.NoError(t, err)
		return fault
	}

	t.Run("slot budget refuses before sending", func(t *testing.T) {
		prior := app.config.Limits
		app.config.Limits.CostMicros = 1
		app.config.Limits.Runtime = 45 * time.Second
		t.Cleanup(func() { app.config.Limits = prior })
		plan := prepareFaultCase(t)
		require.Equal(t, 45*time.Second, plan.slots[0].limits.Runtime)
		r.script()
		completed, err := app.runTitlePilot(ctx, plan)
		require.ErrorIs(t, err, agent.ErrUnavailable, "the selected profile cannot fit the current slot budget")
		require.Empty(t, completed)
		require.Empty(t, r.captured(t))
		_, found, err := app.readTitlePilotRun(ctx, &plan.slots[0])
		require.NoError(t, err)
		require.False(t, found)
	})

	t.Run("unknown usage halts all remaining slots", func(t *testing.T) {
		plan := prepareFaultCase(t)
		require.Equal(t, publications[0], plan.slots[0].binding.PublicationID)
		r.script(actions[:2]...)
		r.unknownAt = 2
		defer func() { r.unknownAt = 0 }()
		completed, err := app.runTitlePilot(ctx, plan)
		require.ErrorIs(t, err, errTitlePilotUnknown)
		require.Len(t, completed, 1)
		require.Len(t, r.captured(t), 2)
		slot := &plan.slots[1]
		stored := requireTitlePilotStopped(t, db, ctx, app, plan, 1)
		fact, err := app.config.Ledger.ReadModelInvocation(ctx, "B", stored.State.PendingInvocationID)
		require.NoError(t, err)
		require.Equal(t, aicapability.InvocationDispatched, fact.Outcome)
		require.Error(t, app.points.ConfirmModelInvocationSettlement(slot.context(ctx), "B", fact.InvocationID, stored.State.RunID))
		_, err = app.runTitlePilot(ctx, plan)
		require.ErrorIs(t, err, agent.ErrConflict)
		require.Len(t, r.captured(t), 2)
	})
	t.Run("terminal invocation without settlement halts the batch", func(t *testing.T) {
		plan := prepareFaultCase(t)
		r.script(actions[:1]...)
		app.config.Ledger.SetUsageSettler(titlePilotUnconfirmedSettler{app.points})
		defer app.config.Ledger.SetUsageSettler(app.points)
		completed, err := app.runTitlePilot(ctx, plan)
		require.ErrorIs(t, err, errTitlePilotUnknown)
		require.Empty(t, completed)
		require.Len(t, r.captured(t), 1)
		stored := requireTitlePilotStopped(t, db, ctx, app, plan, 0)
		fact, err := app.config.Ledger.ReadModelInvocation(ctx, "B", stored.State.PendingInvocationID)
		require.NoError(t, err)
		require.Equal(t, aicapability.InvocationSucceeded, fact.Outcome, "provider result and point settlement are different facts")
		require.Error(t, app.points.ConfirmModelInvocationSettlement(plan.slots[0].context(ctx), "B", fact.InvocationID, stored.State.RunID))
	})
	t.Run("run commit failure leaves an original receipt and stops", func(t *testing.T) {
		plan := prepareFaultCase(t)
		require.NoError(t, db.Exec(`CREATE FUNCTION issue132_reject_run_commit() RETURNS trigger AS $$
			BEGIN RAISE EXCEPTION 'synthetic run commit failure'; END; $$ LANGUAGE plpgsql`).Error)
		require.NoError(t, db.Exec(`CREATE TRIGGER issue132_reject_run_commit BEFORE UPDATE ON product_agent_runs
			FOR EACH ROW EXECUTE FUNCTION issue132_reject_run_commit()`).Error)
		defer func() {
			require.NoError(t, db.Exec("DROP TRIGGER issue132_reject_run_commit ON product_agent_runs").Error)
		}()
		r.script(actions[:1]...)
		completed, err := app.runTitlePilot(ctx, plan)
		require.ErrorIs(t, err, errTitlePilotUnknown)
		require.Empty(t, completed)
		require.Len(t, r.captured(t), 1)
		original, found, err := app.readTitlePilotRun(ctx, &plan.slots[0])
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, agent.Running, original.State.Phase)
		var invocationID string
		require.NoError(t, db.Table("ai_invocations").Select("invocation_id").Where("agent_run_id = ?", original.State.RunID).Take(&invocationID).Error)
		require.NoError(t, app.points.ConfirmModelInvocationSettlement(plan.slots[0].context(ctx), "B", invocationID, original.State.RunID))
		_, found, err = app.readTitlePilotRun(ctx, &plan.slots[1])
		require.NoError(t, err)
		require.False(t, found)
		_, err = app.runTitlePilot(ctx, plan)
		require.ErrorIs(t, err, agent.ErrConflict)
		require.Len(t, r.captured(t), 1)
	})
	var candidates int64
	require.NoError(t, db.Table("product_title_proposals").Count(&candidates).Error)
	require.Zero(t, candidates, "pilot stops at validation, without submitting review")
	t.Log("verified one approved batch, 30 SRC/Catalog receipts, 60 original run slots, ¥37.80 maximum, 123 synthetic sends and pre-send or unknown-cost stops")
}

func requireTitlePilotStopped(t *testing.T, db *gorm.DB, ctx context.Context, app *productAgentApplication, plan *titlePilotPlan, index int) agent.Record {
	t.Helper()
	scope := agent.Scope{OrganizationID: "B", ActorID: "admin"}
	halted := plan.slots[index]
	record, found, err := app.store.Lookup(ctx, scope, halted.binding, halted.key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, agent.StopModelUnknown, record.State.StopReason)
	require.NotEmpty(t, record.State.PendingInvocationID)
	var state string
	require.NoError(t, db.Table("saas_organization_resource_reservations").Select("state").
		Where("organization_id = ? AND owner_attempt_id = ?", "B", record.State.PendingInvocationID).Take(&state).Error)
	require.Equal(t, "reserved", state)
	for _, slot := range plan.slots[index+1:] {
		_, found, err := app.store.Lookup(ctx, scope, slot.binding, slot.key)
		require.NoError(t, err)
		require.False(t, found, "no subsequent slot may be claimed")
	}
	return record
}

func uniqueTitlePilotValues(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, v := range values {
		out[v] = struct{}{}
	}
	return out
}

type titlePilotUnconfirmedSettler struct {
	*resourceadapter.GormModelInvocationRepository
}

func (titlePilotUnconfirmedSettler) SettleAIInvocationUsage(context.Context, string, string, string, int64, time.Time) error {
	return errors.New("synthetic settlement failure after invocation terminal write")
}
