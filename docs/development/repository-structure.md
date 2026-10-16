# Repository Structure

> #729 retires only the unregistered numeric task/status/processor, root Amazon
> executor and exclusive distributed crawler wiring. The maintained command set,
> current acquisition/Store observations, platform APIs/rules/shared contracts and
> standalone SHEIN login worker stay unchanged; no new runtime or trial prerequisite.

## #678：当前入口的命令装配隔离

`cmd/shein-login-worker` 直接消费 `internal/app/runtime/sheinloginworker`，其配置、
同库 StoreAPI/AccountRepository、Redis/service、health-port、退出信号和清理行为保持。
通用 `app/httpapi` 不再承载该专用启动器，也没有旧入口的 forwarding wrapper。
当前应用和图片 worker 的生产 `go list -deps` 均不包含 tenantbridge/compatibility/ListingKit。

已无真实调用方的 ListingAdmin 旧 HTTP handlers/header/list-query helpers 已退休；其原
repository/API/DTO 与有效 owner/幂等/权限测试保留。当前 Organization/StoreCenter
路由和本地 SHEIN 草稿合同不变。独立 SHEIN worker 的原数字账户持久化以及 #529
发布身份预检仍是债务；本批不表示全仓 legacy 清零、部署或用户验收。

后续 #301 已退休独立 SHEIN worker 未消费的旧登录 HTTP Handler/Module、API 与
numeric header resolver。其 bootstrap 只构造原服务，worker 的真实编译依赖也退出
tenantbridge；账户仓库、Redis/profile、浏览器、Store sync、/healthz 与关闭行为保持。
worker 的原数字账户持久化和发布 identity-preflight 仍是债务，无真实数据/登录/部署操作。

## #723：旧刊登 CLI 与专属调度链退休

三个旧刊登 CLI 已从源码和构建入口退出，专属 bootstrap/consumer/runner/scheduler、
local runtime、旧 SHEIN sync 和平台 module/scheduler 随实际消费者排空。
该批原保留的control-plane后续已按#726确认范围连同一次性platform-recovery入口及
专属派发/恢复编排、Dockerfile/操作脚本退出；任务仓库和原claim/rollback/recovery安全检查保留。
共享`listingadmin`/根`listingruntime`、当前采集、Store观察同步、图片/Temporal及独立
`shein-login-worker`保持原owner和入口。独立worker使用原Dockerfile/启动合同。
后续 #738 按[确认矩阵](../architecture/legacy-ownership-controller-retirement-boundary.md)退出
ownership-controller、两个无保留消费者的专属 ConfigMap 源文件及各自装配，清理 auto-shard
专属配置闭包；其它有效配置/Secret/监控、共享解析与独立登录保持，不更换或运行历史镜像。
源码退休不代表实例停用或全数字任务链退休；无实例/数据操作，生产 NOT_READY。

## 目标

这份文档说明仓库当前推荐的目录职责，以及后续结构收口的方向。当前重点是让正式入口、调试入口、长期工具和本地产物位置保持清晰，避免旧入口和临时产物继续污染仓库结构。

## How To Use This Document

先从下面的[当前入口地图](#current-entrypoint-map)判断运行路径，再从[新代码落点](#current-owner-landing)进入已有 owner；无需先遍历历史计划。目录/命令完整清单只维护在本文件，领域合同与完整退休规则仍分别属于 [Project Boundaries](../architecture/project-boundaries.md)、[Module Target Mapping](../refactoring/module-target-mapping.md) 和 [Legacy Register](../refactoring/legacy-register.md)。本地图是导航，不新建架构权威或依赖准入。

当变更涉及以下问题时，应从 `docs/architecture/README.md` 进入后再落到这里：

1. 顶层目录里该放什么，不该放什么。
2. 正式入口、调试入口、长期维护工具应该分别落在哪。
3. 本地日志、浏览器状态、临时文件、调试二进制该放在哪个运行态目录。
4. `internal/`、`cmd/`、`hack/`、`tools/`、`internal/platforms/` 是否开始混入不该存在的源码外产物。

如果问题首先是“代码应该由哪个业务/装配边界拥有”，先看
`docs/architecture/project-boundaries.md` 和对应专项边界文档；只有当问题落到
目录职责、仓库布局或本地产物放置时，再以这份文档作为直接规则来源。

<a id="current-entrypoint-map"></a>

## 当前入口地图

本节及命令清单静态核对于 `main @ eb9e019686564b976c1a1a9828eeddb454adabac`（2026-09-25）。只确认源码路径、装配关系和命令分类，不声称执行过当前基线测试、已部署或通过产品/生产验收。[Current Refactoring Status](../refactoring/current-refactoring-status.md) 中较早基线的成熟度记录不能代替本节的入口核对；本节也不重签那些历史验收。

| 要做什么 | 从哪里进入 | 当前边界 |
| --- | --- | --- |
| 当前账户、组织、来源账号及按配置开放的业务模块 | [`cmd/current-application/main.go`](../../cmd/current-application/main.go) → [`internal/app/runtime/currentapplication`](../../internal/app/runtime/currentapplication) → [`current_application.go`](../../internal/app/httpapi/current_application.go) | 独立应用装配；`NewCurrentApplicationWithOptions` 不调用旧默认 feature composition。采集、Browser capture、成员、推广和商业模块的启用取决于配置、显式 options 和依赖，不能把路由声明当作全部默认开放。 |
| 排查旧 ListingKit 后端退休 | [Legacy Register](../refactoring/legacy-register.md)、[当前应用入口](../../cmd/current-application/main.go) | #29 已移除 root Task/API/默认装配及 #663 的旧 handoff；当前应用只沿原私密 manifest 与当前 owner 启动。残留数字库存/身份历史 leaf 另列退休 debt，不能作为新功能落点。 |
| 平台执行、control-plane 或图片 worker | 本文件下方的受维护命令及其运行装配归属 | 保留已有单一执行/重试 owner；不是多个独立新产品，也不代表平台成熟度相同。 |
| 修改当前 Console / BFF | [`web/listingkit-ui`](../../web/listingkit-ui) | 复用当前页面、Shell 与身份边界；目录仍叫 ListingKit 不等于其中所有页面都应退休。最终导航服从 UI Authority，不由后端目录名决定。 |
| 初始化、预检或恢复 | 本文件下方的运维入口和明确维护脚本 | 命令存在不是执行许可。当前空库初始化与历史 migration/preflight 必须区分；不得为新系统引入历史迁移前置。 |

当前入口示例（仅说明正常命令形态，不授权连接真实依赖）：

```text
go run ./cmd/current-application -config /absolute/private/current-application.json
```

`-config` 是必需的私有 JSON manifest；当前应用并不使用旧 API 的默认 YAML 配置。该命令不会替代显式 schema 初始化，也不是无依赖 demo。不要通过启动旧入口、回退旧表或增加 wrapper 来填补当前模块尚未开放的能力。实际参数和可选模块以该入口及其 runtime 配置为准。

<a id="current-owner-landing"></a>

## 新代码落点

以下是现有归属的阅读入口，不是新的全仓迁移计划；使用具体 API 前仍须满足现有合同与精确依赖 guard。完整 file/package 退休归属只维护在 [Mapping](../refactoring/module-target-mapping.md) / [Register](../refactoring/legacy-register.md)。

| 变更内容 | 当前阅读入口 | 不应做什么 |
| --- | --- | --- |
| 应用配置、依赖组装、生命周期 | `internal/app/runtime/currentapplication`、`internal/app/httpapi` | 不把业务规则写入 app；不把旧默认装配接入当前应用。 |
| 来源证据、标准商品、资产、增强 | `internal/product/sourcing`、`catalog`、`asset`、`enrichment`；采集协调见 `internal/app/productsourcing` | 不把商品事实放回 root ListingKit，不恢复 ProductEnrich/ProductImage 旧 task 系统。 |
| 平台中立上架与平台规则 | `internal/listing/*`、`internal/marketplace/*` | 不再向历史 `listingkit` / `publishing` / `workspace` 壳层新增长期职责，不复制提交状态机。 |
| 当前身份、组织与来源账号 | 既有 `authidentity` / `authruntime` / `workbenchcontext`、`internal/organization`、`internal/sourceaccountregistry` | 不新建 IAM，不把来源账号当组织成员，不新增 `sourceaccount` / `tenantbridge` 旧消费者。 |
| 商业订单、资金、组织资源 | `internal/commercial/billing`、`internal/ledger/money`、`internal/ledger/orgresource` | 保持报价/订单、资金事实、资源余额各自归属；模块存在不代表微信/支付宝渠道已实现或可用。 |
| AI 模型治理与受控工具 | `internal/aicapability`、`internal/commercetool`、具体领域的 Tool adapter | 不把 provider SDK 或数据库访问直接放进 Agent/Tool 合同，不另建业务重试 owner。 |
| 基础运行机制与外部适配 | `internal/platform/*`、`internal/integration/*` | 不仅凭名字向 `core` / `kernel` / `infra` / `pkg` 增加宽泛职责；按已有精确能力复用，不为统一名称搬目录。 |

### 旧路径如何退出

沿当前用户交付路径执行已有的 **EXTRACT → 切换当前调用方 → RETIRE**：先明确仍需保留的行为及当前 owner，再用适用测试确认当前消费者不依赖旧 owner，最后按该任务范围移除旧调用/实现。新路径已存在，不代表旧路径已删除；改目录名也不能证明依赖已断开。

Marketplace/Listing 归 #29，来源闭环与旧 1688 handoff 归 #30，数字租户依赖归 #301，现有自动边界归 #300。退出条件和剩余消费者记录在对应 Issue / Register，不在这里复制实时数量。保持有效的业务、安全、权限、幂等测试；不新增 compatibility/fallback/双读双写，不执行旧数据迁移，不把全仓清理变成当前业务交付前置。本地图本身不授权代码删除或环境操作。

## 顶层目录约定

CURRENT STATE：命令清单核对于上述 `main @ eb9e019686564b976c1a1a9828eeddb454adabac`，
与 `TestCmdContainsOnlyOfficialEntrypoints` 及实际受维护路径一致。README 和 Code Guide
只引用这里；受维护不等于已部署或通过生产验收。

- `cmd/`
  - 只放受维护的产品运行入口或有明确所有者的运维入口。
  - 当前四个产品运行入口为：
    - `1688-public-browser-collector`
    - `current-application`
    - `image-agent-temporal-worker`

  - 当前受维护的运维入口为：
    - `account-acceptance-fixture`
    - `account-audit-ledger-schema-init`
    - `agent-configuration-schema-init`
    - `agent-customization-schema-init`
    - `ai-workbench-schema-init`
    - `project-center-schema-init`
    - `report-center-schema-init`
    - `notification-center-schema-init`
    - `product-agent-credential-provision`
    - `product-agent-runtime-prepare`
    - `local-ai-trial-grant`
    - `1688-batch-import`
    - `1688-local-agent`
    - `commercial-owner-schema-migrate`
    - `ecoservices-schema-init`
    - `fingerprint-browser-installer`
    - `knowledge-schema-init`
    - `issue36-local-trial-init`
    - `listingkit-schema-migrate`
    - `playwright-installer`
    - `product-listing-api-schema-migrate`
    - `supply-asset-init`
    - `product-acquisition-init`
    - `shein-login-worker`
    - `store-center-schema-init`
    - `source-account-registry-schema-init`
    - `organization-membership-schema-init`
    - `referral-schema-init`
  - `1688-public-browser-collector` 是匿名公开 1688 采集的**独立采集进程**（设计 D13）：无数据库凭据、无数据库网络可达、不做授权、不写操作行，只经内部 RPC 返回 untrusted evidence。缺少调用方准入凭据或出网白名单为空时**拒绝启动**。归 #514 src2b-public-browser-v1；列入清单**不授权**部署该服务或对真实 1688 执行网络验收。
  - `image-agent-temporal-worker` 的构建归属为 `deployments/docker/Dockerfile.product-listing-api`，运行装配归 `internal/app/worker/imageagent`。
  - `1688-local-agent` 的维护入口为 `scripts/1688-local-agent-acceptance.ps1`；归 1688 source runtime。列入清单不授权连接真实账号或执行该脚本。
  - `1688-batch-import` 的维护入口为 `scripts/1688-batch-import.ps1`；属 #398 路线 B 的执行器本地队列切片 S1，只驱动本地队列中的单条商品并回读终态。已确认的 actor/组织必须由调用方显式提供，不从浏览器会话推断；退出码 3 表示结果未知，只允许人工核实，不允许重跑。列入清单不授权对真实 1688 账号执行批量采集。
  - 每个运维入口必须由 `.github/`、`deployments/` 或 `scripts/` 中的构建、部署或脚本引用明确其维护所有者；未归类或同时归类为两类的入口不允许保留在 `cmd/`。
  - `supply-asset-init` 由 `deployments/docker/account-compose/SUPPLY.md` 维护，仅通过私密 Asset owner DSN 和精确数据库确认初始化全新空库并授予既有 `supply_asset_runtime` 窄权限；运行进程不执行 DDL、不迁移历史数据。
  - `store-center-schema-init` 由 `deployments/docker/account-compose` 维护，通过显式私密 owner 配置创建当前 Store 空库 schema，并向既有 canonical commercial quota 表授予窄运行权限；不读取、迁移或兼容历史 Store 数据。
  - `knowledge-schema-init` 由 `deployments/docker/account-compose` 的可选 Knowledge overlay 维护，使用私密 `knowledge_owner` 配置通过 Goose 初始化新空 `knowledge` 数据库，并授予 `knowledge_runtime` 所需列权限；运行进程不执行 DDL，不迁移历史数据。依据 #557 / [Knowledge V1 架构](../architecture/agent-knowledge-context-v1.md)。
  - `ecoservices-schema-init` 的构建归属为 `deployments/docker/Dockerfile.account-compose` 的 `schema-init` target；安装者显式提供私密 `ECOSERVICES_SCHEMA_DSN`，仅在名为 `ecoservices` 的专属新数据库安装 schema 并授予既有 `ecoservices_runtime` 最小权限。Compose 不自动调用，不创建申请、订单或付款，不迁移旧数据。依据 #603 / [生态服务 V1 架构](../architecture/ecosystem-services-v1.md)。
  - `agent-configuration-schema-init` 的构建归属为 `deployments/docker/Dockerfile.account-compose` 的 `schema-init` target；安装人员通过私密 `AGENT_CONFIGURATION_SCHEMA_DSN` 与显式 `--runtime-role` 单独运行，只初始化同 ProductAgentDB 的新配置 schema 并授予最小权限。Compose 不自动调用，不自动启用企业，不迁移旧数据。依据 #570 / [企业智能体配置 V1](../architecture/organization-agent-configuration-v1.md)。
  - `agent-customization-schema-init` 的构建归属为 `deployments/docker/Dockerfile.account-compose` 的 `schema-init` target；安装人员显式提供私密 `AGENT_CUSTOMIZATION_SCHEMA_DSN` 和既有 `--runtime-role`，安装独立库的人工需求、私有交付及报告 schema 与最小权限。Compose 不自动调用，不创建需求、交付或交易，不迁移旧数据。获准的当前系统 schema 演进保留已有事实。依据 #611 / [智能体定制 V1](../architecture/agent-customization-v1.md) / [私有交付 V1](../architecture/private-agent-delivery-v1.md)。
  - `product-agent-credential-provision` 由 `scripts/product-agent-credential-provision.ps1` 显式委托；部署者用私有 manifest 与私有凭据输入，通过同一 ProductAgentDB 的独立受限写入角色配置标题组织行。它不自动启用 Agent 或执行付费调用；准入条件见 [标题诊断试用交接](../operations/product-agent-trial.md)。
  - `product-agent-runtime-prepare` 的构建归属为 `deployments/docker/Dockerfile.product-listing-api`；安装者获准后显式选择 `--target run` 或 `--target review`，分别使用私密 `PRODUCT_AGENT_SCHEMA_DSN` / `PRODUCT_REVIEW_SCHEMA_DSN`。它只编排原 owner 的三表全缺新建及现表窄 grants / 完整精确 no-op；部分状态和漂移拒绝，不创建角色或密码，不迁移、不启用模型、不启动 serving 或 worker。两库分别提交，未知结果先核实；依据 [内容标题与 Chat 共存](../architecture/content-agent-chat-coexistence-v1.md)。
  - `local-ai-trial-grant` 的构建归属为 `deployments/docker/Dockerfile.product-listing-api`，装配归 `internal/app/localaitrial`、资源事实归 `ledger/orgresource`。沿私密配置和既有 `commercial_owner_runtime`，默认 inspect 只读；明确获准并开启 grant 后仅一次10000非现金ai_point，复用来源幂等/事务/债务/审计，未知结果原键读回。它不安装 schema/权限、不改钱包/成员限额、不启模型或调用 provider；依据 [#724 专用来源设计](../architecture/local-ai-trial-v1.md) 和[待确认执行清单](../operations/local-ai-trial-v1.md)。列入维护清单不授权真实发点或部署。
  - 不再新增临时调试可执行程序。
  - 历史 Source Account ownership preflight CLI、脚本和专属迁移实现已按 #529 退休；其 [SUPERSEDED 说明](../operations/source-account-ownership-preflight.md) 仅保留历史证据，不是当前运行入口。
  - 旧 `listingkit-owner-scope-dry-run` / `listingkit-owner-scope-exceptions` 及其专属装配、写入方法和包装已按 #529 退休。旧发布身份预检、独占共享reader与tenantbridge已按批准设计退休；schema工具与独立SHEIN worker保持。
  - `source-account-registry-schema-init` 由 `scripts/source-account-registry-local-acceptance.ps1` 维护，初始化当前 SourceAccountRegistry 空库 schema 及 RUN-1 在同一 source-account pool 上的账户资料、个人/企业认证 schema；不读取、迁移或兼容旧 Source Account 数据。
  - `organization-membership-schema-init` 由 `deployments/docker/account-compose` 维护，只初始化本账户中心实例的组织成员回执 schema；不读取、迁移或兼容旧成员数据。
  - `commercial-owner-schema-migrate` 由 `scripts/commercial-owner-schema-migrate.ps1` 维护；分别针对私有 schema-owner manifests 初始化 canonical money 数据库的钱包/结算 schema，以及 commercial-owner 数据库的订单与 orgresource schema，不自动切换数据库或操作生产数据。
  - `account-acceptance-fixture` 由 `deployments/docker/account-compose` 的 `acceptance` profile 维护，只通过官方 ZITADEL API 创建并回读隔离验收组织与角色授权，写入项目私有的脱敏 manifest；不提供生产路由、不写业务事实表。
  - `account-audit-ledger-schema-init` 由 `deployments/docker/account-compose` 维护，仅为全新隔离 Account Compose 的 image/product owner 初始化现有 AI 调用账本并授予审计只读角色查询权限；不启用 Agent 执行或迁移旧实例。依据 #587 / [账户审计用量只读源架构](../architecture/account-audit-usage-readers-v1.md)。
  - `product-acquisition-init` 由 `scripts/product-acquisition-init.ps1` 显式委托，要求私有配置路径和精确空库名称确认；只初始化 #398 当前 Product 采集的五张表及 runtime grants，不创建数据库或角色，不自动执行，不迁移旧数据。准入见 #398 评论 5643032970。
  - `issue36-local-trial-init` 仅由 `deployments/docker/account-compose` 的显式 #36 本机试用 overlay 在新项目中调用：以私有 Store Center owner 配置安装当前 Review/Asset schema 并准备固定合成样本；不作为默认应用入口，不访问共享或生产数据。
  - `referral-schema-init` 由 `scripts/referral-schema-init.ps1` 维护，通过显式 DSN 文件初始化当前推广注册空库 schema；执行范围和验证要求见 [推广注册运维说明](../engineering/referral-registration.md)。
  - 历史爬虫、订阅、兼容 API、地址复制、一次性迁移或调试入口不得回流到 `cmd/`；确需保留时放到 `hack/`、`tools/` 或业务模块内。
  - 不放本地 `logs`、`tmp`、`__debug_bin*` 等运行态产物；这类文件统一放到仓库根 `.local/`。
- `hack/`
  - 放调试、试验、验证程序。
  - `hack/debug` 是当前受管的非生产调试入口目录。
  - `hack/k8s` 可保留 Kubernetes 相关运维验证支持。
  - 不放本地 `tmp`、`logs`、`bin` 等运行态产物；调试产物统一放到仓库根 `.local/`。
- `tools/`
  - 放长期维护的小工具模块。
  - 适合独立可复用的开发工具，不适合一次性验证程序。
  - 不放 `node_modules`、生成的 `.exe`、`result` 输出目录等本地产物；依赖和输出应保持可重建。
- `scripts/`
  - 放运维脚本、迁移脚本、部署脚本和一次性自动化脚本。
- `.local/`
  - 放本地日志、浏览器状态、临时文件、开发期二进制和其他运行态产物。
  - 默认不提交，避免继续污染仓库根目录。
  - 推荐按 `logs/`、`tmp/`、`chrome/`、`bin/`、`dev-logs/`、`playwright-cli/` 分子目录管理。
  - 日志库和测试默认只输出到 stdout；只有 app 运行装配可以通过显式配置启用文件日志。
  - 仓库内受维护的相对日志路径统一位于 `.local/logs/`；测试文件输出使用 `t.TempDir()`。

## 当前护栏

目录约定由以下测试守住：

- `TestCmdContainsOnlyOfficialEntrypoints`
- `TestCmdProductionEntrypointsDoNotImportDomainOrInfraPackages`
- `TestInternalCmdEntrypointsDoNotImportDomainOrInfraPackages`
- `depguard: cmd_domain_dependencies`
- `TestCmdPackagesDoNotImportAppCompatibilityLayers`
- `depguard: cmd_legacy_app_compatibility`
- `TestHackContainsOnlyManagedSupportAreas`
- `TestHackSupportAreasContainNoLocalArtifacts`
- `TestTrackedLocalArtifactsStayOutOfProductionEntrypoints`
- `TestProductionEntrypointsContainNoLocalArtifacts`
- `TestTrackedLocalArtifactsStayOutOfTools`
- `TestToolsContainNoLocalArtifacts`
- `TestInternalPackagesContainNoLocalArtifacts`
- `TestSDSLoginRuntimeStateStaysOutOfInternalPackages`
- `TestPlatformRegistrationPackagesStayThin`
- `TestPlatformRegistrationPackagesContainNoLocalArtifacts`
- `depguard: commercetool_boundaries`

如果需要新增正式入口、调试目录或平台注册文件，应在同一变更中更新这份文档和对应 allowlist。

## internal 当前稳定边界

- `internal/app`
  - 运行装配层，负责当前httpapi、worker、采集及其它有效命令的组装与协调；旧listing bootstrap/consumer/scheduler/control-plane装配已退休。
  - 不承载具体平台业务规则。
  - 其中 `internal/app/httpapi` 当前只负责共享 HTTP runtime 协调；各业务 HTTP builder 已下沉到 `internal/*/httpapi`。
  - 当前独立登录装配归`internal/app/runtime/sheinloginworker`，其它正式入口按本文件命令清单及各自runtime/owner维护；不恢复旧listing/control-plane runtime。
- `internal/*`
  - 不放本地 `.local`、`logs`、`tmp` 等运行态产物；业务包、基础设施包和平台包都应保持源码可审查。
  - SDS 登录态、浏览器状态和 auth/cookie JSON 必须放在仓库根 `.local/sds/` 或其他明确忽略的运行态目录，不能放在 `internal/sdslogin/data/`。
  - `TestInternalPackagesContainNoLocalArtifacts` 检查实际文件系统，包括 Git 忽略路径；`.gitignore` 不能作为在源码包下保留运行态文件的依据。
- `internal/listingkit`
  - CURRENT STATE：根旧Task/API/Workspace与默认HTTP装配已退休；仍有独立源码消费者的旧leaf按Legacy Register登记债务，不是当前产品facade，不得恢复根owner或添加当前消费者。
  - `internal/authz/httpapi` owns the extracted unchanged route-role transport; current Organization authorization is composed in `internal/app/httpapi/server_auth.go`, with identity/grants owned by authidentity/authruntime/workbenchcontext. The old ListingKit HTTP assembly stays absent.
  - 新增平台规则、商品事实规则、可复用资产规则不应继续放入 root `internal/listingkit`。
- `internal/commercetool`
  - 拥有框架中立的 Tool Definition、Schema、Registry、Agent Allowlist、Invocation Policy 和 Tool Audit Port。
  - 生产代码不得依赖 Framework、Transport、Workflow、Persistence、Provider SDK、Marketplace Client 或 domain implementation packages；领域能力由装配层通过窄 Service/Query ports 注入 Executor。
  - `depguard: commercetool_boundaries` 对该目录全部生产 Go 文件使用 strict allowlist，禁止通过宽泛 `internal` 或 SDK 父命名空间绕过边界。
- `internal/listing`
  - 平台中立的 Listing 子领域目标位置，当前包括 `preview`、`studio`、`submission` 等已抽出的稳定 seam。
  - `internal/listing/task` 拥有 task-scoped resource resolution 与 tenant-admin checker 窄合同；现有 ListingKit persistence adapter 可以实现该 Port，但调用方不能依赖 legacy Task DTO 或全局默认权限实例。
- `internal/product/catalog/tools/canonicalinspect`
  - `product.canonical.inspect` 是 Phase 2A 的只读 B0 Tool Adapter：通过 `internal/listing/task` 解析授权资源，只读取权威 `ProductSnapshot`，并输出严格、去除任意 authority metadata 的投影。
  - 该包不拥有 task、repository、write/publish 能力、HTTP/worker/runtime；其 8 MiB Catalog 输入边界由只读 persistence adapter 通过数据库侧条件投影执行，不是共享 Catalog 的全局限制；Phase 2B 之前没有用户可访问的 Agent Runtime。
- `internal/integration/commercetoolauth`
  - 只负责把 `authidentity.AuthenticatedIdentity` 和现有 Casbin owner 适配到 Commerce Tool contracts，不复制 IAM policy。
- `internal/shein` / `internal/temu` / `internal/amazon`
  - 当前仍然存在的历史平台实现目录。
  - 本阶段不为目录一致性做大规模迁移，只在有明确所有权收益时逐步收口。
- `internal/marketplace/*`
  - 新的 marketplace 规则目标位置，当前已承接部分 SHEIN publishing/workspace 与 Amazon marketplace 结构。
- `internal/platforms`
  - 平台注册和选择层，只保留 module descriptor、文档和必要测试。
  - 不放本地 `tmp`、`logs`、`bin` 等运行态产物；这类文件统一放到 `.local/`。
- `internal/pkg`
  - 纯技术通用件目录。
  - 继续保留现状，不在本阶段重命名。

## 后续收口方向

### Phase 2 runtime foundation closure

Historical Phase 2 closure evidence (not a new dependency permission) establishes three runtime ownership roots:

- `internal/app` owns lifecycle and provider registration, migration execution,
  final HTTP instrumentation, and ordered shutdown.
- `internal/platform` owns the application runtime mechanisms: config loading,
  logging, database and Goose, Redis, RabbitMQ, worker pool, Temporal dial,
  feature flags, and tracing.
- `internal/integration` owns external adapters: OpenAI, Gemini, GRSAI, S3, and
  remote image HTTP.

This is a runtime-foundation closure, not the end of the business migration.
At this close, only nine target domain roots satisfy the final dependency rule:
`internal/listing`, `internal/product`, `internal/marketplace`,
`internal/agent`, `internal/knowledge`, `internal/resourcecatalog`,
`internal/commercial`, `internal/ledger`, and `internal/organization`.
Historical business roots remain under non-growth ceilings until their owning
product, marketplace, listing, agent, or organization phase removes concrete
runtime dependencies.

- 平台实现逐步从历史目录收口到统一的平台边界，例如 `internal/marketplace/*` 或经过批准的目标包。
- `internal/listingkit` 按 [Legacy Register](../refactoring/legacy-register.md) 抽取有效行为到当前 owner，调用方切换后退休旧路径；不作为永久 facade。
- `internal/app` 继续保持运行装配职责，避免混入产品或平台业务逻辑。
- 生成型依赖基线和包地图只作为当前验证证据，不作为长期结构说明；需要时重新运行脚本生成。

## 本阶段明确不做的事情

- 不迁移 `internal/listingkit`、`internal/shein`、`internal/publishing` 的正式实现目录。
- 不为目录一致性进行大规模包重命名。
- 不把历史调试或兼容入口重新放回 `cmd/`。
- 不改动业务 API、HTTP 路由、消息结构、数据库结构。
- 不重命名 `internal/pkg`，也不引入新的共享层抽象。
