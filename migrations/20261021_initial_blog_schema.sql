CREATE TABLE IF NOT EXISTS blog_tag (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_on INT UNSIGNED NOT NULL DEFAULT 0,
    created_by VARCHAR(100) NOT NULL DEFAULT '',
    modified_on INT UNSIGNED NOT NULL DEFAULT 0,
    modified_by VARCHAR(100) NOT NULL DEFAULT '',
    deleted_on INT UNSIGNED NOT NULL DEFAULT 0,
    is_del TINYINT UNSIGNED NOT NULL DEFAULT 0,
    name VARCHAR(100) NOT NULL DEFAULT '',
    state TINYINT UNSIGNED NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    KEY idx_blog_tag_name (name),
    KEY idx_blog_tag_state_is_del (state, is_del)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS blog_article (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_on INT UNSIGNED NOT NULL DEFAULT 0,
    created_by VARCHAR(100) NOT NULL DEFAULT '',
    modified_on INT UNSIGNED NOT NULL DEFAULT 0,
    modified_by VARCHAR(100) NOT NULL DEFAULT '',
    deleted_on INT UNSIGNED NOT NULL DEFAULT 0,
    is_del TINYINT UNSIGNED NOT NULL DEFAULT 0,
    title VARCHAR(100) NOT NULL DEFAULT '',
    `desc` VARCHAR(255) NOT NULL DEFAULT '',
    cover_image_url VARCHAR(255) NOT NULL DEFAULT '',
    content LONGTEXT NOT NULL,
    state TINYINT UNSIGNED NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    KEY idx_blog_article_state_is_del (state, is_del),
    KEY idx_blog_article_title (title)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS blog_article_tag (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_on INT UNSIGNED NOT NULL DEFAULT 0,
    created_by VARCHAR(255) NOT NULL DEFAULT '',
    modified_on INT UNSIGNED NOT NULL DEFAULT 0,
    modified_by VARCHAR(255) NOT NULL DEFAULT '',
    deleted_on INT UNSIGNED NOT NULL DEFAULT 0,
    is_del TINYINT UNSIGNED NOT NULL DEFAULT 0,
    tag_id INT UNSIGNED NOT NULL,
    article_id INT UNSIGNED NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_blog_article_tag_article_id (article_id),
    KEY idx_blog_article_tag_tag_id (tag_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
