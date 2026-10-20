# 旧 Task 诊断与回归验收工具：完整退休闭包

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

> Status: **用户已确认冻结设计 `ddc6414b0` 并授权按 §5/§6/§6.1/§7 在 #844 整批实施；准确源与专属尾部已退出，待原 CI、最终独立交付检查及合并授权。** Design Basis: **Reuse Existing Architecture**。调查/实施开工基线 main `4c5f5b16fa1a3afc05566b0f80538a05f99ab4e2`（2026-10-20）。#29/#301 Ready 均读回后才修改可执行源；本文矩阵为用户确认的冻结设计，准确实施 HEAD、CI、检查和授权证据维护在主要 PR，源码退休不等于运行或数据退出。

## 1. 用户结果、范围与停止点

使用者是维护当前应用、内部试用、采集、Store 及独立 SHEIN 登录/同步的开发和运维人员。结果是从仓库退出仍指向已退休 Task 表、旧定价缓存和 preview/readiness API 的诊断/回归入口，并撤销其直接说明对当前版本的错误使用要求，避免把失效工具当成当前能力或试用前置。

没有证据表明当前核心用户操作正在等待本批；这是用户明确要求的源码退休维护，不新增功能或验收平台。一个主要 PR 连续完成两个入口、专属 helper/测试/样例、旧查询帮助和直接验收说明，不按文件、语言或工具机械拆批。历史报告、原失败/阻塞/关闭事实保留；源码退出不证明任何现存环境、数据库或远端调用方已经停止。

本次不做：旧数据迁移/删除、替代 Task API/兼容层、当前 BusinessTask 或定价模型改造、通用 DB 安全框架、prompt replay 退休、商业政策/生产门禁变更、workflow/inventory/依赖调整、工具执行、运行实例/凭据/报告操作及内部试用新增前置。UI/IA/Figma N/A，当前产品 authority 不变。

## 2. Product Authority 与 Design Basis

依据：

- [全新系统产品基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)、[Issue 驱动派工](../engineering/issue-driven-development.md)。
- 已批准的 [#824 数字刊登支撑退休](legacy-numeric-listing-support-closure-retirement-boundary.md)、[#827 SHEIN Publishing 运行闭包](legacy-shein-publishing-runtime-closure-retirement-boundary.md)、[#839 运维源码闭包](legacy-listing-operations-closure-retirement-boundary.md)：Task/Workspace、旧执行/持久缓存及运维执行 owner 已退出；本批关闭遗留的两项诊断入口，不重新打开这些 owner。
- 用户要求完整退休闭包、先设计后实施，保留当前能力、独立 SHEIN 登录/Store 同步，不操作实例或数据、不改两套 workflow、不增加试用前置。

**Reuse Existing Architecture**：完全复用上述已批准退休边界，不改变任何当前领域合同、事实 owner、授权/租户、状态机、事务、恢复或外部副作用协议。没有新增 schema、持久化、provider 请求或跨系统安全边界。预计修改不足 30 个范围文件，且不新增生产业务代码；不是以拆片规避敏感阈值。独立检查只核对实际闭包和保留矩阵，不重新评审全局架构。若发现必须改变上述不变量，停止该部分实现并按 AGENTS.md 升级准入。

Must：准确关闭旧合同的完整源闭包；当前能力/安全断言不降级；实际资产不动；历史证据不重写。Should：复用已有 guard/文档位置，避免新增规则 owner。Out of Scope 如 §1。沿用既有阶段决定，不新接受安全风险；真实旧实例、表、缓存、私有清单及外部操作者是否仍存在为 **UNKNOWN**，UNKNOWN 不能被写成不存在、成功或可清理。

Threat Model 仅覆盖本批误删当前能力、误把旧合同重新当作当前入口、错误宣称运行/数据已退休以及误降低现有商业/安全门禁的风险；不借源码维护扩建授权、审计、验收或诊断平台。当前收费/外部客户/生产发布政策仍有效，生产保持 **NOT_READY**；内部试用和生产放行不能混同。

## 3. 实际消费者与完整闭包

| 根入口 | 静态调用与合同 | 最后消费者 / 专属尾部 |
| --- | --- | --- |
| `tools/listingkit-pricing-cache-inspector/main.go` | 独立 tools 模块 CLI，显式参数/内联默认 DSN → GORM/Postgres；读 `listing_kit_tasks` 的旧 request/result、`shein_resolution_cache_entries` 的 pricing key/最近行；打印旧 SKU/定价/提交时间线 | 未找到当前生产装配、脚本或 workflow 调用；内联旧 request/result/package/pricing DTO 与 key/hash、稳定标识、SKU alias、金额/币种默认及 JSON 输出 helper 仅服务此诊断。没有独立 helper/test 文件；code-health 旧保留行是直接说明尾部 |
| `scripts/listingkit-regression-dataset-check.ps1` | JSON schema_version=1 清单 → task_id/SHEIN/expected_preview_ready/expected_submit_mode 校验；普通模式读 token 并 GET `/api/v1/listing-kits/tasks/{task_id}/preview?platform=shein`，比较旧 readiness/final_review | 自有 Pester 测试、示例清单、validation runs README 为直接消费者；helper 全内联。`ValidateOnly` 只验证该旧清单，不构成当前产品能力；脚本及专属测试/样例一起退出 |
| `tools/db-query/main.go` 的 `printUsage` | 四个旧 `listing_kit_tasks` 示例；实际显式表查询能力仍通用 | 只替换帮助中的旧表/Task 示例。连接、flags、过滤、字段、limit、JSON/CSV/table 与错误输出保持；`main` 可能在处理自定义 `--help` 前连接 DB，不通过运行 CLI 获取帮助 |
| `docs/product/validation/**` 与旧 Temporal PoC 指引 | template、runs README、QA failure samples、unknown tracker 仍要求旧 Task/Workspace/preview/readiness/submit attempt 证据；`temporal-poc-runbook.md` 仍称 current PoC，指向已删 API/worker、preview/events/refresh/publish 与旧测试；`temporal-workflow-evaluation.md` 将 2026-05-18 PoC 结果及旧 submit 的保留/下一步写为当前建议 | §6.1 精确处置当前适用声明与失效命令；dated runs、tracker 各行、PoC 结果和旧结论全部保留，不新造替代验收模板/Temporal PoC |

静态搜索覆盖 tracked 的 `internal/cmd/tools/hack/scripts/tests/.github/docs/config/deployments`，查完整入口名及旧表/API/清单字段。没有把命名匹配等同“无人使用”：`internal/app/httpapi/server_test.go` 中的旧路由是拒绝已退休 API 的安全回归，必须保留；`hack/debug/replay-sale-attribute` 使用通用 debug JSON 和当前 OpenAI client，TaskID 仅为调试元数据，不依赖旧 Task 表/cache/preview，属于本批之外的独立 provider 诊断生命周期。

独立复核后补查同根因的旧命令/preview/events/refresh 与“current PoC”说明：除上述 runbook，还须窄标注其 sibling evaluation 的历史身份。通用 `start/stop-temporal-dev`、当前 Temporal 合同/SDK/worker 与当前 image/agent 编排不依赖旧 Task submit，原样保留；`httpapi-assembly-boundaries.md` 的旧例子是已有装配原则的上下文，不是本批诊断/回归执行指引，不整体历史化或重写稳定边界。dated 旧 run 和离线校验 input-binding 设计中的旧 API 例子保留其历史/拒绝合同语境，不据此恢复路由。

**架构问题**：旧执行/持久化 owner 已按批准闭包退休，两个可执行工具及其当前验收说明却仍消费旧合同，通用工具帮助也继续暗示旧表可用。根因是源合同退休未收口到诊断和说明尾部；修复是删除完整专属闭包并校准当前适用身份，不修补旧表查询、不恢复 preview、不复制旧缓存 key 到当前价格领域。

## 4. 持久化、配置与副作用 owner

| 边界 | 原所有者及行为 | 退休处置 / 当前保留 |
| --- | --- | --- |
| 旧任务与定价缓存 | inspector 仅 SQL 读取两个旧表，解码旧 JSON 并 stdout；DSN/task IDs/limit 由 CLI 配置 | 删除此读取/解释源合同；无继任持久化 owner。实际表/行/JSON/缓存不查、不改、不 drop；不把缓存历史值迁入当前价格 |
| token 与清单 | 回归脚本普通模式读 `LISTINGKIT_API_TOKEN` 或 `.local/listingkit-api-token.txt`、仓库外 JSON；默认示例为占位 task ID，不是受控真实 fixture | 只删除 tracked 示例和专属读取/比较源。真实 token、环境变量、清单、日志/报告不定位、不读取、不清理；不建立新 token 管理或新验收清单 |
| HTTP / 平台 | 回归脚本 GET 旧 preview；不创建任务、保存草稿或 publish。inspector 无平台调用 | 两入口在本批均不执行，包括 ValidateOnly。当前平台 API、执行许可、计量、远端回执、retry/UNKNOWN/恢复和不可安全重复副作用 owner 不变；不重放旧请求 |
| 当前定价 | 当前 `internal/shein/pricing` PriceManager 及关联 calculator/API、`internal/publishing/shein` 仍有效定价值/规则拥有业务行为 | 原样保留；不抽取诊断用 key/alias/hash/default、不复活后续批次已退休的旧 TEMU 计算链。历史 PriceHandler/PriceCalculator 保留表述只按已批准当前消费者边界解释 |
| 通用 DB / prompt replay | db-query 是显式 operator 查询工具；replay-sale-attribute 使用当前 OpenAI client/config 并可产生 provider 调用和本地调试输出 | 原能力/凭据/日志与安全责任保持；replay 源/测试、OpenAI/logger 当前消费者行保持。没有工具执行或 provider 消费证据 |
| 采集、Store、独立 SHEIN | 原生采集/workerpool、queue/Retry、Store 仓库/观察及独立登录 worker 的账号/Profile/Cookie/attempt/Store 同步 owner | 源、合同与物理资产原样；不迁入新 Organization、不新建 ID bridge/同步/fallback，不增加内部试用前置 |

当前身份/权限/tenant、资金/额度/计费、平台合同、AI/SKU/spec/prompt/图像、规则与共享基础层保持。没有新事务、状态转移或恢复执行者；对失败、响应丢失、重试、并发、重启、取消的当前责任完全沿用原 owner。本批不会把旧诊断输出或布尔 readiness 当作当前接受/发布回执。

## 5. 删除 / 保留 / 抽取矩阵

### 5.1 整批删除的四个准确 tracked 文件

```text
tools/listingkit-pricing-cache-inspector/main.go
scripts/listingkit-regression-dataset-check.ps1
scripts/listingkit-regression-dataset-check.Tests.ps1
docs/product/validation/listingkit-production-regression-dataset.example.json
```

不按 `listingkit`、`task`、`pricing`、`tools` 或 `validation` 目录批删。两个入口的全部专属 DTO/helper 都内联，随入口删除；不迁往新 Service。Pester 两个清单测试只验证该退休 manifest 的接受/拒绝，随合同退休，不为保持旧测试重造接口。

### 5.2 混合文件与必须保留

| 对象 | 准确修改 / 保留边界 |
| --- | --- |
| `tools/db-query/main.go` | 仅 `printUsage` 四条旧 Task 表示例及配套注释换为 `<table>` / `<condition>` / `<columns>` 等明确占位语法；保留参数说明、原输出选项及单个结尾换行。无真实数据库/表名示例，无查询能力变化 |
| `tools/db-query/main_test.go`、`scripts/db-query.ps1` | 全部保留，现有 `TestPrintUsagePreservesSingleTrailingNewline` 直接调用纯 printUsage，不连接 DB；wrapper 已退出旧例，本批不重复修改 |
| `tools/go.mod`、`tools/go.sum`、`tools/tools.go`、`tools/json-simplifier/**`、其它通用 DB/port-forward/备份恢复 | 全部保留，GORM/Postgres 仍被通用 db-query 消费，不因 inspector 退出删依赖或整理锁文件 |
| `hack/debug/replay-sale-attribute/**`、当前 OpenAI/logger inventory | 全部保留；不把任意 TaskID/旧名或独立调试副作用当成此旧 Task 闭包 |
| 当前源码与 guard | 有效平台 API/DTO/规则、共享值/两种 SHEIN 属性、当前定价、AI/SKU/spec/prompt/图像、采集/身份/计费/Store/独立 SHEIN 登录与同步保持；拒绝已退休路由的 HTTP 测试保持 |
| 历史材料 | 所有 dated validation runs、tracker 已有 rows/state/evidence、旧设计/计划、原审计计数保留；不改失败为成功，不把未执行写成已验收 |

### 5.3 EXTRACT 结论

**无新增抽取。** 旧 key/hash/identity/SKU alias、任务结果 DTO 和 manifest/token/readiness 比较只为已取消的旧诊断合同服务；JSON/hash/HTTP/格式化用现有标准库，当前有效定价/校验已在当前 owner。为“保住 helper”新增消费者或抽象会重新引入旧合同。

```text
Legacy decision: RETIRE
Reusable behavior: 当前定价/平台纯规则、显式通用 DB 查询和当前诊断能力复用原 owner；无新增抽取
Current owner: internal/shein/pricing、当前 publishing/shein/API/规则、tools/db-query、当前 OpenAI debug client；Store/采集/独立 SHEIN owner 不变
Cutover/deletion condition: 用户确认本设计，#29/#301 Ready 均记录并读回，§5 四文件/§6.1 尾部精确退休，原有效断言和保留矩阵通过，原 CI 与一次最终独立检查完成；合并另行授权
```

## 6. 验证与实施顺序

1. 设计阶段仅修改本文及 §6.1 的四个索引/清单“待确认提案”说明；运行原 Greenfield/架构目录 guard、相对链接与 diff 检查。不运行两个工具、Pester/ValidateOnly、db-query CLI、replay、真实 HTTP/SQL/provider/发布或报告生成。
2. 用户确认后，把冻结设计、完整范围、owner、不变量和验收写入 #29/#301 当前正文并分别读回 Ready，再修改可执行源。同步一次实际 main，只处理真实组合；一 Writer / 一主要 PR，不派生小 PR。
3. 在既有 `tests/listingkit_backend_retirement_test.go` 的 **`TestRetiredListingOperationsClosure`** 精确追加 §5.1 四个 absence 路径，先记录因四源仍在而 RED；不删除原 23 路径、共享 Secret/namespace/Store/monitoring 及其他有效安全断言。完成四源退出及帮助/文档尾部后 GREEN。不新增扫描器/测试框架。
4. 必要开发自检：上述原 guard、Greenfield/ArchitectureReadme 原目录测试、`go -C tools test ./db-query -run '^TestPrintUsagePreservesSingleTrailingNewline$' -count=1`（纯函数，不调用 main）、改动链接/diff。其它完整检查交原 PR CI，不本地重复整套 CI，不重跑已退休 Pester 来制造 PASS。
5. 对照 baseline 精确 diff：除 §5/§6.1 与上述 guard 外不改任何文件；全部 workflow（包括两套 CI）、inventory/历史计数/ceilings/depguard、module/lock/依赖及保护规则 **零改**。inspector 无 core/logger/OpenAI import，不需要当前消费者表校准；如原 CI 暴露新的冲突，先报告具体最小差异，不能自行放宽本零改边界。

### 6.1 直接文档尾部与旧保留结论

以下是**用户确认后实施**的准确矩阵；设计阶段仅四个索引/清单新增“提案待确认”说明，不预先修改其它文档身份。

| 文件 | 实施时的准确处置 |
| --- | --- |
| `docs/product/validation/runs/README.md` | 新增窄 SUPERSEDED 顶注，指出旧 Task/preview/readiness/submit 的使用/最低验收集不属于当前源码/内部试用准入。删除“可重复回归数据集”节的活动脚本命令、已删示例链接和 token/ValidateOnly 使用要求，替换为基线 Git 历史链接；其余历史目录、命名、run 类型/最低集正文保留并受顶注限定。不能留悬空示例链接或新造替代流程 |
| `docs/product/validation/listingkit-real-api-validation-report-template.md` | 仅替换 2026-09-26 顶注。明确用户确认本设计后，旧 Task 源码链已取消，模板不再是当前仓库/内部试用 CURRENT gate；这准确替代原“必须提供等价模板才能历史化”前置。本批是取消旧流程而非迁移到新实现，不为此创建替代模板。模板字段/正文原样保留，可供追溯既有旧路径事实；远端仍存活与否 UNKNOWN，不宣布服务停用，不撤销其证据或恢复责任 |
| `docs/product/validation/listingkit-qa-failure-samples.md` | 只新增窄历史顶注：旧 Task/workspace/readiness 样例的每版本执行要求不作为当前试用门禁；正文/有效安全原则保留，不把历史样例转换为当前新增验收矩阵 |
| `docs/product/validation/unknown-state-and-blocker-tracker.md` | 只新增窄历史顶注：停止将旧链 tracker 当当前试用必填。全部表行、日期、open/mapped/closed/不会修复、历史接口/关闭证据原样，不借退休宣称 unknown 已解决 |
| `docs/product/README.md` | 仅将“真实接口验收报告模板”一条从当前职责清单移到历史 ListingKit 文档并注明上述准确边界。其余产品/Figma/商业 rollout/恢复 SOP 权威与两段通用职责限定不变，不把整个产品目录历史化 |
| `docs/architecture/temporal-poc-runbook.md` | 只新增窄 SUPERSEDED 顶注：旧 ListingKit Task SHEIN publish PoC 的 API/worker/flags/preview/events/refresh/测试和人工发布恢复清单是历史操作证据，不能用作当前启动、发布或试用准入。历史正文/命令原样，不执行、不复活旧入口/迁移窗口，不新建替代 PoC。当前 Temporal 基础设施/通用 dev 脚本、SDK及当前 image/agent worker owner 不变；远端旧 workflow/实例状态 UNKNOWN，不终止或清理 |
| `docs/architecture/temporal-workflow-evaluation.md` | 只新增窄历史顶注：2026-05-18 旧 submit PoC 的落地结果、“保留当前状态机/下一步迁入”建议不再是当前源能力或派工前置。日期、评估/恢复/幂等事实与正文保留；有效 durable execution 原理复用当前 Temporal owner，不改变当前合同，也不宣称真实旧 workflow 已退出 |
| `docs/architecture/README.md` | 索引本文与其设计/实施状态，精确范围和保留边界；不改其它 owner/当前合同 |
| `docs/refactoring/legacy-register.md`、`module-target-mapping.md` | 新增本批准确 RETIRE 说明，两个旧工具无迁移/替代 Task/cache owner；既有历史矩阵/计数与其它保留结论不变 |
| `docs/refactoring/code-health-decisions.md` | 顶注只替代 inspector 在 tools 分组行中的当前 retained-configuration-specific 适用结论、旧 Pester 问题的当前追修含义。原分组行/失败事实/计数原样，db-query/json-simplifier 仍保留；不调整任何 consumer inventory |

**不变的商业与运行边界**：`listingkit-paid-pilot-product-catalog.md`、`listingkit-paid-pilot-execution-plan.md` 中当前收费/外部准入/entitlement/Go-No-Go/回滚/生产门禁及其中商业证据要求不变；引用 validation/runs 目录仍有效，不能因旧 Task 工具退休声称商业验收已通过或邀请客户。`ops/listingkit-error-recovery-sop.md` 对实际仍存活旧环境的恢复/升级指导原样。新顶注不取消这些独立职责，也不能让其历史 Task 字段成为无关内部试用的新前置。

已标历史的 roadmap、next-execution-plan、backlog、dated superpowers plans 与 dated runs 原样；不得执行其旧 agent/sub-skill/迁移/验收步骤。真实私有 JSON、凭据、输出文件和既有失败资产不在 tracked 删除矩阵中。

## 7. 独立检查、交付与权限

设计检查只读真实源/消费者、持久化与平台副作用 owner、§5/§6/§6.1/§7、原保留边界及模板 CURRENT 冲突的准确取代范围。使用原 GitHub 独立复核；同 SHA 已 Running 不重复触发，不常驻增加实现者/评审线程。finding 先分类，BLOCKER 要命中 AGENTS.md 的具体后果；非 Blocker 不无限重开设计。设计复核和 CI 不是用户设计确认。

设计确认后在同一主要 PR 整批实施，形成稳定候选时再同步实际 main；不持续追 main。一次最终独立交付检查验证实际 diff、原 guard 有效断言、当前能力和历史记录保留及现有证据。修复只复核相关增量。本批已按冻结矩阵完成源码实施，待原 CI 与最终独立交付检查；无运行/数据或用户验收结论。

实施交付以“当前操作者不再被引向已退 Task 诊断/回归；通用查询/当前定价及所有保留能力维持原入口”交接，精确 HEAD、CI、review 写 PR。无新运行入口或要求用户操作旧环境；真实 DB/HTTP/token/provider/用户验收 **NOT_RUN**。实现后待独立检查/用户决定，不能自行签发产品验收、上线或关闭 #29/#301。

合并须用户授权最终准确 HEAD，正常 expected-head 合并，不强推/管理员绕过/改保护。合并后只核实实际 merge SHA 的 **event=push / branch=main / completed / success** CI，不用 PR CI 冒充 main CI。部署、运行/数据/凭据/既有报告操作分别授权；内部试用不新增前置，生产继续 NOT_READY。
