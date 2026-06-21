// Package app 是组合根：装配配置、数据库、各模块并返回 Gin 引擎。
package app

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"lingua-buddy/internal/ai"
	"lingua-buddy/internal/article"
	"lingua-buddy/internal/asr"
	"lingua-buddy/internal/auth"
	"lingua-buddy/internal/config"
	"lingua-buddy/internal/conversation"
	"lingua-buddy/internal/dictionary"
	"lingua-buddy/internal/essay"
	"lingua-buddy/internal/grammar"
	"lingua-buddy/internal/history"
	"lingua-buddy/internal/httpx"
	"lingua-buddy/internal/lexicon"
	"lingua-buddy/internal/middleware"
	"lingua-buddy/internal/sentence"
	"lingua-buddy/internal/speech"
	"lingua-buddy/internal/storage"
	"lingua-buddy/internal/training"
	"lingua-buddy/internal/trainrec"
	"lingua-buddy/internal/translation"
	"lingua-buddy/internal/user"
	"lingua-buddy/internal/worddistractor"
	"lingua-buddy/internal/wordlearning"
	"lingua-buddy/internal/wordnote"
)

// New 根据配置与数据库装配并返回 Gin 引擎。
func New(cfg config.Config, db *gorm.DB) *gin.Engine {
	if cfg.AppEnv != "development" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.CORS())

	jwtManager := auth.NewJWTManager(cfg.JWTAccessSecret, 0)

	// ===== 依赖装配 =====
	userRepo := user.NewRepository(db)
	userSvc := user.NewService(userRepo)

	registry := auth.NewRegistry(auth.NewUsernamePasswordStrategy(userRepo))
	authSvc := auth.NewService(userRepo, registry, jwtManager)

	// AI / ASR 与依赖它们的模块。
	aiProvider := ai.NewProvider(cfg.AI)
	userLevel := user.NewLevelLookup(userRepo)

	lexRepo := lexicon.NewRepository(db)
	dictSvc := dictionary.NewService(lexRepo, dictionary.NewHistoryRepository(db), aiProvider, userLevel)

	distractorSvc := worddistractor.NewService(lexRepo)
	tokenManager := wordlearning.NewTokenManager(cfg.QuestionTokenSecret)
	wlRepo := wordlearning.NewRepository(db)
	wlSvc := wordlearning.NewService(wlRepo, lexRepo, distractorSvc, tokenManager)
	asrProvider := asr.NewProvider(cfg.ASR)
	histRepo := history.NewRepository(db)
	translationSvc := translation.NewService(aiProvider, histRepo, userLevel)
	grammarSvc := grammar.NewService(aiProvider, histRepo, userLevel)
	sentenceSvc := sentence.NewService(db)
	wordnoteSvc := wordnote.NewService(db)
	ossStore, ossErr := storage.NewOSS(cfg.Upload)
	if ossErr != nil {
		log.Printf("OSS 初始化失败，语音将回退本地存储: %v", ossErr)
	}
	speechSvc := speech.NewService(db, asrProvider, histRepo, ossStore, cfg.Upload.Dir)
	articleRepo := article.NewRepository(db)
	recRepo := trainrec.NewRepository(db)
	conversationSvc := conversation.NewService(db, aiProvider, userLevel)
	essaySvc := essay.NewService(aiProvider, recRepo, histRepo, userLevel)
	trainingSvc := training.NewService(db, aiProvider, recRepo, userLevel)

	// ===== 路由 =====
	v1 := r.Group("/api/v1")
	v1.GET("/health", func(c *gin.Context) {
		httpx.OK(c, gin.H{"status": "ok", "env": cfg.AppEnv})
	})

	// 公开路由
	auth.NewHandler(authSvc).Register(v1)

	// 需要登录的路由
	authed := v1.Group("")
	authed.Use(middleware.AuthRequired(jwtManager))
	user.NewHandler(userSvc).Register(authed)
	dictionary.NewHandler(dictSvc).Register(authed)
	wordlearning.NewHandler(wlSvc, wlRepo).Register(authed)
	translation.NewHandler(translationSvc).Register(authed)
	grammar.NewHandler(grammarSvc).Register(authed)
	sentence.NewHandler(sentenceSvc).Register(authed)
	wordnote.NewHandler(wordnoteSvc).Register(authed)
	speech.NewHandler(speechSvc).Register(authed)
	history.NewHandler(histRepo).Register(authed)
	article.NewHandler(articleRepo).Register(authed)
	conversation.NewHandler(conversationSvc).Register(authed)
	essay.NewHandler(essaySvc).Register(authed)
	training.NewHandler(trainingSvc).Register(authed)

	return r
}
