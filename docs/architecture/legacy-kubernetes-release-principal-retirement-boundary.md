# 旧 Kubernetes API/UI 发布身份与策略耦合：完整退休闭包

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。承接 [#847 的发布身份留债](https://github.com/qq550723504/task-processor/pull/847#discussion_r4322622300)。

> Status: **DESIGNING / 待独立设计检查与用户确认，Ready=false；未实施。** Design Basis: **Independent Architecture**。调查/设计基线 main `571744cec29160b3be6382a1b9402e242f433297`（2026-10-20）。当前只修改本文及四索引的提案说明；准确 HEAD、CI、复核与授权在同一主要 PR。未经设计确认与准入，不修改声明、脚本、策略、测试或运行指南。

## 1. 产品结果、Authority 与阶段

使用者是维护当前应用和内部试用的操作者。结果是从源码退出已关闭 API/UI publisher 的发行身份、授权声明、令牌配置/刷新 helper 及完整专属尾部，消除继续配置旧身份的指引；保留当前仍有效的固定失败入口、canary 和 UI 配置隔离检查。没有当前核心试用操作在等待此维护，不增加试用前置。用户明确同意先做完整设计，确认后一个主要 PR 整批实施，不按身份、helper、策略或测试机械拆批。

Authority：[Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Project boundaries](project-boundaries.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、已确认的[发布预检退休](release-identity-preflight-retirement.md)及[#847 执行闭包](legacy-kubernetes-release-execution-closure-retirement-boundary.md)。当前试用仍沿 [Account Compose](../../deployments/docker/account-compose/README.md) 与原私密配置；[RUN-1](../operations/current-application-run1.md) 是验收环境，不替代试用或生产交付。无 UI/IA/交互变化，Figma N/A。

Must：完成 §5/§6/§6.1 的整个源码闭包；保留所有仍有效的安全行为及此前保留能力；两套 CI 与全部其它 workflow、必需 gate、保护、inventory/历史计数/ceilings、依赖/锁零改；不操作实例、业务数据、私密凭据、IAM 或已有报告。Should：复用原 Conftest/OPA、原测试 owner 与 Bash/Go 工具，不造新的发布、身份或验收平台。

Out of Scope：当前 native identity/Auth.js/Organization 认证授权、实际集群/IAM/Environment/Secret/凭据撤销、生产替代发布、当前 ImageAgent/SHEIN 恢复和持久化、issue357、#692/#849 等其它 owner。SHEIN 自己的 `shein-login-worker-deploy.yml` 不消费本批 OIDC helper，其既有独立凭据/发布合同不变，不因本设计改造其认证。没有新增 Accepted Risk；实际旧身份是否仍存在、历史 run/re-run 和外部 IAM 的状态均 **UNKNOWN**。真实运行/平台/用户验收 **NOT_RUN**，生产 **NOT_READY**。

## 2. Design Basis 与实际风险边界

采用 **Independent Architecture**：本轮完整删除两套源 Role/RoleBinding 和 Kubernetes GitHub OIDC 认证声明，同时收敛机器策略的输入合同与原 CI 消费者；#847 原保留边界没有批准该删除。必须先明确“没有旧发布身份”的源合同及有效安全行为归属，不在实现时临时决定。预计 **42 个准确实施文件：23 删除源、7 混合源、4 原测试、8 文档**，超过文件提示线但仍是一个闭包/一个主要 PR。

本轮改变的是旧 publisher 的源码授权库存与静态检查输入，不引入新的当前登录/租户/授权事实、业务状态机、schema、事务、计费、provider、retry/UNKNOWN/recovery 或跨系统协议。Threat Model 限于旧身份源复活、错误新增发布权限、删输入导致 Rego 静默 undefined-pass、有效 canary/UI 安全断言退出，以及将源删除冒称为真实撤权。只有实际证据命中 AGENTS 的 BLOCKER 才重新打开已冻结设计；不把外部 IAM 调查或新发布平台加为前置。

## 3. 实际消费者与完整调用闭包

| 对象 | main 基线的源码事实 | 处置 |
| --- | --- | --- |
| 两个 API/UI workflow 与 PS wrapper | 都是原固定失败入口；workflow `permissions: {}`，无 Environment、OIDC、action 或正常 publisher step | 全部原样保留；不创造当前 token issuer/publisher，也不删除阻止旧入口恢复的实际完整形状检查 |
| 两 Role/RoleBinding、AuthenticationConfiguration example | 声明两个旧 subject/environment/audience/workflowRef；没有其它当前 workflow 使用这两身份。源 Role 仍能修改 UI/worker/共享配置，不意味着实际集群已安装或可用 | 五个源声明同批 RETIRE；真实权限不访问、不撤销，不迁移给其它身份 |
| refresh → configure kubeconfig | refresh 请求 GitHub token 后调用 configure；后者做本地 claims fail-fast 并写 kubeconfig，签名由 kube-apiserver负责。非文档消费者只有专属 shell/Go 测试与机器策略 metadata | 两 helper 和专属 shell test 整退；不把 JWT/令牌签发协议挪到当前 native identity，不新建替代 helper |
| machine policy / Rego / verifier | 现有 `oidc/rbac/protectedResources` 要求旧身份源必存在；同一 bundle 还拥有固定失败、UI scopes 隔离、当前 canary/工具 pin 等有效检查 | EXTRACT 后者留原静态安全 owner；旧身份字段/selector/正例同批退休，不能只从 verifier 丢文件令规则失效 |
| negative fixtures | 21 个输入：15 个只约束旧 token/可执行发布角色协议，6 个仍覆盖拒绝 kubeconfig fallback、reusable/composite/action 漂移、policy repo 与 gate Pod 查询 | §5.1 退 15；§5.2 精简两个混合 fixture；六个当前负例及实际 workflow/canary 负例继续被原 CI执行 |
| 其它测试消费者 | production-release 测试部分读取旧 UI Role；gate 测试一项读取 API Role；旧 retirement guard 把 Role 当保留文件并假定 `oidc` 存在 | §6 四文件同步，保留有效 UI/config/gate 不变量，不用 missing-file skip 或空输入成功替代 |
| 当前共享工具/能力 | canary、generic gate executor、显式 container 的 apply/UI/worker wrappers、UI auth ConfigMap、normal schema、SHEIN 登录/Store 都不需要上述 helper 或 subject | 源码及有效行为原样；保留工具不隐含授予执行或创建新的生产发布身份 |

contract → implementation → injection → consumer：固定失败/API/UI 源、当前 canary 与 UI 配置合同 → 原 Rego 与 Go/fake-command guards → 原 verifier 的确定输入 bundle → 原 CI `Release Authority Policy`、Backend/Architecture 检查。退出的是该路径中仅供旧发布角色的输入，不是绕过检查。`listingkit-release-authority` 文件/namespace/CI 名称作为原静态检查入口保留，不再携带旧可执行身份或兼容 decoder。

## 4. 权限、持久化与平台副作用 owner

| 边界 | Owner / 本轮不变量 |
| --- | --- |
| 源发布授权 | 两旧 principal、Role/Binding/auth example 删除后，受治理 assembly 只引用现有 canary manifest；不加 User/ServiceAccount、subject/audience、长期凭据、Role、Binding 或替代执行主体 |
| 机器策略 | 原静态 policy owner 保留 tools、UIAuthorization、releaseGateRunners、documentation；以窄 `assembly.kustomization` 指定现有源聚合。`oidc/rbac/protectedResources` 必须缺席，旧结构重新出现必须拒绝，不作 fallback/双输入 |
| 真实 Kubernetes/IAM/Secret/GitHub Environment | 原运行/身份管理员负责；本批只改 Git 源，实际状态 UNKNOWN，不 apply/delete/scale/get，不读取私密 manifest、token、kubeconfig 或已有报告 |
| 当前身份/业务事实/数据 | 当前 Console/Auth.js/native identity/Organization/domain/schema、ImageAgent store/Temporal、SHEIN/Store 原 owner，全部不变；不迁移表/行/history/资产，不提供兼容或第二事实源 |
| 通用执行器及凭据 | 保留显式目标/immutable/当前 invocation/清理等原合同；凭据仍由原独立授权 owner 提供。保留源码和 fake test 不授权执行，也不借此新增 token fallback |

没有新的多步持久化协议或不可重复外部副作用。现有失败/取消/并发/重启/retry/UNKNOWN 沿当前 owner处理；源码删除不触发真实权限撤销或 worker 停止。若发现具体当前外部契约/持久化资产强制要求旧身份兼容，停相应部分并报告用户，不自行建立 Exception。生产发布保持 NOT_READY，未来真实撤权或新发布分别另获授权及适用方案，不是试用前置。

## 5. 删除 / 保留 / 抽取矩阵

### 5.1 二十三个准确删除源

```text
deployments/kubernetes/listingkit-workbench/release-authority/listingkit-api-release-role.yaml
deployments/kubernetes/listingkit-workbench/release-authority/listingkit-api-release-rolebinding.yaml
deployments/kubernetes/listingkit-workbench/release-authority/listingkit-ui-release-role.yaml
deployments/kubernetes/listingkit-workbench/release-authority/listingkit-ui-release-rolebinding.yaml
deployments/kubernetes/listingkit-workbench/release-authority/kubernetes-authentication-config.example.yaml
scripts/listingkit-configure-github-oidc-kubeconfig.sh
scripts/listingkit-refresh-github-oidc-kubeconfig.sh
scripts/tests/listingkit-github-oidc-refresh-test.sh
policy/listingkit-release-authority/fixtures/negative/excess-rbac-verbs.yaml
policy/listingkit-release-authority/fixtures/negative/excess-rbac-resources.yaml
policy/listingkit-release-authority/fixtures/negative/excess-rbac-resource-names.yaml
policy/listingkit-release-authority/fixtures/negative/shared-environment.yaml
policy/listingkit-release-authority/fixtures/negative/shared-subject.yaml
policy/listingkit-release-authority/fixtures/negative/missing-id-token.yaml
policy/listingkit-release-authority/fixtures/negative/wrong-audience.yaml
policy/listingkit-release-authority/fixtures/negative/workflow-job-owner-drift.yaml
policy/listingkit-release-authority/fixtures/negative/workflow-step-owner-drift.yaml
policy/listingkit-release-authority/fixtures/negative/with-target-bypass.yaml
policy/listingkit-release-authority/fixtures/negative/workflow-ref-feature-branch.yaml
policy/listingkit-release-authority/fixtures/negative/workflow-ref-tag.yaml
policy/listingkit-release-authority/fixtures/negative/workflow-ref-path-prefix-lookalike.yaml
policy/listingkit-release-authority/fixtures/negative/workflow-ref-missing.yaml
policy/listingkit-release-authority/fixtures/negative/trusted-workflow-ref-policy-drift.yaml
```

仅删准确 tracked 文件；不递归清目录或删除真实对象。三项旧 excess-RBAC 负例的有效“不得额外授权”意图改由受治理 bundle 的零 Role/Binding/auth 对象及 assembly 精确库存负例证明；旧 helper claims/audience/ref 正例随已取消签发路径退出，不复用到当前认证。

### 5.2 七个混合源的精确处置

| 文件 | 允许的变化 / 保留 |
| --- | --- |
| `deployments/kubernetes/listingkit-workbench/release-authority/kustomization.yaml` | 只去四个 Role/Binding resource；namespace 与仅剩 `listingkit-release-gate-runners.yaml` 聚合保留，不更换 current canary owner |
| `deployments/kubernetes/listingkit-workbench/release-authority/release-policy.yaml` | 删除整个 `oidc/rbac/protectedResources`；以 `assembly.kustomization` 指向原 kustomization。repository/namespace、tools pin、uiAuthorization、releaseGateRunners、documentation 全部原值，canonicalWorkflows仍是两个固定失败源，无新生产身份 |
| `policy/listingkit-release-authority/release_authority.rego` | 退休旧 role/binding/auth selector、签名比较、两身份/claims/ref/权限库存专属规则；保留全部固定失败完整形状、workflow input 数量、不可变 action、repository/tools、UI 配置隔离、canary/sidecar 等规则。新增窄源退休约束：旧机器字段不可重现、bundle不含Role/RoleBinding/AuthenticationConfiguration、assembly资源精确只含现有canary。政策/assembly/workflow/UI必要输入缺失必须失败，不得靠undefined、跳过输入或marker放行 |
| `policy/listingkit-release-authority/negative_fixtures.rego` | 仅保留六个当前负例规则：kubeconfig fallback、reusable workflow、composite action、gate Pod查询、policy repository、mutable action；退休旧OIDC/角色协议比较，不保留无当前owner的输入字段 |
| `policy/listingkit-release-authority/fixtures/negative/mutable-action-ref.yaml` | 仅保留当前 `workflow.actionUses` mutable-action反例，去旧 identity/identities/token/permissions/audience/job/step/RBAC/expectedWorkflowRef 等无消费者辅助字段；原 mutable action必须仍被拒绝 |
| `policy/listingkit-release-authority/fixtures/negative/pod-rbac-and-query.yaml` | 仅保留当前canary的 `releaseGateCommand` get-Pods负例，去已退Role的rbac/forbiddenResource和旧schema runner名称；Pod访问仍由原executor fake测试及策略负例拒绝 |
| `scripts/verify-listingkit-release-authority-policy.sh` | 从正例bundle只去五个已删声明；保留policy、assembly、canary、UI两ConfigMap/Deployment、两个真实workflow及原pinned Conftest。先正例成功，再对全部六个当前negative fixture逐项期待拒绝；不删除循环、改CI步骤、加skip/成功占位或新runner |

### 5.3 保留能力与有效行为归属

- 两套 CI `.github/workflows/ci.yml` / `commercial-readiness.yml`、其它所有 workflow、固定失败 API/UI 两源和 PS wrapper 原样。CI既有步骤名称仍保留，不以其历史“OIDC/RBAC”标签声称新身份可用；入口实际检查改为旧授权源缺席及当前静态安全合同。
- 当前 canary manifest、两个 ImageAgent 长期 worker、generic gate executor、canary-order guard、所有当前 invocation/generation/command/args/env/volume/SA/resources/sidecar/Pod拒绝负例原样。无真实canary执行或物理drain/停机授权。
- 共享 immutable library、显式目标 apply/UI/worker wrappers与其测试、UI两个ConfigMap/Deployment/allowlist、所有共享凭据键值、current UI/web/原生身份/token handoff、issue357、正常schema、图像/采集/平台/定价/规则/AI/Store/SHEIN登录与同步及此前能力全部保持。
- `tests/listingkit_release_documentation_ast_test.go`、Goldmark AST规则和触发全路径guard零改：仍沿保留的documentation/canonicalWorkflows检查原历史说明及禁止直接生产变更；不因身份退出取消该安全检查。
- EXTRACT仅指把仍有效静态安全行为从旧publisher输入耦合中解开，留原Rego/Go/Conftest检查owner。无生产算法或身份协议抽取，无新目录/检查器/权限系统/兼容字段。三个现有共享utility `preflightBash/writeBashPreflightFake/writeExecutablePreflightFake` 仍服务当前gate fake测试，原文件/body保留，不再机械搬家。

```text
Legacy decision: RETIRE；EXTRACT 仅现有静态安全行为，留已有检查owner
Reusable behavior: 固定失败源完整形状、当前canary/实际invocation/Pod拒绝、UI scopes隔离、不可变镜像/action及工具pin、文档AST安全
Current owner: 原CI静态策略/Go guard/通用执行器与UI、ImageAgent、native identity、schema、独立SHEIN/Store原owner
Cutover/deletion condition: 独立设计检查、用户确认、IMPLEMENTATION_READY和双Issue Ready读回后，同PR完成23删除源/7混合源/4原测试/8文档；原CI/最终独立交付检查后另获准确HEAD合并授权
```

## 6. 四个原测试 owner 与安全覆盖矩阵

| 原文件 | 精确处置 / 必须保留的验证 |
| --- | --- |
| `tests/listingkit_backend_retirement_test.go` | 在原absence guard增补23源缺席及旧身份/field/resource引用退出，先因源仍在RED，实施后GREEN；原§847 guard从“必须有oidc再查元数据”改为整个oidc/rbac/protectedResources缺席，不能nil-panic或忽略缺输入。旧identity-preflight guard只移去已删Role的保留扫描项，其余退休/当前构建/schema/worker/安全断言保持 |
| `tests/listingkit_release_authority_policy_test.go` | machine-policy正例更新为当前字段/精确assembly，去旧OIDC/Role/claim/ref正例、专属constants/cloneOIDCClaims/unsignedJWT和无消费者imports。保留真实固定失败正例及全部bypass负例、native-sidecar、workflow无长效fallback和三个共享utility。原runReleaseAuthorityCrossFilePolicy仅扩展窄fixture输入，覆盖注入Role/RoleBinding/AuthenticationConfiguration、assembly加资源、旧machine字段复活、缺policy/assembly/workflow/UI输入时拒绝；不得新建runner。六negative fixture确实存在、YAML可解析、输入无旧身份字段并仍被原verifier逐项检查 |
| `tests/listingkit_release_gate_runner_test.go` | 只退休读取旧API Role的那一项Pod权限正例，安全意图归前两项零身份/零Role源guard；其余当前runner库存、真实canonical/current-invocation/run-attempt/generation及全部fake-command拒绝断言原样，不能整个删文件或放宽相等比较 |
| `tests/listingkit_production_release_authority_test.go` | 仅退出旧UI Role读文件/授权正例和无代码对应的旧OIDC专属说明；保留固定失败workflow/PS完整形状、UI独占ConfigMap、禁止shared scopes、可信project字段等全部当前断言 |

原15取消fixture的有效意图必须有明确归属：额外RBAC verbs/resources/names → 受治理bundle零授权对象及assembly精确库存拒绝；旧job/step/target/ref/Environment/token权限绕过 → 原真实workflow完整形状及实际bypass负例；旧helper claims与token签发正确性 → RETIRE，无当前调用者、无替代token身份；有效通用六负例按§5.2保留。不能用“15测试删了”冒称安全提升，必须通过现有正/负例证明保留行为仍被调用。

实施自检限原retirement RED→GREEN、受影响原Go/fake-command检查、原Conftest positive/negative bundle和文档guard；稳定候选运行原CI。仅现有fake工具/隔离CI fixture可作执行证据，假工具失败不得落到主机真实工具；本地不执行真实kubectl/发布/helper/token请求、Docker/Compose/SQL/Temporal/provider。不增加专项验收session、故障注入或验证工具。真实环境未执行即NOT_RUN。

### 6.1 八个准确文档与历史保留

| 准确文档 | 用户确认后处理 |
| --- | --- |
| 本文 | 阶段说明反映确认/实施；冻结矩阵不倒推改写，滚动HEAD/CI/review留PR |
| `docs/architecture/README.md` | 当前设计索引/状态，不改其它owner |
| `docs/refactoring/legacy-register.md`、`module-target-mapping.md`、`code-health-decisions.md` | 只加窄顶注取代23源/7混合源的旧保留和旧正例含义；原矩阵/rows/counts/failures保持，不删历史证据 |
| `deployments/kubernetes/listingkit-workbench/README.md` | 只在现有历史顶注补足旧Role/Binding/auth/helper源码退出；历史IAM/credential配置与运维正文逐字保留为证据，不构成当前配置/执行/真实撤权许可 |
| `docs/architecture/release-identity-preflight-retirement.md` | 只加窄superseded顶注，取代其保留旧发布principal/Role/Binding/OIDC helper及输入库存的含义；原冻结设计/历史正文保持 |
| `docs/architecture/legacy-kubernetes-release-execution-closure-retirement-boundary.md` | 只追加窄顶注：本新确认设计取代#847中§5.3保留的准确旧身份/输入部分；原42文件矩阵、历史执行边界、当前canary/原恢复owner与全部历史正文保持 |

当前设计阶段仅本文/四索引 **5文档**，不提前取代原保留结论；其它37个准确文件零改。全批42文件，不修改其它dated spec/plan/报告或issue357，不为历史路径和样本建立兼容decoder。旧identity名字在历史正文或禁止输入中出现，不等于当前执行消费者。

## 7. 独立检查、准入和交付权限

1. 仅设计时，执行原Greenfield/ArchitectureReadme、相对链接/准确矩阵/历史正文/diff检查。不运行相关shell/Pester/helper、kubectl（含dry-run/kustomize）、Docker/Compose、SQL/Temporal/provider或真实状态/凭据探测。
2. 在同一PR做适用独立设计检查：真实消费者、42文件闭包、源零发布身份合同、机器bundle的缺输入/复活拒绝、有效安全覆盖逐项归属、当前SHEIN/UI/schema/ImageAgent及历史保留。不重开全局架构，不默认增加Writer或AI互验线程，Running同SHA不重复触发。
3. Finding先分类并写清当前Must/后果/阻塞层级；最多两轮正常Architecture Review，第二轮后非BLOCKER按IMPLEMENTATION_TEST/BACKLOG收敛，仅复核增量。只有实际AGENTS BLOCKER重新打开冻结边界，不追求自动0 finding或扩大试用准入。
4. 独立设计准入和用户确认后，在#29/#301写明冻结设计IMPLEMENTATION_READY及范围，分别读回Ready后才改正式源。一个Writer/分支/PR整批实施；形成稳定候选前同步实际main，只处理真实组合风险。
5. 完整闭包完成后，一次最终独立交付检查实际diff/保留行为与证据；修复只复核增量。CI/评审不是用户或真实环境验收，不自行签发生产可用。用户另行授权最终准确HEAD后正常expected-head合并，再核对实际merge SHA的push/main CI，不强推/管理员绕过/改保护。

交付是“旧发布身份及错误配置指引从源码退出，当前能力沿原入口与原owner继续”，没有新的操作/验收前置。当前授权仅设计、必要文档检查、提交/推送与PR维护；正式实施、合并、部署、实际实例/业务数据/私密凭据/IAM/已有报告操作和Issue关闭未授权。#29/#301保持OPEN，生产NOT_READY。
