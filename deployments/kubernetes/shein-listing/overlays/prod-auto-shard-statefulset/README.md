# 原 SHEIN shard overlay：源码入口已退休

Status: **RETIRED source entrypoints**（#723 / #726 / #738，#29 / #301）。

原 shard workers、scheduler、control-plane 和 ownership-controller 的 workload
均已从源码退出；本目录已无 Kustomize 装配入口。旧 `/app/listing-service` 历史
镜像不由当前源码重建，不提供替代协调器、空部署或恢复旧消费链的入口。

ownership-controller 和其唯一 kustomization 按[已确认边界](../../../../../docs/architecture/legacy-ownership-controller-retirement-boundary.md)
退休。特定 ConfigMap 源文件在确认无其它仓库消费者后随专属配置 overlay 退出；
共享 Secret/ExternalSecret、Namespace、RBAC及独立 SHEIN 登录 worker 保持原 owner。
#839 [完整运维源码闭包](../../../../../docs/architecture/legacy-listing-operations-closure-retirement-boundary.md)
进一步退出旧专属监控与 rollout helper/test；唯一有效存储告警抽取到现有 monitoring。
当前采集和 Store 观察同步沿原入口使用。

这是源码退休，不证明任何实例已停止、锁已释放或在途任务已处理；没有查询或改动
实际集群、镜像、ConfigMap/Secret、DB/Redis/队列/Profile/数据。真实运行态仍由
原 owner 在另行明确授权后处置，不增加内部试用前置。生产 NOT_READY。
