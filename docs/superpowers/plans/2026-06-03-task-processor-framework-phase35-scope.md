> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 35 Scope Recommendation

### Recommendation

下一步更值得做的是 `ListingKit action execute handoff branch request shaping ownership`。

也就是继续处理下面两条 branch invocation seam：

- [task_generation_action_execute_request_handoff_retry.go](/D:/code-task-processor/internal/listingkit/task_generation_action_execute_request_handoff_retry.go:1)
- [task_generation_action_execute_request_handoff_queue.go](/D:/code-task-processor/internal/listingkit/task_generation_action_execute_request_handoff_queue.go:1)

当前它们都还是 mirrored thin shells，同时知道：

- branch service call
- request clone helper handoff

而 `Phase 34` 刚刚收掉的 branch result routing 已经不再是最强的 residual pressure。

### Why This Before Other Options

#### 1. Result side 已经足够清楚

现在 result side 已经稳定成：

- branch-local result owner
- local result-dispatch owner
- unified result-normalization owner
- unified result-shape owner
- adaptation owner

继续往 result side 深挖，收益已经明显下降。

#### 2. Branch request shaping 仍然是明确的 mixed responsibility

当前 retry / queue branch invocation seam 仍然同时持有：

- clone helper
- service invocation

这比去追求 result side 更深的对称性，更像真实的 ownership hotspot。

#### 3. 这个切片仍然足够 bounded

这一步可以只围绕：

- `cloneRetryGenerationTasksRequest(...)`
- `cloneGenerationQueueQuery(...)`
- branch invocation seam

来做，不需要扩大成 execute / refresh / projection 的新一轮清理。

### Explicitly Not Recommended Next

#### 1. 不建议继续深挖 result-dispatch seam

- [task_generation_action_execute_request_handoff_result_dispatch.go](/D:/code-task-processor/internal/listingkit/task_generation_action_execute_request_handoff_result_dispatch.go:1)

当前它已经是一个足够窄的本地 pairing owner。现在继续拆它，更像为了对称性而拆。

#### 2. 不建议重开 shared clone helper home

- [service_generation_actions.go](/D:/code-task-processor/internal/listingkit/service_generation_actions.go:1)

这会明显扩大写面，也会把范围从 handoff 本地 slice 扩成 cross-consumer helper relocation。

### Proposed Next Phase Name

建议下一阶段命名为：

`Phase 35: ListingKit action execute handoff branch request shaping ownership`
