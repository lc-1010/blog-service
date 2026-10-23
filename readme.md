# Blog Service

基于 Go、Gin、GORM 和 MySQL 的博客系统，公开前台与后台管理以独立进程运行，共享数据库配置。

## 功能

- 前台：`/` 展示已发布文章，支持标题搜索、标签筛选、分页和文章详情。
- 前台服务：`/` 展示已发布文章，只提供公开只读 API。
- 后台服务：独立监听端口，`/admin/` 提供登录、数据概览、文章 CRUD/发布状态管理和标签管理。
- API：公开 API 和管理 API 分别由两个进程提供；管理 API 需要管理员会话。
- 数据：生产服务连接 MySQL；SQLite 演示数据可按 [datasets/README.md](./datasets/README.md) 导入 MySQL 的隔离 `demo_*` 表，不会混入公开博客数据。

当前产品尚未包含会员、评论、商品/订单/支付或缓存等能力。现有架构图、业务流程和功能缺口见 [系统架构与功能缺口](./docs/architecture-and-gaps.md)。

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

服务端口、超时、数据库连接池、日志路径等运行参数在 `configs/config.yaml` 中配置。当前程序从工作目录下的 `configs/config.yaml` 读取配置。

## API

公开接口由前台服务（默认 `http://localhost:8000`）提供。表中管理接口由后台服务（默认 `http://localhost:8001`）提供；除登录和登出外，均要求先登录 `/api/v1/admin/login`。认证使用 HttpOnly、SameSite Strict 的会话 Cookie。列表接口支持 `page`、`page_size`，文章支持 `title`、`tag_id`、`state` 筛选。

| 方法 | 路径 | 访问 | 用途 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/public/articles` | 公开 | 已发布文章列表 |
| `GET` | `/api/v1/public/articles/:id` | 公开 | 已发布文章详情 |
| `GET` | `/api/v1/public/tags` | 公开 | 已启用标签列表 |
| `POST` | `/api/v1/admin/login` | 公开 | 管理员登录 |
| `POST` | `/api/v1/admin/logout` | 公开 | 清除登录 Cookie |
| `GET` | `/api/v1/admin/me` | 会话 | 当前管理员 |
| `GET` | `/api/v1/admin/dashboard` | 会话 | 文章与标签统计 |
| `GET`, `POST` | `/api/v1/articles` | 会话 | 查询、创建文章 |
| `GET`, `PUT`, `DELETE` | `/api/v1/articles/:id` | 会话 | 查询、全量更新、软删除文章 |
| `PATCH` | `/api/v1/articles/:id/state` | 会话 | 发布或转为草稿 |
| `GET`, `POST` | `/api/v1/tags` | 会话 | 查询、创建标签 |
| `PUT` | `/api/v1/tags/:id` | 会话 | 更新标签 |
| `PATCH` | `/api/v1/tags/:id/state` | 会话 | 启用或停用标签 |
| `DELETE` | `/api/v1/tags/:id` | 会话 | 软删除标签 |

`state=1` 表示已发布/启用，`state=0` 表示草稿/停用。管理列表可用 `all_states=true` 返回两种状态；公开接口始终只返回已发布文章和已启用标签。创建或更新文章时必须关联一个已启用标签。管理 API 的 OpenAPI 文档在后台服务的 `/swagger/index.html`。

## 数据库迁移

生产建表和迁移脚本位于 [migrations/](./migrations/)。文章关联表约束每篇文章最多关联一个标签。升级旧数据库时，迁移会添加文章正文并创建关联表，但不会猜测已有文章应归属哪个标签；需要按业务规则为既有文章补充标签。脚本只应对备份后的数据库执行一次。

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
