package repository

import (
	"context"
	"log"
	"photo-storage-backend/database"
	"photo-storage-backend/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func UpdateNotificationProgress(ctx context.Context, userIDStr string, batchIDStr string) error {
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		log.Printf("Invalid userID: %v", err)
		return err
	}

	batchID, err := primitive.ObjectIDFromHex(batchIDStr)
	if err != nil {
		log.Printf("Invalid userID: %v", err)
		return err
	}

	notifCollection := database.GetNotificationCollection()

	filter := bson.M{"user_id": userID, "batch_id": batchID}

	update := bson.M{
		"$inc": bson.M{"completed": 1},
		"$set": bson.M{"read": false},
	}

	res := notifCollection.FindOneAndUpdate(ctx, filter, update, options.FindOneAndUpdate().SetReturnDocument(options.After))
	if res.Err() != nil {
		return res.Err()
	}

	var notif models.Notification
	if err := res.Decode(&notif); err != nil {
		return err
	}

	// Update status if completed
	if notif.Completed >= notif.Total {
		_, err := notifCollection.UpdateOne(ctx, filter, bson.M{"$set": bson.M{
			"status":  "completed",
			"message": "All photos embedded successfully",
		}})
		return err
	}

	return nil
}
