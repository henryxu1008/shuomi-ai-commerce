# 旧 ListingKit 上传配置与本地 UI 输入链完整退休设计

Refs #301。Design Basis: **Independent Architecture**。状态：**DESIGN_ONLY / SOURCE_NOT_STARTED / NOT_READY**；先完成适用设计复核及用户确认，再整批实施。一个主要 PR，不按文件或内部步骤拆批。

## §1 用户结果、范围与设计依据

使用者是维护当前安装和本地 Console 的开发者。旧手工上传 HTTP/Task UI 已退出，但配置层仍能构造一个没有应用消费者的上传 owner，本地启动还设置已无前端消费者的 Task API 地址，并可按继承的 KUBECONFIG 自动读取旧集群 Secret。当前两份试用说明甚至把这个无效地址当作当前连接配置。结果是配置看似生效却不能完成当前操作，且本地身份输入可能被隐式替换。

本批一次收齐两组输入链及其样例、测试和直接文档尾部：

1. `ListingKit.ImageUpload` 字段、三个类型、默认值、九项环境绑定/构造、六项专属 ConfigMap 键及四个旧保活测试；
2. UI launcher 的旧 `ApiBase` 参数、两项 Task API env、自动读取集群 Secret 的函数和调用，以及 Compose 的旧 env 一行。保留整个当前 UI 启动工具，改接其已有 Service API 和显式身份输入。

依据：[Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)及用户本轮完整闭包/保留能力/零运行操作决定。调查基线 `main=22b09ccec793ee17bae630d76f8646d90890a7a9`。本批无页面、IA、交互或 Figma 变更，UI/Figma Authority **N/A**。

Legacy decision: **RETIRE**。Reusable behavior / Current owner：当前 UI/Next、Auth.js、Service API、ImageAgent/SourceMedia 和 imgproxy 已有正确 owner，原样保留，不新建 EXTRACT 包、adapter、配置别名或凭据管理器。Cutover/deletion condition：用户确认本设计、完成适用准入后，在同一 PR 整批删除下面准确对象，原有效断言保留。

Out of Scope：其它旧代码、当前 ImageAgent 准入/存储、任何身份认证/授权协议、当前 Secret 名称迁移、Kubernetes 发布恢复、运行清理、镜像/依赖升级、workflow/inventory/检查算法/保护规则调整。独立 SHEIN 登录和旧身份回写已在 #985 退出，本批不恢复它们。

## §2 前批 KEEP 的准确替代

下表是待本轮用户确认的窄替代；原记录保留历史，不批量撤回其它 KEEP。

| 原依据 | 本批替代 | 仍保留 |
| --- | --- | --- |
| [#846 本地启动](legacy-local-acceptance-closure-retirement-boundary.md) §5 当前 UI launcher/tests、API base 和 secret 行为全保留 | 仅退出旧 Task API 两 env/参数与隐式 Kubernetes Secret 导入 | Next 启动、Service API、dotenv、Auth.js/handoff、isolated 模式、路径/端口/就绪/已验证进程清理 |
| [#847 发布执行](legacy-kubernetes-release-execution-closure-retirement-boundary.md) ConfigMap/Ingress 及 §5 的全局同名配置合同 KEEP | 仅替代全局 `LISTINGKIT_API_BASE` 旧 Task API 输入的保留含义；当时只删旧 ConfigMap 两键不再阻止本批退出无消费者的全局旧输入 | `LISTINGKIT_SERVICE_API_BASE`、当前身份/构建、共享配置、全部其它 KEEP 原样；不修改或恢复旧发布声明 |
| [#852 token 工具](legacy-api-token-tools-retirement-boundary.md)当前 UI launcher/tests KEEP | 同上；不撤回 Auth.js/native identity/Login V2 的正确 owner | token handoff、认证验证、现有显式凭据输入 |
| [#885 SDS](legacy-sds-runtime-closure-retirement-boundary.md)只删 SDS env、其它 API base/启动保持 | 精确退出剩余两个 Task API env，不涉及当前 Service API | 原 SDS 禁入、身份/loopback/环境恢复断言 |
| [#958 provider bootstrap](legacy-idle-provider-bootstrap-closure-retirement-boundary.md)配置测试仅改 fixture owner、断言保持 | 四个只验证旧上传配置的当前测试退出；两个 ArtifactStore 测试和一个 Admission 测试逐字保持 | 当前配置 fixture、现役六组配置校验和其它测试 |
| [#985 SHEIN 登录链](legacy-shein-login-identity-sync-closure-retirement-boundary.md)其余 ConfigMap 键值及 UI 行为保持 | 仅六个旧上传非敏感键及专属双 owner 注释、上述 UI 输入退出 | Secret 样例全部原键值、imgproxy 映射及安全检查、当前 Store 观察/官方授权 |
| [#1026 远程联调](legacy-remote-debug-image-closure-retirement-boundary.md)当前 UI launcher/tests 全保留 | 仅上述闲置/隐式输入，不删 UI launcher | Chromium/Playwright 安装、全部 Dockerfile、标准构建、当前采集和 Store |

## §3 实际消费者、装配与 owner

只读核对 tracked Go/TS/PowerShell、Compose、Kubernetes 样例、两套 workflow、Makefile、tests/docs；未执行 launcher、kubectl、浏览器或服务探测。静态没有消费者不代表已检查外部人工使用，后者 **UNKNOWN**。

| 链 | 实际源码路径与结论 |
| --- | --- |
| 旧上传配置 | `type_listingkit.go` → `NewDefaultConfig` / `setDefaults` / `bindKnownEnvs` → `buildConfig` → `ListingKit.ImageUpload`。精确字段/类型搜索仅命中这条配置自循环及四个配置测试，无应用、上传服务或 current-application 消费者；不能视为有效上传能力 |
| 旧上传样例 | ConfigMap 的 PROVIDER、S3_PUBLIC_BASE/BUCKET/REGION/ENDPOINT/USE_PATH_STYLE 六键只有旧应用绑定；imgproxy 使用独立 `IMGPROXY_*`，不读取这六键。无 LOCAL_ROOT_DIR 样例待删 |
| 旧 Task API | `start-listingkit-local-ui.ps1` 的 `ApiBase` → `Set-UiApiBases` → `LISTINGKIT_API_BASE` / `NEXT_PUBLIC_LISTINGKIT_API_BASE`；Compose 另设置前者。`web/listingkit-ui` 当前生产 TS/配置/脚本无这两个精确 env 消费者，旧 `/api/listing-kits` 路由已受退休 guard 禁入 |
| 隐式凭据导入 | 正常 `Initialize-UiLaunchEnvironment` 先读 `.env`，再调用 `Import-DeployedListingKitAuthSecrets`：按 KUBECONFIG 调 kubectl 读取整份旧 Secret，仅把 ZITADEL_CLIENT_SECRET 解码覆盖到本进程。函数只在本脚本调用，测试通过 AST 导入。删除整个函数及调用，不移动/包装为新管理工具 |
| 当前 UI | `ServiceApiBase` → `LISTINGKIT_SERVICE_API_BASE` → 当前 BFF（如 `data-services-proxy.ts`、`agent-configuration-origin.ts`）→ current-application；`auth.ts`/`auth.config.ts` 继续用现有显式 env 与 NextAuth。正常入口仍为原 launcher 或 `pnpm dev`；本批不运行它们 |
| 当前图像/采集 | `currentapplication.Config.CoreConfig` 原样注入 ImageAgent.Generation/Admission/ArtifactStore 与 SourceMedia；HTTP `imageagent_acquisition_module.go`、`image_set_application.go` 实际检查/使用组织准入和对象存储。与旧上传三个配置类型不同，全部保留 |
| imgproxy | `imgproxy-deployment.yaml` 仍把两个 `TASK_PROCESSOR_LISTINGKIT_IMAGE_UPLOAD_S3_*KEY*` Secret 键映射到 `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`；这是保留消费者。名字历史化不等于可删，原四项精确 Secret allowlist 不变 |
| Store 观察 | 当前 Store/官方授权 → `app/storeobservations`、`runtime/storeobservations` 与原 Store 仓库；不依赖本批配置/旧 Task API/集群导入。原源、测试、持久化和恢复 owner 不变 |

## §4 权限、事实和副作用边界

旧配置类型只有输入值，无表/迁移/业务资产 owner。删除它们不访问、移动或删除任何磁盘上传文件、bucket/object、凭据、session、私密 manifest 或历史运行产物。当前 Product/Asset/ImageSet/Artwork/Organization/Store 事实和数据库角色/事务均原 owner；不建立第二事实源。

### §4.1 Threat Model 与不变量

- 当前授权、组织隔离、准入、预算、JSON/hash/版本、事务/幂等、deadline/取消、UNKNOWN/恢复完全不变。
- 保留当前安全 URL/loopback/独占端口/隔离目录/就绪与已验证进程清理；不把本轮“禁止操作进程”误写成删除这些当前工具能力。
- 保留原 `.env` 只补空值、继承显式进程输入优先、isolated 模式不读仓库 `.env`/清空 KUBECONFIG；保留 AUTH_SECRET 与 Auth.js 原会话/验证语义。本批不读取这些真实值。
- Secret 样例、imgproxy deployment、`tests/listingkit_secret_boundary_test.go` 全文零改；两保留 Secret 键不得被误加入应用的全局 retired-env 拒绝表。
- 当前运行链在失败、响应丢失、重试、重启、取消和并发下的责任不变；无新状态机、事务、provider 发送、credential 轮换或自动恢复触发器。

### §4.2 输入、JSON/hash 与持久化取舍

用户确认本设计意味着旧 `listingkit.imageUpload` YAML 不再属于可用配置；复用现有 `retiredProductRuntimeYAMLKeys` 拒绝该根（包括有值的子键），错误不输出配置/凭据值。九个 env 仅删除应用默认/绑定/构造，不新增全局拒绝、回退或映射到 ImageAgent；原 YAML/环境读取算法不变。保留两 Secret 键的 imgproxy 用法，其余环境变量仍按原规则。已有文件不会被重写或迁移。

旧 UI `-ApiBase` 参数直接退出，不留 alias；两个旧 env 无应用消费者，启动器不再设置/读取/打印，Compose 删除唯一旧 env 行，不增加新的 env 清理副作用。当前 `-ServiceApiBase` 保持名称、默认与 normal 模式只补空值；isolated 仍强制显式 loopback Service API 并覆盖继承值。删除只为旧 ApiBase 计算的闲置局部量。helper 可保留原名 `Set-UiApiBases`，不为改名扩大引用面。

正常 UI 仍使用原显式进程输入/原 `.env`；以后不因设置了 KUBECONFIG 就导入/覆盖 OAuth client secret。已有 Account Compose 的 frontend-secrets 私密卷/进程注入不变，不替换 client、AUTH_SECRET、token 或登录流程；缺少合法配置继续由当前 Auth.js 按原规则处理，不增加默认身份、集群回退或新试用步骤。现存手动依赖集群导入的运行配置未检查，不能声称实际登录验收已通过。

以上是源码输入取舍，不是环境清理或旧会话迁移。发现具体当前外部必须依赖这些旧输入的证据，先报告用户，不擅加兼容；运行验收仍 **NOT_RUN**，不作为内部试用新前置。

## §5 完整修改／保留矩阵

未来整批共 **30 路径**；本轮设计只修改其中新设计与两导航三文件。没有整包搬迁或新运行组件。

### §5.1 配置、启动与样例（13 路径）

| 文件 | 唯一允许的改动 |
| --- | --- |
| `internal/core/config/type_listingkit.go` | 删除 ImageUpload 字段及三个专属配置类型；保留 RolePolicyDatabase、PlatformAdminUsers/Roles、Zitadel 及旧用户名配置拒绝合同 |
| `internal/core/config/loader.go` | NewDefaultConfig 仅删除该空配置构造，最小 gofmt |
| `internal/core/config/loader_builder.go` | buildConfig 仅删除 ImageUpload 构造树 |
| `internal/core/config/defaults.go` | setDefaults 仅删除九条旧默认 |
| `internal/core/config/config.go` | bindKnownEnvs 仅删除九项旧绑定，不动 replacer、dotenv、AutomaticEnv 或其它输入 |
| `internal/core/config/retired_product_runtime.go` | 只在既有 YAML map 加 `listingkit.imageupload` 对应原始 key，拒绝算法和 env map 零改 |
| `internal/core/config/task16c_imageagent_config_test.go` | §6.1 四项旧保活退出；其它三项原样 |
| `internal/core/config/task16c_retired_config_test.go` | §6.1 在既有 YAML 测试增加此根的合成 fixture；原 env 测试不改 |
| `scripts/start-listingkit-local-ui.ps1` | §4.2 旧参数/env/打印、隐式 Secret helper/调用和准确专属提示退出；其余启动、身份、路径/进程行为保持 |
| `scripts/start-listingkit-local-ui.Tests.ps1` | §6.1 对应 AST importer、API/身份输入测试，保留其余有效断言 |
| `deployments/kubernetes/listingkit-workbench/base/configmap.yaml` | 只删除 §3 六旧上传键；双 typed-owner 注释改为当前 ImageAgent 说明；其它键值/注释不动 |
| `deployments/docker/account-compose/docker-compose.yml` | 仅删 frontend environment 的 `LISTINGKIT_API_BASE` 一行；command、Service API、Secret volumes、依赖/健康检查零改 |
| `tests/listingkit_backend_retirement_test.go` | §6.1 仅新增一个闭包 guard，原全部函数/断言不变 |

### §5.2 文档与导航（17 路径）

- 当前两处错误操作说明直接只改所述地址为已有 `LISTINGKIT_SERVICE_API_BASE=<current-application origin>/api/v1`：`docs/operations/product-agent-trial.md`、`docs/operations/ai-workbench-chat-trial.md`。Auth.js、可用性、预算/付费准入和其余步骤保持，不新增流程。
- 八份历史文档仅追加本批准确窄注记，原 blob 作完整后缀：`docs/architecture/legacy-local-acceptance-closure-retirement-boundary.md`、`legacy-kubernetes-release-execution-closure-retirement-boundary.md`、`legacy-api-token-tools-retirement-boundary.md`、`legacy-sds-runtime-closure-retirement-boundary.md`、`legacy-idle-provider-bootstrap-closure-retirement-boundary.md`、`legacy-shein-login-identity-sync-closure-retirement-boundary.md`、`legacy-remote-debug-image-closure-retirement-boundary.md`，及 `docs/refactoring/code-health-decisions.md`。只替代 §2 对象，不改旧数字、状态、检查表或历史结论。
- 四份旧操作参考仅在顶部加准确历史注记并保留原 blob 后缀：`docs/development/listingkit-object-storage.md`、`docs/development/listingkit-local-debug.md`、`docs/development/listingkit-local-prod-db-replay.md`、`deployments/kubernetes/listingkit-workbench/README.md`。旧手工上传/API 示例不再是当前入口；不能把保留的 Service API/Auth.js/imgproxy 一起宣布退休。
- 本设计 `docs/architecture/legacy-listingkit-input-config-closure-retirement-boundary.md`、`docs/architecture/README.md`、`docs/refactoring/legacy-register.md` 共三份维护当前导航/状态，导航旧 blob 完整保留。

矩阵外生产/测试文件零改。发现遗漏消费者或需要改变原安全断言时，先给出真实失败、同根完整核对及准确提案，不能把设计冻结解释为放宽 CI 的授权。普通第三方组件和当前 owner 不因名称包含 ListingKit 被判为旧设计。

## §6 验证与交付

设计阶段只做来源/owner/引用/矩阵核对、适用原文档 guard 与原 CI；不为纯文档制造业务 RED，不启动服务或搭建新验收工具。将准确 HEAD、CI run/attempt、独立设计结论和用户决定留 PR，旧头结果不冒充新头直接证据。

实施确认后，先针对两条链让本批闭包检查在原源真实 RED，再完成整批 GREEN。复用原配置测试、UI Pester 的 AST/合成 fixture、Secret 边界和受影响 Compose/架构检查，不执行启动器顶层，不调用真实 kubectl/端口/浏览器/对象存储/进程。原安全断言不能以移除旧输入为由跳过。无需新临时 DB/Redis，本批不改 schema。

### §6.1 原检查处置（明确冻结边界）

| 位置/函数 | 处置与保持 |
| --- | --- |
| config `TestConfigFixtureBindsListingKitOwnedImageUploadYAML`、`TestNewDefaultConfigDoesNotSelectListingKitImageUploadStorage`、`TestConfigFixtureDoesNotDefaultListingKitImageUploadToLocal`、`TestConfigFixtureBindsListingKitImageUploadS3Environment` | 只保活旧合同，四函数删除；同文件 ArtifactStore YAML/env 和 Admission env 三测试逐字保留 |
| `TestConfigFixtureRejectsRetiredProductRuntimeYAML` | 仅表增旧根和子键合成用例；保留原错误/无值泄露语义。若现有 map 不覆盖空 map，记录事实，不改原通用 YAML 识别算法来扩张合同；本批至少覆盖全部旧有值输入 |
| 新 `TestRetiredListingKitUploadEnvDoesNotRecreateApplicationOwner`（同 retired_config_test 文件） | 九 env 合成值不复活应用配置、两个 imgproxy 键不被全局拒绝；复用 loadConfigFixture，检查禁止类型/字段用闭包 AST，不读真实环境值 |
| Pester `Import-StartUiScriptFunctions` | 从 AST 导入表移除已删 Secret helper，加入已有 Set-EnvIfMissing 以检查 normal 入口；解析/函数提取算法不变 |
| Pester `overrides inherited API bases in isolated acceptance` | 只保留当前 Service API fixture/期望与原 try/finally 恢复；旧两 env 保活项和旧参数退出，仍断言覆盖继承地址 |
| Pester `rejects non-loopback API bases in isolated acceptance` | 令非法地址 fixture 指向仍在使用的 ServiceApiBase；同样要求 loopback 错误，不删安全检查 |
| Pester 新两个用例：normal explicit auth inputs / no retired Task API or cluster-secret entry | 任务自有 TestDrive 合成 .env/合成进程 env 验证显式优先和不覆盖已有 AUTH_SECRET/ZITADEL_CLIENT_SECRET；静态 AST/字符串断言旧参数/变量/Secret helper/kubectl 不存在。只调提取函数，不执行顶层启动或探测进程 |
| 原 Pester 其它用例 | hostile auth env/isolated KUBECONFIG、占用端口拒绝、loopback bind、实际 ready、已验证清理与 wildcard 拒绝保持原断言；可新增同入口空 ServiceApiBase 拒绝用例，不删原保护 |
| 新 `TestLegacyListingKitInputConfigurationClosureIsRetired`（原 backend_retirement_test 文件） | 准确扫描原配置/launcher/sample：三旧类型/字段、默认/绑定/构造、六 ConfigMap 键和旧参数/env/隐式 Secret 调用禁入；当前 Service API、ImageAgent/SourceMedia 及 imgproxy 两 Secret 映射仍必需。保留原 AST/失败关闭辅助，原全部函数逐字不变；历史 docs/合法 Secret 名称不做宽前缀禁入 |
| `TestListingKitWorkloadsUseLeastPrivilegeSharedSecretKeys` 及该文件全部函数、原 release/Compose/CI/安全检查 | 零改、复用原检查；不删除 imgproxy 四项 allowlist，不改 workflow 或检查分类 |

未来完整原 PR CI 适用项必须成功；原条件 SKIPPED 不能写成执行成功。必要测试与编译按实际受影响入口，不重复未变化证据，不增加运行验收/内部试用前置。CI 不是用户验收，source retirement 不是环境清理。

## §7 准入、复核与停止点

当前只三文档，**尚未 IMPLEMENTATION_READY**。一个现有 Writer、独立分支/worktree、同一主要 PR；#1027/#1028 原 owner 不接管。无新实现者、runner 或治理工程。适用独立设计复核须检查 §2 前批 KEEP、§3 消费者、§4.2 两 Secret 名称与显式凭据边界、§5 完整闭包、§6.1 原断言。

先完成复核并处置真实 finding，再交用户确认本设计；两项齐备并将准入写回 #301/PR、读回明确 **IMPLEMENTATION_READY** 后才能修改正式源码。默认最多两轮正常架构复核；非 Blocker 不无限重开设计。设计和源码最终复核分别绑定准确 HEAD；未经用户本轮授权不默认新增 AI 互验线程。GitHub 对同头复核 Running 时不重复触发；失败可一次重试，仍无有效结论时报告，替代子代理另获授权。

实现后仍须完整真实 diff 最终交付检查，只有变化增量需要追加复核。设计确认不授权合并/部署/关单，合并另按最终 expected-head 正常授权；实际 merge SHA 的完整 push/main CI 单独确认。#301 始终 **OPEN**，只更新其当前段；从“以下保留 #726 已合并批次与历史证据，不扩大本轮权限。”起逐字保留，≤65536 字符。无需为本设计派生运行任务或内部试用前置。
