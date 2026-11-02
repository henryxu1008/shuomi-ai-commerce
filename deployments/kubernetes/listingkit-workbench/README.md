# ListingKit Kubernetes 保留清单与历史操作

本目录保留 UI、imgproxy、命名空间与静态发布门禁源码；它不是当前应用的通用部署入口。**Production: NOT_READY**。当前程序启动与配置见 [Account Compose](../../docker/account-compose/README.md)、[模块说明](../../docker/account-compose/MODULES.md)和[镜像构建](../../../docs/operations/listingkit-image-build.md)；已有实例仍按其明确交付与授权维护。

## 保留的源码与边界

- [base/kustomization.yaml](base/kustomization.yaml)：保留命名空间、UI、imgproxy、Ingress/Traefik及其配置；文件存在不证明集群已部署或当前可发布。
- [UI声明](base/listingkit-ui-deployment.yaml)继续消费现有 Service API、Auth.js/ZITADEL 与当前企业身份；旧 API/Task/SDS BFF 不由本目录恢复。
- [imgproxy说明](imgproxy.md)、[声明](base/imgproxy-deployment.yaml)及[配置](base/configmap.yaml)保留原签名、S3读取、来源allowlist与映射。
- [release-authority](release-authority/kustomization.yaml)保留当前静态canary及固定拒绝门禁，不构成活跃发布者或部署授权。旧 API workload、发布/迁移Job、API/UI角色绑定、专属OIDC helper和SHEIN登录worker已按各自确认边界退出。
- [当前SHEIN退休入口](../../../docs/operations/README.md)和[Store观察合同](../../../docs/architecture/store-center-platform-observations-v1.md)说明各自有效owner；不能把源码退休扩大为停用实际资源或取消恢复责任。

原[API发布入口](../../../.github/workflows/listingkit-deploy.yml)和[UI发布入口](../../../.github/workflows/listingkit-ui-deploy.yml)现均固定拒绝执行并退出。它们保留为旧发布/回滚的拒绝门禁，Production NOT_READY；不能用于部署当前程序。

## 仍有效的Secret最小可见性

这里只列当前源码的键映射，不读取、复制或显示任何真实Secret值。[secret.example.yaml](base/secret.example.yaml)是原样例，不能据示例值推断已有环境。

UI仅按[声明](base/listingkit-ui-deployment.yaml)显式读取：

- `AUTH_SECRET`；
- `ZITADEL_ISSUER_URL`、`ZITADEL_CLIENT_ID`、`ZITADEL_CLIENT_SECRET`；
- `TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_TENANT_IDS`、`TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_USER_IDS`、`TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_ROLES`；
- `LISTINGKIT_DEMO_WEBHOOK_URL`。

公开origin/redirect等仍来自ConfigMap；UI不通过`envFrom`导入整个共享Secret，不接收数据库、目录写入或成员邀请写入凭据。

imgproxy继续显式读取签名键 `IMGPROXY_KEY`、`IMGPROXY_SALT`，以及两个仍有效的历史前缀S3键：

- `TASK_PROCESSOR_LISTINGKIT_IMAGE_UPLOAD_S3_ACCESS_KEY_ID`；
- `TASK_PROCESSOR_LISTINGKIT_IMAGE_UPLOAD_S3_SECRET_ACCESS_KEY`。

这两个S3键仍供当前imgproxy读取，名称中的旧前缀不构成删除理由。原Secret样例、声明及allowlist均不修改；不得把历史API/worker或迁移Job的凭据映射重新接入当前Pod。当前SMS配置沿原Account Compose的 `ZITADEL_SMS_CONFIG_FILE`，旧K8s短信/邀请provisioning步骤不再是当前操作入口。

## 当前运行与资源边界

当前 ImageAgent Go worker/store/Temporal 恢复、安全、Service API/Auth.js 及 Store 事实仍由现行 owner 承担；生产 NOT_READY 和当前发布拒绝策略保持。新系统安装不以旧 v2/v3 迁移、旧 drain 观察窗口或历史发布验收为前置。

源码退出不证明任何既有集群对象、IAM 权限、运行进程、workflow 或在途外部副作用已经结束。停止/重启、撤权、Secret 轮换、消息重放、数据清理和部署仍需对应资源的明确授权。

旧部署与恢复步骤不在当前工作树维护；需要追溯时见[固定版本原文](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/deployments/listingkit-workbench-k3s.md)。原记录不是当前部署、兼容或迁移授权。
