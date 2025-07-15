package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Photo struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name     string             `bson:"name" json:"name"`
	Path     string             `bson:"path" json:"path"`
	UploadAt int64              `bson:"upload_at" json:"upload_at"`
}
