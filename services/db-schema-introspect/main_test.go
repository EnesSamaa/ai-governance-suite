package dbschemaintrospect

import "testing"

type fakeCatalog struct{ tables []Table }

func (f fakeCatalog) Tables() ([]Table, error) { return f.tables, nil }
func TestInspectFiltersSensitiveColumnsAndKeepsRelations(t *testing.T) {
	schema, err := Inspect(fakeCatalog{tables: []Table{{Name: "users", Columns: []Column{{Name: "id", Type: "uuid"}, {Name: "email", Type: "text"}, {Name: "display_name", Type: "text"}}, PrimaryKey: []string{"id"}, ForeignKeys: []ForeignKey{{Column: "org_id", ReferencesTable: "orgs", ReferencesColumn: "id"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	table := schema.Tables[0]
	if table.Columns[1].Type != "REDACTED" || !table.Columns[1].Sensitive {
		t.Fatalf("email column not sanitized: %+v", table.Columns[1])
	}
	if table.ForeignKeys[0].ReferencesTable != "orgs" {
		t.Fatal("foreign key lost")
	}
}
func TestSensitiveNameDetection(t *testing.T) {
	if !sensitiveName("encrypted_api_key") || sensitiveName("created_at") {
		t.Fatal("sensitive name classification incorrect")
	}
}
