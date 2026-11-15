# 供应市场列表价格、选入状态与公开商家名称

Execution: #1292 / PR #1295。Design Basis: Independent Architecture。
Admission: IMPLEMENTATION_READY / FROZEN；独立设计及认证名称修订复核通过（final_ui_review，2026-11-15），无BLOCKER。

## Product outcome / scope

2026-11-15用户要求自营/优选列表展示价格、直接加入我的数据、展示已加入状态，优选显示商家信息；
用户最新更正：名称必须来自已认证商家，替代此前自填公开名称选择。只使用subjectverification已通过的企业认证名称，不用内部企业名或自填展示名。
卡片维持宽屏五列和现有筛选；展示数值价格/区间、供货条件，加入与详情两个操作。
已加入显示“已加入我的数据”并可打开我的数据；状态刷新后仍由真实记录决定。
优选展示公开商家名称、原供货省市，不显示内部企业名称/ID、联系人、电话、资质或验证徽章。
不增加商家认证、独立店铺、联系/采购/支付、价格同步或用户资料迁移。

## Basis / owners / boundaries

复用[supply-market.md](supply-market.md)、[supply-market-filters.md](supply-market-filters.md)、
[project-boundaries.md](project-boundaries.md)、最新Figma边界和全新系统基线。
SupplyMarket owns发布、选入命令回执及对外供货声明；Collection owns当前私人条目/批次可见性；subjectverification owns认证结果及企业名称；
Catalog/Sourcing owns原商品/来源；身份/授权owner不变。无新表、索引、第二事实源或跨数据库协议。

路径：UI → 原BFF/HTTP → Market Service原授权/复核 → 原Repository。
Service将原Release包装为成员视图MarketView，不把个人加入状态写入immutable Release。
Repository新增有界视图补充读取：<=50个明确发布ID的价格汇总和本人选入引用；
Receiver既有装配端口增加Collection可见性读取，由productsourcing适配到Collection owner helper。
应用工厂/Receiver注入继续复用原同Product数据库UoW，不由供应市场复制Collection表逻辑。

## Price read projection

列表原轻量Product投影仍只包含名称/首图。成员视图另含prices数组（currency/minimum/maximum），
在数据库对该页已发布款式数值按原币种分组；不加载所有商品、每卡详情或全部款式图片。
0价有效，缺价不推断，不混算币种，不从priceNote抽数字。详情消费同一视图/价格展示规则。
未知价格显示“价格待确认”，完整说明仍在详情；保留现有2MiB/超时/页数限制。

## Current selection truth

“已加入”针对当前明确发布Release ID（原exact发布合同）；新发布版本不会冒充旧版本已保存。
SupplyMarket命令回执按principal+actor+member及页内Release IDs有界读取、去重Collection引用，
Collection owner核对组织/actor/member、条目、原source operation、market种类及当前所属批次未归档状态；移动后用当前批次，加锁后复核归属，冲突返回原错误，不凭旧回执批次假定可见。
旧回执本身不足以证明当前仍在我的数据；归档后允许新的明确选入，历史不删除或迁移。
一个页最多50项，选入引用最多1000个去重引用；超限/损坏/依赖失败返回现有错误，不伪造未加入。
读取可见性需要原Collection read权限；缺此权限时只返回加入状态未知（null），不泄露私人记录，
市场仍可浏览但卡片不能把未知显示成未加入。正常依赖错误不降级为空/false。
原Market读取后身份/企业/成员权限复核继续执行；前端查询key继续含scope和全部筛选。

## Selection write / atomicity / retry

列表和详情均消费原select_release、expectedRevision和useMarketCommands pending/锁/核实恢复。
同一成员同一Release重复新key：原事务锁exact Release之后读取原命令回执，并由Collection owner
按原batch→item锁顺序确认仍可见；存在时返回原Collection receipt，不新增来源/条目/批次。
不存在可见条目时才调用原Receiver.Receive。原release锁串行化同一Release的并发选入；
Collection可见性加锁与归档/移动共享原batch→item顺序，处理多个历史引用保持确定性顺序；返回现有条目的当前批次。内部去重按原select/manage授权，不额外要求read；原写入guard在commit前仍检查。
不改变原事务边界/外部副作用/UNKNOWN owner；不同key可以保存自己的外层回执，但指向同一真实条目。
同key回放继续返回原回执，不因归档或页面重试制造新条目；结果未知只核实/重试原key。
选中按钮在确认回执后更新，刷新/筛选/翻页从服务端读取；不把localStorage布尔值作为事实。

## Merchant display — 用户最新决定

不新增merchantName输入/公开资料事实或新商家认证流程。subjectverification继续是唯一认证owner。
增加仅内部可调用的有界只读名称端口：以原Release.OriginalOwner.OrganizationID读取同Scope企业认证，
只有原owner的VERIFIED、非空CompanyName及有效ProviderVerifiedAt/ProviderOrganizationID等认证绑定才返回名称。
此端口不返回申请人、信用代码、电话、认证链接、Provider IDs或内部组织名称，不触发供应商调用或收费。
Service仅为当前页active优选Release使用该端口，去重原组织ID、最多50个；HTTP不接受任意商家ID。
企业认证由现有CurrentApplication buildSubjectVerificationModule装配，复用已构建的同Scope Service，
通过窄适配器注入Market Service，不让Product域读取认证表或把身份Provider变成商品owner。
认证未通过/无申请：merchant.state=unverified且无名称；模块未配置/依赖不可用：state=unavailable且无名称；
成功时state=verified并返回实际企业名称。页面分别显示商家待认证/商家信息暂不可用/认证名称。
数据读取失败不伪造成未认证；元数据失败不制造商家事实，商品价格和原选入仍可按原权限使用。
证书名称不作为新增认证准入或权限授予，原选入/人工评估资格不改变，不接管认证撤销或状态机。
跨数据库仅为独立只读摘要，不产生跨库写入或一致性恢复协议；每次列表读取原owner结果，不另建缓存/第二事实源。
产品授权只公开已明确发布商家对应的已认证企业名称，不公开个人实名姓名。
原35444如认证配置/真实认证结果不存在，则明确显示不可用/待认证；不得给合成样例伪造VERIFIED，
不得为本次展示开通凭据、发起真实认证、改写认证记录或重新发布商品。验证“已认证名称”用隔离owner fixture，
运行中仅检查实例实际可提供状态。原六件合成商品、发布ID及已经选入/待上传记录保持。
## Validation / handoff

TDD：服务端价格0/区间/缺价；成员/组织/成员ID隔离、刷新状态、归档后状态；
同Release并发不同key只一份实际Collection、同key回放/未知恢复、原授权撤销；
列表按钮携带exact ID/revision、pending禁用、已加入/未知显示、商家字段和私人字段拒绝。
复用现有Postgres/owner接收者及组件/BFF测试；必要标准构建、独立最终检查、35444实际浏览器。
新增文件登记对应索引；保留已运行定时UI/配置/权限/数据，运行镜像绑定标准receipt。
Legacy decision: N/A。没有新迁移/兼容/双读双写/通用框架或专项验收平台。
开发检查/CI不替代用户实际验收；未授权合并或关闭Issue。
