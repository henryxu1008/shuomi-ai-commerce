package workspace

type Guidance[R any, H any] struct {
	Reason      *R
	RepairHints []H
}

type ReadinessCheckSpec struct {
	Key             string
	Label           string
	OK              bool
	Message         string
	FieldPaths      []string
	SuggestedAction string
	WarningOnly     bool
	Taxonomy        ReadinessTaxonomy
}

type ReadinessTaxonomy struct {
	BlockerKey          string `json:"blocker_key,omitempty"`
	Severity            string `json:"severity,omitempty"`
	Domain              string `json:"domain,omitempty"`
	RepairTarget        string `json:"repair_target,omitempty"`
	RepairRoute         string `json:"repair_route,omitempty"`
	Recoverable         bool   `json:"recoverable,omitempty"`
	RequiresEngineering bool   `json:"requires_engineering,omitempty"`
}

type SubmitReadiness[R any, H any] struct {
	Ready         bool                   `json:"ready"`
	Status        string                 `json:"status,omitempty"`
	Summary       []string               `json:"summary,omitempty"`
	BlockingItems []ReadinessItem[R, H]  `json:"blocking_items,omitempty"`
	WarningItems  []ReadinessItem[R, H]  `json:"warning_items,omitempty"`
	Checks        []ReadinessCheck[R, H] `json:"checks,omitempty"`
}

type ReadinessItem[R any, H any] struct {
	Key             string            `json:"key,omitempty"`
	Label           string            `json:"label,omitempty"`
	Message         string            `json:"message,omitempty"`
	FieldPaths      []string          `json:"field_paths,omitempty"`
	SuggestedAction string            `json:"suggested_action,omitempty"`
	Reason          *R                `json:"reason,omitempty"`
	RepairHints     []H               `json:"repair_hints,omitempty"`
	Taxonomy        ReadinessTaxonomy `json:"taxonomy,omitempty"`
}

type ReadinessCheck[R any, H any] struct {
	Key             string            `json:"key,omitempty"`
	Label           string            `json:"label,omitempty"`
	Status          string            `json:"status,omitempty"`
	Message         string            `json:"message,omitempty"`
	FieldPaths      []string          `json:"field_paths,omitempty"`
	SuggestedAction string            `json:"suggested_action,omitempty"`
	Reason          *R                `json:"reason,omitempty"`
	RepairHints     []H               `json:"repair_hints,omitempty"`
	Taxonomy        ReadinessTaxonomy `json:"taxonomy,omitempty"`
}

func BuildSubmitReadiness[R any, H any](
	checks []ReadinessCheckSpec,
	guidanceResolver func(ReadinessCheckSpec) Guidance[R, H],
	blockedSummary string,
	warningSummary string,
	readySummary string,
) *SubmitReadiness[R, H] {
	if len(checks) == 0 {
		return nil
	}

	readiness := &SubmitReadiness[R, H]{}
	var blockers []ReadinessItem[R, H]
	var warnings []ReadinessItem[R, H]
	var resolvedChecks []ReadinessCheck[R, H]

	for _, spec := range checks {
		guidance := guidanceResolver(spec)
		status := "ready"
		if !spec.OK && spec.WarningOnly {
			status = "warning"
		}
		if !spec.OK && !spec.WarningOnly {
			status = "blocking"
		}

		check := ReadinessCheck[R, H]{
			Key:             spec.Key,
			Label:           spec.Label,
			Status:          status,
			Message:         spec.Message,
			FieldPaths:      append([]string(nil), spec.FieldPaths...),
			SuggestedAction: spec.SuggestedAction,
			Reason:          guidance.Reason,
			RepairHints:     append([]H(nil), guidance.RepairHints...),
			Taxonomy:        spec.Taxonomy,
		}
		resolvedChecks = append(resolvedChecks, check)
		if spec.OK {
			continue
		}
		item := ReadinessItem[R, H]{
			Key:             spec.Key,
			Label:           spec.Label,
			Message:         spec.Message,
			FieldPaths:      append([]string(nil), spec.FieldPaths...),
			SuggestedAction: spec.SuggestedAction,
			Reason:          guidance.Reason,
			RepairHints:     append([]H(nil), guidance.RepairHints...),
			Taxonomy:        spec.Taxonomy,
		}
		if spec.WarningOnly {
			warnings = append(warnings, item)
		} else {
			blockers = append(blockers, item)
		}
	}

	readiness.Checks = resolvedChecks
	readiness.BlockingItems = blockers
	readiness.WarningItems = warnings
	readiness.Ready = len(blockers) == 0
	switch {
	case len(blockers) > 0:
		readiness.Status = "blocked"
		readiness.Summary = append(readiness.Summary, blockedSummary)
	case len(warnings) > 0:
		readiness.Status = "ready_with_warnings"
		readiness.Summary = append(readiness.Summary, warningSummary)
	default:
		readiness.Status = "ready"
		readiness.Summary = append(readiness.Summary, readySummary)
	}
	return readiness
}
