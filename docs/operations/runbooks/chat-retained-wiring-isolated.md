# 35444 Chat 保留库：隔离接线验证

> 归类说明（2026-11-19）：本页保留正式原实例入口接线前的隔离验证记录；下文“`--mode apply` 始终拒绝”是历史阶段描述，不代表现行命令。当前 `cmd/ai-workbench-retained-prepare/main.go` 已将 `apply` 分流至受明确原目标确认和正式准入保护的入口，`apply-isolated` 仍只接受登记的无网络副本。正式范围见[接线设计](../../architecture/domains/workbench/chat-35444-retained-wiring-design.md)与[最小准入](../../architecture/domains/workbench/chat-retained-minimal-admission-proposal.md)。
> [#1263](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1263)记录原 35444 已按当次授权接线启动，不是本页命令获得原实例执行许可，也不等于完整产品验收。后续最小准入仅局部替代正式入口中无关 owner 重复全量备份的固定条件，不取消目标、冻结、事务、原调用和预留检查，不放宽隔离入口自己的来源/恢复证据。此页归入维护 runbooks 而非终端用户指南；所有下文命令仅保留记录，本次不执行、不读取私有回执或变更实例，不调用模型或释放预留。

本入口目前只支持已登记、网络隔离的试用副本；`--mode apply` 始终拒绝，**不是原 35444 的更新入口**。需另外完成真实目标入口、真实各 owner 备份恢复及冻结证明，并取得实际更新授权后，才可讨论原实例接线。

来源观察、备份和恢复回执须由受信本机维护者通过实际操作取得：分别固定源容器及 PostgreSQL 身份、dump 摘要、隔离容器完整 ID、镜像、`network none`、目标 database OID、还原结果和维护窗口。旧脚本或自填标签不能证明历史备份来源。私有记录及引用文件须满足现有 `VerifyPrivateFiles` 的权限要求。

显式执行隔离入口时，先按当前代码构建 Windows 宿主命令和 `CGO_ENABLED=0 GOOS=linux GOARCH=amd64` 的 Linux helper。由维护者准备与来源观察、还原回执和目标事实/权限摘要匹配的私有 manifest 与 admission 文件，调用：

```text
ai-workbench-retained-prepare --mode apply-isolated --manifest <private-manifest> --isolated-source <private-source-observation> --isolated-receipt <private-restore-receipt> --isolated-pin <private-isolated-pin> --isolated-dump <private-dump> --isolated-admission <private-admission> --isolated-helper <linux-helper>
```

宿主核对受保护记录、源及目标的实时身份、dump 和 helper 摘要、各 owner 记录及其最早到期时间；仅对与既有隔离 pin 一致且无外部网络、无外部 PostgreSQL socket 挂载的目标执行。子进程消费一次性 root 专用文件并降至 `postgres` 系统用户，在同一数据库连接上核对目标及命名空间，工作台表锁内再次核对原结构、权限、事实、在途状态与期限。成功后用受限 runtime 角色验证 schema。

隔离夹具可以使用明确标记的合成 owner 记录，验证绑定和拒绝行为；这些记录不证明原身份库、图片、Temporal 或商业账本的实际备份、还原或冻结。散列和记录也不能自动保证 owner 持续维持冻结，冻结失效时维护者必须停止操作。

收到执行异常或超时，**不得盲目重试或假定回滚**。保留同一目标和原记录，先以只读方式检查原结构或目标结构、原事实、权限与对应 owner 状态；混合状态或证据不一致时停止并调查。`INSPECT_STRUCTURE_ONLY` 只报告结构和随附摘要的匹配，不是目标来源、owner 证明或更新放行。
