CREATE TABLE IF NOT EXISTS blog_admin_roles (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_on INT UNSIGNED NOT NULL DEFAULT 0,
    created_by VARCHAR(100) NOT NULL DEFAULT '',
    modified_on INT UNSIGNED NOT NULL DEFAULT 0,
    modified_by VARCHAR(100) NOT NULL DEFAULT '',
    deleted_on INT UNSIGNED NOT NULL DEFAULT 0,
    is_del TINYINT UNSIGNED NOT NULL DEFAULT 0,
    name VARCHAR(80) NOT NULL,
    description VARCHAR(255) NOT NULL DEFAULT '',
    permissions TEXT NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_blog_admin_roles_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS blog_admin_users (
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
    role_id INT UNSIGNED NOT NULL,
    state TINYINT UNSIGNED NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE KEY uk_blog_admin_users_username (username),
    KEY idx_blog_admin_users_role_id (role_id),
    KEY idx_blog_admin_users_state (state)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

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

CREATE TABLE IF NOT EXISTS blog_platform_sites (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_on INT UNSIGNED NOT NULL DEFAULT 0,
    created_by VARCHAR(100) NOT NULL DEFAULT '',
    modified_on INT UNSIGNED NOT NULL DEFAULT 0,
    modified_by VARCHAR(100) NOT NULL DEFAULT '',
    deleted_on INT UNSIGNED NOT NULL DEFAULT 0,
    is_del TINYINT UNSIGNED NOT NULL DEFAULT 0,
    name VARCHAR(120) NOT NULL,
    domain VARCHAR(255) NOT NULL,
    description VARCHAR(255) NOT NULL DEFAULT '',
    state TINYINT UNSIGNED NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE KEY uk_blog_platform_sites_domain (domain),
    KEY idx_blog_platform_sites_state (state)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS blog_admin_audit_logs (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_on INT UNSIGNED NOT NULL DEFAULT 0,
    created_by VARCHAR(100) NOT NULL DEFAULT '',
    modified_on INT UNSIGNED NOT NULL DEFAULT 0,
    modified_by VARCHAR(100) NOT NULL DEFAULT '',
    deleted_on INT UNSIGNED NOT NULL DEFAULT 0,
    is_del TINYINT UNSIGNED NOT NULL DEFAULT 0,
    username VARCHAR(100) NOT NULL DEFAULT '',
    action VARCHAR(120) NOT NULL,
    resource VARCHAR(255) NOT NULL DEFAULT '',
    method VARCHAR(10) NOT NULL DEFAULT '',
    status INT NOT NULL DEFAULT 0,
    remote_ip VARCHAR(64) NOT NULL DEFAULT '',
    PRIMARY KEY (id),
    KEY idx_blog_admin_audit_username (username),
    KEY idx_blog_admin_audit_action (action),
    KEY idx_blog_admin_audit_created_on (created_on)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO blog_admin_roles
    (created_on, created_by, name, description, permissions)
VALUES
    (UNIX_TIMESTAMP(), 'system', '超级管理员', '拥有全部平台权限', '["*"]'),
    (UNIX_TIMESTAMP(), 'system', '平台管理员', '管理平台用户、配置与站点', '["users.manage","roles.manage","settings.manage","sites.manage","audit.read","content.manage"]'),
    (UNIX_TIMESTAMP(), 'system', '内容编辑', '管理文章与标签', '["content.manage"]'),
    (UNIX_TIMESTAMP(), 'system', '审计员', '查看平台操作记录', '["audit.read"]');

INSERT IGNORE INTO blog_platform_settings (created_on, created_by, setting_key, value)
VALUES
    (UNIX_TIMESTAMP(), 'system', 'site_name', '纸间'),
    (UNIX_TIMESTAMP(), 'system', 'site_url', ''),
    (UNIX_TIMESTAMP(), 'system', 'site_description', ''),
    (UNIX_TIMESTAMP(), 'system', 'timezone', 'Asia/Shanghai'),
    (UNIX_TIMESTAMP(), 'system', 'allow_registration', 'false'),
    (UNIX_TIMESTAMP(), 'system', 'maintenance_mode', 'false'),
    (UNIX_TIMESTAMP(), 'system', 'support_email', '');
