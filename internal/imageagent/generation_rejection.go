package imageagent

import (
	"context"
	"regexp"
)

// GenerationRejection is bounded evidence from an admitted adapter, never an
// HTTP command or an error-string inference. The owner binds it to the intent.
type GenerationRejection struct {
	Kind         string
	Digest       string
	RequestID    string
	HTTPStatus   int
	ProviderCode string
}

var rejectionRequestID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func (p GenerationRejection) ValidateFor(f GenerationFact) error {
	if f.Validate() != nil || !generationDigest(p.Digest) {
		return ErrValidation
	}
	switch p.Kind {
	case "bailian_free_tier":
		if f.Intent.Provider != "bailian" || f.Intent.Protocol != "dashscope-image-sync-v1" || p.HTTPStatus != 403 || p.ProviderCode != "AllocationQuota.FreeTierOnly" || !rejectionRequestID.MatchString(p.RequestID) {
			return ErrValidation
		}
	case "source_edit_preflight":
		if f.Intent.InputProtocol != ImageSetSchema || p.RequestID != "" || p.HTTPStatus != 0 || p.ProviderCode != "" {
			return ErrValidation
		}
	default:
		return ErrValidation
	}
	return nil
}

// This narrow port atomically appends immutable evidence and changes the
// existing terminal fact. It cannot create a reservation or invoke a provider.
type GenerationRejectionRecorder interface {
	RecordGenerationRejection(context.Context, GenerationIntent, GenerationRejection) (GenerationFact, error)
}

func (f GenerationFact) RecordRejection(proof GenerationRejection) (GenerationFact, error) {
	if err := proof.ValidateFor(f); err != nil {
		return f, err
	}
	if f.State != GenerationDispatchStarted && f.State != GenerationUnknown && f.State != GenerationNoEffect {
		return f, ErrRevisionConflict
	}
	if f.Success != (GenerationSuccess{}) {
		return f, ErrRevisionConflict
	}
	f.State = GenerationNoEffect
	return f, nil
}
