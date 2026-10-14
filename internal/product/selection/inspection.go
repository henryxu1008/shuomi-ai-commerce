// Package selection consumes exact private Collection/Catalog facts for
// advisory comparisons. It owns no Product mutation, persistence or provider.
package selection

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"task-processor/internal/agent"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
)

const MaxInspectionBytes = 32 << 10
const MaxFacts = 32

type CollectionReader interface {
	Select(context.Context, collection.SelectionInput) (collection.AuthorizedSelection, error)
	ReadItem(context.Context, string) (collection.ItemDetail, error)
}
type Inspector struct{ Collections CollectionReader }
type Fact struct {
	ID    string          `json:"id"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value"`
}
type Product struct {
	Ref        agent.SelectionProductRef `json:"ref"`
	SourceKind string                    `json:"sourceKind"`
	Facts      []Fact                    `json:"facts"`
	Missing    []string                  `json:"missing"`
}
type Inspection struct {
	Scope    collection.Scope `json:"-"`
	Products []Product        `json:"products"`
}

func (i Inspector) Prepare(ctx context.Context, contextID string, choices []collection.SelectionInput) (agent.Binding, Inspection, error) {
	var empty agent.Binding
	if ctx == nil || ctx.Err() != nil || i.Collections == nil || len(choices) < 2 || len(choices) > agent.MaxSelectionProducts {
		return empty, Inspection{}, agent.ErrInvalid
	}
	choices = append([]collection.SelectionInput(nil), choices...)
	sort.Slice(choices, func(a, b int) bool { return choices[a].ItemID < choices[b].ItemID })
	binding := agent.Binding{ContextKind: agent.SelectionContextKind, ContextID: contextID, TargetPlatform: "product"}
	out := Inspection{Products: make([]Product, 0, len(choices))}
	for index, choice := range choices {
		if index > 0 && choice.ItemID == choices[index-1].ItemID {
			return empty, Inspection{}, agent.ErrInvalid
		}
		proof, err := i.Collections.Select(ctx, choice)
		if err != nil {
			return empty, Inspection{}, err
		}
		scope, err := proof.Scope(ctx)
		if err != nil {
			return empty, Inspection{}, err
		}
		if index == 0 {
			out.Scope = scope
		} else if scope != out.Scope {
			return empty, Inspection{}, collection.ErrForbidden
		}
		detail, err := proof.Read(ctx)
		if err != nil {
			return empty, Inspection{}, err
		}
		if detail.Item.ArchivedAt != nil {
			return empty, Inspection{}, collection.ErrNotFound
		}
		ref := agent.SelectionProductRef{ItemID: detail.Item.ID, ItemRevision: strconv.FormatInt(detail.Item.Revision, 10), ProductKey: detail.Item.Source.ProductKey, CatalogVersion: strconv.FormatUint(detail.Item.Source.Version, 10), PublicationID: detail.Item.Source.PublicationID}
		if !ref.Valid() {
			return empty, Inspection{}, agent.ErrInvalid
		}
		binding.Selection[index] = ref
		out.Products = append(out.Products, project(ref, detail))
	}
	if !binding.Valid() {
		return empty, Inspection{}, agent.ErrInvalid
	}
	if err := boundInspection(&out); err != nil {
		return empty, Inspection{}, agent.ErrUnavailable
	}
	return binding, out, nil
}

// Preserve every authorized identity while omitting whole fields, fairly across
// candidates. Never truncate a field into a different purported source fact.
func boundInspection(out *Inspection) error {
	for {
		wire, err := json.Marshal(out)
		if err != nil {
			return err
		}
		if len(wire) <= MaxInspectionBytes {
			return nil
		}
		removed := false
		for n := range out.Products {
			p := &out.Products[n]
			if len(p.Facts) == 0 {
				continue
			}
			drop, largest := len(p.Facts)-1, -1
			for index, fact := range p.Facts {
				if fact.Path == "title" && len(p.Facts) > 1 {
					continue
				}
				encoded, _ := json.Marshal(fact)
				if len(encoded) > largest {
					drop, largest = index, len(encoded)
				}
			}
			last := p.Facts[drop].Path
			prefix := ""
			if strings.HasSuffix(last, ".uncertainty") {
				prefix = strings.TrimSuffix(last, ".uncertainty")
			}
			kept := p.Facts[:0]
			for _, fact := range p.Facts {
				if fact.Path == last || prefix != "" && (fact.Path == prefix || strings.HasPrefix(fact.Path, prefix+".")) {
					addMissing(p, fact.Path)
				} else {
					kept = append(kept, fact)
				}
			}
			p.Facts = kept
			removed = true
			wire, err = json.Marshal(out)
			if err != nil {
				return err
			}
			if len(wire) <= MaxInspectionBytes {
				return nil
			}
		}
		if !removed {
			return agent.ErrUnavailable
		}
	}
}
func addMissing(p *Product, path string) {
	for _, missing := range p.Missing {
		if missing == path {
			return
		}
	}
	if len(p.Missing) < 32 {
		p.Missing = append(p.Missing, path)
		return
	}
	p.Missing[31] = "additional_facts_omitted"
}

func (i Inspector) Inspect(ctx context.Context, binding agent.Binding) (Inspection, error) {
	if binding.ContextKind != agent.SelectionContextKind || !binding.Valid() {
		return Inspection{}, agent.ErrInvalid
	}
	var choices []collection.SelectionInput
	for _, ref := range binding.Selection {
		if ref == (agent.SelectionProductRef{}) {
			break
		}
		revision, _ := strconv.ParseInt(ref.ItemRevision, 10, 64)
		version, _ := strconv.ParseUint(ref.CatalogVersion, 10, 64)
		choices = append(choices, collection.SelectionInput{ItemID: ref.ItemID, ExpectedRevision: revision, OriginalPublicationID: ref.PublicationID, OriginalVersion: version})
	}
	actual, out, err := i.Prepare(ctx, binding.ContextID, choices)
	if err != nil {
		return Inspection{}, err
	}
	if actual != binding {
		return Inspection{}, agent.ErrConflict
	}
	return out, nil
}

// Historical source identity remains exact. Grouping revisions can change,
// while current private read authorization still governs every retained item.
func (i Inspector) AuthorizeHistory(ctx context.Context, binding agent.Binding) (collection.Scope, error) {
	if !binding.Valid() || binding.ContextKind != agent.SelectionContextKind || i.Collections == nil {
		return collection.Scope{}, agent.ErrInvalid
	}
	var scope collection.Scope
	for n, ref := range binding.Selection {
		if ref == (agent.SelectionProductRef{}) {
			break
		}
		detail, e := i.Collections.ReadItem(ctx, ref.ItemID)
		if e != nil {
			return collection.Scope{}, e
		}
		if detail.Item.ArchivedAt != nil {
			return collection.Scope{}, collection.ErrNotFound
		}
		if detail.Item.ID != ref.ItemID || detail.Item.Source.ProductKey != ref.ProductKey {
			return collection.Scope{}, agent.ErrConflict
		}
		proof, e := i.Collections.Select(ctx, collection.SelectionInput{ItemID: ref.ItemID, ExpectedRevision: detail.Item.Revision, OriginalPublicationID: detail.Item.Source.PublicationID, OriginalVersion: detail.Item.Source.Version})
		if e != nil {
			return collection.Scope{}, e
		}
		current, e := proof.Scope(ctx)
		if e != nil {
			return collection.Scope{}, e
		}
		if n == 0 {
			scope = current
		} else if scope != current {
			return collection.Scope{}, collection.ErrForbidden
		}
	}
	return scope, nil
}

func factID(ref agent.SelectionProductRef, path string, value json.RawMessage) string {
	raw, _ := json.Marshal(struct {
		Ref   agent.SelectionProductRef
		Path  string
		Value json.RawMessage
	}{ref, path, value})
	h := sha256.Sum256(raw)
	return "fact:" + hex.EncodeToString(h[:])
}
func project(ref agent.SelectionProductRef, detail collection.ItemDetail) Product {
	p := Product{Ref: ref, SourceKind: detail.Item.Source.Kind, Facts: []Fact{}, Missing: []string{"sales", "market_demand", "shipping", "commission", "tax", "landed_cost"}}
	missing := func(path string) { addMissing(&p, path) }
	add := func(path string, value any, known bool) {
		if !known {
			missing(path)
			return
		}
		raw, err := json.Marshal(value)
		if err != nil || len(raw) > 512 || len(p.Facts) >= MaxFacts {
			missing(path)
			return
		}
		p.Facts = append(p.Facts, Fact{ID: factID(ref, path, raw), Path: path, Value: raw})
	}
	s := detail.Product
	add("title", s.Title, s.Title != "")
	add("brand", s.Brand, s.Brand != "")
	add("category_path", s.CategoryPath, len(s.CategoryPath) > 0)
	add("description", s.Description, s.Description != "")
	add("selling_points", s.SellingPoints, len(s.SellingPoints) > 0)
	add("specifications", s.Specifications, s.Specifications != nil)
	for n, a := range s.Attributes {
		if n >= 8 {
			missing("additional_attributes")
			break
		}
		add("attributes."+strconv.Itoa(n), struct{ Name, Value string }{a.Name, a.Value}, a.Name != "" && a.Value != "")
		if a.Trace.IsInferred || a.Trace.NeedsReview {
			add("attributes."+strconv.Itoa(n)+".uncertainty", true, true)
		}
	}
	if len(s.Variants) == 0 {
		missing("variants")
	}
	for n, v := range s.Variants {
		if n >= 6 {
			missing("additional_variants")
			break
		}
		prefix := "variants." + strconv.Itoa(n)
		add(prefix+".sku", v.SKU, v.SKU != "")
		add(prefix+".attributes", variantAttributes(v.Attributes), len(v.Attributes) > 0)
		knownPrice := v.Price != nil && v.Price.Currency != "" && v.Price.Amount >= 0 && !math.IsInf(v.Price.Amount, 0) && !math.IsNaN(v.Price.Amount)
		if knownPrice {
			add(prefix+".price", struct {
				Amount   float64 `json:"amount"`
				Currency string  `json:"currency"`
			}{v.Price.Amount, v.Price.Currency}, true)
		} else {
			missing(prefix + ".price")
		}
		add(prefix+".stock", v.Stock, v.Stock > 0)
		if v.Trace.IsInferred || v.Trace.NeedsReview {
			add(prefix+".uncertainty", true, true)
		}
	}
	if s.Review != nil {
		add("review", s.Review, true)
	}
	if len(s.Warnings) > 0 {
		add("warnings", s.Warnings, true)
	}
	return p
}
func variantAttributes(attrs []catalog.Attribute) [][2]string {
	result := make([][2]string, 0, len(attrs))
	for _, a := range attrs {
		result = append(result, [2]string{a.Name, a.Value})
	}
	return result
}
func boundedText(value string, max int, required bool) bool {
	return (!required || value != "") && len(value) <= max && utf8.ValidString(value) && strings.TrimSpace(value) == value && !strings.ContainsRune(value, '\x00')
}
