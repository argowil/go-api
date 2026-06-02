CREATE TABLE IF NOT EXISTS messages (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id    INT UNSIGNED    NOT NULL,
    user_name  VARCHAR(120)    NOT NULL,
    content    TEXT            NOT NULL,
    created_at DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_messages_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
