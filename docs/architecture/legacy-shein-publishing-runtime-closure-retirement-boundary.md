> #301 / #923 **SUPERSEDED，仅旧租户 Prompt 闭包**：用户已确认[完整退休设计 §1–§7（含 §4.2/§6.1）](legacy-tenant-prompt-closure-retirement-boundary.md)。本文涉及旧 `internal/prompt`、`internal/promptmgmt`、`shared/tenantctx`、五 YAML 及其现役 KEEP/复用建议仅在该准确矩阵内被退休决定替代；如涉及镜像打包，仅退出两条 `prompts` COPY。当前 AI/图像/采集、身份/有效规则、独立 SHEIN 登录与 Store 同步及其它 KEEP 不变。停止全新 schema 创建旧 Prompt 表，不改 Goose 登记或已有表/行，不新增替代 Store/迁移/兼容。以下原正文、历史数字/勾选/证据完整保留，不扩大实施、合并或真实资源权限。

> 当前适用边界（#301 / #912）：用户已确认并按[旧 SHEIN 闲置策略与辅助合同完整退休闭包](legacy-shein-policy-contract-closure-retirement-boundary.md) §1–§7（含 §6.1）整批实施，仅替代本文件中generationtopics/namelimit 与本批价格/活动/属性辅助的 KEEP；当前草稿严格解码、版本及有效规则保持及对应旧 KEEP 检查；准确删除/保留矩阵以新设计 §2/§5/§6.1 为准，其它当前 API/有效规则/采集/独立 SHEIN 登录与 Store 同步不变。下方原历史正文、数字和证据逐字保留，不提供旧 JSON 迁移或运行/数据操作授权。

> #301 / #904 用户确认的[完整 JSON 别名退休设计](legacy-shein-draft-json-alias-retirement-v1.md) §5/§6.1 仅替代本文三旧 Package 字段、回填/镜像/custom JSON、summary-only SKU 就绪回退及旧绑定/规则版本的 KEEP 或双输出要求。当前仅 canonical 字段，typed v1.2 / diagnostic v2.2 / binding go-json.v2；当前类型、有效规则、严格解码、原持久化 owner、独立 SHEIN 登录与 Store 同步保留。不迁移已有记录、不改 workflow/inventory、不增加试用前置。以下历史正文原样保留。

# 旧 SHEIN Publishing 运行适配、解析器与提交恢复完整闭包退休边界

> #833 [旧数字内容配置完整闭包](legacy-numeric-content-management-closure-retirement-boundary.md)：冻结设计 d139bcccc 已确认并整批实施，三族无生产注入的词库/话题配置仓库读写、DTO、全局 setter、专属 schema/迁移和调用尾部源码退出；此前整体保留表述仅在该设计 §5/§6.1 的准确对象上被替代。当前默认清理直接复用原 content 内存工厂，品牌/字段/runtimeCtx、generationtopics 纯规则、Store/API/schema/权限与独立登录/Store 同步保持。物理对象 UNKNOWN、不操作；原历史矩阵/计数保留，全部 workflow/inventory 不改，无新试用前置。

> #831 [Store 管理闭包校准](legacy-numeric-store-management-closure-retirement-boundary.md)只在其 §5 七项无入口 Store 管理方法、五 helper 与 67 配置/查询字段上进一步收窄此前 Store 保留范围。当前账号读/同步、schema 辅助、词库/话题、九方法 StoreAPI、共享值类型及独立 worker 保持原 owner；本设计其余保留和历史证据不变，无数据/运行操作。

Refs #29 / #301。调查基线：`8add3a4455123c04871677bf2c1a3b1366307703`。

> Design Basis：**Independent Architecture**。状态：**IMPLEMENTATION_READY / SOURCE_IMPLEMENTED_IN_REVIEW**。用户已确认冻结设计 `a88ceb527` 的 §5/§6/§6.1/§7；独立设计检查完成无 findings，#29/#301 Ready 后在同一 #827 实施。准确实现 HEAD、CI、最终独立检查写 PR；合并/部署/实例/数据操作未授权。

## 1. 用户结果与一个 Delivery Batch

使用者是维护当前内部试用、采集、草稿/校验、SHEIN 官方上传审核/首次上架及 Store/独立登录同步的操作者。本批把共享 Publishing 包里已经没有当前消费者的旧远端运行链完整退出，保留当前纯草稿和平台校验所需的值、规则与严格输入边界。

目前没有证实某条核心用户操作必须等待本批；这是用户明确要求的源码退休维护，**不增加内部试用前置**。不新增页面、菜单、验收平台、人工操作或环境要求；UI/IA/Figma N/A。

同一批覆盖 managed/runtime 工厂、旧类目/属性/销售属性解析、持久缓存、图片/翻译/提交/回读/重试/恢复及其最后消费者退出后孤立的通用 Submission helper、旧策略和 Prompt 静态资源。不按包、helper、DTO、测试或文件数机械拆 PR。当前官方 Submission、草稿和离线规则是闭包的停止边界，不是兼容旧执行器的理由。

## 2. Product Authority、Design Basis 与不变量

依据：

- [全新系统产品基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)。
- 已确认的 [SHEIN 数字刊登退休边界](legacy-shein-listing-closure-retirement-boundary.md) 与 [#824 支撑闭包边界](legacy-numeric-listing-support-closure-retirement-boundary.md)。本文只在 §5 准确对象上替代“当时仍有消费者”的保留结论；历史证据和其余当前能力保持。
- [当前 SHEIN 供应链](my-supply-chain-shein-v1.md)、[原版本审核/首次上架](shein-publication-closure-v1.md)、[Prompt 管理](zitadel-tenant-prompt-management.md)。现有官方回执、执行许可和 UNKNOWN owner 不变。

本批范围超过 30 文件，且必须区分同包中的当前值合同与旧执行状态/持久缓存，采用 Independent Architecture；这不是新状态机、当前业务 owner 迁移或旧数据迁移许可。复用当前能力和现有 Go/GORM/strict JSON/Prompt 组件，不引入新框架或依赖。

Must：当前草稿确定性组装、严格编解码、离线规则及 `shein.offline_package.v1.1`、有效平台 API/DTO/规则、价格/两种 SHEIN 属性、AI/文本/图像与有效共享合同、采集/Store 观察、独立 SHEIN 登录 worker/Store 同步保持。当前 Organization/tenant/owner/权限、事实/schema、事务、幂等/预算/计费、retry/UNKNOWN/recovery 不变。两套及全部 workflow、必需/安全检查、依赖、`.golangci.yml`、ceilings 和分支保护不改。

Should：当前模块不再编译无入口的旧执行实现；准确防复活与当前能力证据随同一 PR 交付。无新增 Accepted Risk。

Out of Scope：当前官方发布重写、运行实例/数据库/Redis/消息/Profile/凭据/平台/模型操作、数据清理或旧记录迁移、关闭父 Issue、部署、兼容 shell、fallback、双读/双写、第二事实源、通用调度/恢复/指标/验收系统。既有 Prompt 覆盖记录不读取、不删除、不禁用或回填。

Threat Model：误删纯值/安全规则；把 Package 旧状态误当当前 Submission；仅因同包仍被 import 就保留所有运行链；同名 managed/Submission/Inspection/API 误删；用旧测试恢复旧 owner；把源码退休当数据或环境退休。这里不扩张当前平台/运行威胁模型。

## 3. 实际消费者、注入与根因

调查对全部 **3975 tracked Go** 作标准 AST 解析（含分组、alias、blank import 和未编译源文件）；对 **400 个生产包/2204 个当前编译源文件** 作类型定义/使用边核对。另对相关 64 个测试包实例（含 integration tag）作类型检查，1303 个语法单元包含测试变体的重复源文件，不作为独立文件计数。未执行应用 constructor、main、测试用例或 provider。

类型图只是候选证据，不能把“生产零引用”自动等同 RETIRE：当前 Kernel 的时钟/ID/claim-token/manual-resolution 测试注入 API、敏感词/话题当前公共合同明确保留。删除集合以 §5 和附录逐项边界为准。

| 合同 → 实现 → 注入 → 消费者 | 当前事实 | 处置 |
| --- | --- | --- |
| Listing Record → `marketplace/shein/draft.Builder` → `publishing/shein.NewAssembler(AssemblerConfig{})` | 当前构造不注入远端 resolver；把精确 Catalog/ApprovedAsset 投影为本地不完整 Package | 保留当前组装、值/默认行为和唯一 Package 输出；不把旧 factory 填回空配置 |
| `marketplace/shein/validator` → Publishing 严格 clone/prepare/validate → retained workspace readiness | 七个包外生产文件只消费值、纯组装/编解码和 readiness/规则 | 保留规则、错误分类、版本、大小/深度/重复/未知字段限制及原拒绝语义 |
| `publishing/sheinmanaged` → 原 BaseAPIClient/StoreID factory → 旧 Publishing resolver/API ports | 全部 tracked Go 无包外生产 importer，无当前 composition 注入 | 9 生产/1 测试文件整退；**不是** `sheinloginmanaged` |
| Publishing `runtime_*` / resolver / cache / remote submit → 旧 API/AI/缓存/lease/recovery | 构造器和执行声明的消费者在本批闭包内部；当前草稿/validator 不消费这些实现 | 旧实现及专属输入/输出、helper/fixture 整退，不迁移 owner 或造替代入口 |
| 旧 Publishing → `listing/submission` 旧通用 attempt/event/lock/回读恢复 helper | 原数字 caller 已退，剩余消费者同属本批；与当前 ExecutionKernel 不同协议 | 精确退休 51 生产文件；保留当前 Kernel/官方协议 13 个业务源码文件、原 doc.go 及全部有效测试 |
| 旧解析/优化 → `shein/category`、`aicache`、旧 content/submitprep → 18 Prompt keys | 只余本批执行消费者及静态目录引用；目录不是执行需求 | 专属代码和四份 YAML/18 key 元数据同批退；不改变通用 Prompt owner |
| 当前文本清理/话题 → content/submitprep 原服务 → listingadmin 原仓库 | 当前纯组装/清理仍消费租户词库/话题及授权品牌规则 | 保留当前读取、配置/隔离和四张现用表；只退旧提交错误驱动的自动词条写回/重试 |

根因是同包混合当前纯值能力与旧运行/持久化协议，包级 import 被误当成每个内部实现仍有当前消费者。删除顶层数字任务后，无入口的旧提交状态与恢复/helper/Prompt 仍会被编译，继续造成错误装配和 owner 判断风险。本批按声明和真实注入边切断，不能靠保留一个新 facade 收口。

## 4. 持久化、状态与平台副作用 owner

| 边界 | 原 owner / 副作用 | 本批边界 |
| --- | --- | --- |
| 当前 Listing Record / Product / ApprovedAsset / CompletionDraft | 不可变记录、精确版本输入与当前确认/保存 UoW | 原表/JSON当前有效载荷/版本/事务保持；不访问或重写已有记录 |
| 当前 ExecutionKernel + official receipt/intent/effects，含 #821 SHEIN shelf | 当前许可、幂等键/载荷、claim/fence、UNKNOWN/人工 resolution/retry | 13 生产文件及当前 persistence/app/Store composition 全部不改；不接管旧 lease/recovery |
| 旧 Package Submission/SubmissionState/SubmissionEvents 与 Inspection | 已退出的旧数字提交/工作区状态；当前七个消费者不读写这些事实，当前本地 writer 不产生这些值 | 删除字段、独占 DTO/镜像分支及旧状态方法；不迁移/回读/同步历史 JSON，不建立新事实 owner |
| 旧 GormResolutionCacheStore / SheinResolutionCacheEntry | 旧 resolver 的 GORM 持久缓存；无当前构造/注入 | 只退模型/接口/构造/方法，保留当前值元数据 ResolutionCacheInfo；不查表或执行 DROP/AutoMigrate/缓存清理。实际历史对象/行是否存在 UNKNOWN |
| Store/词库/话题 | 原 listingadmin Store、SensitiveWord、GenerationTopicPolicy/Override；独立 worker 原账户/Store DB、Redis/Profile | 原 schema/约束/owner/tenant、配置接口和服务保持。旧错误词发现写回不转交当前应用，已有记录保持 |
| 旧上传/翻译/发布/查询/AI 请求 | 本批旧 runtime/factory/helper，旧响应/attempt/回读状态 | 不执行任何请求，不向当前 Kernel/worker 转交旧请求/结果/重试/UNKNOWN；canonical 平台 API 和当前官方路径保持 |
| Prompt | 原 tenant Store/Registry/HTTP；catalog/schema/Upsert 的 known-key 准入 | 只退 18 个静态执行 key/default/目录项。generic List/SetStatus/GetEnabled/renderer 仍沿原 tenant/key 语义；旧行存在性 UNKNOWN，不伪造消失或禁止 generic rendering |

失败、响应丢失、重试、重启、取消、并发、撤权后的当前行为继续由各现有 owner 负责。退休的旧状态协议不做恢复、归并或数据迁移，不新增跨库一致性问题，因此不引入 outbox/Saga 或协调器。源码扫描与编译不证明任何实例已经停止，也不授权探查实例。

## 5. 精确删除、保留与复用矩阵

### 5.1 整文件闭包

附录 A 是准确逐文件删除集合：**152 生产 Go + 103 专属测试 Go + 4 YAML = 259 文件**。不按目录名称扩删。

| 文件族 | 生产数 | 删除内容 | 停止边界 |
| --- | ---: | --- | --- |
| publishing/shein | 80 | 旧 API builders、解析/AI/cache/runtime、上传/翻译、submission/回读/lease/retry 及专属 helper | 原当前 Package/纯组装/规则/严格 codec 的 40 生产文件按 mixed 或原样保留 |
| publishing/sheinmanaged | 9 | 全部旧 managed publishing factory/adapter，包括仅 package 声明的 bridge.go | sheinloginmanaged 及独立登录/同步全部保留 |
| listing/submission | 51 | 旧 attempt/event/lock/direct-submit/refresh/backfill/projection/phase-detail 等专属协议 | 当前 13 Kernel/official/平台 effect 文件及原 doc.go、当前持久化和测试注入 API不改 |
| shein/category、aicache、content、submitprep | 8 | 旧 category manager/cache、optimizer/string_sanitizer/text_cleaner、language/localized_content/localized_list | 当前词库/品牌/话题服务与规则保留，不整包删 content/submitprep |
| marketplace/shein/publishing | 4 | projection_policy、phase_detail_policy、remote_record_policy、sensitive_retry_policy | 其余纯平台规则，包括文本/价格/图像/模板/属性/重量/站点/POD 等原位保留 |

不得删除整个 `publishing/shein`、`listing/submission`、`marketplace/shein/publishing` 或任何 `shein/api` 根。不得用 `*managed*`、`Submission*` 或 `Inspection*` 全仓命名匹配删当前同名合同。

### 5.2 mixed 生产源码：27 文件与三个 Prompt metadata 文件

附录 B 按原文件/准确包级声明列出允许删除项。除明列字段/分支和机械 unused import 外，其余保留定义不改；删除方法名只在原 receiver 上生效。

- `model.go`：删除八种 Inspection DTO；`BuildRequest` 只退 `Text`、`SheinStoreID`；`Package` 只退 `Inspection`、`Submission`、`SubmissionState`、`SubmissionEvents`。不删当前 Category/Attribute/SaleAttribute/FinalDraft 等值，不压缩其它字段/tag。
- `semantic_fields.go`：只退 Submission 两字段的 conflict 项和镜像分支。DraftPayload/RequestDraft、PreviewPayload/PreviewProduct、FinalSubmissionDraft/FinalDraft 是当前 writer/严格 codec 仍实际消费的值对，其输出/反冲突检查保持，不扩展历史兼容或新增 reader/fallback。
- `resolution_cache_store.go`：只保留原 `ResolutionCacheInfo` 全字段/tag；移除旧 GORM 模型、接口、方法、缓存类型/命中来源常量。它不再拥有数据缓存运行态。
- `submit_prep_sensitive_adapter.go`：只保留 `sheinSensitiveWordSanitizer` 与 `newSheinSensitiveWordSanitizer`；当前构造直接用原 submitprep 服务，旧 Clean/Retry 和配置转发退出，不重建生产包装。
- `shein/submitprep/sensitive_words.go`：保留原 service factory、tenant 配置 setter、词库/话题读取 overlay；只退附录中旧 Product 清理/错误重试/自动发现词条写回及其专属 helper。listingadmin 仓库的 owner 拒绝与合法 CRUD/读取合同不变。
- `generation_topic_runtime.go`、content/word.go、generationtopics/registry.go **整文件保留**，包括当前配置和有效测试接口，不因生产无直接 setter 调用而机械删除。
- 当前公开 resolver/size/pricing/text 的纯值接口和 Assembler 配置不扩展、不替换 owner；当前组装默认路径与有效配置行为保持。删除的是无注入的具体旧实现。
- `internal/prompt/keys.go`、`internal/promptmgmt/catalog.go`、`catalog_manifest.go`：只退附录 C 的 18 key 与对应目录/变量 block/专属注释；其余 key、共享 glossary、schema 组装/排序/错误和 Store/Registry/HTTP 实现不改。

三个 Prompt metadata 文件加 27 core mixed，**共 30 mixed 生产文件**。本轮无新生产 EXTRACT 代码；有效能力已经由当前 owner 实现，原位复用。只为保留有效断言迁移必要的中立测试 fixture，不创建通用测试框架。

### 5.3 当前保留矩阵

| 保留能力 | 精确原 owner / 约束 |
| --- | --- |
| 当前纯 SHEIN 草稿 | marketplace/shein/draft.Builder → 原 Assembler/Package、common/canonical/Product/ApprovedAsset；当前事实过滤、无远端默认注入、输出大小与取消边界保持 |
| 离线校验 | validator + #824 保留的 workspace 11 生产/10 测试；`shein.offline_package.v1.1`、payload prepare/validate/readiness、错误/UNKNOWN/阻塞分类和输入限制保持 |
| 当前 Package 载荷 | 当前 Draft/Preview/Final 值合同、Category/Attribute/SaleAttribute/Size/Pricing/ReviewNotes/Metadata、两个 SHEIN 共享属性及当前纯 matcher/formatter/policy 不因旧 factory 退出而删 |
| 当前 Submission | execution_intent_reader、execution_kernel、execution_kernel_service、official_intent/receipt/resolution、shein_shelf、temu_effects/intent/receipt/supplier_image、tiktok_effects、walmart_effects 共 13 生产文件；原完整持久化/授权/事务/故障与测试注入 API保持 |
| 当前官方平台产品链 | #821 官方上传/原版本审核/首次上架/库存回读、所有当前 Store/Integration/Record/Submission owner；原 API/DTO/transport 和未知结果控制不变 |
| 当前共享/采集/图像 | PriceHandler/PriceManager/PriceCalculator、Reader/Cache/Mapping、有效平台规则、来源采集、当前 Product/Asset/ImageAgent/POD/AI 及共享基础层保持；不把当前 AI/provider 模块归入旧客户端闭包 |
| 身份/Store/独立 worker | listingadmin Store/词库/话题及四张表，sheinlogin/bootstrap/sheinloginmanaged（实际路径 `internal/sheinloginmanaged`）、app/runtime/sheinloginworker、Store observations/同步/查重、浏览器/Profile/凭据/健康/关闭保持 |
| Prompt | 非本批 key/default/catalog，generic Store/Registry/loader/watcher/render、tenant 拒绝、原有覆盖记录和通用列表/status 管理语义保持 |

注：Package 的旧提交/Inspection 输入字段退出后，严格 decoder 对这些旧 key 按现有 unknown-field 路径拒绝；当前 writer 从不产生其非空业务状态，不引入数据转换。其余当前 JSON 值对、2 MiB/64 层、重复/未知字段、别名冲突和 typed-nil/private-cache 不可注入断言保持。这是旧状态形状退出，不修改当前 payload/readiness 规则或版本。

### 5.4 Prompt 静态闭包与 legacy decision

附录 C 的 18 key 分属四份 YAML，所有执行 caller 均在本批；catalog/manifest 引用不作为保留执行需求。沿 #807 已批准的同类静态退休边界：目录/schema/Upsert 对退出 key 按原 unknown-key 行为拒绝且不写 Store；List/SetStatus/GetEnabled/render 保持原 generic 行语义。旧行可能继续被 generic 工具精确读/render，不意味着旧执行能力可用；不加全局禁 key、墓碑、过滤、清理或新 consumer。

```text
Legacy decision: RETIRE
Reusable behavior: 当前组装/值、规则、codec、安全与租户检查沿现有 owner 原位保留；必要中立 test fixture 转入仍消费它的原测试。
Current owner: 当前 Product/Asset/Listing Record/Submission Kernel/Marketplace/Store、Prompt 与独立 SHEIN 登录各原 owner。
Cutover/deletion condition: 用户确认 + 独立准入 + Issue Ready → 原 guard RED → 本矩阵源码删除/有效测试转接 GREEN → 准确实现 CI/最终独立复核；不转交旧运行事实或清理数据。
```

## 6. 实施、验证与验收

设计阶段的 Go AST/类型调查是静态证据，不写成测试用例 PASS。实施使用原隔离测试和必要构建；不运行实际应用、真实 provider、既有 DB/队列/浏览器或旧 recovery 工具，不把源码/CI 当运行或产品验收。

后续获明确实施授权并登记 Ready 后，在同一主要 PR：

1. 开工同步一次实际 main，复核附录/当前接单；稳定候选前再核对真实 main，只处理组合风险，不持续 merge-main 或制造小 PR。
2. 原退休/导入/实际命令依赖 guard 先 RED；只加准确路径/包/声明/字段/Prompt 资源禁入，不造新扫描框架。原包内测试迁移先于删除其共享 fixture。
3. 按附录一次完成整退与 mixed。保留定义/字段/tag、Kernel/Store/worker/当前官方与 workflow 用原源文件/AST 差异核对；测试修复先分类，IMPLEMENTATION_TEST 在同 PR 收敛，不以 fixture 细化重开全局架构。
4. 运行受影响原 publishing/common/shein、SHEIN 当前规则/draft/validator/workspace、listing/submission/Kernel 与原 persistence、content/submitprep/generationtopics/词库/Store/独立 worker 适用回归；Prompt 通用与管理测试使用原 stub/临时 fixture。只做风险相关必要开发自检，不人工重复完整 CI。
5. 当前 current-application、shein-login-worker、product-listing-api-schema-migrate 命令构建及原实际 cmd 依赖边界；只 build，不运行 main。稳定阶段原完整必需 CI 与一次集中独立最终复核绑定准确实现 HEAD；必要修复只复核增量。
6. 进入 In Review，分别报告设计、源码/PR CI、实际 merge/matching main CI、运行/provider/用户验收。实现者不签发产品验收，父 #29/#301 不自动关闭。

### 6.1 原测试、清单、文档与守卫的准确处置

附录 A 的 103 专属测试整退，附录 D 的 **14 个 Publishing mixed 测试与四个 Prompt fixture 测试**按以下边界收敛；其余当前 tests 不改。没有当前包外测试引用本批退休声明/字段；现有 canonical Kernel/集成 PG/平台回执/当前 HTTP/POD 测试保持。

| 原检查/测试 | 允许调整 | 必须保留 |
| --- | --- | --- |
| assembler_test.go | 只退 `TestAssemblerBuildReusesPublishedResolutionCacheOnSecondBuild` 与旧 resolver scope helper 测试；其它涉及旧具体 resolver 的 fixture 改用既有当前纯值接口/固定 Resolution，不重建 factory/cache | 其余 12 个当前组装断言：分组、属性值应用、提示文本、来源事实、价格、review 不触发额外类目等；默认路径及结果不变 |
| size_attribute_test.go | 旧 SizeHeader concrete resolver fixture 改当前接口的固定 header/attribute DTO；中立 `containsReviewNote` 从退出的 resolver test 移入本文件 | 六个现有尺寸/模板/缺省/可选/无 size 行为断言；不把旧接口请求重建为新 mock 框架 |
| listing_copy_test.go、generation_topic_policy_integration_test.go | 退出配置转发 wrapper 后直接调用原 submitprep setter，按原方式 restore；不加生产 facade | 全部 14 copy 与两项 topic persistence→prompt/sanitizer 测试；跨 tenant、词库/覆盖/授权品牌及未授权拒绝保持 |
| submit_prep_test.go | 旧内容翻译/优化/快照/错误重试入口片段退休；保留当前 title extraction 与清理断言，词库/品牌/自由文本断言转接 `SanitizeDraftPayloadSensitiveContent`；仅保留仍需要的 fixture | 租户话题正文、无 tenant 不加策略、现用敏感词/授权品牌/词库 overlay 和字段清理断言，不保留自动发现词写回/旧 retry 协议 |
| semantic_fields_test.go、package_persistence_clone_test.go | 只退 Submission fixture/镜像与两个旧 JSON key；新增准确旧字段 unknown 拒绝可放原 strict fixture | Draft/Preview/Final 当前值/深拷贝/别名反冲突、private-cache 显式 schema、输入界限；persisted_input_test.go 原断言完整保持 |
| model_contract_test.go | 只退 EnsureDraftPayload 专属测试；其余五项保留 | 当前 draft canonical/summary 的来源顺序与结构缺陷拒绝 |
| sale_attribute_value_matcher_test.go | 旧 resolver/matcher facade 专属 11 项退休 | 两项 `ApplySaleAttributeResolution` 当前序列化/无虚构 placeholder 断言；canonical platform matcher/policy 原测试保持 |
| submit_image_policy_test.go、submit_payload_policy_test.go | 退出旧计数/上传/repair/facade 专属测试 | CloneProductForSubmit 深拷贝、当前 supplier code、prepared SKU/尺寸模板拒绝等原断言 |
| submit_payload_validation_prepare_test.go | 不再以退休 PrepareProductForSubmit 为 reference，直接验证现用 PrepareProductForValidation 的预期 | 不生成身份、保留给定身份、规范化/transport 清理的确定性规则 |
| submit_site_sku_policy_test.go | 重量边界 fixture 明确断言原 canonical policy 的 0.01g 下限（原常量未导出，不新增生产 export），去旧私有 alias | 全部三项 SKU/stock/site/weight 原有效断言 |
| variant_image_coverage_test.go | 退休旧执行 enforcement 两项；metadata fixture 直接使用原 canonical policy key/值 | 当前 VariantImageCoverageStatus 的消息/阻塞读取及原 canonical policy 图像规则测试 |
| Prompt 的四个原测试文件 | known-key 正例 fixture 换现有未退休 key；保留 literal generic store/renderer 测试；原 service stub 增加 18 key 的 schema/Upsert 未触 Store 拒绝 | tenant 隔离/权限、其它 key、loader/render、列表/status/已有行通用语义、错误码和 Store 写入前拒绝 |
| tests/listingkit_backend_retirement_test.go | 原精确缺席列表加入附录，原 scoped AST 声明/字段和实际 cmd 依赖守卫覆盖退出根/字段/18 key；旧 retained-file 项仅退出本矩阵文件 | 现有 task/RPC/各平台/身份/采集/Store/worker/当前 schema/安全断言，不禁其它包同名 selector/local/field |
| tests/import_boundaries_test.go | Publishing managed/旧 shein category/aicache 的旧目录或原文件定位改保留父目录 loader + 准确子树/absence；精确退出旧文件 allowlist/布局断言，保留 current Publishing/worker/平台边界 | 读取/解析错误仍失败，不全局忽略 ENOENT、skip、留 stub 或删除当前域守卫；原非 API/管理/OpenAI/submitprep 依赖规则保持 |
| phase2-runtime-inventory.md | **只删附录 E 的六条 CURRENT consumer 行**（logger 两条、OpenAI 四条）；上方可加本批窄说明 | 历史行/计数、closure ceilings/dated evidence、其它当前消费者逐字保留；不把表当实际运行证据 |
| .golangci.yml、go.mod/go.sum、两套及全部 workflow/保护 | **零改**；旧 file-scope 历史条目不形成新 consumer 许可，新增准确 source absence 继续阻止复活 | 原所有 deny/import/权限/安全/PG/并发/发布/分类/Required Gate/并发取消规则/ceilings |
| 本文/索引、Legacy register、module mapping、code-health decisions、external-client/project-boundaries、#817/#824 直接冲突描述 | 实施时只对本矩阵对象加窄 source-retired/superseded 注记与当前规则说明 | 历史审批/计数/取证不重写；设计阶段只交本文和索引 |

共享测试 fixture 的审计结果：`newResolutionCacheTestStore` 与三个 counting cache resolver 仅服务被退休的缓存测试，不迁移；`containsReviewNote` 是唯一需要保留到当前尺寸测试的中立 helper。fixture 细化可在上述 mixed 测试范围内收敛，不能据此扩张生产边界或恢复旧类。

最终检查的文档一致性校准：`internal/publishing/shein/doc.go` 只更新包注释，删除已退出的 compatibility/远端提交恢复职责描述，准确指向当前草稿/值/codec 与原 API/Submission owner。此注释增量不改变生产声明、字段、合同或行为；同根因核对的 `internal/listing/submission/doc.go` 仍准确描述当前 Kernel 协调，原样保留。

## 7. 确认、准入与停止条件

1. 设计阶段仅文档/只读调查；用户已确认冻结设计，原独立检查完成、Issue 已记录 IMPLEMENTATION_READY / Ready 后实施。该准入不构成合并/运行/数据授权。
2. 用户确认范围并获准后，由原唯一 Writer 在同一 PR 先把 #29/#301 当前 Design Basis/Ready 记录读回，再执行原 guard RED→完整源码/有效测试 GREEN。小提交是实现/回滚边界，不另开机械小 PR。
3. 原设计审查最多两轮正常检查。finding 先分类并指出当前 Must/影响/阻塞层级；非 Blocker 走实现测试或 Backlog，不无限补架构版本。
4. 若发现新的当前生产消费者、当前外部可观察兼容义务，或需要改变当前权限/事实/schema/事务/状态/预算/副作用/UNKNOWN owner，停止相应部分并报告具体证据；不得自行新增迁移/fallback、扩大生产文件矩阵或降低原准入。
5. 仅同根因的既有测试定位/fixture 与本 §6.1 指定的 live 行校准在实施内收敛；不得为 CI 绿色改 workflow/保护/ceilings 或把执行失败记 PASS。
6. 合并仍需最终准确 HEAD/expected-head 单独授权；部署、关单、实例或真实数据/Profile/凭据/平台/模型/Prompt 覆盖操作均不在本设计授权内。#29/#301保持 OPEN，运行/用户/生产验收 NOT_RUN。

<!-- Exact manifests follow. These are design data, not executable deletion commands. -->

## 附录 A：整文件退休清单

以下路径均相对仓库根，固定于上述调查基线。三个集合互不重复；只作为源码边界，不能生成运行环境或数据删除操作。

### A.1 生产 Go（152）

```text
internal/listing/submission/action_record.go
internal/listing/submission/attempt_finalize.go
internal/listing/submission/attempt_record.go
internal/listing/submission/confirm_remote_state.go
internal/listing/submission/direct_submit_service.go
internal/listing/submission/enqueue_retry.go
internal/listing/submission/event_draft.go
internal/listing/submission/event_history.go
internal/listing/submission/event_outcome.go
internal/listing/submission/failure_persistence_service.go
internal/listing/submission/failure_state.go
internal/listing/submission/inflight_state.go
internal/listing/submission/lease_acquire_service.go
internal/listing/submission/payload_stage_service.go
internal/listing/submission/phase_detail.go
internal/listing/submission/phase_event.go
internal/listing/submission/projection.go
internal/listing/submission/readiness_gate.go
internal/listing/submission/readiness_projection.go
internal/listing/submission/record_draft.go
internal/listing/submission/recovered_submit_persistence.go
internal/listing/submission/recovery_batch_service.go
internal/listing/submission/recovery_lease.go
internal/listing/submission/recovery_now_service.go
internal/listing/submission/recovery_route_service.go
internal/listing/submission/refresh_guard.go
internal/listing/submission/refresh_request.go
internal/listing/submission/refresh_selection.go
internal/listing/submission/refresh_service.go
internal/listing/submission/remote_completion_state.go
internal/listing/submission/remote_record_id.go
internal/listing/submission/remote_refresh_service.go
internal/listing/submission/remote_refresh_state.go
internal/listing/submission/remote_submit_service.go
internal/listing/submission/remote_sync.go
internal/listing/submission/requeue_service.go
internal/listing/submission/response_outcome.go
internal/listing/submission/result_persistence_service.go
internal/listing/submission/result_state.go
internal/listing/submission/retryable_backfill_service.go
internal/listing/submission/retryable_block.go
internal/listing/submission/source_facts.go
internal/listing/submission/start_failure_service.go
internal/listing/submission/submit_action.go
internal/listing/submission/submit_attempt.go
internal/listing/submission/submit_attempt_plan.go
internal/listing/submission/submit_error.go
internal/listing/submission/submit_lock.go
internal/listing/submission/submit_request_id.go
internal/listing/submission/submit_target.go
internal/listing/submission/success_persistence_service.go
internal/marketplace/shein/publishing/phase_detail_policy.go
internal/marketplace/shein/publishing/projection_policy.go
internal/marketplace/shein/publishing/remote_record_policy.go
internal/marketplace/shein/publishing/sensitive_retry_policy.go
internal/publishing/shein/api_builders.go
internal/publishing/shein/attribute_cache_freshness.go
internal/publishing/shein/attribute_resolver.go
internal/publishing/shein/attribute_value_matcher.go
internal/publishing/shein/cache_identity.go
internal/publishing/shein/category_ai_contract.go
internal/publishing/shein/category_legacy_selector_adapter.go
internal/publishing/shein/category_path_helpers.go
internal/publishing/shein/category_query.go
internal/publishing/shein/category_reselection_signals.go
internal/publishing/shein/category_resolver.go
internal/publishing/shein/category_resolver_ai.go
internal/publishing/shein/category_review_signals.go
internal/publishing/shein/category_semantic_verifier.go
internal/publishing/shein/category_suggest_fallback.go
internal/publishing/shein/category_tree_fallback.go
internal/publishing/shein/derived_refresh.go
internal/publishing/shein/display_attribute_batch_inference.go
internal/publishing/shein/display_attribute_candidate_set.go
internal/publishing/shein/display_attribute_dependency.go
internal/publishing/shein/display_attribute_evidence.go
internal/publishing/shein/display_attribute_field_selector.go
internal/publishing/shein/display_attribute_inputs.go
internal/publishing/shein/display_attribute_missing_inference.go
internal/publishing/shein/display_attribute_prompt.go
internal/publishing/shein/display_attribute_required_repair.go
internal/publishing/shein/display_attribute_resolution_flow.go
internal/publishing/shein/display_attribute_static_matcher.go
internal/publishing/shein/display_attribute_template.go
internal/publishing/shein/final_draft_images.go
internal/publishing/shein/final_draft_submit.go
internal/publishing/shein/generation_topics.go
internal/publishing/shein/pricing_cache_identity.go
internal/publishing/shein/pricing_cache_reconcile.go
internal/publishing/shein/resolver_cache.go
internal/publishing/shein/resolver_cache_codec.go
internal/publishing/shein/resolver_cache_key.go
internal/publishing/shein/review_content.go
internal/publishing/shein/runtime_api_factory.go
internal/publishing/shein/runtime_attribute_resolver.go
internal/publishing/shein/runtime_base_client.go
internal/publishing/shein/runtime_category_resolver.go
internal/publishing/shein/runtime_image_api.go
internal/publishing/shein/runtime_product_api.go
internal/publishing/shein/runtime_sale_attribute_resolver.go
internal/publishing/shein/runtime_translate_api.go
internal/publishing/shein/sale_attribute_ai.go
internal/publishing/shein/sale_attribute_cache_reconcile.go
internal/publishing/shein/sale_attribute_candidates.go
internal/publishing/shein/sale_attribute_mapping_llm.go
internal/publishing/shein/sale_attribute_prompt.go
internal/publishing/shein/sale_attribute_resolution_result.go
internal/publishing/shein/sale_attribute_resolver.go
internal/publishing/shein/sale_attribute_selection_rules.go
internal/publishing/shein/sale_attribute_template_rules.go
internal/publishing/shein/sale_attribute_value_helpers.go
internal/publishing/shein/sale_attribute_value_llm.go
internal/publishing/shein/sale_attribute_value_resolution.go
internal/publishing/shein/sale_attribute_value_segments.go
internal/publishing/shein/sds_images.go
internal/publishing/shein/size_attribute_cache_reconcile.go
internal/publishing/shein/size_attribute_header_resolver.go
internal/publishing/shein/source_dimension_heuristics.go
internal/publishing/shein/source_dimension_selection.go
internal/publishing/shein/studio_ai_images.go
internal/publishing/shein/submission.go
internal/publishing/shein/submission_persistence_input.go
internal/publishing/shein/submission_projection.go
internal/publishing/shein/submission_remote.go
internal/publishing/shein/submission_state.go
internal/publishing/shein/submit_image_upload.go
internal/publishing/shein/submit_image_upload_cache.go
internal/publishing/shein/submit_localization.go
internal/publishing/shein/submit_prep.go
internal/publishing/shein/submit_readiness_action_policy.go
internal/publishing/shein/submit_remote_action.go
internal/publishing/shein/submit_sensitive_retry.go
internal/publishing/shein/submit_validation.go
internal/publishing/shein/text_normalization.go
internal/publishing/shein/variant_image_sets.go
internal/publishing/sheinmanaged/api_builders.go
internal/publishing/sheinmanaged/api_factory.go
internal/publishing/sheinmanaged/attribute_api_factory.go
internal/publishing/sheinmanaged/attribute_resolver.go
internal/publishing/sheinmanaged/bridge.go
internal/publishing/sheinmanaged/category_api_factory.go
internal/publishing/sheinmanaged/category_resolver.go
internal/publishing/sheinmanaged/category_selector_adapter.go
internal/publishing/sheinmanaged/sale_attribute_resolver.go
internal/shein/aicache/cache.go
internal/shein/category/manager.go
internal/shein/content/optimizer.go
internal/shein/content/string_sanitizer.go
internal/shein/content/text_cleaner.go
internal/shein/submitprep/language.go
internal/shein/submitprep/localized_content.go
internal/shein/submitprep/localized_list.go
```

### A.2 专属测试 Go（103）

```text
internal/listing/submission/action_record_test.go
internal/listing/submission/attempt_finalize_test.go
internal/listing/submission/attempt_record_test.go
internal/listing/submission/confirm_remote_state_test.go
internal/listing/submission/direct_submit_service_test.go
internal/listing/submission/enqueue_retry_test.go
internal/listing/submission/event_draft_test.go
internal/listing/submission/event_history_test.go
internal/listing/submission/event_outcome_test.go
internal/listing/submission/failure_persistence_service_test.go
internal/listing/submission/failure_state_test.go
internal/listing/submission/inflight_state_test.go
internal/listing/submission/lease_acquire_service_test.go
internal/listing/submission/payload_stage_service_test.go
internal/listing/submission/phase_detail_test.go
internal/listing/submission/phase_event_test.go
internal/listing/submission/projection_test.go
internal/listing/submission/readiness_gate_test.go
internal/listing/submission/readiness_projection_test.go
internal/listing/submission/record_draft_test.go
internal/listing/submission/recovered_submit_persistence_test.go
internal/listing/submission/recovery_batch_service_test.go
internal/listing/submission/recovery_lease_test.go
internal/listing/submission/recovery_now_service_test.go
internal/listing/submission/recovery_route_service_test.go
internal/listing/submission/refresh_guard_test.go
internal/listing/submission/refresh_request_test.go
internal/listing/submission/refresh_selection_test.go
internal/listing/submission/refresh_service_test.go
internal/listing/submission/remote_completion_state_test.go
internal/listing/submission/remote_record_id_test.go
internal/listing/submission/remote_refresh_service_test.go
internal/listing/submission/remote_refresh_state_test.go
internal/listing/submission/remote_submit_service_test.go
internal/listing/submission/remote_sync_test.go
internal/listing/submission/requeue_service_test.go
internal/listing/submission/response_outcome_test.go
internal/listing/submission/result_persistence_service_test.go
internal/listing/submission/result_state_test.go
internal/listing/submission/retryable_backfill_service_test.go
internal/listing/submission/retryable_block_test.go
internal/listing/submission/source_facts_test.go
internal/listing/submission/start_failure_service_test.go
internal/listing/submission/submit_action_test.go
internal/listing/submission/submit_attempt_plan_test.go
internal/listing/submission/submit_attempt_test.go
internal/listing/submission/submit_error_test.go
internal/listing/submission/submit_lock_test.go
internal/listing/submission/submit_request_id_test.go
internal/listing/submission/submit_target_test.go
internal/listing/submission/success_persistence_service_test.go
internal/marketplace/shein/publishing/phase_detail_policy_test.go
internal/marketplace/shein/publishing/projection_policy_test.go
internal/marketplace/shein/publishing/remote_record_policy_test.go
internal/marketplace/shein/publishing/sensitive_retry_policy_test.go
internal/publishing/shein/attribute_resolver_test.go
internal/publishing/shein/attribute_value_matcher_test.go
internal/publishing/shein/category_resolver_test.go
internal/publishing/shein/category_review_signals_test.go
internal/publishing/shein/category_semantic_verifier_test.go
internal/publishing/shein/derived_refresh_test.go
internal/publishing/shein/display_attribute_candidate_set_test.go
internal/publishing/shein/display_attribute_evidence_test.go
internal/publishing/shein/display_attribute_required_repair_test.go
internal/publishing/shein/display_attribute_resolution_flow_test.go
internal/publishing/shein/display_attribute_static_matcher_test.go
internal/publishing/shein/display_attribute_template_test.go
internal/publishing/shein/final_draft_images_test.go
internal/publishing/shein/final_draft_submit_test.go
internal/publishing/shein/generation_topics_test.go
internal/publishing/shein/pricing_cache_identity_test.go
internal/publishing/shein/pricing_cache_reconcile_test.go
internal/publishing/shein/resolution_cache_store_test.go
internal/publishing/shein/resolver_cache_test.go
internal/publishing/shein/review_content_test.go
internal/publishing/shein/runtime_resolver_fail_closed_test.go
internal/publishing/shein/sale_attribute_cache_reconcile_test.go
internal/publishing/shein/sale_attribute_resolver_test.go
internal/publishing/shein/sale_attribute_value_resolution_test.go
internal/publishing/shein/sds_images_test.go
internal/publishing/shein/size_attribute_cache_reconcile_test.go
internal/publishing/shein/size_attribute_header_resolver_test.go
internal/publishing/shein/source_dimension_selection_test.go
internal/publishing/shein/studio_ai_images_test.go
internal/publishing/shein/submission_persistence_input_test.go
internal/publishing/shein/submission_remote_test.go
internal/publishing/shein/submission_state_test.go
internal/publishing/shein/submission_test.go
internal/publishing/shein/submit_image_upload_cache_test.go
internal/publishing/shein/submit_image_upload_test.go
internal/publishing/shein/submit_payload_prepare_test.go
internal/publishing/shein/submit_remote_action_test.go
internal/publishing/shein/submit_sensitive_retry_test.go
internal/publishing/shein/template_matcher_test.go
internal/publishing/shein/text_normalization_test.go
internal/publishing/shein/variant_image_sets_test.go
internal/publishing/sheinmanaged/bridge_test.go
internal/shein/category/manager_test.go
internal/shein/content/string_sanitizer_test.go
internal/shein/content/text_cleaner_test.go
internal/shein/submitprep/language_test.go
internal/shein/submitprep/localized_content_test.go
internal/shein/submitprep/localized_list_test.go
```

### A.3 专属 Prompt 默认 YAML（4）

```text
prompts/shein/category_selector.yaml
prompts/shein/content_optimizer.yaml
prompts/shein/display_attribute.yaml
prompts/shein/sale_attribute.yaml
```

## 附录 B：mixed 生产声明与字段

下表是 27 个 core mixed 原文件内允许删除的准确声明。方法以 receiver 限定；只删除整条列出的声明（包括其原专属注释），不得按同名全仓删除。字段和函数内部变更只按紧接的明确说明执行。三个 Prompt metadata 文件另由附录 C 限定。

| 原文件 | 允许删除的声明 |
| --- | --- |
| `internal/publishing/shein/ai_contracts.go` | `MultimodalTextGenerator` |
| `internal/publishing/shein/apply_resolution.go` | `RepairSubmitSaleAttributes`、`submitSaleAttributesNeedRepair` |
| `internal/publishing/shein/display_attribute_composition.go` | `compositionAttributeNotes` |
| `internal/publishing/shein/display_attribute_numeric.go` | `containsNumericToken`、`normalizeNumericDisplayAttributeValue`、`numericAttributeNotes` |
| `internal/publishing/shein/interfaces.go` | `AttributeAPI`、`CategoryAPI` |
| `internal/publishing/shein/listing_copy.go` | `normalizeEnglishMaterial`、`normalizeSizeText`、`synthesizeEnglishTitle`、`uniqueNonEmpty` |
| `internal/publishing/shein/model.go` | `InspectionAction`、`InspectionAttributePayload`、`InspectionCategoryPayload`、`InspectionSKCPatchPayload`、`InspectionSKUPatchPayload`、`InspectionSaleAttributePayload`、`InspectionSection`、`Inspection` |
| `internal/publishing/shein/model_contract.go` | `EnsureDraftPayload` |
| `internal/publishing/shein/preview_adapter.go` | `ProductAttributesReadyForSubmit` |
| `internal/publishing/shein/resolution_cache_store.go` | `GormResolutionCacheStore.DeleteResolutionCacheByShortKey`、`GormResolutionCacheStore.DeleteResolutionCache`、`GormResolutionCacheStore.GetManualResolutionCacheByProductIdentity`、`GormResolutionCacheStore.GetManualResolutionCacheBySourceIdentity`、`GormResolutionCacheStore.GetResolutionCache`、`GormResolutionCacheStore.SaveResolutionCache`、`GormResolutionCacheStore`、`NewGormResolutionCacheStore`、`ResolutionCacheHitSourceMemoryCache`、`ResolutionCacheHitSourcePersistentHistoryCache`、`ResolutionCacheHitSourcePersistentManualCache`、`ResolutionCacheHitSourcePublishRemembered`、`ResolutionCacheKindAttribute`、`ResolutionCacheKindCategory`、`ResolutionCacheKindPricing`、`ResolutionCacheKindSaleAttributeCustomDenied`、`ResolutionCacheKindSaleAttribute`、`ResolutionCacheKindSizeAttribute`、`ResolutionCacheShortKeyDeleter`、`ResolutionCacheSourceIdentityGetter`、`ResolutionCacheStore`、`SheinResolutionCacheEntry` |
| `internal/publishing/shein/resolver_cache_clone.go` | `CloneResolutionCacheInfo`、`cloneAttributeResolutionWithCacheNote`、`cloneAttributeResolution`、`cloneCategoryResolutionWithCacheNote`、`cloneCategoryResolution`、`cloneCategorySuggestion`、`clonePendingAttributeCandidates`、`cloneResolutionCacheInfo`、`cloneSaleAttributeCandidateInfos`、`cloneSaleAttributeResolutionWithCacheNote`、`cloneSaleAttributeResolution`、`cloneSaleAttributeTemplateOptions`、`cloneSourceVariantDimensions` |
| `internal/publishing/shein/sale_attribute_assignments.go` | `buildSKCAssignments`、`buildSKUAssignments` |
| `internal/publishing/shein/sale_attribute_custom_values.go` | `buildCustomAttributeValueNameMultis`、`customSaleAttributePermissionDeniedCacheKey`、`customSaleAttributePermissionDeniedCache`、`customSaleAttributePermissionDeniedKey`、`isCustomSaleAttributePermissionDeniedError`、`isCustomSaleAttributePermissionDenied`、`rememberCustomSaleAttributePermissionDenied`、`resetCustomSaleAttributePermissionDeniedCache`、`resolveCustomSaleAttributeValues` |
| `internal/publishing/shein/sale_attribute_readiness.go` | `NormalizeSaleDimension` |
| `internal/publishing/shein/sale_attribute_value_matcher.go` | `ResolveSingleManualSaleAttributeValue`、`ResolveSingleSaleAttributeValue`、`buildResolvedSaleAttribute`、`buildValueAssignmentsPreservingSourceValues`、`buildValueAssignmentsWithDeniedStore`、`buildValueAssignmentsWithPreparationPolicy`、`buildValueAssignments`、`comparableAttributeValueForms`、`comparableOptionScore`、`compareComparableForm`、`isLikelySaleAttributeCodePrefix`、`matchSaleAttributeValueByComparableScore`、`matchSaleAttributeValueDeterministic`、`matchSaleAttributeValueExact`、`matchSaleAttributeValueNormalized`、`normalizeSaleAttributeValue`、`resolveSingleSaleAttributeValue`、`saleAttributeLeadingScalePattern`、`saleAttributeNoisePattern`、`saleAttributeParenContentPattern`、`stripComparableSizeAnnotations`、`trimSaleAttributeCodePrefix` |
| `internal/publishing/shein/semantic_fields.go` | 无整条声明删除；只做下述 Submission 分支退出 |
| `internal/publishing/shein/size_attribute.go` | `applyProductSizeAttributes` |
| `internal/publishing/shein/source_dimensions.go` | `buildSourceVariantDimensions`、`collectVariantAttributeKeys`、`saleAttributeSourceDimensions`、`uniqueDimensionValues` |
| `internal/publishing/shein/submit_image_policy.go` | `CloneImageUploadCache`、`ImageInfoPendingUploadCount`、`ImageInfoURLCount`、`IsSDSImageURL`、`IsUploadedImageURL`、`ProductImageURLCount`、`ProductPendingImageUploadCount` |
| `internal/publishing/shein/submit_payload_policy.go` | `ValidateProductPublishPayload` |
| `internal/publishing/shein/submit_payload_prepare.go` | `PrepareProductForNewSubmit`、`PrepareProductForSubmit` |
| `internal/publishing/shein/submit_prep_sensitive_adapter.go` | `CleanSubmitProductSensitiveWords`、`ConfigureSubmitPrepRepositories`、`RetrySensitiveWordCleanup`、`SetSensitiveWordRepository` |
| `internal/publishing/shein/submit_site_sku_policy.go` | `maxSubmitWeightGrams`、`minSubmitWeightGrams` |
| `internal/publishing/shein/template_matcher.go` | `collectAttributeNames`、`isTemplateImportant`、`isTemplateRequired`、`newTemplateIndex`、`templateIndex.FindAttribute`、`templateIndex.Match`、`templateIndex` |
| `internal/publishing/shein/variant_grouping.go` | `resolveSKCName` |
| `internal/publishing/shein/variant_image_coverage.go` | `DistinctSKCMainImageCount`、`EnforceVariantImageCoverage`、`SKCMainImageURL`、`SetVariantImageCoverageMetadata`、`VariantImageCoverageInput`、`VariantImageCoverageMessageKey`、`VariantImageCoverageStatusKey`、`VariantImageGroupCount`、`VariantImageGroupKey` |
| `internal/shein/submitprep/sensitive_words.go` | `CleanSensitiveWordsWithContext`、`CleanSensitiveWords`、`RetrySensitiveWordCleanupWithContext`、`RetrySensitiveWordCleanup`、`applyAuthorizedBrandFromRuntime`、`autoDiscoveredSensitiveWordRemark`、`detectSensitiveWordLanguage`、`mergeSensitiveWordRemark`、`mergeSensitiveWordTags`、`persistValidationSensitiveWords`、`sensitiveWordRecordKey`、`splitCSV` |

- `internal/publishing/shein/model.go` 字段：`BuildRequest.Text`、`BuildRequest.SheinStoreID`；`Package.Inspection`、`Package.Submission`、`Package.SubmissionState`、`Package.SubmissionEvents`。其余字段及 tag 不改。
- `internal/publishing/shein/semantic_fields.go`：只删 `NormalizePackageSemanticFields` 内 SubmissionState/Submission 两字段镜像分支及 `conflictingPersistedAliases` 对应 pair；当前 Draft/Preview/Final 镜像及冲突检查保持。
- `ResolutionCacheInfo`、当前敏感词构造/服务及 §5.2 明确保留的接口、配置和整文件保持；表内声明退出后的机械 unused import 清理只发生于原 mixed 文件。

## 附录 C：18 个 Prompt key 与目录项

每项只退原常量、对应 catalog/manifest/变量 block 和附录 A 四份默认文件中的静态资源；shared glossary 和其它 key 不改。执行 caller 均属于附录 A 或 B。

| 原常量 | 原 key 字面值 | 原执行 caller |
| --- | --- | --- |
| `KSheinCategorySelectorExtractCoreItemSystem` | `shein.category_selector.extract_core_item_system` | `internal/shein/category/manager.go` |
| `KSheinCategorySelectorSelectCategorySystem` | `shein.category_selector.select_category_system` | `internal/shein/category/manager.go` |
| `KSheinCategorySelectorSelectCategoryUser` | `shein.category_selector.select_category_user` | `internal/shein/category/manager.go` |
| `KSheinCategorySelectorSemanticValidation` | `shein.category_selector.semantic_validation` | `internal/publishing/shein/category_semantic_verifier.go` |
| `KSheinContentOptimizerOptimizeTitleDescriptionSystem` | `shein.content_optimizer.optimize_title_description_system` | `internal/shein/content/optimizer.go` |
| `KSheinContentOptimizerOptimizeTitleDescriptionUser` | `shein.content_optimizer.optimize_title_description_user` | `internal/shein/content/optimizer.go` |
| `KSheinDisplayAttributeBatchInference` | `shein.display_attribute.batch_inference` | `internal/publishing/shein/display_attribute_batch_inference.go` |
| `KSheinDisplayAttributeFieldSelection` | `shein.display_attribute.field_selection` | `internal/publishing/shein/display_attribute_field_selector.go` |
| `KSheinDisplayAttributeFieldSelectionBatch` | `shein.display_attribute.field_selection_batch` | `internal/publishing/shein/display_attribute_field_selector.go` |
| `KSheinDisplayAttributeMissingText` | `shein.display_attribute.missing_text` | `internal/publishing/shein/display_attribute_missing_inference.go` |
| `KSheinDisplayAttributeMissingValue` | `shein.display_attribute.missing_value` | `internal/publishing/shein/display_attribute_missing_inference.go` |
| `KSheinDisplayAttributeRequiredRepair` | `shein.display_attribute.required_repair` | `internal/publishing/shein/display_attribute_required_repair.go` |
| `KSheinDisplayAttributeValueMapping` | `shein.display_attribute.value_mapping` | `internal/publishing/shein/attribute_value_matcher.go` |
| `KSheinDisplayAttributeValueMappingBatch` | `shein.display_attribute.value_mapping_batch` | `internal/publishing/shein/attribute_value_matcher.go` |
| `KSheinSaleAttributeMapping` | `shein.sale_attribute.mapping` | `internal/publishing/shein/sale_attribute_ai.go` |
| `KSheinSaleAttributePromptValueExtraction` | `shein.sale_attribute.prompt_value_extraction` | `internal/publishing/shein/sale_attribute_value_resolution.go` |
| `KSheinSaleAttributeSourceDimension` | `shein.sale_attribute.source_dimension` | `internal/publishing/shein/source_dimension_selection.go` |
| `KSheinSaleAttributeValueBatchMapping` | `shein.sale_attribute.value_batch_mapping` | `internal/publishing/shein/sale_attribute_value_llm.go` |

准确 metadata 文件：`internal/prompt/keys.go`、`internal/promptmgmt/catalog.go`、`internal/promptmgmt/catalog_manifest.go`。没有其它生产 Prompt 管理行为修改。

## 附录 D：保留断言的 mixed 测试

D.1 Publishing（14），逐项处置见 §6.1：

```text
internal/publishing/shein/assembler_test.go
internal/publishing/shein/generation_topic_policy_integration_test.go
internal/publishing/shein/listing_copy_test.go
internal/publishing/shein/model_contract_test.go
internal/publishing/shein/package_persistence_clone_test.go
internal/publishing/shein/sale_attribute_value_matcher_test.go
internal/publishing/shein/semantic_fields_test.go
internal/publishing/shein/size_attribute_test.go
internal/publishing/shein/submit_image_policy_test.go
internal/publishing/shein/submit_payload_policy_test.go
internal/publishing/shein/submit_payload_validation_prepare_test.go
internal/publishing/shein/submit_prep_test.go
internal/publishing/shein/submit_site_sku_policy_test.go
internal/publishing/shein/variant_image_coverage_test.go
```

D.2 Prompt fixture（4）：

```text
internal/prompt/registry_test.go
internal/prompt/store_test.go
internal/promptmgmt/api/handler_test.go
internal/promptmgmt/service_test.go
```

`tests/listingkit_backend_retirement_test.go` 与 `tests/import_boundaries_test.go` 只按 §6.1 扩充原准确 source/import 守卫。不得新建工具框架或改 workflow。

## 附录 E：仅退出六条 CURRENT consumer 行

位置：`docs/refactoring/phase2-runtime-inventory.md` 的 `## Legacy consumer register` 至 `## Historical pre-migration snapshot` 之间。以下行原文精确匹配后删除；其余当前行及全部历史证据/计数不动。

```text
| `task-processor/internal/core/logger` | `task-processor/internal/shein/aicache` | marketplace business consumer | marketplace owner / Phase 4 |
| `task-processor/internal/core/logger` | `task-processor/internal/shein/category` | marketplace business consumer | marketplace owner / Phase 4 |
| `task-processor/internal/integration/openai` | `task-processor/internal/publishing/sheinmanaged` | marketplace business consumer | marketplace owner / Phase 4 |
| `task-processor/internal/integration/openai` | `task-processor/internal/shein/category` | marketplace business consumer | marketplace owner / Phase 4 |
| `task-processor/internal/integration/openai` | `task-processor/internal/shein/content` | marketplace business consumer | marketplace owner / Phase 4 |
| `task-processor/internal/integration/openai` | `task-processor/internal/shein/submitprep` | marketplace business consumer | marketplace owner / Phase 4 |
```
