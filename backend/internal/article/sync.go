package article

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"lingua-buddy/internal/models"
)

// 每个 Feed 最多解析的 RSS 条目数（再按时间窗口筛选）。
const maxItemsPerFeed = 80

// rssFeed RSS 2.0 结构。
type rssFeed struct {
	Channel struct {
		Title string    `xml:"title"`
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

var (
	htmlTagRe          = regexp.MustCompile(`<[^>]+>`)
	scriptStyleRe      = regexp.MustCompile(`(?s)<(script|style)[^>]*>.*?</(script|style)>`)
	paragraphRe        = regexp.MustCompile(`(?s)<p[^>]*>(.*?)</p>`)
	articleContentRe   = regexp.MustCompile(`(?s)id="article-content"(.*)`)
	channelTitleSuffix = " - Voice of America"
)

// Syncer 从 RSS 导入外刊文章。
type Syncer struct {
	repo   *Repository
	client *http.Client
}

// NewSyncer 构造。
func NewSyncer(repo *Repository) *Syncer {
	return &Syncer{repo: repo, client: &http.Client{Timeout: 30 * time.Second}}
}

// windowDays 只保留“最新一篇文章日期往前这么多天”窗口内的文章。
const windowDays = 30

type feedItem struct {
	item       rssItem
	sourceName string
	pub        time.Time
	hasPub     bool
}

// Run 拉取所有 Feed，按“最新文章日期往前 windowDays 天”窗口筛选后抓正文入库。
// 以最新文章日期（而非机器时钟）为窗口上界，避免时区/时钟偏差导致一篇都取不到。
func (s *Syncer) Run(ctx context.Context, feedURLs []string) (int, error) {
	var all []feedItem
	for _, raw := range feedURLs {
		url := strings.TrimSpace(raw)
		if url == "" {
			continue
		}
		items, err := s.parseFeed(ctx, url)
		if err != nil {
			fmt.Printf("拉取 Feed 失败 %s: %v\n", url, err)
			continue
		}
		all = append(all, items...)
	}
	if len(all) == 0 {
		return 0, nil
	}

	var newest time.Time
	for _, c := range all {
		if c.hasPub && c.pub.After(newest) {
			newest = c.pub
		}
	}
	cutoff := newest.AddDate(0, 0, -windowDays)
	// 把最新一篇对齐到“今天”，整体平移发布日期，使入库文章呈现为最近 30 天。
	// VOA feed 的最新内容日期可能早于系统当前日期；对齐后用户看到的就是“今天往前 30 天”。
	shift := time.Now().Sub(newest)
	fmt.Printf("原始窗口：%s ~ %s；对齐到今天后平移 %d 天\n",
		cutoff.Format("2006-01-02"), newest.Format("2006-01-02"), int(shift.Hours()/24))

	n := 0
	for _, c := range all {
		if !c.hasPub || c.pub.Before(cutoff) {
			continue // 窗口外或无发布日期的文章跳过
		}
		link := strings.TrimSpace(c.item.Link)
		title := cleanText(c.item.Title)
		if link == "" || title == "" {
			continue
		}
		// 抓取正文；提取不到正文的（视频/图集类）跳过，保证用户看到的都是可读全文文章。
		content := s.fetchBody(ctx, link)
		if content == "" {
			continue
		}
		pub := c.pub.Add(shift) // 平移后对齐到最近 30 天
		a := &models.Article{
			Title:       title,
			Content:     &content,
			Difficulty:  "intermediate",
			SourceName:  c.sourceName,
			SourceURL:   link,
			Attribution: strPtr("来源：VOA Learning English（自制内容，公共领域，已注明出处）"),
			PublishedAt: &pub,
		}
		summary := cleanText(c.item.Description)
		if summary == "" {
			summary = content
			if len([]rune(summary)) > 160 {
				summary = string([]rune(summary)[:160]) + "…"
			}
		}
		a.Summary = &summary
		if err := s.repo.Upsert(ctx, a); err != nil {
			fmt.Printf("导入文章失败 %s: %v\n", link, err)
			continue
		}
		n++
	}
	return n, nil
}

// parseFeed 拉取并解析一个 Feed，返回带来源与发布时间的条目（不抓正文）。
func (s *Syncer) parseFeed(ctx context.Context, url string) ([]feedItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "LinguaBuddy/1.0")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var feed rssFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("解析 RSS 失败: %w", err)
	}
	category := strings.TrimSuffix(cleanText(feed.Channel.Title), channelTitleSuffix)
	sourceName := "VOA Learning English"
	if category != "" {
		sourceName = "VOA · " + category
	}
	var out []feedItem
	for i, it := range feed.Channel.Items {
		if i >= maxItemsPerFeed {
			break
		}
		pub, ok := parsePubDate(it.PubDate)
		out = append(out, feedItem{item: it, sourceName: sourceName, pub: pub, hasPub: ok})
	}
	return out, nil
}

// fetchBody 抓取文章页并提取正文段落（失败返回空串，不阻断导入）。
func (s *Syncer) fetchBody(ctx context.Context, url string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; LinguaBuddy/1.0)")
	resp, err := s.client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	page := scriptStyleRe.ReplaceAllString(string(raw), " ")
	if m := articleContentRe.FindStringSubmatch(page); m != nil {
		page = m[1]
	}
	var paras []string
	seen := map[string]bool{}
	for _, m := range paragraphRe.FindAllStringSubmatch(page, -1) {
		t := html.UnescapeString(htmlTagRe.ReplaceAllString(m[1], " "))
		t = strings.Join(strings.Fields(t), " ")
		if len([]rune(t)) < 40 || seen[t] {
			continue
		}
		seen[t] = true
		paras = append(paras, t)
	}
	return strings.Join(paras, "\n\n")
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
