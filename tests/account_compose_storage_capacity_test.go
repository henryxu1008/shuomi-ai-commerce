package tests

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const knowledgeCapacityPin = "0566e59b-dfaf-4566-8881-eeaef94e4089"
const ecoservicesCapacityPin = "a9b7b02d-5836-4928-97a8-a4efdf52dbf3"

func TestCapacityComposePreservesFinalModuleManifest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX deployment fixture; also run in the isolated schema container")
	}
	for _, chat := range []bool{false, true} {
		t.Run(map[bool]string{false: "ecoservices", true: "chat"}[chat], func(t *testing.T) {
			root := t.TempDir()
			for _, dir := range []string{"base", "runtime", "knowledge-identity", "ecoservices-identity"} {
				if err := os.Mkdir(filepath.Join(root, dir), 0700); err != nil {
					t.Fatal(err)
				}
			}
			base := map[string]any{
				"identity":                map[string]any{"issuer": "https://auth.example.test", "privateToken": "fixture-token"},
				"commercialOwnerDatabase": map[string]any{"database": "commercial", "password": "fixture-password"},
				"moneyOwnerDatabase":      map[string]any{"database": "referrals"},
				"knowledge":               map[string]any{"enabled": true, "storage": map[string]any{"bucket": "knowledge", "endpoint": "http://knowledge-objects:9000", "secretAccessKey": "fixture-knowledge"}},
				"ecoservices":             map[string]any{"enabled": true, "nonPaymentOnly": true, "storage": map[string]any{"bucket": "ecoservices", "endpoint": "http://127.0.0.1:9101", "secretAccessKey": "fixture-ecoservices"}},
				"productCollections":      true, "dataServices": map[string]any{"enabled": true},
				"projectCenter": map[string]any{"database": map[string]any{"database": "ai_projects"}},
			}
			if chat {
				base["aiWorkbench"] = map[string]any{"enabled": true, "conversationOnly": true, "allowedOrganizationIds": []string{"org-fixture"}}
			}
			writeCapacityJSON(t, filepath.Join(root, "base/current-application.json"), base)
			for name, pin := range map[string]string{"knowledge-identity": knowledgeCapacityPin, "ecoservices-identity": ecoservicesCapacityPin} {
				if err := os.WriteFile(filepath.Join(root, name, "deployment-id"), []byte(pin+"\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			script := capacityFixtureScript(t, root, "storage-capacity-init.sh", []string{"base", "runtime", "knowledge-identity", "ecoservices-identity"})
			run := func(ok bool) []byte {
				t.Helper()
				out, err := exec.Command("sh", script).CombinedOutput()
				if (err == nil) != ok {
					t.Fatalf("capacity assembly success=%v: %v %s", ok, err, out)
				}
				if strings.Contains(string(out), "fixture-token") || strings.Contains(string(out), "fixture-password") {
					t.Fatal("private manifest leaked")
				}
				return out
			}
			run(true)
			output := filepath.Join(root, "runtime/current-application.json")
			first, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := json.Unmarshal(first, &got); err != nil {
				t.Fatal(err)
			}
			if got["storageCapacity"] != true {
				t.Fatal("enterprise capacity remains unavailable")
			}
			for module, pin := range map[string]string{"knowledge": knowledgeCapacityPin, "ecoservices": ecoservicesCapacityPin} {
				storage := got[module].(map[string]any)["storage"].(map[string]any)
				if storage["capacityIdentityProvider"] != "minio-deployment" || storage["capacityMinIODeploymentID"] != pin {
					t.Fatalf("%s identity not pinned", module)
				}
				delete(storage, "capacityIdentityProvider")
				delete(storage, "capacityMinIODeploymentID")
			}
			delete(got, "storageCapacity")
			wantJSON, _ := json.Marshal(base)
			gotJSON, _ := json.Marshal(got)
			if string(wantJSON) != string(gotJSON) {
				t.Fatal("capacity assembly changed another module, identity or credential")
			}
			run(true)
			repeated, _ := os.ReadFile(output)
			if string(first) != string(repeated) {
				t.Fatal("repeat changed private runtime")
			}
			if err := os.WriteFile(filepath.Join(root, "knowledge-identity/deployment-id"), []byte(ecoservicesCapacityPin), 0600); err != nil {
				t.Fatal(err)
			}
			run(false)
			after, _ := os.ReadFile(output)
			if string(after) != string(first) {
				t.Fatal("changed pin overwrote retained runtime")
			}
			if err := os.Remove(filepath.Join(root, "knowledge-identity/deployment-id")); err != nil {
				t.Fatal(err)
			}
			run(false)
		})
	}
}

func writeCapacityJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestCapacityStorageInstallerPinsTrustedIdentity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX deployment fixture")
	}
	for _, module := range []string{"knowledge", "ecoservices"} {
		t.Run(module, func(t *testing.T) {
			root := t.TempDir()
			for _, dir := range []string{"private", "object-runtime", "capacity-identity", "init", "bin"} {
				if err := os.Mkdir(filepath.Join(root, dir), 0700); err != nil {
					t.Fatal(err)
				}
			}
			for _, dir := range []string{"private", "object-runtime"} {
				for _, file := range []string{"access-key", "secret-key"} {
					if err := os.WriteFile(filepath.Join(root, dir, file), []byte("fixture-private-value"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			mutation := filepath.Join(root, "policy-called")
			writeAccountComposeExecutable(t, filepath.Join(root, "bin/mc"), "#!/bin/sh\ncase \"$1 $2\" in\n 'admin info') printf '%s\\n' '{\"info\":{\"deploymentID\":\""+knowledgeCapacityPin+"\"}}' ;;\n 'admin policy') printf 'policy\\n' >> "+shellQuote(mutation)+" ;;\n 'alias set'|'ls --json') exit 0 ;;\n *) exit 99 ;;\nesac\n")
			script := capacityFixtureScript(t, root, "capacity-storage-init.sh", []string{"private", "object-runtime", "capacity-identity", "init"})
			run := func(ok bool) {
				t.Helper()
				cmd := exec.Command("sh", script, module)
				cmd.Env = append(os.Environ(), "PATH="+filepath.Join(root, "bin")+":"+os.Getenv("PATH"))
				out, err := cmd.CombinedOutput()
				if (err == nil) != ok {
					t.Fatalf("identity preparation success=%v: %v %s", ok, err, out)
				}
				if strings.Contains(string(out), "fixture-private-value") {
					t.Fatal("runtime credential leaked")
				}
			}
			run(true)
			run(true)
			pin, err := os.ReadFile(filepath.Join(root, "capacity-identity/deployment-id"))
			if err != nil || string(pin) != knowledgeCapacityPin+"\n" {
				t.Fatal("trusted identity not persisted")
			}
			before, _ := os.ReadFile(mutation)
			if err := os.WriteFile(filepath.Join(root, "capacity-identity/deployment-id"), []byte(ecoservicesCapacityPin+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			run(false)
			after, _ := os.ReadFile(mutation)
			if string(after) != string(before) {
				t.Fatal("changed identity altered runtime policy")
			}
		})
	}
}

func TestCapacityModuleEntrypointKeepsSelectedLayers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX deployment fixture")
	}
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	writeAccountComposeExecutable(t, filepath.Join(bin, "docker"), "#!/bin/sh\nprintf '%s\\n' \"$@\"\n")
	script, err := filepath.Abs("../deployments/docker/account-compose/compose-modules.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, options := range [][]string{nil, {"--chat"}, {"--public"}, {"--chat", "--public"}, {"--acquisition-plans"}, {"--store-observations", "--public"}} {
		args := append([]string{script}, options...)
		args = append(args, "--", "up", "--no-build", "-d", "--wait")
		cmd := exec.Command("sh", args...)
		cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("module entrypoint: %v %s", err, out)
		}
		text := string(out)
		for _, layer := range []string{"knowledge", "data-services", "modules", "ecoservices", "storage-capacity"} {
			if !strings.Contains(text, "docker-compose."+layer+".yml\n") {
				t.Fatalf("missing %s layer", layer)
			}
		}
		if !strings.HasSuffix(text, "up\n--no-build\n-d\n--wait\n") {
			t.Fatal("Compose arguments changed")
		}
		if strings.Contains(strings.Join(options, " "), "--chat") && !strings.Contains(text, "docker-compose.chat.yml\n-f\ndocker-compose.storage-capacity-chat.yml\n") {
			t.Fatal("Chat bypasses final capacity assembly")
		}
		if strings.Contains(strings.Join(options, " "), "--acquisition-plans") && !strings.Contains(text, "docker-compose.acquisition-plans.yml\n-f\ndocker-compose.storage-capacity-acquisition-plans.yml\n") {
			t.Fatal("Acquisition plans bypass final capacity assembly")
		}
		if strings.Contains(strings.Join(options, " "), "--store-observations") && !strings.Contains(text, "docker-compose.store-observations.yml\n-f\ndocker-compose.storage-capacity-store-observations.yml\n") {
			t.Fatal("Store observations bypass final capacity assembly")
		}
	}
}

func TestCapacityBootstrapRejectsRetainedProfileDrift(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX deployment fixture")
	}
	for _, tc := range []struct {
		name, flag string
		pin, ok    bool
	}{
		{"unchanged disabled", "", false, true}, {"unchanged enabled", "1", true, true},
		{"cannot enable retained", "1", false, false}, {"cannot drop layer", "", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			state := filepath.Join(root, "state")
			if err := os.Mkdir(state, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(state, ".bootstrap-complete"), nil, 0600); err != nil {
				t.Fatal(err)
			}
			if tc.pin {
				if err := os.WriteFile(filepath.Join(state, ".storage-capacity-enabled"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			source, err := filepath.Abs("../deployments/docker/account-compose")
			if err != nil {
				t.Fatal(err)
			}
			input, err := os.ReadFile(filepath.Join(source, "bootstrap.sh"))
			if err != nil {
				t.Fatal(err)
			}
			// Stop at the existing origin check after the real retained capacity guard.
			// This fixture never generates secrets or opens a database.
			contents := strings.NewReplacer(
				"state=/state", "state="+shellQuote(state),
				"/usr/local/lib/account-compose/", source+"/",
				`account_pin_origins "$state" retained`, "echo retained-profile-accepted; exit 0",
			).Replace(string(input))
			script := filepath.Join(root, "bootstrap.sh")
			if err := os.WriteFile(script, []byte(contents), 0700); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("sh", script)
			cmd.Env = append(os.Environ(), "ACCOUNT_STORAGE_CAPACITY_ENABLED="+tc.flag, "ACCOUNT_KNOWLEDGE_ENABLED=1", "ACCOUNT_DATA_SERVICES_ENABLED=1", "ACCOUNT_NATIVE_MODULES_ENABLED=1", "ACCOUNT_PUBLIC_TRIAL=", "ACCOUNT_COMMERCIAL_DATABASE=commercial", "ACCOUNT_IMAGE_AGENT_TRIAL=", "ACCOUNT_ISSUE36_LOCAL_TRIAL=")
			out, err := cmd.CombinedOutput()
			if (err == nil) != tc.ok {
				t.Fatalf("retained capacity guard: %v %s", err, out)
			}
			if tc.ok && !strings.Contains(string(out), "retained-profile-accepted") {
				t.Fatal("fixture did not reach the retained guard")
			}
		})
	}
}

func TestCapacityEffectiveComposeUsesFinalPrivateManifest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX entrypoint; effective Compose also checked directly on Windows")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker Compose is required for rendering")
	}
	if err := exec.Command("docker", "compose", "version").Run(); err != nil {
		t.Skip("Docker Compose is required for rendering")
	}
	root := t.TempDir()
	source := filepath.Join("..", "deployments/docker/account-compose")
	files, err := filepath.Glob(filepath.Join(source, "docker-compose*.yml"))
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, filepath.Join(source, "compose-modules.sh"))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, filepath.Base(file)), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ option, input, dependency string }{
		{"", "ecoservices-runtime", "ecoservices-storage-init"}, {"--chat", "chat-runtime", "chat-init"},
		{"--acquisition-plans", "acquisition-plans-runtime", "acquisition-plans-init"}, {"--store-observations", "store-observations-runtime", "store-observations-init"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			args := []string{filepath.Join(root, "compose-modules.sh")}
			if tc.option != "" {
				args = append(args, tc.option)
			}
			args = append(args, "--public", "--", "config", "--format", "json")
			cmd := exec.Command("sh", args...)
			cmd.Env = append(os.Environ(), "COMPOSE_PROJECT_NAME=capacity-render-fixture", "ACCOUNT_IDENTITY_PORT=29443", "ACCOUNT_APPLICATION_PORT=29444", "ACCOUNT_MAIL_PORT=29425", "ACCOUNT_TOOL_RELEASE_DIRECTORY="+root, `CHAT_ALLOWED_ORGANIZATION_IDS=["org-fixture"]`, "ACCOUNT_PUBLIC_APPLICATION_ORIGIN=https://trial.example.test", "ACCOUNT_PUBLIC_IDENTITY_ORIGIN=https://auth.example.test", "ACCOUNT_PUBLIC_APPLICATION_HOST=trial.example.test", "ACCOUNT_PUBLIC_IDENTITY_HOST=auth.example.test")
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("effective Compose rendering: %v", err)
			}
			var document struct {
				Services map[string]struct {
					Command     []string
					NetworkMode string `json:"network_mode"`
					Environment map[string]string
					Volumes     []struct {
						Source, Target string
						ReadOnly       bool `json:"read_only"`
					}
					DependsOn map[string]any `json:"depends_on"`
				}
			}
			if err := json.Unmarshal(out, &document); err != nil {
				t.Fatal(err)
			}
			app, init := document.Services["current-application"], document.Services["storage-capacity-init"]
			if strings.Join(app.Command, " ") != "-config /private/storage-capacity/current-application.json" {
				t.Fatal("application bypasses capacity manifest")
			}
			if init.NetworkMode != "none" || init.DependsOn[tc.dependency] == nil {
				t.Fatal("configuration publishes before its original initializer completes")
			}
			inputs := 0
			for _, mount := range init.Volumes {
				if mount.Target == "/base" {
					inputs++
					if mount.Source != tc.input || !mount.ReadOnly {
						t.Fatal("wrong final module input")
					}
				}
				if strings.Contains(mount.Source, "secret") {
					t.Fatal("capacity configuration received administrative secrets")
				}
			}
			if inputs != 1 {
				t.Fatal("ambiguous final input")
			}
			if document.Services["bootstrap"].Environment["ACCOUNT_STORAGE_CAPACITY_ENABLED"] != "1" {
				t.Fatal("capacity profile is not pinned")
			}
			for _, name := range []string{"knowledge-storage-init", "ecoservices-storage-init"} {
				if document.Services[name].Environment["ACCOUNT_STORAGE_CAPACITY_ENABLED"] != "1" {
					t.Fatal("capacity IAM initializer disabled")
				}
			}
			for _, flag := range []string{"LISTINGKIT_KNOWLEDGE_ENABLED", "LISTINGKIT_DATA_SERVICES_ENABLED", "LISTINGKIT_TOOL_MARKET_ENABLED", "LISTINGKIT_REPORT_CENTER_ENABLED", "LISTINGKIT_NOTIFICATION_CENTER_ENABLED", "LISTINGKIT_ECOSERVICES_ENABLED"} {
				if document.Services["listingkit-ui"].Environment[flag] != "true" {
					t.Fatalf("module flag lost: %s", flag)
				}
			}
		})
	}
}

func capacityFixtureScript(t *testing.T, root, name string, dirs []string) string {
	t.Helper()
	input, err := os.ReadFile(filepath.Join("..", "deployments/docker/account-compose", name))
	if err != nil {
		t.Fatal(err)
	}
	var pairs []string
	for _, dir := range dirs {
		pairs = append(pairs, "/"+dir+"/", filepath.Join(root, dir)+"/")
	}
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(strings.NewReplacer(pairs...).Replace(string(input))), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}
