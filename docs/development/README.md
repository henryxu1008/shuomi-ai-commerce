# 开发说明

返回[文档首页](../README.md)。先确认当前产品和领域 owner，再选择入口；本目录包含开发说明和保留的旧调试记录。

## 当前开发入口

| 目的 | 入口 |
| --- | --- |
| 项目准备、后端启动 | [项目 README](../../README.md)、[Account Compose](../../deployments/docker/account-compose/README.md) |
| 源码职责、正式命令、工具与本地产物 | [Repository Structure](repository-structure.md#current-entrypoint-map) |
| 领域归属、依赖方向和准入 | [架构索引](../architecture/README.md)、[Project Boundaries](../architecture/project-boundaries.md)、[AGENTS.md](../../AGENTS.md) |
| Console / BFF | [前端 README](../../web/listingkit-ui/README.md) |
| 浏览器采集扩展 | [采集插件 README](../../extensions/product-capture/README.md) |
| 显式选择 Git 版本构建镜像 | [镜像构建说明](../operations/listingkit-image-build.md) |
| 品牌、仓库名称和保留的工程标识 | [更名说明](branding-and-repository-rename.md) |
| 配置与凭据边界 | [当前应用配置入口](repository-structure.md#current-entrypoint-map)、[凭据管理](secret-management.md)；当前应用使用显式私有 JSON manifest，旧 YAML / ListingKit 配置不替代当前入口合同 |

## 专项记录与历史材料

以下入口保留原路径供追溯，使用前先读顶部的退休说明及文内限定范围：

- 旧开发入口：[Local Agent PoC](1688-local-agent-poc.md)、[ListingKit 联调](listingkit-local-debug.md)、[线上库复验记录](listingkit-local-prod-db-replay.md)、[旧对象存储上传](listingkit-object-storage.md)。
- 旧验收路径：[CHAIN-1](chain1-local-product-acceptance.md)、[Image Agent 本地验收](image-agent-local-acceptance.md)、[RabbitMQ 主链验收](rabbitmq-mainline-validation.md)。这些旧编排或命令不能作为当前试用步骤。
- 旧实现整理：[全局配置退休](listingkit-config-migration-checklist.md)、[字段清理 inventory](listingkit-semantic-field-cleanup-inventory.md)、[字段迁移 checklist](listingkit-semantic-field-migration-checklist.md)、[旧错误处理指南](error-handling-guide.md)。
- 有特定基线与环境限制的记录：[隔离订阅目录输入](isolated-subscription-catalog.md)、[SHEIN 报名来源 SDS 排查](listingkit-shein-enrollment-source-sds-troubleshooting.md)。不作为通用安装或真实数据操作说明。

当前用户操作与运行交接从[操作说明](../operations/README.md)进入；源码目录和正式命令的完整清单只维护在 Repository Structure。
