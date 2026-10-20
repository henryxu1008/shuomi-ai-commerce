package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	supplyhttp "task-processor/internal/app/supplychain/httpapi"
	"task-processor/internal/httproute"
	record "task-processor/internal/listing/record/target"
)

const materialsTestRecord = "bc6f7af0-dfa7-46a9-b49f-069608294d53"
const materialsTestKey = "36451a84-25c5-4e8d-a610-4d7cbceba82b"

type materialsHTTPFixture struct {
	calls   int
	err     error
	claimed bool
}

func (f *materialsHTTPFixture) prepare(context.Context, string, materialsPrepareInput) (agentconfig.Snapshot, error) {
	f.calls++
	return agentconfig.Snapshot{}, f.err
}
func (f *materialsHTTPFixture) start(context.Context, string, string, agent.ConfigurationSnapshotRef) (agent.Record, bool, error) {
	f.calls++
	return agent.Record{}, f.claimed, f.err
}
func (f *materialsHTTPFixture) readRun(context.Context, string, string) (agent.Record, agentconfig.Snapshot, error) {
	f.calls++
	return agent.Record{}, agentconfig.Snapshot{}, f.err
}
func (f *materialsHTTPFixture) resume(context.Context, string, string, agent.ConfigurationSnapshotRef, uint64, string) (agent.Record, bool, error) {
	f.calls++
	return agent.Record{}, true, f.err
}
func (f *materialsHTTPFixture) adopt(context.Context, string, string, agent.ConfigurationSnapshotRef, string, bool) (record.TargetReceipt, error) {
	f.calls++
	return record.TargetReceipt{}, f.err
}
func (f *materialsHTTPFixture) adoption(context.Context, string, string) (record.TargetReceipt, error) {
	f.calls++
	return record.TargetReceipt{}, f.err
}
func (f *materialsHTTPFixture) projectRun(context.Context, string, agent.Record, agentconfig.Snapshot) (materialsRunDTO, error) {
	return materialsRunDTO{}, f.err
}

func materialsHTTPHarness(f *materialsHTTPFixture, denied bool) *gin.Engine {
	r := gin.New()
	bind := func(ctx context.Context, _ string) (context.Context, error) {
		if denied {
			return ctx, agentconfig.ErrForbidden
		}
		return ctx, nil
	}
	for _, d := range listingMaterialsRoutes(f, bind) {
		r.Handle(d.Method, d.Path, d.Handler)
	}
	return r
}

func TestMaterialsHTTPRejectsCallerFactsAndKeepsOriginalKey(t *testing.T) {
	f := &materialsHTTPFixture{err: agentconfig.ErrArchived, claimed: true}
	r := materialsHTTPHarness(f, false)
	base := strings.Replace(listingMaterialsBase, ":record_id", materialsTestRecord, 1)
	ref := agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: "235141b0-4fae-43b1-9d2b-5a84cc71a6e9", Digest: strings.Repeat("a", 64)}
	raw, err := json.Marshal(map[string]any{"configurationSnapshotRef": ref})
	require.NoError(t, err)
	request := func(method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", materialsTestKey)
		r.ServeHTTP(w, req)
		return w
	}
	for _, body := range []string{
		`{"configurationSnapshotRef":` + string(mustMaterialsJSON(t, ref)) + `,"candidate":{"title":"inject"}}`,
		`{"configurationSnapshotRef":` + string(mustMaterialsJSON(t, ref)) + `,"organizationId":"foreign"}`,
		`{"configurationSnapshotRef":` + string(mustMaterialsJSON(t, ref)) + `,"configurationSnapshotRef":` + string(mustMaterialsJSON(t, ref)) + `}`,
	} {
		w := request("POST", base+"/runs", body)
		require.Equal(t, 400, w.Code, w.Body.String())
	}
	require.Zero(t, f.calls)
	w := request("POST", base+"/runs", string(raw))
	require.Equal(t, 409, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "AGENT_CONFLICT")
	require.Contains(t, w.Body.String(), materialsTestKey)
	require.Contains(t, w.Body.String(), "runMayExist")
	f.err = record.ErrTargetUnknown
	w = request("POST", base+"/runs/"+materialsTestKey+"/adopt", string(mustMaterialsJSON(t, map[string]any{"configurationSnapshotRef": ref, "candidateHash": strings.Repeat("a", 64), "humanConfirmed": true})))
	require.Equal(t, 409, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "OUTCOME_UNKNOWN")
	require.Contains(t, w.Body.String(), base+"/runs/"+materialsTestKey+"/adoption")
	before := f.calls
	for _, tc := range []struct{ method, path, body string }{{"GET", base + "/runs/" + materialsTestKey, "{}"}, {"GET", base + "/runs/" + materialsTestKey + "?candidate=x", ""}, {"POST", base + "/runs/" + materialsTestKey + "/resume", `{"revision":"01"}`}} {
		w = request(tc.method, tc.path, tc.body)
		require.Equal(t, 400, w.Code, w.Body.String())
	}
	require.Equal(t, before, f.calls)
	denied := materialsHTTPHarness(f, true)
	w = httptest.NewRecorder()
	denied.ServeHTTP(w, httptest.NewRequest("GET", base+"/runs/"+materialsTestKey, nil))
	require.Equal(t, http.StatusForbidden, w.Code)
	require.Equal(t, before, f.calls)
}

func mustMaterialsJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, e := json.Marshal(v)
	require.NoError(t, e)
	return b
}

func TestMaterialsRouteAdmissionRequiresNormalSupplyAndAgent(t *testing.T) {
	routes := listingMaterialsRoutes(nil, nil)
	require.Len(t, routes, 6)
	for _, d := range routes {
		require.NoError(t, validateListingMaterialsDescriptor(d))
		for _, mutate := range []func(*httproute.Descriptor){
			func(r *httproute.Descriptor) { r.AuthPolicy = httproute.AuthPolicyPublic },
			func(r *httproute.Descriptor) { r.Permission = "" },
			func(r *httproute.Descriptor) {
				r.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyCachedRead
			},
			func(r *httproute.Descriptor) { r.RequestTimeout = 0 },
			func(r *httproute.Descriptor) { r.Module = "supply-chain" },
		} {
			changed := d
			mutate(&changed)
			require.Error(t, validateListingMaterialsDescriptor(changed))
		}
	}
	for _, flags := range []currentApplicationOptionalRoutes{{ProductAgent: true}, {SupplyChain: true}, {ProductAgent: true, PrivateDraftTrial: true}} {
		admitted := materialsApplicationTestRoutes(flags)
		admitted = append(admitted, routes...)
		require.Error(t, validateCurrentApplicationRoutesInternal(admitted, false, false, false, false, false, false, false, flags))
	}
	flags := currentApplicationOptionalRoutes{ProductAgent: true, SupplyChain: true}
	require.NoError(t, validateCurrentApplicationRoutesInternal(materialsApplicationTestRoutes(flags), false, false, false, false, false, false, false, flags))
}

func materialsApplicationTestRoutes(flags currentApplicationOptionalRoutes) []httproute.Descriptor {
	routes := agentAndPointRoutes(flags.ProductAgent, false)
	if flags.SupplyChain {
		routes = append(routes, supplyhttp.SupplyRoutes(nil, nil)...)
	}
	if flags.PrivateDraftTrial {
		routes = append(routes, supplyhttp.PrivateDraftReadRoutes(nil, nil)...)
	}
	if flags.SupplyChain && flags.ProductAgent {
		routes = append(routes, listingMaterialsRoutes(nil, nil)...)
	}
	return routes
}
