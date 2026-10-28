# 商品采集插件：可恢复本地草稿规划与设计 v1

日期：2026-10-28。状态：IMPLEMENTATION_READY；执行 Issue [#989](https://github.com/qq550723504/shuomi-ai-commerce/issues/989)。用户已明确授权按已审方案开发、测试并提交一个主要 PR；不合并、不更新实例或插件、不重采、不部署。
Design Basis：Independent Architecture（新增插件本地持久化事实和恢复协议）。

## 1. 用户结果与产品边界

使用者在 Amazon、1688 商品页采集后，关闭弹窗、刷新确认页、扩展后台重建或正常重启浏览器，都能找回**已经确认保全**的资料。Amazon 同族采集逐项保全，恢复后由用户选择继续或交接已采部分；保存失败仍可核实原操作。重开页面、重启浏览器均不自动采集、导航子商品或提交。

本设计先按用户“先进行规划设计”的决定完成独立评审，后续开发新授权见第13节。适用现有资料字段、同族归属、规格图片、来源报价和资源限制；不扩充采集字段。前一份同页原键恢复草案仅解决手动重试，不能代替本设计的防丢失要求，其评审结论不自动覆盖本设计。

Must：提交进度前确认本地写入完成；交接和保存不丢弃本地原资料；原键原字节绑定；原身份/企业/成员恢复；UNKNOWN 不误报成功；本地写入失败保留此前检查点并暂停；现有后端仍为已保存商品事实的唯一 owner。

Out of Scope：云同步、跨电脑/跨浏览器配置恢复、后台自动采集/重试、导入任意 JSON、通用草稿平台、通用任务队列、新业务库草稿表、图片二进制下载、账号/令牌/网页 HTML 暂存、生产部署。卸载扩展、删除浏览器配置、用户清除扩展存储、硬盘损坏不在本地恢复保证内。

产品/Figma Authority：复用仓库 `docs/product/final-ui-ia-authority.md` 指定的 Figma 页面 31:463，以及现有 `/capture/amazon`、`/capture/1688` 和商品页插件入口。本轮只提出插件局部“恢复本地草稿”和接收页恢复状态，不新增 Console 顶层菜单。具体新增控件是本次设计建议，不能声称已在 Figma 或已获产品验收。

## 2. 已核实的根因

| 当前路径 | 实际 owner / 限制 | 丢失或卡住的后果 |
| --- | --- | --- |
| Amazon 同族进度 | `FamilyCoordinator` 在进度页内存保存 seed、variants、diagnostics | 进度页刷新、关闭或扩展重载会丢未交接结果 |
| 单商品及交接 | `Controller.payload` 和 `Handoff.pending` 为后台内存；交接有效期 5 分钟 | 后台重建或有效期结束后不能依赖它重取完整资料 |
| 原标签定位 | `ReturnSession` / `FamilyLocation` 使用 `storage.session` | 主要解决同一会话内定位，不是持久草稿；不能靠旧 tabId 恢复跨重启归属 |
| 应用接收 | React 组件内存保存 payload / 冻结 body，sessionStorage 只留摘要与原键标记 | 刷新、关页会丢完整资料；Amazon 首次发送后 UI 只提供原键检查 |
| 业务保存 | Amazon 既有 SRC 锁及三方事务；1688 既有 acquisition operation owner | 已保存商品有原子性/固定版本保护，但不能保全尚未到业务事务的浏览器资料 |

当前 171 项现场的首次请求结果仍未知；原键暂未找到完整记录不等于证明首次请求失败。本方案不能为已经运行的旧 JS 增加入口，也不能自动把旧页内存迁入新草稿。当前页继续保留；该批是否可经受支持的现有入口保全/提交仍未证明。

## 3. 方案选择与唯一 owner

推荐在**原扩展自己的 IndexedDB** 中保存本地草稿，使用浏览器事务；优先采用成熟薄封装（如 `idb`，实施前核实版本和许可证），不自行实现文件日志、事务引擎或备份框架。

选择理由：需要把“规格记录、已保全进度和版本”原子更新；`storage.session` 不符合跨重载/重启需求；`storage.local` 缺少本设计需要的跨记录事务/CAS 接口，继续堆双槽和恢复日志会重复建设。应用自己的持久存储不再另放一份可独立修改的完整草稿，避免双 owner 和同步冲突。

| 职责 | owner |
| --- | --- |
| 未发布来源资料、采集检查点、原发送字节的持久保全 | 扩展 DraftStore；后台服务是唯一写入入口 |
| 页面状态和进度 | 从 DraftStore 派生，页面内存只是投影 |
| 用户/企业/成员与写权限 | 现有应用 context / 后端授权；插件不能把浏览器传来的 scope 声明当服务器身份认证 |
| 原操作结果、固定商品版本与批次引用 | 现有 Amazon / 1688 后端 owner |
| 是否继续采集、确认提交、手动重试和清理本地副本 | 用户明确操作 |

浏览器正常后台终止是预期生命周期，不通过长连接或假心跳保活来解决。恢复时重新读取草稿，后台重启不丢掉已提交的检查点。

为降低 IndexedDB 被浏览器回收的风险，已批准在开发候选增加 `unlimitedStorage` 存储权限，同时保留下述应用自定上限；它不增加网站访问范围或业务权限。本次不更新现有安装或改变其权限。磁盘故障、用户删除等仍不可由该权限解决。

## 4. 最小数据合同

新建本地数据库 `product-capture-drafts-v1`。下列为设计合同，不是本轮执行 schema。

- `drafts`：draftId、schemaVersion、platform、规范来源 URL/ID、mode、createdAt/updatedAt、revision、phase、最后安全错误码、原应用 origin/path、operationKey、绑定 scope、最小回执引用；还必须持久化以下可重建资料，不能只存它们的摘要：单商品完整验证后 payload，或 Amazon root anchor 完整 payload + 完整有序 family manifest（parentASIN、全部 expected members/规格组合）+ manifestDigest + checkpoint cursor/已保全数量 + writer generation/owner 状态。`ready` 后必须保存完整最终来源 payload；`bound` 后再原子绑定完整冻结 body、SHA-256 与 UTF-8 字节数。
- `observations`：draftId + sourceID 唯一键；expected member、经过既有解析/归属校验的规格记录、对应缺失说明和观察时间；只存现有同族投影，不保存所有子页的整套正文/相册。
- 交接 capability 使用既有随机 nonce、短有效期和精确 receiver 绑定；与草稿生命周期分开。过期仅使交接 capability 失效，不删除 payload 或生成新 operationKey。

草稿只包含既有合法商品资料，不含 Cookie、登录凭据、校验 token、购物表单认证字段、完整来源 HTML 或浏览器历史。不使用 storage.sync，不上传草稿，不把 body/nonce/恢复地址写日志。

原始来源 payload 与最终发送 body 的摘要是不同概念。`ready` 前来源资料仍可追加；冻结后不得再改。绑定发送时复用 Amazon 同版本 `freezeAmazonCapture` 的确定性规范化，以及 1688 现有 intent 序列化/校验：应用提供 body，DraftStore 严格解析，并核对由原冻结 payload 经过该平台既有完整受限变换得到的结果，覆盖所有接受字段及顺序，不能只比商品 ID/价格。原 body 的 UTF-8 摘要、字节数与 scope/key 在同一事务持久化。不能把两个不同序列化结果的摘要当作同一摘要，也不新建通用序列化框架。

operationKey 只生成一次并在交接前落盘。绑定 scope 后不能换账号、企业或成员继续该 key；恢复到其他身份只显示不可恢复说明，不暴露被绑定草稿的完整内容。插件本地 scope 用于防误操作，服务器仍必须按原合同授权；同一浏览器配置的本机控制者可访问扩展本地数据属于既有本机信任边界，不宣称本地加密保险箱。

## 5. 持久化与并发不变量

1. 开始采集前先创建草稿记录；创建失败不开始采集。单商品解析成功后先提交 payload，再展示“资料已保全”。
2. Amazon 完整 root anchor、有序 manifest、首项 observation 和初始 cursor/count/generation 先在同一事务中落盘。每个子规格经过现有站点、父子 ASIN、规格与预算校验后，将 observation、诊断、cursor/count 和 revision 在同一事务中提交；收到事务完成后才增加进度并读取下一项。
3. IndexedDB 写事务采用适用浏览器支持的 `durability: strict`，以事务完成为保全边界，不能只等待 `put()` 的请求回调。最小 Edge/Chromium 120 基线的实际支持及正常重启恢复需验证，不把 API 文档替代实测。
4. 后台是唯一写入入口。所有变更携带 expected revision、capture generation；事务内 CAS。后台每次重建先完成恢复初始化：在同一短事务 fence 旧 writer generation，并将未完成采集投影为暂停，再接受其他消息。进度页即使仍开着，旧读取/消息也不能推进 cursor；由用户明确继续才建立新 generation 和已验证归属的 writer。进度页重建同样使旧 generation 失效。该保守恢复可能中断正在读取但尚未保全的一项，已提交项保留；不通过保活绕开生命周期。网络/DOM/摘要计算在事务外完成，短事务只做校验与存储。
5. 一次只运行一笔采集，沿现有单操作限制。首版只保留一个活动采集，不建立通用并发任务调度器。多页面只能观察或竞争同一 generation，只有一个 writer 成功。
6. “页面看到了资料但尚未完成持久化”的最后一项不算已保全；失败或崩溃后允许用户明确重读该项，不能虚报恢复了它。已经提交的项不因单纯恢复被重采。
7. 存储满、写失败、版本冲突或摘要损坏时立即暂停，保留上一成功检查点；不清空旧草稿、不静默降级回内存、不自动删其他资料。

## 6. 状态与正常流程

```text
创建草稿 → 采集并逐项保全 → ready（来源资料冻结）
                       ↘ paused（等待用户继续/仅用已保全资料）
ready → 应用核对并绑定原身份 → bound（原键与发送字节已保全）
bound → 用户确认 → 先保全 sending intent → POST
POST → saved_reported / rejected / unknown
unknown → 用户检查原操作 → 恢复原回执，或继续保留 unknown/未找到
```

- `capturing` / `paused`：已保全观察仍可继续；恢复启动默认显示暂停，不自动读网页。旧 collecting 标记没有当前有效 writer 时投影为暂停。
- `ready`：完整或用户明确选择的部分资料已冻结；这是来源快照，不是“已保存商品”。
- `bound`：原 key、body、digest、scope 已持久化，页面内存丢失也可重新交接这同一发送意图。
- `sending`：先保存 attemptId 和意图再发送；若进程在任何一步丢失，只能视为结果未知，不能根据本地状态判断是否已提交。
- `saved_reported`：应用报告了原后端回执；保存的 publication/operation/version 仅为核对引用。本地页面消息不成为服务器事实，不触发自动删除。
- `rejected`：已有明确失败响应，展示安全错误码和下一动作。保留同一资料；需要改变载荷时不能沿用原 key。
- `unknown`：先通过现有只读原键接口核实，不能凭超时、刷新、一次未找到清空记录或生成新 key。

冻结在最终组装、摘要和预算校验成功后完成，再允许交接。最终完整 payload 成功持久化后，可在同一事务中压缩已被完整表达的中间 observation，避免占用双份空间；不得在最终记录成功前删除检查点。

## 7. 交接、刷新与跨重启恢复

保留既有 `externally_connectable` 精确应用来源、路径、顶层 frame、发送者和随机 nonce 校验；来源网页/content script 不能枚举草稿或读取其他商品 body。扩展内部消息同样验证发送者，不提供任意网页调用的全草稿 API。

应用读取 payload 不消耗草稿；通知 `processing`、`published`、`outcome_unknown` 都不删除资料。提交前应用先让 DraftStore 确认原 body/key/scope 已保全；确认失败则不 POST。应用仍是唯一调用业务保存接口的一方，插件不持有服务器凭据、不绕开用户确认调用业务 API。

**读取顺序是合同的一部分**：先用既有可信应用上下文取得当前 user/org/member，context ready 且无切换/错误后才请求草稿正文。先返回不含商品正文、标题和来源 URL 的最小恢复元数据；已绑定草稿在匹配原 scope 前不得发送正文。应用在发 `capture.read` 前重新核对当前上下文，扩展核对本地绑定、精确接收 tab/top frame/document 与短期 nonce，匹配后才发送 body；应用收到后再核对上下文，等待期间身份/企业/成员变化则丢弃返回值并立即清空旧预览，不展示、不 POST、不改旧草稿绑定。未绑定草稿也等待 context ready 后才读取，绑定前由用户确认归属。插件接受的 scope 仍是可信 receiver 的上下文声明，非服务器认证；业务提交继续服务端授权。当前 receiver 在 context ready 前读取 payload 的旧顺序必须随完整替代流程退休。

接收页刷新后先恢复现有原键标记并检查结果；缺少可用交接 capability 时显示“从插件恢复本地草稿”。用户从商品页的恢复入口、现有快捷键或弹窗打开本地草稿，再签发新短期交接 capability：**同 draftId、原 key、原 body 与原 scope；新 nonce**。不让过期 token 或任意 scope 声明直接换取资料。

原标签尚在且 URL/document/归属均正确时优先复用；不存在、浏览器重启或归属不明时，由用户明确恢复到新的接收标签。旧 tabId/documentId 只作会话内定位，不作跨重启身份；不接管其他会话页面。

Amazon 同族恢复从保全 manifest/cursor 开始。用户点击继续后重新建立明确归属的读取标签，并使用既有身份/规格/校验页检查；父族或规格表变化时暂停并保留旧观察，不覆盖。真人校验继续人工处理，未处理时不自动导航或重试。

现有入口必须同步改变：原“取消本次采集”只终止当前读取、使旧 generation 失效并转 paused，保留所有已提交观察；新文案为“停止采集并保留进度”。它不调用清空持久草稿。原“采集当前商品（新操作）”建立新 draftId/key，先停止活动读取并保留旧草稿，不能覆盖原记录；容量不足拒绝新操作。删除数据只走明确的“清理本地副本”。

## 8. 保存与安全重试

复用原 SRC / Catalog / Collection 保存及固定版本读取，不增加第二业务事实源或迁移历史版本。草稿可恢复并不等于服务器已保存。

Amazon 沿用户已批准的手动同键同 body 重试方向：同一冻结字节摘要、原身份/企业/成员匹配，最近一次可信同锁原键核实未找到且无部分事实错误时，形成一次用户手动重试许可。点击消费许可，复用后端 SRC 锁及两次 LiveWrite 授权；迟到请求无论先后都必须只生成一次事实。404 仍是当时观察，不是原请求已终止证明。重试后 UNKNOWN 不能凭旧许可再发。

1688 保留现有 acquisition operation 的 org+actor scope 与状态语义：恢复先检查原 key/operationId，in-flight 继续待核实，published 读固定版本，failed 按原 owner 的终态处理；不把 Amazon 的无操作表/member 语义套到1688，也不重置其状态。本设计的本地 member 匹配只是当前应用 context 的额外草稿恢复保护，不声称 1688 后端已有同样的 member 绑定，不修改后端业务 owner。恢复资料本身不会自动 POST。

对明确的权限、校验或大小错误保留实际含义；不把已知拒绝统统隐藏为 UNKNOWN。网络超时、非预期响应等仍显示未知。保存失败提示应包含安全错误码、原操作检查入口及“本地资料已保全”的真实状态，不显示内部日志或完整请求。

## 9. 清理、容量与权限

首版建议：一个活动草稿、有限历史本地副本；逻辑存储总额上限 8 MiB，单最终发送 body 仍不超过既有 2 MiB，Amazon 成员/字段等原预算全部保留。总额按实际序列化字节和 metadata 计费，浏览器 quota estimate 仅提前提示，不作为写入成功证明；不得因 `unlimitedStorage` 取消应用上限。

未保存和 UNKNOWN 草稿不按时间自动淘汰。容量不足时拒绝新采集，提示用户清理已核实无用的本地副本；不能为了新商品覆盖旧结果。

首版不因 `published` 页面消息自动清理。应用取得/核实原回执后仍保留本地副本，用户可明确选择“清理本地副本”。发送中不允许清理；UNKNOWN 下须明确提示删除副本不会撤销服务器操作且会失去本地恢复能力。删除只影响本地草稿，不触及已保存版本。实际清理操作继续遵守用户确认要求。

新持久化能力仅面向之后创建的草稿。数据库新建和版本更新不得清空未知或旧现场资料；不将旧运行页当作可迁移数据源，不建立 legacy 兼容 owner。以后本系统 schema 演进另按当前合同处理，损坏/不支持版本保持数据并停止，不自动重建覆盖。

## 10. 页面与主操作

| 位置 / 状态 | 主操作及可理解的信息 |
| --- | --- |
| 商品页有未完成草稿 | “继续上次采集 / 恢复资料”，默认复用；另开新操作须明确选择 |
| 同族采集中 | “已保全 25/171 项”，区分当前读取中但尚未保全的一项；可暂停 |
| 同族暂停 | 继续采集；次要操作“仅使用已保全资料”；“停止采集并保留进度”不删除草稿，关闭页面也不删除已保全进度 |
| 扩展后台/浏览器恢复 | “发现未完成采集，已保全 N 项”；用户选择后才恢复网络动作 |
| 应用确认前 | “资料已保存在本机，请确认账号和企业”；主操作“确认保存” |
| 保存结果未知 | “结果待核实，本地资料仍保留”；主操作“检查原操作” |
| 允许手动重试 | 清楚说明使用同一份资料和原操作；用户点击才发送 |
| 已保存 | 显示固定版本并进入“我的数据”；清理本地副本为次要操作 |

不另建顶层草稿中心或强加新向导；沿原商品页按钮、进度页、应用接收页和弹窗恢复入口完成一条线。技术摘要、generation 和事务细节不进入普通操作文案。

## 11. 实施规划与验证

用户已授权开发和一个主要 PR；同步执行 Issue/当前产品决定与适用架构并达到 IMPLEMENTATION_READY 后，单 Writer 在独立工作区实施。下列是同一交付内的实现/提交边界，不机械拆 PR：

1. DraftStore 与持久化检查点：单写 owner、短事务/CAS、quota 失败处理及真实扩展重载/后台重建恢复。
2. Amazon/1688 单商品与应用交接：冻结资料、key/scope 绑定、刷新恢复和原状态核实；补真实安全错误显示及已批准 Amazon 手动重试。
3. Amazon 同族接入：每项先保全、再显示进度；进度页重建/浏览器重启恢复、父族变化暂停和部分结果交接；完成整条产品路径后集中最终复核。

必要验证覆盖：
- 单项/批次写事务中断，重建后只有完整检查点；已显示保全的数量与磁盘记录一致。
- 关页、正常浏览器重启、后台重建与扩展重载，已保全资料和原 key 不丢；恢复不自动读网页或提交。
- quota/磁盘写失败不覆盖旧记录；损坏摘要/版本冲突拒绝恢复；接近 8 MiB 时验证最终 payload/body 组装所需临时余量与事务顺序，失败仍保留原检查点。
- 交接读取失败、token 过期、应用刷新、登录/企业/成员变化均保留资料且不越权读取或提交。
- 首次 POST 与手动重试各种迟到顺序、响应丢失、撤权、同键异 body，原固定版本和引用至多一次；沿已有隔离 PostgreSQL测试复用，不造新验证平台。
- 页面伪造 published 不删除草稿；清理不影响业务库；超限不自动分批、不改变原 key 载荷。

Legacy decision：EXTRACT + RETIRE。复用当前合法解析、严格 sender 校验、短期交接 capability、原键/摘要和后端事务；当完整替代路径及验证通过后，退休“内存是唯一完整资料 owner”的职责。既有格式与版本只作当前合同，不恢复旧服务端采集或 legacy Service；本次不迁移真实数据。

## 12. 资料与证据边界

已读源码基于离线组合 a19cf48e892264cbe89cd997b59d76f88dcdb0dd；方案需在正式实施前核对届时 main 的实际组合差异。架构敏感范围是新本地持久化与恢复/交接协议，不能沿用“仅同页重试”的独立评审作为本次通过证据。

官方依据：
- https://developer.chrome.com/docs/extensions/develop/concepts/service-workers/lifecycle （后台可能终止；全局变量不能作持久 owner）
- https://developer.chrome.com/docs/extensions/develop/concepts/storage-and-cookies （扩展 IndexedDB、存储共享范围、quota/eviction 与 unlimitedStorage）
- https://developer.chrome.com/docs/extensions/reference/api/storage （session/local 边界）
- https://developer.mozilla.org/en-US/docs/Web/API/IDBDatabase/transaction （事务及 strict durability）

尚未执行任何实现/运行测试，存储权限和新交互仍是本次设计建议。当前 171 项旧页是否可恢复保持 UNKNOWN/未保存，不将设计或未来测试当作这批数据已安全落盘的证明。

独立评审首轮：完整可重建字段合同、正文读取前的正常身份切换隔离、取消/新操作对持久数据的语义三项归 BLOCKER（分别影响核心恢复路径、跨身份资料暴露、已保全资料丢失），已在上述合同补齐。

修订增量独立复核：三项 BLOCKER 已闭合，无新 BLOCKER；本次新持久化/恢复设计的规划阶段评审 PASS。后台重建 fence、迟到旧 writer、正常身份切换、完整来源重建、临界容量及平台差异测试归 IMPLEMENTATION_TEST；尚未运行。

正式实施前：将本轮新产品决定与设计引用同步至执行 Issue/适用架构，按用户后续“按已审方案实现”的决定在开发候选内包含 unlimitedStorage 存储权限与局部恢复交互，满足项目 IMPLEMENTATION_READY 准入。该授权不允许改变现有安装权限、更新实例或重采。

## 13. 执行授权与准备状态

用户在原协调会话 01a16741-448c-78a0-8f3e-fa4d39b17516 明确回复“开始开发并提交 PR”，覆盖独立工作区实现、必要隔离测试、提交、推送和一个主要 PR 维护。独立工作区 codex/capture-durable-drafts 已成功 fetch，核实 origin/main 为 0aab8d5ebe03140e05c2a9c9716d3665da0a0ace；现有采集路径与已审代码无业务差异。执行 Issue #989 已发布并接单；独立设计评审三项 BLOCKER 闭合，无新 BLOCKER，IMPLEMENTATION_TEST 在同一交付收敛。npm 元数据已核实 idb 8.0.4（ISC）和 fake-indexeddb 6.2.6（Apache-2.0），不自行实现事务引擎。局部恢复交互及候选 unlimitedStorage 随已审方案获准；本记录先于正式生产代码提交。已有验证实例、原插件与当前未保存页面保持原样。

## 14. 离线实现与交付边界（2026-10-28）

本节更新第12节的设计阶段状态，不回写历史结论。单 Writer 已实现 IndexedDB DraftStore、逐项检查点、完整 envelope CAS/generation fence、先核身份再读取、冻结 body/key/scope、Amazon 显式原键重试、1688 原 operation 核实及现有入口的恢复接线。使用成熟 idb，不新增后端业务 schema 或第二套保存 owner。

损坏记录逐条隔离并原样保留，不能使其他完整草稿无法恢复；不能通过新写入“修复”或覆盖损坏原件。最终组装 quota 失败保持暂停和原检查点，即使记录暂停原因的写入也失败，进度页仍允许明确恢复/部分交接。同毫秒新操作的排序使用单调 createdAt，避免重建后选错原操作。

独立最终交付检查发现并闭合损坏草稿阻塞完整草稿恢复的 BLOCKER；重建扫描期间旧写入迟到、外层 revision 单独损坏、最终组装容量失败按 IMPLEMENTATION_TEST 补回归。复核基于实际 diff 和调用路径；增量复核未发现新 BLOCKER，不等于用户验收。

已执行：fake-indexeddb 原事务/配额/重建测试，真实 DraftStore 与同族进度页恢复组合测试，receiver 的身份切换/迟到资料/冻结提交/未知结果显式核实测试；UI 类型检查、lint 与生产构建；既有隔离 PostgreSQL 测试和新增原请求/手动重试两种到达顺序、同键异 body、撤权测试。数据库测试只用本任务临时容器和 tmpfs，不访问保留实例。

现有浏览器 fixture 使用独立 Chrome 任务 profile、动态端点、合成页面和无业务后端 receiver。任务代理阻止外网；Chrome 扩展创建的首个读取页可能先于 Playwright 接管，夹具只允许该新建错误页显式进入拦截 fixture，再测暂停/重建/继续，直接首读在这种运行中标为 NOT_RUN_HARNESS_ATTACH_LIMITATION，不能写成真实首读 PASS。后台重建后，若浏览器未提供足够 URL 归属证据，新建接收页并保留旧页符合第7节；这种结果只证明安全回退，不证明原标签复用。

真实 Amazon/1688、保留 Edge、正常浏览器整进程重启、现有插件更新、用户试用及旧171项恢复均 NOT_RUN。开发测试与 CI 只支撑提交评审，不能签发可上线或已验收结论。交付仍不合并、不更新实例/插件、不重采、不部署；最终准确 HEAD、检查结果和运行入口在主要 PR 中维护。
