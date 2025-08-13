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
