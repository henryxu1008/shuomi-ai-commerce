# Walmart 美国站：多商家授权、商品草稿与受控刊登

Refs [#733](https://github.com/qq550723504/task-processor/issues/733)。

- Product Decision：**PD-WALMART-US-MULTIMERCHANT-V1-2026-10-16**。用户确认美国站、开放给多个商家，并确认首期范围；平台申请由用户自行处理。
- Design Basis：**Independent Architecture**。
- Admission Status：**NOT_READY / DESIGN_REVIEW_PENDING**。本文件是设计候选；独立检查完成前不修改正式生产路径。
- 调查基线：main `db47d15e563c0ced7469ba12d251d8c7be1da4dd`，2026-10-16；一个 Writer，分支 `codex/walmart-us-design`。
- 当前授权仅覆盖设计、必要调查、文档检查及主要 PR；设计准入不自动启动实施、部署或真实平台操作。

## 1. 用户结果与本期边界

有权限的企业运营人员在现有店铺中心连接 Walmart Marketplace 美国站商家，查看其平台商品，从已保存商品形成可检查的本地草稿，明确批准确切资料后提交，读取逐 SKU 处理结果与实际发布状态。

Must：

- M1：当前 org/actor/member、模块与 Store grant 有效；真实商家、企业、店铺、应用和生产/沙箱不能混淆；凭据仅在服务端。
- M2：商品读取保留来源、观察时间、覆盖范围与缺失状态；远端观察不成为 canonical Product 或新的库存 owner。
- M3：本地候选可保存/回读；正式 Target 绑定固定商品/素材/商家/规则版本；确定性校验，不能猜 GTIN、合规或履约事实。
- M4：刊登绑定用户批准的准确 Target、SKU 集合、Feed 类型和 wire hash；重放不重复发送，同键不同载荷拒绝；UNKNOWN 不盲重发。
- M5：分别展示提交受理、逐 SKU ingestion、实际 publication/lifecycle/availability；失败、待审核、部分成功和未核实均不冒充成功。
- M6：沿现有产品入口和唯一 owner 接线，正常消费者可完成本期路径；空配置只关闭对应外部能力，不伪造连接或平台数据。

本期支持新商家 Offer 刊登的两种明确路径：匹配 Walmart 已有目录商品（`MP_ITEM_MATCH`），或提交新商品资料（`MP_ITEM`）。用户选择路径；不因匹配失败自动改成新建商品。WFS/转换 Feed 不属于本期。首次刊登必需的报价和按当次 schema 明确填写的库存/履约字段属于批准资料；这不开放独立价格/库存写 API或自动同步。

Out of Scope：价格/库存自动同步、独立调价/补库存、订单履约、退货退款、WFS、其他市场、Agent 自动发布、新 AI/provider 调用、自动 GTIN 豁免申请、自动签署合规声明、更新已有成功 Offer、通用 OAuth/平台 SDK/Saga/同步或验收系统、旧数据迁移。已存在 GTIN 豁免可按官方合同消费；未知/待审资格保留缺失或处理中。

Threat Model：防止本系统并发/重放导致跨企业绑定、凭据外泄、错误商品覆盖或重复发送；处理本系统进程/网络/数据库响应丢失。商家主动在其他系统修改同一远端商品的事实由只读观察报告漂移，本期不提供自动覆盖/对账。没有新批准的安全豁免；应用、合同、保险、安全评估与数据部署批准由用户负责，属于真实开放依赖，不能写成 PASS，也不阻塞本设计。

## 2. Product / Figma Authority

引用 [最终 UI / IA](../product/final-ui-ia-authority.md)、[greenfield 基线](../product/greenfield-no-legacy-migration.md)、[当前供应链合同](my-supply-chain-shein-v1.md)、[Store](store-center-current-application-v1.md)、[平台观察](store-center-platform-observations-v1.md)、[CompletionDraft](completion-draft-structured-v1.md)、[TikTok 已批准特性合同](tiktok-shop-us-v1.md)、[项目边界](project-boundaries.md)与[Legacy Register](../refactoring/legacy-register.md)。其他平台的协议和历史实现结论不自动适用于 Walmart。

2026-10-16 只读复核 Figma `tg48P46SSXl6TBy9lZwg63` / 页面 `31:463` 中当前节点：`1560:359`“店铺中心 / 我的店铺 / 优化版”、`429:5323`“店铺中心 / 店铺商品”、`2667:1948`“我的供应链 / 我的选品 / 待补全”。本期复用上述入口和现有供应链资料/上传路径；不新增顶层菜单、独立 Walmart Workbench 或 Listing Center。节点示例平台、数字和状态不代表实现/验收事实。

首期用户路径：`/workbench/stores` → 店铺详情授权；`/workbench/store-products` → 手动读取平台商品；现有“我的供应链”保存草稿/准备 Target → 检查资料并批准刊登 → 原上传结果详情“查询平台结果”。本地保存草稿不调用 Walmart，不声称存在官方远端草稿 API。

## 3. 当前缺口与复用边界

| 当前代码证据 | 结论与本期变化 |
| --- | --- |
| `internal/marketplace/walmart/*`、`internal/integration/walmart` 只有 README；Asset 有 Walmart 配方 | 不是已接通平台；补有界规则/窄 HTTP adapter |
| `storecenter/store.go` normalizePlatform 仅 SHEIN/TEMU/TikTok；Store schema/DTO/BFF/表单同样限平台 | 扩展显式 Walmart US variant，不把 token 装入 SHEIN SecretKey 或 TikTok Shop DTO |
| `storecenter/product_execution.go`、`app/storeobservations/application.go`、当前平台观察 DTO | 按用途扩 Walmart merchant/read-only 句柄；普通商品读取不获得发布能力 |
| Target/OfficialIntent/OfficialReceipt 为显式平台类型 | 增加 Walmart typed variant；不以未校验任意 map 承载整套协议 |
| `submission.ExecutionKernel` 与同 Product DB official UoW 已有 claim/fence/UNKNOWN/atomic finalize | 继续唯一发送与恢复 owner；增加有界 Walmart Feed admission/receipt/result 解释 |
| `integration/persistence/listing/submission/repository.go` 已成功 effect 保护显式按平台分支 | 新平台必须在同事务补齐规则；默认分支不能放过 pending Feed 的第二次发送 |

现有架构的具体限制：共享内核的 `succeeded` 是一次已定义外部调用的成功，不足以直接表达异步 Feed 内所有商品已发布。解决方式是在同 Submission owner 内持有 Feed 受理回执与原生结果观察，UI 分层投影；不新增第二套执行状态机，也不把长时间 Feed 处理占用一个 10 秒发送 lease。

## 4. Domain / Fact Owner 与调用链

| 事实/行为 | 唯一 owner | 主要 port → 实现 → 注入 → 消费者 |
| --- | --- | --- |
| 身份/企业/模块/Store 权限、服务权益 | 现有 identity/authz/Store/商业 owners | 原 live authorizer → 原实现 → currentapplication → HTTP/worker |
| OAuth attempt、商家绑定、加密 token generation | `internal/storecenter` | 有界 `WalmartAuthorization`/repository → Store 持久化 + integration transport → 可选 Walmart 应用配置 → 店铺授权与短生命周期句柄 |
| canonical Product、来源、批准资产 | 当前 Product/Asset owners | 原私有 source/asset reader → 原 owner → supply application → 草稿/Target |
| 通用未绑定候选 | `listing/completiondraft` | 原 CAS/历史/结构化快照；本期仅显式采用已支持的标题/描述，不自动采用结构化候选 |
| Walmart schema、资料、SKU/Offer 规则 | `internal/marketplace/walmart` | `RuleReader`、typed Draft/Feed/Result → 本域规则 + integration adapter → Target/preparation |
| 正式平台草稿、版本与命令回执 | `listing/record/target` | 原 Source/Store/Asset proofs、Walmart variant → 原 Product DB UoW → 现有编辑/检查入口 |
| 提交、SKU 占用、Feed 回执/结果、UNKNOWN | `listing/submission` | 原 ExecutionKernel/official UoW + Walmart 窄方法 → 原 Product DB adapter → 现有 upload Operation/worker |
| 远端商品列表、覆盖与查询 | `storecenter/observations` | 独立 `WalmartProductsReader` → integration/walmart → 原 observation worker/仓库 → 店铺商品 |
| HTTP/OAuth/DTO/错误解析 | `internal/integration/walmart` | 实现以上 domain-local ports；不拥有业务状态，不暴露任意 URL/API 代理 |

调用链：Console → 同源 BFF → 当前身份/live grants → Store/Target/observations → Walmart 窄接口 → Integration → 原 Submission/所属 UoW → 原页面。源码、模型、DTO、runtime 配置与 consumer 必须一起接通，不能只导出叶子 client。CommerceTool/Agent 以后如消费同一能力，仍走这些窄 port；本期不增工具和模型调用。

## 5. 官方合同与 API allowlist

详细来源与不确定项见 [官方合同证据](walmart-us-contract-evidence.md)。本期固定 US Marketplace，`platform=walmart`、`site=walmart-us`、market=us；沙箱和生产是不同受控配置与身份命名空间，绝不自动 fallback。

| 用途 | 官方方法/路径 | 本期限制 |
| --- | --- | --- |
| OAuth | 固定官方 authorize；POST `/v3/token` | authorization_code / refresh_token；不接收商家主密钥 |
| 权限/有效期 | GET `/v3/token/detail` | 根据实际 full_access/view_only/no_access 判断用途 |
| 真实商家 | GET `/v3/settings/partnerprofile` | 核对 partnerId；该 US-only reference 要求 `WM_MARKET=us`、`WM_GLOBAL_VERSION=3.1`；不把其他 Global API 当 US 默认 |
| 类目/规格 | GET `/v3/items/taxonomy`；POST `/v3/items/spec` | 固定 Feed 类型、具体 schema version/product type；207 部分结果不得冒充完整 |
| 商品读取/匹配 | GET `/v3/items`、官方 Item Search | seller catalog 与公开 Walmart catalog 结果分开，不从标题相似推定匹配 |
| 刊登 | POST `/v3/feeds?feedType=MP_ITEM` 或 `MP_ITEM_MATCH` | 固定 JSON wire；一个 Target 的 SKU 集合一份 Feed，不混入其他 Product/店铺 |
| 结果 | GET `/v3/feeds`、`/v3/feeds/{feedId}`；GET `/v3/items` 按 SKU | 只读；逐页核对原 Feed/SKU/商家，publication 与 ingestion 分开 |

最小申请 Items 与 Account Profile（只消费 profile 读方法）；实际 token/detail 的 item/feeds 权限按调用检查，不假设 consent 展示名称等于返回 key。不申请 Orders、Payments、WFS、Pricing/Inventory 写权限。若官方实际应用把某个必要读取归入额外 scope，先核实对应方法并仅增加必要读权限，不以“全选”解决。

## 6. OAuth、真实身份与凭据

应用配置持有固定 clientID、private secret 文件、encryption-key 文件、application revision、环境、回调、官方 origin 和获批能力；schema/profile 版本不从前端决定。浏览器传来的 `walmartCallbackUri` 仅与 allowlist 比较，不能作为任意跳转或请求 URL。

Begin 在原 live Store.update 权限下保存一次性 attempt：org/原 actor/member/StoreID+version/app revision/environment、state hash、nonce、固定 callback、expiry。Callback 校验 state/clientId/type 与原上下文，code/sellerId 只作为待验证输入；不信任回调中的 org/店铺，不记录 code/token，不进入 localStorage、日志、Temporal 或公开 URL 历史。回调页面沿既有 no-referrer/尽早清除 query 的方式，再以认证后的 POST 完成。

Store 先 durable claim 再交换 code：`WM_PARTNER.ID` 使用本 attempt 的 sellerId；凭据必须先加密持久化，之后才允许只读重试核实。Token Detail 验证实际 token 有效期/权限；Partner Profile 的 `partnerId` 必须与该 sellerId 精确相等，再让用户确认商家展示名称/编号并完成绑定。`partnerStoreId`、注册地、店铺名不是 partnerId；中国注册地不排除 US 市场。

复用 `workbench_store_merchant_bindings` 全局身份占用：identity hash 来自 provider+environment+US+partnerId，不能纳入 appID、token、可改名称或重新授权次数。不同应用/重授权/断开后新建 Store 不得跨企业占用同一真实商家；本期一个商家账号绑定一个原 Store。绑定占用、grant 更新、attempt bound 与连接版本在 Store DB 同事务；已有永久身份占用的保留/删除规则不变。

仅新增两个有界 Store owner 表，沿现有 schema-init/窄 runtime role，serving 不 AutoMigrate：

| 事实 | 字段/约束与状态 |
| --- | --- |
| `workbench_store_walmart_oauth_attempts` | org+attempt PK；Store/原成员/app revision/environment/state hash/nonce/expiry/version；created → exchange_dispatched → credential_received → bound，或 rejected/unknown；code 发送后结果不明即 unknown；暂存密文只为可恢复的身份核实，bind 时清除 |
| `workbench_store_walmart_oauth_grants` | org+grant PK、org+Store 唯一、environment+US+clientID+seller identity 唯一；canonical 密文、AAD/keyID、access/refresh absolute expiry、scopes hash、generation/version、refresh operation claim/expiry/input hash；active/refresh_dispatched/refresh_unknown/revoked/expired |

当前连接 projection 只引用这些事实，不独立签发 connected。AAD 绑定 provider/environment/org/app/grant/generation；令牌不装进旧平台字段。refresh 采用原 CAS/dispatch fencing；一个 generation 一个发送者。响应包含新 refresh token 才替换；未包含则保留原 token 与原到期边界，不把每次刷新当重新延长一年。官方 expires_in/token/detail 与已知授权期限决定有效性，不把文档样例 900 秒写成不可变合同。

交换/刷新在发送后丢响应、lease 过期、进程中断或 DB commit 未知时，读取原 attempt/operation；未取得已持久化成功结果不得重发。refresh_unknown 暂停新句柄，用户重新授权为恢复入口。重授权推进 generation，迟到旧刷新不能覆盖；永久商家绑定保留。断开/撤权/过期关闭新外部操作；已发 Feed/UNKNOWN 回执仍保留原商家和原资料，不因断开重建提交。

本期不新增未经官方合同核实的撤权 webhook；每次外部用途重新核对本地 live grant、到期与 token/detail 当前权限，401/403 停止相应用途并要求核实/重授权。只读失败可以更新不可用投影；不得因此自动重发业务 mutation。对远端撤权仅承诺当前官方检查结果，不声称本地有实时通知。

## 7. 本地草稿、规则与素材

无应用/无店铺时使用原 CompletionDraft 保存候选，原商品/历史/结构化快照不改变。正式 Target 必须取得当前私有 source、绑定商家、批准资产与官方规则；显式采用保存的 CompletionDraft 版本只带当前已支持文字，并提示属性/价格/库存候选尚未自动带入。平台字段在 Walmart Target 编辑器明确填写并保存；不暗改 Product 或替换 SKU。

Walmart Draft 包含：listing mode、Feed type/spec version/productType、固定 source Product/版本、每个 source SKU 的 sellerSKU/identifier/condition/用户报价及 schema 要求的属性、variant group、批准图片引用、商家/权限/规则/asset hashes。核心字段使用显式类型；随 productType 变化的属性用有界 JSON 值，并由完整官方 draft-07 schema 校验，不能作为任意 API body 透传。

复用仓库已有 `github.com/santhosh-tekuri/jsonschema/v6`（go.mod）执行 schema 规则，使用 draft-07；不依赖 CommerceTool 的业务 owner 或手写 JSON Schema 引擎。只从受控 Walmart 端点取得有界 schema；编译器禁止任意远程 `$ref`/本地文件 loader，缺失依赖或不完整规则拒绝该 Target ready，不回退旧样例版本。稳定版本/hash 进入 Target，提交前再次核对当前适用版本；变化须重新保存并批准。

sellerSKU 由本系统为商家+source SKU 生成一次、展示给用户确认并永久保留，长度/字符按该 spec 检查；不会把其他系统已有 SKU 自动收编为可覆盖目标。Walmart catalog ID/GTIN 属于商品目录，sellerSKU 属于该商家 Offer，均不替代本地 canonical ProductKey。变体按同一 Feed 类型与官方规则组织，原 source SKU 一一映射；不增删或合并来源变体。

图片沿现有 Asset 显式批准和 `httpimage.ValidatePublicHTTPSURL`/有界下载检查，绑定实际字节 hash、用途和尺寸。仅使用可供平台读取的已批准公开 HTTPS 图片文件，不把登录页、localhost、私网、含凭据/短期私密授权的 URL 发给平台。不新增 S3/CDN 发布/上传系统；已有素材不满足时明确阻止相应新商品资料 ready，用户在原素材路径提供可用图片。发送前重查绑定/字节；平台后续获取失败按真实结果显示，不把一次探测当永久可用保证。

GTIN/CUSTOM、品牌授权、产地、危险品/税费/履约等必须由真实商品/商家资料满足。AI 不能补造；本期不自动提交豁免或接受平台项目。MP_ITEM_MATCH 和 MP_ITEM 只依据对应官方版本构造 wire，不借用 WFS TradeItem 或其他市场 schema。

## 8. 提交、异步结果与持久化

沿原 Target confirmation → upload Operation/item → 原 worker → Store server-only publish 句柄 → 原 Submission Kernel。用户批准精确 record revision/hash、mode/feedType、SKU 列表及 payload fingerprint；worker 发送前重新消费原 actor/member/Store live authorization、service、连接/generation/scopes、source/asset/rules。批准后材料变化不自动跟随。

Product DB 的既有官方 intent/receipt 增加 Walmart typed variant，保留 feedId、correlationID、market/environment/merchant identity、完整预期 SKU 集合、wire hash、原 record/attempt/fence。token 不入 Product DB。仅新增以下必要事实，仍在原 Submission owner/UoW：

| 事实 | 约束与 owner |
| --- | --- |
| `listing_walmart_sku_bindings` | unique(environment,US,merchant identity,sellerSKU)；绑定原 ProductKey/sourceSKUKey、Store/target、首次命令；同键不同来源冲突；Feed 未发送也不能把已展示 SKU 偷换到别的商品 |
| `listing_walmart_feed_observations` | org+observation PK，引用原 accepted receipt/attempt/Feed/merchant；request generation、完整/partial 标记、逐 SKU ingestion/errors、官方计数与 response hashes/observedAt；append-only，最新指针以 CAS 防迟到读取覆盖；不保存第二份商品或命令正文 |

SKU 占用、发送 intent、kernel target claim 在同 Product DB 事务；accept receipt 与 kernel CompleteOfficial 原子提交；结果观察与对应 evidence/head 同事务。复用现有 command key/commit-unknown 读回；HTTP 在事务外，只在持久 claim 之后发送。数据库恢复不增加跨 Store/Product DB 分布式事务：发送前取得短时 Store proof 并重查；Product UoW 不改 Store 授权事实。

稳定执行 target 继续是 org+platform+Store+SubjectID；SubjectID 由原 ProductKey、永久 merchant identity、US 与 `walmart-offer-create` 副作用单位确定。record/version/Feed ID/应用/token/改标题/SKU 顺序不进入可绕过 UNKNOWN 的身份。sellerSKU 独占约束防止另一个本地 Product 覆盖同商家 SKU。

首次发送前读取同商家 seller catalog：若任何目标 sellerSKU 已存在且不属于原操作已核实回执，则冲突并停止，不自动用 setup API 覆盖。外部不存在检查只是辅助；本系统永久随机 SKU namespace、原事务占用和用户确认共同限定本期新建路径，不接受任意现有 sellerSKU 的更新。

### 8.1 四类结果不能混同

| 证据 | 内部处理 | 用户显示 |
| --- | --- | --- |
| 本地草稿/校验通过 | 未发送 | 已保存/可检查，不是平台成功 |
| 正确响应返回非空 feedId | 原 kernel 此次 `walmart_submit_feed` succeeded，meaning=Feed accepted；永久保存回执 | 平台已受理，待处理 |
| 原 Feed 全部预期 SKU ingestion | 保存该 Feed 的原生状态、每 SKU 结果与覆盖 | 处理中/待审核/失败/部分完成/已入库 |
| 当前 seller catalog 的 SKU publication/lifecycle/availability | 只读事实投影，独立于 accepted receipt | 已发布/未发布/不可售或未知，附观察时间 |

Feed PROCESSED 不推出每 SKU SUCCESS；ingestion SUCCESS 不推出 publication PUBLISHED；PUBLISHED 不保证有库存、Buy Box 或消费者一定可买。未读取的字段不补默认值。整个 Product 的成功/失败统计按准确 expected SKU 全集，缺页/重复/多余 SKU、计数不符、身份不符均保留不完整，不能签发全量结果证据。

首版在原结果详情提供手动“查询平台结果”，只读刷新既有 feedId 与 SKU，并保存观察；不建立新轮询/Scheduler。分页每页最多 50 Feed item，继续在既有 Operation/响应边界内有界读取；超过预算可续读/再查询，部分结果仍不能用于重发准入。较早 read generation 不能覆盖较新结果。平台原生未知状态保留原值与 unknown 投影。

### 8.2 发送、重试与 UNKNOWN

`WM_QOS.CORRELATION_ID` 用于追踪，不是已经证明的幂等键。应用不提供任意 POST 自动重试；不得用官方面向只读查询的 backoff 示例授权重复创建。以下分支由原 Submission owner 负责：

| 失败点 | 结果/恢复 |
| --- | --- |
| claim 前失败 | 未发送，可在相同命令语义下重试 |
| durable claim 后未能证明发送结果，包括 crash/cancel/超时/截断/无有效 feedId | 原 attempt UNKNOWN；保留 target/SKU fence，读取原 durable receipt，不生成新键重发 |
| 取得 feedId 后 DB commit 结果不明 | 按原 intent 读回；有原 accepted receipt 只查询它，无证据仍 UNKNOWN |
| Feed pending/某 SKU INPROGRESS/审核中/观察不完整 | 禁止第二次 create Feed，换 Target revision/应用/Store 不绕过 |
| Feed ERROR 且官方证明没有 item 被处理，或 exact SKU DATA_ERROR 证明该 SKU 未被接纳 | 用户修改并重新批准后，才可提交原失败 SKU 的新版本；原成功 SKU 不重发 |
| SYSTEM_ERROR/TIMEOUT_ERROR/404/列表暂未出现/只有同标题或 SKU 存在 | 不能证明原写入未发生，继续核实；不自动解除 UNKNOWN |

已成功接受 Feed 后的重提准入是同 Submission repository 内的 Walmart 特定 policy：锁原 stable target 与 SKU bindings，核对完整原 Feed evidence、允许修复的 exact failed SKU 子集、新的用户批准/input hash 后，才原子推进原 fence epoch 与新 intent。原未终结 Feed 或成功 SKU 永远不能进入修复子集。部分成功保留已成功的 SKU/ID；修复失败 SKU 不被 UI 当更新已有成功 Offer。其他平台的 admission policy 不变。

对“响应丢失且无 feedId”，本期不承诺自动恢复：Feed 列表/时间窗口/correlation 相似项不是 exact payload 证明。只有可经官方只读证据完整关联原商家、Feed、SKU 集合与原操作的结果，才能消费原 ResolveUnknown；缺字段时保持 UNKNOWN，展示原 correlation/时间供用户向平台核查。不新增“人工点成功/清除 fence/再试一次”按钮，商家重授权也不清理原 UNKNOWN。

## 9. 商品读取、撤权与缓存

扩展原 Store observations 的 platform/kind allowlist、只读 Merchant port/DTO/schema/preflight/BFF 筛选与 runtime 注入。Walmart ProductsReader 只暴露商品列表/详情，不含 mutation。原 StoreProduct purpose 与 member grant 每页及写入前重新检查；撤权或换企业中止，本批已取结果按原部分覆盖规则保存，不泄露给新企业。

`GET /v3/items` 的 cursor 有效期和 offset 上限按官方 endpoint 处理；cursor 失效停止当前完整覆盖声明，不能把缺页当平台空店。大店铺沿原 bounded sync operation 保存可见进度；不改变其他平台预算。观察仅提供原平台已返回的价格/库存/可用性字段，不额外申请 Inventory scope 拼完整性。对本地申请发布的商品，采用原 accepted receipt 的 expected sellerSKU 集合回读，不能由客户端 feedId 或任意 SKU 跨店查询。

本地连接状态、token/detail 验证与实际请求时间均保留。缓存以 org/store/environment/app revision/grant generation/scope/rule version 隔离；缓存不能签发授权、成功或跨商家规则。规则与列表的可复用内容不包含凭据或不必要的税号/profile 原始正文。

## 10. 运行装配、边界与资源限制

配置加入现有 currentapplication 的可选 `walmartUS` 应用入口，复用已有 secret-file、AES-GCM、HTTP client、schema-init、runtime capability/preflight、Store/Listing/observations 接线。配置缺失清楚显示未配置；Store CRUD、已有平台及 CompletionDraft 不整体停用。runtime role 只取得本 owner 现有窄权限；Store/Product 数据库各自持有自己的事务。

HTTP/BFF 使用原认证/同源/strict DTO/ETag/幂等，生产参数中禁止组织自报、任意 URL/Feed type 或原始 provider body。OAuth callback 输入上限 64 KiB；沿当前 Target/BFF 2 MiB 上限并取平台限制的更小值；按单 Product 构建 Feed，不为绕过大小限制偷偷切多份。规则/列表/Feed 读取响应各有上限、分页/超时/累计预算；不足显示覆盖不完整，不截断后当成功。

发送遵守原 ExecutionTimeout/lease；长时间读取不延长发送 permit。401/403 不回退身份；429 按当前商家+app+endpoint 的官方预算与响应头退避，只读请求在 deadline 内有界重试；mutation 只按§8.2处理。身份、token、code、secret、完整 payload/profile 不进入日志；审计记录 actor/操作/版本/结果摘要。公开 correlation/feedId 也经原记录授权读取，不在不同企业日志面板串联。

原 SDK 优先评估：当前已有 OAuth2、JSON Schema 和 HTTP 库可用；Walmart token 请求的 partner header/camelCase authorize 参数由窄 adapter 映射，不生造通用 OAuth 服务。没有核实合格官方 Go SDK 时不依赖来源不明 client，不为本期维护完整 SDK；只实现 allowlist 方法。

## 11. 实现顺序与验证映射

保持一个 Delivery Batch/主要 PR。以下为提交边界，不机械拆 PR；只有后续获准实施才执行：

1. Store Walmart US variant、加密 OAuth/真实 merchant 核实、当前 HTTP/BFF/店铺授权消费者；缺配置场景真实关闭。
2. 商品只读观察与原页面、确定性 schema/图片/两种 listing mode、Target 草稿和显式批准；原 CompletionDraft 仍独立可用。
3. 原 Operation/worker/Submission 的 Feed wire、SKU 占用、accepted receipt、结果读取和原页面分层显示；覆盖 failed-SKU 修复与 UNKNOWN。
4. 必要开发自检与最终独立交付检查；实际应用/店铺/运行环境另获授权后联调。

| Must | 范围匹配的验证 |
| --- | --- |
| M1 | 跨企业/Store/callback/state/app/environment 混淆；partnerId 不符；双回调/refresh 并发/迟到结果/撤权；密文与 API 无 token 回显 |
| M2 | 原 ProductsReader 全页/部分/空店/cursor 失效、撤权、缺字段；Store 商品页面真实 DTO 与计数 |
| M3 | 固定版本/CAS/历史回读；schema draft-07 的 required/enum/if/then；207/缺ref；匹配/新建类型隔离；多 SKU/图片改变后须再批准 |
| M4 | 同键不同 wire；双 worker 仅一次发送；claim/发送/commit 响应丢失；accept receipt/UoW/同 SKU 不同 Product；pending Feed 与修复成功 SKU 的拒绝 |
| M5 | Feed accepted/PROCESSED+SKU INPROGRESS、部分 DATA_ERROR、成功入库但 UNPUBLISHED、缺页/重复 SKU/慢读取覆盖；UNKNOWN 不产生重试入口 |
| M6 | 正常 Console→BFF→原 Operation→worker→回执路径，未配置应用与已有平台 sibling 不回归；不止 adapter 单元测试 |

业务规则实施使用现有 TDD 与隔离 PostgreSQL/UoW测试；复用现有前端/HTTP测试与必需 CI，不开发验收 runner 或专项故障平台。文档阶段只执行既有索引/链接/差异检查。fixture PASS 不等于实际 Walmart；开发者不是最终产品验收者。

## 12. 安装、回退、外部依赖与停止点

按全新安装/空业务数据设计，显式初始化新增表/约束/角色；不新增旧数据迁移、backfill/ID映射、fallback 或双事实源。现有保留环境更新不在本设计授权内。关闭 `walmartUS` 可停止新外部操作，原 grants、SKU 占用、receipts/UNKNOWN 与历史观察保留；不删除真实商家/远端商品或对象，不把停用等同 destroy。

Legacy decision: **EXTRACT | RETIRE**。

- Reusable behavior：当前 Store、Product/Asset、CompletionDraft、Target、Submission、observations、Console/BFF 的有效授权/版本/幂等行为。
- Current owner：上表现有 owners 与新 Walmart 叶子。
- Cutover/deletion condition：只扩当前 consumers；root ListingKit、旧 Task/Workflow、compatibility/tenantbridge 与独立平台 Workbench 不重新进入依赖。未接触的历史代码不由本批扩大清理。

由用户取得服务商/主体/应用/scopes/商家、适用合同/保险/安全评估、部署与跨境处理批准。它们不阻塞文档或独立设计检查；缺失时真实授权/平台读取/刊登/客户开放为 NOT_RUN。官方字段/新错误码的实施测试不足只阻止相应正式能力，不伪造实例。若真实证据证明必须改变唯一 owner/事务/授权/UNKNOWN 并命中 AGENTS.md BLOCKER，才重开设计；非 Blocker 归 IMPLEMENTATION_TEST/BACKLOG，不追加通用系统。

本阶段停止于：独立检查后的 Design Basis、官方证据与 Issue/PR 可评审；不实现生产业务、不合并、不部署、不注册/接受协议、不读写真实商家数据、不代签用户验收。

## 13. 独立检查与 finding 处置

候选形成后执行一次独立只读架构检查，绑定实际提交；按 AGENTS.md 先分类，正常最多两轮。当前：PENDING。准入状态与实际评审证据在 #733/主要 PR 记录；尚未发生的检查不写 PASS。
