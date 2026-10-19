# 旧数字词库与话题管理：完整退休闭包边界

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

> Status: **DESIGNING；待适用独立检查与用户确认，不授权生产实现。** Design Basis: **Independent Architecture**。本轮只交付设计与索引。调查基线 main `5151c86e844898255f9cad71284cdac3f5716f8d`（2026-10-19）；#831 已按批准 HEAD 合并，对应 push/main CI39257253551 SUCCESS。准确设计 HEAD、CI、审核结论留在主要 PR；未完成项不写 PASS。

## 1. 用户结果与一个 Delivery Batch

使用者是维护当前采集、草稿/内容规则、Store 观察及独立 SHEIN 登录/Store 同步的操作者。本批移除已经没有产品入口的旧数字词库、话题策略/覆盖管理面及其专属共享写辅助；三项有效规则读取、内容清理/话题汇总、原持久化与安全边界保持。核心内部试用不等待本批，不新增操作入口或试用前置。

一个完整闭包、一个 Writer、一个主要 PR：三族管理方法、列表/编辑 DTO、专属映射/审计/查询/编码 helper、失去最后消费者的 generic 行写入/按 ID 读取辅助和专属测试一起处理。不能按词库/策略/覆盖、CRUD、DTO 或 helper 继续机械拆批；共享的合法读取与 schema 合同是明确保留边界，不是下批删除承诺。

根因是过宽 Repository 把实际规则读取与旧管理 CRUD 捆绑，测试又用生产创建方法准备规则 fixture。包被当前源码引用，不能证明其全部管理能力仍是需求；测试依赖也不能反向复活旧管理 owner。本批不新造 reader/facade，也不把旧数字配置接入当前 Organization/Store。

## 2. Design Basis、产品决定与 Threat Model

分类 **Independent Architecture**：收窄三种 Repository 公共合同并退休孤立 owner 写辅助，需要先明确当前读合同、权限与 schema 边界。设计依据是 [Greenfield 决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[#824 支持闭包](legacy-numeric-listing-support-closure-retirement-boundary.md)、[#827 Publishing 闭包](legacy-shein-publishing-runtime-closure-retirement-boundary.md)及[#831 Store 闭包](legacy-numeric-store-management-closure-retirement-boundary.md)。本设计只在 §5 的准确对象上替代旧的整族保留表述。UI/IA/Figma N/A；不新增产品能力。

Must：保留三项原规则读取及全部现用词库/话题/覆盖行为，原 schema/唯一约束/owner 安全辅助；保留当前采集、图像/AI/Prompt、草稿/严格 codec/validator、平台 API/DTO/有效规则、两种 SHEIN 属性、PriceManager/PriceCalculator、共享值合同、官方 Submission Kernel、Store 观察、独立 SHEIN 登录及其 Store 同步。保留原 tenant/owner/admin/active-row、分页、错误与未配置行为。

Should：复用原合同/规则/测试与检查，单 Writer 完整闭包。无新增 Accepted Risks。

Out of Scope：新的内容管理 UI/API、当前 Organization 内容配置 owner、schema/表/列/索引变化、新事务/恢复/权限机制；补全生产注入、迁移/旧 ID 映射、兼容包装/fallback/双读双写、第二事实源；实例、已有表/行、Redis/Profile/凭据/provider/Prompt 覆盖记录操作；部署、关闭父 Issue、workflow/依赖/ceilings/保护变化或验收工具。

Threat Model：按同包名误删有效规则/共享读取；把创建 fixture 的旧测试当保留写入口的理由；删旧 owner 写校验时连带放宽当前读权限或 schema 约束；将源码能力/测试注入冒充现用产品配置、数据删除或运行验收。不扩大为新的 IAM、配置治理、事务或极端故障项目。

## 3. 实际消费者：contract → implementation → injection → consumer

| 原合同与实现 | 实际消费者/输入 | 本批决定 |
| --- | --- | --- |
| SensitiveWordRepository → GormSensitiveWordRepository.ListSensitiveWords | submitprep.loadTenantSensitiveWordRecords：数字 tenant、Status=1、Page/PageSize，200 分页；再按语言/大小写去重加入原内存词库 | 保留方法、SensitiveWord/页/持久化投影、TenantID/OwnerUserID/Page/PageSize/Status 和全部读取行为；只退管理搜索字段 |
| GenerationTopicPolicyRepository → ListEnabledTopicKeys | submitprep 的话题词库 overlay 与 publishing/shein 的提示词策略摘要：tenant/platform、enabled+active、distinct key 排序 | 只保留该接口方法；函数完整原样，包括既有 tenant-global、不按 owner 过滤语义，不能偷偷改成 owner-only |
| GenerationTopicOverrideRepository → GetGenerationTopicOverrideByTopicKey | 上述两个原 consumer：tenant/platform/topicKey → Status、附加提示词/多语言词库；原 owner/admin 条件 | 只保留该接口方法；错误/active-row/owner/trim、原读取值与 JSON decode/normalize 全部原样 |
| submitprep / publishing 的原 Set*Repository、current*Repository 与 mutex | 原显式注入能力；无 repo、无合法数字 tenant、读取错误/无记录时沿原行为；现有隔离测试显式注入 | 生产文件零改。不把这些静态 setter 当已启用的运行接线，也不因生产当前未配置就连带删除有效读取 |
| 三个 NewGorm*/AutoMigrate* 与三张 listing* row model | 原 schema/约束和隔离 fixture 的同一 owner；当前根/hack/tools tracked 生产源码没有新装配调用 | 原样保留，不执行。保留源码不宣称真实库存在、可管理或已初始化；不补新的启动/导入入口 |
| listingadmin Store/StoreAPI → 原 SHEIN worker/managed/TEMU client | #831 保留的四项 repository 方法、FindStoreByID、九 StoreAPI 方法、账号读取/缓存/查重/Store 同步 | 全部生产和测试零改，不修改 worker 身份/恢复/外部副作用 |

调查覆盖全仓 tracked Go（含 hack/tools 嵌套模块）：§5 的 17 管理方法只有声明、旧方法互调和测试消费者；五个退出 DTO 无包外生产使用。包外生产调用仅消费三项保留读方法。requireOwnerUserID / takeOwnedTenantRow / updateOwnedTenantRow 的最后生产消费者全部落在本闭包；WithOwnerUserID 仍能为当前读取注入可信上下文，保留。

架构事实必须如实交接：当前规则读取保留了旧数字 tenant 和显式全局注入合同；这不是当前 Organization 的新事实源，也不能声称已有内容管理产品可用。将其改成新的配置管理系统是另一个产品决定，不是本退休设计或内部试用的前置。

## 4. 持久化、权限、状态与副作用 owner

| 边界 | 保留 owner / 不变量 | 允许效果 |
| --- | --- | --- |
| listing_sensitive_word / listing_generation_topic_policy / listing_generation_topic_override | 原 listingadmin 持久化 owner；三张 row model 全字段/tag/TableName 保留，原 schema_migrate、owner_constraint、duplicate 检查/unique index 辅助保留 | 只退源管理合同，绝无 DDL/DML/回填/删除/迁移或运行切换；已有数据存在性/值 UNKNOWN |
| 词库读 | applyOwnedTenantQuery、owner context/admin、deleted=0、Status、分页 20 默认/200 上限及语言汇总 | 读实现仅移五项管理筛选条件；其它 scope、顺序、总数/分页/错误保持 |
| 策略 key 读 | 原 tenant/platform/status=1/deleted=0、distinct/order；tenant-global 按原合同保留 | ListEnabledTopicKeys 完整原样，不增加缓存/身份推断或权限分支 |
| 覆盖读 | 原 tenant/platform/topicKey/active-row + applyOwnerScope；ErrGenerationTopicOverrideNotFound 与 DB 错误区分 | GetGenerationTopicOverrideByTopicKey、toGenerationTopicOverride 和 decode/normalize 完整原样 |
| 当前 owner 上下文 | WithOwnerUserID、WithRequestRoles、requestUserID/roles、authz admin 规则、默认 ownerScopeRequired | 只退不再有写消费者的 requireOwnerUserID/专属错误；不降低开关、角色、空身份读取或任何 schema owner 约束 |
| 现用 Store 写入 | #831 的 StoreID/Status 原专用 helper、RowsAffected/error、tenant/owner/active-row | 完全零改。旧 generic update helper 消失不取消现用 Store 的写入保护 |
| 外部副作用 / 恢复 | 原 worker、平台 API、官方 Submission owner；三项只读配置不增加 provider 调用 | 无新状态机、事务、retry/UNKNOWN/恢复 owner、预算/结算或不可重复副作用 |

超时、取消、响应丢失、重启与并发沿原读取/调用协议；不增加新的持久化步骤或跨库一致性边界。原故障/缺配置时的结果不被改成成功，不增加兼容路径或新的恢复保证。

## 5. 完整删除、保留与复用矩阵

### 5.1 十个 mixed 生产文件；无整生产文件删除、无新增生产抽取

| 准确文件 | 允许退休 | 必须原样保留 |
| --- | --- | --- |
| internal/listingadmin/sensitive_word.go | SensitiveWordRepository 五方法；ErrSensitiveWordNotFound；SensitiveWordQuery 的 Word/Language/Tags/Level/Remark 五字段 | ListSensitiveWords 签名；SensitiveWord、SensitiveWordPage、listingSensitiveWord 全部字段/tag/TableName；Query 的 TenantID/OwnerUserID/Page/PageSize/Status |
| internal/listingadmin/sensitive_word_repository.go | 同五 concrete 方法及 unused import | constructor、AutoMigrateSensitiveWordRepository、ListSensitiveWords 完整原样 |
| internal/listingadmin/sensitive_word_model.go | listingSensitiveWordFromSensitiveWord、applySensitiveWordDefaults、applySensitiveWordAuditFields | toSensitiveWord 的全部直接投影，不把旧写入 trim/default 迁入读路径 |
| internal/listingadmin/sensitive_word_query.go | applySensitiveWordQuery 内五退休字段对应条件 | findSensitiveWordRows、原 owner/tenant/active/status/paging；只删除条件，不重写其它逻辑 |
| internal/listingadmin/generation_topic_policy.go | 三 DTO、ErrGenerationTopicPolicyNotFound；六接口方法；toGenerationTopicPolicy、from helper、audit helper、find/query helper 共五声明 | Repository.ListEnabledTopicKeys 签名；listingGenerationTopicPolicy 全字段/tag/TableName |
| internal/listingadmin/generation_topic_policy_repository.go | 同六 concrete 方法 | constructor、AutoMigrate、ListEnabledTopicKeys、duplicate key 类型/预检完整原样 |
| internal/listingadmin/generation_topic_override.go | Query/Page 两 DTO、六接口方法；from/audit/query helper、两个 encode helper 共五声明 | GenerationTopicOverride、row 全字段/tag/TableName；ByTopicKey 签名、ErrGenerationTopicOverrideNotFound、toGenerationTopicOverride、两个 decode、两个 normalize 完整原样 |
| internal/listingadmin/generation_topic_override_repository.go | 同六 concrete 方法 | constructor、AutoMigrate、GetGenerationTopicOverrideByTopicKey、duplicate key 类型/预检完整原样 |
| internal/listingadmin/access_scope.go | requireOwnerUserID、ErrOwnerUserIDRequired，仅清 unused import | 所有 context/roles/admin/read scope、开关、WithOwnerUserID 函数原样；其注释可准确改为可信读取上下文，不能改行为 |
| internal/listingadmin/repo_paging.go | takeOwnedTenantRow、updateOwnedTenantRow；unused context/errors import | applyOwnedTenantQuery、normalizePage、findPagedRows 完整原样；Store 使用的其它 helper 零改 |

总计准确 **17 concrete + 17 interface 方法、16 helper、3 error var、5 DTO、5 query 字段**；按附录 A/B 的 receiver/type 定界。不能用 GetGenerationTopicOverride 前缀误删 GetGenerationTopicOverrideByTopicKey，或按 Word/Language 等同名字段全仓替换。

### 5.2 保留能力与边界

- 三项原只读方法、SensitiveWord/Override 读值、三张 row model、三 AutoMigrate、duplicate/unique/owner/schema 辅助保持，不新增当前 schema/初始化链，不执行它们。
- submitprep/sensitive_words.go、publishing/shein/generation_topic_runtime.go 及其它所有包外生产文件零改；内存词库、generationtopics registry、提示词摘要/overlay、品牌保护和当前 sanitizer/validator 的有效规则不动。
- 整套 Store/API/独立 SHEIN worker、TEMU/SHEIN/其它平台 API、listingruntime 三文件/四共享值合同、PriceManager/PriceCalculator、两种 SHEIN 属性、AI/图像/采集/官方 Kernel/当前 Organization/Store 观察均零改。
- 未列 §5.1 的生产文件全部零改；无新接口/包/依赖、facade、fallback、第二事实源或新写入口。通用 Prompt 存储/管理与已有覆盖记录不动。

Legacy decision: **RETIRE**。Reusable behavior：三项规则读取、词库/话题/覆盖规则、安全与 schema 能力沿原 owner 保留，无新生产 EXTRACT。Current owner：原 listingadmin 配置持久化、submitprep/publishing 内容规则，各当前 Store/官方执行/worker owner。Cutover/deletion condition：准确设计获确认和适用独立检查、Issue Ready 后，原 guard RED→整批删除/有效测试转接 GREEN→原 CI/一次最终独立检查；无运行/数据切换。

## 6. 实施和验证仍在一个主要 PR

取得明确实施授权及 Ready 后，先在原 retirement guard 添加准确目标的失败断言，再整批删除/转接测试。只做风险相称的原自检和稳定候选 CI，不扩建 checker、runner、session 或故障矩阵。

### 6.1 原测试和检查的准确处置

| 准确文件 | 允许变化与必须保留的证明 |
| --- | --- |
| listingadmin/owner_scoped_repository_test.go | 整退该文件：只剩三项退休 Create 的 ownerless-write 测试与两个专属 helper，均无其它消费者。不存在的创建入口不再伪造写入验收；当前读 scope、默认安全开关、DB owner 约束与 Store 写测试原样保留 |
| listingadmin/owner_scope_test.go | 只退三个 TestRequireOwnerUserID*；TestWithOwnerUserIDSuppliesInternalOwner 改为直接通过原 requestUserIDFromContext 证明可信上下文，不调用退休校验器；WithOwnerUserID 保留 |
| listingadmin/repo_paging_test.go | 只退 TestTakeOwnedTenantRowMapsMissingRecordToNotFoundError / TestUpdateOwnedTenantRowMapsMissingRecordToNotFoundError；其所有生产调用一起退休。三个保留 paging/active/tenant/owner/admin 测试与 fixture 原样；原 Store NotFound/同步保护及覆盖读取错误行为保留 |
| listingadmin/sensitive_word_model_test.go | 退两个 defaults/audit 写专属测试；conversion 测试转为直接 row.toSensitiveWord，保持读字段/数值投影断言，不把旧反向 trim/default 加入读实现 |
| listingadmin/sensitive_word_query_test.go | TestFindSensitiveWordRowsUsesRequestOwnerScope 原样；ResourceFilters 测试转到当前 Status+active/tenant/owner/分页断言，五管理筛选断言退出，词库 Language/Word 读值不删 |
| listingadmin/generation_topic_policy_test.go | CreateAndList 测试去掉已退创建/列表/trim；用原 SQLite row fixture 保留数据库唯一约束拒绝重复及当前 enabled keys 读取的有效证明。原 duplicate-precheck 测试和 openDB helper 原样，不恢复写接口 |
| listingadmin/generation_topic_policy_query_test.go | 两测试只将 Create/GenerationTopicPolicy fixture 改为既有 listingGenerationTopicPolicy 表内 fixture；tenant/platform/status 与跨 owner 的 tenant-global 断言保持。可补同表 deleted fixture 验证原 active 条件，不改生产规则 |
| publishing/shein/generation_topic_runtime_test.go | stubGenerationTopicPolicyRepository / stubGenerationTopicOverrideRepository 仅去各六个退休同名方法；两个现用读方法、fixture/全部摘要/无 tenant/repo/overlay 断言原样 |
| publishing/shein/submit_prep_test.go | stubSensitiveWordRepository 去五退休方法、仅其写入使用的 created/updated 两字段和 List 中 query.Word 旧筛选分支；Status/tenant/分页、全部内容/品牌/话题/清理断言原样 |
| shein/submitprep/generation_topic_overlay_test.go | policy stub 仅去六退休方法，其原 ListEnabledTopicKeys/fixture/跨 tenant overlay 断言原样 |
| publishing/shein/generation_topic_policy_integration_test.go | 两个现有真实 SQLite 读取测试的 factory/seed helper 改为返回原 DB handle、直接同表 seed；用原 JSON 标准库在 fixture 编码，不经退休 CRUD，不新增生产 writer/fixture 框架。原摘要/清理/tenant/override/status 的全部断言保持；测试名可去除不存在的 Admin 产品入口表述 |
| tests/listingkit_backend_retirement_test.go | 原 AST guard 增加本批准确 package/receiver/type/helper/error/query 字段的 absence 与三读合同/schema/Store 保留断言。原 guard 只原位扩展，不改检查器框架、不给 ENOENT/skip 放行，不放宽其它必需断言 |

准确范围：**一个整退测试文件、十个 mixed 原测试文件及一个原 guard**。其它测试零改，尤其 schema_migrate_test、access_scope_test、Store/worker/TEMU、内容/品牌/registry 的有效测试保持。退掉的 ownerless CRUD 测试不是当前 DB 约束、读权限或现用 Store 写入保护的替代物，也不能借此取消 race step。

必要验证：准确十生产文件/测试矩阵及保留声明对照；原 scoped guard RED→GREEN；listingadmin、publishing/shein、shein/submitprep、content/generationtopics、sheinlogin/bootstrap/managed/worker 与 TEMU 现有相关原测试；current-application、shein-login-worker、product-listing-api-schema-migrate 仅编译；现有当前命令依赖与原 CI。无生产行为变化的声明/fixture 退休不制造伪业务 RED；只读方法必须与批准基线逐声明相同（词库查询只少五管理条件）。

未来实施仅可窄校准七份直接相关文档：本设计、README、legacy-register、module-target-mapping、#824/#827/#831 的对应保留说明。保留历史证据/计数，不改写原冻结矩阵。本轮设计只有本文件与 README。

**全部 workflow、tests/import_boundaries_test.go、phase2-runtime-inventory 的 CURRENT/历史行和计数、依赖、.golangci.yml、ceilings/保护零改。** 本批无包/非标准库导入边退出；access_scope 仍消费原身份上下文，不取得其它批次的 inventory 删除权限。原 listingadmin race 和全部必需/安全检查保留。

Runtime/provider/user/production acceptance：**NOT_RUN**。源码调查/fixture/编译/CI/AI 复核均不是运行或用户验收，不增加内部试用前置。

## 7. 准入、确认与停止点

1. 当前只做设计/只读调查，不修改生产/测试、不构建或运行。适用独立设计检查、用户确认后，Issue 明记 Independent Architecture / IMPLEMENTATION_READY / Ready；按本 PR 的 §5/§6/§6.1/§7 获明确实施授权后再整批编码。
2. 单 Writer 复用独占 worktree/主要分支。原 GitHub 同 HEAD 检查运行时不重复触发；未获明确授权不新增子代理/Writer/AI 互验线程。正常架构检查最多两轮，非 Blocker 在实现测试或 Backlog 收敛。
3. 实现发现 §5 外真实管理消费者/外部合同，或必须改读权限、身份、事实/schema、状态/事务/恢复或平台副作用时，停止对应变更，提供证据与最小差异；不静默扩大删除，不用兼容包装保住旧写 owner。冻结设计仅新的 BLOCKER 可重开。
4. 同一 PR 完成整批实现、必要原自检/CI与一次最终独立交付检查；修复只复核相关增量。用户验收、准确 HEAD 合并、matching main CI、部署及真实操作分别授权与记录。
5. 三项保留数字读取及独立 worker 身份债务仍存在，不宣称 #29/#301 清零或新系统内容管理可用。当前核心使用能力保持，不新建产品/治理前置。

## 附录 A：17 个准确退休方法

在对应 Gorm* receiver 与 Repository interface 同时退出，保留读方法不得按前缀匹配：

```text
GormSensitiveWordRepository / SensitiveWordRepository:
  GetSensitiveWord CreateSensitiveWord UpdateSensitiveWord
  UpdateSensitiveWordStatus DeleteSensitiveWord
GormGenerationTopicPolicyRepository / GenerationTopicPolicyRepository:
  ListGenerationTopicPolicies GetGenerationTopicPolicy CreateGenerationTopicPolicy
  UpdateGenerationTopicPolicy UpdateGenerationTopicPolicyStatus DeleteGenerationTopicPolicy
GormGenerationTopicOverrideRepository / GenerationTopicOverrideRepository:
  ListGenerationTopicOverrides GetGenerationTopicOverride CreateGenerationTopicOverride
  UpdateGenerationTopicOverride UpdateGenerationTopicOverrideStatus DeleteGenerationTopicOverride
```

保留三个接口各一个方法：ListSensitiveWords / ListEnabledTopicKeys / GetGenerationTopicOverrideByTopicKey；签名不变。constructors、AutoMigrate 和 row models 不是退休目标。

## 附录 B：准确专属声明

```text
16 helper（仅 internal/listingadmin；toGenerationTopicPolicy 是 listingGenerationTopicPolicy 方法）:
  listingSensitiveWordFromSensitiveWord applySensitiveWordDefaults applySensitiveWordAuditFields
  toGenerationTopicPolicy listingGenerationTopicPolicyFromGenerationTopicPolicy
  applyGenerationTopicPolicyAuditFields findGenerationTopicPolicyRows applyGenerationTopicPolicyQuery
  listingGenerationTopicOverrideFromGenerationTopicOverride applyGenerationTopicOverrideAuditFields
  applyGenerationTopicOverrideQuery encodeStringListJSON encodeStringMapListJSON
  requireOwnerUserID takeOwnedTenantRow updateOwnedTenantRow
3 error var:
  ErrSensitiveWordNotFound ErrGenerationTopicPolicyNotFound ErrOwnerUserIDRequired
5 DTO type:
  GenerationTopicPolicy GenerationTopicPolicyQuery GenerationTopicPolicyPage
  GenerationTopicOverrideQuery GenerationTopicOverridePage
5 field（仅 SensitiveWordQuery）:
  Word Language Tags Level Remark
```

SensitiveWord、GenerationTopicOverride 及三张 listing* row 的同名字段全部保留；WithOwnerUserID、ErrGenerationTopicOverrideNotFound、decode/normalize 与 shared/Store owner helper 不在删除清单。
