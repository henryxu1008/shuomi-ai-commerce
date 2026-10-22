package bootstrap

import "testing"

func TestBuildServiceRejectsIncompleteCurrentDependencies(t *testing.T) {
	if _, err := BuildService(BuildInput{}); err == nil {
		t.Fatal("missing explicit current dependencies admitted")
	}
}
