# Chat 定向结构与 OpenAI 文本用途准备

状态：`IMPLEMENTATION_READY`，限 #1246 隔离实现。独立架构评审核对 `0a9d950d911e2039e345242fd51af56b02cb9d86`，明确要求修正事务提交前后校验顺序，已在本文落实；无其他设计级 BLOCKER。验证项见下文。依赖 #1242 / 已合并 #1243。基线 `58111fce176ee0974f8d4893dec0cf02b1e43fe3`。本文不授权现场数据库、价格、额度、部署或真实调用。

## 用户结果与范围

保留完整商品智能体、原业务和现存预留，为获准企业以统一模型与价目使用内容 Chat 准备三个前置：受限空 Chat 结构安装；OpenAI GPT-5.4 Mini 文本规划和内容用途；可解释的估算与试用预算。新安装、空业务基线继续有效。不得导入、回填、解释或迁移旧业务事实，不设置第二 owner、不改 UI、不创建新计费或授权平台。真实启用与付费执行另行授权。

权威依据：用户 2026-11-11 要求保留原 35444 商品、平台草稿、图片、其他能力及原 15.445282 点预留，只允许隔离验证；[内容 Chat 共存设计](chat-content-full-runtime-coexistence-candidate.md)、[企业共享模型配置](enterprise-shared-model-configuration.md)、[共享连接](shared-channel-credentials.md)、[工作范围](ai-workbench-work-scope.md)、现行权限/事务/资源合同和全新系统基线。Figma 只用于业务流程辅助，不作为 UI 标准。

## 已核对现状与非正式 Spike

截至 2026-11-11 18:21 新加坡时间，原 `product_agent` 库中 Chat 七表均为空；仅只读计数并不授权变更。当前 Chat runtime 配置有独立 `ai_workbench_runtime` 池，开关关闭；现有完整 ProductAgent 继续运行。与当前 fresh schema 逐项对照，缺四列 `conversations.business_scope`、`commands.frozen_scope`、`commands.scope_digest`、`execution_proposals.workbench_context_ref` 与其三条精确 CHECK；其他比较过的字段、约束、索引一致。七表 owner 均为 `product_agent_owner`。再次运行新安装 `CREATE TABLE IF NOT EXISTS` 不会补列，不能冒称安装成功。

非正式 POC 只在独立 PostgreSQL 使用仅结构副本、合成角色/数据：精确四列三约束后现有 `VerifySchema` 以 `ai_workbench_runtime` 成功；七表任一有记录则拒绝；无关示例表保持；合成完整库 dump/restore 后数据及 runtime schema 校验通过。脚本硬编码仅允许测试数据库，**不是生产安装器**；尚未覆盖所有结构漂移、并发、权限或目标库恢复，不能将其直接用于保留实例。

现有 `openai-mini-1165` 仅去密元数据可见并启用；其已准入单图用途不等于 Chat 文本授权。当前一般兼容组件在合成请求中使用已弃用的 `max_tokens`，单图和标题 pilot 虽使用 `max_completion_tokens`，却各自限制用途；不能改名套用。OpenAI 官方文档将 `max_tokens` 标为弃用，不能据此声称已不支持具体型号，也不能将本地模拟视为厂商实测。

## 结构 owner、安装、失败与恢复候选

- 继续使用原 `ai_workbench` schema/七表和原 ProductAgent DB，schema/表 owner 为原 `product_agent_owner`，runtime 仍为现有独立 `ai_workbench_runtime`。不新建第二 Chat store、不删表、不搬迁或回填数据。新安装入口保持不变；仅候选新增显式、有界的空 Chat 结构准备模式，默认不执行，应用启动永不 DDL。
- 准备模式只接受严格识别的当前目标和角色；事务内设有限超时并按固定顺序锁定七表，核查 schema owner、运行角色不拥有对象、不具危险属性/成员关系/跨 owner 权限，检查七表全空及既有结构、约束、索引与允许的原安装结构完全匹配。不能只按表名或列数判断。对原结构准确补四列及三约束，由 owner 事务内核对结构和原角色精确权限后提交；不得赋予角色成员关系。提交后使用独立 `ai_workbench_runtime` 连接运行既有 `VerifySchema`，通过后才可报告结构就绪。若提交成功而 runtime 检查失败，保留已提交结构及原数据、保持 Chat 关闭，并只读诊断；不以当前测试账户或 owner 冒充 runtime，也不删除或逆转列。只读/格式预览不得制造安装回执或业务事实。
- 重新执行时，已完整安装且权限准确时只验证返回已就绪；原结构可首次安装；部分安装、不同约束或 owner、额外不明对象、任何一表非空、超时、不可读取、角色不匹配则拒绝，不猜测修复。事务原子；提交前异常、取消或失败回滚，提交 ACK 不明先只读校验结构与原输入，不重新授权或覆写。提交后检查失败不是事务回滚，不将其记作成功；排查与后续动作需明确授权。必须验证双并发、锁等待、拒绝非空、注入失败及提交后 runtime 拒绝；不跨锁等待外部服务。
- 备份与回退是现场前置：用户批准后再按当前运维合同创建一致的当前备份，离线恢复验证 ProductAgent DB、依赖、owner/ACL 和运行清单；11 月 10 日的历史备份不是本次当前备份。先核全七表与依赖的现场变化，保留原容器镜像/清单及全部卷。旧版本回退不能恢复整个数据库覆盖新增商品或其他事实；若 Chat 新增数据，禁止把旧整库备份直接覆盖现行库。回退与恢复的具体停写窗口和保留数据处置需隔离验证并由用户审批。不中止、删除、释放任何已有 UNKNOWN 或预留。

## OpenAI 文本用途和唯一价格

复用 `openai-compatible` / Eino SDK、原 GuardedClient/Executor/Invocation/Resource 与同一企业连接，不新增 Key 保存。候选新增仅限 `openai`、准确模型 `gpt-5.4-mini-2026-03-17`、全球标准同步 `https://api.openai.com/v1`、`aiworkbench_chat_plan` 与 `product.content.decision` 的明确文本策略版本；其他模型、供应商、图像、标题 pilot 不变。现有组织 current 分别为两个用途提供匹配同一 text 模型定义、连接 revision、价格与 FX 的 profile，仍保留各自 prompt/schema；跨企业、缺任一用途、无有效价目或被撤销均拒绝，不能借单图/标题策略和默认 Key。

共享文本 current 同时约束企业所有已开放的文本用途：现场必须逐一盘点并保留已有用途的合格绑定；同一选择缺任何已开放用途时不能宣称就绪。隔离场景还验证了原标题 pilot、规划和内容在同一引用、连接与价目下同时完成资格核对；这不证明现场所有其他用途已具备绑定或真实原标题调用通过。

候选 wire 复用现有 SDK 构造与严格解析：每次一条 `system` 加一条 `user`，无工具/流式/重定向/自动重试，`max_completion_tokens` 为 profile 有界值、`reasoning_effort=none`、`service_tier=default`、`n=1`、`store=false`。对模型、endpoint、用途、输出上限和已知字段做精确校验，复用现有严格 JSON 和 OpenAI usage mapper；基本 P/C/T 和缓存拆分必须可验证，未知、超界或价格维度不完整沿原 UNKNOWN/预留处理。不得为模拟通过声称真实 Key、型号或服务可用；真实响应、计费和质量需用户另行授权后验证。

两用途均走原冻结 Profile → Quote → 原 claim/预留 → fresh 身份/连接/价目复核 → 仅一次发送 → 原 usage/settlement；变更使旧未发送请求失效，已发送未知继续原键恢复。测试 exact SDK wire、串改字段/模型/输出、同键/并发、撤权/轮换、价目 FX 漂移、缺 usage、超上界、断链/超时，以及普通文本和现有单图/标题的邻近路径。不得因为另一个用途就绪而推断文本就绪。

## 示例预算（候选，不是批准或上线配置）

现有统一 GPT-5.4 Mini 标准价输入/缓存输入/输出每百万 token 为 USD 0.75/0.075/4.50。2026-11-11 [官方外汇数据](https://www.safe.gov.cn/AppStructured/hlw/RMBQuery.do?COLLCC=546877760)的 100 美元为 671.91 元，即 6.7191 CNY/USD；项目正式发布要求 CFETS 允许来源及有效时间，当前网页尚未取得可直接核验当日该记录的 CFETS 正式页面，不应仅凭此计算发布配置。isolated 定点测试以此作 fixture；上线时读当前统一价目/FX 重新计算。

保守候选一次规划上限 4,096 输入 + 512 输出，一次内容决策上限 16,000 输入 + 512 输出，内容沿原 `ModelCalls=2`、不扩大其他 Agent 限额。无缓存折扣、以 6.7191 测算，原定点代码向上取整：规划最多 3.612189 点；每次内容最多 9.611001 点；一次规划加两次内容最多 **22.834191 点**（约 ¥0.228342，实际以系统逐次冻结及结算为准）。该模型路径在合成小商品资料、模拟响应和两次内容决策下通过；非真实模型质量保证，长资料/512 输出限制可能失败。4096 输入用于内容时在 Claim 前失败，8192/9000 在示例中未完成，不能当成可用档位。

当前仅一个已查到成员月限 34 点，已预留 15.445282、已用 2.549024，算术余 16.005694 点，尚未证实是目标试用者；不释放、转移或触碰原预留。若由该成员试用，该候选满额上界不够，供用户考虑的方案是将该成员月限从 34 调整至 42 点，其他不变，则在数据不变时余 24.005694 点，覆盖 22.834191 点；这会增加最多 8 点月度使用权限，须用户单独明确决定，不能自动修改。成员实际权限、其他并发支出、汇率和价目在每次效果前 fresh 校验；数据或汇率变化使余量不足则不发送。这个估算只覆盖一轮规划与两次内容调用，不作为真实账单承诺；会员额度不是单次试用预算/现金扣款上限，也不能拦住其他已有用途花费。额外消息/重试沿原键/账本，不预设用户实际调用次数。

## 验证、交付和停止点

独立评审确认对七表全空、无旧事实承接的当前 schema 作定向演进不构成旧业务数据迁移；现场仍须重核和另行授权。除上述事务边界修正外，owner 权限与 DDL/ACL、提交前回滚、提交后 runtime 拒绝、ACK 不明、并发/重复安装及写入互斥均属 `IMPLEMENTATION_TEST`，必须在隔离实现验证。42 点只是条件示例，不是预算批准或单次现金上限。

实现阶段先按 TDD 覆盖目标缺陷，再用任务专属 PostgreSQL、原 SDK 和本地模型替身验证 schema 角色/失败/恢复、模型 wire 和 usage、共享 current/费用/预算、原产品 Agent 与 Chat 合成链。隔离 PostgreSQL、真实 SDK 配内存替身及合成业务链的测试结果以最终 PR 为准；任何成功不代表真实 OpenAI、真实库存/模型账单、原实例或产品验收。代码/现场安装、统一 FX 发布、成员月限变更、实例切换和真实模型调用分别是独立动作，不由本文批准。

## 现场准备命令与顺序（待审核，不构成执行授权）

空结构准备模式仅识别经隔离核对的 PostgreSQL 17 原始结构和角色；必须使用本次最终审核版本，owner 和 runtime DSN 由现有私密渠道分别提供，不能打印或写入仓库。IP 以数据库 `inet_server_addr()::text` 的当次只读结果为准，数据库、服务器及端口由操作者复核并显式提供。**本命令会执行 DDL，不得在未获现场操作授权时运行：**

```powershell
go run ./cmd/ai-workbench-schema-init --runtime-role ai_workbench_runtime --empty-scope-only --expected-database product_agent --expected-server-address <经核实的数据库IP> --expected-server-port <经核实的端口>
```

命令读取 `AI_WORKBENCH_SCHEMA_DSN` 和 `AI_WORKBENCH_RUNTIME_DSN`；不用命令行传密码。失败或提交回执不明确，Chat 保持关闭，先只读核对，不自动再次执行。前置包括当前一致备份及独立恢复、七表为空、结构与可检查依赖匹配、角色及精确目标复核、保留实例清单和全部业务卷；新旧版本的恢复窗口须事前确定，禁止旧备份覆盖之后新增的非 Chat 事实。

现场方案的后续动作依次是：由统一价格 owner 核验并发布当前有效的官方价目与允许来源的 USD/CNY；为同一企业连接和同一个共享文本模型引用声明规划、内容两个用途；校验试用者和可用余额，按用户决定处理月限；在授权窗口内切换经验证的后端并启用该企业 Chat，保留 ProductAgent 与原有预留。配置预览需包含两个用途的 prompt/schema、输入输出上限、价格版本、FX 生效期及失败处理；任一缺失则保持关闭。真实发送一次消息与现场启用分开授权，其前不发生模型计费。
