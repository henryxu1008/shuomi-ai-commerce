# 四步供应链与原任务继续上传

Product Decision: **PD-SUPPLY-FOUR-STEP-CONTINUATION-2026-11-03**<br>
Execution: [#1079](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1079)<br>
Design Basis: **Independent Architecture**<br>
Admission: **IMPLEMENTATION_READY / FROZEN**（2026-11-03 独立 R1 通过，无剩余设计 BLOCKER）

## 1. 用户结果、范围与优先级

2026-11-03 用户明确批准先更新 Figma 和供应链设计依据、独立评审通过后再开发，暂不合并和部署。当前企业获授权运营人员从“我的数据”选商品，配置一次上传任务；资料缺失时补全，使用 AI 时按当前策略审核并应用，再明确继续原任务，最终看到真实逐项平台结果。

Must：四入口为待上传、待补全、待审核、已上传；待上传内四步为选择商品、选择目标店铺、配置规则与智能体、确认提交。补全/审核是条件分支。固定配置能在刷新、重新登录或 worker 重启后读取；继续准确商品与目标，成功项不重发，UNKNOWN 沿原 owner 核实。保留未绑定的本地补全草稿；它不是平台草稿或可上传资料。发布方式按真实能力显示，平台受理、审核、草稿、上架请求及在架分别展示。已上传只读，SHEIN 首次人工上架从现有店铺商品路径消费原合法能力。

Should：复用 Console、原编辑/审核/结果组件和 SUMI 令牌，保持检索与返回位置，浅/深色一致。

Out of Scope：新平台、新模型/Agent、通用规则/审批/调度平台、系统到点发送 scheduler、自动重发、已上传跨平台二次发布按钮、历史迁移/兼容/回填、真实店铺/付费模型/保留实例操作、合并/部署/关单。平台原生定时上架按现有合同消费，不等于本系统定时上传。未开放能力不伪造入口或成功。

Accepted Risks：沿当前各 owner 合同，本轮无新增已接受风险。运行/真实平台/用户验收不由开发自检签发。

## 2. 当前依据、Figma 与 Threat Model

引用 [最终 UI/IA Authority](../product/final-ui-ia-authority.md)、[供应链](my-supply-chain-shein.md)、[未绑定草稿](supply-unbound-completion-drafts.md)、[SHEIN 发布](shein-publication-flow.md)、[领域边界](project-boundaries.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md)。本决定替代其中五个顶层阶段、已上传再次发布入口和审核后重新配置的产品投影；原版本/授权/事务/幂等/UNKNOWN/官方能力不变量保持。

Figma Authority：`tg48P46SSXl6TBy9lZwg63` / 当前可见页面 `31:463`。原位更新介绍 `926:569` / `926:359`，待上传主状态 `3507:3143` / `3507:3338` 及同组初始/配置状态，补全 `2667:1948` / `2667:2252`，审核 `2738:359` / `2738:951`，上传记录 `2738:655` / `2738:1247`。图中设计示例不是运行事实。

保护当前 Must 涉及的风险：跨企业/操作者/成员/店铺引用；客户端替换目标、资料或批准；并发继续造成重复执行；取消/重启/响应丢失误判为未发送；AI 内容未经合法批准送出；预算/发布方式错误；旧成功回执套用到新资料。外部平台可用性、完整市场调研、理论侧信道和通用运维自治不纳入本次。

## 3. 现状、owner 与调用链

main `c6f53e1a928862e556bbf0a1883c2af03dbedfb7`：`preparation/operation.go` 只有 adapt/optimize/upload 独立命令；原 Temporal operation workflow 执行一个准确操作，missing/review/unknown 是该命令终态，不能清零重跑。`app/supplychain/stages.go` 聚合五阶段，`supply-page.tsx` 混合配置和各阶段操作；删标签无法形成持久继续合同。Review 批准与 Apply 是独立副作用，CompletionDraft 不是 TargetRecord。

| 事实 | 当前唯一 owner / 本轮复用 |
| --- | --- |
| 来源、固定成员、canonical Product | Product collection / sourcing / catalog |
| 不可变候选及平台资料 | Listing completiondraft / record |
| 上传计划、逐项下一步和原操作引用 | **现有 Listing preparation**，仅新增有界计划，不复制商品、批准或平台正文 |
| 候选、批准及 Apply | Product Review / Asset，保持当前字段范围和批准策略 |
| AI 模板、报价、预算及 UNKNOWN | 当前 Agent / Commercial / Resource |
| 身份、Store连接/服务/原生能力 | 当前 Auth / Store，凭据仅在 server-only 执行句柄 |
| 外部 intent/attempt/fence/核实 | 原 Listing Submission，计划不签发发送许可 |
| 持久执行 | 原 Temporal operation workflow，不新建 scheduler/reconciler/runner |

Contract → implementation → injection → consumer：`preparation.UploadPlanService` / repository 窄 port → `integration/persistence/listing/preparation` 当前 Product DB UoW → `app/supplychain` 组合当前 owner 结果 → `currentapplication` 显式注入 → 原 Supply HTTP/BFF/Console。分流与幂等属于 preparation，App 不新增通用策略引擎；原适配、优化、上传命令继续复用，计划不是第二 Submission。

## 4. 四步与页面职责

1. **选择商品**：本人当前企业采集/自有批次；服务端固定完整成员，“整批”不在当前页截断。
2. **选择目标店铺**：当前可访问 Store，显示平台/站点/经营模式；换目标使预览/校验失效。已创建计划目标不可变，明确新建任务而不覆写渠道。
3. **配置规则与智能体**：必选当前已准入规则；AI 可选，仅出现支持当前字段/平台的已启用模板。规则内 AI 自动补全和优化都走原报价/预算/候选/批准，不以第三步开关代替批准判断。无真实能力显示原因，不据原型补建规则市场。
4. **确认提交**：展示准确目标、固定成员、规则/模板版本、发布方式及预算；“确认并开始”创建原任务。所有外部上传继续消费原完整准入，来源/目标缺失时禁用并说明原因。

待补全按原任务筛选并区分未绑定/已绑定。保存不等于继续。“确认补全并继续”展示原任务及准确已保存版本，重新校验后进入审核或上传确认。未绑定草稿先选真实 Store、显式带入，不自动读取最新内容覆盖。

待审核复用真实差异及 Accept/Edit/Reject/Apply。“审核并应用”完成后显示“确认继续上传”；批准、应用和上传分别确认，拒绝不上传。重新生成需新的原 Agent 明确操作与预算，Continue 不重跑模型。

已上传只读原资料与回执，显示店铺、平台/站点、提交方式、平台处理状态、原版本和观察时间；受理不写已上架。保留原核实入口，查询不算再次发布。SHEIN 首次上架从店铺商品详情引用原 source/record/receipt，复用 `SheinPublicationService` 与现有组件；缺原绑定不开放动作，不放宽模式/冷却等条件。

waiting/ready 合并为“待上传”投影，内部事实/校验保留；计数由服务端同一完整范围产生，处理中/失败/UNKNOWN继续可见。四入口不是必须全部经过的线性流水线。

## 5. 有界持久化与事务

当前 Product DB 的 preparation owner 新增 `listing_preparation_upload_plans`、`listing_preparation_upload_plan_items`，仅显式 installer 安装并配置最小 serving 权限/readiness；startup 不 DDL，不操作保留库。不满足 schema 即 fail closed，不转换/回填旧计划。

计划保存不可变配置：Org/Actor/Member、PlanID、command key/hash、PreparationID/revision、可信 Store binding（platform/site/mode/connection/service revision）、固定 SourceID/source version 集合、已准入 rule ref/hash、AI template/quote/mode revision、原生提交方式及适用字段、创建时间。不保存 credential/JWT、商品正文、永久授权或自行计算的最终费用。

逐项仅保存编排引用：SourceID、计划/item revision、当前阶段原 adapt/optimize/upload operation ID、准确 target record/revision、Review/Apply/Asset 回执引用及待用户动作。平台结果从原 owner 投影，不存可独立签发成功的布尔值。其他目标计划可以引用相同来源，原 Submission fence 跨计划防重复。

创建计划/成员共享一个事务；同 scope+key 同载荷返回原计划，异载荷409。逐项推进用 plan/item revision CAS；在**同一 Product DB UoW** 中记录下一步引用并创建原 Operation/items，提取现有 operation persistence 的 transaction-local 插入行为，不能先写计划再跨事务创建命令。准确项/阶段/批准版本只绑定一个子命令；其 key 从 PlanID+SourceID+阶段+确认版本稳定派生，worker 重试不换身份。

显式选择ID数组单请求≤1000；空数组的整批选择沿原服务端完整成员固定、100项分页及既有deadline，不新增1000总量截断，不要求手工拆批。列表≤100，审核沿原一次≤20。请求≤2MiB，普通命令/读沿10s，预览可沿现有18s投影上限。报价按原 owner 有效向量检查，失效须重新明确确认，不静默替换 current。

本次rule ref/hash精确指当前已准入标准配置及TargetRecord内保存的官方规则证据，不假定已有多套自定义规则owner；原型里的货号/定价/AI补全菜单未被本次授权为规则市场开发。只有已接通的原能力进入可用配置。

原PrepareOperation对多数平台会锁current target head；本次transaction-local插入必须消费逐项准确确认的record ID/revision，并在同事务比对head/来源/Store归属，不能预览后仍只读最新head。计划发布方式及原生时间字段必须与该准确TargetRecord一致，AS_DRAFT不能实际发送LISTING；更改方式须先由原Target owner保存新版本并重新确认，发送时不改payload。

原HTTP create-operation的AuthorizeOptimization（模板、报价、执行模式资格）必须提取复用到计划推进的同一准入入口；直接走domain UoW不能漏过该检查。原worker的当前授权与模型预算消费继续保留，计划记录不替代有效报价。

## 6. 继续合同与状态

原 `/supply-preparations` 模块增加创建/读取/列表计划、`POST /upload-plans/:id/continue` 、停止后续步骤及原 key 读取。Continue 接受命令键、expected plan revision、可选明确 SourceIDs 和完整预览 hash；hash覆盖固定逐项revision、准确已保存 TargetRecord/revision、原生提交摘要和原操作状态，由服务端重新读取比对，承载用户确认动作；客户端不得指定下一阶段、成功、预算、trusted binding 或 provider 请求。服务器从计划和原回执决定下一步。

| 当前事实 / 用户动作 | 下一步 / 持久效果 |
| --- | --- |
| 首次确认且尚无目标资料 | 固定配置，事务绑定原 adapt；不支持自动适配的平台进入已有资料编辑器 |
| 缺资料或图片批准 | 待补全，保存准确 target 后由用户继续；不重置原 adapt 终态 |
| 已齐备，计划指定 AI 且无原 AI 操作 | 事务绑定 optimize，原准入/报价/预算重新核对 |
| AI pending/running/UNKNOWN | 显示原状态/核实，Continue 不另建模型运行 |
| 候选未批准/Apply | 待审核，原 owner 审核/应用，不上传 |
| Apply完成并生成新target | 重新校验，不足回补全；原模板不再次执行 |
| 全部资料/批准齐全且明确继续 | 事务绑定准确 upload，消费原 send permit/权限/Store门禁 |
| 已有 upload pending/running | 返回原引用，只 Ensure 原 workflow |
| upload UNKNOWN/成功/已保留远端资产 | 原核实/结果，不新建upload或换键。部分资产成功而刊登失败按现行平台恢复合同 |
| 明确未发送失败 | 仅原 owner 有适用恢复合同且用户另行明确确认时处理，本轮不放宽 |
| 取消 | plan行内 cancelled + revision CAS 阻止未绑定后续步骤；已创建原操作保持原进度，已发送继续由原owner核实 |

Continue 重读原计划/子命令和准确 owner refs，不以最新 head 替代用户确认版本；别的编辑改变head则409重新预览。补全、Review.Apply等保存不宣称与计划跨库原子；Continue读取准确回执并经本地CAS绑定，发送前原owner再核对。无法证明的事实保持 unavailable/unknown。

## 7. 并发、失败、恢复与授权

- verified Org/Actor/Member+原scope限制所有计划读取/继续；管理员不隐式获得他人的来源，新成员不继承原执行权。create/continue需supply.manage；启动上传还需listing.submit及当前Store/服务。原Agent权限和预算沿原执行owner检查。
- 事务前失败无子命令，回滚无部分引用。COMMIT ACK丢失只原key查询；同key重放返回原准确回执，不重新绑定新版本。同revision不同key并发只有一个子命令，另一请求冲突或读回原事实。
- DB提交但Temporal启动失败/丢响应：显示待启动/待核实，使用同child OperationID的原EnsureExecution。查询/刷新不自发创建下一阶段或调用模型。外部UNKNOWN不解释为NoEffect。
- worker从原operation加载durable subject实时授权；撤权停止未发动作，原已发核实不签发新mutation。依赖临时不可用不冒充永久denied/取消。原target fence不被多计划绕过。
- 切换企业/用户取消请求、清空scope输入，旧响应不进入新身份；预览期间上下文/版本变化使旧确认失效。Continue不自动Accept/Apply/批准Asset，失败/取消不推断未发。

## 8. 平台能力与后续上架

本地草稿只保存CompletionDraft/TargetRecord；提交到平台按原生合同；平台草稿仅在已有真实路径时出现。TikTok LOCAL_REPLICATION的AS_DRAFT/LISTING沿原合同；SHEIN上传/审核/首次上架分开；TEMU/Walmart不从统一按钮推导新草稿API。

平台原生定时上架保留原字段/时区，如SHEIN适用模式shelf_way/hope_on_sale_date；无该能力不开放时间输入。本系统定时发送未开放，不作为核心提交前置。

已上传不提供跨平台再次发布，不删原来源/渠道。新目标从待上传明确新建并重新校验。店铺商品的首次上架沿原receipt/精确SKC/version/模式/当前授权/冷却/UNKNOWN，只移动消费者，不改`SheinPublicationService`规则。当前店铺商品列表是平台观察投影，不能要求首次上架前先同步出该行；原已上传只读详情提供带source/record/receipt的店铺商品详情深链接，服务端重新核准原回执归属。该详情复用现有上架组件，可在列表尚无观察行时合法打开，不从平台ID猜测SourceID或权限。

## 9. 实施与必要验证

一个Writer、一个分支/主要PR。独立评审IMPLEMENTATION_READY后：先实现preparation计划/UoW/installer与必要TDD/现有隔离PG；再接App/HTTP/BFF/runtime和四入口，复用补全/Review/Apply/结果及店铺商品首次上架；完整候选执行适用CI及一次独立最终交付检查，交出启动/操作入口和限制，不签发产品验收。

必要证据：同键重放/异载荷冲突、同/异键并发一步一命令、事务晚失败/COMMIT ACK丢失、原Ensure恢复、租户/Actor/成员/Store替换与撤权、准确版本/规则/报价漂移、直接上传/补全后继续/审核Apply后继续、部分成功与UNKNOWN不重发、未绑定草稿不误上传、平台草稿/定时上架不扩能力、服务端四入口计数/深链接/组织切换。仅复用现有框架，不新增故障注入平台或验收工具。

## 10. Legacy、评审与停止点

Legacy decision: **EXTRACT** 当前Preparation/Operation/Temporal、Console、资料编辑、Review/Asset/Submission合格能力；**RETIRE** 五阶段顶层导航和已上传二次发布消费者。无历史迁移/兼容、双读双写、第二商品/批准/发送事实源。

独立架构评审 R1：**IMPLEMENTATION_READY**。Reviewer `/root/supply_design_review` 独立检查实际 owner/持久化/Temporal/Review/Store 调用及 Figma 浅深色四步、补全/审核继续与确认稿，无剩余设计 BLOCKER。准确版本/原生发布方式、同事务与完整选择、原优化准入/标准规则映射、首次上架合法深链接四组为 IMPLEMENTATION_TEST，已在§5/8/9明确，须在本次实现和必要测试收敛。详细评审依据及准确blob/HEAD写[主要PR #1080](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1080)。本次冻结不代表实现或产品验收通过。

findings先按AGENTS.md分类；只有新的BLOCKER重开冻结设计，非阻塞实现细化沿原PR收敛，不按理论hardening扩范围。独立最终交付检查在完整用户路径形成后执行，当前NOT_RUN。

真实平台/模型/实例/安装/用户验收：**NOT_RUN**。合并/部署/关单：**NOT_AUTHORIZED**。
