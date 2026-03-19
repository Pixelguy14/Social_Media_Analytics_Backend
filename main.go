package main

import (
	"context"
	"log"
	"os"
	"time"

	"DataTracker/app/controllers"
	"DataTracker/app/repositories"
	"DataTracker/app/routes"
	"DataTracker/app/services"
	"DataTracker/app/utils"
	"DataTracker/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	firebase "firebase.google.com/go/v4"
	"DataTracker/i18n"
	"DataTracker/ratelimit"
	"DataTracker/worker"
)

func main() {
	ctx := context.Background()

	// 1. Initial Configuration
	err := godotenv.Load("config/.env")
	if err != nil {
		log.Fatal("Error loading .env file from config/")
	}

	secretKey := os.Getenv("JWT_SECRET_KEY")
	if secretKey == "" {
		log.Fatalf("Error: JWT_SECRET_KEY is not set in environment variables")
	}

	// 2. Database & Infrastructure Initialization
	firestoreClient, err := config.InitFirestoreClient(ctx)
	if err != nil {
		log.Fatalf("Error initializing Firestore client: %v", err)
	}
	defer firestoreClient.Close()

	firebaseApp, err := firebase.NewApp(ctx, nil)
	if err != nil {
		log.Fatalf("Error initializing firebase app: %v", err)
	}
	authClient, err := firebaseApp.Auth(ctx)
	if err != nil {
		log.Fatalf("Error getting Firebase Auth client: %v", err)
	}

	// 3. Bloom Filters & Background Tickers
	usernameBFilter := utils.NewConcurrentBloomFilter(10000, 7)
	emailBFilter := utils.NewConcurrentBloomFilter(10000, 7)

	// Repositories (Persistence Layer)
	userRepo := repositories.NewUserRepository(firestoreClient)
	chatRepo := repositories.NewChatRepository(firestoreClient)
	analyticsRepo := repositories.NewAnalyticsRepository(firestoreClient)

	// Warm-up & Logic for Filters
	refresh_filters(ctx, userRepo, usernameBFilter, emailBFilter)
	go func() {
		for range ticker_factory() {
			log.Println("Background: Rebuilding Bloom Filters...")
			refresh_filters(ctx, userRepo, usernameBFilter, emailBFilter)
		}
	}()

	// 4. Services (Business Layer)
	userService := services.NewUserService(userRepo, usernameBFilter, emailBFilter)
	chatService := services.NewChatService(chatRepo, analyticsRepo, userRepo)
	inkAuthService := services.NewInkAuthService(authClient, firestoreClient)
	analyticsService := services.NewAnalyticsService(analyticsRepo)

	// Utilities
	rateLimiter := ratelimit.NewManager(50, 0.83)
	if err := i18n.Init_localization(); err != nil {
		log.Printf("i18n init error: %v", err)
	}
	worker.Start_all_workers(ctx, firestoreClient)

	// 5. Controllers (Admission Layer)
	userController := controllers.NewUserController(userService)
	inkController := controllers.NewInkController(chatService, inkAuthService, analyticsService, rateLimiter)

	// 6. Router & Web Layer
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://192.168.168.104:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// Initializing Routes
	routes.InitializeUserRoutes(r, userController)
	routes.InitializeInkRoutes(r, inkController)

	log.Println("Server working on port 8081")
	if err := r.Run(":8081"); err != nil {
		log.Fatal(err)
	}
}

// Support Functions
func refresh_filters(ctx context.Context, repo repositories.UserRepository, unf *utils.ConcurrentBloomFilter, emf *utils.ConcurrentBloomFilter) {
	unf.Clear()
	emf.Clear()
	existingNames, _ := repo.GetAllUsernames(ctx)
	for _, name := range existingNames {
		unf.Insert([]byte(name))
	}
	existingEmails, _ := repo.GetAllEmails(ctx)
	for _, email := range existingEmails {
		emf.Insert([]byte(email))
	}
	log.Printf("Bloom Filters refreshed! Loaded %d names and %d emails.", len(existingNames), len(existingEmails))
}

func ticker_factory() <-chan struct{} {
	c := make(chan struct{})
	go func() {
		for {
			time.Sleep(24 * time.Hour)
			c <- struct{}{}
		}
	}()
	return c
}
