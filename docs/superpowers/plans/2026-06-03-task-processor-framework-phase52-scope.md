> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 52 Scope Recommendation

### Recommendation

下一步更值得做的是 `ListingKit shared retry request clone aggregate ownership`。

也就是继续处理：

- [task_generation_shared_clone.go](/D:/code-task-processor/internal/listingkit/task_generation_shared_clone.go:1)

里现在还留着的：

- `cloneRetryGenerationTasksRequest(...)`

### Why This Before Other Options

#### 1. Shared queue query clone 已经收干净了

现在：

- queue query clone 已有独立 home
- retry request clone 成了 shared clone 邻域里剩下最显眼的 aggregate home

继续在 queue query 这边做更细对称性，收益已经明显下降。

#### 2. Retry request clone aggregate ownership 是下一个最自然的小切口

当前 retry request clone 仍然同时知道：

- top-level request shallow copy
- `TaskIDs / Slots` slice clone

这已经是很明确的 ownership hotspot，而且写面很小。

#### 3. 这个切片仍然足够 bounded

下一步可以只围绕：

- retry request clone
- current direct consumers
- current consumer-visible behavior stability

来做，不需要马上扩大到 broader action execute orchestration 或 navigation dispatch flow。

### Explicitly Not Recommended Next

#### 1. 不建议回头重开 shared queue query clone home

这个 ownership 问题已经解决了。

#### 2. 不建议直接重开 broader descriptor clone entry / navigation dispatch flow

那会显著扩大写面，超出当前最自然的小切片。

### Proposed Next Phase Name

建议下一阶段命名为：

`Phase 52: ListingKit shared retry request clone aggregate ownership`
