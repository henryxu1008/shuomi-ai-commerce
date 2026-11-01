> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 72 Checkpoint

### Status

`Phase 72` continued the new SHEIN freshness framework line by shrinking:

- [evaluateSheinSaleAttributeFreshnessResolution(...)](/D:/code/task-processor/internal/listingkit/shein_submit_sale_attribute_freshness_evaluation.go:30)

This round extracted the `invalid-sale-attribute collection + repair retry routing` responsibility into its own local home.

### What Changed

- [shein_submit_sale_attribute_freshness_evaluation.go](/D:/code/task-processor/internal/listingkit/shein_submit_sale_attribute_freshness_evaluation.go:30)
  - `evaluateSheinSaleAttributeFreshnessResolution(...)` now keeps:
    - base issue detection
    - invalid-state dispatch
    - outward message shaping
- [shein_submit_sale_attribute_freshness_resolution_repair.go](/D:/code/task-processor/internal/listingkit/shein_submit_sale_attribute_freshness_resolution_repair.go:1)
  - new local home for:
    - invalid sale-attribute collection
    - repair retry routing
    - post-repair re-evaluation
- [phase71_shein_sale_attribute_freshness_boundary_test.go](/D:/code/task-processor/internal/listingkit/phase71_shein_sale_attribute_freshness_boundary_test.go:1)
  - boundary coverage updated to lock the new split

### Resulting Shape

Current layering for this hotspot is now:

- aggregate owner:
  - [shein_submit_freshness.go](/D:/code/task-processor/internal/listingkit/shein_submit_freshness.go:306)
- local evaluation home:
  - [shein_submit_sale_attribute_freshness_evaluation.go](/D:/code/task-processor/internal/listingkit/shein_submit_sale_attribute_freshness_evaluation.go:30)
- narrower invalid-state / repair home:
  - [shein_submit_sale_attribute_freshness_resolution_repair.go](/D:/code/task-processor/internal/listingkit/shein_submit_sale_attribute_freshness_resolution_repair.go:1)

### Next Best Cut

The clearest remaining mixed-owner node is still:

- `evaluateSheinSaleAttributeFreshnessResolution(...)`

What remains mixed there is now narrower:

- base issue detection for primary/secondary template drift
- outward issue aggregation and message shaping

That makes the next best cut a `message-shape / issue-aggregation` ownership split.

### Verification Summary

Fresh verification passed:

```powershell
go test ./internal/listingkit -run "TestEvaluateSheinSaleAttributeFreshness.*|TestSheinSaleAttributeFreshnessBoundary" -count=1
go test ./internal/listingkit/httpapi ./internal/listingkit/temporal -count=1
```
