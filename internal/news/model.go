// Package news manages the home feed posts.
package news

import "time"

// Post is a news item shown on the employee home screen.
type Post struct {
	ID           uint      `db:"id"            json:"id"`
	Title        string    `db:"title"         json:"title"`
	Body         string    `db:"body"          json:"body"`
	ImageURL     *string   `db:"image_url"     json:"image_url,omitempty"`
	AuthorID     uint      `db:"author_id"     json:"author_id"`
	AuthorName   string    `db:"author_name"   json:"author_name"`
	CreatedAt    time.Time `db:"created_at"    json:"created_at"`
	CommentCount int       `db:"comment_count" json:"comment_count"`
}

// Comment is a response beneath a news post.
type Comment struct {
	ID         uint      `db:"id"          json:"id"`
	PostID     uint      `db:"post_id"     json:"post_id"`
	Body       string    `db:"body"        json:"body"`
	AuthorID   uint      `db:"author_id"   json:"author_id"`
	AuthorName string    `db:"author_name" json:"author_name"`
	CreatedAt  time.Time `db:"created_at"  json:"created_at"`
}

// PostDetail is the full news post view including comments.
type PostDetail struct {
	Post
	Comments []Comment `json:"comments"`
}

// CreatePostRequest is the payload for a new news post.
type CreatePostRequest struct {
	Title    string  `json:"title"`
	Body     string  `json:"body"`
	ImageURL *string `json:"image_url,omitempty"`
}

// CreateCommentRequest is the payload for a new comment.
type CreateCommentRequest struct {
	Body string `json:"body"`
}
