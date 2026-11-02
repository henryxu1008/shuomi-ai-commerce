# 产品研究的运行交接

本批范围见 [冻结设计](../architecture/market-research-selection.md)、[#1072](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1072) 和 [PR #1075](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1075)。本文件交付正常接线与操作入口，不证明实例已启用。用户明确不授权合并、部署、真实平台或付费调用；本轮只使用隔离合成测试数据。真实采集、实际模型和用户试用为 NOT_RUN。

## 正常入口与可用路径

1. 正常登录自己的企业，从「AI 工作台 → 项目中心」创建类型为「选品分析」的 PRODUCT_RESEARCH 项目，打开 `/workbench/ai/projects/{projectID}`。
2. 项目目标下方的「研究选品」区域先打开原数据市场 `/workbench/data/market`。真正采集仍要求单独授权、原权限、原最大 DATA_ROW 用量确认及原执行通道；本轮未执行。只读页面不触发采集或模型。
3. 「关联已保存资料」复用我的数据批次选择器，仅允许自己当前可读的服务端 `amazon_data` 结果。展示站点/ASIN、原时间、parser、结构化证据 checksum、准确 publication/version、查询样本范围、已观察价格/币种与缺失项。无价为未知；不同币种不计算统一排名。来源撤权或不可读后遮蔽内容，保留可解除的私有关系槽。
4. 选择 2–5 个可读候选，「比较所选候选」进入原商品事实比较和显式 AI 配置/预算/确认流程。当前能力只比较已有 Catalog 事实，不宣称完成市场证据 AI 分析。模型不可用不阻止人工审阅与决定。真正市场证据工具、权重和风险分析仍留在 #1073 后续范围。
5. 「提交人工审阅」后填写理由，再批准测试或否决；归档项目只读。新关联同站点/ASIN 归组但不覆盖已选证据；明确切换观察会回到待研究，旧决定/理由/证据身份保留在最近历史中。页面展示最近 20 项观察及当前选定项、最近 20 条人工历史；数据不会因窗口隐藏而删除。
6. 勾选来源可读且已批准的候选，按批次进入原我的数据弹窗。研究预填锁定原 item revision/publication/version，重新读取不一致则停止。显式清除预填后属于新的普通商品选择。确认发送非空 ItemIDs，空子集不能退化为整批；原独立整批操作有明确范围。跨批分别确认，不承诺全部成功。
7. 转入仍由 Preparation 执行，成功后沿原回执进入 `/workbench/supply/mine?preparation={id}` 查看待适配。研究页不保存“已转入”布尔状态。转入响应未知沿原 Supply key 核实；研究写响应未知先「核实原操作」，没有回执才允许明确使用原键重试。挂起意图按 actor/org/project 保存于浏览器会话存储，不自动重发。

浏览器桥、1688 货源匹配、利润与销量预测、监测、自动采购/上架未开放。服务端实际支持的站点和字段仍需获准的真实环境验证，不能从 fixture 或 CI 推断。

## 显式新空安装与受限 serving pool

只在另行获准的新空 Product 安装使用现有 SRC/Catalog/Collection/Data Services/Preparation installer；不在已有业务库上升级 CHECK、不迁移旧数据、不让 serving 自动 DDL。Amazon Preparation 安装合同现在包含 `amazon_data`，拒绝 `sds_template`、`custom_dataset` 等其他来源。

研究新增三表 `research_opportunities` / `research_evidence_links` / `research_commands`，位于同一物理 Product 数据库。预先建立 `product_research_runtime` 普通角色，不能为 superuser、角色/数据库创建者、复制角色或有角色成员关系，不能拥有 public CREATE 权限。沿既有私密配置途径提供安装 DSN，不把密码写进 shell 历史或仓库：

```powershell
# MARKET_RESEARCH_SCHEMA_DSN 由操作员按原安全方式注入；本轮未对保留实例执行。
go run ./cmd/market-research-schema-init --runtime-role product_research_runtime
```

installer 只安装本 owner 三表并授予准确权限：三表 SELECT/INSERT；机会行仅更新 selected_link_id/state/reason/revision，关联行仅更新 removed。命令表无 UPDATE/DELETE，其他 owner 表无读取授权。serving 启动验证表约束和真实角色权限。来源与项目读取经原 Service 注入，不给研究 pool 其他 owner 的任意表权限。

在当前私密 JSON manifest 添加 `marketResearch.database`（字段 host/port/user/password/database/maxConnections，user 固定为上述角色，maxConnections 1–4）；host/port/database 与 `productAcquisitionDatabase` 一致，必须使用独立连接池。依赖已有 `projectCenter`、`dataServices`、`productCollections: true` 和 Product pool；Project 保持独立数据库。沿 [Data Services 运行交接](data-services-runtime-handoff.md) 配置显式 browser/driver/Temporal 和原 Resource，但本轮不执行实际任务。缺依赖、错误角色或 schema 均拒绝启动。

```powershell
go run ./cmd/current-application -config C:\private\current-application.json
# 前端仍使用现有同源服务 API 配置，正常启动 web/listingkit-ui：
pnpm --dir web/listingkit-ui dev --hostname 127.0.0.1 --port 3000
```

后端 context 返回 `marketResearchAvailable` 时项目页展示区域；没有安装则不伪造可用。已有选品 AI 与供应链继续使用各自显式模块、权限、配置和恢复 owner，不由研究开关自动开启。研究接口是 `/api/v1/workbench/research/projects/{projectID}`、`POST .../commands`、`GET .../commands/{key}`，前端同源 BFF 保留 verified actor/org 与原键，拒绝浏览器指定事实身份或 scope。

停止沿原 `-shutdown-file` 或正常进程终止执行，等待 serving/worker 结束再释放连接池；不会删除数据库或保存结果。重启读取原本地回执和当前来源，不能因重启重新采集、再次扣费或换键转入。数据保留/备份/恢复沿各数据库现有 owner，不新增研究专属恢复器。
