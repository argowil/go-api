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

// OpenShift is an open/available shift employees can claim.
type OpenShift struct {
	ID                 string `json:"id"`
	OccurrenceID       string `json:"occurrence_id"`
	Date               string `json:"date"`
	StartTime          string `json:"start_time"`
	EndTime            string `json:"end_time"`
	Break              int    `json:"break"`
	TeamID             string `json:"team_id"`
	TeamName           string `json:"team_name"`
	DepartmentID       string `json:"department_id"`
	ShiftID            string `json:"shift_id"`
	InstancesRemaining int    `json:"instances_remaining"`
	ApprovalRequired   bool   `json:"approval_required"`
	Description        string `json:"description"`
}

// ShiftTemplate is a reusable shift definition (name, times, break).
type ShiftTemplate struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	LongName  string `json:"long_name"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Break     int    `json:"break"`
	Color     string `json:"color"`
	TeamID    string `json:"team_id"`
	DeptID    string `json:"department_id"`
}

// CreateOpenShiftRequest is the payload for creating a new open shift.
type CreateOpenShiftRequest struct {
	Date             string `json:"date"`
	DepartmentID     string `json:"department_id"`
	TeamID           string `json:"team_id"`
	ShiftID          string `json:"shift_id"`
	StartTime        string `json:"start_time"`
	EndTime          string `json:"end_time"`
	Break            string `json:"break"`
	Instances        int    `json:"instances"`
	ApprovalRequired bool   `json:"approval_required"`
	Description      string `json:"description"`
}

// ListOpenShifts returns open shifts within a date range.
func (c *Client) ListOpenShifts(from, to string) ([]OpenShift, error) {
	var raw struct {
		Data []struct {
			OpenShift struct {
				ID                 string `json:"id"`
				OccurrenceID       string `json:"occurrence_id"`
				Date               string `json:"date"`
				StartTime          string `json:"starttime"`
				EndTime            string `json:"endtime"`
				Break              string `json:"break"`
				TeamID             string `json:"team_id"`
				DepartmentID       string `json:"department_id"`
				ShiftID            string `json:"shift_id"`
				InstancesRemaining string `json:"instances_remaining"`
				ApprovalRequired   bool   `json:"approval_required"`
				Description        string `json:"description"`
			} `json:"OpenShift"`
			Team struct {
				Name string `json:"name"`
			} `json:"Team"`
		} `json:"data"`
	}

	if err := c.get(fmt.Sprintf("/open_shifts?min_date=%s&max_date=%s", from, to), &raw); err != nil {
		return nil, err
	}

	out := make([]OpenShift, 0, len(raw.Data))
	for _, d := range raw.Data {
		if from != "" && d.OpenShift.Date < from {
			continue
		}
		if to != "" && d.OpenShift.Date > to {
			continue
		}
		breakMin, _ := strconv.Atoi(d.OpenShift.Break)
		remaining, _ := strconv.Atoi(d.OpenShift.InstancesRemaining)
		occID := d.OpenShift.OccurrenceID
		if occID == "" {
			occID = d.OpenShift.ID
		}
		out = append(out, OpenShift{
			ID:                 d.OpenShift.ID,
			OccurrenceID:       occID,
			Date:               d.OpenShift.Date,
			StartTime:          d.OpenShift.StartTime,
			EndTime:            d.OpenShift.EndTime,
			Break:              breakMin,
			TeamID:             d.OpenShift.TeamID,
			TeamName:           d.Team.Name,
			DepartmentID:       d.OpenShift.DepartmentID,
			ShiftID:            d.OpenShift.ShiftID,
			InstancesRemaining: remaining,
			ApprovalRequired:   d.OpenShift.ApprovalRequired,
			Description:        d.OpenShift.Description,
		})
	}
	return out, nil
}

// GetOpenShift returns a single open shift by ID.
func (c *Client) GetOpenShift(id string) (*OpenShift, error) {
	var raw struct {
		Data struct {
			OpenShift struct {
				ID                 string `json:"id"`
				OccurrenceID       string `json:"occurrence_id"`
				Date               string `json:"date"`
				StartTime          string `json:"starttime"`
				EndTime            string `json:"endtime"`
				Break              string `json:"break"`
				TeamID             string `json:"team_id"`
				DepartmentID       string `json:"department_id"`
				ShiftID            string `json:"shift_id"`
				InstancesRemaining int    `json:"instances_remaining"`
				ApprovalRequired   bool   `json:"approval_required"`
				Description        string `json:"description"`
			} `json:"OpenShift"`
			Team struct {
				Name string `json:"name"`
			} `json:"Team"`
		} `json:"data"`
	}
	if err := c.get("/open_shifts/"+id, &raw); err != nil {
		return nil, err
	}
	breakMin, _ := strconv.Atoi(raw.Data.OpenShift.Break)
	remaining := raw.Data.OpenShift.InstancesRemaining
	occID := raw.Data.OpenShift.OccurrenceID
	if occID == "" {
		occID = raw.Data.OpenShift.ID
	}
	return &OpenShift{
		ID:                 raw.Data.OpenShift.ID,
		OccurrenceID:       occID,
		Date:               raw.Data.OpenShift.Date,
		StartTime:          raw.Data.OpenShift.StartTime,
		EndTime:            raw.Data.OpenShift.EndTime,
		Break:              breakMin,
		TeamID:             raw.Data.OpenShift.TeamID,
		TeamName:           raw.Data.Team.Name,
		DepartmentID:       raw.Data.OpenShift.DepartmentID,
		ShiftID:            raw.Data.OpenShift.ShiftID,
		InstancesRemaining: remaining,
		ApprovalRequired:   raw.Data.OpenShift.ApprovalRequired,
		Description:        raw.Data.OpenShift.Description,
	}, nil
}

// ClaimOpenShift assigns an employee to an open shift.
// Creates a roster entry linked to the open shift, then decrements instances_remaining.
// If instances_remaining reaches 0, the open shift is deleted.
func (c *Client) ClaimOpenShift(shiftbaseUserID int, shift OpenShift) error {
	type rosterBody struct {
		Roster struct {
			OpenShiftID  string `json:"open_shift_id"`
			UserID       string `json:"user_id"`
			Date         string `json:"date"`
			DepartmentID string `json:"department_id"`
			TeamID       string `json:"team_id"`
			ShiftID      string `json:"shift_id"`
			StartTime    string `json:"starttime"`
			EndTime      string `json:"endtime"`
			Break        string `json:"break"`
		} `json:"Roster"`
	}

	var body rosterBody
	body.Roster.OpenShiftID = shift.ID
	body.Roster.UserID = strconv.Itoa(shiftbaseUserID)
	body.Roster.Date = shift.Date
	body.Roster.DepartmentID = shift.DepartmentID
	body.Roster.TeamID = shift.TeamID
	body.Roster.ShiftID = shift.ShiftID
	body.Roster.StartTime = shift.StartTime
	body.Roster.EndTime = shift.EndTime
	body.Roster.Break = strconv.Itoa(shift.Break)

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal claim: %w", err)
	}

	rosterURL := c.baseURL + "/rosters"
	log.Printf("shiftbase POST %s (claim open shift %s for user %d)", rosterURL, shift.ID, shiftbaseUserID)
	req, err := http.NewRequest(http.MethodPost, rosterURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "API "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("claim open shift: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("shiftbase returned %d: %s", resp.StatusCode, b)
	}

	newCount := shift.InstancesRemaining - 1
	if newCount <= 0 {
		occID := shift.OccurrenceID
		if occID == "" {
			occID = shift.ID
		}
		if err := c.DeleteOpenShift(occID); err != nil {
			log.Printf("open shifts: auto-delete after last claim failed (shift %s): %v", shift.ID, err)
		}
	} else {
		if err := c.decrementOpenShiftInstances(shift, newCount); err != nil {
			log.Printf("open shifts: decrement instances failed (shift %s): %v", shift.ID, err)
		}
	}

	return nil
}

// decrementOpenShiftInstances updates instances_remaining on an open shift.
func (c *Client) decrementOpenShiftInstances(shift OpenShift, newCount int) error {
	type body struct {
		OpenShift struct {
			InstancesRemaining int `json:"instances_remaining"`
		} `json:"OpenShift"`
	}
	var b body
	b.OpenShift.InstancesRemaining = newCount

	payload, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	occID := shift.OccurrenceID
	if occID == "" {
		occID = shift.ID
	}
	url := c.baseURL + "/open_shifts/" + occID
	log.Printf("shiftbase PUT %s instances_remaining=%d", url, newCount)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "API "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("update open shift: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b2, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("shiftbase returned %d: %s", resp.StatusCode, b2)
	}
	return nil
}

// CreateOpenShift posts a new open shift to Shiftbase.
func (c *Client) CreateOpenShift(req CreateOpenShiftRequest) (*OpenShift, error) {
	instances := req.Instances
	if instances == 0 {
		instances = 1
	}
	type body struct {
		OpenShift struct {
			Date             string `json:"date"`
			DepartmentID     string `json:"department_id"`
			TeamID           string `json:"team_id"`
			ShiftID          string `json:"shift_id"`
			StartTime        string `json:"starttime"`
			EndTime          string `json:"endtime"`
			Break            string `json:"break"`
			Instances        int    `json:"instances"`
			ApprovalRequired bool   `json:"approval_required"`
			Description      string `json:"description"`
		} `json:"OpenShift"`
	}
	var b body
	b.OpenShift.Date = req.Date
	b.OpenShift.DepartmentID = req.DepartmentID
	b.OpenShift.TeamID = req.TeamID
	b.OpenShift.ShiftID = req.ShiftID
	b.OpenShift.StartTime = req.StartTime
	b.OpenShift.EndTime = req.EndTime
	b.OpenShift.Break = req.Break
	b.OpenShift.Instances = instances
	b.OpenShift.ApprovalRequired = req.ApprovalRequired
	b.OpenShift.Description = req.Description

	payload, err := json.Marshal(b)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	url := c.baseURL + "/open_shifts"
	log.Printf("shiftbase POST %s create open shift date=%s", url, req.Date)
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "API "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("create open shift: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("shiftbase returned %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		Data []struct {
			OpenShift struct {
				ID string `json:"id"`
			} `json:"OpenShift"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil || len(result.Data) == 0 {
		return &OpenShift{Date: req.Date}, nil
	}
	created, err := c.GetOpenShift(result.Data[0].OpenShift.ID)
	if err != nil {
		return &OpenShift{ID: result.Data[0].OpenShift.ID, Date: req.Date}, nil
	}
	return created, nil
}

// DeleteOpenShift removes an open shift from Shiftbase.
// Uses scope "original" which deletes the shift (and all instances for recurring shifts).
func (c *Client) DeleteOpenShift(id string) error {
	// id must be the occurrence_id (e.g. "86252227:2026-06-02" or plain "86252227")
	url := c.baseURL + "/open_shifts/" + id + "/original"
	log.Printf("shiftbase DELETE %s", url)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "API "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete open shift: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("shiftbase returned %d: %s", resp.StatusCode, b)
	}
	return nil
}

// ListShiftTemplates returns all shift templates with their associated team_id.
func (c *Client) ListShiftTemplates() ([]ShiftTemplate, error) {
	var shiftsRaw struct {
		Data []struct {
			Shift struct {
				ID           string `json:"id"`
				Name         string `json:"name"`
				LongName     string `json:"long_name"`
				StartTime    string `json:"starttime"`
				EndTime      string `json:"endtime"`
				Break        string `json:"break"`
				Color        string `json:"color"`
				DepartmentID string `json:"department_id"`
			} `json:"Shift"`
		} `json:"data"`
	}
	if err := c.get("/shifts", &shiftsRaw); err != nil {
		return nil, err
	}

	// Build dept→team map so each template carries a usable team_id.
	var teamsRaw struct {
		Data []struct {
			Team struct {
				ID           string `json:"id"`
				DepartmentID string `json:"department_id"`
			} `json:"Team"`
		} `json:"data"`
	}
	_ = c.get("/teams", &teamsRaw) // best-effort; ignore error
	deptTeam := make(map[string]string, len(teamsRaw.Data))
	for _, t := range teamsRaw.Data {
		if _, exists := deptTeam[t.Team.DepartmentID]; !exists {
			deptTeam[t.Team.DepartmentID] = t.Team.ID
		}
	}

	out := make([]ShiftTemplate, 0, len(shiftsRaw.Data))
	for _, d := range shiftsRaw.Data {
		breakMin, _ := strconv.Atoi(d.Shift.Break)
		out = append(out, ShiftTemplate{
			ID:        d.Shift.ID,
			Name:      d.Shift.Name,
			LongName:  d.Shift.LongName,
			StartTime: d.Shift.StartTime,
			EndTime:   d.Shift.EndTime,
			Break:     breakMin,
			Color:     d.Shift.Color,
			DeptID:    d.Shift.DepartmentID,
			TeamID:    deptTeam[d.Shift.DepartmentID],
		})
	}
	return out, nil
}
