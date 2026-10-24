# 插件资料补齐与来源报价只读展示 V1

Refs #916；基线 [插件接收架构](plugin-capture-ingestion-v1.md)、[上架字段设计](../product/plugin-capture-listing-fields-v1.md)。
Status: **IMPLEMENTATION_READY**。2026-10-24 用户已授权先评审、后开发/测试并提交一个主要PR；不含合并、运行更新、真实重采/保存或部署。

## 1. 产品边界、依据与交付

使用者是已登录企业采集人员。用户在已有“我的数据 → 原始商品资料”查看插件保存的固定版本，能看到来源属性、当前商品图、来源报价及准确缺失提示。Product/Figma Authority沿[final-ui-ia-authority](../product/final-ui-ia-authority.md)和字段设计引用的Figma tg48P46SSXl6TBy9lZwg63 / 31:463；只扩展现有详情内容，无新页面/路由/菜单。

Must：原Collection实时授权及org+actor私有引用；准确publication/productKey/version；SRC报价只读且不猜币种/库存/SKU；不能用报价读取失败吞掉已知商品资料；范围内实际提取缺口与测试收敛。
Should：在原详情/原下载中可见报价，原始金额保留十进制字符串。
Out of Scope：批量、全规格族遍历、外链/跨frame详情、富文本、provider/worker/计费、写回已保存事实、回填/兼容/迁移、新权限/表、插件身份/权限/包名更改、任何运行实例操作。
Threat Model沿基线：客户端事实不等于授权；读取商品相关公开事实，不读取账号/凭据/页面全局；没有新的Accepted Risk或权限放宽。

## 2. 事实owner和现状

- SRC.PersistedPublication.Envelope.SupplierOrCostFacts.Facts中的price_fact.N是已持久化来源报价事实；Catalog保存产品/规格快照，Collection保存actor私有的固定引用。
- ToSnapshot不整体投影SupplierOrCostFacts，当前Collection详情只有item/product，因此已经保存的顶层报价未展示；variant的source.price.fact仍在原属性里。
- 本次不把报价复制进新表、Catalog属性或最新版本，不改变现有保存事务和hash。新增的sourceQuotes只是一次GET的有界读取投影。
- 审阅时基线main 12da6cc2856a693f2c58e8a28d6c2faf942d7de3；三条真实记录此前只读对比未发现SRC归档Snapshot与Catalog差异。本轮不再读取或修改保留实例。

## 3. 窄读取合同与调用路径

Collection新增只读SourceQuoteReader窄port，通过Service.WithSourceQuotes注入。port输入为服务端authorize所得Scope和Repository.ReadItem返回的Source；请求不接收外部publication/组织/actor覆盖。

路径：
原GET item详情 → 原PermissionRead/live授权 → 原actor私有ReadItem → 原Catalog.GetSnapshot固定版本和publication核验 → SourceQuoteReader → 当前SRC NewTransactionReader / Read → 小型报价DTO → 原BFF/schema → 原详情/下载。

app/productsourcing实现只读adapter，httpapi/product_collection_application注入。adapter用当前Product DB的READ ONLY / REPEATABLE READ事务复用sourcingpersistence.NewTransactionReader(tx,newCatalogBridge)；其既有Read验证SRC receipt/envelope/snapshot hash及准确Catalog绑定。无schema安装或新grants。不能调用PublishedAcquisitionReader.ReadPublished或BrowserCaptureService.ByKey来恢复/推进操作；它们不是此报价事实的读入口。

沿Issue398现有精确leaf登记方式，只为`internal/app/httpapi/product_collection_application.go`登记`NewCollectionSourceQuoteReader`；其它已受guard覆盖的consumer、subpackage及写入/provider/charge API保持原拒绝规则，不改CI或保护策略。

读取后逐一核验Receipt.OrganizationID==Scope.OrganizationID、ActorID==Scope.ActorID、ProductKey==Source.ProductKey、PublicationID==Source.PublicationID、CatalogVersion==Source.Version。当前member只由原Collection live授权约束，不追加旧member永久独占资产规则。

只投影已批准插件producer：browser_acquisition/v1或v2、amazon_browser_capture/v1；其它producer/own资料为not_applicable。不存在以解析失败切换producer/v1的兼容路径。

ItemDetail增加：
```json
{"sourceQuotes":{"state":"available","publicationId":"原item固定publication","version":"1","prices":[{"amount":"27.00","currency":null,"minQuantity":null}]}}
```

- state为available/unavailable/not_applicable，仅GET读取状态，不持久化，也不是新业务状态机。三个状态字段结构在Go/BFF中保持一致。
- available：publicationId和version必填，并与item.source匹配；prices允许空数组，明确表示本次证据没有商品级报价，不能解释成网页没有价格。
- unavailable：SRC缺失/校验失败/超时/报价格式无效/依赖未注入；不附任何价格，不回退latest，不显示“无报价”。已确认的item/product仍正常返回；原入口授权、item、Catalog读取失败仍沿原拒绝语义。
- not_applicable：own或非本次插件producer；不携带价格。无新自动重试或恢复owner。
- 可缺省sourceQuotes供未使用此装配的现有内部消费者；原HTTP装配必须注入。服务在适用插件分类但缺port时显式unavailable，不能伪造available空值。原BFF允许新字段可选以保留既有内部调用边界，不新增旧格式解释/转换。

sourceQuotes.prices字段来自sourcing owner的纯投影函数，复用现有AcquisitionPrice十进制/币种/数量校验：
- 仅读取price_fact.N，按数值N排序；最多256项。索引或JSON不合法、重复/未知price字段、金额/币种/数量不合现有合同→unavailable，不能静默丢项。
- amount保持原十进制字符串；currency为明确三字母代码或null；minQuantity为明确原数量字符串或null，不默认1，不推断数量单位。
- 非price_fact商业facts不透传；整份Envelope、metadata、内部scope/hash均不外发。既有2MiB envelope/snapshot预算保持，新增报价DTO编码最多64KiB、最多256项，字段沿现有金额/数量/币种正则。报价子读最多2秒且受原10秒父deadline约束；父请求不足以保留响应余量时直接unavailable。仅详情GET的Go handler、BFF及浏览器API总响应预算对齐3MiB，写请求/其它响应仍2MiB，不能截断已知事实。

事务/失败：全部只读，无发布/charge/补偿。并发新保存/归档不改变已选不可变版本；本次不新增跨请求撤权或归档协议，入口和仓库仍执行现有live/actor限制。响应丢失可重新GET原item，没有写入副作用。

## 4. UI及缺失语义

原详情加“来源报价（参考）”：显示原金额、明确币种或“币种待确认”、明确起订数量或未提供；提示不代表本店销售价/库存。available空数组显示“本次未记录商品级报价，规格报价见规格资料”；unavailable显示“来源报价暂未能读取”，不清空或隐藏其余商品资料。JSON下载自然包含该只读投影。

规格区域复用source.price.fact，使用同一严格价格结构显示“来源金额… / 币种待确认”，不继续把source.price.fact原JSON当产品文案。价格字段无效时保留缺失提示，不从别的规格复制。其它来源属性继续原文显示。

missingFacts/reasons复用现有结构，区分未观察/未加载、结构不支持、币种歧义、仅采当前子商品。不宣称网页绝对无值。品牌原文作为通用属性保留，本轮不新增Catalog.Brand映射或目标品牌ID推断；自家SKU、销售价和仓库库存仍归经营者/目标owner。

## 5. 同批提取补齐（复用v2，不变保存合同）

Amazon：
- 读取当前商品概览/详情表的明确键值；复用现有有界文本遍历、hidden/script排除、attributes/Zod。
- 当前#altImages中的已呈现商品图片与主图去重，保留main/gallery角色；排除视频封面和推荐商品，不按文件名改写原图URL或声称全图库完整。无明确规格关系的图只放商品级，不复制到全部规格。
- availability仅保存为来源通用属性，不转为Stock数量；继续selected_product和当前准确ASIN/尺寸/颜色。
- 币种只读明确与当前商品价格绑定的ISO元数据/无歧义符号；冲突记未知，不按站点默认USD，也不混当前售价和划线参考价。

1688：
- 从当前acquisition/browser/script.go选择性EXTRACT报价模型优先级：offerPriceRanges → orderParamModel.orderParam.skuParam.skuRangePrices → priceModel.currentPrices。只有高优先级未提供可用事实时选择下一模型，不混合重复来源或猜促销价。
- 保留jsonc-parser有界字面量解析、原数值词法十进制、SKU属性与图片准确关联。币种仅明确值，仍不默认CNY。
- #desc-lazyload-container等已有明确正文容器只读已呈现安全文字子树，跳过script/style/iframe/input/form/隐藏子树，不因无关script/iframe兄弟节点丢整段安全正文；不执行或跨frame读取，不发detailUrl网络请求，不把SEO/meta摘要当完整描述。
- 缺skuId与卖家SKU含义不混淆，不用货号/skuId生成SKU。未知其它页面模型不靠猜测扩张本轮范围。

成熟组件：继续jsonc-parser/Zod、原SRC/Catalog/Collection及事务reader，不添加解析框架、报价owner、表或恢复协议。Legacy decision: EXTRACT；Reusable behavior为当前合法选择器/报价优先级/stockState识别；Current owner为扩展与现有sourcing/Collection；不依赖或包装legacy service，不复活默认币种/float丢精度行为。

## 6. 必要验证与交付

设计评审只覆盖本次只读合同与上述边界；发现finding先依AGENTS分类，最多两轮正常评审，无BLOCKER后IMPLEMENTATION_READY才修改生产路径。

实现TDD：先让新增Amazon属性/图库/供货、1688补充报价模型/安全正文回归失败，再最小修复。复用原extension、Go sourcing/Collection、BFF/schema与详情测试：准确scope/publication/version、拒绝错配SRC事实且已知product保留；报价按原十进制/币种空/数量未知投影；图片归属/未知不伪造；无新增写入或provider路径。复用既有测试夹具和测试DB策略，不连接保留数据库。

最终稳定候选再同步main，处理真实组合问题，独立检查实际diff和路径，必要CI绑定准确HEAD；一个主要PR同时承载设计和实现。不合并、不关Issue、不装载插件或更新实例，真实产品验收/重采/运行部署NOT_RUN。

## 7. 独立设计评审记录

2026-10-24，独立Reviewer amazon_fix_review核对设计SHA `fe6670e5b49d75ca346979804194ca6742b07271`、#916正文及实际Collection/SRC/HTTP/BFF路径，结论 **IMPLEMENTATION_READY**，无BLOCKER。该结论为设计准入，不代表实现或产品验收。

- Finding：Go HTTP与BFF的详情响应预算需一致。Product requirement affected：附加报价不得吞掉已知商品。Classification：IMPLEMENTATION_TEST。Action：按第3节仅扩大详情GET响应至3MiB，测试接近原上限的商品加报价完整返回。
- Finding：报价子读需早于外层10秒结束。Product requirement affected：报价失败仍显示已知商品。Classification：IMPLEMENTATION_TEST。Action：第3节2秒子预算与父deadline余量检查，测试超时降级为unavailable且没有后台重试。

以上为评审指出的实现收敛，未改变事实owner、权限、状态、事务或恢复协议；设计基线在正式实现前冻结。

