# SHEIN Store Shards Example（历史目录）

旧 `shein-listing` CLI、店铺分片 Deployment 与专属派发链已按 #723/#726 退休；本目录不再是当前源码的部署入口。#836 的[完整配置退休设计](../../../../../docs/architecture/legacy-runtime-config-closure-retirement-boundary.md)进一步退出旧 RabbitMQ 节点/店铺消费配置、env 绑定和仓库 Secret 中的旧 URL 声明。该旧链不再支持按节点、店铺队列或 ownedStores 调度，不能据此执行 apply 或迁移步骤。

本目录保留的 `secret.example.yaml` 仅是静态占位模板，其它当前配置项保持；本文不授权修改真实 Secret、队列或部署。历史示例不构成运行证据。

当前独立 SHEIN 登录 worker、Redis/profile、账户隔离与 Store 同步仍由原 owner 提供，未转为上述旧刊登/分片链；当前采集、平台 API、身份、计费与图像能力不变。实际入口与配置沿各自当前文档，不以旧分片配置作为内部试用前置。
