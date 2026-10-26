# #301 旧 SHEIN 草稿支撑层完整退休设计

> **Independent Architecture / DESIGN_ONLY / DESIGNING / SOURCE_NOT_STARTED / NOT_READY**。本阶段仅本文、架构 README、Legacy register 三文档。用户同意先做四旧包完整设计，不机械拆批；适用设计复核完成、用户确认 §1–§7（含 §4.2、§6.1）并在 Issue/PR 写入读回 IMPLEMENTATION_READY / Ready 后，才在同一主要 PR 整批实施。起始实际 main `5306d3f58212ed0a92575b985aa9c730894bea64`；#964 正常合并，准确 push/main CI40312717064 attempt1 SUCCESS。当前无生产/测试源码修改。

稳定设计基线更新为实际 main f5a1faa65ae27d65b3ba3c1abd07a1e95729024（#967）。相对起始5306唯一新增为Amazon扩展四TS、三测试、一份扩展文档及一份来源测试共9文件；四旧包/68路径/全部原检查/README-register/依赖零交集。原字段/消费者/规则调查可复用，最新组合由准确设计HEAD原CI另行验证，不接管该owner。

## 1. 用户结果与本批范围

使用者是维护当前 SHEIN 草稿、预览/校验、有效清理及独立登录/Store 同步的开发者。用户仍沿原入口生成和查看同样的草稿/预览；本批结果是这条路径不再依赖四个旧包，无新 UI、平台提交或内部试用步骤。

| 一次完整处理 | 有效行为归属 | 整退对象 |
| --- | --- | --- |
| 商品/SKC/SKU/属性数据 | `marketplace/shein/offlinepackage/previewdata`，22真实值合同 | `internal/shein/api/product`、`internal/shein/api/attribute` |
| 文本清理 | 现有 `marketplace/shein/publishing` 的无状态纯规则；draft 显式传值 | `internal/shein/content` 整包、内存 Service/配置/日志外壳、adapter |
| 品牌输入 | 当前纯规则需要的 SourceBrand、Enabled、Name、NameEn 值；普通生产保持空品牌输入 | `internal/shein/authorizedbrand` 整包、Context 存取及私有空 Context 尾部 |

未来同一 PR **68 唯一路径**：15退出文件、24值合同消费者、5清理私有调用/测试、4新纯值/规则/测试文件、3原检查文件、4当前文档、13历史顶注。当前实际仅三设计文档。完整路径见 §5.3。

Must：完整切换与删除；22类型字段/tag/顺序/指针/空值、当前 Package JSON/hash/版本/严格解码、现有有效清理/品牌规则、Builder 外层取消与 ApprovedAsset、独立 SHEIN 登录/Store 同步保持。Should：复用当前 owner、Go strings/regexp/json 和原有效测试。Out of Scope：新授权/品牌查询/词表管理、AI/provider、UI、新状态/schema/事务/恢复/迁移、workflow/任何 inventory/.golangci/依赖/保护、既有实例/数据/凭据/配置/输出/进程；不增加试用前置。无新 Accepted Risk；物理状态 UNKNOWN，不探查。Threat Model 仅覆盖误删当前合同/规则、JSON/hash 漂移、循环依赖、旧 owner/Context/Service 回流与放宽原检查，不扩全局安全目标。

## 2. Design Basis、Authority 与前批 KEEP

**Independent Architecture**：22公开 Go 值合同归属变化、四旧包退出与跨草稿/规则/校验边界，预计68路径；文本清理由旧可变 Service 转成当前纯值规则，必须先冻结最小输入及删除矩阵。持久化事实/权限/授权/租户/幂等/UNKNOWN/recovery owner 不变。UI/IA/交互不变，Figma Authority N/A。

依据：[Greenfield](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[目标架构](project-target-architecture.md)、[项目边界](project-boundaries.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、当前 [SHEIN 供货链](my-supply-chain-shein-v1.md)与用户本轮范围决定。复用当前 offlinepackage/publishing/draft/validator，沿用 Go 标准库算法；没有通用词表平台、序列化框架、旧 Service 包装或兼容层。

| 前批 KEEP | 本次准确替代 | 继续保持 |
| --- | --- | --- |
| #817 / #827 / #833 清理/内容支撑；#893 logger façade | content 内存工厂/adapter、Context 桥和专属日志外壳 | 当前有效清理规则；现有 draft assembler 日志、授权/审计/错误日志各 owner |
| #900 / #940 内存词表、17函数与三个 CleanupContext 槽位 | SensitiveWordService/Config、mutex/logger、内部配置 JSON 及 Context 存取的实现 KEEP | 默认四语言词集、显式自定义词/去重/副本/并发只读、品牌与清理算法，按 §4.2转接纯值 |
| #929 Resolved / WithResolved / FromContext | 四字段 Resolved 与 Context 桥的旧 owner KEEP | Enabled 门控、Name/NameEn 白名单规则；预览 authorized_brand_code 原路径不变 |
| #912 / #926 有效 SHEIN DTO；#960 / #961 当前离线合同 | 原 api/product 21值及其 import 不动的 KEEP；旧 adapter factory | 同一22值的当前 wire、有效校验/规则/图像/价格，pure leaf DAG |
| #964 CustomAttributeRelation | 该真实关系值留 api/attribute 的旧路径 KEEP | 原两个 int64/tag/顺序，Amazon capturedata 17值/5方法及旧 model 禁令完全不动 |
| Module Target Mapping | 旧 SHEIN 草稿支撑 owner 现状 | 其余目标、历史/计数与独立 worker 现有读取 owner |

仅这些准确边界被替代。§5列13旧依据在未来实施时只加顶注，完整原 Git blob/数字/勾选/证据保留；设计阶段不预写退休完成。

## 3. 实际消费者与依赖方向

基线3185 tracked Go 已用标准 Go AST 只读解析，包含测试、未选择编译源及 hack/debug；import 解码且区分 alias/局部遮蔽，另核对嵌套字段、方法、函数签名和私有调用，不以文本无搜索结果自动判闲置。

- `api/product`：24实际 import＝13生产＋11原测试，均为当前 draft/offlinepackage/publishing/validator/workspace；无 HTTP client 或独立登录消费者。
- `api/attribute`：4生产 import；3属于上述24消费者，另一项是将退出的 api/product/model.go。只有 CustomAttributeRelation，两个 int64字段，三个孤立属性DTO已由#964退出。
- `content`：生产仅 draft/local_cleanup.go 与 sensitive_word_adapter.go；另有一个包外规则测试。旧包六生产文件＋两测试。
- `authorizedbrand`：生产 import 只在 content/CleanupContext 和 draft 读取 FromContext；所有3185 Go未发现生产 WithResolved 选择器/函数值消费者，只有测试写 Context。旧包二生产文件＋一 Context 存取测试。
- `local_package.go` 自建 `context.Background()`，三处仅传到旧清理链；local_copy.go 的 runtimeCtx/logs 仅为旧清理参数。`local_copy_test.go` 六处私有 buildLocalCopy 调用一并改接。外层 Builder.Build 的真实 ctx.Err() 前后检查、ApprovedAsset 置换、尺寸限制/严格解码零改。

**原建议必须纠正的架构点**：offlinepackage/source_values.go 等已依赖 publishing.Price/规则；publishing 也消费当前商品DTO。22值若直接放 offlinepackage 根，会形成 `offlinepackage → publishing → offlinepackage` 循环。采用域内 **previewdata 子叶子**，不是父包、跨平台 common 或第二 canonical Product。

新 DAG：`offlinepackage → publishing → previewdata`，`offlinepackage → previewdata`，`draft → offlinepackage + publishing + previewdata`。previewdata 只有22真实struct、**零 import/方法/初始化/变量状态**；publishing 新文本文件只依赖 strings/regexp，无旧包/父offlinepackage/provider/日志/DB/文件/时钟；draft 组装仅显式调用纯规则，不保留 factory/interface/旧 Service。新路径不含仓库保留的 result/logs/tmp 等产物段。

<details><summary>24值合同消费者：未来仅 import/限定名变化</summary>

| 文件 | 类别 | 当前商品选择器 |
| --- | --- | --- |
| `internal/marketplace/shein/draft/local_preview.go` | 生产 | Extra, LanguageContent, Product, ProductAttribute, SizeAttribute |
| `internal/marketplace/shein/draft/local_preview_test.go` | 原测试 | SizeAttribute |
| `internal/marketplace/shein/draft/local_preview_values.go` | 生产 | CostInfo, ImageDetail, ImageInfo, LanguageContent, PriceInfo, ProductAttribute, SKC, SKU, SaleAttribute, SiteInfo, StockInfo |
| `internal/marketplace/shein/offlinepackage/clone.go` | 生产 | Product |
| `internal/marketplace/shein/offlinepackage/clone_test.go` | 原测试 | ImageDetail, ImageInfo, Product |
| `internal/marketplace/shein/offlinepackage/model.go` | 生产 | Product, SizeAttribute |
| `internal/marketplace/shein/offlinepackage/model_test.go` | 原测试 | Product |
| `internal/marketplace/shein/offlinepackage/payload.go` | 生产 | Product |
| `internal/marketplace/shein/offlinepackage/payload_defaults.go` | 生产 | Product, QuantityInfo, SKU, SaleAttribute, SiteDetailImageInfo, SiteInfo, StockInfo |
| `internal/marketplace/shein/offlinepackage/payload_images.go` | 生产 | ImageDetail, ImageInfo, Product, SKC, SiteDetailImageInfo |
| `internal/marketplace/shein/offlinepackage/payload_test.go` | 原测试 | ImageDetail, ImageInfo, PriceInfo, Product, ProductAttribute, QuantityInfo, SKC, SKU, SaleAttribute, SiteInfo, SizeAttribute, StockInfo |
| `internal/marketplace/shein/offlinepackage/readiness.go` | 生产 | ImageInfo |
| `internal/marketplace/shein/offlinepackage/readiness_test.go` | 原测试 | ImageDetail, ImageInfo, Product, SKC |
| `internal/marketplace/shein/publishing/payload_image_policy.go` | 生产 | DetailImage, ImageDetail, SiteDetailImageInfo |
| `internal/marketplace/shein/publishing/payload_image_policy_test.go` | 原测试 | ImageDetail |
| `internal/marketplace/shein/publishing/site_policy.go` | 生产 | SiteInfo |
| `internal/marketplace/shein/publishing/submit_image_policy.go` | 生产 | ImageInfo |
| `internal/marketplace/shein/publishing/submit_image_policy_test.go` | 原测试 | ImageDetail, ImageInfo |
| `internal/marketplace/shein/publishing/submit_payload_normalize_policy.go` | 生产 | Product, ProductVideo, SiteDetailImageInfo, SizeAttribute |
| `internal/marketplace/shein/publishing/submit_payload_normalize_policy_test.go` | 原测试 | Product, SKC, SKU |
| `internal/marketplace/shein/publishing/submit_payload_validation.go` | 生产 | Product |
| `internal/marketplace/shein/publishing/submit_payload_validation_test.go` | 原测试 | ImageDetail, ImageInfo, Product, QuantityInfo, SKC, SKU, StockInfo |
| `internal/marketplace/shein/validator/validator_test.go` | 原测试 | Product, SKC, SKU |
| `internal/marketplace/shein/workspace/submit_payload_readiness_checks_test.go` | 原测试 | ImageDetail, ImageInfo, Product |

api/attribute 的三现役消费者：draft/local_preview.go、offlinepackage/model.go、publishing/submit_payload_normalize_policy.go；一并改接 previewdata.CustomAttributeRelation，不新增另一个关系模型。

</details>

## 4. 合同、清理与持久化边界

### 4.1 22真实值与当前 JSON

新 `offlinepackage/previewdata/model.go` 定义原21商品/嵌套值＋CustomAttributeRelation。字段名、类型、JSON tag/omitempty、声明顺序、nil/空集合及数值完全保持；原 Product.CustomAttributeRelation 只由旧包限定改为同叶子真实类型，不更换字段。没有方法或 codec 需要复制，不能新增 Marshal/Unmarshal 改写。新 Go 包/reflect/标准库类型元数据正常变化，不承诺旧全限定名，不用 alias/register映射维持旧身份。

<details><summary>22类型与字段数（全部完整保持）</summary>

| 类型 | 字段数 |
| --- | --- |
| `Extra` | 23 |
| `LanguageContent` | 2 |
| `Product` | 38 |
| `ProductAttribute` | 4 |
| `ProductVideo` | 3 |
| `SiteInfo` | 2 |
| `SizeAttribute` | 4 |
| `CostInfo` | 2 |
| `DetailImage` | 3 |
| `FieldDisabledInfo` | 3 |
| `ImageDetail` | 10 |
| `ImageInfo` | 3 |
| `PriceInfo` | 4 |
| `QuantityInfo` | 3 |
| `SKC` | 26 |
| `SKU` | 22 |
| `SaleAttribute` | 4 |
| `SiteDetailImageInfo` | 3 |
| `SkcExtra` | 3 |
| `SkuExtra` | 1 |
| `StockInfo` | 2 |
| `CustomAttributeRelation` | 2 |

</details>

`Package.PreviewPayload`、RequestDraft 的 SizeAttributeList、Package 的 CustomAttributeRelation 等引用只改 owner；offlinepackage model/clone/payload/defaults/images/readiness 与五规则文件仅 import/限定名改接。它们是现有平台草稿输出，不成为 canonical.Product 或新事实源。不能以提取器当前未填为由删 Product/Extra/SKC/SKU 的可选或空槽，包括 token 同名字符串槽；不读取任何实际凭据。

当前 `DecodePersistedPackageStrict` 的2MiB/64深度/UTF-8/重复/未知字段/数值/错误 sentinel 及 Clone/persistence 保持；validator RuleVersion `shein.offline_package.v1.2`、DiagnosticRuleVersion `shein.offline_package.v2.2`、BindingVersion `shein.persisted-input.go-json.v2`、hash算法/字节golden、Record digest/读取拒绝全部不变。不对已有Package/JSON/记录做读取、迁移、重写或自动接受。当前wire回读沿原严格codec，无旧包兼容桥。

### 4.2 清理抽取与明确退休的内部语义

采用现有 publishing owner 的最小纯值 API（未来实现，不在本阶段编码）：

```go
type DisplayTextCleanupInput struct {
    Words []string
    SourceBrand string
    BrandEnabled bool
    BrandName string
    BrandNameEn string
}
func DefaultDisplayTextWords() []string
func SanitizeDisplayText(text string, input DisplayTextCleanupInput) string
```

这些是调用值，无 JSON tag、存储、管理/授权语义或隐藏读取。DefaultDisplayTextWords 每次返回独立的原四语言词集；SanitizeDisplayText 只用显式 Words，nil/空 Words 表示没有通用词项，仍执行原品牌/空白规则。**生产 draft 始终显式传 DefaultDisplayTextWords()**，其余品牌字段为零，保持原空 Context 行为；不从 canonical.BrandName、runtime ctx 或 Metadata 自动导入品牌白名单/来源品牌，不恢复 Resolver、账号/API查询或新增授权。

| 原有效行为 | 新规则要求 |
| --- | --- |
| 四语言默认词集、静态/动态词的有效文本与去重 | 原字面值/大小写归一/TrimSpace/去重/每次独立结果保持；自定义词作为显式 Words 输入，支持原多语言值；无排序/扩词/locale新算法 |
| SanitizeDisplayTextWithContext | 处理顺序仍为通用敏感词→Amazon品牌→显式SourceBrand→空白整理；不影响普通文案、结构化属性选择和字段迭代 |
| 授权品牌白名单 | BrandEnabled false 忽略两个名称；true 时原Name/NameEn去重及 normalizeBrandToken/containsFoldedText规则保持。它仅控制清理，不签发品牌授权 |
| removeWordFromText | 原日文/CJK检测、大小写/词边界/连字符处理、regexp编译失败原分支、大小写和标点结果保持；不按注释推测“修复”实际实现 |
| getAmazonBrandWords | 原固定列表与顺序完整保持；SourceBrand trim/空值/允许保留规则保持 |
| 内存副本与并发只读 | 无全局可变词表，每调用独立默认切片；不修改 Words/Package 输入；复用有效自定义词/多语言去重/并发只读断言。调用方不得并发写其同一输入值；当前生产每次自有值，不引入共享写owner |

14块有效算法材料（默认词集、收集/去重和文本/品牌处理）选择性抽取/合并为无状态函数；旧 NewSensitiveWordServiceInMemory、initDefaultConfig、componentLogger 三个 Service/日志生命周期函数退休。新文件不定义 Service、mutex、配置对象、context.Context、logger、缓存、goroutine、生命周期或全局时钟；不把原 Service 换目录保活。Default词集可在函数内沿原 map/列表收集与去重，不借本轮固定其它顺序或优化regex。

**以下旧 KEEP 明确退休**：SensitiveWordConfig 的 static_words/dynamic_words 两JSON槽、可变 map/锁/内部 getter/初始化对象与JSON往返自测，仅为旧内存 Service 实现，没有当前HTTP/Package/持久化消费者；不迁移已有词表文件。真实默认/自定义词处理、独立值副本和并发只读规则保留为新纯值测试，不再断言旧 mutex/private config 存储。旧 `shein.sensitive_word` 初始化/词移除 debug 日志与专属logger helper 随外壳退出；现役 `shein/assembler` 日志、错误与安全/业务审计各原 owner不变，不新增仅为debug日志服务的trace/result框架。

`authorizedbrand.Resolved` 的Code/Context复制语义与WithResolved/FromContext存取测试退休；有效 Enabled/Name/NameEn/SourceBrand 规则转纯值输入。预览仍只沿原 `pkg.Metadata["authorized_brand_code"]` 填 BrandCode，不能因此自动启用清理白名单，不能把它变成认证事实。实际 Builder 保持原空品牌/无resolver上下文。

私有尾部：删除 ensureSanitizerContext、sheinSensitiveWordSanitizer interface/工厂及 adapter 文件；sanitizeSheinListingCopy/sanitizeLocalDraft/各sanitize子函数改为显式值并直接调用 publishing.SanitizeDisplayText。local_copy 两私有函数移除仅供旧清理的 runtimeCtx/logs 参数/无用imports；local_package删除自建Background及三处旧参数传递，保留其assembler logs参数和日志；local_copy_test六调用只改接线，其它断言/fixtures不变。Builder.Build本身、前后ctx.Err、ApprovedAsset、大小/严格解码与返回路径**零diff**。

### 4.3 事实、权限和外部副作用不变

当前 Product/Catalog/Asset/Record/Submission/SourceIdentity/lineage、Package持久化/绑定与官方提交各原owner不变，不新增表/状态/事务/补偿/retry/UNKNOWN/recovery。采集、图像、Price/AI/计量、规则/有效共享层不改。

独立 sheinlogin/session_identity.go 仍只用 client 的 SupplierOperateInfo/Warehouses/StoreAddressList 三个读取端点及 api/other/api/warehouse 转换。client/SessionTransport/Cookie/账号/Profile/Store身份和执行归属fence、tenant隔离、claim/取消/UNKNOWN/清理保持零diff。没有旧账号/新组织映射或新同步。真实实例/数据/配置凭据/日志/profile/轨迹/文件/进程不读取/操作；源码退出不授权物理退休。

## 5. 删除/抽取/保留矩阵

Legacy decision: **EXTRACT + RETIRE**。Reusable behavior：22当前wire值、默认/自定义词与文本/品牌纯规则。Current owner：域内previewdata数据叶子、现有publishing规则、draft显式组装；独立worker读取各原owner。Cutover/deletion condition：24值消费者＋5清理私有调用/测试完成切换，四旧包/adapter完整缺席、没有alias/Context/factory/Service回流，原合同/必要CI与最终源码复核满足；不含物理数据或运行资源操作。

### 5.1 完整处置

| 对象 | EXTRACT / RETIRE / KEEP |
| --- | --- |
| api/product两文件＋attribute/interface.go | 22真实值归 previewdata/model.go；旧两包/导入全退，不留空根或doc.go |
| content六生产＋两测试；authorizedbrand二生产＋一测试 | 原有效规则归新纯policy/测试；5个旧类型及Service/Config/Context、旧私有配置JSON/锁/存取/日志实现退出，不为旧测试保活 |
| draft/sensitive_word_adapter.go | 整退，无新adapter/interface/factory |
| 24值消费者 | 仅原import/限定名改接；13生产/11原测试控制流、fixture/assertion不变 |
| 5清理调用/测试文件 | 仅§4.2输入、私有签名/调用及旧Context专属组装改接；有效字段/文案/品牌/普通值/数字属性断言原样；三RuntimeContext单测可改为ExplicitValues对应名 |
| 两新数据文件＋两新纯规则文件 | 独立合成wire/类型/克隆与有效规则测试；不新增runner/平台/故障矩阵 |
| 三原检查文件与四当前/13历史文档 | 仅§6.1、§5.3准确范围；其它检查/永久禁令及历史保留 |

### 5.2 原测试处置与必要验证

11个值import测试只改owner；local_copy_test六调用只改私有参数。content/processor_authorized_brand_test 两有效品牌场景及cleanup_context_test的普通文案、默认/显式自定义词、多语言去重/副本/并发只读迁到新policy测试；旧Context存取/配置JSON/锁实现细节断言退休。draft/local_cleanup_test保留所有有效输出/changed/普通文案/结构化数字属性/品牌优先级断言，改为显式纯值输入；不能改期望求绿。

未来先用新测试路径对原值/规则做合成基线（不读取既有文件），再改接新owner验证等价；新完整退休guard真实RED→整批切换/删除→GREEN。原Builder三固定字节/hash、strictcodec/clone/validator/版本/Record/Asset授权/取消、原有效品牌/清理/并发、SHEIN登录/Store和实际当前命令依赖保持。仅必要原自检、原CI与适用编译，不扩环境验收；任何真实wire/hash/规则变化先停对应部分提案，不能更新golden/版本收口。

### 5.3 未来68唯一路径（本阶段尚未授权实施）

<details><summary>完整清单：同一主要PR整批覆盖</summary>

| 路径 | 处置 |
| --- | --- |
| `docs/architecture/README.md` | CURRENT_DOC |
| `docs/architecture/legacy-logger-facade-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY |
| `docs/architecture/legacy-numeric-content-management-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY |
| `docs/architecture/legacy-platform-api-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY |
| `docs/architecture/legacy-publishing-support-policy-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY |
| `docs/architecture/legacy-shein-brand-error-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY |
| `docs/architecture/legacy-shein-draft-assembly-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY |
| `docs/architecture/legacy-shein-draft-support-closure-retirement-boundary.md` | CURRENT_DOC |
| `docs/architecture/legacy-shein-listing-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY |
| `docs/architecture/legacy-shein-policy-contract-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY |
| `docs/architecture/legacy-shein-publishing-runtime-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY |
| `docs/architecture/legacy-shein-sensitiveword-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY |
| `docs/architecture/legacy-source-result-contract-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY |
| `docs/architecture/legacy-workerpool-sensitiveword-recovery-closure-retirement-boundary.md` | PREFIX_ONLY_HISTORY |
| `docs/architecture/project-boundaries.md` | CURRENT_DOC |
| `docs/refactoring/legacy-register.md` | CURRENT_DOC |
| `docs/refactoring/module-target-mapping.md` | PREFIX_ONLY_HISTORY |
| `internal/marketplace/shein/draft/local_cleanup.go` | REPLACE_CLEANUP_CALL |
| `internal/marketplace/shein/draft/local_cleanup_test.go` | REPLACE_CLEANUP_CALL |
| `internal/marketplace/shein/draft/local_copy.go` | REPLACE_CLEANUP_CALL |
| `internal/marketplace/shein/draft/local_copy_test.go` | REPLACE_CLEANUP_CALL |
| `internal/marketplace/shein/draft/local_package.go` | REPLACE_CLEANUP_CALL |
| `internal/marketplace/shein/draft/local_preview.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/draft/local_preview_test.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/draft/local_preview_values.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/draft/sensitive_word_adapter.go` | RETIRE |
| `internal/marketplace/shein/offlinepackage/clone.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/offlinepackage/clone_test.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/offlinepackage/model.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/offlinepackage/model_test.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/offlinepackage/payload.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/offlinepackage/payload_defaults.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/offlinepackage/payload_images.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/offlinepackage/payload_test.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/offlinepackage/previewdata/model.go` | NEW_PURE_OWNER_OR_TEST |
| `internal/marketplace/shein/offlinepackage/previewdata/model_test.go` | NEW_PURE_OWNER_OR_TEST |
| `internal/marketplace/shein/offlinepackage/readiness.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/offlinepackage/readiness_test.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/publishing/display_text_cleanup_policy.go` | NEW_PURE_OWNER_OR_TEST |
| `internal/marketplace/shein/publishing/display_text_cleanup_policy_test.go` | NEW_PURE_OWNER_OR_TEST |
| `internal/marketplace/shein/publishing/payload_image_policy.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/publishing/payload_image_policy_test.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/publishing/site_policy.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/publishing/submit_image_policy.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/publishing/submit_image_policy_test.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/publishing/submit_payload_normalize_policy.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/publishing/submit_payload_normalize_policy_test.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/publishing/submit_payload_validation.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/publishing/submit_payload_validation_test.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/validator/validator_test.go` | REWIRE_VALUES_ONLY |
| `internal/marketplace/shein/workspace/submit_payload_readiness_checks_test.go` | REWIRE_VALUES_ONLY |
| `internal/shein/api/attribute/interface.go` | RETIRE |
| `internal/shein/api/product/model.go` | RETIRE |
| `internal/shein/api/product/skc_sku.go` | RETIRE |
| `internal/shein/authorizedbrand/context.go` | RETIRE |
| `internal/shein/authorizedbrand/context_test.go` | RETIRE |
| `internal/shein/authorizedbrand/types.go` | RETIRE |
| `internal/shein/content/cleanup_context.go` | RETIRE |
| `internal/shein/content/cleanup_context_test.go` | RETIRE |
| `internal/shein/content/config.go` | RETIRE |
| `internal/shein/content/logger_port.go` | RETIRE |
| `internal/shein/content/processor.go` | RETIRE |
| `internal/shein/content/processor_authorized_brand_test.go` | RETIRE |
| `internal/shein/content/utils.go` | RETIRE |
| `internal/shein/content/word.go` | RETIRE |
| `tests/import_boundaries_test.go` | EXACT_CHECK_CONVERSION |
| `tests/listingkit_backend_retirement_test.go` | EXACT_CHECK_CONVERSION |
| `tests/validator_contract_boundary_test.go` | EXACT_CHECK_CONVERSION |

</details>

13历史文件仅prefix顶注，未来原完整base/final-main Git blob后缀保留：

<details><summary>旧KEEP依据历史文件</summary>

- `docs/architecture/legacy-logger-facade-closure-retirement-boundary.md`
- `docs/architecture/legacy-numeric-content-management-closure-retirement-boundary.md`
- `docs/architecture/legacy-platform-api-closure-retirement-boundary.md`
- `docs/architecture/legacy-shein-brand-error-closure-retirement-boundary.md`
- `docs/architecture/legacy-shein-draft-assembly-closure-retirement-boundary.md`
- `docs/architecture/legacy-shein-listing-closure-retirement-boundary.md`
- `docs/architecture/legacy-shein-publishing-runtime-closure-retirement-boundary.md`
- `docs/architecture/legacy-source-result-contract-closure-retirement-boundary.md`
- `docs/architecture/legacy-shein-sensitiveword-closure-retirement-boundary.md`
- `docs/architecture/legacy-shein-policy-contract-closure-retirement-boundary.md`
- `docs/architecture/legacy-workerpool-sensitiveword-recovery-closure-retirement-boundary.md`
- `docs/architecture/legacy-publishing-support-policy-closure-retirement-boundary.md`
- `docs/refactoring/module-target-mapping.md`

</details>

## 6. 验证与验收边界

设计阶段只读AST/3185 Go/字段/词表/调用/原检查/前批依据/路径/DAG核对，运行适用既有文档/inventory元数据检查与准确设计HEAD原CI；不执行业务codec/构造器/配置加载、词表清理、真实浏览器/provider或既有资源。设计CI/设计PASS不能冒充未来实施或运行验收。

用户确认Ready后，局部测试及compile-only、准确PRCI、实际merge-SHA push/main CI、真实运行/用户验收分别记录；未执行就保留NOT_RUN。设计/实施不增加内部试用前置；本机CRLF等真实FAIL分列，不改检查/golden掩盖。

### 6.1 原检查、文档与冻结范围的精确转换

未来只允许三既有检查文件中的以下 **11个原函数**转换，并在聚合文件新增一个完整guard；其余函数/helper算法/fixtures/阈值逐字保持（gofmt空白不算逻辑变化）：

| 文件/原函数 | 唯一允许的转换 | 必须保持 |
| --- | --- | --- |
| listingkit_backend_retirement_test.go / TestLegacySourceResultContractClosureIsRetired | attributeOwner及CustomAttributeRelation必需观察改为previewdata；增加旧api/attribute根缺席由新完整guard承担；三个旧DTO禁止仍保留 | Amazon capturedata 17值/5方法、36消费者、旧model根/import永久禁入及其它断言全不动 |
| 同文件 / TestLegacySheinDraftAssemblyClosureIsRetired | draftRoot必需表仅移除newSheinSensitiveWordSanitizer；pureImports以previewdata替换旧product/attribute两个import | 原29值/16规则、Builder、配置/无provider/时钟/old-private禁令、AST/解码/遍历/失败关闭保持 |
| 同文件 / TestLegacyWorkerpoolSensitiveWordRecoveryClosureIsRetired | 仅content的retained/Service工厂/Config字段JSON/锁/logger正向检查，改为当前纯policy函数和五字段输入必需；旧content根缺席/禁入与纯性由新完整guard严格保护 | workerpool/recovery全部原缺席/decoded-import禁令，旧content退休方法/字段黑名单及遍历/错误/无持久化/无goroutine底线保持；不加missing-skip |
| 同文件 / TestNumericAcquisitionAndPlatformTailClosureIsRetired | 仅从仍需读文件的map移除已退authorizedbrand/types.go一行；新guard要求旧根缺席 | 其它数字账号/映射/Task/平台禁令、原import解码与错误处理保持 |
| 同文件 / TestListingKitTaskBackendSourcesAreRetired | required表删除content/cleanup_context与旧adapter两行，local_cleanup行改为真实publishing纯函数调用；以当前policy/input实际必需项补齐 | 其它所有Task/CLI/workflow/temu/API/Store/ptr/日志/依赖/正向断言、全局禁入扫描与算法保持 |
| 同文件 / TestNumericContentConfigurationClosureIsRetired | 仅两条旧content/adapter必读行改为实际pure policy/input必需 | 原generation/topic/全局仓库/Config等退休全集/禁入与其它断言保持 |
| 同文件 / TestSheinListingMixedDeclarationsAreRetired | content加入现有category/aicache严格缺席分支；末尾CleanupContext三字段AST检查改为新DisplayTextCleanupInput五字段（Words/SourceBrand/BrandEnabled/BrandName/BrandNameEn）精确形状/无嵌入/无额外字段 | 原retired声明全集、root SHEIN Go缺席、其它owner/类型/算法保持；旧Context不重建 |
| 同文件 / assertLegacyPlatformAPICallClosureRetired | requiredByFile删除旧authorizedbrand两行、attribute一行、product两行；previewdata/model.go要求22真实值，policy要求真实纯入口 | 其它KEEP行、old API客户端/Resolver/retired根/字段全集、解码/遍历/错误与全部实际worker/client/other/warehouse必需保持 |
| import_boundaries_test.go / TestPublishingSheinNonAPISheinImportsStayAllowlisted | 仅前半content三文件allow表清空，旧content变全禁入 | 所有其它allow/deny、loadGoFileIndex/decoded-import/报错算法、后半managed整段保持 |
| 同文件 / TestPublishingSheinSubmitPrepUsesOnlySensitiveWordAdapter | 旧adapter必须缺席；转换为AST要求现役draft/local_cleanup真正直接调用publishing.SanitizeDisplayText，生产调用显式DefaultDisplayTextWords；禁止旧factory/interface/Context/Service、别名遮蔽或运行注入 | 同名原检查仍失败关闭，parser/decode不忽略错误；不为旧adapter保留stub，也不加新callback或logger依赖到pure policy |
| validator_contract_boundary_test.go / TestValidatorContractImportBoundary | 仅offlinepackage允许表以previewdata一项替换旧product/attribute两项 | 另外两root、扫描/解码/clock/provider/alias/错误逻辑保持；TestValidatorContractGuardRejectsRuntimeAndProviderDependencies含旧product负例及其它函数**零改** |

聚合文件新增 `TestLegacySheinDraftSupportClosureIsRetired`：四旧根/15文件及decoded旧import全缺席（所有tracked Go/测试/debug/构建目标）；22真实struct及五字段纯输入/纯入口、24值消费者实际改接必需；previewdata零import/方法/可变状态/alias，policy只strings/regexp、无Service/context/锁/日志/provider/App/DB/文件/时钟/隐式线程；Builder真实ctx/ApprovedAsset/codec边界仍有实际原检查保护；client/other/warehouse/SessionTransport/login/Store owner必需。复用原trackedFiles/AST/Unquote/helpers，不新增工具/runner，不全局ban同名Product/Resolved/AttributeValue。

当前文档未来仅本文、README、Legacy register、project-boundaries四项，13历史只加准确顶注；README/register保留最终main完整索引后缀。`assertLegacyLoggerFacadeRetired`虽保留旧content词，遍历稳定tracked根，不读缺席文件，**零改**；原Sourcing/旧model/Task/depguard/发布/预算/权限/安全禁令全部保持。原检查中的无消费者旧KEEP不是Authority，按上表准确转换而非加skip或放宽其它边界。

workflow、完整phase2/external-client等inventory（CURRENT/历史/计数/ceilings）、.golangci、root/debug依赖锁、部署及保护**零diff**；四旧包不对应CURRENT依赖行，无校准需要。所有现役语义/字节/hash/版本不变，不改binding或诊断golden。真实CI如证明需矩阵外改动，先查齐同根sibling、分类并给准确最小提案，用户决定例外后再改。

#301每次更新前新读正文，保持OPEN；只改当前段/准确证据，从“以下保留 #726 已合并批次与历史证据，不扩大本轮权限。”起的原完整后缀/勾选逐字保留，正文≤65536字符，旧当前段先归档。其它活跃PR原owner不接管；若只README相交，保留原完整索引与其有效条目。

## 7. Review、确认与停止点

- 一个Writer、复用原独立managed工作区/新codex分支、一个主要PR。准确设计HEAD原CI与适用独立Architecture Review完成后交用户确认§1–§7（明确含§4.2旧Service/配置JSON/日志/Context退休与有效规则抽取，§6.1）；IMPLEMENTATION_READY / Ready写入并读回后才改正式生产代码。当前SOURCE_NOT_STARTED/NOT_READY。
- 起始同步main，最终候选前核对实际main/活跃owner组合，只处理真实增量；未变证据可明确复用，不冒充新SHA直接执行，不机械重复全量。
- 架构最多两轮正常review，finding先按当前Must/具体后果/层级分类，只有新BLOCKER重开冻结设计；不追加全局品牌授权/词表管理、平台或试用前置。
- GitHub启动失败至多一次重试，Running不重复触发；**未经用户本轮授权不默认新增AI互验线程**。旧#964设计/源码复核授权不覆盖本批，不新增多Writer/常驻Reviewer/验收平台。
- 实施后一次集中源码最终交付复核，修复只复核对应增量；设计PASS/CI/另一AI都不是用户/生产验收。合并另交最终准确HEAD授权，正常expected-head，不管理员绕过/强推/改保护；合并后仅认实际merge-SHA push/main整run completed SUCCESS。
- #301 OPEN；独立SHEIN登录/Store同步、当前采集/有效规则和共享层保留。不操作既有实例/数据/凭据/文件/进程、不改workflow/inventory/依赖、不增加试用前置。当前无源码实施/合并/部署/关单/真实运行授权。
