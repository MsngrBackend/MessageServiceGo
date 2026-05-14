package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
)

type Bus struct {
	nc *nats.Conn
}

func Connect(url string) (*Bus, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	return &Bus{nc: nc}, nil
}

func (b *Bus) Close() {
	b.nc.Drain()
}

func (b *Bus) NotifyChatEvent(ctx context.Context, chatID int64, payload map[string]any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return b.nc.Publish(fmt.Sprintf("msngr.chat.%d.event", chatID), data)
}

func (b *Bus) PublishUserStatus(ctx context.Context, userID string, online bool) error {
	data, err := json.Marshal(map[string]any{
		"user_id": userID,
		"online":  online,
	})
	if err != nil {
		return err
	}
	return b.nc.Publish("msngr.user.status", data)
}
