# #301 旧 SHEIN 草稿装配包及专属扩展尾部完整退休设计

> Design Basis: **Independent Architecture / DESIGN_ONLY / DESIGNING / SOURCE_NOT_STARTED / NOT_READY**。当前只交设计、索引与登记三文档；用户确认本 §1–§7（含 §6.1）、适用独立设计复核及 Issue/PR 的 IMPLEMENTATION_READY/Ready 记录读回后，才在同一主要 PR 整批实施。调查基线 `6238d2a6670e75e05d49e71c3a33d6ae2cc0d65d`；接单与 #958 收尾归档 [#301 评论6407227866](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6407227866)。

> 最终候选已正常同步实际 main `f50fd0fbe326f7f30990fd299fd41fea8c9038f0`（#959）。新增仅 Amazon 采集扩展 TS/原测试及其设计/README，与本轮 Go 消费者/codec/原检查无交集，3244 Go/15实际imports调查不变；当前设计 diff 仍仅三文档。#956 Store分支只与共享架构索引潜在相交，其业务/schema/授权未接管；索引/登记保留最终main完整blob后缀。

## 1. 这次做什么

三件事作为一个完整闭包交付：

1. 当前草稿 JSON、严格解码、复制和离线检查真正需要的值合同与纯规则，抽到 `internal/marketplace/shein/offlinepackage`。
2. 当前本地草稿的确定性组装，回到已有的 `internal/marketplace/shein/draft`，由其私有函数完成；保留原入口 `draft.Builder.Build`，不再提供旧通用 Assembler/配置/远端扩展接口。
3. 整退 `internal/publishing/shein` 的 60 个 Go 工件和专属尾部。没有生产注入的 resolver、AI 文案、尺码 header resolver、可选定价装配、私有赋值状态和兼容 setter 不搬到新 owner。

使用者是维护当前本地 SHEIN 草稿、预览和离线校验的开发者。当前操作是 Catalog 精确快照 + ApprovedAsset 精确批准资产 → 本地不完整草稿 → 原 JSON/绑定 hash → 原离线检查。上述操作继续可用；本设计解决它们仍借用旧大包、容易把旧可选执行接口误当当前接线合同的问题。没有声称用户操作必须等待本批，不增加内部试用前置。

“最小”不等于删掉当前 JSON 能表达、严格解码或离线校验正在消费的字段。保留当前完整 JSON 可达的 29 个值类型、字段/tag/顺序/null/omitempty；退出的是旧 Go owner 与运行扩展。不是把旧包一比一改名，也不建立 type alias、旧路径 re-export、兼容包装、fallback 或第二套输入合同。

## 2. Authority、前批 KEEP 替代与准入

产品依据：[全新安装基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[目标架构](project-target-architecture.md)、[项目边界](project-boundaries.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)和本轮用户明确决定。无 UI/IA/页面/交互变化，Figma Authority **N/A**。跨当前草稿/校验/共享值 owner、未来 119 路径及公共 Go 合同归属变化，采用 Independent Architecture；不新增业务状态机或持久化事实。

当前产品边界沿用 [我的供应链 SHEIN](my-supply-chain-shein-v1.md) 的 DRAFT-S1/各数据 owner 分离、[原完成记录店铺归属](issue-36-completed-work-store-scope.md) 的本地记录与独立 Review/Apply/远端提交分离，以及 [官方 publication closure](shein-publication-closure-v1.md) 的许可/回执/UNKNOWN 合同。本地不完整草稿和离线诊断不等于官方可上传 payload 或提交授权；本次不改 Store 独立数据库接线，不把隔离 DRAFT-S1 constructor 挂入当前生产，也不改变这些产品决定或其实际 Figma 页面。

Must：当前草稿/预览/严格 JSON、确定性 hash/绑定版本、有效规则、ApprovedAsset 精确范围、身份/组织隔离、当前采集/图像/AI、PoolManager deadline/取消、独立 SHEIN 登录与 Store 同步保持。Should：当前消费者只依赖当前 owner，直接复用标准库、现有 `sigs.k8s.io/json`、平台纯规则及内存内容清理。Out of Scope：新功能/页面、当前官方刊登或 Store 重写、新 AI/provider、权限/预算/计量/retry/UNKNOWN/恢复、schema/迁移/版本升级、兼容层、运行/数据操作、workflow/任何 inventory/依赖/保护修改。无新增 Accepted Risk。

Threat Model 只覆盖误删当前 wire/有效规则、误连旧远程或私有运行态、资产来源替换、hash 漂移、丢失原失败关闭和旧 owner 通过 alias 回流；不扩大原平台、运行或租户威胁模型。

| 前批依据 | 本轮准确替代 | 保留 |
| --- | --- | --- |
| [#817 SHEIN 数字整链](legacy-shein-listing-closure-retirement-boundary.md) | 纯草稿仍由旧 publishing 包承载、可选扩展继续存在的 KEEP | 当前草稿/预览/JSON/规则本身，当前平台 DTO/API、采集和 worker |
| [#824 支撑闭包](legacy-numeric-listing-support-closure-retirement-boundary.md) | publishing/shein 作为当前组装合同 owner；旧 setter/配置端口 | 当前 Marketplace/Workspace/validator、原持久化与纯规则 |
| [#827 Publishing runtime](legacy-shein-publishing-runtime-closure-retirement-boundary.md) | 最后 60 文件旧 shell、AssemblerConfig/ports、四个非 JSON 私有 map 的复制/应用尾部 | 官方 Kernel/intent/receipt/effects、当前 Package JSON 值、只读离线检查；不恢复旧运行/缓存 |
| [#904 JSON 别名](legacy-shein-draft-json-alias-retirement-v1.md) | 当前 Go 类型和 codec 必须继续位于旧包的 KEEP；两个无实际消费者 setter | canonical JSON、严格拒绝、typed v1.2 / diagnostic v2.2 / binding go-json.v2、原 hash 黄金值全部不变；三个旧 JSON alias 继续拒绝 |
| [#912 策略/合同](legacy-shein-policy-contract-closure-retirement-boundary.md) | 旧桥接定价 alias/装配注入与草稿合同路径 KEEP | 当前 marketplace PricingPolicy 全部纯值行为/测试、API DTO/JSON及规则，不替换为官方 DTO 形状 |
| [#940 敏感词](legacy-shein-sensitiveword-closure-retirement-boundary.md) | publishing 包作为现役清理 adapter 的位置 | 原 content 17 函数、词表/品牌/日文/元数据语义及内存工厂原 owner |
| [#833 内容配置](legacy-numeric-content-management-closure-retirement-boundary.md) | 旧 submit_prep_sensitive_adapter.go 当前路径 KEEP | 同一 content.NewSensitiveWordServiceInMemory(logs)，不建库/恢复旧仓库 |
| [#893 logger](legacy-logger-facade-closure-retirement-boundary.md) | logger guard 继续遍历旧 publishing 根 | 同一显式 Logrus callback、所有当前业务根/全局查询禁入与失败关闭 |
| [#891 预览/下载](legacy-listing-preview-download-closure-retirement-boundary.md) | 旧纯草稿 owner 保留要求 | 当前图像/资产/官方平台及离线检查能力，不恢复 Task 预览或下载链 |

上述替代只在本矩阵生效，用户确认和 Ready 前仍不修改生产。九份旧依据只追加精确前置说明，旧完整正文/数字/勾选/证据保留；其它有日期历史不重写。完整 phase2/external-client inventory、所有 ceilings、`.golangci.yml` 与检查器不因 owner 变动改写。

## 3. 实际消费者与新 owner DAG

使用已有标准 Go AST 工具只读解析基线全部 **3244 tracked Go**（含 cmd/hack/debug、测试和未选择编译源）；decoded imports 确认旧根 60 文件＝39 生产 +21 测试，包外实际 import **15 文件＝7 生产 +8 测试**。其它只出现于 guard/文档的字符串不是生产注入。未运行业务构造器、配置加载、测试用例、浏览器或 provider。

| 当前实际路径 | 消费内容 | 切换 |
| --- | --- | --- |
| marketplace/shein/draft/builder.go | 唯一生产 NewAssembler 调用；传 `AssemblerConfig{}`，BuildRequest 只填 Background/Country/Language | 同一 Builder 直接调用私有 buildLocalPackage；保留快照/纯文本转换、精确 ApprovedAsset 全量替换、清空变体来源图片、前后 ctx.Err、marshal/大小/严格自检 |
| marketplace/shein/validator/{validator,diagnostic}.go | Package/clone/严格解码/纯准备和原离线规则 | 仅 import/Go 类型归属到 offlinepackage；evaluatePackage/版本/envelope/hash/失败分类算法保持 |
| marketplace/shein/workspace/{state,package_template_validation,submit_payload_readiness_checks,submit_readiness_review_checks}.go | 值与 readiness；无 resolver 或远端调用 | 仅 import/type/纯函数 owner 改接；原规则、文案、顺序、blocker/warning/not_evaluated 保持 |
| 包外 8 测试 | draft.Builder、validator/workspace、App HTTP 与 readinessinspect 的真实回归 | 原 fixture/期望/安全断言不变，只改新类型 import/引用 |

新依赖方向：`Listing Record Service → draft.Builder → 私有确定性组装 → offlinepackage 值/codec`；`validator → offlinepackage + workspace + marketplace/shein/publishing 现有规则`；`workspace → offlinepackage`；`offlinepackage → 当前 publishing 纯规则 + publishing/common 共享值 + shein/api/{product,attribute} 当前平台 wire DTO + 标准库/现有 strict-json`。offlinepackage 不反向依赖 draft/workspace/validator、Listing service、App、持久化或任何 provider，防止循环和隐式运行能力。

`publishing/common` 是仍被当前代码消费的共享纯值/投影，不在本轮 RETIRE 根内；当前 SHEIN API DTO 不等于旧 HTTP client，不把它们换成不同形状的官方 API model。`marketplace/shein/publishing` 是已存在的纯规则 owner，全部当前生产/测试不改。新公开表面只保留表中实际消费的 16 个函数、29 个当前 wire 类型、原边界常量和错误；仅内部使用的 helpers 私有化，不新建通用 Assembler/Service/配置/解析器。

### 3.1 两个 owner 的最小职责

- `offlinepackage`：当前 wire model、严格 decoder、JSON clone、draft 结构检查、原 readiness 与纯 payload prepare/validate。它是当前平台输入合同与纯计算 owner，不保存事实，不读取文件/env/时钟/网络，不授权提交。
- `draft`：现有 Builder 接线保留。新增私有 local_package/copy/variants/preview/cleanup helpers，只承接旧空配置的真实路径；不接收可选 resolver、AI、定价策略或品牌授权配置。日志沿原 `Builder.LogFor` 显式传递，内存内容工厂通过独立 sensitive_word_adapter.go 复用，原 callback 原样透传。
- 当前 content/authorizedbrand 的纯清理和合法品牌规则保持原 owner；私有清理 helper 可以沿原规则处理其显式参数，但生产 Builder 继续传与旧实现相同的 Background，不把请求 context 中的任意值升级成品牌授权。

### 3.2 退出的能力边界

旧 CategoryResolver/AttributeResolver/SaleAttributeResolver、SizeAttributeHeaderResolver 及其 input/result、TitleOptimizer/TextGenerator、公开 Assembler/AssemblerConfig/BuildRequest、PricingPolicy type alias、两个 setter、Apply*Resolution 及私有赋值 cache 全部退出；不在新 owner 重建同名接口/配置或 re-export。全局 AI/Google 文本/图像与原平台纯定价算法不退。

本地确定性标题/描述的既有优先级、prompt 污染拒绝/纯规则提取、结构回退、缺英文描述时原失败 metadata/ReviewNote、来源审阅 metadata、默认 SKC/SKU/价格格式/仓库/站点和空值保留。当前空定价策略继续直接复用 `marketplace/shein/publishing.PricingPolicy{}.Apply`（包括默认币种和复制行为），不能简化为直接返回原 price。变体只承接当前 nil-resolution 分支；不搬回通过远程解析结果分组/拆分的旧配置能力。Preview 的纯值映射及有效 numeric/composition/图片/size 投影规则可以按实际调用闭包抽取为私有纯函数，不能连带恢复 resolver 应用/执行入口。

## 4. JSON、hash、版本和持久化边界

### 4.1 当前 wire 合同原样

以下 29 类型是 Package/结构诊断的当前可达值图（不是旧 runtime API）；在新 owner 重新定义，**不是 alias**。公开字段名/顺序/类型/tag、数值精度、null/空集合、omitempty 和嵌套平台 DTO 形状保持：

`AttributeResolution`, `AttributeValueCandidate`, `CategoryResolution`, `CategorySemanticValidation`, `CategorySuggestion`, `DraftPayloadIssue`, `FinalDraft`, `ImageDraft`, `LocalizedText`, `Package`, `PendingAttributeCandidate`, `PricingReview`, `PricingRule`, `RequestDraft`, `ResolutionCacheInfo`, `ResolvedAttribute`, `ResolvedSaleAttribute`, `SKCPackage`, `SKCRequestDraft`, `SKUDraft`, `SKUPriceReview`, `SaleAttributeCandidateInfo`, `SaleAttributeResolution`, `SaleAttributeTemplateOption`, `SitePrice`, `SizeAttributeReview`, `SourceVariantDimension`, `StockInfo`, `TitleDiagnostics`.

严格 decoder 继续使用标准 encoding/json 和现有 `sigs.k8s.io/json`：2MiB 输入/normalized envelope 界限、64 深度、UTF-8/UTF-16 拒绝、未知/重复字段、非对象/尾部输入、非法/非有限/非零数下溢拒绝及当前错误 sentinel 文本保持。三个旧 alias request_draft/preview_product/final_draft 与旧 execution 字段仍拒绝；metadata 内同名普通数据仍允许，不新增自定义 Marshal/Unmarshal、双读或迁移。

四个未导出且不进入 JSON 的 `SaleAttributeResolution.{skcAssignments,skuAssignments,skcValueAssignments,skuValueAssignments}` 只有旧 apply_resolution/clone 尾部读写，当前唯一生产 builder 不填，包外消费者不能赋值，严格 persisted 输入也不能注入。它们不抽取；ClonePackageForPersistence 保留原 JSON 深复制及全部公开字段/嵌套 map/slice 语义，只退出四个私有 map 的追加复制和专属 helper。公开 SKCValueAssignments/SKUValueAssignments 等当前 wire 字段保持，不按名字连删。旧 private clone 的 KEEP 在§2明确替代，不能用新私有 cache 或反射写回保活。

### 4.2 hash 与三项版本不变

`validator.RuleVersion = shein.offline_package.v1.2`、`DiagnosticRuleVersion = shein.offline_package.v2.2`、`BindingVersion = shein.persisted-input.go-json.v2` **保持原值**。Go package owner 变动不改变当前 JSON/规则语义；若真实测试证明语义不能保持，停下报告具体差异，不能改 golden 或自行 bump 版本放行。

diagnostic 的 envelope 字段顺序 binding_version/marketplace/site/action/rule_version/package、Marshal 和 SHA-256 算法不变；request expected digest、Freshness/read/evaluated 绑定、NOT_EVALUATED、ExactApprovedAssetValidator 仅移除已知已验资产 concern 的原行为保持。原 diagnostic_test.go **零改**，两项既有黄金值原样：

- 空 canonical vector：`sha256:82ab67d23adad2d3eeb38dbb22309f364c630e765a75e21891a575fa46f546b3`
- SDK canonical vector：`sha256:10e8edc0efe4eac90d9a60897672d6dd5285350e31c92753c07eaa2f304e7c0d`

Record 的 InputHash/ProductHash/AssetInventoryHash/PackageHash/DiagnosticHash 与当前版本、排序/规范化/hash 算法不改。保持当前 readiness 文案/字段路径，即使普通字符串含 listingkit/final_review 等历史字样也不据名称改值；这些是数据/诊断，不是恢复旧 alias。

### 4.3 不转移 durable fact owner

| 边界 | owner | 本批变化 |
| --- | --- | --- |
| 精确 Catalog/ApprovedAsset | 当前 Product/Asset owner，原组织/版本/批准范围 | 零改；来源图片不能混回草稿 |
| 本地不完整 SHEIN 草稿计算 | 既有 draft.Builder，当前同步有界计算 | 只消除旧 Assembler 调用，输入/输出/取消边界保持 |
| 不可变 Record/Payload/Diagnostic/版本与 hash | listing/record Service + integration/persistence/listing/record 原 owner | 表/schema/索引/事务/权限/幂等、读取/保存均零改；只存相同当前字节，不扫描/重写已有记录 |
| 纯 codec/离线诊断 | offlinepackage/当前 validator/workspace；计算结果不授权平台 mutation | 仅 Go owner/type import 转换；没有缓存/DB/env/provider/隐式时钟 |
| 当前官方 Submission/Kernel/intent/receipt/UNKNOWN、Store观察 | 原 Domain/Integration/App owner | 零改；不接管旧提交、重试、lease/recovery |
| 独立 SHEIN 登录/Profile/Store同步 | sheinlogin 原 owner、fence/执行归属/UNKNOWN | 全部原生产/测试保持，不以本设计探查/接管/清理运行事实 |

没有新增事务/事件/幂等身份、跨库一致性、retry/recovery owner，故不引入 outbox/Saga/平台。当前失败、响应丢失、重启、并发、取消、撤权仍由原 owner 负责；旧 Go 程序的历史实例/记录/凭据存在性 UNKNOWN，源码退出不证明它们不存在。

## 5. EXTRACT / RETIRE 精确矩阵与完整闭包

Legacy decision: **EXTRACT + RETIRE**。Reusable behavior：当前 wire/codec/离线纯规则与空配置本地组装。Current owner：offlinepackage + 既有 draft；既有 publishing 平台规则/content/common 直接复用。Cutover/deletion condition：15 import 消费者改接、当前字节/hash/版本和原有效规则保持、旧根/ports/四私有状态缺席、原检查精确转换、完整CI/最终源码复核满足；不包含运行资源删除。

| 旧源族 | 抽取 | 退休 |
| --- | --- | --- |
| model/source_dimensions/resolution_cache_store | §4.1的29当前值类型、field/tag/order | BuildRequest、四私有 map、旧 owner；CacheInfo 是当前 JSON 元数据，不新建缓存 |
| persisted_input/package_persistence_clone/model_contract | 原严格 codec、JSON深复制、DraftPayloadOf/ValidateDraftPayload | 私有追加 clone、SetDraftPayload/SetPreviewPayload 等无实际消费者 setter |
| submit payload/images/normalization/site-sku/policy、sale/readiness/coverage | 16当前公开函数及其实际纯调用闭包；复用已存在 marketplace 规则；内部 helper 私有化 | 只服务未消费运行扩展的公开便利表面；不复制原已在正确 owner 的算法 |
| assembler/variant_grouping | 空配置唯一真实路径到 draft.local_package/local_variants，原资产/默认/来源metadata/字节结果 | 公共工厂/interface/config、resolver调用、非nil-resolution调度/分组与任意旧BuildRequest字段 |
| listing_copy/title/description | 确定性清理、优先级、规则提取/回退、原未就绪结果和数据 | AI prompt/Generate调用、enrich/LLM提取与TextGenerator、旧文本生成错误接口 |
| preview/display numeric/composition/image_draft | 当前预览纯值映射及其实际调用闭包到 draft 私有 helpers；保留平台值行为 | 不抽取旧 runtime/未来可选入口；不产生新的 PreviewService/API |
| sensitive_content/submit_prep_sensitive_adapter/logger | 当前内存 content adapter 和显式日志原规则至 draft；原工厂函数体/参数原样 | 旧 loggerDependency/公开清理装配入口、旧包依赖；不恢复租户词库仓库 |
| interfaces/ai_contracts/pricing/size_attribute/apply_resolution/resolver_cache_clone | 原市场定价/尺码纯规则在现有 owner 保持；Preview值复制按上行 | 专属接口、alias、SizeHeaderResolver input/result/执行、私有assignment应用及克隆 |

### 5.1 新目标布局与停止边界

新 offlinepackage 9生产+6测试文件；既有 draft 增7私有生产+5测试文件。这些文件名是冻结布局，不是自由扩写授权；源码行为仅以§3–§4和§6为准。当前 publishing/common、shein/api/{attribute,product}、content/authorizedbrand 和 marketplace/shein/publishing 生产/原测试保持；不顺手退休它们，不动当前官方连接/发布/Store与采集。

### 5.2 原测试行为处置

60旧路径物理退出；21旧测试文件按当前行为转到正确 owner。保留原 codec/semantic fields、canonical field precedence、结构/size/图片/价格/ready分类、敏感词/品牌/日文、预览纯值与默认组装的独立 fixture/期望。只允许 package/import/必要类型限定、私有 helper 名和旧 fixture 装配形式改接，不通过删业务断言、改错误/数值/JSON/hash golden 收口。

| 原测试族 | 明确处置 |
| --- | --- |
| persisted_input/semantic_fields/model_contract | 当前严格拒绝/字段/hash相关断言保留；只退 TestSetDraftPayloadWritesCanonicalField 与 TestCanonicalSettersHandleNil 的旧setter接口，字段行为仍直接构造验证。TestPackageSemanticFieldNamesRemainUsable只把setter fixture改成直接字段，期望不变 |
| package_persistence_clone | TestClonePackageForPersistencePreservesCanonicalFieldsAndPrivateAssignmentsWithoutSharingState 保留全部公开 canonical/深复制断言，删除四私有map fixture/断言；TestSaleAttributeResolutionPrivateCloneSchemaIsExplicit 转为四私有字段必须缺席的准确检查。原公开map/slice深复制不得减少 |
| persisted-input private/schema audit | 同一 rejected JSON/大小/深度/Unicode/数值/额外字段 fixture/期望保持。四私有nil字段访问转为反射确认缺席；custom decoder allowlist只改本包PkgPath到新owner，其余decoder/平台类型与失败关闭保持 |
| sale_attribute_readiness/submit_readiness/variant_image_coverage、submit payload/images/site-sku/image clone | 全部有效规则和期望保留到新 owner；旧纯 helper 内部化只改调用名，原平台当前规则测试零改 |
| preview_adapter | 原numeric/composition/quantity/去重纯映射期望保留到 local_preview_test；不借测试复活 resolver/AI；当前typed JSON的相关值仍可表达 |
| listing_copy | 清理/结构英文标题/规则提取/短标题无AI时的原结果保留。只退休名称含 EnrichesRealDoorCurtainTaskTitle、EnrichesShortStructuredTitleWithLLMAddition、GeneratesMissingEnglishDescriptionWithAI、LeavesPromptTitleUnresolvedWhenLLMReturnsPromptLikeTitle、UsesLLMWhenRuleExtractionCannotRecover 的旧AI注入用例；KeepsLongStructuredTitleWithoutLLMAddition只退假AI调用计数，原输出期望保留 |
| assembler/size | 保留 CreatesDefaultSKCWhenCanonicalVariantsMissing、DoesNotPropagatePromptTextIntoSKCNames、Marks1688LLMOnlyFactsForReview 到当前Builder。其它仅检验旧resolver/非nil分组/启用定价/size-header调用的用例退休；对应已存在市场纯规则和默认/Preview size值测试保留 |
| authorized_brand/submit_prep | 纯清理与品牌规则期望保留到私有 local_cleanup 或原content owner；仅退旧AssemblerConfig/BuildRequest.Context的注入装配与旧Prompt静态入口测试。不得改原content 17函数/字段/词表测试 |
| pricing bridge/compatibility_guard/apply_resolution专属测试 | alias/旧owner及private assignment应用用例退出；现有市场定价全部测试保持，新完整原guard承担旧根缺席及当前owner边界保护 |

### 5.3 未来119工件准确清单

总计：旧根60、当前import消费者15、新目标27、原检查3、文档14。当前设计实际仍只有三文档，无生产或测试源码修改。完整清单不授权超出上述语义边界：

<details><summary>119 唯一路径与处置</summary>

| 路径 | 处置 |
| --- | --- |
| `docs/architecture/README.md` | CURRENT_DOC |
| `docs/architecture/legacy-listing-preview-download-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-logger-facade-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-numeric-content-management-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-numeric-listing-support-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-shein-draft-assembly-closure-retirement-boundary.md` | CURRENT_DOC |
| `docs/architecture/legacy-shein-draft-json-alias-retirement-v1.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-shein-listing-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-shein-policy-contract-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-shein-publishing-runtime-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/legacy-shein-sensitiveword-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY_NOTE |
| `docs/architecture/project-boundaries.md` | CURRENT_DOC |
| `docs/refactoring/legacy-register.md` | CURRENT_DOC |
| `internal/app/httpapi/shared_regression_deterministic_test.go` | REWIRE_CURRENT_IMPORT |
| `internal/app/httpapi/shein_records_integration_test.go` | REWIRE_CURRENT_IMPORT |
| `internal/integration/readinessinspect/integration_test.go` | REWIRE_CURRENT_IMPORT |
| `internal/marketplace/shein/README.md` | CURRENT_DOC |
| `internal/marketplace/shein/draft/builder.go` | REWIRE_CURRENT_IMPORT |
| `internal/marketplace/shein/draft/builder_test.go` | REWIRE_CURRENT_IMPORT |
| `internal/marketplace/shein/draft/local_cleanup.go` | NEW_MINIMAL_LOCAL_ASSEMBLY |
| `internal/marketplace/shein/draft/local_cleanup_test.go` | NEW_MINIMAL_LOCAL_ASSEMBLY |
| `internal/marketplace/shein/draft/local_copy.go` | NEW_MINIMAL_LOCAL_ASSEMBLY |
| `internal/marketplace/shein/draft/local_copy_test.go` | NEW_MINIMAL_LOCAL_ASSEMBLY |
| `internal/marketplace/shein/draft/local_package.go` | NEW_MINIMAL_LOCAL_ASSEMBLY |
| `internal/marketplace/shein/draft/local_package_test.go` | NEW_MINIMAL_LOCAL_ASSEMBLY |
| `internal/marketplace/shein/draft/local_preview.go` | NEW_MINIMAL_LOCAL_ASSEMBLY |
| `internal/marketplace/shein/draft/local_preview_test.go` | NEW_MINIMAL_LOCAL_ASSEMBLY |
| `internal/marketplace/shein/draft/local_preview_values.go` | NEW_MINIMAL_LOCAL_ASSEMBLY |
| `internal/marketplace/shein/draft/local_variants.go` | NEW_MINIMAL_LOCAL_ASSEMBLY |
| `internal/marketplace/shein/draft/local_variants_test.go` | NEW_MINIMAL_LOCAL_ASSEMBLY |
| `internal/marketplace/shein/draft/sensitive_word_adapter.go` | NEW_MINIMAL_LOCAL_ASSEMBLY |
| `internal/marketplace/shein/offlinepackage/clone.go` | NEW_CURRENT_CONTRACT_OR_RULE |
| `internal/marketplace/shein/offlinepackage/clone_test.go` | NEW_CURRENT_CONTRACT_OR_RULE |
| `internal/marketplace/shein/offlinepackage/decode.go` | NEW_CURRENT_CONTRACT_OR_RULE |
| `internal/marketplace/shein/offlinepackage/decode_test.go` | NEW_CURRENT_CONTRACT_OR_RULE |
| `internal/marketplace/shein/offlinepackage/doc.go` | NEW_CURRENT_CONTRACT_OR_RULE |
| `internal/marketplace/shein/offlinepackage/draft_shape.go` | NEW_CURRENT_CONTRACT_OR_RULE |
| `internal/marketplace/shein/offlinepackage/draft_shape_test.go` | NEW_CURRENT_CONTRACT_OR_RULE |
| `internal/marketplace/shein/offlinepackage/model.go` | NEW_CURRENT_CONTRACT_OR_RULE |
| `internal/marketplace/shein/offlinepackage/model_test.go` | NEW_CURRENT_CONTRACT_OR_RULE |
| `internal/marketplace/shein/offlinepackage/payload.go` | NEW_CURRENT_CONTRACT_OR_RULE |
| `internal/marketplace/shein/offlinepackage/payload_defaults.go` | NEW_CURRENT_CONTRACT_OR_RULE |
| `internal/marketplace/shein/offlinepackage/payload_images.go` | NEW_CURRENT_CONTRACT_OR_RULE |
| `internal/marketplace/shein/offlinepackage/payload_test.go` | NEW_CURRENT_CONTRACT_OR_RULE |
| `internal/marketplace/shein/offlinepackage/readiness.go` | NEW_CURRENT_CONTRACT_OR_RULE |
| `internal/marketplace/shein/offlinepackage/readiness_test.go` | NEW_CURRENT_CONTRACT_OR_RULE |
| `internal/marketplace/shein/validator/diagnostic.go` | REWIRE_CURRENT_IMPORT |
| `internal/marketplace/shein/validator/validator.go` | REWIRE_CURRENT_IMPORT |
| `internal/marketplace/shein/validator/validator_test.go` | REWIRE_CURRENT_IMPORT |
| `internal/marketplace/shein/workspace/package_template_validation.go` | REWIRE_CURRENT_IMPORT |
| `internal/marketplace/shein/workspace/package_template_validation_test.go` | REWIRE_CURRENT_IMPORT |
| `internal/marketplace/shein/workspace/state.go` | REWIRE_CURRENT_IMPORT |
| `internal/marketplace/shein/workspace/submit_payload_readiness_checks.go` | REWIRE_CURRENT_IMPORT |
| `internal/marketplace/shein/workspace/submit_payload_readiness_checks_test.go` | REWIRE_CURRENT_IMPORT |
| `internal/marketplace/shein/workspace/submit_readiness_review_checks.go` | REWIRE_CURRENT_IMPORT |
| `internal/marketplace/shein/workspace/submit_readiness_review_checks_test.go` | REWIRE_CURRENT_IMPORT |
| `internal/publishing/shein/ai_contracts.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/apply_resolution.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/assembler.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/assembler_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/authorized_brand_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/compatibility_guard_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/description_resolution.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/display_attribute_composition.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/display_attribute_numeric.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/doc.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/image_draft.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/interfaces.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/listing_copy.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/listing_copy_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/logger_port.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/model.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/model_contract.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/model_contract_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/package_persistence_clone.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/package_persistence_clone_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/persisted_input.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/persisted_input_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/preview_adapter.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/preview_adapter_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/pricing.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/pricing_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/resolution_cache_store.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/resolver_cache_clone.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/sale_attribute_assignments.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/sale_attribute_custom_values.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/sale_attribute_readiness.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/sale_attribute_readiness_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/sale_attribute_value_matcher.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/sale_attribute_value_matcher_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/semantic_fields.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/semantic_fields_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/sensitive_content_sanitizer.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/size_attribute.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/size_attribute_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/source_dimensions.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/submit_image_policy.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/submit_image_policy_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/submit_payload_images.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/submit_payload_images_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/submit_payload_normalize.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/submit_payload_policy.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/submit_payload_policy_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/submit_payload_prepare.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/submit_payload_validation_prepare_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/submit_prep_sensitive_adapter.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/submit_prep_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/submit_readiness_status.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/submit_readiness_status_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/submit_site_sku_policy.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/submit_site_sku_policy_test.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/template_matcher.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/title_resolution.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/variant_grouping.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/variant_image_coverage.go` | RETIRE_OLD_ROOT |
| `internal/publishing/shein/variant_image_coverage_test.go` | RETIRE_OLD_ROOT |
| `tests/import_boundaries_test.go` | EXACT_CHECK_CONVERSION |
| `tests/listingkit_backend_retirement_test.go` | EXACT_CHECK_CONVERSION |
| `tests/validator_contract_boundary_test.go` | EXACT_CHECK_CONVERSION |

</details>

## 6. 验证与验收

当前阶段只做只读消费者/owner/原检查调查、119路径/新旧布局/依赖DAG、历史blob后缀/链接/diffcheck和适用原文档检查，准确设计HEAD原CI、适用独立设计复核另列。没有运行旧Assembler/业务配置加载/应用/浏览器/provider；当前原CI成功只证明本设计候选的检查，不冒充实施/产品验收。

确认Ready后，同一批先在原聚合新增一个完整闭包guard取得真实RED，锁定当前Builder代表性受控资产/来源/空值/标题污染/默认SKC字节与hash独立fixture，再抽取当前owner/改接15消费者/整退60旧文件→GREEN。未来代表fixture在切换前取自准确冻结main合成输入、提交固定期望，切换后只对固定字节/hash断言，不能同进程调用旧包作oracle或保留测试runtime兼容副本。复用现有go test/原fixture，不建runner/验收平台/故障矩阵。

必要原验证：新owner codec/clone/规则、原Builder受控ApprovedAsset/确定性/取消、validator/diagnostic原两golden与全部安全拒绝、workspace、原App/readinessinspect合成测试与原guard；当前入口只编译。完整原CI留稳定候选。源码/fixture执行/编译/PRCI/mainCI与真实运行/产品验收分列，后两项NOT_RUN，不新增试用前置；不读现有.env/凭据/DSN/记录或操作运行资源。

### 6.1 原检查的准确转换

1. `tests/listingkit_backend_retirement_test.go` 仅新增一个 `TestLegacySheinDraftAssemblyClosureIsRetired` 完整guard：旧根/decoded import/旧ports/alias/四私有字段必须缺席，29 wire类型与新owner/当前Builder/codec/版本/独立worker必需声明存在，数据owner无provider/DB/App/隐式时钟/文件I/O。复用原AST/trackedFiles/失败关闭helpers，按owner判断，不全局禁Package/Build等通名。
2. 同文件 `TestSheinCanonicalDraftContractIsRetained` 只把扫描根转为offlinepackage，旧必需声明表只删除SetDraftPayload/SetPreviewPayload，Package三个canonical字段/RequestDraft/FinalDraft/codec/clone与原AST算法保持；两个setter的值赋值能力由原字段/新guard与有效测试继续保护，不保留兼容函数。
3. `TestListingKitTaskBackendSourcesAreRetired` 只改三条必需读取路径：submit_site_sku_policy.go→offlinepackage/payload_defaults.go，sensitive_content_sanitizer.go→draft/local_cleanup.go，submit_prep_sensitive_adapter.go→draft/sensitive_word_adapter.go；原 DefaultSiteListByRegion/content.CleanupContext/ensureSanitizerContext/authorizedbrand.FromContext/内存工厂字符串断言全部保留，所有旧缺席/其它KEEP/禁入列表不改。
4. `TestNumericContentConfigurationClosureIsRetired` 只把原须存在的publishing/shein扫描项换成offlinepackage和draft两个当前根，及内存adapter路径改接；原retired声明全集、schema/Store/content/失败关闭与其它断言保持。`assertLegacyLoggerFacadeRetired` 只把businessRoots旧publishing/shein项替换这两个当前根，原日志工件缺席/全局查询禁入/其它业务根和AST检查保持。其它原聚合函数/表/文件列表（含全部既已退休对象）零改。
5. `tests/import_boundaries_test.go` 仅四函数：TestPublishingSheinNonAPISheinImportsStayAllowlisted首段扫描移到现役marketplace/shein，content只许draft/local_cleanup.go、draft/sensitive_word_adapter.go及直接验证清理的draft/local_cleanup_test.go；其余旧实现依赖保持deny，后半managed逻辑逐字保持。TestPublishingSheinOpenAIImportsStayAllowlisted扫描同一当前根，旧测试allow表清空，原禁入算法保持。TestPublishingSheinRuntimeBaseAPIClientAliasStaysRetired扫描同一当前根，原禁alias/读错失败保持。TestPublishingSheinSubmitPrepUsesOnlySensitiveWordAdapter只改读取路径到draft/sensitive_word_adapter.go，其原factory名称/func(string)*logrus.Entry类型、原样传参、直接内存owner调用/全部AST失败关闭断言不改。其它函数、命令deny、helpers零改，不加missing-skip。
6. `tests/validator_contract_boundary_test.go` 的TestValidatorContractImportBoundary只将validator正向allowed old publishing/shein一项替为offlinepackage；在同一现有map中增加新offlinepackage根的精确纯依赖白名单（标准库bytes/encoding/json/errors/fmt/io/math/sort/strconv/strings/time/unicode/utf8、现有sigs.k8s.io/json、当前marketplace/shein/publishing、publishing/common、shein/api/product与attribute）。原遍历/存在性/count/解码/禁止时钟及所有负例/helpers保持；不改无关root或把provider/运行能力加入allowed。
7. 15实际consumer源码只改import/类型/调用owner；生产Builder去旧工厂的一处调用由私有同结果组装替换，已存在资产/取消/格式化/大小/严格自检全保留。validator算法和三项版本、diagnostic_test.go及其golden **零改**，其余无oldimport的current文件零改。21原旧测试的精确行为边界见§5.2；不改fixtures/期望掩盖行为差异。
8. 完整phase2及external-client inventory、历史/current行/计数/ceilings、`.golangci.yml`、`tests/depguard_config_test.go`、`tests/architecture_docs_test.go`、workflow/依赖/保护 **零diff**；现无旧根的CURRENT concrete consumer行需要校准，不删历史目标。新guard承担两个新owner的同等禁入保护；旧lint永久deny行保留，不借整根退出放宽。
9. 文档仅未来清单的14项：九旧依据只前置准确替代注；project-boundaries只校准当前pure owner段/引用，旧历史完整；marketplace/shein/README明确新职责；README/register保留最终main完整blob后缀。#301更新前新读，当前段以外从“以下保留 #726 已合并批次与历史证据，不扩大本轮权限。”起逐字保留，正文≤65536字符且OPEN。

真实原CI如暴露冻结矩阵外遗漏，先查同根sibling、分类并提出准确最小补丁，取得用户明确例外后才应用；本节不授权删检查、降Must、重跑绕过门禁或临时改版本/golden。

## 7. Review、确认与停止点

- 当前一个Writer/一个主要设计PR，只修改三文档。准确设计HEAD原CI/适用独立设计复核满足，交用户确认本§1–§7/§6.1；Issue/PR明确IMPLEMENTATION_READY/Ready并读回后，才准同一PR整批119路径实施。此前SOURCE_NOT_STARTED/NOT_READY。
- 起始已同步main；最终候选前再核对实际main组合，只处理真实组合差异，不为始终最新反复merge。已绑定且未变的证据可以明确复用，不冒充新SHA执行结果。
- Independent Architecture Review最多两轮正常检查，finding先按AGENTS分类并指出受影响当前Must/后果/层级；只有真实BLOCKER重开已冻结设计。实现后一次集中源码最终复核，修复只复核实际增量；设计复核/CI/另一AI都不等于产品验收。
- GitHub复核启动失败至多一次重试；**未经用户本轮授权不默认新增AI互验线程**。此前#958及其它批的设计/源码授权不覆盖本设计或未来实现，不新增实现者/runner/治理平台，不重复触发运行中评审。
- 完整实现/准确原CI/最终源码复核满足后另交用户准确HEAD正常expected-head合并授权，不绕过/强推/改保护。合并后只认实际mergeSHA对应push/main整runcompleted SUCCESS，设计/PRCI不替代。
- #301保持OPEN。当前能力、独立SHEIN登录与Store同步保持；workflow/完整inventory/运行实例/数据/凭据/目录/进程不动，不增加内部试用前置。真实运行/产品验收NOT_RUN；合并/部署/关单/真实操作另授权。
