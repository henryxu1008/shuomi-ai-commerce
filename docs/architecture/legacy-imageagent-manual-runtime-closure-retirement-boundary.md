# 旧 ImageAgent v2/v3 手工图像任务运行链完整退休设计

这批删除**旧 v2/v3 业务启动模式、未绑定组织身份的任务执行链、v2 一次性生成/发布和专属持久化尾部**。当前图组、Artwork 和采集页单主图继续走组织协议；隔离 canary 保留。版本名带 v2/v3 的代码和表不能据此整包删除。

状态：**Independent Architecture / DESIGN_ONLY / SOURCE_NOT_STARTED / AWAITING_REVIEW_AND_USER_CONFIRMATION**。父目标 [#301](https://github.com/qq550723504/shuomi-ai-commerce/issues/301) 保持 OPEN。调查基线 main `ddbcd714568efc507debf5dcca7d89a9b77d6b98`。本次只提交本文、架构索引和 Legacy Register 三份文档；下述源码变更均为待确认设计。

## 1. 用户结果、范围与 Design Basis

使用者是维护当前图像能力的开发者，以及使用采集、图组和 Artwork 的内部试用用户。结果是只剩当前组织身份下的业务执行入口，不再能从旧命令/队列启动另一套手工图像任务；现有用户操作和批准、保存、恢复结果不变。本工作是源码退休，不是当前试用的新前置。

| 完整闭包 | 同一交付批次的处置 |
| --- | --- |
| 入口与装配 | 删除 CLI 的 v2/v3 业务模式、默认 unscoped client/repository、旧分支装配及依赖。保留明确的 organization-v1 业务模式和 canary，不新建替代宿主 |
| 执行与批准 | 删除旧 ImageSlotWorkflow/v2 Activity、六项更早 Legacy Activity、v2 一次性执行/URL 发布、旧业务工作流注册与无组织身份执行分支；保留当前 V3 staging/publication/recovery 和组织业务 |
| 持久化与合同 | 抽取 V3 仍使用的最小身份、用量、dispatch、生成结果及 hash 合同和存储纯 helper；删除 v2 effect 仓库、记录、内存 map 及新安装建表项；不改当前表名和载荷 |
| 声明、检查与文档尾部 | 删除两个旧 Kubernetes worker Deployment 及 kustomization 引用；转换其原保活检查，保留 canary、现有构建目标和安全门禁；收齐直接执行说明与专属 fixture |

**Design Basis：Independent Architecture。** 改动跨 CLI、Temporal、Repository、对象/Asset 边界，并替代前批 KEEP，必须先完成适用独立设计复核和用户确认。当前文档整理为 N/A / Ready；未来源码仍 NOT_READY，不能以本文的实现细节作为已准入事实。

Must：当前图组/Artwork、采集单主图、组织隔离、真实授权、JSON/hash/版本、deadline/取消、UNKNOWN、点数与 Asset 显式批准保持；完整退出旧业务入口；workflow 和完整 inventory 零改。Out of Scope：部署、真实验收、迁移/回填/排空/删除历史、访问已有凭据或进程、订阅残留字段、无关图像/provider 优化，以及恢复 #985 已退出的 SHEIN 登录/身份同步。Store 观察同步及其官方授权保持。

## 2. Authority、前批 KEEP 替代与威胁边界

适用依据：[Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、[当前 UI/IA Authority](../product/final-ui-ia-authority.md)。本批不改变页面、导航、文案、当前 HTTP 路由或交互，不创建 Figma 增量；当前 Figma authority 延续。

| 前批依据 | 本批替代的 KEEP | 继续有效 |
| --- | --- | --- |
| [#977 旧 HTTP/运行门面](legacy-imageagent-http-runtime-closure-retirement-boundary.md) §3–§6.1 | “全部 v2/v3 worker/registration 与 frozen execution 继续保留”的实现限定；旧模式现在按本设计删除 | Service 通用命令、当前 HTTP/组织 worker、CHAIN-1、实际安全断言、WithWorker canary、V3 有效协议 |
| [#988 订阅/计量闭包](legacy-subscription-metering-closure-retirement-boundary.md) §4.1/§6.1 | 除旧 Review 外其余旧 v2/v3 注册/装配零改的限定；两旧 Deployment 的强制存在 | 当前 AI、生成/恢复、钱包/Resource、组织图像、safeReviewTransportLogger 的当前 credential consumer、Store |
| #487 与当前源码的组织采集单主图 | 不替代其当前业务、权限或存储 owner；其 CLI 入口仍为真实消费者 | organization-v1、独立受限池、当前成员/角色检查、点数准入、当前资源恢复、显式批准 |
| 旧八计划、旧 replay/部署说明 | 保活旧业务 wire、v2 URL ledger 和历史 v2 replay 的执行要求 | 历史证据逐字保留；不再构成当前运行/迁移命令或试用前置 |

当前 Threat Model 仍覆盖跨组织/成员混淆、缺失/撤销授权、异载荷幂等冲突、重复 provider/批准、丢响应/重启、资金与对象事实归属错误、敏感内容泄露。没有新增兼容义务、历史运行接管、未知 provider 结果自动重发或通用迁移平台。源码证据不是 retained-runtime 验收；没有调查真实实例状态。

## 3. 实际消费者与架构问题

以下为基线源码的实际调用，不用函数名或测试名称推定“旧”。

| 消费者/入口 | 调用路径 | 结论 |
| --- | --- | --- |
| 当前完整图组/Artwork | cmd/current-application → WithFullImageSet → buildFullImageApplication → NewManagedImageSetTemporalDependencies → NewOrganizationImageSetWorker → organization-v1 注册 | 保留。当前 imageSetSlotExecutor / generation / Asset owner 不迁移 |
| 当前采集单主图 | cmd/current-application 在未配置 WorkerConfigFile 时选择 WithAcquisitionImageAgent → buildAcquisitionImageModule → Service(...WithOrganizationScope...) → NewOrganizationClient | 保留。四条 main-image 路由、UI API/合同/proxy 与真实授权不能随旧 HTTP 宿主删除 |
| 组织独立 worker | deployments/docker/account-compose/docker-compose.image-agent.yml 的 image-trial-worker 明确传 organization-v1 / image-agent-organization-v1 → 现有 CLI ForMode resolver | 保留此明确模式及 Compose/私有配置。不能把该命令整体改成 canary-only，也不操作该环境 |
| 旧业务 worker | 两个 Kubernetes Deployment 分别传 v2 / image-agent-manual、v3 / image-agent-manual-v3 → 同一 CLI/resolver → unscoped Repository/Activities | 整链退休。两个声明不是当前组织 worker 的替代；不为保持旧 base 的 worker 数量创建新 Deployment |
| canary | release-authority runner → 同一 CLI -canary → RunImageAgentCompatibilityCanaryWithWorker → NewCompatibilityCanaryWorker | 保留最小 worker，仅注册 canary，无业务 Activity、DB、provider 或对象写入依赖 |
| 当前批准与恢复 | organization worker → V3 effect CAS/fence/lease → immutable staging/final → ImageSet/Artwork 或组织单主图 PublisherV3 → 原 Asset approval transaction | 保留。PublishApprovedV3、image_agent_v3_slot_external_effects 不是闲置旧链 |
| CHAIN-1/当前验收 | typed Service(...WithOrganizationScope...)、当前 HTTP replay、acceptancetest driver → OrganizationClient/组织 worker | 保留完整用户链及丢响应/并发/重启证据，不改成 seed 最终结果 |

**根因不是某个 v2 字符串。** 同一个 resolver/Activities 构造器混合旧执行、组织生成和 canary；NewActivities 甚至在当前 V3 路径强制要求旧 SlotExternalEffectRepository / one-shot SlotExecutor / Publisher。只删入口会留下失去最后消费者的 ledger/DTO，直接删整包则破坏当前业务。实施应先解除当前构造对旧端口的强制依赖，再按闭包删除；不得用空实现、兼容 alias 或 wrapper 补回旧端口。

同根尾部还包括 ProductImageSlotExecutor.resolveInput 的 legacyPolicy：在 TargetPlatform 为空且 ImagePolicyContext 为 nil 时绕过当前 ProfileResolver，并选择旧操作/审核分支。生产调用中，当前采集单主图明确写入 product/zz/default/general 策略；图组走 resolveImageSetInput，Artwork 走专属 executor。无策略正例仅为 TestExecutorAcceptsHistoricalV3InputWithoutPolicyFields 等历史 fixture，不能据此把这条兼容旁路当有效 V3 协议保留。该 flag/分支及只为其保活的默认操作同批退休；当前明确策略、图组/Artwork 的字段/结果和有效 V3 wire 不变。

保留现有 Temporal SDK、GORM、对象存储、现行 Resource/Generation/Asset owner；不建设第二个任务系统、runner、调度器或持久化事实源。

## 4. Owner、协议与持久化边界

| 事实/副作用 | 唯一当前 owner | 本批变化 |
| --- | --- | --- |
| 组织 Run/Plan/slot/attempt/event/catalog/projection/commit | internal/imageagent + store.NewOrganizationRepository，scope_protocol=image-agent.organization.v1 | 组织协议、复合 scope、Run/Member/BusinessTask 绑定、projection CAS/commit 保持；退出无组织协议创建/读取的生产入口，不转换历史行 |
| 生成报价/dispatch/点数/UNKNOWN | GenerationExecution/Recovery + 当前 Resource/账本 owner | 原授权/幂等/计费顺序保持；无 reserve/review fallback、无自动重发或 UNKNOWN 接管 |
| V3 effect、publication claim/lease/fence | SlotExternalEffectV3Repository + effectpolicy | 原 identity、phase、输入指纹、预算/receipt、租约/fence 和恢复归属不变 |
| 生成对象 | 原 DurableArtifactStore / immutable objectstore | transient bytes/URL 不进入 durable payload；staging recovery、final manifest、安全公开 URL、原45天边界不变 |
| 已批准资产 | 原 Product/Asset approval/inventory transaction | 显式批准、版本/digest/action/幂等与同载荷 attach 保持；不产生第二份批准事实 |
| 旧 v2 effect | SlotExternalEffectRepository / image_agent_v2_slot_external_effects | 专属源码与新安装创建项退出；既有表/行/权限不操作、不 Drop、不清理 |

### 4.1. 纯合同与 helper 抽取

1. 从 domain slot_effect.go 抽取当前 V3 仍用的 ProviderDispatchState/Error/StateOf、ErrBudgetQuoteUnavailable/ErrProviderContractViolation、SlotUsageOperation/Quote/Receipt/CostBasis、review 用量合同/ValidateSlotUsageQuote、SlotExternalEffectIdentity、GeneratedAsset/SlotGeneratedOutput、SlotExecutionFingerprint 到同一当前 domain 的 slot_execution.go。名字/字段/JSON tag、Marshal→SHA-256 算法和返回值保持；不是旧包 alias，不改外部 import owner。
2. 从 store/slot_effect.go 原样抽取 validateSlotEffectIdentity、validateMemorySlotEffectScope、slotEffectKey 到当前 store 的 slot_effect_identity.go；V3 GORM/memory/review 继续直接使用。其余仅 v2 使用的 reserve/transition/decode/clone helper 随 v2 ledger 删除，不复制整个旧文件。
3. SlotWorkflowResult 仍是当前父工作流的结果聚合合同，保持；旧 SlotWorkflowInput 若仍承载当前 childWorkflowID 的纯值输入，仅抽取该 ID 构造所需值到私有当前 helper，保持完整 ID 字节，不为旧 child 留 payload/入口。所有有效 V3 Activity 输入、ResultDigest、scope/version、error 分类保持。

### 4.2. JSON、版本与新安装

- RunModeManual 的持久化值仍被当前图组/Artwork/组织单主图使用，保留；不能把所有包含 manual 的方法/字段当旧任务。WorkflowClient.StartManual 是当前端口，保留当前组织语义。
- 保留当前组织 workflow 名 ImageAgentOrganizationWorkflowV1、OrganizationTaskQueue、组织 scopedWorkflowID/EffectRecovery ID、当前 V3 Activity 名和 payload，以及 V3 timeout/deadline、finalization、budget、projection commit、approval-key、recovery-start 的有效 GetVersion 协议。已有已保存载荷、hash、版本不改、不重写。
- 旧 ImageAgentWorkflow（未绑定组织的注册名）、ImageSlotWorkflow、旧 wire/Legacy Activity 不再注册。共享父函数 ImageAgentWorkflow 仍为当前组织工作流实现，不因同名而删除。仅退休指向 v2/更旧业务的选择、dispatch、persist/publish 分支；出现已退休的 wire 选择须在执行旧 child/Activity 前关闭，不回退、不挂起等待不存在的 handler。当前 V3 有效 replay/marker 组合继续验证，不能笼统删除 history_replay_test.go。
- 三份明确 v2 history fixture（testdata/v2-slot-inflight.json、v2-awaiting-approval.json、v2-pre-atomic-awaiting-approval.json）及只要求旧 wire 继续执行的测试退出；不生成/转换实际 Temporal history，不排空旧队列。没有观测或声明真实旧任务已结束。
- 当前 Image DB 仍使用 image_agent_v2_runs/plans/slots/attempts/events、asset_catalog/manifests、projection_snapshots/commits，以及 image_agent_v3_slot_external_effects。**这些表名和当前 schema/role 检查保留**。只从 AutoMigrate 的新安装列表删除 slotExternalEffectRecord，并校准原精确表清单；不新加迁移文件，不修改已有记录，不 rename/recreate 表。
- Organization API/worker runtime grants、当前 generation 表和 ai_invocations、Asset 表/凭据 owner 不变。worker 权限清单已明确不准访问旧 v2 effect 表，不得借本批增加权限或“清理现有权限”。

失败边界：旧模式/空身份必须在载入私有配置、开 DB、dial/Start 业务 worker 或任何 provider/object/批准之前拒绝；当前命令相同 action+载荷 attach、不同载荷 conflict，取消/deadline 不扩大绝对边界；已发送且未知的结果仍由当前 recovery owner 处理。撤权不抹掉已发生消费，不据异常重发。Artifact/DB/Temporal 失败和响应丢失复用现有 lease/CAS/恢复协议，不新增跨库事务或补偿协议。

## 5. 删除、保留与抽取矩阵

### 5.1. 生产源码与专属声明

| 路径（均相对仓库根） | RETIRE / EXTRACT 的准确范围 | KEEP / 限制 |
| --- | --- | --- |
| cmd/image-agent-temporal-worker/main.go | v2/v3 业务 option/help/分派；无明确组织模式的 business start | 现有命令、config/log-level、明确 organization-v1 及隔离 -canary 分支；旧模式先拒绝再 resolve |
| internal/app/runtime/image_agent_temporal_runtime.go | 旧业务模式选择、传递 one-shot SlotExecutor/Publisher 的字段/装配 | 当前组织启动/lifecycle/context/close、FullImageSetDependencies、当前 DialOrganization...、WithWorker canary。ForMode/WithOptions 现有函数可作为“仅接受组织模式”的真实校验入口，不新增兼容转发别名 |
| internal/app/worker/imageagent/dependencies.go | v2/v3/unscoped 分支、legacyV2AssetMaterializer、旧 v2 publisher/executor/ledger 接线、仅被其使用的 helper/import | 组织 resolver/role-policy reader、当前 capabilities/credential/authorizer、durable artifact timing/store/material reader、generation/recovery/storage capacity。删除无模式私有默认 v3 resolver，而非把它改成默认组织 |
| internal/app/worker/imageagent/image_set_runtime.go、organization_main_admission.go | 仅不再被当前构造需要的 ExecuteSlot one-shot stub/字段 | Quote/GenerateQuoted、当前关闭的无准入生成行为、BuildSlotResult、图组/Artwork admission，现行 NO_SEND/NO_RESERVE/UNKNOWN 行为 |
| internal/imageagent/service.go | 未显式组织 scope 时的旧身份/owner 分支；NewService 必须具备组织 scope，缺失则关闭 | 当前 WithOrganizationScope 作为显式准入、Service 命令、current identity/member/permission/gate；当前类型、JSON、语义不重命名 |
| internal/imageagent/temporal/worker.go、organization_scope.go | NewClient 未绑定组织的构造、旧 queue/workflow 注册及 fallback、默认无组织 Activity identity 恢复；非组织启动路径 | NewOrganizationClient、当前业务/命令/恢复 ID、实时授权、组织范围验证。业务 WorkerConfig/ForMode 只接受 organization-v1，V3 协议不是一个新的业务启动模式 |
| internal/imageagent/temporal/types.go | 六 Legacy payload、旧 ExecuteSlot/PersistSlotResult/PublishApproved payload、旧业务 wire 常量/可执行 mode/专属 child 输入 | 当前 V3/组织 payload、SlotWorkflowResult、durable ports、当前状态/结果/错误。旧 queue 名可仅作拒绝/隔离哨兵；canary ack 的 v3 值继续有效，不能凭该常量接受业务 v3 模式 |
| internal/imageagent/temporal/activity_registration.go | 六 Legacy* handler/registration、PublishApproved(v2) 及 v2注册 | 当前 ExecuteSlotV3/Review/Recovery/Persist/组织 ImageSet/Artwork/PublisherV3 的注册，活动名与有效入参不变 |
| internal/imageagent/temporal/activities.go、activities_execution_v3.go、activities_persistence.go | SlotEffects(v2)/SlotExecutor/Publisher 端口与强制构造依赖，ExecuteSlot(v2)、PersistSlotResult(v2)、restoreActivityIdentity 旧分支 | 当前构造必须要求组织 authorizer、V3 Repository/staged/artifact 和适用的真实 publisher；V3 persist、catalog/scope/预算/真实权限、publication recovery 原行为 |
| internal/imageagent/temporal/workflow.go、slot_workflow.go | 旧 v2 child 函数/选择、旧 Activity dispatch/persist/publish；旧历史选择改明确关闭 | 当前父工作流/V3 child 与当前有效 GetVersion marker/命令/计费/取消/UNKNOWN/恢复；不是全文件删除或重写。非组织输入在业务效果前关闭 |
| internal/imageagent/ports.go、slot_effect.go → slot_execution.go | v2 SlotExecutor/RecoverableSlotExecutor/ApprovedAssetPublisher/PublishApprovedInput、旧 phase/reservation/attempt/repository；§4.1 抽取 | Staged/Budgeted/Reviewer/V3 publication、当前 WorkflowClient、SlotExecutionInput/Result、生成/usage/identity/hash 原合同 |
| internal/imageagent/tools/product_image_executor.go | NewFrozenV2ProductImageSlotExecutor、legacyV2 flag/materializer/分支、ExecuteSlot/PublishSlot(v2)、resolvedSlotInput.legacyPolicy/缺策略旁路及对应旧 slotOperations 分支、仅服务它们的接口断言/无消费者 helper | NewProductImageSlotExecutor、当前明确策略的 quote/generate/review/BuildSlotResult、NewImageSetResultBuilder；artwork_executor.go 当前行为原样保持，不包装旧执行器 |
| internal/imageagent/assetpublication/publisher.go | NewV2Publisher、PublishApproved/approvalCommitV2 及仅 v2 的接口断言 | 当前组织单主图 NewPublisher/PublishApprovedV3、scope/source/version/digest 校验及原 Asset transaction；ImageSet/Artwork publisher 不改 |
| internal/imageagent/store/slot_effect.go → slot_effect_identity.go | v2 repository 全实现及仅 v2 helper；§4.1 原样抽取三 helper 后整退旧文件 | 当前 V3 的 validate/key/scope 不改变，不能通过删 helper 绕过校验 |
| internal/imageagent/store/gorm.go、memory.go、records.go | NewGormRepository 无组织构造、旧 scope 默认生产路径；slotEffects map；slotExternalEffectRecord 和 AutoMigrate 创建项 | NewOrganizationRepository、组织 scope fail-closed、所有当前表/字段/JSON/CAS/事务，V3 map/records 不变。Memory 仍为当前测试 double，不成为默认生产 owner |
| deployments/kubernetes/listingkit-workbench/base/image-agent-temporal-worker-deployment.yaml、image-agent-temporal-worker-v3-deployment.yaml | 两文件删除 | 不 kubectl apply/delete，不清理已部署对象 |
| 同 base/kustomization.yaml | 仅删除上述两个 resources 项 | 其它资源、ConfigMap/Secret/overlay、canary release-authority/policy 均不改 |

这些是一个完整 Delivery Batch；合同/构造去耦与删除是同一分支的实现顺序，不分成机械小 PR。只有矩阵所指的符号闭包允许变化，不能以“同目录”为由改无关能力。

### 5.2. 直接检查、fixture 与文档尾部

| 工件 | 允许的有界转换 |
| --- | --- |
| cmd/image-agent-temporal-worker/main_test.go；internal/app/runtime/image_agent_temporal_runtime_test.go；internal/app/worker/imageagent/dependencies_test.go | v2/v3 正向启动用例改为拒绝且 resolve/open/dial/provider 计数为0；有效组织构造/lifecycle/context/关闭及 canary 原断言保留 |
| internal/imageagent/temporal/{worker,workflow,slot_effect_activity,slot_effect_v3_activity,history_replay,organization_scope,manual_acceptance,canary,budget_authorization,effect_recovery_workflow,artwork_recovery,generation_output_recovery}_test.go | 仅旧 lane/接口 fixture 和注册表转换；当前有效 assertion/recovery/replay/timeout/取消继续保留。真正只验证已退休 v2 的用例删除，不删整个混合测试文件 |
| internal/imageagent/store/{repository_contract,gorm_concurrency,image_set,generation_fact,slot_effect_v3_repository,slot_effect_v3_budget,schema_v2}_test.go | NewGormRepository fixture 改到当前组织 repository 与同值组织 identity；保留 scope/conflict/CAS/DB 并发/budget/lease/fence/腐坏拒绝全部断言；schema 只减旧 v2 effect 表 |
| internal/imageagent/store/slot_effect_repository_test.go | 旧 v2 ledger 专属测试退休；其中如有与当前 identity 验证相同的有效拒绝断言，转到现有 V3 tests，不保留旧 ledger fixture |
| internal/imageagent/assetpublication/{publisher,image_set_adapter}_test.go；internal/imageagent/tools/{product_image_executor_contract,image_set_executor}_test.go | 只移除 frozen-v2 专属分支，适用 one-shot 正向用例改当前 staged BuildSlotResult；V3/scope/hash/批准/规则断言保留 |
| internal/imageagent 域原测试（service_commands_test.go、organization_scope_test.go、policy_context_test.go、image_set_service_test.go、artwork_service_test.go） | 仅补当前组织 scope/verified member fixture、去旧 mode/default 分支；原业务、安全、JSON/hash/版本断言不得减弱。不改变生产字段为迎合 fixture |
| internal/app/worker/imageagent/{organization_main_temporal_postgres,generation_runtime,grsai_synchronous_edit,minio_compat}_test.go | 仅旧 ActivityDependencies/one-shot/默认 resolver 构造字段及专属 stub 改接；原组织准入、provider/资源/对象行为与失败断言不变 |
| tests/imageagent_organization_integration_test.go；internal/app/httpapi/chain1_local_product_acceptance_test.go、image_set_worker_authorization_test.go | 删除 ActivityDependencies 的旧字段和仅实现旧端口的 stub method；现有 current/CHAIN-1 的真实授权、Source/Review/图像/Asset/Store/Draft 全链、丢响应/重建/异载荷冲突保留 |
| internal/app/runtime/productlistingschemamigrate/organization_image_agent_postgres_test.go | TestOrganizationImageAgentInitCreatesOnlyCurrentOwnerTablesPostgres 精确 expected 仅减 image_agent_v2_slot_external_effects；旧表缺席、当前表/权限/失败关闭全部验证 |
| tests/listingkit_deploy_workflow_test.go、listingkit_release_boundary_test.go、commercial_readiness_workflow_test.go、legacy_subscription_metering_retirement_test.go；两个 shell guard | 按 §6.1 精确转换旧 Deployment 保活；当前 Secret/Config/canary、NOT_READY、原检查算法保持 |
| deployments/kubernetes/listingkit-workbench/README.md | 仅前置本批退休注记：两个旧业务声明退出、当前组织 worker/canary 保留、旧 drain/rollout 不再是当前命令；旧发布 NOT_READY 与历史全文保留 |
| docs/architecture/README.md、docs/refactoring/legacy-register.md、本设计 | 当前索引/登记和设计状态最小更新；前批文档/历史不追写成当前 PASS，不抹掉历史 KEEP |
| 新 tests/legacy_imageagent_manual_runtime_retirement_test.go | 复用原 AST/import/path 检查 helper，检查 §5 的关闭/缺席与当前 owner 保留，不另建验证工具或替换原必需检查 |

上述直接测试名单允许删除退出接口的 fixture 字段/实现、补完整当前组织 fixture；不授予整个目录重写测试的权限。实施前逐项映射真实引用，发现未列出的生产消费者或需削弱原有效断言必须停止该项并报告，不暗自扩范围。

## 6. Canary 决定与验证

**保留 canary 及原 WithWorker 执行方式。** 它仍被现有 release-authority 和 CI guard 消费，验证隔离队列 polling/registration/ack，不是旧业务 worker、图片生成或上线验收。保持同一 CLI 路径、-canary/-canary-task-queue、队列 image-agent-manual-v3-canary、ack Build=image-agent-temporal-v3 / WireMode=v3 和最小无 Activity worker。不能因删除旧业务 v3 mode 而删除 ack 字段或恢复其业务分支。

保持 release-authority 的零副本 runner、无 Secret 隔离和 release-policy 原样；不启动它、不执行 canary、不触发部署，不把 canary 作为试用新前置。旧业务队列名只用于明确拒绝/隔离断言，不能用来 poll、接管、转发或降级。保留现有 canary 隔离限制；变更不扩大其允许进入的业务队列集合。

### 6.1. 原检查转换的冻结边界

| 原检查/范围 | 精确转换 | 必须保留 |
| --- | --- | --- |
| scripts/tests/listingkit-image-agent-canary-order-test.sh | worker_manifest 的必需读取和生产队列正向 grep 改两个旧 Deployment 的显式缺席；保留必需读取 workflow/runner | retired API workflow 完整固定失败源、isolated queue、禁止与旧业务 v3 queue 共用、原错误处理；不加 missing-skip |
| scripts/tests/listingkit-apply-api-deployment-test.sh | 原 API driver 用来测试的 v2 worker manifest fixture 换成同 base 已保留的 listingkit-ui-deployment.yaml，所有按 manifest 推出的名字/容器期望随 fixture 同值校准 | apply/patch 顺序、immutable digest、annotation、失败传播、临时文件清理、显式 container/namespace 等原断言；UI 自己的测试和 driver 不改。不执行真 kubectl |
| TestListingKitImageAgentDeploymentsAndCanaryAreCapabilityIsolated | 两旧 Deployment 必需读取/正向 args/selector/same-image 比较转换为缺席，删除仅旧部署使用的私有断言 helper | canary 零副本、精确 command/args、单 init container、无 Secret 全部原断言 |
| TestListingKitImageAgentSteadyStateRendersNoFiniteCanary | base/prod 两项旧 wantWorkers 改为 forbiddenWorkers，要求二者缺席 | 真实 kustomize 渲染、无 finite canary，保留其它当前资源，不把零对象当成功 |
| TestListingKitCommercialImageAgentWorkersUseExactSecretAndConfigScope | 仅删除两个已退 Deployment 的 expected 行及由此确实失去全部消费者的局部集合 helper | 同文件保留 workload 的精确 Secret/Config allowlist、凭据隔离和失败关闭；不可放宽成子集 |
| TestListingKitRetainedWorkloadsExcludeInvitationCredentials；TestRetiredTencentSMSCredentialsAndRouteAreAbsent | 仅从必读工作负载表移除两个已退路径，新退休 guard 强制缺席 | 其它 workload、Secret/配置/路由禁令、原遍历/断言不变 |
| TestLegacySubscriptionMeteringClosureIsRetired | 只从必需读取清单删两旧 Deployment 行 | 所有订阅/Token/OpenMeter 禁入、其余文件检查不变 |
| TestAutoMigrateCreatesOwnerScopedV2TablesWithoutTouchingLegacySchema；TestOrganizationImageAgentInitCreatesOnlyCurrentOwnerTablesPostgres | 当前新安装 expected 表清单仅减旧 v2 effect；补显式缺席 | 当前组织 schema 精确集合、无权限升级、空库/ledger-only installer 安全边界、其它表和存量不动断言 |
| §5.2 的域/Temporal/worker 旧 lane fixture | v2/raw v3 可执行正例变缺席/拒绝；旧 one-shot stub 退出；同输入业务断言接 staged/组织 owner，补当前 identity/member/authorizer | 原有效权限/幂等/同异载荷/计费/取消/绝对 deadline/UNKNOWN/fence/腐坏/批准/响应丢失断言，不能删测试整体掩盖故障 |
| history_replay_test.go | 删除仅指向三 v2 fixture/旧 v2 handler 的 replay、v2 child/旧批准 wire 正例；共享 probe 如混合则改到组织 scope | 当前 V3 pre-finalization、recovery-start、approval-key、result digest、budget/deadline 等有效版本证据；不是取消版本/恢复义务 |
| TestExecutorAcceptsHistoricalV3InputWithoutPolicyFields 及 product_image_executor_contract_test.go 的同根默认 fixture | 无 TargetPlatform/ImagePolicyContext 的旧 v3 正例改为拒绝，断言 quote/provider/review 计数为0；仍有效的规则测试 fixture 改为现有 product/zz/default/general 明确策略 | 当前准确 ProfileResolver key/version、有效政策/主图/场景/审核规则、图组/Artwork staged 和 JSON/hash；不把缺策略 fallback 留作兼容 |

tests/listingkit_backend_retirement_test.go 的 TestLegacyImageAgentHTTPRuntimeClosureIsRetired 现役 ForMode/WithOptions/Service/CHAIN-1/canary 保活项不需删除，整函数保持；其它 root checks、.golangci.yml、Makefile、两套 workflow/其它 .github/workflows、完整 inventory、历史/计数/ceilings/检查逻辑、依赖和保护规则零改。若原 CI 发现本表遗漏的真实冲突，先给根因、同根核对和准确补丁；不能自动放宽 guard 或以重跑代替修复。

验证复用原能力：新退休 guard 真实 RED→GREEN；相关域/Temporal/store/tools/worker/runtime/CLI 原测试及适用 race；当前图组/Artwork、组织采集单主图、原 CHAIN-1 有效回归；现有 CI 自建临时 DB/SDK fixture 和 kustomize/shell guard；受影响入口仅编译；稳定准确 HEAD 的原 CI。无需重复手工跑完整 CI。旧环境/真实 provider/真实 canary/部署为 NOT_RUN，不以合成结果称运行或产品验收通过，不新增验收平台。本机如需新启动运行资源，须另有准确授权；本设计不授权使用任何已有资源。

## 7. 单批交付、复核与停止点

一个主分支/主要 PR、一个 Writer：先抽取 §4.1 当前纯行为并解除旧端口耦合，再切掉旧模式/worker/wire/ledger，收齐 fixture、两声明、原检查与文档尾部；这是同一完整闭包的实现顺序，不机械拆批。切换后新代码不能依赖旧 owner，不留 alias、fallback、双读写或第二事实源。

设计复核须检查真实消费者、组织独立 worker、当前图组/Artwork/单主图、持久化与有效 V3、前批 KEEP 和 §6.1。finding 先按 AGENTS 分类并指出当前 Must/后果；普通非 Blocker 不重开无关架构，最多两轮正常设计复核。只有适用复核完成、用户确认且 #301 当前段明确 IMPLEMENTATION_READY 后才能改生产源码。

沿用已有 GitHub 复核能力，失败后只作一次必要重试。**未经用户本轮授权不默认新增 AI 互验线程；设计复核与源码最终交付复核授权分开。** 未变化代码有效证据可按规则复用，准确 HEAD 的原 CI 必须单独核对。设计文本通过不等于源码/运行验收或合并授权。

完成实现后记录准确 HEAD、实际 diff、保留能力、原 CI 与独立最终复核，交用户决定合并；合并后按实际 merge SHA 核对完整 push/main run。不得管理员绕过/改保护/强推，HEAD 变化遵守准确授权。#301 保持 OPEN，关单另授权；不操作既有实例、数据、凭据或进程，不改 workflow/inventory，不增加内部试用前置。
