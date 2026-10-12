# 企业智能体四种执行权限：接入与保留数据更新

依据 #656 和 [已批准设计](../architecture/enterprise-agent-permission-modes-v1.md)。本说明交接当前候选，代码检查、CI、独立复核、安装核对和用户验收分别记录；不得将任一项替代其它项。

## 入口与行为

- 我的智能体中，打开已启用的标题/图片智能体配置页，或实际已交付的私有质检智能体，使用「执行权限 → 配置权限」。只有当前企业具备配置授权的成员可保存。
- 默认权限保存建议并保留人工采用；自动审批只自动采用已通过现有程序校验的标题，图片保真仍人工确认；只读保留分析与建议，禁止商品/草稿采用和图片生成；完全访问允许已授权业务工具的本地采用，记录「按策略采用，未经逐项人工审阅」。
- 发布、改价、下架、删除、增加付费额度仍须另外授权。权限模式不授予 IAM、资源、店铺、平台或模型凭据权限。图片计划和预算确认仍保留。
- 直接采集页面、对话提案、供应链优化和 worker 使用同一模式事实。每次新执行冻结模式修订和不可变引用；改变模式不会升级已有运行。收到结果未知时保留原请求编号与参数，核实原操作。

## 保留当前安装的数据

只由已获授权的安装 owner 在最终准确 HEAD、必要 CI 与独立复核完成后执行。先停止对应应用和 worker 写入，保留所有命名 volumes、原配置、既有 owner/runtime 角色与启动文件；不要执行 `down -v`、空库初始化、清理或数据库重建。备份与更新前后数据核对使用原安装流程。

schema-init 镜像中的两个既有命令新增有界更新选项。通过私密环境分别注入原 ProductAgentDB 的维护连接；不要输出或提交 DSN/密码。两个 runtime 角色分别沿用原安装配置，不新建角色，不更改其跨 owner 授权。

```powershell
go run ./cmd/agent-configuration-schema-init --runtime-role <existing_agent_configuration_runtime_role> --execution-modes-only
go run ./cmd/ai-workbench-schema-init --runtime-role ai_workbench_runtime --execution-mode-only
```

第一条读取 `AGENT_CONFIGURATION_SCHEMA_DSN`，仅增加 `agent_configuration.execution_modes`、`execution_mode_snapshots` 和这些表的 SELECT/INSERT、模式 head 的指定列 UPDATE 授权。要求原 non-login `agent_configuration_owner` 已存在且拥有 schema；拒绝替换其它 owner。现有命令、配置、模板、运行事实和它们的授权不变。

第二条读取 `AI_WORKBENCH_SCHEMA_DSN`，仅为原 `ai_workbench.execution_proposals` 增加有约束的 nullable `observed_mode_revision`。旧提案保持 NULL、原 digest 和可读历史，不能作为新权限授权；必须重新提出方案。现有表级授权自然覆盖新字段，不撤销或扩展原写权限。

维护命令均可重复执行，不回填模式、不启用智能体、不创建业务记录。更新后应用和 worker 必须从同一最终候选重新构建/重启，以重建数据库连接和 SQL 缓存。沿用当前 unified `data-services-runtime` 和 UI 构建/启动参数，不改用缺少采集依赖的目标。

## 核对与回退

安装 owner 核对原商品、店铺、角色授权、命名 volumes 与 worker poller，再检查模式保存→刷新、原请求重放、只读拒绝采用和后台同源校验。不得调用尚未由用户准备的真实模型、支付、SHEIN/SDS。

需要回退时停写、恢复原镜像及原启动配置，并保持本批次受影响的智能体执行路径停写，保留新增表/列和所有事实，不删除或反向改写记录。新候选写入的模式和策略采用记录不能被旧代码当成人工批准；再次升级时沿原记录读取/核实，不能重新生成或重复采用。真实 provider、保留程序页面验收及用户验收尚未执行的项目记录 `NOT_RUN`。
