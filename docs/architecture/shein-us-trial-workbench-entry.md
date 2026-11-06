# SHEIN 美国站首试：工作台操作入口

**Design Basis: Independent Architecture，IMPLEMENTATION_READY（仅离线）。** 独立高风险检查已核对三动作、同源会话、原单及 UNKNOWN，没有开工阻碍。 [#1073](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1073) / [#1075](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1075)，原 worktree、分支和唯一 Writer。用户确认在现有登录和项目权限下接通查看原单、提交一次、核对原账，并先用合成数据验证。当前固定后端 `49dd3f6c0` 有内部接口，统一实例前端 `645eaa6cb` 没有相应的代理路径。离线范围沿 [首试产品决定](research-commercial-assessment-v1.md#53-shein-美国站商品卡片首试实施方案)、[内部路由](shein-us-trial-entry.md)、[运行及信任边界](shein-us-trial-runtime-wiring.md) 和 [项目 UI Authority](../product/market-research-selection-centers-v1.md)。

## 用户操作

沿 [研究详情 Figma 12114:1646](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=12114-1646) 的「收集与整理资料」原区块和现有按钮、确认及反馈样式。仅受控项目的市场调研详情可见「SHEIN 美国站首试」，并且必须先通过原单 View 授权。读取在授权后确认尚无原单时，具备原项目写权的操作者看到明确的范围与提交确认：类目 1727 第一页、最多 20 张卡片、一次外部请求、最多 25 credits、新增现金 USD 0。取消、打开页面、切换企业及查询均不发起取数。等待和结果不阻塞既有研究功能。项目资料由原 Collection 发布后，仍用「添加已有商品资料」关联到同一项目，然后进入原智能选品页。

提交时禁用并发操作。已占用或超时后以「查看原单」读取权威状态，可对 UNKNOWN 使用「核对原账」。只有后台明示已确认且未超限，才可用原 submit 恢复尚未确认的本地发布。已确认发布数量（包括零）按实际回执展示。未知用量保持原预留，缺失回执不显示为 0；超限显示实际用量且没有继续发布动作。客户端不自动重试任何写入。若一次提交后没有收到回执且原单还不可见，本会话只继续查原单；重开页面仍由固定批准 ID 和 Product 原占用决定能否发送。归档项目沿原约束禁止提交。独立核账身份的运维入口不在本切片，本次同一已获准操作者兼核账者。

## 信任边界和接线

- Next.js 服务端可选运行设置 `LISTINGKIT_SHEIN_US_TRIAL_TARGET=<project UUID>/<approval UUID>`，缺省或格式无效时关闭。只接一个目标；值是受控路由的选择器，批准事实仍只来自后端 `approval.json` 和 Product 账本。设置不进入客户端 bundle，项目页只接收目标匹配布尔值。项目 URL、已有登录和角色均不能覆盖目标。后端依旧无宿主端口，真实门禁独立控制。
- 同一来源的工作台代理只接 `GET /api/workbench/research/projects/:project/shein-us-trial`、`POST .../submit`、`POST .../reconcile`。按服务端目标选择既有 `/api/v1/workbench/research/projects/:project/shein-us-trials/:approval` 对应三动作。复用 Auth.js server session、当前用户/企业核对、原组织选择 Cookie 和同源写检查。浏览器无需也不能提供 bearer 或凭据。请求不接受 body、查询、任意 Header 透传、来源 URL、金额或通用路由。读取及写入都经原后端实时项目、成员、来源、预算及核账授权，绕过 UI 不增加权利。
- 后端原单读取完成实时授权和撤权复查后才会返回结构化 `404 {code: NOT_FOUND}`，此时代理可返回 `NOT_STARTED` 和已核准的批准 ID。缺路由的 404、代理错误、未授权和暂不可读不能解释成“可提交”。只有已授权的原单或明确的未开始可显示操作区。每次写入带最近原单读取所得 `X-Expected-SHEIN-Trial-Approval-ID`，代理先与当前配置比较；配置切换不把旧确认应用到新批准。该校验头不透传。输出限长、严格解析、批准 ID 交叉核对、`private, no-store`，无原文错误/凭据/上游响应转发。写超时、传输不明或回执不匹配走 OUTCOME_UNKNOWN。限时按后端 View 10 秒、Submit 40 秒、Reconcile 35 秒留出有限传输余量；不改其他代理超时或幂等合同。
- UI 身份/企业/项目变化卸载旧作用域并取消读取，进行中的操作不跨上下文展示。各按钮只执行准确动作，不提交新的请求键。Product Data Acquisition 继续拥有唯一占用、用量、来源发布和恢复事实；Research/Collection 的持久化不变。无新库、表、角色、执行 worker、旧迁移、双写或第二账本。

## 离线验证及现场边界

定向回归覆盖关闭/配置错误/错项目、已登录与企业漂移、跨源写、请求体拒绝、真实授权前不可见、批准替换、已授权无记录与缺路由区分、未知或超限准确投影。使用合成上游和现有 Go 单次发送/原单核账夹具，验证同一项目的正常查看→显式提交→查看原回执，及超时后查询/核账和无第二次发送。验证原研究资料/选品路径。预算沿原决定。本批不更新统一实例或配置、不合并、不挂真实凭据、不真实取数/AI/付费。真实首试仍需来源书面许可、有效门禁及账户证据和独立窗口授权。
