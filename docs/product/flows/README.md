# 当前业务流程阅读导航

> **本页只整理入口和交接关系，不建立新的产品/领域合同。** 适用的用户决定按[产品决策与历史 UI/IA 参考](../final-ui-ia-authority.md)、[当前产品战略](../ai-commerce-agent-platform-strategy.md)及[总路线图 #137](https://github.com/henryxu1008/shuomi-ai-commerce/issues/137)读取；实际实现、权限和运行验收在相关原 Issue/PR。Figma 只作为业务流程参考，不是最终 UI/IA 标准。

## 先选择你的用户起点

| 起点和目标 | 下一步真实操作与边界 | 产品/领域依据 | 操作与验收 |
| --- | --- | --- | --- |
| **A：已经有自有或来源商品，要发布到 SHEIN** | 我的数据/自有商品 → 商品刊登待上传 → 选择商品和店铺 → 检查并补全 → 人工确认 → 按已开放能力提交与读取结果。无需先做自动市场调研或完整利润评估 | [来源到标准商品](../product-sourcing-handoff.md)、[四阶段刊登](../../architecture/supply-four-step-continuation.md)、[SHEIN 现行发布与回执](../../architecture/domains/listing/shein-publication-flow.md) | [刊登操作说明](../../operations/my-supply-chain-shein.md)、[商业闭环 #28](https://github.com/henryxu1008/shuomi-ai-commerce/issues/28)、[发布 Gate #33](https://github.com/henryxu1008/shuomi-ai-commerce/issues/33) |
| **B：已有市场候选，希望核对货源和利润再决定** | 选择同一项目的准确候选 → 查阅现有市场证据 → 核对商品规格、来源和意向数量 → 参考费用/利润 → 保存适用评估 → 人工决定并显式转入商品准备 | [经营评估产品范围](research-commercial-assessment-v1.md)、[研究与选品业务设计](market-research-selection-centers-v1.md)、[Research 评估版本合同](../../architecture/domains/research/research-commercial-assessment-persistence-v1.md) | [人工货源核对合同](../../architecture/domains/research/research-supplier-review-v1.md)、[货源与利润 #1074](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1074)、[选品决定 #1073](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1073) |
| **C：只有国家、市场方向、关键词或类目，要主动研究机会** | 在智能市场发现研究职责 → 有权限的企业配置者通过“我的智能体”启用和配置 → 选择/创建研究项目 → 核对允许的来源、范围与预算 → 使用真实可用证据形成候选 → 人工选择后交接选品 | [智能市场交付方向](market-research-selection-centers-v1.md)、[最小研究领域合同](../../architecture/domains/research/market-research-selection.md)、[多来源规划](../market-research-multi-source-plan.md) | [市场研究 Epic #1070](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1070)、[资料与候选 #1072](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1072)；完整主动发现与真实来源接线须分别核验，不能凭有限 ASIN 样本宣称可用 |
| **D：已有 1688/Amazon 链接或浏览器采集证据，要保存商品资料** | 有效当前企业和用户权限 → 合格采集通道 / 插件 → 来源证据与 SourceEnvelope → SRC/Catalog 不可变发布 → Collection/我的数据 → 再决定是否转入刊登 | [来源交接](../product-sourcing-handoff.md)、[Product 唯一事实合同](../../architecture/product-domain-contracts.md)、[1688 浏览器采集当前边界](../../architecture/1688-server-public-browser-acquisition.md) | [浏览器采集插件说明](../../../extensions/product-capture/README.md)、[1688 采集计划的本地试用说明](../../operations/user-guides/acquisition-plans-local.md)；未配置 collector 不恢复静态 HTTP fallback |

## 如何避免交接时混淆业务事实

- **销售目标 ≠ 来源渠道。** SHEIN 美国是当前优先研究/刊登目标；Amazon 观察与 1688 供货是不同的资料来源。目标站点和来源覆盖都必须据实核对。
- **市场观察 ≠ 销量/利润证明。** 参考报价、采购意向、MOQ、库存、销量以及未核实的材料必须明确分开；缺可靠数据用“未知/待核实”，不能以样本或模型猜测写成已证实经营事实。
- **采集成功 ≠ 已批准图片或已发布商品。** Sourcing/Catalog 拥有来源与标准商品，Collection 拥有授权范围内的引用，Asset 拥有素材批准，Listing/Marketplace 管理草稿、规则与远端提交，Research 仅持有准确证据关联和人工决定。
- **草稿保存 ≠ 发布确认。** 修改资料、同意 AI 建议、Apply、提交平台与读取实际结果分别有当前权责；响应未知或权限变更时按原键、准确版本和原事实 owner 核实，不凭页面提示重新发送高风险操作。
- **目标设计 ≠ 已交付能力。** 具体模型/Agent、合法可用来源、实际店铺和真实平台操作是否开放，要以当前实例、准确版本及各 Issue 的实证为准。业务路径可供阅读，不代表已完成客户 E2E。

## 与代码、架构和运行的衔接

[项目领域边界](../../architecture/project-boundaries.md) 是唯一事实归属总入口；[源码地图](../../development/repository-structure.md) 定位可执行程序与真实代码；[架构索引](../../architecture/README.md) 定位更细合同；[运行索引](../../operations/README.md) 说明实际环境与操作。单次需求、提交、CI、人工审查和发布验收继续记录在原 Issue/PR，不在这份流程导航维护滚动 SHA 或状态。

本页是阅读导航，不能作为新增 AI、网络来源、采集收费、外部副作用或数据操作的授权。
