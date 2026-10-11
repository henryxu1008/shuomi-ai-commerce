# 1688 在线采集、规则与待适配接入

Status: **DRAFT / NOT_READY**

Design Basis: **Independent Architecture**。新增采集计划/规则持久化合同、公开列表发现
RPC 与有界恢复状态；必须完成高风险独立复核并达到 IMPLEMENTATION_READY 后正式编码。
Baseline: main `c023edad14ad9255dee3e04a67b539bdaf2c83b8`。
Product Authority: [2026-10-11 用户决定](../product/pd-1688-acquisition-figma-2026-10-11.md)。

## 1. 产品结果、Must 与边界

从 `/workbench/supply/acquisition` 打开与当前 Figma 一致的 1688 在线采集界面：
三种方式、多行输入、采集后处理开关、规则选择和规则模板管理。点击“开始采集”后
形成真实计划并逐项推进；可暂停并用原计划继续。没有后台任务或关闭浏览器继续执行
的产品承诺。当前单商品 operation 的精确结果读取与 UNKNOWN 核实入口继续可用。

Must：成功商品进入我的数据；只成功 publication 消耗 1 DATA_ROW；规则不免单/删数据；
未知操作不能换 key 重新收费；开启开关才将规则通过的精确已保存来源转入待适配；
跨企业/actor/撤权不能读取、推进、改规则或转入；不能伪造成功/过滤结果。
原有匿名采集的挑战/频率/出口风险按原产品决定保持 Accepted Risk。

不扩大到自动上架、AI 执行、支付、永久后台调度或通用任务引擎。不用测试 fixture
宣称真实 1688 当前页面可采集。用户已确认来源公开起售价/最小起订量及公开搜索页/店铺列表页。

## 2. 当前能力与根因

- `BrowserAcquisitionService` 只接受 canonical `detail.1688.com/offer/{id}.html`，
  已有 replay-before-dispatch、资源 admission、精确出版与当前身份恢复。
- collector RPC 只有 Acquire；生产 provider 无 keyword/list discovery 合同。
- 当前 Product publication 与默认 Collection 成员在一个 Product UoW 内保存。
- `collection.Service` / `preparation.Service.Transfer` 已支持受权源选择、版本化批次、
  原 key 回执与同 Product DB 事务。不是新的商品、批准或平台事实 owner。
- `validateSupplyChain` 和 `buildSupplyChainModule` 将准备/转入与官方应用、Asset、
  Temporal 执行一起准入；公开采集保存后的待适配入口因此被无关执行配置关闭。

本批最小修正是单独装配 Preparation 的现有转入/列表/来源读取能力；执行相关 routes
保持现有严格准入。不能放宽完整 SupplyChain 配置，更不能伪造官方应用或店铺。

## 3. Owner 与调用路径

| 事实/能力 | 唯一 owner | 本批变化 |
| --- | --- | --- |
| 采集计划、冻结来源与逐项处理回执、规则模板 | Product Sourcing | 新增有界合同及 Product DB 表 |
| 公共列表发现 evidence | credential-free 1688 collector | 新只读发现 RPC，不保存业务事实 |
| 单条采集 operation、SRC/Catalog publication | 当前 Product Sourcing / Catalog | 复用原合同/状态/计费 |
| 我的数据批次和成员 | Product Collection | 复用原子 publication 成员及精确来源读取 |
| 待适配来源/转入回执 | Listing Preparation | 复用原 owner，补独立装配 |
| DATA_ROW 预留/消费、正式 offer | Resource / Commercial | 不变，计划不得签发消费/退款 |
| 当前身份/组织成员/模块权限 | ZITADEL / Workbench / authz | 不变，无后台代用户 authority |

contract → implementation → injection → consumer：

`sourcing` 计划/规则/列表发现窄 port → `productsourcing` 当前请求编排 + Product PG
repository + collector browser leaf → current-application 显式 capabilities、live 授权及
Preparation 注入 → 同源 BFF → 现有采集入口 / 工具市场在线采集入口 / 我的数据。

只允许一个 Writer 修改这些 shared contracts。复用 GORM/PG 事务、现有 Playwright
浏览器隔离与 admission/throttle；不新建 scheduler、Saga、IAM 或泛用流程框架。

## 4. 计划与持久化

新增表由 Product Sourcing 管理，计划/规则都在当前 Product DB：

- `product_acquisition_plans`：Organization、Actor、Member、ID、CommandKey、InputHash、
  immutable InputJSON、Mode、AutoTransfer、冻结 rule 内容/版本、Revision、CreatedAt。
- `product_acquisition_plan_items`：PlanID、稳定 ordinal/source identity、来源发现结果、
  canonical URL、child acquisition key/operation ID、exact publication/collection 引用、
  rule evaluation、transfer 引用、最近确定错误及恢复状态；不存 Product 正文。
- `product_acquisition_rule_templates`：Organization、Actor、Member、ID、名称、规则、
  Revision、创建/更新/归档时间；同 scope 的幂等 mutation 回执与输入 hash 同事务保存。

Organization + Actor + command key 唯一，同 key 同输入重放；异输入 409。ID 派生于
当前 scope/key。所有外键与读取包含 Organization/Actor，Member 须与当前成员匹配。
计划保存规则值快照；模板以后改名、编辑、归档均不改变已开始计划。
不建历史迁移；初始化/权限清单由显式 schema-owner 命令更新，serving 只校验。
缺 schema/权限/collector capability 时启动或相应能力 fail closed。

技术资源上限：每计划最多 20 条非空输入、全计划最多 200 个去重 canonical 商品，
输入体不超过 64 KiB，计划投影 2 MiB，列表每页不超过 100。上限在界面显示；
发现超过上限须明确显示截断，不能声称整页完整采集。模板每 scope 最多 100 个、
名称 100 字/400 UTF-8 bytes。使用现有 collector concurrency=2 与 throttle，不抬高。

## 5. 执行与恢复

客户端只提交 mode/多行输入/模板 ID+revision/AutoTransfer。服务端生成不可变计划；
指定商品立即 canonicalize，重复链接折叠且明确显示。其它方式先保存发现输入，
每次推进一个发现输入或一个 canonical 商品；从 DOM 提取的链接仍是不可信 evidence。
发现结果冻结在计划后才执行 child acquire，重放不重新搜索已经冻结的来源。

每个 child key 按 PlanID + canonical URL 派生，原 scope + 原 key 调用现有
`BrowserAcquisitionService`；禁止客户端自报 child key、publication 或已成功状态。
计划条目状态投影：`pending → acquisition_pending → published → rule_evaluated`；
确定失败 `failed`；结果不明 `needs_verification`，只保留/核实原 child key。
“正在请求”的本地 UI 状态不是第二份 publication 事实。

并发推进先按 plan revision/step identity 做 CAS 和有限 lease/fence；一个计划同一时间
只有一个有效推进。服务端并发或响应丢失后先读原条目/child operation/transfer 回执，
不将 lease 到期当成操作失败。child 已 published 仅按原精确版本继续后处理，不重采。
pre-dispatch admission/recovery仍由原 Acquisition owner 判断，计划不得覆写它。

浏览器切换组织/actor、撤权、退出、取消或离开页面：取消当前请求并停止发出新步骤；
已经发出的操作保留原 key，回到原 scope 后可继续/核实。客户端迟到响应不能写入另一
scope UI。GET 只读取，不启动发现/采集/转入；继续操作用显式 POST。
没有后台 writer、持久 token、请求外 goroutine、跨企业代执行或新 durable auth owner。

## 6. 发现与 collector 安全边界

新增 `/internal/v1/browser-collector/discover`，严格输入 mode+canonical public list URL
或 keyword；只传必要公开输入，不含 Organization/Actor/token。沿原独立 collector
进程、内部 caller credential、无数据库凭据/数据库网络隔离、禁 ambient proxy、
HTTPS allowlist、资源/响应大小/deadline/collector throttle/cooldown。

关键词最多 200 UTF-8 bytes；服务端构造固定 `s.1688.com` 搜索 URL，不接收任意 origin。
按页由批准种类的结构校验器生成规范 URL：拒绝 IP、credentials、非 443 端口、
fragment、未知查询参数、login/passport/account/write path 和站外跳转。
店铺列表校验不得只检查 `.1688.com` 后缀：只接受单层店铺 hostname 的
`/page/offerlist.htm`，拒绝保留 host（s/detail/login/passport 等），不得接受其它店铺路径。
搜索页只接受 `s.1688.com/selloffer/offer_search.htm`。查询参数采用分别列出的窄白名单
并验证类型/长度/重复项，不透传未知参数；关键词规范编码。公开结构真实可用性未验证。

在全新匿名 context 中读取当前页面可见 canonical detail 链接、去重、按 DOM 顺序
截断；不点击商品、不登录、不无限滚动/翻页、不消费 DATA_ROW。响应是有界来源链接
及真实 page/truncation evidence，不是成功商品、price 或 publication。
遇登录/验证码/拒绝/无法识别页返回来源不可用，不新增挑战自动处理逻辑。
现有单商品采集 leaf 的批准逻辑不扩到新 discovery 路径。
公开结构真值尚未实测，fixtures只能验证结构/边界，真实采集验收保留 NOT_RUN。

## 7. 规则合同

模板规则是采购价 min/max、起批量上限，AND，空条件忽略；全空则全部成功 publication
通过。边界包含等号；输入非负规范十进制、min<=max，MOQ 为正整数。价格以 CNY
原始来源明确事实比较，用 exact decimal/rational，不能采用浮点近似或汇率推断。

用户确认价格取来源公开起售价：沿已封存原 acquisition operation 的
`Command.Envelope.SupplierOrCostFacts.Facts[price_fact.N]`（验证 command hash 与精确
publication）；取有明确 CNY 的来源阶梯价格最低 amount，MOQ 取来源明确阶梯最小
起订量，不使用优化后的 Catalog/规格报价，不能推断缺失币种。只在已填写条件所需事实
缺失/币种不明/矛盾时返回 `unknown`，不猜 0、1、起售价或币种。`failed`/`unknown`
都不自动转入；已保存商品和计费不受影响。采用 Product 当前 source evidence 的价格
事实，不能把 AI 优化/当前 Catalog 漂移结果当原采购事实。

模板新建/改名/编辑/归档使用原当前 scope 权限、expectedRevision 与幂等 key；
不默认种入 Figma 示例“标准规则/低采购价/低起批量”作为企业既有数据。
原始规则事实保留在冻结计划，不因模板归档丢失操作解释。

## 8. 计费与自动转入

每个去重商品先沿单条 Resource admission/reservation 再执行；publication 成功由
原 proof reconcile 消耗 1 DATA_ROW。发现失败不消费、单条确定失败沿原释放、
UNKNOWN 保留原 reservation。计划无独立资源扣减/退款/钱包访问；未通过规则正常消费。
额度不足停止发出新商品，可查看已成功/失败/待核实项；不签发整批成功。

AutoTransfer=false 到此结束。true 且规则通过时，在现有 Collection owner 核对
原 publication/成员/当前权限，然后调用 Preparation 原受权选择与 Transfer。
由 Product DB 冻结实际通过成员集合及源版本、expectedRevision、稳定 transfer key；
创建转入命令与计划引用同事务；Preparation 仍是唯一来源/回执 owner。
回执丢失/COMMIT unknown 按原 key读取，不换 key/重复创建待适配。
来源后来移走/归档或授权撤销须拒绝/显示冲突，不能偷偷扩大成员集或忽略权限。
有采集权但无 supply.manage 时不能开启自动转入；后续权限撤销不能转入。

Preparation-only 装配只授予已有 preparations/sources/transfer 必需表，提供相关原
live-authorized routes 与待适配列表；full execution 的官方应用/Asset/Temporal/规则/
上传/AI 准入不变。两种装配不能重复注册 route 或建立第二 service/fact owner。

## 9. UI、Legacy 与验证

适配仓库现有 Console、Button、Input、Dialog、Select、Textarea、tokens/theme；
高保真 Figma 四节点和对应浅色节点为依据，静态 SVG 按响应下载且保留 intrinsic
尺寸。恢复/逐项进度是当前已批准幂等结果的真实投影，不伪造模板/计数/成功状态。
原 route 保留可用；工具市场在线采集入口复用此组件，不复制逻辑或创建平台 Workbench。

Legacy decision: **N/A**（本计划复用当前 owner）。若读取旧提取器，仅可 EXTRACT
仍有价值的公开 DOM 行为到新 leaf；不得 wrap旧 crawler/task/service，发现RETIRE
依赖即停止使用。不新增 fallback、dual read/write、迁移或相容 schema。

TDD 与风险匹配证据：规范URL/DOM发现边界、同key异输入、规则真实/缺失事实、
精确publication读回、部分失败与UNKNOWN原key、并发推进、资源重放不重复消费、
跨scope/撤权、模板并发更新、转入receipt丢失和Preparation-only不解锁执行路径。
PG 测试使用任务隔离数据库；UI 仅核对本入口/模板/实际状态及窄屏，不建验收平台。
最终一次独立交付检查，交出实际运行入口及操作步骤，产品验收由用户完成。

## 10. 开工顺序与当前停止点

1. 两项产品细节已确认；独立只读复核实际合同/权限/恢复/出网边界，达到 Ready。
2. 规则与计划 owner/PG + domain测试；发现 RPC/leaf + contract测试。
3. 当前请求推进/计费复用/Preparation-only 接线及相关HTTP/PG验证。
4. Figma UI与BFF；整体路径自检，最终独立交付检查，保留可运行本地程序。

当前 **NOT_READY**：产品细节已明确，Architecture Review 未完成。
不修改正式业务代码或建立生产 schema。该设计是当前待复核提案，不是Approved Authority。
