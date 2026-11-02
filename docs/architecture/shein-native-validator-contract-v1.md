# SHEIN 原生资料公共诊断与材料 Tool v2 设计候选

Execution: [#34](https://github.com/henryxu1008/shuomi-ai-commerce/issues/34)；关联 #134。源码调查基线：`f936702227f871e122d478276868e63a6efcc4fc`。

- Design Basis: **Independent Architecture**，仅新增 native 诊断输入绑定、公共结果投影和材料 Tool 版本合同。
- Architecture status: **DESIGN_REVIEW_PASS / IMPLEMENTATION_READY**。R1 绑定设计 HEAD `40ae55c988437d016bfa43eaa53a2ebf23a769db`；0 BLOCKER、两项 IMPLEMENTATION_TEST，具体实施/CI/最终复核状态以 PR #1064 为准。
- 用户已认可设计及 D1、授权原唯一 Writer 在原工作区/PR 实现并复用原唯一 Reviewer 做必要增量及最终复核。仅合成数据/transport与任务自有临时 PostgreSQL；不合并、部署、关单或执行真实业务操作。CI、最终 HEAD 和滚动交付状态维护在 PR，不在本文倒写通过。
- 认可基线为本地设计提交 `6fa0dade9af60f11a8fc51e3c80e4a396569726c`；本轮仅记录认可状态与 D1，候选技术正文不改变。保留实例与其现有记录未执行切换。
- 原 #34 Must 不减少。本文不是已经交付的新能力，不签发运行、产品或发布许可。
- 设计/本增量唯一 Writer：当前会话 `01a17b25-eb0a-7123-bf5b-ea5c4b914071`；不另派 Writer/Reviewer。

## 1. 用户结果、当前价值和范围

使用者是有权查看本人当前企业 SHEIN 资料的运营人员，以及在同一权限和准确版本下运行的平台资料 Agent。

当前人工保存/适配、Agent Preview 和上传重建已经共用 `goods.BuildOfficial`。缺口在于：它们的 native issues/ready 尚未形成 #34 公共诊断合同；现材料 Tool 只披露事实与保存时 issues。用户和 Agent 无法把这个 Tool 的返回值直接当作带准确输入、规则版本和明确评估范围的公共诊断结果。

本候选交付方向：在原生规则 owner 上补一层有界纯投影，给现材料 Tool 增加版本化的公共计算结果；候选重验、人工保存和发送前规则检查继续同源。公共诊断不是新的审批或发送事实。

Scope：

- 仅当前已准入的 SHEIN native Target/官方资料输入；当前适用 site 为 `shein-us`，应用类型继续由实际 Store/rules 决定。
- native 输入内容绑定、独立规则/投影版本、真实 issues 与成功汇总结果。
- 现材料 Tool 的新准确版本、固定 Agent 消费与候选重验；一次现有 Tool 调用内计算，不加模型或 Tool 调用次数。
- 原生人工保存/Agent Preview/上传重建的规则来源及公共投影对应。
- 新执行与既有历史/回执读取的版本边界，明确保持原事实与 UNKNOWN。

Out of Scope：

- 重建规则引擎、Tool Registry、runtime、审核系统、权限体系、账本、队列、数据库或通用验收平台。
- 改 Product/Asset/Store/Target/Report/Submission fact owner，改 schema，迁移、回填、覆盖历史，双写或兼容旧 owner。
- 把原 Package 转成官方 Target，重解释 SHEIN_RECORD/SHEIN_TARGET_RECORD，恢复旧 ListingKit/Workspace/Task-first。
- 新平台、完整 Tool 市场、#134 其他提案/图片工具、#47 Agent 评测或 PostgreSQL 补跑。
- 新增发布、Apply、资产批准、模型或平台调用，增加调用、费用、时长或大小上限。
- 部署或切换保留实例；本文不对已有运行执行升级/停止/恢复。

没有新增 Accepted Risk。Must 的父级保留与本候选覆盖见 §11。

## 2. Product / Architecture / Figma Authority

- [业务智能体职责](../product/business-agent-responsibilities.md)：平台资料修改属于 Listing owner；内容属于 Review/Catalog；full 不是外部发布授权。
- [平台资料 Agent 冻结设计](platform-listing-materials-agent-v1.md)：准确 Target、Store/rules、批准素材、领域候选、四模式与本地采用；远端发布 Out of Scope。
- [SHEIN 供应链](my-supply-chain-shein-v1.md)：native 官方模型、准确 owner refs、live 授权和实际上传。
- [中立 Validator 合同](../../internal/marketplace/validator/README.md)：确定性、scope、结构化结果、零业务副作用，不授权提交。
- [Agent 运行合同](product-agent-runtime-contract.md)：固定验证、repair≤2、准确候选/输入绑定、未知结果不自动重放。
- [原离线报告](report-center-shein-offline-package-source-v1.md)与[原生资料报告](report-center-shein-target-source-v1.md)：两个来源和历史结果含义分开。
- [Greenfield](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)。

本轮不设计新页面、按钮、人工流程或页面命名。沿平台资料原设计引用的当前 Figma Product Authority（我的智能体/配置/确认及 Supply 资料页）；没有重新读 Figma 或执行浏览器/视觉验收。若实现必须改变页面职责，停止并报告，不能从公共诊断类型推导额外 UI。

## 3. 已查证差异与 owner

| 范围 | 现状 / 合同差异 | 候选处理 |
| --- | --- | --- |
| typed Request/Snapshot | Snapshot 强制真实 ObservedAt/ValidUntil；native RuleSnapshot 没有这些期限 | 不构造 now+TTL，不放宽 typed 合同；复用 BoundRequest/DiagnosticResult 的内容绑定与 not_evaluated |
| native 规则 | OfficialDraft 有 Issues/ReadyForUpload/可选 wire；OfficialIssue 只有 Field/Code/Message | 保留 goods builder；只投影真实计算结果 |
| 公共 checks | BuildOfflineChecks 要求非空；native 成功时 issues 为空 | 增加真实的整次 native build 汇总 check，不造逐规则 PASS |
| Tool | Disclosure 带 base.Result.Issues；executor 只读和序列化 | 新版本保留保存时问题，另给本次公共计算结果 |
| 精确观察 | 当前材料校验比较整个 Disclosure 的 digest | v2 稳定语义指纹排除读时元数据，完整核实际事实和诊断语义 |
| 版本历史 | readRun/LoadSnapshot 把存储版本和当前常量比较 | 分开有界历史解码与新执行准入，不能只改常量后让历史失读 |

这些是拟议增量必须处理的边界，不是宣称当前原生上传没有校验或当前系统存在已实测事故。

| 事实 / 行为 | 唯一 owner |
| --- | --- |
| 当前来源、Product/Apply lineage | Collection / Preparation / Catalog / Review |
| 批准素材及 inventory | Asset |
| Store 身份、规则读取及官方执行能力 | Store Center / 当前 RuleReader |
| native 规则及 canonical 官方 payload | Marketplace SHEIN goods |
| 中立结果标准化 | Marketplace validator |
| 材料候选及披露证据核对 | Listing materials |
| 不可变 Target/head/command/CAS | Listing record/target 与原 repository |
| Tool schema/审计/固定 allowlist | 现 Commerce Tool 与材料 invoker |
| run/checkpoint/配置/预算 | 现 Agent/Eino/configuration/Commercial |
| intent/发送许可/回执/UNKNOWN | Submission |
| 历史报告 | Report；不成为当前校验输入或许可 |

## 4. 候选合同版本

以下是已复核冻结的版本合同；保留实例不执行切换。

| 项 | 候选 |
| --- | --- |
| native diagnostic scope | `shein.official_draft` |
| native rule/projection version | `shein.official_draft.v1` |
| native input binding | `shein.official-input.go-json.v1` |
| Tool | `listing.materials.inspect@v2.0.0` |
| 材料 Agent definition | `listing.materials.agent@v2.0.0` |
| 材料 policy / prompt | `listing-materials-policy-v2` / `listing-materials-prompt-v2` |
| proposal kind / model action schema | 保留 `listing-materials-v1` / `listing-materials-action-v1` |
| 配置参数 schema / Agent ID | 保留 `listing-materials-config-v1` / `listing.materials.agent` |
| 离线 Package | v1.2/v2.2/binding v2 全部不变 |

proposal 允许字段、模型 Action 形状和模板参数不变，不能借新 Tool 输出扩大候选字段。新 Prepare 以当前显式定义生成新配置/模式/请求快照；不改既有 activation、模板、模式、额度或快照。

现 configuration schema CHECK 约束参数种类而非将所有快照锁定在 Agent v1；本候选不要求 DDL。读/执行校验中对当前常量的限制须按 §8 分开。若实现发现必须改 schema 或新持久化事实，停止并报告。

## 5. native 输入绑定与纯结果

### 5.1 服务端实际事实

调用者只从已获准 owner 取得事实，不接受模型或浏览器提交可信 org、版本、rules、ready、hash、批准或 URL。既有来源/成员权限、current head、Store 与 exact inventory 检查保留。

一次 native 计算的绑定至少覆盖：

- 目标 marketplace/site、诊断 action、native rule/binding version。
- 当前来源/Target 上下文的稳定摘要、目标 TargetID/expected revision；新建时尚无新 record ID，不伪造已保存 ID。
- Product key/effective version/publication、Apply 引用和实际 Product hash。
- merchant/store binding 摘要、真实规则数据 hash、实际 inventory hash。
- whitelist 后的实际 OfficialDraftInput（包括所有商业/内容字段，不仅候选改动字段）。
- 实际图像观察和 stock-proof 观察的摘要；远端转换结果变化产生不同绑定。

org/actor/member 只用于服务器的上下文摘要与外层授权，不在 Tool 输出披露权限或凭据。任何 digest 都不是授权证明或 CAS。Product/rules/inventory 摘要必须对应本次输入，不能仅复制调用者声明的旧字符串。

新 subject 使用固定 Go struct envelope 与标准 JSON/SHA-256，字段顺序在实现前冻结；map 按现 Go 编码规则、数组顺序有意义，nil/empty 不自行等同。组件摘要对实际分量检查编码错误；包身份不使用旧 Package codec，不造跨语言 canonical JSON 框架。

绑定 envelope 的固定顺序为：binding_version、marketplace、site、action、rule_version、context_digest、product_reference_digest、merchant_digest、rules_digest、inventory_digest、draft_digest、image_observations_digest、proof_observations_digest。分量使用检查编码错误的标准 JSON SHA-256，前缀统一 sha256:。不存在的 Apply 引用等按当前 typed 零值编码，不推断缺失事实。

组件序列化遵守原读取/输入的有界合同和取消/deadline；不为 full native facts 擅加一个会拒绝原有效输入的新总大小门禁。没有新增事实编码器持久化入口或第二数据源。

### 5.2 action、freshness 和结果效力

| action | 用途 | ReadinessBlockersAllowed |
| --- | --- | --- |
| preview | 当前 base / 候选的只读计算 | false；不影响用户读取诊断 |
| save_draft | 原生本地保存目的的诊断 | true；不消除 blockers，不替代原模式/保存授权 |
| publish | 发送前该准确输入的诊断目的 | false；仍不授予发送权 |

只支持本候选明确的 SHEIN/site/action/rule/binding 组合。旧 Package 不新增 preview 或 site 支持。

复用现 BoundRequest、DiagnosticResult、ValidateBoundInput、BuildOfflineChecks。ReadAt/EvaluatedAt 由 App 提供，纯函数不读时钟；二者须有序，但不进入内容 subject，不制造外部 freshness。

ExternalFreshness 固定为 not_evaluated、空 evidence/coverage；native v1 不扩展现 external_package_freshness。NotEvaluated 明列 live authorization、current head/CAS、规则新鲜度取得、Store publish permission、远端图片/提交许可和远端业务结果。外层虽可能已检查其中某项，纯结果仍不签发该项授权。

DiagnosticOnly=true。Ready 只覆盖 native 规则计算；即使 ReadyForUpload=true，SubmissionPayload 缺失或最终发送资格不满足仍由原上传门禁阻止。报告不提供 send_allowed/published/approved。

### 5.3 从原生结果生成 checks

只调用现 goods.BuildOfficial，不复制类目、属性、SKU、图片、价格或库存规则。一次计算的 native Result 可交给公共纯投影；不为生成报告重新运行一次 builder。

| check | status / 字段 |
| --- | --- |
| 汇总 `official_draft.build` | 仅按本次 native ReadyForUpload 映射 ready/blocking；说明为整次构建汇总，不声称逐规则 PASS |
| 每个原 OfficialIssue | rule=`official_draft.issue`；code=原 Code，paths=原 Field，message=原 Message，category=`shein.official_draft`，status=blocking |
| guidance / warnings | 无独立来源就 guidance 空、warnings 空；不推断修复建议或降低原 issue 等级 |

保留每条原 issue 及同 rule 下不同/重复 findings；排序使用现 BuildOfflineChecks，不丢字段、不合并不同问题。原生没有单独 warning 语义，不自行发明。出现 ReadyForUpload 与 Issues 明显不一致，返回 EvaluationFailed/零诊断，不能用汇总掩盖。

计算错误、未知 target/action/version、错误绑定或诊断超限均返回明确错误和零公共结果，不给空成功，不回退 typed 或 Package。公共结果有界编码沿现 2MiB 报告上限；Tool 整体仍≤512KiB，原模型/runtime/请求上限全部保持。

## 6. 原生业务与公共投影的接入

公共结果是已有计算的只读表达，不能成为新的事实 owner、保存政策或“序列化成功才有发布权”的条件。

推荐在当前 SHEIN owner 提供一个有界 native assessment 计算入口：它只调用一次 goods.BuildOfficial，返回该次 native 结果及本次输入身份。公共诊断由同一份 assessment 投影。assessment 是瞬态调用结果，不是持久化 proof、授权凭据或新状态机。

| 消费者 | 接入 / 不变量 |
| --- | --- |
| Target.preparePlatform | 取得原 native 结果写原 Target；issues、ready、wire、record/head/command/CAS、原写权限保持 |
| materials.Preview | whitelist ApplyProposal 后用同一计算；类目变化的 rule_unavailable、RequiresHuman 和原证据限制保持 |
| DomainValidator / adopt | 候选 Valid 的领域含义不变；不能用诊断 Ready 代替结构/字段/证据校验或人工确认；采用前及提交 guard 继续重验 |
| Upload | 图像/证明事实变化后同源重算；原 payload 精确性、live owner 检查、Store publish permission、intent/claim/fence/permit/UNKNOWN 保持 |
| UI/报告 | 现页面继续显示 owner native 结果；不新建页面。历史 Target/Report 不被写成新的当前诊断 |

若某调用需要公共诊断，投影失败就明确返回诊断不可用，不能称 PASS；无公共报告要求的原生保存/发送命令仍依原规则结果与原守卫判断。这里不存在失败后改跑另一套规则：native 计算已经是唯一规则事实，失败的是独立只读投影。不得因投影失败静默换 native 输入、重试另一 Validator 或跳过既有守卫。

如果最小实现无法区分诊断表达失败与原生准入失败，导致新增保存/发布条件或放宽原条件，停止并报告；不能以公共合同接线为由自行改业务政策。内外层适用的错误在实现前与既有类型逐项对应。

## 7. Tool v2、稳定证据和候选重验

### 7.1 一次读取与计算

Tool v2 沿同一 Registry/Invoker、fresh principal、现 read permission、audit/trace、10秒 timeout、deterministic 与 unmetered policy；没有新增调用或预算。None side effects 指业务事实不变，原审计记录保持。

输入仍只有服务器请求绑定的 exact binding，不接受任意候选 JSON、org、URL 或平台凭据。新输出是版本化 Disclosure：保留原事实与保存时 issues，新增必需的 computedDiagnostic。issues 保持“保存时问题”含义，不能被重命名成当前计算结果。

Tool 读取沿现 InspectionReader，必要 owner 访问不移入纯 Validator，也不追加平台请求。完整输出超限拒绝，不截成完整成功、不追加第二个分页/摘要工具或提高上限。新 prompt 解释两种结果及诊断 scope，原模型 Action schema/候选字段不变。

Tool只对本次 base 计算。候选由固定 DomainValidator 在 ApplyProposal 后重验，不能拿 base 的成功作为候选成功；runtime仍最多两次repair。Valid、Unresolved、RequiresHuman 与公共 Ready 分开。

### 7.2 稳定语义指纹

不能直接把带新 ReadAt/EvaluatedAt 的整个 Disclosure 再与稍后重读结果做原字节 digest 比较：时间不同会造成假漂移。

v2 新证据 ID 建议为 `listing-evidence-v2:`＋固定语义 envelope 的 SHA-256。语义 envelope 包括准确 ToolRef、原 binding/record revision、全部披露事实、保存时 issues、native subject、scope/rule/binding、checks/action policy/not_evaluated；排除 evidenceId 自身和仅表读取/评估时刻的 ReadAt/EvaluatedAt。哈希前验证元数据类型、有序时间及所有 public result 约束；不因此宣称 freshness。

DomainValidator 只接受当前 Tool v2 的 recorded audit、有效 CallID、准确 binding 和实际已披露证据。它重新读当前 owner 并重算相同稳定语义，比较完整指纹/内容，排除字段只限上述时刻元数据，不允许任意省略。若 fact/rules/assets/规则算法改变，base 指纹必须改变或 current read 拒绝。候选改变由原 runtime CandidateHash 独立绑定，并在 ApplyProposal 后用同源 native assessment 重验；它不改变未变 base 的 Tool 指纹，也不增加第二次 Tool/model 调用（R1 IMPLEMENTATION_TEST-1）。

记录的 Tool 输出仍保留真实读/评估时间；不改写历史输出使它“等于当前”。不能仅比 evidenceId 字符串而不检查实际完整披露，也不能以模型摘要替代 owner facts。

## 8. 版本切换、历史和 UNKNOWN

当前只改 Tool/Agent 常量会破坏旧 snapshot/run 读取：readRun、stored configuration 校验以及材料采用读取都含当前常量比较。该 sibling path 是实际增量范围，不得只让新执行成功。

候选建议：

- 新 Prepare/Start/resume/adopt 只准当前 v2 的准确 definition/tool/policy/prompt、原权限/模式/配置快照；不把 v1 观察升级为 v2。
- 历史配置/run/已提交回执的只读解析使用其保存版本的有限明确格式；只允许本轮实际的 v1/v2格式，不能变成任意版本或放宽 owner/member/digest/字段校验。
- 同 key 已存在 v1 run 时返回原只读事实或明确版本不匹配，不 Claim、不调用模型、不转新 key、不改 usage/预留。
- 已提交采用回执仍由原 Target command/准确候选与 scope 查回；不重跑当前 builder、切 head 或再次保存。
- Running、未知 provider 结果、历史费用/预留仍归原 owner，不重放、不结算、不释放、不伪装失败；新版本不提供绕过 UNKNOWN 的入口。
- 不保存/迁移副本，不引入 v1 可执行 runtime/Tool 分支；代码只保留当前业务资产的有界历史读行为，不恢复 retired abstraction。

**用户已确认的版本切换决定 D1**：v1 历史和已提交回执只读保留，未完成运行/未采用提案不跨版本续跑或采用；保留原事实与状态。此保留来自本轮明确用户决定，只涉及当前材料业务记录，不构成旧系统迁移或其他历史业务承接义务。若以后要求 v1 继续执行/采用，须先取得新的明确决定并核定安全和生命周期合同，不能在本候选内增加双执行版本或兼容 fallback。本文没有执行任何切换，也没有使当前 v1 失效。

这不是迁移授权。相关只读函数应将“存储格式校验”与“当前新执行准入”分开；新操作不得复用宽松历史读取作为执行许可。

## 9. 权限、事务、失败及副作用边界

- 继续 verified identity / effective organization、原 actor/member-private Source/Target、exact approved inventory、当前 Store access。报告或 Tool grant 不替代源权限；admin 角色不自动拥有他人资产。
- Rule/事实加载在各 owner；pure adapter 不取数据库、网络、模型、时钟或凭据。现 source/Store/Asset 检查和 deadline 不迁入 Validator。
- Target/Report/Run/Submission 原数据库与事务、幂等 key、fingerprint、CAS、commit guard、原回执负责方保持；无新事务/状态/后台恢复机制。
- 新 candidate/图像转换事实/规则输入产生新计算身份，不能以旧 subject 授权新的写入。
- response loss 仍查原 key；diagnostic 投影失败不是 provider no-send 证明，不影响现 UNKNOWN。
- read_only/default/auto/full 仍按原材料领域策略；公共 Ready 不能允许 read_only写入、绕过人工确认或提供发布权。
- 不修改任何已有数据、配置、角色、模型、费用、预留、实例、worker或session，不测试真实平台。

## 10. 最小实施落点与必要检查

拟议生产范围只围绕：

- 新 SHEIN native 纯投影/绑定文件 `internal/marketplace/shein/validator/official_draft.go`，及 goods owner 的一次 assessment 接点。为遵守现有纯 Validator 导入守卫，Product/Store/Source DTO 留在调用方，只传其实际内容的检查式摘要；goods owner 提供 native findings 和原输入摘要，公共 adapter 不反向导入 goods/Catalog/Store，也不修改守卫或 CI。
- 现 materials inspection/validation、materialsinspect schema/invoker、材料 definition/prompt/admission。
- Target.preparePlatform / Supply Upload 的一次 native 计算和共享表达接点；原 goods 业务规则不复制。
- 材料 HTTP 新执行/历史读取/采用回执读取，以及 configuration snapshot 只读校验与当前执行校验分离。
- 不计划改中立 typed Snapshot、旧 Package codec/规则版本、数据库、参数 schema、Registry、通用 Eino loop 或预算。确需改变即列具体缺口并报告。

获得实现授权后，复用现 contract/diagnostic、goods、materials validation/commit、Tool invoker、Eino domain/repair、材料 HTTP 采用/原键和配置 snapshot 测试。先补失败回归，再最小实现；不为每项造新平台或完整故障矩阵。

必要证明包括：同一准确输入确定性/原 native issues-ready-wire不变；真实汇总及不丢 issue；错误/超限无成功报告；时间变动不假漂移、事实变动必拒绝；base不能替候选；v1历史/原键回执可读且不能跨版本新执行；原模式、权限、commit/CAS/发送守卫不变。业务实现测试及独立复核结果在 PR 按准确 HEAD 记录；design-only HEAD 的既有 CI 不当作新能力验证。真实验收继续 NOT_RUN。

旧 #666、#904、#1049 和 main CI 只按其原SHA/范围引用。#47当前1.0.2共享PG继续 UNKNOWN；不因本候选增加已有PG记录。运行/真实保存恢复/浏览器/模型质量/产品验收 NOT_RUN。

## 11. #34 Must 与候选覆盖

| 原 Must | 候选承接 / 保留 |
| --- | --- |
| M1 UI/fixed/Agent同源 | native依旧唯一goods计算；公共投影明确可供Tool读取。旧Package同源单列，不将两种输入混为同一事实。真实消费者接入与对齐由本增量实现/工程复核核对，产品验收 NOT_RUN |
| M2 repair后重验 | 固定领域 Validator、候选hash、稳定披露证据、采用前及提交前检查；不以base report或停止repair代替候选有效 |
| M3 draft/publish语义不漂移 | action policy只报告保存目的；缺项草稿政策、自动采用、最终许可分开，guard/UNKNOWN不变 |
| M4 pure validator无业务副作用 | native计算与公共投影无I/O/时钟/业务写；外层事实读取/审计/保存/发送归原owner |
| M5 SHEIN及adapter扩展 | 复用中立结果与当前SHEIN规则，不新建跨平台引擎；未来平台不因接口存在而被验收 |

本候选未实现前，不能据此勾选父Issue完成。#134其余工具/PoC、#47质量比较不纳入新增Must或本候选前置。

## 12. 决定、准入与交接状态

| 项 | 当前状态 |
| --- | --- |
| 本文的版本、输入/结果/证据设计 | 用户已认可；R1 DESIGN_REVIEW_PASS / IMPLEMENTATION_READY |
| D1 v1未完成运行/未采用提案的切换策略 | 用户已确认：只读保留、不跨版本续跑或采用；尚未实施切换 |
| Independent Architecture Review | R1 通过；复用原唯一 Reviewer |
| 正式业务实现 / 业务回归 / 产品验收 | 原 Writer 已获实现与隔离回归授权；最终工程状态见 PR，产品 NOT_RUN |
| 原 PR / 准确 HEAD CI | 维护同一 PR #1064，准确实现 HEAD 的 CI 与最终复核分别记录 |
| merge / deployment / closure | NOT_AUTHORIZED |
| #34 / #307 | OPEN |
| #47 PostgreSQL / #733咨询 | UNKNOWN / NOT_SENT |

R1 IMPLEMENTATION_TEST-2：拆开有限 v1/v2 的历史候选/hash/原键回执完整性检查与 v2-only 新执行准入；历史投影可显示原提案/已提交回执，但不能因此开放 v1 的新 resume/adopt。此两项在实现/回归层收敛，不重开全局架构审核。实现授权不包含合并、部署、真实操作或关单。

Legacy decision: **EXTRACT**。
Reusable behavior: 现 goods、validator、Tool Registry/审计、材料证据/模式、历史读取和原 owner 的幂等/guard。
Current owner: 上述各当前领域 owner；本候选没有新持久化事实。
Cutover/deletion condition: v2经确认/独立准入及未来实施验收后，新的执行只用v2；旧当前业务记录只读保留。retired旧执行owner不恢复；不保留v1可执行兼容链，不做数据迁移。
