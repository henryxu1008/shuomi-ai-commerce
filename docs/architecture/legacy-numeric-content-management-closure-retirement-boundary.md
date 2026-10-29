> #301 / #968 当前 owner 校准（[本批确认设计](legacy-shein-draft-support-closure-retirement-boundary.md) §1–§7，含 §4.2/§6.1）：四个旧包 `internal/shein/api/product`、`api/attribute`、`content`、`authorizedbrand` 及专属 adapter/Context 尾部完整退出。当前22个真实 wire 值归域内 `internal/marketplace/shein/offlinepackage/previewdata`，有效默认/custom文本与品牌清理归现有 `marketplace/shein/publishing` 纯规则，draft 显式传值；无 alias、Service 包装、品牌查询或新事实源。当前字段/JSON/hash/版本、严格解码、Builder取消/ApprovedAsset、采集及独立SHEIN登录/Store同步保持。历史 KEEP 仅在本批准确范围内被此说明替代；以下原全文/计数/勾选/证据完整保留，不恢复旧 owner，不授权运行或数据操作。

> #960 当前替代边界（仅本轮）：本文件历史 KEEP 中的旧 `internal/publishing/shein` 草稿 owner、扩展端口及私有赋值状态，现由[完整闭包设计](legacy-shein-draft-assembly-closure-retirement-boundary.md) §2/§4/§5 的矩阵精确替代。当前值/严格 codec/离线纯规则归 `internal/marketplace/shein/offlinepackage`，本地确定性组装归已有 `draft.Builder`，原平台规则/content/common 直接复用；旧 resolver、AI 扩展、可选注入、setter/alias 和四私有 map 不保留。当前 JSON/hash/绑定版本、有效规则及独立 SHEIN 登录与 Store 同步保持；无运行/数据/凭据操作或试用新前置。以下原完整正文、历史数字/勾选和证据逐字保留。

> #301 / #929 **SUPERSEDED，仅旧 SHEIN 品牌查询与错误分类闭包**：用户已确认[完整退休设计 §1–§7（含 §4.2/§6.1）](legacy-shein-brand-error-closure-retirement-boundary.md)。本文只有旧 authorizedbrand.ProductAPI/Resolver/Config、查询/匹配 helper、sherr 及 BrandItem/BrandListResponse 的保留或现役建议被准确替代；不重建已退的 Store/Task/API 调用。完整 Resolved/context、当前清理/草稿 JSON/有效规则、采集、独立 SHEIN 登录与 Store 同步及原 owner/claim/fence/UNKNOWN 保留。无生产 EXTRACT 或替代 Resolver，不改 workflow/完整 inventory，不操作既有实例/数据/凭据/进程，不增加试用前置；#301 OPEN。以下完整原文/顶注/历史数字/勾选/批准/证据逐字保留。

> 当前适用边界（#301 / #912）：用户已确认并按[旧 SHEIN 闲置策略与辅助合同完整退休闭包](legacy-shein-policy-contract-closure-retirement-boundary.md) §1–§7（含 §6.1）整批实施，仅替代本文件中generationtopics 纯 catalog/overlay/汇总的 KEEP，最后生产消费者缺席后连同专属测试退出；当前内存词表、清理/品牌/Prompt 保持及对应旧 KEEP 检查；准确删除/保留矩阵以新设计 §2/§5/§6.1 为准，其它当前 API/有效规则/采集/独立 SHEIN 登录与 Store 同步不变。下方原历史正文、数字和证据逐字保留，不提供旧 JSON 迁移或运行/数据操作授权。

# 旧数字内容配置、注入与持久化：完整退休闭包边界

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

> Status: **CONFIRMED / SOURCE_IMPLEMENTED，待最终独立交付检查与合并决定。** Design Basis: **Independent Architecture / IMPLEMENTATION_READY**。用户已确认冻结设计 `d139bcccc250d4e5a3ccf2f0541ead6b70b35a4e` 的 §5、§6、§6.1、§7 并授权本 PR 整批实施；准确设计第二轮独立复核 completed、无 findings/未解决项，Issue Ready 已先于生产修改登记并读回。以下矩阵/停止规则为该冻结依据，§7.1 的设计期停止点已满足；无新的产品/事实/安全边界。调查及开工基线 main `5151c86e844898255f9cad71284cdac3f5716f8d`（2026-10-19）。准确实现 HEAD/CI/review 留在 PR #833；合并、实例/数据、部署与用户验收分别授权，不以源码退休宣称物理对象删除或生产验收。

## 1. 一个用户结果、一个完整 Delivery Batch

使用者是维护当前采集、草稿/内容清理、Store 观察和独立 SHEIN 登录/Store 同步的操作者。结果是完整退出没有当前运行入口的旧数字内容配置：词库、话题策略/覆盖仓库、读取/CRUD、全局注入、数字 tenant 解析、专属 schema/迁移及调用尾部一起退休；当前默认清理、授权品牌和纯规则沿正确 owner 保留。一个 Writer、一个主要 PR，不按三族、读写、schema 或测试机械拆批，不增加试用前置。

初版0c065634e只退管理写面，仍保留三项可注入读取。独立 finding [4314450056](https://github.com/qq550723504/task-processor/pull/833#discussion_r4314450056) 指出这会留下休眠 owner；分类沿原 **BACKLOG**，不是安全/试用 BLOCKER。本设计尚未冻结，现为满足用户完整闭包要求，在同一 PR 的一次增量中处理；不另开产品/治理项目。

根因：测试能调用构造器、Set* hook 和 AutoMigrate，不代表当前产品需要这套数字持久化。全仓 tracked Go（根/hack/tools）显示三族构造/迁移只有测试调用；生产仅定义、不调用五个 Set* hook，两个模块的 repository 变量默认 nil。继续保留并无当前产品依据，补建注入/UI 则是未经请求的新功能。不能用测试 fixture 反向恢复旧 owner。

## 2. Design Basis、Product Authority 与 Threat Model

**Independent Architecture**：覆盖原配置/事实/schema owner 的源码退休与当前清理接线；超过30个范围文件是风险提示，不是拆 PR 目标。必须在准确设计获独立结论、用户确认、Issue IMPLEMENTATION_READY / Ready 与实施授权之后编码。

Authority：[Greenfield 决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[#824](legacy-numeric-listing-support-closure-retirement-boundary.md)、[#827](legacy-shein-publishing-runtime-closure-retirement-boundary.md)、[#831](legacy-numeric-store-management-closure-retirement-boundary.md)及用户完整闭包决定。旧三族整体保留表述仅在本矩阵上被替代。UI/IA/Figma N/A，无新产品入口。

Must：保留当前默认词库/清理、CleanupContext/品牌保护、有效纯话题规则、草稿/严格 codec/validator、平台 API/DTO/有效规则、两种 SHEIN 属性、PriceManager/PriceCalculator、AI/Prompt/图像/采集、官方 Kernel、Store 观察、原 Store/API 与独立 SHEIN 登录/Store 同步。保留当前身份/tenant/owner/admin、Store schema/写入/错误/恢复及必要安全检查。不能将“话题规则保留”变成自动启用全部话题或新增默认禁词。

Should：完整闭包，复用既有 content 工厂、纯 generationtopics 规则与现有有效测试；单 Writer/主要 PR。无新增 Accepted Risks。

Out of Scope：新内容管理 UI/API、当前 Organization 配置持久化、注入/新事实源、新接口/兼容/fallback/双读双写、旧 ID 映射/迁移；新增状态/事务/恢复/权限/计费/副作用；实例/已有表列行、Redis/Profile/凭据/provider/Prompt 记录操作；部署/关父 Issue、workflow/依赖/ceilings/保护变更或验收平台。

Threat Model：把 dormant 读接口当当前能力而留下半截旧 owner；删 fixture 时丢失真正的品牌/字段/规则断言；为保测试给默认产品添加新禁词/自动策略；误删 Store 的共享 schema/权限或把源退休当物理删数据。不得扩大为新的配置治理/IAM/恢复系统。

## 3. 实际路径：contract → implementation → injection → consumer

| 路径 | 当前证据 | 决定 |
| --- | --- | --- |
| 三族 listingadmin Repo/row/AutoMigrate | 包外生产只有下两模块的读取；constructor/AutoMigrate/Set* 调用只有隔离测试，无当前 runtime/HTTP/CLI 装配 | 三族整退，不能以无入口测试注册作为保留依据 |
| shein/submitprep/sensitive_words.go | NewSensitiveWordServiceForContext 先建 content.NewSensitiveWordServiceInMemory，再尝试 nil repo；三个 Set*/mutex、分页/状态、数字 tenant 解析全为旧配置注入 | 包与专属测试整退；当前默认内存行为直接复用 content |
| publishing/shein/generation_topic_runtime.go | 两个 Set*/mutex、数字 tenant 解析、话题 key/override 查询与提示词 block；默认 repo=nil→空 block | 整退；title_resolution 只删两条旧 block 追加，原模型请求/提示词正文/预算/超时不变 |
| submit_prep_sensitive_adapter → sanitizeSheinListingCopy / SanitizeDraftPayloadSensitiveContent | 原 adapter 仅转到上述 factory；无注入时实际服务始终是原 content 内存工厂 | 直接调用原 content 工厂，不新造服务；去 factory 的无用 context 参数，两处调用同步；ensureSanitizerContext/授权品牌仍消费原 runtimeCtx |
| content 与 generationtopics | content 是当前实际清理 owner；generationtopics 是已有纯值 catalog/解析/overlay/汇总规则，无 DB/tenant owner | 全部生产零改；纯规则保留不声称当前有话题管理产品，也不把 dormant DB overlay 变为默认行为 |
| Store/worker/managed/TEMU | #831 已保留的原账号持久化、四 repository 方法、九 StoreAPI、查重/同步与共享权限/schema | 全部能力及生产/有效测试保持；listingadmin 包不能整删 |

生产 Set* 引用与调用、构造/AutoMigrate、三张表名及 schema helper 的全仓调用边均已核对。没有以真实实例、配置或数据访问补证。若后续出现具体当前外部可观察合同，按§7停止对应变更，不自行兼容；目前没有该证据。

## 4. 持久化、身份与副作用 owner

- `listing_sensitive_word` / `listing_generation_topic_policy` / `listing_generation_topic_override` 的旧 Go model、读写/schema/迁移 owner 全部退出，不转交新 owner。**物理表/列/行存在性和值 UNKNOWN；不 DROP/ALTER/回填/查询或清理真实数据库。** 新二进制不再提供这套 dormant 管理/注入协议，不操作已有运行二进制。
- 原 Store owner 不变：Store/listingStore/StoreQuery、GormStoreRepository/StoreAPI、store_access、owner_constraint、ensureOwnerAuditColumns、applyOwnedTenantQuery/normalizePage/findPagedRows、身份/角色/admin scope 及 StoreID/Status 写入保护保留。只删失去最后消费者的 requireOwnerUserID/专属错误与 generic take/update helper。
- 当前清理默认等价：旧 repo 全 nil 时，原 factory 的结果就是 content.NewSensitiveWordServiceInMemory。新接线直接复用它，不增加词、文件持久化、goroutine 生命周期或 provider 行为；原 CleanupContext、SourceBrand/AuthorizedBrand、语言/SKC/属性清理不变。
- 旧数字配置的 enabled/status/分页/跨 owner/unique/schema 测试随整个旧 owner 退出；它们不是当前 Organization 权限或 Store schema 的 owner。当前 Store/authz/worker 租户和错误保护不变，不把旧配置迁成新事实。
- 原话题纯值规则继续存在于 generationtopics，无租户映射、全局 repo、DB 或新生产消费者。其定义/排序/unknown key/overlay/词库汇总在原 owner 上保留；不新造 resolver/配置 UI。
- 无新的跨库/事务/状态/恢复/幂等或外部副作用步骤。超时/取消/重试/UNKNOWN/授权撤销仍由当前各 owner 负责；不新增保证，不执行平台、模型或运行操作。

## 5. 完整删除、保留与复用矩阵

### 5.1 十个整退生产文件（附录 A）

三族八文件全部退出：repository/interface、DTO/query/page、三 row/TableName、constructor、AutoMigrate、CRUD/读方法、JSON 编解码/normalizer、duplicate/schema 专属辅助都不留空壳。再整退两个无注入的 runtime 模块及其数字 tenant/global setter/overlay/提示词尾部。不能只退 CRUD 留休眠读/schema，不能留 no-op setter 或改名包装。

### 5.2 六个 mixed 生产文件

| 准确文件 | 唯一允许的变化 | 必须保留 |
| --- | --- | --- |
| internal/listingadmin/access_scope.go | 退 requireOwnerUserID / ErrOwnerUserIDRequired，清 unused errors import；WithOwnerUserID 注释可准确说明原可信身份上下文 | 其余全部函数/开关/角色/admin/owner context 原样，WithOwnerUserID 函数行为原样 |
| internal/listingadmin/repo_paging.go | 退 takeOwnedTenantRow / updateOwnedTenantRow 和 unused import | applyOwnedTenantQuery、normalizePage、findPagedRows 完整原样 |
| internal/listingadmin/schema_migrate.go | 退两个 postgresColumn* type 与除 ensureOwnerAuditColumns 外七个专属函数（附录 B），清 unused import | ensureOwnerAuditColumns 完整原样，owner_constraint.go 全部零改 |
| internal/publishing/shein/title_resolution.go | 仅删两条 systemPrompt += tenantGenerationTopicPolicyPromptBlock(ctx) | 两原模型入口、基础提示词/JSON/预算/身份/ctx/超时、其余规则原样；不补替代话题配置 |
| internal/publishing/shein/submit_prep_sensitive_adapter.go | 删除 submitprep/context import；newSheinSensitiveWordSanitizer 改无参数，直接返回 content.NewSensitiveWordServiceInMemory() | 原 sanitizer interface、当前 content owner；无新 wrapper/port/global setter |
| internal/publishing/shein/sensitive_content_sanitizer.go | 两处 newSheinSensitiveWordSanitizer(runtimeCtx) 改无参 | ensureSanitizerContext/runtimeCtx/品牌、所有字段/数值/属性/本地化/SKC 清理与判定完整原样 |

### 5.3 保留、复用与禁止范围

整个 content、generationtopics、authorizedbrand 生产代码零改；现有正确行为已在当前 owner，无需新增生产抽取。纯规则不可因其旧持久化调用方退休而误删，也不可为了保留 dormant owner 增加新调用。默认 factory 直接复用现有实现，不发明业务语义。

Store/API/worker、TEMU/SHEIN/其它平台 API/DTO、listingruntime 三文件/四共享值合同、PriceManager/PriceCalculator、两种 SHEIN 属性、AI/Prompt/图像/采集、当前 Organization/Store 观察、草稿/codec/validator/官方 Kernel 全部保持。除§5.2外的 retained 生产文件零改。

Legacy decision: **RETIRE**。Reusable behavior：默认内存清理工厂和纯规则沿已有 content/generationtopics owner 保留；无新生产 EXTRACT。Current owner：当前清理/草稿与 Store/worker 各 owner；旧三族事实/schema owner 无继任。Cutover/deletion condition：确认/独立准入/Issue Ready 后，原 guard RED→同批完整删除/有效断言转接 GREEN→原 CI/最终独立检查；不做运行或数据切换。

## 6. 实施和验证：仍在同一主要 PR

先按准确文件/声明扩展原 guard 捕获 RED，再整批删除/接线/测试转接。不得为了测试复活旧 repo/DDL/tenant mapping、给默认产品新增禁词/启用策略，或建设新 fixture/runner/framework。

### 6.1 测试与两套检查的准确处置

| 原位置 | 退休/转接与保留断言 |
| --- | --- |
| 附录 A 八个旧专属测试文件 | 整退旧 DB/注入/分页/唯一约束/CRUD/ownerless-write 协议。话题纯值断言按下面转到原规则 owner；当前 Store/authz/约束测试不能跟随删除 |
| listingadmin/owner_scope_test.go | 退三个 TestRequireOwnerUserID*；WithOwnerUserID 测试改直接检验 requestUserIDFromContext，保留可信上下文行为 |
| listingadmin/repo_paging_test.go | 只退两个 generic take/update NotFound 测试；三个 paging/active/tenant/owner/admin 原测试完整保留 |
| listingadmin/schema_migrate_test.go | 只退两个旧列迁移测试和仅其使用的 intPtr helper；TestOwnerUserIDConstraintSQLRejectsBlankValuesAndIsIdempotent 原样保留 |
| publishing/shein/submit_prep_test.go | 整退三个旧 Repo stub/其引用（若声明位于其它整退文件随文件退出）、仅旧策略断言使用的 promptPolicySection helper，以及两项 IncludesTenantGenerationPolicyText、两项 LoadsTenant*Repository/Override 注入测试；OmitTenantPolicy 测试转为当前基础提示词无旧配置 block，原 recordingChatCompleter 保留。三项当前 draft 字段/品牌/free-text-SKC 清理测试保留，下述 fixture 转接，不删有效断言 |
| publishing/shein/listing_copy_test.go | 只退 LoadsTenantSensitiveWordsFromRepository / DifferentTenantsLoadDifferentGenerationTopicLexicons / LoadsTenantGenerationTopicOverrideLexicon 三测试；CleansSensitiveWords 去 DB setter，用当前默认词；其余实际标题/AI/描述/不伪造fallback测试原样 |
| content/cleanup_context_test.go | 现有测试原样；将旧 fixture 的明确自定义短语“bpa free”“whimsy”清理能力在原内存服务+AddStaticSensitiveWordsByLanguage 上直接保留，不走 DB/tenant/global hook。复用原 Close/测试模式，不创建新工具 |
| generationtopics/registry_test.go（唯一新增测试文件） | 将旧 runtime/overlay 测试中的 children/baby 指令、unknown key 忽略、override 词/指令及多语言汇总断言转到原纯 Resolve/Assemble/Collect 方法，保留原 caps/去重行为；不测试不存在的持久化或数字租户配置 |
| tests/listingkit_backend_retirement_test.go | 原 guard 增加整退文件/包、schema/generic 声明与禁止复活断言；只移原 required table 中 submitprep/sensitive_words.go 那一项，改要求当前 adapter 调原 content 工厂；Store/worker/平台/纯规则等其余原 required 与安全检查不变 |
| tests/import_boundaries_test.go | **仅**将 TestPublishingSheinSubmitPrepUsesOnlySensitiveWordAdapter 改为检查同一 adapter 直接依赖当前 content 内存工厂且无 submitprep/listingadmin 注入/依赖；旧四 selector allowlist退出。其它测试、AST扫描/构建目标/边数/阈值不变，不skip/放宽 |

fixture 具体边界：原端到端字段覆盖测试使用默认词“amazon”及“trademark/copyright”替代只由旧 DB fixture 注入的“bpa free”；字段覆盖、结构数值不变和 Amazon Basics 授权品牌保留断言必须维持。自定义短语的真实清理能力转到 content 的显式内存配置测试，不能把 BPA 等词偷偷加入生产默认词库，不能用空 fixture/删除断言假装等价。

源范围合计：十整退+六 mixed 生产；八整退+八 mixed 测试（含两个原 guard）+一个直接规则测试文件。属于一个完整结果，即使超过30文件也不机械拆 PR。未列出的测试全部零改，尤其 Store/worker/TEMU、默认 owner 开关、当前内容/品牌/codec/官方执行安全检查。

必要验证：完整矩阵/保留声明与源码调用边核对，原 scoped guard RED→GREEN；listingadmin、publishing/shein、content/generationtopics、sheinlogin/bootstrap/managed/worker、TEMU 相关原测试；现有当前命令依赖 guard 与 current-application / shein-login-worker / product-listing-api-schema-migrate 仅编译；稳定候选原 CI。复用未变化证据，只读设计不执行这些实施检查。

未来仅窄校准本设计、README、legacy-register、module-target-mapping、#824/#827/#831 对应保留说明。历史矩阵/计数/回执原文保留，旧提案0c065634e与其CI/复核不冒充新设计结果。

**两套及全部 workflow、phase2-runtime-inventory 当前/历史行及计数、依赖、.golangci.yml、ceilings/保护零改。** 唯一 import guard 增量已明确上表；没有 core/logger/OpenAI CURRENT 边退出，access_scope 仍为 Store 消费身份上下文。.golangci 原禁止旧 tenantctx 的规则保持，原 absence guard 阻止已退包复活，不新增豁免。

Runtime/provider/user/production acceptance **NOT_RUN**；设计/源码、fixture/编译/CI/AI复核不替代运行与用户验收，不增加试用前置。

## 7. 确认、准入与停止点

1. 当前只改设计/索引，不修改生产/测试、不运行构造/schema/应用/测试。准确增量的独立设计结论、用户确认后，Issue 记 Independent Architecture / IMPLEMENTATION_READY / Ready；获明确按§5/§6/§6.1/§7在本 PR 实施的授权后再编码。
2. 单 Writer/主要分支/worktree。原同 HEAD GitHub 检查运行时不重复触发；未经明确授权不新增子代理/Writer/AI互验。此为首轮 finding 的同 PR 设计增量，正常架构检查最多两轮；非 Blocker 不无限扩文档/威胁模型。
3. 发现当前真实 constructor/setter/表的生产消费者、当前外部可观察配置合同，或必须改变保留身份/Store/schema、默认规则、状态/事务/恢复/副作用时，停止对应改动并提供证据；不静默删除活跃能力，不自加兼容/新产品。
4. 实施收敛后一次最终独立交付检查；修复只审相关增量。准确 HEAD 合并、matching main CI、部署/真实操作、用户验收分别授权和记录。
5. 保留独立 worker/数字 Store 身份债务，不宣称#29/#301全仓清零；本闭包不成为当前内部试用的新前置。

## 附录 A：整退准确文件

```text
10 production:
internal/listingadmin/sensitive_word.go
internal/listingadmin/sensitive_word_repository.go
internal/listingadmin/sensitive_word_model.go
internal/listingadmin/sensitive_word_query.go
internal/listingadmin/generation_topic_policy.go
internal/listingadmin/generation_topic_policy_repository.go
internal/listingadmin/generation_topic_override.go
internal/listingadmin/generation_topic_override_repository.go
internal/publishing/shein/generation_topic_runtime.go
internal/shein/submitprep/sensitive_words.go

8 exclusive tests:
internal/listingadmin/sensitive_word_model_test.go
internal/listingadmin/sensitive_word_query_test.go
internal/listingadmin/generation_topic_policy_test.go
internal/listingadmin/generation_topic_policy_query_test.go
internal/listingadmin/owner_scoped_repository_test.go
internal/publishing/shein/generation_topic_runtime_test.go
internal/publishing/shein/generation_topic_policy_integration_test.go
internal/shein/submitprep/generation_topic_overlay_test.go
```

## 附录 B：mixed 中准确退休声明

仅 internal/listingadmin：access_scope 的 requireOwnerUserID / ErrOwnerUserIDRequired；repo_paging 的 takeOwnedTenantRow / updateOwnedTenantRow；schema_migrate 的 postgresColumnDefinition / postgresColumnTypeMigration，以及 sensitiveWordLegacyColumnMigrations、ensurePostgresColumnTypeMigrations、lookupPostgresColumnDefinition、postgresColumnDefinitionNeedsTypeMigration、normalizePostgresColumnTypeName、normalizeExpectedPostgresType、ensureUniqueIndex。ensureOwnerAuditColumns 与所有当前 Store/owner context 辅助不得按同名/同文件误删。
