package article

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"lingua-buddy/internal/models"
)

// rssFeed RSS 2.0 结构。
type rssFeed struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Category    string `xml:"category"`
}

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

// Syncer 从 RSS 导入外刊文章。
type Syncer struct {
	repo   *Repository
	client *http.Client
}

// NewSyncer 构造。
func NewSyncer(repo *Repository) *Syncer {
	return &Syncer{repo: repo, client: &http.Client{Timeout: 30 * time.Second}}
}

// Run 同步给定的一组 Feed URL，返回成功导入/更新的条数。
func (s *Syncer) Run(ctx context.Context, feedURLs []string) (int, error) {
	total := 0
	for _, raw := range feedURLs {
		url := strings.TrimSpace(raw)
		if url == "" {
			continue
		}
		n, err := s.syncOne(ctx, url)
		if err != nil {
			// 单个 Feed 失败不影响其它（记录并继续）。
			fmt.Printf("同步 Feed 失败 %s: %v\n", url, err)
			continue
		}
		total += n
	}
	return total, nil
}

func (s *Syncer) syncOne(ctx context.Context, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "LinguaBuddy/1.0")
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	var feed rssFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return 0, fmt.Errorf("解析 RSS 失败: %w", err)
	}

	n := 0
	for _, it := range feed.Channel.Items {
		link := strings.TrimSpace(it.Link)
		title := cleanText(it.Title)
		if link == "" || title == "" {
			continue
		}
		summary := cleanText(it.Description)
		a := &models.Article{
			Title:       title,
			Difficulty:  "intermediate",
			SourceName:  "VOA Learning English",
			SourceURL:   link,
			Attribution: strPtr("Source: VOA Learning English (" + link + ")"),
		}
		if summary != "" {
			a.Summary = &summary
		}
		if t, ok := parsePubDate(it.PubDate); ok {
			a.PublishedAt = &t
		}
		if err := s.repo.Upsert(ctx, a); err != nil {
			fmt.Printf("导入文章失败 %s: %v\n", link, err)
			continue
		}
		n++
	}
	return n, nil
}

func cleanText(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	return strings.TrimSpace(s)
}

func parsePubDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC822} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func strPtr(s string) *string { return &s }
