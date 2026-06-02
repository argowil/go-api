-- Migration 007: comments beneath news posts

CREATE TABLE IF NOT EXISTS news_comments (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    post_id     BIGINT UNSIGNED NOT NULL,
    body       TEXT            NOT NULL,
    author_id  BIGINT UNSIGNED NOT NULL,
    created_at DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    KEY ix_news_comments_post_created (post_id, created_at),
    CONSTRAINT fk_news_comments_post FOREIGN KEY (post_id) REFERENCES news_posts (id) ON DELETE CASCADE,
    CONSTRAINT fk_news_comments_author FOREIGN KEY (author_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
