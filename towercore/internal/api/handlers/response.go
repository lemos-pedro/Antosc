package handlers

import (
	"encoding/json"
	"net/http"
)

// writeJSON serializa v como JSON e escreve no response com o status dado.
// Usado por todos os handlers do package para manter uma resposta
// consistente em toda a API.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// apiErrorBody é o envelope de erro padrão documentado em api.md:
// { "error": { "code": ..., "message": ..., "request_id": ... } }
type apiErrorBody struct {
	Error apiErrorDetail `json:"error"`
}

type apiErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// writeError escreve o envelope de erro padrão com o status HTTP dado.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, apiErrorBody{
		Error: apiErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
