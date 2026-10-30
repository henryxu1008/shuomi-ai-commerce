# #301 旧配置别名、单项投影提交与 Dub 完整退休边界

## §1 用户结果、Design Basis 与当前准入

本批把三个已定位的旧设计闭包一次收齐：配置不再通过旧别名取值；图像投影只接受现有的有序数组命令；没有当前消费者的 Dub 归因适配器退出仓库。维护者不再面对两套配置名称或两个 slot 提交通道；当前采集、图像、身份权限与 Store 观察同步继续使用原 owner。

- **Design Basis: Independent Architecture**。本批改变配置输入合同和内部 Repository 命令形状，必须先确认设计，再实施；不是按文件数拆批。
- 当前源码阶段：**`Independent Architecture / USER_CONFIRMED / IMPLEMENTATION_READY / SOURCE_IMPLEMENTED / IN_REVIEW`**。用户已确认准确设计的 §1–§7（含§4.2/§6.1）；适用独立设计复核无待处理问题，准确设计原CI整轮SUCCESS后开始实施，准入及准确证据记录于同PR。此前只读/三文档阶段的N/A不作为生产代码准入依据。
- 基线：`bd2640ffb15baa29bea9639a5796fb3527403354`；用户 2026-10-30 的三组整批设计授权；接单记录见 [#301](https://github.com/henryxu1008/shuomi-ai-commerce/issues/301#issuecomment-6512801376)。同一分支、一个 Writer、一个主要 PR；#1006 的成员角色交付不在本批。
- 本轮按用户已确认的§5完整清单，在同一PR整批实施三组闭包；运行验收、源码最终复核、合并和关单仍分别授权。

**Must**：保留当前采集、图像、身份权限、Store 观察同步；旧身份限制输入不得静默失效，保持现有权限拒绝；保持当前 JSON/hash/版本与投影原子性。**Out of Scope**：workflow/inventory/保护规则/依赖变更，真实环境核对、实例/数据/凭据/进程操作、Dub 调用或外部清理、上线/运行验收、关单。无新增 Accepted Risk；不增加内部试用前置，#301 保持 OPEN。不恢复已经退休的独立 SHEIN 登录或身份回写 worker。

## §2 依据与前批 KEEP 的准确替代

适用 [AGENTS.md](../../AGENTS.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[全新系统产品基线](../product/greenfield-no-legacy-migration.md)、[当前图像架构](product-image-agent-v1.md) 和 [当前推荐业务边界](../engineering/current-application-referrals.md)。不改 UI/IA，没有新增 Figma 产品增量；不扩大现有 Threat Model。

| 前批依据 | 用户确认本设计后替代的范围 | 继续有效的边界 |
| --- | --- | --- |
| [#939 旧全局配置/日志](legacy-global-config-logging-tail-closure-retirement-boundary.md) §3/§5 对配置加载与 `logDeprecatedEnvUsage` 的 KEEP | 仅结束 `envBinding.Deprecated`、22 个别名的取值及其警告；保留原加载器 | dotenv、文件查找、原 Primary、当前日志和校验，logging → config 的真实边仍保留 |
| [#891](legacy-listing-preview-download-closure-retirement-boundary.md)、[#893](legacy-localagent-source-policy-closure-retirement-boundary.md)、[#958](legacy-idle-provider-bootstrap-closure-retirement-boundary.md)、[#977](legacy-imageagent-http-runtime-closure-retirement-boundary.md)、[#988](legacy-subscription-metering-closure-retirement-boundary.md) 对 Dub 准备能力的 KEEP/Out of Scope | 这些历史决定不等于永久保活；本批用户明确选择结束无当前入口、无注入的 Dub 整包 | 当前推荐码/注册关系/统计及其它现用集成，不新增分润、计费、付款或第二账本 |
| 已有有序 `SlotMutations` 协议（源码提交 `5122bde39`）与当前图像 Repository 合同 | 结束并存的 `SlotMutation` 指针及单项转数组 helper | `SlotProjectionMutation` 值类型、数组顺序、合法单槽业务、Repository 事务/恢复与全部当前安全断言 |

历史原文、计数和证据保留。源码实施时只给 §5 列出的六份旧设计加准确替代顶注，不重写旧阶段结论；本设计不把旧外部状态推定为不存在，也不承诺兼容旧配置或旧 Go 命令。

## §3 实际消费者与完整调用链

### §3.1 配置别名

`core/config.newViper → bindKnownEnvs / AutomaticEnv → BuildConfig → 当前配置消费者`。`LoadConfig` 仍供原浏览器采集/debug CLI；两个文件加载入口仍供当前 application/worker/schema-prepare 配置装配。它们与 dotenv、日志不是闲置包。`loadWithViper` 和 `loadWithViperWithoutValidation` 已共用旧配置拒绝入口；身份解析和权限判定仍由当前 authruntime/application 拥有，不移入配置加载器。

基线 `knownEnvBindings` 有 64 项，22 个 `Deprecated` 名称分布如下。所有 Primary **逐项保持**；未列条目不因此改名，包括当前裸名 `ZITADEL_*`、`LISTINGKIT_PLATFORM_ADMIN_*`。

| 逻辑键 | 退休的显式旧名 | 保留 Primary |
| --- | --- | --- |
| `openai.apiKey` | `OPENAI_API_KEY` | `TASK_PROCESSOR_OPENAI_API_KEY` |
| `openai.model` | `OPENAI_MODEL` | `TASK_PROCESSOR_OPENAI_MODEL` |
| `openai.baseURL` | `OPENAI_BASE_URL` | `TASK_PROCESSOR_OPENAI_BASE_URL` |
| `openai.clients.image.apiKey` | `TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_APIKEY` | `TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_API_KEY` |
| `openai.clients.image.baseURL` | `TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_BASEURL` | `TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_BASE_URL` |
| `openai.clients.image.apiStyle` | `TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_APISTYLE` | `TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_API_STYLE` |
| `browser.browserPath` | `BROWSER_PATH` | `TASK_PROCESSOR_BROWSER_PATH` |
| `browser.userDataDir` | `BROWSER_USER_DATA_DIR` | `TASK_PROCESSOR_BROWSER_USER_DATA_DIR` |
| `browser.headless` | `BROWSER_HEADLESS` | `TASK_PROCESSOR_BROWSER_HEADLESS` |
| `browser.poolSize` | `BROWSER_POOL_SIZE` | `TASK_PROCESSOR_BROWSER_POOL_SIZE` |
| `platforms.alibaba1688.enabled` | `PLATFORM_1688_ENABLED` | `TASK_PROCESSOR_PLATFORM_1688_ENABLED` |
| `database.host` | `DB_HOST` | `TASK_PROCESSOR_DATABASE_HOST` |
| `database.port` | `DB_PORT` | `TASK_PROCESSOR_DATABASE_PORT` |
| `database.user` | `DB_USER` | `TASK_PROCESSOR_DATABASE_USER` |
| `database.password` | `DB_PASSWORD` | `TASK_PROCESSOR_DATABASE_PASSWORD` |
| `database.database` | `DB_NAME` | `TASK_PROCESSOR_DATABASE_NAME` |
| `database.max_connections` | `DB_MAX_CONNECTIONS` | `TASK_PROCESSOR_DATABASE_MAX_CONNECTIONS` |
| `database.max_idle_connections` | `DB_MAX_IDLE_CONNECTIONS` | `TASK_PROCESSOR_DATABASE_MAX_IDLE_CONNECTIONS` |
| `listingkit.zitadel.allowedTenantIDs` | `LISTINGKIT_ZITADEL_ALLOWED_TENANT_IDS` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_TENANT_IDS` |
| `listingkit.zitadel.allowedUserIDs` | `LISTINGKIT_ZITADEL_ALLOWED_USER_IDS` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_USER_IDS` |
| `listingkit.zitadel.allowedUsernames` | `LISTINGKIT_ZITADEL_ALLOWED_USERNAMES` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_USERNAMES`（仍只供拒绝检测） |
| `listingkit.zitadel.allowedRoles` | `LISTINGKIT_ZITADEL_ALLOWED_ROLES` | `TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_ROLES` |

根因有两处，不能只删表：

1. 仓库锁定 Viper v1.16.0 的 `find → getEnv` 会先尝试 `AutomaticEnv`，再查 `BindEnv`。点号替换会重新得到上表三个图片客户端紧凑旧名；它们必须在自动入口也退出。
2. `loader_builder.go::legacyListingKitUsernameAllowlistConfigured` 仍从 `Deprecated` 表枚举被禁用的 username 限制。直接删表会漏掉原拒绝输入。另三项旧 allowlist 若被静默忽略，会改变操作者配置的生效/拒绝结果。当前 middleware 在 Required=true 时明确拒绝全空 allowlist，Required=false 时原本不执行该检查，不存在本批修补“全空放行”的依据；新增 name-only 拒绝是用户已确认的 hard-cut 配置合同，用于避免旧输入静默失效，不改变权限算法。

### §3.2 单项投影命令

当前 `ActivityDependencies.Repository → Activities.repository` 注入 `imageagent.Repository`。`activities_persistence.go::PersistSlotResultV3` 与 `activities_recovery_v3.go::reconcileEffectRecoveryProjection` 是旧指针的两个生产构造者；`PersistRunState` 已使用数组。Repository 两个实现共用验证，再分别提交 Memory 备份/回滚和 GORM 事务。

`orderedSlotProjectionMutations` 的三个调用点是 `applyMemoryProjectionMutation`、`validateProjectionMutation`、`applyGormProjectionMutation`；helper 同时拒绝双字段并把指针包装成数组。整闭包是一个命令字段、一个 helper、三个读取点、两个生产构造点和 §6.1 的四份测试 fixture，不能只删字段或另留 wrapper。

`ProjectionCommit` 是同进程 Repository 命令，不是 Temporal activity 输入或 HTTP body；但 **`store/projection.go::projectionCommitFingerprint → hashJSON(input)` 确实把整个命令编码为 JSON 并 SHA-256，结果存入 commit 回执**。Snapshot/EventPayload 另行持久化。直接删 Go 字段会使所有提交（包括数组和 run-only）的指纹改变，这是本批必须先抽取的真实持久化边界，不能把“没有 json tag”当作未持久化证据。

### §3.3 Dub

`internal/integration/dub` 只有 `client.go/types.go/doc.go/client_test.go`。构造器及 partner/link/lead/sale 方法没有外部生产 import、装配或注入，只有包内实现与测试；无 Dub 配置绑定、workflow/构建目标、inventory 行或直接 Go SDK 依赖。`internal/integration/README.md` 仍列它为有效适配器，是专属文档尾部。

当前推荐业务文档明确没有 Dub 或第二统计账本，现有业务不消费上述 partner/link/event 合同。其数据 owner 原为外部 Dub，仓库适配器只发 HTTP mutation；当前推荐关系仍归原业务 owner。历史外部 partner/link/event 是否存在为 `UNKNOWN`，本批不查询、清理、撤销或转移。

## §4 owner、JSON/hash 与事务边界

### §4.1 复用当前能力，拒绝兼容层

**配置 owner 不变**。`envBinding` 保留 Primary，删除 Deprecated 字段/初始化/取值、两警告函数及三个加载入口的专属警告调用。普通加载日志、dotenv 查找/缓存/已有环境优先、文件/YAML/default/显式 Set 语义、有效校验和当前配置结构保持；不删除 logger import 或 inventory 边。

针对恰好三个与 Deprecated 重合的自动键，复用现有 Viper `NewWithOptions(EnvKeyReplacer(...))` 的 `StringReplacer` 扩展点：只把完整的 `TASK_PROCESSOR_OPENAI.CLIENTS.IMAGE.APIKEY/BASEURL/APISTYLE` 自动查询精确映射到相应原 Primary；显式 Primary 原样通过，其它查询保持原点号替换。不得做子串/前缀替换、全局关闭 AutomaticEnv、批量收严其它 64 项的隐式拼写、引入假环境名或新配置框架。此处只读取当前 Primary，不从旧环境变量取值。

原 username helper 改为明确检测原 YAML/取值，以及原 Primary 和 `LISTINGKIT_ZITADEL_ALLOWED_USERNAMES` 的非空存在；原 validator 字段/错误、空 Primary 不遮住旧输入的断言保持。原自动 username 拼写仍被原取值检测捕获；不恢复 username 登录授权。

另外三个旧 `LISTINGKIT_ZITADEL_ALLOWED_TENANT_IDS/ALLOWED_USER_IDS/ALLOWED_ROLES` 进入现有 `retiredProductRuntimeEnvironmentVariables` 拒绝表。两个 loadWithViper 入口在 BuildConfig 前执行原拒绝检查，包括 WithoutValidation；只报告变量名，不打印值。其它退休配置规则和当前 middleware/角色/组织身份校验零改。

**投影 owner 不变**。删除指针后，两个构造者直接给 `SlotMutations: []SlotProjectionMutation{{原字段}}`；三个读取点直接使用数组。无转换 helper、compat alias 或兼容字段。保留调用者数组顺序，不排序、去重、重造 Snapshot 或拆多次 commit；nil/空数组仍允许合法 run/plan-only 提交。

先在原 store owner **EXTRACT 当前数组协议的指纹编码合同**：`projectionCommitFingerprint` 不再直接依赖可变 Go 命令形状，改为明确逐字段的私有编码结构，字段名称/顺序/零值与原命令相同，其中旧位置只输出固定 `"SlotMutation":null`。该占位仅是当前数组回执 hash 的固定字节，不是可赋值的单项命令字段；不保留单项指针/可赋值成员、双读/双 hash 或旧单项归一化。保留数组的 nil/空区别、其它字段原编码和原 `hashJSON` 算法；不加 omitempty、排序、重算记录或新版本。原数组/run-only 回执在完整相同命令下继续匹配；原指针提交的非 null 指纹不兼容，新单元素数组不得被伪装为原指针载荷。

保持原 Memory 锁及失败回滚；GORM 仍在同一事务内做 scope/CAS、run/plan/slot/attempt、event/cursor/receipt 更新。当前事务重试条件/次数、幂等 CommitID、相同键异载荷拒绝、重复 slot/身份验证/版本检查全保留。两个活动的事件 JSON、attempt 身份、UNKNOWN/settlement/恢复顺序不变；响应丢失继续由原 commit/receipt 与恢复 owner 处理，不新增恢复服务或重新发送 provider。

**Dub 无 EXTRACT**。当前 referral 业务不是该适配器替代层；不新建 Dub wrapper、DTO alias、路由或 provider。成熟 Viper/GORM/Temporal 继续复用，go.mod/go.sum 不变。

### §4.2 用户已确认的合同取舍

| 边界 | 确认后行为 | 保持不变 |
| --- | --- | --- |
| 上表 18 个非身份旧名 | 不再成为配置来源，即使 Primary 缺失也不回读；缺值按原 YAML/default/校验处理，不扫宿主凭据 | 当前 Primary、其它已有自动环境输入、动态 client、原文件/Set 优先级 |
| 三个旧 tenant/user-ID/role 限制名 | 只要出现就按现有退休变量机制拒绝加载，包括空值和与 Primary 并存；不静默丢弃限制 | 当前 canonical 限制和 authruntime 权限算法，裸 ZITADEL/平台管理员 Primary |
| 旧 username 限制 | 保留原非空/YAML fail-closed 与无泄露语义；不变成支持的配置 | 原身份/组织/成员角色与所有现有安全拒绝 |
| 单槽保存与恢复 | 相同业务通过现有数组中的一个元素完成；旧 Go 字段编译期退出。**不承诺旧单项指针提交回执继续匹配**；历史活动若重新进入已提交的旧单项命令，可能命中原同键异载荷拒绝，不做回退或迁移 | 当前数组/run-only 指纹字节（含固定 null 占位）、SHA-256算法/格式、Snapshot/EventPayload JSON、绑定版本、事件名、CommitID/幂等键、事务次数和持久化 owner |
| 历史持久化与外部状态 | 不新建/删除/迁移表，不修改已有记录、工作流历史或 Dub 状态 | 当前 schema/grants、Temporal 当前版本/回放/取消恢复、当前 Store 观察同步 |

不设计旧配置迁移、数据迁移、双读/双写、fallback 或阶段兼容窗口。发现当前确有外部可观察兼容义务，先报告证据，由用户决定，不通过测试或 Reviewer 恢复旧合同。

## §5 整批实施矩阵与删除条件

本轮源码阶段以以下 **29 路径** 收齐一个主要 PR；数量只便于核对，不构成拆批依据。原本无须改动的当前消费者不为凑数搬家。

| 路径 | 决策及允许变化 | 保留/删除条件 |
| --- | --- | --- |
| `internal/core/config/config.go` | RETIRE Deprecated/22别名/警告与三个调用；窄改 newViper 三键自动查询 | 全部原 Primary、加载/普通日志、dotenv 和其它配置不变 |
| `internal/core/config/loader_builder.go` | EXTRACT 被警告表混放的 username 拒绝检测，留在原 helper | BuildConfig 及其它 builder 原样；不引入新配置事实 owner |
| `internal/core/config/retired_product_runtime.go` | 只增加 §4.1 三个旧授权限制变量的 name-only 拒绝项 | 原循环、大小写/出现判断、其它 keys/prefixes 和错误规则保持 |
| `internal/core/config/config_env_test.go` | 按 §6.1 转换旧别名/警告/DB fixture/Deprecated 检查 | 当前 binding、dotenv、身份/owner-scope 禁入及其它原断言保留 |
| `internal/core/config/validator_listingkit_test.go` | 保留/扩展 username 拒绝与无值泄露；加入三限制旧名跨两加载入口回归 | 原 YAML/空 Primary 等五组情形不删 |
| `internal/imageagent/projection.go` | RETIRE `ProjectionCommit.SlotMutation` 一字段 | `SlotProjectionMutation` 值类型、数组及所有其它命令/存储字段保持 |
| `internal/imageagent/store/projection.go` | 先EXTRACT当前数组指纹编码（含固定null），RETIRE helper、三调用改直接数组 | 原hashJSON、验证、事务/重试、锁/回滚、事件、幂等比较不变；不兼容旧非null指针回执 |
| `internal/imageagent/temporal/activities_persistence.go` | 仅 PersistSlotResultV3 构造改为原值的单元素数组 | PersistRunState 等其余生产函数零改 |
| `internal/imageagent/temporal/activities_recovery_v3.go` | 仅 reconcileEffectRecoveryProjection 同形转换 | 活动自己的恢复指纹、效果/结算及未知结果语义不变；旧单项commit回执边界见§4.2 |
| `internal/imageagent/store/image_set_test.go` | 原两个单项 fixture 改数组 | 未开始工作保留、授权拒绝、投影一致原断言保持 |
| `internal/imageagent/store/repository_contract_test.go` | 四个原 helper 的单项 fixture 改数组，按 §6 补当前数组边界 | 整套 Memory/GORM 合同与所有有效断言保留 |
| `internal/imageagent/temporal/manual_acceptance_test.go` | 原重启回归 fixture 单项改数组 | workflow 丢响应/重启/安全断言保持，不执行真实环境验收 |
| `tests/imageagent_organization_integration_test.go` | 仅 scope339Fixture.verifyRecoveryCommand 原单项 fixture 改数组 | 原组织/权限/凭据门禁及无副作用断言保持 |
| `internal/integration/dub/client.go` | RETIRE 全文件 | 无外部消费者；不移植 provider mutation |
| `internal/integration/dub/types.go` | RETIRE 全文件 | 无当前 JSON/hash 消费；不保留 alias |
| `internal/integration/dub/doc.go` | RETIRE 全文件 | 整包退出 |
| `internal/integration/dub/client_test.go` | RETIRE 全文件 | 全部只证明退出适配器；不删当前通用 HTTP/凭据安全检查 |
| `internal/integration/README.md` | 只删 Dub 当前列表项和专属段落 | 其它集成和归属说明保持 |
| `docs/development/listingkit-config-migration-checklist.md` | 仅将“旧LISTINGKIT别名仍绑定”当前说明改为本设计拒绝边界 | 其它原文/历史迁移资料不作为当前迁移授权，不扩大本批范围 |
| `docs/architecture/legacy-global-config-logging-tail-closure-retirement-boundary.md` | 加本批配置警告与 Dub KEEP 替代顶注 | 原文/历史/其它 KEEP 逐字保留 |
| `docs/architecture/legacy-listing-preview-download-closure-retirement-boundary.md` | 加仅 Dub KEEP 替代顶注 | 同上 |
| `docs/architecture/legacy-localagent-source-policy-closure-retirement-boundary.md` | 加仅 Dub KEEP 替代顶注 | 同上 |
| `docs/architecture/legacy-idle-provider-bootstrap-closure-retirement-boundary.md` | 加仅 Dub KEEP 替代顶注 | 同上 |
| `docs/architecture/legacy-imageagent-http-runtime-closure-retirement-boundary.md` | 加仅 Dub KEEP 替代顶注 | 同上 |
| `docs/architecture/legacy-subscription-metering-closure-retirement-boundary.md` | 加仅 Dub KEEP 替代顶注 | 当前 Resource/资金/推荐/其它依赖和原文保留 |
| `tests/legacy_config_projection_dub_retirement_test.go` | 增一个本批退休 guard，复用原 AST/文件检查能力 | 精确检查旧字段/helper/包和真实消费者闭包；不全局禁用作为拒绝输入/证据的旧名字符串 |
| 本文件 | 记录批准、实际实施/检查转换与准确例外 | 不倒推架构，不放入滚动 CI SHA |
| `docs/architecture/README.md` | 加当前入口/状态 | 既有全文作为原后缀保留 |
| `docs/refactoring/legacy-register.md` | 加当前三组 RETIRE/EXTRACT 边界 | 既有历史/计数/勾选保持 |

**零改**：当前 authruntime/server_auth/成员权限、所有有效 Config 类型和 validator 实现、图像 service/计划/领域 Snapshot/Workflow 输入、V3 ledger/资产/Resource、schema/migration/grants、当前采集/PoolManager、Store observation、workflow、inventory（含 CURRENT）、依赖/ceilings/保护规则。没有新表、SQL、索引、授权、外部 mutation 或生产配置文件修改。

## §6 验证与验收方法

设计阶段只做源码/调用链、清单和文档链接自检；生产测试/构建/运行验收 `NOT_RUN`。CI 只能证明本次三文档候选，不能证明尚未实施的退休。

获准实施后在同一批先写能捕获目标的原环境/Repository 回归与本批 guard，确认旧源码 RED，再最小实现到 GREEN。使用任务自有合成值、t.TempDir 与原 Memory/SQLite 测试；清理测试环境只针对 t.Setenv 自有值，不读现有凭据/文件。无新增 PostgreSQL/Redis/容器或真实 provider 验收环境要求。

| 当前 Must | 必要证据 |
| --- | --- |
| 22旧名不再取值，Primary仍可用 | 每项隔离旧名/Primary/冲突值合成 fixture；三个自动重入旧名必须 RED→GREEN；保留普通/未注册 AutomaticEnv、其它隐式拼写和动态 client 的代表性不变证据 |
| 身份限制 fail-closed | 原 username 五组断言保留；原三限制旧名在有/无 Primary、空/非空及 validated/WithoutValidation 两入口拒绝，错误不含合成敏感值；canonical 当前约束不变 |
| 单槽/多槽及原子性 | 原 TestRepositoryContract 两实现、单槽结果与恢复；run-only 空数组、有序多槽、重复slot/身份错误、同键异载荷/响应丢失、失败后 snapshot/event/normalized writes 回滚；先冻结原数组/run-only（含nil/空）完整编码与指纹fixture，前后字节一致、同回执重试成功；合成旧非null单项指纹继续冲突而非回退。缺少的用例加入原合同文件，不造新平台 |
| 现役活动与组织门禁 | 原 Temporal restart/recovery/cancellation 相关测试和 scope339 原断言；仅自有 fixture，原有效断言不放宽 |
| Dub与旧接口无尾部 | 精确 AST/引用与缺席检查，当前 referral/图像/配置消费者相关原检查及必要入口仅编译；禁止为编译通过加兼容 alias |

只运行与本批修改直接相关的必要开发自检及仓库原 CI；原 CI 失败查根因/同根路径，不简单重跑或放宽门禁。若遇清单外正式消费者或有效检查冲突，按 §7 提供准确补丁/影响后取得例外，不机械拆新 PR。

### §6.1 原检查的精确处置

| 文件/函数 | 允许转换 | 不能删除或放宽 |
| --- | --- | --- |
| config_env_test / `TestNewViper_BindsLegacyEnvironmentAliases` | 旧成功预期改成无旧值来源，含三个自动重入回归；新增覆盖其余旧名 | Primary精确值/当前来源优先级不变；不改 skip/环境探测绕过 |
| 同文件 / `TestNewViper_BindsDatabaseEnvironmentVariables` | DB_NAME成功fixture改用原Primary，再另证旧名不取值 | 原host/port/name精确值仍断言，不取消DB校验 |
| 同文件 / `TestKnownEnvBindingsDoNotExposeListingKitOwnerScopeToggle` | 仅 Deprecated 字段检查改为完整 binding 形状/无别名通道的等效禁入 | 原 owner-scope binding/Primary 禁令保持；其它身份配置禁入函数不变 |
| 同文件 / 两个 `TestDeprecatedEnvWarnings_*` | 退休仅警告语义的测试；由旧名无取值/Primary不被旧名覆盖的回归承接有效合同 | 不要求保留对退出警告函数的调用，不删除当前日志/无泄露测试 |
| validator_listingkit_test / `TestListingKitLegacyUsernameAllowlistInputsFailClosed`、`clearLegacyListingKitUsernameAllowlistEnv` | 只补明确旧名/自动拼写的合成fixture隔离和拒绝回归 | `TestValidateConfigRejectsLegacyListingKitUsernameAllowlistWithoutExposingValues`、`requireLegacyUsernameAllowlistRejection`及原断言保持 |
| store/image_set_test / `TestSetSlotClosurePreservesUnstartedWorkInNormalizedAndProjectedState` | 两构造换原值数组 | 不变任何结果、错误或安全预期 |
| repository_contract_test / `testProjectionCommitAtomicallyPersistsRecoveryRegistryAndOrderedSlotMutations`、`testProjectionCommitRejectsInvalidSameAttemptRecoveryMetadata`、`testProjectionCommitRejectsMismatchedSlotAttemptIdentityAtomically`、`testProjectionCommitRollbackHidesEventSnapshotAndNormalizedWrites` | 仅原单项setup改数组；数组指纹/回执字节等价与缺少的数组边界加独立用例，测试参考形状可保留固定null字段 | 当前有序多槽、失败原子性、身份/CAS、原完整断言与测试装配保持；不删除整个helper |
| temporal/manual_acceptance_test / `executeManualRecoveryWorkflowRestartAcceptance` | 只改 recorder 中单项fixture为原值数组 | 原重启、事件、恢复和provider调用约束保持 |
| organization integration / `scope339Fixture.verifyRecoveryCommand` | 只改原单项fixture | 原获准scope/权限/凭据与无发送断言保持 |
| dub/client_test 全文件 | 随无消费者适配器整退 | 不据此变更通用HTTP/身份/资金/当前推荐安全测试 |
| 原其它退休/架构/安全检查 | 零改；本批独立 guard 添加上述精确禁入及当前保留 | 不删旧fail-closed断言、不加missing-skip、不修改扫描算法/ceilings/计数；workflow/inventory完全零改 |

## §7 独立复核、停止点与授权

适用独立 Architecture Review 只核对本三组完整闭包、§4.2 合同取舍、真实消费者/owner、JSON/hash/事务和 §6.1。独立结论及准确审核 SHA 记录于同一 PR，作者自检不是独立结论；用户确认前保持 `NOT_READY`。Finding 先按 AGENTS.md 分类；最多两轮正常架构评审，真实 Must 未满足不转成 PASS。适用复核通过且用户确认 §1–§7 后，才把本批标为 `IMPLEMENTATION_READY` 并在同一主要 PR 实施。

未经用户本轮授权不默认新增 AI 互验线程；GitHub 适用复核无法给出结论时，先说明具体失败，再请求一次有界只读设计复核。设计授权与源码最终交付复核授权分开，不把前批授权沿用到本批，不重复触发已运行的同 SHA 复核。

冻结后只对实际新 BLOCKER 重开架构；理论 hardening 转实现测试/Backlog。清单外消费者、未列原检查或需要改变身份/JSON/hash/事务边界，先报告具体证据和最小准确修订，不能自行扩范围。

实施/提交/PR、准确HEAD正常合并、实际merge SHA的完整push/main CI、运行验收和Issue关闭分别授权。#301当前段新读后最小更新；从“以下保留 #726 已合并批次与历史证据，不扩大本轮权限。”起的历史后缀逐字保持，正文不超65536字符；#301保持OPEN。任何设计/源码CI结果均不授权真实环境/数据/凭据/进程操作或增加内部试用前置。
