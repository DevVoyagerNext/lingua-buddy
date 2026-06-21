// Package config 负责从环境变量（含 .env）加载应用配置。
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config 是应用运行所需的全部配置。
type Config struct {
	AppEnv   string
	HTTPPort string

	JWTAccessSecret     string
	QuestionTokenSecret string

	DB    DBConfig
	Redis RedisConfig
	AI    ProviderConfig
	ASR   ProviderConfig
	TTS   ProviderConfig

	Upload  UploadConfig
	Article ArticleConfig
}

// DBConfig MySQL 连接配置。
type DBConfig struct {
	User     string
	Password string
	Addr     string
	Name     string
}

// DSN 生成 go-sql-driver / GORM 使用的连接串。
func (c DBConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		c.User, c.Password, c.Addr, c.Name)
}

// RedisConfig 缓存与限流（MVP 可不启用）。
type RedisConfig struct {
	Enabled  bool
	Addr     string
	Password string
	DB       int
}

// ProviderConfig 适用于 AI / ASR / TTS 等外部能力的通用配置。
type ProviderConfig struct {
	Provider string
	APIBase  string
	APIKey   string
	Model    string
	Voice    string // 仅 TTS 使用
}

// UploadConfig 文件存储配置。语音音频固定走 OSS，与本配置的 Storage 无关。
type UploadConfig struct {
	Storage   string // 非音频文件：local / aliyun_oss
	Dir       string
	OSSBucket string
	OSSRegion string
	OSSEndpt  string
	OSSKey    string
	OSSSecret string
}

// ArticleConfig 外刊导入配置。
type ArticleConfig struct {
	Source   string
	FeedURLs string
	Cron     string
	Timezone string
}

// Load 读取 .env（若存在）后从环境变量装配 Config，并校验必填项。
func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("读取 .env 失败: %w", err)
	}

	cfg := Config{
		AppEnv:              getEnv("APP_ENV", "development"),
		HTTPPort:            getEnv("HTTP_PORT", "8080"),
		JWTAccessSecret:     os.Getenv("JWT_ACCESS_SECRET"),
		QuestionTokenSecret: os.Getenv("QUESTION_TOKEN_SECRET"),
		DB: DBConfig{
			User:     getEnv("DB_USER", "root"),
			Password: os.Getenv("DB_PASSWORD"),
			Addr:     getEnv("DB_ADDR", "127.0.0.1:3306"),
			Name:     getEnv("DB_NAME", "lingua"),
		},
		Redis: RedisConfig{
			Enabled:  getEnv("REDIS_ENABLED", "false") == "true",
			Addr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		AI: ProviderConfig{
			Provider: os.Getenv("AI_PROVIDER"),
			APIBase:  os.Getenv("AI_API_BASE"),
			APIKey:   os.Getenv("AI_API_KEY"),
			Model:    getEnv("AI_MODEL", "qwen-plus"),
		},
		ASR: ProviderConfig{
			Provider: os.Getenv("ASR_PROVIDER"),
			APIBase:  os.Getenv("ASR_API_BASE"),
			APIKey:   os.Getenv("ASR_API_KEY"),
			Model:    getEnv("ASR_MODEL", "paraformer-v2"),
		},
		TTS: ProviderConfig{
			Provider: os.Getenv("TTS_PROVIDER"),
			APIBase:  os.Getenv("TTS_API_BASE"),
			APIKey:   os.Getenv("TTS_API_KEY"),
			Model:    getEnv("TTS_MODEL", "cosyvoice-v2"),
			Voice:    getEnv("TTS_VOICE", "longxiaochun_v2"),
		},
		Upload: UploadConfig{
			Storage:   getEnv("UPLOAD_STORAGE", "local"),
			Dir:       getEnv("UPLOAD_DIR", "./uploads"),
			OSSBucket: os.Getenv("OBJECT_STORAGE_BUCKET"),
			OSSRegion: os.Getenv("OBJECT_STORAGE_REGION"),
			OSSEndpt:  os.Getenv("OBJECT_STORAGE_ENDPOINT"),
			OSSKey:    os.Getenv("OBJECT_STORAGE_ACCESS_KEY"),
			OSSSecret: os.Getenv("OBJECT_STORAGE_SECRET_KEY"),
		},
		Article: ArticleConfig{
			Source:   os.Getenv("ARTICLE_SOURCE"),
			FeedURLs: os.Getenv("ARTICLE_FEED_URLS"),
			Cron:     getEnv("ARTICLE_SYNC_CRON", "0 0 7 * * *"),
			Timezone: getEnv("ARTICLE_SYNC_TIMEZONE", "Asia/Shanghai"),
		},
	}

	if cfg.JWTAccessSecret == "" {
		return Config{}, errors.New("JWT_ACCESS_SECRET 不能为空")
	}
	if cfg.QuestionTokenSecret == "" {
		return Config{}, errors.New("QUESTION_TOKEN_SECRET 不能为空")
	}
	if cfg.DB.User == "" {
		return Config{}, errors.New("DB_USER 不能为空")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
