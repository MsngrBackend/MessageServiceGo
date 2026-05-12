package service

import (
	"context"
	"errors"

	"messageservice/internal/dto"
	"messageservice/internal/repository"
)

var (
	ErrNotFound      = errors.New("chat not found")
	ErrNotMember     = errors.New("you are not a member of this chat")
	ErrAlreadyMember = errors.New("user is already a member of this chat")
)

type ChatService struct {
	repo *repository.ChatRepository
}

func NewChatService(repo *repository.ChatRepository) *ChatService {
	return &ChatService{repo: repo}
}

func (s *ChatService) CreateChat(ctx context.Context, name, creatorID string) (*dto.Chat, error) {
	chat, err := s.repo.Create(ctx, name)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.AddMember(ctx, chat.ID, creatorID); err != nil {
		return nil, err
	}
	return chat, nil
}

func (s *ChatService) GetChat(ctx context.Context, chatID int64, userID string) (*dto.Chat, error) {
	chat, err := s.repo.GetByID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if chat == nil {
		return nil, ErrNotFound
	}

	member, err := s.repo.GetMember(ctx, chatID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrNotMember
	}

	return chat, nil
}

func (s *ChatService) ListUserChats(ctx context.Context, userID string) ([]*dto.Chat, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *ChatService) DeleteChat(ctx context.Context, chatID int64, userID string) error {
	chat, err := s.repo.GetByID(ctx, chatID)
	if err != nil {
		return err
	}
	if chat == nil {
		return ErrNotFound
	}

	member, err := s.repo.GetMember(ctx, chatID, userID)
	if err != nil {
		return err
	}
	if member == nil {
		return ErrNotMember
	}

	_, err = s.repo.Delete(ctx, chatID)
	return err
}

func (s *ChatService) AddMember(ctx context.Context, chatID int64, userID string) (*dto.ChatMember, error) {
	existing, err := s.repo.GetMember(ctx, chatID, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrAlreadyMember
	}
	return s.repo.AddMember(ctx, chatID, userID)
}

func (s *ChatService) RemoveMember(ctx context.Context, chatID int64, targetUserID, requesterID string) error {
	member, err := s.repo.GetMember(ctx, chatID, targetUserID)
	if err != nil {
		return err
	}
	if member == nil {
		return ErrNotFound
	}

	requester, err := s.repo.GetMember(ctx, chatID, requesterID)
	if err != nil {
		return err
	}
	if requester == nil {
		return ErrNotMember
	}

	_, err = s.repo.RemoveMember(ctx, chatID, targetUserID)
	return err
}

func (s *ChatService) ListMembers(ctx context.Context, chatID int64) ([]*dto.ChatMember, error) {
	return s.repo.GetMembers(ctx, chatID)
}
