# Temu US 本土与半托管消费者订单手动同步

Design Basis: Independent Architecture。Admission: PENDING_REVIEW。执行 #1163。

## 用户结果、依据与范围

用户2026-11-09决定：本土店和半托管一起接线，在新的独立本地实例用测试回执完成「手动同步→保存→驾驶舱更新」，验证日期、店铺、父订单去重、缺金额与覆盖不足。引用已冻结的 [Temu订单保存合同](cockpit-temu-order-amount.md)、当前 Store Observations、project-boundaries.md 及 greenfield-no-legacy-migration.md。Figma仅流程参考。

仅当前区域订单 r2 应用、US region211、LOCAL/SEMI；不扩展全托、其他区域、定时采集、发货/取消/退款或财务。禁止真实平台/真实店铺请求，不修改全平台成功率、GMV卡片、合成预览和现有实例。交付Draft PR和保留的独立本地入口，不自动合并、生产部署或关闭父Issue。测试金额不能证明真实半托应用有敏感金额接口权限。

## owner、调用与注入

Store继续唯一拥有原member/Store/OfficialAttempt/Connection和已核验regional身份；Store Observations继续唯一拥有command/sync/checkpoint/records/head及单DB提交。首页选单店/日期→原orders/syncs→原Directory/Begin/Starter/worker/Step→显式TemuOrders port→OfficialObservationAccess→OfficialClient的只读区域接口→原decoder→CommitPage→既有saved metrics/HTTP/BFF→首页invalidate。正式非preparation组合补入TemuOrders；不新建schema、状态机、事务或恢复owner。

内部命令目录允许显式Temu订单，原SHEIN隐式全选及通用列表/详情继续只限SHEIN，不将Temu记录塞入其他平台投影。复用已有StoreOrderSync，Temu独立session namespace；切换筛选不自动POST，原key/载荷不变，未确认时先查询原命令，明确not-found才原键重放，终态清理后可新建同步。

## 身份、权限与边界检查

只使用ReadObservation(orders,Sync=true/false)，不借产品用途。复用ReadSavedTemuOrderBinding的完整本地r2身份验证与UTC表示，读取前解密原Attempt的AES-GCM凭据；解密必须匹配app/partition，不能含SHEIN字段、供应商或其他凭据。当前本地binding与预期binding全字段相等，包括org/store/version/app/partition/LOCAL或regional_semi身份hash、mallId/semiUniqueId、region211、scope hash、服务/token expiry与连接版本。

初始Authorize用原provider.QueryStore核验当前原token区域身份，重新读取本地授权/连接事实；线上核验所得身份、expiry、scope摘要与原facts精确匹配才生成原短期ObservationHandle。订单branch保持与商品branch隔离，订单r2不通过商品binding。每次页面读取前后、保存前复查原member/Store/Attempt/Connection和密文指纹，撤权、换绑、scope/版本/有效期变化丢弃页面，不采用迟到回执。

固定首次Authorize起最多30秒总页期限，取caller更早deadline；QueryStore最多5秒、单HTTP最多10秒，不能续期。OfficialClient每页重新QueryStore并要求准确r2/US/LOCAL或SEMI；列表scope bg.order.list.v2.get必需，bg.order.amount.query独立判断，缺金额scope不发金额请求，保存amount未知。产品scope、CN供应商身份或全托不能进入订单路径。本地统计仍不解密、不调用平台。

## 只读协议、日期、金额与覆盖

消费已批准保存合同中的固定API：同一application IdentityPartition对应的localAPIOrigin，POST /openapi/router，签名JSON type=bg.order.list.v2.get，pageNumber从1、pageSize100、createAfter=Start.Unix、createBefore=End.Unix-1。原48小时窗口、闭区间查询无重叠，不增状态/履约过滤。只parentOrderTime归日期，不用更新或同步时间。

列表先由既有DecodeConsumerOrderPage解析/验证允许字段，<=100/页、2MiB/响应；按本页原parentOrderSn各请求一次type=bg.order.amount.query，原父单ID必须匹配回执。只retailPriceTotal/USD美分，不换金额版本或字段兜底，不按子单、数量、包裹或站点叠加。缺scope、明确接口拒绝/不可用或warning导致该父单amount未知；异常ID/非法非空金额/超限等保持原decoder拒绝。不把拒绝当零，不把金额缺失变成列表覆盖完成证据。总deadline或cancel到期整个页面不可提交。

继续使用现有2MiB读取阶段上限、固定origin/禁止redirect/不自动重试、不输出原body/provider错误。金额按最多100个父单有界读取，受原30秒总预算约束；真实规模吞吐不在本次离线验收范围，不追加并发框架或通用runner。

分页、去重、sticky incomplete、稳定total/100页/10000父单/窗和原全任务上限沿原TemuOrders；只完整窗口与完整head可形成覆盖，short/changed total/重复冲突保持partial，不累加历史generation。同步无平台mutation；失败/丢响应/重启/并发沿原command key/revision和Temporal恢复。无新资金/额度/外部写入或第二事实源，Legacy N/A。

## Must、Threat Model与验证

Must：两个模式可用离线正式链路；同父多子/多站只计一次；原创建日期和店铺筛选联动；金额未知隐藏总额、覆盖不足有提示；订单权限和产品权限隔离、原租户和绑定复验；公开progress只允许temuOrders total/complete，不泄露token/hash。Threat Model沿原租户/member/Store授权、凭据、原单事务/幂等；无新增Accepted Risk。真实授权/业务API、真实吞吐及用户产品验收NOT_RUN。

TDD先RED：两模式订单Authorize与缺list/amount scope、scope/身份/连接变更；固定签名请求和闭区间末秒、按父单单次金额、拒绝/缺失/warning与2MiB/总deadline；Directory和非preparation注入；progress严格去敏；共享UI原key恢复、Temu namespace隔离。原保存/日期/分页测试复用，相关Go/PG、UI/tsc/lint自检；完成路径后独立最终交付检查。

独立新PostgreSQL及本土A/半托B测试店，官方形状回执只经无网络fallback假HTTP transport送入真实OfficialClient/正式保存/HTTP/BFF/UI。A含昨日、末秒、较早日期父单和多子/站/包裹；B含缺金额及7天覆盖。实机手动同步→保存→自动更新，日期/店铺切换、扩大日期覆盖不足与原键重放去重，保留入口与记录供用户试用；不得改现有TikTok/Walmart/SHEIN实例或原合成预览。
