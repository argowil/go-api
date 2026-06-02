-- Migration 002: persist refresh tokens so they can be revoked
--
-- When a user logs out or an admin deactivates an account, delete their rows here.
-- The access token (short-lived JWT) does not need a server-side record.

CREATE TABLE IF NOT EXISTS sessions (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id       BIGINT UNSIGNED NOT NULL,
    refresh_token VARCHAR(512)    NOT NULL,
    user_agent    VARCHAR(512)    NOT NULL DEFAULT '',
    ip_address    VARCHAR(45)     NOT NULL DEFAULT '',
    expires_at    DATETIME        NOT NULL,
    created_at    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    UNIQUE KEY uq_sessions_token (refresh_token(255)),
    KEY ix_sessions_user (user_id),
    CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
