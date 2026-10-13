# TikTok Shop 美国跨境：官方合同证据

Refs [#669](https://github.com/qq550723504/task-processor/issues/669)。

核对日期：2026-10-13，Asia/Singapore。来源是官方 Partner Center 在 Edge 中实际加载的文档正文及展开的 API 字段；没有登录开发者后台、取得凭据或调用商家 API。文档证据不是账号准入、实际应用 scopes 或真实店铺验收。

## 1. 授权与应用

- [Developer onboarding](https://partner.tiktokshop.com/docv2/page/developer-onboarding)：中国注册企业有跨境 ISV 路径；美国使用单独 Partner Center。入驻、应用审核和具体接口准入仍分开。
- [Create your App](https://partner.us.tiktokshop.com/docv2/page/create-your-app)：应用类型、市场、seller type、scopes、redirect URL、webhooks 和凭据由具体应用配置决定。用户确认尚未创建应用。
- [Authorization overview](https://partner.tiktokshop.com/docv2/page/authorization-overview-202407)：美国卖家入口为 `https://services.us.tiktokshop.com/open/authorize?service_id={service_id}`；`service_id` 与 `app_key` 是不同标识。回调给出 `code` 与 `state`，拒绝授权时给出 `error=auth_denied`。
- code 在 30 分钟内一次性使用。令牌交换是 GET `https://auth.tiktok-shops.com/api/v2/token/get`，参数 `app_key`、`app_secret`、`auth_code`、`grant_type=authorized_code`；不能改成标准 OAuth 拼写 `authorization_code`。
- 刷新是 GET `https://auth.tiktok-shops.com/api/v2/token/refresh`，参数为应用标识/secret、`refresh_token` 和 `grant_type=refresh_token`。这些 GET 会改变凭据事实，不能按只读 GET 自动重试或记录完整 URL。
- `access_token_expire_in`、`refresh_token_expire_in` 是绝对 Unix 到期时间；access token 默认约 7 日，不是将返回数值再加到当前时间。返回还包括 `open_id`、`seller_base_region`、`user_type`、`granted_scopes`。
- 当前 overview 列出卖家类型 0，以及 Global Selling 类型 4/5。商品 API header 的说明仍写 `user_type=0`。不能仅凭此笼统描述把 4/5 当作 creator 或 local seller；实际应用类型、seller guide、授权结果与店铺规则须在生产接入前一致核实。

## 2. 真实店铺身份

[Get Authorized Shops 202309](https://partner.tiktokshop.com/docv2/page/get-authorized-shops-202309)：GET `/authorization/202309/shops`。本批申请最小相关 scope `seller.authorization.info`，不因文档列出替代 FS scope 而申请履约能力。

展开字段包含 `shops[].id/name/region/seller_type/cipher/code`：

- `id` 是内部 Shop ID；`code` 是 Seller Center 展示的店铺编号，不能互换。
- `region` 必须确认为 US；`seller_type` 本批必须为 CROSS_BORDER，LOCAL 不在当前范围。
- `cipher` 是 API 的店铺识别 token，不需要解密，也不能由 shop ID 拼造。
- 同一卖家的授权列表可能含多个国家店铺；不能选择第一项、授权整个列表或用 `seller_base_region` 代替目标销售市场。

## 3. 跨境刊登分流

[Get Global Listing Rules 202507](https://partner.tiktokshop.com/docv2/page/get-global-listing-rules-202507)：GET `/product/202507/global_listing_rules`，scope `seller.product.basic`。展开的返回字段 `listing_methods` 明确区分：

| 值 | 官方流程 |
| --- | --- |
| GLOBAL_PUBLISHING | 创建全球商品，再发布至目标本地市场 |
| LOCAL_REPLICATION | 创建本地商品，再复制至其他目标市场 |

本批只发布 US，不调用复制到其他国家。接口还提供仓库及库存分配规则；不可把样例仓库、金额或共享库存模式当成真实店铺事实。

[Create Product 202309](https://partner.tiktokshop.com/docv2/page/create-product-202309)：POST `/product/202309/products`，scope `seller.product.write`。正文明确：已切换到 LOCAL_REPLICATION 的 global seller 可用；未切换者继续使用全球商品接口。US category_version 必须为 v2；`shop_cipher` 为必需店铺输入。`save_mode` 支持 AS_DRAFT 与 LISTING，缺省为 LISTING，因此不能用省略字段表达保存草稿。创建后提交平台审核，不等于可售。

本地创建支持 `idempotency_key`，长度最多 128；文档建议 UUIDv4。已查正文没有定义保留窗口或任意重发的保证，本批仍由现有 Submission Kernel 控制重发，不能以有此字段为由无限重试。

[Create Global Product 202309](https://partner.tiktokshop.com/docv2/page/create-global-product-202309)：POST `/product/202309/global_products`，scope `seller.global_product.write`。先创建全球商品，不自动在 US 上架；其 Body 支持 `external_global_product_id`，不是已经核实的 idempotency key。已读字段没有本地创建的 `save_mode` 或 `idempotency_key`，不得自造。

[Publish Global Product 202309](https://partner.tiktokshop.com/docv2/page/publish-global-product-202309)：POST `/product/202309/global_products/{global_product_id}/publish`，同一 Global Product Modify scope；`publish_target` 指定目标市场。文档说明每个市场只发布一次，但这不能代替请求结果关联或被扩大成可靠的任意重试承诺。本批必须只指定原已授权的 US shop。

## 4. 商品规则与结果

- [Products API overview](https://partner.tiktokshop.com/docv2/page/products-api-overview)：Draft、Pending、Failed、Activate 等平台状态不同；创建/发布成功回执不等于审核通过或可售。修改审核中的版本与在线版本也不同。
- 创建接口要求来自官方的可用叶子类目、类目属性、SKU、商品图片 URI 和相应证明。US 标题 1–255 字符，description 上限 10000 字符，main images 上限 9；这些通用限制不能代替类目实时必填规则。
- 规则入口：Get Categories / Get Global Categories、Get Category Rules / Get Global Category Rules、Get Category Attributes / Get Global Attributes；按官方 listing_method 选择同一套规则，不能混用 local category ID 与 global payload。
- 商品图片和证明文件必须使用 Upload Product Image / Upload Product File 返回的对应 URI/ID，不直接把来源图片 URL 当作 TikTok 已接受资产。
- 确切 SKU/价格/仓库/证明/图片字段、Publish Global Product 的嵌套关联返回、Global Product 查询关联和未知结果 readback 尚需逐字段收敛。未完成前该外部路径保持未准入，不用示例或假数据补合同。

## 5. 签名与 SDK

[Sign your API request](https://partner.tiktokshop.com/docv2/page/sign-your-api-request)：使用官方 HMAC-SHA256 协议，按键排序查询参数，排除 sign/access_token；非 multipart 请求纳入实际发送的 body bytes，包裹 app secret 后签名。202309 及以后用 `x-tts-access-token` header。不得在签名后再次序列化正文，也不得将 bearer token 当 query 参数。

[SDK overview](https://partner.tiktokshop.com/docv2/page/tts-api-sdk-overview)说明 SDK 下载需要开发者账号拥有/管理有效应用，接口集合取决于实际 scopes。优先复用官方 Go SDK 和已有 HTTP/加密/persistence 设施；没有应用时不把第三方未审 SDK或手写完整 SDK当作当前已合格官方实现。

## 6. 证据缺口

| 缺口 | 影响 | 解除条件 |
| --- | --- | --- |
| 实际应用、service_id/app_key、审核与 scopes | 真实授权/API验证 | 用户创建并获批匹配 US 跨境的应用，私密配置；本 Issue 不创建账号/接受协议 |
| Global Selling 4/5 与 resource header 0 的具体适用组合 | 实际凭据准入 | seller guide/应用类型与实际合法测试授权结果一致；不以泛型枚举推定 |
| 发布嵌套字段、SKU关联、仓库/金额/证明规则 | 该生产路径的编码准入 | 官方对应字段和有限规则读证据，设计与测试绑定准确合同 |
| 创建全球商品结果丢失的 qualified readback | 自动恢复/再发送 | 官方可用的确切关联；没有证据时持久 UNKNOWN，禁止盲重发 |
| 官方 Go SDK 的实际 API集合 | transport 具体实现 | 实际应用可下载 SDK；或经独立准入确认最小官方协议 adapter 的复用方式 |

以上保持 UNKNOWN/NOT_RUN，未签发产品验收或 IMPLEMENTATION_READY。
