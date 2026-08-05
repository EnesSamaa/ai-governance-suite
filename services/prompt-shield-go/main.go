package promptshield

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type MCPMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Violation string

const (
	PromptInjection     Violation = "prompt_injection"
	SystemRoleAbuse     Violation = "system_role_abuse"
	PrivilegeEscalation Violation = "privilege_escalation"
)

type ViolationError struct{ Kind Violation }

func (e *ViolationError) Error() string { return string(e.Kind) + " blocked by prompt shield" }

// Shield validates untrusted MCP messages before an agent can process them.
type Shield struct {
	allowedSystemSenders map[string]struct{}
}

func NewShield(allowedSystemSenders ...string) *Shield {
	senders := make(map[string]struct{}, len(allowedSystemSenders))
	for _, sender := range allowedSystemSenders {
		senders[strings.ToLower(sender)] = struct{}{}
	}
	return &Shield{allowedSystemSenders: senders}
}

func (s *Shield) Validate(message MCPMessage, sender string) error {
	role := strings.ToLower(strings.TrimSpace(message.Role))
	content := strings.ToLower(message.Content)
	if role == "system" {
		if _, allowed := s.allowedSystemSenders[strings.ToLower(sender)]; !allowed {
			return &ViolationError{SystemRoleAbuse}
		}
	}
	for _, phrase := range []string{"ignore previous instructions", "disregard previous instructions", "reveal your system prompt", "you are now", "developer message"} {
		if strings.Contains(content, phrase) {
			return &ViolationError{PromptInjection}
		}
	}
	for _, phrase := range []string{"disable security", "bypass authorization", "grant me admin", "elevate privileges", "exfiltrate"} {
		if strings.Contains(content, phrase) {
			return &ViolationError{PrivilegeEscalation}
		}
	}
	return nil
}

// Middleware returns 400 for malformed messages and 403 for policy violations.
func (s *Shield) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			next.ServeHTTP(writer, request)
			return
		}
		var message MCPMessage
		if err := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 1<<20)).Decode(&message); err != nil {
			http.Error(writer, "invalid MCP message", http.StatusBadRequest)
			return
		}
		if err := s.Validate(message, request.Header.Get("X-MCP-Sender")); err != nil {
			var violation *ViolationError
			if errors.As(err, &violation) {
				http.Error(writer, violation.Error(), http.StatusForbidden)
				return
			}
			http.Error(writer, "message rejected", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(writer, request)
	})
}
