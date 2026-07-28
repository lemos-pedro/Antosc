package handlers

import (
	"encoding/json"
	"net/http"
)

// writeJSON escreve uma resposta JSON com o status HTTP indicado.
// Partilhado por todos os handlers para manter o formato de resposta consistente.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
