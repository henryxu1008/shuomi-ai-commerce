# 单候选人工核对已保存 1688 货源 · 独立架构设计

设计准入：**Independent Architecture / IMPLEMENTATION_READY**。原送审版本为 NOT_READY；原独立 Reviewer 通过设计修订 `7e05cbd7449ad714bbb1a0bfe49a471a8cd82bcc`，结论记录于 #1074。后文保留原设计开工检查；本结论仅为实施准入。沿 [#1074](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1074) 和 [Draft #1097](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1097)，基于已完成的人工试算 `0528a77fe105a4dd84c18250dae1ec9b143a8680` 与准确的 #1075 消费基线 `3484e73588ee67e99f8dd3a55b2ecb9cb1a42822`。原试算、SHEIN 只读与转入门禁沿用既有交付。

## 1. 用户结果与范围

获权成员在同一私有 Research 项目的「评估与选品决定」，打开一件已关联并选中有效证据的 Amazon 美国站候选。点「核对已保存货源」，通过现有 Collection 商品选择器选一条已保存的 1688 商品；从原版选择准确 SKU、填写数量和单位，对照规格、写理由，保存后在原候选里查看同一份核对记录和两侧原版。一次只处理一件候选、一条货源和一个 SKU。取消回到原候选并保留当前选择和列表位置。

2026-11-06 用户确认本轮局部产品依据。原「评估与选品决定」内，为一个已准入的 Amazon 美国站候选选择一条已收藏的 1688 货源，确认 SKU 和数量，保存规格差异与理由。结合 [#1074 本轮产品决定](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1074) 与本节，确认、失败恢复和历史回看均在原候选内。新增区块依据用户确认的范围，复用仓库现有候选工作页、收藏选择器和返回行为。[研究详情](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=12114-1646)与[候选评审](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=12114-1746)仅供核对业务流程。界面与验收依据为本轮产品决定、现行领域合同与仓库现有组件，见[当前业务流程参考边界](../product/final-ui-ia-authority.md)。其余业务依据见 [商业选品产品边界](../product/research-commercial-assessment-v1.md)、[货源关系原架构](research-commercial-assessment-v1.md)与[现有产品路径](../product/market-research-selection-centers-v1.md)。

- 「人工确认这条货源关系」表示成员已核对所选资料，愿意把该 SKU 和数量用于后续研究；**不证明客观同款、可下单、有效报价或通过硬性风险**。保留「待核对」用于准确 SKU 已知但规格资料仍不足的情形。
- 对照只读两侧原件已有的结构化规格：Amazon 原版 `Attributes`，1688 原发布中所选 SKU 的 `Attributes`（商品级值单独标出处，不能冒充 SKU 级值）。成员最多选 5 个与这次判断有关的规格，至少 1 项；保存字段键和判定「一致／不同／待核对」，实际值从原版回读。两侧均有明确可比值时才允许选一致或不同；缺项、单位不清或语义无法核实用待核对。未核实项只能保存为待核对；已核实差异可以保留并在理由中说明为何仍选作研究货源。没有来源 SKU ID 时只提示待核对，不生成能冒充准确 SKU 的持久关系。理由必填，纯文本，最多 2,048 UTF-8 字节。
- 采购意向数量为 1–1,000,000 的整数，首版单位只允许「件」(`piece`)，不做盒、套、起批金额或多 SKU 换算；无法说明数量是按件时不保存。固定分开显示「采购价：未知」「该数量适用单价：未知」「MOQ：未知」「混批：未知」。不把收藏价格、库存、阶梯或数字暗示为报价。原人工参考试算和候选决定照常独立使用，不从新关系填价格、不推动原状态。
- 不接自动找货、报价、风险判定、模型、取数、采购、支付和转入，不增加权限角色、供应商目录、第二商品事实源或新页面。万邦补证只影响依赖真实报价的切片。按[全新系统基线](../product/greenfield-no-legacy-migration.md)实施，不升级旧库、不回填、不读取旧系统。

## 2. 权威事实、读取权限与准确身份

| 事实／检查 | 原有 owner 与本片接线 |
| --- | --- |
| 谁能看或保存 | `internal/app/httpapi/market_research.go` 的 `researchProjects` 继续绑定当前企业、actor、有效成员、私有 `PRODUCT_RESEARCH` 项目权限与 `workbench.collection.read`。保存还要当前项目管理权、项目活跃；归档项目仅可按现有读权限查看。用同一已验证身份上下文读取两个来源，不使用服务账号或创建者身份替代请求人。 |
| 候选与 Amazon 原件 | Research 按项目/actor 隔离的现有候选，必须为 `amazon/us`，且选中非移除的证据链接。用现有 Data Services `Provenance` 回读 `amazon_data`，校验原站点为 US、ASIN 对应该候选、来源对象及原件校验值与现有 `EvidenceRef` 一致。原选择和摘要绑定到此次关系；不能让用户手填 ASIN 或链接替代。 |
| 1688 已保存商品及原件 | Collection 负责当前条目、当前批次的可见性以及原发布绑定；Sourcing 与 Catalog 继续负责源发布、SKU、属性及版本。只接已有 `PublishedAcquisitionReader.ReadPublished` 证明的 1688 已发布采集（现有 `AcquisitionProducerKind@v1`、`acquisition_evidence_v1`，已准入的 `public_http` 或 `public_browser` channel）。`kind=acquisition` 只是选择器的初筛；只有标签、仅自有商品、供货市场引用、SHEIN、或未能用此入口证明的记录拒绝。 |
| 允许的 1688 只读入口 | 复用 `internal/app/productsourcing/application.go` 注入 Collection 的 `NewPublishedAcquisitionReader`。它没有采集、发布或扣费能力，但 `ContextAuthorizer` 现有规则要求活跃的 `product_sourcing.write`；本片保留该读前置，不给仅有 Collection read 的用户透传原件，不加 grant、代理身份或替代构造器。账号只有收藏读权时，仍可用原收藏功能，不能保存或查看本片的 SKU 核对细节。 |
| 人工核对关系 | Research 只拥有“某成员在某项目为该市场证据选择哪条货源及理由”的记录，隔离于候选状态、人工决定与试算。1688 不进入 `research.Observation`、候选平台枚举、归并或付费对比。 |

**保存用的输入与存下的锚点。** 客户端只提交当前候选 ID、预期 Research revision、选中的 Amazon link ID/摘要，及 1688 Collection 条目四元组（item ID、当前 revision、原 publication ID、原 version），另加 SKU ID、数量、单位、规格键/判定和理由。可信绑定由服务端回读：在 Collection 当前 actor 范围内核对条目 ID 和当前批次存在且未归档，校验作用域、来源类型、ProductKey、操作 ID、Collection 的 publication ID/version 与 Sourcing receipt、Catalog publication ID/version/tenant/ProductKey 对应。由现有 Sourcing reader 核对 receipt 与原发布的包络摘要；记录并比较 `Envelope.RawReference.Checksum` 原件校验引用，不声称本片再次获取或校验站点原始字节；不把 Catalog ID 与 Sourcing publication ID 当成同一命名空间。用 `Canonical1688Source` 核对原发布的规范 URL 与 offer ID、操作绑定及 producer；不用调用方 URL。1688 的原发布 ID 沿 Sourcing 当前命名空间，形如 `source-run:acquisition:<operation UUID>`。Amazon 市场证据继续使用原 UUID 合同。SKU 必须在该不可变包络的 `ProductCandidate.Variants` 中以非空的来源 `SourceID` 精确且唯一命中；本地位置、生成 ID、标题和价格均不是 SKU 身份。不保存完整商品、原始页或报价。

**区分收藏移动与换版。** 当前写入先读项目选中的原 Amazon link，取得两侧条目的当前 revision，带当前 revision 校验新选择。1688 客户端提交的四元组必须仍匹配，表单期间即使只是移动也要重新确认；不替用户默默换当前选择。Amazon link 的历史 `Selection.Revision` 保留在原 `EvidenceRef` 摘要内，不拿新 revision 重算旧 link 或覆盖旧 Ref。新关系记录两侧保存时的条目 revision 供诊断；在历史读取路径不以它充当商品版本。

Collection 新增窄的内部「按已绑定原件读」能力：输入来自该项目已验证的 Amazon link 或已保存核对关系（预览/写入仍须当前条目检查），不是让客户端传 ProductKey/原发布任意查库。先以当前权限读该 item 和现批次，再比较绑定的 item ID、source kind、ProductKey、原 publication ID/version、operation ID；按原键读取，不调用 `latest`。原件取回后重验同一当前作用域、条目和现批次可读及不可变来源绑定。移动只改变批次和 revision，同一原绑定且新批次仍可读时，原记录继续可读；条目换版、替换来源、删除、归档、当前批次不可读、原件缺失/校验失败时遮蔽。新快照另存成新条目不影响仍可读的原条目。Amazon 复用 `observeAmazon` 的原件校验部件，但不能直接用 `researchSources.Observe` 强制旧 revision 或改宽其他调用方的 `Collection.Select` 合同。

## 3. 保存、读取、恢复

复用原 `POST /api/v1/workbench/research/projects/:project/commands` 命令与 `GET …/commands/:key` 回执，加有界判别分支 `record_supplier_review`。用现有 BFF/会话约束及请求上限；限制 ID/字段数量、键长度、纯文本和整数范围，拒绝未知字段、非规范数量、重复规格或非首版单位。不记录表单、来源内容或请求 body 至日志。只新增候选内窄读入口（同一 Research 路由下 `/candidates/:candidate/supplier-review`）：预览仅接受选择器提供的四元组，不接受凭据、URL、理由或可伪造来源 ID；当前／历史模式按候选读取已存记录，键集分页每次最多 5 条，不扫全项目。响应 `no-store`；预览可选项不含源报价或无关字段。

新表 `research_supplier_reviews` 为仅追加的 Research 关系，主键 `(organization_id,actor_id,project_id,request_key)`，同作用域 FK 到候选和原命令回执，唯一 `(organization_id,actor_id,project_id,candidate_id,candidate_revision)`，按此候选/revision 支持倒序读。只存 Amazon link ID/摘要与其不可变原引用、1688 精确绑定与 SKU ID、数量/单位、规格字段键及人的判定、结论/理由和时间。Market link 不由关系表管理，移除时保留原锚点。原命令仅保留现有最小回执、规范输入 SHA-256 及非敏感路由元数据；供应规格、数量、理由等不重复放在 `intent_json` 或 `decision_json`，不进入原决定历史。

1. **开始**：POST 通过项目写权限检查后先查同作用域原请求键。相同规范输入返回原最小回执，不再预览或保存；不同输入同键冲突。已有回执可在仅失去来源权但项目读权仍在时用原 GET 查到，不返回来源、SKU 或理由。项目读权失去时回执也不可取。
2. **新写入**：先以现有 owner 接口完成预读。紧邻进入 Research 写入事务前，用当前请求身份复验项目写权、两侧有效条目和批次、原发布/版本/摘要与 SKU，失败直接结束。按 [Research 事务合同](market-research-selection.md) 将项目、Collection、Data Services、Sourcing 的授权和 I/O 全部留在锁外，各自使用当前受限连接。完成复验后交给仓储的值只存在于该请求，绑定 scope、project、候选和所选证据。新写入须在该复验凭据的有效期内提交，期限取复验后最多 5 秒、当前 token 截止和请求截止三者最早值。等待锁超时或证据过期时按原请求规则退出，下一次尝试须重新做锁外复验。已提交的同键同输入仍可取得原回执。事务内沿用 (org, actor, project, key) advisory lock、二次查键及候选行锁/CAS，仅检查本地候选 revision、当前非移除的 Amazon link ID/摘要及复验结果是否仍有效。关系、原回执、候选 revision 在同一事务中写入。撤权或原件换版与数据库提交有竞态时，关系只绑定已校验的旧原件，每次展示按当前授权遮蔽；不跨 owner 加锁，不显示最新版本冒充原件。
3. **并发或丢失响应**：两个不同键对同一候选 revision 竞争，最多一个成功，另一方重读后才可由人发起新核对。提交结果未知或响应丢失保持 `UNKNOWN`，客户端沿用 `useResourcePending` 仅存项目作用域与请求键，敏感表单仅留当前内存。先以原键查回执；当下无回执不能当作确定失败，不换键自动再写。原输入仍在内存且用户重试时，只能重放同键同输入。刷新后继续查原键，浏览器不持久化理由或资料。货源核对的挂起恢复使用自己的作用域键，不阻断原人工决定。明确冲突或字段不再可用后按新的候选和原版重新填写。没有队列、定时重试或外部副作用。
4. **查看**：当前区取该候选最新一条关系，且要求它仍绑定当前选中的非移除 Amazon link 和相同可读原件；不可用时不拿上一条顶替。切换市场证据后旧记录仅在历史区，继续受两侧权限检查；切回同一仍在项目内、精确匹配且可读的 link 时可按原关系判定。解除该 link 或任一侧失权，隐藏该条的来源名/链接/ID、SKU、数量、单位、规格差异、人工结论、理由及任何由这些字段推导的信息，只显示「原资料当前不可查看」。历史不能绕过同一校验；项目、成员或企业不匹配直接拒绝整个请求。候选卡的查看动作以新只读接口的结果为准，不受旧 `observation` 因收藏移动而为空的判断阻断，也不影响旧试算与决定的原校验。前端候选/项目切换、重新请求失败或收到遮蔽时清空旧详情和草稿，不在待核实时继续展示旧快照。

## 4. 最小实现路径和空库准入

| 路径 | 拟改责任 |
| --- | --- |
| `internal/product/research`、`internal/integration/persistence/product/research` | Research owner：窄命令与读取合同、现有 CAS/回执事务中的关系写入、当前与历史读取过滤。新表和最小授权随 Research schema owner 管理。 |
| `internal/product/collection/selection.go` 同包、`internal/app/httpapi/market_research.go` 及现有 Research handlers | Collection owner 增加内部精确原件读的窄边界；应用装配重用已注入的 Sourcing 只读能力与 Data Services 原件读，挂载受项目权限保护的 Research 路由。无额外 provider、数据库池或密钥。 |
| `web/listingkit-ui/src/components/workbench/project-center/research-panel.tsx` 与现有 Research BFF/契约 | 复用单候选内现有选择器、挂起键和错误处理；新增核对表单及原记录查看；独立于试算与原决定。 |
| `cmd/product-acquisition-init`、`internal/app/productsourcing/acquisition_initialization.go`、`internal/app/runtime/currentapplication/research.go` | 复用 Product 空库初始化和原 Research restricted pool，协调同一既有库目标上的 schema 及默认关闭的功能装配。 |

现有 `cmd/market-research-schema-init` 不验证空库；将新表直接加到其通用 `InstallSchema` 会使保留库也能安装，是本片必须避免的准入缺口。新表的 DDL 和权限只由现有 Product **空库初始化分支**显式选入：校验私有 manifest、确认实际数据库名/既有库目标、获取现有初始化锁并验证全库无用户关系后，在同一初始化事务里由 Research owner 安装基础 Research schema 与新关系，并给予既有 `product_research_runtime` 对新表 SELECT、INSERT（无 UPDATE、DELETE、DDL 或其他 owner 表权限）。不使用现有 nonempty feature initializer 开关，不加独立补装命令、迁移或回填。原单独 Research installer 仍只处理原合同，新表不进入旧默认安装清单。现有 `VerifyRuntimePermissions` 的表白名单只在检测到准确的预置 Research 关系、约束和 SELECT/INSERT 权限时承认这一个额外表；不把同名任意表、更新权限或其他域表纳入白名单。功能装配默认关闭，明确启用且验证目标、schema、grants 和 reader 后才挂载；缺表或 reader 不开放，过宽 grants 沿原安全规则拒绝。关闭或启用都不自动建表、迁移或读取业务数据；原三表的环境仍可用原 Research。现有角色不额外获得 Sourcing/Collection 表权限。实际库初始化和用户试用仍需另行授权。

用于复核 owner 接线的现有依据：Collection 使用 `internal/app/httpapi/product_collection_application.go` 已注入的 `NewPublishedAcquisitionReader`，由 Sourcing `ContextAuthorizer` 校验现有原件读前置权限。Research 消费沿 `internal/app/httpapi/market_research.go` 注入，读原件的权限和版本仍由当前 owners 返回。Product 初始化沿 `InitializeAcquisitionDatabase` 的既有空库事务，Research store 交付新表、约束和现有角色的最小 grants。应用装配验证这些条件后才注册入口。

## 5. 离线验收及准入条件

合成走查：候选 C revision 7、Amazon US 证据 A、1688 收藏 S revision 2 / 发布 P v3、唯一 `sku-blue-01`，填 20 件。Amazon 原版为蓝色、1 件装，1688 SKU 原版为蓝色、2 件装。保存核对理由「包装差异已看过，仅作货源研究」，两侧资料仍显示原值，数量仍为 20 件，采购价、适用价、MOQ、混批均未知。若包装单位无法判断，则只允许保存为待核对或在数量单位不明确时拒绝保存。正式验证在 Ready 后用 TDD 实现，当前只做文档和代码边界核对：

| Must | 隔离验收 |
| --- | --- |
| 原件与 SKU | 只用合成已保存发布；错站点、错项目/企业/actor、伪造链接/来源、错版、错摘要、未知/重复 SKU 或归档批次不得保存。换版在最终检查前发生，事务无记录、无回执、revision 不变。 |
| 回看与撤权 | 收藏移动且同一原绑定时仍读原值；旧表单 revision 不再能新写。选中另一 Amazon 证据时不显示旧关系为当前。模拟 Amazon、Collection 或 Sourcing 各自失权、换版/归档、项目失权；按上述层级遮蔽，原行仍保留。仅恢复权限且仍能读同一原件时重新显示。 |
| 并发与恢复 | 同键重放一条关系、一张回执、一次 revision 加一；同键异体冲突；两键同 revision 只成一笔；注入本地写入故障验证三项原子回滚；提交后断响应只查原键，回执不含敏感内容。 |
| 空库与回归 | 仅获准隔离空库验证正常安装、最小权限和默认关闭；错库名或任何现有业务/用户关系的库拒绝且无变化。复验原人工试算与原决定，SHEIN 不进入新表单或付费/转入。 |

开工条件：本轮局部产品依据已记录于 §1 和 #1074。原独立 Reviewer 需在新准确 SHA 上确认 Research 外部读取在锁外、Collection/Sourcing 当前只读接线及 Product 空库安装/授权边界，并记录 IMPLEMENTATION_READY。§4 的接口和路径是 owner 交接面，未记录的 owner 同意仍待确认。通过后由原 Writer 沿原分支和 Draft 做隔离 TDD。自动找货、硬性风险判断和完整 #1074 仍单独未完成。
