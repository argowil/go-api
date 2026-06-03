package shiftbase

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Employee mirrors the Shiftbase employee resource.
type Employee struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Function  string `json:"function"`
	Active    bool   `json:"active"`
}

// ListEmployees returns all employees visible to the API key's account.
func (c *Client) ListEmployees() ([]Employee, error) {
	var result struct {
		Data []Employee `json:"data"`
	}
	if err := c.get("/employees", &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// FindEmployeeByEmail looks up a Shiftbase user by email via GET /users/{email}.
// Returns nil, nil when the user is not found (404).
func (c *Client) FindEmployeeByEmail(email string) (*Employee, error) {
	var result struct {
		Data struct {
			User struct {
				ID    string `json:"id"`
				Email string `json:"email"`
			} `json:"User"`
		} `json:"data"`
	}

	err := c.get("/users/"+url.PathEscape(email), &result)
	if err != nil {
		if strings.Contains(err.Error(), "returned 404") {
			return nil, nil
		}
		return nil, fmt.Errorf("find employee by email: %w", err)
	}

	if result.Data.User.ID == "" {
		return nil, nil
	}
	id, err := strconv.Atoi(result.Data.User.ID)
	if err != nil {
		return nil, fmt.Errorf("parse shiftbase user id %q: %w", result.Data.User.ID, err)
	}
	return &Employee{ID: id, Email: result.Data.User.Email}, nil
}
