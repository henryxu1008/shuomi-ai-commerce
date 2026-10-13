# TikTok Shop 美国跨境：店铺授权、商品草稿与受控刊登

Refs [#669](https://github.com/qq550723504/task-processor/issues/669)。

- Product Decision: PD-TIKTOKSHOP-US-CROSSBORDER-2026-10-13，来自本 Issue 中用户确认的当前范围。
- Design Basis: Independent Architecture。
- Admission Status: **NOT_READY / DESIGN_CANDIDATE**。此文件先于正式实现；未完成下述合同收敛和适用独立架构检查，不修改生产业务路径。
- 基线：origin/main `4043651d074eaed1bd0b26e823e67680bfc4ae92`。
- 单 Writer、主分支 `codex/tiktok-shop-us`；一个主要 PR 承载本 Delivery Batch。

## 1. Product Outcome、Must、Scope 与 Out of Scope

当前企业内有店铺权限的运营人员，在硕米店铺中心连接美国跨境 TikTok Shop；从现有已保存商品准备可检查的本地 TikTok 草稿；用户明确批准确切资料后，按店铺官方刊登方式提交并查看真实渠道结果。

Must：复用当前身份/企业/成员店铺授权；平台/真实店铺不可混淆；本地草稿持久保存；规则和源商品版本可追溯；发布绑定确切草稿和用户决定；同键不同载荷拒绝；响应丢失持久 UNKNOWN；不泄漏凭据，不伪造连接、数字、审核或可售状态。

本地保存草稿不等于调用平台 AS_DRAFT，也不产生平台资产。平台 Draft 创建和 LISTING 都是外部副作用，须经过相同授权/执行门禁，界面明确区分。完整美国跨境刊登不能以本地草稿或只有协议 client 代替。

Out of Scope：美国本土、其他市场、订单/履约/售后/退款/财务、自动库存写入、达人/带货内容、Agent 自动发布、新 AI 调用、旧数据迁移、独立 TikTok 工作台、通用 OAuth/Saga/同步框架、额外验收工具。现有可用能力和成熟开源/官方 SDK优先复用。

用户尚未创建应用。该事实阻塞真实运行验证，不自动把所有设计或本地实现变成不可做；明确的协议缺口只阻塞受影响的正式路径。

## 2. Product / Figma / Architecture Authority

- [最终 UI/IA](../product/final-ui-ia-authority.md)与 Figma `tg48P46SSXl6TBy9lZwg63` / `31:463`。2026-10-13 已只读核实节点 `1560:359`「我的店铺/优化版」、`429:5323`「店铺商品」存在。
- 商品准备复用[我的供应链设计](my-supply-chain-shein-v1.md)中已批准的商品批次、待适配、补全、已适配和渠道结果投影，不恢复 Task-first 页面。其 SHEIN 业务合同不能被本文件直接扩成 TikTok 合同。
- [Project target](project-target-architecture.md)、[package boundaries](project-boundaries.md)、[greenfield baseline](../product/greenfield-no-legacy-migration.md)、[Legacy Register](../refactoring/legacy-register.md)。
- 当前 Store CRUD、成员 grant、服务权益和 official connection 的有效不变量，以及唯一 `internal/listing/submission.ExecutionKernel`。
- [官方合同证据及缺口](tiktok-shop-us-contract-evidence.md)。没有应用时不引用假店铺/假 API响应作为真实平台证据。

## 3. 已确认根因与设计边界

当前 main 的 `OfficialMerchantCredential` 为 AppID/OpenKeyID/SecretKey/SupplierID；应用 mode 是 SHEIN self/semi/fully-managed。`OfficialConnections` 把加密凭据保存在单个 Store attempt，没有 TikTok OAuth token generation/refresh 或 seller-grant 多店铺语义。不能把 access token 填进 SecretKey，refresh token 填进 SupplierID，或把跨境 mode 假装成 SHEIN 全托管。

当前 `listing/record/target` 的输入/结果直接是 SHEIN goods，TargetIdentity 固定 shein-us；official intent/receipt 绑定 SHEIN payload、SPU/SKC/SKU。数据库也有 `platform='shein'` / `site='shein-us'` 等约束。只增加前端枚举会产生不可执行入口；包装旧 root ListingKit会恢复已退役 owner。

#655/TEMU 正在修改共享 Store、Listing、Supply、schema、BFF 边界。调查已读取准确 PR 文件范围与活动会话；本 Writer 不修改该 worktree、不接管其运行实例。正式修改共享路径前须按其最终 main 组合核对 current contracts/owner。只做本独立文档和官方证据不与其 Writer 重叠。

## 4. Domain / Fact Owner 与调用链

| 事实/行为 | 唯一 owner | 消费方式 |
| --- | --- | --- |
| identity、live Organization/module/member grant | 现有 ZITADEL/authidentity/authz | 当前 middleware与执行前复核，HTTP不能给可信 actor/org |
| 本地 Store、访问、服务/权益、真实 Shop绑定 | Store Center | TikTok专用窄授权/凭据 Port；不复制 Store到Product库 |
| OAuth seller grant及token generation | Store Center中的有界 TikTok OAuth事实 | Integration负责协议；Store持有加密与事务/恢复事实 |
| canonical Product、来源、批准资产 | 当前 Product/Asset owner | 读取指定版本、hash和回执，不复制第二份可变产品 |
| 本地平台草稿/目标版本 | 当前 Listing record/preparation owner | 添加显式TikTok资料变体与site，不新建第二套Listing生命周期 |
| 平台类目、属性、SKU、价格/仓库规则 | `internal/marketplace/tiktokshop/*` | 纯规则、窄Port，不依赖root ListingKit/HTTP/GORM |
| 外部transport/signing/SDK | `internal/integration/tiktokshop` | 仅实现已准入domain-local Port，凭据不越界 |
| publish intent/attempt/claim/fence/UNKNOWN | 当前 Listing Submission | 复用ExecutionKernel及Product库原事务，不克隆状态机 |
| 运行装配与UI/BFF | current application和现有Console | 按Store平台分派窄能力，诚实显示未配置/未开放 |

调用链：现有 Console → 同源 BFF → 当前身份/live授权 → Store/Listing application → TikTok Marketplace窄合同 → TikTok Integration → 现有 Submission Kernel → 绑定回执/渠道观察 → Console。

不新建通用平台框架；需要修正的共享 seam 仅覆盖本次真实消费者。平台相关DTO与规则留Marketplace；Store凭据和providerSDK不能落入Product/Listing/前端。

## 5. OAuth与准确Shop绑定

应用配置固定 US 授权入口、service_id、app_key、版本、callback和最小获批scopes。secret和credential encryption key使用已有私密文件读取/保护方式；没有配置返回明确不可用，不注入fixture provider。应用ID与授权service_id不混用，配置修订会使旧执行绑定失效。

Begin：在当前成员/Store update权限下，持久化 attempt，绑定org、原actor/member、Store、version、应用revision和一次性随机state hash。凭据交换/refresh必须先持久claim，再发送一次；并发/重复回调不得重新交换code。

Callback：只接收code/state，不从回调参数信任org、Store或app。复用当前登录与live权限完成POST；切企业/换成员时不能完成原attempt。callback页面禁止referrer泄漏，并尽早移除含code的URL；不进localStorage、日志、审计正文或PR。用户拒绝授权保持明确失败，不制造已连接。

Get Authorized Shops：根据授权返回的正式shop信息，让用户确认目标Store；本批仅region=US、seller_type=CROSS_BORDER。明确区分Seller Center code与internal shop ID；不能取第一项或自行拼cipher。Store的预期外部编号与返回code/ID必须按定义一致；不匹配不得自动替换Store身份。

一个seller token可能授权多店铺。**TikTok OAuth grant以provider+app_key+open_id为身份，由Store owner持有单一加密token generation；Store绑定只引用grant、实际shop ID/cipher/US market。** 不为每个Store复制refresh token并各自轮换。新grant/绑定和原attempt completion在Store库事务内提交；同一provider+shop ID不能跨org误绑定。本批不开放grant跨企业共享。

schema候选：由Store初始化新增有界OAuth grant/generation与Shop绑定事实；原connection projection仅引用本事实，不能独立签发connected。精确列/constraint/grants及共享attempt接线需在独立准入前定稿；不会在serving启动时AutoMigrate，不迁移保留库或读取旧profile/session。

Token refresh：Store owner使用CAS和有期限的原refresh operation claim；只有一个dispatch者。成功结果验证app/seller/scopes身份与绝对到期时间后原子更新generation；所有引用Store消费同一revision。丢失结果/进程中断保留refresh UNKNOWN，禁止盲重试或回退旧token；用户重新授权是恢复入口。到期/撤权/应用revision不匹配阻止规则与mutation，不显示连接正常。

## 6. 本地草稿与官方规则

草稿继续引用原受权source、Product版本、批准资产、Store/connection/grant generation、market、listing_method和对应官方rule hash；immutable record CAS/同键fingerprint/operation receipt复用当前Listing owner事务。前端只能提交补充项，不能签发ready或provider approval。

先读global_listing_rules，再按GLOBAL_PUBLISHING或LOCAL_REPLICATION使用配套category/rules/attributes；未知或不适用方法明确阻止发布。两种方法都是当前美国跨境范围，不伪装其中一种已实现另一种。

原图/证明继续由Asset事实引用；TikTok image/file URI要有本平台实际上传/读取回执。上传图片/文件也是外部effect，使用原Submission门禁；草稿无需凭据时可保存未齐字段，但必须标注未完成官方规则校验。缺少实际规则、必填属性、仓库、价格币种、证明或图片URI时禁止ready/publish，不猜数字与默认业务值。

用户检查的是本地资料及确切缺失项。Remote AS_DRAFT另作明确外部操作，不能在本地保存时顺便创建商品。US内容按官方规则使用对应语言；本轮不新增AI翻译或改计费合同。

## 7. 外部提交、幂等与UNKNOWN

必须经过现有显式批准/权限、源/资产/规则/Store freshness和Submission Acquire。批准绑定record hash、market、method、payload、图片/证明、Store/grant revision；任何漂移要求重新校验/批准。令牌从server-only授权Port取得，前端/日志不持有。

LOCAL_REPLICATION：只为原US Shop调用Create Product，不复制到其他市场；save_mode必须显式为本次批准的AS_DRAFT或LISTING。idempotency_key绑定原持久execution key，不能每次HTTP重试重生成。没有获准新SendPermit时不发送，即使provider提供幂等参数。

GLOBAL_PUBLISHING：图片/文件、创建global product、发布US分别使用现有kernel的独立stage action和确切target subject。只有上一stage准确receipt持久完成，下一stage才可Acquire。global product ID及每个SKU关联属于Submission/Marketplace回执，不能由客户端手填冒充。

外部调用成功与stage completion receipt/后续stage引用在Product库现有UoW中提交。平台已成功但本地commit不确定时读取原intent/attempt/receipt，不另发；跨Store库与Product库不声称分布式原子事务，发送前重新消费bounded Store grant/connection proof。

请求已发后的超时、响应丢失、取消、lease expiry都保留原UNKNOWN。恢复仅依据与原Shop、external product correlation、exact payload/variant membership相符的官方readback；0条不能推定未发生，多条/关联不足不能选第一条。没有qualified readback时保留UNKNOWN并交出具体原因；不新建通用reconciler或未经准入的manual success入口。

发布回执只说明平台已接收或创建；通过Get Product取得audit/live状态后分别投影Pending/Failed/Activate。不能把HTTP200、code0、global ID或提交成功当成已审核可售。只做本批发布结果查询，不扩大成全店商品/订单同步。

## 8. 权限、边界与资源限制

沿现有source actor私有范围、liveOrganization/module/member Store grant、必要product/preparation/submit权限和Store有效服务权益。管理员不自动取得其他成员来源资料，不以TikTok接入改商业价格/paid plan或覆盖entitlement。

平台API只允许官方固定HTTPS origins，无redirect转发secret，无客户端任意origin。复用现有request/deadline和payload限额；资料不超过当前2MiB record边界；单次provider I/O受bounded context限制。token交换/refresh虽为GET也不自动重试；读请求与不可安全重复effect明确区分。错误投影allowlist，不透传provider原URL、token、body或隐私字段。

用户未授予真实店铺读写、部署、合并或数据操作权限。只有批准的隔离fixture开发检查可以在代码准入后执行；真实操作和用户验收另行记录。

## 9. 验证、验收和停止点

业务行为按TDD先RED再最小实现；必要验证覆盖：错误org/Store/原成员、回调重放/并发、US/CROSS_BORDER精准选择、多店铺grant单次refresh、expiry/scopes/revision漂移、两种listing_method分流、缺规则拒绝发布、同key异payload、响应丢失不重发、阶段回执关联以及准确平台状态。

使用已有测试替身、PostgreSQL测试设施、UI/BFF测试与相关导入边界；不扩完整故障矩阵或验收平台。变更schema后在新的独立空库核对constraints、runtime role与readiness，不触碰保留实例/卷。

最终独立检查仅覆盖真实diff、这条用户路径及高风险增量；CI和fixture不等于产品验收。交付提供实际正常启动命令/入口、私密配置步骤、缺口；用户或指定验证者使用获批应用和店铺确认效果。

## 10. 正式开工准入待办

1. 按官方字段补齐两种流程的规则、SKU、仓库、金额、图片/文件、global publish响应和qualified readback合同；记录未能取得的具体字段，不用假资料替代。
2. 固定Store内OAuth grant/generation/Shop binding的schema、唯一约束、refresh transition，以及current assembly/原connection projection调用路径。只增加此范围所需事实，不泛化为全平台认证框架。
3. 按#655最终组合确认共享Listing record/intent/receipt扩展方式、constraints、准确UoW和源码owner；不修改其活跃Writer范围。
4. 必要独立Architecture Review按AGENTS finding分类完成，明确IMPLEMENTATION_READY后才正式编码；非Blocker实现测试留在同一批，不无限设计修订。

应用未创建继续阻塞真实验证；上述设计待办才是当前正式编码准入缺口。不得把应用缺失当作所有代码均无法开发，也不得把本设计候选当作完成集成。

## 11. Legacy decision

Legacy decision: EXTRACT | RETIRE。
Reusable behavior: 当前Store授权/加密/审计，Product/Asset/source选择，Listing immutable record，Submission claim/fence/receipt，以及合格Console/BFF。
Current owner: 对应Store、Product/Asset、Listing、Marketplace/Integration和App，见§4。
Cutover/deletion condition: 只有触碰并证实仍有效的历史行为才按正确owner抽取、精确切换调用方并退休原路径；不包装root ListingKit，不引入compatibility/双读/双写/tenantbridge，也不为此任务扩大全仓退休范围。

## 12. 独立准入记录

NOT_RUN。作者只完成只读调查、官方证据和此设计候选；尚未签发IMPLEMENTATION_READY。独立Reviewer需读取#669当前范围、本文件和官方证据，按BLOCKER/IMPLEMENTATION_TEST/BACKLOG/ACCEPTED_RISK/NOT_APPLICABLE分类，指出具体Must和阻塞层级。公开文档矛盾/具体合同缺口按其真实影响处理，不以理论全平台要求扩大本批。
