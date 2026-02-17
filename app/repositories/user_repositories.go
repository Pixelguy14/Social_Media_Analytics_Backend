package repositories

import (
	"DataTracker/app/models"
	"context"
	"errors"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type UserRepository interface {
	Create(ctx context.Context, user models.User) error
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetAllUsernames(ctx context.Context) ([]string, error) // for bloom filtering
	GetAllEmails(ctx context.Context) ([]string, error)    // for bloom filtering
	GetAllUsers(ctx context.Context) ([]models.User, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	ExistsByEmail(ctx context.Context, username string) (bool, error)
	Update(ctx context.Context, id string, user models.User) error
	UpdateFields(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
}

// It’s better to pass the context from the Controller → Service → Repository so that if a user cancels their request, the database query stops immediately.

type FirestoreUserRepository struct {
	client *firestore.Client
}

func NewUserRepository(client *firestore.Client) UserRepository {
	return &FirestoreUserRepository{client: client}
}

// Create creates a new user in the Firestore database.
func (r *FirestoreUserRepository) Create(ctx context.Context, user models.User) error {
	// We use "_" to ignore the DocumentRef and WriteResult since we only care about the error
	_, _, err := r.client.Collection("users").Add(ctx, user)
	return err
}

// GetByID retrieves a user by their ID from the Firestore database.
func (r *FirestoreUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	doc, err := r.client.Collection("users").Doc(id).Get(ctx)
	if err != nil {
		return nil, err
	}
	var user models.User
	err = doc.DataTo(&user)
	if err != nil {
		return nil, err
	}
	user.ID = doc.Ref.ID // Ensure the ID from Firestore is set
	return &user, nil
}

// GetAllUsernames returns a list of all usernames in the Firestore database.
func (r *FirestoreUserRepository) GetAllUsernames(ctx context.Context) ([]string, error) {
	var usernames []string
	// Optimization: Only fetch the "username" field
	iter := r.client.Collection("users").Select("username").Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		// Extract field directly to avoid unmarshaling the whole struct
		if name, ok := doc.Data()["username"].(string); ok {
			usernames = append(usernames, name)
		}
	}
	return usernames, nil
}

// GetAllEmails returns a list of all emails in the Firestore database.
func (r *FirestoreUserRepository) GetAllEmails(ctx context.Context) ([]string, error) {
	var usernames []string
	// Optimization: Only fetch the "username" field
	iter := r.client.Collection("users").Select("email").Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		// Extract field directly to avoid unmarshaling the whole struct
		if name, ok := doc.Data()["email"].(string); ok {
			usernames = append(usernames, name)
		}
	}
	return usernames, nil
}

// Contains checks if a username exists in the Firestore database.
func (r *FirestoreUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	iter := r.client.Collection("users").Where("username", "==", username).Limit(1).Documents(ctx)
	defer iter.Stop()

	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil // Definitely does not exist
	}
	if err != nil {
		return false, err // An actual database error happened
	}
	return true, nil // Found it!
}

// Contains checks if an email exists in the Firestore database.
func (r *FirestoreUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	iter := r.client.Collection("users").Where("email", "==", email).Limit(1).Documents(ctx)
	defer iter.Stop()

	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil // Definitely does not exist
	}
	if err != nil {
		return false, err // An actual database error happened
	}
	return true, nil // Found it!
}

// GetByUsername retrieves a user by their username from the Firestore database.
func (r *FirestoreUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	iter := r.client.Collection("users").Where("username", "==", username).Limit(1).Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}

	var user models.User
	if err := doc.DataTo(&user); err != nil {
		return nil, err
	}
	user.ID = doc.Ref.ID
	return &user, nil
}

// GetByEmail retrieves a user by their email from the Firestore database.
func (r *FirestoreUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	// Queries look for documents where the "email" field matches
	iter := r.client.Collection("users").Where("email", "==", email).Limit(1).Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}

	var user models.User
	if err := doc.DataTo(&user); err != nil {
		return nil, err
	}
	user.ID = doc.Ref.ID // Don't forget to map the Document ID!
	return &user, nil
}

// GetAll retrieves all users and their data from the Firestore database.
func (r *FirestoreUserRepository) GetAllUsers(ctx context.Context) ([]models.User, error) {
	var users []models.User
	iter := r.client.Collection("users").Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var user models.User
		if err := doc.DataTo(&user); err != nil {
			// log.Printf("Warning: Failed to parse user %s: %v", doc.Ref.ID, err)
			continue
		}

		// MANUALLY SET THE ID HERE
		user.ID = doc.Ref.ID

		users = append(users, user)
	}
	return users, nil
}

// Update updates an existing user in the Firestore database.
func (r *FirestoreUserRepository) Update(ctx context.Context, id string, user models.User) error {
	ref := r.client.Collection("users").Doc(id)
	_, err := ref.Set(ctx, user)
	return err
}

func (r *FirestoreUserRepository) UpdateFields(ctx context.Context, id string, updates map[string]interface{}) error {
	// 1. Convert the map into a slice of firestore.Update structs
	var fsUpdates []firestore.Update

	for field, value := range updates {
		fsUpdates = append(fsUpdates, firestore.Update{
			Path:  field,
			Value: value,
		})
	}

	// 2. Pass the slice using the '...' operator to expand it into variadic arguments
	_, err := r.client.Collection("users").Doc(id).Update(ctx, fsUpdates)
	return err
}

// Delete deletes a user from the Firestore database.
func (r *FirestoreUserRepository) Delete(ctx context.Context, id string) error {
	docRef := r.client.Collection("users").Doc(id)
	_, err := docRef.Delete(ctx)
	if err != nil {
		return err
	}
	return nil
}
