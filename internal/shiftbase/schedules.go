package shiftbase

import (
	"fmt"
	"time"
)

// Shift represents a single planned shift from Shiftbase.
type Shift struct {
	ID         int       `json:"id"`
	EmployeeID int       `json:"employee_id"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Break      int       `json:"break"` // break duration in minutes
	Department string    `json:"department"`
	Note       string    `json:"note"`
}

// TimeRegistration is a clocked-in/clocked-out record from Shiftbase.
type TimeRegistration struct {
	ID         int       `json:"id"`
	EmployeeID int       `json:"employee_id"`
	ClockIn    time.Time `json:"clock_in"`
	ClockOut   time.Time `json:"clock_out"`
	Break      int       `json:"break"`
	Approved   bool      `json:"approved"`
}

// ListShifts returns all shifts in the given date range.
// from and to should be in YYYY-MM-DD format.
func (c *Client) ListShifts(from, to string) ([]Shift, error) {
	var result struct {
		Data []Shift `json:"data"`
	}
	path := fmt.Sprintf("/schedules?start_date=%s&end_date=%s", from, to)
	if err := c.get(path, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// ListShiftsForEmployee returns shifts for a single employee within a date range.
func (c *Client) ListShiftsForEmployee(employeeID int, from, to string) ([]Shift, error) {
	var result struct {
		Data []Shift `json:"data"`
	}
	path := fmt.Sprintf("/schedules?employee_id=%d&start_date=%s&end_date=%s", employeeID, from, to)
	if err := c.get(path, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// ListTimeRegistrations returns time registrations within a date range.
func (c *Client) ListTimeRegistrations(from, to string) ([]TimeRegistration, error) {
	var result struct {
		Data []TimeRegistration `json:"data"`
	}
	path := fmt.Sprintf("/time_registrations?start_date=%s&end_date=%s", from, to)
	if err := c.get(path, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}
