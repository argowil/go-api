-- Migration 004: link local users to their Shiftbase employee record
--
-- shiftbase_employee_id is set when the account is created via the admin panel,
-- which calls the Shiftbase API and stores the returned employee ID here.
-- NULL means the user has no Shiftbase account yet (or Shiftbase is not configured).

ALTER TABLE users
  ADD COLUMN shiftbase_employee_id INT UNSIGNED DEFAULT NULL AFTER role,
  ADD KEY ix_users_shiftbase_id (shiftbase_employee_id);
