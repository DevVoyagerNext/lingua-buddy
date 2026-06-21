// Command article-sync 从配置的 RSS Feed 导入外刊文章（建议由系统计划任务每日触发）。
package main

import (
	"context"
	"log"
	"strings"
	"time"

	"lingua-buddy/internal/article"
	"lingua-buddy/internal/config"
	"lingua-buddy/internal/platform/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	db, err := database.Open(cfg.DB.DSN(), false)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	feeds := strings.Split(cfg.Article.FeedURLs, ",")
	if len(feeds) == 0 || strings.TrimSpace(cfg.Article.FeedURLs) == "" {
		log.Fatal("未配置 ARTICLE_FEED_URLS")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	syncer := article.NewSyncer(article.NewRepository(db))
	n, err := syncer.Run(ctx, feeds)
	if err != nil {
		log.Fatalf("同步失败: %v", err)
	}
	log.Printf("外刊同步完成：导入/更新 %d 篇。", n)
}
