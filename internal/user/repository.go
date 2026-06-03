package user

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Repository is the interface every data layer must satisfy.
// This makes it straightforward to swap MySQL for a test double.
type Repository interface {
	FindByID(ctx context.Context, id uint) (*User, error)
	FindByIDAdmin(ctx context.Context, id uint) (*User, error) // no active filter — for admin use
	FindByEmail(ctx context.Context, email string) (*User, error)
	List(ctx context.Context) ([]User, error)
	Create(ctx context.Context, u *User) error
	Update(ctx context.Context, u *User) error
	SetShiftbaseID(ctx context.Context, id uint, shiftbaseID int) error
	Delete(ctx context.Context, id uint) error
}

type mysqlRepository struct {
	db *sqlx.DB
}

// NewRepository returns a MySQL-backed Repository.
func NewRepository(db *sqlx.DB) Repository {
	return &mysqlRepository{db: db}
}

func (r *mysqlRepository) FindByID(ctx context.Context, id uint) (*User, error) {
	var u User
	err := r.db.GetContext(ctx, &u, `SELECT * FROM users WHERE id = ? AND active = 1`, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &u, nil
}

func (r *mysqlRepository) FindByIDAdmin(ctx context.Context, id uint) (*User, error) {
	var u User
	err := r.db.GetContext(ctx, &u, `SELECT * FROM users WHERE id = ?`, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &u, nil
}

func (r *mysqlRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.db.GetContext(ctx, &u, `SELECT * FROM users WHERE email = ? AND active = 1`, email)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &u, nil
}

func (r *mysqlRepository) List(ctx context.Context) ([]User, error) {
	var users []User
	err := r.db.SelectContext(ctx, &users, `SELECT * FROM users WHERE active = 1 ORDER BY last_name, first_name`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

func (r *mysqlRepository) Create(ctx context.Context, u *User) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO users (first_name, last_name, email, password_hash, role, shiftbase_employee_id, active, must_change_password)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		u.FirstName, u.LastName, u.Email, u.PasswordHash, u.Role, u.ShiftbaseEmployeeID, u.Active, u.MustChangePassword,
	)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	id, _ := result.LastInsertId()
	u.ID = uint(id)
	return nil
}

func (r *mysqlRepository) Update(ctx context.Context, u *User) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET first_name=?, last_name=?, email=?, password_hash=?, role=?, active=?, must_change_password=?, updated_at=NOW()
		 WHERE id=?`,
		u.FirstName, u.LastName, u.Email, u.PasswordHash, u.Role, u.Active, u.MustChangePassword, u.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *mysqlRepository) SetShiftbaseID(ctx context.Context, id uint, shiftbaseID int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET shiftbase_employee_id=?, updated_at=NOW() WHERE id=?`,
		shiftbaseID, id,
	)
	if err != nil {
		return fmt.Errorf("set shiftbase id: %w", err)
	}
	return nil
}

func (r *mysqlRepository) Delete(ctx context.Context, id uint) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET active=0 WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}
