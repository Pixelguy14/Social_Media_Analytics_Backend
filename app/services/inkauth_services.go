package services

import (
	"context"
	gatekeeper "DataTracker/auth"
	fireauth "firebase.google.com/go/v4/auth"
	"cloud.google.com/go/firestore"
)

type InkAuthService struct {
	authClient *fireauth.Client
	fireClient *firestore.Client
}

func NewInkAuthService(ac *fireauth.Client, fc *firestore.Client) *InkAuthService {
	return &InkAuthService{authClient: ac, fireClient: fc}
}

func (s *InkAuthService) GetToken(ctx context.Context, username string) (string, error) {
	return gatekeeper.Generate_custom_token(ctx, s.authClient, s.fireClient, username)
}

func (s *InkAuthService) ResetIdentityFilter() {
	gatekeeper.ResetGatekeeper()
}
