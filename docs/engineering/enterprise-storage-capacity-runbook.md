# 企业存储容量 V1 试用入口与启动条件

设计依据：[企业存储容量 V1](../architecture/enterprise-storage-capacity-v1.md)。
容量启用适用于新安装和空的计量 namespace；不迁移、回填或清理旧文件。

## 保留实例、容量关闭时的必要 schema 补齐

最新程序的 `storageCapacity: false` 守卫仍要求原 Commercial owner 可读取六张容量表且没有容量事实或购买订单。保留实例缺少这些表或现四张商业表的新字段时，先按当前用户授权执行下面的有界入口；容量功能继续关闭，原文件不登记为计量 namespace。

```sh
go run ./cmd/commercial-owner-schema-migrate -config <原commercial-schema-owner私密JSON绝对路径> -storage-disabled-additive-only
```

入口只消费这个原 Commercial schema-owner config；`-money-config`（包括空值）和位置参数会被拒绝。它在一个有界 PostgreSQL 事务内使用原模型安装六张 `storage_capacity_*` 表、给现有 `commercial_offers` / `commercial_quotes` / `commercial_order_items` 各增加 `storage_unit_bytes` / `storage_duration_seconds`，给 `commercial_orders` 增加 `storage_activation_receipt`，并只给原 `commercial_owner_runtime` 六表 SELECT。六个整数列的 NOT NULL DEFAULT 0、回执列的 varchar(128) NOT NULL DEFAULT '' 来自当前正式 owner 定义。它不初始化 Money、OrgResource 或其它商业表，不安装或改写 offer/价格、原业务事实、角色、既有授权或凭据，也不修改 runtime manifest/worker 配置。

六表须全部缺失，或已是完整、匹配当前模型且为空的安装；部分/不同形状、已有任何容量事实/购买订单均拒绝。已有七列只检查，不改写其定义。若原 role 已有容量表级、列级或经默认授权/继承得到的写权限，入口拒绝并整批回滚，不自行撤销原权限。`public` 是固定 schema，不使用同名 shadow 表证明为空。重复执行核实原完整安装，不追加订单、offer 或 namespace。锁等待最多5秒、单语句30秒、整次执行1分钟；失败保留原程序和库，按原配置核实，不扩大初始化范围。

当前六表 owner 只定义主键及命名索引，已有六表出现额外 CHECK、外键、唯一或排除等约束、或非 owner 索引时拒绝。商业检查只针对七个新增字段涉及的约束和索引（包含独立/表达式索引及排除约束的索引依赖）；原商业表上无关的合法约束和索引保留。拒绝时不删除、改写约束、索引或原记录，整事务的 DDL/新增 SELECT 一并回滚。

由原安装 owner 在已获准安装窗口使用私密备份和原事实/ACL/卷/worker基线执行；新 DDL/授权与程序合并仍需各自明确人类授权。正常启动沿既有 runtime identity 的 `VerifyStorageCapacityDisabled` 核实六表和订单；确认候选程序操作入口、原事实与 worker 后交接。本入口不执行新程序部署，也不签发用户验收。

事务失败回滚本次 DDL/新增 SELECT。事务提交应答不明时先核实原 schema/ACL，可用同一入口对完整匹配安装重读；不换空库或执行全量初始化。已提交而候选程序失败时，在确认仍无容量事实/购买订单后，仅恢复原 backend7058/UI9c 镜像和私密配置，保留新增空 schema、原数据、命名 volumes 与 worker。出现容量事实后不得切回未计量程序，不删除表/列、恢复旧备份覆盖新数据或用 false 绕过；交原 owner 沿当前恢复合同处理。

## 容量功能的新安装启用

沿项目现有 `commercial-owner-schema-migrate` 命令初始化私有 schema-owner
配置；该命令同时安装容量表、runtime grants 和初始扩容 offer，重复运行不覆盖
后续 owner 管理的价格或禁用状态。普通应用启动不建表、不安装价格。

在 current-application 私有 manifest 中显式设置 `"storageCapacity": true`，
并提供原 CommercialOwnerDatabase、MoneyOwnerDatabase 和已启用文件能力配置。
图片 worker 的原 YAML 同时设置 `storageCapacity: true`，使用当前企业图片
协议及同一 commercial owner。未配置时页面显示容量服务不可用，不能购买容量。

第一次启动逐一验证计量 prefix 为空并登记原 namespace：SourceMedia 的
`product-source/`、Knowledge 的 `knowledge/`、图片的 published prefix。
生态服务的 `ecoservices/files/` 同时登记 namespace；REQUEST 需求附件计量，
APPLICATION 入驻认证材料和供货市场资格材料按原证据合同排除。
AWS 曾启用 versioning 的 bucket 还会检查非当前版本和 delete markers。
相应只读 `ListBucket`、`GetBucketVersioning`、版本列表权限须覆盖这些 prefix；
启用还需只读 `GetBucketAcl`：每次启动用供应商返回的 `Owner.ID` 绑定物理账号。
账号变更重新验证新 namespace 为空；凭据轮换保持同一 owner 的原登记。
owner 缺失或 IAM 失败时启动拒绝，不退回旧四元组身份，也不修改 bucket ACL。
应用与图片 worker 的 commercial owner host/port/database 必须一致。
COS 沿既有严格 non-versioned immutable policy。成功登记后的重启使用原登记，
不把已保存文件当成新安装，也不扫描 bucket 重算用量。不得与旧的未计量 Writer
同时启用；保留共享实例和现有数据，试用使用独立新 profile。

该 owner 一旦保留容量 namespace、占用/预占、容量包或原容量购买订单，
不能通过设置 `storageCapacity: false` 回退。API 和企业图片 worker 会在启动前
复用原 owner 的只读检查拒绝该配置；缺表、权限、连接或检查失败也不视为空。
恢复时保留原商业 owner 配置和数据，重新使用 `storageCapacity: true` 启动，
原订单及 UNKNOWN 沿既有入口继续核实。不删除账本、换空库或清理对象来绕过。

沿现有命令 `go run ./cmd/current-application -config <私有manifest绝对路径>`
启动 backend，沿现有 UI 启动方式设置 Commercial API origin 并启动
`web/listingkit-ui`。正常入口为 `/workbench/account/organization/resources`，
当前企业且具有 CommercialRead 的身份可读取容量；CommercialPurchase 可获取
原报价并确认扩容，余额不足沿原 `/workbench/plans/top-up` 充值。

每企业永久 5 GiB；每包 50 GiB、19.90 元、生效后 30×24 小时，购买分别到期。
页面显示真实已占用、预占、可用量、分类与最先到期的最多 50 笔扩容。
更多购买记录沿原账单与订单分页查看。网络失败时恢复原报价、原请求 key 或原订单，
不能新建一笔购买来替代未知结果。到期/超限保留已有数据及原有限预占。

当前删除能力沿原业务 owner；未开放 destructive delete 的分类首版只能扩容，
历史版本、停用文档和已保存的未使用候选仍计量。未知写入按原业务请求或原 worker
恢复；无可靠关闭证明的预占会继续显示，不按超时清零。容量聚合不暴露文件名/key，
也不改变原文件访问权限。

开发自检与受控 provider/数据库测试属于实现证据。真实 provider/支付、正式环境
IAM、部署、浏览器业务试用和用户验收须按各自授权执行，不能写成已完成。
