# 旧 TEMU 数字刊登剩余整链退休边界

> Execution：[#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)
>
> 当前交付：设计文档与调查；Design Basis：N/A（不修改生产路径、测试或运行配置）。
>
> 后续实施 Design Basis：Independent Architecture；状态：DESIGN_PROPOSED / IMPLEMENTATION_NOT_READY。适用独立设计检查和用户确认须在生产改动前完成，准确 SHA / 评审 / 准入记录写同一 PR 与 Issue，本文不能自行授予开工权。
>
> 调查基线：main `fd31fa9513e7b49ad25049874dc006f9e218c6a3`，2026-10-19。#817 已合入 `0ea035c7855c9cb1f6fe35047e7d663b3ebf9156`，对应 push/main CI [39190495226](https://github.com/qq550723504/task-processor/actions/runs/39190495226) 成功；本批不继承其合并或23行校准授权。

## 1. 用户结果与完整 Delivery Batch

使用者是当前采集、Product/平台草稿与 Store 观察同步的内部试用操作者，以及独立 SHEIN 登录 worker 的操作者。交付结果是：失去正式入口的旧 TEMU 数字 Task AI/属性/规格/SKU/价格链，与最后消费者专属 pipeline、生成器、孤立纯规则包、管理 DTO 投影、prompt 和静态默认资源，在一个主要 PR 完整退出。当前操作继续沿原入口、事实和平台 owner；本维护不增加试用前置，也不宣称新增平台发布可用性。

闭包在当前真实合同与消费者处截止，不按目录层级、文件数或单次测试机械拆批。§5 的88个 Go 文件（79生产/9测试，基线生产10220行）整退；另有6个非 Go 源资源与 mixed 声明。三个仍有消费者的 DTO 值 helper 先在原 `listingadmin` owner 脱钩。没有新业务能力、新持久化、新平台或 AI 调用。

非目标：复活旧 CLI/control-plane/processor、修好或接入旧 AI/SKU 执行器、替换当前平台规则/SDK、给草稿自动生成价格库存或编造规格、维护旧 Task/重试/恢复状态机、建新 pipeline/调度/通用验证工具、迁移旧 ID/资料/Prompt 覆盖、操作实例、DB/Redis/Profile/凭据、部署或真实平台请求。#29/#301 仍保持 OPEN。

## 2. Product Authority、Design Basis 与不变量

适用依据：

- [Greenfield](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md)、[Project boundaries](project-boundaries.md)：只有 EXTRACT / RETIRE，不建兼容、fallback、双读双写或第二事实源。
- [当前 TEMU 三模式/官方站点合同](temu-platform-v1.md)：Product/ApprovedAsset → 明确事实与官方规则 → immutable TargetRecord → 授权 Submission/发送许可/receipt/UNKNOWN。价格、库存、单位和真实规格不能由旧默认值或 AI 猜测替代；本批不改变其阶段限制或真实执行授权。
- [当前供应链](my-supply-chain-shein-v1.md)、[Store current application](store-center-current-application-v1.md)、[Store observations](store-center-platform-observations-v1.md)、[观察接线](store-observations-preparation-runtime.md)：各当前事实/权限/观察/平台副作用 owner 保持。
- [独立 SHEIN worker](shein-worker-identity-persistence-boundary.md)与[#817 SHEIN整链边界](legacy-shein-listing-closure-retirement-boundary.md)：数字登录身份、Profile/Redis/attempt/查重与 Store ID 同步仍属原 owner；两种 SHEIN 共享属性、当前发布、图像、品牌与敏感词行为保持。
- #743 及 #749–#815、#817 中对 TEMU `PriceHandler`、TEMU `PriceCalculator`、AI/SKU/spec、13个 property 文件、四个 TemplateState 字段及旧纯规则的保留，是当时的源码消费者边界。本设计**仅提议窄替代 §5 明列旧实现的保留决定**，须用户确认；这些历史记录不是已授权删除。SHEIN PriceManager/PriceCalculator、共享 Reader/Cache/Mapping、有效平台 API/DTO/规则及其他既有保留能力不变。

未来实施采用 Independent Architecture，是因为整链跨旧 AI/Task/规则/共享合同，超过30文件与2500生产行，且需明确替代旧保留决定。它不新增或修改领域事实 owner、表/schema/数据归属、身份/租户/授权、状态/事务、幂等/retry/UNKNOWN/recovery、AI准入/预算/计量或平台发送 authority。Figma/UI/IA N/A。

Must：当前采集/图像/草稿、Store 观察、有效平台合同/规则、共享基础层、两种 SHEIN 属性、SHEIN 价格能力、独立登录 worker 及其 Store 同步保持；§5.4 三 helper 原值/空值语义保持。两套及其他 workflow、依赖/保护/必需安全检查、.golangci.yml 与 ceilings 零改。没有新增 Accepted Risk。

Threat Model 限于误删仍有消费者的合同、将静态退休变成实例/数据动作、将旧默认 tenant/随机库存/平台副作用引入当前 owner、误禁同名 API 与 worker，以及弱化现有守卫。不扩大为全局安全治理或专项验收平台。

## 3. 实际消费者与架构问题

调查检查4363个 Go 文件的实际 import 声明（含测试与不同 build tag 的源码，排除字符串路径误报），全仓构造器/类型/方法调用、非 Go 接线、prompt 和静态文件引用。31个 `cmd/*` 的 `go list -deps` 并集成功，§5.2 的14个整退包均为零依赖；只读取 Go 元数据，没有运行 main、构造器、业务测试、provider 或运行实例。静态证据不证明外部实例/旧行不存在，也不授权检查或清理它们。

| contract → implementation → injection → consumer | 当前证据 | 处置 |
| --- | --- | --- |
| TemuTaskContext → AIPropertyMapper/AIService/AIContentRewriter → handler/mapper | 原 CLI/调度/processor 已退；AI包无包外 importer、无命令注入。内含三条直接 ChatCompletion 调用路径，不是当前 GovernedModel owner | AI 全包及专属 prompt 整退，不接入当前 Agent/Tool |
| context/handlerbase → property/rules → AI/SKU/spec | 10个非 API TEMU 包只在本闭包内相互引用；原13个 property 文件和四个 TemplateState 字段的最后消费者随整链退出 | 完整退休，不把 DTO/validator/默认填充改名迁入当前 goods |
| sku/variant_processor + ai_mapping_* → property + spec | sku 无包外 importer；spec 只有 sku，property 只有 ai/sku/spec，rules 只有 property，handlerbase 只有 rules | AI/SKU/spec 与专属解析、批次、合并和默认规格整退；保留当前显式规格/商品事实 owner |
| product.PriceHandler → listingadmin.ProfitRuleAPI → GORM profit projection | product 无包外 importer、PriceHandler 无当前构造；读规则并默认倍率、随机库存；两个 GORM facade 构造器没有调用者 | 旧 PriceHandler/管理投影退休；ProfitRule/PricingRule 持久模型、仓库、查询与 owner/tenant 检查保留 |
| pricing.PriceCalculator → marketplace/temu/publishing | pricing 无包外 importer；publishing 唯一包外消费者是本 pricing；其余私有 policies 只由自身 fixture 消费 | 整退两个包，不能因此前抽取过而永久保留孤立第二套规则 |
| old Task pipeline → pipeline/handlers + TEMU | pipeline 仅由旧 TEMU 及其 handlers 引用；handlers 无 importer | `internal/pipeline` 全包闭包退休；不涉及 `internal/amazon/pipeline`、当前 Image/Agent/Temporal 或 workerpool |
| publishing SKU SN policy → pkg/skugen | skugen 唯一包外消费者是本批 publishing；随机/时间戳/hash 策略无当前注入 | 生成器与 fixture 整退，不替代官方 Submission 的稳定身份 |
| filter → data/*_temu.json + core/config.SensitiveWordsConfig | 两个 tracked JSON 仅由本批 filter 读；SensitiveWordsConfig 除本批 reader 外仅有自身 fixture/benchmark | 默认资源、准确 DTO/方法与自身 fixture 同批退；SHEIN/当前敏感词 owner 及 data/sensitive_words.json 不动 |
| gorm_pricing_rule_api → ptrFloat64/ptrString/flexibleTimeValue | 三 helper 还由保留的 mapping/filter GORM DTO 投影消费 | 先原 owner 内机械抽取，再退两旧规则投影；不改共享 DTO 的值/JSON/空值 |
| TEMU API/DTO 与 current official model/goods/rules/integration | 两组合同不依赖本批14包；当前 app/Store/供应链/Target/Submission 使用官方 goods，保留 API 是用户明确边界 | 两组全部保留，不能以旧包装 API 无命令可达为由删除 |

根因是“曾有旧内部消费者”被持续当成当前能力保留依据，留下相互引用的旧执行图和孤立纯函数。具体旧语义包括 SKUBuildRuntime 默认 TenantID/StoreID 为1、价格1.5/2.0倍率回退、库存 `rand.Intn(1000)+10`、重量/尺寸默认与截断、TEMP_ 规格修补；它们不能进入当前明确事实/精确十进制/官方版本规则路径。本批删除旧 owner，不建设新的自动填充或同步链。

## 4. 持久化与平台副作用 owner

| 边界 | 当前/旧 owner | 本批允许与不允许 |
| --- | --- | --- |
| 旧 Task/context/AsinSkuMap/AISkuMapping/pipeline 状态 | 本批 context 仅进程内对象；MemoryManager 仅引用既有 state owner，不是 schema owner | 删内存对象和接线；state/缓存/Reader/Cache/Mapping 与 model.Task 等共享合同保持，不运行或清空任何 owner |
| 数字 Store/ProfitRule/PricingRule/FilterRule/映射 | listingadmin 模型/仓库/查询、state 与保留 API | 只退无消费者的两个 GORM facade 和六个专属管理 DTO/接口；不删模型、表、迁移、行、schema init、tenant/owner/事务/并发检查 |
| 当前 Product/ApprovedAsset/CompletionDraft/TargetRecord | Product、Asset、Preparation/Target 原 owner | 事实、版本、校验、保存/CAS/访问控制全部不变，无迁移/重写 |
| 当前 TEMU 平台写/图片/查询/观察 | integration/temu、listing/submission 及原许可/receipt/UNKNOWN owner；Store observations | 不发请求、不构造旧 API client、不改变幂等/重试/多代回执/平台结果归属；官方和保留旧 API 源合同保持 |
| 本批 AI 三条调用与3默认 prompt | 原旧 handler 调用；prompt Registry/Store 管理覆盖记录 | 仅退源码调用/默认 YAML/key/catalog 元数据；无模型请求，不读写/删除/禁用/迁移任何已有覆盖行，通用管理语义不变 |
| 两份 data/*_temu.json | Git 跟踪的旧 filter 静态默认词表，无程序写入点；不属于本批新持久化 owner | 仅未来源码提交删除这两个 tracked 版本；禁止脚本/启动/清理动作去删除实例、其他 worktree、外部挂载或已有文件。此决定不授权访问真实数据 |
| 独立 SHEIN 登录/Store 同步 | shein-login-worker/sheinlogin/sheinloginmanaged/bootstrap、listingadmin/state、原 client/API | 命令、接线、Profile/Redis/attempt/tenant限定查重、Store ID 写回与登录能力保持；不启停 worker、不碰会话/数据 |

失败/重试/响应丢失/重启/取消/并发的既有事实和恢复 owner 均不变。源码删除不触发恢复或清理；无需新 outbox/Saga/迁移。若发现真实当前 consumer 或外部可观察兼容义务，先报告具体证据和受影响项，不能自行加 fallback，也不能推定获得真实环境调查授权。

## 5. 删除、保留与抽取矩阵

### 5.1 总矩阵与有效规则归属

| 对象 | 决定 | 当前 owner / 保留理由与删除条件 |
| --- | --- | --- |
| §5.2 14包86Go + §5.3 两GORM文件 | RETIRE | 全部旧执行、专属合同/fixture 或最后消费者闭包；用户确认与准入后先完成 §5.4，再整体删除 |
| §5.3 3YAML、README、2静态JSON、mixed DTO/config/prompt条目 | RETIRE | 专属旧链；只删除精确源码，既有覆盖记录/运行文件不处理 |
| §5.4 三 DTO helper | EXTRACT | 同一 listingadmin owner 中现有 filter/mapping 投影仍使用；搬值行为后删旧 GORM 文件，不保留 facade |
| internal/temu/api/** | 保留 | 全部29Go（28生产/1测试）、API/Client/Query/Product/Inventory/Pricing/Template/category、同名 DTO/re-export 与 Cookie/runtime 合同不动；此保留不是复活旧执行入口 |
| internal/marketplace/temu/**（除本批publishing）、internal/integration/temu/**、current Target/Submission/Store/Preparation | 保留 | 包含API/goods/model/rules/observations/workspace；当前 app/供应链/持久层有真实调用，沿 temu-platform-v1，不复制旧政策或混用同名 DTO |
| 属性/规格/数量/包装/SKU唯一性有效约束 | 复用当前 owner；不新增抽取 | goods.Build/CheckLocalProperties/CheckSupplierProperties/checkSupplierSpecifications/checkSupplierPackaging，rules.CheckLocalVariations/CheckLocalNumbers/CheckSupplierPriceLayout 及其原测试保持。旧七步SPU结构、TEMP_ 修补、固定包装1/1、单纯取第一属性和ASIN-only去重不成为新公共合同 |
| 当前价格/库存/重量/单位 | 复用当前 owner | 官方三模式规则要求明确值、合法单位、精确数值；不承接旧倍率回退、猜价、随机stock、静默clamp或默认长宽。SHEIN PriceManager/PriceCalculator 与共享 productpolicy/internal/pricing 保留；只退 TEMU 同名旧 interface/adapter |
| 当前 AI/映射能力 | 保留现有当前产品/平台草稿/Agent/模型合同 | 不删 internal/ai、integration/openai、Agent/Skill/Tool/prompt 通用 owner、当前 CompletionDraft/官方规格能力。旧无入口 AI/SKU 自动映射本身不再保留、不补新的 AI 功能，不宣称当前能力逐函数等价 |
| shared sourceproduct Reader/Cache/Mapping、listingadmin/state/model/app/ports、jsonx/recovery/timeout、core/config/logger | 保留，只有 §5.3 精确 mixed 删除 | Amazon/采集、保留平台 API、SHEIN 发布/worker 等有消费者；不沿 outbound import 整删。共享成本配置、PlatformConfig/loader/default/validator/registry不动 |
| 当前图像/上传/填充/批准资产、Store observations、两种 SHEIN 属性 | 保留 | 原 owner 与原合同/测试保持；不得从 TEMU 名称推导跨平台删减 |

```text
Legacy decision: EXTRACT | RETIRE
Reusable behavior: 三个原值 DTO helper，及当前 goods/rules 已持有的有效约束。
Current owner: listingadmin；当前 Product/Asset/TEMU goods/rules/Target/Submission/Store、共享基础层和独立 worker。
Cutover/deletion condition: 用户确认 §5/§6/§6.1/§7、适用独立复核与正式准入满足，helper 脱钩及当前合同必要自检通过；不依赖运行数据迁移或内部试用前置。
```

### 5.2 86个整退 Go 文件

| 精确目录 | 文件（目录前缀适用于本行每一项） | 生产/测试 |
| --- | --- | --- |
| `internal/marketplace/temu/publishing/` | `numeric_format_policy.go`, `numeric_format_policy_test.go`, `pricing_policy.go`, `pricing_policy_test.go`, `sku_sn_validation_policy.go`, `sku_sn_validation_policy_test.go`, `sku_value_policy.go`, `sku_value_policy_test.go`, `spu_validation_policy.go`, `spu_validation_policy_test.go`, `submission_validation_policy.go`, `submission_validation_policy_test.go` | 6/6 |
| `internal/pipeline/` | `base_handler.go`, `context_impl.go`, `context_interfaces.go`, `errors.go`, `interfaces.go`, `parallel_handler.go`, `pipeline.go` | 7/0 |
| `internal/pipeline/handlers/` | `init_handler.go`, `logging_handler.go`, `validation_handler.go` | 3/0 |
| `internal/pkg/skugen/` | `skugen.go`, `skugen_test.go` | 1/1 |
| `internal/temu/ai/` | `content_rewriter.go`, `prompt_builder.go`, `property_mapper_core.go`, `property_mapper_helpers.go`, `property_mapper_utils.go`, `property_validator.go`, `service.go` | 7/0 |
| `internal/temu/context/` | `ai_property_mapping_input.go`, `ai_property_mapping_types.go`, `ai_types.go`, `sku_runtime_input.go`, `temu_context.go` | 5/0 |
| `internal/temu/filter/` | `prohibited_items_config.go`, `prohibited_items_detector.go`, `prohibited_items_detectors.go`, `prohibited_items_types.go`, `prohibited_items_utils.go`, `prohibited_items_weapons.go`, `sensitive_words_filter.go` | 7/0 |
| `internal/temu/handlerbase/` | `base_handler.go`, `models.go`, `property_types.go`, `sku_builder_interface.go`, `temu_handler.go` | 5/0 |
| `internal/temu/pricing/` | `interfaces.go`, `pricing_rule_calculator.go`, `pricing_rule_calculator_test.go` | 2/1 |
| `internal/temu/product/` | `price_handler.go` | 1/0 |
| `internal/temu/property/` | `attribute_extractor.go`, `conditional_validator.go`, `deduplicator.go`, `default_filler.go`, `mixed_attributes_processor.go`, `required_guardian.go`, `selection_validator.go`, `validation_config.go`, `validation_stats.go`, `validator.go`, `validator_condition.go`, `value_fixer.go`, `value_validator.go` | 13/0 |
| `internal/temu/rules/` | `percentage_sum_rule.go`, `rule_engine.go`, `rule_interface.go`, `text_renderer.go` | 4/0 |
| `internal/temu/sku/` | `ai_mapping_batch_processor.go`, `ai_mapping_handler_input.go`, `ai_mapping_input.go`, `ai_mapping_prompt_builder.go`, `ai_mapping_response_processor.go`, `ai_mapping_single_processor.go`, `ai_mapping_utils_helper.go`, `ai_mapping_validator.go`, `convert.go`, `mapping_processor.go`, `mapping_retention_test.go`, `spec_handler.go`, `variant_processor.go` | 12/1 |
| `internal/temu/spec/` | `dimension_selector.go`, `dimension_unifier.go`, `resolver_service.go`, `runtime_input.go` | 4/0 |

### 5.3 其他精确删除与 mixed 边界

| 路径 | 精确范围 | 必须留下 |
| --- | --- | --- |
| internal/listingadmin/gorm_profit_rule_api.go | 全文件：NewGormProfitRuleAPI、gormProfitRuleAPI/GetProfitRule、ProfitRuleToRespDTO；除旧TEMU外无consumer | ProfitRule领域模型、仓库/查询/持久化与fixture |
| internal/listingadmin/gorm_pricing_rule_api.go | §5.4搬出三helper后全文件：NewGormPricingRuleAPI、gormPricingRuleAPI/GetPricingRule、PricingRuleToRespDTO；constructor/converter无外部caller | PricingRule领域模型、仓库/查询及owner检查；下游mapping/filter仍用的helper原语义 |
| internal/listingadmin/management_api_types.go | 仅 ProfitRuleAPI/ProfitRuleReqDTO/ProfitRuleRespDTO、PricingRuleAPI/PricingRuleReqDTO/PricingRuleRespDTO 六个声明；没有保留 API/SHEIN/state consumer | Store/FilterRule/ProductImportMapping/RawJsonData及其他共享 DTO/字段/JSON，不顺手删其 ProfitRuleId 等引用持久事实的字段 |
| internal/core/config/platform_common.go | 仅 SensitiveWordsConfig、DefaultSensitiveWordsConfig、其GetWordsForLanguage/AddStaticWords/AddDynamicWords共5声明 | CostCalculationConfig/default/CalculateTotalCost及其他配置 |
| internal/core/config/platform_common_test.go | 仅 TestDefaultSensitiveWordsConfig、TestGetWordsForLanguage、TestAddStaticWords、TestAddDynamicWords、TestAddWordsToNilMaps、TestSensitiveWordsConfigMetadata与BenchmarkGetWordsForLanguage | 成本计算测试/benchmark保持；不删除整个mixed文件 |
| internal/marketplace/temu/publishing/README.md | 全文件；该owner与其最后consumer一起退出 | 当前goods/rules设计和源码保持 |
| prompts/temu/{attribute_mapping,content_rewriter,sku_variant_mapping}.yaml | 仅3 tracked 默认模板，随最后caller退出 | 所有其他平台/当前AI/图像prompt |
| internal/prompt/keys.go | KTemuAttributeMappingSystem、KTemuContentRewriterSystem、KTemuSkuVariantMappingSystem及专属注释 | 其他key与Registry/Store/Override语义 |
| internal/promptmgmt/{catalog.go,catalog_manifest.go} | 对应3key的清单项和3metadata项；temu专属category label及promptGroup的temu case | 其余category/group、通用回退/变量词汇表/HTTP/tenant授权/Store，不处理已有覆盖行 |
| data/prohibited_items_temu.json、data/sensitive_words_temu.json | 仅2 tracked静态默认文件，专属reader随包退休 | data/sensitive_words.json、SHEIN当前词表/品牌与现有实际文件不操作 |

### 5.4 必需 EXTRACT

将 `gorm_pricing_rule_api.go` 内原 `ptrFloat64`、`ptrString`、`flexibleTimeValue` 原样移到同一 `listingadmin` 的 `dto_value_helpers.go`（只含这3个private值helper，不引入新抽象/公共合同）；原 filter/mapping 调用点不改。

保留 `ptrFloat64` 对0/负数仍返回非nil指针、`ptrString` 对空串仍返回非nil指针、`flexibleTimeValue(nil)` 为空 FlexibleTime、非nil保留原时间的行为。不能替换为只收positive/nonempty的numeric_helpers。复用既有 mapping/filter DTO fixture并补必要的值/空值断言；先锁现有行为，再机械搬移。没有新的业务规则、事务或平台调用。

## 6. 验证与证据

设计阶段：只运行静态 import/构造器/资源检索、31命令 go list 元数据、文档路径/manifest一致性与原PR文档CI。未运行生产实例/真实provider/业务数据检查；没有当前行为验收或发布准入结论。

确认后的实施顺序在同一Writer/主要PR完成：锁§5.4原值行为 → owner内搬移 → 整体退§5.2/5.3 → 校准§6.1原守卫/当前文档 → 必要当前合同自检与稳定候选原PR CI → 一次最终独立交付检查。可用提交边界便于审阅/回滚，不另开机械小批。回滚仅涉及提交，不能因回滚启动旧执行器或操作数据。

本地只覆盖变化风险：listingadmin原mapping/filter/owner测试及3helper值行为、core/config保留成本配置、prompt/promptmgmt（隔离fixture）、保留temu/api与当前goods/rules合同、原退休/import/consumer/dependency守卫及独立worker编译/静态依赖。完整必需PR CI留稳定候选，不人工重复已覆盖的全量检查。不要求跑被删除包的go test、不运行旧业务constructor，不引入新runner/故障平台/人工验收流程。

证据分别记录准确HEAD、本地结果、PR CI、独立检查；授权合并后才记录实际merge与相同SHA的push/main CI。运行、provider、真实平台及用户验收保持 NOT_RUN；CI成功不等于上线或真实用户验收。

### 6.1 原检查、消费清单与文档的精确处置

| 原位置 | 最小处置 | 不得弱化 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 原absence加入14包、2 GORM文件、6资源和精确mixed退出声明；retained token/file表只移本批文件/六DTO与config项，保留当前goods/API/worker/共享规则断言 | 保留原已退休路径/声明与SKU/平台禁令。整退包原WalkDir/ParseFile/必需文件读取改从存续祖先index并精确过滤，不能忽略ENOENT/解析失败或skip用例；包级absence同时禁止任意新文件重建 |
| 原SKU/格式/条件规则AST检查及repository_structure历史format禁入 | 本批14包全退后，将原私有policy形状与retained定位断言替换为整包/重引入禁令；当前goods/rules有效业务断言保持原owner，修正format守卫诊断中的已退owner名 | 不全局禁PriceCalculator、PropertyItem、SpecInfo、ValidationResult等同名API/共享声明；旧TEMP_、自动填充 fixture不能逼当前owner支持旧设计 |
| tests/import_boundaries_test.go 的7项TEMU context/filter/rules/handlerbase/sku/product/pricing及已退store port检查（共8项） | 沿用store现有模式：loadGoFileIndex从保留internal/temu父root读取，按原子树+路径分隔符过滤，保留原禁令 | loader/读取/parse错误仍失败，原已退子树重新出现也受扫描；API client原独立检查保持 |
| TEMU SyncAndPricing/ProductStoreAndScheduler/RuntimeAndBridge管理及OpenAI allowlist | 仅移本批准确文件豁免，原已不存在的无关processor/pipeline_registry行不顺手扫；不存在子树用存续父root精确index，所有原target仍在 | 保留API client现用管理豁免和全部禁令；不泛化whole-directory许可或关闭scanner |
| TestPhase3PipelineDoesNotGrow | 原baselineMax=10不改；从存续internal祖先精确枚举pipeline子树以支持整退，并由原退休守卫强制包缺席 | 不改共享计数helper的错误处理/其他ceiling，不用保留空stub应付计数 |
| 原CurrentCommands依赖/独立SHEIN worker检查 | 对14包新增准确包及带/边界禁入；保留31cmd调查与既有当前命令/worker断言 | pipeline≠amazon/pipeline；temu/rules≠marketplace/temu/rules；不能禁整个temu根或API；SHEIN worker及Store同步原检查保持 |
| prompt通用管理测试 | 用现有“不在catalog的记录仍保留”fixture验证3key从catalog退出与覆盖语义保持 | 不删除/disable/迁移已有行，不改变Store/Registry/list/tenant权限或通用历史记录语义 |
| phase2-runtime-inventory.md | **只删 Legacy consumer register 的10条当前边**（下表）；可加本批窄来源注记，但原历史段不重写 | 全部其他当前行、dated inventory/计数/归类历史、ceilings保留；不是冻结整个文件后绕过源码一致性检查 |
| Legacy register、code-health-decisions、本文/索引及先前直接冲突保留设计 | 明确本批来源与整退结果；历史TEMU保留描述作窄superseded注记，保留原报告/原数字/证据和所有当前能力约束 | 不批量改历史Issue/PR或改写“当时确有consumer”的事实；设计阶段不标为已退 |
| .golangci.yml、两套及所有workflow、go.mod/go.sum、保护规则 | **零改**；包括原listingadmin -race、必需/安全/PG/release/command步骤与触发权限 | 没有新增验收工具或试用前置；若实际原CI合同冲突，先报具体冲突与最小处理范围，不擅自松门禁 |

消费清单精确10条（左为target，右为consumer，均加`task-processor/internal/`）：

| Target | 退出的准确 consumer |
| --- | --- |
| core/logger | pipeline |
| core/logger | temu/ai |
| core/logger | temu/filter |
| core/logger | temu/handlerbase |
| core/logger | temu/product |
| core/logger | temu/property |
| core/logger | temu/rules |
| core/logger | temu/spec |
| integration/openai | temu/ai |
| integration/openai | temu/sku |

core/logger → temu/api/client 等保留行不动；不能据这10条授权顺手处理其他遗留清单或provider inventory。

## 7. 确认、准入与停止条件

1. 当前仅设计，未修改生产/测试。适用独立设计检查完成并记录准确版本，用户确认§5、§6、§6.1、§7后，Issue/同一PR须明确 Independent Architecture / IMPLEMENTATION_READY / Ready，才可正式实施。确认同时仅窄替代上述旧TEMU实现保留决定，不改变当前能力/实例/数据边界。
2. 确认后同一Writer按完整闭包连续完成EXTRACT、删除、必要自检、finding分类/修复、提交推送和同PR维护，不机械拆小批。发现真实新当前consumer/兼容义务、新BLOCKER，或必须改变身份/tenant/事实/状态/事务/恢复/AI预算/平台副作用时，停止受影响项反馈；不从理论场景推导额外平台建设。
3. Reviewer按AGENTS先分类Finding、绑定当前Must/具体后果/阻塞层级。正常架构复核最多两轮；无新BLOCKER不无限加设计版本，非阻塞项转实现测试或Backlog。真正的Must缺口不能因到达两轮而豁免。
4. 禁止为了保留旧能力新增wrapper/alias/fallback/双事实源/同步器，禁止用删除测试代替当前owner有效约束。保留API、当前goods/rules、SHEIN价格/属性/worker、Reader/Cache/Mapping及所有未列mixed声明。
5. 实施完成进入In Review，交准确HEAD、原CI和一次最终独立检查。合并须另获准确HEAD授权；部署、真实API/模型/Store/Profile/Prompt覆盖记录和数据操作、关单均未授权，不增加内部试用前置。不把源码退休当成运行资源或数据不存在。
