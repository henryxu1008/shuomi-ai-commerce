# Local Account Center：新关闭态候选的配置差异

状态：**OFFLINE_IMPLEMENTED / READONLY_AUDIT_COMPLETE / RUNTIME_SCHEMA_MISMATCH_CONFIRMED / NOT_BUILT / NOT_PUBLISHED / NOT_DEPLOYMENT_READY**。
应用固定f337521d0；既有原身份运行配置/数据库元数据只读核查见[新通道实际差异与最小方案](channel-contract-f337-readonly-review.md)，本轮在其范围上整理design-only候选，不重做现场操作。原离线实现与CI记录仍绑定各自准确SourceRef，见PR #1075。
后续已取得原公共owner的ACK1075-INSTALL；另名新空Run的安装入口仍需最小补丁，见[design-only候选](new-run-ledger-installer-design.md)。角色合同已确认，实际安装操作者/权限未确认；不将候选设计写成可执行入口。
已按[数据服务执行生命周期](../architecture/data-services-execution-lifecycle.md)独立检查并在编码前达到IMPLEMENTATION_READY。
本页为去密审查差异，不含配置正文或发布授权。原a66继续运行；不自行提权、不改凭据、不发布配置、
不构建镜像或更新实例、不执行真实采集/AI/付费、不合并关单。

## 本次用户可见结果与调用路径

原智能市场启用/配置入口、市场调研/智能选品页面、已保存资料及任务/结果历史保持原owner。
采集关闭后，启动不会建provider/dial Temporal/启动采集worker；GET不再因未完任务而恢复执行。
创建与取消不可用，Options明确“实时采集已关闭，可查看已保存资料和历史”，不显示虚假开放站点。
关闭态GET沿同DataAcquisition的窄Reader；显式启用GET仍沿原Service.Read/Results并在原DataKey守卫内恢复。
研究Provenance仍核原SAVED item、SRC/Catalog准确版本、组织/actor/member与前后live权限，不复制或刷新证据。

本轮实际main一次组合d80d7b1e0，直接复用#1089的[共享通道凭据合同](../architecture/shared-channel-credentials.md)，
不重复公共模型、价格、费用、凭据或恢复能力。沿[研究/选品](../architecture/market-research-selection.md)、
[AI关闭态完整图片](../architecture/ai-disabled-application-lifecycle.md)及[全新系统基线](../product/greenfield-no-legacy-migration.md)。
没有新增UI/IA，沿已有Figma/Product Authority；开发自检不替代现场业务或用户验收。

## 相对未发布1e配置基线的具体差异

| 位置 | 新候选目标 | 当前结果/边界 |
| --- | --- | --- |
| dataServices.executionEnabled | 明确false；新系统默认false | 保留DataServices DB、权限和研究依赖；不实例化采集执行资源。合法旧声明元数据可以保留；关闭也可不声明执行参数，无默认站点或Temporal fallback |
| dataServices的HTTP/费用装配 | 关闭态GET→Reader，执行Service/worker/starter不注入；不登记ConsumerAmazonData | 关闭创建/取消不访问provider/funding/admission/Cancel、不写新任务/新费用或释放原预留；其他consumer仍沿原注册表 |
| productAgent.enabled / contentOnly / contentTitleOnly | false / false / false | 保留内容/研究/选品当前policy元数据及原共享Limits，不装配文字executor |
| imageAgent完整模式 / generation | 保留非空workerConfigFile及API/Asset/Product/config读取；generation仅enabled=false | 不读图片worker秘密、不打开workerDB/dial图片Temporal，不声明虚假图片正预算或恢复旧4条图片声明 |
| Chat | 保持enabled=false，退休PointTariff不进入新配置 | 不启用Chat、不新增独立点价或目录fallback |
| modelRoutePricing.writer / bindings | 单Run owner窄writer；三用途声明引用当前ConnectionID/ConnectionProvider等正式模型定义 | 不是将原imageWriter改名或复制旧Image current；新通道缺准入则明确未就绪，不伪造版本 |
| 平台参考价 / 企业三类默认选择 / 操作模式 | 与运行manifest分开，仍未发布 | 沿原完整CAS/current，不在验证时“先保存一版” |
| 通道secret reader | 全部AI执行关闭时不要求实例化或解密Key | 未声明不代表通道已准入；未来执行另核当前owner的合法连接/版本及其受限读取身份 |
| ProductAgent / modelRoutePricing.writer的database | 拟从product_agent同改为product_agent_run_20261104 | host/port/user/password及池4/1保持；新库未创建，原库/manifest不改；不改变其余数据库 |
| LAC三用途definition | 拟connectionId=lac-bailian-cn-beijing、connectionProvider=bailian | 同企业共享显式连接，当前无连接事实；模型ProviderID仍DASHSCOPE，channel/region/用途限制保持 |
| 三用途退休准入引用 | 新候选清空旧org-credential引用，保持未配置 | 不机械改前缀或伪造channel版本；合格profile的连接/版本由原Catalog owner准入填充 |

DataServices密钥元数据及定制记录/原人工交付配置仍沿原领域合同；该开关只关闭实时采集执行，不代表所有POST或全系统只读。
本轮没有实际创建/撤销密钥、提交/交付定制、创建/取消任务或操作具体商品。

## 预算和文字定义保持

| 内容 | 已确认值 |
| --- | --- |
| 市场调研 / 智能选品 | 各¥0.30、20点；各一次合计¥0.60、40点 |
| 原内容 | 候选A已批准：也收紧至¥0.30 / 72000 Token，不重复请求预算决定 |
| 共享Limits | CNY、12步、2调用、72000 Token、300000微元、120秒 |
| 文字声明 | 研究/选品每调用最多10,000,000 micropoints；原内容未声明该独立点数上限，不表示免费；三类仍沿已确认费用/Token上限。prompt≤32000、completion≤4000，input≤32768 bytes、output≤16384 bytes，deadline≤45秒 |
| 模型方案 | 百炼deepseek-v4.1-flash，openai-compatible，非思考、无工具/搜索；用途prompt/schema沿原内容、研究、选品anchors |
| 参考估算 | 输入2元/百万、输出8元/百万；两次最大profile估算约¥0.192 / 19.2点，受上述更小总预算约束 |

这些是版本化参考估算及上限，不承诺厂商实际账单精度；不继续研究精确厂商价或扩大费用/调用次数。
模型和操作定义不拥有第二账本；实际调用/预留/UNKNOWN及人工决定沿原owner，AI建议不自动修改商品或人工决定。

## 私密文件与新公共合同的证据边界

原a66文件仍为C:/Users/Henry/.codex/private/market-research-unified-20261102-01/candidate/current-application.private.json，
SHA256 1271E0D89BC8E8F8DE5DF92FFCE94D9B9032291C9D7EC04FEA8FCC6AFB195B97，原服务只读挂载保持。
原独立未发布文件C:/Users/Henry/.codex/private/market-research-unified-20261104-config-a-01/current-application-1e3509-a.private.json，
SHA256 194B770F10B165768959CFC9B678D99F8F1BFECF4FA65387B87C02CFFF0786EC；Windows ACL仅Henry/SYSTEM、前阶段实际Linux mode0600。
本轮没有改这两个文件、其凭据值或权限，也没有生成/发布另一个含密钥候选。

1e3509的真实只读bind直接LoadConfig PASS保留其原范围；不能把它改写为本轮新代码的解析、装配或启动结果。
旧去密字段diff及解析证据在.local/selection-private-a-parse-1e3509-20261104-01/、selection-config-mode-fix-1e3509-20261104-01/。
旧1e两镜像的User/来源已核，但本轮新代码镜像NOT_BUILT，不能沿用旧镜像身份作为新产物验证。

#1089的新通道模型使用connectionId/connectionProvider和channel-credential-v1引用，旧org-credential版本不自动转换。
配置管理即使AI关闭，也按公共owner的当前schema检查合法通道metadata投影；旧Run配置/角色证据不能冒称覆盖新投影。
**本轮已用原reader/writer核实实际系统目录：新通道两表及去密视图均缺失，旧ai_client_credentials仍存在，当前库不满足新metadata管理前置。**
研究/选品/内容三条候选声明均缺connectionId/connectionProvider，仍带退休org-credential版本；没有改写私密文件或实际绑定。
缺失ai_channel_secret_reader属于标准新空Run准备缺项，AI关闭运行本身不打开其池；不把它与metadata启动阻碍或真实AI执行所需计量混为一谈。
此前合成夹具仍是离线证据，本轮真实库只读、不执行installer/DDL/GRANT；f337实际LoadConfig/Run仍NOT_RUN。
最小方案为保留原a66和原库，由原安装负责人另行准备合法的新空Run资源与当前schema，再审查新私密配置；不迁移旧凭据/历史、不在原库补兼容。
原owner已确认固定ledger CLI不支持另名库；仅改数据库字段不能启动安装。[入口最小设计与工具版本](new-run-ledger-installer-design.md)已获用户认可并完成原 CLI 离线补丁/合成自检；制品构建、实际安装和配置准备仍须分别授权，本轮未执行。

## 其他 worker 与任务队列的实际范围

本表描述当前候选条件，不宣称实际实例已运行新代码，也不从旧0积压快照推定无任务。

| 路径 | 新候选中的边界 |
| --- | --- |
| DataServices实时采集 | executionEnabled=false，无provider/Temporal/worker/恢复GET；不登记Amazon费用owner。原a66中的消费者不由本轮控制 |
| 完整图片 / 文字 / Chat | 沿已批准的关闭态，保留完整应用及历史；不装配新执行或通道secret reader，不推定其他进程已停止 |
| 店铺观察 | 原preparationOnly=true，官方应用不注入，原空观察库准入和官方访问失败关闭保留；准备worker仍存在，不承诺无内部写入 |
| Knowledge | 原内部对象存储/Tika文档处理，不含AI provider；保留既有Sweep和本地资料处理，不扩为通用暂停 |
| Ecoservices | 原nonPaymentOnly=true分支无支付恢复回调，不启用支付或商户申请 |
| 充值 | 两渠道false，不构造provider或读取渠道秘密；原恢复缺匹配provider失败关闭，不做真实支付渠道调用 |
| 资源订单 / 其他consumer / 推荐成熟 | 仍沿原内部owner和任务，可能写原事实；不因关闭Amazon而停止或认领其他consumer，未来更新须覆盖这些既有启动影响 |
| POD / Supply / Fulfillment / 独立采集桥 | 三类worker配置不存在则不装配；原独立桥保持退出，不启动、不控制原会话 |

本批不是全系统无后台写入模式。原a66的服务、worker、队列、数据和卷保持；本轮不停止、不重放，不启动新候选来证明没有任务。

## 已完成与剩余阶段

- 已完成：设计准入、关闭态最小实现、相关RED→GREEN、原启用恢复及合成研究→选品→人工交接回归。证据在.local/data-services-disabled-20261104-01/及PR #1075。
- 已完成的原1e标准镜像保留为准确旧来源证据，不包含本轮采集关闭与#1089，不作为新候选产物。
- 已完成：原身份只读元数据核查，确认现库与#1089不符；具体证据及去密差异见[本轮核查](channel-contract-f337-readonly-review.md)。
- 尚未执行：新空Run资源/安装、实际私密配置的f337严格解析和真实装配、新镜像构建、配置发布、两服务更新与正常登录管理/历史读取；现库不具备管理schema准入。
- 真实执行仍未授权：共享参考价/完整三类选择/模式/通道准入及ai_invocations执行合同各按原owner处理，不迁移旧数据，不改凭据或真实试连。

后续顺序、精确字段白名单、原a66保留项和回退见[两服务更新方案](market-research-unified-update-plan.md)。
用户产品验收NOT_RUN；本轮实现、CI或AI复核不签发产品验收/普遍可用。
