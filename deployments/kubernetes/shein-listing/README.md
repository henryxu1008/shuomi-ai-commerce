# SHEIN 旧刊登清单：源码退休与保留边界

Status: **RETIRED legacy listing and numeric proxy operational sources / retained shared namespace**（#839 / #872，#29 / #301）。
`cmd/shein-listing`、`cmd/listing-scheduler`、Dockerfile.listing 及其专属 build/push/deploy、
单店同步、shard rollout 已退休。旧刊登 Deployment/DaemonSet/StatefulSet、scheduler、
专属 Service/HPA/patch 和单店模板已退出，旧构建/部署步骤仅保留于 Git 历史。

本目录混有保留对象，不能整体删除或当作当前应用发布包：

- 本目录仅保留共享 Namespace、只装配 Namespace 的 base/kustomization.yaml 及退休说明。原 ConfigMap、staging、空单店/分片装配入口已退出。
- `overlays/prod-auto-shard-statefulset` 的 ownership-controller 及唯一装配也已按 #738 退出；
  原 shard worker、scheduler、control-plane 均已退休，详见[该目录说明](overlays/prod-auto-shard-statefulset/README.md)。
- `overlays/prod-store-auto-shard` 的两个专属 ConfigMap 及装配已退出；本批进一步退休旧刊登监控，唯一有效 Prometheus 存储规则归属现有 monitoring，见[完整闭包](../../../docs/architecture/legacy-listing-operations-closure-retirement-boundary.md)。
- #872 按[完整尾部边界](../../../docs/architecture/legacy-queue-proxy-config-tail-closure-retirement-boundary.md)同批退休旧数字 Store 代理脚本/CronJob/专属 RBAC、失去最后源码消费者的四份 shein-listing-secret 源模板及 prod 装配。没有替代凭据模板或部署入口，没有读取/删除真实 Secret、远端属性、旧数据库、实例或进程；物理状态保持 UNKNOWN。
- control-plane专属Dockerfile及build/push/deploy脚本已退出，历史操作不可按当前源码执行；
  参见[源码退休与运行态边界](../../../docs/operations/listing-control-plane-runbook.md)。
- 独立 SHEIN 登录及登录后 Store 身份回写已按 [#985 完整边界](../../../docs/architecture/legacy-shein-login-identity-sync-closure-retirement-boundary.md)从源码退休；旧 CLI/Dockerfile/配置/专属部署入口不再可用。Store 观察同步与官方授权保持。

当前采集、Store 观察同步和当前应用入口见[仓库入口地图](../../../docs/development/repository-structure.md)。
配置对象存在或 apply 成功不表示旧刊登 runtime 可用；不得用旧 YAML 替代当前应用私密 manifest。
[批准边界](../../../docs/architecture/legacy-listing-operations-closure-retirement-boundary.md)只处理源码，
不证明实例已停用、不清理任何资源/数据，也不增加内部试用前置。生产 NOT_READY。
