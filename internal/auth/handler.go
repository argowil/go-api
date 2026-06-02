package auth

import (
	"encoding/json"
	"net/http"
)

// Handler exposes auth endpoints over HTTP.
type Handler struct {
	service *Service
}

// NewHandler creates an auth Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Login godoc
//
//	POST /auth/login
//	Body: { "email": "...", "password": "..." }
//	Returns a token pair on success.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	pair, err := h.service.Login(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	respond(w, http.StatusOK, pair)
}

// Me returns the currently authenticated user's token claims.
//
//	GET /auth/me  (requires Bearer token)
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}

	u, err := h.service.UserByID(r.Context(), claims.UserID)
	if err != nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	respond(w, http.StatusOK, map[string]any{
		"id":                   u.ID,
		"role":                 u.Role,
		"name":                 u.FullName(),
		"must_change_password": u.MustChangePassword,
	})
}

// Refresh exchanges a valid refresh token for a new token pair.
//
//	POST /auth/refresh
//	Body: { "refresh_token": "..." }
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		http.Error(w, "refresh_token is required", http.StatusBadRequest)
		return
	}
	pair, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	respond(w, http.StatusOK, pair)
}

// ChangePassword lets an authenticated user set a new password.
// On success must_change_password is cleared.
//
//	PUT /auth/change-password  (requires Bearer token)
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}

	var req struct {
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NewPassword == "" {
		http.Error(w, "new_password is required", http.StatusBadRequest)
		return
	}
	if len(req.NewPassword) < 6 {
		http.Error(w, "wachtwoord moet minimaal 6 tekens zijn", http.StatusBadRequest)
		return
	}

	if err := h.service.ChangePassword(r.Context(), claims.UserID, req.NewPassword); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
