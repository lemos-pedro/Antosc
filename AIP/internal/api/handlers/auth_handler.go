package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/antosc/aip/internal/api/dto"
	"github.com/antosc/aip/internal/auth"
	"github.com/antosc/aip/internal/repository/postgres"
)

type AuthHandler struct {
	users           postgres.UserRepository
	tokens          *auth.TokenService
	audit           postgres.AuthAuditRepository
	refresh         postgres.RefreshTokenRepository
	refreshTTL      time.Duration
	maxFailedLogins int
	lockoutMinutes  int
}

func NewAuthHandler(
	users postgres.UserRepository,
	tokens *auth.TokenService,
	audit postgres.AuthAuditRepository,
	refresh postgres.RefreshTokenRepository,
	refreshTTLHours, maxFailed, lockMinutes int,
) *AuthHandler {
	if refreshTTLHours <= 0 {
		refreshTTLHours = 168
	}
	if maxFailed <= 0 {
		maxFailed = 5
	}
	if lockMinutes <= 0 {
		lockMinutes = 15
	}
	return &AuthHandler{
		users:           users,
		tokens:          tokens,
		audit:           audit,
		refresh:         refresh,
		refreshTTL:      time.Duration(refreshTTLHours) * time.Hour,
		maxFailedLogins: maxFailed,
		lockoutMinutes:  lockMinutes,
	}
}

func (h *AuthHandler) logAuth(r *http.Request, userID *string, email, event, detail string) {
	if h.audit == nil {
		return
	}
	_ = h.audit.Log(r.Context(), postgres.AuthAuditEntry{
		UserID:    userID,
		Email:     email,
		Event:     event,
		IP:        clientIP(r),
		UserAgent: r.UserAgent(),
		Detail:    detail,
	})
}

func clientIP(r *http.Request) string {
	if x := r.Header.Get("X-Forwarded-For"); x != "" {
		return strings.TrimSpace(strings.Split(x, ",")[0])
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i >= 0 {
		return host[:i]
	}
	return host
}

func (h *AuthHandler) issuePair(w http.ResponseWriter, r *http.Request, u postgres.User, status int) {
	access, exp, err := h.tokens.Issue(u.ID, u.Email, u.Role, u.FullName)
	if err != nil {
		http.Error(w, "erro ao emitir token", http.StatusInternalServerError)
		return
	}
	plain, hash, err := auth.NewRefreshToken()
	if err != nil {
		http.Error(w, "erro ao emitir refresh token", http.StatusInternalServerError)
		return
	}
	expiresRefresh := time.Now().Add(h.refreshTTL)
	if h.refresh != nil {
		_ = h.refresh.Create(r.Context(), u.ID, hash, expiresRefresh, clientIP(r), r.UserAgent())
	}
	writeJSON(w, status, dto.LoginResponse{
		Token:        access,
		RefreshToken: plain,
		ExpiresAt:    exp.UTC().Format(time.RFC3339),
		User:         toUserResponse(u),
	})
}

// Login — POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corpo inválido", http.StatusBadRequest)
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		http.Error(w, "email e password são obrigatórios", http.StatusBadRequest)
		return
	}

	u, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil || u == nil {
		h.logAuth(r, nil, req.Email, "login_failure", "user não encontrado")
		http.Error(w, "email ou password incorretos", http.StatusUnauthorized)
		return
	}

	if u.LockedUntil.Valid && u.LockedUntil.Time.After(time.Now()) {
		h.logAuth(r, &u.ID, u.Email, "login_failure", "conta bloqueada")
		http.Error(w, "conta temporariamente bloqueada — tente mais tarde", http.StatusForbidden)
		return
	}
	if !u.Active {
		h.logAuth(r, &u.ID, u.Email, "login_failure", "user inactivo")
		http.Error(w, "email ou password incorretos", http.StatusUnauthorized)
		return
	}
	if !auth.CheckPassword(u.PasswordHash, req.Password) {
		_ = h.users.RegisterFailedLogin(r.Context(), u.ID, h.maxFailedLogins, h.lockoutMinutes)
		h.logAuth(r, &u.ID, u.Email, "login_failure", "password incorrecta")
		http.Error(w, "email ou password incorretos", http.StatusUnauthorized)
		return
	}

	_ = h.users.ResetFailedLogins(r.Context(), u.ID)
	_ = h.users.TouchLastLogin(r.Context(), u.ID)
	h.logAuth(r, &u.ID, u.Email, "login_success", "role="+u.Role)
	h.issuePair(w, r, *u, http.StatusOK)
}

// Refresh — POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.RefreshToken) == "" {
		http.Error(w, "refresh_token obrigatório", http.StatusBadRequest)
		return
	}
	if h.refresh == nil {
		http.Error(w, "refresh não configurado", http.StatusServiceUnavailable)
		return
	}
	hash := auth.HashRefreshToken(req.RefreshToken)
	rt, err := h.refresh.GetValid(r.Context(), hash)
	if err != nil {
		h.logAuth(r, nil, "", "refresh_failure", "token inválido")
		http.Error(w, "refresh token inválido ou expirado", http.StatusUnauthorized)
		return
	}
	// rotação: revoga o actual
	_ = h.refresh.Revoke(r.Context(), hash)

	u, err := h.users.GetByID(r.Context(), rt.UserID)
	if err != nil || u == nil || !u.Active {
		http.Error(w, "utilizador inválido", http.StatusUnauthorized)
		return
	}
	h.logAuth(r, &u.ID, u.Email, "refresh_success", "")
	h.issuePair(w, r, *u, http.StatusOK)
}

// Logout — POST /api/v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req dto.LogoutRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if h.refresh != nil && strings.TrimSpace(req.RefreshToken) != "" {
		_ = h.refresh.Revoke(r.Context(), auth.HashRefreshToken(req.RefreshToken))
	}
	if p, ok := auth.PrincipalFrom(r.Context()); ok && h.refresh != nil {
		_ = h.refresh.RevokeAllForUser(r.Context(), p.ID)
		h.logAuth(r, &p.ID, p.Email, "logout", "")
	}
	w.WriteHeader(http.StatusNoContent)
}

// Me — GET /api/v1/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFrom(r.Context())
	if !ok {
		http.Error(w, "não autenticado", http.StatusUnauthorized)
		return
	}
	u, err := h.users.GetByID(r.Context(), p.ID)
	if err != nil || u == nil {
		http.Error(w, "utilizador não encontrado", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(*u))
}

// Bootstrap — POST /api/v1/auth/bootstrap
func (h *AuthHandler) Bootstrap(w http.ResponseWriter, r *http.Request) {
	n, err := h.users.Count(r.Context())
	if err != nil {
		http.Error(w, "erro ao verificar utilizadores", http.StatusInternalServerError)
		return
	}
	if n > 0 {
		http.Error(w, "bootstrap indisponível — já existem utilizadores", http.StatusConflict)
		return
	}

	var req dto.BootstrapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corpo inválido", http.StatusBadRequest)
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" || req.FullName == "" {
		http.Error(w, "email, full_name e password são obrigatórios", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		http.Error(w, "password deve ter pelo menos 8 caracteres", http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "erro ao criar password", http.StatusInternalServerError)
		return
	}

	u, err := h.users.Create(r.Context(), postgres.User{
		Email: req.Email, FullName: req.FullName, PasswordHash: hash, Role: "admin",
	})
	if err != nil {
		if errors.Is(err, postgres.ErrEmailTaken) {
			http.Error(w, "email já registado", http.StatusConflict)
			return
		}
		http.Error(w, "erro ao criar admin", http.StatusInternalServerError)
		return
	}
	h.logAuth(r, &u.ID, u.Email, "bootstrap", "primeiro admin criado")
	h.issuePair(w, r, u, http.StatusCreated)
}

// CreateUser — POST /api/v1/auth/users (admin)
func (h *AuthHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corpo inválido", http.StatusBadRequest)
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Role = strings.TrimSpace(strings.ToLower(req.Role))
	if req.Email == "" || req.FullName == "" || req.Password == "" || req.Role == "" {
		http.Error(w, "email, full_name, password e role são obrigatórios", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		http.Error(w, "password deve ter pelo menos 8 caracteres", http.StatusBadRequest)
		return
	}
	if !auth.IsValidRole(req.Role) {
		http.Error(w, "role inválida", http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "erro ao criar password", http.StatusInternalServerError)
		return
	}
	u, err := h.users.Create(r.Context(), postgres.User{
		Email: req.Email, FullName: req.FullName, PasswordHash: hash, Role: req.Role,
	})
	if err != nil {
		if errors.Is(err, postgres.ErrEmailTaken) {
			http.Error(w, "email já registado", http.StatusConflict)
			return
		}
		http.Error(w, "erro ao criar utilizador", http.StatusInternalServerError)
		return
	}
	h.logAuth(r, &u.ID, u.Email, "user_created", "role="+u.Role)
	writeJSON(w, http.StatusCreated, toUserResponse(u))
}

func (h *AuthHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	list, err := h.users.List(r.Context(), 200)
	if err != nil {
		http.Error(w, "erro ao listar utilizadores", http.StatusInternalServerError)
		return
	}
	out := make([]dto.UserResponse, 0, len(list))
	for _, u := range list {
		out = append(out, toUserResponse(u))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AuthHandler) ListAuthAudit(w http.ResponseWriter, r *http.Request) {
	if h.audit == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	list, err := h.audit.List(r.Context(), 200)
	if err != nil {
		http.Error(w, "erro ao listar audit", http.StatusInternalServerError)
		return
	}
	type row struct {
		ID        string  `json:"id"`
		UserID    *string `json:"user_id,omitempty"`
		Email     string  `json:"email"`
		Event     string  `json:"event"`
		IP        string  `json:"ip"`
		Detail    string  `json:"detail,omitempty"`
		CreatedAt string  `json:"created_at"`
	}
	out := make([]row, 0, len(list))
	for _, e := range list {
		out = append(out, row{
			ID: e.ID, UserID: e.UserID, Email: e.Email, Event: e.Event,
			IP: e.IP, Detail: e.Detail, CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AuthHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req dto.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corpo inválido", http.StatusBadRequest)
		return
	}
	if req.Role != nil {
		role := strings.TrimSpace(strings.ToLower(*req.Role))
		if !auth.IsValidRole(role) {
			http.Error(w, "role inválida", http.StatusBadRequest)
			return
		}
		if err := h.users.UpdateRole(r.Context(), id, role); err != nil {
			if errors.Is(err, postgres.ErrUserNotFound) {
				http.Error(w, "utilizador não encontrado", http.StatusNotFound)
				return
			}
			http.Error(w, "erro ao atualizar role", http.StatusInternalServerError)
			return
		}
	}
	if req.Active != nil {
		if err := h.users.UpdateActive(r.Context(), id, *req.Active); err != nil {
			if errors.Is(err, postgres.ErrUserNotFound) {
				http.Error(w, "utilizador não encontrado", http.StatusNotFound)
				return
			}
			http.Error(w, "erro ao atualizar estado", http.StatusInternalServerError)
			return
		}
	}
	u, err := h.users.GetByID(r.Context(), id)
	if err != nil || u == nil {
		http.Error(w, "utilizador não encontrado", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(*u))
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFrom(r.Context())
	if !ok {
		http.Error(w, "não autenticado", http.StatusUnauthorized)
		return
	}
	var req dto.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corpo inválido", http.StatusBadRequest)
		return
	}
	if len(req.NewPassword) < 8 {
		http.Error(w, "nova password deve ter pelo menos 8 caracteres", http.StatusBadRequest)
		return
	}
	u, err := h.users.GetByID(r.Context(), p.ID)
	if err != nil || u == nil {
		http.Error(w, "utilizador não encontrado", http.StatusNotFound)
		return
	}
	if !auth.CheckPassword(u.PasswordHash, req.CurrentPassword) {
		http.Error(w, "password atual incorreta", http.StatusUnauthorized)
		return
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		http.Error(w, "erro ao atualizar password", http.StatusInternalServerError)
		return
	}
	if err := h.users.UpdatePassword(r.Context(), p.ID, hash); err != nil {
		http.Error(w, "erro ao atualizar password", http.StatusInternalServerError)
		return
	}
	if h.refresh != nil {
		_ = h.refresh.RevokeAllForUser(r.Context(), p.ID)
	}
	h.logAuth(r, &p.ID, p.Email, "password_change", "refresh tokens revogados")
	w.WriteHeader(http.StatusNoContent)
}

func toUserResponse(u postgres.User) dto.UserResponse {
	resp := dto.UserResponse{
		ID: u.ID, Email: u.Email, FullName: u.FullName, Role: u.Role,
		Active: u.Active, CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
	}
	if u.LastLoginAt.Valid {
		s := u.LastLoginAt.Time.UTC().Format(time.RFC3339)
		resp.LastLogin = &s
	}
	return resp
}
