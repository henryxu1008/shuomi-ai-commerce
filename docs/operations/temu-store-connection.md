# TEMU 店铺连接配置与交接

执行 [#31](https://github.com/qq550723504/task-processor/issues/31)，依据已冻结的
[TEMU 架构](../architecture/temu-platform-v1.md) 与
[官方接口证据](../architecture/temu-official-contract-evidence.md)。本页说明当前实现的店铺连接与共享规则读取切片。
完整交付范围仍包括规则、资料、提交、恢复与商品观察；当前切片不代表这些能力已经完成。

## 配置

沿用 [Current Application 启动与 Store 权限配置](member-resources-v1.md)。只在全新安装的测试实例中
初始化当前 Store schema；已有实例不会被自动迁移，不能用本切片覆盖保留的试用数据库。
私有 manifest 的 `storeCenter.officialApplications` 可配置以下条目：

```json
{
  "platform": "temu",
  "identityPartition": "CN",
  "appId": "approved-temu-application-id",
  "version": "configuration-version-1",
  "appSecretFile": "C:\\private\\temu-cn-app-secret",
  "credentialKeyFile": "C:\\private\\temu-cn-merchant-key",
  "credentialKeyId": "temu-cn-key-1"
}
```

`identityPartition` 只允许 `CN`、`US`、`EU`、`GLOBAL`，取原官方应用的注册分区。
CN 一个应用支持核验全托管与半托管，经营模式由官方 `semiManagedMall` 明确判断。
其他三个分区只接受官方 `mallType=100` 的本土店铺。身份分区不是国家/站点选择器。
不填写 `type`、`apiOrigin`、`callbackURL`；TEMU 路由固定，不通过配置自定义，也不宣称已开放回调授权。
应用密钥与独立 Base64 32 字节 AES-GCM 密钥保存在两个绝对路径的私有普通文件中。
每项加密 key ID 和文件必须独立；跨平台/分区可以具有相同原始 appId，完整应用身份不能重复。
当前应用总上限仍为 16。配置版本、协议、分区或应用移除后，旧连接失去可用性，需从原 Store 重新连接。

## 运营人员操作

1. 登录，选择企业，在 `/workbench/stores` 的“绑定新店铺”选择 TEMU 并创建店铺记录。
   记录的区域、外部 ID 是元数据，不作为官方商家身份或路由依据。
2. 在该店铺“官方 API 授权”选择原应用分区，点击“开始连接”。缺少配置时保持不可用。
3. 店铺主账号在 TEMU 官方授权管理生成 token，将其填入密码输入框并点击“验证并连接”。
   浏览器不保存 token；提交、退出输入框所在上下文或卸载组件时清空。请求绑定当前用户、企业与原 attempt。
4. CN 用同一应用/token 先读取 PA 的 `bg.open.accesstoken.info.get.global`，再读取 CN 的
   `bg.mall.info.get`；只有两项均有效才连接。US/EU/GLOBAL 只读原分区的
   `bg.open.accesstoken.info.get`。整个核验共用五秒 deadline，不跟随重定向、不改用其他凭据或网关。
5. 失败后若密文已保存，使用“核验原连接”重读原输入；同一 attempt 不接受不同 token。
   需要替换输入时使用“重新连接”，这会使旧 attempt 失效。当前成员资格、Store grant 和应用 revision 持续受检。
6. “断开本地连接”取消本地可用性；远端授权由店铺主账号在 TEMU 官方撤销。
   永久商家占用仍属于原企业/Store，不能通过断开、更换应用/token/模式或新建店铺转移。

浏览器仅显示核验的经营模式、区域、到期时间和身份摘要，不显示官方商家 ID、token、app secret 或密文。
商品/订单数字在未取得观察时显示未知，TEMU 同步按钮尚未开放。
连接与服务开通不等于获得发布许可，也不等于全部站点已可发布。
GE/AM 缺官方规格的发布开放和全覆盖交付门槛仍保留。

## 共享规则查询的当前合同

启用当前 SupplyChain 组合后，原 backend `POST /api/v1/workbench/supply-preparations/target-rules`
支持以下具名 TEMU 查询；需要正常认证上下文、有效企业/成员、保留 source 的读取权限，以及目标店铺
原生 grant、active service、当前已核实连接和各个具名官方 API scope。请求中不传凭据或商家 ID。

```json
{
  "sourceId": "当前保留 source 的 UUID",
  "storeId": "目标 Store 的 UUID",
  "effectiveVersion": "1",
  "temuRules": {
    "parentCatId": 12,
    "catId": 34,
    "language": "en",
    "costTemplateId": "从本店当前运费模板列表选择的 ID"
  }
}
```

上面只是格式示例，类目、版本和模板均须取当前实际资料。经营模式与本土 region/site 由 Store 官方绑定确定；
本土不能传 `siteId`/`mallId`。全托/半托查询不传本土 language/costTemplateId，半托类别查询可显式选择
`supplierSiteId`；完整96站点字典不是当次商家开站或仓库资格。商标列表通过显式
`trademarkPage`/`trademarkSize` 查询一个有界页（size ≤ 100），不声称其他页已读或已取得品牌许可。

返回的 `temu` 是当前类别、属性/规格、所选模板、站点格式、来源/解码内容 hash 与观测时间。
不可选择或未观察到的末级类别不会查询依赖属性；本土属性只在本店当前列表含所选运费模板时读取，
否则保留 `selected_freight_unverified`。原连接改变、读取失败、源权限撤销或整个快照超过2 MiB 时拒绝结果；
整体最多10秒，Store 私密句柄仍只存活5秒。GE/AM 缺规格保留在 `localTarget.issues`。
这些元数据不签发整体 readiness；供应商实际币种、限价、资质等也不由字典推定。
当前 ProductSnapshot 的 Price/Dimensions 使用浮点数、VariantStock 没有缺失标志，不能直接把它们转换成
TEMU 精确价格/单位或已明确库存。后续资料合同必须保留明确输入/来源与批准，不能以数值默填补齐；
本切片没有改动 Product canonical schema。

当前 Console 编辑器/BFF 和受控 Tools 尚未接入这个 TEMU variant；不要把 backend 接线写成浏览器端可发布。
官方 LONG 在 Go/原 JSON 中保留精确整数，后续浏览器 DTO 必须采用字符串 ID/无损读取，不能经过 JavaScript
浮点数再批准。`temuRules` 仅用于读取，不能与 SHEIN draft 混传；原保存目标入口会拒绝该查询参数，
避免落入 SHEIN builder。TEMU typed TargetRecord、图片/商品提交和恢复仍待完成。

## 当前验证边界

开发验证使用固定官方 origin 的合成 HTTP transport、隔离 SQLite 和一次性 PostgreSQL 容器；
覆盖签名、模式、身份、私有输入重放、AAD、成员变更、跨企业占用以及并发事务。
没有调用真实商家 API，没有部署或更改保留实例。
真实应用权限、真实 token、店铺主账号操作和用户验收均为 `NOT_RUN`。
