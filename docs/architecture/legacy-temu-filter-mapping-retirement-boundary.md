# 旧 TEMU 变体筛选与 AI SKU 映射前置编排退休边界

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

- Status: **USER_CONFIRMED / SOURCE_RETIRED**。用户在当前会话确认 #754 冻结设计 `0d8bc37850e949048e6de6fd33a2ea473eb44e65` 的 §5、§6、§6.1、§7；九个专属生产文件、一个旧测试和 mixed input 专属部分已按矩阵退出。合并和真实操作仍须单独授权。
- 调查基线：`main @ ad24d032c1af98056a77f6439c52b079db33f2a3`（#751 已合并）。实施前同步实际 main `bd30076b2ec80364e0c6442ac075b1d763dbd451`；增量仅 #753 前端地点选择及其文档，无目标消费者变化。准确实现 HEAD、CI、独立最终检查与用户决定维护于 PR，不在本文滚动；旧设计 CI/review 仅为历史证据。
- 当前 **Design Basis: Reuse Existing Architecture**：用户确认后，生产修改前已在 #29/#301 登记 **Ready**、冻结依据与唯一 Writer，再转入实施；原设计期 N/A 仅覆盖当时的文档/只读映射。
- 保留当前采集、Store 观察同步、平台 API/同名 DTO、有效规则/共享合同、两种 SHEIN 属性、PriceManager/PriceCalculator、Reader/Cache/Mapping、独立 SHEIN 登录 worker 和 Store sync。两套及其它 workflow、必需/安全检查、保护/ceiling 零改动，不操作实例或数据，不增加内部试用前置。

## 1. 用户结果、范围与非目标

使用者是维护当前内部试用、采集、平台/Store 能力与独立登录的维护者。交付为闲置 Task 变体筛选与 AI SKU 映射前置编排完整专属闭包的源码退休，保留有效规则、batch 与原 owner，防止后续接线重新引入旧 Task owner。此维护不阻塞当前用户操作，不改变入口或新增试用前置。

范围为 `internal/temu/sku/{variant_filter_handler,ai_mapping_handler}.go`、`internal/temu/filter/{rule_handler,rule_manager,rule_stats_provider}.go`，以及经消费者刷新确认专属的 `product/filter_checker.go`、`rules/rule_validator.go`、`handlerbase/{fulfillment_checker,filter_types}.go`：基线 **9 个生产文件共 884 行**；专属 `rules/rule_validator_test.go`（113 行）仅断言旧固定源图门槛，按 §4.1 退休。另仅包括混合 `sku/ai_mapping_handler_input.go` 的 `AIHandlerInput`、`buildAIHandlerInput` 及唯一不再使用的 temucontext import；混合输入文件不整删。

这是同一旧 SKU 前置编排的源码退休闭包，不为每个 helper 分批交付。`sku`、`filter`、`product`、`rules`、`context`、`listingadmin` 和 provider client 均不是整包退休范围。保留 `SkuVariantProcessor`、现有 SKU/AI 映射、`AIBatchInput`、有效规则/状态合同和事实 owner；没有新增 EXTRACT 生产实现或公共合同。

不新增筛选/统计服务、AI 调用入口、模型/provider、主变体选择策略、缓存/IAM/恢复协议、UI、runner 或验收工具。#692 Store 安装等其它 owner 不接管。不执行 main、构造器、规则 API/provider/Preview/DryRun、采集或 schema 工具，不读取真实配置/凭据、DB/Redis/Profile/browser、历史日志/JSON；不启停、部署、迁移、补写、重放、清理或删除实例/数据。静态无消费者不代表历史资产或运行实例清零。

## 2. Design Basis、Authority 与不变量

依据：

- [全新系统基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)：只有 EXTRACT / RETIRE；无 legacy wrapper、fallback、旧 ID 映射、双读双写或第二事实源。
- 已确认的 [CLI](legacy-listing-cli-retirement-boundary.md)、[control-plane](legacy-listing-control-plane-retirement-boundary.md)、[数字执行链](legacy-numeric-task-execution-retirement-boundary.md)、[数字持久化](legacy-numeric-task-persistence-retirement-boundary.md)、[TEMU 前置](legacy-temu-preflight-retirement-boundary.md) 与 [变体读取/缓存](legacy-temu-variant-retirement-boundary.md)：旧数字执行入口已退出；本轮接续此前保留的闲置筛选与 AI 前置链，不推翻其它保留项。
- [既有商品规则 owner](../../internal/marketplace/productpolicy)、[TEMU publishing owner](../../internal/marketplace/temu/publishing/README.md)、[持久化规则状态](../../internal/listingadmin/rule_status.go)、FilterRuleAPI/DTO/仓库与现有 SKU processor/batch 合同：有效行为继续原 owner，不把旧 Task 编排包装成新能力。
- [独立 worker 身份/持久化](shein-worker-identity-persistence-boundary.md)、[当前 Store](store-center-current-application-v1.md)、[observations](store-center-platform-observations-v1.md)、[准备接线](store-observations-preparation-runtime.md)：当前事实、身份、授权、retry/UNKNOWN 继续原 owner。Figma N/A，无 UI/IA 修改。

实施采用 **Reuse Existing Architecture**：用户明确确认 §5、§6、§6.1、§7 后，已在 #29/#301 记录冻结设计、用户结果、上述引用、原 owner/调用路径、不变量、验证、唯一 Writer 及 **Ready**，再删除完整专属闭包/旧测试/专属输入，并精确校准四条对应登记。不新增生产函数/类型/入口、公共领域合同、事实/schema、权限/tenant/身份、状态机/事务、幂等、retry/UNKNOWN/recovery 或 provider 副作用 owner。

Must：§5 保留能力及现有状态/规则、provider/预算/计量/凭据和 Store 合同不减；两套及其它 workflow/必需和安全检查/保护/ceiling 不改。Should：原 guard、当前文档和源码一致。Out of Scope 如 §1；无新增 Accepted Risk。Threat Model 限误删共享 batch/规则/API、把错误统计抽成有效规则、混淆 Task/provider/持久化 owner、复活旧入口，以及误报运行态清零。

若发现当前消费者、当前外部可观察兼容义务或上述保留边界变化，停止对应删除并报告；不得自行增加兼容路径。若实际需要改变高风险边界，补适用 Independent Architecture 准入，不在实现中临时发明架构。

## 3. 实际消费者、注入与调用路径

基线全仓 Go 搜索：`NewVariantFilterHandler`、`NewAISkuMappingHandler` 只有定义，无外部构造、直接实例化或注册接线。三个 filter 专属类型/构造器只在该闭包内相互引用。`go list ./cmd/...` 共 **31 个当前命令**，其依赖并集中 `internal/temu/sku` 与 `internal/temu/filter` 均为 **0**。未运行 main；这是源码/静态图证据，不是实际实例/provider 验收。

| 合同 → 实现 | 注入、调用和消费者 | 边界 |
| --- | --- | --- |
| FilterRuleAPI → VariantFilterHandler → FilterRuleHandler | 无装配入口的 constructor 注入 API，再构造 rule manager、ProductFilterChecker、RuleValidator、stats provider；HandleTemu 委托 FilterVariants | 退休整个专属编排/规则包装闭包；有效规则在原 marketplace owner 保留，API/仓库及模型保留 |
| FilterRuleManager → API.GetFilterRule | 从旧 Task 取 TenantID/StoreID，CategoryID>0 时传入；只做请求装配、日志与错误包装 | 不是持久化 owner；不抽取新的 Task→当前身份 adapter |
| FilterRuleHandler | 主商品/变体筛选、读取失败跳过、保存第一个 status=0 的 FilterRule 到 context、过滤后原地替换 Variants；全部过滤时只告警；GetFilterRuleStats 委托专属 stats provider | 旧 Task 门控、fail-open、主商品回退、in-memory 统计不迁入当前应用；仅由它消费的两个旧规则包装器按 §4.1 退休 |
| AISkuMappingHandler → NewClient → SkuBuilder.variantProcessor | 有 config 时复制 config/注入 logger、调用旧 openai.NewClient，再 NewSkuBuilder；Handle 选输入，调用 GenerateAISkuMapping 后 SetAISkuMapping | 退休这个无消费者的 provider 构造/Task 前置入口；不是退休整个 SKU 映射或 provider transport |
| AIHandlerInput/buildAIHandlerInput | 仅由上述 AI handler 消费；按 variants、主商品、空列表选择并携带日志 Source | 只删专属类型/函数，不抽成当前主变体或 Product 合同 |
| AIBatchInput/buildAIBatchInput | 同文件定义；TotalBatches/BatchVariants 被 ai_mapping_batch_processor.go 的 SkuVariantProcessor 路径实际消费 | 保留整个 batch 合同、方法/校验/切片语义、fmt/model import；只移除退出 Task helper 唯一使用的 temucontext import |

`SkuBuilder → SkuVariantProcessor → BuildAIMappingInput / batch / single processor` 仍有源码消费者，`ai_mapping_input.go`、AI prompt/response/validator/mapping helper、SKU 构建/规格/图片/定价及 mixed context 全保留。不因为“AI mapping”关键词删除共享代码。filter 的 sensitive words、prohibited items 和 marketplace/productpolicy 非专属闭包；`sku/ai_mapping_validator.go` 与拟退休的 `rules/rule_validator.go` 不是同一对象。

消费者刷新确认：ProductFilterChecker 只有 FilterRuleHandler 一个生产构造调用；RuleValidator 只被上述二者构造，字段本身没有生产方法调用，直接方法调用只在专属旧测试；FulfillmentChecker 只被 RuleValidator 消费。FilterCheckResult 全仓仅这三个包装器及其定义文件引用。不能将“仍有一个待删旧 caller”误当当前有效消费者，继续强制保留整条不可达包装链。

## 4. provider、持久化 owner 与已存在的状态冲突

| 行为 | 原 owner 与实际效果 | 本轮约束 |
| --- | --- | --- |
| 规则查询/持久化 | FilterRuleAPI → gormFilterRuleAPI → GormFilterRuleRepository.ResolveFilterRules；读取 listing_filter_rule，tenant/deleted/owner scope 与品类→店铺→默认规则优先顺序归仓库。CRUD、状态、审计事实也归原仓库 | 保留 API/DTO/ToFilterRule、模型/表/索引、查询/CRUD 与 tenant/owner 安全测试；不访问/改写真实数据，不转交事实 owner |
| 旧 API/context 限制 | gormFilterRuleAPI 使用 context.Background，旧 DTO 无当前用户身份/context；ResolveFilterRules 自有 applyOwnerScope | 保留不等于当前 Organization/owner 授权或取消/deadline 验收 PASS；不增加该旧合同当前私有数据消费者。若发现当前 Must 缺陷，报告具体路径再决定最小修复，不扩成 IAM 或新试用前置 |
| AI 请求 | handler 构造旧 openai client；实际 GenerateAISkuMappingSingleBatch → callAIAPI → ChatCompleter.CreateChatCompletion，使用 WithAITimeout(context.Background()) | 潜在网络/计费副作用，不能当纯计算。设计期连 constructor 都不执行；共享 processor/client/prompt/解析与已有预算/审计/UNKNOWN owner 保留，不新增消费者或调用 |
| 结果/恢复 | 目标只写旧 context 的 FilterRule、Variants、AISkuMapping；stats 是临时 map。无专属表、durable 状态机、事务/outbox、幂等或恢复记录 | 不新建持久化结果，不判定历史请求成功/失败、不释放预占/锁、不重放 UNKNOWN；源码删除不删除历史资产 |
| 当前 Product/Store/worker | Product facts、Store 观察同步、登录 session/凭据/Profile/browser、健康/关闭与 Store sync 继续各自原 owner | 不接线、不转交身份或 durable assets，不操作运行态 |

保留旧 client/processor 源码不证明它具备当前 Product/Agent 的 provider 准入、预算、计量、审计或 UNKNOWN 合同；本轮不把该旧 Task 路径接入当前应用，不新增调用，也不因此新建第二套配额、审计或恢复 owner。真实 provider/费用/历史操作状态均未访问，不能据源码退出判定成功、零成本或可重放。

**状态冲突已查明**：listingadmin.RuleStatusEnabled=0、RuleStatusDisabled=1，持久化默认也是 0；基线 ProductFilterChecker 跳过非 0，FilterRuleHandler 也按 0 选择启用规则。但专属 FilterRuleStatsProvider 把 1 计作 enabled。根因是旧编排自己重复解释状态常量，与有效合同相反，不是需要翻转现有 schema/API 或新建统计服务。

Finding: 旧统计启用态与有效状态合同相反。Product requirement affected: 保留有效规则/状态，不将错误统计重新接入当前产品。Classification: **NOT_APPLICABLE（当前用户路径/试用阻塞）**；该链没有当前装配消费者，无证据影响当前 Must。Action: 按 RETIRE 删除专属统计及上游闲置编排，保留原 status=0 合同和有效校验/测试；不为维持旧统计可用做单独 bugfix、补新入口或改变业务状态。

### 4.1 规则包装器与有效行为映射

设计期独立复核提出的“两个包装器没有保留消费者”已分类 **IMPLEMENTATION_TEST**，影响本设计 §5 闭包和保留 guard 的准确性，不构成当前试用/生产 Blocker。完整专属闭包已在冻结前补齐并完成增量复核，用户随后确认；不增加当前产品需求。有效行为已有正确 owner，无须复制或新增 EXTRACT 生产实现：

| 原行为 | 当前有效 owner / 保留决策 | 旧包装的处置 |
| --- | --- | --- |
| 价格上下限、评分下限 | productpolicy.RuleChecker.CheckPriceRange / CheckRating，保持 nil 门槛与边界比较；价格选择 GetProductPrice 原样保留 | ProductFilterChecker 的简化重复检查/字符串诊断与 RuleValidator 的 Task/StoreInfo 装配退出，不将旧结果 DTO 作为当前合同 |
| 库存、评论数、配送 | productpolicy.RuleChecker.CheckInventory / CheckReviewCount / CheckFulfillmentType，GetInventory、IsFBAFulfillment/IsAMZSeller 原样保留；库存上限30等已有确定性规则不改，SHEIN validation 仍消费共享 RuleChecker | RuleValidator/FulfillmentChecker 只是委托和日志/DTO 包装；专属 buildFulfillmentProductValue 与 FilterCheckResult 一并退休，不删除共享算法/API/现有消费者 |
| 启用状态/作用域 | listingadmin.RuleStatusEnabled/Disabled/IsRuleStatusEnabled、规则 API/DTO、ResolveFilterRules 与其 tenant/owner/优先顺序测试保留 | 退休旧 Task 的筛选循环/上下文回填及错误统计，不改变持久化 status=0 合同，不新建替代循环 |
| 固定源图数≥3、先图后其它规则、Task/context 分支与详细日志值 | 全仓依据仅 RuleValidator 及其专属旧测试，未发现当前 Product/Architecture/平台合同要求固定三张 Amazon 源图；旧测试不是 Architecture Authority | RETIRE 该旧 Task 门槛和两个旧测试，不复制到 marketplace，不修改当前平台图片/素材校验、API/DTO、采集或 SKU 图片能力；若刷新发现具体当前合同证据，停止对应项并报告用户 |

此处保留的是仍有效的 marketplace 规则和原合同，不为旧包装器增加新 caller，不用新 wrapper 维持旧 Task 可用。

## 5. 精确删除与有效规则保留矩阵

以下为**用户确认后已实施的精确矩阵**；§3/§4 中调用关系描述删除前的取证基线。有效保留实现没有改动。

| 对象 | 决策 | 删除/保留条件 |
| --- | --- | --- |
| sku/variant_filter_handler.go（57 行） | RETIRE 全文件 | 退出 Task→FilterVariants 外壳和 logger 接线；保留有效筛选规则/API |
| sku/ai_mapping_handler.go（94 行） | RETIRE 全文件 | 退出旧 provider constructor/Task 输入选择、GenerateAISkuMapping 前置调用和 context 回填；不新建替代入口，现有 processor/client 保留 |
| filter/rule_handler.go（204 行） | RETIRE 全文件 | 退出无消费者主/变体门控、错误继续、context 选择/回填和 stats 委托；不迁移这些 Task 策略 |
| filter/rule_manager.go（56 行） | RETIRE 全文件 | 退出旧 Task 请求装配，保留 FilterRuleAPI/DTO/仓库与原查询优先顺序 |
| filter/rule_stats_provider.go（55 行） | RETIRE 全文件 | 无外部消费者，错误状态计数不抽取、不改造成新统计能力；保留 status=0 原事实合同 |
| product/filter_checker.go（106 行） | RETIRE 全文件 | 仅由待删 FilterRuleHandler 构造；其简化价格/评分规则已有 §4.1 正确 owner，不保留重复 Task adapter |
| rules/rule_validator.go（230 行） | RETIRE 全文件 | 仅被该闭包构造，生产无方法调用；有效共享规则按 §4.1 原位保留，旧图片门槛/Task 分支/日志 DTO 不迁移 |
| handlerbase/fulfillment_checker.go（71 行）、filter_types.go（11 行） | RETIRE 两文件 | 仅 RuleValidator 使用的配送包装/诊断 helper，以及仅三包装器引用的 FilterCheckResult；保留 productpolicy 配送算法，handlerbase 其它合同不改 |
| rules/rule_validator_test.go（113 行） | RETIRE 专属旧测试 | 两个测试仅断言固定三张源图及先图后其它规则的旧 Task 行为；不删除 marketplace/status/tenant/provider 的有效测试 |
| sku/ai_mapping_handler_input.go 中 AIHandlerInput/buildAIHandlerInput 与唯一不再使用的 temucontext import | RETIRE 专属部分 | 刷新仅旧 handler 消费；文件、AIBatchInput、buildAIBatchInput、TotalBatches/BatchVariants 与 fmt/model import 全保留，不整文件删除 |
| SkuVariantProcessor、SkuBuilder、AIMappingInput/BuildAIMappingInput、batch/single/prompt/response/validation/mapping helpers、AI DTO/混合 context、provider client | 保留原合同/实现/源码消费者 | 不转交模型调用 owner，不新增当前入口、retry/计费/预算语义或复用旧 Task 作为当前模型 |
| FilterRuleAPI/ReqDTO/RespDTO/ToFilterRule、GORM API/仓库/模型、RuleStatusEnabled/Disabled/IsRuleStatusEnabled、marketplace/productpolicy 和仍在使用的 SHEIN validation | 保留有效规则/状态与持久化 owner | 不删除表/索引/数据，不改变 0=enabled；不复制已有算法或引入新的规则/状态事实源；旧规则包装器本身按上列退休 |
| 其它 filter/sku/product/rules/context、平台 API/同名 DTO、Reader/Cache/Mapping、两种 SHEIN 属性、PriceManager/PriceCalculator、采集/Store 观察、独立登录/Store sync | 全部保留 | 仅按精确路径/符号删除，不按包/关键词整删；不增加内部试用前置 |

Legacy decision: **RETIRE 九个专属生产文件、一个旧测试文件和混合输入专属部分**。
Reusable behavior: 有效规则/状态、批次切片与 SKU 映射均已有 owner，保持原实现，无新增 EXTRACT 生产代码。
Current owner: marketplace/productpolicy、TEMU 规则/SKU、listingadmin 规则事实、原 provider client、Product/Store/独立 worker 各保持。
Cutover/deletion condition: 用户确认冻结 §5/§6/§6.1/§7、Issue Reuse Existing Architecture / Ready；刷新无新消费者，原退休 guard 先 RED 后精确删除 GREEN，必要保留回归、四条对应登记及准确候选 CI/最终检查通过。源码退出不授权运行态或数据退出。

## 6. 验证与交付

设计阶段仅提交本文与架构索引，设计 CI/review 不代表实施或真实使用 PASS。用户确认后，原退休 guard 已先在目标源码仍在时 RED，再按 §5 删除后 GREEN；三条对应 import 豁免和一条 constructor 登记精确退出，完整扫描和其它登记不减。实现验证和最终检查的准确证据维护于 PR。

用户确认后的最小实施检查：

1. 刷新五编排类型、三个规则包装器/专属结果 DTO 和 mixed input 的消费者及 31 命令依赖；确认专属闭包不新增当前调用，否则停止对应删除并报告。
2. 扩展原 TestListingKitTaskBackendSourcesAreRetired：九生产路径/旧测试缺席、限定旧 sku/filter/product/rules/handlerbase 范围的准确专属符号缺席；先确认源码存在而 RED，再精确删除 GREEN。复用现有 AST 能力，不全仓子串禁止 FilterRule、AI mapping、RuleChecker 或 shared batch 名称。
3. 原 guard 保留 mixed input 文件、AIBatchInput/构造函数/两个方法及 batch processor 消费、SkuVariantProcessor、有效状态/FilterRule API/DTO/仓库、productpolicy RuleChecker/价格库存/配送原位与仍有的 SHEIN 调用，不为退休包装器增加保留断言。无保留业务行为变化，不新增替代状态统计、AI 调用或验收框架。
4. 保留包适用编译：TEMU sku/filter/product/rules/context/API、marketplace/productpolicy、现有 provider/worker；复用原 marketplace/TEMU 规则、listingadmin status/FilterRule scope/tenant/owner、batch 所在 SKU 的既有有效回归和原命令编译。没有直接 batch 专属测试不等于可删除其实际消费者；不把编译冒充行为验收，也不为删除代码复制实现测试。
5. 稳定实现候选走原必需 CI、一次准确实现 HEAD 独立最终检查；不人工重复完整 CI。区分本地静态图/fixture/编译、PR CI、合并后匹配 main CI、实际 provider/实例/产品验收；后者本批 NOT_RUN，不签发生产可用或 legacy 全量清零。

### 6.1 原登记、guard、文档与 workflow 精确处置

| 工件 | 本次改动 | 保留边界 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 扩展原九生产路径/一旧测试/专属符号缺席、shared input/状态/productpolicy/provider 保留 guard；已核对无强制保留这几个包装文件的原断言 | 原当前命令依赖/独立 worker/全部安全断言保留，不 skip 缺文件或移除 gate |
| tests/import_boundaries_test.go | 精确移除三条对应消失路径：TestTEMUOpenAIImportsStayAllowlisted 的 ai_mapping_handler.go；TestTEMUProductStoreAndSchedulerRetiredManagementImportsStayBlocked 的 product/filter_checker.go；TestTEMURuntimeAndBridgeRetiredManagementImportsStayBlocked 的 handlerbase/fulfillment_checker.go | 三表分别保留其它11/5/4条，roots/deny/graph/ceiling不变，不借机清理其它旧条目；其余目标未发现其它硬编码路径豁免 |
| tests/integration_provider_logger_contract_test.go | 仅从 TestProductionProviderConstructorInventoryIsComplete 移除 ai_mapping_handler.go 的一项 openAI:1 | 保留完整扫描/相等断言和其它三项 provider constructor；其它 logger/LLM 检查不降级，不因计数变化改扫描范围 |
| Legacy register、phase2-runtime-inventory 当前校准、本文/架构索引与直接冲突说明 | 确认后校准九生产文件/旧测试/专属输入 source-retired 与有效规则的原 owner | filter 仍有 sensitive/prohibited 消费 logger，sku 仍有 logger/openai：三个当前消费者行保留。历史清单/图/证据/ceiling 不改，不提前写已退休 |
| ci.yml / commercial-readiness.yml / 其它 workflow、.golangci.yml、保护规则 | **零改动** | 必需/安全检查、原 race/编译目标、分类/触发/权限/并发、Required CI Gate、所有保护/ceiling 保留；原 SKIPPED 不记 PASS |

## 7. 确认与实施停止点

1. #754 先以 Draft PR 交付设计、准确设计 HEAD CI 和独立设计检查；#29/#301 保持 OPEN，原正文历史保留。旧设计证据不得冒充实施结果。
2. **用户已明确确认冻结矩阵**，在同一分支/PR 继续；生产修改前登记 Reuse Existing Architecture / Ready、冻结设计/确认记录、唯一 Writer 和 §2 不变量。设计确认授权上述有界实施，不代替准确最终 HEAD 的合并授权。
3. 获准实施时同步一次实际 main，最终候选前再核对一次，只处理真实组合风险，不持续机械 merge。出现新消费者/当前兼容义务或保留合同/事实/权限/状态/副作用变化，停止对应项并报告，不擅自扩删除或加 wrapper。
4. 按矩阵删除/登记/原 guard RED→GREEN、必要回归、准确实现候选 CI 和独立最终检查后交付 In Review；设计证据保持历史，不冒充实现结果，不以无变化代码反复审核扩大流程。
5. 合并仍需用户准确 HEAD 的单独授权，遵守 expected-head/保护，无管理员绕过。设计或实施都不授权 Issue 关闭、部署、真实 provider/实例/凭据/数据操作，不增加内部试用前置。
