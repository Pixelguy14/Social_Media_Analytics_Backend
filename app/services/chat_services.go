package services

import (
	"context"
	"DataTracker/app/repositories"
	"DataTracker/validation"
	"cloud.google.com/go/firestore"
)

type ChatService struct {
	repo          repositories.ChatRepository
	analyticsRepo *repositories.AnalyticsRepository
	userRepo      repositories.UserRepository
}

func NewChatService(repo repositories.ChatRepository, ar *repositories.AnalyticsRepository, ur repositories.UserRepository) *ChatService {
	return &ChatService{repo: repo, analyticsRepo: ar, userRepo: ur}
}

func (s *ChatService) ProcessMessage(ctx context.Context, room, username, text, color string) error {
	// Administrative Color Logic
	if color == "red" || color == "#ff0000" || color == "#FF0000" {
		user, _ := s.userRepo.GetByUsername(ctx, username)
		if user == nil || user.Role != "admin" {
			color = "black"
		}
	}
	if color == "" {
		color = "black"
	}

	data := map[string]interface{}{
		"username":  username,
		"text":      text,
		"color":     color,
		"timestamp": firestore.ServerTimestamp,
	}

	if err := s.repo.SaveMessage(ctx, room, data); err != nil {
		return err
	}

	// Async Track
	go s.analyticsRepo.IncrementEngagement(context.Background(), username, "messages")
	return nil
}

func (s *ChatService) ProcessDrawing(ctx context.Context, room, username, color string, blob []byte) error {
	if err := validation.ValidateBitArray(blob); err != nil {
		return err
	}

	// Administrative Color Logic
	if color == "red" || color == "#ff0000" || color == "#FF0000" {
		user, _ := s.userRepo.GetByUsername(ctx, username)
		if user == nil || user.Role != "admin" {
			color = "black"
		}
	}
	if color == "" {
		color = "black"
	}

	data := map[string]interface{}{
		"username":  username,
		"blob":      blob,
		"color":     color,
		"timestamp": firestore.ServerTimestamp,
	}

	if err := s.repo.SaveMessage(ctx, room, data); err != nil {
		return err
	}

	// Async Track
	go s.analyticsRepo.IncrementEngagement(context.Background(), username, "drawings")
	return nil
}

func (s *ChatService) GetUserDrawings(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	return s.repo.GetSavedDrawings(ctx, userID)
}

func (s *ChatService) PersonalSaveDrawing(ctx context.Context, userID string, blob []byte) error {
	if err := validation.ValidateBitArray(blob); err != nil {
		return err
	}
	return s.repo.SaveDrawing(ctx, userID, blob)
}

func (s *ChatService) PersonalDeleteDrawing(ctx context.Context, userID, drawingID string) error {
	return s.repo.DeleteDrawing(ctx, userID, drawingID)
}

func (s *ChatService) ClearLobbies(ctx context.Context) error {
	rooms := []string{"A", "B", "C", "D"}
	for _, r := range rooms {
		_ = s.analyticsRepo.ClearChat(ctx, r) // Reusing the user's implementation in analytics repo
	}
	return nil
}
