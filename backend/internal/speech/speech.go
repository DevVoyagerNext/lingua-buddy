// Package speech 提供音频上传识别与语音历史保存。
package speech

import (
	"context"
	"fmt"
	"io"
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
	dir     string
}

// NewService 构造。dir 为本地音频目录（真实生产音频走 OSS）。
func NewService(db *gorm.DB, provider asr.Provider, hist *history.Repository, dir string) *Service {
	return &Service{db: db, asr: provider, history: hist, dir: dir}
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

	path, err := s.saveFile(userID, filename, data)
	if err != nil {
		return nil, err
	}
	af := &models.AudioFile{
		UserID: userID, FilePath: path, OriginalName: filename,
		MimeType: mime, FileSize: int64(len(data)),
	}
	if err := s.db.WithContext(ctx).Create(af).Error; err != nil {
		return nil, err
	}

	tr, err := s.asr.Transcribe(ctx, data, mime, language)
	if err != nil {
		return nil, httpx.NewError(502, "ASR_FAILED", "语音识别失败，请重试")
	}
	return &TranscribeResult{Text: tr.Text, Language: tr.Language, AudioFileID: af.ID}, nil
}

func (s *Service) saveFile(userID uint64, filename string, data []byte) (string, error) {
	ext := filepath.Ext(filename)
	dir := filepath.Join(s.dir, fmt.Sprintf("%d", userID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	full := filepath.Join(dir, name)
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
