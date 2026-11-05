# Design System：两个现有页面的隔离试点

关联 [#1092](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1092) 与[资产基线](shuomi-design-system-baseline.md)。本次复用真实业务页面，验证其前端表示与交互；没有复制页面或建立另一套运行状态。

## 变更和复用

| 页面 / 入口 | 本次采用 | 原业务边界 |
| --- | --- | --- |
| 添加自有商品 `/workbench/data/mine/own/new` | 原页面、Input/Select、图片链接 Textarea；Textarea 改为现有 control-background/control-radius | 商品描述仍为 DescriptionEditor；原创建、版本、图片与库存声明合同不变 |
| 平台上架资料 `/workbench/supply/materials/[recordId]` | 本次用途、候选类目、本次模板三个局部原生 select 改用现有 Select；中断分支的补充意见采用同一 Textarea | 原 value/onChange、选项、禁用、配置/权限、原请求键和恢复路径不变 |
| 平台资料确认弹窗 | 关闭时将焦点返回仍在页面中的原触发元素；复用现有 AccountDialog 的焦点恢复方式 | 取消不调用 prepare/start；确认和执行协议不变 |
| 浅色辅助文字 / 选中导航 | 原 owner 交接后，仅按变量名调整 muted-foreground 与 sidebar-accent-foreground | 不改主按钮、焦点环、成功色、深色主题或导航结构 |

Select 在本增量增加一个页面导入点。资产 CSV 保留盘点基线，不把原基线 37 个导入点误写成新扫描结果。没有新增组件库、主题 provider、全局变量、业务状态或外部调用。

原 #1075 负责人已只读确认两个页面及 Textarea 没有文件冲突；product-collection/agent-configuration 合同按各自已有版本消费。其分支冻结后，另明确将两个浅色文字令牌交给 #1092。本次只在自己的分支按变量名修改这两项，不改原冻结分支，也不改 shell、导航结构、collection-page、企业 context 或 console.css 其他部分。

## 验证环境与证据范围

运行当前仓库的正式 Next 生产构建和真实页面、ApplicationFrame、shell、next-themes、WorkbenchContextProvider、客户端 API 及 Zod 响应解析。使用任务独占的回环端口与临时浏览器上下文，Auth.js 会话由独立临时测试 secret 生成；不读取已有 cookie、账号、企业数据或凭据。

浏览器在 `/api/` 边界返回合成契约响应。商品、类目、店铺和运行数据都明确为合成；上下文、原草稿、规则、配置和运行响应按当前 schema 验证。外部浏览器请求被拦截；后端配置指向未提供服务的回环地址。没有启动真实模型、供应商、平台提交、计费、数据写入或部署。

因此证据属于**真实前端产物 + 合成会话/API 的受控页面试点**。实际 OIDC 登录、Go/数据库领域授权与事务、真实企业使用、平台/模型调用、保留实例更新及用户验收仍为 NOT_RUN。现有 CI 的通用浏览器检查不冒充本地两页试点。

## 检查场景

| 场景 | 核对内容 |
| --- | --- |
| 两页 dark/light × 1440/390 | 页面可用、没有文档级横向溢出；输入与选择在主题、尺寸切换后保留；宽表格仍在原局部滚动容器中 |
| 自有商品图片链接 | 原 Textarea 的多行链接保留，添加后进入原图片池；没有把商品描述改成 Textarea |
| 自有商品保存冲突 | 仅一次合成保存 POST；原名称、富文本和图片保留；请求中明确库存 0 保留为 0 |
| 材料类目 / 模板 | 原用途切换与候选类目选择保持；规则查询仍使用原只读语义的 POST，不算作模型执行 |
| 材料确认取消 | Esc 和取消按钮关闭原生 dialog，焦点返回原操作；没有 prepare/start 请求 |
| 中断补充意见 dark/light × 1440/390 | 实际 `interrupted && canResume` 分支，原多行意见保留、控件表面一致、键盘能聚焦 |
| 继续结果不明 | 原键只发送一次合成 resume POST；原意见保留并锁定，不自动换键或重发 |
| loading / context error | 使用原工作台加载和重载入口；没有执行请求 |
| denied / record error | 原禁用或错误提示正确；没有执行请求 |
| read-only 原运行 | 建议可读，采用/保存动作不出现，没有执行请求 |

截图及结构化结果在本次任务交付的 `issue1092-page-evidence` 中；结果包含四个相关生产源码文件的 SHA-256，便于与最终提交核对。对比度修复前的证据单独保留。临时浏览器与应用进程在检查完成后退出，没有作为用户试用实例交付。

## 焦点缺陷与回归

实际浏览器验证发现：确认窗卸载后焦点落回 body，未回到「准备平台资料」。随后新增原页面回归，先确认它在旧实现失败（RED），再在卸载时关闭 dialog 并恢复仍连接在文档中的原焦点（GREEN）。回归同时检查取消没有调用 prepare/start。没有为此建立新的弹窗或恢复框架。

## 对比度修正与可访问性范围

修复前，dark 主场景 axe 未报告违规，light 自有商品页宽/窄分别有 11/6 个对比度节点，材料页宽/窄各 3 个。记录的是受测 DOM 节点，不是全站缺陷数量。最小修正如下：

| 令牌 / 背景 | 原组合 | 当前组合 | 对比度约值 |
| --- | --- | --- | --- |
| muted-foreground / background | `#617487` / `#f4f7fb` | `#5f7184` / `#f4f7fb` | 4.48 → 4.67 |
| muted-foreground / secondary | `#617487` / `#f1f5fa` | `#5f7184` / `#f1f5fa` | 4.40 → 4.59 |
| sidebar-accent-foreground / sidebar-accent | `#087f6c` / `#e5f5f0` | `#077b68` / `#e5f5f0` | 4.37 → 4.61 |

修正后重新运行实际页面的 20 项场景，全部通过；两页在 dark/light × 1440/390 的 8 个主场景 axe 均无违规。中断反馈字段的四个主题/屏宽组合、取消焦点以及冲突后名称、描述、两张图片和公共归属保留也均通过。真实后端和真实业务仍未执行。

这只是已验证页面和状态的前端证据；全站组件、全部 hover/浮层/状态、真实业务与用户验收仍需后续范围，不能升级为整个 #1092 的 a11y PASS、产品验收通过或可上线。没有把未测试范围记成已接受风险。

相关检查入口：`pnpm test -- src/components/workbench/collections/own-product-entry-page.test.tsx src/components/workbench/supply/materials-page.test.tsx src/components/ui/form-controls.test.tsx`，以及按既有前端流程运行 lint、typecheck 和 build。准确 HEAD、CI、独立评审和最后一次实际结果写在主要 Draft PR 中，不自动关闭父 Issue。
