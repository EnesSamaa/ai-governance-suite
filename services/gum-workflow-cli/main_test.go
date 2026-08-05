package gumworkflowcli

import "testing"

type fakeIO struct{ value string }

func (f fakeIO) Read(string) (string, error) { return f.value, nil }
func (f fakeIO) Write(string) error          { return nil }
func TestChooseUsesDefaultAndValidates(t *testing.T) {
	got, err := Choose(fakeIO{""}, Prompt{"Mode", []string{"safe", "fast"}, "safe"})
	if err != nil || got != "safe" {
		t.Fatal(got, err)
	}
	if _, err := Choose(fakeIO{"bad"}, Prompt{"", []string{"safe"}, "safe"}); err == nil {
		t.Fatal("invalid accepted")
	}
}
