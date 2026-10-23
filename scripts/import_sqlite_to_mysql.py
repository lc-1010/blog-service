#!/usr/bin/env python3
"""Copy the normalized SQLite demo dataset into isolated MySQL demo_* tables."""

from __future__ import annotations

import argparse
import getpass
import os
import shutil
import sqlite3
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_DB = ROOT / "datasets" / "blog-demo.sqlite"
SCHEMA_FILE = ROOT / "migrations" / "20261023_demo_dataset_mysql.sql"
TABLES = (
    ("members", ("id", "username", "display_name", "member_type", "source_dataset", "created_at")),
    ("author_profiles", ("member_id", "bio")),
    ("categories", ("id", "name", "category_type")),
    ("articles", (
        "id", "author_id", "title", "summary", "content", "status",
        "content_origin", "source_dataset", "source_record_id",
    )),
    ("article_categories", ("article_id", "category_id")),
    ("products", ("id", "source_code", "title", "currency")),
    ("orders", (
        "id", "member_id", "occurred_at", "status", "currency",
        "source_dataset", "source_record_id",
    )),
    ("order_items", (
        "id", "order_id", "product_id", "quantity", "unit_price_minor",
        "line_total_minor", "source_row",
    )),
)
COUNTS_QUERY = "\n".join(
    f"SELECT '{table}', COUNT(*) FROM demo_{table};" for table, _ in TABLES
)


def sql_value(value: object) -> str:
    if value is None:
        return "NULL"
    if isinstance(value, int):
        return str(value)
    if isinstance(value, float):
        raise ValueError("floating point values are not supported in the demo schema")
    if isinstance(value, bytes):
        encoded = value.hex()
    else:
        encoded = str(value).encode("utf-8").hex()
    return f"CONVERT(X'{encoded}' USING utf8mb4)"


def build_insert(table: str, columns: tuple[str, ...], rows: list[tuple]) -> str:
    if not rows:
        return ""
    names = ", ".join(f"`{column}`" for column in columns)
    values = ",\n".join(
        "(" + ", ".join(sql_value(value) for value in row) + ")"
        for row in rows
    )
    return f"INSERT IGNORE INTO `demo_{table}` ({names}) VALUES\n{values};\n"


def build_import_sql(source: sqlite3.Connection, batch_size: int) -> tuple[str, dict[str, int]]:
    statements = [SCHEMA_FILE.read_text(encoding="utf-8"), "\nSTART TRANSACTION;\n"]
    counts: dict[str, int] = {}
    for table, columns in TABLES:
        quoted_columns = ", ".join(f'"{column}"' for column in columns)
        cursor = source.execute(f'SELECT {quoted_columns} FROM "{table}"')
        count = 0
        while True:
            rows = cursor.fetchmany(batch_size)
            if not rows:
                break
            statements.append(build_insert(table, columns, rows))
            count += len(rows)
        counts[table] = count
    statements.extend(("COMMIT;\n", COUNTS_QUERY, "\n"))
    return "".join(statements), counts


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--sqlite", type=Path, default=DEFAULT_DB)
    parser.add_argument("--host", default=os.environ.get("BLOG_DB_HOST", "127.0.0.1"))
    parser.add_argument("--port", type=int, default=3306)
    parser.add_argument("--user", default=os.environ.get("BLOG_DB_USERNAME", "blog_user"))
    parser.add_argument("--database", default=os.environ.get("BLOG_DB_NAME", "blog"))
    parser.add_argument("--batch-size", type=int, default=250)
    args = parser.parse_args()

    if not args.sqlite.is_file():
        parser.error(f"SQLite database does not exist: {args.sqlite}")
    if args.batch_size < 1 or args.batch_size > 1000:
        parser.error("--batch-size must be between 1 and 1000")
    mysql = shutil.which("mysql")
    if mysql is None:
        parser.error("mysql client not found; install a MySQL client and retry")

    password = os.environ.get("BLOG_DB_PASSWORD")
    if password is None:
        password = getpass.getpass(f"MySQL password for {args.user}@{args.host}: ")

    try:
        with sqlite3.connect(f"file:{args.sqlite.resolve()}?mode=ro", uri=True) as source:
            sql, counts = build_import_sql(source, args.batch_size)
    except (OSError, sqlite3.Error, ValueError) as error:
        print(f"Unable to read SQLite demo database: {error}", file=sys.stderr)
        return 1

    command = [
        mysql, "--protocol=TCP", "--default-character-set=utf8mb4",
        "--batch", "--raw", "--skip-column-names",
        "-h", args.host, "-P", str(args.port), "-u", args.user, args.database,
    ]
    env = os.environ.copy()
    env["MYSQL_PWD"] = password
    try:
        result = subprocess.run(
            command,
            input=sql,
            text=True,
            encoding="utf-8",
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            env=env,
            check=False,
        )
    except OSError as error:
        print(f"Unable to run mysql client: {error}", file=sys.stderr)
        return 1

    if result.returncode:
        print(result.stderr.strip() or "MySQL demo import failed.", file=sys.stderr)
        return result.returncode

    print("SQLite source rows imported (existing matching demo_* rows are left unchanged):")
    for table, count in counts.items():
        print(f"  demo_{table}: {count}")
    print("MySQL destination row counts:")
    print(result.stdout.strip())
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
