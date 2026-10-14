# TikTok Shop 美国跨境店铺接入

本候选交付一条路径：店铺授权 → 保存美国跨境商品资料 → 显式确认平台草稿或刊登 → 查看实际创建回执及手动查询平台当前状态。设计依据为 [#669](https://github.com/qq550723504/task-processor/issues/669)、[冻结架构](../architecture/tiktok-shop-us-v1.md)及[官方接口证据](../architecture/tiktok-shop-us-contract-evidence.md)。开发者应用尚未创建；真实店铺、权限审批和用户试用保持 **NOT_RUN**。代码与 CI 不代表商品已审核通过或可生产上线。

## 当前入口

1. 正常登录，选择企业，在 `/workbench/stores` 创建 **TikTok Shop / US** 店铺。外部店铺标识填写 Seller Center 的 **Shop Code**，不是内部 shop ID。
2. 在该店铺的“官方 API 授权”选择已配置的 TikTok 应用，开始连接；在 TikTok 官方页面由实际店铺主体授权。回调路径为 `/workbench/stores/tiktokshop/callback`。应用未配置时显示不可用，不生成授权成功状态。
3. 商品先沿“我的数据 → 转入供应链批次”保存。尚未绑定店铺时，可以在“待补全”保存独立候选标题和描述；参见[原 CompletionDraft 入口](supply-unbound-completion-drafts.md#入口和使用)。
4. 打开 `/workbench/supply/mine`，选择批次和 TikTok 店铺，逐件点击“查看 / 补全”。选择官方允许的发布方式及 AVAILABLE 末级类目，填写英文资料、商家 SKU、价格、仓库和数量。可显式带入待补全草稿 Rn；所选 record/revision 随 Target 保存，手改文字后按手填内容保存。
5. 确认来源图片，选择批准素材，补齐当前类目属性、证明与尺码表。点击“保存并校验 TikTok 资料”。未满足规则的记录仍是本地待补全资料，不能刊登。
6. 在列表单独确认上传。LOCAL_REPLICATION 可明确选择 **AS_DRAFT** 或 **LISTING**；GLOBAL_PUBLISHING 保存全球商品后只刊登 **US**。美国全球刊登每个 SKU 明确选择一个实际发货仓；美国数量不得超过批准的全球库存预算。美国最终售价 sale_price 与全球税前基准 amount 分别填写，币种均为 USD。
7. 查看商品编号、SKU 关联及原操作结果。GLOBAL 的 SUCCESS 表示提交审核；DRAFT 保留实际美国草稿及关联；FAILED 保留全球商品和原失败结果。本地 LISTING 创建响应仅确认资产已创建，审核及在售状态仍待平台确认。点击“查询平台当前状态”，按原创建的商品 ID 在当前授权 Shop 查询最新商品和审核状态，显示查询时间及可公开的刊登前置原因。该查询不会改变原回执或解除 UNKNOWN 占用。

首版读取当前 Product/Asset owner 已批准的 PNG/JPEG，受既有图片解析限制及 TikTok 图片尺寸要求约束。仅提供图片证明；要求未支持文件格式的类目不属于已验证可刊登范围。批量自动适配、自动审核、自动重试未知结果、店铺商品同步、其他市场、美国本土店铺、订单、退款、财务和库存自动同步尚未开放。

## 应用与私有配置

先在 [TikTok Shop Partner Center](https://partner.tiktokshop.com/)完成实际企业和开发者应用审核，申请适用的美国跨境店铺能力。取得 app_key、app_secret 和美国授权入口的 service_id；**service_id 不等于 app_key**。在官方应用登记实际 HTTPS 回调 URL，并按官方当前配置登记生命周期事件接收地址 `/api/v1/platform/tiktokshop/lifecycle`。该事件入口只接受本应用签名的请求。

最小规则读取涉及 `seller.authorization.info`、`seller.product.basic`、`seller.logistics`。全球分类/属性规则另需 `seller.global_product.category.info`，全球仓库需 `seller.shop.info`。LOCAL_REPLICATION 写入需 `seller.product.write`；GLOBAL_PUBLISHING 需 `seller.global_product.write`。以应用实际获批范围和令牌实际 scope 为准；缺权限时拒绝调用，不以勾选 UI 代替官方授权。Seller/Global Selling token 的角色与资源接口组合须在 Development Shop 或获准店铺验证。

在 Current Application 的私有 manifest 中配置独立条目，保留其他既有字段：

```json
{
  "storeCenter": {
    "tiktokShop": {
      "appKey": "actual-approved-app-key",
      "serviceId": "actual-us-authorization-service-id",
      "revision": "tiktok-configuration-1",
      "callbackURL": "https://your-console.example/workbench/stores/tiktokshop/callback",
      "appSecretFile": "C:\\private\\tiktok-app-secret",
      "credentialKeyFile": "C:\\private\\tiktok-credential-key",
      "credentialKeyId": "tiktok-key-1"
    }
  }
}
```

两个文件必须为不同的绝对路径私有普通文件。应用密钥文件存实际 app_secret；凭据保护文件存 Base64 编码的独立 32 字节 AES-GCM 密钥。不要把密钥、授权码或 token 写入聊天、Git、URL 或前端存储。`tiktokShop` 省略时不读取这些私有文件；原 SHEIN/TEMU 配置继续沿原 registry 装配。该示例不是可直接使用的凭据或完整 manifest。

## 全新安装与启动

沿[当前供应链的正常初始化和启动](my-supply-chain-shein-v1.md#初始化与正常启动)准备已获准、明确命名的**全新空** Product、Asset、Store 数据库及各自 runtime role。Product owner manifest 启用 `collections:true,supplyChain:true`；需要未绑定草稿时另启用 `supplyCompletionDrafts:true`。TikTok 使用原 Listing Target/Submission 表，不复制事实 owner；当前 schema 包含 tiktokshop-us 和 global_create。

Store 首次空库安装使用已有初始化器增加 TikTok OAuth 最小 grants：

```powershell
go run ./cmd/store-center-schema-init -config C:\private\store-owner.json -tiktok-shop
go run ./cmd/current-application -config C:\private\supply-current-application.json
```

在前端正常 Auth.js/ZITADEL 配置中启用 `LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED=true`、`LISTINGKIT_SUPPLY_CHAIN_ENABLED=true`，将 `LISTINGKIT_SERVICE_API_BASE` 指向该实际后端的 `/api/v1`。如启用待补全草稿，另配置 `LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED=true`。沿原 UI 启动命令执行 `pnpm dev`；TikTok 无单独的伪造身份入口或独立平台 Workbench。

服务只验证 schema/grants，不执行 DDL。本批次没有授权迁移、修改或重置保留实例。上述命令不能用于旧非空库；不执行数据库覆盖、volume 删除、`down -v` 或兼容迁移。实际应用注册、权限审批、实例准备/部署和商品发布分别需要对应的执行授权。

## 保存与恢复

候选文字保存到原 CompletionDraft owner；正式平台资料保存到原不可变 Target record。当前 Seller grant、加密 token generation 和 Shop binding 由 Store owner 保存；Submission 以原 Product DB 事务保存每个 intent、attempt 和关联回执。两个本地 Store 绑定同一卖家时仍共用 canonical grant，断开一个 Store 不撤销其他 Store 的授权。

重复点击或使用新操作键不能重新创建已成功步骤。成功回执可以在确切相容的批准资料和实际店铺身份下复用；等价批准图片的相同字节和用途复用原上传回执并保留最初来源审计。响应丢失、提交结果不明或 worker 重启保留原 UNKNOWN 占用；没有“填商品编号即成功”或解除占用后盲重发。当前页面展示原回执及待核实状态，远端修改/重新刊登和完整未知结果核验尚未开放。

交付候选的操作结果须由用户或指定独立验证者通过真实入口确认；开发测试、PR CI 和独立代码检查分别记录，不能写成用户验收或真实平台 PASS。
