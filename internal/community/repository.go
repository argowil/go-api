package community

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Repository struct{ db *sqlx.DB } //nolint:unused

func NewRepository(db *sqlx.DB) *Repository { return &Repository{db: db} }

func (r *Repository) History(ctx context.Context, limit int) ([]Message, error) {
	var msgs []Message
	err := r.db.SelectContext(ctx, &msgs,
		`SELECT * FROM messages ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("history: %w", err)
	}
	// reverse to ascending
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (r *Repository) Save(ctx context.Context, m *Message) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO messages (user_id, user_name, content, reply_to_id, reply_preview) VALUES (?, ?, ?, ?, ?)`,
		m.UserID, m.UserName, m.Content, m.ReplyToID, m.ReplyPreview)
	if err != nil {
		return fmt.Errorf("save message: %w", err)
	}
	id, _ := res.LastInsertId()
	m.ID = uint(id)
	return nil
}

func (r *Repository) Edit(ctx context.Context, id, userID uint, content string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE messages SET content=?, edited=1 WHERE id=? AND user_id=?`,
		content, id, userID)
	if err != nil {
		return fmt.Errorf("edit message: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not found or not owner")
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id, userID uint) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM messages WHERE id=? AND user_id=?`, id, userID)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not found or not owner")
	}
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id uint) (*Message, error) {
	var m Message
	err := r.db.GetContext(ctx, &m, `SELECT * FROM messages WHERE id=?`, id)
	if err != nil {
		return nil, fmt.Errorf("message not found: %w", err)
	}
	return &m, nil
}
