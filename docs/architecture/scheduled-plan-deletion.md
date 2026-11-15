# 定时计划删除：保留结果的列表移除

## Product outcome / Scope

2026-11-15 用户决定：定时任务支持删除，停止未来执行并从默认列表移除，保留已有报告与执行记录，不取消在途任务、不更改费用或预留。结束与删除均用现有统一确认弹窗。不调用模型、不发布商品、不操作原预留。本轮实现和既有夹具验证，不部署原 35444、不合并或关闭 Issue。

UI/IA 依据当前用户决定，Figma 仅作业务流程参考。已有合同：[定时任务](ai-workbench-scheduled-tasks.md)、[店铺日报](scheduled-store-daily-report.md)、[领域 owner](project-boundaries.md)。

## Owner / Contract / Wiring

aiworkbench 仍唯一拥有计划、发生记录及控制回执；reportcenter 拥有报告；Agent/Commercial 仍拥有实际任务、计量与预留。无新 owner、服务或恢复平台，无兼容 adapter / 双写（Legacy N/A）。

路径：ScheduleManager → 既有 Workbench BFF 白名单 → `POST /schedules/{id}/delete` → 当前身份/成员及 AgentUse 权限 → 原 ChangeScheduledPlan → 原命令回执事务 → 原 ScheduleBackend.Synchronize / recovery。删除读取/历史仍经当前 scope 与原授权，不新增管理员越权读口。

## 持久化与状态

- 在原 scheduled_plans 增加 `deleted boolean NOT NULL DEFAULT false`，约束 `NOT deleted OR state='CLOSED'`；不删除任何业务行，不改外键、计划身份、规则、次数、预算和 active_occurrence_id。
- 不引入新调度状态：删除事务原子设置 CLOSED、deleted=true、revision+1、registration_state=PENDING，并在原 scheduled_plan_commands 保存成功回执。关闭原本仍不隐藏。
- ACTIVE / PAUSED / 登记未定 / CLOSED 均可删除；CLOSED 也可删除，deleted 后不可编辑、恢复或暂停。准确同键重放先读原回执；同键不同载荷仍冲突。新键重删相同 revision 可返回原 deleted 计划，不再递增或改变占用；旧 revision 拒绝。并发不同键竞争原计划行锁与 revision，最多一次有效改变。
- 当前计划列表查询默认 `deleted=false`，先筛选再分页；不增加回收站/还原 UI。按 ID 查看和历史读取保留，显示“已删除”。成功后返回列表；不确定响应保留原命令与 key，可重试核实；不错误提示删除成功。

## 在途 / 到点竞争 / 恢复

复用原 close 的行锁和历史漏槽推进（含既有有界 catch-up 进度）。删除先提交则新领取因 CLOSED 拒绝；领取先提交则已开始任务/日报照原流程完成。仅迁移未来执行权，不取消 workflow / Agent，不调用计量或释放 owner。已存在 occurrence 的重放仍可恢复同一任务。

已删除 CLOSED 计划仍由原 recovery 扫描未匹配登记及 active_occurrence_id；不得加 deleted 过滤到恢复扫描。Temporal 继续投影为 paused，迟到回调仍受数据库状态检查。同步失败时 deleted/CLOSED 和回执已持久化，后台继续核实，未来执行由 DB 阻断；不恢复 ACTIVE。原 occurrence reconciliation 如达到次数上限不得覆盖 deleted 标记，现有 CLOSED 状态可保持。

## 安装 / 权限

干净安装使用当前 owner schema；已有开发库通过现有 `ai-workbench-schema-init` 的有界 `--schedule-deletion-only` 入口显式安装。仅接受明确数据库/地址/端口及原受限 role，事务/advisory lock 内添加列和约束，授予原 runtime 对 deleted 列的 UPDATE；不赋表级 UPDATE / DELETE，不改其它权限或数据。缺失扩展时 serving 只读检查拒绝启动，不自动 DDL。日报精确约束检查同步增加该约束；其它边界与约束不放宽。

安装仅改变 schema/授予该列权限，保留所有行、历史和报告；本轮只在原有测试夹具安装，不触碰原 35444。回退源码前须评估新约束对旧精确 verifier 的影响，不能宣称旧版无条件可启动；本轮不承诺或执行部署回退。

## UI

复用 ResourceDialog（已有 Esc、焦点恢复、主题、原生 dialog focus trap），结束/删除共享该弹窗。默认焦点为取消；明确“已开始的任务继续执行，报告和历史保留”。用户确认后才发送控制请求，不自动调用模型。删除完成后列表空态不再提示只能选商品；已删除详情仍提供返回与历史。

## 验证与评审

只扩展既有 PostgreSQL / HTTP / ScheduleManager / BFF 合同测试，TDD：先证明现状不支持删除，再实现。覆盖：进行中及 CLOSED 删除；默认列表含分页不返回删除项；原报告/历史/任务/占用及预算不变；跨 scope 拒绝；同键重放/并发/响应丢失恢复；在途能完成/未来拒绝；schema 约束与 runtime 权限；确认、取消、Esc、焦点恢复。复用未改变调度与模型保护的已有证据，不新增测试平台、真实模型调用或部署。

Architecture Review：待独立检查；Admission Status: NOT_READY。实施必须在独立复核确认 IMPLEMENTATION_READY 后开始。
