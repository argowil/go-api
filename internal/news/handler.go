package news

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"argowil/backend/internal/auth"
	"argowil/backend/internal/storage"
)

// Handler serves news endpoints.
type Handler struct {
	repo Repository
	s3   *storage.Client
}

// NewHandler creates a news Handler.
func NewHandler(repo Repository, s3 *storage.Client) *Handler {
	return &Handler{repo: repo, s3: s3}
}

// List godoc
//
//	GET /news
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	posts, err := h.repo.List(r.Context())
	if err != nil {
		log.Printf("news list failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if posts == nil {
		posts = []Post{}
	}
	respond(w, http.StatusOK, posts)
}

// Get godoc
//
//	GET /news/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	post, err := h.repo.Get(r.Context(), uint(id))
	if err != nil {
		log.Printf("news get failed: id=%d err=%v", id, err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	respond(w, http.StatusOK, post)
}

// Create godoc
//
//	POST /news  (teamleader / admin)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}

	var req CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Body) == "" {
		http.Error(w, "title and body are required", http.StatusBadRequest)
		return
	}

	post := &Post{
		Title:    req.Title,
		Body:     req.Body,
		ImageURL: req.ImageURL,
		AuthorID: claims.UserID,
	}

	if err := h.repo.Create(r.Context(), post); err != nil {
		log.Printf("news create failed: author_id=%d title=%q err=%v", claims.UserID, req.Title, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("news created: id=%d author_id=%d title=%q", post.ID, claims.UserID, post.Title)
	respond(w, http.StatusCreated, post)
}

// Update godoc
//
//	PATCH /news/{id}  (teamleader / admin)
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		Title    string  `json:"title"`
		Body     string  `json:"body"`
		ImageURL *string `json:"image_url,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Body) == "" {
		http.Error(w, "title and body are required", http.StatusBadRequest)
		return
	}
	if err := h.repo.Update(r.Context(), uint(id), req.Title, req.Body, req.ImageURL); err != nil {
		log.Printf("news update failed: id=%d err=%v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("news updated: id=%d", id)
	w.WriteHeader(http.StatusNoContent)
}

// Delete godoc
//
//	DELETE /news/{id}  (teamleader / admin)
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.repo.Delete(r.Context(), uint(id)); err != nil {
		log.Printf("news delete failed: id=%d err=%v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("news deleted: id=%d", id)
	w.WriteHeader(http.StatusNoContent)
}

// CreateComment godoc
//
//	POST /news/{id}/comments
func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Body) == "" {
		http.Error(w, "comment body is required", http.StatusBadRequest)
		return
	}

	comment := &Comment{
		PostID:   uint(id),
		Body:     strings.TrimSpace(req.Body),
		AuthorID: claims.UserID,
	}
	if err := h.repo.CreateComment(r.Context(), comment); err != nil {
		log.Printf("news comment failed: post_id=%d author_id=%d err=%v", id, claims.UserID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("news comment: id=%d post_id=%d author_id=%d", comment.ID, id, claims.UserID)
	respond(w, http.StatusCreated, comment)
}

// UploadImage receives a multipart image, uploads it to S3 and returns the public URL.
//
//	POST /news/upload  (teamleader / admin)
func (h *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "file too large (max 10 MB)", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "image field is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		http.Error(w, "only jpg, png and webp are allowed", http.StatusBadRequest)
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "could not read file", http.StatusInternalServerError)
		return
	}

	url, err := h.s3.Upload(r.Context(), data, ext)
	if err != nil {
		log.Printf("news upload failed: file=%s size=%d err=%v", header.Filename, len(data), err)
		http.Error(w, "upload failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("news upload: file=%s size=%d url=%s", header.Filename, len(data), url)
	respond(w, http.StatusOK, map[string]string{
		"url":      url,
		"filename": url,
	})
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
