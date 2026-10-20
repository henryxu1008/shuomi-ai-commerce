# own675 最小配置差异（未生效）

本表供后续私密副本审阅，不是 JSON patch 或完整运行 manifest。真实应用仍缺失，不能据此启用 Store/供应链。此处“当前”沿 [安装方案](../own675-shein-import-trial-installation-v1.md) 的已核对快照；本轮未读取实例或私密文件，执行前必须刷新并逐项比对。

## 原后台 manifest → 后续私有副本

| 路径 | 当前快照 | 后续最小值 / 条件 |
| --- | --- | --- |
| `storeCenter.enabled` | `false` | 待真实应用齐备及另行授权后 `true` |
| `storeCenter.database` | 原 `store_center_runtime` 连接 | 原样保留，不换维护身份 |
| `storeCenter.officialApplications` | 未配置 | 仅一个真实 SHEIN 应用；以下字段齐备前不填写占位对象 |
| `supplyChain.assetDatabase.host` / `port` | 无 `supplyChain` | `127.0.0.1` / `5433`，均为原 network namespace |
| `supplyChain.assetDatabase.user` / `database` | 无 | `supply_asset_runtime` / `own675_supply_assets` |
| `supplyChain.assetDatabase.password` | 无 | 后续新 runtime 私密密码，仅存在私有 manifest；此合同没有 `passwordFile` 字段 |
| `supplyChain.assetDatabase.maxConnections` | 无 | `4`（合同上限 `8`） |
| `supplyChain.temporalAddress` / `temporalNamespace` | 无 | `127.0.0.1:7233` / `default` |

真实应用需准备的键：`platform: shein`、实际 `type`、`appId`、本地修订 `version`、匹配真实应用的 `apiOrigin`、平台认可的 `callbackURL`、`appSecretFile`、`credentialKeyFile`、`credentialKeyId`。本包不给虚构 appId/secret，不预设应用模式或注册地址被允许；完整准备条件沿原安装方案。

原 identity、tenant directory、Product/商业/Store/容量/图片配置、来源实例 pin 等原样保留，原私有 manifest 及其 ACL 不改。不要用本表覆盖整份 JSON 或删除未知旧字段。后台原 `supplyCompletionDrafts=true` 保留，不追加 Observations、Cockpit、TikTok、Walmart、Image/AI Agent 或 data-services。

## 私密挂载与启动文件

| 后续新增项 | 最小边界 |
| --- | --- |
| App Secret | 单独普通私有文件，只读挂入后台的绝对路径；来自真实平台，不在仓库/镜像/前端 |
| 商家凭据加密 key | 与 App Secret 分开，本地 32 字节随机值以标准 Base64 保存，稳定 keyId；仅后续授权生成并私密备份 |
| Linux 文件权限 | 运行 uid 可读且 group/other 无权限；核实容器内实际 mode，不能假定 Compose 默认 `0444` 合格 |
| 新 Asset runtime 密码 | 仅后台私有 manifest；owner DSN/原 Product 维护配置不挂 serving |
| 后台启动参数 | 后续指向私有副本；不覆盖原文件，本轮不改变启动参数 |
| start helper | 后续仅追加经批准的独立 Temporal overlay/等待顺序；本轮不编辑或注册模板 |

## Compose / 前端 / 镜像

| 项目 | 最小变化 |
| --- | --- |
| Temporal | 见非生效片段；一项专属服务、一次新卷目录权限准备、一卷，无新宿主端口 |
| 前端环境 | 后续只将 `LISTINGKIT_SUPPLY_CHAIN_ENABLED=false` 改为 `true`；CompletionDraft、Collections 及其他原开关保留 |
| 后台 / 前端镜像 | 后续在相应更新授权下使用同一准确候选；当前原 `23a301245` 镜像及容器不替换 |
| 原 UI / login | `32768` 请求头上限、原 CA/TLS/登录 origin 与登录回调保持 |
| 图片与容量 | 原对象、卷、URL、容量账本与 sourceMedia 配置保持，不开私网图片绕过、不上传图片 |

原 Store 两表四项 CHECK / Goose `2026101601` 已完成；本包不再提供 Store 改库步骤。Product 13 表及最小 ACL、新 Asset 库/角色/4 表、Temporal 服务/卷、真实应用与上述接线是不同执行阶段，均未在本轮执行。尚未齐备时原运行组合继续不变。
