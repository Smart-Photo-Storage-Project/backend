package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"photo-storage-backend/database"
	"photo-storage-backend/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func UploadPhoto(c *gin.Context) {
	// Parse uploaded file
	file, err := c.FormFile("photo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "photo file is required"})
		return
	}

	name := c.PostForm("name")
	if name == "" {
		name = file.Filename
	}

	// Create uploads directory if it doesn't exist
	uploadDir := "uploads"
	err = os.MkdirAll(uploadDir, os.ModePerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create upload directory"})
		return
	}

	// Save file to disk
	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), file.Filename)
	filepath := filepath.Join(uploadDir, filename)
	if err := c.SaveUploadedFile(file, filepath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save file"})
		return
	}

	// Create DB record
	photo := models.Photo{
		ID:       primitive.NewObjectID(),
		Name:     name,
		Path:     filepath,
		UploadAt: time.Now().Unix(),
	}

	collection := database.GetPhotoCollection()
	_, err = collection.InsertOne(context.Background(), photo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save photo metadata"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "upload successful", "photo": photo})
}

func ListPhotos(c *gin.Context) {
	// Parse pagination params
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}

	skip := (page - 1) * limit

	collection := database.GetPhotoCollection()

	// Sort by newest upload first
	findOptions := options.Find()
	findOptions.SetSkip(int64(skip))
	findOptions.SetLimit(int64(limit))
	findOptions.SetSort(bson.M{"upload_at": -1})

	cursor, err := collection.Find(context.Background(), bson.M{}, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch photos"})
		return
	}
	defer cursor.Close(context.Background())

	var photos []models.Photo
	if err := cursor.All(context.Background(), &photos); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode results"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"page":   page,
		"limit":  limit,
		"photos": photos,
	})
}

func SearchPhotos(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query param `q` is required"})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}

	skip := (page - 1) * limit

	collection := database.GetPhotoCollection()

	filter := bson.M{
		"name": bson.M{
			"$regex":   query,
			"$options": "i", // case-insensitive partial match
		},
	}

	findOptions := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.M{"upload_at": -1})

	cursor, err := collection.Find(context.Background(), filter, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search"})
		return
	}
	defer cursor.Close(context.Background())

	var results []models.Photo
	if err := cursor.All(context.Background(), &results); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode results"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"page":   page,
		"limit":  limit,
		"query":  query,
		"photos": results,
	})
}
