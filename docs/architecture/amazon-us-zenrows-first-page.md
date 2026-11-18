# Amazon 美国站 ZenRows 单页试用

Independent Architecture，IMPLEMENTATION_READY。独立 R1 核对及增量复核确认单次发送、授权、接续和部署边界。关联 [#1458](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1458) 和 [PR #1459](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1459)。用户授权接入并准备 35444。真实调用必须等用户确认准确关键词和 credits 上限。

## 用户结果和试用范围

一位指定的企业成员为一份活跃的私有「产品研究」项目执行一笔获准的 Amazon 美国站关键词搜索。系统只向 ZenRows 提交第一页一次，最多保存 5 个有效且不同的商品。成员从同账号「我的数据」的 Amazon 批次向项目添加原件，选择 2–5 件做现有事实比较。只开放一个显式试用入口，普通 Amazon 路径使用原 Playwright。翻页、详情请求、自动重发、AI 和供货决策在范围外。已有研究比较消费者及 [交接合同](amazon-opportunity-handoff-v1.md) 继续适用。

## 调用和权限

- 服务装配接收可选只读私有配置。默认无配置，无试用路由、凭据或出站行为。配置绑定一份人工确认的 UUID、企业/操作者/成员、项目 UUID、规范关键词、账户引用、专用 Key 摘要、批准有效期、供应商用量依据引用及有效期、确认的单次 credits 上限、适用的包内余额和既有 DataRow 价格版本及最大本地费用。真实操作员核对该证据适用于当前账户、固定请求模式及专用 Key，由操作员保管的配置提供冻结输入。凭据由独立文件读入并匹配摘要。服务启动、GET、工作流恢复均无供应商调用；配置值不由 HTTP 输入替换。
- 复用研究入口的 verified identity、企业 live write、项目管理或只读、当前 Collection 权限，以及 Data Services 的数据市场与原 funding 校验。正式占用和实际发送前都核对当前厂商额度依据、项目、成员和由现有 funding resolver 得出的资金入口。通过既有商业价格读取器核对 DataRow 价格版本与 `当前单价 × 5` 的已确认最大本地费用。收到页面后写入原 job 前再次校验活跃可写项目、成员、资金和价格。已交给普通 Worker 的任务按原规则发布至成员私有 Collection，Worker 继续按原 Data Services 权限和资金规则运行。添加到调研项目时由 Research 再校验项目及商品读取权限。只有指定成员和项目可以 Submit 或读该试用回执；请求不同项目、账户或失权均拒绝。
- 同源工作台 BFF 精确放行 `GET /api/workbench/research/projects/{project}/amazon-first-page/{approval}` 与同路径 `POST`，复用当前 session、Origin、真实企业上下文和服务器令牌转发到相同尾部的原生 `/api/v1/workbench/research/projects/…`。传空 body，路径外不接受业务输入。返回最少的状态、原任务 ID、请求 ID 和已确认 credits，原始响应、API Key 和配置文件不外显。正式试跑入口仅使用上述已登录工作台调用。GET 只读取原回执，POST 成功提交及同批准 ID 的成功回执重放仅为原 job 接续原 starter。

## 单次发送与持久化

1. Product Data Acquisition 使用现有仅追加的 `data_service_commands`。锁定批准 UUID 的事务咨询锁，检查同一 UUID 的 `zenrows_amazon_claim` 是否已被任何企业/操作者占用，然后插入；原绑定冲突统一拒绝且不披露其他身份。持久化成员、原项目、资金入口和完整批准摘要。唯一行竞争的获胜者且成功收到提交确认，才允许调用。旧行的成员/摘要不符为冲突；已有占用只查原结果。无租期续发、无重新生成批准 ID、无在启动/GET 时发送。占用后中断或提交不明时保守保留原单，恢复原回执，已失去响应时需账户持有人核对原单。
2. 供应商请求固定 HTTPS 主机、GET Fetch、目标 Amazon US 页码 1。固定 `js_render=true`、`premium_proxy=true`、`proxy_country=us`、`extract=auto`，不加 mode 或变更供应商运行选项，按官方固定组合 25 credits 并核实当前账户适用的单次上界。复用既有 ZenRows HTTP/1 专用新连接、重定向关闭和错误脱敏；请求预算最多 180 秒及限制响应 5 MB，受批准和额度依据有效期约束。用户确认的 credits 必须覆盖此精确模式的已核实单次上界并有包内余额。专用 Key 的供应商窗口额度可辅助限制后续请求，不用作本次请求硬上界。校验供应商请求 ID 和最终 URL 必须对应获准的第一页。若供应商确认为超限或响应不能验证，记录实测/未知状态，停止发布。缺逐笔整数 credits 时保留 UNKNOWN 证据，发布有效页面仅在派发前确认的精确用量上界成立时允许。不会用金额或汇总推算逐笔 credits。 [Fetch 合同](https://docs.zenrows.com/fetch/api-reference) · [Key 窗口额度行为](https://docs.zenrows.com/first-steps/api-key-credit-caps)
3. Decoder 使用 ZenRows 结构化结果的 ASIN、标题与同一响应里原 Amazon 搜索卡的图片。复用 Amazon 商品站点、ASIN 和允许的图片校验，去重、排除原搜索规则的广告，并且不访问任何商品/图片/下一页 URL。记录独立解析版本 `amazon-zenrows-search-v1`。首试匹配原比较的证据合同，缺可核实库存及准确报价时不写可比价格。保存的事实依然是当前 `amazon_data`，不伪造正常详情解析版本或预计销量。
4. 成功收到有效页面时，在 Product 单事务中追加绑定原 claim 的收到回执，并由现有 admission 写普通 Amazon job 及经过验证的 `PREPARED_EVIDENCE` 项，标记发现完成。创建 ID 由原批准派生，输入摘要含批准及已准备证据。事务内要求原 claim 和授权，锁顺序沿原 admission。普通任务或同键异输入禁止覆盖；回执和确切证据要么共同提交，要么仍可查到原 claim 且结果未知。无成功商品不能调用浏览器续采，原回执表示样本不足。供应商拒绝、畸形或超限走不带 job 的终态回执。断网、取消或无结果证据时视为 UNKNOWN，绝不补发。
5. 提交后显式以原 job 调用既有 starter，成功回执的同 ID POST 可以再次确保这一个工作流已启动；该动作无需再次拥有发送资格。即便 starter 响应丢失，仍可保留已保存的回执，原 Data Services 任务详情也会按现有逻辑接续。正常 Worker 在任务已发现时处理 `PREPARED_EVIDENCE`，预留及结算 DataRow，发布到同一 Collection 批次。准备状态缺证据必须明确失败；不允许退回调用普通抓取。下游继续绑定原 Publication、版本、校验摘要和当前实时权限。新的供应商发送和页面发布没有跨系统原子性，因此响应丢失只保全原单，允许这次没有可用样本。

## 落点和验证

Product Data Acquisition 持有单次发送与数据批次事实，现有 DataRow 资源 owner 仍持有本地计量，Collection/Catalog 与 Research 各消费原事实。接线为配置文件 → private trial sender 和 Product Store → 原研究项目 HTTP 入口 → Data Services Worker/Collection/Research。无新表、列、物理数据库、迁移、同一事实的双写或后台轮询。Legacy N/A。

## 35444 准备和试跑

试用镜像准备时省略 `amazonFirstPageInputDir`。收到真实授权后，由部署维护者将专用 Linux 私有目录用只读挂载传入，沿现有试取目录校验，目录归 root 且权限 0700，文件归 root 且权限 0600。目录中的 `approval.json` 是 `{"version":"amazon-zenrows-first-page@v1","approval":{...}}`，字段由 `AmazonFirstPageApproval` 定义。 `key.json` 持有与批准摘要一致的 `binding` 和专用 `key`。证据依据、额度和专用 Key 私下核对。

获准成员从自己的调研项目确认准确关键词、任务范围和预算。沿同源工作台入口提交原批准 ID，提交请求为空。需同一正常登录会话及当前工作台身份/企业一致性标头。在作业不明时，先以 GET 查原批准。 `PREPARED` 表示已向原数据任务交接，可在「数据市场」按关键词查任务并等保存完成； `UNKNOWN` 只保留原单核对。 `credits: null` 表示逐笔用量待核实。任务完成后在「市场调研 → 添加已有商品资料」选择这一批、比较 2–5 件。页面事实不足时保留缺项，不成为发起下一请求的授权。

适用测试复用 #1459 的隔离 PostgreSQL：独立验证同一并发批准最多一次、成功回执和预备证据原子、超时及权限撤销停止、不翻页或加发详情请求、重启后只接续原 job、异项目/旧版本拒绝及 2–5 件可比较。对测试替身确认字段/HTTP 边界。正式 35444 准备仅按原保留实例运行合同更新经验证的必要服务，保留已有配置及数据。无试用写入时可回退原镜像；产生新版原件后，回退候选必须继续识别 `amazon-zenrows-search-v1`，停用试用通过移除运行配置完成。未确认关键词或 credits 前，无真实批准配置、发送、计费和产品验收。
