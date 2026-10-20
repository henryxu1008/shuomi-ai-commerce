# 自有商品已有草稿与继续编辑 v1

Refs #699；2026-10-14用户批准“已有草稿查询、权限和版本展示合同；商品详情已有草稿列表与继续编辑；独立页面；返回商品和批次上下文；复用供应链/草稿”。Design Basis: Independent Architecture。Admission: IMPLEMENTATION_READY（原独立Reviewer R1通过，仅本次新的只读反查合同）。

## 1. 用户结果与authority

用户回到一件自有商品，查看该商品跨转入批次的已保存草稿，区分固定来源Vn与当前保存Rn，直接继续原草稿。保留“创建待补全草稿”作为显式新Transfer，不自动猜选/复制/创建草稿。修复#694返回所选转入批次的Backlog。

批准基线： [未绑定草稿](supply-unbound-completion-drafts.md)（IMPLEMENTATION_READY）、[单件转入](own-product-draft-entry-v1.md)、[自有商品编辑](own-product-edit-v1.md)，已合并#697；[当前产品决定](../product/supply-unbound-completion-drafts.md)及本次用户决定。UI/IA沿[Final UI authority](../product/final-ui-ia-authority.md)，Figma tg48P46SSXl6TBy9lZwg63/31:463，自有库3500:1100和待补全2667:1948，沿原商品详情/独立草稿页，不新增一级菜单或假frame。用户独立页决定高于原弹窗样例。

本轮仅新的有界读合同及消费者。Out of Scope：schema/index/grants、新fact owner/状态机/事务/恢复协议、候选字段/草稿历史record接口、平台规则/店铺/AI/provider/发布、自动转入/保存/升级/最新猜选、migration/legacy兼容、验证平台、未授权部署/真实数据操作。

## 2. owner与读合同

Collection拥有当前item与data owner权限；Preparation拥有不可变来源与转入批次；CompletionDraft拥有已保存head/record和候选文字。复用源表现有CollectionItemID外键关系，不按ProductKey/当前批次/当前Catalog latest猜关联，不建反向映射或第二事实源。

CompletionDraft新增ListForCollectionItem(ctx,itemID,q)及对应Repository只读method。Service的Collection owner依赖明确包含现有Collection.ReadItem（无optional fallback）。DI仍由composeCompletionDrafts注入原collection.Service和原repository。链路：contract → domain Service → actor/member-qualified SQL heads+records+retained sources → 原HTTP/BFF → 商品详情列表 → 原sourceID独立Read/Save。

HTTP：GET /api/v1/workbench/supply-preparations/collection-items/:item_id/completion-drafts。只接受UUID item_id和原limit/after/keyword；不接受scope、member、store、productKey、版本或任意return URL。返回ItemPage {collectionItemId,items:Record[],total,nextCursor?}；Record复用原结构。客户端和BFF核对echo itemID及每条Source.CollectionItemID/own kind，不能仅检查JSON形状。

只列已明确保存的最新head：一条Source最多一个head，多次新Transfer形成多份独立Source/草稿均分别列出。未保存Preparation不冒充已保存。total是该scope/item/keyword匹配head实际数，空items数组来自真实查询，不显示假0或假草稿。按稳定sourceID升序keyset，不承诺跨分页并发保存的全局快照；显式刷新读取当前事实。

q复用collection.Query：limit默认20、最多100、after为UUID、keyword最多80字节及原转义。SQL count与page同时限制org/actor/member、s.collection_item_id与s.source_kind=own，joins三表各自同scope。复用原r.value校验content hash/draft identity、每条Record.Source与SQL同scope retained Source完全一致及2MiB预算缩页；nextCursor取最后已返回sourceID，不跳过未返回项。不得逐批遍历，也不新增每行IAM/平台调用；只做常数次当前权限/当前item检查。数量和资料均不跨scope泄漏。

## 3. 权限、scope与版本

沿原verified effective identity/live supply.read + data read/owner，org/actor/member从当前identity和authorize proof取得。新的商品反查先用当前Collection.ReadItem核对item存在、ID相同且kind=own，再查询该scope；返回前再次核对live scope和当前item仍可读且own。编辑/移动当前商品允许查询同稳定item的旧草稿，不能强制Source版本等于当前item版本。item自身归档/失去当前item访问时该详情反查拒绝；父批次归档按原ReadItem访问语义，不新增批次权限门禁，已有Source的直接读取仍沿原retained-source合同，不能改变旧read权限。

read-only可查看列表/独立草稿，保存仍须原supply.manage；导航按钮不能授权。撤权、token过期、企业/成员变更、deadline失败、安全字段或retained source错配 fail closed。保持原DOMAIN 10s/HTTP20s、no-store、GET拒body/content encoding/idempotency、严格query、2MiB读取预算；无外部URL下载或provider IO。

来源版本显示Record.Source.Source.Version（固定Vn）；保存版本显示实际head Record.Revision（Rn），保存时间为该record.CreatedAt。原商品当前版本可单独显示，旧来源不是“自动过期/不可用”。继续编辑只传原sourceID，原编辑器重新读实际source+最新head；列表显示R1后他人保存R2，打开时显示实际R2并沿原revision CAS保存，不用旧列表Rn覆盖。列表只展示current head，本轮不开放旧record详情/差异。

## 4. 独立页面与导航上下文

商品详情增加已有待补全草稿区域：候选标题、来源Vn、保存Rn、保存时间、转入批次稳定ID及查看/继续编辑。模块未启用不展示；缺查看权限、读取失败、空结果、分页/刷新分别准确显示，加载不冒充真实数量。当前user/org/item变更remount并abort，迟到响应不能串scope。

独立页沿 /workbench/supply/mine/missing/:sourceId。仅新增可选UUID query preparation、item；重复/非法/未知导航query拒绝，不接受完整return URL。商品入口携带每条实际PreparationID和itemID；列表入口至少携带选中PreparationID；单件转入成功入口由实际固定source生成相同上下文。query不作为数据或写权限，只在原ReadSource后核对：preparation匹配Source.PreparationID，item匹配Source.CollectionItemID且kind=own；错配不得预填或保存。

返回商品资料使用固定本地路径 /workbench/data/mine/own/:item；返回待补全列表保留 ?preparation=:preparation。两种入口仍使用同一独立编辑器/实际保存回读。已有不带query的source入口及原missing?preparation=&source=入口继续用同一现有合同；这是当前公开入口的原语义，不引入legacy owner兼容或双读。

## 5. 持久化、失败与验收

新接口全只读，没有持久化步骤、外部副作用、幂等key或新恢复状态；仍在同Product database原五表读权限内。保存继续原head/record/command事务、CAS、原key同payload、UNKNOWN核实和live source proof；导航上下文不写入record，也不绑定或升级source。query失败只重读，不触发Transfer/Save。

Legacy decision: N/A，复用当前合格owners、SQL/keyset/预算/Console，不依赖RETIRE abstraction或兼容机制。schema、grant、身份、SourceMedia命名空间、现有数据/图片/容量/卷/worker保持。运行实例更新/真实数据操作、合并、关闭Issue另需具体授权；本轮源码与受控PG测试不代签用户/生产验收。

TDD/开发验证：新query多preparation/同item不同Vn、最新Rn、未保存排除、分页keyword/response budget、other item/org/actor/member隔离、当前item不可读或非own拒绝、retained source错配和权限复核；复用fresh PG真实Source/CompletionDraft保存测试，不建立新验证工具。HTTP/BFF严格输入和响应归属；UI独立继续编辑/源V1与当前V2区分、读权限与迟到scope、返回商品/原批次和导航错配不保存；原保存/UNKNOWN路径适用回归、typecheck、必需CI及最终独立检查。

## 6. 独立边界审查

待原独立Reviewer只针对上述新公共只读query、scope/归属和消费者合同检查；无新的Must扩展或全局架构重审。达到IMPLEMENTATION_READY后冻结，以最多两轮正常审查及finding分类规则执行；非Blocker实现/边界验证收敛，只有真实Blocker重开设计。
2026-10-15 R1：own_history_review实查#699正文和实际owner/SQL/BFF/独立页，绑定先于生产的设计fac7e91a3a6b36f89562ace9b85ec6f77498be89，结论IMPLEMENTATION_READY，无当前Must/AGENTS BLOCKER。以下IMPLEMENTATION_TEST在实现收敛：scope-qualified count/page与retained source/echo校验；常数次首尾live权限/当前item检查；跨Preparation固定Vn与实际最新Rn、budget/keyset；sourceID不变但导航UUID改变时将query纳入editor remount key，核对前不保留旧可保存表单。item自身归档按原ReadItem语义，父batch归档不扩新权限模型。最终集中复核实际增量。
