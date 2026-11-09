# #1205 保存经营评估与人工决定引用：局部设计候选

Status: **PRODUCT_CONFIRMED（D1–D3）/ IMPLEMENTATION_READY / IMPLEMENTED_PENDING_FINAL_REVIEW**。Design Basis: **Independent Architecture（局部增量）**。

调查基线：main `cc2262a2e6e9326b76fdf9c89d54e9e9f9103dd6`，2026-11-08。设计提交时为 pending；随后用户明确现有 Codex 对准确 HEAD `df6ee8c7decf18fe2c33da465cd183e61eda74a1` 完成评审并点赞即本轮架构评审通过，并授权当前唯一 Writer 在原工作区/PR #1209 按冻结设计先补 IT1–IT5 回归再实现、自检、推送及自动复核。该准入及授权覆盖本局部增量；不包含额外派工、合并、部署、关单或真实操作。最终实现 HEAD、CI 与评审证据维护于 PR，不以工程证据替代运行或产品验收。

## 1. 用户结果与范围

依据 [#1205](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1205)、#1074、#1073、#1201，以及用户本轮“复用现有 Research，明确服务端计算、评估保存与回读、人工决定引用、权限和原键恢复”的授权。产品 UI 复用原项目候选的经营假设及决定区；Figma 仅参考业务流程，不增加顶层中心。

用户在同一已准入候选内：

1. 填写经营条件，查看服务端参考试算，显式保存；刷新及重新登录后回看同一评估。
2. 显式选定一份评估，以其准确条件和来源记录原人工决定及理由。
3. 修改条件后另存新评估；旧评估和已引用它的决定不覆盖、不改写。

首期保留当前 Amazon US 单候选、单规格、每单一件、一个明确履约方式；USD 销售/成本、可选 CNY 单件采购及人工 CNY→USD 汇率。采购意向数量不等于库存，参考采购成本不等于供应商有效报价。没有评估仍可沿原路径人工研究和决定。正利润不自动批准。

不做自动研究/找货/报价、额外模型调用、自动采购或发布、多 SKU/多币种/组合订单、批量测算、通用财务平台、团队共享、旧库迁移或试算历史回填。不以 #1201 全部审计完成作为本设计的前置。

## 2. 已核对能力与复用决定

以下均为上述基线的 **CODE_VERIFIED**，不是本轮执行测试或浏览器验收：

| 现有入口 | 现有行为与本次复用 |
| --- | --- |
| `internal/product/research/model.go`、`service.go` | 项目私有候选、所选 link/EvidenceRef、人工决定、Execute/Lookup、命令校验和原键 Replay |
| `internal/integration/persistence/product/research/repository.go` | `research_commands` 与候选修改同库同事务；按 scope/project/key 的事务锁、同键异载荷冲突、候选行锁与 expectedRevision；COMMIT 出错返回 UNKNOWN |
| `internal/product/research/supplier_review.go` 及 persistence 同名文件 | 不可覆盖的人工供货核对、member 绑定、准确市场/货源/SKU/数量引用、来源重读与遮蔽、5 条一页；复用其受限仓储和证据端口 |
| `internal/app/httpapi/market_research.go`、`market_research_supplier.go` | 当前有效成员/项目权限、PRODUCT_RESEARCH 类型、归档只读；精确原件读取；证据校验有效期最多 5 秒且受 token/request deadline 截顶 |
| `web/listingkit-ui/src/lib/workbench/research-cost-preview.ts` 及测试 | 固定整数计算、缺项/非法/未支持区分、退款模型、方向汇率、精确符号和按分保本规则 |
| `research-cost-preview.tsx`、`research-panel.tsx`、`resource-pending.ts` | 当前临时试算使用组件 state；Research 命令复用按 user/org/project 隔离的 sessionStorage 原请求、单请求互斥及原键查询 |
| `internal/app/productsourcing/acquisition_initialization.go`、Research schema/privileges | 显式新空库安装、受限 `product_research_runtime`、serving 仅验证不 DDL |

### Eino / Research / DeepResearch 的复用边界

项目已经在 `internal/app/httpapi/market_research_application.go` 将现有 Research Agent 接到 `einoruntime.New`；`market_research_execution.go`、analysis 路由保留 prepare/start/read/resume、配置与预算准入。`analysis.Assessment` 是模型建议 DTO，不是经营评估事实。

Eino 官方提供 [DeepAgents](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_implementation/deepagents/) 的规划、工具调用、子 Agent 等能力；[官方示例](https://github.com/cloudwego/eino-examples) 包含 research assistant、Graph 多源研究和 Deer-Go。它们可用于现有/后续受治理调研 Agent，不能直接充当本项目的企业授权、货源版本、确定性金额计算或人工决定账本。

本批不新建 Agent 框架，不引入 DeepAgents 的 Shell、文件系统、自动重试或额外调用；不改变已批准 Research Agent 定义、调用/费用上限和一套计费。评估预览、保存、读回和引用 **0 次模型调用、0 次平台取数、0 次资源扣减**。Eino checkpoint 仍只管理 Agent 执行，不成为评估记录或人工决定的事实源。日后若让 Agent 使用评估，须沿现有受限 Tool/Service 明确授权，不能在本批隐式接入。

## 3. Owner 与最小接线

`候选页面 → 现有 Research BFF/作用域绑定 → research.Service → Research 计算/仓储端口 → 同一个 Product PostgreSQL 中的 Research 表`。

- Research 拥有用户经营假设、规则版本、计算结果/缺项、评估记录，以及决定对该记录的引用。
- 复用 ProjectCenter 的项目/归档/私有访问；Collection/SRC/Catalog 拥有来源、商品和准确发布版本。Research 只保存准确引用及摘要，不复制标题/图片/商品快照。
- 复用既有 SupplierReview 原记录；无货源时记 `supplierBinding=UNKNOWN`。不得伪造 SKU、库存、报价或供货确认。
- 决定状态仍只由原 Research decide/review/reset 管理；新记录不复制一份决定状态机。Preparation/Asset/Listing/Agent/Ledger 均不发生写入。

## 4. 服务端权威计算

新计算函数放在 `internal/product/research` 的局部纯计算文件；复用 Go 标准库 `math/big` 实现现有整数运算，不引入新计算引擎或金融框架。

输入结构沿现有 CostDraft：渠道、履约、规格说明、采购意向数量、USD 售价、退款概率、出处/日期、正常完成及全额退款两种成本情景、采购币种和人工方向汇率。金额、率及汇率在 JSON 中均为十进制字符串；不接受客户端利润/保本结果字段。

- USD/CNY 单项金额最多两位小数、上限 1,000,000.00；售价下界 USD 0.01。率按百分数 0–100、最多两位；汇率为每 1 CNY 对应多少 USD，大于 0、最多六位、上限 1000。数量 1–1,000,000，采购价始终为单件。
- 保留当前文本/出处/日期限制；空白表示未知，明确零与有理由的不适用分开。零概率情景可不填；提交的结构及显式非法值仍必须拒绝，不能借“未支持/不参与计算”绕过输入校验。
- 规范化后的金额、比例、汇率保存为对应固定小数位的字符串；不使用 float64 或 JS number 金额计算。原输入意图和规范化计算输入分别记录，不用规范化改写既有原键身份。
- 规则版本候选 `manual-unit-economics-v1`：沿 #1074 的期望贡献 `A×P−B`，完成收入系数 1、全额退款为 0；只扣原口径成本，不新增企业固定开支或另一套费用。
- 中间值精确；展示按美元分就近取整、半分远离零。利润符号取精确值；保本为原价格域内第一个不亏的分格点，沿用斜率正/零/负和无可行价规则。
- 结果沿原 `COMPUTABLE`、`INSUFFICIENT_INPUT`、`UNSUPPORTED_COST_RULE`；非法输入是请求拒绝，不保存假结果。缺项返回准确字段；未知不补零，未支持不输出完整利润。

服务端预览与保存调用 **同一函数**；保存重新校验输入并重算，不能相信先前预览结果或前端 hash。迁移原 TS 用例为 Go 对照向量，先确认既有金额/精度/退款/缺项等预期，再切换页面。最终删去 TS 权威公式及其正式调用，保留表单/DTO/渲染；不长期维护两套公式。

## 5. 评估记录与适用性

候选仅新增一张 Research 表 `research_assessments`，不新增请求账本。ID 使用本次保存的原命令 UUID key，`assessmentRevision=1`，记录不可 UPDATE/DELETE；另存就是新 key、新 ID。

| 记录组 | 内容 |
| --- | --- |
| 归属 | org、actor、保存时有效 member、project、candidate；均来自当前身份/原作用域 |
| 审计与并发 | 保存前 candidateRevision、保存后 candidateRevision、保存时间、原 requestKey/RequestHash |
| 市场依据 | 所选 link、完整 EvidenceRef/摘要、原 item/publication/version、候选身份；不存来源正文 |
| 货源依据 | 明确选择的 SupplierReview requestKey/candidateRevision、其准确货源引用/摘要、SKU、数量/单位；或明确 UNKNOWN |
| 条件 | 原有界意图、规范化输入、输入 hash、业务上下文 hash、计算规则版本 |
| 计算 | 结果状态、金额/精确符号/保本或缺项/未支持说明；本条已保存结果不在读取时用新规则重算 |

新增表沿原同 scope/project/candidate 的复合外键关联候选、以 ID 关联原 `research_commands.key`，并限制记录大小 32 KiB、正版本和合法结果状态。决定引用放在原 `Decision/DecisionView`、原 `research_commands.decision_json`，仅存小的 `{assessmentId, revision, contextHash, inputHash}`，不复制完整评估或利润。

**两种版本分开：**

- `expectedRevision` 是本次写入的并发栅栏。保存评估沿原事务推进候选 revision，但不改候选状态、selected link、理由或生成决定历史；原决定仍按 review→decide 的现行状态合同。
- `contextHash` 绑定业务依据：准确市场引用、明确的供货核对/UNKNOWN、SKU/数量/单位、渠道/履约/规格及规范化条件。保存前后技术 revision 只作审计，不纳入这个业务 hash。否则保存或 review/decide 自己增加 revision 就会令刚保存的评估失效。

相同原件出现了新版本，不自动改写评估或自动取数；原版本仍获准且可精确读取时，历史可读。所选 link/ref、供货记录/SKU/数量、费用或模式变化时，旧评估不可无提示地代表新条件。

没有全局“当前成本表”或自动 latest 指针。用户明确选定某份评估时，页面显示“使用此记录中的条件”；编辑任何条件立即取消它作为本次决定依据，要求另存，或显式放弃编辑后重新选择原评估。后端按准确引用和 hash 校验，不把浏览器未保存草稿当作服务端事实。

有绑定供货核对时必须与准确原记录的 SKU、数量、单位及市场关系一致；改变这些内容不能继承旧 CONFIRMED。用户可按现有入口重新核对，也可明确不引用货源、以 UNKNOWN 的手工假设另存，不能自动升级确认。

## 6. 接口候选与页面路径

复用 `/api/v1/workbench/research/projects/:project_id` 和原 BFF/模块，不增加管理中心。

| 入口候选 | 作用 |
| --- | --- |
| `POST /candidates/:candidate/assessment-preview` | 只读计算，带当前 expectedRevision/link/digest、可选准确 SupplierReview 引用及输入；不创建记录/准备任务 |
| 原 `POST /commands`，`action=save_assessment` | 既有 Idempotency-Key，提交同一有界输入；服务端重验、重算并保存 |
| 原 `GET /commands/:key` | 原键找回保存/决定的回执；新增评估命令回执含准确 assessment 引用，不含来源正文和完整成本 |
| `GET /candidates/:candidate/assessments/:assessment_id` | 精确读回该记录，不查“最新一条”，不重算 |
| `GET /candidates/:candidate/assessments?before=...` | 候选内历史，按保存后 candidateRevision 倒序，沿 SupplierReview 的 5+1 有界分页方式；详情还可按 ID 找回，避免分页窗口丢失 |
| 原 `POST /commands`，`action=decide` | 可选 `assessmentRef`；无引用继续原合同，有引用则在同一原决定事务内校验并记录 |

新增入口继承 verified/current identity、live organization、原项目/来源权限、10 秒请求预算、3 秒读体、严格 JSON、64 KiB 请求和 2 MiB 输出上界。拒绝重复/未知字段、异载荷 key、越界 ID/cursor、客户端 scope 和利润结果。预览仍要求原试算的项目可写/active 条件，不借只读请求扩权限。页内最多处理该页及所选准确记录，不扫描全项目。

页面先有“查看参考试算”，再有“保存本次评估”；保存成功明确显示记录 ID、保存时间、原版本、状态。若用户选择引用评估，原决定表单显示引用和条件摘要，仍须人工选状态并填理由；“未引用经营评估”路径保留。已保存历史独立于成本草稿和列表选择，刷新/重新登录按原项目及 assessment ID 读取，不靠组件 state 恢复。

## 7. 保存、决定与原键恢复

### 7.1 保存的同库事务

1. Service 绑定真实身份，校验项目可写/active、命令结构/大小；先走当前原键 Replay。已经提交的相同请求返回原回执，不重新计算、重新看“最新来源”或另存记录。新成员身份不得读回别的 member 的评估内容。
2. 对新命令按准确候选/link/ref 读取市场资料；有供货核对引用时按其 **原 requestKey** 精确加载，不通过列表首项代替。通过现有 SupplierSources 验证原件、SKU、数量及关联；无引用则保持 UNKNOWN。
3. 计算一次，生成规范化输入与结果；再次做当前身份/项目/来源校验，形成沿当前 supplier proof 模式的有界、member 绑定校验材料。有效期最多 5 秒，受 token 和请求 deadline 截顶。
4. 仓储沿现有 key 事务锁，先重新查原键，再锁候选并 CAS；校验 scope/member、selected link/ref/digest、准确供货核对记录与 proof 期限。并发决定/切换/保存导致 revision 不符就明确 409，不自动 rebase。
5. 同一事务推进候选 revision、插入 `research_commands` 原回执和 `research_assessments`；新表关联命令的外键在同一事务内满足。任何一步失败全部回滚。新评估表只给运行角色 SELECT/INSERT。
6. 提交前最后一次当前授权与来源核对，且上下文/校验材料未过期后提交。现有跨库端口不能提供全局原子撤权；边界是**最后门禁前可观察的撤权必须拒绝，最后门禁之后该原一次事务可以完成**。不宣称跨库瞬时撤销，不新增跨库锁/Saga；后续读回和决定使用都重新授权。

Research 拥有全部业务写入和回执；Project/Source 仅被只读核对，无跨 owner 副作用，不需要 outbox 或第二恢复 owner。实现时复用原事务辅助能力，不能把同一保存拆成“先评估、再另写回执”两次提交。

### 7.2 人工决定准确引用

原 decide 的 optional assessmentRef 只允许同 org/actor/member/project/candidate；在候选锁和当前 expectedRevision 下精确读取不可变记录，重验来源/供货可读性、所选准确依据以及 ref 的版本、inputHash/contextHash。当前所选资料与评估绑定不一致则拒绝引用，不改为最新评估。

评估的技术 revision 与候选当前 revision 不机械相等；核对的是准确业务依据。保存→原 review 推进候选版本→decide 的合法路径不因此被拒。决定及其 assessmentRef、理由、回执在原同一事务提交。另存评估不改变旧决定，后续 reset 沿原合同，历史引用不删。

待补/未支持记录可作为明确标记的人工理由依据（D1 已确认），不产生完整利润结论。无引用的 decide 不增加成本填写前置；不把数学结果转成批准、采购、转入或发布许可。

### 7.3 不同故障的结果

| 情况 | 原键处理 |
| --- | --- |
| 同 key 同意图，并发双击或重复 POST | 原 key 锁序列化；只产生一条评估/决定，回原 ID 与回执 |
| 同 key 改费用、来源、action 或评估引用 | 沿既有 RequestHash 比较，409，不覆盖。RequestHash 是原有界命令；规范化计算 hash 另存，不混用 |
| 两个不同 key 同 revision | 候选 CAS 只让一个提交，另一个明确冲突；用户刷新后决定是否另存，不自动覆盖 |
| COMMIT 成功但响应丢失/进程中断 | `GET /commands/:原key` 找原回执，再按 ID 精确读评估；不重新计算/预留/派任务 |
| COMMIT 回报错误，结果不明 | 返回 OUTCOME_UNKNOWN；保留原 key/意图，只查询原回执或显式原键重放，不用新 key“补一笔” |
| 暂时查不到原 key | 不能单凭一次 404 断言旧请求绝未提交；若显式重放，只提交原 key/同意图，经原 key 锁再次核对后决定唯一提交。无自动重试循环 |
| 验证错误、撤权、归档或过期且未提交 | 不产生业务记录；客户端区分明确拒绝与结果不明。已提交的回执不被后来失效改写 |

已确认 D3：复用现有 `useResourcePending` 的 **sessionStorage** 原请求机制，按 user/org/project 和本动作隔离保存本次显式“保存评估”的有界完整命令（含经营输入），确认回执后清除；不写 localStorage、不写 URL/遥测，不存 Key 等凭据。刷新后可原键核对/原内容重放；跨设备从服务端已保存记录找回。此项只限已点击保存后的待核请求，未保存试算草稿不持久化。它是用户已确认的、对旧“临时试算不进入浏览器持久存储”约束的窄扩展，仅适用于本设计新保存命令，不改变其他路径。

## 8. 权限、回读与保密

- 写入、引用和原键重放沿 `workbench.project.manage`、active PRODUCT_RESEARCH、Collection read 及当前真实 member；精确来源仍由其原 owner 授权。有权限选模型不自动得到本权限。
- 列表、精确读回和 Lookup 沿原私有项目 read；归档后只读，类型不符/跨企业/跨 actor 或保存 member 不符不披露详情。scope 仅由服务端 Bind 生成，查询均带完整作用域。
- 每次历史详情读取按冻结的准确市场及货源引用，通过原 owner 读回；结束前再次核对权限及候选/关联稳定性。若来源新版本存在，仍不以它替换旧版；若原版不可读/已 unlink/授权撤销，返回有界占位及 `DETAIL_UNAVAILABLE`，遮蔽来源引用细节、条件、金额和衍生结果，不能用已保存 JSON 绕过来源权限。
- 来源可读但与当前所选上下文不同：历史仍可查看，标“历史条件/不适用于当前选择”；不能以该 ref 执行当前引用决定。当前权限服务失败不返回受保护内容，不把暂时不可用写成记录损坏或删除。
- 读取不重算、不更新评估；返回 `applicability` 是当次授权/上下文投影，不是另一个持久化状态机。原键回执只投影必要的本操作身份，完整评估再单独授权读取。
- 请求体、规范化条件、来源引用和结果不进普通日志、追踪或审计详情；用 operation/key hash、错误码等有界非敏感元数据。GORM 敏感插入沿 supplier 的禁正文日志方式。所有响应 private/no-store；session 仅 D3 范围，切换 user/org/project 不复用请求。

## 9. Schema、角色与工程边界

仅规划在现有 Product 新空库 installer 中安装/验证评估表、索引、约束和权限；新空系统按当前模型使用，未授权的旧库/保留库不适配、不迁移、不隐式升级。缺 schema 的部署不开放入口、不在 serving 做 DDL，也不退回浏览器作为权威计算。

- owner/installer 沿原显式安装角色，不把 owner 权限交给应用。
- `product_research_runtime` 沿原受限 pool，新增评估表只 SELECT/INSERT；原候选只原列 UPDATE，命令/历史无 UPDATE/DELETE；扩展原权限验证的准确表 allowlist，不给其他 owner 表权限。
- 预期修改仅 Research domain/仓储/HTTP、原 composition/installer 的窄接线、原 BFF/页面/测试及本合同；不改 Eino 核心、Agent 定义、Provider、资源/钱账本、来源发布或转入事实。
- 同一用户结果一个主要 PR、一个 Writer；沿现有 Codex 完成范围内自动复核，不新增 Reviewer。冻结设计已获 IMPLEMENTATION_READY 及后续实施授权。

Legacy decision: **EXTRACT**（当前有效的纯计算规则）；**RETIRE**（替换后的 TS 权威实现）。
Reusable behavior: 原精度、退款、缺项、符号和保本规则及用例；现有 Research 权限、事务和恢复。
Current owner: `internal/product/research` 的服务端计算与记录；前端仅输入和展示。
Cutover/deletion condition: 页面接到同一服务端计算并核对原用例后，删除旧 TS 权威公式及调用，不保留 fallback/双算/双写。不承接已消失试算、旧数据、旧 ID 或旧执行状态。

## 10. 实施义务与限定验证

| 义务 | 后续必要证据 |
| --- | --- |
| IT1 同源计算 | 原 TS 向量到 Go：方向 FX、退款 0/100%、显式零/未知、半分正负、微亏符号、斜率正/零/负、首个分格点、上界及未支持；前端无权威公式/伪造结果入口 |
| IT2 持久化与恢复 | 隔离 PG 中评估+命令同事务；同 key 同/异载荷、跨 action、并发、COMMIT/响应丢失、重建服务后原键读回；无重复记录，无第二恢复 owner |
| IT3 适用性与决定 | 保存后 review/decide 的 revision 推进仍可引用；改来源/SKU/数量/费用或选另一评估的 hash 明确区分；决定引用准确 ID，不是 latest；无评估路径及旧决定保持 |
| IT4 权限与遮蔽 | 跨 org/actor/member/project、归档、来源撤权/unlink、原版缺失、末次门禁/有效期、读期间变更；占位不能泄露条件或结果；受限运行角色拒绝越权和改旧记录 |
| IT5 正常 UI 路径 | 原候选预览→保存→整页刷新→重新登录找回；编辑另存；待补明确标记；选择评估后显式人工决定；原 pending/session/key 找回及切换上下文隔离 |

实施授权仅使用合成数据、合成来源及任务自有临时 PostgreSQL。IT1 对应 `internal/product/research/cost_test.go`；IT2–IT4 对应 Research persistence 的 `assessment_postgres_test.go`、现有 supplier owner 隔离回归及 HTTP 严格输入回归；IT5 对应原页面、评估组件、原 pending 和 BFF 回归。组件卸载/重进用于离线验证服务端找回及原键恢复，不冒充真实浏览器刷新/重新登录。具体执行结果与准确 HEAD 写 PR；不开发额外 runner/验收平台。

浏览器、真实业务、保留环境安装/升级、运行及产品验收继续 **NOT_RUN**；现场准入和未取证项 **UNKNOWN**。现有 35444/35446/35447 的实例、数据和会话均不在本轮执行范围。

## 11. 用户已确认的最小决定（2026-11-08）

- **D1 待补也可留档与引用**：合法但缺费用/未支持规则的记录明确标记，用户仍可把它作为人工决定的有限依据并填理由；不生成完整利润、不自动批准。
- **D2 明确选择记录中的条件**：不自动把最新评估作为当前条件；编辑后必须另存或明确放弃编辑再选原记录。历史评估与“适用于当前选择”分开，不因 review/decide 的纯技术 revision 推进就失效。
- **D3 原请求会话恢复**：仅对用户已显式保存、尚待核对的命令，复用现有 sessionStorage 有界保存原经营输入，原回执确定后清除；未保存草稿仍不持久化，不把 UI 临时 state 当作可靠恢复。

初次确认覆盖 D1–D3 及设计提交/推送/Ready PR、正常 CI 与现有 Codex 自动架构评审；后续用户已单独批准按该已评审 HEAD 实施及维护原 PR。未批准保留环境安装、真实数据/Key/平台动作、合并、部署或 Issue 关闭。#1201 只核本批实际市场/供货/准备交接，未证实的缺口保留 UNKNOWN，不自行扩范围。


## 11. 指定统一实例的评估表定向准备（2026-11-09）

用户已确认复用统一实例现有功能/身份/数据，先准备#1205最小整合及评估表安装，检查后再确认更新；35448保留。本决定仅替代本功能“只能新空库安装”的阶段限制，不授权实际库DDL、镜像构建、重启或部署。Design Basis: Reuse Existing Architecture / READY；复用本设计既有表/事实owner/事务/权限/原键及Product显式feature installer，无数据迁移或旧事实重新解释。

现有product-acquisition-init增加互斥的--enable-research-assessments和准确--confirm-database；沿私密owner manifest、30s预算、原source-acquisition-init advisory事务锁。必须是实际目标库的acquisition_owner、原四张Research表owner及schema/供货核对schema正确，runtime无危险角色能力。只有目标表缺失才调用同一InstallAssessmentSchema；存在时严格核对表形状、owner、约束及准确ACL，不能重建/修复或回填。安装/校验同事务，失败全部回滚；新表仅Research runtime SELECT/INSERT，原表/grants/记录不变。

隔离验证覆盖非空库数据/grants保留、重复/并发安装、错误库/owner/原schema/目标schema或ACL漂移拒绝及事务失败无残留。不新增installer框架、第二schema定义或serving DDL。组合源码仅带本功能及路由、已批准退休边界；不随之升级DataServices计价/Store观察schema，旧CI不作为新组合直接执行结果。运行/真实保存恢复/浏览器业务/产品验收NOT_RUN，最终镜像及实例更新另行确认。
