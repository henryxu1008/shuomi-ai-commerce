# 原 SHEIN shard overlay：保留的 ownership-controller 声明

Status: **listing workers / scheduler / control-plane RETIRED；ownership-controller retained**（#723 / #726）。
原 SHEIN shard StatefulSet、headless/worker Service 和 listing-scheduler Deployment
已退休；这里不再提供 shard 扩容、消费者启动或调度器发布操作。

当前 kustomization 仅保留：

- `ownership-controller.yaml`：原 ownership-controller 历史镜像及共享配置/身份声明。
  它仍引用既有 `/app/listing-service` 制品，**不是当前源码可重新构建的入口**；
  本批没有替换该镜像、重建协调器或修改其运行协议。

目录名沿用历史路径，不表示 worker 池仍由当前源码提供。保留声明也不证明任何实例
存在、健康或可发布；真实旧生产者/消费者与在途副作用的停用由原运行 owner 另行取得
明确授权，不是当前内部试用的新前置。不得整体清理这个混合目录或真实 Secret/资源/数据。

control-plane workload及其构建/部署入口已源码退休，不能沿历史操作启动；
参见[运行态边界](../../../../../docs/operations/listing-control-plane-runbook.md)和
[已确认退休边界](../../../../../docs/architecture/legacy-listing-control-plane-retirement-boundary.md)。
