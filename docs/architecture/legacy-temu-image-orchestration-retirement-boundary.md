# 旧 TEMU 图片初始化、校验与并发上传编排退休边界设计

> #807 [图片 API / vision prompt 退休矩阵](legacy-temu-image-api-prompt-retirement-boundary.md)进一步窄替代本文旧 image API/re-export、vision key/default/catalog/专属变量元数据的当批保留；这些对象已 source-retired。其它保留能力、通用 Prompt 语义、已有记录与历史证据不变。

> 后续窄替代：用户确认 #801 [剩余图片执行包退休边界](legacy-temu-image-processing-retirement-boundary.md)后，剩余15文件、孤立 handlerbase.ImageRequirement 与专属 AssetState/两 padded map 闭包也已 source-retired。本文当批保留陈述与检查处置仅对上述对象被新矩阵替代；原取证/批准保留为历史。旧 image API/re-export、vision prompt、DimensionInfo/TextRenderer、共享基础层与此前其它保留能力不变；不授权运行或数据操作。

## 1. 状态、用户结果与范围

- 日期：2026-10-18；执行依据：#29 / #301 与用户“先做设计、确认后实施”的明确授权。
- 状态：**USER_CONFIRMED / SOURCE_RETIRED**。用户已确认 #797 冻结设计 `1fb4702698c34b2dfeaf07ef48e063940a442162` 的 §5/§6/§6.1/§7；七文件及专属 helper/alias/三字段已按矩阵退出。合并与真实操作仍需单独授权。
- 当前 **Design Basis: Reuse Existing Architecture；Admission: Ready**。生产修改前已在 #29/#301 登记用户结果、冻结依据、owner/路径/不变量/验证并读回 Ready。原设计 N/A 仅覆盖当时文档/只读映射；准确实施 HEAD、CI 和最终独立检查维护于 PR，不自行签发用户验收。
- 取证基线：actual main `70c0622059966cb1cadc9613dce065f24323fbe4`，含 #773 的类目/模板退休，以及 #795 的当前官方标题/图片方法。唯一 Writer 使用独占附属 worktree `issue29-temu-image-retirement/task-processor`、分支 `codex/issue29-temu-image-task-retirement-design`；已核对接单与关联 PR，无重复 owner，不接管无关 #692。

使用者是维护当前内部试用、采集及 Store 能力的维护者。交付为三个旧图片编排入口及专属闭包的消费者、有效规则、下载/视觉检测/上传/回写的持久化与平台副作用 owner、精确删除和保留矩阵。**当前没有用户操作被本维护阻塞；本项不增加内部试用前置。** 不新增用户页面或交互，Figma N/A。

已退休 `internal/temu/image` 下三个入口及四个专属校验器，共七个整文件（基线 1,005 行），mixed config 的专属 helper/本地 alias/import，以及 AssetState 的三项专属字段。**目录其余图像处理/填充/上传/vision 能力、两项 padded map 与平台合同保留；无新增 EXTRACT 代码。**

非目标：新图片产品/入口、重接旧 Task、改造保留实现、额外 IAM/规则/上传/恢复平台或验收工具；运行 constructor/main、真实下载/视觉/provider/平台上传、Query/Preview/DryRun；访问配置/凭据/DB/Redis/Profile/历史运行日志；部署、重放、迁移、远端或本地资产清理、Issue 关闭。源码取证不证明运行资产清零。

## 2. Design Basis、authority 与不变量

Architecture / Contract：

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Project Boundaries](project-boundaries.md)：只允许 EXTRACT / RETIRE；不新增 wrapper、fallback、旧 ID 映射、双读双写或第二事实源。
- [当前 TEMU 平台设计](temu-platform-v1.md) §8/§8.1：当前 approved asset、内容 hash、店铺/模式/类目/usage、官方执行 receipt、UNKNOWN 与刷新/恢复各守原 owner。源代码中的旧网页端点及默认值不是当次官方平台规格权威。
- [Product ImageAgent](product-image-agent-v1.md)、[当前官方标题与图片方法](official-skills-and-image-method-v1.md)：图片生成、审查、批准、方法绑定与精确资产选择沿当前 Product/ImageAgent/ApprovedAsset 及既有执行合同；不以旧 Amazon URL 或 raw gallery 代替已批准资产。
- 此前用户确认的 [保存/提交](legacy-temu-listing-submit-retirement-boundary.md)、[前置](legacy-temu-preflight-retirement-boundary.md)、[SKU/SKC](legacy-temu-sku-skc-retirement-boundary.md)、[类目/模板](legacy-temu-category-template-retirement-boundary.md) 退休：旧 Task 入口退出，图像/平台/共享能力保留。用户已确认本文，仅 §5 七文件及专属 helper/alias/三字段的旧保留陈述由本矩阵窄替代，其余保留项及历史证据不变。
- [Store current application](store-center-current-application-v1.md)、[Store observations](store-center-platform-observations-v1.md)、[独立 SHEIN 身份持久化](shein-worker-identity-persistence-boundary.md)：当前与保留数字运行链各守原 owner，不从旧 Cookie/Profile 补当前身份。

本次有界实施采用 **Reuse Existing Architecture**：只退出无当前装配的旧编排及其私有进程值；有效规则已由当前 owner 持有，下载/填充/处理/上传等真实共享能力继续由原 owner 持有。不新增或改变公共领域合同、事实/schema、身份/权限/tenant、状态机/事务、幂等/retry/UNKNOWN/recovery、AI 准入/预算/计量或平台副作用 owner。七生产文件 1,005 行删除加专属 helper/alias/三字段、原 guard/清单和直接冲突说明，未命中 30 文件或 2,500 生产行提示线。若实际需要改变这些边界，停止受影响部分，补适用 Independent Architecture 准入。

Must：保留当前有效图像规则、七文件以外的图像处理/填充/上传/vision 能力、平台 API/DTO、有效规则及共享合同；保留 AI/SKU 映射与 AIBatchInput、PriceHandler/ProfitRuleAPI、PriceManager/PriceCalculator、两种 SHEIN 属性、Reader/Cache/Mapping、四个 TemplateState 字段、当前采集/Store 观察及独立 SHEIN 登录 worker/Store sync。两套及其它 workflows、必需/安全检查、保护/ceiling、依赖均不改。

Should：原 guard、当前清单与源码事实一致。Out of Scope 如 §1；无新增 Accepted Risk。Threat Model 限于误删有效能力/共享合同、旧默认值被提升为当前规则、将静态无消费者冒充运行资产清零；不扩展成新的上传、IAM、恢复或验收系统。

## 3. 实际消费者与调用闭包

基线全仓 Go 精确类型/构造器/字段引用与导入搜索未发现 `task-processor/internal/temu/image` 的导入。三个入口及四个校验器的引用均局限于上述七文件闭包；未发现外部实例化或注册。`.github`、scripts、config、cmd 声明检索未发现这些名字的动态装配。`go list ./cmd/...` 得到 **31 个当前命令**，其静态依赖并集内该包为 **0**；未运行 main。无当前包专属测试文件。这些是源码证据，不是历史实例、回调、数据或远端资产的存在性审计。

| 原 contract → implementation → injection → consumer | 实际行为与 owner | 决定 |
| --- | --- | --- |
| ImageInitHandler.Name/HandleTemu → initSkuImages | 复制非空 Amazon 图片 URL 为 raw 主图 Type=1、宽高=0；SKU carousel 仅为空时填入。只写借用 Task/Product 的进程内 DTO | RETIRE 旧初始化策略；不把 raw URL 初始化提升为当前 ApprovedAsset 准入 |
| ImageValidator.Handle(pipeline.TaskContext)/HandleTemu → getImageRequirement → MainImageValidator/SkuImageValidator | 按 IsClothes 选择旧配置；Main/Sku 只由该入口构造，Main → Parallel → Single、Sku → Single，无独立消费者 | RETIRE 组合入口、四个专属 receiver/constructor/方法及配置 helper/本地 alias，不留下独立 Task 适配接口 |
| ImageValidator.ValidateImageUploadRequirement / GetImageValidationSummary | 统计 main、SKU carousel/dimension，独占写 RequiresImageUpload/TotalImageCount 并返回 map 摘要 | RETIRE 专属统计/标记与两个孤立字段；其余 mixed context 保留，不抽闲置报告 |
| ImageUploadWorker.UploadMainImagesParallel / UploadSkuImagesParallel → uploadWithContext → 调用者传入 callback | callback 为 func(*TemuTaskContext, string, string)(*models.ImageInfo, error)，基线没有当前注入；worker 只拥有 goroutine、结果收集和 raw gallery 回写 | RETIRE 该编排及两个包级结果 DTO；不能推定已经接到某个具体上传服务 |
| uploadWithContext → CurrentSkuContext；updateSkuImage → 原 gallery | 多个 goroutine 借用同一 Task context，独占读写该标记；结果是进程内值，未见 durable receipt/重启恢复 | 删除唯一旧编排及 CurrentSkuContext 字段，不新增锁/旧链 retry/recovery；其余共享能力保留 |

源码架构问题：并发 callback 的上下文标记通过共享 `CurrentSkuContext` 改写，缺少同步隔离；主图路径先启动 goroutine 后才检查 TemuProduct；主图部分失败可能返回 nil，SKU 路径全部失败也可能返回 nil。worker 未持有可判定的持久化上传结果与 UNKNOWN/retry owner。没有实际 callback 注入或运行事故证据，不能声称已发生 SKU 错绑或远端数据损坏；结果索引本身也不证明错绑。

Finding classification: **NOT_APPLICABLE（当前内部试用阻塞）**。源码无当前装配，这些风险按退休旧编排收敛，不修复/复活闲置链，不以此增加内部试用门槛，也不把“不适用当前阻塞”解释成任意重新接入旧链的许可。

## 4. 持久化、身份与平台副作用 owner

| 边界 | 实际 owner / 效果 | 本批处理 |
| --- | --- | --- |
| 当前 Product/ImageAgent/ApprovedAsset、Target/Submission、Store/Observation | 当前领域 owner 保存版本、内容 hash、授权、资产归属；原执行 receipt/fence 与 UNKNOWN owner 控制平台效果 | 全部保留，无调用切换、事实迁移或新持久化；旧 raw DTO 不成为第二事实源 |
| marketplace TEMU goods/rules/model 与当前 app/supplychain、storecenter | goods 按当次资产/店铺/类目/usage/内容/槽位构造图像；supplychain 沿原 Kernel/OfficialImageAction、精确 target scope/receipt；storecenter 使用原 merchant 凭据和 mutation access | 原 API、权限、事务、幂等与恢复均保持，不增加预检或新上传账本 |
| TemuTaskContext.AssetState | PaddedImages/PaddedImageSizes 有 dimension_builder、upload_utils 写入及 upload_processor/helpers 读取；CurrentSkuContext 仅属 worker，RequiresImageUpload/TotalImageCount 仅属 validator | **保留 AssetState 及两项 padded map，RETIRE 三项无其它消费者的专属字段**；不新增 producer，不读取/迁移真实 payload |
| 原 ImageUploadWorker callback | 调用者 callback 决定真实请求；没有当前注入证据。worker 本身只并发调用并修改 raw galleries | 删除 worker caller，不推定回调是 ImageUploadProcessor，不转移其平台结果或恢复所有权 |
| 保留 ImageUploadProcessor / ImageUploadUtils / Helpers / TemuAPIClient / upload_service | UploadSingleImage 沿 needsUpload、signature、原 padded bytes/downloader 和上传响应；服务源码使用 POST /mms/marigold/edit/commit/get_signature 与 /api/galerie/v3/store_image?sdk_version=js-1.0.6&tag_name=local-goods-image | 全部实现/API/DTO 不动；端点是仓库源码映射，不是当前平台支持性确认，不调用、不改鉴权/网络/retry/error owner |
| shared downloader、padding/drawing、vision detector | 原 downloader 负责 GET/图片信息，处理器负责进程内渲染；vision 仍可消费既有 OpenAI ChatCompleter | 下载/渲染/模型合同及 import guard 保留；未运行构造器、下载、模型请求或上传 |
| 远端已上传图片、本地业务记录/历史日志 | 各原平台/资产/数据 owner；源码无消费者不证明这些资产不存在 | 不扫描、采纳、重放、删除或清理；源码退休不授权运行态操作 |
| 当前采集/Store 观察、独立 SHEIN 登录与 Store sync | 原 acquisition/Store/observations/worker owner | 完整保留，不改身份、tenant、持久化或恢复 |

当前图片上传的成功、到期刷新、UNKNOWN 和 supplier/local artifact 差异继续按 TEMU 平台设计 §8/§8.1 与已有实现负责。删除旧 goroutine 编排不赋予自动重传、重用旧签名或绕过当前授权/资产证明的许可。

### 4.1 有效图像行为与旧策略的逐项处置

本批**没有新增 EXTRACT 代码**。当前有效图片规则已有 owner；七文件只拥有旧 Task 校验/过滤/默认值/结果搬运，不是当前批准图片的事实或规则权威。保留真实共享的下载/填充/处理/上传能力，不复制第二份 validator，也不保留专属孤岛。

| 行为 | 当前/保留 owner | 决定 |
| --- | --- | --- |
| 当前已批准资产、内容 hash、店铺/类目/usage、精确 SKU/slot 与重复/顺序约束 | marketplace/temu/goods 的 resolveImages/ValidImageObservation、supplier_images/image_artifact、model，当前 Product/ImageAgent 与官方执行合同 | 原位保留；不以旧 Amazon URL、CurrentSkuContext 或 CDN 子串替代批准、归属、receipt 和内容证明 |
| 当前格式、字节/尺寸、画幅及当次规则检查 | 当前 goods/rules/model 的既有校验；源代码包括 PNG/JPEG、3MiB、上限尺寸/面积及当前服饰/非服饰图形要求 | 原实现/测试保持；这些数值只描述基线代码，未另行确认外部平台规格，不新增规则快照或第二 validator |
| 下载获取图片信息、格式/尺寸/比例/体积验证及 padding | 当前 goods/rules/model 持有有效准入规则；原 pkg/downloader 提供下载/信息，ImagePaddingProcessor 仍被 ImageUploadUtils.padImagesIfNeeded 调用 | RETIRE 四个专属旧校验器；保留 downloader/padding 及其余真实调用，不把旧 URL 扩展名判断、自动过滤/Task 写回、5MB 默认或 padding 后重置 IsValid 复制为当前规则 |
| 图片处理、顺序稳定、尺寸图优先及原最多十图行为、carousel/dimension 构造、标注/绘制与 vision | ImageProcessor/ParallelImageProcessor、ApplyImageResults、builders/annotator/drawer/DrawingUtils/padding/vision | 原位保留；不抽新 mapper，不扩至 provider 调用或新的并发执行 |
| 原上传及 CDN 识别/needsUpload | ImageUploadProcessor/Utils/Helpers、TemuAPIClient/upload_service、mixed config 其余声明 | 原样保留；CDN strings.Contains 不是当前 URL 授权或成功 receipt 的证明，不在本批改造 |
| IsClothes 选择 1340×1785/.75 或 800×800/1.0、3MB、1..10 的 getImageRequirement | 唯一 caller 是退休 ImageValidator；本包 ImageRequirement alias 也仅供此闭包使用，handlerbase 原类型是单独共享合同 | RETIRE 选择器、本地 alias 与专属 import；保留 handlerbase 原类型及 mixed config 其余配置，不把旧默认值另抽为当前权威 |
| Amazon URL 预填、Task 上传标记/统计、共享 context 下的并发 callback 与宽松部分失败 | 三个候选 receiver 的旧编排 | RETIRE，不转成当前业务能力、兼容层或闲置纯值 helper |

闭包止于仍有保留调用的 ImagePaddingProcessor、pkg/downloader 和平台 DTO；原处理/上传/vision 子链不属于七文件的专属下游。此保留不声称整个旧 image 包已由当前命令消费，也不授权新增消费者或恢复 Task-first 设计。

## 5. 精确删除与保留矩阵（用户已确认并实施）

以下短路径均相对 `internal/temu/image/`；除七文件和 config 中专属 helper/alias/import 外，该目录其它源码全部保留。context 仅删三项准确字段。

| 路径 / 对象 | 决定 | 删除边界 / 保留条件 |
| --- | --- | --- |
| init_handler.go（122 行） | RETIRE 全文件 | ImageInitHandler/NewImageInitHandler、Name/HandleTemu/initSkuImages；不重建 alias、空 receiver 或 wrapper |
| validator.go（116 行） | RETIRE 全文件 | ImageValidator/NewImageValidator、Name/Handle/HandleTemu、ValidateImageUploadRequirement/GetImageValidationSummary；四个下游校验器见下方独立行 |
| upload_worker.go（248 行） | RETIRE 全文件 | ImageUploadWorker/NewImageUploadWorker、全部并发上传/context/回写方法，两个**包级** DTO uploadResult/uploadSkuResult |
| config.go | 仅 RETIRE getImageRequirement、本地 ImageRequirement alias 及专属 handlerbase import/注释 | imageConfig/defaultImageConfig、isCDNImage/needsUpload 全部原样保留，不删整个 mixed 文件；handlerbase.ImageRequirement 原类型保留 |
| main_validator.go（132）、sku_validator.go（130）、parallel_validator.go（67）、single_validator.go（190） | RETIRE 四全文件 | MainImageValidator/SkuImageValidator/ParallelImageValidator/SingleImageValidator、全部 constructor/方法/Task 适配、gallery helpers/并发值收集退出；不保留 alias/空 receiver 或复制旧策略 |
| processor.go、parallel_processor.go、carousel_builder.go、dimension_builder.go、dimension_models.go | 全文件原样保留 | 处理、结果顺序/合并、主图/尺寸图构造和共享类型均保持 |
| dimension_annotator.go、dimension_drawer.go、drawing_utils.go、padding_processor.go、vision_detector.go | 全文件原样保留 | 标注/绘制、图像字节、vision/provider/原 import 例外全部保持 |
| upload_processor.go、upload_utils.go、upload_helpers.go、upload_service.go | 全文件原样保留 | 上传/APIClient/签名/请求/响应和错误 owner 保持；保留 upload_processor 中名为 uploadResult 的**局部变量**，不得全局按名字删除 |
| internal/temu/api/image、api/api.go、pkg/downloader、handlerbase/pipeline/logging/ptr | 原样保留 | 平台 API/DTO/re-export（包括 ValidationResult）、handlerbase.ImageRequirement 原类型、共享数据/网络/恢复合同不在专属闭包中 |
| internal/temu/context/temu_context.go 与平台 models | RETIRE 仅三专属字段，其余原样保留 | 仅删 AssetState.CurrentSkuContext/RequiresImageUpload/TotalImageCount；AssetState/PaddedImages/PaddedImageSizes 及其初始化、TemplateState 四字段、raw galleries/ImageInfo/SetTemuProduct 全部保留 |
| 当前 Product/image、marketplace TEMU goods/rules/model、supplychain/storecenter 与原测试 | 原样保留 | 当前图片批准/选择、几何和槽位规则、上传/receipt/UNKNOWN、#795 官方方法与精确图片绑定不动 |
| AI/SKU、定价、属性、Reader/Cache/Mapping、采集/Store 观察/独立 SHEIN 登录与 Store sync | 原样保留 | §2 Must 全部维持，不按 image 同名扩大至 Amazon/SHEIN/其它平台 |

Legacy decision: **RETIRE** 七文件及专属 helper/alias/三字段；有效行为复用现有 owner，无新增抽取。
Reusable behavior: 当前资产/图像/平台校验规则与原共享下载、填充、处理、上传能力原位保留。
Current owner: 当前 Product/ImageAgent/ApprovedAsset、marketplace TEMU goods/rules/model、supplychain/storecenter 官方执行与原 API/图像/共享 owner。
Cutover/deletion condition: 用户已明确确认 §5/§6/§6.1/§7；实施前已登记 Reuse Existing Architecture / Ready 并读回，刷新消费者，原 guard RED→GREEN 精确退休、保留检查及准确实现 HEAD 的原 CI/最终检查完成。不新增兼容或第二事实源。

## 6. 验证与验收边界

设计阶段完成只读映射、31 命令静态图和文档检查，原 CI/review 仅为历史设计证据。实施前重新同步 main/消费者，原 guard 在七文件/专属声明/三字段尚在时准确 RED，删除后 GREEN；保留断言与同名值 fixture 均通过。适用回归、准确实施 HEAD 的原 CI 与最终检查维护于 PR。**设计 CI/review 不替代实施验证。** 真实下载/provider/平台上传、运行环境与用户验收均为 **NOT_RUN**。

本次实施的必要验证边界：

1. 开工同步一次 actual main，复核七个 receiver、专属 helper/alias/三字段、外部导入/直接实例化/注册和当前命令图。若发现实际消费者、具体兼容义务或无当前 owner 的有效行为，停止对应项报告。
2. 先扩原退休 guard，七文件及专属声明/三字段尚在时确认准确 RED，再精确删除并 GREEN；原保留断言也须通过。不为删除旧算法制造新业务规则、runner 或测试框架。
3. 核对 §5 保留源码/平台 DTO/context 字段/当前图片规则与测试未改；复用相关当前 goods/rules/image、SKU/定价与原 import/provider/logger/命令/worker/consumer/ceiling checks。只验证增量组合风险，不调用真实 API/provider。
4. 适用时编译 retained image/context/handlerbase/API，build current-application 与 shein-login-worker，不运行 main。完整必需检查留给稳定实施候选的原 CI，不人工重复全量。
5. 实施以准确 HEAD 的证据进入 In Review；源码/CI/AI review 不是用户验收或上线批准。#29/#301 仍有其它保留债务，保持 OPEN，不增加试用前置。

### 6.1 原检查、清单与文档的精确处置

| 现有位置 | 本次实施的最小变更 | 不得扩大 / 削弱 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go 缺席/保留断言 | 原 TestListingKitTaskBackendSourcesAreRetired 增加七条准确文件路径；精确禁止 AssetState 三专属字段，补其余 image/API/context 两项 padded map 与当前 owner 的保留断言 | **不禁止整个 internal/temu/image 目录**；不删原 root/product 覆盖、五个 mixed 包 AST 扫描、SKU 九组 fixture 或任何原检查 |
| 同一原 guard 的 mixed package/declaration 检查 | 将保留的 image 包纳入原遍历；用包级声明检查精确禁止七个 receiver/constructor、uploadResult/uploadSkuResult、getImageRequirement 和本地 ImageRequirement alias，包括 type alias/constructor var 重建；复用原 SKU 声明检查的局部做法，必要最小 fixture 留在原测试文件 | **不把新名字加进全局 ast.Ident 禁表**：保留 upload_processor.go 的同名局部 uploadResult，亦不禁其它包同名值/selector 或 handlerbase.ImageRequirement 原类型；三字段缺席只约束 AssetState 及其旧引用，不能误禁其它领域同名计数；不加广泛 skip/新扫描框架 |
| tests/import_boundaries_test.go 及原 import/provider/logger/Phase2/命令/worker guard | 沿用原检查；候选七文件与 config helper/alias 不涉及 OpenAI import 例外 | dimension_annotator/vision 仍保留，例外/deny/扫描边界/PriceHandler 唯一 management 豁免与全部 ceiling 原样保持 |
| docs/refactoring/phase2-runtime-inventory.md 当前 consumer register | core/logger → temu/image 与 integration/openai → temu/image **两行都保留**：本批后仍有真实源码消费者 | 不按七文件退休删整包 consumer；dated inventory、历史 debt list、其它行与 closure ceiling 不改 |
| docs/refactoring/code-health-decisions.md 当前 image 行 | 15 declarations 数字明确保留为日期化历史计数；窄注七文件及专属闭包退休而处理/填充/上传/vision 能力保留，消除“当前 TEMU runtime 导入该包”的错误现状表述 | 旧 cmd/temu-listing 示例仅属历史，不恢复 CI 编译目标；不改其它基线/ceiling，不宣称下层已全部退休 |
| Legacy register / 本文 / 架构索引与前批直接冲突说明 | 只标准确七文件及 helper/alias/三字段 source-retired，替代涉及这些对象的旧全保留陈述，其余与历史审批保持 | 不批量重写历史、不将当前图片/平台能力写成退休；原设计阶段仅本文与索引 |
| .github/workflows/ci.yml、commercial-readiness.yml 及其它 workflows、.golangci.yml/依赖/保护 | **零变更** | 原目标、必需/安全检查与触发条件保持；不增加试用/发布/环境前置 |

首轮独立复核的两项 P2 均先归类 **BACKLOG**（不阻塞当前试用）：专属三字段与四个校验器的孤立闭包。已独立复查并纳入本设计矩阵，不扩大当前产品、Threat Model 或实施权限。首版保留表述被本版替代；PR 保留原 findings/分类/回复与准确修订 HEAD，不把首轮结果写成无 finding。

## 7. 确认、实施与停止条件

1. 用户已明确确认冻结设计 §5/§6/§6.1/§7，授权在同一 #797 实施七文件及专属闭包。原设计和双轮意见保留历史；实施、自检与 PR 维护不等于用户验收或生产发布批准。
2. 生产修改前已在执行 Issue 登记 Reuse Existing Architecture / Ready、冻结设计 HEAD、用户结果/owner/调用路径/不变量/验证并读回；随后才按矩阵精确退休并校准原 guard/清单/直接冲突说明。两套 workflows 始终不改。准确实现状态与证据以 PR 为准。
3. 发现新增实际消费者、具体外部兼容义务、有效行为缺 owner，或需要改变身份/tenant/事实/状态/事务/恢复/平台副作用，停止受影响部分报告；不临时补 fallback/adapter，不自行建通用上传/恢复/IAM 系统。
4. 实施后以准确实现 HEAD 的保留验证、原 CI 和一次最终交付检查进入 In Review；未变化证据可明确复用，finding 先分类，修复只复核对应增量。设计通过不冒充实施完成或用户验收。
5. 合并须另有准确最终 HEAD 授权并正常 expected-head/保护；不授权 Issue 关闭、部署、真实实例/provider/凭据/数据操作。当前采集、Store 观察及独立 SHEIN 登录/Store sync 保留，本项不增加内部试用前置。
