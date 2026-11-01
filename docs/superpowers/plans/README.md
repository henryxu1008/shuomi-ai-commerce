# 实施计划与历史记录

当前工作从[项目文档](../../README.md)、[架构索引](../../architecture/README.md)和执行 Issue 进入。本目录混合了可引用的领域方案与旧阶段资料；文件日期、未勾选步骤和目录存在本身不授予实施权限。

- [合同与专项历史保留入口](../README.md)：原 52 项核对中保留的 18 份，以及继续整理保留的 8 份。
- [Commerce Tool 合同与注册表计划](2026-08-31-commerce-tool-contract-registry.md)：相关领域资料示例，具体适用范围由当前批准合同和执行 Issue 决定；未归入下面的旧 Framework 阶段档案。
- [Product / Figma Authority](../../product/final-ui-ia-authority.md)与[全新系统基线](../../product/greenfield-no-legacy-migration.md)：新工作不恢复旧 Task-first、兼容 facade 或历史数据迁移。

<a id="framework-history"></a>

## 旧 Framework 阶段资料

本索引保留 238 份原阶段资料：94 份 checkpoint、88 份 scope、56 份方案/设计。它们记录 2026-05-29 至 2026-06-03 当时的结构、建议和验证；按原文范围追溯，不能作为实时工作队列，也不宣称其中仍有效的 kernel/module 或领域合同已退休。

原文、勾选、机器绝对路径和历史结果完整保留。`Expected: PASS` 不代表已执行，历史验证不代表当前 main 或产品验收。原 `/D:/code/...` 等链接只表示当时的文件位置；当前源码入口使用上方仓库导航，不把旧链接重写成可能错误的当前路径。

18 份无引用、无当前生产目标、无执行结果的旧 ListingKit 方案已删除；其余 scope/checkpoint 保留对应阶段的历史上下文。逐份删除依据见 [#1050 / PR #1051](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1051)，原文可从本轮起点 `7dd3a3050828b38e95ae79fe316501f0b19ecdcf` 的 Git 历史查阅。

### Phase 1–20

| 阶段 | 方案 / 设计 | 原范围建议 | 原检查点 |
| --- | --- | --- | --- |
| 1 | [方案](2026-05-29-task-processor-framework-phase1.md) | — | [检查点](2026-05-29-task-processor-framework-phase1-checkpoint.md) |
| 2A | [方案](2026-05-29-task-processor-framework-phase2a.md) | — | — |
| 2B | [方案](2026-05-29-task-processor-framework-phase2b-worker-runtime.md) | — | — |
| 2C | [方案](2026-05-29-task-processor-framework-phase2c-temporal-runtime.md) | — | [检查点](2026-05-29-task-processor-framework-phase2c-checkpoint.md) |
| 3 | — | [范围](2026-05-29-task-processor-framework-phase3-scope.md) | — |
| 3A | — | — | [检查点](2026-05-29-task-processor-framework-phase3a-checkpoint.md) |
| 3B | — | — | [检查点](2026-05-29-task-processor-framework-phase3b-checkpoint.md) |
| 4A | — | — | [检查点](2026-05-29-task-processor-framework-phase4a-checkpoint.md) |
| 4B | — | — | [检查点](2026-05-29-task-processor-framework-phase4b-checkpoint.md) |
| 5A | — | — | [检查点](2026-05-29-task-processor-framework-phase5a-checkpoint.md) |
| 5B | — | — | [检查点](2026-05-29-task-processor-framework-phase5b-checkpoint.md) |
| 6 | — | [范围](2026-05-29-task-processor-framework-phase6-scope.md) | — |
| 6A | — | — | [检查点](2026-05-30-task-processor-framework-phase6a-checkpoint.md) |
| 6B | — | [范围](2026-05-30-task-processor-framework-phase6b-scope.md) | [检查点](2026-05-30-task-processor-framework-phase6b-checkpoint.md) |
| 7 | — | [范围](2026-05-30-task-processor-framework-phase7-scope.md) | — |
| 7A | — | — | [检查点](2026-05-30-task-processor-framework-phase7a-checkpoint.md) |
| 7B | — | [范围](2026-05-30-task-processor-framework-phase7b-scope.md) | [检查点](2026-05-30-task-processor-framework-phase7b-checkpoint.md) |
| 8 | — | [范围](2026-05-30-task-processor-framework-phase8-scope.md) | — |
| 8A | — | — | [检查点](2026-05-30-task-processor-framework-phase8a-checkpoint.md) |
| 8B | — | [范围](2026-05-30-task-processor-framework-phase8b-scope.md) | [检查点](2026-05-30-task-processor-framework-phase8b-checkpoint.md) |
| 9 | — | [范围](2026-05-30-task-processor-framework-phase9-scope.md) | — |
| 9A | — | — | [检查点](2026-05-30-task-processor-framework-phase9a-checkpoint.md) |
| 10 | — | [范围](2026-05-30-task-processor-framework-phase10-scope.md) | — |
| 10A | — | — | [检查点](2026-05-30-task-processor-framework-phase10a-checkpoint.md) |
| 10B | — | [范围](2026-05-30-task-processor-framework-phase10b-scope.md) | [检查点](2026-05-30-task-processor-framework-phase10b-checkpoint.md) |
| 11 | — | [范围](2026-05-30-task-processor-framework-phase11-scope.md) | [检查点](2026-05-30-task-processor-framework-phase11-checkpoint.md) |
| 12 | — | [范围](2026-05-30-task-processor-framework-phase12-scope.md) | [检查点](2026-05-30-task-processor-framework-phase12-checkpoint.md) |
| 13 | — | [范围](2026-05-30-task-processor-framework-phase13-scope.md) | [检查点](2026-05-30-task-processor-framework-phase13-checkpoint.md) |
| 14 | — | [范围](2026-05-30-task-processor-framework-phase14-scope.md) | [检查点](2026-05-30-task-processor-framework-phase14-checkpoint.md) |
| 15 | — | [范围](2026-05-30-task-processor-framework-phase15-scope.md) | [检查点](2026-05-30-task-processor-framework-phase15-checkpoint.md) |
| 16 | — | [范围](2026-05-30-task-processor-framework-phase16-scope.md) | [检查点](2026-05-31-task-processor-framework-phase16-checkpoint.md) |
| 17 | — | [范围](2026-05-31-task-processor-framework-phase17-scope.md) | [检查点](2026-05-31-task-processor-framework-phase17-checkpoint.md) |
| 18 | — | [范围](2026-05-31-task-processor-framework-phase18-scope.md) | [检查点](2026-05-31-task-processor-framework-phase18-checkpoint.md) |
| 19 | [方案](2026-06-01-task-processor-framework-phase19-layer-temporal-action-branching.md) | [范围](2026-05-31-task-processor-framework-phase19-scope.md) | [检查点](2026-06-01-task-processor-framework-phase19-checkpoint.md) |
| 20 | [方案](2026-06-01-task-processor-framework-phase20-layer-temporal-request-shape-parsing.md) | [范围](2026-06-01-task-processor-framework-phase20-scope.md) | [检查点](2026-06-01-task-processor-framework-phase20-checkpoint.md) |

### Phase 21–40

| 阶段 | 方案 / 设计 | 原范围建议 | 原检查点 |
| --- | --- | --- | --- |
| 21 | [方案](2026-06-01-task-processor-framework-phase21-action-target-resolution-ownership.md) | [范围](2026-06-01-task-processor-framework-phase21-scope.md) | [检查点](2026-06-02-task-processor-framework-phase21-checkpoint.md) |
| 22 | [方案](2026-06-02-task-processor-framework-phase22-clone-helper-ownership.md) | [范围](2026-06-02-task-processor-framework-phase22-scope.md) | [检查点](2026-06-02-task-processor-framework-phase22-checkpoint.md) |
| 23 | [方案](2026-06-02-task-processor-framework-phase23-navigation-action-target-clone-shaping.md) | [范围](2026-06-02-task-processor-framework-phase23-scope.md) | [检查点](2026-06-03-task-processor-framework-phase23-checkpoint.md) |
| 24 | [方案](2026-06-03-task-processor-framework-phase24-review-navigation-queue-clone-shaping.md) | [范围](2026-06-03-task-processor-framework-phase24-scope.md) | [检查点](2026-06-03-task-processor-framework-phase24-checkpoint.md) |
| 25 | [方案](2026-06-03-task-processor-framework-phase25-action-execute-request-handoff.md) | [范围](2026-06-03-task-processor-framework-phase25-scope.md) | [检查点](2026-06-03-task-processor-framework-phase25-checkpoint.md) |
| 26 | [方案](2026-06-03-task-processor-framework-phase26-action-execute-handoff-result-adaptation.md) | [范围](2026-06-03-task-processor-framework-phase26-scope.md) | [检查点](2026-06-03-task-processor-framework-phase26-checkpoint.md) |
| 27 | [方案](2026-06-03-task-processor-framework-phase27-action-execute-handoff-branch-invocation.md) | [范围](2026-06-03-task-processor-framework-phase27-scope.md) | [检查点](2026-06-03-task-processor-framework-phase27-checkpoint.md) |
| 28 | [方案](2026-06-03-task-processor-framework-phase28-action-execute-handoff-branch-result-routing.md) | [范围](2026-06-03-task-processor-framework-phase28-scope.md) | [检查点](2026-06-03-task-processor-framework-phase28-checkpoint.md) |
| 29 | [方案](2026-06-03-task-processor-framework-phase29-action-execute-handoff-interaction-mode-routing.md) | [范围](2026-06-03-task-processor-framework-phase29-scope.md) | [检查点](2026-06-03-task-processor-framework-phase29-checkpoint.md) |
| 30 | [方案](2026-06-03-task-processor-framework-phase30-action-execute-handoff-result-shape-ownership.md) | [范围](2026-06-03-task-processor-framework-phase30-scope.md) | [检查点](2026-06-03-task-processor-framework-phase30-checkpoint.md) |
| 31 | [方案](2026-06-03-task-processor-framework-phase31-action-execute-handoff-mode-routing-pairing.md) | [范围](2026-06-03-task-processor-framework-phase31-scope.md) | [检查点](2026-06-03-task-processor-framework-phase31-checkpoint.md) |
| 32 | [方案](2026-06-03-task-processor-framework-phase32-action-execute-handoff-result-normalization.md) | [范围](2026-06-03-task-processor-framework-phase32-scope.md) | [检查点](2026-06-03-task-processor-framework-phase32-checkpoint.md) |
| 33 | [方案](2026-06-03-task-processor-framework-phase33-action-execute-handoff-mode-pairing-normalization.md) | [范围](2026-06-03-task-processor-framework-phase33-scope.md) | [检查点](2026-06-03-task-processor-framework-phase33-checkpoint.md) |
| 34 | [方案](2026-06-03-task-processor-framework-phase34-action-execute-handoff-branch-result-routing.md) | [范围](2026-06-03-task-processor-framework-phase34-scope.md) | [检查点](2026-06-03-task-processor-framework-phase34-checkpoint.md) |
| 35 | [方案](2026-06-03-task-processor-framework-phase35-action-execute-handoff-branch-request-shaping.md) | [范围](2026-06-03-task-processor-framework-phase35-scope.md) | [检查点](2026-06-03-task-processor-framework-phase35-checkpoint.md) |
| 36 | [方案](2026-06-03-task-processor-framework-phase36-shared-clone-helper-ownership.md) | [范围](2026-06-03-task-processor-framework-phase36-scope.md) | [检查点](2026-06-03-task-processor-framework-phase36-checkpoint.md) |
| 37 | [方案](2026-06-03-task-processor-framework-phase37-action-target-clone-aggregate-ownership.md) | [范围](2026-06-03-task-processor-framework-phase37-scope.md) | [检查点](2026-06-03-task-processor-framework-phase37-checkpoint.md) |
| 38 | [方案](2026-06-03-task-processor-framework-phase38-review-navigation-target-clone-aggregate-ownership.md) | [范围](2026-06-03-task-processor-framework-phase38-scope.md) | [检查点](2026-06-03-task-processor-framework-phase38-checkpoint.md) |
| 39 | [方案](2026-06-03-task-processor-framework-phase39-navigation-descriptor-clone-aggregate-ownership.md) | [范围](2026-06-03-task-processor-framework-phase39-scope.md) | [检查点](2026-06-03-task-processor-framework-phase39-checkpoint.md) |
| 40 | [方案](2026-06-03-task-processor-framework-phase40-navigation-dispatch-plan-clone-aggregate-ownership.md) | [范围](2026-06-03-task-processor-framework-phase40-scope.md) | [检查点](2026-06-03-task-processor-framework-phase40-checkpoint.md) |

### Phase 41–60

| 阶段 | 方案 / 设计 | 原范围建议 | 原检查点 |
| --- | --- | --- | --- |
| 41 | [方案](2026-06-03-task-processor-framework-phase41-navigation-dispatch-plan-step-clone-ownership.md) | [范围](2026-06-03-task-processor-framework-phase41-scope.md) | [检查点](2026-06-03-task-processor-framework-phase41-checkpoint.md) |
| 42 | [方案](2026-06-03-task-processor-framework-phase42-navigation-descriptor-followup-read-clone-ownership.md) | [范围](2026-06-03-task-processor-framework-phase42-scope.md) | [检查点](2026-06-03-task-processor-framework-phase42-checkpoint.md) |
| 43 | [方案](2026-06-03-task-processor-framework-phase43-navigation-descriptor-residual-shape-ownership.md) | [范围](2026-06-03-task-processor-framework-phase43-scope.md) | [检查点](2026-06-03-task-processor-framework-phase43-checkpoint.md) |
| 44 | [方案](2026-06-03-task-processor-framework-phase44-navigation-descriptor-residual-pairing-ownership.md) | [范围](2026-06-03-task-processor-framework-phase44-scope.md) | [检查点](2026-06-03-task-processor-framework-phase44-checkpoint.md) |
| 45 | [方案](2026-06-03-task-processor-framework-phase45-navigation-descriptor-dispatch-plan-delegation-ownership.md) | [范围](2026-06-03-task-processor-framework-phase45-scope.md) | [检查点](2026-06-03-task-processor-framework-phase45-checkpoint.md) |
| 46 | [方案](2026-06-03-task-processor-framework-phase46-navigation-descriptor-clone-shape-routing-ownership.md) | [范围](2026-06-03-task-processor-framework-phase46-scope.md) | [检查点](2026-06-03-task-processor-framework-phase46-checkpoint.md) |
| 47 | [方案](2026-06-03-task-processor-framework-phase47-navigation-descriptor-clone-shape-pairing-ownership.md) | [范围](2026-06-03-task-processor-framework-phase47-scope.md) | [检查点](2026-06-03-task-processor-framework-phase47-checkpoint.md) |
| 48 | [方案](2026-06-03-task-processor-framework-phase48-navigation-descriptor-followup-read-routing-pairing-ownership.md) | [范围](2026-06-03-task-processor-framework-phase48-scope.md) | [检查点](2026-06-03-task-processor-framework-phase48-checkpoint.md) |
| 49 | [方案](2026-06-03-task-processor-framework-phase49-navigation-descriptor-followup-read-slice-clone-ownership.md) | [范围](2026-06-03-task-processor-framework-phase49-scope.md) | [检查点](2026-06-03-task-processor-framework-phase49-checkpoint.md) |
| 50 | [方案](2026-06-03-task-processor-framework-phase50-navigation-followup-read-item-clone-aggregate-ownership.md) | [范围](2026-06-03-task-processor-framework-phase50-scope.md) | [检查点](2026-06-03-task-processor-framework-phase50-checkpoint.md) |
| 51 | [方案](2026-06-03-task-processor-framework-phase51-shared-queue-query-clone-aggregate-ownership.md) | [范围](2026-06-03-task-processor-framework-phase51-scope.md) | [检查点](2026-06-03-task-processor-framework-phase51-checkpoint.md) |
| 52 | [方案](2026-06-03-task-processor-framework-phase52-shared-retry-request-clone-aggregate-ownership.md) | [范围](2026-06-03-task-processor-framework-phase52-scope.md) | [检查点](2026-06-03-task-processor-framework-phase52-checkpoint.md) |
| 53 | [方案](2026-06-03-task-processor-framework-phase53-shared-retry-request-slice-clone-ownership.md) | [范围](2026-06-03-task-processor-framework-phase53-scope.md) | [检查点](2026-06-03-task-processor-framework-phase53-checkpoint.md) |
| 54 | [方案](2026-06-03-task-processor-framework-phase54-shared-retry-request-taskid-slot-pairing-ownership.md) | [范围](2026-06-03-task-processor-framework-phase54-scope.md) | [检查点](2026-06-03-task-processor-framework-phase54-checkpoint.md) |
| 55 | [方案](2026-06-03-task-processor-framework-phase55-shared-retry-request-taskid-slot-final-ownership.md) | [范围](2026-06-03-task-processor-framework-phase55-scope.md) | [检查点](2026-06-03-task-processor-framework-phase55-checkpoint.md) |
| 56 | [方案](2026-06-03-task-processor-framework-phase56-action-target-impact-clone-aggregate-ownership.md) | [范围](2026-06-03-task-processor-framework-phase56-scope.md) | [检查点](2026-06-03-task-processor-framework-phase56-checkpoint.md) |
| 57 | [方案](2026-06-03-task-processor-framework-phase57-action-target-impact-slice-clone-ownership.md) | [范围](2026-06-03-task-processor-framework-phase57-scope.md) | [检查点](2026-06-03-task-processor-framework-phase57-checkpoint.md) |
| 58 | [方案](2026-06-03-task-processor-framework-phase58-action-target-impact-final-slice-ownership.md) | [范围](2026-06-03-task-processor-framework-phase58-scope.md) | [检查点](2026-06-03-task-processor-framework-phase58-checkpoint.md) |
| 59 | [方案](2026-06-03-task-processor-framework-phase59-action-target-filters-clone-aggregate-ownership.md) | [范围](2026-06-03-task-processor-framework-phase59-scope.md) | [检查点](2026-06-03-task-processor-framework-phase59-checkpoint.md) |
| 60 | [方案](2026-06-03-task-processor-framework-phase60-action-target-filters-platform-slice-ownership.md) | [范围](2026-06-03-task-processor-framework-phase60-scope.md) | [检查点](2026-06-03-task-processor-framework-phase60-checkpoint.md) |

### Phase 61–88

| 阶段 | 方案 / 设计 | 原范围建议 | 原检查点 |
| --- | --- | --- | --- |
| 61 | [方案](2026-06-03-task-processor-framework-phase61-action-target-filter-mutation-ownership.md) | [范围](2026-06-03-task-processor-framework-phase61-scope.md) | [检查点](2026-06-03-task-processor-framework-phase61-checkpoint.md) |
| 62 | [方案](2026-06-03-task-processor-framework-phase62-action-target-preview-capability-filter-mutation.md) | [范围](2026-06-03-task-processor-framework-phase62-scope.md) | [检查点](2026-06-03-task-processor-framework-phase62-checkpoint.md) |
| 63 | [方案](2026-06-03-task-processor-framework-phase63-regular-action-key-filter-mutation-ownership.md) | [范围](2026-06-03-task-processor-framework-phase63-scope.md) | [检查点](2026-06-03-task-processor-framework-phase63-checkpoint.md) |
| 64 | [方案](2026-06-03-task-processor-framework-phase64-retry-oriented-action-key-filter-mutation-ownership.md) | [范围](2026-06-03-task-processor-framework-phase64-scope.md) | [检查点](2026-06-03-task-processor-framework-phase64-checkpoint.md) |
| 65 | [方案](2026-06-03-task-processor-framework-phase65-failed-vs-provisional-retry-action-key-mutation-ownership.md) | [范围](2026-06-03-task-processor-framework-phase65-scope.md) | [检查点](2026-06-03-task-processor-framework-phase65-checkpoint.md) |
| 66 | [方案](2026-06-03-task-processor-framework-phase66-provisional-vs-section-retry-action-key-mutation-ownership.md) | [范围](2026-06-03-task-processor-framework-phase66-scope.md) | [检查点](2026-06-03-task-processor-framework-phase66-checkpoint.md) |
| 67 | [方案](2026-06-03-task-processor-framework-phase67-non-retry-regular-action-key-mutation-ownership.md) | [范围](2026-06-03-task-processor-framework-phase67-scope.md) | [检查点](2026-06-03-task-processor-framework-phase67-checkpoint.md) |
| 68 | [方案](2026-06-03-task-processor-framework-phase68-missing-slot-action-key-mutation-ownership.md) | [范围](2026-06-03-task-processor-framework-phase68-scope.md) | [检查点](2026-06-03-task-processor-framework-phase68-checkpoint.md) |
| 69 | [方案](2026-06-03-task-processor-framework-phase69-completion-audit.md) | [范围](2026-06-03-task-processor-framework-phase69-scope.md) | [检查点](2026-06-03-task-processor-framework-phase69-checkpoint.md) |
| 70 | [方案](2026-06-03-task-processor-framework-phase70-closure-summary-and-integration-readiness.md) | [范围](2026-06-03-task-processor-framework-phase70-scope.md) | [检查点](2026-06-03-task-processor-framework-phase70-checkpoint.md) |
| 71 | — | [范围](2026-06-03-task-processor-framework-phase71-scope.md) | [检查点](2026-06-03-task-processor-framework-phase71-checkpoint.md) |
| 72 | — | [范围](2026-06-03-task-processor-framework-phase72-scope.md) | [检查点](2026-06-03-task-processor-framework-phase72-checkpoint.md) |
| 73 | — | [范围](2026-06-03-task-processor-framework-phase73-scope.md) | [检查点](2026-06-03-task-processor-framework-phase73-checkpoint.md) |
| 74 | — | [范围](2026-06-03-task-processor-framework-phase74-scope.md) | [检查点](2026-06-03-task-processor-framework-phase74-checkpoint.md) |
| 75 | — | [范围](2026-06-03-task-processor-framework-phase75-scope.md) | [检查点](2026-06-03-task-processor-framework-phase75-checkpoint.md) |
| 76 | — | [范围](2026-06-03-task-processor-framework-phase76-scope.md) | [检查点](2026-06-03-task-processor-framework-phase76-checkpoint.md) |
| 77 | — | [范围](2026-06-03-task-processor-framework-phase77-scope.md) | [检查点](2026-06-03-task-processor-framework-phase77-checkpoint.md) |
| 78 | — | [范围](2026-06-03-task-processor-framework-phase78-scope.md) | [检查点](2026-06-03-task-processor-framework-phase78-checkpoint.md) |
| 79 | — | [范围](2026-06-03-task-processor-framework-phase79-scope.md) | [检查点](2026-06-03-task-processor-framework-phase79-checkpoint.md) |
| 80 | — | [范围](2026-06-03-task-processor-framework-phase80-scope.md) | [检查点](2026-06-03-task-processor-framework-phase80-checkpoint.md) |
| 81 | — | [范围](2026-06-03-task-processor-framework-phase81-scope.md) | [检查点](2026-06-03-task-processor-framework-phase81-checkpoint.md) |
| 82 | — | [范围](2026-06-03-task-processor-framework-phase82-scope.md) | [检查点](2026-06-03-task-processor-framework-phase82-checkpoint.md) |
| 83 | — | [范围](2026-06-03-task-processor-framework-phase83-scope.md) | [检查点](2026-06-03-task-processor-framework-phase83-checkpoint.md) |
| 84 | — | [范围](2026-06-03-task-processor-framework-phase84-scope.md) | [检查点](2026-06-03-task-processor-framework-phase84-checkpoint.md) |
| 85 | — | [范围](2026-06-03-task-processor-framework-phase85-scope.md) | [检查点](2026-06-03-task-processor-framework-phase85-checkpoint.md) |
| 86 | — | [范围](2026-06-03-task-processor-framework-phase86-scope.md) | [检查点](2026-06-03-task-processor-framework-phase86-checkpoint.md) |
| 87 | — | [范围](2026-06-03-task-processor-framework-phase87-scope.md) | [检查点](2026-06-03-task-processor-framework-phase87-checkpoint.md) |
| 88 | — | [范围](2026-06-03-task-processor-framework-phase88-scope.md) | — |
