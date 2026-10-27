> #301 / #972 当前整批实施：**RETIRE / Independent Architecture / USER_CONFIRMED / IMPLEMENTATION_READY / SOURCE_IMPLEMENTED / IN_REVIEW**。用户已确认冻结设计 `a2b39bd82fc8713c2ed54dea208afe47c5cd8248` 的 §1–§7（含 §4.2、§6.1），同一 PR 收齐102路径：旧1688 LocalAgent整链及父crawler空壳、两个专属Prepare、孤立SDS商品源与底图策略退出。有效来源夹具改现有中立合同，原完整JSON/规范化/hash/发布身份前后等价；原安全与领域断言按§6.1精确转换。保留当前采集、SDS/POD、资产审批/recipe/共享身份权限及独立SHEIN登录与Store同步；workflow/完整inventory/依赖/保护零改，不操作既有实例/数据/凭据/文件/进程，不增加试用前置，#301 OPEN。实施/自检/原CI/最终复核分别记录于 [PR #972](https://github.com/qq550723504/shuomi-ai-commerce/pull/972)，不等于运行/产品验收；合并/部署/关单另授权。以下冻结设计及历史完整保留。

# #301 旧 LocalAgent、SDS 商品源与孤立资产策略完整退休设计

> **DESIGN_ONLY / SOURCE_NOT_STARTED / AWAITING_DESIGN_REVIEW / NOT_READY**。当前仅本文、Architecture README、Legacy Register 三文档；只读复核发现的三项同根闭包遗漏已在本稿集中校准，待相关增量复核；未来同一主要 PR 的 **Independent Architecture** 实施必须先完成适用独立设计复核、用户确认 §1–§7（含 §4.2、§6.1），并在 Issue/PR 记录、读回 IMPLEMENTATION_READY / Ready。起始实际 main 为 `12f5b70fa72545b401f337f7ce4851104f8fa07d`；#969 已正常合并且其准确 push/main CI40449857063 SUCCESS。新 main 仅另含 #971 三份 Knowledge 测试/文档，与本批目标无交集；本基线 push/main CI40451781597 attempt1 整run SUCCESS。不接管其 owner，不拿前批/基线CI当本批设计证据。

## 1. 本批要交付什么

使用者是维护当前采集、来源发布、资产审批和 SHEIN 登录/Store 同步的开发者。当前用户继续从既有供应/采集与浏览器入口取得并保存商品、读取 lineage 和使用批准资产；本批不新增 UI、运行入口或试用前置。结果是旧 PoC 不再被误认作可用入口，失去消费者的状态/适配/策略源码完整退出。

| 三组一次完成 | RETIRE | 当前能力继续使用 |
| --- | --- | --- |
| 旧1688 LocalAgent 整链 | CLI → deviceauth/client → 未注册HTTP → 内存Service/Runner → 旧 crawler/a1688及父crawler空壳；测试、脚本及两处专属Prepare预热门面一并退出 | 当前匿名公共浏览器采集、Browser Capture/扩展、批量导入、原1688/Amazon浏览器与debug、当前身份与授权 |
| 旧SDS商品源/POD适配层 | product_source 的 canonical 写入、baseline、旧子任务执行/readiness、图片/值模型和专属测试 | 当前 integration/sds → app/pod/现有 workflow、供货链、批准资产与当前提交/readiness owner |
| 孤立资产底图策略 | asset/policy 三函数及唯一专属测试 | asset审批/Inventory/ImageSet、recipe及ImageAgent有效图像规则 |

未来 **102 唯一路径**：43整退工件、2混合生产文件、5原测试/检查文件、8当前文档、44历史顶注；全部在同一主要 PR，不按文件数或模块机械拆批。当前实际只改三设计文档。源码不新增生产函数、包、表、DTO owner 或兼容 alias。

Must：完整闭包及实际入口辨认；有效来源/资产回归、JSON/hash/版本/lineage/组织隔离保持；当前采集、浏览器取消/关闭、SDS/POD、共享身份/权限、独立 SHEIN 登录与 Store 同步保持。Should：复用现有 SourceEnvelope/Normalize、Viper、当前采集/POD/资产 owner，不造中间适配器。Out of Scope：新功能/AI/provider、当前状态/事务/权限/计费/retry/UNKNOWN/recovery owner、源码以外的运行退休/迁移/清理、workflow/完整inventory/依赖/保护及其它检查；不操作既有实例、数据、凭据、token、配置/输出文件或进程，不增加内部试用前置，#301保持OPEN。没有新增 Accepted Risk。

Threat Model 仅覆盖误删实际采集/授权/规则、夹具改写丢失有效断言、旧Job/接口/包回流、用测试/历史KEEP保活旧owner或放宽门禁；不扩大全局安全目标。

## 2. Design Basis、Authority 与前批 KEEP

当前纯文档任务 **Design Basis: N/A / Ready**；未来交付选择 **Independent Architecture / NOT_READY**：超过30路径，涉及旧来源公开合同、OIDC辅助、内存Job/claim与SDS子任务状态模型退出，需要一次冻结完整边界与旧KEEP替代。实际当前领域事实/授权/状态/schema/事务/恢复/副作用owner不新增、不改变，不为本批新建全局框架。无UI/IA/交互改动，Figma N/A。

Authority：[Greenfield](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[目标架构](project-target-architecture.md)、[当前项目边界](project-boundaries.md)、[来源交接](../product/product-sourcing-handoff.md)、[Issue驱动规则](../engineering/issue-driven-development.md)、当前[服务端匿名浏览器采集冻结设计](2026-09-26-1688-server-public-browser-acquisition-design.md)及用户本轮范围决定。保留当前浏览器/批量路线；`2026-09-21-1688-batch-local-agent-design.md` 指当前批量采集队列，不是本次旧PoC，不因名字相近而退休。

| 旧KEEP/Authority | 本批准确替代 | 仍有效 |
| --- | --- | --- |
| #885 SDS退休边界 | product_source十文件完整KEEP及原guard对该叶包的例外 | 当前integration/sds、POD/Submission/Asset、共享身份与独立登录/Store同步 |
| #891/#893 | localagent/httpapi、Asset policy、旧1688适配被视作当前合同的含义 | 其它独立集成/PoC如Dub/OpenMeter、实际图像/资产规则、显式日志与身份边界；不能按零import普遍取消能力 |
| #900/#901/#942 | 原旧适配/Runner作为当前1688消费者的保护，及专属preclaim Prepare接线 | 原1688/Amazon浏览器、debug、验证码算法、实际Process/取消/关闭与有效测试 |
| #947/#948/#953/#958及其它历史宽泛KEEP | 仅§5三个旧闭包、旧CLI编译/操作入口与专属尾部 | 公共配置投影/空Profile、当前来源/审核/AI/JSON、真实PoolManager、当前构造/依赖与其它永久禁令 |
| 8/20来源适配、8/21 LocalAgent PoC、8/22 hardening | 旧CLI/API/Job/OIDC/SourceSnapshot方案与复现脚本不再作为当前入口或实施要求 | 当前SourceEnvelope/发布/lineage、授权/组织隔离、显式资产批准及正常第三方协议 |
| 当前handoff/结构/checklist与历史映射 | 旧adapter owner、旧CLI目录目录项和旧专属检查引用 | 当前owner与其它目录/检查；历史批准/失败/计数/勾选不重写 |

所有历史替代只限明确三组和两处专属Prepare。§5列44旧依据未来仅加窄顶注，完整原Git blob后缀保留；不重新解释其它有效规则。当前设计阶段不改它们或声称已退休。

## 3. 实际消费者、入口与依赖方向

按基线 **3174 tracked Go** 用标准Go AST只读解析（含原测试、未选择编译源、hack/debug与negative fixture），解码import、核对别名/选择器/函数值，再查同包调用、HTTP注册、脚本、原检查和文档。未执行构造器、应用、CLI、安装、登录、采集或现有环境检查。零外部import不是自动退休理由；必须同时核对当前产品/owner和完整回路。

| 旧对象 | 实际消费者/闭包 | 判定 |
| --- | --- | --- |
| localagent根 | 包外生产只有旧CLI、其client/httpapi；无当前应用或当前public采集接线 | 内存Service/Runner/Job/claim/summary及logger尾部整退，不保留空壳 |
| localagent/client与deviceauth | 生产消费者均仅cmd/1688-local-agent；HTTP包无任何包外生产import | 与CLI、完整HTTP/DTO/路由模块一起退，不能只删HTTP留下断链CLI |
| integration/crawler/a1688 | 生产只由上述CLI/localagent闭包消费；唯一保留范围外import是tests/issue30_sourcing_contract_test.go | 整根退出；有效测试夹具改接中立合同，不移动旧工厂到新包/测试库 |
| integration/crawler父根 | 唯一旧a1688子包退出后只剩README/doc.go/两个专属边界测试和三helper；无包外生产import或helper调用 | 三工件同批整退，集中guard禁止整个旧根/decoded import，承接并强于原父guard；Catalog原负向禁令保持 |
| Alibaba1688Processor.Prepare → SingleProcessor.Prepare | 仅旧适配器/Runner通过准备接口调用，真实Process不经这两方法 | 仅退两方法及processor.go专属errors import；实际newPublicBrowserManager/CreateBrowser、Process与closeOnContextDone保留 |
| sds/adapter/product_source | 无任何包外生产/测试import；只有自己的五生产/五测试互相消费 | 整退，不能借旧KEEP复活旧子任务/状态/缓存 |
| asset/policy | 三函数只有本包目录内外包测试import，无生产消费者 | 两文件整退；asset/recipe与实际资产审批不动 |

**发现并消除的架构问题**：旧CLI/client仍请求 `/api/v1/local-agent/1688-jobs`，当前应用没有注册这组旧HTTP。维护这套源码会宣称一个不存在的当前业务路径；本批删除整条孤立PoC，不把旧模块重新挂载，也不为其创建兼容API/第二任务队列。

**当前真实路径**：current-application 的 product_acquisition_application/browser_capture_application → app/productsourcing → integration/acquisition/a1688/browser 与 browsercollector → 当前 sourcing/SRC-1/Catalog；cmd/1688-public-browser-collector 是无租户凭据的证据生产者。cmd/1688-batch-import与batchcapture消费当前acquisition协议；其名字中的batch local-agent不是旧PoC。当前 integration/sds由app/pod及其currentapplication注入消费。原debug直接使用Alibaba1688Processor.Process，不依赖旧a1688包装。

**共享保留**：NewDefaultConfig虽只有旧CLI的包外生产选择器，但仍由core/config.setDefaults同包生产调用，整个函数/默认值必须保持。PermissionLocalAgentWrite及模块/角色目录仍供当前商品Agent、工具和授权使用，整个authz/身份源码零改。不能因字符串local-agent继续出现就删除共享权限或当前批量设计。

#966/#956/#692分别拥有TLS、Store浏览器与安装边界；不接管其源码/运行。仅Architecture README索引可能组合，最终候选核对实际main真实增量，沿一个Writer/分支/PR，不持续merge-main或重复派工。

## 4. 行为、状态与持久化 owner

| 边界 | 当前owner/不变量 | 本批准确处置 |
| --- | --- | --- |
| 旧LocalAgent Job | Service.mu/内存map，含TTL/claim token/terminal摘要；无持久库/表/外部队列 | 整套源码退出，不迁移/恢复/接管旧Job，不新增新队列/信封/事实源；既有进程内状态UNKNOWN，不查询或终止 |
| 旧deviceauth/CLI | OIDC discovery/device code/polling、token仅进程内传递；CLI可安装/打开浏览器 | 仅退休调用/源文件，不实际请求IdP、读/改token、安装/浏览器/配置、会话或用户目录；当前Auth.js/Login V2/原生身份CLI库保持 |
| 当前来源与保存 | 当前组织授权、SourceEnvelope/Identity、SRC-1/Catalog/发布/replay/lineage、原持久化owner | owner/事务/版本/hash/权限/UNKNOWN全部零改，旧snapshot/Job/DTO无alias或兼容读取 |
| 旧SDS与资产策略 | 旧纯值/规范化/指针canonical修改，没有当前生产入口或独立持久化owner | 删旧包/专属测试，不删canonical/SourceIdentity/Asset.Kind/recipe或当前POD数据/状态 |
| 当前采集/浏览器 | 当前public/扩展/批量入口和原浏览器/debug、真实deadline取消/关闭 | 仅移除两个专属Prepare预热门面；实际创建/安装/处理/错误分类/取消/清理与原有效测试全文保持 |
| 独立SHEIN/Store | 原sheinlogin服务、session transport、身份/仓库DTO、profile/执行归属/fence/UNKNOWN、Store同步 | 全部零生产diff，不用本次旧OIDC/Job取代，不改其SQL/Redis或运行状态 |

### 4.2 JSON/hash、有效测试与物理数据边界

旧CLI/HTTP的Job/Claim/Failure/summary/Snapshot协议、旧设备码DTO和SDS叶包内部值随旧入口整退，无当前外部协议兼容推定，不保留别名、转发、迁移、双读/双写。当前 SourceEnvelope/SourceIdentity 的字段/tag/零值/四个零槽位、Normalize/SourceKey/hash/读取拒绝/PublicationIdentity/版本及current API的JSON严格解码不改。已有表、记录、profile、cache、文件和运行Job是否存在为UNKNOWN，本批不探查、不清理；无数据迁移或环境停用。

**唯一有效夹具改接**：`tests/issue30_sourcing_contract_test.go` 的 `issue30Envelope` 留原函数名/返回类型，去旧a1688 import，直接用当前 `sourcing.SourceEnvelope` 值和既有Normalize/RawSnapshotChecksum构造固定合成事实；不新增生产extract、不复制旧工厂/解析器/整包到测试库。固定源ID123、原URL/Bottle/brand/category/price/image、2026-09-05 UTC时间、raw snapshot、source-run-1/request-1、全部trace/metadata/空值均保持。

未来切换前后仅以任务自有合成夹具在临时文件核对完整JSON、规范化结果、checksum及PublicationIdentity等价，明确这是测试事实保持，不是旧数据迁移。不得用“hash由新夹具自己计算”代替独立前后对比。原所有Test/fixture helper其余正文不改：replay/conflict/耐久重开/lineage/跨组织/并发/取消、源版本、显式审批/撤权/库存scope/readiness/幂等与原失败断言均保留。必要执行仅原测试自有临时SQLite、mock/httptest；当前设计阶段不运行这些业务测试。

## 5. 完整删除/保留矩阵

### 5.1 四十三个整退工件

只删除以下Git追踪源码/专属脚本，禁止目录名推导的递归清理；不删除工作区的未知或忽略文件。整个旧包无空壳、替代main或迁移适配器。

<details><summary>43个整退路径</summary>

```text
cmd/1688-local-agent/main.go
cmd/1688-local-agent/main_test.go
internal/integration/crawler/README.md
internal/integration/crawler/boundary_guard_test.go
internal/integration/crawler/doc.go
internal/integration/crawler/a1688/README.md
internal/integration/crawler/a1688/legacy_product_snapshot.go
internal/integration/crawler/a1688/legacy_product_snapshot_test.go
internal/integration/crawler/a1688/processor.go
internal/integration/crawler/a1688/processor_test.go
internal/integration/crawler/a1688/product_source.go
internal/integration/crawler/a1688/product_source_test.go
internal/localagent/client/client.go
internal/localagent/client/client_test.go
internal/localagent/deviceauth/client.go
internal/localagent/deviceauth/client_test.go
internal/localagent/httpapi/handler.go
internal/localagent/httpapi/handler_test.go
internal/localagent/httpapi/http_module.go
internal/localagent/httpapi/http_module_test.go
internal/localagent/httpapi/routes.go
internal/localagent/httpapi/routes_test.go
internal/localagent/local_agent_e2e_test.go
internal/localagent/logger_port.go
internal/localagent/runner.go
internal/localagent/runner_test.go
internal/localagent/service.go
internal/localagent/service_test.go
internal/localagent/types.go
internal/product/asset/policy/base_selection.go
internal/product/asset/policy/base_selection_test.go
internal/sds/adapter/product_source/apply.go
internal/sds/adapter/product_source/apply_test.go
internal/sds/adapter/product_source/baseline.go
internal/sds/adapter/product_source/baseline_test.go
internal/sds/adapter/product_source/boundary_guard_test.go
internal/sds/adapter/product_source/execution.go
internal/sds/adapter/product_source/execution_test.go
internal/sds/adapter/product_source/images.go
internal/sds/adapter/product_source/images_test.go
internal/sds/adapter/product_source/model.go
scripts/1688-local-agent-acceptance.ps1
scripts/1688-local-agent-acceptance.Tests.ps1
```

</details>

### 5.2 两个混合生产文件：只退专属预热

| 文件 | 允许 | 保留 |
| --- | --- | --- |
| internal/crawler/alibaba1688/processor.go | 仅Alibaba1688Processor.Prepare完整声明/专属注释与唯一errors import退出 | 其它imports、类型/字段、构造、Process/Shutdown全文 |
| internal/crawler/alibaba1688/single_processor.go | 仅SingleProcessor.Prepare完整声明/专属注释退出 | errors仍用于原错误分类；其它全部imports/字段/构造/函数，包括newPublicBrowserManager/processWithBrowserManager/closeOnContextDone与原所有测试全文 |

### 5.3 五个混合测试/检查文件

`tests/issue30_sourcing_contract_test.go`、`tests/listingkit_backend_retirement_test.go`、`tests/import_boundaries_test.go`、`tests/repository_structure_test.go`、`tests/architecture_docs_test.go`。仅§4.2与§6.1列出的函数/行允许转换；新增一个 `TestLegacyLocalAgentSourcePolicyClosureIsRetired`，不开发新检查器/runner/验收平台，其余原函数/helper/import/fixture/断言保持（旧a1688 import一行退出除外）。

### 5.4 八个当前文档与四十四个历史顶注

| 当前文档 | 未来准确变化 |
| --- | --- |
| 本文、Architecture README、Legacy Register | 只更新本批当前段；完整冻结设计和原索引/登记后缀保持，不冒充源码/运行验收 |
| docs/architecture/project-boundaries.md | 只加当前采集/POD/资产owner及旧闭包退出顶注，原完整Git blob后缀保留 |
| docs/product/product-sourcing-handoff.md | 只加当前采集owner与旧POD/旧来源表为历史观察的顶注，原完整Git blob后缀保留 |
| docs/development/repository-structure.md | 顶注+准确删除当前cmd清单的 `1688-local-agent` 一行及独立维护说明中指向其验收脚本的一行（原基线155/173行）；其余目录/说明/历史/数字不改；原完整blob以起始main引用留证，不复制成仍被解析的当前目录 |
| internal/product/sourcing/README.md | 仅把当前adapter列表中旧crawler/a1688、sds/adapter/product_source两行替换为当前integration/acquisition/a1688/browser与integration/acquisition/browsercollector；保留原Amazon当前项和其它身份/JSON/hash/发布事务/权限/预算/领域边界全文；不加兼容owner |
| docs/architecture/architecture-review-checklist.md | 仅把旧专属TestLocalAgentHTTPAPIReadsIdentityFromNeutralContext引用换新完整退休guard；其它检查/角色/计数/引用保持 |

44历史文件未来仅加窄顶注，准确替代本批旧CLI/适配/policy/专属Prepare KEEP或旧操作要求；其它内容/批准/失败/数字/勾选以完整原Git blob后缀逐字保留。它们不恢复旧迁移、内部试用或生产前置。设计阶段这些文件均不改。

<details><summary>44个仅加历史顶注的路径</summary>

```text
docs/architecture/legacy-1688-captcha-experiment-closure-retirement-boundary.md
docs/architecture/legacy-amazon-source-adapter-closure-retirement-boundary.md
docs/architecture/legacy-config-facade-closure-retirement-boundary.md
docs/architecture/legacy-crawler-facade-diagnostics-closure-retirement-boundary.md
docs/architecture/legacy-global-config-logging-tail-closure-retirement-boundary.md
docs/architecture/legacy-host-image-helper-closure-retirement-boundary.md
docs/architecture/legacy-host-runtime-closure-retirement-boundary.md
docs/architecture/legacy-idle-provider-bootstrap-closure-retirement-boundary.md
docs/architecture/legacy-idle-support-contract-closure-retirement-boundary.md
docs/architecture/legacy-listing-cli-retirement-boundary.md
docs/architecture/legacy-listing-preview-download-closure-retirement-boundary.md
docs/architecture/legacy-logger-facade-closure-retirement-boundary.md
docs/architecture/legacy-numeric-acquisition-tail-closure-retirement-boundary.md
docs/architecture/legacy-numeric-value-contract-closure-retirement-boundary.md
docs/architecture/legacy-openai-adapter-closure-retirement-boundary.md
docs/architecture/legacy-ownership-controller-retirement-boundary.md
docs/architecture/legacy-platform-api-closure-retirement-boundary.md
docs/architecture/legacy-product-policy-closure-retirement-boundary.md
docs/architecture/legacy-retirement-parent-closure-boundary.md
docs/architecture/legacy-sds-runtime-closure-retirement-boundary.md
docs/architecture/legacy-shein-policy-contract-closure-retirement-boundary.md
docs/architecture/legacy-shein-sensitiveword-closure-retirement-boundary.md
docs/architecture/legacy-source-url-timeout-closure-retirement-boundary.md
docs/architecture/legacy-task-identity-mapping-closure-retirement-boundary.md
docs/architecture/legacy-tenant-prompt-closure-retirement-boundary.md
docs/architecture/shein-worker-identity-persistence-boundary.md
docs/development/1688-local-agent-poc.md
docs/product/product-sourcing-mvp-plan.md
docs/refactoring/2026-09-05-issue-30-prepared-slices.md
docs/refactoring/module-target-mapping.md
docs/refactoring/project-migration-roadmap.md
docs/superpowers/plans/2026-08-20-decouple-1688-source-handoff.md
docs/superpowers/plans/2026-08-21-1688-local-agent-poc.md
docs/superpowers/plans/2026-08-22-local-agent-contract-hardening.md
docs/superpowers/plans/2026-08-23-listingkit-review-contracts.md
docs/superpowers/plans/2026-08-26-authidentity-facade-retirement.md
docs/superpowers/plans/2026-08-26-image-agent-core-manual-workflow.md
docs/superpowers/plans/2026-09-01-internal-target-architecture-phase3-product.md
docs/superpowers/specs/2026-08-08-1688-controlled-replay-design.md
docs/superpowers/specs/2026-08-20-decouple-1688-source-handoff-design.md
docs/superpowers/specs/2026-08-21-1688-local-agent-poc-design.md
docs/superpowers/specs/2026-08-22-local-agent-contract-hardening-design.md
docs/superpowers/specs/2026-08-26-image-agent-workflow-design.md
docs/superpowers/specs/2026-09-05-issue-30-sourcing-cutover.md
```

</details>

**明确零改的近名资料**：服务端public browser冻结设计、2026-09-21批量local-agent设计、phase2-runtime-inventory、product-sourcing-inventory。前两项属当前实际能力；后两项保留历史映射/数字，当前owner由本设计及handoff/boundary顶注承接，完整inventory零改。当前authz、批量队列/上限、Dub/OpenMeter、原生身份、所有当前采集/图像/资产/POD/SHEIN源码零改。

43整退 +2混合生产 +5混合检查 +8当前文档 +44历史＝**102路径**；其中未来101个既有文件/一个新设计文件，当前仅3文档。没有未列的源码/检查/依赖清理授权。

```text
Legacy decision: RETIRE
Reusable behavior: 当前有效来源回归直接用现有中立SourceEnvelope/Normalize与固定合成事实；浏览器/身份/POD/资产当前能力原owner原样保留
Current owner: current acquisition/browsercollector + sourcing/SRC-1/Catalog；integration/sds/app/pod；Product Asset/recipe/ImageAgent；现有SHEIN/Store
Cutover/deletion condition: 用户确认完整102路径与§6.1、IMPLEMENTATION_READY读回后，同一批删除旧入口/状态/协议/包/预热及专属尾部；无alias/替代owner/迁移/双事实源
```

## 6. 实施与验证

当前三设计文档只核对链接、§1–§7/§4.2/§6.1、102唯一路径/消费者/矩阵、原索引登记完整后缀、git diff --check与适用原文档检查；不为纯文档制造业务RED/编译/资源验收。准确设计HEAD的完整原PR CI和设计复核单列，旧批证据不批准本批源码。

未来确认与IMPLEMENTATION_READY/Ready读回后，单Writer先按§6.1建立准确guard，在旧代码上取得真实RED，再同一批完成切换/删除至GREEN。有效夹具只用§4.2独立前后事实对比；旧状态/HTTP/deviceauth/工厂/纯策略专属测试随需求退出，不移入新测试包保活。必要自检复用原来源/发布/资产/readiness测试（自有SQLite）、原身份/路由/退休/文档检查、原浏览器无浏览器mock/取消清理/URL/质量/PoolManager测试和SHEIN相关日志/安全回归；不访问真实provider、浏览器、PG/Redis或既有资源。

仅编译当前current-application、shein-login-worker、image-agent-temporal-worker、1688-public-browser-collector、1688-batch-import及原hack/debug/test-1688、test-amazon七入口到任务临时目录，不执行。没有新构建/验收平台，不重复完整CI已覆盖且代码无变的同项检查。最终候选前再核对实际main组合；准确源码HEAD完整原CI与最终交付复核分别留证。

### 6.1 原检查、文档与冻结边界的准确转换

| 文件/原函数 | 仅允许变化 | 不得改变 |
| --- | --- | --- |
| listingkit_backend_retirement_test.go / TestLegacyAlibaba1688CaptchaExperimentClosureIsRetired | 仅从原retained消费者token表删除旧a1688/processor.go、localagent/runner.go两整行项 | 原算法、25保留验证码函数/point/七原测试、原两个debug消费者、其它源/导入/解码/取消/当前算法断言与失败关闭 |
| 同文件 / TestLegacySDSRuntimeClosureIsRetired | 仅删isRetired对product_source的三行例外；原adapter退休前缀即覆盖整个叶包 | 其它旧SDS源/配置/授权/样例禁入、所有原读取/解析/异常处理与三当前命令依赖检查原样 |
| 同文件 / TestCurrentCommandsDoNotCompileLegacyBusinessBackends | 仅从原八命令列表删1688-local-agent一项 | 其它七命令、全部原backend/平台API/旧包禁令、go list与失败处理原样，不missing-skip |
| 同文件新增 TestLegacyLocalAgentSourcePolicyClosureIsRetired | 复用原trackedFiles/AST/decoded-import/缺席设施，检查43明确工件与旧根/命令/脚本缺席、禁止本批四旧包根回流（integration/crawler整个父根、localagent、sds/adapter/product_source、product/asset/policy）与两个receiver.Prepare；正向检查当前采集入口/原Process/浏览器创建取消清理/SourceIdentity与中立fixture/当前POD/资产/SHEIN及共享PermissionLocalAgentWrite保留 | 不按泛词localagent、sds、Prepare、Claim或Permission全仓禁入；不搬旧工厂/DTO到新包，不空目录/dummy/skip；读取/解析/导入错误失败关闭。原assertLegacyLoggerFacadeRetired的localagent负向root和其它所有原函数保持 |
| import_boundaries_test.go / TestLocalAgentHTTPAPIReadsIdentityFromNeutralContext | 整个旧HTTP专属函数退休，新完整guard承接旧根禁入 | TestSheinLoginReadsIdentityFromNeutralContext、其它当前中立身份/Gin/域边界/认证与全部断言原样；不取消当前安全要求 |
| 同文件 / businessHTTPPackages | 仅删localagent/httpapi一条allowedHTTPPackages赋值 | 原遍历/匹配/错误处理、其它所有允许/禁止与authz当前源码不变 |
| import_boundaries_test.go / TestAppCrawlerFetcherCompatibilityLayerIsRetired | 仅将原失败诊断的integration/crawler推荐改为当前integration/acquisition（基线1896行） | 原os.Stat/文件缺席/错误处理/失败关闭及其余全部断言逐字保持，不增加skip |
| issue30_sourcing_contract_test.go / issue30Envelope | 仅去旧a1688 import与helper内旧工厂，按§4.2改固定当前中立值+既有Normalize/Checksum | 其余所有原函数/import/SQLite fixture/失败路径/发布与资产等全部断言逐字保持；不把当前安全测试随旧包删除 |
| repository_structure_test.go / maintainedOperationalCommands | 仅删1688-local-agent一条目录项 | 原其它命令、product/operation分类、当前目录文档解析与文件/错误断言保持 |
| architecture_docs_test.go / TestProjectBoundaryDocumentDefines1688SourceHandoffOwnership | 仅替换旧a1688 converts legacy...期望短语为当前integration/acquisition/a1688/browser/browsercollector与SourceEnvelope owner短语 | Product不依赖integration等永久边界、Greenfield/无迁移/其它原短语与失败关闭原样 |
| 当前checklist/repository structure/handoff/boundary与历史 | 只按§5.4逐项转换；README索引明确列本文文件名，避免新文档漏索引 | 其它所有原文档检查/角色/目录/计数不变；仅当前命令清单/维护说明两行、sourcing当前adapter列表两行及旧检查引用一项可准确替换；其余原文/历史完整保持 |
| 完整inventory/workflow/依赖/保护/其它检查 | **零改**。未发现三组旧目标对应的CURRENT条目，无CURRENT校准需要；现有logger/AI/DB/Redis/身份owner仍有效 | CURRENT/历史/数字/ceilings/accuracy/parser/CodeHealth/lint/必需与安全检查/保护均原样，不因类名相近删共享权限或有效配置 |

若真实CI发现本表遗漏，先检查同根sibling、报告实际失败与准确最小补丁，取得用户精确例外决定后再改；不能把未列项当默认可调或用缺文件跳过。非当前Must缺陷不追加产品/全局hardening/验收平台。

## 7. 复核、确认与停止点

1. 当前只交三文档设计、准确HEAD/原CI/适用独立设计复核，未来生产保持NOT_READY。正常Architecture Review最多两轮，finding先按AGENTS分类；只有命中当前Must及明确BLOCKER后果才重开架构，非Blocker不无限设计。
2. 用户确认本§1–§7（含§4.2/§6.1）后，在同一Issue/PR写入并读回Independent Architecture / IMPLEMENTATION_READY / Ready；root单Writer连续完成102路径，不再机械拆批。设计/CI/AI复核均不代替用户实施决定。
3. 按原GitHub机制核对真实diff。自动复核未完成至多重试一次；**未经用户本轮授权不默认新增 AI 互验线程**，#969设计/源码授权不延伸本批。设计与源码最终交付复核分开，源码形成后一次集中最终检查，修复只复核相关增量，有效未变证据明确复用。
4. 交出当前有效入口/owner、准确源码HEAD、完整原CI、复核/处置与NOT_RUN边界；入口/用户路径不变，不提供旧CLI替身。原测试/自检/CI/另一AI不签发运行/产品验收或生产可用。保持既有生产发布NOT_READY，不增加内部试用前置。
5. 准确HEAD合并、部署、真实环境操作、#301关单分别授权；获合并授权才正常expected-head merge，之后仅确认实际merge SHA匹配push/main完整run completed/SUCCESS，不用PR CI/单gate替代，不绕过保护。#301始终OPEN，仅更新当前段，既定#726 marker起历史/数字/勾选逐字保留；独立SHEIN登录与Store同步保持，不读/操作既有实例/数据/凭据/输出/进程。
