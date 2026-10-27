# #301 旧 ImageAgent 隔离 HTTP 与闲置启动装配完整退休设计

> **DESIGN_ONLY / SOURCE_NOT_STARTED / AWAITING_USER_CONFIRMATION**。本阶段仅本文、Architecture README、Legacy Register 三文档，Design Basis: N/A / Ready。未来源码为 **Independent Architecture / DESIGNING / NOT_READY**；适用设计复核、用户确认 §1–§7（含 §4.2、§6.1）及 Issue/PR 记录、读回 IMPLEMENTATION_READY / Ready 后才实施。设计基线为实际 main `730d16a0100f5929d88f83a28cad6f76986d66c0`；其 push/main Task Processor CI40464764246 attempt1 整 run SUCCESS，只是基线证据，不是本设计或未来源码的结果。#972 已完成，源码收口证据见其 PR；本轮不复做。

## 1. 用户结果、完整范围与不做

维护者继续使用当前商品采集、图片、SDS/POD、资产审批和独立 SHEIN 登录/Store 同步。本批消除一个已不在默认应用注册的旧 HTTP 宿主、一条旧 Task 图片启动/预检链和数个默认启动门面，避免新代码误接它们。没有新页面、使用步骤、运行准备或内部试用前置。

| 一次收齐 | 退出 | 保留的真实入口 |
| --- | --- | --- |
| 隔离 HTTP 与 Task 尾部 | `internal/imageagent/httpapi` 全包、`NewImageAgentOrganizationApplication`、静态 OrganizationImageBinding/map、Task launch/preflight 的两个方法/四类型/四 helper/专属测试 | 当前 acquisition main-image、full ImageSet、Artwork 路由和当前领域 Service/Repository |
| 默认启动/装配门面 | 无调用的 Start/默认 Run/Dial/默认依赖与注册 wrapper、无 worker 的 runtime canary、两个测试专用启动转发、闲置 queue/组合字符串 recovery-ID helper | 显式 WithOptions/ForMode 路径、Organization 当前 worker/client、WithWorker canary、结构化 recovery-ID |
| 有效断言与文档尾部 | 旧 HTTP/Task 实现细节测试和旧 JSON/SSE 专属 DTO；旧 KEEP 的精确保活范围 | 组织隔离、真实权限、审批、预算/取消、响应丢失、恢复和现有整链验收；按 §4.2/§6.1 改接当前 owner |

未来同一主要 PR **49 唯一路径**：9整文件退休（其中5生产/4测试）、8混合生产、13混合测试、1退休 guard、7当前文档、11历史顶注。完整列表见 §5，不因文件或测试数量拆批。本设计当前只修改三份文档，不预写删除完成。

**Must**：闭包收齐；当前能力与有效安全断言保持；无旧 API/alias/fallback 回流；真实 worker/canary、Temporal wire/JSON/hash/版本/队列/ID/恢复不变；无既有资源操作。**Should**：复用当前 Service/授权/Repository、标准库和 Temporal SDK，不造新运行/验收平台。**Out of Scope**：新增产品、授权/租户/身份/计费语义、schema/表/事实 owner/迁移、provider 调用、状态/事务/retry/UNKNOWN/recovery 机制，workflow/完整 inventory/依赖/保护规则以及既有实例/数据/凭据/轨迹/文件/进程。#301 保持 OPEN。无新 Accepted Risk；物理运行状态 UNKNOWN，不探查。

Threat Model 只覆盖误删当前图像/采集能力、错误组织或原成员读取/审批、绕过真实授权/预算、丢失重试/响应丢失/UNKNOWN 断言、误删真实 worker/canary 和旧 API 回流，不扩展全局安全要求。

## 2. Design Basis、Authority 与旧 KEEP 替代

依据：[全新系统产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[项目边界](project-boundaries.md)、[目标架构](project-target-architecture.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、用户本轮完整设计授权及 #301 当前段。源码阶段使用 Independent Architecture：跨 HTTP、领域合同、worker 装配和测试消费者，超过30路径，并显式移除旧公共 Task 合同；需复核消费者、授权及持久化边界。没有 UI/IA/交互改变，Figma N/A；不借退休新增领域/状态/事实 owner。

| 旧 KEEP/历史依据 | 本次准确替代 | 继续有效 |
| --- | --- | --- |
| #339 Organization scope / #334 review admission | 独立旧 HTTP constructor、binding map 和旧接口必须存在的限定，以及 HTTP fixture 对这些对象的直接依赖 | 真实验证身份/所选组织/原成员、仓库隔离、执行时 live grant、审核调用/记账/发送限制和凭据准入 |
| #388 CHAIN-1 | 只有旧 ImageAgent HTTP 宿主才能承接图片段、HTTP snake_case DTO 和 socket-drop 专用代理的实现限定 | Source→Review Apply→真实图片业务用例→显式 Asset 批准→Store→草稿/readiness；同版本、权限、并发/异载荷冲突、丢失响应及重建证据；脚本/Compose/driver/receipt 不退休 |
| #953 OpenAI 闭包 | 两份 root integration 与 Chain1 图片段逐字零改的旧 KEEP，仅按 §4.2/§6.1 指定改接 | 全部有效审核记账/发送/取消/UNKNOWN/敏感信息、Source/Review/Store/草稿及其它代码保持 |
| #846 local acceptance | ImageAgent 全部内部入口/测试永久保活的宽泛含义 | 当前 UI、身份、issue357、真实 ImageAgent/Temporal/独立 SHEIN/Store；已退休旧验收入口不重建 |
| 旧 ImageAgent superpowers 八计划 / Phase3 装配 / Module Target Mapping | §5 指定的旧 HTTP/Task、默认 wrapper 与 runtime 无 worker canary | 领域/当前 HTTP/组织 worker、已有 v2/v3/Organization wire 和 replay、WithWorker canary、业务与恢复规则 |
| #972 SDS source closure | `internal/sds/README.md` 仍称 `adapter/product_source` 为现役的过期顶注 | 当前 SDS/POD 原 owner、既有数据不动和原 dated 历史后缀 |

只替代以上准确对象；旧文档引用/旧公开方法/测试不是当前需求。不存在迁移或兼容义务证据，不读取真实状态寻找义务。11份历史依据未来只追加本批顶注，起始 main 完整原 Git blob 后缀、数字、勾选、历史 PASS/FAIL/UNKNOWN 逐字保留。Module Mapping 原历史行不改，以精确新顶注校准当前归属。

## 3. 实际消费者、当前 owner 与注入

在基线 **3137 tracked Go** 做标准 Go AST 只读核对，包含 tests/hack/未选编译源，检查解码 import、别名/选择器/函数值、同包调用，再查 tracked 文档、TS、脚本和配置。未读取未跟踪配置/凭据；静态结果不代表运行状态。

| 退役对象 | 实际消费者与闭包 | 当前保留路径 |
| --- | --- | --- |
| imageagent/httpapi 七文件 | 唯一生产 importer 是 `app/httpapi/imageagent_organization_application.go`；默认应用不注册。constructor 只由 root Organization integration 与 Chain1 验收创建，静态 #388 guard 保活 | `current_application`→`buildAcquisitionImageModule`→organizationImageCatalog→Service；`WithFullImageSet`→buildFullImageApplication→fullImageModule/Artwork→Service |
| LaunchTaskRun/PreflightTaskRunAssets | 只有旧 Handler 和 `service_task_launch_test.go`；四DTO/四helper无其它业务消费者。测试 helper recordingImagePolicyAvailability 被 `policy_context_test.go` 使用，必须先移动 | acquisitionMainRunInput、当前 catalog/图片源选择、Service.Start/有效 policy/allowlist、当前 ImageSet 产品入口 |
| runtime 默认启动与无 worker canary | Start/StartWithOptions/默认Run/Dial无当前调用；无 worker canary及其私有依赖只有本包两测试；两 Background 启动转发只有本包测试 | `cmd/image-agent-temporal-worker`→Resolve…ForMode→Run…WithOptions；`--canary`→Run…WithWorker；current-application→DialOrganization…/NewOrganizationImageSetWorker |
| 默认 resolver/ImageSet/registration wrappers | ResolveImageAgentTemporalDependencies 仅转 ForMode(v3)；NewImageSetTemporalDependencies 仅被自己 nil-input 测试使用；RegisterActivities无外部调用，RegisterWorker仅旧注册测试 | 明确 ForMode；buildFullImageApplication→NewManagedImageSetTemporalDependencies；现役 NewWorker→RegisterWorkerForMode→RegisterActivitiesForMode |
| TaskQueueName / EffectRecoveryWorkflowID | 前者仅自身 ambient env，后者仅自己的确定性 ID 测试；不参与生产恢复启动 | DefaultTaskQueue/显式 WorkerConfig；EffectRecoveryWorkflowIDForSlot 和 scopedWorkflowID 原样保持 |
| 数据和副作用 | 删除的 HTTP binding 是进程内 fixture map，不是持久化 owner；Task DTO不是存储 schema。无DDL/迁移/Drop | ImageAgent Run/Plan/Event/Effect/预算原 Repository；Product Catalog/Asset/Approval原 owner；Temporal SDK 与对象/模型原 ports |

保持 `Service.Start/Get/RestartFailed/ReplacePlan/RetrySlot/RecoverEffect/ApproveResults/Cancel/Resume/ListEvents`、WorkflowClient 全合同、现有 v2/v3/Organization Activities/Workflow 及其 Store 不变。某方法当前只有领域测试并不自动构成退休理由：本批只退明确的 Task-first workspace 启动/预检门面；通用命令/事件与协议不是这条旧 HTTP 的专属实现。当前 worker/canary 不是本批退休对象，`dataservices.RegisterWorker/RegisterActivities` 等同名独立实现也不触碰。

## 4. EXTRACT/RETIRE、权限与持久化边界

```text
Legacy decision: EXTRACT + RETIRE
Reusable behavior: 当前组织/成员隔离、真实权限、显式资产批准、预算/取消、记账/UNKNOWN、幂等/响应丢失及恢复断言
Current owner: 当前 HTTP 路由 + ImageAgent Service/Repository/Temporal + authruntime/workbenchcontext/authz + Product/Asset 原 owner
Cutover/deletion condition: §4.2 有效断言先改接；§5 旧闭包全删；§6.1 原检查精确转换；原 CI/最终交付检查完成后，另获准确 HEAD 合并授权
```

### 4.1 运行、数据和协议不变

没有新生产 owner/接口/表/字段/注入/迁移，不执行初始化器、Goose、旧或当前 worker/CLI/验收脚本。Task launch 自造 ID 算法只随旧门面删除；不得改当前 acquisition/ImageSet 的 ID、原已保存载荷、hash/规范化、Plan/Policy/版本、Temporal workflow/activity/队列名或 TaskQueue 环境输入的现役显式读取。

Repository 仍由原组织/原成员 scope 限制。HTTP 当前路由继续原验证/LiveWrite/权限；worker 每次 effect/provider 前原执行授权保留。事务、并发 CAS、相同动作同载荷 attach/不同载荷冲突、预算 reservation、取消/deadline、发送前释放与 UNKNOWN owner、失败/重启/响应丢失/恢复均不变。既有表、记录、对象、Temporal history/队列、配置文件和进程不动；全新安装 schema也不改。

### 4.2 有效断言改接，不复刻旧宿主

**不在 tests 重建旧 HTTP Handler、旧路由、旧 DTO 或旧 binding-map service。** 不新增生产 wrapper/alias、兼容 API 或通用验收工具。用现有 typed Service 命令和当前 HTTP owner 区分验证边界：

1. **root Organization/Review integration**：保留原 PostgreSQL schema、真实 Repository/Temporal converter/Activity、IAM httptest/凭据合成 fixture 和全部领域/授权/计量断言。删除 scope339Fixture.server、scope339Request、scope339Path/scope339Body 与 OrganizationImageBinding 依赖；改 typed StartRunInput 和直接 Service.Start/Get/RecoverEffect。测试上下文使用原 verifier→现有 Resolver.Resolve(LiveWrite/CachedRead)→`authz.AuthorizeOrganization`，不自写权限判断；保持缺组织/跨组织/非原 actor、撤权/权限读取失败在仓库/工作流/凭据前拒绝。固定 Catalog port 只返回已发布合成 snapshot 的归一化值，不复制生产 binding/selection 逻辑；真实 Catalog 组织/actor/版本/publication校验继续由现有 organizationImageCatalog 测试保护。`verifyActivity` 内部实际 capability/记账/取消/发送断言不变，仅去掉“actual HTTP”过时说明。review334Fixture及其它 failure matrix不变，只改引用旧 server 的一测试为 typed domain 路径并准确命名。
2. **root 安全断言逐项保留**：Home A/selected B 入工作流和持久化；同ID原载荷幂等与异载荷拒绝；组织/旧scope五表/events/effect隔离；GORM event失败全回滚且starts=0；接受start但响应丢失重试只一durable run/同workflow ID；blocked recovery保持原组织队列/ID；凭据解析前验证scope；review record failure/response_cancel/known_cost/unknown_cost；真实 Temporal budget-before-first-provider。原 `TestServiceAuthorizationRequiresExactOrganization` 保持原位置/完整断言，不因文件混合而误删。
3. **当前 HTTP 的有效断言**：在原 acquisition/ImageSet 两测试文件追加有界用例，实际调用 `readAcquisitionImageJSON`/`readFullImageJSON` 与当前路由，保护未知/重复字段、identity伪造、隐藏超限、非法UTF-8、query/body、跨组织/成员、错误映射和安全URL。当前 8KiB/128KiB 上限保持，不能恢复旧1MiB标准；原 handler snake_case DTO、Task launch/preflight、旧 SSE cursor/schema/heartbeat、旧路由注册和按旧输入自造ID 属实现合同，RETIRE。事件持久化/顺序/ListEvents等领域原测试保持。原 budget 零值/负值/存在性仍由领域 budget/current command 测试保证，不把旧DTO解析当有效运行入口。
4. **CHAIN-1 保留整条可重复验收**：script/Compose/Temporal acceptancetest driver/clean-HEAD attribution/真实PG+Temporal/阶段receipt全不改，不能删除 #388 有效验收。Source→Review Apply、Store创建、Draft保存/readiness及其HTTP响应丢失段原样保持；仅图片段以 `imageagent.NewService(...WithOrganizationScope...)` 承接真实业务用例，输入来自实际已Apply精确Catalog snapshot，复用现有归一化/资产转换，不 seed 最终run/批准/资产事实。原 chain1LiveWriteContext 扩展/窄helper只组合真实 verifier/Resolver/authz；写前真实LiveWrite+ImageWrite，读用ImageRead。删除 chain1ImageApplication/chain1RunView；chain1WaitRun改查Service.Get的真实RunProjection。typed Start/ApproveResults保持原run/plan/action/digest；错误digest/撤权/worker撤权/并发同action attach/异载荷冲突、provider/公开对象/approval rows为0、恢复后唯一批准/inventory/再次变更不沿用批准全部保留。
5. **图片批准丢响应分两层实证**：Chain1图片段的两并发Approve调用在真实命令完成后显式丢弃成功结果并返回测试sentinel，再按原identity重建Service/worker/Temporal核实；不得把未完成/被拒绝当已提交或直接插回执。原旧HTTP socket proxy移除。另在已有 `imageagent_completed_replay_postgres_test.go` 加当前 acquisition `.../main-image/runs/:run_id/approve` 的真实HTTP提交后断连/同请求重放案例，使用原实际Service/Repository/Approval reader；保持原一次业务批准/动作冲突/安全URL/跨actor拒绝断言，不以spy代替持久化。这两个结果分别标为domain/HTTP，不把分项证据冒充重新跑过完整整链。
6. **Task 专用测试**：整退 service_task_launch_test，但先把 recordingImagePolicyAvailability类型/方法原样移至同package的policy_context_test；现有policy测试完整保持。Task特有字段/ID/style预检约定退出；仍有效的显式source/不信crawler排序、unknown asset拒绝、组织/当前身份、admission/有效policy先于catalog、同request幂等/异载荷冲突，在 Service.Start原tests和当前acquisition owner追加/保留断言；不复制taskLaunchRunIdentity或保留旧test输入类型。
7. **运行测试**：当前15个runtime测试的11个启动/授权/关闭测试只把两个私有Background wrapper调用改为现有 Context helper，传原context和显式模式；原dial/close/stop/partialports/无授权不dial断言不变。两个无worker canary测试改测现有WithWorker注入helper，保留context/队列/dial先后/close owner/未授权不启动不执行；原两个WithWorker测试原样保持。默认ImageSet nil-input测试改Managed函数（九个nil参数），V2注册测试改RegisterWorkerForMode(V2)保持完整原workflow/activity集合；recovery-ID测试改结构化ForSlot(..., actionID="")保持原期望字节和attempt区分，不改生产ID算法。

以上是本批设计决定，不留“实现时再选 owner”。若真实执行要求更改生产授权/状态/预算/协议/持久化，停止受影响部分报告具体Blocker，不能用新的test-only production replica掩盖缺口。任何任务自有PG/Temporal或外部端口替身的真实运行须有匹配权限；本轮设计一律 NOT_RUN，不读取已有DSN/env/凭据、不运行入口。

## 5. 精确删除/保留矩阵

### 5.1 混合生产只允许以下声明退出

| 路径 | RETIRE | KEEP |
| --- | --- | --- |
| app/runtime/image_agent_temporal_runtime.go | StartImageAgentTemporalWorker、StartImageAgentTemporalWorkerWithOptions、DialImageAgentTemporalWorkflowClient、RunImageAgentTemporalWorker、RunImageAgentCompatibilityCanary；私有 startImageAgentTemporalWorkerWithDependencies、startImageAgentTemporalWorkerWithOptionsAndDependencies、runImageAgentCompatibilityCanaryWithDependencies；type imageAgentCompatibilityCanaryDependencies | 其它声明/环境配置/错误处理保持，包括 Context helper、Run…WithOptions、DialOrganization…、NewOrganizationImageSetWorker和WithWorker canary全部链 |
| app/worker/imageagent/dependencies.go | ResolveImageAgentTemporalDependencies | Resolve…ForMode及全部私有resolver/provider/database/credential接线 |
| app/worker/imageagent/image_set_runtime.go | NewImageSetTemporalDependencies | NewManagedImageSetTemporalDependencies及所有执行/恢复/预算/对象逻辑 |
| imageagent/temporal/activity_registration.go、worker.go | RegisterActivities、RegisterWorker各一默认转发 | …ForMode及实际全部注册/生命周期 |
| imageagent/temporal/types.go | TaskQueueName、EnvTaskQueue Go常量、EffectRecoveryWorkflowID组合字符串helper；仅因此无用的os import | 所有其它常量/类型/字段/函数，现役环境输入与协议、ForSlot/scoped ID/原版本原样 |
| imageagent/service.go | LaunchTaskRun、PreflightTaskRunAssets、taskLaunchRunIdentity、normalizeTaskLaunchStyleIDs、primaryTaskLaunchSource、validateTaskLaunchStyleIDs；仅因此无用的uuid import | Service全部字段、其它选项/方法及实际执行/事务/鉴权/恢复逻辑 |
| imageagent/ports.go | TaskRunLaunchInput、TaskRunLaunchResult、TaskRunAssetsInput、TaskRunAssetPreflight四整声明/专属注释 | ImagePolicyAvailability、StartRunInput、WorkflowStart、identity、其它所有合同/字段/JSON |

上述路径均相对internal。总计22个指定混合/宿主函数方法退出（其中宿主2个随整文件删除），另有整HTTP包全部声明、四Task类型、一个runtime私有依赖type、一个queue常量与专属字段；不把整个类型内字段重复计数。无新增生产函数/包/兼容alias。只作必要unused-import处理，不顺便格式化其它函数。

### 5.2 49路径冻结清单


**整文件 RETIRE（9）**

- `internal/imageagent/httpapi/dto.go`
- `internal/imageagent/httpapi/handler.go`
- `internal/imageagent/httpapi/handler_task_launch_test.go`
- `internal/imageagent/httpapi/handler_test.go`
- `internal/imageagent/httpapi/module.go`
- `internal/imageagent/httpapi/routes.go`
- `internal/imageagent/httpapi/routes_test.go`
- `internal/app/httpapi/imageagent_organization_application.go`
- `internal/imageagent/service_task_launch_test.go`

**混合生产：只改 §5.1（8）**

- `internal/app/runtime/image_agent_temporal_runtime.go`
- `internal/app/worker/imageagent/dependencies.go`
- `internal/app/worker/imageagent/image_set_runtime.go`
- `internal/imageagent/temporal/activity_registration.go`
- `internal/imageagent/temporal/worker.go`
- `internal/imageagent/temporal/types.go`
- `internal/imageagent/service.go`
- `internal/imageagent/ports.go`

**混合测试：只改 §4.2/§6.1（13）**

- `internal/app/runtime/image_agent_temporal_runtime_test.go`
- `internal/app/worker/imageagent/image_set_runtime_test.go`
- `internal/imageagent/temporal/workflow_test.go`
- `internal/imageagent/temporal/effect_recovery_workflow_test.go`
- `internal/imageagent/service_commands_test.go`
- `internal/imageagent/policy_context_test.go`
- `tests/imageagent_organization_integration_test.go`
- `tests/imageagent_review_invocation_integration_test.go`
- `internal/app/httpapi/chain1_local_product_acceptance_test.go`
- `tests/issue388_chain1_acceptance_contract_test.go`
- `internal/app/httpapi/image_set_module_test.go`
- `internal/app/httpapi/imageagent_acquisition_module_test.go`
- `internal/app/httpapi/imageagent_completed_replay_postgres_test.go`

**退休 guard：只追加本批检查（1）**

- `tests/listingkit_backend_retirement_test.go`

**当前文档：本批依据与当前说明（7）**

- `docs/architecture/legacy-imageagent-http-runtime-closure-retirement-boundary.md`
- `docs/architecture/README.md`
- `docs/refactoring/legacy-register.md`
- `docs/refactoring/module-target-mapping.md`
- `docs/engineering/imageagent-organization-scope.md`
- `docs/development/chain1-local-product-acceptance.md`
- `internal/sds/README.md`

**历史依据：只加准确顶注，原完整blob后缀不改（11）**

- `docs/engineering/evidence/issue334/admission.md`
- `docs/superpowers/plans/2026-08-26-image-agent-assisted-planning.md`
- `docs/superpowers/plans/2026-08-26-image-agent-automatic-repair.md`
- `docs/superpowers/plans/2026-08-26-image-agent-core-manual-workflow.md`
- `docs/superpowers/plans/2026-08-27-image-agent-recovery-compatibility.md`
- `docs/superpowers/plans/2026-08-28-image-agent-budget-authorization.md`
- `docs/superpowers/plans/2026-08-29-image-agent-external-recovery-workflow.md`
- `docs/superpowers/plans/2026-08-29-image-agent-workspace-authorization-and-recovery.md`
- `docs/superpowers/plans/2026-09-01-internal-target-architecture-phase3-product.md`
- `docs/architecture/legacy-openai-adapter-closure-retirement-boundary.md`
- `docs/architecture/legacy-local-acceptance-closure-retirement-boundary.md`

### 5.3 明确保留，不自动扩范围

- `cmd/image-agent-temporal-worker`、`cmd/current-application`、当前 app/httpapi生产路由/目录/转换、app/imageagent、imageagent Domain/Store/Temporal/Tools、current policy/AI/图像、worker有效接线、采集/原Amazon浏览器及debug、SDS/POD/资产审批、独立sheinlogin与Store观察/同步均保持。
- `scripts/chain1-local-product-acceptance.ps1`、其Compose、`internal/imageagent/temporal/acceptancetest/driver.go`、现役canary/发布安全检查原样保持；设计与实施都不运行这些入口。
- 完整workflow、完整inventory（含CURRENT/历史/计数/ceilings）、依赖、保护规则和其它检查零改。旧HTTP/Task闭包不持有logger/OpenAI inventory记录，无需校准inventory。
- Dub/OpenMeter等隔离SDK/PoC并非本批旧HTTP专属尾部，不因零生产import就退休。不存在现有凭据/配置/轨迹/进程清理授权。

## 6. 验证与原检查处置

设计阶段：核对tracked消费者与全部路径、文档链接、原正文/历史后缀保持、只三文档diff、diff --check、原PR CI和适用独立设计复核。未来源码按同一批先完成规则/fixture/原断言改接，新增一个能在旧代码真实失败的闭包guard，再删专属代码。必要本地测试只覆盖受影响包/原合同；已有测试与原CI复用，不新增runner或全量故障平台。

### 6.1 原检查冻结矩阵

| 原文件/函数或组 | 唯一允许变化 | 保留与失败关闭 |
| --- | --- | --- |
| 整HTTP三测试/Task专属测试文件 | 退出旧路由/DTO/SSE/Task机制检查；有效断言先按§4.2落实，shared policy helper移动后才删 | 当前HTTP/Service/组织scope/授权/预算/持久化/取消/UNKNOWN/recovery测试保留 |
| service_commands_test.go / policy_context_test.go | 只追加§4.2(6)当前Service回归、原样接收recordingImagePolicyAvailability；原测试函数/原断言不删不放宽 | 有效policy/startgate、资产/身份/幂等与field absence/hash等保持 |
| image_agent_temporal_runtime_test.go 原15函数 | 11启动函数改两私有wrapper调用为Context helper；两无worker canary函数改为WithWorker helper并准确命名/补实际worker替身；余下2个WithWorker函数零改 | 原context/模式/queue/authorizer/dial/stop/close/端口断言保持；不得删missing-close-owner负例 |
| image_set_runtime_test.go::TestImageSetExecutorDoesNotDispatchUnconfirmedOrSingleImageInputs | 仅NewImageSetTemporalDependencies(nil...)改当前Managed函数的九个nil参数调用 | 原四executor拒绝及ErrValidation断言不变 |
| workflow_test.go::TestRegisterWorkerPreservesFrozenV2Compatibility | 仅RegisterWorker(registrar,activities)改RegisterWorkerForMode(registrar,activities,WorkerWireModeV2) | 原精确workflow/activity列表逐字保持；其它函数/fixture/JSON/replay零改 |
| effect_recovery_workflow_test.go::TestEffectRecoveryWorkflowUsesDeterministicIDAndAttachesDuplicateStart | 去duplicateKey拼接，三调用改ForSlot结构化run/slot/attempt/空action | 原ID期望字节、相同输入相同ID、nextAttempt不同断言保持，其它函数零改 |
| tests/imageagent_organization_integration_test.go | newScope339Fixture和scope339Fixture绑定字段/旧server/request/body/path移除；§4.2列出的主scope/realTemporal budget/init response loss/recovery改typedService；verifyActivity只调整旧HTTP说明 | credential admission、service exactorg、全部真实PG/Activity/provider/记账/隔离/回滚/恢复断言保留，不缩成mock Service |
| tests/imageagent_review_invocation_integration_test.go::TestOrganizationReviewHTTPActivityRecordFailureAndCancellation | 准确改名为ServiceActivity并用typed start/真实scope；同四mode/原verifyActivity | 其它原测试/helper/计量/发送/释放/并发/敏感信息全部不变 |
| chain1_local_product_acceptance_test.go | 仅TestChain1LocalProductAcceptance图片段、chain1ImageApplication、chain1RunView、chain1WaitRun/必要typed fixture和live授权helper；删除旧HTTPsocket专用proxy，按§4.2(5)保留两层丢响应实证 | Source/Review/Apply、Store/Draft、真实Temporal重启、worker/对象/行数、并发/动作/版本/readiness及其余helper不改；受控ports纯测试不删 |
| tests/issue388_chain1_acceptance_contract_test.go::TestIssue388Chain1AcceptanceAssemblyContract | 只把必需constructor字符串替换为当前typed Service.WithOrganizationScope/Start/ApproveResults与真实授权owner检查，并检查旧package/constructor不得回流 | 原真实业务owner、禁止直接seed批准/草稿/Store、tenantbridge禁令、script/Compose/receipt全部其它断言不变；ScriptExecutionReceipts原函数及helper逐字保持 |
| 当前image_set_module_test.go、imageagent_acquisition_module_test.go | 只追加§4.2(3)当前边界测试，不改原函数 | 真实当前上限/strictdecode/原身份/版本/批准与安全URL保持，不改生产parser |
| imageagent_completed_replay_postgres_test.go | 只追加当前HTTP批准提交后断连/原key重放的PG实证，复用现有fixture | 原TestAcquisitionCompletedApprovalHTTPReplaysFromExactOwnerFact及helpers不改；跨scope/receipt/readiness保持 |
| listingkit_backend_retirement_test.go | 只追加TestLegacyImageAgentHTTPRuntimeClosureIsRetired，不改任何原函数/helper | 对9文件缺席、旧包import/声明禁入、§5.1旧声明缺席、显式worker/canary/currentHTTP/typed安全测试及脚本/driver/fixture保留作失败关闭检查；required source缺失必须失败，不加skip |

所有其它原检查/算法/输入/错误处理不改。原测试证明§6.1遗漏时先查同根sibling，给用户准确最小补丁、未应用证据和影响再批准；不得用目录不存在skip、弱化want/删负例或改workflow/inventory换绿灯。实现者不能用新增guard代替真实原断言。

### 6.2 证据、NOT_RUN 与停止点

- 必要自检候选：imageagent/temporal、app/runtime、app/worker/imageagent、auth/current HTTP和root受影响测试；只编译current-application、image-agent-temporal-worker、shein-login-worker及原采集入口，明确compile不是运行。PG测试/真实Temporal整链的执行授权和实际结果单列；没有真实执行保留NOT_RUN/原env-skip，不新增skip或把Skip当PASS。
- 沿原CI、Code Health、guard、必需和安全检查，CI准确绑定设计/源码各自HEAD。实现期用原断言/完整fixture/原函数未改清单核验范围，历史11文档逐份字节后缀对比；当前JSON/hash/版本/wire值完整原合同测试不变。
- 不因本设计或本批验收新增试用前置。实现完成、CI、独立代码检查、merge SHA push/main CI与真实用户/运行验收分开，不能自行宣称产品验收/上线。

## 7. 执行、复核与交付

1. 本轮只读调查、三文档、一个主要PR、原CI和设计复核；未来Independent Architecture暂为NOT_READY/SOURCE_NOT_STARTED。设计复核看真实消费者、49路径、§4.2断言接缝和§6.1。未经用户确认不修改生产/测试，也不运行旧宿主或准入外资源。
2. 设计复核满足且用户确认后，同一PR记录/读回IMPLEMENTATION_READY/Ready，再连续实施完整闭包。单实现者，不为小片增加独立PR/Reviewer；仅在真实Blocker时退回设计，非Blocker按既有分类收敛。
3. 先用当前owner建立必要RED→GREEN/断言改接，再删闭包；稳定候选才完整原CI。最终检查完整真实diff及保留能力，不只采信作者报告。GitHub自动复核失败只允许一次重试，不在同SHA重复提交评审请求；**未经用户本轮授权不默认新增 AI 互验线程**。子代理设计授权不含未来源码最终复核。
4. 证据/准确HEAD/CI/正式review/inline finding与处置放主要PR；Issue只更新当前段，新读正文并保持从“以下保留 #726 已合并批次与历史证据，不扩大本轮权限。”起后缀逐字不变，不修改历史勾选，正文≤65536。#301 OPEN。
5. 源码全部完成后交准确HEAD请求正常合并，expected-head语义，不管理员绕过/强推/改保护；合并后只以实际merge SHA对应event=push/branch=main整run completed/SUCCESS确认。设计/CI不授权合并、部署、关单、provider调用或既有资源操作。
