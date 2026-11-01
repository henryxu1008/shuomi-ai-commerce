> **领域合同归位。** 候选canary先于生产promotion的既有安全顺序；不授权发布。本次仅调整文档位置与引用，不扩大原有准入或改变业务/安全语义；页首已有的精确退休说明和当前产品决定继续优先。当前产品与页面见[产品权威](../product/final-ui-ia-authority.md)，旧阶段材料见[历史归档](../archive/README.md)。

# Image Agent Candidate Release Gate Design

## Goal

Prevent a candidate image-agent Temporal worker from polling or mutating production workflow histories before its compatibility canary succeeds.

## Problem

The deployment workflow currently applies and restarts the production `image-agent-manual-v3` worker before running its compatibility canary. A candidate can poll the production queue during that interval; a failed subsequent canary cannot undo a replay, activity, or projection mutation already made by the candidate.

## Decision

The release workflow uses two explicit phases:

1. **Isolated candidate gate.** Build the candidate image and run the existing side-effect-free compatibility canary on a dedicated candidate queue/deployment. The candidate deployment has no production worker queue binding, no production workflow polling permission, and no route that can execute production histories. Wait for its health and canary success.
2. **Production promotion.** Only after the isolated gate succeeds, apply/restart the production `image-agent-manual-v3` deployment and wait for rollout completion. The existing post-promotion health verification remains, but is no longer the first compatibility check.

The candidate queue is configured as a distinct explicit value, never a naming convention inferred from the production queue. Cleanup removes the candidate deployment after a successful promotion and also on a failed gate without changing the production worker.

## Safety Constraints

- The candidate canary must use the same built image/digest as the later production rollout.
- Candidate workflow execution is side-effect-free and targets only the isolated queue.
- No `kubectl rollout restart` or production deployment patch occurs before the candidate gate reports success.
- A failed candidate gate fails the workflow before production mutation and retains logs/artifacts needed for diagnosis.
- This changes deployment sequencing only; it does not merge, deploy, or change application runtime behavior outside the release workflow.

## Tests and Acceptance

Workflow tests or script-level assertions verify that the candidate apply/health/canary steps precede every production deployment apply/restart command, that candidate and production queue values differ, and that a failed candidate path cannot reach production rollout. A dry-run/rendered workflow inspection confirms the candidate image digest equals the promoted production image digest.
