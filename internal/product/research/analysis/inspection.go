// Package analysis projects exact research evidence into bounded advisory
// inputs. It owns no execution, persistence, Product or human decision.
package analysis

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/currency"
	strict "sigs.k8s.io/json"
	"task-processor/internal/agent"
	"task-processor/internal/authidentity"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/research"
	"task-processor/internal/product/selection"
)

const MaxInspectionBytes = selection.MaxInspectionBytes
const InspectionVersion = "market-research-inspection-v1"

type ResearchReader interface {
	InspectSelection(context.Context, collection.Scope, string, []research.AnalysisChoice) ([]research.AnalysisObservation, error)
	AuthorizeAnalysisHistory(context.Context, collection.Scope, string, []research.AnalysisChoice) error
}
type ProductReader interface {
	Prepare(context.Context, string, []collection.SelectionInput) (agent.Binding, selection.Inspection, error)
	AuthorizeHistory(context.Context, agent.Binding) (collection.Scope, error)
}
type Inspector struct {
	Research ResearchReader
	Products ProductReader
}
type Metadata struct {
	Site          string                `json:"site"`
	ASIN          string                `json:"asin"`
	CapturedAt    string                `json:"capturedAt"`
	ParserVersion string                `json:"parserVersion"`
	Checksum      string                `json:"checksum"`
	Query         research.QuerySummary `json:"query"`
	Currency      string                `json:"currency,omitempty"`
	Price         *float64              `json:"price,omitempty"`
	PriceBasis    string                `json:"priceBasis,omitempty"`
}
type Candidate struct {
	Ref         agent.SelectionProductRef  `json:"ref"`
	Research    agent.ResearchCandidateRef `json:"research"`
	Observation Metadata                   `json:"observation"`
	Facts       []selection.Fact           `json:"facts"`
	Missing     []string                   `json:"missing"`
	Warnings    []string                   `json:"warnings"`
}
type PriceRange struct {
	Currency string  `json:"currency"`
	Basis    string  `json:"basis"`
	Count    int     `json:"count"`
	Minimum  float64 `json:"minimum"`
	Maximum  float64 `json:"maximum"`
}
type Summary struct {
	Selected int          `json:"selected"`
	Unpriced int          `json:"unpriced"`
	Prices   []PriceRange `json:"prices"`
}
type Inspection struct {
	Scope      collection.Scope `json:"-"`
	Version    string           `json:"version"`
	ProjectID  string           `json:"projectId"`
	Candidates []Candidate      `json:"candidates"`
	Summary    Summary          `json:"summary"`
}

func currentScope(ctx context.Context) (collection.Scope, error) {
	id, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	s := collection.Scope{OrganizationID: id.TenantID, ActorID: id.UserID, MemberID: id.EffectiveMemberID}
	if !ok || s.Validate() != nil {
		return s, research.ErrForbidden
	}
	return s, nil
}
func bounded(s string, n int) bool {
	return len(s) <= n && utf8.ValidString(s) && !strings.ContainsRune(s, 0)
}
func validMetadata(m Metadata) bool {
	if len(m.Site) < 1 || len(m.Site) > 4 || len(m.ASIN) != 10 || !bounded(m.CapturedAt, 64) || !bounded(m.ParserVersion, 128) || !bounded(m.Checksum, 256) || m.Checksum == "" || !bounded(m.Query.Mode, 32) || !bounded(m.Query.Keyword, 512) || !bounded(m.Query.CategoryNode, 512) || m.Query.Limit < 0 || m.Query.Limit > 200 || m.Query.ASINCount < 0 || m.Query.ASINCount > 200 || len(m.Query.Fields) > 32 || !bounded(m.Currency, 8) || !bounded(m.PriceBasis, 64) {
		return false
	}
	for _, f := range m.Query.Fields {
		if !agent.ValidID(f) {
			return false
		}
	}
	return m.Price == nil || (!math.IsNaN(*m.Price) && !math.IsInf(*m.Price, 0) && *m.Price >= 0)
}
func (i Inspector) Prepare(ctx context.Context, project string, choices []research.AnalysisChoice) (agent.Binding, Inspection, error) {
	var empty agent.Binding
	if i.Research == nil || i.Products == nil || len(choices) < 2 || len(choices) > agent.MaxSelectionProducts {
		return empty, Inspection{}, agent.ErrInvalid
	}
	scope, err := currentScope(ctx)
	if err != nil {
		return empty, Inspection{}, err
	}
	rows, err := i.Research.InspectSelection(ctx, scope, project, choices)
	if err != nil {
		return empty, Inspection{}, err
	}
	if len(rows) != len(choices) {
		return empty, Inspection{}, agent.ErrUnavailable
	}
	rows = append([]research.AnalysisObservation(nil), rows...)
	sort.Slice(rows, func(a, b int) bool { return rows[a].Choice.Selection.ItemID < rows[b].Choice.Selection.ItemID })
	inputs := make([]collection.SelectionInput, 0, len(rows))
	out := Inspection{Scope: scope, Version: InspectionVersion, ProjectID: project, Candidates: []Candidate{}}
	for _, row := range rows {
		c, o := row.Choice, row.Observation
		meta := Metadata{Site: o.Site, ASIN: o.ASIN, CapturedAt: o.CapturedAt, ParserVersion: o.ParserVersion, Checksum: o.Ref.Checksum, Query: o.Query, Currency: o.Currency, Price: o.Price, PriceBasis: o.PriceBasis}
		if !validMetadata(meta) || c.Selection != o.Ref.Selection || c.EvidenceDigest != o.Ref.Digest() {
			return empty, Inspection{}, agent.ErrInvalid
		}
		inputs = append(inputs, collection.SelectionInput{ItemID: c.Selection.ItemID, ExpectedRevision: c.Selection.Revision, OriginalPublicationID: c.Selection.PublicationID, OriginalVersion: c.Selection.Version})
		out.Candidates = append(out.Candidates, Candidate{Research: agent.ResearchCandidateRef{CandidateID: c.CandidateID, CandidateRevision: strconv.FormatInt(c.ExpectedRevision, 10), SelectedLinkID: c.LinkID, EvidenceDigest: c.EvidenceDigest}, Observation: meta, Warnings: []string{"SAVED_OBSERVATIONS_ONLY"}})
	}
	b, products, err := i.Products.Prepare(ctx, project, inputs)
	if err != nil {
		return empty, Inspection{}, err
	}
	if products.Scope != scope || len(products.Products) != len(rows) {
		return empty, Inspection{}, research.ErrForbidden
	}
	for n, p := range products.Products {
		for _, missing := range rows[n].Observation.Missing {
			if agent.ValidID("observation." + missing) {
				products.Products[n].Missing = appendMissing(products.Products[n].Missing, "observation."+missing)
			}
		}
		if p.Ref.ItemID != rows[n].Choice.Selection.ItemID {
			return empty, Inspection{}, agent.ErrConflict
		}
	}
	attachProducts(&out, products)
	out.Summary = summarize(out.Candidates)
	budget := MaxInspectionBytes
	for {
		wire, e := json.Marshal(out)
		if e != nil {
			return empty, Inspection{}, agent.ErrInvalid
		}
		if len(wire) <= MaxInspectionBytes {
			break
		}
		baseline, _ := json.Marshal(products)
		budget = min(MaxInspectionBytes, MaxInspectionBytes-(len(wire)-len(baseline)))
		if err = selection.BoundInspectionBudget(&products, budget); err != nil {
			return empty, Inspection{}, err
		}
		attachProducts(&out, products)
	}
	b.ContextKind = agent.ResearchSelectionContextKind
	for n, c := range out.Candidates {
		b.ResearchSelection.Candidates[n] = c.Research
	}
	b.ResearchSelection.InspectionDigest = collection.Digest(out)
	if !InspectionMatchesBinding(out, b) {
		return empty, Inspection{}, agent.ErrInvalid
	}
	return b, out, nil
}
func attachProducts(out *Inspection, products selection.Inspection) {
	for n, p := range products.Products {
		c := &out.Candidates[n]
		c.Ref = p.Ref
		c.Missing = append([]string{}, p.Missing...)
		c.Facts = make([]selection.Fact, 0, selection.MaxFacts)
		values := observationValues(c.Observation)
		paths := make([]string, 0, len(values))
		for path := range values {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			raw, e := json.Marshal(values[path])
			if e != nil || len(raw) > 512 {
				c.Missing = appendMissing(c.Missing, path)
				continue
			}
			c.Facts = append(c.Facts, selection.Fact{ID: factID(c.Research, c.Ref, path, raw), Path: path, Value: raw})
		}
		blocked := sourceMetadataPaths(p.Facts)
		for _, f := range p.Facts {
			if blocked[f.Path] {
				c.Missing = appendMissing(c.Missing, f.Path)
				c.Warnings = appendMissing(c.Warnings, "source_metadata_omitted")
				continue
			}
			if len(c.Facts) >= selection.MaxFacts {
				c.Missing = appendMissing(c.Missing, f.Path)
				continue
			}
			f.ID = factID(c.Research, c.Ref, f.Path, f.Value)
			c.Facts = append(c.Facts, f)
		}
	}
}

// Selection's broader Catalog attribute view also carries producer-injected
// source fields. Research discloses only its qualified observation metadata;
// never reinterpret those attributes as product facts or copy provider URLs.
func sourceMetadataPaths(facts []selection.Fact) map[string]bool {
	blocked := map[string]bool{}
	metadata := func(name, value string) bool {
		normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "", ".", "").Replace(name))
		switch normalized {
		case "sourceurl", "availability", "capturedat", "sourcemetadata", "providermetadata", "rawhtml", "rawpayload", "providerresponse":
			return true
		}
		return strings.Contains(value, "://") || strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "data:")
	}
	for _, f := range facts {
		if strings.HasPrefix(f.Path, "attributes.") && !strings.HasSuffix(f.Path, ".uncertainty") {
			var a struct{ Name, Value string }
			if json.Unmarshal(f.Value, &a) == nil && metadata(a.Name, a.Value) {
				blocked[f.Path] = true
				blocked[f.Path+".uncertainty"] = true
			}
		}
		if strings.HasPrefix(f.Path, "variants.") && strings.HasSuffix(f.Path, ".attributes") {
			var attrs [][2]string
			if json.Unmarshal(f.Value, &attrs) == nil {
				for _, a := range attrs {
					if metadata(a[0], a[1]) {
						blocked[f.Path] = true
						blocked[strings.TrimSuffix(f.Path, ".attributes")+".uncertainty"] = true
						break
					}
				}
			}
		}
	}
	return blocked
}
func appendMissing(values []string, path string) []string {
	for _, s := range values {
		if s == path {
			return values
		}
	}
	if len(values) < 32 {
		return append(values, path)
	}
	values[31] = "omitted_fields"
	return values
}
func observationValues(m Metadata) map[string]any {
	out := map[string]any{"observation.site": m.Site, "observation.asin": m.ASIN, "observation.captured_at": m.CapturedAt, "observation.parser_version": m.ParserVersion, "observation.checksum": m.Checksum, "observation.query.mode": m.Query.Mode, "observation.query.asin_count": m.Query.ASINCount, "observation.query.limit": m.Query.Limit, "observation.query.fields": append([]string{}, m.Query.Fields...)}
	if m.Query.Keyword != "" {
		out["observation.query.keyword"] = m.Query.Keyword
	}
	if m.Query.CategoryNode != "" {
		out["observation.query.category_node"] = m.Query.CategoryNode
	}
	if m.Price != nil {
		out["observation.price"] = *m.Price
	}
	if m.Currency != "" {
		out["observation.currency"] = m.Currency
	}
	if m.PriceBasis != "" {
		out["observation.price_basis"] = m.PriceBasis
	}
	return out
}
func factID(r agent.ResearchCandidateRef, ref agent.SelectionProductRef, path string, value json.RawMessage) string {
	return "fact:" + collection.Digest(struct {
		Version  string
		Research agent.ResearchCandidateRef
		Ref      agent.SelectionProductRef
		Path     string
		Value    json.RawMessage
	}{InspectionVersion, r, ref, path, value})
}
func summarize(cs []Candidate) Summary {
	out := Summary{Selected: len(cs), Prices: []PriceRange{}}
	groups := map[string]PriceRange{}
	for _, c := range cs {
		m := c.Observation
		_, currencyError := currency.ParseISO(m.Currency)
		if m.Price == nil || currencyError != nil || len(m.Currency) != 3 || m.Currency != strings.ToUpper(m.Currency) || m.Currency == "XXX" || m.Currency == "XTS" || m.PriceBasis != "displayed_listing_offer" {
			out.Unpriced++
			continue
		}
		key := m.Currency + "|" + m.PriceBasis
		p, ok := groups[key]
		if !ok {
			p = PriceRange{Currency: m.Currency, Basis: m.PriceBasis, Minimum: *m.Price, Maximum: *m.Price}
		}
		p.Count++
		p.Minimum = math.Min(p.Minimum, *m.Price)
		p.Maximum = math.Max(p.Maximum, *m.Price)
		groups[key] = p
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		out.Prices = append(out.Prices, groups[key])
	}
	return out
}
func choicesFromBinding(b agent.Binding) []research.AnalysisChoice {
	choices := []research.AnalysisChoice{}
	for n, p := range b.Selection {
		if p == (agent.SelectionProductRef{}) {
			break
		}
		r := b.ResearchSelection.Candidates[n]
		revision, _ := strconv.ParseInt(r.CandidateRevision, 10, 64)
		itemRevision, _ := strconv.ParseInt(p.ItemRevision, 10, 64)
		version, _ := strconv.ParseUint(p.CatalogVersion, 10, 64)
		choices = append(choices, research.AnalysisChoice{CandidateID: r.CandidateID, ExpectedRevision: revision, LinkID: r.SelectedLinkID, EvidenceDigest: r.EvidenceDigest, Selection: research.Selection{ItemID: p.ItemID, Revision: itemRevision, PublicationID: p.PublicationID, Version: version}})
	}
	return choices
}
func (i Inspector) Inspect(ctx context.Context, b agent.Binding) (Inspection, error) {
	if !b.Valid() || b.ContextKind != agent.ResearchSelectionContextKind {
		return Inspection{}, agent.ErrInvalid
	}
	actual, out, err := i.Prepare(ctx, b.ContextID, choicesFromBinding(b))
	if err != nil {
		return Inspection{}, err
	}
	if actual != b {
		return Inspection{}, agent.ErrConflict
	}
	return out, nil
}
func (i Inspector) AuthorizeHistory(ctx context.Context, b agent.Binding) (collection.Scope, error) {
	if i.Research == nil || i.Products == nil || !b.Valid() || b.ContextKind != agent.ResearchSelectionContextKind {
		return collection.Scope{}, agent.ErrInvalid
	}
	scope, err := currentScope(ctx)
	if err != nil {
		return collection.Scope{}, err
	}
	if err = i.Research.AuthorizeAnalysisHistory(ctx, scope, b.ContextID, choicesFromBinding(b)); err != nil {
		return collection.Scope{}, err
	}
	plain := b
	plain.ContextKind = agent.SelectionContextKind
	plain.ResearchSelection = agent.ResearchSelectionRef{}
	authorized, err := i.Products.AuthorizeHistory(ctx, plain)
	if err == nil && authorized != scope {
		return collection.Scope{}, research.ErrForbidden
	}
	return scope, err
}
func InspectionMatchesBinding(out Inspection, b agent.Binding) bool {
	if !b.Valid() || b.ContextKind != agent.ResearchSelectionContextKind || out.Version != InspectionVersion || out.ProjectID != b.ContextID || len(out.Candidates) < 2 || len(out.Candidates) > 5 || collection.Digest(out) != b.ResearchSelection.InspectionDigest {
		return false
	}
	var refs [agent.MaxSelectionProducts]agent.SelectionProductRef
	var rs [agent.MaxSelectionProducts]agent.ResearchCandidateRef
	for n, c := range out.Candidates {
		refs[n] = c.Ref
		rs[n] = c.Research
	}
	return refs == b.Selection && rs == b.ResearchSelection.Candidates
}
func ParseInspection(raw json.RawMessage) (Inspection, error) {
	var out Inspection
	if len(raw) == 0 || len(raw) > MaxInspectionBytes {
		return out, agent.ErrInvalid
	}
	errs, err := strict.UnmarshalStrict(raw, &out, strict.DisallowDuplicateFields, strict.DisallowUnknownFields)
	if err != nil || len(errs) > 0 || len(out.Candidates) < 2 || len(out.Candidates) > 5 || out.Version != InspectionVersion {
		return Inspection{}, agent.ErrInvalid
	}
	b := agent.Binding{ContextKind: agent.ResearchSelectionContextKind, ContextID: out.ProjectID, TargetPlatform: "product"}
	for n, c := range out.Candidates {
		if !validMetadata(c.Observation) || len(c.Facts) > selection.MaxFacts || len(c.Missing) > 32 || len(c.Warnings) > 32 {
			return Inspection{}, agent.ErrInvalid
		}
		seen := map[string]bool{}
		observation := observationValues(c.Observation)
		blocked := sourceMetadataPaths(c.Facts)
		for _, f := range c.Facts {
			if blocked[f.Path] {
				return Inspection{}, agent.ErrInvalid
			}
			if strings.HasPrefix(f.Path, "observation.") {
				value, ok := observation[f.Path]
				raw, e := json.Marshal(value)
				if !ok || e != nil || string(raw) != string(f.Value) {
					return Inspection{}, agent.ErrInvalid
				}
			}
			if !agent.ValidID(f.Path) || seen[f.Path] || len(f.Value) == 0 || len(f.Value) > 512 || !json.Valid(f.Value) || f.ID != factID(c.Research, c.Ref, f.Path, f.Value) {
				return Inspection{}, agent.ErrInvalid
			}
			seen[f.Path] = true
		}
		for _, s := range append(append([]string{}, c.Missing...), c.Warnings...) {
			if !agent.ValidID(s) {
				return Inspection{}, agent.ErrInvalid
			}
		}
		b.Selection[n] = c.Ref
		b.ResearchSelection.Candidates[n] = c.Research
	}
	b.ResearchSelection.InspectionDigest = collection.Digest(out)
	if !b.Valid() || !reflect.DeepEqual(out.Summary, summarize(out.Candidates)) {
		return Inspection{}, agent.ErrInvalid
	}
	return out, nil
}
