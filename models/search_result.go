package models

type InferenceSearchResult struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	UserID   string  `json:"user_id"`
	UploadAt int64   `json:"upload_at"`
	Path     string  `json:"path"`
	Score    float64 `json:"score"`
}
