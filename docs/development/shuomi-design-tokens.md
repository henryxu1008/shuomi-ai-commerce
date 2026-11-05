# Shuomi 现有设计变量参考

关联 [资产与缺口基线](shuomi-design-system-baseline.md) 和 [#1092](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1092)。这是当前源码变量的使用参考，不新增主题系统，不作为已完成视觉或可访问性验收的证明。

Figma 只作业务流程参考；本表从当前代码读取。变量 owner：`src/app/globals.css` 负责根页面，`src/app/console.css` 负责工作台。选择与持久偏好归原 `next-themes`，业务组件只消费变量。

## 工作台明暗对照

| 语义 / CSS 变量 | Light | Dark | 消费方式 |
| --- | --- | --- | --- |
| 页面 `background` / `foreground` | `#f4f7fb` / `#172b45` | `#0b1422` / `#edf5ff` | `bg-background text-foreground` |
| 卡片 `card` / `card-foreground` | `#fff` / `#172b45` | `#142237` / `#edf5ff` | Card / dialog surface |
| 浮层 `popover` / `popover-foreground` | `#fff` / `#172b45` | `#142237` / `#edf5ff` | 挂到 body 的浮层仍用原主题 |
| 主操作 `primary` / `primary-foreground` | `#087f6c` / `#fff` | `#42d2b0` / `#052b25` | Button，不用作所有正文颜色 |
| 次级 `secondary` / `secondary-foreground` | `#f1f5fa` / `#172b45` | `#1c3047` / `#dce9f7` | 次级表面 / 控件 |
| 辅助 `muted` / `muted-foreground` | `#f1f5fa` / `#5f7184` | `#1c3047` / `#b5c8da` | 辅助说明；浅色最小调色修正受测文字对比度 |
| 强调 `accent` / `accent-foreground` | `#e5f5f0` / `#087f6c` | `#123b32` / `#42d2b0` | hover / 选中 |
| 错误 `destructive` / `destructive-foreground` | `#c23d47` / `#fff` | `#f28d90` / `#26010b` | 危险操作与错误标识 |
| 柔和错误 `destructive-muted` | `color-mix(destructive 12%, card)` | 同公式 | 错误表面；不是实心危险按钮 |
| 成功 `success` / `success-foreground` | `#e5f5f0` / `#087f6c` | `#123b32` / `#42d2b0` | 仅表示真实成功 |
| 警告 `warning` / `warning-foreground` | `#fff4e3` / `#825108` | `#412d15` / `#f3bb6b` | 待处理/风险；结果不明不冒充成功 |
| 信息 `info` / `info-foreground` | `#ebf3ff` / `#245daf` | `#102a49` / `#77aeff` | `info-foreground` 引用 `console-blue` |
| 边界 `border` / `input` | `#d8e4ee` | `#30465c` | 边框，不用 currentColor 代替默认值 |
| 焦点 `ring` | `#087f6c` | `#42d2b0` | focus-visible；不得移除键盘焦点提示 |
| 导航 `sidebar` / `sidebar-foreground` | `#fff` / `#496177` | `#142237` / `#b5c8da` | 由 shell 消费，业务页不重定义 |
| 选中导航 `sidebar-accent` / `sidebar-accent-foreground` | `#e5f5f0` / `#077b68` | `#123b32` / `#42d2b0` | 浅色文字对比度约 4.61:1；不改变主按钮或其他绿色语义 |
| 遮罩 `overlay` | `#0b142280` | `#0009` | 原生 dialog backdrop 等 |
| 阴影 `shadow-color` | `#172b4518` | `#0004` | 使用现有 shadow 语义 |
| 控件表面 `control-background` | `card` | `card` | Input/Select 与本地 Textarea 修正共同采用 |
| 凸起表面 `surface-raised` | `secondary` | `secondary` | 不增加新局部调色板 |

Light 规则覆盖 Dark 默认；`:root:has(.console-theme)` 将工作台变量提供给 body 级 Portal。该机制的存在不等于所有浮层、登录跳转或 hydration 都已浏览器验收。

## 尺度、文字与动效

| 类别 | 当前工作台变量 | 使用原则 |
| --- | --- | --- |
| 圆角 | `radius=12px`，`control-radius=10px` | 普通卡片与普通控件区分；业务专用结构有理由再覆盖 |
| 控件高度 | `control-height=40px`，`control-height-sm=32px` | 多行输入保留所需最小高度，不强制 40px |
| 页面间距 | `page-gutter=24px`，窄屏 16px；`section-gap=20px`，`card-padding=20px` | CSS 控制响应式，不能按 Figma 坐标硬编码 |
| 标题 | `page-title-size=24px`，`section-title-size=20px`，`card-title-size=16px`，`dialog-title-size=20px` | h1/h2 语义与视觉层级同时核对 |
| 正文 / 标签 | `body-text-size=14px`，`label-text-size=13px`，`caption-text-size=13px` | 内容长短不能靠降低字号解决 |
| 指标 | `metric-text-size=28px` | 只有真实业务指标；缺数据用明确未提供状态 |
| 字体 | Noto Sans SC / Microsoft YaHei / PingFang SC / sans-serif | 现有字体栈；不增加字体下载依赖 |
| 动效 | `motion-fast=120ms`，`motion-normal=200ms`，`motion-ease=cubic-bezier(.2,.8,.2,1)` | 复用既有 reduced-motion 分支，不加入持续装饰动效 |

## 根主题与例外

根页也有 light/dark 的 surface、text、primary、status、border、radius 和字体变量。工作台覆盖它们是现有作用域设计，不再创建一套 theme provider 或持久偏好键。根主题目前没有定义 `info` / `info-foreground`，虽然 `@theme inline` 已有映射；如有根页真实消费者，先核需求再补，不能用 Console 内显示正确证明根页已满足。

品牌/平台图标、商品图片、图表系列和用户保存的富文本颜色是合法内容例外。图表、表格、编辑器的业务数据和持久化不属于设计系统。禁止将 UNKNOWN、denied、unavailable、read-only 或 disabled 统一涂成“成功”，也不让 UI 自行推导连接/授权/资源状态。

当前参考只描述变量。主要正文/状态组合、focus、Portal、表单及响应式仍须在真实试点中验证；“明暗配对存在”不能替代对比度、键盘、读屏或用户验收。
