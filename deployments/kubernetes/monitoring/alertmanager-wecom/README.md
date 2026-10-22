# Alertmanager WeCom Adapter

这套清单用于把 Alertmanager webhook 告警转成企业微信机器人 markdown 消息。

## 包含内容

- 轻量 Python webhook 适配器
- 存储保护的窄 `AlertmanagerConfig` 位于相邻 `prometheus-storage` 装配
- `Service`

## Secret

需要先创建：

```bash
kubectl -n monitoring create secret generic alertmanager-wecom-secret \
  --from-literal=webhook-url='https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY'
```

## 应用

```bash
kubectl apply -k deployments/kubernetes/monitoring/alertmanager-wecom
```

## 当前路由范围

#867 已退休 task-processor namespace 下只匹配旧 Amazon service 的专属 AlertmanagerConfig。共享适配器、Deployment/Service/ConfigMap 和 Secret 引用保留，本装配不注册无 matcher 的兜底路由。

共享 Prometheus 存储保护由相邻 `prometheus-storage` 装配：规则与窄配置同在 monitoring，只匹配 `service=monitoring` 且 `alertname=PrometheusStorageAtRisk`，保持默认 namespace 隔离；receiver URL、分组、超时及凭据引用不变。见[完整边界](../../../../docs/architecture/legacy-numeric-acquisition-tail-closure-retirement-boundary.md)。本轮只改源码，真实安装/发送/收信 NOT_RUN。
