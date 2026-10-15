# 商品内容标题试用与现有 Chat 共存的最小运行接线

Status: DRAFT / NOT_IMPLEMENTATION_READY。执行 Issue #688，父项 #137。Design Basis: Independent Architecture；只审本次运行组合与既有 owner 的有界安装入口，不重开内容、Skill、四权限或预算架构。准入前不得修改正式业务路径；最终文档、准确提交、独立边界检查与 Ready 记录须先于实现。

## 用户结果、授权和非目标

原保留程序继续提供已保存的 Chat 会话和个人 Skill 管理；准备一条企业商品内容 title-only 路径，供用户随后提供模型/预算并另行授权后，选择准确个人 Skill 版本、生成建议、人工采用到标准商品并刷新读回。当前源码工作只消除组合阻碍并提供最小原生安装接口；模型尚未配置时保持执行关闭。

依据：用户 2026-10-15 在原负责会话明确批准“CI 和评审通过后按授权合并 #713，再补齐与现有 Chat 共存的接线；改库前最终清单确认；保留数据、密钥和 worker；不开新线程、不启用付费调用”。#713 的批准只覆盖该 PR；后继 PR 合并、真实改库/GRANT、生成私密角色凭据、配置变更、backend/worker 共同停启与付费调用仍分别待相应批准。

复用 [商品内容](product-content-localization-v1.md)、[Skill](skill-capability-v1.md)、[四权限](enterprise-agent-permission-modes-v1.md)、[Chat 会话运行](ai-workbench-conversation-runtime.md)、[原 Agent Runtime](product-agent-runtime-contract.md)、[Provider-neutral 文本](product-agent-text-provider-neutral-v1.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md)。本文只替代 Chat 设计中“conversationOnly 必须没有 enabled ProductAgent”的组合限制，在显式有界配置下允许内容标题执行独立共存；Chat 自身仍无 planner/executor。

产品/Figma Authority 沿 [最终 UI/IA](../product/final-ui-ia-authority.md) 和已批准 Skill 局部位置：文件 tg48P46SSXl6TBy9lZwg63、非归档 31:463，Chat 427:2467、我的智能体 4703:429、能力配置 4703:892、执行确认 4722:429、账户资料 432:534。沿原页面和操作，不新增页面/菜单或假启用状态；本文不声称今天重新验收 Figma 或运行 UI。

非目标：Chat 模型消息/规划/BusinessTask 执行、重新启用独立标题 Agent、其它业务 Agent、POD/素材能力、自动发布、模板/技能自动发布启用、企业自动准入、自动调预算、迁移/映射/兼容层、新 IAM/账本/队列/Run owner、通用安装或验收平台。原已运行的无关 worker 不因本范围被关闭。真实调用和用户验收仍 NOT_RUN。

## 当前根因和实际依赖

1. currentapplication/ai_workbench.go、httpapi/current_application.go 和 httpapi/ai_workbench_application.go 三处分别限制 conversationOnly 配置、依赖注入与 application 构造，后者要求 ProductAgent application 为 nil。这是此前会话管理阶段的显式边界；只删除 guard 会形成不一致，不允许。
2. ProductAgentConfig、runtime 开池和 buildProductAgentApplication 的父装配要求 AssetDB；随后内容分支并不消费 assets/readiness。这是当前多余连接前置，不等于四张 Asset 表都是内容启动必需。NewBoundedApprovedInventoryReader 构造仅验证连接非 nil 和大小上限，不在构造时查四表。不得为满足无消费者依赖默认创建素材表。
3. 原 Run/tool/credential 模型已有，只是保留库尚缺对应三表；ai_invocations 已有完整 53 列，不需要 ALTER/AutoMigrate。Review/Catalog/source 六表已有，需有界运行角色；准确真实清单由原安装 owner 单独绑定。

## 显式组合合同与接线

在可信 ProductAgent 部署配置中新增布尔 contentTitleOnly（默认 false）。它是现 runtime 的装配/能力范围，不是权限或另一套运行时。

- false 保持当前完整配置/装配合同，不自动选择此模式、不放宽原 full Chat 要求。
- true 且 Enabled 时只允许 ContentTextPolicies（至少一个结构合格策略；缺凭据仍不可执行）；TextPolicies/MaterialsTextPolicies/SelectionTextPolicies/DiagnosisTextPolicies 必须为空，不接受 AssetDatabase。RunDB、ProductDB 上独立 Review 连接、原商业 owner、组织 allowlist、准确内容 operation/prompt/schema/provider、单次上限仍按原合同验证。没有准确 route/计价/凭据时不能启用；不得注入假配置来启动。
- Disabled 的保留配置继续按原合同只提供已安装管理能力，不因本实现自动启用内容。其它已启用领域/worker 的现合同不变。
- contract→implementation→injection→consumer：可信配置→currentapplication 配置校验和开池→现 HTTP ProductAgentDependencies→有界内容 application→product.content.agent 原 Runtime、governed model、Skill、Review/Catalog。抽取现共享装配，只在实际消费者存在时创建其依赖；不构造 title application 再将资产能力置 nil。现完整模式继续注入其真实资产消费者。
- HTTP/composition 以服务端已验证 profile 与实际内容依赖接纳共存，不接受客户端 profile 或仅凭 policy map 非空推断。只安装内容执行消费者；其它 Agent 的直接启动/quote 不能因父系统仍注册而变可用。
- 本 profile 的新内容 quote/start 必须 fields=[title]，沿原 typed selection/语言/Skill 准确版本合同，在 receipt/Run/模型副作用前拒绝其它字段/Agent。已有事实和历史不重写、不映射；原 Review/Catalog owner 的既有历史读取/采用权限不被 profile 代替。resume/retry 仍核原冻结请求和当前权限，不能以新配置换请求、Skill 或编号。
- Chat 仍单独 ai_workbench_runtime 池（同逻辑库、不同身份与池），conversationOnly 继续只构造原会话 store，不拿内容的 model/Commercial/执行端口。Chat CRUD、当前身份/组织 allowlist 和 fresh live authorization 原样；send/confirm/start/resume 等原 Chat 执行入口仍在写 receipt/任务/调用前拒绝。Chat model readiness 保持 unavailable，不借内容 route 冒充 planner 可用。
- 商品内容四种模式复用原 agent_configuration/原 live authorizer：default/read_only/auto_approve/full 不由部署旗标授予或转换。full 仍仅已授权企业业务工具；发布/改价/下架/删除/增额度另授权。Skill 发布/启用与企业个人技能准入仍是独立现 owner 动作，不随安装发生。

## 原 owner 的窄安装/权限入口

只补原模型的有界 additive 入口（下节固定命令与 owner 函数），不调用当前 serving 初始化或整库 AutoMigrate，不复制模型 DTO 或建安装平台。

RunDB 阶段在原 product_agent_owner 下单数据库事务：先核目标库/owner、既有 ai_invocations 准确原 schema 和拟消费权限；仅在固定目标三表全部缺失时创建 product_agent_runs、tool_calls、ai_client_credentials 原模型及原索引/约束，授予现 product_agent_runtime 所需有界权限和独立 credential writer 权限。ai_invocations 只验证、不 ALTER/重建/回填；仅对明确已有原列授予实际所需的现权限。不动原 Chat/Skill/配置/知识/计量事实或其 grants。

ProductDB 阶段在原 owner/有界管理员下单数据库事务：验证既有 Review/Catalog/source 六表及对应 owner/约束，不创建/更新业务表；只给新的独立 Review runtime 角色授予实际 SELECT/INSERT 和精确可变列 UPDATE。credential writer 仍只沿 #713 原 CLI/原 owner 写准确组织/client，不取得 serving/其它库业务写权限。

两库不是一个原子事务，不伪称跨库回滚。实际安装角色、表/列/索引、各 GRANT、先后顺序及失败停止点必须由原安装 owner 以最终源码形成用户可确认的准确清单。新的登录角色和私密密码仅在实际授权阶段由原实例 owner 准备；源码命令不生成/记录真实 secret，不改旧角色密码/成员关系、owner 或 PUBLIC ACL。共享账号、原运行身份和原持久任务均保留。

已存在的共享表只验证当前合同，不修改；目标三表若存在，必须三表完整且各属同一批准原 owner、结构精确符合当前模型才允许 verify/no-op；目标组部分存在、外来 owner 或漂移均拒绝并停止，不能自动修表或迁移。权限前置不匹配同样 fail closed；不用宽管理员 serving 替代。事务失败回滚本库变更；响应丢失/提交未知先由原 owner 对照 native schema 和准确清单只读确认，不换键重装或自动补偿删除。失败后仍保持原程序配置，不为“回滚”删除新表/数据/角色/volumes。

## 幂等、状态、外部副作用与保留

无新领域事实 owner、状态机、业务事务或自动恢复协议。Run/Claim/checkpoint/tool call/预算/UNKNOWN 仍由原运行与调用账本负责；内容建议→Review→Catalog 仍用原 ProductDB UoW/CAS，RunDB 不参与伪原子事务。并发、重放、同键异载荷、当前撤权、精确 Skill ref/模式快照沿原合同。

正式运行仍使用 #713 准确 content 凭据、固定 product.content.decision 及原 beforeSend/商业计价/额度；本批不触发 provider 网络调用或修改额度。缺配置显示真实 unavailable/needs_configuration；不 fake success。

任何真实应用更新由原实例 owner 在清单批准后执行；backend 与四类 worker 同进程，不能承诺不重启。须保留原 manifest/私密配置/镜像/15 库业务事实/56 卷/任务身份，实际操作前刷新备份；新 StartedAt 后四 poller 需重新出现、任务身份不变才可报告恢复。源码 CI 不代表实际更新、用户验收或真实模型可用。

Legacy decision: N/A，没有消费已登记 RETIRE 的抽象。共享正确构造提取到当前 app owner；只裁剪新 profile 没有消费者的 title-parent Asset 前置和旧组合互斥，不退休原完整模式的合法能力。无 legacy wrapper、双读/双写、第二事实源、回填或 ID 映射。

## 必要验证、复核与交接

按实际增量 TDD：配置明确 opt-in/不借 full、组织/池/route/限额不放宽；contentTitleOnly+conversationOnly 共存，原 full 组合规则保持；factory 证明不打开/注入 Asset 或其它 Agent，身份和现商业消费者准确；直接 non-title/其它 Agent 先拒绝且无新 Run/receipt/model 调用；Chat CRUD 仍可用，模型/规划/任务执行仍拒绝且无新事实。复用现 Skill→内容→Review→Catalog 测试验证准确个人版本与 title delta/readback，不新增验收 runner。

使用现隔离 PostgreSQL 工具测三表原模型安装/有界角色、共享 ai_invocations 与原 Chat/Skill/Review/Catalog 的数据/schema/ACL 保留，错误 owner/部分目标/结构漂移拒绝和准确重复 no-op；仅与本增量有关的失败/权限行为。不接真实 IAM/provider/业务数据，不建通用故障矩阵。

既有 content_final_review 做一次本次新增高风险边界检查，达到 IMPLEMENTATION_READY 后由原唯一 Writer 在原 worktree 的后继主要分支/PR连续实现；最终完整路径只做一次集中独立增量检查和准确必要 CI。原 #713 未变化代码/证据不重复审。交出配置/私密输入格式、native 命令/权限清单、受控路径与 NOT_RUN 边界；真实安装/模型/页面与用户验收另行批准并记录，不关闭 Issue。
## 固定实现与注入映射（准入合同）

基线为 #713 merge `a0ab3b2d9f34c3f6e84560f8030c5099e5d781c0`。以下新增符号是本设计冻结的实现入口，不声称已存在或已通过运行验收。

| 合同 | 现路径 / 有界增量 | 注入与消费者 |
| --- | --- | --- |
| 可信 contentTitleOnly | `currentapplication.ProductAgentConfig` 与 `httpapi.ProductAgentDependencies` 增加同名布尔值；两层校验 true 的互斥 policy/Asset 条件 | `runtime.go` 只开 Run/Review 池 → `cmd/current-application/main.go` 原 `ApplicationFeatures` → `WithProductAgent`，不能由 HTTP body 覆盖 |
| 内容父依赖拆分 | `product_agent_application.go` 将 Run/ledger/points/configuration、canonical/source、Review、live authorizer 等共同构造抽到同一 app owner 的局部 helper | 完整分支维持真实 Asset/title 工具和原子域；true 分支直接 `buildContentProductApplication`，不先创建标题模型/runtime，不用假的 Asset 池过关 |
| 内容消费者装配 | `product_content_application.go` 原 `contentagent.Registration()` 只含 canonicalinspect/sourceevidenceinspect；保留原 governed executor/definition/Guard/Skill context | true 时 ProductAgent module 仅安装 content 路由和原 Review 路由；configuration/readiness 投影按准确 content 定义，原标题与其它 Agent 不接入可执行端口 |
| 有界内容请求 | 原 `registeredProductAgentRoutes` 与 `startTextRequestWithSkill` 的 shared start/quote 路径，在创建新 Run/快照/receipt/模型副作用前重验 title-only | 保留语言选择及准确 Skill ref；其它字段/Agent 直接拒绝；历史 receipt 不据此重写或重新执行 |
| Chat 共存 | `currentapplication/ai_workbench.go:29`、`httpapi/current_application.go:498`、`ai_workbench_application.go:90` 的三层组合判断同时消费可信 profile | 仅 conversationOnly+contentTitleOnly 可共存；ConversationOnly application 构造仍令自身 agent/plan/execution 为空，而非删 guard 后挂执行父对象；独立池/身份/组织约束保持 |

本文不依赖一个新 `System` 类型：当前实际链为 runtime/Features → command option → `NewCurrentApplication` → `buildProductAgentModule` → common/content application，全部保持当前 owner。

### 原生入口与 owner 分工

新增单一原命令式入口 `cmd/product-agent-runtime-prepare`，仅 `--target run` 或 `--target review` 两种固定目标。Run 使用原 schema-owner 私密 `PRODUCT_AGENT_SCHEMA_DSN`；Review 使用 `PRODUCT_REVIEW_SCHEMA_DSN`。命令不接受任意 SQL/表/权限，不做 schema 扫描服务，不生成 secret 或角色；固定角色预先由获准原实例 owner 准备。

- `internal/app/schema/productagentruntime.PrepareRun(ctx, db)`：固定本库事务编排；校验目标 public schema、`product_agent_owner` 与现 `product_agent_runtime`/`title_credential_writer`；固定三表分组预检、共享调用表只读核验、调用原 owner 的新建/验证钩子、只对清单对象执行 bounded grants。
- `...PrepareReview(ctx, db)`：另一个本库事务；校验 `acquisition_owner` 与预先准备的 `product_review_runtime`，只验证现六表并授予下列权限，不运行全量 Review/SRC/Catalog installer。
- `agentstore.EnsureRuntimeSchema(ctx, tx)`：当前 agent owner 新增局部入口，只复用 `agent/schema.sql` 的两张原表定义。外层已冻结固定三表分组；不得以 `IF NOT EXISTS` 作为结构验证，已有状态只读验证精确列/键/约束/索引/owner，部分不修。
- `openai.EnsureCredentialSchema(ctx, tx)`：当前 credential owner 的局部入口，复用唯一 `AIClientCredential` 模型（包括原唯一键/序列/索引），不复制 DTO 或调用广域 productlisting migration；只有确定缺失才创建。
- `aistore.VerifyModelPointInvocationSchema` 现函数保留；在原 owner 内补用于 prepare 的精确 public relation/owner/模型形状核验，不调用 `AutoMigrateInvocationLedger`。现 invocationRow 是53列权威；不得拿“有这53个列名”替代形状准入，也不补列/重建索引。
- 本次固定 schema/grant 编排可在 `app/schema/productagentruntime` 实现与测试，不给它业务事实或第二 schema 权威。owner 表定义仍分别来自现 Run SQL、credential 模型、invocation 模型、Review/Catalog/SRC 原模型/DDL；必要 owner 校验 helper 放原 owner。共享数据库执行使用既有 GORM/pgx 与现 native installer 风格，不引入通用 migration/权限平台。

两个准备函数只接受 PostgreSQL，使用原平台有界 owner 连接和静默 SQL logger；固定 public 限定，外层 timeout/事务必须覆盖资格核对与授予。核实原管理员有权 SET ROLE 后，在本目标事务中 SET LOCAL ROLE 为准确 product_agent_owner / acquisition_owner，并固定 search_path 为 public,pg_catalog；新对象从创建起属于批准原 owner，不事后改旧对象归属。原实例 owner 在批准窗口排除并行 DDL。未知结果先只读核实；每个目标输出 CREATED/ALREADY_READY 或明确失败及非敏感对象/角色清单，不输出 DSN/密码/模型 Key，不新增持久安装状态机。

角色约束：普通 serving/writer LOGIN 角色不得具备 superuser/createdb/createrole/replication/bypassrls、owner 成员关系或其它继承超额权限；既有角色/ACL/PUBLIC 不重置。新增普通角色的真实创建/密码文件不是此命令副作用。确认现 owner 具备管理自身表的资格，不将运行角色提升成 owner。只有新增目标与明确列出的权限发生变化；缺失 owner/角色、已有危险权限或 schema 不匹配均先失败，不自动 REVOKE/换库修补。

### 精确关系与权限清单

S=SELECT，I=INSERT，U=UPDATE。均另需本目标数据库 CONNECT、public schema USAGE；不授 CREATE/TEMP、DELETE/TRUNCATE/REFERENCES/TRIGGER、WITH GRANT OPTION、owner 或通配 ALL TABLES/SEQUENCES 权限。

| 数据库/角色 | public 关系 | 精确权限 |
| --- | --- | --- |
| Run / product_agent_runtime | product_agent_runs | S,I,U(revision,phase,payload) |
| Run / product_agent_runtime | product_agent_tool_calls | S,I |
| Run / product_agent_runtime | ai_client_credentials | S |
| Run / title_credential_writer | ai_client_credentials | S,I,U(api_key,base_url,model,api_style,timeout_second,enabled,updated_at)；仅原 id 的确切序列 USAGE，禁止序列通配 |
| Run / product_agent_runtime | ai_invocations（原53列，不ALTER） | S,I,U(point_price_version,input_points_per_million_tokens,output_points_per_million_tokens,maximum_prompt_tokens,maximum_completion_tokens,invocation_id,parent_invocation_id,agent_run_id,tenant_id,user_id,member_id,business_task_id,trace_id,capability,operation,route_mode,route_outcome,cache_status,provider_id,model_id,requested_routing_key,routing_key,credential_reference,policy_version,configuration_version,prompt_key,prompt_version,prompt_scope,prompt_hash,started_at,finished_at,latency_milliseconds,attempt,fallback_index,prompt_tokens,completion_tokens,total_tokens,image_count,estimated_cost_micros,estimated_cost_known,usage_known,currency,outcome,error_category,route_error_category,error_code,provider_request_id,upstream_job_id,input_hash,output_hash,review_score,review_needs_human,review_reasons)；原 recorder 的 `Select("*").Updates(&row)` 会更新全部原列，不能声称 outcome-only 权限足够，也不在本批改 recorder |
| Product / product_review_runtime | product_title_proposals | S,I,U(state,payload) |
| Product / product_review_runtime | product_title_operations | S,I |
| Product / product_review_runtime | product_snapshot_versions | S,I |
| Product / product_review_runtime | product_snapshot_heads | S,I,U(current_version,publication_id) |
| Product / product_review_runtime | product_source_publications | S |
| Product / product_review_runtime | product_source_publication_receipts | S |

credential updater 的列来自 `SaveCredential` 原 Assign/UpdatedAt，Review updater 来自 `review/repository.go` 的 state/payload，Catalog head 来自 `catalog/repository.go` 的 current_version/publication_id；INSERT 使用原字段/唯一约束，不增加业务写入。各当前 owner 的直接 SQL/GORM 更新列在实现自检时用原隔离 PostgreSQL 角色实际验证。

三目标全缺时才允许原定义创建；三目标全在且 public/schema/owner/结构/约束/索引与当前模型精确一致、角色有效权限恰好匹配时 ALREADY_READY，只读 no-op。存在一或二张、漂移/外来关系/owner、权限超额或原共享表不合格时拒绝。既有部分权限缺失的“完整三表”也不静默补救：报告准确差异，交原安装 owner 单独决定，避免把未知前次提交或第三方安装当成新空安装。实际先 Run 后 Product；第二库失败不回滚第一库已经提交事实，仍保持执行关闭，不删除第一库新表/数据。

最终真实清单要列出上述53列的展开名称、确切序列、原 schema/ACL 保留摘要、执行参数/镜像摘要、先后顺序和停启边界；由原实例 owner 在准确候选形成后交用户批准。本设计文档/fixture 不是实际改库授权。
