package contextfileparser

import (
	"strings"
	"testing"
)

func TestParseExtractsRulesAndAstNodes(t *testing.T) {
	document, err := Parse("AGENTS.md", strings.NewReader("# Project\ntext\n## System Instructions\n- Never expose secrets\n- Use tests\n```sh\necho ignored\n```"))
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Nodes) != 8 {
		t.Fatalf("nodes=%d", len(document.Nodes))
	}
	if len(document.Rules) != 2 || document.Rules[0].Instruction != "Never expose secrets" {
		t.Fatalf("rules=%+v", document.Rules)
	}
}

func TestParseRejectsUnsupportedContextFile(t *testing.T) {
	if _, err := ParseFile("README.md"); err == nil {
		t.Fatal("unsupported file accepted")
	}
}
