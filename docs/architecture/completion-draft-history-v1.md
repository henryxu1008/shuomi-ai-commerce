# 待补全草稿保存历史与比较 v1

2026-10-17 格式增量见[自有商品富文本合同](own-product-rich-description-v1.md)：描述及格式共同保存/比较；草稿恢复完整旧描述格式并保留当前 structured，平台文字字段使用明确的纯文字投影。原权限、固定来源、CAS、幂等及发布边界不变。

结构化字段的历史和比较语义沿 [结构化候选合同](completion-draft-structured-v1.md)；原准确版本读取/只读权限/固定来源不变。

Refs #704；Design Basis: Independent Architecture。Admission: IMPLEMENTATION_READY，R1独立边界审查通过后冻结。

后续 #708 的[历史内容恢复](completion-draft-restore-v1.md)增加独立的普通 Save 消费者；本合同的历史接口、旧记录和详情继续只读，原历史阶段 Out of Scope 不禁止用户后续明确批准的独立新版本保存。

## 用户结果和依据

2026-10-15用户批准：先明确待补全草稿历史读取、权限和比较范围，再实现保存历史、旧版详情、差异对比；独立页面；保留返回草稿、商品和批次上下文；复用CompletionDraft保存记录。

操作人多次保存候选标题/描述后，可以查看准确Rn并比较同一草稿两个已保存版本。依据：[未绑定草稿产品决定](../product/supply-unbound-completion-drafts.md)、[原草稿合同](supply-unbound-completion-drafts.md)、[已有草稿和导航](own-product-existing-drafts-v1.md)、已合并#700/#697和本次用户决定。本增量取代原合同中“本轮不开放草稿历史record接口”的阶段范围，原保存和平台边界继续适用。

UI/IA沿[Final UI Authority](../product/final-ui-ia-authority.md)，Figma `tg48P46SSXl6TBy9lZwg63 / 31:463`，待补全`2667:1948`；用户明确独立页。沿当前Console、表格、主题tokens和原草稿入口，不新增一级菜单，不伪称已有历史frame。

Must：当前权限下同source已保存历史、准确旧版、同草稿两版本标题/描述比较、固定Vn与保存Rn分别表达、只读身份可查看、返回上下文、真实空/错/未开放状态。

Out of Scope：回滚/复制旧版/新草稿、跨source/跨草稿比较、来源升级、SKU/图片/平台字段编辑、Store/AI/provider/发布、新表/index/grants、新owner/写协议/恢复器、legacy兼容/迁移/清理、通用比较或验收平台。未保存输入不参与历史比较。

## Owner、调用链与读合同

CompletionDraft拥有现有不可变records和head；Preparation拥有固定source；来源资料仍由原Product owner拥有。原表唯一约束`(organization_id,actor_id,draft_id,revision)`可按revision倒序读取；不建第二版本存储或复制来源资料。

contract → completiondraft.Service/Repository新增ListHistory、ReadRevision → 现有PostgreSQL Repository → 原composeCompletionDrafts注入 → 原CompletionDraft HTTP模块和BFF/API → 独立历史/旧版/比较页面。内部Target消费的ReadVersion(recordID)不改为公开无约束ID读取。

- `GET /api/v1/workbench/supply-preparations/sources/:source_id/completion-draft/history?before=Rn&limit=20`。
- `GET /api/v1/workbench/supply-preparations/sources/:source_id/completion-draft/history/:revision`。

历史query只接受before和limit，拒重复/未知字段。before为正规范十进制revision（无前导零/符号/指数），上限Number.MAX_SAFE_INTEGER，与现有JSON整数合同一致；省略为首页。limit默认20、1..100。精确读取revision采用同一规则；不接受org/actor/member、draftID、recordID、来源Vn或完整return URL。旧版GET不接受query。

历史返回`{source:SourceItem,draftId,currentRevision,items:Record[],nextBefore?}`。items按revision严格递减，`revision < before`且不超过本次读取观察到的head revision；每条沿原Record结构，source/draft必须与page一致，revision不可重复。nextBefore为最后已返回revision，只在确有更早项/预算截页时给出。无保存head时返回真实空items、currentRevision=0，无nextBefore；不可读source仍拒绝，不能借空列表探测其他scope。不存在的精确revision返回404。

Repository先读取当前scope固定source与head，校验head记录和retained source；records的所有查询均同时限定org/actor/member、sourceID、确定draftID和revision上界，复用record.value的内容hash/identity验证并逐条核对固定source。currentRevision是本次查询观察到的head，不声称跨页全局快照；并发新增保存不会改变旧record，可刷新首页看新head。无COUNT、无需每行IAM请求。

历史沿原2MiB响应预算按完整Record缩页，预留envelope空间，nextBefore取最后真正返回记录，不跳过被预算截去的记录。精确单记录沿原大小限制；错误不能伪装成空历史。

## 权限与Threat Model

复用verified effective org/actor/member、live supply.read和Data owner权限、原SourceSelector retained-source合同。Service读取前authorize并Select准确source；读后再次核对live scope与固定source。Repository不能用仅org或仅actor的记录读取，payload和分页信息均同scope。取消/超时/撤权/member变更/固定source不一致fail closed，不返回部分结果。

此历史入口属于已知source的直接读取；不把商品详情反查的“当前own item未归档”规则扩展到retained source。导航item不增加或替代权限。用户无manage时可以查看历史、旧版、比较；不生成任何保存调用。

Threat Model限定本轮：跨org/actor/member/source泄漏记录、伪造revision或source/draft错配、页预算遗漏、跨草稿误比较、导航错配和迟到scope响应。无新增权限定义、授权owner或跨系统安全边界。

HTTP/BFF沿原no-store、verified identity/live organization access、HTTP20s/domain10s/body-read3s；GET拒body、content encoding、Idempotency-Key，严格query。BFF和客户端验证page source echo及每条source/draft/revision顺序与游标；精确读取同时核对sourceID和revision，不能只验证JSON形状。

## 比较与页面上下文

路径：
- `/workbench/supply/mine/missing/:sourceId/history`。
- `/workbench/supply/mine/missing/:sourceId/history/:revision`。
- `/workbench/supply/mine/missing/:sourceId/history/compare?from=Rn&to=Rm`，须`from < to`。

三个页面可带原可选UUID preparation/item；沿原严格导航校验，必须与读取的实际Source匹配，item还要求own kind。上下文仅构造固定本地href，不接受任意URL。返回最新草稿沿原独立编辑页；返回商品和待补全列表沿原item/preparation。历史、旧版、比较之间完整携带已验证上下文，导航变化和scope变化remount/abort，迟到结果不能显示到新上下文。

历史列表显示真实保存时间、Rn、候选标题、当前head标记，支持选择两个版本（可跨页选择）后进入独立比较页。无保存时显示“尚无保存历史”，提供返回草稿。

旧版只读显示准确Rn、保存时间、候选标题/描述、固定Vn及来源标识，不把当前商品资料当旧版候选内容，也不提供保存/回滚。

比较页用上述精确读取API并行读取两个已保存Record，核对source完整相同、draftID相同、revision准确后显示。只比较content.title和content.description的精确字符串值；保留空串、空白和换行，不做HTML解释或归一化。采用两个字段的前后值及“已变化/未变化”表达，无字符级diff算法/新依赖；复用现有字段对比页面样式。保存时间和Rn为元数据，不作为内容差异；固定Vn在页头单独展示，图片/SKU/当前商品版本不参与候选比较。两个版本完全相同也明确显示“标题和描述没有变化”。

## 持久化、副作用与验证

全为只读；无事务写入/幂等键/新恢复状态/外部副作用。原Save/CAS/command重放/UNKNOWN协议、head/record/command原子事务不变。读取失败仅重读；不调用Transfer或Save。Legacy decision: N/A，复用当前合格owners，无RETIRE abstraction、fallback、双读/双写或第二事实源。

必要TDD：domain的范围/首尾权限/错误record、实际fresh PostgreSQL多Rn/倒序keyset/预算截页/精确旧版/错scope/source/篡改source拒绝；HTTP/BFF严格参数/GET/echo；UI只读三页面、空/错误、两版本比较、导航返回和迟到响应；相关原Save回归、typecheck/lint/build、最终候选CI及一次最终独立复核。不另建验收工具。实现者开发核对不代签用户/生产验收。

运行沿现有CompletionDraft能力开关与安装；后台/前端同候选，无新schema/grants。本次不更新own675或操作业务数据；交付候选入口、正常启动命令及数据保留说明，实例更新另需具体授权。

## 独立准入

2026-10-15，own_history_review实查#704正文、适用规则及实际Service/SQL/SourceSelector/导航/注入，以先于生产的设计提交834fdf4ebddab140041a716703675b6b26a00abd确认R1 IMPLEMENTATION_READY，无当前Must/AGENTS BLOCKER。冻结本合同；scope/source/draft隔离与首尾权限、head上界/倒序预算分页、同草稿比较及导航迟到隔离均为IMPLEMENTATION_TEST，在本Delivery Batch实现和验证，不重开架构或扩展验收平台。
