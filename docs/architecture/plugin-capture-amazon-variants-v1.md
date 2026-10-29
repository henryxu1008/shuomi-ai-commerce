# Amazon 插件：可选采集同族其他变体

> #989 当前恢复增量：[持久草稿与可靠交接](plugin-capture-durable-drafts-v1.md) 替代本文中“进度页内存是唯一事实 owner、重载/重启不可恢复”的历史限制。当前候选逐项提交 IndexedDB，重建后暂停，由用户明确继续；来源发现、严格父族/规格归属、人工校验和原资源限制不变。旧页的未保全资料不作迁移，本设计更新不授权真实重采或更新运行实例。

Execution: [#955](https://github.com/qq550723504/shuomi-ai-commerce/issues/955)。
Design Basis: **Independent Architecture**。Admission: **IMPLEMENTATION_READY**。
此准入结论适用于第1–10节已审范围。第11节后续续采归属修订已另行达到 **IMPLEMENTATION_READY**；
用户后续明确“继续”，当前按本节已审设计实施和隔离测试；旧运行授权不恢复。当前阶段以 #955 正文为准。
第12节占位规格修订已独立达到 **IMPLEMENTATION_READY**，仅授权离线修复；前面的运行授权不恢复。
独立Reviewer `1688_shadow_design_review` 已检查准确设计候选
`c938459b0dae5546a3afccfd09e103d45b6665cc`，无设计BLOCKER；
[评审及四组IMPLEMENTATION_TEST](https://github.com/qq550723504/shuomi-ai-commerce/issues/955#issuecomment-6398925548)
保留为本批次实现要求，设计评审不冒充运行PASS；已审技术设计保持不变。
用户后续明确“继续推进”，授权在同一主要PR内实现、必要测试和提交维护；
仍不授权合并、更新插件/实例、真实采集、业务写入或部署。当前执行阶段以#955正文为准。

## 1. 用户结果与依据

用户于2026-10-25明确要求先完善插件，批准默认关闭的“同时采集其他颜色／尺寸”选项：
识别真实同族变体，逐项读取，显示进度、可取消、异常暂停，最终复用原应用一次确认保存。
已有 Amazon v9、1688 v5及全部版本、数据、配置、加载目录、插件身份保持；
ProductAgent、Chat、服务端采集继续关闭，其他会话/工作区不接管。

依据：[现有接收与原子保存](plugin-capture-ingestion-v1.md)、
[页内入口](plugin-capture-page-entry-v1.md)、[统一入口](plugin-capture-entry-flow-v1.md)、
[session原标签恢复](plugin-capture-session-return-v1.md)、
[来源报价只读](plugin-capture-source-quotes-v1.md)、
[上架字段](../product/plugin-capture-listing-fields-v1.md)、
[当前产品/Figma Authority](../product/final-ui-ia-authority.md)。
Figma文件 `tg48P46SSXl6TBy9lZwg63 / 31:463` 约束当前Console信息架构；
本次扩展局部控件以用户新决定补充，既有应用路由、导航和确认保存页面不改。

**Must**：默认仍采当前规格；只合并有明确父子关系的真实子ASIN；每个价格/规格图绑定自己的子ASIN；
缺失不猜；有限串行采集、取消/异常暂停；不依赖MV3后台长期存活；一次冻结、一次交接，保存与UNKNOWN只归原应用。
**Should**：原商品标签不被自动切换规格；源页入口仍能返回同一进度页或原应用；错误原因和已观察/未观察数量清楚。
**Out of Scope**：跨父商品/跨站采集、颜色×尺码枚举、每个子ASIN的整套相册/独立长正文与全部参数、
真实库存数量/商家SKU推断、后台自动重试、关闭进度页或重启浏览器后的完整任务恢复、通用爬虫/队列/调度器、服务端worker。
不新增理论安全前置；来源网页本身不被当作绝对真实/可上架保证，不承诺同时点报价或完整规格族。

## 2. 根因、现状与只读证据

`amazon/extractor.ts` 当前只读取已选维度，生成一个当前子ASIN，固定 `selected_product / parentASIN:null`。
但extension/TS/Go的v2合同已经接受 `partial_family`、明确父ASIN和多个唯一子ASIN；
Source/Catalog/Collection已有多规格、各自报价和图片，因此不需要另一套保存系统或新表。

2026-10-25只读查看**未刷新的原商品标签** B0BDLGZCTY：
原规格区有明确 `data-asin`；同一twister候选脚本包含 `currentAsin=B0BDLGZCTY`、
`parentAsin=B0D76YXPWZ`、`dimensions=[color_name,size_name]`、
`dimensionToAsinMap` 的171条唯一子ASIN映射，且包含锚定ASIN；颜色19个、尺寸9个。
这是设计输入的DOM观察，**没有发起插件采集、加载其他子ASIN或保存**。
脚本同处一页不是最终关系证明；实现须以静态AST核实同一当前商品初始化对象的绑定，不以关键词命中代替。
证据文件 `amazon-variant-design/public-page-observation.json` 留在本会话证据目录，不含账号/凭据或完整脚本。

另一个真实限制：Controller/Handoff的完整payload在worker内存里，现有session记录只保存返回定位。
将171次加载简单循环放进MV3 worker会再次暴露后台重建丢状态的问题。
[Chrome生命周期](https://developer.chrome.com/docs/extensions/develop/concepts/service-workers/lifecycle)
说明worker可能终止、全局变量会丢失；本设计不使用计时保活、offscreen、音频或持久后台规避生命周期。

## 3. 最小用户流程

1. Amazon首次采集区域显示默认关闭的复选框；已有操作时，复选框放在原“新操作”确认区。
   勾选不改变“返回应用核实”的含义，不覆盖原操作，也不写全局偏好。
2. 本次按钮捕获 `selected | family` 的明确范围并冻结。未勾选、快捷键和普通弹窗的闲置采集仍走现有单规格路径；
   复选框仅作用于本次页内采集按钮，不隐式扩大其他入口的读取范围。
3. family方式打开一个普通**扩展进度标签**，显示“发现N个，已观察M个”、当前阶段、取消入口；
   由它协调一个本次创建、重复使用的非活动商品读取标签。原用户商品标签不导航，不检查其他用户标签。
4. 全部已发现条目读取完成后自动汇总交给原应用；**只有原应用仍有一次确认保存**，不增加第二个保存确认页。
   遇来源验证码/错误/资源上限立即暂停，不继续下一项；可以“继续采集”（用户明确重试当前项）、
   “仅使用已采数据”或“取消本次采集”。没有自动跳过失败项或自动重试。
5. “仅使用已采数据”明确显示遗漏后才冻结交接；未访问项不造规格记录。
   即使N/N完成也只显示“本页发现的N项已观察”，wire仍为 `partial_family`，不宣称完整规格族。
6. family会话进行中，页内、快捷键和弹窗均优先回到该进度标签，不能另开采集/重置；
   交接后统一沿原ReturnSession回到同一应用接收标签，原键查询不重复保存。
7. 取消只丢弃本次未交接的临时事实，不删除旧记录。交接开始后不再提供能误导为撤销保存的“取消”；
   使用原应用核实，关闭进度标签不撤销已交给应用或已保存的结果。

进度页复用现有 `popup.html/js/css` 资源，以严格的 `#amazon-family=<revision>` 模式作为普通标签运行；
不是依赖工具栏弹窗存活。普通popup模式与family模式分支互斥。仍输出原七个安装文件，
不新增包文件清单、工具市场后端或下载路径改造。布局沿现有控件，单一主任务，取消/部分使用清晰区分。

## 4. Owner与接线

| 边界 | 唯一职责/复用 |
| --- | --- |
| source `page-entry` / `page-messages` | 可信点击、严格本次范围、原tab/document/revision校验；不持有整批价格/图片 |
| 新的窄FamilyCoordinator（普通扩展进度页） | native绑定后自己读取原源tab；发现清单、游标、已观察结果、取消/暂停均只在该页内存；原生tabs/scripting串行读取 |
| background入口适配 | 一次开始/最终转交的原子占位、定位进度页、返回/状态投影；不是遍历循环或商品事实owner |
| 现有amazon解析器 | 每个已定位document中读取当前子商品；沿用高清相册、参数、价格/币种及资源过滤 |
| 现有Controller/Handoff | 接收已冻结的合法v2聚合结果，一次创建原UUID/key/nonce和原Amazon receiver；外部read/status合同不扩展 |
| application → Product UoW → SRC/Catalog/Collection | 原身份/企业确认、授权、免费保存、CAS、准确版本、原键与COMMIT UNKNOWN处置 |

计划主要修改extension的page入口、popup模式、background/Controller窄接入与amazon纯解析/协调模块及相关测试；
不修改Go/BFF wire或持久化schema，不新增后端“采变体”API。若现有合同实际不能容纳本Must，先报告并回到设计，不强行塞入其他字段。
当前 `acorn` + `jsonc-parser` 足够处理有界静态字面量；复用已有相册提取的解析技术和纯校验行为，
不执行网页JS、不访问page globals、不实现通用JS解释器或另装抓取框架。
现有 `background.extractCurrent` 的准确tab/document注入与前后复核行为应抽到当前扩展owner的窄reader供两条路径共用；
family reader必须显式传入本次owned tab/document，不能沿普通入口的active-tab查询误读进度页或其他用户标签。
Legacy decision: **N/A**；不调用/包装旧Amazon service，也不重新启用其worker。

## 5. 同族发现与逐项绑定

- 先沿原源页tab/document检查锚定URL、ASIN及选中规格，再从**该商品**初始化对象读取明确父ASIN、
  当前ASIN、维度标签和真实ASIN组合映射。当前ASIN必须在清单中；父ASIN必须合法且不同于锚定子ASIN。
  推荐商品/媒体配置/其他widget中的同名字段不得用来拼接关系。
- 只接受当前站点的规范 `/dp/<ASIN>`，从已验证映射生成规范URL；不跟随页面任意URL、跨站/外部跳转或拼接颜色×尺寸。
  缺父关系、组合歧义、同ASIN多组合或同组合多ASIN时暂停并说明，不能静默退回单规格或伪造父ASIN。
- 清单去重后冻结，锚定商品复用首次解析；其余逐一在**本次创建的唯一读取标签**加载，最多一个在途读取。
  每次有独立document/generation；等待完成后再次核URL/ASIN/同源/主frame与documentId，再注入原解析器，
  返回后再核document/source。旧页结果、跨导航迟到结果、用户改动读取标签或重定向一律不并入。
- 读取到的选择维度必须与该ASIN的已验证组合一致；若子页提供明确父ASIN，必须与原父ASIN一致。
  清单本身的明确关系是成员依据；子页没有重复提供父ASIN不等于可以采任意新成员。
- 数量来自真实映射，不是页面选择器某一个维度的可见选项数。不能仅有颜色选项就宣称全部尺码已覆盖。
  读取时发现的新/变化清单不自动扩队列；标记来源发生变化并暂停，由用户决定新操作。

## 6. 聚合合同与限制

保持 `captureVersion:2 / schemaVersion:2 / amazon-browser-dom/v1` 和原producer。
根 `sourceID/sourceURL` 仍是最初选中的锚定商品，`observation={scope:partial_family,parentASIN:<明确父ASIN>}`；
不能把父ASIN临时冒充已观察商品身份，不把同族清单改写成多个独立业务商品/多次保存。

- 商品标题、描述、参数、共用相册和根 `priceFacts` 来自锚定商品；不把不同子ASIN的报价扁平混进无归属的根报价。
- `variants[]` 按真实子ASIN去重，各自保留已选维度、明确价格/币种及原解析器的规格主图。
  不复制锚定价格/图片来填其他变体；没有观察到就保留缺失，不以ASIN充当独立商家SKU，不把供货状态当库存数量。
- 共用长正文和参数不重复塞进 `variants.attributes`，后者参与组合校验；不借新增事实改变组合identity。
  每子ASIN完整相册/长正文/不同重量等不在首版“规格图/报价”范围内，不声称已经采全。
- 合并时保留每项真实缺失，移除只对单页解析成立的重复“未遍历规格族”提示，改为准确的发现数/已观察数、
  未观察ASIN及暂停原因摘要。语义同类提示有界合并，不能为凑限额隐藏未知报价或伪造字段。
  说明多变体为顺序观察、不是同时点报价；根时间与观察开始/结束范围通过现有缺失/观察说明表达，不发明新wire字段。
- 原限制不放宽：最多256变体/256各类数组、总成员1024、payload 2MiB；每次DOM解析原65536节点/8秒限制保留。
  子页导航等待30秒上限；整批是冻结的有限清单，不定时自动追加/重试。最终聚合重新执行原strict校验和digest。
  下一个结果会超限时不并入、不截断字段，立即暂停，用户可明确仅用此前合法结果。
- 聚合后按原算法重算最终evidence的 `contentSHA256`，再冻结transport digest；不能沿用锚定页或最后一个子页的摘要。
- 冻结之后禁止追加/修改结果。只有用户明确开始新的采集才生成新的payload/key，不能复用同一key保存变化后的整批。

## 7. 浏览器生命周期、消息与恢复

普通进度页是**临时采集**唯一owner。后台休眠/重建不丢该页的事实和游标，也不触发重采；
后台只按原生消息/标签事件定位仍活着的进度页。关闭/刷新/丢弃该页、禁用/重载扩展、浏览器重启后的完整payload恢复不在本版承诺内，
必须显示中断，不能恢复空payload为“成功”或自动重新采集。

仅新增一个受限 `chrome.storage.session` **FamilyLocation定位记录**，使用严格union：

- 公共定位字段：版本、revision、规范锚定URL、原sourceTabId/sourceDocumentId；
- `pending`：尚未绑定进度页，不含伪造tab/document ID；
- `bound`：浏览器确认的进度tab ID/documentId，及可选读取tab ID/documentId/期望规范URL。

实现细节：读取定位可同时保留一个已核实的前驱规范URL（previousURL）。定位写成功但应答丢失、尚未导航时，
同一owned tab/revision/期望目标的明确重试可幂等复用该定位；其它当前/待导航URL仍拒绝。
这只补齐已审导航定位窗口，不保存商品事实、清单、游标或新的自动恢复任务。

限制为TRUSTED_CONTEXTS。只有这些定位元数据，
无商品事实、清单/游标、价格/图片、账号/企业、凭据、操作key/nonce、publication或保存结果。
原ReturnSession schema/key/读写边界不改；这是新增的采集页面定位，不是修改原记录来冒充应用返回位置。
[storage.session](https://developer.chrome.com/docs/extensions/reference/api/storage) 的生命周期也不被宣称为持久任务存储。

启动顺序固定：先沿原互斥/新操作revision占位并成功写入 `pending`，之后才调用 `tabs.create`；
获得精确native进度tab/document并成功写入 `bound` 后，Coordinator才自己读取原sourceTab/document、发现清单并开启遍历。
后台不读取/缓存后再传递初始商品payload，避免绑定前worker重建导致种子事实丢失或补采。
初始化消息只幂等返回本次导航元数据，不创建第二个进度页，也不重复启动Coordinator的读取promise。
所有入口先完成定位初始化；存储失败/坏记录/定位不明均可见uncertain，
不当作空闲。重建后核实精确own-extension进度URL与revision，握手只投影阶段/计数；不能只凭tab存在认定任务成功。
`pending`、绑定失败或此期间worker重建均不视为空闲、不自动再建/补采；只定位已存在的精确上下文或显示初始化中断，
由用户明确取消/新操作后再开始。若旧tabs.create晚到，其旧revision无法绑定或开始读取。
有活跃进度页时不建第二个协调器；页面已关闭只能处理已确认的中断/取消，不隐式重开。

source消息仅增加严格的本次scope/状态投影；返回状态不暴露子页payload。内部family命令只允许浏览器确认的own-extension身份、
精确进度文档URL/revision和本次创建的tab/document绑定；普通popup、任意源页、其他tab及外部receiver不得提交最终聚合或发起导航。
进度页sender.documentId使用原生
[runtime.getContexts](https://developer.chrome.com/docs/extensions/reference/api/runtime#method-getContexts)
核对当前扩展TAB上下文的documentId、tabId、frame和精确URL；该API在现有最低Chrome120范围内可用。
sender.tab若存在还需一致，不依赖其必定存在；无法确认来源就拒绝，不信任正文自报tab ID。
原标签同URL刷新也因documentId变化被识别为中断，不能把新的空进度页当旧会话继续。
不新增外部采集/任意URL/scripting命令，原 `capture.read/status`、origin/path/frame/key/nonce与租户检查不变。
manifest仍为 `activeTab,scripting,storage`、原六个源host，CSP/connect-src、接收地址、扩展身份不改。
读取采用原生 [scripting](https://developer.chrome.com/docs/extensions/reference/api/scripting) 的host/document约束；不新增fetch通道。

| 边界/事件 | 结果与唯一责任方 |
| --- | --- |
| 原生点击重复/快捷键/弹窗并发 | background先占位；回同一个进度页，不追加队列/新采集 |
| pending写入/创建/绑定期间worker重建 | 不当空闲、不再建tab或补采；bound成立前无商品读取，仅定位或明确中断 |
| 页内checkbox或源document在开始前变化 | revision/document验证失败；明确重新选择，不沿用旧意图 |
| 子页错误/超时/验证码/冲突 | Coordinator暂停，保留该页内已观察事实；无自动重试、无自动保存 |
| 用户继续/仅使用已采/取消 | 继续只重试明确失败项；部分使用冻结合法子集；取消不调用业务保存 |
| 读取返回时已取消/换页/新generation | 丢弃迟到结果；不会回写或交接已取消会话 |
| worker重建，进度页仍活着 | 只定位/握手并回该页；采集owner不变，不重建空批次 |
| 进度页关闭，尚未交接 | 明确中断；仅在ID与期望URL仍匹配时关闭本次owned读取tab；不碰其他用户tab/数据 |
| 最终转交开始、响应丢失或worker重建 | 不重发聚合或创建第二个接收页；查询原ReturnSession/原进度页，结果不明则原操作核实 |
| 应用保存响应丢失/COMMIT UNKNOWN | 完全沿现有应用原键GET与SRC回执核实；插件不重POST、不重采、不更换key |
| 交接后关闭进度页 | 不清除原应用定位/业务状态，不撤回保存；读取tab已在交接前安全结束 |

在最终转交前先原子锁定Coordinator（禁止取消/继续/追加），然后背景入口以revision和精确sender占位，
调用原Controller的一次capture/handoff；原ReturnSession pending先于应用tab创建，bound先于返回成功。
转交一经尝试，未知状态必须回原操作核实；不能把“后台没收到完成应答”当作未交接而重新发。
定位清理不能重置正在交接的Controller或原ReturnSession；无法确认本次临时tab归属时留给用户，不批量清理。

## 8. 实现验证与交付停止点

实现按TDD执行，沿原测试工具，不建runner/验收平台：

1. 当前源页真实形状fixture：同一初始化对象的父/当前/组合绑定；混入推荐或视频parent、缺关系、重复组合、
   错站/错ASIN、并不存在的颜色尺码组合被拒；171项真实清单的有限解析。
2. 原逐页解析器复用：每个子ASIN价格/币种/规格图独立，币种缺失不复制；相同颜色不同尺寸/不同颜色图不串。
3. Coordinator与三个入口：串行、取消/继续/部分使用、旧document迟到、重复点击/worker重建/关闭协调页，
   最终转交各竞态只触发一次；安装七文件及权限/CSP清单不变。
4. 有界聚合：256/1024/2MiB原界限，超限暂停、有限缺失摘要、不静默截断；冻结后不能变更payload或复用key。
5. 复用原partial_family TS/Go与保存回归：一个产品版本、每子ASIN规格图/报价保留、原键/原标签恢复；
   不重复运行未改变的全局矩阵，不以fixture代替真实站点与用户验收。

设计阶段只完成文档检查和独立架构评审，当时的代码及fixture为NOT_RUN。后续用户已授权实现，
实现与必要隔离测试继续同一主要PR #957，执行证据绑定实际代码记录在PR；真实多变体采集、
实例/原插件更新、业务保存和部署仍为NOT_RUN / NOT_AUTHORIZED，不以fixture冒充用户验收。
最终独立Reviewer检查实际差异、完整用户路径和四组IMPLEMENTATION_TEST；修复只复核相关增量。
不拆机械设计/实现PR；不合并或关闭Issue。

## 9. 校验暂停与人工继续的局部补齐（2026-10-26）

Design Basis: **Reuse Existing Architecture**；本节复用第5、7节已审的目标绑定、暂停/明确继续及唯一交接，
不改变状态机、持久化事实、权限、恢复协议或保存合同。用户已授权先评审再开发一个主要PR；
本节已由原独立Reviewer `1688_shadow_design_review` 对设计提交 `8dd2d8223` 完成评审：
**PASS / IMPLEMENTATION_READY**，无设计BLOCKER；页面分类顺序、安全错误投影与同reader/目标门禁
归IMPLEMENTATION_TEST，在本主要PR收敛。执行阶段以#955当前正文为准。

用户场景：同族采集停在1/171，保留读取页已证实为`/checkpoint/continue`真人校验，
readyState complete但无productTitle。此前错误码被两处统一为CAPTURE_REJECTED，不能重建原失败时序；
没有资源触限或正常商品字段解析缺陷证据。Must是暂停原因准确且能沿原操作继续，不承诺所有171项均可访问。
产品依据仍为本文件第1节的用户决定与当前Figma Authority；仅补原进度页提示，不增加页面或操作步骤。

### 最小调用路径及不变量

1. 原进度页拥有一个reader及当前待采目标。`tabs.onUpdated(status=complete)`后读取native tab当前状态；
   忽略旧about:blank的完成事件，确认目标页或同站校验/登录页已完成后再沿原ISOLATED注入。
   等待最多仍30秒；不靠增加延时猜页面就绪，不自动刷新/重试。
2. 原injected入口在商品解析前作有界页面分类：仅现有Amazon精确host和HTTPS；
   明确`/checkpoint/continue`、`/errors/validateCaptcha`或明确真人校验控件→AMAZON_VERIFICATION_REQUIRED；
   `/ap/signin`或明确登录表单→AMAZON_SIGN_IN_REQUIRED；尚在loading或商品标题未就绪→PAGE_NOT_READY；
   已完成的非商品文档仍拒绝，不能从推荐区猜商品。只查看固定路径/少量明确节点，不扫描正文、不执行网页JS。
3. runCapture→extractTab→Coordinator→现有entryMessages只传闭合安全码白名单。
   保留既有CAPTURE_TOO_LARGE/CAPTURE_TIMEOUT、UNSUPPORTED_PAGE、SOURCE_CHANGED/PAGE_CHANGED等本地确定性错误；
   未知异常、任意字符串/对象均降为CAPTURE_REJECTED。familyInfo同根因错误也用安全投影。
   原生主frame/documentId须先核实；错误结果也核对仍是同一native document后才能传播。
   不转发异常文本、HTML、账号、密码、cookie、校验查询参数或token，不增加日志/诊断持久化。
4. 校验/登录沿原paused状态，保留游标、已观察事实和本次reader，提示用户通过原
   “查看需处理的商品页”处理后再明确“继续采集”。不点击校验、不自动登录、不绕过或自动恢复。
5. 明确继续只处理原失败项。读取标签仍在校验/登录时再次显示对应原因，不重导航、不前进。
   用户已回到正确待采ASIN时，核对原reader ID/opener、同源、规范目标及pendingUrl；
   family.reader仅接受经现有amazonPageSource规范化后等于**已登记当前子商品**的URL（如跟踪query），
   以支持同一目标重试及读完后串行推进；待导航URL的query准入仍仅允许同一目标重试。
   不允许任意Amazon页、其他ASIN或校验URL成为定位记录。随后复用当前就绪文档解析，不再一律tabs.update。
   每次仍沿原native documentId注入、后复核以及Coordinator的ASIN/父关系/维度验证，换页或取消的迟到结果丢弃。
   尚在加载的原目标只在原30秒内等待；目标不符/用户导航其它页面拒绝，不擅自改回商品页。
6. 下一规格仍沿原串行导航；一次冻结、一次handoff、原应用一次保存和COMMIT UNKNOWN原键核实均不变。
   取消/仅用已采、worker重建仅定位、关闭进度页中断及保守reader清理规则不变。

Owner保持：family-page管理native标签读取；injected/amazon解析器做页面分类与解析；
capture-tab核原生结果和document；Coordinator只保留原暂停/继续/聚合；原应用仍唯一保存owner。
不新增browser状态/schema、reader命令、持久队列、服务端worker或第二事实源。
原30秒、65536节点/8秒、256数组/1024合计/2MiB限制、七安装文件、权限/host/CSP全部保持。
Legacy decision: N/A；不调用旧服务端或新增抓取框架，复用原解析/交接/保存及Vitest/静态fixture。

### 实现证据要求与停止点

先RED后GREEN：固定checkpoint/signin/正常商品/未就绪fixture；同源识别及未知错误脱敏；
错误主frame/document替换；旧about:blank complete不提前解析；30秒超时/取消清理；
同reader人工处理后继续不再导航，仍校验时不重试、不交接；错误ASIN/外站/并发导航继续拒绝；
报价/规格图归属、一次交接及原保存合同回归复用现有测试。
仅跑受影响回归与既有必需检查，不建设runner或新验收平台。
真实页面重采/保存、原插件或实例更新、部署与合并均NOT_AUTHORIZED，本批开发只交一个主要PR供复核。

## 10. 首个读取标签直接导航（2026-10-26）

Design Basis: **Independent Architecture（既有文档的有界修订）**。
本节修改首次导航与定位登记的顺序和首次准入，须独立评审后达到 IMPLEMENTATION_READY 才能改生产代码；
本节已由原独立Reviewer `1688_shadow_design_review` 评审准确设计提交
`80da8b72d6d97a815926b03de9dc8ee935c5f527`：**PASS / IMPLEMENTATION_READY**，无设计BLOCKER。
首次创建唯一性、校验页登记丢应答幂等、严格归属及取消/晚到结果归 IMPLEMENTATION_TEST，在本主要PR收敛。
第1节产品/Figma依据、第5/7/9节未冲突的不变量继续适用；
本节替代依赖 `about:blank` 可读URL的首次登记前提，不改变后续导航、保存或完整任务恢复范围。

### 用户结果与根因

用户批准：读取标签直接打开首个已校验子商品，保留严格归属校验及校验页恢复；
先评审、补回归、实现与必要测试，只交一个主要PR。不合并、不更新实例/插件、不重采、不部署。
当前阻碍是根商品1/171后出现空白读取页和FAMILY_INTERRUPTED。
在现有无 `tabs` 权限、仅精确host grants下，`Tab.url` 可缺省或提交前为空；
首次 `family.reader` 要求它等于 `about:blank`，而真正的 `tabs.update` 在该登记之后。
未修改的page/bridge/coordinator离线复现URL缺省/为空时导航0次、child解析0次，
与现场提示一致；现场原生Tab响应未留存，具体现场分支仍属高可信推断，不能当作已捕获原生调用轨迹。
[Tabs API](https://developer.chrome.com/docs/extensions/reference/api/tabs#type-Tab) 是可选元数据的依据。

根本问题是将URL可读性当作首次归属证明。不能放宽为“空URL都是本次空白页”，
也不能恢复旧代码中两个undefined偶然相等的放行。维持原权限、六站host与七安装文件。

### 首次调用顺序和唯一owner

1. 原进度页仍是唯一临时事实/游标owner。沿原 `family.init` 核实native进度context后，
   从已冻结、校验的同族成员生成当前站点规范 `/dp/<ASIN>`；只创建一个非活动reader，
   `tabs.create` 直接带该目标、原进度 `openerTabId` 和 `windowId`。原商品标签不导航。
2. 页面立即记住返回的原生reader ID和期望目标，首次定位尚未确认仅为该页内存标记。
   无ID/创建拒绝立即暂停，不猜ID、不自动再建标签。后台重建不拥有或重启此创建动作。
3. 登记前沿现有 `waitForFamilyDocument` 做有界首加载等待，不调用 `tabs.update`。
   首加载的URL缺省/空/about:blank只是等待元数据，绝非归属放行；已知错误ASIN/外站/冲突pending立即拒绝。
   每次检查原生opener及窗口，保持原30秒deadline、取消/关闭监听及清理。
   只有当前目标或同站明确校验/登录文档已完成才能离开等待；旧空白complete不能授权解析。
4. 随后复用 `family.reader` 和 FamilyLocation。后台继续校验精确sender/context/revision、
   原生reader ID/opener/窗口、规范目标与同源。首次登记仅接受当前/待提交的同一目标，
   或该目标同站的明确校验/登录路径；后者只保存**期望规范商品URL**，从不保存校验URL/query/token。
   当前校验文档与pending存在冲突、其他ASIN、外站或不明URL均不放行。
   已有reader时必须仍是同一个native ID；previousURL及同一目标重试保持原严格规则。
5. 首次登记成功后直接沿现有capture-tab解析，不再第二次导航。若是校验/登录，
   原页面分类、安全码、Coordinator paused及“查看需处理的商品页/继续采集”接管；不自动跳过。
   未确认首次定位时，超时/后台响应丢失后的明确继续只等待并核实同一个reader/目标，
   即使后台已经写成功也幂等复用，不重建或重新导航。校验仍在则保持原安全提示；
   人工返回正确ASIN后沿同一document与规格校验继续，错误目标仍拒绝。
6. 后续child保持原先“目标定位写成功→原单reader导航→等待→原生document/ASIN/父关系/维度校验”。
   原每项报价/规格图、有限聚合、一次冻结/交接、原应用保存及原键/原标签恢复均不变。

Owner/接线：family-page管理首次创建及内存定位确认，family-reader负责原生加载等待，
FamilyBridge/FamilyLocation核验并记录session导航定位，现有injected/capture-tab分类及解析，
Coordinator只保留原phase，Controller/Handoff与原应用仍为唯一交接/保存路径。
没有新持久化字段、消息类型、状态机、跨系统事实owner或retry worker；原session schema、ReturnSession、
业务wire/schema/事务、租户权限、key/digest及COMMIT UNKNOWN owner保持。
Legacy decision: **N/A**；只复用当前owner与现有Vitest/JSDOM，无旧service包装或新抓取框架。

### 失败、取消与资源边界

| 窗口 | 结果 |
| --- | --- |
| 创建后元数据未出现、首加载超时 | 有界暂停，保留同一ID/目标；明确继续不创建或重复导航 |
| 首次即跳校验/登录 | 记录期望商品定位后由原分类暂停；人工处理后只继续原规格 |
| 定位写失败/应答丢失/worker重建 | 不解析或转交未确认结果；明确继续幂等核实同一reader，失败可见 |
| 首次定位确认前取消/进度关闭 | 沿原命令处理；后台无已证实reader记录时不猜删，可能保留一个未登记临时页 |
| 定位确认后取消/进度关闭 | 复用原保守清理；只有ID/opener/窗口/允许地址仍匹配才关闭，用户导航的页保留 |
| 已取消的首加载/登记晚到 | 不解析、不转交；不追加事实或恢复Coordinator；沿原串行取消清理 |

保留原30秒导航、65536节点/8秒解析、256数组/1024合计/2MiB限制。
不增加延时猜测、不刷新、不自动重试、不读取隐藏账号/表单或校验token。
初始网络导航早于session reader登记是本次顺序调整；只有该页已创建的native ID能继续，
登记失败不触发第二个标签或第二次导航。无法核实归属时保留临时页符合第7节既有保守清理边界。

### 验证与停止点

先补RED：URL缺省/空/延迟pending、首次目标地址与native归属、旧空白complete、
首次即校验/登录及人工处理后继续、错误opener/窗口/ASIN/外站、登记应答丢失或worker重建、
超时/取消/晚到结果；断言首个 `tabs.create` 带规范子商品URL且首个 `tabs.update(url)` 为0。
复用现有完整page→bridge→reader/capture→Coordinator回归，覆盖后续单reader复用、报价/图片归属与一次handoff。

## 11. 已登记读取标签的续采归属（2026-10-26）

Design Basis: **Independent Architecture（既有恢复协议的局部修订）**。
Admission: **IMPLEMENTATION_READY**。独立Reviewer `1688_shadow_design_review` 对准确设计
`cb76b6ff5ab00d5a080ed50958d001361d29bc50` 结论PASS，无设计BLOCKER。
本节先交付设计和独立评审；用户后续明确“继续”，授权最小实现、必要测试及一个主要PR。
不授权重采、部分交接、保存、插件/实例更新、合并或部署。已有20/171临时结果保持原页暂停，不尝试热修复或刷新原页。

### 用户结果、依据与根因

使用者是已在原进度页暂停、处理网络或来源校验后点击“继续采集”的用户。
可见结果：原进度页仍存活、同一已登记reader和待采商品仍可核实时，继续当前失败项；
保留已有观察结果，不重新读取已完成项、不创建替代标签、不自动继续。
产品/Figma依据沿用第1节及用户2026-10-26批准的最小修复方向；现有进度页、按钮和原应用确认保存不变。
本节不增加Console导航、采集字段或人工前置步骤。

本轮只读现场：进度20/171；原reader仍为同一native tab，和原进度页同一窗口；
当前B0C5D5G2Z8商品文档complete、无pending导航，但原生 `openerTabId` 字段缺失。
`family-page.read` 在登记、等待和解析前要求该字段始终等于进度tab ID，因此每次明确继续都立即
`FAMILY_INTERRUPTED`。未追溯到清除关系的具体浏览器动作，不把网络断开直接写成已证实的清除触发者。
原生只读证据、未修改生产代码的离线门禁探针和216表摘要保留在本会话
`local-verification-962/native-ownership-after-network.json`、`network-recovery-offline-gate.json`、
`network-resume-analysis.json`；分析未新增业务保存，旧Amazon v1–v9及1688 v1–v5保留。

Chrome的opener是可选标签组织关系；Chromium会在部分正常导航/标签选择中清除它，
Tabs API再由该关系派生 `openerTabId`，不能把它永久当作已登记reader的身份凭证。
依据：[Tabs API](https://developer.chrome.com/docs/extensions/reference/api/tabs)、
[TabStrip opener清除](https://chromium.googlesource.com/chromium/src/+/refs/heads/main/chrome/browser/ui/tabs/tab_strip_model.cc)、
[原生Tab字段投影](https://chromium.googlesource.com/chromium/src/+/refs/heads/main/chrome/browser/extensions/extension_tab_util.cc)。
这是既有持续归属条件与浏览器语义冲突，不通过强写opener、降低首次准入或另一套采集系统绕开。

### Owner、合同与最小调用路径

临时清单/游标/事实仍只归存活的FamilyCoordinator进度文档；FamilyBridge/FamilyLocation只归现有
session定位记录；原生标签与文档归浏览器；解析、Controller/Handoff和应用保存owner沿第4/7节。
不新增表、存储字段、定位key、权限、host、公共wire或内部命令；不把payload/游标写入session。
只在原 `family.reader` 确认路径和既有等待函数内表达两种准入阶段，不增加通用任务/恢复框架。

调用路径：原进度文档明确继续 → 校验原revision/context及已有reader登记 → 原生ID/窗口/地址门禁
→ 同一待采项的有界等待/原document精确解析 → 原Coordinator校验和聚合 → 原一次交接及应用保存。

| 阶段 | 必须成立的证据 | opener处理 |
| --- | --- | --- |
| 未登记reader的首次准入 | 原 `tabs.create` 返回的唯一ID；原进度document/revision/frame；创建窗口；第10节允许的首目标或同源校验地址 | 必须明确等于原进度tab；缺失或异主都拒绝 |
| 已登记reader续采/下一项/幂等重试 | 原session记录中同一reader ID；原存活进度document/revision；reader与该原生context同窗口，且进度页内初始窗口约束成立；严格current/pending地址转换 | 缺失允许；存在但不是原进度tab仍拒绝 |
| 查看需处理的商品页 | 同一已登记reader、同一进度身份/窗口，当前/待导航仍为本次目标或其同源校验地址 | 同上；不得借此激活无关/已被用户导航走的标签 |
| 取消/结束时清理 | 继续沿现有ID/opener/窗口/允许地址的保守删除条件 | 缺失或异主不扩大删除；保留标签交给用户 |

“已登记”必须由原FamilyLocation记录证明，不得仅由页面自报、local布尔、同URL或两个缺省值相等推定。
首次写入成功但应答丢失、或worker重建后，复用原 `family.reader` 幂等确认重新核实已有记录；
只有确认同一reader/进度/目标后才进入已登记分支。不重新创建、不重新绑定另一reader，
没有可读记录或仍属首次时维持首次严格准入。已有命令应答可携带本次已核实定位确认，
不能把正文自报的定位当作事实。后台sender/context严格校验不变。

地址门禁沿第7/9/10节，必须区分当前失败项的明确重试与下一个已发现子ASIN的受控导航：

- 进入新项前，原reader当前地址仍须是已登记的规范商品；定位更新后只允许原受控前驱地址，
  复用 `previousURL` 处理写入成功但应答丢失的窗口，不能接受任意同站商品。
- 本次目标只来自已验证同族清单。current/pending只接受该步骤允许的规范目标、受控前驱或同源已识别校验页；
  错ASIN、外站、冲突pending、无可核实URL继续拒绝，不能仅凭 `status=complete` 读取。
- 等待期间窗口变化、显式异主、关闭、错误目标仍中断；只有已登记reader的opener从本次ID变为缺失允许继续等待。
  不把检查前后的两个缺省opener值相等当作首次归属证明。窗口/目标/取消在等待和解析前后仍复核。
- 校验/登录只沿已有安全码暂停；人工回到本次目标后必须明确继续。不会自动点击校验、登录、刷新或重试。
  仍使用准确document注入及前后复核，ASIN、父关系、维度和每项价图绑定验证不变。

本节替代第9节人工继续及第10节“每次核对opener”中**已登记后**的永久相等要求；
首次创建/登记准入、未变化的URL限制与所有保存边界继续有效。持续缺失与显式异主不可混为同一条件。

### 失败、状态、资源与副作用

不增加phase或自动恢复触发器；paused → 明确继续 → collecting仍由存活Coordinator负责。
检查失败继续paused，复用现有闭合错误码；不将通用 `FAMILY_INTERRUPTED` 当作可自动跳过/重采许可。
网络错误/超时的提示仍沿现有分类，此次不建设诊断平台，不输出账号、Cookie、校验query/token或商品payload。

| 事件 | 结果 / owner |
| --- | --- |
| 已登记reader在同窗口正确目标，opener缺失 | 明确继续时复用同一reader，处理失败项；已有结果/计数不清空 |
| opener在等待中正常缺失 | 其余归属/目标条件成立则继续原30秒等待；不增加导航或新deadline |
| 明确异主、换ID/窗口、原进度文档刷新或关闭、错目标 | 继续中断，不接管替代标签或重建批次 |
| worker重建或登记应答丢失 | 复用同session记录及原sender/context校验，幂等确认；失败留paused，不把未知当空闲 |
| storage失败/坏记录 | 原uncertain/不可读取处理，不退回只凭页面内存放行 |
| 取消、解析或定位晚到 | 原AbortSignal及串行处理；晚到不追加结果、不交接、不恢复已取消状态 |
| 交接应答丢失 | 原uncertain/原应用核实；不重采、不重交接、不重复保存 |
| 清理时opener缺失 | 不删除该标签；不为方便清理重设opener |

原串行并发、256项上限、原每项30秒deadline、聚合大小和取消监听清理不变。
本次仅修浏览器只读采集的归属门禁；不改变租户/账号、免费保存授权、CAS、原键/digest、COMMIT UNKNOWN，
不新增provider/AI或不可重复业务写入。进度页关闭/刷新、扩展重载、浏览器重启后完整任务恢复仍在范围外。
Legacy decision: **N/A**；只复用当前插件owner，不调用/包装旧服务端采集或启动其worker。

### 回归与交付停止点

以后获开发授权时，先用原Vitest/JSDOM fixture确认RED，再最小实现GREEN；不新建runner/验收平台：

1. 原page→bridge→reader完整调用中，成功观察数项后暂停；删除原生opener，再明确继续，
   同一reader/游标继续当前失败项，已观察项不重读，下一项仍用原reader，各自价图和一次handoff保留。
2. 等待期间删除opener、同源校验人工返回后缺失opener、登记成功但应答丢失及worker重建后幂等确认。
3. 首次无opener、显式异主、替换reader、窗口变化、错进度document/revision、错ASIN/外站/冲突pending及坏定位仍拒绝；
   同根因覆盖继续、等待、查看读取页；清理缺失opener保留原标签。
4. 复用取消/超时/晚到结果、资源上限、原单规格/1688路径与保存合同既有有效证据；只补本次受影响组合回归。

独立评审确认三组 **IMPLEMENTATION_TEST**，以后实施必须收敛，未完成不能交付本修复：

- 定位已写入但应答丢失、页面仍标记未定位、opener缺失且worker重建时，须先由原记录确认已有登记，
  不能被本地首次门禁提前永久拒绝；不得因当前请求新写入记录而自举成已登记。
- 无记录、坏记录及页面自报不能放行；显式异主、错ID/窗口/当前或pending目标继续拒绝。
- 等待与查看共用已确认归属；查看只激活原允许标签，不推进定位、不发起商品导航；清理不扩大删除。

独立架构评审只审此新增恢复边界及相邻调用，不重开未变化的整体架构。
设计PASS只表示可按此依据实施，不授权开发或实机操作，也不等于产品验收。
当前20项只在原进度页内存；本修复将来替换/重载插件不会被宣称能恢复这批事实。
是否部分交接/保存当前20项、取消或另开验证，仍由用户决定；本轮不执行。
安装manifest及默认单规格/1688不变由现有检查覆盖；必要typecheck/lint/fixture build及PR CI不替代真实页面验收。
设计评审及最终独立交付检查复用同一Reviewer，新增finding先分类；本批真实采集/保存/产品验收 NOT_RUN。

## 12. 同一子页的明确规格数据与占位控件（2026-10-26）

Design Basis: **Independent Architecture（现有逐项规格接纳规则的局部修订）**。
Admission: **IMPLEMENTATION_READY**。独立Reviewer `1688_shadow_design_review` 对准确设计
`b2e7fabf0da951a68d4e7b2d5e3745bf4804d8ce` 结论PASS，无设计BLOCKER。
用户在44/171暂停后明确要求“要处理暂停的问题”；本轮按设计评审→回归→最小实现
形成一个主要PR，保留原44项暂停，不部分交接/保存、不重采、不更新原插件/实例、不部署。
涉及现有接纳规则明确“读取到的选择维度”的含义，不以局部bugfix免除独立准入。

### 用户结果、依据和真实证据

使用者是选择采集同族颜色/尺寸、当前商品能明确识别但规格控件暂未选中的用户。
可见结果：同一子页已明确绑定当前ASIN的静态规格数据可补齐控件未观察的维度，继续原有严格同族校验；
真正的ASIN、父关系或维度冲突仍暂停，不用占位文字生成规格，也不根据原清单猜选项。
本节沿第1节用户/Figma Authority，原页内入口、进度页、按钮和确认保存页面不变；
不新增人工操作步骤、Console导航、采集范围或新的UI流程。

#963实机已验证：同一reader的opener缺失后，从29项推进到38项。
随后44/171在B0D4DGJ47V以 `SOURCE_CHANGED` 暂停。原页和子页均为父ASIN B0D76YXPWZ，
171项映射、维度顺序、值、标签、展示数据逐项一致，当前子ASIN明确对应
`Color=Graige`、`Size=52.0"l X 41.0"w X 8.0"th`。
但子页渲染控件持续显示 `Select Color / Select Size`；现有extractor把它们当作真实属性，
Coordinator的严格比较因此拒绝。两页都有 `deletedLandingAsinInfo`，不能单凭该字段宣称商品删除或跳过。
证据：本会话 `local-verification-963/source-changed-44-readonly-analysis.json`；没有重新触发采集以获取本证据。

只在DOM完成后等更久不足以解决当前稳定占位控件；本轮不新增轮询/重载/自动重试。
既有 `discoverFamily` 已能用Acorn静态AST核实当前ASIN、父ASIN、标签/索引/展示值和唯一组合，
应复用它在**这个子页**取得的结果，而非另造脚本解析器、抓取框架或套用根清单。

### Owner、路径和数据规则

Owner不变：Amazon extractor负责渲染事实；`discoverFamily`负责当前文档内的绑定静态数据；
`extractFamilyTab`负责同一document的采集/元数据组合；Coordinator负责与原清单逐项比对、聚合和唯一交接；
Controller/Handoff和应用保留原保存/UNKNOWN owner。

调用路径：原生tab/目标定位 → 准确document提取 → 原同一document的 `familyInfo/discoverFamily`
→ 校验当前ASIN的明确维度 → 原前后document复核 → 原Coordinator接纳 → 原一次聚合交接/应用保存。
不新增脚本注入入口或消息命令，不访问page globals，不执行来源JS，不改变session定位/schema或外部wire。

1. **渲染占位值不是商品属性。** 只在已有选中维度节点内识别本次真实观察到的英文占位串
   `Select Color`、`Select Size`、`Please Select`（仅大小写和空白归一），不拿选项列表/标题/推荐区判定当前选择。
   默认单规格路径遗漏这些属性并保留原missingFacts提示，不因此扫描同族清单或新增网络访问。
   其他语言或未观察的模板不在本修复中猜测识别。
2. **补齐只发生在原family读取路径。** 已由 `familyInfo` 返回的当前文档manifest必须与采集结果的规范sourceURL
   一致；唯一成员ASIN须等于这个子页的sourceID/variant sourceID。仍以该文档 `currentAsin` 绑定对象为依据，
   拒绝另一个ASIN初始化对象、根页manifest、重复/不完整/冲突映射及不明确父关系。
   无有效当前文档manifest时不补齐，沿原缺失/暂停行为，不退回原清单或猜值。
3. **真实渲染值优先并约束静态数据。** 已观察的真实属性名称和值必须与当前成员一一一致；
   真实值冲突、重复维度冲突、额外/不明维度继续 `SOURCE_CHANGED`。
   不能因为extractor已省略冲突值，就把该维度视为普通缺失再补入；现有冲突诊断必须保留并参与拒绝。
4. 满足以上前提时，只补齐缺失/占位的维度，保留该子ASIN自己的价格/币种/图片；
   不改sku、库存、报价、相册或其他字段，不替换已经观察到的真实不同值。
   两个页面同族清单相同不单独构成补齐授权；当前文档、ASIN和真实值一致性缺一不可。
5. 补齐后仍执行原Coordinator的站点、当前ASIN、父ASIN、完整成员清单和精确组合检查，
   不降低这些门禁，不自动跳过失败项，不放宽为“缺失都接受”。
6. 用既有missingFacts保留“渲染选择未观察、维度来自当前ASIN静态数据”的准确来源诊断，
   按对应子ASIN投影到聚合诊断；不宣称这些值来自已选中控件。
   沿原校验和WebCrypto摘要模式重新计算受改动capture的contentSHA256，最终聚合仍沿原摘要/一次交接。

### 边界、失败与副作用

本节不新增phase、持久化事实、游标/清单存储、后台自动恢复、provider/AI或业务写入。
manifest来自本来就执行的同一document读取，不增加第二次全页解析或额外抓取；
复用原2MiB、AST节点/深度/时间上限、256项上限、单reader串行和30秒deadline，不以补齐之名放宽资源限制。
未勾同族的默认采集不调用新增的同族组合逻辑；1688、报价和保存边界不变。

| 事件/输入 | 必须结果 |
| --- | --- |
| 占位/缺失 + 同一document、同ASIN的有效完整映射 | 仅补缺项，保留来源诊断，再交原Coordinator核验 |
| 一部分真实值 + 其余占位，真实值与映射一致 | 保留真实值，仅补缺项 |
| 真实值或已记录的冲突与映射不一致 | 暂停SOURCE_CHANGED，不覆盖、不跳过 |
| 只有根清单、另一个ASIN对象、无当前成员或无manifest | 不补齐，不把未知当成功 |
| 父关系或子页完整清单变化 | 原门禁暂停，不因当前项可补而放行 |
| 同URL不同document、取消或解析晚到 | 原document复核及AbortSignal拒绝，不追加结果/交接 |
| `deletedLandingAsinInfo` 存在 | 不把它当作删除、补齐或跳过的充分依据 |
| 摘要/校验/聚合大小失败 | 原失败/上限暂停，不转交不完整或旧摘要载荷 |

登录/租户、免费保存、CAS、key/digest、COMMIT UNKNOWN、原键/原标签恢复不变。
Legacy decision: **N/A**；仅复用当前插件owner和Acorn解析，不包装或启动旧服务端采集。
不新增理论安全框架，不把网页静态数据当作商家保证；源数据本身的可信范围沿第1节。

### 验证和交付停止点

准入后先用现有Vitest/JSDOM回归确认RED，再最小实现：

- 真实形状：当前子ASIN、两占位控件、同文档绑定171式映射；补齐后可通过原严格逐项检查，
  使用自己的价图、保留来源诊断、摘要匹配，一次handoff。默认单规格只省略占位并提示缺失。
- 部分真实值、缺失/占位、当前成员唯一性；root/错ASIN对象、无manifest、重复/冲突、
  真值冲突、父或清单变化仍拒绝，不能借缺失名义绕过真实冲突。
- 实际page→bridge→capture/manifest→Coordinator链路，含同reader复用、取消/晚到、opener缺失继续和原资源边界；
  复用已有有效单规格/1688/保存合同证据，只验证本轮受影响组合，不建runner或验收平台。

独立评审仅检查新增字段接纳规则及相邻路径，达到IMPLEMENTATION_READY后冻结；
三组IMPLEMENTATION_TEST在实施中收敛：省略过的真实冲突及额外维度仍拒绝；当前文档/ASIN唯一绑定；
摘要与可折叠来源诊断、异步摘要后的document/取消复核。缺/错/root manifest不得补齐，默认路径不触发发现。
具体实现边界测试归IMPLEMENTATION_TEST，未满足Must不得交付。本轮一个主要PR，不拆设计/实现PR。
实现和CI只证明该候选边界；真实页面验证待用户另行授权更新后进行。
原44项只在当前进度页内存：源码修改不会让该页热更新，重新加载仍会丢弃未交接事实。
本轮保持其暂停，不擅自部分保存、重载、重采，也不承诺跨重载恢复；其处置由用户以后决定。
