# 硕米智能引擎 / Shuomi AI Commerce Console

`web/listingkit-ui` 是当前前端应用的工程目录名。产品 UI、导航和页面归属以
[最终 UI / IA Authority](../../docs/product/final-ui-ia-authority.md) 为准。
当前事实、授权和副作用由各领域 owner 管理，前端不建立第二事实源。

产品品类为 **AI Commerce Agent Platform**。品牌名称依据 [#884](https://github.com/qq550723504/shuomi-ai-commerce/issues/884)；工程旧称 `task-processor`，实际目录与包名保持现状，见[当前源码地图](../../docs/development/repository-structure.md)。本次文档命名不改变页面文字、路由或 Figma 交互。

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

## 身份与领域边界

当前 Console 只消费领域 API；页面导航不产生商品事实或平台执行权限。SDS 只读 BFF 的可信身份/header 转发由 `src/lib/server/upstream-identity.ts` 负责。Auth.js / ZITADEL 和各领域继续核对当前 actor、企业与有效权限。

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
