# 官方技能直接处理已有商品

Design Basis: Reuse Existing Architecture · Ready（2026-10-19）

## 用户结果与产品依据

Issue #774 当前用户授权：六个官方技能的准确版本页可直接选择当前企业已有自有商品；标题进入原优化确认，图片显式确认单件转入后直接选择已有原图。Figma 同一文件 tg48P46SSXl6TBy9lZwg63，深色 7533:2080/2099/2118/2137，浅色 7533:2156/2175/2194/2213。均为可编辑设计，复用原技能库组件、主题变量和 Noto Sans SC。

## 复用依据和调用路径

引用 official-skills-and-image-method-v1.md、image-skill-visual-brief-v2.md、skill-capability-v1.md、product-content-localization-v1.md、product-image-agent-v1.md、own-product-entry-v1.md 及现有单件转入/CompletionDraft 合同。

- 官方准确版本详情 → 选择已有商品（原 listOwnProducts，搜索/分页）→ 标题原 OwnProductOptimizePage → ProductAgentPanel → TitleAgentConfirmation/SkillPicker → 原内容运行、Review 和 Catalog 独立采用。
- 图片选择商品 → 原 OwnProductPreparePage 的显式单件 transfer → 原 listSupplySources 精确回执读回 → 专注原图的页面（原 readSupplySource 校验 item/preparation/kind）→ 同一个 ProductImageSetPanel kind=supply/contextId=sourceId。不新增 Collection 图片上下文，不要求保存标题描述草稿，不绕过原图来源规则。
- 所选 SkillSelection 的 skillId/version/digest 作为有界 URL 导航提示，绑定发起 user/org/target。不是执行请求或授权事实，不持久化新事实，不创建新恢复协议。失效/缺字段/身份企业不匹配时明确阻止继续；不得降级为无技能或自动用最新版。执行前仍读取当前 candidates，原服务端准入不变。

## 不变量与边界

导航和只读 GET 不创建商品/来源/计划，不调用模型、不采用。只有原显式 transfer 确认才能产生单件来源，沿原 CAS、幂等 key、回执和 pending guard；有原请求时不重建参数。入口自身不改任何 local/session storage，不覆盖原请求、agent_key、图片 intent/run 或历史计划。SkillPicker 的预选只影响新确认/新计划；原运行继续使用原 frozen skill。图片 UNKNOWN、锁、核实、保留历史请求以及结算均由原组件和后端决定。

原 user/org/权限/feature gate 不放宽：商品读取、标题运行、图片读写、单件转入分别沿原 owner 校验，scope 变化即拒绝旧提示。Collection 保留不可变原资料，Supply 保留已固定来源，Skill 唯一版本 owner，ImageSet 冻结方法与报价，Asset 明确人工批准；无新状态机、表、事务、权限、计费、恢复或外部副作用合同。URL 内商品/来源 ID 仍需原接口按 actor/org 鉴权并验证对应关系。

本轮不包含旧审核状态刷新、登录修复、额度调整、技能创作助手、个人/团队新执行范围、付费调用、部署、数据清理、迁移或发布。Legacy decision: EXTRACT；复用的是当前 owner 的单件转入与图片组件，不依赖已退休 ProductImage/Queue/worker，不新增 fallback 或第二事实源。

## 验证与交接

先 RED 覆盖准确版本预选不丢失、消失版本不静默换选；入口 scope/严格参数、只读选择、商品搜索分页、受限权限/跨企业；标题确认与图片 prepare 消费同一准确引用；单件转入后来源身份匹配及旧请求原参数；原图片 UNKNOWN 保护回归。相关既有 Vitest、类型/lint、必要生产构建；深浅主题使用现有 tokens。保留实例不更新，真实生成/转入/采用/付费 NOT_RUN。实现后按规则独立最终交付复核，交用户审阅，不签用户验收。
