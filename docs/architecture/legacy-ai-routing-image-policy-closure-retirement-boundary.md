# #301 旧 AI 路由与图像策略支线完整退休边界

> Design Basis: **Independent Architecture**。状态：**DESIGN_ONLY / SOURCE_NOT_STARTED / NOT_READY**；只允许设计交付，适用独立设计复核及用户确认完成后才可标记 IMPLEMENTATION_READY。<br>
> 基线：`main@f6d678d38b364c767734bc301fc9b16eddd6f349`（2026-10-27）；#301 保持 OPEN。<br>
> 本轮三个设计文档的编辑属于 N/A / Ready（仅文档）；不代表下述生产代码已获开工授权。

## 1. 用户结果、范围与当前权限

使用者是维护当前 AI、图片处理、采集及独立 SHEIN 登录/Store 同步的开发者和运营者。本批消除三个已无生产接线的旧支线，让维护者只沿现役 owner 查找调用与安全边界；不新增业务入口，不声称本批解锁了新功能，也不将退休作为内部试用的新前置。

完整闭包一次覆盖：①旧通用 AI PolicyRouter、Legacy/Shadow 选择与旧执行计划；②旧通用 HTTP 图片生成 adapter；③旧本地场景模板、排版/图层及白底/IP 风险启发式。专属合同、配置、测试、检查和文档尾部一起处置，不按文件数或内部模块机械拆 PR。

用户本轮仅同意**先做设计**。当前唯一 writer 在既有独立 worktree `issue301-browser-tail-946/task-processor` 的 `codex/issue301-ai-image-policy-retirement-design` 分支工作；没有另派实现者。生产源码、测试、配置、完整 inventory、workflow、依赖、保护规则均未修改。设计确认也不自动授权合并、部署或关单。

Out of Scope：当前文本/图片 provider、AI 点数/账本与费率管理、Agent/Tool、ImageAgent Service/Temporal/UNKNOWN/审批、平台规则、前端/BFF、采集/原浏览器/debug、PoolManager deadline/取消、SDS/POD、独立 SHEIN 登录与 Store 同步。不得操作既有实例、业务数据、凭据、文件或进程，不读取已有 DSN/token，不创建运行资源，不增加内部试用前置。

## 2. Design Basis 与前批 KEEP 的替代

依据：[全新系统基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[当前架构边界](project-boundaries.md)、[模型与费率管理](model-route-pricing-management-v1.md)、[文本中立执行](product-agent-text-provider-neutral-v1.md)、[图片智能体](product-image-agent-v1.md)。当前 Figma/Product Authority 仍是 [final-ui-ia-authority](../product/final-ui-ia-authority.md)；本批无 UI/IA/页面/交互改动，无新 Figma 增量。

分类采用 Independent Architecture：涉及三个混合包的公共合同和 selective extraction，未来完整矩阵 44 路径，不能用“未调用”代替 owner/JSON/持久化核对。没有新增状态机、持久化事实、授权或外部副作用。

| 原依据 / KEEP | 本批拟准确替代 | 继续有效的边界 |
| --- | --- | --- |
| #874 水印退休设计中 `product/image/heuristics.go` / AssessIPRisk KEEP | 该评分器现仅自身测试调用；其白底/IP 启发式及专属 audit/risk 值退出 | 不把文字含 watermark 当删除依据；当前图像生成、真实 Review/确定性校验和批准不变，metadata 纯规则抽取保留 |
| #923/#953/#958 的当前 AI/图像总体 KEEP | 仅 §5 明列的闲置路由/HTTP adapter/本地模板支线 | 当前中立文本执行、OpenAI/GRSAI/百炼/Google 等实际调用、调用记录/用量/计费、当前图片合同不撤销 |
| 旧 AI Capability / ProductEnrich integrity 设计与 Phase3 的“移到新包后保留” | 旧 PolicyRouter/Planner/fallback/Shadow 与本地 scene profile 的后续开工要求退出 | 合格现役合同由当前 owner 保留，不恢复 ProductEnrich/ProductImage Task、兼容 alias、旧运行宿主或第二事实源 |
| #977 ImageAgent 保留边界 | 不修改其 Service、Temporal、HTTP 当前入口、现役 worker/WithWorker canary | 真实授权、预算、发送/UNKNOWN、恢复、资产批准与原有效断言完整保留 |

本表只有在用户确认本设计并满足设计准入后，才成为本批实施的替代依据。它不是提前宣布源码已退休。其它前批有效 KEEP、历史批准/失败/数字/勾选不改。

## 3. 实际消费者与 owner

调查覆盖基线全部 3128 个 tracked Go 文件：标准 Go AST 解码 import/selector，再逐项核对包内调用、方法/接口、测试和装配；文本搜索另核对 Go 以外的配置/前端/命令和文档。不是以 grep 零命中或测试绿色推定退休。

| 闭包 | 当前实际证据 | 处置与当前 owner |
| --- | --- | --- |
| AI PolicyRouter / ModelCatalog / PolicyResolver / ModelDefinition / TenantModelPolicy | 构造与 Decide 仅 `aicapability/routing_test.go`；RouteRequestContract/ExecutionPlan 仅同包旧测试；无 App、CLI 或 provider 注入 | RETIRE 旧支线。现役 routeconfig → currentapplication/组织凭据 → `integration/aicapability/einomodel.Executor` 继续实际解析/授权/准入/一次发送 |
| ParseRoutingMode / LegacyClients | parser 仅自身测试，Legacy/Shadow 选择及有序 fallback 列表只在旧路由/计划闭包 | 删除可执行选择，不以新 parser 或 facade 替代；账本 RoutingMode/CacheStatus 的有效值合同归 ledger_contract.go |
| `httpimage.ProductImageAdapter` | New 构造仅自身五个测试，接口实现断言不是生产注入；worker 实际构造 OpenAI adapter，GRSAI 同步编辑沿现役 provider factory | RETIRE 自定义 endpoint/token/subject/white/scene HTTP 请求链与专属 DTO/helper；不撤销实际 provider 支持 |
| 同文件 InspectGeneratedArtifact | `app/pod/inputs`、productsourcing/source_media、SupplyChain 四个 image_probe、worker artwork/generation_output、OpenAI adapter、SDS mutations 共 10 个生产文件直接使用 | EXTRACT 到同一 `integration/httpimage/artifact_inspection.go`；公开包/名称/签名/error identity/格式注册不变，消费者零改 |
| SceneProfile/ResolveSceneProfile/BuildScenePlan/BuildSceneLayout/图层 | 无生产 consumer，只有 scene_test；唯一 YAML 由该旧 scene.go embed | 整退配置/模板/内容分槽/几何/图层 helper。当前 policycatalog/marketplace.imagepolicy 的 SceneOptions 与 MergeSceneOptions 是另一条真实路径，保留 |
| IsWhiteBackground / AssessIPRisk / ImageAudit / IPRiskAssessment | 只有 heuristics_test；不存在调用该评分器的当前审核/批准门禁 | RETIRE 启发式及专属值/helper，不把它当成现役审核或平台合规 owner；真实 Reviewer/ValidateReview/平台规则保持 |
| NormalizeGenerationMetadata | `product/image/ports.go` 验证 Candidate 时使用 | 两纯函数移到同包 generation_metadata.go；字段、限制、排序、冲突拒绝和 error 保持 |

两个不能按旧名字删除的实际合同：`CapabilityProductEnrichText = "productenrich.text_understanding"` 仍由现役 einomodel.capabilityFor 写调用账本；`CapabilityProductImageScene` / `OperationProductImageReview` 仍由图片 Review 记录使用。它们保持原值，不做字段更名或数据改写。

HTTP 公共下载还有大量当前消费者：`ValidatePublicHTTPSURL`、`NewPublicImageHTTPClient`、`Download`、错误分类/体积上限、非公有地址和 redirect 检查均在 client.go，整文件零改。CURRENT inventory 的 `httpimage → imageagent` 仍由 `imageagent/asset_catalog.go` 的实际 import 支撑，其余三条 CURRENT 边也不受影响，所以完整 inventory 零改。

## 4. 状态、持久化、权限与副作用边界

### 4.1 唯一 owner 与失败行为

本批没有 schema/迁移/表/索引/缓存/文件持久化变更，不创建或删除任何已有记录。InvocationRecord 及 GORM row 的全部字段、tag、类型和读写/规范化保持；RoutingMode 和 CacheStatus 只搬声明，不改它们在原记录中的 string 表达。RouteOutcome 类型和现役 active 值不变；删除未使用的旧常量不禁止 reader 读取原始 string，不增加 fallback、迁移或双读。

routeconfig 发布版本/当前组织凭据、文本 quote/claim/限流/绝对 deadline/最后发送 gate、资源预留与用量结算仍由现役 owner 执行。图片 reservation/GenerationIntent/dispatch/proof/UNKNOWN/recovery/显式批准仍由 ImageAgent/Commercial/Asset 原 owner 执行。失败、响应丢失、重试、重启、取消、并发和撤权不因本批新增接管者；不修改对应算法或永久断言，不执行 provider 请求。

### 4.2 JSON/hash/规则与抽取约束

三处原 owner 内的纯声明/规则抽取，不建立新领域或 wrapper：

1. `aicapability/ledger_contract.go`：RoutingMode string、RoutingModeActive 原值；CacheStatus string 及 NotApplicable/Hit/Miss 原值。InvocationRecord 字段与其 wire/ORM 合同原样；不搬其它 ledger 行为。
2. `httpimage/artifact_inspection.go`：InspectGeneratedArtifact 原函数体逐字复用，保留 JPEG/PNG/WebP blank-import 注册与 `productimage.ErrOutputValidation` identity。仍为 header/config 检查，不将它伪称完整图片解码或安全准入；现役完整校验与 SSRF/大小门禁不变。
3. `product/image/scene_options.go` 保留 SceneOptions 原十字段、MergeSceneOptions 及四个私有 helper（normalizedSceneOptions/preflightSceneOptions/validateSceneOptions/mergeSceneOptionValue）；`generation_metadata.go` 保留 NormalizeGenerationMetadata/preflightGenerationMetadata。上述五加二纯函数体、字段/tag/归一化/原始输入限额/防 alias/排序与冲突拒绝保持。

没有兼容 alias、转发、fallback 或重新导出。保持当前请求/响应、Temporal workflow/queue/ID、JSON/hash/版本、已保存载荷和读取拒绝；不因为有旧名称而更改事实含义。CacheStatusMiss 是账本值词汇，保留不意味着恢复旧缓存/执行计划。

SceneOptions 与 GenerationMetadata 仍在当前 product/image owner，不把未调用的模板布局整体移到别处保活。当前 SceneRequest.ProfileName、场景参数、风格引用和 metadata Values 均不删字段；旧 embed YAML 不是当前 Marketplace policycatalog，不误删当前质量阈值/默认策略。删除 AssessIPRisk 不意味着图片侵权/平台合规已通过，也不新增替代启发式或人工流程。

### 4.3 Must / Threat Model / Accepted Risk

Must：当前能力与原授权/企业隔离/凭据保护；一次发送和 UNKNOWN 不自动重试；账本/点数/预算/事务不变；有效 JSON/hash/版本；共享图片下载 SSRF/大小/格式检查；当前采集、PoolManager deadline/取消、独立 SHEIN 登录与 Store 同步。复用现有 Threat Model 和 Accepted Risks，不新增侧信道、通用审计平台或验收工具门槛。

旧 router 的跨 provider fallback、旧 HTTP adapter 的 endpoint/token、模板布局算法不是当前 Must；它们的实现细节测试随退休删除。仍有效的输入限额/纯规则/下载安全/鉴权/计量/取消断言按 §6.1 保留，不能借退休降低现役安全要求。

## 5. 完整删除 / 抽取 / 保留矩阵

### 5.1 混合生产与测试的精确雕刻

| 工件 | RETIRE / 允许变更 | KEEP |
| --- | --- | --- |
| aicapability/model.go | ModelFeature/两个 feature 常量、ModelPricing/ModelDefinition/Supports、TenantModelPolicy；三旧 Enrich Vision/Listing/Fusion capability；旧 ProductImage SceneGenerate/SubjectExtract/WhiteBackground 三 operation 与八个 ProductEnrich operation；仅因此无用 time import | Capability/Operation 基础类型，三个现役 capability、六个当前文本 operation 与 ProductImageReview 的名称/原值 |
| aicapability/invocation.go | 只删 RouteOutcomeLegacy/ShadowDecided/ShadowRouteError 三闲置常量 | 全部原类型、字段、tag、函数/方法、接口和 active 常量原样 |
| product/image/model.go | 只删 ImageAudit、RiskLevel/Low/Medium/High、IPRiskAssessment 整声明 | 所有其它字段/类型/常量/注释，包括 SceneRequest/ProfileName 与 Review |
| aicapability/model_test.go | 原函数只保留实际 CapabilityProductEnrichText 值断言，删退休 operation/feature/capability 的保活断言；从 routing_test.go 原样迁入 TestCategoryOf 整函数，仅补标准 errors import | 原现役 capability 原始字符串断言，以及 TestCategoryOf 的 nil/普通错误/CapabilityError 三分支全部原断言 |
| aicapability/productimage_scene_test.go | Scene 合同测试删 SceneGenerate 一项；ModelOperation 合同测试删 SubjectExtract/WhiteBackground 两项 | 两原测试及 CapabilityProductImageScene/OperationProductImageReview 原值与失败断言 |
| aicapability/usage_settlement_test.go | TestObservedFailedReviewSettlesActualTokensWithoutChangingOrdinaryFailure 中旧 SceneGenerate 符号改为 `Operation("productimage_scene_generate")` | 同一负例原 string、settle/不 settle、错误和其它函数逐字保持 |
| aicapability/store/gorm_invocation_recorder_test.go | TestObservedFailedReviewRequiresExistingDispatchedReview 中同一符号改为 `aicapability.Operation("productimage_scene_generate")` | 原 dispatched/身份冲突/无结算/原行数等全部断言及其它函数逐字保持 |

### 5.2 44 路径冻结清单

路径相对仓库根；一条路径只计一次。实施不得以“完整闭包”为由临时扩表；发现确需范围外修改，先给具体证据及准确修订。

**整文件退出（14；混合文件中的有效声明按 §4.2 先抽取）：**

- `internal/aicapability/routing.go`
- `internal/aicapability/routing_test.go`
- `internal/aicapability/route_request_contract.go`
- `internal/aicapability/execution_plan.go`
- `internal/aicapability/execution_plan_test.go`
- `internal/aicapability/routing_mode.go`
- `internal/aicapability/routing_mode_test.go`
- `internal/integration/httpimage/product_image_adapter.go`
- `internal/integration/httpimage/product_image_adapter_test.go`
- `internal/product/image/scene.go`
- `internal/product/image/scene_test.go`
- `internal/product/image/presets/scene_profiles.yaml`
- `internal/product/image/heuristics.go`
- `internal/product/image/heuristics_test.go`

**混合生产/测试（7）：**

- `internal/aicapability/model.go`
- `internal/aicapability/invocation.go`
- `internal/product/image/model.go`
- `internal/aicapability/model_test.go`
- `internal/aicapability/productimage_scene_test.go`
- `internal/aicapability/usage_settlement_test.go`
- `internal/aicapability/store/gorm_invocation_recorder_test.go`

**当前 owner 抽取叶子（7）：**

- `internal/aicapability/ledger_contract.go`
- `internal/integration/httpimage/artifact_inspection.go`
- `internal/integration/httpimage/artifact_inspection_test.go`
- `internal/product/image/scene_options.go`
- `internal/product/image/scene_options_test.go`
- `internal/product/image/generation_metadata.go`
- `internal/product/image/generation_metadata_test.go`

**原退休 guard 仅追加本批检查（1）：**

- `tests/listingkit_backend_retirement_test.go`

**当前文档（5）：**

- `docs/architecture/legacy-ai-routing-image-policy-closure-retirement-boundary.md`
- `docs/architecture/README.md`
- `docs/refactoring/legacy-register.md`
- `docs/architecture/project-boundaries.md`
- `docs/architecture/product-image-agent-v1.md`

**历史文档只加准确 supersession 顶注（10；起始 main 完整原 blob 后缀逐字保留）：**

- `docs/architecture/legacy-watermark-config-closure-retirement-boundary.md`
- `docs/architecture/legacy-tenant-prompt-closure-retirement-boundary.md`
- `docs/architecture/legacy-openai-adapter-closure-retirement-boundary.md`
- `docs/architecture/legacy-idle-provider-bootstrap-closure-retirement-boundary.md`
- `docs/superpowers/specs/2026-08-06-ai-capability-agent-platform-design.md`
- `docs/superpowers/plans/2026-08-06-ai-capability-phase-1.md`
- `docs/superpowers/specs/2026-08-22-productenrich-governed-execution-integrity-design.md`
- `docs/superpowers/plans/2026-08-22-productenrich-governed-execution-integrity.md`
- `docs/superpowers/specs/2026-09-01-internal-target-architecture-phase3-product-design.md`
- `docs/superpowers/plans/2026-09-01-internal-target-architecture-phase3-product.md`

当前仅设计三文档改动：本文件、architecture/README 与 Legacy Register。未来其它41路径均未修改；旧工件仍在，不能标 SOURCE_RETIRED。

### 5.3 必须零改的保留面

`aicapability/routeconfig`、governed_text、model_point/usage_settlement 生产逻辑、错误合同、store/企业 Resource 仓库/schema；httpimage/client.go/client_test.go、OpenAI/GRSAI/百炼/Google 实际 provider、worker/runtime 与前端；product/image/ports.go/source_edit/errors/boundary_guard 及全部其它原测试；Marketplace policycatalog/imagepolicy、ImageAgent/Temporal/现役 WithWorker canary；所有来源采集、当前原浏览器/debug、PoolManager deadline/取消、SDS/POD、Store、独立 SHEIN 登录及同步。保留现有公开包根，不全局禁用 aicapability/httpimage/product/image。

完整 inventory（CURRENT/历史/数字/ceilings/检查）、所有 workflow、`.golangci.yml`、依赖、保护规则零改。不运行 go mod tidy，不顺手清理其它未引用声明。未入表的永久 guard、测试和生产函数原样。

## 6. 实施顺序与必要验证

用户确认和适用设计复核后，在同一主要 PR：①冻结真实 main/消费者并先抽取 §4.2；②追加一个本批 guard，旧支线仍存在时获得真实 RED；③按 §5 收齐删除及纯值合同/测试/文档尾部，原 guard GREEN；④必要原测试及当前入口仅编译，最终候选原 CI 和一次源码最终复核。不新建工具、runner、POC 或运行装配。

### 6.1 原检查的精确处置

| 原检查 | 本批转换 | 保持的实际 Must |
| --- | --- | --- |
| routing_test.go 五 PolicyRouter 测试及专属 stub；routing_mode_test.go parser 测试；execution_plan_test.go 三测试 | 随旧执行链整退；routing_test.go 内的通用 TestCategoryOf 不属于该退休项，按下行迁到原矩阵内 model_test.go | 现役 routeconfig/einomodel 的组织授权、准入/一次发送/无发送释放/UNKNOWN/deadline 及计费测试零改，旧测试不再指定另一套路由 owner |
| routing_test.go / TestCategoryOf | 原整函数迁入已在44路径内的 aicapability/model_test.go，仅补标准 errors import，不新增路径 | nil→空类别、普通错误→ErrorUnknown、CapabilityError→原类别三项断言和输入字面量逐字保留；生产 errors.go/CategoryOf 及全部其它错误合同不变 |
| model_test.go / TestProductEnrichTextCapabilityContracts | 仅删 §5.1 明列退出值的保活断言 | 当前 capability 字符串不改；无全文件跳过 |
| productimage_scene_test.go 两原函数 | 精确按 §5.1 减去三旧 operation | 当前 Review/Scene capability 原值仍显式校验 |
| usage_settlement_test.go / TestObservedFailedReviewSettlesActualTokensWithoutChangingOrdinaryFailure | 仅替换一个符号为相同 typed string | 原非 Review usage-observed-failed 不结算/错误断言逐字保留 |
| store/gorm_invocation_recorder_test.go / TestObservedFailedReviewRequiresExistingDispatchedReview | 仅替换同一符号为相同 typed string | 原已有 dispatched、身份冲突、零行/零结算等全部断言逐字保留 |
| scene_test.go 前四个 TestMergeSceneOptions* | 原四测试整函数搬到 scene_options_test.go | 原始/聚合限额、归一化、去重、override、防输入 alias 全部保留；其它旧 profile/layout 测试及两专属 helper 退休 |
| heuristics_test.go 五个 TestNormalizeGenerationMetadata* | 原五测试整函数搬到 generation_metadata_test.go | 全部原 metadata 限额/冲突/输入副本断言保留；前四个白底/IP 测试及只供其使用的 import 退休 |
| product_image_adapter_test.go 五 HTTP adapter 测试及三个专属 fixture | 退休 endpoint/payload 行为测试；PNG fixture 可原样移到 artifact_inspection_test.go，补当前纯校验对 JPEG/PNG/WebP/非法输入的直接合同测试 | 现役 worker generation_output、SSRF/重定向/大小、SDS/来源/POD/Review 及原安全断言零改；新测试不调用网络/provider |
| tests/listingkit_backend_retirement_test.go | 只追加一个 TestLegacyAIRoutingImagePolicyClosureIsRetired，复用原 AST/tracked-file helper，原函数/算法/skip/期望全不动 | 旧14路径缺席；按具体 owner 限定旧声明禁入，拒绝 alias/转发；现役 type/纯函数与格式注册必需；缺失/解析错误失败，不 skip，不全局 ban 通用同名类型 |

新增 guard 同时检查三当前包根存在、§4.2 叶子的完整实际声明、retired 值/旧文件不复生、当前 operation/capability 和保留字段形状。该 guard 是原测试文件内的有界补充，不另建检查器。原 `TestAICapabilityModuleUsesOnlyApprovedDependencies`、Phase3 Product/Integration import guards、depguard、CURRENT inventory、准入检查及全部其它函数保持。

未来必要自检：本批完整 guard、原 Architecture/Admission/Import 检查；aicapability（含 ledger/store）、product/image、httpimage 原测试及实际 worker 受影响测试；`cmd/current-application`、`cmd/image-agent-temporal-worker`、`cmd/shein-login-worker` 和现役 Amazon/1688 debug 入口仅编译，入口存在性先核对，不启动。原 PR CI 必须绑定准确 HEAD 完整 run；运行/真实 PostgreSQL/Temporal/provider/用户验收单列 NOT_RUN，不用编译或纯 fixture 代替。

若原 CI/真实测试暴露本表外旧 KEEP，先检查同根 sibling paths，再提出准确例外；不得 missing-skip、放宽 parser/AST/import/安全检查、更新 ceilings、绕门禁或默认重跑。

## 7. 设计复核、交付与停止点

设计交付只三个文档。PR 维护准确 HEAD、实际 diff、必要文档检查、原 CI 与独立复核证据；CI green、自动 review completed 或作者自检均不能写成用户验收。当前不改变源级或运行级状态，不关 #301。

独立设计复核只核对 §1–§7、真实消费者、前批 KEEP 替代、三处抽取/44路径矩阵、持久化/JSON 与 §6.1；不重开无关架构、不跑运行环境、不增加平台。按项目 finding 分类和最多两轮正常设计评审收敛；真实 Must 未满足不交付实施。**未经用户本轮授权不默认新增 AI 互验线程**；GitHub 原自动复核启动失败也不当作审查通过，不反复触发。

设计完成并通过适用独立复核后，交用户确认本设计及 §6.1，才记录 USER_CONFIRMED / IMPLEMENTATION_READY 并实施。源码完成后一次独立最终交付检查，新增源码复核授权与设计复核分开；只复核真实变更/具体 finding 增量，不重复未变代码。合并须最终准确 HEAD 的另行授权；合并后只核对实际 merge SHA 的 event=push/branch=main 整 run 终态。

不宣称新业务能力、生产可用或 #301 完成。部署/既有实例、数据、凭据、文件、进程及真实平台操作始终 NOT_AUTHORIZED；保留独立 SHEIN 登录与 Store 同步，不增加内部试用前置。
