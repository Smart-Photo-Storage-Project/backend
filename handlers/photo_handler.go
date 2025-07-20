package handlers

import (
	"context"
	"fmt"
	"math"
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

func UploadPhotos(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form data"})
		return
	}

	files := form.File["photos"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no photos uploaded"})
		return
	}

	uploadDir := "uploads"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create upload directory"})
		return
	}

	// Get user ID from context
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID"})
		return
	}

	collection := database.GetPhotoCollection()

	var photoDocs []interface{}
	var uploadedPhotos []models.Photo
	var failedPhotos []string

	for _, file := range files {
		name := file.Filename
		filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), name)
		filePath := filepath.Join(uploadDir, filename)

		if err := c.SaveUploadedFile(file, filePath); err != nil {
			// Track filename that failed
			failedPhotos = append(failedPhotos, name)
			continue
		}

		photo := models.Photo{
			ID:       primitive.NewObjectID(),
			Name:     name,
			Path:     filePath,
			UploadAt: time.Now().Unix(),
			UserID:   userID,
		}

		photoDocs = append(photoDocs, photo)
		uploadedPhotos = append(uploadedPhotos, photo)
	}

	if len(photoDocs) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no valid files to upload"})
		return
	}

	// Batch insert metadata
	if _, err := collection.InsertMany(context.Background(), photoDocs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save photo metadata"})
		return
	}

	// Success response
	c.JSON(http.StatusOK, gin.H{
		"message":        "batch upload completed",
		"uploaded_count": len(uploadedPhotos),
		"failed_count":   len(failedPhotos),
		"uploaded":       uploadedPhotos,
		"failed_files":   failedPhotos,
	})
}

func ListPhotos(c *gin.Context) {
	// Parse pagination params
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "21")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 21
	}

	skip := (page - 1) * limit

	collection := database.GetPhotoCollection()

	// Sort by newest upload first
	findOptions := options.Find()
	findOptions.SetSkip(int64(skip))
	findOptions.SetLimit(int64(limit))
	findOptions.SetSort(bson.M{"upload_at": -1})

	// User ID from JWT
	userIDStr, _ := c.Get("userID")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	filter := bson.M{
		"user_id": userID,
	}

	totalCount, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count documents"})
		return
	}

	cursor, err := collection.Find(context.Background(), filter, findOptions)
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

	// Calculate totalPages
	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))

	c.JSON(http.StatusOK, gin.H{
		"page":       page,
		"limit":      limit,
		"photos":     photos,
		"total":      totalCount,
		"totalPages": totalPages,
	})
}

func SearchPhotos(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query param `q` is required"})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "21")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 21
	}

	skip := (page - 1) * limit

	collection := database.GetPhotoCollection()

	// User ID from JWT
	userIDStr, _ := c.Get("userID")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	filter := bson.M{
		"user_id": userID,
		"name": bson.M{
			"$regex":   query,
			"$options": "i",
		},
	}

	// Count total matching docs first
	totalCount, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count documents"})
		return
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

	// Calculate totalPages
	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))

	c.JSON(http.StatusOK, gin.H{
		"page":       page,
		"limit":      limit,
		"query":      query,
		"photos":     results,
		"total":      totalCount,
		"totalPages": totalPages,
	})
}
