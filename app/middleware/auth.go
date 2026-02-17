package middleware

// The Middleware decrypts the token and Sets the identity into the gin.Context.

import (
	"log" // log is used for the debug console print texts
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware is a function that wraps an Gin HTTP handler to enforce authentication.
// It checks if the request contains a valid JWT token in the Authorization header and verifies its validity.

// Authentication ensures that only authorized users or systems can access certain resources or APIs.
// Without authentication, anyone can make requests to your server, potentially leading to unauthorized data access, modifications, or disruptions.

// Integrating an authentication middleware into Go is a critical step in building secure, robust, and scalable web applications.
// It helps protect user data, maintain access control, ensure compliance with regulations, and provide a better overall user experience.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Println("Debug: No Authorization header found")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token not given"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		secretKey := os.Getenv("JWT_SECRET_KEY")

		// Parse the JWT token using the provided secret key.
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, http.ErrAbortHandler
			}
			return []byte(secretKey), nil
		})

		// Check if the parsing or verification failed.
		if err != nil || !token.Valid {
			log.Printf("Debug: token validation failed: %v\n", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Token"})
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// Inject the data into the Gin Context
			// You can now access "userID" in any controller downstream
			c.Set("userRole", claims["role"])
			c.Set("userID", claims["sub"])
			c.Set("userEmail", claims["email"])
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		// If the token is valid, proceed to the next handler in main
		log.Println("Debug: Token is valid for user:", c.GetString("userEmail"))
		// Upon successful JWT verification, the AuthMiddleware extracts the sub (Subject) and email claims,
		// injecting them into the Gin Context. This ensures that downstream handlers can perform
		// Authorization checks (Owner-Only access) without re-parsing the authorization header.
		c.Next()
	}
}

// AdminOnly assumes AuthMiddleware has already run and set "userRole" in the context
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("userRole")

		// Defensive check: Convert interface to string safely
		roleStr, ok := role.(string)

		if !exists || !ok || roleStr != "admin" {
			log.Printf("Access Denied: User has role %v, expected 'admin'", role)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Access denied: Admin privileges required",
			})
			return
		}

		// If it is an admin, continue to the next handler
		c.Next()
	}
}
