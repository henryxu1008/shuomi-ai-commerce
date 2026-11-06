package agentconfig

import (
	"task-processor/internal/agent"
	"task-processor/internal/aicapability"
)

func ValidContentPrompt(prompt string, ref agent.WorkbenchContextRef) bool {
	return ref.Absent() && prompt == ContentPromptVersion || ref.Valid() && prompt == ScopedContentPromptVersion
}

// ScopedContentProfile is the registered prompt variant of the currently
// admitted content route. Its model, prices and bounds remain those of that
// route. A scoped request must still prove its exact Workbench command.
func ScopedContentProfile(profile aicapability.ModelProfile) aicapability.ModelProfile {
	if profile.PromptVersion == ContentPromptVersion && profile.OutputSchemaVersion == ContentActionVersion {
		profile.PromptVersion = ScopedContentPromptVersion
	}
	return profile
}
