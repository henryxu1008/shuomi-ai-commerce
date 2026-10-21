# 监控运维说明

本目录定义受版本控制的监控目标状态；它不表示任一集群已经安装了这些组件。生产变更前必须先完成本文件的 preflight，并获得单独的生产写入授权。

监控能力由这几部分组成：

- `kube-prometheus-stack`
- Prometheus PVC 低空间告警及其同 namespace 窄路由
- 企业微信告警适配器
- Grafana 域名入口

## 访问与凭证

旧 TaskEvent/数字刊登专属监控源已按 [#839 完整闭包](../../../docs/architecture/legacy-listing-operations-closure-retirement-boundary.md)退休，不恢复旧兼容观察窗口。此源码交付不创建公网 Grafana 或 Alertmanager Ingress。安装后仅通过已获授权的内部访问方式或临时 `kubectl -n monitoring port-forward service/monitoring-grafana 3000:80` 访问 Grafana。管理员凭证只保留在 Kubernetes Secret 中，不在 Git、终端记录或发布证据中输出。

## 当前部署清单

- `deployments/kubernetes/monitoring/kube-prometheus-stack-values.yaml`
- `deployments/kubernetes/monitoring/prometheus-storage`
- `deployments/kubernetes/monitoring/alertmanager-wecom`
- `deployments/kubernetes/monitoring/grafana-ingress`
- `deployments/kubernetes/cert-manager/letsencrypt-prod`

## 固定版本与部署顺序

- Chart: `oci://ghcr.io/prometheus-community/charts/kube-prometheus-stack`
- Version: `88.3.0`
- OCI digest: `sha256:2b2f2c5c6f76ff661bdb34e6ae3410e01258f39632bffa52b94c0d42a1da0be6`
- Storage: default `local-path`; Prometheus 使用 30Gi RWO PVC，保留 21 天。

安装前必须确认存储类和节点资源：

```bash
kubectl get storageclass local-path
kubectl top node
```

在已获生产授权后执行：

```bash
helm upgrade --install monitoring oci://ghcr.io/prometheus-community/charts/kube-prometheus-stack \
  --version 88.3.0 \
  --namespace monitoring \
  --create-namespace \
  --atomic \
  --timeout 10m \
  -f deployments/kubernetes/monitoring/kube-prometheus-stack-values.yaml

kubectl apply -k deployments/kubernetes/monitoring/prometheus-storage
```

企业微信告警和公网 Grafana Ingress 是独立的既有运维决策；上面只列源码装配供已获授权的运维使用，本批没有执行。安装/实际投递/收信均 NOT_RUN，需要时另行授权应用对应清单，不成为内部试用前置。

## 企业微信告警

企业微信告警通过 `alertmanager-wecom` 适配器转发。

部署前需要先创建 Secret：

```bash
kubectl -n monitoring create secret generic alertmanager-wecom-secret \
  --from-literal=webhook-url='https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY'
```

当前默认只转发：

- `monitoring` namespace 中唯一 `service=monitoring`、`alertname=PrometheusStorageAtRisk` 的共享存储告警

原 task-processor namespace 的 Amazon 专属配置已退休；存储规则及窄 AlertmanagerConfig 位于 monitoring，沿默认 OnNamespace 隔离匹配，复用原接收器/分组/时长设置。固定 Chart 88.3.0 的原 selector 默认值保持；没有禁用全局 namespace enforcement。

## 当前监控覆盖

保留通用 Prometheus/Grafana 和 WeCom 适配器，以及 `monitoring/prometheus-storage` 的 PVC 保护规则。规则与路由同在 monitoring，只匹配 `service=monitoring` 和 `alertname=PrometheusStorageAtRisk`。旧数字 Amazon Service 的 PodMonitor、dashboard、规则及 task-processor 专属路由已在 #867 退出，不用无 matcher 的广泛路由替代。

## 当前已知注意点

- Grafana 使用 `local-path` 持久卷，因此 values 里关闭了 `initChownData`，避免滚动升级时权限修复失败
- 旧 TaskEvent 兼容观察、两个发布周期及14天窗口属于历史方案，不是已完成的运行验收，也不是当前试用前置。此前七条旧告警、两个 PodMonitor 和两份 dashboard 源已退休；通用存储规则及当前 Amazon/适配器/stack 保留。
