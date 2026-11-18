# TikTok Shop US 跨境店消费者订单手动同步

Design Basis: Independent Architecture。Admission: IMPLEMENTATION_READY / FROZEN。

独立准入：/root/architecture_review 已核对本设计初审blob 7c410ed4b45c56a0a59040e7952c9adfc26c25cc及代码47ab10db90ca0842d1c6736443f8c76c20fc753a，无BLOCKER。订单用途/版本和scope、非preparation注入、目录/投影隔离、只读签名POST与读取上限、去敏progress、共享UI的namespace/恢复/终态清理作为IMPLEMENTATION_TEST在同批收敛。

## 用户结果与边界

用户2026-11-09决定：独立本地实例以测试回执跑通手动同步→保存→驾驶舱更新，验证日期/店铺切换、订单去重与覆盖不足。不接真实平台，不改全平台成功率、GMV卡片、合成预览或既有保留实例。本批推进[已批准保存合同](cockpit-tiktok-order-amount.md)的读取接线阶段，仅支持当前Store合同的US CROSS_BORDER，不扩大到本土店、发货/取消/退款、定时采集或财务。交付Draft PR及独立入口，不授权合并/生产部署/父Issue关闭。

## owner与调用链

Store Center继续唯一拥有组织/成员/Store、seller grant和shop binding。Store Observations继续唯一拥有command/sync/checkpoint/head/record。UI复用已有Walmart手动同步组件，增加TikTok平台配置，保留原Walmart storage namespace；共享原key恢复、显式ensure、确认终态清理intent和首页invalidate行为，不新增同步状态owner。

首页明确选单店/日期→原orders/syncs（同key+payload）→原Begin/Starter/worker/Step→TikTokOrderReader→OfficialObservationAccess订单专用短生命周期handle→OfficialClient Orders→原decoder→CommitPage单事务→原saved metrics/BFF。正式非preparation组合注入TikTokOrders；不新增schema。内部命令目录增加TikTok；SHEIN隐式全店及旧列表/详情投影仍限定SHEIN，商品同步不因本批扩大平台范围。

## 身份、权限与凭据

只通过ReadObservation(orders, Sync=true/false)核验成员订单权限，绝不借ProductExecution/product_rules。复用现有TikTok注册、AES-GCM、grant generation/revision、shop binding revision与Store/Connection版本、application key/revision及服务有效期。

Grant active/CanSend且当前US CROSS_BORDER店铺code/id精确匹配；解密token的seller身份hash与grant一致、user_type=0、有效期与当前grant一致，并要求seller.authorization.info和以下之一：seller.order.info或seller.fs.order&fufillment.management。商品scope不代替订单scope。平台读取前以现有AuthorizedShops核对同code/ID/US/CROSS_BORDER，使用新返回cipher，cipher/token不出公开binding、HTTP回执或日志。绑定的ScopesHash为实际sealed token scope排序摘要，不凭本地配置编造。

每页前/后与发布前仍重新核验原Store/成员、grant、binding、connection、服务与token用途；撤销/换绑/刷新版本改变即丢弃本页并停止原任务。复用当前只读handle期限与原30秒activity。TikTok无TokenDetail口，不发明新的在线scope确认服务，也不主动refresh。已保存统计继续使用local-only binding，不解密、不读AuthorizedShops；live/saved身份投影统一UTC表示，但不改变原日期口径或全局binding比较。

## 协议、日期、去重与恢复

按[公开官方订单列表文档](https://partner.tiktokshop.com/docv2/page/get-order-list-202309)与原冻结合同：固定POST /order/202309/orders/search，是只读search。query固定page_size=100、sort_field=create_time、sort_order=ASC、shop_cipher及可选opaque page_token；body仅create_time_ge/lt原窗口Unix秒，不加状态/更新时间/买家/履约过滤。复用现有签名和有界只读HTTP helper，支持具名POST search及2MiB读取上限，不走刊登mutation helper，不自动重试/refresh/跟随跳转，错误去敏。

原窗口48小时，严格[start,end)，不做Walmart扩1秒或SHEIN end-1秒。只取create_time，末秒纳入、下一日午夜排除；日期缺失单列，范围外已知日期拒绝该页。原order.id去重，不展开商品/包裹/组合SKU，不重复乘数量或扣折扣，保留payment.sub_total原币语义。

沿用原100单/页、每窗10000单/512页、总30000页/24h，空/短页有token继续，只有耗尽token且总数一致才发布完整head。重复页/冲突/漂移/超限保守partial，原事务原子发布和同key不同payload冲突不变。公开progress仅允许total/complete，token/hash继续去敏且严格BFF拒绝泄漏。未确认请求按原key查询或原载荷重放，不换新key；终态确认后退休intent，切日期不篡改运行任务。

## 验证与停止点

Threat Model沿原租户/成员/Store授权、密文凭据、原事务/幂等/UNKNOWN；无新增accepted risk或外部mutation。Legacy N/A，无迁移、兼容层、双写或第二事实源。

TDD覆盖固定签名POST/半开秒边界/token/2MiB、orders scope与US店身份/撤权/版本变化、local-only统计、原目录/投影隔离、去敏progress与共享UI原key/终态清理。新的独立PostgreSQL及两家测试店，只注入无网络fallback的HTTP transport，用正式Begin/Step/保存/metrics/BFF/页面验证同期和跨日期记录、缺字段、完整/不完整覆盖及原key重放。保留Walmart/SHEIN现有实例及其数据。开发验证与用户试用确认分开，交付入口/结果/限制，不宣称真实平台验收。
