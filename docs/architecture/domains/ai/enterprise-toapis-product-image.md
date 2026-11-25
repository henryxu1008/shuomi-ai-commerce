# 企业 ToAPIs Flare 商品图组

执行 Issue [#1630](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1630)。设计状态：2026-11-25 两轮独立架构复核，达到 `IMPLEMENTATION_READY`。用户于 2026-11-25 确认 ToAPIs 为第二个渠道、型号 `gpt-image-2.6-flare`、保留 GRSai，并授权隔离环境外套图组实测及 Shopify 开发店草稿上传。依赖 [#1628](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1628) 的[企业自有商品图片合同](enterprise-grsai-product-image.md)。

## 用户结果与边界

渠道管理员按已有组织权限登记固定 ToAPIs 国内节点的企业凭据。商品编辑者使用已有图组、模板、Skill、原图和人工采用入口，明确选择 ToAPIs 或 GRSai，从一次计划确认进入主图、轮播及详情。ToAPIs 首版只支持带一张经限定来源策略验证的公开 1688 CDN 原图的自有商品，生成项各提交一张 PNG。工作台说明会向 ToAPIs 发送所选公开原图链接，并由企业账户承担供应商费用；不预留或扣减硕米点数。本企业用量只显示已确认的出图数，供应商实际账单为权威。供应商 billing 的 pending、settled、refunded 不改变出图事实与硕米计量，不推断金额。仅允许 Shopify 开发店草稿实测，现有连接、协议、发布、隐私和线上环境不改。无第二事实源、旧系统迁移、通用代理、任意模型、自动换渠道或供应商文件上传。

## 供应商合同与来源

[Flare 普通版](https://docs.toapis.com/docs/zh/api-reference/images/gpt-image-2.6/generation) 定义国内 `https://api.toapis.cn`，Bearer 凭据，异步 POST `/v1/images/generations`，`model=gpt-image-2.6-flare`，`n=1`，`size=1:1`，`resolution=1K`，`output_format=png`，一个 `reference_images` URL，固定质量 `high`。报价和 intent 保留实际质量 `high`，界面将 1K 档位与 1024×1024 成品区分。供应商请求和查询仅往同一固定 API origin，不跟随重定向。预检、读回和本次试用优先使用现有三张 1688 CDN 原图。禁止把本地文件、企业私有或带签名的地址发给该渠道。仅接受组织授权目录中的公开 1688 CDN 商品原图：固定 `https://cbu01.alicdn.com/img/ibank/` 文件路径，地址无凭据、自定义端口、百分号转义、点路径段、查询参数和片段。计划准备校验地址和来源形态；执行使用该槽位冻结目录中的实际下载字段 URL，通过公网安全客户端在无 Cookie、无鉴权、禁止跳转的条件下读取同一个将发送给 ToAPIs 的完整 URL，并校验原图类型、3 MiB 上限和已确认摘要。其他图片来源需要有明确的公开分享依据再扩展。ToAPIs 会再次读取该公开地址，产品声明该模式依赖源站在供应商读取时提供同一内容，平台不能证明供应商取得的字节。后台不尝试转发本机私有 assets，也不自行改变可见性。计划准备拒绝无可用公开地址或字节超限的来源，执行前重复验证同一来源。

[任务查询](https://docs.toapis.cn/docs/zh/api-reference/tasks/image-status) 与型号示例共用发送节点 `GET /v1/images/generations/{task_id}`。提交时要求有效 ID 和对应型号，支持 pending、queued、in_progress 作为非终态；以同任务 ID 且型号字段存在时匹配的明确 completed 或 failed 确认终态。完成时要求 `result.type=image` 且恰好一条图片结果，结果由原公网安全无鉴权下载器获取，HTTPS、限制体积和尺寸、不透传供应商凭据。如果结果 URL 或内容不可用，保留已证实的供应商出图事实及现有无效输出收口。文档的 1K 是档位，实际宽高可能变化；只接受原本为正方形且在有界像素预算内的实际 PNG，使用已有 `golang.org/x/image/draw` 将其归一到 1024×1024 后采用，初次与恢复同用一个逻辑。保留原供应商定位摘要、对归一结果再次做原媒体检查与资产字节证明。

## Owner、准入和效果

- Model & Services 管该企业登记、渠道启用及密钥版本。在已有登记与加密凭据 owner 增加独立固定 ToAPIs 模板、渠道和显式 runtime 准入。由现有 managed-route catalog 导出候选，绑定和 revision 由协议、企业、连接、凭据版本与端点摘要派生；GRSai 的身份摘要与选项保持原值。凭据只为匹配企业、渠道、地区、端点和版本的被选用连接解析。平台凭据和其他企业密钥不可选。授权原角色、实时检查和撤销合同延用。
- ImageAgent 管计划、明确费用确认、冻结槽位、派发与核实。用现有 provider-usage 事实和 Temporal 能力，支持逐槽位一次网络提交。ToAPIs 独立供应商协议及品牌在计划、确认、结果和用量可辨识，以现有语义填充质量和实际输出。所有已持久化 GRSai 和平台计划按原字段读取。
- 沿用原持久 claim、防并发和发送边界。先持久化原供应商任务号，再轮询原任务。超时、空响应、未匹配回执或未知状态留给原核实入口；没有任务号的请求保留待核实。文档中的 client_business_id 未证明 POST 去重，且本阶段不开发新的关联查询或扩展恢复状态。禁止自动回退到其他渠道以及自动重发未知提交。用户主动发起的新计划仍需再确认。
- 使用原运行/重启/查询限时、最终事实、素材来源和容量收口。GET 恢复需要同已授权身份、企业、计划、槽位、凭据版本及已保存任务号。凭据失效或查询错误保留待核实。不增加表或新事实归属。登记 schema 只扩展既有模板约束并验证 GRSai 可共存，按独立准入保证单个渠道可选的运行配置。
- Product Asset 仍是素材和人工采用的 owner，沿现有哈希证据与审核入口。开发店通过原 Workbench 出草稿；隔离资产如仅本机可读，使用 Shopify 自身的文件上传和变体关联入口，本期不加正式发布、变体媒体绑定代码或公网入口。

## 验证

用官方响应形态模拟受理、状态及终态，核对协议、凭据保护和单请求、固定来源、未知不重发及恢复同任务。覆盖两独立渠道正常登记和显示、历史计划摘要、实际尺寸归一化、无硕米点数。复用现有隔离环境和黑灰白三张原图，使用现有模板和 Skill 完成整组，检查商品结构与颜色真实性，必要时迭代企业 Skill。开发店以 Shopify 读回草稿、图组与六个颜色和尺码组合。保留原草稿 PR #1629，报告模拟、自检、真实平台与用户验收各自的结果。
