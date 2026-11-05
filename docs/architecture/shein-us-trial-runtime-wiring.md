# SHEIN 美国站单页试取运行接线

[#1073](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1073) / [#1075](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1075)。基线 `645eaa6cb24d4fde529e3547a2f0d6a6dbc1a3ea`。Design Basis 为 Independent Architecture，状态为待独立评审。范围承接[首试产品决定](research-commercial-assessment-v1.md#53-shein-美国站商品卡片首试实施方案)、[取数入口](shein-us-trial-entry.md)、[单次发送](shein-us-one-page-trial-send.md)、[运行依赖](shein-us-trial-installation.md)和[原单凭证合同](zenrows-shein-us-trial-adapter.md)。

一名获权成员使用一份批准单，把 SHEIN 美国站类目 `1727` 第一页的可核实商品卡关联到自己的私有研究项目。最多一次提交、并发一、按原顺序检查前 20 张、上限 25 credits 和新增现金 USD 0。继续复用现有 Collection 和 Research 的关联与回读界面。范围限设计及合成验证。两项 AI 操作各 ¥0.30 / 20 点，执行关闭。

## 私有运行输入

增加一个可选的绝对目录引用 `sheinUSTrialInputDir`。缺省时按现有方式启动。首版只支持统一项目使用的 Linux 容器运行时。未来获准装配时，由现有部署维护者把本次批准专用的目录只读挂载给服务。企业批准人确认试取与成员范围，ZenRows 账户持有人核验账户、凭据、余额和原请求证据。部署维护者依据这些原件更新本次运行输入。凭证编号仅用于追溯原件，发布者身份与目录控制构成信任边界。

| 固定文件 | 读取时机与接受条件 |
| --- | --- |
| `approval.json` | 启动时读取一次。版本 `shein-us-trial-approval@v1`。含批准 ID、项目 UUID、企业／操作者／成员、同企业核账者、资金来源、账户引用、专用 key 引用和批准截止。复用 `SHEINUSTrialApproval.Binding()` 绑定原批准及固定请求。进程重启使用同一批准与绑定时，Product 账本仍识别原单。变更批准需重新部署并走相应准入。 |
| `gate.json` | `access` 和 `budget` 每次检查时读取。版本 `shein-us-trial-gate@v1`。同一快照包含批准绑定、当前允许动作、授权证据引用、核对时刻和授权截止。供发送的额度块另含账户证据引用、额度截止、可用余额、单次最高 credits、零新增现金及专用 key 独占窗口。发送预算检查同时要求当期快照允许 Send，余额至少 25，单次最高为 1 至 25，零新增现金及独占窗口已证实，额度截止覆盖实际发送期限。每次 View、Send、Publish 和 Reconcile 分别核对当前授权及相应动作。额度证明到期后，已占用原单仍可依据有效的 View 或 Reconcile 授权查看或核账。 |
| `key.json` | 仅已有发送权且 Product 占用已落库后，由发送器解析。版本 `shein-us-trial-key@v1`。含批准绑定、账户引用、key 引用及秘密值。用 `NewDedicatedKeyResolver` 核对调用及文件中的引用和批准截止。完成凭据读取后，在同一次 `gate.json` 快照中复查 Send 和全部发送预算条件。两项有效期均须覆盖本次实际发送截止（含调用方期限），且上下文仍有效。通过后直接交给原发送器，期间没有额外外部读取。门禁读到的快照是本次最后许可判定，之后的文件替换影响下一次判定。末次检查失败时本次发送为零，原已占用单按现有 `UNKNOWN` 规则保留 25 credits，不能再发送。批准及专用 key 在同一发送窗口内固定，账户持有人控制 key 的独占使用及撤销。启动、查询、核账和重放都无需读取秘密。 |
| `usage.json` | `readUsage` 在授权核账时按需读取。版本 `shein-us-trial-usage@v1`。含批准绑定、核对时刻、原件引用及有限条原请求记录。已知原请求 ID 时按同一账户、原 ID、最终状态和整数 credits 核实。原件可来自账户持有人核实的 ZenRows Activity Log 详情或供应商逐笔确认。Key 或目标 URL 出现在原件时须与批准相符。账户汇总额度仅用于预算。缺少已核实逐笔证据时沿用原单 `UNKNOWN` 和预留。 |

文件仅接受固定名称、明确版本和严格 JSON，拒绝重复字段、大小写变体、未知字段、超限和失效内容。授权、账户和用量原件分别以本次交接的规范 UUID 引用，供应商原请求 ID 单独保留。批准及门禁文件分别限 8 KiB，凭据限 4 KiB，用量证据限 64 KiB、最多 32 条。启动时逐段打开容器内绝对路径并禁止跟随链接，校验各级 UID 0 且无组或其他用户写权，固定 UID 0、0700 的只读挂载目录描述符。按固定名称从该描述符打开文件，禁止跟随链接和阻塞在非普通文件上，对同一个打开对象验证普通文件、UID 0、0600、单硬链接和尺寸，再读取至上限。部署准入须另在宿主核实源目录及其上级的写权限、UID 映射与挂载规则，只有受信任的系统维护身份可替换输入或其目录，目标挂载保持只读且业务进程无重新挂载权限。发布者在同一个目录中写好符合条件的临时文件，再原子替换门禁或用量文件。读者每次只消费完整快照。正式装配前在目标只读挂载上验证文件替换可见性和权限。输出与日志只保留现有公开回执字段与最少状态。

## 原单、权限与部署边界

- `currentapplication` 负责准备上述文件读取端口，`cmd/current-application` 在显式选择并通过启动验证时调用现有 `httpapi` 试取装配。注入固定批准及同企业核账者、`access`、`budget`、`readKey` 和 `readUsage`，复用 `NewFixedSHEINUSTrialAuthority`、`NewDedicatedKeyResolver`、`NewFetch` 和 `NewLookup`。继续校验 Market Research、Collection、Data Services 和 Product 账本列权限。默认无试取路由，实际启用的候选仍沿原应用启动边界。试取装配独立于普通实时采集和 AI 开关。启动阶段不发送请求。
- 企业及成员身份由现有 HTTP 绑定提供。提交和发布沿原项目写权、实时数据市场和 Collection 管理权限、资金授权及当前来源许可检查。读取沿原项目和资料权限。核账限批准所定的同企业核账者，沿实时数据市场和账户授权，拿到证据后再次校验再记账。即使来源许可撤回，仍可按有效账户授权处理原账。文件许可与应用权限须同时成立。
- Product Data Acquisition 继续拥有唯一的占用、请求 ID、逐笔 credits 和发布观察事实。重复提交、启动后重读和本地发布恢复沿同一批准单。授权变更在每次现有复查时读取当前快照。已发出的原请求仍按账本和逐笔证据核实。收到明确 0 credits 原样确认，收到超限值原样记账并按现有规则停止发布。供应商返回 HTTP 状态或 USD 成本不足以确认用量。[Fetch API](https://docs.zenrows.com/fetch/api-reference) · [ZenRows OpenAPI](https://www.zenrows.com/openapi.json)
- 原 ID 丢失时，`NewLookup` 需要专用 key、固定 URL、原占用与实际发送截止（含调用方期限）、完整接收窗口及唯一终态。当前 Product 账本记录原占用，实际发送截止尚无本阶段可验证的记录来源。首版私有文件接线只接已知 ID 的逐笔证据。丢失 ID 或证据不完整时保持 `UNKNOWN` 和 25 credits 预留，发送次数保持原值。ZenRows 公开说明可在 Activity Log 查看请求详情。真实账户上的逐笔字段和终态证据须由账户持有人核实。[ZenRows Activity Log 发布说明](https://www.zenrows.com/changelog)

## 合成验证与后续准入

2026-11-05 在基线 `645eaa6cb` 上以 Go test overlay 验证临时私有目录、隔离 Product PostgreSQL、既有研究路由夹具和合成发送端。已知原 ID 和缺失原 ID 两组均通过。覆盖启动阶段延后读取凭据、无门禁和重复 JSON 字段拒绝、24 credits 及过短额度截止拒绝、专用引用与账户不匹配、重复提交只发送一次、错单或多条回执保留预留、撤权拒绝、补齐同一原单的 0 credits 后释放预留。核账时移除额度块仍可按有效账户授权处理原单。原 ID 缺失时保留预留且发送次数保持一次。实际 HTTP transport 复用同基线已通过的本地 TLS 合成测试。合成读取使用临时目录与 Windows 私有 ACL，目标部署挂载的行为仍待验证。

正式接线前完成独立 Architecture Review，验证目标运行目录的私有挂载与刷新语义。首次真实执行还需核实书面许可、实际项目及人选、专用凭据和额度证据、逐笔凭证可得性、实际出口与单次请求授权。合成通过与真实执行分别记录。
