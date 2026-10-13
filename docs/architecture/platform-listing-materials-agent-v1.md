# 平台上架资料智能体 v1

Refs #135 / #129 / Roadmap #137。Design Basis: **Independent Architecture**。
Admission: **DESIGN_IN_PROGRESS / NOT_READY**。本文先于正式实现；独立准入达到 IMPLEMENTATION_READY 后再编码。

## 用户结果与依据

用户于2026-10-13批准 [业务智能体职责](../product/business-agent-responsibilities.md)：商品内容与本地化之后，逐个推进平台上架资料、POD、选品、经营诊断。本结果消费准确标准商品，依据实际平台规则准备类目、属性、SKU和图片资料，显示可核对的缺项，保存原 Listing owner 的本地草稿。标题/描述/翻译的标准商品修改属于内容能力。

当前已经准入的正式平台实现是 SHEIN 美国站；未来平台消费各自已批准 owner。本设计不复制 #31/#655 当前 TEMU 实现，不创建另一个 SHEIN SDK、Store 数据源或跨平台规则引擎。不自动开放未准入平台。

依据现 `my-supply-chain-shein-v1`、`supply-unbound-completion-drafts`、`enterprise-agent-permission-modes-v1`、`organization-agent-configuration-v1`、`product-agent-runtime-contract`、`product-content-localization-v1`。#662最终92ea808d7已完成必要CI/独立复核及源码交接；合并/安装/真实模型/用户验收仍是分开的门槛。

当前Figma Authority为 `tg48P46SSXl6TBy9lZwg63` 非归档31:463；本轮已读取4703:429我的智能体与3507:3143待适配的design context和截图。配置4703:892、确认4722:429、待补全2667:1948沿现批准页面与组件。只添加实际注册能力和当前事实，示例平台、数量、成功和POD卡片不能变成已可用状态。

Out of Scope：平台发布/改价/下架/删除/增额度、真实provider和付费模型调用、原实例安装、旧数据迁移/回填、新队列/工作流/Runner/通用编排、通用结果平台、第二套审核或草稿事实。仅允许现配置owner新增本注册schema的有界约束和显式安装入口；不新建业务表，不在服务启动时DDL，不操作现有实例。

## 当前边界及调用链

现 `agent.Action/State/Validator` 的候选明确绑定 `product/enrichment.Candidate`。类目/属性/SKU不能伪装成 Catalog 内容补丁或进入 Product Review 的事实采用。这是本次复用必须处理的边界，不能直接复用标题 DTO 来表示完整平台资料。

| 事实/职责 | 当前唯一 owner |
| --- | --- |
| 标准商品、来源与采用后的准确版本 | Catalog / Review / SourcePublication |
| actor/member限定的来源选择 | Collection / Listing Preparation SourceSelector |
| Store、有效业务访问及官方配置 | Store Center；凭据仅server-only |
| 规则与平台确定性准备/校验 | Marketplace SHEIN goods / 当前 RuleReader |
| 已批准图片、inventory版本/哈希 | Asset owner |
| 模式、启用/模板/快照 | agentconfig原owner |
| 有界运行、请求键、预算、模型账本、checkpoint | 现agent/Eino、RunDB、AICapability/Commercial |
| 不可变本地平台资料及command/CAS回执 | Listing `record/target.TargetService` |
| 未绑定Store的候选文本 | 现CompletionDraft；不保存伪造的平台字段 |

正常Console/BFF → verified identity与当前enterprise/module权限 → 注册平台资料Agent配置/确认 → 当前source/Target/Product/Asset/Store/Rules窄port → 同一Eino有界读取/计算 → 领域校验 → 现Listing Target本地保存/按原command读取 → UI真实记录、版本、缺项与回执。运行层不成为商品或草稿事实owner。

复用 `Application.Source/QueryRules/ReadTarget`、`TargetService.Create/CreateForExecution`、`TargetRepository.ReadTargetCommand`、`EffectiveTargetProductReader.ReadEffectiveTargetProduct`，以及实际approved inventory/rule读取。正式当前来源不调用 `internal/workspace`、`internal/publishing` 或旧ListingKit服务。

## 冻结的输入与有界领域结果合同

继续现Eino/RunDB生命周期，不另建Listing Runtime。v1从供应链当前可编辑的**已保存本地Target**使用；新商品先沿现适配/补全操作选Store并建立本地草稿，缺项草稿也是合法起点。此操作不调用上架，也不借Agent准备偷偷创建上下文。原#135输入SHEIN draft的范围保持，read_only可在已有草稿上读取/计算。

注册 `listing.materials.agent/v1.0.0`，固定 `listing-materials-policy-v1` / `listing-materials-prompt-v1` / `listing-materials-action-v1` / `listing-materials-config-v1`。模板只有现名称/目标SHEIN配置，不开放任意工具、JSON schema或提示词执行。没有Knowledge消费需求，不借Knowledge持久化平台上下文。

`agent.Binding.ContextKind="listing-target-record"`，ContextID是不可变TargetRecord.ID；ProductKey/CatalogVersion/PublicationID/TargetPlatform由SourceSelector、EffectiveTargetProductReader和该record校验后铸造。Request仍是现可比较struct，无JSON输入扩展。组织/actor/member来自当前授权，Store/source/目标revision来自原record，浏览器只选择record、模板、模式revision和请求UUID。开始前必须仍是当前head；已有运行/命令的原key查询不换head或绑定。

只读历史授权与新执行授权分开：history/command read读取本人/member限定的不可变record及原运行，不要求它仍是head或Agent仍启用；新Start/resume/模型dispatch和新adoption才检查current head/config/mode。保存成功使head前进后，原baseRecord/run/receipt仍可准确核实，不能因旧head失效把成功隐藏成UNKNOWN。两者均须当前live身份/企业业务读取授权；撤销读取权限不泄露原数据。

仅为本消费者增加以下可选数据和窄port，不改变已有商品字段或生命周期：

```go
// agent package: bounded transport for a domain-owned proposal, never a fact.
type DomainProposal struct { Kind string; Payload json.RawMessage }
// Optional *DomainProposal on Action and State, omitted when absent.
type DomainValidator interface {
    ValidateDomain(context.Context, Request, DomainProposal, []Observation) (Validation, error)
}
// Eino Config: DomainProposalKind string; DomainValidator agent.DomainValidator.
// Existing Validator path remains the exclusive Product candidate path.
```

配置时只允许原Validator或DomainValidator二选一；Listing定义固定kind=`listing-materials-v1`。propose结果只允许对应一个候选；同时带Product Changes或错误kind立即拒绝，不由模型选择validator或fact owner。当前商品Action/State未出现新字段，历史解码/原键/快照含义不变。领域候选hash由runtime算完整JSON，不相信模型hash；64KiB模型输出、2MiB状态、现步骤/修复/期限上限和checkpoint拒绝行为共用。Listing v1不消费模型Confidence，必须为空；不得将空confidence伪装成高质量或批准。

`internal/listing/materials`拥有严格Payload类型、whitelist patch和确定性验证；只包含候选类目、品牌、属性、SKC/SKU资料及image slot中的批准AssetID和来源证据引用。source/Store/版本/金额/库存/远端URL/平台回执不作为可写输出字段。完整SHEIN DTO不能直接作为未过滤Candidate采用：在原TargetInput.Draft上应用显式允许字段，标题/描述、价格/币种/库存仓库及数量、平台身份/货号等未授权商业事实保持。SKU排列/属性可建议，但原SKU身份和商业事实必须一一对应；不能新增或删除无商品依据的SKU。需要补价格/库存/规格或翻译时标实际缺项，并链接原补全/内容能力。

只读Commerce Tool `listing.materials.inspect/v1.0.0`（owner listing.materials，当前平台SHEIN）读取服务器绑定record、有效标准商品、当前规则和approved inventory，通过现注册/授权/审计Invoker返回有界证据。不让模型指定工具args/数据库/URL/凭据；actual RuleReader、Store产品handle和Asset port仍是唯一读取来源。Payload引用必须存在于此次成功、已披露observation；太大拒绝/显示未披露，不把遗漏当不存在。Listing模型使用现governed TextExecutor、ledger、quote、usage与模型路由治理，新增固定operation=`listing.materials.decision`，不改变计费或额度语义，也不借标题路由配置假装新能力ready。

读取必须校验base仍为当前head、source/effective Product及apply receipt准确、Store binding与record一致、inventory hash一致。规则由当前owner读；没有规则则dependency unavailable。现 `ReadTargetRules` 在缺category时返回真实categories而不是伪schema，这允许先建议类目。类目变化只允许真实披露的末级ID，不同时猜新类目属性；保存后下一次使用新record读取新类目规则再完善。类目不变时属性选项必须来自本次真实schema，linked rules与合法asset slots沿原确定性owner。类目/规则不足仍可产生安全的人工建议，但不能报告整份Ready或自动采用。

Validation.Valid表示候选结构、字段范围和证据足以**人工核对**，不等于平台完整。原goods.BuildOfficial的issues/blocked为实际缺项；自动采用额外要求同类目当前全部规则可确认、无unresolved/blocked、准确资产引用且原确定性构建满足本地采用策略。不存在现成规则的字段不发明默认值。

## 保存、键与恢复

运行claim/commit沿现(scope, context, key, fingerprint)和CAS。source/Target固定输入不能在恢复时自动切到最新资料。新模型调用按原预算/授权准入；响应丢失只以原key核实，已有/可能Claim后错误不宣布零Claim或换键重生成。

本地采用只调用原 `TargetService`；输入保持SourceID、StoreID、ExpectedRevision、EffectiveVersion、ApplyReceiptID及被允许的draft字段，原Target仍保存不可变record/head/command，沿原ProductDB事务。运行元数据不复制正文为“已保存草稿”。

采用命令使用 `collection.StableID(org, actor, "listing-materials-adopt", runID, candidateHash)`，由服务铸造而非浏览器指定；TargetInput为准确base的SourceID/StoreID/effectiveVersion/apply receipt、ExpectedRevision=base.Revision和已验证whitelist patch。采用API只收原run key/config ref、原baseRecordID和candidateHash及显式人工确认，不收修改后的candidate正文。实际候选只能从原RunDB读取。相同run相同候选复用同一个Target命令；变更候选必须沿原有界resume产生新hash，不覆盖旧receipt。

当前TargetService只重新授权来源/企业，没有Agent模式检查。这是实际需新增的安全接入点：增加限定Agent采用的 `CreateForAgent(ctx,key,input,guard)` 及tokenless对应入口，guard必须非nil；原人工Create语义不变。TargetPrepared私有保存commit检查闭包，在原Repository SaveTarget事务的CAS/回执重放之后、实际写入之前，以及现proof.Read的提交前检查点执行。检查闭包只由本受信composition构造，验证当前verified身份/原member、运行准确候选/配置启用/模板、原mode snapshot和policy；read_only或配置/模式漂移拒绝。沿原RunDB检查语义做最后提交检查，不虚构跨DB原子性或增加分布式锁协议。

Guard同时核对Target服务本次读到的实际merchant/Product/inventory/规则与已披露依据。同类目规则hash必须准确相符；更换类目仅允许已披露category catalog中的末级ID，catalog及Store binding须未变，新类目其他字段保持且新owner计算的真实缺项必须人工确认，不能自动采用。source/Product/Store/inventory变化拒绝，不自动切最新版本。Record原不变payload/hash/CAS和事务仍由Target owner负责；闭包只临时验证，不作为持久化第二事实。

RunDB和ProductDB不是一个分布式事务。Run到human_review_required之后仍保存建议，不增加adopted phase或第二receipt。响应通过确定性command key读取原Target command投影；Target成功而HTTP返回丢失时，读取原Target command得到准确记录，不再调用模型、不另建草稿、不补偿删除。先按当前read权限查已有准确回执，再考虑新写模式；切只读/模板归档后仍可读本人原历史与已提交回执，不能撤回已保存事实，也不因此写第二次。没有command回执且结果不明时保持UNKNOWN，同key核实；不得把not found直接当授权换key的证明。

Target响应/COMMIT结果不明返回UNKNOWN，UI保留原key、body、scope与版本。不可伪装失败、自动另起请求或把未保存建议显示为平台草稿。取消、member/权限撤销、重启沿原Run/Target约束处理，不能通过full绕过。

## 四权限与审核

共用 #656 的当前配置持久化、API、UI和执行准入，不映射原标题/内容配置，不自动启用新Agent。

- default：读取/计算建议，显示确切diff/缺项，用户明确审核后保存本地草稿。
- read_only：允许合格读取/计算与建议持久化；拒绝Target创建/更新、资产批准和任何采用入口，包括直接API/worker。
- auto_approve：只有当前显式策略与实际确定性校验允许的完整本地候选可以采用；有阻塞/未决/规则或证据不足时整项转人工，不静默部分采用。
- full_access：当前企业内已经授权业务工具开放；合格本地候选可由当前policy采用，证据/规则不足或更换类目仍需要显式核对，本地保存须原来源/Store/资产/规则/版本/模式与预算约束。publish/reprice/delist/delete/增额度和真实外部写动作不在该授权内。

现Product Review不能用作Listing字段的采用owner。用户确认/diff投影复用现界面与模式策略；实际采用由原Target服务审核输入和保存。不能把标题或内容的Review receipt冒充整份平台资料批准；标准商品内容引用仍须准确原Review/apply lineage。

## 未配置路径与界面

未绑定Store沿当前CompletionDraft编辑/保存候选文本，可查看原标准商品与准备依赖；新平台字段生成/正式校验必须当前真实Store及规则。缺Store/规则/approved inventory/模型配置显示具体缺项和已可操作的现入口；不为了Agent菜单伪造Store、正式SKU、图片、类别或平台成功。

沿现我的智能体卡片与独立配置/历史，供应链待适配/待补全通过原Target编辑器打开已保存草稿上的资料准备入口。缺配置不调用模型；确认绑定所选版本/Store/模板/四权限。结果显示建议/缺项、可保存原因、实际draft record/revision及读取原命令入口。跨企业切换阻止未知命令丢失，不清除历史。

API位于原workbench supply命名空间，以recordID限定 `materials-agent/prepare`、`runs`、准确key的run read/resume、`adopt`和`adoption`只读投影；BFF严格body/状态码/UNKNOWN语义共用当前模式。配置/列表仍走原agentconfig API，registry与UI只显示真实定义和readiness。正常RunDB/configuration、ProductDB Target、AssetDB、治理模型/ledger由当前application composition显式注入。

本v1使用现同步有界Eino入口；没有新后台job需要调度，不扩Preparation operation状态机/Temporal或现Title worker选项。tokenless采用adapter只能在明确存在的worker消费时开放，复用当前member执行授权与同一domain/mode guard，未接消费者不伪称worker已运行。现worker/volumes/原标题与内容执行路径保持，必要回归检查覆盖已有消费者。

当前来源、注册定义/模板和真实能力全部ready才显示可执行；非SHEIN已准入平台消费其owner，未准入项明确不可用。不得竞争已有TEMU Writer或同步其数据。

## Legacy 与验证

Legacy decision: **EXTRACT**。复用当前合法来源/规则/资产、Target创建/读回、agentconfig、治理模型与Eino行为；owner仍是本表当前包。旧Workspace、Task-first商品、旧独立平台Workbench、旧任务queue/API不消费、不包装、不恢复fallback。没有迁移/映射/双写或second source。

实现时TDD先捕获领域结果错误归属、来源/Store/规则/资产引用不匹配、read_only直接采用、配置撤销、同键不同载荷和原命令读回等当前风险；复用已有隔离PG/loopback和测试，不新建验收Runner/矩阵框架。

必要开发检查：原商品/content行为保持，平台读/计算/提案/实际本地保存/刷新读回，四模式API/UI/当前worker消费者，严格body/64KiB候选/原runtime bounds/timeout、必要编译/类型/CI。不调用真实IAM/付费模型/支付/SHEIN/SDS或访问业务数据。

高风险独立准入与最终交付检查按当前AGENTS各一次，修复只看相关增量。阶段交接提供正常入口/启动命令、实际保存owner、不可用原因及NOT_RUN。原数据/命名volumes/worker保留；未授权merge/安装/deploy/Issue关闭或保护规则变化。

## 准入检查点

此版合同已明确，尚未实施，仍待独立准入。检查当前新领域结果port、Target Agent提交guard及唯一command恢复责任；原商品候选/四权限/计费和已批准领域owner不做重复全局评审。具体happy path无法完成、错误授权或数据损坏分类BLOCKER；其余具体实现验证转IMPLEMENTATION_TEST，不追加通用平台。达到IMPLEMENTATION_READY后在同一主要分支连续TDD实现并维护一个主要PR，最终实际用户路径集中复核。
