package dbschemaintrospect

import (
	"sort"
	"strings"
)

type Column struct {
	Name      string
	Type      string
	Nullable  bool
	Sensitive bool
}
type ForeignKey struct {
	Column           string
	ReferencesTable  string
	ReferencesColumn string
}
type Table struct {
	Name        string
	Columns     []Column
	PrimaryKey  []string
	ForeignKeys []ForeignKey
	Constraints []string
}
type Catalog interface{ Tables() ([]Table, error) }
type Schema struct {
	Tables []Table `json:"tables"`
}

func Inspect(catalog Catalog) (Schema, error) {
	tables, err := catalog.Tables()
	if err != nil {
		return Schema{}, err
	}
	sanitized := make([]Table, 0, len(tables))
	for _, table := range tables {
		sanitized = append(sanitized, sanitize(table))
	}
	sort.Slice(sanitized, func(i, j int) bool { return sanitized[i].Name < sanitized[j].Name })
	return Schema{Tables: sanitized}, nil
}
func sanitize(table Table) Table {
	columns := make([]Column, 0, len(table.Columns))
	for _, column := range table.Columns {
		if column.Sensitive || sensitiveName(column.Name) {
			columns = append(columns, Column{Name: column.Name, Type: "REDACTED", Sensitive: true})
		} else {
			columns = append(columns, column)
		}
	}
	table.Columns = columns
	return table
}
func sensitiveName(name string) bool {
	normalized := strings.ToLower(name)
	for _, word := range []string{"password", "secret", "token", "api_key", "ssn", "tckn", "credit_card", "email"} {
		if strings.Contains(normalized, word) {
			return true
		}
	}
	return false
}
