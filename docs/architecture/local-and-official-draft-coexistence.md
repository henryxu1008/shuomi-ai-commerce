# #1158 同一实例内测试草稿与正式刊登共存

Design Basis: **Independent Architecture**。Admission: **ARCHITECTURE_REVIEW_PENDING**。
本轮仅设计与独立 Architecture Review；不得修改正式生产路径、原 35444 配置或数据，不调用模型、不保存/发布、不释放原预留。实现、合并、实例开通及真实平台操作分别等待后续明确授权。

调查基线：`main 34dcf084a4c6a92516cfdbba2c80cb2f6f90712e`。执行 [#1158](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1158)，唯一 Writer `/root`，分支 `codex/issue1158-draft-coexistence-design`。

## 1. 产品决定、结果与范围

2026-11-10 用户决定：同一实例可以保留测试草稿并处理正式刊登；保留现有商品、固定 R1 与黑 M、黑 L、白 M 三条 SKU，测试稿始终禁止发布，正式稿按真实店铺和官方规则校验。本决定替代 [受控本地接线](controlled-local-platform-draft.md) 中实例级 `controlledLocalDraft` / `SupplyChain` 互斥条款；该文其余固定试点范围、持续 provenance、禁止升级及原数据保护合同保持有效。

使用者：从自有商品/R1 准备店铺刊登资料的运营人员。当前阻碍：配置拒绝共存，两种装配同时占用同名模块/相同 HTTP 路径；当前 UI 又用全局开关选一套模式。只删除配置互斥不能交付可用路径。

用户结果：在同一商品/R1 的现有刊登页面选择“本地测试草稿”或“正式平台草稿”；两类记录可分别重开，测试稿不能进入正式执行。选择正式店铺后复用原正式 Target 编辑、R1 带入及销售属性映射，不要求重新录入原商品。

**Must**：

- Catalog/Source/Completion R1/Asset、现存本地 Target 的 ID、记录、hash、原命令回执不变；一份正式 Target 仍只有黑 M、黑 L、白 M，不拆色、不造白 L、不删黑 L。
- `localTrial` 是原记录的不可剥离来源，不是 UI 开关或授权凭证；所有正式执行仍先拒绝测试记录，含直接 native 调用。
- 正式记录重新使用真实 Store/官方应用/商户/规则/图片准入；测试类目、颜色尺码枚举、仓库 ID、hash 或批准不能成为正式证据。
- 既有 org/actor/member、live 权限、单 Product DB 事务、CAS、同键冲突及 UNKNOWN 原键核对不变。
- 正式配置缺失时，本地功能可独立工作；本地停写时，同实例正式功能及获授权测试历史只读均可继续。

Should：复用现有编辑器/带入页/严格 BFF/保存 hook/组件，页面可达、可返回、状态清晰；Figma 只作业务流程材料，不作 UI/IA 权威。

Out of Scope：一键复制/升级测试 Target、通用环境/规则/Store 注册平台、扩大固定本地试点到任意商品/租户、测试稿任意字段编辑、新 IAM/表/状态机/事件或恢复系统、模型调用/素材生成、官方应用登记或授权、官方 API 调用、原实例部署、实际保存/发布/库存补值/换汇、删除/回灌记录、原预留恢复。普通正式流程已有的其它平台行为不扩展。无新增 Accepted Risk。

Threat Model：防止不可信 URL/表单/恢复 intent 选错分支、伪造/去掉标记、跨企业读取、测试规则/图/仓库冒充正式证据，以及全局开关/停写导致正式路径错误副作用或历史丢失。恶意数据库 owner 重写全部事实/hash、绕过宿主机边界及通用旁路优化不在本轮范围；现行外部副作用与权限底线保持。

## 2. 已有事实与架构问题

- #1235 已合并并更新原 35444；同一原商品 V1、Completion record `f7853e11-353a-5214-a7c7-75a8ded6e278`/R1；本地 Store `dc1f9c74-26ed-4ab0-93b3-e8e12f01d32f`，Target `21b58c9b-712c-586b-9590-ea26f29d716f` 当前 R3。仅为已有交付证据，本轮不重新访问/操作实例。
- `runtime/currentapplication/controlled_local_draft.go` 和 `httpapi/current_application.go` 拒绝与正式 SupplyChain 共存；`listingModule.Name()` 都是 `supply-chain`，`ControlledLocalDraftRoutes` 从正式路由中截取七条相同路径。
- `Application.Targets`、`Rules` 目前各只有一个具体依赖；全局替换成多用途隐式 fallback 会增加误调用风险。选择保留两个有明确用途的 app 装配，仍使用同一 Target/Repository owner。
- 已有 `localTrial` 跨 Input/Record/hash，仓储锁 head 后 `SameLocalTrial` 拒绝切换；`PublicationReady`、native builder authority 和上传/计划/素材等消费者有前置拒绝。本轮保留这些防线。
- 正式 `buildListingModule`、Target 编辑器、R1 带入、销售映射和发布/结果核对协议已存在；本轮补共存接线，不另造发布器。

## 3. 最小选择：同一数据 owner，明确分开的接口用途

保留正式 `listingModule` / `supplyRuntime`。本地装配改为独立的窄 kernel module `controlled-local-drafts`，只更换模块身份、路径声明与消费接线，不成为新的领域事实 owner。

正式 base 保持 `F=/api/v1/workbench/supply-preparations`。本地使用不与其动态路径冲突的同级 base `L=/api/v1/workbench/local-platform-drafts`。BFF 同步明确这两个有限路径集合，不注册旧本地地址别名或失败后换分支。

| 本地接口（相对 L） | 用途/权限 |
| --- | --- |
| `GET /context?sourceId=…&completionRecordId=…&revision=1` | 新增的窄只读上下文；当前 live scope + Source/准确 R1 读取授权后，返回该调用者获准的固定 Source/R1/测试 Store 引用、`writesEnabled`；不泄露服务端凭据，不查询官方 API |
| `GET /sources/:source_id/targets/:store_id` | 读取本地 head；沿原 supply.read、Source 与 Store scoped 访问及固定 provenance 核对 |
| `GET /records/:record_id` | 原不可变本地版本 |
| `GET /target-commands/:key` | 原本地操作回执，无 writer/rule/provider 依赖 |
| `POST /target-rules` | 编译 fixture + 原完整 bootstrap；不是官方规则入口 |
| `POST /shein-completion-preview` | 原固定 R1 完整派生 |
| `POST /shein-sales-mapping-preview` | 原三行销售属性完整派生 |
| `POST /targets` | 原 Target 单事务/CAS/命令保存 |

启用本地写入时八条；停写时仅四条 GET。后三类 GET 保留原 scope/record/hash/标记和 Source/Store 可读检查，不依赖 Store 仍具备官方连接。context 中的 `writesEnabled` 仅表示服务端本地准入；新写仍实时核对固定 Scope/Source/R1、Store native 条件和完整输入。对不属于固定试点的调用者拒绝，不通过店铺名称/未激活状态推测“测试店铺”。

正式 `F` 原路径、权限、超时及 worker 装配保持；给其 Target head/record/command 读取注入**仅拒绝 localTrial 的用途 guard**，不额外收紧无标记正式历史的既有读取合同。错误用途返回明确 4xx/不可用状态，无路由重定向或改键重发。

两模块可各实例化原 app，但不注册重复 Method/Path/Module name，不重复装配 SourceMaterials/Completion routes，不给本地 app 注入 uploader、publication、Agent、workflow 或 execution authority。正式 `supplyRuntime` 指针始终指向正式 app，不能因为本地 app 存在而把它当作正式 full pointer。

### Contract → implementation → injection → consumer

| 事实/合同 owner | 现有实现及计划改动 | 注入/消费者闭环 |
| --- | --- | --- |
| 配置准入 | `runtime/currentapplication/{controlled_local_draft,listing_config,runtime}.go`；仅移除两者互斥，保留各自完整前置 | `cmd/current-application` bridge → `WithSupplyChain`/`WithControlledLocalDraft`；同一批准 Asset reader/pool重用、关闭责任不重复 |
| HTTP 装配 | `app/httpapi/{listing_application,controlled_local_draft,current_application}.go`；独立本地模块和路径白名单 | kernel module/route descriptor、允许模块列表与 BFF 全部验证用途；`supplyRuntime` 只给正式消费者 |
| 商品/R1/图片 | 原 Catalog/Source/Completion/Asset owner | 两 app 消费同一现有只读端口，既有批准路径不变；无资料副本表 |
| 本地身份/派生 | `listing/record/target/local_trial.go`、原 completion/sales preview | `NewLocalTrialTargetService` 仍固定 policy+compiled rules；不注入正式 transport |
| 正式身份/规则 | 原 `NewTargetService`、`RuleReader`、`OfficialTargetMerchantStore` | 正式模块消费真实 Store/官方 registry、effective product/当前规则；拒绝 localTrial，不使用本地 reader |
| 保存/原键 | `integration/persistence/listing/record` 原 head/records/commands | 两模块共享同一 owner/事务规则；相同键不会因 URL 不同变成两个事务事实 |
| UI | 原 listing missing/R1、Target editor、strict contracts/proxy、保存 intent | 两种明确用途 → 对应有限 BFF/API → scoped backend；记录标记和服务端 context 决定可见能力，前端参数不授权 |
| 发布/执行 | 原 `PublicationReady`、native goods、upload/plan/material/image/worker 消费者 | 一律按持久化 record 准入后才访问正式依赖，原操作/receipt/UNKNOWN owner 保持 |

## 4. 记录身份与用户路径

不增加或修改 Target identity 算法、表、purpose/family。测试与正式都沿原 `TargetIdentity(org,actor,source,store)`；现有测试 Store 与真实店铺分开，因 Store 不同而有不同 Target。一个真实店铺的一商品仍是一份普通 Target，两个颜色组、三行 SKU。

现存本地 R3 永远是测试稿，不能通过删除 localTrial、改 StoreID、改 rule authority 或另一路径保存来变成正式记录。原已提交版本、body/input hash、原键响应和规则版本不重算/回写。

最小界面路径：

1. 同一原商品/R1 页面分别展示“本地测试草稿”和“正式平台草稿”。测试页显式标注“本地测试 · 不可发布”；正式配置未就绪时给真实缺项说明，不造可用入口/规则。
2. 测试分支沿当前固定试点的创建/带入/映射保存三步；本地 context 精确指出本次测试店铺，避免普通店铺被名称误分类。保留原页面及返回商品/R1上下文。
3. 正式分支选择不同的真实店铺，按现有正式 adapt/Target 编辑流程从**原 Source/Catalog 和准确 R1**准备正式 Target，再显式选择官方类目/属性/枚举/仓库，沿原 preview → 确认 → CAS 保存。当前测试 Store 不提供为此步骤的正式目标；在本地 policy 存在期间，正式 app 的真实规则/商户读取边界对该固定测试 Store 明确拒绝，覆盖 UI 直 POST、adapt/plan/material 与正式执行 sibling paths，不仅隐藏下拉项。它不是新的 Store 状态/持久化用途标签。
4. 不读取测试 Target.Input 作为正式源，不复制测试类目 123、颜色/尺码 ID、`trial-us` 仓库、图片批准、rulesHash 或 Ready 状态；原英文文字、三行编码及 0/空值由同一 R1 的既有带入合同负责。正式映射由当前官方值重新确认。
5. 缺库存仍是空，黑 M 的 0 保持；正式用户后续可按现有编辑器明确填写真实售价、包装、库存、图片，不回写原商品/R1。CNY 不自动换成 USD。规格集合仍 `variant_set_unconfirmed` 时阻断发布，不擅自补/删 SKU 或拆稿。

共享列表/阶段投影沿原 record 的标记展示用途；测试保存不能显示成正式审核、提交或上架成功，也不计入这些正式结果。不新建统计 owner、汇总表或历史回填。

这不是“一键升级测试稿”或新 transfer 协议：新正式 Target 的创建/原键重放由原正式操作负责，R1 的固定选择与后续确认由已存在的带入/映射端口负责。两者独立保存，没有“测试稿已升级”持久化状态，也不在两个 Target 之间做跨记录事务/补偿/同步。

## 5. 安全、事务与副作用不变量

- 同 namespace 仍按原 token/live org/member/supply.read/manage/submit 检查；另一 namespace 不是绕过源记录权限的入口。Context 只能在已授权范围返回固定引用。
- 本地四类 POST 由服务端 policy 验证完整输入/来源；正式 POST 不接受 localTrial。读取后用途 guard 拒绝另一类 head/record/原键；无标记不等于已获正式资格，仍须原正式 merchant/rule/Asset/expiry 准入。
- 两种 mode/URL 都不是许可。仓储 head 锁内 `SameLocalTrial` 保持，阻止已有 Target 脱标、换标或插入测试版本覆盖正式 head；`localTrial`、输入、记录/hash 的完整绑定不变。两个真实/测试 Store 的 Target 不共享 head。
- 同一 Product DB 原 command key 唯一约束仍生效：同键同完整输入只取历史回执；同键不同载荷/用途冲突，不产生新副作用。错误 namespace 的 GET key 拒绝，UI 提示在对应草稿核对原操作，不自动改键、重发或切换 namespace。
- UI 每个新 pending intent 固定 `draftKind`、source/store/base record/规则绑定及原操作键，scope/storage identity 加入用途；切换草稿先明确保留/放弃未保存内容，未知请求先核对原键。历史 durable key 不改、不回灌/迁移浏览器 storage；无法确定用途的未决 intent 停止写入，显式选择对应草稿后只读核对原键。
- `PublicationReady(local)` 永远 false；native `local_trial` 与未知 authority 不能生成 wire。现有正式 authority 表示/版本不改变，不能把浏览器空标记当官方证明。Uploader、prepare/confirm、plan继续、adapt、materials/image/Agent、worker/recovery 继续在任何 provider/工作流/账本写前拒绝本地记录；正式 dispatcher 也不接受本地结果。
- 关闭本地新写不影响原键只读；旧 HTTP POST 超时/返回丢失也只能沿原 command 解决。授权撤销、Store 改变或固定 R1 不匹配时新写 fail closed，不能 fallback 到另一种服务。持久化历史读取按当前授权保留，不因凭据失效把历史改成正式记录。
- 本地 policy 的正式目标拒绝只针对其准确 org/测试 Store，不能按共享 Source/R1 全局封锁正式流程。其他真实店铺的既有正式 pending/UNKNOWN/回执不回写，原恢复 owner 和当前权限检查不受本地写开关改变。
- 超时、最大输入/图片/SKU 量、规则/授权 expiry、单事务失败回滚、正式 UNKNOWN/retry owner均沿当前合同；本轮不新增任务、账本、定时恢复器或模型调用。

## 6. 配置、运行与保留

| 显式配置 | 服务能力 |
| --- | --- |
| 仅 controlledLocalDraft enabled=true | 本地八条接口及现有独立 SourceMaterials/Completion；无正式执行 |
| controlledLocalDraft enabled=false | 本地 context/head/record/原键四 GET；不构建本地 writer/rules/provider |
| 仅合格 SupplyChain | 原正式能力；本地记录即使被直接传入也保持 native 发布拒绝；无本地写入口 |
| 合格 SupplyChain + controlledLocalDraft true/false | 正式完整模块与本地八条/四条同时工作；各自前置独立，路径和模块无冲突 |

`SupplyChain` 一旦显式配置但官方应用/Store等前置不完整，继续拒绝该错误配置，不通过降级成本地模式掩盖；未配置正式模块时可正常使用本地模式。`PrivateDraftTrial` / `SheinOfflineRecords` 等其他互斥范围不变，不顺便泛化所有试用模式。

Product capability 仍按是否安装现有 SupplyChain schema（两个显式装配取 OR）声明，不按新写开关推测。Store/SourceMaterials/Agent 的既有 capability 保持，正式 full pointer 不能被本地 app 覆盖。两者使用同一当前 Asset owner/合格连接，尽量重用已打开 reader/pool并明确一次关闭责任；不同 Asset owner/config 拒绝。无新 DDL、密码重置、表迁移或第二事实源。

UI 两个开关只控制入口展示，服务端白名单与 policy 决定实际能力。L/F 的严格 DTO、错误分类和 pending key 不能因为入口同时开启而混用。旧本地 API 地址不保留兼容别名；UI/BFF/backend 作为同一交付更新，原网页入口/记录和原键数据保持。未来实际更新先停止写入，保存配置/版本及现有备份；后端和 UI 不混搭新旧版本。

当前原 35444 的商品/R1/图片、测试 Store、Target R3、原预留及已登记凭据全部保留；本轮不操作它们。后续若部署共存版本只作同 schema 的程序/配置更新，不为显示两类记录重新初始化或拷贝数据。回退先停写并使用匹配程序/UI/配置；本轮设计不修改 schema/记录格式，仍需隔离副本证明原版本读回现存本地记录/原键。若已有正式新写，回退必须保留并核对其 owner/命令/UNKNOWN，不自动还原旧 checkpoint 丢弃新记录，不清卷/删表/删除本地稿。

## 7. 验证与实施准入

复用现有配置/路由/Target/PG/BFF/浏览器测试及保留副本，不建验收平台。以下四项 **IMPLEMENTATION_TEST，全部当前 NOT_RUN，后续实施及合并前收敛**：

1. **装配与路径**：正式-only、本地-only、双模块、本地停写四状态；真实 schema/capability/Asset共享、唯一 kernel 名称和 Method/Path；保留 SourceMaterials/Completion/Agent，正式缺项 fail closed；本地context范围及四/八路由白名单；旧地址无本地 fallback。
2. **用户/数据路径**：隔离同实例用同一 Catalog/R1，保留现存测试 Target R3并新增恰好一份正式 Target；正式规则/Store transport用受控替身而业务 owner/PG真实。两个 Target 各三SKU，0/空/空、图片归属与原 R1保持；正式枚举/仓库必须重新选，重开/刷新和 mode 切换不串稿、不丢未保存内容；不冒充真实平台资格。
3. **发布及用途隔离**：测试稿即使其他字段齐全，跨 namespace 直 POST/原键/脱标/换 Store/adapt/plan/material/image/worker/发布 prepare/confirm均在外部副作用前拒绝；namespace 读取不能混用途。正式合格 fixture 仍走原真实读者合同与提交准入，不因同实例有测试模块被统一关闭；规格集合未确认继续非Ready/无wire。
4. **保存/退出**：原锁/CAS/完整hash/同键冲突、重试/ACK未知/事务回滚/live撤权；本地停写后历史与原键可读且正式仍可用；同 schema 隔离副本启用/回退保留两类记录、原商品/R1/Asset/权限及原预留。真实连接/模型/发布/原实例验收另列 NOT_RUN，不为设计或fixture PASS改写。

架构敏感边界是“同实例不同用途读取、依赖/路由及执行准入”，不是取消既有发布防线。Reviewer先读本节 Must/Threat Model/Out of Scope；finding按AGENTS分类，仅真实BLOCKER重开冻结设计，最多两轮正常Architecture Review；非阻塞实施细节归上述IT，不追加通用系统或新产品要求。

Legacy decision：N/A。保留的是本系统当前 durable records 与明确用户数据，不提供旧系统兼容/迁移、HTTP fallback、旧ID映射或双写。复用原 owner/成熟 PostgreSQL/Temporal/Gin/严格DTO能力，无新开源替代需求。

## 8. 独立评审状态

待独立 R1，尚未达到 IMPLEMENTATION_READY。准确设计提交、finding 分类、必要修订与最终结论在本节及 #1158/关联 Draft PR 记录；设计通过也不授权实现、合并、部署或业务写入。
