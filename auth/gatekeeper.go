package auth

import (
	"context"
	"fmt"

	"firebase.google.com/go/v4/auth"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"github.com/bits-and-blooms/bloom/v3"
)

// BloomFilter maintains reserved usernames
var bloom_filter = bloom.NewWithEstimates(1000000, 0.01)

func reserve_username(username string) bool {
	if bloom_filter.TestString(username) {
		return false
	}
	bloom_filter.AddString(username)
	return true
}

// Generate_custom_token orchestrates the generation of a token and reserves the name.
// It checks if the username exists as a permanent DataTracker account first.
func Generate_custom_token(ctx context.Context, authClient *auth.Client, firestoreClient *firestore.Client, username string) (string, error) {
	// 1. Check if user is a permanent DataTracker user
	iter := firestoreClient.Collection("users").Where("username", "==", username).Limit(1).Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	isPermanentUser := (err == nil)
	
	// If it's a permanent user, we don't need to check bloom filter for "reservation"
	// but we use their existing UID if they are already logged in or allow them to "re-claim" it.
	if !isPermanentUser {
		if err != iterator.Done && err != nil {
			return "", fmt.Errorf("firestore error: %v", err)
		}

		// 2. If NOT an existing user, check Bloom Filter for anonymous reservation
		if !reserve_username(username) {
			return "", fmt.Errorf("username already taken or reserved")
		}
	}

	// 3. Exchange for token. Use permanent ID if found, otherwise the username-based UID.
	uid := "uid_" + username
	if isPermanentUser {
		uid = doc.Ref.ID
	}

	return exchange_custom_token(ctx, authClient, uid, username)
}

func ResetGatekeeper() {
	bloom_filter.ClearAll()
}

func exchange_custom_token(ctx context.Context, client *auth.Client, uid string, username string) (string, error) {
	claims := map[string]interface{}{
		"username": username,
	}

	token, err := client.CustomTokenWithClaims(ctx, uid, claims)
	if err != nil {
		return "", fmt.Errorf("error minting custom token: %v", err)
	}

	return token, nil
}
