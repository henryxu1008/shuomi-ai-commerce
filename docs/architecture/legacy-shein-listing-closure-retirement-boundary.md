# 旧 SHEIN 数字刊登整链：完整退休边界设计

> #824 后续按用户确认的[共享支撑边界](legacy-numeric-listing-support-closure-retirement-boundary.md)退休八类闲置 listingadmin 管理链、旧 Task/Workspace 尾部及 sherr/handled_errors.go。本文对这些准确对象的保留仅为历史；Store/敏感词/话题、四种共享值合同、现用 sherr、定价/属性/API、当前采集/图像/Store 与独立 worker 能力继续保持。原证据不改，不授权运行或数据清理。

> 后续用户已确认 [#819 TEMU剩余整链边界](legacy-temu-listing-closure-retirement-boundary.md) §5/§6/§6.1/§7。本文件对该闭包内旧 TEMU AI/SKU/spec、PriceHandler/PriceCalculator、property/context/TemplateState、pipeline/skugen及专属纯规则/管理投影/prompt的保留描述仅为历史边界，现由#819精确替代；当前平台API/规则、共享合同、SHEIN价格/两属性、采集/图像/Store观察与独立SHEIN登录/Store同步继续保留。原证据不改，不授权实例或数据操作。

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

- Status: **SOURCE IMPLEMENTED / In Review**。用户已确认冻结设计 `a7c29d4b14a277782ca7218ceca7267bcd92c113` §5/§6/§6.1/§7；独立设计复核已完成，#29/#301 在生产改动前已登记并读回 **Independent Architecture / IMPLEMENTATION_READY / Ready**。三项 EXTRACT 与当前消费者切换、§5.2 的 207 Go 文件和 3 YAML、§5.3 mixed 退休已完成；准确实现 HEAD、原 CI 与最终独立复核结果记录在 [#817](https://github.com/qq550723504/task-processor/pull/817)，不以设计 CI 代替。
- 调查基线：`main @ ea959a5c08fc43e07fa19aef28bd274275a66fba`（#815 之后）。
- Design Basis: **Independent Architecture / IMPLEMENTATION_READY**；用户确认与独立设计复核绑定上述冻结版本；开工时同步 `main @ ca4332aef6e4f98a203ca196dc3ccedce649e31c`。原设计分类/准入前置已满足，范围与停止条件保持冻结；本文不授权合并、部署、运行实例或数据操作。
- 用户决定：整批设计、确认后整批实施，不再按 handler/builder/DTO 拆 PR；保留当前能力、独立 SHEIN 登录 worker 及其 Store 同步；两套 workflow 不改，不操作实例或数据，不增加内部试用前置。

## 1. 用户结果、范围与非目标

使用者是维护当前内部试用、采集/图像、Store 与发布能力的操作者。可确认的交付是：旧数字 Task 驱动的准备→读取/过滤→类目/属性/销售规格→SPU/SKC/SKU/图像/翻译→草稿/发布/错误重提→结果/映射/计数落库及修复队列，连同最后消费者专属 helper、DTO、fixture 和元数据，在一个主要 PR 完整退出。当前用户操作仍沿现有入口和各原 owner；维护本身不阻塞试用，不新增产品入口或宣称开放真实发布。

闭包在**当前真实消费者**处截止，不在文件数或目录层级截止。§5 的207个 Go 文件（149生产/58测试，基线生产21788行）为整退清单，另有精确 mixed 声明、3个 YAML 默认模板和4个 key；三项 EXTRACT 先于删除。该规模不是拆成多个退休小批的理由，也不是允许整删共享模块的依据。

非目标：修复旧数字刊登执行器、延续其 Task/重试/修复协议、创建替代 pipeline/队列/通用上下文、恢复已退休 CLI/control-plane/processor、建新 schema/状态机/事实源/预算或授权系统、旧 ID/数据/Profile 迁移、兼容 wrapper/alias/fallback、双读双写、运行维护或数据清理。当前正式 Submission/Store/AI/worker 的协议不因源码退休获得新能力或新前置。

## 2. Product Authority、Design Basis 与不变量

沿用以下已批准边界，不把历史代码、测试或配置可达性当作新的产品需求：

- [全新系统决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md)、[Project Boundaries](project-boundaries.md)：仅 EXTRACT / RETIRE，不承接旧业务历史。
- [独立 SHEIN worker 身份/持久化](shein-worker-identity-persistence-boundary.md)：原数字账号/Redis/Profile/登录 attempt/查重/Store ID 同步保持原 owner；保留 worker 不授权当前 Organization 使用旧账号。
- [当前供应链与发布](my-supply-chain-shein-v1.md)、[SHEIN 导入](completion-shein-import-v1.md)、当前 `internal/publishing/shein`、`internal/marketplace/shein/publishing`、`internal/listing/submission`：Product/批准资产/target/发送许可/receipt/UNKNOWN 属各原 owner。本设计不把尚未开放的官方执行或旧 seller 网页调用写成生产可用。
- [Store current application](store-center-current-application-v1.md)、[Store observations](store-center-platform-observations-v1.md)、[内部观察接线](store-observations-preparation-runtime.md)：原 Organization/Store 访问权、观察记录与同步保持。
- #742/#743/#744 对两种共享属性、有效平台 API/规则、PriceManager/PriceCalculator 的保留，以及 #749–#815 对 TEMU AI/SKU/spec/PriceHandler 和当前图像/采集/Store 的保留仍适用。只窄替代本设计 §5 明列旧 SHEIN 执行对象的历史保留描述。

未来实施虽为大闭包，**不新增或修改持久事实 owner、schema、身份/租户/权限、状态机/事务、幂等键/retry/UNKNOWN/recovery、AI准入/预算/计量、平台发送 authority**。Independent Architecture 覆盖的是准确删除和当前窄合同脱钩，不是引入新的跨系统协调机制。Figma/UI/IA N/A。

Must：§5 保留/EXTRACT 的当前行为和所有既有保留能力；原 DTO 字段/JSON/数据归属、当前品牌保留与敏感词清理、站点默认值与属性匹配结果、原平台API/规则、采集/Store观察及登录worker能力。两套及其它 workflow、依赖/保护/必需安全检查/ceilings 不改；无新增 Accepted Risk。

Threat Model 仅针对误删真实共享消费者、把旧 source 删减变成数据/外部动作、丢失品牌/语言错误消息合同、恢复旧 Task/发送/修复 owner、误删 worker 登录/Store 同步，和新旧同名 DTO 混淆。不给本维护追加全局安全平台、验收工具或理想化 Threat Model。

## 3. 实际消费者与架构问题

已核对全仓 Go import、构造器、方法/别名与测试；`go list ./cmd/...` 为31个命令，`go list -deps ./cmd/...` 仅作静态图，未运行 main/constructor。14包核心候选未进入该并集；扩展后的 mixed 部分有以下真实共享边，不能据“包可达”宣称旧 handler 已装配，也不能据核心包无装配删共享行为。静态调查不证明实例、外部调用方或旧数据不存在。

| contract → implementation → injection → consumer | 实际状态 | 决定 |
| --- | --- | --- |
| 旧 Task/StepHandler → Store/配额/规则/读取 handler → 构建 handlers | 原 CLI、scheduler、control-plane、processor 已退休；本轮构造器无当前命令注入。Store handler 可改数字 tenant/店铺状态、额度和暂停，不是新 Store owner | 整链 RETIRE，不把它们移入 current-application/登录 worker |
| product/build → attribute/sale → skc/sku/variant/image/size | 除下述窄属性 matcher 外，当前发布不调用这些 builder、模型选择/修复、图像处理或 SKU 生成实现 | 整个 product 子树 RETIRE；当前图像、平台图片 API、规格与规则沿原 owner 保留 |
| publish → checker/validator → PublishProduct/SaveDraft → 自动敏感词/数量修复重提 → saver/result/variant_success | 全包无包外生产 importer/当前装配；内部可重复平台写并另写本地映射/计数 | RETIRE 完整发送/重提/结果链，不抽取旧自动重提协议，不将其职责转交当前 Submission |
| productdata → sourceproduct Reader/Cache/Fetcher；publish → GetVariantByAsinFromVariants | 读取/缓存 handlers 无当前装配；变体 helper 的最后包外 caller 在 publish | RETIRE wrappers/helper；共享 Reader/Cache/Mapping、Amazon/1688 当前采集及错误 owner 保留 |
| mapping repair service/strategies → MappingBuilder → mapping gateway/store/product API；handler → queue/workers | 无包外生产 importer；handler 的 queue/worker/stop/retry DTO/统计属于此闭包 | 整包 RETIRE，既有行、修复结果、日志不处理，不新建 durable repair owner |
| addresscopy → managedclient → 原 Cookie/数字 Store provider → warehouse API | 两包没有当前命令或其它生产消费者；跨源/目标店铺复制具有真实平台写副作用 | 两包完整 RETIRE；`sheinloginmanaged` 是不同包，worker 的 client/API/provider 不删 |
| publishing/shein.attribute_value_matcher → attribute.NewAttributeValueMatcher.FindMatchingPlatformValue | 当前直接消费者只有此窄匹配；matcher 同包的旧 GetPlatformAttributeValues 又拉入旧 mapper helper | EXTRACT 匹配值行为到 marketplace publishing；删除整个旧属性包，不留 matcher facade |
| publishing/shein.submit_site_sku_policy → store.GetSiteListByRegion | 当前只使用地区→默认 SiteInfo 纯值规则，不用旧 Store handlers | EXTRACT 该规则到 marketplace publishing；Store 包及旧 currency helper 同批 RETIRE |
| publishing/shein sensitive adapters → content / submitprep → shein/context.TaskContext | 当前只消费敏感词/品牌/产品载荷与验证消息；不需要旧 Task、仓库、MemoryManager、APIClients、配额、stage 或结果状态 | EXTRACT 窄 CleanupContext/ValidationResult 到 content；改准确消费者，删除 context 全包及旧 aliases |
| publishing/shein/category_legacy_selector_adapter、publishing/sheinmanaged/category_selector_adapter → category.Manager/AISelector | 两个真实当前消费者仍用类目选择/核心商品提取；manager 仅 GetCategoryIDListWithTree 方法依赖旧 TaskContext | manager/测试/AI接口保留；只删除旧 Task 方法及三个 handler/API wrapper 文件 |
| productsync.EnrichedSkuInfo.AmazonMonitorData → root shein.AmazonMonitorData；activity 测试 | 两种共享属性和纯 SKU 成本规则仍使用该具体 DTO | root models.go 只保留 AmazonMonitorData 原字段/tag/类型身份；productsync 两类型与测试不改 |
| content/submitprep/category → aicache；submitprep → language/名称限制/授权品牌/话题及词仓库 | Category/Content cache、当前优化/清理/语言规则有实际消费者。languageconfig/siteconfig 老 wrapper 只有本轮旧 callers | shared cache/现用服务保留；只退4个失去最后调用方的 cache type 常量及两个专属 wrapper 包 |

架构问题：当前属性/站点/敏感词窄行为通过宽 legacy 包拉入 Task/构建/旧持久化协议。只移走顶层 handler 会留下这些依赖，逐层再开 PR 也不解决根因。§5 的三项 EXTRACT 将现用值行为直接交给现有 capability owner，然后整退专属闭包；不 wrap 旧服务或以新 caller 保活。root AmazonMonitorData 是明确的共享停止边界，不能为了根包“清零”损坏已确认合同。

## 4. 持久化与平台副作用 owner

| 边界 | 原事实/效果与失败语义 | 退休处置/保留 owner |
| --- | --- | --- |
| 原数字 Task/Store/规则输入 | listingruntime 合同、listingadmin 原数字 Store/规则/owner 仓库；旧 handler 会同步 Task tenant、禁用/暂停 Store | 只退本轮 callers，不删 shared DTO/仓库/表/行/tenant 或 owner 校验；当前 Organization/Store 不接管旧记录 |
| 原产品读取/缓存 | sourceproduct/fetcher、RawJSONData/源证据与缓存 owner；读取错误由原 fetcher 分类 | 删除旧 wrappers/字符串错误猜测，不删当前采集/原始证据、缓存端口或数据 |
| 原结果/导入映射 | listingadmin ProductImportMapping 原事务/tenant/Store/OwnerUserID；旧 publish/repair 会查询后 Create/Update、标记 published | 只删调用者；不修复、重放、回填、重绑定或删除原行。shared mapping/owner 安全检查保留 |
| 旧配额/暂停/计数 | state.MemoryManager 及 DailyCount/ShopPause 各原 provider；并非全是进程内量。TryReserve/Rollback/Increment 可触达原持久化/店铺状态 | 删除旧执行预占/调整/恢复编排和 Task 标记；state/shared API/仓库原协议及测试保持，不清 reservation/计数/暂停 |
| 平台发布/草稿/自定义属性/图片/翻译/地址复制 | 各原 API/client 的 HTTP 写、平台身份、上传与 provider 请求；旧发送、结果写入/配额之间没有共享事务 | 源码退出这些 callers，不转交旧 retry/UNKNOWN/补偿，不发任何请求；canonical API/DTO/client 及当前发送 owner 保留 |
| repair queue/cache/debug文件 | mapping channel/goroutine/retry 只在进程内；构建/发布还会写 debug JSON/响应/图片临时文件 | 删除专属源码，不停止现有 goroutine/进程，不枚举或删文件/队列/历史日志，不发 recovery dry-run |
| shared AIResultCache | `shein/aicache` 的 DB表、内存缓存/cleanup与原 key；category/content仍消费 | cache owner、构造/表/历史行、原隔离/准入语义不变，仅删4个专属常量。构造器可 AutoMigrate，设计/源码验证不得执行它连接真实库 |
| current 敏感词/品牌/Prompt | content 原 JSON持久配置与 InMemory 选择、submitprep 原 tenant-scoped sensitive/generation-topic仓库、authorizedbrand；Prompt原Store/Registry/HTTP语义 | CleanupContext不拥有持久化/授权/重试；原清理、允许品牌及消息解析保持，原文件/仓库/覆盖记录不读写清理 |
| 当前 Product/Listing/Store/Submission | canonical产品/版本、Review/Apply、已批准资产、target、发送许可、receipt/UNKNOWN各原owner | 不新增事实或状态机；不把旧结果标记当当前平台成功 |
| 独立 SHEIN 登录/Store同步 | sheinlogin/bootstrap、sheinloginmanaged、listingadmin Store，原 Redis attempt/Cookie/control、Profile/browser及tenant-qualified查重 | 代码/配置/键/状态/权限/Store同步全部保留；不停止/重启/访问实例、不执行真实登录或写回 |

响应丢失、重试、重启、取消、并发和部分结果：退休链不再作为当前恢复入口，不接续旧 Task 或另发相同平台动作。保留链继续由各原 owner 负责幂等身份、事务和 UNKNOWN；本设计不证明既有实例已排空或旧副作用已清算。发现具体当前外部契约/运行资产要求兼容时停对应项报告用户；没有兼容 Exception。源码切除不授权任何真实处置。

## 5. 删除、保留与抽取矩阵（待用户确认）

### 5.1 三项先抽取行为

| EXTRACT | 正确 owner / 准确切换 | 必须保持 / 不抽取 |
| --- | --- | --- |
| E1 属性值匹配 | `internal/marketplace/shein/publishing/attribute_value_match_policy.go`：`MatchPlatformAttributeValue(value, sanitizedValue string, candidates map[string]int) int`；current publishing 的原 matcher 调用直接改用此值函数，sanitizedValue 由原 `content.SanitizeForSheinAttribute(value)` 产生 | 保持精确→trim/lower key→原清理值精确/大小写→原 Fields/lower 规范化匹配的顺序、空值/空集合返回0、候选ID选择。不改“颜色不归一化”规则或新增模型。原 matcher 的 GetPlatformAttributeValues/日志采样/mapper/SKU对象不搬迁；不复制 StringSanitizer，复用现有纯清理函数 |
| E2 地区默认站点 | `internal/marketplace/shein/publishing/site_policy.go`：`DefaultSiteListByRegion(region string) []sheinproduct.SiteInfo`；current submitDefaultSites 原调用直接切换 | 原12个地区 US/FR/DE/IT/ES/UK/AU/JP/MX/SA/AE/CA 与未知/空回US、MainSite/SubSite、每次新slice保持；caller原Trim/Upper保持。GetCurrencyByRegion、动态API handler、Store缓存/身份/helper不抽取；原平台Site API保留 |
| E3 敏感词窄输入/消息 | 在现有 `internal/shein/content` 定义 `CleanupContext` 与 `ValidationResult/SKCValidationMessages`；改 content 的 processor/word/validator、submitprep/sensitive_words 和 current publishing 的两个 sensitive adapter/fixture | CleanupContext仅普通Context、AuthorizedBrand原resolved引用、SourceBrand string与ProductData原平台载荷引用；SourceBrand保存原AmazonProduct.Brand在本次调用中的值，不持久化/生成第二来源事实。保留nil/空、允许品牌、移除来源品牌、原case/punctuation及消息提取/清理行为，不添加Task、tenant映射、Store、仓库、MemoryManager、APIClients、stage、quota、retry状态或constructor/worker |

E3 的 ValidationResult 保留原 Form/FormName/Messages/Module/OtherLanguageMessageMap/SkcErrorMessageMap，嵌套保留 Messages/OtherLanguageMessageMap 和**原 JSON tags**（包括原大小写差别）。不把平台验证消息简化成丢失语言/SKC语义的字符串。旧 StepHandler/TaskContext/RuntimeRepository、类型 aliases、旧构造器、属性生成模型和状态全部 RETIRE，不留旧包转发。current `SanitizeDraftPayloadSensitiveContent` 名称/流程保持，其窄输入参数改为 content.CleanupContext；全仓无当前外部 Go 消费者需兼容证据，内部准确调用点一起切换。

### 5.2 整文件删除清单

下表每个名字都为基线实际 Go 文件，生产与测试合计207；生产149/21788行、测试58。列出整树是为了核对完整闭包，实施时精确核对而非操作任何运行目录。提案中的“14包/142文件”是初步核心，已扩展至下列完整清单，不将扩展部分留待后续小批。

| 目录 | 整退文件（含旧测试） |
| --- | --- |
| `internal/shein/addresscopy/` | `helpers.go`、`helpers_test.go`、`models.go`、`service.go` |
| `internal/shein/` | `attribute.go`、`error_exports.go`、`product.go`、`sensitive_words_filter.go`、`task_reason.go`、`types.go` |
| `internal/shein/category/` | `ai_selector.go`、`category_repo.go`、`tree.go` |
| `internal/shein/content/` | `words_processor.go` |
| `internal/shein/context/` | `attribute.go`、`context.go`、`context_test.go`、`sale_attribute_selection_state.go` |
| `internal/shein/languageconfig/` | `languages.go`、`languages_test.go` |
| `internal/shein/managedclient/` | `api_client_test.go`、`bridge.go`、`manager.go`、`runtime_adapters.go` |
| `internal/shein/mapping/` | `builder.go`、`builder_owner_test.go`、`handler.go`、`service.go`、`strategies.go`、`strategies_test.go`、`types.go` |
| `internal/shein/product/attribute/` | `apparel_size_resolver.go`、`custom_processor.go`、`custom_processor_test.go`、`fill.go`、`importance.go`、`importance_test.go`、`input.go`、`mapper.go`、`mapper_runtime_input.go`、`mapper_test.go`、`matcher.go`、`matcher_test.go`、`platform_value_domain.go`、`platform_value_fallback.go`、`platform_value_fallback_llm.go`、`platform_value_fallback_llm_test.go`、`platform_value_resolver.go`、`prompt.go`、`selector.go`、`size_chart_resolver.go`、`size_rules.go`、`size_system.go`、`structured_size_resolver.go`、`task_context_sale_attribute_selection_test.go`、`template.go`、`types.go`、`utils.go`、`validate_repair.go`、`validate_repair_test.go`、`validator.go` |
| `internal/shein/product/attribute/sale/` | `batch_processor.go`、`batch_processor_test.go`、`cache_consistency.go`、`cache_consistency_test.go`、`comparison.go`、`context.go`、`context_test.go`、`debug.go`、`file_saver.go`、`filter.go`、`gpt.go`、`gpt_test.go`、`handler.go`、`json_parser.go`、`json_parser_test.go`、`metadata.go`、`metadata_test.go`、`preparation.go`、`product_data.go`、`prompt_generator.go`、`request.go`、`request_test.go`、`sale_attribute_input.go`、`sale_attribute_output.go`、`single_processor.go`、`single_processor_test.go`、`smart_filter.go`、`smart_filter_input.go`、`smart_filter_test.go`、`utils.go`、`utils_test.go`、`validation.go`、`validation_test.go`、`value_filter.go`、`value_filter_test.go` |
| `internal/shein/product/` | `attribute_repo.go`、`init_data.go`、`init_data_test.go`、`inventory_repo.go`、`product_manager.go`、`product_repo.go`、`result_merger.go`、`shelf_quota.go`、`shelf_quota_test.go`、`spu_limit.go`、`spu_record.go`、`spu_record_test.go` |
| `internal/shein/product/build/` | `attribute_builder.go`、`attribute_builder_test.go`、`attribute_classifier.go`、`build_attribute.go`、`build_input.go`、`skc_list.go`、`spu.go` |
| `internal/shein/product/image/` | `processor.go`、`validation.go` |
| `internal/shein/product/size/` | `chart.go`、`chart_test.go`、`normalizer.go`、`normalizer_test.go` |
| `internal/shein/product/skc/` | `attribute_strategy.go`、`builder.go`、`image.go`、`selection_strategy.go`、`selection_strategy_test.go`、`skc_build_input.go`、`skc_build_output.go`、`translation.go`、`translation_test.go`、`validation.go`、`validation_test.go`、`variant.go`、`variant_runtime_test.go` |
| `internal/shein/product/sku/` | `builder.go`、`creator.go`、`generator.go`、`image_fixer.go`、`image_processor.go`、`sku_runtime_input.go`、`strategy.go`、`strategy_test.go`、`utils.go`、`utils_test.go` |
| `internal/shein/product/variant/` | `composite_matcher.go`、`exact_matcher.go`、`exact_matcher_test.go`、`fuzzy_matcher.go`、`match_helpers.go`、`matcher.go`、`matcher_helpers.go` |
| `internal/shein/productdata/` | `fetch_and_cache.go`、`fetch_and_cache_variants.go`、`raw_json.go`、`raw_json_test.go`、`variant_data.go`、`variant_limit.go`、`variant_limit_test.go` |
| `internal/shein/publish/` | `checker.go`、`error_handler.go`、`exists_check.go`、`handler.go`、`handler_test.go`、`mapping_helper.go`、`publish_input.go`、`publish_input_test.go`、`repository_helpers.go`、`repository_helpers_test.go`、`result.go`、`result_test.go`、`sale_attribute_policy.go`、`saver.go`、`validator.go`、`validator_attributes.go`、`validator_duplicate_test.go`、`validator_test.go`、`variant_success.go` |
| `internal/shein/siteconfig/` | `sites.go`、`sites_test.go` |
| `internal/shein/store/` | `region.go`、`region_test.go`、`site.go`、`site_test.go`、`store_id.go`、`store_id_test.go`、`store_info.go`、`store_info_test.go`、`supplier.go`、`warehouse.go`、`warehouse_repo.go` |
| `internal/shein/translate/` | `translate.go`、`translate_repo.go`、`translate_test.go` |
| `internal/shein/validation/` | `build_attribute.go`、`build_attribute_test.go`、`daily_limit.go`、`daily_limit_test.go`、`filter_rule.go`、`filter_rule_test.go`、`get_rule.go`、`get_rule_test.go`、`quantity.go`、`quantity_test.go`、`reapply_filter.go`、`rule_checker.go` |
| `internal/sheinbridge/saleattribute/` | `handler.go`、`handler_test.go` |

另整退默认模板：`prompts/shein/attribute_selector.yaml`、`attribute_value_fallback.yaml`、`translation.yaml`。它们分别只供本轮旧选择/值兜底/批量翻译；当前 category_selector/content_optimizer/display_attribute/sale_attribute 和所有 TEMU/image prompts 保持。

### 5.3 mixed 精确删除与保持

| 位置 | RETIRE / 切换 | 保留 |
| --- | --- | --- |
| shein/models.go | 只退 ImageUploadResult、AttributeImportance、EnrichedSkuInfo、EnrichedSkcInfo、InventoryInfo、SkuInfo、MappingInfo、CostPriceInfo 等除右栏外旧声明；清失去使用的imports | **AmazonMonitorData原完整声明**：ASIN/Price/Currency/InStock/Stock/LastCheckTime及原tag、类型身份；productsync.EnrichedSkuInfo继续直接引用它 |
| shein/category/manager.go | 只退 GetCategoryIDListWithTree 及该方法专属 shein 根包import | 原 AISelector/Manager/CoreItemInput、Suggest/Tree/children过滤/缓存和所有其余方法/测试；两个当前selector adapter保持 |
| shein/aicache/cache.go | 只退 TypeAttribute、TypeSaleAttr、TypeSKCTranslate、TypeAttrValueFallback 四常量 | TypeCategory/TypeContent、模型/表/key/Get/Set/DB/内存/cleanup原实现；不删已有cache行或改变泛型读写 |
| prompt/keys.go | 退 KSheinAttributeSelectorSystem、KSheinAttributeValueFallbackSystem/User、KSheinTranslationBatchOptimizeSystem 四常量 | 其它全部key、generic Prompt Registry/Store/renderer/loader/watcher |
| promptmgmt/catalog.go、catalog_manifest.go | 仅移除 AttributeSelectorSystem/TranslationBatchOptimizeSystem 两个 catalog键/metadata；fallback两个key基线本来不在catalog，不虚构4个catalog条目或扩大变量清理 | 共享变量字典和其它catalog/label/scope保持；既有tenant覆盖记录不处理，通用list/status/read/render行为不变，catalog/schema/新Upsert不再准入退役键 |
| content/processor.go、word.go、validator.go、processor_authorized_brand_test.go | 仅旧TaskContext/PreValidResult→§5.1窄输入/消息与fixture的必要替换；SourceBrand代替旧model.Product.Brand读取 | 原清理/品牌/SKC/语言规则及JSON配置、存储、并发/Close语义保持；其余当前content文件/测试不改规则 |
| submitprep/sensitive_words.go | TaskContext/PreValidResult创建及applyAuthorizedBrand参数改为E3；不注入旧model | 原tenant敏感词/话题仓库、InMemory构造、品牌解析、产品清理、验证敏感词保存逻辑保持；其它submitprep保持 |
| publishing/shein/attribute_value_matcher.go | 旧attribute包import/constructor改为E1原值规则，准确增加content纯清理调用 | 当前字段/模板/AI候选、MatchedBy字符串 `attribute_value_legacy_matcher`（现有投影值）和其它响应语义保持，不借重命名改外部输出 |
| publishing/shein/submit_site_sku_policy.go | 原store包import/调用改E2 | 其余SKU/stock/quantity/weight/warehouse/payload归一化保持 |
| publishing/shein/sensitive_content_sanitizer.go、submit_prep_sensitive_adapter.go | 参数/接口/ensureSanitizerTaskContext helper改为E3及相应名字，无旧Task兼容层 | runtime AuthorizedBrand传播、nil输入、所有字段处理顺序及当前调用者行为保持 |

保留完整：`shein/api/**`（category/attribute/product/image/marketing/pricing/other/translate/warehouse含DTO/tests）、`shein/client/**`、`authorizedbrand/**`、`sherr/**`、`namelimit/**`、`generationtopics/**`、`activity/**`、`pricing/**`、`productsync/product_sync_types.go` 与营销禁入guard；content除上述改动/旧words_processor外、submitprep和category当前部分。保留的是 canonical平台API/transport；本轮退的是 product/store/category/translate 下无当前消费者的重复旧runtime API wrapper，不删 API目录下的同名服务。

尤其保持两种 **productsync.EnrichedSkuInfo/EnrichedSkcInfo** 的全部字段、JSON tag和类型，不把根包同名旧DTO当它们；activity成本规则/测试及PriceManager/PriceCalculator保持。TEMU AI/SKU/spec/13个property保留文件/PriceHandler、平台 API/规则、四TemplateState字段、Reader/Cache/Mapping、当前图像/采集/Store观察以及独立登录worker/Store sync与此前所有Must保持。listingadmin/listingruntime/state共享合同/仓库、当前Product/Store/Submission不做顺手清扫。

```text
Legacy decision: EXTRACT + RETIRE
Reusable behavior: E1现用属性值匹配、E2现用地区站点默认值、E3现用敏感词输入/验证消息；其它有效规则复用现有owner。
Current owner: marketplace/shein/publishing纯值规则；shein/content/submitprep现用清理；当前Publishing/Product/Store/Submission/平台API和独立登录各原owner。
Cutover/deletion condition: 独立设计复核和用户确认→Issue IMPLEMENTATION_READY/Ready→先抽取/切当前消费者→原guard RED→完整精确删除及保留回归GREEN→准确实现CI/最终独立复核；无运行迁移/清理。
```

## 6. 验证、测试与交付

本设计阶段只跑文档适用检查、文件/引用/消费者核对与原CI；不为“先设计”写生产/测试代码或执行 constructor。独立复核检查实际矩阵与源码/当前消费者，不只看作者摘要；设计确认、实际实现CI、运行/平台/provider/用户验收分别记录。

用户确认后的同一 Delivery Batch：
1. 开工同步一次main、刷新完整矩阵和所有跨边；先以现有测试/TDD锁定E1/E2/E3确切行为，抽取并切换当前caller。不改规则以让旧fixture通过。
2. 在原 retirement guard登记准确整退路径/声明与保留合同，删除前得到目标RED；保留断言本来应满足。再同一PR一次收口完整闭包，不按目录推多个主要PR。
3. 运行受影响当前发布/marketplace规则/content/submitprep/category/API回归、原retirement/import/worker检查与 current-application/shein-login-worker 构建；只编译无测试包，不执行业务main或真实provider。
4. 稳定候选才跑完整原PR CI/一次最终独立交付检查；修复finding先分类，只复核实际增量。实现者/CI/AI不签发用户/生产验收。准确HEAD另获正常expected-head合并授权，合并后核对matching main CI。
5. 原工作区空目录/历史残留若影响absence，准确候选干净源码snapshot另验证，如实分列；不清运行资产或改弱原检查。

### 6.1 旧测试、原守卫和文档的精确处置

| 原位置 / 行为 | 实施处置 | 不可改变 |
| --- | --- | --- |
| 整退58个旧测试文件 | fixture对旧Task/Handler/DTO/队列/缓存批次/自动重提/计数/旧Store写回的绑定全部退休；可复用纯值断言按下行当前owner收口，不将旧模型重建为测试替身 | 不以旧fixture作为新domain authority；不能因文件整退删当前Must覆盖 |
| matcher_test的FindMatchingPlatformValue两项、region_test的GetSiteListByRegion | 原8种匹配/空集合与14站点cases转E1/E2当前policy测试；增加同值规则清理路径/每次新slice必要断言，先锁原结果再切换 | 原GetPlatformAttributeValues无当前caller、GetCurrencyByRegion仅旧SKUcaller，相关旧API形状断言退休；不复活mapper或currency facade |
| current content的两项授权品牌fixture、current publishing authorized_brand/submit_prep测试 | 保留断言，只将Task/RuntimeState/ProductState fixture换E3；补nil、来源品牌、授权品牌、多语言/SKC消息字段和JSON round-trip/清理在当前owner的必要用例 | 不改原敏感词词表/品牌允许/tenant overlay/读写与provider准入，测试用原InMemory/隔离资源，不触真实JSON配置 |
| 旧publish validator、SKU quantity/weight/image/size/selection的有效规则 | 当前官方goods的builder/publish_result持有supplier SKU唯一性/结果绑定；marketplace publishing的submit_prevalidation/payload validation、weight、warehouse、supplier/source_dimension/size/image/readiness policy及current publishing持有其余现用规则。复用这些owner/既有用例，不新抽旧validator或SKU builder | 原duplicate supplier SKU、解析后属性ID、primary-only/grouped SKU、数量/包装/stock、单位/边界、size/image约束必须仍由现有当前owner验证；不虚称旧submit_prevalidation已检查重复SKU。若发现实际Must缺口先分类，不能包回旧服务或自行扩产品/边界 |
| 旧mapping/Task输入的tenant/Store/OwnerUserID断言 | 退休旧MappingBuilder fixture；保留listingadmin owner/Store/映射仓库、state与worker既有隔离/事务/并发检查，当前owner的拒绝空scope不降低 | 不用删除旧builder为由放宽任何共享owner校验或构造新的tenant bridge |
| 原 tests/listingkit_backend_retirement_test.go | absence加§5.2全部精确路径和3YAML；整退专属包加原CurrentCommands的依赖禁入；原retained token map只移除本批退出文件项，CurrentCommands保留文件表只移除本批product/product_repo.go；新增E1/E2/E3/品牌/共享AmazonMonitorData、两属性/worker/API原合同断言 | 已退休路径、原安全/SKU/数量/平台断言保持；root只ban退出声明，AmazonMonitorData允许。category与content继续mixed，不禁整个shein。包级AST复用原扫描机制，局部匹配context/handler/alias/4cache常量，不按泛名全局禁 |
| tests/import_boundaries_test.go 中SheinRetiredManagement/OpenAI原allowlist | 仅移除§5整退文件的准确行；OpenAI保留category/manager、content/optimizer、submitprep/localized_content等现用行；models.go仍允许其共享合同 | 原loader/读错失败、platform/shared/provider禁入和root coverage不变；以前批次已不存在的无关旧行不顺手扫 |
| 同文件PublishingSheinNonAPISheinImports守卫 | 对context/product/store/managedclient等已退包保持或加入empty allowlist；原managedclient→api_factory旧豁免精确移除但禁止项保留；content只允许本批E1/E3的attribute_value_matcher、sensitive_content_sanitizer、submit_prep_sensitive_adapter准确文件 | category两个当前adapter和platform client/API准入保持，不改为空禁入或泛化whole-directory许可；其它所有禁令不放宽 |
| prompt keys/YAML/catalog | 原guard校验4key/3YAML缺席、两catalog条目退出及其余current keys保持；适用Prompt catalog/service测试沿原框架 | Store/Registry/HTTP已有覆盖记录语义不变，不对已有行执行删除、禁用、重写/迁移或读真实覆盖 |
| .golangci.yml、phase2-runtime-inventory/dated counts/ceilings | 用户于2026-10-19明确准许的唯一校准：仅删除Legacy consumer register中本批退出的23条当前消费者边（logger17、OpenAI6）；其余当前行、全部历史记录/计数、.golangci.yml与ceilings不改，原禁入退休路径继续有效 | 原必需守卫不放宽；不降阈值、不扩大豁免/取消安全检查，不改workflow；[授权与处置](https://github.com/qq550723504/task-processor/pull/817#issuecomment-6258036066) |
| 必要文档 | 本文/索引、Legacy Register、code-health中本批直接相关owner行、历史设计中与§5直接冲突的保留表述作窄superseded注记，原证据/数据保持 | 不批量改写旧Issue/PR、不将“设计”写成“源码已退休” |
| 两套及其它workflow、依赖/保护 | **零变更** | 原mandatory/security/PG/race/release/command步骤、触发/权限/并发和保护规则保持，不追加验收平台/runner/试用前置 |

依赖禁入使用完整包名或带`/`边界的子包前缀：`shein/product`不匹配`shein/productsync`，`shein/translate`不匹配`shein/api/translate`，`shein/managedclient`不匹配`sheinloginmanaged`。原根/共享API包不能因相近字符串被误禁。

## 7. 确认、准入与停止条件

1. 用户已确认冻结设计§5、§6、§6.1和§7，适用独立设计复核已完成；正式生产改动前已在同一PR登记/读回 Independent Architecture / IMPLEMENTATION_READY / Ready。用户后续仅准许§6.1所列23条当前消费者行校准，不扩大生产边界或运行/数据权限；准入历史与准确验证证据记录在#817。
2. 确认后范围内EXTRACT、切换、完整删除、finding分类/修复、必要自检、commit/push和同一PR维护可连续完成；同一Writer、一个主要PR，不机械拆批。只有真实新当前consumer/兼容契约、必须改变事实/身份/租户/权限/状态/事务/恢复/平台副作用边界或新的BLOCKER才停对应项反馈。
3. shared边的保留是当前合同证据，不是新增compatibility类别；新代码不依赖退休owner。不新建包装/别名/双事实源/新修复调度器维持旧链，不因维护/理论hardening扩Threat Model或要求先建设工具。
4. 独立复核按AGENTS先分类Finding、绑定当前Must与阻塞层级，正常架构复核最多两轮；非Blocker不无限设计版本。未满足本批Must仍阻止合并。
5. 实施完成进入In Review，交准确HEAD原CI和最终独立检查；合并另需准确HEAD授权，部署/配置凭据/真实API与数据/已有Prompt覆盖记录/关单均未授权。#29/#301保持OPEN，运行/平台/provider/用户验收NOT_RUN，不增加内部试用前置。
