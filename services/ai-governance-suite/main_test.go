package aigovernancesuite

import (
	"errors"
	"testing"
)

type gateway struct{ err error }

func (g gateway) Authorize(string, string, string) error { return g.err }

type audit struct{ called bool }

func (a *audit) Record(string, string) error { a.called = true; return nil }
func TestSuiteAuditsAuthorizedWork(t *testing.T) {
	sink := &audit{}
	if err := (Suite{gateway{}, sink}).Execute("a", "tool", "x"); err != nil || !sink.called {
		t.Fatal(err, sink.called)
	}
	if err := (Suite{gateway{errors.New("denied")}, sink}).Execute("a", "tool", "x"); err == nil {
		t.Fatal("denied request accepted")
	}
}
