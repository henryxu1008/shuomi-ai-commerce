> #301 / #923 **SUPERSEDED，仅旧租户 Prompt 闭包**：用户已确认[完整退休设计 §1–§7（含 §4.2/§6.1）](legacy-tenant-prompt-closure-retirement-boundary.md)。本文涉及旧 `internal/prompt`、`internal/promptmgmt`、`shared/tenantctx`、五 YAML 及其现役 KEEP/复用建议仅在该准确矩阵内被退休决定替代；如涉及镜像打包，仅退出两条 `prompts` COPY。当前 AI/图像/采集、身份/有效规则、独立 SHEIN 登录与 Store 同步及其它 KEEP 不变。停止全新 schema 创建旧 Prompt 表，不改 Goose 登记或已有表/行，不新增替代 Store/迁移/兼容。以下原正文、历史数字/勾选/证据完整保留，不扩大实施、合并或真实资源权限。

> #301 / #915 **USER_CONFIRMED / SOURCE_IMPLEMENTED**：按[旧宿主、HTTP 与图像辅助完整退休闭包 §1–§7/§6.1](legacy-host-image-helper-closure-retirement-boundary.md)的用户确认矩阵，仅退休矩阵列出的闲置 imagex 编码/格式辅助及 httpclient 两 helper；保留当前 HTTP/TLS/retry、图像解码/metadata/生产注册及 Product/Asset owner，不恢复旧 Task 预览或下载链。 仅该准确矩阵取代本文件对应 KEEP，其余边界及以下全部历史正文保留；workflow/inventory 与既有资源不动，独立 SHEIN 登录及 Store 同步保持，#301 OPEN。

# #301 旧 Task 预览与刊登素材辅助尾部完整退休设计

> 用户已确认 [#909 productpolicy 完整退休边界](legacy-product-policy-closure-retirement-boundary.md) §1–§7（含 §6.1）。本文件对 `internal/marketplace/productpolicy` 六工件（doc/filter_rule/rule_checker/price_helper/fulfillment 及专属测试）和四项 KEEP 检查的保留含义已由该边界精确替代为 RETIRE；其余有效规则/API/采集、共享合同及独立 SHEIN 登录与 Store 同步不变。以下原正文、历史数字与验收证据保留，不恢复旧策略或授权既有实例/数据/凭据/进程操作。

Status: **USER_CONFIRMED / SOURCE_IMPLEMENTED**。用户已确认冻结设计 **7c2c7e3b62471491bd6f7f396ab7330ffa735123** §1–§7（含§6.1），[Issue准入/归档](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6340083885)及[PR准入](https://github.com/qq550723504/shuomi-ai-commerce/pull/891#issuecomment-6340084358)和两处 Ready 读回先于生产修改。设计阶段 N/A，本轮 **Design Basis: Reuse Existing Architecture / Ready**；唯一Writer已按47专属工件/15精确尾部整批实施，无新EXTRACT/替代owner。原guard先RED后GREEN，准确实现HEAD/必要自检/原CI及最终交付检查留PR，不冒充运行/用户验收。#301 OPEN，#29不重开。

调查基线：main **eb5b1e06deb529fe04a1e3bf416d8ed9f67fdb0f**，同一仓库 ID 1108239623 已由原 owner 更名为 shuomi-ai-commerce。#885 已正常合并，实际 merge 0b683911dc84b0bfe8bd82d4e2dc190b0596d08b 的 push/main CI39763961555 attempt1 完整 SUCCESS；这是前批完成证据，不是本设计或未来实现的 CI。

## 1. 用户结果、范围与权限

使用者是维护当前应用、采集、图像、Store 和独立 SHEIN worker 的维护者。交付结果是退出旧 Task 预览和刊登图片辅助实现后，仓库不再同时声明两套预览/素材 owner；当前业务仍使用现有 Product、Listing、ImageAgent 与平台 API。没有证据表明本批残留正在阻塞某个当前核心用户操作；本批是用户明确选择的旧代码退休工作，不新增产品、治理或内部试用前置。

本批一次覆盖两个已经失去消费者的旧刊登分支及完整专属尾部：Task 预览/平台 section 拼装，以及刊登素材 downloader/随机指纹扰动；连同四个未实现目录声明、唯一外部测试尾部、原 guard/lint 和直接文档一起收口。它们不构成一条仍运行的业务链，不编造新的共同运行 owner；同批完成源退和过时 owner 声明清理，不按包、文件数、测试或评审机械拆 PR。

设计阶段仅本文及两个PROPOSED索引/登记；用户确认及Issue/PR Ready读回后，同一Writer在同一PR按§5/§6.1整批实施。inventory仅获准一条CURRENT，workflow/依赖/锁/保护及其它检查不改；设计阶段零源码diff的证据保留PR。

仅检查 Git 追踪源码、文档、样例结构和依赖元数据。不运行旧 downloader/构造器、网络探测、图片处理、脚本、浏览器、应用、worker、provider、数据库或 Redis；不读取实际凭据、私密状态、配置值、日志或旧输出，不操作既有实例、数据、凭据或进程，不清理 ignored 目录/文件。#301 不关单，不增加内部试用前置；合并、部署、运行/用户验收另行授权。

## 2. Design Basis、Authority 与不变量

设计阶段 **Design Basis: N/A**；用户确认和Ready读回后，本轮 **Reuse Existing Architecture / Ready**。当前owner/产品/领域/状态/权限/持久化/外部副作用语义保持，依据：

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)和[Issue 驱动规则](../engineering/issue-driven-development.md)。旧内部 Task/Workspace 不是当前产品 authority，不包装、迁移或复活。
- [当前完整来源业务闭环](source-business-closure-v1.md)：来源输入、Catalog/Collection lineage、刷新/读取、显式资产批准与 readiness 已由当前 owner 承担，不由旧 preview.Attachment 或 Task 预览建立第二事实源。
- [独立 SHEIN 身份与持久化边界](shein-worker-numeric-persistence-retirement-v1.md)、[当前值合同](legacy-numeric-value-contract-closure-retirement-boundary.md)及[旧采集尾部边界](legacy-numeric-acquisition-tail-closure-retirement-boundary.md)：独立登录、Store 同步、UNKNOWN 不自动接管和执行 fence 保留；当前 Amazon/1688 来源及平台 API/规则不按零 importer 自动删除。

本批超过 30 文件，完整矩阵预计 62 文件，但没有新产品路径、领域合同、事实 owner、schema/迁移、状态机、事务、权限/租户、计费或 retry/UNKNOWN/recovery/平台副作用边界；当前有效行为沿用现有批准 owner，不新增 EXTRACT 代码。文件风险提示不靠拆 PR 规避。若实际消费者刷新证明必须修改这些边界，停止对应实施，报告并按需要升级 Independent Architecture，不能在实现里临时发明合同。

Must：§5 完整退出；当前能力及业务/权限/隔离/确定性合同保持；原必需和安全检查继续有效；除 §6.1 明确提议的一条 CURRENT inventory 记录外历史/计数/ceilings/accuracy 逻辑不改；全部 workflow/依赖/锁/保护不改；无真实状态操作或新试用前置。Should：复用已有 AST、decoded import、退休 guard、原测试及标准库，不造替代下载器/值容器/验收平台。Out of Scope：新预览产品、下载安全策略改造、共享 config/logger 大迁移、平台 API/规则、当前 UI/IA、其它零引用能力、运行验收或真实资源清理。Accepted Risks：无新增；具体外部人工调用和物理状态仍 UNKNOWN，未执行的验收为 NOT_RUN。UI/Figma 增量 N/A，当前页面/入口及最终 Product/Figma authority 不变。

**精确取代范围**：本文获确认后，旧 listing-preview-boundaries 的“当前 owner/兼容 facade/继续迁移”指令只在 §5 两包上被取代；phase2 inventory 的 downloader 暂留与旧 TEMU 图片退休设计的 downloader KEEP 只在这个已无消费者的包上被取代。历史正文、dated 计数和已执行证据不改。不能从本决定推导删除仍有效的 Listing canonical subject、Product、平台规则或 SHEIN 下载接口。

## 3. 实际消费者与剩余项核对

核对根模块 Windows/Linux 的 Imports/TestImports/XTestImports、Git 追踪的全部构建目标源码引用、debug 独立模块、脚本/配置/前端和当前装配入口；不以 Windows 一次 go list 或一条文本搜索推断完整闭包。设计阶段的 Linux go list 只读元数据，无编译产物执行；不把它称为 Linux 运行测试。

| contract → implementation → injection → consumer | 当前事实 | 决策 |
| --- | --- | --- |
| preview.TaskRepository → TaskPreviewService/GetTaskPreview → Build/Finalize 回调 | 无生产注入/调用者；模型仍以 TaskID、旧 Task 状态、result platform 和历史 shell 为中心。仅包内专属测试 | 整包 RETIRE，不保留 generic service 或改名为 Product preview |
| preview shell/header/status/revision/render metadata/summary/capabilities/projection | Go 生产外部 importer 为零；没有 HTTP/BFF/当前 UI 调用入口；抽离过 legacy DTO 不代表仍有当前需求 | 专属值行为与旧模型同退，不为了旧自测新建 helper |
| preview.Attachment → BuildAttachment → tests/issue30_sourcing_contract_test.go | 唯一外部测试直接把 CatalogProduct/ApprovedAssetInventory 指针装入容器，再和原值自身比较；无复制、验证、持久化、身份或资产批准行为 | 删精确测试包装，不抽新 DTO；Catalog/Asset 当前批准、持久化、lineage、版本/隔离/readiness 测试保持 |
| listing/platform 的 Normalize/SupportedPlatforms/Section/BuildRegisteredSections | 无生产或外部测试 importer；只由本包旧自测验证四平台预览 section 回调 | 整包 RETIRE；不改变现有 marketplace 平台集合/规则/平台 API |
| pkg/downloader 的 ImageDownloader/anti_bot/PlatformImageMutator | Windows/Linux 根模块无生产或外部测试 importer，debug/脚本/前端无引用；只剩自身两项 TLS 构造测试 | 整包 RETIRE；HTTP、退避/阻断内存态、随机字节/MD5 并非当前下载/图像合同 |
| 当前 SHEIN 图片 API | shein/api/image 自有 imageDownloadProvider + plainHTTPImageDownloader，使用标准库 net/http；不 import pkg/downloader | 全部 KEEP，上传/处理/注入接口不改，不改 UA 等顺带整理 |
| 当前图像与采集 | ImageAgent generation/objectstore、SRC source-media、crawler 及其风险/质量/图像规则使用现有各自 owner；当前入口无本批三包依赖 | 全部 KEEP，不用旧增强下载器替换，也不移植其随机指纹/风控重试 |
| listing/export/revision/settings/workflow 四目录 | 各只有 README，无 Go 实现/消费者，是旧 Task 迁移计划的目录声明 | 四份声明同退；不删除当前 listing/record、task/canonical_subject 等真实包 |

### 3.1 没有纳入本轮删除的零 importer / 混合项

全仓核对不能把“零默认生产 importer”当作 RETIRE 充分条件。以下均有当前 authority、测试消费者或独立入口，保留原 owner：

- Amazon API/来源 Processor、SHEIN activity/category/image/translate/generationtopics/namelimit/pricing、TEMU API 根及 marketplace/productpolicy：有效 API/纯规则，不是这批预览实现。
- 当前 integration/dub、geminiimage、openmeter、promptmgmt/api、localagent/httpapi；ImageAgent Temporal 测试辅助、Product Asset policy/recipe/test helper：本轮没有证据可以取消当前能力或合同，不顺带删除。
- listingkit/phoneonboardingpreflight：根模块零 importer 会漏掉 hack/debug/listingkit-phone-onboarding-preflight；Login V2 探针 KEEP。listingadmin 现存安全/退休测试保留。
- core/config、core/logger、pkg/appenv、httpclient、imagex、共享 browser/workerpool/DB/Redis：仍被当前应用和 worker 消费，不能为“退休彻底”递归删除。未来若继续退出混合 legacy 接线，须按实际消费者与当前 owner 另行明确完整合同；本批不提前宣称 #301 全部完成。

## 4. 持久化、配置与副作用 owner

| 边界 | 现有行为/风险 | 本批处置与当前 owner |
| --- | --- | --- |
| 旧 Task preview 读取/回调 | Repository/Build/Finalize 是 generic 函数接口，包自身不持有 DB/Redis/schema、HTTP route 或 durable 状态；无当前装配 | 只删未接线源码，不调用回调，不给当前 Product/Listing 增加恢复或映射层 |
| Catalog/Asset attachment | 只是当前值的指针包装；不执行保存或批准，JSON 字段无当前外部出口 | 只删旧容器；Catalog/Product snapshot/lineage、ApprovedAssetInventory.Scope/Assets 的持久化 owner/编码/批准边界保持 |
| 旧 downloader | HTTP GET/HEAD、重试/等待/UA/风控内存态、字节随机扰动和 hash；没有专属 durable 仓库或 Config 注入链 | 不发网络请求、不写图像文件、不启动构造器；删实现不等于处理任何已下载文件或平台成品。httpclient 原 TLS 默认与安全合同保留 |
| 当前图像下载/处理/上传 | 各当前 transport/SDK、Asset/Collection/Generation owner 承担；SHEIN API 独立下载器仍有效 | 全保留，不引入新 fallback、重试/UNKNOWN owner 或另一持久化事实源 |
| 独立 SHEIN 登录/Store | 当前 Store/SourceAccount、执行归属控制、Redis/浏览器和 fence 属已批准独立 worker | 源码/配置/部署/数据均不动；不登录、不接管 UNKNOWN、不访问 key/Profile/凭据 |
| 配置/依赖 | 三包无专属配置、env、启动命令、部署/workflow 目标或独立 manifest；downloader 只消费共享 logger/httpclient | 无配置/schema/迁移/部署删除；不 go mod tidy、不升级/删共享依赖。inventory 只提议删一个真实消费者关系 |

## 5. 删除、保留与抽取矩阵

**RETIRE：下列准确 47 工件**，其中 23 个生产 Go、19 个专属测试、5 个 README。不是授权按整个 listing 或 pkg 父目录递归清理。

| owner | 准确文件（目录下相对文件名） | 决策 |
| --- | --- | --- |
| internal/listing/preview（33） | README.md; attachment.go; boundary_guard_test.go; doc.go; errors.go; header.go; header_test.go; model.go; platform_boundary_test.go; platform_render_summary.go; platform_render_summary_test.go; platforms.go; platforms_test.go; projection.go; projection_test.go; read_model.go; read_model_test.go; render_preview_capabilities.go; render_preview_capabilities_test.go; render_preview_metadata.go; render_preview_metadata_test.go; result_projection.go; result_projection_test.go; revision_history.go; revision_history_test.go; service.go; service_test.go; shell.go; shell_test.go; status.go; status_test.go; task_read_model.go; task_read_model_test.go | 全退旧 preview 契约、编排、专属测试与声明 |
| internal/listing/platform（6） | errors.go; errors_test.go; registered_builders.go; registered_builders_test.go; sections.go; sections_test.go | 全退旧 section 框架，不留无消费者通用层 |
| internal/pkg/downloader（4） | anti_bot.go; image_downloader.go; image_downloader_test.go; image_processor.go | 全退增强下载/风控重试/随机指纹扰动及专属构造测试 |
| 旧空目录声明（4） | internal/listing/export/README.md; internal/listing/revision/README.md; internal/listing/settings/README.md; internal/listing/workflow/README.md | 撤回旧 Task 迁移占位，不新增对应目录/实现 |

**混合尾部与文档**：仅 §6.1 明确的15文件，与47专属工件合计62文件，已在同一PR整批应用；设计阶段三文档及其CI/复核证据保留PR。

**KEEP**：当前 Product Catalog/Sourcing/Collection/Asset，readiness/sourcefacts，Listing record/preparation/materials/completiondraft/submission 以及 task/canonical_subject.go 和其测试；当前平台 API/DTO/规则、AI/SKU/spec/MappingInfo、两种 SHEIN 属性、PriceManager/PriceCalculator、规范化/hash/四个 JSON 零值槽位；当前身份/计费、采集/1688/Amazon 来源、图像/风险判断/百炼参数、Store、通用 DB/Redis/队列/Retry/Temporal/workerpool/http/image/logging，以及独立 SHEIN 登录/Store 同步/UNKNOWN/fence。路径里出现 Task/preview/image 等词不足以删除真实当前消费者。

**EXTRACT：无新增代码**。当前有效行为已经由上述 owner 提供；BuildAttachment 不承载新的规则，不为两项自身比较断言创造值对象。共享 net/http、httpclient 的 TLS 和图像编码能力原位保留，不复制旧 downloader，不把退役包改造成通用下载器。

```text
Legacy decision: RETIRE（用户确认，已整批源码实施）
Reusable behavior: 当前 Catalog/Asset/Listing/图像/下载/平台规则原 owner 保留，无新 EXTRACT
Current owner: product/*、listing 当前真实包、imageagent、当前 SRC/crawler/transport、平台 API 和独立 SHEIN/Store
Cutover/deletion condition: §1–§7/§6.1 获用户确认及 Ready 读回；刷新无保留消费者；47专属工件和15精确尾部整批退出，原必需检查有效；不操作物理状态
```

## 6. 验证与交付证据

设计阶段只核对追踪源码/消费者、矩阵路径及现有文档检查；稳定设计候选运行一次原 PR CI/有界设计复核，结果绑定准确 HEAD。无源码改动不制造业务测试 RED，不执行旧包网络测试或引入验证框架。

用户确认后：在既有退休 guard 中先补三包/四占位目录的缺席与全部追踪 Go decoded import 禁入，先 RED 再整批源码删除获得 GREEN；解析失败必须报错。重用既有测试设施，不开发通用 dead-code scanner、runner 或验收平台。包内旧实现测试随退出；当前业务、安全、隔离/授权、TLS、批准和确定性测试保留。

必要开发验证：原退休/导入/架构文档检查，issue30 来源与资产批准/readiness 合成 fixture，Product Catalog/Sourcing/Asset、当前 Listing 与 SHEIN 图片 API/httpclient/独立 worker/Store 适用原测试，当前三个入口依赖元数据及必要编译（不执行产物）。测试如需资源，仅任务临时目录/原有 in-memory 或 loopback fixture；不新建 PG/Redis/browser/container，不读取既有配置或真实服务。原需外部条件的测试如实 NOT_RUN/skip。稳定候选交原完整 CI，不重复人工全量重跑。

实现者自检、CI 和 AI 复核不能签发运行/产品验收。最终交付按现有规则一次检查完整 diff、保留能力及证据，finding 先分类；后续只复核实际增量。准确源码 HEAD、原 CI、独立复核及批准留 PR；未变化证据可明确复用，不冒充新 SHA 的执行结果。合并后只有实际 merge SHA 的 push/main 完整 SUCCESS 才可报告 main CI 通过。

### 6.1 原检查、直接文档、inventory 与 workflow 精确处置

本节为**用户确认的准确实施边界，已整批应用**；设计阶段零源/测试/lint/inventory diff证据留PR。以下15文件与47专属工件在同一PR完成：

| 文件 | 同批最小处置 | 保留条件 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 复用原 AST/decoded import/追踪文件和入口元数据设施，增加47路径/三包及四占位目录缺席、生产/测试/debug import 精确禁入；保护当前三个入口无旧依赖 | 旧 guard/保留 owner 合同不删；解码/解析/读取错误 fail，不通过 missing-skip/dummy/alias 伪造退出 |
| tests/import_boundaries_test.go | TestListingPreviewImportsStayPlatformNeutralAcrossBuildTargets 保留名称，改为三包源缺席/所有追踪 Go（含测试、debug、build constraints）decoded import 禁入保护；可委托同一已有 guard helper | 不再读取已删除目录而失败或 skip；其余平台、管理、SDK、授权/租户、HTTP 等检查及 fixture 不动 |
| tests/issue30_sourcing_contract_test.go | 仅去 preview import 及 BuildAttachment 的三行自身比较；在原当前 owner 断言处直接确认持久化 SourceID/ReferenceID 和 ApprovedAssetInventory.Scope，不造替代 wrapper | Catalog 重放/冲突、组织隔离、lineage、跨版本读取、显式批准、版本/platform/product scope、readiness 和后续缺事实拒绝全部保留 |
| tests/architecture_docs_test.go | 仅校准 preview 当前 owner 行及 preview 文档专属断言为“旧预览已退休/当前 owner/原 guard 已转为禁入”；精确保留现有 guard 名称引用 | 不能用历史正文仍含词语就冒充当前 owner 正确；其它文档/边界/安全检查不改 |
| .golangci.yml | 仅将 listing_preview_platform_neutral 这一规则由扫描旧 preview 包改为项目生产 Go 对三个退休包的 import 禁入，沿用原 depguard 配置语义/名称；原市场包禁止项在该已退 owner 上由更强的源缺席/全局禁入取代 | 其它 lint 规则、测试排除、严重度/阈值/检查项不改；不得全局禁止仍有效平台 API，也不得保留失效 glob 冒充有覆盖 |
| internal/listing/README.md | 将“长期承接 Task/迁移旧 facade”的当前声明替换为真实现役包、owner/边界链接及准确退休范围 | 不说整个 Listing/Task 都退，不引入新工作流 |
| internal/listing/task/README.md | 仅校准为真实 CanonicalSubjectReader/Actor/租户与 owner 读取合同，撤回无实现的 lifecycle/workflow 宣称 | canonical_subject.go/test 全部不改；不移动授权或事实 owner |
| docs/architecture/listing-preview-boundaries.md | 顶部窄 supersession 注记：旧 owner/迁移指令失效，链接本设计与当前 Catalog/Asset/Listing；原两个 guard 名称说明现已保护退休禁入 | 历史正文逐字保留，旧 compatibility 语言不恢复权限；不以旧文档要求保活源码 |
| docs/architecture/project-boundaries.md | 顶部窄当前注记；只替换当前 owner 表的 Platform-neutral preview rules 行和相应 guard 解释 | 其它当前 owner、enforcement 名称/索引和 dated 计数不改 |
| docs/architecture/architecture-review-checklist.md | 只校准原 preview owner 那一项和相应 guard 描述为旧包源退/全局禁入，链接本设计 | 其它审核/安全项和原 enforcement 名称保持 |
| docs/architecture/legacy-temu-image-processing-retirement-boundary.md | 顶部只追加 downloader KEEP 精确取代注记；当前图像/编码/填充/上传/共享合同仍保留 | 原历史正文（含 KEEP 表和执行证据）逐字保留，不重写历史 PASS/统计 |
| docs/refactoring/phase2-runtime-inventory.md | 仅删除下面准确一条 CURRENT 消费者行 | 历史、计数、ceilings、映射、parser/accuracy/Code Health 逻辑全部不改 |
| docs/architecture/README.md | PROPOSED 索引转为用户确认/Ready，最终源级状态按实际交付标明 | 不滚动写 CI/SHA，不动其它已完成批次 |
| docs/refactoring/legacy-register.md | PROPOSED 注记转为确认的准确 RETIRE/KEEP 与本设计链接，旧 preview adapter 行中“当前 preview”的有效性仅由窄 supersession 取代 | 旧登记行/历史正文保留，不授权其它 legacy 复活或扩大删除 |
| 本文 | 只更新实际确认/Ready/源码阶段及准确范围；新增真实消费者或边界问题按 §7 停止 | 本文不是运行验收，也不替代 PR 中准确 HEAD/证据 |

用户准确确认且已删除的唯一 CURRENT 行（设计阶段零改证据保留）：

```text
| `task-processor/internal/core/logger` | `task-processor/internal/pkg/downloader` | mixed business helper | product + marketplace / Phases 3-4 |
```

所有 workflow（含两套主 workflow）、依赖/锁、配置/env/部署模板、保护规则 **零改**。测试/depguard 的修改只准确替换已退出 owner 的保护方式，不删 Required/Security/Admission/Code Health 检查，不改 inventory accuracy/ceilings，不把改 CI 当成源码正确的替代。

## 7. 实施准入、停止与关闭边界

1. 冻结设计§1–§7（含§6.1一条CURRENT与单条depguard转换）已获用户明确确认；#301/PR Reuse Existing Architecture / Ready更新及读回先于生产修改。同一Writer/同一主要PR整批实施，不机械拆批；准确源码CI/最终交付检查后再交合并授权。
2. 形成最终源码候选前核对实际 main/保留消费者，处理真实组合风险；不为“始终最新”持续同步或重复 AI 全审。准确 final HEAD 的原 CI/交付检查后再交合并授权。
3. 若发现当前入口、HTTP/前端/脚本/独立模块或具体外部可观察合同仍需要这些包，报告具体用户操作、owner、最小缺口和成本；停止该删除，不自创 compatibility、双读/双写、fallback 或替代值模型。
4. 若必须新增抽取规则、改变保留文件运行行为/权限/事实 owner/状态/恢复边界，先指出具体差异并取得设计决定；普通实现问题按 BLOCKER/IMPLEMENTATION_TEST/BACKLOG/ACCEPTED_RISK/NOT_APPLICABLE 分类，不无限重开全局架构。
5. 若原 CI 指出矩阵外孤立尾部，先核对同根因，列精确修订及保留条件；未获明确范围前不改冻结 KEEP、workflow、inventory 其它行、依赖或保护，不重跑/绕过门禁掩盖原因。
6. 只交付源级退休；真实环境状态 UNKNOWN、运行/用户验收 NOT_RUN。#301 保持 OPEN，不由本批推导全部父目标完成；#29 不重开。历史后缀及勾选逐字保留，更新前新读当前正文，不超过65536字符；不操作既有实例、数据、凭据或进程，不增加内部试用前置。
