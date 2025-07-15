package main

import (
	"log"
	"os"

	"photo-storage-backend/database"
	"photo-storage-backend/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}
	dbName := "photo_storage"

	// Connect to MongoDB
	database.InitMongo(mongoURI, dbName)
	log.Println("Connected to MongoDB")

	// Set up router
	r := gin.Default()

	// Static Files
	r.Static("/uploads", "./uploads")

	// Set up routes
	routes.SetupRoutes(r)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server running at http://localhost:%s\n", port)
	r.Run(":" + port)
}
