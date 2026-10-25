# #301 旧采集 URL／站点便利接口与闲置超时管理完整退休边界

Status: **DESIGN_ONLY / AWAITING_USER_CONFIRMATION**。用户本轮授权三组一起设计，尚未授权实施。设计阶段 **Design Basis: N/A / Ready**，仅本文、架构索引、Legacy Register 三文档；未来实施拟 **Reuse Existing Architecture / NOT_READY**。确认 §1–§7（含 §6.1）并在 Issue/PR 记录、读回 Ready 后，才可在同一主要 PR 修改源码和测试。#301 保持 OPEN，合并、部署、关单另行授权。

## 1. 用户结果、范围与 Design Basis

使用者是维护当前采集、原 Amazon 浏览器/debug 和独立 SHEIN 登录/Store 同步的开发者。无消费者的 URL 便利方法、地区兼容转发，以及从未接入真实请求的第二套超时管理仍被构造、保留测试或被旧 KEEP 表列举，造成当前 owner 和删除边界混淆。本轮交付一份完整退休闭包设计；不声称这些死代码正在阻塞用户采集，也不把退休或额外验收作为内部试用前置。

完整范围：1688 五个闲置 URL 便利方法；Amazon 两个闲置 URL 验证/规范化方法、九个地区/语言/邮编兼容方法及专属 languageMap；整文件 TimeoutManager 及 AmazonProcessor 的字段、构造、赋值和 Shutdown 专属尾部。共 **21 个生产函数/方法、1 个类型、1 个专属 map** 及接线，未来同一批覆盖 §5 的 **14 工件**，不按 helper、文件或测试机械拆批。无新生产 EXTRACT，不搬家保活旧接口。

Must：保留当前采集、原 Amazon 浏览器和有效调试、1688 实际规范化、ASIN/币种/marketplace、地区/邮编原值与有效校验、真实 PoolManager 的 deadline/取消/释放/重建、独立 SHEIN 登录和 Store 同步。不得把名字相似但有消费者的类型、方法、数据和安全断言一并删除。

Out of Scope：改变站点准入/SSRF/URL 安全合同、重新设计地区选择或超时恢复、为旧接口增加兼容层、修改现役配置/default/env/schema、迁移或清理任何真实资源、运行爬虫/provider/浏览器/安装器、改 workflow/完整 inventory/依赖/保护、增加试用前置。没有 UI/Figma 增量，Figma Authority **N/A**。

未来实施完全复用现役 crawler/alibaba1688、crawler/amazon 及其 browser/PoolManager 的当前行为 owner。没有当前消费者切换、新业务能力、公共领域合同、持久化事实、权限/租户、计费、事务、状态机、retry/UNKNOWN 或副作用边界变化，故采用 Reuse Existing Architecture，而非发明第二套超时/站点系统。三组均属既有采集能力内部的闲置合同退出；不新增跨系统一致性。若真实实现需要改变上述边界，停止该部分并报告，重新判断是否必须 Independent Architecture；本文不授权扩大实现。

## 2. 权威、基线与前批 KEEP 的准确替代

适用权威：用户本轮完整闭包设计决定、[Greenfield](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、[AGENTS.md](../../AGENTS.md)、[数据服务 v1](data-services-v1.md)及[独立 SHEIN 持久化边界](shein-worker-numeric-persistence-retirement-v1.md)。只有 EXTRACT / RETIRE，无旧数据兼容、fallback、双读写或第二事实源。Threat Model 沿用当前批准边界，不新增理论风险门禁或新 Accepted Risk。

调查基线 actual main **e29058c56df4294e1c612b3a705ab767bc333188**，tree **b836935f8fcb05856868f2384e3e43f6035aee0e**。#946 已正常合并，实际 merge0018de4849b7012fd4e757364bafb20f181e12d9 对应 push/main CI40173365776 attempt1 完整 SUCCESS；这是前批证据。其后 #945 仅 capture 扩展/文档与 sourcing fixture 增量，与本批候选源码/消费者不重叠。#931/#692 继续原 owner，不接管。

唯一 Writer 会话01a133ef-757c-73e3-b8f6-76958c642397，复用干净 managed 独立工作区，在上述实际 main 新建 codex/issue301-source-url-timeout-retirement-design。接单及前当前段存档见 [#301 / 6395255737](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6395255737)。不为了 main 无关变动持续同步；最终候选前按真实增量核对一次。

| 前批依据 | 本批待确认的准确替代 | 其它 KEEP 继续有效 |
| --- | --- | --- |
| [#893 日志门面](legacy-logger-facade-closure-retirement-boundary.md) | 其保留清单含 timeout_manager.go；仅在本矩阵内退休整个闲置 owner、1688 URL 便利及 Processor 专属接线 | 所有仍消费的 Logrus 注入、实际导航/失败/超时/关闭日志、当前采集与共享行为 |
| [#901 Amazon 来源适配层](legacy-amazon-source-adapter-closure-retirement-boundary.md) | 原 Processor/原浏览器/辅助层的广泛 KEEP，只在本批未消费 URL/地区便利和空超时接线内替代；不混淆已退休 integration 里的同名 DomainResolver | NewAmazonProcessor、单批/context、有效池大小/factory_test、真实浏览器/池/提取/debug/当前 acquisition |
| [#946 浏览器管理闭包](legacy-browser-management-tail-closure-retirement-boundary.md) | 其 Processor 其它字段/构造/Shutdown 与辅助层 KEEP，仅替代本次 timeoutManager 字段、局部构造、赋值和空表 CancelAll 段及 §5 闲置方法 | #946 的实际指纹、代理评分状态、质量重试、周期健康日志及全部其它源码/guard 结果 |
| [#822 Amazon 刊登闭包](legacy-amazon-listing-closure-retirement-boundary.md) | 原 crawler/amazon 来源能力的广泛保留，不再为本批无消费者的便利合同保活 | Amazon-as-source、当前 model/采集/有效纯规则，不按 Task/metrics/worker 等同名整退 |

上述是待确认提案；设计阶段不修改四旧文档。未来实施仅给这四份文件前置窄注，原完整 Git blob 逐字保留为后缀，不重写历史数字、勾选和批准。日期设计[1688 匿名速率/出口增补](2026-09-27-1688-anonymous-rate-and-egress-proxy-addendum.md) §7 已将 GetMobileURL 记为无调用方/无测试的 RETIRE；该记录仅佐证此方法，不恢复其它历史运行/代理要求，本轮不改该文档。

## 3. 实际消费者：contract → implementation → injection → consumer

依据是基线全追踪源的符号/调用、测试和非 Go 配置/文档核对，覆盖独立 debug 模块，区分 receiver 与同名函数。未运行业务构造器或真实资源。静态无调用只证明本仓库源码闭包，不证明任何既有进程/数据已经退出。

| 链路 | 实际消费者与断点 | 决定 |
| --- | --- | --- |
| 1688 URLHelper.GetMobileURL/GetDomain/GetPlatformName/IsProductPage/GetSupplierURL | 五方法无仓库当前调用/专属测试；IsProductPage 只是转发 IsValid1688URL | 五方法精确 RETIRE；不能禁掉 ProductChecker.IsProductPage |
| 1688 SingleProcessor → URLHelper.ValidateAndNormalizeURL → CleanURL/IsValid1688URL/ExtractProductID/BuildStandardURL | single_processor.go 当前真实调用，继续服务原处理器与当前采集入口 | 全部 KEEP，原 regex、logger、错误和清理/规范化语义不改 |
| Amazon URLHelper.NormalizeURL → IsValidAmazonURL | 只有自身旧单测/benchmark，验证方法另只有 NormalizeURL 调用；无真实请求使用二者 | 两方法连三项旧测试一起退，不用新 wrapper 或抽取维持旧规范化合同 |
| InstanceProcessor → AddLanguageParam/AddNoRedirectParam、GetCurrencyFromURL/GetMarketplaceFromURL/ExtractASINFromURL | instance_processor.go:142–143、301、338–345、414、440 有实际调用；marketplace 参与真实结果构造 | URLHelper/构造、全部保留方法、MarketplaceInfo 两字段/完整15项 map/getMarketplaceInfo、ASIN regex 和默认值全部 KEEP |
| DomainResolver.BuildAmazonProductURL/GetDefaultDomain → GetAmazonDomainByRegion；BuildAmazonProductURL → GetLanguageByRegion → languageMap | 两根入口无调用，map 只有此 getter 读取 | 四函数/方法及 languageMap 一起 RETIRE，不删除有其它读取者的 domainMap |
| ShouldUseDefaultZipcode → shouldUseDefaultZipcode；GetDomainMap/GetZipcodeMap/GetDefaultZipcode | wrapper 根入口与三兼容转发无调用/专属测试；纯规则仅为此 wrapper 使用 | 五函数/方法同退，不把旧闲置“US 不补邮编”规则另立当前 owner |
| InstanceProcessor → DomainResolver.ExtractRegionFromURL/GetZipcodeByRegion | instance_processor.go:236、245、252–253 的真实目标地区/邮编逻辑使用两 map | DomainResolver/构造、domainMap/zipcodeMap 全部原值和两方法 KEEP；当前目标上下文、US/JP 有效校验不改 |
| NewAmazonProcessor → NewTimeoutManager(5分钟) → ap.timeoutManager；Shutdown → CancelAll | 构造/赋值/Shutdown 是唯一接线；没有请求调用该类型 ProcessWithTimeout；CreateTimeoutContext 只被它自身调用，GetActiveCount 零调用 | 全文件、字段/构造/赋值/空表取消尾部整退，不留下空壳或修成新超时系统 |
| AmazonProcessor.Process/ProcessWithContext → processWithPoolManager → browser.PoolManager.ProcessWithTimeout(3分钟) → context.WithTimeout(ctx, timeout) → processProduct/acquire/InstanceProcessor | 当前请求使用的真实 deadline/取消路径；BatchProcessor 的 context 继续流向它 | 全部 KEEP，包括 parent ctx、defer cancel、超时实例关闭/异步重建、错误/释放、批量与关闭原顺序 |

发现的旧架构问题：闲置 TimeoutManager.CancelAll 在持有 mu 时调用 stored wrappedCancel，而 wrappedCancel 再次取得同一 mu，非空表情况下存在可由代码推导的自锁。当前仓库请求从未向这张表注册 context，因此**不是已复现的现役超时故障**，也不能据此宣称 Shutdown 已失效。根治本轮死链是退出无消费者的第二 owner；不修补并重新接入旧 manager，不改变真实 PoolManager 恢复协议，不为潜在问题搭建验证平台。

## 4. 配置、持久化与副作用 owner

| 边界 | 当前 owner / 本轮变化 | 不变量 |
| --- | --- | --- |
| URL/站点准入 | 当前 acquisition/dataacquisition 与源站校验，1688 SingleProcessor、Amazon InstanceProcessor | 不扩大输入/站点/重定向权限；1688 现役规范化、Amazon 当前 URL 安全/明确站点合同保持；退休的宽泛旧验证不是当前安全门禁 |
| marketplace/币种/ASIN/地区/邮编 | URLHelper、DomainResolver 的保留方法及 InstanceProcessor/ZipcodeValidator/resultValidator | 两种 MarketplaceInfo 字段及15项数据、domainMap/zipcodeMap 原值、ASIN 规则、fallback/default、US/JP 目标上下文全部不改，不新建配置 owner |
| deadline/取消/批量/资源生命周期 | 当前 PoolManager/BrowserPool/BatchProcessor、调用方 context 与 AmazonProcessor | 原3分钟、parent deadline、取消/有限重试、释放/同步异步重建、锁/关闭错误与 shutdownOnce/closed 不变；只删未使用 manager 的5分钟构造常量及空表处理 |
| TimeoutManager.activeContexts | 无持久化的进程内 map；当前请求不注册 | 整退不需要迁移/重放/双 owner；不据此操作任何既有进程或声称其 map 为空已做运行验收 |
| JSON/商品/lineage/身份/Store | 当前 acquisition/Sourcing/Catalog、独立 SHEIN 登录/Store 各 owner | 不删字段/JSON 零值/持久化记录，不改 schema/迁移、tenant/授权、claim/fence/UNKNOWN、幂等/事务/计量/同步 |

没有新表、SQL/Redis/文件读写、持久化 owner 变化或补偿协议，不需要 outbox/Saga/新队列。languageMap 仅静态私有数据，非用户 JSON/schema。不读取配置私密值、Profile、日志、token、缓存、已有数据或进程；所有运行验收 **NOT_RUN / NOT_AUTHORIZED**。

## 5. 完整删除 / 保留 / 抽取矩阵

未来实施 **14 工件** 一次收齐；当前设计阶段只改最后三份。计数辅助核对，不能替代真实保留路径验收。Legacy decision: **RETIRE**；Reusable behavior: 已由当前 owner 直接承担，不新增生产抽取；Current owner: §3/§4；Cutover/deletion condition: 用户确认本矩阵、复核消费者仍成立、按 §6/§7 完成必要自检和准确原 CI/最终复核。

### 5.1 一个整文件及四个生产混合工件

| 工件 | 精确删除 | 保留 |
| --- | --- | --- |
| `internal/crawler/amazon/timeout_manager.go` | 整文件：TimeoutManager、NewTimeoutManager、CreateTimeoutContext/CancelAll/GetActiveCount/ProcessWithTimeout，全部专属 imports/字段 | 不迁移其配置、日志、map 或方法到新 owner；不波及 browser.PoolManager |
| `internal/crawler/alibaba1688/url_helper.go` | GetMobileURL/GetDomain/GetPlatformName/URLHelper.IsProductPage/GetSupplierURL 五方法 | URLHelper/全部字段、构造及五个现役方法、imports/regex/logger/原行为 |
| `internal/crawler/amazon/url_helper.go` | IsValidAmazonURL/NormalizeURL 两方法 | 其它全部声明/函数/regex/import；MarketplaceInfo.Marketplace/Currency、完整 marketplaceInfoMap 和 getMarketplaceInfo 必须保留 |
| `internal/crawler/amazon/domain_resolver.go` | GetAmazonDomainByRegion/GetLanguageByRegion/ShouldUseDefaultZipcode/BuildAmazonProductURL/GetDomainMap/GetZipcodeMap/GetDefaultDomain/GetDefaultZipcode/shouldUseDefaultZipcode 九函数/方法及 languageMap | DomainResolver/NewDomainResolver、domainMap/zipcodeMap、GetZipcodeByRegion/ExtractRegionFromURL、strings 及全部原值/实现 |
| `internal/crawler/amazon/processor.go` | timeoutManager 字段、NewAmazonProcessor 内唯一 NewTimeoutManager 局部、返回结构赋值；Shutdown 中取消该 manager 的专属注释/if 块 | time import 仍使用；其它字段/构造/初始化/配置/日志、全部请求/批量/context、关闭 once/锁/closed 与 PoolManager→BrowserPool 关闭原顺序/行为 |

### 5.2 两个测试工件

| 工件 | 精确处置 |
| --- | --- |
| `internal/crawler/amazon/url_helper_test.go` | 只退 TestIsValidAmazonURL、TestNormalizeURL、BenchmarkNormalizeURL 及专属注释；TestNewURLHelper、TestGetCurrencyFromURL、TestGetMarketplaceFromURL、TestExtractASINFromURL、TestAddLanguageParam、TestMarketplaceInfoMap、BenchmarkGetCurrencyFromURL、BenchmarkExtractASINFromURL 的 fixture/断言/实现原字节保留，原 imports 保留 |
| `tests/listingkit_backend_retirement_test.go` | 仅新增一个 TestLegacySourceURLAndTimeoutClosureIsRetired，复用原 helper/AST/追踪文件枚举，不改 imports/helper/任何原函数。准确禁入/保留边界见 §6.1 |

### 5.3 四旧依据及三当前文档

| 工件 | 精确处置 |
| --- | --- |
| `docs/architecture/legacy-logger-facade-closure-retirement-boundary.md` | 前置 §2 中 timeout_manager.go/1688 URL/Processor 专属接线的窄 KEEP 替代；旧完整 Git blob 后缀不变 |
| `docs/architecture/legacy-amazon-source-adapter-closure-retirement-boundary.md` | 前置 §2 的原 crawler/amazon 闲置 URL/地区便利与超时接线替代；旧完整 Git blob 后缀不变 |
| `docs/architecture/legacy-browser-management-tail-closure-retirement-boundary.md` | 前置本次 Processor 字段/构造/Shutdown 与精确辅助方法例外；不改变 #946 已交付闭包/guard，旧完整 Git blob 后缀不变 |
| `docs/architecture/legacy-amazon-listing-closure-retirement-boundary.md` | 前置仅本批无消费者便利合同的来源 KEEP 替代，旧完整 Git blob 后缀不变 |
| `docs/architecture/legacy-source-url-timeout-closure-retirement-boundary.md` | 当前设计；未来实施仅前置准确状态/证据注记，确认的原全文保留为后缀 |
| `docs/architecture/README.md` | 当前设计入口；未来实施准确状态注记，原完整索引保留为后缀 |
| `docs/refactoring/legacy-register.md` | 记录完整拟退休/当前保留边界与授权阶段；原完整正文、历史/数字/勾选保留为后缀 |

不改 runtime-flow：其当前应用/原浏览器/PoolManager 说明仍成立，没有已退休 manager 的调用指引。1688 日期增补已记 GetMobileURL RETIRE，无需重复改写。发现其它真实冲突时报告精确引用，不自动扩大矩阵。

## 6. 实施与验证

设计仅本批三文档，验证链接、§1–§7/§6.1、矩阵、原索引/登记完整后缀、git diff --check 及适用原文档测试；不为纯文档制造业务 RED/构建/运行资源。本批设计准确 HEAD 的原 PR CI 和设计复核在 PR 记录，不复用 #946 CI 作为本批成功。

未来确认实施后同一 Writer/分支/PR 完整交付：先记录并读回 Reuse Existing Architecture / Ready 和用户冻结依据；增加 §6.1 原 guard 并确认目标 RED；按 §5 精确删除整个闲置闭包，保留原函数/数据/有效断言与文档后缀，确认 GREEN。不要只删上层方法留下唯一被调用的私有 helper、map、空字段或构造。

必要自检复用原 guard/导入与文档检查、保留 URL/helper/目标上下文测试和 browser 的无浏览器 mock 测试。保留原 TestInstanceProcessorRetriesOnProductQualityFailure、StopsAfterQualityRetryLimit、ResolveDefaultZipcodeForTargetContext，原 PoolManager 的成功/重建/释放/取消/等待/目标区域测试；不修改其断言，不跑真实 browser/provider 测试。根模块 cmd/current-application、cmd/shein-login-worker、cmd/1688-local-agent、cmd/1688-public-browser-collector 及独立 hack/debug 模块 test-amazon/test-1688 共六入口，仅按原命令编译到任务临时目录，不执行。无需临时 PostgreSQL/Redis/browser，也不搭建 runner/验收平台。不人工重复完整 CI 已覆盖且无变化的相同检查。

最终候选前核对实际 main 增量并仅解决真实组合风险；准确 HEAD 的完整原 PR CI 成功及最终交付检查分别记录。只有获准确 HEAD 合并授权后才能正常合并；其后核实实际 merge SHA 的 event=push/head_branch=main 完整 run completed/SUCCESS，不能用 PR CI 或单 gate 代替。不操作保护规则或管理员绕过。

### 6.1 原检查、测试与 inventory 精确矩阵

| 检查/工件 | 允许的准确变化 | 不得改变 |
| --- | --- | --- |
| 原 `tests/listingkit_backend_retirement_test.go` | 只加一个 §5.2 命名 guard：TimeoutManager 文件缺席；以精确包/receiver/声明/字段校验 §5.1 退休对象和无接线，并正向检查现役 URLHelper/DomainResolver 数据/函数、Processor 实际 PoolManager 调用/context 与关闭、PoolManager 的 context.WithTimeout/defer cancel/取消/重建路径和 ProductChecker.IsProductPage | 不新增原逻辑 skip/缺文件视成功/空目录占位；不全局禁用通用 ProcessWithTimeout/CancelAll/GetDomain/IsProductPage 等同名；所有原函数/helper/import/有效断言原样 |
| Amazon 原 URL 测试 | 仅 §5.2 的两旧 Test 和一个旧 Benchmark 退出 | 原八个保留函数和 fixture/断言/benchmark 全文、完整15项 map/两字段等有效行为不变 |
| #901/#946/其它原 guard、原 AST/导入/Code Health/文档检查 | 无转换；本次不改变它们要求的真实 Process/Batch/context/Shutdown/浏览器/指纹/评分/质量/日志 owner | 所有原检查不改；退休对象未被原检查显式保活的静态核对，不等于预授权未来漏项修订 |
| 完整 inventory | **零改**；不需要 CURRENT 校准，两个采集包的现役 Logger/依赖仍在 | CURRENT/历史/数字/计数/ceilings/算法/失败关闭原样；依赖、workflow、安全/必需检查与保护不变 |
| 文档状态与前批 KEEP | 仅 §5.3 四窄替代和三当前文档的准确阶段更新，旧完整 Git blob 后缀保持 | 不重写历史批准、数字或勾选，不把设计/静态/CI证据写成运行或产品验收 |

若真实 CI 发现本矩阵遗漏，先核对同根 sibling，报告根因与最小补丁，再取得用户准确例外决定；不得降低门禁或用 dummy/skip 保活。没有当前产品/安全根因，不另造规则或继续重构保留代码。

## 7. 评审、验收与停止点

1. **当前阶段**：仅设计。三文档/local check、准确设计 HEAD/原 CI 与适用设计复核结果留同一 PR；没有源码实施授权，未来状态维持 NOT_READY。设计结论不是用户实施决定。
2. **设计确认后**：用户确认本 §1–§7（含 §6.1）再记录/读回 Ready，单 Writer 完成14工件，不机械拆批。若实现改变授权/领域/持久化/恢复/副作用边界，按 AGENTS.md 判定真实 BLOCKER 并停止该部分；非 Blocker finding 先分类为 IMPLEMENTATION_TEST/BACKLOG 等，不能无限重开全局架构。
3. **复核**：设计复核与源码最终交付复核分开。按原自动 GitHub 机制检查真实 diff 和当前保留路径；源码形成后一次集中最终检查，修复只复核相关增量，有效未变证据明确复用。自动复核启动失败至多按现流程重试一次；**未经用户本轮授权不默认新增 AI 互验线程**，不把旧批授权扩展到本批。明确权限、持久化和副作用未变化，不为纯内部死链移除追加全局架构、安全平台或运行验收。
4. **交接**：给出准确候选 HEAD、当前消费者/保留证据、原 CI、复核/问题处置和 NOT_RUN 边界。实现者自检/CI/另一 AI review 都不是用户产品验收或可上线证明。原当前应用/采集入口及 SHEIN worker 启动方式不变；本轮不运行它们或交付新运行入口。
5. **权限停止**：实施、准确 HEAD 正常合并、部署、#301 关闭分别授权。#301 始终 OPEN，只更新当前段，历史从既定 #726 marker 起逐字保留，不改历史勾选/数字。保留独立 SHEIN 登录/Store 同步；不操作既有实例、数据、凭据或进程，不改 workflow/inventory，不增加内部试用前置。
