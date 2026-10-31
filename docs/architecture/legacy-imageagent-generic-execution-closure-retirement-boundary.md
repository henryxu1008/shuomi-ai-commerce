# 旧 ImageAgent 通用执行与 OpenAI 装配完整退休边界

> Design Basis: **Independent Architecture**；当前 **DESIGN_ONLY / SOURCE_NOT_STARTED / NOT_READY**。用户仅授权先设计。起始 main `fa6a0fbbe68ade4579b21cdefb2b776dc4afb149`；候选前同步 main `f3dbd6a6f0047e128db9f52431489c312018a7e9`（#1031：只与本批 README 重叠，当前消费者/未来闭包无变化，完整保留新增TikTok索引）。本次提交只新增本设计及两个导航，不修改生产、测试、部署、workflow 或 inventory。一个主要 PR，设计确认后整批实施，不机械拆批。

## §1 给使用者的结果与范围

这批清理的是已经取消的 **抠主体 → 渲染 → 模型审核** 通用管线、只为它构造的 OpenAI Manager/客户端/请求池和配置，以及旧通用批准发布尾部。**采集后生成一张主图、当前图组和 Artwork 都保留。** 当前单主图仍借用通用 executor/发布器，不能把它们直接整删；先抽取真实需要的来源编辑、结果构造与单主图批准能力，再切断旧管线。

维护者交付后不再为已取消的默认 Review/通用 OpenAI 客户端配置全局 key；当前来源图编辑仍使用现有组织凭据、已发布路由/单价和唯一 generation owner。用户入口、操作步骤、可用阶段不变，不增加内部试用前置，不宣称部署或产品验收。

| 同一完整闭包 | 本批决策 | 当前保留能力 |
| --- | --- | --- |
| 旧 Extract/Render/Review、多槽位模型策略及 staged-review 独占预算/恢复 | RETIRE，无新发送入口、旧 Activity stub 或兼容 alias | 当前一次来源白底编辑、来源集合编辑、Artwork，原限额/UNKNOWN/取消恢复 |
| worker 宽能力装配 → Manager/Client/RequestPool → 默认 OpenAI YAML/env | RETIRE 最后消费者及其专属尾部 | 当前 GRSAI/百炼调用、组织凭据读写、当前 AI 文本/Eino/路由能力 |
| 旧 ProductImageSlotExecutor、通用 ApprovedAssetPublisherV3 | EXTRACT 最小当前行为后 RETIRE | 当前主图/图组结果、单主图真实批准事务、图组选择发布与 Artwork 发布 |

Out of Scope：新功能/新 UI、provider 接入或路由扩展，资金/资源模型重做，新的 schema/Goose 迁移、事实 owner、恢复/验收平台，既有实例/数据/凭据/缓存/Profile/进程操作，workflow/保护规则调整，部署/运行/关单。#301 始终 OPEN。旧 SHEIN 登录/身份回写已退休，不恢复；Store 观察同步保持。

## §2 Authority、前批 KEEP 与产品边界

依据 [全新安装决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[项目边界](project-boundaries.md)、[当前来源业务闭环](source-business-closure-v1.md)、[当前单主图执行](../engineering/2026-09-25-issue487-main-image-execution-design.md)、[当前图像能力](product-image-agent-v1.md)与用户本轮设计授权。无 UI/IA/页面命名变化，Figma N/A；不改当前 Product/Figma Authority。

| 既有依据 | 仅由本设计确认后替代的 KEEP | 继续有效 |
| --- | --- | --- |
| #994 [手工图像 runtime](legacy-imageagent-manual-runtime-closure-retirement-boundary.md)、#996 [V2 合同](legacy-imageagent-v2-contract-tail-closure-retirement-boundary.md) | 宽能力装配/通用 executor/旧模型 Review 及泛化发布包装 | 当前 organization 入口、canary/WithWorker、图组/Artwork、当前业务协议 |
| #1001 [历史协议](legacy-imageagent-history-protocol-closure-retirement-boundary.md)、旧 core-manual/review 设计 | 仅旧模型审核 Activity/预算子状态/通用多槽位执行的继续供应含义 | 当前单主图 Start/RestartFailed、Update、绝对 deadline、14个 marker/15调用点、当前回放/恢复/取消和 publication fence |
| #953/#958/#983/#1025 及 logger/config 相关旧 KEEP | Manager/Client/Pool/聊天传输、旧 Review DTO、仅服务其最后消费者的默认配置、日志 port | 当前 image-neutral wire、组织凭据/私有秘密、来源图 provider、当前 Eino/AI 和日志安全 |
| #847 及 Secret 样例 KEEP | 只替代 base/secret.example.yaml 的五个 TASK_PROCESSOR_OPENAI 键 | 其余样例键、当前身份/worker 角色、canary、生产 NOT_READY 门禁 |
| Phase2 CURRENT | 用户已批准未来仅删 `integration/openai → core/config` 一行，见下 | 历史/计数/ceilings/其它行/全部准确性算法逐字不改 |

准确 inventory 例外：[提案6542513160](https://github.com/henryxu1008/shuomi-ai-commerce/issues/301#issuecomment-6542513160)及[用户确认6542567777](https://github.com/henryxu1008/shuomi-ai-commerce/issues/301#issuecomment-6542567777)。它不授权现在改 inventory 或正式实施。未来唯一允许删除的是：

```diff
-| `task-processor/internal/integration/openai` | `task-processor/internal/core/config` | application schema | owning domains / Phases 3-7 |
```

Must：当前身份/组织/member/角色撤销与实时授权、单价/额度/资源预留结算、quote 和 credential/configuration version、同键异载荷拒绝、CAS/事务/不可重复发送、UNKNOWN 及当前恢复唯一 owner 全保持。当前 source/edit 的 URL/字节/媒体/大小/引用数/元数据和输出维度边界、原非直通/非 placeholder 规则不变。

Threat Model 不扩大。内部可信 typed port 的原调用约束不新增通用 admission 框架；用户 HTTP/组织边界仍由当前 owner 校验。旧样例/配置/模型审核不是继续供应或迁移义务。外部人工旧调用 UNKNOWN，不据此开发兼容路径。

## §3 实际消费者与删除条件

本基线通过 tracked Go imports/具体构造器、接口方法及调用、脚本/声明/原检查/文档交叉核对，未运行产品命令或读真实凭据。

| 当前实际路径 | 证据位置 | 处理 |
| --- | --- | --- |
| acquisitionImageRoutes → acquisitionMainRunInput → Service.Start → organization workflow | app/httpapi/imageagent_acquisition_module.go:379 固定 product/zz/default/general、一个 main、一个 source、单次来源编辑且不做 Review | 完整保留 HTTP/JSON/ID、预算、来源快照和最终批准；当前单主图先抽取 |
| 当前图组/Artwork → buildFullImageApplication → NewManagedImageSetTemporalDependencies | app/httpapi/image_set_application.go:255–279、worker/image_set_runtime.go | 保留当前 generation/source-edit/artwork owner；结果构造不再依赖通用 executor |
| 单主图 CLI → ResolveImageAgentTemporalDependenciesForMode | worker/dependencies.go:206–247 仍构造完整 Subject/Scene/Reviewer/Manager，但 organizationMainSlotExecutor 实际用 GenerationExecution | 删 BuildAI/BuildOrganizationCapabilities 及宽依赖/完整性前置，保留真实 DB/角色/授权/存储/时限/关闭顺序 |
| generationProviderFactory → GRSAI/百炼 → 单主图白底或 recipe EditSources | worker/generation_provider.go、tools/product_image_executor.go 的 sourceOnlyMain/recipe 分支 | 只抽取两种现役编辑；Artwork 分支保持 |
| 旧 Manager → Client/RequestPool/SDK，ToClientConfigs → BuildAI | integration/openai/{client_manager,client,pool,images}.go；core/config/type_openai.go 是 CURRENT 行唯一 import | 宽装配退出后无生产消费者；不为保活另造引用，一起 RETIRE |
| 组织凭据 → Get/Save/组织专属 Resolve → 当前 published route | openai/credential_store.go、currentapplication/title_credential_provision.go、einomodel/route.go、worker/generation_provider.go | 表/字段/精确作用域/Get/Save 保持；抽最小配置/解析合同，不删整个 openai 包 |
| 当前主图批准 → PublishApprovedV3 → Asset ApprovalCommit | assetpublication/publisher.go；workflow approval publish 分支 | 抽 AcquisitionMainPublisher，保留事务/收据/摘要；旧通用多槽位 gallery 分支退出 |
| canary → 同一 worker CLI -canary → WithWorker | 现有 release-authority runner/guard | 原路径、队列、ack、无 Activity/DB/provider 行为零改 |

Legacy decision: **EXTRACT | RETIRE**。Current owner：当前 ImageAgent generation/result/acquisition-main、Product Image 有效来源编辑、原 Asset/AI credential/Resource/Store 事实 owner。Cutover/deletion condition：适用设计复核、用户确认 §1–§7 并 IMPLEMENTATION_READY 写回读回；先改全部真实消费者，再删旧声明/文件，无 alias/stub/default forwarding/双事实源。找到表外真实消费者先报告，不自动扩大权限。

## §4 最小抽取与所有权

### §4.1 固定当前能力，不搬家整个旧框架

| 新/保留 owner | 精确最小行为 | 必须退出 |
| --- | --- | --- |
| tools.SourceGenerationExecutor | QuoteSlot/GenerateQuotedSlot 仅来源单主图和合法 ImageSet recipe；原 quote/receipt/fingerprint、source bytes SHA、method prompt、返回材料校验 | 通用按 Role 的 extract/render/scene/review、未 quote GenerateSlot 的发送、Review retry 能力 |
| tools.ImageResultBuilder | NewAcquisitionMainResultBuilder(profile) / NewImageSetResultBuilder 两个明确用途，仅纯结果构造；共享 source/meta/map clone、durableCandidateAssetID 原算法 | ProductImageSlotExecutor/Dependencies、旧能力表、宽执行器作为结果 builder |
| imageagent/acquisition_main.go | 原 product/zz/default/general、一个 source/一个 main、非 Set/Artwork 的当前 admission/执行约束；在写入/调度/provider 前拒绝旧宽计划 | 不给旧多槽位 raw plan 新入口；ImageSet 的同名场景角色仍合法，不宽泛禁 RoleScene 等共享值 |
| worker.SourceMainExecutor | 接现有 GenerationExecution 及窄结果 builder；原 organization authorizer/role/关闭/启动校验 | nil generation 的旧 delegate quote fallback、全部 staged-model-review 方法、无效 Manager 初始化 |
| product/image.SourceWhiteBackgroundRequest + capability | Source/SourceBytes/Product/Authorization；保留 SourceOnly=true 原来源编辑、clone、取消/媒体/溯源/生成材料拒绝 | RenderRequest.Subject/SourceOnly 双模式、Subject/Scene/Reviewer ports/请求/专属 DTO/helper |
| openai.SourceImageAdapter / grsai.SourceImageProvider | 当前白底来源编辑、EditSources、QuoteUsage、pin route/credential/version；GRSAI/百炼原一次发送、维度、源图保真、输出下载/预算 | Extract/Scene/Review、review token/observer/hash/prompt、旧通用 SDK request pool/client/logger |
| openai/credential_config.go | ClientConfig 仅 APIKey/Model/BaseURL/APIStyle/Timeout；ResolvedClientConfig.CacheKey、组织专属 ClientConfigResolver 去无消费者 fallback 参数 | ClientConfig.Logger/SDK retry/reference transport字段、全局/成员 fallback；默认 store 仍可按显式scope Get/Save，不承担隐式解析 |
| product/image.PolicySceneDefaults | 当前 profile JSON/hash 需要的原10值字段与 normalize/clone 行为，调用 NormalizePolicySceneDefaults；保留 catalog/threshold/key/version | SceneOptions/双层 MergeSceneOptions 执行选项接口；无新 Scene renderer |
| assetpublication.AcquisitionMainPublisher | 当前一张 main 的 scope/version/完整来源/provenance/摘要/持久化材料证明和同一 ApprovalCommit 事务；共享 helper 归 publication_helpers | 通用 Publisher/ApprovedAssetPublisherV3 及非主图多槽位发布；不动当前 ImageSet/Artwork publisher |

SourceGenerationExecutor 不引入通用调度平台。现有 provider/Goose/Temporal/flag/GORM/测试设施直接复用。当前 route 只接受原已支持的 GRSAI/百炼，不能因删 OpenAI transport 顺带新增/删除已开放 provider。完整提示词/version 常量值保持：当前 `product-image-prompts-v1`、`source-white-background-v1` 及其白底实际 prompt；recipe 自己的 PromptVersion/来源 prompt 不变。不要保留旧四项 prompts struct 只为取得 Version，抽当前窄常量。

凭据的物理 `ai_client_credentials`、表角色/列/索引、tenant/user/client_name 精确 Get/Save 和 current credential reference/version 不变。不迁表、不重编号、不改已有行，不新增 vault/bridge；当前文本/Eino route 继续按原 GetCredential 读取。`ResolveClientConfig` 只保留当前 organization-only 路径，成员优先/静态 global Manager fallback 一起退出；共享 aiidentity 的实际 current identity 原样消费，不保留旧 WithIdentity 包装。

### §4.2 JSON/hash/历史/事务的明确边界

1. 当前采集 HTTP、source identity/snapshot、StartRunInput/Plan/Slot/AssetCatalog、生成 quote/result/approval receipt 的合法载荷、排序与 hash **逐字等价**，版本不递增。只因源码换 type/文件不改变线协议；对完整合成 fixture 固定原 JSON/quote fingerprint/candidate ID/result digest/approval key 验证。新 PolicySceneDefaults 的10字段、nil/empty 与 profile 序列化形状保持；catalog YAML/version 零改。
2. `PublishApprovedV3` 的 Go 通用 DTO/方法/port 名退出，当前窄名改为 PublishAcquisitionMain/Input/ActivityInput；**现役 Temporal Activity 字符串 `imageagent.publish_approved.v3`、payload字段/JSON/调用顺序和 publication key 保持**，这是当前单主图实际 wire，不是注册旧别名或 stub。ImageSet/Artwork 发布路径及 ack/回执查询不动。原两个事务边界仍是 Image 持久化 publication-start → Asset 原原子 commit/immutable receipt → Image persist-complete；响应丢失仍查原收据，空查找不证明未提交。
3. 当前 generation、staging、publication、UNKNOWN、deadline/cancel/fence、恢复通知及14个 SDK marker/15调用点不变。只退出已取消的模型 Review Activity/独占 reserve/settle/release/unknown 子操作、Review retry 分支及专属 phases/codes/ports。**旧模型审核历史、非当前宽 raw 多槽位计划不再承诺续跑**；不注册旧审核 Activity 来返回假成功，不为其 reserve/release/重发或自动清算。用户确认本设计才确认这项退休边界；真实历史完全不读取/操作。
4. 物理 `review_usage_json` 列和 `store.records.go`/schema/grants/Goose **零改**。当前域 snapshot 的 `ReviewUsage` 是 nil/null：以 `[]json.RawMessage` 保留现行零值 JSON 槽位，非空或旧审核 phase 在现有 decode/validate 边界拒绝，不能解码旧 Review DTO 后继续，也不能静默丢字段再写回。`ReviewActionID` 当前为 omitempty 零值；可退出其 Go 字段而保持当前合法 JSON，旧 Activity 不再注册。其它当前字段/usage nil与empty/hash 不动。新增测试证明拒绝发生在 provider/资金/写回之前；不用一律释放未知预留“清理旧数据”。
5. old OpenAI YAML/env 的完整退出不能影响当前组织凭据。只把已有 retired-config 检查追加 `openai` presence（含null/empty）及仓库自有 `TASK_PROCESSOR_OPENAI_` prefix，删旧 env binding/default/loader/validator/alias replacer；其它 key/env/配置校验次序保持，通用 `OPENAI_API_KEY` 不成为本仓库新全局门禁。五个旧 Secret 样例键仅精确删除：TASK_PROCESSOR_OPENAI_API_KEY、TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_API_KEY、TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_API_STYLE、TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_BASE_URL、TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_MODEL；其余全部保持，不读真实 Secret。`deployments/docker/account-compose/image-trial-init.sh` 还生成一段仅喂旧Manager的 `openai:` 配置（从该根到 `listingkit:` 前），仅删除该完整旧块；其余配置、当前组织凭据写入、角色/初始化/启动顺序与脚本操作不改，不能让现役生成器继续输出会被新loader拒绝的配置。
6. `go.mod/go.sum` 只允许退出失去全部消费者的直接 `github.com/sashabaranov/go-openai v1.41.1` 元数据；先核对保留 Eino/其它模块的真实间接消费者。若仍间接需要则保留对应模块/sum并在PR说明，不因删源码盲删 transitive 记录。无其它版本升级/降级/tidy 漂移，不改 tools 模块或 CI。
7. §2 CURRENT一行例外仅限已批准那一行。所有历史快照/计数/ceilings/检查算法/其它CURRENT、所有workflow及保护规则逐字保持。不为凑文件数修改 KEEP。

## §5 完整退休/抽取/混合矩阵

清单为未来实施的**准确允许范围上限，共 191 路径**（39旧文件、22新文件、98混合文件、27历史文档、1当前文档、设计/两导航3文件、1inventory）。RETIRE/新 owner 是完整闭包，不拆小批；混合/条件依赖文件没有实际必要变化时可零 diff 并记PR，不为达到文件数修改它。未列新消费者/检查/持久化变化须先报告准确提案。当前设计提交只本设计与两导航三文件，未来清单不等于现在授权实施。

### §5.1 RETIRE — 39旧文件

- `internal/app/worker/imageagent/capabilities.go`
- `internal/app/worker/imageagent/capabilities_test.go`
- `internal/app/worker/imageagent/organization_main_admission.go`
- `internal/app/worker/imageagent/organization_main_admission_test.go`
- `internal/core/config/type_openai.go`
- `internal/core/config/validator_openai.go`
- `internal/imageagent/assetpublication/publisher.go`
- `internal/imageagent/assetpublication/publisher_test.go`
- `internal/imageagent/effectpolicy/review.go`
- `internal/imageagent/effectpolicy/review_test.go`
- `internal/imageagent/review_outcome.go`
- `internal/imageagent/store/slot_effect_v3_review.go`
- `internal/imageagent/store/slot_effect_v3_review_memory.go`
- `internal/imageagent/temporal/activities_review_v3.go`
- `internal/imageagent/tools/image_set_executor.go`
- `internal/imageagent/tools/image_set_executor_test.go`
- `internal/imageagent/tools/product_image_executor.go`
- `internal/imageagent/tools/product_image_executor_contract_test.go`
- `internal/integration/grsai/product_image_adapter.go`
- `internal/integration/grsai/product_image_adapter_test.go`
- `internal/integration/openai/client.go`
- `internal/integration/openai/client_manager.go`
- `internal/integration/openai/contextual_client.go`
- `internal/integration/openai/effective_configuration_test.go`
- `internal/integration/openai/identity.go`
- `internal/integration/openai/images.go`
- `internal/integration/openai/images_test.go`
- `internal/integration/openai/logger.go`
- `internal/integration/openai/logger_test.go`
- `internal/integration/openai/pool.go`
- `internal/integration/openai/pool_request_test.go`
- `internal/integration/openai/pool_retry_test.go`
- `internal/integration/openai/product_image_adapter.go`
- `internal/integration/openai/product_image_adapter_test.go`
- `internal/integration/openai/tenant_manager_test.go`
- `internal/integration/openai/types.go`
- `internal/integration/openai/types_contract_test.go`
- `internal/product/image/scene_options.go`
- `internal/product/image/scene_options_test.go`

### §5.2 EXTRACT — 22新文件；无新包/框架或第二事实源

- `internal/app/worker/imageagent/source_main_executor.go`
- `internal/app/worker/imageagent/source_main_executor_test.go`
- `internal/imageagent/acquisition_main.go`
- `internal/imageagent/acquisition_main_test.go`
- `internal/imageagent/assetpublication/acquisition_main_publisher.go`
- `internal/imageagent/assetpublication/acquisition_main_publisher_test.go`
- `internal/imageagent/assetpublication/publication_helpers.go`
- `internal/imageagent/tools/result_builder.go`
- `internal/imageagent/tools/result_builder_test.go`
- `internal/imageagent/tools/source_generation_executor.go`
- `internal/imageagent/tools/source_generation_executor_test.go`
- `internal/imageagent/tools/source_helpers.go`
- `internal/integration/grsai/source_image_adapter.go`
- `internal/integration/grsai/source_image_adapter_test.go`
- `internal/integration/openai/credential_config.go`
- `internal/integration/openai/credential_config_test.go`
- `internal/integration/openai/source_image_adapter.go`
- `internal/integration/openai/source_image_adapter_test.go`
- `internal/product/image/policy_scene_defaults.go`
- `internal/product/image/policy_scene_defaults_test.go`
- `internal/product/image/source_white_background.go`
- `internal/product/image/source_white_background_test.go`

### §5.3 精确混合 — 98文件

按 §4/§6.1 仅改旧声明/消费者/固定fixture，原其它行为零改。内部两个 README 是活跃说明，改为当前 source adapter/凭据 owner；Go 模块两文件条件如§4.2.6。

- `deployments/docker/account-compose/image-trial-init.sh`
- `deployments/kubernetes/listingkit-workbench/base/secret.example.yaml`
- `go.mod`
- `go.sum`
- `internal/ai/contracts.go`
- `internal/ai/contracts_test.go`
- `internal/aicapability/store/gorm_invocation_recorder_test.go`
- `internal/app/httpapi/chain1_local_product_acceptance_test.go`
- `internal/app/httpapi/imageagent_acquisition_module_test.go`
- `internal/app/runtime/currentapplication/title_credential_provision_test.go`
- `internal/app/runtime/image_agent_temporal_runtime.go`
- `internal/app/runtime/image_agent_temporal_runtime_test.go`
- `internal/app/schema/productagentruntime/collation_postgres_test.go`
- `internal/app/worker/imageagent/dependencies.go`
- `internal/app/worker/imageagent/dependencies_test.go`
- `internal/app/worker/imageagent/generation_provider.go`
- `internal/app/worker/imageagent/generation_provider_test.go`
- `internal/app/worker/imageagent/grsai_synchronous_edit_test.go`
- `internal/app/worker/imageagent/image_policy_test.go`
- `internal/app/worker/imageagent/image_set_runtime.go`
- `internal/app/worker/imageagent/organization_main_admission_postgres_test.go`
- `internal/app/worker/imageagent/organization_main_temporal_postgres_test.go`
- `internal/app/worker/imageagent/organization_scope.go`
- `internal/app/worker/imageagent/organization_scope_test.go`
- `internal/core/config/ai_capability_test.go`
- `internal/core/config/config.go`
- `internal/core/config/config_env_test.go`
- `internal/core/config/config_test.go`
- `internal/core/config/defaults.go`
- `internal/core/config/image_generation_test.go`
- `internal/core/config/loader.go`
- `internal/core/config/loader_builder.go`
- `internal/core/config/retired_product_runtime.go`
- `internal/core/config/task16c_imageagent_config_test.go`
- `internal/core/config/task16c_retired_config_test.go`
- `internal/core/config/validator.go`
- `internal/core/config/validator_dependencies_test.go`
- `internal/core/config/validator_format_test.go`
- `internal/core/config/validator_listingkit_test.go`
- `internal/core/config/validator_workbench_test.go`
- `internal/imageagent/assetpublication/image_set_adapter_test.go`
- `internal/imageagent/effectpolicy/decision.go`
- `internal/imageagent/effectpolicy/recovery.go`
- `internal/imageagent/effectpolicy/recovery_test.go`
- `internal/imageagent/model_test.go`
- `internal/imageagent/ports.go`
- `internal/imageagent/projection_test.go`
- `internal/imageagent/recoverable_effect.go`
- `internal/imageagent/service.go`
- `internal/imageagent/service_commands_test.go`
- `internal/imageagent/slot_effect_v3.go`
- `internal/imageagent/slot_execution.go`
- `internal/imageagent/store/slot_effect_v3_codec.go`
- `internal/imageagent/store/slot_effect_v3_gorm.go`
- `internal/imageagent/store/slot_effect_v3_memory.go`
- `internal/imageagent/store/slot_effect_v3_repository_test.go`
- `internal/imageagent/temporal/activities.go`
- `internal/imageagent/temporal/activities_artifacts_v3.go`
- `internal/imageagent/temporal/activities_execution_v3.go`
- `internal/imageagent/temporal/activities_recovery_v3.go`
- `internal/imageagent/temporal/activity_registration.go`
- `internal/imageagent/temporal/approval_ack_test.go`
- `internal/imageagent/temporal/image_set_persistence_test.go`
- `internal/imageagent/temporal/manual_acceptance_test.go`
- `internal/imageagent/temporal/slot_effect_v3_activity_test.go`
- `internal/imageagent/temporal/slot_workflow.go`
- `internal/imageagent/temporal/types.go`
- `internal/imageagent/temporal/worker.go`
- `internal/imageagent/temporal/workflow.go`
- `internal/imageagent/temporal/workflow_test.go`
- `internal/integration/README.md`
- `internal/integration/grsai/synchronous_edit.go`
- `internal/integration/grsai/synchronous_edit_test.go`
- `internal/integration/openai/README.md`
- `internal/integration/openai/credential_store.go`
- `internal/integration/openai/credential_store_test.go`
- `internal/integration/openai/source_edit.go`
- `internal/integration/openai/source_edit_test.go`
- `internal/integration/policy/productimage/catalog.go`
- `internal/integration/policy/productimage/catalog_test.go`
- `internal/marketplace/imagepolicy/boundary_guard_test.go`
- `internal/marketplace/imagepolicy/model.go`
- `internal/marketplace/imagepolicy/resolver.go`
- `internal/marketplace/imagepolicy/resolver_test.go`
- `internal/product/image/model.go`
- `internal/product/image/ports.go`
- `internal/product/image/ports_test.go`
- `tests/imageagent_organization_integration_test.go`
- `tests/import_boundaries_test.go`
- `tests/integration_provider_logger_contract_test.go`
- `tests/issue30_sourcing_contract_test.go`
- `tests/issue388_chain1_acceptance_contract_test.go`
- `tests/legacy_imageagent_manual_runtime_retirement_test.go`
- `tests/legacy_subscription_metering_retirement_test.go`
- `tests/listingkit_backend_retirement_test.go`
- `tests/provider_logger_ssa_loading_test.go`
- `tests/provider_logger_ssa_review_test.go`
- `tests/provider_logger_ssa_test.go`

### §5.4 文档尾部与 inventory

27份历史文档只追加窄失效/抽取顶注，基线原Git blob完整保留为后缀，原数字/勾选/失败证据/其它KEEP不改；不是批量改旧计划。

- `docs/architecture/legacy-browser-management-tail-closure-retirement-boundary.md`
- `docs/architecture/legacy-config-projection-dub-closure-retirement-boundary.md`
- `docs/architecture/legacy-crawler-facade-diagnostics-closure-retirement-boundary.md`
- `docs/architecture/legacy-global-config-logging-tail-closure-retirement-boundary.md`
- `docs/architecture/legacy-host-runtime-closure-retirement-boundary.md`
- `docs/architecture/legacy-idle-adapter-helper-closure-retirement-boundary.md`
- `docs/architecture/legacy-idle-provider-bootstrap-closure-retirement-boundary.md`
- `docs/architecture/legacy-idle-support-contract-closure-retirement-boundary.md`
- `docs/architecture/legacy-imageagent-history-protocol-closure-retirement-boundary.md`
- `docs/architecture/legacy-imageagent-manual-runtime-closure-retirement-boundary.md`
- `docs/architecture/legacy-imageagent-v2-contract-tail-closure-retirement-boundary.md`
- `docs/architecture/legacy-kubernetes-release-execution-closure-retirement-boundary.md`
- `docs/architecture/legacy-openai-adapter-closure-retirement-boundary.md`
- `docs/architecture/legacy-shein-login-identity-sync-closure-retirement-boundary.md`
- `docs/architecture/legacy-subscription-metering-closure-retirement-boundary.md`
- `docs/architecture/legacy-task-openai-convenience-closure-retirement-boundary.md`
- `docs/architecture/product-image-agent-v1.md`
- `docs/engineering/2026-09-25-issue487-main-image-execution-design.md`
- `docs/engineering/evidence/issue334/admission.md`
- `docs/engineering/imageagent-organization-scope.md`
- `docs/engineering/imageagent-review-invocation.md`
- `docs/refactoring/module-target-mapping.md`
- `docs/superpowers/plans/2026-08-06-ai-capability-phase-1.md`
- `docs/superpowers/plans/2026-08-26-image-agent-core-manual-workflow.md`
- `docs/superpowers/plans/2026-08-30-internal-target-architecture-phase2.md`
- `docs/superpowers/plans/2026-09-01-internal-target-architecture-phase3-product.md`
- `docs/superpowers/plans/2026-09-03-pr-282-review-fixes.md`

当前 `docs/development/secret-management.md` 只将旧全局OpenAI样例/说明改为当前组织凭据/已发布路由，保持其它凭据/注册/部署/私密文件安全说明；`docs/architecture/README.md`/`docs/refactoring/legacy-register.md` 只加本批状态和索引，原Git blob完整后缀。当前 `docs/refactoring/phase2-runtime-inventory.md` 未来仅一条CURRENT精确删除，不追加历史顶注或改任何其它字节。本设计本身保持冻结，滚动源码/CI/复核证据写同一PR。

### §5.5 KEEP / 零差异

`store.records.go` 与 productagentruntime schema_contract 及全部 schema/grants/migrations/Goose/SQL/物理字段零改；当前资金/资源、generation/intents/recovery核心、SourceIdentity/来源保存、图组/Artwork、SDK marker位置、公开协议名/队列/ID、Store观察与官方授权、采集/浏览器/安装/构建和 canary/WithWorker/runner 保持。保留 ImageSet/Artwork 的同名 scene/detail/size 等角色值、人工选择/批准、其它域的 ReviewRequired 与当前 readonly inspection，不按“review”关键词全删。GRSAI/百炼实际 HTTP/发送上限/SSRF/取消/输出维度原测试不改，除 §5.3 内 fixture/构造器窄改接。Shared httpimage 及 Logrus 当前实现不变；不添加旧 SDK import 或 dummy consumer。

## §6 验证，不建设验证平台

设计阶段只做消费者/完整闭包/当前矩阵/相对链接/导航/原Git blob后缀与原文档guard；未运行旧命令、provider、DB、测试的副作用分支或业务构建。未来同一批先给旧入口/旧DTO/ReviewActivity和旧默认配置写 scoped guard，确认真实RED；再最小抽取/改接/删除到GREEN。当前单主图、recipe、Artwork、真实Asset批准的原有效断言继续验证；不以另一AI/CI代替产品验收。

### §6.1 原检查/测试的精确转换授权面

下表只准改准确目标及旧符号的机械fixture改接，**原分析算法、失败关闭与其它安全/业务断言保持**。§5.3 所列测试内的已列旧 constructor/DTO/字段替换和移除旧必填配置可在所有直接引用函数/共享fixture进行，不要求为同根编译接线逐行再授权；每处在PR记录，不能据此改无关断言/负例。清单外文件或新安全/事实变化仍须先报告。

| ID | 原检查/fixture | 允许变化 / 必须保留 |
| --- | --- | --- |
| T1 | product_image_executor_contract / image_set_executor / capabilities 原测试 | 取消旧Extract/Scene/Review专属实现断言；将当前一来源一次编辑、坏源digest/缺策略/quote漂移/非直通/完整来源、recipe/method/结果ID断言移到 source_generation/result_builder 原owner测试。空输出/typed nil/URL/字节/维度/取消/quote gate原负例保留在仍执行的当前端口；不留旧fake实现保活 |
| T2 | source adapter / GRSAI / source_edit /同步编辑 | 精确构造器/接口改 SourceImageAdapter/SourceImageProvider、SourceWhiteBackgroundRequest，删旧Subject/Scene/Review专属case/fixture。当前一次发送、route/credential/configuration version pin、源字节/1024实际维度/输出聚合/HTTP失败UNKNOWN/取消断言保持；合成固定payload与真实provider调用不混同 |
| T3 | current config 原测试（§5.3 core/config列） | 旧OpenAI默认/命名继承/compact alias/validator正例改为同输入presence拒绝，删旧专属validator case；非本轮case只删除已退出OpenAI fixture字段。数据库、身份、Workbench/ListingKit拒绝、retired原namespace、当前图像配置、校验分组顺序/错误类型/Hint其它要求不动 |
| T4 | dependencies/organization_scope/原PG和Temporal集成fixture | 去 BuildAI/完整Capabilities前置；原Db/角色/授权/空/typed nil/关闭/deadline检查及current单主图真实端到端保留。TestOrganizationMainRetiredReviewAdmissionCannotReserveAcrossGeneration 的旧 Review stub随旧port退出，等价无预留/无发送拒绝断言改接当前非法旧输入；当前generation资金/race/UNKNOWN/重复成功全部保留。不把模型Review重新当唯一当前provider |
| T5 | Publisher/当前 acquisition/httpapi/Chain1与source tests | 主图批准断言移 AcquisitionMainPublisher/真实source fixture；原跨scope、来源版本、摘要、排序、materialization、同ActionID异载荷、丢回执、no-write及真实Asset原子断言保持。通用多槽位gallery正例退休或变当前窄拒绝；原 HTTP/JSON/status、安全/无副作用和完整业务结果不改 |
| T6 | Review effectpolicy/store/activity/workflow 原case | 删除只能消费已取消Review的reserve/settle/release/outcome/Activity保活case；针对旧review输入/phase/非空ReviewUsage改成明确拒绝且零provider/资金/写回，并保留nil/null当前snapshot字节。其它真实generation/publication/cancel/UNKNOWN/恢复/ledger并发断言逐字保持；原安全底线不能因旧fixture删除而消失 |
| T7 | tests/integration_provider_logger_contract_test.go | 仅从 method-set/current constructor表移走退出的openai logger/client/manager/pool；退休 TestOpenAIConfigsExposeExplicitNonSerializedLogger、TestOpenAILoggerConfigFieldsAreASTComplete 两个旧保活函数，改由T12强制旧声明/ClientConfig.Logger缺席及新最小配置。GRSAI及其它当前logger断言、遍历/解码/归属/错误处理不变 |
| T8 | tests/provider_logger_ssa_{test,review_test,loading_test}.go | productionProviderLoggerRules 仅去旧OpenAI constructor行；直接依赖已删除真实配置的fixture改当前GRSAI logger constructor/匹配adapter。原alias/phi/typednil/全配置store/静态helper/跨module所有正负断言及数量语义保持，通用SSA/nil lattice/package加载算法零改，不加skip或missing容忍；独立fake provider的名字不是业务消费者可保持 |
| T9 | listingkit_backend_retirement 原AST / legacy manual / subscription guard | 仅将本批旧KEEP（Manager/Client/Pool/Logger/OpenAIConfig/ProductImageAdapter/旧capabilities/Review ports）移到同owner禁入；原 TestLegacyRuntimeSkeletonAndIdleConfigClosureIsRetired、TestLegacyAIRoutingImagePolicyClosureIsRetired 精确表项转换。其它同文件指向新adapter/最小配置的准确保活路径/符号改接；保留原AST/decoded import/类型字段/decl禁令/错误处理及全部其它KEEP，不宽泛删除 openai 包检查 |
| T10 | image policy/catalog/scene defaults 原测试 | old SceneOptions/merge nil-base行为改当前 PolicySceneDefaults/Normalize；保留原10字段/规范化/去重/字节上限/clone/ownership/精确policy key和整个catalog hash版本。只有旧执行期override合并case随旧renderer退出；不删policy数据或改变profile JSON |
| T11 | worker Secret与模块元数据、CURRENT | 原Secret精确expected表仅删§4.2五键；旧脚本仅删§4.2明确的openai块，保留当前DB/凭据/装配及原其余断言。其余键/凭据隔离/canary安全断言保持。CURRENT只删用户批准的一行，历史/计数/ceiling/准确性算法不变。SDK模块只按真实保留消费者选择条件删除，无其它依赖漂移 |
| T13 | credential_store / 最小 credential_config / 原 tenant-manager、title_credential_provision、productagentruntime/collation_postgres、generation_provider 有效断言 | Get/Save 的明确tenant/user/client精确查找、enabled/缺失/DB错误和组织专属解析不访问成员/全局/静态fallback等有效负例保留，改接当前共享aiidentity与去fallback参数的最小port。旧Manager/成员优先继承/全局key正例随旧入口退出；CredentialReference/Version、当前published route与生成前后version漂移拒绝不改，不把user_id物理字段删掉或迁移行。新增列明的三测试文件仅将旧 WithIdentity/Identity 改接共享 aiidentity、去 ResolveClientConfig 第三个 fallback 参数及同签名mock；原禁成员fallback、collation精确作用域/漂移证据与拒绝不修复、provider/version/取消及所有其它断言原样保留，不执行真实DB或借用既有凭据 |
| T12 | 新 scoped retirement guard（现有tests目录文件内） | 强制§5.1旧文件/声明和专属producer/Review Activity注册缺席，当前窄owner/源码入口/标准wire/canary/关键安全约束必需；现役根上用原decoded import/AST设施禁旧SDK/旧跨层依赖，无missing-skip。字段/合同rename只改真实owner，不复活旧包或用wrapper隐藏旧依赖 |

所有表列fixture的原PG独占gate、DSN/系统DB拒绝、自有临时库清理与有效断言保持；没有任务自有环境授权与实际执行就是NOT_RUN，不能借用既有DSN/凭据。未来必要自检限受影响原包/guard、原合成fixture及命令仅编译，稳定候选才跑原完整CI，不人工重复完整矩阵，不造runner/故障注入平台/验证工具的验证工具。

## §7 准入、独立复核与停止点

1. 当前三文档交付：Design Basis N/A（仅文档/只读），设计工作Ready；未来源码 Design Basis Independent Architecture，仍NOT_READY。#301当前段记录唯一Writer/分支/基线及设计PR，历史后缀完整保留、正文≤65536、OPEN。旧单主图/Goose等数据不操作。
2. 适用独立 Architecture Review 仅本批实际边界：当前两类来源编辑/结果/批准最小抽取、旧审核历史停止、JSON/hash/零值槽位、组织凭据/资金/恢复不变量、最后消费者/CURRENT、§5/§6.1及保留能力。按五类先分类，最多两轮正常设计评审，不重开无关全局架构。
3. 设计和原CI完成后交用户确认§1–§7（重点§4.2/§6.1）。只有用户明确确认全部退休/抽取/旧审核历史范围，且授权/IMPLEMENTATION_READY在Issue/PR写回读回，才修改正式生产路径。本轮同意先设计、单行inventory例外、CI或另一AI PASS均不自动授权实施。
4. 一个Writer/一个主要PR整批交付，不为文件/证据机械拆批；范围内实现/必要自检/提交推送/原CI修复连续处理。新增真实BLOCKER才退回设计；表外新消费者/持久化/检查变化先准确提案，不降低门禁。
5. 未经用户本轮授权不默认新增AI互验线程；设计与源码最终复核分别绑定准确HEAD、分别授权。原适用复核运行时不重复触发，失败最多一次原重试；独立子代理替代检查先获本轮明确授权，不能把failed/空finding当PASS。
6. 源码完成进入In Review，完整实际diff/消费者与有效证据集中最终检查；准确HEAD、CI/attempt、review、fixture/NOT_RUN/UNKNOWN留同PR。未变代码证据可明确复用，只核对新组合；合并另获准确HEAD授权且不绕过保护，之后核实真实merge SHA的完整push/main CI。
7. #301保持OPEN。真实部署/初始化/grant/迁移/记录/凭据/缓存/Profile/进程/运行验收/关单分别授权；不增加内部试用前置。实现者/CI/AI复核不签发产品验收或上线结论。
