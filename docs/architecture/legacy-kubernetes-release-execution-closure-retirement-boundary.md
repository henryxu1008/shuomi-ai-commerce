# 旧 Kubernetes API/Task 发布执行与 UI 专属接线：完整退休闭包

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

> Status: **DESIGNING / 待独立设计检查与用户确认，Ready=false；未实施。** Design Basis: **Independent Architecture**。调查与开工基线 main `d255feb7f1349108a466530353d2637080a911be`（2026-10-20）。本阶段只交付本文与四索引的提案说明；准确设计 HEAD、CI、复核及授权维护在同一主要 PR。未经用户确认与准入，不改声明、脚本、策略、测试或运行指南，不执行发布/诊断/验收工具。

## 1. 产品结果、Authority 与阶段

使用者是维护当前应用与内部试用的开发、运维人员。结果是退出已经无法交付当前应用的旧 API workload、旧发布专属 schema/回填 Job 及其调用尾部，避免操作者仍从源码创建旧对象或沿隐式旧 API 目标操作。此维护没有正在阻塞的内部试用核心操作，不作为试用新前置；用户明确选择先设计完整闭包，确认后整批实施。

一个 Delivery Batch / 一个主要 PR，统一处理声明、聚合接线、共享执行器的旧默认值、源 RBAC 库存、原测试与准确说明。不按文件数、脚本/声明/测试或模块机械拆批。当前 UI、ImageAgent、正常 schema CLI、通用工具和既有能力继续归原 owner。

Authority：[Greenfield 产品基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Project boundaries](project-boundaries.md)、[Issue 规则](../engineering/issue-driven-development.md)、已确认的[旧发布身份退休设计](release-identity-preflight-retirement.md)与[#846 本地验收退休边界](legacy-local-acceptance-closure-retirement-boundary.md)。当前内部试用沿 [Account Compose](../../deployments/docker/account-compose/README.md) 与既有私密 manifest；[RUN-1](../operations/current-application-run1.md) 仅为独立验收环境，不替代试用或生产交付。没有 UI/IA/交互修改，Figma N/A；全部 web 源与测试零改。

Must：完整删除 §5.1 专属源与 §5.2 接线；保留 §5.3 当前能力及所有仍有效安全断言；两套 CI workflow 和其它 workflow、inventory、历史计数、ceilings、依赖/锁/保护零改；不操作实例、数据、凭据或既有报告，不增加试用前置。Should：复用现有 Bash/Go guard、Conftest/OPA 与当前 owner，不新建平台或抽象。

Out of Scope：当前应用/身份/领域行为、生产发布替代流程、真实部署/撤权/迁移/清理、旧数据兼容、当前 ImageAgent 恢复协议或数据库变化、issue357 全部源/文档/测试、其它 owner 的 #692。没有新增 Accepted Risk；历史实例、卷、IAM、Job、Temporal histories 与 durable assets 的实际状态和归属均 **UNKNOWN**，不得推断不存在或可清理。真实环境/用户验收 **NOT_RUN**，生产 **NOT_READY**。

## 2. Design Basis 与高风险边界

采用 **Independent Architecture**：本轮需要准确收缩源 Role/机器策略的三个 Deployment 目标，并退出共享工具的隐式旧 API 默认合同，涉及声明、执行器和安全策略的组合。原发布退休架构提供产品依据，但没有直接批准这些具体源对象/默认值的删除，故先补本边界设计与独立检查，不在实现中临时决定。

冻结候选预计 **29 个准确实施文件**：7 删除源、8 混合源、7 原测试文件、7 设计/索引/顶注文件。接单的“预计30+”是调查估计，本矩阵校准为29；不为了计数拆批或重开全局架构。此设计不新增当前权限、领域合同、状态机、持久化事实、事务/retry/UNKNOWN/recovery 或 provider 协议。若实施必须改变这些当前边界，停止相应部分，按 AGENTS 的真实 BLOCKER 规则重新准入。

Threat Model 限于旧对象被误创建、隐式旧目标被继续使用、错误扩权、误删当前共享 owner、安全断言因旧输入退出而被弱化，以及源码删除被误写成真实环境退出。既有历史 run/re-run、外部 IAM 或集群权限不在本次调查中，不新增其风险接受或生产保证。

## 3. 实际消费者与闭包根因

| 对象 | main 基线静态事实 | 结论 |
| --- | --- | --- |
| 旧 API/UI workflow 与 PS wrapper | 两 workflow 已固定为空权限/单失败 step，PS wrapper 直接 throw；没有旧 normal/rollback publisher。旧 API Deployment/Service 却仍被 base/prod 聚合 | 退休其独占声明与引用；保留三个已关闭入口，绝不恢复旧 publisher |
| `jobs/listingkit-schema-migrate-job.yaml` | 传 `-scope all`；当前 `listingkitschemamigrate.validateScope` 在配置/DB I/O 前拒绝 `all`，仅 current `commercial/workbench` 正常 | 删除旧发布 Job；保留 CLI 及全部当前初始化/取消/权限合同 |
| `jobs/pod-image-index-backfill-job.yaml` | 执行 `/app/listingkit-shein-pod-image-index-backfill`，tracked command 已不存在 | 删除失效 Job，不恢复回填器或访问旧表 |
| `jobs/product-listing-api-schema-migrate-job.yaml`、两个 schema runner | 仅服务已退旧 API 发布链；Job/runner 包装不是正常 schema CLI 的事实 owner。原 Go/shell 检查和历史文档是剩余消费者 | 同批退休包装；不能据此删除当前 schema 命令/库、授权、grant 或持久化测试 |
| `listingkit-schema-migrate-job.sh` 与专属 shell test | 通用参数表面下，实际仅维护上述旧发布 Job/runner 合同；无当前 workflow/生产脚本调用 | 两源退休，无新 runner 或替代 Job |
| `listingkit-apply-api-deployment.sh` | UI wrapper 显式传 `listingkit-ui`，ImageAgent wrapper 要求显式 container；共享执行器仍正确，但默认 container 仍为 `product-listing-api` | 保留执行器和两个 wrapper；仅退出隐式旧目标，要求显式 `--container` |
| ConfigMap / Ingress | `LISTINGKIT_API_BASE`、`LISTINGKIT_SERVICE_API_BASE` 指向旧 Service；base/prod patch 仍把旧 `/api/v1/listing-kits/integrations/zitadel/sms` 路由到该 Service | 只删两个该 ConfigMap 中的数据项与旧 SMS path；不删除全局同名配置合同，不补新生产 URL/route |
| 共享镜像、UI、worker | ImageAgent v2/v3 仍使用 `task-processor-product-listing-api` 镜像，UI/worker 与共享配置/凭据仍有独立职责 | 保留 prod `images` 映射、UI/worker 声明和共享基础层，不能按名称整删 |
| canary / drain / attestation | 当前 worker CLI 支持 `-canary`/v2/v3；drain 查询的 `image_agent_v2_runs` 仍由 `internal/imageagent/store` 拥有。原 runner、OIDC、不可变镜像及精确 SHA/attempt 检查仍保护当前/历史安全合同 | 保留 canary、共享 gate executor、drain、只读 attestation verifier 及有效负例，不以无旧 publisher 为由全删 |

这是旧发布关闭后，物理声明与默认目标尚未退出的源码组合问题，不是 current-application 缺新发布平台。所有调查限 tracked 源；没有执行 kubectl、Docker/Compose、SQL、Temporal、HTTP/provider、attestation 或旧/当前启动器。

## 4. 所有权、持久化与平台副作用

| 边界 | Owner 与本轮行为 |
| --- | --- |
| 当前领域事实、正常初始化 | 原 current-application / 各 domain / 原 schema CLI 与 installer；不改表、scope 语义、schema/grant、事务、备份恢复或启动协议 |
| ImageAgent runs、slots、effects、Temporal history | 原 ImageAgent/store/Temporal owner；`image_agent_v2_runs` 名称不代表旧 Task 表。v2/v3 worker、canary、恢复、staging/retention、queue/timeout/UNKNOWN 全保留 |
| SHEIN 数字账户/Profile/DB/Redis/Store | 独立登录 worker 及其原 service/repository/Store 同步；本批源与测试零改，无映射、迁移或第二 owner |
| UI、身份与凭据 | 当前 Console/Auth.js、native identity、token handoff 和私密配置 owner；web、原生 CLI/库、issue357 零改，UI 声明与 auth ConfigMap/Secret 隔离保持 |
| Kubernetes 对象/namespace/Role/Secret | 本轮只修改 Git 源。实际集群对象及既有权限仍由原运行 owner 负责；不 apply/delete/scale/patch/get，不读取私有文件或 live credentials |
| 源授权库存 | 仅从 API Role 和机器策略删除三项退役 Deployment 名称；其它 verbs/resources/names、issuer/subject/audience/workflowRef、API/UI 身份区分不变。源收缩不等于 live 撤权 |

没有新增可重试副作用或持久化步骤：失败/取消/并发/重启沿当前 owner 合同处理。源码删到一半或测试失败由同一分支修复，不允许带半套声明、旁路或虚假成功交付。不运行旧恢复/回填来“验证删除”；未来若出现具体当前外部契约或 durable asset 的强制兼容证据，停相应部分并报告用户，不自行发明 compatibility exception。

剩余 Kubernetes 聚合源保留 UI、图像、独立 worker 等已存在对象，**不构成可发布 current-application 的完整新生产包**；不把旧 URL 改指私密 manifest，不声明生产或实例已切换。当前试用仍沿其既有入口，不受本维护准入影响。

## 5. 删除 / 保留 / 抽取矩阵

### 5.1 七个准确 tracked 删除源

```text
deployments/kubernetes/listingkit-workbench/base/product-listing-api-deployment.yaml
deployments/kubernetes/listingkit-workbench/base/product-listing-api-service.yaml
deployments/kubernetes/listingkit-workbench/jobs/listingkit-schema-migrate-job.yaml
deployments/kubernetes/listingkit-workbench/jobs/product-listing-api-schema-migrate-job.yaml
deployments/kubernetes/listingkit-workbench/jobs/pod-image-index-backfill-job.yaml
scripts/listingkit-schema-migrate-job.sh
scripts/tests/listingkit-schema-migrate-job-test.sh
```

只删准确源，不递归删目录、实际 Job/Pod/Deployment、SQL/table/row、卷、镜像或历史记录。正常 schema CLI 不在列表内。

### 5.2 八个混合源的准确编辑

以下 Kubernetes 路径均位于 `deployments/kubernetes/listingkit-workbench/`。

| 文件 | 仅允许的变化 |
| --- | --- |
| `base/kustomization.yaml` | 只移除旧 API Deployment/Service 两个 resource；其它 UI、ImageAgent、SHEIN、imgproxy、namespace/config 保持 |
| `base/ingress.yaml`、`overlays/prod/patch-ingress.yaml` | 各只移除指向旧 API Service 的旧 SMS path；保留 UI、`/img`、TLS/host/注解，不添加当前 SMS 替代 route |
| `base/configmap.yaml` | 仅删除指向旧 Service 的 `LISTINGKIT_API_BASE` 与 `LISTINGKIT_SERVICE_API_BASE` 两项；其它值原样。同名当前 env/字段消费者不修改 |
| `release-authority/listingkit-release-gate-runners.yaml` | 仅删除 `product-listing-api-schema-migrate-runner` 与 `listingkit-schema-migrate-runner` 两个完整 document；保留 canary document 逐字原样，不删除整个 manifest |
| `release-authority/listingkit-api-release-role.yaml` | 仅从 Deployment resourceNames 删除旧 API 和两个 schema runner；保留两长期 worker/canary 与其它全部权限，不增加 Pod、create、list 或 wildcard 权限 |
| `release-authority/release-policy.yaml` | 仅同步上行三项 `protectedResources` 名称及 `releaseGateRunners.names` 中两个 schema runner；其它策略、readonly attestation/身份常量和有效安全合同保持 |
| `scripts/listingkit-apply-api-deployment.sh` | 退出 `CONTAINER=product-listing-api` 隐式默认；无显式 `--container` 在任何 kubectl mutation 前失败。保持原参数/不可变镜像、DNS、envFrom/临时文件/退出语义，不新增执行器或目标 fallback |

### 5.3 当前保留边界（源/合同不因旧名称退出）

- 两套 CI `.github/workflows/ci.yml`、`commercial-readiness.yml` 以及其它全部 workflow 零改，含两个固定拒绝发布入口和 SHEIN 发布 workflow；PS 拒绝 wrapper 原样。
- 当前 UI/全部 web 源与测试、UI Deployment/Service/auth ConfigMap/network policy、当前身份/原生 CLI/库/token handoff、issue357 全部保持。base/prod 的共享镜像映射、所有 namespace/共享 Secret/example、ZITADEL 与 Account Compose 保留；不读取私密 manifest。
- ImageAgent v2/v3 两长期 worker、当前 canary document、共享 `listingkit-run-release-gate-deployment.sh`、`listingkit-image-agent-v2-drain-check.sh`、`verify-listingkit-api-release-attestation.sh` 及对应有效测试保留。当前 worker/schema command、Dockerfile/build、drain 的 inventory/recovery horizon 不变。
- `listingkit-apply-ui-deployment.sh`、`listingkit-apply-image-agent-worker-deployment.sh`、immutable-image library 及测试保留。前两者已显式给 container，继续原 owner；不复制/改名建立第二套发布工具。
- OIDC configurator/refresh、API/UI RoleBindings、AuthenticationConfiguration、UI Role、Rego 文件及全部 negative fixtures、policy verifier、原 canary-order guard原样；后者继续检查拒绝入口和真实保留 canary/production queue 隔离。negative fixture 中的旧 runner 字符串是禁止输入，不是新的执行消费者。
- 正常 schema CLI/库/权限/取消、当前平台/定价/规则/AI/图像/采集/workerpool/queue/Retry、通用 DB/port-forward、Store、独立 SHEIN 登录/同步与此前全部能力原样；inventory/历史计数/ceilings/depguard/依赖/锁/保护零改。

**无新增抽取代码。** 有效通用行为已经在当前执行器、策略、schema 与 ImageAgent owner，不搬移 helper 或为了清理创造消费者。

```text
Legacy decision: RETIRE
Reusable behavior: 复用现有显式 UI/worker apply、ImageAgent canary/drain、schema、OIDC/RBAC/immutable/attestation owner；无新增抽取
Current owner: 当前 Console/身份/domain/schema、ImageAgent/store/Temporal、原发布安全工具及独立 SHEIN/Store owner
Cutover/deletion condition: 本设计独立检查与用户确认、IMPLEMENTATION_READY及双Issue Ready读回后，同PR完成七源/八混合源及全部准确尾部；原CI和一次最终独立交付检查，合并另行准确HEAD授权
```

## 6. 原 guard 与有效安全检查的处置

只适配下列七个原测试文件，不新建 runner 或验收平台。旧目标的实现细节测试不能成为恢复源的理由；有效安全断言不得删、skip 或因缺文件提前成功。

| 原测试文件 | 精确处置 / 保留不变量 |
| --- | --- |
| `tests/listingkit_backend_retirement_test.go` | 在既有退休 guard 增补七源缺席及旧接线退出，先因源仍在 RED，再删除后 GREEN；所有此前退休/当前构建/HTTP/auth/worker断言保留 |
| `scripts/tests/listingkit-apply-api-deployment-test.sh` | 旧 API fixture/隐式目标正例改为现有 worker 的显式 container，新增缺 container 时非零且 fake kubectl 未调用的 RED→GREEN；保留原不可变镜像、UI envFrom、patch/apply 错误、参数/临时文件安全场景 |
| `tests/listingkit_deploy_workflow_test.go` | 只退休两个已取消 schema Job 的 deadline-wrapper 断言；保留原 worker/canary capability 隔离及当前镜像 routing label 检查 |
| `tests/listingkit_secret_boundary_test.go` | 只退出旧 API/已删除 Job 的正向绑定与专属 helper；保留 UI/SHEIN/imgproxy 的完整最小 Secret allowlist、禁用 envFrom Secret 与禁止身份高权 token 断言 |
| `tests/commercial_readiness_workflow_test.go` | 仅退休旧 API readiness、旧 Job image/legacy-scope 正例；旧 API invitation/SMS 正向绑定退出，改为旧入口/路由缺席并保留所有当前 workload 的身份/SMS token 隔离、canonical Secret examples。ImageAgent scopes、当前 Docker/worker/schema/build/商业门禁与 workflow 断言保持 |
| `tests/listingkit_release_gate_runner_test.go` | inventory 只保留 canary；原 fixture target/相同 invocation 字符串换成真实保留 canary。canonical/current-invocation、所有 command/args/env/envFrom/volume/mount/SA/security/resources/image/sidecar/Pod拒绝等正负例及断言完整保留，不删除整个测试文件 |
| `tests/listingkit_release_authority_policy_test.go` | 仅同步 runner 精确库存为单一 canary，并核对三个退役 Deployment 不再获源Role权限。OIDC claims/ref/audience、无长效fallback、create-free bootstrap、native sidecar、失败入口旁路及所有安全负例保持 |

执行器的显式目标规则仅退出已取消的默认值，不改变当前 caller 的身份或授权；所有当前 wrapper/owner 已显式给值。source Role/机器策略必须三名称同时减除且其它字段逐字保持，原 Rego + positive bundle/negative fixtures证明没有扩权和输入缺失；不能放宽策略、删除 verifier/CI step、改选择条件或伪造缺失输入。

实施自检限原 guard 的 RED→GREEN、受影响原 Go 测试、两个已存在 fake-command fixture 族（apply、gate）及原发布规则检查；不得执行真实 kubectl/SQL/Temporal/发布动作。完整检查留稳定候选的原 PR CI。假命令必须优先于主机 PATH，失败不允许落到真实工具；不新建故障注入、人工验收或验证工具。保留未变代码的已有有效证据，修复只查相关增量。

### 6.1 直接说明、旧保留结论与设计阶段文件

| 准确文档 | 确认后处置 |
| --- | --- |
| 本文 | 仅阶段说明反映已确认/实施；冻结矩阵保持，滚动 HEAD/CI/review留PR |
| `docs/architecture/README.md` | 索引本文与状态，不改其它 owner |
| `docs/refactoring/legacy-register.md`、`module-target-mapping.md` | 顶注仅替代本批七删除源/八混合源的旧保留含义；原矩阵、历史行与计数不重写 |
| `docs/refactoring/code-health-decisions.md` | 仅顶注取代旧 schema Job shell-test 的 retained-generated-contract 当前含义及已消失回填包装的当前可执行含义；保留历史 rows/counts/failures、通用 helper 与安全测试结论 |
| `deployments/kubernetes/listingkit-workbench/README.md` | 仅窄历史顶注：准确源删除、旧 API URL/SMS 接线退出；历史操作/SQL/drain/rollback正文保留为证据，不运行、不作为当前试用前置。保留当前 UI/worker/OIDC/RBAC/schema 与持久化 owner 边界 |
| `docs/architecture/release-identity-preflight-retirement.md` | 仅追加窄 superseded 顶注，取代其两个 schema runner/旧发布Job源的保留含义；原冻结设计正文与评审历史逐字保留。正常schema CLI、ImageAgent canary/恢复、安全规则及失败发布入口不受取代 |

不修改其它 dated spec/plan、历史验证报告、商业/生产/恢复政策或 issue357。旧源码路径/操作样本在历史正文及禁止输入中保留，不据此恢复源。全批29文件；本设计阶段只改本文/四索引共5文档，其它24准确文件确认前零改。

## 7. 独立检查、准入、交付与权限

1. 当前只交设计：五文档、原 Greenfield/ArchitectureReadme guard、相对链接与 diff 检查；禁止运行相关 shell/Pester、kubectl（含dry-run/kustomize）、Docker/Compose、token/attestation/drain helper、SQL/Temporal/provider或真实状态探测。
2. 在同一主要 PR 做适用独立设计检查：核对实际消费者、29文件矩阵、七源/八混合源、source Role三名称收缩、当前UI/worker/schema及全部有效安全断言；不重新审查全局架构。同SHA Running不重复触发，不默认增加实现者或AI互验线程。
3. Finding 先分类；BLOCKER须明确 AGENTS 的具体后果、当前Must与阻塞层级。最多两轮正常设计检查；非Blocker hardening不变成试用前置或新平台。没有新增风险接受。
4. 独立检查达到设计准入、用户确认后，在#29/#301写明冻结设计 **IMPLEMENTATION_READY** 与范围，分别读回Ready才改正式源/测试。同一Writer/分支/PR整批实施；稳定候选前核对实际main，只处理真实组合变化。
5. 完整闭包形成后一次最终独立交付检查真实diff、保留路径/断言和现有证据，范围内修复只复核增量；原CI与检查不等于用户运行验收，不签发可上线或生产接受。

交付结果为“旧 API/Task 发布源与误导接线退出，现有当前能力继续沿原入口”，不新增部署/验收步骤。用户另行授权最终准确 HEAD 后正常 expected-head 合并，不强推/管理员绕过/改保护；然后核实实际 merge SHA 的 event=push/branch=main/completed/success CI，source PR CI不得冒充main。

本次授权仅设计及必要文档检查/提交/推送/PR维护。正式实施、合并、部署、现有实例/业务数据/凭据/IAM/已有报告操作、Issue关闭与其它批次均未授权；#29/#301保持OPEN，生产NOT_READY。
