# Research：市场证据、商业评估与人工选品

> **阅读导航，不替代现行 Research 合同与产品决定。** 当前事实边界以 [项目边界](../../project-boundaries.md)、[最小研究领域合同](market-research-selection.md) 为准。Refs [#1343](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1343)。

## 用户流程与权威材料

| 用户操作/关键问题 | 适用文档 | 当前 owner / 源码 |
| --- | --- | --- |
| 用户从市场、关键词、类目或具体商品开始研究 | [市场调研与智能选品产品流程](../../../product/flows/market-research-selection-centers-v1.md)、[多来源规划](../../../product/market-research-multi-source-plan.md) | 研究项目/入口按当前 Project Center 和 Research 合同 |
| 在研究项目中引用已保存的来源观察和候选 | [市场研究与选品最小领域合同](market-research-selection.md)、[研究证据分析增量](market-research-evidence-analysis-v1.md) | `internal/product/research/`（准确引用与决定），Collection/Data Services（来源读回） |
| 在批准的一笔试用中取得 Amazon 美国站关键词第一页的样本 | [ZenRows 单页试用](../../amazon-us-zenrows-first-page.md) | Product Data Acquisition（取数与保存），Research（比较） |
| 在准确候选内人工核对供应货源、SKU 与数量 | [人工货源核对合同](research-supplier-review-v1.md) | `internal/product/research/supplier_review.go` |
| 录入费用假设，计算并保存经营评估 | [人工参考试算历史合同](research-manual-cost-preview-v1.md)、[评估持久化现行增量](research-commercial-assessment-persistence-v1.md) | `internal/product/research/cost.go`、`assessment.go` |
| 比较 2–5 件自己已保存的商品，按需生成有据建议 | [独立商品对比合同](product-selection-comparison-agent.md)、[智能体业务职责](../../../product/decisions/business-agent-responsibilities.md) | Collection/Catalog（私有候选和固定版本）、`internal/product/selection/`（比较）、Agent/agentconfig（运行与配置） |
| 用户作出人工决定，并显式转入商品准备 | [市场研究与选品合同](market-research-selection.md)、[商业评估产品边界](../../../product/flows/research-commercial-assessment-v1.md) | Research 决定及原 Collection/Listing 交接，各自持有事实 |

## 事实与验证的分界

- **市场观察不等于真实销量/市场整体趋势。** 保留来源、采集时间、站点/国家、版本及数据可得性；缺真实销量、采购报价、MOQ 或适用费用时明确 UNKNOWN，不自动补齐。
- **Research 不复制 Canonical ProductSnapshot。** 研究证据关系、候选比较、经营假设/评估及人工决定属于 Research；原始来源/商品/已批准素材仍分别属于 Sourcing、Catalog、Collection、Asset；平台提交由 Listing/Marketplace 负责。
- **参考成本计算与人工决定分开。** 早期浏览器内临时试算方案已由 [#1205 评估持久化增量](research-commercial-assessment-persistence-v1.md) 进一步收敛，不能恢复两份并行的权威利润计算，也不能把正利润自动转成批准/采购。
- **销售目标与资料来源分开。** 当前优先研究目标为 SHEIN 美国站；Amazon 可作为竞品/市场资料、1688 可作为供货来源，不能把来源平台当作销售目标或由合成样本推断真实平台已接通。
- **2–5 件候选的固定对比只是获准窄切片。** 原合同/测试能证明准确样本比较与人工决定路径，不证明从任意市场方向主动发现机会的完整智能体已经可用。AI 执行仍需要单独启用、允许工具、当前预算/权限与准确回执。

## 交付状态在哪里查询

[市场调研 Epic #1070](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1070)、[竞品调研 #1072](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1072)、[选品决策 #1073](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1073)、[货源/利润 #1074](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1074) 和 [执行排序 #137](https://github.com/henryxu1008/shuomi-ai-commerce/issues/137)。不能用设计冻结、PR 合并或离线 PASS 替代真实来源、模型、费用、企业权限或客户验收。

本批只建立阅读导航与适用边界，原冻结合同保留路径和历史证据。
