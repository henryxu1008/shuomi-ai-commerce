package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStoreObservationsRetainedInitializerIdentityAndRepeat(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX initializer execution; exercised in Linux")
	}
	if _, e := exec.LookPath("jq"); e != nil {
		t.Fatal("jq required")
	}
	for _, variant := range []string{"repeat", "changed-base", "changed-output", "revoked-grants", "interrupted", "configured-app", "wrong-store-host", "wrong-store-port"} {
		t.Run(variant, func(t *testing.T) {
			root := t.TempDir()
			for _, dir := range []string{"base", "runtime", "secrets/store-owner", "bin"} {
				if e := os.MkdirAll(filepath.Join(root, dir), 0700); e != nil {
					t.Fatal(e)
				}
			}
			base := `{"storeCenter":{"enabled":true,"database":{"host":"127.0.0.1","port":5433,"user":"store_center_runtime","database":"store_center"},"officialApplications":[]},"identity":{"tenantDirectoryToken":"synthetic"},"dataServices":{"temporalAddress":"127.0.0.1:7233","temporalNamespace":"default"},"aiWorkbench":{"conversationOnly":true},"ecoservices":{"nonPaymentOnly":true}}`
			write := func(name, data string) {
				t.Helper()
				if e := os.WriteFile(filepath.Join(root, name), []byte(data), 0600); e != nil {
					t.Fatal(e)
				}
			}
			write("base/current-application.json", base)
			write("secrets/store-owner/store-owner-password", "synthetic-test-password")
			calls := filepath.Join(root, "owner-calls")
			writeAccountComposeExecutable(t, filepath.Join(root, "bin", "store-center-schema-init"), "#!/bin/sh\ntest \"$1\" = -config || exit 97\nif test \"$3\" = -observations-verify; then test ! -f "+shellQuote(filepath.Join(root, "revoked"))+"; exit $?; fi\ntest \"$3\" = -observations-only || exit 97\nprintf 'called\\n' >> "+shellQuote(calls)+"\n")
			input, e := os.ReadFile(filepath.Join("..", "deployments", "docker", "account-compose", "store-observations-init.sh"))
			if e != nil {
				t.Fatal(e)
			}
			script := strings.NewReplacer("/base/", filepath.Join(root, "base")+"/", "/runtime/", filepath.Join(root, "runtime")+"/", "/secrets/store-owner/", filepath.Join(root, "secrets/store-owner")+"/").Replace(string(input))
			write("installer.sh", script)
			run := func() error {
				command := exec.Command("sh", filepath.Join(root, "installer.sh"))
				command.Env = append(os.Environ(), "PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
				out, e := command.CombinedOutput()
				if e != nil {
					t.Logf("guard: %s", out)
				}
				return e
			}
			if variant == "interrupted" {
				if e := os.Mkdir(filepath.Join(root, "runtime/.initializing"), 0700); e != nil {
					t.Fatal(e)
				}
			}
			if variant == "configured-app" {
				write("base/current-application.json", strings.Replace(base, `"officialApplications":[]`, `"officialApplications":[{}]`, 1))
			}
			if variant == "wrong-store-host" {
				write("base/current-application.json", strings.Replace(base, `"host":"127.0.0.1"`, `"host":"127.0.0.2"`, 1))
			}
			if variant == "wrong-store-port" {
				write("base/current-application.json", strings.Replace(base, `"port":5433`, `"port":5434`, 1))
			}
			if variant == "interrupted" || variant == "configured-app" || variant == "wrong-store-host" || variant == "wrong-store-port" {
				if run() == nil {
					t.Fatal("unsafe admission accepted")
				}
				if _, e := os.Stat(calls); e == nil {
					t.Fatal("owner invoked before admission")
				}
				return
			}
			if e := run(); e != nil {
				t.Fatal(e)
			}
			before, e := os.ReadFile(filepath.Join(root, "runtime/current-application.json"))
			if e != nil {
				t.Fatal(e)
			}
			if !strings.Contains(string(before), `"conversationOnly": true`) || !strings.Contains(string(before), `"nonPaymentOnly": true`) {
				t.Fatal("unrelated execution flags changed")
			}
			if variant == "changed-base" {
				write("base/current-application.json", base+"\n")
			}
			if variant == "changed-output" {
				write("runtime/current-application.json", string(before)+"\n")
			}
			if variant == "revoked-grants" {
				write("revoked", "revoked")
			}
			err := run()
			if (err == nil) != (variant == "repeat") {
				t.Fatalf("repeat guard: %v", err)
			}
			got, e := os.ReadFile(calls)
			if e != nil {
				t.Fatal(e)
			}
			if string(got) != "called\n" {
				t.Fatal("retained initializer provisioned again")
			}
		})
	}
}
