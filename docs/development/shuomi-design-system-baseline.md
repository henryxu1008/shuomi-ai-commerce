# Shuomi Design System：资产与缺口基线

关联 [#1092](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1092)。本报告记录 Phase 0 源码基线及现有表单控件的有限复用切片；两页受控前端试点另有记录，不代表整个设计系统或真实业务验收已完成。

## 依据和边界

- 盘点基线：`d12b8c7633a8e3a6f02e4c19f4a29344b14c8d34`，2026-11-05。
- 最新产品决定：Figma 仅用于业务流程参考，不再决定视觉、布局、导航层级、页面拆分、组件或响应式。规则更新由 [PR #1107](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1107) 单独负责；本报告不修改其文件。
- 继续复用 Next.js / React / Tailwind / shadcn / Radix、现有 CSS Modules 和 `next-themes`。身份、企业、权限、业务事实、计费、幂等、恢复及外部副作用仍由原领域 owner 负责。
- Design Basis: Reuse Existing Architecture，沿 Issue 已有分类及工作台主题合同。Textarea 消费原 control-background/control-radius；供应材料页的三个原生选择框复用现有 Select，保留原选项、value/onChange、禁用状态与调用；确认弹窗关闭后恢复原焦点。经原 owner 交接后，只对两个浅色文字令牌做最小调色。主题 provider、导航结构、API、权限、状态机、持久化和外部副作用不变。本文、资产表本身是现状记录。

## 方法与可追溯清单

[完整资产 CSV](shuomi-design-system-assets.csv) 逐项记录路径、用途、依赖、生产导入位置、测试导入位置、相邻测试及可达 page/layout 模块。路径相对 `web/listingkit-ui`。

扫描 `src` 的 TypeScript AST，解析 import、re-export 及字面量 dynamic import，解析 `@/` 和相对路径，排除整条 type-only import。生产导入数是导入该模块的不同非测试文件数，**不是控件渲染次数或验收覆盖率**。具名混合 type import、未执行分支、隐式 layout 组合和条件 feature gate 仍须人工核对；静态可达路由不能证明页面已开放。

数量基线：15 个基础组件实现、26 个 UI TSX（含测试）、375 个 workbench TSX（含测试）、148 个 workbench page 路由、5 个 provider、9 个本项目 `use-*` hook，另有 12 个 vendor Tiptap hook。没有 `.agents/skills` 或 `.codex/skills` 的已跟踪仓库技能；适用规则来自根目录与前端的 `AGENTS.md`。

| 基础组件 | 生产导入文件 | 当前用途 / 依赖 | 现有直接测试 |
| --- | ---: | --- | --- |
| Button | 168 | 操作、Slot 链接；Radix Slot / CVA | `button.test.tsx` |
| Card | 107 | 表面、卡片、标题和内容 | `card.test.tsx` |
| Input | 55 | 原生输入、统一控制表面和圆角 | `form-controls.test.tsx` |
| Select | 37 | 原生 select；不是 Radix 浮层 | 同上 |
| Textarea | 17 | 原生多行输入；与 Input/Select 存在样式差异 | 同上 |
| Table | 16 | 原生表格语义和滚动容器 | `table.test.tsx` |
| Badge | 8 | 紧凑状态、类别 | `badge.test.tsx` |
| Checkbox | 2 | 原生 checkbox | `form-controls.test.tsx` |
| Label | 2 | 原生表单 label | 同上 |
| Alert | 0 | 已有 inline status 封装 | `alert.test.tsx` |
| EChart | 0 | 已有 ECharts / ResizeObserver 封装 | `chart.test.tsx` |
| DataTable | 0 | TanStack 排序、表头、空态 | `data-table.test.tsx` |
| Separator | 0 | 视觉分隔 | `separator.test.tsx` |
| Skeleton | 0 | 加载占位 | `skeleton.test.tsx` |
| VirtualList | 0 | TanStack Virtual 滚动 | `virtual-list.test.tsx` |

零生产导入不等于应删除：这些组件可以作为候选资产，但不能称为“全站已复用”。`Table` 的 16 个直接导入包含未采用的 `DataTable`；同理，不把 import 次数当作全部可运行页面的使用次数。

## 现有模式与 owner

| 资产 | 实际情况 | 后续处理原则 |
| --- | --- | --- |
| `console/console-page.tsx` | 99 个生产导入；组合标题、面包屑、页面操作、筛选容器；ConsoleState 支持 loading/empty/error/unavailable | 优先复用，denied/read-only 仍按业务 owner 正确表达；不为补齐枚举强改权限行为 |
| `account/account-dialog.tsx` | 3 个导入；原生 modal/drawer，关闭后恢复原焦点 | 与 ResourceDialog 对照行为再考虑提取，不仅按外观合并 |
| `resources/resource-dialog.tsx` | 16 个导入；原生 dialog，支持 locked，Esc 受锁定约束 | 保留不可中断操作边界；原生 top layer 不等于 React Portal |
| `stores/store-catalog-dialogs.tsx` | 2 个导入，有组件和消费测试 | 店铺合同由 Store owner 提供，设计系统不生成连接状态 |
| `providers/theme-provider.tsx` | ApplicationFrame 注入；next-themes，class 模式，偏好键 `listingkit-theme` | 盘点基线仅支持手动明暗选择；当前选择规则见[设计变量参考](shuomi-design-tokens.md) |
| `providers/toast-provider.tsx` | ApplicationFrame 已挂载；context 未导出，未找到 useToast/业务消费者；无直接测试 | 不能作为当前业务反馈已启用的证据。硬编码配色是潜在问题，优先级低于真实使用控件 |
| `providers/workbench-context-provider.tsx` | 99 个生产导入，负责企业等运行上下文 | 不复制或改动；#1075 活跃分支持有相关修改 |
| 业务 command hooks | Collection 6 个、Supply 9 个、Data 2 个消费位置；API、授权和结果不明处理归原 hook/领域 | 复用其反馈输入，禁止为了统一组件再造业务状态 |
| `lib/query/use-*.ts` | 店铺 Catalog 和店铺详情读取沿既有 QueryProvider | 主题组件不拥有缓存或实时授权规则 |

## 按影响与成本排序的缺口

| 顺序 | 证据与影响 | 最小建议 | 成本 / 状态 |
| --- | --- | --- | --- |
| 1 | 基线 `Textarea` 使用 `rounded-md bg-background`；Input/Select 使用 `--control-radius` 和 `--control-background`。工作台 card 内长文本控件表面不同；17 个生产导入。当前 `radius-md` 恰为 10px，与 control-radius 相同，不能称为已出现圆角视觉缺陷 | 多行输入复用同一控件变量，保留原高度、resize、属性、事件和业务表单 | 小；已本地修正。不是页面改版 |
| 2 | ThemeProvider 没有直接主题偏好/hydration 测试；Console 在 `:root:has(.console-theme)` 为 body 浮层提供变量 | 复用现有隔离浏览器路径，核对已用浮层的主题切换、Portal/top-layer 和首屏；不再建 provider | 中；真实页面检查待执行 |
| 3 | 原生对话框的焦点恢复和 locked 行为不同，且业务约束不同 | 先选资源与店铺两个真实场景验证打开、Esc、锁定、关闭恢复及窄屏；只提取已证明相同的展示部分 | 中；不把两套局部实现自动认定为冗余 |
| 4 | 基础控件只有 class/结构等轻量测试；form-controls 未验证错误、只读与键盘交互 | 后续改动采用浏览器交互检查；保留 aria-invalid、aria-describedby 和 disabled/readOnly 原生语义 | 小到中；现有测试 PASS 不等于明暗/a11y 已验收 |
| 5 | DataTable / VirtualList 没有生产消费者，表格筛选和批量操作仍在页面 | 仅在已选两页证明共同需求后采用现有 TanStack 封装；先比较原页合同 | 中；不批量新建高级表格、多选或日期范围组件 |
| 6 | Toast 固定使用 16 处 palette 色工具类，但业务消费链不存在 | 记录未使用资产；当实际页面需要全局反馈时再核对语义与主题 | 低优先；不凭目录存在宣布业务问题已修复 |

隔离浏览器在旧候选确认：浅色 `muted-foreground=#617487` 在 `background=#f4f7fb` 上约 **4.48:1**，在 secondary 上约 **4.40:1**；选中导航 `#087f6c` 在 `#e5f5f0` 上约 **4.37:1**，均低于小字号的 4.5:1。随后原 #1075 owner 确认分支冻结、文件干净，将这两个浅色文字令牌交给 #1092。仅调整 muted-foreground 为 `#5f7184`、sidebar-accent-foreground 为 `#077b68`，对应约 **4.67 / 4.59 / 4.61:1**。未整文件替换品牌色，也未修改原 #1075 分支；两页实际复核及全站限制见试点记录。

源码色值扫描是候选筛选，不能当作缺陷计数：`description-editor.css` 的 169 处色值涉及编辑器/正文内容语义；`console.css` 的 74 处和 `globals.css` 的 58 处主要是合法主题定义；营销样式不在工作台试点范围。商品原图、品牌图标、图表系列、用户保存的文字/高亮颜色不应机械替换为主题色。

## 两个实际页面的隔离试点

1. 自有商品录入 `collections/own-product-entry-page.tsx`：继续复用完整页面与原 Input、Select、Textarea；真正受 Textarea 修正影响的是「图片链接」，描述仍为原 DescriptionEditor。核对图片链接多行输入、明暗/尺寸切换后保留、禁用及保存冲突后的原输入；后端提交和来源事实继续沿原 owner。
2. 供应材料 `supply/materials-page.tsx`：将「本次用途、候选类目、本次模板」三个局部原生 select 改用现有 Select；「补充核对意见」只在 `interrupted && canResume` 的真实分支出现。核对该分支的长文本、禁用、原编号与结果不明恢复反馈。没有新增重试、上传或平台发送行为。

使用实际 Next 生产产物、原页面、shell、provider、客户端 API 与 Zod 解析运行隔离浏览器检查；会话为任务独占的合成 Auth.js 会话，API 在浏览器边界返回经当前契约验证的合成数据，**不是实际 OIDC 登录或真实后端/企业数据验收**。试点细节与限制见[页面试点记录](shuomi-design-system-pilots.md)。Chat 属 #1087；研究/导航/企业 context 属 #1075；原负责人已确认两页与 Textarea 没有文件冲突，本批不改其持有文件和接口。

## 验证状态与下一步

- 基线已有基础 UI 测试：11 个文件、21 项通过。执行 `node node_modules/vitest/vitest.mjs run src/components/ui`；未改这些实现。
- 首轮控件对比为临时、合成检查：渲染基线快照与 Textarea 候选，消费实际 Tailwind/Console CSS；light/dark、1440/390 均无横向溢出，多行输入及 Tab 到只读字段通过。首轮发现的浅色对比度问题与后续两个实际页面的修复复核分开记录；控件截图不冒充页面或全站验收。
- 完整 `lint/typecheck/test/build` 结果与候选绑定后在交付记录维护；全站浮层、系统偏好、真实登录页面、真实业务、用户试用和部署均为 NOT_RUN。
- 本资产清单与 Textarea 小修可交付审查。远端 Issue 接单/正文更新被自动审批拒绝，正文未变；本地准备继续沿 Issue 现有 Reuse 范围，远端记录和 Draft PR 发布另行处理，不合并、不关闭父 Issue。
