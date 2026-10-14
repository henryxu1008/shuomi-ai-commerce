# 自有商品版本更新与独立编辑页 v1

Execution Issue: [#675](https://github.com/qq550723504/task-processor/issues/675)。Design Basis: **Independent Architecture**。Admission Status: **NOT_READY / REVIEW_PENDING**；本文件先于正式实现。

## 用户结果、依据和范围

2026-10-14 用户明确：“可以，先明确版本更新合同，再做已保存商品的编辑功能，继续用独立页面。”当前企业操作人在自有商品详情进入独立编辑页，预填已保存的准确资料，修改名称/品牌/描述、通用属性、商品图片和SKU资料，保存后返回同一商品详情并刷新回看新版本。

Product/Figma Authority沿[Final UI/IA](../product/final-ui-ia-authority.md)、[原独立录入](own-product-entry-v1.md)所引用的Figma `tg48P46SSXl6TBy9lZwg63 / 31:463`、自有库3500:1100/添加3502:405与当前Console组件。用户本轮明确独立编辑页面，路径 `/workbench/data/mine/own/<itemId>/edit`；不新建顶层商品中心，不伪称已有编辑Figma frame，不使用模态框。

本轮替代旧阶段“已保存资料编辑Out of Scope”，原版本不可变、安全和恢复要求继续有效。新功能可独立产生用户价值，一个主要编辑PR；图片#683未合并，编辑分支基于其准确已复核HEAD并同步开工main；可用堆叠PR只展示本轮diff，不代为合并#683。

Must：完整预填与独立编辑保存；同一itemId/productKey、batch、商品数量保持；追加不可变新快照；旧快照/已创建草稿固定引用保留；并发/移动/归档冲突拒绝；原键幂等及UNKNOWN；当前企业/actor权限；未经修改的数据准确保留（含0价格/库存、稳定variant sourceId、通用图与SKU图分别保存、既有多张SKU图和自定义SKU标题）。

Out of Scope：采集/市场等其它来源编辑；草稿自动升级、AI建议/图片批准/平台发布；历史列表/回滚UI；库存实时同步、资金、真实provider；新schema/数据迁移、对象删除/清理、新恢复或验收平台、合并/关单/共享或生产部署。没有新增Accepted Risk。

## 唯一owner与版本语义

复用[Source/Catalog/Collection原子合同](my-supply-chain-shein-v1.md)和当前版本仓储：Sourcing持每次用户声明的不可变证据/发布回执；Catalog持同一 `(org,productKey)` 的不可变快照序列和head；Collection持actor-private条目、当前原始来源引用、组织分组与命令回执；既有preparation/target/draft各自持创建时固定的source publication/version。编辑不是AI Review.Apply，不承接已批准资产。

区分三个数：`item.revision`是Collection条目CAS（移动/归档/编辑都会增加）；`item.source.version`是当前用户资料指向的Catalog版本；Catalog head也可包含已有Review产生的衍生快照，因此资料版本可能跳号，不能把head当成用户当前原始资料，也不能要求版本号等于item revision。

保存追加新Source publication与Catalog快照，随后在同一事务把该own条目的publication/version更新为新回执、revision+1；productKey、itemId、batchId、createdAt不变，batch revision+1，数量不增。历史证据和Catalog快照不UPDATE/DELETE。详情“保存时间”当前实际投影createdAt，应改为“创建时间”；版本显示真实source.version。

现preparation.SourceSelector先用Collection.AuthorizeOwner确认实时actor/企业权限，再加载其自身RetainedSource和准确Catalog版本，未把读取已建草稿绑到Collection当前版本。这允许保留既有草稿/适配/资产引用，编辑不修改这些owner。新发起批次选择仍检查batch/item修订与原始publication/version；编辑后的过期选择报冲突，不静默选新版。用已有固定选择/retained-source测试收敛，不扩建草稿迁移。

## 命令、调用和事务

原 `POST /collections/commands`增加严格action `revise_product`，载荷只含：`itemId`、`expectedRevision`、规范正十进制字符串`expectedSourceVersion`（<=MaxInt64）、`expectedPublicationId`、完整`product: OwnProduct`。客户端不指定productKey/新版本/新publication/batch。原HTTP未知/重复字段与price/stock缺失/null拒绝保持，直接domain调用同样校验形状和完整OwnProduct预算。

`collectionCommandSchema → 原BFF/HTTP → Collection.Service.Mutate → 原Collection.Repository.Execute → 新own修订publisher → 原Sourcing.InternalProducer/TransactionWriter → Catalog原PublishSnapshot → Collection当前引用/CAS + 命令回执 → 原ReadItem/exact reader → 独立编辑页/详情`。

Service先授权/验证和生成稳定操作id；事务内从已授权item取得目标原引用，仅own且item/batch活跃、三项expected与锁定行一致。复用现batch-before-item锁顺序：先定位batch，锁该batch，再锁item并复核归属及revision；与移动/归档/批次选择一致。Publisher接口独立增加ReviseOwn方法，复用原注入工厂，不包装legacy service。原create/import路径与其ExpectedBaseVersion=0保持。

修订publicationId沿原scope+commandKey生成；ProductKey只来自锁定的own条目。OwnEnvelope用本次operation作为新的用户证据身份，稳定variant sourceId保留；raw reference metadata记录前一publication/version以说明本次声明依据。新图片仍是候选，既有公开图/变体图不被重复上传、批准或删除。

修订传给Catalog的ExpectedBaseVersion为nil（原合同的原子追加），不是要求旧原始版本等于Catalog head；Catalog既有head行锁按实际序列追加，保证已有衍生版本之后仍能编辑原始用户资料。完整新快照来自编辑输入，不继承未选择的AI衍生内容；用户编辑CAS由已锁Collection三项expected提供。单个编辑与AI追加各自被Catalog锁序列化，旧引用不变。不读head猜新版本、不实现第二套版本算法。

所有Source证据、Catalog新版本、Collection引用/revision及回执继续共同提交于原Product DB UoW。任一错误全部回滚；无跨DB写、schema/授权改变、外部provider副作用、新状态机或后台恢复。用原repository afterCommit seam捕获响应丢失，不建新runner。

## 幂等、冲突、授权与恢复

现scope+commandKey事务锁/输入hash先于条目CAS：已提交同键同载荷直接重放原回执，即使条目后来又更新；同键不同载荷冲突。两个不同键同expected只能一方提交，另一方REVISION_CONFLICT；被移动/归档/来源版本变化同样拒绝。不可在冲突/UNKNOWN后生成新键静默重试。

失败/超时/取消在commit前回滚；commit开始或响应丢失沿原OUTCOME_UNKNOWN，先GET原by-key，NOT_FOUND时原键原载荷显式重试；重启由原持久回执核实。已保存后再次读取当前详情可能已有更晚编辑，显示当前真实资料；不把重放回执当成覆盖当前资料的许可。

权限沿原verified org/actor/member和collection manage/read，publisher再次确认实时scope；跨actor/org、非own、归档记录拒绝，不能用客户端productKey读/写Catalog。旧版本仅由已授权固定引用消费者读取；本轮不暴露任意Catalog版本访问API。页面按actor+org+item重新挂载，切企业/身份不带输入；pending冻结编辑，保留原键和完整载荷，foreign pending保持原有阻塞与处理入口。

## 页面和无损消费

复用现OwnProductEntry表单为共享录入/编辑表单，不复制整页；新增保持行为。编辑入口仅在详情已成功读取own且具manage权限时显示；未读到准确资料前不挂可保存空表单。编辑保存按钮“保存新版本”，页内显示基于资料版本，成功导航同item详情；取消返回同item，沿原dirty确认/组织flight guard。

预填从ReadItem当前准确snapshot构造OwnProduct，稳定variant source_id/原标题、属性、所有公共及SKU图片、数值声明0均保留；缺失必需SKU数值/身份或不可无损表达的当前own资料显示明确读取错误，不猜0/币种/新ID或静默删字段。多SKU规格维度从已有属性推导；原删去组合保持，不自动重新生成笛卡尔积；单个已有SKU即使有销售属性也保留。

现表格选择一张SKU图片。为避免丢失原多图，模型保留每行原图片组与原标题；当前选择不变时全部保留，既有SKU专用图可作为原图选项，不混入公共池。选择不同图片时明确提示替换该SKU图片组；用户主动从公共池删除图时才同步移除对应SKU引用。新生成组合用原稳定规则分配新ID，未变组合保留全部已填资料。

## 验证与交付

TDD先捕获：修订严格形状及权限；v1→新版本、同item/key/数量、旧快照读取；已有衍生head仍能追加用户新版；并发同基准仅一次、归档/移动冲突、同键重放/异载荷；事务中断回滚和commit-response UNKNOWN原回执；已有retained source/draft继续准确读v1。复用原collection/productsourcing PG测试、固定选择与Source/Catalog owner测试，无新测试平台。

前端先捕获准确预填/0/稳定ID/多SKU图与自定义标题、未生成新组合不补删行、独立链接、保存新版本载荷、dirty/foreign/pending/UNKNOWN恢复、冲突不覆盖、失败不导航。适用类型/ESLint、稳定候选CI及独立最终复核；不重复未变化图片/容量代码的全套证据。

运行交接使用授权范围内的本地可操作预览；若升级原保留独立试用实例则仅当前原owner/配置/命名卷，保留数据和图片，不跑初始化或schema。需要超出既有实例准备权限的真实操作另行说明。设计与实现不代签用户/生产验收。

Legacy decision: **N/A**：当前Owner与原版本算法，无RETIRE引用、旧ID mapping/迁移/fallback/双读写。合并、关单、共享/生产、真实provider与数据清理未授权。
