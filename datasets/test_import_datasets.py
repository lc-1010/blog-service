import io
import sqlite3
import unittest
from pathlib import Path

import import_datasets as datasets


class DatasetImportTests(unittest.TestCase):
    def setUp(self):
        self.db = sqlite3.connect(":memory:")
        self.db.execute("PRAGMA foreign_keys = ON")
        datasets.create_schema(self.db)

    def tearDown(self):
        self.db.close()

    def test_retail_import_anonymizes_members_and_marks_synthetic_posts(self):
        csv_data = """InvoiceNo,StockCode,Description,Quantity,InvoiceDate,UnitPrice,CustomerID,Country
100001,P1,Tea,2,12/1/2010 8:26,1.25,12345,United Kingdom
C100002,P2,Cup,-1,12/1/2010 8:30,3.00,12345,United Kingdom
100003,P3,Book,1,12/1/2010 8:40,, ,United Kingdom
"""
        rows = datasets.read_csv_rows(io.StringIO(csv_data), 100)
        counts = datasets.import_retail_rows(self.db, rows)

        self.assertEqual(counts["members"], 1)
        self.assertEqual(counts["orders"], 2)
        self.assertEqual(counts["rows_imported"], 2)
        self.assertEqual(counts["rows_skipped"], 1)
        self.assertEqual(
            self.db.execute("SELECT id FROM members").fetchone()[0],
            "retail-member-000001",
        )
        self.assertEqual(
            self.db.execute("SELECT status FROM orders WHERE source_record_id = 'C100002'").fetchone()[0],
            "cancelled",
        )
        self.assertEqual(
            self.db.execute("SELECT unit_price_minor, line_total_minor FROM order_items WHERE quantity = 2").fetchone(),
            (125, 250),
        )
        article = self.db.execute(
            "SELECT content, content_origin FROM articles WHERE source_dataset = ?",
            (datasets.SYNTHETIC_SOURCE,),
        ).fetchone()
        self.assertEqual(article[1], "synthetic")
        self.assertIn("不是实际顾客发布", article[0])
        self.assertNotIn("12345", str(self.db.execute("SELECT id FROM members").fetchall()))

    def test_poem_import_maps_authors_articles_and_source_tags(self):
        poems = [
            {
                "id": "poem-1",
                "author": "李白",
                "title": "静夜思",
                "paragraphs": ["床前明月光。", "疑是地上霜。"],
                "tags": ["唐诗三百首", "思乡", "五言绝句"],
            },
            {
                "id": "poem-2",
                "author": "李白",
                "title": "赠汪伦",
                "paragraphs": ["李白乘舟将欲行。"],
                "tags": ["唐诗三百首", "友情"],
            },
        ]
        counts = datasets.import_poems(self.db, poems)

        self.assertEqual(counts, {"poems": 2, "authors": 1, "categories": 3})
        self.assertEqual(
            self.db.execute("SELECT COUNT(*) FROM author_profiles").fetchone()[0],
            1,
        )
        article = self.db.execute(
            "SELECT content, content_origin, source_dataset FROM articles WHERE id = 'tang-poem-poem-1'"
        ).fetchone()
        self.assertEqual(article, ("床前明月光。\n疑是地上霜。", "dataset", datasets.TANG_SOURCE))
        self.assertEqual(
            self.db.execute(
                """SELECT c.name, c.category_type
                   FROM categories c
                   JOIN article_categories ac ON ac.category_id = c.id
                   WHERE ac.article_id = 'tang-poem-poem-1'
                   ORDER BY c.name"""
            ).fetchall(),
            [("五言绝句", "form"), ("思乡", "topic")],
        )

    def test_import_is_idempotent_for_poems(self):
        poem = [{
            "id": "poem-1",
            "author": "杜甫",
            "title": "春望",
            "paragraphs": ["国破山河在。"],
            "tags": ["唐诗三百首", "忧国忧民"],
        }]
        datasets.import_poems(self.db, poem)
        datasets.import_poems(self.db, poem)
        self.assertEqual(self.db.execute("SELECT COUNT(*) FROM articles").fetchone()[0], 1)
        self.assertEqual(self.db.execute("SELECT COUNT(*) FROM article_categories").fetchone()[0], 1)


if __name__ == "__main__":
    unittest.main()
