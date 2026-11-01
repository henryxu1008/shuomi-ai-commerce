> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 38 Scope Recommendation

### Recommendation

下一步更值得做的是 `ListingKit review navigation target clone aggregate ownership`。

也就是继续处理：

- [service_generation_navigation_dispatch_helpers.go](/D:/code-task-processor/internal/listingkit/service_generation_navigation_dispatch_helpers.go:1) 里的 `cloneGenerationReviewNavigationTarget(...)`

而不是继续回头深挖已经稳定下来的 action-target aggregate clone。

### Why This Before Other Options

#### 1. Action-target aggregate clone 已经收干净了

现在：

- shared helper home 已独立
- action-target aggregate clone 已显式委托 nested clone shaping

继续在 action-target 这层追求更细对称性，收益已经明显下降。

#### 2. Review-navigation target clone 是下一个明显的 aggregate hotspot

当前 `cloneGenerationReviewNavigationTarget(...)` 仍然同时知道：

- conditional clone
- descriptor clone
- queue/session/preview query clone
- nested action target clone

这已经是很明确的 aggregate clone owner。

#### 3. 这个切片仍然足够 bounded

下一步可以只围绕：

- review navigation target aggregate clone
- nested clone delegation
- current consumer-visible behavior stability

来做，不需要马上扩大到 navigation dispatch execution flow 本身。

### Explicitly Not Recommended Next

#### 1. 不建议继续深挖 shared clone helper home

那个 ownership 问题已经解决了。

#### 2. 不建议直接重开 broader review navigation dispatch 流

那会显著扩大写面，超出当前最自然的小切片。

### Proposed Next Phase Name

建议下一阶段命名为：

`Phase 38: ListingKit review navigation target clone aggregate ownership`
