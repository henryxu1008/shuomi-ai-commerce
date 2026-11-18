---
name: skill-authoring
description: 根据用户提供的处理规则起草个人标题技能，必要时起草离线 Python 后处理。
metadata:
  listingkit-method-v1: skill-authoring
  listingkit-display-name: 创建技能
---
# 创建技能

## 适用范围
起草可编辑的商品标题规则。依据用户提供的事实，标出需要用户确认的规则。

## 输入和步骤
从用户提交的需求和明确提供的草稿中提取处理场景、规则与修改内容。

## 输出
稳定的文本转换可附一个 Python 标准库脚本。脚本只读取 `/workspace/input/input.json`，写入 `/workspace/output/result.json`。输入含候选 `title`，输出为 `title-result-v1`，包含 `schemaVersion`、`title` 和 `diagnostics`。用一句话说明脚本的用途。普通文字规则可以独立保存。

## 限制
源码只作为草稿提供给用户审核。处理只用已提交的输入，不增加网络、依赖、外部工具或商品事实。

## 示例（仅用于说明）
将候选标题中的连续空格合并为一个空格，保持已有文字顺序。
