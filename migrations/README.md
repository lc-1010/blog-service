# Database setup

The public frontend and the admin process are two Go services that share the
same MySQL database. The services do not create or migrate production tables
during startup.

## New local database

From the repository root, run the setup script. It prompts for the MySQL root
password and the password to assign to `blog_user`; passwords are not stored in
the repository. It creates the `blog` database, grants the local application
user access, and creates the initial tables:

```sh
./scripts/init_mysql.sh
```

The script can be pointed at a different MySQL server with `MYSQL_HOST`,
`MYSQL_PORT`, and `MYSQL_ROOT_USER`. It can be run again for the initial
schema; it does not alter existing table definitions.

The initial tables are `blog_tag`, `blog_article`, and `blog_article_tag`.
Both services use the database and credentials from `configs/config.yaml`.
`BLOG_DB_HOST`, `BLOG_DB_USERNAME`, `BLOG_DB_PASSWORD`, and `BLOG_DB_NAME`
environment variables override those database connection settings.

To import the SQLite demo dataset without mixing it with application content,
use:

```sh
BLOG_DB_PASSWORD='database-password' python3 scripts/import_sqlite_to_mysql.py
```

This creates and fills separate `demo_*` tables using
[`20261023_demo_dataset_mysql.sql`](./20261023_demo_dataset_mysql.sql). The
import is repeatable and leaves existing demo rows unchanged. These sample
orders and synthetic members are not production data or login accounts.

## Existing database from the original blog service

Back up the database and apply
[`20261020_article_content_and_tags.sql`](./20261020_article_content_and_tags.sql)
once:

```sh
mysqldump -u root -p blog > blog-backup.sql
mysql -u root -p blog < migrations/20261020_article_content_and_tags.sql
```

The upgrade adds the article body and the one-tag-per-article association
table. It does not guess labels for existing articles; assign associations
according to the content owner's decisions before relying on tag filtering.
Do not run the fresh-install schema as a replacement for this upgrade on an
existing database.

SQLite integration tests create tables in isolated in-memory databases with
GORM `AutoMigrate`; they do not execute either production MySQL script.
