# Alertmanager WeCom Adapter

这套清单用于把 Alertmanager webhook 告警转成企业微信机器人 markdown 消息。

## 包含内容

- 轻量 Python webhook 适配器
- `AlertmanagerConfig`
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

原 task-processor namespace 的配置只接收：

- `service=amazon-crawler-api`

共享 Prometheus 存储保护由相邻 `prometheus-storage` 装配：规则与窄配置均位于 monitoring，只匹配 `service=monitoring` 且 `alertname=PrometheusStorageAtRisk`，接受默认 namespace 隔离；沿用本适配器、原 receiver URL、分组与超时，不新建程序或修改 Secret。旧 SHEIN 专属分支已退休，见[完整边界](../../../../docs/architecture/legacy-listing-operations-closure-retirement-boundary.md)。本批只更新源码，真实安装/发送/收信 NOT_RUN。

如果后续要扩展到其他服务，可以继续在 `matcher` 里追加，或拆分成新的 `AlertmanagerConfig`。
