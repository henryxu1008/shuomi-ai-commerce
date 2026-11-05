# 数据服务关闭采集执行，保留配置、资料与历史

Status: **IMPLEMENTATION_READY（仅离线实施）**。Design Basis: **Independent Architecture**。
执行 Issue #1073，唯一 Writer 与主要 PR #1075 不变。
2026-11-04原独立Reviewer核对HEAD `bb4b6d41d0abd014d08d7d7a44a1b53e000ebe8a`、
本设计blob `4f6459910641a4b9bc3edc5f7d50ab148ada74ad`及现有consumer，结论无BLOCKER、可以实施。
唯一IMPLEMENTATION_TEST为启用态GET必须仍消费原Service恢复；两态接线已澄清，须以ADMITTED/RUNNING测试收敛。
本准入不授权镜像构建、配置发布、实例更新、真实调用或合并。

## 用户结果与当前依据

2026-11-04用户明确允许：补最小设计，准入后离线实现和相关回归，形成新的固定候选；
保留资料、配置和历史查看，关闭未授权真实采集、AI和付费调用。原a66继续运行，预算不变；
不自行提权、不改凭据、不发布配置、不构建镜像或更新实例、不执行真实调用、不合并关单。
这替代此前应用只能固定1e3509的开发限制，不改变其已构建镜像的证据范围。

使用者为LAC配置管理员及市场调研/智能选品人员。目标是候选应用能够提供原数据市场、
任务/结果、研究证据与配置入口，同时采集关闭后启动及GET均不启动/恢复采集，也不新增采集费用。
没有新页面、菜单或产品流程；沿[数据服务](data-services.md) §2 的现Figma/Product Authority、
[研究/选品](market-research-selection.md)和[当前UI/IA](../product/final-ui-ia-authority.md)，未开放能力如实显示原因。
复用[数据服务原授权合同](data-services-native-authorization.md)、[AI关闭态生命周期](ai-disabled-application-lifecycle.md)、
[共享通道凭据](shared-channel-credentials.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md)。

Scope：数据服务的配置→运行装配→HTTP consumer→DataAcquisition原读取逻辑；
并只读核对当前候选中其他启动路径是否能触发被禁止的真实调用，不改变无关内部任务。
Out of Scope：全系统只读/暂停、新执行治理平台/调度器、其他worker重构、新数据源、真实资料/历史查询、
新schema/状态/费用/权限/恢复owner、凭据/部署配置发布、旧兼容或迁移、真实平台/AI/付费调用、实例更新/合并。

## 根因、owner与注入路径

组合基线为07c5d3fa1（原#1075与实际main d80d7b1e0/#1089组合）。已有代码有两处耦合：
`currentapplication.Run`在DataServices存在时必建Amazon provider、dial Temporal并Start worker；
`DataAcquisition.Service.Read/Results`对ADMITTED/RUNNING任务调用EnsureExecution，浏览历史也可能恢复工作。
只跳过worker或让Temporal报错，会分别留下可提交/恢复入口，或让原历史无法读取。

事实owner保持：DataAcquisition原job/item/来源、SRC/Catalog/Collection原证据，DataService原密钥/定制记录，
Organization Resource原费用及UNKNOWN，Temporal原执行身份，ZITADEL/WorkbenchContext原组织/成员授权。
新增的是同一owner的窄读取端口与装配条件，不新增服务、事实副本、表、账本、缓存或恢复责任方。

| 合同 | 实现/注入 | consumer |
| --- | --- | --- |
| `dataServices.executionEnabled`，bool，默认false | 当前Config→ApplicationFeatures→cmd→DataServicesDependencies→Module | 显式false或未声明均不创建采集执行资源；true才沿原完整执行准入。不解释旧字段、不提供默认站点/Temporal或fallback |
| DataAcquisition `Reader` | 原Repository的窄Read/Items/WithResultRead与LiveAccess；抽取原Read/Results/Provenance的读取逻辑 | 关闭态任务/by-command/results和研究Provenance；无provider、starter或charges依赖，不暴露执行方法 |
| 原Service启用态 | 与Reader共用内部验证/投影；原Service.Read/Results仍在原credential守卫事务内恢复未完任务 | 显式启用时保留原有效重放/恢复行为及其现有测试，不借关闭态新增另一恢复协议 |
| HTTP读取与执行分开 | 关闭GET→纯Reader；启用GET→原Service.Read/Results，消费同一窄读取接口；原Service只在enabled时注入 | 启用的ADMITTED/RUNNING恢复仍在原WithResultRead内。Console/DataKey共用创建和取消入口在关闭时拒绝，发生在provider/funding/admission/Cancel之前；认证/权限描述符保持 |
| Options | 关闭时AcquisitionReady=false、Sites空及业务原因 | 原页面消费已有不可用投影；不显示可采集或因查询选项做provider Ready探测 |

## 开关与装配不变量

- 原DataServices DB、schema/权限、独立窄池、同物理ProductDB、原组织/成员准入和proxy约束始终验证；关闭执行不删除模块或研究依赖。
- 关闭时不构造/Ready Amazon provider、不dial其Temporal、不构造/Start worker，不注入starter；不会以已有任务作为验证输入。
- 关闭态可省略执行元数据；若仍声明browser/driver/站点/Temporal，则沿原静态格式校验。元数据合法不代表执行开放。
- enabled=true须保留全部原执行参数、资源、provider、Temporal、worker准入，缺项失败；不把nil执行依赖当关闭态成功。
- 关闭态HTTP/模块注入必须明确，无执行依赖混入；GET使用窄Reader，命令不能调用nil Service或回退为enabled。
- 关闭时不调用Amazon charge工厂、不登记ConsumerAmazonData owner；其原reservation/UNKNOWN保持，不结算、释放或重报价。
  其他已有resource consumers继续使用原工厂/owner列表；RecoverDue仅ClaimDue已登记owners，不因关闭Amazon而关掉或认领别的consumer。
- DataKey元数据/当前配置、定制记录及原人工交付配置路径沿原owner；本开关只控制实时采集执行，不声称全数据服务或全系统只读。
  本轮不实际调用这些写入口、不改凭据、创建/取消任务、提交定制或交付具体商品。

## 授权、事务与失败/恢复

Reader复用原live CheckRead、scope、actor/member、credentialID/revision及来源精确校验。
外部三个GET仍由原WithResultRead持key FOR SHARE，job/items/SRC读取共用绑定事务；
撤权、跨企业、成员变化、guard提交失败继续拒绝/丢弃结果，原单连接/上下文/大小/分页上限不变。
关闭态读取ADMITTED/RUNNING只返回原事实，不改状态、不Ensure、不刷新证据、不重建执行或费用。
Provenance仍从原SAVED item和准确Catalog版本核验，读前/后live检查不变，不凭页面可见性放行来源。

无新持久化状态转换、事务、幂等键、补偿或自动恢复。取消关闭时拒绝且不释放quota/reservation；
重新启用须显式配置并通过原准入，原执行仍由原Service/Temporal/Resource owner按原键恢复。
进程退出只关闭本进程已拥有的资源；不停止其他进程/worker、清队列、认领历史任务或写UNKNOWN。
GET失败返回原错误，不能将读取失败显示为空历史或已完成；关闭创建/取消返回明确不可用和HTTP503，不写成功回执。

Threat Model沿现有租户、成员、凭据撤销、原键、来源/费用归属和外部副作用Must；无新增Accepted Risk，
不把理论全系统静默要求加入本批。主要实施风险是关闭分支漏接、GET间接恢复、原credential事务边界丢失和误停其他consumer。

## 其他启动路径与本批边界

| 当前候选条件/路径 | 已有约束及本轮处理 |
| --- | --- |
| ProductAgent=false、完整图片Generation=false、Chat=false | 复用原关闭态与#1089注入；不打开executor/图片worker/通道secret reader，不新增模型调用或预算 |
| 店铺观察preparationOnly=true | 原官方应用绑定为nil及空观察库准入，官方读取失败关闭；保留现有准备worker，不为本批新增观察执行模式 |
| Knowledge | 原对象存储/Tika解析，无文字/图片provider；保留原内部处理，不扩为全系统暂停 |
| Ecoservices nonPaymentOnly=true | 原分支无支付恢复回调；不启用支付或商户申请 |
| 充值两渠道Enabled=false | 不构造/读取渠道秘密或provider；原恢复原provider核对缺失即失败关闭，不能触发支付渠道调用 |
| 资源/其他consumer/推荐成熟 | 原内部owner和记账可能执行、写原事实；本批不改，不承诺全系统无后台写入。未来更新授权仍须包含这些已有启动影响 |
| POD/Supply/Fulfillment未配置、独立采集桥退出 | 本应用不新增对应worker或启动桥；不推定其他进程的运行状态 |

这张表是当前候选及源码范围，不是实际实例执行证明或更新授权。若未来配置启用了不同的provider，必须重新核其启动影响。
#1089已合入的新模型/通道接口直接复用，不重建公共能力；原私密候选和1e的LoadConfig证据不自动升级为新候选配置验证。
仅准备去密字段差异，保留原文件/hash/ACL；不生成或发布新的含密钥配置，不安装/改真实库。

## 必要离线验证与交付

1. 先RED证明关闭配置不应要求provider/Temporal/worker生命周期、读取未完任务不应Ensure，关闭创建/取消不应访问funding/provider/写repo。
2. 最小实现后GREEN：关闭态无执行依赖但管理/任务/结果/Provenance读取成立；原启用态生命周期/恢复与相关负例仍通过。
3. 复用现有合成PostgreSQL/HTTP夹具核非空保存结果、DataKey守卫/撤权/单连接和费用owner装配，不建新runner/验收系统，不访问保留业务库。
4. 复用#1075已通过的研究→比较→人工决定→准确子集交接回归；组合#1089的相关fixture沿当前channel owner，不恢复退休凭据表。
5. 形成新准确代码候选与去密配置差异，必要CI和原独立Reviewer增量检查；镜像构建、配置发布、实例启动/更新、真实调用及用户验收均NOT_RUN。

本设计的独立检查仅覆盖新增执行边界及实际main组合，不重复全局模型/价格/账本或未变化代码评审。
