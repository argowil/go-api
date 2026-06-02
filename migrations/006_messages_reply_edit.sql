ALTER TABLE messages
  ADD COLUMN edited        TINYINT(1)       NOT NULL DEFAULT 0      AFTER content,
  ADD COLUMN reply_to_id   BIGINT UNSIGNED  NULL                    AFTER edited,
  ADD COLUMN reply_preview VARCHAR(200)     NULL                    AFTER reply_to_id;
