package repository

import (
	"context"
	"log"
	"photo-storage-backend/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func MarkAsEmbedded(ctx context.Context, userIDStr string, name string) error {
	userId, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		log.Printf("Invalid userID: %v", err)
		return err
	}

	filter := bson.M{
		"user_id": userId,
		"name":    name,
	}
	update := bson.M{
		"$set": bson.M{"embedded": true},
	}

	collection := database.GetPhotoCollection()

	res, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("Update failed: %v", err)
	} else {
		log.Printf("Matched: %d, Modified: %d", res.MatchedCount, res.ModifiedCount)
	}
	return err
}
