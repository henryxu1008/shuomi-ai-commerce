# 发布身份预检与旧发布入口退休设计

> **Narrow source supersession — #850 / #29 / #301.** The [confirmed principal closure](legacy-kubernetes-release-principal-retirement-boundary.md) supersedes only this historical design's retained old API/UI release principals, Role/Binding/authentication declarations, OIDC helpers and their static input inventory. These sources have retired without replacement identities; valid fixed-failure, UI/canary and tool safety remain at their original owners. The frozen historical body below stays unchanged and grants no live IAM/credential/resource/data operation. All current retained capabilities, including independent SHEIN login/Store sync, remain; no new trial prerequisite.

> **Narrow supersession — #847 / #29 / #301.** The [user-confirmed Kubernetes execution closure](legacy-kubernetes-release-execution-closure-retirement-boundary.md) supersedes only this historical design's retained two schema runners/three old publishing Jobs and schema wrapper, old API producer-bound drain/attestation/invitation-preflight consumers, exclusive credential templates and invitation-Secret source get grant. Those sources now retire together; the frozen body below remains historical evidence and grants no current execution/credential-provisioning/rollback/worker-stop authority. Normal schema CLIs, current ImageAgent canary/Go recovery, retained source OIDC/RBAC and valid safety rules, UI/Store/SHEIN owners and the fixed-fail publishing entrypoints keep their contracts. This is source-only, with no live resource/data/IAM operation or new trial prerequisite.

Status: **IMPLEMENTATION_READY / FROZEN DESIGN**。用户于 2026-10-15 确认开始实施，适用独立 Architecture Review 已完成；生产发布保持 **NOT_READY**。评审、批准与准确 HEAD 的滚动证据只记在 #529 / #301 和同一主要 PR。

## 1. 产品决定与用户结果

执行依据：[#529](https://github.com/qq550723504/task-processor/issues/529)、[#301](https://github.com/qq550723504/task-processor/issues/301)。用户于 2026-10-14 明确批准先设计、确认后实施，并选择：**沿现有内部试用交付，设计旧发布链退休边界，生产发布保持 NOT_READY**。独立 SHEIN worker 必须保留。

本设计解决构建与维护操作者的一个具体问题：当前应用已经直接使用合法 Organization、独立 owner/pool 与私有 manifest，但当前源码和镜像仍提供需要旧租户 metadata 的身份预检；旧 Kubernetes 发布入口也无法发布当前应用。完成后，当前内部试用交付不再携带该旧预检/数字身份映射，旧发布入口明确失败关闭，不向操作者提供错误的发布或回滚成功路径。

当前内部试用启动本身不调用该预检；Account Compose镜像也不编译它。本批清理的是其它当前源码/镜像包装与错误发布入口，不把它描述成内部试用启动故障，也不为此新增身份预检平台。

这不是新增生产发布能力。内部试用、源码/CI、实际平台调用、用户验收和生产发布分别记录。旧源码停止使用不代表既有集群、镜像、IAM、数据或历史工作流运行被撤销。

### Authority

- [全新系统基线](../product/greenfield-no-legacy-migration.md)：#365 / PD-GREENFIELD-NO-LEGACY-MIGRATION-2026-09-08；不承接旧数据、ID、profile、任务或权益。
- [Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md)、[project boundaries](project-boundaries.md) 与 [Issue 规则](../engineering/issue-driven-development.md)。Legacy 仅 EXTRACT / RETIRE，无 compatibility exception。
- 当前内部试用合同：[Account Compose](../../deployments/docker/account-compose/README.md)、[Native Modules](../../deployments/docker/account-compose/MODULES.md) 及每个已准入 owner 的原配置/初始化合同。已保留实例继续由其原运行负责人维护。
- [RUN-1](../operations/current-application-run1.md) 是独立身份/路由验收环境，不能作为内部试用完整功能或生产部署的替代证据。
- [旧 Kubernetes 文档](../../deployments/kubernetes/listingkit-workbench/README.md) 及旧 RC/rollback 文档仅保留历史证据，不恢复旧阶段要求。无 UI、IA、页面或 Figma 变化。

## 2. Scope、Must 与边界

Must：

1. 保留 current-application、当前 Console、合法认证/Organization/领域 owner 及已准入内部试用入口；沿原配置和事实保存合同运行。
2. 保留独立 SHEIN worker 的 cmd/runtime、AccountRepository/StoreAPI、Redis、浏览器/profile、健康/关闭、原账户隔离和恢复语义；其独立构建/部署定义不并入旧 API 发布链退休。
3. 退休旧 identity-preflight CLI、独占 runtime/readers/resolver，移除镜像包装和真实调用；不能以改名、换目录、fallback 或新预检框架保留旧 metadata 依赖。
4. 当前默认源码中的旧 API/UI 发布入口在任何 build/push、凭据/OIDC读取或外部 mutation 前明确拒绝；旧 wrapper 同样拒绝。失败不能写成 passed、READY 或可发布。
5. 保留有效认证、跨组织、权限、幂等、worker、schema、备份恢复及发布安全断言。CI classifier、Required CI Gate、检查选择条件和分支保护不变；旧协议断言按第 7 节重定向或退休，不跳过检查。

Out of Scope：生产 Kubernetes 发布的替代 workflow、新的发布平台/runner/验证工具、真实数据迁移或历史兼容、provider/资金/AI 业务改变、现存服务的停止/销毁、真实 IAM/RBAC/集群/凭据操作、历史 runs/artifacts/镜像删除、其它活跃 owner 的批次。当前正式生产发布继续 NOT_READY。后续生产合同须单独准入，不能把它建设完成作为本次源码退休前置。

### Threat boundary

保护对象是当前交付和当前默认源码入口：避免旧映射污染 current identity、越权扩权、误触发旧发布及错误成功状态。沿原 live Organization/grant 与领域合同拒绝越权，不能用配置、菜单或测试 fixture 代替授权。

当前设计不声称旧 run/re-run、历史镜像或已授予的集群权限失效。默认分支源码的拒绝入口不会撤销外部既有执行者。那些状态没有读取或变更，保持 NOT_RUN，由另行授权的运行/发布 owner 判断；本设计不自行接受新风险，也不把历史服务继续运行解释为新系统的兼容义务。

## 3. 准确现状与消费者

只读调查基线为 **main 5748d409bd2426df71910284159d4ffd23f393d4**。下面是源码证据，不是实际发布/登录/用户验收。

实际 `go list -deps`：

| 命令 | 四个旧 package 的真实编译依赖 |
| --- | --- |
| `cmd/current-application` | 无 |
| `cmd/image-agent-temporal-worker` | 无 |
| `cmd/shein-login-worker` | 无 |
| `cmd/listingkit-identity-preflight` | identitypreflight、ownerreconcile、userdirectory、tenantbridge 全部存在 |

旧 CLI 的唯一生产装配是 `internal/app/runtime/listingkitidentitypreflight`；整个旧预检链是 ownerreconcile/userdirectory 的唯一生产消费范围，其中runtime和identitypreflight都消费userdirectory。其余为专属测试与原 guard。上述 cmd/runtime/三包/tenantbridge 共 26 个 tracked Go 文件。后续加上 ops/测试/文档可能超过 30 个范围文件，采用 Independent Architecture、一个主要 PR，不为计数拆 PR。

预检调用链：旧 YAML config → ListingKit owner DB → ZITADEL metadata DB（`projections.org_metadata2` / `projections.user_metadata5`）→ owner reconciliation / legacy numeric Organization resolver → 旧 owner inventory + user directory → passed/blocked 报告。inventory 包含旧 Task、POD、listing_store 等表；不是 current 全新安装 schema/grant 合同。

| 包装/消费者 | 准确源位置 | 现状 |
| --- | --- | --- |
| 当前应用镜像 | `deployments/docker/Dockerfile.product-listing-api` | 默认运行 current-application，但仍 build/copy/chmod 旧预检 binary |
| 专用预检镜像 | `deployments/docker/Dockerfile.listingkit-identity-preflight` | 只编译旧 CLI |
| API workflow | `.github/workflows/listingkit-deploy.yml` | 仅 workflow_dispatch；含 normal / rollback。normal 构建专用预检镜像并运行预检 runner |
| UI workflow | `.github/workflows/listingkit-ui-deploy.yml` | tag / workflow_run / dispatch；依赖旧 API exact-run attestation 后 build/push/rollout |
| PowerShell 发布 wrapper | `scripts/build-push-deploy-listingkit-workbench.ps1` | build/push/schema/preflight/API/UI 编排 |
| 专属 Job wrapper | `scripts/listingkit-identity-preflight-job.sh`、对应 tests | 创建 Job 并等待退出；只服务旧 CLI |
| Job / runner | `jobs/listingkit-identity-preflight-job.yaml`、`release-authority/listingkit-release-gate-runners.yaml` 的 identity Deployment | 调用旧预检 binary |
| Source authority | `release-policy.yaml` / API Role 的 identity runner 资源项 | 固定旧 runner 名称、权限/step 等合同与检查耦合 |

这些 Kubernetes 路径均位于 `deployments/kubernetes/listingkit-workbench/`。旧 workflow 要求 `/app/product-listing-api` 与旧 YAML/shared-DB rollout；当前镜像运行 `/app/current-application` 并要求私有 JSON manifest，原兼容性门禁已拒绝它。新系统不能补回退休 binary 或伪造兼容标签来过门禁。

当前 tenantbridge frozen import edges 恰好三条：preflight runtime、它的测试、Workbench resolver-isolation 测试。最后一条是有效安全回归，不能为归零直接丢弃业务断言。

## 4. 当前交付 owner 与合同

当前链：现有内部试用的私有 manifest/批准的 Compose profiles → currentapplication 原配置验证、原 owner 的独立 schema/grant 初始化 → current HTTP/Console 和 opt-in worker → 原授权/领域/持久化 owner。没有新的发布编排、身份目录、状态机或 second fact source。

| 关注点 | 唯一 owner / 保持的合同 |
| --- | --- |
| identity / Organization / live roles | 当前 authidentity、authruntime、workbenchcontext、authz 与 native ZITADEL/成员 owner |
| 领域事实/数据库/初始化 | 各已批准 current owner、原独立 pool/角色与 schema installer；正常新系统 schema 演进和备份恢复保留 |
| SHEIN standalone 服务/账户/队列/profile | 原 sheinlogin service + app/runtime/sheinloginworker；原数字账户持久化仍是 #301 债务，本批不改造成 Organization owner |
| 当前应用/Console/worker 打包 | 原 Docker/Compose/manifest owner；当前应用 image 仅移除旧预检包装，原其它 worker/schema 工件保持 |
| 旧发布入口的拒绝 | 原两个 workflow 文件和原 PS wrapper 的最小失败入口；没有替代执行能力 |
| 发布权限验证 | 原 CI job + Conftest/OPA + 原 guard/负例；适配本阶段有效对象，不另建治理体系 |

current-application 正常命令仍是 `go run ./cmd/current-application -config <absolute-private-manifest>`；SHEIN worker 仍为 `go run ./cmd/shein-login-worker -config <existing-private-worker-config> -health-port 8086`。既有实例的 restart、overlays、数据保持规则仍由其原 owner 决定；本设计不启动、重置或替换实例，也不把 RUN-1 的 disposable launcher 当成完整交付。

Organization ID 是当前 owner 的 opaque string，合法 ZITADEL ID 可以只包含数字。删除旧 numeric mapping 不等于禁止数字字符串 ID，也不能做数值相等推断。旧 header/旧 metadata 不能覆盖经过验证的 effective Organization 和当前 scoped roles。

## 5. 源码退休方案

### 5.1 先关闭旧 API/UI 发布及 PS 入口

后续实施在同一主要 PR 中将两个旧 workflow 收敛为明确的失败入口：仅保留手动 dispatch 的可见弃用说明、最小 job、空 token permissions 与直接非零退出。不保留旧 tag/workflow_run 自动触发、normal/rollback 输入绕行、checkout/build/push、environment、secrets、id-token、复用 workflow/action 或 mutation job。

实际形状必须固定到这两个文件：顶层只有原name、无inputs的workflow_dispatch、`permissions: {}`、一个拒绝job；job只有固定runner和一个step。step的run是固定弃用错误消息与下一行`exit 1`，无GitHub表达式、shell变量/命令替换、if/strategy/needs/outputs或第二step。原checker核对这个精确形状和字面命令，不依靠注释、marker或一般化的“没有发现kubectl”字符串扫描。

PS wrapper 在无副作用参数绑定之后直接拒绝，不能要求用户先输入凭据或执行 ValidateScript。入口不能读取配置/凭据、构建上传镜像、查询或修改集群。专属 identity Job wrapper 及其测试物理删除。

wrapper只保留原文件名、必要的可选静态参数和固定throw；不留下throw之后的旧函数/编排供dot-source或其它入口调用。不能用输入决定是否拒绝。

**设计确认的具体后果**：当前源码不再提供旧 API/UI normal release 或旧 rollback 操作入口；UI 不接受历史成功 API receipt 来重开该入口。它们的历史 Git/已发布镜像/回执保留。这里不停止任何现有 worker/服务，也不撤销历史 run 权限。若出现具体当前的旧 rollback 操作义务，必须在相关实施前报告用户，不能自行恢复兼容或默认留旁路。

### 5.2 删除独占旧身份链，清除包装

RETIRE 下列源码及其仅用于旧协议的测试：

- `cmd/listingkit-identity-preflight` 与 `internal/app/runtime/listingkitidentitypreflight`。
- `internal/listingkit/identitypreflight`、`ownerreconcile`、`userdirectory`。重新确认生产 consumers 为零后整体删除；旧目录/metadata/fingerprint/fixed exceptions 不建立新 owner。
- `internal/tenantbridge`。先完成 5.3 的有效回归转换，清除全部 import 后删除 package/测试；不得新增 fake legacy resolver/global registry 来保留测试钩子。
- 专用预检 Dockerfile、专属 Kubernetes identity Job，以及 common runner manifest 中的 identity Deployment。
- 当前应用 Dockerfile 的旧预检 COPY/build/copy/chmod 项；保留 current-application、ImageAgent、现有 schema 工件。Account Compose 的现有 image/manifest/initializer 合同不改。

Source policy / API Role / common runner 的精确 identity runner 项随真实对象退出同步删除；其它 subject/audience/workflowRef、API/UI 区分、权限资源和 worker/schema 定义不得扩权或机械批量清理。该 source 修订不执行 apply，不代表 live IAM/RBAC 已变化。

准确 maintained command inventory 移除这一个 command；其它 schema、worker、installer、backup/recovery 不因名称包含 legacy/listingkit 自动退休。`shein-login-worker-deploy.yml`、SHEIN Dockerfile/deployment、当前 ImageAgent/Temporal worker 定义与业务实现保持。

### 5.3 保留安全行为，归零实际依赖

Workbench 原测试保留 canonical effective Organization、当前目录 scoped viewer role、旧 token role不能继承管理员权的断言。去掉对已不存在的全局 legacy resolver 的安装与调用计数；结合原真实 command dependency/absence/import guard，证明没有映射实现可达。

必要的增量只覆盖这个迁移后的断言：包括合法纯数字 Organization ID 按当前目录/授权合同处理，不被解析为旧 int64；原跨组织 role隔离/撤权/无权限测试不删除。不能为测试造新的桥接接口或改 serving 身份逻辑。

`legacyConsumerBaseline` 从三条减到空集，删掉三条对应 depguard 文件豁免，保留 deny prefix 与检测逻辑。沿原退休/真实依赖守卫覆盖 CLI、独占 packages、包内源码和包装退出；原 retained-preflight reader guard按已取消消费者退休或改为 absence断言，不用 skip/遇文件不存在就成功掩盖真实遗漏。

## 6. 状态、失败、恢复和外部副作用

没有新业务状态机、schema、数据库 owner、事务边界、event/outbox、补偿或恢复协议。当前 serving 的幂等身份、相同键不同载荷、UNKNOWN、预算/计量、权限撤销和取消处理均保持其原合同。

| 行为/失败点 | 规定结果 |
| --- | --- |
| 调用退休入口、重复调用、旧参数、历史 receipt | 始终非零拒绝；不创建 release/READY/成功回执；不进行外部 I/O |
| 修改源码或编译中途失败 | 未发布候选由原分支修复；不能交付半套 command/package 删除或重开旧入口 |
| current manifest/grant/schema 不合法 | 原 current startup fail-closed；不读取旧 metadata 或启动旧 preflight作为 fallback |
| current 服务请求失败/取消/并发/重启 | 原 domain、事务、worker 和 durable recovery owner处理；本批没有新的重试入口 |
| 源文件/runner manifest 删除 | 仅 Git 内容变化；不会删除已有 Job/Deployment/DB/profile/volume、停止执行者或修改 live permissions |
| 将来生产发布 | 继续 NOT_READY，需另行批准当前合同、权限/配置及适用实际验证；不能复活本次退休协议 |

退出顺序是源码提交/实现边界，不是对存量环境的 rollout/migration。保留历史证据不能用作兼容授权或将来当前 producer 的直接审批。

## 7. 有效门禁与测试分类

原 Release Authority CI job/Conftest/OPA、负例和 Required Gate 继续运行；classifier/选择条件/保护不变。**不能直接删掉 verifier调用、让 policy跳过所有 workflow、加入任意“retired”豁免或用新 YAML marker绕过实际源码检查。**

必要适配限于这两个已退休入口和已消失 identity runner：原 checker在同一 owner内检查实际 YAML 的唯一失败入口、空权限、无 secrets/actions/环境/自动触发/可执行旁路；原 wrapper检查在任何 operation前失败。正向检查覆盖真实两个 stub，增量负例证明重新加入 publish/mutation/id-token/environment/第二 job/条件绕过或 UI自动触发会失败。这是原 guard的局部输入/断言同步，不新增通用退休系统或 runner。

| 原断言 | 后续分类与处理 |
| --- | --- |
| API/UI 必须执行旧 identity-preflight、旧镜像 label、旧 v2/v3 release receipt步骤 | NOT_APPLICABLE于当前内部试用阶段；旧协议专属断言退休，不把旧流程复写到新应用 |
| 当前入口不能执行未授权 build/push/mutation或伪造成功 | Must；由原 workflow/wrapper边界guard核验实际失败入口和旁路 |
| audience/subject/workflowRef严格绑定、无长效kubeconfig fallback、无过宽RBAC、immutable image/attestation校验 | 保留原 helper/Conftest/OPA及已有负例的有效行为；不能为了 stub通过放宽这些约束 |
| 旧 runner行必须存在/有CLI | retired-object断言随源对象退出；其它worker/schema保护继续按原合同验证 |
| Workbench不能被旧角色/旧映射改变scope | Must；按5.3保留行为，原编译/absence边界证明旧package不可达 |
| 当前domain/worker/schema/备份恢复 | 原owner及必要检查保留；不新增完整故障矩阵或专项验收平台 |

仅真正可复用的能力保持原正确 owner，使用已有 Conftest/OPA/Go guard；只服务旧发布编排的 helper/test 在生产 consumers为零后RETIRE，不能整体搬入新的 compatibility/historical可执行目录。实施前逐 helper核对当前非测试消费者；未核对对象不按名字删除，也不因为未来生产可能需要而复活旧协议。

旧source Role移除 identity target属于收缩，所有其它资源/verbs/subjects/audiences保持。来源于历史特权run的实际执行能力未审计，不能把source CI标成集群撤权PASS。

## 8. 实现顺序与验收

采用 **同一主分支/主要 PR**。用户具体确认和适用架构准入已满足，按本文冻结边界实施。后续实际diff若要求改变这里保持的授权/事实/状态/事务/恢复边界，按 BLOCKER规则回到设计；非Blocker在本批必要实现测试中收敛。

实施顺序：

1. 对原退役/入口guard写能捕获目标遗漏的最小RED；同根因 workflow、PS、Docker、runner、library相邻路径一并核对。
2. 原入口关闭、专属CLI/packages/包装退出、有效Workbench回归转换和准确policy/guard同步在同一个稳定候选完成；中间按提交边界验证，不单开机械辅助PR。
3. 精确source/Go dependency freeze 3→0、maintained inventory和历史文档同步；只运行与风险相关的必要测试。稳定候选再跑适用CI、独立最终交付检查。
4. 实现完成交出准确HEAD及当前内部试用入口/保存/已知限制，由用户决定合并；获授权后核对实际merge SHA的push/main CI。生产/实际使用验收仍分别授权。

| 验收对象 | 所需证据 |
| --- | --- |
| 原源码consumer清零 | tracked Go的实际imports、四命令真实依赖、packages/CLI/包装absence；zero baseline与精确depguard，无新legacyconsumer |
| 旧入口失败关闭 | 原YAML/PS boundary tests及适用shell/Pester fixture，证明首次操作前拒绝、重复/旧输入/receipt均无输出成功或外部操作 |
| 有效权限/发布安全 | 原Conftest/OPA负例、实际stub增量负例、保留helper的相关断言；CI RequiredGate及选择规则未改 |
| current identity及独立worker保持 | 原Workbench scoped-role/组织安全回归、SHEIN worker/service账户隔离/健康/关闭、current command编译；不启动真实账号登录 |
| current初始化/持久化保持 | source diff及原适用schema/grant检查；已有不变代码证据可明确复用，无实际旧metadata、DB、IAM、profile或volume访问 |
| 设计交付 | 本文 + #529/#301当前范围/状态、一个Draft PR、准确设计HEAD的必要文档检查/CI与适用独立Architecture Review |

本设计不要求本轮启动 RUN-1、重新初始化已保留Store或执行生产Canary。没有真实执行的浏览器/平台、运行/用户、旧服务、IAM/RBAC和生产验收保持 NOT_RUN。

## 9. Legacy decision与实施准入

```text
Legacy decision: RETIRE
Reusable behavior: 已存在current identity/domain/worker/schema能力；原发布权限验证中独立有效的安全断言，保留在其当前owner。
Current owner: currentapplication/auth/Organization及各domain/worker；原CI权限验证owner；不新增旧metadata或发布事实owner。
Cutover/deletion condition: 当前内部试用无旧链编译依赖，旧default-source入口先失败关闭，唯一CLI消费者及包装退出，有效Workbench断言保留后bridge零消费者删除。
```

用户已确认产品阶段和本具体方案并授权开始实施，包括旧API/UI normal/rollback入口在当前源码停止使用、精确identity runner/source权限项收缩，以及原发布guard按本阶段有效对象同步。独立SHEIN worker保留，存量实例/数据/历史执行者不操作，生产保持NOT_READY。

适用独立Architecture Review针对上述真实边界进行，最多两轮正常评审；findings先分类，不从P1/P2自动推导新机制。具体确认/评审已完成，Issue在生产修改前记录IMPLEMENTATION_READY与Ready；合并、部署和产品验收仍分别授权。
