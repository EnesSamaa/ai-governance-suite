package schemahashverifier

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
)

type Fingerprint string
type Change struct {
	Name     string
	Expected Fingerprint
	Actual   Fingerprint
}
type Registry struct{ baselines map[string]Fingerprint }

func NewRegistry() *Registry { return &Registry{baselines: make(map[string]Fingerprint)} }

// CanonicalHash produces stable SHA-256 output for semantically equivalent JSON objects.
func CanonicalHash(schema []byte) (Fingerprint, error) {
	var value any
	if err := json.Unmarshal(schema, &value); err != nil {
		return "", err
	}
	canonical, err := json.Marshal(normalize(value))
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return Fingerprint(hex.EncodeToString(sum[:])), nil
}

func (r *Registry) Pin(name string, schema []byte) (Fingerprint, error) {
	if name == "" {
		return "", errors.New("schema name is required")
	}
	hash, err := CanonicalHash(schema)
	if err != nil {
		return "", err
	}
	r.baselines[name] = hash
	return hash, nil
}

func (r *Registry) Verify(name string, schema []byte) (*Change, error) {
	expected, found := r.baselines[name]
	if !found {
		return nil, errors.New("schema has no pinned baseline")
	}
	actual, err := CanonicalHash(schema)
	if err != nil {
		return nil, err
	}
	if actual == expected {
		return nil, nil
	}
	return &Change{Name: name, Expected: expected, Actual: actual}, nil
}

func normalize(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		ordered := make(map[string]any, len(typed))
		for _, key := range keys {
			ordered[key] = normalize(typed[key])
		}
		return ordered
	case []any:
		for index := range typed {
			typed[index] = normalize(typed[index])
		}
		return typed
	default:
		return value
	}
}
