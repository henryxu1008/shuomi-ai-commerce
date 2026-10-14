package shopdiagnosis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"task-processor/internal/agent"
	c "task-processor/internal/operationscockpit"
	"time"
	"unicode/utf8"
)

const MaxInspectionBytes = 32 << 10
const MaxFacts = 32

type Fact struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Value string `json:"value"`
}
type Inspection struct {
	Scope   agent.ExecutionIdentity `json:"-"`
	Ref     agent.StoreDiagnosisRef `json:"ref"`
	Facts   []Fact                  `json:"facts"`
	Missing []string                `json:"missing"`
}
type Reader interface {
	Read(context.Context, agent.ExecutionIdentity, string, c.Period) (NativeFacts, error)
	Authorize(context.Context, agent.ExecutionIdentity, string) error
}
type Inspector struct {
	Reader   Reader
	Identity func(context.Context) (agent.ExecutionIdentity, error)
}

func (i Inspector) Prepare(ctx context.Context, contextID, storeID string, period c.Period) (agent.Binding, Inspection, error) {
	if i.Reader == nil || i.Identity == nil || !c.UUID(contextID) || !c.UUID(storeID) || !period.Valid() {
		return agent.Binding{}, Inspection{}, agent.ErrInvalid
	}
	id, e := i.Identity(ctx)
	if e != nil {
		return agent.Binding{}, Inspection{}, e
	}
	native, e := i.Reader.Read(ctx, id, storeID, period)
	if e != nil {
		return agent.Binding{}, Inspection{}, e
	}
	out, e := project(native, id)
	if e != nil {
		return agent.Binding{}, Inspection{}, e
	}
	b := agent.Binding{ContextKind: agent.StoreDiagnosisContextKind, ContextID: contextID, TargetPlatform: "business", StoreDiagnosis: out.Ref}
	if !b.Valid() {
		return agent.Binding{}, Inspection{}, agent.ErrInvalid
	}
	return b, out, nil
}
func (i Inspector) Inspect(ctx context.Context, b agent.Binding) (Inspection, error) {
	if !b.Valid() || b.ContextKind != agent.StoreDiagnosisContextKind {
		return Inspection{}, agent.ErrInvalid
	}
	actual, out, e := i.Prepare(ctx, b.ContextID, b.StoreDiagnosis.StoreID, c.Period{Start: b.StoreDiagnosis.StartDate, End: b.StoreDiagnosis.EndDate})
	if e != nil {
		return Inspection{}, e
	}
	if actual != b {
		return Inspection{}, agent.ErrConflict
	}
	return out, nil
}
func (i Inspector) AuthorizeHistory(ctx context.Context, b agent.Binding) (agent.ExecutionIdentity, error) {
	if i.Reader == nil || i.Identity == nil || !b.Valid() || b.ContextKind != agent.StoreDiagnosisContextKind {
		return agent.ExecutionIdentity{}, agent.ErrInvalid
	}
	id, e := i.Identity(ctx)
	if e == nil {
		e = i.Reader.Authorize(ctx, id, b.StoreDiagnosis.StoreID)
	}
	return id, e
}
func digest(value any) string {
	raw, _ := json.Marshal(value)
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}
func factID(ref agent.StoreDiagnosisRef, path, value string) string {
	return "fact:" + digest(struct {
		Ref         agent.StoreDiagnosisRef
		Path, Value string
	}{ref, path, value})
}
func boundedText(s string, max int, required bool) bool {
	return (!required || s != "") && utf8.ValidString(s) && len(s) <= max && strings.TrimSpace(s) == s && !strings.ContainsRune(s, '\x00')
}
func normalize(a c.StoreAggregate) c.StoreAggregate {
	a.Records = append([]c.RecordEvidence{}, a.Records...)
	a.Excluded = append([]c.RecordEvidence{}, a.Excluded...)
	a.Gaps = append([]c.Period{}, a.Gaps...)
	for _, refs := range [][]c.RecordEvidence{a.Records, a.Excluded} {
		sort.Slice(refs, func(i, j int) bool {
			if refs[i].ID != refs[j].ID {
				return refs[i].ID < refs[j].ID
			}
			if refs[i].Revision != refs[j].Revision {
				return refs[i].Revision < refs[j].Revision
			}
			if refs[i].Period.Start != refs[j].Period.Start {
				return refs[i].Period.Start < refs[j].Period.Start
			}
			return refs[i].Period.End < refs[j].Period.End
		})
	}
	sort.Slice(a.Gaps, func(i, j int) bool {
		if a.Gaps[i].Start != a.Gaps[j].Start {
			return a.Gaps[i].Start < a.Gaps[j].Start
		}
		return a.Gaps[i].End < a.Gaps[j].End
	})
	return a
}
func project(n NativeFacts, id agent.ExecutionIdentity) (Inspection, error) {
	a := normalize(n.Current)
	previous := normalize(n.Previous)
	out := Inspection{Scope: id, Ref: agent.StoreDiagnosisRef{StoreID: n.Store.ID, StartDate: a.Period.Start, EndDate: a.Period.End}, Facts: []Fact{}, Missing: []string{}}
	add := func(path, value string) {
		if !boundedText(value, 512, true) || len(out.Facts) >= MaxFacts {
			if len(out.Missing) < 32 {
				out.Missing = append(out.Missing, path)
			}
			return
		}
		out.Facts = append(out.Facts, Fact{Path: path, Value: value})
	}
	j := func(v any) string { raw, _ := json.Marshal(v); return string(raw) }
	add("store.name", n.Store.Name)
	add("store.platform", n.Store.Platform)
	add("financial.period", a.Period.Start+"/"+a.Period.End)
	add("financial.unit", "CNY_fen")
	add("financial.source", "manual")
	add("observations.source_digest", digest(n.Observations))
	add("financial.complete", strconv.FormatBool(a.Complete))
	add("financial.record_count", strconv.Itoa(len(a.Records)))
	// Entire sorted original evidence remains in the identity even if the
	// display-sized reference field is omitted. No retrieval clocks enter it.
	add("financial.source_digest", digest(struct{ Current, Previous c.StoreAggregate }{a, previous}))
	add("financial.gaps", j(a.Gaps))
	if !a.Complete {
		out.Missing = append(out.Missing, "financial.complete_coverage")
	}
	for index, name := range []string{"products", "orders"} {
		saved := n.Observations[index]
		prefix := "observations." + name
		add(prefix+".state", saved.State)
		add(prefix+".complete", strconv.FormatBool(saved.Complete))
		compact := func(values []c.ObservationEvidence) any {
			type source struct {
				SyncID, Status         string
				ObservedAt, Start, End *time.Time
			}
			result := []source{}
			for _, v := range values {
				result = append(result, source{v.SyncID, v.Status, v.ObservedAt, v.Start, v.End})
			}
			return result
		}
		add(prefix+".source", j(compact(saved.Sources)))
		add(prefix+".latest", j(compact(saved.Latest)))
	}
	if len(a.Records) > 0 {
		t := a.Totals
		for _, v := range []struct {
			path  string
			value int64
		}{{"revenue", t.Revenue}, {"refunds", t.Refunds}, {"procurement", t.Procurement}, {"logistics", t.Logistics}, {"platform", t.Platform}, {"advertising", t.Advertising}, {"other", t.Other}, {"net_revenue", t.NetRevenue}, {"net_profit", t.NetProfit}} {
			add("financial."+v.path, strconv.FormatInt(v.value, 10))
		}
		if t.Margin != nil {
			add("financial.margin", j(t.Margin))
		}
	} else {
		out.Missing = append(out.Missing, "financial.amounts")
	}
	if growth := c.Growth(a, previous); growth != nil {
		add("financial.growth", j(growth))
	} else {
		out.Missing = append(out.Missing, "financial.growth")
	}
	rules, e := c.Rules(c.Snapshot{Period: a.Period, Stores: map[string]c.StoreAggregate{a.StoreID: a}})
	if e != nil {
		return Inspection{}, e
	}
	for index, r := range rules {
		add("financial.rule."+strconv.Itoa(index), j(struct{ ID, Kind, Level, Reason string }{r.ID, r.Kind, r.Level, r.Reason}))
	}
	for index, name := range []string{"products", "orders"} {
		saved := n.Observations[index]
		prefix := "observations." + name
		if saved.State == "available" || saved.State == "incomplete" || saved.State == "latest_incomplete" {
			add(prefix+".counts", j(saved.Counts))
		} else {
			out.Missing = append(out.Missing, prefix+".counts")
		}
	}
	add("financial.records", j(a.Records))
	add("financial.excluded", j(a.Excluded))
	// Budget prioritizes mandatory provenance and uncertainty before optional
	// financial reference lists, rules and counters. Typical maximum is <32KiB.
	out.Ref.EvidenceDigest = digest(struct {
		Ref     agent.StoreDiagnosisRef
		Facts   []Fact
		Missing []string
	}{out.Ref, out.Facts, out.Missing})
	for k := range out.Facts {
		out.Facts[k].ID = factID(out.Ref, out.Facts[k].Path, out.Facts[k].Value)
	}
	raw, e := json.Marshal(out)
	if e != nil || len(raw) > MaxInspectionBytes {
		return Inspection{}, agent.ErrUnavailable
	}
	return out, nil
}
