# 市场研究与选品：最小领域合同

Design Basis: **Independent Architecture**。状态：**FROZEN / IMPLEMENTATION_READY**。
依据 [#1071](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1071)，实施 [#1072](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1072) 与 [#1073](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1073) 的本合同最小切片，父目标 [#1070](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1070)。用户于2026-11-02明确批准本版设计、服务端 Amazon 先贯通、浏览器桥后接，并授权隔离实施；不合并、不部署，真实平台和付费调用仍未授权。独立 R1 及对应增量复核无剩余 Architecture BLOCKER，产品确认已满足，允许按冻结合同实施。实际完成、CI、真实运行与用户验收分别记录于同一主要 PR #1075，本文不代签。

## 1. 用户结果与产品边界

当前企业成员从项目中心建立或打开自己的「产品研究」项目，选择 Amazon 站点及关键词、类目或 ASIN，经现有数据市场明确确认采集范围和最大用量，关联已保存证据，筛选 2–5 个候选，按需发起现有选品对比，记录人工决定，再通过原商品流程转入「我的供应链 → 待适配」。仅查看证据和人工决策不要求模型可用。

首批按用户决定以现有服务端匿名 Amazon 通道贯通；真实可用站点和可得字段必须实际验证，不能预先宣称。Browser [#31](https://github.com/henryxu1008/openbrowser-platform/issues/31) 沿相同证据消费合同接入；其设备配对、单任务授权和采集执行仍须自己的设计准入。本选择不取消父 Epic 的浏览器交付，也不把它伪装为已经接入的备选通道。

### Must

- 一个项目身份、一个商品事实 owner；证据、模型建议、人工决定和商品准备分别表达。
- 准确来源、站点、采集时间、版本、parser、校验和与缺失信息可追溯；观察不等于销量、趋势、利润或市场总体结论。
- org + actor 私有，不能因企业管理员身份、项目关联或浏览器回执获得他人的证据权限。
- 2–5 个候选的事实比较与模型可用性分离；模型调用仍需原显式配置、预算确认、权限和计量。
- 人工「批准测试」只表示研究决定，不是合规批准、资产批准、上架批准或采购许可。
- 新关联/决定同键重放不重复写；收费采集、AI 和转入的 UNKNOWN 均回到原 owner 核实。
- 缺数据、混币种、挑战、离线、撤权、来源不再可读、未知写响应均显示真实状态。

Should：沿现有 Console、项目类型、Data Market 表单、CollectionPicker 和供应链流程，保持明暗主题及 390px 可读。

Out of Scope：#1074 的 1688 自动匹配与利润模型、定期监测、销量预测、自动选中/采购/刊登、市场共享数据集、研究团队协作、新 IAM/计费/队列/调度器、第二项目中心、第二商品库、通用跨库事务或验收平台。

Accepted Risks：本轮未新增风险接受决定。公开页面可能无数据或受挑战是诚实失败结果，不能作为绕过限制、编造指标或降低当前 Must 的理由。

产品依据：当前执行 Issue、[全新系统基线](../product/greenfield-no-legacy-migration.md)、[UI Authority](../product/final-ui-ia-authority.md)、[当前领域边界](project-boundaries.md)。不从旧计划或历史测试恢复已撤销需求。

## 2. 当前能力矩阵与真实通道前置

以下是对开工 main `b324a044cfea4c83e4a80eece440d6515693b6ab` 的源码核查；「存在测试」不是本轮执行 PASS，合并也不代表实例已启用。

| 能力 | 当前实现与证据入口 | 判断与本次缺口 |
| --- | --- | --- |
| Amazon 查询与发现 | `internal/product/dataacquisition/query.go`、`internal/integration/acquisition/amazon/client.go` 的 NormalizeQuery / Discover / Fetch | 已实现 keyword/category/asin，最多 200 条、发现最多 10 页；Readiness 只检查显式 driver/browser 文件与配置。真实站点本轮 NOT_RUN。 |
| Amazon 解析 | `amazon/parser.go`、`parser_test.go` / `client_test.go` / `readiness_test.go` | 已有页面结构、价格币种、挑战及出口限制测试；当前 parser 明确缺 variants/rating/reviewCount，没有销量/趋势事实。补字段须单独证明，缺项可以保留未知。 |
| 来源与商品保存 | `internal/app/dataservices/publication.go`、`results.go`，Product dataacquisition persistence | Evidence → SourceEnvelope/receipt → Catalog → Collection 在原 Product UoW；Results 核对原 publication 与 Evidence checksum。不得重新建商品快照。 |
| 数据服务实际装配 | `cmd/current-application/main.go`、`internal/app/runtime/currentapplication/data_services.go`、`internal/app/httpapi/current_data_services.go` | 已有显式安装/受限 Product pool、准确 IAM、Resource、Temporal 和 worker 接线。PR #623 已合并；不能从此断言本机实例启用或采集成功。 |
| 1688 和插件来源 | `internal/product/sourcing`、`internal/app/productsourcing`、`extensions/product-capture`；现有 capture 接收和 Collection | 当前获取/保存能力可复用；不等于货源匹配、利润估算或远程浏览器授权已交付。 |
| 项目身份 | `internal/aiworkbench/projectcenter/contracts.go`、`service.go`、既有 persistence/HTTP | 已实现 PRODUCT_RESEARCH 类型及私有项目；现有 PRODUCT 引用解析为 acquisition operation UUID，不接受任意 Catalog/ProductKey。不能假定它已支持 Amazon Data Services job 或研究 owner 引用。 |
| 商品对比 | `internal/product/selection`、`internal/app/httpapi/product_selection_*`、前端 `agents/product-selection-flow.tsx` | PR #681 已合并；2–5 个准确 Collection/Catalog 引用、事实矩阵、显式受治理分析/原键恢复。批准合同不含市场统计、研究证据 DTO 或商品 mutation。 |
| 转入商品准备 | `internal/listing/preparation/service.go`、`model.go`；Preparation schema 与 Collection capability/UI | Transfer / ReadByKey 已有单批非空 ItemIDs 能力，但当前 schema 的 source_kind CHECK 和 Collection supply_transfer_supported 排除 amazon_data；事务 callback 当前仅显式拒绝 sds_template，前端只提交整批。必须补 §5.3 的 Amazon 准入、准确事务校验和原弹窗子集消费，不能称现成路径可用。 |
| 研究证据关系与机会决定 | 当前源码未发现对应领域实现 | 本设计新增的最小事实；不能以 README 的“市场调研”示例当交付。 |
| 本地浏览器研究桥 | Browser #31；相关商家环境工作在 commerce #1019/#1020、Browser #14 | 当前为独立待设计范围；既有启动/CDP/插件能力不等于 SaaS 最小授权与研究任务回执。商家 Store 流程不自动成为匿名研究的前置。 |

真实服务端通道需要：获准的新空安装或明确授权的实例、受限 Product/Resource pool 和 schema、准确 IAM 与原生模块权限、显式 driver/browser、配置站点、现有 Temporal namespace/worker，以及可用 DATA_ROW 资金来源。沿[运行交接](../engineering/data-services-runtime-handoff.md)和[原生装配](../../deployments/docker/account-compose/data-services.md)核实；不读取凭据或变更保留实例来完成本轮设计。

本轮真实 Amazon/1688、模型、Native 浏览器、付费数据和用户验收均 **NOT_RUN**。无实际通道证据时，只能交付代码/设计准备，不能声称研究业务可用。

## 3. 事实 owner 与最小持久化

| 业务名词 | 唯一 owner / 复用决定 |
| --- | --- |
| ResearchProject | 复用 `aiworkbench/projectcenter.Project` 的 PRODUCT_RESEARCH 类型；项目名称、目标、归档、模板、访问仍归 projectcenter。研究页不复制这些字段或新建项目状态机。 |
| MarketObservation | 原 dataacquisition/SRC/capture 保存证据的授权投影；研究只持有准确引用和关联关系，不保存原 HTML、商品正文、Cookie、价格副本或另一个证据快照。 |
| ProductOpportunity | 拟新增 `internal/product/research` 的项目私有候选关系与人工决定；它不拥有商品、采集任务、AI 结果或 Preparation 结果。 |
| 商品、批次、建议与准备 | Catalog 独占商品；Collection 独占成员/批次；Agent 独占 run/proposal/history；Preparation 独占转入事实。研究关联不改变这些权限。 |

拟新增 `product/research` 中的三类记录，放在当前 Product 物理数据库、通过 feature-local 受限 serving pool 访问：

1. `research_evidence_links`：scope、projectID、关联 UUID、原来源种类/operation 或 job+item locator、准确 Collection item/publication/version、原 owner checksum 身份。唯一性包含 scope + projectID + 原 observation 身份；不缓存来源正文。
2. `research_opportunities`：scope、projectID、候选 UUID、站点/ASIN 的归组身份、准确关联集合、当前人工选定 evidence ref、正 revision、决定及用户理由；可存原 Agent run locator，不复制输出。站点/ASIN 从可信来源派生，客户端无权指定事实身份。
3. `research_commands`：scope + UUID key、operation、规范 payload hash、原结果 ID/revision 的不可变回执；同时保存本 owner 的决定意图、理由、决定时间、当时 evidence refs/digest、关联 run locator 和提交 revision。与关系/决定同事务写入，构成最小决定历史，不存来源正文或模型输出。读取历史仍逐项授权来源与 run，失效内容遮蔽。

用户已授权本冻结设计的隔离实现与新空测试安装；最初未授权变更保留实例或业务数据库；用户后续于2026-11-02明确批准仅更新 `task-processor-unified-20261010`，适用下方“指定实例启用决定”，不扩展为其他环境授权。installer 按当前显式安装约定实施并验证最小权限，serving 不 DDL，不给研究角色其他 owner 表的任意读写权。Project DB 保持原逻辑隔离，不搬进 Product DB，不建立跨库外键或事务。

ProjectReader 通过当前项目 Service 读取：项目必须属于相同 org + actor，新增关联/决定还要求项目 active 且类型为 PRODUCT_RESEARCH；归档后只读历史。修改项目类型不删除研究关系，变回研究类型后可继续，研究内容不作为 Project 元数据事实。项目中心无需扩展 PRODUCT 引用种类：研究关系从自己的 projectID 索引投影，项目中心仅增加明确的研究入口。

## 4. 来源合同、去重与真实性

中立的 `ObservationRef` 包含 source kind、owner-issued locator、Collection item ID/revision、ProductKey、CatalogVersion、PublicationID、原校验身份。浏览器与供应商输入不直接成为它；服务端经当前来源 owner 验证并导出准确引用。

`ObservationView` 在授权读取时投影 platform/site/sourceID、来源 URL、capturedAt、parser/version、checksum 的算法及语义、已观察字段、missing/warnings、货币和单位、查询/样本范围、渠道。字段分为 `OBSERVED`、`ESTIMATED`、`UNKNOWN`；本版不生成估算指标。若来源没有 parser、时间或 hash，明确缺失，不用当前时间或重算的 Catalog hash冒充原证据。服务端 Evidence checksum 是结构化采集证据的校验，不声称是原始 HTML hash；浏览器 contentSHA256 也保留其既有语义。

Source reader 沿原 Results/SRC/Collection 精确核对，不读 current Catalog head 替代原版本。研究不得按任意 ProductKey 绕过 Collection；也不能把前端价格、模型描述或 Browser task ID 当成事实证明。

当前 Results 按原 query.Fields 投影，既不返回 Collection ItemID，也不开放 parserVersion/checksum 字段。因此在原 dataacquisition/SRC owner 增加窄的实时授权 provenance reader/DTO，核对原 job/item、SRC receipt/checksum 与准确 Collection 关系后，返回上述元数据。保持原外部 Results 字段投影、计量和事实 owner 不变；研究不直读其他 owner 表，不重新计算 Catalog hash 代替原证据。未满足原授权或无法验证绑定时不产生 ObservationRef。

- 同来源原键重放/重复关联：复用原 observation，不重新采集、计量或新增证据。
- 同站点同 ASIN 在项目内归为同一候选，不同时间/渠道的 observation 保持独立；跨站点不合并币种/价格，子 ASIN 不按图片相似合并。
- 新 observation 不覆盖已分析/已决定的证据；用户明确切换 evidence selection，CAS 增加 revision，清除当前决定并回到待研究。旧决定/原 run 身份保留在原回执/历史中。
- 输入查询/采集时间范围只代表本次样本。覆盖率仅是已取得字段/样本完整度，不声称市场份额、需求强度或总体代表性。
- 删除关联只解除本项目关系，不删除 SRC/Catalog/Collection。原来源删除、归档或撤权后，研究投影遮蔽其身份/URL/正文/指标，不从副本、旧响应或模型输出补回。

## 5. 用户路径与跨 owner 一致性

### 5.1 创建研究与取得证据

先由项目中心正常创建项目，再从研究区域打开原数据市场表单。站点/查询/字段/limit 和最大 DATA_ROW 用量沿原合同显式确认；后端仍使用原 Data Services UUID、Funding 和 Temporal，不创建研究自己的采集任务或扣费路径。

采集完成后，用户从本人原任务/我的数据中选择已保存结果，明确关联到项目。关联前失败不影响原任务；关联失败后原 job/批次仍可在数据市场/我的数据找回并重试关联，不再发起采集。本版不承诺项目创建、收费采集和关系保存三者原子成功，不自动补偿已保存商品或退费。

Project/source 的授权和身份读取在关系事务前完成；事务内只写本 owner 关系/决定/回执。Project 与 Product 数据库不做跨库写入；提交期间来源失效不能赋权，提交响应及后续 GET 必须重新授权来源，返回不可用关系槽而不泄露原始内容。不可用槽保留研究自己的 slot ID，供本人解除关联。

### 5.2 候选、分析与人工决定

候选关系初态 `TO_RESEARCH`。用户显式提交当前 evidence selection 供人工审阅后为 `PENDING_CONFIRMATION`，不要求模型已运行。本人以准确 revision、evidence digest 和理由设置 `APPROVED_FOR_TEST` 或 `REJECTED`；允许以新命令返回待研究，重新分析不自动批准。所有转换都是本 owner 局部事务，不改变 Product/Agent 状态。

选品事实对比继续调用原 `/workbench/product-selection` inspect/prepare/run 路径，只消费当前 2–5 个有效 Collection item 的准确绑定。当前 Agent 仅比较其已支持的 Catalog 事实；研究页并列展示来源时间/缺项供人核对，**不得把它标成已完成市场证据分析**。AI 停用/配额不足不阻止证据查看和人工决定。

#1073 如需模型消费 ObservationView、商业权重或合规风险工具，应在自己的正式实施前补足 Agent context/tool/proposal 的版本合同与相应准入；本文未默许向旧 prompt 拼接任意研究 JSON，也未改变现有模型准入/计价。父 Epic 的“市场证据分析”要求继续保留，不能用本版商品事实比较代签。

人工状态不替代确定性风险：侵权/禁售/认证检查未执行为 UNKNOWN，已明确的硬性禁止不能用模型高分消除；后续 Listing 沿当前有效 gate，本文不新建通用合规引擎。

### 5.3 转入当前商品准备

**当前存在必须补齐的正式准入增量。** `preparation/schema.go` 的 source_kind CHECK 与 Collection `batchReadColumns` 的 supply_transfer_supported 当前仅接受 acquisition/own/market/sds_finished；`preparation/repository.go` 的事务 callback 当前只显式拒绝 sds_template，没有相同的四类 allowlist。Amazon Data Services 保存为 amazon_data，现有测试明确不支持转入。此设计提出仅增加 **amazon_data**，并让事务来源校验消费相同的准确准入集合，不顺带开放 custom_dataset、sds_template 或其他未授权来源。

新空安装的原 Preparation installer/VerifySchema、事务来源校验和 Collection eligibility 必须同时消费该准确来源集合。Amazon 来源仍来自原同库 Collection/SRC/Catalog，检查当前 org/actor/member、active item、batch revision、publication/version 和 owner 原证据；保留原 FK、事务、回执、权限及受限角色。Preparation 只保存原来源引用，不改为 acquisition 类型，不复制商品，不以后台脚本改 CHECK 逃过 installer 准入。原阶段未授权旧实例 schema 演进；后续仅指定统一实例按下方决定执行精确、有界的显式维护，不允许 serving 自动修复。

批准后定位原批次，并**补原 Collection 转入弹窗的商品子集选择与准确预填**；当前弹窗仅发送 batchId/expectedRevision，不能原样复用。研究预填只是建议，不是可信来源；BFF/服务端沿原 owner 重读当前批次和商品，核对批准时的准确来源，变化则提示重新选择/确认，不静默替换为 current head。用户看清本批具体商品后再次确认。原 Preparation.Transfer 是唯一写入者；研究入口调用时必须提交原 BatchID、准确 batch revision 和**非空** ItemIDs，空选择禁止提交，不能退化为整批。原独立整批操作仍明确标为“整批”，不改变其已有语义。跨批候选分别走原批次确认，不新增批间自动 fan-out、恢复器或全部成功承诺。

研究 UI 不维护“已转入”布尔事实；只有当前授权的 Preparation 原回执/来源读回能显示已转入，缺少回执为未核实。批准只是决定；转入失败不回滚批准，也不伪造 Preparation。原响应丢失使用原 Transfer key 调 ReadByKey，不从研究再生成新 key 重投。

原商品流程允许用户独立选择其他商品，不把研究批准变成全局权限前置；用户离开研究流程后修改原选择属于新的显式商品操作，不能冒称仍执行原研究决定。

## 6. 权限、事务与恢复不变量

研究读/写分别复用现有 `workbench.project.read/manage` 与 `workbench.collection.read`，scope 取 verified identity、当前 effective org/member。项目 metadata 读写由 Project owner 再授权；任何来源内容须其原读取权限。原 Data Services、Browser、Agent、Supply 操作仍分别要求自己的 grant。研究不授予权限，不为企业管理员开放他人项目或来源。

所有 repository predicate 包含 org + actor + projectID；不信任客户端 org/actor/role、ProductKey、金额、批准标志或 raw evidence。当前身份和项目权限失效时整体拒绝；仅来源失效时只返回安全不可用槽，不返回受保护标识/标题/计数/历史模型输出。存储的 run locator 不赋予 Agent 读取权，也不让其旧输出恢复已撤权来源。

关联/选定证据/人工决定使用 UUID Idempotency-Key + operation + 规范 payload hash。同键异载荷冲突；原回执、关系变更和 revision 在同一 PostgreSQL 事务，按 scoped key 锁及项目候选行锁串行。外部 owner I/O 不在锁内。原回执重放只返回结果身份与提交 revision，客户端再读当前状态，不能用旧响应覆盖新决定。

| 边界/事件 | 唯一责任方与结果 |
| --- | --- |
| 研究关系/决定提交前失败 | research 本地回滚，原 owner 事实不受影响。 |
| 研究 COMMIT 响应丢失、重启或并发 | research 按原 key 查询/重放 receipt；同 key 同载荷只提交一次，同 revision 的不同决定只能一个成功。无后台任务，无新恢复器。 |
| Project 先创建、来源先保存而关联失败 | 原 owner 成功仍保留；用户从原列表找回并重试关系。没有跨库删除/退款补偿。 |
| 采集取消、挑战、超时、收费 UNKNOWN | 原 dataacquisition + Resource owner 核实原 reservation/job，研究只投影结果，不能重采、补扣、释放或换渠道。 |
| Agent 未配置、调用 UNKNOWN | 原 Agent/Resource 核实原 request/config；研究只保留 locator，不能直接调用 provider、重开 run 或承诺无扣费。 |
| Preparation 响应 UNKNOWN | 原 Preparation ReadByKey 核实；研究无第二 transfer receipt 或重试状态机。 |
| 归档/撤权、切换 org、迟到响应 | 归档阻新写；撤权 fail closed；UI 用 org+actor+session epoch 隔离，A→B→A 的旧响应不能恢复原内容。 |
| 原来源不可读/缺失历史材料 | 遮蔽受保护投影，保留安全关系槽，不从当前 head、其他渠道或模型补证。 |

新 HTTP/BFF 沿 feature-local descriptors、正常 session/Origin/CSRF、LiveWrite scope。JSON 严格拒绝未知/重复字段，写 body ≤64 KiB；一批关联 ≤200 个观察，候选分页 ≤50，比较 2–5；研究只读请求 deadline 10s、每页来源解析总预算 2s，超限明确不完整，不把未读取算作无数据；原 owner 更严格的限制优先。AI/采集不占用研究 GET 请求，不在 GET 隐式执行。

## 7. Browser #31 的准确边界

浏览器只提供用户允许的站点/任务参数采集、受控执行和原回执；不拥有研究项目、候选或商业决定。渠道返回的采集内容仍沿既有 capture 接收 → SRC/Catalog/Collection → 授权 ObservationRef，不能以签名 task receipt 绕过正文验证或用户确认保存。

#31 必须自行明确配对与单任务授权、org/actor/device/任务绑定、期限与撤销、去重/原键核实、取消/离线/迟到结果及敏感数据限制；本设计只定义其研究消费者端口，不提前批准传输或执行协议。数据服务失败不能静默切本地浏览器，Browser 失败也不能自动改付费供应商；换渠道是新的显式意图和原 owner 的授权/用量确认。

真实 Cookie、代理密钥和 Profile 不上传；遇挑战停止，不能自动登录、换代理、解 CAPTCHA 或开放公网 CDP。已有 Store 绑定仅在研究确实消费该受管环境时适用，不强制为匿名研究创建店铺。

## 8. Product / Figma 已批准增量

当前已只读核对文件 `tg48P46SSXl6TBy9lZwg63` 的非归档页 `31:463`：数据市场深色 `430:323` / 浅色 `430:525`、项目详情 `568:365`、项目中心当前总览 `9634:1265` / `9634:1585`，以及 MyAgents `4703:429`。现有项目详情含历史示例进度，不能直接复制成真实研究进展。

批准增量仅在 PRODUCT_RESEARCH 项目详情增加「研究选品」业务区域；项目中心/数据服务/我的智能体的原导航归属不变。数据市场负责取得资料，研究区域负责关联、证据、候选与决定，原我的数据负责转入。

后续阶段说明：用户随后明确确认[两个中心 V1 产品设计](../product/market-research-selection-centers-v1.md)，新增研究/选品入口的目标替代本段首批导航范围限制；新增 AI 合同见[独立设计增量](market-research-evidence-analysis-v1.md)。本冻结合同的事实owner、权限、状态与恢复不变量不变。当前实例仍运行首批实现，新阶段正式代码和实例更新另行授权。

内容顺序：项目目标 → 选择研究资料（打开原数据市场 / 关联已保存资料）→ 证据与缺项 → 选中 2–5 项事实对比及显式 AI 操作 → 人工决定 → 前往原批次转入。动作各自确认，不以查看页面隐式执行采集/模型/转入；空状态也能继续正常原流程。价格、数量、时间等示例只用于设计排版并显著标记，不能成为产品默认数据。

明暗主题复用现有 SUMI semantic variables、Noto Sans SC 和 Console 组件；390px 使用顺序卡片，证据/缺项/操作仍完整。用户已确认下列可编辑设计；节点名称中的候选字样保留设计阶段追溯，不再代表准入未满足：[深色桌面 11981:919](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=11981-919)、[浅色桌面 11981:1006](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=11981-1006)、[深色手机 11981:962](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=11981-962)、[浅色手机 11981:1049](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=11981-1049)。原已批准 frame 不变。产品确认已记录于 Product Authority；原批次弹窗子集选择按下列同版设计和 §5.3 实施。

原弹窗的子集转入设计同样已批准：[深色桌面 11984:966](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=11984-966)、[浅色桌面 11984:1005](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=11984-1005)、[深色手机 11984:985](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=11984-985)、[浅色手机 11984:1024](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=11984-1024)。仅补原操作的明确子集确认，不建立研究专属转入页或结果事实。

## 9. Contract → implementation → injection → consumer

| 合同 | 拟实施位置 | 注入/消费 |
| --- | --- | --- |
| 项目私有身份与 lifecycle | 当前 projectcenter；研究定义窄 ProjectReader port | app 注入当前 Service 的 scoped read；项目详情投影业务区域，无跨库 join/写入。 |
| 原 observation 精确读取 | 原 dataacquisition/SRC owner 补窄 provenance reader/DTO，研究消费；capture/Collection 原 owner | app 窄 adapter 调原授权方法，明确补 Collection 绑定和原 parser/checksum 元数据，不向 domain 暴露 GORM/provider SDK。 |
| 关联、候选、人工决定、回执 | 新 `internal/product/research` 与 `internal/integration/persistence/product/research` | feature-local `research/httpapi` + 原 descriptor registrar；currentapplication 显式 opt-in/schema 验证/受限 pool。 |
| 事实比较和分析 | 当前 product/selection + Agent/配置/Resource | 原 inspect/prepare/run API 与完整确认流程；不发明第二 Agent runtime。 |
| 商品准备 | 当前 Collection / listing/preparation 补 amazon_data 准入；原弹窗补子集选择 | 同步 installer/readiness/事务 allowlist/eligibility，明确非空子集，仍原 Transfer/ReadByKey。 |
| 前端 | 当前项目详情组件/同源 BFF/窄 contracts | 原权限和 context epoch、Console theme、原操作恢复 UI；未配置时真实不可用。 |

复用 PostgreSQL/GORM 的现有局部事务与 receipt/CAS 模式、Playwright Go、Temporal、Eino、Casbin 及仓库已有组件。没有当前 Must 需要新 Saga、调度服务、数据供应商平台或自造通用框架。Legacy decision: **N/A**，无旧系统迁移、双读双写、wrapper 或 fallback。

## 10. 实施顺序、验证与交付

同一主要 PR 连续交付本设计与已授权实施。独立 R1 围绕新事实、授权、幂等和 owner 边界；正常评审最多两轮，后续非 Blocker finding 按 IMPLEMENTATION_TEST/BACKLOG 收敛，不要求理论场景清零。产品增量、首批源通道和独立评审均已满足，当前冻结合同为 IMPLEMENTATION_READY。

准入后，一个 Writer、一个主要实现 PR 连续完成 #1072 的研究关联/候选、#1073 最小人工决定与现有比较/转入消费。Browser #31 在其仓库按自己的合同实施；无需为每个模块拆 PR。#1073 的真正市场证据 AI 分析、权重及其他尚未覆盖验收仍留在其 Issue，不因最小路径交付关闭它或父 Epic。

业务规则先 RED 再 GREEN，复用现有测试和隔离数据库，不新建 runner：同 scope 准确证据关联和 provenance reader、同 observation 去重/不同时间不覆写、跨 org/actor/撤权遮蔽、同键异载荷与并发决定、批准后换 evidence 仍可追溯原决定/理由/digest、响应丢失原键恢复、缺价/混币种/模型不可用、Amazon source kind 的 installer/readiness/真实 Transfer 与错误来源拒绝、原弹窗非空子集不带入未选商品和跨批分别确认、归档/迟到响应。原计量与执行代码不变时引用已有证据，只补本次组合风险，不机械全量重跑。

实际使用验收另行授权：正常登录自己的企业 → 创建产品研究项目 → 在获准站点真实采集 → 查看原证据/缺失并关联 → 比较候选/可选实际模型分析 → 人工决定 → 原批次确认转入 → 在待适配读回同一 publication/version。截图、fixture、代码评审、CI、真实平台、用户验收各自记录，未执行保持 NOT_RUN。

实现交付必须提供正常访问入口、启停/数据保留方法和已配置通道/功能限制；不因交付要求操作保留实例、初始化现有业务库或调用付费服务。合并、部署、真实环境/数据和 Issue 关闭均需对应明确授权。

## 11. 指定实例启用决定（2026-11-02）

用户在实现交付后指定 `task-processor-unified-20261010`，并明确允许可恢复备份后新增研究表/受限角色、定向扩展 Amazon 来源约束，更新并短暂重启前后端。身份、原模块配置、全部业务数据和卷保留；不合并、不关闭 Issue，不调用真实平台/模型/付费服务，不删除或重建现有数据。此决定仅替代与该指定实例有关的旧阶段禁止，其他环境仍未获授权。

Operational Design Basis: **Reuse Existing Architecture**，复用本文冻结的 Product/Project/Source/Preparation owner、原角色最小权限、PostgreSQL 局部事务、固定镜像与配置回退。没有旧系统数据迁移、ID 映射、双读双写或新的通用恢复机制。

实施前固化实际容器/镜像/挂载与数据库备份；校验现有 Preparation source CHECK 恰为 acquisition/own/market/sds_finished，再用有界锁等待、同一事务显式扩展为本文批准的五类。新三表与 restricted role 依批准 SQL 安装，不重跑历史 initializer；schema/角色漂移则停止，不自动修补。用户数据不变，Project 保持原独立数据库。原 Browser Kernel 配置与其他模块 settings 保留。 现场确认当前 manifest 未含研究所需 Data Services 子树；仅复用该实例仍保管的原显式 Data Services 配置与受限角色，作为本次必要接线补回，不替换其他配置。启用时原 data_acquisition_jobs 为空，不提交采集、模型或计费任务。

回退以先前镜像与配置为主，保留新增 durable records/回执与所有卷；出现新 Amazon 转入后不得为了回退收紧 CHECK 或删除记录。备份还原是异常处置，不在正常镜像回退中自动覆盖持续写入的数据。实际安装、启动、健康与用户试用分别记录于 #1072/PR #1075，合成/CI 不代真实平台和用户验收。
