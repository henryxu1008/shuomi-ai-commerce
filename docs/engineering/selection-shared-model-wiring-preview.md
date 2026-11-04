# Local Account Center：统一模型配置与 AI 关闭边界

本轮**离线实现完成，待最终增量检查；NOT_PUBLISHABLE**。用户已明确保留完整应用与历史查询，仅关闭新执行。
[最小生命周期设计](../architecture/ai-disabled-application-lifecycle.md)在正式代码前通过原独立 Reviewer，IMPLEMENTATION_READY；
本次因抽取原图片读取端口与改变关闭态装配采用 Independent Architecture，不把此前 owner 方向确认当准入。
#1075 原会话为唯一 Writer，#1078 原 owner 已确认文件交接且不并发修改。本轮不自动合入其未合并的 Flare 候选。

关闭态保留原图片/图案模块、来源权限、Run/原键/库存/批准/采用回执读取；原 Service 与无执行依赖的 RunReader 共用原读取实现。
执行关闭时不读图片 worker 秘密配置、不打开 workerDB、不连接图片 Temporal、不装配或启动执行 Service/worker；
原历史与 UNKNOWN 不被清除、重算或重发。完整模式身份显式传递，不因缺 workerConfig 而退到单图。
统一模型管理独立于文字执行启用，仍沿原 Run Repository/CAS/current、专用 writer、合法定义/锚点及完整选择合同。
缺 image credential metadata 沿原未就绪结果，不远程试连或放宽资格。

本轮目标缺陷先 RED 再 GREEN；本地 HTTP 相关回归68 PASS、owner/生命周期73 PASS，均0 SKIP/FAIL。
其中原市场调研→智能选品合成 HTTP 使用原共享配置/参考价/模型适配/Invocation/账本；不等于真实来源或远端调用验收。
源码候选、准确CI与原独立Reviewer最终增量回执由PR #1075维护，不在本文伪造新SHA的执行结果。

原 a66 实例保持可用，不做旧兼容/迁移、不改凭据、不发布配置、不更新实例、不真实采集/AI/付费、不合并。
预算仍每项¥0.30/20点、合计¥0.60/40点和原两调用/更小限制。旧价格精度研究已撤回，不恢复为交付前置。

## 现有实例、固定候选与全新系统目标

| 配置或行为 | 原 a66 实例的已核现状 | 297 固定候选 | 建议的最小全新系统目标 |
| --- | --- | --- | --- |
| 模型配置 owner | manifest 为 `imageWriter`，4 条 image binding，没有统一 `writer` | 仅接受 `writer` / `bindings`，严格拒绝 `imageWriter` | 一个 Run/ProductAgent 配置 owner、一个 writer/current；提供新的合法 manifest，不承接原 image current |
| 原配置库连接 | `productAgent.database` 存在，文字执行关闭 | 即使 `Enabled=false`，原 runtime 已能打开 agentDB | 复用这条连接生命周期，不新建配置服务/数据库 owner |
| 统一配置管理加载 | 原图片配置装配 | `modelRouteDefinitions` 要求 `ProductAgent.Enabled=true` 及文字 anchors | 管理加载可独立成立；仍校验 schema、租户、定义身份、用途、原 anchor/参数及预算，不以删除检查取得资格 |
| reader / writer | 当前真实权限本轮未查询 | 原 reader + `model_route_config_writer`，writer 与 Run DB 同目标、pool ≤2 | 沿当前安装器准备合法角色/表/视图/窄 grant；本轮不安装、不创建角色或复制密码，现有资源是否满足仍 UNKNOWN |
| 三类选择与价格 | 本轮不读取或改写实际 current/价格 | 原 SharedModelService / Options / Current / CAS | 继续同一完整选择快照；平台参考价/FX 沿原 owner。管理可用、配置已保存、技术资格和允许执行分别表达 |
| 图片资格读取 | worker 配置文件引用存在；未读取其凭据或队列 | `prepareSharedModelServices` 允许 `imageCredentials=nil` | 复用现有 `CREDENTIAL_READER_MISSING` / 未就绪结果；不为管理页强启 worker、读取真实凭据或新增 reader 角色 |
| 图片生成声明 | generation 只有 `priceVersion` / `pointsPerImage`，没有 `enabled` 键 | 当前 generation 合同为 enabled / unitVersion / maximumMicropoints | 全新 manifest 使用当前字段和原预算，不解释旧字段或原数据；缺字段不冒充已关闭/已准入 |
| 图片执行生命周期 | worker 引用不能证明实际执行关闭 | 有 WorkerConfigFile 会加载完整 worker 配置、连接 Temporal 并启动 worker；生成配置还要求 Configured | 关闭态保留完整模块及原历史读取，不加载/启动对应执行 worker、不做远程检测；启用才满足原完整依赖与门禁 |
| 凭据归属 | 本轮未读取密钥 | 保留各原 credential owner 的去密投影与执行句柄 | 只引用合法的当前身份，不把图片 Key 借给文字、不复制 Key、不默认使用另一用途的绑定 |

源代码依据：`internal/app/runtime/currentapplication/product_agent.go`、`model_route_pricing.go`、`runtime.go`，`internal/integration/aicapability/routeconfig/shared_catalog.go` 与 `internal/core/config/type_imageagent.go`。owner 已确认原管理服务可复用，尚缺的是此处关闭生命周期与资格规则的有界细化。

上述目标不能直接套入现有 a66 manifest。原实例保持原样；未来全新安装的数据库、当前表/视图、单位及角色/grant 需要独立授权后验证，不能把缺项通过迁移、兼容、双写或第二 current 补齐，也不从本方案推导删除原数据的许可。

## “AI 关闭”的建议边界

| 操作 | 建议关闭态的行为 | 本轮实际授权 |
| --- | --- | --- |
| 正常启动、登录、读取页面、手工研究/比较/决定 | 保留原业务权限与事实读取，不调用模型 | 只读核对 a66 运行状态，不操作具体业务记录 |
| 查看三类模型选项、当前选择、参考价/本地资格 | 复用原管理服务；缺凭据元数据等依赖时明确未就绪，不远程试连 | 仅整理源码/去密配置差异，不读取或发布真实 current |
| 保存模型选择或发布价格 | 未来可由原授权角色经原 CAS/回执完成；不会自动开放执行 | **本轮禁止保存/发布**，可管理是目标能力，不是此次写入许可 |
| 纯本地、无副作用的现有校验/费用展示 | 可复用；不新增报价 API。会创建执行快照、Invocation 或预留的业务准备入口遵守原关闭门禁 | 不实际 prepare/start，不产生真实预留或执行事实 |
| 文字、图片生成、多模态理解 | 对应关闭时拒绝新执行；Agent 已启用、模型可选、技术 Ready、页面“只读建议”均不能替代调用授权 | 不执行任何真实 AI，研究/选品预算不变 |
| worker、真实模型检测、轮询、重新生成、重试发送 | 关闭时不产生新的 provider 请求，不通过其他入口绕过；启动管理服务不启动这些执行路径 | 不启停 a66 worker，不连接真实 provider 或重发任务 |
| 原历史、回执、UNKNOWN | 可按原权限读已有事实；关闭不证明“未发送”，不自动清除预留。已有本地终态处理仍由原 owner/既有授权负责 | 不改变余额、旧调用、预留或恢复事实，不运行恢复/结算动作 |
| 数据采集 | 沿 Data Services 原独立授权；不能由 AI 关闭推导采集已停或获准 | 本轮不采集、不探测真实平台 |

`ProductAgent.Enabled=false` 只说明该文字执行入口关闭，不能代表全系统 AI 关闭。图片生成还有原 Generation/tenant admission/worker，业务操作还有当前授权、启用状态、模式、来源和预算门禁；未来应逐项明确关闭状态，沿原入口拒绝执行，不新增一套全局持久化 gate。

原 a66 的 generation 没有当前 `enabled` 字段，本轮没有检查 worker 秘密配置、任务队列或真实 provider 日志，因此图片是否完全停止仍 **UNKNOWN**。保持实例可用不等于将这项 UNKNOWN 改成关闭，也不授权修改现有开关。

## 分工、验证与实际开放停止点

- #1075 原 Writer 已按获准设计在原 owner 内实现生命周期与只读装配；#1078 原 owner 的 Flare/provider/quote/worker/fact/ledger 实现保持独立，不新增公共 owner。
- 关闭态角色/锚点/预算负例、非空历史列表、原键和跨身份拒绝、原批准回执的启用/关闭两分支，以及所有图片/图案执行命令拒绝均纳入原测试。POD provenance guard 保留，缺合法生成 bytes reader的新操作明确不可用。
- 真实安装/配置发布仍需独立授权和资源核对；现阶段不提供可直接运行的真实 manifest/DSN/grant 脚本，不复制凭据、不迁移旧 current、不替换a66。
- 开发自检、CI和独立工程检查不等于产品验收；现场运行/真实调用/用户试用仍NOT_RUN。

研究与选品预算继续每项 ¥0.30 /20 点，两项合计 ¥0.60 /40 点；每项最多两调用及原更小限制保持。下方参考文字接线保留原预算；本轮生命周期实现不自动发布其中任何配置。

## 前序参考文字接线候选

当前依据：[公共 owner 确认的参考策略与分工](../architecture/enterprise-shared-model-configuration.md#本批文字流程的接口与分工)（#1078 `5cc8248e5d659e3004c2c6686c582528c8fdd0d4`）、[本批接线合同](../architecture/bailian-deepseek-shared-text-admission.md)。原共享选择、CAS/current、Invocation 与资源账本保持唯一；不重复建设公共模型/计费能力。

## 当前 → 候选差异

| 配置项 | 当前可审查信息 | 本批候选 |
| --- | --- | --- |
| 使用企业 | Local Account Center；本轮未重新读取运行数据 | 仍是该企业；实际写入前由原授权 owner fresh 核组织/成员/权限 |
| 文字绑定 | 2026-11-03 已有非秘密快照记录 `local-content-route-830-20261020`，型号 deepseek-v4.1-flash，协议 openai-compatible | 只引用现有组织级绑定，不创建、复制或改写 Key；当次凭据版本/端点摘要需 fresh 检查，不从旧快照推定当前有效 |
| 型号与策略 | 百炼 DASHSCOPE / deepseek-v4.1-flash | 固定非思考 v1，保留原 byte/deadline；不自动换 1017、不提高调用上限 |
| 企业选择 | 同一共享配置位置维护三类选择 | text 选择该绑定；image/multimodal 保留明确选择和各自未就绪提示。整个快照一次 CAS 保存，文字使用不等生图完成 |
| 平台价目 | 原平台参考价/FX owner | 新参考版本，策略 reference-text-totals-v1；CNY 恒等 FX；输入 ¥2/百万 Token、输出 ¥8/百万 Token，不应用缓存优惠。是草案参考值，不是实时准确厂商价 |
| 研究操作 | product.research.agent / product.research.decision | 项目资料分析，保留 market-research-prompt-v1 / market-research-action-v1、原项目/成员权限及只读设置 |
| 选品操作 | product.selection.agent / product.selection.decision | 独立商品对比，保留 product-selection-prompt-v1 / product-selection-action-v1、准确保存商品及独立历史 |
| 操作预算 | 用户已确认每项 ¥0.30 /20点，两项 ¥0.60 /40点 | 数值不变；每项两调用，每次 ≤32000/4000 Token，原更小限制优先。原文本定义每调用 maximumMicropoints=10000000，参考改价后仍核最大 R，不把 10 点当实际收费或强制预留。配置预算控制不等于供应商实际账单硬限 |
| 用量与结算 | 原 Invocation/资源微点账本 | 显式基础 P/C/T 校验后按冻结参考价估算；费用投影带 referencePriceVersion。缺缓存细分不阻断；显式完整的 0/0/0 按原 A=0 结算/恢复，缺基础事实/矛盾/超限仍 UNKNOWN，保留 R 且不重发 |
| BoundEvidenceVersion | 不再索取供应商完整硬界证明 | 不填假值；参考策略资格来自有效策略、配置和准确 profile 身份。其他策略不自动放宽 |

候选的准确 binding reference/price revision 由原 owner 合法发布时生成，本文件不伪造 ID、CAS 或真实 Ready。旧内容用途的 revision/凭据版本不能替代研究与选品各自的定义和新共享选择版本。

## 用户流程与离线验证

入口仍复用智能市场 → 我的智能体 → 本企业启用/配置 → 项目资料分析或独立商品对比。项目业务链为市场调研中选择同一项目和 2–5 项已保存资料 → 智能选品查看比较/费用估算 → 明确确认只读分析 → 人工决定与理由 → 按批次到我的数据核对 → 转入我的供应链。

原 HTTP fixture 已使用 SharedModelService、PlatformPricesService、SharedCatalog、原配置 Repository、受控模型 Adapter、Invocation 与资源账本；只在任务专属 PG 中保存合成配置，provider 仅本机 httptest。研究上游项目/来源为固定合成适配。回归核对完整业务交接、原键重放、没有自动转入、没有额外模型调用/重复结算；界面核对只携带已批准的准确资料，分批分别确认。不能据此宣称真实 SHEIN/Amazon 取数、远端模型或运行实例已经可用。

## 仍需分开处理的实际开放事项

本批离线实现不再被精确价格/峰闲时点/全缓存分项证明阻塞。实际开放仍需另行授权配置发布与指定实例更新，并核对当前新系统安装/运行合同和该企业实时非秘密绑定信息；不升级旧库、不做迁移/兼容、不删除旧数据，不擅自初始化保留卷。真实采集、真实 AI/付费及用户试用均未执行。

本轮准确源码、开发自检、原独立 Reviewer 增量结果和必需 CI 在 PR #1075 维护；实现完成只进入待独立验证/用户试用，不签发产品验收、上线或合并许可。
