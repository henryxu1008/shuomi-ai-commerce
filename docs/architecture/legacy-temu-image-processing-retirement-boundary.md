> #301 / #923 **SUPERSEDED，仅旧租户 Prompt 闭包**：用户已确认[完整退休设计 §1–§7（含 §4.2/§6.1）](legacy-tenant-prompt-closure-retirement-boundary.md)。本文涉及旧 `internal/prompt`、`internal/promptmgmt`、`shared/tenantctx`、五 YAML 及其现役 KEEP/复用建议仅在该准确矩阵内被退休决定替代；如涉及镜像打包，仅退出两条 `prompts` COPY。当前 AI/图像/采集、身份/有效规则、独立 SHEIN 登录与 Store 同步及其它 KEEP 不变。停止全新 schema 创建旧 Prompt 表，不改 Goose 登记或已有表/行，不新增替代 Store/迁移/兼容。以下原正文、历史数字/勾选/证据完整保留，不扩大实施、合并或真实资源权限。

> #301 / #915 **USER_CONFIRMED / SOURCE_IMPLEMENTED**：按[旧宿主、HTTP 与图像辅助完整退休闭包 §1–§7/§6.1](legacy-host-image-helper-closure-retirement-boundary.md)的用户确认矩阵，仅退休 imagex 无消费者的编码/格式 helper、专属 Format 类型/常量及原 Size 测试调用尾部；保留 ImageInfo、FromBytesWithFormat、Inspect 与 JPEG/PNG/WebP 生产注册，当前处理/填充/上传及平台合同不变。 仅该准确矩阵取代本文件对应 KEEP，其余边界及以下全部历史正文保留；workflow/inventory 与既有资源不动，独立 SHEIN 登录及 Store 同步保持，#301 OPEN。

> #301 / #891 narrowly supersedes this document's historical KEEP of `internal/pkg/downloader`: its complete unused listing download/anti-bot/random-fingerprint closure is source-retired under the [confirmed boundary](legacy-listing-preview-download-closure-retirement-boundary.md). Current image decoding/encoding, filling/upload, ImageAgent/SRC and SHEIN API downloads, shared httpclient/TLS, acquisition, Store and independent SHEIN login/Store sync remain with their existing owners. Only the obsolete package's KEEP is superseded; the original historical body and evidence below are preserved verbatim. No runtime/data/credential/process operation or new trial prerequisite.

# 剩余旧 TEMU 图片构建、标注与上传子链退休边界设计

> 后续用户已确认 [#819 TEMU剩余整链边界](legacy-temu-listing-closure-retirement-boundary.md) §5/§6/§6.1/§7。本文件对该闭包内旧 TEMU AI/SKU/spec、PriceHandler/PriceCalculator、property/context/TemplateState、pipeline/skugen及专属纯规则/管理投影/prompt的保留描述仅为历史边界，现由#819精确替代；当前平台API/规则、共享合同、SHEIN价格/两属性、采集/图像/Store观察与独立SHEIN登录/Store同步继续保留。原证据不改，不授权实例或数据操作。

> 后续窄替代：用户确认 #807 [图片 API / vision prompt 退休矩阵](legacy-temu-image-api-prompt-retirement-boundary.md)后，本文当批保留的旧 image API/re-export、vision key/default/catalog/专属变量元数据也已 source-retired。§3.1/§5 对这些对象的保留仅是 #801 历史边界；其它合同、通用 Prompt 管理/存储和全部保留能力不变，不操作已有覆盖记录、实例或数据。

## 1. 状态、用户结果与范围

- 日期：2026-10-18；执行依据：#29 / #301 与用户本轮明确授权“先设计、确认后实施”。
- 状态：用户已确认冻结设计 `348ec6b6e` 的 §5/§6/§6.1/§7，并授权在 #801 继续实施。旧执行包、孤立类型与专属 context 状态已 source-retired；原检查和必要清单按矩阵校准，最终 HEAD/CI/独立检查写 PR。
- 当前 **Design Basis: Reuse Existing Architecture / Ready**：生产修改前已在 #29/#301 登记并读回，按 §2 复用现有 owner/合同。设计阶段 N/A、CI/review 保留为历史，不替代实施证据；workflow、旧 image API、vision prompt 和运行数据均不改。
- 取证基线：actual main `4faf6f471699da44f763872cc9e98c16f105fe40`，含 #797 已合并的七文件退休。唯一 Writer 复用独占附属 worktree `issue29-temu-image-retirement/task-processor`，新分支 `codex/issue29-temu-image-closure-retirement-design`；已核对父 Issue 接单/关联 PR，无重复 owner，不接管 #799/#800/#692。

使用者是当前内部试用、采集、Store 与图像能力维护者。交付为剩余旧图片构建、并发处理、标注/填充/vision/上传闭包的实际消费者、有效行为处置、持久化及平台副作用 owner 和精确删除/保留矩阵。当前没有用户操作被本维护阻塞，**不增加内部试用前置，也不作为 #799 当前图片接线的前置**。没有新页面/交互，Figma N/A。

已退休 `internal/temu/image` 剩余 15 个完整 Go 文件（基线 1,952 行）及 `handlerbase/types.go` 的孤立 ImageRequirement；mixed context 只移除两项 padded map 和因此完全无成员/消费者的 AssetState 定义、嵌入与初始化。有效规则复用现有 owner，**无新增 EXTRACT 代码**；不保留空壳或复制无当前需求的图像算法。

非目标：新图像/标注产品、provider 接线、重接旧 Task/上传/vision、额外 IAM/恢复/验收工具；修改保留规则、API/DTO/Prompt Management；运行 constructor/main、真实下载、模型/API/平台上传、Preview/DryRun；读取运行配置、凭据、DB/Redis/Profile/历史日志；部署、重放、迁移、远端/本地业务资产清理与 Issue 关闭。只读取跟踪中的 Go/文档/prompt 源码，不读取私密运行配置。

## 2. Design Basis、authority 与不变量

Architecture / Contract：

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Project Boundaries](project-boundaries.md)：只允许 EXTRACT / RETIRE，不新增 wrapper、fallback、双读双写、旧 ID 映射或第二事实源。
- [当前 TEMU 平台设计](temu-platform-v1.md) §8/§8.1、现有 marketplace TEMU goods/rules/model：资产批准、内容 hash、当次店铺/模式/类目/usage、几何/位置、官方上传效果及 receipt/UNKNOWN 各守原 owner；旧网页默认值不能成为当前规格权威。
- [Product ImageAgent](product-image-agent-v1.md)、[官方标题与图片方法](official-skills-and-image-method-v1.md)：当前资产/图片方法选择、生成/审查/采用、预算与产物合同保留。#799 的 Bailian 接线由原 owner 实施；其未合并实现不是本设计的批准依据或验收证据。
- [前批图片编排](legacy-temu-image-orchestration-retirement-boundary.md)、[SKU/SKC](legacy-temu-sku-skc-retirement-boundary.md)、[类目/模板](legacy-temu-category-template-retirement-boundary.md) 退休：历史取证和批准保持；用户已确认本文，只用 §5 替代本批 16 文件及专属 context 闭包的旧保留陈述，不覆盖其它保留能力。
- [Prompt Management](zitadel-tenant-prompt-management.md)：catalog metadata、tenant override、管理 API 与 runtime rendering 分别有原 owner。vision prompt 有目录引用但没有独立运行消费者，见 §3/§3.1；目录引用不证明该 prompt 是当前有效产品能力。
- [Store current application](store-center-current-application-v1.md)、[Store observations](store-center-platform-observations-v1.md)、[独立 SHEIN 身份持久化](shein-worker-identity-persistence-boundary.md)：当前及保留运行链不借旧 Cookie/Profile 补当前身份，不操作运行资产。

本次有界实施采用 **Reuse Existing Architecture**：只退出无装配的旧单域图片编排及专属内存值，当前正确能力/合同沿原 owner 保持，没有新产品流程、领域合同、事实/schema、身份/权限/tenant、状态机/事务、幂等/retry/UNKNOWN/recovery、AI 准入/预算/计量或平台副作用 owner。16 个完整生产文件共 1,964 行，加一个 mixed context 的专属删除；原两 guard 与必要清单/直接冲突文档预计不超过 30 个范围相关文件，生产删除量低于 2,500 行。不通过机械拆 PR 规避阈值；若实际超线且既有批准不足，或必须改变上述边界，停止受影响部分补适用 Independent Architecture 准入。

Must：保留当前 Product/ImageAgent/ApprovedAsset、有效图像规则与原官方上传/receipt/UNKNOWN，平台 API/DTO/re-export、共享 imagex/downloader/goroutine/perf/timeout/prompt/logging/ptr、原 DimensionInfo/TextRenderer、AI/SKU/AIBatchInput、PriceHandler/ProfitRuleAPI、PriceManager/PriceCalculator、两种 SHEIN 属性、Reader/Cache/Mapping、四 TemplateState 字段、当前采集/Store 观察与独立 SHEIN 登录 worker/Store sync。两套及其它 workflows、必需/安全检查、保护/ceiling、依赖不改。

Should：原 guard、当前清单及直接冲突说明与源码一致。Out of Scope 如 §1；无新增 Accepted Risk。Threat Model 限于误删真实共享消费者、将旧 URL/尺寸/重试策略提升为当前合同、误报运行资产清零；不扩成新的图像、IAM、恢复或验收平台。

## 3. 实际消费者与调用闭包

基线全仓精确 Go 导入搜索未发现 `task-processor/internal/temu/image` 的外部导入；构造器、receiver、15 文件中的私有 DTO/helper 仅在该包内部定义/互用，未发现 cmd/scripts/config/workflow 装配。静态 `go list ./cmd/...` 得到 **31 个当前命令**，其依赖并集包含该包 **0 次**；未运行 main。没有包内专属测试文件。这不是历史实例、覆盖模板、字节缓存或远端资产存在性审计。

| contract → implementation → injection → consumer | 基线实际行为 | 决定 |
| --- | --- | --- |
| ImageProcessor → carousel/dimension builders、upload utils | Task context/旧 Amazon Product/AI SKU mapping → raw main/SKU galleries；没有外部构造/消费 | RETIRE 整个 root 与下游专属闭包，不新增 replacement processor |
| ParallelImageProcessor → goroutine.NewProcessor/perf.NewTracker → ImageProcessor | 共用同一 TemuTaskContext、后台 context、每任务超时参数；按变体索引收集并写 SKU galleries，尺寸图优先裁至十图 | RETIRE 此编排/Task/Result；shared goroutine/perf 的其它 SHEIN 消费者保留 |
| Carousel/Dimension builders → upload utils/padding/annotator | 依第1/3/4张位置挑图，旧9/10上限，按ASIN从AISkuMapping找尺寸，进程map保存标注/填充字节；上传后写raw metadata | RETIRE Task 选择策略/搬运；不把旧索引/尺寸猜测作为当前已批准资产/slot规则 |
| Annotator → DimensionDrawer/DrawingUtils、rules.TextRenderer、imagex/downloader | 解码、白边、箭头/文本和编码；默认构造 NewVisionDetector(nil)，内部标注直接跳过检测 | RETIRE 包内实现及 DimensionInfo 本地 alias；保留原 TextRenderer/handlerbase.DimensionInfo、共享解码/编码/下载 |
| VisionDetector → prompt.GlobalRegistry → 借用 ChatCompleter | 可显式传入客户端时构造 base64 文本请求，按 YES/NO 解析；基线没有包外或当前构造器注入 | RETIRE detector 及其包内 fallback 文本；不把此处提示词/YES分支复制成当前 vision 准入 |
| UploadProcessor/Utils/Helpers → download/signature/upload service → 借用 APIClient | 旧 CDN 子串跳过、部分失败继续、批量不全时逐项再上传；填充字节/尺寸map、原URL、响应和默认尺寸共同组raw ImageInfo | RETIRE 编排/局部 TemuAPIClient 接口/helper；平台 api/image、APIClient/DTO/鉴权 owner 保留 |
| prompt.KTemuVisionDetectorDetect → promptmgmt catalog/manifest + prompts/temu/vision_detector.yaml | 有目录/schema/Upsert known-key 源码引用，但 **没有独立运行消费者**；promptmgmt 本身在31命令依赖中也为0，不能宣称管理入口已开放或该 key 有执行效果 | 本批范围外保持源码，列入 §3.1 后续 RETIRE 候选；目录引用不构成有效能力/兼容性依据，不新增消费者 |

架构问题：旧上传按“压缩后的成功切片索引”写回原 galleries，前项失败时后项可能覆盖前项 metadata；批量部分成功后再逐项尝试可能重复上传已成功项；adjustImageDimensions 只改 metadata 而不对应改远端像素；共享 padded map 在并发处理下没有明确隔离，子任务未消费传入取消 context；CDN strings.Contains 不证明 URL 授权、资产归属或上传成功。以上是可见源码风险，不是已发生的运行事故。当前没有装配用户路径，分类 **NOT_APPLICABLE（当前内部试用阻塞）**；通过退休旧编排收敛，不修旧链、不新增锁/上传账本/恢复平台或试用前置。

### 3.1 独立检查指出的范围外残留

准确范围是完整 `internal/temu/image` **执行包**、孤立 ImageRequirement 与专属 context 闭包；不是所有旧 image API/Prompt 残留已经清零。两项事实成立，均分类 **BACKLOG**：

- `internal/temu/api/image` 在执行包退出后只剩 `api/api.go` re-export 与 guard 引用，没有当前生产行为消费者。旧 Cookie/MALL_ID API 不属于当前官方 `internal/integration/temu` 接入依据；本批按既定平台 API/DTO 范围边界不改它，不将 re-export 称为有效需求。
- vision prompt 删除 detector 后只剩 key/默认模板/catalog metadata，目录管理也不能带来执行效果。其 key/default/catalog/known-key 是后续精确退休对象；不将目录引用称为独立语义消费者或保留能力证明。

两项登记为后续 **RETIRE 候选**，并非新增 compatibility 类别或永久保留权利；不新增入口、消费者、包装或运行效果承诺。没有当前命令用户路径受阻或项目列明的 BLOCKER 后果，不扩大本轮16文件设计，也不增加内部试用前置。后续需确认各自精确 API/re-export/Prompt 删除边界；不因此访问、迁移或删除任何运行覆盖记录/资产。

## 4. 持久化、身份与平台副作用 owner

| 边界 | owner / 效果 | 本批处理 |
| --- | --- | --- |
| 当前 Product/ImageAgent/ApprovedAsset 与 generation/asset publication | 当前版本、源选择、授权、预算/计量、产物字节和采用均由原 owner 负责 | 原源码/合同/测试保持；不将 raw Product URL、旧 map 或自报尺寸作为第二事实源 |
| 当前 marketplace TEMU goods/rules/model、storecenter 官方 mutation 与 supplychain/Submission 原执行合同 | TEMU 图片观察/批准内容/用途/几何/slot、原 merchant access；当前跨平台官方效果/receipt/UNKNOWN 各走原平台 owner | 不改平台路径或将 SHEIN ImageSet requirements 冒称 TEMU 实现；当前 ImageSet source/target 和 #799 接线整体由原 owner 保留 |
| AssetState 两项 map | 只有旧 image 包读写 PaddedImages/PaddedImageSizes；context 的定义/嵌入/构造分配为其专属进程载体 | 已一起 RETIRE 两字段、初始化、空载体及嵌入；不迁移/读取/删除任何运行字节或历史payload，其余 context 保持 |
| handlerbase.ImageRequirement | #797 后仅余定义，无生产消费者；不等于当前 official image requirements | RETIRE 此唯一类型文件；当前各平台 OfficialImageRequirements、handlerbase.DimensionInfo 等不同合同保持 |
| download/signature/upload | 旧 service 经借用 SendTEMURequest；POST /mms/marigold/edit/commit/get_signature、/api/galerie/v3/store_image?sdk_version=js-1.0.6&tag_name=local-goods-image；GET 由共享 downloader 负责 | 只退出 caller/局部服务，不调用端点，不转移 signature/网络/tenant/Cookie/结果不确定性 owner。端点只是仓库源码映射，不是外部平台支持性确认 |
| vision/provider | 旧 detector 可借用 OpenAI ChatCompleter；prompt key/template 管理另有 consumer/owner，可能持有 tenant override | 只退 detector。provider客户端/当前图像方法/Prompt Registry/目录/override仓库全部保持；不读写配置、覆盖记录或发请求 |
| 旧图像远端资产、数据库/文件/历史日志 | 各原平台/资产/数据 owner；静态无当前 caller 不证明资产不存在 | 不扫描、重放、采纳、清理或删除；source retirement 不授权运行态操作 |
| 当前采集、Store 观察、独立 SHEIN 登录/Store sync | 原 acquisition/Store/observations/worker owner | 身份/tenant/持久化/恢复不动，功能和运行边界保持 |

### 4.1 有效行为 EXTRACT / RETIRE

本批**不新增 EXTRACT 代码**：当前有要求的行为已有正确 owner 或成熟共享实现；没有当前要求的旧编排/算法不因“纯值”而复制为闲置工具。

| 行为 | 正确 owner / 现状 | 处置 |
| --- | --- | --- |
| 批准资产/内容hash、店铺/类目/usage、格式/体积/实际尺寸与精确槽位 | Product/ApprovedAsset、marketplace TEMU goods 的 ValidImageObservation/resolveImages、supplier_images/image_artifact、原官方平台模型/规则 | 原位保留，不用 URL 子串、默认宽高、ASIN/切片索引或 metadata 修正替代当前事实 |
| 解码/编码/图像检查、下载/请求边界 | internal/pkg/imagex、internal/pkg/downloader；当前 ImageAgent generation/objectstore、SHEIN API 等仍有生产消费者 | 原位保留标准库与已有组件；不抽新下载器/格式库，不移植旧网络入口 |
| 尺寸值、文本渲染 | handlerbase/property_types.go 的 DimensionInfo 仍被 rules.TextRenderer.DrawSummaryInfo 消费 | 原类型/单位语义、TextRenderer/字体实现与其它规则保持；只退 image/dimension_models.go alias，不能删共享 owner |
| 白边、绘线、旧位置选图/最大9或10图/尺寸优先、AI尺寸转标注 | 候选包内专属设计；当前图像生成/已批准资产选择/原平台规则不依赖它；当前 scope 没有新增“自动尺寸标注/自动白边”产品要求 | RETIRE 包内算法/默认值/Task搬运，不新增同名 helper 或恢复旧刊登能力；保留其它包已有实际图像能力 |
| 批量失败后逐项重试、CDN跳过、上传metadata与进度统计、context缓存 | 旧 Task/网页端编排，无当前持久化receipt/UNKNOWN依据 | RETIRE，不复制到当前效果/恢复协议；不增加“先完善通用上传系统”前置 |
| vision模板/管理schema及旧 image API/DTO | 仅余目录或 re-export 静态引用，无独立运行消费者；不是当前有效能力依据 | 本批范围外不改，按 §3.1 登记后续 RETIRE 候选；不新增 consumer、不承诺效果、不将暂留源码解释为兼容或永久权利 |

## 5. 精确删除与保留矩阵（用户已确认）

短文件名相对 `internal/temu/image/`。完整包退休覆盖其现存15文件及包内全部 receiver/constructor/私有值，不通过改名或搬目录继续保留旧实现。

| 路径 / 对象 | 决定 | 边界 |
| --- | --- | --- |
| processor.go（77）、parallel_processor.go（215） | RETIRE 两整文件 | ImageProcessor/ParallelImageProcessor、ImageProcessingTask/Result、全部转发/并发/回写与数量策略 |
| carousel_builder.go（124）、dimension_builder.go（173） | RETIRE 两整文件 | Carousel/Dimension builder、Task选图/ASIN尺寸匹配/标注字节/上传编排及全部私有方法 |
| dimension_annotator.go（125）、dimension_drawer.go（124）、drawing_utils.go（63）、dimension_models.go（7） | RETIRE 四整文件 | annotator/drawer/drawing、DimensionInfo本地alias、全部字节/绘线/文本转发/构造器；原 DimensionInfo/TextRenderer 保留 |
| padding_processor.go（168）、vision_detector.go（140） | RETIRE 两整文件 | 包内 ImagePaddingProcessor/PaddingResult、VisionDetector、下载/白边/检测/fallback文本/解析；共享与目录合同保留 |
| upload_processor.go（321）、upload_utils.go（117）、upload_helpers.go（148）、upload_service.go（99）、config.go（51） | RETIRE 五整文件 | 原上传编排及 TemuAPIClient局部接口、signature/upload/download helper、config/CDN/默认值、全部DTO/方法，不删平台同名类型 |
| internal/temu/handlerbase/types.go（12） | RETIRE 整文件 | 只有孤立 ImageRequirement，无保留consumer，不保留空文件/type alias |
| internal/temu/context/temu_context.go | RETIRE 专属两字段及载体 | 只删 PaddedImages/PaddedImageSizes、其map初始化、AssetState struct/匿名嵌入/构造literal；不留下空 AssetState 或新缓存/bytes owner |
| handlerbase/property_types.go、rules/text_renderer.go、其它 handlerbase/rules/context | 原样保留 | DimensionInfo、TextRenderer及字体/单位、Runtime/Product/Template/Publish/Variant/RuleState、四TemplateState字段、API/Store/AI SKU合同全部保持 |
| internal/temu/api/image、api/api.go、平台 product/models | 本批原样保留 | image API/Upload/Signature/ValidationResult/ProcessingOptions/PaddingResult/re-export 不因同名跟删；其中旧 image API/re-export 无当前行为 consumer，后续 RETIRE 候选见 §3.1，不冒称当前官方 API |
| internal/prompt/keys.go、prompts/temu/vision_detector.yaml、promptmgmt catalog/manifest/service/store | 本批原样保留 | vision key/default/catalog 无独立执行 consumer，后续 RETIRE 候选见 §3.1；本批不改变 known-key 或管理行为，不访问/清理可能的持久化覆盖 |
| shared imagex/downloader/goroutine/perf/timeout/logging/ptr、integration/openai/provider | 原样保留 | 当前图像/其它SHEIN消费者仍使用这些原能力；只退出两个旧OpenAI caller，不删共享包或更换客户端 |
| 当前 Product/ImageAgent/ApprovedAsset、TEMU goods/rules/model、app supplychain/storecenter、#799候选 | 原样保留/不接管 | 当前图片/官方效果/权限/资产/预算/UNKNOWN/恢复保持；#799未合并实现不作为本批前置 |
| AI/SKU、定价、两SHEIN属性、Reader/Cache/Mapping、采集/Store观察、独立SHEIN登录/Store sync | 原样保留 | 此前全部保留能力按原owner保持，不扩大到其它平台同名目录或真实资产 |

Legacy decision: **RETIRE** 旧完整 image 包、孤立 ImageRequirement 与专属 context 状态闭包；无新增抽取。
Reusable behavior: 当前有效图像/资产/平台规则、共享图像/下载/文本渲染和原API/Prompt合同原位保留。
Current owner: 当前 Product/ImageAgent/ApprovedAsset、marketplace TEMU goods/rules/model、原官方效果/Store/Submission、shared imagex/downloader、handlerbase.DimensionInfo/TextRenderer、平台API与Prompt Management。
Cutover/deletion condition: 用户确认 §5/§6/§6.1/§7，登记/读回 Reuse Existing Architecture / Ready、刷新消费者、原guard RED→GREEN并保留有效检查；准确实施HEAD原CI/最终检查完成。不新增compatibility、fallback或第二事实源。

## 6. 验证与验收边界

设计阶段已只读核对16完整文件、mixed状态与真实共享消费者、31命令静态依赖、Prompt/平台/当前owner及 #799 边界；历史设计只交本文与索引。当前实施按下列边界验证，原guard已在旧源码尚在时准确RED，精确退休后GREEN；最终CI/独立检查与准确HEAD见PR。真实下载、生成/provider、平台上传/运行/用户验收均 **NOT_RUN**。设计CI/review不替代实施验证。

实施的必要验证与验收约束（结果与准确 HEAD 写 #801）：

1. 开工同步一次 actual main，复核整个 image 包外部导入/构造器/注册、ImageRequirement、两map/AssetState与16文件闭包，确认未与#799当前owner重叠。发现真实新consumer、具体兼容义务或有效行为缺owner，停止对应项报告。
2. 在原退休测试先改准确guard并确认源码尚在时RED，再删除/GREEN；检查原当前规则、平台/Prompt/DimensionInfo及其它保留能力。无新业务规则，不为删除算法制造新测试平台。
3. 复用当前 TEMU goods/rules、SKU/定价、handlerbase/context/API、shared imagex/downloader、Prompt Management原回归和当前命令依赖/编译；只验证改动的组合风险。current-application/shein-login-worker只build，不运行main。
4. 完整必需检查留给稳定实施候选原CI，不人工重复全量或新增故障矩阵/runner。最终以准确实现HEAD证据进入In Review；finding先分类，修复只复核增量。
5. PR CI/源码无caller不等于main CI、真实效果、用户验收或运行数据已清零；#29/#301仍有其它债务，保持OPEN，不增加内部试用前置。

### 6.1 原检查、清单与文档的精确处置

| 现有位置 | 按确认设计实施的最小变更 | 不得扩大 / 削弱 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go 缺席/保留map | 用整个 internal/temu/image 目录缺席覆盖包括#797七路径在内的全部Go文件，增加 handlerbase/types.go 缺席；移除已退休15文件、ImageRequirement、AssetState/两map/其初始化的旧保留token；新增原 DimensionInfo/TextRenderer及当前有效规则/平台保留断言，共享Prompt合同复用原检查 | 原root/product/CLI/运行/身份/安全断言与有效保留map保持；整目录缺席比旧准确文件/包级名字检查更强，不为同名平台/其它包做全局禁令，不为§3.1残留新增保留断言 |
| 同一测试原mixed AST与image专属fixture | 仅从必存遍历移除将消失的image项，保留原sku/filter/product/rules/handlerbase五包；image专属helper/十fixture随整目录禁令替代而退出，不留空调用；复用原AST局部检查防handlerbase.ImageRequirement alias/type及context.AssetState/两字段/初始化重建，保持其它领域同名值 | 原SKU九fixture和五mixed扫描不得删；旧三Asset字段/Publish禁令保持。不得使用遇缺就广泛skip、改root/ignore或新扫描框架。context仍有其余合同，不能把其检查整体删除 |
| tests/import_boundaries_test.go TestTEMUOpenAIImportsStayAllowlisted | 仅删 image/dimension_annotator.go、image/vision_detector.go 两条失去的豁免 | 其它AI/SKU等条目、root/loader/deny/alias检查及scope原样；不顺带清理无关历史条目或修改provider准入 |
| docs/refactoring/phase2-runtime-inventory.md 当前consumer register | 仅移除 core/logger → temu/image 与 integration/openai → temu/image 两行，因为完整包无源码consumer | 日期化inventory/历史debt list、其它consumer和全部closure ceiling保持，不误报logger/OpenAI或其它平台全部退休 |
| docs/refactoring/code-health-decisions.md image行、Legacy Register、本文/索引与直接冲突说明 | 标明image包/孤立类型/专属状态source-retired，保留日期化15 declarations计数；当前验证目标换为retained context/API/rules/当前goods与原guard。窄替代#797等当批保留陈述，Prompt/API/共享保留明确 | 历史审批/取证保留，不恢复旧命令、不重写历史统计，不将全部TEMU/Prompt/平台合同记成退休；设计阶段只交本文与索引 |
| .github/workflows/ci.yml、commercial-readiness.yml及其它workflows、.golangci.yml/依赖/保护 | **零变更** | 必需/安全检查、编译目标、触发条件、保护/ceiling保持，不增加试用/发布前置 |

## 7. 确认、实施与停止条件

1. 用户已确认冻结设计 `348ec6b6e` 的 §5/§6/§6.1/§7，当前在同一主要 PR 实施；原 DESIGN_ONLY/N/A 只是历史设计阶段，不再限制已获授权的矩阵内修改。
2. 已在生产修改前于执行 Issue 登记并读回 Reuse Existing Architecture / Ready、冻结设计及用户结果/owner/路径/不变量/验证；随后退出完整旧包/孤立类型/专属状态、校准原检查与清单，两套 workflow 保持不变。
3. 新真实消费者、当前外部可观察兼容义务、有效行为缺owner或必须改变身份/tenant/事实/状态/事务/恢复/平台效果时，停止受影响部分报告；不临时造adapter/fallback或扩成图像/IAM/恢复平台，不接管#799。
4. 实施完成交准确HEAD原CI/最终检查进入In Review；未变有效证据可明确复用，非Blocker不自动扩范围或重开全局设计。设计/源码/AI检查不能签发用户验收。
5. 合并需准确最终HEAD单独授权并正常expected-head；不授权Issue关闭、部署、真实实例/API/provider/凭据/业务数据操作。保留当前采集/Store观察、图像/平台/共享合同及独立SHEIN登录/Store sync，不增加内部试用前置。
