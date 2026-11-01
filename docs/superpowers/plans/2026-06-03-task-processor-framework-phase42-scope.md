> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 42 Scope Recommendation

### Recommendation

下一步更值得做的是 `ListingKit navigation descriptor follow-up read clone ownership`。

也就是继续处理：

- [generation_navigation_descriptor_clone_shape.go](/D:/code/task-processor/internal/listingkit/generation_navigation_descriptor_clone_shape.go:1)

里 still inlined 的 `FollowUpReads` clone shaping。

### Why This Before Other Options

#### 1. Dispatch-plan step clone 已经收干净了

现在：

- shared helper home 已独立
- action-target / review-navigation / descriptor / dispatch-plan aggregate clone 都已显式委托 nested clone shaping
- dispatch-plan step clone 也已显式收成更窄的本地 seam

继续在 dispatch-plan 这层追求更细对称性，收益已经明显下降。

#### 2. Descriptor follow-up read clone 是下一个明显的 aggregate hotspot

当前本地 descriptor shape seam 仍然同时知道：

- follow-up read slice clone
- nested query clone

这已经是很明确的 residual ownership hotspot。

#### 3. 这个切片仍然足够 bounded

下一步可以只围绕：

- descriptor follow-up read clone
- shared query clone delegation
- current consumer-visible behavior stability

来做，不需要马上扩大到 descriptor builder 或 dispatch flow 本身。

### Explicitly Not Recommended Next

#### 1. 不建议继续深挖 dispatch-plan step clone home

那个 ownership 问题已经解决了。

#### 2. 不建议直接重开 broader descriptor builder / navigation dispatch flow

那会显著扩大写面，超出当前最自然的小切片。

### Proposed Next Phase Name

建议下一阶段命名为：

`Phase 42: ListingKit navigation descriptor follow-up read clone ownership`
