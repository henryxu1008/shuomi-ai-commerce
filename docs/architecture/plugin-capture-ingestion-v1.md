# 插件采集增量架构：规格图片与 Amazon 接收保存 V1

> #989 当前恢复增量：[持久草稿与可靠交接](plugin-capture-durable-drafts-v1.md) 已达到 IMPLEMENTATION_READY 并获准离线实现。本文中“内存是未保存资料唯一 owner”、旧交接协议及 UNKNOWN 不可重试的限制由该增量替代；原后端身份授权、固定 key/body、事务保存和原键核实继续复用。不迁移旧内存现场，不改变已有保存版本，也不授权更新实例或插件。

> 2026-10-23 命名更新：当前源码/操作入口为 `extensions/product-capture`，私有 npm 包为 `@shuomi/product-capture`。下文原设计中的旧目录名保留为冻结依据的历史路径；本次不改变平台路由、合同或运行实例。

Refs [#902](https://github.com/qq550723504/shuomi-ai-commerce/issues/902)，主要文档 PR #903。字段依据：[插件上架字段设计](../product/plugin-capture-listing-fields-v1.md)。

Status: **IMPLEMENTATION_READY（本增量架构设计层） / DEVELOPMENT_NOT_AUTHORIZED**。
Design Basis: **Independent Architecture**（新增来源接收合同、producer准入、原键/保存恢复与现有规格图片映射）。用户2026-10-23批准继续设计及独立评审，**暂不开发**。评审通过也不自动产生开发、合并、部署、插件安装或真实采集授权。

## 1. 产品边界、范围与现状

用户结果：Amazon / 1688插件采到的明确规格图片，保存后仍归属正确来源规格；Amazon插件证据经当前身份/企业授权后保存到原Product/Catalog/我的数据，可按原键核实结果。保存不调用来源站、不扣DATA_ROW、不创建收费Amazon job、不自动适配/上架。

本增量只设计上述两条路径及其最小接收/读取接线；不开发DOM覆盖、批量/按页/关键词、富文本、评分/库存数量规则、自动转入、后台执行、下载图片或新Asset存储。原字段表其余缺口继续保留，不宣称这两项设计使资料自动齐全。服务端采集新增研发继续暂缓；身份、接收、保存服务保留，暂停采集不等于停用整个后端。

当前事实：
- `SourceEnvelope.ProductCandidate.Variants[].Images`、`AssetCandidate`、`ToSnapshot/snapshotVariants`和Catalog规格图片已存在，并有`variant_images_test.go`；**不新增规格图片owner、Catalog列或图片映射表**。
- 1688 `browserVariantWire`、`AcquisitionVariant`和`MapAcquisitionEvidence`未承载该关联；既有v1严格解析/字节规范化已绑定历史operation，不能原地宽松接受新字段。
- Amazon原生服务端入口依赖job/item、资金/扣量和执行证明，不能假装插件也执行了这些步骤。
- 原SRC publication与Catalog同事务保存；Collection有`AppendDataPublication`可参与原Product事务。旧`AppendPublished`批次名/类型固定1688，不用于Amazon。
- `capturedResultReader.Verify`是收费job结果验证，固定producer/checksum；它不作为插件结果读入口，也不为插件放宽验证。

明确Must：准确规格关联、插件执行/回执的原身份/企业/成员隔离、同键异载荷冲突、Product三方原子保存、UNKNOWN原键核实、插件免费且无服务端采集。已保存Collection/Catalog资产继续当前org+actor owner与有效访问权限，不新增“原member永久独占资产”的产品规则。Should：原接收页/我的数据UI复用，缺失与读取状态清楚。Out of Scope如上。没有新增Accepted Risk或放宽原保护规则。

## 2. 唯一owner与调用路径

| 责任 | 现有owner / 拟增量 |
| --- | --- |
| 浏览器商品资料、规格图片原始关联 | extensions/1688-capture的有界解析/交接；扩展明确Amazon来源分支，不建通用插件平台 |
| 1688证据/原键生命周期 | internal/product/sourcing + app/productsourcing既有BrowserCaptureService/AcquisitionOperationStore；显式v2wire/map准入 |
| Amazon插件请求校验、映射和原键用例 | app/productsourcing中一个有界AmazonBrowserCaptureService；领域校验归product/sourcing中的Amazon浏览器DTO/映射，站点/ASIN沿product/dataacquisition现有纯校验 |
| 来源证据与权威回执 | SRC现有PublicationStore/InternalProducer；新增准确descriptor `amazon_browser_capture/v1`，不是收费`amazon_data/v1` |
| 产品及规格图片 | Catalog已有不可变版本、Variant.Images与Asset候选；图片批准仍归Asset |
| 我的数据引用/批次 | Collection已有scope/PublicationBatch/AppendDataPublication，source kind复用`amazon_data`（平台数据分类）；producer metadata区分插件，分类不作为收费证明 |
| 身份与访问 | 当前ZITADEL/Workbench/native authorizer，现有product_sourcing.write和Collection manage/read，无新角色/IAM |
| 装配 | 当前application显式options、受限Product pool、descriptor registrar、现有server-auth BFF；不启动provider/Temporal/Resource charge |
| UI消费者 | 固定接收页、原我的数据详情/下载、后续待适配只读引用；不修改候选/平台/批准资产owner |

1688：插件v2 → 原receiver/BFF显式v2请求 → v2严校验/Map → 原StartPrepared/Claim/resolve → 原source/Catalog/Collection事务 → 原operation回执。
Amazon：插件v2 → 固定Amazon receiver/BFF → AmazonBrowserCaptureService → 单Product UoW内SRC publication + Catalog + Collection → 原SRC/Collection回执。**没有新的Amazon插件任务表/队列/状态机/资金proof。**

公共保存port优先复用`sourcingpersistence.NewTransactionWriter(tx, existingCatalogBridge)`和`collectionpersistence.AppendDataPublication`。两个app层已有Catalog bridge重复接线可做最小EXTRACT到现有Product composition owner，不能为此建通用框架，也不能让新代码依赖收费DataService应用模块。

## 3. 规格图片：显式v2wire与中立映射

### 3.1 冻结v1，新增v2

新插件只发`captureVersion=2`，v2的1688和Amazon为两个严格可辨分支，不能把Amazon ASIN塞进offerID。v1字节顺序、parserVersion、hash及已持久命令校验不改；无“v2失败转v1”或弱校验fallback。

v2的每个规格直接内嵌`images: [{url, role}]`，与`sourceID / sku / attributes / price`属于**同一个规格对象**，不使用数组下标外键、名称匹配、另一张关联表或客户端Target SKU。规格图片结构只接受当前允许的图像URL/角色字段，客户端不能设置Asset批准、对象存储路径或内部imageID。

- 有来源规格ID则要求同一capture内唯一；没有ID可保留明确且无歧义的属性组合，当前缺失事实机制保留缺sourceID。重复属性组合或无法区分的多规格不猜测。
- Amazon父ASIN/子ASIN只按来源明确关系记录；同一capture的商品ID为用户选中商品。声明为已选子商品的资料不冒充完整变体族；没有显式父子关系不合并多个ASIN。
- 一张图可由来源明确关联到多个规格；只在各自规格内去重，不能全局去重后丢掉关联。没有关联证据的图片只进商品级候选，不随机分到SKU。
- 空图片数组与未观察到/截断分开，用现有missingFacts/warnings记录；不从相册第一张补满所有规格，不以缺图伪造单规格。
- 保留来源顺序和可证明的role；不改图、不下载、不认定图片永久可用或已批准。

### 3.2 传输→domain→Catalog→consumer

1688 v2解析后构造带`Images`的AcquisitionVariant增量，并显式调用v2映射；图片URL验证/嵌套条数先于分配与Normalize。映射到已有`ProductVariantCandidate.Images []AssetCandidate`；SourceEnvelope预检与Catalog `snapshotVariants`继续拥有最终约束。不得把图片塞入任意attributes字符串。

1688新命令固定descriptor `browser_acquisition/v2`，metadata包含v2 contract/parser/capture digest；`StartPrepared`、repository.validCommand、`browserCaptureOperation`必须按确切descriptor/version验证原冻结命令。新版本fingerprint包含v2contract及服务端canonical payload digest；v1原算法/历史命令不重算。发布guard仍走原browser acquisition免费渠道证明，不通过换producer绕开原operation guard。

1688版本判别固定如下，不能试算多个hash猜版本：v1沿原`captureVersion=1 / schemaVersion=1 / browser_acquisition/v1 / src2b-acquisition-v1 / 1688-browser-dom/v2`；v2固定`captureVersion=2 / schemaVersion=2 / browser_acquisition/v2 / src2b-browser-capture-v2 / 1688-browser-dom/v3`。v2 fingerprint为对`["src2b-browser-capture-v2", "browser_capture", canonicalSourceURL, serverPayloadSHA256]`作既有JSON数组规范编码后SHA-256，原operation org+actor+key命名不变，因此同键跨版本仍冲突。

v2只能经`StartPrepared`把原命令与operation原子保存，不创建v2空command的acquiring阶段。持久读取先有界解析原command，再由严格producer版本、contract/parser/channel一致组合选择**唯一**验证器，最后检查该版本fingerprint、command hash及原Source身份；Claim/Verify/恢复同样如此。browser capture记录无command、未知组合或不一致一律STATE_INVALID，不降级。既有非capture acquiring路径及旧版本校验保持原合同；不得为新v2复用空命令路径。新增中立字段必须保证冻结v1命令的编码不变（例如独立v2 DTO映射及可省略空字段），以原命令字节/哈希回归证明，而非重算历史记录。

Amazon v2映射同一已有Variant.Images，不采用服务端Amazon Evidence的float64/无变体结构承载完整插件输入。共用中立属性/十进制/图片纯校验及SourceEnvelope，不共用两平台来源ID。来源金额保留精确十进制和币种，沿现有1688“可精确投影才给Catalog数值，否则保留源事实与precision warning”的行为EXTRACT；来源库存缺失不转成可售库存。

Source/Catalog精确版本读取、`CollectionSourceDetail`投影/前端strict contract须显式带规格images；源码中已有的Catalog规范化与只读内容消费者优先复用。图片被源→快照→详情/JSON下载读回必须仍绑定相同规格。接收/读取层不能`strip unknown`后报告成功。Asset批准与Target/SKU显式配对路径保持原规则。

## 4. Amazon固定接收合同与权限

拟定端点（均为设计，未注册）：
- POST `/api/v1/workbench/sourcing/amazon/browser-captures`：请求有Idempotency-Key UUID及严格v2payload，唯一可写入口。
- POST同base的`/verify`：原payload的只读核实，不保存/不重新采集。
- GET同base的`/by-key/:key`：无payload恢复读取。
- GET同base的`/:capture_id`与`/:capture_id/published`：准确回执及固定版本详情。ID均为下节定义的UUID，不能任意指定ProductKey。
- 同源BFF采用既有serverAuth、可信后端base、当前expected-user/org校验与Origin/CSRF保护；浏览器不接触服务端凭据，不接受客户端URL覆盖。

保存需要当前登录身份、有效企业/精确成员及`product_sourcing.write`和`workbench.collection.manage`；插件Verify/ByKey/Read/Published需要原scope/member及现有source/collection.read能力，复用原CachedRead/LiveWrite策略及其失效语义，**不是用写权限代替所有读取权限**。POST在准入及事务发布前用原LiveWrite精确核验，成员撤销/重建不复活旧marker。跨企业/他人/不同原member不能继续原capture执行或读取其插件回执。

保存完成后的Collection列表、详情、JSON下载及Catalog资产，继续现有org+actor owner与当前有效成员/权限规则。合法重新加入同企业的同actor能否读取已有资产由当前规则决定，不以旧capture的member metadata增加全局资产过滤。插件回执的执行上下文绑定与资产所有权是两种既有职责，不互相替代。

新receiver固定为同一获配HTTPS安装origin下`/capture/amazon`；原1688固定`/capture/1688`。插件按显式source分支选择编译固定path，不接收页面消息中的destination。站点初期沿当前Figma明确的us/ca/uk/de/jp五站、准确amazon域名，1688沿原detail.1688.com；其它现有服务端站点保持原合同，不随本设计开放插件。

v2 evidence仅接受来源、site、sourceID、canonicalURL、标题/描述/属性、规格/图片/报价、观察范围与现有缺失/解析元数据。source platform与route必须一致。producer、actor/org/member、ProductKey、费用、资金来源、操作状态、Target、授权、资产批准由接收方生成，payload不得携带。

插件仍复用当前文档ID/页地址双检、原tab/frame/extension消息与固定origin绑定。Amazon识别沿当前ResolveSite/NormalizeASIN行为EXTRACT；不放开任意来源/任意URL/重定向抓取，不处理验证码、不导出Cookie/Token/账户状态，不在接收端访问Amazon。新站点权限、安装包和实际页面交互均待后续获准实施。

边界：浏览器证据是不可信的客户端商品声明。tab/origin检查和服务端摘要绑定本次内容，**不证明Amazon官方认证、全站完整或资产许可**。只有内部固定producer可调用发布port；客户端不能借此获得任意内部producer或已有付费执行授权。

## 5. Amazon原键、唯一事实与事务

### 5.1 无新操作表的身份绑定

复用既有SRC publication作为已保存事实，不增加插件operation表或把浏览器状态当第二事实源：
- key为客户端在用户确认本次冻结payload前生成并保留的UUID。
- `captureID/publicationID = Collection.StableID(org, actor, "amazon-browser-capture:v1", key)`。只由服务端求值；同scope/key不同payload得到**同ID冲突**，不能把payload hash放进ID规避冲突。
- ProductKey复用Amazon现有actor-private `"amazon-"+StableID(org, actor, "amazon-product", site, asin)`。不同capture可产生该商品新不可变版本，不修改已保存旧版本。
- 原member ID、key、v2contract、完整identity、payload canonical digest和费用来源`plugin_free`写入**服务端构造的**SourceEnvelope.RawReference.Metadata；source trace固定`amazon-browser-capture:<captureID>`。客户端hash仅声明，不作为权威。
- canonical digest覆盖所有接受的payload字段（含有序图片、规格关系、观察范围、原时间/缺失），重复字段/大小写别名/未知字段拒绝；服务端versioned typed canonicalization不由客户端决定。冻结后重试不重新取DOM、更新时间或变更内容。
- Src原始envelope/receipt是来源事实；Catalog为唯一产品版本；Collection只有原引用。metadata不是费用账本，也不是新auth owner。

已有SRC receipt包含org/actor/producer/InputHash/expectedBase/CatalogVersion。新metadata绑定额外原member与请求digest；现有字段不能“推断为新请求”。现有SourceEnvelope/receipt JSON范围已可容纳，不计划新增schema列、迁移或覆盖。

### 5.2 单个Product事务

Amazon service通过显式Product UoW，使用同一个受限Product pool：
1. 严格有界parse/规范化，当前LiveWrite授权得到org/actor/member；不得使用客户端scope。
2. 开启READ COMMITTED事务，先取得按**同SRC现有org+publicationID advisory-lock命名/算法**的锁，随后检查原SRC证据/回执；复用窄持锁能力EXTRACT，不能另造锁空间。锁等待受deadline限制。
3. 若已有完整回执：检查原actor/member、producer、site/ASIN、key和请求digest全部一致；读取固定SRC/Catalog/Collection引用，返回原结果。异载荷409；scope不匹配拒绝。该分支不读最新Catalog生成新ExpectedBase，不补写/反归档Collection。
4. 若不存在：只有在同锁、当前主库新语句确认**SRC与Catalog/Collection三方均不存在本capture记录**后可首次保存。部分存在为STATE_INVALID/待核实，不修补、不重发。按原Catalog slot锁/ExpectedBase机制准备当前版本。不能用先前GET404当首次写许可。
5. 再用原LiveWrite authorizer核对scope/member；创建固定`amazon_browser_capture/v1` InternalProducer，调用NewTransactionWriter(tx, CatalogBridge)，完成SRC→Catalog。
6. 使用`Collection.NewPublicationBatch`和`AppendDataPublication(tx,...)`，batchKind/sourceKind均`amazon_data`，批次ID基于captureID，名称明确“Amazon <站点> · 插件采集”，Source.OperationID=captureID。不能调用固定1688命名的AppendPublished。Collection内部现有batch/item锁与scope/member核验保留。
7. 同一COMMIT成功后才返回200 saved以及captureID/publicationID/catalogVersion/batchID/itemID/payload digest/原键；不能在COMMIT前让handoff标published。只凭SRC回执而没有同事务Collection引用不称完整成功。

事务锁顺序：SRC capture advisory lock → Catalog product/publication slot与head → Collection batch → item，沿现有owner算法。同一ProductKey不同key并发由Catalog版本锁/ExpectedBase冲突收敛，不重排已有表。无外部provider或Resource调用。事务/请求最长沿现有Source publication/HTTP有界预算（最大20秒，事务执行不超过10秒），外部IAM沿现有5秒上限，剩余deadline不足即拒绝继续。

**实现注意**：现有NewTransactionWriter复用调用方tx而不会代COMMIT；外层必须只在所有owner成功后提交。`PublicationReceipt.InputHash`含ExpectedBase；重放必须先用固定存储内容，不按最新head重建command再Publish。原Collection数据若后续已移动/归档，重放仍返回原已保存receipt并如实表示当前可见状态，不创建新批次/引用。

三方检查是各当前owner的窄内部只读port，不是新修复服务：SRC用`NewTransactionReader`验证原证据+回执，Catalog按org/publicationID确认存在性并按原ProductKey/Version校验，Collection按org+actor+确定itemID读取**含归档行**及当前batch/归档状态，核对publicationID/version/kind/operationID与原记录。不能使用会隐藏归档行的普通ReadItem/ReadBatch的404推断未保存；首次三方不存在检查与重放检查都在同一捕获锁及调用方事务内。Collection现有member_id可核对原执行上下文，但不改变普通资产读取策略；移动后不要求当前batchID等于初始batchID，也不把移动/归档当损坏。内部port不向客户端提供任意scope、ProductKey或包含归档数据的通用查询。

## 6. UNKNOWN、重试、浏览器丢失与并发

| 触发 | 允许行为 | 禁止行为 |
| --- | --- | --- |
| 解析/授权/大小校验在事务前明确失败 | 返回对应失败；用户修正后是新明确操作 | 假报已保存/已扣量 |
| 事务明确回滚、同锁读确认无三方事实 | 用户可原键原payload显式再提交 | 生成新key自动重试、隐式重采 |
| COMMIT响应丢失/超时/断线 | UNKNOWN；ByKey/Verify沿相同锁在主库读取原事实，受deadline限制 | 超时当失败、查不到一次即重发/重复写 |
| 同键旧事务仍持锁 | 等待既有有界锁；超时返回待核实 | 返回权威NOT_FOUND或另起publication |
| 同键并发同payload | 一次提交，其余只返回同版本回执 | 多份Collection item/新Catalog版本 |
| 同键异payload、site/ASIN/关系改变 | 409；保留原操作与payload | 再绑定、取新base覆盖原command |
| 同商品不同key | 明确新capture，可追加新版本；CAS冲突读取后由用户确认新操作 | 把同key当新版本或静默合并规格图 |
| receiver刷新/插件worker终止 | 从本身份+org+member+source隔离marker读取原key并查回执；GET不发起保存 | 页面reload自动抓取或提交 |
| payload丢失但key保留 | 先ByKey读取；已存在则恢复回执；未存在且无完整原payload时暂停该保存 | 用新DOM内容套旧key或把缺失payload判保存成功 |
| 退出/切企业/成员撤销 | 取消新提交，保留原上下文marker；回原身份且当前授权有效时只核实原键 | 搬运payload到新企业、清浏览器存储解决未知 |
| 已保存后Collection归档/移动 | 精确原publication仍是历史事实；读回当前归档/移动状态 | backfill/重建批次、撤销原归档 |
| 读取损坏/证据或回执只存在一半 | STATE_INVALID/UNKNOWN，保留现场交原owner | repair写、最新版本fallback、复制另一个回执 |

GET ByKey/Read/Published与POST Verify均**不写业务事实**。ByKey必须在复用同advisory锁后以READ COMMITTED的新语句读取原主库；不得从replica/cache的404宣告不存在。锁无法在预算内取得时保持UNKNOWN。GET没有payload不判断新payload一致性；Verify/重提POST才检查原digest。ByKey回执本身必须核对当前准确scope/member与固定内部producer；不直接信任由客户端算出的ID。

保存前沿原handoff保留有界冻结payload和隔离恢复marker；不把长期明文浏览器缓存作为新默认要求。marker至少原key/source/site/sourceID与digest，保存失败/未知不自动清除；存储失败未发送则拒绝发出保存。若原完整payload仅内存存在而丢失，则按表中无payload规则处理，无需建设新持久恢复平台。

## 7. 读链路、免费渠道与装配闭环

Amazon新service的原键/Published reader使用现有`NewTransactionReader`核验准确SRC receipt/envelope，再核固定Catalog/Collection引用及原member，不借用收费job的`capturedResultReader.Verify`。当前Collection `Service.readItem`已通过`WithSnapshots(NewBoundedSnapshotReader)`按ProductKey+Version读取并核对PublicationID，不依赖Amazon job或producer分支；该详情读取直接复用，不新增source resolver。`web/listingkit-ui/src/lib/contracts/product-collection.ts`的`collectionDetailSchema`也已有`variants[].images`，保留这个字段即可，不重复设计中立图片schema。原键恢复新增的SRC核实不替代Collection既有权限校验。

Source detail/JSON下载/BFF strict schema必须包括商品与规格图片、来源角色/观察范围及缺失事实；不从当前Catalog latest读取原capture详情。GET只读：403/404/503不伪装无缺失或回退另一版本。读取权限撤销仍fail closed；已保存事实不删除。

当前`collection-page.tsx`的原始资料弹窗只直接展示商品图；`variants[].images`虽已随详情进入JSON下载，却没有按规格展示。后续只在原弹窗增加按来源规格ID/明确属性分组的图片查看，复用原Image proxy和有界展开，不新建页面或Asset owner。父/子商品、部分观察与缺失提示复用原来源/Warnings区；原missingFacts需经已有SourceWarning投影保留可读原因，不能由空图自动推导完整。JSON下载沿原`downloadJSON(detail)`，保留已有嵌套关系。本段为原资料详情的局部消费设计，未修改Figma或UI。

创建者私有scope保持与现有Amazon数据一致。手动保存同一源商品在不同actor得到各自ProductKey与Collection，不把他人已有商品当自己的回执。插件receiver没有DataKey认证、service身份或platform-admin旁路。

新模块只有receiver、映射、source transaction ports和只读reader，显式由current-application注入；构造缺Product pool、Collection port、Live auth、Catalog bridge任一项拒绝开放。UI可用性来自实际注入状态，不靠工具启用事实或Figma“可用”。不依赖Data Services providerReady/Temporal/资金余额，不访问/重试来源页。

插件free来自**内部固定路由→固定producer→无charge ports的用例装配**，不是请求字段传`free=true`。原收费Amazon DataQuery、worker、DATA_ROW ConsumerChargeOwner/RecoverDue完全不改；不会因插件保存释放/消费任何旧reservation。1688依旧走原browser_capture免费合同。对于SRC已有guard按producer名字分类的现状，不改收费guard默认规则，只显式登记新插件descriptor及其当前native authorizer，不提供客户端选择descriptor能力。

## 8. 大小/安全/资源边界

- 复用BrowserCapture及SourceEnvelope：请求/编码envelope/snapshot各不超过2MiB；单字符串8KiB；每个collection不超过256项；**所有嵌套**属性/规格/商品图/规格图/报价/warnings/missing合计不超过1024，进入Normalize/JSON大对象前检查。一个URL关联多个规格每次引用均计数，不能以去重绕过预算。
- 本期一个保存请求对应一个商品capture；不接列表发现/分页/递归URL、二进制图片、跨域frame、完整DOM、cookie/profile或任意脚本。来源描述仅既有纯文本边界，富文本另设计。
- 角色/路径/图片URL复用当前public-image准入，不放宽到内网、data/blob/file或含凭据URL。接收不下载；后续Asset仍独立做原网络/字节/批准检查。
- 使用当前HTTP body-read deadline与明确request deadline；取消/超时取消事务，结果不确定仍原键核实。连接并发沿当前有界Product pool（原上限不抬高）；不新增无界goroutine、队列或后台补写。
- 新Amazon保存无provider额度；持久化增长按原Product资源边界承接，本次不预建通用配额/清理平台，不自动GC任何历史回执。若实测达到当前资源瓶颈，单独记录问题，不删除业务事实解限。
- 日志/错误只返回关闭词汇错误码及安全ID；不回显请求体/源脚本、凭据、完整未验证URL。Source payload在授权边界内，不自动复制到公共Issue/PR。

## 9. 切换与已有数据保护

本次没有schema/代码/插件/运行变更。未来实施的v2入口和公开读取接线可以在隔离新测试库验证，不初始化、迁移、清理现有保留实例。

1688新写仅获准的v2插件/receiver调用；既有v1是**当前已批准合同及历史事实**，并非登记RETIRE的legacy owner。历史v1 operation/command保持原校验、原键、原owner的既有读取/恢复能力；不转换payload、不双写、不为新capture回退v1。切换前若存在未完成handoff/UNKNOWN，保留原版本路径处理原命令，不能把它重新标为v2；若不能在现有合同内完成，停下报告用户，不设计临时兼容适配器。新安装是否保留v1写入口由后续切换配置明确，本设计不打开新旧双通路。

Amazon是新增显式插件来源；原服务端source/job/charged receipts及worker历史保持，不迁移为插件记录。新kind不进DataKey或收费job协议，没有旧ID映射、第二事实源或Legacy包装。

Legacy decision: **EXTRACT**（只抽取仍正确的当前纯解析/桥接行为）；无RETIRE代码改动。Reusable behavior：当前images mapping、站点/ASIN、十进制和source/catalog/collection事务。Current owner如§2；Cutover：新路径实际准入后只对新用户操作使用新wire，历史命令不改写。不得引入已登记RETIRE的crawler/service/task抽象。

## 10. 实施验证计划与评审准入

本轮不执行下列代码/环境验证；它们为未来实施中与边界直接相关的必要用例，不新增runner或故障平台：
- 复用现有strict-wire/parser/variant_images用例：两规格不同图片、同图多规格、空图/未知/截断、重排、重复/未知字段、嵌套超限；源→Catalog→exact详情/下载图片归属不变。
- 1688v1原命令hash/Verify回归不变；v2确切descriptor/metadata/fingerprint/guard准入；跨version不同payload同key冲突，无fallback。
- Amazon真实Postgres同UoW：任一步失败无SRC/Catalog/Collection部分提交；同键同payload顺序/并发同回执；异payload/跨scope/member拒绝；同商品不同key的版本冲突不覆盖。
- 复用现有transaction故障注入点做响应丢失/COMMIT未知、ByKey锁超时/新语句不存在、旧tx提交后读取、归档后重放不复活、固定版本不读latest。
- 必要HTTP/BFF和UI原键恢复：权限/组织切换、未登录拒绝、原payload丢失、不产生收费job/reservation/provider调用；接收页只确认保存，不自动转入或发布。
- 仅在用户另行授权后，真实浏览器采集、保存读回、安装、试用和实际运行接线；否则全部NOT_RUN。设计评审/CI不等于用户验收。

Independent Architecture Review轮次：**R1与R2增量复核已完成，架构设计冻结为IMPLEMENTATION_READY**。R1按AGENTS分类后处理：

| Finding / 当前Must | 分类与阻塞层级 | 文档处置 / 实施验证 |
| --- | --- | --- |
| 原member绑定与Collection资产org+actor读取承诺冲突 | BLOCKER，实施准入；按原稿会违反其宣称的访问控制 | §1/4明确插件原执行与回执绑定；资产读取沿既有规则，无新增全局member过滤 |
| 1688 v2持久版本判别未明确 | IMPLEMENTATION_TEST；未满足则阻止未来实现合并 | §3明确版本组合、StartPrepared、解析/校验顺序和单一fingerprint；未来覆盖Read/Claim/Verify/恢复与跨版本冲突 |
| 无新操作表依赖外层事务及含归档重放检查 | IMPLEMENTATION_TEST；未满足则阻止未来实现合并 | §5明确同SRC锁/READ COMMITTED及三方窄port；未来验证部分状态拒绝、归档/移动不回填及COMMIT未知恢复 |

R1确认无需新增规格图片表、Amazon插件操作表或公共解析平台，现有固定版本详情/JSON下载不依赖收费job。R2复核上述修订及实际受影响边界，确认原BLOCKER已解决，无新增BLOCKER，结论IMPLEMENTATION_READY。两项IMPLEMENTATION_TEST保留，未来实现合并前须满足本轮Must，不等于已测试或用户验收。本次用户“暂不开发”仍有效。准确候选HEAD、状态收口提交与独立review证据在PR维护，不把旧SHA评审当新HEAD认可。
