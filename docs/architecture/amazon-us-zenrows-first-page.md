# Amazon 美国站 ZenRows 单页试用

Independent Architecture，待独立评审。关联 [#1458](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1458) 和 [PR #1459](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1459)。用户授权接入并准备 35444。真实调用必须等用户确认准确关键词和 credits 上限。

## 用户结果和试用范围

一位指定的企业成员为一份活跃的私有「产品研究」项目执行一笔获准的 Amazon 美国站关键词搜索。系统只向 ZenRows 提交第一页一次，最多保存 5 个有效且不同的商品。成员从同账号「我的数据」的 Amazon 批次向项目添加原件，选择 2–5 件做现有事实比较。只开放一个显式试用入口，普通 Amazon 路径使用原 Playwright。翻页、详情请求、自动重发、AI 和供货决策在范围外。已有研究比较消费者及 [交接合同](amazon-opportunity-handoff-v1.md) 继续适用。

## 调用和权限

- 服务装配接收可选只读私有配置。默认无配置，无试用路由、凭据或出站行为。配置绑定一份人工确认的 UUID、企业/操作者/成员、项目 UUID、规范关键词、有效期、单次供应商上限、供应商账户确认和指定 Data Services 费用版本。凭据由独立文件读入。服务启动、GET、工作流恢复均无供应商调用；配置值不由 HTTP 输入替换。
- 复用研究入口的 verified identity、企业 live write、项目管理或只读、当前 Collection 权限，以及 Data Services 的数据市场与原 funding 校验。每次提交在占用发送资格前和实际发送前检查当前项目、成员和资金入口；发布前复查。服务器使用既有 funding resolver 及 DataRow 计价确认，按原规则对保存的每件商品计量。只有指定成员和项目可以 Submit 或读该试用回执；请求不同项目、账户或失权均拒绝。
- POST 使用同一个批准 ID，GET 只读取。路由消耗者为经原工作台会话绑定的试用操作。传空 body，路径外不接受业务输入。返回最少的状态、原任务/批次 ID、请求 ID 和已确认 credits，原始响应、API Key 和配置文件不外显。

## 单次发送与持久化

1. Product Data Acquisition 使用现有仅追加的 `data_service_commands` 按 (org, actor, approval ID) 插入 `zenrows_amazon_claim`，持久化成员、项目、完整批准摘要和到期时间。唯一行竞争的获胜者且成功收到提交确认，才允许调用。旧行的成员/摘要不符为冲突；已有占用只查原结果。无租期续发、无重新生成批准 ID、无在启动/GET 时发送。占用后中断或提交不明时保守保留原单，恢复原回执，已失去响应时需账户持有人核对原单。
2. 供应商请求固定 HTTPS 主机、GET Fetch、目标 Amazon US 页码 1、固定美国地区和固定手动用量参数，关闭客户端重定向与自动重试，使用专用不复用的连接及有界日期、超时和响应长度。用户确认的 credits 必须覆盖此精确模式的已核实单次上界并有包内余额。专用 Key 的供应商窗口额度可辅助限制后续请求，不用作本次请求硬上界。校验供应商请求 ID 和最终 URL 必须对应获准的第一页。若供应商确认为超限或响应不能验证，记录实测/未知状态，停止发布。缺逐笔整数 credits 时保留 UNKNOWN 证据，发布有效页面仅在派发前确认的精确用量上界成立时允许。不会用金额或汇总推算逐笔 credits。 [Fetch 合同](https://docs.zenrows.com/fetch/api-reference) · [Key 窗口额度行为](https://docs.zenrows.com/first-steps/api-key-credit-caps)
3. Decoder 使用 ZenRows 结构化结果的 ASIN、标题与同一响应里原 Amazon 搜索卡的图片。复用 Amazon 商品站点、ASIN 和允许的图片校验，去重、排除原搜索规则的广告，并且不访问任何商品/图片/下一页 URL。记录独立解析版本 `amazon-zenrows-search-v1`。首试匹配原比较的证据合同，缺可核实库存及准确报价时不写可比价格。保存的事实依然是当前 `amazon_data`，不伪造正常详情解析版本或预计销量。
4. 成功收到有效页面时，在 Product 单事务中追加绑定原 claim 的收到回执，并由现有 admission 写普通 Amazon job 及经过验证的 `PREPARED_EVIDENCE` 项，标记发现完成。创建 ID 由原批准派生，输入摘要含批准及已准备证据。事务内要求原 claim 和授权，锁顺序沿原 admission。普通任务或同键异输入禁止覆盖；回执和确切证据要么共同提交，要么仍可查到原 claim 且结果未知。无成功商品不能调用浏览器续采，原回执表示样本不足。供应商拒绝、畸形或超限走不带 job 的终态回执。断网、取消或无结果证据时视为 UNKNOWN，绝不补发。
5. 正常 Data Services Worker 在既有任务已发现时处理 `PREPARED_EVIDENCE`，预留及结算 DataRow，发布到同一 Collection 批次。结果回执可在服务重启后只读，普通任务的原恢复机制继续本地发布。准备状态缺证据必须明确失败；不允许退回调用普通抓取。下游继续绑定原 Publication、版本、校验摘要和当前实时权限。新的供应商发送和页面发布没有跨系统原子性，因此响应丢失只保全原单，允许这次没有可用样本。

## 落点和验证

Product Data Acquisition 持有单次发送与数据批次事实，现有 DataRow 资源 owner 仍持有本地计量，Collection/Catalog 与 Research 各消费原事实。接线为配置文件 → private trial sender 和 Product Store → 原研究项目 HTTP 入口 → Data Services Worker/Collection/Research。无新表、列、物理数据库、迁移、同一事实的双写或后台轮询。Legacy N/A。

适用测试复用 #1459 的隔离 PostgreSQL：独立验证同一并发批准最多一次、成功回执和预备证据原子、超时及权限撤销停止、不翻页或加发详情请求、重启后只接续原 job、异项目/旧版本拒绝及 2–5 件可比较。对测试替身确认字段/HTTP 边界。正式 35444 准备仅按原保留实例运行合同更新经验证的必要服务，保留已有配置及数据。未确认关键词或 credits 前，无真实批准配置、发送、计费和产品验收。
