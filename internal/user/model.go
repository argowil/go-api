// Package user contains everything related to Argowil employee accounts.
package user

import "time"

// User represents a person who can log in to the platform.
type User struct {
	ID                  uint      `db:"id"                    json:"id"`
	FirstName           string    `db:"first_name"            json:"first_name"`
	LastName            string    `db:"last_name"             json:"last_name"`
	Email               string    `db:"email"                 json:"email"`
	PasswordHash        string    `db:"password_hash"         json:"-"`
	Role                string    `db:"role"                  json:"role"`
	ShiftbaseEmployeeID *int      `db:"shiftbase_employee_id"  json:"shiftbase_employee_id,omitempty"`
	Active              bool      `db:"active"                 json:"active"`
	MustChangePassword  bool      `db:"must_change_password"   json:"must_change_password"`
	CreatedAt           time.Time `db:"created_at"            json:"created_at"`
	UpdatedAt           time.Time `db:"updated_at"            json:"updated_at"`
}

// FullName returns the user's display name.
func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

// CreateUserRequest is the payload for creating a new account.
type CreateUserRequest struct {
	FirstName           string `json:"first_name"`
	LastName            string `json:"last_name"`
	Email               string `json:"email"`
	Password            string `json:"password"`
	Role                string `json:"role"`
	ShiftbaseEmployeeID *int   `json:"shiftbase_employee_id,omitempty"`
	MustChangePassword  bool   `json:"must_change_password"`
}

// UpdateUserRequest allows partial updates to an existing account.
type UpdateUserRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Email     *string `json:"email,omitempty"`
	Role      *string `json:"role,omitempty"`
	Active    *bool   `json:"active,omitempty"`
}
