# 旧 ListingKit 本地启动与 ImageAgent 验收编排：完整退休闭包

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

> Status: **DESIGNING / 待独立设计复核及用户确认；未实施，Ready=false。** Design Basis: **Reuse Existing Architecture**。调查基线 main `da2467230a4455a98586868e9fbfd0200661c2d3`（2026-10-20，#844 已合并且对应 push/main CI 通过）。本阶段只改本文和四个索引/清单的提案说明；准确设计 HEAD、CI、复核与用户确认维护在同一主要 PR，确认前不改生产/测试/Compose/运行指南。

## 1. 用户结果、范围与非目标

使用者是维护当前应用、内部试用、采集、图像、Store 与独立 SHEIN 登录/同步的开发和运维人员。结果是完整退出已退旧 Task API 的本地启动和 ImageAgent 旧验收编排，避免操作者先启动服务、写入环境或修改身份，再在已退休 API 入口失败。当前有效能力继续使用原入口；维护本身不成为内部试用前置。

一个 Delivery Batch / 一个主要 PR 覆盖两个 fail-closed launcher、仍可执行的旧验收 orchestrator、全部内联 helper、专属测试/Compose 和直接说明尾部。八个准确源一起退出，不按语言、测试、配置或文件数拆小批。没有发现当前核心业务必须等待此维护；不为旧验收补建新平台或修复旧 Task happy path。

不做：当前 UI 启动、Auth.js/OIDC/cookie/refresh/会话、token handoff、issue357 runtime、原生身份 CLI/共享 ZITADEL、正常 schema/迁移命令、当前 ImageAgent/Temporal/图像/模型、通用 port-forward/DB、采集/Store/独立 SHEIN owner 的实现修改；不新增 runner/验收 session/替代验收流程，不迁移或访问真实状态。全部 workflow（包括两套 CI）、inventory/历史计数/ceilings/depguard、依赖/锁/保护零改。UI/IA/Figma N/A，当前产品 authority 不变。

## 2. Product Authority 与 Design Basis

依据：[Greenfield 产品基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Project boundaries](project-boundaries.md)、[Issue 驱动派工](../engineering/issue-driven-development.md)，以及已批准的 [#824 数字刊登支撑闭包](legacy-numeric-listing-support-closure-retirement-boundary.md)、[#844 Task 诊断/回归闭包](legacy-task-diagnostics-retirement-boundary.md) 与 [旧发布身份预检退休边界](release-identity-preflight-retirement.md)。旧 Task/API 与共享 YAML 启动不是当前产品 owner，入口 fail-closed 不是保留不可达源码和失效调用方的永久理由。

**Reuse Existing Architecture**：只完成上述批准退休决定的专属运维源尾部，不改变任何当前 domain/fact owner、认证/授权/tenant、状态机、事务、持久化、恢复或不可安全重复的副作用协议。§5 明确保留仍有当前消费者的身份、token 与 UI owner，且不把它们包成旧流程的替代实现。预计实施 22 个准确范围文件（含本文/四索引、文档与原 guard），无新增生产业务代码；没有通过拆片规避敏感阈值。若实施发现必须改这些当前边界，停止该部分并按 AGENTS.md 升级准入。

Must：完整退出八源及其直接说明；当前能力/必要安全断言和历史事实保留；不操作实例、数据、凭据或已有报告；不增加试用前置。Should：复用当前 owner/标准库/成熟 CLI/原 guard，无新抽取或治理平台。Out of Scope 见 §1。不新增 Accepted Risk；真实旧环境/卷/文件/身份/workflow 是否存在与其业务归属为 **UNKNOWN**，未调查不等于不存在、可停止或可删除。

Threat Model 限于误删当前能力、错误运行旧编排、旧源码退休被误写成资源清理/验收成功，以及无授权地改变当前凭据/身份/进程边界。不扩大为新的统一启动器、资源回收器或安全框架。商业收费/外部准入/生产发布门禁不变；生产 **NOT_READY**，真实环境/用户验收 **NOT_RUN**。

## 3. 实际调用与闭包根因

| 对象 | 静态事实 / 调用路径 | 处置 |
| --- | --- | --- |
| `start-listingkit-local-api.ps1` | 顶部直接 throw 拒绝旧 API；其后仍有共享 YAML/config/env、Kubernetes Secret fallback、端口/PID/日志、旧 `cmd/product-listing-api` build/start/readiness 与 isolated path helper | 整源退休；它们全内联，不抽取到当前应用。专属 AST/Pester 测试一起退出 |
| `start-listingkit-local-dev.ps1` | 顶部直接 throw；不可达 body 仍停止端口/PID、启用 port-forward、调用旧 API 与 UI launcher，写 `.local/tmp` 日志/receipt | 整源退休；只删除自己的 orchestration，不删其通用 port-forward/UI 被调用者 |
| `image-agent-local-acceptance.ps1` | `start` 先 up 共享 ZITADEL 与专属 Postgres/Redis/Temporal/MinIO，创建 bucket、调用两个正常 schema 命令、写环境 marker，再调用已 fail-closed API；`provision/authorize` 先调用原生身份 CLI，再调用旧 API，后续 UI/worker 无法完成 | 全部六个 mode 与内联 helper 一起退出；不补接 current-application 来恢复旧验收 |
| 同一 orchestrator 的 `seed/status/stop` | seed 指向已不存在的 `internal/app/runtime/imageagentacceptance/cmd`；status 查询已记录服务/PID；stop 可停止自己的 API/UI/worker/Compose 和共享 ZITADEL，Reset 分支可 down -v 专属卷 | 专属源合同退出，既有资源与原所有者不动；本批不执行任何 mode（包括 status、WhatIf、stop 或 unsafe-input 测试） |
| 专属 Compose / 两种验收脚本测试 | 仅服务固定 `task-processor-image-agent-acceptance`；Pester 只测试旧 seed/authorize/env 规则；脚本 smoke 测试会直接执行 seed、stop 和 Reset WhatIf | 源一起退休，不执行其“自检”来制造 PASS；不误删被它们读取的共享 ZITADEL `.env.example` |
| 当前共享消费者 | UI launcher 自有当前 Next/Auth 初始化及 isolated 安全测试；`cmd/account-acceptance-fixture` 等直接用 `internal/zitadelprovision`；身份 CLI 的 multi-org 场景仍有独立说明；issue357 配置同一 token handoff | 全部保留，详见 §5.2；旧名/同目录不构成退休依据 |

查找覆盖 tracked scripts/cmd/internal/web/tests/deployments/.github/docs 的完整源名、旧 executable/seed、Compose project、token env 与身份 import。无 workflow 直接调用八源；调用方均已列入本闭包或历史说明。两个 launcher 的旧 helper 与 orchestrator 的文件/ACL/URL/Compose/进程等 helper 都内联，未找到被当前代码单独 import/dot-source 的共享 helper 文件。

**架构问题**：已有退休决定只在叶子 API/dev 入口加 throw，仍可执行的上游验收编排却在检查这一点前执行资源与身份副作用，手册也继续把链路写成可用。根因是切换没有收口到完整启动/验收图。修复是整退专属源与当前适用声明，保留当前原生 owner，不增强旧幂等/补偿、不新增兼容启动器或替代 seed。

## 4. 持久化、凭据、资源及副作用 owner

| 边界 | 旧源行为 / owner | 删除与保留边界 |
| --- | --- | --- |
| 私有环境与 PID/日志 | `.local/image-agent-acceptance` 下 compose/runtime/zitadel env、API/UI/worker PID/日志、user-token/管理 PAT；旧 API/dev 另有 `.local/tmp` 与 UI `.local-dev` | 只退 tracked 生成/消费源；不查找、读取、更新、归档或删实际文件，不扫描当前保留会话 |
| 专属 Docker 资源 | project `task-processor-image-agent-acceptance`；DB `image_agent_acceptance`、Redis/Temporal、MinIO bucket `listingkit-assets`、卷 `acceptance-postgres-data` / `acceptance-s3-data`；旧编排对 `listingkit_acceptance_environment` 执行 CREATE/DELETE/INSERT 写 marker，已退 seed 的消费者不再存在 | Compose source 删除不等于 compose down/stop、drop 或 bucket/volume 清理。实际标记表/行/文件/容器/卷仍归原资源 owner，状态 UNKNOWN；不新增 stop-only 兼容层或清理器 |
| 共享 ZITADEL / 原生身份 | 旧编排使用共享 ZITADEL Compose 与固定 local project；native CLI 创建/更新 application、grant/operator、多 org 等，并管理受保护 runtime 文件 | 删除上游旧编排不删除用户/Org/Project/grant/secret、轮换凭据或 revoke；CLI/库/私有文件 helper 与共享 Compose 原样，不重跑 provision/authorize |
| 当前 UI 与 token | Auth.js initial/refresh callback 调用 `persistZitadelAcceptanceToken`；专属旧 orchestrator 及当前 issue357 配置 `LISTINGKIT_ACCEPTANCE_TOKEN_FILE` | 当前 UI launcher、Auth callback、handoff helper/test、cookie/session/身份/refresh 均零改，不把 token 导出换为浏览器可见 session 字段，不读取现有 token |
| 当前应用/worker/schema | account-compose/私密 manifest/current-application、正常 schema CMD、ImageAgent/Temporal worker/图像与平台执行 owner | 全部保留。旧编排曾调用不意味着它们专属；不停止当前工作进程，不改 queue/SDK/许可/UNKNOWN/恢复/计费合同 |

保留真实资源的通常操作仍归当前 Docker/身份/应用 owner，未来维护须另外明确资源与授权；可通过本基线 Git 源追溯旧配置，不能据此执行旧恢复/Reset。现无真实资产调查或 cleanup 义务证据，不将未知物理态转成历史数据迁移、兼容义务或本批/试用新前置。若后来出现具体当前外部契约或 durable asset 的强制兼容证据，停止受影响部分并报告用户，不自行创建例外。

**当前 owner 的独立差异记录（不在本批修复）**：`scripts/issue357-runtime.mjs` 的 `startApplications` 配置 token env，并经 `scripts/issue357/serve.mjs` 传入 Next；`docs/engineering/issue357-real-provider-runtime.md` 则要求该 env 未设置、无 token 导出。尚未检查任何实际 run；不能宣称实际凭据已/未导出。本批对两者和 handoff 全部零改，不把此差异当作“最后消费者已消失”的证据，也不放宽原 no-export Must。分类为本退休批次之外的当前 owner 核对项，不新增其风险接受或试用前置；本批不签发该 runtime 的安全/产品验收。

## 5. 删除 / 保留 / 抽取矩阵

### 5.1 八个准确 tracked 删除源

```text
scripts/start-listingkit-local-api.ps1
scripts/start-listingkit-local-api.Tests.ps1
scripts/start-listingkit-local-dev.ps1
scripts/start-listingkit-local-dev.Tests.ps1
scripts/image-agent-local-acceptance.ps1
scripts/image-agent-local-acceptance.Tests.ps1
scripts/tests/image-agent-local-acceptance-script-test.ps1
deployments/docker/image-agent-acceptance/docker-compose.yml
```

不按 `listingkit` / `acceptance` / `image-agent` 关键词或目录批删，不删除实际数据目录/卷。专属脚本测试的旧路由、先 authorize 再启动、版本/env 升级和 Reset/source ownership 断言随该旧程序退休；不把它们重写成新验收平台。当前 owner 的同类 ACL/身份/隔离/安全断言见下一节，保持原样。

### 5.2 必须保留（包括没有代码改动的边界）

| 对象 | 当前依据 / 保留条件 |
| --- | --- |
| `scripts/start-listingkit-local-ui.ps1` 及 `.Tests.ps1` | 当前 Console/Next 启动能力与 isolated API URL/loopback、auth env、readiness、进程/路径边界仍有效。整个源/测试保留，含其现有 `IsolatedAcceptance` 选项与 helper；不以旧 orchestrator 退出为由删除独立手工能力或改现有 secret/PID 行为 |
| 通用 port-forward/watcher 与 DB/Temporal dev 工具 | `start-listingkit-local-portforward`/测试、watcher、remote-temporal、SHEIN RabbitMQ 转发、start/stop-temporal-dev、db-query 等原样。只退调用者；不删除参数化人工工具或调整其默认连接/停止语义 |
| `internal/zitadelprovision/**`（含 `cmd/**` 和全部测试） | 原生 account fixture/应用验收与独立 multi-org CLI 仍消费，provision/authorize/private file/loopback/角色/原子写入能力保持；不把整包标旧 Task，不退任何 CLI mode 或专属 path guard |
| `web/listingkit-ui/src/auth.config*`、`auth.cookies.test.mjs`、`lib/server/zitadel-acceptance-token*` 及 issue357 sources/tests | handoff 还有当前 source consumer；§4 差异不授权默改。全部 web/issue357 文件零改，包括防 secret 进入 session、初次/刷新行为、cookie 隔离和 ACL/reparse/atomic 写入测试 |
| `deployments/docker/zitadel/**`、`account-compose/**`、当前私密 manifest/运行入口、正常 schema 命令 | 当前共享身份/安装/数据 owner，不退 ZITADEL 网络/镜像/版本/env example，不退 `cmd/listingkit-schema-migrate` / `cmd/product-listing-api-schema-migrate` 或其正常 schema 行为，不接管 #692 |
| 此前全部当前能力与证据 | 当前定价/API/纯规则、ImageAgent/Temporal/AI/SKU/prompt/图像、采集/workerpool/queue/Retry、身份/tenant/资金/计费、Store 观察与独立 SHEIN 登录/Store 同步原样。dated 验证报告/历史行/计数与失败事实不改写 |

### 5.3 EXTRACT 结论

**无新增抽取。** 当前 UI/native identity/通用工具已有自己的 owner 和标准库/成熟组件，保留这些实现而非搬移旧 orchestrator helper；不为保住随机 secret、env/URL/ACL/PID helper 发明消费者。旧 Task seed/启动协调本身不再是当前产品要求。

```text
Legacy decision: RETIRE
Reusable behavior: 复用既有当前 UI、原生身份/凭据、ImageAgent/Temporal与通用工具 owner；无新增抽取
Current owner: Console/Auth.js、zitadelprovision/authruntime、account-compose/current-application、当前 worker/图像/采集/Store/独立SHEIN原样
Cutover/deletion condition: 用户确认本设计，#29/#301 Ready 双读回后，同PR整退八源与准确说明；原 guard/保留矩阵/原CI和一次最终独立检查完成，合并另行准确HEAD授权
```

## 6. 原 guard、自检与实施顺序

1. 当前仅设计：本文与四索引/清单提案说明；运行原 Greenfield/ArchitectureReadme guard、改动链接/diff 检查。**不执行** launcher、Pester/脚本 smoke、Compose config/up/down/stop、provision/authorize/seed/status/WhatIf、token helper、端口探测、当前/旧 runtime 或真实 SQL/HTTP/provider。
2. 用户确认后，将冻结设计、矩阵、当前 owner/不变量及验收写 #29/#301 当前正文，分别读回 Ready 再改源。一 Writer / 一主要 PR，同步一次实际 main，形成稳定候选前再核对一次，只处理真实组合；不常驻增加线程或机械拆 PR。
3. 精确迁移 `tests/listingkit_backend_retirement_test.go` 的 **`TestRetiredLocalLaunchersRejectBeforeOperations`**：旧两文件的 throw/firstFunction 顺序检查已变为退休实现细节，换成 §5.1 八个准确路径缺席；保留原 `start-listingkit-api-local-replay.ps1` 必须缺席断言。先因八源仍在 RED，再删除后 GREEN。源码缺席继续保证旧入口不能执行任何操作，不通过恢复文件/空壳 launcher 让测试过关。
4. 同文件其它 guard（含 #844 四源、当前 HTTP/auth/构建/平台/共享 Secret/Store/监控等）原样，CI/workflow 零改。当前 UI、身份 CLI/库、token/cookie 等生产和测试文件全部原样；不删除仍有效的安全断言，不为源退休执行实际 stop 测试。
5. 必要开发自检：上述原 guard，原 Greenfield/ArchitectureReadme；精确 22 文件 diff、保留路径/历史正文对照、仓库新增/改动相对链接、diff check。其它完整检查交原 PR CI；不本地重复整套 CI，不执行退休 Pester 来制造 PASS。若原 CI 暴露冲突，先给具体根因/最小变更；不能自行改变 workflow/inventory/保留边界或创建工具平台。

### 6.1 直接说明与旧保留结论（确认后实施）

本阶段只在四索引/清单写“待确认提案”；下表其余文件与原 guard 均未修改。历史正文保留，不执行旧计划中的 agent/sub-skill/部署/验收步骤。

| 准确文件 | 实施时处置 |
| --- | --- |
| `docs/development/image-agent-local-acceptance.md` | 只新增窄 SUPERSEDED 顶注，明确旧六-mode/Task API/seed/专属 Compose 运行指引不再可执行或作为试用前置；不新建替代验收。保留正文、原生 multi-org CLI 的独立授权/角色安全边界、手动真实验证 NOT_RUN 条件；不宣称底层身份 CLI、token handoff 或当前 ImageAgent 能力退休 |
| `docs/development/listingkit-local-debug.md` | 只追加窄顶注：旧 API/dev 源已删除、不能从历史命令启动当前应用；当前 UI/通用工具保留，现有 account-compose/私密 manifest 入口、#529 身份与发布边界及历史正文原样 |
| `docs/development/listingkit-local-prod-db-replay.md` | 仅校准现有历史顶注，把“launcher 在操作前拒绝”更新为源已删除；其余旧共库/端口覆盖正文保留为历史，不访问其中的旧绝对 `.env` 路径、不改为当前数据或恢复指引 |
| `docs/superpowers/specs/2026-08-30-image-agent-local-acceptance-design.md`、`2026-08-30-image-agent-acceptance-command-ownership-design.md` | 各只新增窄历史顶注：旧 Task local acceptance/seed/运行编排合同被本确认边界取代；当前领域 command ownership/原生身份/token/图像安全原则保持，旧 Proposed/Implementation Ready 不恢复已退入口。原正文/日期/失败证据不重写 |
| `docs/superpowers/plans/2026-08-30-image-agent-local-acceptance.md`、`2026-08-30-image-agent-local-acceptance-pr-hardening.md`、`2026-08-30-image-agent-acceptance-command-ownership.md` | 各只新增窄历史顶注，旧实施/测试/人工验收步骤不再是当前派工或执行授权；不运行其中 stop/reset/脚本测试或子代理指令。原计划正文保留 |
| `docs/architecture/README.md` | 索引本文和当前设计/实施状态；其它 owner/索引不改 |
| `docs/refactoring/legacy-register.md`、`module-target-mapping.md` | 顶注仅覆盖八个准确专属源与旧 launcher 保留结论；当前共享/native/worker owner 及其它历史矩阵/计数保留 |
| `docs/refactoring/code-health-decisions.md` | 只在顶注取代两项 API/dev Pester retained-generated-contract 与旧编排的当前适用性。原 rows/counts/失败证据保留；UI 测试、watcher 和其它工具保留结论及全部 consumer inventory 零改 |

不修改已有历史顶注的 SDS 排查记录、dated multi-org 计划/verification、其它旧验收报告、商业/生产/恢复政策、issue357 工程文档或其当前源。引用保留指南的历史链接仍存在；旧文档里的代码路径不是恢复源的命令，旧绝对工作区路径不作为仓库相对链接或真实文件访问目标。

## 7. 独立检查、交付与权限

原 GitHub 设计检查核对真实消费者、八源闭包、§5/§6/§6.1/§7、资源/凭据 owner、共享 UI/token/身份与 native worker 保留，以及物理资源 UNKNOWN 的边界。当前没有新增认证/持久化/副作用协议，不重新评审全局架构。同 SHA Running 不重复触发；不默认增加实现者或 AI 互验线程。finding 先分类，BLOCKER 须命中 AGENTS.md 的具体后果，最多两轮正常设计检查，不把非 Blocker hardening 变成新平台或试用前置。

独立设计结论和 CI 不等于用户确认。确认后在同一主要 PR 实施；完整闭包形成后一次最终独立交付检查真实完整 diff、保留路径/原有效断言和现有证据，修复只查相关增量。用户/指定独立验证者的实际验收不由实现者或 CI 代签。

交付为“旧启动/验收专属源与误导要求退出，当前 UI/身份/图像/采集/Store/独立 SHEIN 继续原入口”，无新增启动/验收要求。不执行任何旧/当前实例操作；所有真实环境、token/IAM/资源/provider/用户验收仍 NOT_RUN，生产 NOT_READY。

合并另需用户授权最终准确 HEAD，并正常 expected-head 合并、不强推/管理员绕过/改保护；之后核实实际 merge SHA 的 event=push/branch=main/completed/success CI。PR/设计 CI 不冒充 main。部署、实例/数据/凭据/旧资源清理、报告操作及 Issue 关闭均须独立授权；#29/#301不因本批自动关闭。
