package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"photo-storage-backend/models"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EmbedJob struct {
	UserID   string      `json:"user_id"`
	UploadAt int64       `json:"upload_at"`
	Photos   []PhotoMeta `json:"photos"`
	BatchID  string      `json:"batch_id"`
}

type PhotoMeta struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func PublishEmbeddingJob(rmqURL string, photos []models.Photo) error {
	conn, err := amqp.Dial(rmqURL)
	if err != nil {
		return fmt.Errorf("connect to RabbitMQ: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"embedding_jobs",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	job := EmbedJob{
		UserID:   photos[0].UserID.Hex(),
		UploadAt: photos[0].UploadAt,
		Photos:   make([]PhotoMeta, len(photos)),
		BatchID:  photos[0].BatchID.Hex(),
	}
	for i, p := range photos {
		job.Photos[i] = PhotoMeta{
			Name: p.Name,
			Path: p.Path,
		}
	}

	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = ch.PublishWithContext(ctx,
		"",     // default exchange
		q.Name, // routing key
		false, false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         data,
		})

	if err != nil {
		return fmt.Errorf("publish message: %w", err)
	}

	log.Println("Job published to RabbitMQ")
	return nil
}
