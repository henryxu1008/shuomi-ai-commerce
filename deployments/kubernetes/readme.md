# Kubernetes 部署清单

> #822 已退休旧 Amazon 刊登部署源码；该清单不再提供 Amazon Listing 部署入口。当前 Amazon 来源采集 Processor/browser 保持，旧数字 Task/HTTP 采集部署声明已在 #867 退休；本批未操作集群或已有资源。

当前仓库内提供这些 K8s 清单：

- `cert-manager/letsencrypt-prod`
- `monitoring/alertmanager-wecom`
- `monitoring/prometheus-storage`（共享 PVC 保护及同 namespace 窄路由）
- `zitadel/local`

旧 SHEIN/TEMU 刊登运维源码已按 [#839 完整闭包](../../docs/architecture/legacy-listing-operations-closure-retirement-boundary.md)退休。shein-listing 仅保留 Store 代理同步所需共享凭据/Namespace装配，temu-listing 仅保留退休说明；二者不是当前应用发布入口。当前采集、Store观察同步与应用入口见[仓库入口地图](../../docs/development/repository-structure.md)。本次仅改源码，未操作集群或资源。

## ZITADEL 2026-05-16 历史切换记录

以下只保留 2026-05-16 的切换事实，不代表当前运行状态。该记录中的 `https://auth.shuomiai.com` 已不再使用 `zitadel` 命名空间内置的
`zitadel-postgres16`。

自 `2026-05-16` 起，线上 ZITADEL 数据库改为复用 `yudao-cloud` 命名空间内
现有的 PostgreSQL 18 实例：

- 实例 Service：`postgresql-v18.yudao-cloud.svc.cluster.local:5432`
- 独立数据库：`zitadel_auth`
- 独立用户：`zitadel`

这次切换没有把 ZITADEL 并进业务库 `ruoyi-vue-pro`。身份库和业务库仍然分
离，只是共用同一 PostgreSQL 实例。

旧资源：

- `StatefulSet/zitadel-postgres16`
- `Service/zitadel-postgres16`
- `Service/zitadel-postgres16-hl`
- `Secret/zitadel-postgres16`
- `PVC/data-zitadel-postgres16-0`

已经在 `2026-05-16` 清理完成。当前没有“切回旧 Pod/旧 PVC”的快速回滚点。
如果后续需要回退，只能基于外部备份、导出文件或重新创建独立实例后再恢复。

上述清单仅供各自已获授权的运维使用；旧刊登 staging/prod 不再提供运行部署步骤。

旧 Amazon API/LB/专属监控及发布源按[完整边界](../../docs/architecture/legacy-numeric-acquisition-tail-closure-retirement-boundary.md)退出；共享 Namespace/凭据、WeCom 与 Prometheus 存储窄路由保留。未查询或变更实际集群。

新建本地应用从 [Account Compose](../docker/account-compose/README.md) 进入。集群接入凭据由当前运维凭据渠道提供，不写入文档或命令示例。
