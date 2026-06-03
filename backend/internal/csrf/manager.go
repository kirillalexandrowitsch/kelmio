package csrf

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
)

const HeaderName = "X-CSRF-Token"

const nonceSize = 32

var errMissingSession = errors.New("session token is required")

type Manager struct {
	secret []byte
}

func NewManager(secret string) (*Manager, error) {
	if strings.TrimSpace(secret) != "" {
		return &Manager{secret: []byte(secret)}, nil
	}

	generated := make([]byte, 32)
	if _, err := rand.Read(generated); err != nil {
		return nil, err
	}
	return &Manager{secret: generated}, nil
}

func (m *Manager) Generate(sessionToken string) (string, error) {
	if strings.TrimSpace(sessionToken) == "" {
		return "", errMissingSession
	}

	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	encodedNonce := base64.RawURLEncoding.EncodeToString(nonce)
	signature := m.sign(sessionToken, nonce)
	encodedSignature := base64.RawURLEncoding.EncodeToString(signature)
	return encodedNonce + "." + encodedSignature, nil
}

func (m *Manager) Valid(sessionToken string, token string) bool {
	if strings.TrimSpace(sessionToken) == "" || strings.TrimSpace(token) == "" {
		return false
	}

	noncePart, signaturePart, ok := strings.Cut(token, ".")
	if !ok || noncePart == "" || signaturePart == "" {
		return false
	}

	nonce, err := base64.RawURLEncoding.DecodeString(noncePart)
	if err != nil || len(nonce) != nonceSize {
		return false
	}

	signature, err := base64.RawURLEncoding.DecodeString(signaturePart)
	if err != nil {
		return false
	}

	expectedSignature := m.sign(sessionToken, nonce)
	return subtle.ConstantTimeCompare(signature, expectedSignature) == 1
}

func (m *Manager) sign(sessionToken string, nonce []byte) []byte {
	sessionHash := sha256.Sum256([]byte(sessionToken))

	mac := hmac.New(sha256.New, m.secret)
	mac.Write(sessionHash[:])
	mac.Write([]byte{0})
	mac.Write(nonce)
	return mac.Sum(nil)
}
