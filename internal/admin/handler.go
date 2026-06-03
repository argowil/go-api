// Package admin exposes endpoints for the admin/teamleader management panel.
// All routes require at least the teamleader role.
package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"strconv"

	"github.com/go-chi/chi/v5"

	"argowil/backend/internal/shiftbase"
	"argowil/backend/internal/user"
)

// CreateEmployeeRequest is the payload the mobile app sends when an admin
// creates a new employee account.
type CreateEmployeeRequest struct {
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	Email         string `json:"email"`
	Password      string `json:"password"`
	Role          string `json:"role"`
	TeamID                 string `json:"team_id"`
	CreateShiftbaseAccount *bool  `json:"create_shiftbase_account"`
}

// Handler handles admin panel API requests.
type Handler struct {
	users                user.Repository
	sb                   *shiftbase.Client
	defaultDepartmentIDs []int
}

// NewHandler creates an admin Handler.
// defaultDepartmentID is used when the create-employee request omits department_ids;
// pass 0 to send no default (Shiftbase will reject if no team is configured there).
func NewHandler(users user.Repository, sb *shiftbase.Client, defaultDepartmentID int) *Handler {
	var defaults []int
	if defaultDepartmentID > 0 {
		defaults = []int{defaultDepartmentID}
	}
	return &Handler{users: users, sb: sb, defaultDepartmentIDs: defaults}
}

// ListEmployees returns all users.
//
//	GET /admin/employees
func (h *Handler) ListEmployees(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusOK, users)
}

// CreateEmployee creates a new employee account in the local database and,
// when Shiftbase is configured, also creates the employee there and links the
// two accounts via the returned Shiftbase employee ID.
//
//	POST /admin/employees
func (h *Handler) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	var req CreateEmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.FirstName == "" || req.LastName == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "first_name, last_name, email and password are required", http.StatusBadRequest)
		return
	}

	var shiftbaseID *int

	wantShiftbase := h.sb.Enabled() && (req.CreateShiftbaseAccount == nil || *req.CreateShiftbaseAccount)
	if wantShiftbase {
		log.Printf("admin create employee: attempting Shiftbase create for %s", req.Email)
		teamID := req.TeamID
		if teamID == "" && len(h.defaultDepartmentIDs) > 0 {
			teamID = strconv.Itoa(h.defaultDepartmentIDs[0])
		}
		sbID, err := h.sb.CreateEmployee(shiftbase.CreateEmployeeRequest{
			FirstName: req.FirstName,
			LastName:  req.LastName,
			Email:     req.Email,
			TeamID:    teamID,
		})
		if err != nil {
			log.Printf("admin create employee: Shiftbase create failed for %s: %v", req.Email, err)
			http.Error(w, fmt.Sprintf("could not create employee in Shiftbase: %v", err), http.StatusBadGateway)
			return
		}
		log.Printf("admin create employee: Shiftbase create succeeded for %s with id=%d", req.Email, sbID)
		shiftbaseID = &sbID
	}

	svc := user.NewService(h.users)
	u, err := svc.Create(context.Background(), user.CreateUserRequest{
		FirstName:           req.FirstName,
		LastName:            req.LastName,
		Email:               req.Email,
		Password:            req.Password,
		Role:                roleOrDefault(req.Role),
		ShiftbaseEmployeeID: shiftbaseID,
		MustChangePassword:  true,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("could not create user: %v", err), http.StatusUnprocessableEntity)
		return
	}

	respond(w, http.StatusCreated, u)
}

// GetEmployee returns a single employee.
//
//	GET /admin/employees/{id}
func (h *Handler) GetEmployee(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	u, err := h.users.FindByIDAdmin(r.Context(), id)
	if err != nil {
		http.Error(w, "employee not found", http.StatusNotFound)
		return
	}
	respond(w, http.StatusOK, u)
}

// UpdateEmployee updates name, email or role of an existing employee.
//
//	PATCH /admin/employees/{id}
func (h *Handler) UpdateEmployee(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req user.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	svc := user.NewService(h.users)
	u, err := svc.Update(context.Background(), id, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	respond(w, http.StatusOK, u)
}

// SyncShiftbase creates a Shiftbase account for an existing employee and links the ID.
//
//	POST /admin/employees/{id}/sync-shiftbase
func (h *Handler) SyncShiftbase(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if !h.sb.Enabled() {
		http.Error(w, "Shiftbase is not configured", http.StatusServiceUnavailable)
		return
	}

	u, err := h.users.FindByIDAdmin(r.Context(), id)
	if err != nil {
		http.Error(w, "employee not found", http.StatusNotFound)
		return
	}
	if u.ShiftbaseEmployeeID != nil {
		http.Error(w, "employee already linked to Shiftbase", http.StatusConflict)
		return
	}

	teamID := ""
	if len(h.defaultDepartmentIDs) > 0 {
		teamID = strconv.Itoa(h.defaultDepartmentIDs[0])
	}

	sbID, err := h.sb.CreateEmployee(shiftbase.CreateEmployeeRequest{
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Email:     u.Email,
		TeamID:    teamID,
	})
	if err != nil {
		// 422 = email already exists in Shiftbase — look up the existing employee instead
		if isShiftbase422(err) {
			existing, lookupErr := h.sb.FindEmployeeByEmail(u.Email)
			if lookupErr != nil {
				log.Printf("admin sync shiftbase: lookup failed for user_id=%d: %v", id, lookupErr)
				http.Error(w, "email already exists in Shiftbase but lookup failed", http.StatusBadGateway)
				return
			}
			if existing == nil {
				http.Error(w, "email already exists in Shiftbase but could not find the account", http.StatusBadGateway)
				return
			}
			log.Printf("admin sync shiftbase: linking existing shiftbase_id=%d for user_id=%d", existing.ID, id)
			sbID = existing.ID
		} else {
			log.Printf("admin sync shiftbase: failed for user_id=%d: %v", id, err)
			http.Error(w, fmt.Sprintf("could not create Shiftbase employee: %v", err), http.StatusBadGateway)
			return
		}
	}

	if err := h.users.SetShiftbaseID(r.Context(), id, sbID); err != nil {
		log.Printf("admin sync shiftbase: db update failed for user_id=%d: %v", id, err)
		http.Error(w, "could not link Shiftbase ID", http.StatusInternalServerError)
		return
	}

	log.Printf("admin sync shiftbase: user_id=%d linked to shiftbase_id=%d", id, sbID)
	respond(w, http.StatusOK, map[string]int{"shiftbase_employee_id": sbID})
}

// DeleteEmployee soft-deletes an employee account and removes them from Shiftbase.
//
//	DELETE /admin/employees/{id}
func (h *Handler) DeleteEmployee(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if h.sb.Enabled() {
		u, err := h.users.FindByIDAdmin(r.Context(), id)
		if err != nil {
			http.Error(w, "employee not found", http.StatusNotFound)
			return
		}
		if u.ShiftbaseEmployeeID != nil {
			if err := h.sb.DeleteEmployee(*u.ShiftbaseEmployeeID); err != nil {
				log.Printf("admin delete employee: Shiftbase delete failed for id=%d: %v", id, err)
				http.Error(w, fmt.Sprintf("could not delete employee from Shiftbase: %v", err), http.StatusBadGateway)
				return
			}
		}
	}

	if err := h.users.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func isShiftbase422(err error) bool {
	return err != nil && strings.Contains(err.Error(), "422")
}

func parseID(r *http.Request) (uint, error) {
	n, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	return uint(n), err
}

func roleOrDefault(role string) string {
	switch role {
	case "admin", "teamleader", "employee":
		return role
	default:
		return "employee"
	}
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
