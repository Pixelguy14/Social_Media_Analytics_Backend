package repositories

import (
	"context"
	"google.golang.org/api/iterator"

	"cloud.google.com/go/firestore"
)

type AnalyticsRepository struct {
	client *firestore.Client
}

func NewAnalyticsRepository(client *firestore.Client) *AnalyticsRepository {
	return &AnalyticsRepository{client: client}
}

func (r *AnalyticsRepository) IncrementEngagement(ctx context.Context, username string, action string) error {
	docRef := r.client.Collection("analytics").Doc(username)
	
	_, err := docRef.Set(ctx, map[string]interface{}{
		"username": username,
		action:     firestore.Increment(1),
		"total_actions": firestore.Increment(1),
	}, firestore.MergeAll)
	
	return err
}

func (r *AnalyticsRepository) ResetAll(ctx context.Context) error {
	iter := r.client.Collection("analytics").Documents(ctx)
	defer iter.Stop()

	batch := r.client.Batch()
	for {
		doc, err := iter.Next()
		if err == iterator.Done { break }
		if err != nil { return err }
		batch.Delete(doc.Ref)
	}
	_, err := batch.Commit(ctx)
	return err
}

func (r *AnalyticsRepository) GetAllAnalytics(ctx context.Context) ([]map[string]interface{}, error) {
	iter := r.client.Collection("analytics").Documents(ctx)
	defer iter.Stop()

	var results []map[string]interface{}
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		results = append(results, doc.Data())
	}
	return results, nil
}

func (r *AnalyticsRepository) ClearChat(ctx context.Context, roomID string) error {
	iter := r.client.Collection("rooms").Doc(roomID).Collection("messages").Documents(ctx)
	defer iter.Stop()

	batch := r.client.Batch()
	count := 0
	for {
		doc, err := iter.Next()
		if err == iterator.Done { break }
		if err != nil { return err }
		batch.Delete(doc.Ref)
		count++
		if count >= 400 { // Firestore batch limit
			_, _ = batch.Commit(ctx)
			batch = r.client.Batch()
			count = 0
		}
	}
	if count > 0 {
		_, err := batch.Commit(ctx)
		return err
	}
	return nil
}
