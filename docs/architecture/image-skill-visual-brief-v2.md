# 图片技能视觉 brief V2

Execution: [#774](https://github.com/qq550723504/task-processor/issues/774)，同一 Writer / [PR #805](https://github.com/qq550723504/task-processor/pull/805)。
Design Basis: **Independent Architecture**；Admission: **IMPLEMENTATION_READY**。候选 46cb9db0662a1367df705ba967b5a644cc63ea82 已完成适用只读独立检查，无架构 BLOCKER；本准入提交先于生产实现。

## 产品决定、范围及依据

2026-10-18 原会话用户明确批准“按视觉 brief 合同继续修正”：三份官方图片技能增加独立视觉资源，只将真实商品事实、所选画面任务和视觉描述交给北京 qwen-image-3.0，不增加文字模型调用；冻结准确版本，保留全部旧技能、任务、图片、数据和配置。此前三个真实输出把 SKILL.md/资料画进图片，属于当前商品图片 happy path 的实际阻碍。原 2 条 UNKNOWN 各保留 1 点、3 张错误输出各已结算 1 点，不修改、释放或重发这些事实。

使用者：商品图片运营及官方技能维护者。结果：正常安装三个官方 v2 → 原商品图片入口准确选择 → 原计划/报价/确认 → 每个所选任务一次图像调用 → 可检查的商品图。任务/素材仍由用户选择，结果仍待人工采用。

Must：独立视觉资源、完整包精确校验与原可用性栅栏；不发送 SKILL.md、YAML、文件路径标签、方法边界标记、内部身份/版本/摘要字段；不编造事实或扩大任务/素材/权限；新策略进入原指纹；历史只读/恢复与新派发区分；幂等安装且保留所有既有内容/状态。Should：原选择、版本及失效提示在两主题清楚。Out of Scope：Skill 创作助手/新编辑器、个人图片直接执行、新 LLM 编译、任意脚本/外网工具、新表/事实 owner/账本/状态机/重试/恢复框架、平台发布、旧数据迁移/清理。无新增 Accepted Risk。

引用 [官方技能 V1](official-skills-and-image-method-v1.md)、[Skill V1](skill-capability-v1.md)、[百炼接线](bailian-product-image-wiring-v1.md)、[完整图片](product-image-agent-v1.md) 及原 ImageSet/Generation/Asset/Resource 合同。本文件仅替代旧设计“将全部方法包直接拼接给图像模型”和“本轮只发布 v1”的局部规则。

Figma Product Authority：文件 tg48P46SSXl6TBy9lZwg63 / 页面 31:463；选择/计划深浅 6861:1826 / 6861:1864、失效深浅 6861:1845 / 6861:1883。保留布局与主题，仅对齐“仅列可执行视觉版本；缺 brief 重新选择”的说明。官方目录仍展示真实历史版本，不伪造可执行性。

## Owner 与资源合同

Skill owner（internal/skill、persistence/skill）仍唯一拥有包、不可变版本、状态和许可。沿原 yaml.v3/goldmark 完整校验、16 文件/每文件 16KiB/总包 48KiB、规范排序和完整包 digest；不收紧通用包或更改标题语义。ImageSet/Generation 仍拥有冻结计划、报价、意图与唯一派发，Asset/Resource owner 不变。

当前图片执行资源为**准确路径 `references/visual-brief.md`**：恰一份、非空且非纯空白、UTF-8、允许 LF/TAB 而无其它控制字符，最多 4KiB。复用现完整包规范化（CRLF→LF）；资源位于完整包 digest 内，不接受客户端正文/单独 URL/声明 digest。内容是给图像模型的画面构图语言，不是步骤手册、示例商品或工具指令。三个内置资源只描述各自构图风格，无商品事实示例。运行时不回退到 SKILL.md、其它 references 或内置发布材料。

图片候选与新选择的准确解析在原 Store 的 live 权限/目标检查后，再检查此资源；缺少/超界则不列为可执行候选，准确选择显式拒绝。通用目录/查看/复制/导出不受影响，已有个人/团队包不改写。团队包只有自身准确版本已有合格 brief 才可被新图片执行消费；个人仍拒绝。实际使用再次在原 AcquireUseForTarget 栅栏内验证完整包与 ref/digest，再提取 brief。客户端不提供 owner、policy 或资源正文。

## 官方不可变版本发布

六个稳定发布 Key/Skill ID 不变。标题继续原 v1 原始字节；图片 v1 原始字节保留，在同 ID 追加 v2（原 SKILL.md + 各自视觉资源）。发布目录声明准确版本及原 v1 包，不以名称匹配或隐式升级。

复用显式 InstallOfficial、原平台维护权限和单事务安装锁；不在启动/GET 安装。空目录先建立原 v1 再追加 v2，最终仍六个 Skill；当前六 v1 目录只新增三个 v2。命令只返回六个本次发布目标准确引用及新增/已存在，不把内部历史版本数当技能数。

追加前锁定原 Skill 行并核对平台 owner、无复制来源、准确原 v1 digest，当前 head=1 且该原版本仍 ENABLED/SHARED，才沿原 publish/enable 状态转换插入 v2 并 CAS 更新 head/revision。原 v1 行/状态/许可/创建时间不变；新 v2 沿原官方许可。任何人为新 head、冲突版本、错误归属、停用/撤销的原 v1 均拒绝整个事务，不覆盖或重启用。准确 v2 已存在时校验其 digest/归属后只读返回，即使后来被停用或 head 已前移也不改它。重复/并发安装不增版本。原 title/个人/团队版本与配置不写。

## 策略、冻结与调用路径

新选择固定 `image-method-visual-brief-v2`，旧 `image-method-prompt-v1` 仍是**已保存事实的有效版本**；binding.Valid 可识别两者，未知 policy 拒绝。不得把旧绑定自动映射到新策略、替换旧版本/正文或用新代码重算旧 digest。

合同 → 实现 → 注入 → 消费者：

1. 原 Skill HTTP/BFF 候选/准确选择调用 Store 检查合格资源；现 SkillPicker 只显示这些准确版本。历史 Run API 的 policy 枚举接受 v1/v2，准确版本摘要沿原展示。
2. 原 PrepareImageSet 用新 policy + exact SkillVersionRef，取得原 Skill fence；在原 ImageSet 构造器中将商品标题/类型/排序属性、所选任务画面说明、背景/文字语言、已确认 evidence 组成确定性的自然语言视觉指令。无 JSON/YAML 序列化块，无原图 URL、ID、版本、内部字段名称或文件标记。保留真实参考图、事实优先与只允许明确请求的顾客文字的约束。无方法路径仍按当前旧构造器，不追溯影响已有计划。
3. recipe.Prompt 冻结事实/任务指令，binding 冻结策略与完整包 ref/digest。最终提示词字节**确定性地等于冻结 recipe.Prompt + 固定视觉过渡句 + 准确包中的 brief**；准备、报价、派发复用同一组合器，每处校验最终 ≤16KiB，不截断。既有 slot/input/plan/quote fingerprint 已包含 recipe 和 binding，因此完整冻结视觉输入，不在 checkpoint/API 复制方法全文或新增摘要事实。
4. 原 skillimage adapter 在准确包/权限栅栏内按传入 policy 返回仅 brief；旧 policy 只校验准确包/可用性并返回空视觉材料，不再返回原文。ImageMethodContext 接收 policy，所有实际准备/派发/采用装配显式传递，不用默认猜测。
5. Generation 的报价和**不存在旧 fact 的新派发**在 PrepareProvider / reserve / intent / transport 前拒绝 v1 policy；route 和直接 tools 生成也明确拒绝 v1，不容许绕过上层。新确认在原 admission receipt 恢复分支之后、写新准入之前同样拒绝 v1。原未选技能路径保持。任何旧新派发候选均需要重新准备新版本计划，不修改旧计划；新 request key 与不同 payload 按原冲突合同处理。已有准备请求按已存 binding.PolicyVersion 计算并比较唯一 input digest，返回其原冻结计划；仅不存在的请求使用 v2，不尝试多个策略或改写旧 digest。
6. 已存在 admission/confirm/Asset receipt 按原准确回执优先恢复；已有 prepared/UNKNOWN/success effect 的恢复只由原 owner 读取/结算/下载物化，不重新报价或 POST。原结果的新人工采用仍可经 v1 准确包与当前权限/可用性 fence 检查，既不需要 brief 也不获得新 POST 许可。Regenerate/新增 slot 属新派发，v1 必须拒绝。原历史 Run 读取与 UI 展示不因旧 policy 失效。
7. 最后沿既有 ProductImage SourceEdit → 已冻结百炼 route → 单次 HTTP；只有此一个图像模型调用。固定北京/qwen-image-3.0/同步协议、1点价格、凭据 reference、source byte/digest、request deadline、观察/结算/下载顺序不变。不改 provider 参数或新增 text model。

## 权限、持久化与失败

沿原 current principal/企业/成员/图片能力/平台或当前团队版本检查。Skill shared-row fence 持有到既有 prepare/confirm/new effect/Asset commit 返回；撤销写锁、锁顺序和请求超时不变。资源内容始终是受约束的表达材料，不能授予网络、工具、预算或事实权限。

安装仅 Skill DB 单事务；运行计划/效果和 Resource 仍各自原 owner/原恢复协议，不增加跨库写。缺 brief/超界/旧策略/失权在新 intent/reserve/provider 前拒绝；已存在事实不通过这些前置检查获得再次执行许可。取消/断连/响应丢失仍按原 before-effect/UNKNOWN 规则；provider POST 一旦交付 transport 不自动重发。成功观察后的下载恢复不变。日志不写正文/密钥/base64/签名URL。

Legacy decision: N/A；消费当前已保存事实及其准确策略，不恢复 legacy task/queue/worker，不新增兼容 adapter、双读双写或迁移。v1 可读不是可重执行的 fallback。

## 有界验证与交付

先 TDD 捕获真实缺陷：三个官方准确包经生产 adapter→组合器→factory/SourceEdit→受控 HTTP，只出现 brief/真实事实，SKILL 正文哨兵与文件/边界标记零泄露、一次 POST；缺 brief、篡改 digest、超界零派发。新策略/recipe/ref 绑定、相同键冲突；v1 历史 projection/receipt/success/UNKNOWN 恢复保持且新 quote/confirm/generation 拒绝。沿已有权限/撤销测试验证，不新增验收平台。

现有 PostgreSQL fixture 验证六 v1→三个 v2追加、全部原行保留、标题和个人团队不变、停用/人工新 head 冲突、并发幂等及事务回滚。前端只对 policy 枚举/候选解释及旧新版本展示作必要检查，复用现主题。

完成适用独立设计检查、IMPLEMENTATION_READY 后同 PR805 实现；最终检查只看真实增量及有效证据。获准保留实例按当前 source combination 最小更新必要后端/UI，与已有 UI owner 顺序协调，不覆盖其变更。显式安装 v2，原三技能各新计划/正常确认一次真实 IKEA 原图调用，检查输出商品形态、场景/详情区别与文本、费用及历史保留；任何再次画文档则停止继续调用并如实报告。保留所有旧错误图/UNKNOWN、配置/卷/凭据。没有 Apply/发布/Issue关闭/#805合并授权，不将开发检查或 AI review 当用户验收。

## 独立设计检查

Round1：独立只读 Reviewer 核对候选 46cb9db0662a1367df705ba967b5a644cc63ea82 及实际 Skill Store、ImageSet、Generation、Temporal recovery、Asset 消费链，结论 IMPLEMENTATION_READY，无 BLOCKER。没有生产修改或真实调用。

三项均分类 IMPLEMENTATION_TEST（当前 Must 为历史准确事实不变、新派发受控、旧版本保留）：(1) 旧同键按存储 policy 计算 digest，v1读取/确认回执恢复与指纹不变；(2) v1空材料+lease仍可被历史/采用消费，Generation/route/直接tools新生成拒绝，旧success/UNKNOWN恢复零POST；(3) PG并发追加/冲突回滚/旧行保留及三个真实包生产adapter→组合器→factory/SourceEdit完整受控HTTP。上文已明确两处易漏调用细节；这些是冻结方案的实现验证，不扩大 owner/状态/权限边界，不需要重开设计。最终交付复核只检查本次实际增量。

真实试用增量：6ba39cbea 的首张 v2 Run0e8ea735-f5ce-5c66-abac-27c4ba5d9631仍夹资料文字；后续调用停止。准确资源和原图 SHA 已确认，原文 SKILL 已排除，但构造器错误地加入整个 source.Evidence（含来源描述、内部试用说明/URL）。分类 IMPLEMENTATION_TEST，属于上述只发送当前视觉任务/真实事实的实现遗漏；新 recipe 只消费 RequiredTaskEvidence 当前指定键，不加入其它来源说明。所需规格/使用/配件证据检查及 digest 保持，旧 recipe/policy/版本/事实均不重算，不改 composer/provider 参数或增加模型调用。RED 已捕获两个任务的来源描述泄露，最小修复及增量复核沿同一 PR；真实视觉效果不因开发检查而转为 PASS。
