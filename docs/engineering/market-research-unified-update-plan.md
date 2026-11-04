# 市场调研与智能选品：固定1e3509两服务更新方案

状态：**ACTUAL_BIND_LOADCONFIG_PASS / NOT_AUTHORIZED_TO_EXECUTE**。用户本轮授权仅收紧候选POSIX mode到0600；内容hash/Windows ACL不变，新只读bind下直接固定1e LoadConfig已PASS。0777阻碍已解决；不构建应用镜像、不发布配置、不启停/更新实例、不真实采集/AI/付费、不合并。
2026-11-04用户已批准候选A，包括原内容也收紧为¥0.30/72000 Token；该决定已完成，不重复作为待审预算阻碍。
目标项目仅 `task-processor-unified-20261010`；服务仅 `current-application` 与 `listingkit-ui`。
应用 SourceRef固定 `1e3509bda06ad9b8160b0c5a7c5d451de9496c61`，tree `074f19ab3dd06a8f9552ba73d13b344055913b6c`。
后续文档HEAD/main变化不自动改变构建来源。原 [PR #1075 CI41765722589](https://github.com/henryxu1008/shuomi-ai-commerce/actions/runs/41765722589) SUCCESS及独立增量PASS只按其原绑定引用。
前一文档候选 `596cc3c1ff0998bd19ebd2c213125c410a9ad960` 的 [CI41785450029](https://github.com/henryxu1008/shuomi-ai-commerce/actions/runs/41785450029)
已 completed/SUCCESS，Required CI Gate `126430070664` SUCCESS；该结论不等于后续文档SHA的CI结果。0777阻碍由本轮授权权限修正及实际bind直接加载证据解除，不能仅由CI绿色推定。

本轮 Design Basis: **N/A（现有私密文件合同的单文件权限修正、直接离线解析及文档）**，复用[当前生命周期合同](../architecture/ai-disabled-application-lifecycle.md)、
[配置差异](selection-shared-model-wiring-preview.md)和[标准镜像构建入口](../operations/application-image-build.md)，不修改现有应用/Compose生产文件。

## 已核原版本与保留对象

2026-11-04 本轮fresh只读核对原身份/后端/UI均running/healthy，前后端来源仍
`a66d7d0cde74cded28d77060ac941851cc342a30`。原manifest SHA256仍为
`1271E0D89BC8E8F8DE5DF92FFCE94D9B9032291C9D7EC04FEA8FCC6AFB195B97`；本轮只输出白名单元数据，不展示/更改凭据。上轮已按授权在新私密候选复用原值。

| 对象 | 当前固定引用 | 后续动作边界 |
| --- | --- | --- |
| 原后端镜像 | `sha256:2848da261bfc92412fbd2af99719b4963cd6ad45a3bc834a476f844ee2406ffc` | 保留为回退可寻址镜像，不覆盖/删除；未来替换为1e构建回执中该服务镜像 |
| 原UI镜像 | `sha256:d106b5dfe863e4222214daa0bee42696f9243a9abf5e92308852aa51c71494b1` | 同上，使用同一固定SourceRef |
| 身份服务 | 原v4.17.1镜像、原容器健康 | 不更新/重建/重启。若前置健康不满足，停止两服务更新，不默认恢复依赖 |
| 原配置挂载 | bind，只读；容器目标 `/private/market-research/current-application.json` | 原宿主文件保留。新版本未来改为另一个经审查的新私密manifest绑定同一目标；不是覆盖旧文件或原样沿用旧结构 |
| 两服务当前Compose来源 | 同一 `candidate.compose.private.json`，文件存在 | 未来使用这两个服务自身的已核来源及经审查最终overlay；不从身份容器历史112个overlay列表重放旧部署链 |
| 网络 | 两服务沿原同一保留网络命名空间 | 保留；不重建该命名空间/数据库容器或改变网络。执行前fresh检查当前引用，不能复制过时容器ID |
| 原挂载 | 后端12项、UI5项 | 除明确批准的后端manifest bind外，类型/目标/只读标志/卷名逐项保持；数据库、对象、配置卷和其他服务均不动 |

本轮证据 `.local/selection-unified-config-plan-1e3509-20261104-01/runtime-metadata.json` 仅含容器/镜像/挂载/路径元数据，无env、命令正文、密码或Key。
补充只读记录位于 `.local/selection-unified-readonly-1e3509-20261104-01/`：原reader/writer的READ ONLY元数据、LAC无密钥绑定/版本投影、配置开关和3个已配置队列的describe结果；均不提交仓库。
本轮生成及解析记录位于 `.local/selection-private-a-parse-1e3509-20261104-01/`。独立私密候选为
`C:/Users/Henry/.codex/private/market-research-unified-20261104-config-a-01/current-application-1e3509-a.private.json`，
SHA256 `194B770F10B165768959CFC9B678D99F8F1BFECF4FA65387B87C02CFFF0786EC`。
新文件ACL仅Henry/SYSTEM，原凭据原值复用，未切入原后端/未发布；去密diff只含hash/字段路径/白名单目标值。
固定1e字段严格解析及未知/重复/尾随JSON拒绝检查PASS的输入为0600内存副本；该结果不覆盖真实bind权限。
此前同UID/GID 0:0、capabilities全删除/无网络/只读根的真实bind复核发现mode0777，作为更新启动BLOCKER保留历史。
本轮按用户明确授权，仅将这一个未发布候选mode收紧0600；候选hash194B…0786EC、原hash1271…及二者Windows Owner/Group/Access SDDL完全不变。
再用新工具进程在**同一实际目标只读bind**直接调用固定1e LoadConfig，文件门禁、严格JSON和完整静态validate均PASS，未复制或规范化输入。
uid/gid仍0:0、capabilities=0、network=none，不调用Run/DB/worker/provider。原服务挂载不变，未发布/构建应用镜像/更新实例。
历史失败回执见.local/selection-config-mount-review-1e3509-20261104-01/；当前修正与直接加载回执见.local/selection-config-mode-fix-1e3509-20261104-01/。
现存297镜像仅是此前构建证据，不包含本轮关闭态实现，不能拿来替代1e。1e镜像 **NOT_BUILT**，不编造新镜像digest。

## 必须一起审查的具体差异

| 内容 | 1e候选目标 | 保持与停止点 |
| --- | --- | --- |
| 后端镜像/来源标签 | 由准确1e标准构建回执确定 | 保留供应链四步、批次深链、图片入口、原不可读历史隔离和研究/选品人工准确交接；不使用旧应用镜像作为FROM |
| UI镜像/来源标签 | 同一1e标准UI产物 | 原身份/环境/端口/命令/会话密钥引用保持，不新增凭据或模型请求 |
| 后端运行manifest | 独立候选mode0600，新只读bind直接LoadConfig已PASS；单writer为127.0.0.1:5433/product_agent、pool1。3条文字bindings、productAgent=false、Generation仅enabled=false，完整图片模式及原读取owner保留 | 0777模式阻碍已解决，内容/hash/Windows ACL不变。仍仅在工具中加载，未应用到原服务；不把解析通过升级为正式启动或执行资格 |
| 文字用途声明 | 同一原Client的内容/研究/选品分别按当前prompt/schema声明；移除已退出PointTariff，新增用途要求contentOnly=false | 已写入独立候选并严格解析；准确Profile/候选标识见配置页。标识未发布，不复制旧current或独立点价，实际部署未应用 |
| 共享预算 | 已批准A：CNY、12步、2调用、72000 Token、300000微元、120秒；研究/选品各10点每调用 | 两项已批¥0.30/20点不变。原内容¥1/139264也按用户决定收紧；单次原最大调用可容纳、两次最大调用不可容纳。实际配置尚未应用，不新增按用途预算owner |
| 关闭态图片目录 | 原4条旧image声明不进入本次新bindings；不声明图片单位/正预算，保留非空workerConfigFile引用和完整历史模块 | 这是显式新文件差异，原a66文件、Image current、凭据及历史均不改，不换算旧点价。图片候选如实未准入；未来生成另行提供合法准入/预算，不能为关闭态伪造数值 |
| 关闭Chat元数据 | 保留enabled=false及其余原配置，只移除planningTextPolicies中退休PointTariff | 首次严格解析实际拒绝该未知字段，按同根因补齐；不启用Chat、不新增Chat候选/点价或兼容路径 |
| 企业模型选择/参考价/操作模式 | 不随镜像或运行manifest发布 | 原CAS/current不迁移、不覆盖、不填假revision。未Ready如实显示，不在验证时“先保存一版” |
| schema、角色、grant、连接额度 | 原Run reader/writer、图片API、Asset、Run配置及两审计reader的只读元数据已核；writer connlimit4，无需改凭据/授权 | ai_invocations执行合同仍不匹配，非关闭态管理前置。不执行installer/ALTER ROLE、不迁移/清库；元数据匹配不等于1e已装配或历史GET验收通过 |
| 图片/文字执行 | 关闭新执行，原历史与回执读取保留 | 不读取图片worker秘密文件/开启workerDB/图片Temporal；DataServices、观察准备模式、Knowledge及原商业恢复依赖保留，不冒称其停止。Ecoservices保留nonPaymentOnly=true，该分支无支付恢复回调；POD/Supply/Fulfillment不新增，Chat/两支付渠道不启用，独立采集桥保持退出 |

原Run、Image、Asset、Product owner和已有durable记录保持。预算仍每项¥0.30/20点、各一次合计¥0.60/40点；
原两调用及更小Token/byte/deadline限制不提高。参考估算不承诺厂商实际账单精度。

## 进入未来执行前的停止点

1. **配置条件**：[配置差异](selection-shared-model-wiring-preview.md)的writer/元数据已有证据；候选已按授权收紧0600并在直接只读bind下通过固定LoadConfig。后续只核同一文件hash/ACL/mode漂移及最终镜像User/Compose覆盖，不重复修正或使用复制品，不启动a66试错或跳过门禁。
2. **副作用条件**：productAgent=false、generation=false、Chat保持关闭；其他worker的保留范围已在配置页列明。保留配置不授权启动/消费：DataServices无独立执行开关，移除会破坏市场调研依赖；Knowledge和原商业恢复可能写事实，观察准备模式仍有worker。未来更新须单独覆盖这些启动影响，不能只批准两镜像就默认批准真实采集/付费或后台任务处置。不能用存在的任务做检查，不能运行重发/恢复/结算来证明关闭，也不在本轮修改源码或停止原a66。
3. **固定来源条件**：准确1e源码/原审核与CI仍为所批准应用范围；若发现需要业务代码变更，停回原PR修复/核增量，不能悄悄把SourceRef换为分支HEAD/main。
4. **操作授权条件**：构建、真实配置文件发布、两服务更新、业务只读检查分别在用户明确授权内。批准本文不等于批准它们；真实采集/AI/付费仍需另外授权。

本轮已收敛writer身份/配置表权限及LAC版本，不再将它们笼统记为UNKNOWN。
本轮已确定三用途准确字段和已批准A，图片按零元数据关闭态处理、旧4声明不进入新目录；非AI消费者保留范围已明确。
Image API的原权限查询required=true/forbidden=false，Asset所需列/约束/权限匹配，Run配置8表/图片参数列可读，两审计reader无写权。
独立私密文件和去密diff已生成；此前0600内存副本的字段解析证据保留。本轮已按授权修正候选mode0600，
在真实新只读bind下固定LoadConfig直接PASS、hash/Windows ACL不变，原0777门禁阻碍已解决，不再列为待授权修正。
候选未应用到原服务；真正更新前还须核最终镜像Config.User/Compose用户覆盖、文件/ACL/版本漂移，并明确保留消费者的启动授权；
1e真实启动、身份与历史GET只能在相应获准阶段验证，本轮未读业务行、不把NOT_RUN写成PASS。
共享三类选择和平台参考价尚未发布，真实文字调用还受调用记录表合同不匹配及用户授权限制。
因此本文仍不称DEPLOYMENT_READY；已生成的是未应用候选，不提供可直接执行的发布overlay或启动命令。这些不同层级的缺项不互相替代。

| 两服务更新前项目 | 当前结论 | 下一阶段边界 |
| --- | --- | --- |
| 候选内容/宿主ACL | hash194B…0786EC、Henry/SYSTEM匹配；实际只读读取hash也匹配 | 原文件/凭据/预算不变，使用前核漂移 |
| 候选Linux私密模式 | **RESOLVED：0600，新只读bind直接LoadConfig PASS** | 使用前只核同一文件漂移；若复制/替换文件则重新核实际mode，不能沿用本次证据。不改原a66或代码 |
| 固定1e前后端镜像 | NOT_BUILT，未获构建授权 | 后端仍用标准data-services-runtime；构建回执核准确SourceRef/tree/imageID/User，不能拿297/c26/旧镜像替代 |
| 配置发布与两服务替换 | NOT_AUTHORIZED / NOT_RUN | 当前没有可执行overlay；未来仅两镜像及后端manifest绑定变化，原网络/命令/其余挂载/卷/身份服务保留 |
| 保留消费者启动 | 源码与候选配置范围已明确，未获启动/消费授权 | DataServices、观察准备worker、Knowledge、商业/consumer/充值恢复及推荐成熟须明确；不是两个AI false即可承诺无后台副作用 |
| 关闭态与历史交付 | 正式启动/正常登录入口/历史GET NOT_RUN | 获准更新后沿既有入口检查；不跑生成、恢复、确认执行或真实商品操作 |
| 真实文字AI | 价目/共享选择/模式未发布；执行表合同差异仍存在 | 属未来真实执行范围，不随本次配置或两镜像发布，不迁移旧数据，不拿它阻止关闭态管理方案 |

其中商业/consumer/充值恢复和推荐成熟在server装配阶段就安排首次立即执行，Knowledge在Listen前启动，
DataServices及观察worker在Listen后、Serve前Start；没有用户HTTP操作也可能处理原任务。
当前原a66保持原状；本轮既不停止其消费者，也不启动候选来“证明没有任务”。若后续仍要求绝无真实采集/后台写入，
固定1e与当前保留配置不能仅靠AI关闭保证，须先明确产品/更新范围，不能擅加开关、清队列或提权绕过。

## 后续获准后的最小顺序（构建、发布、更新均NOT_RUN）

1. **fresh核对**：核准确项目、两服务当前ID/镜像/配置哈希/挂载与依赖健康。若与上述a66不同，先查并行更新差异，不覆盖。原私密Compose/manifest及原镜像引用留作回退，禁止把备份写进仓库。
2. **只读资源预检**：复用本轮原身份/配置schema/grant/连接额度证据，并在执行前核漂移及尚未核实的原历史owner；不自行提权或修复真实库。真实执行schema的差异不伪装已通过，也不自动升级成关闭态路径的前置。
3. **单独批准配置发布后**：权限修正与直接bind门禁已经完成，只采用这个同一候选，核准确hash/ACL/mode/去密diff，原a66文件保留。未来才切换经审查的新宿主文件绑定，容器目标不变；不重复生成或覆盖原文件，不借此改凭据/发布共享选择或价目。
4. **单独批准构建后**：运行下方标准入口，构建本地两镜像；核回执commit/tree/target/imageID/来源标签。失败不继续，不换源、不取消TLS验证、不触碰保留服务。
5. **单独批准更新后**：从两服务当前已核Compose来源追加最终overlay，只允许两镜像/来源标签和上述后端manifest bind差异。使用原project、明确服务名单和no-deps语义；不得应用身份服务历史overlay链、重建网络依赖或运行整个项目初始化。
6. **受限检查**：先后端健康/身份发现，再UI健康/版本；未登录研究/选品入口仍登录跳转、BFF仍拒绝未登录。需要登录后核管理/历史时先明确只读范围，不发prepare/start/confirm或模型“测试连接”。读取失败保留原错误，不以空历史或假Ready通过。
7. **后核保留项**：只两服务ID/镜像及经批manifest挂载允许变化；其他服务/配置/挂载/卷保持。记录实际关闭分支和未核项，不把页面可开当真实模型或用户验收。

未来构建命令模板，仅在单独批准构建后使用（本轮不执行）：

```powershell
pwsh -NoProfile -File scripts/build-listingkit-images.ps1 `
  -SourceRef 1e3509bda06ad9b8160b0c5a7c5d451de9496c61 `
  -Component All -OutputDirectory <新的任务回执目录>
```

标准输出标签为 `local/current-application:git-1e3509bda06ad9b8160b0c5a7c5d451de9496c61` 和对应UI标签；实际可用镜像ID以成功回执为准。
后端使用1e标准Dockerfile的 `data-services-runtime`；UI使用标准 `Dockerfile.listingkit-ui`。构建不推送、不更新实例或发布配置，不以297/旧发布镜像为基础拼补丁。
Compose更新命令须在获准后使用审过的真实文件路径填实；本轮不拼一条含占位路径的“可执行部署命令”。

## 回退必须同时覆盖镜像和后端配置绑定

若新后端启动/健康失败、身份正常入口或原历史读取回归、关闭态仍出现新执行资格，则停止继续更新。
未来回退只能恢复这两个服务的原镜像/来源标签，**并恢复后端原manifest宿主绑定**；仅换回a66镜像却继续挂新manifest不是有效回退。
恢复时沿原环境/命令/网络/其余挂载/卷；不重启身份或数据库，不删旧/新文件、镜像、数据和卷。

回退不恢复整个数据库，不覆盖新durable事实、不回写旧共享current、不释放预留/清UNKNOWN或运行恢复器。
如更新期间发生实际配置发布、schema/角色改动或业务写入，已超出此最小方案：保留证据，由原owner在单独授权下处置，不能自动用旧备份覆盖。
若出现超出两服务的并行更新或原配置文件缺失，也先停止，不从112条历史overlay试拼环境。

回退后的健康/身份/未登录鉴权与原两镜像/配置哈希重新核对，成功只说明恢复原版本；不签发新功能验收。
本轮原a66可用，配置/数据/卷保留；构建、备份、发布新manifest、两服务更新、现场业务检查及回退实际执行均NOT_RUN。
