# Walmart 美国站首期操作与配置

关联 [#733](https://github.com/qq550723504/task-processor/issues/733) / [PR #734](https://github.com/qq550723504/task-processor/pull/734)。设计依据：[冻结架构](../architecture/walmart-us-v1.md)、[官方合同证据](../architecture/walmart-us-contract-evidence.md)。本次只交付实现与隔离验证；没有合并、部署或访问真实店铺。

## 用户路径

1. `/workbench/stores` 创建 Walmart、US 店铺，在详情的官方 API 授权中选择已配置应用并前往官方授权。回调立即清理 URL 中的 code/state；原用户和原企业完成连接，再确认平台读取到的商家名称及 Partner ID。凭据仅在 Store 服务端加密保存。
2. 授权页可明确刷新官方授权。刷新结果不明时停止平台访问，重新官方授权；不自动重发 refresh。刷新或重授权改变 generation 后，刊登前重新读取规则并保存、批准新 Target。商家占位及随机 SKU namespace 保留。
3. `/workbench/store-products` 选择该店铺同步商品。显示 SKU、WPID/GTIN、平台名称/售价和原生发布、生命周期及可售状态，保留取得时间、部分覆盖和未知。首期不读取库存数量。
4. 既有供应链入口 `/workbench/supply` 中打开已保存商品，选择 Walmart 店铺，逐件“查看 / 补全”。明确选择匹配已有目录 `MP_ITEM_MATCH` 或新建卖家配送商品 `MP_ITEM`；读取已配置版本的官方规则。填写真实标识、成色、首次报价、重量和条件必需属性。完整商品须选用该商品、该版本已批准的稳定公开 HTTPS 图片。
5. 保存只生成本地 Target。可展开检查准确载荷、来源 SKU → sellerSKU、规则版本和指纹。CompletionDraft 只显式采用已保存版本的标题/描述；结构化候选仍保留在原草稿。
6. 选择已检查的商品后确认刊登；批准固定 record/revision、Feed 类型、SKU 集合及 wire hash。整个批次超过当前已加载并检查范围时，先分组选取商品。本期不自动适配 Walmart 或调用新的 AI。
7. Operation 完成表示 Feed 已受理。商品详情“查询平台结果”分别显示 Feed 状态、逐 SKU ingestion 和实际 publication/lifecycle/availability。每次最多刷新 10 个成功 SKU 的发布事实，重复查询优先取得尚未读取或较旧的 SKU；逐项显示发布时间戳。仍持有 SKU 的每个 Feed 都保留其最新观察，重复查询不会挤走其他 SKU；历史 Feed 可单独刷新，已转入修复操作的 SKU 不会被旧查询覆盖。
8. 完整终结结果确认的失败 SKU，可从原 Feed 选择子集，修正资料、保存新 Target 后再次明确批准。仍在处理、缺页、重复/缺失 SKU、计数不一致、未知结果和成功 SKU 不能据此重发。分次修复保留原批次剩余失败项；当前 Feed 未安全终结时禁止第二次提交。

没有取得 Feed ID 的 UNKNOWN 保留原操作、关联编号、批准时间和资料指纹。页面只提供查看原结果，不提供重置、手填商品号认领或盲重发入口。

## 配置与启动

使用当前应用、Console、独立 Store/Product/Asset 数据库和既有 Temporal worker 组合。不要改 RUN-1 或共享保留环境。本批未创建可登录的试用实例；以下是后续获准环境的正常启动方式。

现有完整私密 manifest 的 `storeCenter` 中加入以下片段；它不是独立可启动配置。应用获批、scopes、主体/商家及平台流程由用户处理。版本必须替换为该应用实际支持的准确版本。

```json
{
  "walmartUS": {
    "clientId": "<approved-client-id>",
    "revision": "walmart-us-r1",
    "environment": "sandbox",
    "callbackURL": "https://<approved-console-host>/workbench/stores/walmart/callback",
    "clientSecretFile": "C:/private/walmart/client-secret.txt",
    "credentialKeyFile": "C:/private/walmart/credential-key.txt",
    "credentialKeyId": "walmart-us-key-1",
    "fullSpecVersion": "<exact-full-spec-version>",
    "matchSpecVersion": "<exact-match-spec-version>"
  }
}
```

Secret 文件为私密普通文件；credential key 为 32 字节随机密钥的 Base64。与其他应用分开文件及 key ID。开发者凭据、token、code、state 不进入前端配置、工作流、Product 数据库或普通日志。环境固定为 `sandbox` 或 `production`，不从客户端接受 API host。未配置应用时 Store CRUD 保持可用，真实授权/刊登明确不可用。

只对已授权、全新空数据库显式初始化。Store 使用现有命令新增的 capability flag；Product 使用原初始化器、开启现有 supply chain capability。Serving 不创建/升级表，没有旧环境迁移路径：

```powershell
go run ./cmd/store-center-schema-init -config C:/private/store-owner.json -observations -walmart-us
go run ./cmd/product-acquisition-init -config C:/private/product-init.json -confirm-empty-database <exact-empty-product-database>
go run ./cmd/current-application -config C:/private/current-application.json
```

Console 沿现有身份配置启动，`LISTINGKIT_SERVICE_API_BASE` 指向该应用的 `/api/v1`，`LISTINGKIT_PUBLIC_BASE_URL` 与授权回调 origin 一致：

```powershell
cd web/listingkit-ui
pnpm exec next dev --hostname 127.0.0.1 --port <console-port>
```

真实 OAuth 需要获批的 HTTPS 回调；上述 loopback 开发命令本身不满足平台回调或对外开放条件。已有 [current application 说明](current-application-run1.md) 只用于理解启动合同，不授权扩展其保留实例。

## 数据与限制

- Store DB：原 Store/member/service/connection，加密 Walmart attempt/grant；永久商家占位沿原 merchant binding。断开不释放身份、删除 grant 或撤销平台授权；平台端撤销由商家自行处理。
- Product DB：原 Target、Operation、ExecutionKernel intent/claim/receipt；Walmart SKU 永久占位和 append-only Feed observations。SKU namespace 使用保留的随机 canonical Store grant ID，刷新、重授权和换 app 不更换。
- 准确载荷/schema 以字节保存，避免 JSONB 重写已批准 wire；schema hash 使用规范 JSON。Feed 受理与 kernel succeeded 不能代表已发布。
- 本期每 Feed 最多 1,000 SKU、2 MiB wire；Target/规则仍受现有文档大小限制。店铺商品采用官方 offset，每页 10 条，最多 1,000 页；超过预算保留部分覆盖。
- 不开放自动调价/库存同步、订单、WFS、自动目录匹配 fallback、GTIN 豁免申请或身份覆盖。服务端拒绝 `GTIN/CUSTOM`、自动调价/WFS/标识更新触发字段。
- 本地测试使用合成平台 transport、既有应用/页面测试和隔离 PostgreSQL；没有真实 OAuth、真实店铺或真实刊登结果，用户试用与生产验收 NOT_RUN。

## 必要验证

本批覆盖：原成员/企业与商家确认、回调单次 claim、refresh/重授权围栏、跨 app/跨企业及断开后的永久占位、两种 Target 构建和准确 wire、正常 Operation/worker 消费、SKU 失败子集分次修复与旧查询 fencing、原生商品观察与未知库存、Feed accepted/ingestion/publication 分层，以及回调/BFF/编辑器组件。候选 HEAD、独立检查及准确 CI 见 PR，不把历史设计 CI 或另一 AI 检查写成产品验收。
