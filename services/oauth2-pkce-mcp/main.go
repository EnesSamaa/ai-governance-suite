package oauth2pkcemcp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"
)

func GenerateVerifier() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func ChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func AuthorizationURL(endpoint, clientID, redirectURI, state, verifier string, scopes []string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", errors.New("invalid authorization endpoint")
	}
	query := u.Query()
	query.Set("response_type", "code")
	query.Set("client_id", clientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("state", state)
	query.Set("code_challenge", ChallengeS256(verifier))
	query.Set("code_challenge_method", "S256")
	if len(scopes) > 0 {
		query.Set("scope", strings.Join(scopes, " "))
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}

type Claims struct {
	Subject  string `json:"sub"`
	Issuer   string `json:"iss"`
	Audience string `json:"aud"`
	Expires  int64  `json:"exp"`
}

// TokenValidator validates compact HS256 JWT access tokens issued by a trusted MCP gateway.
type TokenValidator struct {
	secret           []byte
	issuer, audience string
	now              func() time.Time
}

func NewTokenValidator(secret []byte, issuer, audience string) *TokenValidator {
	return &TokenValidator{secret: append([]byte(nil), secret...), issuer: issuer, audience: audience, now: time.Now}
}

func (v *TokenValidator) Validate(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || len(v.secret) == 0 {
		return Claims{}, errors.New("malformed token")
	}
	signed := parts[0] + "." + parts[1]
	want := v.signature(signed)
	got, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(got, want) {
		return Claims{}, errors.New("invalid token signature")
	}
	var header struct {
		Algorithm string `json:"alg"`
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, errors.New("invalid token header")
	}
	if json.Unmarshal(headerBytes, &header) != nil || header.Algorithm != "HS256" {
		return Claims{}, errors.New("unsupported token algorithm")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, errors.New("invalid token payload")
	}
	var claims Claims
	if json.Unmarshal(payload, &claims) != nil {
		return Claims{}, errors.New("invalid token claims")
	}
	if claims.Subject == "" || claims.Issuer != v.issuer || claims.Audience != v.audience || claims.Expires <= v.now().Unix() {
		return Claims{}, errors.New("token claims rejected")
	}
	return claims, nil
}

// Issue is intended for trusted integration tests and gateway-side token issuance.
func (v *TokenValidator) Issue(claims Claims) (string, error) {
	header, err := json.Marshal(struct {
		Algorithm string `json:"alg"`
		Type      string `json:"typ"`
	}{"HS256", "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signed := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	return signed + "." + base64.RawURLEncoding.EncodeToString(v.signature(signed)), nil
}

func (v *TokenValidator) signature(value string) []byte {
	mac := hmac.New(sha256.New, v.secret)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}
