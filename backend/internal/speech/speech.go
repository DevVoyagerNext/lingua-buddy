// Package speech 提供音频上传识别与语音历史保存。
package speech

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"lingua-buddy/internal/asr"
	"lingua-buddy/internal/history"
	"lingua-buddy/internal/httpx"
	"lingua-buddy/internal/models"
	"lingua-buddy/internal/storage"
)

const maxAudioBytes = 20 << 20 // 20MB

var allowedMime = map[string]bool{
	"audio/webm": true, "audio/wav": true, "audio/x-wav": true,
	"audio/mpeg": true, "audio/mp3": true, "audio/mp4": true, "audio/x-m4a": true, "audio/m4a": true,
}

// Service 语音服务。
type Service struct {
	db      *gorm.DB
	asr     asr.Provider
	history *history.Repository
	store   storage.Provider
	dir     string
}

// NewService 构造。优先用 OSS（Paraformer 需要公网音频 URL）；OSS 未配置时退化为本地存储。
func NewService(db *gorm.DB, provider asr.Provider, hist *history.Repository, store storage.Provider, dir string) *Service {
	return &Service{db: db, asr: provider, history: hist, store: store, dir: dir}
}

// TranscribeResult 识别结果。
type TranscribeResult struct {
	Text        string `json:"text"`
	Language    string `json:"language"`
	AudioFileID uint64 `json:"audio_file_id"`
}

// Transcribe 保存音频并同步识别（不写历史）。
func (s *Service) Transcribe(ctx context.Context, userID uint64, filename, mime string, data []byte, language string) (*TranscribeResult, error) {
	if len(data) == 0 {
		return nil, httpx.NewError(400, "UPLOAD_INVALID", "音频为空")
	}
	if len(data) > maxAudioBytes {
		return nil, httpx.NewError(400, "UPLOAD_INVALID", "音频超过 20MB 限制")
	}
	if mime != "" && !allowedMime[mime] {
		return nil, httpx.NewError(400, "UPLOAD_INVALID", "不支持的音频类型: "+mime)
	}

	ext := filepath.Ext(filename)
	var filePath, audioURL string
	if s.store != nil && s.store.Available() {
		// OSS：上传后取签名 URL 供 Paraformer 拉取。
		key := fmt.Sprintf("audio/%d/%d%s", userID, time.Now().UnixNano(), ext)
		if err := s.store.Save(key, data, mime); err != nil {
			return nil, httpx.ErrInternal("音频上传失败: " + err.Error())
		}
		url, err := s.store.SignedURL(key, time.Hour)
		if err != nil {
			return nil, httpx.ErrInternal("生成音频 URL 失败: " + err.Error())
		}
		filePath, audioURL = key, url
	} else {
		// 本地回退（仅在未配置 OSS 时；此时真实 Paraformer 无法访问，建议配置 OSS）。
		local, err := s.saveLocal(userID, ext, data)
		if err != nil {
			return nil, err
		}
		filePath = local
	}

	af := &models.AudioFile{
		UserID: userID, FilePath: filePath, OriginalName: filename,
		MimeType: mime, FileSize: int64(len(data)),
	}
	if err := s.db.WithContext(ctx).Create(af).Error; err != nil {
		return nil, err
	}

	tr, err := s.asr.Transcribe(ctx, audioURL, language)
	if err != nil {
		log.Printf("ASR 失败 user=%d url=%s: %v", userID, audioURL, err)
		return nil, httpx.NewError(502, "ASR_FAILED", "语音识别失败，请重试")
	}
	return &TranscribeResult{Text: tr.Text, Language: tr.Language, AudioFileID: af.ID}, nil
}

func (s *Service) saveLocal(userID uint64, ext string, data []byte) (string, error) {
	dir := filepath.Join(s.dir, fmt.Sprintf("%d", userID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	full := filepath.Join(dir, fmt.Sprintf("%d%s", time.Now().UnixNano(), ext))
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return "", err
	}
	return full, nil
}

// SaveResult 用户确认后保存语音历史（识别文本 + 可选译文）。
func (s *Service) SaveResult(ctx context.Context, userID, audioFileID uint64, text, translation string) (*models.HistoryRecord, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, httpx.ErrValidation("识别文本不能为空")
	}
	var af models.AudioFile
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", audioFileID, userID).Take(&af).Error; err != nil {
		return nil, httpx.ErrNotFound("音频不存在")
	}
	rec := &models.HistoryRecord{
		UserID: userID, RecordType: history.TypeSpeech,
		InputText: text, ResultText: translation, AudioFileID: &audioFileID,
	}
	if err := s.history.Create(ctx, rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// Handler 暴露语音接口。
type Handler struct{ svc *Service }

// NewHandler 构造。
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register 注册路由。
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/speech/transcribe", h.transcribe)
	rg.POST("/speech/results", h.saveResult)
}

func (h *Handler) transcribe(c *gin.Context) {
	fileHeader, err := c.FormFile("audio")
	if err != nil {
		httpx.Fail(c, httpx.NewError(400, "UPLOAD_INVALID", "缺少 audio 文件"))
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		httpx.Fail(c, httpx.NewError(400, "UPLOAD_INVALID", "无法读取音频"))
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxAudioBytes+1))
	if err != nil {
		httpx.Fail(c, httpx.ErrInternal("读取音频失败"))
		return
	}
	mime := fileHeader.Header.Get("Content-Type")
	res, err := h.svc.Transcribe(c.Request.Context(), httpx.MustUserID(c), fileHeader.Filename, mime, data, c.PostForm("language"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, res)
}

type saveResultReq struct {
	AudioFileID uint64 `json:"audio_file_id"`
	Text        string `json:"text"`
	Translation string `json:"translation"`
}

func (h *Handler) saveResult(c *gin.Context) {
	var req saveResultReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.ErrValidation("请求体格式错误"))
		return
	}
	rec, err := h.svc.SaveResult(c.Request.Context(), httpx.MustUserID(c), req.AudioFileID, req.Text, req.Translation)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, rec)
}
