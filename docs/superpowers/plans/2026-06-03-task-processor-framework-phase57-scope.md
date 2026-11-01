> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 57 Scope Recommendation

### Recommendation

下一步更值得做的是 `ListingKit action target impact slice clone ownership`。

也就是继续处理：

- [task_generation_action_impact_clone_shape.go](/D:/code/task-processor/internal/listingkit/task_generation_action_impact_clone_shape.go:1)

里现在共同存在的：

- `Platforms` slice clone
- `QualityGrades` slice clone
- `States` slice clone

### Why This Before Other Options

#### 1. Action target impact aggregate home 已经收干净了

现在：

- aggregate home 已只保留 top-level shallow copy
- shape home 已接住 impact slice clone shaping

继续在 aggregate home 这层追求更细对称性，收益已经明显下降。

#### 2. `Platforms / QualityGrades / States` 是下一个最自然的小切口

当前 shape home 仍然同时知道：

- `Platforms`
- `QualityGrades`
- `States`

这已经是很明确的 ownership hotspot，而且 direct consumers 很清楚。

#### 3. 这个切片仍然足够 bounded

下一步可以只围绕：

- impact slice clone shaping
- current `cloneAssetGenerationActionImpact(...)` consumers
- current consumer-visible behavior stability

来做，不需要马上扩大到 broader action target clone redesign 或 generic clone framework。

### Explicitly Not Recommended Next

#### 1. 不建议回头重开 shared retry request clone layering

那条线目前已经没有同等级的 mixed final owner 还留着。

#### 2. 不建议直接重开 broader action target aggregate routing

`task_generation_action_target_clone_shape.go` 当前已经是清晰的 aggregate router。

#### 3. 不建议直接重开 action execute orchestration

那会显著扩大写面，超出当前最自然的小切片。

### Proposed Next Phase Name

建议下一阶段命名为：

`Phase 57: ListingKit action target impact slice clone ownership`
