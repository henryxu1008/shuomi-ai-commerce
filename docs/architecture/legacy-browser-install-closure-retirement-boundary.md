> 当前用户决定 / #301 / #1030：用户已确认冻结设计 f5e560cd7 的 §1–§7（含 §4.2/§6.1），IMPLEMENTATION_READY；同一 #1030 按36路径整批实施，不机械拆批。退出旧预装基础镜像、两个仓库安装CLI及browserinstall/httpclient/timeout专属闭包；官方SDK标准安装、显式指纹浏览器/匹配Profile、当前能力与Store观察保持。无新安装器、alias或试用前置；workflow/所有inventory/既有实例数据凭据缓存进程零改，#301 OPEN。源码自检、原CI及最终复核另记同一PR，不代表产品或运行验收，合并另授。生产修改前授权与准入已[写回并读回](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1030#issuecomment-6540889996)。以下[冻结设计/原导航](legacy-browser-install-closure-retirement-boundary.md)及其原阶段性措辞完整保留，当前授权状态以本条及PR为准。

# 旧浏览器预装与自下载整链退休设计

Refs #301。Design Basis: **Independent Architecture**。当前 **DESIGN_ONLY / SOURCE_NOT_STARTED / NOT_READY**；适用独立设计复核、用户确认和准入读回完成前，不修改生产代码。一个主要 PR 整批收齐，不按内部模块拆批。

## §1 用户结果与设计依据

维护者以后只面对两条已有的有效浏览器路径：标准镜像继续使用当前官方 Playwright SDK 安装；本地 1688 批量采集继续显式接收指纹浏览器路径和匹配的 Profile。退出已弃用的预装基础镜像，以及只为它服务的两个仓库安装 CLI、自下载包和专属 HTTP/timeout 尾部。不会再把无人消费的旧基础镜像称为当前安装入口。

这项设计需要用户确认的产品取舍是：**取消本仓库旧 CLI 自动解析版本、下载指纹浏览器及旧镜像预装的交付承诺，保留指纹浏览器本身及现有显式输入要求。** 普通 Chromium 不被当作指纹浏览器的等价替代。真实外部人工调用情况未调查，为 UNKNOWN；本设计不宣称旧安装入口从未被人工使用。

依据：[Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)及用户本轮授权。调查基线 `main=97edf80a2851e3c319b3e01f9fd33a627cc4e42d`。无页面、IA 或交互变更，Figma Authority **N/A**。当前仅编写本设计与两份导航，非正式源码实施。

Legacy decision: **RETIRE**。Reusable behavior / Current owner：官方 SDK 安装、当前批量采集的显式浏览器输入、平台日志、当前 HTTP 客户端/Retry 都已有独立有效 owner，全部保持。没有需要从旧下载器抽取的新行为，不搬家或包装死代码。Cutover/deletion condition：用户确认 §1–§7、适用准入完成后，在同一 PR 按 §5/§6.1 整批删除并完成原 CI/最终交付检查。

Out of Scope：新安装器或版本选择平台、浏览器内核发行/下载/升级、实际制品和缓存清理、Profile 复用或迁移、SDK/依赖升级、当前抓取算法和登录授权调整、发布恢复、workflow/inventory/保护规则变更。当前采集、图像、身份权限及 Store 观察同步保持；已退休的独立 SHEIN 登录与身份回写不恢复。#301 保持 OPEN，无新增内部试用前置。

## §2 前批 KEEP 的窄替代

下列替代在用户确认本设计后生效。旧记录仍是历史证据，不批量撤销其它 KEEP。

| 原依据 | 仅退出的保留含义 | 继续保留 |
| --- | --- | --- |
| [#1010 调试树](legacy-restored-debug-closure-retirement-boundary.md) §4.1/§5/§6.1 | 因安装 CLI 尚有 Docker caller 而 EXTRACT 到 browserinstall 的下载闭包、CLI 与其 KEEP 检查 | 当前 Collector 配置、tools lint 固定用途、acquisition/extension/batchcapture、显式配置加载及其它当前 owner |
| [#1026 远程镜像](legacy-remote-debug-image-closure-retirement-boundary.md) §3.2/§5，以及承接其注记的 [#897 执行骨架](legacy-runtime-skeleton-config-closure-retirement-boundary.md) | “全部其它 Dockerfile/安装器保留”中的本批一个弃用基础镜像、两个 CLI 及专属依赖 | account-compose/browser-collector/UI Dockerfile、标准构建脚本与官方 SDK 安装，全部其它当前能力 |
| [#914 timeout](legacy-idle-support-contract-closure-retirement-boundary.md)、[#937 HTTP 门面](legacy-idle-support-facade-closure-retirement-boundary.md)、[#1017 HTTP/auth](legacy-http-auth-support-closure-retirement-boundary.md) | 两个下载常量、httpclient.Config/Client/New/Do 的旧 KEEP | 当前 Context deadline/取消、真实 HTTP/API/SDK 传输、有效配置、身份权限与租户隔离 |
| [#893 logger](legacy-logger-facade-closure-retirement-boundary.md)、[#939 配置尾部](legacy-global-config-logging-tail-closure-retirement-boundary.md)、[#969 支撑层](legacy-support-control-tail-closure-retirement-boundary.md) | 只为下载链保留的 logger port/注入、HTTP 字段/调用检查及 CLI flags | appenv.ComponentLoggers、platform/logging 真实锁/输出、Goose migration 的会话锁/Up 安全断言及全部真实消费者 |
| [#870 宿主](legacy-host-runtime-closure-retirement-boundary.md)、[#915 helper](legacy-host-image-helper-closure-retirement-boundary.md)、[#948 crawler 门面](legacy-crawler-facade-diagnostics-closure-retirement-boundary.md) | 广泛共享 helper KEEP 中仅本批 httpclient/timeout 下载专属闭包 | 图像解码注册、当前 HTTP/配置/AI、有效算法与当前采集；不因同名 timeout/retry 删除其它 owner |
| [1688 本地批量设计](../superpowers/specs/2026-09-21-1688-batch-local-agent-design.md) §3.1 表的 fingerprint-chromium 安装“直接复用”一行 | 不再复用 cmd/fingerprint-browser-installer；本仓库不承担这条自下载实现 | 指纹内核、相同二进制版本创建的 Profile、现有显式 --browser、路线 B、scope/Consent、队列/UNKNOWN/崩溃安全、验证码/登录人工边界 |

最后一项是有效设计中的准确替代，不能把整份批量设计标成历史，也不能用“零 Go caller”代替用户对安装能力的取舍。

## §3 实际消费者与当前 owner

只读检查 Git 追踪的源码、测试、Docker、脚本、Makefile、两套 workflow 与直接文档；未运行安装器、浏览器、服务或网络下载。

| contract → implementation → injection → consumer | 证据与决定 |
| --- | --- |
| 弃用 runtime-base → 两个仓库 CLI → browserinstall → pkg/httpclient / pkg/timeout | Dockerfile 的 Deprecated 注释明确仅保留人工预装试验；它是两个 CLI 唯一非测试、非文档仓库调用方。当前标准 Dockerfile 无 FROM 此基础镜像，workflow/Makefile 无这些 CLI 目标。整链 RETIRE |
| fingerprint CLI → version/dir/timeout env → NewChromeDownloaderWithTimeout → CheckAndDownload | CLI 是 browserinstall 唯一生产 import；该包是 httpclient 唯一生产 import，timeout 两常量仅为该 CLI/下载器及测试使用。logger port 与四个专属测试文件同退，无新的共享消费者 |
| 标准镜像 → Dockerfile.account-compose → go.mod 固定的官方 SDK CLI → Chromium/driver | scripts/build-listingkit-images.ps1 使用明确 Git archive 和标准目标。playwright-installer-builder 是有效 SDK 构建 stage 名，不是旧 cmd/playwright-installer；脚本、原 mock 测试、Dockerfile、SDK pin/路径保持零改 |
| 公共采集 → Dockerfile.browser-collector → 官方 SDK CLI → collector-chromium | 当前公共采集程序与启动脚本直接使用既有 driver/browser 路径，无旧自下载依赖；保持零改 |
| 本地批量 → cmd/1688-batch-import --browser/--profile → batchcapture.DriverOptions → LaunchPersistentContext | 已有必填显式路径、文件存在检查和 ExecutablePath 传递；代码要求指纹浏览器和匹配版本的 Profile，从不调用旧 CLI/下载器。程序、脚本、扩展、scope/队列/身份/原测试零改 |
| 当前日志、Retry、API、Store | appenv → platform/logging、当前集成/SDK/标准 net/http、shared/resilience、storeobservations 仍各归原 owner；不依赖本批三个旧包，不重写为新 facade |

### §3.1 制品准备边界

指纹浏览器的二进制构建/发行/版本完整性属于其发行方及已有分发 owner；本地执行者只消费已准备的合法二进制和匹配 Profile，这是现有必填输入，不新增操作步骤。本批不指定新的下载源、下载工具或替换内核，也不检查用户机器上是否已经准备好。

[内核发行目录](browser-kernel-release-catalog-v1.md)及其[操作边界](../operations/browser-kernel-catalog.md)只说明已存在的独立元数据 owner：后端不下载制品，原生安装归桌面 owner。它不是本批的安装器替代，不将其接入本地 batchcapture，不要求启用目录、准备数据库或完成跨项目交付作为试用前置；不接管该 owner。其真实发行、下载和安装不因本设计而变成 PASS。

## §4 权限、持久化与副作用

本批不新增状态机、表、迁移、事务、身份或领域合同。当前业务事实、组织隔离、Consent、JSON/hash/绑定版本、幂等、Retry/UNKNOWN/恢复、PoolManager deadline/取消及 Store 观察均由原 owner 继续负责。

### §4.1 Threat Model / Must

- 当前入口不能转为调用旧包、自动下载、自动 latest、普通 Chromium 回退、旧镜像回退或新的包级日志门面。
- 不误删官方 SDK installer stage、依赖 pin、标准构建/启动路径、有效指纹浏览器要求及 Profile 版本要求；原真实安全断言保留。
- 不读取或打印既有凭据、Cookie、私密配置、目录内容、安装缓存或 Profile，不启动/终止进程，不运行下载器/解压命令/镜像构建。
- 当前目录/Store/身份/API 权限、数据库角色/迁移和其它下载实现均零改。未授权真实验收是 NOT_RUN，外部旧人工消费者是 UNKNOWN，不能将取消的旧入口标成已验证等价能力。

### §4.2 文件、输入、JSON/hash 与失败责任

| 边界 | 本批取舍 |
| --- | --- |
| 旧 PLAYWRIGHT_BROWSERS 与 FINGERPRINT_CHROME_VERSION/DIR/DOWNLOAD_TIMEOUT | 随旧 CLI/镜像退出；不改全局环境读取、不增加拒绝前缀/新默认/alias，不映射到当前 --browser 或 PLAYWRIGHT_DRIVER_PATH/BROWSERS_PATH |
| 下载器的版本解析、GitHub 请求、缓存查找、写包、解压/tar、chmod/查找可执行文件 | 源级能力整退；不抽取相同实现，不留下可执行 wrapper。已有文件/缓存/镜像/日志全部不读取、不重写、不删除 |
| 当前显式浏览器/Profile | 原验证、路径、版本匹配和真实启动/关闭语义不变；本轮只读源码，不执行它们，不承接历史登录态 |
| 现有制品/外部人工使用 | 不保证旧 CLI/镜像继续可构建；仓库不会通过兼容层续供。用户设计确认仅决定源码边界，不授权删除或迁移这些外部资源 |
| 当前 JSON/hash/绑定版本/持久化 | 没有需转换的业务 JSON、hash、版本或保存载荷；不改 schema/Goose 历史、运行记录、角色、数据/账务/Store，不引入新事实源 |
| 失败/响应丢失/重试/重启/取消/并发 | 旧下载链退出后不再有应用侧下载动作或恢复责任；当前采集、安装、API、queue/UNKNOWN/cleanup 仍按原 owner，不新增 retry、恢复器或后台触发 |

## §5 完整删除/保留矩阵

未来实施共 **36 路径**：11 源/专属测试/Docker 工件、2 混合检查文件、23 文档（含本设计/两导航）。当前设计提交只改后三者，不提前修改其余文件。若发现新的消费者/冻结 KEEP/检查变更，先报告准确提案，不以“完整闭包”扩大授权。

### §5.1 RETIRE：11 工件

```text
deployments/docker/Dockerfile.product-listing-api-runtime-base
cmd/playwright-installer/main.go
cmd/playwright-installer/main_test.go
cmd/fingerprint-browser-installer/main.go
cmd/fingerprint-browser-installer/main_test.go
internal/integration/browserinstall/chrome_downloader.go
internal/integration/browserinstall/chrome_downloader_test.go
internal/integration/browserinstall/logger_port.go
internal/integration/browserinstall/logger_port_test.go
internal/pkg/httpclient/httpclient.go
internal/pkg/timeout/timeout.go
```

删除整个两个 CLI、browserinstall/httpclient/timeout 五个根及上述 Dockerfile。专属 env 解析、超时/重试/日志注入测试随已取消能力退出，不移成另一包的保活测试。无生产 EXTRACT，无兼容 alias，无新下载库或安装工具。

### §5.2 检查与文档尾部：25 路径

- `tests/listingkit_backend_retirement_test.go`、`tests/repository_structure_test.go`：仅 §6.1 精确转换。
- 本设计、`docs/architecture/README.md`、`docs/refactoring/legacy-register.md`：设计与当前导航，旧正文/数字/勾选保留为完整后缀；准确 HEAD/CI/复核留 PR，不制造滚动证据提交。
- `docs/development/repository-structure.md`：仅去维护命令列表中两个准确 CLI 行、加窄当前说明；其它维护命令/owner/分类文字不变。它和测试的命令归属表是本批明确调整对象，**不属于允许改动的 CURRENT consumer inventory**。
- `docs/superpowers/specs/2026-09-21-1688-batch-local-agent-design.md`：仅加 §2/§3.1 的准确局部替代注记、改 §3.1 表的一条安装器复用行为显式指纹浏览器输入 owner；其余原文、产品决定、scope/Consent/队列/恢复逐字保持，不把全设计作废。

以下 **18 份历史设计/直接引用**只加针对安装闭包的顶注，完整原 Git blob（含既有顶注、计数和勾选）逐字作为后缀保留，不逐条重写历史：

```text
docs/architecture/legacy-restored-debug-closure-retirement-boundary.md
docs/architecture/legacy-remote-debug-image-closure-retirement-boundary.md
docs/architecture/legacy-runtime-skeleton-config-closure-retirement-boundary.md
docs/architecture/legacy-idle-support-contract-closure-retirement-boundary.md
docs/architecture/legacy-idle-support-facade-closure-retirement-boundary.md
docs/architecture/legacy-http-auth-support-closure-retirement-boundary.md
docs/architecture/legacy-support-control-tail-closure-retirement-boundary.md
docs/architecture/legacy-logger-facade-closure-retirement-boundary.md
docs/architecture/legacy-global-config-logging-tail-closure-retirement-boundary.md
docs/architecture/legacy-host-runtime-closure-retirement-boundary.md
docs/architecture/legacy-host-image-helper-closure-retirement-boundary.md
docs/architecture/legacy-crawler-facade-diagnostics-closure-retirement-boundary.md
docs/CODE_WIKI.md
docs/amazon-listing-workbench-api.md
docs/database-connection-optimization.md
docs/refactoring/code-health-decisions.md
docs/refactoring/module-target-mapping.md
docs/superpowers/plans/2026-08-13-dead-and-duplicate-code-cleanup.md
```

### §5.3 KEEP：零差异边界

当前两个标准浏览器 Dockerfile、UI/Compose/运行脚本、build-listingkit-images.ps1 及其原测试、browser-collector-start.sh、go.mod/go.sum/tools、当前 batchcapture/命令/扩展及全部原测试、公共采集/图像/身份/Store、内核目录与桌面合同均零改。其余原检查/依赖/CI/保护规则零改。

**所有 inventory 文件零改**，包括 phase2-runtime-inventory.md 的顶部旧快照、CURRENT consumer 行、历史/计数/ceilings 和准确性检查。该文件 CURRENT 执行表只列平台日志→config/appenv、OpenAI→config、httpimage→imageagent，未包含本批退出链；删除旧 CLI 不使剩余四行失效。上文与新导航窄替代其旧 browserinstall KEEP 的产品含义，不重写历史快照，不校准其它库存。若真实 CI 暴露失配，先呈准确提案，不自动删行/放宽检查。

## §6 实施与验证

设计确认后同一 Writer 连续完成：先让退休 guard 对现有旧链真实 RED，再整批删除、精确转换检查/文档到 GREEN。当前是纯文档设计，不为形式写业务测试。复用已有 guard/import/结构检查及原 SDK 构建 mock，不新建验证平台、不执行下载/安装/镜像构建/实际浏览器。

### §6.1 原检查精确转换

| ID / 原位置 | 本批允许变化 | 必须保持 |
| --- | --- | --- |
| T1 `TestLegacyHTTPAuthSupportClosureIsRetired` | 仅删 retired/required 表中 httpclient/timeout 两 owner 的行；由 T5 整根缺席及 import 禁入取代 | 当前 HTTP/Auth/Config 字段和声明、loadGoFileIndex、AST/读取/解析错误失败及其余断言 |
| T2 `TestLegacyLoggingHTTPMigrationSupportClosureIsRetired` | 仅去 httpclient const、其 retired/retiredFields 和四字段 Config shape 行、New/Do 与 browserinstall 构造 fragment；扫描 filter 去 HTTP/downloader 两项，去仅为 downloader.httpclient.Config 服务的 SelectorExpr 分支 | 原 logging/migration 的声明/字段/精确形状/全部 fragment、构造禁令、锁/Close/Up/Goose、安全结果、遍历/formatNode/错误算法；其它 CompositeLit 与 AST 分支不变 |
| T3 `TestLegacyRuntimeSkeletonAndIdleConfigClosureIsRetired` | 仅去 retiredSupport/retained 中 httpclient/timeout 两 owner 的行，owner filter 去这两项；已有 builder.go/builder_test.go 文件缺席项保留 | 全部其它 owner/声明/字段/Config/当前日志/Retry、JPEG/PNG/WebP 解码注册、原解析/失败关闭/完整遍历 |
| T4 `TestLegacyRestoredDebugClosureIsRetired` | KEEP 路径表仅删 fingerprint CLI 和 browserinstall 两项 | type_browser_collector.go、tools 三项 KEEP；旧 crawler/hack/listingkit 整根缺席与 decoded import 禁令、循环/错误行为 |
| T5 同文件新增 `TestLegacyBrowserInstallClosureIsRetired` | 复用 assertRestoredDebugPathsAbsent 强制五旧根及 Dockerfile 缺席；复用 assertNoBannedImportPrefixes 在现有 internal/cmd/scripts/tools 根禁入三个旧包；对当前两个 SDK Dockerfile、标准构建/原测试、batchcapture/批量命令和公共采集保留路径做必要直接断言 | 不对合法 SDK stage 名 playwright-installer-builder 作宽泛字符串禁入；不加 missing-skip/空包/错误忽略，不扫描或执行真实浏览器；现有 helper 算法零改 |
| T6 `maintainedOperationalCommands()` | 仅删 fingerprint-browser-installer、playwright-installer 两行 | 命令分类/owner 查验算法、错误退出及全部其它命令不变；与 §5.2 的维护命令文档同步 |

T2 的 Config shape 是整个四字段旧结构（Timeout/MaxRetries/RetryDelay/EnableLogging），不是删除当前其它 Config shape。T1–T4 仅退本批旧 owner 的保活，原其它函数和检查零改；不删整项混合安全测试。新增 guard 复用现有辅助函数，故意重新出现旧目录、import 或旧 Dockerfile 时必须失败。

必要开发验证限：上述原 guard/结构/导航检查、当前标准构建原 mock、无浏览器/网络/实际文件输入的相关原配置/批量纯测试；对当前 batch/public collector/current-application 等受影响依赖入口只编译，绝不运行产物。不扩展原测试矩阵、不借此运行旧包测试/下载动作。稳定候选跑原完整 PR CI，源码最终复核另绑定准确 HEAD；CI 不是实际安装或采集验收。

## §7 准入、评审与交付停止点

1. 当前只提交本设计、architecture/README 和 legacy-register 三文档；#301 当前段记录范围/唯一 Writer，历史后缀逐字保留，正文不超过 65536，状态 OPEN。未完成适用设计复核及用户确认前为 NOT_READY，不改正式源码。
2. 设计复核只核对真实依赖、前批 KEEP 替代、指纹浏览器/制品准备责任、§4.2/§6.1 与完整保留能力。finding 先按仓库规则分类；最多两轮正常架构评审，不将无当前用途的新下载平台或真实发行验收作为前置。
3. 达到适用设计复核结论后仍交用户确认 §1–§7；准入/授权在 Issue/PR 读回后才 IMPLEMENTATION_READY 并整批实施。用户确认同时决定 §1 的安装能力取舍，不是同意接管真实资源。
4. **未经用户本轮授权不默认新增 AI 互验线程；设计与源码最终复核授权分开，分别绑定准确 HEAD。** 同 SHA 原 GitHub 复核运行时不重复触发；失败按现有单次重试/独立复核边界处理，不用“无 finding”冒充已复核。
5. 实施与必要自检、提交/推送、原 CI/finding 修复留同一主要 PR；冻结矩阵外真实消费者、KEEP/检查变化先报准确提案，不绕过门禁。merge-main 只核组合，已有未变化代码证据可明确复用，不假称新 SHA 实测。
6. 源码完成交 In Review，准确实现 HEAD、CI run/attempt、独立最终检查及限制留 PR。合并仍另获准确 HEAD 授权；合并后确认实际 merge SHA 的完整 push/main CI，不用 PR CI/单个 Required Gate 替代。
7. 产品/真实环境验收、下载/安装/构建推送、运行/进程/缓存/Profile/数据操作和关单均未授权；保持 NOT_RUN/UNKNOWN 的区别。不增加试用前置，#301 保持 OPEN。
