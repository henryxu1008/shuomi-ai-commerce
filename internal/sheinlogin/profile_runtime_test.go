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

func TestForcedProfileIsFreshAndDoesNotAlterExistingSession(t *testing.T) {
	root := t.TempDir()
	scope := fixtureAttempt().Scope
	privateFixture(t, filepath.Join(root, ".shein-login-owner"), []byte("shein-login:v1:"+scope.InstallationID))
	a := Account{Scope: scope, Generation: scope.Generation}
	old, err := prepareProfile(root, a)
	if err != nil {
		t.Fatal(err)
	}
	oldSession := filepath.Join(old, "synthetic-authenticated-session")
	if err = os.WriteFile(oldSession, []byte("retained synthetic session"), 0600); err != nil {
		t.Fatal(err)
	}
	id := fixtureAttempt().ID
	fresh, err := prepareAutomationProfile(root, a, id)
	if err != nil || fresh == old {
		t.Fatal("force profile was not isolated", fresh, err)
	}
	if _, err = os.Stat(filepath.Join(fresh, "synthetic-authenticated-session")); !os.IsNotExist(err) {
		t.Fatal("old session was copied")
	}
	if data, err := os.ReadFile(oldSession); err != nil || string(data) != "retained synthetic session" {
		t.Fatal("old session was changed")
	}
	if _, err = prepareAutomationProfile(root, a, id); err == nil {
		t.Fatal("prior forced attempt profile was silently adopted")
	}
}
