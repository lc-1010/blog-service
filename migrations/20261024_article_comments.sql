CREATE TABLE IF NOT EXISTS blog_article_comments (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_on INT UNSIGNED NOT NULL DEFAULT 0,
    created_by VARCHAR(100) NOT NULL DEFAULT '',
    modified_on INT UNSIGNED NOT NULL DEFAULT 0,
    modified_by VARCHAR(100) NOT NULL DEFAULT '',
    deleted_on INT UNSIGNED NOT NULL DEFAULT 0,
    is_del TINYINT UNSIGNED NOT NULL DEFAULT 0,
    article_id INT UNSIGNED NOT NULL,
    member_id INT UNSIGNED NOT NULL,
    parent_id INT UNSIGNED NOT NULL DEFAULT 0,
    content TEXT NOT NULL,
    state TINYINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    KEY idx_blog_article_comments_article_state (article_id, state, parent_id, created_on),
    KEY idx_blog_article_comments_member (member_id),
    CONSTRAINT fk_blog_article_comments_article
        FOREIGN KEY (article_id) REFERENCES blog_article (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_blog_article_comments_member
        FOREIGN KEY (member_id) REFERENCES blog_members (id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
