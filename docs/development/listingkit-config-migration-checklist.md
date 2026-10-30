# ListingKit 旧全局配置退休说明

本页记录源码配置边界，依据 [#301 / #1017 完整退休设计](../architecture/legacy-http-auth-support-closure-retirement-boundary.md) §4.2。旧 allowlist 中间件及其全局 SMS 装配已退出；不再沿本页旧版本指引搬迁部署环境变量或启用旧授权门面。本轮不读取或改动既有配置、凭据、实例或数据，不增加内部试用前置。

## 当前 owner

- current-application 沿当前 Zitadel Verifier、身份、组织解析和 permission 准入；身份权限不能通过旧开关关闭。
- 当前私密 manifest 的 `listingKitAuthorization` 只配置平台管理员的主体/角色；core 的 `listingkit.platformAdminUsers/platformAdminRoles`、issuer/client/project、组织目录与成员邀请凭据 owner 保持。
- 当前 SMS 由 `ZITADEL_SMS_CONFIG_FILE` 的私密 JSON 与 [当前 SMS 装配](../../internal/app/httpapi/zitadel_sms.go) 提供；文件权限、HMAC、模板和过期值校验保持。旧全局 SMS 字段不是该 JSON 的兼容入口。
- 当前图像 admission、采集、Store 观察同步及 JSON/hash/版本不受本次退休影响。

## 已退休的输入

两条正式配置 loader 对以下输入按存在性拒绝，包括空值、false、null 和空列表/map；报错只包含键名，不包含配置值：

- `listingkit.zitadel.authorizationRequired`
- `listingkit.zitadel.allowedTenantIDs`
- `listingkit.zitadel.allowedUserIDs`
- `listingkit.zitadel.allowedRoles`
- `listingkit.zitadel.sms` 整个旧 subtree，含 signingKey、tencentSecretID、tencentSecretKey、tencentAppID、tencentSignName、tencentTemplateID、phoneVerificationExpiryMinutes 及任意专属尾部。

对应的 11 个 Primary 与 11 个 AutomaticEnv 名精确列在设计 §4.2，全部拒绝，不静默忽略，不把某一值覆盖成默认值。不得据此拒绝整个 LISTINGKIT/ZITADEL 前缀或当前 SMS 私密文件输入。

原无前缀别名 `LISTINGKIT_ZITADEL_ALLOWED_TENANT_IDS`、`LISTINGKIT_ZITADEL_ALLOWED_USER_IDS`、`LISTINGKIT_ZITADEL_ALLOWED_ROLES` 继续拒绝；对应 `TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_TENANT_IDS`、`TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_USER_IDS`、`TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_ROLES` 现在也属于退休输入，不再作为替代指引。

## 用户名配置仍仅作拒绝陷阱

`listingkit.zitadel.allowedUsernames`、`TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_USERNAMES` 与 `LISTINGKIT_ZITADEL_ALLOWED_USERNAMES` 继续沿原校验拒绝，不参与授权。当前权限归当前身份/组织与平台管理员 owner；不恢复用户名白名单，不新增 fallback、双读或数据迁移。

本次仅实施已确认源码退休；运行、部署与 #301 关闭均未授权，真实运行验收保持 NOT_RUN。
