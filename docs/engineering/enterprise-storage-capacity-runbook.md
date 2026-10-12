# 企业存储容量 V1 试用入口与启动条件

设计依据：[企业存储容量 V1](../architecture/enterprise-storage-capacity-v1.md)。
本批适用于新安装和空的计量 namespace；不迁移、回填或清理旧文件。

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
COS 沿既有严格 non-versioned immutable policy。成功登记后的重启使用原登记，
不把已保存文件当成新安装，也不扫描 bucket 重算用量。不得与旧的未计量 Writer
同时启用；保留共享实例和现有数据，试用使用独立新 profile。

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
