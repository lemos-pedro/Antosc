package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/antosc/aip/internal/api/dto"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	resp := dto.HealthResponse{
		Status:  "ok",
		Service: "aip-api",
		Time:    time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
