# 旧 Listing Control Plane：源码退休与运行态边界

Status: **RETIRED source entrypoints**（#726，#29 / #301）。
用户已确认[退休设计 §5、§6、§6.1、§7](../architecture/legacy-listing-control-plane-retirement-boundary.md)。
本说明替代原控制面构建、发布、运行和 smoke 操作；原操作仅留在 Git 历史。

## 已退出的源码入口

`listing-control-plane`、`shein-import-platform-recovery` 及专属 runtime/control 库、
一次性平台归一 helper/DTO/测试、Dockerfile/操作脚本和 control-plane workload 已退休。
不存在可沿旧命令或旧 build/push/deploy 步骤启动的当前源码入口，不提供 forwarding stub。
两套 workflow 精确移除旧 runtime race，主 CI 移除旧 build；原仓库并发安全和其它必需检查保持。

## 保留的 owner 与当前入口

任务/事件仓库、claim-token rollback、eligible-row recovery 及其并发/owner/tenant 安全检查、
有效 ListingAdmin/根 ListingRuntime 合同与共享配置留原 owner；旧 task/status 随后已按
[#729 范围](../architecture/legacy-numeric-task-execution-retirement-boundary.md)退出源码，不转移原持久化责任。
仍保留的旧仓库债务不是新当前业务依赖或第二事实源。

前批保留的历史 ownership-controller 声明，随后已按 #738 的[确认边界](../architecture/legacy-ownership-controller-retirement-boundary.md)
连同专属配置 overlay 和 auto-shard 配置闭包源码退休；没有更换或运行该历史镜像。
其它有效共享配置、Secret/RBAC/监控对象与原任务仓库安全合同保持。
源码清单不证明实例不存在、健康、停用或在途副作用已解决。

当前应用继续沿原私密 manifest 入口；采集、Store 观察同步及图片/Temporal 原合同保持。
独立 SHEIN 登录 worker 保留原命令、账号仓库、Cookie/attempt/验证码/Profile、
tenant-qualified Store 查重、health/shutdown 与恢复合同。参见[入口地图](../development/repository-structure.md)
和[worker 身份/持久化边界](../architecture/shein-worker-identity-persistence-boundary.md)。

## 运行态与数据停止点

本批没有启动、停止、重启、部署或销毁实例，没有查询/修复真实 DB/Redis/RabbitMQ、
执行 recovery dry-run、重放/清空消息、读取凭据/Profile 或调用浏览器/provider。
旧表、行、索引、key、journal、消息、镜像、卷、备份、日志及所有历史资源保持。
真实旧运行态和在途业务若需处置，由其原 owner 另获明确授权；不得隐式转交当前应用或登录 worker。
这不是当前内部试用的新前置。真实运行/用户验收 NOT_RUN，生产 NOT_READY；#29/#301保持 OPEN。
