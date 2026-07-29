package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
	"towercore/internal/infrastructure/security"
	"towercore/pkg/apierror"

	"github.com/google/uuid"
)

type UserHandler struct {
	repo interfaces.UserRepository
}

func NewUserHandler(repo interfaces.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

type createUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (h *UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.create(w, r)
	case http.MethodGet:
		h.list(w, r)
	default:
		apierror.MethodNotAllowed(w)
	}
}

func (h *UserHandler) list(w http.ResponseWriter, r *http.Request) {
	users, err := h.repo.List(r.Context())
	if err != nil {
		apierror.Internal(w)
		return
	}
	if users == nil {
		users = []domain.User{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(users)
}

func (h *UserHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "invalid json payload")
		return
	}

	req.Username = strings.ToLower(strings.TrimSpace(req.Username))
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Password = strings.TrimSpace(req.Password)
	req.Role = strings.ToLower(strings.TrimSpace(req.Role))

	if req.Username == "" || req.Password == "" {
		apierror.BadRequest(w, "username and password are required")
		return
	}
	if req.Role == "" {
		req.Role = "viewer"
	}

	hash, err := security.HashPassword(req.Password)
	if err != nil {
		apierror.Internal(w)
		return
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:           uuid.NewString(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         req.Role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.repo.Create(r.Context(), user); err != nil {
		if strings.Contains(err.Error(), "unique") {
			apierror.Write(w, http.StatusConflict, "conflict", "username or email already exists")
			return
		}
		apierror.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"user_id":    user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"role":       user.Role,
		"created_at": user.CreatedAt,
	})
}