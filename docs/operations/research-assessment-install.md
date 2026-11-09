# 为当前 Research 安装经营评估表

本入口属于 #1205 的现有 Product owner，准备依据见 [评估设计 §11](../architecture/research-commercial-assessment-persistence-v1.md)。仅添加已批准的评估表，不迁移数据、不重跑空库初始化、不启用来源或调用模型。

## 操作权限与停止点

工具准备和隔离测试不授权对任何实例执行。实际执行前单独批准准确源码/工具版本、目标数据库、备份及前后端更新范围。不得用本命令覆盖 schema 漂移；已有目标表不合格时拒绝并保留现场。

- 使用已存在的 `acquisition_owner`，确认该角色实际拥有准确目标库和四张现有 Research 表；应用继续使用受限 `product_research_runtime`。
- owner manifest 为私密普通文件，沿原 installer 的格式和访问限制；不得传 DSN/密码到命令行、日志或 PR。
- manifest `schemaVersion=1`、`collections=true`、`dataServices=true`、`researchSupplierReview=true`；database 为批准的准确当前库/本机端口和 owner。这些标志用于验证现有依赖，不安装/更新 DataServices。
- 实际操作前具备可恢复备份，记录原 Research 表 schema/ACL及业务记录基线；本工具不创建备份，也不替代备份恢复核验。

## 既有命令的互斥功能模式

```text
product-acquisition-init --config /private/approved-owner.json --confirm-database product_acquisition --enable-research-assessments
```

不得与 `--confirm-empty-database` 或其他 `--enable-*` 同用。上方仅为操作说明，本交付未在统一实例执行。

原 feature-installer 事务锁串行化显式初始化，默认30秒。核对准确库/owner、原 Research schema、SupplierReview schema和运行角色后：目标缺失时调用同一 `InstallAssessmentSchema`；目标已存在时只验证，不重建、不重授旧 ACL。字段类型/非空/写入默认值、主键/唯一/FK/check、索引有效性、RLS/触发器和准确授权不符均拒绝。安装后验证失败全部回滚；返回丢失可重跑同一目标，完整已安装状态只校验。

运行角色只获得新表 SELECT/INSERT，无 UPDATE/DELETE/TRUNCATE/REFERENCES/TRIGGER 或转授权限；PUBLIC及其他角色不能取得新表访问。原表、原授权、命令、候选和决定不改。不会填充“历史评估”，旧临时试算无法恢复为新评估。

## 统一实例版本边界

现用后端 `1b63fdb5e2eba8cff609f2980e57bad819c16d41`、UI `c72d64808b7cc86bb0912a141616ef1bfc919cbd` 各自保留。#1209/#1217 的评估和准入按当前组件基线定向组合，匿名HTTP采集继续按 #1193 退休；不带入其他计价/Store观察 schema 变化。组合提交是明确的安装候选，不把历史镜像改标签冒充新源码。

新组合必须用自身的相关检查证据，旧 PR CI仅是未变化代码的来源证据。完整main不是本实例当前库可直接启动的版本；本工具只解决评估表，不补 price_snapshot 或 Store订单约束。

更新前再次核对在途负责人、实际镜像及配置是否变化；出现变化先报告。随后仅在单独授权下构建并定向更新前后端，保留配置、卷、身份、密钥和会话，不执行整套compose down/init。

## 回退与验收

尚未写新评估时保留旧镜像和新表可回到旧应用；已有新评估/引用命令后不承诺旧程序可以解释这些记录。不能DROP表或用整库回滚覆盖并发业务写入，需按实际写入阶段决定修复/回退。

先只读核就绪、准确版本、原成员/权限和旧记录读回；样本写入、评估保存/引用/刷新恢复的实际验收另获授权。35448保留，不继续补装配。技术检查不代表运行或产品验收通过。
