# 旧 TEMU SKU/SKC 构建链退休边界

> 后续窄替代：用户确认 #797 [图片编排退休边界](legacy-temu-image-orchestration-retirement-boundary.md) 后，仅其中七文件及专属 helper/alias/三个 AssetState 字段退出；本批保留的 ImageProcessor/ParallelImageProcessor、构建/填充/上传/vision 与平台合同不变，历史消费者和批准证据保留。

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

后续用户确认的 [#773 类目/模板边界](legacy-temu-category-template-retirement-boundary.md) 已退休本文当批保留的五个 category/template handler 与两个专属 DTO 文件。下文消费者与删除前矩阵保留为历史取证；平台 API/DTO、四个 TemplateState 字段、有效规则和其它共享能力仍保留，仅这七文件由后续矩阵窄替代。

- Status: **USER_CONFIRMED / SOURCE_RETIRED**。用户已确认冻结设计 3ceea910b709d12863d62165a18064dd16070848 的 §5/§6/§6.1/§7；三项纯值行为已抽取，六文件与混合构建专属闭包已退休。合并与真实操作仍须单独授权。
- 调查基线：main **bdd5bae1c07bdf22c861720a814dc19f53664b1d**，已包含 #768；实施开工同步 actual main **849007918ccb2e07bb98918d18ad0b829237005d**，#770 仅 UI/文档，无范围变化。准确实现 HEAD、CI 与独立检查维护于主要 PR；旧设计证据仅历史。
- 当前 **Design Basis: Reuse Existing Architecture；Admission: Ready**。用户确认后，生产修改前已在 #29/#301 登记冻结依据、owner/不变量与 Ready。原设计 N/A 只覆盖当时文档/只读映射，设计 CI 不代替实施结果。
- 保留 AI/SKU 映射、PriceHandler、平台 API/DTO、有效规则与此前全部保留能力。两套及其它 workflow、必需/安全检查、保护规则、ceiling 和依赖不改；不操作实例或数据，不增加内部试用前置。

## 1. 用户结果、范围与非目标

使用者是维护当前内部试用、采集与 Store 能力的维护者。交付为退出无装配旧 SKU/SKC payload 构建编排、保留确定性值规则和 AI 映射的准确矩阵，避免再次将旧 Task、默认数字身份和 raw payload 当作当前执行入口。当前没有用户操作被此维护阻塞，设计与后续源码退休都不是试用前置。

已退休的整文件范围为 `internal/temu/sku/` 的六个生产文件：builder.go（174 行）、item_builder.go（475 行）、skc_builder.go（265 行）、parallel_builder.go（103 行）、build_input.go（84 行）、price_calculator.go（17 行），基线共 **1,118 行**。另精确裁剪混合 variant_processor.go、convert.go、spec_handler.go 的构建专属部分；不是删除整个 sku 包或 SkuVariantProcessor。

已先按 §4.1 抽取三项已有值规则，再退出旧构建 owner。`SkuPriceCalculator` 只有 logger 字段/constructor，没有计算方法或消费者；它不是应保留的 PriceManager/PriceCalculator，也不是 product.PriceHandler。三者不能按近似名称一起删除。

非目标：新刊登入口、平台能力、AI provider 接线、工厂/缓存/恢复平台、schema、状态机、公共领域合同或验证工具；修复保留 AI/价格/图像实现的历史行为；运行 constructor/main、真实 API/AI、采集、上传、Query/Preview/DryRun、配置/凭据/DB/Redis/Profile/历史日志读取；部署、重放、迁移、清理、关闭 Issue。

## 2. Design Basis、Authority 与不变量

依据：

- [Greenfield](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md)、[Project Boundaries](project-boundaries.md)：只有 EXTRACT / RETIRE，无 wrapper/fallback、旧 ID 映射、双读双写或第二事实源。
- 已确认的 [保存/提交](legacy-temu-listing-submit-retirement-boundary.md)、[前置](legacy-temu-preflight-retirement-boundary.md)、[变体](legacy-temu-variant-retirement-boundary.md)、[筛选/AI 前置](legacy-temu-filter-mapping-retirement-boundary.md)、[Store/初始化](legacy-temu-store-init-retirement-boundary.md)、[SPU/零售价查询](legacy-temu-spu-price-query-retirement-boundary.md)：旧 Task 入口已退出，共享规则/映射/价格仍按原 owner 保留。此前“保留 SKU 构建”的陈述是当批删除边界；用户确认本设计后，仅 §5 的旧构建 caller 保留项由本矩阵替代，其它保留边界不变。
- [既有 TEMU publishing](../../internal/marketplace/temu/publishing/README.md)：规则与 payload 值处理 owner；已有私有 submissionSpec 可复用，不需新公开 DTO 或依赖旧 Task/API DTO。
- [当前 Store](store-center-current-application-v1.md)、[观察](store-center-platform-observations-v1.md)、[独立 SHEIN 身份/持久化](shein-worker-identity-persistence-boundary.md)：事实、授权、凭据、同步及恢复归原 owner。Figma N/A，无 UI/IA 修改。

本次实施复用以上架构：只抽取已执行的无 IO 私有值规则、删除旧编排，并收窄无外部消费者的包内 Go constructor 接线。它不是新增/改变公共领域合同、AI 调用/准入/预算/计量、平台 retry/UNKNOWN、身份/tenant、持久化事实、状态机或事务 owner。保留的映射算法、输入/响应合同与 provider 调用实现不变。若实际需要改变上述边界或增加当前消费者，停止对应部分，补适用 Independent Architecture 准入，不能在实现里临时发明架构。

Must：保留 AI/SKU 映射与 AIBatchInput、PriceHandler 全部售价/库存/倍数行为及 ProfitRuleAPI、PriceManager/PriceCalculator、平台 API/同名 DTO/re-export、有效规则/共享合同、两种 SHEIN 属性、Reader/Cache/Mapping、当前采集/Store 观察、独立 SHEIN 登录 worker 及 Store sync。原 guard 的有效覆盖不能减少。

Should：源码、当前清单及直接冲突说明与实际边界一致。Out of Scope 如 §1；无新增 Accepted Risk。Threat Model 只覆盖误删共用 AI/规格/价格/图像规则、旧默认值进入当前领域、旧入口复活、上传/查询效果 owner 混淆及源码退出被误报为运行态清零；不扩张为新 IAM/预算/重试系统。

## 3. 实际消费者、注入与调用

全仓 Go 导入搜索没有 `task-processor/internal/temu/sku` 的外部导入；NewSkuBuilder 仅有定义。六个候选文件及构建方法只由本包闭包相互引用，无外部构造、直接实例化或注册；测试中还有准确保留字符串，应按 §6.1 校准，不能当作运行消费者。`go list ./cmd/...` 为 **31 个当前命令**，其依赖并集里 sku 为 **0**。只求静态图，没有运行命令 main；这不证明历史实例或资产已清零。

| 合同 → 实现 | 注入/消费及实际效果 | 决策 |
| --- | --- | --- |
| NewSkuBuilder → PriceHandler、ImageProcessor/ParallelImageProcessor、SkuSpecHandler、SkuItemBuilder、SkuSkcBuilder、SkuMappingProcessor、SpecResolverService、SkuVariantProcessor | 唯一总装；SpecResolver 以 builder 自身作为 SpecQueryAPI。Name/HandleTemu/BuildVariantSkcs/CreateDefaultSkc/ProcessSkcItem 等无外部调用 | RETIRE 总装和构建方法；保留原价格、图像、映射/规格能力，不能因构造图整包删除 |
| SkuVariantProcessor → BuildVariantSkcs/BuildSkcsFromAIMapping/CreateDefaultSkc | 选已有或生成的 AI 映射、校验/解析规格 → 单/多 SKC → TemuProduct.SkcList 内存赋值 | 只退这三种构建入口与 §5 列出的专属调用链；不退同 receiver 的 AI 生成、分批、修正、验证和规格解析能力 |
| SkuSkcBuilder → SkuItemBuilder / SkuParallelBuilder | 模板/用户规格分组、ASIN 去重，串行或并行处理图像，再组 raw SKU/SKC；并行失败可走串行路径 | RETIRE 此 payload 装配与回退，不复制成新工作流；不删共享映射/图像实现 |
| SkuItemBuilder → PriceHandler、skugen、ImageProcessor、SkuSpecHandler | 读变体/Store 配置、算价格库存/编码、写借用 AsinSkuMap，组包装/物流/规格/图像 DTO | 先抽取 §4.1 三项值规则，再退出 Task、raw DTO 构建和专属 helper；原能力按原 owner 留存 |
| SKUVariantBuildInput / SKUProcessInput | 仅 item/parallel 文件构造和消费；借用 TemuProduct/SKUBuildRuntime，失败时可能降级 | RETIRE 六文件闭包内的输入包装；不删除 context.SKUBuildRuntime 或其构建/映射方法 |
| SkuPriceCalculator | 全仓仅类型/constructor 定义，无方法和消费者 | RETIRE 孤立空壳；不匹配或触碰其它 PriceCalculator |
| SkuVariantProcessor 的 AI 部分 → ChatCompleter / SkuMappingProcessor / SkuSpecHandler / SpecResolverService | AI 单批/分批及归一化共用同 receiver；八个 ai_mapping* 文件和 mapping_processor.go 仍依赖它 | 必须保留原 receiver、AI/验证方法及映射输入响应；按 §5 最小去除构建耦合 |

架构问题：构建、AI 请求、规格解析和 raw payload 混在同一个 receiver/constructor；BuildVariantSkcs 的错误可只警告并返回 nil，item 校验失败也可只记日志。并行图像路径失败后回退串行，没有构建层 durable receipt 判断已发生上传，可能再次调用上传；不能把它当作当前可靠副作用协议。此处是源码风险，不宣称已发生重复上传或数据事故。当前无装配用户路径，分类 **NOT_APPLICABLE（当前试用阻塞）**；通过退休该编排收敛，不修旧链或新建恢复平台，保留实现不等于当前执行已验收。

## 4. 持久化与平台副作用 owner

| 边界 | 原 owner / 可观察效果 | 处置 |
| --- | --- | --- |
| Product、Store、Observation、Submission、身份/授权/凭据 | 当前领域、Organization 与原 repository/workflow owner；本闭包不直接持有这些 durable facts | 不迁移 raw SKU/TemuProduct 为新事实，不新建数据库、outbox、幂等键或第二保存链；源码删除不触碰真实状态 |
| TemuProduct.SkcList、AI 映射、AsinSkuMap | 借用进程值。SKUBuildRuntime.SaveAsinSkuMapping 实际按 outSkuSN→ASIN 写 map；方法名不是持久化收据 | 删除旧构建写入 caller；context 原字段/方法、SKU 映射响应和原所有权保留，不反向修正 map 方向或新建持久化 |
| 价格/库存/配置 | PriceHandler 经 ProfitRuleAPI 读取原 Store/利润规则，使用现有 productpolicy/publishing/skugen；保留 nil runtime 默认数字身份和随机库存的原实现不表示认可为当前合同 | PriceHandler/ProfitRuleAPI/DTO/repository 原样保留；只退 builder/item 两个旧 caller，不新增使用默认身份的消费者，不操作规则或账务 |
| 规格查询/解析 | builder.QuerySpecID → 原 QueryAPI.QuerySpec → POST /mms/marigold/edit/commit/spec_query，经 APIClient/原鉴权发送；SpecResolverService 原注入 SpecQueryAPI | 退 builder 专属 query adapter；原平台 API/DTO/port、ResolveSpecRuntimeInput、resolver/HasTemporaryIDs/解析与标识更新保留，不把查询名称当作无副作用/可安全重试保证，不调用平台 |
| 图像 | ImageProcessor/ParallelImageProcessor、dimension/carousel builders、upload processor/service 与原 APIClient；可下载、检测、渲染、获取签名和上传 | 只退旧 SKU 编排 caller；所有图像/vision/provider/上传/API/错误 owner 原样保留；不因“构建”一词将上传当作纯内存动作 |
| AI 映射 | 原 SkuVariantProcessor → ChatCompleter.CreateChatCompletion，可能发生外部请求/费用。单批使用原 timeout + context.Background，分批遇错返回，原归一化/限额/提示词不变 | 保留八个 ai_mapping* 文件、模型选择/温度/timeout、批次和错误行为；不调用 provider，不宣称预算/取消/UNKNOWN 已验收，不新增默认客户端或接线 |

没有本轮新的幂等/恢复 owner；不重试历史请求、判断历史结果、释放预占/锁、续跑任务或读取真实配置/实例。发现具体当前消费者、外部可观察兼容义务或 durable owner 时停止对应删除并报告；不自动加兼容。

### 4.1 有效值行为与旧装配行为逐项处置

本轮不重新解释平台最新规范，只保存以下已存在的确定性代码行为；不新增平台准入或当前提交入口。

| 原行为 | 决策 / owner | 精确语义 |
| --- | --- | --- |
| item.deduplicateSpecs | EXTRACT 为既有 publishing 的私有值函数，复用 submissionSpec | 以 ParentSpecID 精确字符串为 key，保留首次值与原顺序；不 trim/排序/过滤空值。同一空 parent 也只保留首次，不修改输入；不把旧 DTO metadata 搬入新合同 |
| item.buildSkuSpecList 的未解析临时 SpecID 检查 | EXTRACT 为 publishing 私有纯谓词，与去重分离 | 对 SpecID 精确、区分大小写的 TEMP_ 前缀作检测；原顺序是先按 parent 去重，再检测保留项，不能因被淘汰重复项的 TEMP_ 新增拒绝。不新增 ParentSpecID 检测、解析、默认规格或错误修复。旧 caller 的日志/返回空 DTO slice 不保留为新执行协议；resolver/HasTemporaryIDs 的原独立行为不变 |
| item.buildMultiplePackage 中 classification=1 的单品约束 | EXTRACT 私有 scalar 值函数 | 输入 classification=1 时 pieces/individuallyPacked 输出 1/1；其它 classification 原数值透传，包括 0/负数，不加新验证。不提取原缺失字段默认赋 1、字符串重复字段或完整包装 DTO；不启用新调用入口 |
| 图像最多 10 张，尺寸图优先、保持顺序 | 已有 ParallelImageProcessor.ApplyImageResults 包含同一规则，原位保留 | 不复制第二份算法，不让 publishing 依赖 image/API；item 的重复实现随 caller 删除，不新增图像数量门槛 |
| 物流数值、价格、SKU 生成 | 已有 publishing.FormatWeight/FormatDimension、productpolicy、PriceHandler、skugen 原位保留 | 只删旧包装/日志/配置解析 caller，不复制、不修默认值、倍率/货币换算或随机库存 |
| 规格验证/解析、用户规格转换、尺寸判断、AI 响应修正与映射数量修复 | 原 SkuSpecHandler、SpecResolverService、SkuMappingProcessor 和 AI owner 保留 | convertUserInputSpecsToGoodsSpecProperties 仍被 AI input 使用，绝不可作为构建专属删除；ValidateSpecs、IsSizeSpec、CreateDefaultSpec、convertSpecInfos 保留 |
| 模板颜色分组、笛卡尔积补组合、无匹配组合日志、默认映射挑首项、ASIN 串并行筛选、USD/市场价二倍等 raw payload 固定赋值、净含量传递 | RETIRE 此旧装配与专属 helper | 不把未实现的缺失 SKU 生成、默认库存 10、浮点转分/翻倍、旧空字段填充、map 顺序或吞错升级为当前领域规则；不丢弃原 AI/平台 DTO 或其字段 |

私有函数只接受 submissionSpec 或 scalar 值，不导入旧 Task/context/model/API DTO、pipeline/provider/logger；不新增公开类型、adapter、legacy wrapper 或第二事实源。用户确认以上选择性抽取后才编码；若出现额外有效规则或需新公开合同，仍须先报告相应缺口。

## 5. 精确抽取、删除与保留矩阵

以下是**用户确认后已实施的精确矩阵**。删除前行为与消费者取证仍按原基线保留；保留实现不变。

| 对象 | 决策 | 精确边界 |
| --- | --- | --- |
| sku/builder.go、item_builder.go、skc_builder.go、parallel_builder.go、build_input.go、price_calculator.go | 先 §4.1 EXTRACT，再 RETIRE 六全文件 | 对应类型/constructor、SKUVariantBuildInput/SKUProcessInput、item 三个私有 payload struct 及独占 helper 全退，不保留 alias/receiver 空壳或兼容串行路径 |
| variant_processor.go | 保留混合 owner，仅退构建闭包 | 删除 BuildVariantSkcs、resolveVariantAIMapping、resolveAndBuildVariantSkcs、assignBuiltVariantSkcs、BuildSkcsFromAIMapping、prepareAndBuildVariantSkcs、buildSkcsFromPreparedAIMapping、resolveSkcBuildSpecs、buildMultipleSkcs、buildSingleSkc、CreateDefaultSkc、resolveDefaultBuildProduct、resolveAndBuildDefaultSkc、prepareAndBuildDefaultSkc、buildDefaultSkcFromPreparedMapping、selectDefaultAIMapping、buildDefaultSkuFromAIMapping、resolveDefaultAIMapping、prepareDefaultAIMappingForBuild 共 19 方法；只去 skcBuilder/itemBuilder 两字段和对应 ctor 参数/赋值、仅因此失用的 template/models import |
| 同一 NewSkuVariantProcessor | 最小内部接线收窄 | 保留函数名/返回类型，参数已收窄为 logger、aiClient、specHandler、specResolver；保持原 mappingProcessor 初始化，不新增默认 provider、PriceHandler、builder 或 spec adapter。全仓无外部 Go 调用；这不是公共领域合同变更。若刷新发现外部使用，停止此裁剪 |
| variant_processor.go 的其它方法/字段 | 保留 | GenerateAISkuMapping、100 变体/20 批上限、分批选择、validateAIMappingSpecs、resolveAIMappingSpecIDs、prepareAIMappingForBuild、ensureAIMappingMatchesVariants、normalizeAIMappingForBuild、生成结果归一化/维度统一等原实现不变；名称含 ForBuild 不等于构建专属，仍承载保留映射校验/解析能力 |
| sku/convert.go | 仅 RETIRE SkuBuilder.marshalWithoutHTMLEscape | 只删此 receiver 方法与因此失用的 jsonx import；convertSpecInfos 全字段转换原样保留。SkuVariantProcessor 同名 JSON helper 及 jsonx 原能力不动 |
| sku/spec_handler.go | 只 RETIRE 专属组合 helper | 删除 SpecDimension/SpecValue 两类型及 collectNonColorSpecsForColor、generateNonColorSpecCombinations、generateCartesianProduct、createSpecCombinationKey、createNonColorSpecKey、createSpecCombinationKeyFromSpecs、isColorSpec 七方法；保留 SkuSpecHandler/ctor、IsSizeSpec、convertUserInputSpecsToGoodsSpecProperties、ValidateSpecs、CreateDefaultSpec 全部实现/所需 import |
| marketplace/temu/publishing 私有规则与测试 | EXTRACT §4.1 三项值行为 | 复用现有 submissionSpec；只用标准库/现有 owner，不接旧 DTO、模型或执行链，不新增入口/状态 |
| 八个 ai_mapping* 文件、mapping_processor.go | 原样保留 | AIBatchInput/TotalBatches/BatchVariants、AIMappingInput、prompt/批次/parse/响应校准/限数/VariantAttributes/helper 不裁剪；仍在 sku 包，不新增当前调用 |
| PriceHandler/ProfitRuleAPI、PriceManager/PriceCalculator、spec/image/property/AI/text/rules/template/category、skugen、平台 API/DTO/re-export | 原 owner 原样保留 | builder 专属 QuerySpecID 适配退出不删除 QuerySpec API/SpecQueryAPI；没有新 spec 实现、默认客户端或 fallback。其它同名构建器/PriceCalculator 不匹配 |
| context / handlerbase | 原合同保留 | TemuTaskContext/TemuProduct、SKUBuildRuntime/BuildSKUBuildRuntime/SaveAsinSkuMapping、AI 响应/PublishState/StoreInfo/PriceQueryResponse 及 handlerbase.SkuBuilder/SpecHandler 不动；保留接口不表示仍有旧构建实现或当前执行入口 |
| 当前采集/Store 观察、共享来源/缓存、两 SHEIN 属性、独立 SHEIN 登录 worker/Store sync | 保留 | 原身份、授权、持久化/恢复 owner 不动，不扩大环境权限 |

Legacy decision: **EXTRACT 三项已有值行为；RETIRE 六文件和混合构建专属闭包**。
Reusable behavior: publishing 三项私有值规则；AI/SKU 映射、价格/规格/图像/平台等已有 owner 保留。
Current owner: TEMU publishing 规则；原 SKU AI/映射、PriceHandler、SpecResolver/平台 API、Product/Store/Observation 与独立 SHEIN 各自负责。
Cutover/deletion condition: 用户确认 §5/§6/§6.1/§7；生产修改前 Issue Reuse Existing Architecture / Ready；消费者刷新、值规则和原 guard RED→GREEN、准确实施 HEAD 的必要 CI/最终独立检查。源码删除不等于平台/运行态退出。

## 6. 验证与交付

原设计阶段只提交本文与架构索引；其 CI/review 不是实施或用户验收。确认后已完成三值规则 TDD RED→GREEN；原退休 guard 在六文件仍存在时准确 RED（19 构建方法、7 helper、两字段，无保留断言失败），精确删除后 GREEN。准确实现 HEAD 的保留回归/CI/最终检查维护于 PR，不把静态图当真实平台验收。

用户确认后采用以下最小实施验证范围：

1. 刷新六文件、19 方法、两字段、组合 helper/类型及构造/导入/注册和当前命令依赖，确认没有当前消费者或兼容义务。
2. 三项值规则 TDD RED→GREEN：规格首次 parent 去重的稳定顺序/空 key/nil/空/单项/不变输入；精确 TEMP_、大小写/位置/ParentSpecID 不新增判断，先去重再检测，被淘汰重复项含 TEMP_ 不新增失败；classification=1 的 1/1 及其它 classification 原值透传。不新增价格、库存或平台要求。
3. 原退休 guard 先准确 RED，再删源码 GREEN；限 TEMU sku 的目标类型、constructor、19 receiver 方法和 helper，不误封 handlerbase 接口、平台 DTO 或 SHEIN 同名实现。
4. AI 保留回归使用最小本地值与 stub，构造收窄后的原映射 owner：验证规格/空模板、批次值、映射数量/维度统一与错误分支；只在受影响边界补测试，不新建 runner/验收框架、不调用真实 AI/API。保留字段/方法、convertSpecInfos 及用户规格转换须证明未被误删。
5. 保留 publishing/productpolicy/product/rules 规则与原 import/provider/logger/consumer/ceiling/命令图 guards；sku/context/handlerbase/spec/image/ai/property/api 适用编译。current-application 与 shein-login-worker 只编译，不运行 main。完整必需检查交给稳定实施候选原 CI，不人工重复全量。

### 6.1 原 guard、清单、文档和 workflow 的准确处置

| 工件 | 本次实施改动 | 不变的防线 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 扩展原 TestListingKitTaskBackendSourcesAreRetired，增加六文件缺席与 sku 限定的类型/ctor/19 方法/专属 helper AST 禁令；保留矩阵只移除 builder.go 的构造字符串行与 item_builder.go 的价格调用字符串行，转为原 PriceHandler/AI 映射/规则/API/共享合同断言 | 不是删除 PriceHandler 保留断言；原其它断言、31 命令与独立 worker guards 全保留。禁止旧构建回归，不禁止 sku 整包或 handlerbase 同名接口 |
| tests/import_boundaries_test.go | TestTEMUOpenAIImportsStayAllowlisted 仅移除 sku/builder.go 一条消失路径豁免（原九条→八条） | ai_mapping_single_processor.go、variant_processor.go 和其它六条均保留；root/扫描/deny/helper 不动。management product 的唯一 price_handler.go 豁免完全不动；无目录缺失适配或 skip |
| docs/refactoring/phase2-runtime-inventory.md 当前 consumer register | integration/openai→temu/sku 仍由 AI 文件消费，保留该行；core/logger 的 SKU 唯一直接 import 在 parallel_builder.go，删除此文件后仅移除 core/logger→temu/sku 一行 | 不删整个 SKU/OpenAI consumer；logrus 与 core/logger 不是同一导入。dated inventory、closure ceiling 与其它行不变 |
| code-health-decisions.md 的 SKU/product 行 | 精确注明六文件/混合构建闭包退休、映射与 PriceHandler 保留，旧 runtime 构建依据仅历史；原 dated 21/20 declarations 留存 | 不把 SKU/product 整包写为 removed，不顺带清理其它历史行或提高 ceiling |
| external-client-boundary-inventory.md 当前 TEMU SKU/OpenAI 描述 | 只注明 builder 直接 OpenAI import/一个豁免退出，AI mapping/provider seams 仍在 | 其它树、guard 覆盖、provider 计量/安全 owner 不改，不能宣称全部 OpenAI 依赖已退休 |
| Legacy register、本文/架构索引、publishing README 与直接冲突的前批当前说明 | 已标 source-retired 和三项值规则 owner，精确替代前批对构建 caller 的保留陈述 | 不重写历史设计/审批证据；AI/价格/规格/图像/平台等保留矩阵不被笼统覆盖 |
| 两套及其它 workflow、.golangci.yml、依赖、必需/安全检查、保护/ceiling | **零修改** | 不删除 job/编译目标、不减少扫描范围、不改保护规则、不因专属 caller 删除而降低 provider inventory 防线 |

sku 目录仍存在，不需要目录缺席适配。若原 CI 发现同范围额外硬编码引用，先定位、分类并核对 sibling，只校准准确已退出目标，不能扩大删除、删 gate、改必需检查或加通用 skip。

## 7. 确认与实施停止点

1. #771 原设计已交付；用户确认后在同一主要分支/PR继续实施。#29/#301保持 OPEN，不将源码退休作为内部试用新增前置。
2. 用户已明确确认冻结 3ceea910b 的 §5、§6、§6.1、§7；生产修改前已登记 Reuse Existing Architecture、冻结设计、owner/不变量/验证与 Ready。设计确认不替代最终准确 HEAD 合并授权。
3. 实施开工同步一次 main，稳定最终候选再核对实际组合；发现新消费者/具体兼容义务、新有效规则/公共领域合同、身份/tenant、持久化/状态/事务或平台/AI 副作用协议变化，停止对应部分报告，不增 wrapper/默认客户端或临时新架构。
4. 三项规则/原 guard RED→GREEN、精确拆除与保留回归、准确实施 HEAD CI/最终独立检查后交付 In Review；设计 CI/review 不冒充实施或用户验收。
5. 合并仍需用户准确最终 HEAD 单独授权及正常 expected-head/保护。设计和后续实施均不授权 Issue 关闭、部署、真实 provider/实例/凭据/数据操作；平台/运行/产品与用户验收分别保持 NOT_RUN。
