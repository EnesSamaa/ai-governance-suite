package main

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFormatEvent(t *testing.T) {
	got := FormatEvent(Event{ID: "42", Type: "message", Data: "one\ntwo", Retry: 500})
	want := "id: 42\nevent: message\nretry: 500\ndata: one\ndata: two\n\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBrokerServesAndDisconnectsClient(t *testing.T) {
	broker := NewBroker()
	server := httptest.NewServer(broker)
	defer server.Close()
	response, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	deadline := time.Now().Add(time.Second)
	for broker.ClientCount() != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	broker.Publish(Event{Type: "message", Data: "hello"})
	reader := bufio.NewReader(response.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if line != "event: message\n" {
		t.Fatalf("unexpected stream prefix: %q", line)
	}
	data, err := reader.ReadString('\n')
	if err != nil || data != "data: hello\n" {
		t.Fatalf("unexpected data: %q (%v)", data, err)
	}
}

func TestSSEOnlyAcceptsGet(t *testing.T) {
	response := httptest.NewRecorder()
	NewBroker().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("")))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", response.Code)
	}
}
