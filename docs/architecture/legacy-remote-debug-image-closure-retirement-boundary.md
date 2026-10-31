> #301 / #1029 [用户确认的完整输入链退休](legacy-listingkit-input-config-closure-retirement-boundary.md)：本批仅窄替代当前 UI launcher/tests KEEP 的旧 Task API 输入与隐式集群 Secret 导入；当前 UI、安装器、全部 Dockerfile、标准构建和其它当前能力继续保留。 两组30路径在同一 #1029 整批实施，用户已确认冻结设计 e343a825b 的§1–§7（含§4.2/§6.1），IMPLEMENTATION_READY / SOURCE_IMPLEMENTED / IN_REVIEW。退出旧手工上传配置及旧 UI Task API/隐式集群 Secret 导入；当前 UI/Auth.js/Service API、图像/采集、imgproxy 两有效 Secret 键与原安全检查、Store 观察保留。workflow/inventory/既有实例数据凭据进程不动，无试用前置，#301 OPEN。以下原文、历史、计数和勾选完整保留；源码 CI/最终复核证据另见同一 PR，不代表产品或运行验收。

> #301 / #1026 当前USER_CONFIRMED / IMPLEMENTATION_READY / IMPLEMENTING：用户已确认冻结设计bd53b2044的§1–§7（含§4.2/§6.1），[完整31路径边界](legacy-remote-debug-image-closure-retirement-boundary.md)在同一PR整批实施，不机械拆批；生产修改前授权与准入读回见[6525933152](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1026#issuecomment-6525933152)。七准确工件/专属helper与Pester退出，三原guard窄转换，20旧文档仅顶注且原blob完整后缀。当前安装、标准构建/全部能力与Store观察保持，无alias/新owner；workflow/完整inventory/依赖/保护与既有资源零改，无试用前置，#301 OPEN。必要源码自检/原CI/最终复核在同一PR另列，设计证据不替代源码，合并另授。以下冻结设计/原索引/历史逐字保留，旧阶段时态由本条当前决定替代。

# 旧 SHEIN／ListingKit 远程联调与专属镜像链完整退休设计

Design Basis: **Independent Architecture**。状态：**DESIGN_ONLY / AWAITING_REVIEW_AND_USER_CONFIRMATION / SOURCE_NOT_STARTED / NOT_READY**。父 Issue [#301](https://github.com/henryxu1008/shuomi-ai-commerce/issues/301) 保持 OPEN。

调查基线：main `8551e5787339c5d0d48693f43e713db87da55063`。本轮只交本文、Architecture README 和 Legacy Register 三文档；未来实施的完整矩阵为 **31 路径**，不是本轮已执行的删除清单。准确设计 HEAD、CI、独立复核与用户决定记在同一 PR。

## §1 用户结果与完整范围

使用者是维护当前采集、图像和 Store 观察同步的开发者。当前问题是：旧 API、数字刊登和独立 SHEIN 登录链退出后，旧远端共库联调与专属镜像入口仍被文档或测试称为通用保留能力。其中脚本还建议启动已退休的 `cmd/product-listing-api`。旧 KEEP、当前源码与当前启动依据脱节，容易引导维护者接回旧环境。

本批一个主要 PR 收齐三组，不按脚本、Dockerfile、测试或文档拆批：

| 组 | 退休对象 | 用户可见结果 |
| --- | --- | --- |
| 远程联调 | 四个旧 PG/Redis/RabbitMQ/Temporal 转发与监视脚本，连同内联探针、进程枚举/替换、等待与停止 helper | 仓库不再提供旧共库联调入口或旧 API 启动建议；不产生替代转发工具 |
| 专属镜像 | SHEIN runtime base 构建/推送脚本及其唯一 Dockerfile | 退出无当前仓库构建消费者的旧镜像生产路径，不删除已发布镜像或浏览器安装能力 |
| 保活尾部 | 一份专属 Pester、原 Go 检查的精确转换、直接说明及旧 KEEP 校准 | 旧入口不再因默认端口断言或过期文档被保活，原当前能力与安全检查保持 |

Must：收齐 §5/§6.1；Chromium/Playwright 安装、当前标准构建、采集、图像、身份权限、JSON/hash/版本与 Store 观察同步不变。Should：复用既有退休 guard 和当前安装/构建 owner，不建设新平台。没有新增产品功能、页面、路由或人工验收步骤。

Out of Scope：执行目标脚本、构建/推送/拉取镜像、Docker/Compose/kubectl/Helm、SQL/远端探针、凭据读取、既有实例/数据/文件/日志/进程操作；workflow、完整 inventory（CURRENT、历史、计数、ceilings）、依赖/锁、保护规则改动；旧资源迁移/归档/删除、兼容 alias/fallback、新转发器或恢复工具。内部试用不增加前置。独立 SHEIN 登录和登录后身份回写已在 #985 源退，本批不恢复。

## §2 Design Basis、权威和旧 KEEP 替代

用户明确选择 Independent Architecture。本批涉及远端连接/进程副作用入口和镜像生产职责退出，采用独立文档明确界限；文档完成不等于正式实施准入。

依据：[全新系统产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[项目边界](project-boundaries.md)、[当前目标架构](project-target-architecture.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)及本轮授权。UI/Figma Authority：N/A，不改变页面、IA 或交互。没有新增 Accepted Risk；未检查的外部人工调用、远端服务和制品消费者仍为 UNKNOWN，不写成不存在。

| 先前依据 | 仅被本设计替代的 KEEP | 当前依据与继续保留的能力 |
| --- | --- | --- |
| [#723 CLI 退休](legacy-listing-cli-retirement-boundary.md) §5 | `Dockerfile.shein-runtime` 作为可复用 runtime/base 的保留 | 当前标准构建使用 account-compose 的独立浏览器层；旧 Dockerfile 唯一源码入口为本批脚本。两个安装命令、browserinstall 和其它 Dockerfile 不变 |
| [#839 运维退休](legacy-listing-operations-closure-retirement-boundary.md) §3/§5 | SHEIN PG/Redis/RabbitMQ 转发器的通用人工工具 KEEP | 数字队列与登录业务退出；仓库无当前业务/工作流调用此工具。不能因手工入口无静态 caller 就直接断言无人使用，也不能仅因曾列为工具而永久保留 |
| [#846 本地验收](legacy-local-acceptance-closure-retirement-boundary.md) §5.2 | 四个 port-forward/watcher 及其 Pester 的原样 KEEP | 旧共库 replay 已标历史，当前应用以明确私密 manifest 和现有 Compose/运行 owner 装配；当前 UI、Auth.js handoff、issue357、原生身份与 Temporal dev 工具继续保留 |
| [#872 队列/代理退休](legacy-queue-proxy-config-tail-closure-retirement-boundary.md) §4/§5.2 和 code-health Task 11 | SHEIN 转发器及 watcher 的 retained-deferred；通用转发工具中这四个准确对象 | 不再保留旧数字链专属远端联调入口；PG/Redis/RabbitMQ/Temporal 共享基础层、声明和正常诊断能力不随本批退出 |
| [#939 全局配置尾部](legacy-global-config-logging-tail-closure-retirement-boundary.md) §5/§6.1 | 仅删两行 Redis 建议、其余脚本/Pester逐字 KEEP | 改为四源及专属测试完整缺席，旧 Redis 建议自然随源退出；原其它配置禁入、读取失败和当前配置校验不变。其十三份承接顶注一并窄校准 |
| [#884 品牌/更名清单](../development/branding-and-repository-rename.md) | 本批脚本的保留含义，不是 Docker Hub 名称、tag 或现存资源 | 只退仓库生产入口；不重命名、删除、推送镜像或接管运行 owner，外部制品消费者继续 UNKNOWN |
| [旧 1688 preflight 计划](../superpowers/plans/2026-08-08-1688-runtime-preflight.md) | 本批 port-forward/测试的现行操作含义 | 只追加准确历史注记，不改原计划勾选、失败或旧路径解析证据，不把旧计划当当前派工 |

以上是待确认的窄替代，不批量重写旧产品决定。用户确认本设计后仅这些准确对象的 KEEP 失效，原文保留为历史，不撤销其它有效规则。

## §3 实际消费者、装配和正确 owner

### §3.1 待退休链

按 tracked 源、脚本、Dockerfile、Makefile、两套 workflow、tests 和 docs 交叉搜索准确文件名、镜像名和入口。调查只读源码，没有执行程序或探测端口。

| 对象 | 实际源码消费者与行为 | 退出边界 |
| --- | --- | --- |
| `scripts/start-listingkit-local-portforward.ps1` | 专属 Pester 静态解析默认服务；`TestRetiredTaskAndListingRuntimeConfigClosure` 必读其内容以禁两条旧 Redis 建议；已标历史的共库 replay 和 dated preflight 计划。脚本会枚举并停同类 kubectl、转发 DB/Redis/可选 Temporal、打印已退 API/旧路由建议 | 整份源与 Pester 同退，原 Go 必读转换；参数可覆盖不等于有当前业务消费者 |
| `scripts/start-shein-local-rabbitmq-portforward.ps1` | 无当前程序、维护脚本、workflow、Makefile 或测试调用；KEEP 文档保留其人工工具身份。仍转发旧 namespace 的 PG/Redis/AMQP/管理端口，枚举并停止旧转发进程 | 内联等待/进程/日志 helper 一起退，不抽为通用 ops 库 |
| `scripts/start-listingkit-remote-temporal-portforward.ps1` | 仅历史共库 replay 有准确操作引用；脚本直接转发远端 Temporal，打印旧 `LISTINGKIT_TEMPORAL_*` 建议 | 不改当前 Temporal SDK、worker 队列、Update、回放或恢复；没有替代连接入口 |
| `scripts/watch-listingkit-local-portforward.ps1` | code-health retained-deferred；没有当前源码调用。固定旧 Pod/端口，直连 PG/Redis/Temporal 探针，失败时停止/重建转发，在 `.local` 写状态/日志 | 整份监视与探针/重启逻辑同退；不枚举进程、不读取或清理其已有状态文件 |
| `scripts/build-push-shein-runtime-base.ps1` | 唯一构建目标为 `deployments/docker/Dockerfile.shein-runtime`；脚本内自用示例及品牌清单，当前 workflow/标准构建无调用。会 docker build/push `task-processor-shein-runtime`，可附推 latest | 构建与推送入口一并退；不访问 registry、不变更现存 tag/image |
| `deployments/docker/Dockerfile.shein-runtime` | 唯一仓库构建引用是上述脚本。编译两个安装器，以 crawler-env 为基底，安装浏览器；没有当前 Dockerfile 以该镜像名为 FROM | 退出这份装配，不退出两个安装器、共享下载代码或其它浏览器构建 |

### §3.2 保留链：contract → implementation → injection → consumer

| 当前能力与 owner | 实际调用/装配，整链保持 |
| --- | --- |
| Chromium 安装 | `cmd/fingerprint-browser-installer` → `integration/browserinstall.NewChromeDownloaderWithTimeout` → 原下载/版本/超时/日志；非目标 `Dockerfile.product-listing-api-runtime-base` 仍编译并调用该命令，命令及其原测试/共享工具都保留 |
| Playwright 安装 | `cmd/playwright-installer` 使用当前 Playwright Go SDK；非目标 runtime-base 仍调用。`Dockerfile.account-compose` 与 `Dockerfile.browser-collector` 另外按 go.mod 的 SDK 编译官方 installer 并安装 Chromium；不改版本、路径或注册 |
| 当前标准镜像构建 | `scripts/build-listingkit-images.ps1` → 指定 Git archive → `Dockerfile.account-compose` 的 `data-services-runtime` 和 `Dockerfile.listingkit-ui` → 原镜像校验/receipt；不引用 SHEIN runtime。现有 `runtime` 目标选项及其原安全测试保留，不因旧专属镜像退出改构建参数 |
| 当前采集与图像 | 现有 acquisition/Collector、浏览器层、current-application、ImageAgent worker/当前图像及 Artwork 的原装配；不注入这些旧转发脚本，不改 SDK/provider/预算/UNKNOWN 或当前 JSON/hash |
| Store 观察同步 | 当前 Store/官方授权 → `app/storeobservations` 与原观察仓库/Temporal 装配；不以旧数字 Store、登录 Cookie 或旧 RabbitMQ 代替。全部当前源、测试和持久化保持 |

本批不新增 EXTRACT。当前正确行为已经有 owner，不能为保住旧入口再包装安装器、造新连接管理器或迁入进程控制 helper。普通远端客户端与基础设施不因这些历史脚本退出被判为 legacy。

## §4 状态、持久化、权限及副作用边界

| 边界 | 源码事实与本批决定 |
| --- | --- |
| 旧 DB/Redis/RabbitMQ/Temporal | 脚本只提供网络通道/探针，不是这些服务的数据或恢复 owner。源码退休不停止服务、断开连接、回收端口、消费/删除消息、读写表/键或结束历史任务 |
| 本机进程与日志 | 四脚本会枚举、停止、启动子进程并写日志；watcher还写本机状态。删除 tracked 源不追踪、停止或清理已有进程、日志、PID、`.local`、临时文件或用户会话 |
| 镜像与安装制品 | 旧构建脚本生产的远端镜像/tag 和本地缓存归原镜像/运行 owner；只删除生产入口，不删除已发布制品、安装目录、浏览器缓存或改镜像仓库名称 |
| 当前事实和安全 owner | Organization/身份授权、Store、Product/Catalog/Asset、图像资金/预算、当前事务/幂等/回执、retry/UNKNOWN/取消恢复全部原样，无 schema/版本/序列化/JSON/hash 或权限变化 |

### §4.1 Threat Model 与失败边界

保护 Must 是不误退当前安装/构建/采集/图像/Store，不把静态调查误当运行验收，不在修改检查时允许其它旧链复活。移除脚本内旧等待/retry/重启能力不把其持久化责任交给当前应用；没有新的状态机、事务、幂等身份、自动恢复触发器或外部 mutation。

当前 SDK/服务/业务在响应丢失、重试、取消、重启与并发下的行为不变，本批只断开六个旧源码入口。原远端操作是否仍运行未检查，不能宣称已安全停机或环境清零。发现具体当前外部契约或仍需该入口完成当前 Must 的证据时，停止对应删除并报告；不得自行添加兼容或运行清理。

### §4.2 产品及持久化取舍

确认本设计意味着仓库不再承诺通过这四个脚本访问旧共库环境，也不再通过该 SHEIN 脚本构建/推送专属镜像；不是取消当前能力、停止现有通道或删除制品。旧路径不保留 alias/重定向，不改写为转调新工具，不迁移历史配置。已有私密 manifest、当前 Compose/CLI、数据库/队列事实、安装目录、已保存载荷及其 hash/版本均不修改、不读取。

运行态处置需要将来单独授权，且不成为本轮设计/源码退休或内部试用的新前置。没有当前外部消费者证明，不以 UNKNOWN 自动恢复这些入口或推导新的验收任务。

## §5 完整删除／保留矩阵

### §5.1 用户确认后整退七个准确工件

```text
scripts/start-listingkit-local-portforward.ps1
scripts/start-shein-local-rabbitmq-portforward.ps1
scripts/start-listingkit-remote-temporal-portforward.ps1
scripts/watch-listingkit-local-portforward.ps1
scripts/build-push-shein-runtime-base.ps1
deployments/docker/Dockerfile.shein-runtime
scripts/start-listingkit-local-portforward.Tests.ps1
```

五个操作脚本、一个 Dockerfile 和一份专属 Pester 为完整删除单元；不保留内联 helper/空壳，不新增替代实现。不递归清理目录、ignored 文件、缓存、镜像或任何运行对象。

### §5.2 混合检查与文档尾部

仅一个混合测试源：`tests/listingkit_backend_retirement_test.go`，三个准确函数的变化见 §6.1。

20 份既有文档只追加窄顶注，原 Git blob 必须逐字节成为后缀，不改旧数字、计数、勾选或失败证据。十三份 #939 承接顶注也一并收齐，避免把“其余转发/Pester不改”继续当当前权限。顶注只替代本批七对象，链接本文；其余 KEEP 和原 owner 不变。

```text
docs/architecture/legacy-listing-cli-retirement-boundary.md
docs/architecture/legacy-listing-operations-closure-retirement-boundary.md
docs/architecture/legacy-local-acceptance-closure-retirement-boundary.md
docs/architecture/legacy-queue-proxy-config-tail-closure-retirement-boundary.md
docs/architecture/legacy-global-config-logging-tail-closure-retirement-boundary.md
docs/architecture/legacy-config-facade-closure-retirement-boundary.md
docs/architecture/legacy-host-image-helper-closure-retirement-boundary.md
docs/architecture/legacy-idle-support-contract-closure-retirement-boundary.md
docs/architecture/legacy-idle-support-facade-closure-retirement-boundary.md
docs/architecture/legacy-logger-facade-closure-retirement-boundary.md
docs/architecture/legacy-platform-api-closure-retirement-boundary.md
docs/architecture/legacy-redis-proxy-shared-closure-retirement-boundary.md
docs/architecture/legacy-runtime-config-closure-retirement-boundary.md
docs/architecture/legacy-runtime-skeleton-config-closure-retirement-boundary.md
docs/architecture/legacy-sds-runtime-closure-retirement-boundary.md
docs/refactoring/code-health-decisions.md
docs/development/branding-and-repository-rename.md
docs/development/listingkit-local-prod-db-replay.md
docs/superpowers/plans/2026-08-08-1688-runtime-preflight.md
docs/superpowers/plans/2026-08-30-internal-target-architecture-phase2.md
```

另有本文及 `docs/architecture/README.md`、`docs/refactoring/legacy-register.md` 三个当前设计/导航文件；导航亦只加当前条目，原内容完整保留。合计 **7 删除 + 1 混合 Go 测试 + 20 直接旧文档 + 3 当前设计/导航 = 31 路径**。本设计提交只应用最后三文档；其余尚未实施。

### §5.3 KEEP／不动清单

- `cmd/{fingerprint-browser-installer,playwright-installer}`、`internal/integration/browserinstall`、原测试及下载/超时/HTTP/日志基础层全部保持。
- `scripts/build-listingkit-images.ps1`、`.test.mjs`、镜像构建说明、所有非目标 Dockerfile（包括 product-listing-api-runtime-base）、account/acquisition/referrals Compose、当前 Collector/图像/Store观察装配与正常 schema 命令保持。
- `scripts/start-listingkit-local-ui.ps1`/测试、start/stop-temporal-dev、issue357全部源/测试、当前私密 manifest 机制、身份 CLI、安全断言保持；不把已退 db-query 或 SHEIN 登录恢复为保留对象。
- Kubernetes 所有 Namespace/基础设施/存储/网络/监控/发布策略源、全部 `.github/workflows/**`、完整 inventory（含 CURRENT、dated/历史、计数、ceilings 和准确性检查）、依赖/锁、lint/工具/保护规则零改。
- 不改生产 Go/TS/业务配置，不操作源码外实例、数据、凭据、安装制品或进程。发现矩阵漏项先核实同根闭包并报告精确差异，不借退休做无关整理。

```text
Legacy decision: RETIRE
Reusable behavior: 无新抽取；当前安装、构建、采集、图像和Store观察由原owner继续提供
Current owner: integration/browserinstall、当前Playwright SDK/标准Dockerfile与build-listingkit-images、当前领域/应用/Store观察owner
Cutover/deletion condition: 适用独立设计复核与用户确认，§5/§6.1准确闭包实施，当前能力与原必需/安全检查保留；源码退休不代表真实资源停用
```

## §6 必要验证与验收边界

本设计阶段只做 tracked 来源/消费者、矩阵/链接/导航一致性、`git diff --check` 和原适用文档检查。不执行脚本/Pester/构建/Docker/安装或真实探针，不为纯文档制造 RED。原 PR CI 按既有 workflow 执行适用检查；静态、隔离 fixture、SKIPPED 和真实验收分别记录。

用户确认后同一 PR：先修改原退休检查并确认因目标源仍存在而 RED，再整退七工件及窄尾部至 GREEN；运行三个受影响 guard 与现有安装/构建源合同的必要检查。当前安装器如需编译仅 `go build`/`go test -run '^$'`，不得执行安装/下载；镜像验证复用已有 mock/text tests，不实际 build/push。完整候选运行原 CI，一次集中最终独立复核；不反复重跑无关测试或新增验证工具。

### §6.1 原检查、文档、workflow／inventory 精确处置

| 准确对象 | 用户确认后允许的变化 | 必须保持／验证 |
| --- | --- | --- |
| `TestRetiredLocalLaunchersRejectBeforeOperations` | 原准确缺席表增加四个转发/watcher脚本和专属 Pester，共五路径；原循环/Stat/非缺席错误失败及已有路径不变 | 所有旧 API/dev/验收/token 脚本和独立 replay 缺席仍强制，不能加 missing-skip 或删整个函数 |
| `TestRetiredListingCLIsHaveNoBuildOrDeploymentEntrypoints` | 原准确缺席表只增加 SHEIN base build/push 脚本与 Dockerfile 两路径 | 原 Makefile/两 workflow必读、旧 CLI/runtime/auto-shard禁入、current-application必须编译、失败关闭全部原样；读取 workflow不等于修改 workflow |
| `TestRetiredTaskAndListingRuntimeConfigClosure` | 仅把末尾“必读旧 local-portforward 并禁两项 Redis env文本”替换成该脚本准确路径必须缺席；源不存在即不可能继续提供旧建议 | 前部所有旧配置/源缺席、当前 Secret必读与 RabbitMQ键禁入全部原样；不改其它函数或配置加载/校验逻辑，不保空壳满足必读 |
| 专属 Pester `start-listingkit-local-portforward.Tests.ps1` | 随唯一被测脚本删除，退出旧参数默认值和 parser helper 的保活 | 它只有旧源解析/默认服务测试，没有独立当前安全/业务断言；不移植旧端口/远端默认值到新工具，当前 UI/构建/Pester其它测试零改 |
| §5.2 的20文档 | 仅加准确 superseded/source-retired 顶注，明确本批七对象、当前安装/构建/Store观察和 UNKNOWN/NOT_RUN；原完整 blob 逐字节后缀保持 | 不重写旧计划勾选/数字/失败，不再把旧共库步骤当当前runbook，不增加新的运行操作要求 |
| 本文、Architecture README、Legacy Register | 当前状态/准入/同一PR链接与准确矩阵；README完整索引，Register登记 RETIRE/原owner | 其它原正文/历史完整保留，不把设计完成写成实施/运行验收成功 |
| 所有其它测试与guard、lint、workflow、完整inventory、历史计数/ceilings、依赖/锁、保护规则 | **零改** | 原解析/读取失败、当前安装/构建/身份/事务/采集/图像/Store观察安全断言不放宽；失败先查真实根因，不简单重跑或绕门禁 |

本轮核对仅发现上述三项 Go 函数和一份专属 Pester 依赖待退源。若实现/原 CI 暴露另一个消费者或未列原检查，先核对 sibling paths，交付准确根因、最小补丁及矩阵修订，取得必要确认后再应用；不能先改后补授权。

## §7 评审、确认、实施与停止点

1. 本轮交付三文档及一个主要设计 PR。适用独立设计复核检查真实源、全部31路径闭包、§2旧KEEP替代、§4.2/§6.1和当前保留能力。最多两轮正常架构评审，finding先分类；CI绿色或作者自检不是独立结论，也不是用户确认。
2. 未经用户本轮授权不默认新增 AI 互验线程。原 GitHub 复核运行时不重复触发；确实无法完成时可报告准确状态，请用户决定是否授权一次只读子代理设计复核。源码最终复核与设计复核分别绑定准确 HEAD，授权分开，不扩大为多 Writer。
3. 设计复核和必要原 CI 完成、用户确认 §1–§7（含§4.2/§6.1）后，将冻结设计 HEAD/准入与唯一 Writer 写 #301 和同一 PR 并读回，才可标 IMPLEMENTATION_READY、正式修改七目标及其测试/文档尾部。未满足时保持 SOURCE_NOT_STARTED，不执行预删/试运行。
4. 同一完整批次连续实施、必要自检/提交/推送与范围内 CI/review 修复；不机械拆PR。发现当前Must无法满足、当前外部可观察契约、需触碰运行资源/凭据或改变授权/事实/事务/恢复边界时，停止对应部分报告，不自行扩大权限或发明兼容。
5. 最终交付准确源码 HEAD、原 CI、独立最终复核及全部 NOT_RUN；合并等待准确 HEAD 明确授权，正常 expected-head 合并后核对实际 merge SHA 的完整 push/main CI。部署、真实环境操作及关闭另授权。#301 始终 OPEN，不把本轮当内部试用前置或宣称全仓/环境清零。
