import sqlite3
import unittest

import import_sqlite_to_mysql as importer


class MySQLDatasetImportTests(unittest.TestCase):
    def test_sql_values_are_encoded_without_interpolating_text(self):
        self.assertEqual(importer.sql_value(None), "NULL")
        self.assertEqual(importer.sql_value(123), "123")
        self.assertEqual(importer.sql_value("李白'\\n"), "CONVERT(X'e69d8ee799bd275c6e' USING utf8mb4)")
        with self.assertRaises(ValueError):
            importer.sql_value(1.25)

    def test_insert_statement_keeps_unicode_and_null_values(self):
        sql = importer.build_insert(
            "articles",
            ("id", "title", "source_record_id"),
            [("poem-1", "静夜思", None)],
        )
        self.assertIn("INSERT IGNORE INTO `demo_articles`", sql)
        self.assertIn("CONVERT(X'e99d99e5a49ce6809d' USING utf8mb4)", sql)
        self.assertIn(", NULL)", sql)
        self.assertNotIn("静夜思", sql)

    def test_import_sql_contains_all_tables_in_foreign_key_order(self):
        connection = sqlite3.connect(":memory:")
        connection.executescript("""
            CREATE TABLE members (id TEXT, username TEXT, display_name TEXT,
                member_type TEXT, source_dataset TEXT, created_at TEXT);
            CREATE TABLE author_profiles (member_id TEXT, bio TEXT);
            CREATE TABLE categories (id TEXT, name TEXT, category_type TEXT);
            CREATE TABLE articles (id TEXT, author_id TEXT, title TEXT, summary TEXT,
                content TEXT, status TEXT, content_origin TEXT, source_dataset TEXT,
                source_record_id TEXT);
            CREATE TABLE article_categories (article_id TEXT, category_id TEXT);
            CREATE TABLE products (id TEXT, source_code TEXT, title TEXT, currency TEXT);
            CREATE TABLE orders (id TEXT, member_id TEXT, occurred_at TEXT, status TEXT,
                currency TEXT, source_dataset TEXT, source_record_id TEXT);
            CREATE TABLE order_items (id INTEGER, order_id TEXT, product_id TEXT,
                quantity INTEGER, unit_price_minor INTEGER, line_total_minor INTEGER,
                source_row INTEGER);
        """)
        connection.execute(
            "INSERT INTO members VALUES (?, ?, ?, ?, ?, ?)",
            ("member-1", "member-1", "测试作者", "author", "sample", "2026-01-01 00:00:00"),
        )
        connection.execute("INSERT INTO author_profiles VALUES (?, ?)", ("member-1", ""))
        connection.execute("INSERT INTO products VALUES (?, ?, ?, ?)", ("product-1", "p1", "Tea", "GBP"))
        connection.execute(
            "INSERT INTO orders VALUES (?, ?, ?, ?, ?, ?, ?)",
            ("order-1", "member-1", "2010-12-01T08:26", "completed", "GBP", "sample", "invoice-1"),
        )
        connection.execute(
            "INSERT INTO order_items VALUES (?, ?, ?, ?, ?, ?, ?)",
            (1, "order-1", "product-1", 2, 125, 250, 1),
        )
        sql, counts = importer.build_import_sql(connection, 10)
        connection.close()

        self.assertEqual(counts["members"], 1)
        self.assertIn("START TRANSACTION", sql)
        self.assertIn("COMMIT", sql)
        self.assertLess(sql.index("demo_members`"), sql.index("demo_author_profiles`"))
        self.assertLess(sql.index("demo_orders`"), sql.index("demo_order_items`"))
        self.assertNotIn("ROLLBACK", sql)


if __name__ == "__main__":
    unittest.main()
