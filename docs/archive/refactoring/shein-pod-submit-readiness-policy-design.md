> **历史重构记录，非当前派工或操作依据。** EvaluatePODSubmitReadiness等旧策略族已在#961准确退休矩阵列明；旧设计仅归档，不创造当前函数或兼容owner。原有状态、数字、结果和精确替代范围保留；归位不改变产品或运行语义。当前依据从[文档首页](../../README.md)进入，历史材料见[归档索引](../README.md)。

# SHEIN POD Submit Readiness Policy

## Goal

Move the pure SHEIN decision that turns SDS POD execution state into a submit-readiness outcome from root ListingKit to `internal/marketplace/shein/publishing`.

## Ownership

`internal/product/sourcing/sdspod` remains the source of execution normalization, blocking semantics, and base readiness wording.

`internal/marketplace/shein/publishing` will own `EvaluatePODSubmitReadiness(action, execution)`. It combines the normalized SDS execution facts with the SHEIN submit-action policy and returns a neutral decision:

- whether a POD check applies;
- whether it is ready;
- whether a non-ready result is only a warning;
- the action-aware message.

The policy reuses `SubmitActionAllowsReadinessBlockers`; `save_draft` remains non-blocking and retains its current fallback wording whenever POD has not succeeded.

`internal/listingkit` retains only the adaptation from `PodExecutionSummary` to `sdspod.Execution` and construction of the workspace `ReadinessCheckSpec` DTO.

## Compatibility

The existing outcomes remain unchanged:

- disabled POD does not add a check;
- required, unfinished POD blocks `publish`;
- optional degraded or bypassed POD is a warning for `publish`;
- non-succeeded POD is a warning for `save_draft`.

## Verification

- publishing package tests cover all four outcomes above;
- ListingKit readiness tests continue to cover adapted workspace blocker/warning results;
- a boundary guard rejects local POD readiness action/status policy in ListingKit.
