package materials

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	contract "task-processor/internal/marketplace/validator"
	"task-processor/internal/product/collection"
)

func TestMaterialsDisclosureProvidesScopedPublicNativeDiagnostic(t *testing.T) {
	i := materialInspection(t)
	d, err := i.Disclose()
	require.NoError(t, err)
	raw, err := json.Marshal(d)
	require.NoError(t, err)
	var wire struct {
		Computed *contract.DiagnosticResult `json:"computedDiagnostic"`
	}
	require.NoError(t, json.Unmarshal(raw, &wire))
	require.NotNil(t, wire.Computed, "the material tool must compute the public native diagnostic, not just return saved issues")
	require.Equal(t, "shein.official_draft", wire.Computed.Scope)
	require.True(t, wire.Computed.DiagnosticOnly)
	require.Equal(t, contract.NotEvaluated, wire.Computed.Freshness.Status)
	require.NotEmpty(t, wire.Computed.OfflineChecks.Checks)
	require.NotEmpty(t, wire.Computed.OfflineChecks.Blockers)
	require.NoError(t, contract.ValidateBoundInput(wire.Computed.Input, ""))
	require.Equal(t, "v2.0.0", InspectToolRef().Version)
}

func TestMaterialsModelAdmissionAcceptsExactV2Disclosure(t *testing.T) {
	i := materialInspection(t)
	d, err := i.Disclose()
	require.NoError(t, err)
	raw, err := json.Marshal(d)
	require.NoError(t, err)
	projected, err := (ModelAdmission{}).ProjectEvidence(raw)
	require.NoError(t, err, "the fixed model must consume the actual v2 Tool output before proposing")
	require.JSONEq(t, string(raw), string(projected))
}

func TestMaterialsEvidenceExcludesOnlyReadTimesAndValidatesRecordedPayload(t *testing.T) {
	first := materialInspection(t)
	later, err := NewInspection(first.Base, first.Product, first.Rules, first.Inventory, time.Now().UTC().Add(time.Hour), time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)
	require.Equal(t, first.EvidenceID, later.EvidenceID)
	d1, err := first.Disclose()
	require.NoError(t, err)
	d2, err := later.Disclose()
	require.NoError(t, err)
	require.NotEqual(t, d1.ComputedDiagnostic.Input.ReadAt, d2.ComputedDiagnostic.Input.ReadAt)
	request := agent.Request{Binding: first.Binding, PolicyVersion: PolicyVersion, PromptVersion: PromptVersion}
	candidate := proposalFor(t, first, `"brandCode":"known-brand"`)
	v := Validator{Reader: inspectionFixture{later}}
	result, err := v.ValidateDomain(context.Background(), request, candidate, observedInspection(t, first))
	require.NoError(t, err)
	require.True(t, result.Valid)
	for _, mutate := range []func(*Disclosure){
		func(d *Disclosure) { d.ComputedDiagnostic.Input.ReadAt = time.Time{} },
		func(d *Disclosure) {
			d.ComputedDiagnostic.Input.EvaluatedAt = d.ComputedDiagnostic.Input.ReadAt.Add(-time.Second)
		},
		func(d *Disclosure) { d.ComputedDiagnostic.OfflineChecks.Checks[0].Message = "forged" },
		func(d *Disclosure) { d.Draft.Product.BrandCode = "forged" },
	} {
		d, err := first.Disclose()
		require.NoError(t, err)
		mutate(&d)
		h := observedInspection(t, first)
		h[0].Output, err = json.Marshal(d)
		require.NoError(t, err)
		_, err = v.ValidateDomain(context.Background(), request, candidate, h)
		require.Error(t, err)
	}
	// The Tool describes the base once. Distinct supported candidates keep that
	// evidence identity but have distinct runtime candidate hashes and are each
	// applied/revalidated by the native rules without another Tool/model call.
	other := proposalFor(t, first, `"categoryId":13`)
	require.NotEqual(t, collection.Digest(candidate), collection.Digest(other))
	p, err := ParseProposal(other.Payload)
	require.NoError(t, err)
	preview, err := Preview(later, p)
	require.NoError(t, err)
	require.True(t, preview.RequiresHuman)
	require.False(t, preview.Result.ReadyForUpload)
	require.Contains(t, preview.Result.Issues[len(preview.Result.Issues)-1].Message, "新类目")
	result, err = v.ValidateDomain(context.Background(), request, other, observedInspection(t, first))
	require.NoError(t, err)
	require.True(t, result.Valid)
}
