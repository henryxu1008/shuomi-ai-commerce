package routeconfig

import (
	"context"
	"errors"
	"sort"

	"task-processor/internal/aicapability"
)

type TextChoiceInput struct {
	ExpectedRevision string                `json:"expectedRevision"`
	Reference        ModelBindingReference `json:"reference"`
	Quote            string                `json:"quote"`
}

type TextChoiceView struct {
	Revision string        `json:"revision"`
	Current  *ModelOption  `json:"current,omitempty"`
	Purposes []string      `json:"purposes,omitempty"`
	Options  []ModelOption `json:"options"`
}

func (q ModelQualification) TextChoiceToken() (string, error) {
	if q.Category != ModelCategoryText || !q.Ready {
		return "", ErrNeedsConfiguration
	}
	return sharedDigest(struct {
		Org     string
		Ref     ModelBindingReference
		Price   aicapability.OfficialModelTariff
		Profile string
	}{q.OrganizationID, q.Reference, q.Price, q.ProfileVersion})
}

func textOption(q ModelQualification) ModelOption {
	option := ModelOption{Category: q.Category, Reference: q.Reference, Channel: q.Channel, ChannelName: q.ChannelName, Model: q.Model, Source: q.Source, Ready: q.Ready, Reason: q.Reason}
	if q.Ready {
		if q.Text != nil {
			option.BillingMode = q.Text.BillingMode
		}
		if q.Price.Valid() {
			option.Price = &q.Price
		}
		option.Quote, _ = q.TextChoiceToken()
	}
	return option
}

func readTextSnapshot(ctx context.Context, reader CurrentReader, scope Scope) (SharedModelSelection, string, error) {
	r, err := reader.Current(ctx, scope)
	if errors.Is(err, ErrNotFound) {
		return SharedModelSelection{SchemaVersion: 2, UnitVersion: aicapability.AIPointMicrounitV1}, NoRevision, nil
	}
	if err != nil {
		return SharedModelSelection{}, "", err
	}
	if r.Scope != scope || !validSharedRevision(r) || r.Payload.SharedModels.Validate() != nil {
		return SharedModelSelection{}, "", ErrInvalid
	}
	return *r.Payload.SharedModels, r.ID, nil
}

func (s *SharedModelService) TextChoices(ctx context.Context, org string) (TextChoiceView, error) {
	if s == nil || s.Repository == nil || s.Catalog == nil || !ValidFact(org, 128) {
		return TextChoiceView{}, ErrUnavailable
	}
	choice, version, err := readTextSnapshot(ctx, s.Repository, Scope{OrganizationID: org, Capability: SharedModelCapability})
	if err != nil {
		return TextChoiceView{}, err
	}
	result := TextChoiceView{Revision: version, Options: []ModelOption{}}
	if choice.Text.ID != "" {
		q, err := s.qualify(ctx, org, ModelCategoryText, choice.Text)
		if err != nil {
			return TextChoiceView{}, err
		}
		if q.Reason == "PLATFORM_ACCESS_WITHDRAWN" || q.Reason == "PLATFORM_BINDING_REPLACED" || q.Reason == "PLATFORM_CHANNEL_UNAVAILABLE" {
			q.ChannelName, q.Model = choice.TextChannelName, choice.TextModel
		}
		current := textOption(q)
		result.Current = &current
	}
	readOptions := s.Catalog.Options
	if textCatalog, ok := s.Catalog.(interface {
		TextOptions(context.Context, string) ([]ModelOption, error)
	}); ok {
		readOptions = textCatalog.TextOptions
	}
	options, err := readOptions(ctx, org)
	if err != nil {
		return TextChoiceView{}, err
	}
	for _, option := range options {
		if option.Category != ModelCategoryText {
			continue
		}
		q, err := s.qualify(ctx, org, ModelCategoryText, option.Reference)
		if err != nil {
			return TextChoiceView{}, err
		}
		if q.Ready {
			result.Options = append(result.Options, textOption(q))
			if len(result.Purposes) == 0 {
				for op := range q.TextProfiles {
					result.Purposes = append(result.Purposes, op)
				}
				sort.Strings(result.Purposes)
			}
		}
	}
	return result, nil
}

func (s *SharedModelService) ChooseText(ctx context.Context, org, actor, key string, input TextChoiceInput) (Revision, error) {
	if s == nil || s.Repository == nil || !ValidFact(org, 128) || !ValidFact(actor, 128) || !ValidFact(key, 128) ||
		!ValidFact(input.ExpectedRevision, 128) || !validReference(input.Reference) || !ValidFact(input.Quote, 128) {
		return Revision{}, ErrInvalid
	}
	scope := Scope{OrganizationID: org, Capability: SharedModelCapability}
	fingerprint, err := sharedDigest(struct {
		Operation string
		Scope     Scope
		Actor     string
		Input     TextChoiceInput
	}{"choose-text", scope, actor, input})
	if err != nil {
		return Revision{}, err
	}
	return s.Repository.Publish(ctx, Command{Scope: scope, ActorID: actor, Key: key, Fingerprint: fingerprint, Input: Input{ExpectedRevision: input.ExpectedRevision}},
		func(tx CredentialReader, _ string) (Payload, error) {
			reader, ok := tx.(CurrentReader)
			if !ok {
				// The bound store supplies its scoped transaction reader.
				if tx != nil {
					return Payload{}, ErrUnavailable
				}
				reader = s.Repository
			}
			old, version, err := readTextSnapshot(ctx, reader, scope)
			if err != nil {
				return Payload{}, err
			}
			if version != input.ExpectedRevision {
				return Payload{}, ErrConflict
			}
			if connection := PlatformConnection(input.Reference.ID); connection != "" {
				locker, ok := tx.(interface {
					LockPlatformSelection(context.Context, string) error
				})
				if !ok {
					return Payload{}, ErrUnavailable
				}
				if err := locker.LockPlatformSelection(ctx, connection); err != nil {
					return Payload{}, err
				}
			}
			if IsManagedBindingID(input.Reference.ID) {
				locker, ok := tx.(interface {
					LockManagedSelection(context.Context, string, string) error
				})
				if !ok {
					return Payload{}, ErrUnavailable
				}
				if err := locker.LockManagedSelection(ctx, org, input.Reference.ID); err != nil {
					return Payload{}, err
				}
			}
			qualified, err := s.qualify(ctx, org, ModelCategoryText, input.Reference)
			if err != nil {
				return Payload{}, err
			}
			token, err := qualified.TextChoiceToken()
			if err != nil {
				return Payload{}, err
			}
			if token != input.Quote {
				return Payload{}, ErrConflict
			}
			old.SchemaVersion, old.Text = 2, input.Reference
			old.TextChannelName, old.TextModel = qualified.ChannelName, qualified.Model
			return Payload{SharedModels: &old}, nil
		})
}
