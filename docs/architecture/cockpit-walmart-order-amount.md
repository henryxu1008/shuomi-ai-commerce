# Walmart US 消费者订单保存与单店商品金额

Design Basis: Independent Architecture。Admission Status: NOT_READY / pending independent review。执行 #1163，用户 2026-11-08 授权先确定读取、保存、日期、去重、金额及覆盖合同，再在独立本地实例用测试订单验证。Figma仅辅助业务流程。

## 用户结果与阶段范围

运营人员在独立本地实例显式选一家 Walmart US 店铺，可按首页昨日/近7天/近30天查看「已保存订单商品金额（原币）」；不称GMV。测试订单经过 Walmart 官方形状的离线回执解析、原 Store Observations 保存事务、正式统计HTTP/BFF和首页。原观察 owner 内增加 Walmart 原生订单投影，不塞进 SHEIN Order。

本批是订单来源准备与离线验证，不启用真实店铺在线订单同步、不注入可联网订单 reader、不新增订单同步按钮；通用订单管理页面仍为原 SHEIN 范围。正式平台授权/采集接线和真实业务验收后续单独安排。全平台成功率、GMV卡片、原合成预览、其他平台、订单发货/确认/取消/退款、资金/结算、客户隐私、迁移、正式部署、合并和Issue关闭均不在本批。

## 官方依据与数据意义

- [US All orders API](https://developer.walmart.com/us-marketplace/reference/getallorders)：createdStartDate/createdEndDate按订单创建范围，最大10,000订单，最近180天；默认SellerFulfilled，另有WFSFulfilled和3PLFulfilled。一个customerOrderId可对应多个purchaseOrderId。
- [All orders示例](https://developer.walmart.com/us-marketplace/docs/get-all-orders)：订单列表已含orderLines/charges，meta.nextCursor是后续页信号；不使用released接口（只含已释放Created订单）。
- [Orders概览](https://developer.walmart.com/us-marketplace/docs/order-management-api-overview)：PRODUCT表示该订单行的商品费用，SHIPPING为运费。
- [Token details](https://developer.walmart.com/us-marketplace/docs/retrieve-access-token-details)：在线订单读取须检查orders的view_only/full_access；item权限不等于orders权限。本批不新增在线凭据或伪造scope，离线reader不能证明真实权限。

## 日期、身份与金额合同

1. 日期取原orderDate（epoch milliseconds，归一为UTC RFC3339），不是statusDate/发货/保存时间。首页沿UTC+8完整自然日[start,end)，最多30天。缺orderDate保留为时间未知，非法非空值拒绝。查询时间固定在原sync.Range，原OrderWindows使用闭秒窗口；相邻窗口重叠按身份去重，不能漏午夜边界。原平台日期边界包含性未通过真实调用确认，未来在线适配须扩大边界读取后按本地orderDate精确筛选，不能推测完整。
2. 保存身份为organization/store/generation/purchaseOrderId；customerOrderId只用于说明拆单，不用来合并不同PO。订单行按lineNumber唯一；相同SKU不同lineNumber分别保留，包裹/状态分组/数量不生成额外金额。重复同PO相同载荷跨页/跨窗口只保存一次；不同载荷不得静默覆盖，记录coverage冲突并禁止发布完整head。不同generation不累加。
3. WalmartOrder仅允许purchaseOrderId、customerOrderId、orderDate、orderType，以及lineNumber、quantity、PRODUCT费用列表（currency/原十进制amount）。不存地址、姓名、邮箱、电话、trackingURL、支付资料、raw response或凭据。订单身份/行集合必须有效；空行集合或字段缺失不能被当成零元。
4. 直接累计每个原生订单行的PRODUCT chargeAmount，**不再次乘orderLineQuantity**；指标表达原API保存的PRODUCT费用合计。不加入SHIPPING、tax、FEE、退款、供货/采购金额，不从刊登售价推算；不另行反推折扣前/后成交。每行恰有一个PRODUCT费用才进入金额；缺失、多个PRODUCT或显式折扣标记无法解释时保留缺字段/待核实并隐藏合计。原始quantity留存但不参与费用乘法；虚拟包不按pack元数据展开。保留所有原订单类型及状态所对应原费用，不依据状态扣减，不声称已付款或净收入。
5. 金额为非负十进制（最多20位整数、8位小数）；json.Number/原字符串精确读取，不经float；各3位大写原币分别累加。同一PO的所有PRODUCT费用必须同币，混币PO标字段待核实并隐藏金额，不拼成虚假订单总额；不同PO可分原币展示。复用SHEIN已有math/big.Int精确聚合方式。明确0和缺失不同；期间完整空集合不猜币种或填USD0。

## 读取、分页和覆盖

WalmartOrderQuery固定原Window、履约分区和上一页游标；WalmartOrderPage为allowlisted typed orders、reported total、nextCursor。离线decoder只接受有界官方JSON形状，拒绝结构错误、重复行/身份、超出每页30单、超过2MiB或单单256KiB；未知业务字段不透传。全流程只读，没有外部mutation或UNKNOWN resend。

固定履约范围SellerFulfilled/WFSFulfilled/3PLFulfilled三个分区。每个原时间窗的三个分区均须遍历完成；不把默认卖家自发货冒充全店。缺分区、count漂移、游标循环、超限、重复身份冲突、未完成同步保留原owner的partial/failed状态及原因，无完整金额。replacementInfo覆盖是固定合同的一部分，保留REPLACEMENT/PREORDER而不悄悄漏数；真实adapter尚未接线，不能据离线fixture宣称平台实际覆盖。

复用原Checkpoint JSON，在其中保存Walmart分区、nextCursor和有界游标hash；游标不出HTTP/日志。nextCursor缺省或明确空字符串代表该分区末页，不能以短页/累计条数代替；存在游标必须继续，包括空页。reportedCount仅辅助完整性，前后变化或页数耗尽保守partial。每分区最多10,000条/512页、cursor长度8192，原总30,000页/24小时和30秒activity限制继续生效；到限额partial。本批不新增自动拆分/采集调度器。

原Service.Step在Walmart Orders分支消费显式注入的WalmartOrderReader，返回前复验原Access授权/handle/binding；同一原Repository.CommitPage事务保存records/checkpoint/status/head。正式组合不注入此reader，Walmart Orders原Access/目录同步入口保持未开放；独立fixture仅注入离线reader，使用原Begin、Step和CommitPage，不建设第二runner或数据库owner。

## owner、持久化、授权和读取路径

唯一owner为internal/storecenter/observations，平台订单规则为internal/marketplace/walmart；官方形状decoder在internal/integration/walmart。扩展Record.WalmartOrder及原records JSON variant，SQL约束明确Walmart site/id与SHEIN variant互斥，不增加表或迁移/双读。修改当前全新安装schema及其严格verifier快照；不对任何保留数据库ALTER/重建。测试用新独立PostgreSQL。

只读单店统计路径：当前verified组织/actor/member + store.orders.read → 原Store本地ReadObservation(orders)及本地Walmart grant/application/binding → 当前heads/latest → 原records.WalmartOrder窄日期投影 → 精确聚合 → 重读来源/授权/本地binding → 正式HTTP/BFF → Walmart单店独立面板。本地已保存读取不调用TokenDetail、Merchant、Orders、refresh或任一provider；不借用product_publish/rules权限。grant必须active、未过期，匹配组织/店铺/应用/环境/商家、generation/version/StoreVersion/ConnectionRevision；当前Store/成员授权由原ReadObservation落实，不建立第二授权体系。

统计仅当前完整head，head/latest为同一完整sync、ObservedAt有效、Progress.Incomplete=false、Range覆盖所选完整日期，且Walmart三个分区确有原checkpoint完成标记；快照深拷贝保留nil/empty语义，响应前复验。候选先按日期过滤再限251（缺时间保留），最多250订单；非法非空时间不可用。金额/币种/日期/PRODUCT费用歧义任一缺失则隐藏全部金额并显示相应计数；覆盖不足或limited不返回金额/完整订单数。

复用OrderAmountMetrics的complete/missing_fields/coverage_incomplete/limited形状，Walmart缺费用歧义另有明确missing提示计数；SHEIN形状与行为不变。新增 GET /api/v1/workbench/store-observations/orders/walmart-saved-amount-metrics，严格单店/日期、20秒、现有身份envelope/no-store/BFF。只在显式Walmart US单店展示，日期/店铺/企业变化取消旧请求、不留旧值。

## 验证和准入

先RED再最小实现：原生JSON数字/字符串金额、小数精度、数量不重复相乘、PO与customerOrder拆分、多行同SKU、跨页/窗口重放及冲突、跨generation、午夜日期边界、缺金额/币种/日期及PRODUCT歧义；三履约分区、短页有cursor、空页有cursor、循环/超限、源漂移/撤权/换绑；原SQL事务及严格schema、跨租户、head与checkpoint原子发布。只运行与本批直接相关既有测试，最终CI按仓库规则。

独立本地实例准备两家Walmart US测试店、不同日期/履约分区、多行/数量及同customerOrder多PO、旧批次与重放、缺字段与不完整覆盖，离线reader→正式保存→统计GET/BFF→浏览器首页无刷新复测；全程网络拦截，其他保留实例保持不变。实现完成独立最终检查，交入口/数据/限制供用户确认；不自行签发用户验收或合并。Legacy decision: N/A。
