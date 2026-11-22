package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPhase2TargetDirectionDepguardRulesCoverApprovedBoundaries(t *testing.T) {
	configPath := filepath.Join("..", ".golangci.yml")
	rules := loadDepguardRules(t, configPath)
	targetRule := requireDepguardRule(t, rules, "target_domain_concrete_infrastructure")
	platformRule := requireDepguardRule(t, rules, "platform_domain_dependencies")
	targetFiles := stringSet(targetRule.Files)
	platformFiles := stringSet(platformRule.Files)
	targetDeny := depguardDenyPackageSet(targetRule)
	platformDeny := depguardDenyPackageSet(platformRule)

	domains := []string{
		"listing", "product", "marketplace", "agent", "knowledge",
		"resourcecatalog", "commercial", "ledger", "organization",
	}
	for _, domain := range domains {
		for _, glob := range []string{
			fmt.Sprintf("**/internal/%s/*.go", domain),
			fmt.Sprintf("**/internal/%s/**/*.go", domain),
		} {
			if _, ok := targetFiles[glob]; !ok {
				t.Errorf("target_domain_concrete_infrastructure must cover %s", glob)
			}
		}

		packagePath := "github.com/henryxu1008/shuomi-ai-commerce/internal/" + domain
		for _, suffix := range []string{"$", "/"} {
			pattern := packagePath + suffix
			if _, ok := platformDeny[pattern]; !ok {
				t.Errorf("platform_domain_dependencies must deny %s", pattern)
			}
		}
	}

	for _, glob := range []string{
		"**/internal/platform/*.go",
		"**/internal/platform/**/*.go",
	} {
		if _, ok := platformFiles[glob]; !ok {
			t.Errorf("platform_domain_dependencies must cover %s", glob)
		}
	}

	for _, packagePath := range []string{
		"github.com/henryxu1008/shuomi-ai-commerce/internal/platform",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/integration",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/infra",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/app",
		"gorm.io",
		"go.temporal.io",
		"go.opentelemetry.io",
		"github.com/open-feature",
		"github.com/aws",
		"github.com/redis",
		"github.com/rabbitmq",
	} {
		for _, suffix := range []string{"$", "/"} {
			pattern := packagePath + suffix
			if _, ok := targetDeny[pattern]; !ok {
				t.Errorf("target_domain_concrete_infrastructure must deny %s", pattern)
			}
		}
	}
}

func TestPhase3ProductDomainBoundaryDepguardRuleCoversOnlyTargetSubpackages(t *testing.T) {
	rules := loadDepguardRules(t, filepath.Join("..", ".golangci.yml"))
	rule := requireDepguardRule(t, rules, "phase3_product_domain_boundaries")
	files := stringSet(rule.Files)
	denied := depguardDenyPackageSet(rule)

	for _, glob := range []string{
		"**/internal/product/catalog/**/*.go",
		"**/internal/product/sourcing/**/*.go",
		"**/internal/product/review/proposal/**/*.go",
		"**/internal/product/asset/**/*.go",
		"**/internal/product/image/**/*.go",
	} {
		if _, ok := files[glob]; !ok {
			t.Errorf("phase3_product_domain_boundaries must cover %s", glob)
		}
	}

	for _, packagePath := range []string{
		"github.com/henryxu1008/shuomi-ai-commerce/internal/app",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/platform",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/integration",
		"gorm.io/gorm",
		"go.temporal.io",
		"github.com/aws/aws-sdk-go-v2",
		"github.com/aws/smithy-go",
		"github.com/sashabaranov/go-openai",
		"github.com/tencentcloud/tencentcloud-sdk-go",
	} {
		if _, ok := denied[packagePath]; !ok {
			t.Errorf("phase3_product_domain_boundaries must deny %s", packagePath)
		}
	}
}

func TestPhase3FinalProductHardCutDepguardRules(t *testing.T) {
	rules := loadDepguardRules(t, filepath.Join("..", ".golangci.yml"))

	retired := requireDepguardRule(t, rules, "phase3_retired_product_imports")
	retiredFiles := stringSet(retired.Files)
	if _, ok := retiredFiles["**/*.go"]; !ok {
		t.Error("phase3_retired_product_imports must cover every Go entrypoint via **/*.go")
	}
	retiredDenied := depguardDenyPackageSet(retired)
	for _, packagePath := range []string{
		"github.com/henryxu1008/shuomi-ai-commerce/internal/catalog",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/asset",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/imageasset",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/productenrich",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/productimage",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset/assettest",
	} {
		for _, suffix := range []string{"$", "/"} {
			if _, ok := retiredDenied[packagePath+suffix]; !ok {
				t.Errorf("phase3_retired_product_imports must deny %s%s", packagePath, suffix)
			}
		}
	}

	consumers := requireDepguardRule(t, rules, "phase3_product_consumers_read_only")
	consumerFiles := stringSet(consumers.Files)
	for _, root := range []string{"listingkit", "sds", "amazonlisting"} {
		for _, glob := range []string{
			"**/internal/" + root + "/*.go",
			"**/internal/" + root + "/**/*.go",
		} {
			if _, ok := consumerFiles[glob]; !ok {
				t.Errorf("phase3_product_consumers_read_only must cover %s", glob)
			}
		}
	}
	consumerDenied := depguardDenyPackageSet(consumers)
	for _, packagePath := range []string{
		"github.com/henryxu1008/shuomi-ai-commerce/internal/product/image",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent/store",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent/temporal",
	} {
		for _, suffix := range []string{"$", "/"} {
			if _, ok := consumerDenied[packagePath+suffix]; !ok {
				t.Errorf("phase3_product_consumers_read_only must deny %s%s", packagePath, suffix)
			}
		}
	}
}

func TestDepguardRuleParsingUsesYAMLSemantics(t *testing.T) {
	rules := parseDepguardRules(t, []byte(`
linters-settings:
  depguard:
    rules:
      semantic_rule:
        files:
          # - "**/internal/commented/*.go"
          - '**/internal/single-quoted/*.go'
        deny:
          # - pkg: "github.com/henryxu1008/shuomi-ai-commerce/internal/commented$"
          - pkg: github.com/henryxu1008/shuomi-ai-commerce/internal/unquoted$
`))
	rule := requireDepguardRule(t, rules, "semantic_rule")
	files := stringSet(rule.Files)
	denied := depguardDenyPackageSet(rule)

	if _, ok := files["**/internal/commented/*.go"]; ok {
		t.Fatal("commented file glob must not be a semantic depguard entry")
	}
	if _, ok := denied["github.com/henryxu1008/shuomi-ai-commerce/internal/commented$"]; ok {
		t.Fatal("commented deny package must not be a semantic depguard entry")
	}
	if _, ok := files["**/internal/single-quoted/*.go"]; !ok {
		t.Fatal("single-quoted file glob must retain its YAML value")
	}
	if _, ok := denied["github.com/henryxu1008/shuomi-ai-commerce/internal/unquoted$"]; !ok {
		t.Fatal("unquoted deny package must retain its YAML value")
	}
}

func TestCommerceToolDepguardUsesExactStrictAllowlist(t *testing.T) {
	rules := loadDepguardRules(t, filepath.Join("..", ".golangci.yml"))
	rule := requireDepguardRule(t, rules, "commercetool_boundaries")

	if rule.ListMode != "strict" {
		t.Errorf("commercetool_boundaries list-mode = %q, want strict", rule.ListMode)
	}

	wantFiles := []string{
		"**/internal/commercetool/*.go",
		"**/internal/commercetool/**/*.go",
	}
	assertExactStringSet(t, "commercetool_boundaries files", rule.Files, wantFiles)

	wantAllow := []string{
		"$gostd",
		"github.com/go-openapi/jsonpointer$",
		"golang.org/x/mod/semver$",
		"github.com/santhosh-tekuri/jsonschema/v6$",
		"go.opentelemetry.io/otel/attribute$",
		"go.opentelemetry.io/otel/trace$",
	}
	assertExactStringSet(t, "commercetool_boundaries allow", rule.Allow, wantAllow)
	for _, packagePath := range rule.Allow {
		if strings.HasPrefix(packagePath, "$") {
			continue
		}
		if !strings.HasSuffix(packagePath, "$") {
			t.Errorf("commercetool_boundaries non-variable allow entry %q must use depguard exact-match suffix $", packagePath)
		}
	}

	allowed := stringSet(rule.Allow)
	for category, packagePaths := range map[string][]string{
		"internal packages":   {"github.com/henryxu1008/shuomi-ai-commerce/internal", "github.com/henryxu1008/shuomi-ai-commerce/internal/commercetool"},
		"agent frameworks":    {"github.com/cloudwego/eino"},
		"transport":           {"github.com/gin-gonic/gin", "github.com/rabbitmq/amqp091-go"},
		"workflow":            {"go.temporal.io/sdk"},
		"persistence":         {"gorm.io/gorm"},
		"provider SDKs":       {"github.com/aws/aws-sdk-go-v2", "go.opentelemetry.io/otel"},
		"marketplace clients": {"github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace"},
	} {
		for _, packagePath := range packagePaths {
			if _, ok := allowed[packagePath]; ok {
				t.Errorf("commercetool_boundaries must not allow broad %s dependency %q", category, packagePath)
			}
		}
	}
}

func TestPhase2RetiredRuntimePathsHavePermanentDepguardRules(t *testing.T) {
	rules := loadDepguardRules(t, filepath.Join("..", ".golangci.yml"))
	rule := requireDepguardRule(t, rules, "phase2_retired_runtime_paths")
	files := stringSet(rule.Files)
	denied := depguardDenyPackageSet(rule)

	for _, glob := range []string{"**/internal/*.go", "**/internal/**/*.go"} {
		if _, ok := files[glob]; !ok {
			t.Errorf("phase2_retired_runtime_paths must cover production files with %s", glob)
		}
	}
	for _, path := range []string{
		"github.com/henryxu1008/shuomi-ai-commerce/internal/core/lifecycle",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/infra/database",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/infra/redisclient",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/infra/lock",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/infra/rabbitmq",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/infra/worker",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/infra/clients/openai",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/infra/clients/geminiimage",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/infra/clients/grsai",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/infra/storage",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/infra/resilience",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/infra/metrics",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/infra/monitoring",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/pkg/safeimagehttp",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/pkg/hashx",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/pkg/mathx",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/pkg/ptr",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/pkg/strx",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/pkg/timex",
	} {
		for _, suffix := range []string{"$", "/"} {
			if _, ok := denied[path+suffix]; !ok {
				t.Errorf("phase2_retired_runtime_paths must deny %s", path+suffix)
			}
		}
	}
}

type depguardConfig struct {
	LintersSettings struct {
		Depguard struct {
			Rules map[string]depguardRule `yaml:"rules"`
		} `yaml:"depguard"`
	} `yaml:"linters-settings"`
}

type depguardRule struct {
	ListMode string         `yaml:"list-mode"`
	Files    []string       `yaml:"files"`
	Allow    []string       `yaml:"allow"`
	Deny     []depguardDeny `yaml:"deny"`
}

type depguardDeny struct {
	Package string `yaml:"pkg"`
}

func loadDepguardRules(t *testing.T, configPath string) map[string]depguardRule {
	t.Helper()
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}
	return parseDepguardRules(t, content)
}

func parseDepguardRules(t *testing.T, content []byte) map[string]depguardRule {
	t.Helper()
	var config depguardConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		t.Fatalf("parse depguard YAML: %v", err)
	}
	return config.LintersSettings.Depguard.Rules
}

func requireDepguardRule(t *testing.T, rules map[string]depguardRule, ruleName string) depguardRule {
	t.Helper()
	rule, ok := rules[ruleName]
	if !ok {
		t.Fatalf(".golangci.yml must define %s", ruleName)
	}
	return rule
}

func stringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func assertExactStringSet(t *testing.T, name string, got, want []string) {
	t.Helper()
	gotSet := stringSet(got)
	wantSet := stringSet(want)
	for value := range wantSet {
		if _, ok := gotSet[value]; !ok {
			t.Errorf("%s must contain %q", name, value)
		}
	}
	for value := range gotSet {
		if _, ok := wantSet[value]; !ok {
			t.Errorf("%s must not contain extra value %q", name, value)
		}
	}
	if len(got) != len(want) {
		t.Errorf("%s has %d entries, want exactly %d", name, len(got), len(want))
	}
}

func depguardDenyPackageSet(rule depguardRule) map[string]struct{} {
	result := make(map[string]struct{}, len(rule.Deny))
	for _, deny := range rule.Deny {
		result[deny.Package] = struct{}{}
	}
	return result
}

func TestPlatformRegistrationDepguardPatternsRespectPackageBoundaries(t *testing.T) {
	configPath := filepath.Join("..", ".golangci.yml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}

	config := string(content)
	start := strings.Index(config, "      platform_registration_boundaries:\n")
	if start == -1 {
		t.Fatalf("%s must define platform_registration_boundaries", configPath)
	}
	end := strings.Index(config[start+1:], "\n      aicapability_boundaries:")
	if end == -1 {
		t.Fatalf("%s must keep platform_registration_boundaries before aicapability_boundaries", configPath)
	}
	config = config[start : start+1+end]

	for _, packagePath := range []string{
		"github.com/henryxu1008/shuomi-ai-commerce/internal/app/httpapi",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/asset",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/listingkit",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/productimage",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/publishing",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/workspace",
	} {
		for _, suffix := range []string{"$", "/"} {
			pattern := fmt.Sprintf(`- pkg: "%s%s"`, packagePath, suffix)
			if !strings.Contains(config, pattern) {
				t.Errorf("%s must contain depguard package-boundary pattern %s", configPath, pattern)
			}
		}
	}
}

func TestAICapabilityDepguardUsesStrictAllowlist(t *testing.T) {
	configPath := filepath.Join("..", ".golangci.yml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}

	config := string(content)
	start := strings.Index(config, "      aicapability_boundaries:\n")
	if start == -1 {
		t.Fatalf("%s must define aicapability_boundaries", configPath)
	}
	end := strings.Index(config[start+1:], "\n      source_handoff_legacy_http:")
	if end == -1 {
		t.Fatalf("%s must keep aicapability_boundaries before source_handoff_legacy_http", configPath)
	}
	config = config[start : start+1+end]

	if !strings.Contains(config, "list-mode: strict") {
		t.Errorf("%s must make aicapability_boundaries a strict allowlist", configPath)
	}

	for _, packagePath := range []string{
		"$gostd",
		"gorm.io/gorm",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability",
	} {
		pattern := fmt.Sprintf(`- "%s"`, packagePath)
		if !strings.Contains(config, pattern) {
			t.Errorf("%s must allow %s for AI capability contracts", configPath, packagePath)
		}
	}
}

func TestImageAgentEffectPolicyDepguardUsesStrictAllowlist(t *testing.T) {
	configPath := filepath.Join("..", ".golangci.yml")
	config := readRepositoryText(t, configPath)
	start := strings.Index(config, "      imageagent_effectpolicy_boundaries:\n")
	if start == -1 {
		t.Fatalf("%s must define imageagent_effectpolicy_boundaries", configPath)
	}
	end := strings.Index(config[start+1:], "\n      source_handoff_legacy_http:")
	if end == -1 {
		t.Fatalf("%s must keep imageagent_effectpolicy_boundaries before source_handoff_legacy_http", configPath)
	}
	config = config[start : start+1+end]

	for _, required := range []string{
		"list-mode: strict",
		`- "**/internal/imageagent/effectpolicy/*.go"`,
		`- "**/internal/imageagent/effectpolicy/**/*.go"`,
		`- "$gostd"`,
		`- "github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent"`,
	} {
		if !strings.Contains(config, required) {
			t.Errorf("%s imageagent effect policy boundary must contain %s", configPath, required)
		}
	}

	for _, forbidden := range []string{"gorm.io/", "go.temporal.io/", "internal/imageagent/store"} {
		if strings.Contains(config, forbidden) {
			t.Errorf("%s imageagent effect policy boundary must not allow %s", configPath, forbidden)
		}
	}
}

func TestAlibaba1688CrawlerDepguardPatternKeepsHTTPAPIAdapterException(t *testing.T) {
	configPath := filepath.Join("..", ".golangci.yml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}

	config := string(content)
	start := strings.Index(config, "      alibaba1688_listingkit_root:\n")
	if start == -1 {
		t.Fatalf("%s must define alibaba1688_listingkit_root", configPath)
	}
	end := strings.Index(config[start+1:], "\n  govet:")
	if end == -1 {
		t.Fatalf("%s must keep alibaba1688_listingkit_root before govet", configPath)
	}
	config = config[start : start+1+end]

	if !strings.Contains(config, `- pkg: "github.com/henryxu1008/shuomi-ai-commerce/internal/listingkit$"`) {
		t.Errorf("%s must deny the exact ListingKit root facade", configPath)
	}
	if strings.Contains(config, `- pkg: "github.com/henryxu1008/shuomi-ai-commerce/internal/listingkit/"`) {
		t.Errorf("%s must keep ListingKit subpackage adapters outside the root-facade guard", configPath)
	}
}

func TestZitadelAuthRuntimeDepguardPatternCoversListingKitPackageTree(t *testing.T) {
	configPath := filepath.Join("..", ".golangci.yml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}

	config := string(content)
	start := strings.Index(config, "      authruntime_zitadel_listingkit:\n")
	if start == -1 {
		t.Fatalf("%s must define authruntime_zitadel_listingkit", configPath)
	}
	end := strings.Index(config[start+1:], "\n      alibaba1688_listingkit_root:")
	if end == -1 {
		t.Fatalf("%s must keep authruntime_zitadel_listingkit before alibaba1688_listingkit_root", configPath)
	}
	config = config[start : start+1+end]

	for _, suffix := range []string{"$", "/"} {
		pattern := fmt.Sprintf(`- pkg: "github.com/henryxu1008/shuomi-ai-commerce/internal/listingkit%s"`, suffix)
		if !strings.Contains(config, pattern) {
			t.Errorf("%s must contain depguard package-tree pattern %s", configPath, pattern)
		}
	}
}

func TestCmdProductionDepguardPatternCoversDomainAndInfraTrees(t *testing.T) {
	configPath := filepath.Join("..", ".golangci.yml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}

	config := string(content)
	start := strings.Index(config, "      cmd_domain_dependencies:\n")
	if start == -1 {
		t.Fatalf("%s must define cmd_domain_dependencies", configPath)
	}
	end := strings.Index(config[start+1:], "\n      cmd_legacy_app_compatibility:")
	if end == -1 {
		t.Fatalf("%s must keep cmd_domain_dependencies before cmd_legacy_app_compatibility", configPath)
	}
	config = config[start : start+1+end]

	for _, packagePath := range []string{
		"github.com/henryxu1008/shuomi-ai-commerce/internal/amazon",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/amazonlisting",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/asset",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/infra",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/listingkit",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/productenrich",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/productimage",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/publishing",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/shein",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/temu",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/workspace",
	} {
		for _, suffix := range []string{"$", "/"} {
			pattern := fmt.Sprintf(`- pkg: "%s%s"`, packagePath, suffix)
			if !strings.Contains(config, pattern) {
				t.Errorf("%s must contain depguard package-tree pattern %s", configPath, pattern)
			}
		}
	}
}

func TestTemporalRuntimeDepguardPatternCoversHTTPAPIPackageTrees(t *testing.T) {
	configPath := filepath.Join("..", ".golangci.yml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}

	config := string(content)
	start := strings.Index(config, "      temporal_runtime_httpapi:\n")
	if start == -1 {
		t.Fatalf("%s must define temporal_runtime_httpapi", configPath)
	}
	end := strings.Index(config[start+1:], "\n      authruntime_zitadel_listingkit:")
	if end == -1 {
		t.Fatalf("%s must keep temporal_runtime_httpapi before authruntime_zitadel_listingkit", configPath)
	}
	config = config[start : start+1+end]

	for _, packagePath := range []string{
		"github.com/henryxu1008/shuomi-ai-commerce/internal/app/httpapi",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/listingkit/httpapi",
	} {
		for _, suffix := range []string{"$", "/"} {
			pattern := fmt.Sprintf(`- pkg: "%s%s"`, packagePath, suffix)
			if !strings.Contains(config, pattern) {
				t.Errorf("%s must contain depguard package-tree pattern %s", configPath, pattern)
			}
		}
	}
}

func TestDomainAppHTTPAPIDepguardPatternCoversBusinessTrees(t *testing.T) {
	configPath := filepath.Join("..", ".golangci.yml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}

	config := string(content)
	start := strings.Index(config, "      domain_app_httpapi_boundaries:\n")
	if start == -1 {
		t.Fatalf("%s must define domain_app_httpapi_boundaries", configPath)
	}
	end := strings.Index(config[start+1:], "\n      platform_registration_boundaries:")
	if end == -1 {
		t.Fatalf("%s must keep domain_app_httpapi_boundaries before platform_registration_boundaries", configPath)
	}
	config = config[start : start+1+end]

	for _, packagePath := range []string{
		"amazon",
		"amazonlisting",
		"product/asset",
		"product/catalog",
		"product/review/proposal",
		"product/image",
		"product/sourcing",
		"listing",
		"listingkit",
		"marketplace",
		"pricing",
		"publishing",
		"sds",
		"shein",
		"temu",
	} {
		for _, pattern := range []string{
			fmt.Sprintf(`- "**/internal/%s/*.go"`, packagePath),
			fmt.Sprintf(`- "**/internal/%s/**/*.go"`, packagePath),
		} {
			if !strings.Contains(config, pattern) {
				t.Errorf("%s must cover domain package files with %s", configPath, pattern)
			}
		}
	}

	for _, suffix := range []string{"$", "/"} {
		pattern := fmt.Sprintf(`- pkg: "github.com/henryxu1008/shuomi-ai-commerce/internal/app/httpapi%s"`, suffix)
		if !strings.Contains(config, pattern) {
			t.Errorf("%s must deny app/httpapi package-tree pattern %s", configPath, pattern)
		}
	}
}

func TestInfrastructureBusinessDepguardPatternCoversInfrastructureTrees(t *testing.T) {
	configPath := filepath.Join("..", ".golangci.yml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}

	config := string(content)
	start := strings.Index(config, "      infrastructure_business_boundaries:\n")
	if start == -1 {
		t.Fatalf("%s must define infrastructure_business_boundaries", configPath)
	}
	end := strings.Index(config[start+1:], "\n      domain_app_httpapi_boundaries:")
	if end == -1 {
		t.Fatalf("%s must keep infrastructure_business_boundaries before domain_app_httpapi_boundaries", configPath)
	}
	config = config[start : start+1+end]

	for _, packagePath := range []string{"infra", "integration", "platform", "platformbase", "platformtask"} {
		for _, pattern := range []string{
			fmt.Sprintf(`- "**/internal/%s/*.go"`, packagePath),
			fmt.Sprintf(`- "**/internal/%s/**/*.go"`, packagePath),
		} {
			if !strings.Contains(config, pattern) {
				t.Errorf("%s must cover infrastructure package files with %s", configPath, pattern)
			}
		}
	}
	for _, pattern := range []string{
		`- "!**/internal/integration/persistence/listing/submission/*.go"`,
		`- "!**/internal/integration/persistence/listing/submission/**/*.go"`,
		`- "!**/internal/integration/persistence/product/catalog/*.go"`,
		`- "!**/internal/integration/persistence/product/catalog/**/*.go"`,
	} {
		if !strings.Contains(config, pattern) {
			t.Errorf("%s must exclude the exact approved domain persistence adapters with %s", configPath, pattern)
		}
	}

	for _, packagePath := range []string{
		"amazon",
		"amazonlisting",
		"asset",
		"product/catalog",
		"listing",
		"listingkit",
		"marketplace",
		"pricing",
		"productenrich",
		"productimage",
		"publishing",
		"sds",
		"shein",
		"temu",
		"workspace",
	} {
		for _, suffix := range []string{"$", "/"} {
			pattern := fmt.Sprintf(`- pkg: "github.com/henryxu1008/shuomi-ai-commerce/internal/%s%s"`, packagePath, suffix)
			if !strings.Contains(config, pattern) {
				t.Errorf("%s must deny business package-tree pattern %s", configPath, pattern)
			}
		}
	}
	adapterRule := requireDepguardRule(t, loadDepguardRules(t, configPath), "product_catalog_persistence_boundary")
	adapterFiles := stringSet(adapterRule.Files)
	for _, pattern := range []string{
		"**/internal/integration/persistence/product/catalog/*.go",
		"**/internal/integration/persistence/product/catalog/**/*.go",
	} {
		if _, ok := adapterFiles[pattern]; !ok {
			t.Errorf("product_catalog_persistence_boundary must cover only the approved adapter with %s", pattern)
		}
	}
	adapterDeny := depguardDenyPackageSet(adapterRule)
	for _, packagePath := range []string{
		"github.com/henryxu1008/shuomi-ai-commerce/internal/amazonlisting$",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/listingkit$",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace$",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/productenrich$",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/productimage$",
	} {
		if _, ok := adapterDeny[packagePath]; !ok {
			t.Errorf("product_catalog_persistence_boundary must keep adapter free of %s", packagePath)
		}
	}
	if _, denied := adapterDeny["github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog$"]; denied {
		t.Error("product_catalog_persistence_boundary must allow the adapter to implement Catalog-owned ports")
	}

	submissionRule := requireDepguardRule(t, loadDepguardRules(t, configPath), "listing_submission_persistence_boundary")
	submissionFiles := stringSet(submissionRule.Files)
	for _, pattern := range []string{
		"**/internal/integration/persistence/listing/submission/*.go",
		"**/internal/integration/persistence/listing/submission/**/*.go",
	} {
		if _, ok := submissionFiles[pattern]; !ok {
			t.Errorf("listing_submission_persistence_boundary must cover only the approved adapter with %s", pattern)
		}
	}
	submissionDeny := depguardDenyPackageSet(submissionRule)
	for _, packagePath := range []string{
		"github.com/henryxu1008/shuomi-ai-commerce/internal/amazonlisting$",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/listingkit$",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace$",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/product$",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/publishing$",
	} {
		if _, ok := submissionDeny[packagePath]; !ok {
			t.Errorf("listing_submission_persistence_boundary must keep adapter free of %s", packagePath)
		}
	}
	if _, denied := submissionDeny["github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission$"]; denied {
		t.Error("listing_submission_persistence_boundary must allow the adapter to implement Submission-owned ports")
	}
}

func TestProductCatalogPersistenceDepguardDeniesEverySiblingProductDomain(t *testing.T) {
	configPath := filepath.Join("..", ".golangci.yml")
	adapterRule := requireDepguardRule(t, loadDepguardRules(t, configPath), "product_catalog_persistence_boundary")
	adapterDeny := depguardDenyPackageSet(adapterRule)

	for _, packagePath := range []string{
		"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/product/sourcing",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/product/enrichment",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/product/review/proposal",
		"github.com/henryxu1008/shuomi-ai-commerce/internal/product/image",
	} {
		for _, suffix := range []string{"$", "/"} {
			pattern := packagePath + suffix
			if _, ok := adapterDeny[pattern]; !ok {
				t.Errorf("product_catalog_persistence_boundary must deny sibling Product domain pattern %s", pattern)
			}
		}
	}
	for _, suffix := range []string{"$", "/"} {
		if _, denied := adapterDeny["github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"+suffix]; denied {
			t.Errorf("product_catalog_persistence_boundary must allow its Catalog-owned port pattern ending in %q", suffix)
		}
	}
}

func TestProjectBoundaryListingKitDepguardPatternCoversDomainTrees(t *testing.T) {
	configPath := filepath.Join("..", ".golangci.yml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}

	config := string(content)
	start := strings.Index(config, "      project_boundary_listingkit:\n")
	if start == -1 {
		t.Fatalf("%s must define project_boundary_listingkit", configPath)
	}
	end := strings.Index(config[start+1:], "\n      infrastructure_business_boundaries:")
	if end == -1 {
		t.Fatalf("%s must keep project_boundary_listingkit before infrastructure_business_boundaries", configPath)
	}
	config = config[start : start+1+end]

	for _, packagePath := range []string{
		"amazon",
		"product/asset",
		"product/catalog",
		"product/review/proposal",
		"product/image",
		"infra",
		"integration",
		"marketplace",
		"platform",
		"product/sourcing",
		"publishing",
		"shein",
		"temu",
		"workspace",
	} {
		for _, pattern := range []string{
			fmt.Sprintf(`- "**/internal/%s/*.go"`, packagePath),
			fmt.Sprintf(`- "**/internal/%s/**/*.go"`, packagePath),
		} {
			if !strings.Contains(config, pattern) {
				t.Errorf("%s must cover project-boundary package files with %s", configPath, pattern)
			}
		}
	}

	for _, suffix := range []string{"$", "/"} {
		pattern := fmt.Sprintf(`- pkg: "github.com/henryxu1008/shuomi-ai-commerce/internal/listingkit%s"`, suffix)
		if !strings.Contains(config, pattern) {
			t.Errorf("%s must deny ListingKit package-tree pattern %s", configPath, pattern)
		}
	}
}

func TestWalmartDepguardRegistersOnlyFrozenAdapterPorts(t *testing.T) {
	rules := loadDepguardRules(t, filepath.Join("..", ".golangci.yml"))
	cases := []struct {
		name         string
		files, ports []string
	}{
		{"walmart_official_transport", []string{
			"**/internal/integration/walmart/client.go",
			"**/internal/integration/walmart/products.go",
		}, []string{"$gostd", "github.com/google/uuid$", "github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/walmart$", "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/walmart$"}},
		{"walmart_effects_persistence", []string{
			"**/internal/integration/persistence/listing/submission/walmart_effects_policy.go",
			"**/internal/integration/persistence/listing/official/walmart_results.go",
		}, []string{"$gostd", "github.com/google/uuid$", "gorm.io/gorm$", "gorm.io/gorm/clause$",
			"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/record$",
			"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/submission$",
			"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission$", "github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/walmart$", "github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection$"}},
	}
	for _, tc := range cases {
		rule := requireDepguardRule(t, rules, tc.name)
		if rule.ListMode != "strict" || len(rule.Files) != len(tc.files) || len(rule.Allow) != len(tc.ports) {
			t.Fatalf("%s must keep its exact strict file/port allowlist: %+v", tc.name, rule)
		}
		files, ports := stringSet(rule.Files), stringSet(rule.Allow)
		for _, file := range tc.files {
			if _, ok := files[file]; !ok {
				t.Errorf("%s missing exact file %s", tc.name, file)
			}
		}
		for _, port := range tc.ports {
			if _, ok := ports[port]; !ok {
				t.Errorf("%s missing exact port %s", tc.name, port)
			}
		}
	}
	for _, tc := range []struct{ name, exclusion string }{
		{"infrastructure_business_boundaries", "!**/internal/integration/walmart/client.go"},
		{"infrastructure_business_boundaries", "!**/internal/integration/walmart/products.go"},
		{"listing_submission_persistence_boundary", "!**/internal/integration/persistence/listing/submission/walmart_effects_policy.go"},
		{"supply_official_uow_persistence", "!**/internal/integration/persistence/listing/official/walmart_results.go"},
	} {
		files := stringSet(requireDepguardRule(t, rules, tc.name).Files)
		if _, ok := files[tc.exclusion]; !ok {
			t.Errorf("%s must delegate the exact adapter to its strict rule", tc.name)
		}
	}
}
