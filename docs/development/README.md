# 开发说明

返回[文档首页](../README.md)。先确认当前产品和领域 owner，再选择开发入口。

| 目的 | 入口 |
| --- | --- |
| 项目准备、后端启动 | [项目 README](../../README.md)、[Account Compose](../../deployments/docker/account-compose/README.md) |
| 源码职责、正式命令、工具与本地产物 | [Repository Structure](repository-structure.md#current-entrypoint-map) |
| 领域归属、依赖方向和准入 | [架构索引](../architecture/README.md)、[Project Boundaries](../architecture/project-boundaries.md)、[AGENTS.md](../../AGENTS.md) |
| Console / BFF | [前端 README](../../web/listingkit-ui/README.md) |
| 浏览器采集扩展 | [采集插件 README](../../extensions/product-capture/README.md) |
| 显式选择 Git 版本构建镜像 | [镜像构建说明](../operations/listingkit-image-build.md) |
| 品牌与保留工程标识 | [更名说明](branding-and-repository-rename.md) |
| 配置与凭据 | [私有 JSON manifest 入口](repository-structure.md#current-entrypoint-map)、[配置退休与拒绝边界](listingkit-config-migration-checklist.md)、[凭据管理](secret-management.md) |

## 专项说明与保留证据

- [当前本地联调入口](listingkit-local-debug.md)：仅导航到当前应用、UI 与身份合同。
- [CHAIN-1 隔离验收装配](chain1-local-product-acceptance.md)：仍有当前 runner 和合同检查，只在对应授权下使用。
- [线上库复验记录](listingkit-local-prod-db-replay.md)、[SHEIN 报名来源 SDS 排查](listingkit-shein-enrollment-source-sds-troubleshooting.md)：保留具体基线、环境限制与原始结果，不作为可重复执行的通用安装说明。

旧 LocalAgent PoC、Task 调试/上传、双字段兼容迁移、旧 RabbitMQ 验收及订阅试用 seed 已退出当前文档树；完整原件见[清理前固定版本](https://github.com/henryxu1008/shuomi-ai-commerce/tree/c0210ff34f44c371b5a1993aed89f439cfbdc1ff/docs/development)。当前操作从[运行说明](../operations/README.md)进入，工程交接与验证记录从[工程文档](../engineering/README.md)进入。
