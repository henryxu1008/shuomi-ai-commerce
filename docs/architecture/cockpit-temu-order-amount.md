# Temu US 消费者订单保存与单店商品金额

Design Basis: Independent Architecture。Admission Status: IMPLEMENTATION_READY / FROZEN。执行 #1163。用户2026-11-08明确要求本土店和半托管一起做，先确定读取/保存合同，再用独立本地测试订单验证。适用产品依据为该用户决定、greenfield-no-legacy-migration.md及project-boundaries.md；Figma不作为UI权威。

## 用户结果与本批边界

运营人员在新独立实例显式选择一家Temu美国本土/半托管测试店，可按UTC+8完整日的昨日/近7天/近30天查看「已保存订单商品金额（原币）」。离线官方形状回执经过原Store Observations Begin/Step/CommitPage、PostgreSQL、正式GET/BFF与首页；包括缺金额/币种/日期和覆盖不足提示。

只准备来源和本地读取，不注入联网TemuOrderReader、不开真实订单同步/通用订单目录。全托管、非美国地区、真实商家数据、外部平台、全平台成功率、GMV卡片、合成预览、其他保留实例、发货/取消/退款/结算mutation、迁移、正式部署、合并及Issue关闭均不在本批范围。离线回执不证明真实平台权限或产品验收。

## 官方证据与固定金额来源

2026-11-08只读公开文档核对，无商家API请求：

- [区域身份](https://partner.temu.com/documentation?menu_code=fb16b05f7a904765aac4af3a24b87d4a&sub_menu_code=93de550b56c8417caccb88824be3e614)：bg.open.accesstoken.info.get返回mallId、mallType（1 SEMI/100 LOCAL）、半托管semiUniqueId、regionId（美国211）、expiredTime和apiScopeList。regionId不是siteId；CN/PA uniqueId供应商身份不等于区域mallId/semiUniqueId。
- [消费者订单列表](https://partner.temu.com/documentation?menu_code=fb16b05f7a904765aac4af3a24b87d4a&sub_menu_code=554fd46b45ee49269cbdd6d4008a5dc1)：bg.order.list.v2.get支持SEMI/LOCAL；pageNumber从1开始、pageSize最大100、totalItemNum及pageItems。parentOrderMap.parentOrderSn为父订单，parentOrderTime为创建Unix秒。createAfter/createBefore必须成对且闭区间；不以更新时间/确认时间代替创建时间，不加状态/履约/待发货过滤。
- [订单金额V1](https://partner.temu.com/documentation?menu_code=fb16b05f7a904765aac4af3a24b87d4a&sub_menu_code=ef6952c9640e4d17b1200e140954d4fc)：bg.order.amount.query按parentOrderSn单独读取，敏感接口需另行授权；parentOrderMap.retailPriceTotal为美国/加拿大返回的商品促销总价，amount为本币最小单位、美国美分，currency原币。当前公开页面标Local，不从列表权限推断半托管金额权限；未来在线半托金额能力须取得准确区域token的显式scope和成功回执证据，否则金额未知。本批两模式仅验证测试回执合同，绝不声明真实商家均可调用。
- [金额版本指南](https://partner.temu.com/documentation?menu_code=fb16b05f7a904765aac4af3a24b87d4a&sub_menu_code=49ab02414b9a41a783d2fbb129109620)：V2/V3有明确站点集合，不按版本数字自动替换美国V1。固定本批美国V1的retailPriceTotal；不以retailPriceTotalTaxIncl/TaxExcl、customerPaid、basePrice、estimatedRevenue或其他版本字段兜底。

金额在父订单级取一次，不乘quantity、不累加子订单orderSn/SKU/组合商品/包裹/履约仓库/站点；不重复扣折扣、不减退款、不加入运费税费，也不推断已支付/净收入。保留各订单原状态，不因取消状态删除原保存事实。当前只支持USD美分整数，精确转换成两位小数字符串；负数、非整数、溢出或非USD非空币种拒绝，缺币种/amount保留未知，明确0是0.00。不同币种/地区必须另有最小单位合同，不盲除100。

## Store身份与权限：不复用供应商身份

事实owner仍为Store的原OfficialVerificationFacts/Attempt/Connection，不新增表或订单身份owner。区域订单应用显式声明SupportedModes=LOCAL|SEMI及新独立revision（r2），与原商品应用r1能力区分；配置不自动切换，正式runtime仍保留现有配置和不开在线reader。原CN/PA供应商商品应用、r1本土商品应用和现有商品执行合同不放宽。注册可接受精确的r1商品应用或精确的r2区域订单应用，无fallback/旧数据迁移。

r2仅US/GLOBAL分区，token/app/读路由必须同分区。区域核验LOCAL仍用local namespace+mallId；SEMI新增regional_semi namespace，保留原mallId及semiUniqueId，两者均必填，身份哈希绑定平台/分区/namespace/mallId/semiUniqueId，不映射CN或其他分区身份。新r2 client仅在显式配置时允许mallType1，r1仍只允许100；全托管不可进入区域订单。保存到原Store已核验事实，原商家唯一归属、服务到期、连接/尝试版本与权限owner不变。

订单Binding为现有binding窄投影（site=temu-order-us、region211、mode本土/半托、r2订单应用），新增专用ValidTemuOrderApplication，不让订单binding通过ValidTemuApplication或产品权限路径。r2订单应用不授予商品发布/图片/规则能力；原商品路径拒绝该订单binding，CN供应商连接不可伪造区域订单资格。

本地读取走ReadObservation(orders,Sync=false)，验证当前组织/actor/member、订单读权限、Store、app/version、连接/attempt/状态、原facts身份哈希/模式/region/到期与scopes摘要；不解密credential、不QueryStore、不借用product_rules。本地ReadSavedTemuOrderBinding返回同一完整binding，当前binding与捕获head全字段相等，包括scope摘要；连接、身份或权限变更使统计不可用。列表与金额线上权限未来独立证明，本批不创建有效线上能力假象。

## 原生观察、日期与去重

Store Observations唯一owner；新增TemuOrder typed variant，不塞入SHEIN/Walmart/TikTok，也不让integration引用Marketplace业务root。只保留parentOrderSn、固定site、region211、原parentOrderTime的UTC RFC3339秒、原parentOrderStatus及原V1商品金额/币种。PII、买家地址、电话、邮箱、卡号、物流/原rawresponse一律不保存或日志输出。缺日期留空，非法非空日期拒绝。

组织/店铺/generation/parentOrderSn是保存身份。同一父订单多子单、多站点、多页重放均不增加金额；不同父单独立，历史generation不叠加。相同ID同payload保留首次事实，相同ID冲突或同窗跨页重复标sticky incomplete，不能覆盖原值制造完整head。

首页UTC+8[start,end)最多30天，仅父订单创建时间入组。原48小时OrderWindows保留，reader查询createAfter=Start.Unix、createBefore=End.Unix-1，闭区间不重叠。统计要求原sync.Range.End达到exclusiveEnd；已知日期在查询窗外拒绝页，不以更新/确认/抓取日期兜底。缺日期记录参与缺失提示，SQL先按日期筛选再LIMIT251。

## 来源读取、分页、保存和恢复

TemuOrderReader是显式注入的窄port：固定binding/window/page，返回typed父订单页、总数。integration提供纯decoder：列表success=true/result.totalItemNum/pageItems，数组必需、<=100、2MiB；每条父单region211，父单ID页内唯一。金额输入按原请求父单ID关联，回执parentOrderSn必须一致；缺回执/敏感权限失败/计算未完成/非空warning保留金额未知，不取其他字段或把错误当零。金额payload仍需有界，不保存raw。未注入reader不能采集，正式组合不注入。

复用原checkpoint新增TemuOrderProgress的首个total与Complete。pageNumber顺序推进；默认pageSize100，累计原始条数达到稳定total才能结束该窗，短页/空页未达total直接partial，total变化、重复页/ID、载荷冲突、count越界均partial。每窗最多10000父单/100页、原总30000页/24小时及30秒activity限制不变。每窗完成才推进下一窗，全窗耗尽且无incomplete才Complete。

返回后复验Access.Authorize、Merchant.Check/binding，由原CommitPage行锁单DB事务原子保存records/checkpoint/status/head。仅Temu Orders新增100条能力（与TikTok同上限），原其他平台限制不变。typed variant严格互斥、ID/site/region匹配。只有完整TemuProgress发布head，sticky incomplete不得清除。失败/响应丢失/并发仍由原sync/revision/command key恢复，无新恢复owner、无外部mutation或盲重试。仅改fresh schema constraint与strict verifier；新独立PG，不ALTER/迁移/重播种保留实例。

## 统计接口与页面

复用OrderAmountMetrics及math/big精确累计，单店GET /api/v1/workbench/store-observations/orders/temu-saved-amount-metrics，经原身份envelope、订单读权限、20秒/no-store、BFF、严格日期query。只当前完整head且head/latest同一完整sync、有效ObservedAt及足够Range；最多250候选，缺金额/币种/时间隐藏全部金额并给数量，覆盖不足/limited不返回完整合计。空集合为无记录，不猜0或币种。

响应前再次读取当前授权/绑定与heads/latest，任一漂移不可用；前端复用独立单店金额面板、日期/店铺筛选及迟到响应隔离，只TemuUS单店可见。页面明确美国V1商品促销总价、按父订单计、不含税运费/不代表GMV或结算，真实平台金额授权未验证。原全平台/GMV/合成预览不变。

## 验证与停止点

Threat Model沿原verified身份、租户/成员/Store授权、凭据、原事务/幂等。Accepted Risks无新增；Legacy decision N/A。新高风险边界仅区域SEMI原Store身份及原记录variant/checkpoint，不新增IAM、runner或通用恢复平台。

TDD：r1拒绝SEMI、r2两模式身份/分区/缺semiUniqueId、商品与订单权限隔离；decoder父子金额不叠加/ID关联/精确美分/缺字段/PII不保存；日期尾秒/次日/窗边界；稳定total、短页、重复/冲突、100条事务与互斥variant；SQL日期先筛再限、租户/权限/连接/来源漂移。复用相关现有测试与CI，最终一次独立交付检查。

独立本地新PG用本土与半托两家测试店、两组日期、同父多子/多站、重放、旧generation、缺金额/币种和覆盖不足，经正式保存和HTTP/BFF/浏览器无刷新切换；全程禁止外部请求。保留可访问入口供用户试用，不自行签发产品验收或合并。

独立架构复核：/root/architecture_review核对设计1179fa2ea403ba9e4d406c6b0117a847af43f171与实际owner/授权/事务，未发现BLOCKER，结论IMPLEMENTATION_READY。r1/r2与产品拒绝、金额关联/缺失/警告、100条原子保存、稳定total/重复页与末秒边界归IMPLEMENTATION_TEST在本批收敛。

