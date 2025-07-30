package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
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

	// Image embedding
	inferenceURL := os.Getenv("INFERENCE_URL")
	fmt.Println(inferenceURL)
	if inferenceURL == "" {
		inferenceURL = "http://localhost:8000/embed/images"
	}
	resp, err := SendImagesToInference(uploadedPhotos, inferenceURL)
	if err != nil || resp.StatusCode >= 300 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send images to inference"})
		return // Blocking for now, will be refactored to asynchronous later
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

	userIDStr, _ := c.Get("userID")
	userID := userIDStr.(string)

	// Prepare JSON body
	reqBody := map[string]string{
		"text":    query,
		"user_id": userID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode request"})
		return
	}

	// Send request to inference service

	inferenceSearchURL := os.Getenv("INFERENCE_SEARCH_URL")
	if inferenceSearchURL == "" {
		inferenceSearchURL = "http://localhost:8000/embed/text"
	}
	fmt.Println(inferenceSearchURL)
	resp, err := http.Post(inferenceSearchURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "inference service error"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "inference failed", "details": string(body)})
		return
	}

	// Parse response from inference
	var inferenceResponse struct {
		Results []models.InferenceSearchResult `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&inferenceResponse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse inference response"})
		return
	}

	// Wrap to match expected frontend format
	c.JSON(http.StatusOK, gin.H{
		"page":       1,
		"limit":      len(inferenceResponse.Results),
		"query":      query,
		"photos":     inferenceResponse.Results,
		"total":      len(inferenceResponse.Results),
		"totalPages": 1,
	})
}

func SendImagesToInference(images []models.Photo, inferenceURL string) (*http.Response, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if len(images) == 0 {
		return nil, errors.New("no images to send")
	}

	_ = writer.WriteField("user_id", images[0].UserID.Hex())
	_ = writer.WriteField("upload_at", strconv.FormatInt(images[0].UploadAt, 10))

	for _, photo := range images {
		file, err := os.Open(photo.Path)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		// Add file
		part, err := writer.CreateFormFile("files", filepath.Base(photo.Path))
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(part, file); err != nil {
			return nil, err
		}

		// Add per-image metadata
		_ = writer.WriteField("names", photo.Name)
		_ = writer.WriteField("paths", photo.Path)
	}

	_ = writer.Close()

	req, err := http.NewRequest("POST", inferenceURL, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	return client.Do(req)
}
