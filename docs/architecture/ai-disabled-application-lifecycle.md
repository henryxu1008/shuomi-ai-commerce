# 完整应用关闭新 AI 执行，保留配置与历史读取

Status: **IMPLEMENTATION_READY（仅离线实施）**。执行 Issue #1073，主要 PR #1075。

2026-11-04 原独立 Reviewer 已核对本设计 blob `e6035025672f0330fca0d1175756c0b039a3ae7d`
及 HEAD `ea778e472ea3a29c7cb083f8d5eeff2f341fe805` 的实际调用路径，结论无 BLOCKER、可实施。
该回执不授权配置发布、实例更新、真实调用、合并或产品验收。

## 产品结果与依据

2026-11-04 用户明确选择：“保留完整应用和历史查询，仅关闭新执行（补齐最小只读解耦设计后实施）”。
使用者是 Local Account Center 配置管理员及研究/选品人员。关闭执行后，仍能登录、管理统一模型配置、
查看原图片/图案任务、按原请求键核实准备结果、读取库存/批准回执和原费用事实；不能创建新的图片执行。

本增量采用 **Independent Architecture**：从当前图片 Service 抽出只读端口并改变关闭态装配，
在正式代码前进行一次有界检查。它不重开研究、模型、账本或 Flare 的全局设计。
产品依据为本次用户决定、[全新系统基线](../product/greenfield-no-legacy-migration.md)、
[统一模型配置](enterprise-shared-model-configuration.md)、[研究/选品](market-research-selection.md)、
[图片业务](product-image-agent.md)和[原效果恢复](image-agent-effect-recovery.md)。
沿原图片文档中的 Figma 1998:545/Console 交互权威；不新增页面、菜单、业务流程或视觉设计。
现有能力字段如实投影“执行关闭”，不改变原历史状态或把未就绪显示为可执行。

Scope：原 currentapplication 管理/执行生命周期、完整图片 HTTP 装配及已有历史读取行为的抽取。
Out of Scope：单图路径扩展、配置安装/发布、凭据修改、旧兼容/迁移、Flare/provider/报价/计费重建、
新 schema/状态/权限/持久化 gate、自动恢复/结算、真实平台/AI/付费调用、实例更新、合并、用户验收。
预算仍每项 ¥0.30/20 点，合计 ¥0.60/40 点；原两调用及更小限制不变。

## Owner、唯一 Writer 与依赖

#1078 原 owner 已明确交接 currentapplication、必要 cmd 测试和
`image_set_application.go` / `current_application.go` / `image_set_module.go` 的本次装配修改；
#1075 原会话是唯一 Writer。必要的 artwork 聚合读取与能力投影属于同一装配增量，不改其 provider/worker。
对方已结束 Flare 生产修改，不并发写这些文件；稳定候选 d9a5c7bd3351b66a23fc66259e459d91333e86ad
尚未合并，本次不自动合入它。原稳定共享接口 f35 继续可用。
未来组合须保留 d9 从原冻结 quote 派生 referencePriceVersion 的只读语义。

事实 owner 完全不变：Run/ProductAgent 的配置 Repository/CAS/current、Image 的 Run/Plan/Slot/GenerationFact、
Asset 的库存/批准回执、Product 的来源/图案采用回执、原 Invocation/资源账本各自唯一。
管理 writer 仍同 Run DB 的 `model_route_config_writer`，独立窄池 ≤2。
没有新 current、历史副本、数据库、缓存事实、旧字段解释或第二恢复责任方。

## 装配合同

| 入口/条件 | 实现与注入 | 消费者与效果 |
| --- | --- | --- |
| ProductAgent 配置存在、Enabled=false | 沿原 Run DB 生命周期开 reader；统一管理专用 writer/manager 独立于执行启用 | 原 Options/Current/CAS；不装配文字 executor/Review/Asset 执行池 |
| 关闭态声明文字 policies | 复用原 policy 元数据校验：租户 allowlist、profile/用途、字节/Token/时间/成本/两调用预算 | 定义合法可管理不等于准许执行；候选不得脱离原 anchor/更宽参数 |
| 无文字 anchors、执行关闭 | 允许合法管理目录（包括明确 image bindings）；仍校验 writer/定义/图片范围/原预算 | 不虚构文字 consumer，不开放无来源的显式文字候选 |
| 完整图片：WorkerConfigFile非空、Generation.Enabled=false | 保留 API Image、Asset、Product、配置的当前池及权限验证；显式传递完整模式身份 | 原完整模块仍注册，不降级单图；不读 worker 私密配置/开 workerDB/dial Temporal/创建或启动 worker |
| 完整图片执行启用 | 维持原完整配置、workerDB、Temporal、授权、来源、冻结预算与所有资格校验 | 原完整执行链；缺一个必要依赖即拒绝启动，不 fallback |
| 缺 image credential metadata | 沿 SharedCatalog 原 nil reader → CREDENTIAL_READER_MISSING | 未就绪；不远程试连、不借用另一用途凭据、不放宽完整选择快照 CAS |

现 Generation 的单位/预算元数据与 Enabled 分开校验。显式关闭且提供单位/上限时仍须合法；
零声明只能表示没有执行资格，不制造正数预算。原 DB 角色/隔离、allowlist、路径/URL 等静态校验保留。
不改 Tenant Admission 来伪装关闭，不新增另一个开关。无 worker 文件的现单图路径保持原合同。

完整图片执行关闭时，企业容量仍验证原 commercial owner 的容量 schema，并为其他已启用的存储写入
消费者执行原 namespace 准入与 meter 接线；不能随图片执行一起关闭容量保护。
图片 worker 的计量启用及相同 commercial owner 检查，仅适用于实际装配图片写入执行器的路径。
关闭态不读取 worker 私密配置来满足该检查；完整执行开启及原单图路径的 worker 准入不变。

## 原历史读取抽取与 HTTP 边界

在当前 `internal/imageagent` owner 内抽出 RunReader：只有 Get、GetPreparedImageSet、
GetPreparedArtwork、GetPreparedArtworkByKey 四种已有只读行为。复用原确定性 run ID、
verified identity、organization/member/actor/run 检查；Service 的已有读取委托同一实现。
读取器只依赖原 Repository，不持有 workflow/provider/executor，不暴露变更方法；不是旧 Service 兼容包装。

完整装配的共用读取部分（来源授权/配置/Run/库存/批准/GenerationFact/URL 投影）先成立；
仅执行分支再构造 quote、素材执行 reader、publisher、Service 和 worker dependencies。
图案采用回执仍从原 Product/Collection 读取；可复用原 ArtworkReceiver 的只读回执方法，
关闭分支不调用其发布/变更方法。当前授权/来源/配置不可读时沿原错误，不返回空的伪历史。

原 GET sources、runs/recent、by-key、run、inventory、approval，以及 artwork by-key/run 保留原身份、
live permission、来源/原冻结配置校验及原状态/UNKNOWN/费用投影。历史不刷新模型、不重新报价、不重解释原键。
本地 requirements 查询可继续使用原纯读取。依赖执行报价的 model-options 在关闭态明确不可用。

关闭分支不构造可执行 Service；HTTP 在原认证与权限边界内明确拒绝 prepare/confirm/regenerate/restart、
preview/approve、cancel/recover/resume 和 artwork prepare/confirm/save/recover/resume/cancel 的执行命令。
原请求键结果改由原 GET 核实，不能靠 POST 重放启动 workflow。新预留/Invocation/Run/Plan/执行快照均不产生。
图片响应中的 generationAllowed/approval/regeneration/candidateSelection/save 可用性置 false，
保留 generationAdmitted 等历史事实。图片/图案 agent 能力明确显示执行关闭，避免 nil quote 或虚假 AVAILABLE。

POD 的原 provenance/来源批准 guard 仍装配并校验历史身份；关闭时不为它加载生成 worker 的秘密。
如新操作需要生成对象 bytes 而没有合法 byte reader，沿原不可用结果，不能跳过 guard、冒充可读或改用任意 URL。
现有历史 GET 与 durable URL/回执投影不依赖这条执行素材下载路径。

## 不变量、失败和恢复

无新事务/状态机：此为进程装配选择，不写业务事实。启动缺历史 schema/grant/owner 时明确失败，
开过的池沿原 defer 关闭。关闭态重启同样只读；撤权/跨企业/成员变化沿原 live 权限和原读 owner 拒绝。
HTTP body/timeout/大小限制和 DB 窄池保持；不新增轮询/远程探测或自动重试。

历史读取不改变已发送事实、不把 UNKNOWN 视作未发送、不释放 R、不自动结算，不发送恢复/取消命令。
已在外部存在的 worker/workflow 不被本地开关推定已停止；本轮不控制其他进程。
恢复仍由原 owner 在其原明确授权下执行，不新增调度器或补偿器。
本次 Threat Model 沿现租户/成员、撤权、伪造原键、漂移和外部不明结果；不追加理论 hardening/新 Accepted Risk。

## 最小验证与交接

TDD 先验证关闭配置/生命周期缺陷，再实现：管理合法性/anchor/预算负例；关闭完整模式仍带原 API/Asset 池和
完整模式标志，worker配置打开器/workerDB/Temporal/Start 均未被调用；执行启用缺依赖仍失败。
该生命周期回归同时覆盖 `storageCapacity=true`，并验证关闭图片执行仍拒绝缺失容量 schema、
取消的启动及其他存储来源身份漂移；容量关闭的既有事实守卫及执行开启的 worker/owner 负例保留。
RunReader 原键与跨组织/actor/member负例、无执行调用；原图片/图案 GET 历史/回执与响应可用性保持，
命令在关闭时拒绝且无新事实；原 active 图片测试及研究→选品合成 HTTP 回归。
复用任务专属 PG/本机 fixture/httptest，不建设 runner/故障矩阵平台。缺 DSN 的 SKIP 不报 PASS。

独立检查只评本增量的实际 diff/调用路径，未变的公共账本/原业务证据按原绑定复用。
实现只进入 In Review；CI/自检/AI review 不等于产品验收。真实配置发布、保留实例更新和真实调用仍 NOT_RUN。
