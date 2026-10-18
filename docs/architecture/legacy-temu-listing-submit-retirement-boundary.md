# 旧 TEMU 刊登保存、提交与结果落库链退休边界

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

后续用户确认的 [#810 文案边界](legacy-temu-content-handler-retirement-boundary.md) 已退休本文当批保留的旧名称/描述/品牌编排、三个失去最后消费者的 publishing policy/result/test 与孤立格式 helper。下文消费者与矩阵保留为历史；仍有消费者的 NormalizeProductSubmissionName/left regexp、PriceHandler、AI/prompt 和其它有效规则/API/合同保持，不据旧保留措辞恢复已退休对象。

- Status: **USER_CONFIRMED / SOURCE_RETIRED**。用户已确认 #749 冻结设计 `e0003f1969d28b4a34f52af50d0f70843b791cf4` 的 §5、§6、§6.1、§7 并授权同一 PR 实施；最终交付检查/CI 证据在 PR，未授权合并或真实操作。
- 设计调查基线：`main @ 5063d3245069f5cdcf5d77a0876c731aab4919eb`；实施开工同步 `main @ 88a462c4e1695c1b241e615905bc9016bccc698e`。原 #747/设计 CI 仅为历史，不作为实现候选验证；最终 HEAD、CI 和复核证据维护于 PR。
- 当前 Design Basis: **Reuse Existing Architecture；Admission: Ready**。复用本节批准的 owner/合同，生产修改前已记录于 #29/#301。此前 N/A 仅覆盖文档调查，不作为本次实施准入；本次不新增公共合同、事实/权限/tenant/事务/状态机或恢复边界。
- 保留平台 API 及同名 DTO、有效规则、共享合同、两种 SHEIN 属性、PriceManager/PriceCalculator、当前采集、Store 观察同步、独立 SHEIN 登录 worker 及其 Store 同步。两套 workflow 不改，不操作实例或数据，不增加试用前置。

## 1. 用户结果、范围与非目标

使用者是维护当前内部试用、平台能力与独立登录的维护者。结果是明确无当前装配的旧数字 Task 刊登写入闭包、事实与副作用归属、逐文件删除/规则保留边界，避免重新接入旧保存/提交及“成功后补写”协议。本项不解除或增加内部试用阻塞，不改变当前用户入口和操作步骤。

范围是 `internal/temu/product` 中 Commit 创建/详情、草稿保存、提交与草稿回退、成功结果 Mapping、旧每日额度预占/计数/暂停调用、专属 DTO/日志与调试文件 helper，以及 `internal/temu/context/temu_context.go` 的专属发布成员。拟完整退休 **14 个生产文件（基线共 2,574 行）及 1 个专属测试**，并精确删除混合 context 中 8 个专属字段与 8 个方法；有效规格/包装校验先 EXTRACT。**整个 product/context 包不退休**：还有 SKU builder/filter/价格等源码消费者，名称/品牌/描述、SPU、价格和其它保留能力不能按目录整删。

不开发新的刊登、重试、对账、恢复或验收系统，不新增 UI/菜单、运行入口、公共领域协议或 provider 调用。无新开源依赖；复用现有 Go、平台 API、规则 owner 与 guard。#748 模型诊断、#746 自有商品、#692 Store 安装为其它 owner，不接管其工作。

不运行命令 main、构造器、平台 Save/Submit/CreateCommit/Query、Preview/DryRun，不读取真实配置/凭据、DB/Redis/browser/Profile、历史日志/JSON，不启停/部署实例，不迁移、清理、补写、撤回或重放。删除调试 writer 源码不授权删除其曾产生的文件。

## 2. Design Basis、Authority 与不变量

依据：

- [全新系统基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)：仅 EXTRACT / RETIRE，不新增旧 ID 映射、兼容壳、wrapper、fallback、双读双写或第二事实源。
- [CLI](legacy-listing-cli-retirement-boundary.md)、[control-plane](legacy-listing-control-plane-retirement-boundary.md)、[数字执行链](legacy-numeric-task-execution-retirement-boundary.md)、[任务持久化](legacy-numeric-task-persistence-retirement-boundary.md)、[商品/库存同步](legacy-numeric-product-inventory-sync-retirement-boundary.md)、[自动核价](legacy-auto-pricing-retirement-boundary.md)、[活动](legacy-shein-activity-retirement-boundary.md)、[批量重新上架](legacy-temu-bulk-relist-retirement-boundary.md)：旧执行入口已经退出。本设计确认后仅替代这些批次对 §5 本闭包保留待后续决定的部分，不重写历史证据。
- [TEMU publishing owner](../../internal/marketplace/temu/publishing/README.md)：平台规则与 payload shaping 归既有 `internal/marketplace/temu/publishing`，旧 handler 不能作为新系统的规则 owner。既有名称、格式、描述、品牌、数值与价格政策完整保留。
- [独立 worker 身份/持久化](shein-worker-identity-persistence-boundary.md)、[当前 Store](store-center-current-application-v1.md)、[observations](store-center-platform-observations-v1.md)、[准备接线](store-observations-preparation-runtime.md)、[当前 SHEIN 业务](my-supply-chain-shein-v1.md)：登录/观察/Product/Submission 与当前 retry/UNKNOWN 由原 owner 负责。Figma N/A，无 UI/IA 变化。

本次只复用已批准的 Hard-Cut、平台 API/共享 DTO 和现有规则职责：两个确定性校验抽离到既有 marketplace owner 的**包内私有值函数**，不导出新类型/接口，不新建领域合同、事实/schema、身份/权限/tenant、事务/状态机、幂等或 retry/UNKNOWN/recovery/mutation owner。原数字 Task 的 ID、上下文、回填、日志、默认请求装配与执行错误策略不转成当前产品合同。

14 个旧生产文件删除已有 2,574 行，再加混合 context 的专属成员退出和规则抽离，命中生产变更 2,500 行风险线；现有责任边界覆盖这次有界退休/规则归位，不按行数拆 PR 或搬包规避准入。若实际必须新增公共领域合同、事实/权限/恢复边界，停止对应项并补适用 Independent Architecture，不以本设计 N/A 或未来 Reuse 越过准入。

Must：§5 保留清单和有效校验不减；原 owner/权限/tenant/凭据/安全检查不变；两套 workflow、保护与 ceiling 零改动；不接管历史资产、在途操作或 UNKNOWN。Should：guard 与当前源码一致。Out of Scope 如 §1；无新增 Accepted Risk。Threat Model 限误删有效行为、错误转交事实/副作用和把源码退出当运行态清零，不扩为新试用前置。

## 3. 实际消费者与协议边界

基线上刷新全仓 import/构造器/专属类型，并执行 `go list ./cmd/...`、`go list -deps -f '{{.ImportPath}}' ./cmd/...`：**31 个当前命令均不依赖旧 temu/product、temu/context 或 temu/store**。未执行任何命令 main。这是静态消费证据，不是实例/数据审计。

| 合同/实现 | 实际注入与消费者 | 处置依据 |
| --- | --- | --- |
| ProductSaveHandler / ProductSubmitHandler | NewProductSaveHandler 仅由旧 Submit constructor 调用；NewProductSubmitHandler 无生产装配 | 旧平台写入入口退出，不替换为当前 Submission adapter |
| CommitCreateHandler / CommitDetailHandler | 两 constructor 只有定义，没有生产装配 | 旧 Commit 身份装配、详情回填与 TaskContext 流程退出，平台 API/DTO 保留 |
| SavePublishResultHandler | constructor 除定义仅在本包旧日志测试出现；注入 MappingAPI 与 state.MemoryManager | 退出 success 后 Mapping/计数/暂停编排，不转交恢复 |
| ProductRequestInput / 三 Output / SavePublishResultInput | 消费均在本闭包及其专属测试；输入依赖旧 Task、Store、Amazon snapshot、规则和预占字段 | 专属 DTO 与转换退出，不复制到新公共类型或私有 Task 壳 |
| Utils / ErrorAnalyzer / Fixer / Validator | 只由旧 Submit 链构造；前两者包含文件/错误策略，Fixer 修改商品，Validator 含两项校验 | 专属调试/错误/自动修改退出；有效校验按 §5 EXTRACT 到既有规则 owner |
| 额度预占与发布成员 | TEMU NewCheckDailyLimitHandler 仅定义；DailyQuota 三字段只在它和结果 DTO 之间读写；SaveResult/SavedToDraft/CommitDetail/SubmitResponse/ProductData 及对应方法只在本闭包使用 | 退出该 handler 和 8 专属字段/8 方法；不因位于混合 context 而保留孤儿发布壳 |
| 其余 product / context / state | temu/sku 的 builder、item_builder 与 filter/rule_handler 仍 import product；图像/属性/SKU/规则源码消费 context，price_query_handler 仍写 PriceQueryResponse；state 还有其它调用方 | 不退休整包，保留 TemuTaskContext/PublishState 与 PriceQueryResponse、MemoryManager、其它 context 成员及共享合同；同名 SHEIN/Amazon 字段不动 |

发现的静态问题需要在设计中区分：

1. Product API 的 Submit 在 `Success=false` 时返回 error；旧 handler 在 error 分支已经返回，其后 `!response.Success` 的错误分析/草稿回退分支按当前 API 合同不可达。不能宣称这条 fallback 已被运行验证，也不能为复活它改变 API 错误合同。源码仍含“草稿保存成功→nil 完成”的旧语义，一并退休。
2. 平台 Submit/CreateCommit/Save、SQL Mapping、每日计数与暂停是独立步骤；Mapping 写错只记录继续，handler 最终可能返回 nil。无共享事务、durable receipt 或唯一恢复 owner，不能由日志/内存完成标记推出平台与本地一致。
3. 规格校验失败后 AutoFixSpecConfiguration 没有再校验，包装 Fixer 会删除 Multiple Sets 属性并把所有 SKU 改成单件。这是旧自动修改策略，未获得当前产品 Authority；只保留检测约束，不移植自动修改或建立新修复入口。
4. 提交/模板调试 writer 用旧 Task/商品编号写 `logs/*.json`，ErrorAnalyzer 还包含按时间命名的模板快照。它们不是安全日志、事实或恢复协议，源代码退出不允许扫描/删除历史文件。

以上未断言真实事故，不作为内部试用的新阻塞，也不通过修补旧服务继续支持其运行。

## 4. 持久化、平台副作用与失败 owner

| 边界 | 基线 owner / 调用能力 | 删除与保留条件 |
| --- | --- | --- |
| TEMU Commit / 草稿 / 刊登 | api/product 的 CreateCommit、Save、Submit 分别发送 create_new、save、submit；query.QueryCommitDetail 读 query_commit_detail。平台拥有 Commit/商品及结果；API/client 拥有 transport/鉴权/错误合同 | 只退 caller；API 全树、请求/响应/同名 DTO、既有 re-export/别名/工厂与 endpoint 原样保留。不得发送、撤回、补偿或重放 |
| listing_product_import_mapping | listingadmin GORM API→CreateProductImportMappingForStore→原 repository；在同一 DB 事务内从 tenant-qualified Store 派生 owner，插入原表并提交 | 只删除旧 caller、旧 Task ID/ASIN/规则元数据转换；原表/schema/字段（含 ImportTaskID）、事务、owner/tenant 查询/写入校验与测试不动，不导入当前 Product/Submission |
| Store / Cookie / 暂停接口 | 原 listingadmin Store、TEMU client/auth；GORM Cookie 返回未配置，pause 返回 false/nil，不能从接口推定当前运行启用 | Store/凭据/worker 查重及同步不变，不新增 provider，不探测或操作真实状态 |
| 每日额度与计数 | 旧 CheckDailyLimitHandler 调用 state.DailyCountManager.TryReserveQuota 并将预占写入 TaskContext；结果 handler 读取标志。manager 通过 DailyListingCountAPI provider 读/写并持本地 scope 锁，未配置 client 的 IncrementCount 仅返回增量。不能据“MemoryManager”推定 Redis 或某张新表 | 退出本链预占/结果两个 caller、专属标志与 DTO；state、接口、其它调用方/既有单位与预占/计数语义原样保留，不清计数、不释放任何已有预占、不接管当前 Resource Ledger |
| 暂停/恢复 | ShopPauseManager 持有内存 pause map、cleanup，并可调用配置的 StoreClient.SetStorePauseStatus | 退出预占拒绝/结果侧 PauseShopUntilEndOfDay caller；原 manager/StoreAPI/cleanup/keys/格式/恢复与其它消费者保留，不清暂停、不恢复实例 |
| TaskContext / 输出 | ProductRequestInput/Output 与 TemuTaskContext.PublishState 为进程值、API 响应及旧预占标志，不是本链的 durable receipt | 专属 DTO、§5 的 8 字段/8 方法退出；保留 PublishState/PriceQueryResponse 与其余混合 context/pipeline/model 合同，不迁移到当前 Product/Store/Observation/Submission，不新建事实表 |
| 本地调试文件 | 旧 Utils/ErrorAnalyzer 在相对 logs 下写请求/模板 JSON；jsonx/fileio 是共享工具 | 退出专属 writer/字段/import；共享工具与其它 writer 保留，不读取/迁移/删除现有文件 |
| 当前产品与独立 SHEIN 登录 | 当前 Organization/Store/Observation/Submission、账号 DB、Cookie/attempt/stream/Lua、browser/Profile 与 Store sync 各归原 owner | 原格式/身份/权限/tenant/状态/事务/恢复不变，不把 TEMU Task 元数据导入独立 worker 或当前产品 |

失败、响应丢失、重试、重启、取消、并发及在途写入不建立新协议；历史平台/本地结果归原平台及既有运行 owner，源码退休不能判断已提交/未提交/可安全重试/已回滚。没有共享事务的旧组合不变成新恢复系统；真实操作仍需另外明确授权。

## 5. 用户确认的删除/抽取/保留矩阵

除另标路径外，以下路径相对 `internal/temu/product/`。14 个完整生产文件共 2,574 行；**有效两项校验先抽离，再完整退休这些原文件及 1 个专属测试，并删除混合 context 的指定成员**，不留下服务/receiver、旧 DTO、alias/wrapper 或 Task 生命周期壳。

| 对象 | 确认处置 | 明确保留 / 删除条件 |
| --- | --- | --- |
| save_handler.go（182） | RETIRE 全文件及 ProductSaveHandler/constructor/Handle、保存调用、Task 输出回填、默认 request builder、计数/JSON 委托及 fileUtils 字段 | 原 api.Product/SaveRequest/Response/Result、API Save 及 jsonx/fileio 不动；旧固定 CanSave/SupportMaxRetailPrice 等装配不作为新的当前规则合同 |
| submit_handler.go（279） | RETIRE 全文件及 ProductSubmitHandler/constructor/Handle、Submit/草稿回退、NONRETRYABLE、Task 输出与 request builder、校验/自动修改装配、未消费 mappingClient 字段 | NormalizeProductSubmissionName、全部现有 marketplace 名称/格式规则及原 API Submit 保留，不改变 API 错误合同以触发旧 fallback |
| commit_create_handler.go（185） | RETIRE 全文件：CommitCreateHandler、创建/身份回填、旧 Task StoreID 校验、TERMINATED/错误字符串分类及未被调用的 isNonRetryableError | CreateCommit API/DTO、现有 NormalizeParenthesesSpacing / CleanProductTitle 规则保留；不抽取无消费者的 OperateType 默认/旧 Commit 生命周期 |
| commit_detail_handler.go（280） | RETIRE 全文件：CommitDetailHandler、查询、ID 完整性 guard、API→旧 Product 字段回填、分类树投影及尾部旧上下文注释 | query API/CommitDetailRequest/Response/Result/CategoryTree 原 DTO 保留；旧回填只服务 Task 编排，无当前独立契约，不转成当前 Product 事实 owner |
| save_publish_result_handler.go（79） / publish_result_steps.go（123） | RETIRE 两完整文件、SavePublishResultHandler/constructor、Mapping 写入与“错误继续”、计数/暂停、响应日志 | listingadmin Mapping/Store、state 和对应真实 owner/接口/事务/安全检查全部保留 |
| publish_result_input.go（323） | RETIRE 全文件、SavePublishResultInput、build/metadata/日志/计数等方法 | 不因其部分方法无 IO 就保留数字 Task DTO。原 productpolicy.GetProductPrice、Filter/Profit/PricingRule、Store/Mapping、原 DailyLimitType 与共享预占合同不变 |
| request_models.go（37） / result_output.go（65） | RETIRE 两完整文件：ProductRequestInput、CommitDetailOutput、SaveProductOutput、SubmitProductOutput、build/apply/getSubmitResponse 接缝 | 同名或相近的平台 API DTO/re-export、TemuTaskContext/PublishState 与下行未指定的成员不删；原独立 client/state 接线不转移 |
| submit_utils.go（105） | RETIRE 全文件、ProductSubmitUtils/constructor、旧 retryable/nonretryable code/关键词策略、Task 调试 writer、计数/JSON 包装 | 保留实际 API/transport error、jsonx 与其它调用方，不把旧关键字 map 变成当前 retry/UNKNOWN Authority |
| submit_error_analyzer.go（312） | RETIRE 全文件、ProductSubmitErrorAnalyzer/constructor、模板/属性解析、文件快照与日志建议 | 专属诊断片段无当前调用方，不能因无 IO 子函数保留 analyzer 壳；不干涉 #748 等当前 provider 诊断 owner |
| submit_fixer.go（152） | RETIRE 全文件、ProductSubmitFixer/constructor、AutoFixSpecConfiguration、FixMultiplePackageConfiguration | 不移植旧 AI/模板自动填值、规格修复后免复验、降级单件策略；只保留下行明确的检测约束，不新建自动修复产品 |
| submit_validator.go（235） | **先 EXTRACT 两个检测行为**到既有 marketplace/temu/publishing 的 submission_validation_policy.go，再 RETIRE 原文件/导出类型/constructor/TaskContext/模板 lookup 与旧诊断 reader | 保留 SKU 使用的每个 parent/spec 必须已配置且 VID 非0；无缺失时通过。保留属性值包含 Multiple Sets（原大小写规则）时任一 SKU pieces<=1 的检测及首个属性位置。详见下段，不留旧 helper class |
| publish_result_input_test.go（62） | RETIRE 整个专属文件和 newPublishResultContext fixture | 其 handler 日志/DTO 缺失 Store/owner 接缝消失；实际 owner/tenant/写入安全由 §6 原 listingadmin 测试继续验证，不用测试保留旧 Task 构造 |
| check_daily_limit_handler.go（217） | RETIRE 全文件及 TEMU CheckDailyLimitHandler/constructor/Handle/HandleTemu、SPU/SKC/SKU 增量投影、TryReserveQuota/拒绝暂停与 SetDailyQuotaReservation caller | 无装配、专属预占消费者随结果链退出，不抽取旧 Task 限额协议；state manager、DailyListingCountAPI、Store 日限额字段/单位和其它消费者不动 |
| internal/temu/context/temu_context.go（部分） | RETIRE 8 字段：PublishState.SaveResult、SavedToDraft、CommitDetail、SubmitResponse、ProductData、DailyQuotaReserved、DailyQuotaDate、DailyQuotaIncrement；RETIRE 8 方法：SetPublishProductData、SetSaveResponse、SetCommitDetailResponse、SetSubmitResponse、SetSavedToDraft、SetDailyQuotaReservation、ClearDailyQuotaReservation、ApplySaveResult | **保留 PublishState 类型/嵌入与 PriceQueryResponse**、TemuProduct/ProductState、Runtime/Template/Asset/Variant/RuleState、所有其它字段/方法及 API DTO。只在 TEMU context 内精确删成员，不做跨平台同名 token 清理 |
| 其余旧 product / context / state | **原样保留** exists/cache/build/SPU/品牌/名称/描述/价格/raw JSON 等剩余文件；context 除上述指定成员外原样保留，以及 pipeline/model/handlerbase、temu/store、state 和共享 DTO | 这只是本次未退休的混合/其它消费者边界，不授权新消费者、复活入口或清理计数/暂停；同名 SHEIN/Amazon handler 与字段不在本批 |
| 用户指定能力 | 原样保留全部平台 API/DTO/re-export（含 #747 同名 BulkRelistOptions 等）、有效规则/共享合同、两 SHEIN 属性、全部定价、采集、Store observations、独立 SHEIN 登录和 Store sync | 两套 workflow/必需及安全门禁、保护/ceiling 不改，不增加试用前置 |

规则抽离限定在**现有 marketplace owner 的包内私有值函数和私有快照**，不新增导出 API/领域类型，不引入 temu/context、Task/Store/AI runtime、repo/API client、文件、时钟、provider 或新依赖。输入只含使用中的规格 parent/spec/名称、已配置规格 parent/spec/VID，以及属性值和各 SKU 包装件数；不携带 Task/Store/商品身份、tenant/owner、预占或提交状态。保留原匹配/缺失/非零 VID 与首个 Multiple Sets 位置的判定，不优化去重/合并/边界算法。原 AI→SKU 取名仅用于旧日志的上下文 reader 退休；保留显式传入的名称/ID用于检测信息，不把 AI 快照当事实源。函数不修改商品，不执行自动修复或请求。

建议使用私有名称 validateSubmissionSpecs、validateSubmissionPackaging；新文件只有必要值投影/校验，不保留 ProductSubmitValidator/TaskContext 或其 wrapper。原名称、品牌、描述、数字格式与价格规则继续沿原函数和测试，不重复实现。此 EXTRACT 没有新当前调用入口，不宣称刊登能力已开放。

Legacy decision: **RETIRE + EXTRACT 两项有效校验**。Reusable behavior / Current owner: 上述两项归既有 marketplace/temu/publishing；平台 API、共享规则/持久化/身份归原 owner。Cutover/deletion condition: 用户确认本矩阵、Issue Ready、消费者刷新仍符合、有效校验和共享安全检查先保全，再源代码缺席。发现具体当前外部契约、durable assets 或保留行为无法脱离旧 owner，停止对应项报告，不自行增加兼容。

## 6. 测试处置与验证

用户已确认下列测试处置；相应 RED/GREEN、本地必要检查与准确候选 CI/复核证据维护在 #749，不把源码缺席或 fixture PASS 当运行/用户验收：

- 在原 source-absence/保留合同 guard 先得到 **RED**（旧源仍在），再 EXTRACT/RETIRE 并 **GREEN**。两个抽离的业务校验采用 TDD：在现有 marketplace publishing 测试包新增 submission_validation_policy_test.go，以缺失行为为 RED，最小抽离后 GREEN；不新建 runner/验收平台。
- 校验样本只覆盖本次原行为：完整/缺失 parent/spec、使用中 VID=0、未使用配置不影响使用中的判定、多个 SKU/重复规格、显式名称与 ID 错误信息；无 Multiple Sets、大小写/首个位置、多件套任一 pieces=0/1 及全部>1。确认不修改传入值；不新增模板自动修复/平台请求测试或完整故障矩阵。
- 原 brand_clear_handler_test.go、name_validator_test.go 与全部 marketplace 规则/定价测试原样保留。仅退出 publish_result_input_test.go 的旧 handler/Task DTO fixture，不删除当前安全要求。
- 保留 listingadmin 的 TestGormProductImportMappingRepositoryCreateRejectsMissingOwner、TestProductImportMappingForStoreDerivesOwnerInWriteTransaction、TestProductImportMappingForStorePreservesStoreLookupErrors（含无 Store/ownerless Store）、TestProductImportMappingRequestIdentityCannotBeOverriddenByPayload，以及 import_task_owner 的 ownerless/cross-tenant Store 测试和 scoped 查询。原实际事务/错误传播继续，不用旧日志测试替代它们。
- 原 state DailyCount/ShopPause 测试、API/client TLS/错误/已有 wire、安全/tenant、两属性、营销及独立登录/Store sync 检查保留。旧 quota handler 与 8 字段/8 方法退出用原缺席 guard RED/GREEN；保留 PublishState/PriceQueryResponse、TemuProduct 与其它 context 成员断言，不复造 Task fixture。必要编译保留的 API/product/filter/SKU/context 与当前命令；仅运行相关原测试/guard，完整 tests 与 listingadmin race 由原 CI 执行，不人工重复整套 CI。
- 静态/本地 fixture/编译、PR CI、准确 main CI、运行与产品验收分别记证据。真实平台、实例/数据、部署/用户验收 NOT_RUN，不补新试用前置。

## 6.1. Guard、文档及 workflow 的精确边界

| 对象 | 确认实施处置 | 保留 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 原缺席列表加入 §5 的14生产路径及 publish_result_input_test.go；原保留清单加入新 policy 与 test、原 api/product/api.go、api/query/query.go 及其 DTO 文件；原源码扫描精确禁止 TEMU 旧 handler（含额度）/constructor/专属 DTO 及 Utils/Fixer/Validator/ErrorAnalyzer 重建，并检查 TemuTaskContext 中上述8字段/8方法缺席 | 不能要求整个 product/context 包缺席或全局禁用 Save/Submit/同名 API DTO/其它平台同名字段；原缺席/依赖/独立 worker/属性/定价/workflow race 检查完整保留 |
| 同一 guard 的 retained symbols | 检查 API Save/Submit/CreateCommit/QueryCommitDetail、原请求/响应/re-export、PublishState/PriceQueryResponse/TemuProduct 和其余混合 context 成员存在；marketplace 两私有校验存在且不依赖 Task/context/client/state/IO | 不生成新全仓扫描器/验收工具，不用宽泛 forbidden token 误伤平台 DTO 或其它 SHEIN 同名 handler |
| tests/import_boundaries_test.go | 仅从 TestTEMUProductStoreAndSchedulerRetiredManagementImportsStayBlocked 删除 submit_handler.go、publish_result_input.go、save_publish_result_handler.go 三条消失路径 allowlist | product/store root 和其它 allowlist/deny/安全规则保留；无 missing skip、无新豁免。其余 graph/ceiling不改 |
| phase2-runtime-inventory.md | **当前消费者行不删**：product 中其它文件仍 import core/logger 和 integration/openai；可加本批部分闭包退休校准说明 | 不像 #747 整包退出那样删除 product 行；所有 dated 图、phase4列表、closure ceiling 原样保留 |
| legacy-register、本文、architecture README | 用户确认后校准本闭包 source-retired，并注明两项规则原 owner/冻结依据 | 不清空 legacy 表、不宣称 #301清零；历史 external-client hotspot 不重写 |
| marketplace/temu/publishing/README | 用户确认后在原 owner 文档注明两项无 IO 校验及旧提交闭包退出 | 不扩为新刊登产品/公共领域合同，不改变其它当前规则所有权 |
| ci.yml / commercial-readiness.yml | **零改动** | 所有其它 workflow、.golangci.yml、Required CI Gate、必需/安全检查、保护/ceiling 原样保留；原 skipped 不改写 PASS |

设计阶段仅提交本文与架构索引；用户确认后按上表实施生产/test/guard 与文档变更，两套及其它 workflow、保护和 ceiling 零改动。

## 7. 确认与实施停止点

1. 设计已以 Draft PR 交付并完成文档/链接、适用原 CI 和独立设计检查；设计阶段证据归冻结设计，不作为实现候选 PASS。#29/#301 保持 OPEN。
2. **用户已确认冻结设计 §5、§6、§6.1、§7 并授权本范围实施**；不增加新领域/副作用/恢复边界，不把设计授权扩为合并、部署或真实操作。
3. 在同一主要分支/PR继续；生产修改前已在 Issue 写 Reuse Existing Architecture / Ready、冻结设计、原 owner/不变量/非目标与唯一 Writer。已刷新实际 main、31 个 cmd 的依赖、包外 constructor/DTO 和 sibling helpers，未出现新消费者；新高风险边界仍按 §2 停止并补适用准入。
4. 原 guard 和两项校验按 §6 先 RED→必要 EXTRACT/RETIRE→GREEN，保留共享规则、安全/tenant/owner 检查。finding先分类并查 siblings；真实 Must 未满足不得合并，不放宽门禁。
5. 最终候选前同步一次实际 main，只处理真实组合风险；准确实现 HEAD、CI、一次独立最终检查与限制写 PR，不重复同 SHA 全量评审/测试。完成进入 In Review，仍需准确 HEAD 的单独合并授权；正常 expected-head 合并后再核对实际 merge SHA 的 push/main CI。
6. 不授权 Issue关闭、部署/运行、真实平台/实例/数据操作，不宣称用户验收、上线或全量 legacy清零，不增加内部试用前置。所有用户保留能力继续归原 owner。

## 8. 源码交付校准

§5 的 15 个原文件已缺席，context 仅退出指定 8 字段/8 方法，PublishState/PriceQueryResponse 与其它混合成员保留。两个检测位于既有 marketplace 的私有值函数/快照，不依赖 Task/context/client/state/IO、不修改输入、不自动修复，也没有新增当前刊登调用入口。原 guard 覆盖精确缺席及保留合同；仅删三条消失路径 allowlist，product 的 logger/openai 当前消费者行、所有历史图及 ceiling 保留。

Legacy decision: **RETIRE 专属编排/DTO/helper/发布状态；EXTRACT 两项有效校验**。平台/API/Mapping/state/Store/worker 的事实、事务、权限/tenant 与副作用继续归原 owner。源码交付不代表历史资产/预占/在途状态清零；真实平台、实例/数据、部署与用户验收保持 NOT_RUN。
