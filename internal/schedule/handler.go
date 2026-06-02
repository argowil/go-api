// Package schedule serves roster and time-registration data.
// When Shiftbase is configured it proxies from the Shiftbase API.
// When SHIFTBASE_API_KEY is empty it falls back to the local MySQL database
// (useful during development or if you decide not to use Shiftbase).
package schedule

import (
	"encoding/json"
	"net/http"
	"time"

	"argowil/backend/internal/shiftbase"
)

// Handler serves schedule endpoints.
type Handler struct {
	sb *shiftbase.Client
}

// NewHandler creates a schedule Handler.
func NewHandler(sb *shiftbase.Client) *Handler {
	return &Handler{sb: sb}
}

// ListShifts godoc
//
//	GET /schedules?from=YYYY-MM-DD&to=YYYY-MM-DD
func (h *Handler) ListShifts(w http.ResponseWriter, r *http.Request) {
	from, to := dateRange(r)

	if !h.sb.Enabled() {
		respond(w, http.StatusOK, []any{})
		return
	}

	shifts, err := h.sb.ListShifts(from, to)
	if err != nil {
		http.Error(w, "could not fetch shifts from Shiftbase", http.StatusBadGateway)
		return
	}
	respond(w, http.StatusOK, shifts)
}

// ListTimeRegistrations godoc
//
//	GET /time-registrations?from=YYYY-MM-DD&to=YYYY-MM-DD
func (h *Handler) ListTimeRegistrations(w http.ResponseWriter, r *http.Request) {
	from, to := dateRange(r)

	if !h.sb.Enabled() {
		respond(w, http.StatusOK, []any{})
		return
	}

	regs, err := h.sb.ListTimeRegistrations(from, to)
	if err != nil {
		http.Error(w, "could not fetch time registrations from Shiftbase", http.StatusBadGateway)
		return
	}
	respond(w, http.StatusOK, regs)
}

// dateRange reads from/to query params and defaults to the current week.
func dateRange(r *http.Request) (string, string) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" {
		now := time.Now()
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		monday := now.AddDate(0, 0, -(weekday - 1))
		from = monday.Format("2006-01-02")
	}
	if to == "" {
		now := time.Now()
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		sunday := now.AddDate(0, 0, 7-weekday)
		to = sunday.Format("2006-01-02")
	}
	return from, to
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
