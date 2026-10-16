PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS members (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    member_type TEXT NOT NULL CHECK (member_type IN ('customer', 'author')),
    source_dataset TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS author_profiles (
    member_id TEXT PRIMARY KEY REFERENCES members(id) ON DELETE CASCADE,
    bio TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS categories (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    category_type TEXT NOT NULL CHECK (category_type IN ('topic', 'form', 'classification'))
);

CREATE TABLE IF NOT EXISTS articles (
    id TEXT PRIMARY KEY,
    author_id TEXT NOT NULL REFERENCES members(id),
    title TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'published',
    content_origin TEXT NOT NULL CHECK (content_origin IN ('dataset', 'synthetic')),
    source_dataset TEXT NOT NULL,
    source_record_id TEXT
);

CREATE TABLE IF NOT EXISTS article_categories (
    article_id TEXT NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (article_id, category_id)
);

CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    source_code TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'GBP'
);

CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    member_id TEXT NOT NULL REFERENCES members(id),
    occurred_at TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('completed', 'cancelled', 'returned')),
    currency TEXT NOT NULL DEFAULT 'GBP',
    source_dataset TEXT NOT NULL,
    source_record_id TEXT NOT NULL,
    UNIQUE (source_dataset, source_record_id)
);

CREATE TABLE IF NOT EXISTS order_items (
    id INTEGER PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL REFERENCES products(id),
    quantity INTEGER NOT NULL,
    unit_price_minor INTEGER NOT NULL,
    line_total_minor INTEGER NOT NULL,
    source_row INTEGER NOT NULL UNIQUE
);

CREATE INDEX IF NOT EXISTS idx_articles_author ON articles(author_id);
CREATE INDEX IF NOT EXISTS idx_article_categories_category ON article_categories(category_id);
CREATE INDEX IF NOT EXISTS idx_orders_member_time ON orders(member_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items(order_id);
