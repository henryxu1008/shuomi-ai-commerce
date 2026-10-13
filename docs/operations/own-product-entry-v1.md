# 自有商品独立页面使用与交接

Refs #675；设计与准入见 [own-product-entry-v1](../architecture/own-product-entry-v1.md)。

## 正常运行入口

沿现程序启动和 Auth.js/ZITADEL 配置；后端与 Console 应使用同一候选，以支持 `OwnVariant.images`。本次没有schema变更，不运行数据库初始化/迁移，也不需要新服务或新provider。

Console 保留原 `LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED=true` 和合法 `LISTINGKIT_SERVICE_API_BASE`，当前成员需 `workbench.collection.manage`（新增）及 `workbench.collection.read`（库/详情）。不得用样例身份代替现有登录与企业上下文。

前端开发启动（已使用现有获准的私密Auth与后端配置时）：

```powershell
Set-Location C:\Users\Henry\.codex\worktrees\own-product-page\task-processor\web\listingkit-ui
pnpm install --frozen-lockfile
pnpm dev
```

在配置的 Console 域名打开 `/workbench/data/mine?view=own`，点击“添加自有商品”，进入 `/workbench/data/mine/own/new`。从库或供应链来源入口查看本人自有商品，进入 `/workbench/data/mine/own/{itemId}`。

## 使用步骤

1. 确认当前企业，填写商品名称、品牌/描述及图片。在“商品属性”填写材质、产地、适用季节等属性名称/值，可添加或删除属性；属性名称不能重复，填写了一半的行须补全或删除，整行空白忽略。
2. 选择单规格或多规格。多规格填写名称与逗号分隔的值，例如颜色“红色,蓝色”和尺码“S,M,L”，点击“生成变体组合”。
3. 逐行填写SKU、币种、来源价格、来源库存；或填写公共值后“应用到全部变体”。仅批量覆盖已指定字段。每个变体可从商品图片池选择一张图；不售卖的组合可删除。
4. 修改规格后显式“更新变体组合”；保留相同组合资料，移除不在当前规格中的组合。确认后“保存商品”。
5. 保存完成返回自有商品库；打开商品的正常详情页，通过大图/缩略图和“当前SKU”查看对应图片、来源价格/库存，下面核对全部变体、同款通用属性、描述和保存信息。来源价格/库存不是平台实时同步数据，保存不自动适配、批准图片或发布。

单规格未填写SKU时可先保存基础资料。填写SKU后，币种、价格与库存须同时声明；空值不变成0，明确填写0可保存并准确读回。最多256变体、40公共图片，规格/图片合计和编码大小还受原Source限制；输入过大时减少组合或图片。

上传期间图片编辑暂时锁定；40张图时不启动新上传，原媒体pending核实入口保留。商品保存结果不确定时保留原请求并锁定表单，先“核实原操作”；仅未找到原结果后显式“重试原请求”，复用原键/原载荷。切换身份或企业清空未保存输入，不携带上一个身份资料。未保存表单不构成持久草稿，关闭或刷新前须保存。

## 数据与当前交付状态

用户证据、不可变Catalog原始版本、Collection来源引用和命令回执沿原Product事务保存；只改变录入与读取投影，不重建事实owner。原始资料不支持本轮原地修改；后续平台适配继续保留原始版本及Asset显式批准。

本轮实际开发验证：六组合单请求、图片按变体绑定、0价/0库存、原键恢复、身份切换、上传回调和旧own来源链接；原真实PostgreSQL测试使用新隔离container，读回与重放/异载荷冲突通过。浏览器只核对独立新增页/六组合/批量填写/深浅主题，未向共享实例提交商品。

当前可查看的本地UI预览：`https://localhost:52775/workbench/data/mine/own/new`。该预览使用独立前端及GET-only读取转发，禁止对原实例写入；“保存商品”不能作为当前实例试用验收，完整保存已在隔离测试中核对。现运行实例没有更新，候选合入/安装后才开放该实例的正式新增路径。

预览资源仅属于本任务：前端exec session、`own675-preview-read` 和 `own675-preview-namespace-read` 两个只读转发容器。停止预览可停止这两个命名容器及前端进程；不操作其他容器、卷、原实例或其数据。证书沿原可信配置使用，未修改系统证书信任。

状态保持In Review，待独立验证/用户试用；用户验收、共享部署/实际图片上传、平台/付费AI均NOT_RUN。未合并、部署或关单。
