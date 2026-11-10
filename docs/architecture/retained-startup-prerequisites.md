# 保留实例启动前置：OwnIntake v3 与采集价格快照

Execution: #1233；关联 #1231 / PR #1232、#1158、#1178。Design Basis: **Independent Architecture**。状态：**NOT_READY，等待独立 Architecture Review**。

本轮仅设计和评审。实现、配置修改、DDL、副本启动/还原、原实例更新均未获本轮授权；下文为后续实施合同，不是已执行结果。

## 1. 用户结果、产品决定和边界

使用者是原 localhost:35444 试用实例维护者。目标是在当前原生系统的商品、R1、图片、采集任务/明细、权限和预留保留的前提下，解除两项已定位的新版启动前置；之后才继续原商品路径。不是开通新模型或恢复执行原采集。

依据：2026-11-10 本聊天用户明确要求最小方案、独立评审及保留现有任务/明细；[当前领域边界](project-boundaries.md)、[全新系统产品基线](../product/greenfield-no-legacy-migration.md)、[DATA_ROW 定价](data-row-platform-pricing-zero-orders.md)、[OwnIntake OpenAI Mini](own-product-intake-openai-mini.md)、[库存语义](own-product-intake-stock-review.md)、[AI 关闭时的完整应用](ai-disabled-application-lifecycle.md)。本设计仅以本次明确保留决定增补当前系统 schema 演进，不撤销全新系统基线或 #1178 的其他边界。

**Must**：

- 不删除、重建或改写已有任务/明细、身份、输入 hash、intent、结果、终态证据、时间、资源引用；不制造冻结价或免费事实。
- 配置版本对齐不换企业、连接、凭据、模型、价目/汇率、token/调用/成本上限，不重写历史调用/报价/预留。
- 缺价的失败记录可按当前权限只读查看；不能被执行、重采、按新价恢复、计量/释放或结算。正常新任务仍必须冻结合法 Commercial 快照。
- owner 显式维护、事务/并发/重入安全；不放宽启动、租户及有效权限检查。
- 副本全部验证后，才有后续原实例操作决定；本轮不宣称新版已可启动。

**Should**：错误明确到配置字段或具体任务/明细及拒绝原因，去敏输出，不打印配置/密钥正文。

**Out of Scope**：原35444配置/数据/服务/会话变更、应用模型/采集 provider 调用、平台保存发布、新报价/订单/调价/汇率发布、资源释放或结算（尤其原15.445282 AI点）、旧任务续跑、未知消费恢复、价格回填、其他 schema 缺口、通用升级器/验收平台。已有 #1232 四 CHECK 修复不改设计/实现、不塞入本次 Product 方案。

**Threat model**：误库/误 owner、ACL漂移、并发写/丢回执、缺价被当0、原键被当新执行、跨企业读/缓存串用、备份回退丢数据。不是建设新 IAM、迁移框架或 provider 未计费证明协议。只读价格缺失不证明供应商未计费，也不证明账本无孤立记录；一律不据此结算。

## 2. 当前证据与根因

源码基线 main `55b95607cd93263ee727295e15f0ded7170dc4c4`；#1232 实现 `9d6dea921c0810dfaf48e3afca696d3ef85798c8` 的隔离整机启动直接证据：

1. 原私有配置因 `own product intake policy exceeds original consumer admission` 拒绝。`agentconfig.OwnIntakePromptVersion/OwnIntakeActionVersion` 已是 prompt-v3 / candidate-v3，原消费与显式 binding 仍 v1。
2. 上轮仅任务配置副本对齐四字段后，继续在构造 Data Services 时拒绝。当前 `dataacquisition.VerifySchema` SELECT 必须读 `price_snapshot`；原表没有该列。
3. 本轮只读事务再次确认：Product DB 和两表 owner 均为 `acquisition_owner`；**1 个 amazon_data / FAILED job，1 个 amazon_data / FAILED item，reason=resource_unavailable；job.quota_reserved=0；item.reservation_id=NULL、charge_state=''、无 claim/lease/evidence/source/saved_at，有 terminal_evidence；无 SHEIN trial 行。** 不是可直接留空价格的 SHEIN 来源。
4. 原 job 无冻结价格/offer/version，原 intent 不含可还原售价的字段。旧代码固定单价、当前 offer、已付账单或未记录 reservation 均不是当次 Commercial 冻结证据。

根因是原生系统 schema / 配置随能力演进后没有对应的显式维护路径；仅补列不能解决读取和可执行对象解码，改 SELECT/默认0则会隐藏事实缺失。须同时明确缺价终态的当前领域语义、全部消费者和有界 owner 入口。

私密原证据位于 `C:/Users/Henry/.codex/private/empty-observations-contract-design-20261110/`：`acl-fixed-original-startup-proof.json`、`acl-fixed-v3-startup-proof.json`、`isolated-config-delta.json`、`independent-startup-blocker.json`、原始16库指纹。仅作输入，不能当本方案实施 PASS。

## 3. Owner 与调用路径

| 事实/合同 | 当前 owner / 路径 | 本次变化 |
| --- | --- | --- |
| OwnIntake版本 | agentconfig → currentapplication.ProductAgentConfig → modelRouteDefinitions → SharedCatalog / 原模型服务 | 精确私有配置四字段对齐；生产常量/校验不降级 |
| 当前价格 | Commercial offer / PriceReader | 不变；维护入口不读当前价补历史 |
| 任务/明细/冻结快照 | product/dataacquisition → integration/persistence/product/dataacquisition → app/dataservices → HTTP/BFF/UI | 一个当前合同增加受限缺价终态只读投影，仍在原表/原owner |
| 显式维护 | cmd/product-acquisition-init → app/productsourcing → 原 acquisition / dataacquisition persistence | 提议增加单一互斥维护模式，复用原私有 manifest、事务/权限能力 |
| 计费/预留/恢复 | 原 Commercial/Resource/AI invocation owner | 不变；缺价对象不得进入任何写恢复/结算 |
| Store CHECK | #1231 / #1232 原 Store owner | 独立前置，复用其未变通过证据，不复制维护协议 |

不增加表、第二价格源、旧 Service 包装、schema 版本双读或 fallback。受限缺价终态是一个当前数据状态，fresh installer/维护后 schema/领域解码/UI 共同消费同一合同；不以旧版本号走不同业务实现。

## 4. OwnIntake v1 → v3 配置最小对齐

原私有配置只读留 hash/权限清单，在任务私有副本中按**企业+operation+definition/connection身份**寻找唯一条目，不把曾观察到的数组 `[3]` 当稳定身份。找不到、找到多项或非精确 v1/v3 组合即拒绝。

| 字段 | 允许前态 | 唯一后态 |
| --- | --- | --- |
| productAgent.ownIntakePolicies[原企业].profile.promptVersion | own-product-intake-prompt-v1 | own-product-intake-prompt-v3 |
| 同profile.outputSchemaVersion | own-product-intake-candidate-v1 | own-product-intake-candidate-v3 |
| 匹配 modelRoutePricing.bindings[].text.profile.promptVersion | 同v1 | 同v3 |
| 匹配binding的outputSchemaVersion | 同v1 | 同v3 |

两个 profile 作为一份配置原子替换；完整私有配置解析/Validate/modelRouteDefinitions/SharedCatalog 准入一次验证，四字段以外逐字段/敏感值hash不变。已经完整v3则无变化；半改或未知版本拒绝，不能自动猜修。保留原文件和副本，写临时文件后同目录原子替换、保留当前私有ACL/0600挂载；不打印密钥，不新建通用配置升级工具。

显式 binding 是部署准入描述，不是已发布路由/价格。保留 connection/definition ID、reference version、BoundEvidenceVersion、adapter/usage mapping、price version、FX、企业/operation、费用和token限制；不能仅为绕开校验变更它们。v3来自当前代码的prompt/schema，不改旧 invocation、保存候选、历史quote/profile hash或原 reservation绑定。若当前价格定义读取因新profile失配则明确阻塞，不自动发布定义/费率/FX、不回退v1。

真实模型调用仍禁止。后续只做离线配置准入与受控模型响应，验证 v3 的结构/来源/数值规则，库存0与未填区分；不恢复语义关键词穷举。费用/点数准入沿原 owner，本轮不重新授权¥0.30/30点内的任何真实调用。

## 5. 缺失 price_snapshot 的最小当前合同

### 5.1 数据与展示：缺失不等于0

新增原列 `price_snapshot jsonb`，无 DEFAULT、不 UPDATE 原行。原1job和1item的所有既有列逐字节/值保持；原job新列为SQL NULL，不是 JSON null、空对象或0价。不改变任何原状态，不新增墓碑/归档表。

**有冻结快照**：原 `PriceSnapshot.Validate`、合法显式0价、原键/金额/计量规则保持。

**无冻结快照**：Amazon只允许受限 `FAILED` 终态只读。资格必须由原 owner 在一致快照中验证：job.quota_reserved=0；所有所属 item 均FAILED、terminal_evidence非空、无 reservation/charge_state/claim/lease/evidence/source/saved_at；无活动或已保存/待核实项；身份/FK/原intent的确定性结构、来源和作用域仍合格。不存在子项的FAILED也可在无预留/无结果/终态原事实齐全时只读。实际维护验收必须包含原1+1，不能以空库替代。

此资格表示“没有本次能力所需冻结价，只能读”，**不签发未计费/可释放证明**。仅终态字符串或一个 evidence UUID不足以证明零消费；不按reason关键词推导业务事实。

当前Job/HTTP/前端严格合同使用可辨别的价格状态：

- `priceStatus=FROZEN`：price为合法快照；confirmedFen/pendingFen保持整数分字符串。
- `priceStatus=UNAVAILABLE_TERMINAL`：price、confirmedFen、pendingFen明确为null，state只能FAILED、saved=0、pending=0；显示“价格快照缺失，仅可查看”，不显示免费或按0元价目折算。失败条数/原原因/原时间仍可读。

没有新持久化status列；该状态由原记录+null快照严格派生，不是客户端可写标记。只读投影仍由同一owner当前记录得出，不复制任务事实。前后端同版升级，不支持无status的兼容解码。

### 5.2 读取与执行必须分开准入

不能只让 `jobRow.job()` 跳过价格解析。当前Job可能被读和写路径共用；实现须让 `Read/List/ByCommand/Items/结果只读` 在同一个只读一致快照验证上述跨表资格，输出明确缺价状态。分页、overview及strict BFF也必须可解析，不让一条记录拖垮整个页面。

**任何执行/写入口**在使用价格、重算指纹、移动quota、写命令回执或调用provider/resource之前，必须从锁住的当前原行重验合法冻结价；不能信任传入Job字段。覆盖 Admit原键POST、Discover/FailDiscovery、Claim、BindReservation、PrepareEvidence、Publish、Fail/Cancel、ChargeIntent/ChargeProof、background恢复等同根因路径。缺价终态只允许GET原键读取，不允许原键POST重新准入或新价替换；返回可定位的冻结价缺失拒绝。涉及既有未知资源回调则返回原未知/拒绝语义，不产生bind/释放/结算。

`quotaMove` 和 `originalItemIntent` 不得用零值Price运算；所有调用者须有服务端当前冻结价前置。不能使用第二个模型补证。

Usage沿原按SAVED条目/原job冻结价聚合；这类FAILED无SAVED行，只参与失败数量，不增加金额。SQL不把缺价SAVED汇总成0；缺价SAVED/活动/有预留在维护及启动数据不变量校验中拒绝。单条缺价显示null金额不改变真实月度已确认金额事实。

### 5.3 Fresh与维护后的同一schema

由dataacquisition owner定义唯一当前列/约束：SHEIN保留必须SQL NULL；Amazon有价仍为有界JSON对象；Amazon无价只允许FAILED且quota_reserved=0。跨表的“全部失败/无写副作用”由同一owner一致性校验补足；CHECK不能跨表，不伪称DB已独立证明全部资格。

新Amazon准入仍先从Commercial读取/确认合法价，再在原Product事务插入；不得直接创建UNAVAILABLE_TERMINAL。fresh installer输出相同最终合同；VerifySchema和数据不变量检查拒绝未知/弱化结构、可执行缺价或有结果缺价，不为“能启动”全局skip。

## 6. 显式Product维护、权限与失败恢复

提议原CLI增加 `--align-acquisition-price-snapshot --confirm-database product_acquisition`，与空库初始化/所有feature开关互斥。不把现有 `InstallSchema CREATE IF NOT EXISTS` 当ALTER入口，不重跑Grant/初始化。

原私有manifest的严格解析、绝对路径/私有权限和合法loopback沿用；准确 `product_acquisition` 与当前native `acquisition_owner`，不接收任意table/SQL/角色。原dataServices/collection及已安装能力声明须准确完整；不删除配置隐藏已安装能力。

1. 维护窗口停止**任务副本**应用写入/恢复器；固定单连接原生PG事务，READ COMMITTED、30秒context、5秒锁等待、25秒statement。沿 `current_database()||':source-acquisition-init'` advisory key，再固定jobs→items ACCESS EXCLUSIVE锁。所有判断锁后完成，不复用事务外count。
2. 原owner确认库/schema/两表owner；完整两表A合同（列/类型/default/nullability/CHECK/索引/FK/flags）与完整当前B精确比较，复用pgschema还原等价能力；不只查“列不存在”。其他Product已安装合同、原角色权限/禁权、未知ACL/default/column/member、RLS/触发规则/非native对象漂移仍拒绝。
3. 复用现有有效权限primitive，缺价列的A结构由本owner单独核对，B由完整当前Verifier核对。**不能先调用只认B的VerifyDataServicesOwnerPermissions完整schema包装从而再次卡死A；不能反向放宽旧消费者或跳过有效权限。** 原 `data_services_runtime` 两表表级SELECT/INSERT/UPDATE自然覆盖新列，不加GRANT/REVOKE；任何额外权限不自动修。
4. 锁后检查所有待补列行的受限终态资格；若有活动/已保存/任何未决引用则整次拒绝。精确A→B只加一列和当前价格CHECK；禁止重命名/替换两表、改其余约束/数据/owner/ACL/history。B重入只验证，无DDL/数据写。
5. 前后比较所有既有列、原关系identity/OID、约束/权限、历史及其他Product事实；新增列/指定CHECK是唯一允许差异。事务内完整B与跨行不变量验证，commit后只读再次核实。无新Goose行/维护事实表。

单Product DB事务已足够，不跨Commercial写，无Saga/outbox/新恢复器。DB锁竞争让writer先胜时检查实际新行；不合格则拒绝，不能漏入。与其他owner维护串行，不能在持库锁时改文件/调用外部服务。

失败前/事务内取消/DDL错误：PG回滚全部；配置副本不能先投入运行。提交响应丢失：不盲目执行另一命令或Down；原入口重入判A/B，B后置完整则已完成，A可安全重试，混合/未知拒绝。配置替换与DB不是跨资源原子事务：维护窗口保持服务停机，以B校验+完整v3配置hash两项齐备才启动；任一未齐备不向用户暴露半更新。

资源界限：只处理固定两表、固定schema增量；不导出secret/evidence正文，不新增无限扫描runner。结构与NULL资格查询受上述deadline/statement timeout约束，超时拒绝保留原状，不能截断前N条宣称全部合格。

## 7. 后续副本验证与回退（本轮 NOT_RUN）

复用 #1232 现有16库逐库同snapshot dump/摘要、角色/ACL、对象hash和标准镜像构建/启动入口；新副本/端口/容器/卷有准确任务前缀、内网无provider出口，原对象挂载只读。逐库快照不是跨库原子快照。若下一阶段要写副本/启动，须另获用户实施/演练授权。

实施必须按四组义务收敛，不建立新的专项验收平台：

1. **配置**：原v1拒绝；完整v3通过当前真实配置/catalog链；唯一绑定定位、半改/重复/错企业拒绝；只四字段差异；模型/密钥/费率/FX/预算/原invocation及预留hash保持；受控v3候选保留黑M0/黑L空/白M，不真实调用。
2. **数据/权限接线**：真实A副本1+1不删不改，NULL终态在列表/详情/原键/strict BFF显示仅可读；fresh B相同；误库/非owner/弱CHECK/权限漂移拒绝；原有效权限primitive必须实际跑到A→DDL→B，检查同根因包装路径。
3. **并发/重试/行为**：两个维护者、writer-first、锁取消/DDL中断、commit回执丢失+B零DDL重入；所有缺价写入口及资源proof/recovery拒绝；新付费/真实0价使用受控合法快照正常，缺价活动/SAVED/预留负例仍拒绝。无重复任务/明细/资源操作。
4. **整机/回退**：标准显式Git镜像，同时应用 #1232 的独立Store前置、Product本增量和v3配置副本，验证后台启动/UI可读及原商品/R1/SKU/图库和任务记录；无模型/provider/发布动作。实际登录/原用户只读访问若未做须另列NOT_RUN；单401不替代完整路径。

回退不执行DROP COLUMN/反向CHECK/Data Down。停止全部任务后台与恢复器，恢复**相同时间点的任务副本**Product与Store及如被后台写入的其他库、原v1配置和对象清单；使用准确当前业务源 `0f41a88...` 加已批准 #1225 比较器修复的回退源 `703744a60...`、原UI版本，按标准builder精确source/tree留证。不能拿更老7dec后端替代当前业务基线。

回退后后台启动/原UI以及原商品/R1/1job/1item读取、16库既有列数据/ACL/hash、原预留状态/金额和对象bytes核对。后台活动如只改next_check_at也要如实列差异并恢复任务副本，不能把容器健康当数据完全相同。原35444不参与restore/DDL/重启。没有业务新写的演练窗口中可回原快照；若窗口内出现新业务写入，则停止自动回退，不能用旧dump覆盖新事实。

如发现第三个独立启动阻塞，记录owner和阻塞层级，先停止整机通过宣称，不顺便补列/禁模块/扩配置；该项不自动扩本范围。

## 8. 决策、实施停止点与评审

不选择：清空任务、将Amazon改成SHEIN、回填0/当前价/固定5分/假offer、只加nullable列却保持执行解码默认值、跳过DataServices或恢复v1容错。以上均会丢事实、错误计价或隐藏未满足合同。

本设计选择“原记录不变 + 当前明确缺价失败终态只读 + 新执行严格冻结价”。它只保留当前系统已有事实，不赋予旧任务跨版本执行/恢复义务。不能把设计准入当作实施/原实例操作批准。

架构评审按本轮Must/Threat Model判断BLOCKER；最多两轮正常审查，非阻塞细化列IMPLEMENTATION_TEST或BACKLOG。正式实现前必须达到IMPLEMENTATION_READY，并获得用户下一阶段实施/副本演练决定。当前生产代码、维护命令、副本验证、真实启动/用户验收全部NOT_RUN。
