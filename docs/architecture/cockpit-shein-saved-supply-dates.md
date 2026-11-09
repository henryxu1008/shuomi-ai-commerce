# 已保存全托供货单的原始日期观察

Design Basis: Independent Architecture. Admission Status: REVIEW_PENDING. 执行 #1163；用户2026-11-09明确批准仅基于已保存数据的有界日期观察。本合同覆盖本次新增只读派生字段，不重新打开原时间保存合同。

## 用户结果、范围与依据

驾驶舱显式选择SHEIN全托店铺，沿用当前最新已保存批次。首页日期按钮作为原文日期区间，分别展示急采下单日期与备货下发日期落在区间内的原单数、订购数量观察；日期缺失或格式无法确认单列。页面明确“仅当前已保存批次，不代表店铺全量；按原文日期，不作时区转换”。旧批次不补回/混入，也不把partial写成全量。

用户最新范围替代[原批次观察](cockpit-shein-supply-orders.md)中不提供任何日期观察的阶段限制；[原始时间保存](cockpit-shein-supply-business-times.md)的原值、来源、缺失状态及timezone=unconfirmed不变。[当前owner](project-boundaries.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md)继续适用。Figma不作为UI验收权威。

日期含义是用户明确选择的本地观察产品口径，不宣称已确认平台联合时间的查询映射或时区。只比较已保存原串的日历日期，无平台时间查询/翻页、Webhook、真实IAM/店铺/业务API、金额/GMV/全平台成功率、合成预览改动或迁移。不按创建日归集后冒充当日发货/收货量。本批没有新增持久化事实或第二事实源。

## 日期、分类与去重合同

1. `type=1`急采只读`businessTimes.values.addTime`；`type=2`备货只读`allocateTime`。其他type不猜映射，单列采购类型待核实。不用updateTime/request*、母单时间、本地同步时间补日期。
2. 仅原合同可验证的local_datetime或offset_datetime字符串可提取原文YYYY-MM-DD；先用标准库验证格式/日历及原合同一致性，再使用原文年月日。即使带-05:00/Z也不转UTC/UTC+8、不改变日历日。时区未确认仍可观察原文日期，这不等于已确认真实事件或平台自然日。
3. missing/null/empty/unrecognized/unexpected_type均进入日期待核实列表，保留原单ID、应取字段和原状态；不trim、不把数字猜成epoch、不把待核实算进任一日期或当0。1970等格式合法原串不擅自设哨兵/判缺失，仍按原文日期表达，保留原含义未核实提示，不宣称事件已发生。
4. 分类优先级互斥：明确JIT母单→母单单列；母子标识未知→母子待核实；非母单但采购type未知→采购类型待核实；非母单且type已知→日期可读且区间内的急采/备货，或日期待核实，或区间外。上述ID集合构成已保存ID集合的完整无重复分区。缺日期列表在所有日期区间都可见，不伪造日期归属。
5. 原单唯一身份沿用org+store+sync+orderNo及原读取的固定集合，重复响应和时间冲突继续由原保存合同处理。每批最多200原单，读取重复/缺失/越界身份整体拒绝，不在统计时自行挑一个版本。JIT母单不叠加；子单作为自身原单按自身字段归属，母子待核实不计入两组。
6. 两组订购数量复用原supplySummaryAccumulator的精确整数、重复SKU/缺数量处理，只取ordered观察值；不新增浮点/金额计算。0个符合原单时显示0单/0订购量；有原单但订购数量缺失时显示待核实，不能补0。数量是这些原单当前已保存的订购数量，非当日发生的履约增量。

## owner、调用与UI

原`internal/storecenter/observations`拥有只读派生计算；现有PG VisitSupplyOrders仍只遍历同一当前绑定/同一sync，记录和全批数量/日期投影在同一次遍历中生成。复用前后head/revision/fullSync/handle.Check复验，不增加查询库、缓存或持久化投影，不调用provider。

原GET supply-orders/observation保留明确无日期的批次观察模式，扩展可选但必须成对的startDate/endDate，和storeId之外不接受其他参数。日期为严格YYYY-MM-DD、真实日历日期、起点<=终点，起止日期均包含；这是本地原文日期筛选合同，不冒充平台端点保证。区间不枚举逐日，固定最多200原单，不人为要求平台历史可查范围或当前日期。不能复用会固定UTC+8再转UTC的workbenchmetrics.Parse。

返回可选originalDates仅在已保存terminal批次且请求显式日期区间时存在，包含range、urgent/stock的ID及ordered数量、undated、motherIds/unknownMotherIds/unknownPurchaseTypeIds/outsideRangeIds。running/失败/未知/unsupported不给缓存投影冒充当前。现有无日期批次读取仍是当前有效用例，不是旧Service/fallback。

HTTP只解析结构，业务范围/日期分类在domain。BFF严格校验日期/店铺绑定、响应range相等、各分区唯一且总集合与savedOrderIds一致，并沿用tenant/user/no-store/响应预算。每条未定日期只带有界ID/字段/状态，不复制原始响应/大量时间正文，原值详情仍由原已保存Detail提供。

SheinSupplyOrders用同一个请求/缓存键包含scope/store/startDate/endDate，日期、店铺变化同批元数据和日期投影一起更新；保留AbortSignal，加载/失败隐藏旧事实。原全批数量如保留须明确标题“本批全部数量（不按日期筛选）”，日期两组单独显示，避免把全批12误读为筛选后的值。复用现有Card/数量组件与原详情，不改全平台卡片/GMV/合成预览。无新的跨租户、权限、状态、事务或恢复边界。

## 验证与交付

独立设计检查冻结后TDD：local/offset/Z原日不变、跨月/闰日、起止包含、反向/非法/单端区间；type1/2字段选择、缺失/空值/空串/空白/数字等待核实；1970不私设哨兵；重复原单、JIT母子、未知分类、精确数量与partial；跨店/权限/换绑/并发head变化拒绝，BFF范围及分区篡改拒绝，UI切日期/店铺/刷新隐藏旧值。复用既有单位/PG/HTTP/BFF/组件测试，不新增runner。

独立本地实例复用原750fb728... PG/shein_supply_times_browser、店铺和6条原回执，前后固定原记录指纹。原A：11月1日急采PO-A 1单/5订购，子单缺日期单列、母单100不叠加；原B：格式未识别与母子待核实、partial提示保留，旧99不混入。原批无备货日期正例，在隔离且明确标记的补充店铺/批次中由正式adapter→Step→PG保存一条type2回执，addTime与allocateTime不同日且带偏移，用于备货分支，不改写原6条。浏览器切原日期/店铺实测，补充证据与原回执证据分开，外部平台调用0。

交付一个Draft PR和保留本地入口；不自动合并、关闭Issue或签发用户产品验收。真实店铺与平台全量日期合同仍是NOT_RUN，不阻止本次有界已保存观察。
