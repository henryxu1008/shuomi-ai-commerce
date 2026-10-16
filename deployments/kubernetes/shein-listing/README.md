# SHEIN 旧刊登清单：源码退休与保留边界

Status: **RETIRED listing entrypoints / retained shared declarations**（#723，#29 / #301）。
`cmd/shein-listing`、`cmd/listing-scheduler`、Dockerfile.listing 及其专属 build/push/deploy、
单店同步、shard rollout 已退休。旧刊登 Deployment/DaemonSet/StatefulSet、scheduler、
专属 Service/HPA/patch 和单店模板已退出，旧构建/部署步骤仅保留于 Git 历史。

本目录混有保留对象，不能整体删除或当作当前应用发布包：

- 原 Namespace、ConfigMap、Secret/ExternalSecret 及相邻监控/安全声明保持。
- `overlays/prod-auto-shard-statefulset` 保留 control-plane 和 ownership-controller 声明；
  原 shard worker 与 scheduler 已移除，详见[该目录说明](overlays/prod-auto-shard-statefulset/README.md)。
- control-plane 仍沿原 Dockerfile.listing-control-plane 与
  `scripts/build-push-deploy-listing-control-plane.ps1`；参见[原 runbook](../../../docs/operations/listing-control-plane-runbook.md)。
- 独立 SHEIN 登录 worker 使用原 `cmd/shein-login-worker` 和 Dockerfile.shein-login-worker，
  账号仓库、Redis/Profile、Cookie/attempt、健康及关闭合同保持；[身份/持久化边界](../../../docs/architecture/shein-worker-identity-persistence-boundary.md)不变。

当前采集、Store 观察同步和当前应用入口见[仓库入口地图](../../../docs/development/repository-structure.md)。
配置对象存在或 apply 成功不表示旧刊登 runtime 可用；不得用旧 YAML 替代当前应用私密 manifest。
[批准边界](../../../docs/architecture/legacy-listing-cli-retirement-boundary.md)只处理源码，
不证明实例已停用、不清理任何资源/数据，也不增加内部试用前置。生产 NOT_READY。
