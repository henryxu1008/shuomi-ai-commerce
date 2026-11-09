# SHEIN 全托采购单批次只读观察

Design Basis: Independent Architecture. Admission Status: IMPLEMENTATION_READY. 执行 #1163，用户2026-11-09明确批准全托授权域名校验、供货单读取/保存/数量观察，并选择“按采购单号批次”。本批一个Writer、一个主要PR；不自动合并或关闭Issue。

## 产品结果与范围

运营人员在驾驶舱显式单选SHEIN全托店铺，读取/查看一批明确原采购单号的已保存观察；切店时身份、批次、数量和覆盖提示一起更新。页面明确“仅此批单号，不代表店铺全量；不按首页自然日统计”。提供原单数、下单/送货/收货/入仓/次品数量观察，JIT母单单列且不并入非母单数量；数量未知不当0。

用户确认的scope是明确单号集合，不建设时间窗口遍历或整店全量。单次仅一家店铺，最多200个原单号；固定排序/去重后成为命令输入。原始单号字段不接受逗号/控制字符/空白首尾，单号长度<=128。无自然日区间、金额、币种聚合、供货有效需求或结算收入推断。

原 checkpoint 的64KiB预算在准入时按最坏 MissingIDs 重复全量单号计算；供货单进度的紧凑序列化最多60KiB，为JSONB文本空格、原字段及notes保留4KiB。多字节长单号须缩小批次，不能等保存漏返回进度时再失败。

Out of Scope: 真实店铺授权、真实令牌/业务API；消费者订单全托准入；全平台成功率/GMV/原合成预览；ERP/采购/发货执行或地址；定时同步/Webhook；墨西哥特殊全托；迁移、fallback、双写或新同步框架。旧实例不更新，新增独立离线实例与新数据库。

## 业务依据与当前owner

- 用户最新范围 > 本合同 > 当前源码。Figma仅流程参考，不作UI权威。
- [官方授权](https://open.sheincorp.com/zh/documents/system/2169474d-1d4a-41a9-b9fd-427f63f54a63)、[API域名](https://open.sheincorp.com/zh/documents/system/49b797df-016e-40eb-882b-ab9bbaa8c630)：统一SEM.com授权，全托正式API固定openapi.sheincorp.cn；self/semi固定.com。原tempToken/签名/AES协议不变。
- [采购单](https://open.sheincorp.com/zh/documents/apidoc/detail/3002055-1000002)的请求页码pageNumber，响应pageNo；本批只按orderNos集合读，不依赖时区或时间边界。原始价格信息不进入本模型或指标。[JIT关系](https://open.sheincorp.com/zh/documents/apidoc/detail/3001813)是后续需要时的独立能力，本批母单来自原采购单明确类型事实，不因空关系结果推断普通单。
- [Store Observations](store-center-platform-observations.md)、[全托与ERP边界](order-procurement-fulfillment.md)、[当前owner](project-boundaries.md)。Store继续唯一拥有授权凭据；`internal/storecenter/observations`拥有已保存只读平台观察；`internal/integration/shein`拥有DTO解析/签名读取，PostgreSQL adapter拥有原同库事务；ERP不是观察前置，不持有第二观察事实。

## 合同 → 实现 → 注入 → 消费

1. 在原Observations模型新增kind=`supply-orders`和独立SupplyOrder变体，不冒充消费者Order。原单保留ID/type/status/scene/countryMarket及JIT原始分类和数量行；无地址、联系人、原始响应、价格、币种或敏感客户信息。市场不从region/countryMarket推成US。
2. Store独立readonly purpose=`store_supply_orders_read`，复用当前`workbench.store.orders.read/sync`及Store read/assignment、组织/member实时检查；仅注册fully_managed SHEIN可打开该handle，普通self/semi不放行。新purpose不复用fulfillment执行或publish权限；每次Provider请求前后仍重读原cipher/attempt/connection及binding。
3. `OfficialStoreConnectionConfig.validate/prepare`是正式应用唯一配置注入入口，在读secret/网络之前验证Type→APIOrigin固定配对；授权URL保持SEM.com，registry原Type/revision绑定不变。适配器可提供纯校验函数，不新增凭据事实或兼容默认。三模式配置测试使用对应域名，错配拒绝。
4. Provider扩展独立SupplyOrderReader能力；GET原purchase-order-infos，按<=20个已固定单号的小块查询，pageNumber=1/pageSize=200/selectJitMother=1，复用原签名transport；每块只能返回请求集合内身份。响应count/list与集合不符、额外身份、同身份不同载荷、字段结构错误不得伪装完整。重复相同orderNo/载荷只保留一份；未返回身份记缺覆盖，而非无货/取消。块大小保证URL有界，本批不翻任意时间列表。
5. 复用原`Service.Begin/Step`、command、sync、checkpoint、worker和CommitPage，不新增运行状态机或retry owner。SupplyOrderIDs固定进原command输入及checkpoint，原key同规范输入返回原命令/同步，不同输入冲突；一店一批，最多10块。原24h期限及访问失败/重试/恢复入口不变；readonly provider重试没有远端写副作用。
6. 原`store_observations` schema只在全新安装增加kind和payload约束，无新数据库/新schema owner。serving Verify验证当前结构，禁止ALTER/迁移/旧实例安装。原组织+店铺+sync+orderNo键保存，批次版本不相加，原generation/CAS/事务回滚保障继续使用。
7. 新专用supply观察读取使用当前绑定下最新批次。未完成批次不给旧批数字冒充当前；terminal partial可显示其明确已保存部分并标缺覆盖，不能与上次完整head混合。读取以单个MVCC语句逐条消费最多200条原记录，每条沿用schema的256KiB限制，累计器只保留身份集合、计数和精确数量；不因整体大于4MiB拒绝已合法保存的批次，也不一次物化所有SKU行。完成批只表示已尝试该单号集合，始终声明非店铺全量/非原子快照。
8. 原HTTP/BFF严格schema扩展专用supply路由，原tenant/user envelope、live scope、bounded JSON、same-origin/idempotency/CSRF准入继续适用。新增驾驶舱独立面板/子组件按单店取数，日期独立的子query key纳入现有operations-cockpit命名空间及刷新动作，保留AbortSignal；不把自然日参数送给supply API，日期切换不改变该批次含义。原stats、金额和全平台卡片调用保持。

## 数量与JIT

JIT原始标识保留；识别官方“是/否”及固定US语言的Yes/No，其他/缺失归未核实，不猜普通单。母单数量单列，不加到非母单总数/数量；未知分类单列且不并入已确认非母单。子单是独立原单，不按母单合并吞掉。

每行数量使用原orderQuantity/deliveryQuantity/receiptQuantity/storageQuantity/defectiveQuantity的明确非负整数，各字段独立nullable，原0保留。复用精确数量解析，不float求和；累计采用有界整数/十进制字符串。重复SKU行无法证明唯一时该单数量标待核实，不自行按SKU合并或累加。非母单原单数与各数量分开；任一字段缺失，该字段总数不签发完整值并显示缺失行数。包含原保存各状态，明确不是待发/有效需求/已结算数；本批不引入取消/退货净额规则。原type/status/scene未知码可只读保留，未知不是可发货批准。

## 状态、事务、恢复与权限不变量

原Begin在同一PG事务固定命令及子sync；Step按revision CAS提交记录和checkpoint。next index只随成功CommitPage推进；并发旧revision失败不会重复计数。不同块的原单号集合不重叠，expected requested集合不在恢复时重新选取；snapshot只属于其原商家binding。终态completed/partial/failed/suspended和原Ensure/Command/Status负责恢复，重启不换key。provider读超时可按原sync重读，保存失败不推进索引；返回丢失按原key查回执。

访问撤销或Store换绑必须拒绝后续读取和保存。API不接受前端组织/member作为授权事实；revalidation前后原绑定变化不发布页面数字。supply不能从consumer/products路径穿透，其他平台/模式不能使用全托handle。记录及统计只有当前授权店铺/当前绑定可见；缺失/陈旧/partial不伪装0或整店全量。

本批远端只GET，不调用exchange/QueryStore或任何业务API做验收；新增独立本地fixture用无fallback RoundTripper返回官方形状数据，IAM/Store材料为明确测试替身。正式代码可用现有Temporal执行，离线测试直接执行原Step，不把它称为真实IAM/Temporal或真实店铺验收。

## 验证与交付

TDD先复现：域名错配、原单重复、JIT母子、未知分类/缺数量/明确0、漏返回/返回额外身份/身份冲突、原key重放、跨店/换绑/权限失效和partial原批次显示。复用原单位/PG/HTTP/BFF/组件测试，必要范围不建设新runner或故障平台。

独立新PG和两家全托店A/B：A含普通单/JIT母子/原单重复，B含缺返回或partial，切换回A无需刷新；原数值按请求集合去重，母子不叠加，覆盖不足提示。日期按钮不改变supply请求或数字，金额与GMV不接，浏览器所有外部请求为0。保留独立试用入口与记录。交付草稿PR/本地入口/限制，独立最终复核不代替用户产品验收，不擅自合并/关闭Issue/更新旧实例。

独立架构复核：d4430a03bdd368befdd9406e532c1979288f14b3，IMPLEMENTATION_READY，无BLOCKER。准入贯通、latest部分批次、CAS/分块恢复、JIT/缺数量、域名回归为IMPLEMENTATION_TEST，在同批实现验证收敛。
