# 旧 TEMU 变体读取、后处理与缓存编排退休边界

后续用户确认的 [#812 完整文本与要点闭包边界](legacy-temu-text-bullet-retirement-boundary.md) 已退休 TextProcessor/CleanProductTitle/专属空白 helper、旧文本与要点编排，以及失去最后消费者的 NormalizeProductSubmissionName/全部专属 regexp/测试。下文当批保留、消费者和计数为历史证据；仅这些具体对象被窄替代，平台 CheckText API/DTO、PriceHandler、AI/prompt 和其它有效能力保持，不据旧措辞恢复已退休代码。

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

- Status: **USER_CONFIRMED / SOURCE_RETIRED**。用户已确认 #751 冻结设计 `70c60af893aee799f4618190f7ca99e8830676ac` 的 §5、§6、§6.1、§7 并授权同一 PR 实施；四文件与专属 helper 已源码退休。准确实现 HEAD、适用 CI、独立最终检查及合并决定维护于 PR，不在本文滚动。
- 调查基线：`main @ b296819751ca8a6b8e20344eb8f21b34bfc5c640`；实施开工同步 `main @ ac7957b81d10f7849c7912b3f5aa0563065c6e1d`。#752 的模型证据读取/TLS 改动不重叠本轮删除范围。
- 当前 **Design Basis: Reuse Existing Architecture；Admission: Ready**。生产修改前已在 #29/#301 登记并读回；原设计 N/A、CI/复核只是历史设计证据，不代替实施验证或授权合并/真实操作。
- 保留当前采集、Store 观察同步、有效平台 API/DTO、规则与共享合同、两种 SHEIN 属性、PriceManager/PriceCalculator、独立 SHEIN 登录 worker 及其 Store 同步。两套及其它 workflow、必需/安全检查、保护/ceiling 零改动；不操作实例或数据，不增加内部试用前置。

## 1. 用户结果、范围与非目标

使用者是维护当前内部试用、采集、平台/Store 能力与独立登录的维护者。交付是闲置旧 Task 变体读取、批量获取、后处理和缓存闭包的实际消费者、持久化/平台副作用 owner、删除及有效规则保留矩阵，防止后续接线重新引入旧 Task owner。此维护不阻塞当前用户操作，不改变当前入口、操作步骤或试用前置。

范围为 `internal/temu/sku/` 中四个生产文件：`variant_json_data_handler.go`、`parallel_variant_handler.go`、`cache_variants_handler.go`、`variant_data_processor.go`，基线共 **628 行**；另仅覆盖混合文件 `ai_mapping_utils_helper.go` 的私有 `extractAsinListFromMap` 函数及其专属注释。没有查到可随之整删的专属 DTO 或测试。`sku`、`context`、`product`、`sourceproduct`、`crawler/fetcher`、`pipeline` 均不是整包删除范围。

不新建变体服务、并发引擎、缓存策略、主变体选择合同、刊登入口、状态机、公共领域合同、UI、provider 调用或验收工具；复用已有 Go/原 guard 与共享实现，不引入依赖。#692 Store 安装等其它 owner 的范围不接管。#748 合入其内部试用分支，不是本轮 main 基线或本轮交付。

不执行命令 main、目标构造器、平台 Query/Preview/DryRun、采集、缓存或 schema 工具；不读取真实配置、凭据、DB/Redis、Profile/browser、历史日志/JSON；不启停、部署、迁移、补写、清理、重放或删除实例/数据。静态无消费者不等于运行实例、历史缓存或在途资产清零。

## 2. Design Basis、Authority 与不变量

适用依据：

- [全新系统产品基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)：只有 EXTRACT / RETIRE，无 legacy wrapper/fallback、旧 ID 映射、双读双写或第二事实源。
- 已确认的 [CLI](legacy-listing-cli-retirement-boundary.md)、[control-plane](legacy-listing-control-plane-retirement-boundary.md)、[数字执行链](legacy-numeric-task-execution-retirement-boundary.md)、[数字持久化](legacy-numeric-task-persistence-retirement-boundary.md)、[TEMU 保存/提交](legacy-temu-listing-submit-retirement-boundary.md) 和 [前置 handler](legacy-temu-preflight-retirement-boundary.md)：旧执行入口已退出；本轮只接续此前保留的闲置变体编排，不推翻其它保留项。
- [TEMU publishing owner](../../internal/marketplace/temu/publishing/README.md)：平台规则和 payload shaping 继续归原 owner；已有 `internal/shared/strx.CleanProductTitle` 复用原实现，不在旧 Task 外再包装或复制一份算法。
- [独立 worker 身份/持久化](shein-worker-identity-persistence-boundary.md)、[当前 Store](store-center-current-application-v1.md)、[observations](store-center-platform-observations-v1.md)、[准备接线](store-observations-preparation-runtime.md)、[当前 SHEIN](my-supply-chain-shein-v1.md)：当前事实、身份、授权与 retry/UNKNOWN 不转交。Figma N/A，无 UI/IA 修改。

原设计 N/A 仅覆盖文档调查。本次实施采用 **Reuse Existing Architecture**：用户已确认冻结 §5、§6、§6.1、§7；生产修改前 #29/#301 已写明冻结设计、用户结果、上述引用、原 owner/调用路径、不变量、验证、唯一 Writer 及 **Ready**。仅删除无装配消费者的四文件与专属函数，不新增生产函数/类型/公共接口或当前入口；既有标题清理已有正确共享 owner，无须新增 EXTRACT 实现。

Must：§5 保留能力不减；当前 tenant/权限/凭据、共享来源字段、平台/Store/持久化合同及必需/安全检查不变；不改变事实/schema、状态机、事务、幂等或 retry/UNKNOWN/recovery owner。Should：原 guard 与当前源码盘点一致。Out of Scope 如 §1；无新增 Accepted Risk。Threat Model 限误删有效规则/同名共享类型、混淆缓存和 Product fact owner、复活旧 Task，以及把源码退出误写为运行态验收。

若发现当前消费者、当前外部可观察兼容义务、持久化 owner 或保留边界与映射不符，停止对应删除并报告具体证据；不得自行加兼容路径。若实际需要新增或改变高风险边界，则重新判断 Independent Architecture 准入，不在实施中临时发明合同。

## 3. 实际消费者、注入与调用路径

基线全仓 Go 搜索中，四个目标类型/构造器仅在各自文件内定义和使用，无外部构造、直接实例化、注册或装配调用。`go list ./cmd/...` 共 **31 个当前命令**；其 `go list -deps -f '{{.ImportPath}}' ./cmd/...` 依赖并集中 `internal/temu/sku` 命中 **0**，实施刷新结果相同。未运行 main；这是源码/静态图证据，不是运行态或真实平台验收。

| 合同 → 实现 | 实际路径与消费者 | 处置 |
| --- | --- | --- |
| ProductReader → VariantJsonDataHandler | 未调用的 constructor 注入 Reader；Task/TemuProduct 检查 → ASIN 列表 → FetchRequest → FetchVariants → SetVariants → 标题/描述回填 | 退休 Task 适配；保留 Reader、批量获取及原数据 owner |
| ProductReader → ParallelVariantHandler | 未调用的 constructor 注入 Reader；perf tracker 包裹同类 Task 路径；fetchVariantsParallel 仅调用一次 FetchVariants，没有独立 goroutine pool | 退休该编排；不新建替代并发实现，perf 共享包保留 |
| ProductCache → CacheVariantsHandler | 未调用的 constructor 注入 Cache；从 pipeline.AmazonContext 取 variants，再从 Task 组请求，CacheVariants 错误仅警告并返回 nil | 退休 Task 装配和吞错策略；共享缓存规则/IO 保留 |
| VariantProcessor | NewVariantProcessor 无调用；HandleTemu 只委托 ProcessSingleProduct；另有无消费者的 ProcessVariantData | 退休孤立 in-memory 投影，不混同仍保留的 SkuVariantProcessor |
| extractAsinListFromMap | 仅被前两个 handler 的 getAsinListFromMap 调用；空 map 返回空 slice，否则迭代 map key，不排序/过滤/规范化 | 删除两个 caller 后只删此函数；混合文件其余方法与 import 不随之删除 |

两个 Reader handler 均按 `AsinSkuMap → AmazonProduct.Variations（跳过精确空 ASIN）→ VariantAsins` 取值；map 顺序未定义。VariantJsonDataHandler 规范化的 mainProductAsin 仅用于日志，不据此选择/剔除主项。二者均有旧 100 条 NONRETRYABLE 上限，Task.Platform 被填入源请求，且使用 context.Background，不能宣称具备当前取消/deadline 或源/目标平台隔离合同。

混合文件的 `getProductDimensions/getItemWeight/getProductDetailField/buildAIVariant` 仍被 `ai_mapping_single_processor.go` / 其它保留方法消费，`marshalWithoutHTMLEscape` 同样不在本轮删除范围。`SkuBuilder → NewSkuVariantProcessor → SKU/AI mapping` 的现有源码路径、AI API/DTO、映射/规格规则和输入保真均保留；不因为名字相近退休 `variant_processor.go`。

共享 Reader/Cache 仍被 `internal/shein/productdata/{variant_data,fetch_and_cache_variants}.go` 使用；SHEIN 有同名 VariantJsonDataHandler，且其包函数 GetVariantByAsinFromVariants 由 `shein/publish/mapping_helper.go` 使用。这些均保留。`pkg/perf` 还被 SHEIN productdata 与 TEMU image 消费，不整包删除或搬迁。

## 4. 持久化、平台副作用与有效规则 owner

| 行为 | 实际 owner / 语义 | 本轮边界 |
| --- | --- | --- |
| 批量变体获取 | ProductReader → sourceproduct.ProductFetcher 或 remote fetcher。local FetchVariants 按 VariantFetchRequest 委托 FetchProduct；可读缓存/触发采集、网络，且逐项 CacheProduct；重定向 ASIN 以副本保留请求身份，失败项可跳过 | Reader 并非保证零副作用的纯 DB 读；连删除前调查也不调用它。保留来源字段、批量/缓存/错误策略及既有测试，不复制到新 Task/服务 |
| 批量缓存 | ProductCache → ProductFetcher/remote → sourceproduct.CacheManager → RawJsonDataClient；CacheManager 跳过 nil、按 variant.Asin 组 ProductID，决定 freshness/skip/save，并保留部分失败语义 | 原 Raw JSON API/仓库/模型才是共享持久化 owner；不转给 handler、当前 Product、Store 或 worker。不新增事务、key、schema、补偿/恢复或数据动作 |
| 旧 Task projection | VariantJsonDataHandler 设置 CleanedTitle/ProductDescription，清理非 nil variants 的 Title，并只取 variants[0]；ParallelVariantHandler 拷贝原始首项 title/description；VariantProcessor 写 TemuProduct.GoodsName/GoodsDesc 并可能原地清理 variant.Title | 旧 in-memory 写入不是 canonical Product 事实。首项为 nil 时也不会寻找首个非 nil 项；不抽取/修补这套主项选择、原地修改、空列表回退或 Task 门控 |
| 标题清理 | internal/shared/strx.CleanProductTitle 已拥有现有去 emoji/中文/特殊字符、空白清理行为，仍被 temu/rules/text_processor.go 使用；当前 marketplace 文本 shaping 原路径保留 | 保留实现、消费者与有效测试，不新增 wrapper，不把旧处理循环或修改 source model 的副作用作为有效规则复制 |
| 当前 Product/Store/独立 worker | 当前 Product facts、Store observation/sync、平台 API/授权/凭据、独立 SHEIN 登录 session/Profile/browser 与健康/关闭继续原 owner | 不增加接线/调用、不转移权限或 durable assets、不判定历史操作结果、不释放锁/预占、不 replay UNKNOWN |

沿用前置 handler 设计 §4 已记录的共享旧合同限制：Raw JSON 查询键为 platform/product/region，不带 tenant/store；旧 GORM API 的 context/身份合同不能等同当前 Organization 授权。保留共享层不证明隔离验收 PASS，不增加其当前私有数据消费者；若调查发现实际当前 Must 缺陷，报告具体用户路径与后果，再决定最小修复，不自动扩为 IAM/缓存改造或试用前置。

目标四组件没有自己的 durable 状态机、事务/outbox、恢复记录或幂等键；缓存和采集 IO 已由共享 owner 执行。删除这些无装配 caller 不迁移数据、不认领历史运行态；源码证据无法判定旧实例是否仍运行。

## 5. 精确删除与有效规则保留矩阵

以下为用户已确认的精确矩阵，**四文件与混合文件的单个专属函数已源码退休**。基线调用映射保留用于解释边界，不代表目标仍存在。

| 对象 | 决策 | 删除/保留条件 |
| --- | --- | --- |
| internal/temu/sku/variant_json_data_handler.go（211 行） | RETIRE 全文件 | 移除旧 Task ASIN 选择、FetchRequest 装配、FetchVariants 调用、SetVariants、原地清理/首项回填、孤立 lookup 和 Shutdown；不删 SHEIN 同名组件/lookup |
| internal/temu/sku/parallel_variant_handler.go（227 行） | RETIRE 全文件 | 移除旧 Task 批量请求/性能日志/首项投影；不保留旧 100 条上限、NONRETRYABLE 字符串或 Background 接线，不删除 perf 共享包 |
| internal/temu/sku/cache_variants_handler.go（76 行） | RETIRE 全文件 | 退出 Task→CacheVariants 和失败继续策略；不改共享缓存部分成功/失败/freshness/upsert 原语义 |
| internal/temu/sku/variant_data_processor.go（114 行） | RETIRE 全文件 | 仅孤立 VariantProcessor；不删 variant_processor.go 或 SkuVariantProcessor，也不把旧 TemuProduct 投影包装为新服务 |
| ai_mapping_utils_helper.go 中 extractAsinListFromMap 及专属注释 | RETIRE 单函数 | 仅两个旧 caller 全部消失且无新消费者后删除；文件、其它五个方法、SkuVariantProcessor、jsonx、model/context 和仍需 import 保留 |
| strx.CleanProductTitle、temu/rules/text_processor.go、marketplace/temu/publishing 现有文本/校验/SKU 规则 | 保留既有 owner | 清理算法已在共享层，无新增 EXTRACT 代码；不改规则、格式、测试，不复制 source model 原地 mutation 或 map/首项策略 |
| ProductReader/ProductCache/Stats、Factory/local/remote、sourceproduct FetchRequest/VariantFetchRequest、CacheManager/SourceFetcher/DataParser/RawJsonDataClient、Raw JSON API/仓库/模型 | 保留合同/实现/源码消费者 | 不删 FetchVariants/CacheVariants 或缓存能力，不新增 consumer/wrapper；来源身份、现有缓存/错误规则及测试保留 |
| 其余 SKU builder/mapping/spec/filter/product/context/pipeline、共享 Mapping 与 Store owner 事务、安全测试 | 保留，非删除闭包 | 不删 context 字段/方法/DTO/接口断言，不删表、索引、旧资产或数据 |
| SHEIN productdata 同名 handler/lookup、独立登录 worker/Store sync；平台 API/同名 DTO、EnrichedSkuInfo/EnrichedSkcInfo、PriceManager/PriceCalculator、当前采集/Store 观察 | 全部保留 | 不按类型名或 variant/cache 关键词全仓封禁；不新增内部试用前置 |

Legacy decision: **RETIRE 四个闲置旧 Task 组件和一个专属函数**。
Reusable behavior: 标题清理、共享批量读取/缓存与 SKU 映射仍有价值，均已有正确 owner，保留原实现；不新增抽取函数或公共合同。
Current owner: strx/现有 marketplace 文本规则、sourceproduct/fetcher/Raw JSON、SKU mapping、当前 Product/Store/worker 各保持原 owner。
Cutover/deletion condition: 用户确认冻结矩阵、Issue 实施 Ready 已满足；刷新无新消费者，原 guard 已先捕获目标存在而 RED，再精确删除 GREEN。保留包编译、既有必要回归、准确候选 CI 与最终检查按 §6 在 PR 记录。源码删除不授权运行态或数据退出。

## 6. 验证与交付

原设计阶段只提交本文与架构索引，其 diff/相对链接、静态消费者/命令图、原适用 PR CI 和一次独立设计检查是历史设计证据，不替代本次实现或真实使用 PASS。本次仅扩展原 guard 并精确删除，不新建验收流程或工具。

本次范围内的最小验证：

1. 刷新四构造器、类型、专属 helper 与保留共享调用，复核 31 命令的实际依赖；如消费者变化，停止对应删除并报告。
2. 扩展原退休 guard：四路径缺席、限定 `internal/temu/sku` 的四个准确类型/构造器及专属 helper 缺席。先确认现有源码使新增断言 RED，再删除 GREEN；通过原 Go AST 能力精确区分 VariantProcessor 与 SkuVariantProcessor，不能子串封禁或全仓禁止 SHEIN 同名类型。
3. 同 guard 保留混合 helper 文件/其它方法、SkuVariantProcessor/builder、strx 原函数、Reader/Cache 与 FetchVariants/CacheVariants、SHEIN 同名 lookup/handler、平台/共享合同和独立 worker 的必要存在/原位检查。不改保留业务规则，因此不新增一套主变体选择、缓存失败或并发测试框架。
4. 适用编译保留的 TEMU sku/context/product/filter/rules/API、共享 fetcher/sourceproduct、SHEIN productdata 与独立 worker；复用现有标题/文本、来源身份/请求字段、缓存及 owner/tenant/Store 安全回归和原命令编译目标。没有直接 CleanProductTitle 专属测试不等于可删除算法；仅保留检查和原有效回归，不为无行为变更复制实现测试。
5. 完整必需 CI 留给稳定实现候选；一次准确实现 HEAD 独立最终检查。开发测试、PR CI、合并后匹配 main CI、实例/平台/用户验收分别记录，后者本批 NOT_RUN；不宣称生产可用或全部 legacy 清零。

### 6.1 原 guard、文档和 workflow 精确处置

| 工件 | 实施处置 | 保留边界 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 扩展原 TestListingKitTaskBackendSourcesAreRetired 的四路径/准确旧 SKU 符号缺席和 §5 保留项；复用现有 AST 能力 | 原 TestCurrentCommandsDoNotCompileLegacyBusinessBackends、独立 worker、安全断言保留，不 skip 缺文件或删原 gate |
| tests/import_boundaries_test.go、integration_provider_logger_contract_test.go、.golangci.yml | 基线与实施刷新均未见四目标路径或专属 helper 的硬编码 allowlist，**本轮零改动** | 不类推上批 exists_check_handler 的 allowlist 删除；ai_mapping_handler 原豁免、其它 roots/deny/graph/ceiling 保留，不臆造消失条目 |
| Legacy register、phase2-runtime-inventory 当前状态、本文/架构索引与直接当前说明 | 按用户确认矩阵校准四组件/专属函数 source-retired 和保留规则 owner；前置设计的变体消费者说明标为其历史基线 | SKU 包仍有 logger/openai 源码消费者，不删除整个消费者行；历史清单/图/证据及 ceiling 不改，不将源码退出宣称为运行态清零 |
| ci.yml / commercial-readiness.yml / 其它 workflow、保护规则 | **零改动** | 原必需/安全检查、race/编译目标、分类/触发/权限/并发及 Required CI Gate 保留；原 SKIPPED 不改写 PASS |

## 7. 确认与实施停止点

1. 原设计以一个 Draft PR 交付，准确设计 HEAD 的适用 CI 与独立检查通过；证据保持历史。#29/#301 保持 OPEN，原正文历史保留。
2. **用户已确认冻结设计并授权在原分支/#751 继续实施**；生产修改前 Issue 已登记 Reuse Existing Architecture / Ready、冻结设计/确认记录、唯一 Writer 和 §2 不变量。此实施确认不授权合并、部署或真实操作。
3. 开工已同步一次实际 main；形成最终候选前再核对一次，不持续机械 merge。仅解决真实组合风险；新消费者/公共合同/持久化/权限/状态/副作用变化停止对应项并报告，不擅自扩大删除。
4. 按矩阵实施、原 guard RED→GREEN、必要保留回归、准确候选 CI 和独立最终检查后交付 **In Review / 待用户确认**。设计证据可追溯但不冒充实现证据；不用未变化代码重复审核扩展流程。
5. 合并仍需准确 HEAD 的单独用户授权；遵守 expected-head 和保护，无管理员绕过。此设计/实施均不授权部署、Issue 关闭、真实平台/实例/凭据/数据操作，也不增加内部试用前置。
