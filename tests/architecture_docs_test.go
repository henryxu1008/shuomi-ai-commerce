package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestPluginCaptureSessionReturnDocumentKeepsNavigationOnlyOwner(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "plugin-capture-session-return-v1.md"), []string{
		"Independent Architecture", "IMPLEMENTATION_READY", "chrome.storage.session", "TRUSTED_CONTEXTS",
		"ReturnSession", "pending | bound", "returnState: none | available |", "application's publication authority",
		"No payload/title/images/prices", "No storage read error or", "before awaiting",
		"never proof", "browser restart", "No merge, retained instance/plugin update",
	})
}

func TestSupplyCompletionDraftDocumentPreservesOriginalAuthorityAndExplicitRollout(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "supply-unbound-completion-drafts.md"), []string{
		"IMPLEMENTATION_READY", "PD-SUPPLY-UNBOUND-COMPLETION-2026-10-12", "2667:1948",
		"internal/listing/completiondraft", "supplyCompletionDrafts", "RuntimeCapabilities union",
		"listing_completion_draft_records", "listing_completion_draft_heads", "listing_completion_draft_commands",
		"CAS", "COMMIT", "UNKNOWN", "org/actor/member", "live owner", "64 KiB",
		"另行取得具体授权", "保留三表/所有记录", "不生成 Store merchant/rules",
	})
}

func TestSheinImportDocumentBindsExistingOwnersAndExplicitConfirmation(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "completion-shein-import-v1.md"), []string{"Independent Architecture", "IMPLEMENTATION_READY", "SourceItem", "baseRecordId", "rulesHash", "merchantHash", "selection", "supply.manage", "expectedRevision=Rm", "FindTargetCommand", "409", "UNKNOWN", "不发布", "不在 SQL 事务内重复官方网络请求"})
}

func TestOwnProductDraftEntryReusesFixedSourceAndExplicitSaveContracts(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "own-product-draft-entry-v1.md"), []string{
		"Reuse Existing Architecture", "BatchSelectionInput.ItemIDs", "expectedRevision", "先于确认资料",
		"不自动执行Save", "UNKNOWN", "retained-source", "原owner安装合同", "不引入新的能力开关",
	})
}

func TestOwnProductExistingDraftsDocumentKeepsBoundedReadAndNavigationAuthority(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "own-product-existing-drafts-v1.md"), []string{
		"Independent Architecture", "ListForCollectionItem", "org/actor/member", "2MiB", "keyset",
		"不建反向映射或第二事实源", "不新增每行IAM", "导航按钮不能授权", "不是“自动过期/不可用”",
		"原sourceID", "不接受完整return URL", "只列已明确保存", "UNKNOWN", "不代签用户/生产验收",
	})
}

func TestStoreObservationsPreparationDocumentKeepsExecutionClosed(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "store-observations-preparation-runtime.md"), []string{
		"IMPLEMENTATION_READY", "preparationOnly=true", "zero", "configured official applications",
		"Available and CanSync remain false", "refuses any existing row", "No schema/role changes occur in serving",
		"observations-only", "does not run Store base migrations", "No previous initializer is rerun",
		"Rollback", "revoking only", "leave all four tables and their data intact",
		"Real SHEIN/model/payment and final user business acceptance remain NOT_RUN",
	})
}

func TestCommerceToolBoundaryDocumentsDefineNeutralRegistryOwnership(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "project-target-architecture.md"), []string{
		"internal/commercetool",
		"Tool Definition", "Schema", "Registry", "Agent Allowlist", "Invocation Policy", "Tool Audit Port",
		"kernel/module.Registry", "startup contribution collection", "not the Tool Runtime",
		"Service/Query ports", "composition root",
		"read", "compute", "propose", "write", "publish",
		"depguard: commercetool_boundaries",
	})
	requireDocumentPhrases(t, filepath.Join("..", "docs", "development", "repository-structure.md"), []string{
		"internal/commercetool",
		"Framework", "Transport", "Workflow", "Persistence", "Provider SDK", "Marketplace Client",
		"domain implementation packages",
		"depguard: commercetool_boundaries",
	})
}

func TestProductAgentRuntimeDocumentPreservesOwnerAndExecutionLimits(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "product-agent-runtime-contract.md"), []string{
		"internal/agent", "SDK DTO 留在 Integration", "Store 必须原子 claim/CAS",
		"GovernedModel", "target platform", "2 MiB", "64 KiB",
		"不清空既有 History/候选/校验", "不提交超限 Record", "fake",
	})
}

func TestConcreteImportsStayWithinCurrentOwnerAllowlist(t *testing.T) {
	targets := []string{
		"task-processor/internal/core/logger",
		"task-processor/internal/platform/logging",
		"task-processor/internal/platform/database",
		"task-processor/internal/platform/redis",
		"task-processor/internal/platform/queue/rabbitmq",
		"task-processor/internal/platform/workerpool",
		"task-processor/internal/integration/openai",
		"task-processor/internal/integration/geminiimage",
		"task-processor/internal/integration/grsai",
		"task-processor/internal/integration/s3",
		"task-processor/internal/integration/httpimage",
	}
	importersByTarget := directInternalImporterMap(t, targets)
	// These are the same current owner exceptions previously recorded in the
	// runtime consumer document. New dependencies still require explicit review.
	allowed := map[string]map[string]bool{
		"task-processor/internal/platform/logging":      {"task-processor/internal/core/config": true, "task-processor/internal/pkg/appenv": true},
		"task-processor/internal/integration/httpimage": {"task-processor/internal/imageagent": true},
	}
	for _, target := range targets {
		expected := map[string]bool{}
		for _, importer := range importersByTarget[target] {
			if !isPhase2IntentionalConcreteWiring(importer) {
				expected[importer] = true
			}
		}
		for importer := range expected {
			if !allowed[target][importer] {
				t.Errorf("unapproved concrete import %s -> %s", importer, target)
			}
		}
		for importer := range allowed[target] {
			if !expected[importer] {
				t.Errorf("stale concrete import allowance %s -> %s", importer, target)
			}
		}
	}
}

func isPhase2IntentionalConcreteWiring(importer string) bool {
	for _, prefix := range []string{
		"task-processor/internal/app",
		"task-processor/internal/platform",
		"task-processor/internal/integration",
	} {
		if importMatchesPrefix(importer, prefix) {
			return true
		}
	}
	return false
}

func requireDocumentPhrases(t *testing.T, path string, required []string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for _, phrase := range required {
		if !strings.Contains(string(content), phrase) {
			t.Errorf("%s must mention %q", path, phrase)
		}
	}
}

func directInternalImporterMap(t *testing.T, targets []string) map[string][]string {
	t.Helper()
	cmd := exec.Command("go", "list", "-f", `{{.ImportPath}}|{{join .Imports ","}}`, "./internal/...")
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list internal importers: %v\n%s", err, out)
	}
	result := make(map[string][]string, len(targets))
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		importer, imports, ok := strings.Cut(strings.TrimSpace(line), "|")
		if !ok {
			continue
		}
		for _, target := range targets {
			for _, imported := range strings.Split(imports, ",") {
				if imported == target || strings.HasPrefix(imported, target+"/") {
					result[target] = append(result[target], importer)
					break
				}
			}
		}
	}
	return result
}

func TestTemporalBoundaryDocumentDefinesStableReviewRules(t *testing.T) {
	path := filepath.Join("..", "docs", "architecture", "temporal-boundaries.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	required := []string{
		"# Temporal Boundaries",
		"project-boundaries.md",
		"repository-wide entrypoint",
		"HTTP API",
		"service facade",
		"workflow runtime",
		"RabbitMQ",
		"concrete Temporal worker bootstrap",
		"TestTemporalSDKImportsStayInRuntimeAndOrchestrationAdapters",
		"TestTemporalRuntimePackagesDoNotImportHTTPAPI",
		"depguard: temporal_runtime_httpapi",
		"Review Questions",
	}
	for _, phrase := range required {
		if !strings.Contains(string(content), phrase) {
			t.Errorf("%s must mention %q so Temporal changes have a stable review boundary", path, phrase)
		}
	}
}

func TestProjectBoundaryDocumentKeepsRouteRegistrationInOwningHTTPAPI(t *testing.T) {
	path := filepath.Join("..", "docs", "architecture", "project-boundaries.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	text := string(content)
	if !strings.Contains(text, "| API route registration | owning module `internal/*/httpapi` first; `internal/app/httpapi` only for shared runtime aggregation |") {
		t.Fatalf("%s must keep new route registration owned by module httpapi packages, with app/httpapi limited to shared aggregation", path)
	}
}

func TestProjectBoundaryDocumentDefines1688SourceHandoffOwnership(t *testing.T) {
	path := filepath.Join("..", "docs", "architecture", "project-boundaries.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	normalized := strings.Join(strings.Fields(string(content)), " ")
	for _, phrase := range []string{
		"internal/product must not import internal/listingkit, internal/compatibility, internal/crawler, or internal/integration",
		"internal/integration/acquisition/a1688/browser and internal/integration/acquisition/browsercollector produce current internal/product/sourcing SourceEnvelope values",
		"internal/product/catalog", "internal/product/asset",
		"PD-GREENFIELD-NO-LEGACY-MIGRATION-2026-09-08",
		"does not authorize legacy profile reuse, migration or cutover",
	} {
		if !strings.Contains(normalized, phrase) {
			t.Errorf("%s must mention %q", path, phrase)
		}
	}
	if strings.Contains(normalized, "issue30-clean-slate-cutover.md") {
		t.Errorf("%s must not retain the superseded clean-slate execution entrypoint", path)
	}
}

func TestCurrentDocsDoNotReferenceRetiredFrameworkEntrypoints(t *testing.T) {
	checks := map[string][]string{
		filepath.Join("..", "docs", "development", "repository-structure.md"): {
			"task/                    # 已废弃的 legacy polling 入口",
		},
	}
	for path, banned := range checks {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, phrase := range banned {
			if strings.Contains(string(content), phrase) {
				t.Fatalf("%s still references retired framework entrypoint %q; keep current docs aligned with deleted runtime/task-RPC compatibility surfaces", path, phrase)
			}
		}
	}
}

func TestHTTPAPIAssemblyBoundaryDocumentDefinesCurrentOwnership(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "httpapi-assembly-boundaries.md"), []string{
		"cmd/* -> internal/app/httpapi -> internal/*/httpapi -> domain service/runtime/contracts",
		"composition root", "route registration", "不导入 `internal/app/httpapi`", "RunOptions",
		"具体权限与原作用域", "denied/unavailable", "不把 DDL", "depguard: domain_app_httpapi_boundaries",
	})
}

func TestArchitectureReviewChecklistReferencesExistingDocuments(t *testing.T) {
	checklistPath := filepath.Join("..", "docs", "architecture", "architecture-review-checklist.md")
	checklistContent, err := os.ReadFile(checklistPath)
	if err != nil {
		t.Fatalf("read %s: %v", checklistPath, err)
	}

	reviewReferences := markdownSection(t, string(checklistContent), "## Review References")
	const referenceRule = "Every review reference must resolve to an existing repository document"
	if !strings.Contains(reviewReferences, referenceRule) {
		t.Errorf("%s Review References must state %q so review entrypoints cannot drift into dead links", checklistPath, referenceRule)
	}

	assertMarkdownReferencesExistingDocuments(t, checklistPath, reviewReferences)
}

func markdownSection(t *testing.T, content, heading string) string {
	t.Helper()

	start := strings.Index(content, heading)
	if start == -1 {
		t.Fatalf("markdown content must include heading %q", heading)
	}

	section := content[start+len(heading):]
	if nextHeading := strings.Index(section, "\n## "); nextHeading != -1 {
		section = section[:nextHeading]
	}
	return section
}

func normalizedArchitectureDocReferences(t *testing.T, content string) []string {
	t.Helper()

	referencePattern := regexp.MustCompile("`([^`]+\\.md)`")
	matches := referencePattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		t.Fatalf("markdown content must include at least one doc reference")
	}

	references := make([]string, 0, len(matches))
	for _, match := range matches {
		reference := match[1]
		if !strings.HasPrefix(reference, "docs/") {
			reference = "docs/architecture/" + reference
		}
		references = append(references, reference)
	}
	return references
}

func architectureReadmeExplicitReferences(t *testing.T, content string) map[string]bool {
	t.Helper()

	references := make(map[string]bool)
	for _, reference := range normalizedArchitectureDocReferences(t, content) {
		name := filepath.Base(reference)
		if !strings.Contains(name, "*") {
			references[name] = true
		}
	}
	return references
}

func TestArchitectureReadmeCoversEveryArchitectureDocument(t *testing.T) {
	path := filepath.Join("..", "docs", "architecture", "README.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	explicit := architectureReadmeExplicitReferences(t, string(content))
	documents, err := filepath.Glob(filepath.Join("..", "docs", "architecture", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, document := range documents {
		name := filepath.Base(document)
		if name != "README.md" && !explicit[name] {
			t.Errorf("%s must be explicitly indexed in %s", name, path)
		}
	}
}

func TestSelectedApplicationOverviewDocumentPreservesReadOnlyOwnerBoundaries(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "selected-application-overview-v1.md"), []string{
		"IMPLEMENTATION_READY / FROZEN", "Service.Choice", "submit_selected",
		"MemberID 属于当前", "不跨 owner 直接读", "2000", "并发最多4", "总请求10s",
		"不返回局部计数或假0", "scope 必须完全一致", "无 mutation intent",
	})
}

func TestSelectedApplicationMultiProductDocumentPreservesIndependentOwnersAndOriginalIntentRecovery(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "selected-application-multi-product-v1.md"), []string{
		"Independent Architecture", "IMPLEMENTATION_READY / FROZEN", "SupplyMarket", "submit_selected",
		"useResourcePending", "navigator.locks", "1 MiB", "journal 在整批处理期间不可变",
		"每件仍提交一个 submit_selected", "每件 Repository.Execute 仍在自己的同库事务",
		"不新增后端批次 API/表", "刷新/重启页面：不自动 POST", "404 仅表示未读到回执，不证明未执行",
		"完全相同键/载荷", "同一 member navigator lock", "无新公开URL", "不使用保留实例真实上传/提交/审批/发布",
	})
}

func TestSupplyMarketChoiceEligibilityDocumentPreservesQualificationBoundaries(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "supply-market-choice-eligibility.md"), []string{
		"IMPLEMENTATION_READY", "not_optimized", "incomplete_product", "sealed selection",
		"ReadEffective", "PublicProjection", "create_official_draft", "submit_selected",
		"不提交申请", "正向流程仍为 NOT_RUN",
	})
}

func TestArchitectureReadmeReferencesExistingDocuments(t *testing.T) {
	readmePath := filepath.Join("..", "docs", "architecture", "README.md")
	readmeContent, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read %s: %v", readmePath, err)
	}

	assertMarkdownReferencesExistingDocuments(t, readmePath, string(readmeContent))
}

func assertMarkdownReferencesExistingDocuments(t *testing.T, sourcePath string, markdownContent string) {
	t.Helper()

	for _, reference := range normalizedArchitectureDocReferences(t, markdownContent) {
		name := filepath.Base(reference)
		if strings.Contains(name, "*") {
			continue
		}

		targetPath := filepath.Join("..", filepath.FromSlash(reference))
		if _, err := os.Stat(targetPath); err != nil {
			t.Errorf("%s references %s, but it must resolve to an existing repository document: %v", sourcePath, reference, err)
		}
	}
}

func TestAppAssemblyBoundaryDocumentDefinesStableAssemblyVocabulary(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "app-assembly-boundaries.md"), []string{
		"repository-wide entrypoint", "default ownership and dependency direction", "Assembly Vocabulary",
		"build / initialize", "register", "start", "coordinate", "`bootstrap` builds and registers",
		"`runner` starts and supervises", "`consumer` assembles and coordinates", "domain packages cannot import app",
		"without DDL", "Review Questions",
	})
}

func TestCommercialWalletBillingContractLocksCanonicalOwners(t *testing.T) {
	path := filepath.Join("..", "docs", "architecture", "unified-base-prepaid-resources-v1.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	text := string(content)

	for _, required := range []string{
		"internal/ledger/money",
		"internal/commercial/billing",
		"internal/ledger/orgresource",
		"server-owned",
		"commercial_order_item",
		"Idempotency-Key",
		"RECONCILIATION_REQUIRED",
		"workbench.commercial.read",
		"workbench.commercial.purchase",
		"workbench.commercial.wallet_topup",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("%s must lock %q", path, required)
		}
	}

	for _, forbidden := range []string{
		"Figma example prices never seed production",
		"browser never receives a generic positive-credit API",
		"Invoice creation is deliberately not authorized by this contract",
	} {
		if !strings.Contains(text, forbidden) {
			t.Errorf("%s must retain safety boundary %q", path, forbidden)
		}
	}
}

func TestCurrentStoreDesignPreservesGreenfieldOwnersAndUnopenedServices(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "store-center-current-application-v1.md"), []string{
		"IMPLEMENTATION_READY", "全新系统产品基线", "storeCenter.quotaDatabase", "commercialOwnerDatabase", "AuthPolicyCurrentIdentity", "LiveWrite", "record_status", "pending_activation", "store_center_runtime", "store_quota_runtime", "Activate/Renew/Reactivate", "EXTRACT", "RETIRE",
	})
}

func TestDataServicesDesignPreservesOriginalScopeFundsAndDelivery(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "data-services-native-authorization.md"), []string{
		"PD-DATA-SERVICES-NATIVE-IAM-2026-10-10", "IMPLEMENTATION_READY",
		"本轮不新增本地企业业务停用功能", "nil status port", "deadline checks",
		"No schema, durable fact owner", "always-allow adapter", "actor-private",
	})
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "data-services-v1.md"), []string{
		"IMPLEMENTATION_READY", "PD-DATA-SERVICES-2026-10-09",
		"成功保存一条消费一个 DATA_ROW，5 分/条", "创建者和企业", "撤权/离职后失效",
		"不注入伪造的全站", "Resource DB 保持独立", "failed_fenced", "UNKNOWN",
		"挑战/不支持结构有明确原因，停止相应请求", "DeliveryAuthority", "same-transaction批次回执",
		"线下确认规格与报价", "#619负责公共导航/runtime", "合并、关单、共享/生产部署、真实数据/付费provider均NOT_AUTHORIZED",
	})
}

func TestCockpitDesignPreservesManualFactsAndCurrentStoreOwner(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "operations-cockpit-v1.md"), []string{
		"IMPLEMENTATION_READY / FROZEN", "人工", "创建人", "operations_cockpit", "同一DB transaction", "UNKNOWN", "不能按天", "JS安全", "NOT_RUN",
	})
}

func TestStoreObservationDesignPreservesReadOnlyPlatformAndScopedFacts(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "store-center-platform-observations-v1.md"), []string{
		"IMPLEMENTATION_READY / FROZEN", "PD-STORE-CENTER-READONLY-COMMERCE-2026-10-09",
		"internal/storecenter/observations", "只读句柄仅暴露 Products/Orders/OrderDetails/Track",
		"父命令回执", "比较checkpoint revision", "terminal状态和head原子提交", "单调generation",
		"原成员", "allowlist", "queryType=1/queryOrderType=4", "page≤50", "NOT_RUN", "Legacy decision: RETIRE",
	})
}

func TestStoreCenterFigmaDesignPreservesCurrentOwnersAndUserThresholds(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "store-center-figma-v1.md"), []string{
		"IMPLEMENTATION_READY", "tg48P46SSXl6TBy9lZwg63", "没有默认阈值", "负责人只保存 canonical active member ID",
		"quantity <= threshold", "返回冻结结果", "runtime只读 schema/权限preflight", "不创建新 database",
		"基础目录不受 observations 500 店上限裁剪", "平台观察 generation", "NOT_RUN",
	})
}

func TestSupplyMarketDesignPreservesDisclosureAndOriginalExecutionOwners(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "supply-market-v1.md"), []string{
		"IMPLEMENTATION_READY", "Independent Architecture", "硕米平台账号",
		"同 Product UoW", "verified global platform authorizer",
		"SendPermit", "UNKNOWN", "productsourcing.NewMarketReceiver",
		"productsourcing.ReceivePOD", "运行构造只Verify", "不下单、不付款、不删除",
	})
}

func TestEcoservicesDesignPreservesOriginalFundsAndPrivateDomainOwners(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "ecosystem-services-v1.md"), []string{
		"IMPLEMENTATION_READY", "internal/ecoservices", "internal/commercial/billing", "money",
		"SERVICE_PURCHASE", "NON_COMMISSIONABLE", "CUSTOMER_ACCEPTED", "AR1", "UNKNOWN",
		"CurrentIdentityWithVerifiedRoles", "LiveWrite", "全新", "原订单", "私有",
	})
}

func TestEcoservicesQualificationDesignKeepsFinancialAdmissionSeparate(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "ecoservices-qualification-runtime.md"), []string{
		"IMPLEMENTATION_READY", "ecoservices.nonPaymentOnly=true", "application_submit",
		"application_review", "application_reject", "agreement_accept", "before repository Apply/replay",
		"ACTIVE plus MerchantID", "No Commercial/Money purchase service", "incompatible retained facts fails startup",
		"ecoservices_runtime", "NOT_RUN",
	})
}

func TestPlatformSupportDesignPreservesPersonalHistoryAndFactOwners(t *testing.T) {
	requireDocumentPhrases(t, filepath.Join("..", "docs", "architecture", "platform-support-chatwoot.md"), []string{
		"IMPLEMENTATION_READY", "ZITADEL/Auth.js", "Chatwoot owns", "No second support store",
		"platform_support_subject", "Every list/read/send/download", "No automatic retry",
		"No widget SDK", "No specialized reproduction, additional reviewer", "Actual app-ZITADEL-user acceptance",
	})
}
