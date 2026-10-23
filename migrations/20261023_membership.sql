CREATE TABLE IF NOT EXISTS blog_platform_settings (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_on INT UNSIGNED NOT NULL DEFAULT 0,
    created_by VARCHAR(100) NOT NULL DEFAULT '',
    modified_on INT UNSIGNED NOT NULL DEFAULT 0,
    modified_by VARCHAR(100) NOT NULL DEFAULT '',
    deleted_on INT UNSIGNED NOT NULL DEFAULT 0,
    is_del TINYINT UNSIGNED NOT NULL DEFAULT 0,
    setting_key VARCHAR(100) NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_blog_platform_settings_key (setting_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS blog_members (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_on INT UNSIGNED NOT NULL DEFAULT 0,
    created_by VARCHAR(100) NOT NULL DEFAULT '',
    modified_on INT UNSIGNED NOT NULL DEFAULT 0,
    modified_by VARCHAR(100) NOT NULL DEFAULT '',
    deleted_on INT UNSIGNED NOT NULL DEFAULT 0,
    is_del TINYINT UNSIGNED NOT NULL DEFAULT 0,
    username VARCHAR(100) NOT NULL,
    display_name VARCHAR(100) NOT NULL DEFAULT '',
    password_hash VARCHAR(100) NOT NULL,
    member_type VARCHAR(20) NOT NULL DEFAULT 'member',
    state TINYINT UNSIGNED NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE KEY uk_blog_members_username (username),
    KEY idx_blog_members_type_state (member_type, state, is_del)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS blog_member_profiles (
    member_id INT UNSIGNED NOT NULL,
    bio TEXT NOT NULL,
    avatar_url VARCHAR(500) NOT NULL DEFAULT '',
    PRIMARY KEY (member_id),
    CONSTRAINT fk_blog_member_profiles_member
        FOREIGN KEY (member_id) REFERENCES blog_members (id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS blog_member_sessions (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    member_id INT UNSIGNED NOT NULL,
    token_hash CHAR(64) NOT NULL,
    expires_at DATETIME NOT NULL,
    revoked_at DATETIME NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_blog_member_sessions_token_hash (token_hash),
    KEY idx_blog_member_sessions_member (member_id, revoked_at, expires_at),
    CONSTRAINT fk_blog_member_sessions_member
        FOREIGN KEY (member_id) REFERENCES blog_members (id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO blog_platform_settings
    (created_on, created_by, setting_key, value)
VALUES
    (UNIX_TIMESTAMP(), 'system', 'allow_registration', 'false');
