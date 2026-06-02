package shiftbase

// Employee mirrors the Shiftbase employee resource.
// Field names follow Shiftbase's JSON keys as documented at
// https://developer.shiftbase.com/
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
