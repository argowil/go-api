-- Migration 003: local schedule tables
--
-- Used when Shiftbase is not configured (SHIFTBASE_API_KEY is empty).
-- If you use Shiftbase exclusively these tables stay empty but don't hurt.

CREATE TABLE IF NOT EXISTS shifts (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id       BIGINT UNSIGNED NOT NULL,
    start_time    DATETIME        NOT NULL,
    end_time      DATETIME        NOT NULL,
    break_minutes SMALLINT        NOT NULL DEFAULT 0,
    department    VARCHAR(100)    NOT NULL DEFAULT '',
    note          TEXT,
    created_by    BIGINT UNSIGNED NOT NULL,
    created_at    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    KEY ix_shifts_user   (user_id),
    KEY ix_shifts_start  (start_time),
    CONSTRAINT fk_shifts_user       FOREIGN KEY (user_id)    REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_shifts_created_by FOREIGN KEY (created_by) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE IF NOT EXISTS time_registrations (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id       BIGINT UNSIGNED NOT NULL,
    clock_in      DATETIME        NOT NULL,
    clock_out     DATETIME,
    break_minutes SMALLINT        NOT NULL DEFAULT 0,
    approved      TINYINT(1)      NOT NULL DEFAULT 0,
    approved_by   BIGINT UNSIGNED,
    note          TEXT,
    created_at    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    KEY ix_time_reg_user    (user_id),
    KEY ix_time_reg_clockin (clock_in),
    CONSTRAINT fk_time_reg_user        FOREIGN KEY (user_id)     REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_time_reg_approved_by FOREIGN KEY (approved_by) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
