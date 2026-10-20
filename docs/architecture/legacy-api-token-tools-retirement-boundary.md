# 旧 ListingKit API 取 token、保存与认证诊断链：完整退休设计

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

> Status: **DESIGNING / 待用户确认，Ready=false；仅设计，三个脚本和所有正式源码未修改。** Design Basis: **Reuse Existing Architecture**。调查基线为 main `b04459eab2366b5389d3219d2c242a04f63bccc4`（2026-10-20）。用户确认后，按 §5、§6、§6.1、§7 在同一主要 PR 整批实施；准确设计/实施 HEAD、原 CI、独立检查和授权证据维护在 PR，不把历史证据当本轮批准。

## 1. 用户结果、Authority 与范围

使用者是维护当前应用、内部试用、采集和 Store 的操作者。交付结果是仓库不再提供指向已退休 ListingKit API 的取 token、保存和认证诊断入口，也不再把这些旧命令或文件约定说明为当前操作路径。本维护没有正在阻塞的核心试用操作，不增加内部试用前置；三个脚本及专属尾部是一个退休闭包，不机械拆批。

产品依据为用户本轮明确范围、[Greenfield 产品基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[项目边界](project-boundaries.md)、[Legacy Register](../refactoring/legacy-register.md) 和 [Issue 驱动派工](../engineering/issue-driven-development.md)。复用 [#846 当前 UI/token handoff 保留边界](legacy-local-acceptance-closure-retirement-boundary.md)、[#842 原生身份及 Login V2 探针保留边界](legacy-yudao-identity-operations-retirement-boundary.md)、[#844 旧 Task 诊断退休边界](legacy-task-diagnostics-retirement-boundary.md)。不重新打开已退休 API/Task/Workspace，也不以旧文档或测试推导兼容义务。UI/IA/Figma N/A。

- Must：删除准确旧源闭包；保留 Auth.js handoff、原生 Login V2 探针及此前全部当前能力；凭据与真实副作用 owner 不变；不读现有凭据、不操作实例或数据；全部 workflow 和有效安全检查不变。
- Should：复用现有 absence guard 和索引位置，历史正文/结果原样保留；不新建替代取 token 工具或验收框架。
- Out of Scope：新认证/授权/租户协议、token 迁移/管理、IAM 清理、旧接口恢复、当前 UI/issue357/probe 改造、现有凭据文件/运行报告/进程环境读取、运行或 provider 操作、部署/生产验收、workflow/inventory/依赖/保护规则调整。
- Accepted Risks：不新增接受风险；实际旧 client、token、session、本地文件及外部人工消费者是否仍存在为 **UNKNOWN**。源码删除不能证明其不存在、失效、可删除或已经停止使用。

Threat Model 覆盖误删当前认证/同步能力、旧命令被误当当前入口、误触凭据/远端系统和伪造运行退出事实。理论 IAM hardening 或通用凭据治理不是本批前置；生产仍 **NOT_READY**，真实运行和用户验收 **NOT_RUN**。

## 2. Design Basis 与不变量

**Reuse Existing Architecture**：本批只退出已有退休合同的操作者源码，当前身份、Organization、Auth.js session/handoff、采集及 Store owner 已存在，不新增或改变领域/公共合同、事实 owner、schema、状态机、事务、恢复、授权/租户、计费/provider 准入或外部副作用语义。完整预计矩阵 14 个文件，不新增生产代码；不是以拆片规避架构阈值。

调用关系按 §3 退出，当前调用关系按 §4/§5 保持；无业务行为需要 EXTRACT。旧脚本的 dotenv、Bearer 前缀、文件 IO 和 HTTP helper 全部内联且仅服务旧入口，不搬入当前认证，不包装、不留 fallback、双读/双写或第二 token 事实源。成熟 OAuth/Auth.js 与原生身份能力继续由已有实现承担，不重复建设。

没有新增高风险边界，不要求重复全局 Architecture Review；适用独立检查只核对实际闭包、凭据/副作用 owner 和保留矩阵。若出现具体当前消费者、外部可观察合同或 durable 资产要求兼容，停止受影响部分并报告用户；不得自行恢复旧链。若确需改变上述架构边界，再按 AGENTS.md 升级准入。

## 3. 源码消费者与完整闭包

| 准确源 | 源合同与调用 | 最后消费者 / 专属尾部 |
| --- | --- | --- |
| `scripts/listingkit-fetch-machine-token.ps1` | 参数/进程环境/`.env` → issuer/token URL、client ID/secret、scope → OAuth `client_credentials` POST `/oauth/v2/token` → 显式 TokenFile 或 `.local/listingkit-api-token.txt`；默认调用 auth-check，可显式 SkipAuthCheck | 旧机身份取 token 和旧 API 验证同链；dotenv、Basic/Bearer、HTTP/写文件 helper 全内联，无独立专属 helper/test 文件 |
| `scripts/listingkit-save-token.ps1` | 安全输入 → 去 Bearer 前缀 → 同一旧 TokenFile；打印 auth-check 操作提示 | 是旧 API token 文件写入器，不是 Auth.js handoff；没有当前生产/CI 注入 |
| `scripts/listingkit-auth-check.ps1` | `LISTINGKIT_API_TOKEN` 优先，否则读取显式/默认旧 token 文件；BaseUrl 参数/`LISTINGKIT_API_BASE_URL`，默认 localhost:8085 → Bearer GET `/api/v1/listing-kits/settings-health` | 路由无当前生产注册，成功/失败打印与退出码只解释旧 API 认证；不改指向当前 API，不恢复 settings-health |

完整图为 `fetch → issuer POST → 旧 token 文件 → auth-check → 旧 settings-health`，以及 `save → 同一文件 → auth-check 指引`。统一采用下面的宽名称查询（包含 `settings-health`，不只完整 API 路径）：

```sh
git grep -l -I -E 'listingkit-(auth-check|fetch-machine-token|save-token)|listingkit-api-token|LISTINGKIT_MACHINE_|LISTINGKIT_API_TOKEN|settings-health' b04459eab2366b5389d3219d2c242a04f63bccc4 -- docs
git grep -l -I -E 'listingkit-(auth-check|fetch-machine-token|save-token)|listingkit-api-token|LISTINGKIT_MACHINE_|LISTINGKIT_API_TOKEN|settings-health' b04459eab2366b5389d3219d2c242a04f63bccc4 -- . ':!docs/**'
```

该基线结果为 **10 份 docs、非 docs 仅上述三个脚本**，没有当前源码、workflow 或构造器消费者。完整 API 路径的窄查询只命中6份 docs，不能用此数代替宽查询。§6.1 对 local-debug、两份 Aug12 指引、仍称 maintained operator tool 的 Aug8 回放说明，以及 June auth-blocked 记录末段未限定的未来操作推荐共5份加窄适用标注，原正文/结果全部保留；旧 Task 设计全文不改，另4份分类如下：

| 宽名称额外命中 | KEEP / 零改依据 |
| --- | --- |
| `docs/operations/listingkit-release-candidate-runbook.md` | 已有 #529/#301 HISTORICAL 顶注，明确旧 release/preflight/步骤不是当前准入；token 要求属于被覆盖的历史协议，不重复加顶注 |
| `docs/product/validation/runs/2026-06-21-shein-sds-batch-production-closure-regression.md` | settings-health-card.tsx 只出现在旧 npm lint 修复结果，无取 token/保存/认证工具操作路径；保留原 partial pass 与 BLOCKED |
| `docs/refactoring/httpapi-runtime-inventory.md` | 已有 DATED EVIDENCE 顶注，settings-health 是旧装配盘点，不是当前owner或执行入口；inventory及原计数完全不改 |
| `docs/superpowers/plans/2026-08-13-remove-default-shein-store.md` | 旧数字Task/默认Store feature计划里的端点消费列表，无本批三脚本或token约定；属于旧设计历史引用，不扩大为当前Store合同或执行授权 |

这是仓库静态证据，不证明没有外部人工消费者。搜索仅限 Git tracked 源，不读取真实 `.env`、token 文件、进程环境或现存运行资料。脚本没有专属 Pester、配置 schema、部署声明、workflow、跨文件 helper 或依赖需要连带删除。

**架构问题与根因**：旧 API 和启动/验收 owner 已退休，源工具和指引仍将旧本地 token 文件与 settings-health 绑定，形成误导性的“认证诊断”路径。根治是退出全部三个入口及直接适用声明，不修补旧 URL、不复用旧文件给当前 session，也不新建认证工具。

## 4. 凭据、持久化与平台副作用 owner

| 边界 | 既有 owner / 行为 | 删除边界与保留条件 |
| --- | --- | --- |
| issuer/client/secret/scope | 原生 ZITADEL 和既有身份管理员拥有 client/grant/签发/过期/撤销；旧 fetch 只是 OAuth 调用者 | 删除源调用者，不执行 POST，不撤销/轮换/导出/验证 client、secret、token 或 session；实际状态 UNKNOWN，无新 issuer 或授权协议 |
| 旧本地材料 | 操作者拥有 `.env`、进程变量、显式 TokenFile、`.local/listingkit-api-token.txt`；fetch/save 曾写文件，check 曾读文件 | 删除 tracked 读写代码；不定位、读取、删除、迁移、改权限、重命名真实材料或报告。专属 `LISTINGKIT_MACHINE_*` 仅退出此源，物理配置不动；共享 ZITADEL 配置名不能全局删除 |
| 旧 HTTP 诊断 | auth-check 向旧 settings-health 发 GET、解释响应；非当前认证事实 owner | 删除诊断源，不发请求/验证现有 token；当前 HTTP 身份校验、Organization/权限/租户验证与失败语义保持原 owner |
| 当前 Auth.js handoff | `web/listingkit-ui/src/auth.config.ts` 的初始/刷新 callback → `src/lib/server/zitadel-acceptance-token.ts` → `LISTINGKIT_ACCEPTANCE_TOKEN_FILE`；issue357 使用此独立约定 | 源/helper/tests、session/cookie、原文件/原子写/权限/安全合同及 issue357 runtime/env/serve 全保持；不改成读取旧 API token，不从 fetch/save 抽代码。已有 issue357 文档与源码差异不在本轮修复或真实验证 |
| Login V2 probe | `internal/listingkit/phoneonboardingpreflight/**` 与 `hack/debug/listingkit-phone-onboarding-preflight/**` 拥有显式 non-production、有限超时、临时 org/技术用户、SMS/session proof 和 cleanup；独立 provision/login token 输入 | 完整源码与 fake 测试保留，不能因 listingkit/preflight 命名误删。探针不执行，不创建/清理身份、组织、SMS、session，不读取探针凭据 |
| 当前数据/平台/worker | 当前应用/身份、平台 API/规则、采集、Store 及独立 SHEIN 登录/同步的原 domain/repository/worker owner | 表、账户/Profile/Cookie、attempt、缓存/队列、provider、retry/UNKNOWN/recovery、资金/计费/权限/幂等/副作用均不变；不操作实例/数据，不接管其它 owner |

源码退休不构成 token 撤销或物理数据生命周期变更。没有新持久化、事务或自动恢复者；失败、响应丢失、重试、重启、取消和并发责任仍由原 owner 承担。当前采集、Store 观察和独立 SHEIN 登录/Store 同步不是这三个脚本的消费者，原入口和启动方式不增加新步骤。

## 5. 删除 / 保留 / 抽取矩阵（确认后整批实施）

### 5.1 DELETE：三个准确 tracked 源

```text
scripts/listingkit-fetch-machine-token.ps1
scripts/listingkit-save-token.ps1
scripts/listingkit-auth-check.ps1
```

三个入口和其全部内联 helper 一起删除，不留抛错壳、空文件、alias、替代取 token 命令或新 endpoint。没有业务 EXTRACT，也不删除任何实际 token/配置/数据。

### 5.2 EDIT：现有守卫与十份文档

```text
tests/listingkit_backend_retirement_test.go
docs/architecture/legacy-api-token-tools-retirement-boundary.md
docs/architecture/README.md
docs/refactoring/legacy-register.md
docs/refactoring/module-target-mapping.md
docs/refactoring/code-health-decisions.md
docs/development/listingkit-local-debug.md
docs/superpowers/plans/2026-08-12-1688-runtime-acceptance.md
docs/superpowers/specs/2026-08-12-1688-runtime-acceptance-design.md
docs/product/validation/2026-08-08-1688-controlled-replay.md
docs/product/validation/runs/2026-06-21-shein-sds-batch-production-closure-auth-blocked.md
```

共 14 个准确路径：3 DELETE + 1 原 test owner + 10 docs；只有这一个主要 PR。设计阶段只新设计和四索引共五文档；后五份操作说明与 test owner 留到用户确认后。变更含义与历史保护见 §6/§6.1，不能借此扩大到同目录其它源。

### 5.3 KEEP：当前能力和历史证据

| 保留对象 | 原 owner / 不变范围 |
| --- | --- |
| Auth.js callback/helper/tests、`scripts/issue357-runtime.mjs`、`scripts/issue357/**` | 当前 handoff 和 issue357 既有合同；当前 UI launcher/tests、原生身份 CLI/库及 schema 工具全部保留 |
| Login V2 两个 probe 包、当前 ZITADEL adapter/装配、Organization/成员/角色/手机号 onboarding | 不是 Yudao 导入/恢复或旧 API token 工具；共享 ZITADEL Compose/Kubernetes、Secret/配置/示例和通用身份工具保持 |
| 当前采集/workerpool、通用 queue/Retry、平台 API/DTO/有效规则、两种 SHEIN 属性、AI/SKU/spec/prompt/图像/定价 | 仅以当前实际消费者解释此前保留能力；不复活后续已批准退休的旧实现，当前 PriceManager/PriceCalculator 等有效 owner 与安全断言不变 |
| Store 观察同步、`internal/listingadmin` 有效仓库、`internal/listingruntime` 共享值及独立 SHEIN 登录 worker 和 Store 同步 | 当前 owner、登录/健康/关闭、数据、账户/Profile/Cookie、Store 仓库与同步合同保持；不引入 Organization bridge 或第二事实源 |
| `docs/architecture/legacy-task-diagnostics-retirement-boundary.md` | §4 旧 token/清单行描述已退出 #844 的历史消费者，不是当前脚本保留要求；全文不改 |
| Aug8 replay 与 June auth-blocked 混合历史/操作说明（§5.2 准确路径） | 原历史执行/失败/阻塞、token 用法与占位示例正文全部不改，只按 §6.1 加窄适用顶注；不读取其引用的真实工件或重新执行 |
| §3 宽查询的另外四份历史/偶然文档命中 | 逐项 KEEP / 零改，理由见 §3；不按端点名称批改文档，不修改原运行证据或 inventory |
| 全部 workflow、必需和安全检查、inventory/历史计数/ceilings、依赖/锁、保护规则与其它当前能力 | 零改；通用 DB/Temporal/prompt replay/port-forward、共享存储与凭据保留，不按名称批删 |

```text
Legacy decision: RETIRE
Reusable behavior: 当前 OAuth/Auth.js/native identity/handoff 已有有效 owner；旧三脚本无须抽取
Current owner: 当前 Console/Auth.js、原生 ZITADEL/HTTP 身份与 Organization、Store/独立 SHEIN worker 原 owner
Cutover/deletion condition: 用户确认冻结设计、双 Issue Ready 读回后，整批准确源与专属说明退出；原检查/最终独立检查通过，合并另按准确 HEAD 授权
```

## 6. 必要验证与证据边界

确认后在现有 `TestRetiredLocalLaunchersRejectBeforeOperations` 的 absence 列表增加三个准确路径，不改原八项与 replay 断言。三源仍在时先 RED（只检查源存在，不运行脚本），删源后 GREEN；不为缺文件 skip、不增加新 runner/全局检查器，不改有效 current/security/worker/platform 断言。tracked 引用复查必须只剩新增 absence、明确历史引用及本设计说明，没有当前执行者或新读取源。

保留源码及所有 workflow/inventory/依赖逐项核对零 diff。开发自检只需该原 guard、适用 Greenfield/Architecture README 文档检查与 diff/新增链接/历史正文保护；稳定候选跑原未修改 PR CI，不人工重复完整 CI。当前 Auth.js/probe/worker 的既有检查按原 CI 保留，不为了未变源码添加镜像测试或运行验收。所有真实 HTTP/token/probe/PowerShell 工具执行、Docker/Compose/kubectl、SQL、SMS/provider 和既有凭据/环境/报告操作均 NOT_RUN。

### 6.1 文档尾部与两阶段边界

| 准确文档 | 设计阶段 | 用户确认后的实施阶段 |
| --- | --- | --- |
| 本文、Architecture README、Legacy Register、Module Target Mapping、Code Health Decisions | 本文写拟议矩阵；四索引仅增加待确认顶注/入口，不宣称已删、Ready 或验收通过；原索引正文不动 | 更新同一新增入口/顶注及本文状态为用户确认/实际源码退出；准确滚动 HEAD/CI/检查留 PR，原历史正文/计数/失败继续不动 |
| `docs/development/listingkit-local-debug.md` | 零改 | 仅加窄顶注，明确三个 token 脚本及旧 API 文件/环境约定已退出、以下命令为历史；保留 Auth.js handoff/原生身份和原全部正文，不改写旧示例为新命令 |
| Aug12 旧 1688 runtime acceptance plan/spec（§5.2 两个准确路径） | 零改 | 仅加窄顶注，说明旧验收源和 token 约定不再是当前执行/试用要求，不能按旧步骤恢复接口/脚本；保留原模式/安全限制/历史正文与结果。当前采集能力不因旧验收工具退出而退休 |
| `docs/product/validation/2026-08-08-1688-controlled-replay.md` | 零改 | 仅加窄顶注，取代 Runtime acceptance tool 段仍称 maintained operator tool 及当前 token 文件/验收路径的适用表述；原 replay 结果、限制、模式/安全说明与全文历史证据不改，不重跑或补造 PASS |

| `docs/product/validation/runs/2026-06-21-shein-sds-batch-production-closure-auth-blocked.md` | 零改 | 仅加窄顶注，明确末段取 token/保存/诊断的推荐流程、未来验收文件复用与机器调试建议已失效；保留原始 run 证据、失败/UNKNOWN、所有正文及占位示例，不改变当前浏览器用户/机器身份语义 |

历史保留不授权执行旧命令。除上列精确顶注外，其它旧设计、dated 验证记录、issue357 文档和通用 Prompt/凭据/存储语义不改；不重写历史结论为 PASS、不改源码引用数量来制造全仓清零。

## 7. 检查、交付与权限停止点

设计提交后按原 GitHub 检查核对三个根入口、完整引用、§5/§6/§6.1/§7 与当前保留能力；不增加 Writer，不重复触发同 SHA Running 复核，不新建验收工具。finding 先按 AGENTS.md 分类，BLOCKER 必须指出具体当前 Must 与阻塞后果；非 Blocker hardening 不扩架构或内部试用前置。当前为 Reuse Existing Architecture，不为形式引入全局架构评审。

**本轮只交设计**。用户确认后，把当前设计分类、冻结依据和范围写入 #29/#301，并读回两处 Ready，再在同一 PR 实施 §5/§6/§6.1 的完整 14 路径。实现结束交一次最终独立交付检查，核对真实 diff、保留路径与现有证据；不能以作者自检/绿色 CI 代替用户实际验收。用户未确认前不得修改/删除三脚本、guard 或五份操作说明。

合并仍须用户单独按最终准确 HEAD 授权并使用 expected-head 正常保护语义；获准合并后核对实际 merge SHA 的 `push/main` CI，不能用 PR CI 或上一批 main CI 代替。当前无合并、部署、Issue 关闭、凭据/实例/数据/IAM/真实平台操作授权。父目标保持 OPEN，生产 NOT_READY，真实运行/用户验收 NOT_RUN；本设计及未来源码退休不增加试用前置。
