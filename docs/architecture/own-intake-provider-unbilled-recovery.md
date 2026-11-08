# #1165 供应商明确拒绝且未计费的原调用恢复

Design Basis 为 Independent Architecture。源码调查基线为 `cd3675393e7e102425d96bbcd72c6aef6e2489cb`，试用后端证据基线为 `fb8cdbc013548b5f13a2c558922441334c576607`。离线实现设计的 Admission Status 为 IMPLEMENTATION_READY，独立评审依据见文末。本次真实恢复的执行证据为 NOT_READY。执行位置为 [#1158](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1158)，延续已合入的 [#1165](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1165)。

## 产品决定、范围与当前事实

用户于 2026-11-08 要求先评审“供应商明确拒绝且确认未计费”的恢复，重点为绑定原调用、零消费结算和幂等释放 15.445282 AI 点。本轮只交付设计和独立评审。实现、schema 安装、账本修改、释放、部署、真实模型调用、商品保存和 Issue 关闭均留在后续授权阶段。

使用者是负责本地试用的授权操作员。当前操作场景为 OpenAI 组织账务暂停时，一次商品整理请求收到拒绝，应用仍占用成员和企业 AI 点。预期交付为操作员核实原调用的拒绝与未计费证明后，将原调用终结为已拒绝零消费，原预留仅释放一次。成员随后能继续使用原 30 点月限。

业务和安全依据为 [原商品整理](own-product-ai-intake.md)、[OpenAI Mini 接入](own-product-intake-openai-mini.md)、[统一配置与微点合同](enterprise-shared-model-configuration.md)、[当前 owner](project-boundaries.md) 和 [全新系统基线](../product/greenfield-no-legacy-migration.md)。本设计只增补下述拒绝终态，其他 UNKNOWN、实际用量和原定价协议继续适用。Figma 在此无 UI/IA 权威作用。

本轮 Must 为原调用和原预留的完整绑定、可审计的未计费证明、A=0、原 R 幂等释放、原成员原月份的计数、跨库失败可收敛、原请求停止且不再派发。Should 为沿现有审计读取展示拒绝原因和恢复状态。范围外包括所有 429 自动退款、任意厂商/模型、成功后取消、部分输出失败、超时/断线自动退款、批量冲正、重新发点、更改月限、价格重算、通用工单/审批/对账平台和商品保存。没有新增 Accepted Risk。

只读证据快照如下。它是评审案例，不在通用代码硬编码企业、调用或金额。

| 项目 | 原值 |
| --- | --- |
| 企业 / actor / member | `394513989062623234` / `394513839896395778` / `394513991663091714` |
| Run / 请求键 | `b5499069-a4cb-4851-b7ea-ce3760de098e` |
| Invocation | `b5499069-a4cb-4851-b7ea-ce3760de098e:step:1` |
| Reservation | `0d63cf7a-2874-4229-a415-ff812ce1cd4d` |
| 操作 / provider / 模型 | `product.own.intake.generate` / `openai` / `gpt-5.4-mini-2026-03-17` |
| 冻结 tariff | USD 输入每百万 0.75、输出每百万 4.50，FX 6.7168 |
| 原输入 / 输出上界 | 6,660 / 4,000 Token |
| R / 已消费 | 15,445,282 微点 / 0 |
| 请求时间 / 响应时间 UTC | 2026-11-08 07:56:27 / 07:56:29 |
| 原响应 | 一次 handoff，HTTP 429，223 bytes，usage 未知 |
| 请求 ID 哈希 | `49492ae97619605c93975aadb54776e37231d9a2d59d2f96aa5d3375c861226d` |
| 运行 / 调用 / 预留 | stopped + model_outcome_unknown / dispatched / reserved |
| 平台侧只读关联 | 同一项目、密钥最后使用时间相符，按该密钥筛选唯一错误为 Billing is not active |

账务页原有逾期发票，现已读回 Paid，组织余额 $94，暂停提示已解除。这只解释后续资格恢复。平台汇总零 Token、零费用、已付款、余额充足、本地默认零值或当前 key 可用，都不能单独证明原调用未计费。[OpenAI 错误说明](https://developers.openai.com/api/docs/guides/error-codes)

## 现状缺口与 owner

`GuardedClient` 可观察一次受控请求的状态和请求 ID 哈希，但当前诊断未持久化可结算的拒绝证明。`Executor.GenerateWithGate` 对缺 usage 的错误返回 UNKNOWN，原 Invocation 保持 dispatched。`GormModelInvocationRepository.modelPointTerminal` 的普通零消费释放仅承认发送前拒绝。运行停止并未产生商业终态，所以 RecoverDue 继续保留 R。

| 事实 | 唯一 owner |
| --- | --- |
| 派发身份、供应商观测、拒绝终态和证明 | aicapability 的原 `ai_invocations` |
| 原运行、候选及商品整理结果投影 | agent 的原 `product_agent_runs` |
| 企业/成员/月预留、消费、债务、释放与审计 | orgresource 的原商业账本 |
| 外部账户资格、账单和拒绝来源 | OpenAI 本身 |

复用现有 `InvocationRecovery.ResolveDispatchedInvocation`、`GormInvocationRecorder`、原 ModelInvocationFactReader、`GormModelInvocationRepository.finalize` 和 `RecoverDue`。使用现有 GORM、pgx、私密文件权限检查和 PostgreSQL 事务/CAS，无新恢复框架、第二账本或跨库事务平台。

## 证明合同与绑定

增加受限领域值 `ProviderRejectionProof`，版本 `own-intake-openai-unbilled-v1`。通过现有 Invocation 可选字段持久化，规范 JSON 上限 8 KiB，拒绝重复/未知字段。其内容为以下不可变事实。

- 原 dispatch fingerprint。它由原 Invocation 完整冻结身份计算，包含 tenant、actor、member、invocation、run、operation、capability、input hash、provider、wire model、credential reference、policy/configuration 版本、原价目/汇率和 P/C 上界。沿现有 `modelPointFingerprint` 的固定编码，补充证明不能改变此摘要。
- 原 reservation ID、原数量及单位、原 member month、原 reserve receipt fingerprint。恢复服务和商业 owner 各自验证原绑定，输入不能自行选择金额、月份或收费结果。
- provider request ID 的 SHA-256、HTTP 状态、受限拒绝原因 `billing_inactive`、供应商拒绝时间。原始请求 ID、响应正文、密钥、提示词和图片不入证明。
- 证据类型、规范化证据包摘要、操作员身份、观察时间、原稳定恢复操作键和批准依据引用。自动派发、签收和网页评论均不构成操作员批准。
- 固定非收费判定策略版本及证明摘要。没有调用方自由填写的 `unbilled=true`、`amount=0`、`refund` 或任意错误码。

证明的合法来源仅为受限操作员从供应商受认证控制面或支持回执核实的拒绝记录。当前阶段只增加原调用恢复入口，自动捕获并结算未来所有错误留在范围外。该入口不携带 provider client，不加载模型密钥，不调用模型。

证明须同时确认该原请求已被拒绝且该请求没有产生计费用量。精确 provider request ID 能取得时，其 SHA-256 必须与原观测相同。没有可比对的 provider request ID 时，仅允许受限操作员核对一组无歧义的关联证据，包括同一 provider organization/project、原 key tracking identity、原调用秒级时间、原一次 handoff，以及供应商按该 key 和相应时间范围筛选的唯一拒绝记录。原密钥最后使用时间及本地该 key 的唯一调用辅助校验。若存在多个可能请求、迟到计费、平台只有汇总零用量、无法核对原 key 版本、缺明确未计费结论，证明不成立，保持 UNKNOWN。不能仅通过相似文本或人为选择一个请求消除歧义。

“Billing is not active”是本次已观察到的供应商控制面拒绝类别。官方通用错误页没有给出该次具体 wire error.code，本设计不编造其值，也不把所有 insufficient_quota/429 映射成该类别。证据包须有 provider 侧明确的未计费结论，或经独立复核准入的该准确拒绝类别的“不执行且不计费”语义依据。通用文档和使用量列表本身不满足这项要求。本次现有证据足以定位账务阻塞；实际恢复前仍须把明确未计费依据与无歧义原请求关联一起补齐，当前为执行证据待核，不得在设计评审中预签发本次退款。

规范化的最小证据包只保存必要元数据和来源摘要。完整网页截图/支持回复留在原受保护操作材料中，证明摘要写入 Invocation。受限操作员在保留证据可追溯的前提下作事实核验，系统信任这条既有本机操作边界。普通租户调用者、前端表单、Run 的停止原因及任意外部链接不具备出具证明资格。

## 状态和持久化

使用现有 `InvocationFailed` 终态，附新拒绝证明和受限 ErrorCategory/ErrorCode。区分收费已确定为零与供应商没有返回 token usage。

| 事实字段 | 明确拒绝未计费时 |
| --- | --- |
| Outcome | failed |
| ProviderRejectionProof | 完整且校验通过 |
| UsageKnown / BillableUsage | false / nil，保留未观测事实 |
| Prompt/Completion/Total | 0，占位值仍不作为用量证明 |
| EstimatedCostKnown / EstimatedCostMicros | true / 0，含义为本系统已确认无收费 |
| FinishedAt | 固定原供应商拒绝时间，不使用每次重试当前时间 |
| 原输入/输出、tariff、actor/member 等 | 原冻结值 |
| 商业终态 | committed，A=0，释放 R-A |
| Run | 保持 stopped，原请求只读投影“供应商拒绝，零消费恢复中/已恢复” |

该分支实际发生了一次请求交接，不能改写为 `rejected_before_dispatch`。商业 finalize 的第三种判定是“有可信拒绝证明的 A=0”，与 no-dispatch 和有实际用量的终态分别校验。它复用 committed 的 A=0 结算路径，保留真实派发历史。每次判断同时核对 proof、原 fee policy 与 allowed operation，其他厂商和操作不会因存在 proof 字段获准。

原 Run 的执行历史及 model_outcome_unknown 记录保留。商品整理读取时组合原 Invocation 与原 reservation receipt 产生当前恢复展示，不新建运行、不改原候选或历史消耗，不把原保守预算记录改写成实际 provider usage。收费恢复成功仍无商品候选，人工录入路径按原功能可用。

唯一新增持久化字段为原 `ai_invocations.provider_rejection_proof_json`，nullable text，SQL NULL 表示缺证明。更新当前模型/列合同、原 schema owner、受限 UPDATE 列集合与只读 owner 投影。没有新表、proof 台账、公共 API 或额外状态机。证据 JSON 是原 Invocation 的组成部分，文件只作佐证，不是第二事实 owner。

这是当前原生系统的正常 schema 演进，符合全新系统基线。未来正常安装建立当前合同。保留实例的能力扩展须在取得单独授权后，经原 ProductAgent schema owner 的有界显式操作，仅添加该空列及必要的精确列权限，不重写既有行、迁移旧 ID/旧模型或回填任何证明。先检查准确关系 owner、既有当前列合同、角色和预留身份，异常停止。应用启动只验证。当前本轮不安装、不更新 schema。

无证明时 `InvocationRecord.ProviderRejectionProof` 序列化使用 omitempty，原固定 JSON 字节和 reserve/finalize 哈希继续成立。`modelPointFingerprint` 归一化时清空新证明，沿用原收费身份。`samePointDispatchMetadata` 同样把终态证明视为终态字段。`modelPointFactHash` 对新终态包含完整证明，因此相同原调用换证明、金额或结论会冲突。既有所有普通终态的 JSON/指纹及已结算回执不得变化。

## 调用路径、权限和边界

拟增加一个受限原生命令 `cmd/model-invocation-recovery`，复用原私密配置加载与权限核验习惯，默认 inspect。配置最大 16 KiB，Linux 0600 或 Windows 仅当前操作员、SYSTEM 和管理员。输入包含固定 organization、member、invocation、reservation、expected dispatch fingerprint、证据包引用/摘要、操作员、明确授权引用及执行开关。稳定操作键从原 invocation 和本恢复协议版本派生，同一键不同证据拒绝。一次命令只处理一条原调用。

- contract 为 aicapability 的类型化拒绝证明及既有 InvocationRecovery。恢复服务使用可信操作员配置和注入的原 Invocation/Reservation reader 校验原冻结身份、同租户、原 member month 和授权范围。
- implementation 由 aicapability 原 recorder 完成调用侧 CAS 和不可变终态记录，orgresource 原 adapter 消费持久化调用事实完成结算。入口程序集成在 app 层，命令只解析明确配置并装配现有受限 DB pool，各池上限 1。
- consumer 为指定商品整理操作及原商业恢复扫描。保持只有原 owner 可以出具收费事实，商业 owner 不接受命令直接传入零费率或任意退款金额。
- inspect 强制只读连接，仅验证身份、来源、当前证明和账本状态。resolve 要求保护配置执行开关、匹配的明确授权、必要写角色和准确目标。批准范围与当前操作员不符、跨企业、错误 member、错误 operation、原金额/版本冲突均停止。
- 终结已发生的原费用按原成员/月处理，即使成员后来离开、key 被撤销、当前模型禁用或当前报价过期，恢复也不读取新 key/价目或转移归属。它不赋予新模型执行权限。受限操作员的当前执行授权仍必须有效。证据不完整时不得以撤销或禁用作为退款依据。

Threat Model 包括普通用户伪造零消费、跨企业/成员替换、同键异证明、重复/并发恢复、终态覆盖和跨库 ACK 丢失。现有受保护本机操作员及受限 DB 凭据是可信维护边界，不另建 IAM、人工审批平台或供应商签名基础设施。凭据保护与租户隔离仍为 Must。

## 事务、幂等和恢复

两个事实位于不同数据库，复用原 fixed-owner receipt/proof 与 RecoverDue。Invocation 先产生可信终态，商业账本随后收敛。没有跨库原子提交或新 outbox。

1. 使用只读接口读取原调用、Run、原预留、原 reserve receipt 和原 member month，核对固定指纹、允许策略、R 与 evidence。执行前再次核对原状态与受限维护授权。
2. 在 Invocation owner 的单事务内对原 invocation 加行锁/条件更新，要求原状态仍为 dispatched 或与目标完全相同的已终结事实。先完成全部证据与状态校验，再写 failed、proof、固定 FinishedAt 和零收费结论。字段用显式列清单保留不可变原值。
3. 若发现其他成功/用量已知终态，或另一份证明已赢得 CAS，拒绝覆盖。相同 proof 和同一终态返回原事实。CAS 失败不得退回不受限 RecordInvocation 更新。
4. 通过原 recorder 的商业适配器和 owner reader 读取刚持久化的终态。沿原 `model-point-finalize:<invocation digest>` 操作键与完整 fact proof hash，核对原 reserve fingerprint 和 receipt。已经完成时返回原结算回执。
   当前 `RecordInvocation` 对普通 failed 会进入 `ReleaseAIInvocationUsage`，而 `SettleSuccessfulInvocation` 只转发 known-usage 的成功/观测失败。实现须在原 recorder 的结算触发点显式识别本 proof 终态，调用原商业结算入口，以 total=0 交叉核对原事实。商业 adapter 的新 proof 分支先于普通 Token 校验得出 A=0、noDispatch=false，进入 committed finalize。它不经过只承认 noDispatch 的普通 Release 分支。Resolve、首次 RecordInvocation 和 RecoverDue 三条调用链均须在隔离测试验证，不能只放宽底层 terminal 判断。
5. 商业库一个事务内锁定原 reserve/finalize operation、reservation、原 member month、企业 bucket 和适用债务。沿原锁顺序，仅在匹配状态下执行 counter.reserved -= R、counter.consumed += 0、bucket.reserved -= R、bucket.consumed += 0，并由原 debt-first credit 函数释放 R。写 committed/A=0/settled_at、原事件和原审计，最后完成原 finalize operation。
6. 跨库写失败、超时、进程终止、响应丢失时，inspect 原 invocation/finalize key。调用事实已终结但商业未完成时，原 RecoverDue 或同键恢复重试收敛商业事务。调用是否提交未明时仅先 inspect。任何情况不再派发模型。
7. 并发恢复、回执重读、恢复扫描和用户原请求查询均沿同一身份，累计仅一次 R 释放。成功终态后出现不相容的收费事实不能覆盖原终态，作为冲突报告给原 owner，进入原人工核对范围。

事件和审计复用当前 model point finalize 身份，A=0 的收费结算与差额释放在同一个事务中。不得新增零价订单或 grant 代替释放，不把未知直接改为已取消。记录证明摘要及原 Invocation/Reservation 的关联便于审计。

| 失败位置 | 可见事实及唯一恢复责任 |
| --- | --- |
| 证明不足/授权失效/绑定冲突 | 原 UNKNOWN + R，由操作员补齐准确原证据 |
| Invocation 事务回滚 | 原调用和预留完整保留 |
| Invocation commit ACK 丢失 | 先 inspect 同一 invocation/proof，不换键 |
| 调用终态已提交，商业暂不可用 | 原 Invocation 已拒绝；R 保留，原 RecoverDue 负责结算 |
| 商业事务中任何一步失败 | 企业/月计数、债务、事件、receipt 一起回滚 |
| 商业 commit ACK 丢失/重启 | 固定 finalize key 返回同一回执，不重复释放 |
| 已有成功或不同拒绝证明 | 冲突，不覆盖，不返还第二次 |
| 原调用日志丢失且无可关联供应商证明 | 继续 UNKNOWN，不能从付款、零汇总或超时推断 |

## 本次 15.445282 点的算术和后续执行前提

R 取原预留行的 15,445,282 微点，并与原 receipt/冻结 tariff 重算交叉验证。原上界费用为 (6,660×0.75 + 4,000×4.50)/1,000,000 = $0.022995，FX 6.7168 得 ¥0.154452816，按 1 点=¥0.01 向上取整到微点即 15.445282 点。恢复实际收费 A 固定由受认可的明确未计费证明得出 0，无 Token 推算和新报价参与。

当前原 member 的 2026-11 UTC 月计数 reserved=15,445,282、consumed=0，企业 available=9,984,554,718、reserved=15,445,282、consumed=0。在无并发消费/债务时，一次结算后企业 available=10,000,000,000、reserved=0、consumed=0，原月 reserved=0、consumed=0，原 30 点月限剩余恢复到 30 点。存在债务时沿 debt-first 分配，不能承诺 R 全部进入 available。并发其它消费按实际原事务结果计算。

实际执行前必须完成以下事实核对。独立设计通过并不证明它们已满足。

- 该次明确拒绝且未计费的证据强度符合上文合同，原 fingerprint、证据和预留一致。
- 原调用尚未被其他 owner 结算、原预留和 reserve receipt 匹配，角色/当前 schema/写权限可用。
- 当前实例能力扩展、二进制更新、具体操作员、原调用的 resolve 与原 R 释放均获得后续明确授权。
- 本轮设计结论不含模型调用或商品保存授权。原请求不能重新生成候选，后续新试用仍按当前预算与明确授权执行。

## 实现验证与架构准入

### 当前 schema 切换与失败策略

原 `VerifyModelPointInvocationSchema` 和 schema owner 严格核列数，已有表的 AutoMigrate 只验证。添加 proof 列后旧二进制重启会失败。未来安装计划必须绑定同一当前版本的 schema owner、Invocation writer、商业 fact reader 和服务构件，逐一枚举共享该关系的消费者。原实例更新中通用的“失败则回退旧镜像”脚本不能直接沿用。

先在隔离环境建立准确当前 schema 并验证目标版本启动、普通无 proof 路径和恢复关闭配置。拟执行构件、私密配置、精确关系/ACL 元数据及恢复材料先固定。获准维护窗口内停止目标服务的新写入并核实在途任务，完成准确 owner 的空列/精确列 ACL 扩展，随后只读验证并切换匹配版本服务。普通启动继续无 DDL，resolve 开关默认关闭。服务健康及原行/预留读回一致后，才可在另行获准的单次恢复中开启 resolve。

空列安装或启动验证失败，且查证没有任何 proof 写入时，由原 schema owner 在明确操作清单下恢复本次新增列/ACL 的元数据，再启动原准确构件。元数据逆操作须同时确认列完全为空，已有调用/预留及其他关系未变。任何 proof 已持久化后，保留新列和事实，只允许匹配新合同的已验证构件恢复服务或向前修复，关闭 resolve 写入口仍可读取及收敛原结算。不得自动回退旧镜像、删除非空 proof 列或用旧备份覆盖新增终态。当前设计评审没有授权这套真实切换或逆操作。

只安排与本改动直接相关的现有测试，不建设新的验收平台。业务规则先 RED/GREEN，沿已有 Executor 受控传输、Invocation recorder 和 orgresource PostgreSQL 测试扩展。

| 当前 Must | 隔离验证 |
| --- | --- |
| 准确原身份 | 改 tenant/member/run/input/model/tariff/reservation/证据绑定任一项即拒绝 |
| 明确未计费 | 完整证明 A=0；只有 429/默认零/平台汇总零/含输出/收费未知均保持 UNKNOWN |
| 原身份和收费语义 | 发生 handoff 的拒绝保持真实派发历史，UsageKnown=false，无伪造 BillableUsage |
| 指纹稳定 | 原无 proof 的 reserve/finalize 固定字节向量一致，新 proof 不改变原 dispatch fingerprint |
| 幂等与并发 | 同键多次、两个命令和 RecoverDue 并行只有一次释放；异 proof/既有成功冲突 |
| 跨库失败 | 复用现有 recorder/adapter seam 模拟调用终态后商业失败、ACK 丢失，原扫描可收敛 |
| 原月份/债务 | UTC 跨月仍释放原月，成员离开仍归原 member，债务优先，其他余额/限额不改 |
| 当前实例安全演进 | 一列显式能力扩展只改变列/精确 ACL，无既有行写入；无新增实例初始化 |
| 最终用户结果 | 原请求显示拒绝、A=0/恢复状态，无候选；商品数量不变；15,445,282 微点一次收口 |

运行时入口读取、原拒绝记录恢复、实际 R 释放、真实模型与保存验证当前均为 NOT_RUN。设计文本检查只核引用、owner 和差异范围，不冒充离线实现或真实恢复通过。

独立 Reviewer 应针对本文件的准确 blob 与源码基线，按 BLOCKER / IMPLEMENTATION_TEST / BACKLOG / ACCEPTED_RISK / NOT_APPLICABLE 分类 findings。BLOCKER 必须指明跨租户、错误授权、错误释放/重复扣费、数据损坏、不可安全恢复副作用或核心恢复 happy path/rollout 无法完成中的具体后果。最多两轮正常设计评审，非阻塞实现义务留在本批后续实现验证收敛。

## 独立架构评审结论

2026-11-08，原唯一独立 Reviewer 完成两轮。首轮绑定 `c03b5e1986df069aa4f50993fd91d2d90e5be325`，第二轮仅复核结算触发链和 schema 切换两处增量，最终绑定正文 blob `c2d468a9b4de21d8f845a17180956445ca232e75`，对应提交 `85e65ca3e66354a4bef733538e7ff9186a11c495`。源码基线均为 `cd3675393e7e102425d96bbcd72c6aef6e2489cb`。

最终结论为 IMPLEMENTATION_READY，设计 BLOCKER 为 0。recorder/adapter/RecoverDue 三链、原固定哈希、同/异证明幂等和竞态、跨库 ACK 丢失、原月份/debt-first、准确 schema 切换列为 IMPLEMENTATION_TEST，必须在后续实现和隔离验证中收敛。当前提交只补结论与状态元数据，已评审设计正文未变。

本次 provider 明确未计费及原调用关联证明仍须按本文条件补齐。实际原 15.445282 AI 点恢复仍为 NOT_READY，真实 schema/账本/部署/provider/商品操作均为 NOT_RUN。后续正式实现和真实执行权限继续由用户阶段决定。
