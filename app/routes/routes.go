package routes

import (
	"crypto/rsa"
	"DataTracker/app/controllers"
	"DataTracker/app/middleware"
	"DataTracker/ratelimit"

	"github.com/gin-gonic/gin"
)

func InitializeUserRoutes(r *gin.Engine, uc *controllers.UserController, publicKey *rsa.PublicKey, resetLimiter *ratelimit.Manager) {
	// 1. PUBLIC
	r.POST("/api/users/login", uc.Login)
	r.POST("/api/users/", uc.CreateUser)

	// Public Reset Flow (Protected by sensitive 3/hr rate limiter)
	r.POST("/api/users/forgot-password", func(c *gin.Context) {
		if !resetLimiter.Allow(c.ClientIP()) {
			c.JSON(429, gin.H{"error": "Too many reset attempts. Please try again in an hour."})
			c.Abort()
			return
		}
		uc.ForgotPasswordInitiate(c)
	})
	r.POST("/api/users/reset-password/confirm", func(c *gin.Context) {
		if !resetLimiter.Allow(c.ClientIP()) {
			c.JSON(429, gin.H{"error": "Too many reset attempts. Please try again in an hour."})
			c.Abort()
			return
		}
		uc.ConfirmResetPassword(c)
	})

	// 2. PROTECTED (Common logic for Users and Admins)
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware(publicKey))
	{
		// These routes use OwnerOrAdmin to verify the authenticated user
		// either owns the resource (:id matches their JWT's sub claim) or is an admin.
		protected.GET("/users/:id", middleware.OwnerOrAdmin("id"), uc.GetUser)
		protected.PUT("/users/:id", middleware.OwnerOrAdmin("id"), uc.UpdateUser)
		protected.DELETE("/users/:id", middleware.OwnerOrAdmin("id"), uc.DeleteUser)

		// 3. ADMIN ONLY
		admin := protected.Group("/admin")
		admin.Use(middleware.AdminOnly())
		{
			admin.GET("/users", uc.GetAllUsers)
			// Sensitive: Limit to 3 requests per hour per IP
			admin.POST("/users/:id/reset-password", func(c *gin.Context) {
				if !resetLimiter.Allow(c.ClientIP()) {
					c.JSON(429, gin.H{"error": "Too many reset requests. Limit is 3 per hour."})
					c.Abort()
					return
				}
				uc.AdminResetPassword(c)
			})
			// Example: An admin-only feature that regular users can't even attempt
			// admin.GET("/dashboard-stats", uc.GetSystemStats)
			// admin.POST("/users/:id/suspend", uc.SuspendUser)
		}
	}
}

func InitializeInkRoutes(r *gin.Engine, ic *controllers.InkController, publicKey *rsa.PublicKey) {
	inkGrp := r.Group("/api/inktochat")
	{
		// 1. PUBLIC
		inkGrp.POST("/token", ic.GetToken)
		inkGrp.GET("/spam", ic.GetSpamDrops)

		// 2. ADMIN ONLY
		adminStats := inkGrp.Group("/analytics")
		adminStats.Use(middleware.AuthMiddleware(publicKey), middleware.AdminOnly())
		{
			adminStats.GET("/", ic.GetAnalytics)
			adminStats.POST("/reset", ic.ResetSystem)
			adminStats.POST("/clear-chats", ic.ClearLobbies)
		}

		// 3. PROTECTED LOBBY ACTIONS (Anti-Spoofing Enabled)
		lobby := inkGrp.Group("/")
		lobby.Use(middleware.AuthMiddleware(publicKey))
		{
			lobby.POST("/draw", ic.PostDrawing)
			lobby.POST("/message", ic.PostMessage)
		}

		// 4. AUTHENTICATED USERS ONLY (Personal Save Gallery)
		drawingsGrp := inkGrp.Group("/drawings")
		drawingsGrp.Use(middleware.AuthMiddleware(publicKey))
		{
			drawingsGrp.POST("", ic.SavePersonalDrawing)
			drawingsGrp.GET("", ic.GetPersonalDrawings)
			drawingsGrp.DELETE("/:id", ic.DeletePersonalDrawing)
		}
	}
}
