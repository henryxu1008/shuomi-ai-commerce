# 本机 TLS 维护（操作者入口）

Refs [#965](https://github.com/qq550723504/shuomi-ai-commerce/issues/965)；[冻结边界](../../../docs/architecture/local-tls-revocation-maintenance.md)。浏览器用户按下载安装流程使用；这些命令只供部署操作者在明确批准的维护窗口使用。

## 当前交付边界

本入口提供独立维护脚本、无凭据离线 fixture 与 opt-in 接线模板。2026-10-27 另行授权的本机接线与正常 Schannel 检查记录见 [#965](https://github.com/qq550723504/shuomi-ai-commerce/issues/965)；该记录只证明当时准确版本和现场，不自动授权再次签发、接线、重启或请求。以下命令是操作合同，每次现场维护仍须获得准确动作授权。

## 显式命令合同

依赖 Linux 私有文件系统、POSIX shell/coreutils 和 OpenSSL 3；不新增 daemon/服务/端口，不重放全量 bootstrap。`STATE` 必须在操作者控制的目录下，为不存在的绝对路径；父目录须已存在。准备会持久保存新 CA 签名 key，请先批准准确路径和访问范围。

| 命令 | 输出/效果 | 不自动进行的后续动作 |
| --- | --- | --- |
| `sh local-tls-maintain.sh init STATE PORT` | 新根、原生 CA state、首份空 CRL；PORT 对应既有 application entrypoint（当前 35444） | 不签发 localhost、不安装信任、不改现有卷 |
| `sh local-tls-maintain.sh issue STATE` | 新 `releases/<HEX_SERIAL>/localhost.crt`、`.key` 和 complete 标志；校验 SAN/purpose/CRL | 不吊销旧 leaf、不部署、不重启 |
| `sh local-tls-maintain.sh refresh-crl STATE` | 验签后原子更新 DER `public/root.crl`，旧 CRL 留在 history | 不改路由、不自动安排下次刷新 |
| `sh local-tls-maintain.sh revoke STATE HEX_SERIAL` | 仅撤销本 CA 原生 `issued/<serial>.pem`，再生成/验签/发布新 CRL | 不撤销任意文件、不替换正在运行的 leaf |

每个 native 调用保存 `.operation/<n>.stdout`、`.stderr`、`.exit`。成功移到 `history/<time>-<pid>-<action>`；失败/未知保留 `.operation`，所有后续操作拒绝。**不能删除锁然后重试**：先保留整个 state，核对已前进的 OpenSSL index/serial/撤销事实与未发布 CRL，再由操作者批准准确恢复动作。没有自动回退或幂等重试承诺。不要将 key 内容或整个 state 上传至 Issue/PR。

`public/` 仅有公开根与 DER CRL；CA private state 不挂到 UI/Traefik。CA 期限十年，leaf 三十天，CRL 七天。必须由操作者在 nextUpdate 前刷新 CRL；本批没有调度服务。缓存可能在有效期内继续使用旧 CRL，不能承诺撤销瞬时传播。缺失/过期/不可达应保留系统验证失败，不能关闭吊销检查。

## 另行批准后的接线顺序

1. 核对准确 target/端口/既有根的全部消费者、新根信任切换范围及中断预算。旧 bootstrap 的 CA key 已按原脚本清理，不寻找或假设能重新签旧 CA。
2. 按批准路径准备新 state 和 release。保留旧证书/配置/卷；不得把整个 state 挂给任何 serving 容器。新根的信任安装、应用 trust bundle、服务证书替换分别获得批准。
3. `public/root.crl` 必须先存在。`docker-compose.local-tls-crl.yml` 只给原 listingkit-ui 增加只读公开目录 bind；`create_host_path:false` 防止错误路径隐式创建。该 override 未加入默认启动脚本。
4. 将 `traefik-local-tls-crl.fragment.yml` 的唯一 router/middleware **合并**进准确当前 dynamic file，不能用 fragment 替换全文件。保留所有原 router/service/middleware/TLS；没有新的 service 定义。业务仍为 HTTPS，只有 localhost 的精确 GET/HEAD CRL 路径允许 HTTP。现有 file-provider watch=false，需要单独批准准确一次加载/重启。
5. 单独核对实际 CRL 获取、有效期和正常 Windows Schannel HTTPS 的状态/头/正文/退出码。任何异常或未知立即停止留证，不自动重试/回退。内部 401 或本 PR CI 不能代替外部入口验证。

以上为另行授权后的操作顺序，具体执行结果记录在对应 Issue/PR，不从文档或源码 CI 推断现场状态。本机 TLS 修复不代表公开 Internet 部署、Native 身份交互、Store 授权或产品业务验收完成。撤销负例必须是另行授权的隔离证书，不能撤销现场使用的 leaf。

## 离线测试

`go test ./tests -run '^TestAccountComposeLocalTLS' -count=1 -v`

Linux 执行真实 shell、以临时 PATH 中的文本 fixture 替代 OpenSSL，不产生实际密码学材料或网络请求。Windows 仅做入口/语法与模板检查，Linux 编排案例明确 SKIP，由准确 HEAD 的原 CI 执行。此 fixture 不验证实际 OpenSSL/Schannel 兼容性；另行授权的隔离实测和现场检查按 #965 的准确版本记录，不把 fake runner 结果写成密码学验证。
