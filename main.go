package main

import (
	"fmt"
	"golang.org/x/net/html"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/bmaupin/go-epub"
)

func fetchPage(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; RanobeFetcher/1.0)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bodyBytes), nil
}

func parseTitle(doc *html.Node) string {
	titleNode := htmlquery.FindOne(doc, "/html/body/div[2]/div[5]/div[1]/div[3]/h1")
	if titleNode == nil {
		return ""
	}
	return strings.TrimSpace(htmlquery.InnerText(titleNode))
}

func parseContent(doc *html.Node) (string, error) {
	divNode := htmlquery.FindOne(doc, "/html/body/div[2]/div[5]/div[1]")
	if divNode == nil {
		return "", fmt.Errorf("div not found")
	}

	pNodes := htmlquery.Find(divNode, ".//p")
	if len(pNodes) == 0 {
		return "", nil
	}

	var parts []string
	for _, p := range pNodes {
		htmlP := htmlquery.OutputHTML(p, true)
		trimmed := strings.TrimSpace(htmlP)
		if trimmed == "" || trimmed == "<p></p>" {
			continue
		}
		parts = append(parts, htmlP)
	}

	return strings.Join(parts, "\n"), nil
}

func main() {
	e := epub.NewEpub("Моя ранобэ книга")
	e.SetAuthor("Автор")

	for i := 215; i < 483; i++ {
		url := fmt.Sprintf("https://ranobehub.org/ranobe/510/2/%d", i)

		log.Printf("Fetching %s\n", url)

		htmlStr, err := fetchPage(url)
		if err != nil {
			log.Printf("Error fetching %s: %v\n", url, err)
			continue
		}

		doc, err := htmlquery.Parse(strings.NewReader(htmlStr))
		if err != nil {
			log.Printf("Parse error on %s: %v\n", url, err)
			continue
		}

		title := parseTitle(doc)
		if title == "" {
			title = fmt.Sprintf("Глава %d", i+1)
		}

		contentHTML, err := parseContent(doc)
		if err != nil {
			log.Printf("Parse content error on %s: %v\n", url, err)
			continue
		}
		if strings.TrimSpace(contentHTML) == "" {
			log.Printf("No content found on %s — пропускаю\n", url)
			continue
		}

		fullHTML := fmt.Sprintf("<h1>%s</h1>\n%s", html.EscapeString(title), contentHTML)

		_, err = e.AddSection(fullHTML, title, "", "")
		if err != nil {
			log.Printf("Error adding section for %s: %v\n", url, err)
			continue
		}
	}

	outFile := "Повелитель тайн Том 2.epub"
	if err := e.Write(outFile); err != nil {
		log.Fatalf("Error writing epub: %v\n", err)
	}

	log.Printf("EPUB saved to %s\n", outFile)
}
