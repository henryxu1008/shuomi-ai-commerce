# 硕米智能引擎产品战略

硕米围绕商品获取、内容完善、平台草稿与经营工作组织可检查的业务结果。UI、导航和页面职责以[当前 Figma Authority](final-ui-ia-authority.md)为准；业务智能体分工以[当前职责](business-agent-responsibilities.md)为准。

项目按[全新系统基线](greenfield-no-legacy-migration.md)推进。当前需求、开放范围和排序记录在执行 Issue / PR；本页不维护旧阶段路线图或复制派工队列。

## 产品使命


平台使命：

> **让跨境电商团队能够把选品、商品理解、内容与图片生产、平台适配、校验和上架等重复知识工作交给可审计、可约束、可恢复的 AI Agent，同时保留业务事实、平台规则和高风险操作的确定性控制。**

平台不追求“完全无人值守”。

目标是逐步建立：

```text
AI-first execution
+ deterministic guardrails
+ human approval
+ durable workflow
+ measurable business outcome
```


## 核心原则


### 4.1 Agent 负责不确定性，领域服务负责确定性

适合 Agent 的工作：

- 商品语义理解；
- 类目候选判断；
- 缺失资料诊断；
- 属性候选推断；
- 文案与图片策略选择；
- 根据 validator 结果选择修复动作；
- 根据目标选择下一项只读或受控工具。

不应交给 Agent 自由决定的工作：

- 利润公式；
- 权限判断；
- 租户隔离；
- SKU 唯一性；
- 幂等控制；
- 平台确定性 validation；
- 业务状态机；
- 远端提交状态确认；
- 账务、配额和审计规则。

### 4.2 Tool-first，不复制业务能力

Agent 不重新实现商品、图片、平台和发布逻辑。

正确关系：

```text
Agent
  -> Tool Contract
    -> Existing Domain Service
      -> Provider / Marketplace / Database
```

Agent 只是受控调用方。

### 4.3 单一事实源

- 商品事实：商品领域拥有；
- 平台 Listing 草稿：平台/Listing 领域拥有；
- 用户业务任务：当前 BusinessTask 产品合同；内部执行/恢复由相应领域与 Temporal/RabbitMQ 路径负责，不由旧 ListingKit Task 充当产品事实；
- Agent 单次推理状态：Agent Runtime 拥有；
- 模型调用与成本：AI Invocation Ledger 拥有；
- 身份、租户与授权：ZITADEL 和现有授权边界拥有。

不得因为引入 Agent 建立第二份商品数据库、第二套发布状态机或第二套权限模型。

### 4.4 高风险动作必须显式授权

默认将工具分为：

```text
Read Tool
Compute Tool
Propose Tool
Write Tool
External Side-effect Tool
```

初期 Agent 只允许前三类。

保存草稿、修改价格、上传远端素材、创建平台商品、发布商品等操作必须经过明确策略与人工确认，后续才逐项开放。

### 4.5 每次 Agent 运行必须有界

至少限制：

- 最大 step；
- 最大模型调用次数；
- 最大工具调用次数；
- 最大运行时长；
- 最大估算成本；
- 最大修复循环次数；
- 工具 allowlist；
- provider / model policy。

Agent 不允许无限自循环。

### 4.6 受控开放，不以旧架构兜底

新 Agent 能力通过 feature flag、tenant allowlist 和评测门禁逐步启用；不可用时停止相应能力，不自动回接旧 ListingKit / Task-first 路径。

当前领域中仍有效的确定性流程按合同保留。这不是保留旧 Service、旧工作台或双写/fallback 的许可；Legacy 只有 EXTRACT / RETIRE。


## 用户交互


长期交互不应只存在 Chat 页面；下列三种交互方式必须投影到 Figma 已批准的页面，不另建平行菜单或恢复旧工作台。

### 6.1 Goal-based Agent Workspace

在 Figma「AI工作台」及其业务任务页面中，用户直接表达目标：

```text
“分析这款 1688 商品，生成 SHEIN 美国站资料，
售价保证毛利率至少 25%，先不要发布。”
```

Agent 返回：

- 对目标的结构化理解；
- 执行计划；
- 当前步骤；
- 使用过的证据和工具；
- blocker；
- 需要人工决策的问题；
- 最终候选结果。

### 6.2 Embedded Copilot

Agent 能力嵌入 Figma 对应的供应、店铺商品或其他已批准业务页面，复用当前 Console 与领域 owner；不以已有 ListingKit 页面为产品模板，不保留旧 Listing Workspace。

例如在属性区：

```text
诊断缺失属性
生成候选
解释依据
重新验证
```

在图片区：

```text
诊断图片问题
推荐处理方式
生成候选场景图
按当前产品合同进行程序校验与用户确认
```

具体图像步骤以当前批准流程为准；能力示例不要求每次额外调用模型质量评分，也不恢复已取消的 Extract / Review 步骤。

### 6.3 Batch Agent Jobs

用于批量商品：

```text
100 个来源商品
  -> Agent 批量诊断
  -> 自动处理低风险项
  -> 聚合阻断项
  -> 人工只审核例外
```

这是平台真正形成运营杠杆的入口。


## Agent 与领域服务


### 10.1 普通 Go 服务

用于短时、确定性转换、校验和领域规则。

### 10.2 RabbitMQ

继续承载简单后台任务、事件分发和已有稳定 worker。

### 10.3 Temporal

继续拥有：

- 长业务流程；
- durable waiting；
- 人工审核等待；
- 外部异步任务等待；
- 跨进程恢复；
- 业务级 retry / compensation。

Temporal 可以把 Agent Run 当成一个明确输入输出的 Activity/能力调用。

### 10.4 Agent Runtime

只负责单次推理中的：

- 动态工具选择；
- 有限修复；
- 推理 checkpoint；
- human interrupt；
- Agent trace。

不得让 Temporal 和 Agent Runtime 同时拥有同一层 retry 语义。


## 权限与人工确认


Agent 平台的发布门禁必须比传统工作台更严格。

### 11.1 工具权限等级

建议统一元数据：

```text
risk_level: read | compute | propose | write | external_side_effect
requires_human_approval: bool
idempotency_required: bool
allowed_roles: []
budget_class: low | medium | high
```

### 11.2 人工确认

第一阶段以下动作默认必须人工确认：

- 应用商品字段补丁；
- 修改销售价格；
- 删除/替换关键资产；
- 保存平台草稿；
- 创建远端商品；
- 正式发布；
- 批量写操作。

### 11.3 Agent 永远不能绕过

- tenant scope；
- role authorization；
- deterministic validator；
- publish readiness；
- idempotency；
- budget policy；
- data policy。


## 需求评审


以后新增需求时，依次回答：

1. 这是用户业务目标，还是底层能力？
2. 是否已有领域 owner？
3. 能否用确定性代码解决？
4. 如果需要 AI，是固定 AI capability 还是动态 Agent？
5. 如果是 Agent，需要哪些 Tools？
6. Tool 是否可以只读或 proposal-only？
7. 是否涉及外部副作用？
8. 谁拥有最终业务状态？
9. 如何 eval？
10. 如何安全停用未获准或不可用的能力，而不回接退休路径？

只有确实需要动态工具选择、有限自修复或人工 interrupt 的场景才进入 Agent Runtime。


## 当前依据

产品决定描述用户结果；架构合同描述事实 owner、安全与副作用边界；操作说明描述启用条件和使用入口。现有源码、模型能力图和示例页面不自动证明某功能已在具体实例开放。
