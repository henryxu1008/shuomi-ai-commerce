# SHEIN 候选类目规则查询与只读解释：C3-2

状态：**DESIGN_CANDIDATE / NOT_IMPLEMENTATION_READY**。Design Basis：Independent Architecture。
Refs #134 / #128。本文件是 design-only 合同候选；D1/D2/D3 已获产品认可，独立架构复核和实现尚未执行。
已有 C3-1 / #1076 的 Ready、实现和 CI 不覆盖本新增合同；本次不操作保留环境。

## 1. 用户结果与 Authority

在已有 SHEIN 美国站资料流程中，用户选择一个候选叶子类目，让现有“平台上架资料智能体”
查询并解释该类目的要求。用户能区分当前已保存类目、候选规则事实、模型解释及未评价条件。
查询不改变已保存或页面内未保存的资料；采用类目仍通过原编辑、人工保存和正式校验。

依据：

- [本轮最小设计候选与用户范围选择](https://github.com/henryxu1008/shuomi-ai-commerce/issues/134#issuecomment-6580341645)：
  用户明确选择“SHEIN、一次一个候选类目、用户先选类目，Agent 查询解释”。
- [独立文档准备接续](https://github.com/henryxu1008/shuomi-ai-commerce/issues/134#issuecomment-6580509953)。
- [D2/D3 产品认可及 design-only Draft PR 授权](https://github.com/henryxu1008/shuomi-ai-commerce/issues/134#issuecomment-6581006724)：
  用户明确回复“认可 D2/D3，授权 Draft PR”；该授权不包含独立评审、实现、合并或真实操作。
- [平台上架资料 Agent](platform-listing-materials-agent.md)、[Commerce Tool](commerce-tool-foundation.md)、
  [Agent 运行合同](product-agent-runtime-contract.md)、[C2-1 / C3-1](product-image-facts-advice.md)。
- [当前领域边界](project-boundaries.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md)。
- [UI/IA Authority](../product/final-ui-ia-authority.md)：现 Figma `tg48P46SSXl6TBy9lZwg63`、非归档 `31:463`。
  仅沿原资料编辑器和材料结果页；本文没有重新读取 Figma 或签发布局/视觉验收。
  若实现改变页面结构/操作顺序，须核适用节点，不从旧截图推定许可。

调查基线：main `bee30be7dd322a1d5d5777038fa98d939a570545`。当前代码依据为
`Application.QueryRules`、`RuleReader`、`OfficialProductAccess`、`materials`、原 Registry / AgentConfig / Eino / DTO。
旧系统历史不是本设计前置；本系统现行权限、原键、事务、用量和恢复合同继续有效。

### 1.1 已确认的产品范围

| 决定 | 候选 / 状态 |
| --- | --- |
| D1：候选由谁选 | **已确认**：用户事先选择一个类目；模型不自行搜索、换类目或扩选。 |
| D2：首次查询覆盖 | **已产品确认**：查询类目基础模板、填写/图片要求；依赖候选属性取值的关联分支显式未评价。不是完整刊登诊断或“所有关联规则已查完”。 |
| D3：新版本与运行效力 | **已产品确认**：§5 版本表的新定义负责新执行；被替换的本系统 v3 原记录/已提交回执只读，不跨版本续跑/采用，不迁移或重写。本次取得单独明确认可，不从旧 D2 自动外推。 |

D2 明确本阶段的规则覆盖，不把未评价的关联组合记 PASS；其它未撤回 Must 保持。
若后续要求验证具体属性组合的完整关联规则，须先补候选下的明确取值合同，不能从基础查询推定完整诊断。
D3 是已单独认可的执行资格/恢复入口产品决定，仍须适用独立架构复核和后续实现授权。
本设计不声称存在旧系统历史承接义务，也不建立 legacy adapter；正常版本资格不得通过数据迁移或第二事实源解决。

Must：准确授权和绑定、真实规则/实际观察、未知与确定事实区分、只读解释不得保存/采用、固定预算、原键及 UNKNOWN。
Should：复用现 reader、结构化结果、schema 组件、Agent Run 和现页面，只补当前用途的窄接缝。
Out of Scope：C2-2 看图、模型选类目、多候选/跨平台扫描、属性自动映射、保存/发布、资产批准、额外 planner/reviewer、
新规则库/报告库/缓存事实源、通用参数/编排/状态引擎、旧系统兼容/迁移、部署及任何真实操作。
没有新增 Accepted Risk；D2/D3 是明确的阶段范围及版本资格决定，不把未覆盖能力记为风险已接受或实现通过。

## 2. 用户操作与只读用途

1. 从原 SHEIN 资料流程打开一个实际已保存、当前可读且可作为新执行基础的 Target record。
2. 在独立于草稿编辑值的查询选择中选一个类目。此选择不调用原编辑器的改类目/清属性操作。
3. 确认本次查询的原记录、店铺和候选类目；服务端 prepare 固定准确绑定/模式/模型配置，再开始原有界运行。
4. 查看结构化规则、覆盖/未知、Agent 带引用的解释。模型文字标为解释，不能替代原规则事实。
5. 用户另行决定是否回原编辑器换类目、保存并校验。结果页没有“采用查询结果”或直接发布动作。

不新增 Agent 产品、工作台或导航。原 `listing.materials.agent` 提供明确的只读查询用途；
同一 Agent 的资料编辑用途保持原 Target 写入授权和审核，二者不能只依赖 UI 按钮区分。

## 3. Owner 与调用链

| 合同 → 实现 → 注入 → 消费 | 责任 / 本轮新增接缝 |
| --- | --- |
| 用户选择 → App prepare → 真实 base/Source/有效 Product/Store reader → 原配置快照及 Run | 用户只选 base record / category / 原模式与模板；可信身份和事实由 server 铸造，选择进同一 fingerprint。 |
| 候选规则读取 → Supply `RuleReader` 的窄类目投影 → 原 `RuleStore` / `OfficialProductAccess` → 原 SHEIN provider | 复用类目、填写标准、属性模板读取与可信 ProductType 派生；不复制 SDK 或另建规则系统。 |
| 薄查询 Tool → 原 Commerce Registry → fresh principal / 原角色与资源授权 / audit → 原 Eino 观察 | 新准确 ref 和严格 schema；不借付费生成/Prepare/Save 冒充 read。 |
| 原 governed 模型 → 严格领域结果变体 → 原固定校验 / Run commit → recorded-output DTO / BFF / 原结果页 | 解释与可写提案分开；不调用 Target adopt，不另建运行或报告 owner。 |

原 Registry 按 Agent 的完整 allowlist 绑定；候选 v4 的两个精确 Tool 必须在同一注册集合中存在。
不能把含两个 ref 的定义分别绑定到只注册一只 Tool 的 registry，也不能把原 `len(tools)==1` 改成任意工具许可。
模型与领域准入再按本次绑定用途限制实际可调用 ref，不能靠提示词实现隔离。

## 4. 输入与强类型绑定

候选新增 `agent.Binding.CandidateRules`，复用已有领域专用 binding ref 的模式，仍为可比较 struct；
不在 `Action` 增加模型自由 Arguments，不把选择塞入 GoalSummary、ContextID、Knowledge 或可写 Draft。

| 部分 | 候选合同 |
| --- | --- |
| HTTP prepare | 路径选真实 baseRecordID；body 为 requestKey、expectedModeRevision、原模板选择及一个 `candidateCategoryId`。拒绝额外 Draft、可信身份、Store、ProductType、平台路径/凭据。 |
| Binding.ContextKind | `listing-candidate-rules-v1`；ContextID 仍是真实 baseRecordID，不生成伪业务记录。 |
| 原 Product 字段 | ProductKey / CatalogVersion / PublicationID / TargetPlatform 由原 Source/有效版本/apply receipt 核实；仅 `shein`。 |
| CandidateRulesRef | BaseRevision、BaseDigest、StoreID、Site、MerchantDigest、CandidateCategoryID、SelectorDigest；由服务端固定，完整成组出现。 |
| 固定摘要 | 覆盖完整 base、来源/有效版本、Store 绑定、候选类目及用途；不把摘要本身当授权。 |

候选类目 ID、商品类型 ID 和数字版本采用各自现行的规范十进制串与范围校验，
转原 SDK 整数时精确解析，不通过浮点重编码。record / Store / Product 等标识沿各自 owner 的既有校验。
候选 ID 在当前有权 Store 目录中验证为实际可用叶子；ProductType 仍由 `goods.ProductTypeForCategory` 推导。
浏览器预览目录不是执行授权，prepare、dispatch 和释放结果时仍核原 owner。

其它 binding 用途必须拒绝非空 CandidateRules；查询用途拒绝 Selection / StoreDiagnosis 等相邻 ref，不能混用上下文。
原普通 `listing-target-record` 不携带该选择。相同 key 改选择/用途/base 版本为冲突，不默默切最新记录。

App 将 query binding 映射回真实 base 的原 reader 参数，检查当时 head/revision/digest。
Gateway 已从 Eino 取得完整 server Binding 和 CallMetadata，无须模型追加参数或另一份 request-local 事实源。

## 5. 候选准确版本与结果分派

以下是供独立架构复核的技术候选，执行资格遵循已批准 D3；不是现有实现或部署升级授权：

| 项 | 候选版本 / 行为 |
| --- | --- |
| Agent | `listing.materials.agent@v4.0.0`；同一产品内 `materials-edit` / `category-rules` 两种绑定用途。 |
| 现 inspect | 保留 `listing.materials.inspect@v3.0.0` 的原输入/输出与含义，只用于普通资料用途。 |
| 新查询 Tool | `listing.materials.rules.query@v1.0.0`，read、SideEffectNone、unmetered；只用于候选查询绑定。 |
| policy / prompt | `listing-materials-policy-v4` / `listing-materials-prompt-v4`，均以用途校验，不由模型选择运行用途。 |
| 模型 action schema | `listing-materials-action-v2`；保留 tool / propose / interrupt 三种 Action，不添加通用 next node 或执行参数。 |
| DomainProposal.Kind | `listing-materials-v2`：一个固定 kind 下的关闭结果联合，继续复用现 Runtime 单领域 validator 接缝。 |
| 模板参数 | `listing-materials-config-v1` 不增加默认候选类目；候选属于每次请求，不能被模板/模型悄悄替换。 |
| Tool / UI 结果 | `shein-candidate-rules-v1` / `listing-rule-explanation-v1`，准确 schema 与 ref 同时准入。 |

`DomainProposal.Payload` 的候选联合：

- `purpose=materials-edit`：仅含原材料 Proposal 白名单内容；继续复用原 ApplyProposal / Preview / Target 校验行为。
- `purpose=category-rules`：仅含只读解释、规则引用和未确定项；禁止 CategoryID 写入提案、attributes、images、SKUs、URL、actionID 等写入指令。

用途取自不可变 Binding，模型 payload 必须匹配；互斥字段、未知字段、错误 ref 均拒绝。
查询结果可使用现 `human_review_required` 表示需人核对解释，UI 显示“规则解释已生成，待核对”；
不增加 Run phase / 表 / 状态机，不把成功解释塞到 interrupt/unresolved 伪装成中断。
缺依据、预算不足或模型失败仍按原真实停止/中断语义展示，不能记成解释成功。

查询用途的模型准入只允许新查询 Tool 和只读解释；普通资料用途仍只允许现 inspect 和可写白名单。
完整代码注册、配置快照、profile、模型准入、gateway、validator、DTO 必须引用同一版本/用途表，
不能只改 allowlist 或升级 prompt 字符串。content / title / image / 其它 Agent 不获得新能力。

## 6. 规则覆盖与窄读取投影（D2 已产品确认）

当前 `RuleReader.ReadTargetRules` 会读取当前输入的 SKU 属性组合并查询 LinkedRules。
不能对旧草稿只替换 categoryId 后整体复用：旧类目的属性值不证明它们适用于候选类目。

建议在原 Supply / Marketplace owner 内抽取必要的类目基础读取片段：同一 Store handle 的类目目录、
填写标准、属性模板与原 goods 图片约束。现正式资料完整读取继续使用原关联组合/分批读取，不放松或截断。
新投影不写回或清空原 Draft，不把可写入口包成只读，不复制规则常量或新建缓存库。

| section | 首阶段覆盖 / 事实含义 |
| --- | --- |
| category | 当前 Store 实际可用叶子、路径和可信 ProductType；非法/非叶子不是“该类目没有要求”。 |
| fill / attributes | 官方实际返回的类目基础模板与字段标志；缺失/未知标志保留 unknown，不能按 false 展示“无需填写”。 |
| image requirements | 复用原 goods 要求；若依赖当前 SKU 结构，绑定准确 base 结构摘要并说明适用范围，不宣称检查了真实图片。 |
| conditional linked rules | 未有该候选类目下明确已选属性组合，保持 `not_evaluated` 与原因；不请求/沿用旧类目组合，不签发“全部关联条件已满足”。 |
| readiness / publication | 不提供 readyForUpload、save permit、publish permit；这些继续属于保存后的原正式校验。 |

空属性组合在真实 provider 中的适用性和完整覆盖证据当前 UNKNOWN；首阶段不靠它补造关联规则。
本阶段按已批准 D2 交付基础查询，不能外推为具体属性组合的完整关联规则诊断；后者需新的明确输入合同。

## 7. Tool 结果、模型解释与实际消费者

候选 Tool 结果为关闭 schema：

| 字段 | 必须表达 |
| --- | --- |
| schemaVersion | `shein-candidate-rules-v1`。 |
| binding | 准确 base/source/原始与有效版本/apply receipt/Product/Store/site/merchant/候选 selector；不披露凭据或把模型值当可信身份。 |
| evidence | 精确 ToolRef、CallID、recorded audit 关联、规则内容 digest、读取时间、本地算法版本；官方规则版本没有依据时为 unknown。 |
| rules | 实际取得的候选类目、填写/属性模板及适用图片要求，取自原 owner 的有界投影。 |
| coverage | 每 section 为 available / unknown / not_evaluated / not_applicable，并附具体原因；部分已知不表示全体可用。 |
| unknowns | 本次缺证据/未覆盖项的关闭分类与说明，不混同依赖失败、无权限和已知空集合。 |

候选 Tool 复用现 `materials.MaxInspectionBytes` 的 512 KiB 输出上限；超限明确拒绝，不截断后说完整。
原模型实际 input/quote、Run 2 MiB、HTTP/BFF 整体边界照旧；profile 若更小按原限制停止，不能自动加预算或删依据。
没有真实平台规模验收，不能从沿用资源上限推定所有类目都已支持。

Agent 解释以 `listing-rule-explanation-v1` 给出有界 summary、带规则引用的说明及 unknowns。
引用须指向本次成功、正确 CallID/ToolRef/audit/binding/digest 的结构化规则。
固定 validator 验证完整引用与只读字段形状，不把自然语言语义判断伪装成确定性证明。
页面同时展示原规则依据；模型解释质量、完整性仍需独立实际验收。

原材料结果 DTO / BFF / 页面增加明确用途分派：普通材料保留 before/after/proposal/adoption；
查询结果只含规则事实、覆盖、解释、用量和原运行编号，不提供 after、adopt URL 或可采用 candidateHash。
内部 Run 可保留校验 hash 用于幂等，它不成为 Target 写入凭证。
复用现字段/文本显示组件，不把模型生成的链接、命令或 HTML 当可执行操作。

历史展示只读取原 recorded payload；缺失、损坏、原权限已撤销时按原不可用/遮蔽处理，
不刷新当前规则、重算解释或追加模型调用来“恢复”旧结果。

## 8. 权限、采用隔离与重新核验

继承现 verified identity / live organization、LocalAgentWrite / Agent use、Source/Product read 和 Store rules purpose。
server 从 base 解析成员资源范围；用户持有类目 ID 或某条规则摘要不构成权限。
查询前后核当前 base/head、Source/有效 Product、Store 绑定/连接/服务状态和成员权限；
复用原 Store 每次 credential 读取和版本校验，不能借只读 Tool 绕过原正式模型准入。

| 原模式 | category-rules 用途 |
| --- | --- |
| default / read_only | 可在原权限/配置/预算允许时查询与持久化原运行/解释。 |
| auto_approve / full_access | 同样只读；不因模式较高而自动保存、采用、生成或发布。 |

自动采用、直接 adopt API 和其它调用路径均在进入 Target owner 前拒绝查询用途；
即便伪造候选 hash、同名字段或人工确认标志也不能升级用途。
普通材料保存仍经原 Proposal / Preview / Source / Store / 资产 / current mode guards，未被此设计替代。
查询规则与正式保存时规则不同，必须按原 owner 的新校验报告，不以旧查询快照承诺可上架。

## 9. 请求、持久化与恢复边界

无新业务表、报告、调度任务或数据库 owner。选择属于原配置快照/Run 的输入元数据；
观察与解释通过原 Run + opaque checkpoint 的一次 CAS commit 保存。
Registry audit 与 Run 原有写入责任不改；不为此添加 outbox、Saga、自动 reconciler 或双事实源。

| 场景 | 候选合同 |
| --- | --- |
| prepare 后、claim 前中断 | 同 key/完全同选择找回原快照，按原准入再决定 claim；改 selector/base/version 为冲突。 |
| claim/commit 结果不明 | 原 Run/CAS/UNKNOWN owner 处理；不因本查询是 read 就自动另起模型调用或换 key。 |
| Tool 成功、结果已提交后读/重放 | 仅返回原 recorded 内容；当前读授权仍校验，不要求用户将旧 base 重新保存为 head。 |
| 平台查询失败/读取漂移/审计失败 | 保留实际失败或停止原因，不生成空规则成功、不采信无 recorded audit 的结果。 |
| 模型发出后失联 | 原 invocation/预留/UNKNOWN 原样保留，不结算、释放、重发或切模型。 |
| 人工明确更换候选类目 | 独立新意图，走新确认/原预算准入；不能借它重放原未知运行或把其费用归零。 |

Tool 内不调用 AI，保持 read / SideEffectNone / unmetered；外层解释仍由原 governed model 计量。
沿原 ModelCalls / steps / token / cost / deadline，工具读取沿原有界 timeout。
可以在现预算内走“查询观察→解释”；若准入/资料大小/剩余额度不足，就明确停止，不提高调用/费用上限。

## 10. 版本资格与全新系统边界（D3 已产品确认）

候选 v4 不在 v3 的定义原位加能力。已批准 D3 规定新执行/claim/resume/adopt 仅接相应新定义；
本系统被替换 v3 的原记录/已提交回执只按原格式只读，不跨版本继续或采用。
这会改变 v3 的新执行资格，本轮已取得用户单独明确认可；适用独立复核仍未执行，旧 #1076 D2 不作为本次批准。

该建议只涉及本系统自己的版本资格，不能推出旧系统数据、ID、profile、业务资产或账务承接义务。
不迁移/backfill、不包装旧 Service、不建 compatibility adapter/双读双写或第二事实源。
若确有当前外部可观察兼容义务，先列具体证据报告用户；没有证据不扩大兼容机制。
保留实例和资料不等于本轮有权访问、升级、迁移、删除或处理其中的运行。

Current owner：原 Supply / Listing / Marketplace / Store / AgentConfig / Run / AI ledger。
Legacy decision：不触碰已退休 owner；仅复用当前合法行为，不新增 legacy dependency。

## 11. 实现义务与必要验证（尚未执行）

| 层 | 将来获准实现后仅扩既有测试 |
| --- | --- |
| 绑定 / 配置 / Run | 完整 selector、相同 key 异类目冲突、元数据篡改、错误用途、跨组织/成员/Store、准确版本及 claim/中断找回。 |
| 原规则 owner / Tool | 实际叶子与可信 ProductType、空/未知标志、条件分支未评价、无旧属性值泄漏、来源/记录/Store 漂移、超限/超时/审计失败。 |
| 模型 / 固定 validator | 必须先实际查询、解释引用绑定、无写入字段/可执行内容、既有预算、失败/UNKNOWN 不重放。 |
| 正常应用 / DTO / UI | 原 Registry→原 owner→合成模型→只读结果；四模式均无 adopt/save，原编辑保存与图片/内容相邻路径保持；历史仅读 recorded 数据。 |

先失败回归再最小实现，沿现 fixtures、合成 transport、任务自有临时 PostgreSQL；不连接保留库，
不新建 runner/验收平台，不人工重复正常 CI 已覆盖的全量检查。真实 provider/关联规则覆盖/解释质量/产品试用另有证据门槛。
实际开发 diff 超出已批准权限、状态/恢复、预算或事实 owner 时停报，不在实现里临时发明新框架。

## 12. 本次交付与停止点

本次仅交付 design-only 文档。D1/D2/D3 已产品确认，准确 schema/版本按本表供独立架构复核。
独立架构复核 NOT_RUN，不能进入正式生产实现；文档检查不代替架构批准、实现或验收。
本轮已获原工作区提交、推送、创建一个 design-only Draft PR 及核对准确 HEAD CI 的授权；
证据维护在该 PR，不为滚动状态制造文档提交。尚未授权派评审、实现或合并，不借原 PR/main CI 记新设计 PASS。
后续如获明确授权，按适用高风险检查点复用原 Reviewer，不增加人员或重复派工。

C2-2 继续 DEFERRED / NOT_IMPLEMENTED。缺证据 UNKNOWN；运行、真实保存/恢复、保留实例浏览器、
模型质量和产品验收 NOT_RUN。#47 PostgreSQL UNKNOWN、#307 OPEN、#733 咨询 NOT_SENT，#134/#128 保持 OPEN。
本轮不部署、不真实启停/配置保存/采集/保存/恢复、不调用真实模型或平台、不发邮件短信；全部保留实例/数据/会话不动。
