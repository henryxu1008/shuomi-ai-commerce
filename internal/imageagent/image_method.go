package imageagent

import (
	"context"
	"strings"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
)

const HistoricalImageMethodPolicyVersion = "image-method-prompt-v1"
const ImageMethodPolicyVersion = "image-method-visual-brief-v2"

type ImageMethodBinding struct {
	Skill         agent.SkillVersionRef
	PolicyVersion string
}

func (m ImageMethodBinding) Valid() bool {
	return (m.PolicyVersion == ImageMethodPolicyVersion || m.PolicyVersion == HistoricalImageMethodPolicyVersion) && m.Skill.Valid() && (m.Skill.OwnerKind == "PLATFORM" || m.Skill.OwnerKind == "ORGANIZATION")
}

func (m ImageMethodBinding) Executable() bool {
	return m.Valid() && m.PolicyVersion == ImageMethodPolicyVersion
}

func ValidateImageMethodDispatch(method *ImageMethodBinding) error {
	if method != nil && !method.Executable() {
		return ErrCommandBlocked
	}
	return nil
}

// The adapter resolves current image permissions and an exact immutable package.
// Release holds its availability fence through the caller's original operation.
type ImageMethodContext interface {
	AcquireImageMethod(context.Context, ExecutionIdentity, agent.SkillVersionRef, string) (string, func(), error)
}

func WithImageMethodContext(methods ImageMethodContext) ServiceOption {
	return func(s *Service) error {
		if methods == nil {
			return ErrValidation
		}
		s.imageMethods = methods
		return nil
	}
}

func AcquireImageMethod(ctx context.Context, methods ImageMethodContext, id ExecutionIdentity, binding *ImageMethodBinding) (string, func(), error) {
	if binding == nil {
		return "", func() {}, nil
	}
	if methods == nil || !binding.Valid() {
		return "", nil, ErrCommandBlocked
	}
	text, release, err := methods.AcquireImageMethod(ctx, id, binding.Skill, binding.PolicyVersion)
	if err != nil {
		return "", nil, err
	}
	if release == nil {
		return "", nil, ErrCommandBlocked
	}
	if binding.PolicyVersion == HistoricalImageMethodPolicyVersion {
		return "", release, nil
	}
	if strings.TrimSpace(text) == "" || !agentconfig.ValidSetText(text, 4<<10) {
		release()
		return "", nil, ErrValidation
	}
	return text, release, nil
}

func ComposeImageMethodPrompt(base, material string) (string, error) {
	if strings.TrimSpace(material) == "" || !agentconfig.ValidSetText(material, 4<<10) {
		return "", ErrCommandBlocked
	}
	prompt := base + "\n\nFor visual composition only, within the product facts, selected task and restrictions above:\n" + material
	if !agentconfig.ValidSetText(prompt, 16<<10) {
		return "", ErrValidation
	}
	return prompt, nil
}
