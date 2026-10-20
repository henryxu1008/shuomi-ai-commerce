# 旧 Kubernetes API/Task 发布执行与 UI 专属接线：完整退休闭包

> **Narrow source supersession — #850 / #29 / #301.** The [confirmed principal closure](legacy-kubernetes-release-principal-retirement-boundary.md) supersedes only §5.3's retained old API/UI principal/Role/Binding/authentication/OIDC-helper sources and their obsolete machine-policy inputs. Current canary, valid static safety, shared tools and all current owners remain. The original 42-file matrix and historical body below are unchanged; no live IAM/resource/data operation, recovery-owner change or new trial prerequisite is implied.

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

> Status: **用户确认设计 / IMPLEMENTATION_READY；本批源码已实施，In Review，待独立交付检查。** Design Basis: **Independent Architecture**。冻结设计 `a73f441442b8593d88f52fd776fbbb1b8123d733`，调查与开工 main `d255feb7f1349108a466530353d2637080a911be`（2026-10-20）；双Issue在正式源修改前已写回并读回Ready。§5/§6/§6.1/§7为本次冻结边界，准确实现HEAD、CI、复核与权限在同一主要PR；无合并、部署或真实环境操作授权。

## 1. 产品结果、Authority 与阶段

使用者是维护当前应用与内部试用的开发、运维人员。结果是退出已经无法交付当前应用的旧 API workload、旧发布专属 schema/回填 Job 及其调用尾部，避免操作者仍从源码创建旧对象或沿隐式旧 API 目标操作。此维护没有正在阻塞的内部试用核心操作，不作为试用新前置；用户明确选择先设计完整闭包，确认后整批实施。

一个 Delivery Batch / 一个主要 PR，统一处理声明、聚合接线、共享执行器的旧默认值、源 RBAC 库存、原测试与准确说明。不按文件数、脚本/声明/测试或模块机械拆批。当前 UI、ImageAgent、正常 schema CLI、通用工具和既有能力继续归原 owner。

Authority：[Greenfield 产品基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Project boundaries](project-boundaries.md)、[Issue 规则](../engineering/issue-driven-development.md)、已确认的[旧发布身份退休设计](release-identity-preflight-retirement.md)与[#846 本地验收退休边界](legacy-local-acceptance-closure-retirement-boundary.md)。当前内部试用沿 [Account Compose](../../deployments/docker/account-compose/README.md) 与既有私密 manifest；[RUN-1](../operations/current-application-run1.md) 仅为独立验收环境，不替代试用或生产交付。没有 UI/IA/交互修改，Figma N/A；全部 web 源与测试零改。

Must：完整删除 §5.1 专属源与 §5.2 接线；保留 §5.3 当前能力及所有仍有效安全断言；两套 CI workflow 和其它 workflow、inventory、历史计数、ceilings、依赖/锁/保护零改；不操作实例、数据、凭据或既有报告，不增加试用前置。Should：复用现有 Bash/Go guard、Conftest/OPA 与当前 owner，不新建平台或抽象。

Out of Scope：当前应用/身份/领域行为、生产发布替代流程、真实部署/撤权/迁移/清理、旧数据兼容、当前 ImageAgent 恢复协议或数据库变化、issue357 全部源/文档/测试、其它 owner 的 #692。没有新增 Accepted Risk；历史实例、卷、IAM、Job、Temporal histories 与 durable assets 的实际状态和归属均 **UNKNOWN**，不得推断不存在或可清理。真实环境/用户验收 **NOT_RUN**，生产 **NOT_READY**。

## 2. Design Basis 与高风险边界

采用 **Independent Architecture**：本轮需要准确收缩源 Role/机器策略的三个 Deployment 目标及一项邀请Secret读取权限，并退出共享工具的隐式旧 API 默认合同，涉及声明、执行器和安全策略的组合。原发布退休架构提供产品依据，但没有直接批准这些具体源对象/默认值的删除，故先补本边界设计与独立检查，不在实现中临时决定。

冻结候选预计 **42 个准确实施文件**：15 删除源、12 混合源、8 原测试文件、7 设计/索引/顶注文件。旧 API producer 的 drain、receipt、邀请凭据预检和机器策略消费者一起退出，不把声明和其专属身份尾部拆批。规模超过风险提示线，仍保持一个主要 PR，只检查本轮边界。此设计不新增当前权限、领域合同、状态机、持久化事实、事务/retry/UNKNOWN/recovery 或 provider 协议。若实施必须改变这些当前边界，停止相应部分，按 AGENTS 的真实 BLOCKER 规则重新准入。

Threat Model 限于旧对象被误创建、隐式旧目标被继续使用、错误扩权、误删当前共享 owner、安全断言因旧输入退出而被弱化，以及源码删除被误写成真实环境退出。既有历史 run/re-run、外部 IAM 或集群权限不在本次调查中，不新增其风险接受或生产保证。

## 3. 实际消费者与闭包根因

| 对象 | main 基线静态事实 | 结论 |
| --- | --- | --- |
| 旧 API/UI workflow 与 PS wrapper | 两 workflow 已固定为空权限/单失败 step，PS wrapper 直接 throw；没有旧 normal/rollback publisher。旧 API Deployment/Service 却仍被 base/prod 聚合 | 退休其独占声明与引用；保留三个已关闭入口，绝不恢复旧 publisher |
| `jobs/listingkit-schema-migrate-job.yaml` | 传 `-scope all`；当前 `listingkitschemamigrate.validateMigrationScope` 在配置/DB I/O 前拒绝 `all`，仅 current `commercial/workbench` 正常 | 删除旧发布 Job；保留 CLI 及全部当前初始化/取消/权限合同 |
| `jobs/pod-image-index-backfill-job.yaml` | 执行 `/app/listingkit-shein-pod-image-index-backfill`，tracked command 已不存在 | 删除失效 Job，不恢复回填器或访问旧表 |
| `jobs/product-listing-api-schema-migrate-job.yaml`、两个 schema runner | 仅服务已退旧 API 发布链；Job/runner 包装不是正常 schema CLI 的事实 owner。原 Go/shell 检查和历史文档是剩余消费者 | 同批退休包装；不能据此删除当前 schema 命令/库、授权、grant 或持久化测试 |
| `listingkit-schema-migrate-job.sh` 与专属 shell test | 通用参数表面下，实际仅维护上述旧发布 Job/runner 合同；无当前 workflow/生产脚本调用 | 两源退休，无新 runner 或替代 Job |
| `listingkit-apply-api-deployment.sh` | UI wrapper 显式传 `listingkit-ui`，ImageAgent wrapper 要求显式 container；共享执行器仍正确，但默认 container 仍为 `product-listing-api` | 保留执行器和两个 wrapper；仅退出隐式旧目标，要求显式 `--container` |
| ConfigMap / Ingress | `LISTINGKIT_API_BASE`、`LISTINGKIT_SERVICE_API_BASE` 指向旧 Service；base/prod patch 仍把旧 `/api/v1/listing-kits/integrations/zitadel/sms` 路由到该 Service | 只删两个该 ConfigMap 中的数据项与旧 SMS path；不删除全局同名配置合同，不补新生产 URL/route |
| 共享镜像、UI、worker | ImageAgent v2/v3 仍使用 `task-processor-product-listing-api` 镜像，UI/worker 与共享配置/凭据仍有独立职责 | 保留 prod `images` 映射、UI/worker 声明和共享基础层，不能按名称整删 |
| ImageAgent canary / 恢复 owner | 当前 worker CLI 支持 `-canary`/v2/v3；`image_agent_v2_runs` 仍由 `internal/imageagent/store` 拥有，Go worker/store/Temporal 恢复不依赖旧发布 shell | 保留 worker/canary、共享 gate executor 和当前恢复/持久化合同，不删表、history、队列或相关有效测试 |
| 旧 producer-bound drain / attestation | drain 无条件查询 `product-listing-api` Deployment/Pods 的旧 release identity；attestation verifier 只接受旧 API workflow/receipt。非测试消费者仅旧发布历史指引，无法代表 current-application 的恢复入口 | 两脚本及专属测试/身份字段同批 RETIRE；不换一个 Deployment 名称来伪造当前恢复/发布协议 |
| 旧机器策略执行元数据 | `releaseIdentity` 指向旧 API；`producerRouting/releaseAttestation`、job/step/group/action 清单描述已失败关闭的旧 publisher；Rego/Go 仍强制这些旧形状 | 同批退出旧形状，只保留真实的固定拒绝入口、OIDC/RBAC、UI ownership、不可变镜像及 canary 安全合同 |
| 旧邀请凭据预检 | 旧 API 声明与 credential-preflight 步骤退出后，`validate-listingkit-invitation-secret.sh` 无剩余非文档消费者；API Role/机器策略仍给旧publisher读取高权限专用Secret | 预检/专属测试及该一项源get权限同批退休；三个仍有当前消费者的测试utility先原样抽取到已有安全测试owner，旧publisher专属邀请/SMS K8s Secret examples同时退出；当前身份/私密凭据/其它共享模板不变 |

这是旧发布关闭后，物理声明与默认目标尚未退出的源码组合问题，不是 current-application 缺新发布平台。所有调查限 tracked 源；没有执行 kubectl、Docker/Compose、SQL、Temporal、HTTP/provider、attestation 或旧/当前启动器。

## 4. 所有权、持久化与平台副作用

| 边界 | Owner 与本轮行为 |
| --- | --- |
| 当前领域事实、正常初始化 | 原 current-application / 各 domain / 原 schema CLI 与 installer；不改表、scope 语义、schema/grant、事务、备份恢复或启动协议 |
| ImageAgent runs、slots、effects、Temporal history | 原 ImageAgent/store/Temporal owner；`image_agent_v2_runs` 名称不代表旧 Task 表。v2/v3 worker、canary、恢复、staging/retention、queue/timeout/UNKNOWN 全保留 |
| SHEIN 数字账户/Profile/DB/Redis/Store | 独立登录 worker 及其原 service/repository/Store 同步；本批源与测试零改，无映射、迁移或第二 owner |
| UI、身份与凭据 | 当前 Console/Auth.js、native identity、token handoff 和私密配置 owner；web、原生 CLI/库、issue357 零改，UI 声明与 auth ConfigMap/Secret 隔离保持 |
| Kubernetes 对象/namespace/Role/Secret | 本轮只修改 Git 源。实际集群对象及既有权限仍由原运行 owner 负责；不 apply/delete/scale/patch/get，不读取私有文件或 live credentials |
| 源授权库存 | RBAC 变化仅从 API Role/机器策略删除三项退役 Deployment 名称及对 `listingkit-member-invitation-secret` 的专属get项；其它 verbs/resources/names、issuer/subject/audience/workflowRef、API/UI 身份区分不变。旧执行元数据取消另见 §5.2，不增加新身份或权限；源收缩不等于 live 撤权 |

没有新增可重试副作用或持久化步骤：失败/取消/并发/重启沿当前 owner 合同处理。源码删到一半或测试失败由同一分支修复，不允许带半套声明、旁路或虚假成功交付。不运行旧恢复/回填来“验证删除”；未来若出现具体当前外部契约或 durable asset 的强制兼容证据，停相应部分并报告用户，不自行发明 compatibility exception。

剩余 Kubernetes 聚合源保留 UI、图像、独立 worker 等已存在对象，**不构成可发布 current-application 的完整新生产包**；不把旧 URL 改指私密 manifest，不声明生产或实例已切换。当前试用仍沿其既有入口，不受本维护准入影响。

**当前恢复与旧发布 drain 分开定责**：当前恢复继续由原 ImageAgent Go worker/store/Temporal 实现，不承诺保留一个能够对新应用签发 drain/可停结论的 shell 入口。被删除的校验器没有新的当前 producer 身份可消费；不新建替代工具，也不把旧 API 名称换成 current-application。历史 inventory/retention/rollback 文本仅作证据，不能据此停止任何仍可能拥有 histories/assets 的 worker。未来物理停机/退役需要原运行 owner 的独立授权与适用方案，不是本批或内部试用前置。

## 5. 删除 / 保留 / 抽取矩阵

### 5.1 十五个准确 tracked 删除源

```text
deployments/kubernetes/listingkit-workbench/base/product-listing-api-deployment.yaml
deployments/kubernetes/listingkit-workbench/base/product-listing-api-service.yaml
deployments/kubernetes/listingkit-workbench/jobs/listingkit-schema-migrate-job.yaml
deployments/kubernetes/listingkit-workbench/jobs/product-listing-api-schema-migrate-job.yaml
deployments/kubernetes/listingkit-workbench/jobs/pod-image-index-backfill-job.yaml
scripts/listingkit-schema-migrate-job.sh
scripts/tests/listingkit-schema-migrate-job-test.sh
scripts/listingkit-image-agent-v2-drain-check.sh
scripts/verify-listingkit-api-release-attestation.sh
tests/listingkit_v2_drain_script_test.go
policy/listingkit-release-authority/fixtures/negative/release-identity-owner-drift.yaml
scripts/validate-listingkit-invitation-secret.sh
tests/listingkit_invitation_secret_preflight_test.go
deployments/kubernetes/listingkit-workbench/base/member-invitation-secret.example.yaml
deployments/kubernetes/listingkit-workbench/base/tencent-sms-secret.example.yaml
```

只删准确源；Secret example源退出不授权删除任何实际Secret。不递归删目录、实际 Job/Pod/Deployment、SQL/table/row、卷、镜像或历史记录。正常 schema CLI 不在列表内。

### 5.2 十二个混合源的准确编辑

以下 Kubernetes 路径均位于 `deployments/kubernetes/listingkit-workbench/`。

| 文件 | 仅允许的变化 |
| --- | --- |
| `base/kustomization.yaml` | 只移除旧 API Deployment/Service 两个 resource；其它 UI、ImageAgent、SHEIN、imgproxy、namespace/config 保持 |
| `base/ingress.yaml`、`overlays/prod/patch-ingress.yaml` | 各只移除指向旧 API Service 的旧 SMS path；保留 UI、`/img`、TLS/host/注解，不添加当前 SMS 替代 route |
| `base/configmap.yaml` | 仅删除指向旧 Service 的 `LISTINGKIT_API_BASE` 与 `LISTINGKIT_SERVICE_API_BASE` 两项；其它值原样。同名当前 env/字段消费者不修改 |
| `base/secret.example.yaml` | 仅校准指向旧 API、release-scoped identity preflight、旧 migration Job 的消费者说明及专属邀请Secret provisioning指引，改为当前UI/ImageAgent/SHEIN的显式allowlist边界；保留所有键和值、读写token隔离及禁止向UI/browser/imgproxy泄露高权token的不变量。不新增凭据、mount、权限或当前身份流程 |
| `release-authority/listingkit-release-gate-runners.yaml` | 仅删除 `product-listing-api-schema-migrate-runner` 与 `listingkit-schema-migrate-runner` 两个完整 document；保留 canary document 逐字原样，不删除整个 manifest |
| `release-authority/listingkit-api-release-role.yaml` | 从 Deployment resourceNames 删除旧 API 和两个 schema runner；删除仅授予 `listingkit-member-invitation-secret` get 的整条Rule。保留两长期 worker/canary 与其它全部权限，不增加 Pod、create、list 或 wildcard 权限 |
| `release-authority/release-policy.yaml` | 同步上行三项 `protectedResources` Deployment名称、一项邀请Secret/get及两个 schema runner；删除旧 `producerRouting`、`releaseAttestation`、`releaseIdentity`，及 API/UI identity 下仅供旧执行协议的 `job/oidcTokenStep/kubeconfigStep/protectedGroups/allowedSteps/allowedActions`。保留 issuer/usernamePrefix、environment/audience/subject/workflowRef/role、工具 pin、UI ownership、其余RBAC/文档与当前 canary 合同，不制造新发布身份 |
| `scripts/listingkit-apply-api-deployment.sh` | 退出 `CONTAINER=product-listing-api` 隐式默认；无显式 `--container` 在任何 kubectl mutation 前失败。保持原参数/不可变镜像、DNS、envFrom/临时文件/退出语义，不新增执行器或目标 fallback |
| `policy/listingkit-release-authority/release_authority.rego` | 只退休旧 producerRouting/releaseIdentity 形状、动态旧 oidcTokenStep 和 protectedGroups 两条专属断言；真实 workflow 遍历、固定失败完整形状、OIDC/Role/Binding/claims、工具 pin、UI scopes、canary/sidecar/Pod拒绝和不可变 action 规则保持。禁止靠 undefined 字段或跳过 bundle 输入让原检查静默成功 |
| `policy/listingkit-release-authority/negative_fixtures.rego` | 只删除旧 `releaseIdentity.owner` 专属规则；其它全部共享安全规则原样 |
| `policy/listingkit-release-authority/fixtures/negative/mutable-action-ref.yaml` | 只移除不再消费的旧 `releaseIdentity` 辅助字段；不可变 action 负例本身与拒绝结果保持 |

### 5.3 当前保留边界（源/合同不因旧名称退出）

- 两套 CI `.github/workflows/ci.yml`、`commercial-readiness.yml` 以及其它全部 workflow 零改，含两个固定拒绝发布入口和 SHEIN 发布 workflow；PS 拒绝 wrapper 原样。
- 当前 UI/全部 web 源与测试、UI Deployment/Service/auth ConfigMap/network policy、当前身份/原生 CLI/库/token handoff、issue357 全部保持。base/prod 的共享镜像映射、所有 namespace/共享 Secret及除§5.1外的其它examples、ZITADEL 与 Account Compose 保留；共享Secret example仅按§5.2校准旧消费者说明，所有键和值原样，不读取私密 manifest。
- ImageAgent v2/v3 两长期 worker、当前 canary document、共享 `listingkit-run-release-gate-deployment.sh` 及全部有效 canary/执行器负例保留。当前 worker/store/Temporal 的恢复、history/staging/retention、schema command 与 Dockerfile/build 原样；当前镜像的 routing label 及其原测试不因旧机器策略字段退出而删除。旧发布 drain/receipt 不冒充这些当前合同的 owner。
- 当前恢复已有直接证据 owner：`internal/imageagent/temporal/history_replay_test.go`、`generation_recovery_test.go`、`effect_recovery_workflow_test.go` 与 `internal/imageagent/store/generation_output_recovery_test.go` 及其实现全部零改。无需用旧发布 shell 包装这些 owner，也不新增物理 drain/停机入口。
- `listingkit-apply-ui-deployment.sh`、`listingkit-apply-image-agent-worker-deployment.sh`、immutable-image library 及测试保留。前两者已显式给 container，继续原 owner；不复制/改名建立第二套发布工具。
- OIDC configurator/refresh、API/UI RoleBindings、AuthenticationConfiguration、UI Role、policy verifier 与原 canary-order guard原样；后者继续检查拒绝入口和真实保留 canary/production queue 隔离。除 §5.1/§5.2 明列的一个取消 producer-owner 负例/规则和一个辅助字段，其余共享 negative fixtures/规则全部保持；禁止输入中的旧 runner 字符串不算执行消费者。固定失败 workflow 的真实整形检查不弱化，不恢复旧动态步骤模型。
- 正常 schema CLI/库/权限/取消、当前平台/定价/规则/AI/图像/采集/workerpool/queue/Retry、通用 DB/port-forward、Store、独立 SHEIN 登录/同步与此前全部能力原样；inventory/历史计数/ceilings/depguard/依赖/锁/保护零改。

**无生产行为抽取；仅原样抽取三个现有测试utility。** `preflightBash`、`writeBashPreflightFake`、`writeExecutablePreflightFake` 仍被当前 OIDC/canary executor 测试使用，先原样移入已有 `tests/listingkit_release_authority_policy_test.go` 并保留所需标准库import，再删旧邀请预检测试文件。仅旧drain消费的 `writePythonPreflightFake` 和专属预检scenario随旧测试退休。不得新建测试framework或另开helper平台；有效生产通用行为已归当前执行器/策略/schema/ImageAgent owner，不创造新消费者。

```text
Legacy decision: RETIRE；EXTRACT 仅限三个仍有消费者的原测试utility
Reusable behavior: 复用现有显式 UI/worker apply、ImageAgent canary/Go恢复、schema、OIDC/RBAC/immutable 与当前 invocation owner；测试utility原样移至既有安全测试owner
Current owner: 当前 Console/身份/domain/schema、ImageAgent/store/Temporal、原发布安全工具及独立 SHEIN/Store owner
Cutover/deletion condition: 本设计独立检查与用户确认、IMPLEMENTATION_READY及双Issue Ready读回后，同PR先抽取测试utility，再完成十五删除源/十二混合源及全部准确尾部；原CI和一次最终独立交付检查，合并另行准确HEAD授权
```

## 6. 原 guard 与有效安全检查的处置

除 §5.1 旧 producer-bound drain/邀请预检两个专属测试文件整退，只适配下列八个原测试文件，不新建 runner 或验收平台。先原样移出三项共享utility再删除其旧测试owner，不能让当前测试失去helper。旧目标的实现细节测试不能成为恢复源的理由；有效安全断言不得删、skip 或因缺文件提前成功。

| 原测试文件 | 精确处置 / 保留不变量 |
| --- | --- |
| `tests/listingkit_backend_retirement_test.go` | 在既有退休 guard 增补十五源缺席及旧接线/producer字段/专属Secret权限退出，先因源仍在 RED，再删除后 GREEN；所有此前退休/当前构建/HTTP/auth/worker断言保留，不全局禁用仍合法的镜像/字段名称 |
| `scripts/tests/listingkit-apply-api-deployment-test.sh` | 旧 API fixture/隐式目标正例改为现有 worker 的显式 container，新增缺 container 时非零且 fake kubectl 未调用的 RED→GREEN；保留原不可变镜像、UI envFrom、patch/apply 错误、参数/临时文件安全场景 |
| `tests/listingkit_deploy_workflow_test.go` | 只退休两个已取消 schema Job 的 deadline-wrapper 断言；保留原 worker/canary capability 隔离及当前镜像 routing label 检查 |
| `tests/listingkit_secret_boundary_test.go` | 只退出旧 API/已删除 Job 的正向绑定与专属 helper；保留 UI/SHEIN/imgproxy 的完整最小 Secret allowlist、禁用 envFrom Secret 与禁止身份高权 token 断言 |
| `tests/commercial_readiness_workflow_test.go` | 仅退休旧 API readiness、旧 Job image/legacy-scope 正例；旧 API invitation/SMS 正向绑定退出，改为旧入口/路由缺席并保留所有当前 workload 的身份/SMS token 隔离、除已删旧publisher专属邀请/SMS Secret模板外的canonical Secret examples。ImageAgent scopes、当前 Docker/worker/schema/build/商业门禁与 workflow 断言保持 |
| `tests/listingkit_release_gate_runner_test.go` | inventory 只保留 canary；原 fixture target/相同 invocation 字符串换成真实保留 canary。canonical/current-invocation、所有 command/args/env/envFrom/volume/mount/SA/security/resources/image/sidecar/Pod拒绝等正负例及断言完整保留，不删除整个测试文件 |
| `tests/listingkit_release_authority_policy_test.go` | 原样承接三项共享utility及标准库import；同步 runner 为单一 canary、三Deployment/一邀请Secret源权限退出与旧producer/执行字段缺席。只删除取消producer-owner fixture的必存在项及旧元数据正例。OIDC claims/ref/audience、无长效fallback、create-free bootstrap、native sidecar、固定失败旁路和其余全部安全负例保持，不能只让结构体忽略缺字段 |
| `tests/listingkit_release_boundary_test.go` | 退休两个旧 API receipt 精确 SHA/attempt 专属测试及无其它消费者的 scenario/helper；原 drain runbook 检查改为历史证据/不得据此停 worker 的边界，不再要求旧脚本可执行。保留 steady-state 两worker/无有限canary、v2-producing rollback 禁令与 durable-history/staging安全条件。当前 canary executor 的真实 invocation/run-attempt 与其余有效测试不变 |

执行器的显式目标规则仅退出已取消的默认值，不改变当前 caller 的身份或授权；所有当前 wrapper/owner 已显式给值。source Role/机器策略必须三名称同时减除且其它字段逐字保持，原 Rego + positive bundle/negative fixtures证明没有扩权和输入缺失；不能放宽策略、删除 verifier/CI step、改选择条件或伪造缺失输入。

实施自检限原 guard 的 RED→GREEN、受影响原 Go 测试、两个已存在 fake-command fixture 族（apply、gate）及原发布规则检查；不得执行真实 kubectl/SQL/Temporal/发布动作。完整检查留稳定候选的原 PR CI。假命令必须优先于主机 PATH，失败不允许落到真实工具；不新建故障注入、人工验收或验证工具。保留未变代码的已有有效证据，修复只查相关增量。

### 6.1 直接说明、旧保留结论与设计阶段文件

| 准确文档 | 确认后处置 |
| --- | --- |
| 本文 | 仅阶段说明反映已确认/实施；冻结矩阵保持，滚动 HEAD/CI/review留PR |
| `docs/architecture/README.md` | 索引本文与状态，不改其它 owner |
| `docs/refactoring/legacy-register.md`、`module-target-mapping.md` | 顶注仅替代本批十五删除源/十二混合源的旧保留含义；原矩阵、历史行与计数不重写 |
| `docs/refactoring/code-health-decisions.md` | 仅顶注取代旧 schema Job shell-test、旧producer drain/receipt与已消失回填包装的当前执行保留含义；保留历史 rows/counts/failures、通用 helper 与仍有效安全测试结论 |
| `deployments/kubernetes/listingkit-workbench/README.md` | 仅窄历史顶注：准确源/producer身份消费者与邀请/SMS模板删除、旧 API URL/SMS 接线退出；当前SMS仍用原Account Compose的ZITADEL_SMS_CONFIG_FILE，旧K8s凭据provisioning片段不构成当前许可。历史操作/SQL/drain/attestation/rollback正文保留为证据，不可作为当前执行、回滚或可停worker许可。保留当前Go恢复/UI/worker/OIDC/RBAC/schema与durable owner，未来真实停止另行授权，不增加试用前置 |
| `docs/architecture/release-identity-preflight-retirement.md` | 仅追加窄 superseded 顶注，取代其两个schema runner/旧Job及旧producer-bound drain/attestation/邀请预检消费者与专属源get权限保留含义；原冻结设计正文/历史逐字保留。正常schema CLI、当前ImageAgent canary/Go恢复、有效安全规则与失败发布入口不受取代 |

不修改其它 dated spec/plan、历史验证报告、商业/生产/恢复政策或 issue357。旧源码路径/操作样本在历史正文及禁止输入中保留，不据此恢复源。全批42文件；本设计阶段只改本文/四索引共5文档，其它37准确文件确认前零改。

## 7. 独立检查、准入、交付与权限

1. 当前只交设计：五文档、原 Greenfield/ArchitectureReadme guard、相对链接与 diff 检查；禁止运行相关 shell/Pester、kubectl（含dry-run/kustomize）、Docker/Compose、token/attestation/drain helper、SQL/Temporal/provider或真实状态探测。
2. 在同一主要 PR 做适用独立设计检查：核对实际消费者、42文件矩阵、旧producer-bound消费者完整退出、source Role三Deployment/一Secret收缩、共享测试utility原样归位、当前UI/worker/Go恢复/schema及全部有效安全断言；不重新审查全局架构。同SHA Running不重复触发，不默认增加实现者或AI互验线程。
3. Finding 先分类；BLOCKER须明确 AGENTS 的具体后果、当前Must与阻塞层级。最多两轮正常设计检查；第二轮非Blocker按IMPLEMENTATION_TEST/BACKLOG收敛，只核对修复增量，不重开全局架构。专属邀请预检/权限尾部与三个共享测试utility归位属于该边界的最小增量。非Blocker hardening不变成试用前置或新平台，没有新增风险接受。
4. 独立检查达到设计准入、用户确认后，在#29/#301写明冻结设计 **IMPLEMENTATION_READY** 与范围，分别读回Ready才改正式源/测试。同一Writer/分支/PR整批实施；稳定候选前核对实际main，只处理真实组合变化。
5. 完整闭包形成后一次最终独立交付检查真实diff、保留路径/断言和现有证据，范围内修复只复核增量；原CI与检查不等于用户运行验收，不签发可上线或生产接受。

交付结果为“旧 API/Task 发布源与误导接线退出，现有当前能力继续沿原入口”，不新增部署/验收步骤。用户另行授权最终准确 HEAD 后正常 expected-head 合并，不强推/管理员绕过/改保护；然后核实实际 merge SHA 的 event=push/branch=main/completed/success CI，source PR CI不得冒充main。

本次授权仅设计及必要文档检查/提交/推送/PR维护。正式实施、合并、部署、现有实例/业务数据/凭据/IAM/已有报告操作、Issue关闭与其它批次均未授权；#29/#301保持OPEN，生产NOT_READY。
