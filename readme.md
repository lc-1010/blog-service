# Blog Service

基于 Go、Gin、GORM 和 MySQL 的博客系统，公开前台与后台管理以独立进程运行，共享数据库配置。

## 功能

- 前台：`/` 展示已发布文章，支持标题搜索、标签筛选、分页和文章详情。
- 前台服务：`/` 展示已发布文章，并提供公开文章、标签和会员 API。
- 后台服务：独立监听端口，`/admin/` 提供登录、文章与标签管理，以及平台总览、用户/角色权限、系统配置、站点登记和操作日志。
- 会员 API：支持可开关的注册、用户名登录、可撤销会话、个人资料与密码修改；新注册账号默认为普通会员。
- 评论：登录会员可对已发布文章发表评论，评论默认待审核；管理员可查看、通过或拒绝，公开端只展示已通过评论。
- API：公开 API 和管理 API 分别由两个进程提供；管理 API 需要管理员会话，并按角色权限校验平台和内容操作。
- 数据：生产服务连接 MySQL；SQLite 演示数据可按 [datasets/README.md](./datasets/README.md) 导入 MySQL 的隔离 `demo_*` 表，不会混入公开博客数据。

当前产品尚未包含评论举报、互动通知、商品/订单/支付或缓存等能力；会员 API 仍缺邮箱验证、找回密码和管理员会员处置界面。现有架构图、业务流程和功能缺口见 [系统架构与功能缺口](./docs/architecture-and-gaps.md)。

## 本地启动

要求 Go 1.20+ 和 MySQL。前台和后台是两个独立进程，共用 `blog` 数据库。首次初始化可运行脚本创建数据库用户及表：

```sh
./scripts/init_mysql.sh
```

脚本会分别提示 MySQL root 密码和 `blog_user` 的数据库密码；输入你设置的 `blog_user` 密码即可。密码不会写入仓库。默认连接用户为 `blog_user`、数据库为 `blog`。可通过 `BLOG_DB_HOST`、`BLOG_DB_USERNAME`、`BLOG_DB_PASSWORD`、`BLOG_DB_NAME` 环境变量覆盖配置。旧版数据库请先备份，再按 [迁移说明](./migrations/README.md) 执行升级脚本；应用启动不会自动创建或修改生产表。

配置后台凭据和至少 32 字节的会话签名密钥。分别启动公开服务和后台服务（可在两个终端运行）：

```sh
export BLOG_DB_PASSWORD='输入初始化时设置的数据库密码'
export BLOG_ADMIN_USERNAME='admin'
export BLOG_ADMIN_PASSWORD='replace-with-a-strong-password'
export BLOG_ADMIN_SESSION_SECRET='replace-with-at-least-32-random-bytes'
go run ./cmd/admin
```

另一个终端启动公开前台：

```sh
export BLOG_DB_PASSWORD='输入初始化时设置的数据库密码'
go run .
```

默认打开 `http://localhost:8000/` 访问前台、打开 `http://localhost:8001/admin/` 访问后台。`Server.HttpPort` 和 `Server.AdminHttpPort` 可在配置文件中分别调整。两进程共享 MySQL 数据库配置，但使用独立日志文件。后台环境变量缺失或会话密钥不足 32 字节时，后台认证接口会返回 `503`，而不是使用默认凭据。使用 HTTPS 部署时，登录会话 Cookie 会设置 `Secure`；反向代理应正确终止 TLS 并覆盖转发头 `X-Forwarded-Proto: https`。

平台管理功能需额外应用数据库迁移：`mysql -u blog_user -p blog < migrations/20261023_platform_admin.sql`。该迁移创建平台用户、角色、系统设置、站点登记和操作日志表，并初始化内置角色。详见 [迁移说明](./migrations/README.md)。新增管理账号由平台用户管理页面创建，密码以 bcrypt 哈希保存；环境变量管理员是保留的超级管理员入口。

会员 API 需应用 `mysql -u blog_user -p blog < migrations/20261023_membership.sql`。`allow_registration` 默认关闭，可在平台设置页面开启；会员会话令牌只以哈希形式保存，修改密码会撤销该会员全部会话。会员账号独立于管理员账号和 SQLite 演示会员。

评论 API 需先应用会员迁移，再运行 `mysql -u blog_user -p blog < migrations/20261024_article_comments.sql`。新评论状态为 `0`（待审核）；后台可将状态设为 `1`（通过）或 `2`（拒绝）。公开接口只返回已发布文章下状态为 `1` 的评论。

服务端口、超时、数据库连接池、日志路径等运行参数在 `configs/config.yaml` 中配置。当前程序从工作目录下的 `configs/config.yaml` 读取配置。

## API

公开接口由前台服务（默认 `http://localhost:8000`）提供。表中管理接口由后台服务（默认 `http://localhost:8001`）提供；除登录和登出外，均要求先登录 `/api/v1/admin/login`。认证使用 HttpOnly、SameSite Strict 的会话 Cookie。列表接口支持 `page`、`page_size`，文章支持 `title`、`tag_id`、`state` 筛选。

| 方法 | 路径 | 访问 | 用途 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/public/articles` | 公开 | 已发布文章列表 |
| `GET` | `/api/v1/public/articles/:id` | 公开 | 已发布文章详情 |
| `GET` | `/api/v1/public/tags` | 公开 | 已启用标签列表 |
| `GET` | `/api/v1/public/articles/:id/comments` | 公开 | 查询已发布文章下已通过的顶层评论；用 `parent_id` 查询回复 |
| `POST` | `/api/v1/public/articles/:id/comments` | 会员会话 | 创建待审核评论；可选 `parent_id` 回复 |
| `POST` | `/api/v1/public/members/register` | 公开 | 注册会员（需开启开放注册） |
| `POST` | `/api/v1/public/members/login` | 公开 | 会员登录并创建服务端会话 |
| `POST` | `/api/v1/public/members/logout` | 公开 | 撤销当前会员会话 |
| `GET` | `/api/v1/public/members/me` | 会员会话 | 获取当前账号与资料 |
| `PATCH` | `/api/v1/public/members/me/profile` | 会员会话 | 更新显示名、简介和头像 URL |
| `PATCH` | `/api/v1/public/members/me/password` | 会员会话 | 修改密码并撤销所有现存会话 |
| `GET` | `/api/v1/public/members/:username` | 公开 | 查看会员公开资料 |
| `POST` | `/api/v1/admin/login` | 公开 | 管理员登录 |
| `POST` | `/api/v1/admin/logout` | 公开 | 清除登录 Cookie |
| `GET` | `/api/v1/admin/me` | 会话 | 当前管理员 |
| `GET` | `/api/v1/admin/dashboard` | 会话 | 文章与标签统计 |
| `GET` | `/api/v1/admin/platform/dashboard` | `audit.read` | 平台用户、角色、站点与操作统计 |
| `GET`, `POST` | `/api/v1/admin/platform/users` | `users.manage` | 列出和创建管理账号 |
| `PUT` | `/api/v1/admin/platform/users/:id` | `users.manage` | 更新账号、角色、状态或密码 |
| `GET` | `/api/v1/admin/platform/members` | `users.manage` | 分页查询会员（不返回密码哈希） |
| `PATCH` | `/api/v1/admin/platform/members/:id` | `users.manage` | 设置会员/作者身份并启用或停用账号 |
| `GET` | `/api/v1/comments` | `content.manage` | 分页查询评论，支持 `article_id` 筛选 |
| `PATCH` | `/api/v1/comments/:id/state` | `content.manage` | 审核评论（0 待审、1 通过、2 拒绝） |
| `GET`, `POST` | `/api/v1/admin/platform/roles` | `roles.manage` 写；用户或角色管理读 | 查询和创建角色 |
| `PUT`, `DELETE` | `/api/v1/admin/platform/roles/:id` | `roles.manage` | 更新/删除角色；被用户使用或内置角色不可删除 |
| `GET`, `PUT` | `/api/v1/admin/platform/settings` | `settings.manage` | 查询和更新白名单系统设置 |
| `GET`, `POST` | `/api/v1/admin/platform/sites` | `sites.manage` | 查询和登记站点 |
| `PUT`, `DELETE` | `/api/v1/admin/platform/sites/:id` | `sites.manage` | 更新/删除站点登记 |
| `GET` | `/api/v1/admin/platform/audit-logs` | `audit.read` | 分页查询管理操作日志 |
| `GET`, `POST` | `/api/v1/articles` | 会话 | 查询、创建文章 |
| `GET`, `PUT`, `DELETE` | `/api/v1/articles/:id` | 会话 | 查询、全量更新、软删除文章 |
| `PATCH` | `/api/v1/articles/:id/state` | 会话 | 发布或转为草稿 |
| `GET`, `POST` | `/api/v1/tags` | 会话 | 查询、创建标签 |
| `PUT` | `/api/v1/tags/:id` | 会话 | 更新标签 |
| `PATCH` | `/api/v1/tags/:id/state` | 会话 | 启用或停用标签 |
| `DELETE` | `/api/v1/tags/:id` | 会话 | 软删除标签 |

`state=1` 表示已发布/启用，`state=0` 表示草稿/停用。管理列表可用 `all_states=true` 返回两种状态；公开接口始终只返回已发布文章和已启用标签。创建或更新文章时必须关联一个已启用标签。管理 API 的 OpenAPI 文档在后台服务的 `/swagger/index.html`。

默认角色为超级管理员（环境变量管理员）、平台管理员、内容编辑和审计员。新增数据库管理账号不能分配不受限的超级管理员权限；每次请求都会重新读取角色权限。管理 API 的写操作会记录操作者、路由、HTTP 方法、状态码和客户端 IP；不记录密码或请求正文。

系统配置中的“开放注册”会控制会员注册入口；站点名称、网址、描述、时区和维护模式目前仅保存为平台设置，尚未与公开站点运行行为联动。修改维护模式不会启用维护拦截。站点管理用于维护站点元数据，不会自动配置 DNS、TLS 或部署站点。

## 数据库迁移

生产建表和迁移脚本位于 [migrations/](./migrations/)。文章关联表约束每篇文章最多关联一个标签。升级旧数据库时，迁移会添加文章正文并创建关联表，但不会猜测已有文章应归属哪个标签；平台管理表由独立迁移脚本创建。请按 [迁移说明](./migrations/README.md) 顺序备份并执行迁移。

SQLite 演示数据库由 `datasets/import_datasets.py` 单独生成，不是生产数据库，也不会被服务自动读取或迁移。

## 验证

```sh
go test ./...
python3 -m unittest discover -s datasets -p 'test_*.py' -v
```

Go 集成测试在隔离的内存 SQLite 数据库中覆盖文章/标签 DAO、业务层、API、公开内容过滤、后台认证和嵌入式页面，不需要启动 MySQL。Python 测试使用合成小样本，不访问网络或真实顾客标识。

需要生成或重建本地演示数据时：

```sh
python3 datasets/import_datasets.py --db datasets/blog-demo.sqlite --max-rows 10000
```

默认不会覆盖已有数据库；确认需要重建时使用 `--replace`。来源、许可、字段映射和隐私处理说明见 [datasets/README.md](./datasets/README.md)。
