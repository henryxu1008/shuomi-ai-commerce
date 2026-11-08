# 驾驶舱 · TikTok Shop US 提交审核率

状态 `IMPLEMENTATION_READY`。执行 [#1163](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1163)，用户批准 TikTok Shop US 单店统计及隔离本地验收。

## 产品范围

运营人员显式单选一家 TikTok Shop US 店铺时，驾驶舱独立显示「TikTok Shop US 提交审核率」，使用首页原 UTC+8 完整日、店铺和身份筛选。选择全部、仅选平台或选择其他平台时隐藏。复用[驾驶舱刊登与任务口径](cockpit-listing-task-metrics.md)及 [Walmart 单店接线](cockpit-walmart-feed-rate.md)，保留全平台卡片、GMV 和合成预览。Figma 只作流程参考。外部平台联调、正式部署和执行写路径在范围外。

按原 `organization_id, attempt_id` 去重，期间取原 attempt 创建时间。只读 `tiktok_publish_us` 的全球商品最终 US 发布，由原 intent 的 global 回执引用及 immutable Target 的 `GLOBAL_PUBLISHING` / `LISTING` 双重证明。图片、全球创建、本地创建和主动存草稿不进入样本。匹配回执的 `submitted_for_review` 计成功，`publication_failed` 和平台校验失败回执的 `remote_draft` 计失败。回执表承载业务结果，通用 attempt 的 `succeeded` 只证明回执留存。UNKNOWN 单列，进行中、未发送和取消不进分母。零分母显示「暂无已确认结果」，读依赖不可用或超限独立提示。

## 合同和边界

Design Basis 为 Independent Architecture。新增 Listing `GET /api/v1/workbench/supply-preparations/tiktok-publish-metrics`，复用 `workbench.supply.read`、Store 读及成员授权、同源 expected user/org BFF 和单店参数校验。响应复用现有单店统计的 bounded status/counts 形状，只有读取时间和三个计数。响应无业务明细。

Store 从现有本地 owner 读取当前有效 US 跨境店绑定、授权和 app 配置元数据。只返回当前 app、商户、Shop 身份，不解密凭据，不调用 AuthorizedShops 或平台状态。进入和返回前重验 Listing/Store 权限与该身份。孤立验收对既有刊登读也使用测试本地 Store 绑定，避免已有 AuthorizedShops 路径。

Submission OfficialRepository 使用本域窄读范围，按原 intent 的 org/actor/member、当前店/商户/app/Shop、global 引用、已留存回执或结果未知的 attempt 状态、action 和日期筛选，先收窄再按原 attempt 限制 251 条。超过 250 返回 limited。逐条以原 attempt、intent、immutable Target、精确引用的全球创建回执及 US 发布回执核对身份和指纹。用现有 ResolveTikTokPayload 和 CorrelatesGlobalPublication 验证最终请求与回执关系。所选范围内证据缺失或冲突则整个统计 unavailable。最终 intent 和最终回执仍按原 actor/member 精确读取。全球创建回执仅沿该 intent 的不可变 ID 与哈希读取，并验证同企业、店铺、应用、商户、US Shop、商品与草稿关系，保留现有写端允许的跨成员复用。重用已有 read-only repository 和 typed domain validators。数据读取按本次请求，不承诺跨 owner 同一快照。未引入数据库结构、写事务、执行幂等、恢复、计费、旧兼容或对外副作用。

UI 接入同一筛选 query key、取消和加载周期。只读失败允许刷新，迟到响应不混入其他范围。

## 验收

RED/GREEN 覆盖页面单选、日期切换和分类。隔离 PostgreSQL 通过原提交内核与回执存储，Chromium 使用正式 BFF/Go 统计端点验证已提交审核、明确同步失败、校验失败保留草稿、主动草稿、未知及原键回放。用两家 TikTok 测试店与不同日期确认切换，无外部平台请求，并核对原合成预览入口。