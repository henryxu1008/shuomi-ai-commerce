# Shopify 店铺自助授权与商品草稿上传

执行任务 [#1613](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1613) · Independent Architecture · **IMPLEMENTATION_READY / FROZEN**。

调查基线 `7a9df541e51d1a030d32c26f6f18daf21d30f536`。平台接口依据为公开文档，Dev Store 实证待执行。

## 用户结果与范围

2026-11-23，用户明确要求各企业客户自行授权店铺，Amazon 和 Shopify 先做商品，再做订单，并明确商品首版「要能上传产品」。Shopify 首版终点是 Shopify 后台真实可见的商品草稿。用户能在硕米工作台看到提交结果、商品编号和「在 Shopify 查看」。最终销售渠道发布由商家在 Shopify 后台操作。

1. 「店铺中心 → 我的店铺」的店铺详情允许有管理权限的当前企业成员「连接 Shopify」。进入 Shopify 自己的安装页面，由 Shopify 店主选择店铺并授权。安装后，在硕米工作台确认 Shopify 店铺与本地目标对应。首版由店主完成绑定，之后由企业按现有的店铺和商品刊登权限分工。
2. 「商品刊登 → 待上传」复用现有商品和目标店铺的选择、核对和确认。核对页展示将发送的商品、规格、SKU、金额和 Shopify 店铺货币、可用图片及目标店铺。最终动作「上传为 Shopify 草稿」。取消返回原选择。选择的企业、店铺、商品版本或核对字段变化后，原确认失效。
3. 在相同业务流程回看「上传中」「已在 Shopify 建立草稿」「需修改后再上传」或「上传结果待核实」。如果图片仍在 Shopify 处理，结果单独显示图片状态。已有结果放在现有「已上传」视图，可重新核实。店铺断开时提示先重新连接。

首版按单商品显式确认。允许已核实的完整规格组合，超出有界合同的商品在确认前给出可操作说明。范围外包括自动或批量写入、全店商品同步、更新或删除 Shopify 商品、发布到销售渠道、回写库存、订单、买家数据、模型调用及计费变动。两个平台商品阶段均形成可评审结果后，才启动订单实施。

依据是 [店铺中心合同](../../store-center-application.md)、[平台边界](README.md)、[商品刊登现行流程](../listing/README.md)、[Product Domain Contracts](../catalog/product-domain-contracts.md) 及 [当前权限和租户边界](../../platform/identity/auth-and-tenancy.md)。Shopify 上展示的名称、店铺选择和安装入口使用 Shopify 官方提供的界面。

## 事实归属与调用链

| 事实或能力 | 当前 owner、实现位置与交接 |
| --- | --- |
| 当前企业、成员与权限 | 现有 Current Identity、Effective Organization、店铺成员授权和商品刊登授权。沿当前权限键与检查路径，由应用组合层传给领域。 |
| Shopify 外部身份及连接 | Store Center 持有本地 Store、稳定 Shopify Shop GID、规范域名、应用与安装身份、Shopify 店铺的本地不透明标识、当前授权代次和撤销状态。接入现有官方连接和 Store DB owner，以 TikTok/Walmart 的 purpose-qualified 接线、连接状态和审计作为复用参考。 |
| 商品、素材和核对版本 | Product/Catalog、Asset、Listing Target 继续提供已保存版本、允许发送的素材和核对版本。Shopify 规则仅验证载荷，来源数据和媒体许可由原 owner 决定。 |
| 明确意图、一次性发送、回执及未确认结果 | [Listing Submission](../../../../internal/listing/submission/README.md) 的 `ExecutionKernel` 及当前持久化 owner。候选以原执行事务和窄平台结果扩展保存 Shopify 证据。沿现有目标写入、审计和回读 seam。 |
| Shopify 协议和远端商品 | Shopify 持有远端商品。Repo 内 Shopify adapter 提供 OAuth / GraphQL 和 webhook 协议边界、字段转换、结果分类和定向回查。它不另建提交状态机。 |

调用链一为店铺详情 → 同源 BFF → Current Identity 和 Store 管理权限 → Store Center 安装意图 → Shopify 安装与 OAuth → Store Center 原子确认绑定 → 店铺详情。

调用链二为商品刊登 → 同源 BFF → 当前企业、Listing 与 Store live 权限 → 原 Target 核对版本和已批准内容 → Listing Submission 持久化原意图、发放一次性 SendPermit → Store server-only 商品写入句柄 → Shopify GraphQL → Listing 原回执。核实结果使用同一原意图和同一 Shopify Shop GID 的只读句柄。回写 UI 前按当前企业和店铺可见性检查，客户端查询缓存绑定当前企业和目标店铺。

现有 Store 官方产品授权目前只识别 SHEIN、TEMU、TikTok Shop、Walmart，且其凭据结构各异。Shopify 需要自己的可靠店铺身份与授权代次。将 Shopify 接入现有授权入口及商家绑定判别，复用既有 Store 权限和 Listing 执行事务。沿当前各产品的既有服务准入，不由 Shopify 连接赋予平台服务权益。新持久字段、唯一约束、窄运行角色、readiness 与初始化由对应 owner 明确管理。

## Shopify 分发、授权与撤销

- 用公开分发服务相互独立的商家。建议首轮采用有限可见的 App Store 页面，其审核规则与完全公开可见相同。店铺中心跳转到固定的 Shopify 安装入口，随后由 Shopify 发起安装。安装和配置流程不向客户索取 Shopify 域名、API Key 或令牌。应用商店发起的安装也先走 Shopify OAuth，完成后再配对当前硕米企业。公开分发方式由获授权应用管理者选择。[分发](https://shopify.dev/docs/apps/launch/distribution) · [可见性](https://shopify.dev/docs/apps/launch/distribution/visibility) · [审核要求 §2.3](https://shopify.dev/docs/apps/launch/shopify-app-store/app-store-requirements#provide-seamless-and-secure-installation)
- 使用 standalone OAuth authorization code grant 的 online 模式。连接或重新连接须见到 `associated_user.account_owner=true`，且 `associated_user_scope` 与应用当前获准范围均含 `write_products`。用原在线令牌查询 `shop { id myshopifyDomain currencyCode }` 和 `currentAppInstallation { id app { id } accessScopes { handle } }`，核对显式配置的应用身份。只记录店主验证结论及所需范围，不保存员工个人资料或原始响应。Shopify 已安装时的员工访问权限、同店签出的 application-only token 和旧安装的证据均不能用于绑定店铺。[独立应用 OAuth](https://shopify.dev/docs/apps/build/authentication-authorization/authenticate-standalone-apps) · [在线令牌和撤销](https://shopify.dev/docs/apps/build/authentication-authorization/access-tokens#online-access-tokens) · [当前安装](https://shopify.dev/docs/api/admin-graphql/2026-10/queries/currentAppInstallation)
- 安装意图绑定原企业、成员、浏览器会话、StoreID 及目的，最长 10 分钟。Shopify 端先发起且未选店铺时，OAuth 后只允许当前登录用户在有权限的已有 Shopify 店铺记录中显式确认，确认前固定企业、成员和 StoreID。回调仅产生一次性短期证明。Store owner 沿现有密封凭据机制保存配对期间原 online token 的密文，密封关联绑定这次证明、浏览器会话、应用、Shop GID 和 Installation GID。确认时仍用原 online token 向 Shopify 核实同一安装、店铺、scope 和有效性。事务重读当前企业及成员权限，按本地授权代次比较、记录绑定并一次性销毁证明和令牌；失败、撤销、过期及取消均销毁或到期清除。令牌不得出现在浏览器、任务载荷或日志，历史备份恢复亦须先清除过期和已消费的证明。
- 回调验证 Shopify HMAC、时间窗、一次性 state 与浏览器关联，拒绝重复敏感查询参数。外联主机仅来自已验证的 Shopify `myshopify.com` 标识及配置的固定 Shopify 端点，禁止自动跳转。身份由 Shopify Shop GID 决定，域名用于路由。Store owner 对 `应用 + Shopify Shop GID` 建立全局唯一排他绑定和一个随机本地店铺标识。同企业更换本地记录或重装时仍沿原店铺标识，修改关联须有当前 Shopify 店主及硕米原企业权限并显式确认，不复制旧授权。另一企业争用只返回通用错误，不换绑、不泄露原企业、不撤销原连接。现有店铺额度沿 Store 记录创建路径消费。
- 只请求 `write_products`，其包括商品读取。产品写入和回查在现有 Store Purpose 授权通过后用 Shopify client assertion 换取 application-only token。令牌限当前有界服务端调用，不能持久化。发放写入句柄前，用这枚令牌核实 Shopify 的 AppInstallation GID、Shop GID、有效范围和当前 store currency，Store 按当前本地连接代次复核；同一令牌执行唯一 mutation，授权错误、卸载或代次变化时不重新签券再发送。回查需当前同一 canonical shop 的有效连接。应用密钥、签名私钥沿当前秘密管理，声明签名使用已有 JWT 库，Dev 和正式应用配置独立。Dev Store 证实重装、旧令牌失效及 installation identity 行为，若不能区分新旧安装，待设计澄清和复核后才开该能力。[权限管理](https://shopify.dev/docs/apps/build/authentication-authorization/manage-access-scopes) · [Application-only token](https://shopify.dev/docs/apps/build/authentication-authorization/application-only-access-tokens)
- 每个本地授权代次固定一个已验证的 Shopify 安装和一次店主确认。卸载、scope 改变和合规事件走独立的 topic 路径，以原始有界正文验证 HMAC，结合应用路由、严格正文类型及店铺身份去重。事件头及供应商时间仅作校验和排查，不能独立用于开通、复活授权或决定不可逆清除。已知撤销在 Store 事务内停止该代次的新句柄并作废相关未消费证明；身份或前后顺序不明时暂停使用并只读向 Shopify 核实。实时核对和同一令牌发送保证迟到事件不能放行新安装。重装要求新的 Shopify 店主授权及硕米确认，旧已知结果只可按原企业和 canonical shop 回查。[Webhook 验证](https://shopify.dev/docs/apps/build/webhooks/verify-deliveries) · [Webhook 传递](https://shopify.dev/docs/apps/build/webhooks/delivery-structure)

### 卸载和商品权限撤销通知

- 两条固定 POST 分别接收 `app/uninstalled` 和 `app/scopes_update`。按本地应用配置用固定 client secret 校验有界原始正文的 HMAC；验签后解析各主题的独立字段形状。主体分别取已签名的 Shop 数值 ID 或 Shop GID；对权限变更只有从有 `write_products` 变为无该权限时进入撤销核实。其他有效 scope 通知直接确认，未受理为撤销事件。忽略正文中的联系资料，店铺域头不选目标、不用于外部请求。按可信应用、签名 Shop 身份、主题和事件标识去重，核对正文摘要；同一事件标识的不同正文不得改变已受理事件。Header 与平台时间不给予代次或身份授权。[通知样本](https://shopify.dev/docs/api/webhooks/latest#app-uninstalled) · [HTTPS 验签与响应](https://shopify.dev/docs/apps/build/webhooks/verify-deliveries)
- Store Center 在同一事务内受理事件并固定当前 Shop 和授权代次。当前有本地授权时，待核实状态使该代次的商品、回查和服务期数入口暂不可用；无本地授权时只记录去重和确认边界，无远端调用。只在数据库受理成功后返回成功。去重、受理及暂停均不产生扣费、退费、远端变更或清除命令。
- 恢复使用现有运行周期任务。Store 以原事件身份、当前尝试版本、可恢复的到期认领和持久退避驱动。锁外只用原绑定域名对 Shopify 执行安装与权限读取。只有精确应用、店铺与安装且明确保留权限时可完成该待核实项而不撤销。安装已更换，或同一安装明确缺少商品权限，才撤销固定的旧代次。令牌拒绝、网络、响应不完整等只记录待核实并重试。Shopify 的通用令牌失败不能证明卸载。写结论时复核原事件尝试、绑定归属和代次；旧事件或回执不影响新代次。一个事件的核实成功不能放行另一件未核实的有效事件。
- 安装确认沿用在线店主令牌复核。两条安装路径在查询 Shopify 前读取各自 canonical app+Shop 最新的本地受理版本，在确认事务中重新核对。首次绑定也通过该身份的事务锁和受理记录与通知序列化。确认前发生了新受理时，原核实结果作废，必须重新读取 Shopify 后才能确认。已撤销安装的旧证明不能提交，临时证明沿原到期机制清理。事件不得反向锁定安装证明；新授权只能通过原店主和企业确认产生。Shopify 通知用 Store owner 的专用初始化创建空事件表，不迁移旧授权。未安装时原 Shopify 能力照常使用，不能开启通知入口。已安装时，关闭入站仍继续限制并恢复已受理的待核实授权。运行配置及隔离验收保持开发阶段范围。
- 本切片用官方形状及合成验证覆盖验签、两主题、不同企业、首绑和重装时序、同键重放、数据库故障、代次与重试。真实平台拒绝与最终卸载表现，留待获准开发店确认；隐私事件、清除命令和发布审核沿原设计单独跟踪。

### 店铺服务扣减前核实 Shopify 授权

独立增量设计复核为 `IMPLEMENTATION_READY`，绑定基线 `e5bcc655c81eb346b2ac3d251bee97a335426313` 和受审设计对象 `3af56454435d27de4755e34545ad9323e5e92d7d`。同店换绑与未撤销代次的事务核对关闭了复核发现的阻塞。

Design Basis 增量依据为本任务已准入的 Shopify 授权合同、[店铺服务资源合同](../../unified-base-prepaid-resources.md)和用户对 #1613 的要求。修复阻止商家在 Shopify 应用已卸载、商品权限已撤销或平台无法核实时仍消耗店铺服务期数。覆盖 Shopify 首次开通、续费和重新开通。公共撤销通知、连接列表的展示状态、Shopify 审核、价格、额度计算及资源 Owner 沿原范围。

- Store Center 继续拥有当前授权绑定和店铺服务效果。现有 Shopify 集成使用应用令牌和只读 Admin 查询核对 Shopify 店铺、应用、安装身份及 `write_products`，保持开发店限制。读取请求一律使用当前成员可访问店铺的已验证域名。网络调用前后核对本地绑定代次，确认无变化才视为通过。本地列表状态不能作为 Shopify 扣减授权。
- 正常服务命令沿现有成员鉴权、原命令查询、用量政策之后核实 Shopify 授权，通过才签入 Store 原意图。挂起的原意图沿原操作标识获取或恢复额度预留。提交 Store 服务变更前再核实授权，并把结果绑定当前企业、本地 Store、ShopKey、授权代次和安装身份。Store 事务先锁店铺并保留原终态优先，再在同一事务锁住当前 Shopify 关系及授权行，确认该 ShopKey 仍属于当前 Store 且同一代次未撤销，随后检查成员权限和服务变更。Shopify 关系锁先于原店成员授权锁，避免与跨本地店铺换绑的锁序相反。店铺版本和连接引用的既有校验继续有效。读 Shopify 设有独立五秒总上限，网络调用在 Store 事务外执行。
- 首次核实失败不会创建新意图或预留。第二次核实失败则由原 Store 操作记录可证明的本地未执行结果，沿现有资源结算释放同一预留。Shopify 无响应也阻止 Store 执行，失败证明只说明工作台未提交服务变更。若另一个同键执行已写入终态，原终态优先并按原证据结算。成功回执的重放不依赖当前 Shopify 授权。应用的授权判定点为服务变更前最后一次 Shopify 验证，Shopify 与 Store 事务之间没有跨系统原子锁。
- 保留现有 Store/Resource 事务、幂等身份、预留恢复与成员额度归属，无 schema 或旧数据迁移。隔离测试覆盖失效及核实失败、预留中撤销及释放、原回执重放、同店换绑期间旧本地记录的阻断、成员与企业用量，以及有效连接仍可开通。Shopify 真店行为使用批准测试店另行验证。

## 合规清除与恢复

- `customers/data_request`、`customers/redact`、`shop/redact` 为公开应用必接事件，无效签名返回 401。确认前持久化事件受理结果或清除命令。首版无买家或订单数据，买家事件只存不可关联个人的处置回执。Store Center 持有店铺清除命令及完成状态，Listing 仅处理自己拥有的回执和证据。 [隐私履约](https://shopify.dev/docs/apps/build/compliance/privacy-law-compliance)
- `shop/redact` 正文只有店铺身份。Store Center 从当前权威授权历史和 Shopify 安装核实固定已撤销的代次，清除命令保存不可变的原应用、内部 shop 标识、原组织、原授权代次集合及受理身份。必须已能证明目标为旧代次，Shopify 仍有效或新建立的当前代次排除在外。代次无法判定时持久保留待核实，由 Store Center 核实原卸载证据或向 Shopify 确认，绝不按当前绑定推断清除范围。Shopify 的 30 天履约时限随受理命令保留及告警。
- Store 受理事务同时禁止该旧代次新授权、回执展示和发送，原执行结果只接受满足代次 fence 的写入。清除执行以原命令身份重试。Listing 对受影响代次和创建目标加锁，先设拒绝晚到写入的屏障、记录源商品防重复限制，再删除旧目标关联、复制的 Shopify 事实、回执和读取证据，同一事务保存无店铺身份的完成回执。Store 清除命令范围的旧证明和授权材料。共享的域名、Shop GID 和本地店铺标识在新代次仍引用时保留；没有引用时，只在 Listing 完成并核对后才解除排他绑定并清除店铺标识。这个顺序保证并发重装使用已有创建目标或已生效的源商品限制。跨库复用现有 Temporal 运行底座及 Store durable command。提交命令后可按原键补启动，重复执行各 owner 的同键操作。Store 收齐两边确认才记录履约完成。执行权限定到 Shopify 私有列和记录，保留自有商品与独立商业事实。
- 所有可识别的 Shopify 字段须可按被撤销代次清除，通用长期日志、内核证据字符串和 Temporal 参数只使用本地不透明任务标识符以及受控结果类别。备份责任归相应数据库 owner，恢复时先封闭 Shopify 读取和写入，重放备份时间点之后的清除命令，核对 Store 与 Listing 的完成回执，再恢复访问。清除完成后仅保留重放所需的本地旧代次标识、命令身份和状态，保存到相关备份过期，访问限恢复。不能安全重放的备份不能直接接回运行实例。

## Shopify 写入、回查及结果

锁定 GraphQL Admin API `2026-10`。使用公开 `productSet` mutation 的同步执行模式，省略 upsert 的 `identifier`，发送 `status: DRAFT`。使用 app-owned、storefront 不可读的产品自定义 ID 记录不透明的提交来源键。该键对应用/本地商品/目标店稳定，使用声明式 Shopify metafield 定义和其唯一约束。目标或载荷不同不能复用原确认。平台定义及权限在发送前作应用配置验证。用 `productByIdentifier` 只读核对原远端证据。同一来源已确认创建时只展示原结果，后续修改由商家在 Shopify 管理。禁止把 Shopify upsert 用作未知结果重试，因为它可覆盖规格和其他列表字段。Shopify 自定义 ID 的平台唯一约束、创建时写入和录入失败语义由 Dev Store 实证，代码对未知结果始终保守。[productSet](https://shopify.dev/docs/api/admin-graphql/2026-10/mutations/productSet) · [自定义 ID](https://shopify.dev/docs/apps/build/metafields/working-with-custom-ids) · [productByIdentifier](https://shopify.dev/docs/api/admin-graphql/2026-10/queries/productByIdentifier)

- 允许发送标题、经净化的描述、明确的商品选项与规格、SKU、该店铺货币下用户核对的十进制定价，以及已批准且可由 Shopify 读取的图片。价格币种从真实店铺读取，原价币种不同则要求在核对时输入目标店价，禁止静默换算和用零填补。缺必需标题、价格或规格组合冲突在发送前可修正。图片仅用现有资产 owner 许可给 Shopify 读取的稳定、受控 HTTPS 来源。核对页明确本次随商品发送的图片。先验证 `productSet.files` 直接引用批准素材在 `write_products` 下的实际权限与媒体结果；需要独立文件写入权限时回报证据，相关能力保留未开放。本路径不调用独立的 `fileCreate` 或库存、商品发布接口。 [productSet 字段](https://shopify.dev/docs/api/admin-graphql/2026-10/input-objects/ProductSetInput)
- request body 沿现有 Listing 契约限制，Shopify 单次限制暂定最多 3 个选项维度、30 个规格、10 张图，数值在 Shopify 候选验证时按有界单商品路径确认。外部请求沿已有有界 transport、上下文和 deadline，由 provider 响应给出成本和限流信息。日志仅保留受控错误类别与询问所需编号，不含凭据、授权码、完整载荷或 Shopify 原始错误。商家看到可行动的字段提示。
- 原提交身份绑定企业、店铺、应用、Shop GID、目标版本、币种和规范化载荷指纹。Listing 在发放 SendPermit 的现有事务内同时锁定 Shopify 创建目标，用 `企业 + Store owner 分配的稳定本地 shop 标识 + 不随编辑/重新建 Target 变化的源商品 ID` 做唯一键。该窄索引只指向当前原意图和已有回执，没有第二套提交状态。通用内核仍使用原账户和 Target 关系；Shopify 准入必须先通过此索引，不能因为新的本地 StoreID、Target、商品版本、操作键或重装而绕过。成功、有进行中尝试和结果待核实的目标都不能发新许可。只有原 owner 已持久化确定未创建或未发送证据，且用户重新核对并确认，才可在同槽创建新尝试。
- 确认时显示真实店铺货币。发送前按同一安装重新核对当前成员、权限、连接代次、版本及币种。若在发放许可之后、实际发送之前发现变化，以持久可验证的未发送证据关闭原尝试，用户更正后重新确认。没有肯定未发送证据时原操作保持待核实。没有跨 Store/Listing DB 的假原子事务，已送到 Shopify 的请求按原提交记录结果。
- Shopify 不保证重装后的 app-owned metafield 仍在。保有合法旧回执时按原 Product GID 在同店查当前事实；结果待核实只采纳可靠匹配的原标识。店铺数据清除前，Listing 将旧授权代次的尝试归档为不可对外读取，并针对仍归本企业的受影响源商品保留无店铺/应用标识且不可关联回执的本地“历史上传需核实”限制，范围限已有成功、待核实或可能在途的原尝试。清除精确店铺关联后，如果无法以可靠证据判断新目标与旧目标不同，显示需核实并拒绝再次创建。空读、更换店铺记录、重装和清除都不会解除限制。正常未受影响商品及已有可确定的不同目标按现有路径处理。[自定义标识卸载行为](https://shopify.dev/docs/apps/build/metafields/definitions#app-uninstall-and-reinstall)

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
- 通过现有 Listing 持久执行测试覆盖双击、跨进程并发、响应丢失、跨本地 StoreID 与商品版本的目标冲突、清除与同店重装的并发、没有正面回查证据时保持待核实、已存在商品被商家改动时不覆盖。覆盖上传草稿和媒体待处理的反馈。沿既有 Store/BFF/刊登测试检查窄屏、键盘确认、错误反馈、切企业和刷新后的同一结果。
- 先用获准 Shopify Dev Store 的生成测试数据及两个测试企业验证授权。同一 Dev Store 可以作为跨企业绑定冲突的测试目标，另一 Dev Store 验证两个企业各用自己的店。核对 Shopify 后台的商品 GID、DRAFT、商品内容、规格/价格/图片、权限及重装。在专用开发店演练未知回查和应用合规事件；真实商家试用另需发布后取得权限。 [Shopify 开发店](https://shopify.dev/docs/apps/build/stores) · [生成测试数据](https://shopify.dev/docs/apps/build/stores/generated-test-data)
- 公开应用审核另外需要获授权的 Dev Dashboard、提交资格、TLS 回调、隐私政策和演示/测试说明，适用时提供专用测试登录。核对 Dashboard 实际身份验证和应用审核状态，当前 Shopify 对同一组织的新提交有数量与已有应用付费店安装门槛。若 Shopify 判定当前硕米收费涵盖此公开应用，先由产品与应用 owner 确认 Shopify Billing 条款及商业安排。本批没有批准增设收费或更改现有套餐。 [审核提交](https://shopify.dev/docs/apps/launch/app-store-review/submit-app-for-review) · [当前 App Store 要求](https://shopify.dev/docs/apps/launch/shopify-app-store/app-store-requirements)

Legacy decision 为 **EXTRACT**。复用现行 Store 角色、授权和绑定 seam、Listing 确认与执行内核、当前资产白名单及标准 UI，不接入旧租户或旧店铺链。

## 开发店候选配置

- 当前运行时仅允许 Shopify 证实为开发店的安装执行授权和商品调用。以空白隔离环境的 Store 与 Product 当前结构启动。店铺库使用 `store-center-schema-init --shopify` 显式安装 Shopify 结构和授予运行权限，再配置 `storeCenter.shopify`，分别使用开发应用 ID、OAuth 凭据、配对加密密钥及应用签名私钥。回调和启动地址分别注册为工作台域名下的 `/api/shopify/callback` 与 `/api/shopify/launch`。
- 开发应用使用 `write_products`。在应用配置声明 `[product.metafields.app.product_upload_source]`，类型为 `single_line_text_field`，启用 `capabilities.unique_values`，保持 Storefront 无读取权限。开始写入前核实店铺上的实际定义。[Shopify 元字段能力](https://shopify.dev/docs/apps/build/metafields/use-metafield-capabilities)
- 从「店铺中心 → 我的店铺」创建 Shopify 店铺并完成官方授权。在「商品刊登」选择该店铺，填写当前店铺币种下的售价并选择素材，保存资料，再在上传计划核对后确认。回到商品行可查看 Shopify 商品编号、图片状态或核实原操作。核实到账户店铺的商品后，可以直接打开 Shopify 后台的对应商品。开发店安装、商品写入、图片导入和重装待实际验证。
- 公共商家试用仍须接通 Shopify 端发起的安装恢复、关系撤销与合规清除处理，并走 Shopify 审核。开发店候选保持正式店拦截。

### 从 Shopify 发起安装的隔离候选

开发应用继续使用现有启动及回调地址。配置 `storeCenter.shopify.startedInstallation` 后，同一地址也能接收没有预选店铺的 Shopify 签名请求。店主先完成 Shopify OAuth，再回到硕米工作台登录并在店铺中心选择一条已有 Shopify 店铺记录。工作台确认界面从 Store Center 读取已核实的 Shopify 域名。确认时固定当前企业、成员、店铺和版本，在提交事务中消费原证明并连接店铺。同一原确认可读回已保存的结果。

此候选使用 `store-center-schema-init --shopify` 在空白隔离 Store 安装短期安装证明表。服务只接受经过核实的开发店。工作台会在证明过期后清除该记录和密文。公开分发、撤销通知、合规数据清除、实际应用审核和客户试用继续单独验证。

## 独立架构评审

第 1 轮独立检查绑定 `189fbbc6178a23044476d42c78b94fef987eba66`，结论 `NOT_READY`。2026-11-24，原独立评审者对 [PR #1615](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1615) 的修订 `ced51c5dfba09664ad13c12eb828f298d049dae4` 增量复核，三个实施级 BLOCKER 已解除，结论为 `IMPLEMENTATION_READY`。生产开放仍须满足受控试用和平台前置。

| Finding | 受影响的 Must / 阻塞后果 | 本次设计修订 |
| --- | --- | --- |
| B1 授权发起人及安装代次 | 企业授权与身份所有权。只读员工或旧授权可能取得本地写入资格。 | 店主及实际范围验证、原 online 证明消费、安装与本地代次 fence、活跃安装二次核对。 |
| B2 稳定创建目标 | 重复且不可安全恢复的外部商品创建。换记录、重装或清除后可能再次创建。 | 原提交事务中的 Shopify 唯一创建目标、既有结果优先与清除后保守限制。 |
| B3 迟到的清除 | 不可逆数据丢失及不安全恢复。旧清除可能处理新绑定的结果。 | 不可变撤销代次和 owner 范围、执行和恢复的持久防护、未确认时停止清除。 |

R1 的平台实证、上传结果分支和合规履约用例归为 `IMPLEMENTATION_TEST`。设计范围以外的生产修改、Dev Store 请求、审核提交、真实客户验收、合并与部署均为 **NOT_RUN**。
