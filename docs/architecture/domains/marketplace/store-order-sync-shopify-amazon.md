# Shopify 与 Amazon 美国站订单同步与查看首版

当前候选为 Design Basis（Independent Architecture）。用户批准的顺序是先做 Shopify 同步与查看，Amazon 先核对 API、权限并准备隔离样本。两边商品阶段已有合入的工程结果，订单有独立交付范围。正式订单功能在执行 Issue 通过独立架构评审并标为 `IMPLEMENTATION_READY` 后实施。 [排期 #137](https://github.com/henryxu1008/shuomi-ai-commerce/issues/137) · [Shopify 商品 PR #1615](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1615) · [Amazon 商品 PR #1621](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1621)

执行任务分别见 [Shopify #1622](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1622) 与 [Amazon #1623](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1623)。

## 用户结果

企业店主在店铺中心为本企业的现有店铺开启订单读取。订单操作员从「订单管理 → 平台订单记录」选择自己的店铺，手动同步最近 30 天下单的订单，查看订单号、下单时间、平台状态，以及详情中的商品名称、SKU 和下单数量。显示实际同步截止时间。撤销授权、同步不全、失败、确实没有订单各自有业务提示。重复同步同一批记录，列表每店每单只显示一条当前记录。 [现有导航决定](../../../product/final-ui-ia-authority.md)

首版只同步与查看。不开放发货回传、取消、退款、平台订单创建/编辑、自动同步、历史补采、经营报表或履约命令。不读取姓名、邮箱、电话、收货地址、支付数据、订单金额、买家自由填写内容或完整平台响应。开发和隔离验证使用合成数据。

## Owner 与调用顺序

沿用 [店铺观察合同](../../store-center-platform-observations.md)。Store Center 持有企业店铺和当前授权事实。Store Observations 持有同步命令、每店同步、进度、观察记录与最近一次完整结果。平台持有原订单。已保存记录是有时间和范围的只读观察。产品刊登、仓库履约与运营报表不消费此首版结果。

- 在 Store Observations 的订单读取 port 接入平台适配器，经现有应用层、HTTP/Temporal 注入和 `observations.Service` 进入 `/workbench/store-orders/records`。外部请求由 `internal/integration/shopify` 或 `internal/integration/amazon` 发出。现有名称与状态在平台订单视图解释，保留来源状态；遇到新值显示「平台状态暂未识别」，SKU 空值显示「平台未提供」。
- 现有 `Service.List` / `Detail` 和 `ObservationPage` 的订单消费者当前以 SHEIN 投影为主。为新平台添加受支持的只读投影及店铺选择，明确保留 SHEIN 默认行为。新平台使用显式判别的最小订单、订单行投影，校验业务字段，金额、物流和履约操作在该视图省略。既有存储只落允许字段，不写入原始平台响应。稳定身份是企业、StoreID、来源绑定与平台订单 ID，订单行按平台订单行 ID，展示号与 SKU 仅供查看。
- 沿用 `workbench.store.orders.read/sync`。同步还须通过订单读取和当前店铺读取。入口、已保存记录列表/详情、同步结果与 worker 都绑定原企业、当前成员及当前店铺，不接受客户端提供商家身份或平台游标。创建命令及每次平台读取前检查当前授权；页事务及发布最近结果前复核相同的授权代次。已保存记录仅凭当前本地订单授权与相同来源绑定读取；本地撤权或换绑后旧代次立即不可读。平台拒绝表示本次平台读取失败，单个 401/403 不证明远端已卸载。确认撤销后由现有 Store Center 路径记录失效。
- 应用允许订单读取、店主明确同意订单用途、凭据可访问平台订单是不同条件。实现把新增的订单用途证明收敛到 Store Center 的当前平台授权记录及代次。缺少明确证明的既有连接默认为仅商品用途。现有产品授权入口继续可用，不能被订单授权失败锁住。新用途的申请、回调与提交沿用现有店主授权机制，绑定原用途、企业、店铺、成员及应用。令牌仍按现有机制密封或短时使用，观察任务不保存令牌。产品写入凭证或写入许可不能用作订单用途证据。
- 复用现有快照表、命令去重、每店代次和 Temporal。若当前 Store 授权记录需要新增用途字段，沿 Store owner 的规范定义，应用端对未记录订单用途取默认拒绝。Legacy decision 为 N/A，无历史订单回填、旧接口适配或第二套订单存储。

## 平台读取合同

### Shopify

复用当前 GraphQL Admin API `2026-10`。订单启用后以店主明确授予的 `read_orders` 为准，核对店主在线授权及获准权限，再确认同一 App、Shop GID、Installation GID、域和本地连接代次。店铺授权升级为显式操作，沿同一安装的 Shopify 支持流程，保持既有商品权限。工作进程只在读取前换取短时服务端令牌，并在同一安装复核当前 `read_orders` 后使用；scope 缺失时提示店主处理订单授权。首版 30 天在该权限通常开放的 60 天内，申请中无 `read_all_orders` 或订单写入权限。 [权限管理](https://shopify.dev/docs/apps/build/authentication-authorization/manage-access-scopes) · [当前应用安装](https://shopify.dev/docs/api/admin-graphql/2026-10/queries/currentAppInstallation)

`orders` 按 `created_at` 固定起止并按 `CREATED_AT` 查询，按 `pageInfo` 翻页。读取 `id`、`name`、`createdAt`、`updatedAt`、`cancelledAt`、`displayFulfillmentStatus`。取消时间仅用于判断是否取消；已关闭与发货完成各按平台语义。商品行取 `id`、商品标题 `title`、`sku` 和 `quantity`（含后来退款或移除的原下单单位）。每个订单的 `lineItems` 是独立连接，按同一订单 ID 与行游标续读。若在续读途中版本变动，结果不能当作同一版本的完整商品清单。HTTP 200 含 GraphQL 错误、权限裁剪或必要字段缺失时，显示失败或部分结果。 [orders](https://shopify.dev/docs/api/admin-graphql/2026-10/queries/orders) · [Order](https://shopify.dev/docs/api/admin-graphql/2026-10/objects/Order) · [LineItem](https://shopify.dev/docs/api/admin-graphql/2026-10/objects/LineItem)

Shopify 将订单资源视为 Level 1 受保护客户数据。开发店可在对应开发应用配置后用开发店测试订单验证，公开应用需 Shopify 审核与适用数据保护要求。商品首版中买家事件只留处置回执的合同不覆盖保存订单。真实订单能力保持关闭，直到设计和验证如何用签名事件中确切 shop/order ID 履行订单数据请求、订单清除和整店清除，以及如何防止在途同步、重新同步或备份恢复重新暴露已清除订单。Store Center 负责应用、店铺和代次的事件受理与路由，Store Observations 负责本域记录的读取/清除；具体生命周期先完成独立准入。开发店试验仅使用专用隔离环境和明确合成订单，正式配置和客户数据均不接入。 [受保护数据](https://shopify.dev/docs/apps/launch/protected-customer-data) · [隐私合规事件](https://shopify.dev/docs/apps/build/compliance/privacy-law-compliance) · [商品首版设计](shopify-product-draft.md)

### Amazon 美国站

以 Orders `v2026-01-01` 的 GET `searchOrders` 为基线。使用同一确切授权卖家、应用、连接代次，固定北美端点和美国站 `ATVPDKIKX0DER`。首请求给 `createdAfter` 与 `createdBefore`，截止按平台至少两分钟前的限制冻结，结束时间在界面如实显示。后续页用原查询范围与 `nextToken`，不得越界换站、换卖家或沿用另一轮 token。校验返回的 `salesChannel.marketplaceId`；使用 `orderId`、`createdTime`、`lastUpdatedTime`、可用的 `fulfillment.fulfillmentStatus`，以及 `orderItems` 的 `orderItemId`、`product.title`、`product.sellerSku`、`quantityOrdered`。按官方迁移表先用基础响应验证状态，不请求扩展数据集；真实结果缺状态时按「平台未提供」呈现，若实证表明基础响应无法支持已批准的状态目标，核对平台最小数据集后再决定开放。 [searchOrders](https://developer-docs.amazon.com/sp-api/reference/searchorders) · [字段映射](https://developer-docs.amazon.com/sp-api/docs/orders-api-migration-guide) · [Marketplace IDs](https://developer-docs.amazon.com/sp-api/docs/marketplace-ids)

官方将 `Inventory and Order Tracking` 与 `Product Listing` 都列为该操作可用角色。正式执行只接受实际获准并满足此用途的非受限角色、应用启用与客户卖家明确允许订单读取；若现有角色足够，无须另加角色。Amazon 当前暂无公开 SP-API 应用，首版合同和合成响应验证可以先走，真实卖家授权和 API 权限实测留待公开应用、角色和测试卖家满足条件后另行授权。Amazon 公开网页采集不进入此路径。 [Orders API 角色映射](https://developer-docs.amazon.com/sp-api/docs/orders-api) · [Amazon 商品执行 #1614](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1614)

## 进度、恢复与验证

固定本次范围和原命令键。页结果、同一同步的去重键和检查点沿现有同事务提交，worker 重启从已保存检查点只读续页。Shopify 行详情只有完整才保存为本次完整订单；同一订单重复出现时以可比较的平台更新时间保留较新投影，出现无法解释的同版本差异或范围/标识异常时中止完整发布。Amazon 游标最多 24 小时。限流和短暂传输故障按现有有界只读重试策略，超时、无进展游标、分页超过现有预算、响应不完整或凭据不可用时按现有失败/部分规则结束。已终止命令只返回原结果；用户明确重启同步使用新命令键。部分结果与最近一次完整结果各自标明，覆盖不完整的同步绝不提升完整结果。单次平台访问沿现有 worker 活动截止与响应大小边界，平台游标只留服务器检查点。

- Shopify 优先验证已绑定店铺的合成订单从同步按钮到列表与详情的完整路径，含双层分页、取消或发货状态、空 SKU、重复运行没有重复订单、缺权/撤权、GraphQL 200 错误和两企业隔离。
- Amazon 先用官方响应结构和合成 US 卖家/订单验证同一用户路径，含游标、截止时间、跨站拒绝、跨企业隔离、过期或重复游标和状态缺失。
- 两边覆盖一轮部分同步不会覆盖以前完整结果。核对订单详情可在正常入口查看，商品与既有平台行为未改变。隔离工程验证、Shopify 开发店合成订单实测、公开应用审批与真实商家试用分别记证据。
