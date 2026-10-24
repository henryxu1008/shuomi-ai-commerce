> #301 / #926 **SUPERSEDED，仅本批旧平台调用闭包**：用户已确认[完整退休设计 §1–§7/§6.1](legacy-platform-api-closure-retirement-boundary.md)。本文涉及旧 Amazon/TEMU API、SHEIN 闲置 Client/manager/DTO、SPAPI/旧平台包装配置的 KEEP/当前 owner 建议，只在该准确删除/保留矩阵内被替代；不移植旧 auth refresh/retry 或平台副作用。当前 SHEIN DTO/完整 JSON/有效规则、SessionTransport/三个读取端点、采集/原 Amazon 浏览器/debug、官方 Store、独立 SHEIN 登录与 Store 同步及原 owner/fence/UNKNOWN 保留。只抽原仓库两纯函数，不加 compat；workflow/完整 inventory/计数/检查逻辑及既有实例/数据/凭据/进程不动，不增加试用前置。以下原完整正文、历史数字/勾选/批准/证据逐字保留。

> #301 / #923 **SUPERSEDED，仅旧租户 Prompt 闭包**：用户已确认[完整退休设计 §1–§7（含 §4.2/§6.1）](legacy-tenant-prompt-closure-retirement-boundary.md)。本文涉及旧 `internal/prompt`、`internal/promptmgmt`、`shared/tenantctx`、五 YAML 及其现役 KEEP/复用建议仅在该准确矩阵内被退休决定替代；如涉及镜像打包，仅退出两条 `prompts` COPY。当前 AI/图像/采集、身份/有效规则、独立 SHEIN 登录与 Store 同步及其它 KEEP 不变。停止全新 schema 创建旧 Prompt 表，不改 Goose 登记或已有表/行，不新增替代 Store/迁移/兼容。以下原正文、历史数字/勾选/证据完整保留，不扩大实施、合并或真实资源权限。

# 旧 TEMU 图片 API 与 vision prompt 残留退休边界设计

> 后续用户已确认 [#819 TEMU剩余整链边界](legacy-temu-listing-closure-retirement-boundary.md) §5/§6/§6.1/§7。本文件对该闭包内旧 TEMU AI/SKU/spec、PriceHandler/PriceCalculator、property/context/TemplateState、pipeline/skugen及专属纯规则/管理投影/prompt的保留描述仅为历史边界，现由#819精确替代；当前平台API/规则、共享合同、SHEIN价格/两属性、采集/图像/Store观察与独立SHEIN登录/Store同步继续保留。原证据不改，不授权实例或数据操作。

## 1. 状态、用户结果与范围

- 日期：2026-10-18；执行依据：#29 / #301，用户授权“先做退休设计，确认后再实施”。
- 状态：**USER_CONFIRMED / SOURCE_RETIRED**。用户已确认 #807 冻结设计 `aa21423ac8583d510dca334365c82623b0d3d750` §5/§6/§6.1/§7；六源文件矩阵中的旧 API/re-export、vision key/default/catalog/专属变量元数据已退出。原设计 N/A 仅覆盖当时文档调查，不替代实施证据；准确实施 HEAD、CI/独立检查维护于 PR。
- 取证基线：actual main `23d4d501d21d831ad0fda80278b443473c453b8c`，含 #799 当前图片接线与 #801 已合并的旧 image 执行包退休。唯一 Writer 复用附属独占 worktree `issue29-temu-image-retirement/task-processor`，新分支 `codex/issue29-temu-image-contract-retirement-design`；已核对父 Issue/接单/关联 PR，无同范围 owner，不接管 #692。
- 当前 **Design Basis: Reuse Existing Architecture / Ready**：修改生产前已在 #29/#301 登记并读回用户结果、冻结依据、owner/路径、不变量与验证。通用 Prompt 实现、workflow 与运行数据不改；用户验收仍待独立确认。

使用者是当前内部试用、采集/Store/图像能力维护者。本批交付为 #801 §3.1 两项 BACKLOG 的精确收尾设计：旧 image API/re-export、vision key/default/catalog/专属变量元数据的实际消费者、有效行为处置、持久化/平台副作用 owner 和删除/保留/验证矩阵。当前无用户操作被此维护阻塞，**不增加内部试用前置**。没有新页面/交互，Figma N/A。

本批覆盖六个源文件：两个整文件（旧 API 233 行、YAML 16 行）和四个 mixed 文件中的专属声明/词条；无新增 EXTRACT 代码。保留其它 API/DTO、当前图像/平台/规则与全部此前保留能力。源码退休不等于历史数据库记录或远端图片不存在。

非目标：重接旧上传/vision，新增产品/IAM/目录状态机/黑名单/墓碑/迁移/恢复/验收工具；修改共享 Service/Store/Registry/HTTP 行为、其它 prompt 或其它平台同名类型；执行 main/constructor、真实下载/模型/API/上传/Preview/DryRun；读取私密配置、凭据、DB/Redis、tenant override 或历史运行日志；部署、重放、资产/数据清理和 Issue 关闭。

## 2. Design Basis、authority 与不变量

Architecture / Contract：

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Project Boundaries](project-boundaries.md)：只允许 EXTRACT / RETIRE；不新增 compatibility、fallback、双读双写、旧 ID 映射或第二事实源。
- [#801 图片执行包退休](legacy-temu-image-processing-retirement-boundary.md) §3.1 / §5：该批保留旧 API/prompt 的决定仅约束该批，已明确为后续 RETIRE 候选；本文经用户确认后，只窄替代 §5 所列对象的旧保留陈述，历史取证/批准保持。
- [TEMU 当前平台设计](temu-platform-v1.md) §3/§8：当前官方协议/签名/HTTP owner 是 `internal/integration/temu`；旧 Cookie/MALL_ID client 不作为当前官方接入基础。当前 marketplace goods/rules/model 的有效资产/几何/usage/slot 规则不从旧 API URL 猜测派生。
- [Product ImageAgent](product-image-agent-v1.md)、[官方标题与图片方法](official-skills-and-image-method-v1.md)、[#799 当前图片接线](bailian-product-image-wiring-v1.md)：当前生成/编辑/技能选择、预算、产物/ApprovedAsset、审批采用与错误/UNKNOWN owner 保持；合并源码不是 provider/用户验收。
- [Prompt Management](zitadel-tenant-prompt-management.md)：catalog/schema 与 Service Upsert 已有 known-key 语义；原 tenant Store/Registry 与通用记录读取/状态操作不因此退休。须按 §4 区分目录与持久化记录，不能把目录引用当成执行需求。
- [Store current application](store-center-current-application-v1.md)、[Store observations](store-center-platform-observations-v1.md)、[独立 SHEIN 身份持久化](shein-worker-identity-persistence-boundary.md)：原身份、事实/持久化、官方效果/receipt/UNKNOWN 与 worker/Store sync 不变。

本次实施采用 **Reuse Existing Architecture**：删除无执行消费者的旧 API 与目录资源，使用原 unknown-key/schema/Upsert 错误语义；不改 Service/Store/HTTP 接口形状、认证/tenant、事实/schema、状态/事务、幂等/retry/UNKNOWN/recovery 或平台/provider 副作用 owner。目录不再提供旧 key 是本次明确的有界变化，**不是把实施冒称 N/A**。六源文件加原检查/直接冲突文档预计低于30文件，生产删除量低于400行；不通过机械拆 PR 规避阈值。若发现必须改变保留边界或既有批准不足，停止受影响部分补适用 Independent Architecture 准入。

Must：当前图像（含 #799）、有效 API/DTO/平台规则/共享基础层、DimensionInfo/TextRenderer、AI/SKU/AIBatchInput、PriceHandler/ProfitRuleAPI、PriceManager/PriceCalculator、两种 SHEIN 属性、Reader/Cache/Mapping、四 TemplateState 字段、当前采集/Store 观察及独立 SHEIN 登录/Store sync 全部保持。其它 prompt key/template/catalog/glossary、tenant Store/Registry/HTTP 合同本批不动；不据此宣称其它旧入口全部已开放。两套及其它 workflow、必需/安全检查、保护/ceiling/依赖零变更。

Should：原检查与当前清单准确反映退休对象。Out of Scope 如 §1；无新增 Accepted Risk。Threat Model 仅覆盖误删真实共享消费者/同名类型、误把旧 URL/成功响应当当前资产或 receipt、混淆目录删除与数据删除/执行不可达；不扩成新 IAM/上传账本/目录治理平台。

## 3. 实际消费者与副作用闭包

基线全仓精确 import、声明/selector、旧 key/literal/template/变量引用检索，以及 `go list ./cmd/...` 的31命令静态依赖核对：旧 `internal/temu/api/image` 和 `promptmgmt` 均为0；未运行 main。此证据不覆盖私密配置、外部调用方、既有实例/覆盖记录或远端资产。

| contract → implementation → injection → consumer | 源码事实 | 处置 |
| --- | --- | --- |
| root api → image.API / NewAPI 与四 DTO alias | 子包仅被 root api.go 重导出和原退休 guard 引用；没有业务构造/调用。root api 仍被 context、rules/text_check_handler、spec/runtime_input 消费，但只用其它合同 | RETIRE image 子包及准确六个重导出声明，不删整个 root API 或 client |
| image.NewAPI → 借用 client.ClientAPI + logrus → shared ImageDownloader | 无当前构造/注入；五 receiver 方法为 NeedsUpload、DownloadImage、GetUploadSignature、UploadWithSignature、Upload | RETIRE manager/方法/专属DTO；原 client/downloader/logger 保留，不新增替代 manager |
| GetUploadSignature / UploadWithSignature → SendTEMURequest | POST /mms/marigold/edit/commit/get_signature 与 /api/galerie/v3/store_image?sdk_version=js-1.0.6&tag_name=local-goods-image；GET 由共享 downloader 负责 | 只退出 caller，不调用端点，不转移凭据/网络/receipt/UNKNOWN owner；端点仅源码映射，不是当前平台支持性确认 |
| 借用 client → AuthManager / retry / CookieManager → StoreRuntime.GetStoreAPI | SendTEMURequest 经鉴权/重试，可调用 SetStorePauseStatus(storeID, true, auth_expired)；另有 client 构造期 initializeMallID → UpdateStoreId，不是每次图片请求都执行初始化 | 原 client/鉴权/cookie/重试及 listingadmin.StoreAPI 所属持久化 owner 保留；只退出 image caller，不删除或转移 Store 副作用，不读写实例状态 |
| vision YAML / KTemuVisionDetectorDetect → catalog/manifest → schema/Upsert | #801 后无 render/execution caller；只余 key/default/catalog 元数据引用，目录本身不提供检测执行能力 | RETIRE 此 key/default/两目录词条与专属变量元数据；不退休通用 Prompt Management |
| Base64Image glossary → vision manifest/template | 全仓源码/模板仅此一个变量使用；其它图像/编码能力不依赖该词条 | RETIRE 准确词条；不禁止其它业务使用 base64，不删共享编码或通用 glossary |

架构问题：NeedsUpload 用固定位置子串识别 CDN，且空 URL 可直接被包装成“无需上传”结果；UploadWithSignature 只依据响应 URL 并自报 jpg，不拥有当前批准资产/店铺/usage/receipt 事实。以上旧实现不能作为当前 URL 授权或上传成功权威；无当前装配/事故证据，分类 **NOT_APPLICABLE（当前内部试用阻塞）**，通过退休收敛，不修旧链或新增上传账本。

## 4. 持久化、目录准入与数据 owner

| 边界 | 现有 owner / 本批后的语义 | 删除边界 |
| --- | --- | --- |
| 当前 Product/ImageAgent/ApprovedAsset、官方图片效果与 Store/observations | 原资产、预算/计量、权限/tenant、上传/receipt/UNKNOWN、恢复 owner 保持 | 不删除/迁移/扫描产物、URL、数据库/对象或远端图片 |
| 借用旧 client 的 Store ID / auth pause | 原 StoreRuntime / listingadmin.StoreAPI 是 UpdateStoreId / SetStorePauseStatus 的调用边界，client/auth/cookie/retry 保留且其它 API 仍消费 | 不把旧 image API 描述为 HTTP-only，不改原持久化/恢复协议或迁移至当前 application / SHEIN worker；无运行 Store 操作 |
| tenant_prompt_templates | 原 prompt.TenantPromptTemplate / TenantPromptStore / GormTenantPromptStore；tenant_id+key 唯一，tenant-qualified 查询/更新保持 | 不改 schema/索引/表/字段/查询谓词，不读取/删除/禁用/迁移/回填任何记录；是否存在旧 key 记录 UNKNOWN |
| ListTemplateCatalog / GetTemplateSchema | 移除该目录项后不列出旧 key；GetTemplateSchema 按原 unknown-key 合同返回 ErrTemplateNotFound，原 HTTP handler 映射404 | 不保留隐藏项/兼容 alias，不新增“已退休”状态机或特殊响应 |
| Service UpsertTenantTemplate | store 配置时，旧 key 经原 GetTemplateSchema 拒绝，在 store.Upsert/GetEnabled 前返回 ErrTemplateNotFound；无 store 时原 ErrServiceUnavailable/503 不变 | 不改 Service/Store 实现或错误码，不为旧 key 开写入旁路 |
| ListTenantTemplates / SetTenantTemplateStatus | 保持通用记录语义：ListTenant 可返回已有旧 key 行；SetStatus 不查 catalog，按原 tenant/key 更新已有行的 enabled，行不存在404、无store503 | 不擅自过滤列表、增加写入准入、清理/禁用数据或修改原接口；enabled 是行标志，不是检测功能可用性或效果承诺 |
| Registry / loader / watcher / GetEnabled / GetTenantFromContext | 删除跟踪中的默认 YAML 不清空运行缓存或DB；generic Store/Registry 不校验 catalog，已有 enabled 行仍可能按精确 tenant/key 被读取/渲染 | 不增加全局禁 key、墓碑/黑名单、fallback 或新consumer；不声称“所有方式都无法渲染该字符串”，没有业务执行 caller 才是本次取证结论 |
| 其它 key、共享图像/平台 API、独立 SHEIN 身份/Store sync | 原 owner 和原合同保持 | 不按同名 ImageAPI/ValidationResult/UploadResult 跟删其它包或当前能力 |

原 Prompt Management 文档“writes only for catalog keys”概述不能被扩大解释为整个 Store/所有方法都校验 catalog：实际 Upsert 有准入，PATCH status 合同和实现只约束已有行；generic Registry 也不查目录。本文明确这个区别，**不借退休改写公共管理/租户协议**。上述原 HTTP tenant 来源源码不是当前认证接线或安全验收证据；本批不开放入口、扩身份边界或修建 IAM。

已有覆盖行的原始管理和 renderer 不等于保留旧 vision 执行链；不添加任何调用它的 wrapper、兼容路径或回退。若后续发现具体当前外部可观察契约/真实执行消费者，停止相关项报告用户，不能自行推定兼容义务或清理数据。

### 4.1 有效行为 EXTRACT / RETIRE

本批**无新增 EXTRACT 代码**。有效的图像下载/格式与像素检查、当前资产/平台规则、通用 Prompt 渲染/tenant 存储/目录校验已有 owner，全部原位保留；旧 CDN 猜测、上传搬运、无执行者的 prompt 及仅服务它的 metadata 不复制为闲置工具。

## 5. 精确删除与保留矩阵（用户已确认并实施）

| 路径 / 对象 | 决定 | 边界 |
| --- | --- | --- |
| internal/temu/api/image/image.go（233行）及整个包 | RETIRE 整文件/目录 | API/NewAPI/五方法；ValidationResult、UploadRequest、UploadResponse、TemuUploadResponse、UploadResult、SignatureResponse、UploadSignature、ProcessingOptions、PaddingResult 全部专属DTO；不留 alias/空壳/改名副本 |
| internal/temu/api/api.go | RETIRE 专属 import + 六声明 | image import；ImageAPI = image.API、NewImageAPI = image.NewAPI；UploadSignature/SignatureResponse/UploadResult/ValidationResult 四 alias 及因此为空的专属注释/type block；其它 root API/constructor/DTO、product.ImageInfo/client/category/query/inventory/pricing 保持 |
| internal/prompt/keys.go | RETIRE 精确 key/专属注释 | 仅 KTemuVisionDetectorDetect = "temu.vision_detector.detect" 与 vision_detector.yaml 注释；其它常量原样 |
| prompts/temu/vision_detector.yaml（16行） | RETIRE 整文件 | 唯一默认检测模板；其它 YAML/加载根/扫描规则/运行覆盖配置不动 |
| internal/promptmgmt/catalog.go | RETIRE 一个 key entry | 只移除 promptTemplateCatalogKeys 中旧常量条目；组装、分组/排序/错误语义与其它 key 不动 |
| internal/promptmgmt/catalog_manifest.go | RETIRE 一个 metadata block + 一个 glossary entry | 旧 key 的 label/description/scopes/Base64Image variables block；准确 Base64Image glossary 词条。实施刷新引用若有真实新共享consumer，仅停该词条报告，不按名字全局删除 |
| promptmgmt Service/HTTP/types、prompt Store/Registry/loader/watcher、tenant_prompt_templates | 本批原样保留 | §4 全部数据/通用管理语义保持；不新增清理/黑名单/兼容机制，也不为保留旧检测能力新增consumer |
| TEMU root/client及其它API/DTO、property.ValidationResult、SHEIN ImageAPI等同名合同 | 原样保留 | 实际 root API 消费者仍在；不整包清扫，不全局禁泛型名字或改为另一旧 owner |
| 当前图像/规则/平台、shared imagex/downloader、DimensionInfo/TextRenderer及此前全部能力 | 原样保留 | 含#799接线、AI/SKU/定价/两SHEIN属性/Reader/Cache/Mapping/四TemplateState/采集/Store观察/独立SHEIN登录与Store sync；不运行/部署 |

Legacy decision: **RETIRE** 旧 image API/专属 re-export 与无执行 caller 的 vision key/default/catalog/专属变量元数据。
Reusable behavior: 当前图像/平台/共享下载检查、通用 Prompt 渲染/tenant Store/管理合同沿原 owner 保留，无新增抽取。
Current owner: Product/ImageAgent/ApprovedAsset、marketplace TEMU goods/rules/model、原官方effect/receipt/UNKNOWN/Store、internal/pkg/imagex/downloader、Prompt Management / prompt Store/Registry 与独立SHEIN身份持久化。
Cutover/deletion condition: 用户确认 §5/§6/§6.1/§7 → 执行Issue登记/读回 Reuse Existing Architecture / Ready → 刷新消费者 → 原guard与目录/Upsert测试 RED→GREEN → 准确实施HEAD原CI/最终检查。无运行切换、数据迁移或兼容义务。

## 6. 验证与验收边界

设计阶段只读映射六文件、原 guard、实际 root API 消费者、目录/Store/Registry/HTTP 合同与31命令依赖；当时只交本文/索引；实施按下列冻结要求验证并校准原清单。真实下载、平台上传、模型/provider、运行实例与用户验收均 **NOT_RUN**；静态未见 caller 不等于外部消费者或数据不存在。

实施的必要验证（冻结设计要求）：

1. 开工同步一次 actual main；刷新 image子包/六重导出、旧key/literal/template/Base64Image 所有消费者及31命令依赖。出现新真实consumer、有效行为缺owner或具体兼容义务，停止对应项报告。
2. 原退休 guard 在旧 API/YAML 尚在时准确RED；原 promptmgmt 测试使用已有 stub store，针对旧 literal 验证 catalog/schema/Upsert 不可用与写 store 前拒绝，在 metadata 尚在时确认RED。再精确删除/GREEN；无新runner/专项环境。
3. 保留并复用其它 catalog/schema/Upsert、unknown-key 不触 store、通用 status、原 Store tenant隔离/GetEnabled/ListTenant/SetEnabled、Registry/loader/renderer 测试；不借测试新增旧 key 的生产caller。必要时在原测试用例追加最小 orphan-row 断言，确认列表/status不因目录删除被过滤/改写，不增加新API行为。
4. 复用 retained TEMU root/client/context/rules/spec/API/当前goods、共享imagex/downloader、当前图像/Prompt既有回归及两个当前命令 build；只验证本轮组合风险，不运行main。完整必需CI留给稳定实施候选，不人工重复全量。
5. 准确实现HEAD原CI/最终检查完成后 In Review；PR CI、main CI、运行/平台/用户验收分别记录。#29/#301其它债务保持OPEN，不增加内部试用前置。

### 6.1 原检查、清单与文档的精确处置

| 原位置 | 实施的最小变更 | 不得削弱 / 扩大 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 原严格缺席列表增加整个 internal/temu/api/image 与 vision YAML；原root API保留token仅退出上述image专属token，移除旧image.go保留项；复用原局部AST声明做法防root api六声明/constructor var或alias重建，keys/catalog/manifest精确禁止旧key资源重建 | 保留原image/孤立ImageRequirement/AssetState缺席、五mixed包/SKU九fixture、Publish/身份/采集/Store/当前图像/安全断言与其它API/DTO保留。root api 名字检查只约束包级声明，不禁其它包同名type/field/selector/local，不新建通用扫描器或广泛skip |
| internal/promptmgmt/service_test.go、必要原api/handler测试 | 在已有stub/unknown-key路径补旧literal的catalog缺席/schema ErrTemplateNotFound/Upsert无store写入；保留无store503和已有raw-list/status/其它key测试 | 不用真实DB/接口，不删tenant隔离/通用store/renderer测试，不把catalog移除写成已有行消失或所有generic rendering被禁 |
| tests/import_boundaries_test.go、phase2-runtime-inventory | **基线无本子包专属OpenAI豁免或consumer行，本批零变更**；复用原consumer/边界/ceiling检查 | 不照搬#801删除logger/OpenAI行，不改loader/root/ignore/保护/ceilings或日期化统计；若刷新出现新consumer，先报告 |
| Legacy Register、code-health image/API说明、本文/索引、#801及直接冲突说明、Prompt Management概述 | 用户确认后窄标记本矩阵source-retired，原管理语义按§4澄清；删除不再存在的API测试目标，保留root/client/其它API/Prompt当前目标；原声明数/审批/取证保留为历史 | 不将整个TEMU/Prompt owner或所有同名DTO退休，不重写历史证据；设计阶段只交本文与索引 |
| .github/workflows/ci.yml、commercial-readiness.yml及其它workflows、依赖/保护 | **零变更** | 必需/安全检查、编译目标、触发/并发/权限/保护与全部ceiling保持，无新试用/发布前置 |

## 7. 确认、实施与停止条件

1. 用户已确认冻结设计 §5/§6/§6.1/§7，授权在同一主要 PR #807 继续实施；本次只执行已确认矩阵，不从原设计 N/A 推定其它业务修改权限。
2. 实施前已在执行 Issue 记录并读回 Reuse Existing Architecture / Ready、冻结设计 HEAD、用户结果/owner/调用路径/不变量/验证，再按矩阵精确退休及校准原检查与清单。
3. 新真实consumer、具体外部可观察兼容义务、有效行为缺owner或必须改变身份/tenant/事实/状态/事务/管理/恢复/副作用边界时，停止受影响部分报告，补适用准入；不临时造adapter/fallback、清理覆盖数据或扩成目录治理/IAM平台。
4. 实施完成交准确HEAD原CI/最终独立检查，finding先分类；未变有效证据明确复用，非Blocker不自动扩范围或重开全局设计。实现者/CI/AI检查不签发用户验收。
5. 合并需准确最终HEAD另授权并正常expected-head；#29/#301保持OPEN，不授权部署、真实API/provider/实例/凭据/业务数据操作。保留当前图像/共享/采集/Store与独立SHEIN登录/Store sync，不增加内部试用前置。
