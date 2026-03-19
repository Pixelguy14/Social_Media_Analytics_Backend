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
			admin.GET("/users", uc.GetAllUsers)
			// Example: An admin-only feature that regular users can't even attempt
			// admin.GET("/dashboard-stats", uc.GetSystemStats)
			// admin.POST("/users/:id/suspend", uc.SuspendUser)
		}
	}
}

func InitializeInkRoutes(r *gin.Engine, ic *controllers.InkController) {
	inkGrp := r.Group("/api/inktochat")
	{
		// 1. PUBLIC
		inkGrp.POST("/token", ic.GetToken)
		inkGrp.GET("/spam", ic.GetSpamDrops)

		// 2. ADMIN ONLY
		adminStats := inkGrp.Group("/analytics")
		adminStats.Use(middleware.AuthMiddleware(), middleware.AdminOnly())
		{
			adminStats.GET("/", ic.GetAnalytics)
			adminStats.POST("/reset", ic.ResetSystem)
			adminStats.POST("/clear-chats", ic.ClearLobbies)
		}

		// 3. PROTECTED (Messaging & Drawing)
		// We allow anonymous/guest users here as long as they have a custom token? 
		// Actually, the user has been using them as PUBLIC but with internal rate limits.
		// Let's keep them in the root of inkGrp for now as per previous main.go state.
		inkGrp.POST("/draw", ic.PostDrawing)
		inkGrp.POST("/message", ic.PostMessage)

		// 4. AUTHENTICATED USERS ONLY (Personal Save Gallery)
		drawingsGrp := inkGrp.Group("/drawings")
		drawingsGrp.Use(middleware.AuthMiddleware())
		{
			drawingsGrp.POST("", ic.SavePersonalDrawing)
			drawingsGrp.GET("", ic.GetPersonalDrawings)
			drawingsGrp.DELETE("/:id", ic.DeletePersonalDrawing)
		}
	}
}
