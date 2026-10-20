# 原 SHEIN store auto-shard 配置 overlay：源码退休

Status: **RETIRED exclusive configuration sources**（#738，#29 / #301）。

heavy ConfigMap 的唯一仓库消费为已退 ownership-controller，lite ConfigMap 无
其它仓库消费者；原 kustomization 只装配这两份文件。按[已确认删除矩阵](../../../../../docs/architecture/legacy-ownership-controller-retirement-boundary.md)
退出两个专属 ConfigMap 源文件及其装配，本目录不再提供旧配置部署操作。

原内容包含通用字段不表示该特定资源仍是当前共享合同。有效 core/config、Viper
解析、通用 RabbitMQ 基础层、平台/API/采集和独立 SHEIN 登录 worker 仍在各自
原 owner 下保留；不把旧样例包装为新部署模板，不新增协调/兼容/恢复入口。

#839 [完整运维源码闭包](../../../../../docs/architecture/legacy-listing-operations-closure-retirement-boundary.md)
退出其余旧刊登 ConfigMap/空 overlay 装配及专属监控；Store 代理同步仍使用的
shein-listing-secret 源、共享 Namespace 和独立登录/Store 同步保持。

没有操作实际 ConfigMap、Secret、实例、镜像、Redis ownership/锁、DB、消息或
其它数据。仓库无消费者不等于真实集群无消费者；真实状态和持久化 owner 未转移，
由原运行 owner 另获明确授权后处理，不增加内部试用前置。生产 NOT_READY。
