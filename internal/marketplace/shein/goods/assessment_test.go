package goods

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	model "task-processor/internal/marketplace/shein/model"
	sheinvalidator "task-processor/internal/marketplace/shein/validator"
	contract "task-processor/internal/marketplace/validator"
	"task-processor/internal/product/asset"
)

func officialRequest(t *testing.T) contract.BoundRequest[sheinvalidator.OfficialIdentity] {
	t.Helper()
	digest, err := sheinvalidator.OfficialContentDigest("actual-source")
	require.NoError(t, err)
	return contract.BoundRequest[sheinvalidator.OfficialIdentity]{Input: sheinvalidator.OfficialIdentity{ContextDigest: digest, ProductReferenceDigest: digest, MerchantDigest: digest}, Target: contract.Target{Marketplace: "shein", Site: "shein-us"}, Action: contract.Preview, RuleVersion: sheinvalidator.OfficialDraftRuleVersion, BindingVersion: sheinvalidator.OfficialDraftBindingVersion, ReadAt: time.Unix(100, 0).UTC(), EvaluatedAt: time.Unix(101, 0).UTC(), Freshness: contract.ExternalFreshness{Status: contract.NotEvaluated}}
}

func TestSalesMappingDiagnosticVersionRejectsPriorEvaluationBinding(t *testing.T) {
	require.Equal(t, "shein.official_draft.v2", sheinvalidator.OfficialDraftRuleVersion)
	assessment := AssessOfficial(OfficialDraftInput{}, OfficialRuleSnapshot{}, asset.ApprovedAssetInventory{}, nil, nil)
	current, err := assessment.Diagnostic(officialRequest(t))
	require.NoError(t, err)
	require.Equal(t, sheinvalidator.OfficialDraftRuleVersion, current.RuleVersion)
	prior := officialRequest(t)
	prior.RuleVersion = "shein.official_draft.v1"
	_, err = assessment.Diagnostic(prior)
	require.Error(t, err, "new execution cannot claim the former rule version")
}

func TestOfficialDiagnosticUsesActualNativeFindingsAndStableContent(t *testing.T) {
	draft, rules, inventory := OfficialDraftInput{}, OfficialRuleSnapshot{}, asset.ApprovedAssetInventory{}
	a := AssessOfficial(draft, rules, inventory, nil, nil)
	req := officialRequest(t)
	d, err := a.Diagnostic(req)
	require.NoError(t, err)
	require.Equal(t, BuildOfficial(draft, rules, inventory, nil, nil), a.Native())
	require.True(t, d.DiagnosticOnly)
	require.Equal(t, sheinvalidator.OfficialDraftScope, d.Scope)
	require.Equal(t, contract.NotEvaluated, d.Freshness.Status)
	require.Nil(t, d.Freshness.Evidence)
	require.Empty(t, d.Freshness.Coverage)
	require.Len(t, d.OfflineChecks.Checks, len(a.Native().Issues)+1)
	require.Empty(t, d.OfflineChecks.Warnings)
	for _, issue := range a.Native().Issues {
		require.Contains(t, d.OfflineChecks.Checks, contract.Check{Rule: "official_draft.issue", Code: issue.Code, Category: sheinvalidator.OfficialDraftScope, Status: contract.CheckBlocking, Paths: []string{issue.Field}, Message: issue.Message})
	}
	req.ReadAt, req.EvaluatedAt = req.ReadAt.Add(time.Hour), req.EvaluatedAt.Add(time.Hour)
	later, err := a.Diagnostic(req)
	require.NoError(t, err)
	require.Equal(t, d.Input.Digest, later.Input.Digest)
	require.NotEqual(t, d.Input.ReadAt, later.Input.ReadAt)
	// Actual fact/observation changes, unlike timestamps, change the subject.
	draft.Product.BrandCode = "different"
	changed, err := AssessOfficial(draft, rules, inventory, nil, nil).Diagnostic(req)
	require.NoError(t, err)
	require.NotEqual(t, d.Input.Digest, changed.Input.Digest)
	changed, err = AssessOfficial(OfficialDraftInput{}, rules, inventory, []OfficialImageObservation{{RemoteURL: "https://fixture.invalid/changed"}}, nil).Diagnostic(req)
	require.NoError(t, err)
	require.NotEqual(t, d.Input.Digest, changed.Input.Digest)
}

func TestOfficialAssessmentDiagnosticDoesNotAliasNativeConsumerFindings(t *testing.T) {
	a := AssessOfficial(OfficialDraftInput{}, OfficialRuleSnapshot{}, asset.ApprovedAssetInventory{}, nil, nil)
	before, err := a.Diagnostic(officialRequest(t))
	require.NoError(t, err)
	native := a.Native()
	native.Issues[0].Message = "consumer annotation"
	after, err := a.Diagnostic(officialRequest(t))
	require.NoError(t, err)
	require.Equal(t, before, after, "a consumer cannot change the diagnostic of already assessed input")
}

func TestOfficialDiagnosticRejectsOversizedProjectionWithoutChangingNativeResult(t *testing.T) {
	a := AssessOfficial(OfficialDraftInput{}, OfficialRuleSnapshot{}, asset.ApprovedAssetInventory{}, nil, nil)
	a.result = OfficialDraft{Issues: []OfficialIssue{{Field: "product", Code: "fixture", Message: strings.Repeat("x", sheinvalidator.MaxDiagnosticBytes)}}}
	d, err := a.Diagnostic(officialRequest(t))
	require.Error(t, err)
	require.Equal(t, contract.DiagnosticResult{}, d)
	require.Len(t, a.Native().Issues[0].Message, sheinvalidator.MaxDiagnosticBytes)
}

func TestOfficialDiagnosticBindsCallerLoadedOwnerFacts(t *testing.T) {
	a := AssessOfficial(OfficialDraftInput{}, OfficialRuleSnapshot{}, asset.ApprovedAssetInventory{}, nil, nil)
	r := officialRequest(t)
	before, err := a.Diagnostic(r)
	require.NoError(t, err)
	changed, err := sheinvalidator.OfficialContentDigest("another actual owner fact")
	require.NoError(t, err)
	for _, mutate := range []func(*sheinvalidator.OfficialIdentity){
		func(i *sheinvalidator.OfficialIdentity) { i.ContextDigest = changed },
		func(i *sheinvalidator.OfficialIdentity) { i.ProductReferenceDigest = changed },
		func(i *sheinvalidator.OfficialIdentity) { i.MerchantDigest = changed },
	} {
		r := officialRequest(t)
		mutate(&r.Input)
		after, err := a.Diagnostic(r)
		require.NoError(t, err)
		require.NotEqual(t, before.Input.Digest, after.Input.Digest)
		r.ExpectedDigest = before.Input.Digest
		d, err := a.Diagnostic(r)
		require.Error(t, err)
		require.Equal(t, contract.DiagnosticResult{}, d)
	}
}

func TestOfficialDiagnosticProjectionPreservesDuplicatesAndNeverInventsPerRulePass(t *testing.T) {
	a := AssessOfficial(OfficialDraftInput{}, OfficialRuleSnapshot{}, asset.ApprovedAssetInventory{}, nil, nil)
	a.result = OfficialDraft{ReadyForUpload: true}
	d, err := a.Diagnostic(officialRequest(t))
	require.NoError(t, err)
	require.Equal(t, contract.Ready, d.OfflineChecks.Status)
	require.Len(t, d.OfflineChecks.Checks, 1, "only the aggregate build check has an actual success source")
	require.Empty(t, d.OfflineChecks.Blockers)
	issue := OfficialIssue{Field: "product.skus[0]", Code: "required", Message: "original finding"}
	a.result = OfficialDraft{Issues: []OfficialIssue{issue, issue}}
	req := officialRequest(t)
	req.Action = contract.SaveDraft
	d, err = a.Diagnostic(req)
	require.NoError(t, err)
	require.Len(t, d.OfflineChecks.Blockers, 3)
	require.True(t, d.ActionPolicy.ReadinessBlockersAllowed)
	req.Action = contract.Publish
	d, err = a.Diagnostic(req)
	require.NoError(t, err)
	require.False(t, d.ActionPolicy.ReadinessBlockersAllowed)
	require.Equal(t, contract.Blocked, d.OfflineChecks.Status)
}

func TestOfficialDiagnosticRejectsInvalidMetadataAndReturnsZero(t *testing.T) {
	a := AssessOfficial(OfficialDraftInput{}, OfficialRuleSnapshot{}, asset.ApprovedAssetInventory{}, nil, nil)
	for _, mutate := range []func(*contract.BoundRequest[sheinvalidator.OfficialIdentity]){
		func(r *contract.BoundRequest[sheinvalidator.OfficialIdentity]) { r.Target.Site = "unknown" },
		func(r *contract.BoundRequest[sheinvalidator.OfficialIdentity]) { r.Action = "unknown" },
		func(r *contract.BoundRequest[sheinvalidator.OfficialIdentity]) {
			r.RuleVersion = sheinvalidator.DiagnosticRuleVersion
		},
		func(r *contract.BoundRequest[sheinvalidator.OfficialIdentity]) {
			r.BindingVersion = sheinvalidator.BindingVersion
		},
		func(r *contract.BoundRequest[sheinvalidator.OfficialIdentity]) { r.ReadAt = time.Time{} },
		func(r *contract.BoundRequest[sheinvalidator.OfficialIdentity]) {
			r.EvaluatedAt = r.ReadAt.Add(-time.Second)
		},
		func(r *contract.BoundRequest[sheinvalidator.OfficialIdentity]) {
			r.ExpectedDigest = "sha256:" + strings.Repeat("a", 64)
		},
		func(r *contract.BoundRequest[sheinvalidator.OfficialIdentity]) {
			r.Freshness.Status = contract.FreshnessValid
		},
		func(r *contract.BoundRequest[sheinvalidator.OfficialIdentity]) { r.Input.ContextDigest = "unverified" },
	} {
		r := officialRequest(t)
		mutate(&r)
		d, err := a.Diagnostic(r)
		require.Error(t, err)
		require.IsType(t, &contract.Error{}, err)
		require.Equal(t, contract.DiagnosticResult{}, d)
	}
	nan := math.NaN()
	draft := OfficialDraftInput{Product: model.PublishProduct{SKCs: []model.ProductSKC{{SKUs: []model.ProductSKU{{Weight: &nan}}}}}}
	bad := AssessOfficial(draft, OfficialRuleSnapshot{}, asset.ApprovedAssetInventory{}, nil, nil)
	d, err := bad.Diagnostic(officialRequest(t))
	require.Error(t, err)
	require.Equal(t, contract.DiagnosticResult{}, d)
	require.NotEmpty(t, bad.Native().Issues, "native business result remains available without diagnostic admission")
	a.result.ReadyForUpload = true
	d, err = a.Diagnostic(officialRequest(t))
	require.Error(t, err)
	require.Equal(t, contract.DiagnosticResult{}, d)
}
