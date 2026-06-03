// Package community provides a real-time group chat for all employees.
package community

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"argowil/backend/internal/auth"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Handler struct {
	repo   *Repository
	hub    *Hub
	secret string
}

func NewHandler(repo *Repository, hub *Hub, jwtSecret string) *Handler {
	return &Handler{repo: repo, hub: hub, secret: jwtSecret}
}

// Members returns all active users with their online status.
//
//	GET /community/members
func (h *Handler) Members(w http.ResponseWriter, r *http.Request) {
	type member struct {
		ID     uint   `json:"id"`
		Name   string `json:"name"`
		Role   string `json:"role"`
		Online bool   `json:"online"`
	}

	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}

	type row struct {
		ID        uint   `db:"id"`
		FirstName string `db:"first_name"`
		LastName  string `db:"last_name"`
		Role      string `db:"role"`
	}
	var rows []row
	if err := h.repo.db.SelectContext(r.Context(), &rows,
		`SELECT id, first_name, last_name, role FROM users WHERE active = 1 ORDER BY first_name, last_name`,
	); err != nil {
		http.Error(w, "could not load members", http.StatusInternalServerError)
		return
	}

	online := h.hub.OnlineIDs()
	out := make([]member, 0, len(rows))
	for _, row := range rows {
		out = append(out, member{
			ID:     row.ID,
			Name:   row.FirstName + " " + row.LastName,
			Role:   row.Role,
			Online: online[row.ID],
		})
	}
	respond(w, out)
}

// History returns the last 50 messages.
//
//	GET /community/messages
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	msgs, err := h.repo.History(r.Context(), 50)
	if err != nil {
		http.Error(w, "could not load messages", http.StatusInternalServerError)
		return
	}
	respond(w, msgs)
}

// Connect upgrades to WebSocket.
// Auth via ?token=<access_token> since WS clients can't set headers.
//
//	GET /community/ws?token=...
func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		// also accept Bearer header for flexibility
		token = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	claims, err := auth.ParseAccessToken(h.secret, token)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("community ws upgrade failed: user_id=%d err=%v", claims.UserID, err)
		return
	}
	defer conn.Close()

	ch := h.hub.Subscribe(claims.UserID)
	defer h.hub.Unsubscribe(ch)

	log.Printf("community ws connected: user_id=%d name=%q", claims.UserID, claims.Name)
	defer log.Printf("community ws disconnected: user_id=%d name=%q", claims.UserID, claims.Name)

	h.hub.Broadcast(WSEvent{Type: "presence", UserID: claims.UserID, Online: true})
	defer h.hub.Broadcast(WSEvent{Type: "presence", UserID: claims.UserID, Online: false})

	// Write pump — send broadcasts to this client.
	go func() {
		for b := range ch {
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
				conn.Close()
				return
			}
		}
	}()

	// Read pump — receive messages from this client.
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var req struct {
			Content   string `json:"content"`
			ReplyToID *uint  `json:"reply_to_id"`
		}
		if json.Unmarshal(raw, &req) != nil || strings.TrimSpace(req.Content) == "" {
			continue
		}
		if len(req.Content) > 2000 {
			continue
		}

		msg := Message{
			UserID:   claims.UserID,
			UserName: claims.Name,
			Content:  strings.TrimSpace(req.Content),
		}

		// Attach reply preview if replying to another message.
		if req.ReplyToID != nil {
			if ref, err := h.repo.FindByID(context.Background(), *req.ReplyToID); err == nil {
				msg.ReplyToID = req.ReplyToID
				preview := ref.UserName + ": " + ref.Content
				if len(preview) > 200 {
					preview = preview[:197] + "..."
				}
				msg.ReplyPreview = &preview
			}
		}

		if err := h.repo.Save(context.Background(), &msg); err != nil {
			log.Printf("community save failed: user_id=%d err=%v", claims.UserID, err)
			continue
		}
		msg.CreatedAt = time.Now()
		log.Printf("community message: id=%d user_id=%d len=%d", msg.ID, claims.UserID, len(msg.Content))
		h.hub.Broadcast(WSEvent{Type: "new", Message: &msg})
	}
}

// EditMessage lets a user edit their own message.
//
//	PATCH /community/messages/{id}
func (h *Handler) EditMessage(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct{ Content string `json:"content"` }
	if json.NewDecoder(r.Body).Decode(&req) != nil || strings.TrimSpace(req.Content) == "" {
		http.Error(w, "content required", http.StatusBadRequest)
		return
	}
	if err := h.repo.Edit(r.Context(), uint(id), claims.UserID, strings.TrimSpace(req.Content)); err != nil {
		log.Printf("community edit failed: msg_id=%d user_id=%d err=%v", id, claims.UserID, err)
		http.Error(w, "not found or forbidden", http.StatusForbidden)
		return
	}
	log.Printf("community edit: msg_id=%d user_id=%d", id, claims.UserID)
	h.hub.Broadcast(WSEvent{Type: "edit", ID: uint(id), Message: &Message{
		ID: uint(id), Content: strings.TrimSpace(req.Content), Edited: true,
	}})
	w.WriteHeader(http.StatusNoContent)
}

// DeleteMessage lets a user delete their own message.
//
//	DELETE /community/messages/{id}
func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.repo.Delete(r.Context(), uint(id), claims.UserID); err != nil {
		log.Printf("community delete failed: msg_id=%d user_id=%d err=%v", id, claims.UserID, err)
		http.Error(w, "not found or forbidden", http.StatusForbidden)
		return
	}
	log.Printf("community delete: msg_id=%d user_id=%d", id, claims.UserID)
	h.hub.Broadcast(WSEvent{Type: "delete", ID: uint(id)})
	w.WriteHeader(http.StatusNoContent)
}

func respond(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(body)
}
