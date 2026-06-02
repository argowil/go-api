package shiftbase

import (
	"fmt"
	"strconv"
)

// FirstTeamID returns the ID of the first team in Shiftbase.
func (c *Client) FirstTeamID() (int, error) {
	var resp struct {
		Data []struct {
			Team struct {
				ID string `json:"id"`
			} `json:"Team"`
		} `json:"data"`
	}
	if err := c.get("/teams", &resp); err != nil {
		return 0, err
	}
	if len(resp.Data) == 0 {
		return 0, fmt.Errorf("no teams found in Shiftbase")
	}
	id, err := strconv.Atoi(resp.Data[0].Team.ID)
	if err != nil {
		return 0, fmt.Errorf("parse team id %q: %w", resp.Data[0].Team.ID, err)
	}
	return id, nil
}
