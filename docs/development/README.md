# 开发入口

返回[文档首页](../README.md)。先确定当前产品位置、执行 Issue 和领域 owner，再修改实现。

| 目的 | 入口 |
| --- | --- |
| 源码、命令和目录职责 | [Repository Structure](repository-structure.md#current-entrypoint-map) |
| 后端启动与模块组合 | [项目 README](../../README.md)、[Account Compose](../../deployments/docker/account-compose/README.md) |
| Console / BFF | [前端 README](../../web/listingkit-ui/README.md) |
| 浏览器采集插件 | [插件 README](../../extensions/product-capture/README.md) |
| 配置与凭据 | [凭据管理](secret-management.md) |
| 镜像构建 | [显式 Git 版本构建](../operations/application-image-build.md) |
| CHAIN-1 有界环境 | [隔离环境装配](local-product-acceptance.md) |
| 准入、接单与评审 | [AGENTS.md](../../AGENTS.md)、[Issue 驱动规则](../engineering/issue-driven-development.md) |

<a id="dependency-and-package-checks"></a>

## 依赖与包边界检查

包依赖由当前架构、`.golangci.yml` 和源码检查约束。需要分析报告时复用 `scripts/analyze-project-deps.ps1` 或 `scripts/dependency-baseline.ps1`，输出保存在已忽略的 `.local/refactoring`；报告不是新的领域事实或额外 CI 门禁。
