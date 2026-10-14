# 单件自有商品到待补全草稿

Refs #694；[既有合同澄清](../architecture/own-product-draft-entry-v1.md)。复用Collection、Preparation和CompletionDraft，不新增schema、领域owner或运行服务。

## 正常使用

1. 正常登录并确认当前企业，进入 `/workbench/data/mine?view=own`，打开已有自有商品。
2. 点击“创建待补全草稿”，进入 `/workbench/data/mine/own/<itemId>/prepare`。页面读取当前所属批次版本后再读取展示资料，确认来源版本与图片、属性、SKU。
3. 点击“确认转入并填写草稿”。原Transfer只选择该item，成功后进入 `/workbench/supply/mine/missing/<sourceId>`。
4. 页面明确“尚未保存”；候选标题/描述与固定原资料分开。填写后点击“保存待补全草稿”，显示实际保存Rn，刷新或重新进入继续编辑。
5. 待补全列表的“查看 / 编辑”使用同一独立页。当前已存在 `missing?preparation=<id>&source=<id>` 入口也展示同一独立内容，不使用弹窗。

之后编辑、移动或归档Collection商品不改已有来源和草稿；原固定版本仍按当前data/supply及原member权限读取。确认后的批次变化（包括同批其它商品）按原CAS拒绝，重新读取后确认。该页面默认当前声明版本，不选择旧版本，不自动升级旧草稿。

## 结果核实

转入与候选保存是两项原业务操作，不声称一次原子提交。转入后关闭编辑器，原Preparation仍在“待适配”，候选还未保存；之后可从那里继续。

UNKNOWN沿原Supply intent保存key/body/scope，点击“核实原操作”或“重试原请求”，不能换key自动重发。确认页只核实属于这件商品的单件transfer；已成功但source读取失败时保留成功回执，使用“重新读取已转入来源”或原“查看已转入商品”，不重复转入。恢复旧key读取其实际固定来源，不能从当前商品重建旧结果。

新的明确转入操作使用新key，建立独立Preparation；不覆盖已有草稿。同key异载荷和候选保存revision冲突仍由原owner拒绝。独立编辑器核对已保存record与展示固定source一致后才预填，恢复后读取实际head。

## 已有运行能力与安装边界

Backend私有Application manifest需要既有 `productCollections:true` 和 `supplyCompletionDrafts:true`；保持原身份、membership、Commercial/resource、Product配置和所有已有enabled能力。该路径不要求Store/SHEIN、模型、Asset批准或worker，不打开正式Target/上传。

UI需要既有 `LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED=true`、`LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED=true`。缺少能力不显示入口；服务端仍执行live权限，前端开关不代表运行成功。当前data read + supply read/manage允许转入和填写；只读身份不能保存。

新安装沿当前空库初始化与正常应用启动合同。现有保留实例若尚无CompletionDraft schema/grants，必须按 [原有bounded additive owner安装](supply-unbound-completion-drafts.md) 获取准确数据库、原runtime角色及serving更新的具体授权后再执行。该说明本身不授予部署/schema/grants或业务写入权限。

原owner入口仅安装/验证Preparation两表与CompletionDraft三表并追加最小权限，不调用旧非空库全量初始化，不重置原grant、角色或数据。启用后的所有Product消费者使用一致enabled capability union；constructor只Verify，不在startup安装。现有Store、worker、数据、图片、容量账本、备份和命名卷保留。原保留start脚本继续避免重跑setup。

候选源码、PR CI和隔离PostgreSQL测试不代表当前实例已经更新，也不代签用户/生产验收。交付按原准入执行必要开发自检、稳定CI和独立检查；保留实例安装及实际页面试用另记录准确代码/镜像和结果。真实provider/平台、共享或生产部署、Issue关闭与数据清理仍不属于本任务。
