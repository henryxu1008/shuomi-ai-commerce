# SHEIN 旧刊登清单：源码退休与保留边界

Status: **RETIRED legacy listing operational sources / retained Store credential assembly**（#839，#29 / #301）。
`cmd/shein-listing`、`cmd/listing-scheduler`、Dockerfile.listing 及其专属 build/push/deploy、
单店同步、shard rollout 已退休。旧刊登 Deployment/DaemonSet/StatefulSet、scheduler、
专属 Service/HPA/patch 和单店模板已退出，旧构建/部署步骤仅保留于 Git 历史。

本目录混有保留对象，不能整体删除或当作当前应用发布包：

- 本目录仅保留共享 Namespace、四份 Secret/ExternalSecret 源、base/prod 共享凭据装配及退休说明。原 ConfigMap、staging、空单店/分片装配入口已退出。
- `overlays/prod-auto-shard-statefulset` 的 ownership-controller 及唯一装配也已按 #738 退出；
  原 shard worker、scheduler、control-plane 均已退休，详见[该目录说明](overlays/prod-auto-shard-statefulset/README.md)。
- `overlays/prod-store-auto-shard` 的两个专属 ConfigMap 及装配已退出；本批进一步退休旧刊登监控，唯一有效 Prometheus 存储规则归属现有 monitoring，见[完整闭包](../../../docs/architecture/legacy-listing-operations-closure-retirement-boundary.md)。
- `shein-listing-secret` 仍由 Store 代理同步消费，四份源声明、字段、远端映射、namespace 和窄 RBAC 保持。保留的 prod 装配不部署刊登工作负载，不是当前应用发布入口；本批未应用任何清单。
- control-plane专属Dockerfile及build/push/deploy脚本已退出，历史操作不可按当前源码执行；
  参见[源码退休与运行态边界](../../../docs/operations/listing-control-plane-runbook.md)。
- 独立 SHEIN 登录 worker 使用原 `cmd/shein-login-worker` 和 Dockerfile.shein-login-worker，
  账号仓库、Redis/Profile、Cookie/attempt、健康及关闭合同保持；[身份/持久化边界](../../../docs/architecture/shein-worker-identity-persistence-boundary.md)不变。

当前采集、Store 观察同步和当前应用入口见[仓库入口地图](../../../docs/development/repository-structure.md)。
配置对象存在或 apply 成功不表示旧刊登 runtime 可用；不得用旧 YAML 替代当前应用私密 manifest。
[批准边界](../../../docs/architecture/legacy-listing-operations-closure-retirement-boundary.md)只处理源码，
不证明实例已停用、不清理任何资源/数据，也不增加内部试用前置。生产 NOT_READY。
