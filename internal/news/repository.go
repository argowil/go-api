package news

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Repository handles news post persistence.
type Repository interface {
	List(ctx context.Context) ([]Post, error)
	Get(ctx context.Context, id uint) (*PostDetail, error)
	Create(ctx context.Context, p *Post) error
	Update(ctx context.Context, id uint, title, body string, imageURL *string) error
	CreateComment(ctx context.Context, c *Comment) error
	Delete(ctx context.Context, id uint) error
}

type mysqlRepository struct {
	db *sqlx.DB
}

// NewRepository returns a MySQL-backed Repository.
func NewRepository(db *sqlx.DB) Repository {
	return &mysqlRepository{db: db}
}

func (r *mysqlRepository) List(ctx context.Context) ([]Post, error) {
	var posts []Post
	err := r.db.SelectContext(ctx, &posts, `
		SELECT n.id, n.title, n.body, n.image_url, n.author_id, n.created_at,
		       CONCAT(u.first_name, ' ', u.last_name) AS author_name,
		       COUNT(c.id) AS comment_count
		FROM news_posts n
		JOIN users u ON u.id = n.author_id
		LEFT JOIN news_comments c ON c.post_id = n.id
		GROUP BY n.id, n.title, n.body, n.image_url, n.author_id, n.created_at, u.first_name, u.last_name
		ORDER BY n.created_at DESC
		LIMIT 50
	`)
	if err != nil {
		return nil, fmt.Errorf("list news: %w", err)
	}
	return posts, nil
}

func (r *mysqlRepository) Get(ctx context.Context, id uint) (*PostDetail, error) {
	var post PostDetail
	err := r.db.GetContext(ctx, &post, `
		SELECT n.id, n.title, n.body, n.image_url, n.author_id, n.created_at,
		       CONCAT(u.first_name, ' ', u.last_name) AS author_name,
		       COUNT(c.id) AS comment_count
		FROM news_posts n
		JOIN users u ON u.id = n.author_id
		LEFT JOIN news_comments c ON c.post_id = n.id
		WHERE n.id = ?
		GROUP BY n.id, n.title, n.body, n.image_url, n.author_id, n.created_at, u.first_name, u.last_name
		LIMIT 1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("get news post: %w", err)
	}

	var comments []Comment
	err = r.db.SelectContext(ctx, &comments, `
		SELECT c.id, c.post_id, c.body, c.author_id, c.created_at,
		       CONCAT(u.first_name, ' ', u.last_name) AS author_name
		FROM news_comments c
		JOIN users u ON u.id = c.author_id
		WHERE c.post_id = ?
		ORDER BY c.created_at ASC, c.id ASC
	`, id)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	if comments == nil {
		comments = []Comment{}
	}
	post.Comments = comments
	return &post, nil
}

func (r *mysqlRepository) Create(ctx context.Context, p *Post) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO news_posts (title, body, image_url, author_id) VALUES (?, ?, ?, ?)`,
		p.Title, p.Body, p.ImageURL, p.AuthorID,
	)
	if err != nil {
		return fmt.Errorf("create post: %w", err)
	}
	id, _ := result.LastInsertId()
	p.ID = uint(id)
	return nil
}

func (r *mysqlRepository) CreateComment(ctx context.Context, c *Comment) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO news_comments (post_id, body, author_id) VALUES (?, ?, ?)`,
		c.PostID, c.Body, c.AuthorID,
	)
	if err != nil {
		return fmt.Errorf("create comment: %w", err)
	}
	id, _ := result.LastInsertId()
	c.ID = uint(id)

	err = r.db.GetContext(ctx, c, `
		SELECT c.id, c.post_id, c.body, c.author_id, c.created_at,
		       CONCAT(u.first_name, ' ', u.last_name) AS author_name
		FROM news_comments c
		JOIN users u ON u.id = c.author_id
		WHERE c.id = ?
		LIMIT 1
	`, c.ID)
	if err != nil {
		return fmt.Errorf("load comment: %w", err)
	}
	return nil
}

func (r *mysqlRepository) Update(ctx context.Context, id uint, title, body string, imageURL *string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE news_posts SET title=?, body=?, image_url=? WHERE id=?`,
		title, body, imageURL, id,
	)
	if err != nil {
		return fmt.Errorf("update post: %w", err)
	}
	return nil
}

func (r *mysqlRepository) Delete(ctx context.Context, id uint) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM news_posts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete post: %w", err)
	}
	return nil
}
