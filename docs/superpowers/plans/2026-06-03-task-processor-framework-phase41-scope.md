> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 41 Scope Recommendation

### Recommendation

下一步更值得做的是 `ListingKit navigation dispatch-plan step clone ownership`。

也就是继续处理：

- [generation_navigation_dispatch_plan_clone_shape.go](/D:/code/task-processor/internal/listingkit/generation_navigation_dispatch_plan_clone_shape.go:1)

里 still inlined 的 step-level clone shaping。

### Why This Before Other Options

#### 1. Dispatch-plan aggregate clone 已经收干净了

现在：

- shared helper home 已独立
- action-target / review-navigation / descriptor aggregate clone 都已显式委托 nested clone shaping
- dispatch-plan aggregate clone 也已显式委托 nested clone shaping

继续在 aggregate owner 这一层追求更细对称性，收益已经明显下降。

#### 2. Dispatch-plan step clone 是下一个明显的 aggregate hotspot

当前本地 shape seam 仍然同时知道：

- step slice clone
- step-level query clone

这已经是很明确的 residual ownership hotspot。

#### 3. 这个切片仍然足够 bounded

下一步可以只围绕：

- dispatch-plan step clone
- step query clone delegation
- current consumer-visible behavior stability

来做，不需要马上扩大到 dispatch-plan execution flow 本身。

### Explicitly Not Recommended Next

#### 1. 不建议继续深挖 dispatch-plan aggregate clone home

那个 ownership 问题已经解决了。

#### 2. 不建议直接重开 broader dispatch-plan execution flow

那会显著扩大写面，超出当前最自然的小切片。

### Proposed Next Phase Name

建议下一阶段命名为：

`Phase 41: ListingKit navigation dispatch-plan step clone ownership`
