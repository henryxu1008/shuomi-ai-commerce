# #1070 Amazon 美国站调研机会与选品交接 V1

**Independent Architecture · 待独立评审 · NOT_READY**

## 用户结果与依据

验收企业 A 的调研人员在同一个研究项目里，从 Amazon 美国站关键词或类目的一批已保存样本挑选可比较的证据，写下一个商品方向并明确确认。随后进入智能选品，按该方向人工核对已收藏的 1688 货源、试算，再决定是否只批准并转入一件。首试的机会来自人工判断，界面标为「人工整理」。本轮范围到选品交接，下游沿用现有操作；自动机会发现、自动找货、持久化交接、跨人接续和商品发布均在范围外。

- 本设计依据用户授权的 #1070 最小离线实施范围。W1–W4 清单将用户指定的场景和检查项落实为合成验证用例，清单位于 `C:/Users/Henry/code/task-processor/docs/engineering/issue1070-first-trial-walkthrough.md`，SHA-256 `7C867FD36B246968E4BECD920F367E0928151FB4B911D53CAB331D9AD7C32318`，文件摘要用于固定技术验证输入。市场调研和智能选品两个智能体的完整业务验收标准仍待用户确认，实际业务验收仍待用户或指定验证者执行。页面复用当前市场调研与智能选品的项目页，IA 依据固定源码中的 `docs/product/final-ui-ia-authority.md`。
- 设计源码固定为 `7e2c00961bb05f28d19bf5a0fb1fbb28187c2a12`。`e2cab17d40a7fd664853cb2106065d21c2983703` 是原链路的合成验收基线。当前源码把最后一步标为「转入待上传」，目标仍由原 Preparation 回执定位。新交接的运行验证为 `NOT_RUN`。

## 从查询到交接

1. **选定本次研究样本。** 在项目内明确 Amazon 美国站、查询方式和规范化查询值（关键词或类目节点）、本次样本范围。去已有 Data Services 查询入口，离线验收使用隔离环境的合成采集结果与原 owner 产生的任务、批次、发布引用。用户回到项目后明确选定匹配的一笔已保存任务及其 Collection 批次，使用现有添加资料入口关联原件。按来源任务核对站点、查询方式、值与批次；选中另一笔同词任务仍算换批。展示原任务保存数与人工选取的证据数，分别取自任务状态和实际选择。仅在该任务、该批次内按 Amazon 站点及 ASIN 去重；同一 ASIN 的不同批次保留各自原件。验收清单的五条上限是静态样本范围。
2. **对照原件，整理一条商品方向。** 从当前批次选 2–5 个可读的原版候选，由既有只读 `analysis/inspect` 取回事实，人工选择代表候选、按用途和差异对照样本并填写要求。对项目内已有的同站点同 ASIN，原 `link` 只增加当前批次的证据链接，已有 `selectedLinkId` 保持原值。若当前选中的仍是其他批次，用户在「已收集的资料记录」里对准确的当前批次链接点「使用这份资料」，以现有 `select` 命令提交候选 `expectedRevision` 与 `linkId`。复读新 revision、链接和摘要匹配后才允许 `inspect`。原命令把当前决定重置为待研究，历史决定仍随各自原版证据可回看；交接流程不会自动选中链接或沿用旧批准。至少需要两件能支持所写方向的样本，且必需规格和可比较价格有依据，才可确认；其他缺项明确列出。只有一件或只剩标题和图片时，使用已有原件查看入口，提示缺失字段并停在待补证。保留原件与采集状态。页面只展示样本观测及人工判断。
3. **确认并进入同一项目的智能选品。** 操作前复读原批次和所选链接。界面让用户逐项核对平台、用途、规格、差异、代表候选与证据后点「确认并进入选品」。进入时再次按项目权限及原引用核验，并展示这次确认的要求和证据；已收藏货源由人挑选。交接不预选 1688 商品，不产生选品批准、费用或转入记录。原有智能选品入口仍支持直接打开普通项目。

交接只包含以下内容：

| 内容 | 最小合同 |
|---|---|
| 研究范围 | 当前项目、Amazon 美国站、规范化关键词或类目节点、原任务 ID、原 Collection 批次 ID、样本范围 |
| 原版证据 | 代表候选与对照候选的 ID、期望 revision、选中的观察链接 ID 与原任务/批次引用、Collection item/revision、publication ID、原版本与 digest。引用由现有 owner 提供，页面读回原版后才展示 |
| 商品要求 | 平台和用途、每条必要规格及可接受差异、代表候选、可比较的样本展示价区间和币种、可选人工目标价、缺项及逐项依据。样本价与目标价分开，缺失保持为空 |
| 人工确认 | 当前用户、项目及有效成员上下文、当前标签页 handoff ID、确认时的查询代次与证据版本、确认时间。交接与转入复核只在当前标签页使用，静态验收另记用户、时间与精确证据引用 |

## 现有 owner 与最小实现

| Owner / 入口 | 沿用合同及改动点 |
|---|---|
| Data Services / Collection | `internal/product/dataacquisition/query.go` 的关键词/类目规范化和原任务，`internal/integration/persistence/product/dataacquisition/repository.go` 的发布与 Collection 批次，`web/listingkit-ui/src/components/workbench/data-services/data-services-page.tsx` 的原查询和作业回执入口。复用原任务读与原件选择，不新增提交或发布命令 |
| ProjectCenter / Research | `internal/app/httpapi/market_research.go` 的项目授权、来源绑定与候选读写；`internal/integration/persistence/product/research/repository.go` 的原链接与显式切换；`internal/product/research/analysis_read.go` 和 `web/listingkit-ui/src/lib/api/research-analysis.ts` 的原版检查。项目的调研页与选品页复用 `web/listingkit-ui/src/components/workbench/project-center/research-panel.tsx`；在 `web/listingkit-ui/src/app/workbench/layout.tsx` 下放调研、选品和收藏转入弹窗共用的客户端内存状态，只供该交接入口使用 |
| 1688 / 试算 | `research-supplier-review.tsx` 复用已收藏货源的授权回读及保存，SKU、数量、人工理由由 Research 货源核对关系保有；`web/listingkit-ui/src/lib/workbench/research-cost-preview.ts` 沿用现有试算。供应价由人录入并核对适用 SKU、数量档、日期和依据，缺价字段保持空白 |
| 人工批准 / Preparation | 复用 Research 的人工批准、Collection 原批次的明确非空子集转入，以及 Preparation 的原件回读。本交接只提供代表候选，另一次人工批准才允许发起该子集。供应商 SKU 留在原核对关系 |

### 时序、权限与失败边界

- 当前标签页按租户、用户、有效成员、项目和权限上下文隔离，只持有一份草稿。流程为「待选证据 → 待确认 → 已确认」。规范化查询、样本范围、任务、批次、所选证据或商品要求改变时，立即清除确认及当前选品预选；换范围后重新选证据。每次换范围递增代次，迟到的旧任务读取、检查响应和旧确认按原代次丢弃。重复点击同一确认只交接同一草稿。
- URL 仅表达项目、入口或本标签页的 handoff ID 与代次。事实和确认留在内存。刷新、换身份/企业/有效成员/项目、项目归档、权限上下文失效时清空；历史导航若找不到该代次，只提示回调研重新确认。此入口不会从通用选品页的 `sample` 参数恢复旧确认。
- 确认、选品入口及发起批准前，依据当前权限复读项目和所选原件。只接受确认时的原版本与 digest。授权撤回、读失败、期望版本或来源不符时，清除交接显示的要求和依据，保留底层原记录并停止本次操作。Research 的货源核对、开始评估和批准会推进候选 revision；仅对本流程刚完成的操作，凭现有回执中的新 revision 复读并确认同一原证据后可继续。其他变更需重新确认。既有写入口独立鉴权并按其版本规则执行。
- 交接没有数据库写入或外部平台请求。货源核对和批准分别走原命令和事务，响应丢失按原键查回执。沿用已有 API 限制、可取消读取和错误返回；静态核对最多选五件对照。

### 收藏转入弹窗与最后一次复验

固定源码中，选品的旧链接把 `batchId` 与 `transfer` 原件四元组传给 `web/listingkit-ui/src/app/workbench/data/mine/page.tsx`。`web/listingkit-ui/src/components/workbench/collections/collection-page.tsx` 打开 `TransferSelection`，`web/listingkit-ui/src/components/workbench/collections/transfer-selection.tsx` 最终复读 Collection 批次与商品，再交给 `supply.execute("transfer", input)`。这个旧路径没有项目或批准复验。

1. **打开同一个弹窗。** 新交接在 A 的批准回执已复读后，进入收藏页的专用路径 `/workbench/data/mine/research-transfer`，沿用 `CollectionPage → TransferSelection`。路径固定为研究交接模式，参数只带 handoff ID 和代次。`CollectionPage` 从同标签页内存取得项目、批准版本、原批次和 `[A]`，核对后打开只读范围的单件弹窗。参数缺失、无效、过期或上下文变更时留在该路径提示回调研，关闭或禁用旧弹窗，清除展示与预选。研究交接模式不提供「清除预填并重新选择」、整批或普通转入按钮，无法因丢参数退回普通模式。原 `/workbench/data/mine` 的主动收藏转入及旧预填入口保持现有职责。
2. **最终按钮点击后复验，再发出原命令。** 同一个点击中执行可取消的只读校验。用 ProjectCenter 当前读取核对同一活跃研究项目和有效成员，用 Research 当前候选读取核对 A 仍为 `APPROVED_FOR_TEST` 且 revision 等于已核准版本。用原 `inspect` 复读已确认的 2–5 条证据，A 使用已核准的 revision，逐条核对所选链接与原版本/digest。沿弹窗原 Collection 复读核对同一个未归档、可转入的批次及确认页的 batch revision，且 A 的 item/revision/publication/version 仍匹配。缺失、不可读、超时或任一不符即停止。异步读完成后，用最新内存状态再次比较 handoff ID/代次、身份、项目和同批 `[A]`，只允许一个在途点击。通过后才调用 `supply.execute("transfer", {batchId, expectedRevision, itemIds: [A.itemId]})`。检查期间若换词、关闭弹窗、切换身份，或当前上下文和复读结果显示授权失效，作废此次读取及提交。
3. **区分提交复验和事务。** 以上是客户端最终提交复验。`internal/listing/preparation/service.go` 沿用当前身份与供应链权限校验，`internal/integration/persistence/listing/preparation/repository.go` 按原操作键去重，在事务内锁定 Collection 批次并检查 revision、精确子集，原子保存 Preparation 与原件引用。Preparation 原合同未接收项目、批准或 handoff ID，无法覆盖客户端复验完成到服务端提交之间的项目撤权或批准变更。若产品要求把该竞态纳入服务端强制约束，本设计须回到独立评审，确认原 owner 的服务端合同后才可实施。
4. **结果未知时查原单。** 复用原待核实操作键和只读回执。在现有待核实请求元数据中为该笔标记「仅核实」，只用于客户端，避免刷新、交接失效或转到普通页面时从通用反馈重发同一笔写请求。`useListingCommand` 对该笔限制重发，`ListingCommandFeedback` 只提供查原单。交接事实、项目和确认仍只在内存。

## W1–W4 评审与实施证据

| 场景 | 验收断言 |
|---|---|
| W1 正常 | Q1 `under sink organizer` 的 `AMZ-A@v3` 与 `AMZ-B@v1` 形成 USD 24.99–29.99 的样本展示价。人工确认可调宽、免打孔、给管道留空间、单件装，最小宽度可由 38 调整至 40 cm，代表选 A。选品复读 A 原件，再人工选 `SOURCING-S1@v2 / sku-adjustable-white-01`、20 件。合成适用报价 ¥40/件，试算 USD 8.55/次销售尝试，保本 USD 20.00。只批准 A，B 仍待调研。专用弹窗最终点击复读同一项目、A 的批准与原证据及同批 `[A]` 后，仅一次提交。回执精确含 `[A]`，Preparation 读回 `AMZ-A@v3`，1688 SKU 和数量保留在核对关系。提交响应未知时以原键查回执，普通页也无法重发该笔 |
| W2 换词 | Q1 改为 `under sink mat`，即使 Q1 的任务或检查响应随后才到，也只能用新任务的 C/D；原方向、代表和货源选择清空，原记录仍可回看。用同 ASIN 的另批次原件核对 `link` 保留旧 `selectedLinkId`，只在用户显式 `select` 并复读后才对照新批次，旧决定留在历史。已打开的 Q1 弹窗在换词、缺 handoff 参数、项目撤权、A 不再获批或原版不匹配时均不得调用 `supply.execute`，包括失效发生在最终点击的复读过程中；无普通转入降级。另用静态类目 `900107001` 核对方式与批次绑定 |
| W3 证据不足 | Q3 只有 E 的标题与主图，提示缺尺寸、安装方式、可比价格等，不生成可交接的方向，无货源匹配、试算或转入。E 仍可回看 |
| W4 缺有效报价 | 仅见「¥28 起」。SKU、20 件和人工核对理由保存；正常成交与退款情景采购价均为空，试算为 `INSUFFICIENT_INPUT`，无贡献与保本数。用户记录待询价、暂不批准，无转入 |

实施前由独立 Reviewer 按四个场景检查现有任务、批次、项目和原版读合同能否覆盖所需字段，并核对旧响应、撤权、版本冲突和单件回执边界。确认 `IMPLEMENTATION_READY` 后，原 Writer 再用同一组合成原件验证 UI 交接、版本复读和原键恢复，保留 W1 的真实本地 owner 回执。若现有读合同无法取到必需版本或证据，则明确缺项回到设计决定。

真实取数单独准入。先核对企业 A 是否已有可复用的原件，再确认按拟用渠道获取、保存和企业内展示资料的许可、企业账户与操作者权限、可用额度、请求上限及核账证据。三条固定样本和 ¥0.15 参考上限仍按原范围，关键词或类目实取另行确认。试取、AI、实时采集保持关闭，统一实例保持原状。
