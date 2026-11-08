# 驾驶舱 · Walmart Feed 提交受理率

状态 `IMPLEMENTATION_READY`。执行 [#1163](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1163)，用户批准先接 Walmart 单店指标及隔离本地实机验证。

## 产品范围

运营人员在 `/workbench` 显式选中一家 Walmart US 店铺时，现有指标下方单独显示「Walmart Feed 提交受理率」。日期、店铺与其余统计使用同一轮筛选。平台筛选、全部店铺或其他平台店铺不展示该指标。沿用 [首页统计设计](cockpit-listing-task-metrics.md) 的 UTC+8 完整日窗口、当前用户与店铺授权及 [Walmart Feed 合同](walmart-feed-recovery.md)。采用当前驾驶舱组件。Figma 仅供业务流程核对。

统计单位为期间创建的 `walmart_submit_feed` 原 attempt，范围包含经用户明确确认的新 Feed attempt 及重新提交的 Feed。按 `organization_id, attempt_id` 去重，日期使用 attempt 的 `created_at`。已受理须有同 attempt、同 intent、同绑定的有效 Feed 回执。明确拒绝须由同 attempt 的 Walmart 授权拒绝证据支持。结果未知单列，进行中、未发送和取消不计入分母。受理率为已受理除以已受理与明确拒绝之和。分母为零显示「暂无已确认结果」，读不到或超限单独提示。SKU 处理结果、平台审核、全平台卡片、GMV、合成预览、执行与恢复、真实平台调用和正式部署在范围外。

## 合同与边界

Design Basis 为 Independent Architecture，新增单店聚合读取合同。Listing 提供 `GET /api/v1/workbench/supply-preparations/walmart-feed-metrics`，沿现有 Listing HTTP admission、`workbench.supply.read`、工作台 BFF 和 expected user/org 校验。查询沿用 `startDate`、`endDate`、`storeId`，只接受一个店铺。结果仅有读取时间、complete/limited 与 complete 时的 accepted/rejected/unknown 计数。无回执、商品、成员或凭据明细。

Listing 通过现有 Store 授权校验 org/actor/member 和当前 Store grant。Walmart 商户归属读取 Store owner 保存的当前连接、授权和服务范围，通过现有 Store 本地 material read，使用当前 app identity 配置，复用 Walmart 绑定构造。不打开凭据，不读取 TokenDetail，不访问 Walmart。进入与返回前重验该 Store 绑定和 Listing 权限。旧商户记录不进入当前店铺范围。授权、读取或证据依赖异常按现有 unavailable/forbidden 处理。

Submission 的 OfficialRepository 新增有界候选读取，关联原 intent 的 org/actor/member，只选当前店铺、Walmart action、publish 和日期窗口。最多读取 251 个唯一候选，超过 250 返回 limited。通过既有 attempt、intent、原 immutable TargetRecord 以及 receipt reader 验证 owner、原记录、指纹和完整绑定。复用 Walmart 持久化拒绝的证据核对，将通用校验从仅修复场景的谓词中提取并保留原行为。读取时间取请求开始，结果按读取时已提交的状态分类。独立汇总不承诺与其他 owner 的全局时间点一致。

UI 将显式单选 Walmart 店铺的查询加入现有筛选 query key、AbortSignal 和结果加载周期，当前范围以外的迟到响应不渲染。维持现有 Listing 与任务的响应结构。新路径不访问 SKU observation，不触发外部平台、计费、发送或补偿。不引入 schema、索引、写事务、新持久事实或旧数据兼容层。幂等、重试、UNKNOWN 和恢复继续由原 Submission owner 负责。只读失败可重新读取，部分结果不显示为完整统计。独立复核通过的设计为 ce177c88075d20fdd2b181aad13ffb8b430b31b5。当前商户身份和环境先于候选上限缩小范围，并以类型化数据复验。独立验收对既有刊登统计与新增指标都使用本地 Store 测试绑定，避免既有刊登统计的 TokenDetail 路径触发外部调用。

## 验证

- 本地 PostgreSQL 复用原 attempt、intent、Feed 回执和拒绝写入逻辑，覆盖成功、明确拒绝、结果未知、原键回放、同目标新 attempt、日期边界、成员和店铺范围、零分母及失败与不可用。
- 用 Chromium 在独立验收实例切换日期、Walmart 店铺和其他店铺。核对同一浏览器文档上的请求和显示，验证全平台上传成功率卡片和 GMV 原样。现有合成预览另行保留。