# Amazon 单商品插件交接

Refs #905、父设计 #902；设计依据为 PR #903 的 `plugin-capture-ingestion-v1` 冻结基线。此文交付实现入口，不代表已安装、部署或用户验收。

## 入口与准备

沿现有 [current-application 启动方法](../operations/current-application-run1.md)准备独立新测试环境的私密 manifest、ZITADEL 和受限 Product pool。正常启动命令：

```powershell
go run ./cmd/current-application -config C:\private\current-application.json
```

当前装配在既有 Product acquisition 数据库和 Collection 能力均启用时，显式加入 `WithAmazonBrowserCapture()`。缺少 Collection、原 schema、当前授权或受限 Product pool 时拒绝开放；接收保存本身不依赖 Amazon Data Services 的 provider、Temporal 或余额。此实现不初始化/迁移现有实例。

冻结设计 §2 的当前 application 装配对应 `internal/app/httpapi/amazon_browser_capture_application.go`。既有精确依赖 guard 仅登记此文件的 `NewAmazonBrowserCapture`、`NewAmazonCaptureContextAuthorizer` 与 `AmazonCaptureResult`，其他文件仍不准入；该文件不能调用收费 owner、数据库初始化或目标子包。

前端在 `web/listingkit-ui` 用 `pnpm install --frozen-lockfile`、`pnpm dev` 正常启动，沿现有 HTTPS 应用 origin 与可信后端 `LISTINGKIT_SERVICE_API_BASE` 接线。保留原 serverAuth、Origin 和 CSRF 配置，不向浏览器交付后端凭据。

扩展构建沿 [现有扩展说明](../../extensions/product-capture/README.md)，本候选已复用 #906 交付的同一公共壳并登记 Amazon 模块，同时包含原“我的数据”规格图片展示。编译固定应用 origin 的 `/capture/amazon`，来源页无法指定接收地址。本批次没有安装到用户日常浏览器，也未修改保留实例。

## 用户操作

1. 在 us / ca / uk / de / jp 五站的一个商品详情页，明确点击插件采集。默认只观察当前选中商品。[#955可选同族范围](../../extensions/product-capture/README.md) 仅在明确勾选页内选项后，沿有界进度页读取本页真实同族映射，最后仍交原应用一次确认保存；不分页或启动服务端采集。新候选尚未更新原插件或实例，真实验证须另行授权。
2. 插件打开固定 `/capture/amazon`。正常登录、选择企业后，检查当前账号/企业与来源资料及缺失说明，点击“确认保存”。需 `product_sourcing.write` 与 `workbench.collection.manage`；确认时成员 ID 会作为一致性条件传至 native 接收方，再经现有实时授权核验。
3. 服务端完整提交后显示原 capture ID 和固定版本。到“我的数据” `/workbench/data/mine` 查看/下载来源资料；明确关联的图片保持在各自来源规格下。图片是未批准候选，保存不等于资产批准、字段完整、可上架或已发布。
4. 超时、停止等待或刷新后，点击“检查原操作”。只按原键读取原事实，不重取 DOM、不重新保存、不自动生成新键。归档/移动按当前状态显示，不重建引用。身份/企业/成员改变时返回原上下文。

完整冻结 payload 只保留在本页内存。发送前在当前标签会话保存按账号/企业/成员隔离的有界 marker（原键、source/site/ASIN、规范摘要）；marker 保存失败不发送。刷新后 payload 丢失时只能核实原键；未发现完整保存记录也不自动重发。保留恢复页面，不分享其私密地址。

## 合同与限制

Native base：`/api/v1/workbench/sourcing/amazon/browser-captures`；同源 BFF base：`/api/workbench/sourcing/amazon/browser-captures`。

- `POST base`、`POST base/verify` 使用原 UUID `Idempotency-Key` 和严格 v2 payload。
- `GET base/by-key/:key`、`GET base/:capture_id`、`GET base/:capture_id/published` 只读。读权限沿当前 supply/collection read 策略，插件回执要求原成员；普通 Collection/Catalog 资产读取仍沿原 org+actor 规则。
- 同键异载荷冲突；COMMIT 未知保留 `OUTCOME_UNKNOWN`。SRC 同锁与 READ COMMITTED 新语句核实，SRC/Catalog/Collection 部分存在时拒绝修补。
- 免费来自固定 `amazon_browser_capture/v1` producer 与无收费 port 的装配。不会创建收费 job、reservation 或 provider 调用。
- 金额保留十进制原事实；模糊 `$` 不推断币种。缺少规格、父子关系、图片或库存不补造。超过当前详情可读长度/诊断数量的输入在保存前拒绝，不截断后报告完整。
- 每页选中属性读取既有 `variation_*` 和新版 inline twister 标题中的维度/选中值；默认不读取其它变体。明确同族模式逐页复用这个解析器，不把未观察选项补成事实。重复布局值相同只保留一次，冲突值省略并标明缺失，显式隐藏节点不作为选中事实。
- `#feature-bullets` 的商品要点按来源顺序写入现有纯文本 `description`；原 `#productDescription` 有值时保留在前，段间空行、要点间换行。脚本、显式隐藏内容、区块标题和区块外详情链接不带入。合并后仍受原 8 KiB UTF-8 上限，不截断保存；未观察到描述、要点或选中属性时由原 `missingFacts` 提示。此处没有新增卖点字段、富文本或保存 owner。
- 富文本、批量、关键词、自动转入、真实来源采集、安装部署、保留环境保存/计费和用户验收均不在本次执行授权内，均为 **NOT_RUN**。

开发测试使用本任务独立 PostgreSQL 的空测试库与合成 DOM。运行入口与访问说明不构成真实环境操作授权。
