# TEMU 官方接口与站点证据

> 对应 [#31 接入设计](temu-platform-v1.md)；收集日期 2026-10-11；E30 与规格交叉核对补充于 2026-10-12。
>
> 来源仅为 TEMU 官方页面，公开 partner 文档与用户登录后可读的 GLOBAL / agentpartner 文档。
>
> 时间为页面显示的更新时间；没有调用商家业务 API、读取应用密钥或验证实际店铺权限。

## 1. 授权、分区、创建与结果合同

| 来源 | 官方页面 | 显示更新时间 | 本轮核实的事实 |
| --- | --- | --- | --- |
| E01 | [Registration Overview](https://partner.temu.com/documentation?menu_code=52ef88bdef1d4527b15f6d303b173e48) | 2025-08-15 | US/EU/GLOBAL 面向 Local/Cross-border；Mainland China partner.kuajingmaihuo.com 面向 Cross-border/Fully Manage；Cross-border 标签不是全托/半托身份值 |
| E02 | [分区说明](https://agentpartner.temu.com/document?cataId=875196199516&docId=909799935182) | 2026-01-05 | 官方称四个区域 CN/US/EU/GLOBAL；app_key/secret/token/接口必须同区域；PA 与 CN 共用应用/授权入口，但文档未给出身份字段等价映射 |
| E03 | [PA 网关调用说明](https://agentpartner.temu.com/document?cataId=875198836203&docId=929750644349) | 2026-05-06 | 部分 CN API 迁移到 PA，需要新授权/新 type；未列举接口保留 CN；FAQ 明确分别以 CN info.get / PA info.get.global 查询当前 token 的 API 权限；基本信息中的旧链接 915047650924 当前无权限，实际清单引用此可读叶子页 |
| E04 | [鉴权信息](https://agentpartner.temu.com/document?cataId=875196199516&docId=896168820140) | 2026-06-25 | app_key 为应用维度唯一值，token 为店铺维度唯一值；CN 手动授权，页面说明自研 365 天/第三方 90 天；新 token 会使旧 token 失效；本系统以实际 expiredTime 为准 |
| E05 | [Seller Authorization Guide](https://partner.temu.com/documentation?menu_code=38e79b35d2cb463d85619c1c786dd303) | 2026-01-07 | Local/Cross-border 后台与 manual/callback 适用不同；scope/期限可由商家选择、授权可撤销；in-app 标记 Launch Soon，不能默认所有应用可用 |
| E06 | [Endpoints and Request Method](https://partner.temu.com/documentation?menu_code=38e79b35d2cb463d85619c1c786dd303&sub_menu_code=8311de2b2d434e4d805e88413ab815d8) | 2025-01-26 | US/EU/GLOBAL HTTPS POST router；例举国家不是全部站点字典 |
| E07 | [bg.open.accesstoken.create](https://partner-us.temu.com/documentation?menu_code=fb16b05f7a904765aac4af3a24b87d4a&sub_menu_code=82674d12ebe64af2820d62ebbc2ecc16) | 2025-07-10 | 一次性 code、10 分钟；结果含 mallId/scope/expiry；不能重发丢失结果的 exchange |
| E08 | [区域 bg.open.accesstoken.info.get](https://partner.temu.com/documentation?menu_code=fb16b05f7a904765aac4af3a24b87d4a&sub_menu_code=93de550b56c8417caccb88824be3e614) | 2025-11-19 | mallId、expiredTime、apiScopeList、mallType 1-SEMI/100-LOCAL、semiUniqueId、regionId（示例 USA-211）；没有 full 枚举；regionId 不是 siteId |
| E09 | [CN bg.open.accesstoken.info.get](https://agentpartner.temu.com/document?cataId=875198836203&docId=881490740978) | 2026-06-28 | result mallId/expiredTime/apiScopeList；没有 uniqueId 或 mode 字段 |
| E10 | [CN bg.mall.info.get](https://agentpartner.temu.com/document?cataId=875198836203&docId=881490244071) | 2025-02-25 | result semiManagedMall、isThriftStore；不能与区域 token 的 LOCAL 混为一谈 |
| E11 | [PA bg.open.accesstoken.info.get.global](https://agentpartner.temu.com/document?cataId=875198836203&docId=929722395417) | 2026-05-26 | uniqueId 明确为 Unique identifier of mall；mallName/expiredTime/apiScopeList；未见 mallId、mallType、semiManagedMall。商家身份可直接选 PA uniqueId，无须把它强行转换成 CN mallId |
| E12 | [签名规则](https://agentpartner.temu.com/document?cataId=875196199516&docId=896167235113) | 2025-05-11 | 外层 key ASCII 排序，精确参数字符串，secret 前后拼接，MD5 大写；nested wire JSON 必须一致；签名前文本含 secret 不可记录 |
| E13 | [bg.glo.goods.add](https://agentpartner.temu.com/document?cataId=875198836203&docId=925526695187) | 2026-08-25 | 上传供应商货品，返回 productId/productSkcId/productSkuId/extCode；半托 bindSiteIds、站点价格/仓库路由；价格 CNY 分/USD cent；没有证实全托消费者逐国创建参数 |
| E14 | [bg.glo.goods.list.get](https://agentpartner.temu.com/document?cataId=875198836203&docId=924479235154) | 2026-06-29 | skcExtCode、bindSiteIds、page/pageSize 筛选；结果含 productId、SKU ID/外部码、SKC 外部码、bindSites；状态 0 未发布/100 在售/200 下架终止/300 删除 |
| E15 | [bg.glo.goods.detail.get](https://agentpartner.temu.com/document?cataId=875198836203&docId=925528074151) | 2025-08-27 | productId 必填；详情主要基本资料、类目与详情素材；本轮未见完整 SKU/价格回显，不可宣称 exact body readback |
| E16 | [bg.glo.product.search](https://agentpartner.temu.com/document?cataId=875198836203&docId=931835549486) | 2026-10-11 读取；未摘录更新时间 | 货品生命周期查询，productSkuIdList、pageNum/pageSize；页大小上限 100；dataList 包含 product/SKC/SKU、selectStatus 等；不把创建等同在售 |
| E17 | [temu.local.goods.v2.add](https://partner.temu.com/documentation?menu_code=fb16b05f7a904765aac4af3a24b87d4a&sub_menu_code=074e7513eea7489a87131b0e41ec83a8) | 2026-09-07 | Local、Product Management；goodsBasic.externalGoodsId 单店唯一，最大 40 字符；创建返回 goodsId/SKU；未证实与 outGoodsSn 的映射 |
| E18 | [temu.local.goods.v3.add](https://partner.temu.com/documentation?menu_code=fb16b05f7a904765aac4af3a24b87d4a&sub_menu_code=419748d505a3483f8d210d978cb813f8) | 2026-09-16；更新记录含 09-28 | Local；官方自动类目/属性推断、图片 URL 抓取；result externalGoodsId/goodsId；不是全/半托合同，不能与 V2 约束互换。2026-10-12 重读：price 明确按店铺国家的币种/精度并引用 E24；weight/length/width/height 为 STRING，美国 lb/in、其他国家 g/cm；V3 也不提供 GE/AM 缺失价格规则 |
| E19 | [temu.local.goods.list.retrieve](https://partner.temu.com/documentation?menu_code=fb16b05f7a904765aac4af3a24b87d4a&sub_menu_code=ce4883149f784c208e1b9a52566ea2c8) | 2026-06-17 | Local，pageSize≤100/默认25、opaque pageToken；outGoodsSnList/goodsIdList 过滤；goodsSearchType 包含 ACTIVE/INACTIVE/INCOMPLETE/DRAFT/DELETED；DRAFT 不是保存草稿 API 证据 |
| E20 | [bg.local.goods.publish.status.get](https://partner.temu.com/documentation?menu_code=fb16b05f7a904765aac4af3a24b87d4a&sub_menu_code=db14c29a74744247828e8ebc69443dfe) | 2025-07-03 | Local；goodsIdList 必填；返回 goodsId/status/subStatus；丢失创建标识时不能单凭该 API 解决 UNKNOWN |
| E21 | [temu.local.mall.tags.get](https://partner.temu.com/documentation?menu_code=fb16b05f7a904765aac4af3a24b87d4a&sub_menu_code=ec4ff89b268441cd9400a64706c215b0) | 2025-12-02 | Local store tags：新品/官方翻新/MTO/二手/US-to-CA/书店等资质提示；不是全部站点枚举，不自动授权履约或扩大本批产品类型 |
| E22 | [Brazil 接口差异](https://partner.temu.com/documentation?menu_code=38e79b35d2cb463d85619c1c786dd303&sub_menu_code=0613960f489243358637212facf31549) | 2025-08-17 | 巴西商品 costTemplateId 要求；有些示例引用较早商品 API，实际采用 V2 必须核对对应当前合同；税票/订单/物流不进入 #31 |
| E23 | [半托管站点列表](https://agentpartner.temu.com/document?cataId=875196199516&docId=893022798763) | 2025-09-29 | 96 项、70 true/26 false、关联站点；完整摘录见 §3；不借用为 Local 或全托清单 |
| E24 | [Product Differences in site currency, volume](https://partner.temu.com/documentation?menu_code=38e79b35d2cb463d85619c1c786dd303&sub_menu_code=2cc05ca389ca4eeb8e704e903ebcc6bd) | 2026-01-26 | 独立 39 项规格表，siteId/partition/语言/币种/重量尺寸与精度/价格边界；无是否开站/商家可售字段，见 §4 |
| E25 | [Origin Region List](https://partner.temu.com/documentation?menu_code=38e79b35d2cb463d85619c1c786dd303&sub_menu_code=40f3cd9aa5fa409cb004f403c61c7261) | 2026-08-14 | originRegion1/2 是原产地字典，不是 mall regionId → siteId 或开站依据；不能用国家名称相同推导权限 |
| E26 | [CN/PA 接口列表](https://agentpartner.temu.com/document?cataId=875196199516&docId=899312840459) | 2026-03-09；2026-10-11 当前内容 | 同一基础 API 组明确列 CN bg.mall.info.get：查询当前 token 对应全托/半托类型；PA info.get.global：查询当前 token 对应店铺 uniqueId 和接口权限。与 E03/E04/E11 组成同一店铺授权的固定验证流程，不能仅凭客户端模式选择 |
| E27 | [货品发布流程](https://agentpartner.temu.com/document?cataId=875196199516&docId=896172443264) | 2026-04-09 | 适用全托/半托；全托跳过站点/仓库/运费模板步骤；一次请求一个 SPU；半托仅 siteSupplierPrices，全托仅 supplierPrice；图片需官方上传 URL，自定义规格/尺码表可能需要独立 create 操作 |
| E28 | [Add Products Guide V2.0](https://partner.temu.com/documentation?menu_code=fb16b05f7a904765aac4af3a24b87d4a&sub_menu_code=ea85c857a38d42a7a22fb6779776b166) / [Product Listing Instructions](https://partner.temu.com/documentation?menu_code=38e79b35d2cb463d85619c1c786dd303&sub_menu_code=d7943785ff2e4185976170f07924256e) | 2026-02-08 / 2025-10-11 | 类目读取针对当前店铺；availableStatus=0 才可售；V2 流程按授权 mall 组装资料，无客户端选择任意目标店铺/站点的步骤。币种/包装规格引用 E24；不因此扩大到订单功能 |
| E29 | [temu.local.goods.cats.rules.get](https://partner.temu.com/documentation?menu_code=fb16b05f7a904765aac4af3a24b87d4a&sub_menu_code=5853e21ac9b74fdc96497a52c848765f) | 2026-09-23 | catId 请求；fieldRules/expandCatType/secondHand/warnings；有 category unavailable / shop status abnormal 等拒绝；当前返回未见站点映射/币种字段，不能拿它冒充映射接口 |
| E30 | [Model Site and Region](https://partner.temu.com/documentation?menu_code=38e79b35d2cb463d85619c1c786dd303&sub_menu_code=07938e60f95344a0a9444f524ea61e31) | 2026-07-16 00:11:23 | 2026-10-12 在用户已登录的 GLOBAL 会话读取完整正文及 235 行地区表。87 行有 siteId（83 个不同 siteId），39 行 LOCAL=true、57 行 SEMI=true、9 行 SEMI=coming soon；空白不是 false 或授权。regionId 与 siteId 是明确的不同字典，见 §5；该页面介绍订单履约地区，不取代当前 token/mall 的发布权限 |
| E31 | [Product Basic Introduction](https://partner.temu.com/documentation?menu_code=38e79b35d2cb463d85619c1c786dd303&sub_menu_code=2bdd2c8933604967a613caa5f87e5963) | 2025-10-11 11:52:57 | 当前店铺类目/品牌/运费模板、创建后进入审核、创建/审核分离；基础价格/包装示例为 USD/lb/in，不能拿这些美国示例替缺失的非美国规格。站点特殊接口章节当前仅 Brazil，未找到 GE/AM 补充页 |
| E32 | [Guide of price](https://partner.temu.com/documentation?menu_code=fb16b05f7a904765aac4af3a24b87d4a&sub_menu_code=e351264f0a1749cba4f112345a86813c) | 2026-01-30 14:04:02 | 2026-10-12 已登录后读取：支持商家明确手工设置基础价格，计算器结果仅供参考；指南的计算器流程引用 baseprice.recommend 和较早 bg.local.goods.add，不能当作 V2 的新强制合同。价格管理 API 仅面向获白名单的本土自研应用，需店铺层申请；不因此授权独立调价或自动申请。正文没有 GE/AM 的币种/精度/范围，未关闭规格缺口 |

| E33 | [bg.goods.cats.get](https://agentpartner.temu.com/document?cataId=875198836203&docId=877327073796) | 2025-03-13 15:57:45 | 2026-10-12 登录后核对：全托/半托类目读取仍属 CN；根层省略 parentCatId，siteId 可选、showHidden 为布尔。result.categoryDTOList 保留父类目、末级、层级及隐藏类型；缺隐藏标志不是可选择。权限必须用 E09 的当前 CN scope，不能拿 PA 权限推断 |
| E34 | [bg.local.goods.cats.get](https://partner.temu.com/documentation?menu_code=fb16b05f7a904765aac4af3a24b87d4a&sub_menu_code=b49f03d2b1744700ab762b9fe80d1af7) | 2026-09-23 15:55:07 | 2026-10-12 登录后核对：独立 result.goodsCatsList，父字段 parentId、末级 leaf、availableStatus=0 可售/1 不可售，另有 expandCatType/secondHandCategory。E28 根查询 parentCatId=0；读取针对当前 mall，不能增加任意 siteId。缺 availability 保留未知 |

| E35 | [bg.goods.attrs.get](https://agentpartner.temu.com/document?cataId=875198836203&docId=877328004022) | 2025-03-13 15:57:46 | 2026-10-12 登录后读取：供应商当前末级模板仍走 CN；properties 包含 required/isSale、输入/选择上限、精度、范围、单位、父属性展示条件与属性值关联。缺 required/precision 不能默认为 false/0。supportedType 会覆盖当前 mall 渠道，本批读取不暴露该覆盖项或旧创建时间 |
| E36 | [temu.local.product.attributes.get](https://partner.temu.com/documentation?menu_code=fb16b05f7a904765aac4af3a24b87d4a&sub_menu_code=06587aa615384310a1bac727aa82b64d) | 2026-01-30 17:57:30 | 2026-10-12 登录后读取完整属性模板：catId 必填，language/costTemplateId 可选；所选运费模板会影响跨境必填属性。required、transnationalAttribute、controlType、精度/范围/值规则和父值→子属性值组分别保留；缺失标志不能当 false，显式 precision=0 不能当未知 |

E17 在 2026-10-12 登录后展开核对：业务参数直接位于外层 JSON（示例没有 `request` wrapper），本土创建没有客户端 `siteId`/`mallId` 选项。价格 `amount`/`currency`、包装四维数值均为 STRING，数量 LONG 的显式零与缺失不同；单位必须由 E24 原站点规格和明确资料一致确定。`LocalGoodsV2` 保留可选品牌/产地/二手/证书等事实，不默认填写无品牌声明、生产时间或库存；数值检查不签发整体 readiness。创建返回 goodsId、skuInfoList 及 warnings，不证明审核或在售。

新增条件性冲突：E17 `goodsBasic.secondHand.businessScope` 描述 0=Regular/1=Collectibles/2=Luxury，E29 `secondHand.businessScope` 描述 1=Regular/2=Collectibles/3=Luxury。当前不建立加减一 alias，不凭字段同名转换或批准二手资料；在本批实际资料校验中保持未核实，须有对应当前接口的明确官方依据才消费。

### 文档限定的 router

| 身份分区 | transport route | 完整 URL | 适用说明 |
| --- | --- | --- | --- |
| CN | CN | https://openapi.kuajingmaihuo.com/openapi/router | 未迁移 CN API，含本轮 CN 身份接口 |
| CN | PA | https://openapi-b-partner.temu.com/openapi/router | E03/E26 已列举 migrated type 与当前 token 授权查询；不是任意 CN fallback |
| US | US | https://openapi-b-us.temu.com/openapi/router | 原 US 应用/token 与该 API scope |
| EU | EU | https://openapi-b-eu.temu.com/openapi/router | 原 EU 应用/token 与该 API scope；不是“地理欧洲全部” |
| GLOBAL | GLOBAL | https://openapi-b-global.temu.com/openapi/router | E24 用 GLO 标签，不能由“非欧美国家”推导全部路由 |

E02 的全/半托“发品 CN”与 E03 新迁移路由按具体 API/version 对齐，不能随意选择旧入口。E03 把 bg.goods.add 对应到 bg.glo.goods.add/temu.goods.add，新实现只固定已查证的一种 type；别名不形成第二发送机会。类别/属性/父规格/尺寸/图片/运费模板/品牌沿 E03 的 migrated API 选择，正式请求还须逐项验证 scope 与当前参数。

## 2. 身份验证结论与站点/恢复依据

- CN result.mallId 与 PA result.uniqueId 没有已读取的等值关系；这不需要成为实现前置。E04 店铺粒度 token + E11 PA mall 唯一标识 + E26 同一当前 token 的固定 CN 类型/PA 身份查询，支持选 PA uniqueId 为唯一 supplier 身份，CN 类型为该凭据下的附属验证事实。不拿名字关联，也不跨 US/EU/GLOBAL 试探；真实成功响应与权限仍在实现/授权验证层收敛。
- 本土 regionId 与 siteId 值域不同，E30 正文已证实例如 US 211→100、Canada 37→101、Belgium 20→142。该表还把 Guam 83、Puerto Rico 164、Virgin Islands (U.S.) 219、Northern Mariana Islands 236 映到 site 100；不能反向把一个 site 当成唯一 mall region。E24 规格字典没有实时商家资格。Local API 目标是当前 token 的 mall，不要求虚构一个“全部可售站点”授权 API；共享字典不能自证商家授权。
- Local externalGoodsId 与 list.retrieve 的 outGoodsSn：两个名称分别已查证，映射关系未查证；不能将未知关系写进自动恢复实现。
- PA detail.get 不完整回显 SKU/价格/body hash；list.get 的 correlation 与 bindSites 只能用于足够明确的正向证据。空查询不证明无副作用。
- 没有真实 application 发布、merchant scope/eligibility、真实 token/创建/撤权证据；全部 NOT_RUN，不影响只读设计，但不能冒充三模式及所有国家产品验收。

## 3. 半托管完整站点表

从 E23 可见表结构读取全部 96 行；true=70，false=26。这个统计指文档快照，不能等同当前 70 国全部商家可售。未列出的 160/195 等编号不补造。关联栏“泛欧”保留原义，实际泛欧资格和完整站点集合按 E13 的业务合同核实（含 6000135/6000136/6000137 等错误要求），不能简单强制所有欧洲地理国家。

| siteId | 官方站点名 | 是否开站 | 关联站点 |
| --- | --- | --- | --- |
| 100 | 美国 | true | - |
| 101 | 加拿大 | true | - |
| 102 | 英国 | true | - |
| 103 | 澳大利亚 | true | 103,104 |
| 104 | 新西兰 | true | 103,104 |
| 105 | 德国 | true | 泛欧 |
| 106 | 法国 | true | 泛欧 |
| 107 | 意大利 | true | 泛欧 |
| 108 | 荷兰 | true | 泛欧 |
| 109 | 西班牙 | true | 泛欧 |
| 110 | 墨西哥 | true | - |
| 111 | 葡萄牙 | true | 泛欧 |
| 112 | 波兰 | true | 泛欧 |
| 113 | 瑞典 | true | 泛欧 |
| 114 | 瑞士 | true | - |
| 115 | 希腊 | true | 泛欧 |
| 116 | 爱尔兰 | true | 泛欧 |
| 117 | 塞浦路斯 | true | 泛欧 |
| 118 | 日本 | true | - |
| 119 | 韩国 | false | - |
| 120 | 沙特 | true | - |
| 121 | 新加坡 | false | - |
| 122 | 阿联酋 | true | - |
| 123 | 科威特 | true | - |
| 124 | 挪威 | true | - |
| 125 | 智利 | true | - |
| 126 | 马来西亚 | true | - |
| 127 | 菲律宾 | true | - |
| 128 | 中国台湾 | false | - |
| 129 | 泰国 | true | - |
| 130 | 卡塔尔 | true | - |
| 131 | 约旦 | true | - |
| 132 | 巴西 | false | - |
| 133 | 阿曼 | true | - |
| 134 | 巴林 | true | - |
| 135 | 以色列 | true | - |
| 136 | 南非 | true | - |
| 137 | 捷克 | true | 泛欧 |
| 138 | 匈牙利 | true | 泛欧 |
| 139 | 丹麦 | true | 泛欧 |
| 140 | 罗马尼亚 | true | 泛欧 |
| 141 | 保加利亚 | true | 泛欧 |
| 142 | 比利时 | true | 泛欧 |
| 143 | 奥地利 | true | 泛欧 |
| 144 | 芬兰 | true | 泛欧 |
| 145 | 斯洛伐克 | true | 泛欧 |
| 146 | 克罗地亚 | true | 泛欧 |
| 147 | 斯洛文尼亚 | true | 泛欧 |
| 148 | 立陶宛 | true | 泛欧 |
| 149 | 爱沙尼亚 | true | 泛欧 |
| 150 | 拉脱维亚 | true | 泛欧 |
| 151 | 马耳他 | true | 泛欧 |
| 152 | 卢森堡 | true | 泛欧 |
| 153 | 塞尔维亚 | true | - |
| 154 | 摩尔多瓦 | true | - |
| 155 | 黑山 | true | - |
| 156 | 冰岛 | true | - |
| 157 | 安道尔 | false | - |
| 158 | 波黑 | true | - |
| 159 | 阿尔巴尼亚 | true | - |
| 161 | 科索沃 | false | - |
| 162 | 哈萨克斯坦 | false | - |
| 163 | 秘鲁 | true | - |
| 164 | 哥伦比亚 | true | - |
| 165 | 格鲁吉亚 | true | - |
| 166 | 亚美尼亚 | true | - |
| 167 | 阿塞拜疆 | true | - |
| 168 | 乌克兰 | false | - |
| 169 | 乌拉圭 | false | - |
| 170 | 毛里求斯 | true | - |
| 171 | 摩洛哥 | true | - |
| 172 | 多米尼加 | false | - |
| 173 | 哥斯达黎加 | false | - |
| 174 | 土耳其 | false | - |
| 175 | 阿尔及利亚 | false | - |
| 176 | 巴拿马 | true | - |
| 177 | 肯尼亚 | false | - |
| 178 | 厄瓜多尔 | true | - |
| 179 | 特立尼达和多巴哥 | false | - |
| 180 | 危地马拉 | false | - |
| 181 | 乌兹别克斯坦 | false | - |
| 182 | 洪都拉斯 | false | - |
| 183 | 萨尔瓦多 | false | - |
| 184 | 巴基斯坦 | false | - |
| 185 | 斯里兰卡 | true | - |
| 186 | 蒙古 | false | - |
| 187 | 越南 | true | - |
| 188 | 文莱 | true | - |
| 189 | 阿根廷 | false | - |
| 190 | 尼日利亚 | false | - |
| 191 | 柬埔寨 | false | - |
| 192 | 北马其顿 | true | - |
| 193 | 孟加拉 | false | - |
| 194 | 吉尔吉斯斯坦 | true | - |
| 196 | 马尔代夫 | false | - |
| 197 | 列支敦士登 | true | 原文空白 |

## 4. 本土完整规格表的 39 项站点

E24 的标题是站点币种/体积差异，没有“是否开站”或本 merchant eligibility 字段。下表保留原文 country/partition/siteId/base currency，**不能作为完整当前开放清单或 regionId → siteId 证明**。原表还包含 supportLanguageList、推荐零售价币种、重量/尺寸与范围、精度/价格范围；实际规则须绑定原完整官方版本。

| siteId | 官方 Region | 官方 Site Region | 基础价币种 |
| --- | --- | --- | --- |
| 100 | United States -US | US | USD |
| 101 | Canada-CA | GLO | CAD |
| 102 | United Kingdom-UK | EU | GBP |
| 103 | Australia-AU | GLO | AUD |
| 105 | Germany-DE | EU | EUR |
| 106 | France-FR | EU | EUR |
| 107 | Italy-IT | EU | EUR |
| 108 | Netherlands-NL | EU | EUR |
| 109 | Spain-ES | EU | EUR |
| 110 | Mexico-MX | GLO | MXN |
| 111 | Portugal-PT | EU | EUR |
| 112 | Poland-PL | EU | PLN |
| 113 | Sweden-SE | EU | SEK |
| 115 | Greece-GR | EU | EUR |
| 118 | Japan-JP | GLO | JPY |
| 119 | Korea-KR | GLO | KRW |
| 137 | Czech Republic-CZ | EU | CZK |
| 138 | Hungary-HU | EU | HUF |
| 139 | Denmark-DK | EU | DKK |
| 140 | Romania-RO | EU | RON |
| 141 | Bulgaria-bg | EU | EUR |
| 142 | Belgium-BE | GLO | EUR |
| 143 | Austria-AT | EU | EUR |
| 144 | Finland-FI | EU | EUR |
| 145 | Slovakia-SK | EU | EUR |
| 174 | Türkiye-TR | EU | TRY |
| 147 | Slovenia-SI | EU | EUR |
| 116 | Ireland-IE | EU | EUR |
| 148 | Lithuania-LT | EU | EUR |
| 146 | Croatia-HR | EU | EUR |
| 132 | Brazil-BR | GLO | BRL |
| 149 | Estonia-EE | EU | EUR |
| 114 | Switzerland-CH | EU | CHF |
| 122 | United Arab Emirates-AE | EU | AED |
| 120 | Saudi Arabia-SA | EU | SAR |
| 150 | Latvia-LV | EU | EUR |
| 117 | Cyprus-CY | EU | EUR |
| 124 | Norway-NO | EU | NOK |
| 152 | Luxembourg-LU | EU | EUR |

不按地理修正 Belgium/GLO、Türkiye/AE/SA/EU；本表与实际 app partition 不一致时不能携带 token 试另一个网关。

原规格的关键差异：美国 lb/in，其余本表为 g/cm；重量/尺寸均有精度与范围。JPY/KRW/NOK 价格类型列为 Integer，其他为 2 decimal points；币种并不代表可以统一用相同整数单位。加拿大基础价下限为 2.00，多数站点为 0.02，日本/韩国/挪威为 2、捷克/匈牙利为 5。保留原文规则与不一致，不由旧 TEMU pricing policy 或汇率计算替换。

2026-10-12 再次读取完整 15 列/39 行，机器可读摘录保存于
[当前 TEMU 规则数据](../../internal/marketplace/temu/rules/data/local-specifications.tsv)：
美国重量/尺寸均为 1 decimal point，范围分别 `[0.1,9999.9]` / `[0.1,999.9]`；
AU/NL/RO/BE 的尺寸最大值原文为 `9999999.9`，不同于多数非美国站点的 `9999.9`。
BG/FI 的重量原文为 `[0.1, 10000000)`，保留开区间。数据 hash 绑定完整列值，
不把 snapshot 数据校验当 merchant 资格或完整发布 ready；GE/AM 缺口保持。

## 5. 官方地区映射与交叉核对

E30 全部 235 行通过可见 DOM 表格读取，下列完整保留其中有 SiteID 的 87 行；另 148 行没有 SiteID，不补造可发布目标。地区名仅作标签，映射按原 RegionID/SiteID，允许多 region 对应同一 site。LOCAL 的 39 个 true 是该版本一般交易开放说明；SEMI 的开放值来自此地区/履约页，不能替代 E23 的 PA 半托发布字典。空白与 coming soon 均保留原义，不转换成 true。

| regionId | siteId | 官方地区 | 数据部署区 | SEMI 标记 | LOCAL 标记 |
| --- | --- | --- | --- | --- | --- |
| 83 | 100 | Guam (GU) | US | 空白 | 空白 |
| 164 | 100 | Puerto Rico (PR) | US | 空白 | 空白 |
| 211 | 100 | United States (US) | US | true | true |
| 219 | 100 | Virgin Islands (U.S.) (VI) | US | 空白 | 空白 |
| 236 | 100 | Northern Mariana Islands (MP) | US | 空白 | 空白 |
| 37 | 101 | Canada (CA) | GLOBAL | true | true |
| 210 | 102 | United Kingdom (UK) | EU | true | true |
| 12 | 103 | Australia (AU) | GLOBAL | true | true |
| 144 | 104 | New Zealand (NZ) | GLOBAL | true | 空白 |
| 76 | 105 | Germany (DE) | EU | true | true |
| 69 | 106 | France (FR) | EU | true | true |
| 98 | 107 | Italy (IT) | EU | true | true |
| 141 | 108 | Netherlands (NL) | EU | true | true |
| 186 | 109 | Spain (ES) | EU | true | true |
| 128 | 110 | Mexico (MX) | GLOBAL | true | true |
| 163 | 111 | Portugal (PT) | EU | true | true |
| 162 | 112 | Poland (PL) | EU | true | true |
| 191 | 113 | Sweden (SE) | EU | true | true |
| 192 | 114 | Switzerland (CH) | EU | true | true |
| 79 | 115 | Greece (GR) | EU | true | true |
| 96 | 116 | Ireland (IE) | EU | true | true |
| 52 | 117 | Cyprus (CY) | EU | true | true |
| 100 | 118 | Japan (JP) | GLOBAL | true | true |
| 185 | 119 | Republic of Korea (KR) | GLOBAL | 空白 | true |
| 174 | 120 | Saudi Arabia (SA) | EU | true | 空白 |
| 179 | 121 | Singapore (SG) | GLOBAL | 空白 | 空白 |
| 209 | 122 | United Arab Emirates (AE) | EU | true | true |
| 105 | 123 | Kuwait (KW) | EU | true | 空白 |
| 151 | 124 | Norway (NO) | EU | true | true |
| 42 | 125 | Chile (CL) | GLOBAL | true | 空白 |
| 119 | 126 | Malaysia (MY) | GLOBAL | true | 空白 |
| 160 | 127 | Philippines (PH) | GLOBAL | true | 空白 |
| 197 | 129 | Thailand (TH) | GLOBAL | true | 空白 |
| 165 | 130 | Qatar (QA) | EU | true | 空白 |
| 101 | 131 | Jordan (JO) | EU | coming soon | 空白 |
| 29 | 132 | Brazil (BR) | GLOBAL | 空白 | true |
| 152 | 133 | Oman (OM) | EU | true | 空白 |
| 16 | 134 | Bahrain (BH) | EU | true | 空白 |
| 97 | 135 | Israel (IL) | EU | true | 空白 |
| 184 | 136 | South Africa (ZA) | EU | true | 空白 |
| 53 | 137 | Czech Republic (CZ) | EU | true | true |
| 90 | 138 | Hungary (HU) | EU | true | true |
| 54 | 139 | Denmark (DK) | EU | true | true |
| 167 | 140 | Romania (RO) | EU | true | true |
| 32 | 141 | Bulgaria (BG) | EU | true | true |
| 20 | 142 | Belgium (BE) | EU | true | true |
| 13 | 143 | Austria (AT) | EU | true | true |
| 68 | 144 | Finland (FI) | EU | true | true |
| 180 | 145 | Slovakia (SK) | EU | true | true |
| 50 | 146 | Croatia (HR) | EU | true | true |
| 181 | 147 | Slovenia (SI) | EU | true | true |
| 113 | 148 | Lithuania (LT) | EU | true | true |
| 64 | 149 | Estonia (EE) | EU | true | true |
| 108 | 150 | Latvia (LV) | EU | true | true |
| 122 | 151 | Malta (MT) | EU | true | 空白 |
| 114 | 152 | Luxembourg (LU) | EU | true | 空白 |
| 175 | 153 | Serbia (RS) | EU | true | 空白 |
| 130 | 154 | Moldova (MD) | EU | true | 空白 |
| 134 | 155 | Montenegro (ME) | EU | coming soon | 空白 |
| 91 | 156 | Iceland (IS) | EU | coming soon | 空白 |
| 5 | 157 | Andorra (AD) | EU | 空白 | 空白 |
| 26 | 158 | Bosnia and Herzegovina (BA) | EU | 空白 | 空白 |
| 3 | 159 | Albania (AL) | EU | 空白 | 空白 |
| 116 | 160 | North Macedonia (MK) | EU | coming soon | 空白 |
| 102 | 162 | Kazakhstan (KZ) | EU | 空白 | 空白 |
| 159 | 163 | Peru (PE) | GLOBAL | true | 空白 |
| 45 | 164 | Colombia (CO) | GLOBAL | true | 空白 |
| 75 | 165 | Georgia (GE) | EU | true | true |
| 10 | 166 | Armenia (AM) | EU | true | true |
| 14 | 167 | Azerbaijan (AZ) | EU | true | 空白 |
| 208 | 168 | Ukraine (UA) | EU | 空白 | 空白 |
| 212 | 169 | Uruguay (UY) | GLOBAL | 空白 | 空白 |
| 126 | 170 | Mauritius (MU) | EU | coming soon | 空白 |
| 135 | 171 | Morocco (MA) | EU | true | 空白 |
| 57 | 172 | Dominican Republic (DO) | GLOBAL | 空白 | 空白 |
| 203 | 174 | Türkiye (TR) | EU | 空白 | true |
| 4 | 175 | Algeria (DZ) | EU | 空白 | 空白 |
| 156 | 176 | Panama (PA) | GLOBAL | coming soon | 空白 |
| 59 | 178 | Ecuador (EC) | GLOBAL | true | 空白 |
| 201 | 179 | Trinidad and Tobago (TT) | GLOBAL | 空白 | 空白 |
| 213 | 181 | Uzbekistan (UZ) | EU | 空白 | 空白 |
| 61 | 183 | El Salvador (SV) | GLOBAL | 空白 | 空白 |
| 187 | 185 | Sri Lanka (LK) | EU | coming soon | 空白 |
| 217 | 187 | Vietnam (VN) | GLOBAL | 空白 | 空白 |
| 31 | 188 | Brunei Darussalam (BN) | GLOBAL | coming soon | 空白 |
| 9 | 189 | Argentina (AR) | GLOBAL | 空白 | 空白 |
| 106 | 194 | Kyrgyzstan (KG) | EU | coming soon | 空白 |

交叉核对结论（2026-10-12，E24/E29/E17 均重新读取）：

| 差异 | 官方原文事实 | 本批处理 |
| --- | --- | --- |
| 地区编号 | US 211→100、CA 37→101、BE 20→142；83/164/219/236 也映到 100 | 用 exact regionId 查版本固定映射，再按 siteId 查规格；不把 regionId 强转 siteId，不反向猜唯一 region |
| Belgium 分区 | E24（01-26）为 GLO；E30（07-16）部署区为 EU | 分区字段保留冲突证据，E30 是较新的地区部署说明；原应用身份分区始终决定 HTTP route，E24 仅供价格/单位规格，不触发跨网关 fallback |
| 半托开放和 North Macedonia ID | E23（2025-09-29）与 E30（2026-07-16）的半托值不同；E23 North Macedonia=192，E30=160 | 两表属于不同接口/用途，PA 发布选择使用 E23/E13 的合同，不把区域订单页的 160 alias 为 PA 的 192，不用 E30 的 SEMI 行切换到区域商家身份 |
| 两个本土 39 项集合不同 | E30 LOCAL=true 含 GE 75→165、AM 10→166，但 E24 没有 165/166；E24 有 SA 120/LU 152，而 E30 LOCAL 栏为空白 | 分别保存开放说明、规格与授权事实；不能以条数相同声称已全部匹配，空白不是 true/false。GE/AM 的币种/单位规则仍需官方依据 |
| 发布规则不返回此规格 | E29 fieldRules 只有 filedName、requirementType（required/optional/disabled）；E17 V2 price.currency/amount 与 packageInfo 的四个数值均为 STRING，没有显式重量尺寸单位字段 | 不能由这个 API 的存在推定可动态查到 GE/AM 规格；不能默认 USD/lb/in、EUR/g/cm、法币或邻国规则 |

本土一般开放说明覆盖 39 个 true，其中 37 项在 E24 找到价格/单位规格、2 项缺少规格；E24 与 E30 本土集合的并集是 41 项。缺口不能作为擅自把 GE/AM 移出用户范围的理由；也不能用真实商家 mutation 试错。只有官方合同或真实只读官方资格/规则证据可以补齐受影响发布目标，当前未调用商家 API。

## 6. 验证状态

| 项目 | 状态 | 意义 |
| --- | --- | --- |
| 官方页面、迁移路由、三模式创建/读取/状态 API | PASS（文档读取） | 只证明本文件列出的合同事实，不是业务调用成功 |
| 半托 96 行/70 true/26 false、本土 39 行规格摘录 | PASS（文档读取） | 文档版本绑定；不证明所有 merchant 可售 |
| CN/PA 同授权身份/模式验证合同 | PASS（文档读取） | E03/E04/E11/E26；选 PA uniqueId，不要求 mallId 转换；设计增量复核已关闭 G1 BLOCKER，实际调用 NOT_RUN |
| 本土 regionId → siteId 与部署区 | PASS（文档读取） | E30 完整正文与 87 个映射行；精确映射，不是 merchant 授权 |
| 本土全站点价格/单位规格闭环 | UNKNOWN（GE/AM 2 项缺口） | 37 个 LOCAL=true 站点可关联 E24；GE/AM 无已读规格，SA/LU 的一般开放栏为空白；不冒充全覆盖 |
| externalGoodsId/outGoodsSn 关系与失响应 exact readback | UNKNOWN | 不做错误 adopt；原 UNKNOWN fence 保留 |
| 真实官方应用/真实 merchant 授权、发布、撤权、消费者状态 | NOT_RUN | 本轮未授权业务 API/真实数据 |
| 用户可操作产品/生产验收 | NOT_RUN | 已准入并开始店铺连接实现；未部署/未完成全范围，文档/开发自检/CI/AI评审不构成验收 |
