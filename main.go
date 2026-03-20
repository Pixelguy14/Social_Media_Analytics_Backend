package main

import (
	"context"
	"log"
	"os"
	"time"
	"strings"

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
	"github.com/golang-jwt/jwt/v5"
)

func main() {
	ctx := context.Background()

	// 1. Initial Configuration
	err := godotenv.Load("config/.env")
	if err != nil {
		log.Fatal("Error loading .env file from config/")
	}

	// 1.5 Load RSA Keys for RS256 JWT
	privateKeyBytes, err := os.ReadFile("config/private.pem")
	if err != nil {
		log.Fatalf("Fatal: Failed to read private RSA key from config/private.pem: %v", err)
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBytes)
	if err != nil {
		log.Fatalf("Fatal: Failed to parse private RSA key: %v", err)
	}

	publicKeyBytes, err := os.ReadFile("config/public.pem")
	if err != nil {
		log.Fatalf("Fatal: Failed to read public RSA key from config/public.pem: %v", err)
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyBytes)
	if err != nil {
		log.Fatalf("Fatal: Failed to parse public RSA key: %v", err)
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
	// Sensitive Actions: 3 requests per hour (3/3600 = 0.000833 tokens/sec)
	resetRateLimiter := ratelimit.NewManager(3, 0.000833)

	if err := i18n.Init_localization(); err != nil {
		log.Printf("i18n init error: %v", err)
	}
	worker.Start_all_workers(ctx, firestoreClient)

	// 5. Controllers (Admission Layer)
	userController := controllers.NewUserController(userService, privateKey)
	inkController := controllers.NewInkController(chatService, inkAuthService, analyticsService, rateLimiter)

	// 6. Router & Web Layer (Harden for Production)
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// Silence the Proxy Warning
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Printf("Warning: Could not set trusted proxies: %v", err)
	}

	// 7. Dynamic CORS Configuration
	corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	allowedOrigins := []string{"http://localhost:5173"} // Default
	if corsOrigins != "" {
		allowedOrigins = strings.Split(corsOrigins, ",")
	}
	log.Printf("CORS: Allowing origins: %v", allowedOrigins)

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// Initializing Routes
	routes.InitializeUserRoutes(r, userController, publicKey, resetRateLimiter)
	routes.InitializeInkRoutes(r, inkController, publicKey)

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
