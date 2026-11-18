# Walmart US 消费者订单手动同步接线

Design Basis: Independent Architecture. Admission: IMPLEMENTATION_READY.

独立准入：2026-11-09 /root/architecture_review 已核对本设计（初审 blob 1a9feeac78bdff9851202bfab02bfc73a6fc38e6）及 main 0685b62e126ebb3eaf296551aea531a896257c30 当前路径，无 BLOCKER。orders scope、内部命令目录与 SHEIN 列表隔离、读取时 2MiB 限制、去敏 progress 严格 schema、Walmart-only 运行配置为同批 IMPLEMENTATION_TEST，按本冻结基线实现。

## 用户结果与范围

2026-11-09 用户决定：独立本地实例使用测试回执跑通「手动同步 → 保存 → 驾驶舱更新」，验证日期、店铺、采购单去重及覆盖不足。此决定推进此前仅保存阶段，不授权连接真实平台。首页选中一家 Walmart US 店铺后，明确点击同步所选完整日期；不自动同步，不扩大到后台定时采集。金额沿用[已批准合同](cockpit-walmart-order-amount.md)，不称 GMV。全平台成功率、GMV 卡片及合成预览不变。

## owner、调用路径与权限

唯一事实 owner 仍是 storecenter/observations，唯一 Store/grant owner 仍是 Store Center。UI → 原 orders/syncs（原 command key、Start/End、单个 Store）→ Service.Begin → 原 Starter/worker → Service.Step → WalmartOrderReader → OfficialObservationAccess 原短生命周期 handle → OfficialClient GET /v3/orders → 原 CommitPage 事务。既有 Repository、head、checkpoint、worker 与数据库 schema 不变。注入仅在正式非 preparation-only 组装中设置 WalmartOrders reader。

同步入口复用原 capabilities 的运行/成员同步权限门禁；它不声称店铺已获得平台 orders scope。真正 Begin、每页读取前后、发布前均按当前 Store/订单用途重新核验。Walmart products 与 orders 分别使用其 purpose；orders 仅接受当前有效 orders view_only/full_access，不能借用 item scope。复用原 application/revision/environment、merchant、grant generation/version、connection、服务有效期、密文凭据保护及 scopesHash。失效、换绑、撤权停止原任务。读取统计仍走 local-only saved binding，不为页面金额读取请求 TokenDetail。

Directory 的 orders 内部命令可见集合允许当前 SHEIN/Walmart；原 SHEIN 列表查询仍限定 SHEIN，不能把 Walmart 行送入 SHEIN 投影。其他平台同步不开放。不存在旧系统兼容、fallback、第二数据源或双写。

## 读取合同与边界

复用已批准 WalmartOrderQuery/Page 与 decoder。固定 US 官方域名、GET /v3/orders，createdStartDate/createdEndDate、三履约分区、replacementInfo=true、limit=30，不过滤状态。日期是首页 UTC+8 完整日期 [start,endExclusive)，原窗口每端扩 1 秒，保存时按原范围过滤毫秒日期。游标仅为后续页请求参数，不跟随 URL/域名；只允许同 /v3/orders 的相对查询串分页形状，不能覆盖本次固定日期/分区/limit；拒绝重复键、未知控制参数、控制字符或超长内容。页体读取最多 2 MiB，原订单256KiB、每PO100行、每分区512页/10000单等既有限额保持。协议错误、游标循环或计数矛盾保守 partial/failed，不冒充全量。

purchaseOrderId 为同店/同 generation 内原单身份，不以 customerOrderId 合并；原事务维持记录、checkpoint、head 原子发布及同键载荷冲突处置。无平台 mutation，不自动换键重试。同步响应丢失先 GET 原 command key；未确认前仅允许原键恢复。原任务 pending/running 可显式 ensure，页面挂载或切换日期不自动 POST。上下文以 org/user/store 隔离，恢复后按原输入核验。切换日期/店铺不改写运行任务输入；完成后失效当前企业的首页统计查询，当前筛选重新读取。

## 验证与交付

TDD 覆盖 GET 参数、游标固定路径与边界、正文上限、orders/item scope 区分、组装注入、命令可见范围与 SHEIN 列表隔离、UI 原键恢复和统计更新。独立本地实例使用新数据库/两店测试回执和 fake HTTP transport，正式 Begin/Step/CommitPage/metrics 路径，无真实凭据或外网 fallback。实机验证同步前后统计、7/30天与店铺切换、重复采购单、缺金额与覆盖不足。CI/代码评审不替代用户验收；交付入口及 Draft PR，不在本轮合并。
