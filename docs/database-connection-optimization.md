# 数据库连接池优化 - 方案7实施总结（历史）

> Status: HISTORICAL / 非当前运维 runbook。本文保留旧数字刊登方案的工程记录、数量及效果描述，不是本轮实测或当前配置合同。旧 SHEIN workload、推荐 apply/rollout 和验证步骤已按 [#839 完整退休边界](architecture/legacy-listing-operations-closure-retirement-boundary.md)退出；不得据此操作实例、数据库或凭据。

## 问题背景

shein-listing在K8s中启动了32个pod，每个pod默认10个数据库连接，导致最多320个数据库连接，造成数据库压力过大。

## 解决方案

采用**应用层连接代理 + 环境变量配置**的方式，通过代码层面控制每个pod的数据库连接数。

## 实施内容

### 1. 核心代码实现

#### 文件: `internal/infra/database/connection_proxy.go`
- 创建ConnectionProxy结构，使用信号量控制并发DB操作
- 提供Execute方法包装所有数据库调用
- 支持统计监控和超时控制

#### 文件: `internal/infra/database/postgres_db.go`
- 增强NewDatabaseFromConfig函数，支持从环境变量读取配置
- 新增环境变量支持：
  - `DATABASE_MAX_CONNECTIONS`: 最大数据库连接数
  - `DATABASE_MAX_IDLE_CONNECTIONS`: 最大空闲连接数
- 优先级：配置文件 > 环境变量 > 默认值

### 2. K8s配置更新

#### 文件: `deployments/kubernetes/shein-listing/base/deployment.yaml`
添加环境变量：
```yaml
env:
  - name: DATABASE_MAX_CONNECTIONS
    value: "3"
  - name: DATABASE_MAX_IDLE_CONNECTIONS
    value: "2"
```

## 效果对比

### 优化前
- Pod数量: 32
- 每Pod连接数: 10
- **总连接数: 32 × 10 = 320**

### 优化后（仅调整连接数）
- Pod数量: 32
- 每Pod连接数: 3
- **总连接数: 32 × 3 = 96** (减少70%)

### 进一步优化（配合HPA限制）
如果同时调整HPA限制pod数量为10：
- Pod数量: 10
- 每Pod连接数: 3
- **总连接数: 10 × 3 = 30** (减少90%)

## 当前运行入口与限制

旧刊登 Deployment 已不在当前源码中，保留的 shein-listing prod 装配只声明共享 Namespace/凭据，不能用于启动或滚动更新旧刊登服务。原推荐 apply/rollout、修改旧刊登配置后重建部署、exec/logs/数据库连接查询等验证流程已退出，历史步骤仅见 Git 历史。

当前采集、Store及独立 SHEIN 登录/Store 同步沿[仓库入口地图](development/repository-structure.md)使用；配置、数据库和运行 owner 不转移。本轮没有运行环境或数据库验收，不增加试用前置。

## 历史注意事项（不作为当前操作步骤）

1. **渐进式调整**: 建议先从10降到5，观察稳定后再降到3
2. **监控告警**: 设置数据库连接数告警阈值（如80%）
3. **性能测试**: 调整后需要进行压力测试，确保不影响业务
4. **回滚方案**: 如果出现问题，可以快速调整环境变量值回滚

## 历史优化建议（不构成本轮工作）

1. **引入PgBouncer**: 如果需要更多pod但保持低连接数，可以考虑部署PgBouncer连接池中间件
2. **读写分离**: 将读操作分流到从库
3. **缓存优化**: 加强Redis缓存使用，减少数据库查询
4. **批量操作**: 优化代码中的N+1查询问题

## 历史文件清单（按当时路径记录）

- `internal/infra/database/connection_proxy.go` - 连接代理核心实现
- `internal/infra/database/connection_proxy_test.go` - 单元测试
- `internal/infra/database/postgres_db.go` - 数据库连接增强
- `deployments/kubernetes/shein-listing/base/deployment.yaml` - K8s部署配置

## 历史技术细节

### 环境变量优先级
```
1. 配置文件中的 max_connections/max_idle_connections
2. 环境变量 DATABASE_MAX_CONNECTIONS/DATABASE_MAX_IDLE_CONNECTIONS
3. 默认值 (10/5)
```

### 代码实现逻辑
```go
// postgres_db.go 中的逻辑
maxConn := cfg.MaxConnections
if maxConn <= 0 {
    // 尝试从环境变量读取
    if envMax := os.Getenv("DATABASE_MAX_CONNECTIONS"); envMax != "" {
        if parsed, err := strconv.Atoi(envMax); err == nil && parsed > 0 {
            maxConn = parsed
        } else {
            maxConn = 10  // 默认值
        }
    } else {
        maxConn = 10  // 默认值
    }
}
```

## 联系信息

如有问题，请联系开发团队。

---
更新日期: 2026-06-08
