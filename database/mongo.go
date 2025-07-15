package database

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var photoCollection *mongo.Collection
var userCollection *mongo.Collection

func InitMongo(uri, dbName string) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("Mongo connection failed:", err)
	}
	photoCollection = client.Database(dbName).Collection("photos")
	userCollection = client.Database(dbName).Collection("users")
}

func GetPhotoCollection() *mongo.Collection {
	return photoCollection
}

func GetUserCollection() *mongo.Collection {
	return userCollection
}
