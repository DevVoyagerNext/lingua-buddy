// Command migrate 创建/更新业务表（不触碰只读 ecdict）。
package main

import (
	"log"

	"lingua-buddy/internal/config"
	"lingua-buddy/internal/models"
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

	log.Println("开始迁移业务表...")
	if err := db.AutoMigrate(models.BusinessModels()...); err != nil {
		log.Fatalf("迁移失败: %v", err)
	}
	log.Println("迁移完成：15 张业务表已就绪（ecdict 未改动）。")
}
