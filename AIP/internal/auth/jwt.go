package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("token inválido")
	ErrExpiredToken = errors.New("token expirado")
)

// Claims mínimos no JWT (HS256, implementação própria sem dependência extra).
type Claims struct {
	Sub   string `json:"sub"`   // user id
	Email string `json:"email"`
	Role  string `json:"role"`
	Name  string `json:"name"`
	Exp   int64  `json:"exp"`
	Iat   int64  `json:"iat"`
}

type TokenService struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenService(secret string, ttlHours int) *TokenService {
	if ttlHours <= 0 {
		ttlHours = 24
	}
	return &TokenService{
		secret: []byte(secret),
		ttl:    time.Duration(ttlHours) * time.Hour,
	}
}

func (s *TokenService) Issue(userID, email, role, name string) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(s.ttl)
	claims := Claims{
		Sub:   userID,
		Email: email,
		Role:  role,
		Name:  name,
		Exp:   exp.Unix(),
		Iat:   now.Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, err
	}
	head := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body := base64.RawURLEncoding.EncodeToString(payload)
	sig := s.sign(head + "." + body)
	return head + "." + body + "." + sig, exp, nil
}

func (s *TokenService) Parse(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}
	expected := s.sign(parts[0] + "." + parts[1])
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, ErrInvalidToken
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var c Claims
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, ErrInvalidToken
	}
	if time.Now().Unix() > c.Exp {
		return nil, ErrExpiredToken
	}
	if c.Sub == "" || c.Role == "" {
		return nil, ErrInvalidToken
	}
	return &c, nil
}

func (s *TokenService) sign(msg string) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(msg))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// ValidRoles lista roles aceites (alinhadas com prompts/personas + admin).
func ValidRoles() []string {
	return []string{
		"admin",
		"om",
		"engenharia",
		"controller",
		"financeiro",
		"diretor_tecnico",
		"ceo",
		"conselho_administracao",
	}
}

func IsValidRole(role string) bool {
	for _, r := range ValidRoles() {
		if r == role {
			return true
		}
	}
	return false
}

func IsAdmin(role string) bool {
	return role == "admin"
}

// ErrWeakSecret para bootstrap
var ErrWeakSecret = fmt.Errorf("AIP_JWT_SECRET demasiado curto (mínimo 32 caracteres)")
