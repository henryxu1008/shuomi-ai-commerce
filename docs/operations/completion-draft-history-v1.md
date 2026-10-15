# 待补全草稿保存历史

Refs #704；合同见[历史读取与比较](../architecture/completion-draft-history-v1.md)。本说明是候选使用交接，实例部署、用户试用和生产验收分别记录在PR，不能由源码或CI推定。

## 使用

1. 正常登录当前企业，从自有商品详情的“已有待补全草稿”继续编辑，或在供应市场 → 我的供应链 → 待补全中打开原来源。
2. 已有保存版本时，点击编辑页的“保存历史”，进入独立页`/workbench/supply/mine/missing/:sourceId/history`。未保存输入不会进入历史，返回草稿会重新读取最新已保存版本。
3. 历史按Rn从新到旧排列，显示保存时间和当前版本；“查看更早保存”继续翻页，“刷新最新历史”回首页。无保存head时显示真实空历史。
4. 点击“查看 Rn”进入独立只读旧版。固定来源Vn与保存Rn分别显示；候选标题、描述来自当时保存记录。
5. 勾选两个不同版本，可跨页选择，点击“对比所选版本”。独立比较页显示两个已保存版本的标题、描述前后值，保留空值、空白、换行，明确标记变化或未变化。

详情和比较页可返回保存历史；所有页面保留返回草稿和原转入批次的入口，从商品进入时也保留返回商品。导航参数必须与实际来源匹配；不接受任意返回URL。仅有查看权限时仍可读这三个页面。读取失败明确报错，可重试，不将错误当作空历史。

同草稿的定义是相同固定source和draftID；同一商品的不同转入草稿分别有自己的历史，不混合比较。旧版只读；本能力不执行保存、回滚、来源升级、店铺适配或平台调用。

## 启动与数据

沿现有current-application配置和`WithSupplyCompletionDrafts`装配；启用ProductCollections与SupplyCompletionDrafts即可，不要求官方Store/平台配置。后台、前端使用同一候选源码。正常源码启动沿`go run ./cmd/current-application -config <private-config.json>`和`web/listingkit-ui`中的`pnpm build` / `pnpm start`；私有配置与原数据库接线继续使用既有实例交接，不能将无配置启动当作产品可用。

新增两个只读GET，位于原`/api/v1/workbench/supply-preparations/sources/:sourceId/completion-draft/history`及`/:revision`；前端沿原`/api/workbench`BFF转发。数据来自原CompletionDraft records/head和Preparation source，不新增表、索引或grants，serving不运行DDL。

更新保留实例须有具体目标授权，仅替换同候选后台/前端；保留原镜像、私有配置、数据库、图片、命名卷和启动层，不重跑setup/初始化。本轮没有更新own675或真实业务记录；实际运行试用为NOT_RUN，相关fresh PostgreSQL与UI测试属于开发证据。
