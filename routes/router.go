package routes

import (
	"photo-storage-backend/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.POST("/upload", handlers.UploadPhoto)
		api.GET("/photos", handlers.ListPhotos)
		api.GET("/search", handlers.SearchPhotos)
	}
}
