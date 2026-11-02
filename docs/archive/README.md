# 历史设计、计划与验证记录

返回[文档首页](../README.md)。本目录保留已经被替代、取消，或因冻结检查/准确引用而需要追溯的原始材料。
当前产品依据见[产品索引](../product/README.md)，领域合同见[架构索引](../architecture/README.md)。

历史文中的 `IMPLEMENTATION_READY`、当前/下一步、旧命令及 PASS/UNKNOWN 只属于原记录范围，不能恢复已取消的实施、迁移、兼容、部署或数据操作。
尤其 PR #283 V7 / #284 V7 只是历史证据，不是当前 Greenfield 实现基线。文件位置与可读性不改变这条边界。

原始正文与结果保留；本次只调整阅读前缀与准确导航。既有冻结检查保留原长度/摘要，在逆映射本轮明列导航路径后验证原文，不放宽内容断言。

## 按材料类型查找

| 原始材料 | 阅读位置 |
| --- | --- |
| 旧产品 / Console / 资源设计 | [旧设计与取消边界](#旧设计与取消边界) |
| 冻结实施计划 | [实施计划与冻结历史](#实施计划与冻结历史) |
| 带日期和准确基线的验证 | [实际验证记录](#verification-history) |
| 重构阶段报告、原始测试输出 | [历史重构材料](#refactoring-history) |
| 旧架构 / PoC、开发与运行记录 | [历史架构、开发与运行材料](#architecture-history) |
| Kubernetes 发布与旧集成联调 | [历史部署与旧集成](#deployment-and-integration-history) |
| 保留在原领域旁的工程 / 产品证据 | [工程原始附件](../engineering/README.md#evidence)、[产品验证记录](../product/validation/runs/README.md) |

退休矩阵中的有效边界另见[架构退休导航](../architecture/retirement-index.md)，不与普通过期报告混放。

## 旧设计与取消边界

- [2026-08-30-shuomi-workbench-store-center-zitadel-multi-org-design.md](designs/2026-08-30-shuomi-workbench-store-center-zitadel-multi-org-design.md) — 首期工作台/企业/店铺设计快照；当前页面与owner由final-ui-ia、auth-and-tenancy和Store当前合同维护，不保留第二套实施入口。
- [2026-09-02-shuomi-console-phase1-hard-cut-design.md](designs/2026-09-02-shuomi-console-phase1-hard-cut-design.md) — 第一阶段高层历史与撤销V7准入证据；Greenfield检查继续验证其取消边界，当前产品Authority不由旧Console计划替代。
- [2026-09-03-shuomi-account-entry-zitadel-native-flow-design.md](designs/2026-09-03-shuomi-account-entry-zitadel-native-flow-design.md) — 原Login Fork/账号入口与V7引用只作历史；现有用户登录由官方Login V2产品决定替代，Greenfield取消旧V7实施权威。
- [2026-09-03-shuomi-resource-ledger-store-design.md](designs/2026-09-03-shuomi-resource-ledger-store-design.md) — 受冻结hash保护的旧资源/Store设计证据；当前资源由统一方案与已确认#1041退休边界维护。
- [2026-09-05-issue-30-sourcing-cutover.md](designs/2026-09-05-issue-30-sourcing-cutover.md) — 已明确SUPERSEDED的旧迁移/cutover证据；保留反恢复禁令与原历史。
- [2026-09-05-source-account-organization-cutover.md](designs/2026-09-05-source-account-organization-cutover.md) — 已明确SUPERSEDED的旧组织迁移证据；当前SA1/SA2与Greenfield不从这里恢复迁移。

## 实施计划与冻结历史

- [2026-06-20-listingkit-async-studio-image-generation.md](implementation-plans/2026-06-20-listingkit-async-studio-image-generation.md) — 普通旧实施步骤不再作为当前排期；本份因现有冻结检查/准确历史引用或原批准证据完整归档。
- [2026-08-06-ai-async-job-binding.md](implementation-plans/2026-08-06-ai-async-job-binding.md) — 普通旧实施步骤不再作为当前排期；本份因现有冻结检查/准确历史引用或原批准证据完整归档。
- [2026-08-06-ai-capability-phase-1.md](implementation-plans/2026-08-06-ai-capability-phase-1.md) — 普通旧实施步骤不再作为当前排期；本份因现有冻结检查/准确历史引用或原批准证据完整归档。
- [2026-08-21-provider-neutral-ai-contracts.md](implementation-plans/2026-08-21-provider-neutral-ai-contracts.md) — 普通旧实施步骤不再作为当前排期；本份因现有冻结检查/准确历史引用或原批准证据完整归档。
- [2026-08-25-zitadel-authruntime-extraction.md](implementation-plans/2026-08-25-zitadel-authruntime-extraction.md) — 普通旧实施步骤不再作为当前排期；本份因现有冻结检查/准确历史引用或原批准证据完整归档。
- [2026-08-26-image-agent-assisted-planning.md](implementation-plans/2026-08-26-image-agent-assisted-planning.md) — 普通旧实施步骤不再作为当前排期；本份因现有冻结检查/准确历史引用或原批准证据完整归档。
- [2026-08-26-image-agent-automatic-repair.md](implementation-plans/2026-08-26-image-agent-automatic-repair.md) — 普通旧实施步骤不再作为当前排期；本份因现有冻结检查/准确历史引用或原批准证据完整归档。
- [2026-08-26-image-agent-core-manual-workflow.md](implementation-plans/2026-08-26-image-agent-core-manual-workflow.md) — 普通旧实施步骤不再作为当前排期；本份因现有冻结检查/准确历史引用或原批准证据完整归档。
- [2026-08-29-image-agent-external-effect-finalization.md](implementation-plans/2026-08-29-image-agent-external-effect-finalization.md) — 普通旧实施步骤不再作为当前排期；本份因现有冻结检查/准确历史引用或原批准证据完整归档。
- [2026-08-30-internal-target-architecture-phase2.md](implementation-plans/2026-08-30-internal-target-architecture-phase2.md) — 普通旧实施步骤不再作为当前排期；本份因现有冻结检查/准确历史引用或原批准证据完整归档。
- [2026-09-03-shuomi-phone-registration-onboarding-plan-v7.md](implementation-plans/2026-09-03-shuomi-phone-registration-onboarding-plan-v7.md) — Greenfield已明确PR #283/#284 V7只是历史证据，不是当前实现基线；完整原文归档，移除“仍承载当前合同”宣传。
- [2026-09-03-shuomi-resource-ledger-store-plan-v7.md](implementation-plans/2026-09-03-shuomi-resource-ledger-store-plan-v7.md) — Greenfield已明确PR #283/#284 V7只是历史证据，不是当前实现基线；完整原文归档，移除“仍承载当前合同”宣传。

<a id="verification-history"></a>

## 实际验证记录

- [2026-08-09-listingkit-canonical-zitadel-subject.md](verification/2026-08-09-listingkit-canonical-zitadel-subject.md) — 原始实际执行与限制证据完整归档；旧操作不成为当前命令/运行授权。
- [2026-08-10-listingkit-owner-reconciliation.md](verification/2026-08-10-listingkit-owner-reconciliation.md) — 原始实际执行与限制证据完整归档；旧操作不成为当前命令/运行授权。

- [ZITADEL 多 Organization 验证](verification/zitadel-multi-organization-authorization.md) — 保留 2026-08-31 合成测试、本地冒烟及准确 SHA；真实双组织、撤销传播与启用 Workbench 仍为原记录的 pending。

原目录与已退出索引的完整内容可从[归位前固定 Git 版本](https://github.com/henryxu1008/shuomi-ai-commerce/tree/bafc9a27208e36b1e3c560cf5196b9c8d690e508/docs/superpowers)追溯。归档不是停用真实实例或清理真实数据的授权。

<a id="refactoring-history"></a>

## 历史重构材料

以下是旧阶段的计划、观察、执行/失败记录和测试原输出。它们保留原文与基线，不是当前排期；当前重构只从[规则入口](../refactoring/README.md)进入。

- [2026-09-05-issue-30-prepared-slices.md](refactoring/2026-09-05-issue-30-prepared-slices.md) — 旧sourcing/cutover准备与取消依据，Greenfield guard继续验证不能恢复旧迁移前置。
- [2026-09-05-publication-identity-cutover-issue-draft.md](refactoring/2026-09-05-publication-identity-cutover-issue-draft.md) — 已明确SUPERSEDED的旧身份cutover草案；保留取消边界与原分析，不再作开工前置。
- [checkpoint-validation-2026-06-17.md](refactoring/checkpoint-validation-2026-06-17.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。
- [current-refactoring-status.md](refactoring/current-refactoring-status.md) — 2026-09-05成熟度/阶段快照与后续有界记录，不是当前状态看板；保留原结果和Greenfield撤销要求。
- [2026-06-26-next-growth-sequence.md](refactoring/decisions/2026-06-26-next-growth-sequence.md) — 2026-06-26阶段增长排序决定按当时范围归档；当前排序属于#137与执行Issue，旧决定不重新派工。
- [dependency-analysis.md](refactoring/dependency-analysis.md) — 2026-06-08依赖数量与文件分布快照，不能作为当前源码目录或切包依据。
- [dependency-baseline.md](refactoring/dependency-baseline.md) — 2026-06-09真实依赖/包规模/已知测试失败快照，非当前工作表；当前扫描输出移到.local/refactoring，原数据归档。
- [final-summary.md](refactoring/final-summary.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。
- [httpapi-runtime-inventory.md](refactoring/httpapi-runtime-inventory.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。
- [legacy-store-routing-retirement-checklist.md](refactoring/legacy-store-routing-retirement-checklist.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。
- [listingkit-boundary-checkpoint.md](refactoring/listingkit-boundary-checkpoint.md) — 已SUPERSEDED FOR GREENFIELD的旧检查点，保留取消兼容/迁移要求的证据与原结果。
- [listingkit-frontend-refactoring-roadmap.md](refactoring/listingkit-frontend-refactoring-roadmap.md) — 旧UI/Task路线图含真实执行记录；原测量/交付证据归档，当前页面与排期从Figma和Issue读取。
- [listingkit-refactoring-progress-2026-06-24.md](refactoring/listingkit-refactoring-progress-2026-06-24.md) — 日期绑定的PR/源码/部署/测试观察记录保留，不再摆作当前进度。
- [next-phase-plan.md](refactoring/next-phase-plan.md) — 已明确不是当前execution authority的旧平台资产阶段计划，保留原批准与取消边界。
- [phase1-acceptance-report.md](refactoring/phase1-acceptance-report.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。
- [phase1-summary.md](refactoring/phase1-summary.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。
- [phase2-final-summary.md](refactoring/phase2-final-summary.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。
- [phase2-progress-report.md](refactoring/phase2-progress-report.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。
- [phase2-systematic-refactoring-summary.md](refactoring/phase2-systematic-refactoring-summary.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。
- [product-sourcing-inventory.md](refactoring/product-sourcing-inventory.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。
- [project-migration-roadmap.md](refactoring/project-migration-roadmap.md) — Greenfield已撤销其迁移/兼容实施权威，归档并继续保留反恢复断言。
- [project-wide-refactoring-plan.md](refactoring/project-wide-refactoring-plan.md) — 旧全仓拆解计划已SUPERSEDED，不能恢复旧facade/迁移；完整原计划与准入限制归档。
- [service-slimming-checkpoint.md](refactoring/service-slimming-checkpoint.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。
- [shein-pod-submit-readiness-policy-design.md](refactoring/shein-pod-submit-readiness-policy-design.md) — EvaluatePODSubmitReadiness等旧策略族已在#961准确退休矩阵列明；旧设计仅归档，不创造当前函数或兼容owner。
- [studio-migration-attempt-report.md](refactoring/studio-migration-attempt-report.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。
- [submission-inventory.md](refactoring/submission-inventory.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。
- [tdd-global-state-elimination.md](refactoring/tdd-global-state-elimination.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。
- [test-baseline.txt](refactoring/test-baseline.txt) — 原始Go测试输出原字节归档，失败/未执行不改成成功；新扫描不得覆盖此历史原件。
- [2026-07-09-doc-cleanup-validation.md](refactoring/validation/2026-07-09-doc-cleanup-validation.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。
- [2026-07-13-listingkit-backend-baseline.md](refactoring/validation/2026-07-13-listingkit-backend-baseline.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。
- [week1-progress-report.md](refactoring/week1-progress-report.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。
- [week2-final-summary.md](refactoring/week2-final-summary.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。
- [week2-progress-report.md](refactoring/week2-progress-report.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。
- [workflow-reorganization-decision.md](refactoring/workflow-reorganization-decision.md) — 旧阶段的实际执行/观察、失败或取消记录，完整保留其日期、数字、提交及验证结果；不再作为当前排期或操作入口。

<a id="architecture-history"></a>

## 历史架构、开发与运行材料

旧阶段快照、协议/PoC和排查记录退出当前目录，以下原结果及运行态UNKNOWN完整保留。普通过期排期、拆分与候选验证清单仅保留在[清理前固定版本](https://github.com/henryxu1008/shuomi-ai-commerce/tree/9990bad9ff14e31bc4d33f2d110f1ebac7b88939/docs/architecture)，不作为当前执行队列。

| 历史材料 | 保留依据 |
| --- | --- |
| [旧 Code Guide 原文](https://github.com/henryxu1008/shuomi-ai-commerce/blob/cfd08cf5425f0cc4fb05db6c7df4bc265cfb7f93/docs/CODE_WIKI.md) / [删除前导航页](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/CODE_WIKI.md) | 旧包规模、CURRENT STATE 与 ListingKit 快照只作历史；重复导航页已撤销，当前源码职责和正式命令统一见[源码入口地图](../development/repository-structure.md#current-entrypoint-map)。退休矩阵中的原路径记录保留 |
| [historical-platform-migration-inventory.md](architecture/historical-platform-migration-inventory.md) | 2026-08-26包规模与候选成本快照归档；原数量/判断及现有文档检查保留，当前owner依Project Boundaries/Legacy Register |
| [listingkit-refactor-status.md](architecture/listingkit-refactor-status.md) | 旧ListingKit边界与阶段完成观察归档；保留原结果及Greenfield否定compatibility的范围，不宣传当前状态 |
| [openmeter-shadow-metering-poc-report.md](architecture/openmeter-shadow-metering-poc-report.md) | #988已退休旧计量运行；镜像digest、资源观察、并发/quota与PoC结果属于必要实测证据，完整归档 |
| [task-event-v2-migration.md](architecture/task-event-v2-migration.md) | 数字任务V2协议及消费者已源码退休；schema/ACK/拒绝/旧观察门槛作为历史保留，不恢复两版发布/14天前置 |
| [task-status-lifecycle.md](architecture/task-status-lifecycle.md) | 旧数字Task状态机已退休；完整状态/错误/幂等历史归档，不作为BusinessTask/ImageAgent统一状态机，原guard断言保留 |
| [temporal-poc-runbook.md](architecture/temporal-poc-runbook.md) | #844确认旧Task SHEIN publish PoC退休；保留旧运行/恢复边界与远端UNKNOWN，当前Temporal基础设施/worker不变 |
| [temporal-workflow-evaluation.md](architecture/temporal-workflow-evaluation.md) | 2026-05-18旧PoC落地结果与评估归档；保留事实及恢复责任，不重新派工旧submit迁入，当前Temporal边界仍有效 |
| [temu-refactor-playbook.md](architecture/temu-refactor-playbook.md) | 旧TEMU Task管线编辑经验归档；示例owner/API已退休，当前TEMU合同与有效纯规则仍由现有owner承担 |
| [temu-pipeline-stages.md](architecture/temu-pipeline-stages.md) | 旧SKU/刊登映射管线与stage词汇归档；按#921及刊登退休矩阵保留原历史，不恢复ProductImportMapping |
| [temu-architecture-patterns.md](architecture/temu-architecture-patterns.md) | 旧TEMU管线对象与编排模式归档；保留原设计经验与准确替代说明，不作为当前API或第二套状态机 |
| [listingkit-local-prod-db-replay.md](development/listingkit-local-prod-db-replay.md) | 旧共库复验/转发/schema入口均退休；原操作及后续取消证据归档，不连接共享DB/Redis或读凭据 |
| [listingkit-shein-enrollment-source-sds-troubleshooting.md](development/listingkit-shein-enrollment-source-sds-troubleshooting.md) | 旧页面/Task/SDS fallback的实际排查样例归档；保留401/空结果/504观察，不恢复跨租户fallback |
| [source-account-ownership-preflight.md](operations/source-account-ownership-preflight.md) | Greenfield取消且#529退休的A/B1预检；真实junction/失败/支持平台结果保留，现有Greenfield检查只改读取路径 |
| [store-service-history-hardcut.md](operations/store-service-history-hardcut.md) | #552退休的Store历史迁移与约束/backfill说明归档；保留原安全/UNKNOWN/授权停止点，不执行旧数据操作 |

<a id="deployment-and-integration-history"></a>

## 历史部署与旧集成联调

- [旧ListingKit Kubernetes部署/发布/恢复记录](deployments/listingkit-workbench-k3s.md)：原IAM/Secret边界、drain/rollback与durable staging约束完整保留；当前保留清单见[部署入口](../../deployments/kubernetes/listingkit-workbench/README.md)，不执行历史命令。
- [旧SDS联调记录](integrations/sds-original-trial.md)：2026-04-23等真实接口观察、失败与恢复样本保留；旧client/login/adapter已退休，当前POD/SDS沿[现有集成](../../internal/integration/sds)、[POD应用](../../internal/app/pod)及原Product/Asset/Submission owner。
