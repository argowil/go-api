-- Migration 005: force password change on first login
--
-- When an admin creates an account the password is temporary.
-- must_change_password is set to 1 so the app redirects the employee to
-- a change-password screen before they can use anything else.
-- It is cleared to 0 once they set their own password.

ALTER TABLE users
  ADD COLUMN must_change_password TINYINT(1) NOT NULL DEFAULT 0 AFTER active;
