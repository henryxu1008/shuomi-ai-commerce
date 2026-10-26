# #301 旧全局 Amazon 结果合同与 SHEIN 孤立属性 DTO 完整退休设计

> **USER_CONFIRMED / Independent Architecture / IMPLEMENTATION_READY / SOURCE_IMPLEMENTED / IN_REVIEW**。用户已确认冻结 §1–§7（含 §4.2、§6.1）；Issue/PR 的 IMPLEMENTATION_READY / Ready 已在正式修改前写入并读回。同一 #964 完整53路径已实施：17真实值/5原方法归 Amazon capturedata 纯叶子、36消费者改接、旧全局 model 根及三个孤立 SHEIN DTO 退出，无 alias。设计准确 HEAD f496b920b 的原 CI40301812628 与独立只读设计复核 PASS 仅作设计证据；源码必要自检、准确候选 CI 与最终交付复核见 [PR #964](https://github.com/qq550723504/shuomi-ai-commerce/pull/964)，分别记录，不冒充运行验收。#301 OPEN，合并/部署/关单另授权，运行/产品验收 NOT_RUN。

稳定设计候选同步实际 main `6199063811dbb7904aa678735c1b7bb660b8d186`（#963）。唯一增量为另一 owner 的 Amazon capture 扩展三份TS、四份测试与其设计文档，共8文件；本批53路径/全部Go/AGENTS/workflow/依赖和架构索引/register零交集或零字节差异。原3184 Go/字段/消费者调查可复用，最新组合由本设计准确HEAD的原CI验证；不接管该扩展实现。

> 用户已批准的准确命名修订：原CI40305890688证明仓库原artifact检查保留整个路径段`result`；仅将本批纯叶子命名改为`capturedata`，包括两个新文件、包名、36import和本轮guard/文档引用。领域归属、全部字段/五方法/JSON/错误/行为不变，不修改artifact检查或增白名单；完整清单仍53路径。用户已明确批准准确提案 #964/6413700926 后应用本修订；领域边界及全部原检查保持。

## 1. 这次清理什么、保留什么

同一批完成两组，不按目录、文件或提交机械拆 PR：

1. **退出旧全局 `internal/model` 根。** 该根只剩 `amazon_product.go`，实际合同全部服务原 Amazon 浏览器采集与调试。将17个真实值类型及5个现有纯方法放入 Amazon 域内的 `internal/crawler/amazon/capturedata` 叶子 owner，36个当前 import 文件一次切换，不保留旧包、type alias、re-export、转发或第二模型。
2. **删除 SHEIN 三个孤立属性 DTO。** 只退 `api/attribute` 的 `AttributeInfo`、`AttributeValue`、`AttributeValueGroup`；保留该文件内实际使用的 `CustomAttributeRelation` 原字段与 JSON。

使用者是维护原浏览器采集、批量结果、调试 JSON 和当前 SHEIN 离线合同的开发者。结果是原操作继续使用同样的采集/结果/错误/JSON，依赖不再穿过全局 catch-all；不是新增采集、平台提交或试用验收前置。当前业务来源入口仍是 `currentapplication → integration/acquisition/amazon → Product/dataacquisition → Sourcing/Catalog/SRC`，不改成走原浏览器。

本批源码阶段完整 **53 唯一路径**：36个当前 import 文件、1旧模型、1混合属性文件、2新叶子文件、1原检查文件、5当前文档与7历史顶注。设计阶段只有三文档；本次已按确认范围实施。完整清单见 §5.3。

## 2. Design Basis、Authority 与前批 KEEP 替代

**Independent Architecture**：改变17个公开 Go 值/错误类型的包归属，跨 root/browser/extractor/debug 消费边界且超过30路径。领域数据的持久化事实、状态/权限/恢复 owner 不变；本设计只覆盖本批的值合同 owner 与当前消费者切换，不重复全局架构。UI/IA/交互未变化，Figma Authority **N/A**。

依据：[Greenfield](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[目标架构](project-target-architecture.md)、[项目边界](project-boundaries.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、[来源业务闭环](source-business-closure-v1.md)、[Amazon 当前路径](amazon-crawler-runtime-flow.md)和用户本轮范围决定。复用 Go `encoding/json`、`time`、`fmt` 与原纯算法，不建设新 DTO 框架、序列化器或兼容层。

Must：两个闭包完整切换/删除；所有当前字段/tag/顺序/数值/空值、五个纯方法及错误 As/Unwrap/文案保持；实际采集/原浏览器/debug、质量重试、真实 PoolManager deadline/取消/释放/重建、当前 JSON/hash/SourceIdentity/lineage/事实 owner、独立 SHEIN 登录与 Store 同步保持。Should：最小叶子 owner、现有检查/原测试和合成值 fixture。Out of Scope：新业务/UI/provider、采集/排序/错误分类算法、schema/迁移/历史 JSON 重写、身份/租户/计费/UNKNOWN/恢复、workflow/任何 inventory/依赖/保护修改、既有实例/数据/凭据/文件或进程操作。无新增 Accepted Risk；物理运行态 UNKNOWN，不探查。Threat Model 仅为误删有效值/错误语义、wire 漂移、依赖循环、旧 owner 回流或放宽原检查，不扩全局安全目标。

| 前批依据 | 本次准确替代 | 继续保留 |
| --- | --- | --- |
| [#822 Amazon 刊登](legacy-amazon-listing-closure-retirement-boundary.md)、[#824 数字支撑](legacy-numeric-listing-support-closure-retirement-boundary.md) | `internal/model` / `amazon_product.go` 原 owner 的 KEEP | 同一17值/5方法和原采集/结果/批量合同；旧 Task 六文件及原禁令仍退 |
| [#901 Amazon 来源适配](legacy-amazon-source-adapter-closure-retirement-boundary.md)、[#948 采集外观](legacy-crawler-facade-diagnostics-closure-retirement-boundary.md) | 原全局 model import 不动的 KEEP；#948 guard 的一个保留 owner 身份 | 浏览器 Processor/提取器/variations 转换/有效池大小、取消/质量/代理/日志/debug 的原行为及断言 |
| [#926 旧平台 API](legacy-platform-api-closure-retirement-boundary.md) | 全局 Amazon 模型路径；attribute 的三个孤立 DTO KEEP 与原 requiredByFile 三项 | CustomAttributeRelation、当前 SHEIN product/warehouse/other DTO、SessionTransport 与三个登录/Store 读取端点 |
| [#909 旧 Product 策略](legacy-product-policy-closure-retirement-boundary.md) | 共享 `internal/model.Product` 路径不动的旧 KEEP | 同一 Product 值/JSON归新叶子，不恢复旧 Product policy 机制 |
| [模块目标映射](../refactoring/module-target-mapping.md) | 全局 model catch-all 的现状 | owning-domain 方向、其它历史/计数及全部当前业务 owner |

仅上述准确范围被替代。7份旧依据在本次实施中只加顶注，原完整 Git blob/数字/勾选/证据保留；其它旧泛称 KEEP 不成为恢复全局模型的理由。原三文档设计阶段未将未来退休写成完成；当前源码状态由本文顶部及 PR 证据说明。

## 3. 实际消费者与依赖方向

基线3184个 tracked Go 已用标准 Go AST 只读解析，包含测试、未选择编译源及 hack/debug。import 用 `strconv.Unquote` 解码、区分 alias/局部遮蔽；另核对完整类型字段、函数签名/receiver、隐式 JSON 方法及包内闭包，不将签名中的 ProcessFunc 等误判闲置。

| 证据 | 决定 |
| --- | --- |
| 旧 model 36个实际 import 文件＝29生产＋7测试 | 全在 `crawler/amazon` 树及唯一 `hack/debug/test-amazon/main.go`；没有其它平台/当前 Product owner import。36文件全部改接，不机械按子包拆批 |
| api/attribute 四个实际生产 import | 仅选择 CustomAttributeRelation；三个待退 DTO 只在同文件内互相嵌套，没有当前生产/测试/命令/注册消费者 |
| 现有 child variations.VariationValue / Variation | 不能作为结果类型替代：Variation 的 Currency、Image、Attributes字段/tag/omitempty与结果模型不同；该包还包含 parser/Playwright/logging，并非纯值 owner。原转换函数继续存在，只改其结果类型限定 |
| 原 runtime 与原浏览器路径 | 当前 acquisition/amazon 保持自己的固定站点/解析/挑战合同；原 debug 仍 NewAmazonProcessor → Process → Shutdown，批量走 ProcessBatchWithContext → BatchProcessor，不插入新桥 |

新 DAG：`AmazonProcessor / BatchProcessor / InstanceProcessor / browser / extractor / debug → crawler/amazon/capturedata → encoding/json + fmt + time`。新叶子不导入父 amazon、browser、extractor、variations、Product/Store/App/provider/DB/文件/日志或时钟；唯一时间能力是既有 time.Time 值的编解码，不调用 time.Now。把值放进父包或 extractor 会迫使子包依赖运行层并带来循环风险，因此选最小域内叶子；不创建跨平台 common。

新公开 Go 类型身份变为 result owner；所有消费者同批切换，旧类型不作为运行期桥。没有 gob/反射类型名注册、旧 Go 对象持久化或跨版本进程交换的当前消费者证据；不据此推定历史数据迁移义务，未知现存资源不读取。Sourcing 对 crawler/旧model 的现有导入禁令继续原样阻止新业务绕入原浏览器。

Go 包身份（包括 reflect 或标准库错误所携带的 Go 类型元数据）随 owner 正常变化，不承诺旧全限定 Go 类型字符串；JSON 字段/字节与 ProductNotFoundError 的显式文案保持。不能为保持旧 Go 包名新增 alias、注册映射或兼容解析入口。

<details><summary>36个实际 import 文件与准确选择器</summary>

| 文件 | 类别 | 当前选择器（未来仅 owner/限定名改接） |
| --- | --- | --- |
| `hack/debug/test-amazon/main.go` | 生产 | `Product` |
| `internal/crawler/amazon/batch_processor.go` | 生产 | `Product, ProductRequest, ProductResult` |
| `internal/crawler/amazon/batch_processor_test.go` | 原测试 | `Product, ProductRequest, ProductResult` |
| `internal/crawler/amazon/browser/error_detector.go` | 生产 | `ProductNotFoundError` |
| `internal/crawler/amazon/browser/pool_manager.go` | 生产 | `Product` |
| `internal/crawler/amazon/browser/pool_manager_test.go` | 原测试 | `Product` |
| `internal/crawler/amazon/extractor/availability_extractor.go` | 生产 | `Product` |
| `internal/crawler/amazon/extractor/bestseller_extractor.go` | 生产 | `Product` |
| `internal/crawler/amazon/extractor/brand_extractor.go` | 生产 | `Product` |
| `internal/crawler/amazon/extractor/categories_extractor.go` | 生产 | `Product` |
| `internal/crawler/amazon/extractor/delivery_extractor.go` | 生产 | `Product` |
| `internal/crawler/amazon/extractor/description_extractor.go` | 生产 | `Description, Product` |
| `internal/crawler/amazon/extractor/error_detector.go` | 生产 | `ProductNotFoundError` |
| `internal/crawler/amazon/extractor/extractor.go` | 生产 | `Product` |
| `internal/crawler/amazon/extractor/feature_parser_extractor.go` | 生产 | `Product` |
| `internal/crawler/amazon/extractor/features_extractor.go` | 生产 | `Product` |
| `internal/crawler/amazon/extractor/image_extractor.go` | 生产 | `Product` |
| `internal/crawler/amazon/extractor/list_price_extractor.go` | 生产 | `Product` |
| `internal/crawler/amazon/extractor/list_price_extractor_test.go` | 原测试 | `PriceBreakdown, Product` |
| `internal/crawler/amazon/extractor/parent_asin_extractor.go` | 生产 | `Product` |
| `internal/crawler/amazon/extractor/price_extractor.go` | 生产 | `Product` |
| `internal/crawler/amazon/extractor/price_extractor_test.go` | 原测试 | `PriceBreakdown, Product` |
| `internal/crawler/amazon/extractor/product_details_extractor.go` | 生产 | `Product, ProductDetail` |
| `internal/crawler/amazon/extractor/rating_extractor.go` | 生产 | `Product` |
| `internal/crawler/amazon/extractor/seller_extractor.go` | 生产 | `Product` |
| `internal/crawler/amazon/extractor/ships_from_extractor.go` | 生产 | `Product` |
| `internal/crawler/amazon/extractor/size_chart_extractor.go` | 生产 | `Product, SizeChart` |
| `internal/crawler/amazon/extractor/title_extractor.go` | 生产 | `Product` |
| `internal/crawler/amazon/extractor/variations_extractor.go` | 生产 | `Product, Variation, VariationValue` |
| `internal/crawler/amazon/extractor/video_extractor.go` | 生产 | `Product, VideoInfo` |
| `internal/crawler/amazon/instance_processor.go` | 生产 | `NullableTime, Product` |
| `internal/crawler/amazon/instance_processor_test.go` | 原测试 | `Product` |
| `internal/crawler/amazon/processor.go` | 生产 | `Product, ProductRequest, ProductResult` |
| `internal/crawler/amazon/processor_test.go` | 原测试 | `ProductRequest` |
| `internal/crawler/amazon/result_validator.go` | 生产 | `Product` |
| `internal/crawler/amazon/result_validator_test.go` | 原测试 | `Product, Variation` |

</details>

<details><summary>SHEIN 当前四个消费者</summary>

| 文件 | 当前合同 |
| --- | --- |
| `internal/marketplace/shein/draft/local_preview.go` | `CustomAttributeRelation`，零改 |
| `internal/marketplace/shein/offlinepackage/model.go` | `CustomAttributeRelation`，零改 |
| `internal/marketplace/shein/publishing/submit_payload_normalize_policy.go` | `CustomAttributeRelation`，零改 |
| `internal/shein/api/product/model.go` | `CustomAttributeRelation`，零改 |

</details>

## 4. 值、JSON、错误与持久化边界

### 4.1 最小结果 owner 与原字段合同

新 `capturedata/model.go` 定义17个真实 struct，原字段、字段类型、JSON tag/omitempty、声明顺序、指针/嵌入值/集合零值完全保持；只改 package 及本批消费者的 import/`model.` 为 `amazonresult.`。不留旧包 alias、顶层 type alias、重导出或 wrapper，不归并 child variations 的近似类型。

| 类型 | 原字段数 | 合同 |
| --- | --- | --- |
| `BuyboxPrices` | 2 | 全部字段/类型/tag/声明顺序/指针与空值保持 |
| `CustomersKeywords` | 3 | 全部字段/类型/tag/声明顺序/指针与空值保持 |
| `CustomersSay` | 2 | 全部字段/类型/tag/声明顺序/指针与空值保持 |
| `Description` | 3 | 全部字段/类型/tag/声明顺序/指针与空值保持 |
| `InactiveBuyBox` | 1 | 全部字段/类型/tag/声明顺序/指针与空值保持 |
| `NullableTime` | 1 | 全部字段/类型/tag/声明顺序/指针与空值保持 |
| `PriceBreakdown` | 3 | 全部字段/类型/tag/声明顺序/指针与空值保持 |
| `Product` | 71 | 全部字段/类型/tag/声明顺序/指针与空值保持 |
| `ProductDetail` | 2 | 全部字段/类型/tag/声明顺序/指针与空值保持 |
| `ProductNotFoundError` | 3 | 全部字段/类型/tag/声明顺序/指针与空值保持 |
| `ProductRequest` | 2 | 全部字段/类型/tag/声明顺序/指针与空值保持 |
| `ProductResult` | 2 | 全部字段/类型/tag/声明顺序/指针与空值保持 |
| `SizeChart` | 5 | 全部字段/类型/tag/声明顺序/指针与空值保持 |
| `Subcategory` | 2 | 全部字段/类型/tag/声明顺序/指针与空值保持 |
| `Variation` | 4 | 全部字段/类型/tag/声明顺序/指针与空值保持 |
| `VariationValue` | 2 | 全部字段/类型/tag/声明顺序/指针与空值保持 |
| `VideoInfo` | 5 | 全部字段/类型/tag/声明顺序/指针与空值保持 |

`ProductRequest` 的 URL/Zipcode 与 `ProductResult` 的 Product/Error 原无tag形状也保持，不增 tenant、Task、Store 或来源身份字段。Product 的多项可选字段不能因当前 extractor 未填就删除：它们属于这次明确保留的完整结果 JSON，而非三个孤立 SHEIN 属性 DTO。

### 4.2 五个既有纯方法：本轮没有 JSON 行为变化提案

| 原方法 | 精确保留的行为 |
| --- | --- |
| VariationValue.UnmarshalJSON | 原 canonical `variant_name` / `values`，原 `variant name` 读取与 canonical非空优先；原 json.Unmarshal错误/receiver行为不改。局部 `type Alias VariationValue` 是防止递归的局部新定义类型（没有 `=`），不是旧包兼容alias或重导出 |
| NullableTime.UnmarshalJSON | 原 null、空字符串/直接空输入→nil；正常时间沿原encoding/json/time.Time；非法输入仍返回原标准库错误，不加入新接受范围或格式 |
| NullableTime.MarshalJSON | 原nil→JSON null、非nil→原time.Time JSON，保留嵌入Time和timestamp,omitempty组合的实际输出，不擅改成*time.Time导致空字段省略 |
| ProductNotFoundError.Error | 原中文文案、ProductID、Message和Cause组合逐字保持，不替换成FetchError或通用错误 |
| ProductNotFoundError.Unwrap | 原Cause原样返回，errors.Is/As及包装链保持；Browser ErrorDetector的typed判断和PoolManager的原字符串判断均不改 |

这些是本轮用户要求明确的结果值与解码边界，作为当前完整合同 EXTRACT；不是新增旧 Service/Task/ID wrapper、fallback、历史数据读取或迁移。当前生产装配是 typed extractor 填值，debug 的 json.Encoder 输出结果；没有把“未发现生产反序列化入口”伪装成存在外部历史兼容义务。保留既有纯值方法不新建读取通道或第二事实源，当前 wire 的回读不改变源结果。不能在机械 owner 切换里顺带收窄解码或硬化 receiver/错误语义；用户确认本节后按同一合同实施。

与#961不同，本批不增加任何排序/格式化/默认值修复。VariationsExtractor原map遍历、原转换和可选字段行为保持，不能借新类型owner加入属性排序、补字段、统一currency/omitempty或修复旧解析算法。若真实测试证明必须改变原JSON/hash/算法/版本或持久化边界，停止该部分并提供准确提案，不能用更新golden收口。

### 4.3 原事实与副作用 owner 保持

| 边界 | 保持要求 |
| --- | --- |
| 当前 Product/Catalog/SRC/SourceIdentity/lineage/hash与业务保存 | 原 acquisition/dataacquisition/Sourcing 等 owner 零改；不把raw结果移进业务事实或恢复旧mapper/cache |
| 原浏览器实例/PoolManager/BatchProcessor/质量重试/deadline取消 | 36消费者只改import和类型限定，函数体控制流/超时/释放/重建/队列与断言不变；不存在商品仍走原不重建路径 |
| debug JSON文件写入 | saveProductToFile 的 os.Create/Encoder/SetIndent/文件名与操作顺序不改，只变参数类型owner；设计/实现均不执行该入口，不读写已有输出文件 |
| FailureArtifactStore | 原独立metadata/文件副作用owner零改，本轮不读已有artifact或凭据 |
| 组织/授权/计费/官方提交/UNKNOWN/独立SHEIN登录与Store同步 | 原owner/状态机/事实/配置与fence零改，无新表、事务、恢复、provider调用或部署 |

## 5. 删除、保留、抽取矩阵与完整清单

Legacy decision: **EXTRACT + RETIRE**。Reusable behavior：17个原结果值类型、5个纯JSON/错误方法。Current owner：域内纯叶子 `internal/crawler/amazon/capturedata`，原 browser/extractor/processor/debug 仅调用；CustomAttributeRelation留原API owner。Cutover/deletion condition：36消费者全切换、旧model整根/旧import缺席、三个孤立DTO缺席、原值/JSON/错误/当前行为保持、原检查准确转换与必要原CI/最终源码复核满足；不含既有运行资源退休。

### 5.1 准确处置

| 对象 | 处置 | 保留/禁止 |
| --- | --- | --- |
| internal/model/amazon_product.go及整internal/model根 | 17类型/5方法一次抽取到capturedata/model.go后整退旧文件与根 | 已退task/retry/status六文件不恢复；不创建空doc.go、旧alias或跨平台入口 |
| 36原import文件 | 只改import/alias限定与相关类型引用 | 29生产＋7原测试（含独立debug），原算法/fixture/assertion/body不改，不将代码搬到新运行框架 |
| internal/shein/api/attribute/interface.go | 精确删除AttributeInfo、AttributeValue、AttributeValueGroup及三项专属注释 | CustomAttributeRelation原两个int64字段/tag/无omitempty/声明顺序保持；四当前消费者/其它SHEIN DTO零改 |
| 新capturedata/model_test.go | 最小合成值合同回归：固定结果JSON/嵌套值/空值、五方法/原异常与错误链；复用原测试设施 | 独立手写期待与固定bytes，不以同一实现计算oracle；不新增runner/平台/数据库/真实浏览器验收 |
| 原检查与文档 | 仅§6.1准确转换、§5.3文档 | workflow/完整inventory/.golangci/依赖/保护及其它检查零改 |

### 5.2 原测试与验证边界

七个现有model-import测试仅owner/限定名改接；所有fixture、错误/关闭/取消/顺序/URL/池及质量断言原样。原 root/browser/extractor/variations 的其它测试零改，当前 acquisition/amazon、Sourcing/SourceIdentity/hash、SHEIN JSON/codec/Store/登录和权限原测试零diff。没有旧模型专属测试需要连带删除；三个孤立SHEIN DTO也无当前专属测试，不制造空测试或stub成功。

实现前可在未来新model_test.go以原model类型运行合成固定wire/方法基线PASS，再在同一路径只改测试owner验证抽取后等价；这是冻结未变化合同，不为机械移动制造业务RED。完整退休guard必须在旧根/DTO存在时真实RED、切换删除后GREEN。现有时间/nullable和VariationValue的输出/错误以独立合成样例确认；不读取历史文件、修改原golden或跑真实入口。

### 5.3 用户确认并整批实施的53唯一路径

<details><summary>完整清单：同一主要PR收齐</summary>

| 路径 | 处置 |
| --- | --- |
| `docs/architecture/README.md` | CURRENT_DOC |
| `docs/architecture/amazon-crawler-runtime-flow.md` | CURRENT_DOC |
| `docs/architecture/legacy-amazon-listing-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-amazon-source-adapter-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-crawler-facade-diagnostics-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-numeric-listing-support-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-platform-api-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-product-policy-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-source-result-contract-closure-retirement-boundary.md` | CURRENT_DOC |
| `docs/architecture/project-boundaries.md` | CURRENT_DOC |
| `docs/refactoring/legacy-register.md` | CURRENT_DOC |
| `docs/refactoring/module-target-mapping.md` | PREFIX_ONLY_HISTORY_NOTE |
| `hack/debug/test-amazon/main.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/batch_processor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/batch_processor_test.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/browser/error_detector.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/browser/pool_manager.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/browser/pool_manager_test.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/availability_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/bestseller_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/brand_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/categories_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/delivery_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/description_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/error_detector.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/feature_parser_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/features_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/image_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/list_price_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/list_price_extractor_test.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/parent_asin_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/price_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/price_extractor_test.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/product_details_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/rating_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/seller_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/ships_from_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/size_chart_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/title_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/variations_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/extractor/video_extractor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/instance_processor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/instance_processor_test.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/processor.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/processor_test.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/capturedata/model.go` | NEW_MINIMAL_PURE_VALUE_OWNER |
| `internal/crawler/amazon/capturedata/model_test.go` | NEW_SYNTHETIC_CONTRACT_REGRESSION |
| `internal/crawler/amazon/result_validator.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/crawler/amazon/result_validator_test.go` | REWIRE_CURRENT_CONTRACT_ONLY |
| `internal/model/amazon_product.go` | EXTRACT_AND_RETIRE_OLD_ROOT |
| `internal/shein/api/attribute/interface.go` | RETIRE_THREE_EXACT_TYPES_KEEP_RELATION |
| `tests/listingkit_backend_retirement_test.go` | EXACT_TWO_CHECK_CONVERSIONS_PLUS_ONE_GUARD |

</details>

## 6. 必要验证与验收

原设计阶段只进行源码/原AST/3184 Go消费者/字段/隐式方法/错误路径/原检查/历史依据/链接/清单的只读核对，运行适用既有文档/inventory元数据检查与准确设计HEAD原CI，不调用构造器、编解码业务路径、浏览器/provider、配置加载、数据库或已有资源。设计CI/设计评审不替代未来源码CI/用户验收。

用户确认Ready后，完整guard RED→17类型/5方法抽取＋36消费者切换＋三DTO/旧根删除＋§6.1精确转换→GREEN。必要自检限新纯值合同、原 Amazon root/browser/extractor/variations mock/错误/批量/取消、当前有效SHEIN与Sourcing合同及原必需检查；新值叶子与debug/main只编译到任务TEMP，不执行产物。独立hack/debug模块保留当前go.mod/go.sum与root replace，原Amazon入口仅编译。当前命令视影响只编译，不自行扩成环境验收、故障矩阵或新平台。

局部测试、compile-only、原PR CI、实际merge-SHA push/main CI、真实运行/用户验收分别记录；当前后两者NOT_RUN。既有本机CRLF等真实FAIL保留并分列环境证据，不改测试/黄金值求绿。源码缺席或另一个AI评审不是产品验收或部署授权。

### 6.1 原检查、当前文档及冻结边界的精确转换

1. `tests/listingkit_backend_retirement_test.go` 新增一个 `TestLegacySourceResultContractClosureIsRetired`：旧internal/model整根与decoded旧import缺席，API attribute原owner三DTO缺席，17真实struct/5纯方法及CustomAttributeRelation必需；新叶子只允许encoding/json/fmt/time、无provider/App/DB/文件/日志/隐式clock/父运行层依赖、无顶层alias/reexport/转发；明确各原生产消费者和debug都改接新owner。复用原trackedFiles/AST/decoded-import及错误即失败helper，按owner限定，不全局ban Product/AttributeValue同名声明，不新增工具/runner。
2. 同文件 `TestLegacyCrawlerFacadeDiagnosticsClosureIsRetired` **只改两处owner字符串**：retained表的 `internal/model:Product` → `internal/crawler/amazon/capturedata:Product`；原过滤条件 `owner != "internal/model"` → `owner != "internal/crawler/amazon/capturedata"`。原过滤是精确map owner，不是prefix；两处必须一起转换，否则新叶子Product不会被观察。其余retired全集、fragments、shapes、SHEIN/1688/池/变体/取消/AST/decoder/失败关闭断言与算法逐字保持。
3. 同文件 `assertLegacyPlatformAPICallClosureRetired` **只把requiredByFile的attribute/interface.go一行**由四声明缩为CustomAttributeRelation。全部其它KEEP行、retiredByOwner、decode/遍历/错误/正向断言保持；新guard负责三DTO确切缺席。原函数是helper，不能因为不以Test开头漏进矩阵。
4. 除上述 **两个原函数（含helper）与一个新增guard**，所有原检查/helpers/原测试逻辑零改。`TestPhase3...`现有Sourcing对internal/model及crawler的永久导入禁止保持；旧internal/model Task路径和retired-owner禁令保持，即使旧根消失也不移除禁令。原guard已经扫描稳定根/tracked Go，未发现额外missing-root读取入口；不要加missing-skip、dummy root或删检查。
5. 当前文档仅本文、架构README、Legacy register、project-boundaries、amazon-crawler-runtime-flow五项，说明新叶子归属、原debug和当前application的两条不同路径/事实边界、原检查转换；7旧依据只加准确顶注，原Git blob/数字/勾选/正文保留。README/register保留最终候选main完整索引后缀。dated inventory/历史观察/旧计划不是当前调用证据，不机械改其旧路径。
6. workflow、完整phase2/external-client等inventory（CURRENT/历史/计数/ceilings）、.golangci、root与debug依赖/锁、部署/保护规则 **零diff**。本批两个合同不占四条CURRENT inventory行，无校准需要。#956 Store与#692样例当前只与本批架构README相交，无源码重叠，不接管其实现。
7. #301更新前新读当前正文，保持OPEN；仅当前段/准确PR证据可更新，从“以下保留 #726 已合并批次与历史证据，不扩大本轮权限。”起的完整历史后缀及旧勾选逐字保留，正文≤65536字符，前当前段先归档评论。

8. 原CI40305890688真实失败后，用户批准准确命名例外（#964/6413700926）：本批新叶子由result改名capturedata，两个新文件/36import与本批guard/文档同步；完整manifest仍53路径。原repository_structure的result产物禁令、architecture_docs显式索引算法及其余检查零改，不增加白名单/skip。README只恢复反引号文件名索引格式。17值/五方法body/JSON/hash/事实owner和所有保留能力不变，原失败与新准确CI分别记录。

本节已核对root消失的同根source/requiredOwner观察及原helper。真实CI若证明还须矩阵外改动，先查齐同根sibling，按AGENTS分类、给精确最小提案并取得用户例外后再改；不能临时放宽原检查/inventory/版本或省略仍有效断言。

## 7. Review、确认与停止点

- 一个Writer、原独立managed工作区新分支、一个主要PR。先完成准确设计HEAD原CI与适用独立Architecture Review，交用户确认本§1–§7（明确含§4.2整个值codec/错误合同保持及§6.1）；记录并读回Independent Architecture / IMPLEMENTATION_READY / Ready后，才同一PR整批53路径实施。本轮准入已写入读回，当前SOURCE_IMPLEMENTED/IN_REVIEW。
- 起始同步main；最终候选前核对实际main/其它owner组合，仅处理真实增量。未变化代码证据可明确复用，不冒充新SHA直接执行，不为无关main变化机械重复全量审查/CI。
- 设计最多两轮正常review；finding按当前Must/具体后果/阻塞层分类，只有新BLOCKER重开冻结设计；非Blocker在实现测试或Backlog收敛，不追加全局平台、试用流程或验收矩阵。
- GitHub启动失败至多一次重试，Running不重复触发；**未经用户本轮授权不默认新增AI互验线程**。旧#961的设计/源码复核授权不覆盖本批，不新增并行Writer、全局架构或工具项目。
- 用户确认实施后一次集中源码最终交付复核，修复只复核相关增量；设计PASS/源码CI/另一AI都不签发用户/生产验收。合并另交最终准确HEAD授权，正常expected-head合并后仅认实际merge-SHA push/main整runcompleted SUCCESS，不管理员绕过/强推/改保护。
- #301保持OPEN；当前能力/原Amazon浏览器/debug与独立SHEIN登录/Store同步保持。不读取/操作既有实例、数据、凭据、输出/轨迹/profile文件或进程；不改workflow/inventory/依赖/保护，不增加试用前置。本轮源码实施已获授权；合并、部署、关单或真实运行仍另需授权。
