# Supply 未绑定店铺待补全草稿

当前字段扩展见 [结构化候选合同](completion-draft-structured-v1.md)，达到其增量准入后实施；原owner/事务/权限是复用基线。

Design Basis: Independent Architecture
Admission Status: IMPLEMENTATION_READY / FROZEN — 2026-10-12 独立高风险边界检查通过。

## 1. Outcome / authority / boundary

依据 [PD-SUPPLY-UNBOUND-COMPLETION-2026-10-12](../product/supply-unbound-completion-drafts.md)，#605 当前阶段允许真实保存未绑定店铺的候选标题和描述。当前 Figma 文件 `tg48P46SSXl6TBy9lZwg63` / 页面 `31:463`，待补全 `2667:1948` 的 2026-10-12 design context + screenshot 是页面依据；待适配沿用 `3507:1965` / `3507:3143`。

本设计只新增当前 Listing 内的 CompletionDraft，不把正式 `TargetRecord.store_id` 改为可空。原 [我的供应链 SHEIN v1](my-supply-chain-shein-v1.md) 的正式商户、来源、资产、规则、上传和 UNKNOWN 边界不变。此前缺 SHEIN 时停止在 Preparation 的阶段决定由本产品决定替代；独立只读合成试用设计不用于正式业务。

用户路径：已保存来源 → 原 Transfer 固定 PreparationSource → 查看原资料 → 编辑候选标题/描述 → 保存不可变草稿版本 → 刷新读取 → 后续选择真实店铺，显式带入保存内容，再走原 TargetEditor / Target.Create / Submit。

Out of Scope：新 canonical Product、修改来源、自动 Store 绑定/同步、自动审核或发布、AI/支付/provider 调用、通用草稿平台、后台 worker、旧记录迁移、无 Store 的 Ready/Upload。

## 2. Owner / contracts / injection

| 内容 | 唯一 owner |
| --- | --- |
| 原始商品正文与版本 | Product Catalog / SourcePublication |
| 原始来源与转入成员 | Product Collection / Listing Preparation |
| 未绑定候选标题/描述、不可变保存版本与命令回执 | `internal/listing/completiondraft` |
| 已绑定平台资料与校验结果 | 原 `internal/listing/record/target` |
| 资产批准、正式店铺/类目、外部提交 | 原 Asset、Store/SHEIN、Submission |

最小领域合同：`Save(ctx,key,SaveInput{sourceId,expectedRevision,content:{title,description}})`、`Read(ctx,sourceId)`、`ReadCommand(ctx,key)`、`List(ctx,preparationId,Query)`。回执有 immutable record ID、draft ID、revision、原 SourceItem、候选 content、createdAt、replayed；`status=pending_completion`，无 Store/official rule/approval/provider result 字段。

调用链：正常 Console/BFF → verified identity + live Organization/module grants → completiondraft.Service → 原 SourceSelector/Collection owner + 原 Catalog exact snapshot → completiondraft persistence 同 Product pool → 正常 JSON 回执。App/integration 只组合，不拥有正文。

启用为显式 `supplyCompletionDrafts` 能力，要求 ProductCollections、原 Product pool、当前 membership 与 live permissions，不要求 SupplyChain、Store 官方配置、Asset pool、Temporal 或模型。runtime、HTTP route admission、schema/grant verifier、BFF、UI flags 必须一致；disabled 不开放写路径。Product acquisition capability union 须完整覆盖所有已启用消费者，不能靠忽略严格 grant verifier 消除启动失败。

Serving 只验证 schema/grants，不能 DDL。新增显式 schema-owner 安装入口只创建本节三表及最小权限；新安装/受控测试可使用，现有保留实例的安装在候选完成后另行取得具体授权。原有 Compose 层、私有配置值、业务卷不重置。

组合细节：该 opt-in 独立组合原 Preparation core 和六项转入/来源路由，不要求 AcquisitionPlans/BrowserCollector。与 AcquisitionPlans 同时启用时复用同一 core 并避免重复注册；完整 SupplyChain 启用时复用其 app/core 和原完整路由。三个草稿表权限加上必要的原 Preparation 权限进入所有 Product 消费者共同的 RuntimeCapabilities union，旧 disabled composition 对新增权限仍拒绝。不能修改全库授予函数来安装当前实例，保留实例只用新增 owner 的有界 additive grant/verify 入口。

## 3. Persistence / transaction / idempotency

使用当前 Product DB 和 GORM/PostgreSQL，不引入额外 DB、缓存、队列或协调器：

1. `listing_completion_draft_records`：不可变 record，组织/actor/member、source_id、draft_id、revision、content JSON + hash、created_at。PK 含组织/actor/record ID；UNIQUE 组织/actor/draft_id/revision；source 外键含组织/actor，指向原 `listing_preparation_sources`。记录固定完整 SourceItem 原始身份与版本，不由客户端提交 ProductKey/Publication/version。
2. `listing_completion_draft_heads`：组织/actor/source 唯一，当前 record ID/revision、member。FK 指向本 owner 的准确 record 和原 source。唯一 head 的创建/更新使用 CAS；不得修改不可变 record。
3. `listing_completion_draft_commands`：组织/actor/key 唯一，member、规范 input hash、immutable record ID。命令回执不从 mutable head 重建。

三步置于一项原生 DB 事务。复用现有 advisory transaction lock 的“同组织/actor/命令键”锁，以及 draft/source 的更新序列；建立 head/record/command 原子提交。权限只需 records/commands SELECT+INSERT，heads SELECT+INSERT+UPDATE；不授予 DELETE、TRUNCATE、schema CREATE、owner、角色继承或跨库权限。

`draftID=StableID(org,actor,sourceID,"completion-draft")`；record ID 绑定 org/actor/key。同键同规范载荷重放原记录，不能返回较新 head；同键不同载荷冲突。首存 expectedRevision=0，后续为读取的准确 revision；并发不同键保存同 revision 只允许一个成功。命令 key 为 UUID，input 包含 sourceID、expectedRevision、完整候选内容。

COMMIT 成功但响应丢失、deadline、未知结果：返回 UNKNOWN；前端保留原 key/body/context，优先 GET 原命令，not-found/网络失败不允许换键或编辑后重发。原键恢复或相同 key/body 安全重试均重新核对当前权限；无定时恢复 worker。确定的 validation/conflict/permission 拒绝可由用户刷新并开始新动作。拒绝状态不修改已保存 head。

取消、进程崩溃或事务前失败均不留下部分记录。事务内重读原 source，核对 org/actor/member、PreparationID/CollectionItemID/原 Product/Publication/version 与授权 proof 完全一致。执行前及提交前检查 proof 有效、deadline 和授权，不以旧 session、浏览器 flag 或历史 module grant 代替 live owner。

## 4. Authorization / read behavior / bounds

复用 preparation.ContextAuthorizer、Collection.AuthorizeOwner 和 SourceSelector：读需要当前 supply.read + 当前 data read/owner；写再需要 supply.manage。org/actor/member 全部取 verified effective identity，与 token/上下文一致。source 不接受浏览器的 scope、Store、ProductKey、其他 member 或任意 catalog version 作为读写 authority。

每个 head、record、command、list SQL 都同时限制 org/actor/member。更换企业或成员、撤权、token 过期、来源不可读时拒绝，命令重放也不能绕过。服务在重放/按键读取前核对 owner；取出回执后对原 source 再 Select。列表按当前 owner + 固定 preparation/source 外键过滤，先检查当前 Data owner 授权，不泄漏其他成员/企业 payload。转入后 Collection 改名/移动/归档不改草稿来源，沿原 retained-source 合同读取。

输入严格 JSON，拒绝重复/未知字段、额外 query 和非法 ID。普通文本允许待补全的空标题/描述；限 title 1024 UTF-8 bytes、description 32768 bytes，完整请求最多 64 KiB；拒绝非法 UTF-8/control null。只作为 text/textarea 渲染，不执行 HTML。查询复用 bounded Query 和 keyset cursor，最多 100 条；列表总数必须来自同 scope 的实际草稿 head，不使用原型样例数字。HTTP body read timeout 3s，request 20s / domain 10s；不读外部 URL或下载来源图片。

## 5. UI / later Store path

- “待适配”在该能力开启时提供“创建待补全草稿”，保留查看原资料。
- `/workbench/supply/mine/missing` 在未配置官方应用时仍显示真实草稿列表、来源/批次、保存版本和编辑入口；顶部明确“未绑定店铺，平台规则待配置”。原 Figma 目标站点/缺失字段/AI 状态只在有事实时显示；未绑定的官方缺失项数为未知，不能由两项通用文本计算。
- 编辑器先读取原 source + latest saved record，未保存变化不冒充已保存。显示原始资料与独立候选字段；保存/响应不确定时按原命令恢复。组织切换取消请求、隔离 UI command state，迟到响应不得显示到新 scope。
- 完整 SupplyChain 启用后，既有 TargetEditor 在用户显式选择“带入待补全草稿 Rn”时，仅将该保存版本的 title/description 作为用户输入带入；现有 target 不自动覆盖。Local draft 不生成 Store merchant/rules、Asset receipt、ApplyReceipt 或 ready 状态。
- 正式 Target.Create 仍使用原 source/effective version、真实 Store、当前 official rule snapshot 和资产批准；所有必要字段由该店铺规则决定。TargetResult 创建失败不消费/删除本地草稿；创建成功保存独立正式 record，未绑定草稿历史继续可读。同源手动文本输入的合法性和来源产品 lineage 沿现有目标保存合同，不将本地 draft ID 当批准权限。
- UI/BFF API 只开放四项本地草稿读写/列表，不开放原 private-draft trial POST，不改变既有 Ready/上传的 capability gate。

## 6. Legacy / verification / rollout

Legacy decision: EXTRACT | RETIRE。复用当前 Preparation/Catalog 的 source proof、稳定 ID、GORM UoW、严格 body/cursor、同源 BFF/context intent 机制；不依赖旧 DraftService/ListingKit Workspace/task 或只读合成 Store，不新增双读/双写/兼容层。

TDD 与必要验证：空缺字段可保存、刷新准确读回、并发 revision CAS、同键不同载荷拒绝、同键重放旧 record、事务部分失败回滚与 COMMIT-unknown 原键恢复、跨 org/actor/member 和撤权/来源读取拒绝、payload/查询上限；真实隔离 PostgreSQL 下 schema/最小 grants 与 extra grant 拒绝；API/BFF 无 Store 可保存但原 target/执行未开放；UI late response/context isolation、保存版本与再次编辑、显式带入不覆盖正式 target；scope 内 Figma 视觉核对、必需 CI 和最终一次独立交付检查。

数据保留：当前实例原事实、备份、schema、grant 和命名卷在开发/隔离测试期间不修改。候选完成后交出新增三表、最小 grant、回退/验证步骤和镜像组合，取得本地安装授权后才能应用；回退停新能力后保留三表/所有记录，不删除或恢复旧全库覆盖新记录。真实 SHEIN/付费模型、合并、部署、Issue 关闭和用户验收分别保留其原授权与 NOT_RUN。

## 7. Independent Architecture R1

2026-10-12，独立 Reviewer `/root/native_auth_review` 实查 #605 当前正文、原 source/owner/TargetRecord 与 runtime/grant 组合，以 `3f049c8691c43747436a37e45caf3fb90d32b42e` 加 §2 组合补充（设计 blob `a9a0860702e66fdaff99ac96015c4ec005623a8b`）确认 **IMPLEMENTATION_READY，无设计 BLOCKER**。只复核本次新草稿边界，不重复未变化的 #654。

Finding R1: 保存、回读与原键恢复。Classification: IMPLEMENTATION_TEST；影响准确持久化、不越权和不覆盖并发更新 Must，合并及本地试用前须通过。短效 source proof 在提交及回执读取时重新核对 live scope/source；真实 PG 验证原子性、revision CAS、旧键旧 record、COMMIT UNKNOWN、撤权/member 替换。

Finding R2: 独立能力及权限组合。Classification: IMPLEMENTATION_TEST；影响无 Store/collector 可转入与保存且执行关闭 Must，合并及本地试用前须通过。落实原六 Preparation 路由、plans/full 去重、所有 Product 消费者同一 capability union、有界 additive grants；回退只撤本能力独有 grant，保留事实和其他消费者权限，严格 disabled preflight 不放松。
