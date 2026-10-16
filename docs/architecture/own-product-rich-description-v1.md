# 自有商品描述富文本 v1

Refs #745。Design Basis: **Independent Architecture**。Admission Status: **NOT_READY / DESIGNING**。设计先于正式代码；独立评审只检查本轮格式、安全及其调用边界。

## 产品结果、权威和范围

2026-10-17 用户批准：先明确格式、安全过滤、历史比较/恢复和平台带入合同，再做自有商品描述富文本编辑、保存与预览，覆盖草稿和历史；独立页面，复用图片上传和版本保存，保留数据、图片、配置、授权，不更新 own675、不调用真实平台、不自动发布。

Product/Figma Authority：[当前 IA](../product/final-ui-ia-authority.md)，tg48P46SSXl6TBy9lZwg63 / 31:463，自有商品 3500:1100、添加 3502:405、待补全 2667:1948；用户独立编辑/历史页面决定优先，使用原 ConsolePage、导航上下文、主题 token，无新一级菜单或虚构 Figma frame。

Must：新建/编辑可排版、插入已有来源图片、原版本保存并读回；详情/固定来源/候选编辑/历史/恢复可安全预览；格式变化可见，恢复旧描述及格式为新 Rn，保留当前属性/价格/库存；平台带入明确有损文字投影且原内容不变。
Should：基础工具可键盘操作，深色主题和窄屏可用，粘贴去掉不支持内容且提示限制。
Out of Scope：任意 HTML/CSS、脚本、视频、iframe、表格、字体/颜色、协作、草稿新图片上传、真实 AI/平台调用、发布、新 schema/权限/配置、数据迁移/回填/清理、既有试用服务/镜像变更。隔离开发自检不操作保留实例。
Threat model：当前有权限用户提交不可信描述，直接 API 不信任编辑器；防存储 XSS、危险 URL、代理绕过、跨来源图片替换、丢失格式、恢复错误覆盖、并发覆盖及幂等载荷混淆。既有公开图片 URL 的公开性不变，不新增私有对象授权或通用 SSRF 规则。

## 单一内容事实及格式

复用现有 description 字符串，增加可选 descriptionFormat = "html-v1"；缺省/空表示原纯文本。ProductSnapshot/Canonical JSON 用既有 snake_case：description_format；Collection/CompletionDraft DTO 用 descriptionFormat；SourceEnvelope 新字段空时 omitempty，保留原纯文本 JSON/hash。其他格式拒绝。

description 是该格式下的唯一原始内容，不另存 text/HTML 双事实。纯文本即使包含 <b> 或 script 字样也只作为转义文字，不自动侦测 HTML。当前纯文本保留原字节/旧回执/hash/历史，无批量重写、双读/双写或旧 owner adapter。只有用户实际修改描述时，编辑器输出 html-v1；只改标题/属性时保留原 description 和缺省格式。打开编辑器不自动保存或转换。

html-v1 支持 p、br、h2/h3、strong、em、s、ul/ol/li、blockquote、a、img。仅允许 a.href/target/rel 和 img.src/alt/title；不允许 style/class/id、事件、srcset、data/blob/relative/protocol-relative URL、外部嵌入、表单、SVG/MathML。href 只允许规范 HTTPS、无 userinfo/fragment；target 只允许 _blank（并强制安全 rel）或省略。图片同样 HTTPS，src 必须逐字位于当前保存商品公共/变体图片池，候选必须位于固定来源图片池；任意替换 URL、跨来源对象不准入。图片仍是来源候选，不等于 Asset 批准。

使用成熟 Tiptap/ProseMirror（MIT）编辑器；前端 sanitize-html 过滤粘贴/渲染，服务端 bluemonday + x/net/html 实施相同固定 allowlist。服务端拒绝过滤后内容发生语义变化的 HTML，不静默丢弃后写入；允许实体/空元素等纯序列化差异，保留请求原安全字节以便精确幂等。HTML 文档、注释、危险标签/属性/URL拒绝。不重新发明 HTML 解析器、编辑器或通用过滤框架。

写入前同时检查原字符串与过滤结果、UTF-8/NUL、格式、节点/嵌套边界（最多 1024 节点、32 层），前端显示原 UTF-8 字节数。沿用 Product 描述 8192 字节、SourceEnvelope aggregate/encoded limits、CompletionDraft 描述32768/整体64KiB、Collection2MiB；不隐式扩容。过滤只在内存，不访问 URL/对象/网络。

## owner、调用与安全渲染

Product 内容语义局部组件 internal/product/description 提供 Validate(value,format,allowedImages)、PlainText；无数据库、网络、权限或新状态。Collection OwnEnvelope → SourceEnvelope → Catalog snapshot/canonical → 原持久化/精确读取完整携带格式。Catalog 写边界验证富文本及图片引用；Collection 更严格限制8192字节。内容优化 Review 对标题变化保留格式，显式 AI 描述替换仍按原纯文本合同清除格式，不将模型输出当 HTML。

前端通用 DescriptionEditor/DescriptionView 复用于新建、编辑、详情、固定来源、CompletionDraft、旧版/恢复预览；所有渲染先固定过滤，图片 src 通过原 /api/image-proxy，原代理 TLS/allowlist/本机试用政策不变。链接固定 rel=noopener noreferrer nofollow，不透传 event/style。未知格式前端合同拒绝或显示错误，绝不直接 innerHTML。编辑器与预览图片也使用代理，保存的始终是原来源 URL。

插图复用新建/编辑页现有 SourceImageUploader 和公共图片池：先上传确认成功并加入池，再选图插入描述；不新建上传入口、存储/容量 owner 或恢复协议。移除图片池中仍被描述引用的图会阻止保存并提示先移除描述引用，不自动改其他内容。草稿只能使用固定来源已有图片，不新增候选文件或偷偷跟随新 Product 版本。

原 Collection/supply read/manage、org/actor/member 绑定、SourceSelector 事务内原出处证明和撤权校验不变；预览不赋权，不能绕过当前 read。contract → implementation → injection → consumer：Collection/Completion 公共 DTO → Product 描述验证 + 原 Collection/Completion 服务与仓储 → 当前已有 HTTP/BFF 注入 → 独立页面；不加 runtime 组件/feature flag/owner/table/grant。

## 历史、恢复、并发与失败

格式是版本内容，纳入原命令 JSON/hash、不可变版本、pending 意图及同键不同载荷检测；空格式 omitempty 保留原纯文本编码。原 key、CAS、固定 source Vn、原库事务、回执查读、UNKNOWN 与准确请求重试全部不变。服务端过滤不改 hash 或请求载荷，无模糊等价重放。

商品/草稿比较明确显示“描述格式”行（纯文本/富文本）；描述原值变化完整显示安全预览，格式或排版变化不可当作没有变化。必要时提供转义源码作准确审阅，不执行 diff HTML。相同字节/格式视为相同，不引入新 diff 持久化事实。

恢复页读取准确旧 Rn、当前 head 和固定 source；只把 old.title/description/descriptionFormat 覆盖到当前 head 的 clone，保留当前 structured。plain 旧版本恢复必须清除当前 html-v1 标记；只格式变化也应允许恢复。确认仍调用原 Save(expectedRevision)，冲突重新读取并确认，无自动覆盖/新键重试。原版本和原 Product 永远不改。

验证/过滤失败无持久化；上传已成功但商品未保存仍按原容量事实保留，不清理对象。响应丢失/取消/重启继续原 pending 查读，不能转换格式后重试旧 key；重复保存仍沿原回执。历史读取不重写/重净化事实，显示层防御过滤不冒充原历史字节。

## 平台带入和其他消费者

SHEIN 当前 goods Description 是现有 LanguageContent.Name 文字合同，不据此推定任意 HTML 支持。本轮按确定性 PlainText 转换：保留段落/换行/列表边界及链接可见文字，去掉排版/链接地址/图片，不下载、复制、批准图片。预览明确告知有损文字带入，原 html-v1 候选保留；选中 description 才覆盖原英文描述，其他语言/未选字段/图片/Asset均不动。最终 Save 仍服务端从准确候选 Rn/base Target Rm 重算并检查全部值、rules/merchant hash、CAS及当前权限，不接受前端自报转换值。

所有当前将 Product/Completion 的 description 带入既有纯文字消费者的路径（初始 SHEIN/TEMU、TikTok/Walmart carry、现有商品 AI evidence/plain contract）使用同一纯文字投影或明确拒绝，不能把 markup 当文字或让格式标记丢失后误解释。UI 与 Go 投影用同一用例固定换行/实体/链接/图片规则；纯文本逐字返回，富文本空内容可为空。共享 projection 是临时派生值，不存第二 Product 字段，不改平台权限/发布或外部副作用协议。现有平台 draft 手工文字编辑保持原合同。

## Legacy 与验证/交接

Legacy decision: N/A。只演进当前 Product/Completion/Target 内容合同；不依赖 RETIRE owner、不增加 compatibility/fallback、schema migration、双 owner、重试 runner 或验收平台。
TDD：先捕获格式传播/纯文本旧hash、直接API危险HTML/URL/图片绑定/容量限制、safe实体序列化、正文纯文字投影、title-only保留/description替换清格式；原保存/历史读取链、同键冲突、来源固定和恢复 structured 保留。前端用现有 Vitest/Testing Library 检查真实编辑器或其组件边界、粘贴/显示安全、保存/回填/格式diff/恢复、pending及平台预览；复用现有测试设施，不造新runner。必要 typecheck/lint/build及稳定候选CI；独立最终检查读真实diff/受影响路径。开发自检和CI不等于用户验收。
交接提供 PR、现有独立路由及正常启动方式；own675 不部署、不改配置/库/图片/授权，真实平台/模型/发布与用户试用 NOT_RUN。

依据开源官方文档：[Tiptap Next.js](https://tiptap.dev/docs/editor/getting-started/install/nextjs)、[StarterKit](https://tiptap.dev/docs/editor/extensions/functionality/starterkit)、[bluemonday](https://github.com/microcosm-cc/bluemonday)。复用 [Product edit](own-product-edit-v1.md)、[结构化候选](completion-draft-structured-v1.md)、[恢复](completion-draft-restore-v1.md)、[SHEIN带入](completion-shein-import-v1.md)、[SourceMedia](own-product-source-media-trial-v1.md)。

