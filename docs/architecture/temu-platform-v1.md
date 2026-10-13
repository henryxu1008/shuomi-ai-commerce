# TEMU 三种模式与全部官方站点接入设计

> Execution：[#31](https://github.com/qq550723504/task-processor/issues/31)
>
> Design Basis：Independent Architecture
>
> Status：IMPLEMENTATION_READY / FROZEN；第二轮独立相关增量评审绑定 4d4bc4feff7eff67fb9ea8a9c067cdc21f7db1ef。2026-10-12 新读 E41 形成 G4，本土图片7天有效期的 §8.1 最小修正已由原独立 Reviewer 绑定 72696b1bbe02bc6cd7ce0417627e8e98dba4f129 复核为 IMPLEMENTATION_READY，无新增设计 BLOCKER。对应原子准入/多代回执要求为 IMPLEMENTATION_TEST。GE/AM 规格仍阻塞受影响发布及全覆盖发布验收；2026-10-13 用户批准准备代码先合并，见下方阶段决定，不构成产品验收。
>
> 调查基线：main @ f1d77a84fa653813235ddbfeedac59ea6bb0a94c，2026-10-11
>
> 本文是已独立准入的设计基线；准入时只有文档，正式实现必须遵守本合同和实际相关依赖。

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
| 本土 | temu.local.goods.v2.add；查询 temu.local.goods.list.retrieve；bg.local.goods.publish.status.get | US / EU / GLOBAL，匹配原应用分区 | 官方授权 mall 的一个商品 | E24 规格 39 项、E30 LOCAL=true 39 项；两集合不同，37 项交集、GE/AM 缺完整价格规格；不是实时商家可售清单 |

全托/半托的 PA 路由来自 2026-05-06 迁移说明，优先于 2026-01-05 通用分区页“发品 CN”的旧描述；未迁移 API 仍用 CN。进一步读取当前官方接口清单 E26，基础组同时明确 CN bg.mall.info.get 查询当前 token 的店铺类型、PA info.get.global 查询当前 token 的店铺 uniqueId/权限；E04 定义 token 为店铺粒度，E11 定义 uniqueId 为 mall 唯一标识。因此选 **PA uniqueId 作为 supplier 的 canonical merchant identity**，在同一 exact app/token 的验证过程中读取 CN 类型，不要求 mallId 与 uniqueId 等值或一张转换表。具体验证是设计选择，依据为这些官方合同；真实 token 的 scope/成功响应仍须实现验证，失败不连接。

区域 token 信息的 mallType=1 表示 SEMI、100 表示 LOCAL，没有全托管值；regionId 示例 USA-211，不等于 siteId=100。CN bg.mall.info.get 的 semiManagedMall 必须是该 exact token 的实际响应，不能仅凭客户端“全托/半托”选项绑定。区域接入本批只接受 LOCAL；不能把同一半托商家作为另一份区域身份重新接入。2026-10-12 已在用户登录后的 GLOBAL 官方 Model Site and Region（E30）读取完整映射：US 211→100、CA 37→101、BE 20→142；不再要求用户提供已存在的字段对照表。实际文档冲突及 GE/AM 剩余价格规格见 §6/§11。

全部站点的含义是完整承接官方开放的目标与限制，不是允许发布到所有国家。半托 false 项保留为未开放；关联站点按官方要求作为一次操作；本土表不能从半托表复制（如半托韩国/巴西 false，而本土规格表有韩国/巴西）。全托管由平台分配消费者市场，创建供应商货品不证明所有国家上架；界面显示原平台事实及“由平台分配”，不补造消费者国家清单或逐国成功。

本土 V3 自动推断类目/属性是已存在的官方能力，不能复制一个本地推断平台。本批优先采用当前 Recommended V2 的显式资料合同，以便用户检查创建内容；不静默切换 V3 或把平台推断当作本地审批。V2 externalGoodsId 单店唯一、最大 40 字符，V3 同名字段规则不同，必须按版本绑定；externalGoodsId 与列表 outGoodsSn 的映射尚未证实，不用于自动 adopt。

2026-10-12 新读 E42 确认 V2 也可能由平台补填/转换缺失或不合格信息，并在成功响应返回要求前往 Seller Center 确认的 warnings。按既有 M3/M5 保留批准 payload 与官方警告；本地仍拒绝缺失事实/规格，不依赖补填通过。成功只确认创建，不能声称远端全部字段等于批准内容、已审核或在售；没有已证实 opt-out 参数，不自行增加。V2 自定义规格使用当前模板的 parentSpecId/specName，不先调用独立规格 create；模板必须重读，不能将观察到的旧 variationType=0 强改为2。该合同补充属于 IMPLEMENTATION_TEST（结果投影/提交切片门槛），不改变 owner/状态/授权边界或重开冻结设计。公告的 UDP 未有本批应用分区依据，不猜网关。

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

Store 的 OfficialConnectionProvider 改为可区分平台、身份分区、授权方式的窄能力，提供 VerificationFacts（官方身份、模式、权限、有效期和证据版本）。Credential 内明确 SHEIN 与 TEMU 私密 variant，不把 access_token 塞入 OpenKeyID 或 SupplierID。应用选择身份为 platform + identityPartition + AppID；application revision 绑定授权方式、支持的模式集合、受控路由映射和配置版本，所有请求匹配原 attempt。SHEIN 既有应用类型仍是其专属事实；TEMU CN 的同一应用支持全托/半托，实际 mode 来自官方店铺验证，不把当前 SHEIN 单一 application mode 原样套用。不会建立一个可以任意路由、调用任意 API 的通用代理。

Listing 只消费无需密钥的 MerchantBinding：原 org/store/connection/application/service facts，加 platform/mode/identityPartition、官方 merchant identity hash、目标 site 或绑定 siteSet、rules/contract revision。Marketplace port 返回模式对应的元数据/校验结果与纯 payload；Integration port 仅暴露本批 read-rules/read-product/create-product/必要媒体操作，不提供任意 type 或 URL。

Runtime 在现有 currentapplication 注入配置好的 SHEIN/TEMU adapter、原 credential protection 和平台实现；配置缺失只让对应能力 unavailable，不回退其他平台/分区。现有 SHEIN DTO 保持明确类型，不更名为不受约束 map[string]any；共享 orchestration 分派已验证的 tagged variant，所有平台 case 都显式验证。

Console 复用 /workbench/stores、/workbench/store-products 和当前“我的供应链”资料/上传流程；站点/类型/状态来自 owner，不新增顶层 Listing Center。受控 diagnosis/readiness/propose 工具读取相同资料和规则，建议不自动批准、更换 target 或发起外部 mutation。后续 Commerce Tool 消费同一窄 port，不能绕过 Store/Submission。

## 5. 授权、身份与凭据边界

身份分区与 transport route 分开存：CN 的 migrated API 使用 PA，未迁移身份查询使用 CN；本土依原应用的 US/EU/GLOBAL 分区。API/路由白名单由官方版本绑定，禁止从客户端 URL、callback_host、mall 名称、地理位置猜测；不得在网关间探测 token 或遇错退旧 API。HTTP 禁止带密钥的跨 origin 重定向。

全托/半托采用官方手动 token 入口；本土支持官方手动 token，并只在当前应用确实开放 callback 时启用回调。私密输入仅送到本系统认证后的 Store TLS endpoint，限体积，忽略日志/body tracing，写入原 AES-GCM 保护边界，关联数据包含 org/store/attempt/application/partition。前端只能输入和查看非敏感状态，读取接口不回显 token/secret，Temporal/TargetRecord/日志不保存原始凭据。

手动安装不是 token exchange：在原连接 owner 内以原 attempt 和 exact private-input digest 保存密文，再只读核实；同 attempt 相同输入重放不会再建授权，不同输入冲突。未核实前不是 connected。回调沿原 durable exchange claim 一次发送；绑定 state hash、原成员/店铺、application revision、固定 callback URL 和允许的 callback_host。code 文档有效期 10 分钟不延长本系统原更短的 attempt 截止时间；丢失 exchange 响应保留 UNKNOWN，不能重发一次性 code。

全托/半托验证固定流程：在原 CN 应用/同一密封 token 下，PA info.get.global 读取非空 uniqueId、实际 expiredTime 与 scope；CN bg.mall.info.get 读取明确的 semiManagedMall（true=半托、false=全托，字段缺失不能默认为 false）。两个请求均须官方成功、同一原 attempt/credential digest，才能生成 supplier VerificationFacts；PA uniqueId 是身份，CN mode 是附属事实，不读取或合并另一 token。缺 CN 类型查询或 PA 身份查询权限拒绝完成连接；其他 publish/rules/read scope 各按实际能力核验。不存在失败后换 token/应用/网关的 fallback。

本土连接只使用原 US/EU/GLOBAL 应用的 info.get，验证 mallId、mallType=LOCAL、regionId、实际 expiredTime 与 scope；官方 mall 身份的归属不依赖把 regionId 强转 siteId。规格字典缺失/不匹配时，连接身份与只读事实仍可独立成立，但该目标的发布准备不宣称 ready；G2 的剩余依据只影响价格/单位规格及全部站点的发布闭环。只申请本批需要的 API scope，不因文档“建议全选”扩大权限。缺 publish scope 可以明确展示只读可用/提交不可用，不冒充完整接入。

沿现有全局商家归属约束在 Store DB 原事务中占用官方 identity → org/store；TEMU supplier 身份为 platform + supplier namespace + PA uniqueId，本土为 platform + 官方身份分区 + local namespace + mallId。mode/siteSet/regionId/spec profile、token、AppID、应用版本和客户端店铺名均不进入可变身份。不同应用、重授权、断开后新建 Store 不得绕过同一官方商家的持久占用；重验证 uniqueId/mallId 变化拒绝原绑定，不能把它当作正常 token 轮换。区域 SEMI 不走 Local 接入，防止同一半托身份出现第二 namespace。无凭据迁移、身份转移或跨企业自动接管流程；独立准入前不正式建立索引。

每次规则/同步/提交/恢复重新消费实时成员和 Store grant、active service、官方连接、scope、有效期与版本。断开/撤权/重授权使新句柄失效；原已发送 receipt/UNKNOWN 仍由原 Submission 保存和核实，不恢复旧连接、不改其原商家归属。同步只给读产品目的，不能通过复用 publish purpose 提权。

## 6. 确定性资料与全部站点

规则快照绑定 mode + verified merchant + official identityPartition/route + site/siteSet + category + API/spec revision。类别、属性、规格、品牌、尺寸表、图像和物流模板使用官方当前接口与本批必要 scope；源 URL 与时间/内容 hash 随快照保存，失败/缺失显示不可核实，不用历史 fixture 伪装实时官方规则。

半托按完整字典保留站点/开站/关联组；泛欧规则与 103/104 澳新组按官方 mode/资质约束形成同一个 payload。列表 70 true 是文档快照，实际店铺/类目/仓库资格仍须官方验证。已绑定远端站点是观察事实，不因旧文档 false 就抹去；新增提交资格也不能仅由历史远端 bindSites 推定。

本土操作的目标始终是 exact token 授权的 mall，当前 V2/列表/类目查询按该店铺执行，不能增加客户端 siteId/mallId 来切换目标。按以下固定流程消费官方字典；官方字典只确定格式和一般地区说明，当前店铺可售类目、资质、运费模板和 API 权限须从同一授权 mall 查询，不发明通用“全部 merchant 可售站点”接口。US-to-CA 等 tag 不产生第二创建目标或自动授权后续经营操作。

1. 使用当前验证的 regionId 查询 E30 的版本固定映射（87 行、有 83 个不同 siteId），再按得到的 siteId 查询 E24 的完整价格/语言/包装规格。不按地区名、洲、法币、邻国或客户端选项匹配；允许多个 region 映到 site 100，不以 site 反向制造唯一 mall region。235 行中的 148 行没有 siteId，保持未核实，不新增可发布目标。
2. 身份分区/HTTP route 始终由原应用决定。E30 的部署区用于核对地区事实，不能改写 application binding；不一致时停止该目标发布核实，不携带 token 探测另一网关。E24 的 Site Region 不参与凭据路由。Belgium 规格表 GLO 与较新 E30 部署 EU 保留来源差异；采用原 EU 应用时可按 E30 核对地区，并按 E24 的同一 site 142 读取 EUR/g/cm 等规格，不能按旧 GLO 标签换 GLOBAL 网关。
3. 分字段、用途和版本保留来源：E23/E13 是 PA 半托发布合同，E30 是区域 site/region 与一般交易/履约说明。两表 SEMI 标记和 North Macedonia 的 192/160 不一致时分别保存，不把不同接口值 alias，不由区域 SEMI 开放值建立第二商家 namespace，也不扩大半托 bindSiteIds。对当前操作缺唯一官方依据的字段保持未核实。
4. 一般开放说明、规格覆盖、当次 merchant scope/资格是三个独立事实。E30 的 true 不能替代授权，空白不能转换为 false 或 true，coming soon 不变成已支持。E24 SA 120/LU 152 有规格、E30 LOCAL 为空白，不能仅凭规格宣称开放；当次官方 mall/类目/权限的验证与字典来源一起记录，不能据历史商品或名称猜资格。
5. E30 LOCAL=true 39 项与 E24 39 项只有 37 项交集；GE region 75/site 165、AM region 10/site 166 尚缺已读取的完整价格规格，保留在本批范围，当前资料显示缺官方规格，不批准/不发送。两个集合的并集 41 项均可表达各自事实，不能只按“39 行”宣称全覆盖、不能自行删除 GE/AM。当前 E29 fieldRules 只有 required/optional/disabled，V2 currency/amount/packageInfo 为字符串，不能假装它返回缺失规则；V3 price 仍明确引用 E24，切 V3 不解决该币种/精度缺口。本批全覆盖发布验收仍须补齐这些目标的官方依据。
6. 规格快照固定原文 precision/range/currency/units 与 hash；价格以规范化十进制字符串或精确整数表示，禁止浮点舍入或静默单位/汇率转换。用户明确资料的单位与绑定规格不一致时待补全；规则版本/region/profile 漂移使资料重新校验，但不会释放 §8 的同 mall + Product fence。

全托 target 是供应商货品，不创建虚构的国家 target。提交后平台各市场状态仅在官方返回明确归属时投影，未知国家状态保持未知。全部三种模式的范围仍保留；未开站或本商家无资质的组合说明官方限制，不宣称普遍可用。

资料可固定用户明确提供的价格/币种、初始数量/仓库、产地、品牌授权、证书/合规等事实；不采纳旧成本乘 1.5，不默填价格/数量/法律陈述。半托 supplierPrice 单位按 CNY 分/USD cent，不能套用本土站点币种规则；本土 JPY/KRW/NOK 的文档整数约束与各站点精度独立验证。ApprovedAsset 绑定内容 hash/version 和安全 URL，图片 URL 不得临时指向其他资产。

2026-10-12 原独立 Reviewer 绑定 a9224b1bbb576b55d94fa2b497178aa30cc806f6 澄清：M3不要求本地预测创建接口的全部动态币种、申报价上限及经营资格判定。当前身份/scope、当前可选类目/属性、已批准的明确事实及 E13/E27/E45 可核实字段合同必须成立；动态限制由对应官方响应作为 authority，沿原结果/UNKNOWN 合同处理。未证实存在的通用 prequalification API 不能成为所有供应商永久拒绝的新增前置（NOT_APPLICABLE）。现有两个无条件保护仍须被具体实现检查替代，完整 supplier 路径目前 NOT_READY，归 IMPLEMENTATION_TEST 而非新架构 BLOCKER。E44 supplierId 来源缺口只阻止受影响具名查询，不猜身份映射；E43不套本土7天期限、不声称永久有效。此澄清不开放未核实站点、缺失事实、自动合规签署或盲重发。

productComplianceStatementReq 等协议字段只有已有明确合法的批准依据时消费，否则待补全并提示官方前置；不在系统中自动签署。官方图片上传/转换若有副作用，沿 §8 原内核记录，纯资料适配不得暗中调用它。平台直接抓取图片也属于 create 操作效果，不单独假造本地成功。

TargetRecord 保存精确 source、Product/Asset revisions、mode、siteSet、merchant/application binding、rule hash、明确资料与校验结果；准备后任一漂移必须生成/检查新资料版本。Upload 精确绑定 saved record/hash、已有 approval、payload hash 和原 actor/member；不同版本不能复用旧批准或 SendPermit。

2026-10-12 最终交付检查继续使用冻结设计。供应商具名 consumer 已用当前类目/预设规格/尺码表、E43 原图片回执与当次 artifact 证明替代历史的无条件待核实保护。独立 Reviewer 对本土预设规格缺省 parent 的成功关联、半托 E13 固定备货时长和尺码表跨页编辑三项均分类 IMPLEMENTATION_TEST，修复后相关增量复核 PASS，无新增 BLOCKER；不重开原 owner/事务/UNKNOWN 边界。已有尺码表使用 E47 有界页与当前 settings，`supplierSizeOffsets` 只作读取提示；保存与实际发送前仍通过原 RuleReader 重读全部所选页和完整 rule hash，缺页/重复/漂移/撤权拒绝，不把客户端提示或缓存当证明。E17 可选外部 SKU 编号未提供时只使用唯一原规格关联，明确编号仍精确匹配；不推定平台的缺省编号生成行为，不改写批准 wire。E13 的259200秒仅限定制商品，其资格由当次官方响应决定。

## 7. 持久化与事务

Store DB：现有 Store record、connection、attempt、credential protection 和 merchant-binding owner；新增平台验证事实及必要唯一约束仍由该 owner 保存。记录密文与完成验证、归属占用原子提交；失败回滚不能出现 connected 但无已验证凭据。

Product DB：TargetRecord、official intent、执行 reservation/fence、official receipt 与 qualified resolution 继续用原同库 UoW。新增 TEMU tagged body 没有第二表系或不同队列 owner。不可变 intent 仅引用/摘要 canonical facts；不复制可更新 Product。原 finalize 必须同时保存必要远端标识与 outcome，否则不投影已上传。

Asset DB：ApprovedAsset、批准 receipt/current inventory head 仍由原 Asset owner 的独立 `SupplyChainDependencies.AssetDB` 保存；不复制到 Product DB、不增加 Product serving 角色的 Asset 表权限。当前分库接线依据为 `internal/app/httpapi/supply_chain_application.go` 的独立 AssetDB reader/approval 构造与 productDB official/kernel 构造。

只读远端商品观察保留 org/store/official merchant/connection revision、sync owner、observedAt、原文状态/远端标识/必要详情。沿现有观察 owner 的事务和 bounded pagination；增加明确 TEMU product variant，不能写入 shein_observations 模式专用 JSON 伪装 SHEIN，不重构无关订单/物流。sync 状态与 Submission 状态不是同一事实。

跨 Store DB、Product DB、HTTP 不持有分布式事务；先实时 Store authorize，再 Product UoW claim，发送前重新检查原短期 binding。任何 Product DB commit 不确定都不能取得第二 permit。HTTP 之后授权变化不修改已发生的平台事实，原 owner 保留证据，新的读取仍按当前访问控制。复用既有 Temporal 执行/恢复入口，不能新建 TEMU reconciler、outbox 平台或业务队列。

全新安装更新当前 schema 定义与 guard；不写旧数据迁移/双读/双写。任何现场兼容义务须另有用户决定，不由本设计推导。

## 8. 副作用、稳定幂等与 UNKNOWN

商品副作用 fence 仍是原内核 Organization + Platform + Store + SubjectID。Store 全局 verified merchant binding 防止相同商家改 Store 绕过。SubjectID 以 canonical ProductKey、官方商家稳定 identity、provider-effect namespace 确定：

- 全托/半托：supplier-product 一个单位，站点集合、站点顺序、模式参数、TargetRecordID、revision、operation、AppID/版本、token 轮换不能进入可改变的 fence 身份。它们属于 intent/payload fingerprint；改变它们不能绕过未解决 UNKNOWN 或已成功创建。
- 本土：local-goods + canonical ProductKey + verified mall identity 一个单位；siteId、regionId、规格 profile 不是可变 fence 身份。接口目标是 token 授权 mall，更换这些资料不能产生第二创建目标；原 region/绑定漂移拒绝发送。
- 图片：独立 image-effect namespace + exact Asset content hash + transformation/action + merchant；不随批次/record 变化；成功结果复用需当前授权和未过期，UNKNOWN 不重做。本土临时链接沿 §8.1 同一个 fence 的已知成功续期，不生成第二身份。

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

### 8.1 本土图片临时链接（G4 已独立准入增量）

Finding：E41 已把 E40 的成功返回地址改成7天临时链接；现有成功永不再取 permit 的图片设计不能完成“保存资料超过7天后提交”。Product requirement affected：M3/M4。Classification：BLOCKER，命中“核心 happy path 按当前设计无法完成”，层级仅为本土图片发送/成功结果复用实现；不重开 Store 授权、站点字典或其他冻结边界。Action：在原 Submission owner / target fence / UoW 内增加以下明确的本土图片合同，无新状态、新队列或新事实 owner。

- `temu-local-image-v2` 是明确图片副作用类型；stable subject 保留 exact approved Asset content hash、原官方 mall identity、API版本及 verified category + usage（这两项控制转换规则），排除 record、app、token、过期时间、operation 和临时URL。换record/批次/应用不能绕过同一图片的 UNKNOWN。
- 首次图片发送仍用原 kernel 唯一 committed permit；receipt 保存完整校验后的 images[].url、原请求 category/usage/source Asset hash、响应hash、原 attempt/fence epoch、提交时间和使用截止。E40 没有 expiry 字段，截止保守取原 durable attempt.CreatedAt + 7天，不能从查询串、当前读取时间或用户输入延长；单次请求沿已有10秒/更短 caller deadline。
- 每次使用须当前授权且严格早于截止；到期或使用窗口不足使本次商品 payload 无法取得发送许可。先核查原商品 fence：商品已成功只投影原结果；claimed/UNKNOWN 不执行新图片操作或商品 mutation。仅在商品从未发送或已有经过证明的 definitive-no-effect 时进入图片准备。保存资料/纯规则检查不能暗中上传。
- 图片续期仅接受当前同一 stable fence 的最新 attempt 为明确 succeeded、原同库 typed receipt 和 exact intent 均完整有效、结果已到期，且当前 saved record 引用的 approved Asset/content/category/usage/merchant 不变。此时沿既有 fence.Epoch + 1 和新的不可变 intent 获得一次 permit；旧成功/回执不改写、不删除。最新 fence 锁定、current immutable TargetRecord、原 intent/receipt、内容/用途/类别/商家引用与服务器截止核对和新 claim 必须在原 Product DB 事务内完成，不能在 app 先检查后调用无约束 Acquire。
- 当前批准集合由原 AssetDB owner 核对：claim 前以及实际 send 前再次读取 `org + canonical ProductKey + temu + effectiveVersion` 的 current inventory，要求其摘要与 saved.InventoryHash 完全一致，并重新核实所选实际图片内容；成员/Store/凭据/来源/当前 Record 同时走既有实时检查。任何漂移阻止 provider 调用，不释放原 fence、不新取 permit。已经开始的 HTTP 或不确定提交继续由原 UNKNOWN 保存，不把后续撤权/批准变化解释为未发送。此处沿既有分库执行事实检查，不宣称跨库锁定 current Asset head 或其在远端 HTTP 期间不变，也不建立分布式事务、第二 Asset 快照 owner 或通用证明协议。
- 商品 permit 的 Product 事务必须按 `TemuImages` 的 exact receipt ID/hash、当前成功 image fence/epoch、原图片 attempt 和服务器 UseBefore 核对全部引用；所有地址应覆盖该次最多10秒的发送窗口。实际 send 前再次核实这些引用及窗口，durable intent 不能绕过准备后发生的图片到期或新 UNKNOWN generation。
- 并发续期只有一条获得 permit；其余读取同一新 attempt 或被 current claimed fence 拦截。新 attempt 响应丢失、commit不确定、取消/重启均留 UNKNOWN；7天经过也不解除该 UNKNOWN。重放旧 intent 只返回旧结果，不延长有效期、不新发。新 intent key 由原成功 attempt + exact intent input 绑定，客户端不能指定续期代数或时间。
- TEMU 商品 succeeded 无条件挡住同 effect 的新创建；只有上述本土图片允许已知成功后的到期续期。供应商图片及 SHEIN 沿各自已证实合同，不套用本土公告。更新原 fresh-install receipt/schema guard，使同一图片保留多代 attempt 回执（attempt主键唯一、按 current fence 精确读取），商品成功的单 effect 唯一约束保留；无迁移或第二ledger。
- 新地址用于新 payload fingerprint；仍绑定当前 saved record/ApprovedAsset/用户明确提交批准并核查 exact image receipt 来源与有效期。不得把图片续期当成重新批准已变资料的捷径，也不得因 E41 建议 try again 就重发已发送的商品。必要验证：已知成功未到期复用、到期同 fence 原子续期、UNKNOWN/缺receipt拒绝、双并发单permit、续期commit/响应丢失不再发、旧intent重放不延长、不同商家/用途/资产不可串用、商品UNKNOWN前置拒绝、新地址改变payloadhash。使用已有 Go/原 PG tests，不新建验收平台。

2026-10-12 供应商 E43 consumer 的实现细化已由原 Reviewer 绑定 `eda7da8c36a7704b320ee308c165bc9a2acd3cfb` 及对应协议增量复核，结论 IMPLEMENTATION_READY，无新增 BLOCKER。供应商图片 stable subject 使用批准来源内容 hash、官方 uniqueId identity hash、E43 API、cateId 和 imageBizType；record、app/token、mode、siteSet 和同一业务类型的图片位置不新建 fence。原 intent 持久化固定 observation/options manifest（仍 ≤2 MiB），精确批准字节只在原 permit 下重新取得、核对后临时转 base64（固定 E43 请求上限3 MiB，响应仍2 MiB）。不同返回字段必须指向同一确定 URL，否则原 UNKNOWN。首次 completion 在原 UoW 保留独立 source hash 与平台返回 artifact hash；无 TTL 的回执不声明 URL 永久有效，也不能续传成功或 UNKNOWN。每次商品 claim/send 使用 exact current receipt/fence，并安全下载、核对当前同一 artifact；外部观察后再核对当前 Record/Asset/source/成员。URL 不可用、内容漂移、未知证明均不发送商品。此处没有新的 ledger、跨库锁或分布式事务。

供应商 catType 保留 opaque，不套用 Local 枚举；明确图片 slot 用于构造请求及该选择的几何检查，本地 gallery/all-SKC-preview 结构不宣称已核实服饰分类。E13 的类别条件、当前经营/币种/仓库资格由官方创建响应作 authority；不新增虚构通用资格查询。当前 E46/E47/E49 具名尺码表查询须消费：明确空返回不设永久 gate，失败/缺字段不能当 optional；要求存在时，只引用当前类目/官方 class 与当前 metadata group/mapping records 对应的已有非 reusable businessId。现有 reusable 模板需另行批准创建，不能直接当发布 businessId。E44 required supplierId 未取得证实身份时只关闭该具名查询；用户明确批准并与商品/商家/站点绑定的 warehouseId、路由、数量仍按 E13 检查，手填 ID 不标注为官方已验证。

独立增量复核：原 Reviewer 实际检查 72696b1bbe02bc6cd7ce0417627e8e98dba4f129 的产品/权限范围、SUB-K1 Acquire/write/epoch、official intent/receipt/finalize 和 fresh schema，明确本增量 IMPLEMENTATION_READY。G4 设计 BLOCKER 已闭合；以下属于 IMPLEMENTATION_TEST，阻塞受影响图片切片的合并/试用而不阻止按设计编码：I1 原事务同时验证最新 fence/成功 attempt/typed intent/receipt/资产用途类别商家/服务器截止并原子推进 epoch（不能只放开 succeeded）；I2 fresh schema 多代图片回执与 current fence 精确读取同步调整，商品及 SHEIN 成功保护、旧 intent/回执/截止保留。未运行正式实现/真实 API 或用户验收。

2026-10-12 真实实现增量检查发现上述历史“所有检查在 Product DB”错误覆盖了独立 AssetDB owner。Finding：Product tx 查询 Asset 表使正式分库部署的首次图片/续期/商品 Acquire 无法完成。Product requirement affected：M3/M4。Classification：BLOCKER，命中“核心 happy path 按当前设计无法完成”，仅阻塞 G4 增量实施/合并/试用。本节按真实 owner 修正：Product 内核的 fence/intent/receipt/已批准内容引用仍原子验证；当前 Asset owner 在 claim 前和实际 send 前实时核对。其余冻结设计不重开。该相关设计增量待原独立 Reviewer 绑定本文修订复核；真实分库及最小权限组合、准备后 TTL 到期和多代回执属于对应 IMPLEMENTATION_TEST。原同库测试只证明内核状态/并发行为，不能作为真实分库组合或产品验收。

该最小修订已由原 Reviewer 逐行复核并绑定文档内容 SHA256 `eb792b08f0e2b823867b26ccdb2e0cfe5fd0d6d660ab7f5d87f38013bd0c5a27`，明确 **IMPLEMENTATION_READY**：保留原 AssetDB owner、claim前/send前实时事实检查、原 Product tx 的 record/intent/receipt/fence/TTL 不变量；无新增 owner、分布式事务或 Accepted Risk。允许仅修复该已证实 BLOCKER，完成后只复核分库/最小权限及商品图片 TTL 对应增量。历史“待复核”描述和同库测试限制保留为原审计时点，不替代本次准入。

对应代码增量已由同一 Reviewer 绑定 `bb858aa` 加未提交修复、policy 文件 SHA256 `8ce10cab89cd2c6b8cda2f4a333acd2aac688d67de3d166777cceff91db2f60d` 复核：Product tx 跨库读取 BLOCKER 关闭；双数据库、无 Asset 表的受限 Product serving role、原 fence/多代回执行为及 product→image 锁顺序/未来10秒 TTL gate 已检查。当前 `temuFacts` 的真实 Native consumer 在 claim前/每次send前调用仍为 M2/M3/M4/M6 的 IMPLEMENTATION_TEST；相关独立单测 PASS 和作者分库 PG3.675s PASS 不是实际消费完成、完整交付或产品验收。

2026-10-12 对应 Native consumer 已由同一独立 Reviewer 绑定 `bb858aa3439d7a2db0fa103aea4c919bf8dc6b81` 加实际增量、内容摘要 `8fa69cd294d3b8bf59e91a761021c6d060c2f33b8778fae7530f923bdd4844ba` 检查通过，无未关闭 BLOCKER，前述 G4 consumer 项关闭。实际 claim/send 前消费当前资料、Asset 与图片回执；发送 context 受原 permit lease deadline 限定；观察结束重查 Record/Asset 当前 heads 并重选同一不可变 source。新发现的许可到期、观察期间资料/批准变化、五秒 source proof 生命周期均归入 IMPLEMENTATION_TEST 并经 RED/GREEN 与独立相关执行收敛，过期/漂移不调用 provider，保留原 UNKNOWN/fence。本结论只覆盖 G4/私密 mutation 对应增量，不替代供应商创建、同步、最终用户路径或真实平台验收。

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
| M1 | 完整 96 项半托字典、39 项规格及 E30 全部 87 行映射/39 个 LOCAL=true；37 项匹配和 GE/AM/SA/LU 差异可见、不能冒充全部通过；true/false/空白/coming soon/关联组、按原分区 API、未证明组合拒绝；三模式实际 DTO fixtures |
| M2/M6 | 跨企业/店铺、AppID 同名不同平台/分区、原 attempt/member 绑定、错误模式/site、缺 scope/过期/撤权、secret 无投影；断开/轮换不重新归属 |
| M3 | 缺品牌/产地/证书/价格不补造；精度/单位/规则/Asset/approval 版本漂移拒绝；exact saved payload 匹配 |
| M4 | 两并发 permit、不同 siteSet/operation/app/record 仍同 fence；响应丢失/commit unknown/取消/重启不重发；图片独立 effect；同 key 异载荷冲突 |
| M5 | 上传回执与审核/在售分离；未知错误不成功；列表无结果/多匹配不 definitive；状态 query 必需 ID；成功 receipt 原子保存 |
| M6 | 实際 Console 与 tool 调用同一 owner，server-only 凭据，当前 grant/服务校验，readonly sync 不取 publish 权限 |

代码自检、fixture 和 PR CI 仅是工程证据；实际官方 application 可用/商家权限、真实发布与用户路径验收单独 NOT_RUN。完成实现后给用户运行入口与操作步骤，不能由作者签发产品验收或生产可用。必需 CI/独立最终交付检查/合并授权仍单独满足。

## 11. 准入缺口、评审与停止点

下列记录第一轮 findings、补充官方依据与剩余读取条件；原草案要求的字段映射并非都属必要前置。调查结论不冒充已完成全覆盖核实。

| ID | 缺少的具体依据 | 受影响 Must / 后果 | owner / 解锁条件 / 层级 |
| --- | --- | --- | --- |
| G1 | 第一轮缺口现由 E03/E04/E11/E26 补齐：PA uniqueId 明确为 mall 唯一标识，同一店铺 token 的 CN 类型/PA 身份接口均由当前清单指定；无须 CN mallId 转换 | M2/M4：固定一个 supplier 身份与同凭据模式验证，未核实/缺权限不连接，变更 token/app/siteSet 不能重置 fence | 独立相关增量复核已关闭 BLOCKER；真实响应/权限验证为 IMPLEMENTATION_TEST，不是缺官方映射的永久 BLOCKER |
| G2 | E30 已完整读取，region→site 和分字段冲突处理已独立复核闭合；GLOBAL 登录/映射 BLOCKER 关闭。真实剩余：GE/AM 的价格币种/精度等规格未在已读 E24/E17/E29 中找到；E18 V3 价格也引用 E24 | M1/M3：两站点仍在用户范围，缺依据不能正确构造完整发布资料；保留缺规则和拒绝发送，不能靠地理/用户输入自证平台格式 | IMPLEMENTATION_TEST：按既定官方规则合同补齐数据并验证缺规则/冲突/漂移不能取得批准或 SendPermit；阻塞 GE/AM 发布开放及全覆盖发布验收；按2026-10-13阶段决定不再阻塞本批准备代码合并，不阻塞已准入的整体生产实现 |
| G3 | externalGoodsId 与 outGoodsSn 明确映射、PA exact recovery 证据覆盖 | M4/M5：盲目 adopt 错误结果；当前可保留 UNKNOWN 避免错误副作用 | IMPLEMENTATION_TEST；已知成功 ID 可读回、无法 qualified 时保持 UNKNOWN；实施/真实验证层处理，不授权自动重发 |
| G4 | 2026-10-12 新读 E40/E41：本土图片 URL 仅7天有效，原成功永久复用会阻断超过窗口的正常提交 | M3/M4：过期链接不可用；通过变化fence盲重传则可能绕过UNKNOWN | 原设计 BLOCKER 已由 §8.1 最小原owner修正和绑定72696b1的独立增量复核闭合；增量 IMPLEMENTATION_READY / FROZEN，I1/I2 为 IMPLEMENTATION_TEST（受影响切片合并/试用门槛），不重复全局架构评审 |

第一轮独立评审的原分类：G1/G2 为 BLOCKER；G3 为 IMPLEMENTATION_TEST。进一步官方查证与独立相关增量复核确认 G1 不需要原草案要求的 CN mallId→PA uniqueId 映射表，G1 已转 IMPLEMENTATION_TEST；本土 mall fence 修正无新增 BLOCKER。E27 辅助规格/尺码表 create 在实现层按批准与必要性收敛，准备/读取不能暗中 mutation，需要未批准创建时保持待补全。第二轮相关增量复核确认 G2 映射/冲突处理闭合，剩余 GE/AM 规则数据按 IMPLEMENTATION_TEST 在实现层收敛，并明确整体 IMPLEMENTATION_READY；缺规格拒绝发送不是 M1 完成或范围豁免。原历史评审保留。店铺服务沿当前 ServiceLifecycleExecutor 与 orgresource store_service_v1 reservation/receipt，未发现平台专属新计费依赖。准确候选 SHA、Reviewer 证据与增量复核写主 PR #655。

当前设计 IMPLEMENTATION_READY / FROZEN，生产开工前确认 Issue Ready 和实际相关依赖/owner。CN 与 GLOBAL 文档现均可访问，不再要求用户重复登录。已读页面、剩余字段与受影响操作都在官方证据中明确；本轮未读取应用管理密钥或商家数据，也未执行官方业务 API。GE/AM 数据补齐与其验收门槛保留，不能用真实 mutation 试错取代未授权商家验证。

Architecture Review：第一轮由独立只读 Reviewer 核对原始范围/Threat Model、代码实际 owner 与全部设计 diff，结论 NOT_READY；后续只复核实际相关增量，最多两轮正常评审。finding 先分类为 BLOCKER / IMPLEMENTATION_TEST / BACKLOG / ACCEPTED_RISK / NOT_APPLICABLE，列明受影响 Must、具体后果与层级。只有 G1/G2 等实际核心路径/身份/不可重复副作用条件未满足时阻止开工；非 Blocker 不扩成新的通用机制。

独立评审已明确 IMPLEMENTATION_READY，当前 Issue Design Basis/Ready 记录与相关依赖核对完成后允许生产实现。冻结后只有新的实际 BLOCKER 才重开架构；不为未变 G1/G3 或规则数据补齐发起第三轮全面评审。评审与准入证据写 #31/主要 PR，不以 CI/文档存在代替准入。

## 12. Legacy decision

Legacy decision：EXTRACT | RETIRE。

Reusable behavior：已符合当前事实/授权/事务合同的 Store、Product/Asset、TargetRecord、SUB-K1、UoW 与确定性工具复用；旧平台规则仅在当前官方依据逐条成立时提取纯行为。

Current owner：本文件 §3 对应 owner。旧 Cookie client/task-first/management/service DTO 与无产品依据定价不参与新调用链。

Cutover/deletion condition：本批新正式路径不得依赖 internal/temu Cookie/旧 listingadmin；相关当前调用切到正确 owner 后移除旧消费者。无 fallback/双读/双写/同步或 compatibility 例外；无关历史遗留不以本任务扩大清理范围。

## 2026-10-13 用户阶段决定：准备代码先合并

用户在 [#31](https://github.com/qq550723504/task-processor/issues/31) 明确要求先合并代码并记录当前缺口。此决定替代此前“GE/AM 官方规格补齐作为本批代码合并前置”的阶段要求，不改变冻结的 owner、权限、状态机、事务或副作用合同。

GE75/site165、AM10/site166 仍缺官方币种、金额精度/边界及完整包装规格，状态为 UNKNOWN；服务器与编辑器继续拒绝受影响发布批准和 SendPermit。官方询问信已获批准发送，EU 规格表仍未列出两站。两站发布开放及三模式/全站点最终验收继续由 #31 跟踪，不以本次合并写成 PASS。

本次合并不授权真实商家 API、具体商品发送、部署、保留试用库升级/迁移/删除或 Issue 关闭。真实 provider、完整 native 浏览器与用户/生产验收仍 NOT_RUN。保留实例的新 main 商业库结构升级限制、E44 supplierId 来源和缺 qualified readback 的 native UNKNOWN 亦记录在 #31；不猜身份、采用手工裁定或盲重发。