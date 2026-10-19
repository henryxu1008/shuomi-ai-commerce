# SHEIN 美国站：审核、首次上架与状态回读最小闭环

Refs [#605](https://github.com/qq550723504/task-processor/issues/605)。

> 2026-10-19 后续实施授权：用户已明确批准原 owner 按本冻结设计实施、落实四组必要验证并维护同一 PR #821。以下「本轮只设计/实施未授权」是冻结时的历史停止点，已由该最新阶段决定替代；不改变冻结的业务合同。合并、部署、真实店铺、现有运行实例和数据操作仍未授权。

- Product Decision: **PD-SHEIN-PUBLICATION-CLOSURE-2026-10-19**，用户先批准设计、Figma和独立评审，冻结后进一步明确授权按本方案实施并落实四组验证、维护同一 PR #821；不合并、部署或操作真实店铺。
- Design Basis: **Independent Architecture**，原因是新增官方上架副作用、执行合同与恢复准入。
- Status: **IMPLEMENTATION_READY / FROZEN**；独立评审结论与最终准入见 §13，后续实施已获授权，业务合同保持冻结。
- Investigation baseline: `origin/main @ e2f4a2f8b20ffce7530e6962038b9b04a0e76348`。
- Design branch: `codex/shein-shelf-closure-design`；本文件为设计依据，签发实施准入不等于获得实施、合并、部署或真实店铺授权。

## 1. 用户结果、范围与产品决定

已有企业运营人员在「供应市场 → 我的供应链 → 我的选品/自有商品 → 已上传 → 查看商品」中，能够区分上传受理、原版本审核、人工上架请求及平台当前在架/库存状态，并在允许的经营模式下明确确认首次上架。沿用当前五个阶段，不新增发布中心、工作台或另一套任务流。

Must：美国站 `shein-us`；绑定现有官方首次上传成功回执；逐 SKC 审核；自运营和半托管的有界首次人工上架；全托管只读；逐 SKU 库存证据；重复提交/响应丢失/重启不盲目再发；当前企业、店铺和服务授权。

Should：尽量展示平台原文失败原因和观测时间；分项展示部分查询失败；复用现有界面、授权、签名客户端、Submission Kernel 和数据库事务。

Out of Scope：重复上下架循环、下架、编辑重发、自动改价/改库存、采购/备货/质检/物流、保证消费者结账、自动纠正平台状态、后台轮询/cron/webhook、新审计平台、新 IAM、旧数据迁移、旧 Service 包装、真实店铺试验及运行实例变更。

Accepted boundary：平台查询不是原子跨接口快照；平台可能最终一致，外部控制台也能改变在架状态。显示各项来源/时间与未知，不把当前在架倒推为原请求成功，不为这一边界建设通用对账系统。

本决定是对 [完整供应链设计](my-supply-chain-shein-v1.md) 的有界增量。2026-10-12 未绑定店铺的补全草稿及已有交付继续有效；本轮实施来源于用户后续明确授权，不从历史阶段推定运行授权。

## 2. Product / Figma Authority

沿用 [最终 UI / IA](../product/final-ui-ia-authority.md) 和 [Greenfield baseline](../product/greenfield-no-legacy-migration.md)。唯一文件 `tg48P46SSXl6TBy9lZwg63`，当前页面 `31:463`；入口为浅色 `2738:655`、深色 `2738:1247`，商品资料详情为 `5543:429` / `5543:4285`。资料快照继续只读，新增「平台状态」及独立确认，不把查看资料变成隐式上架。

本轮新增节点如下。原资料详情已原位加入「平台状态」入口；既有「查看商品」连线保持。模式示例导航只用于原型展示，不是运行时允许用户切换店铺模式的控件；正式产品从 Store/原回执读取实际模式，不呈现示例切换或「成功回读示例」。原型统一标明「设计示例」，不得将示例 SPU、SKU、店铺、日期或库存当作真实事实。

| 设计状态 | 浅色节点 | 深色节点 |
| --- | --- | --- |
| 自运营，审核通过/等待确认 | [7232:2272](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=7232-2272) | `7232:2491` |
| 半托管，未来定时等待平台 | `7232:2311` | `7232:2530` |
| 全托管，只读等待平台 | `7232:2346` | `7232:2565` |
| 当前已在架、原请求 UNKNOWN | `7232:2381` | `7232:2600` |
| 请求成功与当前库存分别回读 | `7232:2416` | `7232:2635` |
| 首次上架确认 | `7232:2451` | `7232:2670` |

审核驳回、原版本缺失与部分查询失败说明板为 `7232:2710`，局部复用组件为 `7232:2267`。交互为原资料详情 → 平台状态 → 首次确认 → UNKNOWN 示例；成功回读为独立示例，不假设真实请求一定成功。半托管人工/已到期的确认复用同一摘要与确认行为，必须显示实际模式和原计划，不能借用自运营示例值。非默认状态的查询按钮为静态展示，不声称原型执行了 HTTP。

界面术语：上传回执显示「已提交，等待平台审核」；审核显示原版本及逐 SKC 结果；上架请求显示「请求已受理，等待回读」；实时平台状态显示「当前已上架 / 当前未上架 / 状态未知」；库存另列「有可用库存 / 无可用库存 / 库存未知」。没有「真实可售=true」的无条件断言。

## 3. 三种经营模式

| 模式 | 原上传计划 | 审核通过后的操作 | 未满足条件时 |
| --- | --- | --- | --- |
| 自运营 | 原始提交回执/资料 | 精确 SKC 原版本审核通过、当前未在架、授权及冷却满足后，用户确认「上架到美国站」 | 审核待定/驳回/未知只查询；已在架只回读 |
| 半托管 | 原 `shelf_way=1` 自动 / `2` 定时及 `hope_on_sale_date` | 默认等待原计划；原自动计划或定时时间已到、当前仍未在架且满足全部前置时，才允许用户另行明确确认人工请求；平台仍可能按商家/商品政策拒绝 | 不发明第三种「人工 shelf_way」；未来定时/未知计划关闭人工入口，平台禁止自主上架时不绕过 |
| 全托管 | 保留原 `shelf_way`、`hope_on_sale_date`、`shelf_require` | 不调用人工上下架接口、不显示人工上架按钮；只读平台审核/在架/SKU/库存 | 显示「等待平台安排」及已提交计划；未返回原因时不编造缺货/质检/未入仓原因 |

Store 的当前经营模式与原回执模式不一致时，保留原事实并停止 mutation，提示重新核实店铺；不能用 UI 切换模式解锁。半托管时间按官方北京时间语义解析为明确时刻，展示用户时区及原时区；格式/时区缺失视为未知。[当前发布合同](https://open.sheincorp.com/documents/apidoc/detail/3001707-1000001) 只定义以上两种 shelf_way。时间已到只解除本地计划限制，不证明平台已允许自主上架；尊重平台失败结果。

## 4. 官方合同与保守解释

公开官方文档只作接口合同；真实应用权限、品类开放、店铺模式和库存归属需后续获准环境验证，设计不证明真实店铺可用。

| 能力 | 官方接口/文档 | 本方案消费边界 |
| --- | --- | --- |
| 原版本审核 | `/open-api/goods/query-document-state`，[3000727](https://open.sheincorp.com/zh/documents/apidoc/detail/3000727-1000001) | 精确 `spuName + version`，单次最多 10 个 SPU；逐 `skcList` 保留 documentSn、documentState、failedReason 的语言/原文 |
| 人工首次上架 | `/open-api/goods/modify-skc-shelf`，[2000097](https://open.sheincorp.com/documents/apidoc/detail/2000097-2000002) | 仅支持该能力的自运营/半托管；每次一个 SKC、一个美国站；同 SKC/site 操作须超过 20 分钟 |
| 平台当前商品 | `/open-api/goods/spu-info`，[3001811](https://open.sheincorp.com/zh/documents/apidoc/detail/3001811) | 仅审核通过商品可查；按 `shelfStatusInfoList.siteAbbr` 匹配美国站，shelfStatus 0 下架/1 在架；SKU mallState 1 在售/2 停售；不含库存 |
| 库存查询 | `/open-api/stock/stock-query`，[3001695](https://open.sheincorp.com/zh/documents/apidoc/detail/3001695) | 原回执平台 SKU，最多 100 个一批；使用 invType VI/PI/JI，不继续依赖将退役的 warehouseType 入参 |

审核 documentState：`-1` 受理失败、`1` 待审核、`2` 通过、`3` 驳回、`4` 撤回、`5` 申诉。未知值保留 raw 并投影 UNKNOWN，绝不沿用旧程序把 3/5 当成功的规则。

上架 body 固定为 `skc_site_info_list:[{shelf_state:1,site_list:["shein-us"],skc_name:<server-owned>}]`。不支持客户端任意 SKC/site/body。官方虽支持批量，本期故意每个本地 attempt 只有一个外部目标，以便明确成功/失败/未知。

成功响应必须同时满足业务成功码、`total_count=1`、`success_count=1`、`failure_count=0`、无 failure_results，身份及响应结构可匹配；仅 HTTP 200 / code=0 不足。矛盾计数、错误身份、结构无法解释进入 UNKNOWN。failure_results 的 identity、code、msg、line_index 保留为有界证据，不把部分失败折叠成成功。

库存使用 `totalUsableInventory` / 精确仓库的 `usableInventory`；不以 totalInventoryQuantity、在途或提交库存代替。VI 是商家虚拟库存；PI/JI 是不同履约库存类型，不相加。允许按原回执模式及既有库存/仓库合同选择适用 invType；无法证明 US 站点/仓库适用时只显示该查询库存及归属未知，不参与美国站有货判定。

全部批次/页完整返回并且 SKU、SKC、SPU、库存类型和所需站点/仓库范围匹配才得出有/无库存。遗漏、重复冲突、超限截断、部分失败均逐项 UNKNOWN；零库存必须有显式成功的零值证据。对当前开放文档没有分页字段的返回不得编造 page 参数；一旦实际合同出现分页，遍历到明确结束，否则按不完整处理。

## 5. 现有 owner、缺口与最小调用链

| 事实/规则 | owner 与当前证据 | 本轮设计增量 |
| --- | --- | --- |
| 店铺、当前成员权限、连接、服务/凭据 | `internal/app/storecenter/product_access.go`、`internal/storecenter/product_execution.go` | 新增精确只读状态/上架目的，复用当前权限和短时 MerchantHandle；每次调用重新授权，不新增 IAM |
| 原资料、上传版本、SPU/SKC/SKU、计划 | Listing Record + 原官方上传成功 receipt | 不可变引用；不覆盖原回执，不从当前草稿猜标识/版本 |
| 审核/模式/库存解释 | `internal/marketplace/shein` | 纯规则 DTO/validator；官方文档规则替换冲突的旧规则 |
| 签名 HTTP / 原始响应 | `internal/integration/shein` | 沿 OfficialClient 窄 adapter 新增四接口；秘密留在 Store 执行边界 |
| intent/attempt/permit/UNKNOWN/fence | Listing Submission + official persistence | 新增 shelf_on kind 与窄 PrepareShelfIntent；沿原 Kernel，不增加第二执行状态机 |
| 页面组合和投影 | `internal/app/supplychain`、Console/BFF | 当前详情的状态读取/确认入口，原上传阶段不重写 |

现有 `supplychain/upload.go` 成功文案是「已上传，等待平台审核」，但现有 publication status 分派没有 SHEIN 完整回读。现有 official kind 只准 publish/image/global_create；新上架不能伪装成 publish 或另建旧 Service。

**必须修正的合同缺口**：`OfficialRepository.PrepareOfficial` 会核对当前 TargetRecord head。原商品上架应绑定原成功上传 receipt 对应不可变 record，即使本地后来有新草稿也不应被替换或被 head 校验误阻止。因此新设同 owner 的 `PrepareShelfIntent(originalReceiptID, remoteSKC, site, expectedObservationDigest, confirmation)` 窄入口，只读取并验证原成功事实；原 create 准备入口的 head 不变量继续保留。

合同 → 实现 → 注入 → 消费：Console/BFF 的当前 Supply 详情 → App Supply 用例 → Store Access(状态读取/上架) + 原回执读取 Port → SHEIN 纯 policy → Integration 官方 adapter → Submission Kernel/official repository；由现有 `internal/app/httpapi/supply_chain_application.go` composition 注入，不能仅写 adapter 未接 consumer。

读接口返回三个独立 observation：原版本 audit、当前 SHEIN US shelf/SKU、qualified stock；各有 source、observedAt、complete/error。查询不修改原上传 receipt 或 execution 状态。共享店铺同步缓存不能代替本次原回执的精确查询，也不能消费 Supply 的 UNKNOWN fence。

## 6. 输入、授权与身份

客户端只能提交当前本地 receipt ID、选中的 receipt 内 SKC、确认 token/operation key；企业来自验证会话。服务端校验 receipt 属于当前企业/Store/平台/SHEIN US、原上传成功、原不可变 record/hash/mode/版本及远端标识完整。source receipt ID 不能成为新的跨租户查询入口；拒绝任意 SPU/SKU/站点或第三方 URL。

状态读取要求既有当前店铺读权限和对应官方服务；上架要求既有当前店铺提交权限、有效服务/连接、可用官方 scope。延续现有短时 MerchantHandle 与 mutationSent one-call 约束；准备、获取 permit 后发送前都验证当前授权。等待确认期间撤权/换店/换应用/模式变化，确认失效，无发送。上架作用于相同商家身份，不允许因重新连接同一商家绕过目标 fence。

Store 执行目的明确区分拟新增的 `product_status`（既有读权限，CanWrite=false）和 `product_shelf_on`（既有提交/写权限，CanWrite=true）。`internal/app/supplychain/authorization.go` 当前以非 Rules 目的推导 CanWrite，实施必须改为显式目的枚举映射，未知目的拒绝；不能将新增只读目的误变写权。read adapter 的 POST 查询仍属读操作，不消费 mutation permit；只有固定上架方法消费唯一 mutationSent。

确认摘要显示企业、店铺、模式、美国站、SKC、原 SPU/version、原计划和动作；token 绑定这些事实及预检摘要，5 分钟有效。服务器消费 token 时再次读取审计和在架状态；相关业务事实变化要求重新查询确认。摘要只包含精确身份、原计划、审核版本/结果、在架状态，不包含每次变化的 observedAt/traceId，避免重复只读查询误令确认失效。原计划未来定时、审核未通过/缺失、当前在架、权限未知都不能获得发送准入。读到已在架时直接展示当前事实并不创建发送。

## 7. 幂等与首次上架身份

target = 现有 `OrganizationID + Platform=shein + StoreID + SubjectID`。SubjectID 为版本化独立命名空间 `shein-shelf-on-us-v1:<digest(remoteSKC, site)>`（按现有 bounded identifier 编码）；action=`shelf_on`。不把 operation key、原 record/revision、连接 ID、应用 ID 或审计版本放进 target，以免同一远端目标可绕过 UNKNOWN。原 publish target 与 shelf target 明确分开。

intent key 是一次用户确认的稳定身份；payload hash 包含原 receipt/record/hash、商家绑定、SKC/site、模式/计划、审核版本、确认摘要及固定 body。同键同载荷回放已存结果；同键异载荷 conflict。并发不同键操作同 target 由现有原子 target fence 互斥。payload/确认事实从服务端构造。

本期只支持首次上架：成功 shelf target 永久保留原成功回执，不因后续外部下架、刷新、换 key 或换连接再次开放。需再次上架/下架属于后续产品决定。已在架的只读短路不伪造本系统上架成功回执。

## 8. 持久化、事务与恢复

只扩展现有 official intents/receipts 的 kind 约束为 shelf_on、窄 owner 合同及现有 Submission target 成功策略；不新增 observation 表/定时器/通用恢复平台。正式 schema 初始化/grants/readiness 在后续实施中显式更新；不在 serving startup 自动迁移，本轮不执行任何 schema。

沿现有提交边界使用同一 Product/Listing 数据库的**两段事务**，不把 Kernel Acquire 塞入外层未提交事务：T1 的 `PrepareShelfIntent` 验证原回执/不可变 record/确认摘要并提交 durable shelf intent，**不产生发送权**；T2 在现有 Acquire 的目标锁/事务内核对该 durable intent/hash、稳定远端 target、既有成功/UNKNOWN fence及历史操作冷却，原子取得 attempt/claim。只有 T2 成功提交后 Kernel 才发出唯一 SendPermit。Acquire 提交结果不明时不发 permit，只按原键核实；准备成功而未取得 attempt 可重新取得，但不能直接发送。

T3 复用 `CompleteOfficial` + `NewTransactionFinalizer`：完整成功响应的 shelf success receipt 与 attempt succeeded 原子提交。现有 transaction finalizer 只做完成/核实，明确不能 Acquire。原上传 receipt 永不改写；实施在原 repository/Kernel 的窄 shelf 分支实现以上校验，不另建事务管理器、Saga或第二状态机。

失败证据保存在现有 attempt 的 EvidenceProviderResponse；**不写永久唯一 success receipt**，防止明确无效果失败占住成功唯一键。只有经过官方合同/确定性 fixture 资格认定的完整 no-effect 业务错误才能 failed_definitive；默认错误映射是 UNKNOWN。初版 no-effect allowlist 可以为空，不能为提高成功率猜错误码。发送前本地拒绝可安全取消且无外部效果。

| 失败位置 | durable 结果与后续动作 |
| --- | --- |
| 准备事务前/事务回滚 | 无 permit，不发送；同键重读准备 |
| 准备提交响应丢失 | 按同键重读 durable intent/attempt；不能新 key 重发 |
| claim 后、发送前重启/取消/租约过期 | 沿 Kernel 转 UNKNOWN；不把未观测到发送当未发送证据 |
| 已发送、超时/断连/取消/不可解释响应 | UNKNOWN，原 target fence 保持，不重发 |
| 明确完整成功响应、原子落库成功 | succeeded + receipt；随后读平台当前状态 |
| 平台成功、落库失败/提交结果丢失 | 原 attempt UNKNOWN/待核实；按原键读落库结果或带关联证据恢复，不能重新发送 |
| 完整且获资格认定的 no-effect 响应 | failed_definitive；保留错误/操作时间；新的显式确认仍需重新预检与冷却 |
| 进程退出/刷新页面/客户端取消 | 不删除 durable attempt，不释放 UNKNOWN，不消费第二张 permit |

冷却从本系统最近一次发送边界的**已持久化保守上界**记起，同目标必须**超过** 20 分钟（等于不准入）：有明确响应时可用其 FinishedAt，无明确完成证据时取原 LeaseExpiresAt 或更晚可证实上界；claim起点可能早于实际HTTP发送，不能单独用它提前解锁。明确失败也不能绕过平台操作间隔。时钟用服务端 UTC，平台时间仅做证据。本系统无法覆盖卖家后台外部操作，平台拒绝如实展示；不承诺跨外部操作者的全局节流。20 分钟经过仅解除冷却，不解除 UNKNOWN，也不是 retry 授权。

恢复唯一 owner 是 Submission。现有 Kernel 有 provider readback/manual resolution 合同，但当前 Supply 只接入封印的 publish readback，**没有注入 manual resolution authorizer**。本期不新增人工解除/强制重试 API，也不声称现有 Supply 已能人工解锁 shelf。新增的 shelf 只读状态查询不能自动完成原 attempt；仅在原执行保存链能恢复确切关联响应时，才通过同 owner 的封印 completion/readback 证据完成原 intent。

平台当前在架只能证明当前状态，不能在缺少因果关联时将原 UNKNOWN 写成 succeeded。可显示「当前已上架，原请求结果待核实」，继续保留 fence；没有确切关联证据时本期就保留 UNKNOWN，不要求用户执行未开放的人工作业。未来若要开放人工 resolution，需要另行有界产品授权并注入现有 authorizer，不能由本期顺手开放。单纯“没查到/还是下架/已过20分钟”不能清除 UNKNOWN。

## 9. 实时投影与库存判定

每次显式「查询平台结果」执行有界读取；人工上架成功落库后自动读取一次。没有后台轮询。各分项成功独立显示，失败不覆盖已有 durable 事实，旧 observation 必须显示旧时间/未刷新。

| 当前证据 | 用户文案/可执行动作 |
| --- | --- |
| 原版本待审/驳回/撤回/申诉/缺失 | 展示原文及版本；不准人工上架；读不到 spu-info 不能替代审核结果 |
| 审核通过、US shelf=0 | 当前未上架；按模式、计划、授权及 fence 决定能否确认 |
| US shelf=1、某 SKU mallState=1、适用 US 库存明确 >0 | 该 SKU「平台在售，有可用库存」；不保证结账 |
| US shelf=1、SKU 停售或适用库存为0 | 「平台已上架；SKU 停售/无可用库存」 |
| US shelf=1、SKU/库存缺失或归属不明 | 「平台已上架；可售条件待核实」 |
| 平台站点缺失/查询异常/未知枚举 | 对应字段未知；保留成功分项与 observedAt |
| 原上架 attempt UNKNOWN、当前已在架 | 当前已上架，原请求结果待核实；仅查询，不再发送 |

结果按 SKC/SKU 展示，聚合为「部分在架/部分有货」时同时显示逐项依据，不用任一 SKU 有库存代表整件全部可售。已提交量、上次上传库存和本地草稿不是平台当前库存。

原回执缺 version/远端 SKU 时，显示具体无法核实的分项并关闭依赖它的操作；不得使用“最新 SPU 审核”或同货号搜索填补。新首次上传实施必须验证成功 receipt 已保存完整版本和 SKU 关联；如真实响应不提供，必须按官方明确同次任务关联补齐，未闭合前不开放上架。无需旧数据迁移或旧 ID 映射。

## 10. 外部调用与资源边界

复用当前签名 transport、凭据遮蔽、Context deadline 与响应大小限制；每个外部调用使用有界 timeout，App 整体查询预算 30 秒，读接口最多一次短暂传输重试且受整体 deadline 约束，mutation 禁止 transport 自动重试/重定向重放。审核每批≤10，库存每批≤100；单次详情原回执最多100 SKU，超出时按明确续查的只读分批返回 complete=false，不静默截断或无限并发。写 body 远小于现有2MiB Kernel上限。上架单次 HTTP timeout 8秒，符合现有短时 MerchantHandle有效期；授权过期/实际 deadline 更短时使用更短预算。

平台限流、缺 scope、服务到期、连接撤销只影响本项可用性，不用旧 Cookie 通道 fallback。日志与证据屏蔽 token/secret/signature，失败原文限制长度，保留结构/摘要/hash 与原请求关联；前端不获得密钥或任意 provider 请求能力。

## 11. Legacy decision 与复用规则

Legacy decision: **EXTRACT | RETIRE**。

- Reusable behavior：准确的平台ID匹配、原始错误/多语言原因保留、合法的模式与上架计划字段解释、当前有效的必填/尺寸/图片等确定性校验；抽取前逐条对照当前官方合同。
- Current owner：SHEIN Marketplace policy、Listing immutable receipt、Store Access、Integration adapter、Submission Kernel、Supply App/Console。
- RETIRE：Cookie/网页私有接口、旧任务/工作台/DTO、DefaultConfirmed、库存存在即上传确认、SPU不匹配时用最近一条、documentState 3/5当成功、敏感操作自动重试；不得 wrapper/fallback/双写/第二事实源。
- 旧重量≤0补0.01、供货价乘0.85等猜值不成为真实产品事实或本轮准入规则。
- Cutover/deletion condition：当前官方 consumer切换并有准确业务测试后移除受影响旧路径；本轮只确定设计，不恢复已退休能力，也不删除共享旧代码。

## 12. 验证与后续实施交接

以下是必要实现验证，不是本轮执行结果，也不要求建设新验收工具。复用 Go 现有 test/HTTP fixture/PostgreSQL transaction tests，按业务规则 TDD：

1. 精确原 receipt/version/SKC/SKU、跨企业/Store拒绝；本地新草稿不改变原目标；模式漂移/撤权/确认过期在发送前拒绝。
2. 三模式/未来定时/审核各枚举/缺失值；全托管与未来定时半托管无手动发送；已在架只读。
3. 相同键回放/异载荷冲突、并发不同键、不同原record/连接不能绕 fence；单 SKC/site body与单次调用。
4. 以持久化保守发送上界测试20分钟严格边界，claim早于HTTP不能提前解锁；UNKNOWN经过冷却仍禁止；T1无发送权、T2提交不明无permit，T3成功receipt+attempt原子性、失败不占成功唯一约束；超时/取消/提交响应丢失/重启不重复发送。
5. code0但失败/矛盾计数不能成功；未知错误码不归 no-effect；readback在架但无因果关联保留UNKNOWN。
6. US shelf、mallState、库存类型/范围、显式0、缺SKU/部分失败/超限、PI/JI不相加；前端只展示证据支持文案。
7. composition/API/Console真实consumer形成一条fixture用户路径；新增status目的只有读权、shelf目的须提交权、未知目的拒绝；没有manual authorizer时无人工解除入口，现有未绑定草稿与上传语义不回归。普通文档检查不能替代实现验证。

后续另获实施授权后，同一 Delivery Batch/主要 PR 连续完成；最终交付保留「实现完成，待独立验证/用户试用」。真实官方 scope、模式/品类和库存映射、店铺路径均为 **NOT_RUN**，须按用户另行授权核实；CI/AI评审/Figma均不等于店铺或产品验收。

## 13. 独立评审、冻结与准入

当前 **IMPLEMENTATION_READY / FROZEN**。评审仅针对本增量的 Must、owner、上架副作用、权限、幂等、事务、UNKNOWN/recovery及UI语义，不重新打开无关历史架构。Reviewer按AGENTS分类 finding：BLOCKER / IMPLEMENTATION_TEST / BACKLOG / ACCEPTED_RISK / NOT_APPLICABLE，并指出具体用户后果。最多两轮正常评审；只在真实BLOCKER时重开已冻结边界。

2026-10-19 独立 Reviewer `/root/shein_architecture_review` 对原稿 `4d53bfdaa` 完成 R1，并对全部对应澄清增量 `6fa88c5fbb14bb28defd997c1e5d23d30d6e653a` 定向复核，正式结论为 **IMPLEMENTATION_READY / FROZEN，无剩余 BLOCKER**。证据是直接只读核对实际 official repository、ExecutionKernel、Store授权、Supply接线和 §2 Figma三模式/确认/UNKNOWN/失败节点，不只是作者报告。准入后的本次状态元数据变更不冒充新的代码执行证据。

达到Ready也仍为**设计交付；实施NOT_AUTHORIZED；合并/部署NOT_AUTHORIZED；真实店铺NOT_AUTHORIZED**。实施测试和真实店铺试用没有执行，不能以准入、Figma或未来CI代替用户验收。

R1 独立 Reviewer `/root/shein_architecture_review` 已直接核对当前 official repository、Kernel、Store权限及 Figma 主要状态。以下均分类为 **IMPLEMENTATION_TEST**，无新增平台或设计BLOCKER；说明先分类后修正，并只做对应增量复核：

| Finding / 受影响 Must / 后果 | 处置与验证归属 |
| --- | --- |
| 准备与Acquire当前独立提交，finalizer不能Acquire；含糊外层事务可能提前发送 | §8明确T1/T2/T3，提交后permit及成功原子完成；§12实现验证，阻塞未满足的实现切片合并 |
| Supply没有manual authorizer；声称已有人工恢复入口会令运营无法完成核实 | §8准确区分Kernel合同与Supply接线，本期不开放人工解锁，无因果证据保持UNKNOWN；§12负向入口测试 |
| claim早于实际发送，可能提前解除严格冷却 | §8使用持久化FinishedAt/LeaseExpiresAt等保守上界，§12边界测试 |
| 当前非Rules目的推导CanWrite；新增status可能误获写权；原receipt与head/版本SKU完整性需覆盖 | §6显式只读/写目的映射，§5窄原receipt入口、§9/12精确关联及权限测试；不得等同已测通过 |

半托管 shelf_way 仅1/2的官方澄清已纳入 §3（未增加第三种计划）。设计准入不豁免以上实现 Must；真实接口/店铺验证仍为NOT_RUN。
