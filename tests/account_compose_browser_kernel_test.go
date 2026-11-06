package tests

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func kernelIngressYAML(t *testing.T, name string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "deployments", "docker", name))
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := yaml.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestAccountComposeBrowserKernelIngress(t *testing.T) {
	fragment := kernelIngressYAML(t, "account-compose/traefik-browser-kernel.fragment.yml")
	http := fragment["http"].(map[string]any)
	if len(fragment) != 1 || len(http) != 2 {
		t.Fatal("fragment may only add HTTP routers and its own service; preserve TLS and middleware")
	}
	services := http["services"].(map[string]any)
	wantServices := map[string]any{
		"browser-kernel-api": map[string]any{"loadBalancer": map[string]any{
			"servers": []any{map[string]any{"url": "http://127.0.0.1:8085"}},
		}},
	}
	if !reflect.DeepEqual(services, wantServices) {
		t.Fatalf("kernel reads must use the same current-application: %v", services)
	}
	routers := http["routers"].(map[string]any)
	if len(routers) != 2 {
		t.Fatal("only the catalog and artifact GET routers are allowed")
	}
	const base = "/api/v1/browser/kernel-releases"
	const uuid = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"
	wantRules := map[string]string{
		"browser-kernel-catalog":  "Host(`localhost`) && Method(`GET`) && Path(`" + base + "`)",
		"browser-kernel-artifact": "Host(`localhost`) && Method(`GET`) && PathRegexp(`^" + base + "/" + uuid + "/artifact$`)",
	}
	for name, wantRule := range wantRules {
		router := routers[name].(map[string]any)
		want := map[string]any{"entryPoints": []any{"application"}, "rule": wantRule,
			"priority": 200, "service": "browser-kernel-api", "tls": map[string]any{}}
		if !reflect.DeepEqual(router, want) {
			t.Fatalf("%s must retain HTTPS, original path/query/Authorization and backend auth: %v", name, router)
		}
	}

	// Controlled, in-memory checks of the actual rule values. This does not run
	// Traefik, open a socket, contact a retained endpoint or replace backend auth.
	const id = "12345678-abcd-1234-abcd-123456789abc"
	artifact := base + "/" + id + "/artifact"
	for _, target := range []string{base, artifact} {
		for _, query := range []string{"", "?os=windows&arch=amd64&clientVersion=1.0.0", "?cursor=bad&limit=1&limit=2"} {
			t.Run(target+query, func(t *testing.T) {
				req := httptest.NewRequest("GET", "https://localhost:35444"+target+query, nil)
				matches := 0
				for _, raw := range routers {
					rule := raw.(map[string]any)["rule"].(string)
					if kernelIngressRuleMatches(t, rule, req.Method, req.URL.Hostname(), req.URL.Path) {
						matches++
					}
				}
				if matches != 1 {
					t.Fatal("exactly one read route must match independently of query validation by the backend")
				}
			})
		}
	}
	negativePaths := []string{base + "/", base + "-extra", base + "/" + id,
		artifact + "/", artifact + "/extra", strings.Replace(artifact, id, strings.ToUpper(id), 1),
		strings.Replace(artifact, id, "not-a-uuid", 1), strings.Replace(artifact, id, strings.ReplaceAll(id, "-", ""), 1),
		strings.Replace(artifact, id, "{"+id+"}", 1), strings.Replace(artifact, id, "urn:uuid:"+id, 1),
		"/api/v1/platform/browser/kernel-releases/" + id, "/api/v1/platform/browser/kernel-releases/" + id + "/withdraw",
		"/api/v1/platform/browser/kernel-releases/" + id + "/recommendation",
		"/api/auth/session", "/api/v1/stores", "/api/v1/account/profile", "/workbench", "/data-api/v1/amazon/jobs"}
	for _, raw := range routers {
		rule := raw.(map[string]any)["rule"].(string)
		for _, method := range []string{"HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"} {
			for _, target := range []string{base, artifact} {
				if kernelIngressRuleMatches(t, rule, method, "localhost", target) {
					t.Errorf("new route accepted %s %s", method, target)
				}
			}
		}
		for _, target := range negativePaths {
			if kernelIngressRuleMatches(t, rule, "GET", "localhost", target) {
				t.Errorf("new route accepted adjacent/platform path %s", target)
			}
		}
		if kernelIngressRuleMatches(t, rule, "GET", "other.example.test", base) {
			t.Fatal("new route accepted a different public host")
		}
	}
}

// The shape assertion above limits this helper to three documented v3 matchers.
// Evaluate the PathRegexp with Go's regexp engine, as Traefik v3 does.
func kernelIngressRuleMatches(t *testing.T, rule, method, host, path string) bool {
	t.Helper()
	shape := regexp.MustCompile("^Host\\(`([^`]+)`\\) && Method\\(`([^`]+)`\\) && (Path|PathRegexp)\\(`([^`]+)`\\)$")
	parts := shape.FindStringSubmatch(rule)
	if parts == nil {
		t.Fatalf("unsupported ingress rule shape: %s", rule)
	}
	pathMatches := parts[4] == path
	if parts[3] == "PathRegexp" {
		pattern, err := regexp.Compile(parts[4])
		if err != nil {
			t.Fatal(err)
		}
		pathMatches = pattern.MatchString(path)
	}
	return strings.EqualFold(host, parts[1]) && method == parts[2] && pathMatches
}

func TestAccountComposeBrowserKernelFragmentComposition(t *testing.T) {
	fragment := kernelIngressYAML(t, "account-compose/traefik-browser-kernel.fragment.yml")["http"].(map[string]any)
	// Each shipped full dynamic variant can be the final Compose mount. Never
	// replace one with a kernel-only dynamic file or discard an earlier overlay.
	for _, name := range []string{"referrals-compose/traefik-dynamic.yml", "account-compose/traefik-data-services.yml",
		"account-compose/traefik-image-trial.yml", "account-compose/traefik-source-media-trial.yml"} {
		t.Run(name, func(t *testing.T) {
			original := kernelIngressYAML(t, name)
			before, _ := yaml.Marshal(original)
			combined := kernelIngressYAML(t, name)
			http := combined["http"].(map[string]any)
			for section, additions := range fragment {
				existing := http[section].(map[string]any)
				for key, value := range additions.(map[string]any) {
					if _, collision := existing[key]; collision {
						t.Fatalf("fragment would overwrite %s.%s", section, key)
					}
					existing[key] = value
				}
			}
			ui := http["routers"].(map[string]any)["listingkit-ui"].(map[string]any)
			if ui["service"] != "listingkit-ui" || ui["rule"] != "Host(`localhost`)" {
				t.Fatal("Web BFF/Store fallback must stay on the UI")
			}
			for section, additions := range fragment {
				for key := range additions.(map[string]any) {
					delete(http[section].(map[string]any), key)
				}
			}
			after, _ := yaml.Marshal(combined)
			if string(before) != string(after) {
				t.Fatal("adding/removing only the fragment must preserve every original router/service/middleware/TLS value")
			}
		})
	}
	base := kernelIngressYAML(t, "account-compose/docker-compose.yml")
	traefik := base["services"].(map[string]any)["traefik"].(map[string]any)
	if traefik["image"] != "traefik:v3.6.8" || !strings.Contains(fmt.Sprint(traefik["command"]), "--providers.file.filename=/etc/traefik/dynamic.yml") {
		t.Fatal("recheck the pinned Traefik version/file-provider contract")
	}
	for _, directory := range []string{"account-compose", "referrals-compose"} {
		entries, err := os.ReadDir(filepath.Join("..", "deployments", "docker", directory))
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !(strings.HasSuffix(entry.Name(), ".yml") || strings.HasSuffix(entry.Name(), ".sh")) || entry.Name() == "traefik-browser-kernel.fragment.yml" {
				continue
			}
			data, err := os.ReadFile(filepath.Join("..", "deployments", "docker", directory, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			for _, forbidden := range []string{"traefik-browser-kernel.fragment.yml", "browser-kernel-api", "/api/v1/browser/kernel-releases"} {
				if strings.Contains(string(data), forbidden) {
					t.Fatalf("kernel ingress must stay opt-in; found %s in %s", forbidden, entry.Name())
				}
			}
		}
	}
}
