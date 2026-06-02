-- Migration 006: news posts for the home feed
--
-- Teamleiders and admins can create posts with an optional image.
-- Images are stored on disk and served via /uploads/*.
-- image_url stores just the filename, the frontend prepends the base URL.

CREATE TABLE IF NOT EXISTS news_posts (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    title      VARCHAR(255)    NOT NULL,
    body       TEXT            NOT NULL,
    image_url  VARCHAR(500)    DEFAULT NULL,
    author_id  BIGINT UNSIGNED NOT NULL,
    created_at DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    KEY ix_news_created (created_at DESC),
    CONSTRAINT fk_news_author FOREIGN KEY (author_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
