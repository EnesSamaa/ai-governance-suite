package firecrawlmcpg

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type Page struct {
	URL      string         `json:"url"`
	Markdown string         `json:"markdown"`
	JSON     map[string]any `json:"json"`
}
type Fetcher interface {
	Fetch(url string) (string, error)
}
type HTTPFetcher struct{ Client *http.Client }

func (f HTTPFetcher) Fetch(url string) (string, error) {
	response, err := f.Client.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 5<<20))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

type Service struct{ Fetcher Fetcher }

func (s Service) Crawl(url string) (Page, error) {
	html, err := s.Fetcher.Fetch(url)
	if err != nil {
		return Page{}, err
	}
	markdown := HTMLToMarkdown(html)
	return Page{URL: url, Markdown: markdown, JSON: map[string]any{"content": markdown}}, nil
}
func (s Service) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(writer, "GET required", http.StatusMethodNotAllowed)
		return
	}
	page, err := s.Crawl(request.URL.Query().Get("url"))
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadGateway)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(page)
}
func HTMLToMarkdown(html string) string {
	var output strings.Builder
	for len(html) > 0 {
		open := strings.IndexByte(html, '<')
		if open < 0 {
			writeText(&output, html)
			break
		}
		writeText(&output, html[:open])
		html = html[open:]
		close := strings.IndexByte(html, '>')
		if close < 0 {
			break
		}
		rawTag := strings.TrimSpace(html[1:close])
		closingTag := strings.HasPrefix(rawTag, "/")
		parts := strings.Fields(strings.Trim(rawTag, " /"))
		tag := ""
		if len(parts) > 0 {
			tag = strings.ToLower(parts[0])
			if closingTag {
				tag = "/" + tag
			}
		}
		switch tag {
		case "h1":
			output.WriteString("# ")
		case "h2":
			output.WriteString("## ")
		case "h3":
			output.WriteString("### ")
		case "li":
			output.WriteString("- ")
		case "/h1", "/h2", "/h3", "/p", "/div":
			output.WriteString("\n\n")
		case "/li", "br":
			output.WriteByte('\n')
		}
		html = html[close+1:]
	}
	lines := strings.Split(output.String(), "\n")
	for index, line := range lines {
		lines[index] = strings.TrimSpace(line)
	}
	return strings.TrimSpace(strings.Join(compactEmptyLines(lines), "\n"))
}

func cleanText(value string) string { return strings.Join(strings.Fields(value), " ") }
func writeText(output *strings.Builder, value string) {
	text := cleanText(value)
	if text == "" {
		return
	}
	if output.Len() > 0 {
		current := output.String()
		last := current[len(current)-1]
		if last != ' ' && last != '\n' && last != '#' && last != '-' {
			output.WriteByte(' ')
		}
	}
	output.WriteString(text)
}
func compactEmptyLines(lines []string) []string {
	output := make([]string, 0, len(lines))
	empty := false
	for _, line := range lines {
		if line == "" {
			if !empty {
				output = append(output, line)
			}
			empty = true
			continue
		}
		output = append(output, line)
		empty = false
	}
	return output
}
