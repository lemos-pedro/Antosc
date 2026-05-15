package security

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

const (
	passwordHashVersion    = "v1"
	passwordHashIterations = 120000
	passwordSaltLen        = 16
)

func HashPassword(plain string) (string, error) {
	if strings.TrimSpace(plain) == "" {
		return "", errors.New("password is required")
	}
	salt := make([]byte, passwordSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	h := derivePasswordHash([]byte(plain), salt, passwordHashIterations)
	return fmt.Sprintf(
		"%s$%s$%s",
		passwordHashVersion,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(h),
	), nil
}

func VerifyPassword(stored, plain string) bool {
	parts := strings.Split(stored, "$")
	if len(parts) != 3 || parts[0] != passwordHashVersion {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	got := derivePasswordHash([]byte(plain), salt, passwordHashIterations)
	return subtle.ConstantTimeCompare(got, want) == 1
}

func derivePasswordHash(password, salt []byte, iterations int) []byte {
	state := make([]byte, 0, len(password)+len(salt))
	state = append(state, password...)
	state = append(state, salt...)
	sum := sha256.Sum256(state)
	out := sum[:]
	for i := 1; i < iterations; i++ {
		buf := make([]byte, 0, len(out)+len(password)+len(salt))
		buf = append(buf, out...)
		buf = append(buf, password...)
		buf = append(buf, salt...)
		next := sha256.Sum256(buf)
		out = next[:]
	}
	return out
}
