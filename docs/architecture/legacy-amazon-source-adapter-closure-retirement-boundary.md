> **#948 窄替代注记（已获用户实施确认，源码已实施 / In Review）**
> #948 替代本设计中闲置的 AmazonProcessor.ProcessBatch KEEP；当前 Process/ProcessWithContext/ProcessBatchWithContext、真实批量 owner 和关闭行为原样保留。
> 唯一准确范围见 [#948 冻结设计](legacy-crawler-facade-diagnostics-closure-retirement-boundary.md) §1–§7（含 §6.1）；以下原全文为历史证据，原数字、勾选及其它 KEEP 不变。

> #301 / #947 **SUPERSEDED，仅闲置URL/地区便利与空超时接线**：用户已确认[完整设计§1–§7/§6.1](legacy-source-url-timeout-closure-retirement-boundary.md)。本文原crawler/amazon Processor/浏览器/辅助的广泛KEEP，只替代本批URL两方法、DomainResolver九函数/languageMap、TimeoutManager及其字段/构造/赋值/Shutdown专属尾部；不混淆已退integration同名解析器。NewAmazonProcessor/当前单批/context/有效池大小、URL/ASIN/币种/marketplace/地区邮编/有效规则、真实PoolManager取消/生命周期、采集/原浏览器debug与独立SHEIN登录/Store同步保持。workflow/完整inventory及既有资源不动，无试用前置，#301 OPEN。以下原完整Git blob/历史/数字/勾选逐字保留。

> #301 / #946 **SUPERSEDED，仅闲置浏览器便利/配置旁路/诊断/统计闭包**：用户已确认[完整退休设计§1–§7/§6.1](legacy-browser-management-tail-closure-retirement-boundary.md)，只替代本文原Processor全方法/池/诊断广泛KEEP中的无人调用三统计方法及§5专属尾部。原七工件退休、NewAmazonProcessor/Process/Batch/context/Shutdown/有效池大小/factory_test、真实指纹/代理评分/质量重试/周期健康日志、当前采集/原浏览器debug及独立SHEIN登录/Store同步保持，原恢复/关闭/UNKNOWN不变。workflow/完整inventory/既有资源不动，不增试用前置；以下原完整Git blob/历史/数字/勾选逐字保留。

> #301 / #926 **SUPERSEDED，仅本批旧平台调用闭包**：用户已确认[完整退休设计 §1–§7/§6.1](legacy-platform-api-closure-retirement-boundary.md)。本文涉及旧 Amazon/TEMU API、SHEIN 闲置 Client/manager/DTO、SPAPI/旧平台包装配置的 KEEP/当前 owner 建议，只在该准确删除/保留矩阵内被替代；不移植旧 auth refresh/retry 或平台副作用。当前 SHEIN DTO/完整 JSON/有效规则、SessionTransport/三个读取端点、采集/原 Amazon 浏览器/debug、官方 Store、独立 SHEIN 登录与 Store 同步及原 owner/fence/UNKNOWN 保留。只抽原仓库两纯函数，不加 compat；workflow/完整 inventory/计数/检查逻辑及既有实例/数据/凭据/进程不动，不增加试用前置。以下原完整正文、历史数字/勾选/批准/证据逐字保留。

# #301 旧 Amazon 来源适配层完整退休闭包设计

Status: **USER_CONFIRMED / SOURCE_IMPLEMENTED**。用户已确认 §1–§7（含 §6.1），源码修改前已在 #301/主要 PR 记录并读回 **Reuse Existing Architecture / Ready**；七个准确工件及专属文档尾部在同一 PR 整批退休。当前采集、原浏览器/池/有效池大小测试与调试、共享合同及独立 SHEIN 登录/Store 同步保持；workflow/inventory/其它检查/依赖/保护不改，没有真实资源操作或新试用前置。下文设计阶段的“确认前/拟/待授权”描述保留为冻结设计依据，不撤回本次准确实施授权。准确源码 HEAD、RED/GREEN、原 CI、源码最终复核及另行合并批准留主要 PR，设计证据不替代实现或运行验收。接单及前当前段存档见 [#301 / 6357545307](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6357545307)，实施接单见 [#901 / 6358228880](https://github.com/qq550723504/shuomi-ai-commerce/pull/901#issuecomment-6358228880)。#301 保持 OPEN。

调查基线 main **`eeb54dda7cb0da9fef0b310456cb8594b9dc3bf3`**，已包含 #900；该 merge 的 push/main CI39899645077 完整 SUCCESS 是前批证据，不代替本次设计或后续实现检查。

## 1. 用户结果、完整范围与权限

使用者为维护当前采集、数据服务与 Sourcing 的维护者。交付一个完整源码结果：旧 Amazon 来源适配层及其最后消费者退出后的专属构造/接口尾部退出，当前采集与调试仍由已有 owner 承担。不按构造器、规划、转换或测试拆小批，不新增产品能力或内部试用前置。

根因是过时 owner 指向和无装配的第二条来源抽象仍存在：`integration/crawler/amazon` 没有包外追踪 Go import，却仍包含 Legacy 包装、未知地区回退美国站、SHEIN/TEMU 转 Amazon、旧 model→SourceEnvelope 和批量对齐协议；部分文档仍称它为当前入口。[已批准数据服务设计](data-services-v1.md) §3 已要求当前路径不依赖这个 wrapper/fallback，当前 runtime 实际注入的是 `integration/acquisition/amazon`。本轮退休旧壳，不再包装旧 Service 或移名保活。

完整范围为旧包的两个生产文件、两个专属测试及 README；最后调用者退出后的 `crawler/amazon/factory.go`，以及仅剩自身编译断言的 `crawler/amazon/scraper.go`；原 guard 和准确当前说明尾部见 §5/§6.1。原浏览器 Processor、池、提取器、有效池大小规则和 `factory_test.go`、调试入口、当前 acquisition/1688、共享合同与独立 SHEIN 登录/Store 同步明确 KEEP。

本轮只设计。确认前不删源码、不改测试/guard或运行旧构造器；不读既有私密配置、凭据、Profile、缓存、日志或 ignored 状态，不操作既有实例、数据、凭据或进程，不执行应用/浏览器/provider入口。workflow/inventory 全零改，不增加试用前置；实施、合并、部署、关单另授权。#301 保持 OPEN，#29 不重开，不接管 #692。

## 2. Design Basis、Authority 与不变量

设计阶段 **Design Basis: N/A**：只读调查和三文档，不改变产品/领域/运行语义。实施拟 **Reuse Existing Architecture**：复用当前 dataacquisition/acquisition、Sourcing/Catalog/SRC 与原浏览器各 owner；没有现役调用者切换，没有新业务/公共领域合同、事实 owner、schema、权限/租户、状态机/事务、恢复、计费/provider或副作用边界。用户确认 §1–§7（含 §6.1）后，同一 PR 先记录并读回 Reuse Existing Architecture / Ready，再整批实施。

适用依据：

- [Greenfield](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)：只有 EXTRACT/RETIRE，无兼容、旧 ID 映射、双读/双写、fallback 或第二事实源。
- [数据服务 v1](data-services-v1.md) §1/§3/§4：已批准的明确站点、匿名证据和当前 Product/Resource/Temporal owner，不借本轮新增输入、平台、预算或恢复要求。
- [来源业务闭环](source-business-closure-v1.md)、[项目边界](project-boundaries.md) 与 [SHEIN 独立持久化边界](shein-worker-numeric-persistence-retirement-v1.md)：当前 Sourcing/lineage、Organization、Store、独立 worker/fence/UNKNOWN 保持。
- [#867 数字采集尾链](legacy-numeric-acquisition-tail-closure-retirement-boundary.md)、[#868 来源值合同](legacy-numeric-value-contract-closure-retirement-boundary.md) 与 [#822 Amazon 刊登闭包](legacy-amazon-listing-closure-retirement-boundary.md)：其旧适配层/构造/接口的广泛 KEEP 只在本设计确认的准确对象上被 supersede；原浏览器、平台 API/规则、共享 model/SourceRequest/SourceIdentity 保留，历史正文不重写。

Must：完整闭包、无新 wrapper/迁移；保留全部当前采集/原浏览器/调试和共享行为、SHEIN 登录/Store 同步；workflow/inventory/其它检查/依赖/保护不改，无真实资源操作或试用前置。Should：使用原 guard、原测试设施与当前已存在行为，不新造 mapper、框架、runner 或验收平台。Out of Scope：整退 crawler/amazon、Amazon API、1688/localagent、通用 model/词库/Prompt/配置，以及新数据服务或环境验收。Accepted Risks：无新增；运行/用户验收 NOT_RUN、物理实例/数据状态 UNKNOWN。UI/IA/Figma N/A。

Threat Model 仅覆盖误删真实采集/共享值/有效测试、漏留 Legacy wrapper/工厂/接口、把未装配回退重新接入当前 Product、把源码退休当真实清理。没有新的授权/租户协议，不扩全局安全目标。若实际需要改变当前 owner/公共合同/持久化/权限或副作用边界，停止该部分正式实现，升级 Independent Architecture；不能以小改动或拆批规避。

## 3. 实际消费者：contract → implementation → injection → consumer

| 对象 | 基线源码/装配证据 | 决定 |
| --- | --- | --- |
| integration/crawler/amazon | 全仓 Git 追踪 Go 的包导入及专属符号检索：包外无 import/构造/调用；自己的两个测试覆盖自己的包装/规划/转换 | 整包 RETIRE，不按导出名称推定外部兼容义务 |
| Source/Processor/LegacyCrawlSource/NewLegacySource/NewLegacyCrawlSource | 只有包内包装/委派；NewLegacySource 是 CreateProcessor 唯一调用点 | 同批完整退构造、单/批请求、Shutdown 委派；不删除实际 Processor/Shutdown |
| AmazonCrawlRequestPlanner/DomainResolver/ZipcodePolicy/SourceFetcher | 只在该包及原测试使用；包含域名/语言/邮编默认与 batch→sequential 分支 | 整退旧规划/默认/回退与专属 DTO；不强加到当前明确站点合同 |
| AmazonSourceEnvelope/SourceProductResult/NormalizeAmazonBatchResults | 只在该包及原测试；旧 model 值转换、raw checksum/时间/图片/价格/变体映射及结果对齐 | 退旧 mapper/helper；当前 Sourcing/SourceEnvelope/hash/lineage 和当前 parser 独立保留，不新建替代 mapper |
| crawler/amazon.CreateProcessor | factory.go 只做 NewAmazonProcessor 构造和池大小日志，唯一外部调用为上述旧包 | 最后消费者退出后整文件 RETIRE；NewAmazonProcessor/effectiveBrowserPoolSize 的真实 owner KEEP |
| crawler/amazon.Scraper | scraper.go 中接口与其自身 `var _ Scraper = (*AmazonProcessor)(nil)` 断言之外无追踪消费者 | 整文件 RETIRE；不按名称全局禁用其它当前 Scraper/Source/Processor |
| 当前 Amazon 数据服务 | runtime/currentapplication/runtime.go 注入 acquisition/amazon.New；data_services.go 验证原 Options；dataacquisition 只消费其中立证据 | client/parser、固定站点/SSRF/限制/取消/挑战/归属、当前保存/计量/幂等/恢复 KEEP，源码零改 |
| 原浏览器与调试 | hack/debug/test-amazon/main.go 直接 NewAmazonProcessor→Process→Shutdown；原 root/browser/extractor/variations 和工厂名测试独立有效 | 原调试入口、配置/日志/结果写出与浏览器行为 KEEP，不运行或重配 |

只读 `go list -deps`：current-application 包含当前 acquisition/amazon、不含旧 integration/crawler/amazon 或原 crawler/amazon；两个独立 worker 不含这三包。调试入口属于 `hack/debug/go.mod` 独立模块，在其目录以 `go list -mod=readonly -deps ./test-amazon` 核实包含原 crawler/amazon、不含旧 integration 包。这里只查编译元数据，未运行构造器/应用/采集，也不把本机 GOOS 图冒充全部 build tags；全追踪导入/声明保护见 §6.1。

## 4. 持久化与副作用 owner

| 边界 | owner / 本轮准确变化 | 保留及失败语义 |
| --- | --- | --- |
| 旧 adapter/请求/Envelope/结果 | 仅进程内委派/纯值，无 SQL/Redis/文件 schema 或 durable owner | 无继任持久化 owner；不迁移/删除已有商品、snapshot、缓存或结果，不继续旧对齐/回退协议 |
| 旧工厂/Shutdown 包装 | 构造实际浏览器与委派关闭的旧入口，当前无包外装配 | 仅删追踪源码入口，不构造/关闭浏览器，不接管、drain 或终止既有进程 |
| 当前 Amazon 采集执行/证据 | acquisition/amazon + dataacquisition，原 runtime 注入；Product SRC/Catalog/Collection 保存 | 固定站点/neutral DTO、权限/创建者隔离、UoW、lineage/计量/幂等/UNKNOWN/恢复保持，本轮不发请求或改变这些合同 |
| 原浏览器进程/池/诊断输出 | crawler/amazon 原 Processor/PoolManager/BrowserPool/InstanceProcessor 与 debug caller | 原取消/重试/风险/代理/邮编/质量/失败证据/Shutdown及原文件输出 owner 保持，真实状态 UNKNOWN |
| 共享值与独立 worker | 原 model/Sourcing/规则、组织/Store/SHEIN runtime 各 owner | SourceRequest 现有字段、SourceIdentity 六 Source* 与四 JSON 零值槽位、规范化/hash/读取拒绝合同、两 SHEIN 属性与全部此前能力零改 |

没有新的登录/凭据/事务/恢复或外部 effect，不需要新 outbox/Saga/队列/数据库。当前新 adapter 已按批准架构持有其有效解析/校验行为；本批没有需要新增代码抽取的现役行为，也不以保留无调用算法为由包装旧 owner。删除旧副作用入口不等于确认任何真实运行态已退出。

## 5. 完整删除、保留与抽取矩阵

| 准确工件 | 拟同批处置 | 保留边界 |
| --- | --- | --- |
| internal/integration/crawler/amazon/processor.go | 整退 Source/Processor/LegacyCrawlSource、所有工厂/方法/Shutdown 委派 | 原 crawler/amazon 的真实构造/Process/批量/取消/关闭不动 |
| internal/integration/crawler/amazon/product_source.go | 整退全部 resolver/default maps/语言邮编、planner/input、Envelope mapper/helper、fetcher/两个 source port、平台映射、batch result DTO/normalizer | 当前 acquisition/amazon、dataacquisition、Sourcing/SourceRequest/SourceEnvelope/hash、model、1688 adapter 全部零改 |
| 同目录 processor_test.go、product_source_test.go | 随旧 owner 整退两个专属包装/规划/转换测试；不搬旧 fallback/顺序回退/DTO 协议 | 原 root/browser/extractor/variations、当前 acquisition/dataacquisition/共享 hash/Source JSON、其它安全/规则测试不删不弱化 |
| 同目录 README.md | 随包整退，不留空壳或“新代码继续放此处”指令 | 原 parent crawler 与当前 acquisition 的有效说明保持其准确 owner |
| internal/crawler/amazon/factory.go | 最后调用者退出后整退 CreateProcessor 和仅它使用的构造/池日志尾部 | processor.go、NewAmazonProcessor、effectiveBrowserPoolSize、factory_test.go 原两个有效池大小断言完整零改 |
| internal/crawler/amazon/scraper.go | 整退孤立 Scraper 与自身 compile assertion | 原 Processor 的全部方法/接口实现能力与其它同名当前合同保持 |

共七个准确整退工件（六 Go/含两个专属测试 + 一个 README）。再加 §6.1 的原 guard、五个准确当前说明以及本轮三文档，共十六个范围工件，同一主要 PR 整批完成；不删原 crawler 根，不扩展至其它未装配平台或 HTTP 包。独立设计复核识别的两处 ACTIVE/current Amazon owner 说明在下表明确收齐，不把它们归入 dated 观察豁免；这只校准设计的完整文档尾部，七工件及源码/共享合同保留边界不变。

```text
Legacy decision: RETIRE
Reusable behavior: 当前 parser/原浏览器及共享值已由其现有 owner 持有，本批无需新增 EXTRACT 代码
Current owner: acquisition/amazon + dataacquisition/Sourcing/Catalog/SRC；原浏览器/debug、SHEIN/Store owner 各自保持
Cutover/deletion condition: 设计确认及 Reuse Existing Architecture / Ready 读回→原guard RED→七工件与专属尾部整批退→有效原测试/编译/原CI及准确源码最终复核；不等于运行态清理
```

## 6. 必要验证与证据

设计阶段只检查三文档、链接/当前索引、准确当前调用/编译元数据和适用原文档/Phase2 检查，提交准确设计候选的原 CI 与一次设计复核。不为文档制造业务 RED，不跑旧 adapter 测试/构造器来保活旧协议。

实施获准后先在原 retirement guard 捕获旧包/两个原 root 尾部仍存在的 RED，再整批退出至 GREEN。保留当前 acquisition/amazon 的固定站点/解析/挑战/限制回归、原 root 的有效池大小/处理器/批量/取消/质量 mock、Sourcing/SourceIdentity/hash 等原有效测试；没有业务新实现，不写镜像旧包装协议的新测试。只运行与改动相关的既有内存/合成 fixture 和原必需 CI，不新增故障矩阵、runner 或验收平台。

必要元数据/编译为三个当前命令和独立 debug 模块的原 Amazon 入口；只构建至任务 TEMP，不执行产物/浏览器、下载或 provider。真实采集/SHEIN/Store 未变化代码和有效证据可明确复用，不能把本批源码缺席或 design/CI 绿灯当平台/用户验收。

### 6.1 原检查、文档、inventory 与 workflow 精确处置

当前仅提案，用户确认前全部不应用：

| 对象 | 实施获准后的准确处置 | 不得改变 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 原 tracked Go/AST/decoded-import 设施增一个本闭包 guard：旧包/七文件缺席、全追踪Go禁止导入旧包；仅 crawler/amazon owner 的 CreateProcessor/Scraper 声明缺席；正向保持 NewAmazonProcessor/原处理器及方法、effectiveBrowserPoolSize/factory_test、debug 的直接调用与当前 runtime 的 acquisition 注入 | 原 #900/#897/其它守卫、读取/解析失败即失败、当前命令/route/logging/品牌/安全检查不改；不全局禁 Source/Processor/Scraper，不 missing-skip/dummy/新runner |
| internal/integration/crawler/README.md | 仅准确移除旧 Amazon approved child，指向当前 acquisition/amazon，并保留原浏览器/debug owner；不改变 a1688 或父 guard 的其余合同 | 本轮不重写父边界/其它子包规则，不改 doc.go/boundary_guard_test.go |
| docs/architecture/amazon-crawler-runtime-flow.md | 准确区分当前 runtime→acquisition/amazon 与 debug→原 NewAmazonProcessor；去掉旧 Source/CreateProcessor 的“当前入口/测试”指令，原浏览器 owner/能力和真实状态限制保持 | 不恢复旧发布/Task/缓存、不修改真实实例或把 debug 宣称已部署业务入口 |
| docs/refactoring/code-health-decisions.md | 仅添加本轮准确 supersession 顶注：旧 adapter 和 CreateProcessor/Scraper 退出，原 crawler 其余 owner 与测试保持 | 原表格、retained 状态/88与5等历史计数/失败正文逐字保留；不改变检查/ceiling/扫描算法或借台账制造消费者 |
| internal/product/sourcing/README.md | 仅将“Concrete source adapters currently live in”列表的 `internal/integration/crawler/amazon` 路径替换为当前 `internal/integration/acquisition/amazon` | 原六 Source*、四 JSON 零槽/规范化/hash、Atomic source publication、事实/权限/事务/接口和其它列表条目（含1688/SDS）全部不改 |
| docs/product/product-sourcing-handoff.md | 仅将 ACTIVE/current owner 表 Fetching/parsing/provider DTOs 一行中的旧 Amazon 路径替换为当前 `internal/integration/acquisition/amazon` | 该行其它 owner/cells、原产品/共享合同/权限/事实 owner/历史观察基线及其余正文原样，不顺带校准其它平台或改当前需求 |
| 本文、Architecture README、Legacy Register | 三文档的本批设计/当前索引与准确 supersession | 原 #822/#867/#868/#900 及其它历史正文/矩阵/计数保留；dated product-sourcing inventory/roadmap/module mapping、历史 commercialization/deployment 记录不作为当前调用证据，准确本闭包由本文及更新 runtime flow 指引 |
| phase2-runtime-inventory.md 及其它 inventory | **零改**；本包不占四条 CURRENT 行，本轮无需校准 | CURRENT/历史/计数/全部 ceilings/accuracy/parser/排除规则原样；自然减少源码不修改上限 |
| 其它测试/lint/CodeHealth/import/SDK/SSA、全部 workflow/命令/部署/依赖/锁/保护 | **零改**，原目标和检查照常执行 | 不删skip门禁、不重跑求绿/管理员绕过/强推，不更新任一 go.mod/go.sum（含 debug）或读取凭据 |

若真实 CI 要求本矩阵外编辑、发现有效消费者、必须改冻结 KEEP 或公共合同，先提供精确提案/原因；不能自行扩大删除或校准 inventory/原检查。文件名 factory_test 不等于只服务旧 factory；其真实调用为 processor.go 中 effectiveBrowserPoolSize，必须保留。

## 7. 准入、独立复核、停止及父目标边界

1. 本轮仅 N/A 设计三文档、适用原检查/准确设计复核；待用户确认 §1–§7（含 §6.1），不把设计 PASS 当实施或运行结果。
2. 确认后原唯一 Writer 在同一主要 PR 记录 Reuse Existing Architecture / Ready 并读回，再整批实施；没有新 owner/边界则不重复全局架构审核。真实需要改变准入声明的边界时按 §2 停止升级。
3. 设计复核与完整源码最终复核分别绑定准确 HEAD；未变有效证据可复用，finding先按当前Must/后果/阻塞层分类，只复核修复增量。GitHub启动失败不是 finding/PASS；Running不重复触发，未经用户授权不默认新增AI互验线程。
4. 具体当前外部可观察兼容/持久化运行态证据出现时先报告用户；不存在可自行执行的兼容/迁移Exception，不因旧导出构造器、README或专属测试保活旧抽象。
5. 保留当前采集、原浏览器/调试、全部有效共享合同、独立SHEIN登录/Store同步/fence/UNKNOWN；不接管 #692，不操作既有实例/数据/凭据/进程或增加试用前置。
6. 完成实现后交付准确源码/原CI/最终复核证据，合并须另获准确HEAD授权；合并后只以实际merge SHA的push/main整轮结果确认主干CI。不签发产品/运行/生产验收。
7. #301保持OPEN、#29不重开；更新Issue前新读正文，仅当前段与准确PR证据可变，从“以下保留 #726 已合并批次与历史证据，不扩大本轮权限。”起的历史后缀及勾选逐字保留，正文不超过65536字符。实施/合并/部署/关单权限分别判断。
