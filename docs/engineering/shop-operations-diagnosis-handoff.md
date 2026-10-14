# 店铺经营诊断源码交接

#685 / PR #686，独立设计准入和证据见 [架构依据](../architecture/shop-operations-diagnosis-agent-v1.md)。本批复用 #656 四种权限及 #680 选品的已交接公共能力；PR 以选品分支为显式依赖。源码交付与合并、安装、真实模型执行和用户验收分别记录。

入口：正常 Console 登录、选择企业后，打开 `/workbench/agents/mine/shop.operations-diagnosis.agent`。管理员在原智能体配置中显式启用；四种权限仍由原设置保存。选择一间可访问店铺和 1–366 天的 UTC+8 自然日周期，点“读取经营事实”。当前成员同时需要经营明细、经营建议及原 Store 读取授权；AI 分析另需原 Agent 使用和本地执行权限。

事实读取不调用模型。页面区分人工财务完整/部分覆盖、人民币元显示（源单位为分）、原记录版本与缺项。平台观察单独展示原同步范围、观察时间和完整性，计数不替代本周期销量或收入。SHEIN 只读既有保存观察；TEMU 明确“暂不支持离线读取”，诊断不会进入会查询平台的原 saved-read 分支。人工经营路径仍可使用。

模型由部署者后续在原 serving manifest 的 `productAgent.diagnosisTextPolicies` 配置：仅明确授权企业，独立 `shop.operations-diagnosis-prompt-v1` / `shop.operations-diagnosis-action-v1` profile，原凭据版本/endpoint identity/点数 tariff/总预算准入。不能借用标题、内容、上架资料或选品配置；诊断单独配置即可，其他未配置消费者保持 NEEDS_CONFIGURATION。原组织凭据 owner 保存密钥，manifest 和浏览器不保存密钥。

读取事实后，填写本次诊断目标，点“准备分析”，核对当前模式、单次上限及总预算，再点“确认生成建议”。只会产生有实际事实引用的 AI 建议，不修改经营财务、店铺、商品或平台。数据不足保持不足；没有健康分、预测收益或自动任务。

建议与原事实/配置/模式保存在既有 `product_agent_runs` 和 `agent_configuration` owner。浏览器只保存当前用户/企业的原准备和待确认编号。刷新仅 GET 原结果；响应未知时先“核实原请求”，明确重试仍用原编号。运行中或费用未知不能另起付费诊断。历史已停用或财务后来修改时仍可在当前授权下读取旧建议；店铺归档或撤权则拒绝披露。

正常运行沿用已安装的 current-application 和 Console，无新 worker、数据库表或前端启用变量。现有 serving manifest 需要原 Store Center / operationsCockpit 与 Agent configuration/runtime；这里只增加独立模型 policy。正常启动示例（绝对私密路径按已有配置替换）：

```powershell
go run ./cmd/current-application -config C:\private\current-application.json
# 从 web/listingkit-ui 目录使用原 Console 启动命令
pnpm.cmd dev
```

已有数据、命名 volumes、worker 和配置保持。不要对保留实例运行空库初始化、迁移或删除卷。本次只在已有隔离 fixture 数据库验证；真实 IAM/模型/支付/SHEIN/SDS、保留实例安装与用户试用均 NOT_RUN。准确 HEAD、CI 与原独立最终复核写主要 PR，不把开发自检或 CI 当产品验收。
