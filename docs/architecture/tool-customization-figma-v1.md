# 工具定制完整 Figma 表单与私有参考资料

Status: IMPLEMENTATION_READY. Design Basis: Independent Architecture. Execution: #730.

2026-10-16 独立只读 Architecture Review 确认设计 commit 0de8660e8 / blob f27ef4128e21baaf052b3d6feccd1beb5c762ba2 无架构 BLOCKER，私有下载和原scope/key大载荷恢复边界可实施；具体实现与故障验证在原批次收敛，不新增全局机制。

## Product outcome and authority

2026-10-16 用户先要求工具市场所有页面对齐 Figma，随后明确撤回“三字段流程，其余列为差异”的选择，要求直接完整对齐。当前决定替代 #730 原轻量展示范围：用户可填写名称、使用场景、类型、执行规则、可选参考资料、联系人、手机或微信，并明确同意专员联系；提交后刷新可见全部原始需求及下载资料。原企业启用、实际采集及人工进度继续使用现有 owner。

Figma Product Authority：[final-ui-ia-authority](../product/final-ui-ia-authority.md)，文件 tg48P46SSXl6TBy9lZwg63 / page31:463；首页404:1639/411:1898、官方430:3479/3678、我的430:5511/5710、介绍430:4495/4694、提交6395:437/694、进度6395:951/1159，本会话已实时读取高保真上下文和截图。使用场景原型未规定选项，沿现有智能体定制使用必填自由文本，不发明业务枚举。提交表单和需求列表/详情的四种类型显示数据采集、系统连接、自动化流程、其他；原 OUTPUT 在这些需求视图中显示为“其他”，不新增执行能力。定制介绍页的能力说明卡按430:4495/4694保留“输出与权限”，与6395:437/694提交表单的“其他”分别遵循各自Figma节点。

Must：完整表单真实保存、原样回读；显式联系同意默认不勾选；可选资料持久化并受同一企业/平台权限下载；提交/附件/初始事件/回执原子写入；未知结果保留原键和原载荷，不能生成第二需求；阶段与记录沿已有工具合同，不因示例伪造付款/验收/安装。

Out of scope：自动联系、邮件/SMS发送、在线支付/报价owner、自动安装/执行任意代码、Agent调用、文件预览/解析/知识索引、对象存储平台、旧数据迁移/兼容、共享或生产部署、真实数据操作、改变已有权限和状态转换。

## Existing capabilities, owner and implementation path

复用 [tool-market-v1](tool-market-v1.md) 的 toolmarket owner、GORM/PostgreSQL单事务、命令原键/强revision和CachedRead/LiveWrite授权；复用 [agent-customization-v1](agent-customization-v1.md) 已有同意、2MiB参考资料/bytea单事务下载及大载荷pending的有界实现模式。文件类型用 Go net/http.DetectContentType 标准库；严格JSON沿 sigs.k8s.io/json、浏览器Zod。无需新Agent工具、通用工单/上传平台或第二事实owner。

contract → implementation → injection → consumer：
toolmarket Demand/Attachment/Repository.Download → integration/persistence/toolmarket 同库PG → 已有 Handler/BuildRoutes/current application注入 → 已有 /api/tool-market BFF → 工具提交、企业详情、专员详情。
Repository接口新增下载方法；current application仍注入同一个具体Store，不新增配置或对象库。共享大载荷缓存只抽取智能体定制现有字节保留行为为窄工具函数；各域保留其自身schema、path、scope和intent校验，不抽象业务流程或新恢复owner。

## Facts, persistence and transaction

Demand：kind（DATA/CONNECTION/AUTOMATION/OUTPUT，UI OUTPUT显示其他）、title<=120 Unicode字符、description<=4000、scenario<=240、contactName<=80、contactMethod<=160、consent=true、files可选最多3个。均拒绝控制字符；多行描述允许换行/tab。不存在字段不补默认、不兼容旧三字段提交。

requests 增加 scenario/contact_name/contact_method/consent/consent_version；created_by + created_at 是明确同意的actor/time，consent_version固定 tool-customization-contact-v1。不可由客户端指定版本、actor或时间。原需求内容不提供编辑。列表仍只返回当前轻量summary，不带联系资料；详情返回上述输入、createdBy、consentVersion及附件metadata，不返回base64。

同schema新增 attachments：id UUID、request_id FK、name、content_type、sha256、size、data bytea；最多3条由服务和事务写入限定。每文件非空、<=2MiB，名称<=200字符且不含路径/control，MIME由服务探测仅PDF/PNG/JPEG/UTF-8纯文本（含CSV）。不接受HTML/SVG、压缩包或office可执行格式。参考资料作为不可信字节，仅下载；不执行、不远程抓URL、不公开对象链接。

首次 Execute(create)沿原命令事务：当前授权guard → scope/actor/key锁和完整命令fingerprint → request及全部attachment → 初始event → 最终授权guard → receipt → commit。任何失败全部回滚；同键不同任意输入/文件字节冲突；响应丢失重放原回执，必须重新授权且不重复文件/需求。授权失效零新增事实。进度更新、强revision、终态、历史分页均不改变。附件无独立上传阶段，避免跨存储事务/孤儿文件/额外恢复状态；无需outbox/Saga。

fresh schema定义一次性扩展；InstallSchema仍拒绝已有schema，Serving只VerifySchema检查新列及附件表。无ALTER/迁移/fallback/旧记录默认补齐。已有保留实例不改、不重置；准确新候选只在全新隔离数据库验证。部署/保留环境如何升级不在本次授权内，不将代码交付宣称运行已升级。备份须整体覆盖tool_market schema。

## Permission and threat boundary

企业需求/参考资料沿 workbench.tools.read + CurrentIdentity/CachedRead 和当前企业predicate；提交沿 customize + LiveWrite/事务内相同actor/org guard。当前企业有读权的成员可读该企业联系人/附件，与企业共享需求一致。附件通过 requests JOIN绑定当前企业；跨企业request/file ID均NOT_FOUND。

专员下载新增 GET admin/requests/:id/files/:file，精确复用 verified platform roles / OrganizationAccessNone / platform_admin；不依赖客户成员身份、不从body或query生成平台权。UI仅调用当前scope对应端点；客户端权限不替代服务端核验。

下载每次重新授权、校验UUID和精确request/file绑定，有界读取<=2MiB，验证保存的size/SHA256；Content-Disposition用mime.FormatMediaType生成attachment名称、nosniff、private no-store，不内联预览。BFF验证状态/MIME/大小和Content-Disposition，浏览器使用经过metadata校验的文件名下载；不向用户返回provider token或内网地址。

联系同意只允许专员就本需求沟通，不替代登录/组织/权限或付款同意。无自动联系副作用。接口不记录正文/联系人/文件内容到日志，错误仅既有code；React文字渲染，不注入HTML。

## Bounded HTTP, BFF and recovery

仅POST requests 允许JSON envelope<=9MiB（3个2MiB附件base64约8MiB）；其他写请求保持16KiB。响应JSON仍256KiB：详情单条<=约35KiB +16条2000字符最坏转义约192KiB +固定metadata，低于上限；列表不增隐私字段。企业/专员新增下载路由在精确descriptor验证中登记；保持10s请求deadline、BFF12s/外层15s，有界本地6MiB写入，不扩大其他provider超时。

前端发送前先验证类型/文件数量/大小、读取文件并冻结完整意图。文件准备期间阻止重复提交及组织切换，组件卸载后不派发。沿既有ResourcePending保存短命令；大命令仅原scope+key的完整字节驻留有界缓存，sessionStorage保存小marker（路径、原key、CAS/absent、SHA256），经精确写入回读确认后才dispatch。最多18MiB缓存；挤出原载荷只保留marker，暂停新提交并允许只读查看原需求；不自动换键。短命令含联系资料仅浏览器会话存储，不新增localStorage持久化。

未知响应可原scope+key重试同完整body；导航/回到原组织恢复内存载荷；刷新丢失大载荷时保留marker并显示“附件载荷无法恢复，请在进度查看并联系平台核实”，不能误报失败或重发新需求。与已有智能体定制相同的当前受限体验，必须明确交接；不为此新增通用对账平台。已知400/404/412按原合同清除该命令；5xx/断连/超时保留。缓存工具不绕过原scope/path/schema授权。

## Validation and delivery

TDD：缺场景/联系人/显式consent被拒绝；文件超限/类型/路径及未知字段拒绝；全字段与文件同事务保存/独立重读；同键重放和任意contact/file差异冲突；跨企业/跨request附件404；平台路径无需客户成员但必须平台授权；后半事务失败零附件/需求/事件/回执；原进度CAS与分页保持。

HTTP/BFF：9MiB仅create、严格JSON、下载安全headers/MIME/大小、原身份/企业/同源门；UI：未勾选不提交、类型与字段/附件完整payload、提交跳详情/刷新回读、下载、失败/空态、UNKNOWN原键/大载荷marker、组织切换与准备中卸载。复用已有测试设施、隔离PostgreSQL，不建立新验收runner。

截图验证六页桌面深浅和390px，使用真实组件/生产构建与明确合成fixture；开发测试不签发产品验收。最终交付一个主要PR与正常启动/路径、限制；用户/真实专员联系/共享运行验收保留NOT_RUN。未经授权不合并或关单。

Legacy decision: EXTRACT（已有同意/bytea附件及pending字节保留行为进入当前toolmarket与窄共享缓存工具）/ RETIRE（旧三字段正式提交合同，直接切换，无兼容）。不触碰旧Service、Task-first、迁移或双读双写。

## Review admission

按AGENTS.md独立Architecture Review审核本次新字段、隐私附件及事务/恢复增量；不重新审核已覆盖的启用/采集/计价。新范围在IMPLEMENTATION_READY之前不修改生产业务合同/schema；既有已获准展示变更保留。
