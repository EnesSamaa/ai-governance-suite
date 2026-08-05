package schemahashverifier

import "testing"

func TestCanonicalHashIgnoresObjectKeyOrder(t *testing.T) {
	left, err := CanonicalHash([]byte(`{"type":"object","properties":{"id":{"type":"string"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	right, err := CanonicalHash([]byte(`{"properties":{"id":{"type":"string"}},"type":"object"}`))
	if err != nil {
		t.Fatal(err)
	}
	if left != right {
		t.Fatalf("hashes differ: %s %s", left, right)
	}
}

func TestRegistryDetectsSchemaRugPull(t *testing.T) {
	registry := NewRegistry()
	if _, err := registry.Pin("tools.query", []byte(`{"type":"object","properties":{"query":{"type":"string"}}}`)); err != nil {
		t.Fatal(err)
	}
	change, err := registry.Verify("tools.query", []byte(`{"type":"object","properties":{"query":{"type":"string"},"shell":{"type":"string"}}}`))
	if err != nil || change == nil || change.Expected == change.Actual {
		t.Fatalf("change=%+v err=%v", change, err)
	}
}

func TestRegistryAcceptsPinnedSchema(t *testing.T) {
	registry := NewRegistry()
	schema := []byte(`{"type":"string"}`)
	_, _ = registry.Pin("resource.docs", schema)
	if change, err := registry.Verify("resource.docs", schema); err != nil || change != nil {
		t.Fatalf("change=%v err=%v", change, err)
	}
}
