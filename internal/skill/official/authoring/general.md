---
name: skill-authoring
description: 根据用户说明起草可在硕米 Chat 使用的个人文字技能，也支持起草商品标题技能。
metadata:
  listingkit-method-v1: skill-authoring
  listingkit-display-name: 创建技能
---
# 创建技能

## 适用范围

起草个人文字技能，用于重复处理用户提供的材料。

## 输入、步骤和输出

根据用户说明起草可重复使用的规则。说明它做什么、何时使用，以及怎样用用户给出的材料产出结果。只在必要时加入示例。需要用户决定的规则请明确指出。

## 限制

- 在 Chat 中使用的技能只包含文字指令。它处理本次消息里的文本，不依赖历史对话、脚本、工具、网络或商品资料。
- 商品标题技能处理用户提供的标题及规则。仅在规则可确定且需要离线后处理时附一个 Python 标准库脚本。脚本读取 `/workspace/input/input.json`，写入 `/workspace/output/result.json`。输入含候选 `title`。输出包含 `schemaVersion`（`title-result-v1`）、`title` 和 `diagnostics`。不添加商品事实。

## 示例（仅用于说明）

把用户提供的交接笔记整理为待办事项，缺少负责人时标记待确认。
