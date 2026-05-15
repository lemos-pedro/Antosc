package security

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

type AccessClaims struct {
	Sub  string `json:"sub"`
	Role string `json:"role"`
	Iat  int64  `json:"iat"`
	Exp  int64  `json:"exp"`
}

func SignAccessToken(secret, subject, role string, ttl time.Duration) (string, time.Time, error) {
	if strings.TrimSpace(secret) == "" {
		return "", time.Time{}, errors.New("token secret is required")
	}
	if strings.TrimSpace(subject) == "" {
		return "", time.Time{}, errors.New("subject is required")
	}
	if ttl <= 0 {
		return "", time.Time{}, errors.New("token ttl must be > 0")
	}
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)
	claims := AccessClaims{
		Sub:  subject,
		Role: role,
		Iat:  now.Unix(),
		Exp:  expiresAt.Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, err
	}
	encPayload := base64.RawURLEncoding.EncodeToString(payload)
	sig := signToken(secret, encPayload)
	token := fmt.Sprintf("%s.%s", encPayload, sig)
	return token, expiresAt, nil
}

func VerifyAccessToken(secret, token string) (*AccessClaims, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, errors.New("token secret is required")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid token format")
	}
	payloadEnc := parts[0]
	sig := parts[1]
	if !verifyToken(secret, payloadEnc, sig) {
		return nil, errors.New("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadEnc)
	if err != nil {
		return nil, errors.New("invalid token payload")
	}
	var claims AccessClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, errors.New("invalid token payload")
	}
	now := time.Now().UTC().Unix()
	if claims.Exp <= now {
		return nil, errors.New("token expired")
	}
	if strings.TrimSpace(claims.Sub) == "" {
		return nil, errors.New("invalid token subject")
	}
	return &claims, nil
}

func signToken(secret, payload string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func verifyToken(secret, payload, signature string) bool {
	want := signToken(secret, payload)
	return hmac.Equal([]byte(want), []byte(signature))
}
