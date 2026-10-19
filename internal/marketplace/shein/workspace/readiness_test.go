package workspace

import "testing"

func TestBuildSubmitReadinessPreservesBlockingAction(t *testing.T) {
	readiness := BuildSubmitReadiness(
		[]ReadinessCheckSpec{
			{Key: "category", Label: "类目", OK: false, SuggestedAction: "确认类目"},
			{Key: "request_draft", Label: "草稿", OK: true},
		},
		func(spec ReadinessCheckSpec) Guidance[string, string] { return Guidance[string, string]{} },
		"blocked",
		"warnings",
		"ready",
	)

	if readiness == nil || readiness.Status != "blocked" || readiness.Ready {
		t.Fatalf("readiness = %#v", readiness)
	}
	if len(readiness.BlockingItems) != 1 || readiness.BlockingItems[0].Key != "category" || readiness.BlockingItems[0].SuggestedAction != "确认类目" {
		t.Fatalf("blocking items = %#v", readiness.BlockingItems)
	}
}

func TestBuildSubmitReadinessCarriesBlockerTaxonomy(t *testing.T) {
	taxonomy := ReadinessTaxonomy{
		BlockerKey:          "missing_category",
		Severity:            "blocker",
		Domain:              "category",
		RepairTarget:        "category_review",
		RepairRoute:         "workspace.category",
		Recoverable:         true,
		RequiresEngineering: false,
	}
	readiness := BuildSubmitReadiness(
		[]ReadinessCheckSpec{
			{
				Key:      "category",
				Label:    "类目",
				OK:       false,
				Taxonomy: taxonomy,
			},
		},
		func(spec ReadinessCheckSpec) Guidance[string, string] { return Guidance[string, string]{} },
		"blocked",
		"warnings",
		"ready",
	)

	if readiness == nil || len(readiness.BlockingItems) != 1 || len(readiness.Checks) != 1 {
		t.Fatalf("readiness = %#v, want one blocking item and one check", readiness)
	}
	if readiness.BlockingItems[0].Taxonomy != taxonomy {
		t.Fatalf("blocking taxonomy = %#v, want %#v", readiness.BlockingItems[0].Taxonomy, taxonomy)
	}
	if readiness.Checks[0].Taxonomy != taxonomy {
		t.Fatalf("check taxonomy = %#v, want %#v", readiness.Checks[0].Taxonomy, taxonomy)
	}
}
