# Code Health Rules

当前检查复用 `make code-health-audit` 和 `make code-health-verify`，配置和 analyzer 版本以仓库脚本与 lockfile 为准。源码职责见[Repository Structure](../development/repository-structure.md)，依赖方向见[系统边界](../architecture/project-boundaries.md)。

## 检测与处理

- 编译失败以及新增 Knip file / dependency / export / type findings 按现有 CI 配置处理。检查失败时保留 `.local` 分析报告用于定位。
- Go deadcode 和 jscpd 输出是候选，不能单凭未引用或相似度删除实现。先核对当前正式入口、平台差异、权限、错误分类、幂等、恢复和持久化效果。
- 测试、fixture、生成合同和操作脚本是否保留，以其当前消费者和行为合同判断；不因检测噪声新增 wildcard allowlist、假入口或扩大 ignore。
- 跨领域相似实现优先保持各自 owner；只有同一 owner、行为中立且已有有效回归证据时做最小复用，不为消除重复建立通用框架。
- 依赖修改限本次交付需要，不顺带升级其他模块；lockfile 和 analyzer 配置变更单独说明原因。
- 文档不保存滚动检测计数、已完成删除账本或过期工具列表。具体 finding、准确代码和验证结果放在执行 Issue / PR 中。

旧实现存在或旧测试通过不形成新的产品需求。全新系统继续遵守当前 owner、权限、事务和外部副作用规则。
