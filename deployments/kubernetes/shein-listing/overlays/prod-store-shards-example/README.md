# SHEIN Store Shards Example（历史目录）

旧 `shein-listing` CLI、店铺分片 Deployment 与专属派发链已按 #723/#726 退休；本目录不再是当前源码的部署入口。#836 的[当前系统边界](../../../../../docs/architecture/project-boundaries.md)进一步退出旧 RabbitMQ 节点/店铺消费配置、env 绑定和仓库 Secret 中的旧 URL 声明。该旧链不再支持按节点、店铺队列或 ownedStores 调度，不能据此执行 apply 或迁移步骤。

本目录的 Kustomize 装配已按 #839 [当前系统边界](../../../../../docs/architecture/project-boundaries.md)退出。#872 的[当前系统边界](../../../../../docs/architecture/project-boundaries.md)进一步退休旧数字 Store 代理两入口及其专属声明；失去最后源码消费者的 `secret.example.yaml` 同批退出，本目录仅保留历史说明。没有读取/修改真实 Secret、队列、数据库或部署，历史示例不构成运行证据。

当前独立 SHEIN 登录 worker、Redis/profile、组织/UUID Store隔离、私密Binding/ProxyFile与Store同步仍由[#863当前owner](../../../../../docs/architecture/project-boundaries.md)提供，未转为上述旧刊登/分片/数字代理链；当前采集、平台 API、身份、计费与图像能力不变。实际入口与配置沿各自当前文档，不以旧分片配置作为内部试用前置。
