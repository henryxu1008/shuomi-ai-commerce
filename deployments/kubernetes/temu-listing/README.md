# TEMU 旧刊登清单：源码退休

Status: **RETIRED entrypoint / retained resource declarations**（#723，#29 / #301）。
`cmd/temu-listing` 及专属消费/调度链已物理退休，不再是受维护 runtime。
本目录不能用于部署当前应用，也没有可启动旧 TEMU 消费者的 Deployment、Service 或 HPA。

保留的 `base` / staging / prod 内容是原 Namespace、ConfigMap、Secret 示例和
ExternalSecret 声明；kustomization 仅引用这些保留对象。目录存在不表示旧工作负载可部署，
不应将配置对象 apply 成功当作运行交付。旧部署操作说明只保留于 Git 历史。

当前入口与使用方式见[仓库入口地图](../../../docs/development/repository-structure.md)；
当前 Store 观察与官方执行使用各自原 owner，不使用这里的数字任务队列作为 fallback。
[批准的退休边界](../../../docs/architecture/legacy-listing-cli-retirement-boundary.md)
不授权停用旧实例、清理队列、配置/凭据或数据，也不增加内部试用前置。生产 NOT_READY。
