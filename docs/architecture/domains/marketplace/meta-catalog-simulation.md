# Meta 商品目录隔离模拟

执行依据为 [#1617](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1617) 及 [Google 与 Meta 商品上传设计](google-meta-product-upload.md)。本切片 Design Basis 为 Independent Architecture，已通过架构评审并达到 IMPLEMENTATION_READY（仅隔离模拟）。准入仅覆盖隔离模拟，真实平台接入保留 NOT_READY。

## 用户结果和范围

客户在硕米工作台自行完成模拟 Meta 商业授权，选本企业已有目录。运营人员从已有自有商品进入上传页面，核对原商品、目标目录、规格、已审核主图、售价和购买链接，确认后新增一个商品并找回记录。对连接了这个目录的 WhatsApp Business 账号，可以查询绑定验证结果。Facebook Shops 的商品写入结果可查，实际买家可见性留待真实商店验证。两处渠道共享一个目录、一个稳定商品编号和一次新增。

本阶段采用官方形状的模拟接口，保留普通运行时的默认关闭。订单、聊天消息、广告、批量同步、建立或改绑 Meta 目录、变更 Facebook 商店渠道、真实 OAuth 和部署不在本切片范围。Google 已落地路径不变。无旧系统迁移或兼容入口。

## 官方依据与平台差异

- 仅使用商家企业已拥有，且已验证授权用户具有 MANAGE 权限的目录。候选的真实授权方式是 Facebook Login for Business，需通过应用审核和试用账号实测所选配置可访问现有目录。企业资产读取使用 Business 的 `owned_product_catalogs` 和目录管理权限证据。本切片模拟独立商业授权，不模拟要求收到安装回调后推送全量库存的 Facebook Business Extension。[Business Manager Catalog](https://developers.facebook.com/docs/business-management-apis/business-asset-management/guides/catalog) · [FBE 的全量同步条件](https://developers.facebook.com/docs/facebook-business-extension/fbe/guides/catalog)
- 单品写入形状为 `POST /{catalog_id}/products`，明确发送 `allow_upsert=false`。稳定 `retailer_id` 在指定目录内唯一。只读取确切身份，禁止以未找到为理由覆盖或把另一写入方留下的同号商品当作原提交成功。以保存的 Meta 商品 ID 查询同商品；已创建、查询失败和返回不明各自呈现。售价以 ISO 币种的最小单位编码，本次界面支持 USD，明确展示并校验。[Meta 单品接口](https://developers.facebook.com/docs/marketing-api/reference/product-item/)
- 仅使用 `GET /{waba_id}/product_catalogs` 核对同一目录。官方列有额外 WhatsApp 权限，目录写入的授权本身不能替代。核对账号的受权归属，并将未分配授权、无匹配目录和已匹配目录分别显示。缺少 Facebook Shops 买家端视图的直接证据时显示「商店展示待核验」。目录商品查询不触发消息或绑定修改。[WhatsApp 目录查询](https://developers.facebook.com/docs/graph-api/reference/whats-app-business-account/product_catalogs/) · [Facebook Shops 与目录](https://developers.facebook.com/docs/facebook-business-extension/fbe/guides/catalog)

## 责任与输入输出

| 责任 | 现有 owner 和复用路径 |
| --- | --- |
| 有效企业、成员、角色与请求同源 | 现有工作台、Go 路由认证和 Store / Listing 活体权限；模拟进程注入合成会员身份，工作台使用已有 BFF |
| 外部目标与客户授权 | Store Center 孤立目录连接；每条连接属于授权 Store，沿 Google 的回调与连接事务边界，使用当前 AES-GCM 密钥保护 |
| 商品、变体和已审核图 | Product Catalog 与 Asset 已有只读接口，首版只选自有商品的一项规格 |
| 提交内容和结果 | Listing Submission 的确认快照、稳定请求身份和现有持久化执行内核；复用授权快照、发送闸门和回执证据 |
| 平台请求与证据 | 独立 Meta 协议客户端；只有测试注入本地模拟上游，应用和浏览器不能指定真实回调或把客户端凭据暴露给页面 |

工作台进店铺中心连接，用户自行点击授权并返回企业页面。随后选择已验证目录；需要核对 WhatsApp 渠道时选择已获授权的现有 WhatsApp Business 账号。商品页面从原来源保留最近的预览与提交记录。授权能力缺失或上传未确认时，页面呈现明确的下一步。继续用 [页面职责与线性操作流程](../../../product/final-ui-ia-authority.md) 和 [主架构](../../project-boundaries.md)。

## 不变量与验证

- 授权 state 随机、一次性、短期、绑定原组织、成员、应用和店铺版本。先持久领取兑换权，再调用外部模拟。拒绝他企业读取、抢绑定和复用回调；失败后开启新授权。授权目标唯一，由 Store 持久保存。确认前通过服务端模拟证据核对目录管理权，浏览器不能借目录号授权写入。重连限原目录，代次前进；解绑销毁本地凭据并保留目标身份。回调、接口响应、日志及商品计划不得携带客户令牌。
- 预览固定源版本、源内容、审核图版、目录、授权代次和确切字段。发送前复查同一企业的实时权限、当前绑定和原资料。Listing 内核固定同一确认、同一目标及变体。重复提交只复用，已成功的相同目录商品禁止再次新增。上游明确拒绝可显示失败，能够证明未发送则可重新确认；真正响应丢失保留 UNKNOWN 及目标互斥。读取目录中的已有同号商品只提供观察，不凭此解除 UNKNOWN。
- Store 授权、Product 商品计划在各自数据库内负责事务。写入与读回之间不引入新 scheduler。默认运行配置不注册 Meta API 或真实凭据。用户入口和 API 只在隔离模拟明确注入时开启，模拟账号不复用客户账号，也不会在普通环境触发任何 Graph API 请求。模拟 OAuth、目录和渠道读取全部限定 loopback，拒绝回跳到未核对地址。
- 隔离验收用可重复的全链路回归。浏览器从连接入口走模拟 consent、选目录、预览、确认、刷新查结果，并核对 WhatsApp 匹配与 Facebook 待验。至少验证 0 次预览写入、同意图只写 1 次、同号冲突拒绝或超时原结果保留、不同租户无法使用原授权。真实授权、展示资格及部署明确留待单独阶段。


## 复现隔离模拟

使用 Docker、项目锁定的 Go 和前端依赖以及 Playwright Chromium。以下 Go 集成测试自行启动临时 PostgreSQL、预置企业下已启用的模拟店铺和自有商品，并在结束时清理临时数据库。客户的授权操作从该店铺的「授权信息」开始，商品刊登入口从自有商品进入。这个阶段使用预置店铺，店铺创建入口仍沿用当前开放的平台范围。

```powershell
pnpm -C web/listingkit-ui install --frozen-lockfile
pnpm -C web/listingkit-ui exec playwright install chromium
$env:META_WORKBENCH_BROWSER = '1'
go test ./internal/app/listing -run '^TestMetaMockAuthorizationPublicationAndSavedResult$' -count=1 -v
Remove-Item Env:META_WORKBENCH_BROWSER
```

去掉浏览器环境开关可单独运行数据库回归。浏览器测试会在本机 3210 启动工作台，使用合成身份，由本机模拟 Meta 服务显示同意页。模拟结果仅适用于该隔离验收。
