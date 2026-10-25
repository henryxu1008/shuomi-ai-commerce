# 硕米智能引擎 / Shuomi AI Commerce

**AI 驱动的跨境电商智能体平台 · AI Commerce Agent Platform**

[![CI](https://github.com/qq550723504/shuomi-ai-commerce/actions/workflows/ci.yml/badge.svg)](https://github.com/qq550723504/shuomi-ai-commerce/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat&logo=go)](https://go.dev/)

硕米智能引擎面向跨境电商卖家与运营团队，将商品采集、资料管理、AI 内容与图片处理、平台适配和店铺运营连接起来。目标是让团队围绕一个商品或业务目标协作，减少重复录入和多工具切换，并在关键操作前检查结果、作出确认。

[产品与设计](./docs/product/final-ui-ia-authority.md) · [本地环境](./deployments/docker/account-compose/README.md) · [前端开发](./web/listingkit-ui/README.md) · [采集插件](./extensions/product-capture/README.md) · [架构文档](./docs/architecture/README.md)

## 围绕商品完成工作

产品希望连接这样一条业务路径：

```text
采集商品资料 → 保存来源与素材 → AI 辅助处理 → 平台适配与校验 → 确认草稿或提交
```

下面列出主要场景与已有应用入口。具体可用范围取决于实例配置、当前账号与企业权限；这条目标路径不代表所有环节已在每个实例中开放。

| 场景 | 在哪里开始 | 主要用途 |
| --- | --- | --- |
| 获取与整理商品资料 | 工具市场 → 我的工具；数据服务 → 我的数据 | 通过 1688 / Amazon 浏览器采集等接入方式获取资料，查看已保存商品、来源、规格和素材。 |
| AI 辅助内容与图片处理 | AI工作台；商品处理页面 | 结合已开放的智能体、技能和知识，准备标题、描述、图片等候选结果，再检查与确认。 |
| 准备平台商品 | 供应市场 → 我的供应链 | 整理自有或来源商品，补充资料，按已接入平台的规则准备和检查草稿。 |
| 管理店铺与商品 | 店铺中心 | 管理店铺、官方 API 授权与已保存店铺商品；同步和平台操作依各平台接入情况开放。 |
| 团队协作与资源管理 | 我的账户；套餐与权益 | 管理企业成员、角色和可用资源，查看使用情况与审计记录。 |

运营驾驶舱、工具市场、数据服务和生态服务等页面的完整归属见[产品导航与设计依据](./docs/product/final-ui-ia-authority.md)。平台名称出现在代码或页面中，不等于该平台的全部能力已可使用。

AI 负责理解需求和生成候选；权限、商品事实、价格与平台规则、资源扣减和实际提交由业务服务校验与执行。生成内容不会自动成为已发布商品，草稿与提交也有各自的确认和授权条件。

## 使用与当前状态

项目持续开发中，按具体业务路径交付。**设计目标、代码实现、实例开放和用户验收分别确认。** 未配置的模型、店铺连接、支付渠道或可选模块会限制对应功能；菜单、容器健康或 CI 通过不代表完整业务路径已验收。

- **已有实例**：使用该实例交付的应用地址与账号，登录后进入 `/workbench`；使用范围以该实例说明为准。
- **新建本地环境**：从 [Account Compose](./deployments/docker/account-compose/README.md#start) 开始，按说明准备独立项目、端口、本地 TLS 和私密登录资料。需要组合更多已接入模块时，阅读[统一本地应用说明](./deployments/docker/account-compose/MODULES.md)及其中的配置要求。
- **浏览器采集**：参照[插件说明](./extensions/product-capture/README.md)构建与目标应用地址匹配的包。Chrome / Edge 使用同一扩展源码，采集结果在应用内确认保存。
- **开发与交付进度**：查看[当前工作排序 #137](https://github.com/qq550723504/shuomi-ai-commerce/issues/137)及对应 Issue / PR；历史运行记录按其日期和版本阅读。

新建环境使用全新空业务库。已有实例的重启、升级和插件更新应沿用原配置、安装身份与数据；新安装步骤不用于直接升级已有实例。

## 本地开发

### 准备依赖

- Go：版本以 [go.mod](./go.mod) 为准，当前为 `1.26.0`。
- Node.js：与 [CI](./.github/workflows/ci.yml) 保持一致，当前为 `22`。
- pnpm：使用[前端 package.json](./web/listingkit-ui/package.json) 的 `packageManager` 指定版本。
- Docker Compose：使用上述本地环境时需要。数据库、身份服务、对象存储及其他依赖按启用模块准备。

```bash
git clone https://github.com/qq550723504/shuomi-ai-commerce.git
cd shuomi-ai-commerce
go mod download
```

仓库与产品已更名，Go 模块仍是 `task-processor`，前端目录仍是 `web/listingkit-ui`。这些是当前真实工程标识，详见[品牌与仓库更名说明](./docs/development/branding-and-repository-rename.md)。

### 启动后端

先按本地环境说明完成所需空库初始化，并准备绝对路径的私有 JSON manifest，再在仓库根目录运行：

```bash
go run ./cmd/current-application -config /absolute/private/current-application.json
```

`-config` 必填；该命令启动应用，不代替 schema 初始化。配置与凭据保存在仓库外的私密位置。完整命令归属和模块接线见[当前入口地图](./docs/development/repository-structure.md#current-entrypoint-map)。

### 启动前端

另开终端，在前端目录运行：

```bash
cd web/listingkit-ui
pnpm install --frozen-lockfile
pnpm dev
```

开发首页为 `http://localhost:3000/`，Console 为 `http://localhost:3000/workbench`。完整登录和业务操作还需要服务端 API origin、ZITADEL / Auth.js 及对应模块配置，详见[前端说明](./web/listingkit-ui/README.md)。该目录没有需要复制的 `.env.example`。

### 开发检查

按实际修改选择相关 Go 测试；前端常用检查为：

```bash
pnpm lint
pnpm typecheck
pnpm test
pnpm build
```

上述前端命令在 `web/listingkit-ui` 下运行。必需检查由仓库 CI 执行，运行结果与真实平台操作、用户试用分别记录。

## 阅读代码

后端使用 Go，Web Console 使用 Next.js / React 和 TypeScript；浏览器扩展与应用分别构建。

| 位置 | 内容 |
| --- | --- |
| [`cmd/current-application`](./cmd/current-application) | 当前后端启动入口 |
| [`web/listingkit-ui`](./web/listingkit-ui) | Web Console、页面与 BFF |
| [`extensions/product-capture`](./extensions/product-capture) | 1688 / Amazon 浏览器采集插件 |
| [`internal/product`](./internal/product) | 来源、商品快照、集合与素材等商品能力 |
| [`internal/listing`](./internal/listing) / [`internal/marketplace`](./internal/marketplace) | 平台中立上架能力与平台规则 |
| [`internal/aicapability`](./internal/aicapability) / [`internal/commercetool`](./internal/commercetool) | AI 能力治理与受控电商工具 |
| [`deployments/docker/account-compose`](./deployments/docker/account-compose) | 本地环境与可选模块装配 |
| [`docs`](./docs) | 产品、架构、开发与运行文档 |

更多 owner、运行入口与目录职责见[仓库结构说明](./docs/development/repository-structure.md#current-owner-landing)。

## 参与开发

先阅读 [AGENTS.md](./AGENTS.md) 和 [Issue 驱动开发规则](./docs/engineering/issue-driven-development.md)，核对执行 Issue 的范围、设计依据和当前负责人。产品导航与交互以[最终 Figma UI / IA Authority](./docs/product/final-ui-ia-authority.md)及已批准产品决定为准；领域合同从[架构索引](./docs/architecture/README.md)进入。

新业务遵循[全新系统产品基线（PD-GREENFIELD-NO-LEGACY-MIGRATION-2026-09-08）](./docs/product/greenfield-no-legacy-migration.md)，按全新安装、空业务数据和当前模型设计。历史 profile 和 cutover gate 不是当前验收前置。

旧 ListingKit 产品与混合架构按 [Hard-Cut Policy](./docs/refactoring/legacy-hard-cut-policy.md) 和 [Legacy Register](./docs/refactoring/legacy-register.md)执行 `EXTRACT | RETIRE`；仍有价值的行为归入当前 owner，不新增旧 Service 包装、兼容 fallback 或双读双写。现有前端目录名不改变这一边界。合并、部署和真实环境操作遵循各任务的明确授权。
