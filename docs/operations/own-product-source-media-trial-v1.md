# 自有商品图片上传的隔离试用

设计：[来源图片试用边界](../architecture/own-product-source-media-trial-v1.md)。正式公网模式继续使用原`sourceMedia`；本文件仅用于用户授权的loopback本地试用实例，不用于真实云、共享或生产发布。

复用当前正常登录、Product collections、商业/资金owner及原容量schema/权限。先只读核实该独立实例实际六表/ACL，再继续；缺失时报告原owner，不能重跑非空Product初始化、重置schema/价格/权益或改变授权。来源上传开启后必须使用`storageCapacity:true`，默认每企业5GiB沿原合同；本轮不购买扩容。

保留原`trial.env`、base与本项目原overlay，再加入`deployments/docker/account-compose/docker-compose.source-media-trial.yml`。先构建该overlay使用的官方固定MinIO/mc服务，以及包含新合同的current-application/UI；不能用不识别新flag的旧程序启用上传。API/UI正常配置和私有登录文件不改变。

首次显式准备本项目来源图片：

1. 使用这些相同compose文件启动`source-media-objects`及其秘密卷初始化；它们只使用本项目新对象/密钥卷和同项目loopback namespace，不发布对象或控制台端口。
2. 用`--profile setup run --rm --no-deps source-media-init`安装本项目bucket的精确对象匿名GET政策；没有ListBucket/写入权限，原MinIO控制面保持私有。
3. 用`--profile setup run --rm --no-deps source-media-configure`更新当前私有manifest，原owners不变。helper只接受明确`ISOLATED_TRIAL_ONLY`、现Product collections/owners与未配或同一试用存储；不执行DDL或重建数据库。保存启用前manifest，不打印密钥。
4. 使用同一配置启动/重建`current-application`、`listingkit-ui`与`traefik`。backend固定配置`sourceMediaIsolatedTrial:true`、loopback S3 endpoint和精确localhost图片base；UI只设置`IMAGE_PROXY_SOURCE_MEDIA_TRIAL_BASE`，不打开`IMAGE_PROXY_ALLOW_PRIVATE`，TLS沿已有可信CA。

入口仍是 `/workbench/data/mine/own/new`：选择JPEG/PNG（每张≤3MiB）并点击“上传并添加图片链接”；图片加入公共池后，在对应SKU选择该图。正常保存返回自有库，打开独立详情；刷新、SKU选取和缩略图沿原图片代理显示准确对象。图片候选仍需在适配时批准，试用链接仅本机可见，不能以此宣称公网或平台可用。

上传结果不确定时保留原scope/hash/bytes。GET“核实原图片”只证明文件可读，不代表容量确认；若已可读或未找到，仍使用同一文件点击“重试原文件”，原Upload重放核实/确认容量后才清意图、加入图片。不换键、不换文件、不用GET写账本、不新增后台扫描。若同一文件仍未知，保持原意图和预留。

成功登记后，普通start只保留相同原owner、`storageCapacity:true`、配置和对象/密钥卷；namespace复用原登记，不重新要求已用prefix为空。stop只停止服务，不删除卷。上传而未保存商品的对象仍计量，不擅自删除候选。恢复/关闭新上传时保留容量true与全部对象/账本；不能恢复旧false manifest、换空owner、删表或清prefix。最小关闭新上传方法是移除SourceMedia配置和试用flag，保留同商业/资金owner及容量true；已有图片/商品仍保留。

实际候选、镜像、私有交接位置、浏览器执行与CI结果写Issue #675及当前PR，本文不保存滚动运行状态。实现者自检不等于用户/生产验收。
