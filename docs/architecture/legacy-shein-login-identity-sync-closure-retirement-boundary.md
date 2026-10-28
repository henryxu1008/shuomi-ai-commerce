> #301 / #985 **USER_CONFIRMED / IMPLEMENTATION_READY / SOURCE_IMPLEMENTED / IN_REVIEW**：用户已确认冻结设计 `8fd07fd3af2a90a314ff7fdc8f0410a507895d34` 的 §1–§7（含§4.2/§6.1）。独立 SHEIN 登录及登录后 Store ID 回写、专属 Cookie/Profile/Redis/执行控制表源码和旧 session API 整批退休；保留 Store 观察同步、官方授权与当前能力。原安全/并发断言按§6.1改接，CI仅去旧worker build一行并删其专属deploy workflow，其余 workflow/inventory/保护规则保持。实施期单行 core_alignment 尾部例外已单独获用户批准并列入矩阵。无既有实例/数据/凭据/进程操作，无试用新前置，#301 OPEN。准确源码HEAD/必要自检/原CI/最终复核留同PR，不等于运行或产品验收，合并/部署/关单另授权。以下冻结设计正文保留，阶段标签由本段替代。

# 独立 SHEIN 登录与身份回写完整退休设计

本批只做一件事：**删除已不再需要的独立 SHEIN 浏览器登录及登录后 Store 身份回写；保留 Store 观察同步。** 当前是设计交付，尚未删除代码，也不关闭或清理任何既有运行资源。

状态：**DESIGN_ONLY / SOURCE_NOT_STARTED / AWAITING_REVIEW_AND_USER_CONFIRMATION**。父目标 [#301](https://github.com/qq550723504/shuomi-ai-commerce/issues/301) 保持 OPEN。调查基线 main `e3a8975441539d2686dea4a85f381bef8f1ce073`。

## 1. 用户结果与 Design Basis

使用者是当前 Store、采集和图像能力的用户及维护者。交付结果是仓库不再构建、装配或宣称支持独立 SHEIN 登录 worker；验证码等待、私有 Cookie/Profile、登录执行归属和登录后商家 ID 写回一起退出。用户仍通过当前官方授权连接使用 Store，观察同步仍读取商品、订单等平台事实；不新增操作步骤或内部试用前置。

本轮用户先确认“除了 Store 观察同步，其他两个应该用不到”，随后同意完整退休设计。该决定准确替代以前“保留独立 SHEIN 登录与其 Store 同步”的要求，**不取消当前 Store 观察同步、官方授权或当前采集浏览器**。这也不是恢复之前已撤回的“所有 Cookie 接口都退休”的泛化指令；本批只删除实际查明仅供本 worker 使用的闭包。

本次文档阶段 **Design Basis: N/A / Ready**，只允许本文、架构索引和 Legacy Register 三份文档。未来实施采用 **Independent Architecture / NOT_READY**：结束一项独立执行能力及其可选持久化事实 owner，涉及 Redis、Profile 和 Store 控制表，并超过 30 文件。适用独立设计复核、用户确认 §1–§7（尤其 §6.1 的 workflow 例外）、Issue/PR 的 IMPLEMENTATION_READY 读回完成后，才允许修改生产路径。

依据为本轮用户决定、[greenfield 产品基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、[Store 观察同步](store-center-platform-observations-v1.md)和[其当前运行装配](store-observations-preparation-runtime.md)。不改变 UI、IA 或交互，Figma N/A。

- Must：完整退出专属源码、测试、配置、构建/部署声明及当前文档；Store 观察同步、官方连接、采集、图像、身份/组织隔离和有效规则保持。
- Should：复用当前 Store 的真实接口承接有效安全断言，不造新运行框架、兼容 alias、替代登录服务或第二身份事实源。
- Out of Scope：真实登录/HTTP/provider 调用、既有数据库/Redis/Profile/凭据/进程操作、迁移/清理、部署、依赖升级、保护规则和试用准入变化。inventory 全文零改。
- Accepted Risks：无新增风险接受；运行状态 UNKNOWN、真实环境和产品验收 NOT_RUN。源码退休不等于既有运行实例停用，更不等于生产发布已就绪。

## 2. 前批 KEEP 替代与 Legacy 决策

| 旧依据 | 本批确认后的准确替代 | 继续有效的边界 |
| --- | --- | --- |
| #722 的 SHEIN worker 身份/持久化设计、#863 当前身份重接与可选执行控制表 | 独立 CLI、登录服务、Store 身份回写及其控制表源码整退；不再新增 fresh worker 安装 | 当前身份/组织/成员、Store 主事实和官方连接继续由原 owner 负责；旧数字账号不恢复 |
| #926 的 SessionTransport、商家/仓库查询及合同 KEEP；#968 的 worker/API 尾部 KEEP | 最后消费者随 worker 退出，`internal/shein` 剩余整根退休，不另建 transport/DTO 兼容层 | `internal/integration/shein` 的官方 API、marketplace 当前值合同/规则/草稿与 JSON 保留 |
| #939 的 RedisConfig KEEP | 仅独立 SHEIN 私有配置使用的 `core/config/type_redis.go` 随最后消费者退出 | 当前 Redis 驱动/实际消费者和其自有配置不改，不将类型搬入新公共门面 |
| #946/#947/#948 等保留实际浏览器行为并列出的 SHEIN 调用点 | 仅删 SHEIN 专属调用、配置和断言 | 共享浏览器管理、实际指纹、代理评分、采集、PoolManager deadline/取消不改 |
| #854 源级收口、历次 #301 批次和索引/Register 中泛称“保留独立 SHEIN 登录/Store 同步” | 本文仅对独立登录及其身份回写两项拥有新的当前决定；其余历史范围不重写 | Store 观察同步和其他当前能力继续保留；历史勾选、计数、失败证据不改 |

Legacy decision: **RETIRE**。Reusable behavior: 当前 Store、官方连接、观察同步、身份授权和共享浏览器已有正确 owner；本批不需要新增生产 EXTRACT。Current owner: `storecenter`、`app/storecenter`、`app/storeobservations`、`app/runtime/storeobservations`、`integration/shein` 和当前采集 owner。Cutover/deletion condition: 设计复核与用户确认后，在同一主要 PR 完成 §5/§6.1、当前能力检查和准确 HEAD CI；合并、运行资源停用和 #301 关闭均另行授权。

本轮不再把已撤回的 worker KEEP 当作当前产品需求。不会复制它的状态机到 Store observations，也不会为“保留登录能力”留下 wrapper、空 facade、fallback 或双写。

## 3. 实际消费者与完整调用闭包

在基线解析全部 **3108 个 tracked Go 文件**的真实 import、alias、selector 和声明，包含测试/debug/fixture/构建条件文件；再核对同包、非 Go、构建、样例和文档引用。静态消费者证据不能证明真实实例正在或没有运行；本次不进行运行探测。

```text
退休：cmd/shein-login-worker
  -> app/runtime/sheinloginworker -> sheinlogin/bootstrap
  -> live 身份/Store 授权 -> 私有配置及 Redis/Profile
  -> login/status/verify/cancel/activate/quiesce + RunWorker
  -> Cookie session -> 商家身份查询
  -> Store RecordSheinLoginIdentity -> 专属 fence / audit receipt

保留：current-application -> 当前 Store/official connection/组织授权
  -> app/storeobservations -> runtime/storeobservations 的现有 Temporal 工作流
  -> integration/shein 官方授权 API -> 当前观察事实与 checkpoint
```

| 对象 | 实际消费者和副作用 | 处置 |
| --- | --- | --- |
| `app/runtime/sheinloginworker` | 只有 worker CLI；不由 current-application/HTTP/UI 装配 | CLI/runtime/bootstrap 一起退出 |
| `sheinlogin` | 生产包外只有 CLI/runtime/bootstrap；runtime 的 health 是本 worker 健康端口，不是当前 Store API | 全部登录、验证码、取消、私有配置、Redis stream/Lua/attempt、Profile/automation 及专属测试退出 |
| `storecenter.SheinLoginRepository` | 唯一生产装配在 worker runtime；`RecordSheinLoginIdentity` 唯一生产调用在 worker 的 pending/replay 路径 | 独立 repository、控制表/安装器/验权、三 audit action 及其专属分支退出 |
| `internal/shein/client`、`api/other`、`api/warehouse` | 所有包外 import 均来自 `sheinlogin/session_identity.go`，没有当前官方 API/草稿 consumer | CookieData、SessionTransport、三个读 endpoint、supplier DTO、warehouse DTO/转换及专属测试整退 |
| `core/config.RedisConfig` | 包外只被 SHEIN Config/Store 生产与专属测试消费；本包无引用 | `type_redis.go` 整退；同名通用 Redis 能力不因此退出 |
| shared browser、config、ZITADEL、authidentity/authz、membership、database、appenv、workbenchcontext | 都有其他现役生产消费者 | 保留原 owner；不借本批清理通用 helper、依赖或权限 |
| schema-init/Makefile/CI/部署声明/原 guard | 仍声明可安装、编译或必须保留 worker，属于闭包的一部分 | §5/§6.1 精确替换，不能靠缺文件 skip 通过 |

已只读核对活跃 [#956](https://github.com/qq550723504/shuomi-ai-commerce/pull/956) 的准确 head `88bfd9163c632dab48d097658c5fb55f8fd94066` 与 30 文件 diff：它的浏览器操作/审计不消费本 worker 或旧 SessionTransport；但与本批的 `cmd/store-center-schema-init/main.go`、`storecenter/audit.go`、`current_schema.go` 重叠。其新增浏览器合同、审计和 schema 能力不属于本批删除范围。若它先合入，形成最终候选时检查实际组合；只删除 SHEIN 登录项，不回退其能力、不替另一个 owner 实施未合入代码。它的拟新增 `capabilities.SheinLogin` 排除项须在组合时随已删除字段准确去除；如果出现新的真实 consumer 或需要触碰当前清单外文件，先报告范围变化。

现有架构问题是“当前 Store 观察”与“登录后回写 Store ID”长期共用一个简称，且原检查把后者当作产品需求。当前链路已各有 owner，正确处理是结束独立登录事实 owner，保留真实观察链，不用改名或新门面延续旧能力。

## 4. Owner、持久化与外部副作用边界

### 4.1 退出的 owner 与保留事实

独立 worker 的 scope 是 Organization/Store UUID/installation/binding revision/fingerprint/generation；Redis 存 attempt、stream、验证码/取消与 Cookie，Profile 存浏览器状态，Store 可选表 `workbench_store_shein_login_bindings` 存执行归属。它们随源码退出，不转移给观察同步，不自动接管 UNKNOWN，不新增恢复者。

原 `RecordSheinLoginIdentity` 只在当前授权事务中写 Store 的 external_store_id、identity_key、version 和 updated_by/at，并写原审计回执；不写官方 token。退休后该写入入口消失。**这些 Store 主字段不删除、不改名、不清空**，官方授权连接/商家绑定仍由当前 Store owner 完成。当前组织、成员、JWT、计费和平台凭据所有权不变。

worker 专属 `shein_login_identity_sync / shein_login_activate / shein_login_quiesce` 的生产者、常量、allowlist 和 action/outcome 分支一同退休。审计表、字段、索引、通用 Record/Get/GetByStoreID、当前 action 与 credential-shaped 字段拒绝保持。基线通用读取按调用方指定 action 查询，没有遍历旧三 action 的当前列表入口；不新增旧 action 的兼容解码或历史重放器。旧物理记录不会被删除；本轮不承诺历史 worker 回执仍能由已退出的 typed API 读取。若发现具体当前外部观察合同，按 Hard-Cut 停下来报告，不自行兼容或清数据。

### 4.2 Schema、JSON/hash 和真实资源

1. 删专属 DDL、安装/grant/权限验证函数，删 schema-init 的 `-shein-login-worker` 分支及只供它使用的 `runWithCapabilities`。当前 Store 基础、observations、Cockpit、TikTok/Walmart 和 core alignment 模式/角色/拒绝组合保持；传已退休 flag 由标准 flag parser 拒绝，不增加 fallback。
2. `RuntimeCapabilities` 仅移除 SheinLogin 字段；`currentColumnNames` 仅移除专属表一行；`VerifyCurrentSchemaForCapabilities` 移除对应 verify 分支；`verifyCoreSchema` 移除该表的 capability skip。当前表/列/约束/默认值/索引、schema version、原 SQL 查询/失败关闭和其余 capability 分支不变。原 base installer 本来不创建该可选表；未来全新安装不再有其显式安装入口。
3. 不加 DropTable、DROP ROLE、REVOKE、UPDATE/DELETE、迁移/回填或旧新数据库桥。既有控制表、审计行、Store 字段、角色/grants、Redis namespace 和 Profile/证书/密钥文件全部保持。删除仓库中的样例/声明不等于执行对应资源动作。
4. 本批退出的私有配置 JSON、Cookie/browser state、attempt 和控制命令 DTO 没有当前公共领域合同消费者，不搬进新 owner。当前来源/商品/草稿/预览、SourceIdentity 四零值槽位、官方凭据/商家绑定、观察事实与 checkpoint 的 JSON/hash/规范化/版本、已保存载荷零改，不重算 hash。
5. 不启动 CLI、health、浏览器或 Redis 消费者，不读真实 token/config/profile/cookie/轨迹，不连接既有数据库。设计只读 tracked 源码和 GitHub 元数据；实施仅合成 fixture/任务临时资源内做批准的必要检查，不能借测试默认配置连接真实环境。
6. 源码删除可防止未来构建/安装该 worker，**不能证明已运行进程或已有权限已停止**。本批交付明确运行停用 NOT_RUN/NOT_AUTHORIZED；不提供自动接管、kill/unlock、清理脚本或把停用验收变成内部试用前置。

## 5. 完整删除与保留矩阵

### 5.1 整根和整文件退休

以下 51 个已跟踪工件是准确整退集，不按文件数拆批；根目录退出后不得留 alias、占位 Go 包、注释-only facade 或重新命名副本：

```text
cmd/shein-login-worker/main.go
internal/app/runtime/sheinloginworker/options.go
internal/app/runtime/sheinloginworker/runtime.go
internal/app/runtime/sheinloginworker/runtime_test.go
internal/shein/api/other/interface.go
internal/shein/api/warehouse/address_conversion.go
internal/shein/api/warehouse/address_conversion_test.go
internal/shein/api/warehouse/interface.go
internal/shein/client/cookie_values.go
internal/shein/client/endpoint.go
internal/shein/client/endpoints.go
internal/shein/client/session_transport.go
internal/shein/client/session_transport_test.go
internal/sheinlogin/authorization.go
internal/sheinlogin/authorization_readiness_test.go
internal/sheinlogin/authorization_test.go
internal/sheinlogin/automation.go
internal/sheinlogin/automation_artifacts.go
internal/sheinlogin/automation_browser.go
internal/sheinlogin/automation_login_flow.go
internal/sheinlogin/automation_network.go
internal/sheinlogin/automation_page.go
internal/sheinlogin/automation_test.go
internal/sheinlogin/automation_verify.go
internal/sheinlogin/bindings.go
internal/sheinlogin/bindings_test.go
internal/sheinlogin/bootstrap/build.go
internal/sheinlogin/bootstrap/build_test.go
internal/sheinlogin/browser_state.go
internal/sheinlogin/current_service_test.go
internal/sheinlogin/current_store_test.go
internal/sheinlogin/logger_port.go
internal/sheinlogin/logger_port_test.go
internal/sheinlogin/native_redis_integration_test.go
internal/sheinlogin/profile_owner.go
internal/sheinlogin/profile_runtime.go
internal/sheinlogin/profile_runtime_test.go
internal/sheinlogin/service.go
internal/sheinlogin/session_identity.go
internal/sheinlogin/store.go
internal/sheinlogin/types.go
internal/sheinlogin/worker.go
internal/core/config/type_redis.go
internal/storecenter/shein_login.go
internal/storecenter/shein_login_schema.go
internal/storecenter/shein_login_test.go
internal/storecenter/shein_login_postgres_test.go
config/shein-login-worker.current.example.json
deployments/docker/Dockerfile.shein-login-worker
deployments/kubernetes/listingkit-workbench/base/shein-login-worker-deployment.yaml
.github/workflows/shein-login-worker-deploy.yml
```

### 5.2 混合文件精确改动

| 文件 | 允许改动 | 必须保留 |
| --- | --- | --- |
| `cmd/store-center-schema-init/main.go` | §4.2 的 flag、组合条件和专属 wrapper | 其余模式、配置读取防护、权限/事务/deadline/观测安装原行为 |
| `internal/storecenter/current_schema.go` | §4.2 四处专属 capability/schema 项 | 所有当前 Store/official/observations/Cockpit 表和安全验证 |
| `internal/storecenter/core_alignment.go`（实施期用户准确例外） | 仅从 verifyUnchangedCoreContract 的 coreTable 条件删除 `&& table != sheinLoginTable`；退出表已不在 currentColumnNames，冗余排除项失去常量 | 原表筛选、所有约束/index drift/权限/SQL/版本/事务及其它行为保持；[准确提案与根因](https://github.com/qq550723504/shuomi-ai-commerce/pull/985#issuecomment-6462215161)，用户已明确批准 |
| `internal/storecenter/audit.go` | §4.1 三常量、三个 action 分支、allowlist 三项及失用 import | 通用审计所有其它语义/shape、作用域、幂等与敏感字段拒绝 |
| `Makefile` | `.PHONY`、FAST_TEST_PACKAGES、help、build-all 中 worker 项及 build-shein-login-worker rule | 当前应用/原有效测试/race/health/lint 构建规则；不运行 clean |
| `.github/workflows/ci.yml` | 只删 `go build ./cmd/shein-login-worker` 一行 | 当前应用 build 和完整原测试/集成/权限/race/Required Gate/触发/permissions |
| `deployments/kubernetes/listingkit-workbench/base/kustomization.yaml` | 只删 worker resource 一行 | UI/imgproxy/现役图像 worker 和其余 resource/overlay |
| 同目录 `configmap.yaml` | 只删三个 SHEIN_LOGIN_SERVICE headless/force/execution 键及各自专属注释 | 所有其它键值；不连集群 apply |
| 同目录 `secret.example.yaml` | 删 SHEIN_APP_ID/APP_SECRET 和四个 SHEIN_LOGIN_SERVICE BASE_URL/SHARED_KEY/TENANT_ID/IDENTIFIER 样例键，改专属 worker 注释 | 其余原键值和 least-privilege 注释；不读取真实 Secret |
| `scripts/start-listingkit-local-ui.ps1`、`.Tests.ps1` | 只删隔离验收分支设置 SHEIN_LOGIN_API_BASE 的一行及对应 env cleanup/期望项 | 当前 UI/Auth.js/handoff/loopback 防护和其余全部断言 |
| `internal/listingadmin/current_store_login_concurrency_test.go` → `current_store_concurrency_test.go` | §6.1 将有效并发/隔离/重放断言改接当前 Store repository/audit，删专属登录 fixture | 该目录维持 test-only，两个原 workflow 的 race 路径可继续执行有效测试，无生产包/兼容层 |
| §6.1 指定的六个 root 测试文件 | 准确转换 worker KEEP/缺失根遍历和构建目标，加入闭包禁入 | 无缺失 skip、无降级必需/安全检查，其余算法/断言零改 |

### 5.3 文档闭包与历史

实施阶段当前文档改真实入口，禁止留下可执行的 worker 安装指引：

```text
docs/architecture/legacy-shein-login-identity-sync-closure-retirement-boundary.md
docs/architecture/README.md
docs/refactoring/legacy-register.md
docs/development/repository-structure.md
docs/architecture/project-boundaries.md
docs/architecture/architecture-review-checklist.md
docs/operations/shein-login-worker-current.md
deployments/kubernetes/listingkit-workbench/README.md
deployments/kubernetes/shein-listing/README.md
```

`shein-login-worker-current.md` 转成简短退休入口，明确原命令/样例/部署已退出、Store 观察同步的正确文档及真实资源 NOT_RUN，不保留活跃安装步骤，也不删证据所在 Git 历史。

以下直接冲突设计只加精确替代顶注，起点 main 完整原正文逐字保留为后缀：

```text
docs/architecture/shein-worker-identity-persistence-boundary.md
docs/architecture/shein-worker-numeric-persistence-retirement-v1.md
docs/architecture/legacy-platform-api-closure-retirement-boundary.md
docs/architecture/legacy-global-config-logging-tail-closure-retirement-boundary.md
docs/architecture/legacy-shein-draft-support-closure-retirement-boundary.md
docs/architecture/legacy-retirement-parent-closure-boundary.md
docs/architecture/release-identity-preflight-retirement.md
docs/superpowers/specs/2026-08-02-shein-login-cancellation-design.md
docs/superpowers/plans/2026-08-02-shein-login-cancellation.md
docs/superpowers/plans/2026-08-02-shein-login-cookie-export.md
docs/superpowers/plans/2026-08-02-cookie-availability-fix.md
```

其他前批设计/历史报告中的笼统 worker KEEP 由 §2 与 Register 新当前段统一替代，不为它们机械追加上百条重复顶注，不改历史数字/勾选。**external-client-boundary-inventory.md 全文零改**：当前热点仅作历史上下文，相关原 guard 名保持并转换为更强退休禁入，CURRENT 消费者表没有来自本次退出包的 logger/OpenAI 行，无需校准。Code Health 历史、计数、ceilings、依赖/lock、保护规则不改；真实 CI 如证实另有遗漏，给出准确根因和补丁再请求例外，不能自动放宽。

### 5.4 明确保留且不改的能力

`app/runtime/storeobservations`、`app/runtime/currentapplication/store_observations.go`、`app/storeobservations`、Store observation repository/schema、`integration/shein` 官方客户端、当前 official application/connection/token/merchant binding、Store CRUD/metadata/member grants/组织权限和账务；当前 AI/图像/资产批准/SDS/POD/商品草稿预览与 JSON/hash/版本；Amazon 原浏览器与公共1688/批量/扩展采集及 debug；共享 browser/config/HTTP/TLS/Redis/database/logging/Retry/PoolManager deadline、取消与 UNKNOWN。不因英文 `Warehouse`、`Store`、`Session` 同名而误删 marketplace 或官方 DTO。

## 6. 原检查与验证

### 6.1 冻结的检查转换及 workflow 例外提案

这部分是未来整批实施的准确批准范围。**当前设计阶段没有改任何测试、workflow、inventory 或生产源码。** 以“workflow 零改”删 worker 会留下必然失败的构建与仍可发布的旧入口，因此明确请求用户在确认设计时允许且仅允许：

1. `ci.yml` 删除 §5.2 的一个旧 build 行；
2. 删除专属 `shein-login-worker-deploy.yml` 整文件；
3. **`commercial-readiness.yml` 全文零改**，保留 `make build-all`、完整测试及 `go test -race ./internal/listingadmin`；其余 workflow 零改。两套通用 workflow 的必需/安全检查均保留，不改保护规则，不触发任何部署。

实施期准确例外：用户已批准仅删除 `internal/storecenter/core_alignment.go` 中唯一的 `&& table != sheinLoginTable`。该表已从 currentColumnNames 退出，保留其余 coreTable/SQL/权限/约束/索引/版本断言，不恢复旧常量；具体根因和未应用提案见 [PR 评论](https://github.com/qq550723504/shuomi-ai-commerce/pull/985#issuecomment-6462215161)。此项不改变上述 workflow 两项边界。

实施期第二项准确检查例外（用户已批准）：仅在原 `tests/import_boundaries_test.go` 的 TestProjectBoundaryDomainsDoNotImportListingKitFacade 增加旧 SHEIN 根缺席断言并从域列表删该旧根；TestSheinRetiredManagementImportsStayBlocked / TestSheinOpenAIImportsStayAllowlisted 改为根缺席 + 原 assertNoBannedImportPrefixesBelow 从保留 internal 根检查，旧专属 allowlist 随根退出。其余当前域/原禁令/算法不变，不加 missing-skip；[完整提案与真实失败证据](https://github.com/qq550723504/shuomi-ai-commerce/pull/985#issuecomment-6462341589)。此项仅校准本表遗漏的三函数，不改变生产、workflow/inventory 或其它检查。

六个 root 测试文件的准确边界：

| 文件/函数 | 允许转换 | 保留约束 |
| --- | --- | --- |
| `tests/repository_structure_test.go` / maintainedOperationalCommands | 只删 worker 一项 | 对实际命令的维护集合/结构算法不变 |
| `tests/commercial_readiness_workflow_test.go` / TestListingKitRetainedWorkloadsExcludeInvitationCredentials、TestRetiredTencentSMSCredentialsAndRouteAreAbsent | 两处 workload 遍历仅删 worker manifest | 其余 workload/共享 Secret/邀请与旧 SMS 凭据拒绝不变 |
| `tests/listingkit_secret_boundary_test.go` / TestListingKitWorkloadsUseLeastPrivilegeSharedSecretKeys、TestSheinWorkerUsesOnlyFreshPrivateFileInputs | 前者删除 worker map 项；后者从必须挂载私有文件改为 worker manifest/Dockerfile/专属 deploy workflow 必须不存在 | 其余 workload least-privilege 与所有共享凭据断言不变；不添加 missing-skip |
| `tests/import_boundaries_test.go` / TestSheinLoginReadsIdentityFromNeutralContext、TestSheinLoginBootstrapRetiredManagementImportsStayBlocked、TestSheinLoginUsesManagementAPIPortAliases、TestSheinLoginServiceRetiredManagementImportsStayBlocked | 保持已有测试名，退出“读缺失文件/根”的旧 KEEP；从保留 `internal` 根失败关闭地遍历真实 imports，显式断言 worker/service/bootstrap 根缺席并禁旧 owner 再入 | 不影响当前 neutral identity/context/授权检查；managed 旧 guard 不改，不放行 management |
| 同文件 / TestListingKitNonAPISheinImportsStayAllowlisted、TestPublishingSheinNonAPISheinImportsStayAllowlisted、businessHTTPPackages、TestAppHTTPAPIListingKitSupportImportsStayAllowlisted | 只删除已经无合法 consumer 的旧 shein/client、sheinlogin allowlist/业务包枚举项 | 原遍历、import 解码、失败关闭及其它禁令零改；其它负向旧包字符串保留 |
| `tests/legacy_hard_cut_test.go` / TestSheinWorkerNumericPersistenceClosureStaysRetired、TestUnusedSheinManagedProviderAndVerifyDTOStayRetired | 加整根缺席和旧 import 禁入；已退出根内字段扫描可收为根缺席，原数字管理/DTO 禁令继续保留 | 不把 source absence 当 skip；test-only listingadmin 不得恢复生产源码 |
| `tests/listingkit_backend_retirement_test.go` / 下列完整函数表 | 仅删除所列旧 worker 的 KEEP 项，显式加强整根/import/命令/部署禁入 | 其余现役 owner、原负向断言、源码解析/错误处理和全部其它测试不变 |

backend 文件逐项冻结：

- `TestLegacyLocalAgentSourcePolicyClosureIsRetired`：删 worker 的 NewService/Execute 必需调用两行及 store.go/CLI 必需文件项。
- `TestLegacyPublishingSupportPolicyClosureIsRetired`、`TestLegacySheinDraftAssemblyClosureIsRetired`、`TestLegacyIdleProviderBootstrapClosureIsRetired`：仅删 worker 必需声明表行。
- `TestLegacyOpenAIAdapterClosureIsRetired`：删 worker 必需声明表行及遍历条件中专属 worker owner 白名单；保留 proposal 和所有原禁止声明/导入。
- `TestLegacyCrawlerFacadeDiagnosticsClosureIsRetired`：删 worker 四方法必需表、两个 fence 调用必需项及专属 owner 枚举；crawler/capturedata 的全部算法与取消检查不变。
- `TestLegacyBrowserManagementTailClosureIsRetired`：只删 automation.go 的 NewManager/SetFingerprint 必需项，真实采集指纹/代理评分/质量重试的原检查不变。
- `assertLegacyLoggerFacadeRetired`：businessRoots 可删已经整退的 shein/client、shein/content、sheinlogin 三项，其余边界扫描保持；旧日志禁令不得取消。
- `TestLegacyLoggerFacadeClosureIsRetired`、`TestLegacyListingPreviewDownloadClosureIsRetired`、`TestLegacySDSRuntimeClosureIsRetired`、`TestLegacyRedisProxyAndSharedClosureIsRetired`、`TestLegacyHostConfigFeatureFlagTracingClosureIsRetired`、`TestLegacyWatermarkClosureIsRetired`、`TestCurrentCommandsDoNotCompileLegacyBusinessBackends`：各自命令列表只删 shein-login-worker；原 retained 命令必须成功 go list，全部旧依赖禁令保持，并纳入本次退出根。
- `TestLegacyRuntimeSkeletonAndIdleConfigClosureIsRetired`：只从 core/config 的 retained 表删 RedisConfig 及 Host/Port/Password/DB/PoolSize 六项，将类型/专属文件加入退休禁入；其余配置校验/原字段/加载、JSON/图像解码和所有安全断言不变。
- `TestRetiredListingOperationsClosure`：只删 worker manifest 必需 map 项，原 storage 告警/namespace 窄路由、现役资源和负向断言不变。
- `TestStandaloneSheinWorkerDoesNotCompileCurrentHTTPApplication`：保留测试名；从编译已退 CLI 改为精确根缺席及保留入口的旧依赖禁入；原共享 HTTP worker/旧 HTTP DTO 缺席检查继续执行。
- `TestRetiredListingCLIsHaveNoBuildOrDeploymentEntrypoints`：CI 必需 build 表只剩当前应用；把 worker 加入旧 CLI/Makefile/构建禁入集合，原其它 CLI/config/auto-shard 禁令全留。
- `assertLegacyPlatformAPICallClosureRetired`：retiredRoots 增加本轮退出的 other/warehouse/client；requiredByFile 只删它们五个 KEEP 行，endpointSources 两行转成明确文件/声明缺席，不再读缺失文件；原配置/当前 DTO/全部旧 API 禁令保持。
- `TestLegacySheinDraftSupportClosureIsRetired`：当前文件必需表只删八个 worker/API 专属项；draft builder、严格 decode、清理 Input 精确五字段及所有其它断言不变。
- `TestLegacyImageAgentHTTPRuntimeClosureIsRetired`、`TestLegacyAIRoutingImagePolicyClosureIsRetired`、`TestLegacySourceNormalizationFactsClosureIsRetired`：各自文件存在/读取表只删本轮退出 CLI/service/store 文件，现役 worker/WithWorker canary、AI、SourceIdentity/JSON/hash/全部原断言保持。
- `TestLegacyAssetSourceOpenAITailClosureIsRetired`：删 worker Execute 必需表和 CLI/store 文件必需项，AI 当前合同/hash/来源保存/显式资产批准原断言保持。
- 在该现有文件增加一个本轮完整闭包 guard：强制退出根/专属能力符号/flag/构建部署缺席，扫描真实 import 字面量含转义而非只 grep；明确当前应用、Store 观察/官方 API、当前清理/采集、shared browser 等 KEEP。复用现有 AST/helper，不另建检查框架。

`internal/listingadmin` 的原 race 目标必须仍执行有效测试：将登录专属测试改为当前 `GormStoreRepository`/`GormAuditRepository` 的真实合成 fixture，不创建 SheinLogin 替身。保留并验证同 org 外部身份冲突无更改、跨 org 读写拒绝、不同 org 同外部身份合法、并发相同创建只落一 Store/创建 audit、相同 key 不同 payload 拒绝、较新版本不能被旧重放覆盖及 connection_ref 保持。使用 `CreateOrReplay` 的**原 ErrAlreadyExists 口径**（不得误用旧 worker ErrIdempotencyConflict），以及 `Get/Save/EditBasic` 的原版本合同；旧 login receipt 的 version=2、claim/fence 和回执协议本身随能力退休，不伪造新通用协议承接它。原 Store/audit 全部其它测试与生产方法零改。

新 guard 必须在删除前真实 RED、完成后 GREEN；安全断言改接先证明能捕获相应违规。其它新增测试仅用于本批真实行为，不镜像实现或增加验收工具。未列明函数/检查逻辑零改；若原 CI 证实遗漏，先分类、同根核对，再给精确修订申请，不用扩大 allowlist 或放宽检查自救。

### 6.2 必要验证与证据分层

设计阶段只校验三文档、链接、准确清单、完整历史后缀、diff --check，原 PR CI 照常运行；不跑业务应用/登录/环境测试。实施阶段按风险做本轮 root guard、Store/current schema/CLI 原测试、改接后的 listingadmin race，以及当前应用、image-agent-temporal-worker、store-center-schema-init、两个实际采集入口的仅编译（不执行）。必要本地测试后由稳定准确 HEAD 原 CI 完整验证，不机械重复全套。

当前 Store/audit/official/observations 的 PostgreSQL 权限与 schema 断言继续由原 CI 在任务自有 fixture 执行；不为本批创建新通用 runner、故障平台或真实验收实例。源码/静态/合成 fixture、PR CI、实际 merge-SHA push/main CI 和真实运行/产品验收分别记录。禁止把 PR 绿色或 Required Gate 单独成功当成整个 main run 成功。

## 7. 准入、复核和交付停止点

当前只三文档，SOURCE_NOT_STARTED。先完成适用独立设计复核，将 finding 分类为 BLOCKER / IMPLEMENTATION_TEST / BACKLOG / ACCEPTED_RISK / NOT_APPLICABLE；最多两轮正常架构复核，不因理论 hardening 无限重开。新 BLOCKER 必须对应当前 Must 和实际后果。

设计原 CI 和复核完成后交用户确认完整范围，尤其 §6.1 的 workflow 两项准确例外；确认并读回 IMPLEMENTATION_READY 后，单一实现者在同一 PR 连续完成全闭包。未授权合并/部署/关单。源码完成后一次集中最终交付复核；finding 修复仅复核相关增量，不让无关 main 变化造成重复全量评审。

**未经用户本轮授权不默认新增 AI 互验线程；设计复核与源码最终复核授权分开。** GitHub 复核启动失败不记成 PASS；同 SHA 已运行不重复触发，最多一次必要重试。若需独立子代理，先准备可审设计/准确源码 HEAD 和现有证据，再单独询问该只读范围，不默认增加实现者。

交付说明优先让用户看清“登录及 ID 回写退出、Store 观察继续”。设计未确认不实施；实现未独立验证不称验收通过；合并需准确 HEAD 明确授权并遵守保护；获准合并后才核对实际 merge SHA 的完整 push/main CI。#301 始终 OPEN，真实实例/数据/凭据/进程停用保持 NOT_RUN/NOT_AUTHORIZED，不增加内部试用前置。
