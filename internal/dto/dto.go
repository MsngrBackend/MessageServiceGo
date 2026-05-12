package dto

type Chat struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ChatMember struct {
	ID     int64  `json:"id"`
	ChatID int64  `json:"chat_id"`
	UserID string `json:"user_id"`
}
