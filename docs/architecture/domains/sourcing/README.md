# Sourcing：来源证据与受控发布

> **阅读导航，不是第二份领域合同。** 实际事实 owner 与依赖方向以 [项目边界](../../project-boundaries.md) 和 [Product Domain Contracts](../catalog/product-domain-contracts.md) 为准；本页只帮助从当前业务场景找到准确合同、代码与运行说明。Refs [#1343](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1343)。

## 从哪里开始

| 使用场景/要回答的问题 | 先读哪份权威文件 | 代码与实现入口 |
| --- | --- | --- |
| 商品来源如何变成固定版本的 ProductSnapshot | [Product Sourcing Handoff](../../../product/product-sourcing-handoff.md)、[Product Domain Contracts](../catalog/product-domain-contracts.md) | `internal/product/sourcing/`、`internal/app/productsourcing/`、`internal/product/catalog/` |
| 来源证据、发布身份、同键重放与结果待核实 | [来源业务闭环合同](source-business-flow.md)、[Sourcing 包说明](../../../../internal/product/sourcing/README.md) | `internal/product/sourcing/`、`internal/integration/persistence/product/sourcing/` |
| 1688 公开商品页如何采集 | [1688 受控浏览器采集合同](../../1688-server-public-browser-acquisition.md)、[产品决定](../../../product/decisions/1688-server-public-browser.md) | `internal/integration/acquisition/a1688/browser/`、`internal/integration/acquisition/browsercollector/` |
| Amazon / 1688 插件的规格图片、来源接收与原键保存设计 | [历史增量设计](plugin-capture-ingestion.md)、[采集字段产品对照](../../../product/decisions/plugin-capture-listing-fields.md) | `internal/app/productsourcing/`、`internal/product/sourcing/`、`extensions/product-capture/` |
| 已保存来源怎样进入 Collection/后续商品准备 | [Product Sourcing Handoff](../../../product/product-sourcing-handoff.md)、[商品主领域合同](../catalog/product-domain-contracts.md) | `internal/product/collection/`、`internal/product/asset/` |

## 本领域不变量

- **来源证据与商品事实不同。** 外部来源/浏览器采集结果先作为待核对证据；`SourceEnvelope`、`SourceIdentity`、不可变发布回执及准确版本由原 Sourcing/Catalog owner 管理。来源平台（例如 1688）不是最终销售平台（例如 SHEIN）。
- **新采集能力与历史发布读回不同。** 2026-11-08 已合并 [PR #1193](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1193) 退休 1688 匿名静态 HTTP provider；当前仓库 `a1688/` 下仅保留 `browser/`。未配置明确批准的 browser collector 时不得恢复静态 HTTP 默认抓取或 fallback。历史 `public_http` 证据标识的读回/验证按原事实 owner 的权限、版本与回执处理，不能由此推断该采集器仍能接受新请求。
- **源账号连接不是匿名采集前置。** 可选 1688 Connection [#396](https://github.com/henryxu1008/shuomi-ai-commerce/issues/396) 为 Later；无用户 1688 登录态的范围不允许隐式依赖旧 Cookie/profile。身份/企业/额度、幂等、COMMIT/UNKNOWN 与原键恢复仍必须遵守现行合同。
- 图片 URL、模型建议、商品候选不会自动成为 ApprovedAsset；采用和平台发布另由原 owner 检查。

## 哪些材料不能直接代表当前能力

[旧 SRC-2B1 任务 #398](https://github.com/henryxu1008/shuomi-ai-commerce/issues/398)、[市场调研总体目标 #1070](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1070) 的历史阶段正文和旧 provider 比较是追溯资料。判断准确实例是否能真实采集、是否有合法来源与权限时，应读取该任务最新范围及实际配置/回执；设计/CI 不代签现场运行结果。

**本批只添加导航，不移动或替换上面的合同文件。**
