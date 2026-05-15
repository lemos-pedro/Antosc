package apierror

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Write(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{
		Code:    code,
		Message: message,
	})
}

func Unauthorized(w http.ResponseWriter, _ *http.Request) {
	Write(w, http.StatusUnauthorized, "unauthorized", http.StatusText(http.StatusUnauthorized))
}

func BadRequest(w http.ResponseWriter, message string) {
	Write(w, http.StatusBadRequest, "bad_request", message)
}

func Internal(w http.ResponseWriter) {
	Write(w, http.StatusInternalServerError, "internal_error", "internal server error")
}

func MethodNotAllowed(w http.ResponseWriter) {
	Write(w, http.StatusMethodNotAllowed, "method_not_allowed", http.StatusText(http.StatusMethodNotAllowed))
}
