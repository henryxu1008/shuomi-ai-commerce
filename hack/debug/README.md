# Debug Tools

这个目录保存仓库内受管的调试可执行程序。

- 这里只放非生产发布入口
- 正式服务入口仍只在根目录 `cmd/`
- 运行调试工具前，请先准备本地配置文件或必需环境变量
- `image-reconcile-774` 仅核对 #774 已获人类授权的四笔图片预留；默认 dry-run，正式处理须有私密证据、备份及原 worker 静默，不调用 provider 或初始化 schema。参见[冻结设计](../../docs/architecture/bailian-image-no-generation-reconciliation-v1.md)。

示例：

```bash
cd hack/debug
go run ./test-amazon --help
```
