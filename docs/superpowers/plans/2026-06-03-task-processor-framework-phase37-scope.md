> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 37 Scope Recommendation

### Recommendation

下一步更值得做的是 `ListingKit action target clone aggregate ownership`。

也就是继续处理：

- [task_generation_action_target_clone.go](/D:/code-task-processor/internal/listingkit/task_generation_action_target_clone.go:1)

这一层 currently aggregated clone-shaping 语义，而不是继续围着 shared helper home 做更细的对称性清理。

### Why This Before Other Options

#### 1. Shared helper home 已经收干净了

现在：

- shared helper 定义已经有单独 home
- direct consumers 继续稳定地调用 shared seam

继续在 helper home 本身深挖，收益已经明显下降。

#### 2. Action-target clone 是下一个明显的 aggregate hotspot

当前 `cloneAssetGenerationActionTarget(...)` 仍然同时知道：

- top-level target copy
- filters clone
- queue clone
- retry clone
- expected impact clone
- navigation target clone

这已经不是单一 helper 调用了，而是明确的 aggregate-shape owner。

#### 3. 这个切片依然能保持 bounded

下一步可以只围绕：

- action target clone aggregate
- nested clone delegation
- current consumer-visible behavior stability

来做，不需要马上扩大到 navigation dispatch 执行流本身。

### Explicitly Not Recommended Next

#### 1. 不建议继续在 shared helper home 上追求更细对称性

例如再拆成 queue helper 文件和 retry helper 文件，这种收益已经不高。

#### 2. 不建议直接重开 broader navigation dispatch cleanup

那会显著扩大写面，超出当前最自然的小切片。

### Proposed Next Phase Name

建议下一阶段命名为：

`Phase 37: ListingKit action target clone aggregate ownership`
