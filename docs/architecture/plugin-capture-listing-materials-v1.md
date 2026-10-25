# 插件上架资料补齐：高清相册、参数、正文/详情图与重量 V1

Refs #916；基线 plugin-capture-ingestion-v1、plugin-capture-source-quotes-v1。

Status: IMPLEMENTATION_READY。Design Basis: Reuse Existing Architecture。独立 Reviewer initial_revision_final_review 已核对设计提交 339dee012fd4cdc5068a62ec20c8e8ff47533191、Issue #916 当前范围、两平台解析/validator 和 Go v2→Envelope→Catalog 映射，确认现有合同可承载本轮资料，无设计 finding；准入记录先于生产代码修改。此结论不代表用户验收或运行更新授权。

## 用户结果与边界

2026-10-24，协调会话 01a16741-448c-78a0-8f3e-fa4d39b17516 用户授权：补齐 Amazon/1688 插件高清相册、商品参数、正文/详情图及重量，参考旧服务端有效提取规则；设计评审后开发、测试、提交一个主要 PR。不合并、不更新实例/现有插件、不重采、不部署；保留数据、配置、加载目录、插件身份，ProductAgent/Chat/服务端采集保持关闭。

产品/Figma Authority：上述本轮决定、docs/product/plugin-capture-listing-fields-v1.md、docs/product/final-ui-ia-authority.md，Figma tg48P46SSXl6TBy9lZwg63 / 31:463。复用现有原始资料详情和入口，无路由、布局、命名或交互更改。

只补当前页已经提供的事实。Amazon 保持 selected_product；相册只升级当前已观察条目，不遍历其他 ASIN/隐藏图库。1688 保持现有已定位 context/SKU 模型，不新增 __INIT_DATA 解析或跨页读取。供应商/物流服务、评分销量、视频、结构化尺码表、完整规格族、目标分类/品牌 ID、本店价格库存、AI 均在范围外。没有新增 Accepted Risk。

## Owner 与复用链路

源页受信入口 → 原平台 captureDocument/captureAmazonDocument → 原 v2 strict validator/hash → 原 Controller/Handoff → 原 receiver/BFF/产品授权 → 原 SRC/Catalog/Collection 事务 → 原详情与原键恢复。

改动只在 extensions/product-capture 的纯提取和小型内部辅助函数、回归夹具及依赖声明。商品参数使用 attributes，正文使用 description，详情图片使用已有 images 的 detail role；重量作为有来源名称的原始参数，绝不写 Stock/销售价/未经证实的标准重量字段。不新增公共 wire、parserVersion、schema、持久化事实 owner、权限、状态机、事务、幂等键、恢复协议或保存重试。原操作和固定版本从未被新 DOM 重建或回填。

## Amazon 高清相册

只读当前文档内联 script 文本中的 colorImages.initial 数据。已观察形态为 initial: A.$.parseJSON('JSON数组')；同时容许直接静态数组。调用名称只用于验证已知包装，不执行调用、eval、Function、页面 global 或网络。Acorn（锁文件已有开发间接依赖，实施时显式声明直接生产依赖）静态解析 JS，jsonc-parser 以严格 JSON 模式检查字符串内数组及重复键；不自制 JS 解释器。

候选数组必须包含 variant=MAIN 且 hiRes 或 large 与当前已准入主图 URL 完全相同的记录；再用每个当前可见相册 thumb URL 精确关联同数组条目。多组产生不同映射、同 thumb 不同地址时不升级并给 missingFacts，不能任取第一组。其他颜色、推荐图、视频记录不进入相册。只使用明确 hiRes，其次原 large；不替换文件名尺寸后缀、不下载探测，不将 large 冒称已验证高清。主图优先、按原相册顺序去重；首缩略图映射回主图时合并。图库不自动归给规格；当前规格继续只有原选中主图。

候选缺失/语法不支持/关联不成立时，保留已有准入的主图和缩略图并报告 images.hiRes 缺失原因，不吞掉已知资料、不自动重试。所有 URL 仍执行 publicImage、2048 字节限制及当前服务端图片防护，不去除 query 或放宽域名/凭据保护。

## 参数、正文和详情图片

Amazon 增补旧有效详情选择器 technicalSpecifications_section_1；已有三种表保留。仍仅当前商品表的两格 name/value，冲突同名值省略并提示，隐藏/脚本/表单节点排除。Item Weight/重量/产地/型号等保留来源原文，不推断单位、净重/毛重或目标枚举。

2026-10-24，#916 用户批准修复 addToCart 表单祖先误伤：仅原已定位的价格 `.a-offscreen` 与 `#availability` 节点显式允许读取自身安全文字子树，忽略字段之外的 form 祖先；字段内部的 form、input/textarea/select/button、script、隐藏/推荐/其他商品子树仍排除。复用原有界文字 walker 和预算，不读取表单整体或控件值，不全局放开其他正文/参数/图片的表单排除。既有价格 root 内的 priceCurrency meta 校验沿同一受限祖先规则保留，继续排除隐藏/推荐元数据，不从表单其他位置取币种，避免恢复金额后漏掉原币种或冲突；不增加元数据来源。价格十进制、划线价、币种歧义与原 v2 保存/恢复合同不变。

Amazon 正文追加旧有效 A+ 商品模块段落（#aplus .aplus-module p、#aplus_feature_div .celwidget p、.aplus-v2 .aplus-module-content p），按源顺序提取有界纯文本并去除重复段落，保留已有正文/卖点。详情图片仅当前 productDescription/A+ 根内已声明的有效 img URL。#916 的 2026-10-24 懒加载修复仅由 Amazon 调用显式启用：依次尝试同节点 data-old-hires、data-src、src，取首个通过原 URL 准入且不是已观察 grey-pixel.gif 占位路径的原始地址；仅占位节点省略，不猜测或改写文件名，不泛禁正常 GIF。排除隐藏、视频/表单/推荐区的规则先于候选读取；每个尝试候选仍检查原 2048 UTF-8 字节限制，集合/总项/节点/时间/载荷预算不变。1688 默认选择保持不变，已保存版本不回填或覆盖。不读取整页 body、外链或 iframe，不抓富文本/执行脚本。

1688 保留原 featureAttributes；补当前 #desc-lazyload-container 内两格参数表，与原参数按名称合并，同值去重、冲突省略并提示。该容器正文继续安全纯文本；增加该容器中有效 img，role=detail，图库已有 URL 不重复，不把详情图铺给所有 SKU。

1688 从原已解析 context 的 result.data.productPackInfo.fields.unitWeight、shippingServices.fields.unitWeight 分别读取来源包装/来源物流重量。只接收静态字符串或原始数值词法，精度不经过 Number；两种来源分开命名，不覆盖或互相替代。字段自身未提供单位时名称明确“单位未提供”，并记录 missingFacts；不默认克/千克，不换算、不假定标准包装。若参数表含明确重量单位则原文保留，与上述原始模型值不混同。

### 2026-10-25：1688 开放 Shadow DOM 详情增量

Increment status: **IMPLEMENTATION_READY**。独立 Reviewer `1688_shadow_design_review` 已核对设计 HEAD `e1e07ef48ac023357c6b67c26a5a6b3ce25a7164`、#916 当前范围、TS/BFF/Go/httpimage 链路及用户提供的三个离线工件，无设计 BLOCKER；宿主外层视频过滤、v2 顶层 detail 专属准入和直接载荷 2048 UTF-8 URL 限制作为 IMPLEMENTATION_TEST 落实。用户批准先独立设计评审，再开发、测试并提交一个主要 PR；不合并、不更新插件/实例、不重采、不部署。Design Basis: **Reuse Existing Architecture**，沿用上方当前 owner、v2 字段和保存/恢复链路；此准入先于正式生产代码修改，不代表真实商品或生产验收。

用户指定 Pi 会话提供的商品 917086279832 离线 DOM 显示：`#description[data-module="od_product_description"] .module-od-product-description .html-description` 的 open `shadowRoot` 内 `#detail` 承载详情。宿主本次为 `v-detail-p`，标签名不作为契约。39 个 img 中 34 个声明 src、5 个为空；34 图均 `loading="lazy"`，没有 data-src，非空白文字节点为 0。其中 30 条仅含 `__r__=13位十进制数字`，原插件 query 限制会拒绝它们。此为用户提供存档的结构证据，不冒充本会话实时源站读取或历史采集时就绪状态。

**详情根与过滤**：仅在当前已校验商品文档中定位上述明确宿主；先按现有主文档子树排除筛选宿主，合格现代宿主只有一个时，显式进入其 open root，且只接受其中唯一 `#detail`。有合格现代宿主但 root/内容未准备或根歧义时，保留已知图库/SKU 等事实并用现有 missingFacts 说明 `images.detail` 未就绪/不明确，不任取一个根。没有合格现代宿主时，已支持的当前页面 `#desc-lazyload-container` 形态仍使用同一解析 owner；多个候选不任取。两种页面形态不建立旧 Service、数据或状态兼容层。

参数表、正文和详情图共用选定根，继续复用两格参数、纯文本 walker、图片去重和预算。候选宿主/图片集合在遍历前沿用 256 上限；新遍历计入原总节点、8 秒及载荷预算。必须同时检查宿主到主文档的祖先与 Shadow 根内候选的祖先，保持 hidden/form/推荐/其他商品/视频排除，不能因 parentElement 在 Shadow 边界结束而漏查宿主外层。禁止泛扫所有 ShadowRoot、读取 closed root/iframe、整页 body、页面 globals、Cookie/storage 或外链。不增加滚动、等待图片字节、固定睡眠、额外请求或新恢复状态。

**URL 例外**：通用 `publicImage`、Amazon 和 1688 v1 保持原规则。仅 1688 v2 顶层 `images` 中 `role=detail` 的提取及 strict payload 校验共用窄准入：原无 query URL 按既有规则；带 query 仅接受 HTTPS 精确主机 `cbu01.alicdn.com`、路径 `/img/ibank/` 下且原始 query 严格为单一 `__r__=[0-9]{13}` 的地址。拒绝额外/重复/编码变体参数；userinfo/端口/fragment/敏感路径/控制字符、公网主机及 2048 UTF-8 字节限制不变。图片原始 URL 连同 query 完整保留，不删参数、改文件名或网络探测。图库、SKU 图片和其它 role 继续原 URL 准入，不能借 detail 标签扩展规格图片合同。

接收 BFF、Go evidence 映射和现有 httpimage 已可承载这些原始 HTTPS URL，不改变后端、schema、下载器或 SSRF 防护。本轮不调用任何下载能力，不宣称真实图片预览成功。

**事实与回归**：纯图片详情的 description 保持空，可在现有 missingFacts.reason 说明无文字节点，不用标题/meta/程序说明或 OCR 填充。新图仅追加 detail role，不铺给 SKU；保留原图库与规格关联、报价、参数和完整 URL 去重。三个入口继续共用 runCapture；交接、Product 事务、保存幂等、原键/原标签恢复不变，不回填 Amazon v5/v6 或 1688 v1/v2。

获准开发后，先用受控 JSDOM 重建用户提供的结构与有界图片样本复现失败，再最小实现；覆盖 34 src 图/5 空图/纯图无文字、30 条合法 query、宿主外层及根内排除、歧义/未就绪、集合/字节/时间边界和图库/SKU/Amazon/v1 sibling paths。现有保存映射证据复用，源码测试不替代真实重采/保存验收，不新增验收平台。

## 资源与失败边界

### 2026-10-25：Amazon 当前商品的副标题与列表参数

本增量状态：IMPLEMENTATION_READY；Design Basis: Reuse Existing Architecture。独立 Reviewer 1688_shadow_design_review 已核对设计提交3a4af62f5b44c32b110a35d9549cfa69edba80d5、#916、已有AX证据及TS/Go链路，无设计级BLOCKER。标题/列表不同链接准入、唯一分支/顶层li/首次冒号与冲突、原字节/节点/时间预算三项IMPLEMENTATION_TEST由本批回归收敛。#916 用户在缺口分析后明确同意先补当前选中 ASIN 的标题/副标题、列表参数与来源重量；先设计评审，再开发、测试、提交一个主要 PR。完整规格族另行设计。本增量不授权合并、原实例/插件更新、重采或部署。

版本7采集前已观察的 B0BDLGZCTY 页面中，titleSection 内主标题之外还有明确副标题；detailBulletsWrapper_feature_div 下的列表包含 `Product Dimensions: 44 x 32 x 6 inches; 7.96 pounds`、`Item model number: XL`、制造商、首次上架日期等。已保存资料只有 overview 表格参数。这里引用已取得的 AX 容器/列表/文字证据；回归使用对应结构型夹具，不冒充重新取得的完整网页 HTML。本轮浏览器未保留可用的原标签连接，不为源码实现重采或另开真实采集任务。

**标题**：保留原 productTitle 读取。仅当唯一、未被原规则排除的 titleSection 包含该主标题时，读取该区域中主标题 heading 之外的可见安全文字；排除链接、按钮/控件、隐藏、表单、推荐、其它商品及已知品牌/评论辅助区。按不包含主标题的相邻内容分支归组，只在有一个确定的、非重复副标题值时以 `主标题 | 副标题` 拼接；多个不同候选保留主标题并用 missingFacts 说明歧义，不任取一个。区域不存在则保留原行为。新增遍历计入原预算，完整标题仍最多2000 UTF-8字节，不截断。

**列表参数与来源重量**：只读取上述明确 wrapper（及其已知 detailBullets_feature_div 子区域）的顶层 li。复用有界安全文字读取，按显式冒号/全角冒号分离名称和值，清理显示用双向控制符和空白；无明确名称/值的项省略。排除嵌套列表、按钮/控件文字与原子树排除；排名/评论行不作为本轮参数。列表普通链接沿用原 excluded 准入，保留合法 Manufacturer 等参数的安全可见文字，但不跟随 href；其它商品链接仍排除。标题仍排除所有链接文字，两者不共用过宽拒绝规则。每个保留项名称使用 `Product details · 原名称`，该前缀标识来源区域，仍是原 attributes 的普通 name/value，不新增模型或事实owner。同来源同名同值去重、冲突按原 putFact 省略并提示。

该来源前缀用于保留两个不同来源的原文：overview 中 `44L x 32W x 6.5Th` 不被 details 的 `44 x 32 x 6 inches; 7.96 pounds` 覆盖。后者完整作为来源参数保存，重量和单位仍可见；不拆成未经源标签明确的净重/毛重/包装重，不换算、不选优，也不据此填标准重量字段。已有两格表参数的名称、合并/冲突行为保持。

当前 owner 仍为插件 Amazon 纯解析；新 helper 仅服务同一 captureAmazonDocument，三入口共用原提取→严格v2校验/hash→交接→receiver→SRC/Catalog/Collection保存链路。wire、parserVersion、schema、权限/租户、状态机、幂等/恢复、持久化与外部副作用均不改变；1688不调用该helper。只借鉴旧代码正确的具体wrapper定位，不引入旧包/DTO/Service、整页body兜底或排名逻辑。币种推断、SKU/库存编造、展开/遍历其它ASIN、完整规格族及新UI均不在范围。

验收：先RED复现主副标题分段、可见列表参数/含重量原文及overview不同尺寸并存；再最小实现。覆盖排除/歧义、同来源冲突、空项/双向符号、集合/字节/节点/时间预算与旧标题/表格/Amazon相册和1688 sibling不变；原Go纯v2映射测试补证完整标题/来源参数保真。只用现有Vitest/JSDOM、原typecheck/lint/fixture构建与纯映射检查，不创建runner/验收系统。真实重采、保存、图片字节及运行验收NOT_RUN。

沿用最终载荷 2 MiB、集合最多 256、总项 1024、描述 8192 UTF-8 字节、URL 2048 字节、8 秒等原限制。新增 DOM/AST 遍历使用现有级别的总节点预算 65536、深度 32；最多 256 个通过现有子树排除且包含 colorImages 的相册候选内联 script，候选文本累计最多 2 MiB，先限输入再解析。无关脚本仍计入原遍历节点与时间预算，不占相册候选名额；语法不支持的候选仍计数。此口径由 #916 的 2026-10-24 用户修复决定明确，纠正 #933 将整页内联脚本数量误作相册候选数量的阻断。参数/段落/图片均在累积前计数、限字节。超限按原 CAPTURE_TOO_LARGE/TIMEOUT 停止，不截断冒称完整。

新图片元数据语法不支持可以明确保留原有相册；资源超限、来源变化和严格 payload 校验失败仍拒绝本次捕获。正文参数不执行页面脚本，不读账号/cookie/storage，也不调用来源网络。所有入口沿同一平台提取器，无额外按钮、持久缓存、重放或副作用 owner。

### 2026-10-25：同商品服务端对照后的参数、相册与报价文字增量

本增量状态：**IMPLEMENTATION_READY**；Design Basis: **Reuse Existing Architecture**。独立 Reviewer 1688_shadow_design_review 已核对准确设计a42b2ab1bc5bc589734baed01d6211da5eaff6d6、#916、真实TS/Go合同及只读投影，无设计BLOCKER。局部折叠豁免/同根ASIN、无thumb/唯一MAIN/整组冲突、严格报价模板/参考价隔离三项IMPLEMENTATION_TEST在同一#945回归收敛；此准入记录先于新增生产代码。用户核对 B0BDLGZCTY 旧程序 JSON 与插件版本7的34字段/28参数后，明确决定“设计通过后开发并维护 #945”。继续由原唯一 Writer 和 worktree 连续交付，维护同一主要 PR；不合并、不更新实例或现有插件、不重采、不保存、不部署。ProductAgent、Chat、服务端采集不由本增量启动；已有版本、配置、加载目录和插件身份保留。产品依据为该决定及既定字段清单，UI/路由不变，原 Figma authority 保持。此前各增量的 Ready 不替代本段独立增量检查。

**已取得实证**：原 Edge 商品标签1016968462可恢复只读访问；未刷新、点击、执行采集或更改其选择。`#productDetails_feature_div #prodDetails #productDetails_expanderSectionTables` 下五个可展开区域（Item details / Measurements / Features & Specs / Materials & Care / Style）的 `table.a-keyvalue.prodDetTable` 共28行，位于左右 depthSections 中。每个 `.a-expander-container` 有可见 `a.a-expander-header[role=button][data-action=a-expander-toggle][aria-expanded=false]` 及直接 `.a-expander-content.a-expander-extend-content` 子节点，后者仅以 `display:none` 折叠；ASIN行明确为当前 B0BDLGZCTY。当前 Item Weight 是7.94 pounds，旧导出为3.61 kg，不能跨时点换算或选择其一。相册 `colorImages.initial = A.$.parseJSON('…')` 的9条MAIN/PT记录绑定当前主图，PT06–PT08确实没有可见缩略图；另一个colorImages对象是其它规格集合，不可并入。主报价根仍绑定本ASIN，`.a-price .a-offscreen` 当前为空，`#apex-pricetopay-accessibility-label` 提供 `$39.98 with 15 percent savings`；同根 `.apex-basisprice-offscreen-label[data-basisprice-label]` 提供 `List Price: $46.99`，没有明确币种元数据。

**参数**：仅新增上述唯一、结构匹配的详情根和左右 depthSections；不使用全页 `table.a-keyvalue`。先验证允许的展开结构、同根安全ASIN行与当前 sourceID 一致，缺少/冲突/重复根均省略该可选增量并在原 missingFacts 说明，仍保留已知资料。复用 `tableFacts/productText/putFact`；每行仍仅两个单元格，排除排名/评论、控件、脚本、推荐、其它商品及嵌套表。名称继续 `Product details · 原名称`，保留重量、承重、厂家零件号的原标签/单位，不把它们变成净毛重、仓库库存或商家SKU，不散布到规格。

本增量明确区分展示折叠与内容禁读：只有上述已验证结构中、拥有安全可见展开头的直接内容节点，才允许局部忽略它自身的 `style.display=none`。`excluded` 的其它调用保持原行为；通过显式可选节点参数仅供新 Amazon 详情路径使用。父/子其它节点的display:none、visibility:hidden、hidden、aria-hidden、表单、控件、推荐和其它商品过滤全部保留；不忽略任意隐藏祖先，不修改源DOM或自动点展开。该局部扩展不能被笼统当作原可见规则已覆盖。

**完整当前相册**：复用 Acorn/jsonc-parser 和既有字面量准入，仅当 `initial` 内有唯一MAIN记录与当前已观察主图的显式hiRes/large一致时，接收该同组声明的MAIN/PT图片记录；无需每条都有可见thumb，不能借此接受其它颜色组、VID或全页hiRes。显式hiRes优先，缺失时保留原large，绝不拼尺寸后缀。保留现有DOM缩略图顺序，再追加同组未出现图片并去重；已选规格仍只有原已绑定主图。多个绑定组的完整记录冲突时沿原提示/保留原图库语义，不任取组。没有缩略图但有明确主图绑定也可读取同组；没有主图绑定不补图。原解析/集合/字节/节点/时间超限仍拒绝，不截断。

**主报价文字与参考原价**：仍限定原两个corePrice根，并拒绝其中明确声明不同ASIN的根。保留原当前价节点与划线价排除；只增加同根 `#apex-pricetopay-accessibility-label` 的安全文字候选，用已实证的 `data-pricetopay-savings-label="{priceToPay} with {savings} percent savings"` 识别并剥离其节省比例后缀；剩余金额仍走原十进制词法/歧义/币种规则，不计算折扣或读取整块报价/控件。旧文字与新标签冲突仍不选优。币种继续只用同一当前价根的原ISO元数据或无歧义符号，不增加来源、不按站点或裸$猜USD。

参考原价只从同根已实证、未被过滤的 `.apex-basisprice-offscreen-label[data-basisprice-label="{label} {price}"]` 读取完整安全文字；仅明确List Price及有效原金额保留为通用属性 `Source reference price`，同名冲突沿putFact省略并提示。它不进入priceFacts、variants.price或只读sourceQuotes.prices，不覆盖现价，也不成为本店售价；不读取aria-hidden的视觉重复价、价格说明弹窗、其它卖家、单位价或推荐价格。当前报价无金额时，参考价也不得兜底成现价。

**复用与不变量**：当前owner仍为扩展Amazon纯解析，三入口→captureAmazonDocument→原严格v2/hash→交接/receiver→原Product/SRC/Catalog/Collection事务与原键/原标签恢复不变。新增 helper 只在该提取器装配；wire/parserVersion/schema、权限/租户、事实owner、状态机、事务、幂等/恢复和外部副作用均不改。Legacy decision EXTRACT：只借鉴旧合法字段/相册来源和明确参考价信号，保留成熟解析组件；不引入旧Go包、全页正则/表格、默认币种或旧Service。

**验收**：设计通过后先RED复现26非指标参数（含18个版本7缺项）、5个合法折叠组及ASIN绑定、任意隐藏/控件/推荐/错误ASIN/重复根拒绝；9记录/6缩略图及无缩略图、MAIN/PT/VID、冲突组/large原值/预算；空旧价格节点+当前价标签、同根List Price与未知币种、现价冲突/只有原价/其它ASIN及原表单过滤。再最小实现；改动原测试中“同组未观察thumb必排除”的旧断言，而非删除其它有效归属/过滤断言。复用现有Vitest/JSDOM、typecheck/lint、fixture-only build和原Go纯v2映射验证，不新建runner或连接保留实例。结构夹具须标明源自只读投影重建，不冒充完整原HTML；真实重采/保存/图片字节与产品验收仍NOT_RUN。最终同步一次实际main、集中独立交付复核及准确HEAD CI进入同一#945。

**规格族单独设计，暂不实现**：已有partial_family/parentASIN/variants合同可作为候选；只读取绑定当前子ASIN的明确父子/维度映射，不以9×19笛卡尔积造规格，不复制当前价/库存/主图到其它ASIN，不把厂家零件号当SKU。继续原256集合/1024总项等限制，不截断冒称完整；旧JSON的171ASIN不是逐规格价格、库存、图片已齐证明。此方向尚无该增量的设计准入，不进入本轮生产修改；视频、榜单/评价、新类目或供应商owner、新UI同样在本轮范围外。

## Legacy decision

Legacy decision: EXTRACT。
Reusable behavior: 旧 Amazon ImageExtractor 的显式 hiRes/large 及 Description/ProductDetails 选择器，旧1688 PackInfo/Shipping 的明确字段路径。
Current owner: extensions/product-capture 平台解析；保存事实继续由现有 Product/SRC/Catalog/Collection owner 持有。
Cutover/deletion condition: 插件纯解析回归及原 v2 映射验证后使用新提取逻辑；无旧 Go 包/DTO/Service 依赖，不修改另一会话的退休代码。不复制旧全页正则、尺寸后缀猜测、默认币种/库存/包装或跨来源值兜底。

## 验证与停止点

TDD 先 RED：双平台原 v2 输出的高清映射、字段顺序变化和 JSON 字符串包装；当前商品绑定、冲突/视频/推荐排除、主图去重/规格归属；新增参数与 A+ 去重/安全正文；详情图与原始重量精度、单位未知、资源边界。再最小实现 GREEN。复用现有 Vitest/JSDOM、typecheck/lint、插件打包和原保存映射测试；不新建 runner/验收平台，不连接保留数据库/浏览器/实例或下载图片。

稳定候选同步一次真实 main，集中最终独立复核实际 diff 和有效证据，准确 HEAD 必需 CI 留在同一主要 PR。真实高清资源可用性/像素、两商品重采保存、运行更新与用户验收均 NOT_RUN。Issue 保持 OPEN，不合并、不部署。

## 1688 计价单位、商家名称、发货地及规格可订数量增量

Design Basis: **Reuse Existing Architecture**。本增量由用户于 2026-10-25 明确批准，执行 Issue #916：先独立设计评审，再开发、测试和提交一个主要 PR。目标是经营者在原插件确认及已保存商品资料中查看这四项来源信息；不增加 UI、路由、供应商 owner 或库存合同。Product/Figma Authority 沿用本文及原 ingestion/source-quotes/listing-fields 设计；三个入口继续共用 `captureDocument`。该增量单独于 Amazon 币种 PR #949，历史交付和原运行实例不变。

用户自行运行旧导出入口产生 `1688_product_917086279832_20261025_182717.json`，与插件 v4 的冻结 Envelope/Catalog snapshot 离线对照：计价单位为“件”，有公司名称、发货地及 35 个规格的可订数量；插件当前未读这四项。已有 6 个属性、46 个图片 URL/role 及 35 个规格的金额/图片关联保存前后一致，缺口不在保存链。两次采集时间不同，只借鉴有效字段规则，不把旧导出数量回填旧版本。旧端固定 CNY、生成 SCRAPED 编码、将未知单位重量标克等行为不得抽取。

| 同一已核实 offer 的明确来源 | 现有 v2 保存位置 | 显示名称与边界 |
| --- | --- | --- |
| `result.data.productTitle.fields.unit` | 商品 `attributes` | `来源计价单位`；“件”只作计价/计数单位，不是重量单位 |
| `result.data.productTitle.fields.shopInfo.companyName`；缺失/空时仅取同一 shopInfo 的 `authCompanyName` | 商品 `attributes` | `来源供应商名称`；公司显示名称优先，不读取 sellerLoginId/账号、联系人或新建 Supplier |
| `result.data.shippingServices.fields.location` | 商品 `attributes` | `来源发货地`；只取明确位置，不抓整页地址或推断经营地址 |
| 当前 `skuModel.skuInfoMap` 每项的 `canBookCount` | 商品 `attributes`，每个明确原 sourceID 一项 | `来源可订数量（规格ID <原sourceID>）`；绑定同项 `skuId`，不写规格维度或 Catalog `Stock`，不复制到其它规格、不从总量分摊 |

当前 owner 是 `extensions/product-capture/src/extractor.ts` 的 1688 纯解析；只读取已有静态 AST/JSONC 树的精确字段，不执行脚本、读 globals、发网络请求或采集登录资料。已存在的 offerId 校验先于读取；无关组件、其它商品、推荐节点、顶层同名字段均不能作来源。

商品文本为可选字符串，空白/缺失/null/非法类型不生成属性或默认值，其余按现有 `putFact` 规范化。商家名称只按表中明确的优先级选择，不能合并两个名称或用货号/品牌/登录名代替。数量只接受非负十进制整数字面量（包含明确 `0`）；数字节点从原文本切片读取，字符串规范化首尾空白后验证，保留精度、不经 `Number`/`parseInt`，不接受负数、小数、指数、单位词、布尔或对象。不把缺失/空/非法数量转成零；可选值不可用时仍保留该规格原有 ID、维度、报价和图片。

评审核实当前 TS/Go v2 都以完整 `variants[].attributes` 判重复组合。可订数量是来源观察值，不能成为维度、帮助重复维度通过准入，或使 sourceID-only 规格因数量相同而被误判重复。因此数量按上表放到商品来源参数，`variants[].attributes` 完全保留原维度读取，不修改公共组合语义、不新增前后两套准入。只有同一 `skuInfoMap` 项的明确非空 `skuId` 可作为关联；缺 ID 不用下标、名称或笛卡尔积造关联，省略该数量并留下原 missingFacts/warnings 形态的诊断。原仅有 sourceID、无维度的合法规格仍可保留数量；数量值变化不改变规格维度或准入结果。当前 UI 仅在已有参数列表按原规格 ID 显示，不承诺新增规格库存列。

所有新增来源属性复用 `putFact` 的同名去重和冲突省略规则，冲突沿原 `attributes` 诊断，不覆盖已有属性；数量标签包含对应原规格 ID，不影响其它规格数量。无新增字段时原 payload/golden 和维度读取不变。新增值计入原 v2 的单值 8192 UTF-8 字节、每集合 256、总项 1024、2 MiB 及原 65536 节点/深度 32/8 秒预算；不得截断绕限或扩大限制。256 个规格并不保证还有容量保存 256 个额外商品参数：若新增信息触及现有上限则按原规则拒绝，不截断冒称齐全。

调用路径保持 contract → `captureDocument` → 原 `validateCapture`/canonical hash → 页内按钮、快捷键和弹窗共享交接 → 原 receiver → SRC/Catalog/Collection 同一 Product UoW → 原属性展示及原键/原标签恢复。商品/规格 attributes 原本即为来源事实；不新增 wire/schema/parserVersion/数据库或第二事实 owner，不改变来源报价、价格、SKU、权限/租户、幂等/UNKNOWN/retry、事务、浏览器存储或恢复协议。尤其规格可订数量只作带来源标签的参数，不是自有可用库存，不做库存写入/同步。

Must：四项有证据且未触及原预算则读取并绑定正确层级；可订零值及大整数字面量不丢失；不可用可选值不影响已知资料；冲突不覆盖；原报价/规格图片/保存恢复保持；币种、真实商家 SKU、重量单位继续未知即缺失。Out of Scope：销量/评价/视频、供应商管理、物流承诺、其它平台改造、旧版本回填、实例或现有插件更新、真实重采/保存、部署及服务端采集。当前来源值为不可信页面资料，沿原纯文本展示和保存校验；不扩大现有 Threat Model 或另建治理/验收工具。

验证先 TDD RED：按已证实字段重建有明确标记的 fixture（不是完整原页 DOM），逐规格不同数量、显式零/大整数、字段缺失/空/非法类型/整数格式、其它块/其它商品隔离、同名冲突及既有预算；再最小实现。复用 Vitest/JSDOM、原严格 v2/hash 与 Go 纯映射/ToSnapshot 测试，证明四项来源属性保留且 `Stock` 不变；原 Amazon/1688 sibling、typecheck/lint/fixture-only build 按风险检查。无新依赖、runner、真实浏览器/数据库/容器测试；最终稳定候选一次独立完整交付复核，准确 HEAD CI 和证据留主要 PR。

Legacy decision: EXTRACT。Reusable behavior: 上表四个已存在的明确提取路径。Current owner: 当前插件纯解析及既有 Product/SRC/Catalog/Collection 保存 owner。Cutover/deletion condition: 纯解析和原映射验证后使用新提取；不导入旧 Go crawler/DTO/Service，不修改另一退休会话的代码。真实运行和用户验收 NOT_RUN，现有数据、配置、加载目录和插件身份保留；ProductAgent、Chat 和服务端采集不启动。
