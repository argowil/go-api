package user

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// Handler exposes user management over HTTP.
// All routes require at least admin role — enforced at the router level.
type Handler struct {
	service *Service
}

// NewHandler creates a user Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.List(r.Context())
	if err != nil {
		log.Printf("user list failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusOK, users)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	u, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		log.Printf("user get failed: id=%d err=%v", id, err)
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	respond(w, http.StatusOK, u)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	u, err := h.service.Create(r.Context(), req)
	if err != nil {
		log.Printf("user create failed: email=%s err=%v", req.Email, err)
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	log.Printf("user created: id=%d email=%s role=%s", u.ID, u.Email, u.Role)
	respond(w, http.StatusCreated, u)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	u, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		log.Printf("user update failed: id=%d err=%v", id, err)
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	log.Printf("user updated: id=%d", id)
	respond(w, http.StatusOK, u)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		log.Printf("user delete failed: id=%d err=%v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("user deleted: id=%d", id)
	w.WriteHeader(http.StatusNoContent)
}

func parseID(r *http.Request) (uint, error) {
	n, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	return uint(n), err
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
