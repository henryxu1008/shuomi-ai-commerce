# 原 SHEIN shard overlay：保留的控制面声明

Status: **listing workers / scheduler RETIRED；control-plane declarations retained**（#723）。
原 SHEIN shard StatefulSet、headless/worker Service 和 listing-scheduler Deployment
已退休；这里不再提供 shard 扩容、消费者启动或调度器发布操作。

当前 kustomization 仅保留：

- `listing-control-plane.yaml`：原 control-plane 镜像、配置引用、健康与运行合同。
- `ownership-controller.yaml`：原 ownership-controller 历史镜像及共享配置/身份声明。
  它仍引用既有 `/app/listing-service` 制品，**不是当前源码可重新构建的入口**；
  本批没有替换该镜像、重建协调器或修改其运行协议。

目录名沿用历史路径，不表示 worker 池仍由当前源码提供。保留声明也不证明任何实例
存在、健康或可发布；真实旧生产者/消费者与在途副作用的停用由原运行 owner 另行取得
明确授权，不是当前内部试用的新前置。不得整体清理这个混合目录或真实 Secret/资源/数据。

有效 control-plane 操作继续参考[原 runbook](../../../../../docs/operations/listing-control-plane-runbook.md)；
范围与限制见[已确认退休边界](../../../../../docs/architecture/legacy-listing-cli-retirement-boundary.md)。
