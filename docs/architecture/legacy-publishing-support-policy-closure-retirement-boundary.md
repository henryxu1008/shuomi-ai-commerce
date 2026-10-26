# #301 旧 Publishing 公共层、闲置 SHEIN 策略与 JSON 尾部完整退休设计

> **Independent Architecture / USER_CONFIRMED / IMPLEMENTATION_READY / SOURCE_IMPLEMENTED / IN_REVIEW**。用户已确认本§1–§7（明确含§4.2、§6.1）并授权同一#961整批78路径实施；[准入6409852021](https://github.com/qq550723504/shuomi-ai-commerce/pull/961#issuecomment-6409852021)与Issue/PR Ready记录已读回。设计准确HEAD `3e9371e87da82ae561403640a7dab06813030bfc` 的原CI40278175508 SUCCESS与独立只读设计复核PASS可追溯；开工正常同步main `0068545e6dd74d87323bdc8831680aa9f686b9ba`。本文件§1–§7为冻结设计与处置矩阵，文中“未来/拟/当前未实施”表述保留设计阶段原始语境，实施滚动证据以[#961](https://github.com/qq550723504/shuomi-ai-commerce/pull/961)为准；不将设计CI或源码自检当运行/产品验收。#301 OPEN，合并另授权。

## 1. 这次做什么

同一批完成三组：

1. 让旧 `internal/publishing/common` 整包及最后的 `internal/publishing` 根退出。当前仍用的八个值类型和本地组装行为，放回已有 SHEIN owner，不新建通用 common 包或兼容 alias。
2. 退出已无当前生产入口的十一组 SHEIN 策略及三个混合文件里的专属尾部。保留现役定价、离线校验、图片/站点/重量规则；旧选仓 alias/转发一起退出。
3. 退出 `internal/pkg/jsonx` 的两个最后辅助函数。其最后的旧 AI 扩展消费者已随 #960 退出，当前 Go 消费者为零。

使用者是维护当前本地 SHEIN 草稿、预览和离线诊断的开发者。交付结果是同一 Builder/codec/validator 路径继续工作，旧公共 owner、无接线策略及其原检查/文档尾部完整退出；没有声称用户必须等待本批才能试用。当前设计发现并明确提出一项已有确定性缺陷的最小修复：新生成的属性数组按原始属性名排序，详 §4.2；不是暗中改变持久化 hash 合同。

本轮源码未来精确 **78 唯一路径**：旧根10文件、十一策略族22文件、三个混合策略源码/测试6文件、15当前值消费者、2当前不变量测试、5新最小owner文件、3原检查、15文档。完整清单见 §5.3，一批完成，不按文件/包/提交机械拆 PR。

## 2. Design Basis、Authority 与前批 KEEP 替代

采用 **Independent Architecture**：公共 Go 值合同归属变化、超过30文件，且必须明确当前 JSON/排序/持久化边界。依据 [全新安装决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[目标架构](project-target-architecture.md)、[项目边界](project-boundaries.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)及本轮用户决定。无 UI/IA/页面/交互新增，Figma Authority **N/A**。

当前产品合同沿用 [我的供应链 SHEIN](my-supply-chain-shein-v1.md)、[完成记录店铺归属](issue-36-completed-work-store-scope.md)与[官方 publication closure](shein-publication-closure-v1.md)。本地不完整草稿/离线诊断不等于官方可上传 payload、提交授权或默认生产接线；不将隔离 DRAFT-S1 路径强行装入 current-application。

Must：完整闭包、八个当前值类型的全部字段/tag/顺序/空值语义、原严格 JSON/数值/Unicode/限制、当前 hash/绑定与诊断算法/版本/golden、Record 事实与权限/事务 owner、真实采集/图像/AI、PoolManager deadline/取消及独立 SHEIN 登录/Store 同步保持。§4.2 只修复新生成数组的确定性，不重写既有 payload。Should：直接复用当前 owner、标准库 sort/maps/strconv/encoding/json、现有 strict-json 与原测试，不建设新共享层。Out of Scope：新业务/UI/provider、官方提交/Store 重写、schema/迁移、权限/身份/租户、预算计费/retry/UNKNOWN/恢复、workflow/任何 inventory/依赖/保护变更、既有资源/凭据/数据/进程操作。无新增 Accepted Risk；物理运行态存在性 UNKNOWN，不探查。

Threat Model 只覆盖误删当前 wire/有效规则、旧 owner 通过 alias 回流、字段或字节/绑定漂移、数组非确定性、把旧机制测试当当前产品要求；不扩全局安全模型，不增加试用前置。

| 前批依据 | 本轮准确替代 | 继续保留 |
| --- | --- | --- |
| [#817 数字 SHEIN 整链](legacy-shein-listing-closure-retirement-boundary.md)、[#824 支撑闭包](legacy-numeric-listing-support-closure-retirement-boundary.md) | publishing/common 为跨平台公共 owner、整批纯策略继续 KEEP 的旧边界 | 当前草稿/JSON/平台 DTO/有效规则与当前各 owner |
| [#827 Publishing runtime](legacy-shein-publishing-runtime-closure-retirement-boundary.md) | common 公共壳、SDS/POD/尺码/来源维度/旧预校验和图片写入策略尾部 | 当前 Kernel/intent/receipt/effects、纯 codec/预览/离线读取 |
| [#891 预览/下载](legacy-listing-preview-download-closure-retirement-boundary.md) | common/本批闲置策略的广泛 KEEP | 当前图像处理、资产范围、草稿与有效 API DTO |
| [#909 productpolicy](legacy-product-policy-closure-retirement-boundary.md)、[#912 策略/合同](legacy-shein-policy-contract-closure-retirement-boundary.md) | 本次明确纳入原排除范围的独立 CostPriceRule/CalculateCostPrice、十一族和混合尾部 | 当前 PricingPolicy.Apply 的全部公式、输入/输出及原测试；不以它替代退休的成本转换公式 |
| [#914 闲置支撑](legacy-idle-support-contract-closure-retirement-boundary.md) | jsonx/UnmarshalString/CleanLLMResponse 当时仍有消费者的 KEEP | 真实 Amazon browser Processor、AI短超时/下载常量与当前 JSON/AI owner |
| [#926 平台 API 闭包](legacy-platform-api-closure-retirement-boundary.md) | 本批十一族与三个混合尾部的纯规则 KEEP | 原 Amazon 浏览器/调试、当前平台 wire DTO、SessionTransport/Store/独立登录 |
| [#960 草稿装配](legacy-shein-draft-assembly-closure-retirement-boundary.md) | common 作为仍保留依赖、marketplace 纯策略整体不动、原相关 guard/README 边界 | 29现有值类型/16公开纯函数、严格 JSON/版本/golden及固定本地 Builder；新增七值类型与Price归属按 §4.1 |

只有本矩阵和 §5/§6.1 的精确范围被替代。九旧依据在未来实施时只前置精确说明，原完整 blob/数字/勾选/证据保留；其它历史不重写。当前仅三设计文档，不将上表写成已经源码退休。

## 3. 实际消费者、闭包与新依赖方向

基线全部 **3211 tracked Go** 已用标准 Go AST 只读解析，含测试、cmd/hack/debug 与未选择的编译源；导入路径用 strconv.Unquote 解码，区分包 alias 与局部遮蔽。另逐项核对声明/类型字段/变量初始化/包内调用闭包，而非只按字符串或函数引用数删除；例如 fallback 仍调用的 variantOptionSplitter 必须随当前行为保留。

| 当前证据 | 结论 |
| --- | --- |
| common 18处实际 import＝9生产+9测试，全部在 SHEIN 子树 | 生产 import 不等于现役入口：SDS 文件只有旧机制，warehouse 的 common 使用仅在闲置 alias/转发。其余真正服务 draft/offlinepackage/PricingPolicy |
| marketplace/shein/publishing 9处实际 import＝8生产+1测试 | 8生产文件为 draft/local_package、offlinepackage/{payload,payload_defaults,payload_images,readiness}、validator/{validator,diagnostic}、workspace/submit_payload_readiness_checks；只保留其实际规则调用闭包 |
| 十一策略族未出现在上述生产选择器及其包内传递闭包 | 无当前应用/独立CLI/debug/反射注册/构造入口；不将其自测互调当当前业务能力。无服务初始化/端口调用证据 |
| jsonx 0个 Go import | UnmarshalString/CleanLLMResponse 无生产或测试消费者，整个两文件包退出；不替换标准库或其它 owner 同名严格 JSON 功能 |
| goroutine 1个生产 import；timeout 3生产+2测试 import | Amazon extractor/browser 与下载/安装入口真实使用，整体不动 |

旧 common 的18个 import 文件，除将整退的 SDS源码/测试外，剩16个全部纳入本批；warehouse源码同时属于混合声明处置，去重后为15个普通值消费者+1混合文件。8个当前平台规则生产消费者中，只有draft/local_package.go同时改接common源值；其它7个没有common import，源码零改，受原回归与当前字段合同保护。

新 DAG：`Record → draft.Builder → 私有本地源值/变体组装 → offlinepackage 当前值`；`draft / offlinepackage → publishing.Price + 现有平台纯规则`；`offlinepackage.Variant → catalog/canonical.Dimensions/Weight` 只用现有 Product 值类型；`validator → offlinepackage + workspace + 当前 publishing 纯规则`。publishing 不反向导入 offlinepackage；数据 owner 不依赖 draft/Record/App/provider/DB/文件或隐式时钟，不形成循环，不新增通用 value/common 包。

当前 actor/Organization/Store、Record/payload/diagnostic、官方 Submission/UNKNOWN、独立登录/Profile/fence/Store 同步仍由原 owner 负责。Go 值类型移动不转移 durable fact owner。

## 4. 当前合同、持久化与唯一行为修复提案

### 4.1 八个值类型与有效行为抽取

| 当前内容 | 目标 | 保留要求 |
| --- | --- | --- |
| common.Price | 现有 marketplace/shein/publishing 新 price.go 定义真实 struct | Currency/Amount/CostPrice 类型/tag/顺序全部相同；PricingPolicy.Apply 仅签名所有者改接，本体/零策略/全部启用公式和测试不改，不保留 type alias |
| Variant、ImageSet、PublishImageBundle、BundleSlot、MissingSlot、Attribute、Site | 现有 offlinepackage 新 source_values.go 定义七个真实 struct | 全部字段/tag/顺序、指针/null/omitempty/数值语义同原；Variant.Price直接指向 publishing.Price，Dimensions/Weight直接复用原canonical值类型；不丢 image_bundle/SourceImages/缺槽/元数据字段 |
| BuildVariants 及 fallback 私有完整闭包 | draft/local_source_variants.go 私有化 | DEFAULT-001、颜色×尺码顺序、属性价格推断、Stock/图片/尺寸/重量、正则、clonePrice、空值语义全部保留；无新resolver或配置 |
| BuildImages/FlattenAttributes/BuildAttributes/DefaultSites/FormatFloat/CloneMap/ParseFloatDefault/LastCategory/UniqueStrings | draft/local_source_values.go 私有化 | 原批准资产角色/顺序/去重、零值格式及nil/空集合语义保留；CloneMap可直接用标准库maps.Clone但必须保留len=0→nil；BuildAttributes唯一排序变化见 §4.2 |
| FirstNonEmpty | 直接复用 draft/local_copy.go 已有同语义私有 firstNonEmpty | 不复制第二实现、不新建公共工具 |
| ResolveBrand/WithBrandHint 当前调用固定空hint | 既有 local_package.go 直接取 product.Brand / strings.TrimSpace(product.Title) | nil product 原保护保持；不搬运没有生产入口的 brand hint 参数分支 |
| 原29 wire类型/16公开纯函数、当前平台 API DTO/content/authorizedbrand | 原当前owner | 字段与公开算法不动；model.go只改七个原common类型的限定引用。两个JSON/诊断golden、typed v1.2 / diagnostic v2.2 / binding go-json.v2 零改 |

BundleSelectionRule、WarehouseOption、CollectReviewNotes、CloneImageSet 及旧按国家选仓实现没有当前调用；不搬运。ImageSet深复制的当前要求由原 ClonePackageForPersistence 的有效测试承担，详 §5.2。

### 4.2 已有属性数组顺序缺陷：设计中显式提出的最小修复

Finding：common/helpers.go 的 BuildAttributes 将 map 直接 range 成 []Attribute；draft/local_package.go把同一结果用于 Package.ProductAttributes 与 DraftPayload.ProductAttributeList，预览也消费该序列。Go map不保证顺序，所以有多个属性时，重复生成的原JSON字节/PackageHash及诊断绑定可能不同。现有 Builder 测试已要求“pure draft mapping must be byte-deterministic”，但 #960 三个固定字节向量没有覆盖多属性 map。

Classification：**IMPLEMENTATION_TEST**，影响现有确定性 Must，阻塞未来源码候选的该项自检；不是新增产品需求、状态机、事务或全局架构 Blocker，不增加试用前置。当前阶段只读确认，未执行 Builder 或修改源码。

拟在本批 EXTRACT 时只改新私有 buildSourceAttributes：收集原始属性名，用标准库 sort.Strings 按 Go 字符串顺序排序，再用原始 Name/Value 逐项构造；不trim、改名、归并、去重或改值，不修改输入map。len=0仍返回nil；没有其它数组重排、属性优先级或来源规则变化。

**这是唯一有意的运行输出变化，须用户确认本设计（含本节）后才能实施。** 多属性新生成草稿将获得确定顺序，其新字节/hash可能与旧程序某一次随机输出不同；不承诺所有旧生成结果逐字相同。原三组固定Builder字节/hash、两个diagnostic黄金值和全部现有fixture保持，不通过更新golden或版本收口。

持久化界限：Record.Service.Create 对实际payload做原digest、诊断再保存，canonicalInputHash按当前产品/批准资产/规则输入形成；读取/既有记录和回执不因类型移动而重建草稿。本批不修改 Service、Repository、SQL、InputHash/digest/绑定算法或版本，不读取/扫描/重写现有记录，不添加迁移、双读/双写、兼容或fallback。修复只作用于未来新的本地组装；已有payload的原字节/hash继续原样严格读取与校验。

实现先用合成多属性输入的独立期望（字段顺序、重复生成、原始值、不修改输入）在原函数上记录真实RED，再最小排序→GREEN，并保留原全部固定向量。若实际测试表明必须改变既有golden、原诊断/hash算法或持久化边界，停止对应修改并报告，不自行扩大本节。

### 4.3 各事实与副作用 owner 不变

| 边界 | 处置 |
| --- | --- |
| ProductSnapshot / ApprovedAsset精确范围、原版本和权限 | 原Product/Asset owner零改，来源图片不得回流，清空variant源图及批准资产替换保持 |
| 同步有界Builder、ctx前后检查、marshal/大小/严格自检 | 入口与逻辑保持，只有§4.2排序；不添加远端I/O、时钟、预算或取消策略 |
| Record/payload/diagnostic/输入hash/回执、权限/事务/租户 | 原Service及已注入Repository owner零改；本地计算不成为第二事实源，无schema/迁移 |
| 现有官方Kernel/intent/receipt/effects、Store观察、登录Profile/fence/UNKNOWN | 原owner/原状态机与副作用零改；不移交旧重试/恢复/上传/选仓责任，不探查运行态 |

没有新增资金、授权、跨系统安全边界、provider mutation或持久化协议，不引入outbox/Saga/通用调度器。真实运行、历史数据/凭据存在性 UNKNOWN；源码退休不证明物理资源已退出。

## 5. 完整删除、保留、抽取矩阵

Legacy decision: **EXTRACT + RETIRE**。Reusable behavior：八个当前值类型、当前固定本地源值/变体组装及有效纯规则。Current owner：既有 offlinepackage / draft / publishing，直接复用标准库与原 Product 值。Cutover/deletion condition：全部当前消费者切换、两旧根及闲置策略/alias缺席、JSON/版本/golden保持、§4.2确定性修复和§6.1原检查转换满足、准确源码CI与最终复核完成；不含运行资源删除。

### 5.1 十一整退族与三个混合文件

以下十一族的生产源码和专属原测试一起退出；只检验已退出机制的fixture不搬运成当前接口。并非整个 SHEIN publishing 包退休：当前生产选择器及其完整纯调用闭包继续原owner。

| 策略族（均在 marketplace/shein/publishing） | 退休闭包 |
| --- | --- |
| `attribute_value_match_policy.go` + `_test.go` | 旧属性匹配入口，含该文件全部声明/常量/正则/专属helper |
| `attribute_value_segment_policy.go` + `_test.go` | 旧属性值分段，含该文件全部声明/常量/正则/专属helper |
| `cost_pricing_policy.go` + `_test.go` | 旧成本转换公式（不替换为当前PricingPolicy），含该文件全部声明/常量/正则/专属helper |
| `pod_submit_readiness_policy.go` + `_test.go` | 旧POD提交readiness编排，含该文件全部声明/常量/正则/专属helper |
| `sds_image_policy.go` + `_test.go` | 旧SDS图片索引/查找/合并，含该文件全部声明/常量/正则/专属helper |
| `size_attribute_policy.go` + `_test.go` | 旧SDS尺码表构建/头部映射，含该文件全部声明/常量/正则/专属helper |
| `source_dimension_policy.go` + `_test.go` | 旧来源维度选择，含该文件全部声明/常量/正则/专属helper |
| `submit_mode_policy.go` + `_test.go` | 旧最终草稿提交模式规范化，含该文件全部声明/常量/正则/专属helper |
| `submit_prevalidation_policy.go` + `_test.go` | 旧预校验对象/入口，含该文件全部声明/常量/正则/专属helper |
| `text_normalization_policy.go` + `_test.go` | 旧刊登文本格式策略，含该文件全部声明/常量/正则/专属helper |
| `variant_image_policy.go` + `_test.go` | 旧变体图片集合匹配，含该文件全部声明/常量/正则/专属helper |

| 混合文件 | 精确退休 | 原样保留 |
| --- | --- | --- |
| submit_image_policy.go | FinalDraftImageInput、OrderFinalDraftImages、FirstNonEmptyImageURL、ProductImageInfoFromFinalDraft、ProductImageDetailsCoverFinalDraft、finalDraftImageInputHasImage、ReorderFinalDraftProductImages、GalleryWithoutMain、NormalizeImageRoleOverrides、NormalizeImageRoleOverride、IsUploadedImageURL、IsSDSImageURL、CloneImageUploadCache | FinalSubmitImagesRequireSKC、FinalSubmitImageReadinessInput、SubmitImageDraftInput、FinalSubmitImagesReady、SubmitImageDraftHasImage、SubmitImageURLSliceHasImage、SubmitImageURLsHaveImage、ProductImageInfoHasImage、UniqueNonEmptyImageURLs；原body/文案/顺序/条件保持 |
| variant_image_coverage_policy.go | VariantImageCoverageState/GroupInput/MainImageInput、EnforceVariantImageCoverage、VariantImageGroupCount/Key、DistinctVariantImageMainImageCount、VariantImageMainImageURL、SetVariantImageCoverageMetadata、专属blocked长文案常量 | VariantImageCoverageStatus、两个原metadata key常量、blocked状态常量和reader原fallback文案/空值/裁剪/判断全部保持；不删metadata字段/已有值 |
| warehouse_policy.go | WarehouseOption alias、SelectWarehouseCodeForSite转发 | SubmitPreferredWarehouseCode及其DEFAULT常量，原函数体/字段规则不变 |

十四个策略生产文件精确退休 **119个声明**（含函数/类型/常量/变量，不混称119个函数），包括上述三个混合文件。完整名字按基线AST列在下方供复核；任何在别的owner下同名的当前声明不受全局禁名影响。

<details><summary>策略声明准确清单</summary>

- `internal/marketplace/shein/publishing/attribute_value_match_policy.go`：`MatchPlatformAttributeValue`。
- `internal/marketplace/shein/publishing/attribute_value_segment_policy.go`：`ComparableAttributeSegments`。
- `internal/marketplace/shein/publishing/cost_pricing_policy.go`：`CostPriceRule`, `CalculateCostPrice`。
- `internal/marketplace/shein/publishing/pod_submit_readiness_policy.go`：`saveDraftPODReadinessFallback`, `PODSubmitReadiness`, `EvaluatePODSubmitReadiness`, `firstNonEmptyPODReadinessMessage`。
- `internal/marketplace/shein/publishing/sds_image_policy.go`：`DefaultSDSImageKey`, `SDSImageLookupInput`, `RegisterSDSVariantImageSet`, `FirstSDSImageSet`, `ResolveSDSImages`, `SourceSDSSKUFromSupplierSKU`, `ImageSetFromSDSMockups`, `MergeSDSImageSet`, `NormalizeSDSImageKey`, `LookupSDSImagesBySKU`, `UniqueNonEmptySDSImageStrings`, `AppendUniqueSDSImageURLs`。
- `internal/marketplace/shein/publishing/size_attribute_policy.go`：`cmValuePattern`, `leadingNumericValuePattern`, `apparelSizeAttributeIDs`, `sizeHeaderAliases`, `sdsSizeCell`, `SizeChartTemplateAttribute`, `SizeSaleAttributeRef`, `BuildSizeAttributesFromProductSize`, `BuildSizeAttributesFromProductSizeWithTemplates`, `BuildSizeAttributesFromProductSizeWithHeaderAttributeIDs`, `UnmappedSDSSizeHeaders`, `IsSizeChartTemplateAttribute`, `isKnownGarmentSizeChartAttributeName`, `IsSizeSourceDimension`, `parseSDSProductSizeTable`, `indexSizeSaleAttributeRefs`, `indexSizeChartTemplateAttributes`, `indexSizeChartTemplateAttributeIDs`, `indexSizeHeaderAttributeIDOverrides`, `sdsSizeAttributeID`, `isValidSizeHeaderAttributeID`, `normalizeSDSSizeHeader`, `extractCentimeterValue`, `normalizeSizeText`, `intSlicesEqual`。
- `internal/marketplace/shein/publishing/source_dimension_policy.go`：`sourceDimensionLeadingScalePattern`, `sourceDimensionNoisePattern`, `SourceDimension`, `SourceDimensionSelection`, `SelectSourceDimensionsFallback`, `SourceDimensionExists`, `SourceDimensionPrimaryPriority`, `SourceDimensionSecondaryPriority`, `IsDescriptiveSourceDimension`, `IsNumericScaleSourceDimension`, `IsNumericLikeSourceDimensionValue`, `normalizeSourceDimensionText`, `normalizeSourceDimensionValue`, `trimSourceDimensionCodePrefix`, `isLikelySourceDimensionCodePrefix`。
- `internal/marketplace/shein/publishing/submit_mode_policy.go`：`NormalizeFinalDraftSubmitMode`。
- `internal/marketplace/shein/publishing/submit_prevalidation_policy.go`：`submitProductValidationInput`, `submitProductValidator`, `PreValidateSubmitProduct`, `PreValidateSubmitProductWithOptions`, `submitProductValidator.preValidate`, `submitProductValidator.validateBasicProductInfo`, `submitProductValidator.validateSKCAndSKUData`, `submitProductValidator.validateResolvedMappings`, `validateResolvedAttributes`, `validateResolvedSaleAttributes`, `validateResolvedSaleAttribute`, `validateQuantityTypeAndValue`, `validateSubmitQuantity`, `summarizeSubmitValidationIssues`。
- `internal/marketplace/shein/publishing/text_normalization_policy.go`：`sheinHTMLTagPattern`, `sheinEmojiPattern`, `sheinSpecialCharPattern`, `sheinWhitespacePattern`, `sheinAttributeSpecialCharRegex`, `sheinAttributeInchRegex`, `sheinAttributeFtRegex`, `sheinAttributeReplacementTokens`, `NormalizeSheinContentText`, `SanitizeSheinAttributeText`, `IsValidSheinAttributeText`, `IsOnlySheinAttributeSpecialChars`, `RemoveRemainingSheinAttributeSpecialChars`。
- `internal/marketplace/shein/publishing/variant_image_policy.go`：`VariantImageSet`, `VariantImageSKCInput`, `NormalizeVariantImageSets`, `FindVariantImageSet`, `NormalizeVariantImageKey`, `uniqueNonEmptyImageStrings`。
- `internal/marketplace/shein/publishing/submit_image_policy.go`：`FinalDraftImageInput`, `OrderFinalDraftImages`, `FirstNonEmptyImageURL`, `ProductImageInfoFromFinalDraft`, `ProductImageDetailsCoverFinalDraft`, `finalDraftImageInputHasImage`, `ReorderFinalDraftProductImages`, `GalleryWithoutMain`, `NormalizeImageRoleOverrides`, `NormalizeImageRoleOverride`, `IsUploadedImageURL`, `IsSDSImageURL`, `CloneImageUploadCache`。
- `internal/marketplace/shein/publishing/variant_image_coverage_policy.go`：`variantImageCoverageBlockedMessage`, `VariantImageCoverageState`, `VariantImageGroupInput`, `VariantImageMainImageInput`, `EnforceVariantImageCoverage`, `VariantImageGroupCount`, `VariantImageGroupKey`, `DistinctVariantImageMainImageCount`, `VariantImageMainImageURL`, `SetVariantImageCoverageMetadata`。
- `internal/marketplace/shein/publishing/warehouse_policy.go`：`WarehouseOption`, `SelectWarehouseCodeForSite`。

</details>

### 5.2 原测试与当前断言的准确处置

| 测试族 | 处置 |
| --- | --- |
| common/variant_fallback_test.go 两个用例 | 完整fixture/数值/默认SKU/颜色×尺码/价格/顺序断言转到draft/local_source_variants_test.go；只改包/import/私有函数/类型owner，不重造oracle |
| common/image_set_test.go TestCloneImageSetDeepCopiesSlices | 经现有offlinepackage.ClonePackageForPersistence包装Package.Images，保留原主/白底/图库/SourceImages独立期望及两slice不共享断言；不保留无调用的CloneImageSet公开helper |
| common/warehouse_test.go三个用例 | 与旧按国家选仓机制一起退休；现役SubmitPreferredWarehouseCode原用例完整保留 |
| 十一整退策略族测试 | 整文件退出，不将旧SDS/POD/成本/生成机制的测试要求提升为当前产品要求；其实际当前替代已存在的规则与测试零改 |
| submit_image_policy_test.go | 精确保留TestFinalSubmitImagesRequireSKCSkipsSaveDraftOnly、TestFinalSubmitImagesReadyRequiresFinalDraftAndImages、TestFinalSubmitImagesReadyUsesActionSpecificSKCStrictness、TestSubmitImageDraftHasImageChecksAllImageSources、TestUniqueNonEmptyImageURLsTrimsAndDedupes、TestProductImageInfoHasImageChecksImageInfoList六函数全部fixture/断言；其余九函数随闲置机制退 |
| variant_image_coverage_policy_test.go | 退三个Enforce/GroupCount用例；TestVariantImageCoverageMetadataRoundTrip去Set writer fixture，直接构造原metadata map保持reader原阻断/消息期望，补nil/非blocked/空消息fallback直接断言；不复活writer或stub固定成功 |
| warehouse_policy_test.go | 只退三个TestSelectWarehouseCodeForSite*；TestSubmitPreferredWarehouseCodeUsesFirstConfiguredWarehouse原body不变 |
| publishing/pricing_policy_test.go | 只将common.Price改为当前本包Price，全部公式/舍入/边界值/断言原样；不删配置分支或替换成本公式 |
| 其它当前common import测试 | 仅类型限定/import改接；builder_test.go只增加§4.2确定性回归，原固定三字节/hash及原控制资产测试保持；clone_test.go只追加上列原图像深复制要求 |
| validator/diagnostic_test.go、当前content/采集/worker/Store/权限与持久化原测试 | 零diff，既有向量/值不改、不减少拒绝/隔离/取消断言；两个diagnostic黄金值不变 |

### 5.3 未来78唯一路径冻结清单

当前设计实际只有三文档；以下是设计确认、Ready读回后同一PR的整批实施范围，不是现在的修改授权。文档15项＝6当前职责说明＋9旧依据顶注；旧依据/索引/登记保留最终main完整blob后缀。新目标仅5文件，不自由新增common包、helper导出、alias或框架。

<details><summary>78路径与准确处置</summary>

| 路径 | 处置 |
| --- | --- |
| `docs/architecture/README.md` | CURRENT_DOC |
| `docs/architecture/legacy-idle-support-contract-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-listing-preview-download-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-numeric-listing-support-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-platform-api-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-product-policy-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-publishing-support-policy-closure-retirement-boundary.md` | CURRENT_DOC |
| `docs/architecture/legacy-shein-draft-assembly-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-shein-listing-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-shein-policy-contract-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-shein-publishing-runtime-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/project-boundaries.md` | CURRENT_DOC |
| `docs/refactoring/legacy-register.md` | CURRENT_DOC |
| `internal/marketplace/shein/README.md` | CURRENT_DOC |
| `internal/marketplace/shein/draft/builder_test.go` | CURRENT_INVARIANT_TEST |
| `internal/marketplace/shein/draft/local_cleanup.go` | REWIRE_CURRENT_VALUE_OWNER |
| `internal/marketplace/shein/draft/local_cleanup_test.go` | REWIRE_CURRENT_VALUE_OWNER |
| `internal/marketplace/shein/draft/local_package.go` | REWIRE_CURRENT_VALUE_OWNER |
| `internal/marketplace/shein/draft/local_preview.go` | REWIRE_CURRENT_VALUE_OWNER |
| `internal/marketplace/shein/draft/local_preview_test.go` | REWIRE_CURRENT_VALUE_OWNER |
| `internal/marketplace/shein/draft/local_preview_values.go` | REWIRE_CURRENT_VALUE_OWNER |
| `internal/marketplace/shein/draft/local_source_values.go` | NEW_MINIMAL_CURRENT_OWNER |
| `internal/marketplace/shein/draft/local_source_variants.go` | NEW_MINIMAL_CURRENT_OWNER |
| `internal/marketplace/shein/draft/local_source_variants_test.go` | NEW_MINIMAL_CURRENT_OWNER |
| `internal/marketplace/shein/draft/local_variants.go` | REWIRE_CURRENT_VALUE_OWNER |
| `internal/marketplace/shein/draft/local_variants_test.go` | REWIRE_CURRENT_VALUE_OWNER |
| `internal/marketplace/shein/offlinepackage/clone_test.go` | CURRENT_INVARIANT_TEST |
| `internal/marketplace/shein/offlinepackage/draft_shape_test.go` | REWIRE_CURRENT_VALUE_OWNER |
| `internal/marketplace/shein/offlinepackage/model.go` | REWIRE_CURRENT_VALUE_OWNER |
| `internal/marketplace/shein/offlinepackage/readiness_test.go` | REWIRE_CURRENT_VALUE_OWNER |
| `internal/marketplace/shein/offlinepackage/source_values.go` | NEW_MINIMAL_CURRENT_OWNER |
| `internal/marketplace/shein/publishing/README.md` | CURRENT_DOC |
| `internal/marketplace/shein/publishing/attribute_value_match_policy.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/attribute_value_match_policy_test.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/attribute_value_segment_policy.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/attribute_value_segment_policy_test.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/cost_pricing_policy.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/cost_pricing_policy_test.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/pod_submit_readiness_policy.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/pod_submit_readiness_policy_test.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/price.go` | NEW_MINIMAL_CURRENT_OWNER |
| `internal/marketplace/shein/publishing/pricing_policy.go` | REWIRE_CURRENT_VALUE_OWNER |
| `internal/marketplace/shein/publishing/pricing_policy_test.go` | REWIRE_CURRENT_VALUE_OWNER |
| `internal/marketplace/shein/publishing/sds_image_policy.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/sds_image_policy_test.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/size_attribute_policy.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/size_attribute_policy_test.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/source_dimension_policy.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/source_dimension_policy_test.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/submit_image_policy.go` | RETIRE_EXACT_DECLARATIONS |
| `internal/marketplace/shein/publishing/submit_image_policy_test.go` | EXACT_MIXED_TEST_CONVERSION |
| `internal/marketplace/shein/publishing/submit_mode_policy.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/submit_mode_policy_test.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/submit_prevalidation_policy.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/submit_prevalidation_policy_test.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/text_normalization_policy.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/text_normalization_policy_test.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/variant_image_coverage_policy.go` | RETIRE_EXACT_DECLARATIONS |
| `internal/marketplace/shein/publishing/variant_image_coverage_policy_test.go` | EXACT_MIXED_TEST_CONVERSION |
| `internal/marketplace/shein/publishing/variant_image_policy.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/variant_image_policy_test.go` | RETIRE_IDLE_POLICY_FAMILY |
| `internal/marketplace/shein/publishing/warehouse_policy.go` | RETIRE_EXACT_DECLARATIONS |
| `internal/marketplace/shein/publishing/warehouse_policy_test.go` | EXACT_MIXED_TEST_CONVERSION |
| `internal/marketplace/shein/workspace/package_template_validation_test.go` | REWIRE_CURRENT_VALUE_OWNER |
| `internal/marketplace/shein/workspace/submit_payload_readiness_checks_test.go` | REWIRE_CURRENT_VALUE_OWNER |
| `internal/pkg/jsonx/llm.go` | RETIRE_OLD_ROOT |
| `internal/pkg/jsonx/unmarshal.go` | RETIRE_OLD_ROOT |
| `internal/publishing/common/helpers.go` | RETIRE_OLD_ROOT |
| `internal/publishing/common/image_set.go` | RETIRE_OLD_ROOT |
| `internal/publishing/common/image_set_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/common/types.go` | RETIRE_OLD_ROOT |
| `internal/publishing/common/variant_fallback.go` | RETIRE_OLD_ROOT |
| `internal/publishing/common/variant_fallback_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/common/warehouse.go` | RETIRE_OLD_ROOT |
| `internal/publishing/common/warehouse_test.go` | RETIRE_OLD_ROOT |
| `tests/import_boundaries_test.go` | EXACT_CHECK_CONVERSION |
| `tests/listingkit_backend_retirement_test.go` | EXACT_CHECK_CONVERSION |
| `tests/validator_contract_boundary_test.go` | EXACT_CHECK_CONVERSION |

</details>

## 6. 验证与验收

设计阶段只做源码/原AST/全Go消费者/前批KEEP/字段/原检查/链接/历史blob和diffcheck；运行适用原文档/inventory元数据检查和准确设计HEAD原CI，完成适用独立设计复核。没有运行业务构造器、Builder、配置加载、浏览器、provider或运行验收。设计CI不替代未来源码CI/复核。

用户确认Ready后：①在原聚合加一个完整闭包guard，真实旧根/声明/import存在时RED；②用原Builder合成多属性输入及独立排序期望记录§4.2缺陷RED；③抽取八值/10私有helper及fallback完整闭包、复用当前firstNonEmpty、改接消费者、精确退32整文件和三个混合尾部、转换原检查→GREEN。只用现有go test和标准库；不创建runner、治理inventory或验证工具平台。

必要原验证：新/受影响draft与offlinepackage值/strict codec/clone/Variant/固定三字节hash，publishing保留公式/图片/metadata reader/站点重量，validator/diagnostic原两golden与拒绝，workspace、App合成确定性回归；readinessinspect若需数据库仅编译不连接已有资源。当前命令只编译。完整原CI留稳定候选。局部测试、compile-only、PR CI、实际merge-SHA main CI、真实运行和产品验收分别列明，后两者NOT_RUN；本机CRLF等既有FAIL保留并分列环境证据，不改golden或检查掩盖真实失败。

### 6.1 原检查的准确转换（未来实施）

1. `tests/listingkit_backend_retirement_test.go` 仅新增 `TestLegacyPublishingSupportPolicyClosureIsRetired`：整internal/publishing根、pkg/jsonx、十一族源码/测试、旧alias/119声明/专属尾部及decoded imports必须缺席；八个新owner真实值类型、当前Builder私有函数、PricingPolicy/有效reader/codec/Record/独立worker必需声明存在；数据owner无provider/App/DB/文件/时钟/反向draft依赖；按owner限定而非全局ban同名符号。复用原AST/trackedFiles/错误即失败helpers。
2. 同文件 `TestLegacySheinDraftAssemblyClosureIsRetired` **只改pureImports白名单**：删task-processor/internal/publishing/common，增task-processor/internal/product/catalog/canonical（仅原Dimensions/Weight值引用）；原29声明/私有map禁入/alias禁入/原版本/独立worker/所有其它断言原样。不要移动guard或重写其算法。
3. 同文件 `TestLegacyRuntimeSkeletonAndIdleConfigClosureIsRetired` **只删required表中jsonx的UnmarshalString/CleanLLMResponse一行**；retired全集、owner筛选中的jsonx项、原AST/全kernel解码/其它KEEP/image解码注册及失败关闭全保留。新guard负责整个jsonx包缺席。
4. 同文件 `TestListingKitTaskBackendSourcesAreRetired` **只删required读取表attribute_value_match_policy.go/MatchPlatformAttributeValue一行**；原Current/CLI/workflow/namespace/其它平台规则/内存词表工厂/禁止声明/算法全保持。新guard负责该策略明确缺席。
5. `tests/import_boundaries_test.go` 的 `TestPublishingSheinNonAPISheinImportsStayAllowlisted`（仅managed后段）、`TestPublishingSheinManagedAPIImportsStayAllowlisted`、`TestPublishingSheinManagedRetiredManagementImportsStayBlocked`：**各仅把loadGoFileIndex起点从filepath.Dir(managedRoot/root)上移至现存internal根**。旧managedRoot/root前缀过滤、allow/deny表、decoded path/error处理、原非API前段/content窄适配保持；不加missing-skip或删除旧禁令。上移是因整个旧publishing父根退出。
6. 同文件 `TestProductEnrichCanonicalImportsStayRetiredAcrossBuildTargets`：**旧common一条扫描表项改为offlinepackage、draft、marketplace/shein/publishing三当前owner**，catalog原项保留；原未选择源/禁止produchtenrich/解析失败处理算法不变。`TestProjectBoundaryDomainsDoNotImportListingKitFacade`：**只在既有Below分支条件追加internal/publishing域根**，使原parent=internal/old-prefix过滤继续执行；原roots列表、所有其它域/ban/helper保持。两个BusinessDomainsDoNotImportApp*原本已用父根Below，零改，不机械删旧root列表。
7. `tests/validator_contract_boundary_test.go` 的 `TestValidatorContractImportBoundary`：**offlinepackage白名单同样只删旧common、增canonical类型包**；其它当前root/依赖、禁止时钟/反射负例、错误/count/遍历/解码helpers全保持。
8. 上述 **9个已有原检查函数＋一个新增guard** 之外，所有原检查函数/表/helpers零改；`tests/architecture_docs_test.go`、`tests/depguard_config_test.go`、完整phase2及external-client inventory（含CURRENT/历史/计数/ceilings）、.golangci.yml、workflow/依赖/保护零diff。原dormant旧root depguard禁止行保留，不因旧根消失移除永久禁令。
9. 文档仅§5.3的15项：9旧依据只前置精确替代说明，完整旧blob保留；README/register保留稳定候选最终main完整后缀，project-boundaries与两个SHEIN职责README只校准当前owner/实际规则及原检查引用。#301更新前新读，当前段以外从“以下保留 #726 已合并批次与历史证据，不扩大本轮权限。”开始逐字保留，不改历史勾选，正文≤65536字符且OPEN。

本节已查齐旧父根退役的同根读取入口，不只改第一处报错。真实原CI若暴露矩阵外遗漏，先查全同根sibling、按AGENTS分类、给准确最小提案并取得用户例外后再改；不删除/放宽检查、不用重跑绕过、不临时改版本/golden。

## 7. Review、确认与停止点

- 当前一个Writer、一个主要设计PR，只有三文档。准确设计CI/适用独立Architecture Review完成后，交用户确认本§1–§7（明确含§4.2唯一排序修复及§6.1）；在Issue/PR记录Independent Architecture / IMPLEMENTATION_READY / Ready并读回后，才同一PR整批78路径实施。当前SOURCE_NOT_STARTED/NOT_READY。
- 起始同步实际main；稳定候选前再核对实际main/其它owner组合，仅处理真实增量。#956 Store、#692样例和#931依赖不接管；共享索引相交仅精确组合。未变代码证据可明确复用，不冒充新SHA直接执行。
- 最多两轮正常设计评审，finding先分类/指出当前Must、后果和阻塞层；只用真实BLOCKER重开冻结设计。§4.2为既有确定性缺陷的IMPLEMENTATION_TEST，不能凭其无限扩大属性策略、输入规范化或验收平台。
- GitHub评审启动失败至多一次重试，运行中不重复触发；**未经用户本轮授权不默认新增AI互验线程**。#960等旧设计/源码授权不覆盖本设计或未来源码复核，不新增并行Writer/架构治理项目。
- 用户确认实施后完成一次集中源码最终交付复核，修复只查增量；设计PASS、CI或另一AI都不是产品验收。最终源码CI/复核满足后另交准确HEAD正常expected-head合并授权，不管理员绕过/强推/改保护；合并后只认实际merge-SHA push/main整runcompleted SUCCESS。
- #301保持OPEN；独立SHEIN登录与Store同步及当前能力保持。没有部署/实例/数据/凭据/目录/进程操作、schema迁移或新增试用前置；运行/产品验收NOT_RUN。本设计不授权当前源码实施、合并或关单。
