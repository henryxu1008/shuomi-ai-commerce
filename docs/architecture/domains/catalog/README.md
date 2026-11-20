# Catalog：商品事实、版本、引用与素材批准

> **阅读导航，不定义新的商品模型或事实 Owner。** 事实边界以 [项目边界](../../project-boundaries.md) 与 [Product Domain Contracts](../../product-domain-contracts.md) 为准。Refs [#1343](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1343)。

## 按使用任务查阅

| 用户或开发问题 | 首选合同/业务说明 | 源码入口 |
| --- | --- | --- |
| 外部采集证据如何变成不可变商品版本 | [Product Domain Contracts](../../product-domain-contracts.md)、[来源交接](../../../product/product-sourcing-handoff.md) | `internal/product/catalog/`、`internal/product/sourcing/` |
| 用户如何把商品放进“我的数据”、选择和引用版本 | [Product Domain Contracts](../../product-domain-contracts.md)、[来源业务流程](../../source-business-flow.md) | `internal/product/collection/` |
| 自有商品录入、规格及保存历史 | [自有商品录入](own-product-entry.md)、[编辑](own-product-edit.md)、[SKU 包装](own-product-sku-packaging.md) | `internal/product/collection/`、`internal/product/catalog/` |
| 商品描述如何安全编辑、插图并保留版本 | [富文本内容合同](own-product-rich-description.md)、[编辑器与描述图片增量](rich-description-simple-editor-images.md) | `internal/product/description/`、`internal/product/collection/`、`internal/listing/completiondraft/` |
| 本机来源图片试用的存储与容量边界在哪里 | [来源图片试用合同](own-product-source-media-trial.md)、[操作说明](../../../operations/runbooks/own-product-source-media-trial.md) | `internal/app/productsourcing/`、`internal/ledger/storagecapacity/` |
| 来源图片、生成图和用户上传图什么时候可用于刊登 | [Product Domain Contracts](../../product-domain-contracts.md)、[素材来源与批准](../../source-business-flow.md) | `internal/product/asset/`、`internal/imageagent/` |
| 查看已批准图片的事实检查与非执行性建议 | [图片事实与建议](product-image-facts-advice.md) | `internal/product/asset/`（已批准素材与纯检查），Listing/Marketplace（当前 Target 和规则） |
| 从商品资料带入目标平台的草稿 | [待补全草稿产品决定](../../../product/flows/supply-unbound-completion-drafts.md)、[对应架构合同](../listing/supply-unbound-completion-drafts.md) | `internal/listing/`、`internal/marketplace/` |
| 自有商品单图整理如何准入、遇到拒绝如何保留原账本边界 | [OpenAI Mini 单图接入](own-product-intake-openai-mini.md)、[原调用未计费恢复](own-intake-provider-unbilled-recovery.md)、[运行说明](../../../operations/runbooks/model-route-pricing.md#原单图调用的明确拒绝未计费恢复) | `internal/agent/`、`internal/aicapability/`、`internal/integration/orgresource/` |
| 已保存商品如何按目标语言及字段生成完整内容候选并原子采用 | [商品内容与本地化](product-content-localization.md) | `internal/product/contentagent/`、`internal/product/review/`、`internal/product/catalog/` |
| 已保存自有商品如何无店铺优化并形成真实 Apply 资格 | [自有商品内容优化](own-product-optimization.md)、[优选资格](../supply-market/supply-market-choice-eligibility.md) | `internal/product/contentagent/`、`internal/product/review/`、`internal/product/catalog/` |

## 关键分工

- **Sourcing** 保存来源身份、归一化包络和发布证据；**Catalog** 拥有 canonical immutable ProductSnapshot/版本；**Collection** 管理当前授权用户/企业的商品引用、批次及显式转入；**Asset** 管理素材批准的权威记录。相关原子性、同键冲突、来源版本及租户身份沿已批准的原合同。
- 已保存商品、来源图片、待补全候选内容或 AI 草稿 **都不自动证明** 已批准素材、真实报价、平台校验通过或远端发布成功。Listing/Marketplace 分别拥有目标平台资料、草稿和提交状态；Research 只引用准确已保存事实。
- `internal/product/*` 的代码目录反映真实实现，不代表需要新建一个统一 Catalog/Source/Collection 数据库或在本文复制完整 Schema。所有命令和环境准入继续按具体受控入口。

## 有效性与使用建议

当前产品入口可以变化（如导航从“我的供应链”调整为“商品刊登”），而商品事实、不可变版本和授权边界不因 UI 路径变化而重新归属。历史 Figma 页面仅可辅助理解业务操作，不能作为新产品界面权威。

本批**不搬动** `product-domain-contracts.md`、`source-business-flow.md` 或其他被大量引用的合同。后续若确需迁移，先审核消费者与 Markdown 锚点，再在独立 PR 更新。
