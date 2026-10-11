# Store Center：我的店铺完整 Figma 路径 v1

日期：2026-10-11。Design Basis：**Independent Architecture**。
Admission：**IMPLEMENTATION_READY**。独立评审基线 `5b54485c4a5e6cd20a46b1931570a2a9d15f0c75`，设计级 BLOCKER = 0；本文件先于正式实现。
基线：main `c023edad14ad9255dee3e04a67b539bdaf2c83b8`。
执行 Issue：[#641](https://github.com/qq550723504/task-processor/issues/641)。
唯一 Writer：会话 `01a129ba-3227-78a2-a8d2-07d7fdac4423`；一个 Delivery Batch，一个主要 PR。

## 1. 用户结果与产品决定

当前企业用户在 `/workbench/stores` 查找真实店铺，组合名称/ID、平台、站点、类型、状态和分组筛选，查看真实授权与同步状态、商品/订单/库存摘要，绑定官方 SHEIN 店铺，管理展示名称、负责人、分组和用户设定的低库存阈值，发起同步、处理具体异常。分组、归属与设置刷新/重启后仍存在。批量选择只服务于加入分组，不建设通用批处理平台。

用户在本会话明确选择（2026-10-11）：

- 完整实现当前店铺页的 Figma 功能，先补设计准入再开发。
- 使用官方 API 授权。沿用 #561 决定，旧子账户密码/验证码登录不恢复。
- API 操作不需要 Figma 的固定 IP、代理配置、网络检测；这些能力和其网络状态展示退出本批范围。
- 负责人只记录业务归属，访问权限继续由现有店铺授权管理。修改负责人不授予、回收或改变任何 grant。
- 低库存阈值由用户设置，不植入默认业务阈值。未设置显式展示“未设置”；缺失库存保持未知。

Must：完整服务端筛选与真实总量；主题/宽窄屏和卡片/弹窗结构对应 Figma；分组 CRUD、单店/批量归组、负责人/阈值设置可保存回读；官方授权、同步原命令恢复、连接和记录/服务状态分离；A/B 企业及成员店铺权限隔离。

Should：异常能定位到具体店铺/原因/观察时间，键盘可使用对话框和多选，原有合法操作可继续使用。

Out of Scope：网络/代理；子账户登录/验证码/浏览器托管；新增加拿大等平台读取站点；财务/广告同步、销售额/利润公式、AI 自动执行策略或库存回写；发货/售后/退款；新支付/套餐/额度语义；旧数据迁移/兼容；通用框架、专项验收平台；合并、部署、真实商家/业务数据操作。详情仅补齐列表动作所需的管理、授权、数据同步和设置路径，不自动扩展上述经营/AI 能力。

Accepted Risks 沿当前授权、平台观察合同：IAM live 校验与本地数据库不是分布式事务；官方商品遍历不是平台原子快照；观察只能代表其取得时间/范围；开发测试不证明真实应用已获批或用户验收。

## 2. Authority 与可见映射

Authority：[最终 UI/IA](../product/final-ui-ia-authority.md)，文件 `tg48P46SSXl6TBy9lZwg63` / 页面 `31:463`；2026-10-11 已实际读取截图、节点、文案和 prototype reactions。

| 交互 | 当前节点 | 实现与事实边界 |
| --- | --- | --- |
| 我的店铺深/浅色 | `1560:359` / `1560:716` | 四概览、紧凑筛选、横向资产卡；移除网络状态，样例不是业务事实 |
| 平台/站点/类型/状态/分组筛选 | `1622:359`，`1610:359/370/392/381` | 服务端完整集合筛选；未知独立于正常/需要处理；站点选项来自现有记录，不能把 region 字符串当平台支持证明 |
| 分组管理/新建 | `1583:363` / `1579:390` / `1869:359` | 删除分组不会解绑店铺；店铺自动移出该组，这是 Figma 明确行为 |
| 批量选择/加入分组 | `1868:570` / `1883:2274/2471` | 保留优化卡片布局并提供 checkbox；操作只使用明确选定的真实 store IDs |
| 更多操作 | `1608:359` | 改名称/负责人/分组、重新官方授权、解绑，复用当前 owner 与版本控制 |
| 绑定店铺 | `1548:402` / `1604:503` | 保留弹窗结构，用 #561 官方授权替代旧 API 未开放/子账户文案；无固定 IP 开关 |
| 管理/授权/同步/设置 | `1569:359/676/975/1604` | 真实基础事实与列表动作；无网络页签、虚假利润或 AI 权限开关 |
| 同步范围弹窗 | `1854:364/434` | 复用商品/订单观察 begin/by-key/ensure/status，选择当前店铺或当前授权集合 |
| 进入店铺 | `1560:672` reaction → `429:5323` | 跳到现有店铺商品页并携带该店铺筛选，不继续误跳记录编辑页 |

当前根因：`StoreListPage` 只消费基础 CRUD DTO 与 platform/record-status 三个筛选；没有完整目录检索/归属 owner 和跨 owner 的有界状态投影。仅改 CSS 或把 Figma 数字写进页面不能解决。已有表名/目录中 listingkit 字样不授权删除合格 Console/BFF。

## 3. owner 与调用链

复用已批准设计：[Native Store 与预付资源](unified-base-prepaid-resources-v1.md)、[Store core](store-center-current-application-v1.md)、[成员资源与官方连接](member-resource-allocation-v1.md)、[成员目录](member-directory-query-v1.md)、[平台观察](store-center-platform-observations-v1.md)、[认证/租户](auth-and-tenancy.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md)。Native Store 决定覆盖旧 core 的 quota/provisioning/resume 含义。

| 事实 | 唯一 owner | 本轮动作 |
| --- | --- | --- |
| 店铺记录、name/region、版本、启停/解绑审计 | `internal/storecenter`，Store DB | 复用原用例，扩展当前目录合同 |
| 官方连接、应用模式、private material | 同一 Store official connection owner/registry | 消费当前状态和 AppID/revision/type；不复制密钥或创建第二授权 |
| 成员权限、身份与店铺 grant | ZITADEL / membership / Store member grants | 原授权不变；负责人只保存 canonical active member ID |
| 店铺分组、业务负责人、低库存阈值 | Store owner，新本域事实 | 同 Store DB，独立窄 repository，用当前事务/CAS/receipt/audit 方式 |
| 商品/库存、订单、同步命令与当前 generation | `marketplace/shein/observations` | 新窄只读摘要查询；只改本 owner 的投影，不跨 owner 直读表或写第二 head |
| 同步执行 | 现有 observations service / Temporal | UI 接现有 begin/by-key/status/ensure；不新建 runner/scheduler |
| 服务期/续费/资源扣减 | 现有 Store/Commercial/resource owner | 原调用链与权限不变，只保留真实操作 |

调用链：Console → 同源 strict BFF → CurrentIdentity / EffectiveOrganization / live action & store authorization → Store catalog/profile/group usecase → scoped Store repository；状态摘要经注入的 official connection/observation narrow ports 组合 → DTO → React Query。App 负责授权/组合，领域筛选、库存比较和数据覆盖规则留各 owner。禁止 App 直读 observations 表或注入带 Publish 的宽 merchant handle。

## 4. 目录筛选与状态/指标

扩展现有 `GET /api/v1/workbench/stores` 的严格允许参数：page/pageSize、q、platform、region、applicationType、groupId、attention；现有 record status 参数保留其合法当前语义，不作为复合正常状态的别名。q 是 trim 后的 Unicode literal substring，匹配自定义名称、当前 Store UUID、externalStoreId；不把 `%`/`_` 当 SQL 通配符。参数单值、未知键/非法编码拒绝，q ≤200 UTF-8 bytes，pageSize ≤100。

所有 predicate 先作用于当前成员可访问集合再 count/page；排序沿现有稳定 updatedAt+ID。不在前端一页筛选或用首100条代替完整集合。Facets/概览也在同一授权集合及筛选语义下计算，不能泄漏未授权分组店铺数量。组列表可供当前 Store reader 浏览名字，组计数只包含其可见店铺。企业级分组CRUD只允许当前enterprise administrator且具有`workbench.store.update`；单店profile和批量归组须已有update权限与每店现有访问授权，不能借共享组操作修改自己未授权的店铺。

normal/attention 是只读展示分类，不保存第二业务状态机：connected 且所需已开放商品/订单观察当前 binding 下成功完整，并且无已知阈值预警才可 normal；明确断连/过期、同步 failed/partial/suspended 或已知低库存归 attention。未配置能力、未同步、缺项/不完整库存不是 normal，也不是假 0；显示 unknown/incomplete 与原因。记录 disabled/deleting、服务 pending/expired 等继续分别展示，不能互相推导。

概览返回 total、normal、attention、syncHealthy 和 unknown/coverage。同步正常只取当前 binding 下完整成功的 required kind；全托管消费者订单沿现有 unsupported，不伪装已同步。Metric 注明筛选范围及取得时间；unknown 不计入正常，用户可以进入具体原因。

平台/站点/类型不得从 Figma 样例证明：platform 仍 SHEIN；类型来自实际官方应用 registry/type（未授权显示未确认，不接受前端自报半/全托管覆盖官方事实）；站点筛选沿实际 region 字段，但读取仅使用现有 observations capability（目前 shein-us）。加拿大样例不是扩站授权。

在线商品取当前完整产品 generation 的 active SKC 数，待处理订单取当前观察 owner 的 pending 定义。未知/缺权限/unsupported/未同步用 nullable value+reason+coverage，不能填 0。最近同步取各 kind owner 的实际时间，不把 Store updatedAt 叫同步时间。

只读组合有明确 request deadline、现有最多16独立 Store检查并发上限，provider页仍串行。完整目录大于现有观察 owner 支持集合或 deadline/响应预算时，保留基础目录、把摘要显示 unavailable/incomplete；不能静默截断并声称全量。响应预算沿现有2MiB，集合查询不引入定时后台刷新或持久化聚合缓存。正式实现前按现有 owner 支持的批量查询选择最小路径，避免逐店加载完整商品详情。

## 5. 分组、归属与阈值持久化

新增当前 Store-owned groups、profiles、命令 receipts 三类最小事实；复用 Store 数据库/borrowed pool，不创建新 database。profile 唯一 `(org,store)`，含 nullable groupId、responsibleMemberId、lowStockThreshold、version 与 provenance；group 唯一 `(org,groupId)`、name、version、soft delete；receipt `(org,operationId)` 固定 actor/member、action、canonical payload hash 和结果版本。无本地成员目录副本、无新 grants。

新 profile/group 是本域 metadata aggregate，各自显式版本；其读 DTO 用 `profileVersion`/`groupVersion`，不误复用 Store CRUD version。缺 profile 表示未设置（version=0），不是另一个 Store。Store identity FK 带 org；group FK 同 org。负责人值经现有活跃企业成员查询校验，但不读取私有联系方式。后来离职保留 ID/provenance，并显示成员已不可用；不把 rejoin 的新 MemberID 自动映射回来，也不因此删店铺/历史事实。

所有写入固定 UUID Idempotency-Key + expected profile/group version。相同键相同规范载荷返回原 receipt；不同载荷 conflict；重试也重新授权。请求失联/刷新后按原操作键读取，不能把当前值误当原命令成功。profiles/group mutation 和本地 receipt/audit 在同一 Store 事务提交；事务 rollback 不留下部分成功。使用 GORM+现有锁/CAS模式，不造通用事务框架。

重放先验证当前身份、动作与原目标 scope，再核对 receipt 的 actor/member/action/hash；返回冻结结果，不重新执行后来版本、已删除分组或已离职负责人的状态验证。同键不同身份冲突。所有 metadata 写统一按 group ID → Store ID → grant → profile 的锁序；无 profile 时由 Store 行锁及唯一键序列化首次创建，version=0 只创建一次。删除组先锁组，再按 Store ID 顺序锁受影响记录/grant/profile。

单店调整 group/responsible/threshold 合并成固定窄 profile command。批量归组接受显式最多100 Store IDs+每项 profile version，同 scope 全部鉴权，按 Store ID 锁顺序，在一个 Store 本地事务里校验/更新/receipt；任一不可访问/删除/版本冲突则全部拒绝，UI保留选择、回读再确认。包含隐藏店铺返回当前不泄漏的404/deny，不返回其资料。

删除分组在同一事务锁组及其归属 profile、清空 groupId 并递增受影响 profile version、写 receipt/audit，然后 soft delete group；不变店铺记录、负责人、grant、连接、服务/余额。组赋值/重命名/删除用同组锁顺序，不能在删除后复活引用。Store 解绑沿原删除协议；metadata不参与额度/服务恢复，不物理删除历史审计。

分组名字 trim、1..60 codepoints，控制字符拒绝；允许名称重用但 UI按group UUID区别，不发明名称唯一商业规则。阈值 nullable 非负安全整数，上限沿平台数量/JSON安全界限；清空代表关闭该店铺低库存判定。没有默认阈值。

低库存规则：当前保存的 SKU/warehouse quantity 已知且非负，`quantity <= threshold` 为一条预警；按 `(SKU ID, warehouse ID)` 去重，卡片明确“SKU/仓库预警”。未知不参与比较；统计同时返回 unknown inventory count 与 coverage。阈值/库存范围变化重新计算；比较只来自当前平台观察 generation，不持久化第二库存事实或写平台。没有设置则value=null/reason=threshold_not_set，不报 0。列表点击预警进入该店铺商品/库存明细，并显示阈值和来源时间。

## 6. 授权、同步与异常动作

绑定入口按 Native Store CreateOrReplay 创建记录，响应丢失用原 create key/指纹回读，再用原 official begin/exchange/status/recover；不恢复旧 quota/provisioning/resume 路由。每步保留原 idempotency/CAS/version/UNKNOWN。记录创建成功不等于授权成功。选择应用保留实际 AppID/revision/type。私密 callback参数及时清理，敏感材料不进普通 DTO、日志、URL或localStorage。response lost/UNKNOWN不重取或重发credential exchange。

“立即同步”复用现有商品/订单同步对话框与权限/capability，当前店铺先选定；用户可显式扩到当前全部授权店铺。一个kind一个原命令身份，复用原固定集合/child keys/Temporal/restart；双击不重复命令。同步进度只显示owner当前checkpoint，不能用本地timer造百分比。需要处理进入具体店铺的authorization/sync/settings原因，不能把所有异常都跳新绑店页。

“管理”到当前店铺管理详情；“进入店铺”到`/workbench/store-products`的现有store筛选；重新授权/解绑、启停和续费复用现有组件和风险提示。没有相应权限不开放操作；切企业/撤权清空选择、弹窗、旧结果并阻止晚到结果/提交。修改负责人不会变更 Store member grant，无隐藏自动授权。

## 7. 安全、安装、Legacy 与验证

Threat Model仅当前Must：跨企业/未授权店铺读写与数量泄漏、通过metadata提升权限、敏感credential泄漏、并发归组/删除导致资料丢失或复活、命令重试产生错误重复效果、把陈旧/缺失平台数据冒充正常。

runtime只读 schema/权限preflight；新表由现有schema-owner空安装入口显式安装，serving角色最小SELECT/INSERT/UPDATE及receipt/audit append-only。constructor不得AutoMigrate；缺schema能力显示unavailable。真实/共享实例schema操作与部署需要另行授权；不碰当前保留volume或旧数据，不做backfill/旧ID映射。

Legacy decision: **RETIRE**。复用合格 current Console、Store/official/observations/GORM/Temporal行为；不包装legacy source-account cookie/session、旧sheinsync、Task-first Product、宽publish handle或共享DB builder。当前Store事实不复制，profile不是第二Store identity。

实现顺序（同一分支/主要PR）：Store metadata本地合同/事务与目录查询 → observations窄摘要/库存阈值投影 → strict BFF/client/Query → Figma卡片、筛选、分组/归属/阈值弹窗与既有授权/同步接线 →最终集中交付检查。

业务TDD先验证：跨org FK/成员scope、负责人不改grants、分页完整筛选、同键冲突与原receipt、组删除/归组并发、批量原子失败、未知库存/去重/阈值边界、当前binding/coverage/权限过滤。复用现有SQLite/Postgres隔离测试与Temporal testsuite；不新增验收runner。最终只跑相关Go/UI/BFF测试、类型/构建和必需CI；截图实际对照深/浅色与窄屏，真实平台调用/产品验收保持NOT_RUN。

## 8. 独立评审与准入

2026-10-11，独立只读 reviewer `store_figma_architecture_review` 检查 #641、设计基线与当前代码，结论 `IMPLEMENTATION_READY`，BLOCKER = 0。以下均为 `IMPLEMENTATION_TEST`，合并前收敛，不重开全局架构：统一锁序/首次创建/原子批量与撤权；冻结 receipt 重放；摘要 predicate 不得在摘要不足时静默忽略；schema-owner、append-only grants 与所有当前消费者 capability 接线；Native Store 引用及原键恢复澄清。

摘要端口逐店返回 nullable value、coverage/reason/取得时间并保留各模块读权限和前后 source 校验。基础目录不受 observations 500 店上限裁剪；摘要不足时，依赖摘要的筛选明确不可用/不完整，不能退成未筛选分页、假 0 或完整 total。schema 缺失明确 unavailable，receipt/audit 仅 SELECT/INSERT；同步实际 Store/Supply/resource 消费者的窄 capability 清单。完成本文件准入记录后唯一 Writer 可实施；运行及用户验收仍单列。
