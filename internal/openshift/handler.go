// Package openshift serves open shift endpoints.
// Employees can view and claim open shifts; teamleaders can create and delete them.
package openshift

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"argowil/backend/internal/auth"
	"argowil/backend/internal/shiftbase"
	"argowil/backend/internal/user"
)

// Handler serves open shift endpoints.
type Handler struct {
	sb    *shiftbase.Client
	users user.Repository
}

// NewHandler creates an open shift Handler.
func NewHandler(sb *shiftbase.Client, users user.Repository) *Handler {
	return &Handler{sb: sb, users: users}
}

// ListOpenShifts returns open shifts for the next 4 weeks.
//
//	GET /open-shifts?from=YYYY-MM-DD&to=YYYY-MM-DD
func (h *Handler) ListOpenShifts(w http.ResponseWriter, r *http.Request) {
	if !h.sb.Enabled() {
		respond(w, http.StatusOK, []any{})
		return
	}

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" {
		from = time.Now().Format("2006-01-02")
	}
	if to == "" {
		to = time.Now().AddDate(0, 0, 28).Format("2006-01-02")
	}

	shifts, err := h.sb.ListOpenShifts(from, to)
	if err != nil {
		log.Printf("open shifts: list failed: %v", err)
		http.Error(w, "could not fetch open shifts", http.StatusBadGateway)
		return
	}
	respond(w, http.StatusOK, shifts)
}

// ClaimOpenShift lets an employee sign up for an open shift.
//
//	POST /open-shifts/{id}/claim
func (h *Handler) ClaimOpenShift(w http.ResponseWriter, r *http.Request) {
	if !h.sb.Enabled() {
		http.Error(w, "Shiftbase not configured", http.StatusServiceUnavailable)
		return
	}

	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}

	u, err := h.users.FindByID(r.Context(), claims.UserID)
	if err != nil {
		http.Error(w, "user not found", http.StatusInternalServerError)
		return
	}
	if u.ShiftbaseEmployeeID == nil {
		http.Error(w, "account has no Shiftbase employee linked", http.StatusUnprocessableEntity)
		return
	}

	shiftID := chi.URLParam(r, "id")
	shift, err := h.sb.GetOpenShift(shiftID)
	if err != nil {
		log.Printf("open shifts: fetch shift %s failed: %v", shiftID, err)
		http.Error(w, "open shift not found", http.StatusNotFound)
		return
	}

	if shift.InstancesRemaining <= 0 {
		http.Error(w, "no spots remaining for this shift", http.StatusConflict)
		return
	}

	if err := h.sb.ClaimOpenShift(*u.ShiftbaseEmployeeID, *shift); err != nil {
		log.Printf("open shifts: claim %s by user %d failed: %v", shiftID, claims.UserID, err)
		http.Error(w, fmt.Sprintf("could not claim shift: %v", err), http.StatusBadGateway)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateOpenShiftRequest is the payload for creating an open shift.
type CreateOpenShiftRequest struct {
	Date             string `json:"date"`
	ShiftTemplateID  string `json:"shift_template_id"`
	DepartmentID     string `json:"department_id"`
	TeamID           string `json:"team_id"`
	StartTime        string `json:"start_time"`
	EndTime          string `json:"end_time"`
	Break            string `json:"break"`
	Instances        int    `json:"instances"`
	ApprovalRequired bool   `json:"approval_required"`
	Description      string `json:"description"`
}

// CreateOpenShift lets a teamleader create a new open shift.
//
//	POST /open-shifts
func (h *Handler) CreateOpenShift(w http.ResponseWriter, r *http.Request) {
	if !h.sb.Enabled() {
		http.Error(w, "Shiftbase not configured", http.StatusServiceUnavailable)
		return
	}

	var req CreateOpenShiftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Date == "" || req.ShiftTemplateID == "" {
		http.Error(w, "date and shift_template_id are required", http.StatusBadRequest)
		return
	}

	shift, err := h.sb.CreateOpenShift(shiftbase.CreateOpenShiftRequest{
		Date:             req.Date,
		DepartmentID:     req.DepartmentID,
		TeamID:           req.TeamID,
		ShiftID:          req.ShiftTemplateID,
		StartTime:        req.StartTime,
		EndTime:          req.EndTime,
		Break:            req.Break,
		Instances:        req.Instances,
		ApprovalRequired: req.ApprovalRequired,
		Description:      req.Description,
	})
	if err != nil {
		log.Printf("open shifts: create failed: %v", err)
		http.Error(w, fmt.Sprintf("could not create open shift: %v", err), http.StatusBadGateway)
		return
	}
	respond(w, http.StatusCreated, shift)
}

// DeleteOpenShift lets a teamleader remove an open shift.
//
//	DELETE /open-shifts/{id}
func (h *Handler) DeleteOpenShift(w http.ResponseWriter, r *http.Request) {
	if !h.sb.Enabled() {
		http.Error(w, "Shiftbase not configured", http.StatusServiceUnavailable)
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.sb.DeleteOpenShift(id); err != nil {
		log.Printf("open shifts: delete %s failed: %v", id, err)
		http.Error(w, fmt.Sprintf("could not delete open shift: %v", err), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListShiftTemplates returns available shift templates.
//
//	GET /open-shifts/templates
func (h *Handler) ListShiftTemplates(w http.ResponseWriter, r *http.Request) {
	if !h.sb.Enabled() {
		respond(w, http.StatusOK, []any{})
		return
	}
	templates, err := h.sb.ListShiftTemplates()
	if err != nil {
		http.Error(w, "could not fetch shift templates", http.StatusBadGateway)
		return
	}
	respond(w, http.StatusOK, templates)
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
