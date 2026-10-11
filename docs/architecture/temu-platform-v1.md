# TEMU 三种模式与全部官方站点接入设计

> Execution：[#31](https://github.com/qq550723504/task-processor/issues/31)
>
> Design Basis：Independent Architecture
>
> Status：NOT_READY；第一轮独立评审确认 G1/G2 阻止准入，不是 IMPLEMENTATION_READY
>
> 调查基线：main @ f1d77a84fa653813235ddbfeedac59ea6bb0a94c，2026-10-11
>
> 本文只定义待评审合同；本轮未修改生产代码、schema 或正式业务路径。

## 1. 用户结果与产品依据

用户在当前会话确认“都需要支持”，随后明确要求“继续推进 #31，先核实全托管、半托管、本土和全部站点的官方接口，完成架构设计和评审，准入后再开始实现”。本设计覆盖三种模式及全部官方开放站点，不能自行缩为 SHEIN 式 US 单站或 TEMU 半托管单模式。

获授权企业运营人员在现有“我的店铺”连接 TEMU 官方应用；从当前 Product / ApprovedAsset 形成可检查的 TEMU 平台资料，查看缺失信息与确定性校验，明确提交后查看平台创建标识、审核/销售状态与未确定结果；“店铺商品”同步官方商品事实；AI工作台只通过现有受控工具读取、诊断、校验与提出修改建议。

当前 [UI / IA Authority](../product/final-ui-ia-authority.md)、[Store Figma 设计](store-center-figma-v1.md)、[共享供应链设计](my-supply-chain-shein-v1.md)、[全新安装基线](../product/greenfield-no-legacy-migration.md) 共同适用。2026-10-11 只读核对 Figma 文件 tg48P46SSXl6TBy9lZwg63、页面 31:463：1560:359“我的店铺”、429:5323“店铺商品”当前可见。原型示例“当前开放 SHEIN”、数量、网络正常、1548:402 的 API 未开放/子账户路径不是 TEMU 能力或新认证合同依据。

### Must

- M1：三种模式、全部官方站点进入同一产品范围；保留模式、身份分区、API 路由和实际站点各自语义。
- M2：Store 是企业店铺、成员访问、服务权益、连接与凭据唯一 owner；只消费官方验证的商家身份、模式、权限和有效期。
- M3：当前 Product/ApprovedAsset/不可变 TargetRecord 是资料依据；规则可追溯到官方版本，缺少事实保持待补全，模型不能制造品牌、产地、证书、库存或价格。
- M4：只有用户明确授权的当前资料可提交；单一 Submission owner 控制实际副作用、稳定 target fence、UNKNOWN 与恢复；不得逐站重复创建同一个供应商货品。
- M5：官方创建成功、审核状态、在售状态分开投影。连接、服务、同步、提交是独立事实，不能互相冒充。
- M6：Console 与 AI 工具复用当前 owner，跨企业/店铺隔离；UI、BFF、Temporal 和 Agent 不持有 token/secret 或自行调用平台。

Should：清晰指出缺配置、缺权限、未开站、缺资料和结果未知的具体原因；复用官方能力、现有组件与已有效证据，不为接入建立新框架。

Out of Scope：消费者订单、发货/履约、售后/退款/资金、广告、独立调价/改库存/商品删除；自动开官方应用或替商家授权；自动接受法律协议；第二套 Workbench/Product/Listing/Agent runtime/Tool Registry/IAM；旧数据迁移、旧 Service 包装、Cookie fallback；新验收平台或通用恢复服务。创建货品所需的初始价格、数量和仓库资料属于 M3/M4，后续经营 mutation 不因此进入范围。

Accepted Risks：本任务没有新增用户批准的风险豁免。官方资料版本可能滞后、实际资质可能变化，均以当次官方验证和 fail closed 处理，不能写成已接受的错误路由风险。真实商家/API、用户使用与生产验收均为 NOT_RUN。

## 2. 官方合同结论及完整性边界

来源、更新时间与完整站点摘录见 [官方合同证据](temu-official-contract-evidence.md)。以下区分事实与设计选择。

| 模式 | 官方创建/查询合同 | 路由 | 平台操作的单位 | 站点依据 |
| --- | --- | --- | --- | --- |
| 全托管 | bg.glo.goods.add / list.get / detail.get / bg.glo.product.search | PA；按当前迁移说明绑定 API | 一个供应商货品及其 SKC/SKU | 创建接口未见逐消费者国家选站参数；不得伪造逐国提交 |
| 半托管 | 同一供应商货品 API，另有 productSemiManagedReq、bindSiteIds、站点价格与仓库路径 | PA；身份来自 CN 应用，不能混用区域凭据 | 一个供应商货品，可一次绑定多个站点 | 官方字典 96 项，70 true、26 false；泛欧与澳新关联约束 |
| 本土 | temu.local.goods.v2.add；查询 temu.local.goods.list.retrieve；bg.local.goods.publish.status.get | US / EU / GLOBAL，匹配原应用分区 | 官方授权 mall 的一个商品 | 独立规格表 39 项；包含分区、siteId、语言、币种、单位；不是实时开站/商家可售清单 |

全托/半托的 PA 路由来自 2026-05-06 迁移说明，优先于 2026-01-05 通用分区页“发品 CN”的旧描述；未迁移 API 仍用 CN。官方称身份分区为 CN/US/EU/GLOBAL，PA 与 CN 的应用/授权入口相同。**不能据此推导 CN token 与 PA token、mallId 与 uniqueId 的等价关系。** 此关系按 §11 G1 收集官方依据。

区域 token 信息的 mallType=1 表示 SEMI、100 表示 LOCAL，没有全托管值；regionId 示例 USA-211，不等于 siteId=100。CN bg.mall.info.get 的 semiManagedMall 与 PA token 信息的 uniqueId 分别是不同接口事实，不能仅凭客户端“全托/半托”选项绑定。按 §11 G2 核实本土 mall 与站点的权威关联。

全部站点的含义是完整承接官方开放的目标与限制，不是允许发布到所有国家。半托 false 项保留为未开放；关联站点按官方要求作为一次操作；本土表不能从半托表复制（如半托韩国/巴西 false，而本土规格表有韩国/巴西）。全托管由平台分配消费者市场，创建供应商货品不证明所有国家上架；界面显示原平台事实及“由平台分配”，不补造消费者国家清单或逐国成功。

本土 V3 自动推断类目/属性是已存在的官方能力，不能复制一个本地推断平台。本批优先采用当前 Recommended V2 的显式资料合同，以便用户检查创建内容；不静默切换 V3 或把平台推断当作本地审批。V2 externalGoodsId 单店唯一、最大 40 字符，V3 同名字段规则不同，必须按版本绑定；externalGoodsId 与列表 outGoodsSn 的映射尚未证实，不用于自动 adopt。

“平台草稿”在本任务首先是当前 TargetRecord 中尚未提交的资料；列表支持 DRAFT 筛选不证明有远端保存草稿 mutation。只实现已证实的官方操作，不发明 save-draft endpoint。

## 3. 当前代码根因与职责

现有代码不是平台中立合同已经完成、只缺一个枚举值：

- Store normalizePlatform / readOfficialMaterial 只接受 SHEIN；OfficialMerchantCredential 是 OpenKeyID/SecretKey/SupplierID；QueryStore 的身份结果未用于持久化权威验证。
- OfficialApplicationRegistry 仅按 AppID 索引；runtime prepare 固定构造 SHEIN client。仅增加 TEMU 选项会造成跨平台/分区应用歧义。
- current TargetRecord、OfficialIntent/Receipt/Resolution 直接引用 SHEIN payload，验证固定 shein/shein-us；这些合格 owner 的实现需要窄增量，不是复制 TEMU 生命周期。
- SUB-K1 execution_kernel 与同 Product DB official UoW 已具备 permit/fence/UNKNOWN/finalize 能力，应复用；当前 fence 的 StoreID 与单站 SubjectID 必须适配供应商货品的真实副作用单位。
- Store 观察实现的现有 schema/DTO 是 SHEIN 专用；TEMU 商品读取不能冒充 SHEIN，也不能导入其消费者订单范围。
- internal/integration/temu 目前只有 README。internal/temu 的 Cookie/MALL_ID/task-first client 和旧定价策略（默认成本乘 1.5、legacy PricingRuleDTO）不是当前官方接入基础。

| 事实/职责 | 唯一 owner | 本批最小变化 |
| --- | --- | --- |
| 企业、成员、有效组织 | 当前 Identity/Organization/Store grants | 消费既有实时授权，无新 IAM |
| 店铺、连接、私密凭据、官方商家归属 | internal/storecenter | 平台/分区标签的凭据、验证事实与原连接状态；原 Store DB 事务 |
| Product / ApprovedAsset / source | 当前 Product/Asset owners | 读取版本固定的事实；不复制事实 |
| 平台规则/确定性资料校验 | internal/marketplace/temu | 按模式、站点、官方版本定义纯合同，不调用 HTTP/GORM |
| 平台资料 | internal/listing/record/target | 当前 owner 内增加明确 TEMU typed variant；不包装 legacy Draft-S1 |
| 提交/结果/恢复 | internal/listing/submission | 原内核、官方 intent/receipt/resolution/UoW；窄 typed variant |
| 官方协议、签名、HTTP、DTO 转换 | internal/integration/temu | 实现 domain-facing 窄 port；不拥有业务状态 |
| 远端商品观察 | 当前 Store observations owner | 原授权/同步/投影路径内增加 TEMU 产品类型；不扩订单 |
| 接线、BFF、Console、Tools | 当前 app/runtime、各 domain httpapi、现有 Console/Tool 消费者 | 平台选择与事实投影，无平台独立工作台 |

## 4. Contract → implementation → injection → consumer

调用顺序：Console/BFF 或受控工具 → 当前 Effective Organization/Actor/Member → Listing preparation/target 服务 → StoreProductAccess 取得当前商家绑定与短生命周期 server-only 执行句柄 → TEMU 官方 adapter 读取元数据或执行已批准操作 → 当前 TargetRecord/Submission UoW → 现有供应链与店铺商品投影。

Store 的 OfficialConnectionProvider 改为可区分平台、身份分区、授权方式的窄能力，提供 VerificationFacts（官方身份、模式、权限、有效期和证据版本）。Credential 内明确 SHEIN 与 TEMU 私密 variant，不把 access_token 塞入 OpenKeyID 或 SupplierID。应用选择身份为 platform + identityPartition + AppID；application revision 还绑定模式/授权方式/受控路由映射/配置版本，所有请求匹配原 attempt。不会建立一个可以任意路由、调用任意 API 的通用代理。

Listing 只消费无需密钥的 MerchantBinding：原 org/store/connection/application/service facts，加 platform/mode/identityPartition、官方 merchant identity hash、目标 site 或绑定 siteSet、rules/contract revision。Marketplace port 返回模式对应的元数据/校验结果与纯 payload；Integration port 仅暴露本批 read-rules/read-product/create-product/必要媒体操作，不提供任意 type 或 URL。

Runtime 在现有 currentapplication 注入配置好的 SHEIN/TEMU adapter、原 credential protection 和平台实现；配置缺失只让对应能力 unavailable，不回退其他平台/分区。现有 SHEIN DTO 保持明确类型，不更名为不受约束 map[string]any；共享 orchestration 分派已验证的 tagged variant，所有平台 case 都显式验证。

Console 复用 /workbench/stores、/workbench/store-products 和当前“我的供应链”资料/上传流程；站点/类型/状态来自 owner，不新增顶层 Listing Center。受控 diagnosis/readiness/propose 工具读取相同资料和规则，建议不自动批准、更换 target 或发起外部 mutation。后续 Commerce Tool 消费同一窄 port，不能绕过 Store/Submission。

## 5. 授权、身份与凭据边界

身份分区与 transport route 分开存：CN 的 migrated API 使用 PA，未迁移身份查询使用 CN；本土依原应用的 US/EU/GLOBAL 分区。API/路由白名单由官方版本绑定，禁止从客户端 URL、callback_host、mall 名称、地理位置猜测；不得在网关间探测 token 或遇错退旧 API。HTTP 禁止带密钥的跨 origin 重定向。

全托/半托采用官方手动 token 入口；本土支持官方手动 token，并只在当前应用确实开放 callback 时启用回调。私密输入仅送到本系统认证后的 Store TLS endpoint，限体积，忽略日志/body tracing，写入原 AES-GCM 保护边界，关联数据包含 org/store/attempt/application/partition。前端只能输入和查看非敏感状态，读取接口不回显 token/secret，Temporal/TargetRecord/日志不保存原始凭据。

手动安装不是 token exchange：在原连接 owner 内以原 attempt 和 exact private-input digest 保存密文，再只读核实；同 attempt 相同输入重放不会再建授权，不同输入冲突。未核实前不是 connected。回调沿原 durable exchange claim 一次发送；绑定 state hash、原成员/店铺、application revision、固定 callback URL 和允许的 callback_host。code 文档有效期 10 分钟不延长本系统原更短的 attempt 截止时间；丢失 exchange 响应保留 UNKNOWN，不能重发一次性 code。

验证原应用 scope、实际 expiredTime、商家身份与模式；CN/PA 身份关联和本土 site 关联缺证据时拒绝完成连接（见 G1/G2），不能拿配置或用户声明补齐。只申请本批需要的 API scope，不因文档“建议全选”扩大权限。缺 publish scope 可以明确展示只读可用/提交不可用，不冒充完整接入。

沿现有全局商家归属约束在 Store DB 原事务中占用官方 identity → org/store；TEMU identity 必须平台/官方身份分区加稳定 mall identity，不以 token、AppID、应用版本或客户端店铺名为身份。不同应用、重授权、断开后新建 Store 不得绕过同一官方商家的持久占用。这里的稳定 identity 选择以 G1/G2 的官方证明为先；未解决之前不正式建立索引。无凭据迁移、身份转移或跨企业自动接管流程。

每次规则/同步/提交/恢复重新消费实时成员和 Store grant、active service、官方连接、scope、有效期与版本。断开/撤权/重授权使新句柄失效；原已发送 receipt/UNKNOWN 仍由原 Submission 保存和核实，不恢复旧连接、不改其原商家归属。同步只给读产品目的，不能通过复用 publish purpose 提权。

## 6. 确定性资料与全部站点

规则快照绑定 mode + verified merchant + official identityPartition/route + site/siteSet + category + API/spec revision。类别、属性、规格、品牌、尺寸表、图像和物流模板使用官方当前接口与本批必要 scope；源 URL 与时间/内容 hash 随快照保存，失败/缺失显示不可核实，不用历史 fixture 伪装实时官方规则。

半托按完整字典保留站点/开站/关联组；泛欧规则与 103/104 澳新组按官方 mode/资质约束形成同一个 payload。列表 70 true 是文档快照，实际店铺/类目/仓库资格仍须官方验证。已绑定远端站点是观察事实，不因旧文档 false 就抹去；新增提交资格也不能仅由历史远端 bindSites 推定。

本土使用独立 39 项规格表，不由国家的洲、欧盟成员身份或币种推导网关；保留 Belgium=GLO、Türkiye/AE/SA=EU 等原文值，同时与原授权应用分区核对，冲突拒绝发送，待官方依据修正。mall regionId 不是 siteId；没有可靠映射不能用 211 当 100，也不能由用户选择站点绕过验证。

全托 target 是供应商货品，不创建虚构的国家 target。提交后平台各市场状态仅在官方返回明确归属时投影，未知国家状态保持未知。全部三种模式的范围仍保留；未开站或本商家无资质的组合说明官方限制，不宣称普遍可用。

资料可固定用户明确提供的价格/币种、初始数量/仓库、产地、品牌授权、证书/合规等事实；不采纳旧成本乘 1.5，不默填价格/数量/法律陈述。半托 supplierPrice 单位按 CNY 分/USD cent，不能套用本土站点币种规则；本土 JPY/KRW/NOK 的文档整数约束与各站点精度独立验证。ApprovedAsset 绑定内容 hash/version 和安全 URL，图片 URL 不得临时指向其他资产。

productComplianceStatementReq 等协议字段只有已有明确合法的批准依据时消费，否则待补全并提示官方前置；不在系统中自动签署。官方图片上传/转换若有副作用，沿 §8 原内核记录，纯资料适配不得暗中调用它。平台直接抓取图片也属于 create 操作效果，不单独假造本地成功。

TargetRecord 保存精确 source、Product/Asset revisions、mode、siteSet、merchant/application binding、rule hash、明确资料与校验结果；准备后任一漂移必须生成/检查新资料版本。Upload 精确绑定 saved record/hash、已有 approval、payload hash 和原 actor/member；不同版本不能复用旧批准或 SendPermit。

## 7. 持久化与事务

Store DB：现有 Store record、connection、attempt、credential protection 和 merchant-binding owner；新增平台验证事实及必要唯一约束仍由该 owner 保存。记录密文与完成验证、归属占用原子提交；失败回滚不能出现 connected 但无已验证凭据。

Product DB：TargetRecord、official intent、执行 reservation/fence、official receipt 与 qualified resolution 继续用原同库 UoW。新增 TEMU tagged body 没有第二表系或不同队列 owner。不可变 intent 仅引用/摘要 canonical facts；不复制可更新 Product。原 finalize 必须同时保存必要远端标识与 outcome，否则不投影已上传。

只读远端商品观察保留 org/store/official merchant/connection revision、sync owner、observedAt、原文状态/远端标识/必要详情。沿现有观察 owner 的事务和 bounded pagination；增加明确 TEMU product variant，不能写入 shein_observations 模式专用 JSON 伪装 SHEIN，不重构无关订单/物流。sync 状态与 Submission 状态不是同一事实。

跨 Store DB、Product DB、HTTP 不持有分布式事务；先实时 Store authorize，再 Product UoW claim，发送前重新检查原短期 binding。任何 Product DB commit 不确定都不能取得第二 permit。HTTP 之后授权变化不修改已发生的平台事实，原 owner 保留证据，新的读取仍按当前访问控制。复用既有 Temporal 执行/恢复入口，不能新建 TEMU reconciler、outbox 平台或业务队列。

全新安装更新当前 schema 定义与 guard；不写旧数据迁移/双读/双写。任何现场兼容义务须另有用户决定，不由本设计推导。

## 8. 副作用、稳定幂等与 UNKNOWN

商品副作用 fence 仍是原内核 Organization + Platform + Store + SubjectID。Store 全局 verified merchant binding 防止相同商家改 Store 绕过。SubjectID 以 canonical ProductKey、官方商家稳定 identity、provider-effect namespace 确定：

- 全托/半托：supplier-product 一个单位，站点集合、站点顺序、模式参数、TargetRecordID、revision、operation、AppID/版本、token 轮换不能进入可改变的 fence 身份。它们属于 intent/payload fingerprint；改变它们不能绕过未解决 UNKNOWN 或已成功创建。
- 本土：local-goods + canonical ProductKey + verified mall 的官方站点身份。原连接不能通过更换客户端站点生成第二目标；站点漂移拒绝原 Store binding。
- 图片：独立 image-effect namespace + exact Asset content hash + transformation/action + merchant；不随批次/record 变化；成功结果复用需当前授权，UNKNOWN 不重做。

StoreID 仍用真实 native StoreID，不能塞 merchant hash 冒充 StoreID。稳定原文外部编号由既有 canonical 商品/SKC/SKU 身份与 merchant 生成并受官方长度约束；不是每次请求随机生成。它是 correlation，**不是官方幂等承诺**，不自行添加 provider idempotency header。

唯一流程：live reauth → exact source/record/approval/rules → Store handle → 原 intent/UoW → 首次 committed SendPermit → 发送前原 binding 核验 → 一次官方 mutation → 同 UoW finalize。

| 事件 | 持久化/行为 |
| --- | --- |
| 同 intent key、同 payload | 返回原记录/结果，不新发送 |
| 同 key、不同 payload/binding | 冲突；不能覆盖 |
| 不同 key、同 supplier effect | 原稳定 target fence 拦截，siteSet/应用变更也不能绕过 |
| 首次 claim/commit 不确定 | 无第二 permit；沿原恢复查已提交记录 |
| 明确成功响应 | 校验 success、必要 product/goods/SKC/SKU 标识、当前请求对应；原子 receipt + outcome |
| 未识别业务码、超时、空/截断响应、发送后取消/重启/lease 过期 | 原 attempt UNKNOWN，不能 blindly retry |
| 可证实的拒绝且无写入 | 仅明确、经过官方合同证明的码允许 definitive-no-effect；初始未知码全保守 UNKNOWN |
| 断开/改应用/改资料/改站点 | 不清除原 fence；新请求拒绝漂移或返回原结果 |

成功只表示创建/上传被官方确认，不表示审核通过或在售。bg.glo.product.search/列表 site status 和本土 publish.status.get 是后续独立读取。

PA list.get 支持 skcExtCode 并返回 productId/SKC/SKU 编号、外部编号、bindSites，detail.get 主要是基本信息/类目/详情素材，**未发现其完整回显 SKU/价格/body hash**。恢复只能在当前原商家、固定 correlation、单一 product 和全部预期 SKC/SKU/绑定站点均有可验证证据时采用；不足则 UNKNOWN。不能把标题相似、单个 SKU 匹配或 detail HTTP 200 当 exact payload 证明。

本土 publish.status.get 要求已知 goodsIdList，不能解决创建响应丢失且无 goodsId。列表 outGoodsSnList 能检索外部编号，但 externalGoodsId → outGoodsSn 尚未有明确官方映射依据；未证实前不得以此自动 adopt。正常成功后的 goodsId 可用于状态回读；无法 qualified 核实时保留原 UNKNOWN，不新增人工流程平台。

当前用户输入 SHEIN SPU 的核实入口（internal/app/supplychain/upload_resolution.go）仍由服务器重新查询官方资料、核对原 intent/record/merchant/完整 correlation，再生成短期 OfficialResolution proof。不是用户声明成功即可裁定；通用内核的 EvidenceManualResolution 也不证明已有可直接开放的 TEMU 人工裁定路径。TEMU 仅在本批 typed variant 具有同等 qualified 官方证据时复用该核实流程；输入远端 ID 只是查询线索，截图、口头结论、相似标题与未证明的字段映射都不能解除 fence。现有证据不足时持续 UNKNOWN，不提供直接“标为成功/失败/重试”的 TEMU 入口。

查不到、权限丢失、未进入在售列表、重复匹配均不是未发生操作的证据。恢复只做原只读核实/原 UoW resolution，不能创建新 intent、重上传图片或重新发送货品。原始 code/token/签名字符串不进入 receipt。

## 9. 资源、协议与组件复用

复用 Go net/http、crypto/md5、标准 JSON 与仓库现有 bounded transport/credential protection/UoW/Temporal。官方提供签名规则与 Python 示例，未在本轮找到经验证适用于当前全部分区/模式的官方 Go SDK；不下载未知 SDK、不建设自有 HTTP 框架。签名按官方 outer ASCII key 排序、精确 wire value、secret 前后拼接、MD5 大写；测试使用固定公开 fixture，日志不保存签名前字符串。

HTTPS 固定官方 origins；禁止任意 type/host、跨 host redirect、无限分页/自动重试。单个请求/response 最大 2 MiB（与当前 execution payload 上限一致），读取采用 max+1 检测超限，超限不部分采纳。身份验证沿现有 5 秒总 deadline；其他单次官方请求最多 10 秒，提交/核实沿原 caller deadline 取更短值。

Provider 页大小不得超过官方上限（本土 retrieve、product.search 100）；分页串行、原 bounded sync/recovery budgets 内检查 ctx、重复 token/页、数量和总数据量。达到预算保留“未完成/未核实”而不是完整成功。按官方当前 quota 对各 app/API 限速，不猜全平台 QPS；只读重试在原 deadline/budget 内，mutation 无自动重试。

平台响应原文只保留必要 public identifiers、状态、error/trace 和 response hash；错误文本先截断并去凭据，秘密载荷不作为供前端调试的“原始请求”。官方要求提供 debug 请求不是泄漏 secret 的授权。

## 10. 实施顺序与必要验证（准入之后）

一个 Writer、一个 branch/worktree、一个主要 PR 连续交付全范围。依赖先后为：Store 官方身份/应用/平台 contract → TEMU bounded transport 与官方规则资料 → 当前 TargetRecord/Submission typed variant 与 UoW → 三模式提交/恢复和商品观察 → 原 Console/Tools 接线。切片是可验证提交边界，不机械拆 PR；不能先上线一个平台选项冒充已支持。

当前 #29/#33 未关闭，不把历史退出清单作为新 owner 的永久 facade 许可，也不机械等待无关全部重构。实际相关依赖是当前 Store grant/service/官方连接、Product/Asset 授权、TargetRecord/UoW/内核和 Console 路径；这些在调查 main 已存在，但 SHEIN 耦合须按本设计最小补齐。正式实现前刷新 main 与活跃 Store/Console Writer；未准入不改共享生产路径。

业务实现先做能捕获目标错误的 RED，再最小实现和必要自检。沿现有测试验证当前 Must，不开发新 runner/故障注入平台：

| Must | 实现中必要证据 |
| --- | --- |
| M1 | 完整 96 项半托字典、39 项本土规格事实；true/false/关联组、按分区 API 映射、未证明组合拒绝；三模式实际 DTO fixtures |
| M2/M6 | 跨企业/店铺、AppID 同名不同平台/分区、原 attempt/member 绑定、错误模式/site、缺 scope/过期/撤权、secret 无投影；断开/轮换不重新归属 |
| M3 | 缺品牌/产地/证书/价格不补造；精度/单位/规则/Asset/approval 版本漂移拒绝；exact saved payload 匹配 |
| M4 | 两并发 permit、不同 siteSet/operation/app/record 仍同 fence；响应丢失/commit unknown/取消/重启不重发；图片独立 effect；同 key 异载荷冲突 |
| M5 | 上传回执与审核/在售分离；未知错误不成功；列表无结果/多匹配不 definitive；状态 query 必需 ID；成功 receipt 原子保存 |
| M6 | 实際 Console 与 tool 调用同一 owner，server-only 凭据，当前 grant/服务校验，readonly sync 不取 publish 权限 |

代码自检、fixture 和 PR CI 仅是工程证据；实际官方 application 可用/商家权限、真实发布与用户路径验收单独 NOT_RUN。完成实现后给用户运行入口与操作步骤，不能由作者签发产品验收或生产可用。必需 CI/独立最终交付检查/合并授权仍单独满足。

## 11. 准入缺口、评审与停止点

下列是实际官方身份/站点合同缺口，不是理论 hardening；调查结论不冒充已完成全覆盖核实。

| ID | 缺少的具体依据 | 受影响 Must / 后果 | owner / 解锁条件 / 层级 |
| --- | --- | --- | --- |
| G1 | 新授权 PA token 的 uniqueId 与 CN mallId/半托标记的权威关联；同一当前 token 是否能按 scope 对这两个网关核实 | M2/M4：误绑定店铺模式或不稳定商家 identity 将绕过原 fence；没有身份验证完整路径 | 本 Writer 收集 TEMU 官方依据；明确同凭据验证合同或官方映射接口后固定单一身份，独立评审；正式身份/提交实施前 |
| G2 | 本土 token regionId（USA-211）与 siteId 的权威关联、全量当前可售/授权 site 来源；39 项规格表没有开站字段，局部路由不能靠地理补全 | M1/M2：无法证明当前 mall 对应选择站点；误路由/错站点声明或全部站点遗漏 | 本 Writer 核实官方 merchant-site/catalog 合同及分区冲突处理；不能由用户填写 siteId 自证授权；正式连接/全覆盖实施前 |
| G3 | externalGoodsId 与 outGoodsSn 明确映射、PA exact recovery 证据覆盖 | M4/M5：盲目 adopt 错误结果；当前可保留 UNKNOWN 避免错误副作用 | IMPLEMENTATION_TEST；已知成功 ID 可读回、无法 qualified 时保持 UNKNOWN；实施/真实验证层处理，不授权自动重发 |

第一轮独立评审分类：G1/G2 为 BLOCKER（当前官方身份/站点缺口使核心连接路径无法完成，G1 同时影响身份所有权与稳定 fence）；G3 为 IMPLEMENTATION_TEST，证据不足不 adopt/不重发的设计可保留。店铺服务沿当前 ServiceLifecycleExecutor 与 orgresource store_service_v1 reservation/receipt，未发现平台专属新计费依赖。准确候选 SHA、Reviewer 证据与增量复核写主 PR #655；此分类没有签发 READY 或产品验收。

未取得 G1/G2 官方依据前，当前状态 NOT_READY / Backlog，不进入正式生产开发。登录文档已可访问，不再以登录作为 blocker；本轮未读取应用密钥或商家数据，也未执行官方业务 API。可以继续只读查证与设计评审，不能用真实调用试错取代未授权商家验证。

Architecture Review：第一轮由独立只读 Reviewer 核对原始范围/Threat Model、代码实际 owner 与全部设计 diff，结论 NOT_READY；后续只复核实际相关增量，最多两轮正常评审。finding 先分类为 BLOCKER / IMPLEMENTATION_TEST / BACKLOG / ACCEPTED_RISK / NOT_APPLICABLE，列明受影响 Must、具体后果与层级。只有 G1/G2 等实际核心路径/身份/不可重复副作用条件未满足时阻止开工；非 Blocker 不扩成新的通用机制。

只有官方依据与实际设计闭环、独立评审明确 IMPLEMENTATION_READY、当前 Issue Design Basis/Ready 记录完成后，才冻结本文件并允许生产实现。评审与准入证据写 #31/主要 PR，不由作者自行改 READY，不以 CI/文档存在代替准入。

## 12. Legacy decision

Legacy decision：EXTRACT | RETIRE。

Reusable behavior：已符合当前事实/授权/事务合同的 Store、Product/Asset、TargetRecord、SUB-K1、UoW 与确定性工具复用；旧平台规则仅在当前官方依据逐条成立时提取纯行为。

Current owner：本文件 §3 对应 owner。旧 Cookie client/task-first/management/service DTO 与无产品依据定价不参与新调用链。

Cutover/deletion condition：本批新正式路径不得依赖 internal/temu Cookie/旧 listingadmin；相关当前调用切到正确 owner 后移除旧消费者。无 fallback/双读/双写/同步或 compatibility 例外；无关历史遗留不以本任务扩大清理范围。
