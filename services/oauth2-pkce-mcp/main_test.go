package oauth2pkcemcp

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPKCEAuthorizationURL(t *testing.T) {
	verifier := "a_very_long_valid_verifier_value_0123456789"
	got, err := AuthorizationURL("https://id.example/authorize", "client", "https://app/callback", "state", verifier, []string{"mcp.read", "mcp.write"})
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(got)
	if u.Query().Get("code_challenge") != ChallengeS256(verifier) || u.Query().Get("code_challenge_method") != "S256" {
		t.Fatalf("bad PKCE query: %s", got)
	}
}

func TestTokenValidation(t *testing.T) {
	validator := NewTokenValidator([]byte("test-secret"), "mcp-gateway", "remote-mcp")
	token, err := validator.Issue(Claims{Subject: "agent-1", Issuer: "mcp-gateway", Audience: "remote-mcp", Expires: time.Now().Add(time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := validator.Validate(token)
	if err != nil || claims.Subject != "agent-1" {
		t.Fatalf("claims=%+v err=%v", claims, err)
	}
	parts := strings.Split(token, ".")
	if parts[2][0] == 'A' {
		parts[2] = "B" + parts[2][1:]
	} else {
		parts[2] = "A" + parts[2][1:]
	}
	if _, err := validator.Validate(strings.Join(parts, ".")); err == nil {
		t.Fatal("tampered token accepted")
	}
}

func TestExpiredTokenIsRejected(t *testing.T) {
	validator := NewTokenValidator([]byte("test-secret"), "issuer", "audience")
	token, _ := validator.Issue(Claims{Subject: "agent", Issuer: "issuer", Audience: "audience", Expires: time.Now().Add(-time.Second).Unix()})
	if _, err := validator.Validate(token); err == nil || !strings.Contains(err.Error(), "claims") {
		t.Fatalf("unexpected error: %v", err)
	}
}
