package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Notification struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BatchID   primitive.ObjectID `bson:"batch_id" json:"batch_id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	CreatedAt int64              `bson:"created_at" json:"created_at"`
	Status    string             `bson:"status" json:"status"`
	Total     int                `bson:"total" json:"total"`
	Completed int                `bson:"completed" json:"completed"`
	Failed    int                `bson:"failed" json:"failed"`
	Message   string             `bson:"message,omitempty" json:"message,omitempty"`
	Read      bool               `bson: "read" json:"read"`
}
