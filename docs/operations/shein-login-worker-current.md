# 独立 SHEIN 登录 worker：源码已退休

独立 SHEIN 浏览器登录及登录后 Store ID 回写已按 [#985 完整设计](../architecture/legacy-shein-login-identity-sync-closure-retirement-boundary.md)整批退休。旧 CLI、验证码/取消、私有 Cookie/Profile/Redis、执行控制表安装器、专属配置样例、Dockerfile 和部署 workflow 已退出；不再提供此 worker 的安装或运行入口，也没有兼容替代入口。

当前需要的能力沿原 owner 使用：

- [Store 观察同步](../architecture/store-center-platform-observations-v1.md)与[当前运行装配边界](../architecture/store-observations-preparation-runtime.md)：消费官方授权连接读取商品/订单等事实。
- [仓库入口地图](../development/repository-structure.md)：当前应用、采集、图像与 schema 工具。

本轮只完成源码退休，不证明任何已有实例已停用。既有 Store 字段/表/审计、角色/grants、Redis 键、Profile/凭据和进程均未读取、迁移、清理或停止；真实运行停用 NOT_RUN/NOT_AUTHORIZED。生产发布仍 NOT_READY，不增加内部试用前置，#301 保持 OPEN。

原安装说明只作为 [冻结设计时的历史证据](https://github.com/qq550723504/shuomi-ai-commerce/blob/8fd07fd3af2a90a314ff7fdc8f0410a507895d34/docs/operations/shein-login-worker-current.md)，不构成当前执行或环境操作授权。
