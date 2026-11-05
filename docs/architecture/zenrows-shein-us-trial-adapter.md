# ZenRows 发送与原单用量查证 · SHEIN 美国站一页试取

## 范围和依据

本独立设计承接 [#1074](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1074)、[单次发送离线合同](shein-us-one-page-trial-send.md)及[商品卡片首试](research-commercial-assessment-v1.md#53-shein-美国站商品卡片首试实施方案)。起点 `9a15e4a6d44a9d77a439635c143432360f248c71`，[#1097 Required CI Gate](https://github.com/henryxu1008/shuomi-ai-commerce/actions/runs/41989193331/job/127120670822) 为 `SUCCESS`。使用者是一个私有研究项目的获权成员。最终业务目标仍为取得一页证据并能通过现有 Collection 和 Research 回看。当前只评审发送回执和原单用量，走查静态输入，无 UI、正式适配器实现、运行装配或来源调用。页内容准备和 `shein_us_data@v1` 发布由现有 Product 路径承接，实际试取启用前须确认这条交接可运行。

保留已批准的固定类目 `1727` 首个列表、按来源顺序取前 20 张、一次客户端提交、并发一、25 credits 与新增现金 USD 0。无新增产品需求、IAM、钱包、调度器、接口或历史迁移。SHEIN 许可、账户适用证明和万邦独立补证并行。

## 核实到的接口与证据

| 证据 | 可用于本试取的结论 |
| --- | --- |
| [Fetch 参考](https://docs.zenrows.com/fetch/api-reference)与[公开 OpenAPI](https://www.zenrows.com/openapi.json) | 同步 Fetch 为 `GET https://api.zenrows.com/v1/`，OpenAPI 全局 `apiKeyQuery` 表示 Fetch 的 `apikey` 查询参数。响应中 `X-Request-Id` 是请求标识，`X-Request-Credits` 是该请求实际扣减整数 credits，`X-Request-Cost` 是美元成本。OpenAPI `info.version=1.0.0`，2026-11-05 核对原件 SHA-256 `c8b4b12902a78ed93bb6d644c7a127d49fd7811dc7520f354056bfa971deb53a`。 |
| [定价](https://docs.zenrows.com/first-steps/pricing)与[额度限额](https://docs.zenrows.com/first-steps/api-key-credit-caps) | 公开 Fetch 档位 1/5/10/25，Extract 随 Fetch，404/410 可扣费。key cap 及 `X-Key-Credits-Remaining` 是限额余额。月包已消耗额度及当次美元折算均不能证明本次有无新增现金支出。 |
| [OpenAPI 路径清单](https://www.zenrows.com/openapi.json)与[API 目录](https://www.zenrows.com/.well-known/api-catalog) | `GET /v1/subscriptions/self/details` 给套餐周期、汇总 `usage_credits`、`credit_limit` 与 key cap 信息，可作为将来预算证据的补充，无法按 Fetch 请求 ID 查账。Batch 的 task history 只针对 Batch 作业。当前公开契约未找到同步 Fetch 按 ID 读取逐笔终态 credits 的端点。 |
| [Activity Log 发布说明](https://www.zenrows.com/changelog)与[取消行为](https://docs.zenrows.com/fetch/api-reference#cancelled-request-behavior) | 公开说明证实仪表板可按产品、URL、时间找请求详情。尚未验证具体账户界面能否同时提供原 ID、key、终态和 credits，也未验证是否可导出或通过受支持接口读取。客户端取消后，ZenRows 可以继续执行。 |

## 适配器与当前 owner

- 放置在 `internal/integration/acquisition/zenrows`，复用 Go `net/http`、现有非秘密账户和凭据引用方式。内部应用层注入现有 `SHEINUSTrialSender` 与 `SHEINUSTrialLookup`，由 `SHEINUSTrialAuthority` 校验当前授权和账户证据。Product Data Acquisition 原批准单、固定绑定、占用事务与用量行继续是唯一事实 owner。适配器不自主占用、补发、提交账务或恢复任务，不修改现有 schema 或账务单位。
- 只有原单占用已明确提交且实时授权/25 credits 与零新增现金证据齐备，现有 domain 才允许调用 sender。固定目标 `https://us.shein.com/Women-Dresses-c-1727.html`，设置 `mode=auto`、`extract=auto`、`proxy_country=us`。拒绝上层替换主机、路径、目标或参数。URL 经标准编码，`apikey` 只在受控传输瞬间注入。拦截、重定向、重试及监控不得泄漏 query、凭据、Cookie 或原始响应。业务日志只记录批准关联、经过校验的请求 ID 和最少状态。
- 专用 HTTP/1.1 transport，禁用连接复用、客户端重定向及请求重试。实际部署前核对出口代理、网关和 SDK 均无重放，仍只允许 `api.zenrows.com:443` 及正常 TLS 校验。无默认环境代理继承，若真实环境要求指定出口，需以同一审批验证该出口。收到 3xx 时不跟随。服务商自动模式内部选择策略的许可独立核实。[Go HTTP 传输](https://pkg.go.dev/net/http#Transport)
- 继承批准期限和现有 30 秒上限。首试响应体建议上限 5,000,000 字节，流式限读到上限加一，关闭 body，拒绝超限、截断或不合法数据。解析业务卡片前只允许已获准的字段，内容不符合市场、目标或格式时不产出卡片。完整可信供应商回执的账务与内容是否合格分别判断。预期真实 transport 验证用本地 TLS 替身计出站次数及重定向/断连行为，测试约束不提供真实环境自动启用路径。

## 回执、超时和原单用量

1. 对固定 TLS Fetch 响应读取唯一、大小写不敏感的 `X-Request-Id`，校验现有 domain 的 `[A-Za-z0-9_-]{1,128}`。只从唯一 `X-Request-Credits` 解码非负十进制整数，范围 0 至 2,147,483,647，与现有 Product 用量列一致。拒绝重复、逗号拼接、非整数、溢出或不明来源。回执仅当唯一有效 ID、响应完整、未超截止且现有授权复核通过，才把 credits 传入原单记账。显式 0 可作为已知结果，HTTP 状态、美元成本、账户或 key 余额差值均不补成 0 或估算用量。确认的 26 等超额实际值原样保存并标超限，禁止截断。
2. 传输超时、正文截断、额度字段缺失、调用方失权或响应迟到，均保留已知原 ID，费用 `UNKNOWN`，维持原 25 credits 逻辑预留。无可确认 ID 则保留现有批准、账号/专用 key、固定 URL 和持久化占用时间。即使无供应商记录也不自动补发、换 key 或释放额度。产品内容校验结果不会覆写明确的账务回执。
3. Lookup 只接受当前获权账户核账方可查的同一原单凭证。已固定 ID 时，只接同一账号的该原 ID。缺 ID 时，需专用 key 的独占发送时窗、同一账号、固定 Fetch URL、可信的提供商接收时间，且原单完整范围内唯一。时间语义需经账户材料核实并考虑时钟偏差，不能用本地超时推断供应商结束，不能用完成时间冒充接收时间。只有原单已终态且实际 credits 明确才交给 domain。只有汇总/截图无关联、记录不可见、时间或 key 无法证实、多个匹配或在途时保持 `UNKNOWN`。
4. 公开资料确认了响应头解析的依据，但没有证实可用的自动原单查询 transport。Lookup 的离线接口可以消费脱敏、可追溯到原单的合成核账证据。真实路径需账户 owner 验证受支持的逐笔凭证（例如能满足字段要求的 Activity Log 原记录或供应商确认）及其可信交接。查不到时保持关闭，不通过仪表板私有接口、Batch 或余额差值补算。本设计不新增录入界面、账务事实源或人工流程。未来如取得正式查询端点，将其绑定到同一原单并验证只读语义后再独立准入。

## 静态走查结果

使用公开请求 ID 形态 `67fa4e35647515d8ad61bb3ee041e1bb`，其他值为合成。UTC 试取窗口 `08:00:00–08:00:30`。模拟已验证的专用 key 和单一批准绑定。对以下 15 组执行了内存合同断言，未发出请求，未执行真实适配器或账户核账。

| 静态输入 | 预期和走查结果 |
| --- | --- |
| H1/H2 完整响应有原 ID 和 credits 25/0 | 分别确认 25/0，保留原 ID。 |
| H3 有原 ID 但只有 `X-Request-Cost: 0.001` 和 key 剩余 0 | 只保留 ID，credits `UNKNOWN`。 |
| H4/H5 已收到 ID 和 25，但超过截止或正文截断 | 固定原 ID，保持 25 预留，按原单查证。 |
| H6/H7 ID 或 credits 有重复头 | ID 重复拒绝固定 ID。只有 credits 重复时保留唯一 ID，费用未知。 |
| H8/H9 明确 26 或给出小数、负数、溢出、拼接值 | 26 实记并标超限。其余均为未知。 |
| L1/L2 已固定 ID，凭证为同一原单 10 credits 或另一 ID | 前者确认 10。后者拒绝替换或补发。 |
| L3/L4 无 ID，专用 key 与接收时窗内一条或两条匹配 | 唯一的终态原单回填 ID 与 10 credits。两条则继续未知。 |
| L5/L6 缺记录、在途、汇总差额、错账户/key/时窗、时间含义错或窗口证明不全 | 均保持 `UNKNOWN` 和原预留，新增发送 0 次。 |

实现准入与实际运行分开。独立评审只判定离线发送回执适配器与原单证据匹配边界。真实 Fetch、真实自动 Lookup、账户数据与 SHEIN 访问均未获准。除许可与账务证据外，实际试取还需固定候选、同批准单离线联测、真实出口禁重放和 Collection/Research 交接证据，再走独立执行审批。
