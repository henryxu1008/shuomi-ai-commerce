# 1688 window.context 定位合同

当前定位规则供采集链路使用；来源页脚本是输入数据，不是可执行授权。平台提取器和浏览器插件保持相同的边界、顺序与重复项语义。

## 定位算法


输入：单个**无 `src`** 的内联 `<script>` 文本 `text`。

1. **找赋值点**：从左到右找第一个 `context`，满足：
   - 匹配 `(?:window\s*\.\s*)?context\s*=\s*`；
   - 且 `context` 前一字符不是 `[A-Za-z0-9_$]`，也不是 `.`（排除 `foo.context=`）；
   - 该条件天然排除 `window.contextPath = ...`（`context` 与 `=` 之间必须是空白）。
2. **唯一性**：所有内联脚本中符合条件的赋值点必须**恰好 1 个**。0 个或 ≥2 个 → `UNSUPPORTED_PAGE`（保留现有"重复即拒绝"语义）。注意实现顺序：第 4 步的"RHS 之后只允许空白与一个 `;`"逐命中先行检查，因此第二个赋值点通常先以 `trailing` 被拒。两者都是 fail-closed 的 `UNSUPPORTED_PAGE`，无语义差别；实现时不必强求错误码分类。
3. **取 RHS**（去前导空白后）：
   - 以 `{` 开头 → **Case A（静态字面量）**：字符串/转义感知配平扫描出对象字面量；其后只允许空白和至多一个 `;`，否则 `UNSUPPORTED_PAGE`（保留现有"拒绝可执行后缀"语义）。
   - 以 `(` 开头 → **Case B（IIFE）**：配平扫描第一组括号（被调用表达式）；其后必须紧跟 `(`，否则 `UNSUPPORTED_PAGE`；配平扫描实参表，按顶层逗号切分；取其中**对象字面量**形式的实参，**必须恰好 1 个**，否则 `UNSUPPORTED_PAGE`。该对象即 payload。
   - 其他 → `UNSUPPORTED_PAGE`。
4. **RHS 之后**只允许空白和至多一个 `;`。

Case B 在真实页面上：实参为 `window.contextPath`（标识符，非对象）与 `{...}`（对象）→ 恰好 1 个对象实参。


## 输入与安全不变量


- 只解析脚本**文本**，任何情况下**不读取 `window.context` 全局**、不 `eval`/`Function`、不访问 cookie/storage/网络（插件的 `extractor.test.ts` 已对这些设置抛错 getter）。
- 严格 JSON：拒绝重复键；深度 ≤ 32；节点数与字节数有界；`offerId` 必须与来源一致。
- 任何不确定情形一律 **fail-closed** 为 `UNSUPPORTED_PAGE` / `ErrUnsupported`，绝不产出猜测事实。
- 服务端 `failFetch` 的既有映射（`SOURCE_UNAVAILABLE`）与插件 `CAPTURE_REJECTED` 语义不变。

#### 插件扫描前输入预算（2026-10-31，#399 资源缺口修复）

用户已授权修复 #400 review 4057387941；Design Basis 为 Reuse Existing Architecture，当前准入及单一 Writer 见 #399。本节只补当前插件的既有资源不变量，不恢复暂停的服务端解析器工作，也不新增通用解析平台。

- 现有 `MAX_BYTES = 2 MiB` 同时约束本次1688无 `src` 内联脚本扫描的累计 UTF-8 输入，包含未命中 `context` 的脚本；脚本数量仍最多256。超限必须在 `includes`、定位或配平扫描前以 `CAPTURE_TOO_LARGE` 拒绝，不能仅限制提取后的对象。
- 读取整段 `Text.data` 前，先用常数时间的 `Text.length` 与剩余字节预算比较（UTF-8字节数不会小于UTF-16码元数）；通过后只对这个有界字符串使用既有 `byteLength` 核准UTF-8并扣减预算。恰好等于上限允许，后续脚本仍共用剩余预算。
- 每段脚本处理前检查原8秒截止，过期以既有 `CAPTURE_TIMEOUT` 停止，不再完整读取或扫描下一段；不增加timer、worker或后台保活。
- 原candidate、归一化JSON和payload的2MiB检查、65536节点、深度32、8秒、JSON/IIFE/数字键/严格字段、原始金额/ID及内容摘要不放宽；保存、权限、持久事实和UNKNOWN/恢复合同不变。
- 回归复用原extractor测试，覆盖完整读取前拒绝、无关脚本累计、UTF-8前缀、精确上限及扫描阶段截止；不是新runner或真实浏览器验收。
- Amazon相册/同族的候选筛选及无关脚本规则属于另行批准的合同；本次不把1688累计扫描准入直接套到它们，也不处理批量入口的manifest权限预检。


## 大小与重复项边界


用户在验证码可读后重新提供会话，`§3.1` 定位规则在**真实页面**上命中正确（3 处 `context=` 命中里 2 处被正确排除，接受项位于脚本偏移 `34`，RHS 为真实 IIFE `(function(b,d){var c=d.module|…`）。但采集仍然失败，实测到两个与夹具无关的真实缺陷：

- **D6 — 载荷不是严格 JSON。** 真实载荷里 `"skuWeight":{6290953586037:0.3000,…}` 使用**未加引号的数字对象键**；`parseTree` 报 **90 个解析错误**（全部落在 `skuWeight`，偏移约 `87084`/`87104`）。仅在对象键位置补引号后 **错误 90 → 0**，标题/SKU/图库均可读，`roundTripOk: true`。
  - 处理：新增 `quoteBareNumericKeys()`，**只**在对象键位置给裸十进制数字补引号；字符串内容与其余 token 逐字复制；补引号后**仍执行**严格解析，因此注释、尾随逗号、重复键依然被拒（有对应用例）。
- **D7 — 节点预算过小。** 归一化后实测 **nodes = 16501**，超过插件原先的 `16384` 上限 ⇒ 即使解析成功也会判 `CAPTURE_TOO_LARGE`。
  - 处理：上限 `16384 → 65536`（约为实测值的 4 倍）；**2MB 字节上限仍是主要资源护栏**；服务端 `4096` 上限与通道①不在本次范围。

**parserVersion 仍为 v2**：D6/D7 只放宽输入接受范围，未改变任何字段语义，且当前不存在 v2 持久化数据。如需 v3 需用户另行决定。

D6/D7 之外的第三项限制在**服务端**（不在本设计范围内，如实记录）：真实载荷经 `MapAcquisitionEvidence` 展开后 `warnings = 275`、`missingFacts = 183`，其中 `warnings` 超过 `MaxSourceEnvelopeCollectionItems = 256`（触发时累计 `items = 475`，仍低于 `MaxSourceEnvelopeAggregateItems = 1024`，`stringBytes = 18375`），提交被 `413 SOURCE_TOO_LARGE` 拒绝。该上限属 server owner，需单独产品决定（见 §6）。


用户对上述服务端上限选择**选项 C**：保留每变体明细，但把**同字段模式 + 同原因/码**的重复项折叠成一条并携带数量，使集合大小与变体数解耦；不做「抬高上限」（选项 A）也不做「按 (code,message) 去重丢失 per-variant field」（选项 B）。

- 位置：`MapAcquisitionEvidence`（`internal/product/sourcing/acquisition.go`）在客户端警告回显之后、`validateSourceEnvelopePreflight` **之前**调用 `foldSourceEnvelopeRepeats(&envelope)`，因此被持久化的 durable command 本身就是有界的。
- 字段模式归一：`variants.3.sku`、`variants[0].sku` 统一为 `variants[].sku`（先 `\[[0-9]+\]` → `[]`，再 `\.[0-9]+` → `[]`），因为服务端用点号索引、插件用方括号索引。
- **单例保持逐字节不变**：仅当同组项数 > 1 才折叠；`count == 1` 直接返回原条目，`Count` 为 0 且 `omitempty`，因此既有 envelope 与既有断言（如 `public_test.go` 的 `variants.0.sku`）不受影响。
- `MissingFact` / `SourceWarning` 各新增一个**可加**的可选字段 `Count int \`json:"count,omitempty"\``（不是把数量编进 `Reason`/`Message` 文本：警告 DTO 不传 `Message`，文本编码的数量无法展示）。
- 未改动任何上限常量（`MaxSourceEnvelopeCollectionItems` 仍为 256）。
- 实测（真实采集证据体）：`missingFacts 183 → 7`、`warnings 275 → 11`，归一化后 envelope 17 760 字节（< 2 MiB 上限）。
