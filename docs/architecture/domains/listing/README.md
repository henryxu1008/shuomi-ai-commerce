# Listing：商品准备、草稿与提交

> 这是现有合同的**阅读索引**，不新增事实 Owner 或提交权限。统一架构入口：[项目边界](../../project-boundaries.md)、[目标架构](../../project-target-architecture.md)、[Listing 源码说明](../../../../internal/listing/README.md)。Refs [#1351](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1351)。

## 从业务操作选择合同

| 操作 | 现行合同与操作说明 | 原事实 Owner |
| --- | --- | --- |
| 从我的数据选商品、配置店铺、确认刊登 | [四步供应链及原任务继续](../../supply-four-step-continuation.md)、[商品刊登操作](../../../operations/my-supply-chain-shein.md) | `internal/product/collection` 选择商品，`internal/listing/preparation` 准备与继续 |
| 保存不绑定店铺的待补全候选 | [未绑定草稿产品决定](../../../product/supply-unbound-completion-drafts.md)、[CompletionDraft 结构化合同](completion-draft-structured.md) | `internal/listing/completiondraft` |
| 核对平台资料、类目、素材与校验规则 | [平台资料智能体 v1](platform-listing-materials-agent.md)、[SHEIN 原生 Validator](../../shein-native-validator-contract.md)、[提交前 readiness 策略](../../shein-submit-readiness-action-policy.md) | `internal/listing/materials` / `readiness` 及当前 Marketplace 规则 |
| 选择 SHEIN 候选类目并只读查询规则 | [候选类目规则查询与解释（C3-2）](listing-candidate-category-rules.md) | `internal/listing/materials` 使用 Marketplace 提供的真实规则；不保存、不采用、不发布 |
| 人工审核与素材批准 | [Product Domain Contracts](../../product-domain-contracts.md)、[Human Review #36](https://github.com/henryxu1008/shuomi-ai-commerce/issues/36) | `internal/product/review`、`internal/product/asset` |
| 执行一次明确批准的提交并核对结果 | [SHEIN 发布闭环](../../shein-publication-flow.md)、[原任务继续](../../supply-four-step-continuation.md) | `internal/listing/submission` 持有提交意图/回执，平台 adapter 持有协议 |

## 草稿与备注的当前合同

- [保存历史与差异比较](completion-draft-history.md) — 精确版本的只读历史，旧记录不可改写。
- [恢复旧版内容为新版本](completion-draft-restore.md) — 用户显式预览并普通 Save 为新版本，保留原历史版本。

- [待补全草稿内部备注](completion-draft-internal-note.md) — 原 Listing CompletionDraft 的可选纯文本内部备注；沿已有事务、权限与版本保存，不进入任何平台预览或发送载荷。关联 [#1168](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1168)。

## 当前业务视图与事实边界

当前一级入口为**商品刊登**，四个视图是**待上传、待补全、待审核、已上传**；待上传内四步为选择商品、选择目标店铺、配置规则与智能体、确认提交。已上传只读已有发布记录，SHEIN 首次人工上架由获授权的**店铺商品**路径执行，不在已上传恢复二次提交按钮。见 [当前流程与历史 UI/IA 决定](../../../product/final-ui-ia-authority.md)；[旧 SHEIN 完整设计](../../my-supply-chain-shein.md)的五阶段顶层投影已被后续四阶段产品决定替代，其他有效安全合同保留。

- **Catalog/ProductSnapshot、来源证据和 Asset 批准**不归 Listing；候选草稿不改写来源事实或已批准素材。
- **本地 CompletionDraft ≠ 官方平台草稿**；保存候选、确定性校验、人工批准和平台发送分属不同准入，不能相互代签。
- **受理/审核/首次上架/当前在架库存**分别保存或观察。响应未知保持原操作身份和 UNKNOWN，按原提交 owner 核实，不盲目重发。
- 不创建第二套权限、资金/资源账本、任务调度或跨平台重试服务。

完整 SHEIN 商业 E2E 与客户发布 Gate 分别在 [#28](https://github.com/henryxu1008/shuomi-ai-commerce/issues/28)、[#33](https://github.com/henryxu1008/shuomi-ai-commerce/issues/33) 验证。本页不代表任一真实店铺已开放。
