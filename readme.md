# blog

```
.
├── configs
├── docs
├── global
├── go.mod
├── go.sum
├── internal
│   ├── dao
│   ├── middleware
│   ├── modle
│   ├── routers
│   └── service
├── main.go
├── pkg
├── readme.md
├── scripts
├── storage
└── third_party
```
目录结构
- configs: 配置文件。
- docs:文档集合。
- global:全局变量。
- internal:内部模块。
  - dao:数据访问层 (Database Access Object),所有与数据相关的操作都会在 dao进行,例如 MySQL、Elasticsearch 等。
  - middleware: HTTP 中间件。
  - model:模型层,用于存放model 对象。
  - routers:路由相关的逻辑。
  - service:项目核心业务逻辑。
- pkg:项目相关的模块包。
- storage:项目生成的临时文件。
- scripts:各类构建、安装、分析等操作的脚本。
- third party:第三方的资源工具,如 Swagger Ul

## 数据库
- 标签管理
- 文章管理

```sql

CREATE DATABASE
IF NOT EXISTS blog_service DEFAULT CHARACTER 
SET utf8mb4 DEFAULT COLLATE utf8mb4_general_ci;

```

```sql
/* 公共字段*/
 `created_on` int(10) unsigned DEFAULT '0' COMMENT '创建时间',
    `created_by` varchar(100) DEFAULT '' COMMENT '创建人',
    `modified_on` int(10) unsigned DEFAULT '0' COMMNET '修改时间',
    `modified_by` varchar(100) DEFAULT '' COMMENT '修改人',
    `deleted_on` int(10) unsigned DEFAULT '0' COMMENT '删除时间',
    `is_del` tinyint(3) unsigned DEFAULT '0' COMMENT '是否删除 0 未删 ， 1 已删',
```
- blog_tag
```sql
CREATE TABLE `blog_tag` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `name` varchar(100) DEFAULT '' COMMENT '标签名',
    `created_on` int(10) unsigned DEFAULT '0' COMMENT '创建时间',
    `created_by` varchar(100) DEFAULT '' COMMENT '创建人',
    `modified_on` int(10) unsigned DEFAULT '0' COMMENT '修改时间',
    `modified_by` varchar(100) DEFAULT '' COMMENT '修改人',
    `deleted_on` int(10) unsigned DEFAULT '0' COMMENT '删除时间',
    `is_del` tinyint(3) unsigned DEFAULT '0' COMMENT '是否删除 0 未删 ， 1 已删', 
    `state` tinyint(3) unsigned DEFAULT '1' COMMENT '状态 0 禁用，1 启用',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='标签管理';
```
- blog_article 
```sql
CREATE TABLE `blog_article` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `title` varchar(100) default '' COMMENT '文章标题',
    `desc` varchar(255) default '' COMMENT '文章简述',
    `cover_image_url` varchar(255) default '' COMMENT '封面图片地址',
    `content` longtext COMMENT '文章内容',
    `created_on` int(10) unsigned DEFAULT '0' COMMENT '创建时间',
    `created_by` varchar(100) DEFAULT '' COMMENT '创建人',
    `modified_on` int(10) unsigned DEFAULT '0' COMMENT '修改时间',
    `modified_by` varchar(100) DEFAULT '' COMMENT '修改人',
    `deleted_on` int(10) unsigned DEFAULT '0' COMMENT '删除时间',
    `is_del` tinyint(3) unsigned DEFAULT '0' COMMENT '是否删除 0 未删 ， 1 已删',
     `state` tinyint(3) unsigned DEFAULT '1' COMMENT '状态 0 禁用，1 启用',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文章管理'; 
```

- blog_article_tag
```sql 
   CREATE TABLE `blog_article_tag` (
     `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
      `article_id` int(11)   NOT NULL COMMENT '文章ID',
       `tag_id` int(11)   NOT NULL COMMENT '文章ID',
        PRIMARY KEY (`id`)
   ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文章关联标签'; 
```


## 建立model 

每个表对应的model struct 结构
```
internal/modle
.
├── article.go
├── article_tag.go
├── model.go
└── tag.go
```

## 定义路由和handler 

```
.
├── api //handler
│   └── v1
│       ├── article.go
│       └── tag.go
└── router.go
```
`go run main.go` 
路由注册成功

```shell

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] POST   /api/v1/tags              --> blog-server/internal/routers/api/v1.Tag.Create-fm (3 handlers)
[GIN-debug] DELETE /api/v1/tags/:id          --> blog-server/internal/routers/api/v1.Tag.Delete-fm (3 handlers)
[GIN-debug] PUT    /api/v1/tags/:id          --> blog-server/internal/routers/api/v1.Tag.Update-fm (3 handlers)
[GIN-debug] PATCH  /api/v1/tags/:id/state    --> blog-server/internal/routers/api/v1.Tag.Update-fm (3 handlers)
[GIN-debug] GET    /api/v1/tags              --> blog-server/internal/routers/api/v1.Tag.List-fm (3 handlers)
[GIN-debug] POST   /api/v1/articles          --> blog-server/internal/routers/api/v1.Article.Create-fm (3 handlers)
[GIN-debug] GET    /api/v1/articles          --> blog-server/internal/routers/api/v1.Article.List-fm (3 handlers)
[GIN-debug] GET    /api/v1/articles/:id      --> blog-server/internal/routers/api/v1.Article.Get-fm (3 handlers)
[GIN-debug] PATCH  /api/v1/articles/:id/state --> blog-server/internal/routers/api/v1.Article.Update-fm (3 handlers)
[GIN-debug] PUT    /api/v1/articles/:id      --> blog-server/internal/routers/api/v1.Article.Update-fm (3 handlers)
[GIN-debug] DELETE /api/v1/articles/:id      --> blog-server/internal/routers/api/v1.Article.Delete-fm (3 handlers)

```

## 公共组建
- 错误
  - pkg/errcode 
- 配置
  - global
  - pkg/setting
- 日志写入
  - pkg/logger
- 数据库连接
- 响应处理


## swag
- go install github.com/swaggo/swag/cmd/swag@latest
