package dto

import "time"

type Chat struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ChatMember struct {
	ID     int64  `json:"id"`
	ChatID int64  `json:"chat_id"`
	UserID string `json:"user_id"`
}

type Message struct {
	ID        int64     `json:"id"`
	ChatID    int64     `json:"chat_id"`
	Content   string    `json:"content"`
	SenderID  string    `json:"sender_id"`
	CreatedAt time.Time `json:"created_at"`
}
