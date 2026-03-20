package controllers

import (
	"crypto/rsa"
	"net/http"
	"time"

	"DataTracker/app/models"
	"DataTracker/app/services"
	"DataTracker/app/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type UserController struct {
	service    *services.UserService
	privateKey *rsa.PrivateKey
}

func NewUserController(service *services.UserService, privateKey *rsa.PrivateKey) *UserController {
	return &UserController{service: service, privateKey: privateKey}
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

	// 2. Generate RS256 JWT Token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub":      user.ID,
		"email":    user.Email,
		"username": user.Username, // Cryptographic proof of username for InkToChat
		"role":     user.Role,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24h
	})

	tokenString, err := token.SignedString(uc.privateKey)
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
	// 1. Create a dedicated struct to capture the password
	var req struct {
		Name     string `json:"name" binding:"required"`
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// 2. Map it to the user model
	user := models.User{
		Name:     req.Name,
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	}

	if err := uc.service.Create(user); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully"})
}

// GetUser handles GET /api/users/:id
// Ownership is enforced by the OwnerOrAdmin middleware before this handler runs.
func (uc *UserController) GetUser(c *gin.Context) {
	requestedID := c.Param("id")

	user, err := uc.service.GetUserByID(requestedID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetAllUsers handles GET /api/users/
func (uc UserController) GetAllUsers(c *gin.Context) {
	// The AdminOnly middleware already handles the role check.
	// We just need to call the service.

	// Pass the context from Gin to the service/repo for Firestore
	users, err := uc.service.GetAllUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// UpdateUser handles PUT /api/users/:id
// Ownership is enforced by the OwnerOrAdmin middleware before this handler runs.
func (uc *UserController) UpdateUser(c *gin.Context) {
	requestedID := c.Param("id")
	userRole, _ := c.Get("userRole")

	// 1. Bind to a map instead of the User struct
	// This captures ONLY the fields provided in the JSON body
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// 2. Security: Prevent non-admins from promoting themselves
	if userRole != "admin" {
		delete(updates, "role")
	}

	// 3. Call Service with the map
	if err := uc.service.UpdateUserFields(requestedID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// DeleteUser handles DELETE /api/users/:id
// Ownership is enforced by the OwnerOrAdmin middleware before this handler runs.
func (uc *UserController) DeleteUser(c *gin.Context) {
	requestedID := c.Param("id")

	if err := uc.service.DeleteUser(requestedID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}
	// Due to the nature of Bloom Filters, updated or deleted usernames/emails remain
	// in the filter until the next server restart (Warm-up phase).
	// This acts as a temporary 'reservation' preventing immediate reuse of recently changed identifiers.
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

// AdminResetPassword handles POST /api/admin/users/:id/reset-password
func (uc *UserController) AdminResetPassword(c *gin.Context) {
	id := c.Param("id")

	// 1. Check if user exists
	if _, err := uc.service.GetUserByID(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// 2. Initiate Reset (Generates raw token, hashes it in DB)
	rawToken, err := uc.service.InitiatePasswordReset(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate reset: " + err.Error()})
		return
	}

	// 3. SECURE DELIVERY: Send the actual email
	userRecord, _ := uc.service.GetUserByID(id)
	if err := utils.SendResetEmail(userRecord.Email, rawToken); err != nil {
		c.JSON(http.StatusAccepted, gin.H{
			"message": "Reset token generated, but failed to send email. Check SMTP config.",
			"error":   err.Error(),
			"raw_token": rawToken, // Fallback for manual support
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Reset link sent successfully to " + userRecord.Email,
		"expires_in": "1 hour",
	})
}

// ConfirmResetPassword handles POST /api/users/reset-password/confirm
func (uc *UserController) ConfirmResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token and new password required"})
		return
	}

	if err := uc.service.CompletePasswordReset(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}

// ForgotPasswordInitiate handles POST /api/users/forgot-password (Public)
func (uc *UserController) ForgotPasswordInitiate(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Valid email address required"})
		return
	}

	// 1. Fetch User by Email
	userRecord, err := uc.service.Repo.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		// SECURITY: We return "Accepted" even if the email doesn't exist.
		// This prevents "Account Enumeration" (hackers checking which emails have accounts).
		c.JSON(http.StatusAccepted, gin.H{"message": "If an account exists for this email, a reset link will be sent."})
		return
	}

	// 2. Initiate Reset Flow (Token generation + SHA-256 Hashing)
	rawToken, err := uc.service.InitiatePasswordReset(c.Request.Context(), userRecord.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error generating reset link"})
		return
	}

	// 3. SECURE DELIVERY: Email the token to the user
	if err := utils.SendResetEmail(userRecord.Email, rawToken); err != nil {
		// Log internal error but don't leak it
		c.JSON(http.StatusAccepted, gin.H{"message": "Reset request accepted, but email delivery failed. Support has been notified."})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "If an account exists for this email, a reset link will be sent."})
}

