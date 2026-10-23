ALTER TABLE blog_article
    ADD COLUMN content LONGTEXT NULL;

UPDATE blog_article
SET content = ''
WHERE content IS NULL;

ALTER TABLE blog_article
    MODIFY COLUMN content LONGTEXT NOT NULL;

CREATE TABLE blog_article_tag (
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
