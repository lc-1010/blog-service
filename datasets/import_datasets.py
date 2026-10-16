#!/usr/bin/env python3
"""Import public demo datasets into a normalized SQLite database."""

from __future__ import annotations

import argparse
import csv
import hashlib
import io
import json
import re
import sqlite3
import sys
import urllib.request
from contextlib import closing
from datetime import datetime
from decimal import Decimal, ROUND_HALF_UP
from pathlib import Path
from typing import Iterable, TextIO

ROOT = Path(__file__).resolve().parent
UCI_URL = "https://archive.ics.uci.edu/static/public/352/data.csv"
TANG_URL = (
    "https://raw.githubusercontent.com/chinese-poetry/chinese-poetry/"
    "d651e494e0ecb49555f9f473aa7c97ed0e4b53b7/"
    "%E5%85%A8%E5%94%90%E8%AF%97/%E5%94%90%E8%AF%97%E4%B8%89%E7%99%BE%E9%A6%96.json"
)
RETAIL_SOURCE = "uci-online-retail-352"
TANG_SOURCE = "chinese-poetry-tang-300"
SYNTHETIC_SOURCE = "synthetic-retail-posts"
COLLECTION_TAG = "唐诗三百首"
FORM_MARKERS = ("绝句", "律诗", "排律", "古风", "乐府", "曲辞", "词牌")


def stable_id(prefix: str, value: str) -> str:
    digest = hashlib.sha256(value.encode("utf-8")).hexdigest()[:20]
    return f"{prefix}-{digest}"


def create_schema(connection: sqlite3.Connection) -> None:
    connection.executescript((ROOT / "schema.sql").read_text(encoding="utf-8"))
    connection.execute("PRAGMA foreign_keys = ON")


def read_csv_rows(stream: TextIO, max_rows: int) -> Iterable[tuple[int, dict[str, str]]]:
    reader = csv.DictReader(stream)
    for row_number, row in enumerate(reader, start=1):
        if row_number > max_rows:
            break
        yield row_number, row


def import_retail_rows(
    connection: sqlite3.Connection,
    rows: Iterable[tuple[int, dict[str, str]]],
    max_rows: int = 10_000,
) -> dict[str, int]:
    customer_ids: dict[str, str] = {}
    order_ids: dict[str, str] = {}
    first_product_by_customer: dict[str, str] = {}
    counts = {"rows_read": 0, "rows_imported": 0, "rows_skipped": 0}

    for row_number, row in rows:
        if counts["rows_read"] >= max_rows:
            break
        counts["rows_read"] += 1
        customer_source_id = (row.get("CustomerID") or "").strip()
        if not customer_source_id:
            counts["rows_skipped"] += 1
            continue

        customer_source_id = re.sub(r"\.0$", "", customer_source_id)
        member_id = customer_ids.get(customer_source_id)
        if member_id is None:
            member_id = f"retail-member-{len(customer_ids) + 1:06d}"
            customer_ids[customer_source_id] = member_id
            connection.execute(
                """INSERT OR IGNORE INTO members
                   (id, username, display_name, member_type, source_dataset)
                   VALUES (?, ?, ?, 'customer', ?)""",
                (member_id, member_id, f"Retail Customer {len(customer_ids):06d}", RETAIL_SOURCE),
            )

        invoice_source_id = (row.get("InvoiceNo") or "").strip()
        if not invoice_source_id:
            counts["rows_skipped"] += 1
            continue
        order_id = order_ids.get(invoice_source_id)
        if order_id is None:
            order_id = f"retail-order-{len(order_ids) + 1:07d}"
            order_ids[invoice_source_id] = order_id
            invoice_date = datetime.strptime(
                (row.get("InvoiceDate") or "").strip(), "%m/%d/%Y %H:%M"
            ).isoformat(timespec="minutes")
            status = "cancelled" if invoice_source_id.upper().startswith("C") else "completed"
            connection.execute(
                """INSERT OR IGNORE INTO orders
                   (id, member_id, occurred_at, status, source_dataset, source_record_id)
                   VALUES (?, ?, ?, ?, ?, ?)""",
                (order_id, member_id, invoice_date, status, RETAIL_SOURCE, invoice_source_id),
            )

        product_code = (row.get("StockCode") or "").strip()
        if not product_code:
            counts["rows_skipped"] += 1
            continue
        product_id = stable_id("retail-product", product_code)
        product_title = (row.get("Description") or "").strip() or product_code
        connection.execute(
            """INSERT OR IGNORE INTO products (id, source_code, title)
               VALUES (?, ?, ?)""",
            (product_id, product_code, product_title),
        )
        if member_id not in first_product_by_customer:
            first_product_by_customer[member_id] = product_title

        quantity = int((row.get("Quantity") or "0").strip())
        unit_price = Decimal((row.get("UnitPrice") or "0").strip())
        unit_price_minor = int((unit_price * 100).quantize(Decimal("1"), rounding=ROUND_HALF_UP))
        connection.execute(
            """INSERT OR IGNORE INTO order_items
               (order_id, product_id, quantity, unit_price_minor, line_total_minor, source_row)
               VALUES (?, ?, ?, ?, ?, ?)""",
            (order_id, product_id, quantity, unit_price_minor, quantity * unit_price_minor, row_number),
        )
        counts["rows_imported"] += 1

    for member_id, product_title in first_product_by_customer.items():
        article_id = stable_id(SYNTHETIC_SOURCE, member_id)
        connection.execute(
            """INSERT OR IGNORE INTO articles
               (id, author_id, title, summary, content, content_origin, source_dataset)
               VALUES (?, ?, ?, ?, ?, 'synthetic', ?)""",
            (
                article_id,
                member_id,
                f"演示分享：{product_title[:80]}",
                "由公开零售商品目录自动生成的合成演示内容。",
                f"此内容为合成演示数据，依据公开商品名称“{product_title}”生成；"
                "不是实际顾客发布或评价，不代表真实购买者观点。",
                SYNTHETIC_SOURCE,
            ),
        )

    counts["members"] = len(customer_ids)
    counts["orders"] = len(order_ids)
    counts["synthetic_articles"] = len(first_product_by_customer)
    return counts


def import_poems(connection: sqlite3.Connection, poems: list[dict]) -> dict[str, int]:
    authors = sorted({(poem.get("author") or "").strip() for poem in poems} - {""})
    author_ids = {name: stable_id("poet", name) for name in authors}
    for name, member_id in author_ids.items():
        connection.execute(
            """INSERT OR IGNORE INTO members
               (id, username, display_name, member_type, source_dataset)
               VALUES (?, ?, ?, 'author', ?)""",
            (member_id, member_id, name, TANG_SOURCE),
        )
        connection.execute(
            "INSERT OR IGNORE INTO author_profiles (member_id, bio) VALUES (?, '')",
            (member_id,),
        )

    category_names: set[str] = set()
    for poem in poems:
        category_names.update(
            tag.strip() for tag in poem.get("tags", [])
            if tag and tag.strip() and tag.strip() != COLLECTION_TAG
        )
    category_ids = {name: stable_id("poetry-category", name) for name in category_names}
    for name, category_id in category_ids.items():
        if any(marker in name for marker in FORM_MARKERS):
            category_type = "form"
        elif "年级" in name or "古诗" in name or "课外" in name:
            category_type = "classification"
        else:
            category_type = "topic"
        connection.execute(
            """INSERT OR IGNORE INTO categories (id, name, category_type)
               VALUES (?, ?, ?)""",
            (category_id, name, category_type),
        )

    for poem in poems:
        source_id = str(poem.get("id") or "").strip()
        author = (poem.get("author") or "").strip()
        title = (poem.get("title") or "").strip()
        paragraphs = poem.get("paragraphs") or []
        if not source_id or not author or not title or not paragraphs:
            continue
        article_id = f"tang-poem-{source_id}"
        content = "\n".join(str(part).strip() for part in paragraphs if str(part).strip())
        connection.execute(
            """INSERT OR IGNORE INTO articles
               (id, author_id, title, content, content_origin, source_dataset, source_record_id)
               VALUES (?, ?, ?, ?, 'dataset', ?, ?)""",
            (article_id, author_ids[author], title, content, TANG_SOURCE, source_id),
        )
        for tag in poem.get("tags", []):
            tag = (tag or "").strip()
            category_id = category_ids.get(tag)
            if category_id is not None:
                connection.execute(
                    """INSERT OR IGNORE INTO article_categories (article_id, category_id)
                       VALUES (?, ?)""",
                    (article_id, category_id),
                )

    return {
        "poems": len(poems),
        "authors": len(authors),
        "categories": len(category_names),
    }


def fetch_bytes(url: str) -> bytes:
    request = urllib.request.Request(url, headers={"User-Agent": "blog-service-dataset-import/1.0"})
    with urllib.request.urlopen(request, timeout=60) as response:
        return response.read()


def import_retail_url(connection: sqlite3.Connection, max_rows: int) -> dict[str, int]:
    request = urllib.request.Request(
        UCI_URL, headers={"User-Agent": "blog-service-dataset-import/1.0"}
    )
    with closing(urllib.request.urlopen(request, timeout=60)) as response:
        stream = io.TextIOWrapper(response, encoding="utf-8-sig", newline="")
        return import_retail_rows(connection, read_csv_rows(stream, max_rows), max_rows)


def drop_schema(connection: sqlite3.Connection) -> None:
    connection.executescript(
        """PRAGMA foreign_keys = OFF;
           DROP TABLE IF EXISTS order_items;
           DROP TABLE IF EXISTS orders;
           DROP TABLE IF EXISTS products;
           DROP TABLE IF EXISTS article_categories;
           DROP TABLE IF EXISTS articles;
           DROP TABLE IF EXISTS categories;
           DROP TABLE IF EXISTS author_profiles;
           DROP TABLE IF EXISTS members;
           PRAGMA foreign_keys = ON;"""
    )


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--db", type=Path, default=ROOT / "blog-demo.sqlite")
    parser.add_argument("--max-rows", type=int, default=10_000)
    parser.add_argument("--retail-csv", type=Path, help="Use a previously downloaded UCI CSV")
    parser.add_argument("--tang-json", type=Path, help="Use a local Tang poems JSON file")
    parser.add_argument("--replace", action="store_true", help="Replace an existing database")
    args = parser.parse_args()
    if args.max_rows <= 0:
        parser.error("--max-rows must be greater than zero")
    if args.db.exists() and not args.replace:
        parser.error(f"{args.db} already exists; choose another path or pass --replace")

    args.db.parent.mkdir(parents=True, exist_ok=True)
    connection = sqlite3.connect(args.db)
    connection.execute("PRAGMA foreign_keys = ON")
    try:
        if args.replace:
            drop_schema(connection)
        create_schema(connection)
        with connection:
            if args.retail_csv:
                with args.retail_csv.open(encoding="utf-8-sig", newline="") as stream:
                    retail_counts = import_retail_rows(
                        connection, read_csv_rows(stream, args.max_rows), args.max_rows
                    )
            else:
                retail_counts = import_retail_url(connection, args.max_rows)

            if args.tang_json:
                poems = json.loads(args.tang_json.read_text(encoding="utf-8"))
            else:
                poems = json.loads(fetch_bytes(TANG_URL).decode("utf-8"))
            tang_counts = import_poems(connection, poems)
        print(json.dumps({"retail": retail_counts, "tang": tang_counts, "database": str(args.db)}, ensure_ascii=False, indent=2))
    except Exception as error:
        connection.rollback()
        print(f"dataset import failed: {error}", file=sys.stderr)
        return 1
    finally:
        connection.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
