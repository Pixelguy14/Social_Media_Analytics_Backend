package middleware

// The Middleware decrypts the token and Sets the identity into the gin.Context.

import (
	"crypto/rsa"
	"log"
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
// AuthMiddleware validates the RS256 JWT token using the provided public key.
// The public key is bound at registration time, ensuring verify-fail-fast at startup.
func AuthMiddleware(publicKey *rsa.PublicKey) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token not given"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// Parse the JWT token using the RSA public key.
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			// Verify that the signing method is RSA (RS256)
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, http.ErrAbortHandler
			}
			return publicKey, nil
		})

		// Check if the parsing or verification failed.
		if err != nil || !token.Valid {
			// Structured security log for failed auth
			log.Printf("Security Alarm: Auth Failure from IP %s: %v", c.ClientIP(), err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Token"})
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// Inject the data into the Gin Context
			c.Set("userRole", claims["role"])
			c.Set("userID", claims["sub"])
			c.Set("userEmail", claims["email"])
			// Used to identify the sender without trusting request body strings
			c.Set("username", claims["username"])
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		// Only log PII in non-production environments
		if os.Getenv("GIN_MODE") != "release" {
			log.Println("Debug: Token is valid for user:", c.GetString("userEmail"))
			// Upon successful JWT verification, the AuthMiddleware extracts the sub (Subject) and email claims,
			// injecting them into the Gin Context. This ensures that downstream handlers can perform
			// Authorization checks (Owner-Only access) without re-parsing the authorization header.
		}
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
			log.Printf("Security Alarm: Forbidden Access Attempt at %s from IP %s. User role: %v", c.Request.URL.Path, c.ClientIP(), role)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Access denied: Admin privileges required",
			})
			return
		}

		// If it is an admin, continue to the next handler
		c.Next()
	}
}

// OwnerOrAdmin assumes AuthMiddleware has already run and set "userID" and "userRole" in the context.
// It verifies that the authenticated user's ID matches the URL parameter specified by paramName,
// OR that the user has the "admin" role. If neither condition is true, it returns 403 Forbidden.
// This eliminates the need for individual controllers to perform their own ownership checks.
func OwnerOrAdmin(paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		loggedInID, _ := c.Get("userID")
		userRole, _ := c.Get("userRole")
		requestedID := c.Param(paramName)

		roleStr, _ := userRole.(string)
		isOwner := (loggedInID == requestedID)
		isAdmin := (roleStr == "admin")

		if !isOwner && !isAdmin {
			log.Printf("Security Alarm: IDOR Attempt at %s from IP %s by UserID %s to ResourceID %s", c.Request.URL.Path, c.ClientIP(), loggedInID, requestedID)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Access denied: you can only access your own resources",
			})
			return
		}

		c.Next()
	}
}

