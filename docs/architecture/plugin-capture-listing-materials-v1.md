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

沿用最终载荷 2 MiB、集合最多 256、总项 1024、描述 8192 UTF-8 字节、URL 2048 字节、8 秒等原限制。新增 DOM/AST 遍历使用现有级别的总节点预算 65536、深度 32；最多 256 个通过现有子树排除且包含 colorImages 的相册候选内联 script，候选文本累计最多 2 MiB，先限输入再解析。无关脚本仍计入原遍历节点与时间预算，不占相册候选名额；语法不支持的候选仍计数。此口径由 #916 的 2026-10-24 用户修复决定明确，纠正 #933 将整页内联脚本数量误作相册候选数量的阻断。参数/段落/图片均在累积前计数、限字节。超限按原 CAPTURE_TOO_LARGE/TIMEOUT 停止，不截断冒称完整。

新图片元数据语法不支持可以明确保留原有相册；资源超限、来源变化和严格 payload 校验失败仍拒绝本次捕获。正文参数不执行页面脚本，不读账号/cookie/storage，也不调用来源网络。所有入口沿同一平台提取器，无额外按钮、持久缓存、重放或副作用 owner。

## Legacy decision

Legacy decision: EXTRACT。
Reusable behavior: 旧 Amazon ImageExtractor 的显式 hiRes/large 及 Description/ProductDetails 选择器，旧1688 PackInfo/Shipping 的明确字段路径。
Current owner: extensions/product-capture 平台解析；保存事实继续由现有 Product/SRC/Catalog/Collection owner 持有。
Cutover/deletion condition: 插件纯解析回归及原 v2 映射验证后使用新提取逻辑；无旧 Go 包/DTO/Service 依赖，不修改另一会话的退休代码。不复制旧全页正则、尺寸后缀猜测、默认币种/库存/包装或跨来源值兜底。

## 验证与停止点

TDD 先 RED：双平台原 v2 输出的高清映射、字段顺序变化和 JSON 字符串包装；当前商品绑定、冲突/视频/推荐排除、主图去重/规格归属；新增参数与 A+ 去重/安全正文；详情图与原始重量精度、单位未知、资源边界。再最小实现 GREEN。复用现有 Vitest/JSDOM、typecheck/lint、插件打包和原保存映射测试；不新建 runner/验收平台，不连接保留数据库/浏览器/实例或下载图片。

稳定候选同步一次真实 main，集中最终独立复核实际 diff 和有效证据，准确 HEAD 必需 CI 留在同一主要 PR。真实高清资源可用性/像素、两商品重采保存、运行更新与用户验收均 NOT_RUN。Issue 保持 OPEN，不合并、不部署。
