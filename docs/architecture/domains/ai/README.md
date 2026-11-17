# AI 领域文档：#132 标题评测候选

本页是架构合同的**定位入口**，不对设计进行准入、也不授权真实模型调用。参见 [AI Capability](../../ai-capability-platform.md)、[Product Agent 运行合同](../../product-agent-runtime-contract.md)、[项目事实边界](../../project-boundaries.md)及 [Issue #132](https://github.com/henryxu1008/shuomi-ai-commerce/issues/132)。

## 当前迁入的两份候选

- [标题对照的模型准入、固定组与批次预算](issue132-title-evaluation-admission.md)：原 `DESIGN_CANDIDATE / NOT_IMPLEMENTATION_READY`，正式评测 `NOT_RUN`；预算和模型边界必须通过原 Issue/评审确认。
- [标题先导的本机操作入口](issue132-title-pilot-local-entry.md)：原 `R2_CANDIDATE`，仅讨论隔离运行装配、操作者授权、原键恢复；不等于受权启动、付款或正式评测。

原仓库路径分别为 `docs/architecture/issue132-title-evaluation-admission.md` 和 `docs/architecture/issue132-title-pilot-local-entry.md`。本批只进行文档位置归类和相对引用调整，原规则、结论和评审状态保持；历史提交可由 Git 追溯。迁移执行 [#1357](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1357)。

## Provider 与应用执行生命周期合同

- [Provider-neutral AI 请求/响应与适配接口](ai-provider-contracts.md)：迁移历史设计；模型路由、租户、治理、预算及供应商请求行为不改变。
- [关闭新 AI 执行并保留配置和历史读取](ai-disabled-application-lifecycle.md)：原 #1073/#1075 离线实施合同，原权限、原键与 UNKNOWN 不变；不授权真实调用或部署。

## 官方技能的受控准备与商品入口

- [官方模板自动准备](official-skill-automatic-preparation.md)：沿既有唯一 Skill owner、固定发布包与事务锁；保留当前权限和历史版本。
- [官方技能直接处理已有商品](official-skill-product-entry.md)：复用准确版本选择、原商品与图片来源验证、人工确认及原冻结运行。

目录移动不改变 #1276、#774 的产品决定，也不授权实例发布、真实模型调用或付费。

## 图片方案与执行职责

- [通用图片模板与技能入口](generic-image-template-entry.md)：准确 Skill 版本和企业模板选择的历史设计，真实调用预算及用户验收仍沿原门禁。
- [Image Agent Workflow](image-agent-workflow.md)：Run/Plan/Slot、Temporal 编排、派发保护、UNKNOWN、显式人工批准及不可重复副作用的原有职责说明。

上述路径归类不变更原 #774 产品决定、执行状态、权限或模型预算。

## 图片语言与 API 最小权限合同

- [目标语言、已确认文案与不可变计划](image-text-language-confirmation.md)：沿原 Review、准确模板/Skill、预算及人工确认，不覆盖历史未完成真实效果。
- [图片事件表最小读取权限](image-api-event-read-permissions.md)：原 API role 的有界 SELECT 修正与权限测试；仅保留既有设计和历史状态，不修改实际授权。

## Image Agent 外部效果终结协议

- [效果终态与取消保护](image-agent-effect-terminalization.md)：历史设计，关注 provider/staging/publication claimed 时的取消及最终状态边界。
- [外部效果收尾与预算结算](image-agent-effect-finalization.md)：历史设计，关注 v3 Temporal 原版本门禁、明确未发送/未知结果与结算。

两份合同保持各自原有版本、阶段约束及历史决定；目录整理不将时间窗不同的方案自动合并，也不批准真实执行。

## Image Agent 效果恢复及兼容协议

- [外部效果独立恢复工作流](image-agent-effect-recovery.md)：保留 v3 claimed/UNKNOWN 恢复事实与不重复调用 provider 约束。
- [Image Agent Recovery and Compatibility](image-agent-recovery.md)：保留原 v2/v3 兼容、S3 持久归档、lease/fence 恢复的历史合同。

仅更改阅读路径；不合并不同阶段的恢复协议，不授权实际重试或生产部署。

## 图片预算准入与候选发布保护

- [Image Agent Budget Authorization](image-agent-budget-authorization.md)：原有预算作为外部效果执行许可、预留/结算及 UNKNOWN 的历史设计。
- [Image Agent Candidate Release Gate](image-agent-candidate-release-gate.md)：候选 worker 的隔离兼容验证及生产提升边界；不代表已有部署许可。

仅迁移历史合同的阅读位置，不改变预算、权限、真实 worker 发行或环境状态。

## AI 供应商准入与受控绑定

- [DeepSeek 共享文字参考价准入](bailian-deepseek-shared-text-admission.md)：保留原 #1075/#1078 参考估算、配额、原账本及离线接线边界，不将候选费率视为真实账单。
- [GRSAI VIP 图片绑定](grsai-vip-image-binding.md)：保留原 #840 准确型号、版本、凭据、medium/1024 约束及后续可选路线的新决定，不自动发布绑定。

以上合同只调整所在目录，不改变已有模型路由、凭据身份、预算、UNKNOWN 或真实运行资格。

## 人工智能体定制与原图恢复

- [智能体定制人工进度](agent-customization.md)：原人工需求、线下报价及私有交付后续准入分别保留，不以需求创建自动执行。
- [VIP 已付费原图恢复](vip-original-output-recovery.md)：保留准确原请求/权限/结算证据和 UNKNOWN；恢复不触发再次生图或扣费。

迁移仅涉及文档路径，不签发新模型调用、真实支付、部署或业务验收。

## 百炼图片接线与 Google 标题传输

- [百炼商品图片接线 V1](bailian-product-image-wiring.md)：保留固定模型路由、企业凭据、原预算及图片 UNKNOWN 与真实试用限制。
- [Google Interactions 标题路由增量](product-agent-google-interactions.md)：保留独立原生协议的来源、计量边界、旧兼容探针历史及人工 Review/Apply 约束。

目录迁移不触发供应商调用、不发布配置、不改变历史准入或费用上限。

## Image Agent V3 效果状态策略

- [Image Agent V3 Effect Policy](image-agent-effect-policy.md)：保留原纯领域决策、repository CAS、UNKNOWN、预算/租约/fencing 与 Temporal 兼容性设计，不据目录变更重新授予实现或部署权限。

## 单商品图片可选模型路线

- [Single-product image plan optional route V1](image-plan-optional-route.md)：保留 #840 原选路、准确候选版本及人工确认的历史设计边界，不将文件迁移视为模型发布或生产授权。
