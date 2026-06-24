package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"towercore/internal/core/services"
	"towercore/pkg/apierror"
)

type AuthHandler struct {
	service *services.AuthService
}

func NewAuthHandler(service *services.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

type loginRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	TokenType string `json:"token_type"`
	Token     string `json:"access_token"`
	ExpiresAt string `json:"expires_at"`
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
}

func (h *AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierror.MethodNotAllowed(w)
		return
	}
	h.login(w, r)
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "invalid json payload")
		return
	}

	identifier := strings.TrimSpace(req.Username)
	if identifier == "" {
		identifier = strings.TrimSpace(req.Email)
	}

	token, user, expiresAt, err := h.service.Login(
		r.Context(),
		identifier,
		strings.TrimSpace(req.Password),
	)
	if err != nil {
		if err.Error() == "invalid credentials" {
			apierror.Unauthorized(w, r)
			return
		}
		apierror.BadRequest(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(loginResponse{
		TokenType: "Bearer",
		Token:     token,
		ExpiresAt: expiresAt.UTC().Format(http.TimeFormat),
		UserID:    user.ID,
		Username:  user.Username,
		Role:      user.Role,
	})
}