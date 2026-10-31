> #1032 当前实施：用户已确认准确设计 7b761786ef998d3d4bb9cf72d5ae214fae35431e 的 §1–§7（含 §4.2、§6.1）；准入 USER_CONFIRMED / IMPLEMENTATION_READY。按67路径整批抽取四项当前 Image 维护操作、退休两旧 CLI 与聚合建表闭包，不留 alias。原冻结设计/导航原文逐字保留于下；滚动 CI/复核证据见 PR #1032。既有 Goose 记录不动，workflow/inventory 零改，不操作既有资源，#301 保持 OPEN。

# 旧 schema CLI 与跨域聚合建表整链退休设计

Refs #301。Design Basis: **Independent Architecture**。当前 **DESIGN_ONLY / SOURCE_NOT_STARTED / NOT_READY**；适用独立设计复核、用户确认 §1–§7 和 Issue/PR 准入读回完成前，不修改正式源码。三组闭包在一个主要 PR 完成，不机械拆批。

## §1 用户结果与产品依据

使用者是维护当前安装和初始化的开发者。当前各业务域已有自己的持久化 owner，但两个旧 schema CLI 仍提供跨域聚合建表入口；当前 Organization Image 的四项维护操作还挂在旧 Product Listing 命令下。维护者容易沿旧指引把 Store、Resource 或图像相关表安装到错误的聚合数据库。

本批交付一个清楚的源码结果：

1. 退休 `listingkit-schema-migrate → workbench/schema` 的 Store/audit/Resource 聚合链；当前 Store、Resource 初始化继续使用既有正式 owner。
2. 退休 `product-listing-api-schema-migrate` 的默认宽迁移及其专属 Goose 注册；先将四项有效 Organization Image 操作抽到当前 ImageAgent 维护入口，再整退旧命令和包，不留 alias。
3. 一起收齐镜像编译/打包、现有图像初始化脚本、旧 CLI 独占版本输出、原检查和直接文档尾部。

依据：[Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、[当前图片架构](product-image-agent-v1.md)、[当前图片操作合同](../operations/product-image-agent-v1.md)、[Store 当前装配](store-center-current-application-v1.md)及用户本轮决定。调查基线 `main=750ec7dc7e934dad741edfc47d522d1a4eb53e32`。本批无页面、IA、交互或 Figma 变更，Figma Authority **N/A**。

采用 Independent Architecture，因为维护入口、跨域 DDL 的允许范围和代码 owner 改变，且完整闭包超过30路径；持久化事实仍归原业务域，不新建数据 owner。当前设计提交仅本文件和两导航，未来实施矩阵为 §5 的67路径。

Legacy decision: **EXTRACT | RETIRE**。Reusable behavior：当前 Organization Image 初始化、仅有当前账本时的准备、HTTP/worker 窄授权，以及 Store/Resource 的有效组织约束。Current owner：现有 Image/Asset/AI invocation、Store、Resource owner；维护装配归新的窄 `app/runtime/imageagentschemainit`，不建立通用迁移平台。Cutover/deletion condition：适用设计复核、用户确认和准入完成后，同一 PR 先抽取/改接全部当前消费者，再删除旧根与专属尾部，完成原 CI/最终交付检查。

Out of Scope：真实建库/DDL/grant/迁移/清理，现有 Goose 记录改写、历史数据承接，旧 CLI 兼容层，新 schema 框架/runner，当前 runtime 自动修复，依赖升级、workflow/inventory/保护规则调整。当前采集、图像、身份权限、Store 观察同步和此前全部当前能力保持；不恢复已退休 SHEIN 登录/身份回写，不增加内部试用前置，#301 保持 OPEN。

## §2 前批 KEEP 的准确替代

以下替代待用户确认生效，仅作用于本批准确对象，旧证据保留为历史。

| 原依据 | 本批替代 | 仍保留 |
| --- | --- | --- |
| [#988 订阅退休](legacy-subscription-metering-closure-retirement-boundary.md) §4.1/§5：Workbench Store/audit/Resource 和 workbench scope KEEP | 退出跨域聚合 wrapper、唯一旧 CLI 与专属测试；不再把隔离测试用 AutoMigrate helper 当正式安装入口 | 当前 Store `app/schema/storecenter.Migrate`、Resource 的 `commercial-owner-schema-migrate` 及原表/约束/权限/账本 |
| [#1010 调试/配置](legacy-restored-debug-closure-retirement-boundary.md)、[#846 本地工具](legacy-local-acceptance-closure-retirement-boundary.md) 的两 schema CLI KEEP | 两旧名字/根退出；有效 Image 操作改到新窄命令，当前私密输入和缺参先失败保持 | 当前 SourceAccount 初始化、worker/canary、其余当前命令及配置加载语义 |
| [#822 Amazon](legacy-amazon-listing-closure-retirement-boundary.md)、[#923 Prompt](legacy-tenant-prompt-closure-retirement-boundary.md) 的 productlisting schema/Goose KEEP | 旧聚合 baseline 的源码注册及默认 Run 整退；不再承诺用该注册续跑旧聚合数据库 | 既有表/记录/Goose历史不操作；共享 Goose runner 和其它当前域的版本链不变 |
| [#994 手工图像链](legacy-imageagent-manual-runtime-closure-retirement-boundary.md) 的 Organization Image PG 测试 KEEP | 完整测试随有效操作移动到新 owner，原 gate/断言保留 | 当前物理表名、索引、空库/账本边界、权限和拒绝路径，不恢复旧 v2 effect 表 |
| [#915 宿主 helper](legacy-host-image-helper-closure-retirement-boundary.md) 及 [TEMU 初始化承接注记](legacy-temu-store-init-retirement-boundary.md) | 仅失去唯一 CLI 消费者的 appenv.VersionInfo/PrintVersionInfo 及专属版本注释退出 | ComponentLoggers、SetupLoggerWithLevel、真实平台日志/锁/输出、其它同名 owner |
| [#847 发布执行](legacy-kubernetes-release-execution-closure-retirement-boundary.md)、[#1030 安装链](legacy-browser-install-closure-retirement-boundary.md) 的 Docker/安装 KEEP | 两 Dockerfile 仅把两旧 schema CLI 的 COPY/build/打包换成一个窄 Image 命令 | 当前程序/SDK 安装 stage、镜像基础/版本/标签/端口/启动、生产 NOT_READY 门禁与全部 workflow |
| 其余 §5.3 明列的旧设计/编译清单及旧 schema-unification 设计 | 仅本批路径/版本 helper/旧聚合注册的保留含义被替代 | 其它产品决定、规则、检查与历史计数/勾选，不批量清理旧计划 |

## §3 实际消费者与 owner

调查只读 Git 追踪源码、测试、Docker、脚本、workflow/Makefile 和直接文档；未加载私密 manifest、环境凭据或运行实例。仓库外人工调用两个旧 CLI 的情况为 **UNKNOWN**，不是“确认无人使用”。

| 链：contract → implementation → injection → consumer | 当前事实与设计 |
| --- | --- |
| listingkit CLI → ParseFlags/Run → workbench/schema.AutoMigrateRuntime | 仅支持 workbench；all/commercial/shein-sync 已拒绝。唯一该 schema 的生产 import；内部调用 Store/audit 测试用 AutoMigrate 与 Resource AutoMigrate，把不同 owner 装进同一 DB。整链 RETIRE |
| product-listing CLI 无专项 flag → runtime.Run/runWithDependencies → app/schema/productlisting.Migrate → 2026083001 | 唯一聚合 registry 消费者；同时安装 credential、invocation、Image、Asset、Catalog。当前窄操作不依赖该默认分支。整链 RETIRE |
| 当前 Image 四专项 flag → Grant/WorkerGrant/Initialize/PrepareLedgerOnly | 三项被 account-compose/image-trial-init.sh:76–78 真实调用；ledger-only 有当前操作说明。四项全部 EXTRACT，保持代码行为，不删除能力 |
| 新维护装配 | `cmd/image-agent-schema-init` → `app/runtime/imageagentschemainit` → 现有 currentapplication/configadapter、bounded OpenExistingWritableContext 和既有各 schema/grant 方法。没有旧包 forwarding/wrapper 或默认宽迁移 |
| 当前 Store | `cmd/store-center-schema-init` → `app/schema/storecenter.Migrate` → 当前显式 PostgreSQL schema/版本与窄 grant；Store 观察等仍原 owner。测试用 AutoMigrateStoreRepository/AutoMigrateAuditRepository 仍可供隔离 fixture，不因最后生产 caller 退出而删除 |
| 当前 Resource | `cmd/commercial-owner-schema-migrate` → migrateOwnerSchemas → orgresource.AutoMigrate；money 与 commercial 使用原独立 pool/owner，原账本/事务不变 |
| 当前 Product/Catalog/Asset | 当前 product-acquisition-init、supply-asset-init 及既有正式安装路径继续原样；不由新 Image 命令补 Catalog/Review/Store/Resource 表 |
| 装配尾部 | 两 Dockerfile 是两旧 CLI 的仓库 build/打包 owner；image-trial-init.sh 只消费三专项动作。workflow 仅引用现有 Dockerfile，不直接编译两旧 CLI，Makefile/scripts 无其它运行 caller |
| 版本输出 | appenv.VersionInfo/PrintVersionInfo 唯一生产调用为旧 listingkitschemamigrate；SetupLoggerWithLevel 仍被当前 worker 使用，平台日志/DB Open/OpenShared 仍有真实当前消费者，全部 KEEP |

### §3.1 当前 Image 维护入口

新命令保留原 `-config`、`-current-application-manifest` 及四个专项 flag 的名字/输入语义：

- `-initialize-organization-image-agent`；
- `-prepare-ledger-only-organization-image-agent`；
- `-grant-image-agent-runtime`；
- `-grant-image-agent-worker-runtime`。

保留原缺少/空白 config 在 execute 前失败、help 不执行、专项动作互斥；在 runCommand 调用 execute 之前验证恰好一个专项动作。**只给 config、零专项动作也必须在配置/DB I/O 前失败**；不能恢复旧默认 Run。四专项函数仍执行原绝对路径/不同文件、准确当前 manifest 物理 DB 与非 runtime owner 校验；新命令不增加可选择的表、DB、角色或通用 SQL。

现有图像初始化脚本三行仅替换 executable 名，参数、调用顺序、私有文件、权限和其它 shell 行保持。当前操作说明只替换旧命令名并准确说明四专项入口；不增加用户部署/试用步骤，不执行脚本。

## §4 持久化、Goose 与安全边界

### §4.1 必须保留的不变量

Image/Asset/credential/invocation 仍为同一原物理 Image owner；HTTP/worker/runtime role 不充当 schema owner。两 grant 函数仍用30秒 bounded Context、只开既有准确 DB，分别调用原窄 grant，不加权限、不建表、不合并两个角色。Store、Resource、money、Product 等事实/权限/租户边界、当前 JSON/hash/绑定版本/幂等与 UNKNOWN/恢复、deadline/取消均不变。

初始化和 ledger-only 仍用原2分钟 Context、同一 advisory transaction lock、原事务、空库/精确单表检查；后者仍锁定并验证当前 ai_invocations，保留原 ledger 字节/事实，不重建或解释旧业务数据。原 PostgreSQL 才可执行、原 owner/索引/约束、重复/未知/混合库拒绝均保留。现有 image_agent_v2_* 物理表名仍承载当前模型，本批不改名或清表。

错误、响应丢失、重试、重启、取消和并发继续沿原事务/锁/错误语义；不新增自动恢复或自动重试。提交失败不宣称没有持久化，维护者仍沿原独立授权/核对边界；不新建恢复平台或操作步骤。

### §4.2 Goose 历史与本轮 hard-cut 取舍

| 边界 | 明确决定 |
| --- | --- |
| 旧 `app/schema/productlisting` baseline `2026083001` | 随唯一默认聚合入口整包退出，不重编号、不搬到新 owner、不改成空 migration/别名，也不新建注册继续承接旧聚合库 |
| 既有 `goose_db_version`、已应用记录、业务表和内容 | 不读取、升级、删除、回滚、stamp、reset 或改写；本批没有 DROP/Down/SQL 操作。源码删除不是已有记录被删除 |
| 当前独立域 Goose | `platform/database/migration`、其原 runner/SQLite锁/PostgreSQL session lock/Up测试，当前 Store 专用版本表和各已批准迁移全部零改 |
| 新 Image 维护入口 | 沿现有四项操作的事务/DDL/grant；不引入 Goose registry/版本表，不消费/解释旧2026083001或复用其收据 |
| 全新/当前 ledger-only 安装 | 原空库检查拒绝任何已有业务关系；原 ledger-only 检查只接收精确当前账本，含旧 Goose/未知额外表的库仍拒绝。两个 grant 的原边界不增加额外限制，也不自动迁移任何表 |
| 用户确认本设计的含义 | 不再承诺两个旧 CLI/宽聚合 registry 可执行或可用来续跑旧数据库；已有事实不动，不提供兼容迁移/双读/转发。发现当前外部兼容义务须先报告，不自行接受或建设桥 |
| 已保存 JSON/hash/版本/事实 | 没有业务载荷转换；当前 Image、Asset、Catalog、Store、Resource、身份和其读取/恢复合同原样。迁移代码的删除不授权环境切换或数据清理 |

本轮只设计、未来也只获源级实施授权；实际初始化/grant/迁移/旧文件清理仍需另行范围。既有实例、数据、凭据、缓存、Profile和进程零操作。真实环境/用户验收 **NOT_RUN**；外部旧入口使用 **UNKNOWN**。不把它们转成 PASS 或内部试用前置。

## §5 删除/抽取/保留矩阵：完整67路径

未来实施为17个旧工件退出、6个当前命令/owner源与测试、11个混合源码/测试/装配文件，以及33文档。当前设计只修改本设计与两导航。发现表外消费者、冻结 KEEP 或检查变更先给准确提案，不用“完整闭包”扩大授权。

### §5.1 RETIRE：17旧路径/6旧根

```text
cmd/listingkit-schema-migrate/main.go
cmd/product-listing-api-schema-migrate/config_input_test.go
cmd/product-listing-api-schema-migrate/main.go
cmd/product-listing-api-schema-migrate/main_test.go
internal/app/runtime/listingkitschemamigrate/flags.go
internal/app/runtime/listingkitschemamigrate/options.go
internal/app/runtime/listingkitschemamigrate/runtime.go
internal/app/runtime/listingkitschemamigrate/runtime_test.go
internal/app/runtime/productlistingschemamigrate/organization_image_agent.go
internal/app/runtime/productlistingschemamigrate/organization_image_agent_postgres_test.go
internal/app/runtime/productlistingschemamigrate/runtime.go
internal/app/runtime/productlistingschemamigrate/runtime_test.go
internal/app/schema/productlisting/migrations.go
internal/app/schema/productlisting/runtime.go
internal/app/schema/productlisting/runtime_test.go
internal/workbench/schema/runtime.go
internal/workbench/schema/runtime_test.go
```

两 cmd、两旧 runtime、旧 productlisting schema 与 workbench/schema 六根全部缺席；不保留报错 stub/alias/旧包转发。

### §5.2 EXTRACT / 混合尾部：17路径

| 位置 | 准确处理 |
| --- | --- |
| `cmd/image-agent-schema-init/main.go`、`main_test.go`（2新） | 保留四专项 flag/现有私密输入及 help/缺参/互斥安全，删除旧默认动作，测试仅合成输入/execute spy；不搬旧连端口1的 binary-default 试验 |
| `internal/app/runtime/imageagentschemainit/runtime.go`、`organization.go`、`runtime_test.go`、`organization_postgres_test.go`（4新） | 4公开有效函数、准确 target validator、closeDB、原 Organization事务实现与其有效测试移动；只改 package/import/当前入口说明，不改SQL/角色/时限/锁/函数体行为。旧 Run/Dependencies/MigrateAll、默认 loader试验及其专属 helper不搬 |
| `internal/pkg/appenv/apputil.go` | 仅 VersionInfo/PrintVersionInfo、专属版本分隔注释和 package 描述中的version output退出；两个现役日志函数/所有import逐字保持 |
| `internal/storecenter/gorm_repository_test.go`、`internal/integration/orgresource/gorm_repository_test.go` | §6.1有效组织约束归属转换；仅测试，生产 helper/schema均零改 |
| `tests/listingkit_backend_retirement_test.go`、`tests/import_boundaries_test.go`、`tests/commercial_readiness_workflow_test.go`、`tests/commercial_schema_contract_test.go`、`tests/repository_structure_test.go` | 仅 §6.1指定检查/命令表转换，无 missing-skip/全函数删除 |
| `deployments/docker/Dockerfile.account-compose`、`deployments/docker/Dockerfile.product-listing-api` | 两旧CLI的 COPY/build/运行层COPY与后者chmod条目精确换为一个新窄Image CLI；其它Docker行/stage/SDK/target不变 |
| `deployments/docker/account-compose/image-trial-init.sh` | 仅3行调用的 executable 名替换，原专项参数/顺序/其它行不变 |

### §5.3 文档尾部：33路径

本设计、`docs/architecture/README.md`、`docs/refactoring/legacy-register.md`为设计与导航。`docs/development/repository-structure.md`维护命令列表仅两旧改一新；`docs/operations/product-image-agent-v1.md`两处旧命令名改新，并加准确当前入口注记，其余产品/操作合同保持。

以下28份直接旧 KEEP/旧指引只加本批窄替代顶注，完整原 Git blob（含既有顶注/计数/勾选）逐字保留为后缀；不批量改历史案例或重新启动旧计划：

```text
docs/architecture/legacy-amazon-listing-closure-retirement-boundary.md
docs/architecture/legacy-config-facade-closure-retirement-boundary.md
docs/architecture/legacy-host-image-helper-closure-retirement-boundary.md
docs/architecture/legacy-idle-support-contract-closure-retirement-boundary.md
docs/architecture/legacy-imageagent-manual-runtime-closure-retirement-boundary.md
docs/architecture/legacy-kubernetes-release-execution-closure-retirement-boundary.md
docs/architecture/legacy-local-acceptance-closure-retirement-boundary.md
docs/architecture/legacy-numeric-content-management-closure-retirement-boundary.md
docs/architecture/legacy-numeric-listing-support-closure-retirement-boundary.md
docs/architecture/legacy-numeric-task-execution-retirement-boundary.md
docs/architecture/legacy-numeric-task-persistence-retirement-boundary.md
docs/architecture/legacy-platform-api-closure-retirement-boundary.md
docs/architecture/legacy-restored-debug-closure-retirement-boundary.md
docs/architecture/legacy-retirement-parent-closure-boundary.md
docs/architecture/legacy-runtime-config-closure-retirement-boundary.md
docs/architecture/legacy-shein-policy-contract-closure-retirement-boundary.md
docs/architecture/legacy-shein-publishing-runtime-closure-retirement-boundary.md
docs/architecture/legacy-subscription-metering-closure-retirement-boundary.md
docs/architecture/legacy-task-identity-mapping-closure-retirement-boundary.md
docs/architecture/legacy-tenant-prompt-closure-retirement-boundary.md
docs/architecture/legacy-browser-install-closure-retirement-boundary.md
docs/architecture/legacy-temu-store-init-retirement-boundary.md
docs/operations/acceptance-environment-split.md
docs/operations/store-service-history-hardcut.md
docs/development/listingkit-local-prod-db-replay.md
docs/refactoring/code-health-decisions.md
docs/superpowers/plans/2026-08-08-listingkit-schema-migration-unification.md
docs/superpowers/specs/2026-08-08-listingkit-schema-migration-unification-design.md
```

其它旧计划中的历史编译清单不改写，不能据其恢复本批已退出路径。专属 schema-unification 两文档按本设计 hard-cut 取代，不恢复原HTTP自动建表/兼容 wrapper。

### §5.4 KEEP / 零差异

所有 workflow、inventory 文件（包括顶部历史快照/CURRENT/计数/ceilings/准确性算法）、保护规则、依赖、SDK构建/安装、现役命令/启动/Store观察/图像/身份/Resource事实和schema/grants零改。上面的维护命令归属表及其文档是明确的两删一增对象，不授权改 CURRENT consumer inventory。

platform/database.Open/OpenShared/CloseShared、真实日志/SetupLoggerWithLevel 仍有当前消费者，保持；SourceAccount 初始化不变。当前各域 schema模型、Store fixture AutoMigrate helper、当前 Goose runner和测试、新版Image physical表/JSON/hash/版本及全部其它原测试保持。新命令/装配不增加企业、权限、表或provider调用。

## §6 验证与原检查归属

只复用当前测试、Goose/GORM、flag parser 与 guard设施，不造新runner/验收平台。当前设计只做路径/消费者/链接/历史后缀/范围和适用原文档检查。实施先让退休 guard 对旧链真实 RED，再抽取/改接/删除到 GREEN；提交稳定候选才运行原完整CI，CI不是实际用户验收。

### §6.1 精确转换矩阵

| ID / 原位置 | 允许变化与必须保留 |
| --- | --- |
| T1 原 workbench/schema 四测试 | 跨域完整表清单、旧聚合重复执行/legacy sentinel/nil wrapper随旧入口退出。8表组织字段断言拆归当前Store5表与Resource3表，NULL Resource bucket拒绝保留；不能把有效组织隔离约束当旧实现细节删除 |
| T2 Store/Resource原 owner测试 | 只在 TestGormStoreRepositoryMigratesRepeatableScopedSchema 内追加原5张Store表organization_id的string/NOT NULL检查，复用当前DB/column类型；其余原断言保持。orgresource/gorm_repository_test.go只新增同包原fixture驱动的3表类型/NOT NULL与原NULL bucket拒绝两测试；其它函数保持，无生产diff/新共库fixture |
| T3 旧 productlisting schema 六测试 | 随专属 registry/default迁移退出：2026083001计数/重复执行、旧sql.DB闭包绑定和宽表清单不是新owner合同。原共享Goose runner/锁/Up与当前Store/Resource/Image schema安全测试全部零改；当前 Image 精确表清单与严格拒绝由原PG测试承接，不添加旧库兼容 |
| T4 旧 runtime/CLI测试到新 owner | RunDefaultLoader/MigrateAll与旧binary默认连接专属测试RETIRE；三项Image owner/worker/显式输入校验移动。原 Organization PG测试整段移动，仅package变，原exclusive gate/DSN限制/临时DB清理、精确表/索引/账本保存/重开/未知库拒绝全保持。config_input_test两用例并入新main_test：显式config fixture补一个已有专项flag，help/缺参/无execute原断言保留，并验证零/多动作拒绝且无执行 |
| T5 TestAppRuntimeLegacyAppConfigFlagsStayRetired | roots表仅旧listingkitschemamigrate改为imageagentschemainit；loadGoFileIndex、AppConfig/app-config两禁令和读/解析失败不变；旧根由T10强制缺席，无missing-skip |
| T6 TestListingKitReleaseImageRetainsNormalSchemaCLI | required三项仅改为新image-agent-schema-init的cmd/out/app路径；原Docker读取/失败断言及其它发布检查保持 |
| T7 TestAccountComposeCommercialSchemaUsesGORMUniqueConstraintNames | 原 retired DDL与两个init.sh禁入不变；required owner仅workbench/schema/runtime.go改为app/schema/storecenter/migrations.go，orgresource/schema.go保持，准确归属注释跟随；实际约束由T2及原域测试承担 |
| T8 listingkit_backend_retirement_test.go三个现有函数 | TestLegacyRuntimeSkeletonAndIdleConfigClosureIsRetired的retained表仅去 appenv.VersionInfo/Version/BuildTime/PrintVersionInfo四项，追加同owner retiredSupport两声明；TestListingKitTaskBackendSourcesAreRetired的旧runtime_test map key仅改为internal/app/runtime/imageagentschemainit/runtime_test.go，原两项旧fixture/task token禁令、读取循环/错误处理保持，旧根由T10强制缺席；appenv KEEP行仅删VersionInfo/PrintVersionInfo两token。TestCurrentCommandsDoNotCompileLegacyBusinessBackends命令表仅旧product-listing-api-schema-migrate改image-agent-schema-init，其余go list/旧依赖禁令不变 |
| T9 maintainedOperationalCommands / 当前维护文档 | 仅两旧命令行删除、新image-agent-schema-init一行增加；分类/owner/错误逻辑和其它命令不变。正式inventory文件/检查不改 |
| T10 新TestLegacySchemaCLIClosureIsRetired | 复用assertRestoredDebugPathsAbsent/decoded import设施强制六旧根及其生产import缺席；新窄owner/命令必需、无旧聚合调用/2026083001注册、两Docker旧编译条目/旧脚本调用禁入；不宽泛禁AutoMigrate、Goose或当前Image v2物理表名。原helper算法不改 |

除表列函数/准确条目和新增 scoped guard外，原检查零改；原Type/Value/Func AST、失败关闭、命令依赖扫描及其它当前KEEP保持。遇真实漏列先报精确提案，不能降低门禁。新CLI保留四专项行为原函数体/SQL/时限/权限检查，任何新安全语义变化不属于机械抽取。

实施必要自检限相关原guard/导航/命令归属、当前owner的原合成/任务自有临时fixture测试和新增有价值回归，四现役命令与新命令只编译，不运行维护产物/脚本/镜像构建。原PG专属测试仅在明确任务隔离且获准的环境运行，禁止借用既有DSN或凭据；没有执行就NOT_RUN，保留原gate，不新增手工验收前置或fixture平台。完整原CI与准确HEAD绑定；不人工重复全量矩阵。

## §7 准入、评审与停止点

1. 当前仅三文档设计；#301 当前段记录唯一Writer/分支、设计范围与原CI/适用复核，历史后缀逐字保留、正文不超65536且保持OPEN。生产/测试/镜像/脚本均未开始。
2. 适用独立设计复核聚焦实际消费者、跨域DDL退出、四有效操作与当前fact owner、Goose历史、§4.2/§6.1及保留能力，不重开无关全局架构。finding按现有五类先分类，最多两轮正常评审。
3. 适用设计复核完成后交用户确认§1–§7；只有授权/准入在Issue/PR写回读回并达到IMPLEMENTATION_READY后才整批实施。确认同时决定旧CLI/默认宽迁移不再续供，不授权操作真实资源。
4. 未经用户本轮授权不默认新增AI互验线程；设计和源码最终复核授权分开、绑定准确HEAD。原复核运行时不重复触发，失败仅按既有一次重试/适用独立检查边界处理。
5. 一个Writer连续完成同一主要PR的范围内实现、自检、提交推送与原CI/finding修复；只有新真实BLOCKER才重开架构。表外必要改动先精确提案，不自动改workflow/inventory或保活旧入口。
6. 最终源码交In Review，准确HEAD、原CI/attempt、独立最终复核、NOT_RUN/UNKNOWN及限制留PR；合并另获准确HEAD授权，合并后确认实际merge SHA对应完整push/main CI。开发自检/CI/AI复核均非用户验收。
7. #301始终OPEN；部署、初始化/grant、迁移、数据/凭据/缓存/Profile/进程操作与关单另授。保留当前能力和Store观察，无新增内部试用前置。
