package services

import (
	"context"
	"DataTracker/app/repositories"
)

type AnalyticsService struct {
	repo *repositories.AnalyticsRepository
}

func NewAnalyticsService(repo *repositories.AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

func (s *AnalyticsService) TrackAction(ctx context.Context, username string, action string) error {
	return s.repo.IncrementEngagement(ctx, username, action)
}

func (s *AnalyticsService) GetAllStats(ctx context.Context) ([]map[string]interface{}, error) {
	return s.repo.GetAllAnalytics(ctx)
}
