# B1：受保护成员单业务角色的原生条件写入

日期：2026-10-29。Design Basis：**Independent Architecture / B1-only increment**。
状态：**IMPLEMENTATION_READY；本地源码实现与隔离测试已完成，未部署**。执行与接单记录为 [Issue #954](https://github.com/henryxu1008/shuomi-ai-commerce/issues/954)。设计准入本身不授予执行权限；用户随后明确批准“本地正式实现与隔离测试”，见末节。远端 fork、push/合并、发布/部署及真实权限变更不在本轮范围。

## 用户结果与冻结依据

拥有当前企业 MemberManage 的管理员，在既有成员详情中，为已有受保护成员增减一个同企业业务角色。追加角色的模块集合必须恰为 `store-browser`；撤销遵循原设计对已分配同企业自定义角色的规则。其他原生角色不变，尤其不得恢复已被原生控制台撤销的系统/平台角色。

本增量只替换旧设计中不安全的完整角色集 provider 写回合同。继承 `openbrowser-platform/design/store-workbench/protected-browser-role-grants/README.md` 的 M1–M7、Must / Threat Model / Accepted Risk 和已完成两轮的 REVIEW；B1 以外的有效结论不重复评审。系统角色展示已由原独立补丁处理，本设计不再修改它。

UI 继续使用原成员详情和权限角色页面，Figma authority 为 `tg48P46SSXl6TBy9lZwg63` 的 `1541:386` / `1536:473`。没有新增页面或视觉要求，本轮没有重新核对 Figma 像素。

范围外：管理员默认获得 browser.use、受保护成员整组替换/移除、任意权限字符串、新 IAM/授权表/后台协调器、provider 自动升级或部署、schema/ACL、账号初始化、迁移、数据清理和浏览器环境操作。普通账号试用、原关闭 UNKNOWN 与 Ant PR #17 的关闭修复保持独立。

## 根因、实测依据及证明边界

应用读取含平台角色的快照后，原生控制台撤销该角色，应用再写回完整旧角色集，就会重新授予平台权限。本地 reservation、冻结列表、再次读取和事后补写都不能关闭这个原子性缺口。

固定官方 ZITADEL `v4.17.1 / a9311b8c702531832575351a663e98a2242778e5`，复用原生 UserGrantWriteModel、事件、Eventstore、PostgreSQL v3 pusher 和原生 SQL 的隔离验证已完成：[结果与限制](https://github.com/henryxu1008/shuomi-ai-commerce/issues/954#issuecomment-6491395339)。

- 原生无条件完整列表写回复现了平台角色重授。
- 同一 READ COMMITTED 事务内检查序列再 append 的候选仍失败。
- 同一 SERIALIZABLE 事务内检查精确 aggregate 原生序列并 append，撤销先提交时得到原生 `events2_pkey / 23505` 冲突；最终只有撤销事件。
- 已撤销后再调用拒绝；条件写入先提交、之后原生撤销的情况保留业务角色且不恢复平台角色。四项定向用例通过。

证据只覆盖原生追加边界；完整 Commands 包、RPC/HTTP、原生 permission/precondition 和 ListingKit adapter 未执行。正式实现仍须补针对新增合同的测试，不重跑已完成且未变化的 spike。临时 fixture 已停止。

## Owner 与调用链

| 合同/事实 | 唯一 owner | 实现、注入与消费者 |
| --- | --- | --- |
| 原生成员、角色与 grant revision | ZITADEL 原生 usergrant event stream | 原生 command/write model 和 eventstore；scoped read 与条件命令 |
| 企业业务角色定义与模块 | 现有 Membership RoleStore | 现有 live-write authorizer 与窄角色校验 |
| 一次命令、dispatch、结果证据 | 现有 Membership ReceiptStore | 原 JSON payload、revision CAS、target reservation 与 audit；不是授权事实 |
| provider 请求/响应映射 | 现有 ZITADEL Membership adapter | 原注入与私密凭据；不新增浏览器直连或 service token |
| 可操作资格与结果 | Membership / 既有成员 UI | 原 MemberManage 和 exact-org 路径；Ant 仍消费实际 permission ID |

调用链：既有成员 HTTP 命令 → Membership 准入与回执 → 既有 adapter 的新条件分支 → provider 原生条件 command → 原生 Eventstore/PostgreSQL → 已提交结果 → 原回执/audit。应用不直连 provider 数据库。

## 原生读取与单角色命令合同

以下为待实现的 provider 合同；不是声称官方 v4.17.1 已提供这些 API。

1. `GetAuthorizationRevision` 接受 authorization ID 和期望 organization/project/project-grant 作用域。instance 从已认证原生上下文取得，不能由应用任意指定。按原生读权限核验后，从同一原生 write-model 观察返回 instance、organization、project、project-grant、authorization、user、active 状态、role keys 和非零 `nativeSequence`。
   读取请求的 project-grant 使用显式 presence：未提供时，由原生端从精确 organization/project/authorization 的模型发现真实 grant，并以该真实 grant 执行原生权限检查；显式提供（包括空字符串表示项目拥有方）时必须精确相等。返回的真实 grant 必须冻结到后续条件写入，写入始终精确匹配，不能把缺失值当作全部 grant。此处为最终实现复核所发现 B1-GRANT 的合同澄清；未改变 M1–M7、事实 owner 或状态/副作用边界。
2. roles、状态和序列来自同一 write-model 观察；不得把最终一致的目录投影与另一次 MAX(sequence) 拼成一个版本。未知原生事件导致完整 aggregate head 与 write-model sequence 不一致时 fail closed。`Aggregate.Version` 的 semver、changeDate 和应用 observedVersion 摘要均不是 CAS token。
3. `ChangeAuthorizationRoleConditional` 仅接受上述精确作用域、authorization ID、`expectedNativeSequence`、一个 `roleKey`、ADD/REMOVE，以及既有 dispatch ID 作为无持久化的请求关联值；不接受完整目标角色集。uint64 序列在 JSON 中用规范非零十进制字符串，避免 JavaScript 精度损失。
4. provider 继续执行原生 `user.grant.write`、project/grant ownership、主体/角色有效性检查。应用继续执行 MemberManage、exact-org、保护主体、角色归属与模块准入；原生端不另建 ListingKit 角色定义事实源。客户端不能凭序列取得额外权限。
5. provider 从与 expected sequence 完全匹配的原生模型计算一个 key 的增/删；其他 key 逐值保留。已存在的 ADD、缺失的 REMOVE、无效/不活跃 grant 均不写入，不隐式创建/删除 assignment。保留原应用角色数量及请求大小限制。

ListingKit 客户端的 `expectedVersion` 可继续作为 UI 乐观版本，但服务端须用该版本重新核验同一 authoritative revision，随后冻结 native scope、sequence、方向、key、before/after roles 和 dispatch ID。新字段进入原有受限 JSON payload；不增加表、索引或 DDL。冻结后 adapter 只发送一次条件命令，不重读、重算或退回旧 UpdateAuthorization。

## 原生事务、提交与通知

条件 command 在原生 owner 内执行以下一次性流程：

1. 按上述 scope 加载并核验原生模型，确认模型 sequence 等于请求 sequence；复用当前原生 precondition/permission 检查，不宣称把其他 aggregate 的授权撤销也纳入新 CAS。
2. 创建一个 **SERIALIZABLE** 事务，在该事务 snapshot 中读取精确 `(instance_id, aggregate_type=usergrant, aggregate_id=authorization_id)` 的完整最新序列。它必须同时等于 expected sequence 与冻结模型 sequence。
3. 用原生 UserGrantChanged 事件和现有 v3 pusher，在**同一事务**中 append。复用现有事件 PK、约束、字段命令和队列事务，不增加序列表或锁服务。该路径不得自动重试、rebase 或改用新角色快照。
4. 只有数据库 commit 明确成功，才向原生订阅者发布映射后的事件并生成 APPLIED 响应。响应绑定原 scope、dispatch、expected sequence、新 sequence、authorization ID 与原生 changeDate。

当前 `Eventstore.PushWithClient` 在调用方尚未 commit 时执行 `notify`，因此不能直接用它完成第 3 步后再在外层 commit。最小改动是在原生 Eventstore 内抽取既有 push/map 行为，让新条件方法拥有事务及 commit；新路径先 push/map、后 commit、再调用原 `notify`。原有调用者语义不改，不向应用暴露任意发布入口。没有发送前通知或提交失败后的成功响应。

通知不是新持久化事实源。提交后进程退出或通知/响应未完成时，已提交原生事件仍是事实；应用按原协议保留 UNKNOWN，不再次发命令。现有原生消费者的恢复方式不因本增量获得额外承诺；不为此新建 outbox、补写器或可靠消息平台。

## 明确冲突与 UNKNOWN：必要的有界回执增量

代码核对发现，现有 `commands.go` 把所有 writer error 保留为 dispatched；`operation.go` 的 EventReject 仅允许 ready。这不能直接表达“原生已确认零写入”的条件冲突。既不能把冲突伪装成 APPLIED，也不能放宽通用 EventReject 来释放旧 UNKNOWN。

provider 为新条件方法返回结构化 `VERSION_CONFLICT_NO_EFFECT`：只在序列不匹配且尚未 append，或已确认事务因序列冲突而终止、没有任何提交时产生。包括精确作用域、dispatch、expected sequence、方向和 role key 的请求关联值。`23505` 仅在命中本次 usergrant append 的 `events2_pkey` 时可归为该结果；不能把所有 unique violation 或任意 HTTP 409 当作该证明。`40001` 必须是数据库明确的事务中止。提交/连接结果无法判断时绝不产生该响应。

Membership 仅对新 business_role_grant/revoke 增加 `EventProviderNoEffect`：

- 只允许 `dispatched → rejected`，要求当前 revision、同一非空 dispatch ID、原 scope/target/native sequence/key/方向与可信 adapter 解析的零写入结果全部匹配。
- 原结果证据写入同一 receipt/audit 事务；已持久化 rejected 后才释放既有 target reservation。相同 idempotency key 始终读回该终态；刷新后由用户显式发起新命令，不自动换 key。
- 不接受通用 error、超时、上下文取消、后续角色观察、人工声明或其他 provider 的 409 作为依据。旧 invite/replace/remove、已完成操作、错误 dispatch 和没有新证据的 UNKNOWN 都不能走此转换。
- 证据保存失败、commit 结果不明或进程在保存前退出，仍保留原 dispatched/UNKNOWN 与 reservation。Verify/重启不补发，也不重新查询 provider 来伪造原请求结果。

这是一处明确的新转换，仅用于新条件方法的可验证零副作用结果；旧 EventReject 与 UNKNOWN 不重发不变量保持。继续使用原 phase、payload、CAS 和 `role` audit 类别，不增加 schema。

| 故障/交错 | 结果及恢复责任 |
| --- | --- |
| 下发前版本、资格、scope 或权限变化 | 原 ready 拒绝，外部写入 0 次 |
| 原生撤销先提交 | 条件比较不符，或本次事务冲突回滚；明确零写入结果按上文落回执 |
| 条件写入先提交、之后原生撤销 | 条件命令成功；后来原生事件为最新事实，不补写旧角色 |
| provider commit 明确失败且已确认零写入 | 仅满足结构化冲突合同的结果可终止；其他错误继续原保守处理 |
| commit/响应/本地回执保存不明、重启或取消 | 原 key UNKNOWN；原生事实可独立读，但不能据此补 ack 或重发 |
| 同 target 多 key、同 key 重放或错误 dispatch | 现有 reservation/CAS/指纹拒绝；无第二次 provider send |
| role definition 或 actor 权限变化 | 继续既有 scoped/live-write 合同，不增加跨 aggregate 的新原子性承诺 |

## 最小验证与准入

复用已完成的四项原生追加证据；正式实现按 TDD 只补新增合同必需证据：

- 原生 read 的 scope、roles/sequence 一致性；错误企业/project/grant、无权主体和系统角色请求在应用入口被拒绝。
- 新条件 command 的并发撤销路径、精确冲突分类，以及失败/冲突不通知、commit 后才通知。原生公开 handler/precondition 必须真实接入，不能以直接事件测试代替。
- 新回执转换仅接受完全绑定的零写入结果；错误结果/旧 kind/UNKNOWN 不释放，保存失败和重启不重发。复用既有 ReceiptStore PostgreSQL 测试能力，不建设新验收工具。
- 窄 adapter 请求没有完整角色集、无旧 API fallback、无自动重试；准确依赖版本不具备新合同则入口保持不可用。

本增量须进行一次针对 B1 的独立检查：原生版本 token、比较/append/commit/通知边界、零副作用结果与回执转换。不重开其他 M1–M7 或正常关闭设计。独立检查没有 BLOCKER 后，才可标记该设计 IMPLEMENTATION_READY；这仍不等于真实环境、正式 provider 分支/发布或部署授权。

Legacy decision: N/A（本次为设计文档，未消费或修改登记为 Legacy 的实现）。复用的是当前原生模型、事件/pusher、原生权限检查、Membership 单次下发及回执 CAS；ZITADEL / Membership 各自保持 owner。新条件命令只供新单角色通路消费；不得加入旧完整列表写回 fallback、第二事实源、历史回执转换或数据迁移。

## B1 增量独立检查记录

2026-10-29，复用已有独立复核者 `pr956_delivery_increment_review`，只读检查上述合同、固定原生源码与当前 Membership 实现。被审候选 SHA256 为 `742D139F410947F4285B1E7C98A7BFDA5BCEEFEB9D9BA9675D9F550D41EE9EA5`；随后仅更新本文准入状态与本记录，合同未变。结论：没有新增 BLOCKER，B1-only 设计达到 IMPLEMENTATION_READY，不代表实现/权限/运行验收或 provider 开发、发布、部署授权。

两项 IMPLEMENTATION_TEST 阻塞未来实现切片合并，不阻塞当前设计准入：

1. M2/M4/M5：真实公开 handler/precondition、禁止重试、失败不通知、commit 后通知。追加层 spike 不能替代该调用链证据。
2. M5：零写入结果与 kind/dispatch/scope/sequence/key/方向完整绑定，结果证据、`role` 审计和 reservation 释放原子保存。当前 ReceiptStore 只对 completed 写审计，实现必须有界补齐新拒绝结果的审计；旧 UNKNOWN、错误响应或保存失败继续保持原保留语义。

## 2026-10-29 本地实现交接

用户明确授权本地正式实现与必要隔离测试。后端开工基线为 `de55cf5f205f91fae246429be33eebf5bfee70e7`，本地分支为 `codex/b1-native-conditional`。配套 provider 固定官方 v4.17.1 基线，本地源码提交为 **`82ff277bc320d9b0da51d9a987271962841d753a`**；它不是官方发布，也未推送至任何远端。仅凭原 v4.17.1 版本号不能认定具有这两个新原生接口。

本地源码入口：provider 原 `zitadel.authorization.v2.AuthorizationService` 中的 `GetAuthorizationRevision` / `ChangeAuthorizationRoleConditional`；应用原成员路径下 `POST .../:member_id/business-roles/grant` 和 `.../revoke`，body 仍仅为 `{role, expectedVersion}`，原单企业头、唯一操作 key 与 MemberManage/live-write 门禁不变。真实原生 grant/sequence 由服务端发现并冻结。未增加 GUI 授权入口；既有严格回执消费者已识别新 kind/step 与绑定的原生 ack。

开发验证已覆盖：原生事务提交/通知及旧 Push 回归（使用仓库声明的 Go 1.25.11、默认 vet）；真实 Commands/precondition/PostgreSQL 的角色增减、撤销交错和拒绝路径；不同项目拥有企业的非空原生 grant 发现、增减、并发撤销及错误/显式空 grant 拒绝；应用原回执/audit/reservation 原子保存、审计失败回滚、旧 UNKNOWN 不重发；精确 HTTP 路由权限；已有 UI 回执解析及类型检查。完整 provider 启动、真实 bearer/IAM、管理员真实授权、GUI 新授权流程和环境联调均 **NOT_RUN**。

最终独立源码复核发现 B1-GRANT（空 grant 假设阻断合法企业路径），按同一设计修复并补 RED→GREEN；增量复核已关闭该 finding，无剩余具体源码阻塞。该结论只覆盖本地源码阶段，不是用户、运行或上线验收。共享实例、已有 schema/ACL、数据和管理员权限未改；原 spike 未重跑，新临时 fixture 在保存证据后停止。Issue 保持 OPEN / In Review，远端 CI 未运行。
