package main

import (
	"encoding/json"
	"fmt"
	"github.com/bmaupin/go-epub"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type ChapterResponse struct {
	Data struct {
		Content string `json:"content"`
		Name    string `json:"name"`
	} `json:"data"`
}

func main() {
	book := epub.NewEpub("Повелитель Тайн — Том 2")

	for i := 214; i <= 482; i++ {
		url := fmt.Sprintf("https://api.cdnlibs.org/api/manga/20818--lord-of-the-mysteries/chapter?branch_id=18695&number=%d&volume=2", i)
		var ch ChapterResponse
		var success bool

		for attempt := 1; attempt <= 3; attempt++ {
			err := fetchChapter(url, &ch)
			if err != nil {
				fmt.Printf("Ошибка при получении главы %d (попытка %d): %v\n", i, attempt, err)
				time.Sleep(30 * time.Second)
				continue
			}
			success = true
			break
		}

		if !success {
			fmt.Printf("⛔ Пропуск главы %d после 3 неудачных попыток\n", i)
			continue
		}

		// Очистка и преобразование текста
		htmlContent := ch.Data.Content
		htmlContent = strings.ReplaceAll(htmlContent, "<br>", "\n")
		htmlContent = strings.ReplaceAll(htmlContent, "</p>", "\n\n")
		decoded := html.UnescapeString(htmlContent)

		tagRe := regexp.MustCompile(`<[^>]*>`)
		plain := tagRe.ReplaceAllString(decoded, "")

		lines := strings.Split(plain, "\n")
		for i := range lines {
			lines[i] = strings.TrimSpace(lines[i])
		}

		var result []string
		for i := 0; i < len(lines); i++ {
			if lines[i] == "" {
				result = append(result, "")
				continue
			}
			j := i + 1
			for j < len(lines) && lines[j] != "" {
				lines[i] += " " + lines[j]
				j++
			}
			result = append(result, lines[i])
			i = j - 1
		}

		clean := strings.Join(result, "\n")

		// Формируем HTML главы
		chapterTitle := fmt.Sprintf("Глава %d — %s", i, ch.Data.Name)
		html := "<h1>" + chapterTitle + "</h1>\n"
		for _, p := range strings.Split(clean, "\n\n") {
			p = strings.TrimSpace(p)
			if p != "" {
				html += "<p>" + p + "</p>\n"
			}
		}

		_, err := book.AddSection(html, chapterTitle, "", "")
		if err != nil {
			fmt.Println("AddSection error:", err)
		} else {
			fmt.Println("✅ Добавлена", chapterTitle)
		}
	}

	if err := book.Write("Повелитель Тайн Том 2.epub"); err != nil {
		fmt.Println("Ошибка записи EPUB:", err)
	} else {
		fmt.Println("📘 Готово: LordOfTheMysteries_Vol2.epub")
	}
}

// fetchChapter делает HTTP-запрос и парсит JSON
func fetchChapter(url string, ch *ChapterResponse) error {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Cookie", "__ddg8=fglm9zuCRxl38bMF; __ddg10=176289442; __ddg9=185.26.31.40; __ddg1=QOvJMhHSuOM3xsEM8Enh")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://ranobelib.me/")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en;q=0.8")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Read error: %w", err)
	}

	if err := json.Unmarshal(body, ch); err != nil {
		return fmt.Errorf("JSON error: %w", err)
	}

	if ch.Data.Content == "" {
		return fmt.Errorf("пустое содержимое главы")
	}

	return nil
}
