# Local 非现金 AI 试用：入口与待确认执行清单

依据：[产品决定](../product/local-ai-trial-v1.md)、[已准入设计](../architecture/local-ai-trial-v1.md)、[Issue #724](https://github.com/qq550723504/task-processor/issues/724)。本文件交付可操作入口和一次标题试用清单；源码准入、隔离测试和 CI 不等于已经发点、配置模型或真实调用。最终 HEAD/CI/独立复核及原运行 owner 的准确镜像/二进制/私密配置摘要写主要 PR 和原 #688 交接，不在此滚动重标构件。

当前授权到代码与清单。**原 #688 汇总给用户确认之前，真实发点、成员额度修改、凭据/模型接线、技能发布或企业准入、部署/停启、生成/Apply 和付费调用均不执行。** 未获未知最终 HEAD 合并许可；不提交优选申请。

## 1. 专用原生入口

在原资源 owner 内提供 `cmd/local-ai-trial-grant`；已有 API 镜像打包为 `/app/local-ai-trial-grant`。无 HTTP 赠点接口、启动 seed、自动续发或新表。默认 `inspect` 使用现有库的只读连接，核角色/表权限后只读回执；`grant` 还要求私密配置中明确启用当前批准。

```powershell
go build -o C:\private\local-ai-trial-grant.exe ./cmd/local-ai-trial-grant
# 以下为获准后在正确原实例上的操作示例；当前不执行。
C:\private\local-ai-trial-grant.exe -config C:\private\trial-724.json -operation-id trial-724-local-v1
```

配置是绝对路径、普通私密文件，最多16KiB；Linux mode0600，Windows ACL 仅原用户/管理员/SYSTEM。拒绝重复或未知 JSON 字段、尾随 JSON；不从命令行、环境变量或浏览器接收密码。以下仅是格式说明，密码由原运行 owner 私密引用现有受限账号，不创建这个示例文件或新密码：

```json
{
  "organizationId": "394513833252618242",
  "operatorId": "local-ai-trial-operator",
  "approvalEvidenceId": "01a13f90-861e-79c2-ba35-c3c951cb0af6",
  "grantEnabled": false,
  "database": {
    "host": "<原资源库host>",
    "port": 5433,
    "user": "commercial_owner_runtime",
    "password": "<原受限账号私密引用>",
    "database": "<原资源库准确名称>"
  }
}
```

`operation-id` 由本次操作清单固定并保留，不能用当前时间自动生成重试身份。输出只有非秘密的组织、固定来源 digest、是否找到及不可变回执。不存在时 `found:false`，不伪装发放成功；存在时返回原 operation/event/批准证据、gross/debtRepaid/net 和发放时各余额。

确认后才可把同一私密配置的 `grantEnabled` 开启，并明确执行 `-action grant`。资源固定为一次10,000 ai_point；不能传 amount/resource/source/campaign。现债务优先偿还，净可用点数可以低于10,000，不补第二笔。原 allocated/reserved/consumed、商业订单/钱包、成员月限不被重置或绕过。

同键和异键同来源都返回原回执；新部署、日期、模型或新批准证据不能改变首次试用来源。换证据属于冲突，不是第二次额度。结果未知、输出丢失或中断时，保留原配置/批准/operation，用默认 inspect 核实；关闭 grant 仍可读取原回执。读取失败继续停止，不能换键/来源、直接 SQL 加余额、自动负向撤销、覆盖备份或重跑 initializer。

## 2. 后续唯一执行者与目标

仅原「运行最新程序」负责人操作现有 `https://localhost:35444/workbench` 实例，原 #688 汇总确认。本地独立开发测试不得连接这个业务库。

| 项目 | 固定目标与保留要求 |
| --- | --- |
| 企业/actor/成员 | Local `394513833252618242` / actor `394513839896395778` / member `394513850164051970`；先从真实会话读回，不用显示名判定归属 |
| 原测试自有商品 | `7b9ba0c8-34c6-53e8-88d8-2825dec050c0`，SKU `FLOW-CHECK-20261015-01`；不新建或搬移商品 |
| 生成字段/语言 | 只选标题、zh-CN；图片、描述、属性、SKU、价格、库存均不纳入此次建议 |
| 真实运行范围 | 一次受治理 Run，最多2次模型调用；人工审核、原 Review/Apply。已产生候选或 UNKNOWN 时不再创建另一 Run 重发 |
| 资格与供货页 | Apply 后刷新真实“已优化的自有商品”资格，核 Figma `997:764` / `1004:892`；止于提交前，不点击申请提交 |
| 数据保留 | 原历史、凭据、容量修复、命名卷、Temporal持久任务及原 worker；先备份/基线，只加本次合法新事实 |

## 3. 模型接线与预算清单

仍使用已选 DASHSCOPE / `deepseek-v4.1-flash`，复用本机百炼 `default` 私密凭据。准确北京同业务空间兼容 Base URL 是 `https://llm-6sbnuu3doularkgj.cn-beijing.maas.aliyuncs.com/compatible-mode/v1`；此前 `/api/v1` 是另一协议路径，不把它当成 OpenAI-compatible 地址。原 Key 不发聊天、不进 Git/build context/普通日志。

1. 原 owner 先读取当前私密配置和业务状态，保存备份及摘要，确认与本清单目标一致。A 已完成不需要重装原表/角色；任何新增差异先停下报告，不执行迁移、drop/revoke 或初始化。
2. 执行保持关闭，仅准备准确 Local 的 `productAgent.contentTextPolicies`；独立 client 建议 `local-content-deepseek-trial`，操作固定 `product.content.decision`。先检查所有 capability 的 client 引用，不能覆盖 title/planning/其他能力或别组织凭据。
3. 沿[现有受限凭据入口](skill-capability-v1.md#商品内容模型凭据)，由 `title_credential_writer` 向同 ProductAgentDB 写 `consumer:content`；私密输入 APIStyle/AdapterKind 为 `openai-compatible`。超时60秒，profile deadline45秒。先留凭据准入版本空，取得真实 `credentialVersion` 和 endpoint digest 后准确绑定，不伪造版本/health。
4. 按下表冻结计量版本/上限。点数来源、成员月限、钱包、credential、技能及 Run 各是原 owner 的事实，不宣称跨库整体原子；后步失败保留前步已提交回执，不重复赠额。

| 配置 | 本轮值 |
| --- | --- |
| PromptVersion / OutputSchemaVersion | `product-content-agent-v1` / `product-content-action-v1` |
| PointTariff | 版本 `local-dashscope-deepseek-v4-1-flash-trial-20261015-v1`，输入/输出每百万200/800点 |
| CostPricingVersion / Currency | `dashscope-beijing-20261016-v1` / `CNY`；忙时原价输入/输出每百万¥2/¥8，即 InputMicrosPerMillion=2000000、OutputMicrosPerMillion=8000000 |
| 单调用上限 | MaximumPromptTokens=65536、MaximumCompletionTokens=4096；MaximumInputBytes=131072、MaximumOutputBytes=16384、DeadlineBound=45秒 |
| 原 Run 限制 | steps=12、modelCalls=2、tokens=139264、costMicros=1000000（¥1）、runtimeSeconds=120，继续 `contentTitleOnly:true` |
| 总试用预算 | 累计≤¥100；只批准本次一个 Run，不通过额度允许任意新 Run。启用前核历史成本/UNKNOWN/可用余量，不完整则不派发 |

2026-10-16 已核[阿里云官方北京原价](https://help.aliyun.com/zh/model-studio/deepseek-v4-1-flash)：输入忙时¥2/输出¥8每百万；本清单不假定缓存或闲时优惠。按上述 token 上限推算每调用≤¥0.16384，两调用≤¥0.32768，仍受单Run¥1与总试用¥100限制；实际执行前重新核对价格、版本、usage映射及当前累计。10,000内部非现金点数不代表供应商免费，也不能独自证明总预算已受控。

5. 准确来源确认后仅一次发放，保存原生回执，再从正常资源/用量入口读回。按正常成员月限入口将目标成员设为10,000，使用实际 expected version，保留已消耗/预占；不设置成员分配余额或生成钱包/购买订单。
6. 原「我的智能体」配置入口只启用目标 Local 的商品内容能力；Chat继续 conversationOnly，Store继续 preparationOnly，容量、OIDC/CA、Compose/env和持久任务保持。最终私密配置差异、真实准入版本、如需停启的确切组件/步骤由原 owner 放入确认清单后才实施。

## 4. 个人技能准入与一次标题试用

沿[现有技能使用说明](skill-capability-v1.md)，不新建技能运行时。原运行 owner 在 2026-10-16 10:17:36（UTC+8）以单次 READ ONLY 查询确认已有个人技能：

| 当前事实 | 值 |
| --- | --- |
| 名称 / Skill ID | `local-trial-title-normalization-20261015` / `b7afbaa6-9dd5-5026-9811-72a517b07bfa` |
| owner / 版本 / revision | 原 actor `394513839896395778`，v1 / revision1，headVersion1 |
| digest | `ad162c94fef4c489c8267d82e88dc9875142ad532f4d0fca70eb0bc36c9ba8f8` |
| 发布与共享 | DRAFT / PRIVATE，allowCopy=false、allowExport=false，当前不可用于运行 |
| Local + product.content.agent 准入 | skill_admissions 无行，原合同默认 allowPersonal=false |

这是只读基线，不是已发布/准入回执，执行前核实际 revision。复用原技能，不重建、共享或复制；不选 latest、不把格式通过当已准入。

确认后的操作顺序：

1. **我的账户 → 账户资料 → 我的技能**：核对原个人技能的确切 ID、版本、digest 和仅基于商品事实的正文；仅按清单发布/启用这个版本。个人所有权不转移，已有技能正文不覆盖。
2. **商品内容与本地化配置 → 团队技能**：由有 configure 权限的原成员明确允许该企业使用个人技能，使用当前 revision/原键回执。现有合同是“组织 + Agent 的 allowPersonal 开关”，不是 v1 白名单；开启后其它符合原个人所有权/发布状态的方法也可能准入，本次 Run 另固定上表 v1/digest。此效果必须一并在执行清单确认，不新增白名单体系、不复制/批准其他人的技能。首次采用人工审核模式，不启 auto_approve/full_access。
3. 从原自有商品详情进入“内容完善与本地化”，只选 zh-CN 标题，在原执行确认选择“使用技能/方法”的准确版本。记录技能 ID/归属/版本/digest、商品当前版本、输入摘要和请求 key 后才确认一次生成；将商品已有信息发送供应商属于本次确认内容。
4. 等待原 Run/任务结果，核对实际 route/credential/tariff/usage/cost和技能证据；无使用量或 UNKNOWN 保留原预占/原 Run，由既有 owner 原键读回，不换模型、技能版本、请求 key 再生成。
5. 在原 Review 页核建议只改标题且依据真实商品。仅经人类授权后执行接受/Apply；读回新 Catalog 版本、原版本历史和未改字段，再刷新详情逐字核实。Apply/CAS 冲突不再调用模型，沿原 Review/Catalog 协议处理。
6. 进入优选申请选择页，确认只有已优化的自有商品合格，选择原测试商品；核供货页截图、字段、单位、布局和表单状态与 Figma。原已决定保留单件申请，不扩批量提交。填写检查至提交前，**不发申请提交请求**。

## 5. 停止、恢复与交接

任何身份/企业/商品版本、schema/权限、准入/凭据版本、资费/预算、技能状态或服务健康不符即停，不用 fallback/第二事实源。已提交 grant、限额、凭据、Skill、Run、Review、Catalog 回执分别保留；不将“停用”解释为删除、重领或撤销已消费事实。

默认先停止新生成，按原 owner 合格入口关闭相应执行/准入或该 content 凭据；执行结果未知先读回，不能盲目重复 mutation。停启仅正常 SIGTERM/原队列与worker恢复顺序，保留全部任务，不 KILL/清卷/覆盖数据库。旧应用回退仍沿 A 的全历史 structured/UNKNOWN/漂移条件；存在新结构化事实或不能安全判断时停止，不能退旧 DTO 或用备份覆写。

最终交接分别列清：代码/CI/独立复核、实际配置/发点、真实模型/费用/技能证据、Review/Apply/刷新版本及供货页截图、未执行项和用户待确认项。当前真实 B 结果全部 NOT_RUN，不自行签发用户验收或关闭 Issue。
