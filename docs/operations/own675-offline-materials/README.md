# own675 离线安装材料

状态：`PREPARED_ONLY / NOT_AUTHORIZED_TO_EXECUTE`。本目录交付参数模板、非生效 Compose、配置差异和失败处置；不包含凭据、二进制或镜像归档，不是已完成安装。目标仅为 `task-processor-own675-20261014-c1e22342`。

用户 2026-10-16 批准制作材料，未批准安装、创建角色/卷、启动服务、替换镜像/配置、业务保存或真实平台调用；不自动发布。现有数据、图片、配置和授权保留。真实 SHEIN 应用仍未提供，原 Store/供应链开关保持关闭。

Design Basis: Reuse Existing Architecture，沿 [现有安装方案](../own675-shein-import-trial-installation-v1.md)、Product/Asset 原 owner 和原 Temporal 接线，不新增安装器、runner、事实 owner 或恢复协议。接单/准入、准确 HEAD、检查证据及后续授权维护在 [Issue #735](https://github.com/qq550723504/task-processor/issues/735) / [PR #736](https://github.com/qq550723504/task-processor/pull/736)。

## 材料入口

| 文件 | 内容 / 消费者 |
| --- | --- |
| [product-owner.json.example](product-owner.json.example) | 现有 `product-acquisition-init` 的私有维护 manifest 格式 |
| [asset-owner.dsn.example](asset-owner.dsn.example) | 现有 `supply-asset-init` 的私有 owner DSN 格式 |
| [compose.supply-temporal.yml.example](compose.supply-temporal.yml.example) | 只包含专属 Temporal、卷权限准备及专属卷的片段 |
| [configuration-diff.md](configuration-diff.md) | 后续安装、文件挂载和前后端接线的最小差异 |
| [failure-handling.md](failure-handling.md) | 首次失败、提交不确定、重入与停用的准确边界 |

全部 `.example` 只供审阅，不被原启动器、bootstrap、schema-init 或默认 Compose 引用。模板中的 `__PRIVATE_...__` 是未提供值的标记，不能交给安装器；不得填写虚构 SHEIN 应用使启动校验通过。原私密文件不复制进本目录。

## 候选和执行位置

Product 13 表入口及 Store 核心入口的已验证代码基线为 `deaf7453bddd74d6fa1deece043d3832a5e51720`。本轮已同步 main `68b4e99a4cb6afd95deea7ed95b9dad6b4e65edd`，该增量不改 Product/Asset 安装器或 Temporal 合同；最终材料提交见 PR。未来执行须重新绑定经批准的准确源码、构建命令、二进制 SHA256 和同候选应用镜像，不能用浮动分支名或本目录状态代替部署授权。

两个安装器均从仓库现有 `cmd/product-acquisition-init`、`cmd/supply-asset-init` 构建，不增加 wrapper。未来维护进程需位于 own675 的现有网络命名空间；模板中 `127.0.0.1:5433` 指该命名空间内的 business PostgreSQL，不是 Windows 宿主机直连参数。维护镜像必须具备 `Asia/Shanghai` zoneinfo，不能仅凭二进制 `--help` 成功认定可建连。此处不提供会启动 helper 的脚本。

未来私有副本需为绝对路径、普通文件，Linux 下仅维护 uid 可读（如 `0600`）；Windows 下沿现有私有 ACL 检查。维护文件不挂入 serving，不轮换原 Product/Store 密码。两个示例没有真实密码，也未通过真实建连验证。

## Product 参数模板（未执行）

```text
product-acquisition-init --config <私有product-owner.json绝对路径> --enable-supply-chain --confirm-database product_acquisition
```

- 维护身份沿原 `acquisition_owner`。只接受原 14 表及原 runtime 授权满足既有基线；首次新增既有 13 表和 12 表精确 grant，完整重入只读核实。原表、数据、ACL 和图片不改。
- 不使用 `--confirm-empty-database`，不组合其他 feature 开关，不调用全库重授权限。表清单和 S/I/U 精确矩阵沿 [现有安装方案](../own675-shein-import-trial-installation-v1.md#现有-product-数据库新增-13-张原-14-张及原授权保留)。
- Product grant 增量会使旧 Completion-only 配置不再通过运行权限校验；真实应用和完整配置未就绪时，不先执行该步再按原配置恢复应用。

## Asset 参数模板（仅新独立空库，未执行）

```text
supply-asset-init --dsn-file <私有asset-owner.dsn绝对路径> --confirm-database own675_supply_assets --install-empty-schema
```

前置：后续另获准的维护者创建新专属库 `own675_supply_assets` 及独立 `supply_asset_owner` / `supply_asset_runtime`，维护 owner 拥有该库和 public schema；不指向 Product、Store、系统库或其他已有业务库。新角色无 SUPERUSER/CREATEDB/CREATEROLE/REPLICATION/BYPASSRLS 或角色继承，runtime 不得拥有表；维护凭据不进入后台。安装器不创建数据库/角色，也不安装 Temporal。

| 新 Asset 表 | `supply_asset_runtime` 最小权限 |
| --- | --- |
| `product_approved_assets` | SELECT / INSERT |
| `product_approval_receipts` | SELECT / INSERT |
| `product_approved_inventory_heads` | SELECT / INSERT / UPDATE |
| `product_approved_inventory_version_heads` | SELECT / INSERT / UPDATE |

另仅该库 CONNECT、public USAGE；无 CREATE/TEMP、DELETE 或 DDL。不向其他数据库新增授权，也不修改原库 PUBLIC/角色的既有授权；本材料未验证其他库的有效权限，不把未显式授权写成已通过跨库隔离验收。

Asset 工具会在该新库内撤销 PUBLIC/runtime 的 database/schema 权限及 runtime 表权限，再按 owner 合同授予上述最小权限；这与 Product 的精确新增 delta 不同。只有新专属 Asset 库属于该维护边界，不能把它用于“修复”原库授权。成功后再次带空库 flag 会拒绝；无该 flag 也会写 ACL，详见失败处置。

## Temporal 片段的使用边界

片段复用仓库固定的官方镜像、`server start-dev`、原 `runtime-network` 和新专属卷；内部试用 namespace 为 `default`，地址为 `127.0.0.1:7233`，没有新增宿主端口。只准备两项服务，未复用整份 data-services/image-agent overlay，未连接其他实例。

文件名为 `.example`，两项服务还受非默认 profile `own675-supply-prepared` 约束。它必须与含现有 `runtime-network` 的原 Compose 文件组合；不能独立启动，也不添加到原 start helper。profile 只是防误启措施，不是执行授权；显式启用 profile 或指定服务仍能使 Compose 执行，因此本轮只允许静态 `config` 检查，不运行 `up/run/start/create/build/pull`。

后续若获准，先核实网络和端口无冲突、新卷确实不存在，再仅对新卷准备 `1000:1000` 目录权限，等待 Temporal healthy 后才接入同候选应用。若发现已有同名卷或服务，先核实归属和内容，不能覆盖、清空或递归 chown。停止服务与删除卷分开，不使用 `down -v`。

## 本轮检查与交接

检查限定为模板字段/参数与现有源码一致、Compose 静态组合、profile 关闭时不加入新服务、服务/端口/卷范围、相对链接和 diffcheck。沿用未改代码已绑定的安装器隔离测试，不新建验证平台或重复全量测试。

本轮不访问真实 SHEIN、不执行安装器、不创建测试数据库/容器/卷，不读取 own675 私密配置或修改原 start helper。检查通过只代表材料可审阅，不代表新 schema、运行健康、官方规则、Target R1→R2 或用户验收已完成。
