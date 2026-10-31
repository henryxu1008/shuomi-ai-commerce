> #1033 源码退休替代说明：本文为历史证据，原文完整保留。旧通用抠主体/渲染/模型审核、OpenAI Manager/请求池/全局与成员凭据回退及通用多槽位发布的 KEEP/供应承诺由[当前完整退休边界](legacy-imageagent-generic-execution-closure-retirement-boundary.md)替代；仅抽取当前来源编辑、结果构造、单主图批准和策略值规则。当前图组/Artwork、JSON/hash、预算/UNKNOWN/恢复和 Store 观察同步保持；本注记不授权旧历史续跑、真实资源操作或增加试用前置。

> #301 / #1025 USER_CONFIRMED：用户已确认冻结4b0ea8227的§1–§7（含§4.2/§6.1）及两组49路径。Design Basis: Independent Architecture / USER_CONFIRMED / IMPLEMENTATION_READY / IMPLEMENTING；生产修改前准入6523692682已写入#301并读回。按本完整矩阵先抽取最小合同/原校验，再整退旧task/AI便利链/独占ptr，不留alias；原JSON/hash/版本/事务/身份权限/审核route/计量/UNKNOWN和当前能力、Store观察保持。原CanRead四项有效隔离行为交当前读取fixture；当前实现/自检/准确源码CI/最终复核留同一#1025，设计证据不替代源码证据。workflow/完整inventory/依赖/保护与既有资源不动，无新试用前置，#301 OPEN；合并另授。以下冻结设计完整保留。

# 旧 listing/task 与 OpenAI 便利调用链完整退休设计

状态：**DESIGN_ONLY / AWAITING_REVIEW_AND_USER_CONFIRMATION / SOURCE_NOT_STARTED / NOT_READY**。Design Basis：**Independent Architecture**。父 Issue [#301](https://github.com/henryxu1008/shuomi-ai-commerce/issues/301) 保持 OPEN。

本次只交设计和三处导航，不修改生产代码、测试或 lint。调查基线是实际 main `194ec7a41da3ecd054a4c658e59f73267860e0e0`。准确设计 HEAD、原 CI、评审与用户确认记录维护在同一 PR，不用历史证据代替当前结论。

## §1 本批做什么，以及不做什么

两组一起交付，使用一个主要 PR，不按 helper、文件或测试拆小批：

| 完整闭包 | 为什么可以退休 | 先保留什么，再删除什么 |
| --- | --- | --- |
| 旧 `internal/listing/task` | 当前没有 `CanonicalSubjectReader` 实现或调用；草稿记录和报表只借用 Actor 与两项字符串校验 | 将最小身份值与校验归还 `listing/record`，然后整退旧包、无调用的任务到商品绑定、专属测试与 README |
| OpenAI 旧便利调用链 | `Generate`、`AnalyzeImage` 没有业务调用，却被宽接口强制实现；两个 `GetStats` 只有内部转发；ptr 只供这两个旧便利方法使用 | 当前审核使用单方法 `CreateChatCompletion` 合同；退出两个宽接口、四个闲置方法和独占 `shared/ptr`，当前图像传输、请求池与治理不变 |

使用者是维护当前草稿、报表、图像审核及内部试用的开发者。可见交付是这些当前操作继续由当前 owner 完成，编译和依赖不再保活旧任务设计及无调用的 AI 便利协议。没有新增产品能力、页面或用户操作步骤。

Must：收齐生产调用、接口义务、stub、原检查、lint 和直接文档尾部；保持当前能力、Store 观察同步、身份权限、错误映射、JSON/hash/版本、事务和外部副作用控制。Should：沿用现有 owner 和检查 helper，减少旧内部概念，不加框架。

Out of Scope：新任务/读取器、兼容 alias、fallback、双读双写、第二事实源；配置改造、凭据存取/轮换、计费或 provider 语义变更；部署、真实运行验收、实例/数据/凭据/进程操作；workflow、完整 inventory（含 CURRENT）、历史计数、ceilings、依赖和保护规则修改。独立 SHEIN 登录及身份回写已在 #985 退休，不恢复；当前 Store 观察同步继续保留。不增加内部试用前置。

## §2 Design Basis、权威与前批 KEEP 替代

本批移动当前公共值合同 owner 并收窄 provider-neutral 接口，且完整闭包超过 30 路径，采用 Independent Architecture。文档提交不是生产开工准入；适用设计复核、原 CI 和用户确认完成后才能实施。

产品依据：[全新系统产品基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[当前目标架构](project-target-architecture.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)及本轮用户授权。Figma / UI Authority：N/A，不变更 UI、路由或交互。Threat Model 继续覆盖当前组织/owner 隔离、角色与令牌校验、当前事务/回执、准确审核路由和计量/UNKNOWN；没有新增 Accepted Risk，也不扩为全局治理项目。

| 前批依据 | 只替代的 KEEP | 替代依据与继续保留的边界 |
| --- | --- | --- |
| [#891 preview/download](legacy-listing-preview-download-closure-retirement-boundary.md) §3/§5/§6.1 | `task/canonical_subject.go`、测试与 README 的整体 KEEP | 只有 Actor/校验还被当前记录使用；它们抽到 record。旧 Reader/Subject/Lineage/授权 seam 没有实现/调用，不为文档制造消费者 |
| [#937 support facade](legacy-idle-support-facade-closure-retirement-boundary.md) | IntPtr/Float32Ptr 与 ptr 根 KEEP | 两个 helper 的唯一生产消费者是本批退出的 Client.Generate/AnalyzeImage；最后消费者消失后整包退出。外部 SDK 同名 helper、Retry 等共享层不受影响 |
| [#953 OpenAI adapter](legacy-openai-adapter-closure-retirement-boundary.md) §5/KEEP | Client.Generate、旧 TextChatCompleter 实现义务 | 当前审核实际只调用 CreateChatCompletion；先建立最小合同，再删除无调用义务。当前 Client/Manager/SDK/图像/审核治理保留 |
| [#983 asset/source/OpenAI tail](legacy-asset-source-openai-tail-closure-retirement-boundary.md) §4.2/§6.1 | 两个宽接口声明的逐字保活 | 仅接口声明改成单方法合同，17 个已退 alias 仍禁入；其余请求/响应字段、tag、构造器、JSON/hash/版本原样 |
| 早期 canonical inspection / provider-neutral AI / phase2 设计与计划 | 旧 Task 查询入口、Generate/AnalyzeImage、ptr 的当前保留含义 | 直接依据加准确顶注，原文作为历史保留；当前 canonical.inspect 已按 ProductKey/CatalogVersion 读取 Catalog，不添加 record 或 task 依赖 |

上述替代必须由用户确认本设计后生效。历史说明、测试名字或 detector 结果都不是恢复旧合同或删除当前能力的授权。

## §3 实际消费者与完整调用闭包

### §3.1 旧任务包

全 tracked Go 源（含测试/build-tag）调查：生产 importer 恰为 `listing/record/{record,diagnostic,collection}.go`、`app/listingrecordstore/repository.go`、`app/reportcenter/sources.go` 五个文件。它们仅消费 `Actor`、`ValidateActor`、`ValidateTaskID`，没有读取 CanonicalSubject。后者的三个错误、SourceLineage、Clone、CanonicalSubjectReader、TenantAdminChecker、CanReadCanonicalSubject 都只在旧包及其专属测试中出现。

当前链保持：可信 AuthIdentity/effective Organization → record use case 的身份/权限校验 → record.Reader/Store/CollectionReader → 原 listingrecordstore 与两个原表；报表 Sources 仍经过原 Policy/组织/owner 检查后调用 record.Reader。原 AuthIdentity/authz 继续拥有身份与权限事实，新 Actor 只是调用期值，不是目录、凭据或第二身份 owner。

七个当前测试 importer 在 §5 列齐，包括 integration build-tag 和 chain1 用例。`tests/commerce_tool_canonical_inspection_boundary_test.go` 仍遍历旧根，属于应转换的原检查，不是运行消费者。当前 canonical.inspect 实际输入 ProductKey/CatalogVersion，直接调用 Catalog.GetSnapshot；其生产/测试全部保留。

### §3.2 OpenAI 便利链

`ai.ChatCompleter` 通过 TextChatCompleter 强制 Generate/GetDefaultModel，另强制 AnalyzeImage。生产类型引用仅三处：Manager.GetClientWithRoute 返回类型、ProductImageAdapterConfig.ReviewClient、worker capabilities.adapter 的 reviewer 变量。真实审核在 `ProductImageAdapter.Review` 只调用 CreateChatCompletion，Model 来自已选择的 ReviewModel，不调用三个额外方法。

`Client.Generate/AnalyzeImage` 的固定 prompt/token/temperature 拼装无业务调用；`Client.GetStats` 仅转调 `RequestPool.GetStats`，后者无其他调用。`shared/ptr` 的 IntPtr/Float32Ptr 唯一生产 import/调用都在上述两个旧方法。没有反射、配置名称或命令注册这些方法的接线。

当前链保持：原 worker 的组织/策略/预算及 route selection → Manager 准确配置/凭据版本校验 → 当前 Client.CreateChatCompletion → 原 RequestPool/SDK → 原严格审核响应解码及计量记录。图像 Generate/Edit/async/source-edit、ImageGenerator.GetDefaultModel、client/manager Close、限流/信号量/重试、logMetrics 和安全日志都不退出。

## §4 最小 owner、持久化与安全不变量

### §4.1 两项选择性抽取

**record 身份值归 `internal/listing/record/actor.go`：** 新 `record.Actor` 仅 `OrganizationID string`、`UserID string`、`Roles []string` 三字段；OrganizationID 取原 Actor.TenantID 的同一可信值，不做数字转换、映射或推断。`ValidateActor(Actor) error` 与 `ValidateOperationID(string) error` 保留原 nil/非 nil 判定集合：组织 1–64 bytes、用户 1–128 bytes、角色 1–32 项且各 1–128 bytes、operation 1–128 bytes，拒绝空白及首尾空白，继续按 bytes 而非 rune 计限。不新增排序、去重、trim 后接受或角色授权。

五个上限与原 canonicalString 条件作为 record 私有实现，不发布新的通用验证框架。失败返回 record 现有 ErrInvalid；原调用点继续映射到相同的 ErrForbidden/ErrInvalid/ErrNotFound 等外部结果。旧 ErrInvalidActor/ErrInvalidTaskID 只被旧包测试比较，没有当前错误透传依赖；不为它们保留 alias。AuthIdentity 的 TenantID 与 EffectiveOrganizationID 相等、令牌未过期、角色与实例 Authorizer 判定仍在原调用位置。

**审核传输合同留 `internal/ai/contracts.go`：** 新 `ai.ChatCompletionClient` 只有 `CreateChatCompletion(context.Context, *ChatCompletionRequest) (*ChatCompletionResponse, error)`。上述三个生产声明和四个当前测试文件改接它，旧 ChatCompleter/TextChatCompleter 完全删除，不留 alias/嵌套旧接口。原 concrete Client 与现有请求池实现不迁移 owner；不新建 reviewer service、provider 或代理包装。

### §4.2 JSON/hash、事务、路由与副作用

- Actor 不进入持久化或 HTTP JSON；字段改名只发生在调用期值。canonicalInputHash 的匿名 wire 仍严格按原顺序编码 `organization_id, owner_user_id, product_key, snapshot_version, store_id, country, language, action, product_hash, asset_inventory_hash, rule_revision, policy_revision`，原值逐项不变。只把 actor.TenantID 取值改为 actor.OrganizationID；原 json.Marshal、sha256、`sha256:` 前缀保持。Roles 仍只参与原授权，不加入 hash。
- record 的 Input/Receipt/Record、payload/diagnostic 字节、规则与绑定版本、PackageHash/DiagnosticHash、资产排序、原错误响应、游标和读取拒绝不变；不重写已保存载荷、不迁移任何记录。SourceIdentity 四零值槽位、Source/Catalog JSON 与 hash 完全不动。
- 两个原表 `listing_shein_records` / `listing_shein_record_operations`、组织/owner SQL 过滤、回执唯一键、同 operation 不同输入拒绝、原事务/锁/并发及失败回滚不变。repository 只改 Actor 类型/字段与验证调用，不调整 SQL 或权威 owner；报表来源权限和输出 JSON 不变。
- AI 所有请求/响应类型、字段顺序/tag、UsageKnown/async/route metadata、模型/prompt/token/temperature/重试参数、凭据 cache key/version/hash 与原构造器不变；仅退两个无字段接口和四个无调用方法。审核仍用原精确 route/quote/credential/configuration binding，不转走旧便利方法或 static fallback。
- 原预算/资金/计量、取消/deadline、retry/UNKNOWN/recovery、不可重复 provider 副作用与清理 owner 原样；删 GetStats 不删活跃的 rateLimit 状态/锁、semaphore、logMetrics、Close 或安全日志。不读现有凭据，不连接 provider/数据库/对象存储，不操作既有进程。

## §5 同一主要 PR 的完整删除、改接与保留矩阵

未来源码阶段共 **49 个唯一路径：14 生产、17 测试、17 文档、1 lint 配置**。新增文件也计入；两组一起实施，不机械拆批。当前设计阶段只修改 D1/D2/D3/D15 四份文档。原 CI 已真实检出遗漏 D15，适用独立设计复核指出 D16/D17 为直接当前说明/绑定依据尾部；本次据此补齐矩阵，不修改原检查或提前实施 D16/D17。

### §5.1 生产矩阵（14）

| ID | 路径 | 精确动作 |
| --- | --- | --- |
| P1 | `internal/listing/task/canonical_subject.go` | 整退；先按 §4.1 抽取 Actor/两项有效校验，旧 Subject/Lineage/Reader/Checker/Clone/授权与错误全部退出 |
| P2 | `internal/listing/record/actor.go` | 新增 §4.1 的三个字段、两验证函数、五私有限制和私有 canonical 字符串规则；复用 record.ErrInvalid |
| P3 | `internal/listing/record/record.go` | Reader/Store/内部函数与 Create 改用 record.Actor/OrganizationID/ValidateOperationID；wire/hash 算法、权限/回执/事务流程原样 |
| P4 | `internal/listing/record/diagnostic.go` | 只改 Actor/字段/校验 owner 与 import，原超时/权限/严格诊断/错误保持 |
| P5 | `internal/listing/record/collection.go` | 同上；分页、游标和身份拒绝保持 |
| P6 | `internal/app/listingrecordstore/repository.go` | 参数类型/组织字段/两验证调用 owner 改接；原 SQL/过滤/检查/事务逐字保持 |
| P7 | `internal/app/reportcenter/sources.go` | Sources.Read 的 Actor literal/import 改接 record；原 scope/role/owner/错误/JSON 保持 |
| P8 | `internal/ai/contracts.go` | 用 §4.1 单方法 ChatCompletionClient 替代两个宽接口；其余全部 wire 声明/tag/字段原样 |
| P9 | `internal/integration/openai/client.go` | 退 Generate/AnalyzeImage/GetStats 及专属 ptr import；NewClient/CreateChatCompletion/GetDefaultModel/Close/字段/日志保持 |
| P10 | `internal/integration/openai/pool.go` | 只退 RequestPool.GetStats；全部活跃限流/重试/SDK/日志/关闭算法原样 |
| P11 | `internal/integration/openai/client_manager.go` | 仅 GetClientWithRoute 返回新接口；原 resolveClientWithSelection/缓存/版本/轮换/Close 原样 |
| P12 | `internal/integration/openai/product_image_adapter.go` | 仅 ReviewClient 字段类型改接；审核请求、严格解码、授权/observer/hash/预算原样 |
| P13 | `internal/app/worker/imageagent/capabilities.go` | 仅 reviewer 变量类型改接；route/quote/配置校验与装配原样 |
| P14 | `internal/shared/ptr/pointer.go` | 整退最后两个独占 helper，不改其他共享包或外部 SDK 同名函数 |

### §5.2 测试矩阵（17，处置见 §6.1）

| ID | 路径 | 动作 |
| --- | --- | --- |
| T1 | `internal/listing/task/canonical_subject_test.go` | Actor 与 ID 原有效用例移交 T2，旧 Reader/Subject/Clone/Checker 专属测试随旧包退 |
| T2 | `internal/listing/record/actor_test.go` | 新 owner 承接原有效验证 fixture/独立预期，不扩展验收平台 |
| T3 | `internal/listing/record/record_test.go` | Actor owner/字段改接，补固定原输入 wire/hash oracle；原规则/版本/权限/错误断言保持 |
| T4 | `internal/listing/record/collection_test.go` | 仅 Actor owner/字段、stub/import 改接，原分页/身份断言保持 |
| T5 | `internal/app/reportcenter/sources_test.go` | 同上，原 scope/owner/来源拒绝/快照断言保持 |
| T6 | `internal/app/httpapi/chain1_local_product_acceptance_test.go` | 唯一 ReadOfflinePackage fixture 改接；整个真实用例断言和原执行 gate 保持 |
| T7 | `internal/app/httpapi/shared_regression_postgres_test.go` | 唯一读取 Actor literal 改接；原共享连接/组织切换断言保持 |
| T8 | `internal/app/httpapi/shein_diagnostic_contract_test.go` | reader 类型/literal/import 改接；全部原 HTTP 错误/拒绝先于读取断言保持 |
| T9 | `internal/app/httpapi/shein_records_integration_test.go` | recordActor helper/表内 literal 改接；旧 CanRead 四项安全行为由当前 ReadOfflinePackage 原 fixture/表承接；原并发/冲突/不可泄露/体积/事务断言保持 |
| T10 | `internal/ai/contracts_test.go` | 当前接口 assertion 改接、chat stub 三多余方法退；原 JSON literal 与 image stub 全保持 |
| T11 | `internal/integration/openai/product_image_adapter_test.go` | stub 退 Generate/AnalyzeImage/GetDefaultModel；原审核/请求/计量/严格解码/副作用断言保持 |
| T12 | `internal/integration/openai/tenant_manager_test.go` | callChat 参数类型改接，全部原配置/身份/路由/轮换/隔离用例保持 |
| T13 | `internal/integration/openai/types_contract_test.go` | chat interface assertion 改接；ImageGenerator assertion 原样 |
| T14 | `internal/shared/ptr/pointer_test.go` | 仅属已无消费者 helper 的专属测试整退 |
| T15 | `tests/commerce_tool_canonical_inspection_boundary_test.go` | 转换三项旧根/文档保活检查，当前 canonical Tool 禁令/原 parser 原样 |
| T16 | `tests/target_architecture_phase2_test.go` | 转换 ptr 根保活与 provider 精确禁入；其余算法/合成反例/解码检查保持 |
| T17 | `tests/listingkit_backend_retirement_test.go` | 四项 KEEP 校准与新增一个完整闭包 guard；其它检查保持 |

### §5.3 文档与 lint（18）

| ID | 路径 | 动作 |
| --- | --- | --- |
| D1 | `docs/architecture/legacy-task-openai-convenience-closure-retirement-boundary.md` | 本设计、确认状态与准确边界；滚动 CI/HEAD/评审留 PR |
| D2 | `docs/architecture/project-target-architecture.md` | 当前导航/owner 顶注，澄清原 Task 查询描述已被当前 Catalog/record 取代；既有正文作为历史原样保留 |
| D3 | `docs/refactoring/legacy-register.md` | 当前段与两组取舍；全部原文/历史计数/勾选后缀原样保留 |
| D4 | `internal/listing/task/README.md` | 随旧包整退；当前 owner 由 D1/D2/D5 说明，不新建兼容目录 |
| D5 | `docs/development/repository-structure.md` | 只加当前 owner/退休路径顶注；原文字/目录历史保留 |
| D6 | `docs/architecture/legacy-listing-preview-download-closure-retirement-boundary.md` | 只顶注准确替代 canonical_subject/README 的 KEEP，其余 §5/§6.1 原样 |
| D7 | `docs/architecture/legacy-idle-support-facade-closure-retirement-boundary.md` | 只顶注替代 ptr 两 helper/root KEEP，原 Context DB 等全部保留 |
| D8 | `docs/architecture/legacy-openai-adapter-closure-retirement-boundary.md` | 只顶注替代 Client.Generate/宽接口义务 KEEP，其它禁入/治理保持 |
| D9 | `docs/architecture/legacy-asset-source-openai-tail-closure-retirement-boundary.md` | 只顶注限定两个无字段接口逐字保活的替代，原数据合同/JSON 保持 |
| D10 | `docs/superpowers/specs/2026-08-21-provider-neutral-ai-contracts-design.md` | 只顶注宽接口取舍及当前单方法合同，历史原文保持 |
| D11 | `docs/superpowers/plans/2026-08-21-provider-neutral-ai-contracts.md` | 同上，不重跑旧计划或旧子代理步骤 |
| D12 | `docs/superpowers/specs/2026-09-05-commerce-tool-canonical-inspection-design.md` | 只顶注当前 Catalog/record owner 与旧 Task query 退休，当前 Tool 只读行为保留 |
| D13 | `docs/superpowers/plans/2026-09-05-commerce-tool-canonical-inspection.md` | 同上，不恢复旧 Reader、数据迁移或 Task runtime |
| D14 | `docs/superpowers/plans/2026-08-30-internal-target-architecture-phase2.md` | 只顶注旧 ptr/任务合同的本批取舍，历史数字/勾选/步骤保持 |
| D15 | `docs/architecture/README.md` | 新设计加入原架构索引；原正文/历史完整保留，不改索引检查 |
| D16 | `internal/listing/README.md` | 仅当前 owner 顶注：record 承接有效 Actor/校验，旧 task 退休；独立 SHEIN 登录/身份回写已由 #985 退休，Store 观察保留；原正文后缀保持 |
| D17 | `docs/superpowers/specs/2026-09-05-shein-offline-validation-input-binding-design.md` | 仅顶注替代旧 Actor/读取/校验 owner 与旧源文件链接的当前含义；保留原 §3.1 绑定、JSON/hash/版本规则和完整后缀，#904 的 alias 退休不扩大为整份依据退休 |
| L1 | `.golangci.yml` | 仅转换 listing_task_canonical_subject_boundaries 与收窄 canonical_inspection_tool_boundaries，精确见 §6.1；其它规则零改 |

矩阵外明确 KEEP：当前 AuthIdentity/authz/所有 HTTP 路由、record HTTP handlers/schema/数据库 DDL、canonical.inspect 生产与原行为测试、全部治理 Executor/AI 路由/凭据仓库、当前图像/V3/采集/Store 观察、Logger/Retry/限流/真实锁、SDK 和依赖。所有 workflow 与名字含 inventory 的文件零改；`phase2-runtime-inventory.md` 等保留原历史记录，不因提及旧名而制造当前消费者。

## §6 同批实施与必要验证

确认前只核四文档、实际消费者、矩阵/链接、原文后缀与设计 CI，不运行生产测试/程序。源码获准后：先在原 T17 增加准确根缺席/旧声明禁入检查并确认真实 RED；T3 在旧 owner 仍存在时固定合成输入的原 wire/hash 独立 literal，再抽取/改接/删除至 GREEN。两组在同一分支连续完成，可分提交，不拆主要 PR。

开发自检限当前受影响的 record/报表/OpenAI/AI/worker 与原聚合/import/depguard 检查，相关 race，以及 current-application、image-agent-temporal-worker、report-center-schema-init 的仅编译；原 integration-tag/chain1 用例只做编译或使用原 CI 明确自有 fixture，未实际执行不得写 PASS。不新增 runner、session、临时数据库要求或故障注入平台；不人工重复原 CI 的完整检查。原 CI 最终仍须成功，源码最终独立检查不能由作者自检替代。

### §6.1 原检查的准确转换

下表是同根测试/lint 的完整处置，不是降低强度。旧实现细节可以退出，业务、安全、权限、并发、幂等、平台与确定性断言保留；原读取/遍历/AST/decoded import/错误失败关闭算法不改，不加 skip、空目录绕过或豁免。当前合同 owner 的机械替换适用于 T3–T13 列明的全部引用/fixture，不逐个函数再次拆授权。

| 检查/文件 | 允许变化 | 必须保持 |
| --- | --- | --- |
| T1→T2 / TestActorValidation、TestCanonicalSubjectTaskIDValidation | 原所有 fixture/边界/独立期望移至 record，字段 OrganizationID、ID validator 改名，错误 sentinel 改用现有 ErrInvalid；补等边界正例。旧 CanRead/Clone/ReaderContract 三测试及两 stub 退；CanRead 的四项有效安全行为按下行由当前读取 owner 承接 | 原字节长度、空白/角色数量拒绝集合；原 Record.Clone 的实际载荷防御拷贝测试保留，不保活已无消费者的 SourceLineage.Clone |
| T9 / TestSheinRecordHTTPPostgresRoundTrip 的原 ReadOfflinePackage 表 | 在原当前 auth/repository fixture 验证同组织 owner 成功、跨 owner 普通角色拒绝、配置管理员同组织成功、平台管理员跨组织拒绝四项；复用已有覆盖，缺少的只补原表/fixture，不新增 runner/权限实现或旧 Checker | 不能仅因旧函数退休而丢掉有效隔离断言；原 HTTP/SQL 拒绝与无泄露保持，测试执行 gate 和当前事实 owner 不变 |
| T3 / TestCanonicalInputHashBindsRuleAndPolicyRevisions | 只 Actor/字段替换，原版本差异断言保留；另加固定 wire 与 SHA256 literal 的抽取前后等价用例 | 原匿名 JSON 字段顺序/值、算法、前缀、规则/绑定版本敏感性，不把期望调用待测算法生成 |
| T4–T9 / 全部旧 Actor 类型/literal/stub/import | 一对一改接 record.Actor/OrganizationID，不改 fixture 值、原测试 gate 或断言；无引用行零改 | 原所有取消/owner/租户/权限/事务/并发/不可泄露/大小/回执/分页断言，特别是 repository SQL 与原数据库测试 |
| T10 / TestContractsExposeMinimalInterfaces、contractChatStub | assertion 指向 ChatCompletionClient；chat stub 只留 CreateChatCompletion；三多余方法退 | TestContractsMarshalStableJSON 整个函数/字面期望及全部 image/async 合同保持 |
| T11–T13 / productImageChatStub、callChat、TestOpenAIClientImplementsProviderNeutralContracts | 新接口类型/assertion；仅 chat stub 三多余方法退 | 原当前审核/图像/限流/route/version/quote/请求/响应 fixture 与所有断言保持；ImageGenerator.GetDefaultModel 不退 |
| T14 / TestIntPtr | 随最后消费者退出而退，不写新 helper 测试 | 所有其他共享/外部 SDK/helper 测试不变 |
| T15 / TestCommerceToolCanonicalInspectionUsesOnlyNarrowDomainPorts | canonicalinspect allow 表仅删旧 task；旧 task 根遍历改为显式 os.Lstat 缺席，不给 Tool 新增 record 依赖 | 当前 Tool 的 authz/commercetool/catalog allow 项、原 import parser 和失败关闭保持 |
| T15 / TestCommerceToolCanonicalInspectionDeclaresNoMutationOrCompatibilityOwner | 保留当前 Tool 全部 AST 禁令；旧 task 无 Repository 遍历换为读取当前 P2 单文件并断言最小 Actor/无第二 Repository，旧根缺席另验 | 不扫描整个 record 包来禁止其合法 Store；原 Tool owner/投影禁令及读取错误处理保持 |
| T15 / TestCommerceToolCanonicalInspectionGovernanceIsRecorded | 当前文档 owner token 从 internal/listing/task 改 internal/listing/record；配置 rule 名、product.canonical.inspect/ProductSnapshot/Phase 2B 原要求保持 | 原文档/配置必读与失败关闭；旧名字在历史文档出现不代表当前依赖 |
| T16 / TestSharedPackageTargetsExist | 必需目录表仅退 ptr，resilience 继续必需；ptr 完整根缺席由 T17 明确强制 | 不以目录缺失 skip 共享检查，原其余 shared/Retry 断言不变 |
| T16 / isAllowedIntegrationProviderAdapterImport、TestIntegrationProviderAdaptersEnforceRootSpecificContractMatrix | 在原 broad shared allow 前明确拒绝 ptr 根/子包；openai/grsai 的 ptr 从 allowed 移 forbidden；httpimage 原 ptr 禁令保持 | 其余 allow/deny、完整 parser/解码/路径/合成反例与原计数算法不改；转义 import 原测试保持原 fixture/3项预期 |
| T17 / TestLegacyOpenAIAdapterClosureIsRetired | required 表仅删 Client.Generate；本批新 guard 明确禁止四个闲置方法 | 原其它 required、已退生成/Manager/API 禁令与原算法保持 |
| T17 / TestLegacyRuntimeSkeletonAndIdleConfigClosureIsRetired | retained 表退 ptr 两项；原 retiredSupport 的同 owner 表追加 IntPtr/Float32Ptr | 原六项 ptr 禁令、其它全部当前包/字段/解码注册/扫描与错误处理保持；根缺席由新 guard 强制 |
| T17 / TestListingKitTaskBackendSourcesAreRetired | 原 required 表删 pointer.go 一行，改要求 P2 中 Actor/ValidateActor/ValidateOperationID | 原其它文件/CLI/当前能力全部断言保持；不减整个 required 检查 |
| T17 / TestLegacyAssetSourceOpenAITailClosureIsRetired | 精确声明表的 ChatCompleter/TextChatCompleter 两项退，加入 ChatCompletionClient 单方法准确声明 | 其余全部数据类型/构造器/字段/tag 逐字断言、alias 禁入、Catalog/Source 合同与算法原样 |
| T17 / 新增 TestLegacyTaskOpenAIConvenienceClosureIsRetired | 复用原 trackedFiles/AST/decoded import/helper：两根与直接文件显式缺席、跨 tracked Go 的旧 import 禁入；owner-qualified 禁四方法/两宽接口/旧 Subject；精确新 Actor/单方法合同及当前 Client/请求池活跃方法保留 | 不使用泛化 Generate/Actor 名字禁令，不新增扫描器/宽豁免；解析/文件读取失败关闭 |
| L1 / listing_task_canonical_subject_boundaries | 同一稳定 rule 名改为 files=**/*.go、list-mode=lax、deny 两旧根的 exact `$` 与子包 `/` 共四项；不留原 allow。T17 精确验配置 | 已退 owner 的保护变为全源禁入；其它 lint 规则和所有 workflow 不变 |
| L1 / canonical_inspection_tool_boundaries | allow 只删 task 根一项，原 gostd/authz/commercetool/catalog 保留 | strict/list-mode、files 范围及其余规则逐字保持，不扩大 allowlist |
| D2/D3/D5–D17 | 仅本批当前导航/准确 supersession 顶注；原后缀原样 | 历史勾选、数字、失败/批准证据、其它 KEEP 与旧表记录不改；任何 inventory 零改 |

若发现矩阵外真实消费者或检查冲突，先给具体当前证据及最小提案；不得擅自删安全断言、扩大清单或改 inventory/workflow。非 Blocker 的实现细化按既有 finding 分类收敛，不重开全局架构。真正改变当前权限、事务、持久化或副作用边界的缺陷，按 AGENTS 的 Blocker 标准报告。

## §7 设计复核、用户确认与停止点

设计复核核真实消费者/无接口隐含调用、最小 owner/无新身份事实、前批 KEEP 替代、原 JSON/hash/错误/事务/route 保留和 §6.1 完整性。最多两轮正常 Architecture Review；按既有五类 finding 先分类，非 Blocker 不派生新框架或设计版本链。

先使用原 GitHub 适用复核，同 SHA 已运行时不重复触发；启动失败最多一次重试。**未经用户本轮授权不默认新增 AI 互验线程；设计与源码最终复核分别绑定准确 HEAD，授权分开。** 需要独立子代理时先说明准确范围和原因，若原适用复核先完成且无待处理问题则不新增。设计复核不跑测试/构建，不操作既有资源；源码最终复核只读真实 diff 与已有证据。

四文档、原设计 CI 和适用设计复核形成可审阅结果后，交用户确认本 §1–§7（含 §4.2/§6.1）及 49 路径完整边界；未确认前 SOURCE_NOT_STARTED，不改正式生产/测试/lint。未来实施继续同一主要 PR，形成完整路径后做一次最终交付检查，修复只复核对应增量。

准确 HEAD 合并、实际 merge SHA 的完整 main CI、部署、真实运行/产品验收和 Issue 关闭分别记证/授权；不从设计确认推定合并授权，不把 CI/fixture/AI review 写成产品验收。#301 保持 OPEN，运行/产品验收 NOT_RUN；Store 观察同步与全部当前能力保留，不增加内部试用前置。
