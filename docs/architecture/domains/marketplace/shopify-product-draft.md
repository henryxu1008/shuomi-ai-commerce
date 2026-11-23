# Shopify 店铺自助授权与商品草稿上传

执行任务 [#1613](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1613) · Independent Architecture · **NOT_READY** · 待独立架构评审。

调查基线 `7a9df541e51d1a030d32c26f6f18daf21d30f536`。下文是待评审设计。平台接口依据为公开文档，Dev Store 实证待执行。

## 用户结果与范围

2026-11-23，用户明确要求各企业客户自行授权店铺，Amazon 和 Shopify 先做商品，再做订单，并明确商品首版「要能上传产品」。Shopify 首版终点是 Shopify 后台真实可见的商品草稿。用户能在硕米工作台看到提交结果、商品编号和「在 Shopify 查看」。最终销售渠道发布由商家在 Shopify 后台操作。

1. 「店铺中心 → 我的店铺」的店铺详情允许有管理权限的当前企业成员「连接 Shopify」。进入 Shopify 自己的安装页面，商家选择店铺并授权。安装后，在硕米工作台确认显示的 Shopify 店铺与本地目标对应。
2. 「商品刊登 → 待上传」复用现有商品和目标店铺的选择、核对和确认。核对页展示将发送的商品、规格、SKU、金额和 Shopify 店铺货币、可用图片及目标店铺。最终动作「上传为 Shopify 草稿」。取消返回原选择。选择的企业、店铺、商品版本或核对字段变化后，原确认失效。
3. 在相同业务流程回看「上传中」「已在 Shopify 建立草稿」「需修改后再上传」或「上传结果待核实」。如果图片仍在 Shopify 处理，结果单独显示图片状态。已有结果放在现有「已上传」视图，可重新核实。店铺断开时提示先重新连接。

首版按单商品显式确认。允许已核实的完整规格组合，超出有界合同的商品在确认前给出可操作说明。范围外包括自动或批量写入、全店商品同步、更新或删除 Shopify 商品、发布到销售渠道、回写库存、订单、买家数据、模型调用及计费变动。两个平台商品阶段均形成可评审结果后，才启动订单实施。

依据是 [店铺中心合同](../../store-center-application.md)、[平台边界](README.md)、[商品刊登现行流程](../listing/README.md)、[Product Domain Contracts](../catalog/product-domain-contracts.md) 及 [当前权限和租户边界](../../platform/identity/auth-and-tenancy.md)。Shopify 上展示的名称、店铺选择和安装入口使用 Shopify 官方提供的界面。

## 事实归属与调用链

| 事实或能力 | 当前 owner、实现位置与交接 |
| --- | --- |
| 当前企业、成员与权限 | 现有 Current Identity、Effective Organization、店铺成员授权和商品刊登授权。沿当前权限键与检查路径，由应用组合层传给领域。 |
| Shopify 外部身份及连接 | Store Center 持有本地 Store、稳定 Shopify Shop GID、规范域名、应用身份、当前授权代次和撤销状态。接入现有官方连接和 Store DB owner，以 TikTok/Walmart 的 purpose-qualified 接线、连接状态和审计作为复用参考。 |
| 商品、素材和核对版本 | Product/Catalog、Asset、Listing Target 继续提供已保存版本、允许发送的素材和核对版本。Shopify 规则仅验证载荷，来源数据和媒体许可由原 owner 决定。 |
| 明确意图、一次性发送、回执及未确认结果 | [Listing Submission](../../../../internal/listing/submission/README.md) 的 `ExecutionKernel` 及当前持久化 owner。候选以原执行事务和窄平台结果扩展保存 Shopify 证据。沿现有目标写入、审计和回读 seam。 |
| Shopify 协议和远端商品 | Shopify 持有远端商品。Repo 内 Shopify adapter 提供 OAuth / GraphQL 和 webhook 协议边界、字段转换、结果分类和定向回查。它不另建提交状态机。 |

调用链一为店铺详情 → 同源 BFF → Current Identity 和 Store 管理权限 → Store Center 安装意图 → Shopify 安装与 OAuth → Store Center 原子确认绑定 → 店铺详情。

调用链二为商品刊登 → 同源 BFF → 当前企业、Listing 与 Store live 权限 → 原 Target 核对版本和已批准内容 → Listing Submission 持久化原意图、发放一次性 SendPermit → Store server-only 商品写入句柄 → Shopify GraphQL → Listing 原回执。核实结果使用同一原意图和同一 Shopify Shop GID 的只读句柄。回写 UI 前按当前企业和店铺可见性检查，客户端查询缓存绑定当前企业和目标店铺。

现有 Store 官方产品授权目前只识别 SHEIN、TEMU、TikTok Shop、Walmart，且其凭据结构各异。Shopify 需要自己的可靠店铺身份与授权代次。将 Shopify 接入现有授权入口及商家绑定判别，复用既有 Store 权限和 Listing 执行事务。沿当前各产品的既有服务准入，不由 Shopify 连接赋予平台服务权益。新持久字段、唯一约束、窄运行角色、readiness 与初始化由对应 owner 明确管理。

## Shopify 分发、授权与撤销

- 用公开分发服务相互独立的商家。建议首轮采用有限可见的 App Store 页面，其审核规则与完全公开可见相同。店铺中心跳转到固定的 Shopify 安装入口，随后由 Shopify 发起安装。安装和配置流程不向客户索取 Shopify 域名、API Key 或令牌。应用商店发起的安装也先走 Shopify OAuth，完成后再配对当前硕米企业。公开分发方式由获授权应用管理者选择。[分发](https://shopify.dev/docs/apps/launch/distribution) · [可见性](https://shopify.dev/docs/apps/launch/distribution/visibility) · [审核要求 §2.3](https://shopify.dev/docs/apps/launch/shopify-app-store/app-store-requirements#install-and-authenticate-correctly)
- 候选使用 standalone OAuth authorization code grant。安装时用在线凭据核实这次商家授权和 `shop { id myshopifyDomain currencyCode }`，不保存 associated user 的个人资料和原始响应。产品写入和后续回查在已绑定且有当前业务权限时请求短时 application-only token，使用 Shopify 支持的 client assertion。短时 token 仅在服务端请求或短缓存内使用，缓存键包含应用、店铺和授权代次。此路径避免为同一家店的不同配对请求轮转已连接安装的离线凭据。Dev Store 要先证实正式拟用的授权方式以及 application-only token 可调用的范围和权限。若路径不可用，调整设计和适用评审后才启用相应能力。[独立应用 OAuth](https://shopify.dev/docs/apps/build/authentication-authorization/authenticate-standalone-apps) · [Application-only token](https://shopify.dev/docs/apps/build/authentication-authorization/application-only-access-tokens)
- 硕米端发起时，安装意图有有限 TTL，绑定原企业、成员、会话、StoreID 及目的。Shopify 端发起且未选店铺时先建立等效的短期浏览器配对证明，OAuth 后仅允许当前登录用户在有权限的已有 Shopify 店铺记录中显式确认。外部回调只能产生短期证明。Store owner 在最终事务中再次核对原登录和当前权限，并一次性消费该证明。企业上下文改变、撤权、取消或超过期限均要求重新开始。
- 回调严查规范 Shopify HMAC、时间窗、一次性 state 与浏览器关联，拒绝重复敏感查询参数。外联主机仅来自已验证的 Shopify `myshopify.com` 店铺标识及配置的固定 Shopify 端点，显式禁止自动跳转。真实店铺身份由平台返回的 canonical Shop GID 决定，域名是当前路由属性。Store 内对应用和 Shopify Shop GID 实施跨企业排他绑定及版本约束。另一企业尝试连接时只收到通用错误，不能换绑、撤销原连接、获知原企业或读取旧记录。现有店铺额度沿 Store 记录创建路径消费。
- 只请求 `write_products`，其包括商品读取。安装完成及 API 调用按实际有效 scope 和本地当前权限判定；禁止因 scope 足够而绕过本地商品和店铺角色。短时令牌禁止进入数据库、浏览器、任务载荷和日志。应用客户端密钥与签名私钥沿当前服务端秘密管理，声明签名使用已有 JWT 库。配置中分开开发应用和正式应用，缺配置时关闭 capability。禁止在生产继续使用开发应用或测试店凭据。[权限管理](https://shopify.dev/docs/apps/build/authentication-authorization/manage-access-scopes)
- 接收签名的 `app/uninstalled`、`app/scopes_update` 和三项合规事件。当前连接撤销、scope 被移除或失效时，停止新发送及远端回查，清理短时缓存，旧响应不能复活连接。注册事件以原始正文验证 HMAC，校验 topic、应用入口和正文中的店铺身份，使用供应商事件身份去重；Shopify 事件时间用于保守判断代次，身份或前后顺序不明时先暂停凭据使用，重新向官方验证。回调返回前持久化必要事件或已处理结果，供原 owner 幂等重试。重装须重新配对，不能从旧授权自动取得新发送许可。一个已核实的旧提交可在同一企业、同一 Store、同一 Shopify Shop GID 重新授权后只读核实。[Webhook 传递](https://shopify.dev/docs/apps/build/webhooks/delivery-structure) · [隐私履约](https://shopify.dev/docs/apps/build/compliance/privacy-law-compliance)
- 合规端点为 `customers/data_request`、`customers/redact`、`shop/redact`。首版没有读取买家或订单数据，买家事件仅记录履约必要回执。店铺清除由 Store owner 建立撤销与删除意图，Listing owner 删除其可识别的 Shopify 回执和关联。执行内核的长期记录仅存本地不透明标识、载荷指纹及受控结果类别，Shopify 域名、GID、远端消息正文不得写入通用长期日志和内核证据字符串。跨库清除复用当前持久任务机制，两个 owner 同键确认后才标记已履约。核实备份恢复后的删除重放与 Shopify 的 30 天时限。保留用户原有自有商品及独立商业数据。若现有 schema 无法按合同履约，先在设计复核确认窄清除接线及备份责任。

## Shopify 写入、回查及结果

锁定 GraphQL Admin API `2026-10`。使用公开 `productSet` mutation 的同步执行模式，省略 upsert 的 `identifier`，发送 `status: DRAFT`。使用 app-owned、storefront 不可读的产品自定义 ID 记录不透明的提交来源键。该键对应用/本地商品/目标店稳定，使用声明式 Shopify metafield 定义和其唯一约束。目标或载荷不同不能复用原确认。平台定义及权限在发送前作应用配置验证。用 `productByIdentifier` 只读核对原远端证据。同一来源已确认创建时只展示原结果，后续修改由商家在 Shopify 管理。禁止把 Shopify upsert 用作未知结果重试，因为它可覆盖规格和其他列表字段。Shopify 自定义 ID 的平台唯一约束、创建时写入和录入失败语义由 Dev Store 实证，代码对未知结果始终保守。[productSet](https://shopify.dev/docs/api/admin-graphql/2026-10/mutations/productSet) · [自定义 ID](https://shopify.dev/docs/apps/build/metafields/working-with-custom-ids) · [productByIdentifier](https://shopify.dev/docs/api/admin-graphql/2026-10/queries/productByIdentifier)

- 允许发送标题、经净化的描述、明确的商品选项与规格、SKU、该店铺货币下用户核对的十进制定价，以及已批准且可由 Shopify 读取的图片。价格币种从真实店铺读取，原价币种不同则要求在核对时输入目标店价，禁止静默换算和用零填补。缺必需标题、价格或规格组合冲突在发送前可修正。图片仅用现有资产 owner 许可给 Shopify 读取的稳定、受控 HTTPS 来源。核对页明确本次随商品发送的图片。先验证 `productSet.files` 直接引用批准素材在 `write_products` 下的实际权限与媒体结果；需要独立文件写入权限时回报证据，相关能力保留未开放。本路径不调用独立的 `fileCreate` 或库存、商品发布接口。 [productSet 字段](https://shopify.dev/docs/api/admin-graphql/2026-10/input-objects/ProductSetInput)
- request body 沿现有 Listing 契约限制，Shopify 单次限制暂定最多 3 个选项维度、30 个规格、10 张图，数值在 Shopify 候选验证时按有界单商品路径确认。外部请求沿已有有界 transport、上下文和 deadline，由 provider 响应给出成本和限流信息。日志仅保留受控错误类别与询问所需编号，不含凭据、授权码、完整载荷或 Shopify 原始错误。商家看到可行动的字段提示。
- 上游建立一次稳定提交身份，绑定原企业、原店铺、Shop GID、应用身份、稳定来源 ID、准确商品/目标版本及规范化载荷指纹。持久意图/目标 fence 提交后才能发放 SendPermit。调用前再次检查当前成员、权限、连接代次及目标版本。已批准数据在此次发送期间冻结。跨进程并发或重试通过原 Listing intent 和 fence 收敛，店铺 OAuth 与 Listing 不跨库做假原子事务。

| 证据 | 持久结果及下一步 |
| --- | --- |
| Shopify 返回可验证的 Product GID、原键及草稿状态 | 以原意图保存商品编号、时间、草稿证据和必要媒体状态。受控后台链接以经过验证的店铺身份和 Product GID 构造。后续状态独立回查。 |
| Shopify 已证实未创建且返回可修正的拒绝 | 记录原拒绝并给出可操作的字段反馈。重新核对后显式确认新尝试，继续使用相同创建目标。只允许安全证据充分的拒绝分支。 |
| HTTP 超时、响应丢失、部分响应、平台返回与本地保存之间中断 | 原提交保持待核实。只读按准确的 app-owned 自定义 ID 核对 Shop GID、Product GID、原提交证据及实际状态。空读、读权限失效或返回不匹配的商品均不作成功或未发送证据。禁止自动重发和换新键。 |
| Shopify 草稿已建，但媒体处理仍在进行或失败 | 展示真实商品编号、草稿与图片状态，保留 Shopify 管理入口。禁止重新创建商品。 |
| 当前平台商品已被商家更新、发布或删除 | 保持原上传回执，另标最新观察、缺失或状态变化。若收敛未知结果时状态已变化，只有可信原键、身份和结果证据才能采用原创建结果，当前状态照真实数据呈现。 |
| 应用卸载、scope 变化、离开企业或本地撤权 | 过期绑定停止新调用和凭据使用。同一个旧结果只能经当前同店权限重新确认，店铺绑定变更不继承原发送许可。 |

## 隔离验证、受控试用和准入

- TDD 覆盖当前 Issue 的高风险路径，优先复用既有真实隔离 PostgreSQL 和 provider fixture transport。验证双企业、两个 Store、同一 Shopify 店铺争用、回调重放/取消/过期、授权重试、撤销与晚到结果、有权的原提交回查。许可、shop binding、应用、商品版本和 scope 每一项不匹配均要拒绝。
- 通过现有 Listing 持久执行测试覆盖双击、跨进程并发、响应丢失、幂等冲突、没有正面回查证据时保持待核实、已存在商品被商家改动时不覆盖。覆盖上传草稿和媒体待处理的反馈。沿既有 Store/BFF/刊登测试检查窄屏、键盘确认、错误反馈、切企业和刷新后的同一结果。
- 先用获准 Shopify Dev Store 的生成测试数据及两个测试企业验证授权。同一 Dev Store 可以作为跨企业绑定冲突的测试目标，另一 Dev Store 验证两个企业各用自己的店。核对 Shopify 后台的商品 GID、DRAFT、商品内容、规格/价格/图片、权限及重装。在专用开发店演练未知回查和应用合规事件；真实商家试用另需发布后取得权限。 [Shopify 开发店](https://shopify.dev/docs/apps/build/stores) · [生成测试数据](https://shopify.dev/docs/apps/build/stores/generated-test-data)
- 公开应用审核另外需要获授权的 Dev Dashboard、提交资格、TLS 回调、隐私政策和演示/测试说明，适用时提供专用测试登录。核对 Dashboard 实际身份验证和应用审核状态，当前 Shopify 对同一组织的新提交有数量与已有应用付费店安装门槛。若 Shopify 判定当前硕米收费涵盖此公开应用，先由产品与应用 owner 确认 Shopify Billing 条款及商业安排。本批没有批准增设收费或更改现有套餐。 [审核提交](https://shopify.dev/docs/apps/launch/app-store-review/submit-app-for-review) · [当前 App Store 要求](https://shopify.dev/docs/apps/launch/shopify-app-store/app-store-requirements)

Legacy decision 为 **EXTRACT**。复用现行 Store 角色、授权和绑定 seam、Listing 确认与执行内核、当前资产白名单及标准 UI，不接入旧租户或旧店铺链。

架构独立评审聚焦租户绑定、短时授权签发与撤销、外部创建/回查、合规清除、对外开放前置。当前未获 `IMPLEMENTATION_READY`。设计之外的生产修改、Dev Store 请求、审核提交、真实客户验收、合并与部署均为 **NOT_RUN**。
