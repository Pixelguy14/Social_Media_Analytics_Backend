package worker

import (
	"context"
	"log/slog"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

func start_circular_buffer(ctx context.Context, client *firestore.Client, room_id string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			enforce_buffer_limit(ctx, client, room_id)
		}
	}
}

func enforce_buffer_limit(ctx context.Context, client *firestore.Client, room_id string) {
	coll := client.Collection("rooms").Doc(room_id).Collection("messages")
	
	iter := coll.OrderBy("timestamp", firestore.Desc).Offset(100).Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			slog.Error("failed to iterate old messages", "error", err)
			continue
		}

		_, err = doc.Ref.Delete(ctx)
		if err != nil {
			slog.Error("failed to delete old message", "doc_id", doc.Ref.ID, "error", err)
		} else {
			slog.Info("deleted old message", "room", room_id, "doc_id", doc.Ref.ID)
		}
	}
}

// Start_all_workers starts the workers for all lobbies
func Start_all_workers(ctx context.Context, client *firestore.Client) {
	rooms := []string{"A", "B", "C", "D"}
	for _, room := range rooms {
		go start_circular_buffer(ctx, client, room)
	}
}
