# 官方 Simple Editor 与描述图片独立上传

Refs #745 / PR #746。Design Basis: **Independent Architecture**（本轮新增 CompletionDraft 描述图片引用事实）；状态 **IMPLEMENTATION_READY / FROZEN**。独立 Reviewer own_history_review 已核对 design-only HEAD `5a441ccad1b6b2d02c22b91c20a63e40f15f98c5`，无设计 BLOCKER；编码/hash、引用/预算、保存/恢复、上传联锁、官方UI/主题归 IMPLEMENTATION_TEST，在当前批次合并前收敛。准入先于本轮生产实现。

## 用户结果与边界

2026-10-17 用户批准沿用 Tiptap，改用官方 Simple Editor 成套界面、适配应用深色模式、保留图片上传/保存/过滤/历史恢复合同及独立页面。随后明确图片可在描述内独立上传，并选择待补全草稿也可新增图片，先明确合同。

本决定替代原富文本合同“草稿只用固定来源图片”的本轮范围限制，**不改变固定 Product 来源版本**。新图片仅为候选描述资源，不回写 Product，不改变 SKU 图片、Asset 批准或平台图片。

Must：官方 UI 可实际编辑；商品与草稿描述直接选择文件、沿原上传确认后插入；安全保存/读回/历史/恢复；原权限、容量、固定来源、CAS/幂等/UNKNOWN 保留。Should：中文可访问名称，深/浅色、键盘和窄屏可用。

Out of Scope：新增通用图库/上传服务/恢复器、图片替换原商品或 SKU、自动 Asset 批准/平台图片带入、真实平台/模型、发布、own675 更新、配置/表/授权变更、清理/迁移、颜色/下划线/对齐/待办/代码/表格/协作。已上传未保存的对象仍按原容量规则保留，不删除。

Product/Figma Authority：本轮用户决定；[最终 IA](../product/final-ui-ia-authority.md)，Figma tg48P46SSXl6TBy9lZwg63 /31:463，自有商品3500:1100/3502:405、待补全2667:1948；沿原独立页面/返回商品与批次上下文。菜单及链接浮层是编辑器控件，不用模态框承载商品业务流程。

## Owner 与调用路径

- 官方 MIT Simple Editor 源码固定 [799929bea4804c73767562b69f8acc2acdb8ac86](https://github.com/ueberdosis/tiptap-ui-components/tree/799929bea4804c73767562b69f8acc2acdb8ac86)。按实际依赖复用工具栏/菜单/链接控件、hooks、icons 和样式，保留许可证与适配记录。官方演示上传/内容/外部字体/独立主题不进入产品。编辑内核保持当前 Tiptap 3.32.0。
- DescriptionEditor 提供当前 allowlist 对应的官方控件；应用 `.dark` / Console tokens 拥有主题，不产生第二主题状态。组件样式限定编辑器及其 portal，不修改 html/body/应用布局。
- 图片字节、确定性 org/actor/hash object key、不可变存储、JPEG/PNG 检查、容量预占/确认仍归现有 SourceMedia/S3/storagecapacity；同页只有一个 SourceImageUploader 实例及 pending owner，避免两个组件同时消费同一个 localStorage 意图。
- 新建/编辑商品：原 SourceImageUploader 成功 → 原表单图片池追加 → 编辑器插入原 URL → 原 Collection Save；原商品公共图片最多40、SKU图片/来源事实及版本能力不变。
- 草稿：原上传成功 → 当前候选描述插入原 URL → 原 CompletionDraft Save 将描述及其新增图片引用一起保存；固定 SourceSelector/SourceItem 完全不变。无需新上传 API、数据库表、配置或 runtime 接线。

## 描述图片内容合同

CompletionDraft.Content 增加可选 `descriptionImages: string[]`（Go `omitempty`），表示**当前候选描述中额外准入的图片 URL**，作为完整描述资源引用随原 Content 版本化，不是独立图库或另一份 Product。

- 缺省及空列表均不序列化；现有纯文本/富文本请求 JSON、hash、回执和历史字节不改写。UI 无修改不补字段。已有记录不回填。
- 数组最多40项，逐项规范 HTTPS、无 userinfo/fragment/空白/反斜杠、UTF-8 URL最多2048字节，且去重。沿当前 Product 公共图片 URL 语义允许公开来源链接；它不证明文件是由当前 actor 上传、属于私有对象或已获 Asset 批准。上传入口仍只能返回当前 scope 的原 SourceMedia URL，不新增私有对象访问能力。
- 服务端只允许 `html-v1` 非空图片引用；每个 descriptionImages URL 必须实际出现在当前描述 img.src 中，不允许用此字段存无关图库。描述内图片必须逐字属于 **固定来源公共/全部SKU图片集合 ∪ 此版本 descriptionImages**。未声明的池外图片、危险标签/URL/属性仍拒绝，不静默净化后保存。
- 描述删除插图后 UI 同步移除不再引用的 descriptionImages；对象/容量事实不删除。引用列表顺序与内容按原 JSON 精确保存/hash，不做“等价载荷”重放。
- html-v1 描述32768字节、整个 CompletionDraft Save64KiB不变，URL/列表均计入。过限在本地及服务端拒绝，不能先保存半个版本。
- 商品不新增 descriptionImages 字段，仍由原商品图片池提供全部描述图片。候选新增图只保存在候选 Content，不向固定 Product 或其他草稿同步。

## 权限与上传失败

查看/保存草稿继续实时 `workbench.supply.read/manage`、原 actor/member/org 与固定出处证明。历史读取不因新增图要求上传权限，也不远程读取/计量图片。

二进制上传继续调用原 Collection media API，仍要求原 **workbench.collection.manage** 及 SourceMedia/容量配置；草稿页面还需已获原 supply read/manage 且准确来源读取成功。没有上传权限只关闭上传控件，不伪造成功、不扩角色/grant，也不阻止原本可用的草稿文字编辑。公共图片链接不是权限凭证；渲染继续原 `/api/image-proxy` 的 TLS、URL、网络和试用限制。

原3MiB/JPEG/PNG/像素限制、org/actor/hash/bytes 意图、上传前容量准入、不可变PUT+内容读回+容量确认、GET核实不清pending、同文件重放确认全部不变。成功确认之前不加入池/不插入描述；失败或 UNKNOWN 不生成成功占位图。busy/pending 阻止保存、刷新或离开丢失关联；身份/企业切换仍用原 guard/abort，迟到响应不能写其他表单。配置不可用如实报错，不能回落为未计量上传。

SourceMedia 对象与 Collection/Completion Save 是原有分离边界：上传成功不等于草稿保存成功。保存失败/CAS冲突/响应丢失保留本地内容与原意图；原 key/完整 Content 核实重放，不能换键/转换格式。上传成功未保存的对象保留且计量，不新建补偿或清理。

## 历史、恢复、结构化保存与平台

- descriptionImages 与描述/格式共同进入原 immutable record/head/command/hash；相关读/列表/精确历史/回执严格 DTO 完整携带，不从最新 Product 或当前 head 拼旧版。
- 旧版详情和对比安全预览准确旧描述，额外图片引用参与差异展示；来源图片仍来自准确固定 Vn。显示图片走原代理，任何历史原文都不执行。
- **恢复仍只恢复完整标题/描述**：旧版 title、description、descriptionFormat 及其 descriptionImages 一起替换为新 Content 的描述部分；保留当前 structured 的属性/价格/库存，固定来源/SKU/原商品图片不变。描述插图是完整描述资源，不恢复 Product/SKU 图库。旧版无 descriptionImages 时清除此描述字段；对象/容量不删除。页面明确显示将恢复的描述和插图，沿当前 head CAS 追加版本，不改旧版。
- 标题/结构化编辑保留当前 description、format 和 descriptionImages；相同字段/恢复就绪检查、回执准确内容比较覆盖新字段，避免静默丢图。
- 现有平台/SHEIN 仍只使用确定性纯文字投影：不复制 img/新增图片列表、不下载或批准图片、不调用平台、不发布。候选原富文本和引用保留。

## 验证与准入

新公共内容字段及恢复资源属于本轮新事实合同，使用 Independent Architecture。官方组件依赖预计约85文件，其他改动限当前 Content/DTO/UI/测试；文件数不拆PR、不隐藏阈值。沿一个 Writer、原PR #746；独立高风险合同检查达到 IMPLEMENTATION_READY 后编码，完整用户路径形成再最终增量检查。

TDD：Go/TS 字段缺省与旧hash、图片并集/实际引用/危险及过限、不可变版本/原CAS/同键不同资源、完整描述恢复但structured不变；真实组件直接上传成功后才入池/插入、UNKNOWN不清、无权限和只读、不改固定来源、迟到scope隔离；原历史/结构化/平台文字投影回归。复用原SourceMedia权限/容量/同文件恢复和隔离PG测试，不造新runner/验收平台。

必要 typecheck/lint/build/稳定候选CI；独立本地前端页面验证深浅色、菜单/链接/图片/窄屏，隔离数据仅用fixture。不更新 own675、不启动/替换保留实例或调用真实平台。自检/CI/独立复核不等于用户验收。

Legacy decision: N/A。当前 Product/SourceMedia/Completion/Target owner 复用；无 RETIRE 依赖、fallback、双写、迁移或第二图片/容量事实源。
