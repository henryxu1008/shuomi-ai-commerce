# TikTok Shop US 消费者订单保存与单店商品金额

Design Basis: Independent Architecture。Admission Status: IMPLEMENTATION_READY / FROZEN。执行 #1163，用户2026-11-08授权先确定消费者订单读取、保存、日期、去重、商品金额及覆盖合同，再在独立本地实例用测试订单验证。Figma仅辅助业务流程。

## 用户结果、阶段与范围

运营人员在新的独立本地实例显式选择一家当前支持的TikTok Shop US跨境店，可按首页昨日/近7天/近30天查看「已保存订单商品金额（原币）」。测试订单经过官方形状的离线decoder、原Store Observations Begin/Step/CommitPage、真实PostgreSQL、正式统计GET/BFF和首页。当前Store仅支持US CROSS_BORDER绑定，不宣称覆盖US本土店。

本批为来源准备和离线验证。正式组合不注入联网TikTokOrderReader，不开订单同步按钮，不改变原SHEIN订单管理目录。外部平台、真实店铺/业务数据、全平台成功率、GMV卡片、合成预览、其他保留实例、订单发货/取消/退款、财务结算、迁移、正式部署、合并及Issue关闭均在范围外。用户试用确认与开发自检分开。

## 官方依据与固定口径

- [Get Order List 202309](https://partner.tiktokshop.com/docv2/page/get-order-list-202309)：POST /order/202309/orders/search；create_time_ge包含下界、create_time_lt排除上界，Unix秒；使用shop_cipher，page_token/next_page_token及total_count；page_size最多100。接口可附加状态/更新时间/买家/履约过滤，本合同不设置这些过滤，固定sort_field=create_time、sort_order=ASC和原创建时间范围。在线读取未来必须单独证明seller.order.info或seller.fs.order&fufillment.management，刊登权限不替代订单权限。
- [Get Order Detail 202507](https://partner.tiktokshop.com/docv2/page/get-order-detail-202507)的公开“下载Markdown”字段说明：payment.sub_total是订单全部SKU商品小计，美国为税前，已反映seller_discount/platform_discount；与含运费/税费等的total_amount不同。列表示例已经含payment/currency/sub_total，本批只读列表形状，不新增详情调用。
- 直接保存原payment.sub_total和payment.currency，不再次扣折扣、乘数量、展开line_items/sub_item_info/combined_listing_skus/包裹，不以total_amount、original_total_product_price或刊登售价兜底。不加入税运费、手续费、供货金额、退款或结算款。保留所有原返回订单的状态/类型，不依据取消/未付款等状态推断已支付或净成交；样品/换货等不另行臆造排除范围。页面解释这是保存的原商品小计，不是GMV或净收入。

## 原生事实、日期和去重

唯一owner为internal/storecenter/observations；新TikTokOrder不塞进SHEIN Order或WalmartOrder，不由integration引用Marketplace业务root。allowlist只有id/site、原create_time归一的UTC RFC3339秒级Date、status/order_type和原商品小计/币种。客户姓名/电话/地址/邮箱/支付卡/买家留言/物流单号/原raw response不入记录或日志；未知字段忽略。金额原十进制字符串或JSON数字精确读取，不经float，非负、最多20位整数和8位小数；明确0与缺失不同，非法非空值拒绝。币种3位大写，不猜USD。缺create_time保留未知，非法非空时间拒绝。

日期只取订单create_time，不取paid_time/update_time/抓取时间。首页沿UTC+8完整自然日[start,end)，最多30天。原OrderWindows按48小时分块，相邻端点共享但各次查询严格[start,end)，不扩大读取；原sync.Range.End须达到统计exclusiveEnd，不能借用SHEIN的end-1秒覆盖证明。窗口尾前1秒纳入，窗口端点仅属于后窗，统计次日00:00排除；范围外的已知日期返回视为来源不符合固定查询合同，拒绝该页，不悄悄当作完整覆盖。

保存身份organization/store/generation/order.id。一个订单多个SKU、包裹、组合商品仍只算一单原sub_total；不同order.id分别计，历史generation不累加。原CommitPage的同sync行锁事务内比较规范typed载荷：相同ID同载荷保留首次payload/observedAt，仅原window_key/page_index作首次位置；同窗跨页重复标sticky incomplete（防止重复页掩盖缺订单），载荷冲突保留原值并sticky incomplete，不静默覆盖。原命令键/载荷哈希幂等和原Step恢复责任不变，无第二写入/恢复owner。

扩展原Record.TikTokOrder和records JSON variant，SHEIN/Walmart/TikTok订单及商品variant严格互斥，site/id匹配。仅修改当前fresh-install schema及strict verifier快照，使用新的独立PostgreSQL；不ALTER、迁移、重建或重播种任何保留实例，无新表/双读双写/legacy adapter。

## 分页、覆盖与事务边界

TikTokOrderQuery携带固定原Window.Start/End及上一页token；TikTokOrderPage含typed订单、必须存在的非负total_count和next_page_token。decoder接受有界官方code=0/data.orders/total_count形状，orders必须为数组，每页<=100、raw<=2MiB，token<=8192且无控制符；缺/空token表示末页，null/错误类型拒绝。单页重复ID拒绝；缺字段的有效订单保留为待核实。

复用原Checkpoint增加TikTokOrderProgress（token/hash列表、该窗首个total_count、Complete）。原Windows各窗依次遍历；短页/空页有token必须继续，只有token耗尽才结束该窗。每窗total_count须一致且末页原始累计条数相等，循环token、重复页、载荷冲突、总量漂移或上限均为partial，不发布完整head。每窗最多10000单/512页，总30000页/24小时、原30秒activity限制不变，不新增调度/自动拆窗机制。token/hash只保存在原checkpoint，Sync.MarshalJSON清除，clone深拷贝包括Total指针。

Service.Step只在TikTok Orders分支消费显式注入的离线reader；返回后原Access.Authorize、Merchant.Check和binding复验，再由原CommitPage同事务保存records/checkpoint/status/head。未注入reader不可采集。正式Directory/OfficialObservationAccess仍不开放TikTok在线订单。完整head发布必须有TikTok Complete且全部Windows耗尽；若事务发现incomplete清除Complete。保存/响应丢失仍按原sync/revision恢复，不发新键、无外部mutation/UNKNOWN resend。

## 本地授权与统计路径

路径：verified组织/actor/member + store.orders.read → 原Store ReadObservation(orders, Sync=false) → 本地TikTok grant/app/shop绑定投影 → 当前heads/latest → 原records.TikTokOrder窄日期投影 → 原精确金额聚合 → 重读来源/授权/本地绑定 → 正式GET/BFF → 单店面板。不得借用ReadProductExecution/product_rules，读取保存数据不解密credential、不查AuthorizedShops、不refresh、不调用任一provider。

本地ReadSavedTikTokOrderBinding复用ReadObservation的当前Store/成员授权与Store事实：grant active且有效、组织/app key/revision一致，shop为同一US CROSS_BORDER、binding active、grant ID/generation/revision、binding revision、StoreVersion/ConnectionRevision、服务/凭据到期均匹配。返回现有Binding的本地身份投影，ScopesHash留空，不伪造已授予的线上订单scope。统计要求捕获head的Binding.ValidTikTokApplication成立；比较该绑定与本地投影时仅排除ScopesHash（它属于捕获时在线scope证明，本地读取不声明其当前有效）；其余字段须全等。grant generation/revision变化仍使统计不可用。未来联网reader须另外验证真实订单scope并把其证明摘要放进捕获Binding，当前离线fixture仅有测试scope，不证明平台权限。

统计只使用当前完整head，head/latest为同一个完整sync，ObservedAt有效、Progress.Incomplete=false、TikTok Complete及Range覆盖查询。先按已保存Date过滤再LIMIT251，缺日期保留以提示，最多250订单；非法非空时间不可用。复用OrderAmountMetrics与原math/big.Int聚合，按订单原币分别相加，不换汇。缺金额/币种/时间任一项则隐藏全部金额，给对应订单数量；coverage_incomplete/limited不返回完整合计。完整空集合显示无记录，不猜币种填0。响应前复验同一来源和本地授权/绑定，迟到响应不串店。

新增GET /api/v1/workbench/store-observations/orders/tiktok-saved-amount-metrics，沿原单店日期校验、20秒、身份envelope/no-store及BFF。仅显式选择TikTok Shop US单店展示，复用已有独立金额面板结构，不改变其他平台、全平台成功率/GMV或合成预览。无新公共通用平台、缓存事实或权限体系。

## 验证、威胁模型和停止点

Threat Model沿当前verified身份、租户/成员/Store授权、原单DB事务、幂等及凭据保护，不扩大到线上采集。Accepted Risks无新增；Legacy decision N/A。必须覆盖当前范围的跨租户、撤权/换绑/来源漂移、原PII不入保存、金额精度及不完整覆盖；不建设新的验收runner或故障平台。

先RED/GREEN验证decoder秒级日期/缺字段/精确金额、订单多行/包裹不乘、原ID重复/冲突/旧generation、分页短页/空页有token/循环/总数漂移/上限、token不出HTTP、原PG事务与fresh schema/跨租户/日期先筛后限、精确末秒与次日边界、本地grant授权不访问provider。只运行相关必要测试，最终稳定候选走原CI并做一次独立交付检查。

新的独立本地实例准备两家US跨境测试店、两组日期订单、同ID重放/旧批次、多商品/包裹原小计、缺金额/币种及范围覆盖不足，经离线reader→原正式保存→GET/BFF→首页切换，无刷新实机验证。网络拦截，其他实例不动。交付入口/结果/限制供用户试用；不擅自合并/部署/关闭Issue。

独立架构复核：/root/architecture_review实际核对设计a0d7e9078ebe2c190856c0e128c6095e13d41fcd，结论IMPLEMENTATION_READY，无BLOCKER。原CommitPage仅对TikTok orders放行100单，31/100可保存、101拒绝，其他平台保持30；ScopesHash仅专用saved-TikTok比较排除，全局ValidTikTokApplication不变。上述边界、token/总数/重复页与原子head、秒级半开日期、clone/去敏、PII/金额和绑定漂移均为IMPLEMENTATION_TEST，在本批实现验证收敛。
