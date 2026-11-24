# Marketplace：平台协议与当前授权边界

> **导航，不代表平台能力正式开放。** 先读 [项目边界](../../project-boundaries.md)、[目标架构](../../project-target-architecture.md)和[店铺合同](../../store-center-application.md)。Refs [#1351](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1351)。

## 按目标平台阅读

| 平台 | 当前领域合同 | 实际代码入口 |
| --- | --- | --- |
| SHEIN 美国站 | [SHEIN 发布、审核与首次上架](../listing/shein-publication-flow.md)、[原生校验](shein-native-validator-contract.md)、[提交动作分类](shein-submit-readiness-action-policy.md) | `internal/marketplace/shein`、`internal/listing/submission` |
| TEMU（模式和站点须分别准入） | [TEMU 设计](temu-platform.md)、[官方合同证据](temu-official-contract-evidence.md) | `internal/marketplace/temu` |
| TikTok Shop 美国跨境 | [TikTok Shop US](tiktok-shop-us.md)、[官方合同证据](tiktok-shop-us-contract-evidence.md)、[发送前恢复](../listing/tiktok-pre-dispatch-no-send.md) | `internal/marketplace/tiktokshop` |
| Walmart 美国站 | [Walmart US](walmart-us.md)、[官方合同证据](walmart-us-contract-evidence.md)、[Feed 恢复](../listing/walmart-feed-recovery.md) | `internal/marketplace/walmart` |

已完成设计准入、待实现：[Shopify 店铺自助授权与商品草稿上传](shopify-product-draft.md)（#1613）。

当前代码中的平台目录不证明该平台全部站点、经营模式、官方应用、API 凭据或客户验收已经可用。Amazon 研究与来源采集是独立来源/研究能力，不自动等同于当前已开通的 Amazon Listing 平台适配器。

## 三个不同的事实责任

- **Store Center** 拥有企业的店铺记录、当前官方连接、授权范围与官方观察（`internal/storecenter`）。已保存店铺不等于已批准外部操作。见[Store 平台观察](../../store-center-platform-observations.md)。
- **Listing** 持有准备资料、目标商品版本、稳定提交身份、原回执和 UNKNOWN；其 submission owner 决定可否外部提交，而不是由前端或 Agent 直接操作平台。
- **Marketplace Adapter** 提供具体站点/模式的规则、校验、协议映射及受控平台端点，不拥有 Product、Asset、Money、Resource、Review 或另一套提交状态机。SDK 的成功/失败、平台审核与在售观测必须保留准确语义和时间。

**本地草稿不是远端 AS_DRAFT，上传受理不是平台审核通过，当前在架状态不证明原请求已成功。** 平台连接、经营模式、资料版本、权限、人工确认、额度和未知结果恢复依原 owner 逐项核实；不跨平台复用外部操作键。

查找用户端操作先看 [Listing 导航](../listing/README.md)。真实平台和客户发布验收看 [#28](https://github.com/henryxu1008/shuomi-ai-commerce/issues/28)、[#33](https://github.com/henryxu1008/shuomi-ai-commerce/issues/33)及各平台具体执行任务。本页不修改外部 API 或真实部署。
