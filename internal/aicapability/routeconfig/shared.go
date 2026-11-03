package routeconfig

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"task-processor/internal/aicapability"
)

const SharedModelCapability = "ai.models.enterprise"

type ModelCategory string

const (
	ModelCategoryText       ModelCategory = "text"
	ModelCategoryImage      ModelCategory = "image-generation"
	ModelCategoryMultimodal ModelCategory = "multimodal"
)

type ModelBindingReference struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// SharedModelSelection contains choices only. Mutable prices, FX and secrets
// belong to their original owners and are resolved again for each execution.
type SharedModelSelection struct {
	SchemaVersion  int                   `json:"schemaVersion"`
	UnitVersion    string                `json:"unitVersion"`
	DefaultChannel string                `json:"defaultChannel"`
	Text           ModelBindingReference `json:"text"`
	Image          ModelBindingReference `json:"image"`
	Multimodal     ModelBindingReference `json:"multimodal"`
}

func (s SharedModelSelection) Validate() error {
	if s.SchemaVersion != 1 || s.UnitVersion != aicapability.AIPointMicrounitV1 || !ValidFact(s.DefaultChannel, 128) {
		return ErrInvalid
	}
	for _, ref := range []ModelBindingReference{s.Text, s.Image, s.Multimodal} {
		if !ValidFact(ref.ID, 64) || !ValidFact(ref.Version, 128) {
			return ErrInvalid
		}
	}
	return nil
}

func (s SharedModelSelection) reference(category ModelCategory) (ModelBindingReference, error) {
	if s.Validate() != nil {
		return ModelBindingReference{}, ErrInvalid
	}
	switch category {
	case ModelCategoryText:
		return s.Text, nil
	case ModelCategoryImage:
		return s.Image, nil
	case ModelCategoryMultimodal:
		return s.Multimodal, nil
	default:
		return ModelBindingReference{}, ErrInvalid
	}
}

func sharedDigest(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", ErrInvalid
	}
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:]), nil
}

func (s SharedModelSelection) CategoryDigest(category ModelCategory) (string, error) {
	ref, err := s.reference(category)
	if err != nil {
		return "", err
	}
	return sharedDigest(struct {
		SchemaVersion int
		Unit, Channel string
		Category      ModelCategory
		Reference     ModelBindingReference
	}{s.SchemaVersion, s.UnitVersion, s.DefaultChannel, category, ref})
}

// ModelQualification is a non-secret, use-time result from trusted admission.
// A concrete catalog must verify every opened consumer, current credentials,
// applicable price/FX and conservative bounds. The service never infers these
// from a model name, a saved choice or caller-provided readiness.
type ModelQualification struct {
	OrganizationID       string                               `json:"organizationId"`
	Category             ModelCategory                        `json:"category"`
	Reference            ModelBindingReference                `json:"reference"`
	Channel              string                               `json:"channel"`
	Model                string                               `json:"model"`
	Ready                bool                                 `json:"ready"`
	Reason               string                               `json:"reason"`
	BoundEvidenceVersion string                               `json:"boundEvidenceVersion"`
	ProfileVersion       string                               `json:"profileVersion"`
	Price                aicapability.OfficialModelTariff     `json:"price"`
	Text                 *aicapability.ModelProfile           `json:"-"`
	TextProfiles         map[string]aicapability.ModelProfile `json:"-"`
	Image                *ImageProfile                        `json:"-"`
}

type SharedCategoryReader interface {
	ReadCategory(context.Context, string, ModelCategory) (ModelQualification, string, error)
}

type ModelOption struct {
	Category  ModelCategory                     `json:"category"`
	Reference ModelBindingReference             `json:"reference"`
	Channel   string                            `json:"channel"`
	Model     string                            `json:"model"`
	Ready     bool                              `json:"ready"`
	Reason    string                            `json:"reason,omitempty"`
	Price     *aicapability.OfficialModelTariff `json:"price,omitempty"`
}
type SharedModelOptions interface {
	Options(context.Context, string) ([]ModelOption, error)
}

type SharedModelCatalog interface {
	ValidateDefaultChannel(context.Context, string, string) error
	QualifyModel(context.Context, string, ModelCategory, ModelBindingReference) (ModelQualification, error)
}

type SharedModelPreview struct {
	Ready  bool                 `json:"ready"`
	Models []ModelQualification `json:"models"`
}

type SharedModelInput struct {
	ExpectedRevision string               `json:"expectedRevision"`
	Selection        SharedModelSelection `json:"selection"`
}

// SharedModelService reuses the original revision/current transaction and
// receipt repository. It does not dispatch, grant permissions or mint resources.
// HTTP/runtime callers must first obtain verified identity and fresh authority.
type SharedModelService struct {
	Repository Repository
	Catalog    SharedModelCatalog
}

func (s *SharedModelService) qualify(ctx context.Context, org string, category ModelCategory, ref ModelBindingReference) (ModelQualification, error) {
	if s == nil || s.Catalog == nil || ctx == nil || ctx.Err() != nil || !ValidFact(org, 128) {
		return ModelQualification{}, ErrUnavailable
	}
	q, err := s.Catalog.QualifyModel(ctx, org, category, ref)
	if err != nil {
		return ModelQualification{}, err
	}
	if q.OrganizationID != org || q.Category != category || q.Reference != ref || !ValidFact(q.Channel, 128) || !ValidFact(q.Model, 128) {
		return ModelQualification{}, ErrInvalid
	}
	if q.Ready && (q.Price.Validate() != nil || q.Price.Model != q.Model || q.Price.Channel != q.Channel || !ValidFact(q.BoundEvidenceVersion, 128) || !ValidFact(q.ProfileVersion, 128)) {
		q.Ready, q.Reason = false, "DEPENDENCY_MISSING"
	}
	if q.Ready {
		valid := false
		if category == ModelCategoryImage && q.Image != nil && q.Text == nil {
			n, e := q.Image.Billing.ReservationMicropoints(q.Model)
			valid = e == nil && n == q.Image.Points && q.Image.Model == q.Model && q.Image.Billing.OfficialTariff == q.Price && q.Image.PriceVersion == q.Price.PriceVersion
		} else if category != ModelCategoryImage && q.Text != nil && q.Image == nil {
			valid = q.Text.Validate() == nil && q.Text.ModelID == q.Model && q.Text.OfficialTariff == q.Price
		}
		if !valid {
			q.Ready, q.Reason = false, "CONSUMER_NOT_QUALIFIED"
		}
	}
	if q.Ready {
		q.Reason = ""
	} else if !ValidFact(q.Reason, 64) {
		q.Reason = "DEPENDENCY_MISSING"
	}
	return q, nil
}

func (s *SharedModelService) Preview(ctx context.Context, org string, selection SharedModelSelection) (SharedModelPreview, error) {
	if selection.Validate() != nil {
		return SharedModelPreview{}, ErrInvalid
	}
	if s == nil || s.Catalog == nil || ctx == nil || ctx.Err() != nil || !ValidFact(org, 128) {
		return SharedModelPreview{}, ErrUnavailable
	}
	if err := s.Catalog.ValidateDefaultChannel(ctx, org, selection.DefaultChannel); err != nil {
		return SharedModelPreview{}, err
	}
	out := SharedModelPreview{Ready: true, Models: make([]ModelQualification, 0, 3)}
	for _, category := range []ModelCategory{ModelCategoryText, ModelCategoryImage, ModelCategoryMultimodal} {
		ref, _ := selection.reference(category)
		q, err := s.qualify(ctx, org, category, ref)
		if err != nil {
			return SharedModelPreview{}, err
		}
		out.Ready = out.Ready && q.Ready
		out.Models = append(out.Models, q)
	}
	return out, nil
}

func (s *SharedModelService) Publish(ctx context.Context, org, actor, key string, input SharedModelInput) (Revision, error) {
	if s == nil || s.Repository == nil || ctx == nil || ctx.Err() != nil || !ValidFact(org, 128) || !ValidFact(actor, 128) || !ValidFact(key, 128) || !ValidFact(input.ExpectedRevision, 128) || input.Selection.Validate() != nil {
		return Revision{}, ErrInvalid
	}
	scope := Scope{OrganizationID: org, Capability: SharedModelCapability}
	fingerprint, err := sharedDigest(struct {
		Scope Scope
		Actor string
		Input SharedModelInput
	}{scope, actor, input})
	if err != nil {
		return Revision{}, err
	}
	// Repository.Publish checks the original receipt before invoking build,
	// then commits one complete payload under its existing CAS transaction.
	cmd := Command{Scope: scope, ActorID: actor, Key: key, Fingerprint: fingerprint, Input: Input{ExpectedRevision: input.ExpectedRevision}}
	return s.Repository.Publish(ctx, cmd, func(_ CredentialReader, _ string) (Payload, error) {
		preview, err := s.Preview(ctx, org, input.Selection)
		if err != nil {
			return Payload{}, err
		}
		if !preview.Ready {
			return Payload{}, ErrNeedsConfiguration
		}
		selection := input.Selection
		return Payload{SharedModels: &selection}, nil
	})
}

// ReadCategory fresh-qualifies only the required category. Other category
// changes cannot silently replace this model or invalidate its unchanged choice.
// This remains configuration evidence, not a provider dispatch authorization.
func (s *SharedModelService) ReadCategory(ctx context.Context, org string, category ModelCategory) (ModelQualification, string, error) {
	if s == nil || s.Repository == nil || ctx == nil || ctx.Err() != nil || !ValidFact(org, 128) {
		return ModelQualification{}, "", ErrUnavailable
	}
	scope := Scope{OrganizationID: org, Capability: SharedModelCapability}
	r, err := s.Repository.Current(ctx, scope)
	if err != nil {
		return ModelQualification{}, "", err
	}
	p := r.Payload
	if r.Scope != scope || r.SchemaVersion != 1 || p.SharedModels == nil || p.PlatformPrices != nil || p.Text != nil || p.Image != nil || p.Optional || p.BindingID != "" || p.CredentialVersion != "" || p.Rates != (Rates{}) {
		return ModelQualification{}, "", ErrInvalid
	}
	ref, err := p.SharedModels.reference(category)
	if err != nil {
		return ModelQualification{}, "", err
	}
	q, err := s.qualify(ctx, org, category, ref)
	if err != nil {
		return ModelQualification{}, "", err
	}
	if !q.Ready {
		return q, "", ErrNeedsConfiguration
	}
	choiceDigest, _ := p.SharedModels.CategoryDigest(category)
	priceDigest, _ := q.Price.Digest()
	digest, err := sharedDigest(struct {
		Organization, Choice, Price, Profile, Bound string
	}{org, choiceDigest, priceDigest, q.ProfileVersion, q.BoundEvidenceVersion})
	return q, digest, err
}

func (s *SharedModelService) Current(ctx context.Context, org string) (Revision, SharedModelPreview, error) {
	if s == nil || s.Repository == nil || !ValidFact(org, 128) {
		return Revision{}, SharedModelPreview{}, ErrUnavailable
	}
	scope := Scope{OrganizationID: org, Capability: SharedModelCapability}
	r, err := s.Repository.Current(ctx, scope)
	if err != nil {
		return r, SharedModelPreview{}, err
	}
	p := r.Payload
	if r.Scope != scope || r.SchemaVersion != 1 || p.SharedModels == nil || p.PlatformPrices != nil || p.Image != nil || p.Text != nil || p.Optional || p.BindingID != "" || p.CredentialVersion != "" || p.Rates != (Rates{}) {
		return Revision{}, SharedModelPreview{}, ErrInvalid
	}
	preview, err := s.Preview(ctx, org, *p.SharedModels)
	return r, preview, err
}
