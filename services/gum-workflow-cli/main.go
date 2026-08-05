package gumworkflowcli

import "strings"

type Prompt struct {
	Label   string
	Options []string
	Default string
}
type IO interface {
	Read(prompt string) (string, error)
	Write(value string) error
}

func Choose(io IO, prompt Prompt) (string, error) {
	answer, err := io.Read(prompt.Label + " [" + strings.Join(prompt.Options, "/") + "]: ")
	if err != nil {
		return "", err
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		answer = prompt.Default
	}
	for _, option := range prompt.Options {
		if answer == option {
			return answer, nil
		}
	}
	return "", &InvalidChoice{Value: answer}
}

type InvalidChoice struct{ Value string }

func (e *InvalidChoice) Error() string { return "invalid choice: " + e.Value }
