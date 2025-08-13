package handlers

import (
	"context"
	"net/http"
	"photo-storage-backend/database"
	"photo-storage-backend/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetNotifications(c *gin.Context) {
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))
	notifCollection := database.GetNotificationCollection()

	cursor, err := notifCollection.Find(context.Background(), bson.M{"user_id": userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch notifications"})
		return
	}

	var notifs []models.Notification
	if err := cursor.All(context.Background(), &notifs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode notifications"})
		return
	}

	c.JSON(http.StatusOK, notifs)
}

func MarkNotificationsRead(c *gin.Context) {
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))
	notifCollection := database.GetNotificationCollection()

	// Parse request body
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Convert string IDs to ObjectIDs
	objectIDs := make([]primitive.ObjectID, 0, len(body.IDs))
	for _, idStr := range body.IDs {
		objID, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			continue // skip invalid IDs
		}
		objectIDs = append(objectIDs, objID)
	}

	if len(objectIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no valid IDs provided"})
		return
	}

	// Filter notifications belonging to this user
	filter := bson.M{
		"_id":     bson.M{"$in": objectIDs},
		"user_id": userID,
	}

	update := bson.M{
		"$set": bson.M{"read": true},
	}

	_, err := notifCollection.UpdateMany(context.Background(), filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
