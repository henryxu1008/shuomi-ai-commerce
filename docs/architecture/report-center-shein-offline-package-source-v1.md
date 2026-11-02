> 当前准入：用户已认可下列原设计 HEAD `f45ce2bd2c4f589d5f319d8786e73b2e45628ea1`；[独立 R1](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1049#issuecomment-6554638529) 为 **DESIGN_REVIEW_PASS / IMPLEMENTATION_READY**。用户现已授权原唯一 Writer 在原工作区和 PR #1049 先补离线回归、最小实现、自检，并复用原 Reviewer 做增量及最终复核。当前 **IMPLEMENTING**；未授权合并、部署、关单或真实操作。下列候选正文为已审冻结依据，原提交时状态保留为历史；技术证据维护在 PR，运行及产品验收 NOT_RUN、缺证据 UNKNOWN。

# Report Center：原 SHEIN 离线包诊断正常接线候选

Execution: [#628](https://github.com/henryxu1008/shuomi-ai-commerce/issues/628)。

- Design Basis: **Independent Architecture**；涉及正常来源接线、live 授权、持久化准入和完整用户路径。
- Status: **DESIGN_ONLY / USER_SCOPE_CONFIRMED / DESIGNING / NOT_IMPLEMENTATION_READY**。
- Investigation baseline: `745117838485a5d6dc1aa081f818b22931e5a2d1`，2026-11-01。
- 用户已确认“保持原离线包范围”，并授权当前唯一 Writer 在原独立工作区整理设计和提交 design-only Draft PR。**没有实现、评审派工、合并、部署或关单授权。**

## 1. 用户结果与产品依据

使用者是当前企业内同时具有原来源权限和报告权限的内部用户。用户从“我的数据”选择本人有权读取的准确商品版本、精确已批准素材和 SHEIN 店铺上下文，显式生成一份本地离线包及保存时诊断，查看结果后再手动保存到“我的报告”，之后查看、收藏和下载历史快照。

“原离线包”指现有 `offlinepackage.Package` 及 `shein.offline_package.v2.2` 诊断合同。它不读取或诊断 Supply TargetRecord 中后来人工修改的 SKU、售价等平台字段。`action=save_draft|publish` 沿原诊断目的枚举；`publish` 在这里也不执行平台提交。

依据：

- 用户本轮原范围决定与 design-only 授权；[原个人报告冻结基线](ai-workbench-reports-v1.md)的个人/当前企业、手动保存、只读历史、原键和收藏合同。
- [我的供应链 SHEIN](my-supply-chain-shein-v1.md) §3：DRAFT-S1 业务能力可复用，隔离单库 constructor 不能照搬正常独立库组合。
- [全新系统基线](../product/greenfield-no-legacy-migration.md)、[当前系统边界](../product/greenfield-no-legacy-migration.md)、[当前 UI / IA 权威](../product/final-ui-ia-authority.md)。
- 原报告 Figma 权威：`tg48P46SSXl6TBy9lZwg63` / `31:463`，总览 `427:5695`/`427:5900`、最近 `662:481`/`662:641`、收藏 `579:4677`/`579:4900`、全部 `579:5123`/`579:5346`。沿用原已查阅依据；本轮未重读 Figma，不宣称新视觉验收。
- [#1045 资料来源设计](report-center-shein-target-source-v1.md)仅覆盖 `SHEIN_TARGET_RECORD`；该 PR 已合入，不能作为本批离线诊断已交付的证据，也不因本批重新打开其冻结设计。

Must：同一正常程序可完成本地结果生成→读取保存时诊断→手动保存报告；精确输入与摘要可追溯；本人/企业隔离和当前来源权限；保存报告不执行源业务；原记录与报告各自原子持久化、原键找回；默认关闭且启动前完成 schema/角色准入；保留原资源上限。

Out of Scope：TargetRecord 平台字段诊断、实时规则/在线许可、真实授权或平台调用、模型/Agent/费用、采集/上传/发布、旧系统迁移/导入/兼容、团队共享、额外文件格式、通用诊断或验收平台。没有新增 Accepted Risk。

## 2. 已有能力与实际缺口

| 当前路径 | 已查证能力 / 缺口 | 本候选处理 |
| --- | --- | --- |
| `internal/marketplace/shein/validator/diagnostic.go` | 严格解码已保存 Package，规范化绑定，复用同一个 `evaluatePackage`；无时钟或 I/O | 复用原校验；不造第二规则引擎 |
| `internal/listing/record/record.go` | 精确 Catalog/ApprovedAsset/Store 输入、不可变记录与原 operation；创建时保存 DiagnosticResult | 复用业务/哈希/事务语义，在当前 owner 补正常 live 授权接线 |
| `internal/app/listingrecordstore` | 两表原事务、延期 FK、原键回执、有界原包/诊断读回 | 沿同一事实 owner，补正常私有读取和 schema/权限准入，不另建结果表 |
| `buildLocalTrialListingRoutes` / `issue36TrialModule` | 仅显式隔离组合，把 Product、Asset、Store 读端口指向一个库；正常 local-trial 与 Product acquisition 还有互斥准入 | 不挂载或包装该 constructor，不启用 local-trial 来凑正常来源 |
| `buildReportCenter` | 正常只注入 Review 与可选 TargetRecord；没有 offline Record reader | 注入同一正常离线记录 owner 的窄 reader，保留现有 Report owner |
| `reportcenter.Sources.Read(SHEIN_RECORD)` | 已有保存时诊断白名单；却将 `d.Input.Digest` 与原包字节摘要 `r.PackageHash` 比较 | 按 §5 修复绑定；不删除校验或用相同假摘要蒙混 |
| 当前 Supply inspection / ProductInputs | 前者重新读取官方规则并限定当前 head；后者明确不评估 marketplace rules | 都不作为原离线包诊断的替代或隐式依赖 |

摘要问题是 **IMPLEMENTATION_TEST / 本切片合并前 Must**：源码推导可定位错误，现有 source unit fixture 人为令两种摘要相等，未覆盖真实生产链。本轮未运行复现，不记录实测 FAIL；它不构成重新设计整个 Report 系统的理由。

## 3. Owner、端口与正常装配

| 事实 / 行为 | 当前 owner 与候选消费方式 |
| --- | --- |
| 当前来源准入 | 复用当前 Collection 的 live actor/org/member 授权及 `ReadItem`，服务器核准 item 的精确 Product/version；不接受任意 ProductKey 作为访问权 |
| 商品与素材 | Catalog 有界版本 reader 从 Product pool 读；ApprovedAsset 有界精确 inventory reader 从独立 Asset pool 读；不得使用未批准图片或 unversioned fallback |
| 店铺上下文 | 从当前 member-scoped Store owner 读本地组织/店铺/SHEIN 平台元数据，只投影 `StoreReference`；不取凭据、官方执行句柄或查询平台 |
| 原包 / 诊断 / operation | `internal/listing/record` 与 `internal/app/listingrecordstore`，位于当前独立 Product/Listing pool 的原两表；唯一结果及原键 owner |
| 报告 / 收藏 / report command | 原 Report 独立 pool 与三表；从记录 owner 读取已保存结果，沿原 Report 事务写历史展示快照 |

正常 `currentapplication` 复用原 Catalog/Collection、Asset、Store 与 Report 实例构建一个离线记录能力并注入 reader；不另起服务器、不另建完整 Supply/Agent app，不要求 Temporal、模型或官方商品应用。端口由各 owner 实现，App 只装配和核准上下文，不吸收诊断规则或直接跨库 SQL。

原隔离模块仍按原显式准入独立存在；正常能力与 local-trial/private-draft-trial 混配拒绝启动，不把它作为缺依赖时的 fallback。不得恢复已退休的 Store source-access/audit 入口；`#1046` 当前边界保持。

新创建在本次当前来源读取和提交前各核同一 actor/org/member、来源精确版本与本地 Store 权限；来源改变即拒绝，不改用当前 head。跨库读取不宣称原子快照或原子撤权：固定 Catalog 不可变版本，素材以精确读取时完整 inventory 绑定；不加跨库锁、Saga 或同步。

## 4. 权限与当前请求边界

1. 复用 VerifiedIdentity、effective Organization/live resolver 和当前权限判定。只用已验证 user subject，不从 URL、角色名称、报告 grant 推导来源权限。
2. 原离线来源读/写权限仍是 `listingkit.admin.read` / `listingkit.admin.write`；新创建还须当前来源的 Collection 读取与 member Store 读取准入。普通 SupplyManage 或 ReportManage 不自动映射为这些权限，不修改真实授权。缺权限时入口明确不可用。
3. 正常来源的 list/read/command lookup 必须在读正文前限定 organization＋owner user；不继承旧隔离 reader 的 tenant-admin 看他人记录特权。源表没有 member 列，不能冒称 SQL 已绑定原 member；本次 live member 用于当前访问核验，原记录所有权仍是 org/user。
4. Report source preview 要 ReportRead＋原来源读；新报告保存要 ReportManage＋原来源读。Report adapter 继续核 ID/org/owner，不给 admin 例外。历史报告读/收藏/下载只需原 Report 当前权限与 membership，不重新取得源数据。
5. 新生成用同一 Record owner 的私有授权证明：App 核准来源后，由 Record 用例在 `Prepared`/原 INSERT 事务的新增写入和提交前重验 live scope/权限。只事前检查或写入后返回403不满足此约束。此为待评审的 transient guard，不新增持久化状态、事实或事务 owner；不让调用者提供 guard/actor，也不以 legacy 权限映射 wrapper 绕过原门禁。

当前 static `Authorizer`、隔离构造器和 persistence owner-bypass 不能直接充当上述 normal live/私有能力；未来只能在同一当前 owner 内 EXTRACT 有效行为并接合格端口，不增加兼容执行链。既有允许的隔离行为与有效隔离测试须按原范围保留。

## 5. 摘要身份及最小修复候选

所有摘要不是授权证据，仍需 §4 的来源和本人约束。

| 摘要 | 原算法 / 用途 | 正确比较 |
| --- | --- | --- |
| `PackageHash` | `sha256:`＋原始持久化 Package 字节 SHA-256 | 核原包字节完整性，不与诊断 subject 比较 |
| `DiagnosticResult.Input.Digest` | `sha256:`＋固定结构 `{binding_version,marketplace,site,action,rule_version,package}` 的原 Go JSON 规范化摘要 | 与同一原绑定算法从原包计算的 expected subject 比较 |
| `DiagnosticHash` | `sha256:`＋已保存 DiagnosticResult 字节 SHA-256 | 核诊断原文，包括原保存时的读取/评估时间，不用当前时钟重造 |
| `InputHash` | 原 org/user、Product/version/hash、精确 inventory hash、Store/locale/action、rule/policy revision 的 canonical input | 记录 operation 身份及 `SHEIN_RECORD` 的原 SourceRef.version，不替换成 package/diagnostic hash |
| Report content digest | 原 `Snapshot.Bytes` 对白名单 Document 的摘要（裸64hex） | Report 内容一致性；不与来源摘要混用 |

修复候选：在现有 `internal/marketplace/shein/validator` 中抽取 `DiagnosticValidator` 已有的严格解码/固定结构编码/subject digest 计算为一个有界纯 helper，原 validator 与保存结果读取验证复用同一算法。helper 不运行 `evaluatePackage`、不取时间、规则或平台数据，不接受 provider 字节或执行接口。

Record owner 先核原包/诊断字节摘要与版本；Report source adapter 使用原包、原 target/action/rule/binding version 得到正确 expected subject，再调用原 `ValidateBoundInput`，保留诊断元数据/新鲜度一致性检查。**不能改成比较 Raw PackageHash、跳过 expected 校验、覆写持久化摘要，或调用 `DiagnosticService.Diagnose`/validator 全量重算来“验证”历史报告。**

固定结构字段名/顺序、Package codec、前缀、原 rule/binding version 均不改；不支持的来源版本拒绝新捕获，不自动翻译或用当前版本替换。旧报告仍由 Report 自己读回；本候选不迁移或重解释旧记录。

## 6. 本地生成与保存报告是两个明确操作

候选正常路由命名为 `/api/v1/workbench/offline-shein-records`（尚未实现）：POST 生成；GET 私有集合、`/:id` 保存时诊断与 `/commands/:key` 原记录回执。沿原 verified/live descriptor、严格 JSON、错误映射、deadline/取消及大小边界；不裸挂隔离 routes 或增加通用 router。

POST 的候选严格意图为 `collectionItemId`＋原 `record.Input`；后者来自正常 source preview，仍由服务器重新验证与当前获准 item 的 Product/version 一致。item locator 仅用于新操作的访问准入，不加入或改写原 domain InputHash；客户端不能提交 Package、Diagnostic、摘要、正文、owner/org、时钟或凭据。

1. 在现有“我的数据”准确来源上下文选择店铺和离线检查目的，显式“生成离线诊断”。复用原 Builder＋ExactApprovedAssetValidator 及原两表事务，仅生成本地记录；缺精确批准素材保留 `not_ready`，不伪造诊断。
2. 原记录回执确认后，现有诊断展示组件显示**保存时**结果和未评估范围。旧诊断 GET 是显式重新计算结果，不能冒充保存时结果。正常保存结果使用明确 saved DTO；旧 GET v2 strict schema 固定未评估集合，不能无条件套给已核精确批准素材的保存结果。
3. 用户再从原报告页面的准确 `SHEIN_RECORD` 来源预览点“保存此版本”。不手抄 UUID，不在保存报告时生成记录、重新诊断、改素材、调用模型或查平台。源页到报告页仅带类型/ID；完整 version 重新由服务器读取，pending intent 和缓存 key 保留 kind/id/version 与 user/org/权限上下文。
4. 沿原报告最近/收藏/全部、JSON/文本和准确源记录链接。复用既有页面职责，不新建工作台/菜单或人工验收流程；未启用来源时不展示可用的新保存入口。

记录的 `not_evaluated` 按原合同保留：在线模板新鲜度、Store远端授权、cookie、POD、人工审核、submission gate 和缺失的 external-package freshness 等不转PASS。ExactApprovedAssetValidator 只解除其确实核过的素材 provenance 项；saved DTO 的差异来自原 owner 的事实，不让 UI 自行补删检查或重判 readiness。

## 7. 原键、事务与中断

| 场景 | 候选约束 |
| --- | --- |
| 新诊断记录 | 原 `listing_shein_records`＋`listing_shein_record_operations` 单事务与延期FK；未提交全部回滚。不新增结果/回执/恢复表 |
| 原记录响应/COMMIT丢失 | 保留原 operation key＋原 public Input；先走同一 owner 的私有 command lookup，使用现有 FindOperation 的已提交事实并核 owner、record/operation InputHash 一致。读取不重新取 Product/Asset/Store 或执行诊断；不直接暴露当前未做本人过滤的原 FindOperation |
| lookup结果 | 原已提交结果可恢复；返回的 Input 必须与 UI 原意图核对。同 key 不同 Input 冲突；404不等于确认没有并发/在途提交，5xx/丢响应继续UNKNOWN，不自动换key生成。不得把 source locator 差异伪装成原 InputHash 已包含的字段 |
| 创建重试 / 来源变化 | 不改变原 Create 的 exact fact/InputHash 冲突合同；先找回已提交结果，不能用当前素材/规则替换原结果。确需新诊断是用户明确的新操作，不是UNKNOWN自动重发 |
| 诊断已提交、报告尚未保存 | 离线记录仍属于原 owner；用户可再次明确保存报告。没有跨库回滚、删除或“清理孤儿结果” |
| Report响应丢失 | 原 scope/key/fingerprint先Lookup，回读已提交报告；新命令才读取源并比完整ref，继续原收藏set-target/锁/回执 |
| 源关闭、权限撤销或进程重建 | 历史报告/回执/收藏仍从Report原store读取；恢复结果不授予当前源访问或新生成权限。源owner原记录仍须当前本人/企业/原读权限 |

原记录 private command lookup 只是消费既有 durable operation 的只读入口，不另建 retry owner/状态机。提交前 guard 仍在原 Record 事务内；源事实读取与记录提交、记录与Report提交之间不宣称分布式原子性，不添加 scheduler/outbox/Saga。

## 8. Schema、角色与启用准入

候选一个默认关闭的 normal `sheinOfflineRecords` opt-in，同时准入本地生成/私有读取和 Report source 注入。启用要求当前 Product/Collection、独立 Asset、member Store、live resolver 与 Report pool 已具备；不要求模型、官方 API 或完整 Supply worker。该名字只是候选，不是已存在配置，也不改变真实 manifest/env。

- Product/Listing pool：原两张 offline record/operation 表，不新增事实表；在现有 `cmd/product-acquisition-init` 的**新空库初始化路径**以明确 capability 安装原DDL与窄grants。只扩现有 initializer，不新增迁移器/安装框架，不复用现有非空库 feature switch 对保留库追加本能力。
- serve 开始前只读核原表/列/大小约束、org-operation唯一键、record ID键、原延期FK和本人集合索引，以及准确 runtime capability。原 `NewRepository` 只验证 dialect/依赖，不能冒称已有这套检查。schema/角色不合格则拒绝 opt-in，在 listen 前失败，不等保存时报错。
- 原表只给 source runtime 必需 SELECT/INSERT；不授予 UPDATE/DELETE/TRUNCATE/DDL/owner membership，不授予 Product role 读取 Store/Asset 表或凭据的权限。与已有 acquisition runtime capability 验证/矩阵一起维护，不能让新grants被未声明消费者默许。
- Asset 与 Store 继续各自现有 initializer/角色及独立 pool，不复制表或合并身份。Record新增写入只在 Product/Listing原事务；不得跨pool塞进同一GORM事务。
- Report schema **无需新kind或新表**，`SHEIN_RECORD` 已是现有明确种类；原#1045第三种类及源opt-in不改义。本能力关闭不使已有三类报告/原键/收藏失效，不解除其他来源的准入。
- 不读取、ALTER、重建、复制、导入或升级任何保留库；其schema/数据存在性UNKNOWN。未来真实新安装/配置/启用也必须另行授权，不因设计确认或源码合入自动执行。

## 9. Legacy decision 与验证

Legacy decision: **EXTRACT**。

- Reusable behavior: 当前 DRAFT-S1 的确定性 Builder/validator、不可变 Record、两表回执事务、有界读取；原 Report 保存/恢复/收藏和展示。
- Current owner: Catalog/Asset/Store各持事实；Listing Record持离线结果/operation；Report只持个人历史展示。
- Cutover/deletion condition: 本批获后续独立准入后，仅正常新能力消费合格窄端口；不包装隔离 constructor、不恢复 Task-first/root ListingKit/Store旧入口，不引入兼容、fallback、双读双写或历史迁移。旧隔离有效行为保持原显式范围，不扩为默认正常装配。

未来只复用现有测试，先RED再最小实现：

1. 任务自有临时PG及合成source，通过正常装配实际生成Record→读取保存时Diagnostic→Report metadata/ref→明确保存→读取/收藏/JSON/文本。**不能用两个相同假摘要或seed报告表替代真实生成链**；先捕获 §5 错误，再验证正确绑定与篡改/错版本/跨动作拒绝，helper不执行规则。
2. 两actor/org、当前member/source/Store撤权、提交前撤权、只有报告权限、仅普通Supply权限均按 §4 拒绝；schema/missing source/混入隔离组合在启动时拒绝，off时历史报告仍可用。
3. 原记录提交/回执丢失的私有lookup、原Input冲突和UNKNOWN；报告原键/异ref、source关闭后回读、收藏旧key不重施。沿原事务回归验证回滚，不新建故障平台。
4. 原2MiB Package/诊断/精确事实读限制、原strict codec深度/UTF与JSON、原HTTP/BFF边界保持；Report仍≤128KiB/16 sections/256 fields/单值16000字节。全量必需检查超限拒绝，不截断、不提高上限。
5. saved/重新计算DTO差异、scope/类型/版本缓存与迟到响应隔离、入口关闭；transport/依赖计数证明无模型、平台、规则网络、采集、上传或源事实写入（除用户明确的新本地Record）。

## 10. 当前交付与停止点

本轮只交设计候选和README索引，适用文档检查/准确HEAD CI记录在Draft PR。**生产代码、SQL、HTTP/DTO/配置均未修改；功能回归、架构复核及运行/产品验收NOT_RUN，现场准备UNKNOWN。**

用户先确认候选；如需独立架构复核须另行授权复用原唯一Reviewer，本轮不派评审。只有设计达到IMPLEMENTATION_READY且取得实现授权后，当前Writer才可在同一批主要PR先补回归再编码。不得因文档CI成功自行标Ready、合并、部署或关闭#628；旧范围与原历史证据保留，不改写成产品验收通过。
