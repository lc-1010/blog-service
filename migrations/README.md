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

The platform administration console also requires the user, role, settings,
site, and audit tables. Apply the platform migration after creating or upgrading
the blog schema:

```sh
mysql -u blog_user -p blog < migrations/20261023_platform_admin.sql
```

This migration is repeatable and seeds the built-in roles and safe site
settings. The `BLOG_ADMIN_USERNAME` / `BLOG_ADMIN_PASSWORD` account remains an
environment-configured unrestricted administrator; its session secret must
contain at least 32 bytes. Additional administrators are created in the console
with bcrypt-hashed passwords and role-scoped permissions. Apply the migration
before using platform management. The application does not auto-migrate
production databases.

The platform settings allowlist deliberately excludes database credentials and
session secrets. The `allow_registration` setting controls the member
registration endpoint; the maintenance-mode setting is not yet enforced by
request handling. The site registry stores site metadata and does not create
DNS, TLS certificates, or deploy applications.

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

## Member authentication

Apply [`20261023_membership.sql`](./20261023_membership.sql) before using the
public member APIs. It creates separate member, profile, and revocable session
tables and seeds registration as disabled. Enable `allow_registration` through
the platform settings page only when the registration policy is ready. Member
accounts are distinct from admin accounts and imported demo members.

SQLite integration tests create tables in isolated in-memory databases with
GORM `AutoMigrate`; they do not execute either production MySQL script.

## Article comments

After applying [`20261023_membership.sql`](./20261023_membership.sql), apply
[`20261024_article_comments.sql`](./20261024_article_comments.sql) before
using comment APIs. New comments are pending by default; administrators with
`content.manage` can approve (`state=1`) or reject (`state=2`) them. Only
approved comments on published articles are visible publicly.
