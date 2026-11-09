# SHEIN 全托业务时间原值保存

Design Basis: Independent Architecture. Admission Status: REVIEW_PENDING. 执行 #1163，用户2026-11-09批准本批，单Writer、单主要PR。

## 产品结果与依据

运营人员选择SHEIN全托店铺，在原供货单批次中选择已保存原单，查看时间原值、字段来源及缺失状态。仅保存/查看，不提供日期筛选、自然日汇总或金额。全平台成功率、GMV、合成预览、保留实例不改；不连接真实店铺/交换令牌/调用业务API。

复用[供货单观察](cockpit-shein-supply-orders.md)、[Store Observations](store-center-platform-observations.md)及全新系统产品基线。Figma不作为UI权威。[官方采购单](https://open.sheincorp.com/documents/apidoc/detail/3002055-1000002)、[时间FAQ55](https://open.sheincorp.com/documents/faq-detail/55)、[组合FAQ48](https://open.sheincorp.com/documents/faq-detail/48)已于2026-11-09公开只读核对，证据在#1163。采购单时区、边界包含性未确认；消费者订单UTC+8与签名毫秒时间戳均不是本合同依据。

## 唯一事实owner与保存合同

`internal/storecenter/observations`继续唯一持有只读观察，`internal/integration/shein`解析官方DTO，原PG records payload的supplyOrder内新增必需`businessTimes`，不新增表、索引、owner或第二事实源。所有新回执必须显式生成完整集合；读取缺少此合同的记录拒绝，不为旧数据补值/迁移。原SQL变体限制和256KiB记录预算不放宽，应用Valid负责这个嵌套合同，保持原schema/catalog约束不变。

bundle固定`contract=shein-supply-business-times/v1`，`source=/open-api/order/purchase-order-infos`，`timezone=unconfirmed`。`values`是严格固定13键对象，键即原字段来源（每个值不另复制来源）：addTime、allocateTime、updateTime、requestDeliveryTime、requestReceiptTime、requestTakeParcelTime、reserveTime、receiptTime、checkTime、storageTime、returnTime、requestCompleteTime、deliveryTime。请求边界combineTime不是返回事实，不发明该字段。

每个值保存`state`、`rawType`、`raw`。raw是JSON解码后的原字符串，保留空白、原格式和显式偏移；数字保留原词法文本，布尔保留true/false，不解释为epoch或时间。missing/null的raw为null且rawType分别missing/null；空串是state=empty/rawType=string/raw=""。字符串仅用标准库检查日历与固定格式：无偏移年月日时分秒为local_datetime，RFC3339为offset_datetime，其他为unrecognized；不trim。数字/布尔为unexpected_type，不静默丢失。每字段raw最多256 UTF8字节，拒绝NUL及非法UTF8（PG JSONB不可存），其余合法原文保留。对象/数组、超长原文或不可保存文本拒绝整页，原事务不推进；不截断、不伪装missing。值的状态/类型/原文一致性由domain Valid校验。

平台时区一律unconfirmed，即使原文带偏移也只保存原文，绝不转换/输出time.Time、UTC/epoch、业务日期或默认时区。识别格式不等于确认事件发生。1970样例不自动解释为真实时间、0或缺失。要求/截止字段不等同实际事件。同步startedAt/observedAt仍只是本地观察元数据，绝不填充缺失原单时间；重复同步保持missing/null。

## 调用、权限与事务

合同 → 原OfficialClient.QueryObservedSupplyOrders捕获固定字段presence/null/原类型 → 原Store handle与reader注入 → 原Step/CommitPage保存 → 原只读Detail HTTP/BFF → 驾驶舱原单折叠详情。既有数量、母子分类与coverage不变。typed原单去重包括时间bundle；相同原单与时间去重，冲突时间沿用原身份冲突/缺覆盖规则，不自行选择最后一条。

原org+store+kind+sync+orderNo主键、绑定、原key、CAS及同PG事务不变；adapter失败不提交整页，提交失败不推进checkpoint；未知响应仍由原Command/Ensure恢复。只GET无外部写，无新状态机/恢复协议/授权方式。

开放supply原已存在的Detail路径（此前不注册），只读已保存记录，不调用远端详情。Service.Detail复验scope、sync/store/kind和原merchant binding，读取后Check；另校验supply变体及Valid。BFF严格解析仅该确切detail路径及完整typed DTO，校验tenant/user、store/sync/record身份，不开放supply列表/物流/任意路径。沿用no-store与现有限流/2MiB响应预算。只返回单原单，避免200单时间扩大首页DTO。组件按scope/store/sync/order key隔离和取消请求，刷新/报错/权限不明期间隐藏旧事实，换店/换批清空原单选择。不从本地时钟造时间事实。

## 验证与交付

TDD覆盖missing/null/empty/空白、无偏移/显式偏移/非法日历、1970、数字/布尔、超长/复合结构、原值PG往返、重复与时间冲突、跨店/绑定错误和原数量/coverage。复用现有测试/原离线fixture，不扩展通用runner。新建独立本地PG及两家全托测试店，新schema空数据；原35726及其他预览不改。用无fallback传输替身走正式parser/Step/PG/HTTP/BFF，浏览器查看13字段、切换店铺/原单，确认缺失不补同步时间，日期切换不改变该批事实，外部业务请求0。开发自检和代码复核不等于用户产品验收。

交付Draft PR和保留试用入口；不自动合并/关闭Issue。日期统计、真实时区/查询边界确认、金额均明确后续，不把本批宣称自然日可用。
