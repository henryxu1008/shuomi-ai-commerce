# Repository Structure

本页按当前源码组织开发入口。领域规则见[系统边界](../architecture/project-boundaries.md)和[架构索引](../architecture/README.md)；产品位置见[UI / IA Authority](../product/final-ui-ia-authority.md)。

<a id="current-entrypoint-map"></a>

## 当前入口与职责

| 路径 | 职责 |
| --- | --- |
| `cmd/current-application` | 当前后端应用命令 |
| `internal/app` | composition root、配置装配、HTTP、worker 与运行注入 |
| `internal/product` | sourcing / catalog / asset / review / proposal，来源与商品事实 |
| `internal/listing` | materials / completiondraft / preparation / record / submission |
| `internal/marketplace` | 平台合同、确定性校验、平台接入与提交边界 |
| `internal/storecenter` | 当前企业店铺与可见性 owner |
| `internal/authruntime`、`internal/authz`、`internal/workbenchcontext` | 官方身份适配、当前授权和企业上下文 |
| `internal/commercial`、`internal/ledger` | 商业订单、Money / Resource 事实与结算 |
| `internal/aicapability`、`internal/commercetool`、`internal/imageagent` | 能力、工具和图片执行合同 |
| `internal/integration` | Provider SDK、Marketplace Client、存储和浏览器等外部适配 |
| `internal/platform`、`internal/kernel` | 平台装配、注册与最小内核合同 |
| `web/listingkit-ui` | Console / BFF，入口见其 README |
| `extensions/product-capture` | 浏览器采集插件，入口见其 README |
| `deployments/docker/account-compose` | 有界本地环境组合与配置说明 |
| `tests`、`.golangci.yml` | 当前包依赖、权限与源码结构检查 |

Framework、Transport、Workflow、Persistence、Provider SDK、Marketplace Client 不能反向进入中立 domain implementation packages。`internal/commercetool` 只持有工具合同、注册、准入和窄端口，受 `depguard: commercetool_boundaries` 约束；正式业务仍由领域 owner 执行。

## 命令与配置

- 当前应用启动从[根 README](../../README.md)及 [Account Compose](../../deployments/docker/account-compose/README.md)选择已说明的模块组合。
- 本机 1688 批量导入：`cmd/1688-batch-import`，本地队列在 `internal/batchcapture`；服务器采集为 `cmd/1688-public-browser-collector`。使用条件见[批量采集合同](../architecture/1688-batch-import.md)。
- 图片异步执行：`cmd/image-agent-temporal-worker`，运行与恢复见[图片操作说明](../operations/product-image-agent.md)。
- `cmd/*-schema-init`、`commercial-owner-schema-migrate`、`product-acquisition-init`、`supply-asset-init` 等是显式安装命令。只在对应合同和目标环境授权内使用，serving runtime 不执行 DDL。
- 试用数据、凭据 provision 和 runtime prepare 命令遵守其对应操作合同；命令存在不表示可向保留实例执行。
- 私有配置使用当前显式 JSON manifest 与 secret file 路径，具体值不提交仓库；见[凭据管理](secret-management.md)。
- 镜像构建使用[显式 Git 版本入口](../operations/application-image-build.md)，构建和部署分别授权。

## 工作树与本地产物

正式源码放在当前责任目录，开发脚本放 `scripts/`。`.local/`、`.tmp/` 和 `tmp/` 用于已忽略的本地产物，不作为领域实现或新文档归档；依赖报告入口见[开发说明](README.md#dependency-and-package-checks)。

新增文件先判断所属 owner。跨域通过窄 Service / Query port 和应用显式注入，不因测试、旧包名或工程标识建立第二套业务入口。

## Canonical inspection

`product.canonical.inspect` reads an authorized exact ProductSnapshot and the current `internal/listing/record` contract. It does not own a Task lifecycle, update Catalog facts or grant publication permission. The original permission and canonical-inspection import guards continue to apply.

## 维护中的命令索引

- `cmd/`
  - 产品运行入口为：
    - `1688-public-browser-collector`
    - `current-application`
    - `image-agent-temporal-worker`
  - 运维入口为：
    - `1688-batch-import`
    - `account-acceptance-fixture`
    - `account-audit-ledger-schema-init`
    - `agent-configuration-schema-init`
    - `agent-customization-schema-init`
    - `ai-workbench-schema-init`
    - `browser-kernel-schema-init`
    - `commercial-owner-schema-migrate`
    - `ecoservices-schema-init`
    - `image-agent-schema-init`
    - `issue36-local-trial-init`
    - `knowledge-schema-init`
    - `local-ai-trial-grant`
    - `model-invocation-recovery`
    - `notification-center-schema-init`
    - `organization-membership-schema-init`
    - `product-acquisition-init`
    - `product-agent-credential-provision`
    - `product-agent-runtime-prepare`
    - `market-research-schema-init`
    - `project-center-schema-init`
    - `referral-schema-init`
    - `report-center-schema-init`
    - `source-account-registry-schema-init`
    - `store-center-schema-init`
    - `supply-asset-init`
