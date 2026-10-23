# 旧 TEMU 默认属性处理管线完整闭包退休边界

> 用户已确认 [#909 productpolicy 完整退休边界](legacy-product-policy-closure-retirement-boundary.md) §1–§7（含 §6.1）。本文件对 `internal/marketplace/productpolicy` 六工件（doc/filter_rule/rule_checker/price_helper/fulfillment 及专属测试）和四项 KEEP 检查的保留含义已由该边界精确替代为 RETIRE；其余有效规则/API/采集、共享合同及独立 SHEIN 登录与 Store 同步不变。以下原正文、历史数字与验收证据保留，不恢复旧策略或授权既有实例/数据/凭据/进程操作。

> 后续用户已确认 [#819 TEMU剩余整链边界](legacy-temu-listing-closure-retirement-boundary.md) §5/§6/§6.1/§7。本文件对该闭包内旧 TEMU AI/SKU/spec、PriceHandler/PriceCalculator、property/context/TemplateState、pipeline/skugen及专属纯规则/管理投影/prompt的保留描述仅为历史边界，现由#819精确替代；当前平台API/规则、共享合同、SHEIN价格/两属性、采集/图像/Store观察与独立SHEIN登录/Store同步继续保留。原证据不改，不授权实例或数据操作。

## 1. 状态、用户结果与范围

- 日期：2026-10-18；Refs #29 / #301。用户已确认 #815 冻结设计 c8653f2c94c8e3d15692eb776db34bc51d53584f 的 §5、§6、§6.1、§7，要求按完整退休闭包推进，不再机械拆小批。本设计在一个主要 PR 一次覆盖入口、编排、四阶段、专属 context/cache/config/stat/DTO/alias 和测试/检查/文档处置。
- **USER_CONFIRMED / SOURCE_RETIRED；Design Basis: Reuse Existing Architecture**。正式生产修改前已在 #29/#301 登记并读回 Ready；§5 完整11文件闭包已源码退休，13文件及原合同保持。原设计N/A、CI/复核为历史，不替代准确实施HEAD/CI/最终独立检查；这些滚动证据维护于#815，合并另授权。
- 取证 actual main：**ccbd10132686ea2c0364fbc9ff4fc9c0cfaac2f3**。#812 已按准确源 HEAD 合并，其 matching push/main CI39132777972 SUCCESS；该证据只是前批，不替代本设计/实现验证。实施开工同步 actual main **49933ae1e0fa00defbe2ca336face29e4534fa90**，#814 的 Chat 输出/确认收据增量不触及本闭包或保留消费者。
- 单 Writer 会话01a133ef-757c-73e3-b8f6-76958c642397；复用附属独占 worktree issue29-temu-image-retirement/task-processor，新分支 codex/issue29-temu-property-closure-retirement-design。已核对接单、活跃会话和开放 PR，无同范围 owner；#813 图片/provider 与 #692 安装工作保持原 owner，不派额外 Writer。

使用者为内部试用、采集/Store/内容维护者。结果是退出没有外部装配的旧默认属性管线及其全部专属依赖，防止旧 Task owner 经局部 adapter/缓存/阶段合同复活。当前没有用户操作被此维护阻塞，**不增加内部试用前置**；无 UI/Figma 改动。

完整候选为 property 包内 **11个整生产文件、1379行、42个包级声明**。没有发现专属测试、mixed 生产删减或需新增 EXTRACT 的业务行为。该包另有 **13个被明确保留的 AI/SKU/spec 能力消费的文件**，不因命令不可达或与旧管线同包而退休；四个规则文件、handlerbase 当前合同也保留。本批没有下一轮再补的专属管线/缓存/阶段 DTO 债务，不整删 property 包。

非目标：退休/重写 AI 属性与 SKU 映射、修正已有保留规则、增加新当前 caller/formatter/默认填充管线、兑现旧并行/retry/命中率配置、改平台 API/DTO/正式 Product 事实。不得执行 main/constructor、真实模型/平台 API/下载上传/Preview/DryRun，读取实例配置/凭据/DB/Redis/已有 Prompt 覆盖记录/业务数据，或部署、迁移、重放、清理、关单。

## 2. Design Basis、authority 与不变量

本次实施采用 **Reuse Existing Architecture**；原设计阶段 N/A 不替代生产准入。依据：

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Project Boundaries](project-boundaries.md)：旧实现/测试不是 requirement，只允许 EXTRACT / RETIRE，不新增 wrapper、fallback、旧 ID 映射、双读双写或第二事实源。
- [TEMU 平台设计](temu-platform-v1.md) M3/M4、当前 goods/rules/publishing/Product/ApprovedAsset/TargetRecord 与 Submission/effect/receipt/UNKNOWN owner：已批准资料、官方规则及明确授权负责当前内容与平台副作用；旧 PropertyMapper 名称和“新的管道架构”注释不是当前架构准入。
- [#768 SPU边界](legacy-temu-spu-price-query-retirement-boundary.md)、[#749 保存/提交边界](legacy-temu-listing-submit-retirement-boundary.md)、[#771 SKU构建边界](legacy-temu-sku-skc-retirement-boundary.md)、[#773 类目模板边界](legacy-temu-category-template-retirement-boundary.md)、[#812 文本要点边界](legacy-temu-text-bullet-retirement-boundary.md)：此前保留 PropertyMapper/属性处理属于当批范围。用户本次确认只窄替代 §5 的11文件/42声明，不恢复已退休内容，也不撤销 AI/SKU/有效规则等其余 Must。
- [当前内容/Product](chat-task-object-goal-design-brief.md)、[Store current application](store-center-current-application-v1.md)、[Store observations](store-center-platform-observations-v1.md)、[独立 SHEIN 身份持久化](shein-worker-identity-persistence-boundary.md)：各原领域事实、权限、Store sync 与持久化 owner 不变。

路径：旧 TemuTaskContext → PropertyMapper → PropertyMappingOrchestrator → DefaultPropertyProcessingPipeline → mapping/validation/fixing/finalization stages → 专属 PropertyContext/cache/stat/config/alias；阶段调用的仍有保留消费者的 default filler/validator/rule engine 保持原 owner。不是把整包或泛型缓存迁往新 namespace。

不变量：不改变保留合同的身份/tenant/权限、canonical fact/schema、状态机/事务、幂等/retry/UNKNOWN/recovery、AI 准入/预算/计量及平台/provider 副作用 owner；不增加装配、公共领域合同、schema 或当前状态。本批11文件/1379生产变更行及原 guard/必要文档低于敏感提示线，已有批准架构覆盖有界删除；若实际出现新保留 consumer/兼容义务或必须改变这些边界，停对应项并补适用准入。

Must：§5 保留13文件、AI/SKU/AIBatchInput/content_rewriter prompt/key/catalog/规格处理、规则引擎/百分比规则/TextRenderer、handlerbase PropertyFeature/PropertyFeatureDetector/DimensionInfo、PriceHandler/ProfitRuleAPI、PriceManager/PriceCalculator、平台 API/DTO/有效规则/共享合同、Reader/Cache/Mapping、两 SHEIN 属性、四 TemplateState 字段、当前图像/采集/Store观察、独立 SHEIN 登录 worker/Store sync及此前其它保留能力。通用 Prompt 存储/管理/已有覆盖记录不动。两套及其它 workflow、依赖/保护/必需安全检查/ceilings 不改；无新增 Accepted Risk。

Threat Model 只覆盖误删 AI/SKU/spec 共享行为、泛型/同名跨包误禁、把旧默认填充/局部修复当当前资料事实、把进程内 cache/统计当持久化 owner 或运行清理授权。不扩成新属性治理、缓存平台、IAM 或验收矩阵。

## 3. 完整消费者与架构问题

全仓构造器、类型、方法调用、别名及测试已核对；实际 Go imports 中 property 的直接生产 importers 只有 temu/ai、temu/sku、temu/spec。**31个 cmd 的静态依赖并集中 property 为0**。这三条保留边只消费下表叶子能力，不装配11文件中的任何包级 owner；13个保留文件中也没有对42个拟退声明的标识符引用。未执行 main/constructor；源码证据不证明实例、外部调用方或历史数据不存在。

| contract → implementation → injection → consumer | 实际消费者/行为 | 决定 |
| --- | --- | --- |
| PropertyMapper/NewPropertyMapper → Orchestrator | 只有定义、自用和原 retirement guard 的保留条目；旧 SPU caller 已在 #768 退出，无包外生产/测试装配 | RETIRE adapter 及完整传递专属闭包，不只移走一层 facade |
| Orchestrator → Pipeline → 四stage | 固定100/200/300/400排序；读旧 TemplateInfo/GoodsProperties、TemuTaskContext.GetContext，处理后覆写传入 ExtensionInfo 的 GoodsProperties | RETIRE全部编排/阶段及专属合同；旧模板与平台 Product/Extension DTO、四 TemplateState 字段保留 |
| MappingStage → DefaultPropertyFiller | 只按模板填必填默认值，不调用 AIService/AIPropertyMapper，不是真正的模型映射入口 | RETIRE此旧调用方；filler仍被 RequiredPropertyGuardian/AIPropertyMapper使用，保留 |
| ValidationStage → ValidationRuleEngine | 可注入原engine；非严格模式错误只记录后继续，严格模式才返回 | RETIRE阶段；engine、接口、百分比规则仍被保留filler消费 |
| FixingStage → PropertyFeature | 旧局部修复把首项改100，不保证其它同PID项合计仍100；只改当前切片 | RETIRE阶段，不迁移为当前规则或修好后复活 |
| FinalizationStage → PropertyDeduplicator | 去重后统计缺必填；非严格模式只记录，严格模式报错 | RETIRE阶段；deduplicator仍被AI/filler消费，保留 |
| Pipeline/StageChain/StageResult → PropertyContext/ProcessingStats/ProcessingConfig/Cache | 配置、统计、结果DTO、排序与阶段接口仅在闭包内，StageChain/StageResult无额外生产caller | 全部RETIRE，不留下“通用”阶段/统计框架 |
| PropertyContext → NewPropertyCache → InMemoryPropertyCache | 每次构造开启cleanup goroutine，TTL/map/queue只在进程内；接口无Close，管线没有调用具体cache.Close。专属cache factories/key/stats无保留caller | 整文件/生命周期实现RETIRE，不另外造关闭协议、缓存服务或执行已有实例清理 |
| property.PropertyFeature / PropertyFeatureDetector / NewPropertyFeatureDetector | 前者为context.go的handlerbase类型alias；后两者为feature_detector.go的alias/var re-export，仅pipeline/fixing/cache消费 | 随最后callerRETIRE局部aliases；handlerbase canonical feature/detector与rules同名alias保持 |
| AIPropertyMapper/AIService → property validators/fixer/guardian/defaults/stats | 直接构造七类validator/helper、条件validator与统计；default filler/guardian继续用rules engine | 保留原13文件中的这组行为和全部原调用，不把“无当前cmd装配”当撤销AI保留Must |
| SKU单项映射 → AttributeExtractionManager；SpecDimensionUnifier → MixedAttributesProcessor | 生产选择器与调用仍存在；两者有包内专属 extractor/helper | 保留两个完整文件及当前SKU/spec边，不退休共享依赖 |

旧管线可见的问题包括：默认填充冒用映射名称、百分比局部修复可留下错误总和、缓存协程生命周期未由管线关闭、统计与配置字段不等于真实执行能力。没有当前装配/事故证据，分类 **NOT_APPLICABLE（当前内部试用阻塞）**，按 RETIRE 退出；不重建旧链，不由这些现象扩展新产品/缓存平台需求。保留叶子行为本轮原样，不签发其全局正确性或当前生产就绪认可；发现实际 Must 缺陷按 AGENTS 先分类处理。

## 4. 持久化与副作用 owner

| 边界 | 所有权与效果 | 本批处置 |
| --- | --- | --- |
| 旧属性结果 | 旧 TemuTaskContext/传入平台 ExtensionInfo 的进程内 GoodsProperties；管线/stage可就地修改属性值/列表 | 只删旧 mutator/编排，不删除平台DTO/字段、当前Product资料或历史行 |
| cache/config/stat/stage结果 | 内存map、channel、goroutine、TTL、锁、时间/计数；无持久化表/队列/文件owner | 源码退休，不查运行状态/缓存、不清DB/Redis/文件/卷，不添加迁移/恢复 |
| 旧默认管线的外部调用 | 11文件无实际HTTP/provider/平台请求或DB/消息写入；logrus日志/本地协程不是平台mutation。不存在外部自定义stage注入的当前源码证据 | 不把同包AI/规则/SKU未来可能调用的能力算成该管线真实副作用；不运行constructor或测试真实模型 |
| 保留AI/SKU/spec/平台合同 | 原AI/provider准入、预算、请求及规格查询/图像处理等副作用归各原owner | 全部保持，不新接旧管线，也不以本次删除取消原安全/恢复要求 |
| 当前Product/Store/独立SHEIN | 当前Product/版本/Review/采用、官方effect/receipt/UNKNOWN、Store观察和worker身份持久化沿原owner | 无新事实源、schema、授权/事务/恢复协议或数据操作 |

## 5. 完整删除与保留矩阵（用户已确认，源码已退休）

下列均在 internal/temu/property/；**整退11生产文件/1379行，不存在mixed生产删减或整退测试**。本次没有新EXTRACT代码、替代pipeline、caller或旧fixture迁移。

| 文件（基线行数） | RETIRE包级声明及全部receiver/内部实现 |
| --- | --- |
| mapper_adapter.go（81） | PropertyMapper、NewPropertyMapper |
| orchestrator.go（139） | PropertyMappingOrchestrator、NewPropertyMappingOrchestrator |
| pipeline.go（213） | PropertyProcessingPipeline、DefaultPropertyProcessingPipeline、NewPropertyProcessingPipeline |
| mapping_stage.go（83） | PropertyMappingStage、NewPropertyMappingStage |
| validation_stage.go（79） | PropertyValidationStage、NewPropertyValidationStage、NewPropertyValidationStageWithEngine |
| fixing_stage.go（99） | PropertyFixingStage、NewPropertyFixingStage |
| finalization_stage.go（85） | PropertyFinalizationStage、NewPropertyFinalizationStage |
| stage.go（177） | PropertyStage、BasePropertyStage、NewBasePropertyStage、StageError、NewStageError、StageResult、NewStageResult、PropertyStageChain、NewPropertyStageChain |
| context.go（180） | PropertyContext、PropertyFeature、ProcessingStats、StageStats、ProcessingConfig、NewPropertyContext、NewProcessingStats、NewDefaultProcessingConfig |
| cache.go（231） | PropertyCache、InMemoryPropertyCache、CacheItem、NewPropertyCache、NewPropertyCacheWithTTL、GenerateFeatureCacheKey、CacheStats |
| feature_detector.go（12） | PropertyFeatureDetector、NewPropertyFeatureDetector；仅property局部alias/var，canonical handlerbase定义和rules别名保留 |

保留13文件全部原实现；这是沿用用户明确保留的AI/SKU/spec与有效共享规则，不是为无关预存函数增加永久保留权或新增consumer：

| 保留文件 | 保留消费者 / 原行为 |
| --- | --- |
| attribute_extractor.go | SKU ai_mapping_single_processor.go → NewAttributeExtractionManager；原三类extractor/result/manager |
| mixed_attributes_processor.go | spec/dimension_unifier.go → NewMixedAttributesProcessor；混合规格值处理 |
| validator.go、validator_condition.go | AIService / AIPropertyMapper → NewPropertyValidator；原必填/条件/选项校验 |
| conditional_validator.go | AIPropertyMapper → NewConditionalPropertyValidator；条件约束 |
| value_validator.go、value_fixer.go、validation_config.go | AIPropertyMapper → NewPropertyValueValidator/NewPropertyValueFixer → 原config；DefaultPropertyValidationConfig返回值类型仍为PropertyValidationConfig |
| required_guardian.go、default_filler.go | AIPropertyMapper → NewRequiredPropertyGuardian/GetDefaultFiller → NewDefaultPropertyFiller → rules engine |
| deduplicator.go | AIPropertyMapper及上述filler → NewPropertyDeduplicator |
| selection_validator.go | AIPropertyMapper → NewPropertySelectionValidator |
| validation_stats.go | AIPropertyMapper → NewPropertyValidationStatsCollector；原PropertyValidationStats/PropertyTypeStats/PropertyFixDetail，区别于拟退ProcessingStats/StageStats |

另保留 rules/rule_interface.go、rule_engine.go、percentage_sum_rule.go、text_renderer.go；handlerbase/property_types.go中的canonical PropertyFeature/PropertyFeatureDetector/NewPropertyFeatureDetector与DimensionInfo；AI/SKU/spec所有生产/测试/Prompt、平台API/client/DTO、旧mixed TemuTaskContext和四TemplateState字段、当前goods/rules/publishing/PriceHandler及此前全部Must。不借同名做全局删除/禁令。

```text
Legacy decision: RETIRE
Reusable behavior: AI/SKU/spec仍消费的13个property文件与规则/平台合同原样保留；无新增EXTRACT或替代管线。
Current owner: 现有AI/SKU/spec/property叶子与rules/handlerbase；当前Product/Store/独立SHEIN各原事实和副作用owner。
Cutover/deletion condition: 用户确认§5/§6/§6.1/§7 → Issue Ready先登记读回 → 刷新完整闭包 → 原guard RED→精确删除GREEN → 准确实现HEAD原CI/最终独立检查；无运行迁移或数据清理。
```

## 6. 验证与交付

原设计阶段仅文档/只读调查，其CI与复核保留为历史；用户确认后已在生产修改前登记并读回Ready，按以下方法实施。准确候选、自检、原PR CI和最终独立代码检查记录于PR，设计CI不代替实施；运行/平台/provider/用户验收 **NOT_RUN**。

本批在同一主要PR按以下方法连续完成整闭包：

1. 开工同步actual main，复核11文件/42声明、13保留文件、三个生产importer与31命令依赖；发现新真实consumer/具体当前兼容义务停相关项，不扩大授权。
2. 仅在原 tests/listingkit_backend_retirement_test.go 增11精确缺席文件及42个property局部包级声明禁令；移除原mapper旧保留条目，新增下表保留合同断言。删除前保留断言应GREEN、待退文件/声明导致RED；再一次完整退休确认GREEN。
3. 运行保留property/rules/handlerbase、AI/SKU/spec/context/API、当前TEMU goods/rules/publishing/productpolicy、SHEIN content/submitprep的相关既有回归/编译；原退休/import/consumer/worker/ceiling检查与current-application/shein-login-worker build。无测试包只报告编译，不执行main/constructor/真实provider。
4. 稳定候选/最终组合才跑完整原PR CI和一次最终独立代码检查；finding先分类，修复只复核实际增量，不重跑未变代码或新建runner/验收平台。
5. 原worktree既有空目录残留若影响退休guard，准确候选干净源码snapshot另验证并如实分列，不清理运行资产或放宽检查。合并另需准确最终HEAD授权，matching main CI后核验。

### 6.1 原检查与文档精确处置

| 原位置 | 已确认实施变更 | 保持边界 |
| --- | --- | --- |
| 原absence列表 | 增§5的11准确Go文件路径，不禁止整个property目录存在 | 原所有缺席路径、SKU9 fixture和此前安全断言保留 |
| 原mixed包AST walk | 原sku/filter/product/rules/handlerbase五包保持，增加property作为第六个mixed包；复用原包级FuncDecl(Recv=nil)/TypeSpec/ValueSpec检查，只在property匹配§5的42声明 | 覆盖type alias/var re-export；不禁止方法、field、selector、local或别包同名。CacheItem/CacheStats/ProcessingStats/StageError/PropertyFeature等泛名不能全局禁；handlerbase/rules同名合同保持，不建新scanner |
| 原retained token map | 仅移除mapper_adapter.go的PropertyMapper/BuildGoodsProperties条目；新增下行13文件的准确保留声明/当前调用断言，删除前即满足 | 不把13文件改为“遇缺跳过”，不新增consumer维持旧策略 |
| 同一map中的13保留文件 | attribute_extractor：AttributeExtractionManager/NewAttributeExtractionManager；mixed_attributes_processor：MixedAttributesProcessor/NewMixedAttributesProcessor；validator：PropertyValidator/NewPropertyValidator；validator_condition：func (v *PropertyValidator) filterByPropertyRelations(properties []models.PropertyItem, data temucontext.PropertyMappingData) []models.PropertyItem、func (v *PropertyValidator) checkShowCondition(templateProp temutemplate.TemplateRespGoodsProperty, refPidToSelectedVids map[int]map[int]bool) bool；conditional_validator：ConditionalPropertyValidator/NewConditionalPropertyValidator；value_validator/value_fixer：PropertyValueValidator/NewPropertyValueValidator、PropertyValueFixer/NewPropertyValueFixer；validation_config：PropertyValidationConfig与func DefaultPropertyValidationConfig() PropertyValidationConfig；required_guardian/default_filler：RequiredPropertyGuardian/NewRequiredPropertyGuardian/GetDefaultFiller、DefaultPropertyFiller/NewDefaultPropertyFiller；deduplicator：PropertyDeduplicator/NewPropertyDeduplicator；selection_validator：PropertySelectionValidator/NewPropertySelectionValidator；validation_stats：PropertyValidationStatsCollector/NewPropertyValidationStatsCollector及三个原结果类型 | 使用当前源码实际type/constructor/方法声明、参数/返回类型，不只匹配注释/泛名；保留13文件及AI/SKU/spec源码相对基线零diff，不新增业务测试或认可旧默认值的新产品需求 |
| 同一map的保留消费者/规则 | AIService/AIPropertyMapper中上述构造/条件validator/GetDefaultFiller注入、SKU的property.NewAttributeExtractionManager、SpecDimensionUnifier的property.NewMixedAttributesProcessor选择器；rules接口/engine/NewValidationRuleEngine/RegisterRule/ValidateAndFixAll、PercentageSumRule构造/Validate/Fix及原default_filler的engine调用；handlerbase canonical PropertyFeature/PropertyFeatureDetector/NewPropertyFeatureDetector | 沿用/补全原map准确条目；PropertyValidationStats不同于拟退ProcessingStats，handlerbase feature不同于局部alias，不能只按字符串全局禁。已有CheckText/API/DTO/worker等断言保持 |
| CurrentCommands保留路径与测试 | 无待退property路径条目可删；未发现专属property测试，不删除/移动SKU mapping_retention_test或其它既有测试 | 原命令/standalone worker/retained platform存在与依赖检查保持 |
| tests/import_boundaries_test.go、phase2-runtime-inventory.md、.golangci.yml | **零变更**。property包及AI/SKU/spec import边保留；TextRenderer/PriceHandler仍消费core/logger，OpenAI准入、loader/root/deny/ignore/ceilings仍有效 | 不重写dated inventory/声明计数、不趁本批放宽安全/CI/豁免 |
| 必要文档（本批） | Legacy Register、code-health property行、本文/索引及#768的PropertyMapper保留和#773的旧属性编排consumer说明窄更新为11文件source-retired/13文件原消费者保留；#749/#771/#812其余保留边界不改 | 原报告82声明/配置可达措辞留作历史，不把old root ./internal/temu当当前测试目标，不整包改写旧批次证据 |
| 两套及其它workflow、依赖/保护 | **零变更** | 不删必需/安全检查，不新增试用前置 |

## 7. 确认与停止条件

1. 本设计一次覆盖整条默认属性管线及所有专属生命周期/阶段合同，不将adapter、stages、cache/context/DTO拆到后续PR。用户明确确认§5/§6/§6.1/§7后，在同PR登记并读回Reuse Existing Architecture / Ready才改生产；原设计阶段N/A仅覆盖设计；本次已按Reuse Existing Architecture登记并读回Ready。
2. 冻结后范围内实施、finding分类/修复、自检、commit/push、PR维护连续完成，不逐项请示。只有新的真实保留consumer/当前具体兼容义务，或必须改变身份/tenant/权限/事实/schema/状态/事务/恢复/副作用边界，才停对应项报告并补适用准入。
3. 不新增旧consumer、wrapper/fallback、双写或新事实源来维持旧框架；不将这项维护或新增工具/治理/矩阵变成内部试用前置。13保留文件是明确Must和现有消费者边界，不代表本批新增runtime/正确性/生产认可。
4. 实施完成In Review，交准确HEAD原CI与最终独立代码检查；实现者/CI/AI不签发用户或生产验收。非Blocker不自动扩Threat Model/重开架构，未满足当前Must仍阻止合并。
5. 合并另需用户准确最终HEAD授权与正常expected-head；#29/#301保持OPEN。部署/实例/凭据/已有Prompt覆盖记录/业务数据/迁移清理/关单未授权，运行/平台/provider/用户验收保持NOT_RUN。
