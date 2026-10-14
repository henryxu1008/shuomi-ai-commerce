# 硕米 Console

`web/listingkit-ui` 是当前前端应用的工程目录名。产品 UI、导航和页面归属以
[最终 UI / IA Authority](../../docs/product/final-ui-ia-authority.md) 为准。
当前事实、授权和副作用由各领域 owner 管理，前端不建立第二事实源。

## 启动

在此目录使用 `package.json` / `pnpm-lock.yaml` 固定的依赖：

```bash
pnpm install --frozen-lockfile
pnpm dev
```

正式构建检查：

```bash
pnpm build
```

开发时访问 `http://localhost:3000/` 为公开首页；`/workbench` 为当前 Console。
发布仍使用仓库既有的 standalone 镜像与启动接线；本批不改变部署方式。
按照已有 [Current Entrypoint Map](../../docs/development/repository-structure.md#current-entrypoint-map)
及对应领域运行文档配置服务端 origin、ZITADEL/Auth.js 和已开放能力。
配置与凭据通过本地私密环境提供；本目录没有需要复制的 `.env.example`。
缺少认证或领域能力配置时保持既有 denied/unavailable 状态。

## 当前入口

- `/`：公开产品首页。
- `/login`、`/invitations/**`：当前身份入口。
- `/workbench/**`：当前 Console，包括账户、商品来源、内容/图片、业务任务、Review、店铺等已接入路径。
- `/capture/1688`：现有受保护的浏览器采集接收页。

Figma 中的页面和菜单不表示所有底层能力已经开放；各能力继续遵循现有部署开关、授权与当前 owner 的真实状态。

## 旧投影退休

Issue #664 物理移除 `/listing-kits/**` 的 Task-first、Workspace、平台孤岛和独立后台页面，
以及专属组件、hooks、Task client/DTO/generated contracts、旧默认导航壳和
`/api/listing-kits/**` Task BFF/mock。旧登录页专属的 `/api/shein-login/**`、
`/api/sds-login/**` BFF 一并退出。旧 URL 不转发、不映射旧 Task、不返回兼容成功。

当前应用、Console、Auth.js/ZITADEL、通用 UI 组件和当前领域路径继续复用。
SDS 只读 BFF 共用的身份/header 转发行为由 `src/lib/server/upstream-identity.ts`
承接；当前来源/图片/Review/Store 的事实与执行 owner 未改变。
后端 root ListingKit、数字租户桥接和其它旧命令仍由 #29/#301/#529 的原范围处理。

这是源码交付说明，不表示保留实例已更新、真实 provider 已验证或用户/生产验收已通过。
详细决定见 [Legacy Register](../../docs/refactoring/legacy-register.md)。

## 开发检查

```bash
pnpm typecheck
pnpm lint
pnpm test
pnpm test:e2e
pnpm build
```

既有 Auth.js 服务端加密 session/JWT 和受保护 BFF 继续负责凭据传输；
不在浏览器 session payload 暴露 access/ID token，不接受浏览器伪造身份或企业。
