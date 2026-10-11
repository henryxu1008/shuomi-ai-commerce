# TEMU 官方接口与站点证据

> 对应 [#31 接入设计](temu-platform-v1.md)；收集日期 2026-10-11。
>
> 来源仅为 TEMU 官方页面，公开 partner 文档与用户登录后可读的 agentpartner 文档。
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
| E18 | [temu.local.goods.v3.add](https://partner.temu.com/documentation?menu_code=fb16b05f7a904765aac4af3a24b87d4a&sub_menu_code=419748d505a3483f8d210d978cb813f8) | 2026-09-16；更新记录含 09-28 | Local；官方自动类目/属性推断、图片 URL 抓取；result externalGoodsId/goodsId；不是全/半托合同，不能与 V2 约束互换 |
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
| E30 | [Model Site and Region](https://partner.temu.com/documentation?menu_code=38e79b35d2cb463d85619c1c786dd303&sub_menu_code=07938e60f95344a0a9444f524ea61e31) | 未读取正文 | 官方全站搜索 region 找到，归 Developer Guide → Order Management；点击跳到 GLOBAL 登录页。CN 登录不通用。已找到现成候选文档，正文待有权访问；不能写成官方没有此资料或已核实映射 |

### 文档限定的 router

| 身份分区 | transport route | 完整 URL | 适用说明 |
| --- | --- | --- | --- |
| CN | CN | https://openapi.kuajingmaihuo.com/openapi/router | 未迁移 CN API，含本轮 CN 身份接口 |
| CN | PA | https://openapi-b-partner.temu.com/openapi/router | E03/E26 已列举 migrated type 与当前 token 授权查询；不是任意 CN fallback |
| US | US | https://openapi-b-us.temu.com/openapi/router | 原 US 应用/token 与该 API scope |
| EU | EU | https://openapi-b-eu.temu.com/openapi/router | 原 EU 应用/token 与该 API scope；不是“地理欧洲全部” |
| GLOBAL | GLOBAL | https://openapi-b-global.temu.com/openapi/router | E24 用 GLO 标签，不能由“非欧美国家”推导全部路由 |

E02 的全/半托“发品 CN”与 E03 新迁移路由按具体 API/version 对齐，不能随意选择旧入口。E03 把 bg.goods.add 对应到 bg.glo.goods.add/temu.goods.add，新实现只固定已查证的一种 type；别名不形成第二发送机会。类别/属性/父规格/尺寸/图片/运费模板/品牌沿 E03 的 migrated API 选择，正式请求还须逐项验证 scope 与当前参数。

## 2. 身份验证结论与待读取的站点/恢复依据

- CN result.mallId 与 PA result.uniqueId 没有已读取的等值关系；这不需要成为实现前置。E04 店铺粒度 token + E11 PA mall 唯一标识 + E26 同一当前 token 的固定 CN 类型/PA 身份查询，支持选 PA uniqueId 为唯一 supplier 身份，CN 类型为该凭据下的附属验证事实。不拿名字关联，也不跨 US/EU/GLOBAL 试探；真实成功响应与权限仍在实现/授权验证层收敛。
- 本土 regionId（USA-211）与 siteId（US-100）值域不同。已找到 E30 Model Site and Region，但尚无 GLOBAL 登录，正文待读；不是断言官方没有映射。E24 规格字典没有实时商家资格。Local API 目标是当前 token 的 mall，不要求虚构一个“全部可售站点”授权 API；共享字典不能自证商家授权。
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

## 5. 验证状态

| 项目 | 状态 | 意义 |
| --- | --- | --- |
| 官方页面、迁移路由、三模式创建/读取/状态 API | PASS（文档读取） | 只证明本文件列出的合同事实，不是业务调用成功 |
| 半托 96 行/70 true/26 false、本土 39 行规格摘录 | PASS（文档读取） | 文档版本绑定；不证明所有 merchant 可售 |
| CN/PA 同授权身份/模式验证合同 | PASS（文档读取） | E03/E04/E11/E26；选 PA uniqueId，不要求 mallId 转换；设计增量复核已关闭 G1 BLOCKER，实际调用 NOT_RUN |
| 本土 region/site 规格关联 | UNKNOWN（待访问 E30） | 已找到官方 Model Site and Region；GLOBAL 登录后读取，不断言文档缺失 |
| externalGoodsId/outGoodsSn 关系与失响应 exact readback | UNKNOWN | 不做错误 adopt；原 UNKNOWN fence 保留 |
| 真实官方应用/真实 merchant 授权、发布、撤权、消费者状态 | NOT_RUN | 本轮未授权业务 API/真实数据 |
| 用户可操作产品/生产验收 | NOT_RUN | 无正式实现；文档/CI/AI评审不构成验收 |
