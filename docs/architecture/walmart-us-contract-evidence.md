# Walmart 美国站官方合同证据与边界

Refs [#733](https://github.com/qq550723504/task-processor/issues/733)；[首期设计](walmart-us-v1.md)。核对日期：2026-10-16，Asia/Singapore。来源为官方公开开发者文档/API reference、实际服务商表单和 Figma 只读 metadata；未使用凭据、登录商家、调用业务 API 或提交申请。协议样例不是实际应用或平台验收证据。

## 1. 已确认的官方依据

| ID | 官方来源 | 本批使用的窄结论 |
| --- | --- | --- |
| E01 | [Solution Provider onboarding](https://developer.walmart.com/us-marketplace/docs/get-started-as-a-solution-provider) | 服务商评估、获邀 Center/沙箱、OAuth、应用演示/审核；US 不再以旧 Delegated Access 接入 |
| E02 | [OAuth 2.0](https://developer.walmart.com/us-marketplace/docs/oauth-20-authorization) | authorize 参数包含 clientId/redirectUri/clientType=seller/nonce/state；回调含 code/clientId/state/sellerId；交换与刷新使用 WM_PARTNER.ID 和应用 Basic 凭据；返回 token，access 短期、refresh 文档为一年；不从样例推定 token 轮换规则 |
| E03 | [Token Detail reference](https://developer.walmart.com/us-marketplace/reference/gettokendetail) / [guide](https://developer.walmart.com/us-marketplace/docs/retrieve-access-token-details) | GET /v3/token/detail，正式 reference 要求 WM_SEC.ACCESS_TOKEN；返回 validity/expiry/实际 scopes，不是 seller identity 接口；guide 样例缺 token header，不能照抄 |
| E04 | [Scopes](https://developer.walmart.com/us-marketplace/docs/api-scope-walmart-marketplace) | Items 包含 item setup/feed；Account Profile 覆盖 partnerprofile；最小范围申请，不能以全选代替用途判定 |
| E05 | [Partner configurations guide](https://developer.walmart.com/global-marketplace/docs/get-partner-configurations-1) / [reference](https://developer.walmart.com/global-marketplace/reference/getpartnerconfigurations) | 虽在 global 文档路径，明确 US only；GET /v3/settings/partnerprofile 返回 partnerId 与展示/注册信息，reference 使用 token header、WM_MARKET=us、WM_GLOBAL_VERSION=3.1；guide Bearer 样例与 reference 不同，以 reference 绑定并在应用联调验证 |
| E06 | [Item management](https://developer.walmart.com/us-marketplace/docs/item-management-api-overview-1) / [Create items](https://developer.walmart.com/us-marketplace/docs/create-items-on-walmartcom) | 区分 seller catalog、Walmart catalog、匹配建 Offer 与 Full Item Setup；不能从目录匹配推出商家已持有 Offer |
| E07 | [Taxonomy](https://developer.walmart.com/us-marketplace/docs/understanding-the-requirements-for-listing-an-item) / [Get Spec](https://developer.walmart.com/us-marketplace/docs/get-item-setup-requirements) | GET /v3/items/taxonomy；POST /v3/items/spec；按 feedType/version/productType 读取 draft-07 schema；最多20类型，207可能部分；版本固定与条件规则必须处理 |
| E08 | [Bulk item setup](https://developer.walmart.com/us-marketplace/reference/itembulkuploads) | POST /v3/feeds；JSON inline 或文件；MP_ITEM/MP_ITEM_MATCH/WFS/维护/删除有不同语义；变体按同组字段；correlation用于追踪，不是已证明的服务端幂等保证 |
| E09 | [Offer by match](https://developer.walmart.com/us-marketplace/docs/create-an-offer-for-an-existing-walmart-item) | MP_ITEM_MATCH 使用标识和商家 SKU、价格、condition/运重等；4.x与5.x样例不能混用；匹配失败改 full setup必须显式选择并重新批准 |
| E10 | [Full setup](https://developer.walmart.com/us-marketplace/docs/create-a-new-item-full-item-setup) / [schema key points](https://developer.walmart.com/us-marketplace/docs/item-setup-schema-key-points) | MP_ITEM 与 WFS字段不同；GTIN豁免与合规有真实资格；PROCESSED仍可能逐项INPROGRESS，不用CUSTOM填充全部商品 |
| E11 | [Feeds](https://developer.walmart.com/us-marketplace/docs/feeds-overview) | feedId关联处理；RECEIVED/INPROGRESS/PROCESSED/ERROR 与逐项SUCCESS/INPROGRESS/DATA_ERROR/SYSTEM_ERROR/TIMEOUT_ERROR不同；Feed ERROR说明无item被处理；原文重试指导不等于可盲重发新建 mutation |
| E12 | [List feeds](https://developer.walmart.com/us-marketplace/docs/list-all-feed-statuses) / [Feed item status](https://developer.walmart.com/us-marketplace/reference/getfeeditemstatus) | GET /v3/feeds 与 /v3/feeds/{feedId}；includeDetails=true，详细分页limit≤50；特殊字符编码；完整expected SKU结果核对，不能只看计数 |
| E13 | [All items](https://developer.walmart.com/us-marketplace/reference/getallitems) | GET /v3/items，sellerSKU、GTIN、variant等；lifecycle/publication/availability分开；cursor 有限有效期、offset上限，列表不保证任意大店铺一页完整 |
| E14 | [Rate limiting](https://developer.walmart.com/us-marketplace/docs/rate-limiting) | 按seller/app/API、Feed大小限制及响应头安排有界读取，不用一个全局固定吞吐假设；429/413不冒充已处理 |
| E15 | [Sandbox](https://developer.walmart.com/us-marketplace/docs/walmart-api-sandbox) | 静态沙箱用模拟数据；US动态沙箱只覆盖部分API；沙箱结果不是生产授权/真实商家验收 |
| E16 | [API terms / ISA](https://developer.walmart.com/us-marketplace/page/terms-and-conditions) | 数据使用、安全评估与跨境处理条款属于用户负责的实际开放条件；设计不代签、不宣称已满足，不把商家数据作为跨平台复用训练素材 |

官方来源中的1P、其他市场或WFS样例不扩成本批要求。公开schema版本号示例可能不同，不硬编码某个示例为所有商家的当前版本。上线应用必须绑定实际允许的版本及契约修订。

## 2. 官方证据没有证明的行为

- 没有从已读文档证实 Feed POST 的通用 idempotency-key、保留窗口、create-only/CAS 或按 correlation 唯一找回原 wire 的保证。设计采用本地 durable fence，不推导可自动重发。
- Token Detail 没有证明 sellerId，因此真实商家核实使用 profile 的 partnerId；回调字符串/商家输入名称不足以独立绑定。
- 交换 code/刷新响应丢失后的可安全重试保证、refresh token实际轮换/撤权通知等没有完整证实；结果不明保守保留原 attempt/generation UNKNOWN，重授权不抹除 Feed UNKNOWN。
- Feed、item状态中未知值/缺失页/非完整字段不能作为已成功或无副作用证据；结果不明保持待核实。尚未取得 feedId 时不承诺自动匹配Feed列表完成恢复。
- 具体商家实际scope、类目、GTIN/品牌/项目资格、spec版本/媒体可取性、可用库存及最终可售，必须来自该商家的实际合同/返回；本地测试不能证明。

上述缺口已有明确安全行为，不自动要求增加通用框架。实现时只为受影响方法补证据与测试；若必要核心路径被真实合同否定，再按 AGENTS.md BLOCKER 规则处理。

## 3. 主体申请与真实操作边界

[官方入口](https://gecrm.my.site.com/channelpartnerprospectform/) 在实际页面固定 Marketplace US，公司Country提供China并与Hong Kong分列。只证明表单层面可提出申请，不代表已获批、仅大陆主体一定能签约或取得生产权限。用户已明确自行负责此项；不继续代办或把它设为文档前置。

[App注册/审核](https://developer.walmart.com/us-marketplace/docs/app-registration-and-approval-process-for-publishing-to-app-store)要求技术和营销信息、OAuth URL/scopes、支持渠道及demo。当前没有应用凭据、获批scope或测试商家证据；全部真实操作NOT_RUN。适用合同/保险、安全评估、部署与跨境数据批准由用户解决，不自行认定豁免或新增长期平台需求。

## 4. 当前产品投影核对

2026-10-16 使用只读 Figma metadata 核对 `tg48P46SSXl6TBy9lZwg63`：`1560:359`、`429:5323`、`2667:1948` 存在且根节点未标hidden；对应我的店铺、店铺商品、供应链待补全。引用[当前产品Authority](../product/final-ui-ia-authority.md)确定菜单与页面归属；没有新增/修改Figma，没有把原型可见性写成运行功能或用户验收。
