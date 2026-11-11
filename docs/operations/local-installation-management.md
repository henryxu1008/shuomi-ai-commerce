# 本机安装与模块管理

执行范围见[#1240](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1240)及[已批准架构](../architecture/installation-and-module-management.md)。这是宿主显式启动的本机工具，业务后端、BFF和企业用户均不获得Docker或数据库owner权限。此入口不代表原35444已更新。

## 准备与启动

在Windows PowerShell 7中，从仓库准备准确Git版本。需要现有Docker、Go、Node/pnpm依赖缓存；构建只生成版本包，不初始化或更新任何实例。

```powershell
./scripts/build-listingkit-setup.ps1 -SourceRef '<准确提交SHA>' -OutputDirectory 'C:\Users\Henry\AppData\Local\Shuomi\packages\setup-<版本>'
& 'C:\Users\Henry\AppData\Local\Shuomi\packages\setup-<版本>\listingkit-setup.exe'
```

已有本轮准确源码的`All`镜像receipt时，可传`-ImageReceipt <文件路径>`复用；仍验证源码SHA及实际镜像ID，不能复用旧版本后端。标准镜像入口仍是`build-listingkit-images.ps1`，不会从旧应用发布镜像继承。

当前本机版本包需要完整包目录和本机Docker中同版本的两个应用镜像，不能只复制exe。界面会分别指出“应用服务安装文件”或“网页界面安装文件”未就绪；镜像缺失、身份或源码版本不匹配均保持阻断。若镜像已被清理，沿上方标准入口在新目录重新准备完整包并重新打开工具，不手改`release.json`引用其他版本。镜像仓库连接失败须先恢复网络后再构建；此操作不初始化或更新保留实例。

打开启动窗口显示的`http://127.0.0.1:<随机端口>`，在配对表单输入同窗口的一次性配对码。不要将配对码放在网址、聊天或截图里。配对一小时后失效，退出后须重新启动工具。浏览器只收集非秘密配置及本机文件路径。对外凭据在正常HTTPS业务入口及原凭据owner登记。

## 首次安装

选择**安装新实例**，填写新project及三个不同的本机端口，检查并预览后明确确认。首版开放已批准的“基础安装（账户与店铺）”和“业务功能安装”（知识库、数据服务、商品资料、通知、报告、工具市场、生态和存储）；Chat、采集计划、店铺观测等后置组合还需要身份建立后的显式组织范围和原owner前置，不伪装为一次新空安装选项。

“业务功能安装”需要本次准确HTTPS接收地址的采集插件包。沿现有入口准备目录后填入界面，不改JSON、不复用另一地址的包：

```powershell
./scripts/build-tool-market-plugin.ps1 -CaptureAppUrl 'https://localhost:<应用端口>/capture/1688' -OutputDirectory '<本机新插件目录>'
```

新装前重复核对project、容器、卷、网络和端口。任何原项目资源都会拒绝，不自动改名或清空。确认后只运行原Compose/bootstrap/owner安装流程；没有模型调用、采集、发布或演示商品。

默认由Docker分配网络地址；本机默认地址池耗尽时，可在表单显式填写未占用的私有IPv4地址段。工具只在此段内划分该组合的网络，预览显示准确地址段，确认前再次核对与现有Docker网络不重叠；不清理其他网络、不修改Docker全局地址池。

完成后在回执查看HTTPS入口和私密handoff目录。管理员为`local-bootstrap-operator@localhost`，口令文件`operator-password.txt`，公开根证书`root-ca.pem`。是否信任该本机CA由使用者决定；工具不跳过浏览器证书检查。未配置的店铺、模型、支付等能力仍不可执行。

## 管理现有实例

选择具体project查看模块。当前后端和UI可能来自不同更新批次，工具必须找到同时重现两者的完整Compose组合，才允许预览。缺少准确组合或原生只读检测失败时显示阻碍，不能以仓库默认模板覆盖私密overlay。

表中分别显示原生schema/运行权限、配置是否存在、后端及前端启用状态。未核实的范围保持待核实；启用状态不能替代业务权限或外部就绪。勾选所需变更，再检查、预览及确认。缺失的模块配置可选择已准备的同实例私密Config绝对路径，只带入所选缺失段；不会覆盖身份及其他模块。没有合格保留库维护入口的缺schema模块必须由原owner处理，管理工具不会运行新空初始化、手写DDL或提升运行角色。

应用只重建准确`current-application`与`listingkit-ui`，保持其当前镜像、其他服务、卷、网络、身份、配置和权限。不删除业务数据、不释放未决预留。原配置及完整组合保留在原计划私密目录。缺配置依赖时整个计划阻断，不静默启用其他模块。

## 中断、核实与回退

回执固定保存在当前Windows用户配置目录的`Shuomi/setup/v1`，与稳定Docker daemon/project身份绑定，不按浏览器会话或临时输出目录变化。不要删除或改名该目录来绕过未决操作。全局和实例OS锁阻止跨进程同时派发。

浏览器关闭或请求超时不能证明安装失败。重新启动、配对后，从实例列表的**原操作回执**选择**核实原操作**；这只核对原命令结果，不重新发送安装。缺少成功证据则保留待核实，沿原owner维护入口处理。新空installer中断后拒绝自动重放的原合同保持。

现有配置变更不涉及schema修改，原计划内保留`original.compose.json`、`original-current-application.json`和准确镜像信息。发生真实健康失败时，维护者先核对当前写入/漂移及该组合原回退条件，再沿原Compose入口恢复原配置；工具不会自动恢复旧数据库dump覆盖新业务事实。首次安装失败保留新项目现场，绝不自动`down -v`或删除卷。工具不提供通用DDL、脚本编辑、数据库回灌或任意实例删除按钮。

开发测试、隔离验证、CI与用户试用分别记录在PR；未实际执行的组合不得写为通过。原35444更新、真实provider/支付/发布及用户验收不由本工具开发授权自动包含。
