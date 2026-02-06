package config

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
)

// The firebase.go file contains the initialization and configuration logic for Firebase Firestore, it handles database access.

// Any services or repositories dependent on Firebase Firestore are initialized after Firestore itself.

// .json and .env files must be ignored by Git to keep sensitive information secure. This includes any credentials or API keys required for accessing Firebase services.
// JWT_SECRET_KEY and GOOGLE_APPLICATION_CREDENTIALS inside the .env

// InitFirestoreClient initializes and returns a Firebase Firestore client.
func InitFirestoreClient(ctx context.Context) (*firestore.Client, error) {
	// We pass nil for options because the SDK will automatically
	// look for the GOOGLE_APPLICATION_CREDENTIALS env var.
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error initializing Firebase app: %v", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting Firestore client: %v", err)
	}

	log.Println("Firebase Firestore client initialized successfully!")
	return client, nil
}
