package handlers

import (
	"net/http"
	"strconv"

	"github.com/antosc/aip/internal/analytics"
	"github.com/antosc/aip/internal/repository/postgres"
)

type EmbeddingsHandler struct {
	embeddings postgres.EmbeddingRepository
}

func NewEmbeddingsHandler(embeddings postgres.EmbeddingRepository) *EmbeddingsHandler {
	return &EmbeddingsHandler{embeddings: embeddings}
}

// Similar — GET /api/v1/towers/{tower_id}/similar?k=5
func (h *EmbeddingsHandler) Similar(w http.ResponseWriter, r *http.Request) {
	towerID := r.PathValue("tower_id")
	k, _ := strconv.Atoi(r.URL.Query().Get("k"))
	if k <= 0 {
		k = 5
	}
	target, err := h.embeddings.Get(r.Context(), towerID)
	if err != nil {
		http.Error(w, "erro ao ler embedding", http.StatusInternalServerError)
		return
	}
	if target == nil {
		http.Error(w, "embedding não encontrado — corre embeddings_job", http.StatusNotFound)
		return
	}
	all, err := h.embeddings.ListAll(r.Context())
	if err != nil {
		http.Error(w, "erro ao listar embeddings", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tower_id": towerID,
		"similar":  analytics.TopSimilar(*target, all, k),
		"model":    target.ModelVersion,
	})
}
