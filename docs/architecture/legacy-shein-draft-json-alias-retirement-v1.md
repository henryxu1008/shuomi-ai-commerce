# #301 旧 SHEIN 草稿 JSON 别名兼容整链退休设计

Status: **DESIGN_REVIEW / 方案候选；尚未 IMPLEMENTATION_READY，生产实施未授权**。Design Basis: **Independent Architecture**。本次实际只提交本文、Architecture README 与 Legacy Register 三文档；源码、原测试、fixture、workflow/inventory 全不改。准确设计 HEAD、CI、独立设计复核和用户确认留主要 PR。接单及上一当前段存档见 [#301 / 6358945095](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6358945095)。

调查基线：main `3ba87121afe83244631ee8714a8311f3f34f7b78`（#901 正常合并；其 push/main CI39909308665 成功是前批证据，不代替本设计或未来实现）。以下处置均是待确认设计；#827 的原 KEEP 在本方案获确认前仍有效。

## 1. 用户结果、范围与 Authority

使用者为维护当前 SHEIN 草稿生成、预览、保存和离线诊断的维护者。交付一个完整源码结果：当前 Package 只接受/输出 `draft_payload`、`preview_payload`、`final_submission_draft`，旧三别名及其回填、镜像、codec、历史 readiness 回退和专属尾部同批退出；当前能力继续沿原 owner 使用。不把字段、setter、测试、前端版本或 fixture 拆成机械小批。

这是用户明确要求的源码退休维护，未证实有核心用户操作必须等待本批；**不增加内部试用前置**。不新增页面、菜单、请求入口、人工流程或验收平台。当前独立 SHEIN 登录与 Store 同步/fence/UNKNOWN、官方发布和所有此前保留能力不受本批接线。

适用依据：

- [Greenfield](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[项目边界](project-boundaries.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)：无兼容类别、旧 ID 映射、迁移、双读/双写或第二事实源。
- [#827 完整 Publishing 退休设计](legacy-shein-publishing-runtime-closure-retirement-boundary.md) §2/§5.2/§6.1/附录 C、[#824 支撑闭包](legacy-numeric-listing-support-closure-retirement-boundary.md)：本次只拟替代 §5 中逐项指定的 KEEP，不恢复旧 Task/Workspace/managed/runtime。
- [当前持久输入绑定设计](../superpowers/specs/2026-09-05-shein-offline-validation-input-binding-design.md) §3.1：JSON 字段/normalization 变化必须升级 binding_version，规则语义变化升级 rule_version；继续复用标准 Go JSON、SHA-256 与已锁定的 sigs.k8s.io/json。
- [当前 Validator 合同](../../internal/marketplace/validator/README.md)、[Listing Record owner](../../internal/listing/record/README.md)、[当前诊断 UI 合同](../engineering/shein-diagnostic-ui.md)：诊断不授权发布，存储/权限/幂等保持原 owner。
- [最终 UI/IA Authority](../product/final-ui-ia-authority.md)，Figma 当前非归档 `31:463`：不改页面/导航/中文标签/交互。前端只调整原 wire schema 的精确版本，技术 FieldPaths 跟随当前字段，不创造新的产品投影；不需 Figma 写入或新界面设计。

Must：保留草稿/预览、严格输入和深拷贝、有效平台/图片/价格/敏感词/尺寸/审核规则、当前身份/租户/事实/事务/幂等/UNKNOWN、副作用控制、独立登录/Store 同步；同一批完成后端与前端版本，不保留旧版本 fallback。Should：复用原 setter/codec/fixture/guard，不新增框架或 mapper。Out of Scope：旧记录迁移或清理、整退 Publishing/Workspace/RequestDraft 类型、删除全部 SkcList、重写当前官方发布/图片规则、真实运行验收、workflow/inventory/依赖/保护变更。无新增 Accepted Risk。

Threat Model 只覆盖双字段事实混淆、旧别名重新进入当前保存/判定、失配的摘要/规则版本、误删有效规则/私有 clone 状态以及 tenant/不可变记录破坏。不扩全局安全目标。出现具体当前外部兼容义务或必须改变本矩阵外事实/权限/持久化协议的证据时报告用户，不自行加迁移或兼容路径。

## 2. Design Basis、根因与版本决定

采用 **Independent Architecture**：变更 Package 的持久化 JSON 接受/输出合同和内容绑定，连带规则/前端协议，且闭包超过30工件。不是 N/A 文档修改或普通删字段。设计先行；独立设计复核达到 IMPLEMENTATION_READY 且用户确认 §1–§7（含 §6.1）后，原唯一 Writer 才在 Issue/PR 记录并读回 Ready，再改正式源码。

实际根因：`model.go` 三旧字段仍声明为历史 JSON 兼容面；`NormalizePackageSemanticFields` 读旧回填并镜像，两个 setter 双写，Marshal/Unmarshal 触发修改；strict decoder 虽拒绝未知字段，却因这三字段属于 schema 而仍接受它们。单改字段或 serializer 会漏掉 clone、readiness、内容摘要及前端精确版本。当前标准库/strict 库足够，不新增自制 JSON 方案。

| owner 常量/合同 | 当前基线 | 拟一次切换为 | 原因 |
| --- | --- | --- | --- |
| SHEIN typed Validator.RuleVersion | shein.offline_package.v1.1 | **shein.offline_package.v1.2** | 缺 DraftPayload 时 summary-only 不再提供 SKU/variants ready；其它规则不变 |
| DiagnosticRuleVersion | shein.offline_package.v2.1 | **shein.offline_package.v2.2** | 同一 evaluatePackage 的上述变化及准确 FieldPaths；不复制第二规则判定 |
| BindingVersion | shein.persisted-input.go-json.v1 | **shein.persisted-input.go-json.v2** | 去三别名/normalization 后 JSON 合同和绑定字节变化 |
| 原共享合成 fixture 数据集 | product-shein-shared / 1.0.1 | **1.0.2** | 只校准版本、受影响向量/FieldPaths 和原 loader pin；保留9个PS用例与15项规则，不扩验收平台 |

新值由原 validator owner 声明；应用依赖原常量自动注入，不新增配置、协商、旧新并行或版本映射。typed 旧 rule、diagnostic 旧 rule/binding 按原 UnsupportedVersion 拒绝，TS Zod 只接受新精确版本，不 union 旧版本。HTTP error/status 和 action/site 准入保持。

## 3. 实际消费者：contract → implementation → injection → consumer

| 路径 | 当前证据与本批边界 |
| --- | --- |
| 显式 DRAFT-S1 装配 → listing/record.Service → marketplace/shein/draft.Builder → Publishing Assembler | `app/httpapi/shein_records_application.go` 注入 Builder 与 ExactApprovedAssetValidator。Builder 只用精确 Catalog/ApprovedAsset，`json.Marshal(pkg)` 后 strict 校验；保持构造/授权/approved 归属/确定性规则，不注入 resolver/AI |
| record.Service → 原 listingrecordstore | 保存不可变 payload/diagnostic、PackageHash/DiagnosticHash、RuleRevision/PolicyRevision；原共享事务和 operation 幂等不变 |
| 显式 DiagnosticService → Reader.ReadOfflinePackage → DiagnosticValidator | 授权 owner 限定读、有界 JSON 副本 → strict decode →同一对象编码/摘要与 evaluatePackage；保持权限/取消/上限/错误 |
| typed Validator → ClonePackageForPersistence → evaluatePackage/workspace | 继续纯内存计算和 deep copy；去 aliases 不删除私有 SaleAttributeResolution map 的复制 |
| Publishing 内部 → NormalizePackageSemanticFields | 14生产文件直接包含该调用/定义；全部在附录A，清掉调用并保留 nil/错误语义，不能留下 no-op facade |
| Package.RequestDraft/PreviewProduct/FinalDraft | 包外无生产 Go 直接字段消费者；遗留直接 selector/literal主要在8个测试文件。RequestDraft/FinalDraft **类型本身**及 BuildPreviewProduct 等同名有效函数保留 |
| workspace 的 request_draft/preview_product | 仍是稳定 check ID、taxonomy/repair target；不等同旧 JSON 字段。ID/分类/标签/顺序/修复路由保持；FieldPaths/消息中的三旧字段路径准确替换 |
| TS shein-diagnostic parser → 原 BFF/页面和 fixture | wire schema 硬编码 v2.1/go-json.v1，必须与后端同PR换新值；页面仍展示服务端诊断，不客户端重算规则或制造就绪 |

不能把 Go 包被编译当作默认正式装配：`NewSheinRecordApplication` 注释和构造是显式 admitted Product DB 入口，并非默认 current-application 配置开关。当前私有草稿试用/官方 goods/TargetRecord 是另外的现有路径，不因名称含 SHEIN 而改接本 Package 或宣称它已普遍可用。现有当前入口与两个独立 worker 的真实运行状态仍 UNKNOWN；本批不启用它们。

原显式入口 `POST /api/listing/shein-records`、`GET /api/listing/shein-records/:record_id/offline-diagnostic` 和页面深链 `/workbench/shein-records/{record_id}/diagnostic` 保持路径/权限/请求语义；本方案只约束既有服务端载荷与诊断wire版本，不创建列表、选择器或运行入口。

## 4. 持久化、版本和失败语义

| 边界/owner | 拟变化 | 保持的当前语义 |
| --- | --- | --- |
| publishing/shein Package | 仅三canonical字段序列化；旧三字段根层即使null/与新值相同也不属于strict schema | 类型、其它字段/tag/数值及nil/empty语义、SKU顺序/重复、Metadata和公开resolution字段保持 |
| listing/record + app/listingrecordstore | 新创建的原记录携新RuleRevision/PolicyRevision及新摘要 | 原两表/列/约束/UoW/SQL/owner/operation唯一性/授权/大小防线原样，无新schema或数据库owner |
| 当前 DiagnosticValidator | v2绑定固定顺序envelope：binding_version/marketplace/site/action/rule_version/package；编码新schema并SHA-256 | Go map排序/转义/数组/nil行为、只读独占副本、不重新获取事实、不把read/evaluated时间放digest保持 |
| 原typed clone | 只复制canonical字段；去alias归一化和镜像身份断言 | 深拷贝全部可变值/当前private maps，nil输入、错误及不修改输入保证保持 |
| 原权限/外部状态/发布 | 无变化 | tenant/actor重验、freshness/NOT_EVALUATED、预算/幂等/claim/fence/UNKNOWN、Store及独立worker原owner保持 |

### 4.1 新合同的准确接受/拒绝

- 三canonical字段的合法有界 JSON：按原schema/规则处理。`{}`或缺草稿仍可作结构合法的诊断输入，但草稿/variants缺失不得由summary填成ready；不把syntax和业务readiness混为一层。
- root旧 `request_draft`/`preview_product`/`final_draft`，无论旧值独有、与新值相同、相冲突或null：strict unknown-field拒绝，无回填/忽略后成功/再试旧decoder。Metadata等自由map的同名字串、平台DTO和稳定check ID不做全局词禁令。
- 去 Package 自定义 MarshalJSON/UnmarshalJSON；标准库只编码当前struct。唯一不可信持久输入仍走原 DecodePersistedPackageStrict。clone内部原json.Unmarshal只消费本次自有canonical编码，不成为另一个外部reader。
- strict仍复用 sigs.k8s.io/json；duplicate/unknown/case mismatch、UTF-8/UTF-16、深度64、2MiB、尾随值、数值overflow/underflow、嵌套custom decoder审计与private-map不可注入完整保持。删别名冲突helper由更早unknown拒绝接替，不删现有非法输入测试。
- DraftPayloadOf只读canonical值；SetDraftPayload/SetPreviewPayload保留有效canonical写入及nil语义，不重建Normalize no-op函数。旧别名相关 helper/注释/调用尾部全退。

### 4.2 既有记录、重试与运行边界

本批不读取既有实例/数据/凭据，不能推定实际存在或不存在旧记录；不授权部署或原地升级环境。现有 rows/物理对象状态 UNKNOWN，不更改、重算、回填、reseeding或删除历史payload/diagnostic/hash/version。

保留原通用 Record Reader/List 的不可变字节读取/owner校验，不新增历史JSON适配器：重新诊断仍只调用当前strict codec。含旧别名的载荷被拒绝；字节本来符合新canonical schema的输入按现有reader/当前规则处理，不基于历史版本选择旧decoder，不把旧创建报告改写为当前PASS。旧expected digest依原比较返回stale，外部freshness证据不得换摘要“续期”。

当前 Service.canonicalInputHash 已包括rule/policy revision。相同operation key跨版本重提按原repository input-hash不同的ErrConflict拒绝，不复用旧receipt、不覆盖记录；同版本同payload的响应丢失/并发/重试沿原幂等归一。原 record_test.go 只补窄的同事实不同rule/policy生成不同input hash断言，复用原replay/conflict用例；不改Service/repository生产代码或另造事务。声明版本变更不是增加operation重试、恢复协议或旁路。重启/取消/撤权/依赖失败按现有owner拒绝或返回，无新补偿/outbox/Saga。

## 5. #827 KEEP 替代与删除/保留矩阵

| 原KEEP或工件 | 拟准确替代 | 明确保留 |
| --- | --- | --- |
| #827 §5.2当前Package、§6.1及附录C保留Draft/Preview/Final镜像 | **只**退休Package三个旧字段及镜像/回填/自定义JSON和两个专属别名比较helper，清理所有调用 | Package、RequestDraft/FinalDraft类型与三个canonical字段/其它公开结构、现有setter的有效写入 |
| #827/#824的typed v1.1/diagnostic绑定旧版本 | 按§2原owner同步切换新版本，只接受新值；上述准确旧KEEP被本次确认替代 | 同一规则计算、Snapshot/error/租户/输入界限/NOT_EVALUATED、官方Kernel和API不变 |
| HasAnySubmitSKU + workspace.submitVariantStructureReady | 去缺DraftPayload时以Package.SkcList给SKU/variants就绪的历史回退 | 有draft时的权威顺序/结构拒绝、所有其它有效校验 |
| Package.SkcList/HasSingleSKC/HasSKCImage等 | **KEEP**；只删除Normalize调用，保留原有效分支 | 当前Assembler/ApplyResolution/图像/审核/定价对摘要的真实消费、single-SKC主图与swatch规则不因含fallback字样被删 |
| 原消息/FieldPaths | 准确三旧字段路径映射到draft_payload/preview_payload/final_submission_draft；只替换对应技术字段名 | request_draft/preview_product等check ID、taxonomy/code/中文标签/顺序/修复路由、其它路径/消息保持 |
| workflow/inventory/当前官方与worker/现有schema | **零改** | 包括semantic-field-cleanup-inventory.md等全部inventory、历史/计数/ceilings/解析器；不按旧名字清理有效当前能力 |

Legacy decision: **RETIRE**。Reusable behavior: 当前草稿/预览/严格codec/深拷贝/平台规则继续原位使用；本批无新增EXTRACT代码、mapper或owner。Current owner: Publishing当前值与codec、Marketplace validator/draft/workspace、Listing Record持久化各自不变。Cutover/deletion condition: 独立设计PASS/IMPLEMENTATION_READY且用户确认→Ready读回→原guard与原行为测试RED→完整同批切换/GREEN→原CI/准确源码最终复核；不等于物理运行态清理。

## 6. 必要验证与 §6.1 精确处置

设计阶段只做文档/链接/索引/实际调用/元数据检查及原CI，不运行生产构造器或旧协议测试保活。实施后沿原测试TDD：三旧key拒绝、canonical-only输出、缺草稿summary不能ready为RED；同批退镜像、升级版本并GREEN。复用原SDK/empty/map-order/nil/数组向量，独立核算新envelope SHA；不能只以实现生成它自己的expected值。

### 6.1 原检查、fixture、文档和冻结边界

生产范围仅附录A的19文件；测试/fixture范围仅附录B的23文件；文档范围仅附录C的9文件，合计51个范围工件（不要求为计数制造无效修改）。本轮实际仍仅三文档。

| 对象 | 获实施授权后的最小处置 | 不得改变 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 原tracked Go/AST设施新增准确Package旧字段、别名Normalize/自定义JSON/专属helper缺席及canonical/strict/worker KEEP保护 | 全部其它guards及读取/解析错误即失败；不全局禁RequestDraft/FinalDraft类型或旧check ID，不missing-skip/新runner |
| 原Publishing测试 | 旧field fixture换canonical；原镜像/旧字段双输出断言改canonical-only/旧key拒绝；缺草稿SKU fallback改false | 组装、价格/图片/敏感词/销售属性、nil、private-map深拷贝和全部非兼容断言；不是删除整个测试文件 |
| persisted_input_test | 原alias冲突case仍保留错误预期并明确已为旧key未知；原custom decoder审计删除Package被豁免条件，time.Time原例外保持 | 全部严格输入limits/UTF/numeric/private-map测试，不放宽unmarshal/error |
| 原Validator/Workspace/Builder测试 | 跟随新版本/canonical fixture/准确FieldPaths；补旧版本拒绝和现有builder输出只有新key | 稳定ID/分类/全部有效规则和authoritative Draft优先，不重算外部规则 |
| TS原schema/两fixture文件 | 三文件只对新rule/binding精确值、旧值拒绝作最小校准 | Zod严格性、纳秒比较、stale/error映射/租户BFF/页面/布局/状态流与其它测试不改 |
| product_shein_v1 manifest/cases/README + 原loader/deterministic consumer | dataset1.0.2、新rule/binding与准确cases hash；仅PS-008绑定向量、PS-009 subject digest及实际受影响路径/原schema样例，既有alias等价正例改canonical等价/旧key拒绝 | 9个PS ID/层/15规则/权限/过期/边界/PG语义断言不减少，不扩工具/验收矩阵；不执行PG或实例作为新前置 |
| 原持久绑定spec、#824/#827、semantic-field migration checklist | 仅准确supersession顶注，引用本设计；旧KEEP/版本/双输出建议的当前意义仅在本矩阵被替代 | 旧正文/历史表/计数逐字保留，不重写历史或其它SDS/Submission合同；所有inventory零改 |
| 当前Validator README/诊断UI说明/三设计索引 | 说明新版本/canonical字段与当前路径；不恢复旧UI/发布入口 | 产品交互、当前规则owner/权限/新系统基线不变 |

原开发自检限相关Publishing/Marketplace测试、原共享deterministic fixture、原TS wire测试/typecheck及当前命令编译元数据；完整原PR CI在稳定候选执行。不新增故障矩阵/手工验收session/runner/专用PostgreSQL，不为本批构造真实服务/读现有配置。现有PG并发/授权等未变测试由原CI执行，不削弱或额外变成试用前置。真实运行/用户验收 NOT_RUN。

若真实CI要求矩阵外修改、原计数/检查/KEEP改变或发现实际未覆盖消费者，先提供准确提案与影响，不自动修改workflow/inventory/其它guard/依赖，也不以修改fixture来迎合错误实现。

## 7. 评审、Ready、交付与停止点

1. 当前仅Independent Architecture设计候选，生产未Ready。独立设计检查绑定准确HEAD，最多两轮正常评审；finding先按当前Must与实际后果分类，IMPLEMENTATION_TEST在实现层收敛，非BLOCKER不无限重开设计。
2. 准确设计复核PASS/IMPLEMENTATION_READY且用户确认§1–§7/§6.1后，在#301/主要PR读回Ready，原唯一Writer同一分支整批实施。设计授权不等于生产修改/合并/部署/关单授权。
3. 原GitHub设计/最终复核Running不重复触发；启动失败不当finding/PASS。未经用户本轮授权不默认新增AI互验线程；设计与源码最终复核授权/准确HEAD分别记录，修复只复核增量。
4. 实现与必要自检/原CI完成后进入In Review，交付准确source HEAD、真实diff/调用路径和一次最终检查；不自签产品/环境验收。合并必须另获准确HEAD授权，合并后仅实际merge SHA的push/main整轮CI才是主干证据。
5. #301保持OPEN，#29不重开、不接管#692；更新Issue前新读正文，仅改当前段与准确PR证据。从“以下保留 #726 已合并批次与历史证据，不扩大本轮权限。”起63542字符历史与勾选逐字保留，正文不超过65536字符。
6. 不操作既有实例/数据/凭据/进程，不读取私密状态或运行浏览器/provider；workflow/inventory/依赖/锁/保护零改。当前草稿/预览/严格解码/有效规则、采集、独立SHEIN登录/Store同步保留，不增加内部试用前置。

## 附录 A：生产修改边界（19）

以下均为mixed原文件最小修改，不按目录整退：

```text
internal/publishing/shein/model.go
internal/publishing/shein/semantic_fields.go
internal/publishing/shein/model_contract.go
internal/publishing/shein/persisted_input.go
internal/publishing/shein/package_persistence_clone.go
internal/publishing/shein/submit_readiness_status.go
internal/publishing/shein/apply_resolution.go
internal/publishing/shein/assembler.go
internal/publishing/shein/listing_copy.go
internal/publishing/shein/preview_adapter.go
internal/publishing/shein/sale_attribute_readiness.go
internal/publishing/shein/sensitive_content_sanitizer.go
internal/publishing/shein/size_attribute.go
internal/marketplace/shein/workspace/package_template_validation.go
internal/marketplace/shein/workspace/submit_payload_readiness_checks.go
internal/marketplace/shein/workspace/submit_payload_validation_readiness_checks.go
internal/marketplace/shein/validator/validator.go
internal/marketplace/shein/validator/diagnostic.go
web/listingkit-ui/src/lib/api/shein-diagnostic.ts
```

## 附录 B：原测试/fixture边界（23）

```text
internal/publishing/shein/semantic_fields_test.go
internal/publishing/shein/model_contract_test.go
internal/publishing/shein/package_persistence_clone_test.go
internal/publishing/shein/persisted_input_test.go
internal/publishing/shein/assembler_test.go
internal/publishing/shein/preview_adapter_test.go
internal/publishing/shein/sale_attribute_value_matcher_test.go
internal/publishing/shein/submit_readiness_status_test.go
internal/marketplace/shein/workspace/package_template_validation_test.go
internal/marketplace/shein/workspace/submit_payload_readiness_checks_test.go
internal/marketplace/shein/workspace/submit_payload_validation_readiness_checks_test.go
internal/marketplace/shein/validator/validator_test.go
internal/marketplace/shein/validator/diagnostic_test.go
internal/marketplace/shein/draft/builder_test.go
internal/app/httpapi/shared_regression_deterministic_test.go
internal/app/httpapi/shared_regression_fixture_test.go
internal/listing/record/record_test.go
tests/listingkit_backend_retirement_test.go
web/listingkit-ui/src/lib/api/shein-diagnostic.test.ts
web/listingkit-ui/src/test/fixtures/shein-diagnostic.ts
internal/app/httpapi/testdata/product_shein_v1/manifest.json
internal/app/httpapi/testdata/product_shein_v1/cases.json
internal/app/httpapi/testdata/product_shein_v1/README.md
```

## 附录 C：文档边界（9）

```text
docs/architecture/legacy-shein-draft-json-alias-retirement-v1.md
docs/architecture/README.md
docs/refactoring/legacy-register.md
docs/architecture/legacy-shein-publishing-runtime-closure-retirement-boundary.md
docs/architecture/legacy-numeric-listing-support-closure-retirement-boundary.md
docs/superpowers/specs/2026-09-05-shein-offline-validation-input-binding-design.md
docs/development/listingkit-semantic-field-migration-checklist.md
internal/marketplace/validator/README.md
docs/engineering/shein-diagnostic-ui.md
```
