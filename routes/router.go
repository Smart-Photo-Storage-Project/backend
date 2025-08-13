package routes

import (
	"photo-storage-backend/handlers"
	"photo-storage-backend/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")

	// Public routes
	api.POST("/register", handlers.Register)
	api.POST("/login", handlers.Login)

	// Protected routes
	apiAuth := api.Group("/")
	apiAuth.Use(middleware.AuthMiddleware())
	{
		apiAuth.POST("/upload", handlers.UploadPhotos)
		apiAuth.GET("/photos", handlers.ListPhotos)
		apiAuth.GET("/search", handlers.SearchPhotos)
		apiAuth.GET("/notification", handlers.GetNotifications)
		apiAuth.POST("/notification", handlers.MarkNotificationsRead)
	}
}
