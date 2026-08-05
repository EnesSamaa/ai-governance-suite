package contextfileparser

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type NodeKind string

const (
	Heading   NodeKind = "heading"
	Paragraph NodeKind = "paragraph"
	ListItem  NodeKind = "list_item"
	CodeBlock NodeKind = "code_block"
)

type Node struct {
	Kind  NodeKind
	Level int
	Text  string
}
type Rule struct {
	Source      string
	Section     string
	Instruction string
}
type Document struct {
	Nodes []Node
	Rules []Rule
}

func ParseFile(path string) (Document, error) {
	base := strings.ToUpper(filepath.Base(path))
	if base != "CLAUDE.MD" && base != "SKILL.MD" && base != "AGENTS.MD" {
		return Document{}, errors.New("unsupported context filename")
	}
	file, err := os.Open(path)
	if err != nil {
		return Document{}, err
	}
	defer file.Close()
	return Parse(base, file)
}

func Parse(source string, input interface{ Read([]byte) (int, error) }) (Document, error) {
	var document Document
	section := ""
	fenced := false
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "```") {
			fenced = !fenced
			document.Nodes = append(document.Nodes, Node{Kind: CodeBlock, Text: line})
			continue
		}
		if fenced {
			document.Nodes = append(document.Nodes, Node{Kind: CodeBlock, Text: line})
			continue
		}
		if level := headingLevel(line); level > 0 {
			section = strings.TrimSpace(line[level:])
			document.Nodes = append(document.Nodes, Node{Kind: Heading, Level: level, Text: section})
			continue
		}
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			text := strings.TrimSpace(line[2:])
			document.Nodes = append(document.Nodes, Node{Kind: ListItem, Text: text})
			if isInstructionSection(section) {
				document.Rules = append(document.Rules, Rule{Source: source, Section: section, Instruction: text})
			}
			continue
		}
		if line != "" {
			document.Nodes = append(document.Nodes, Node{Kind: Paragraph, Text: line})
			if isInstructionSection(section) {
				document.Rules = append(document.Rules, Rule{Source: source, Section: section, Instruction: line})
			}
		}
	}
	return document, scanner.Err()
}

func headingLevel(line string) int {
	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	if level > 0 && level < len(line) && line[level] == ' ' {
		return level
	}
	return 0
}
func isInstructionSection(section string) bool {
	lower := strings.ToLower(section)
	return strings.Contains(lower, "rule") || strings.Contains(lower, "instruction") || strings.Contains(lower, "system")
}
