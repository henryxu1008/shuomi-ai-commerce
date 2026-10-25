# Amazon 插件：可选采集同族其他变体

Execution: [#955](https://github.com/qq550723504/shuomi-ai-commerce/issues/955)。
Design Basis: **Independent Architecture**。Admission: **IMPLEMENTATION_READY**。
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
