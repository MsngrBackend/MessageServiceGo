package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"messageservice/internal/dto"
)

type ChatRepository struct {
	pool *pgxpool.Pool
}

func NewChatRepository(pool *pgxpool.Pool) *ChatRepository {
	return &ChatRepository{pool: pool}
}

func (r *ChatRepository) Create(ctx context.Context, name string) (*dto.Chat, error) {
	chat := &dto.Chat{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO chats (name) VALUES ($1) RETURNING id, name`,
		name,
	).Scan(&chat.ID, &chat.Name)
	if err != nil {
		return nil, err
	}
	return chat, nil
}

func (r *ChatRepository) GetByID(ctx context.Context, id int64) (*dto.Chat, error) {
	chat := &dto.Chat{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name FROM chats WHERE id = $1`,
		id,
	).Scan(&chat.ID, &chat.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return chat, err
}

func (r *ChatRepository) ListByUserID(ctx context.Context, userID string) ([]*dto.Chat, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT c.id, c.name
		 FROM chats c
		 JOIN chat_members cm ON cm.chat_id = c.id
		 WHERE cm.user_id = $1
		 ORDER BY c.id`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chats := make([]*dto.Chat, 0)
	for rows.Next() {
		c := &dto.Chat{}
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		chats = append(chats, c)
	}
	return chats, rows.Err()
}

func (r *ChatRepository) Delete(ctx context.Context, id int64) (bool, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM chats WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *ChatRepository) AddMember(ctx context.Context, chatID int64, userID string) (*dto.ChatMember, error) {
	m := &dto.ChatMember{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO chat_members (chat_id, user_id) VALUES ($1, $2) RETURNING id, chat_id, user_id`,
		chatID, userID,
	).Scan(&m.ID, &m.ChatID, &m.UserID)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (r *ChatRepository) RemoveMember(ctx context.Context, chatID int64, userID string) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM chat_members WHERE chat_id = $1 AND user_id = $2`,
		chatID, userID,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *ChatRepository) GetMember(ctx context.Context, chatID int64, userID string) (*dto.ChatMember, error) {
	m := &dto.ChatMember{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, chat_id, user_id FROM chat_members WHERE chat_id = $1 AND user_id = $2`,
		chatID, userID,
	).Scan(&m.ID, &m.ChatID, &m.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

func (r *ChatRepository) GetMembers(ctx context.Context, chatID int64) ([]*dto.ChatMember, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, chat_id, user_id FROM chat_members WHERE chat_id = $1 ORDER BY id`,
		chatID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]*dto.ChatMember, 0)
	for rows.Next() {
		m := &dto.ChatMember{}
		if err := rows.Scan(&m.ID, &m.ChatID, &m.UserID); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}
