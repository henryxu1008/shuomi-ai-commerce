# Image Agent Workflow

图片执行复用当前 ImageAgent 与 Temporal。完整用户能力见[商品图片智能体](product-image-agent.md)，操作入口见[图片使用说明](../operations/product-image-agent.md)。本页定义运行职责，不维护阶段交付路线图。

## Owner 与运行路径

1. Product 的 exact 来源、Catalog 版本和 Asset 事实经授权后的窄读取端口进入计划。
2. 商品级 Run 绑定 actor / org / member、不可变 Plan revision、模板和来源版本；每个 Slot 与 attempt 有唯一效果身份。
3. 用户确认准确计划与预算；模型规划只产生候选，不授予执行、资源或发布权限。
4. Temporal 负责跨进程执行、等待、取消和恢复；有界 planner / provider 活动不建立第二套持久业务状态机。
5. 每个实际生成沿固定 Image owner 的资源准入、dispatch fence、provider 观察、terminal proof 和原 operation 结算。
6. 结果由用户预览并显式决定完整采用集合，再进入 Product Asset 不可变批准与 inventory；生成成功不自动批准或发布。

## 状态、效果与恢复

- Run / Plan / Slot / attempt 是执行身份，不以提示词、latest 配置、报价变化或新请求键重新解释原调用。
- 派发前持久化不可变意图和资源绑定；同键不同载荷拒绝。相同效果不能由 HTTP、worker 和 UI 各自重试。
- 结果丢失、超时、撤权、取消及晚到响应按原 effect / UNKNOWN 合同处理；没有可靠无副作用证明不释放占用或重复派发。
- 部分失败保留已成功结果与原费用。明确重生成是新计划/attempt 和新预算决定，不是绕过原 UNKNOWN。
- 用户可见进度来自原执行 owner；BusinessTask 仅投影其状态、待确认和取消入口，不接管重试、计费或业务事实。
- 风格参考只提供风格，不能替代商品真实身份或把未经确认的文字、规格、品牌放进输出。

## 当前专项合同

- [固定 Image 资源回执与成员限额](source-white-background.md)
- [预算与授权](image-agent-budget-authorization.md)
- [效果策略](image-agent-effect-policy.md)、[终态](image-agent-effect-terminalization.md)、[结算](image-agent-effect-finalization.md)
- [效果恢复](image-agent-effect-recovery.md)、[工作流恢复](image-agent-recovery.md)
- [候选采用与发布门禁](image-agent-candidate-release-gate.md)
- [安全图片下载](image-download-safety.md)、[生成原图恢复](vip-original-output-recovery.md)

权限、租户、预算、来源真实性和不可重复副作用规则均由这些当前 owner 合同与行为检查执行。具体能力开放、部署及真实模型效果由当前执行 Issue 和用户验证分别确认。
