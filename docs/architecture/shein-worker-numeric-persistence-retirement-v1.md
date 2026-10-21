# #301 独立 SHEIN worker：旧数字账号、Store 与 Profile/Redis 完整退休闭包

> **Implementation admission (2026-10-21): Independent Architecture / IMPLEMENTATION_READY / FROZEN.** The user confirmed §1–§7, including the authorized CLI, optional Store execution-ownership table and §6.1, and authorized the complete batch in #863. Frozen design HEAD: `94fa4cab189650aaee770225657a35e784aeaeb0`; [independent design review](https://github.com/qq550723504/task-processor/pull/863#issuecomment-6306296890) completed without remaining findings. [Implementation receipt](https://github.com/qq550723504/task-processor/pull/863#issuecomment-6306911694). The proposal/pending language below records the design-stage history and is superseded only by this explicit approval. One writer implements this complete closure in the same PR. Task-owned temporary PostgreSQL/Redis and synthetic fixtures are authorized; existing instances/data/credentials, workflows and trial prerequisites are unchanged. UNKNOWN never permits automatic takeover. #301 stays OPEN; merge, deployment and runtime acceptance remain separately authorized.


Refs [#301](https://github.com/qq550723504/task-processor/issues/301)。

Status: **PROPOSED / DESIGN_ONLY / 待用户确认**。Design Basis: **Independent Architecture**；生产实施 **NOT_READY / NOT_AUTHORIZED**。调查基线为 `main e174977d35fe779e966c29018ac01889747e8887`（2026-10-21）。用户本轮授权完整设计，不授权实现、真实环境操作、合并或关单；适用独立检查与用户确认完成后才冻结为 IMPLEMENTATION_READY。

## 1. 用户结果、产品决定与范围

使用者是独立 SHEIN 登录及 Store 同步的维护者。一个 Delivery Batch 保留独立进程的登录、强制登录、验证码、取消、Cookie、attempt、重启 interrupted、health/shutdown 和登录后 Store 身份同步，退出旧数字账号/Store 仓库及其专属接线。不按账号、Store、Redis、Profile、测试或配置机械拆批。

本设计供用户确认以下**尚未生效的产品增量**：

1. 独立 worker 使用**正常创建的当前 Organization + 当前 Store UUID**；每个 Store 最多一个部署配置的网页登录 binding。账号凭据来自任务/环境 owner 新提供的私有配置引用，不扩展 1688 SourceAccountRegistry，不新增账号页面、账号注册表或凭据 vault。
2. 登录后识别的 merchant 只经 **Store Center** 的有权、幂等窄命令同步至当前 Store；网页登录不构成官方连接、付费服务激活或发布权限。重复/不匹配阻止本 binding 的同步及后续使用，不自动禁用当前 Store、另一店铺或 paid entitlement。
3. 新 worker 只消费新 namespace 和新 Profile root；不续跑旧数字行、Redis attempt、Cookie 或 Profile。源码退休不要求删除、迁移或停止既有资源；这些资源完全不在本轮权限内。
4. 同一个独立二进制增加有限 CLI producer，使登录/验证码/取消/状态及binding activate/quiesce真正可达；沿当前身份与Store授权，不恢复旧HTTP，也不是新增验收工具。Store owner增加一个可选执行归属控制表，跨installation切换必须等旧owner关闭确认；此必要入口/持久化增量随整份设计待用户确认。

Authority：[Greenfield](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[当前项目边界](project-boundaries.md)、[Store current application](store-center-current-application-v1.md)、[Store 平台观察](store-center-platform-observations-v1.md)、[原 worker 身份边界](shein-worker-identity-persistence-boundary.md)、[#854 源级收口](legacy-retirement-parent-closure-boundary.md)、[Issue 规则](../engineering/issue-driven-development.md)。当前身份和成员资格继续归原 ZITADEL/Organization/授权 owner。**本提案只在用户确认后替代旧 worker 文档 §1/§3/§4/§6/§7 及收口文档 §4.1 对准确数字实现的保留，不取消其安全行为。**

Must：独立登录/Store 同步能力保留；明确唯一事实 owner；同组织授权、撤权、取消、同键/异载荷、终态和迟到响应保护；Store 付费/官方连接不受网页登录影响；无旧 ID 映射/兼容/双读写；当前采集、Store 观察同步、图像、平台 API/规则及其它保留能力不变。

Out of Scope：来源采集功能或真实验收、账号/Store UI、官方 onboarding/发布、支付、新 IAM/调度/恢复平台、生产准备、旧数据或凭据读取/迁移/清理、实例启停、浏览器/provider 调用、workflow/inventory/依赖/锁/保护规则改变。#29 不重开，#301 继续 OPEN；本轮不增加内部试用前置。没有新增 Accepted Risk；网页挑战、正常外部限流等沿已有边界。Figma N/A：没有新增页面、导航、HTTP 登录入口或页面交互布局；有限CLI产品行为在§3.1单独明确。

## 2. Design Basis、调查证据和实际消费者

**Independent Architecture**：涉及数字身份→当前不透明组织/Store 身份、凭据输入 owner、Redis/Profile 命名与恢复、Store 同步事务和权限。30+ 相关文件只是风险提示，不是拆 PR 理由；不能复用原 §7 的 Reuse Existing Architecture 准入。设计/索引可以先交付；正式生产路径和 schema 不得先写后补依据。

以下是对准确基线源码的只读调查，非运行验收：

| 实际消费者 / 调用链 | 当前事实和风险 | 本闭包处置 |
| --- | --- | --- |
| `cmd/shein-login-worker` → `app/runtime/sheinloginworker/runtime.go` → `sheinlogin/bootstrap/build.go` | 直接打开全局 `config.Database`；构造 `listingadmin.NewGormStoreRepository/NewGormStoreAPI`；只启动 health HTTP，生产代码没有enqueue/verify/cancel producer | 保留命令、独立进程和 health；同一二进制增加§3.1有限CLI，不把Service port当可达入口；显式当前Store/授权配置，不复活旧HTTP |
| `sheinlogin/accounts.go` → `ListingAdminAccountProvider` → `listingadmin.StoreRepository` | 从 `listing_store` 的数字 tenant/row ID、用户名、密码、代理和 URL 形成账号；5 秒缓存按旧角色/用户隔离 | 数字 provider/cache/DTO 退出；按当前组织/Store binding 精确读取，不缓存活授权或把配置当授权 |
| `sheinlogin/service.go`、`types.go`、`runtime.go`、`worker.go` | Account、attempt、Cookie、session、锁均以两项 int64 为 scope；worker 自行重读凭据；验证码会话只属于 worker | 有效服务/并发/状态行为原位抽取为当前 scope；不把 int64 改名成 Organization，不保留数字 overload |
| `sheinloginmanaged/bridge.go` → StoreAPI page/get/update | 同 tenant 分页查重、空 merchant 回填、不一致/重复禁用旧行；完整页失败关闭是 #722 有效安全要求 | 替换为当前 Store 的单事务命令；退出旧 DTO、factory 和分页接线，不复制另一 Store 仓库 |
| `sheinlogin/store.go` | `shein:cookie:<tenant>:<row>`、全局 `shein:login_attempts`、Lua/stream/控制键；登录成功原子写 Cookie/终态，之后 Store 同步独立失败；还删除 `listing:task:pause:*` | 复用有效 Redis 原子算法，换新完整 scope；退休旧 key/parser/pause cleanup；明确成功与同步结果分别记录 |
| `sheinlogin/profile_runtime.go`、automation 系列 | `<root>/<tenant>/<row>`、缓存修剪、锁文件清理、按命令行子串查找并终止浏览器 | 新 root/当前 scope；保留浏览器启动/关闭及 owner 内缓存处理，退出宽进程扫描/解锁接线 |
| `sheinlogin/service.go` → `shein/client` | `StoreConfig`、CookieProvider、全局 LocalLoginRefresher 仍是数字 store/tenant；supplier/warehouse HTTP 使用其 cookie | worker 改消费当前 scope 的显式 transport/session；保留平台 endpoint/解析/HTTP/TLS，不给旧数字 wrapper 再加新消费者 |
| `temu/api/client/{client,interfaces,cookie_manager,auth_pause_handler,manager}.go` | 唯一非 SHEIN 的 listingadmin import 尾部：StoreRuntime、Cookie/暂停/MallID 回写。全仓只有接口和其包内调用，没有生产 GetStoreAPI 实现或包外 manager 构造 | 一并退休这条无注入接线；保留 ClientAPI、有效请求/响应、HTTP/TLS/错误检测等平台合同；不得新造 DTO facade 延长 StoreAPI 生命 |
| `sourceaccountregistry/domain.go` | 平台仅 `1688`，状态仅 `pending_connection`，无网页登录凭据/会话 | 完全保留，不把登记当 SHEIN 登录或复制凭据 |
| `storecenter/{store,member_access,gorm_repository,audit,current_schema}.go` | 当前 Store UUID、成员访问、事务、版本、`identity_key` 和 audit；唯一性为组织内 platform + region + external ID | 复用当前 owner 与唯一索引；不对其它地区或组织建立新 global merchant owner |
| `.github/workflows/ci.yml` / `commercial_readiness_workflow_test.go` | 固定运行 `go test -race ./internal/listingadmin -count=1`；当前该包生产文件 11、测试 6 | workflow 零改；该路径转为只含当前 Store/worker 并发合同的测试，不留生产壳、不减少断言，见 §6.1 |

补充直接证据：`rg` 的三个旧包生产 import 只落在 SHEIN accounts/service/bootstrap/runtime 与上述 TEMU 两 import 文件；`go list` 读取这些包和原 `tests` 成功（不启动构造器或服务）。旧 root/tenantbridge 已退休，当前 `current-application` 不是独立网页登录 consumer。包外调用与 constructor 必须在实施开工前再次精确核对，新增真实 consumer 先报告，不隐藏进删除范围。

## 3. 目标 owner 与端到端接线

### 3.1 身份与账号配置

- **Store identity**：当前 Store Center 持有 Organization、Store UUID、platform/region、external merchant、record/service/official connection；新账号在正常当前流程新建，旧行 ID 不导入、不转换、不关联。
- **Login binding**：现有独立 worker owner 持有有界只读部署配置，键为 `(OrganizationID, StoreID)`，值含配置 revision、登录 URL、私有 credential file 引用、代理引用和 browser 参数。最多 100 个显式 binding；拒绝重复 scope、未知字段、缺失 revision、非 SHEIN/删除/disabled Store。无需新账号 DB 表/CRUD/第二 Store 注册表。一个 binding 对应一个当前 Store；更换账号必须显式更新 revision，旧任务不接用新凭据。
- **Credentials**：环境 owner 新提供的私有文件/secret mount；现有正常配置加载方式和 OS/挂载保护复用，不开发 vault/取 token 工具。绝对路径、非仓库公共文件、有限尺寸；启动不打印内容，读取失败不 fallback。配置中的 org/actor/role/Store 字符串从来不是 verified authority。
- **Access**：服务命令入口消费当前 verified identity 和已有 Organization/live-write、StoreMemberAuthorizer/成员 grant。每次 enqueue、控制、worker 执行和 Store commit 重新验证相同 actor/member/org/store；Store grant/组织撤权或 binding revision 改变即拒绝。没有当前授权依赖时 worker 不就绪，不以固定 operator、配置布尔值、Redis载荷或缓存代替授权。runtime 只装配既有当前授权 client/port，不新建 IAM。
- **重启后的活授权依据**：复用 `internal/app/worker/imageagent/organization_scope.go` 已有的 ZITADEL `AuthorizationClient.ReadExactServiceProjectAuthorization` + 私有服务 token + 当前角色策略的调用形态，核对持久化的 actor/member/org仍是同一 active grant，并沿现有 OrganizationBusinessStatusChecker拒绝停用组织；应用层针对 `PermissionWorkbenchStoreLifecycle` 装配该现有能力，并由 StoreMemberAuthorizer再次核对当前成员和目标Store。只复用当前授权算法/接口，不引入 ImageAgent、Run、AI权限或其DB依赖，不保存/续用用户旧JWT，不仅凭Redis字段重建“已验证身份”。服务 token 由既有私有配置正常提供，本轮不申请、读取或创建凭据。
- **对外入口**：同一 `shein-login-worker -config <private-current-config>` 二进制默认 `-action serve` 保留原health/RunWorker；新增 `login/status/verify/cancel/activate/quiesce` 六个有限CLI动作，调用当前Service/Store命令port。producer模式不启动worker循环、HTTP或浏览器，不调用旧inline Login/ForceLogin，不直接教操作者写Redis。无当前Console登录页/新HTTP服务。

**正常CLI合同（拟议，设计确认后才实现）**：六动作均需要 `-organization <opaque ID> -store <current UUID> -token-file <private absolute file>`；复用当前JWT verifier验证签名/issuer/audience/expiry，再用上述当前grant与Store成员授权，不信任flags里的组织声明。login携带明确 `-request-key <UUID>`，只有显式 `-force-login` 才强制登录；返回attempt UUID和状态，不等到完成、不自动重试。status按原attempt UUID只读（binding状态由授权后的Store返回），无自动登录/Store同步；verify/cancel必须给原attempt UUID，错scope/已终态明确拒绝。activate/quiesce均携带独立request-key，按§4.1控制表事务处理；activate不能略过旧owner的quiesced确认。验证码仅stdin输入，不进argv、日志或JSON结果；token-file是调用者经现有正常身份流程私下提供的新有效token，本轮不新增取token脚本、不创建/读取任何真实token。

CLI的安全JSON只返回scope/attempt ID、登录状态、sync状态和通用错误；不输出密码、Cookie、验证码或raw provider/artifact。超时/响应丢失返回UNKNOWN并提示沿**同request-key/原attempt**查询，不能自动生成新key。verify/cancel的当前成员与actor必须与原attempt一致；status也先授权再读。producer读取有界binding元数据和当前Store，不读登录密码；仅worker执行时读取新私有credential引用。用户可在登录待码时另起相同二进制提交验证码，worker继续持有原浏览器。这是已有保留能力的操作入口，未增加测试session、runner、部署或内部试用门禁。

拟议接线（箭头为组合，不是新增事实 owner）：

```text
cmd/shein-login-worker → 当前 worker runtime / 显式配置
  → 既有当前身份/Organization + Store member 授权
  → 当前 Store reader + 私有 login-binding reader
  → sheinlogin 的 scope/attempt/browser/session 有效行为
  → 显式新 Redis state + 新 Profile root
  → SHEIN supplier/warehouse 窄 HTTP transport（只读；无隐式重登录）
  → Store Center RecordSheinLoginIdentity（本地单事务/原 audit receipt）
```

### 3.2 Store 同步与副作用

新增**当前 Store owner 的窄内部命令** `RecordSheinLoginIdentity`，不是旧 StoreAPI adapter。输入精确 `(org, actor/member, store UUID, expected Store version, binding revision, attempt UUID, observed external merchant)`；拒绝 client 自带 Cookie、官方 token、status、paid service 或其它 Store 字段。实际 merchant 来自当前授权 worker 的成功会话读取，外部 merchant 数值类型可以保留，不作为本地身份。

复用 `MemberScopedStoreRepository` 的 live authorizer、成员 grant 锁、当前 Store 行/版本、`identityKey`/组织内唯一索引和 GORM 本地事务。事务内：核对当前 active SHEIN Store/region、版本和原值；原值为空时设 external ID 并更新 identity_key/version/updated metadata，原值相同作幂等确认；不同或唯一冲突记录安全的 blocked 结果，不重绑、不禁用其它 Store。当前唯一性按组织+平台+region，不搬旧分页 DTO，不泄露冲突行的信息。

命令以原 `workbench_store_audit_logs` 的 `(org, attempt UUID, 新的 login_identity_sync action)` 作为操作回执，payload fingerprint 包含 Store、binding revision、expected version 和识别结果；成功/blocked 结果与 Store 修改在**同一借用事务**记录。复用原 audit 字段，不新建通用回执表或状态平台；新增 action/失败码和严格校验属于本批领域合同变化。相同键/载荷重读原安全回执及结果版本；不同载荷 conflict；后来的 Store 改动不被原重放改回。审计回执不得包含密码、Cookie、验证码或任意 provider body。目标Store/grant加锁后，在更新merchant前设GORM/PostgreSQL savepoint；唯一冲突先回滚到该savepoint，再于仍持有目标锁的外层事务记录blocked回执，不能在已经aborted的事务里继续写audit。无新跨库事务框架。

**明确的行为变化，待用户确认**：旧“禁用 listing_store 行”退出；冲突只阻止本 login binding 的同步/使用并显示 `sync_blocked`。当前 Store 的 record/service/官方连接生命周期及已购买期限不因网页登录失败改变。网页登录成功和 Store 同步成功分别可观察，不将前者报作后者，不自动使官方 Connection ready，不把 Cookie接入当前官方 Observation/Submission。

不新增账号库/业务 DB；Store/Audit仍唯一 owner，另有§4.1的**一个可选Store登录控制表**。worker 使用显式 Store 连接与最小角色：读取当前 Store/成员 grant；仅允许本命令需要的 external_store_id/identity_key/version/updated 列、audit SELECT/INSERT和该控制表的有界读写，不能更新 service、金额、connection_ref 或执行 DDL/DELETE。角色与精确校验在原 Store schema init owner 增量定义，只对新建/显式配置目标安装；不操作既有 DB 或替 #692 执行安装。

## 4. Redis、Profile、状态与恢复边界变化

### 4.1 新资源身份，不接旧状态

| 持久化对象 | 旧 owner / 格式 | 拟议当前 owner / 变化 |
| --- | --- | --- |
| 账号及凭据 | 数字 `listing_store` 行/明文字段、repository/cache | 当前 Store 是业务身份；worker 私有 binding 配置 + 新 credential 引用是部署输入，无旧行读取/复制/迁移 |
| Store merchant | 旧行 StoreID/status | 当前 `workbench_stores` external_store_id/identity_key + 同事务 audit receipt；付费/官方字段不变 |
| 执行归属（本提案新增事实） | 旧全局Redis namespace内的active attempt，没有跨installation归属 | 当前Store owner可选 `workbench_store_shein_login_bindings` 控制表；同org/Store唯一installation/generation与active-owner，原audit保留activate/quiesce操作回执；不存账号/密码/Cookie |
| Cookie/attempt/验证码/控制/失败 | `shein:*` 数字键和全局 stream | worker 原 Redis owner 的全新 `shein-login:v1:<installation UUID>:` namespace；所有 scope/值都验证 org+Store UUID+binding revision；Redis仍是此 worker attempt/Cookie事实源，不搬到 Product/Store |
| Profile/artifact | root/数字tenant/数字row；宽进程扫描 | 显式**新专属绝对 root**；`v1/<sha256(org)>/<Store UUID>/<binding revision>`，hash仅安全路径编码，不是身份映射；禁止旧root/路径导入和任意客户端相对路径 |
| 旧 task pause | 登录成功清 `listing:task:pause:*` | 源码与所有调用一起退休；新 worker 不读取/删除这些旧 Task 键 |
| worker/共享配置 | 全局 Database；`EffectiveSheinCookieRedis` 缺省到全局Redis；共用SDS LoginServiceConfig | worker 新显式严格配置；Store DB/Redis/root必须指明且启动核验。SDS/采集等仍有消费者的共享配置保持，不能删除共用字段 |

禁止旧 DB/Redis/Profile 的扫描、启动自动探测、rename/copy/backfill、双读写、旧键 fallback 或“当前 Store UUID→旧数字 ID”表。配置缺失失败关闭。新 namespace 和安装 ID 不单独证明资源归属；配置及部署 owner 必须明确指定新专属端点/root，服务只核对自己的标记和准确 scope，不扫共享 key。Redis 使用原 standalone go-redis/Lua/stream模型；本轮不引入 Cluster、分片或新队列系统。

配置/命令解码拒绝未知字段和尾随内容：当前组织/actor/member沿原200字节上限，Store/attempt/installation/binding revision为canonical UUID；配置文件最多1 MiB、单credential文件最多64 KiB、单命令最多64 KiB。新root/credential路径只能来自可信部署配置，不能来自队列或用户请求；binding revision不能含路径片段。沿原有限并发、请求deadline及验证码/Cookie TTL上限，不因换owner取消限额。失败摘要/公共结果继续有界且不含secret，原raw network/body/私有artifact不得进入Store audit、队列或普通日志；artifact只在该worker新专属root下保留。

**跨installation执行归属**：新namespace不是排他保证。当前Store owner按 `(organization_id, store_id)` 唯一行保存installation UUID、binding revision、非secret配置fingerprint、单调generation、`active/quiescing/quiesced`、active attempt UUID/worker owner nonce及关闭确认；有当前Store的组合FK，不存凭据或第二份merchant。`activate`在live授权/Store+grant锁和该行锁下首次认领；同key/配置重放，异载荷冲突。不同installation/revision只能在旧行quiesced且active attempt已关闭确认后，以比较旧generation的同事务activate递增generation；操作回执复用原audit。旧namespace/manifest不能覆盖当前行或再次认领旧generation。

worker在任何网页登录、自动验证码、携带Cookie的supplier/warehouse请求前，先在Store事务中把唯一active attempt/owner nonce认领到当前generation，再执行外部动作；Cookie提交、控制与Store同步前再次验证generation/owner。active锁在Store表跨namespace唯一，Redis原Lua只负责同一有效generation内的attempt原子性。generation必须进入attempt/Cookie/profile元数据；不能把失配解释为缺Cookie并重新登录。producer只向当前active generation提交；原key重放不得绕过当前活授权。

`quiesce`事务先封住新认领，现有唯一owner收到控制后停止浏览器/验证码/HTTP操作并完成自有context/进程关闭，最后携精确owner nonce记录quiesced确认；没有active owner时可直接确认。必须在会话关闭完成后才清active/写确认，不能在登录终态或ACK时提前释放。新activation在确认前始终拒绝；TTL、心跳丢失、45秒reclaim、连接断开都不能证明旧浏览器停止。无法取得关闭证明的崩溃/UNKNOWN保持fenced，交原环境owner在另行授权的准确资源范围排除旧执行者后处置，本批不提供force-takeover、自动kill或跳过确认命令。不同generation的原sync重放只可读既有回执，不能产生新的Store写入。

这是专用于当前Store登录归属的一张表和现有命令事务，不建通用lease/IAM/调度平台。表/角色由原Store初始化命令的**显式可选worker能力**安装；普通Store core schema校验及内部试用不要求这张表，只识别已登记的可选模块存在，不放宽未知对象/权限检查。新worker必须验证此能力存在和最小权限；设计阶段零DDL。该fence只约束新当前scope，不能冒称已停止/映射任何旧数字worker或实际资源。真正安装/切换仍另行授权，本设计不执行。

Profile 只处理当前 worker 创建、标记且在新root以内的目录；使用原 Playwright persistent context/session Close 和进程生命周期。profile busy 时失败关闭，不扫描并杀死全机浏览器、不删除无法证明拥有的锁。缓存修剪只在已关闭的自有 Profile 内按有限相对清单进行，不递归触碰 root 外路径。重启不重建验证码浏览器；旧缓存/登录态不承接。

### 4.2 保留登录状态机，分清 Store 同步

原登录状态 `queued → launching → waiting_verify_code → succeeded|failed|cancelled|interrupted` 与终态/取消优先保护继续复用。凭据不入队列；attempt记录完整scope、actor/member、binding revision、请求fingerprint、时间与不含敏感信息的结果。每scope最多一个active attempt；同key同载荷返回原attempt，不同载荷冲突；force-login也是新的明确命令，不能由读接口暗中发起。

新增的是原成功后 Store 同步的**明确结果字段/责任**，不另建一套登录状态机：`not_requested/pending/applied/blocked/unavailable`。Redis 的成功 Lua 同时提交 Cookie、登录终态、当前 scope 和 `sync=pending`；并保留只读识别/原同步命令所需的有界结果。进入 Store 阶段前把 observed merchant/expected version/命令fingerprint固定在同一attempt，不从后来状态重算同key载荷。

| 失败/并发位置 | 必须得到的结果与唯一恢复责任 |
| --- | --- |
| enqueue 响应丢失、重复提交 | 原 Redis Lua 原子去重；同 key 重读，不生成第二次登录；授权失败不得返回其它主体旧结果 |
| 取消与迟到浏览器成功竞争 | 原 active/attempt比较原子保护；cancelled/failed/interrupted后不得写Cookie或Store；succeeded仅允许本attempt已固定的pending Store同步，不允许重新登录/覆盖Cookie或后来attempt。终态后的cancel返回未接受，不能伪造已取消同步；控制事件带完整scope |
| worker 在验证码窗口崩溃 | 原pending reclaim可把不可恢复浏览器attempt记interrupted，但不解除Store active-owner或推定浏览器已关闭；无关闭证明则保持fenced，不重新登录/接管、不把验证码交给新attempt。真实排除旧执行者由原环境owner另行授权处置 |
| Cookie成功提交后、Store事务前崩溃 | 登录成功保持，sync仍pending；worker原reclaim路径只恢复只读识别/原Store命令，不发登录/验证码，不新增后台恢复器 |
| Store commit成功但响应丢失 / Redis记录未成功 | 唯一Store audit key+fingerprint重读原回执，再把原attempt sync标applied；不回滚Cookie、不重写Store、不换key |
| Store conflict/撤权/disabled/换revision | 不写Store；记blocked或unavailable，通用安全原因，不带他组织/店铺信息；新revision须经quiesce→activate，旧任务不能套新凭据或越过generation |
| Redis失败、scope不一致、回执不确定 | 不伪造成功；原attempt/原key核实，Store已提交部分不反向回滚；后续重试仅幂等原命令 |
| 应用退出/超时 | 原deadline、context、最大并发、health/shutdown保持；资源关闭不触发destroy/数据删除 |

Store 同步仅本地数据库可重放；supplier/warehouse GET不更新平台数据，不调用全局 auth refresher。失败/会话过期时返回明确失败，重新登录必须走原独立明确命令。同步未完成时不可把Cookie成功包装成整链ready。没有XA/Saga/outbox或新reconciler；复用原Redis pending consumer与Store本地事务/专用归属控制处理本次跨边界组合。

## 5. 完整删除 / 抽取 / 保留矩阵

下表为**拟议实施边界**，设计确认前删除数为零。源码路径是实际基线工件；新增当前文件名是落点建议而非已实现能力。每个mixed文件只处理指定闭包，不按目录名称整体清扫。

| 准确对象 | 决定与当前 owner | 删除/切换条件 |
| --- | --- | --- |
| `listingadmin` 十一生产源：`store.go`、`store_repository.go`、`store_query.go`、`store_model.go`、`store_access.go`、`access_scope.go`、`owner_constraint.go`、`repo_paging.go`、`gorm_store_api.go`、`management_api_types.go`、`schema_migrate.go` | RETIRE整个数字Store生产owner/DTO/自动schema补列及专属测试；有效权限/分页失败关闭/并发行为转为当前owner测试 | SHEIN新scope接线完成且TEMU尾部退出；全tracked生产Go零import；旧表绝不在新serving初始化/访问；CI测试路径按§6.1处理 |
| `sheinlogin/accounts.go`、`bootstrap/build.go`、`bootstrap/store_support.go` | RETIRE数字repository/provider/cache/StoreAPI构造与旧Redis构造；EXTRACT当前binding reader及窄Store/权限组合到原登录/runtime owner | 不留下别名、adapter、legacy loader；worker缺当前配置拒绝启动 |
| `sheinloginmanaged/bridge.go`、其 `bridge_test.go/duplicate_store_test.go` | RETIRE整个旧managed包；EXTRACT有效scope拒绝/重复及失败关闭断言至Store与登录owner | 当前Store单事务/同region组织唯一冲突测试覆盖后，原package与测试删除 |
| `sheinlogin/{types,service,worker,runtime,store}.go` | EXTRACT保留有效命令、登录/验证码/取消/终态/Lua算法；RETIRE数字scope方法/旧wire键/旧Task pause副作用 | 所有公开/私有scope为当前组织+UUID；原有效测试改写并先RED，不保留数字overload或新旧双分支 |
| `sheinlogin/profile_runtime.go` 及 automation系列、`browser_state.go` | EXTRACT浏览器检测/登录/验证码/只存Cookie/自有session关闭；RETIRE数字路径、宽进程kill/不受控解锁和专属断言 | 新root和scope安全测试成立，原有效页面/验证码行为保留；不执行真实浏览器 |
| `app/runtime/sheinloginworker/{runtime,options}.go`、`cmd/shein-login-worker/main.go` | 保留独立命令/端口/信号/close顺序；新增有限CLI producer，替换数字DB与隐式配置接线 | 六CLI动作经JWT/live授权调用当前命令；serve/producer不混启动浏览器，current Store/new Redis/new root显式验证，不装当前应用HTTP/Temporal/平台发布 |
| `shein/client/{runtime_types,cookie_manager,api_client,login_service_config}.go` 的数字Cookie/refresher接线，以及 service中的 `serviceCookieProvider/ConfigureRuntimeSheinAPIClients` | RETIRE worker对数字wrapper、全局refresher和数字StoreConfig的依赖；EXTRACT需要的supplier/warehouse/session HTTP能力至现有SHEIN transport owner | 先确认包外consumer；仍有有效APIconsumer的公开合同保持，不能删平台API/DTO/TLS/HTTP。新的worker不包装旧wrapper，不用0/哈希int替代真实scope |
| TEMU `client.go`/`interfaces.go`/`cookie_manager.go`/`auth_pause_handler.go`/`manager.go` 的 StoreRuntime及调用尾部 | RETIRE无生产注入的数字Store/Cookie/暂停/MallID回写支路；保留被各platform API消费的ClientAPI/HTTP/错误处理 | 保留有效HTTP构造/平台合同；无外部实现在全tracked代码再确认。禁止转移旧StoreAPI为TEMU同名facade |
| `core/config` 的SHEIN专属数字fallback/load/default/env/example及worker直接引用 | RETIRE仅准确失去消费者的SHEIN旧配置；保留SDS/1688/共享Redis/browser/HTTP当前合同 | 字段/函数消费者归零后删；SDS的共用LoginServiceConfig字段不可因SHEIN迁出一并删 |
| Store Center原Record/member grant/identity/audit及可选login-binding控制表 | 当前owner新增窄sync/activate/quiesce方法、一个专用控制表、action/最小DB角色；merchant及回执复用原事务/表，不复制Store模型 | 新权限/事务/同键、跨installation排他/关闭确认、paid-state不变证据通过；普通Store不增加可选表前置，原Store/API/官方/观察合同保留 |
| 原absence/import/依赖/安全guard、准确文档/部署样例引用 | 修改仅准确旧保留断言及未来worker示例；当前命令不编入数字owner，历史正文/计数保留 | 每项映射到新scope/保留行为；无新baseline豁免。操作说明是未来配置说明，不授权部署/运行 |

全闭包包括输入、运行、持久化、客户端旁路、配置、有效测试和直接操作说明尾部；不是把旧 `listingadmin` 整包复制到另一个目录。worker包名称是否变化没有退休含义：所有被保留算法必须依赖当前合同，旧owner与格式实际退出才算完成。

```text
Legacy decision: EXTRACT | RETIRE
Reusable behavior: 浏览器登录/验证码/取消、Cookie-only、attempt原子性、同scope隔离、Store安全同步及有效平台HTTP
Current owner: 当前身份/Organization与Store Center；独立sheinlogin的binding/attempt/session/Redis/Profile；原SHEIN/TEMU平台transport
Cutover/deletion condition: 新配置/当前scope/授权/Store事务与原安全回归就位，同一PR实际消费者归零后删除旧生产owner/格式；无运行或数据清理含义
```

## 6. 实施顺序、既有检查与保持不变项

确认后，同一分支/主要PR连续完成：先为当前scope、binding权限和Store同步写有效RED；实现窄合同和权限/事务；原位切换worker所有键/会话/Profile/客户端；移除TEMU及配置尾部和数字生产owner；集中补准确guards/文档并完成原CI。以上是同一交付批次的提交/自检边界，不是多个PR或重复设计/全仓审核。

### 6.1 两套 workflow 零改，旧 race 目标保持有效

本约束是实质删除边界：`internal/listingadmin` **不再含生产Go源**，不保留空package/shim/facade。现有 `go test -race ./internal/listingadmin -count=1` 目标保留为**测试专用目录**，测试直接消费当前 Store/登录公开合同；明确在原测试头说明历史CI路径，不新增runtime导入。重新归属原并发安全断言：当前组织隔离、同operation并发收敛/异载荷冲突、Store身份唯一冲突和旧receipt不倒退。测试必须实际执行和产生断言，不能全部skip、空测试或只检查文件不存在。真实PostgreSQL权限/事务证明放当前Store原集成测试owner，使用其已有fixture方式，不新增runner。

这是对既定“旧源码缺席”的准确限定：旧数字**生产**owner和消费者消失，历史CI目录名称可以保留有价值的当前安全测试；不得把测试目录冒充旧能力已可兼容。原commercial guard中检查race命令的断言保持原样。若实现证明该路径无法承载必要有效检查，报告具体问题/最小方案，不能自行改workflow或放宽门禁。

### 6.2 必要验证与证据层级

| 当前Must | 最小必要验证（实施阶段，当前NOT_RUN） |
| --- | --- |
| 真正退休/没有旁路 | 原tracked AST/import/dependency guards更新为旧生产owner无import、TEMU StoreRuntime无实现/消费、独立worker及current-application依赖无listingadmin/managed；现有guard覆盖build tags，不能仅grep名字 |
| 当前scope/凭据/撤权 | 原账号与service测试改写；两个不透明org/Store UUID、同值数字org文本不转int、错member/Store/disabled/依赖失败、换revision/撤权后拒绝、日志与队列不含secret |
| 入口实际可达 | 现有cmd/runtime测试扩为同一二进制serve/login/status/verify/cancel/activate/quiesce合成调用；producer不启动浏览器、当前JWT/成员拒绝、验证码stdin、同key查询、错attempt与UNKNOWN；不是只绕过CLI直接测Service |
| Cookie/attempt/验证码 | 原miniredis及worker mock测试保留并发enqueue、取消/迟成功、错attempt验证码、原终态、重启interrupted、Cookie-only、同步pending与原key核实 |
| Store事务与权限 | 当前Store既有SQLite/PG fixture：member隔离、同region唯一冲突、空值填充/相同重放/不匹配blocked、回执+Store原子性、响应丢失后重放、不同载荷冲突、后续version不倒退、paid/connection字段和role权限不变 |
| 跨installation/配置切换 | 同一Store两个namespace/installation同时activate只准一个；active浏览器未关闭拒绝切换；quiesce拒新任务且等待精确owner确认；旧generation/迟回调拒绝；reclaim/TTL/DB断开不释放active fence；普通Store无该可选表仍沿原合同可运行 |
| Profile/配置/生命周期 | 新临时路径/合成配置测试：root越界/旧namespace/隐式全局Redis拒绝，错revision不复用，busy不kill/unlock，health/关闭资源顺序保持；不启动真实浏览器/worker |
| 保留能力与CI | 受影响平台HTTP/TLS、Store observations/官方/授权原测试与相关命令编译；稳定候选走原必需CI和一次集中独立最终检查，不人工重复全套CI |

**本轮设计只执行源码/依赖调查、文档链接/diff与适用文档检查。** 不执行上表未来业务测试、不创建PG/Redis/浏览器实例，不读取私有配置/凭据，不操作任何现存资源。未来实施的合成持久化验证范围需随实施授权明确；真实登录/Store同步/provider/用户验收保持NOT_RUN。#692/#849/#862等既有owner不接管。

## 7. 评审、确认、退出条件与停止点

本轮交付独立设计、准确索引、#301当前段与一个主要PR。适用独立高风险检查仅核对本次当前scope/凭据输入、Store事务/授权、Redis成功与同步恢复、Profile归属、完整退休矩阵及§6.1，最多两轮正常设计检查；不重开当前官方Store/计费/采集全局架构。finding先分类，只有明确命中当前Must的BLOCKER重开设计；非阻塞建议进入实现测试或Backlog。

用户需确认**§1四项产品/持久化增量（含有限CLI）、§3/§4的owner与状态/恢复、§5整批矩阵及§6.1保留有效race路径**。在此之前本提案不改变既有worker保留决定，不签生产IMPLEMENTATION_READY。确认与适用独立准入满足后，才在同一主要PR实施；实现/CI/最终检查完成交In Review，合并仍需准确HEAD授权。

源码完成条件是旧数字账号/Store生产owner、旧键/Profile/配置与旁路消费者退出，保留行为由当前owner安全承接；不以更名、空壳、注释或只绿CI替代。当前应用/独立worker去旧数字依赖证据与旧实例资源状态分别记录；即使完成本批，#301整体关闭仍待原验收及另行明确授权，不能自动关闭。

本设计不授权部署、实例启停、连接/读取现有DB或Redis、访问/复制旧凭据和Profile、迁移/删除行键/卷/文件、真实SHEIN调用或父Issue关单。已保留实例和数据保持UNKNOWN/未操作，真实运行及用户验收NOT_RUN；内部试用不等待本设计或将来的worker切换。
