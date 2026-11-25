# Google Shopping 与 Meta 商品上传首版

执行依据为 [#1617](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1617)。产品范围已确认。Google 商品上传的独立架构复核已通过，按本设计为 **IMPLEMENTATION_READY**。Meta 目录写入、Facebook Shops 和 WhatsApp 的首版路径仍是候选，需取得平台准入及实证。分阶段执行状态以 Issue 当前正文为准。

## 用户结果和范围

企业运营人员在硕米工作台连接本企业的渠道，选择已有的自有商品和目标，核对实际提交内容，确认后上传并查看平台结果。先完成 Google Merchant Center 路径，再完成 Meta 商品目录路径和 Facebook Shops、WhatsApp Business 展示验证。一次提交一个商品，每个适用变体有可回查的外部身份。订单、收款、消息发送、广告、库存持续同步及迁移现有数据源留在本次范围之外。

用户先在「店铺中心」完成渠道授权，在现有商品准备路径选取目标并核对商品及各变体。确认前显示接收账号、站点或语言、商品链接、关键字段与具体缺项。返回商品准备页保留可修正输入，修改商品版本、目标或授权后要求重新核对。点击上传进入原有提交结果，显示「正在提交」「平台已收到」「商品已写入」「符合展示条件」「需要修正」或「正在核对提交结果」，按平台实际可观测证据提供相应状态。没有买家可见的证据时不显示「已展示」。

采用当前[产品业务流程依据](../../../product/final-ui-ia-authority.md)、[店铺中心合同](../../store-center-application.md)及 [Listing 领域导航](../listing/README.md)。Shopify 已管理的同一批商品继续使用原有同步渠道，硕米直连首版不接管已有记录。Legacy decision 为 N/A。

## 所有权和调用路径

| 责任 | 承接方式 |
| --- | --- |
| 客户企业、角色与操作权 | 现有身份、Effective Organization 和店铺／商品操作权限；UI 按已返回的操作权限提示，服务端独立判定 |
| 官方授权与目标归属 | Store Center 持有渠道连接、外部账户或目录身份、授权版本、可用范围和撤销状态；Google 数据源及 Meta 目录是被授权的外部目标 |
| 商品事实与确认资料 | Product Catalog 继续持有商品事实，Asset 持有可用图片；Listing Preparation／Record 持有已确认的商品版本、平台字段和外部目标 |
| 外部提交与恢复 | Listing Submission 持有提交身份、远端回执、各变体结果和 UNKNOWN；Google／Meta adapter 只负责官方协议、字段映射和读回证据 |

拟用入口为现有店铺中心和商品准备／结果页，经过当前 BFF、权限中间件及 Store Center 或 Listing 领域服务。应用装配通过已有模块注入平台客户端和现有持久化实现。店铺连接、点击确认、领取发送许可分别由所属事务处理。外部 HTTP 调用发生在提交事务之外。平台不作为本地商品资料的第二事实源。

Store Center 将 Google Merchant Center 账号或 Meta 商品目录各映射为一个规范店铺目标，并沿现有授权仓储以事务原子绑定企业、目标和新授权版本。对同一平台真实环境中的外部账号或目录，写入连接采用数据库唯一约束，禁止以另一应用版本、连接或企业重复绑定相同外部目标；同企业重连更新原授权并保留本地店铺身份。脱离商家或撤销时停止使用该绑定。本轮不转移远端目标所有权。接入需要增补 Store 模型和刊登目标的平台校验。复用连接服务与提交内核，并在相应 owner 明确接入新增渠道；禁止将 Merchant Center 或 Meta 目录伪装成已支持的店铺类型，禁止绕过平台分派走旧接口。代码依据见[领域边界](../../project-boundaries.md)、[提交内核](../../../../internal/listing/submission/README.md)。

## 授权和外部目标

- 连接新渠道时，先用 Store Center 建立企业内的未连接店铺记录。沿现有连接合同以 StoreID、预期版本、actor、企业、渠道、回跳和有效期绑定一次性 state。回调重新校验当前会话、有效企业权限、state 及原记录版本，领取一次兑换权。服务端把短期候选凭据按现有密钥保管方式加密，并从平台确认用户可操作的外部资源及实际权限，浏览器只接收可选目标。客户确认时再次核对同一 actor、企业、授权尝试、记录版本与候选有效期，再原子消费候选并绑定到这条记录。过期或已经使用的候选不能提交。如果目标已属于同企业的另一条连接，提示返回原连接重连；属于其他企业时拒绝绑定。未完成连接的记录不可作为提交目标。浏览器提交的外部 ID 只作为待核对的选择。各渠道凭据沿用现有凭据保管边界，连接关联授权版本，凭据不进入浏览器、提交载荷或日志。
- 用户确认和每次外部发送前，读取当前企业权限、连接归属、连接版本、目标及写入能力。解绑、企业切换、有效权限撤销或授权版本变化均使未发送的旧确认失效。已被平台接收的写入可能继续处理；解绑之后保留已有提交历史，新查询仍须通过当前授权校验。
- Google 用第三方 OAuth 和 Merchant API，应用登记、可用的 OAuth 范围、测试用户及公开 OAuth 验证分别确认。首版只选择客户明确指定、专供硕米写入的 API 主数据源。客户需确认原有 Shopify／第三方同步不使用硕米分配的 offer ID 前缀。无法确认专用来源及编号范围时不开放写入。上传动作不创建账号、接管数据源或开通广告。应用级准入和客户账户／数据源权限分别核对。发布目的地仅开放 Free Listings，并在商品载荷明确排除适用的付费目的地；通过处理后的目的地状态回查，意外进入付费目的地的商品不能作为首版验收通过。[Google 商品目的地字段](https://developers.google.com/merchant/api/reference/rest/products_v1/ProductAttributes)[Google 第三方 OAuth](https://developers.google.com/merchant/api/guides/authorization/access-client-accounts) · [API 数据源](https://developers.google.com/merchant/api/guides/data-sources/api-sources)
- Meta 用商业登录和经审核的目录访问。候选接入是 Meta Business Login 加客户已有目录的单品 Catalog API，需验证该组合实际允许客户授予单品写入权限。若实际获准的 Meta onboarding 合同强制首次全量同步，逐品写入路径在该接入方式下保持未开放，并将冲突交回产品决定。不能为了通过接入要求将首次上传扩成批量替换。WhatsApp 首版使用客户在 Meta 已绑定的目录，用获准的账号／目录访问权限调用读取接口核对绑定。该接口另有 WhatsApp Business 权限要求，Meta 目录可写不代表已有这些权限；缺权限或无法查询时显示待确认，不发送消息。[目录权限及审核](https://developers.facebook.com/docs/facebook-business-extension/fbe/faqs/) · [Meta 访问验证](https://developers.facebook.com/docs/development/release/tech-providers/) · [目录集成要求](https://developers.facebook.com/docs/facebook-business-extension/fbe/guides/catalog) · [WhatsApp 目录绑定](https://developers.facebook.com/docs/graph-api/reference/whats-app-business-account/product_catalogs/)

## 提交、保护现有商品和恢复

保存确认时绑定本企业、商品／变体版本、外部目标、授权版本和规范载荷摘要。同一确认和同一载荷复用已有提交结果；版本或载荷变化需要新的确认。稳定外部标识由本地商品和变体身份确定，可从平台回查。按远端可独立写入的商品项保存回执。确认时持久保存一份不可变商品提交计划，首个远端写入前确定所有子项和各自的稳定外部身份。一个用户可见的父结果从子项回执派生。重启根据同一计划回读现有子项，UNKNOWN 保留原提交记录和目标互斥。部分成功时保留每个已写入项，只能对可确定未写入且已改正的项重新确认。对 Google 和 Meta 分别在 Listing 定义已确认的提交意图和回执类型，平台 adapter 不复用 SHEIN 的载荷或回执。现有内核负责互斥和单次发送许可。提交服务在调用 adapter 前对同一确认的连接和当前权力再做实时检查，过期或失权只按现有证据规则处理。浏览器和工作流均不能直接调用平台 adapter。按当前运行合同定义期限和有界请求。复用适用的额度与权限边界。

| 目标 | 写入和回查规则 |
| --- | --- |
| Google | 外部商品身份由 Merchant Center 账号、语言、feed label 和稳定 offer ID 组成。API 主数据源是提交绑定字段，不能成为重复发送的区分条件。每个规范外部账号的连接使用持久化、唯一分配的编号前缀，加稳定的商品变体编码。重连时保留原店铺身份和编号前缀。Google 没有条件创建，预查只提供附加证据，不能消除并发覆盖。先读取外部身份和来源，确认命中其他来源或无法核对已存在商品归属时拒绝写入；只允许在已确认的专用来源及编号范围创建本次新商品。复查确认的目标、来源和商品版本后调用 `productInputs.insert`，保存返回的资源名，再由 `products.get` 读处理状态及 Free Listings 目的地资格。异步未读到商品只表示仍需回查。 |
| Meta | 目标身份为客户授权的目录及稳定 `retailer_id`。通过产品目录单品接口创建，首版显式关闭 `allow_upsert`，遇到已存在的同号商品先回读，不覆盖他方拥有的商品。写入完成证据使用远端回执或符合要求的商品回读。Facebook Shops 和 WhatsApp 的绑定、可展示状态单独读取或由真实店铺验收核实，不以目录记录存在替代。 |

外部请求超时、响应丢失、进程重启或发送之后撤权时，沿既有 UNKNOWN 规则保留未决结果。内核命令的 StoreID 使用 Store Center 的规范店铺 ID，SubjectID 使用平台外部商品身份导出的稳定标识；授权或商品版本变化不变更这两个值。回查须能把证据归于原企业及原外部目标和已确认商品项；仅凭编号相同、暂时查无结果或平台在售无法确认原请求。Google 处理后可能改写字段，无法从回读中排除另一写入方或确认原提交时继续保留 UNKNOWN，不能直接补发。支持人员从原提交记录定位用户确认、尝试和平台回执。采用[Google 提交和状态](https://developers.google.com/merchant/api/guides/products/add-manage)、[Google 写入覆盖规则](https://developers.google.com/merchant/api/reference/rest/products_v1/accounts.productInputs/insert)和 [Meta 商品接口](https://developers.facebook.com/docs/marketing-api/reference/product-item/)。

## 设计评审与验收

- 2026-11-24 独立设计复核确认 Google 防覆盖、规范连接、授权候选绑定、稳定提交身份和 UNKNOWN 边界。待实现验证的场景是不同企业争用同一目标、撤权和过期、变体部分成功、丢失响应，以及回查商品未进入付费展示目的地。Meta 真实试用阻塞在客户目录的单品授权与写入证据。
- 隔离检查使用官方形状的模拟供应商，覆盖有效确认上传、缺项、同意图重放、并发与丢响应、他企业目标、撤权后发送拦截，以及平台回执到用户状态的映射。
- 真实试用使用明确授权的试用账号和商品。Google 验收 Merchant Center 中可回查商品和处理状态。Meta 验收目录中可回查商品，以及具备资格的 Facebook Shop 和已绑定的 WhatsApp 买家端能看到同一份目录商品。平台应用获批、账号具备资格、真实写入、真实展示分别记录证据。
- 当前没有真实账号写入、应用审核结果或用户验收证据。应用注册／审核和真实试用按对应操作授权执行。


## Google 按需续期设计增量

**Independent Architecture · DESIGNING · 候选待独立复核。** 本节的准入独立于上文已批准的 Google 商品上传首版。依据 #1617 的真实试用前待办及 2026-11-25 派工，目标是已授权企业的运营人员在访问令牌到期后仍可确认原商品计划，并查询原提交。例行续期保留企业、规范店铺、授权代次、应用登记、Merchant 账号、专用数据源、语言、feed label 和独占编号前缀。Product Catalog、Listing 的计划、意图键、发送许可和结果分别由其现有 owner 保管。范围限 Google 已绑定连接的续期与消费，沿用现有人工重新授权入口。Meta、后台定时续期、应用切换、目标转移、平台真实操作和部署在本切片范围外。[设计派工](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1617#issuecomment-7152546099) · [首版验证 #1618](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1618)

### 入口、权限与原记录

- **普通读取和准备。** 店铺列表与详情、连接状态、商品选项、近期记录列表和商品预览均只读本地授权及业务资料，预览只保存 Listing 计划。把当前 `PublicationBinding` 的有效连接检查与访问令牌有效期判断分开。已绑定且可按需续期的店铺在短期访问令牌到期后仍可选择及预览，状态读取不发 OAuth 或 Merchant 请求。明确无法续期时提示联系有权限的成员在店铺中心重新授权。首版授权候选中的账号、数据源选择仍走原流程。[当前连接实现](../../../../internal/app/storecenter/google_connections.go) · [Google 上传页面](../../../../web/listingkit-ui/src/components/workbench/listing/google-upload-page.tsx)
- **确认上传。** Listing 校验当前有效企业、原商品及计划、刊登读取／管理／提交权限，对仍未存在尝试的商品项校验当前计划商品版本、已审核图片、StoreID、完整目标及授权代次。先查询原意图，已存在的同计划商品项只返回原尝试。对尚无尝试的项，在第一次 `PrepareGoogleExecution` 前让 Store Center 取得可用凭据。并发确认仍由内核原子检查意图和目标，只有新领取者进入预查和发送。临时续期失败或等待超时如发生在该项领取前，保留计划及待确认操作，返回可区分的失败信息，用户可稍后确认原计划。多项计划中已领取项继续显示原结果，只允许继续仍无尝试的项。现有 `Confirm` 先依赖检查有效期的 `PublicationBinding`，随后 `send` 在取得凭据前领取尝试；若只把刷新塞进 `Publish`，续期故障可留下终态 `not_sent`，同计划重放无法恢复。将续期前移是本次最小修正。[当前确认和发送顺序](../../../../internal/app/listing/google_publication.go#L264-L339) · [内核发送门](../../../../internal/listing/submission/tiktok_no_send.go)
- **发送前验证。** Listing 仅持有非秘密的目标、授权代次和 Store Center 内部签发的凭据版本引用。Store Center 在进入 Merchant 预查前重新读取并打开同一快照，适配器继续核验专用数据源和同号商品。随后核对本次实际将使用的凭据版本、有效期、应用、绑定、当前店铺及权限；Listing 重查商品、审核图片和提交权限后，走原有 `BeginGoogleSend`。这一阶段不续期、不替换预查所用凭据、不绕过既有单次发送许可。若快照在预查中失效，沿既有发送前停止证据收敛，已有尝试只回原结果。撤权提交与最后一次本地校验后的远端请求之间无法原子化，已经越过发送边界的请求继续按原回执或 UNKNOWN 处理。Merchant 拒绝凭据时不触发插入重试。[Merchant 预查和插入](../../../../internal/integration/googlemerchant/products.go#L89-L125)
- **原记录回查。** 进入原记录后，由用户触发「查询提交结果」；Listing 先按当前有效企业、商品及原计划范围读本地记录。只有已有成功或 UNKNOWN 的商品项需查远端，缺尝试或已知未发送的项仅返回本地结果。在 `Result → ReadProduct` 显式传入原记录的 StoreID、完整目标及 offer ID，确认当前授权仍可访问同一目标后才按需续期和调用 `products.get`。同目标的显式重授权生效后可按新授权查旧记录，写入仍须以新代次重新确认并受原品项互斥与防重发保护。保留原数据源及远端商品归属核验。当前 `ReadProduct` 只接收店铺和 offer ID，依赖当前授权的目标；本轮补上原记录目标比对。OAuth 无法续期、Merchant 查询失败或权限缺失时，保留有权查看的已保存结果并说明本次未能查到平台最新状态。来源、商品身份匹配的回读也只按既有证据规则处理，UNKNOWN 不自动改判或补发。[当前结果回查](../../../../internal/app/listing/google_publication.go#L374-L405) · [Merchant 回读](../../../../internal/integration/googlemerchant/products.go#L127-L156)
- **操作资格。** 上传与结果回查沿现有 BFF、已验证身份和 live Organization access 路由。两者均要求现有店铺读取权限、同一有效企业、可访问此店铺的当前成员及店铺授权、有效记录和服务期、未撤销的当前绑定及匹配的 OAuth 应用 ID/修订。查看远端结果使用现有刊登读取权限，上传额外要求原有管理与提交权限。Store Center 在这些路径内代企业更新凭据，使用窄仓储操作，允许符合原操作资格的非管理员触发；授权、绑定、重新授权和断开仍由原店铺管理员能力控制。调用链沿用官方应用注册注入 `GoogleConnections`、Store 成员作用域仓储、Google OAuth/Merchant adapter 和现有凭据保护，再接到 Listing；工作流及浏览器没有独立的续期入口，也不接触凭据。[Listing 路由](../../../../internal/app/listing/httpapi/google_publication.go) · [现有 Store 读取与管理员写入边界](../../../../internal/storecenter/google_repository.go)
- **界面恢复。** 当前页面在确认请求前就切到提交结果视图。实现时，明确返回「本次还未提交」且确认未领取发送尝试的续期故障须保留原确认入口；请求回执丢失时先引导查原记录。原计划有权继续确认且目标项仍无尝试时可回到同一已保存计划，不生成新 ID，也不自动提交。暂时故障提示「暂时无法连接 Google，请稍后再试」，明确需重新授权时提供店铺中心入口。原记录已有结果时同时保留已知结果与「暂时无法查询 Google 的最新状态」等提示。权限丧失维持原有访问限制。

### Store Center 持久化与跨实例领取

Store Center 是凭据及续期状态的唯一 owner。保留 `workbench_store_google_oauth_grants` 的加密凭据和有效期，增加同库的一行续期元数据，以 `(organization_id, store_id)` 为主键和现有 grant 外键。该行只记录 grant 代次、凭据修订、随机领取 ID、阶段和阶段截止时间，按需记录稳定的内部失败类别。阶段为「空闲／冷却」「已领取」「可能已发出」「需要重新授权」。领取 ID 仅在两个持有阶段存在，截止时间用于领取、可能已发出或冷却，按阶段约束字段组合及凭据修订正值。同代次没有元数据行时，在持锁发起首轮续期时从凭据修订 1 建立。例行续期成功在一个 Store 事务内更新当前 grant 的密文、密钥标识、访问有效期和更新时间，同时推进凭据修订并归还领取权。授权代次、店铺版本和连接引用保持原值。把加密套用原 grant 的组织、店铺、应用、grant ID 与授权代次绑定。通知、错误、日志和诊断只携带脱敏状态，凭据内容、密文和上游响应体均留在受保护边界内。[当前 grant 模型和绑定](../../../../internal/storecenter/google_connection.go) · [凭据保护](../../../../internal/integration/googlemerchant/credential.go)

新增元数据表可作为当前 Google Store schema 的显式增量安装，现行两张 Google 表无需 ALTER 或回填。实现时由现有 Store schema owner 入口添加安装和只读校验分支，空库安装和已正确安装 Google 首版的当前 Store schema 均可在明确执行该初始化步骤时安装。现有 Google installer 检出首版表后直接返回，需显式涵盖新增表的安装和完整性校验。启用本功能时 runtime 验证表、约束和角色，缺失即拒绝启用。使用现有 `store_center_runtime` 的窄权限，在新增表仅开放 SELECT、INSERT、UPDATE，DDL 由 owner 运行。Product 库、原 Listing 计划和尝试无需 schema 变化。实施验证使用隔离库。保留实例上的初始化仍需其操作授权，运行时不自行改表或重置数据。Legacy decision 为 N/A。[当前 schema 安装和验证](../../../../internal/storecenter/google_schema.go) · [Store 初始化入口](../../../../cmd/store-center-schema-init/main.go) · [Store 数据库合同](../../store-center-application.md)

1. **读取并领取。** 仅在续期元数据允许使用、且满足足够可用时间时，已保存凭据可用于本次操作。处于「可能已发出」时等待权威结果，期限已过的合格请求先将状态收敛为需重新授权。申请领取时，在短事务中按现有顺序锁店铺、适用的成员店铺授权行、grant、续期元数据，按 PostgreSQL 当前时间核对服务期和领取截止时间。快照包含企业、请求 actor/member、适用的成员授权版本、店铺记录版本及连接引用、grant ID、应用 ID/修订、代次、凭据修订、完整目标。仅空闲（且冷却已结束）或过期但仍处于「已领取」的行可由合格请求写入新的随机领取 ID。正式读取许可、领取、保存均校验有效企业及本次操作权限，复用装配层现有 Organization live resolver 重新判定，再跑 Listing 与 Store 的原权限检查，不能只沿用首次请求缓存的结论。外部身份解析在 Store 事务之外，店铺本地成员授权及其版本在同一 Store 事务内重验。外部组织权限变化以最后一次实时校验的可观察边界为准。[现有 live resolver 使用方式](../../../../internal/app/httpapi/product_review_application.go#L154)
2. **授权发出。** 解密和本地准备发生在事务外。在独立短事务中重新核对身份和快照、同一个领取 ID、阶段及未过期截止时间，原子变更为「可能已发出」。只有拿到该变更提交成功确认的原请求可进行一次 OAuth POST。等待者、重启进程或未确认提交的请求不能依据存储状态发送。确认后在事务外调用带取消且有硬截止时间的 OAuth adapter。发送期限从发出许可事务记录的截止时间倒推，给保存预留 5 秒，与本地请求剩余期限取更早者，迟获提交确认或进程暂停后不得重新起算发送窗口。仅使用当前已绑定应用。Google endpoint 的客户端认证方式固定为请求参数，`x/oauth2` 的认证方式探测会在未知方式下尝试第二次请求，本路径必须保持固定方式并禁止重定向或传输层自动重放。复用当前响应体大小限制。Google OAuth 客户端和 Merchant 客户端分别接收明确凭据，Merchant SDK 不隐式刷新。[Google 刷新协议](https://developers.google.com/identity/protocols/oauth2/web-server#offline) · [现有 OAuth adapter](../../../../internal/integration/googlemerchant/oauth.go) · [x/oauth2 v0.36.0 Google endpoint](https://github.com/golang/oauth2/blob/v0.36.0/google/google.go) · [x/oauth2 v0.36.0 兑换实现](https://github.com/golang/oauth2/blob/v0.36.0/internal/token.go#L215-L256)
3. **条件保存。** OAuth 返回后，用原请求身份在固定短预算内重验当前操作权限，即使客户端刚断开也只能为本次已发出的操作保存结果，不能继续 Merchant 调用。按领取时锁顺序重读授权条件和整个快照，要求同一领取 ID、阶段、未过期且当前凭据修订一致。店铺或成员授权版本变化、服务失效、Store 断开、开始重授权导致连接引用改变、成功重授权绑定新代次、应用或目标变化，以及已更新的凭据均使旧写回被拒。不得用迟到响应覆盖较新凭据，也不得将丢弃的响应再对 Google 发起撤销请求。重新授权在 `Begin` 改写连接引用时就阻断旧连接和领取保存，完成核对并 `Bind` 时才推进授权代次、重置续期元数据。断开沿用原事务清除密文并标记撤销，失效元数据不可被旧领取继续使用。未完成或超时的显式重授权继续按现有连接规则阻断旧引用。
4. **等待和期限。** 同一授权的其他合格请求只能在自身期限内短间隔重读，读到胜出的已提交结果后用自己的当前权限和预期目标核验。上传还要求预期代次相同。任何拒绝或超时均停止等待，当次不以过期快照调用 Merchant。候选预算为领取后 4 秒内确认发出，OAuth POST 最长 10 秒，结果落库及读回最多 4 秒，「可能已发出」的期限为发出许可事务后 15 秒，全部受实际剩余请求预算和服务期约束。使用已有 Merchant 10 秒预算并留 5 秒安全余量检查凭据可用性。续期及等待不计入已领取的 Listing 30 秒发送准备租期，单个请求对同一 grant 最多发出一次 OAuth 兑换。无可用时间、取消或权限不明即停止进入下一有副作用步骤。所有截止时间在数据库比较时使用实际当前时间，锁等待也受上述预算约束。[当前 Listing 路由预算](../../../../internal/app/listing/httpapi/google_publication.go#L19-L35) · [当前 Merchant 上限](../../../../internal/integration/googlemerchant/products.go#L21-L25)

### OAuth 结果收敛

| 观察结果 | Store Center 行为 |
| --- | --- |
| 合规的刷新成功 | 要求 Bearer、有效 token、足够的访问有效期及所需 `content` scope。返回 scope 时校验，省略时继承同一授权的原范围。刷新令牌省略时仅保留本次快照的旧值，返回有效不同值时原子替换整套凭据，禁止再用旧值。条件保存成功或权威读回确认后才提供给消费者。 |
| 授权明确失效 | 可信 OAuth 响应为 `invalid_grant`、明确 `access_denied`、明确缺所需 scope，或持有的刷新令牌确定缺失／无效，进入「需要重新授权」，此代次的旧凭据停止用于新的平台操作。沿现有管理员授权流程恢复。可信端点明确拒绝的 `invalid_client` 是应用配置侧临时不可用。按明确未兑换归还领取并冷却，界面提示暂不可用，由应用 owner 排查。 |
| 明确未执行的临时拒绝或可证明未发出 | 只接受可明确确认未兑换的有界失败，例如发送许可前的本地取消，或可信端点明确拒绝且可确认本次未兑换的限流响应。原持有人可在同一领取条件下归还领取，保留原凭据并设置短冷却，后续有效操作才可再领取。遵守有界的服务端重试时间提示，当前请求不连发。前轮没有失权或不明结果且旧访问令牌仍满足当前操作余量时，才可使用仍有效的已保存凭据。 |
| 可能发出后超时、丢响应或进程退出 | 通用 5xx、读体失败、无法验证的响应、发送后的取消、成功回执未能加密或保存都不足以证明未兑换。只有原持有人在期限内可条件保存；期限到后由下一个有资格的请求将此代次收敛为「需要重新授权」。发出许可事务成功后即使进程在真正发送前退出，也按可能已兑换保守处理。此代次不再兑换或使用原 token，避免 Google 已轮换但新值丢失时重兑旧值。仅「已领取」且从未进入「可能已发出」的过期领取可以替换。现有 `Refresh` 将多数失败归为 `ErrGrantUnavailable`，实现须补窄的脱敏结果分类。没有肯定证据的网络错误不归入可重试拒绝。 |
| PostgreSQL 提交回执不明 | 领取或发出许可提交未获确认时不发 OAuth，原活请求可有界重读并凭其确知未发出的事实条件归还。保存成功结果时发生提交回执不明，从 Store PostgreSQL 主库以新事务重读，并重验当前权限、连接快照及授权代次。找到同一授权下已推进的凭据修订时用安全胜出的已提交凭据。仍是原领取且未过期时，仅允许原请求用内存中同一已收到的 OAuth 结果重试条件保存，不再次兑换。看不到可用提交时返回暂不可用，保留领取直到按期限收敛。 |

Google 文档允许离线授权续期，也列出用户撤销、限期授权和应用处于 Testing 等失效因素。设计按 OAuth 允许轮换的合同保护原凭据。Google 文档未给出按刷新请求 ID 查询结果或安全重兑的合同。这里的「需要重新授权」只影响 Store 凭据可用性，Listing 的 Merchant UNKNOWN 仍绑原提交、外部身份和互斥。[Google 令牌有效期](https://developers.google.com/identity/protocols/oauth2#expiration) · [OAuth 2.0 刷新与轮换](https://www.rfc-editor.org/rfc/rfc6749#section-6)

### 取舍和最小验收

用现有 Store PostgreSQL 短事务及一行续期元数据协调多实例，避免在网络等待时持事务、锁或数据库连接。进程内锁只能协调单进程，单独 PostgreSQL advisory lock 在崩溃后也无法区分是否已兑换。内存 token cache、通用刷新框架、定时任务及跨库协议均不引入。成本是过期路径增加短事务和有界重读，少数无法判明的兑换需管理员重新授权；Store 版本变化时选择保守弃用旧快照。安全检查对准当前跨企业访问、错误授权、迟到覆盖及重复商品写入；外部平台在最后一次授权检查后已接收的请求和真实店铺展示继续按首版合同管理。

以下为后续实现的增量验收，当前均未运行。复用现有隔离 Store/Product PostgreSQL、OAuth/Merchant 模拟、Chromium 工作台路径和 #1618 已覆盖的未变代码证据。[现有隔离入口](../../../../internal/app/listing/google_publication_postgres_test.go)

| 路径 | 最小新增证据 |
| --- | --- |
| 已绑定连接到期，确认原计划，再查原记录 | 浏览器验证列表、预览与载入记录零远端调用。过期后仍可确认原计划，Store 续期后代次、目标、计划 ID、商品版本与证据绑定不变，只有一笔预期 Merchant 插入。原记录主动回查续期后仍指向原资源与数据源。对已保存 UNKNOWN 重查或重放增长的插入数为零。 |
| 多实例并发 | 两个独立服务实例共用隔离 Store PostgreSQL，确认和回查并发命中同一修订，只有一个 OAuth POST，等待方只使用合格的已提交新凭据。同计划竞态只领取一个已有内核许可。对刷新省略／轮换令牌分别确认保存结果和下一次续期使用值。 |
| 绑定及权限竞态 | 阻塞模拟 OAuth 回执，分别执行断开、重授权开始未完成、同目标新代次绑定、成员店铺权限撤销，以及凭据修订胜出后投递迟到回执。断言旧结果无法写回，失权请求不能用新凭据或查远端，无权调用的企业和过期服务都发不出 OAuth 或 Merchant 请求。对同目标新代次，旧计划不能用于发送，原记录按现有权限仍可查。 |
| 失败、丢回执和重启 | 模拟拒绝、临时且明确未兑换、可证明未发出、服务已处理后丢响应、领取和发出阶段进程退出、PostgreSQL 已保存但回执丢失。分别验证冷却后允许新领取、确定失效／不明时停止重兑、从主库恢复已提交结果，以及无 Merchant 副作用。浏览器核对安全的发送前临时失败仍可继续使用原确认，并核对已经存在尝试或结果不明时只回原记录。 |
| 真实平台 | OAuth 应用及测试用户、实际刷新及撤销行为、Merchant Center 试用账号、专用数据源、真实商品写入和展示资格均待另行授权验收。 |

## Google 隔离试用

在已安装前端依赖和 Playwright Chromium、且 Docker 已运行的开发环境，从仓库根目录执行：

```powershell
$env:GOOGLE_WORKBENCH_BROWSER = '1'
go test ./internal/app/listing -run '^TestGoogleMockAuthorizationPublicationAndSavedResult$' -count=1 -v
```

浏览器经过工作台的真实服务代理和 Go 路由。测试使用合成企业会话和自有商品、两个隔离 PostgreSQL 数据库、模拟的 Google OAuth 与 Merchant API。它检查授权、选择账号及数据源、预览、发送和返回商品页查回记录。恢复普通 Go 用例前，移除本次设置的环境变量。

实际部署前由各数据库 owner 显式安装 Google 表。Store 使用现有 `store-center-schema-init --google-merchant --config ... --confirm-database ...`。Product 使用现有初始化清单里的 `googleMerchant: true` 和 `supplyChain: true`，或对符合当前供货合同的数据库使用 `product-acquisition-init --enable-supply-chain`。应用读取 `runtime.storeCenter.googleMerchant` 私有配置。工作台编译时以 `NEXT_PUBLIC_GOOGLE_MERCHANT_ENABLED=true` 展示入口。真实 OAuth、客户数据源和展示资格按试用账号另行验收。
