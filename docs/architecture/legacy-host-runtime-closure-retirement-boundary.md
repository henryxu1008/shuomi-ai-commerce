# #301 旧宿主 HTTP、会话与运行辅助层完整退休设计

Status: **CONFIRMED / Reuse Existing Architecture / Ready → In Progress**。用户已明确确认冻结设计 `a76db47d78037631573b42d81010ace8e23cb19d` 的 §1–§7（含 §6.1），授权在 #870 同批实施。#301/PR 准入已在生产修改前记录；下列原设计阶段时态保留为历史，实施权限以本状态及用户确认取代，删除/保留矩阵和停止条件不变。调查基线 main `245dcc60ccaf81b846603ba34f4cb39911018733`（#869 已合并），[准确 push/main CI39586724728](https://github.com/qq550723504/task-processor/actions/runs/39586724728) 已成功，仅作基线证据。#868 的源码退休与准确 main CI 已完成；#301 保持 OPEN，#29 不重开。本设计与后续实施沿一个 Writer、一个主要分支/PR 连续交付，不按包、文件或 helper 机械拆小批。准确设计 HEAD、CI、独立检查与用户确认留主要 PR。

## 1. 用户结果、范围与权限

使用者是维护当前应用、采集、Store 与独立 SHEIN worker 的维护者。结果是旧宿主已经失去装配的 HTTP/本地会话、组件运行辅助、自更新三组代码及其专属尾部一次退出；当前身份/HTTP、启动关闭、采集、Store 和登录同步继续由既有 owner 承担。当前没有证据证明核心业务正被这三组阻塞，不新增功能入口，也不通过维护增加内部试用前置。

三组完整范围：

1. `internal/infra/httpx` → `internal/infra/auth`：旧 HTTP 响应/解析/日志/CORS/健康/爬虫指标，以及自建 Session、文件 token、恢复和过期清理。
2. `internal/app/monitoring` → `internal/app/lifecycle`、`internal/core/errors`，加已无消费者的 `internal/app/ports`：旧组件监控、生命周期、错误/重试包装及采集/Task/Prompt 装配接口。
3. `internal/app/updater`：版本元数据、比较、下载/校验/重试、替换/重启、标记/错误日志和 adapter 的完整闭包。

**本轮仅授权只读调查、本文、Architecture README/Legacy Register 提案入口及 #301/PR 维护。生产源码、测试、schema、配置、workflow、inventory、依赖/锁、ceilings、保护零改。** 不手动调用旧会话/更新 constructor、应用、业务 CLI、旧脚本或任何 dry-run；不读既有 token、私密配置、更新产物、进程环境或运行报告，不操作实例、业务数据、凭据、源码外文件、Profile、数据库、队列、浏览器或 provider。不启动本地 PG/Redis/browser；原 CI 按既有规则运行已有隔离 fixture，不取得真实环境权限。设计确认后才实施；合并、部署、关单另行授权。

## 2. Design Basis、Authority 与边界

当前只读/文档工作：**Design Basis: N/A**。当前实施：**Reuse Existing Architecture / Ready → In Progress**，复用下列已批准当前架构，不新增或修改现役权限、租户、事实 owner、状态机、事务、恢复或副作用协议：

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)、[Issue 规则](../engineering/issue-driven-development.md)。未装配旧设计只作 RETIRE，保留能力继续原 owner；没有 Compatibility 类别、迁移、旧 ID 映射、wrapper、fallback、双读写或第二事实源。
- [当前身份/组织合同](auth-and-tenancy.md)及其引用的已批准 Identity/Organization 设计；当前 `authruntime/zitadel`、`authidentity`、`authz`、`workbenchcontext`、`app/httpapi/server_auth.go` 与 Auth.js handoff。旧 `infra/auth` 的 Java API token/本地 Session 不是这些 owner 的实现或持久化适配器。
- [HTTP 装配边界](httpapi-assembly-boundaries.md)、[App 装配边界](app-assembly-boundaries.md) 的有效装配原则；当前 `cmd/current-application` → `app/runtime/currentapplication`、`app/httpapi`、各域 HTTP 模块、`kernel/module` 与 closers。本文精确取代旧宿主组件/接口的待接线表述，不重设计当前装配。
- [#863 独立 worker 冻结设计](shein-worker-numeric-persistence-retirement-v1.md)：当前 Organization/Store、受权 CLI、会话/Store 审计、唯一执行 fence、UNKNOWN 不自动接管；[当前来源闭环](source-business-closure-v1.md)；[#868 值合同边界](legacy-numeric-value-contract-closure-retirement-boundary.md)。它们的所有权、持久化、授权和失败协议保持。

预计删除 31 个生产 Go 文件和 7 个专属测试，命中大于 30 文件提示线。现有架构完整覆盖：三组均无当前命令装配，移除的是无消费者旧实现和旧目标，未改变共享配置 schema、当前可观察领域/授权/持久化合同，也不引入替代框架或跨系统机制，因此不为文件数新建 Independent Architecture 或拆 PR 规避门槛。若刷新发现当前消费者或确需改变上述当前边界，停止对应正式实现并报告；现有架构不能覆盖时升级 Independent Architecture，不能在实现中发明新协议。

Must：完整退出三组及专属尾部；当前身份、权限、租户/组织、采集、图像、平台 API/规则、Store、独立 SHEIN 登录/同步、当前日志/监控/启动关闭和有效共享合同不变；workflow/inventory 与原必需/安全检查不弱化；不接触物理资源，不新增试用前置。Should：复用原 guard、标准库与已有当前 owner，不制造替代 Session、HTTP facade、自更新器、错误/生命周期库或新验收工具。Out of Scope：业务扩展、现役 IAM/配置/日志/重试重构、源码外资产处理、新迁移/codec/兼容、真实运行/发布验收。Accepted Risks：无新增；既有物理文件、历史运行态与人工使用情况保持 UNKNOWN，不把未访问写成不存在。

Threat Model 只覆盖本批误退当前身份/worker/HTTP/共享基础层、把源码退休扩成文件/凭据/进程操作、保留旧放行认证或自更新作为未来 owner、以及删 guard 留旧入口。旧 `RequireAuth` 直接调用 next、旧 health/nil-ready 总成功是未装配源码事实，不据此宣称当前应用存在认证绕过或已完成真实安全审计。UI/IA/Figma 增量 N/A，没有新增页面、CLI 产品动作或用户操作步骤。

## 3. 实际消费者与 contract → implementation → injection → consumer

在准确基线使用 tracked imports 与 `go list -json ./internal/... ./cmd/...` 的生产/测试 imports 交叉核对；31 个 cmd 和额外原生身份 CLI `internal/zitadelprovision/cmd` 的依赖均不包含三组。另检索全部 tracked Go（含 scripts/hack/tools 等直接引用）及专属文档/guard；没有执行产品命令、constructor、旧测试的副作用分支或访问物理状态。

| 旧入口/包 | 实际非测试消费者 | 测试/装配结论 |
| --- | --- | --- |
| `infra/httpx`：8 生产 + 1 测试 | 无包 importer；自身 auth.go 导入 infra/auth | 只有自身 metrics 测试；原爬虫 HTTP 服务/handler 已在前批退出；没有当前 route/module 注入 |
| `infra/auth`：4 + 1 | 仅 infra/httpx 引用 SessionManager 类型 | 自身禁止已退 client-credentials flow 的 guard；NewSessionManager/NewFileTokenStore 没有包外构造调用；不是当前 Auth.js/native JWT 会话 |
| `app/monitoring`：5 + 0 | 无 importer | 只在内部嵌入 lifecycle.BaseComponent、调用 core/errors 和 timeout；旧存在性 guard 不等于运行消费者 |
| `app/lifecycle`：3 + 1 | 仅 app/monitoring | 自身排序/循环检测测试；当前 kernel/module 与 application/worker closers 不导入它 |
| `core/errors`：3 + 3 | 仅 app/monitoring | examples_test 的自包外测试 import 不算生产消费者；RetryConfig 来自现有 core/config，但不反向赋予旧错误包装当前职责 |
| `app/ports`：1 + 0 | 无 importer | CrawlSource、ListingRuntimeHealthValidator、TaskPublisher、数字 TaskReporter、PromptRegistry 无实际接线；当前源 port/Prompt owner 有自己的合同 |
| `app/updater`：7 + 1 | 无 importer | constructor 只在本包内相互调用；adapter 测试是旧调用流程证据，没有当前命令/HTTP/worker 装配或独立 CLI |

有效当前路径保持：

| 当前合同 | 实现/注入/消费者与保留结果 |
| --- | --- |
| 认证/组织授权 | Auth.js handoff → 现有 app/httpapi route auth → authruntime/zitadel、authidentity/authz/workbenchcontext → 各当前业务域；原 route policy、拒绝/撤权/组织隔离和凭据输入不改 |
| 当前程序与模块生命周期 | cmd/current-application → app/runtime/currentapplication/current HTTP composition 与 kernel/module/既有 closers；原启动、取消、shutdown-file 和资源关闭保留，不接旧 lifecycle.Manager |
| 独立 SHEIN 登录/Store 同步 | cmd/shein-login-worker → app/runtime/sheinloginworker → sheinlogin/bootstrap、当前授权/SessionTransport/Store repo 与同步；原 healthHandler、DB/Redis/browser/session/fence/UNKNOWN 和 CLI 合同不改 |
| 采集/平台/价格/属性 | 当前 1688/Amazon Processor/source adapters、源保存/读取/lineage、平台 API/DTO、规则、价格与图像 owner；MappingInfo/两种属性、四来源 JSON 零值槽位及原规范化/hash/拒绝合同完整保留 |
| 日志/监控/Retry/共享值 | platform/logging/metrics/observability、仍有消费者的 core/logger/config、pkg/timeout/httpclient、shared/strx、kernel/module 等保持。当前 HTTP health/ready 与 storage/WeCom 告警不依赖旧 httpx/monitoring，不改部署声明 |

“命令未装配”单独不是全仓删除理由；本矩阵同时依据已退宿主语义、实际消费者退出、当前职责已有 owner 与本次用户设计范围。其它未装配但已确认有效的库/工具/API/测试 seam 不纳入。

## 4. 持久化、副作用及失败 owner

| 边界 | 旧实现能做的事与 owner | 本批处置 / 当前 owner |
| --- | --- | --- |
| 本地 Session | SessionManager 内存 map、随机 token/30天 expiry、AccessToken/RefreshToken/TenantID；构造时恢复并启动清理 goroutine | 旧本地认证协议整体 RETIRE，无新 Session/token owner；当前 native/JWT/组织授权保持 |
| 文件 token | FileTokenStore 的调用方 filePath；Save 建目录/0600 JSON 写入；Load 读取并可删除过期文件；Revoke/Delete 删除持久化 token，Save 失败仍继续原会话创建 | 只删除声明/读写代码；不寻找、打开、stat、复制、撤销、重写或清理任何既有 token 文件；物理状态/归属 UNKNOWN，不迁移给当前身份系统 |
| 旧 HTTP | AuthMiddleware 放行、响应/请求解析、CORS/日志、总成功 health/nil-ready、crawler_ 指标文本 | 均无现役 route 注入，整体 RETIRE；当前 HTTP 授权/拒绝、健康与指标沿各现有 owner，无重定向/兼容路由 |
| 组件监控/生命周期 | 内存运行标记/依赖排序、指标 map、进程起始时间与周期健康/metrics goroutine；core/errors 的 Task 错误码、panic/retry/包装 | 旧进程组件协议整体 RETIRE；没有 DB/队列事实需要接管。当前 registry/closers、通用 Retry/panic/timeout/观测保持，不移植旧状态到新 manager |
| 更新远端读取 | VersionManager GET 调用方 updateURL；版本比较；FileDownloader GET/重试/SHA-256校验 | 整体 RETIRE，不访问 URL，不用新的开源 adapter 包装旧流程；当前版本/交付/安装 owner 和流程不改 |
| 更新文件 | 临时目录 task-processor-new.exe、当前 executable 的 .old、Windows *.exe.new、temp_update.ps1、.update-marker、update-error.log；可复制/覆盖/移动/删除 | 源码及内联脚本一起 RETIRE；这些字符串不是本批文件操作清单，不检查或处理已有产物，也不清理用户磁盘 |
| 重启/进程副作用 | exec/cmd/start、PowerShell 延迟替换、taskkill、os.Exit，Linux/Mac 启动新程序后退出；旧 adapter 按下载→MarkApplied→Restart 编排 | 全部与 updater 同退；设计/验证不执行 Restart/Replace/Mark/constructor。没有新发布器/自动恢复/补偿，不操作当前进程 |
| 现役持久化/外部副作用 | 原身份/IAM、SRC/Catalog、Store/worker 的 DB/Redis/Profile/会话/平台及计费 owner | 源码、schema、权限、事务和副作用协议均不变；没有新的跨库事务、outbox/Saga 或恢复事实 |

这三组没有当前命令消费者，**不发生持久化 owner 转移**。旧 token、更新标记或备份文件不成为当前系统输入；也不因源码删除被宣称失效、已归档或可清理。若发现当前外部可观察契约确实依赖旧流程，提交具体证据并停止对应实施，不能预建兼容例外。

失败/响应丢失/重试/重启/取消/并发/撤权/资源上限由未修改的当前 owner 继续处理。旧 updater 的分步文件操作、MarkApplied/Restart 失败、旧会话持久化/内存不一致只作旧代码事实，不新建恢复协议或运行故障矩阵修复它。任何验证都不触发这些副作用；真实文件/进程行为 NOT_RUN。

## 5. 完整删除 / 保留 / 抽取矩阵

### 5.1 RETIRE：38 个准确 Go 文件，三个根一起交付

以下仅是用户确认后的源码删除提案，当前不应用：

| 根 | 完整文件（含专属测试） |
| --- | --- |
| internal/infra/httpx | auth.go、cors.go、health.go、logging.go、metrics.go、params.go、parser.go、response.go、metrics_test.go |
| internal/infra/auth | cleanup.go、manager.go、session.go、token_store.go、client_credentials_boundary_test.go |
| internal/app/monitoring | collector.go、health_checker.go、metric_operations.go、process_info.go、types.go |
| internal/app/lifecycle | component.go、interfaces.go、manager.go、manager_test.go |
| internal/core/errors | errors.go、helpers.go、productjson_errors.go、errors_test.go、helpers_test.go、examples_test.go |
| internal/app/ports | platform.go（全部五个孤立接口） |
| internal/app/updater | autoupdate_adapter.go、file_downloader.go、file_manager.go、models.go、update_manager.go、updater.go、version_manager.go、autoupdate_adapter_test.go |

三组闭包包含上述全部类型/方法/字段/定时循环、专属 DTO/错误码/接口、内联 PowerShell/重启编排及专属实现测试。不是只删 NewUpdater/SessionManager 留 helper 或 adapter；不复制到新包、不建空目录/空壳通过 guard，不新增替代 constructor/route/manager。

### 5.2 KEEP / EXTRACT 结论

**本批无新增 EXTRACT。** 没有当前消费者需要这些封装或旧返回形状；一般“可复用”不等于当前需求，不为保存未消费算法新建 shared/errors、HTTP facade、lifecycle 或自更新库。旧依赖中的标准库 errors/context/net/http、已存在的当前重试/生命周期/身份能力可直接继续使用。

KEEP：所有当前权限/身份、Auth.js/native CLI、HTTP/health/ready/module/closers、采集、图像、AI/prompt、平台 API/规则、PriceManager/PriceCalculator、两属性/MappingInfo、SourceKey/fingerprint/编码/hash、Store 与独立登录/同步；core/logger/config（含通用 Retry）、platform logging/metrics/observability/workerpool/queue、pkg timeout/httpclient/recovery、shared helpers 均零改。不因失去 updater 的 strx.Contains 使用就扩删 shared/strx；不改同名当前错误/Session/lifecycle/HTTP 组件。没有新的业务源码或抽取文件。

直接文档/原 guard 随 §6.1 校准才是闭包完成；它们的历史执行、失败、计数、已确认当前合同原样保留。现有 updater 第三方候选/延期替换要求仅对这个已无消费者的旧宿主目标被本设计精确替代，**不是引入第三方更新器的授权**。

```text
Legacy decision: RETIRE
Reusable behavior: 本批无新抽取需求；标准库及现有有效身份/HTTP/生命周期/Retry/观测/平台规则继续原 owner
Current owner: app/runtime/currentapplication、app/httpapi、authruntime/authidentity/authz/workbenchcontext、kernel/module、platform 与独立 SHEIN/Store 原 owner
Cutover/deletion condition: 用户确认完整矩阵及 §6.1 后，三组无消费者源码和专属尾部同批缺席，原当前合同/必需检查保持；无兼容、迁移、替代 owner 或真实状态操作
```

## 6. 必要验证与交付

设计阶段本地只作静态源码/元数据/引用和适用原文档检查，不为文档制造 RED，不直接运行旧包业务流程/会话或更新 constructor；原 CI 的已有隔离测试按既有 workflow 执行，不扩成真实调用。准确基线 main CI 只是基线；本批设计 CI/复核不能替代后续实现证据。

未来用户确认后的必要验证：

1. 沿现有 retirement guard 增加七个准确根缺席/禁入，先 RED 再删除后 GREEN；复用原 AST/import parser，覆盖当前命令/HTTP/worker/身份 CLI，不新增扫描器/runner。
2. 原当前 HTTP/身份/组织/来源/localagent、worker/Store、配置/Retry/module/共享基础层的相关已有测试保持；只运行与本批实际风险相关的必要检查，完整原 CI 留稳定候选。31 个 cmd 与原生身份 CLI 的真实依赖不含退出包；current-application/shein-login-worker 原构建保持。
3. 核对 production DELETE 仅 §5.1，当前能力/共享合同/配置/schema、workflow/inventory/依赖/锁/ceilings/保护零差异；既有副作用文件/进程不检查、不执行，不为证明源码删除而启动 PG/Redis/browser 或部署。
4. 交出准确实现 HEAD、原 CI、一次完整候选的最终独立检查及已知限制。源码/静态和构建证明不等于真实运行/用户验收；不增加试用步骤、专项环境或新门禁。

### 6.1 原检查、文档、inventory 与 workflow 的精确处置

| 对象 | 用户确认后同批最小校准 | 必须保持 |
| --- | --- | --- |
| §5.1 七个专属测试 | 随所属旧实现退出。旧 updater adapter 流程/失败、lifecycle 排序/循环、core/errors 行为、旧 crawler metrics 和 auth 包内禁入测试不被当成当前业务要求 | 当前 kernel/module/启动关闭、授权/tenant/UNKNOWN、健康/Retry/平台/worker等原有效测试；旧 client-credentials 类型禁入在仓库根 absence/原禁入 guard 继续覆盖，不因删测试允许复活 |
| tests/listingkit_backend_retirement_test.go | 增加七根准确 absence/import 禁入，保留原爬虫 handler 缺席及八命令真实依赖、worker 隔离；无新工具 | 所有无关旧 absence、两属性/MappingInfo/SRC hash/原当前合同与安全约束 |
| tests/legacy_hard_cut_test.go | TestLegacy1688TaskHandoffAndRouteStayRetired 中旧 httpx 扫描根改为根缺席检查；1688 当前源码扫描/禁入保持，不能 missing-skip 或吞读取错误 | sourceaccount/旧 handler/profile/queue 缺席和引入禁令、AST 解析失败/编码 import 检测不变 |
| tests/target_architecture_phase2_test.go | TestLegacyMonitoringPathIsRetired 的 app/monitoring 必须存在改为 infra/monitoring 和 app/monitoring 均缺席/禁入；仅精确切换旧目标 | 原解析/跨平台/SDK/领域边界，platform/observability 当前 guard 与所有其它原检查不变 |
| tests/import_boundaries_test.go | 原 global ports 根缺席继续保留；引导“use app/ports”的两条诊断改为“使用当前 owner 合同，两个旧根都不可复活”，同批补 app/ports 缺席/禁入 | 原索引/读取失败、import decoding、其它边界和当前 native/授权/商业守卫不变 |
| tests/architecture_docs_test.go 与当前 guard 索引 | 如实际引用必须校准，仅更新上述已退出准确目标/guard含义；旧目录词可作为历史保留，不为原 phrase 检查造空包或重写历史 | Architecture README/Review Checklist 一致性、原历史 closure counts、其它架构/源码检查与 ratchet 不变 |
| 当前文档与直接说明 | Architecture README/Legacy Register/module-target-mapping/code-health 顶注；project-boundaries、app-assembly-boundaries、httpapi-assembly-boundaries 的准确旧目标加窄失效/当前 owner 指引；updater-third-party-validation 与 open-source-replacement-evaluation 对旧 updater 的待替换要求加 superseded 指引；code-health 的 Amazon source adapter 行明确 httpx 说明已过时，仅去已退测试目标，保留当前 source adapter | 原产品/Figma、身份/worker/Store/平台合同和所有 dated 历史正文/执行/失败/计数；不把旧 Task 示例改造成新需求或进行全仓文档重写 |
| 当前错误处理指南与同根因引用 | docs/development/error-handling-guide.md 整份旧 core/errors 指南标记 RETIRED/历史，撤销其当前导入/迁移指导及 errors.go、examples_test.go、errors_test.go 三个“当前参考”；原 Task/错误码示例只保留为明确历史。amazon-crawler-commercialization-execution-plan.md 对旧 httpx/health.go 的准确说明补窄失效指引，沿原 #867 历史定位 | 不创建新的统一错误框架/迁移手册，不改变现役错误/拒绝合同，不按旧 Amazon 商业化计划启动新任务；历史示例不冒作当前可编译入口 |
| phase2-runtime-inventory.md / ceilings | **零改**。已核对 CURRENT 消费者表没有因本轮删除而退出的受管 provider→consumer 行；历史 debtor 行、旧迁移计划/计数不改 | 若实施原 CI 证明存在冲突，先给出准确行与真实消费者证据、报告用户，不自行删表/改计数或改 guard 绕过 |
| 所有 workflow / CI、依赖与保护 | **零改**，沿原分类/触发执行；不新增 missing-skip 或把安全检查改成成功空壳 | 两套 workflow、必需/安全/PG/资金/商业/发布、当前 listingadmin test-only race 与当前入口构建，依赖锁/ceilings/保护全部保持 |

本轮设计提交只包含本文与索引/登记两处提案入口；以上源码、测试和其它文档校准均未应用。它们只有在用户确认后才属于同一主要 PR 的实施范围。

已针对七个准确包扫描全部 tracked docs 的同根因引用。phase1-summary/phase1-acceptance-report、dated superpowers plans/specs 和 test-baseline.txt 属历史证据，保持原文/计数，不恢复当前权威；phase2-runtime-inventory.md 连历史说明也零改。当前开发指南遗漏已在上表明确纳入本闭包，不能只删源码后留作推荐导入路径。

## 7. 设计确认、复核与停止点

1. 本轮交付完整三组设计与实际证据，集中一次独立最终设计检查核对 §2/§3/§4/§5/§6.1，不默认增加 AI 互验线程、不重复同 SHA Running 请求，不为已存在且未改的当前架构重新做全局审核。
2. 用户确认 §1–§7（含 §6.1）后，在 #301/同一 PR 记录 **Reuse Existing Architecture / Ready**、唯一 Writer 与冻结设计 HEAD，再开始正式生产删除。设计提交/CI/Reviewer 不构成实施、合并、部署或真实操作授权。
3. 确认后同批完成全部删除、专属测试/文档尾部、必要自检/提交/推送及范围内 review/CI 修复；一个主要 PR，不按七个包或测试拆批。只有真实当前 Must 的 BLOCKER 才重新打开冻结设计；普通 hardening/实现测试不增加试用前置。
4. 若出现真实当前消费者/外部可观察契约、现役权限/持久化/副作用/恢复 owner 变化、需要新协议/迁移/兼容、inventory/workflow 必须修改或越出 §5，停止对应部分并报告具体 Must/阻塞层级与最小差异。不能按猜测保留旧适配层或降低检查解围。
5. 完整实施候选一次最终独立检查，修复只复核对应增量；交出最终准确 HEAD/原 CI/复核供用户单独授权合并，合并后核对真实 merge SHA 的 push/main CI。#301 始终 OPEN；原未完成运行/用户验收仍按原范围 NOT_RUN，不由源码退休自动取消、签发或关单。
