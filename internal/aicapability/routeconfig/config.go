// Package routeconfig owns published execution configuration, not credentials,
// invocation history, resource reservations, or provider calls.
package routeconfig

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"task-processor/internal/aicapability"
)

var (
	ErrInvalid            = errors.New("invalid model route configuration")
	ErrUnavailable        = errors.New("model route configuration unavailable")
	ErrNotFound           = errors.New("model route configuration not found")
	ErrConflict           = errors.New("model route configuration conflict")
	ErrNeedsConfiguration = errors.New("model route binding needs configuration")
	ErrOutcomeUnknown     = errors.New("publication outcome unknown")
)

const ImageCapability = "imageagent.generation"
const NoRevision = "NONE"

type Scope struct {
	OrganizationID string `json:"organizationId"`
	Capability     string `json:"capability"`
}

func ValidFact(s string, max int) bool {
	return s != "" && len(s) <= max && utf8.ValidString(s) && strings.TrimSpace(s) == s && strings.IndexFunc(s, unicode.IsControl) < 0
}
func (s Scope) Valid() bool {
	if !ValidFact(s.OrganizationID, 128) {
		return false
	}
	switch aicapability.Operation(s.Capability) {
	case aicapability.OperationProductAgentDecision, aicapability.OperationProductContentDecision, aicapability.OperationListingMaterialsDecision, aicapability.OperationProductSelectionDecision, aicapability.OperationShopDiagnosisDecision, aicapability.OperationAIWorkbenchChatPlan:
		return true
	}
	return s.Capability == ImageCapability
}

type Rates struct {
	InputCost      string `json:"inputCost,omitempty"`
	OutputCost     string `json:"outputCost,omitempty"`
	InputPoints    string `json:"inputPoints,omitempty"`
	OutputPoints   string `json:"outputPoints,omitempty"`
	PointsPerImage string `json:"pointsPerImage,omitempty"`
}

// Decimal strings avoid JSON floating point and JavaScript integer truncation.
func ParseRate(s string, scale int) (int64, error) {
	if scale < 0 || scale > 6 || len(s) > 24 || s == "" {
		return 0, ErrInvalid
	}
	parts := strings.Split(s, ".")
	if len(parts) > 2 || parts[0] == "" || (len(parts[0]) > 1 && parts[0][0] == '0') {
		return 0, ErrInvalid
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if fraction == "" || len(fraction) > scale {
			return 0, ErrInvalid
		}
	}
	for _, r := range parts[0] + fraction {
		if r < '0' || r > '9' {
			return 0, ErrInvalid
		}
	}
	v, err := strconv.ParseInt(parts[0]+fraction+strings.Repeat("0", scale-len(fraction)), 10, 64)
	if err != nil || v <= 0 || v > 1e12 {
		return 0, ErrInvalid
	}
	return v, nil
}
func FormatCost(v int64) string { return fmt.Sprintf("%d.%06d", v/1000000, v%1000000) }
func normalize(r Rates, image bool) (Rates, error) {
	if image {
		if r.InputCost != "" || r.OutputCost != "" || r.InputPoints != "" || r.OutputPoints != "" {
			return Rates{}, ErrInvalid
		}
		v, e := ParseRate(r.PointsPerImage, 0)
		return Rates{PointsPerImage: strconv.FormatInt(v, 10)}, e
	}
	if r.PointsPerImage != "" {
		return Rates{}, ErrInvalid
	}
	in, e := ParseRate(r.InputCost, 6)
	if e != nil {
		return Rates{}, e
	}
	out, e := ParseRate(r.OutputCost, 6)
	if e != nil {
		return Rates{}, e
	}
	ip, e := ParseRate(r.InputPoints, 0)
	if e != nil {
		return Rates{}, e
	}
	op, e := ParseRate(r.OutputPoints, 0)
	if e != nil {
		return Rates{}, e
	}
	return Rates{InputCost: FormatCost(in), OutputCost: FormatCost(out), InputPoints: strconv.FormatInt(ip, 10), OutputPoints: strconv.FormatInt(op, 10)}, nil
}

type ImageProfile struct {
	Provider             string `json:"provider"`
	Model                string `json:"model"`
	Protocol             string `json:"protocol"`
	EndpointDigest       string `json:"endpointDigest"`
	CredentialReference  string `json:"credentialReference"`
	ConfigurationVersion string `json:"configurationVersion"`
	PriceVersion         string `json:"priceVersion"`
	Points               int64  `json:"points"`
}
type Binding struct {
	MaximumImagePoints      int64                      `json:"maximumPointsPerImage,omitempty"`
	ID                      string                     `json:"id"`
	CredentialVersion       string                     `json:"credentialVersion"`
	Provider                string                     `json:"provider"`
	Model                   string                     `json:"model"`
	Protocol                string                     `json:"protocol"`
	UsageMappingVersion     string                     `json:"usageMappingVersion,omitempty"`
	Currency                string                     `json:"currency,omitempty"`
	MaximumCostMicros       int64                      `json:"maximumCostMicros,omitempty"`
	MaximumInputBytes       int                        `json:"maximumInputBytes,omitempty"`
	MaximumOutputBytes      int                        `json:"maximumOutputBytes,omitempty"`
	MaximumPromptTokens     int64                      `json:"maximumPromptTokens,omitempty"`
	MaximumCompletionTokens int64                      `json:"maximumCompletionTokens,omitempty"`
	DeadlineSeconds         int64                      `json:"deadlineSeconds,omitempty"`
	Text                    *aicapability.ModelProfile `json:"-"`
	Image                   *ImageProfile              `json:"-"`
}
type Credential struct {
	OrganizationID, ClientName, Model, APIStyle, Version, EndpointDigest, Reference string
	TimeoutSeconds                                                                  int
	Enabled, KeyPresent                                                             bool
}
type CredentialReader interface {
	Credential(context.Context, string, string) (Credential, error)
}

// Catalog is the existing, server-owned admission description. It must not
// select a current route or make a provider request.
type Catalog interface {
	Scopes(string) []Scope
	Candidates(Scope) []string
	Qualify(Scope, string, Credential) (Binding, error)
}
type Payload struct {
	Optional          bool                       `json:"optional,omitempty"`
	BindingID         string                     `json:"bindingId"`
	CredentialVersion string                     `json:"credentialVersion"`
	Rates             Rates                      `json:"rates"`
	Text              *aicapability.ModelProfile `json:"text,omitempty"`
	Image             *ImageProfile              `json:"image,omitempty"`
}

func Freeze(b Binding, r Rates, revision string) (Payload, error) {
	if !ValidFact(revision, 128) || !ValidFact(b.ID, 64) || !ValidFact(b.CredentialVersion, 128) || (b.Text == nil) == (b.Image == nil) {
		return Payload{}, ErrInvalid
	}
	rates, e := normalize(r, b.Image != nil)
	if e != nil {
		return Payload{}, e
	}
	p := Payload{BindingID: b.ID, CredentialVersion: b.CredentialVersion, Rates: rates}
	if b.Text != nil {
		t := *b.Text
		t.RoutingPolicyVersion = revision
		t.CostPricingVersion = revision
		t.PointTariff.PriceVersion = revision
		t.InputMicrosPerMillion, _ = ParseRate(rates.InputCost, 6)
		t.OutputMicrosPerMillion, _ = ParseRate(rates.OutputCost, 6)
		t.PointTariff.InputPointsPerMillionTokens, _ = ParseRate(rates.InputPoints, 0)
		t.PointTariff.OutputPointsPerMillionTokens, _ = ParseRate(rates.OutputPoints, 0)
		cost, err := t.MaximumCost()
		if t.Validate() != nil || err != nil || b.MaximumCostMicros <= 0 || cost > b.MaximumCostMicros {
			return Payload{}, ErrInvalid
		}
		p.Text = &t
	} else {
		i := *b.Image
		i.Points, _ = ParseRate(rates.PointsPerImage, 0)
		if b.MaximumImagePoints <= 0 || i.Points > b.MaximumImagePoints {
			return Payload{}, ErrInvalid
		}
		i.PriceVersion = revision
		i.ConfigurationVersion = revision
		p.Image = &i
	}
	return p, nil
}

type Input struct {
	Optional          bool   `json:"optional,omitempty"`
	ExpectedRevision  string `json:"expectedRevision"`
	BindingID         string `json:"bindingId"`
	CredentialVersion string `json:"credentialVersion"`
	Rates             Rates  `json:"rates"`
	RollbackFrom      string `json:"rollbackFrom,omitempty"`
}
type Revision struct {
	ID            string    `json:"id"`
	Scope         Scope     `json:"scope"`
	Parent        string    `json:"parent"`
	SchemaVersion int       `json:"schemaVersion"`
	Payload       Payload   `json:"payload"`
	Digest        string    `json:"digest"`
	ActorID       string    `json:"actorId"`
	PublishedAt   time.Time `json:"publishedAt"`
	Key           string    `json:"key"`
	Fingerprint   string    `json:"-"`
	RollbackFrom  string    `json:"rollbackFrom,omitempty"`
}
type Command struct {
	Scope                     Scope
	ActorID, Key, Fingerprint string
	Input                     Input
}
type Repository interface {
	CredentialReader
	Optional(context.Context, Scope, string) (Revision, error)
	Current(context.Context, Scope) (Revision, error)
	Receipt(context.Context, Scope, string) (Revision, error)
	Revision(context.Context, Scope, string) (Revision, error)
	History(context.Context, Scope, string, int) ([]Revision, error)
	Publish(context.Context, Command, func(CredentialReader, string) (Payload, error)) (Revision, error)
}
type CurrentReader interface {
	Current(context.Context, Scope) (Revision, error)
}

type OptionalReader interface {
	CurrentReader
	Optional(context.Context, Scope, string) (Revision, error)
	Options(context.Context, Scope) ([]Revision, error)
}

type Service struct {
	Repository Repository
	Catalog    Catalog
}

func (s *Service) supported(scope Scope) bool {
	if s == nil || s.Repository == nil || s.Catalog == nil || !scope.Valid() {
		return false
	}
	for _, sc := range s.Catalog.Scopes(scope.OrganizationID) {
		if sc == scope {
			return true
		}
	}
	return false
}
func (s *Service) binding(ctx context.Context, scope Scope, id, version string, reader CredentialReader) (Binding, error) {
	allowed := false
	for _, v := range s.Catalog.Candidates(scope) {
		allowed = allowed || v == id
	}
	if !allowed {
		return Binding{}, ErrNeedsConfiguration
	}
	c, e := reader.Credential(ctx, scope.OrganizationID, id)
	if e != nil {
		return Binding{}, e
	}
	b, e := s.Catalog.Qualify(scope, id, c)
	if e != nil || b.ID != id || b.CredentialVersion != version {
		return Binding{}, ErrNeedsConfiguration
	}
	return b, nil
}
func (s *Service) Bindings(ctx context.Context, scope Scope) ([]Binding, error) {
	if !s.supported(scope) {
		return nil, ErrUnavailable
	}
	out := []Binding{}
	for _, id := range s.Catalog.Candidates(scope) {
		c, e := s.Repository.Credential(ctx, scope.OrganizationID, id)
		if e != nil {
			return nil, e
		}
		b, e := s.Catalog.Qualify(scope, id, c)
		if e == nil {
			out = append(out, b)
		}
	}
	return out, nil
}
func (s *Service) Current(ctx context.Context, scope Scope) (Revision, error) {
	if !s.supported(scope) {
		return Revision{}, ErrUnavailable
	}
	r, e := s.Repository.Current(ctx, scope)
	if e != nil {
		return r, e
	}
	if r.Payload.Optional {
		return r, ErrNeedsConfiguration
	}
	return s.qualifiedRevision(ctx, scope, r)
}

func (s *Service) qualifiedRevision(ctx context.Context, scope Scope, r Revision) (Revision, error) {
	b, e := s.binding(ctx, scope, r.Payload.BindingID, r.Payload.CredentialVersion, s.Repository)
	if e != nil {
		return r, ErrNeedsConfiguration
	}
	p, e := Freeze(b, r.Payload.Rates, r.ID)
	p.Optional = r.Payload.Optional
	if p.Optional && !optionalBinding(scope, b) {
		return r, ErrNeedsConfiguration
	}
	if e != nil || !reflect.DeepEqual(p, r.Payload) {
		return r, ErrNeedsConfiguration
	}
	return r, nil
}

func optionalBinding(scope Scope, b Binding) bool {
	return scope.Capability == ImageCapability && b.Image != nil && b.Provider == "grsai" && b.Model == "gpt-image-2-vip" && b.Protocol == "grsai-json-sync-v1"
}

func (s *Service) Optional(ctx context.Context, scope Scope, binding string) (Revision, error) {
	if !s.supported(scope) || scope.Capability != ImageCapability || !ValidFact(binding, 64) {
		return Revision{}, ErrUnavailable
	}
	r, e := s.Repository.Optional(ctx, scope, binding)
	if e != nil {
		return r, e
	}
	if !r.Payload.Optional || r.Payload.BindingID != binding {
		return r, ErrNeedsConfiguration
	}
	return s.qualifiedRevision(ctx, scope, r)
}

func (s *Service) Options(ctx context.Context, scope Scope) ([]Revision, error) {
	if !s.supported(scope) || scope.Capability != ImageCapability {
		return nil, ErrUnavailable
	}
	rows := []Revision{}
	for _, id := range s.Catalog.Candidates(scope) {
		r, e := s.Optional(ctx, scope, id)
		if errors.Is(e, ErrNotFound) || errors.Is(e, ErrNeedsConfiguration) {
			continue
		}
		if e != nil {
			return nil, e
		}
		rows = append(rows, r)
	}
	return rows, nil
}

// PublishedOptions is a maintenance-only head projection. Invalidated credentials
// remove member eligibility, not the durable CAS identity needed to replace it.
func (s *Service) PublishedOptions(ctx context.Context, scope Scope) ([]Revision, error) {
	if !s.supported(scope) || scope.Capability != ImageCapability {
		return nil, ErrUnavailable
	}
	rows := []Revision{}
	for _, id := range s.Catalog.Candidates(scope) {
		r, e := s.Repository.Optional(ctx, scope, id)
		if errors.Is(e, ErrNotFound) {
			continue
		}
		if e != nil {
			return nil, e
		}
		if !r.Payload.Optional || r.Payload.BindingID != id {
			return nil, ErrNeedsConfiguration
		}
		rows = append(rows, r)
	}
	return rows, nil
}
func (s *Service) canonical(ctx context.Context, scope Scope, in Input) (Input, error) {
	if !s.supported(scope) || in.Optional && scope.Capability != ImageCapability || !ValidFact(in.ExpectedRevision, 128) || !ValidFact(in.BindingID, 64) || !ValidFact(in.CredentialVersion, 128) || in.RollbackFrom != "" && !ValidFact(in.RollbackFrom, 128) {
		return Input{}, ErrInvalid
	}
	var e error
	in.Rates, e = normalize(in.Rates, scope.Capability == ImageCapability)
	return in, e
}
func (s *Service) payload(ctx context.Context, scope Scope, in Input, reader CredentialReader, revision string) (Payload, error) {
	b, e := s.binding(ctx, scope, in.BindingID, in.CredentialVersion, reader)
	if e != nil {
		return Payload{}, e
	}
	if in.Optional && !optionalBinding(scope, b) {
		return Payload{}, ErrInvalid
	}
	if in.RollbackFrom != "" {
		old, e := s.Repository.Revision(ctx, scope, in.RollbackFrom)
		if e != nil {
			return Payload{}, e
		}
		if old.Payload.Optional != in.Optional || old.Payload.BindingID != in.BindingID || old.Payload.CredentialVersion != in.CredentialVersion || old.Payload.Rates != in.Rates {
			return Payload{}, ErrConflict
		}
	}
	p, e := Freeze(b, in.Rates, revision)
	p.Optional = in.Optional
	return p, e
}
func (s *Service) Validate(ctx context.Context, scope Scope, in Input) (Payload, error) {
	in, e := s.canonical(ctx, scope, in)
	if e != nil {
		return Payload{}, e
	}
	current, e := s.Repository.Current(ctx, scope)
	if in.Optional {
		current, e = s.Repository.Optional(ctx, scope, in.BindingID)
	}
	if e != nil && !errors.Is(e, ErrNotFound) {
		return Payload{}, e
	}
	id := current.ID
	if id == "" {
		id = NoRevision
	}
	if id != in.ExpectedRevision {
		return Payload{}, ErrConflict
	}
	return s.payload(ctx, scope, in, s.Repository, "preview")
}
func (s *Service) Publish(ctx context.Context, scope Scope, actor, key string, in Input) (Revision, error) {
	in, e := s.canonical(ctx, scope, in)
	if e != nil || !ValidFact(actor, 128) || !ValidFact(key, 128) {
		return Revision{}, ErrInvalid
	}
	raw, _ := json.Marshal(struct {
		Scope Scope
		Actor string
		Input Input
	}{scope, actor, in})
	hash := sha256.Sum256(raw)
	cmd := Command{Scope: scope, ActorID: actor, Key: key, Fingerprint: hex.EncodeToString(hash[:]), Input: in}
	return s.Repository.Publish(ctx, cmd, func(reader CredentialReader, id string) (Payload, error) {
		return s.payload(ctx, scope, in, reader, id)
	})
}
