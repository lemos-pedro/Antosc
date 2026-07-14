package dto

// PredictionResponse representa uma previsão IA para uma torre.
// Se não existir previsão, a API devolve 404 — nunca um valor inventado
// (consistente com a regra "no fabricated data" do projeto).
type PredictionResponse struct {
	TowerID          string  `json:"tower_id"`
	Model            string  `json:"model"`
	Score            float64 `json:"score"`
	Status           string  `json:"status"`
	Explanation      string  `json:"explanation,omitempty"`
	PredictionWindow int     `json:"prediction_window_days"`
	CreatedAt        string  `json:"created_at"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
