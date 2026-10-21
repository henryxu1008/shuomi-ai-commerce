# #301 旧数字上下文与共享值合同完整退休设计

Status: **Independent Architecture / IMPLEMENTATION_READY / 用户已确认 / 实施中**。冻结设计 HEAD `01d8b8a5087486849d2b883f6c4f554164713857` 的原 CI 与适用独立设计复核均已完成，用户明确确认 §1–§7（含 §6.1）；[实施准入](https://github.com/qq550723504/task-processor/pull/868#issuecomment-6312908855) 已先于生产代码记录 Ready → In Progress。下文设计阶段的权限与待确认表述保留为历史阶段条件，由本状态和该授权精确替代；同一 Writer、分支和 PR 整批实施 §5，workflow/inventory 零改，合并/部署/关单仍未授权。

设计调查及实施开始基线为 main `23bde99ec2a436b96085e9dc3f02748f144a67b4`（#867 实际合并），[准确 push/main CI39570885435](https://github.com/qq550723504/task-processor/actions/runs/39570885435) 已通过，仅作基线证据。#301 OPEN；#29 已源级关闭，不重开。准确设计 HEAD、原 CI 与独立复核记录在本批 PR，不把设计审核或源码退出当运行/用户验收。

## 1. 用户结果、范围与权限

使用者是维护当前采集、来源保存/刷新、平台规则、Store 和独立 SHEIN worker 的维护者。当前主链已有可用 owner；本维护清除尚无生产调用的数字日志/分片 API、共享规则输入的旧数字管理表面和现役来源身份的双轨赋值，保持同一来源保存/刷新、证据读取和幂等身份稳定。没有证据证明当前核心操作因此不可用，不新增业务入口或验收平台。

一个 Delivery Batch、一个 Writer、一个主要 PR，完整覆盖三组的实现/测试/文档直接尾部；实现/提交可分可验证步骤，不按字段/helper 拆 PR。三组并非一律整包删除：所有有效纯规则、两种 SHEIN 属性及真实 JSON/持久化合同按 §5 保留或抽取。

本轮授权仅为只读调查、本文及 Architecture README/Legacy Register 提案指引、#301 当前段与 PR 维护。**源码、测试、schema、配置、workflow、inventory、依赖/锁、ceilings、保护零改**。不读既有凭据/私密配置/运行报告；不运行应用、浏览器、provider、Docker、kubectl、SQL 或发布脚本；不操作任何既有实例、数据、Profile、队列或资源。不增加内部试用前置；正式实施、合并、部署和关单分别待授权。

## 2. Design Basis、产品依据与准入

当前设计文档：**Design Basis: N/A**，不改变运行语义。未来完整实施：**Independent Architecture / DESIGNING / NOT_READY**。本批明确改变活跃 SourceIdentity 正常化合同，并抽取两个旧输入类型；不能按“少量字段/机械删除”规避准入。用户确认 §1–§7（含 §6.1）、适用独立复核完成并在 #301 记录 **IMPLEMENTATION_READY / Ready** 后，唯一 Writer 才能改正式路径。

Authority：

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)。只有 EXTRACT / RETIRE；没有迁移、旧 ID 映射、compatibility adapter、fallback、双读写、第二事实源或执行例外。
- [来源业务闭环](source-business-closure-v1.md)：当前 SRC/ProductDB 与 Catalog publication/version、Collection/Preparation/Asset 的原 owner、权限和事务；当前严格来源身份、保存/刷新/读取/显式批准能力继续有效。本批不扩 #30/#307 功能，不改页面/导航/交互，Product/Figma 增量 **N/A**。
- [#867 完整尾链设计](legacy-numeric-acquisition-tail-closure-retirement-boundary.md)：当前 Amazon Processor/browser、平台 API/规则、价格与共享合同继续保留。本提案仅在用户确认后替代其对 PricingRule/OperationStrategy **完整旧类型**和 SourceIdentity **别名行为**的保留；ProductImportMapping/两属性、来源持久化合同不被替代。
- [#863 独立 worker 设计](shein-worker-numeric-persistence-retirement-v1.md)：Organization/Store UUID、native live 授权、受权 CLI、会话、Store 审计/事务、唯一执行归属、UNKNOWN 不自动接管全部保持。

Must：当前六个 Source* 值、SourceKey/fingerprint、ProductKey/publication/version、EnvelopeJSON/SnapshotJSON/InputHash/lineage 的原有效合同稳定；跨组织/撤权、输入大小/取消、幂等冲突/损坏检测不弱化；当前规则/API/属性/采集/Store/独立 worker 保留。Should：复用原规则、原测试/guard/CI，必要值类型放回当前消费 owner。Out of Scope：来源业务扩展、身份/计费/Store/worker 改造、数据或配置迁移、codec/version 新协议、所有真实环境操作、新 runner/验收矩阵/平台。无新增 Accepted Risk。

Threat Model 仅覆盖本批实际改变的来源身份派生、当前证据编码/幂等完整性、租户权限与保留能力回归；现有 native 授权/数据库/worker 协议继续承担原责任。不扩大到真实平台、历史数据、运维 side-channel 或新故障平台；没有因退休而接受新的安全/业务风险。

## 3. 实际消费者与根因

调查使用 tracked 源检索、Go 包 imports 及 31 个 cmd 依赖元数据；未执行产品命令或启动服务。字段名相同不等于同一类型，商业系统字符串 TenantID、SDS 平台自己的 numeric tenant/merchant 字段均不归本批。

### 3.1 数字日志与分片辅助岛

`platform/logging.LoggerHelper.LogTaskStart/LogTaskComplete/LogTaskFailed` 与 `WithTaskContext/WithStoreContext` 只被原 logging 测试/benchmark 和 core/logger 导出别名触及，没有生产调用；三项专属 FieldTaskID/FieldTenantID/FieldStoreID 常量也只在此闭包与别名测试内使用。core/logger 其余真实消费者、统一 singleton、trace/request/context/logger/hook 与通用日志能力继续存在，不删整个 facade 或 logging 包。

`shared/hashx.ShardForTenantShop` 无生产调用；SimpleHash 只被它调用。hashx 当前包无生产 import，但本批仅退这个已失去用途的 Tenant/Shop 分片闭包，保留 MD5/HashString/ConsistentHash 及 shared 根存在/依赖方向约束；不扩成全仓 dead-code 工程。

### 3.2 共享规则输入有值消费者，旧管理身份无职责

包级图：`listingruntime` 仅被 `shein/pricing`、`shein/activity`、`shein/productsync` 引用。pricing/activity 无外部生产包 import；productsync 由 activity 消费。保留纯规则是已确认产品决定，不能仅因没有命令直接装配就删除有效行为。

- `listingruntime.PricingRule` 共 20 字段，唯一该类型生产消费者是 `shein/pricing/calculator.go`，只读 PriceMin/PriceMax/RuleType/RuleValue/FixedValue。`publishing/shein.PricingRule` 是另一当前类型，完整保留，不按同名合并/替换。
- `listingruntime.OperationStrategy` 共 37 字段，只有 activity 的 registration_config/promotion_create_products 读取其中 8 个值字段；IsEnabled/Status 没有真实调用。保留活动价格、折扣、库存、保本、风险规则及原负例，不恢复调度/平台 mutation。
- `listingruntime.ProductImportMapping` 通过 `productsync.EnrichedSkuInfo.MappingInfo`（`mapping_info`）进入共享属性 JSON。字段级没有业务读取不等于没有编码消费者；其完整类型/字段顺序/值、两种属性和原 JSON 语义继续保留。它不是当前身份查询或旧表仓库，不增加这些行为。

### 3.3 来源身份双轨和当前持久化编码必须分开判断

现有显式来源 owner 是 `product/sourcing.SourceIdentity` 的六个字段：SourceType、SourcePlatform、SourceID、SourceURL、SourceVersion、SourceFingerprint。旧 Platform/Region/ProductID/StoreID 仍被 SourceRequest.Identity、1688/Amazon 适配器写入，NormalizeSourceIdentity 又把 Source* 与旧值双向互填；localagent 的 summary 从旧 ProductID 读取。旧 Key() 的调用只在测试，当前持久化使用 SourceKey()/PublicationIdentity。

同时存在具体**当前**持久化消费者：

1. `sourcing.InternalProducer.prepare` → NormalizePublicationEnvelope → json.Marshal(envelope) → CanonicalPublicationInputHash；该 hash 同时编码 Identity 和原 EnvelopeJSON。
2. `integration/persistence/product/sourcing.repository.read/verifyLoaded` 解码现有 EnvelopeJSON，重算 InputHash，并把 NormalizePublicationEnvelope 后重新编码的 bytes 与原 bytes 精确比较。
3. 当前严格发布已拒绝四旧字段非零/非空输入；通用正常化随后填入别名，发布正常化再清空它们。**即便字段值全空，原 struct 的默认 JSON 仍包含四个槽位**；物理删除、改 tag/omitempty、改类型/字段顺序均会改变当前编码、InputHash 和有效证据读取结果。

原 guard 使用 TrimSpace：Platform/Region/ProductID 仅含空白时仍属原来允许的输入，随后被原 trim 和发布层清空规范为零值。**三个槽位的原 TrimSpace 和 NormalizePublicationEnvelope 四项最终清空均保留**，保证空白与空串的规范编码/hash一致；不能同时删除这两层，也不能改成拒绝原本合法空白。

所以本批建议：退休双轨**行为和当前调用方依赖**；保留四槽位现有声明/顺序/零值 JSON 与严格拒绝作为当前持久化合同。不给这些槽位赋身份语义，不读旧值派生新身份，不新增 codec/兼容 reader、不重写旧记录。若用户将来要求物理删除这些当前编码槽位，必须先另作明确产品决定与持久化设计；本批不把它作为试用前置，也不暗设迁移许可。

## 4. owner、不变量、持久化与失败边界

| 边界 | 当前 owner / 原合同 | 本批要求 |
| --- | --- | --- |
| 身份/授权/资源/账务 | authidentity、Organization、native grants、commercial/ledger | 无改动；字符串 TenantID 可为纯数字，不能据外形转换/退休 |
| 来源采集输入 | 1688/Amazon 自己的 request、Processor/source port | Region/StoreID/Zipcode 等当前请求值保持；只停止把 Region/StoreID 装入 SourceIdentity，不将输入 StoreID 当 Organization owner |
| 来源语义身份 | sourcing 六个 Source*、SourceKey、原弱身份 fingerprint | 相同当前六值产生相同 key/fingerprint；不再用旧四字段补值或接受旧值作为身份。弱值识别/显式准入与严格发布要求保留 |
| localagent 回执 | 原 Job/claim/token/过期/terminal 状态；EnvelopeSummary | summary.ProductID 改读相同来源 SourceID，外部 product_id/source_key/其余字段和值不变；不改状态、授权、重试或 terminal API。SubmitResult 的现有 HTTP 回执只返回 summary，不增加返回 envelope 或新入口 |
| 当前 SRC/Catalog 事实 | 原 ProductDB 事务、SRC receipt/evidence + Catalog bridge | schema、repository、事务/锁、幂等键、ExpectedBaseVersion、原版本读取/刷新不变；有效 canonical envelope bytes、InputHash、SnapshotHash/EnvelopeHash 和 publication identity 不变 |
| 四个来源编码槽位 | 原 SourceIdentity layout + 原严格发布/完整性验证 | KEEP 原四字段类型/声明顺序/默认 JSON、三个字符串原 trim 和发布四项最终清空，不删除/重排/omitempty/换codec；TrimSpace 后非空或 StoreID 非零照旧拒绝。仅退互填、来源适配器赋值和旧 Key；原发布 reject/preflight 不删 |
| 纯规则 | 现有 shein/pricing 与 shein/activity | 仅抽取 5/8 字段到消费 owner，公式、默认/边界/风险判断不变；没有新的 manager/provider/平台调用 |
| 共享属性 | ProductImportMapping + EnrichedSkuInfo/EnrichedSkcInfo | 全部字段/JSON/平台类型完整保持，不按字段名清空 MappingInfo 或删结构 |
| 独立 SHEIN worker/Store | #863 current native owner 与原审计/UNKNOWN fence | 源码/配置/持久化/Redis/Profile/CLI/SessionTransport 均不变，保留独立登录与同步 |

没有新状态、事件、表、第二事实源、补偿/对账/恢复入口。相同幂等键不同输入继续由原 producer/store 拒绝；响应丢失、重试、重启/取消、并发、撤权、跨组织、缓存漂移、deadline/输入上限继续原协议。来源当前规范数据编码若不再相同，会影响当前 receipt 读取和幂等，属于实施/合并层 BLOCKER（数据/证据损坏、核心保存读取无法完成），不能通过重算旧数据或放宽比较处理。

## 5. 完整删除 / 保留 / 抽取矩阵

### 5.1 无消费者数字辅助链：RETIRE

| 源码 | 精确删除对象 | 保留对象 |
| --- | --- | --- |
| platform/logging/helpers.go | LogTaskStart、LogTaskComplete、LogTaskFailed、WithTaskContext、WithStoreContext；FieldTaskID/FieldTenantID/FieldStoreID | LoggerHelper 其余方法、generic 字段、singleton、context/trace/request、API/进度/重试/metric/hook 与 logrus |
| core/logger/compat.go | 上述两项 With* 别名和三项 Field* re-export；LoggerHelper 自然不再导出已退方法 | 其余类型/函数/常量别名、所有当前消费者，不重写日志架构 |
| logging/helpers_test.go；core/logger/compat_test.go | 仅五项旧 API 测试、WithTaskContext benchmark、别名调用和三常量断言/原数量校准 | 所有通用正负断言、共享实例/trace/request/schema契约等原测试 |
| shared/hashx/hashutil.go | ShardForTenantShop、只服务它的 SimpleHash、失去用途的 fmt import | MD5、HashString、ConsistentHash、shared 目录/依赖 guard；不引入替代分片器 |

### 5.2 规则有效值归当前 owner：EXTRACT + RETIRE

| 原类型 | 精确抽取到现有消费者 | 原类型退出的其余字段 / 行为 |
| --- | --- | --- |
| listingruntime.PricingRule | 在现有 shein/pricing/calculator.go 定义 AutoPricingRule，只含 PriceMin、PriceMax、RuleType、RuleValue、FixedValue，类型和原语义不变；原 GetAutoPrice/ApplyRule 直接消费它 | ID、Name、RuleCode、Description、StoreID、CategoryID、AcceptCondition、RejectCondition、Status、Remark、TenantID、TargetProfitMargin、MinProfitMargin、AcceptBelowTarget、ReappealBelowMin；删除原 rules.go，无 type alias/转换wrapper |
| listingruntime.OperationStrategy | 在现有 shein/activity/registration_config.go 定义包内 activityStrategy，只含 ActivityDiscountRate、ActivityLimitedDiscountRate、ActivityStockRatio、ActivityMinProfitRate、ActivityLimitedMinProfitRate、ActivityPriceMode、ActivityPartakeType、FixedPriceAdjustment，原字段类型不变；原纯函数直接消费 | ID、TenantID、StoreID、Name、Platform、Status、StockChangeThreshold、StockChangeAction、OutOfStockAction、MinProfitRate、LowProfitAction、PriceUpdateMultiplier、StockUpdateRatio、ActivityEnabled、ActivityType、PromotionRatio、TimeLimitedDiscountRate、TimeLimitedMinProfitRate、TimeLimitedPriceMode、TimeLimitedUserLimit、TimeLimitedUserLimitNum、TimeLimitedStockLimit、TimeLimitedStockLimitPercent、PriceIncreaseThreshold、PriceDecreaseThreshold、PriceIncreaseAction、PriceDecreaseAction、RestoreStockAmount、Remark；IsEnabled；删除原 operation_strategy.go，无旧仓库/配置替代owner |
| listingruntime.ProductImportMapping | **KEEP 完整原 types.go**，包含其数字字段，继续原 MappingInfo 编码职责 | 不物理收缩、不改 JSON、不新增 numeric lookup 或任何 DB 读写；该明确保留结果也是本组完整核对的结论 |

相关 calculator_test 与 activity registration_direct_products 等原有效测试只改为同值的当前类型，不删除价格/多SKU/保本/库存/风险断言；同名 publishing/shein.PricingRule、PriceManager、PriceCalculator/CostProfitCalculator、平台 API/DTO、全部营销规则及两属性类型保持。不建通用策略框架或新计算器。

### 5.3 来源旧别名行为完整 hard-cut：RETIRE + 直接消费当前值

| 精确位置 | 退出行为 | 当前合同/消费者处置 |
| --- | --- | --- |
| sourcing/source_identity.go | SourceType/SourcePlatform/SourceID 从旧字段补值；向旧 Platform/ProductID 反填；Key() 及只为它的 strconv import/旧迁移注释 | 六 Source* 正常化、Validation/Valid/WeakButFingerprintable、fingerprint/SourceKey 算法保留；四编码字段声明/类型/顺序/JSON默认及三个字符串原 trim 保持，标明只参与现有编码/规范化和拒绝检查，不再是身份输入 |
| sourcing/source_request.go/test | Identity() 向旧 Platform/Region/ProductID/StoreID 赋值；只断言旧 Key/alias 的测试部分 | 显式由当前 source request 的 Platform/ProductID 构造 SourceTypeCrawler、SourcePlatform、SourceID；request 的其余字段、正常化/variant 行为保持；断言 current Source* 和 SourceKey，不新增 fallback |
| integration/crawler/amazon 与 a1688/product_source.go 及对应测试 | SourceIdentity 旧四字段赋值、Key 专属断言 | 直接构造六 Source*；现有平台源 ID/URL 获取和缺失事实规则、Processor/browser/raw checksum/商品/图像/变体/价格不变；当前 SourceKey 值、真实来源错误/取消断言保留 |
| localagent/service.go/test | summary 从旧 Identity.ProductID 读取 | 直接读 Identity.SourceID，原 summary.product_id 值和 API/CLI/client 合同不变；不改 Job 状态或 token/claim/URL校验 |
| sourcing/normalize.go/test、source_identity_test.go、source_request_test.go 等直接尾部 | 仅旧互填/旧 Key 可用的实现细节期望；原发布层四项最终清空全部保留 | NormalizePublicationEnvelope 的六值要求与四槽位非零拒绝、preflight/输入上限、description校验原样；原负例保留，不改为忽略非法字段 |
| sourcing/publication*、catalog_asset_handoff、acquisition、SRC repository、source evidence projection、各当前 producer | 不作业务改造 | 保留现有编码/hash/lineage/授权/保存读取/刷新与同值 key；现有 canonical 当前 producer 无需跟随新“协议”重写。仓库/SQL/schema不改，已有损坏/旧字段拒绝用例保留 |

本组没有承诺物理删除四 JSON 槽位或迁移现有来源记录。当前四槽位只支持原严格零值合同；没有双读、旧值派生、wrapper、兼容 codec、对历史记录的放宽或任何新 Compatibility 类别。

```text
Legacy decision: EXTRACT | RETIRE
Reusable behavior: 原价格/活动纯规则，六字段来源身份/键/fingerprint，当前JSON与幂等/证据拒绝合同
Current owner: shein/pricing、shein/activity、product/sourcing、SRC/Catalog、localagent；其余现有平台/Store/worker owner不变
Cutover/deletion condition: 用户确认完整矩阵+适用独立复核+IMPLEMENTATION_READY；当前消费者同批使用已批准值；旧API/互填/Key/两旧规则类型缺席，全部当前持久化/属性合同保持
```

## 6. 必要验证、交付与已有证据

设计阶段仅静态消费者/字段/持久化链与适用原文档检查；不为文档制造RED或运行资源。实施阶段使用原测试文件与 retirement guard：无消费者声明/旧类型/Key/互填先RED，按 §5 改动后GREEN；当前价格/活动/平台规则和 Source* 正负用例继续运行。

必须验证且只覆盖本批风险：

1. 五日志API/两别名/三常量、分片闭包及两旧类型不存在；通用日志/hash helper保留；原包import/AST方向保持。
2. 抽取 5/8 字段后原价格、库存/折扣/风险/多SKU用例同值通过，无平台 mutation。
3. 当前六 Source* 正常化/弱值fingerprint/SourceKey、1688/Amazon结果与 localagent summary 同值；legacy-only不能产生有效来源身份；原严格发布对 TrimSpace 后非空字符串/非零 StoreID 仍拒绝，输入上限/取消不变；三个槽位为纯空白/空串时沿原 trim+最终清空得到相同零值 JSON/InputHash，不改成更严格拒绝。
4. 用任务自有**合成的当前 canonical envelope**，复用原 producer/store 单测绑定原基线编码/hash作前后比较，精确检查 EnvelopeJSON/SnapshotJSON/InputHash/PublicationIdentity 与原相同；保存/读取/相同键不同输入/损坏拒绝沿原有效测试。只允许在现有测试内加必要 fixture，不建新runner、黄金文件平台、脚本或故障矩阵。不读取既有业务记录。
5. 原日志/价格/活动/sourcing/source adapters/localagent 与原 current-application/shein-login-worker 构建；完整原CI只在稳定候选运行。原CI内PG/权限/资金/race照常，不能把PR CI或源码编译当真实实例/用户验收。

无需为这批源码/值改动另启动本地 PG/Redis/browser。若确有必要执行额外资源，先说明具体缺口和有界资源权限；未执行保留NOT_RUN。当前采集、有效API/规则、Store和独立SHEIN登录/同步仍沿现有入口使用，没有新的内部试用步骤。

### 6.1 原检查、文档、inventory 与 workflow 的精确处置

| 对象 | 未来同批最小校准 | 不变 |
| --- | --- | --- |
| 原 logging 与 core/logger 测试 | 只删退休API/三常量/benchmark，别名常量数量由16随三项退出为13 | 原singleton、通用context、trace/request与其它正负断言 |
| 原 pricing/activity 测试 | 同值改当前5/8字段输入类型，删除只依赖退出旧字段的fixture初始化（若刷新发现新读者先报告） | 全部有效业务/平台/确定性断言，PriceManager/Calculator与营销API |
| tests/listingkit_backend_retirement_test.go / import guards | 增加本批精确声明/行为absence及当前5/8字段owner检查；旧rules.go/operation_strategy.go若有保留项，仅改当前owner定位 | 其它全部旧absence、两属性/MappingInfo完整保留、八命令真实依赖、严格读取失败、安全/SDK/租户/商业/存储/ratchet约束 |
| 原 sourcing/publication/SRC/localagent 测试 | 旧Key/别名正例改当前canonical；保留且明确四槽位非零拒绝/编码/hash/损坏等原有效断言 | 不加missing-skip，不删当前保存/读取/刷新/权限/幂等负例，不改来源证据JSON校验 |
| 文档 | Architecture README/Legacy Register及直接module/code-health/来源说明只校准§5准确对象；#867保留结论标为被本矩阵精确替代，历史执行/失败不改 | 当前原产品/Figma决定、worker/Store/平台合同与所有无关历史证据 |
| phase2-runtime-inventory.md / ceilings | **零改**；当前调查没有因本批退完整provider→consumer包边的必要校准。若原CI证实不同，提供准确行/消费者证据，报告后再决定，不能自行删表或改计数 | 所有CURRENT/历史行、计数、ceilings、依赖/锁/保护 |
| 所有workflow/CI | **零改**，按原分类/触发执行 | 必需/安全/PG/资金/发布、listingadmin原当前合同race、当前入口构建等原检查不变 |

## 7. 设计确认、复核与停止点

1. 本轮仅设计文档/索引/登记及Issue当前段；一次适用独立架构检查核对真实消费者、§5/§6.1完整矩阵、当前持久化hash和四槽位/MappingInfo保留边界。最多两轮正常设计复核；finding先按AGENTS分类，不重复同SHA Running请求，不默认增加AI互验线程。
2. 用户确认 §1–§7（含当前编码/两属性/MappingInfo保留及§6.1），独立复核收敛后在同一PR记录 **Independent Architecture / IMPLEMENTATION_READY / Ready** 与唯一Writer。未确认前正式实现NOT_READY；本设计不授权合并/部署/真实数据/关单。
3. 确认后同批完成整个 §5、必要自检/提交/推送及review/CI修复；不机械拆PR，不根据文件数规避门槛。只有真实当前 Must 的 BLOCKER 才重开冻结架构，普通实现测试/非阻塞hardening不新增试用前置。
4. 完整候选集中做一次独立最终交付检查，核对实际diff、保留能力与现有有效证据；修复只复核增量。若当前JSON/hash改变、非法槽位被静默接受、需要新持久化/身份/恢复owner或真实数据转换，停止对应实施，报告具体Must/阻塞层级；不能用兼容层或降低原检查化解。
5. 交出准确最终HEAD、原CI和复核供用户决定合并；合并另行绑定准确HEAD，之后核对实际merge SHA的push/main CI。#301继续OPEN；O1–O6未完成运行/用户验收仍按原范围NOT_RUN，不由本批自动签发或关闭。
