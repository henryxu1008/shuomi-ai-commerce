# CompletionDraft 结构化候选 v1

Design Basis: Independent Architecture
Admission Status: IMPLEMENTATION_READY / FROZEN — 2026-10-15 R1独立检查通过，设计提交dc4e33327e0222049580372da582767fc930b359；固定SKU集合/深复制、未填与零/旧hash、完整回执与消费者保留结构作为IMPLEMENTATION_TEST在合并前收敛。

## 1. Outcome / authority / boundaries

Issue #718，依据 [PD-COMPLETION-STRUCTURED-2026-10-15](../product/completion-draft-structured-v1.md) 与用户只恢复文字的明确选择。Figma/UI authority、独立页与非目标见产品决定；本设计先于生产实现。

结果：草稿→独立结构化编辑→原 Save 追加完整 Rn→刷新继续编辑→历史详情与差异比较；返回草稿、商品和批次上下文贯穿。价格/库存只是候选，不触达平台、资金或实际库存。

不做 SKU 增删、编码/规格/图片编辑、来源升级、源商品修改、结构化历史恢复、Target 映射、Store/provider/AI/发布、后台恢复器、schema/grants、保留数据迁移或验证平台。

## 2. Owner / content / injection

候选唯一 owner 仍为 `internal/listing/completiondraft`；固定 SourceItem、原 Product snapshot、资产和平台事实保留 Preparation / Catalog / 原 owner。复用 [保存](supply-unbound-completion-drafts.md)、[历史](completion-draft-history-v1.md)、[文字恢复](completion-draft-restore-v1.md) 的权限、版本、事务和原键恢复。

原 Content{title,description} 新增可选 structured：

```json
{"title":"...","description":"...","structured":{
  "attributes":[{"name":"材质","value":"棉"}],
  "variants":[{"sourceIndex":0,"price":{"currency":"CNY","amount":0},"stock":0}]
}}
```

- structured 缺失/null 均为本记录尚未保存结构化候选，规范写 JSON 时省略。原文字 hash 和待核实命令 hash 不变；不回填、迁移或重新解释旧记录。这是当前有效记录的未填语义，不是 legacy adapter / fallback / 新事实源。
- 对象内 attributes/variants 必须显式数组；空属性是已保存空集合。Save 是完整候选快照而非 patch，数组顺序保留并参与 hash/比较。
- 属性 <=256；名称合法 UTF-8 <=200 bytes、trim 后非空且 trim 后唯一；值 <=8192 bytes、允许空串。拒 null 字符/非法 UTF-8，文本原样存储并仅文字渲染，不映射平台属性或销售规格。
- SKU 身份为**固定 SourceItem Vn + 原 snapshot.variants 的零基 sourceIndex**，不按可空/重复 SKU 编码猜匹配。最多256行，须严格为0..n-1，数量与准确固定 snapshot 相同；服务端拒重复/越界/缺行/重排/新增 SKU，不接收来源编码、规格、图片等字段。
- price 为 {currency,amount} 或 null；stock 为整数或 null。null=未填，0=明确零。币种 `[A-Z]{3}`，金额 finite/nonnegative 且 <=9007199254740991，库存整数0..9007199254740991。签名零规范为JSONB读回的0，保持保存与回执hash一致；不做换算/舍入/资金或平台校验，沿用 Product 数值表示。
- 来源 >256 变体时明确超出结构化编辑预算；文字/历史仍可用。自有商品当前 UI 同为256变体上限。
- 未保存结构化内容时编辑页可从**准确固定来源**预填，明确尚未保存。已保存结构化内容（含空属性、null价格/库存）准确读回，不能被来源原值或 `||` 覆盖；无法规范预填的来源值显示未填并由表单验证。

调用链：`/workbench/supply/mine/missing/:sourceId/structure?preparation=...&item=...`→当前context/权限→原readSupplySource/readSupplyCompletionDraft→严格Zod+原useSupplyCommand/save-completion-draft/BFF→原Service+SourceSelector准确Catalog Vn及候选成员验证→原Product pool Save事务→原版本/命令回执。App/injection/路由/能力开关不另加owner。

Save 在 mint Prepared 前复制完整输入并校验固定 snapshot；Prepared.Read 返回内容副本，避免新增数组/指针在 hash 与事务间被外部修改。内容 equality 比完整值，不能比指针地址；原内容/命令 hash 核对覆盖全部字段。

## 3. Permissions / transaction / failures

原 verified identity、live membership/module grants、org/actor/member/source归属、read/manage、SourceSelector与提交前authority proof不变。只读可看当前/历史，不能保存；撤权、错scope/source/批次/item、token过期或迟到响应拒绝。导航ID不授予权限，组织切换remount+abort，原pending intent隔离。

沿用同Product DB的records.content_json、heads、commands；**无DDL/grants/新表/额外持久化边界/第二保存协议**。原完整快照、CAS、advisory transaction fence、不可变命令回执和提交前授权不变。

1. 验证全部字段后才写；未知字段、错误属性/数字/固定SKU集合拒绝，无部分record/head/command。
2. expectedRevision=准确预览head；并发不同键同revision仅一个成功。409后锁保存，用户显式重读再确认，不自动merge/覆盖当前候选。
3. 同scope/key/完整规范输入重放原record；同键不同结构内容冲突，新head不能替代旧回执。
4. 失败按原rollback；COMMIT/响应丢失、重启、取消沿原durable browser intent、原key查询和同key retry。无新恢复器/盲建新key，pending时文字/结构/恢复均沿原写锁。
5. 原64KiB整个Save JSON、3s body、20s HTTP/10s domain、2MiB response、分页/no-store不变。表单和intent检查相同总预算，不扩大运行配置。

## 4. Consumers / history / restore

- 原文字编辑Save携带**当前head的structured**；结构编辑Save携带**当前head的title/description**。两页共用Rn CAS，任何409后显式重读，不从旧页面自动覆盖新字段。
- 历史准确显示保存值；缺structured明确未保存，空属性、null、0分别展示。比较按属性行位置/name/value与固定SKU sourceIndex逐字段显示增删/名称/值、币种/金额/库存变化，不用最新商品或来源值填充历史。
- **恢复只采用旧Rn的title/description + 当前预览head的structured**，Save(expectedRevision=m)追加m+1；只因结构字段不同不能启用文字恢复，当前版/相同文字不写冗余版本。恢复预览只比较实际恢复的文字，并明确当前属性、价格、库存保持不变。
- 回执核对完整发送内容/source/draft/revision，重放回执不冒充最新head。结构化历史/比较与文字恢复预览分别组合，保留原独立历史/比较/恢复页。
- Target既有applyCompletionText及服务端文字核对只消费文字，UI明示尚未带入属性/价格/库存；不把通用候选当平台资料。

## 5. Legacy / tests / delivery

Legacy decision: N/A。当前有效CompletionDraft/Preparation/Catalog/Console；无retired owner、migration、双读/双写或fallback。复用React/Zod/表格/主题/GORM/PostgreSQL、既有保存与恢复组件，不建通用草稿框架。

必要TDD与自检：

- Domain：0/null/空集合、Unicode/预算、准确SKU集合、伪造/越界/重复/重排及非法数字拒绝，完整input copy/hash/equality与原权限/固定来源/撤权。
- 复用现有fresh testcontainers PostgreSQL：结构快照读回、原无字段记录hash/旧命令回执不变，同键不同结构冲突、完整CAS、原事务/UNKNOWN和历史。不能接保留DB。
- API/BFF/Zod：严格同载荷/预算/原route，结构回执和历史解析，非法/超限请求拒绝。
- UI：独立页预填/准确重读、0/null/清空、权限/导航/晚到/UNKNOWN/409；文字保存与恢复保留当前structured；历史逐字段比较、Target文字提示。
- 类型/生产lint/build、必要相关测试及最终准确HEAD CI；README架构索引齐备。独立Reviewer只做设计准入与完整交付里程碑，finding先分类，修复复核对应增量。

合并、Issue关闭、保留实例安装/业务写入、用户/生产验收分别另授权。不把CI、fixture或AI检查当产品验收。

后续发布须协调更新后台和前端，不设计混合旧DTO兼容路径。原文字记录/待核实旧命令在新二进制中保持hash；已有结构化新记录后不直接降级旧二进制，不能忽略字段或重算历史hash来伪造兼容。保留数据并修复前进，不自动回写/迁移/删除。本轮无真实部署授权。

2026-10-16 增量：[SHEIN 带入预览与确认](completion-shein-import-v1.md) 仅扩展 SHEIN Target 的候选带入范围；本文 CompletionDraft 保存、版本、权限和文字历史恢复合同不变。
