# Report Center：当前 SHEIN 已保存资料来源候选

Execution: [#628](https://github.com/henryxu1008/shuomi-ai-commerce/issues/628)。

- Design Basis: **Independent Architecture**，新增当前来源的读取、授权、版本和持久化种类合同。
- Status: **DESIGN_ONLY / USER_CONFIRMED / IMPLEMENTATION_READY（架构）**；正式实施仍 NOT_AUTHORIZED / SOURCE_NOT_STARTED。
- Investigation baseline: `161b1de17e1deb769fd661162a9fb38786ab003e`，2026-11-01。
- 用户已认可新来源与新空 Report 库候选、确认离线诊断未覆盖，并授权单一独立 Reviewer；原候选 `1ebabf3f84b2fda8512a50117717fda8e4b9f10f` 的 R1 无设计级 BLOCKER，两处文档澄清同批修正。准确 HEAD 的最终增量/CI证据维护在 [PR #1045](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1045)；未授权实现、合并、部署或关闭。

## 1. 用户结果、权威和停止点

使用者是在当前企业内有报告权限、且本人有权读取原资料的运营人员。用户从现有 SHEIN 已保存平台资料进入“我的报告”，明确保存所查看的准确版本，之后查看、收藏和下载 JSON/文本；报告是历史展示快照，不成为商品、审核、授权、上传或恢复输入。

依据：用户于 2026-10-10 确认的个人/当前企业/手动保存/只读版本决定，本轮“先设计当前 SHEIN 来源、交用户确认、暂不实现”的明确授权，以及以下现行合同：

- [个人报告冻结基线](ai-workbench-reports-v1.md)：Report owner、原键、独立 Report DB、收藏和下载。
- [SHEIN 供应链](my-supply-chain-shein-v1.md)、[审核/上架增量](shein-publication-closure-v1.md)：当前 TargetRecord 与官方回执不是旧 offline Record。
- [当前 UI / IA 权威](../product/final-ui-ia-authority.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md)。
- 原报告 Figma：`tg48P46SSXl6TBy9lZwg63` / `31:463`，总览 `427:5695` / `427:5900`、最近 `662:481` / `662:641`、收藏 `579:4677` / `579:4900`、全部 `579:5123` / `579:5346`。这是原已查阅权威，本轮未重新读取 Figma，不宣称新的视觉验收。

本候选复用原报告页面、保存确认、收藏、下载和原来源页面职责，不新增报告中心或并列工作台。只补来源选择/定位及真实类型说明；不从图面增加数量、格式或分析能力。

**本候选只交付当前已保存平台资料及其保存时构建结果的报告设计。** 它不把 `TargetRecord.Result` 叫作旧 `DiagnosticResult`，不取消 #628 原离线诊断目标。旧正常 SHEIN 离线诊断保存仍未开放，剩余目标保持 OPEN / 未完成，不用本候选、未来局部实现或 CI 替代。

明确不做：新模型/平台调用、重算规则或诊断、资料/Store/资产变更、Apply/上传/上架、团队共享、任意文件/Office/PDF、新报告或校验引擎、通用来源注册器/迁移器、历史报告迁移或重解释、真实数据与配置操作。

## 2. 已查证现状与根因

| 当前路径 | 已查证事实 | 本候选处理 |
| --- | --- | --- |
| `internal/app/httpapi/current_report_center.go::buildReportCenter` | 正常装配仅注入可选 Review reader；明确不能把当前 TargetRecord 替换为旧 offline Record | 仅从已创建的正常 `supplyRuntime.app` 注入当前只读端口，不另建 app/worker，不挂隔离试用装配 |
| `internal/app/reportcenter/sources.go::Sources.Read` | `SHEIN_RECORD` 绑定 `record.Reader.ReadOfflinePackage`、旧 InputHash 与保存时 DiagnosticResult | 原类型/含义保留；当前来源用独立种类，不按 UUID 猜 owner、不 fallback |
| `internal/listing/record/target/target.go::TargetRecord` | 每个保存版本有独立 ID、TargetID/Revision、原来源/effective version、Product/Inventory/Rules hash、Merchant、Input/Result 和 CreatedAt；没有旧 offline InputHash/Diagnostic | 用原不可变资料和原已保存 Issues/ReadyForUpload，不伪造诊断或当前平台结果 |
| `internal/app/supplychain/application.go::ReadRecord` | 原 preparation read 授权、精确 ID、原 source 访问与可选原消费者检查 | 复用正常只读方法；不把 ReportManage 当成 Supply/source 权限 |
| `internal/integration/persistence/listing/record/repository.go::readRecord` | SQL 限 organization/actor/member/id，验证原 body hash；保存追加记录、更新另一个 head | 精确 record ID 读取，不用当前 head 静默替换旧版本；Report 不直接读 Product SQL |
| `internal/reportcenter/service.go::Save` | 先按 scope/key/fingerprint 找回原命令；新命令才读源/比完整 ref；Report 事务重验报告权限后保存快照/收藏初值/回执 | 复用，不建立跨库事务、新重试 owner 或补偿流程 |
| `internal/integration/persistence/reportcenter/schema.sql` | `kind varchar(20)` 的 CHECK 只允许 TITLE_REVIEW/SHEIN_RECORD | 第三种类需要明确 schema 准入；不是只给 UI 开按钮 |
| 当前 `VerifySchema` | 验证列及受限角色/表列权限，未验证第三种类 CHECK | 新来源启用须额外只读验证该 CHECK；不能让 UI 可选、实际保存才失败 |
| 当前报告 source picker / contracts | 两种种类，SHEIN 新保存禁用；旧 sourceHref 指旧诊断页面 | 分开显示新资料来源与旧未开放离线诊断，不能用旧诊断链接打开当前资料 |

本轮只读代码和公开仓库证据，未读取任何保留数据库、Report 行、用户资料、凭据或运行配置。

## 3. 来源、版本与只读投影

候选新种类为 **`SHEIN_TARGET_RECORD`**（19 字符）。仅支持当前 native SHEIN TargetRecord；TITLE_REVIEW 和 SHEIN_RECORD 的身份、含义及保存报告不变。未绑定 CompletionDraft、其他平台、上传/审核/库存回执不是该种类，不通过同一个入口冒充。

请求仍只有现有 `SourceRef {kind,id,version}` 和报告幂等 key。`id` 是当前精确 TargetRecord ID，不是 ProductKey、SourceID、TargetID、StoreID 或 latest/head。浏览器不提交可信 actor/org/member、报告正文、标题、规则、ready、hash 或 URL。

版本候选复用现有 `collection.Digest`：

1. `bodyDigest = collection.Digest(record)`，与 native owner 已验证的不可变 body 语义一致。
2. `version = "sha256:" + collection.Digest({projection: "shein-target-report-v1", bodyDigest})`；`collection.Digest` 本身只返回裸64hex，前缀在这里显式添加。使用固定字段结构，不使用 map 或当前时间/店铺名称/当前 head。
3. 新种类的 version 必须是规范 `sha256:<64 lowercase hex>`，长度不超过原 96 字符。projection 格式变化须显式换版本，不能同 ref 生成不同正文后覆盖。

`kind/id/version` 三元组区分两个事实 owner；恰好相同 UUID 不发生跨类命中。Report 自己的 content digest 仍由现有 `Snapshot.Bytes` 计算，与原资料 digest/来源版本分别显示。

固定白名单投影使用现有 Document schema 1：

| 报告部分 | 允许内容 | 不能推导的含义 |
| --- | --- | --- |
| 保存时资料身份 | 精确 record/target/revision、原 source、Product key/固定及 effective version、Store/site、CreatedAt、三类事实 hash | 不当作当前 head、当前授权或平台商品状态 |
| 资料摘要 | 原保存标题/描述等必要文本、准确 SKU/图片数量等可解释摘要 | 不是完整 payload/全 SKU 导出；不复制图片、URL、外部附件或完整请求 |
| 保存时构建结果 | 原 Result.Issues 的 field/code/message，原 ReadyForUpload 明确标成“保存时构建结果” | 不执行 BuildOfficial/validator；不能称当前可上传或旧 offline diagnostic PASS |
| 范围说明 | 未重新评估规则/资产与在线条件；未读取平台审核、在架、库存及实际效果 | 缺证据 UNKNOWN / 未执行 NOT_RUN，不能从保存成功得到发布许可 |

不含 provider 原始响应、SubmissionPayload、密钥/签名、merchant 执行句柄、临时链接、图片内容或其他 actor 私有资料。所有文本按原纯文本呈现；不执行 HTML/Markdown、外链或提示中的指令。

上限沿现有合同：Document ≤128 KiB、≤16 sections、≤256 fields、单值≤16000字节，沿原 label/title/UTF-8/控制字符校验；原 HTTP/BFF 上限不提高。必需投影或全部 Issues 超限就拒绝新捕获且不写回执，不静默截断/拆分为“完整报告”。SKU/图片数量是明确摘要，不伪装复制了未保存的明细。

## 4. 授权、装配和用户入口

调用链：现有 Supply 资料入口 → 原报告页精确来源预览/明确保存 → Report Read/Manage + verified effective org → 已装配正常 Supply 的只读 `ReadRecord` → 原 actor/member-qualified repository/source 授权 → 白名单 Snapshot → 原 Report 事务 → 原列表/收藏/下载。

- 使用当前请求绑定的 org/user 与 native membership，不能从 Report grant、URL、记录内容或角色名称推导 source 权限；无 admin 越过本人限制。
- 原 reader 必须是正常 full Supply 的现有实例；不能注入 isolated trial、SourceMaterialsOnly 或第二个 Service/数据库连接。正常 `supplyRuntime` 构建先于当前 Report 模块，已有注入点可复用；不要求启用 Product Agent 或模型。
- 只读 `ReadRecord` 已有 scope/来源约束必须走完；适配器另核 ID、SHEIN 平台及对应 native形状，非SHEIN/错ID/不完整值拒绝。实际 mounted reader 的跨org/actor/member拒绝是实施验证义务，不只测替身。
- Source 预览要求 ReportRead 和 source read；新保存要求 ReportManage 和 source read。Report 的 live 身份不得被静态执行scope代替。读不到、无权限、缺owner与超限分别沿原错误策略处理，不伪造空报告。
- 原来源页面上的候选“保存报告”入口只携带 kind/id 到现有报告页，不发送保存、Apply或上传；报告页重新读取服务器生成的完整ref，再由用户显式保存。不追加手抄UUID、CLI或人工验收流程。
- 复用报告页预览/保存状态；新增类型筛选/名称与 same-origin source 参数白名单。query key、source item key 与 pending intent 都含 kind/id/version 和当前 user/org/权限上下文，类型碰巧同 ID 不复用旧预览或命令。
- “打开原来源”必须进入 Supply owner 的精确已保存版本读取，不能沿旧 `/workbench/shein-records/:id/diagnostic`，也不能改读当前 head。现有 Supply 通用详情以 source/store 为上下文，若缺少 record-only 深链，最小补齐是在现有 Supply 资料页面增加受限 record locator（候选 `/workbench/supply/mine/adapted?record=<UUID>`）：通过原 `GET .../records/:record_id` 授权后展示原保存记录，不自动执行规则/模型/平台或写操作。这是必要 consumer 接线，不新建工作台/事实owner；不能以一个泛列表链接冒称精确溯源。
- 历史报告 Read/List/Favorite/Download 不重新读源；来源撤权/删除不让报告成为访问源的通行证，当前 report membership/本人权限仍在每次请求校验。原设计允许保留已获授权导出的历史快照，此边界不扩大。

## 5. Report schema 与启用边界

本候选不新增表/快照owner或数据迁移；未来新安装的同一 Report schema 将 CHECK 精确扩大为三种已定义 kind，其他结构/唯一键/FK/列权限不变。新种类落在原 varchar(20) 内。保留库当前 CHECK 未读取，不根据源码猜其实例版本。

拟增加一个默认关闭的、仅管理新捕获的显式来源 opt-in（候选名 `ReportCenter.SheinTargetSourcesEnabled`，不是本轮已有配置）：

| 请求状态 | 准入与结果 |
| --- | --- |
| 未启用新来源 | 原 TITLE_REVIEW/已保存报告照原合同；新种类新捕获不可用，不静默转旧来源 |
| 已启用 | 必须已有正常 Report + full Supply reader，在 listen 前只读验证 kind CHECK 支持精确三类，并复用原 VerifySchema 角色/列权限检查。原唯一键/FK DDL保持不变，复用原安装及事务回归；当前VerifySchema未验证键/FK，不将其写成已有检查或扩大为新机制。不满足则拒绝配置，不宣称来源可用 |
| 新空 Report DB | 后续获准实现时，沿原 native initializer安装已批准的新 CHECK，不在serve时DDL |
| 旧/保留 Report DB只有两类 | 本候选**不提供 ALTER、数据迁移、重建、copy/双读或升级工具**。不能打开opt-in凑可用，不能更换真实连接。将来启用须有合格的正常初始化条件，相关环境动作另行授权 |

这是同一当前 Report owner 的来源能力准入，不建立旧/新Report同步、schema兼容执行链或第二事实源。原已存 SHEIN_RECORD 不改标签/ID/version/正文；现有 Report 类型解析保留其原含义，绝不把它路由给 Target reader。新来源关闭不阻断已提交新类命令的原键回读：种类解析与快照读取不依赖来源启用，只有新捕获依赖源端口。

**用户确认时须看见的限制：本候选不是在现有保留Report库直接启用的方案。** 若要求在该库追加种类并保留原数据原址使用，应先另定当前owner的schema演进及恢复边界，不能由实现者顺便写DDL或引入兼容层。本轮只交设计，不执行任何安装/数据库动作。

## 6. 事务、原键、撤权和故障

保持原三表事务及唯一约束：scope/key/fingerprint先Lookup；同key同ref回读原结果，不再读变化/已失权的源；同key异ref冲突；新key才做当前源授权、固定版本与Snapshot校验，再在Report事务重验ReportManage，原子保存report、初始favorite与command。

source读取与Report提交仍是不同边界。原保存资料不可变；用户要求的是所读的精确版本，读完后出现新head不改变历史内容。source授权在源读取时生效，Report权限在提交时重验；不宣称跨库原子撤权或加跨库锁。既有快照读取不重新获取源权限。

| 中断/并发 | 沿用结果 |
| --- | --- |
| 预览与新保存之间版本不同 | 比完整ref，409；不偷偷保存另一版本 |
| 源读取失败/失权/错平台/超限 | 新捕获失败、无Report/回执；不重试源业务或造诊断 |
| 读完后出现新版本 | 若已选不可变原记录仍获准读取，保存其历史版本；不是最新资料声明 |
| Report事务失败 | 原事务回滚；无半份报告/孤立收藏 |
| COMMIT或响应丢失 | 保留原key/ref为UNKNOWN，原键恢复；不用新key猜重建 |
| 两key同精确来源 | 原唯一身份收敛一份报告，内容digest不同冲突，不覆盖 |
| 收藏/取消收藏重放 | 原set-target/row lock/命令回执，旧key不重施旧目标，返回当前收藏 |
| 进程重建或来源被关闭 | 原已提交报告/回执/收藏由Report原store恢复；不启动源执行、平台查询或scheduler |

原权限/错误映射、期限、取消、source query缓存隔离及download数据边界保持。没有新执行状态机、异步恢复owner、outbox、Saga或后台任务。

## 7. Legacy decision 与未覆盖项

Legacy decision: **EXTRACT**（复用当前已批准行为）；不对旧owner加wrapper。

- Reusable behavior: 当前 Report store/事务/私有页面，当前 Supply精确读取/原授权，既有Document/JSON/文本导出与native schema初始化。
- Current owner: Report保存展示快照/收藏/命令；Supply/Target/source owner保存业务事实；Store/Asset/Review/Submission继续原职责。
- Cutover/deletion condition: 新种类达到独立准入并获实施授权后，仅它消费当前reader。旧SHEIN_RECORD保持原定义和历史报告读取，不作为新来源fallback；本轮无删除或旧装配复活。未来是否退休旧捕获adapter须单独查当前消费者，不在本设计顺手处理。
- 当前未覆盖：旧offline DiagnosticResult的正常生成/捕获、未绑定草稿及其他平台报告、完整SKU/媒体导出、平台实时状态、全面合规、真实安装/保存/恢复和产品验收。以上没有转PASS或Accepted Risk，#628不能因本设计或未来这一局部来源就自动关闭。

## 8. 后续最小验证与批准门槛

用户已确认本候选范围并授权单一独立架构复核；原准确候选R1已给出 IMPLEMENTATION_READY（架构），本批两处文档澄清和README历史后缀修复由同一Reviewer只做增量核对。架构准入不授权实施；必须取得后续明确实施授权，才在同一Delivery Batch/主要PR先补失败回归，再补最小实现。本轮不另派Reviewer、不写生产路径或schema。

后续复用现有 Report、Supply mounted reader和PostgreSQL测试，验证：

1. 同一正常应用的native保存记录→server ref→明确保存→列表/收藏/JSON/文本，精确版本和历史labels；不以seed报告表代替源读取。
2. 两actor/两个org/更换member、撤权和仅报告管理权限不能读别人的资料；非SHEIN与错ID拒绝，viewer/源失败不伪造0或空报告。
3. 旧SHEIN_RECORD与新kind同UUID不混用；三元身份/原键异载荷冲突、同键回读、响应丢失、收藏旧key不重施，以及source关闭后已提交结果仍可读。
4. 快照hash/formatter版本/原revision稳定；Issues完整与原限额、越界/不可信文本、type/query/body严格校验；当前业务字段/权限/原记录不写。
5. 新空schema与只读CHECK准入；缺第三kind时opt-in拒绝，默认关闭时原报告能力不因此改变；不对保留库执行安装。
6. 原来源/报告两端定位同一exact record、权限上下文/迟到响应隔离；源码与测试替身核计数确保报告捕获不调用模型、平台、规则重算、保存/Apply或Submission。

不新建runner/验收平台，不默认执行完整人工矩阵。未来开发自检、独立复核、PR CI、actual merge CI、运行和用户验收各自记录。

本轮状态：仅文档/引用/索引自检及已授权单一独立架构复核；原R1为 IMPLEMENTATION_READY（架构），准确最终HEAD/增量及CI见PR回执。正式实现、PG功能/故障回归及真实环境均 **NOT_RUN**，现场准备与旧库状态 **UNKNOWN**。旧候选README历史后缀CI失败保留，不以架构Ready或本地自检代替新HEAD CI；未批准新风险。

用户认可的边界是：独立新kind及明确资料/诊断边界；仅正常full Supply读取和显式source opt-in；新schema正常初始化而非保留库变更；最小来源定位consumer。本轮架构复核与文档收敛不自动授权实施、更多评审派工、合并、部署或关单。
