# 多商品优选申请 v1

Design Basis: Independent Architecture。Admission Status: NOT_READY / DESIGN_REVIEW_REQUIRED。

Authority：[产品决定](../product/selected-application-multi-product.md)、Issue #710 当前正文；复用冻结 supply-market-v1、supply-market-choice-eligibility、own-product-optimization-v1。生产代码尚未修改，独立架构检查通过并明确 IMPLEMENTATION_READY 后实施。

## 1. Outcome / Scope

多选合格 own 商品 → 公共信息/逐件差异 → 汇总确认 → 单件申请结果。最大 20 件，跨搜索/页保留选择。公共默认为发货地、售后、通用附件；每件可覆盖/不引用，库存/产能、MOQ、leadDays、priceNote逐件填写。每件应用后的最终供货及 fileIds 必须通过原 marketCommandSchema；同一 item 不得重复、不得静默替换版本。一次明确披露确认绑定最终各件资料。

不新增后端批次 API/表、整批事务/审核/审批/发布、后台调度、第二业务事实 owner、通用草稿/验收平台、迁移或真实环境操作。Legacy decision: N/A，消费的均为当前 owner。

## 2. Owners / contract → implementation → injection → consumer

- 当前 Collection/Catalog/Apply owns 精确商品/资格，SupplyMarket owns 单件声明、申请、审核事件/回执；PrivateFile owner owns 私密 upload receipt/object。现有 authority/事务均不变。
- SelectedApplicationForm 读取 listOwnProducts/readMarketChoice；新增本域小型纯函数冻结最终逐件命令、合并附件、验证选择和 journal。
- 同源 writeMarket/resolveMarket → 现有 BFF → SupplyMarket Service → Repository.Execute；每件仍提交一个 submit_selected，原 principal/actor/key 命令锁、InputHash 和同库事务原样负责唯一性。
- 复用 useResourcePending 的 scope、localStorage、大小检查、不可变 persist/精确 clear，以及 navigator.locks；新增本域 hook 消费不可变单件命令集合。MarketBoundary 仍按 user/org key 重建子树。
- 共享 member pending 槽扩展为显式 single intent 或 selected-batch intent 的受限 union。其它单件消费者遇到 batch 时锁定新写并提供返回本申请页的入口；本页遇到 single 时沿原单件 notice/核实继续处理。不是旧 abstraction 的 fallback/双读；当前单件与当前多件两种意图共用唯一槽/锁，不能同时写。
- Qualifications 复用原上传/核实；公共附件沿原 namespace，商品专属附件使用稳定 item ID namespace，禁止待确认文件被另一个商品编辑器接收。公共/各件 pending upload 都阻止冻结/切换编辑目标；不删除原文件。

## 3. UI 与确认

沿原 Figma 997:764 的多选列表、1004:892 的分区/主题；[产品决定](../product/selected-application-multi-product.md)定义逐件差异、汇总/结果状态。无新路线、菜单或独立 Workbench。

1. 多选：上限 20、明确已选数/跨页保留、可移除；错误或刷新中的当前资格不作为新选择依据。
2. 公共信息 + 逐件供货：继承公共值是显式默认关系，取消继承后复制当前默认供用户修改，公共改动只影响继承项。引用公共附件与专属附件分开，合并去重且每件最多 6 个；服务端仍核总字节/对象。
3. 核对：对所有已选项重新读取 Choice，必须与原 selection 的 item/revision/publication/original/effective version/Apply 完全一致。失效项逐件说明，不能静默接入更新版本；用户明确移除或返回重选。修改任何信息后披露确认清空。最终按钮明确申请件数与“逐件独立审核”，不产生发布。
4. 结果：每件显示商品名、状态、原回执/申请链接、具体错误；明确成功与待确认不能显示同一种“提交失败”。只把明确失败项交回编辑；成功项保持，未决项不生成新键。

## 4. Browser journal / states / persistence

最终明确提交才生成一个 batch key 和每件独立 key，将顺序固定的 {title, single MarketIntent} 数组（1–20件）持久化到原 member pending 槽。每个 intent 必须 admin=false/action=submit_selected，单件命令键与 item ID 均唯一。只存必要标题/来源引用/声明/附件 refs，不存 Product 全文、凭据或对象内容。journal 受严格 Zod/字节字段与总长度 1 MiB 上限；persist 成功回读确认后才允许第一个 POST。

journal 在整批处理期间不可变，不写状态游标或业务事实。每件成功以 owner 的回执为准，当前页面可暂存已观测结果；刷新后重新按原键只读核实，不能从本地数字推定服务器执行。

展示态：尚未处理 → 处理中 → 已提交 | 明确失败 | 结果待确认。这里只是网络/回执展示，非新增审核状态机。实际单件审核仍 SUBMITTED/EVALUATING/SUPPLEMENT_REQUIRED/APPROVED/REJECTED 等原状态；批准不等于发布。

已提交：有效回执必须有 recordId/revision，唯一 intent 的回执不得被错误计为另一件成功；已观察到的成功项不会再次发送。resolveMarket 200 也必须满足单件申请回执要求。

整批每件均为有效回执或本次明确的“不提交”响应、且无未知/尚未处理后，才精确 clear 原 journal，展示结果；明确失败可由用户带回编辑并在下一次明确确认时产生新键。刷新前未完成的 journal 不丢弃，成功回执可由原 owner 再次查询。

## 5. Partial failure / retry / UNKNOWN

逐件串行，只有一件 HTTP 在途。每次发送前核 user/org live context、abort、原 journal/锁及当前项 key/payload；不建立后台队列或浏览器外执行者。

- 原先新提交：原键/载荷冻结、持久化后发送。成功保留回执并进入下一项。
- 明确 INVALID_REQUEST/400、REVISION_CONFLICT/409、NOT_FOUND/404 写响应：按现单件语义确认该次未创建申请，逐件报错并继续其它件；不回滚先前成功。输入需要修改时只编辑明确失败项，待旧批无未决后重新确认。
- 401/403、身份/组织变化：立即停止后续并保留原 journal，不尝试在另一组织/身份继续；恢复正确身份后先原键查询。
- 网络错误/超时、取消、非预期响应/缺失申请回执或其它服务故障：当前项 UNKNOWN，停止后续，保留全部原键/载荷，不生成替代键或后台重发。
- 刷新/重启页面：不自动 POST。用户“核实本批结果”只 GET by-key；404 仅表示未读到回执，不证明未执行，保持未决。不得由查询错误清除原 journal。
- 用户明确“使用原请求继续”时，在同一锁内先 GET 每项原回执，已有有效回执跳过；只有当前原键尚未读到回执才以完全相同键/载荷 POST。原 Repository advisory lock + key/hash/transaction 处理丢响应或重复在途；同键异载荷仍冲突。此按钮并非自动换键重试。
- 页面卸载中止本地等待，服务器可能已提交；不标记“取消成功”。保留 journal，恢复后核实。localStorage 损坏/不可写、锁不可用或 journal 被别页更改均暂停写，不清理浏览器数据掩盖问题。

所有处理/核实/清除 journal 的路径均使用同一 member navigator lock；其它同域单件页受 union notice 锁定，避免两个页面竞争清除。只读查询失败不会被“新申请”绕过。

## 6. Authority / private files / transactions

每件仍经过当前 BFF expected-user/org、Service live authorization、exact Selection/Apply 核实。批次不能把单件可读权限变成授权；企业切换边界由原 MarketBoundary/context guard 和每次发出前 live check 保持，已有 pending 按原准入阻止错误切换。

公共附件引用复用当前 ReadUpload 的 organization_id + actor_id + member_id + id 约束；每条申请仍独立执行 FileVerifier（最多6件、合计60MiB、真实私密对象/hash/type/size）。ReadAttachedFile 仍先核申请访问和附件实际引用。无新公开URL、跨企业文件库或上传授权。原 receipt 没有单次消费标记，故同scope多条申请引用同一已保存附件不需新owner/迁移。

每件 Repository.Execute 仍在自己的同库事务中提交 record/event/command receipt。没有跨件原子性承诺或补偿；第二件失败不会撤销第一件，也不复制审批/发布权限。外部副作用仍只有现有受控上传，命令本身不调用模型或平台发布。

## 7. 风险与验证

当前 Must/Threat Model：跨user/org/member、精确商品版本、错用附件/供货值、重复申请、未知结果误报或换键重发。产品不要求整批原子、后台跑完或未提交草稿持久化；这些不是本轮阻塞理由。

TDD/相关测试：

- 多选/跨页/上限与不合格拒选，资格变化/失败后不能用旧快照；公共继承/逐件覆盖互不污染、附件 union/逐件限额、汇总值准确、修改后重新确认。
- 每件 key/供货值/selection 不同而可共用同scope附件；成功/明确失败混合时仅失败可编辑，已成功不重发。
- 超时/丢响应/刷新/恢复只读查询，查询404保持未决；显式继续仍原键原载荷，已成功跳过；无效回执、身份/组织变化、另页锁、存储失败 fail closed。
- 原自营、补充等单件消费者及待确认回执行为保留；遇到batch明确导向本页，不误解析/清除。
- 附件已有owner的同scope复用和跨scope拒绝复用现有测试/必要局部回归，不搭建新验收平台。
- 必要相关测试、typecheck、改动lint、稳定候选完整PR CI；一次独立高风险架构检查与最终完整交付检查。未改变代码只复用明确既有证据，不重复全量矩阵。

仅隔离模拟/测试数据；不使用保留实例真实上传/提交/审批/发布，不修改其配置、数据库、worker、卷或历史预留。默认本轮新 PR 不合并/部署/关闭Issue；交付可操作源码/构建入口与限制，由用户试用验收。
