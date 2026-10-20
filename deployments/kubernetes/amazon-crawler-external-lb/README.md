# Amazon Crawler External LB 部署说明

这套清单用于在 K8s 集群内暴露一组集群外部的 Amazon crawler API 节点，供获准的 crawler 消费者通过集群内稳定地址访问。

对应 Service 名称：

- [service.yaml](D:/code/task-processor/deployments/kubernetes/amazon-crawler-external-lb/service.yaml)

对应后端节点清单：

- [endpointslice.yaml](D:/code/task-processor/deployments/kubernetes/amazon-crawler-external-lb/endpointslice.yaml)

## 作用

部署完成后，集群内可以通过这个地址访问外部 crawler：

`http://amazon-crawler-external-lb.task-processor.svc.cluster.local:8080`

旧 `amazon-listing` 消费者及其专属部署源码已随 #822 退休；本清单的 Service/EndpointSlice 和当前来源采集能力保持，本文不授权部署或运行操作。

## 部署前要改的只有一件事

编辑：

- [endpointslice.yaml](D:/code/task-processor/deployments/kubernetes/amazon-crawler-external-lb/endpointslice.yaml)

把里面的外部 crawler IP 改成你的真实节点列表。

注意：

- 这里只能写 IP，不能写域名
- 端口统一在 `ports.port` 配成 `8080`
- 新增 crawler 节点时，只要继续追加一个 `endpoints` 条目即可

## 部署命令

```bash
kubectl apply -k deployments/kubernetes/amazon-crawler-external-lb
```

## 部署后检查

```bash
kubectl -n task-processor get svc,endpointslice -l kubernetes.io/service-name=amazon-crawler-external-lb
```

正常情况下应该看到：

- `Service` 已创建
- `EndpointSlice` 里有你配置的外部 IP

## 连通性验证

建议从集群内起一个临时 Pod 测试：

```bash
kubectl -n task-processor run curl-test --rm -it --image=curlimages/curl -- \
  curl -sS http://amazon-crawler-external-lb.task-processor.svc.cluster.local:8080/health
```

如果 crawler API 没有 `/health`，就改成你当前可用的探测接口。

## 已退休的旧 Amazon 刊登消费者

#822 已删除旧 `amazon-listing` 配置/Secret 模板与部署入口，不再作为本 LB 的当前消费者或部署顺序。历史关联不能成为复活旧消费者的依据；本轮没有访问、修改或删除任何实际 Service、EndpointSlice、Secret、namespace 或外部 crawler。
