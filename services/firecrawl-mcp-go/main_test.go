package firecrawlmcpg

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeFetcher struct{}

func (fakeFetcher) Fetch(string) (string, error) {
	return "<h1>Title</h1><p>Hello <b>agent</b></p><li>One</li>", nil
}
func TestCrawlProducesAgentReadableContent(t *testing.T) {
	page, err := (Service{Fetcher: fakeFetcher{}}).Crawl("https://example.test")
	if err != nil {
		t.Fatal(err)
	}
	if page.Markdown != "# Title\n\nHello agent\n\n- One" {
		t.Fatalf("markdown=%q", page.Markdown)
	}
}
func TestHandlerReturnsJSON(t *testing.T) {
	response := httptest.NewRecorder()
	(Service{Fetcher: fakeFetcher{}}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/crawl?url=https://example.test", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
}
