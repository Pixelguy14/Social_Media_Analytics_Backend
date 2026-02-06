package routes

import (
	"DataTracker/app/controllers"
	"DataTracker/app/middleware"

	"github.com/gin-gonic/gin"
)

func InitializeUserRoutes(r *gin.Engine, uc *controllers.UserController) {
	// 1. PUBLIC
	r.POST("/api/users/login", uc.Login)
	r.POST("/api/users/", uc.CreateUser)

	// 2. PROTECTED (Common logic for Users and Admins)
	// Both groups use AuthMiddleware
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		// These use your "Smart" controllers that check (Owner OR Admin)
		protected.GET("/users/:id", uc.GetUser)
		protected.PUT("/users/:id", uc.UpdateUser)
		protected.DELETE("/users/:id", uc.DeleteUser)

		// 3. ADMIN ONLY (Strictly for things ONLY admins can do)
		admin := protected.Group("/admin")
		admin.Use(middleware.AdminOnly())
		{
			// Example: An admin-only feature that regular users can't even attempt
			// admin.GET("/dashboard-stats", uc.GetSystemStats)
			// admin.POST("/users/:id/suspend", uc.SuspendUser)
		}
	}
}
