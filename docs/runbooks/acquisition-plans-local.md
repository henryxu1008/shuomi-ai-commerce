# 1688 采集计划本地试用

依据：`docs/architecture/1688-acquisition-figma-v1.md`。

在当前账户中心组合上追加
`deployments/docker/account-compose/docker-compose.acquisition-plans.yml`。
此层要求原有 base、knowledge、data-services、modules、ecoservices、chat 六层，
以及对应的现有私密配置和 named volumes。启动/重启必须保留全部层与卷。

显式 `acquisition-plans-init` 使用现有 Product schema owner，只创建当前计划、规则和
Preparation 空表，并追加窄权限。已有商品、采集回执、资源、账号和企业事实不被重置。
它不启用适配/审核/上传执行表，不读取历史事实做迁移。serving 不执行 DDL。

独立执行入口（私密 manifest 按现有 product-acquisition-init 格式，
`collections:true, acquisitionPlans:true`；必须填写确切专用数据库）：

```text
product-acquisition-init -config /absolute/private/manifest.json -enable-acquisition-plans -confirm-database product_acquisition
```

collector 使用仓库已有 Playwright Go 与固定模块所需的 Chromium 版本。
它在独立容器网络命名空间内运行，无数据库凭据、无 host port，不与 runtime-network
共用 loopback。当前数据库只监听 runtime-network 的 loopback，collector 不能访问。
调用凭据在私密卷生成，浏览器和 HTTP RPC 共享同一个 client 的并发、频率与冷却控制。
首次启动沿现有 10 分钟 quarantine；不为试用取消限制。列表发现不会解验证码。

前端只有 `LISTINGKIT_ACQUISITION_PLANS_ENABLED=true` 才开放计划界面；后端
`acquisitionPlans=true` 必须与 Collections、schema、权限和 collector 配置同时满足。
Preparation-only 开放“我的供应链 / 待适配”和来源查看，完整执行能力仍单独准入。

入口：`/workbench/supply/acquisition`。指定商品支持多行 offer ID 或公开商品链接；
按页接受公开搜索结果或店铺商品列表，按关键词读取公开搜索结果。最多 20 行、
200 个去重商品，只读当前页可见内容，不自动翻页或使用登录态。

规则取封存来源的明确 CNY 公开起售价和最小起订量，AND、含边界；未知事实保留
“我的数据”。成功商品沿原计费消费 1 DATA_ROW，规则不免单。自动转入只选规则
通过的原成员/版本；没有匹配项不转入任何商品。

离开页面停止推进。恢复先读原计划，再用原 child/transfer key 继续。
未知创建和模板操作保留原请求，不能换 key 猜测重发。

页面/接口检查和本地 fixtures 不等于真实 1688 验收。真实公开来源可用性、实际资源
扣费和用户试用需另行确认；未经实际执行保留 `NOT_RUN`。
