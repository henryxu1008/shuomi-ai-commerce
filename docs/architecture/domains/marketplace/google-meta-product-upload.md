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
