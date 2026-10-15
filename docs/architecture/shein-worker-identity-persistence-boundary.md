# 独立 SHEIN worker：身份与持久化边界设计

Refs [#301](https://github.com/qq550723504/task-processor/issues/301)。

- Status: **DESIGN_REVIEW / 待用户确认；正式实施未授权**。
- Design Basis: **N/A（本批仅文档及只读现状映射）**。不改变身份、权限、事实 owner、状态机、持久化或外部副作用；本文件不是新业务合同的实施准入。
- 调查基线：`main @ b3940ea9448d0190a3b69617160b176c36f4a11a`。
- 用户决定：保留当前采集能力、有效共享基础层和独立 SHEIN worker；不操作运行实例或数据；设计确认后再实施；本项不作为内部试用的新前置。

## 1. 用户结果与设计决定

操作者继续使用当前内部试用应用，并保留原独立 SHEIN worker。维护者可以按具体合同判断哪些代码可删、哪些仍拥有运行态，避免为了清空数字字段而破坏 worker，或把旧账号接入当前 Store。

本设计提出三个边界决定，待用户确认：

1. 当前新业务只消费当前 Organization、Store、官方应用/凭据和各业务 owner 的合同。保留旧 worker 不增加一条当前业务的登录、身份或恢复路径。
2. 独立 worker 及其仍被消费的数字账号仓库、Redis、Profile、会话、状态机和 Store 同步保持原 owner。保留的是符合原租户隔离要求的能力，不包括已确认的无 tenant 条件查重缺陷，见 §5.1。它们仍是退出条件未满足的旧设计债务，不是新系统的合法事实来源，也不是新增 Compatibility 类别。
3. 下一实施候选退休已无生产消费者的另一套 managed 账号 provider 和孤立验证码请求 DTO，并最小修复 §5.1 的既有租户隔离缺陷，见 §7。使用中的 provider、验证码服务、Store 同步及 worker 不在删除清单中；本轮没有实施这些删除或修复。

保留 worker 的决定与“全仓数字身份依赖清零”目前不能同时作为已完成结果。#301 保持开放；当前应用退出旧运行链的成果和保留 worker 的债务分别记录。不得把未删除写成已退休，也不为关闭父 Issue 自动发明新的账号/凭据系统。

## 2. Authority、范围与准入

按以下既有依据解释边界，不扩展它们的产品范围：

- [全新系统基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md)：只有 EXTRACT / RETIRE，无旧数据迁移、ID 映射、fallback、双读/双写或第二事实源。
- [当前身份/企业空间](../superpowers/specs/2026-08-30-shuomi-workbench-store-center-zitadel-multi-org-design.md) 与 [Store current application](store-center-current-application-v1.md)：当前身份与 Store 记录归原 owner；店铺记录存在不等于官方连接可用。
- [我的供应链 SHEIN](my-supply-chain-shein-v1.md) §8 的 StoreProductAccess / 官方执行合同：新业务请求不得调用旧登录 Profile 或 seller 网页发布 runtime；Submission 保持唯一发送、UNKNOWN 与恢复责任。
- [Store 平台观察](store-center-platform-observations-v1.md) 及[内部准备接线](store-observations-preparation-runtime.md)：保留当前记录/观察与原权限、配置和真实可用性边界，不因本设计新增 provider 调用或试用前置。
- [发布预检退休](release-identity-preflight-retirement.md)、#703/#719/#721：旧 bridge、1688 数字账号/队列/Profile 和未注册导入 HTTP 已退休，不能通过 SHEIN 边界恢复它们。

本批交付仅此设计与直接引用校准，UI/IA 不变，Figma N/A。当前采集、共享层、所有生产源码、schema、Redis 键、配置、部署、CI 和保护规则均不改。

Architecture Review 对本次纯文档交付为 N/A；最终 PR 仍按原流程做完整差异检查。§7 的有界退休可在用户确认并把准确实施范围写入 Issue 后，按 Reuse Existing Architecture 复用这些依据。若未来改变身份/授权、DB 所有权、状态机、事务、恢复或外部副作用，则另做 Independent Architecture、适用独立评审及 IMPLEMENTATION_READY；本文件不代替该准入或用户实施授权。

## 3. 三类身份与事实所有权

| 对象 | 当前 owner / 权威 | 本设计决定 |
| --- | --- | --- |
| 当前用户、Effective Organization、成员权限 | ZITADEL / 当前身份、成员及授权 owner | Organization 是不透明标识；即使是数字字符串，也不解析成 legacy tenantID，不做相等推断 |
| 当前店铺记录、访问权、官方连接和 merchant-bound 执行能力 | Store Center 及当前官方应用/凭据端口 | Organization + 当前 StoreID 和版本/连接引用保持原合同；不从旧 `listing_store`、Cookie、Profile 或登录状态补齐 |
| 当前来源账号登记 | `sourceaccountregistry` | Organization 注册事实；`pending_connection` 不表示可采集，不提供 SHEIN 登录兼容 |
| 旧登录账号 | `sheinlogin.ListingAdminAccountProvider` → `listingadmin.StoreRepository` | 使用数字 `TenantID` + 旧 Store 行 `ID`，读原 `listing_store`；这是 worker 原输入，不是当前 Organization/Store owner |
| SHEIN 外部店铺标识 | 旧行的 `StoreID string`，或当前 Store/官方连接各自验证的 merchant 标识 | 旧数字行 ID、外部店铺字符串与当前 StoreID 是不同概念；同名/同值不构成授权、绑定或身份映射 |

旧 provider 从原仓库读取平台、用户名/密码、登录 URL、代理和店铺属性。凭据字段不是新 Store 模型的创建输入；不抄入 Product/Listing、当前 Store、日志或设计证据。这里记录字段职责，不读取任何真实凭据。

## 4. 实际调用链与共享包边界

保留的独立命令链：

```text
cmd/shein-login-worker
  → app/runtime/sheinloginworker
  → 原 DB handle + listingadmin.NewGormStoreRepository / NewGormStoreAPI
  → sheinlogin/bootstrap.BuildService
  → sheinlogin.NewListingAdminAccountProvider(repository)
  → sheinlogin.NewService → Service.RunWorker
      → Redis attempt / stream / control
      → 原数字账号加载 → 原 browser / Profile / verification session
      → 原 Cookie / attempt 完成 → 原 Store ID 同步与重复店铺处理
```

`bootstrap` 还注入 `sheinloginmanaged.NewStoreSyncClientFactoryWithStoreAPI`、`NewDuplicateStoreLookupWithStoreAPI`，并配置原进程内 SHEIN client refresher。这些是使用中的 worker 接线，不能随 unused provider 整包删除。worker 的 `/healthz`、信号取消、资源 Close 与原数据库/Redis/browser 配置继续保留；旧 HTTP 登录 API 已退出，保留服务不授权重新挂载它。

基线上实际执行 `go list -deps`，只反映源码编译依赖：

| 命令 | 已核对的相关依赖 |
| --- | --- |
| `current-application` | 有 `storecenter`、`listingadmin` 和根 `listingruntime`；没有 `sheinlogin`、`sheinloginmanaged`、其 bootstrap 或 `listingkit/sheinsync` / `listingkit/store` |
| `1688-local-agent` / `1688-public-browser-collector` | 没有上述 SHEIN 登录、ListingAdmin/ListingKit/ListingRuntime 或 Store Center 包 |
| `image-agent-temporal-worker` | 有当前 `storecenter`；没有上述 SHEIN 登录与旧 ListingAdmin/ListingKit/ListingRuntime 包 |
| `shein-login-worker` | 有 `listingadmin`、`sheinlogin`、`sheinloginmanaged`、`sheinlogin/bootstrap`；没有当前 HTTP application、Store Center、`listingkit/sheinsync` / `listingkit/store` |

当前应用的 `listingadmin` 依赖路径为 `app/httpapi → marketplace/shein/draft → publishing/shein → listingadmin`；`listingruntime` 路径经过同一 publishing owner → `shein/authorizedbrand → listingruntime`。这不等于当前应用读取 `listing_store` 或已获得旧 worker 登录能力。共享包混有仍消费的合同，不能以目录名或数字字段作整包删除依据，也不为此自动扩大为发布系统重构。

## 5. 持久化、隔离与失败边界

| 边界 | 原 owner 与身份 | 必须保留的行为 / 本批限制 |
| --- | --- | --- |
| 原账号 DB | `listingadmin.StoreRepository` / `listing_store`，数字 tenant + Store 行 ID | worker 每次按原账号 provider 读取；原仓库权限/过滤/缓存边界保持，不把当前成员权限映射到旧数字账号 |
| Redis Cookie / 验证码 / wait / failure / login time | `sheinlogin.RedisStore`，原 `shein:*:<tenant>:<store>` 等键 | 同一 Store 行号在不同 tenant 下隔离；Cookie-only browser state、TTL 和原错误处理不改 |
| Redis attempt / stream / control | 同一 RedisStore；随机 attempt ID、原 active/latest 引用、`shein:login_attempts` consumer group | 原 Lua 原子创建/完成、terminal 状态保护、ack、lease、attempt-specific code/cancel 保持；不换键或将队列交给另一事实 owner |
| 本地浏览器 Profile / artifacts | `sheinlogin` browser owner；原绝对 ProfileRoot 下 tenant/store 路径与原 artifact 路径 | 不搬运、复用、重命名或删除 Profile；原锁清理、浏览器进程处理与缓存整理代码保留，本设计不执行它们 |
| 进程内锁 / 并发 / 验证 session | `sheinlogin.Runtime` / `Service`，tenant+store key | 同 Store 跨 tenant 不共享锁/session；并发上限、headless 行为、取消和 Close 保持；不把内存锁称为跨进程事务 |
| 登录后原 Store 写入 | `Service.syncStoreIDAfterLogin` → 原 StoreAPI | 保留外部 Store ID 写回和同 tenant 冲突/查重能力；现有无 scope 查询及错误后继续写入须按 §5.1 最小修复，不写当前 Store、不改变当前 merchant binding |
| 当前业务事实与官方副作用 | 当前 Store / Product / Listing / Submission 各原 owner | 无旧账号/Profile fallback；不新增 browser 登录前置，不复活旧队列/API/身份 bridge |

静态读取到的现状协议如下，不将其重述为新保证：

- 登录 attempt 的 active 状态为 `queued / launching / waiting_verify_code`；终态为 `succeeded / failed / cancelled / interrupted`。原 Lua 仅允许 active attempt 的对应完成，取消与终态保护仍归原 owner。
- 创建操作按原 tenant+store active 引用复用已有 attempt；它不是新系统的 request-key + payload-hash 合同，不额外承诺“不同载荷必冲突”。密码不进入 `LoginAttempt` 或 stream body。
- 验证窗口的浏览器 session 由原 worker 进程持有。原 stale stream claim 路径把无法恢复的 active session 标成 `interrupted`，而不是在新进程重建成功状态或自动开始新的登录。
- Cookie、成功 attempt 和相关 Redis 状态有原 Lua 提交边界；Redis、账号 DB、Profile 文件及 SHEIN 外部动作没有一个共享事务。原登录后 Store 同步失败不能推导为 Cookie/attempt 回滚；本次不新增补偿、outbox、Saga 或重试 owner。
- 响应丢失、重试、重启、并发、验证码超时、取消和 Redis 故障沿原服务处理；Store 查重的失败关闭增量明确见 §5.1。源代码和已有测试可以说明分支，但不证明当前部署的恢复、备份、Redis 持久化或平台行为。
- 真实登录、发送验证码、SHEIN API、原 Store 写入和 Profile 维护属于外部/数据副作用。本批全部 NOT_RUN，不以文档确认授权执行；生产继续 NOT_READY。

### 5.1 已确认的查重租户边界缺陷与最小设计修正

独立检查发现：`sheinloginmanaged.findDuplicateStore` 的 `PageStores` 请求未设置 `TenantID`。`gormStoreAPI.PageStores` 将零值传到 `ListStores`，`applyOwnedTenantQuery` 仅在 tenant>0 时加过滤，后台空 owner context 也不收窄查询；因此可能返回其他 tenant 的记录。`syncStoreIDAfterLogin` 随后把返回行的 tenant/行 ID/名称写进当前行备注，并禁用当前行。以上仅是已核对的源码路径，没有访问真实库或确认实际泄露事件。

Finding classification: **BLOCKER（跨租户访问/数据混淆）**。受影响 Must 是既有租户隔离与正确 Store 写入；阻塞把此行为确认为可原样复用，以及后续 worker 实施/验收。没有跨 tenant 全局查重的产品授权，不分类为 Accepted Risk。它不增加当前独立内部试用路径的前置。

后续实施只在原 helper/service owner 补齐既有隔离合同：

1. `account.TenantID` 必须为正，账号行 ID 与解析出的外部 Store ID 必须有效；输入无效时在查询/Store mutation 前拒绝，不以 tenant=0 表示“全部”。
2. 每一页 `StorePageReqDTO` 都显式使用 `account.TenantID`，保留原分页和平台过滤。查重只考虑同 tenant、非自身、启用中的 SHEIN 行；同外部 Store ID 出现在其他 tenant 不能成为禁用当前行的依据。
3. 查询返回非空行若不属于该 tenant，视为 scope 错误，返回不含外租户 ID/名称/凭据的错误；不得把它当成合法 duplicate 或“无重复”继续写入。nil/无效 page 或查询错误也视为未完成查重。
4. `syncStoreIDAfterLogin` 在使用 duplicate 的任何字段、写备注/日志或禁用前再校验同 tenant。查重失败、scope 异常或未完成时立即结束这次 Store 同步，不继续 UpdateStoreId 或 UpdateStoreStatus；仅记录不含外租户信息的失败分类。
5. 同 tenant 的有效重复店铺仍可沿原规则处理；同 tenant 自身和 disabled 行继续排除。这里只收窄原数字 worker 的可见/写入范围，不把旧账号映射成当前 Organization，也不新增 global merchant owner。
6. Cookie/attempt 成功提交仍在原 Redis 边界，后续 Store 同步失败不伪造该提交已回滚、不改为“新 attempt 重新登录”，也不宣称 Store 同步成功；不新建状态机/恢复协议。

同根因相邻路径已静态核对：原账号 provider 的查询已显式带 tenant；实际 Store 同步工厂忽略 factory 的 tenant 参数，不能把 factory 本身当成隔离保证。原 GORM StoreAPI 仍是内部能力，本次不因此重构整个 StoreAPI/IAM；在查重查询和服务副作用入口落实上述同 tenant 约束。若实现证明还必须改变公共授权合同或持久化/恢复边界，再停止并按 Independent Architecture 准入。

验证要求写入下一候选，全部尚未执行：先用现有 helper/service/GORM 测试设施做 RED→GREEN；证明每页 tenant 参数、两个 tenant 同外部 ID 不互相禁用/泄露、异常外租户返回与查询失败均无 Store 写入，以及同 tenant 重复、自身/disabled 排除和分页仍符合原合同。不得只改文案、仅在展示时隐藏数据，或只依赖 stub 忽略原 GORM 路径。

## 6. 保留能力与非目标

保留现有 worker 的账号读取、登录/强制登录、Cookie 状态、验证码与取消、durable attempt 消费/终态保护、重启 interrupted 处理、跨 tenant 隔离、同 tenant Store 同步/重复识别、health/shutdown、Redis/browser/Profile 配置和清理实现。§5.1 已确认的跨 tenant 缺陷不属于获准保留的不变量。保留源码能力不声明现有用户入口、真实账号可用、运行实例健康或产品验收通过。

当前匿名采集、Browser Capture、本地执行器、SRC-1/Catalog、SourceAccountRegistry、共享任务 trusted TenantID / 结果作用域 / workerpool / Redis / browser / HTTP 与 Amazon 保持各自原 owner。本文件不给“数字类型”统一赋予 legacy 含义。

不新增账号页面、登录 HTTP、operator 步骤、验收 runner、状态平台、IAM、凭据 vault、映射/迁移、官方连接或发布能力。内部试用继续沿已有交付范围进行；不等待 worker 原生 Organization 化、旧数据处理或本设计落地。其它活跃 PR 与运行实例继续由各原 owner 负责。

## 7. 删除矩阵与下一有界实施候选

以下是设计候选，**当前没有执行任何删除**。正式实施前须取得用户确认、刷新消费者并在 #301 写入准确 Ready 范围。

| 工件 | 设计处置 | 源码删除条件 |
| --- | --- | --- |
| `sheinloginmanaged.AccountProvider`、`storeClientFactory`、其 `tenantAccountCache`、`NewAccountProviderWithStoreClientFactory`、GetAccount/ListAccounts | RETIRE 候选；这是另一套通过 StoreAPI 读取账号的闲置具体实现，不是 `sheinlogin.AccountProvider` 接口 | 全仓生产引用为零；唯一外部包消费者 bootstrap 仍只调用两项 Store 同步/查重工厂；原仓库 provider 保持原接线，不用别名/包装替代 |
| 同文件 `mapStoreToAccount`、`MapStoreToAccountForTest`、`normalizeLoginURL` 及专属测试 | 随上述闲置实现 RETIRE | 只被其自身/测试使用；比较当前 `sheinlogin/accounts.go` 的有效平台/凭据/URL映射断言，已有覆盖复用，确有缺口才在保留 owner 补齐，不保留生产 test-only API |
| `sheinlogin.VerifyCodeRequest` | RETIRE 候选；已退休 HTTP 的孤立 wire DTO | 全仓仅定义，无生产或有效测试消费者；保留 `SubmitVerifyCode`、`SubmitVerifyCodeForAttempt`、attempt/取消协议及 `LoginRequest`，后者仍由 worker/service 使用 |
| 使用中的 `findDuplicateStore` / `syncStoreIDAfterLogin` 查重与写入入口 | 保留能力，并按 §5.1 最小修复现有 tenant 隔离缺陷；不是删除目标 | 未实施。原 scope 合同的 focused RED→GREEN、实际 GORM 路径及同/跨 tenant 服务副作用断言通过前，不得称 worker 此边界已修复或放行下一实施候选合并 |
| `sheinlogin.ListingAdminAccountProvider`、`AccountProvider` 接口、原 Account/LoginRequest/LoginAttempt、原 Store 仓库与两项 managed 工厂/查重 helper | 保留，仍被 worker 实际消费 | 本批删除条件未满足。须先有用户确认的独立替代能力或明确退休该能力的决定，当前权威/安全行为切换且原消费者退出；不能先删再包装旧 owner |
| worker Redis 键/stream/Lua、Profile、锁/session、登录/验证码状态机、原 Store 同步 | 保留原 owner 与格式 | 本批删除条件未满足。若未来改变其身份/持久化/恢复，另做 Independent Architecture；不能把旧键、Profile 或行 ID 搬到新系统续跑 |
| 当前共享包里的有效合同、平台规则、基础层与当前采集 | 保留；仅具体无消费者旧工件可 RETIRE | 基于当前消费者逐项判断，禁止 blanket-delete `listingadmin` / `listingruntime` / `sheinloginmanaged` 或整套 shared 层 |
| 已退休的 bridge、旧 HTTP/Task/Profile/预检入口 | RETIRE / 保持缺席 | 原 absence、实际依赖和权限守卫继续；不得因 worker 保留而恢复 |

§7 闲置 provider/mapper/DTO 的退休与 §5.1 最小租户隔离修复可在同一主要实施 PR 收敛，使用 Reuse Existing Architecture：删无消费者代码和只验证旧实现的测试，保留并补齐原 Store 同步/查重测试、当前账号 provider 的权限/缓存隔离和验证码/attempt 测试，复用原 guard。修复的是已批准隔离合同未被执行的局部缺陷，不新增权限模型；设计确认和正式实施授权仍待用户。纯库删除不是实际数据库、Redis、文件、镜像或实例删除授权。

对测试先分类：闲置构造器/旧 transport wire/旧 mapper 实现细节可为 NOT_APPLICABLE；当前租户/账号/凭据隔离、平台过滤、取消/终态、幂等和副作用保护仍是 Must，不能通过删测试规避。必要 finding 沿原分类规则处理；不能借本候选追加新的通用验证平台。

若用户未来要求 worker 消费当前 Organization/Store，必须先决定该业务是否仍需要网页账号登录、由哪个当前 owner 正常创建/授权/保存新事实，再设计权限、凭据、状态与恢复。当前官方业务合同禁止的旧 Profile 调用不能通过“直接接新 Store”绕开；不在本设计中预建新 schema、adapter 或旧新 ID 对照。

## 8. 检查与交付停止点

本批检查只包含实际源码/消费者映射、上述命令的 `go list -deps`、文档链接、diff 范围及 `git diff --check`，并使用原 PR 适用 CI/独立检查。不运行 worker、浏览器、Docker、数据库/Redis查询、迁移、初始化或平台调用；已有测试名/历史 PASS 仅是后续复用位置，不冒充本次执行。

后续 §7 实施的必要检查复用现有能力：

- 原 current commands / standalone SHEIN actual dependency 与退休 guard；不得放宽现有 CI/保护。
- `sheinlogin` 的原账号 owner/角色缓存隔离、跨 tenant 读写/锁/session、Cookie-only、attempt 原子创建/取消/终态/重启测试；`sheinloginmanaged` 保留的原同步/查重测试；bootstrap 与独立 runtime 配置/Close 测试。
- §5.1 的查重租户条件、实际 GORM 查询与服务副作用 RED→GREEN；这是真实 BLOCKER 的最小必要测试，不新增专项工具或人工流程。
- 保留命令编译及受影响当前采集/共享层原回归；无关未变代码证据可明确复用，不人工重复完整 CI。

本批交付后停在 **设计待用户确认**，不自动进入 §7 实施、不合并、不部署、不关闭 #301。设计、源码实现、准确候选CI、合并/main CI、真实运行和用户验收分别记录；未执行项目保持 NOT_RUN。当前保留实例的地址、数据、Redis拓扑/持久化、Profile 和凭据均未读取或更改。
