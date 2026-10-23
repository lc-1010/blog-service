CREATE TABLE IF NOT EXISTS demo_members (
    id VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    username VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    member_type VARCHAR(16) NOT NULL,
    source_dataset VARCHAR(80) NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_demo_members_username (username),
    KEY idx_demo_members_source (source_dataset, member_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS demo_author_profiles (
    member_id VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    bio TEXT NOT NULL,
    PRIMARY KEY (member_id),
    CONSTRAINT fk_demo_author_profiles_member
        FOREIGN KEY (member_id) REFERENCES demo_members (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS demo_categories (
    id VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    name VARCHAR(255) NOT NULL,
    category_type VARCHAR(32) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_demo_categories_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS demo_articles (
    id VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    author_id VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    title VARCHAR(512) NOT NULL,
    summary TEXT NOT NULL,
    content LONGTEXT NOT NULL,
    status VARCHAR(20) NOT NULL,
    content_origin VARCHAR(20) NOT NULL,
    source_dataset VARCHAR(80) NOT NULL,
    source_record_id VARCHAR(128) NULL,
    PRIMARY KEY (id),
    KEY idx_demo_articles_author (author_id),
    KEY idx_demo_articles_origin (source_dataset, content_origin),
    CONSTRAINT fk_demo_articles_author
        FOREIGN KEY (author_id) REFERENCES demo_members (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS demo_article_categories (
    article_id VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    category_id VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    PRIMARY KEY (article_id, category_id),
    KEY idx_demo_article_categories_category (category_id),
    CONSTRAINT fk_demo_article_categories_article
        FOREIGN KEY (article_id) REFERENCES demo_articles (id),
    CONSTRAINT fk_demo_article_categories_category
        FOREIGN KEY (category_id) REFERENCES demo_categories (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS demo_products (
    id VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    source_code VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    title VARCHAR(512) NOT NULL,
    currency CHAR(3) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_demo_products_source_code (source_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS demo_orders (
    id VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    member_id VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    occurred_at VARCHAR(32) NOT NULL,
    status VARCHAR(20) NOT NULL,
    currency CHAR(3) NOT NULL,
    source_dataset VARCHAR(80) NOT NULL,
    source_record_id VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_demo_orders_source (source_dataset, source_record_id),
    KEY idx_demo_orders_member_time (member_id, occurred_at),
    CONSTRAINT fk_demo_orders_member
        FOREIGN KEY (member_id) REFERENCES demo_members (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS demo_order_items (
    id BIGINT NOT NULL,
    order_id VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    product_id VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    quantity INT NOT NULL,
    unit_price_minor BIGINT NOT NULL,
    line_total_minor BIGINT NOT NULL,
    source_row BIGINT NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_demo_order_items_source_row (source_row),
    KEY idx_demo_order_items_order (order_id),
    CONSTRAINT fk_demo_order_items_order
        FOREIGN KEY (order_id) REFERENCES demo_orders (id),
    CONSTRAINT fk_demo_order_items_product
        FOREIGN KEY (product_id) REFERENCES demo_products (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
