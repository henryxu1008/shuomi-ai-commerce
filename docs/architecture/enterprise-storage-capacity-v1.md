# 企业存储容量 V1 设计草案

Issue: [#659](https://github.com/qq550723504/task-processor/issues/659)

Design Basis: **Independent Architecture**

Admission Status: **NOT_READY**
产品参数：**待用户决定**。本稿不批准正式价格、schema、开发、部署或真实数据操作。

企业管理员在现有“资源与额度”查看企业共享存储的上限、已占用、写入预留和剩余空间，成员沿原商品素材、图片和知识库路径保存内容。空间不足时，在支付或写入前得到明确提示；管理员可以从企业钱包购买有期限的扩容包。现有文件访问权限、资产批准、AI 点数、知识生命周期和资金 owner 保持各自职责。

## 1 产品依据与建议

用户于 2026-10-12 要求先设计企业存储容量模块，复用现有存储和商业能力，并对默认容量、计量范围、超限处理、扩容收费提出建议。**批准设计工作不等于批准本节参数或启动开发。**

| 决策 | 首版建议 | 原因 |
| --- | --- | --- |
| 基础容量 | 每个企业共享 **5 GiB**，无有效期、不按月清零 | 足以开始商品素材与知识文档试用；5 GiB 约能容纳 2,560 个 2 MiB 对象，实际还包含版本和其它素材 |
| 共享方式 | 按当前 Organization 汇总；成员、店铺不额外赠送，不分配成员存储余额 | 避免成员离开或店铺变化使企业资产失去容量归属；首版沿原领域权限控制写入 |
| 扩容商品 | 一种商品：**增加 50 GiB，¥19.90，30 天一期**；可购买多个单位 | 单一商品减少套餐、折扣和升级规则；整分价格可沿现有报价与钱包合同 |
| 计时 | 成功扣款并激活权益时开始，30 × 24 小时，UTC 半开区间 | 不从创建报价、付款预留或浏览器收到响应时计时 |
| 重复购买 | 每次购买都是立即生效的独立容量包，容量叠加，各自到期 | 不混淆“增加容量”与“延长时间”；首版只提供购买，不提供容易误解的续期按钮 |
| 超限 | 阻止新增占用；已有读取、下载和不产生新对象的操作继续按原权限开放 | 不以到期或容量不足破坏已有商品、文档和生成结果 |
| 预警 | 页面在占用加预留达到 80% 时提示，达到 100% 时提示无剩余空间 | 仅在现有页面显示，不新增短信、邮件或通知调度系统 |
| 到期与删除 | 到期移除相应容量权益；**不自动删除、归档或冻结已有文件读取** | 用户数据安全优先；长期超限保留会产生平台成本，需用户确认该首版政策 |
| 其它费用 | 首版不另外向用户收请求或流量费，不自动按超量从钱包扣款 | 本商品只销售容量与时长；不承诺无限公网分发，不新增流量计费机制 |

所有界面明确使用 **GiB**，1 GiB = 1,073,741,824 bytes。内部存整数 bytes，API 用十进制字符串；价格用整数分，1990 分/包/期，禁用浮点权威计算。基础容量来自经批准且带版本的产品策略，不来自旧订阅套餐，不给用户开放任意赠送接口；策略缺失返回 unavailable，不隐式无限容量。首版策略冻结，后续调整默认容量须另作产品决定，不能通过配置覆盖既有承诺。

### 1.1 价格与成本

¥19.90 是建议零售价，不是供应商报价或已确认利润。阿里云官方[计费概述](https://help.aliyun.com/zh/oss/billing-overview)使用标准存储 0.12 元/GB/月说明小时结算；以其单位举例，50 GB 持有 30 天的纯存储成本约为 6 元。实际费用还包含请求、[公网下载和 CDN 回源流量](https://help.aliyun.com/zh/oss/traffic-fees)、地区、冗余和折扣差异。这只是成本参照，不能当作本项目的实际 COS/AWS 账单或毛利证明。

当前 `internal/integration/s3` 使用 AWS SDK v2，严格对象能力为 aws/cos 两种模式；“OSS”在本产品中表示对象存储，不代表本轮新增阿里云适配。首版继续使用已有部署选定的供应商。建议价适合先提供一个容易理解的容量商品；正式销售前核对部署地区和实际访问成本，沿 Commercial 安装已批准 offer，不由前端或容量 owner 写价格。5 GiB 基础容量、平台备份及到期超限保留均由平台承担成本。

### 1.2 当前 Must 与边界

设计必须覆盖企业隔离、原权限、并发容量限制、写入与扣款重放、UNKNOWN 保留、到期不删数据、真实资源展示。开发后应交付“查看容量 → 保存文件 → 不足提示 → 钱包扩容 → 回读权益 → 再保存”的可操作路径。

本轮只交设计。建议首版开发不包含新套餐等级、自动续费、按比例退款、存储包退款自助入口、成员存储分配、通用网盘、跨企业转移/去重、文件迁移、旧配额兼容、GC 平台、冷热分层、生命周期自动删除、流量商品、CDN 重构、OSS SDK 重写或验收工具建设。客户退款/争议不得调用旧订阅退款；没有容量订单已批准退款合同就不提供自动退款承诺。依法或渠道接受的已验证资金冲正继续由 Money 处理，不拒收、不伪造成容量清理。

尚未批准自动清理或流量硬限额；不能标为 Accepted Risk。到期不删和不单收流量的成本后果由本节明确交用户决定，不由 Reviewer 自行扩大为新存储/流量平台。

## 2 当前能力与产品位置

| 当前能力 | 复用点 | 容量缺口 |
| --- | --- | --- |
| `internal/integration/s3/uploader.go` | PutImmutable、InspectObject、ReadObject、CopyImmutable；原 AWS/COS 不可覆盖策略 | 不拥有企业占用或容量权益；不能从上传次数推断占用 |
| `internal/app/productsourcing/source_media.go` | 原实时企业授权、内容 hash、确定性 key、限长及读回核验 | 上传到对象存储前未预留企业容量 |
| `internal/knowledge/service.go`、`types.go` | 原 Revision、ADMITTED/OBJECT_STORED/处理恢复；单文件 10 MiB | 原资料停用不是物理删除，没有 destructive delete 用户合同 |
| `internal/imageagent/objectstore/durable.go` | 原 staging、recovery bundle、不可变结果发布和重放 | 需要在 paid dispatch 前预留正式结果空间；不能生成付费结果后才发现空间不足 |
| `internal/commercial/billing` | server-owned offer、immutable quote/order、wallet reserve/commit、原身份恢复 | 当前资源商品只支持店铺期数、AI 点数和数据条数，没有 bytes × 有效期合同 |
| `internal/ledger/money` | 原 CNY 钱包、预留决策、扣款回执、资金冲正与债务 | 不拥有容量或对象事实 |
| `internal/ledger/orgresource` | 可借鉴现有 PostgreSQL 事务、幂等与证据模式 | 不把有期限容量塞入永久消耗余额；既有三类资源合同不变 |
| 旧 `listingsubscription` 的 `oss_storage`、`storage_bytes_current` | 仅作为历史语义参考 | 不接回新基础方案，不保留旧套餐/fallback/双账本 |

产品依据为 [Final UI / IA Authority](../product/final-ui-ia-authority.md)、[统一基础与预付资源](unified-base-prepaid-resources-v1.md)、[Commercial Wallet/Billing](commercial-wallet-billing-contract.md)、[正式价格](commercial-retail-pricing-v1.md)、[Knowledge](agent-knowledge-context-v1.md) 与 [图片智能体](product-image-agent-v1.md)。

2026-10-12 已读取当前 Figma [资源与额度 1627:359](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=1627-359) 的结构与截图，以及 [企业资源摘要 1634:359](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=1634-359)。现稿只有期数、点数/原 Token 文案、数据三类卡片。建议在现有页面增加“企业存储”区域，不新增一级导航、成员分配列或独立网盘；当前 Figma 并未批准新区域、扩容对话框或本稿价格。

主入口保持 `/workbench/account/organization/resources`；“我的权益”只读复用同一容量投影。存储区域显示总容量、已占用、写入预留、剩余、观察时间、来源分类和扩容包到期日；管理员看到“购买扩容包”，其它成员只按原 read 权限查看汇总。没有文件清理合同的分类不显示虚假的“释放空间”，可以链接原素材/知识页面并说明停用仍占用空间。

扩容沿原 wallet/quote/order 交互，明确显示“新增 50 GiB × 数量，生效后 30 天；这会增加容量，各包分别到期”，展示原报价、实际扣款和权益状态。loading、denied、unavailable、UNKNOWN 不显示为 0 或成功。跨企业切换/迟到响应沿当前 user/org tuple 隔离和原 BFF 约束。

## 3 计量范围

**按企业实际保留的计量对象字节计一次，不按累计上传量、文件引用数量、商品条数、浏览次数或云账单金额计容量。** 稳定对象身份为 trusted namespace + immutable object key；namespace 配置绑定 provider/account/region/bucket 及允许的前缀，内部保存不可逆身份，不向浏览器暴露 bucket/key/凭据。

| 内容 | 企业容量 | 归属与释放规则 |
| --- | --- | --- |
| 手动商品素材、采集后真正保存的图片 | 计入 | 当前操作企业；外链尚未下载不占用 |
| AI 生成的可保留候选与正式结果 | 计入 | 原 Run 企业；是否采用不决定是否占用，同一最终 key 多次读取/批准只计一次 |
| 知识库上传原文档、实际仍保留的旧版本 | 计入 | Knowledge Revision 企业；解析失败、停用、被新版本替代仍占用 |
| 独立持久化的缩略图、派生图、导出文件 | 计入 | 每个真实对象各计一次；当前仅从数据库实时生成的下载不制造占用 |
| 已开放供货文件或用户业务附件 | 计入 | 其当前领域 owner 明确的资产所有企业；购买方只引用/读取不重复计量 |
| 外部平台 CDN 图片、第三方 provider 临时 URL | 不计入 | 只有通过原受控下载路径真正保存后才计入；不授权新的网络访问 |
| PostgreSQL 商品/报表/解析文本、日志、索引、审计、备份 | 不计入 | 沿各原 owner 与运维资源范围，不将数据库容量混为 OSS 容量 |
| 原 AI staging 与恢复 bundle、应用暂存、平台系统资料 | 不计入 | 属于既有恢复/工程成本，不把相同生成文件的工作副本重复卖给用户；不据此新增 TTL 或删除 |
| 认证材料、资金/服务不可变凭证等必须保留的证据对象 | 不计入 | 原证据 owner 与保留合同优先，不能因容量不足妨碍已发生副作用的证据落地 |

同企业同一物理 key 重放/多处引用只占一次。不同 key 即使内容 hash 相同仍按实际副本各计一次；不开展跨 key 或跨企业去重。同 key 不能改变 owner 企业、size/hash 或分类；冲突拒绝并保留原事实。业务对象名称、引用和可读性仍归原领域，容量登记不是第二个 Asset/Knowledge catalog。

非用户容量不代表没有成本。排除的 staging、备份、证据和公网访问继续由平台承担，并沿现有运行边界管理；本稿不授权新的清理或无限量服务。受控启用时，所有已启用业务对象写入必须注册为“计量并接入”或上述明确的“平台承担”；未分类的写入不能静默绕过。尚未配置/未开放的业务无需为容量模块提前实现。

首版只有已批准、原 owner 支持的物理删除才可释放；Knowledge 当前只有停用，SourceMedia 和图片批准也不提供安全 purge。**本批不新建这些领域的 destructive delete。** 因而某些分类首版只能通过扩容继续使用，页面必须如实说明。容量模块提供安全释放的内部合同，未来领域删除经独立准入后才能调用。

## 4 Owner 与持久化

建议在当前模块化单体中增加有界 `internal/ledger/storagecapacity`，只拥有容量权益、占用、预留与不可变操作回执。选择单独 owner 是因为“可使用的当前容量”与“永久可消费的点数/条数”合同不同；不建设通用资源框架，不引入微服务或新的数据库实例。

| 事实 | 唯一 owner | 建议持久化位置 |
| --- | --- | --- |
| 产品基础容量策略及销售商品/报价/订单 | Commercial/Billing | 原 commercial 数据库；冻结的版本化基础策略与原 offer/quote/order |
| CNY 资金、扣款/释放/冲正、债务 | Money | 原独立 Money 数据库与角色，不与容量池合并 |
| 容量 lot、已占用、写入预留、操作回执 | ledger/storagecapacity | 原 commercial 数据库的独立表，由本模块窄 repository 访问 |
| 业务文件归属、引用、批准、可读性、删除资格 | Product/Asset、Knowledge、Supply 等原 owner | 原业务数据库与当前权限；容量不得代替这些事实 |
| 对象存在、字节、hash、不可覆盖 | 当前 S3 集成与供应商 | 原 bucket/namespace；不每个企业动态开 bucket、不变更 trust/ACL |

首版最小事实包括：

- `storage_capacity_accounts`：org、已占用/预留 bytes、version。汇总与对象/操作在同一容量事务更新，可通过本模块回执核对，不是第二套异步统计。
- `storage_capacity_lots`：原 order/item、bytes、原产品合同指纹、PREPARED/ACTIVE、激活时间/到期时间、原 Money commit proof。唯一 `(org, order_item)`，到期按时间读取，不靠第二 scheduler 改状态。
- `storage_capacity_reservations`：org、原 owner 操作身份、输入/输出上界指纹、预留上界与剩余、dispatch/关闭 fence、version。已知文件身份可直接绑定；AI 预留以原 Run/revision/slot/attempt 为身份，生成前不要求最终对象 key。未绑定结果仍占预留。
- `storage_capacity_objects`：org、namespace/key 身份、hash、大小、分类、原 owner/operation、写入状态/预留和唯一 dispatch 标记。全局 namespace/key 唯一，跨 org 认领拒绝。
- `storage_capacity_receipts`：按原 scoped operation identity 保存不可变决策及 payload fingerprint，承载预留、确认、未写释放或已批准删除释放；不逐次读写日志建新审计平台。

优先复用 PostgreSQL/GORM、已有 transaction runner 的合格行为和唯一约束模式；不是重写数据库或支付客户端。Money 继续独立 pool。当前 commercial owner 角色只补本模块所需表权限，业务 adapter 通过注入端口调用，不直接读 Money 或其它业务表；沿已有运行权限检查，不扩大为全权 schema owner。

仓库已有 [OpenMeter 技术评估](openmeter-shadow-metering-poc-report.md)，其中明确普通 metering/entitlement 可复用，但访问判断不是原子并发 quota reservation。首版沿已有 PostgreSQL 原子预留模式控制写入，不为容量购买上线新的计量平台；未来若有实测分析需求，可消费本容量 owner 的投影，不能反过来把异步计量变成第二占用/扣款事实。

`quota(t) = base_policy.bytes + SUM(ACTIVE lot.bytes WHERE starts_at <= t < expires_at)`

`remaining(t) = MAX(0, quota(t) - occupied_bytes - reserved_bytes)`
`overage(t) = MAX(0, occupied_bytes + reserved_bytes - quota(t))`

购买和预留使用同一企业容量 account 行锁；新预留必须在取得锁之后读取可信当前时间（PostgreSQL clock_timestamp 或同等服务端时钟），不能沿用事务开始时间跨过排队期间的到期点。读 summary 在同一数据库 snapshot 使用一个可信 UTC 时间。lot 的到期和查询时间均使用数据库/服务端时间，不信客户端。报价不增加容量；PREPARED lot 不计入 quota。普通 GET 不结算或补写资源，缺少合法状态返回 unavailable，真正新安装空空间才返回真实 0。

到期可以合法导致占用加预留高于当前 quota；因此不把 `occupied + reserved <= quota` 设成永远成立的约束。必须在**每次新预留的原子准入点**满足上限。已成功获得的有限预留可完成原对象/作业，即使期间包到期；不能换键/扩大数量。到期不收回原预留、不清除占用。

## 5 写入与 UNKNOWN 合同

业务 owner 保留业务状态、权限和唯一重试责任；容量 owner 只判定与核实占用，S3 只执行原不可变 I/O。端口由服务端装配，不接受浏览器提供的 org、namespace、object key、大小证明或终态证明。

### 5.1 预留与确认

1. 沿原 HTTP/worker live 身份及领域权限读取 exact 原操作。已有持久化操作的 Knowledge/Image 绑定原 Revision 或 Run/revision/slot/attempt；SourceMedia 的同步上传按结构化 `(org, actor, source_media_upload, content_hash)` 派生稳定操作身份，其容量 intent 在 PUT 前持久化，不假设已有上传业务表。不能建立持久化容量 intent 的路径不得发生 PUT。
2. 读取有界 bytes、验证原文件/hash/MIME。已知大小使用精确 bytes；未来图片结果在 paid dispatch 前使用已批准配置的输出数量 × 每张上界，包含所有会保留的正式对象，不放宽已有文件/运行限制。
3. `ReserveWrite(org, owner_operation, input_fingerprint, output_bounds, known_objects?)` 在容量数据库事务锁企业 account，重查相同键/已知对象，按锁后当前 quota 预留新增上界，原子保存 reservation 和 receipt。AI 在 paid dispatch 前只提供原已持久化 Run/revision/slot/attempt、已冻结输入指纹、输出数量/字节上界；此时不存在结果 hash/key。已知文件可同时绑定精确对象；同键异载荷 conflict。
4. 生成后，可信原 owner 用原 reservation 调用 `BindObjects`，冻结结果 manifest、namespace/key/hash/size 与输出索引。绑定是原预留内的 materialization，不重新做新容量准入、不扩大数量/字节上界、不改变业务执行状态；各对象按固定身份顺序锁，namespace/key 唯一。相同已确认对象复用、不重复占用；错误 org 或同 key 异载荷拒绝。结果未绑定、provider UNKNOWN、部分结果 UNKNOWN 都继续占 reserved。业务 owner 以原操作身份/指纹找回绑定，不换键。
5. 在实际 PUT/COPY 前，`ClaimDispatch` 与 `AbortUnstarted` 在同一容量 reservation/object 锁和 CAS 下决定唯一赢家：关闭未写意图后拒绝旧 permit；认领发生后记录 dispatched，任何 TTL/404/普通业务失败都不得自动释放。外部 PUT 必须先取得原 dispatch 认领；容量侧不自行新增上传重试。
6. 沿原不可覆盖 S3 PUT/COPY。成功后由受限 trusted adapter 对原 identity 做 Inspect/hash/size 核实；不是只信上传响应或 caller 的成功布尔值。
7. `ConfirmStored` 在容量事务按唯一对象把实际 reserved 转 occupied；真实大小不得超过原绑定/输出上界。重复确认返原 receipt。只有原 owner 已关闭所有未生成/未写输出、排除迟到物化后，才能释放作业未使用的上界余量；不能首次绑定少量结果就释放整个上界。对象即使暂时未进入业务可读/批准状态也占用；解析失败/业务库后续写失败不能退已存在 bytes。
8. 原 owner 完成其既有业务终态。容量确认失败保留原预留，不能把已保存对象当作免费空间。多对象部分成功逐个确认，未确定部分继续预留。

AI 在派发前的容量预留与点数预留分别保持原 owner：容量失败不新增 provider POST 或点数扣款；点数准入失败以可信“未生成/未写入”终态释放空间。生成成功已有原扣点语义不变，不能因用户未采用或容量核实失败退成功模型费。容量预留成功不是 dispatch 授权，原 IAM、预算、价格与 paid effect fence 仍需满足。

### 5.2 对象状态与失败边界

| 当前状态/事件 | 容量效果 | 恢复入口 |
| --- | --- | --- |
| `RESERVED`，原 owner 确认从未认领 PUT/COPY 且关闭该 intent | 可转 `ABORTED`，释放原预留 | 原 owner 终态与容量 receipt；关闭后拒绝任何迟到 dispatch |
| `PUT_DISPATCHED`，网络超时、响应丢失或崩溃 | 保留 reserved，显示待核实 | 原 owner 的有界恢复，核 exact key/hash/size |
| Inspect 确认原对象匹配 | 转 `STORED`，reserved → occupied | 原 receipt 幂等确认；不需要重发 PUT |
| Inspect 返回临时错误、权限错误或内容冲突 | 不释放，不声称空空间 | 原 owner 沿现有 failure/UNKNOWN 显示，不创建新对象或跳过校验 |
| 已 dispatch 后 Inspect 暂时返回不存在 | 不仅凭 404、超时、租约到期或“任务失败”释放 | 迟到 PUT 仍可能落地；沿原 immutable exact-key 恢复。只有明确供应商无创建终态或已批准删除 proof 才可释放 |
| `STORED` 后业务失败、停用、撤权、版本替换 | occupied 保留 | 沿原业务状态处理；容量不得删除或冒充释放 |
| 原 owner 有批准删除资格，并确认物理删除 | 以原 object/operation 唯一转 `RELEASED`、occupied 减少 | 只接受有界原终态与供应商删除证据；无公开释放接口 |

不持数据库事务跨 S3、provider 或 Money 网络调用；不使用跨数据库事务。接入遵循实际原恢复能力，不把同步方法描述成已有自动扫描：

- **SourceMedia** 当前是同步 exact-key PUT/readback，没有原业务 upload operation 或后台扫描。容量 intent 绑定原 org/actor/content identity；用户通过原相同内容上传重放，在受控 readback 中核实/确认原占用，不新增对象身份。原 GET 读取不发起容量 mutation。首版不承诺该路径自动后台恢复；未重放的未知预留如实保留并展示，不建采集恢复平台。
- **Knowledge** 当前 Processor 只扫描 ADMITTED/OBJECT_STORED/PROCESSING，AVAILABLE/FAILED 不能假设会再次被扫描。Upload 与 Processor 必须先核实/确认容量，再沿原 ConfirmObject 进入 OBJECT_STORED/解析；容量确认失败时保留原可核实 ADMITTED 阶段和原 lease 续行，不能先完成业务终态。原 Inspect 404 的 UPLOAD_INCOMPLETE 或完整性失败仍不能授权释放已 dispatch 预留；按原同 Revision 的显式上传重放核实，不把 parser 重试当作 PUT 重试。
- **Image** 沿原 Run/slot/attempt、manifest 和 worker 恢复，在发布终态前确认对应对象；保留原生成/点数终态事实，不通过容量恢复新增 provider POST。先读取并验证原不可变无生成终态，再在容量事务锁原 reservation 原子关闭并释放未绑定输出的余量；不持容量事务跨业务数据库或 provider 调用。

没有原操作数据/可靠终态时保留 UNKNOWN，不伪造恢复成功，不建立通用 object reconciler。

仅释放未 dispatch 的过期意图也必须由原 owner 在持久化 fence 下关闭；TTL 本身不是安全证明。并发读、批准、引用、删除的合法性仍由原文件 owner 决定。当前未批准 destructive delete 的领域不因本表存在而获得删除权限，S3 versioned bucket 的 delete marker 也不能当作物理 byte 释放。

## 6 扩容与资金合同

当前 `Offer/Quote/OrderItem` 的 Validate 和 `ResourceTypeForProduct` 只接受三类资源；现有 `ServicePurchases` 是第三方子商户原渠道购买，明确不用企业钱包，亦不能借来卖存储包。需要在原 Billing 内增加一个 typed 容量产品分支，而不是把期限装成 DATA_ROW 或恢复旧 SubscriptionPlan。

建议 `ProductKind = STORAGE_CAPACITY_PACK`、`OrderKind = STORAGE_CAPACITY_PURCHASE`，使用原 `commercial_offers/quotes/orders/items`。冻结 `capacity_bytes_per_unit=53687091200`、`duration_seconds=2592000`、单位数量、1990 分单价、pricing/terms version、企业、报价有效期与 request fingerprint；不要滥用原 `term_months` 或 `store_period_days`。报价/订单的容量合同另用窄 typed 字段保存，其余商品不变。数值上界与总价乘法在服务端按 int64 检查。

首版 offer 建议每次 1–10 单位（50–500 GiB），报价沿原有效期与数量检查；多包无需新增等级，后续容量规模限额只有当前实测风险才另作决定。订单描述、筛选、账单汇总、公开价格与登录购买均从同一 offer/quote 快照读完整单位和期限。扩容包不进入 orgresource 消耗账本。

### 6.1 购买流程

复用原状态集合和 Money purchase 操作，不新建支付渠道或第二购买引擎：

`PENDING → FUNDS_RESERVED → FULFILLING → FULFILLED`，未知沿 `RECONCILIATION_REQUIRED`。

1. 具有当前 CommercialPurchase 权限的管理员读取报价、显式确认，原 scoped idempotency key 建立订单；目标企业来自 trusted 当前身份。第一次资金副作用前重验原授权，失去权限不新开订单。
2. 原 Money reserve 使用原 order 身份。结果未知保留订单并查询原 reserve decision，不能改键再次预留。
3. 在原 commercial 数据库准备唯一 `PREPARED` lot 并绑定订单/item/terms fingerprint；这是交付准备，**不可被任何容量准入读取为有效容量**。确定未准备的失败才可沿原 release；准备结果未知先原身份回读。
4. 以原 Money reservation commit，取得与金额/企业/order 一致的可信 commit receipt。结果未知保留，不 release、不激活、不复扣。
5. Billing 与 storagecapacity repository 在**同一 commercial 数据库 Unit of Work**锁原订单和企业容量行，核原 Money commit receipt及 prepared lot，原子写 ACTIVE lot、唯一激活时间/到期和 FULFILLED 订单。激活时刻即有效期起点，重复执行不重设 30 天；订单/lot 任一步失败一同回滚。
6. UI 只有 Money commit proof 与 ACTIVE lot/原订单合同同时有效时展示成功；钱已扣而权益待核实显示原订单待核实，允许查询/恢复该原订单，不能提示再次购买来补救。

新 typed order 的验证必须明确允许 `Money COMMITTED + lot PREPARED + FULFILLING/RECONCILIATION_REQUIRED` 的合法待激活形状，并核原 org/order/reservation/currency/amount/terms proof。它不要求 orgresource grant、不进入现有 `service.go` 的 grant-before-commit 分支、不调用 `ResourceTypeForProduct` 的三类资源映射。旧三类资源的 Validate、grant proof 和处理顺序保持不变。

当前 `commercialbilling.UpdateOrder` 自开事务，不是现成跨 repository UoW。本批需要原 Billing 内的窄 storage fulfillment transaction：向 storagecapacity 与 Billing repository 传递同一个 commercial tx，事务内共同锁原订单/account、核 prepared lot 与原 Money commit proof、写 ACTIVE lot/激活 receipt/FULFILLED。不能分别调用两个自开事务方法后声称原子成功。

Money 与 commercial 仍是两个数据库，第 4/5 步之间可能中断；原 Billing bounded scan 加入 STORAGE_CAPACITY_PURCHASE 并按 kind 分派到窄 `ReconcileStorageOrder`，以原 commit proof 重试同库激活，不能把它交给当前只调用 ReconcileResourceOrder 的实现。这一组合复用已存在的每批有界扫描/恢复生命周期，不建第二 scheduler。并发管理员重复提交、原响应丢失、撤权后后台收束和重启都只能产生一次金额 effect 和一个原 lot；后台只收束已准入原订单，不通过撤权恢复新购买能力。

### 6.2 到期与再次购买

例如基础 5 GiB，加一个 50 GiB 包后总容量 55 GiB。已用 20 GiB 时该包到期，总容量恢复 5 GiB、超出 15 GiB，已有数据继续可读；所有新占用拒绝。再购买一个 50 GiB 包后恢复 55 GiB，原已用不会重新收费或清零。

提前再买一包会在其激活后增加另一份 50 GiB，并不是延长第一包；各自保留原 starts/expires。首版没有自动扣费、自动续期、自然月对齐、按剩余天数折算、按超量账单或“永久免费扩容”。有效期变化只影响容量准入，不改 Money 既有债务与 provider 资金冲正规则。

本批不开放容量退款商品或复用旧 subscription refund。资金冲正不能删除用户对象；如果正式开放需要把新容量销售纳入既有退款/纠纷受理流程，该具体容量权益冲正合同必须先明确，是销售接线条件，不自动授权本轮另建退款状态机。

## 7 权限与接线

| Contract | Implementation | Injection | Consumer |
| --- | --- | --- | --- |
| SummaryReader | storagecapacity + 窄 GORM repository | current application commercial composition | 原资源页、我的权益，同源 BFF |
| ReserveWrite/BindObjects/ClaimDispatch/AbortUnstarted/ConfirmStored | storagecapacity 事务；trusted exact-object inspector | app 内原 Knowledge、SourceMedia、Image/Supply 的 scoped adapters | 原业务 HTTP/worker；无浏览器写容量 API |
| PrepareLot/ActivateLot | storagecapacity + Billing 的同库 UoW | 原 commercial billing builder | typed storage order + 原恢复 loop |
| Immutable put/inspect/copy/read | 现 `integration/s3` | 原 bucket 配置、restricted namespace registration | 原领域对象存储端口；不绕过私有读/原 PublicBase 政策 |

拟新增只读 `GET /api/v1/workbench/storage/summary`，沿当前 verified identity、live Organization、既有 CommercialRead permission、bounded response、15 秒 deadline、no-store；返回当前 org、policy/terms version、quota/occupied/reserved/remaining/overage decimal strings、observed_at、分类聚合和有界 lot 摘要（最多 50，余项原分页）。不返回其它成员的文件名、key 或正文。

购买沿原 Commercial quotes/orders 路由的 typed branch，body 只允许 offer、数量和原操作身份。第一版目录首批 1–10 单位；请求大小沿原商业 JSON limit。**只保留既有三类资源 API 语义**；公开原 resource-offers 不能静默变成任意容量商品，建议增加同一 Billing catalog 的窄只读 `GET /api/v1/commercial/storage-offers`，沿现 price-only descriptor，输出 bytes、duration、价格、版本及数量上界，不含企业/钱包/订单。不复制价格源或初始化逻辑。

容量 summary 读取不会赋予文件 read/write。Reserve/dispatch 前原业务 live 权限仍必需；任意 user/org/对象归属漂移拒绝。已 dispatch 的原恢复可核实占用和财务收束，但不授予被撤成员读取内容或新派发能力。只读观察失败时拒绝依赖它的新准入，不能回落为无限额度。

在 current application 开关中有界启用本模块。商业池、合法 schema、namespace 注册、所有已开启计量写入 adapter 必须就绪；声明“容量受控”而有直写绕过时启动/受影响写入 fail closed。模式尚未配置则容量产品 unavailable，不把旧路径存在当作容量可用。API 与图片 worker 必须注入相同容量 owner，不能只限制浏览器上传。

## 8 超限与操作结果

| 操作 | 满额或到期超限行为 |
| --- | --- |
| 新上传、新下载落盘、增加持久化文件 | `STORAGE_CAPACITY_EXCEEDED`，不 PUT、不新扣款；返回可信所需/剩余 bytes |
| 新 AI 生成 | 在原 paid dispatch 前预留全部有界结果空间；不足则 0 新 POST，提示扩容 |
| 已获得有限预留的原生成/保存 | 继续原身份收束；成功真实 bytes 由 reserved 转 occupied；不扩大预留 |
| 原相同对象重放或增加引用 | 不增加物理占用，原权限和幂等检查通过可完成 |
| 读取/下载已有可读内容、编辑纯数据库资料 | 容量不额外阻止；原权限、停用和内容生命周期仍生效 |
| 隐藏、归档、停用、版本切换 | 不虚假释放 bytes |
| 物理释放 | 仅原 owner 已批准安全删除 + 可信对象终态；首版不提供通用清理入口 |
| 容量服务故障、写入 UNKNOWN | `STORAGE_CAPACITY_UNAVAILABLE` / 原 `OUTCOME_UNKNOWN`；拒绝新占用，保留原 operation 查询入口 |

未知预留单列展示为“写入待核实”，不能并入成功文件数字，也不能由用户用“取消”按钮强行清零。清理按钮是否可用由原领域能力决定，不强制新增删除能力作为查看/扩容的前置。

## 9 实施与验证边界

用户确认产品规则后，在同一 Delivery Batch、一个 Writer/分支/主要 PR 连续完成：容量账户与预留 → Billing typed 容量商品/同库激活 → 当前实际开启的 Knowledge/SourceMedia/Image 及业务附件写入 adapter → 原资源/权益/订单页面。未开启的业务无需提前实现，但不能漏掉当前组合实际开启的计量写入。切片是提交/验证边界，不机械拆 PR。

按现有 TDD 和 PG/MinIO/worker/BFF 测试验证改变的风险，不创建 runner：

| 不变量 | 必需开发证据 |
| --- | --- |
| 企业隔离、原权限、没有直接 mint/release | 当前 live middleware/BFF 负例、trusted owner adapter、跨 org/key 认领拒绝 |
| 新预留时占用加预留不超过当前 quota | 真实 PG 两连接并发抢最后空间；行锁与唯一对象重放；锁等待跨到期、包激活竞争、ClaimDispatch 与 Abort 同锁竞争 |
| 相同 key 不重复占用，不同 payload 不覆盖 | 原 immutable S3 与容量 receipt 重放/冲突测试 |
| 已写不释放、UNKNOWN 不清零 | PUT 成功回执丢失、容量确认失败、迟到写入/404、原恢复和重启 |
| paid dispatch 前容量就绪，原点数合同不变 | 生成前仅原 slot 身份可预留；结果 BindObjects 不超上界且重放唯一；容量不足 0 POST；预留后点数失败；部分生成成功与未绑定 UNKNOWN |
| 容量确认接线符合实际恢复入口 | SourceMedia 同内容 Upload 重放，无假后台扫描；Knowledge 容量确认失败保留可扫描阶段，不从 UPLOAD_INCOMPLETE/解析终态释放 |
| 只在 Money 确认后激活，有效期只定一次 | 原钱包响应丢失、PREPARED 不可用、Money commit 后同库事务失败、重启/并发激活 |
| 到期与业务生命周期不丢数据 | UTC 边界、已用高于到期 quota、既有预留可收束；disable/未采用不减占用 |
| 页面与报价读同一真实 owner | 无 offer/unavailable/0/UNKNOWN 区分；切 org 迟到响应；bytes/价格精度与原订单 |

独立检查仅审核本轮新容量/财务边界及最后形成的用户路径，最多两轮正常架构评审，finding 按仓库分类。CI、自检、设计评审不签发产品验收。当前没有实现，全部运行/支付/真实 provider/浏览器业务验收为 NOT_RUN。

只对全新安装、空业务数据和新计量 namespace 准入。不扫描旧 bucket 补写，不迁移旧订阅值，不对现有保留试用实例默认为 0。启用时用部署初始化证据确认计量 namespace 全新/空且只由已注册 writer 写入；非空 namespace、历史业务实例或混用其它应用必须拒绝容量受控启动，报告具体对象范围等待单独当前数据处置决定，不能用隐藏基线绕过。正常 stop/restart 保留卷、容量回执、订单和对象。

## 10 Legacy 决策与开工条件

```text
Legacy decision: EXTRACT | RETIRE
Reusable behavior: 当前 S3 不可变 I/O；有效报价/钱包/幂等及恢复模式。
Current owner: Commercial/Billing、Money、ledger/storagecapacity、原业务文件/资产 owner。
Cutover/deletion condition: 新安装受控写入直接使用容量端口；旧 oss_storage 套餐/usage 不作为依赖，无 fallback/双写/迁移。
```

本稿尚不能标 IMPLEMENTATION_READY。开工前需要用户决定第 1 节参数与数据保留/费用政策，确认现有 Figma 页面新增存储区域和购买交互的产品修订，并完成针对实际容量与财务边界的独立设计检查。正式销售还须确定实际云成本及适用的容量销售退款/纠纷受理边界；这些不把只读设计或隔离开发写成生产可售。

用户确认后由协调方将批准决定写入本 Issue 和本稿，并按适用准入冻结。价格、容量或其它建议未经批准不能写入正式初始化配置；独立 Reviewer 无权代用户批准商业参数、数据清理或继续开发。
