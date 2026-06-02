package shiftbase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
)

// CreateEmployeeRequest is the payload sent to Shiftbase when creating a new employee.
// The Shiftbase API wraps fields in a "User" key: POST /api/users
type CreateEmployeeRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	TeamID    string `json:"team_id,omitempty"`
}

type createEmployeeBody struct {
	User CreateEmployeeRequest `json:"User"`
}

// CreateEmployeeResponse is the relevant part of the Shiftbase response.
type CreateEmployeeResponse struct {
	Data struct {
		User struct {
			ID string `json:"id"`
		} `json:"User"`
	} `json:"data"`
}

// DeleteEmployee removes an employee from Shiftbase by their Shiftbase user ID.
func (c *Client) DeleteEmployee(shiftbaseID int) error {
	url := c.baseURL + fmt.Sprintf("/users/%d", shiftbaseID)
	log.Printf("shiftbase DELETE %s", url)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "API "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("shiftbase delete employee: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("shiftbase DELETE %s -> %d %s", url, resp.StatusCode, string(body))
		return fmt.Errorf("shiftbase returned %d: %s", resp.StatusCode, body)
	}
	log.Printf("shiftbase DELETE %s -> %d", url, resp.StatusCode)
	return nil
}

// CreateEmployee creates a new employee in Shiftbase and returns their employee ID.
// This ID is stored locally so future API calls can reference the correct Shiftbase record.
func (c *Client) CreateEmployee(req CreateEmployeeRequest) (int, error) {
	body, err := json.Marshal(createEmployeeBody{User: req})
	if err != nil {
		return 0, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + c.createEmployeePath
	log.Printf("shiftbase POST %s payload=%s", url, string(body))
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}

	httpReq.Header.Set("Authorization", "API "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("shiftbase create employee: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("shiftbase POST %s -> %d %s", url, resp.StatusCode, string(respBody))
		return 0, fmt.Errorf("shiftbase returned %d: %s", resp.StatusCode, respBody)
	}

	var result CreateEmployeeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}
	id, err := strconv.Atoi(result.Data.User.ID)
	if err != nil {
		return 0, fmt.Errorf("parse shiftbase user id %q: %w", result.Data.User.ID, err)
	}
	log.Printf("shiftbase POST %s -> %d employee_id=%d", url, resp.StatusCode, id)

	return id, nil
}
