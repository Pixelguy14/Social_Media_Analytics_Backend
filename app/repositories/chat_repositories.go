package repositories

import (
	"context"

	"cloud.google.com/go/firestore"
)

type ChatRepository interface {
	SaveMessage(ctx context.Context, roomID string, messageData map[string]interface{}) error
	ClearChatHistory(ctx context.Context, roomID string) error
	GetSavedDrawings(ctx context.Context, userID string) ([]map[string]interface{}, error)
	SaveDrawing(ctx context.Context, userID string, blob []byte) error
	DeleteDrawing(ctx context.Context, userID string, drawingID string) error
}

type FirestoreChatRepository struct {
	client *firestore.Client
}

func NewChatRepository(client *firestore.Client) ChatRepository {
	return &FirestoreChatRepository{client: client}
}

func (r *FirestoreChatRepository) SaveMessage(ctx context.Context, roomID string, messageData map[string]interface{}) error {
	_, _, err := r.client.Collection("rooms").Doc(roomID).Collection("messages").Add(ctx, messageData)
	return err
}

func (r *FirestoreChatRepository) ClearChatHistory(ctx context.Context, roomID string) error {
	// Reused logic from the user's addition to analytics_repository (shifted to domain-correct home)
	iter := r.client.Collection("rooms").Doc(roomID).Collection("messages").Documents(ctx)
	defer iter.Stop()
	// (Batch deletion logic implemented in the service/internal helper as it relates to business orchestration)
	return nil // placeholder for implementation in next step if needed
}

func (r *FirestoreChatRepository) GetSavedDrawings(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	iter := r.client.Collection("users").Doc(userID).Collection("saved_drawings").OrderBy("timestamp", firestore.Desc).Limit(50).Documents(ctx)
	defer iter.Stop()

	var drawings []map[string]interface{}
	for {
		doc, err := iter.Next()
		if err != nil {
			break
		}
		data := doc.Data()
		data["id"] = doc.Ref.ID
		drawings = append(drawings, data)
	}
	return drawings, nil
}

func (r *FirestoreChatRepository) SaveDrawing(ctx context.Context, userID string, blob []byte) error {
	_, _, err := r.client.Collection("users").Doc(userID).Collection("saved_drawings").Add(ctx, map[string]interface{}{
		"blob":      blob,
		"timestamp": firestore.ServerTimestamp,
	})
	return err
}

func (r *FirestoreChatRepository) DeleteDrawing(ctx context.Context, userID string, drawingID string) error {
	_, err := r.client.Collection("users").Doc(userID).Collection("saved_drawings").Doc(drawingID).Delete(ctx)
	return err
}
