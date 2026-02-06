package main

import (
	"context"
	"log"
	"os"

	// Import structure:
	// main -> routes -> controllers   -> models
	// 					 repositories  -> models
	// 			         services      -> models

	"DataTracker/app/controllers"
	// "DataTracker/app/middleware"
	"DataTracker/app/repositories"
	"DataTracker/app/routes"
	"DataTracker/app/services"
	"DataTracker/app/utils"
	"DataTracker/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Always run go mod tidy if you've made some changes to the structure of the code (routes, new files, new packages, etc.)

// Order of creation: firebase.go, main.go, bloom_filter.go, auth.go, user_models.go, user_repositories.go, user_services.go, user_controllers.go, routes.go and main.go again

// Flow of a request: User Request → Routes(routes.go) → Controller(user_controllers.go) → Service(user_services.go) → Repository(user_repositories.go) → Database (firebase.go)

// This project tries to follow Go's Clean Architecture: (N-Tier)
// Models: Define the Data.
// Repositories: Define the Interface for storage and the Implementation (Firestore).
// Services: Define the Logic and use the Repository interface.
// Controllers: Define the API entry points and use the Service.

func main() {
	ctx := context.Background()

	// Retrieve the key from .env that is located inside the config folder
	// This is important for accessing sensitive configuration settings like JWT secret key and Firestore credentials.
	// -- configuration layer --
	err := godotenv.Load("config/.env")
	if err != nil {
		log.Fatal("Error loading .env file from config/")
	}

	secretKey := os.Getenv("JWT_SECRET_KEY")
	if secretKey == "" {
		log.Fatalf("Error: JWT_SECRET_KEY is not set in environment variables")
	}

	// Initialize the Firestore client
	// This establishes a connection to Google Cloud Firestore, which is used for storing and retrieving data.
	// Firestore client is initialized before any services or repositories that depend on it.
	// -- database initialization --
	firestoreClient, err := config.InitFirestoreClient(ctx)
	if err != nil {
		log.Fatalf("Error initializing Firestore client: %v", err)
	}
	defer firestoreClient.Close()

	// Initialize the repository
	// -- repository layer --
	userRepo := repositories.NewUserRepository(firestoreClient)

	// Initialize the bloom filter
	// Populate the Bloom Filter from the database during server startup to ensure it is up-to-date and ready for use.
	// We use two distinct filters to avoid collision between usernames and emails.
	// With a low count of bits and hashes, the "False Positive" rate can climb significantly
	// -- utils initialization --
	usernameBFilter := utils.NewConcurrentBloomFilter(10000, 7) // Increased size for better accuracy
	emailBFilter := utils.NewConcurrentBloomFilter(10000, 7)

	// --- Warm-up Phase ---

	// Populate Usernames
	existingNames, _ := userRepo.GetAllUsernames(ctx)
	for _, name := range existingNames {
		usernameBFilter.Insert([]byte(name))
	}

	// Populate Emails
	existingEmails, _ := userRepo.GetAllEmails(ctx)
	for _, email := range existingEmails {
		emailBFilter.Insert([]byte(email))
	}
	log.Printf("Bloom Filters ready! Loaded %d names and %d emails.", len(existingNames), len(existingEmails))

	// Initialize the Service
	// --- business layer ---
	userService := services.NewUserService(userRepo, usernameBFilter, emailBFilter)

	// Initialize UserController
	// This creates an instance of the UserController, which handles user-related logic.
	// -- API handler --
	userController := controllers.NewUserController(userService)

	// Set up Gin router
	r := gin.Default()

	// Enable CORS for a React app
	// YOu can edit the port, currently is running on localhost:5173
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// Use auth middleware for specific routes that need validation
	// This middleware ensures only authenticated users can access certain routes by checking for valid tokens in the Authorization header
	// r.Use(middleware.AuthMiddleware())
	// We are handling the middleware use in the routes file

	// Initialize routes, passing the userController
	// This sets up all API endpoints related to users and their operations.
	routes.InitializeUserRoutes(r, userController)

	log.Println("Server working on port 8081")
	if err := r.Run(":8081"); err != nil {
		log.Fatal(err)
	}
	// Gin needs to know which proxy server is sending the request so it can correctly identify the user's real IP address. By default, it trusts everything, which could allow an attacker to spoof their IP.
}

// important before testing:
/*
To connect this Go backend to Firebase/Google Cloud, two manual steps are required in the Google Cloud Console to prepare the environment:

https://console.cloud.google.com/apis/api/firestore.googleapis.com/
https://console.cloud.google.com/apis/api/firestore.googleapis.com/metrics?project=general-backend-testing
This enables the communication bridge between external applications (your Go server) and Google's database services.
Enabling this API grants your GOOGLE_APPLICATION_CREDENTIALS the right to "talk" to Firestore via RPC
(For enabling postman use)

https://console.cloud.google.com/datastore/setup
https://console.cloud.google.com/datastore/create-database?project=general-backend-testing
This initializes the physical storage location and configuration for your data.
Enabling the API (Step 1) gives you the ability to use a database, but Step 2 actually creates it.
This step defines the Database ID (usually (default)), the data consistency mode (Native Mode), and the geographic region where the data resides.
(For activating the database)
*/

// Routes to test in postman:
/*
POST: http://localhost:8081/api/users/
body (raw json):
{
    "name": "Julian",
    "email": "Julian@Test.com",
    "username":"JJ",
    "password":"1234"
}
*/

// During the initial integration phase, the AuthMiddleware was disabled in main.go to allow for "Bootstrapping" the database.
// Once the first admin user is created and a JWT strategy is finalized, uncomment the line disabling the AuthMiddleware.
// After enabling, all requests must include a Bearer <JWT> in the Authorization header.

/*
your middleware just says "Yes" or "No." A professional step is to have the middleware extract the user_id from the token and "inject" it into the Gin context.

// Inside Middleware
c.Set("currentUserEmail", claims["email"])
c.Next()

// Inside your GetUser Controller
email, _ := c.Get("currentUserEmail") // Now the controller knows WHO is asking!

// For future project: Firebase Auth
*/

/*
Dual-Layer Validation strategy:
Probabilistic Layer: A Concurrent Bloom Filter provides O(1) checks for username/email availability.

Persistence Layer: A Firestore Repository handles deep-verification and ACID-compliant writes using the official Google Cloud SDK."
*/
