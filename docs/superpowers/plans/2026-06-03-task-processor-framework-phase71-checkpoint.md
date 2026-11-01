> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 71 Checkpoint

### Status

`Phase 71` opened a new ListingKit framework line after the `Phase 70` closure review.

This round confirmed that `shein_submit_freshness.go` is a real new hotspot, and landed the first ownership slice around `evaluateSheinSaleAttributeFreshnessWithCustomValidation(...)`.

### Why This Became The Next Line

The discovery pass found that:

- [shein_submit_freshness.go](/D:/code/task-processor/internal/listingkit/shein_submit_freshness.go:1) is still a large mixed-owner production file
- the sale-attribute freshness path had direct behavior coverage already in place
- the aggregate owner was still mixing:
  - guard logic
  - template-context preparation
  - invalid-value collection
  - repair routing
  - outward issue/message shaping

That made it a strong, testable next framework target.

### What Changed

- [shein_submit_freshness.go](/D:/code/task-processor/internal/listingkit/shein_submit_freshness.go:306)
  - `evaluateSheinSaleAttributeFreshnessWithCustomValidation(...)` now acts as `guard + context dispatch + resolution dispatch`
- [shein_submit_sale_attribute_freshness_evaluation.go](/D:/code/task-processor/internal/listingkit/shein_submit_sale_attribute_freshness_evaluation.go:1)
  - new local home for:
    - `buildSheinSaleAttributeFreshnessTemplateContext(...)`
    - `evaluateSheinSaleAttributeFreshnessResolution(...)`
- [phase71_shein_sale_attribute_freshness_boundary_test.go](/D:/code/task-processor/internal/listingkit/phase71_shein_sale_attribute_freshness_boundary_test.go:1)
  - locks the new aggregate vs resolution ownership split

### Resulting Shape

Current shape for this new line is:

- aggregate owner:
  - `evaluateSheinSaleAttributeFreshnessWithCustomValidation(...)`
- local evaluation home:
  - `buildSheinSaleAttributeFreshnessTemplateContext(...)`
  - `evaluateSheinSaleAttributeFreshnessResolution(...)`

This is the first reduction only; the resolution home still mixes:

- base issue detection
- invalid-sale-attribute collection
- repair retry routing
- issue/message shaping

### Next Best Cut

The strongest next slice is to keep narrowing `evaluateSheinSaleAttributeFreshnessResolution(...)`, especially around:

- invalid-sale-attribute collection and repair retry routing
- issue aggregation / outward message shaping

### Verification Summary

Fresh verification passed:

```powershell
go test ./internal/listingkit -run "TestEvaluateSheinSaleAttributeFreshness.*|TestSheinSaleAttributeFreshnessBoundary" -count=1
go test ./internal/listingkit/temporal -count=1
```
