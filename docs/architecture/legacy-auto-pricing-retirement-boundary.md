# 旧 SHEIN/TEMU 自动核价链退休边界

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

- Status: **PROPOSED / DESIGN_ONLY，待用户确认 §5、§6、§7；未授权生产实现**。
- 调查基线：`main @ 45e1117371a8525ee60e206250ede7f5546699af`。#742 已按准确 HEAD 合并，其实际 push/main CI 证据保留在原 PR，不作本设计候选的直接检查结果。
- 本轮 Design Basis: **N/A**，仅只读调查、设计与索引文档，不改变产品、领域合同、运行语义、权限、状态、持久化或外部副作用。未来有界源码退休拟采用 **Reuse Existing Architecture**，用户确认本矩阵并在 Issue 记录 Ready 后才能开始，不以设计 PR/CI/review 当实施授权。
- 用户边界：保留 PriceManager、有效平台 API/规则、共享合同、两种 SHEIN 属性类型、当前采集、Store 观察同步、独立 SHEIN 登录 worker 及其 Store 同步；两套 workflow 不改，不操作运行实例或数据，不增加内部试用前置。

## 1. 用户结果、范围与非目标

使用者是维护当前内部试用源码、平台能力和独立登录 worker 的维护者。本设计交付可逐项核对的旧核价闭包及保留边界，避免把已失去装配入口的自动核价服务重新接入，或误删同包的价格查询、纯规则及共享合同。它不解除一个新的试用阻塞，也不把旧代码清理变成用户试用前置。

拟退出旧 SHEIN 自动核价服务的读取规则/映射、评估/再次议价、提交与旧接口回退，以及 TEMU 自动核价/决策、数字 Store/规则/映射读取、Amazon 辅助缓存/重试和专属 runtime adapter。精确源码/测试矩阵见 §5。保留同包有效价格 API、公式与规则选择行为，不按 `pricing`、`Rule`、`Store` 或数字字段批量删除。

不设计新的核价产品、定时执行器、活动报名、批量重新上架、刊登流程、平台客户端或通用 recovery/IAM/验收平台；不增加 UI/菜单或外部账号操作。当前官方 Product/Store/Submission、定价准备、来源采集和登录仍由原 owner 负责。本轮无需引入新开源组件，调查和未来实现均复用现有 Go/仓库工具、原测试和已有合同。

## 2. Design Basis、Authority 与不变量

适用依据：

- [全新系统基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)：仅 EXTRACT / RETIRE，无旧数据迁移、旧 ID 映射、旧 Service 包装、fallback、双读双写或第二事实源。
- [旧 CLI](legacy-listing-cli-retirement-boundary.md)、[control-plane](legacy-listing-control-plane-retirement-boundary.md)、[数字执行链](legacy-numeric-task-execution-retirement-boundary.md)、[任务持久化](legacy-numeric-task-persistence-retirement-boundary.md)、[商品/库存同步](legacy-numeric-product-inventory-sync-retirement-boundary.md)：旧入口/派发/执行/任务与同步闭包已退休。只有用户确认后，本设计才替代这些批次对 §5 核价实现的保留待后续决定；原矩阵和检查结果保持历史证据。
- [独立 SHEIN worker 身份/持久化](shein-worker-identity-persistence-boundary.md)：账号、Cookie/attempt、Redis/browser/Profile、同 tenant 查重及登录后 Store 同步保持原 owner，不改身份/持久化格式，不恢复已退休 HTTP/bridge。
- [当前 Store](store-center-current-application-v1.md)、[Store observations](store-center-platform-observations-v1.md)、[准备接线](store-observations-preparation-runtime.md)、[当前 SHEIN 业务](my-supply-chain-shein-v1.md)：当前 Organization、官方连接、观察、Submission 及 UNKNOWN/retry 责任不由旧核价服务接管。没有 UI/IA 变化，Figma N/A。

未来实施不新增/转交当前事实或 schema owner、公共当前领域合同、身份/权限/租户规则、状态机、事务、幂等键、retry/UNKNOWN/recovery 或平台写入责任。退出的类型仅是无当前消费者的旧服务协议；保留的 PriceCalculator 等合同不改签名/行为。本矩阵涉及约 1,991 行专属生产文件删除及一个混合接口文件收缩；不是通过搬目录、拆 PR 或包装旧 owner 绕过准入。现有已批准边界足以覆盖该源码退休，拟复用架构而不另造核价体系；若刷新发现须改变保留边界，则停止对应部分并升级适用 Independent Architecture。

Must：§5 保留清单完整；有效公式/规则/安全/租户/owner/平台检查继续；两套 workflow、必需/安全检查及保护规则不改；真实持久化、平台操作与恢复不移交。Should：当前源码地图、guard 清单与实际消费者一致。Out of Scope：真实配置/凭据、DB/Redis/Profile/平台探测或 dry-run，实例启停/部署、迁移/清理/补写/重放、其它活跃批次及治理工具。无新增 Accepted Risk；Threat Model 只覆盖误删能力、误转交事实/副作用和把源码缺席当运行态清零，不扩大到全局安全重构。

## 3. 实际消费者：合同 → 实现 → 注入 → 入口

基线上执行 `go list -f '{{.ImportPath}}|{{.Deps}}' ./cmd/...` 并核对全部 **31 个**当前命令，**没有一个命令依赖 `internal/shein/pricing` 或 `internal/temu/pricing`**；未运行命令 main。全仓 Go import、构造器和符号检索区分源码消费者、测试构造及真正装配，不把包编译/函数存在当用户入口。

| 合同/实现 | 基线实际消费者/装配 | 拟处置与限制 |
| --- | --- | --- |
| SHEIN `AutoPricingService` → `autoPricingServiceImpl` | NewAutoPricingService 只有定义，无包外生产构造；auto_pricing_repo_test 在同包构造旧实现。读取 PricingRule/Mapping，借 API 接受/拒绝/再次议价 | RETIRE 服务与专属评估、请求构造、时间窗口及测试，不提供替代构造器 |
| SHEIN `PriceManager / CostPriceRequest` | 全仓唯一包外生产 import 是 `shein/product/product_repo.go`，NewProductAPI 构造 pricing.NewPriceManager 并保存为产品 API 的专职价格查询处理器 | **完整保留 price_manager.go、原 ProductAPI 接线、所有方法/字段/DTO**；这是明确源码消费者，不声称已验证当前用户入口或真实 API |
| SHEIN `PricingAPI` / 平台 pricing/product API、client | pricing_repo.go 实现原平台 transport；独立 `shein/api/pricing` 定义接口/请求/响应，PriceManager 使用 product DTO/client | 全部保留，不因旧服务少一个 caller 删除平台 API、endpoint、凭据处理或错误合同 |
| SHEIN `AutoPricingCalculator / CostProfitCalculator` | 原 calculator_test、cost_profit_calculator_test 验证独立公式/区间/默认值/成本利润行为；旧 service 使用 AutoPricingCalculator | 按用户保留有效规则的要求原样保留四个源码/测试文件，不因 AutoPricing 名字删除纯规则，也不把保留当新装配授权 |
| TEMU `AutoPricingService → PricingDecisionService` | 全仓没有 `internal/temu/pricing` 包外 import；两个构造器只有同包调用/测试。自动服务分页后调用 AcceptPrice/RejectPrice；再次议价分支并没有完整执行器 | RETIRE 编排、旧策略决策/统计/缓存/重试，不补齐旧功能，不把它们迁入当前应用/worker |
| TEMU PricingRuntime、StoreConfigService、PricingDataService | runtime source 接口仅内部 adapter/测试；读取 Store/PricingRule/Mapping，带原本地仓库→API 回退，没有当前生产装配 | RETIRE adapter 与专属 port/DTO 转换、reader 和测试，不替换成当前 Store/Observation adapter |
| TEMU `PricingRuleCalculator / PriceCalculator` | 原 calculator 委托 `marketplace/temu/publishing/pricing_policy.go` 的 CalculateMinAcceptablePrice / SelectApplicablePricingRule；原 test 保证计算日志 | **保留两文件、PriceCalculator 两方法与 listingadmin.PricingRuleRespDTO**。实际公式 owner 已在 marketplace；不复制、搬移或新建第二规则 owner |
| 两种 SHEIN 属性、活动/刊登及共享规则 | productsync 两类型仍被 activity 使用；当前 publishing/marketplace 和其它源码仍消费 listingadmin/root runtime 合同 | 全部保留，不借核价退休启动活动、发布、mapping 或整个共享层清理 |
| 当前观察/采集与独立 worker | 当前 cmd 依赖图不经两个核价包；原 Store observations/官方端口/Temporal 与登录账号→StoreAPI→managed 同步/查重仍分别装配 | 原接线、能力/不可用语义、配置、权限与恢复保持；不启动进程或接管原运行 owner |

架构问题是混合 pricing 包同时包含无装配的数字 Store 核价编排和可保留的 API/公式。SHEIN 原服务在 BatchReQuote 返回错误后直接尝试 BatchHandleCostDiscuss；这没有证明第一次平台动作未发生，不能当可复用的 UNKNOWN/recovery 协议。TEMU 的本地→API 回退、Background context、内存 cache 和 platform mutation 也不是当前的统一事务/恢复 owner。处理方式是退出旧闭包而保留原 API/规则，**不修补或包装闲置服务以继续运行，也不引入新恢复系统**。以上是静态协议风险，未读取真实状态或确认发生过平台事件，不把它变成当前独立试用前置。

## 4. 持久化、事务与外部副作用 owner

| 边界 | 基线职责 | 退休/保留条件 |
| --- | --- | --- |
| `listing_pricing_rule` | listingadmin 原模型、PricingRuleRepository/GORM API；服务 ListByStoreID/GetPricingRule 只读取并映射规则 | 只退出本次 caller/转换，不改仓库、原 owner/tenant/权限、schema/索引/行、DTO 或公式。marketplace 纯规则继续消费 PricingRuleRespDTO，不创建新规则表 |
| `listing_product_import_mapping` | 原仓库 FindLatest、ProductImportMappingAPI；SHEIN 按平台商品、TEMU 按 SKU/Store 读取成本、倍率、tenant/region；原仓库拥有 scoped 查询/更新责任 | 保留全仓库/API/query/事务/字段（含 ImportTaskID）及原测试，不将旧映射导入当前 Product/Store/Submission，不转交历史事实/恢复 |
| `listing_store`、OperationStrategy | 原 StoreRepository/StoreAPI 和 root listingruntime 合同；TEMU reader 投影配置，含 rebargain、price type/reject strategy 等 | 只删除专属配置投影/reader；保留模型/API/owner/字段和有效安全断言。登录 worker 仍依赖它，不能删其 provider、Store 写回或查重 |
| Amazon fetch/cache | 旧 PricingDecisionService 的 sync.Map/CacheKey/TTL 是进程内缓存；共享 appfetcher/sourceproduct/raw JSON 原 owner 处理获取/缓存 | 专属 cache/等待重试/helper 随服务退出。共享 fetcher、HTTP/browser/cache/RawJSONData、权限/结果作用域/取消行为与原测试不改，不搬缓存/历史或新建队列 |
| 平台报价写入 | SHEIN BatchReQuote 与旧接口 fallback；TEMU 读取待核价后逐 SKU AcceptPrice/RejectPrice。统计和决策多为内存，平台动作和本地读取不构成共享事务 | 只退出调用编排，平台 API/DTO/transport 保留但本轮不调用；不转交发送、幂等、retry、UNKNOWN、补偿、对账或恢复责任，不把内存统计视为 durable receipt |
| 当前业务和独立登录 | 当前 Organization/Store/观察/Submission 以及原 worker 的账号 DB、Redis Cookie/attempt/stream/Lua、browser/Profile、锁、Store ID 同步 | 全部沿原 owner 与格式，不改变状态机/事务/租户、Cookie 提交与后续 Store 同步边界，不将旧核价配置或账号映射到当前系统 |

失败、响应丢失、重试、重启、取消、并发与恢复均不建立新协议：本批只删除旧实现，不对在途平台动作作完成/回滚推断；真实历史和最后写入/恢复仍归原运行/数据 owner。删除源码不代表旧实例已停止、平台动作已确认或真实表可清理，也不宣称 exactly-once。

共享 `core/config.AutoPricingConfig`、PlatformConfig、loader/default/validator/registry、Store.EnableAutoPrice/EnableRebargain/TemuPriceRejectStrategy/PriceType、PricingRule/mapping DTO、运行 manifest/环境变量/部署配置均保留。它们仍有其它配置/合同消费者；删除一个服务不授权收缩共享字段或安全门禁。本设计没有访问真实 DB/Redis/文件 Profile、凭据或平台。

## 5. 待确认的精确源码删除矩阵

**仅为提案，当前源码未删除。** 拟 RETIRE **16 个完整文件（11 生产、5 测试，共 2,470 行）**，并精确收缩 **1 个混合生产文件**。这不是数据库/运行资源删除清单。未列入矩阵的生产文件和有效测试全部保留。

| 对象 | 拟处置 | 删除条件 / 明确保留 |
| --- | --- | --- |
| SHEIN 5 个专属生产文件 | RETIRE `auto_pricing.go`、`pricing_evaluator.go`、`pricing_calculator.go`、`pricing_builder.go`、`time_helper.go` | 无包外服务/请求 builder/时间 helper 消费；服务接口/实现、PricingDecision/Statistics、旧 DTO 转换、fallback 请求/调用和专属时间窗口一起退出；不留 alias/stub/转发构造器 |
| SHEIN 2 个专属测试 | RETIRE `auto_pricing_repo_test.go`、`time_helper_test.go` | 前者只验证已退出 reader/评估接缝，后者只验证该专属时间 helper；不削弱平台 API 日期 wire 字段或有效规则测试，不搬旧 fixture 维持 service |
| SHEIN 6 个保留文件 | 原样保留 `price_manager.go`、`pricing_repo.go`、`calculator.go`、`calculator_test.go`、`cost_profit_calculator.go`、`cost_profit_calculator_test.go` | PriceManager/CostPriceRequest、产品 API 装配、平台 transport、纯公式/区间/默认值/成本利润及全部断言不变；不整删 pricing 包 |
| TEMU 6 个专属生产文件 | RETIRE `auto_pricing_service.go`、`pricing_decision_service.go`、`pricing_rules.go`、`pricing_data_service.go`、`store_config_service.go`、`runtime_adapter.go` | 全仓无包外消费者/当前命令注入；包括专属 runtime/source/store port、ServiceConfig/PricingContext/CacheKey/CacheItem、Background reader/DTO 转换、cache/retry 和 mutation 决策；不将其专属策略认作共享纯公式 |
| TEMU 3 个专属测试 | RETIRE `pricing_data_service_test.go`、`store_config_service_test.go`、`pricing_rules_test.go` | 旧 reader/Store 投影和旧 Amazon cache retry helper 随服务退出；后者取消测试只绑定该退役 helper，共享 fetcher 的 context 传递/取消测试及平台安全规则完整保留，不加新的通用 retry helper |
| TEMU `interfaces.go` 混合合同 | 只删 StoreConfigProvider、ProductDataProvider、ProductFetcher、DecisionMaker、PricingProcessor 五个无外部消费者的旧服务接口及专属注释；相应清理 context/model/temupricing imports | **PriceCalculator 两方法、listingadmin import/DTO 与 package 保留**；不改签名、增加替代接口、rename/wrap 或转移事实 owner |
| TEMU 2 个保留文件及 marketplace 规则 | 原样保留 `pricing_rule_calculator.go`、`pricing_rule_calculator_test.go`；保留 `marketplace/temu/publishing/pricing_policy.go` 及其原测试 | 公式、默认倍率、区间选择、原日志/DTO 合同保持，不因无旧 service consumer 再清理 adapter 或复制到新 owner |
| 两平台有效 API/规则、Product/Store/Mapping/RawJSON/规则与共享层 | 全部保留 | 包括 SHEIN/TEMU api/pricing、SHEIN PriceManager 的产品接线、client/凭据/endpoint、活动/刊登、listingadmin/root listingruntime/当前 publishing、所有 shared 配置。不从移除 caller 推导整个仓库/字段可删 |
| 两种 SHEIN 属性、当前采集/Store observations/独立登录和 Store sync | 原样保留 | productsync 的 EnrichedSkuInfo/EnrichedSkcInfo、完整 product_sync_types.go/marketing_client_boundary_test.go 及当前消费者保持；worker 各原能力/配置/格式/恢复不改；#301 不据此宣称清零 |

未来实施开始前重新核对本矩阵的外部构造/类型/fixture 消费者和全部当前命令图；只允许删除已确认闭包。若发现新消费者，不以 missing skip、兼容壳或新增豁免绕过；停止相应项，报告具体当前 requirement，由用户决定后续范围。

## 6. 既有测试、guard、文档与 CI 的处置

下表也是待确认的未来实施计划，本设计阶段不修改测试/guard/生产源码。

| 既有对象 | 同批拟处置 | 必须保留 |
| --- | --- | --- |
| `tests/listingkit_backend_retirement_test.go` | 原 absence 清单加入 §5 16 个文件，混合接口禁五个退役声明；沿原读文件检查确认保留文件/PriceCalculator/PriceManager 存在，扫描实际两个 pricing 包禁止旧服务/adapter 声明复活 | 先在旧源码存在时得到目标 RED，删除后 GREEN；两包都保留，**不禁止整个 pricing import root**；原当前命令/worker 依赖与全部旧 absence 不减，不建新扫描器 |
| `TestSheinRetiredManagementImportsStayBlocked` | 仅删 pricing/auto_pricing.go、pricing_calculator.go、pricing_evaluator.go 三个退出 allowlist 项 | calculator.go/test、activity、productsync 两类型及其它条目和原 deny 断言保持，不新增豁免 |
| `TestTEMUSyncAndPricingRetiredManagementImportsStayBlocked` | 仅删 auto_pricing_service、pricing_data_service、pricing_decision_service、store_config_service 四个退出 allowlist 项 | **保留 interfaces.go / pricing_rule_calculator.go 两项及 pricing 扫描根**，原禁止导入断言不变；整个 sync 缺席仍由原 absence 保护，不删/固定通过此 test |
| `TestTemuPricingRuntimeUsesCapabilityNames` / `TestTemuPricingFallbackLogsUseCapabilityNames` | RETIRE 两个只读取本次退出 adapter / reader 文件的旧结构/日志守卫 | 以原 absence/符号检查替代旧工件约束，不恢复文件、不在 read/parse 失败时 skip |
| `TestTemuPricingUsesManagementAPIPortAliases` | **完整保留**；仍扫描保留的 pricing 包 | 与上两项不同，此检查仍有当前源码对象，不能机械照搬 #742 的三项 sync guard 删除 |
| `TestTemuRuntimeErrorsUseCapabilityNames` | paths 只减 pricing_data_service.go / store_config_service.go | 三个现存 API client 路径、禁止短语和读取失败语义保留 |
| `TestRetiredManagementRuntimeNamingHotspotsUseCapabilityNames` | 只减 TEMU auto_pricing_service.go / pricing_decision_service.go 两项 | 其它命名检查与原 fail 行为保持，不扩大旧有历史缺席处理 |
| 已有公式/平台/owner/context 测试 | 原样保留 SHEIN calculator/cost_profit、TEMU calculator 和 marketplace pricing_policy、平台/Store/owner/worker/当前观察与 shared fetcher context 测试 | 旧服务 test 随旧实现 RETIRE；有效安全/隔离/公式/取消断言不能借此删除，不为形式新增 runner/故障矩阵 |
| 当前文档 | 实施时校准本设计状态、architecture README、legacy-register、原 #742 当前保留说明、必要现时地图/runbook/两份 guard 清单 | 当前清单移除两个退役守卫；保留 port alias/import/absence 等仍有效条目。历史矩阵、日期和结果不回写 |
| `docs/refactoring/phase2-runtime-inventory.md` | 实施后实际依赖确认再只减当前登记的 core/logger → shein/pricing 与 → temu/pricing 两行 | 保留 logrus 依赖、其它消费者、原 ceilings 与历史/dated inventories；不能将保留包误记为整个目录退休 |
| 两套 workflow：`ci.yml` / `commercial-readiness.yml` | **零改动** | 原触发/选择/权限/并发/保护、必需和安全门禁、完整 listingadmin race、current-application / shein-login-worker build、PG/资金/发布检查及证据保持；其它部署 workflow 同样不改，不部署 |

本轮设计验证仅检查文档索引/相对链接、diff 范围、实际依赖与消费者。未来开发自检限原 retirement/import/命名/文档 guard、保留 pricing/marketplace/API/owner/fetcher/worker/观察的必要原测试及命令编译/实图；具体执行按改动风险匹配，不为设计预跑所有业务测试。稳定候选运行原适用 PR CI，一次集中独立最终检查；未变代码的有效证据明确复用，不冒充新 SHA 直接执行，SKIPPED 不记 PASS。

## 7. 实施准入、交付与停止点

1. 当前授权只有只读调查、设计/索引文档、必要文档检查、commit/push 和 Draft PR 维护。本轮 N/A 不授权生产实现。设计交付后停在 **DESIGN_ONLY / 待用户确认**；不自行将未来源码退休标 Ready 或 IMPLEMENTATION_READY。
2. 用户明确确认 §5、§6、本节后，在 #29/#301 记录 Reuse Existing Architecture / Ready、准确冻结设计、唯一 Writer/同一主要 PR、开工 main、保留 owner/不变量及消费者刷新，再修改生产源码。原 #742 的实施或合并授权不延伸到本闭包；无需重复全局架构评审，边界变化按 §2 停止/升级。
3. 使用同一主要分支/PR 连续完成未来范围内实施和必要修复，不机械拆 PR；在原退休 guard 先 RED，再最小删除/收缩和 GREEN。真实 Must finding 先分类并检查 siblings；未满足不能合并，不以删安全断言、加空壳/alias/fallback/第二 owner 或放宽门禁保绿。
4. 最终候选前同步一次实际 main，只处理真实组合问题，完整候选集中做独立最终交付检查；修复只复核相关增量，同 SHA/未变代码不重复完整评审。准确候选/CI/review/批准证据写 PR，设计证据和实现证据分列，不用文档 CI 代表源码退休已完成。
5. 未来实现完成停在 In Review，待独立验证/用户决定。合并另需准确 HEAD 授权并核对实际 merge 对应 push/main CI；不部署、不关 #29/#301、不操作运行实例/真实配置凭据/Profile/DB/Redis/队列/provider/数据（含真实 dry-run），不清理镜像/卷/备份/历史。
6. 当前产品和保留环境由原 owner 继续交付，#692/其它活跃批次不接管；内部试用不等待此设计/实现或 worker Organization 化。运行/用户/平台验收 NOT_RUN、生产 NOT_READY；保留 worker 的数字持久化等剩余债务明确存在，两个父 Issue 保持 OPEN。

```text
Legacy decision: RETIRE
Reusable behavior: PriceManager/平台 transport、SHEIN 纯公式/成本利润、TEMU marketplace 公式与原 calculator/PriceCalculator、两 SHEIN 属性和当前共享能力保持原样。
Current owner: 各平台 API/client、marketplace/publishing/Listing 的既有规则 owner，原 listingadmin/Store/Mapping/规则仓库、当前 observations 与独立登录 owner；历史/恢复仍归原运行 owner。
Cutover/deletion condition: 用户确认 §5/§6/§7、执行 Issue Ready、精确消费者归零且保留能力/有效检查满足后，仅退出确认的源码；真实运行状态不在本矩阵内。
```
