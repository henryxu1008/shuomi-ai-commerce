# POD设计智能体 v1：从设计意图生成图案

Refs #672 / #137；源码依赖未合并#662/#666。Design Basis: **Independent Architecture**。Status: **IMPLEMENTATION_READY**。原只读content_final_review第2轮于d0063eeaf0df2288351decd89319d3768c899b31确认两个开工Blocker关闭并达到准入；6bb依赖组合已局部APPROVED。本文先于正式POD实现，冻结后仅新的具体Blocker重开设计；已列IMPLEMENTATION_TEST在实现层收敛，准入不等于功能完成或用户验收。

## 产品结果与依据

用户2026-10-13明确纠正：原业务是先让AI画出全新图案，再继续后续步骤。第一版完整路径为：输入主题/设计要求 → 有界报价与确认 → AI从零生成图案 → 查看实际图片并确认保存/用途 → 选择POD模板款式 → 布局方案/预览 → 显式确认SDS执行 → 原设计操作核实效果图。已有图案是补充入口，不替代从零生成。

产品依据：[业务职责](../product/business-agent-responsibilities.md)、#137/#672当前决定；原#612图片设计、[product-image-agent-v1](product-image-agent-v1.md)、[supply-market-v1](supply-market-v1.md)、organization-agent-configuration-v1与enterprise-agent-permission-modes-v1。POD是用户意图的专业入口；图片生成、资产、数据接收和供应商执行分别保持原owner。

Figma Authority为tg48P46SSXl6TBy9lZwg63当前非归档31:463。我的智能体/配置/确认沿4703:429、4703:892、4722:429；生成结果与图片核对沿#612的4908:429/5198:429及当前组件。模板选择沿429:3258/429:3347，现SDS编辑器沿#622批准的状态与当前Figma样式；原型没有完整SDS页面，不伪称存在。新入口位于我的智能体与现POD设计流程，不新建顶层中心或平台工作台。原型价格、图案和成品示例不是业务事实。

Scope：单个创作Run生成一个可检查图案，沿现生成结果存储/确认和POD设计链路交付可核实结果；明确原键、费用/实际像素、资产、来源和供应商操作。必要的创作输入、批准生成作品接收与配置/模式/API/UI/worker接线随本批实现。

Out of Scope：新图片SDK/模型路由平台/计费账本/队列/审核或POD事实owner；批量营销、订单、采购、生产下单；迁移/映射/自动启用；Legacy包装；真实IAM/付费模型/SDS/SHEIN/支付调用；安装部署、合并、关单、CI/保护变化。新专业入口不竞争#612通用商品图片owner或#31/#655 TEMU。

## 现状与最小合同补齐

- `internal/integration/openai/images.go`与`integration/grsai/client.go`已有GenerateImage传输能力。当前`imageagent.GenerationExecution`、ImageSet recipe与worker主要消费商品原图、References及SourceBytes；不能把调用SDK当成端到端接通。
- `internal/app/pod.Service.Artwork`只列出已保存Source的真实图片；`OriginalInputs`、Asset RoleDesign批准及POD Plan绑定准确来源。新图案必须通过当前生成结果与真实作品接收，不能给POD造假Source/原图或借模板空白图进行所谓文生图。
- 同一ImageAgent运行/slot/effect owner增加明确的创作输入变体与执行分支。创作输入只含用户设计意图、所选有界输出规格、原配置/模式/报价引用；它不是ProductSnapshot。没有参考图时References为空，不请求原图bytes。商品图片现有分支仍必须真实原图/证据。
- 创作输入与输出由固定协议`pod-artwork-v1`、注册定义`pod.design.agent/v1.0.0`识别；不可通过商品`custom`提示词、任意source URL或兼容分支绕过。具体新增标识必须在对应精确schema/配置/worker verifier同时消费，不留下菜单空壳。

## 权威事实与调用链

| 边界 | owner与合同 |
| --- | --- |
| 注册/模板/四模式 | 原agentconfig；独立POD参数schema与明确启用，不借标题/图片模板、模式或默认模型授权；原configuration_snapshots及image_run_admissions承载精确POD typed payload |
| 创作Run/plan/slot及恢复 | 原ImageAgent与Temporal；新增创作输入变体，不增加第二个Agent工作队列或状态机 |
| 报价/预留/生成与用量 | 原GenerationExecution、GenerationFact、GenerationResources、orgresource点数/月限及存储；一次外部effect，原price/route/credential refs冻结 |
| 实际生成图片 | 原生成输出materialization与owned artifact store；实际bytes/hash/尺寸/类型，模型URL不直接成为批准资产 |
| 保存作品 | 当前productsourcing接收、Source/Catalog/Collection UoW；窄generated-artwork receiver从原生成结果reader获取事实 |
| 图案批准与消费 | 原Asset/SourceApproval RoleDesign；准确作品来源、版本与生成事实，不自动把下载成功当批准 |
| 布局与成品 | 原POD Plan/Repository/Processor及Submission ExecutionKernel；原account/manifest/artwork/fence与原回执读回 |

contract → implementation → injection → consumer：创作输入/输出变体落原imageagent；无参考图的受控creator port落原生成executor/tools，SDK仍在integration/worker adapter；原OrganizationImageCapabilities/current-application注入同一事实/资源/模式/存储及原worker。实际生成结果经app层窄receiver进入当前Source/Catalog/Collection，Asset负责批准；POD原Inputs/Service消费这些准确作品，原Processor/Kernel产生SDS效果图。领域不直接依赖SDK/GORM或Legacy服务。

本批合同与原消费点固定如下，名称是新合同，不是声称已有实现：

| 最小合同 | 实现与注入 | 当前消费者与不变量 |
| --- | --- | --- |
| `ArtworkPlan`、`PrepareArtworkInput`、`WithArtworkDependencies`、`Service.PrepareArtwork/GetPreparedArtwork/ConfirmArtwork` | 原`internal/imageagent` Service/Repository，原app组织scope组装注入配置reader和quote reader | `Plan.Artwork`和`SlotExecutionInput.Artwork`为互斥的精确JSON变体；一Run一slot；原Temporal及projection保存，不走通用Start/SubmitPlan自由输入 |
| `ValidateArtworkPlan/ValidateArtworkExecution` | 原Plan ingress与原slot executor | 明确空Sources、空References、空ProductContext；仅此协议可无原图。`AssetCatalog`是空资产清单的真实manifest，原`NormalizeAssetCatalog`已允许空ProductContext；不得放虚构ProductID或source asset。创作意图的完整digest另存`ArtworkPlan.InputDigest` |
| `ArtworkProvenance`精确输出变体 | 原`GeneratedAsset`、`StagedAssetRef`、`PublishedAssetRef`及`AssetCandidate`可选typed字段，原manifest/slot-effect持久化 | `SourceAssetID`为空且provenance为创作协议、真实scope/run/revision/slot/attempt、inputDigest、intentID/fingerprint；字段由执行owner从当前输入/GenerationFact铸造，provider不得自报。原商品结果仍要求真实SourceAssetID且不得附创作provenance |
| 受控`ArtworkCreator`与`ArtworkSlotExecutor` | 原tools及worker generation provider，现integration GRSAI同步客户端`GenerateImage`，同一`GenerationExecution` | quote、reserve、BeginDispatch和success/finalize仍原owner；creator固定一次POST，纯创作分支跳过ReadSourceBytes；商品分支不改变原必需source校验 |
| `GeneratedArtworkReader` / `ReceiveGeneratedArtwork` / `ArtworkPublisher` | app读原ImageAgent projection、GenerationFact与finalized artifact；原Temporal最终采用publication活动调用当前Collection窄采用合同进入原`Repository.Execute`、`product_collection_operations`及`OwnPublisher/Source InternalProducer`的Product UoW | 输入只有原Run/slot/resultHash/配置引用；不收URL或任意商品正文；stable command/operation派生自原scope/run/slot/hash；原publication/collection receipt重放，不复制生成费用。不能调用没有command receipt且只接受market/SDS的`AppendReceived`冒充恢复 |
| `ArtworkOriginReader` / POD创作执行guard | app在原Source reader上读取不可变`RawReference/Trace`，注入原POD `OriginalInputs`/`DesignExecutionAuthorizer` | 生成origin由receiver铸造；原Source及原Run/GenerationFact交叉核验配置/模式/actor；前端删标记不能关闭guard。原POD Processor/Kernel消费同一fresh gate，不增加工作队列 |

原ImageSet专用worker构造与原组织运行授权扩展显式支持这一个创作协议，沿原serving root注入；不能注册后留worker拒绝新协议。原商品ImageSet publisher不采用创作作品，作品确认调用上述窄receiver；原RoleDesign批准和POD Create仍是各自领域动作。参数、模板与配置快照精确属于`pod.design.agent/v1.0.0`，原ImageRunAdmission仅复用准入行为，不冒用`product.image.agent`身份。

配置owner新增typed `ArtworkStartCommand/ArtworkConfigurationSnapshot`、`PrepareArtworkConfiguration/LoadArtworkConfiguration/AdmitArtworkRun/CheckArtworkExecutionMode`，复用原snapshot row、模式snapshot/check与image_run_admissions row的锁/幂等准入写法；不增加表。`ContextKind=pod-artwork-v1`、subject/definition必须是精确POD注册项，payload含原mode snapshot、真实member/run/request/inputDigest/参数digest/quote及硬上限。原`PrepareImageConfiguration/AdmitImageRun/CheckImageExecutionMode`仍只接受product.image.agent，不能替POD放行。服务确认前及worker reserve/dispatch前从原persisted ArtworkPlan定位精确configuration/admission，fresh检查启用epoch、模式revision、成员及到期/额度；去掉或替换Artwork字段不能退回商品或通用运行分支。

### 无原图产物的原staging/final合同

当前`prepareGeneratedSlotArtifacts`、`NormalizeStagingManifest/NormalizeFinalManifest`及slot-effect候选验证要求非空`SourceAssetID`。因此仅补输入不够；本批同步补上述`ArtworkProvenance`判别，不能填占位source来通过校验。两个合法且互斥的形状为：商品结果有真实SourceAssetID且无ArtworkProvenance；纯创作结果无SourceAssetID且有完整ArtworkProvenance。缺两者、两者同时有、伪协议、scope或digest不符均拒绝。

`Normalize*Manifest`验证创作provenance的结构、规范ID/hash和唯一操作`generate_artwork`，保留同样的object key、真实hash/bytes/type/尺寸界限。`prepareGeneratedSlotArtifacts`和`ValidatePublishedAssetRefForSlot`从当前原persisted ArtworkPlan/GenerationFact核scope/run/revision/slot/attempt/inputDigest/intentID/fingerprint；空SourceAssetID仅在这个已验证计划中合法。原artifact store staging→final复制该精确字段，manifest摘要/slot-effect candidate fingerprint纳入它；`ValidateSlotEffectV3Candidates`与专用Artwork result builder核同一绑定。原恢复输入及结果也携带该变体，重新从原projection校验，不能因omit字段降级。

正式Source接收只消费`finalized`的单一候选：真实GenerationSucceeded、确定settlement、final manifest、candidate digest与provenance全匹配。UNKNOWN、仅staging、未结算或另一个attempt的图片不可采用。原commodity ImageSet publication明确拒绝Artwork变体；`ReceiveGeneratedArtwork`接收后，Source资产才获得真实新SourceID和RoleDesign批准来源，POD消费的是该真实Source与原作品origin，不把生成意图摘要当已有source asset。生成origin继续通过原Source RawReference/Trace追溯原fact，不新增产物事实owner。

## 输入、生成与持久化

主题/要求有界UTF-8，输出规格只取当前已配置模型实际支持的值。服务器铸造固定prompt版本，完整规范化意图及输出规格参与input digest、quote和plan；前端不能指定actor/member/month/provider URL、价格、credential、任意工具或模型参数。图案图片生成本身承担创作，不为形式再增加一条文本Agent运行。

纯创作manifest明确未绑定不存在的Product/原图。`CatalogHash`仍是原规范化空资产清单hash，而不是不存在的商品版本；完整创作意图另由GenerationFact的`InputProtocol=pod-artwork-v1`、`InputDigest`绑定。`ArtworkPlan.InputDigest`绑定原规范化Prepare输入；生成时`ArtworkGenerationInputDigest`基于完整`SlotExecutionFingerprint`（含准确plan/配置/意图/quote/slot等）生成最终输入digest，不把这两种摘要混用。`SourceDigest`在此精确协议表示该真实创作执行输入摘要，必须等于GenerationFact.InputDigest；原商品协议仍表示真实source bytes摘要。ArtworkProvenance中的inputDigest取原GenerationFact的最终digest，按原persisted Plan计算并重验。持久化Fact的Validate、恢复与staging fingerprint都执行该判别，禁止跨协议混用；旧原图合同不能松成“没有源图就随便生成”。

SDK creator使用当前明确provider/configuration/price准入。prepare和发送前校验原route/credential/model/price版本、模式、成员、当前额度及存储。缺少配置拒绝对应新生成，禁止fallback到标题模型或其它provider；无预算/价格时不显示免费或可用。实际调用与计量仍在同一GenerationExecution中完成。

Provider POST只能由原fact的单次dispatch permit发出，adapter固定一次尝试；请求丢失、结果下载失败或用量不明保持原UNKNOWN/预留，由原recovery owner读回，不另起Run/slot重新付费。生成bytes完整解码并验证类型/大小/像素；候选保存原owned artifact，实际尺寸和限制向用户披露，不冒称已达到印刷质量或生成了供应商成品。

生成候选不是正式Product/RoleDesign资产。确认作品后，从原成功且已核实的artifact取得真实图片，当前receiver生成自有设计作品的Source/Catalog/Collection事实，不造SKU、库存、价格或供应商商品。SourceEnvelope标明`SourceType=agent_generated`、`SourcePlatform=pod_artwork`、真实原Run/slot/hash及creator/member/config来源，ReferenceType为创作协议；不是`user_input`或SDS成品。现Source原始envelope是生成origin唯一持久化owner；Catalog仅投影真实标题/要求与图片。原生成事实和费用不复制/迁移；人工上传路径保持原含义。

候选公开读取地址由原`DurableAssetPublicURLResolver`从finalized immutable public object key生成，不能用瞬时provider URL。真实bytes/hash/尺寸/type必须与该原finalized artifact和GenerationFact一致。原Source/Asset安全URL检查与原POD下载/完整decode仍适用；无可读且安全的持久图片地址时，作品采用失败并保留原生成结果，不伪造地址、不重新付费生成。

作品采用key为当前scope、原Run/slot和准确结果hash派生的稳定UUID键。复用`product_collection_operations`和原`Repository.Execute`锁/原key+inputHash冲突/receipt原子提交；新增窄内部adoption command，普通Collection Mutation不接受该action或生成envelope。`Source.Kind=own`表示自有作品，已被当前Collection schema接纳；精确AI来源在不可变SourceEnvelope，不能改为`user_input`。原`OwnPublisher`只复用其Source/Catalog事务发布行为，作品标题/要求/图片来自原受控reader，无虚构商业字段。

同一Product UoW创建Source、Catalog、Collection及command receipt，所有写入同提交/同回滚。原command受准确org/actor/member和原输入hash约束，已存在receipt先重放；仅新写由窄guard在事务内复查当前身份/权限/启用/模式/原结果绑定，不能把守卫放在publisher之外或只在UI。original-key读取/恢复走现只读scope权限，不因mode改变暗中重建作品。并发相同command读同receipt；同key不同意图拒绝。Asset批准沿其原命令另一步完成，跨库不伪称原子事务。中断后读原生成/作品command/批准receipt继续；已完成生成不会因作品保存失败再次调用模型。

生成Run沿原状态：确认plan→Executing→原slot/materialization/evaluation→AwaitingFinalApproval。确认“保存作品”的`Service.AdoptArtwork`只允许当前准确result digest/单一候选，通过原最终采用command/publication活动消费`ArtworkPublisher`，不会进入商品Source修改或ImageSet批准publisher。默认需人确认；自动模式只在原内部采用policy允许时提交同一准确command。`ArtworkAdoptionReceipt`包含真实Collection command/批次/条目、Source publication/version及原Run/slot/hash；原publication ACK/读回机制将该receipt绑定Run并完成。它表示已保存作品候选，不表示RoleDesign已批准或SDS成品已生成。Product事务成功而Run ACK丢失时，原publication recovery用同一command读取原receipt再补ACK，不重复接收作品或调用模型；没有新跨库补偿/恢复owner。

没有新表、迁移或第二账本。需要新增精确配置约束与现有JSON输入/结果变体时由显式initializer实施，serving仅Verify、不DDL/种子/启用。当前facts/模板/请求/模式/volumes保持；发现既有owner无法保存本批必需事实时先报告具体缺口，不临时新建事实源。

## 权限与POD执行

所有入口和worker复用verified Effective Organization、原actor/member及当前业务权限。原付费图片准入`AdmitImageRun`拒绝`read_only`；本POD复用该策略，不扩大它。`read_only`可查看原生成结果/历史并准备意图/报价，拒绝确认模型发送、正式作品采用、RoleDesign批准及POD外部写入。`default`明确确认；`auto_approve/full_access`只处理原policy允许的已授权内部动作。模型调用仍受原报价/预算/用量/存储约束，full不提供新增额度或假用量成功。

图案作为印花用途的选择/批准沿原RoleDesign合同，不由模型自批；当前领域需要确认时，full不能覆盖。未授权外部发布/改价/下架/删除/增加额度均拒绝。SDS“生成效果图并保存成品”明确产生远端素材/设计，仍有单独的当前业务授权与用户确认，不能被智能布局或图案生成成功暗中触发。

fresh模式检查也接在**实际SourceApproval新批准写入口**。当前`SourceApprovalService.Approve`经`ReadSourceSelection`直接`CommitApproval`，不能仅补POD Create/worker来阻read_only批准。Source读取仍保持原只读语义；新批准通过窄`SourceApprovalCommitGuard`/原Repository的guarded commit入口，在原Asset事务的新receipt/asset写入前由app从不可变Source重读真实ArtworkOrigin并校验当前POD配置/模式/成员。普通人工来源沿原批准合同；真实AI作品缺该guard则拒绝新批准。原准确action/payload已提交receipt优先原重放/只读读回，不运行新写gate，不给新action或另一payload放行。guard只消费原mode owner及原来源证据，不新增权限体系或批准事实owner。直接批准API、普通POD批准按钮及随后Create/worker同用这条边界；相同key并发/变更模式/撤销的必要测试在实现层收敛。

由POD智能体产生的作品保留实际生成执行origin。进入原POD创建/worker前从权威作品/生成事实取得origin并重新核对模式，不能相信前端可删的agent标记；普通人工来源不被误认为Agent来源。新写/尚未发送的步骤执行当前gate，原已提交receipt及UNKNOWN只读核实不转成新写许可。此guard消费原POD Inputs/ExecutionAuthorization与原mode owner，原fence/SendPermit/状态及唯一recovery不变。

模板/款式在原POD选择后才冻结；当前manifest的layer/print参数、真实批准图案hash与尺寸共同约束位置、缩放和旋转。第一版布局方案复用现`fitGeometry`与react-konva，按真实图案比例给出可编辑初始布局；不额外调用文本模型。用户能移动/缩放/旋转并核对预览，原POD Plan.Validate与当前输入重新验证负责确定性正确性。画布预览不冒称SDS成品图。

## API、UI、未配置与交接

我的智能体独立POD配置/权限/历史，正常工作台提供“生成新图案”意图与原报价确认、实际候选、作品确认；再进入现SDS模板/设计器，沿原操作结果页核实效果图。UI显示的是实际阶段与原receipt，Model NOT_CONFIGURED、结果UNKNOWN、作品未批准、SDS NOT_CONFIGURED分别披露。4908:429图片操作和5198:429图片预览的当前高保真design context及截图已读取；只复用其布局/控件/视觉规则，实际图案动态展示，不使用原型示例冒充生成结果。

API沿当前ImageAgent/POD命名空间与BFF：strict body、exact key/ref/hash、同源、身份/企业断言与当前nonce规则。Model generator超时沿原有界generation窗口；响应丢失保留原键，浏览器不得自动重发。关闭/刷新恢复原Run与作品/批准/POD命令，切企业前阻止未知请求丢失。

模型配置到位可执行图案生成，不把SDS缺配置作为其无关前置；SDS未就绪只阻模板/远端效果图对应步骤。原worker由原serving root启动/停止，不因新页面重建实例。正常current-application与原frontend入口、明确initializer及角色/配置条件随实现交接；现有运行实例不在本轮修改。

## Legacy、验证与准入

Legacy decision: **EXTRACT**。复用仍正确的文生图传输/图案提示语、当前生成与图案/模板/布局行为；owner为现ImageAgent/Source/Asset/POD。旧Task-first、旧SDS自动重试/自动删除/按最新成品猜测不消费、不包装、不恢复。Source receiver接收新实际作品是当前生成结果采用，不是旧数据迁移或ID映射。

TDD与现有隔离PG/loopback/存储/组件检查覆盖当前完整用户路径：无原图合法从零生成且没有伪造引用；预算/模式/成员/配置撤销；实际bytes与源作品/批准绑定；原key不同意图拒绝、单次POST与UNKNOWN费用/存储保留；生成成功后作品保存失败重读不重生成；read_only直接入口/worker；原POD实际布局/模板/hash与原效果图关联。沿现测试收敛，不新建验收runner/故障平台。

单Writer/主要PR；原content_final_review做新创作/资产采用/worker授权高风险检查及最终完整路径检查，修复仅增量。准入未完成不写POD正式生产代码。真实IAM/付费模型/SDS/SHEIN、安装/业务数据/用户试用均NOT_RUN，不借SDK测试、CI或另一AI审核宣称可上线。

## #674 源码操作入口与边界

入口为 `/workbench/agents/mine/pod.design.agent`（我的智能体 → POD设计智能体）。正常 current-application、原 ImageAgent Temporal worker 与原 Collection/Asset/POD consumer 共用本批接线。生成 API/BFF 为 `/api/v1/workbench/pod-artwork/*` / `/api/workbench/pod-artwork/*`；该路径不以 SDS 凭据或 SDS 前端开关作为原创生成的前置。

使用步骤：
1. 显式启用当前企业 POD设计智能体；创建 `pod-artwork-config-v1` 原创图案模板（风格、约束），选择精确模板版本或企业默认版本；在同页配置四种权限。
2. 填写主题和设计要求，准备原报价。准备不会发送模型请求；确认原计划/报价/点数后才准入原生成 Run。
3. 页面读取原 Run 及实际 owned 图片，检查图案后“确认并保存作品”。原 Collection receipt 保存 Source、Catalog 与“我的数据”条目；生成成功不冒充保存/印花批准。
4. 从已保存作品进入 `/workbench/supply/catalogs/sds/design?artwork=<原条目ID>`。选择原 SDS 模板、款式与图案，明确批准 RoleDesign；沿原 `fitGeometry`/Konva 调整、居中适配并核对布局。
5. 明确点击“生成效果图并保存 SDS 成品”后，原 POD worker 执行；结果页读取同一设计操作的真实成品效果图与 UNKNOWN 核实，不用画布预览冒称成品。

正常原配置仍要求当前企业图片模型路由、凭据、有效价格/预算、原 Image/worker 独立数据库角色与 durable object store。缺失时 capability/API 如实拒绝相关执行；SDS 未配置只阻模板/远端效果图步骤。企业启用、模板和权限不会自动创建、映射或升级。

原配置数据库仅由授权安装方使用现 `agent-configuration-schema-init --runtime-role <现角色> --artwork-configuration-only`，以私密 `AGENT_CONFIGURATION_SCHEMA_DSN` 安装本批精确约束；该 initializer 保留原配置/快照/模式/模板/请求，不重新初始化非空数据库。Serving 只 Verify。完整运行使用项目正常 `start.ps1`/current-application 的原启动方式与原 worker；本源码工作未执行共享实例安装或重启。

浏览器保留 user/org scoped 原请求和原 Run 编号。关闭/刷新后核实原准备键、原确认 admission 或原 Collection receipt；不自动重发 POST。作品采用、SourceApproval 与 POD 新写从不可变 origin 核原 config/mode，原已提交 receipt 读回不授权另一次新写。

本批检查为合成事实、隔离数据库、loopback/owned storage、组件与原 CI；真实 IAM、付费模型、SDS/SHEIN、部署安装与用户试用均 **NOT_RUN**。源码交接、CI 或独立 AI 复核不等于生产或用户验收。
