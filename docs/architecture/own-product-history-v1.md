# 自有商品历史读取、旧版详情和对比 v1

Execution Issue: [#675](https://github.com/qq550723504/task-processor/issues/675)。Design Basis: **Independent Architecture**。Admission Status: **REVIEW_PENDING**。本文先于正式实现。

## 产品结果与依据

2026-10-14 用户明确“先明确历史读取和权限合同，再做版本历史、旧版详情和差异对比，继续用独立页面”。当前企业操作人从自有商品详情查看手工保存历史，打开任一旧版完整资料，选择两个保存版本查看变化。沿 [Final UI/IA](../product/final-ui-ia-authority.md)、原 [独立录入](own-product-entry-v1.md) 引用的 Figma `tg48P46SSXl6TBy9lZwg63 / 31:463` 和 Console 组件；本次用户明确独立页面，不伪称已有历史 Figma frame，不增加顶层中心。

Must：历史列表按真实手工保存版本倒序分页、真实保存时间；旧版不可变资料、图片/SKU/0值准确展示；两版本基础资料、通用属性、公共图片及稳定 source_id 对齐的 SKU 增删/修改可比较；实时原 actor-private 权限、企业隔离、同商品绑定；深浅主题；保留旧草稿引用和现有资料。

本决定替代编辑合同中的“历史列表 UI Out of Scope”。Out of Scope：回滚/覆盖/恢复旧版、编辑旧版、草稿自动升级、AI 衍生版本历史/任意 Catalog 浏览、其它来源商品、平台/provider/资金/外部副作用、新 schema/迁移/清理、通用 diff 框架或验收工具。没有新增 Accepted Risk。

## 归属与历史集合

沿 [编辑合同](own-product-edit-v1.md)：Collection 持原 actor 的 item/productKey/当前来源引用；Sourcing 持每次 `own_product/v1` 发布的不可变声明和回执/保存时间；Catalog 持准确不可变快照，版本号允许因 Review 衍生快照而跳号。历史不是 `1..head`，也不是 item.revision。

手工历史集合限定为：从当前已授权、活跃、kind=own 的 Collection item 推导 org、actor、productKey、当前 source.version；仅该 org+原 actor+productKey 的 `own_product/v1` Source 发布，Catalog version <= 当前 source.version。当前唯一 own writer 每次声明与该条目当前引用/CAS在同一 Product UoW 提交，故此集合是该商品已保存的手工资料；不扫描 Catalog head，不纳入其它 producer、actor、商品或比当前引用更新的发布。既有 previous_publication_id/previous_catalog_version 是不可变修订依据，首次创建 expectedBase=0；修订须有规范的更早基准。读取逐个经现有 Source read/receipt/Catalog binding 校验，拒绝不完整/损坏证据，不返回原 envelope、checksum、raw reference 或内部身份。

不新增历史表、第二事实源、版本算法、事件、状态机或后台 owner。Sourcing persistence 提供有界只读引用查询，沿现有 Source reader 校验真实回执/快照；Collection service 是对外 item 权限和投影 owner。

## HTTP 与调用合同

GET `/collections/items/:item_id/history?limit=20&before=<version>`：limit 1..20，before 可选规范正十进制字符串 <=MaxInt64，表示严格早于该版本；返回 current item、至多 limit 个 `{version,publicationId,savedAt,title}`，有后页才返回 nextBefore。before 是筛选边界，不是授权证据，不接受客户端 org/actor/productKey/publication/head。没有伪造 total。

GET `/collections/items/:item_id/history/:version`：version 同上，只允许在上述手工历史集合中的准确 Source 发布，返回 current item、selected version 元数据、该发布已验证的 product。未来版本、衍生版本、其它商品/actor/org 返回 NOT_FOUND；格式/重复/未知 query、GET body/Idempotency-Key 拒绝。两 GET 沿 verified identity/live membership、`workbench.collection.read`、no-store、现 10s Collection timeout、单响应 2MiB；历史引用查询限制 limit+1，详细内容沿现 Source/Catalog 2MiB 单快照边界。

`route descriptor → 原 BFF allowlist/严格校验 → Collection.Service → 原 actor-scoped Repository.ReadItem → 新 Sourcing 只读 history reader → 原 Source 内部读取与 Catalog binding → 对外 projection`。生产构造器显式注入 history reader，无可选 legacy fallback。完成读取后再次实时授权并复读活跃 item/相同 productKey；撤权/换企业/归档不交付已读取资料。移动分组不撤销原 actor 商品读取权；编辑并发只改变当前引用，已选不可变旧版保持准确且不升级为新版。

全程只读，沿原 read-only/repeatable-read Source 事务；不获得写权限。失败/超时仅明确错误和显式重读，无 UNKNOWN mutation、新命令键或自动恢复协议。索引分页使用版本字符串边界，不依赖 JS number；一个浏览过程新增保存不会插入正在看的旧页，返回首屏刷新显示新版本。

## 页面与对比

独立路径：`/workbench/data/mine/own/<itemId>/history`、`.../history/<version>`、`.../history/compare?from=<version>&to=<version>`。详情增加“版本历史”；历史列出保存版本/时间/名称/当前标记，支持选择两版；旧版页复用现详情图库/属性/SKU展示，明确历史版本及真实 savedAt，只提供历史/当前资料导航；当前编辑入口仍位于当前详情。

对比页分别读取两个已授权不可变详情，避免组合两个最大快照超单响应预算；同 org/actor/item 挂载、取消旧请求和切企业 guard 复用当前 Console 模式。两版身份与 productKey 严格相同后才展示。基础文本逐字段、属性按 name、公共图按有序 URL、SKU 按 source_id（SKU编码可改）比较 title/sku/规格/币种/价格/库存/有序图片。显式0与缺失分开；own 当前快照库存省略0沿已有声明合同准确显示0。新增/删除SKU分别显示，属性重排不误报，图片顺序变化显示；不依赖未读取的 Catalog head，不合成“恢复版本”按钮。无新依赖，复用当前 React/Zod/表格，局部确定性对比适合现模型，无通用 diff 系统。

## 验证与交付

TDD：own 保存v1/衍生v2/ownv3只能列v3/v1，准确时间和旧资料；分页过滤；拒未来/衍生/其它商品和actor/org/无read权/撤权/归档，边界输入/有损坏证据；真实生产注入。前端 route/proxy/API严格绑定、0与稳定ID增删/修改对比、只读独立链接/切scope取消。复用现 fresh PG、Vitest/typecheck/ESLint/必要 CI；最终一个独立交付检查。

一个主要历史 PR 堆叠在编辑#687，原#683/#687保留，不代为合并。单Writer本会话，原附属worktree；开工同步main111bd（已包含），稳定候选再同步实际main一次。授权本地保留实例使用原卷/owner升级试用，不初始化/迁移/删数据/清理；运行交接和截图提供真实入口，开发检查不代签用户/生产验收。合并/关单/共享或生产/provider仍未授权。

Legacy decision: N/A，复用当前 owner，无 RETIRE abstraction、旧映射/双写/迁移。
