package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/authidentity"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/research"
	"task-processor/internal/product/selection"
)

type researchReader struct {
	rows   []research.AnalysisObservation
	denied bool
}

func (r *researchReader) InspectSelection(context.Context, collection.Scope, string, []research.AnalysisChoice) ([]research.AnalysisObservation, error) {
	if r.denied {
		return nil, research.ErrForbidden
	}
	return r.rows, nil
}
func (r *researchReader) AuthorizeAnalysisHistory(context.Context, collection.Scope, string, []research.AnalysisChoice) error {
	if r.denied {
		return research.ErrForbidden
	}
	return nil
}

type productReader struct{ out selection.Inspection }

func (p productReader) Prepare(_ context.Context, id string, _ []collection.SelectionInput) (agent.Binding, selection.Inspection, error) {
	b := agent.Binding{ContextKind: agent.SelectionContextKind, ContextID: id, TargetPlatform: "product"}
	for i, v := range p.out.Products {
		b.Selection[i] = v.Ref
	}
	return b, p.out, nil
}
func (p productReader) AuthorizeHistory(context.Context, agent.Binding) (collection.Scope, error) {
	return collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}, nil
}
func fixture(t *testing.T) (Inspector, context.Context, string, []research.AnalysisChoice, *researchReader) {
	t.Helper()
	r := &researchReader{}
	p := productReader{out: selection.Inspection{Scope: collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}}}
	choices := []research.AnalysisChoice{}
	project := uuid.NewString()
	for _, id := range []string{"22222222-2222-4222-8222-222222222222", "33333333-3333-4333-8333-333333333333"} {
		pub := uuid.NewString()
		sel := research.Selection{ItemID: id, Revision: 1, PublicationID: pub, Version: 1}
		ref := research.EvidenceRef{Selection: sel, Source: collection.Source{Kind: "amazon_data", ProductKey: id, PublicationID: pub, OperationID: pub, Version: 1}, JobID: uuid.NewString(), Checksum: strings.Repeat("a", 64)}
		choice := research.AnalysisChoice{CandidateID: uuid.NewString(), ExpectedRevision: 2, LinkID: uuid.NewString(), EvidenceDigest: ref.Digest(), Selection: sel}
		price := 19.99 + float64(len(choices))*5
		r.rows = append(r.rows, research.AnalysisObservation{Choice: choice, Observation: research.Observation{Ref: ref, Site: "us", ASIN: "B000123456", SourceURL: "https://private.example/should-not-disclose", CapturedAt: "2026-11-02T00:00:00Z", ParserVersion: "amazon-v1", Query: research.QuerySummary{Mode: "keyword", Keyword: "desk organizer", Limit: 5, Fields: []string{"title", "price"}}, Currency: "USD", Price: &price, PriceBasis: "displayed_listing_offer", Missing: []string{"rating"}}})
		choices = append(choices, choice)
		p.out.Products = append(p.out.Products, selection.Product{Ref: agent.SelectionProductRef{ItemID: id, ItemRevision: "1", ProductKey: id, CatalogVersion: "1", PublicationID: pub}, SourceKind: "amazon_data", Facts: []selection.Fact{{ID: "old-fact", Path: "title", Value: json.RawMessage(`"saved item"`)}}, Missing: []string{"market_demand", "landed_cost"}})
	}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "member", TokenExpiresAt: time.Now().Add(time.Minute)})
	return Inspector{Research: r, Products: p}, ctx, project, choices, r
}
func TestResearchDisclosurePinsOriginalMetadataAndMasksUnknownFields(t *testing.T) {
	i, ctx, project, choices, r := fixture(t)
	b, out, err := i.Prepare(ctx, project, choices)
	require.NoError(t, err)
	require.True(t, b.Valid())
	raw, err := json.Marshal(out)
	require.NoError(t, err)
	require.LessOrEqual(t, len(raw), MaxInspectionBytes)
	require.NotContains(t, string(raw), "private.example")
	require.Contains(t, string(raw), "desk organizer")
	require.Contains(t, string(raw), `"path":"observation.query.keyword"`)
	require.Contains(t, string(raw), `"path":"observation.price"`)
	require.Contains(t, string(raw), "market_demand")
	require.Equal(t, 2, out.Summary.Selected)
	require.Len(t, out.Summary.Prices, 1)
	require.Equal(t, 19.99, out.Summary.Prices[0].Minimum)
	require.Equal(t, 24.99, out.Summary.Prices[0].Maximum)
	_, err = i.Inspect(ctx, b)
	require.NoError(t, err)
	r.denied = true
	_, err = i.Inspect(ctx, b)
	require.Error(t, err)
	_, err = i.AuthorizeHistory(ctx, b)
	require.Error(t, err)
}
func TestResearchProposalCannotBorrowEvidenceOrInventStructuredMarketMetrics(t *testing.T) {
	i, ctx, project, choices, _ := fixture(t)
	_, out, err := i.Prepare(ctx, project, choices)
	require.NoError(t, err)
	p := Proposal{Overview: "继续核实缺项", OverviewEvidenceIDs: []string{out.Candidates[0].Facts[0].ID}, Assessments: []Assessment{{CandidateID: out.Candidates[0].Research.CandidateID, Recommendation: "RESEARCH_MORE", Advice: "补充资料后再测试", EvidenceIDs: []string{out.Candidates[0].Facts[0].ID}}}}
	require.NoError(t, ValidateProposal(out, p))
	p.Assessments[0].EvidenceIDs = []string{out.Candidates[1].Facts[0].ID}
	require.Error(t, ValidateProposal(out, p))
	_, err = ParseProposal(json.RawMessage(`{"overview":"profit","sales":100}`))
	require.Error(t, err)
}
func TestSampleSummaryKeepsCurrenciesAndUnknownPriceBasisSeparate(t *testing.T) {
	i, ctx, project, choices, r := fixture(t)
	r.rows[1].Observation.Currency = "EUR"
	_, out, err := i.Prepare(ctx, project, choices)
	require.NoError(t, err)
	require.Len(t, out.Summary.Prices, 2)
	r.rows[1].Observation.PriceBasis = ""
	_, out, err = i.Prepare(ctx, project, choices)
	require.NoError(t, err)
	require.Len(t, out.Summary.Prices, 1)
	require.Equal(t, 1, out.Summary.Unpriced)
	r.rows[1].Observation.PriceBasis = "displayed_listing_offer"
	r.rows[1].Observation.Currency = "ZZZ"
	_, out, err = i.Prepare(ctx, project, choices)
	require.NoError(t, err)
	require.Equal(t, 1, out.Summary.Unpriced)
	require.Len(t, out.Summary.Prices, 1)
}

func TestResearchDisclosureBoundsFiveRichCandidatesBeforeModelUse(t *testing.T) {
	i, ctx, project, choices, r := fixture(t)
	p := i.Products.(productReader)
	for len(r.rows) < 5 {
		n := len(r.rows)
		row := r.rows[0]
		row.Choice.CandidateID = uuid.NewString()
		row.Choice.LinkID = uuid.NewString()
		row.Choice.Selection.ItemID = fmt.Sprintf("%08d-5555-4555-8555-555555555555", n+4)
		row.Observation.Ref.Selection = row.Choice.Selection
		row.Observation.Ref.Source.ProductKey = row.Choice.Selection.ItemID
		row.Choice.EvidenceDigest = row.Observation.Ref.Digest()
		r.rows = append(r.rows, row)
		choices = append(choices, row.Choice)
		product := p.out.Products[0]
		product.Ref.ItemID = row.Choice.Selection.ItemID
		product.Ref.ProductKey = row.Choice.Selection.ItemID
		p.out.Products = append(p.out.Products, product)
	}
	sort.Slice(p.out.Products, func(a, b int) bool { return p.out.Products[a].Ref.ItemID < p.out.Products[b].Ref.ItemID })
	for n := range p.out.Products {
		p.out.Products[n].Facts = nil
		for f := 0; f < 32; f++ {
			value, _ := json.Marshal(strings.Repeat("x", 500))
			p.out.Products[n].Facts = append(p.out.Products[n].Facts, selection.Fact{Path: fmt.Sprintf("detail.%02d", f), Value: value})
		}
	}
	i.Products = p
	b, out, e := i.Prepare(ctx, project, choices)
	require.NoError(t, e)
	wire, e := json.Marshal(out)
	require.NoError(t, e)
	require.LessOrEqual(t, len(wire), MaxInspectionBytes)
	for _, c := range out.Candidates {
		require.LessOrEqual(t, len(c.Facts), 32)
		require.NotEmpty(t, c.Missing)
		require.NotEmpty(t, c.Research.CandidateID)
	}
	_, e = ParseInspection(wire)
	require.NoError(t, e)
	require.True(t, InspectionMatchesBinding(out, b))
}
