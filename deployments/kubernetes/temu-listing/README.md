# TEMU 旧刊登清单：源码退休

Status: **RETIRED operational sources**（#839，#29 / #301）。
`cmd/temu-listing` 及专属消费/调度链已物理退休，不再是受维护 runtime。
本目录不能用于部署当前应用，也没有可启动旧 TEMU 消费者的 Deployment、Service 或 HPA。

原 `base` / staging / prod 的 ConfigMap、无仓库 reader 的 TEMU Secret/ExternalSecret、
独立重复 Namespace 源和全部 Kustomize 装配已退出，本目录仅保留本说明。
共享 task-processor Namespace 的其它当前声明保留；没有删除或操作实际 Namespace、
Secret、远端属性或任何实例/数据。旧部署源和操作说明只保留于 Git 历史。

当前入口与使用方式见[仓库入口地图](../../../docs/development/repository-structure.md)；
当前 Store 观察与官方执行使用各自原 owner，不使用这里的数字任务队列作为 fallback。
[批准的完整退休边界](../../../docs/architecture/legacy-listing-operations-closure-retirement-boundary.md)
不授权停用旧实例、清理队列、配置/凭据或数据，也不增加内部试用前置。生产 NOT_READY。
