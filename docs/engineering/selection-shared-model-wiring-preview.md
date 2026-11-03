# Local Account Center：共享文字配置差异预览

本文件是可审查的**离线候选**，不是配置发布请求。用户已授权 #1075 原工作区实现与合成回归；仍不改凭据、不发布配置、不更新 task-processor-unified-20261010、不执行真实采集/AI/付费、不合并。旧价格精度研究及峰闲/完整厂商上界前置已撤回，历史调查仅由 Git 版本追溯，不恢复为当前交付要求。

当前依据：[公共 owner 确认的参考策略与分工](../architecture/enterprise-shared-model-configuration.md#本批文字流程的接口与分工)（#1078 `5cc8248e5d659e3004c2c6686c582528c8fdd0d4`）、[本批接线合同](../architecture/bailian-deepseek-shared-text-admission.md)。原共享选择、CAS/current、Invocation 与资源账本保持唯一；不重复建设公共模型/计费能力。

## 当前 → 候选差异

| 配置项 | 当前可审查信息 | 本批候选 |
| --- | --- | --- |
| 使用企业 | Local Account Center；本轮未重新读取运行数据 | 仍是该企业；实际写入前由原授权 owner fresh 核组织/成员/权限 |
| 文字绑定 | 2026-11-03 已有非秘密快照记录 `local-content-route-830-20261020`，型号 deepseek-v4.1-flash，协议 openai-compatible | 只引用现有组织级绑定，不创建、复制或改写 Key；当次凭据版本/端点摘要需 fresh 检查，不从旧快照推定当前有效 |
| 型号与策略 | 百炼 DASHSCOPE / deepseek-v4.1-flash | 固定非思考 v1，保留原 byte/deadline；不自动换 1017、不提高调用上限 |
| 企业选择 | 同一共享配置位置维护三类选择 | text 选择该绑定；image/multimodal 保留明确选择和各自未就绪提示。整个快照一次 CAS 保存，文字使用不等生图完成 |
| 平台价目 | 原平台参考价/FX owner | 新参考版本，策略 reference-text-totals-v1；CNY 恒等 FX；输入 ¥2/百万 Token、输出 ¥8/百万 Token，不应用缓存优惠。是草案参考值，不是实时准确厂商价 |
| 研究操作 | product.research.agent / product.research.decision | 项目资料分析，保留 market-research-prompt-v1 / market-research-action-v1、原项目/成员权限及只读设置 |
| 选品操作 | product.selection.agent / product.selection.decision | 独立商品对比，保留 product-selection-prompt-v1 / product-selection-action-v1、准确保存商品及独立历史 |
| 操作预算 | 用户已确认每项 ¥0.30 /20点，两项 ¥0.60 /40点 | 数值不变；每项两调用，每次 ≤32000/4000 Token，原更小限制优先。配置预算控制不等于供应商实际账单硬限 |
| 用量与结算 | 原 Invocation/资源微点账本 | 显式基础 P/C/T 校验后按冻结参考价估算；费用投影带 referencePriceVersion。缺缓存细分不阻断，缺基础事实/矛盾/超限仍 UNKNOWN，保留 R 且不重发 |
| BoundEvidenceVersion | 不再索取供应商完整硬界证明 | 不填假值；参考策略资格来自有效策略、配置和准确 profile 身份。其他策略不自动放宽 |

候选的准确 binding reference/price revision 由原 owner 合法发布时生成，本文件不伪造 ID、CAS 或真实 Ready。旧内容用途的 revision/凭据版本不能替代研究与选品各自的定义和新共享选择版本。

## 用户流程与离线验证

入口仍复用智能市场 → 我的智能体 → 本企业启用/配置 → 项目资料分析或独立商品对比。项目业务链为市场调研中选择同一项目和 2–5 项已保存资料 → 智能选品查看比较/费用估算 → 明确确认只读分析 → 人工决定与理由 → 按批次到我的数据核对 → 转入我的供应链。

原 HTTP fixture 已使用 SharedModelService、PlatformPricesService、SharedCatalog、原配置 Repository、受控模型 Adapter、Invocation 与资源账本；只在任务专属 PG 中保存合成配置，provider 仅本机 httptest。研究上游项目/来源为固定合成适配。回归核对完整业务交接、原键重放、没有自动转入、没有额外模型调用/重复结算；界面核对只携带已批准的准确资料，分批分别确认。不能据此宣称真实 SHEIN/Amazon 取数、远端模型或运行实例已经可用。

## 仍需分开处理的实际开放事项

本批离线实现不再被精确价格/峰闲时点/全缓存分项证明阻塞。实际开放仍需另行授权配置发布与指定实例更新，并核对当前新系统安装/运行合同和该企业实时非秘密绑定信息；不升级旧库、不做迁移/兼容、不删除旧数据，不擅自初始化保留卷。真实采集、真实 AI/付费及用户试用均未执行。

本轮准确源码、开发自检、原独立 Reviewer 增量结果和必需 CI 在 PR #1075 维护；实现完成只进入待独立验证/用户试用，不签发产品验收、上线或合并许可。
