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

## 图片语言与 API 最小权限合同

- [目标语言、已确认文案与不可变计划](image-text-language-confirmation.md)：沿原 Review、准确模板/Skill、预算及人工确认，不覆盖历史未完成真实效果。
- [图片事件表最小读取权限](image-api-event-read-permissions.md)：原 API role 的有界 SELECT 修正与权限测试；仅保留既有设计和历史状态，不修改实际授权。
