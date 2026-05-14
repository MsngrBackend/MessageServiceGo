package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/websocket"

	handler "messageservice/internal/httpapi"
	inats "messageservice/internal/nats"
	"messageservice/internal/repository"
	"messageservice/internal/service"
	"messageservice/internal/ws"
)

func main() {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:ultramegasecret@localhost:5432/messaging_db"
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	bus, err := inats.Connect(natsURL)
	if err != nil {
		log.Printf("NATS unavailable, running without it: %v", err)
		bus = nil
	} else {
		defer bus.Close()
	}

	chatRepo := repository.NewChatRepository(pool)
	chatSvc := service.NewChatService(chatRepo)
	msgSvc := service.NewMessageService(pool)

	mux := http.NewServeMux()

	handler.NewChatHandler(chatSvc).RegisterRoutes(mux)
	handler.NewMessageHandler(msgSvc, bus).RegisterRoutes(mux)
	mux.Handle("/ws/chat", websocket.Handler(ws.WsHandler))

	addr := ":8080"
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
