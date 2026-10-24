# #301 旧租户 Prompt 管理与运行整链退休设计

Refs [#301](https://github.com/qq550723504/shuomi-ai-commerce/issues/301)。#301 保持 OPEN，#29 不重开。

- Status: **USER_CONFIRMED / SOURCE_IMPLEMENTED**。用户已确认 §1–§7（含 §4.2/§6.1），在 #923 同批实施完整65工件，不机械拆批。
- Design Basis：**Independent Architecture / IMPLEMENTATION_READY；Issue/PR Admission: Ready**。准确设计原 CI 与本轮独立只读设计复核 PASS、无 finding；[人类确认/接单6380854744](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6380854744)及 Issue/PR Ready 已读回，先于正式源码修改。
- 基线 main **e651da8e8b3e70646c4ac54a00c536a28ba21d81**，tree **84fa36bd5e3c77a76b6fbab5ce93c320f8368ab7**。#921 的 push/main CI40066055622 SUCCESS 只是前批证据。
- 唯一 Writer 原会话/独占 worktree，分支 `codex/issue301-tenant-prompt-closure-design`；[接单及前批当前段逐字归档6380591455](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6380591455)。首次 git fetch 连接失败；GitHub API 核实 main 后使用本地同一 commit 对象，不把失败写成同步成功。准确 HEAD、CI、实际复核与用户决定留主要 PR。

## 1. 用户结果、Scope 与产品依据

使用者是当前 AI、图像、采集和 schema 的维护者；交付结果是旧租户 Prompt 管理/运行、默认租户上下文及专属建表/打包/检查尾部完整退出，现役能力继续消费原 owner。当前没有因此阻塞的核心用户操作；不为父目标清零新增平台、运行验收或内部试用前置。一个主要 PR 覆盖完整闭包，不按文件数、目录或测试证据机械拆批。

Authority 为本轮用户“先 Independent Architecture 设计、确认后实施”的直接决定、[Greenfield 产品基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)及[Issue 驱动规则](../engineering/issue-driven-development.md)。本设计提出准确替代 §2 的旧实现 KEEP；**用户确认前旧 KEEP 不自动失效**。当前 AI/图像仍按 [Product Image Agent](product-image-agent-v1.md) 与现役领域/adapter 合同执行。UI/IA/Figma N/A：旧 Prompt Management 文档列出的页面/BFF 已不在当前树，本轮不改变任何现役页面、导航、接口或开放功能。

Scope：`internal/promptmgmt` 全管理链、`internal/prompt` 全 Registry/loader/watcher/renderer/Store/keys、`internal/shared/tenantctx`、`prompts` 五份 YAML，以及 §5/§6.1 精确专属尾部。旧自由 tenant/header/query/body 选择、`default` 租户、全局回退不转交当前身份或授权 owner。没有新 EXTRACT、wrapper、Prompt Store、IAM、迁移或兼容协议。

Must：完整闭包；当前 AI 准入/模型与凭据选择/预算/计量/Prompt 版本和 hash、ImageAgent/Temporal/Tool 执行与恢复、当前身份/Organization/JSON、来源采集/原浏览器/debug、平台 API/规则、SHEIN 草稿与有效清理/品牌、独立 SHEIN 登录和 Store 同步及 owner/fence/UNKNOWN 保留。workflow/完整 inventory（含 CURRENT、历史、计数、ceilings）零改；不读既有有效配置/.env/凭据，不操作既有实例、数据、文件、进程或平台/provider。

Should：复用现有 owner、标准库、Go AST/import/依赖检查和原任务临时 fixture。Out of Scope：当前 AI Prompt 语义/内容/版本变更、新 Prompt 管理产品、任意模板输入或新模型调用、数据库迁移/backfill/DROP/清理、其它零 importer 的平台 API/集成/localagent 退休、依赖/锁/保护或额外治理工具。Accepted Risks 无新增；实际旧表/覆盖行/运行态 UNKNOWN，运行/产品验收 NOT_RUN。

Threat Model 限误退现役同名 Prompt/身份合同、把旧默认租户或客户端 tenant 字段移植到当前授权、误把表声明删除当成数据删除、为保活历史测试复建旧 store/第二事实源。不得全仓禁用 `prompt`、`TenantID`、模板或版本/hash 字段等通用名字。

## 2. Design Basis、根因与前批 KEEP 替代

根因不是单个空 HTTP 包：旧业务消费者退出后，未装配的 Service/catalog/HTTP、无初始化调用的 Registry/Store、模板资源和 schema `AutoMigrate` 留成互相保活的闭包；旧文档与强制读取检查继续要求其存在。`tenant_id` 为字符串也不能证明其是现役 Organization owner；零 importer 本身同样不能撤销用户 KEEP。

拟实施必须走 Independent Architecture：整体退出旧租户/持久化公共合同、替代显式 KEEP，且准确范围超过 30 工件。设计只界定本轮退出，不重开全局身份、Agent、数据库或 Prompt 架构。仍正确且现役的 AI Prompt 行为已在当前 owner，无须从旧 registry 搬出另一套抽象。

| 前批/旧依据 | 本次确认后仅替代 | 继续有效 |
| --- | --- | --- |
| [Prompt Management](zitadel-tenant-prompt-management.md)、[auth/tenancy](auth-and-tenancy.md)及旧 Agent 平台/目标架构 spec、产品策略与 phase2a plan | 旧 prompt/promptmgmt/tenantctx 的现役管理、运行、租户覆盖与复用/搬迁建议；旧页面路径只是历史 | 当前真实 AI 的 Prompt 版本/hash/审计、Organization/授权、已实现 Agent/Tool/Console；无新 Prompt 平台交付 |
| [#807](legacy-temu-image-api-prompt-retirement-boundary.md)、[#827](legacy-shein-publishing-runtime-closure-retirement-boundary.md)及 TEMU/SHEIN 中间批次 | generic Store/Registry/loader/watcher/render、Service/HTTP/catalog、非当批 keys/YAML 必须保留的准确含义 | 已退平台专属 key/API 不复活；现役 AI/图像/provider 与当前有效规则原样 |
| [#854](legacy-retirement-parent-closure-boundary.md)、[#880](legacy-redis-proxy-shared-closure-retirement-boundary.md)、[#891](legacy-listing-preview-download-closure-retirement-boundary.md)、[#893](legacy-logger-facade-closure-retirement-boundary.md)、[#912](legacy-shein-policy-contract-closure-retirement-boundary.md)、[#921](legacy-task-identity-mapping-closure-retirement-boundary.md) 等广泛 KEEP | 仅本轮准确三包/五资源及其目录/日志/存储/管理保留，不再作为当前业务 owner 或后续抽取目标 | aiidentity/context/roles、authidentity、同名当前 AI Prompt/Tool、有效 shared、平台 API、采集/localagent、Store/SHEIN |
| [旧发布执行闭包](legacy-kubernetes-release-execution-closure-retirement-boundary.md)的 Dockerfile/build 保留 | 仅 `Dockerfile.product-listing-api` 两条 `prompts` COPY | 所有原镜像/base、命令/build/copy/chmod、标签、配置、UI/worker 与安全检查 |
| module-target-mapping、dependency-baseline、历史 phase6 分类 | 旧 Prompt owner 仍需迁入/保留为现役实现的含义 | 原历史行/计数/证据；完整 inventory 文件不改，由本文与 Legacy Register 说明不构成重建义务 |

§5.3 二十七份直接依据仅加窄 superseding 顶注，原完整正文逐字后缀保留；不重写旧批准、失败、勾选、历史数字或执行步骤。旧已有覆盖记录不导入、不禁用、不删除，也不为它们新增兼容读写。

## 3. 实际消费者：contract → implementation → injection → consumer

调查覆盖 tracked Go（含 alias、tests/build tags、`hack/debug` 独立模块）、声明/selector、非 Go 路径/key/路由、原测试、schema 与打包。区分自调用、专属测试、保活检查和实际生产注入。

| 对象 | 基线实际调用/注入 | 决定 |
| --- | --- | --- |
| promptmgmt.Service/DTO/catalog/metadata 与五条 `/api/v1/listing-kits/prompts` 路由 | Service/catalog 只被自身 API/测试消费；`promptmgmt/api` 无包外 Go importer/当前装配；旧页面/BFF 不存在，前端 button 测试的同名示例链接不是产品入口 | 整包含专属测试退休；不挂载旧接口、不搬迁客户端 tenant 来源 |
| PromptRegistry/GlobalRegistry、InitGlobal/SetTenantPromptStore、租户 Get/Render、全局 fallback、loader/watcher/renderer | 实现内部和自身测试消费；生产调用者只剩 promptmgmt 与 schema 类型注册，没有当前入口初始化 Registry/Store 或执行模板 | `internal/prompt` 整退；标准库 `text/template`/YAML/fsnotify 等依赖及锁文件不清扫 |
| TenantPromptTemplate/GormTenantPromptStore | `app/schema/productlisting/runtime.go` 仅 AutoMigrate 类型；没有当前 repository 注入或真实业务读写 caller | 模型/Store 整退，仅退出当前建表接线，不做 DROP/记录操作 |
| shared/tenantctx | 生产 importer 仅 prompt/registry.go、store.go、promptmgmt/service.go、api/handler.go；另有 SHEIN 三测试文件六个原用例设置无效旧上下文 | 整根退休；六测试只去旧 wrapper/import，保留原 context/authorizedbrand resolved 与全部业务断言 |
| 五份 YAML/20 key、catalog schema/变量 glossary | 唯一加载器为无现役初始化调用的旧 Registry；没有当前 AI/图像/采集执行引用；Dockerfile 两次复制仍打包 | 全资源/keys/catalog 同批退；两个 COPY 退出，不能只删 HTTP 后留下闲置运行层 |
| 当前图像 Prompt | worker/capabilities、generation_provider → integration/openai `DefaultProductImagePrompts`/ProductImageAdapter；独立版本 `product-image-prompts-v1` 与 source-white-background 合同 | 全部源码/版本/hash/预算/凭据/效果/恢复 KEEP，不用旧 YAML 替换或重算 |
| 当前文本/Agent/模板 | 当前 product/enrichment、publishing/shein/marketplace 草稿、agentconfig 内容/图像配置、AI/Tool 执行合同和各原 adapter | KEEP；同名 Prompt/Template 是当前数据或执行合同，不属于旧 owner |

原 `shared/aiidentity.Identity`、OpenAI identity、authidentity/effective Organization、ImageAgent/Tool 授权和原源码/测试全不变。独立 sheinlogin/bootstrap/worker 与 storecenter 的 binding/claim/fence/UNKNOWN、登录及固定 Store 同步继续由原 owner 负责。

## 4. 持久化、初始化、授权及副作用边界

### 4.1 旧持久化 owner 与当前事实

`prompt.TenantPromptTemplate`/GormTenantPromptStore 只拥有旧 `tenant_prompt_templates` 行及 tenant/key/content/version/enabled/timestamps；旧表声明和唯一键不是当前授权来源。删除类型/Repository 不把 tenant 字符串映射为 Organization，不创建新表、不迁移内容、不复制为 Agent 配置、不为未知旧记录提供 fallback。

实际旧表是否存在、是否有在途写入或覆盖行 UNKNOWN，仍由原环境/数据 owner 处置；本轮无环境访问、停止或清理授权。当前 AI/图像/Invocation Ledger/Asset/Catalog/Organization/Store 各自原事实 owner 不变，当前 JSON/规范化/hash/版本及读取拒绝不变。没有新事务、幂等、恢复/UNKNOWN、缓存或副作用 owner；不调用模型、平台、浏览器或登录。

### 4.2 当前建表与 Goose baseline 的准确变化

架构现状需要明确：`migrations.go` 的 baseline 版本 **2026083001** 回调委托当前 `AutoMigrateRuntime`；它不是冻结的独立 DDL 快照。移除 Prompt 注册会改变**新建空 schema 的 transitive 建表集合**，不能宣称所有历史 baseline 行为逐字不变。

本设计选择 Greenfield 当前初始化边界：只从 `runtime.go` 移除 Prompt import 与最后 AutoMigrate/error 分支。`migrations.go`/Migrations/RunDB/连接一致性/Goose 版本与登记合同全文件 KEEP，不改已发布版本号、历史记录或 down 行为，不新增迁移步骤、修复/重放旧数据或保留 legacy model 来复现旧 DDL。已有登记的 baseline 不因本轮自动重跑；显式调用 AutoMigrateRuntime 也须对任务合成既有 Prompt 表保持结构/行不变。当前 AI 凭据、Invocation Ledger、ImageAgent、approved assets、Catalog 的 AutoMigrate 调用及次序/错误完整保留。

用原临时 SQLite fixture 证明两件事：空 schema 不再制造旧表；预先合成的旧表/行原样留存。当前 schema 的全部正向检查和 Goose/连接/幂等测试保持。此项新建集合变化是用户确认本文时需要批准的准确边界；不是批准真实数据库操作。若发现当前 Must 要求旧 DDL 精确重放、现役 Store 注入或具体外部可观察依赖，停止对应删除并报告，不自行增加兼容/迁移。

### 4.3 旧租户与当前身份

旧 HTTP 允许 body/header/query/default 选择 tenant；shared/tenantctx 的空值→default、默认租户匹配旧空行、旧 Prompt 的显式全局 fallback 同闭包退休，不转入当前 authidentity/AI。SHEIN 测试只移除未被生产读取的旧 wrapper；`authorizedbrand.WithResolved`、基础 context、合法品牌保留/敏感清理/原 payload 断言不减。其它 owner 的 TenantID/WithTenantID/Template/Prompt 字段不删、不改 JSON 或当前授权。

## 5. 删除、保留与完整尾部矩阵

### 5.1 二十六个整退工件

| 根 | 精确文件 | 数量 |
| --- | --- | --- |
| internal/prompt | keys.go、loader.go、logger_port.go、registry.go、renderer.go、store.go、watcher.go；loader_test.go、registry_test.go、renderer_test.go、store_test.go | 11 |
| internal/promptmgmt | catalog.go、catalog_manifest.go、service.go、types.go、service_test.go；api/handler.go、api/http_module.go、api/handler_test.go、api/http_module_test.go | 9 |
| internal/shared/tenantctx | tenant.go | 1 |
| prompts | productenrich/generation.yaml、llm_scorer.yaml、understanding.yaml；productimage/generation.yaml、studio_generation.yaml | 5 |

退出上述整根、20 个准确 key、DTO、catalog 变量、日志私有 port、旧泛型渲染/文件监听与专属测试；不新建替代模板库或复制无当前 consumer 的纯算法。只处理任务 worktree 源文件，不调用旧 constructor、文件监听、Store 或外部接口。

### 5.2 九个精确 mixed 尾部工件

1. `internal/app/schema/productlisting/runtime.go`：只去 Prompt import/AutoMigrate 分支；其余当前 schema owner 原样。
2. `internal/app/schema/productlisting/runtime_test.go`：仅 §6.1 两原测试转换；原 helper/其它测试不变。
3. `internal/publishing/shein/authorized_brand_test.go`。
4. `internal/publishing/shein/listing_copy_test.go`。
5. `internal/publishing/shein/submit_prep_test.go`：上述三文件只 §6.1 六用例的旧 wrapper/import。
6. `deployments/docker/Dockerfile.product-listing-api`：仅删除 `COPY prompts/ prompts/` 与 `COPY --from=builder /src/prompts /app/prompts`。其余行逐字 KEEP；不 build/run Docker，不扩大到共享镜像/发布退休。
7. `scripts/test-fast.ps1`：仅删除包数组 `./internal/promptmgmt` 一项。其它包、参数、失败传播/边界检查不变；该脚本另有旧 listingadmin 路径，是既有独立残留，不为本轮修复/恢复或把整脚本可用作为新前置。
8. `tests/listingkit_backend_retirement_test.go`：仅 §6.1 两原函数。
9. `tests/import_boundaries_test.go`：仅 `businessHTTPPackages` 的 promptmgmt/api 一个旧允许项。

`migrations.go`、所有 workflow/完整 inventory/.golangci/ceilings/保护、依赖/锁及全部当前源码零改。inventory 中 phase6 历史分类提及旧包仍逐字留存，不能用它恢复旧 owner。原 logger guard 的 Prompt 名称是负向防复活而非存在性要求，整段保持，不顺手重写其它检查。

### 5.3 二十七份直接依据：只加窄顶注

以下文件在实施时只添加指向本文的 superseding 前缀；原完整正文逐字作为后缀，历史数字/勾选/证据不改。只替代旧 Prompt/tenantctx 现役 KEEP、旧复用建议或本文两条打包语义，不能取消其中其它当前能力：

```text
docs/architecture/zitadel-tenant-prompt-management.md
docs/architecture/auth-and-tenancy.md
docs/refactoring/module-target-mapping.md
docs/refactoring/dependency-baseline.md
docs/product/ai-commerce-agent-platform-strategy.md
docs/superpowers/specs/2026-08-06-ai-capability-agent-platform-design.md
docs/superpowers/specs/2026-08-30-internal-target-architecture-migration-design.md
docs/superpowers/plans/2026-05-29-task-processor-framework-phase2a.md
docs/architecture/legacy-amazon-listing-closure-retirement-boundary.md
docs/architecture/legacy-api-token-tools-retirement-boundary.md
docs/architecture/legacy-host-runtime-closure-retirement-boundary.md
docs/architecture/legacy-listing-preview-download-closure-retirement-boundary.md
docs/architecture/legacy-logger-facade-closure-retirement-boundary.md
docs/architecture/legacy-redis-proxy-shared-closure-retirement-boundary.md
docs/architecture/legacy-retirement-parent-closure-boundary.md
docs/architecture/legacy-shein-listing-closure-retirement-boundary.md
docs/architecture/legacy-shein-policy-contract-closure-retirement-boundary.md
docs/architecture/legacy-shein-publishing-runtime-closure-retirement-boundary.md
docs/architecture/legacy-temu-content-handler-retirement-boundary.md
docs/architecture/legacy-temu-image-api-prompt-retirement-boundary.md
docs/architecture/legacy-temu-image-orchestration-retirement-boundary.md
docs/architecture/legacy-temu-image-processing-retirement-boundary.md
docs/architecture/legacy-temu-listing-closure-retirement-boundary.md
docs/architecture/legacy-temu-property-pipeline-retirement-boundary.md
docs/architecture/legacy-temu-text-bullet-retirement-boundary.md
docs/architecture/legacy-task-identity-mapping-closure-retirement-boundary.md
docs/architecture/legacy-kubernetes-release-execution-closure-retirement-boundary.md
```

本文、Architecture README、Legacy Register 三当前文档仅维护本批状态/链接，两个索引原历史完整后缀保留。其它历史 Prompt 名称不要求全仓清扫；完整 inventory 不属于该文档清单。拟实施 **26 整退 + 9 mixed + 27 窄顶注 + 3 当前文档 = 65 工件**，保持同一主要 PR。

### 5.4 KEEP / EXTRACT 结论

**无新 EXTRACT 代码**。`integration/openai/product_image_adapter.go` 的 ProductImagePrompts/DefaultProductImagePrompts/版本/hash 和 worker 注入、aicapability/Invocation Ledger、当前 product/image/enrichment、agentconfig/Agent/Tool、publishing/shein 当前 Prompt 构造/草稿/品牌/清理、各有效平台 API/规则、来源采集/原浏览器/debug、身份/Organization/JSON/SourceIdentity 四槽位/原 hash/读取拒绝、Store、独立 SHEIN 登录和 Store 同步全部原 owner KEEP。不把旧目录中的同名 concept 当成这些合同的实现。

```text
Legacy decision: RETIRE
Reusable behavior: 当前AI/图像/采集/授权/存储/规则已由现役owner提供，原位复用，无新抽取或替代Prompt层
Current owner: 现役Product/ImageAgent/AI adapter/agentconfig/authidentity/Invocation Ledger/Store与独立SHEIN各原owner
Cutover/deletion condition: 本文复核、用户确认§1–§7/§6.1及IMPLEMENTATION_READY/Ready读回；按准确65工件一次收齐，原检查有效，现役能力与数据边界不变
```

## 6. 实施顺序与必要验证

设计阶段只做三文档设计的原索引/相对链接/准确范围检查、原完整 PR CI、适用只读独立设计复核；不执行生产路径或业务测试。确认后同一 Writer/主要 PR 先按 §6.1 扩展原 guard 与原 schema 临时 fixture，在旧目标尚存在时记录准确 RED；再 stage 精确删除、转换六测试旧 context/建表/打包/脚本/旧依据至 GREEN。不以 missing-skip/删有效断言或重建旧模块换绿。

实施必要自检只复用原 schema package（任务自有 `t.TempDir` SQLite）、六个受影响 SHEIN 用例、修改的原 guard/边界 helper、原 Architecture README 检查及当前身份/SourceIdentity JSON/hash/拒绝与独立 SHEIN/Store 原适用合成 fixture。旧专属 Prompt 测试随 owner 退出，不迁移为新 Prompt 框架。保留原八命令依赖检查；编译 current-application、image-agent-temporal-worker、shein-login-worker、source-account-registry-schema-init、product-listing-api-schema-migrate、listingkit-schema-migrate、1688-local-agent、1688-batch-import、1688-public-browser-collector 和 `hack/debug/test-amazon` 独立模块，产物仅任务 temp，全部不执行。

稳定候选用原完整 CI；不人工重复 CI 全量检查，不建 PG/Redis/browser/provider 环境、runner 或故障矩阵。不读真实私密配置/凭据或任何旧记录；原 CI 既有隔离 fixture 不等于环境/产品验收。最终核对准确65路径、27旧文档历史后缀、索引后缀、mixed 文件允许范围及全部 KEEP 零差异。

### 6.1 原检查/测试的精确转换

| 文件 / 原函数或项 | 唯一允许转换 | 其余保持 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go / TestTemuImageAPIAndVisionResourcesAreRetired | 三个旧 metadata 文件强制读取/旧 key 检查转换为 prompt/promptmgmt/shared/tenantctx/prompts 整根 fail-closed Lstat 缺席；在同函数复用 trackedFiles/parser/decodeGoImportPath/importMatchesPrefix 对全部 tracked Go（含 tests/build tags/nested module）禁止三个旧 Go 根/子路径 decoded import | 原 TEMU root API owner-qualified AST/类型/函数/值声明禁入和所有读取/解析错误不弱化；不加全仓通用 Prompt/tenant/key 禁词或 missing-skip |
| 同文件 / TestCurrentCommandsDoNotCompileLegacyBusinessBackends | 原八命令依赖禁入加三个旧 Go 根及子路径 | 原八命令、平台/共享 KEEP、workflow/安全/race 检查、go list 错误全保留；其它函数/global/helper逐字零改，包含 assertLegacyLoggerFacadeRetired |
| tests/import_boundaries_test.go / businessHTTPPackages | 仅删除 promptmgmt/api 的旧 HTTP 允许项 | 其余允许项、收集逻辑、所有测试/global/helper 不变；不动 localagent/api、当前 Prompt/模板 owner 或 depguard |
| schema/productlisting/runtime_test.go / TestAutoMigrateRuntimeLeavesLegacyProductTaskTablesRetired | 原旧表缺席列表加 tenant_prompt_templates | 原 ProductEnrich/ProductImage/Amazon 退出断言与原 fixture 不变；目标存在时准确 RED |
| 同文件 / TestAutoMigrateRuntimeLeavesExistingAmazonTaskTableUntouched | 改名 TestAutoMigrateRuntimeLeavesExistingRetiredTablesUntouched；沿原 helper 将原 Amazon case 原样保留，并增任务合成 tenant_prompt_templates sentinel 表/行 case，校验 AutoMigrateRuntime 前后原结构/行字节不变 | 不查询/创建既有实例，不执行 DROP/历史数据迁移；全部其它函数/Goose版本/幂等/连接不匹配/现役表与列正向断言保留 |
| publishing/shein/authorized_brand_test.go / TestSanitizeDraftPayloadSensitiveContentPreservesAuthorizedBrandFromRuntimeContext | 去 sharedtenantctx import/WithTenantID 外层，直接使用原 authorizedbrand.WithResolved context | 原合法品牌/payload/清理断言与基础context完整保留 |
| publishing/shein/listing_copy_test.go / TestBuildSheinListingCopyCleansSensitiveWords | 去旧 import/wrapper，传原 context.Background() | 词表/文案输入与期望值/所有其它用例不变 |
| publishing/shein/submit_prep_test.go / TestPromptEntryPointsUseBasePromptWithoutLegacyPolicy、TestSanitizeDraftPayloadSensitiveContent_CleansDraftTextFields、TestSanitizeDraftPayloadSensitiveContent_PreservesAuthorizedBrandFromRuntimeContext、TestSanitizeDraftPayloadSensitiveContent_CleansFreeTextAttributesAndSKCNames | 仅去旧 import/四处 WithTenantID 外层；保留内层 Background 或 authorizedbrand.WithResolved | 原 fake AI、基础Prompt/敏感清理/结构化字段/合法品牌/SKC/JSON/有效规则断言全部不减；六用例是现役测试，不随 Prompt 包退出 |
| Dockerfile / scripts/test-fast.ps1 | 仅 §5.2 两 COPY / 一个旧 package 数组项 | 全部其它行/失败传播/编译目标不变，不执行原脚本或构建镜像 |

schema 的当前源码唯一变化是 §4.2 明确的 Prompt 注册退出，不改 migrations.go 的版本、回调或元数据。若原 CI 实证发现表外保活要求、冻结 KEEP 冲突或漏列消费者，先报告具体失败与最小修订，取得用户决定后再应用；“完整闭包”不是任意扩范围或改 workflow/inventory 的授权。

## 7. 准入、复核、交接与权限

### 7.1 设计交付与开工

三文档设计完成适用原检查、准确设计 HEAD 原 CI、一次适用独立设计复核与实际 finding 分类后，交用户确认 §1–§7（含 §4.2/§6.1）。没有待修订设计 BLOCKER 才可准备 IMPLEMENTATION_READY；用户确认及 Issue/PR Ready 读回之前仍只交设计，不修改65工件中的其它源码/测试/资源/旧依据。

### 7.2 实施交付

确认后原 Writer 在同一 PR 连续实施、自检、提交/推送及范围内 finding/CI 修复。最终只读检查完整真实 diff、真实消费者/兄弟路径、现役 AI/身份/JSON/独立SHEIN/Store 保留和本矩阵；修复只复核对应增量。只有命中项目具体 BLOCKER 的新证据重开冻结设计，理论 hardening/无关平台建设不阻止有界编码。实现者自检、CI、另一 AI 复核均不等于运行或用户验收。

### 7.3 评审与授权

同 HEAD GitHub 评审运行时不重复触发；仅明确启动失败后一次受控重试。空正式 reviews/摘要布尔值/标签不是实际 PASS。**未经用户本轮授权不默认新增 AI 互验线程**；此前 #921 设计/源码子代理授权不覆盖本轮。备用只读设计或源码复核先提供准确 HEAD、边界与失败证据，获用户决定后启动，不从旧 plan 派工文字推定权限。

设计阶段仅授权三文档；用户随后已明确确认本文并授权同一 #923 整批实施、必要隔离自检、提交/推送及 PR/Issue 维护。合并、部署、#301 关闭及既有实例/数据/凭据/进程操作仍 NOT_AUTHORIZED；不新增试用前置。#301 OPEN，#29 不重开，真实环境 UNKNOWN，运行/产品验收 NOT_RUN。未来获准确 HEAD 正常合并授权后，只以实际 merge SHA 对应 event=push/head_branch=main 完整 CI completed/SUCCESS 确认，不用 PR CI 或单 Gate 替代。
