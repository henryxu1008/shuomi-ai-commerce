package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestComposeTofuPreflightFailureCanRetrySameProject(t *testing.T) {
	for _, profile := range []string{"account", "acquisition", "referrals"} {
		for _, failure := range []string{"apk", "init"} {
			t.Run(profile+"/"+failure, func(t *testing.T) {
				f := newComposeTofuFixture(t, profile)
				output, err := f.run(failure, "")
				if err == nil {
					t.Fatalf("preparation unexpectedly succeeded: %s", output)
				}
				f.requireMarker(t, ".terraform-started", false)
				f.requireMarker(t, ".terraform-complete", false)
				if strings.Contains(f.calls(t), "apply ") {
					t.Fatal("preparation failure reached native apply")
				}
				if failure == "init" {
					if _, err := os.Stat(filepath.Join(f.state, "config", ".terraform", "prepared")); err != nil {
						t.Fatal("expected partial provider preparation to remain for retry", err)
					}
				}
				// Deliberately reuse every directory/file, with no reset or marker cleanup.
				if output, err := f.run("", ""); err != nil {
					t.Fatalf("same-project retry failed: %v, %s", err, output)
				}
				f.requireMarker(t, ".terraform-started", false)
				f.requireMarker(t, ".terraform-complete", true)
				wantApplies := 1
				if profile == "referrals" {
					wantApplies = 2 // Existing main and probe operations.
				}
				if got := strings.Count(f.calls(t), "apply "); got != wantApplies {
					t.Fatalf("apply count=%d, want %d", got, wantApplies)
				}
				before := f.calls(t)
				if output, err := f.run("apk", ""); err != nil {
					t.Fatalf("completed project failed on replay: %v, %s", err, output)
				}
				if f.calls(t) != before {
					t.Fatal("completed project reran preparation or native operations")
				}
			})
		}
	}
}

func TestComposeTofuMutationFailureStillRefusesReplay(t *testing.T) {
	for _, profile := range []string{"account", "acquisition", "referrals"} {
		failures := []string{"apply", "output"}
		if profile == "referrals" {
			failures = append(failures, "probe-init")
		}
		for _, failure := range failures {
			t.Run(profile+"/"+failure, func(t *testing.T) {
				f := newComposeTofuFixture(t, profile)
				output, err := f.run(failure, "")
				if err == nil {
					t.Fatalf("failure unexpectedly succeeded: %s", output)
				}
				f.requireMarker(t, ".terraform-started", true)
				f.requireMarker(t, ".terraform-complete", false)
				stateFile := filepath.Join(f.state, "terraform.tfstate")
				if profile == "referrals" {
					stateFile = filepath.Join(f.state, "main", "terraform.tfstate")
				}
				beforeState, err := os.ReadFile(stateFile)
				if err != nil {
					t.Fatalf("expected failed/unknown apply state: %v, output=%s, calls=%s", err, output, f.calls(t))
				}
				beforeCalls := f.calls(t)
				output, err = f.run("", "")
				if err == nil || !strings.Contains(string(output), "initialization is incomplete") {
					t.Fatalf("incomplete project did not refuse replay: %v, %s", err, output)
				}
				if f.calls(t) != beforeCalls {
					t.Fatal("incomplete project invoked preparation or native operations again")
				}
				afterState, err := os.ReadFile(stateFile)
				if err != nil || string(afterState) != string(beforeState) {
					t.Fatal("incomplete replay modified the original state")
				}
			})
		}
	}
}

func TestComposeTofuPreflightRechecksStateBeforeApply(t *testing.T) {
	for _, profile := range []string{"account", "acquisition", "referrals"} {
		for _, marker := range []string{".terraform-started", ".terraform-complete"} {
			t.Run(profile+"/"+marker, func(t *testing.T) {
				f := newComposeTofuFixture(t, profile)
				output, err := f.run("", marker)
				if err == nil {
					t.Fatalf("state changed during preparation was accepted: %s", output)
				}
				if strings.Contains(f.calls(t), "apply ") {
					t.Fatal("state changed during preparation reached native apply")
				}
				got, err := os.ReadFile(filepath.Join(f.state, marker))
				if err != nil || string(got) != "other-initializer\n" {
					t.Fatalf("preparation changed state: %v, output=%s, calls=%s", err, output, f.calls(t))
				}
			})
		}
	}
}

type composeTofuFixture struct {
	state, script, bin, pat, log, sh string
}

func newComposeTofuFixture(t *testing.T, profile string) composeTofuFixture {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX filesystem/permissions required; executed on Linux CI")
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("POSIX shell required")
	}
	root := t.TempDir()
	f := composeTofuFixture{state: filepath.Join(root, "state"), script: filepath.Join(root, "init.sh"), bin: filepath.Join(root, "bin"), pat: filepath.Join(root, "pat"), log: filepath.Join(root, "calls"), sh: sh}
	ca, inputs := filepath.Join(root, "ca"), filepath.Join(root, "inputs")
	runtimeDir, frontend := filepath.Join(root, "runtime"), filepath.Join(root, "frontend")
	source := filepath.Join(root, "terraform-source")
	for _, dir := range []string{f.state, f.bin, ca, inputs, runtimeDir, frontend, filepath.Join(source, "probe")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{f.pat, filepath.Join(ca, "root-ca.pem"), filepath.Join(inputs, "operator-password"), filepath.Join(inputs, "viewer-password"), filepath.Join(inputs, "insufficient-password"), filepath.Join(source, "main.tf")} {
		if err := os.WriteFile(path, []byte("fixture-not-a-real-credential\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeAccountComposeExecutable(t, filepath.Join(f.bin, "apk"), "#!/bin/sh\nprintf 'apk\\n' >> \"$TOFU_FIXTURE_LOG\"\n[ \"$TOFU_FIXTURE_FAIL\" != apk ]\n")
	writeAccountComposeExecutable(t, filepath.Join(f.bin, "curl"), "#!/bin/sh\nprintf 'discovery\\n' >> \"$TOFU_FIXTURE_LOG\"\n")
	writeAccountComposeExecutable(t, filepath.Join(f.bin, "tofu"), `#!/bin/sh
set -eu
stage=$(basename "$PWD")
printf '%s %s\n' "$1" "$stage" >> "$TOFU_FIXTURE_LOG"
case "$1" in
 init)
  mkdir -p .terraform
  printf 'partial-cache\n' > .terraform/prepared
  if [ "$stage" = probe ]; then
   [ "$TOFU_FIXTURE_FAIL" != probe-init ] || exit 42
  else
   [ "$TOFU_FIXTURE_FAIL" != init ] || exit 42
   if [ -n "$TOFU_FIXTURE_LATE_MARKER" ]; then
    printf 'other-initializer\n' > "$TOFU_FIXTURE_STATE/$TOFU_FIXTURE_LATE_MARKER"
   fi
  fi ;;
 apply|destroy)
  test -f "$TOFU_FIXTURE_STATE/.terraform-started" || exit 98
  for arg do
   case "$arg" in -state=*) printf 'fixture-state\n' > "${arg#-state=}" ;; esac
  done
  [ "$TOFU_FIXTURE_FAIL" != apply ] || exit 42 ;;
 output)
  [ "$TOFU_FIXTURE_FAIL" != output ] || exit 42
  printf 'fixture-output\n' ;;
 *) exit 99 ;;
esac
`)
	script, err := os.ReadFile(filepath.Join("..", "deployments", "docker", profile+"-compose", "tofu-init.sh"))
	if err != nil {
		t.Fatal(err)
	}
	origin, err := filepath.Abs(filepath.Join("..", "deployments", "docker", "account-compose", "deployment-origins.sh"))
	if err != nil {
		t.Fatal(err)
	}
	replacer := strings.NewReplacer(
		". /usr/local/lib/account-compose/deployment-origins.sh", ". "+shellQuote(origin),
		"state=/state", "state="+shellQuote(f.state),
		"terraform_source=/terraform-source", "terraform_source="+shellQuote(source),
		"source=/terraform-source", "source="+shellQuote(source),
		"trusted_ca=/private/ca", "trusted_ca="+shellQuote(ca),
		"ca=/private/ca", "ca="+shellQuote(ca),
		"tofu_inputs=/tofu-inputs", "tofu_inputs="+shellQuote(inputs),
		"inputs=/tofu-inputs", "inputs="+shellQuote(inputs),
		"runtime=/runtime", "runtime="+shellQuote(runtimeDir),
		"frontend=/frontend", "frontend="+shellQuote(frontend),
	)
	writeAccountComposeExecutable(t, f.script, replacer.Replace(string(script)))
	return f
}

func (f composeTofuFixture) run(failure, lateMarker string) ([]byte, error) {
	cmd := exec.Command(f.sh, f.script)
	cmd.Env = append(os.Environ(), "PATH="+f.bin+":"+os.Getenv("PATH"),
		"BOOTSTRAP_PAT_FILE="+f.pat, "ACCOUNT_IDENTITY_PORT=8443", "ACCOUNT_APPLICATION_PORT=8080",
		"ACCOUNT_PUBLIC_TRIAL=", "ACCOUNT_PUBLIC_APPLICATION_ORIGIN=", "ACCOUNT_PUBLIC_IDENTITY_ORIGIN=",
		"TOFU_FIXTURE_STATE="+f.state, "TOFU_FIXTURE_LOG="+f.log,
		"TOFU_FIXTURE_FAIL="+failure, "TOFU_FIXTURE_LATE_MARKER="+lateMarker)
	return cmd.CombinedOutput()
}

func (f composeTofuFixture) calls(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(f.log)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(data)
}

func (f composeTofuFixture) requireMarker(t *testing.T, marker string, present bool) {
	t.Helper()
	_, err := os.Stat(filepath.Join(f.state, marker))
	if present && err != nil || !present && !os.IsNotExist(err) {
		t.Fatalf("marker %s: present=%v, stat=%v", marker, present, err)
	}
}
