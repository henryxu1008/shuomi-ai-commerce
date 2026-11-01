> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 61 Scope Recommendation

### Recommendation

下一步更值得做的是 `ListingKit action target filter mutation ownership`。

也就是继续处理：

- [generation_overview.go](/D:/code/task-processor/internal/listingkit/generation_overview.go:290)

里的：

- `actionFiltersForKey(...)`

### Why This Before Other Options

#### 1. Action target clone-related helper 这条线已经基本收干净了

现在：

- impact clone 已收成 aggregate / shape / final local homes
- filters clone 已收成 aggregate / shape / final local homes

继续在 clone helper 这条线上追求更细对称性，收益已经明显下降。

#### 2. `actionFiltersForKey(...)` 是下一个最自然的小切口

当前它仍然同时知道：

- preview capability action specialization
- quality grade rewriting
- retryability toggles
- execution-quality resets
- action-key specific mutation branching

这已经是很明确的 mutation ownership hotspot，而且 direct consumers 很清楚。

#### 3. 这个切片仍然足够 bounded

下一步可以只围绕：

- `actionFiltersForKey(...)`
- `buildAssetGenerationActionTarget(...)`
- current consumer-visible filter mutation behavior stability

来做，不需要马上扩大到 broader action target orchestration 或 generic mutation framework。

### Explicitly Not Recommended Next

#### 1. 不建议继续深挖 action target filters clone final home

`Platforms` 现在已经是单一、直接的 final owner 了。

#### 2. 不建议回头重开 shared retry request 或 impact clone layering

这两条线目前也已经没有同等级的 mixed owner 还留着。

#### 3. 不建议直接重开 action execute orchestration

那会显著扩大写面，超出当前最自然的小切片。

### Proposed Next Phase Name

建议下一阶段命名为：

`Phase 61: ListingKit action target filter mutation ownership`
