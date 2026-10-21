package sheinlogin

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestProfileUsesCurrentScopeAndRejectsRelativeRoot(t *testing.T) {
	scope := fixtureAttempt().Scope
	if _, e := resolveProfileDir("relative", scope); e == nil {
		t.Fatal("relative root accepted")
	}
	root := t.TempDir()
	dir, e := resolveProfileDir(root, scope)
	if e != nil {
		t.Fatal(e)
	}
	if !filepath.IsAbs(dir) || filepath.Base(dir) != scope.BindingRevision || filepath.Base(filepath.Dir(dir)) != scope.StoreID {
		t.Fatal(dir)
	}
	scope.OrganizationID = "org-b"
	other, e := resolveProfileDir(root, scope)
	if e != nil || other == dir {
		t.Fatal("organizations share profile")
	}
	if _, e = os.Stat(filepath.Join(root, "SingletonLock")); !os.IsNotExist(e) {
		t.Fatal("profile resolver touched files")
	}
}

func TestIsProfileInUseError(t *testing.T) {
	cases := []string{
		"profile appears to be in use",
		"ProcessSingleton already held",
		"profile directory is locked",
		"SingletonLock exists",
	}
	for _, message := range cases {
		if !isProfileInUseError(errors.New(message)) {
			t.Fatalf("expected profile-in-use for %q", message)
		}
	}
	if isProfileInUseError(errors.New("other error")) {
		t.Fatal("did not expect unrelated error to match profile-in-use")
	}
}
