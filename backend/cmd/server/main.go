// Command server 启动 Lingua Buddy HTTP API。
package main

import (
	"log"
	"net/http"

	"lingua-buddy/internal/app"
	"lingua-buddy/internal/config"
	"lingua-buddy/internal/platform/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	db, err := database.Open(cfg.DB.DSN(), cfg.AppEnv == "development")
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	r := app.New(cfg, db)

	addr := ":" + cfg.HTTPPort
	log.Printf("Lingua Buddy API 启动于 %s (env=%s)", addr, cfg.AppEnv)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
