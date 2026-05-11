package service

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Message struct {
	ID        int64
	ChatID    int64
	Content   string
	SenderID  string
	CreatedAt time.Time
}

type MessageService struct {
	db *sql.DB
}

func NewMessageService(db *sql.DB) *MessageService {
	return &MessageService{db: db}
}

func (s *MessageService) AddMessage(
	ctx context.Context,
	chatID int64,
	content string,
	senderID string,
) (*Message, error) {
	query := `
		INSERT INTO messages (chat_id, content, sender_id, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id, chat_id, content, sender_id, created_at
	`

	msg := &Message{}
	err := s.db.QueryRowContext(ctx, query, chatID, content, senderID).
		Scan(&msg.ID, &msg.ChatID, &msg.Content, &msg.SenderID, &msg.CreatedAt)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *MessageService) GetMessagesByChat(
	ctx context.Context,
	chatID int64,
	limit int,
	offset int,
) ([]*Message, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, chat_id, content, sender_id, created_at
		FROM messages
		WHERE chat_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.QueryContext(ctx, query, chatID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		err := rows.Scan(&msg.ID, &msg.ChatID, &msg.Content, &msg.SenderID, &msg.CreatedAt)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func (s *MessageService) UpdateMessage(
	ctx context.Context,
	messageID int64,
	content string,
) (*Message, error) {
	query := `
		UPDATE messages
		SET content = $1
		WHERE id = $2
		RETURNING id, chat_id, content, sender_id, created_at
	`

	msg := &Message{}
	err := s.db.QueryRowContext(ctx, query, content, messageID).
		Scan(&msg.ID, &msg.ChatID, &msg.Content, &msg.SenderID, &msg.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return msg, nil
}

func (s *MessageService) DeleteMessage(
	ctx context.Context,
	messageID int64,
) (bool, error) {
	query := `DELETE FROM messages WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, messageID)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}