# 标题技能脚本：使用与私有执行配置

对应 [#1293](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1293) 与[批准设计](../architecture/skill-script-sandbox.md)。默认没有 `productAgent.scriptSandbox`，执行环境关闭；编辑/导入/检查/分享/复制独立可用。设计§23完成独立WSL执行profile的适用D-IT1/init-proc验证；§24另记录用户授权的保留current-application/UI受控内部试用，已走通标题脚本到人工采用。只对该独立组合成立，不外推业务部署或用户验收。本轮不更新 Account Compose 或保留业务实例。

## 用户操作

1. 正常登录，在“我的技能”或有权限的“团队技能”新建技能。用“导入 SKILL.md / 文本包”一次选择 `SKILL.md`、`execution.json`、`main.py`，或导入包含完整 files 的 JSON 包；`.py` 映射到 `scripts/`。每文件16KiB、包48KiB、最多16文件。检查和保存只校验结构，不运行代码。
2. `SKILL.md` 显式标记 `metadata.listingkit-method-v1: product-title`。准确版本摘要涵盖全部文件，改脚本须保存新版本。团队/官方权限、发布启用、个人企业准入沿原库处理，复制/导入许可不等于执行许可。
3. 在现有商品内容建议入口选择有效企业、准确技能版本和仅标题范围，确认原模型报价后启动。仅 default/read_only；其他模式、未配置环境、多字段和自动调度拒绝。模型只读方法文本，不接收脚本/manifest，不决定代码派发。
4. “脚本后处理结果”分别显示结果与清理状态，成功显示前后标题。失败、未派发、UNKNOWN 均停止，不自动重试/跳过。沿原编号读取不会重派。default 沿原 Review 人工接受/采用；read_only 禁止提交采用。结构成功不表示业务事实或语义正确。

严格 execution.json（无额外字段）：

```json
{"schemaVersion":"skill-script-v1","profile":"python-title-json-v1","entrypoint":"scripts/main.py","phase":"title-candidate-postprocess","inputContract":"title-candidate-v1","outputContract":"title-result-v1"}
```

固定 `/usr/local/bin/python3 -I -S`、UID/GID65534、cwd `/workspace`。入口读 `/workspace/input/input.json`，写 `/workspace/output/result.json`，无自由命令参数。输入为 `schemaVersion`、`title`、`targetLanguage`、`evidenceRefs`。输出如 `{"schemaVersion":"title-result-v1","title":"处理后标题","diagnostics":[]}`：UTF-8、含 diagnostics、无额外/重复字段，整个文件≤8KiB、诊断≤8条且各≤256字节。无 pip、网络或业务 API；诊断不作为事实，也不展示远端原始日志。

## 部署 owner 配置

复用 current-application 私有运行 manifest 与原启动方式，在已有 `productAgent` 下增加可选 `scriptSandbox`。不新增 schema、启动器或部署平台。以下尖括号须替换为真实已验部署事实，不是可直接开放的配置：

```json
{
  "domain": "<后端专用TLS主机:端口>",
  "caFile": "<后端私密只读挂载内CA文件>",
  "credentialFile": "<后端私密只读挂载内凭据JSON>",
  "trialEvidence": "sha256:<完整适用部署验收记录的64位小写摘要>",
  "descriptor": {
    "profile": "python-title-json-v1",
    "revision": "<执行profile版本>",
    "imageDigest": "sha256:<批准Python镜像不可变ID或digest>",
    "execdDigest": "sha256:<批准同版execd不可变ID或digest>",
    "networkRevision": "<已验网络策略版本>",
    "toolVersion": "v1.0.0"
  }
}
```

凭据 JSON 只含 `user`、`password`、`lifecycle`，后两项各至少16字节；使用真实独立随机服务凭据和私密挂载，不提交 Git、不进入浏览器、Skill、镜像层、模型或审计。user/password 用于 Traefik BasicAuth，lifecycle 为 Server API key；逐沙箱 execd token 自动生成，无须用户传递。CA/凭据文件各≤16KiB，客户端强制 HTTPS、自定义 CA、TLS≥1.2、同源、禁止重定向/环境代理，进程仅获必要读取权限。

部署固定 Linux Docker/runsc、专用 worker、批准镜像/execd、PID64、1CPU/256MiB/写层128MiB、TTL120秒、命令10秒、整步30秒，无续期/降级。单实例串行，创建前检查专用 worker 存量小于2；首版一个后端实例/专用 worker，不自行横向扩副本。metadata 只核对实例身份/描述，不能证明 runtime/firewall。execd 必须启用 init 模式及 §22 内层凭据保护。server-proxy 维护模式按锁定版本和原试用边界使用，升级核对受影响契约。

计算和等待槽在整步预算内提前预留独立清理5秒、当前Commerce审计2秒及回执返回1秒，避免明确回执在外层期限耗尽后被丢弃。更早的调用者deadline优先；不足以执行时返回not_dispatched，不增加总时间。可信collector确认失败（如缺少结果、文件超限或安全文件检查拒绝）显示failed/invalid_result；缺失/冲突终态或断流仍为UNKNOWN。计算结果与cleanup completed/pending分别保留，均不自动重试。

开放前完成设计 §8.3.4 的其他宿主地址/IPv6/既有公网入口、共享网络调用者、真实后端挂载/路由、proxy鉴权/raw端口绕行等余项，以及 §22 实际 init/proc 拒绝证据；不能只凭公网连接失败、guest uname、合成401或旧 POC。记录适用镜像/profile/network 与完整证据后，才填写 trialEvidence；任意摘要不能替代验收，变更相应边界后核对受影响证据。

复用§23独立WSL证据时必须保持同一受信拓扑、镜像/execd/runtime与准入边界。该环境没有业务公网入口，backend角色为正式adapter验证进程；未来current-application真实挂载/路由/公网配置在目标部署中检查，不能把N/A或validation-only fixture值提升为PASS。init会剥离控制凭据，bootstrap的EXECD_ENVS配置路径可以存在；业务profile不提供Jupyter/业务凭据，每条脚本/collector仍空覆盖三个敏感环境变量。provider metadata中的descriptor摘要使用无padding Base32（52字符）满足上游label限制，Run/receipt完整SHA-256不变。

§24的保留应用组合另有实际后端挂载/路由增量记录，使用该组合自己的trialEvidence，未直接复用validation-only值。试用UI与身份/API入口仅本机回环；模型响应、usage及价目为明确的本地合成数据，无外部模型调用。保留环境使用正常登录和当前Skill/Run/Review/Catalog owner，操作入口、私密登录交接与按归属停止命令见worktree忽略目录`.local/issue1293-app/HANDOFF.md`。本机开发UI首次编译曾触及默认堆上限，当前以8GiB上限运行并保留独立缓存；UI重启后读取原提案，不重新执行脚本。

未配置无可执行 profile；不完整配置启动失败，不静默改成 runc/文本执行。撤下 profile 后原 Run/receipt 仍可查询，不自动恢复 Running/UNKNOWN。测试继续使用指定独立 WSL Engine/socket；停止自有资源与操作业务实例分别授权，不通用 prune、WSL shutdown 或删除业务卷。
