# Amazon 美国站隔离订单同步与查看

本补充设计用于 [#1623](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1623)。沿用 [共享设计及独立评审 #1624](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1624) 的 Amazon 协议和隐私边界。Shopify 的隔离订单链路已合入 `main`，本次基于 `49ac60a3c6361405ebf0abefa2b75fb69c38fed3` 复用 [#1626](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1626) 的同步、列表、详情和验证方法。

## 使用范围

测试企业的成员从「订单管理 → 平台订单记录」选择自己的合成 Amazon 美国站店铺，同步最近 30 天下单的订单，查看订单号、时间、平台状态、商品名称、SKU 和下单数量。列表按店铺和订单身份去重。商品授权须配合明确的模拟订单用途授权。界面如实显示同步范围和部分结果，缺少状态显示「平台未提供」。

本次只允许合成授权及订单。在生产组装中不注册 Amazon 订单 reader、可用能力或真实凭据入口。不读真实 Shopify 或 Amazon 订单、不处理发货回传、退款、改单、经营分析、买家数据或历史补采，也不送审、部署或合并。

## Owner、调用链和同步边界

- Store Center 持有订单用途授权，与 Store 的现有企业、店铺及 Amazon 商品 OAuth 应用/授权绑定关联。合成注入用独立表在隔离 PostgreSQL 保存一次明确订单同意和单调代次，绑定企业、店铺、源卖家授权代次和应用。生产安装器不创建该表。当前成员及 Store 读取、订单读取和同步权限仍由现有域校验。企业切换、店铺停用、授权代次变化或撤权均禁止读旧观察。授权证明、任务和读取不带明文密钥，商品令牌及授权不会直接发放订单权限。
- `internal/integration/amazon` 的订单适配器只接受显式合成 transport，使用固定北美端点和 US marketplace。每次读前由隔离 Store 应用层装配验证当前订单用途、身份与绑定。Store Observations 沿现有请求、Repository、worker port 和 HTTP 组装持有同步及快照。浏览器复用原 Next.js 代理和订单页面。只发送订单 GET，回执、命令与详情路由沿现有边界，无写平台行为。
- 使用现有观察命令键及 scope/ID；一次 Amazon 同步只允许一店，按服务端时钟确定窗口，开始为同步时刻往前 30 天，截止至少两分钟前。原范围在各页冻结，校验闭区间与 US 站。响应必须有 `orders` 数组，末页省略 `pagination`。同一 `pagination.nextToken` 只作该任务原范围的 `paginationToken`，有效期 24 小时。游标无效、重复、达到预算、平台拒绝或响应异常时明确失败或部分结果，旧完整结果仍可见。任何新增游标和查询值都留在服务端。 [Amazon searchOrders](https://developer-docs.amazon.com/sp-api/reference/searchorders) · [官方模型](https://github.com/amzn/selling-partner-api-models/blob/main/models/orders-api-model/orders_2026-01-01.json)
- 订单身份由企业、店铺、来源绑定和 `orderId` 决定，商品行由 `orderItemId` 决定。只保存标题、SKU、数量、创建和更新时间、已知状态及来源标识。范围或站点错配的记录丢弃并标记覆盖不完整。重复版本保留较新的平台更新时间，同版本不同内容标记部分。无需 `includedData`。不保存原始响应或额外数据集。列表和详情只打开当前同一绑定，不加入报表、物流或仓配工作台。
- 沿用 SQL 原子页、检查点和最近完整结果，Store owner 在同一隔离 PostgreSQL 的页发布事务锁定 Store 和当前 Amazon 双代次授权，撤权先完成时晚到结果不能发布。任务重跑使用原键，恢复从既有检查点只读继续。跨体系 IAM 沿现有活授权检查，读取再次核对。Legacy decision 为 N/A，无旧事实或迁移。

## 工程验证及平台实测

独立架构评审确认本合成范围为 `IMPLEMENTATION_READY`。Amazon 商品授权在同一组织和店铺行推进代次，隔离同意表以稳定 Store 键引用，发布时按 Store → Amazon 当前授权 → 订单同意的顺序锁定并比对应用、beta、原授权代次和独立同意代次。商品重连可以推进原授权，旧订单同意立即失效。

测试覆盖合成 transport 请求限制、固定窗口、去重、部分不替换上次完整、游标反例、错站、未知状态、两企业隔离、只有商品授权、原代次失效与晚到响应。使用既有前端组件和真实 Go HTTP/隔离 DB 测试，尽量复用 Shopify 的浏览器接线方法。评审 PR 记录实际完成的验证以及跳过项。公开 SP-API 应用未提供，真实开发者应用权限、卖家流程、远端账号及客户数据生命周期留待独立授权和接入评审。
