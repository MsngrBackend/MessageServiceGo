package service

import (
	"context"
	"errors"

	"messageservice/internal/dto"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageService struct {
	pool *pgxpool.Pool
}

func NewMessageService(pool *pgxpool.Pool) *MessageService {
	return &MessageService{pool: pool}
}

func (s *MessageService) AddMessage(ctx context.Context, chatID int64, content, senderID string) (*dto.Message, error) {
	msg := &dto.Message{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO messages (chat_id, content, sender_id, created_at)
		 VALUES ($1, $2, $3, NOW())
		 RETURNING id, chat_id, content, sender_id, created_at`,
		chatID, content, senderID,
	).Scan(&msg.ID, &msg.ChatID, &msg.Content, &msg.SenderID, &msg.CreatedAt)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (s *MessageService) GetMessagesByChat(ctx context.Context, chatID int64, limit, offset int) ([]*dto.Message, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, chat_id, content, sender_id, created_at
		 FROM messages
		 WHERE chat_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		chatID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]*dto.Message, 0)
	for rows.Next() {
		msg := &dto.Message{}
		if err := rows.Scan(&msg.ID, &msg.ChatID, &msg.Content, &msg.SenderID, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (s *MessageService) UpdateMessage(ctx context.Context, messageID int64, content string) (*dto.Message, error) {
	msg := &dto.Message{}
	err := s.pool.QueryRow(ctx,
		`UPDATE messages SET content = $1 WHERE id = $2
		 RETURNING id, chat_id, content, sender_id, created_at`,
		content, messageID,
	).Scan(&msg.ID, &msg.ChatID, &msg.Content, &msg.SenderID, &msg.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return msg, nil
}

func (s *MessageService) DeleteMessage(ctx context.Context, messageID int64) (bool, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM messages WHERE id = $1`, messageID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
