package utils

import (
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey []byte

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is not set in environment")
	}
	log.Println("[JWT INIT] JWT_SECRET loaded successfully")
	log.Printf("[JWT TOKEN] %s\n", secret)

	jwtKey = []byte(secret)
}

// jwtKey := []byte("my-32-character-ultra-secure-and-ultra-long-secret")

func GenerateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func ParseToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil || !token.Valid {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if uid, ok := claims["user_id"].(string); ok {
			return uid, nil
		}
	}

	return "", err
}
