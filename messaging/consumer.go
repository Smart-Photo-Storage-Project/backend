package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"photo-storage-backend/repository"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EmbeddingResult struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	UserID   string `json:"user_id"`
	UploadAt int64  `json:"upload_at"`
}

func StartEmbeddingResultConsumer(rmqURL string) error {
	conn, err := amqp.Dial(rmqURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	q, err := ch.QueueDeclare(
		"embedding_results",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,  // auto-ack
		false, // exclusive
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	go func() {
		for d := range msgs {
			var result EmbeddingResult
			if err := json.Unmarshal(d.Body, &result); err != nil {
				log.Printf("Failed to parse embedding result: %v", err)
				continue
			}

			log.Printf("Received embedding result for photo: %s", result.Name)
			log.Printf("Received embedding result for photo: %s", result.UserID)

			// Update MongoDB
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := repository.MarkAsEmbedded(ctx, result.UserID, result.Name)
			if err != nil {
				log.Printf("Failed to update photo status: %v", err)
			}
		}
	}()

	log.Println("Embedding result consumer started...")
	return nil
}
