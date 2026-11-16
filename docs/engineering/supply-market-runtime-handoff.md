# 供应市场 v1 正常程序接线

Issue #622 / PR #625。设计依据为已冻结的
[供应市场 v1](../architecture/supply-market.md)。本接线使用现有
`cmd/current-application`，Product/Collection、Asset、Submission 和 Temporal owner。

## 新安装与配置

只对预先创建、明确确认的空 Product 数据库执行已有初始化命令。初始化私密 JSON
沿现有 `schemaVersion: 1` / `database` 格式，追加以下开关；已有供应链实例同时保留
`supplyChain: true`。现有数据库会被拒绝，不提供修复或旧数据迁移路径。

```json
{"collections": true, "supplyMarket": true, "pod": true}
```

```powershell
go run ./cmd/product-acquisition-init -config C:/private/product-init.json -confirm-empty-database product
go run ./cmd/supply-asset-init -dsn-file C:/private/asset-owner.dsn -confirm-database assets -install-empty-schema
```

部署 owner 预先配置 `source_acquisition_runtime` 和 `supply_asset_runtime` 窄角色。
Product initializer 安装 Collection 的 market/sds_template/sds_finished 约束、市场、
当前 Review 读取所需 schema，以及 POD/Submission schema；授权不包含市场/Collection
删除或 Review 写入。POD fence 仅因精确终态释放和行锁需要 DELETE/UPDATE。
Asset initializer 复用当前批准图案 schema，不新建第二 Asset owner。

当前程序的私密 manifest 继续使用已有 identity、Product 和其他实际启用的 owner 配置，
并设 `productCollections: true`。新增部分示例（占位值须由部署 owner 替换）：

```json
{
  "supplyMarket": {
    "storage": {
      "region": "REGION",
      "bucket": "PRIVATE_QUALIFICATION_BUCKET",
      "accessKeyId": "PRIVATE_KEY_ID",
      "secretAccessKey": "PRIVATE_KEY",
      "mode": "aws"
    }
  },
  "pod": {
    "assetDatabase": {
      "host": "127.0.0.1", "port": 5432,
      "database": "assets", "user": "supply_asset_runtime",
      "password": "PRIVATE_PASSWORD", "maxConnections": 4
    },
    "temporalAddress": "127.0.0.1:7233",
    "temporalNamespace": "default",
    "credentialFile": "C:/private/platform-account.private.json",
    "ossHosts": ["QUALIFIED_OSS_HOST"]
  }
}
```

资质 bucket 必须为私有 immutable S3/COS 存储，无 PublicBase。COS 模式还必须明确
设置现有 `cosImmutableNonVersionedBucketPolicy: true`。S3-compatible 测试服务可使用
`endpoint` 指向本机或私有网络；外部 endpoint 必须 HTTPS。不得使用公开商品图片桶
分发资质资料。普通启动不创建 bucket、schema、角色或业务记录。

POD 的 Asset 目标须与已启用 Image Agent、Product Agent、供应链的 canonical Asset
目标相同；已有供应链时复用其窄 Asset pool 和相同 Temporal client。其他 owner
数据库不得与 Asset 共用。`ossHosts` 必须来自已经完成的实际协议 qualification，
不能填写任意上传域名。省略 `pod` 时市场及人工申请可独立使用，SDS 定制保持关闭。

凭据文件使用当前 `sds-platform-session-v1` 格式及 UUID revision，仅服务器读取。
程序使用固定 HTTPS 商户接口核实当前凭据的实际 merchant ID，拒绝 redirects、cookie
jar、重复 JSON 字段、越界/过期/不可读凭据及身份不符；不自动登录或刷新，不读取旧
Redis/auth_state fallback。文件必须为当前用户私密文件，Windows 校验 ACL。

```powershell
go run ./cmd/current-application -config C:/private/current-application.json
```

正常生命周期启动 `product-pod-current` worker，停止 worker 后才关闭 Temporal/DB。
固定 workflow ID 为 `pod-design/<operation-id>`，恢复沿已有 original intent、
SendPermit/attempt、merchant/template fence 及 read-only observation，不重复发送。
启动依赖或 schema/权限验证失败时服务拒绝启动，并清理已打开资源。

## 前端与使用入口

前端连接同一正常 API，设置 `LISTINGKIT_SERVICE_API_BASE=<API origin>/api/v1`。
实际后台已准入后，启用 `LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED=true`、
`LISTINGKIT_SUPPLY_MARKET_ENABLED=true`；仅在 POD account、Asset、Temporal 和已核定
OSS hosts 都完成配置后才设置 `LISTINGKIT_SDS_POD_ENABLED=true`。沿既有 HTTPS/登录
配置启动 UI；这些变量不代替后端依赖检查或权限。

企业成员登录并选择企业后可进入：

- `/workbench/supply/official`、`/selected`：查看真实已发布商品，选择并保存到本人
  Collection。空市场显示空状态，不 seed 假发布。
- `/workbench/supply/catalogs`：按 API / 浏览器插件展示货盘目录，可按国内仓、海外仓和
  仓库所在国家/地区筛选；页首「申请对接货盘」在当前页面打开模态框，与原申请页
  复用同一表单及提交/核实能力，取消返回原筛选结果。提交进行中禁止关闭；成功后
  显示准确记录的进度链接。原独立申请页与进度入口保留，目录不再提供「进入货盘选品」。
  侧栏只显示「供应市场 → 货盘集成」两级；既有选品、申请或 SDS 深链接仍高亮该入口。
- `/workbench/supply/applications`：本人申请、资料补充和进度。优选申请继续要求
  本人自有商品及实际 Apply lineage。
- `/workbench/supply/catalogs/sds`：选模板/款式和本人已批准图案，配置设计；核实
  原 operation 的效果图和已保存成品后再选择入本人 Collection。

市场和已核实 SDS 成品批次可以沿既有“加入我的供应链”转入准备流程，保留原
source kind、publication 和 operation；未定制 SDS 模板须先生成并核实成品。
混合手动批次包含未定制模板时，转入会明确拒绝并回滚，先将成品单独分组。
新安装的 preparation schema 只接收 acquisition/own/market/sds_finished；启动
检查拒绝仍只有 acquisition/own 的旧约束，不在运行期修改既有数据库。

首次发送前的临时读取失败仅在原 Kernel 确认 OSS attempt 不存在时，在同一
固定工作流内等待重查，最长15分钟。已有 attempt 或读取结果不明确时不重试。
前置等待超时/工作流关闭后显示 UNKNOWN，保留原操作和 fence，不新建工作流。
固定工作流总超时35分钟，覆盖前置重查和完整核实窗口；单项activity的排队时间
计入其3分钟/30秒上限，不扩大任何已发送操作的重试权限。

平台专员通过 `/workbench/supply/operator` 及原申请 UUID 详情执行人工评估、发布。
该入口独立使用现有 server session 身份，后台每次核对 verified global platform
权限，不要求成为客户企业成员。导航仅在原平台读取成功后展示专员入口；企业
角色/自定义模块、浏览器 body/header 不能授予平台权限。成员页面仍保留企业 gate。
平台路由直接消费已验证的全局身份，并由路由与领域分别核实平台权限；不绑定要求
企业上下文的商品读取 capability。成员路由继续使用该 capability，不能从平台身份
推导任何客户成员或商品权限。实例配置中的已有平台授权无需为进入自营页面重复添加。

### 自营商品添加与发布

具备全局平台权限及当前企业自有商品查看/管理、供货申请权限的专员，从「硕米自营」页点击
**添加自营商品**（`/workbench/supply/official/new`），选择本人已有商品，或点击
**新建商品**复用完整自有商品录入页。新建保存成功后，使用回执中的准确 item ID
回到供货信息步骤；普通自有商品录入入口仍返回自有商品库。

填写库存/产能、起订量、供货周期、发货省市、价格/售后说明并确认披露后，点击
**保存草稿并预览**。草稿保存后直接进入原平台记录详情，核对确切版本的图片、
商品资料、SKU价格和供货信息，再点击 **确认发布**。保存草稿不会自动公开；
创建和发布须由同一账号完成，服务端原 LiveWrite 与平台授权分别校验。

稍后可从「硕米自营 → 管理自营商品」（`/workbench/supply/operator?kind=official`）
继续预览/发布。管理页从实际 release 显示草稿、已发布或已撤销，不能把原记录的
`DRAFT` 阶段直接当成未发布。记录绑定不可变商品版本；商品资料更新后需建立
新的发布草稿，已有草稿和已发布内容不自动跟随。读取发布状态失败时不能再次发布。

结果待确认时继续使用原操作的核实/恢复入口；只有拿到确切保存回执才进入下一步，
不因超时新建另一件商品或草稿。前端平台权限预检复用原专员只读 API，仅用于入口
与及时反馈，不代替各写命令的真实权限验证。本流程无模型调用或外部平台刊登。

所有事实保存在现有 Product/Asset 数据库，资质 bytes 留在私有对象存储。保留命名
volume/数据库并按原配置 stop/start；不执行 `down -v`。UNKNOWN/UNCONFIRMED 保存
原 key/body/account revision 和 fence，先核实原操作；不另起操作覆盖或重发。

### 货盘目录资料与 Logo

目录是公开资料展示，不是连接器、接入回执或实时库存。四家当前均未接入硕米 API，
硕米插件也尚无这四家的适配器；平台公开提供 API 不代表硕米已经接通。
国内仓按中国仓解释；仓库类型与地区必须匹配同一条仓库资料，切换类型时清除不适用
的地区。已核实国家集合仅为公开覆盖的子集，其他地区待确认，不据此推断无仓或有货。

资料核对日期：2026-11-16。名称、API 说明及仓库资料来源：

| 货盘 | API 资料 | 当前展示的仓库资料 |
| --- | --- | --- |
| 大健云仓 | [技术服务](https://www.gigacloudtech.cn/tech/) | [官方介绍](https://www.gigacloudtech.com/about)：美国、加拿大、英国、德国、日本 |
| Doba | [Retailer API](https://open.doba.com/) | [官方供应商介绍](https://www.doba.com/high-quality-suppliers)：美国 |
| SaleYee | [官方 API 接入说明](https://global.saleyee.com/help/shipment.html) | [官网仓库导航](https://www.saleyee.com/)：美国、英国、德国、法国、捷克、西班牙、加拿大、日本、澳大利亚、墨西哥、沙特阿拉伯 |
| CJdropshipping | [开放 API](https://developers.cjdropshipping.com/en/api/start/apikey.html) | [官方仓库介绍](https://elites.cjdropshipping.com/article-detail?id=5747)：中国、美国、英国、德国、西班牙 |

CJ 官方资料对波兰仓开放状态存在不同表述，本次不纳入已核实地区。目录不改变原有
1688/SDS 的运行开关、权限、申请、保存或恢复路径。

Logo 原样保存于 `web/listingkit-ui/public/brands/catalogs/`，避免页面依赖第三方图片
热链。商标权利归各平台所有；以下为本次取得的官方资源地址：

| 文件 | 官方资源 |
| --- | --- |
| `gigacloud.png` | https://www.gigacloudtech.cn/_nuxt/logo.Bg5iPk6P.png |
| `doba.svg` | https://ds.micstatic.com/london/_next/static/images/logo-4f93dc67c2e6de0bf8db184f586c12d3.svg |
| `saleyee.png` | https://www.saleyee.com/Content/Images/logo.png |
| `cj.svg` | https://frontend.cjdropshipping.com/config-resource/cj/new_logo.svg |

## 当前证据与交付边界

本轮有受限角色的 disposable PostgreSQL 空库安装、市场/POD HTTP composition、
精确 route admission、额外删除权限拒绝、凭据身份校验、运行资源清理及 UI/BFF
开发自检。这些是接线检查，不是用户验收。

本轮未改变保留中的 31544 实例/数据卷，未追加真实 SDS 素材/成品写入，也未重试
旧未确认 operation。正常实际 runtime、真实浏览器1688/SDS 路径和用户试用仍为
`NOT_RUN`；实际环境的私有资质存储、OSS qualification host 及外部依赖须按部署
owner 配置后再试用。本 PR 不授权合并、共享/生产部署、Issue 关闭或真实数据操作。
