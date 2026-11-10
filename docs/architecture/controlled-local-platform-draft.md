# 原 35444：受控本地平台草稿接线

> 2026-11-10 最新产品决定：同一实例应允许测试草稿与正式刊登共存，最小方案及独立评审见 [共存设计](local-and-official-draft-coexistence.md)。本文的实例级互斥条款在该新方案准入后由其替代；持续 localTrial、测试稿永久不可发布、固定试点范围和原数据保护仍有效。#1235 的实现、合并及原实例更新已完成，准确运行证据维护在 #1158；下方设计阶段授权/状态作为历史保留。本轮新授权仅含共存设计及评审，不授权修改原实例或生产代码。

Execution: [#1158](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1158)。Design Basis: **Independent Architecture**。
Admission: **IMPLEMENTATION_READY / FROZEN**。2026-11-10 独立 R1 及文字增量复核完成，冻结设计为 `950b055d4ff77ec2c51015ee6dd97a797186c3cc`，0 设计 BLOCKER；具体接线 finding 已分类 IMPLEMENTATION_TEST，见 §9。后续实施依用户明确授权推进，设计准入本身不授权原实例操作。
调查基线：`683d83b831020d9266ea06b636c7beced7d97453`，唯一 Writer `/root`，分支 `codex/issue1158-controlled-local-draft`。

## 1. 产品决定、结果与边界

2026-11-10 用户明确授权：先设计原 35444 的受控本地平台草稿接线并做独立评审；复用合成店铺、规则、现有 Target 保存；同一份草稿保留黑 M、黑 L、白 M，库存 0 / 未填写 / 未填写，原商品、R1、图片保持。**不写生产代码、不改原实例、不调用模型、不保存或发布、不释放原预留。** 实现、隔离验证、合并和实例开通均待后续阶段授权。

阶段更新：同日用户后续明确授权按冻结设计实现，并在原库隔离副本完成创建、带入、映射、保存、刷新和四项实施验证。仅副本可保存合成店铺及 Target；不改原 35444、不调用模型、不发布、不释放原预留，合并与原实例开通仍未授权。上段保留为设计阶段记录，不撤销后续实施授权。

使用者是本地试用维护者/运营人员；当前阻塞操作是从现有已保存 R1 预览并保存一份绑定测试店铺的平台草稿。交付目标是“创建基础稿 → 带入 R1 文字/明确库存 → 映射销售规格 → 保存同一 Target 新版本 → 刷新重开”，明确标识**本地试用、合成规则、不可发布**。

Must：一商品、一 Source、一固定 Completion R1、一测试 Store、一普通 Target；两个颜色组、仅三行真实 SKU；0 与空分开；不自动换汇；公共图不变、不造专属图；原商品/R1/图片不写；原 Scope/权限/CAS/原键恢复保留；本地记录永远不能成为正式发布依据。

Out of Scope：真实 SHEIN 授权/服务开通、平台接受三行的资格证明、官方网络读写、发布/上下架/上传/结果核实、AI/素材生成/批准/资源购买或结算、拆色 family、补白 L/删黑 L、泛化模拟平台/规则引擎/新的 IAM 或 runner。已有正常能力不关闭、不换库、不重设密码；不修改当前真实平台规则或已批准风险边界。没有新增 Accepted Risk。

依据：[当前 owner](project-boundaries.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md)、[R1 带入](completion-shein-import.md)、[已实现销售映射](shein-sales-attribute-mapping.md)、[现有离线质检边界](private-draft-offline-trial.md)。Figma 只作流程参考；导航、页面和无障碍使用当前组件，不按原型画面验收。

## 2. 当前证据与最小选择

- #1232/#1234 已更新原 35444；商品 V1、Completion R1、原失败缺价任务的登录后只读检查已完成。原实例仍无 Target 表、无 supplyChain 配置，前端 `LISTINGKIT_SUPPLY_CHAIN_ENABLED=false`。
- 唯一原店铺是驾驶舱测试记录，pending_activation，无官方 connection/merchant binding。本轮不改它，不将它伪装为 connected。
- 普通 `SupplyChainConfig` 同时要求官方应用、有效商户/规则和 Temporal；现有 `PrivateDraftTrial` 只挂 GET 质检读取且与原 AI/供应市场配置互斥，不能当带入保存入口。`SheinOfflineRecords` 是另一离线资料流程，不是本次 Target/销售映射的替代 owner。
- 当前 Catalog/Source/Completion、TargetService、严格 BFF、单 Product DB 事务、合成规则 fixture、Store native 创建和现有显式安装入口可复用。Asset DB 已是 `image_agent` / `supply_asset_runtime`；不建新库。

**最小选择**：增加一个固定一组 Scope/Source/Store 的 `controlledLocalDraft` 装配。复用 Target owner 与原三个创建/预览/保存算法，只替换其规则/店铺读取为受控本地来源，并加不可丢失的试用 provenance 和 native 发布拒绝。不是全局放宽 `validateListing`、重开旧服务或把 synthetic provider 装成官方连接。

## 3. 固定输入与业务结果

| 事实 | 本轮固定值/约束 |
| --- | --- |
| Organization / Actor / Member | `394513989062623234` / `394513839896395778` / `394513991663091714`，仍须正常登录和 live 权限 |
| Collection item / Product | `99b8f948-84fd-5be8-a6b1-cafc071c7fe2` / `own-27ef901c-30e0-57f9-81d9-56032b0efa61`，版本 1 |
| Preparation / Source | `cb910ad1-5f9e-56dc-a99f-6319ee47f7f5` / `41cb802a-fa17-523e-9444-12044f47e5dc` |
| Completion | record `f7853e11-353a-5214-a7c7-75a8ded6e278`，revision 1；不静默换 head |
| Store | 后续另行授权后由原 Store owner 创建一条新“离线测试 · #1158 本地草稿”记录；准确 ID 取原创建回执再写私密配置；不复用/改写驾驶舱店铺 |
| Rules | 固定 `issue1158-local-draft-v1` 的不可变编译内 fixture，version + 内容 hash；类别/颜色/尺码 ID 明确是 synthetic，不是官方 ID |

目标最终布局：黑色组 `{QA1195-BLK-M, QA1195-BLK-L}`，白色组 `{QA1195-WHT-M}`。sourceIndex 0/1/2 完整双射，库存分别有一项 quantity=0 / 无库存条目 / 无库存条目。不得写成三个 0。

R1 当前价格为 CNY 29；原文/币种不变。模拟自营规则仍按 USD 校验，**不选择 R1 价格带入**，Target 价格留待补全，不把 CNY 29 写成 USD 29。黑 M 的 0 通过明确的 synthetic US 仓映射；另外两行不选择库存带入。用户显式确认颜色/尺码枚举，不靠名称正则、SKU 编码或模型推断业务事实。

原有一张黑白合成公共主图仍归原 Catalog/Source，界面只读展示原图；不复制到 Asset 批准库、不新增/批准/上传图片、不造白 M 专属图。Target 平台图片槽先为空，显示待批准；本地草稿可以保存不完整资料。以后批准图片属于另一明确授权，不在本批。

## 4. Owner、contract → implementation → injection → consumer

| 责任 | 复用/最小增量 |
| --- | --- |
| 登录、企业、成员、权限 | 原 authidentity/workbenchcontext、SourceSelector 与 `supply.read/manage`；没有固定身份替代登录 |
| 原商品/来源/R1 | 原 Catalog/Collection/Preparation/Completion，只读精确版本；不创建第二正文 |
| Store | 原 Store aggregate/repository、native create key/receipt；active record + pending_activation + disconnected，无 merchant binding |
| 合成规则 | 当前 SHEIN RuleReader 的闭合本地实现，复用已测 DTO/fixture；只接受固定类目/属性/仓，无 URL、密钥、任意规则 JSON 配置 |
| 草稿与 provenance | 原 `TargetInput/TargetRecord` typed 合同 → 原 TargetService → 原 Repository `body_json`/head/command；无新 owner、表或 SQL 列 |
| 装配 | `currentapplication` 解析私密 controlledLocalDraft → 明确的本地 constructor → TargetDependencies；不创建 OfficialProductAccess/provider/Temporal/Submission |
| UI/BFF/HTTP | 当前缺项页和带入页的窄分支，严格完整 DTO；服务器 route allowlist 和 Scope 决定准入，UI 开关只决定可见性 |

配置必须 literal-loopback HTTPS，固定上述 Scope/Source/Product/R1 和准确测试 Store、trialId、fixtureVersion/hash、显式 acknowledgment。`enabled` 缺省 false，仅控制本地预览/新写；配置块存在且 schema 已安装时，false 仍保留准确固定范围的只读 reader/原键核对及 native capability 声明。配置块缺省表示未开通；安装后完整删配置不是停用方式，须按既有 capability 拒绝不一致启动，或在明确回退窗口恢复匹配 checkpoint。与 full SupplyChain、PrivateDraftTrial、另一离线记录模式互斥；原 SourceMaterials/Agent/供应市场/数据服务保持可并存。不得为了接线而关闭原功能或取消其准入。模式字段、Scope 和规则均不由浏览器或模型决定。

装配明确区分真正 full SupplyChain app 与 controlled app：后者不得作为 `buildSourceMaterialsModule` 的 non-nil full 参数。原 SourceMaterials 沿原 owner 的独立准入/`composeSourceApprovals` 装配，保留其原能力；Source/Preparation routes 由原模块拥有并准确去重，本地只补 Target 小集合。原模块本来可用的图片批准入口不等于本地 Target 白名单的批准能力，试用模块不能调用它或把本地 Target 传作执行输入。此为 R1 核实的具体接线义务，不新增另一套素材 owner。

Store 每次 preview/new save 通过当前成员访问读取准确记录及版本，核对 SHEIN US、active record、pending_activation、未连接且没有 merchant binding。名称只是标签，不能成为唯一授权依据。创建 Store 不购买资源、不授予官方商户身份；普通成员仍要求原店铺 grant，管理员沿现有规则，不额外发明 grant。

Target 普通身份仍沿 `TargetIdentity(org,actor,source,store)`，purpose=standard，无 colorPartition/family。新 synthetic StoreID 使身份与任何真实 Store 分开；不能改 StoreID 把本地 Target 变成正式草稿。

## 5. 不可升级为正式记录的来源约束

新增可选的**typed `localTrial` 绑定**，贯通 TargetInput/TargetRecord、BFF/响应/历史/原键 intent。内容只含 contractVersion、trialId、固定 Source/Completion 引用、fixtureVersion/hash、准确 StoreID；Scope 由当前认证派生。预览/规则读取由服务端给出绑定，写入时与私密固定配置和已读事实精确重验。它不是权限 token；不得只相信客户端 `local=true`。

- 所有本地写入必须有匹配绑定；record.localTrial 与 input.localTrial 相同，进入原完整 inputHash/bodyHash。仓储验证 typed 结构及原 Source/Store 身份，事务锁 head 后拒绝在本地/正式 provenance 间切换。已有本地 head 即使请求删掉标记，也不能被正常编辑、重新适配、材料建议或其他平台请求改成正式。
- `localTrial` 是跨版本持续 provenance，不是一次性 SheinImport/SheinSalesMapping；清理旧操作绑定时不得清除它。Preview 当前返回新 TargetInput 字面量的路径必须承接/重验 provenance。严格 BFF 的本地 merchant/record 分支保留该完整字段，并允许明确的“无官方连接/无官方有效期”形状；不能为了通过现有 merchantSchema 的官方正数/日期要求而伪造值。普通无 marker 的字段保持可省略零值，不回填或重算既有 record/hash；未知 marker 拒绝。
- 更新后的 `ValidPlatformRecord` 只在闭合 localTrial 分支接受本地记录，核对 provenance、非其他平台/非拆色、试用 rule authority、非 Ready、无 payload；普通分支保留原官方准入。本地 merchant projection 保留真实 StoreID/版本和 synthetic supplier hash，但官方 application/connection revision/expiry 不伪造为有效；它必须不满足正常 `ValidApplication`。
- 现有 `validImportMerchant` 官方判断保持。TargetService 以**受信 constructor 的本地绑定**调用明确的本地上下文校验，然后复用原 derive/save；不得把“AppID 有 offline 前缀”设成全局官方判断的豁免。
- 规则快照新增闭合 authority 标识：正常 RuleReader 为 official，本地 reader 为 local_trial；unknown 拒绝。该标识进入 rules hash。SHEIN native builder 对 local_trial 仍运行原结构/格式/库存/规格检查，但永远 `ReadyForUpload=false`、`SubmissionPayload` 为空；本地试用诊断明确不是官方规则/资格。不是仅 UI 隐藏上传按钮或生成 wire 后再隐藏。
- `PublicationReady`、上传计划 preview/continue、适配/材料建议、直接 upload/submission、发布/上下架 prepare/confirm 及 worker native 准入均拒绝 localTrial 记录/绑定，**在任何官方连接、probe、网络、工作流或账本效果前**拒绝。本地 module 本身不装配这些 routes/clients/workers。即使完整合成规则/图片 fixture、普通 SupplyChain 后来启用、删除请求标记或重放历史记录，也不能使本地历史可发布。

数据库 owner 恶意直接改正文并重算所有 hash 不在当前 threat model；沿原可信 DB/私密配置运维边界，不新增签名/IAM/通用 admission 平台。对正常 caller 越权、跨企业、篡改 marker、provider 可达和 durable store 的错误效果是本轮 Must。

## 6. 端到端流程、版本与持久化

现有带入与销售映射是互斥的一次性操作绑定，因此不强行合成一个新保存协议：

1. **基础 Target T1**：在固定 Source 下选择准确本地测试 Store 和 fixture 类目；复用原初始草稿构造，三个原 SKU 在基础稿中，库存/价格待带入。服务器给出完整受控 input；用户确认后原 `POST targets` 保存 revision 1。
2. **带入数值/文字 T2**：固定 Completion R1 和 T1 base record，复用 `POST shein-completion-preview`；选择标题/描述、只选择黑 M 的 stock=0、不选 CNY 价格。预览后确认，原 `POST targets` + expectedRevision=1 保存 revision 2；普通一次性绑定清理规则保持。
3. **销售映射 T3**：对 T2 同一 Target，复用 `POST shein-sales-mapping-preview` 和已冻结 #1227 算法，用户确认源维度/值映射；原 POST/CAS 保存 revision 3，两个颜色组、三条 SKU、0/空/空。刷新/重开及历史仍是同一 Target。不是三个商品或三份草稿；预期 1 head、3 immutable records、0 split、0 submission。

T1 在受信本地 constructor 内从固定 Catalog/Source 和 fixture 生成闭合 bootstrap 输入，保存前比较**完整**期望 input，不只比较 localTrial/来源等元数据。后续无 SheinImport/SheinSalesMapping 操作绑定的任意手工 edit 拒绝；T2/T3 继续原完整重派生，另核固定 R1、准确三行双射、无其他 SKU/平台/图片写入。复制 marker 不能添删 SKU、挪库存、夹带图片或改别的字段。这里是本地准入范围，不新增状态机或另一个保存协议。

每次只读 preview 不创建 command/head/record，不调用 provider。编辑选项清除旧预览。保存按原完整请求、原 key、expectedRevision、TargetPrepared 源授权重验和 Product DB 单事务写 record/head/command；不改事务、增加跨库 Saga 或异步 Task。

R1/Store/规则/fixture/source/基线绑定漂移时拒绝，提示明确冲突；不自动换 R1、店铺、类目或 key。R1 head 后续增加不改变显式选择的不可变 R1；来源归档/撤权沿原授权拒绝新写。Fixture 本轮冻结不做热更新；若版本变化，旧预览不可新写，须显式重新预览。重新配置合成 Scope/Store 不复用原 Target。

幂等沿原协议：同 key/完整 payload 已提交返回原准确回执；异 payload 409；并发首次创建只形成一 head，T2/T3 CAS 只一写成功。丢回执、超时、断网、重启保留原 key/完整 intent，GET target-command 核对，不盲目另建。历史 replay 返回的 localTrial 标记不能剥除，不能误显示为当前 head 或官方结果；无自动补偿/释放/发布。

## 7. 最小路由与资源边界

只复用现有 supply-preparations 的当前来源/批次/记录/head/历史/command GET、`POST target-rules`（只读合成选择）、`POST shein-completion-preview`、`POST shein-sales-mapping-preview` 和 `POST targets`。每条路由继续原 permission + fixed scope/source/store 的服务端校验；其他 Source/Store 不可读写。基本稿仍用现有 Target 保存，不新增任务创建/适配 operation 接口。

不挂载 operations/ensure/cancel、color-split、upload-plan、upload、resolve、发布/货架、Agent/图片生成/批准入口。原应用其他模块保留原路由，但不能消费本地 Target 作为执行/生成/发布输入。当前 `supplyChainAvailable` 不作为服务器权限；新增单独受控可见性让原 R1 页进入本地带入，不把 full SupplyChain 打开。

本地只读 Query port 从原 Target 的 ReadHead/ReadRecord/ReadCommand 与同一 Repository 提取/复用，沿原授权并复验返回值的固定 Source/Store/typed marker；不构造带 Rules/provider 的可写服务来“凑”读取。现 `target-command` 直接依赖 app.Targets.ReadCommand，必须改为明确读 port；既有 PrivateDraftReadRoutes 没有该路由，不能误认为已覆盖。`enabled=false` 仍挂授权 GET/head/record/原键读取，不挂 POST；历史回执保留原 fixture 和 local 标记，不冒充当前规则/当前 head。正常 full TargetService 可实现同一 Query port，只有一个事实 owner，无 fallback/双读。

不新增外部 endpoint/HTTP transport，合成 reader 无网络能力；不启动新的 Temporal worker。沿当前 2 MiB 请求/record、256 SKU/映射、10s Target deadline、原连接池上限。固定 fixture 有界、无输入规则文件解析器、无全库扫描或无限 cache。

## 8. 后续安装、原数据保留与回退

**本设计阶段均 NOT_RUN**。后续首先在保留库隔离副本验证，获准后才可操作原 35444：

- 从现有标准 builder 构建准确提交；维护窗口停止相关写入/恢复器，复用 16 库/roles、对象和 Temporal 备份还原入口与已经验证的 schema 等价比较。原商品/R1/主图、原失败任务和原 15.445282 点预留取新基线。
- 复用 `product-acquisition-init --enable-supply-chain --confirm-database product_acquisition` 的现有**完整** installer 合同，声明所有当前已安装能力；只安装缺失的现行 Target/operation/upload/官方回执等表及既有固定角色最小权限。即使表存在也不意味着挂载执行 routes。不得只手工建三张表或重跑空库初始化，不修改其他已有数据/owner/权限。
- controlledLocalDraft 的 Product capability 核验必须声明已安装 SupplyChain schema/权限，而 HTTP/worker 装配仍是上述窄子集；不把 native Verify 降级或伪报 schema 未安装。Marker 在原 body_json 内，无新增 SQL 列/表；旧无 marker 的合格普通记录保持语义，未知 marker 拒绝。
- 测试 Store 仅由原 Store owner 原 key 创建/核对，随后把准确回执 ID 写私密本地配置；独立持久化步骤逐项留原 key/receipt，未知创建先查原 key，不复制或回灌隔离环境 ID/record。此前已存在合格测试 Store 时只能明确选定，不自动改驾驶舱店铺。
- 先启动并验证权限/历史/图片/marker，再仅经浏览器显式完成 T1/T2/T3。原 AI/商业/数据服务和对象卷不替换；本批没有原预留释放或计费调用。已有后台 next_check_at 可能更新，数量/归属/结算不可变化。
- 功能关闭后所有新本地 preview/write 关闭，记录/marker 保留、不转成正式记录；正常授权历史/原键只读回执仍带试用标记。版本回退先停写，以同一维护 checkpoint 恢复受影响 Product/Store及实际被后台写入的相关库、配置和对象清单，使用已验证的回退构件。无 DROP/Down/自动删测试记录；若后续已有新业务写入，不能盲目恢复旧 checkpoint，须重新明确回退范围。

## 9. 验证与验收、评审停止点

复用既有 Go/native PG/前端与浏览器测试，不建设 runner 或新的验收平台。已有 #1227 映射算法证据只对未变代码复用；本轮实现后必须收敛四项 **IMPLEMENTATION_TEST**：

1. **准入与无副作用**：新模式缺省/非本机/Scope/Store/Source/R1错配拒绝；原 Agent/SourceMaterials 可并存，controlled app 不冒充 full 导致原 SourceMaterials 少挂 approval/Preparation 能力；新增路由白名单与 live 撤权；本地 native 标记、官方无效 binding、rule authority 贯通，直接 native upload/worker/plan/Agent 尝试均在任何外部调用/账本写之前拒绝。包含合成“资料全齐”仍无 wire、去 marker/换 Store/停用新写而 GET 原键仍可核对的反例。
2. **一份草稿与数据保持**：真实隔离 PG＋浏览器经原 T1/T2/T3 保存/刷新重开；恰好一 Target/三 records/零 split/零 submission，黑 M/黑 L/白 M，0/空/空；CNY 不变、Target 不冒写 USD；原商品、Source、R1、原图和批准库不写，原预留数量/结算及调用数不变。
3. **保存协议**：T1完整 bootstrap 输入比较及直 POST 添删 SKU/夹带字段/无操作 edit 的拒绝；T2/T3 原完整派生和准确三行/0空验证。同键/异载荷、并发创建/CAS、数据库中途失败原子回滚、commit ACK 丢失、重启后原键核对、预览后绑定/授权变化；完整 localTrial 字段/hash 贯通预览新 Input、BFF local merchant/record 分支、仓储/历史/原键，不能只比元数据或清理一次性绑定时脱标。原 key 已提交回执保持准确历史且不可作为发布许可。
4. **安装与退出**：同一保留副本通过完整 owner installer、capability 校验、新模式启动，原模块数据/角色/图/任务保持；关闭模式不升级记录，按 checkpoint 回退原业务能启动，备份与私密配置保留。原 35444 的部署和业务保存必须另获授权，不能以副本 PASS 代替。

架构正常评审最多两轮；finding 先按当前 Must/后果分类。仅跨租户/越权、错误发布/扣费、数据丢失或核心路径无法完成等真实 BLOCKER 重开设计；实现细节转上述 IT，理论 hardening/backlog 不无限加机制。只有独立结论明确 **IMPLEMENTATION_READY** 才可在后续授权下正式实现。本轮不自签产品验收、不合并、不部署或关闭 Issue。

R1 分类记录：T1 完整输入约束、SourceMaterials full 指针误判、local DTO/持续 provenance/停用 Query 接线均为 **IMPLEMENTATION_TEST，实施及合并前**；分别归入上述 IT-1/IT-3/IT-4，不作为已执行测试或已修生产代码。当前 0 设计 BLOCKER；独立增量确认后冻结，后续须在实现验证收敛这些 Must。

## 10. 当前配置与操作入口

仅供本轮获准的隔离副本；原实例不得据此自行开通。普通原有 manifest 不含 `controlledLocalDraft`，行为保持。副本开通时先用原 Store owner 的创建回执取得准确测试 StoreID，再增加 `controlledLocalDraft` 配置块：

- `acknowledgment`: `CONTROLLED_LOCAL_DRAFT_NO_EXECUTION`；`enabled` 默认 false。
- `organizationId` / `actorId` / `memberId`、`sourceId`、`productKey`、`completionRecordId` / `completionRevision`：必须等于 §3 固定引用，revision 必须为 1。`storeId` 必须为本次副本 native 回执 ID，拒绝原驾驶舱店铺；不得将副本 ID 回灌原实例。
- `fixtureVersion`: `issue1158-local-draft-v1`；`fixtureHash` 为编译内 `listingapp.ControlledLocalRules()` 的当前完整 digest。没有任意 URL/规则 JSON/模型选择。
- `assetDatabase`: 复用当前 `image_agent` owner 的 `supply_asset_runtime` 私密连接，连接上限 8；不得粘贴凭据到 Issue/PR/聊天。与已有 SourceMaterials/Agent 连接指向同一 Asset owner。
- 前端仅另开 `LISTINGKIT_CONTROLLED_LOCAL_DRAFT_ENABLED=true`，保留普通 SupplyChain 开关 false。入口为固定 R1 页的「SHEIN 带入预览」；只有该 Source 展示本地入口。

操作依次为：选择准确测试店铺 → 核对并创建基础草稿 → 重新读取目标 → 选择 R1 文字及黑 M 的 0 库存/本地 US 仓（不选 CNY 价格）→ 预览并保存 → 重新读取 → 明确选择颜色/尺码及三个枚举对应 → 核对三条真实 SKU → 确认映射并保存 → 刷新重开。页面单独展示已保存版本的 SKU/库存；原图仍属于原商品，平台图片继续待批准。

未知保存结果保留原操作键，只使用原命令查询和重试。关闭时保留完整固定配置并设 `enabled=false`，服务端只保留 Target/head/record/原键 GET；不创建可写 TargetService，也不构造 Rules/provider 来读取。安装过完整 schema 后不得直接删除配置绕过 capability 校验。安装与回退仍按 §8，通过现有 owner 命令和匹配 checkpoint，不执行 Down、密码重置、原数据复制或预留释放。

实现验证、准确候选、独立交付复核及运行入口记录于 #1158 / 同一主要 PR，区别离线、真实隔离副本和原实例阶段，不把本节说明作为产品验收结果。
