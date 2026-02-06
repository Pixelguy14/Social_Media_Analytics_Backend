package controllers

import (
	"net/http"
	"os"
	"time"

	"DataTracker/app/models"
	"DataTracker/app/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type UserController struct {
	service *services.UserService
}

func NewUserController(service *services.UserService) *UserController {
	return &UserController{service: service}
}

// Login handles POST /api/users/login
func (uc *UserController) Login(c *gin.Context) {
	var loginReq struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email and password required"})
		return
	}

	// 1. Verify credentials via Service
	user, err := uc.service.Login(loginReq.Email, loginReq.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// 2. Generate JWT Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"role":  user.Role,
		"exp":   time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24h
	})

	secretKey := os.Getenv("JWT_SECRET_KEY")
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	// 3. Return user (Password is automatically hidden by json:"-" tag!) and token
	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"user":  user,
	})
}

// CreateUser handles POST /api/users/
func (uc *UserController) CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	if err := uc.service.Create(user); err != nil {
		// Return 409 Conflict if user already exists
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully"})
}

// GetUser handles GET /api/users/:id
func (uc *UserController) GetUser(c *gin.Context) {
	// SECURITY: Get ID from the middleware context (set in auth.go)
	loggedInID, _ := c.Get("userID")
	requestedID := c.Param("id")

	// Optional: Only allow users to see their own data
	if loggedInID != requestedID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	user, err := uc.service.GetUserByID(requestedID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser handles PUT /api/users/:id
func (uc *UserController) UpdateUser(c *gin.Context) {
	requestedID := c.Param("id")
	loggedInID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole")

	// 1. Security Check
	if loggedInID != requestedID && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to update this user"})
		return
	}

	// 2. Bind to a map instead of the User struct
	// This captures ONLY the fields provided in the JSON body
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// 3. Security: Prevent non-admins from promoting themselves
	if userRole != "admin" {
		delete(updates, "role")
	}

	// 4. Call Service with the map
	if err := uc.service.UpdateUserFields(requestedID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// DeleteUser handles DELETE /api/users/:id
func (uc *UserController) DeleteUser(c *gin.Context) {
	loggedInID, _ := c.Get("userID")
	userRole, _ := c.Get("userRole") // Get role from middleware
	requestedID := c.Param("id")

	// Logic: Allow if the user is the owner OR if the user is an admin
	if loggedInID != requestedID && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to delete this user"})
		return
	}

	if err := uc.service.DeleteUser(requestedID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}
	// Due to the nature of Bloom Filters, updated or deleted usernames/emails remain
	// in the filter until the next server restart (Warm-up phase).
	// This acts as a temporary 'reservation' preventing immediate reuse of recently changed identifiers.
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}
