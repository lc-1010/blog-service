#!/bin/bash
set -eu

MYSQL_HOST=${MYSQL_HOST:-127.0.0.1}
MYSQL_PORT=${MYSQL_PORT:-3306}
MYSQL_ROOT_USER=${MYSQL_ROOT_USER:-root}
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
SCHEMA_FILE="$SCRIPT_DIR/../migrations/20261021_initial_blog_schema.sql"

if ! command -v mysql >/dev/null 2>&1; then
    echo "mysql client not found; install a MySQL client and retry." >&2
    exit 1
fi

if [ ! -r "$SCHEMA_FILE" ]; then
    echo "schema file not found: $SCHEMA_FILE" >&2
    exit 1
fi

read -r -s -p 'Enter the password to assign to MySQL user blog_user: ' APP_DB_PASSWORD
printf '\n' >&2

if [ -z "$APP_DB_PASSWORD" ]; then
    echo "The application database password must not be empty." >&2
    exit 1
fi

case "$APP_DB_PASSWORD" in
    *[!A-Za-z0-9@#._-]*)
        echo "Use only letters, numbers, and @ # . _ - in the password." >&2
        exit 1
        ;;
esac

{
    printf "CREATE DATABASE IF NOT EXISTS blog CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;\n"
    printf "CREATE USER IF NOT EXISTS 'blog_user'@'localhost' IDENTIFIED BY '%s';\n" "$APP_DB_PASSWORD"
    printf "ALTER USER 'blog_user'@'localhost' IDENTIFIED BY '%s';\n" "$APP_DB_PASSWORD"
    printf "CREATE USER IF NOT EXISTS 'blog_user'@'127.0.0.1' IDENTIFIED BY '%s';\n" "$APP_DB_PASSWORD"
    printf "ALTER USER 'blog_user'@'127.0.0.1' IDENTIFIED BY '%s';\n" "$APP_DB_PASSWORD"
    printf "GRANT ALL PRIVILEGES ON blog.* TO 'blog_user'@'localhost';\n"
    printf "GRANT ALL PRIVILEGES ON blog.* TO 'blog_user'@'127.0.0.1';\n"
    printf "USE blog;\n"
    cat "$SCHEMA_FILE"
} | mysql --protocol=TCP -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_ROOT_USER" -p

unset APP_DB_PASSWORD
printf 'Database blog and its initial tables are ready.\n'