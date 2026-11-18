# SHEIN US 单店已保存订单商品金额（原币）

Design Basis: Independent Architecture. Admission Status: IMPLEMENTATION_READY。执行 #1163。用户 2026-11-08 批准先明确口径并实现单店只读统计，在独立本地实例用测试订单验日期、店铺、金额缺失和覆盖不足；不叫 GMV、不调用外部平台、不动全平台成功率、GMV 卡片、原合成预览。Figma 仅可参考业务流程，不作 UI 验收标准。

## 用户结果与边界

驾驶舱显式选择一家 SHEIN US 店铺时，在独立面板显示「已保存订单商品金额（原币）」，随昨日/近7天/近30天及店铺变化。数据来源为 Store Observations 已保存消费者订单，不取 Listing attempt、供货成本、采购/ERP 金额或人工经营收入。本批不判断已支付 GMV、净成交额、结算或真实销售覆盖。

复用 operations-cockpit.md、store-center-platform-observations.md、cockpit-listing-task-metrics.md 和当前项目边界；保留现有观察权限、Store/当前商家绑定与同步 owner。本批新增只读投影合同，无新事实、表、状态机、写事务、平台调用或恢复协议。

## 日期、去重、金额和覆盖

- 日期为 UTC+8 已完成自然日，首页日期转 [start 00:00, end次日00:00)，最多30天。按已保存 Order.IssuedAt（原列表 orderCreateTime）筛选，与原同步查询范围一致。详情 orderTime/CreatedAt、更新时间、观察时间不作为入组日期。界面明确「按平台订单列表时间」，不冒充支付时间或消费者下单日期。
- 只读当前组织、当前成员 Store 和 store.orders.read 权限下、当前本地商家绑定匹配的 SHEIN US 订单。沿原观察可见性，不改为仅同步发起人的私有数据。Binding.Site=shein-us，平台为原 owner 合法 SHEIN 绑定；全托管观察原不支持继续不可用。
- 只使用原 owner 的当前完整 head generation，不拼历史批次、不把多个同步版本相加。以 organization+store+sync+platform order ID 唯一记录读取，每订单计一次；同一 generation 如替身返回重复相同记录去重，重复身份载荷不同返回不可用。一个订单的多个 goodsId/包裹不重复加订单总额；订单取消/换货等状态不作额外剔除，因指标仅表达所保存订单的商品金额，不表示支付/有效成交。
- 金额仅来自 Order.Amount.Value（productTotalPrice）和 Currency（orderCurrency），不取 SupplyCost。复用原生十进制约束（非负、最多20位整数和8位小数），Go math/big.Int 按8位定标精确求和，输出规范十进制字符串、至少2位小数，不转 float、不做汇率/币种精度舍入或支付记账。最多250条，输出长度有界；不同3位大写原币分行，绝不跨币相加。
- 当前 head 必须 status=completed、Progress.Incomplete=false、observedAt非空，并且 Range 覆盖整段查询。原同步 Window.End 为闭区间、秒精度，必须 >=查询 exclusiveEnd-1秒。原头/最新批次不同或最新尚非completed，保守显示覆盖不足，不借旧数据填全量。响应前重读 Heads 验证源 ID/revision/generation/binding/range/status 未变化，复验授权、Store目录和原 handle。
- 范围覆盖不足/无同步时无金额和订单总数，提示缺覆盖与原同步范围/取得时间。覆盖完整但期间无订单时明确空结果（不凭空猜 USD 0）。缺 Amount/值、币种或列表时间时，返回 matchedOrderCount（仅日期明确且落入区间）及缺失字段数量，时间未知独立 unknownTimeCount，隐藏所有金额，提示「金额/币种/日期字段不完整」，不把缺失当0。日期缺失的行无法判定所属期间，保守计入时间未知并隐藏金额。
- 新窄 Repository.OrderAmountRecords 仅投影 OrderAmountRecord 的店铺、sync、订单ID、site、IssuedAt和Amount，避免加载商品/包裹正文；按明确 org/store/sync、订单类型及 [IssuedAt起,止) 读取，同时包含列表时间缺失的记录；候选先筛日期再限251。大于250返回limited，提示缩小日期。非法非空时间导致不可读取，不猜日期。绑定、来源或依赖不可证明时 unavailable/原403，均不返回数值。
- GET 无外部平台访问：SHEIN 原 ObservationAccess.Authorize/Check 只验证本地 Store/连接材料和 IAM；不调用 Orders/OrderDetails/Track，不启动 Sync/Ensure。保留原 sync 同一 generation 原子发布和不可变 completed 事实，不增加跨 owner 快照或恢复协议。

## 调用与响应

contract → implementation → injection → consumer：
Store Observations OrderAmountQuery/OrderAmountMetrics → observations.Service.SavedOrderAmounts + 私有精确聚合 → 现有 storeobservations.Repository 的窄读取端口 → 既有 store-observations HTTP/BFF → cockpit-overview 的显式 SHEIN US 单店面板。

GET /api/v1/workbench/store-observations/orders/saved-amount-metrics?startDate=...&endDate=...&storeId=一个UUID。沿既有当前身份、live Store Orders read 授权、服务端 user/org envelope、20秒请求上限和严格 query/BFF/schema。客户端平台不是权限依据。现有 lifecycle 未就绪继续 unavailable。

status 为 complete / missing_fields / coverage_incomplete / limited。共同 capturedAt、来源（可读时 sync ID/取得时间/范围）；complete 含订单数和各币 amount/count；missing_fields 含匹配订单数及缺 Amount/Currency/Time 数量，无 amounts；coverage_incomplete、limited 无金额及完整订单数。缺权限/不可用走原HTTP错误。返回不含订单明细、客户隐私或凭据。

前端复用首页同一 statsQuery / user/org query key / AbortSignal 和读取态，日期/店铺/权限切换不显示旧值；仅显式单选 SHEIN US 时请求/展示，其他店铺或多店隐藏。金额按币种列示，无汇总百分比。GMV卡片和人工经营统计不消费本响应。

## 验证与交付

先 RED：UTC+8 两端、IssuedAt 与 CreatedAt 不同、重复订单/包裹、多同步批次不叠加、两个店铺日期、精确小数/多币、缺金额/币种/时间、完整空结果、同步范围不足/未完成/新head变化、原权限/换绑/依赖失败及候选上限。复用已有单元、隔离 PostgreSQL、BFF/schema 和首页组件测试；只测本次风险，不建设验收平台。

独立本地实例在新端口准备两家 SHEIN US 测试店、不同日期及覆盖范围、原生保存订单和异常字段。正式 GET → BFF → 首页浏览器验证无刷新切换、已知/缺失/不完整提示；无平台/provider客户端。保留既有实例与预览，提供入口。实现完成开一个草稿 PR，合并、正式部署、关闭 #1163 另行授权。Legacy decision: N/A，无迁移/第二事实源。

独立架构复核：设计提交 092642aac62703cab7264bf19f7b3cc67e9322f6 及窄投影补充，由 /root/architecture_review 检查真实 CommitPage/Stop 原子与 terminal 不可变、Heads、SHEIN 本地授权及 HTTP/BFF，结论 IMPLEMENTATION_READY，无 BLOCKER。未知日期计数、SQL先日期后限额/非法日期失败、源切换/撤权/换绑作为 IMPLEMENTATION_TEST 在本批收敛。

## 半托消费者订单同步接线复用

用户后续批准只补齐 SHEIN 半托消费者订单同步接线，并在独立本地实例使用测试回执；本土和全托不纳入此次实现。Design Basis: Reuse Existing Architecture，执行范围见 #1163。复用已批准的 `semi_managed` 应用注册、Store 商家绑定和独立只读观察 handle，经 `order-list` / `order-detail`、原 `Service.Begin/Step` 和 PostgreSQL `CommitPage` 保存，现有单店金额接口和首页消费者保持不变；无新 schema、事实 owner、状态机、权限或恢复协议。

半托商品金额仍只消费 `productTotalPrice/orderCurrency`；`totalCostPrice/saleCurrency` 是独立供货成本，不用于填补商品金额。平台只提供币种、未提供金额时保留已知币种，分别报告缺金额与缺币种；两字段都未知时不制造 Price，明确零值继续保留。订单金额与供货成本的同根因字段丢失一并修正。

有界集成测试 `TestSheinSemiOrdersSyncToSavedAmountsPostgres` 使用正式同步 POST、真实应用注册/凭据保护/签名客户端/响应解析及原 PostgreSQL 事务，外层 HTTP RoundTripper 仅返回离线官方形状回执，不含默认传输或网络拨号。IAM/Store 核验材料为测试替身，工作流启动器在测试内执行原 Step；不将其称为真实商家授权或 Temporal 集群验收。验证当前完整 generation、原命令重放、相邻闭区间窗口重复订单、商品/包裹不重复计价、列表时间与详情时间不同、供货成本不兜底，以及缺字段/覆盖不足。驾驶舱本地实例复用现有验收承载，不新增通用 runner 或验收平台。

本轮不请求外部平台，不改全平台成功率、GMV 卡片或原合成预览。真实半托店的应用/接口权限、实际字段和完整销售覆盖仍需后续单独授权核对；离线通过不代表真实平台或产品验收通过。

## 半托及普通自运营：首页手动同步复用

用户2026-11-09明确批准在驾驶舱首页按所选店铺、完整日期手动同步，保存后金额自动更新；全托保留独立供货单面板。Design Basis: Reuse Existing Architecture，执行#1163，复用本合同、cockpit-shein-self-operated-orders.md及现有StoreOrderSync，不新增API、schema、权限规则、状态机、事务或恢复owner。

SHEIN首次无head金额读取同样执行原Access.Open(orders,Sync=false)及Check，验证当前成员、店铺、应用模式和加密授权业务类型；无来源不表示消费者资格成立。原半托/普通self准入成功且本次读取已完成的金额响应才可显示消费者同步条；刷新、加载及失败期间隐藏控件，保留原回执恢复和终态清理hooks，不自动POST/ensure，不从US标签、供货单not_applicable或请求失败推断资格。全托及非空特殊self业务类型沿原准入拒绝，首页不显示消费者入口；后续POST仍由原Begin/Access独立复验，不以UI作为授权。

共享组件增加独立shein-order-sync namespace。SHEIN时间读取是UTC+8闭区间，所以请求所选起日00:00:00至末日23:59:59；其他平台原半开窗口不变。按orderCreateTime下发日期/原orderNo/productTotalPrice+orderCurrency，原key与日期载荷固定，未确认只查询/重放原键，确认终态清理；完成失效原首页统计查询，当前筛选自动更新。换店/日期和迟到响应继续由原query key及AbortSignal隔离。

独立本地新实例使用半托/普通self两组不同店铺与日期、缺字段、重复商品/包裹及全托/特殊类型测试回执，通过原真实SHEIN签名客户端、正式保存/HTTP/BFF验证首用、日期/店铺、原key去重、金额自动更新及全托面板独立。平台transport无网络fallback，IAM/Store/调度为明确测试替身；真实授权/业务API、生产部署及用户产品验收NOT_RUN。保留原所有实例、全平台成功率、GMV卡片和合成预览。
