# Amazon 美国站商品上传

- 执行 Issue [#1614](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1614)。关联 [#32](https://github.com/henryxu1008/shuomi-ai-commerce/issues/32)、[发布总门禁 #33](https://github.com/henryxu1008/shuomi-ai-commerce/issues/33)。
- Design Basis **Independent Architecture**。Admission Status **IMPLEMENTATION_READY**，限已授权的隔离实现。独立复核 R1 提出的两个阻塞已修正，R2 完成增量复核。真实平台放行条件见下文。
- 调查基线 `7db22705dd9b4986a4e98cdb37ab6e615c12bace`。

## 用户操作与首版范围

2026-11-23 用户要求不同企业的客户自行授权，并明确商品首版「要能上传产品」。Amazon 和 Shopify 商品阶段都形成可评审结果后再实施订单。基础流程沿 [现行业务决定](../../../product/final-ui-ia-authority.md) 和 [商品刊登合同](../listing/README.md)。

1. 企业有店铺管理权限的人员从「店铺中心 → 我的店铺」连接 Amazon 美国站。卖家在 Seller Central 授权硕米智能引擎的公开应用，返回工作台查看连接结果。只有经过 Amazon 证实拥有美国站经营资格的卖家绑定才能成为目标。
2. 获店铺和刊登权限的运营人员从「商品刊登 → 待上传」选择已保存商品和该店铺。复用现有选择、资料核对和确认提交步骤。显示站点、卖家 SKU、GTIN、品牌、商品类型、美元售价、可卖数量、卖家发货和官方校验结果，核对后由用户明确确认提交。真实 0 保留，缺值要求填写，来源报价和采集库存不会转成该店的售价和库存。主操作用「提交到 Amazon」，确认文案说明 Amazon 处理后商品可能公开销售。切企业、改店铺或更新待提交资料会使旧确认失效。返回或取消保留可编辑资料。测试草稿沿原规则禁止提交。
3. 原提交结果页返回可查的卖家 SKU、提交回执和可取得的 ASIN。提交后保留结果入口，允许按原身份查看最新状态。「已上传」继续作为只读结果入口，只有正式取得受理证据的记录才进入该类展示。待确认及失败从原操作结果定位。用户可以看见要补的字段、需要到 Amazon 处理的资格问题，或稍后核对的提示。使用现有表单、错误组件、键盘焦点与窄屏布局。

本候选以美国站 Seller、单件商品、单卖家 SKU、已有合法 GTIN、具备该品类销售资格、由卖家发货的新品为首个实证样本。实现围绕首个合格商品类型提供完整规则、资料和结果闭环。已有 ASIN 命中或候选有歧义时提示在 Seller Central 确认，现阶段不发送。首版范围为真实新刊登，排除订单、买家个人资料、批量与变体、GTIN 豁免、已有 SKU 的更新或删除、自动定价、自动 AI 执行和全店商品同步。首个真实卖家、合格商品及类型目前未知。

## 官方协议与应用准入

| 目的 | 采用的合同 |
| --- | --- |
| 客户自行授权 | 注册 **Public / Sellers / Production** SP-API 应用。应用及开发者资料需要 Amazon 批准的 **Product Listing** 角色。官网流程可从工作台开始，必须登记公开 HTTPS 登录和回调地址。生产试用资格核对门户的实际应用状态和授权上限。Amazon 单列未刊登公开卖家应用最多 25 个 OAuth 授权，列入 Appstore 后不限量。该上限与审核通过分开记录，面向客户发行按门户和适用的 Appstore 要求执行。 |
| 美国站绑定 | Seller Central 美国站，SP-API 北美端点 `sellingpartnerapi-na.amazon.com`，站点 `ATVPDKIKX0DER`。以同一 access token 调 `getMarketplaceParticipations` 核实美国站参与资格及暂停状态，再用 `searchListingsItems` 对回调给出的 seller ID 验证卖家级访问。连接探测不导入商品。应用只使用 Product Listing，对接仅放行本表的必要接口。 |
| 目录与商品资料 | 用 `searchCatalogItems` 按真实 GTIN 查重，避免把他人公开页面的 ASIN 当成本商品已验证的 ASIN。用 Product Type Definitions 取得当前 seller、marketplace、productType、`LISTING` 需求的定义和标识。缺资料、未支持的官方字段、规则版本失效或目录/资格不能核实时停止。 |
| 预览、写入与读回 | 用 Listings Items v2021-08-01 对同一 seller、marketplace 和 SKU 校验预览及正式 `putListingsItem`，用 `getListingsItem` 读回 `summaries,issues`。预览 `VALIDATION_PREVIEW` 不保存目录内容，本地资料仍沿原草稿。正式调用可能创建可销售的刊登，当前未发现可用于首版的 Amazon 远端草稿合同。 |
| 额度与环境 | 复用现有受控 HTTP 客户端、超时、限制和配置。Amazon Listings Items 提供静态 sandbox，适合检查协议；真实 seller grant、规则、图片抓取、账户资格和正式写入分别验证。真实 Preview 仍调用真实授权平台，须按真实操作范围批准。 |

官方依据 [网站授权](https://developer-docs.amazon/sp-api/docs/website-authorization-workflow)、[注册公开开发者](https://developer-docs.amazon/sp-api/docs/register-as-a-public-developer)、[网站要求](https://developer-docs.amazon/sp-api/docs/website-guidelines)、[发布应用](https://developer-docs.amazon/sp-api/docs/list-your-app-on-the-selling-partner-appstore)、[授权上限](https://developer-docs.amazon/sp-api/docs/application-authorization-limits)、[卖家站点权限](https://developer-docs.amazon/sp-api/docs/sellers-api)、[各站点标识](https://developer-docs.amazon/sp-api/docs/store-identifiers)、[Catalog Items](https://developer-docs.amazon/sp-api/docs/catalog-items-api)、[Product Type Definitions](https://developer-docs.amazon/sp-api/docs/product-type-definitions-api)、[创建刊登](https://developer-docs.amazon/sp-api/docs/create-a-listing)、[Listing 状态与校验 FAQ](https://developer-docs.amazon/sp-api/docs/listings-apis-faq)。

真实试用所需的开发者档案、应用状态、角色和网站归属，从企业名下的 [Solution Provider Portal](https://solutionproviderportal.amazon.com/) 取得非保密证据。2026-11-24 已访问门户，页面要求登录，企业应用及角色状态仍未核实。公开网站按 Amazon 要求可在未登录时访问，并提供产品用途、联系、隐私和价格信息。真实美国站样本需由卖家本人确认。仓库中的公开页面采集、ZenRows 和浏览器插件证明有来源侧能力，不能证明本应用有卖家授权或 Amazon 角色。

## Owner 与装配

| 合同与事实 owner | 实现、注入与消费者 |
| --- | --- |
| 当前身份和企业权限 | 当前 verified Effective Organization、actor、member、店铺 grant 和适用刊登权限。店铺授权开始/完成沿现有管理授权，刊登候选与结果沿现有 Listing 和 Store scopes。没有 platform-admin 跨企业例外；BFF 仅转发同源动作或窄 OAuth 回调，服务端判权。 |
| Store Center | `internal/storecenter` 保留当前 Store、应用、连接、审计、加密凭据的归属。参考 TikTok/Walmart 当前独立官方连接，新增 Amazon 的窄授权与只读/刊登用途能力。Store 的 app 构造器显式注入，要求已配置和运行就绪才开放。 |
| 源商品和素材 | Catalog/Collection/Asset 提供当前企业已保存快照/准备记录和批准图片。沿已有读取端口，刊登准备只引用准确版本。首版使用现有 `app/listing/image_probe.go` 受控下载检查与已批准的公开 HTTPS 素材。确认 Amazon 能抓取且在异步处理期可用，缺合法素材或有效链接则阻止。不能为租户私有素材打开公共权限。见 [Amazon 图片规范](https://developer-docs.amazon/sp-api/docs/submit-media)。平台和浏览器提供的 URL 不作为任意服务器读取地址。 |
| Listing | 现有 Target 版本、Preparation/Operation 和 Submission kernel 持有唯一准备、确认、fence/attempt 与结果。采用 Amazon 专用 Target 字段和原领域归属，复用版本、原键、受控发送和工作流。当前 `app/listing/upload.go` 有按平台分派，SHEIN 默认路径有专属图片、发布和结果假设；添加明确 Amazon 分派及 capability，不能把 Amazon 送入默认分支或照搬其它平台结果状态。 |
| Marketplace 与发布 | Amazon adapter 提供固定端点的 LWA、站点校验、Amazon 商品规则、校验预览、一次正式提交和只读查证。客户授权和凭据库留 Store，提交结论留 Listing。前端沿 `/workbench/stores` 和 `/workbench/listing`，消费当前企业服务端的可用能力与结果。 |

Legacy decision 为 **RETIRE**。沿 [#822](https://github.com/henryxu1008/shuomi-ai-commerce/pull/822) 保留退休结果，不引入旧刊登或数据迁移。目录查重只消费 SP-API 的官方商品标识。

### 卖家授权与撤权

- 区分可信应用配置、待完成 OAuth、当前连接三类事实。Store 数据库事务持有短时待授权。仅从工作台发起，开始请求绑定已登录 session、org、actor、member、store、应用、站点和环境，将随机流引用放入窄路径 `HttpOnly; Secure; SameSite=Lax` 短时 cookie，库内仅存摘要，一个会话同时有一个待完成的 Amazon 流程。开始向北美 Seller Central 发送应用 ID 及随机 state。Amazon 的第一跳 Login URI 未承诺回传初始 state，依据 cookie 和同一仍有效的工作台会话领取原流程。此跳与最终回调使用不同的已注册 HTTPS URI，设置 `Cache-Control: no-store` 与 `Referrer-Policy: no-referrer`。公开官方文档示例为 `https://amazon.com/apps/authorize/confirm/{appId}`，首版接受与审查配置一致的固定官方美国域名、HTTPS 默认端口及此 exact 路径，从固定目的地重新组装 URI；拒绝未知参数、歧义编码、重复参数及非预期环境。将 Amazon 原 `amazon_state` 原值传回并发送新的一次性 state，固定 `redirect_uri`。最后匹配新的 state、同一会话/Store scope、预期环境及第一跳 seller ID，再领取一次性的换码许可。错企业、失效、取消和重复回跳不会发起新的换码。预览应用的 `version=beta` 来自可信配置，使用独立应用配置和环境，生产回调拒绝 beta。
- 向固定 LWA HTTPS 地址换码，五分钟内交换且不重试已领取而结果不明的 code。用该 grant 的 short-lived access token 服务端校验 `getMarketplaceParticipations` 和 `searchListingsItems`：后者只请求美国站首条摘要，要求有效 200，空店可连接，不保存商品。由于参与资格接口不返回 seller ID，必须用 seller 路径的授权探测证明 token 与它一致，403/404/节流及无法验证时不生成绑定。该探测的误配拒绝必须先以获准的真实双店授权读验证；隔离仅验证服务消费规则。凭据使用现有用途绑定加密封存，日志、错误、前端和代理访问记录须过滤 OAuth 参数和 token。Store 同一事务检查原尝试、连接代次和 seller/marketplace 唯一归属后保存 token、连接及成功回执。相同已完成回调只返回经当前权限判定的结果。重连沿同一企业、同一个已绑定的 Store 和 seller/marketplace，递增代次；换应用不改变 seller/marketplace 的唯一归属，跨企业或跨 Store 转移不在首版。去重标识清除以后，生产连接仍只能从能核实原归属的既有 Store 重新授权，首版禁止自动把该 seller 视为新连接；具体约束见下文。
- 执行、预览、查证均重查企业、成员、店铺、应用、绑定代次和所需角色，使用同一已核验连接。业务写入前现取短时 LWA 权限，避免跨工作流复用旧授权。已知本地撤权立即停止获取和展示 Amazon 数据，Amazon 拒绝授权失败时阻止后续上传并呈现相应的访问提醒，通用 401/403、LWA 错误和缺角色均不足以证明卖家主动撤权或触发清除；Amazon 已接收的送达请求无法本地撤回。定期续期按官方 365 天规则提示与重授权，原授权仍可用时允许在其范围内完成查证。已取得但随后撤权的结果不能写回当前可见连接。
- Store 负责撤权触发、阻止访问和调度有界清除；Listing 在自己的库执行对应清除。即时禁止前端和工作流读回平台资料，按适用 Amazon 条款期限清除平台凭据、ASIN、平台问题、平台回执及可识别的 seller 信息；独立的原用户商品、原内部操作 ID 和不含 Amazon 数据的防重占用留各自 owner。平台字段放可清除分区，避免写入不可变 Target、Temporal history、日志和通用 receipt；备份恢复必须重新应用撤权/清除边界，先于任何业务读取。是否可保留 seller/marketplace 去重标识，以获准可保留的非敏感证据为准，不以哈希自动划为可保留。若适用条款禁止保留该身份且无法证明安全的原归属，暂停该连接的生产重新启用，保留本地原 Store 和 UNKNOWN 约束；生产装配在保留与安全重绑定策略经证据确认以前默认关闭，生产放行新店前验证清除约束无法被新建 Store、切应用或切企业绕过。隔离实现使用合成身份、样本与回执。见 [重授权](https://developer-docs.amazon/sp-api/docs/reauthorize-applications)、[撤权](https://developer-docs.amazon/sp-api/docs/revoke-authorizations) 和 [数据保留及恢复](https://developer-docs.amazon/sp-api/docs/protecting-amazon-api-applications-data-encryption-and-recovery)。

### 商品规则、确认与副作用

- Target 记录绑定准确的源商品版本、批准素材、本地 StoreID 和不含 Amazon 身份的内部刊登引用、美国站常量、稳定 SKU、商品类型内部引用、规则指纹、有效期和确认载荷哈希。Amazon seller 原值、平台定义和待判定可否保留的派生值留在可清除分区。服务端使用现有确定性校验能力或成熟 JSON Schema 校验器，按 Amazon 元 schema 支持首类必要关键字；外部 schema 绝对地址仅由显式 Amazon 签发受控规则文档提供，限制允许的目标、大小和超时，禁用任意外部引用。规则服务的缺项、Amazon 业务校验和没有把握的商品事实交由运营人明确修正或确认。
- `putListingsItem` 会完全替换同 SKU 的既有资料，官方合同没有条件式仅创建保证。为首版使用应用专属、由服务端分配的稳定 SKU，以 canonical Product 和原本地刊登关联分配，按固定站点和同一已核验卖家使用，跨成员、修订和应用重连稳定；现有 Target 的 actor 身份只用于访问，不用来派生第二个 Amazon SKU。SKU 对用户只读，无自定义写入。在同一 Product DB claim 事务用生产环境 + market + 已证实可持久化的 seller 标识 + SKU 作全局占用，同时以原 Store + SKU 作现有 Submission fence，在正式发送前用 searchListingsItems 按确切卖家和 SKU 查询，只有验证完整的 200 零结果才允许进入发送。发现已有 SKU 或无法确认便回到原提交核对；普通 GET 404 只用于展示暂未找到。真实放行前由店铺确认该专属编号空间无外部写入。若无法建立唯一 SKU 写入约定，停止这一店的首版正式提交。 Store 解除授权不得删除本地原 Store ID、重置该刊登 fence 或把该 seller 绑定到新 Store；当前删除入口在有此业务事实时阻止实体删除。应用更换不改变已有刊登身份。
- `OfficialIntent` / `OfficialReceipt` 已有按平台的类型，Amazon 沿 [Walmart effect](../../../../internal/listing/submission/walmart_effects.go) 和 [Walmart SKU 占用](../../../../internal/integration/persistence/listing/submission/walmart_effects_policy.go) 的有界模式扩展 Listing，复用现有 official repository 和 kernel。现行 `CompleteOfficial` 的成功语义不可直接处理拒绝。准备时以 Product DB 同一事务建立 Amazon intent 和现有 kernel claim/fence，scope 为当前 org、store、稳定 SKU，intent 绑定准确 Target revision、canonical Product、内部连接代次、美国站常量、确认摘要及固定用途。kernel 完成的成功只证明已受理，失败只证明本次在确定语义下未受理。接到有效 Amazon 响应后，在 Product DB 的同一事务校验原 claim token/epoch、写入 Amazon 专用回执并以现有 kernel CAS 终结原 attempt。提交事务不确定时查原键，原状态未取得确定收口前依旧占用。Amazon 专用回执以原 attempt/intent 唯一关联，存受理/拒绝对应的最少可披露平台字段；通用成功引用只指内部回执 ID。原持久化 operation 与 UI 投影只拿同一结果，缺回执不把 ASIN 或观测提升为原请求成功。后续平台观测由同一 Listing owner 按原 attempt + SKU 读取，和原回执分别保存；只读观测不能解除 UNKNOWN fence。
- 最终发送前复核授权、同版本 Target/商品/素材、商品类型规则和官方预览。用现有 Submission 执行令牌在唯一受控 transport 发送一次。原键同载荷返回原状态，异载荷拒绝，持久化 fence 保证并发与 re-login/重启不会重新获得发送权。已读无 SKU 只是发送前校验，不能清除发送后的 UNKNOWN。HTTP 客户端和工作流均不得自动重放此 mutation。只有能证明进入发送之前的终止，或 Amazon 返回明确、无异步副作用的未受理，才允许用户修正后重新确认；不新增自动重试。未证明的各种响应中断、无回执、回执写入不明或进程丢失都保持原键占用。
- `ACCEPTED` 表示 Amazon 已受理，记录 submissionId。明确 `INVALID` 表示本次未受理，按分类返回安全的字段/下一步。后续 `getListingsItem` 观察独立记录原 SKU、同站点、可用 ASIN、状态和时间。`BUYABLE` 才支持显示「可购买」，可发现是独立属性。已受理后的明确阻断问题展示 Amazon 返回的具体原因；缺少回读或无当前记录显示「处理中」或「暂未查到」。若响应丢失，可凭同一专用 SKU 回读并展示可核实的刊登事实；缺该次回执仍不把观察记为原提交成功。没有可核实结果时保留「提交结果待确认」，允许有界只读核对并引导客户查看 Seller Central。用户无法从前端手填成功或换键绕过原占用。
- 只允许受控端点、方法、站点和规范化参数；用现有服务的 deadline、请求/响应、并发与频次限制。读取允许按原有限策略重试，平台节流、错误响应或权限漂移绝不制造可上传/已上架的结果。客户端断开只停止等待，服务端使用原生命周期和收口边界保存真实结果。

## 验证与试用门槛

1. 隔离验证走现有正常入口，复用 Store/Listing 测试夹具，必要时以合成 OAuth/SP-API 服务和 Amazon 静态模型核对协议。两个企业分别连接与刊登，覆盖跨企业及未获店铺授权、撤权、重连、回调重放、US 资格、规则变更或素材失效，以及同一商品跨成员并发提交。核对刷新/重启后对同一提交和 SKU 的查证，已有 SKU、确认取消、已受理后读回延迟、未受理、发送后断线和迟到结果；检查一个确认仅有一次出站写及 UI 的准确状态。 对 callback 第一步不回我方初始 state、cookie/会话丢失、第一/最终 seller 不同、正确授权码却与 seller 不符及 shop 空列表分别建证据。
2. 形成可评审实现后记录准确候选、必要测试和独立检查、正常入口与目前可用范围。仓库层级为 Independent Architecture，生产代码开工前须完成本稿评审并在 Issue 标为 `IMPLEMENTATION_READY`。
3. 真实试用提前确认公开应用状态和产品角色、官网登录/回调与隐私、可用美国站卖家、合格 GTIN/类型及图片、独占 SKU、允许创建真实商品并可能开放购买的明确授权。先由卖家本人授权，核实本人站点，运行有限校验预览，确认后只提交一件，再读回同一 SKU 和 Amazon 状态。跨企业的真实试用须分别有已授权的第二企业和卖家，否则维持隔离验证证据。平台审核提交、实例部署和真正写入各有明确授权。真实平台、客户试用和商品上架为 **NOT_RUN**。
