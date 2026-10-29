package tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestAccountComposeLocalTLSPublicationBoundary(t *testing.T) {
	dir := filepath.Join("..", "deployments", "docker", "account-compose")
	readYAML := func(name string) map[string]any {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		var value map[string]any
		if err := yaml.Unmarshal(data, &value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	fragment := readYAML("traefik-local-tls-crl.fragment.yml")
	if len(fragment) != 1 {
		t.Fatal("fragment must not replace TLS configuration")
	}
	http := fragment["http"].(map[string]any)
	if len(http) != 2 {
		t.Fatal("fragment must reuse existing service and entrypoint")
	}
	routers := http["routers"].(map[string]any)
	if len(routers) != 1 {
		t.Fatal("only one public CRL router is allowed")
	}
	router := routers["local-tls-crl"].(map[string]any)
	if router["rule"] != "Host(`localhost`) && (Method(`GET`) || Method(`HEAD`)) && Path(`/.well-known/local-tls/root.crl`)" || router["service"] != "listingkit-ui" {
		t.Fatal("public route must have exact localhost/method/path scope and existing UI target")
	}
	if _, ok := router["tls"]; ok {
		t.Fatal("CRL retrieval must not depend on the target TLS chain")
	}
	if fmt.Sprint(router["entryPoints"]) != "[application]" || fmt.Sprint(router["middlewares"]) != "[local-tls-crl-headers]" || len(router) != 5 {
		t.Fatal("unexpected route behavior or entrypoint")
	}
	middlewares := http["middlewares"].(map[string]any)
	if len(middlewares) != 1 {
		t.Fatal("no authentication/redirect/retry middleware may be added")
	}
	headers := middlewares["local-tls-crl-headers"].(map[string]any)["headers"].(map[string]any)["customResponseHeaders"].(map[string]any)
	if headers["Content-Type"] != "application/pkix-crl" || headers["X-Content-Type-Options"] != "nosniff" {
		t.Fatal("CRL response metadata")
	}
	override := readYAML("docker-compose.local-tls-crl.yml")
	if len(override) != 1 {
		t.Fatal("override must not add networks, volumes or configuration")
	}
	services := override["services"].(map[string]any)
	if len(services) != 1 {
		t.Fatal("override must not add services")
	}
	ui := services["listingkit-ui"].(map[string]any)
	if len(ui) != 1 {
		t.Fatal("only the public-artifact mount may change")
	}
	mounts := ui["volumes"].([]any)
	if len(mounts) != 1 {
		t.Fatal("only one public directory mount is allowed")
	}
	mount := mounts[0].(map[string]any)
	if mount["target"] != "/app/public/.well-known/local-tls" || mount["read_only"] != true || mount["type"] != "bind" || mount["source"] != "${LOCAL_TLS_PUBLIC_DIR:?set the explicitly approved public CRL directory}" || mount["bind"].(map[string]any)["create_host_path"] != false {
		t.Fatal("public mount must be explicit, readonly and never create the host path")
	}
}

func TestAccountComposeLocalTLSEntry(t *testing.T) {
	path := filepath.Join("..", "deployments", "docker", "account-compose", "local-tls-maintain.sh")
	if _, err := os.ReadFile(path); err != nil {
		t.Fatalf("independent TLS maintenance entry is unavailable: %v", err)
	}
	sh, err := exec.LookPath("sh")
	if err != nil && runtime.GOOS == "windows" {
		sh = `C:\Program Files\Git\bin\sh.exe`
	}
	if sh == "" {
		t.Skip("shell syntax check requires sh")
	}
	if output, err := exec.Command(sh, "-n", path).CombinedOutput(); err != nil {
		t.Fatalf("shell syntax: %v\n%s", err, output)
	}
}

// OpenSSL is replaced with a local text-only command fixture. These tests do
// not create keys, certificates or CRLs and do not prove cryptographic/TLS acceptance.
func TestAccountComposeLocalTLSOffline(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("private-state permissions and actual shell orchestration require Linux; CI runs this test")
	}
	t.Run("isolated issuance with CDP and native verification", func(t *testing.T) {
		f := newLocalTLSFixture(t)
		f.ok("init", f.state, "35444")
		f.ok("issue", f.state)
		config := f.read("openssl.cnf")
		for _, want := range []string{"CA:true,pathlen:0", "keyCertSign,cRLSign", "CA:false", "serverAuth", "unique_subject = no", "DNS:localhost,IP:127.0.0.1,IP:::1", "http://localhost:35444/.well-known/local-tls/root.crl"} {
			if !strings.Contains(config, want) {
				t.Errorf("issued certificate policy is missing %q", want)
			}
		}
		log := f.calls()
		for _, want := range []string{"ca -config", "-gencrl", "-verify -CAfile", "-crl_check", "-verify_hostname localhost", "-verify_ip 127.0.0.1", "-verify_ip ::1"} {
			if !strings.Contains(log, want) {
				t.Errorf("native verification boundary missing %q", want)
			}
		}
		for _, path := range []string{"private/root-ca.key", "releases/1000/localhost.key"} {
			info, err := os.Stat(filepath.Join(f.state, path))
			if err != nil || info.Mode().Perm() != 0o600 {
				t.Fatalf("private material permissions for %s: %v", path, err)
			}
		}
		entries, err := os.ReadDir(filepath.Join(f.state, "public"))
		if err != nil || len(entries) != 2 {
			t.Fatalf("public artifacts: %v / %v", entries, err)
		}
		for _, entry := range entries {
			if entry.Name() != "root-ca.pem" && entry.Name() != "root.crl" {
				t.Fatalf("unexpected published artifact %s", entry.Name())
			}
		}
		f.mustExist("releases/1000/complete")
		f.mustNotExist(".operation")
		old := f.read("releases/1000/localhost.crt")
		f.ok("issue", f.state)
		f.mustExist("releases/1001/complete")
		if f.read("releases/1000/localhost.crt") != old || strings.Contains(f.calls(), "-revoke") {
			t.Fatal("renewal must preserve the prior release without revoking it")
		}
		before := f.calls()
		f.bad("init", f.state, "35444")
		if f.calls() != before {
			t.Fatal("existing state must not issue a second root")
		}
	})
	t.Run("each issuance failure locks state without automatic retry", func(t *testing.T) {
		for step := 1; step <= 6; step++ {
			t.Run(strconv.Itoa(step), func(t *testing.T) {
				f := newLocalTLSFixture(t)
				f.ok("init", f.state, "35444")
				f.resetCalls()
				f.env = append(f.env, fmt.Sprintf("TLS_FIXTURE_FAIL_AT=%d", step))
				f.bad("issue", f.state)
				if got := len(strings.FieldsFunc(strings.TrimSpace(f.calls()), func(r rune) bool { return r == '\n' })); got != step {
					t.Fatalf("calls=%d, expected exactly %d and no retry", got, step)
				}
				f.mustExist(".operation/result")
				f.mustNotExist("releases/1000/complete")
				before := f.calls()
				f.bad("refresh-crl", f.state)
				if f.calls() != before {
					t.Fatal("unknown prior outcome must stop later operations")
				}
			})
		}
	})
	t.Run("failed conversion preserves published CRL and logs", func(t *testing.T) {
		f := newLocalTLSFixture(t)
		f.ok("init", f.state, "35444")
		before := f.read("public/root.crl")
		f.resetCalls()
		f.env = append(f.env, "TLS_FIXTURE_FAIL_AT=4")
		f.bad("refresh-crl", f.state)
		if f.read("public/root.crl") != before {
			t.Fatal("partial CRL must not replace published CRL")
		}
		if !strings.Contains(f.read(".operation/4.stderr"), "fixture failure") || !strings.Contains(f.read(".operation/result"), "77") {
			t.Fatal("native stderr and exit status were not retained")
		}
	})
	t.Run("zero exit is not proof of CRL signature success", func(t *testing.T) {
		for _, diagnostic := range []string{"verify failure", "", "unrecognized success"} {
			t.Run(diagnostic, func(t *testing.T) {
				f := newLocalTLSFixture(t)
				f.ok("init", f.state, "35444")
				before := f.read("public/root.crl")
				f.resetCalls()
				f.env = append(f.env, "TLS_FIXTURE_CRL_VERIFY="+diagnostic)
				f.bad("refresh-crl", f.state)
				if f.read("public/root.crl") != before || strings.Contains(f.calls(), "-outform DER") {
					t.Fatal("unverified CRL reached publication")
				}
				f.mustExist(".operation/result")
			})
		}
	})
	t.Run("revoke is limited to a known serial", func(t *testing.T) {
		f := newLocalTLSFixture(t)
		f.ok("init", f.state, "35444")
		f.ok("issue", f.state)
		f.resetCalls()
		for _, serial := range []string{"../private/root-ca.key", "1000\n", "FFFF"} {
			f.bad("revoke", f.state, serial)
		}
		if f.calls() != "" {
			t.Fatal("invalid or unknown serial reached OpenSSL")
		}
		f.ok("revoke", f.state, "1000")
		if !strings.Contains(f.calls(), "-revoke "+f.state+"/issued/1000.pem") || !strings.Contains(f.calls(), "-gencrl") {
			t.Fatal("revocation did not use native CA state and refresh the CRL")
		}
	})
	t.Run("unsafe state is rejected before signing", func(t *testing.T) {
		for _, unsafe := range []string{"permission", "symlink"} {
			t.Run(unsafe, func(t *testing.T) {
				f := newLocalTLSFixture(t)
				f.ok("init", f.state, "35444")
				if unsafe == "permission" {
					if err := os.Chmod(f.state, 0o755); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := os.Remove(filepath.Join(f.state, "public/root.crl")); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(filepath.Join(f.state, "private/root-ca.key"), filepath.Join(f.state, "public/root.crl")); err != nil {
						t.Fatal(err)
					}
				}
				f.resetCalls()
				f.bad("issue", f.state)
				if f.calls() != "" {
					t.Fatal("unsafe state reached OpenSSL")
				}
			})
		}
	})
	t.Run("invalid init has no signing effects", func(t *testing.T) {
		f := newLocalTLSFixture(t)
		for _, port := range []string{"0", "65536", "80\n", "35444/path", "035444"} {
			f.bad("init", f.state, port)
		}
		f.bad("init", "relative-state", "35444")
		if f.calls() != "" {
			t.Fatal("invalid init reached OpenSSL")
		}
		if _, err := os.Stat(f.state); !os.IsNotExist(err) {
			t.Fatal("invalid init created state")
		}
	})
}

type localTLSFixture struct {
	t                  *testing.T
	state, script, log string
	env                []string
}

func newLocalTLSFixture(t *testing.T) *localTLSFixture {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	f := &localTLSFixture{t: t, state: filepath.Join(dir, "state"), log: filepath.Join(dir, "calls")}
	var err error
	f.script, err = filepath.Abs(filepath.Join("..", "deployments", "docker", "account-compose", "local-tls-maintain.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "openssl"), []byte(localTLSFakeOpenSSL), 0o700); err != nil {
		t.Fatal(err)
	}
	f.env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "TLS_FIXTURE_LOG="+f.log)
	return f
}

func (f *localTLSFixture) run(args ...string) ([]byte, error) {
	f.t.Helper()
	c := exec.Command("sh", append([]string{f.script}, args...)...)
	c.Env = f.env
	return c.CombinedOutput()
}
func (f *localTLSFixture) ok(args ...string) {
	f.t.Helper()
	if out, err := f.run(args...); err != nil {
		f.t.Fatalf("%s: %v\n%s", args[0], err, out)
	}
}
func (f *localTLSFixture) bad(args ...string) {
	f.t.Helper()
	if out, err := f.run(args...); err == nil {
		f.t.Fatalf("%s unexpectedly succeeded: %s", args[0], out)
	}
}
func (f *localTLSFixture) calls() string { data, _ := os.ReadFile(f.log); return string(data) }
func (f *localTLSFixture) resetCalls() {
	f.t.Helper()
	if err := os.WriteFile(f.log, nil, 0o600); err != nil {
		f.t.Fatal(err)
	}
}
func (f *localTLSFixture) read(path string) string {
	f.t.Helper()
	data, err := os.ReadFile(filepath.Join(f.state, path))
	if err != nil {
		f.t.Fatal(err)
	}
	return string(data)
}
func (f *localTLSFixture) mustExist(path string) {
	f.t.Helper()
	if _, err := os.Stat(filepath.Join(f.state, path)); err != nil {
		f.t.Fatal(err)
	}
}
func (f *localTLSFixture) mustNotExist(path string) {
	f.t.Helper()
	if _, err := os.Stat(filepath.Join(f.state, path)); !os.IsNotExist(err) {
		f.t.Fatalf("unexpected path %s", path)
	}
}

const localTLSFakeOpenSSL = `#!/bin/sh
set -eu
printf '%s\n' "$*" >> "$TLS_FIXTURE_LOG"
step=$(wc -l < "$TLS_FIXTURE_LOG" | tr -d ' ')
if [ "$step" = "${TLS_FIXTURE_FAIL_AT:-none}" ]; then
  echo 'fixture failure' >&2
  exit 77
fi
verb=$1
shift
out=''; key=''; sign=no; verify=no; previous=''
for arg in "$@"; do
  case "$previous" in -out) out=$arg;; -keyout) key=$arg;; esac
  [ "$arg" != '-in' ] || sign=yes
  [ "$arg" != '-verify' ] || verify=yes
  previous=$arg
done
if [ "$verb" = crl ] && [ "$verify" = yes ]; then
  printf '%s\n' "${TLS_FIXTURE_CRL_VERIFY-verify OK}" >&2
  exit 0
fi
[ -z "$key" ] || printf '%s\n' NOT_A_PRIVATE_KEY > "$key"
[ -z "$out" ] || printf '%s\n' NOT_A_CRYPTOGRAPHIC_ARTIFACT > "$out"
if [ "$verb" = ca ] && [ "$sign" = yes ]; then
  serial=$(cat "$LOCAL_TLS_STATE/serial")
  printf '%s\n' NOT_A_CERTIFICATE > "$LOCAL_TLS_STATE/issued/$serial.pem"
  printf '%04X\n' "$((0x$serial + 1))" > "$LOCAL_TLS_STATE/serial"
fi
echo 'fixture stdout'
echo 'fixture stderr' >&2
`
