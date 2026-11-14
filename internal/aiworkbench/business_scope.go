package aiworkbench

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	sigjson "sigs.k8s.io/json"
	"task-processor/internal/agent"
)

const BusinessScopeVersion = "workbench-scope-v1"

var scopeIdentifier = regexp.MustCompile("^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$")

type ScopeSelection struct {
	Kind      string `json:"kind"`
	ProjectID string `json:"projectId,omitempty"`
	StoreID   string `json:"storeId,omitempty"`
}

func (s ScopeSelection) Valid() bool {
	switch s.Kind {
	case "general":
		return s.ProjectID == "" && s.StoreID == ""
	case "project":
		return scopeIdentifier.MatchString(s.ProjectID) && s.StoreID == ""
	case "store":
		return scopeIdentifier.MatchString(s.StoreID) && s.ProjectID == ""
	default:
		return false
	}
}

func (s ScopeSelection) ValidTaskFilter() bool {
	return s.Valid() || s == (ScopeSelection{Kind: "unrecorded"}) || s == (ScopeSelection{Kind: "all"})
}

type BusinessScope struct {
	Version   string `json:"version"`
	Kind      string `json:"kind"`
	ProjectID string `json:"projectId,omitempty"`
	StoreID   string `json:"storeId,omitempty"`
}

func (s BusinessScope) Valid() bool {
	if s.Version != BusinessScopeVersion {
		return false
	}
	if s.Kind == "project" {
		return scopeIdentifier.MatchString(s.ProjectID) && (s.StoreID == "" || scopeIdentifier.MatchString(s.StoreID))
	}
	return (ScopeSelection{Kind: s.Kind, ProjectID: s.ProjectID, StoreID: s.StoreID}).Valid()
}

type ScopeView struct {
	Scope       BusinessScope `json:"scope"`
	ProjectName string        `json:"projectName,omitempty"`
	StoreName   string        `json:"storeName,omitempty"`
}

type ProjectScopeFacts struct {
	ID, Title, Goal, StoreID string
	Revision                 uint64
	Archived                 bool
}

type StoreScopeFacts struct {
	ID, Name, Platform, Region string
	Version                    uint64
}

type StoreConnectionFacts struct {
	Version    uint64
	ObservedAt string
	Connected  bool
}

// ScopeReader uses the current owners' private Project and member Store reads.
type ScopeReader struct {
	Project    func(context.Context, Scope, string) (ProjectScopeFacts, error)
	Store      func(context.Context, Scope, string) (StoreScopeFacts, error)
	Connection func(context.Context, Scope, string) (StoreConnectionFacts, error)
}

type ScopeModelInput struct {
	ProjectTitle string `json:"project_title,omitempty"`
	ProjectGoal  string `json:"project_goal,omitempty"`
	StoreName    string `json:"store_name,omitempty"`
	Platform     string `json:"platform,omitempty"`
	Region       string `json:"region,omitempty"`
}

type FrozenScope struct {
	Owner           Scope
	ConversationID  string
	MemberID        string
	Scope           BusinessScope
	ProjectRevision uint64
	StoreVersion    uint64
	ModelInput      ScopeModelInput
	StoreChoice     *StoreChoiceScope `json:",omitempty"`
}

// EffectiveModelInput projects the current store without changing the frozen base.
func (f FrozenScope) EffectiveModelInput() ScopeModelInput {
	result := f.ModelInput
	if f.StoreChoice != nil {
		result.StoreName, result.Platform, result.Region = "", "", ""
		if selected := f.StoreChoice.Store; f.StoreChoice.Kind == "store" && selected != nil {
			result.StoreName, result.Platform, result.Region = selected.Name, selected.Platform, selected.Region
		}
	}
	return result
}

type StoreChoiceScope struct {
	Kind  string
	Store *SelectedStoreScope `json:",omitempty"`
}

func (s *StoreChoiceScope) Matches(input *StoreChoiceInput) bool {
	if s == nil || input == nil {
		return s == nil && input == nil
	}
	if !input.Valid() || s.Kind != input.Kind {
		return false
	}
	if s.Kind == "none" {
		return s.Store == nil
	}
	return s.Store != nil && s.Store.StoreID == input.StoreID && s.Store.ConnectionVersion == input.ConnectionVersion
}

// SelectedStoreScope is the exact owner-provided store observed for a single
// command. Scope above remains the immutable conversation origin.
type SelectedStoreScope struct {
	StoreID, Name, Platform, Region string
	StoreVersion, ConnectionVersion uint64
	ObservedAt                      string
}

func (f FrozenScope) Validate() error {
	if !f.Scope.Valid() || !scopeIdentifier.MatchString(f.Owner.OrganizationID) ||
		!scopeIdentifier.MatchString(f.Owner.ActorID) || !scopeIdentifier.MatchString(f.ConversationID) ||
		!scopeIdentifier.MatchString(f.MemberID) || (f.Scope.ProjectID != "") != (f.ProjectRevision != 0) ||
		(f.Scope.StoreID != "") != (f.StoreVersion != 0) ||
		f.Scope.ProjectID == "" && (f.ModelInput.ProjectTitle != "" || f.ModelInput.ProjectGoal != "") ||
		f.Scope.ProjectID != "" && f.ModelInput.ProjectTitle == "" ||
		f.Scope.StoreID == "" && (f.ModelInput.StoreName != "" || f.ModelInput.Platform != "" || f.ModelInput.Region != "") ||
		f.Scope.StoreID != "" && f.ModelInput.StoreName == "" {
		return ErrInvalid
	}
	if choice := f.StoreChoice; choice != nil {
		if choice.Kind == "none" && choice.Store != nil || choice.Kind == "store" && choice.Store == nil || choice.Kind != "none" && choice.Kind != "store" {
			return ErrInvalid
		}
	}
	if f.StoreChoice != nil && f.StoreChoice.Store != nil {
		s := f.StoreChoice.Store
		if !scopeIdentifier.MatchString(s.StoreID) || s.Name == "" || s.Platform == "" || s.StoreVersion == 0 || s.ConnectionVersion == 0 || len(s.Name) > 512 || len(s.Platform) > 128 || len(s.Region) > 128 || !utf8.ValidString(s.Name) || !utf8.ValidString(s.Platform) || !utf8.ValidString(s.Region) {
			return ErrInvalid
		}
		if _, err := time.Parse(time.RFC3339Nano, s.ObservedAt); err != nil {
			return ErrInvalid
		}
	}
	for _, field := range []struct {
		value string
		limit int
	}{
		{f.ModelInput.ProjectTitle, 256}, {f.ModelInput.ProjectGoal, 4096},
		{f.ModelInput.StoreName, 512}, {f.ModelInput.Platform, 128}, {f.ModelInput.Region, 128},
	} {
		if len(field.value) > field.limit || !utf8.ValidString(field.value) {
			return ErrInvalid
		}
	}
	raw, err := json.Marshal(f)
	if err != nil || len(raw) > 16<<10 {
		return ErrInvalid
	}
	return nil
}

func scopeDigest(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (f FrozenScope) Digest() string { return scopeDigest(f) }

func (r ScopeReader) Preview(ctx context.Context, owner Scope, selection ScopeSelection) (ScopeView, string, error) {
	if !selection.Valid() {
		return ScopeView{}, "", ErrInvalid
	}
	s := BusinessScope{Version: BusinessScopeVersion, Kind: selection.Kind, ProjectID: selection.ProjectID, StoreID: selection.StoreID}
	f, err := r.read(ctx, owner, s, true, true)
	if err != nil {
		return ScopeView{}, "", err
	}
	view := ScopeView{Scope: f.Scope, ProjectName: f.ModelInput.ProjectTitle, StoreName: f.ModelInput.StoreName}
	return view, scopeDigest(struct {
		Owner Scope
		View  ScopeView
	}{owner, view}), nil
}

// FilterOption observes only the selected resource's current read authority.
// A project's current store binding cannot disqualify older scoped tasks.
func (r ScopeReader) FilterOption(ctx context.Context, owner Scope, selection ScopeSelection) (ScopeView, string, error) {
	if !selection.Valid() {
		return ScopeView{}, "", ErrInvalid
	}
	fixed := BusinessScope{Version: BusinessScopeVersion, Kind: selection.Kind, ProjectID: selection.ProjectID, StoreID: selection.StoreID}
	var view ScopeView
	var err error
	if selection.Kind == "project" {
		if r.Project == nil {
			return ScopeView{}, "", ErrUnavailable
		}
		project, e := r.Project(ctx, owner, selection.ProjectID)
		if e != nil {
			return ScopeView{}, "", e
		}
		if project.ID != selection.ProjectID || project.Title == "" {
			return ScopeView{}, "", ErrUnavailable
		}
		view = ScopeView{Scope: fixed, ProjectName: project.Title}
	} else {
		view, err = r.View(ctx, owner, fixed)
		if err != nil {
			return ScopeView{}, "", err
		}
	}
	return view, scopeDigest(struct {
		Purpose string
		Owner   Scope
		View    ScopeView
	}{"task-filter", owner, view}), nil
}

func (r ScopeReader) Freeze(ctx context.Context, owner Scope, conversation, member string, scope BusinessScope) (FrozenScope, error) {
	f, err := r.read(ctx, owner, scope, false, true)
	if err != nil {
		return FrozenScope{}, err
	}
	f.ConversationID, f.MemberID = conversation, member
	return f, f.Validate()
}

// FreezeStoreChoice never calls a provider; Connection reads the owner-held
// observation. No choice is distinct from an explicit clear.
func (r ScopeReader) FreezeStoreChoice(ctx context.Context, owner Scope, choice *StoreChoiceInput) (*StoreChoiceScope, error) {
	if choice == nil {
		return nil, nil
	}
	if !choice.Valid() {
		return nil, ErrInvalid
	}
	if choice.Kind == "none" {
		return &StoreChoiceScope{Kind: "none"}, nil
	}
	if r.Store == nil || r.Connection == nil {
		return nil, ErrUnavailable
	}
	view, confirmation, err := r.Preview(ctx, owner, ScopeSelection{Kind: "store", StoreID: choice.StoreID})
	if err != nil {
		return nil, err
	}
	if confirmation != choice.ScopeConfirmation {
		return nil, ErrProposalStale
	}
	store, err := r.Store(ctx, owner, choice.StoreID)
	if err != nil {
		return nil, err
	}
	connection, err := r.Connection(ctx, owner, choice.StoreID)
	if err != nil {
		return nil, err
	}
	if store.ID != choice.StoreID || view.StoreName != store.Name || !connection.Connected || connection.Version != choice.ConnectionVersion {
		return nil, ErrProposalStale
	}
	selected := &StoreChoiceScope{Kind: "store", Store: &SelectedStoreScope{StoreID: store.ID, Name: store.Name, Platform: store.Platform, Region: store.Region, StoreVersion: store.Version, ConnectionVersion: connection.Version, ObservedAt: connection.ObservedAt}}
	return selected, nil
}

// View rechecks read permissions against the originally bound resources. A
// rebind does not cause the historical scope to inherit the new store.
func (r ScopeReader) View(ctx context.Context, owner Scope, scope BusinessScope) (ScopeView, error) {
	f, err := r.read(ctx, owner, scope, false, false)
	if err != nil {
		return ScopeView{}, err
	}
	return ScopeView{Scope: f.Scope, ProjectName: f.ModelInput.ProjectTitle, StoreName: f.ModelInput.StoreName}, nil
}

func (r ScopeReader) Check(ctx context.Context, frozen FrozenScope, member string, freshInput bool) error {
	if err := frozen.Validate(); err != nil {
		return ErrUnavailable
	}
	if frozen.MemberID != member {
		return ErrNotFound
	}
	current, err := r.read(ctx, frozen.Owner, frozen.Scope, false, freshInput)
	if err != nil {
		return err
	}
	if freshInput && current.ModelInput != frozen.ModelInput {
		return ErrProposalStale
	}
	if frozen.StoreChoice != nil && frozen.StoreChoice.Store != nil {
		selected := frozen.StoreChoice.Store
		if r.Store == nil {
			return ErrUnavailable
		}
		currentStore, err := r.Store(ctx, frozen.Owner, selected.StoreID)
		if err != nil {
			return err
		}
		if currentStore.ID != selected.StoreID {
			return ErrUnavailable
		}
		if freshInput && (currentStore.Name != selected.Name || currentStore.Platform != selected.Platform || currentStore.Region != selected.Region || currentStore.Version != selected.StoreVersion) {
			return ErrProposalStale
		}
		if r.Connection == nil {
			return ErrUnavailable
		}
		connection, err := r.Connection(ctx, frozen.Owner, selected.StoreID)
		if err != nil {
			return err
		}
		if !connection.Connected {
			return ErrUnavailable
		}
		if freshInput && connection.Version != selected.ConnectionVersion {
			return ErrProposalStale
		}
	}
	return nil
}

func (r ScopeReader) read(ctx context.Context, owner Scope, fixed BusinessScope, selectStore, freshInput bool) (FrozenScope, error) {
	if !fixed.Valid() {
		return FrozenScope{}, ErrInvalid
	}
	f := FrozenScope{Owner: owner, Scope: fixed}
	if fixed.ProjectID != "" {
		if r.Project == nil {
			return FrozenScope{}, ErrUnavailable
		}
		p, err := r.Project(ctx, owner, fixed.ProjectID)
		if err != nil {
			return FrozenScope{}, err
		}
		if p.ID != fixed.ProjectID || p.Revision == 0 || p.Title == "" {
			return FrozenScope{}, ErrUnavailable
		}
		if freshInput && (p.Archived || !selectStore && p.StoreID != fixed.StoreID) {
			return FrozenScope{}, ErrProposalStale
		}
		if selectStore {
			f.Scope.StoreID = p.StoreID
		}
		f.ProjectRevision, f.ModelInput.ProjectTitle, f.ModelInput.ProjectGoal = p.Revision, p.Title, p.Goal
	}
	if f.Scope.StoreID != "" {
		if r.Store == nil || !f.Scope.Valid() {
			return FrozenScope{}, ErrUnavailable
		}
		s, err := r.Store(ctx, owner, f.Scope.StoreID)
		if err != nil {
			return FrozenScope{}, err
		}
		if s.ID != f.Scope.StoreID || s.Version == 0 || s.Name == "" {
			return FrozenScope{}, ErrUnavailable
		}
		f.StoreVersion, f.ModelInput.StoreName, f.ModelInput.Platform, f.ModelInput.Region = s.Version, s.Name, s.Platform, s.Region
	}
	return f, nil
}

func (c PlanningCommand) ScopeRef() agent.WorkbenchContextRef {
	f := c.FrozenScope
	if f == nil || f.Validate() != nil || f.Owner != c.Scope || f.ConversationID != c.ConversationID ||
		f.MemberID != c.MemberID || f.Digest() != c.ScopeDigest {
		return agent.WorkbenchContextRef{}
	}
	return agent.WorkbenchContextRef{Contract: BusinessScopeVersion, ConversationID: c.ConversationID, CommandKey: c.IdempotencyKey, Digest: c.ScopeDigest}
}

type taskScopeCursor struct {
	Version string
	Query   string
	First   string
	Last    string
}

func scopeQueryDigest(owner Scope, filter ScopeSelection) string {
	return scopeDigest(struct {
		Owner  Scope
		Filter ScopeSelection
	}{owner, filter})
}

func canonicalTaskID(id string) bool {
	value, err := uuid.Parse(id)
	return err == nil && value != uuid.Nil && value.String() == id
}

func EncodeTaskScopeCursor(owner Scope, filter ScopeSelection, first, last string) (string, error) {
	if !filter.ValidTaskFilter() || !canonicalTaskID(first) || !canonicalTaskID(last) {
		return "", ErrInvalid
	}
	raw, err := json.Marshal(taskScopeCursor{BusinessScopeVersion, scopeQueryDigest(owner, filter), first, last})
	if err != nil {
		return "", ErrUnavailable
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func DecodeTaskScopeCursor(owner Scope, filter ScopeSelection, cursor string) (string, string, error) {
	if len(cursor) > 512 || !filter.ValidTaskFilter() {
		return "", "", ErrInvalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return "", "", ErrInvalid
	}
	var data taskScopeCursor
	bad, err := sigjson.UnmarshalStrict(raw, &data)
	if err != nil || len(bad) != 0 || data.Version != BusinessScopeVersion ||
		data.Query != scopeQueryDigest(owner, filter) || !canonicalTaskID(data.First) || !canonicalTaskID(data.Last) {
		return "", "", ErrInvalid
	}
	return data.First, data.Last, nil
}
