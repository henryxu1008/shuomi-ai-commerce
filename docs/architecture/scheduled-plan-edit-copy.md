# 定时任务：编辑计划与复制新建

Issue #1297。Design Basis: Independent Architecture。状态：DESIGNING，等待本轮增量边界评审；不得据此提前修改生产路径。

## 用户结果与范围

用户2026-11-15明确要求：未结束计划可编辑后续任务设置；已结束计划可复制新建；保留已开始任务和执行历史，保存前重新核对时间与预算。实现后更新原35444，复用现有环境与验证；不调用真实模型、不发布商品、不释放原预留。

复用[定时任务合同](ai-workbench-scheduled-tasks.md)、[单店日报](scheduled-store-daily-report.md)及[删除合同](scheduled-plan-deletion.md)。本决定仅替代原“只能改时间、来源/次数/单次预算不可修改”这一局部产品限制：新值只约束修订后的未来任务；原任务原报价/预留/资源回执不变。Figma不作为界面权威。

- 同一计划内类型不互换：商品内容可换商品、目标、语言/处理字段；日报可换店铺；两者均可改时间、次数。
- 原有项目/店铺会话背景、模板与身份仍由原owner读取并授权，不让客户端提交ExecutionSpec、成员、模型或金额。
- 已结束计划不可重新启用；“复制新建”带入可读设置，等待显式核对和保存，独立ID、计数从0开始，不复制任务/历史/报告/预留。
- 不新增表、持久化owner、调度器、恢复扫描器、兼容适配或第二账本。不改原35444已有计划进行验收；本机真实页面停在保存前，受控保存/并发在既有夹具验证。

## 当前owner及接线

`ScheduleManager/ ScheduleForm → BFF ai-workbench合同 → 原schedule quote/create/PATCH → aiWorkbenchApplication → 原aiworkbench Store事务 → 既有Temporal Synchronize`。

计划、控制回执、occurrence仍由Workbench PG单一owner负责；BusinessTask的ExecutionRequest、日报report_execution.Origin、报告和扣点各归原owner。原Start/Resume、人工Review、发送前权限和费用保护不变。

### 输入与确认

1. quote/create沿现有scheduledInput，增加可选`sourceScheduleId`（复制来源）及`expectedRevision`（完整编辑预览）。创建、复制、编辑身份互斥；复制/编辑都按当前用户和企业读取原计划，要求来源可读，类型相同，成员绑定一致。已删除来源不开放复制。
2. 编辑quote携带原scheduleId、expectedRevision及完整可改设置；重新通过现有buildProposal（只读确定性准备，不调用规划模型）、当前商品/模式/模型价目与预算计算，或日报当前店铺授权。项目/店铺FrozenScope和模板从原ExecutionSpec取回；不把有背景的计划变为无背景。
3. PATCH沿现有expectedRevision/rule/confirmation，增加可选`changes`（完整selection/goal或report，以及maxOccurrences）。未带changes仍是同一个规范PATCH的“只修改时间”，不是旧系统迁移或fallback；不引入第二路写入。UI“编辑计划”始终提交完整changes。
4. 现有基础confirmation继续绑定实际ExecutionProposal、规则及预算。完整编辑另绑定plan ID、revision及已消耗次数；复制绑定来源ID及其revision。编辑/复制的保存会重新按当前事实计算；配置、价目、来源、权限、修订或已消耗次数变化使旧确认失效，要求用户重核，不替换模型或自动发送。
5. 复制只带入原类型和可读业务设置，不继承已过期的一次执行日期或截止日期；用户明确填写未来日期。周期时刻可以回填，仍须预览。任何字段改变使预览失效；失败保留输入。取消/返回不保存。

### 次数与预算

- maxOccurrences仍指计划总次数；编辑后必须严格大于consumedOccurrences，不能重置/返还已消耗次数。含未决任务的计数仍算已消耗。
- 原once约束继续要求总次数1；已有消耗的周期计划不能改为一次计划。日报继续只允许once/daily、UTC+8。
- perRunUpperBound及按max计算的totalUpperBound继续用原定点整数合同保存并检查溢出。确认页另明确显示剩余次数及“后续执行积分上限 = 新单次上限 × 剩余次数”，不把新费率重新套用成历史实际消费，不承诺累计金额止损。
- quote和保存均不预留/扣点，不调用模型；实际每次任务仍由原资源owner按该任务的冻结请求报价、预留、结算。历史和在途任务保持其原预算、原预留与原调用身份。

## 事务、并发与在途保护

- 完整编辑仍在原ChangeScheduledPlan事务内，先scope/key锁、原回执、plan行锁，再核对revision、未删除/未结束、原成员/类型/owner和已观察consumed值。
- 先按旧规则完成现有有界漏槽记录；未完成时保留旧规则/业务设置和原操作键，不提交新设置。原SKIPPED/claimed历史不重写。
- 单次事务更新未来selection/report、目标、服务端ExecutionSpec、规则及预算/次数，递增revision并待Temporal核实；PAUSED继续PAUSED，其他合法未结束状态沿原PENDING_REGISTRATION。createdAt、creationFingerprint、consumed、activeOccurrenceId保持。
- Claim与编辑共用plan行锁和revision：Claim先提交会改变consumed，编辑保存拒绝旧观察；编辑先提交，旧revision的新Claim拒绝。原已绑定slot的重放先读occurrence/BusinessTask，不创建新任务。
- 同根因的准备失败路径`RecordScheduledPreflightReceipt`必须显式携带读取时revision；行锁后先回放已有occurrence，无回执时拒绝旧revision，再沿原`recordScheduledMissingBefore`校验规则/slot。迟到的旧准备失败不得按新内容写BLOCKED/SKIPPED占槽。
- 在途商品任务用其已保存ExecutionRequest恢复；日报用已保存report_execution.Origin恢复。类型、owner、member不变，所以原Dispatch/Reconcile分支与身份仍正确；切换未来商品/店铺不改旧任务原请求，也不清理其占用。
- 同一UTC槽仍只有原唯一身份，改期不生成第二次任务。未知结果保持activeOccurrence和资源预留；编辑不构成任何释放证明。

## 幂等与恢复

- 创建/复制沿原scoped key→plan ID与COMPLETED/REJECTED回执，同键同载荷重放只返回原结果；同键不同载荷冲突。每次显式“复制新建”得到独立新操作键，点击复制本身不发保存请求。
- 复制创建事务在原scope/key锁及已提交回执优先重放之后，对源计划加原plan行锁，重新核对来源revision、未删除、owner/member、类型及服务端来源内容指纹；与源编辑/删除共享同一行锁排序。来源在quote后改变时原创建键按现有确定拒绝流程处理，不能检查后无锁创建。来源引用只作提交前条件，不新增持久来源owner/表。
- 完整编辑的命令指纹绑定严格解析后的全部PATCH载荷；现有只改时间/其他控制命令保持原规范指纹。成功及确定拒绝共用原锁与回执。没有新的状态或恢复owner。
- 重放已提交回执先于新报价/未来时间/源版本验证；原回执已绑定完整载荷，不拿后来计划内容重新解释旧confirmation，避免后续编辑造成旧成功无法恢复。
- DB提交后Temporal失败/响应丢失仍沿原“核实原操作”同键恢复；并发/修订冲突不静默覆盖。权限/依赖不可用不伪造确定未提交。前端只凭原commandAccepted:false退休未知键。

## 权限和外部副作用

原VerifiedIdentity、当前组织/成员及workbench.agent.use不变，来源和目标商品/店铺同时按当前权限核对；不可通过知道来源ID复制他人计划或读撤权内容。配置、模板、FrozenScope由服务端取原绑定，修改字段不扩大权限。

保存可能登记未来调度，沿已有授权/幂等/状态协议；本任务验证不在原35444保存或执行计划。实现/构建不授权合并或Issue关闭；用户已授权本机更新，但不授权真实模型、发布、预留释放。后台原预留相关恢复条件不改。

## 必需列权限与受控接线

当前runtime只允许修改时间/状态列，完整编辑不能沿用原权限声称可用。保持同一个owner与列级权限，新增且仅新增`scheduled_plans`的UPDATE列：`task_selection, goal_summary, execution_spec, report_spec, max_occurrences, per_run_upper_bound_micros, total_upper_bound_micros`。类型、owner/member、创建身份/指纹、budget_unit均不可改，不授表级UPDATE或跨库权限。

- 新装`GrantRuntime`和启动`VerifyRuntime`的准确列界限同步加入这7列；缺列/多列、表级UPDATE、角色或跨owner漂移仍拒绝。
- 复用`ai-workbench-schema-init`增加有界`--schedule-edit-permissions enable|disable`操作，要求原ai_workbench_runtime角色和现有expected-database/address/port准确目标；不运行InstallSchema、不建表、不修改业务行。
- owner事务内沿已有角色/跨owner检查及事务锁，识别“完整旧权限”或“完整新权限”两个明确安装状态，部分新增权限或其他漂移拒绝；enable仅授上述7列，disable仅收回这7列，前后按相应准确权限集合验证。并发/重复同一目标操作幂等，不做泛化权限重装。
- 原35444更新只短暂停止必要后台服务，保留配置/卷/身份，执行该原owner最小授权后启动新镜像；先只读核对无可触发真实模型的活跃计划，原预留恢复条件不变。不通过关闭已有功能掩盖启动问题，不运行全量备份或新环境演练。

## 必要验证及回退

复用既有Go/PG/HTTP及前端/浏览器夹具，不建立新验收平台：

1. 编辑商品目标/语言及日报店铺/次数，确认新未来设置持久化；旧BusinessTask/Occurrence/报告/active占用与已消耗次数不变；PAUSED不自动恢复。
2. 源/目标撤权、改类型/身份、低于或等于已消耗次数、旧quote、CAS冲突拒绝；Claim与编辑竞争至少一方拒绝旧观察，不重复任务。
3. 提交/响应丢失、后续再次编辑、同键并发与不同载荷重试；原回执可恢复，同key只一次有效变更。
4. 复制不自动保存，显式保存只新建一份，计数0、无历史/原预留；重复原key不多建，跨权限来源拒绝。
5. UI回填/重新核对、失败输入保留、窄屏/主题/键盘；现有已通过且未变的调度/资源矩阵引用，不重复。
6. 在原PG夹具使用真实受限runtime角色完成完整编辑；允许上述7列、拒绝身份/类型/创建字段及表级UPDATE；部分授权/额外权限拒绝。enable/disable重复及并发不改业务行；disable后按旧准确权限验证，旧版启动校验可通过。

无schema变更，原数据不迁移，installer仅增加上述最小权限操作。部署复用原实例配置/卷，先固定现行镜像和配置。回退须停止新后台，通过原owner收回本轮7列UPDATE并验证旧准确权限，再恢复原后台/UI镜像；不能只换镜像让旧启动校验拒绝。编辑后的计划/回执仍符合原schema，不回滚事实、不删除新计划、不撤销在途任务。旧UI不能解析新完整PATCH时保留待核对状态，不能另造新key或假称旧请求成功；需要时恢复新版沿原key核对。原预留保护与恢复条件不变，并以原记录只读核对证明。

## 第一轮评审处置

- Finding：runtime缺少完整编辑列权限，原“grants不变/仅换镜像回退”不可运行。Classification: BLOCKER（核心编辑/rollout无法完成）。Action：按上节7列最小授权、真实runtime验证及先撤回权限后退镜像处理；不扩为数据库治理工程。
- Finding：旧revision准备失败可能写入新修订同一slot。Classification: IMPLEMENTATION_TEST（本批交付前必须收敛）。Action：补同一事务的revision fence与原回执优先重放，复用PG竞态验证。
- Finding：复制检查后到创建前来源可能被改动。Classification: IMPLEMENTATION_TEST（本批交付前必须收敛）。Action：在原创建事务锁源行重核来源身份/类型/revision/指纹，复用原回执和拒绝栅栏。
