package asr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Paraformer 阿里云 DashScope 录音文件识别（异步：提交任务 → 轮询 → 拉取转写 JSON）。
type Paraformer struct {
	base   string
	key    string
	model  string
	client *http.Client
}

// NewParaformer 构造。base 形如 https://dashscope.aliyuncs.com/api/v1。
func NewParaformer(base, key, model string) *Paraformer {
	if model == "" {
		model = "paraformer-v2"
	}
	return &Paraformer{
		base:   strings.TrimRight(base, "/"),
		key:    key,
		model:  model,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type submitResp struct {
	Output struct {
		TaskID     string `json:"task_id"`
		TaskStatus string `json:"task_status"`
	} `json:"output"`
	Message string `json:"message"`
}

type pollResp struct {
	Output struct {
		TaskStatus string `json:"task_status"`
		Code       string `json:"code"`
		Message    string `json:"message"`
		Results    []struct {
			TranscriptionURL string `json:"transcription_url"`
			SubtaskStatus    string `json:"subtask_status"`
			Code             string `json:"code"`
			Message          string `json:"message"`
		} `json:"results"`
	} `json:"output"`
	Message string `json:"message"`
}

type transcriptionDoc struct {
	Transcripts []struct {
		Text string `json:"text"`
	} `json:"transcripts"`
}

// Transcribe 提交识别任务并同步等待结果（内部轮询）。
func (p *Paraformer) Transcribe(ctx context.Context, audioURL, language string) (Transcript, error) {
	taskID, err := p.submit(ctx, audioURL, language)
	if err != nil {
		return Transcript{}, fmt.Errorf("%w: submit: %v", ErrFailed, err)
	}

	deadline := time.Now().Add(90 * time.Second)
	for {
		if time.Now().After(deadline) {
			return Transcript{}, fmt.Errorf("%w: 轮询超时", ErrFailed)
		}
		select {
		case <-ctx.Done():
			return Transcript{}, fmt.Errorf("%w: %v", ErrFailed, ctx.Err())
		case <-time.After(2 * time.Second):
		}
		pr, err := p.poll(ctx, taskID)
		if err != nil {
			return Transcript{}, fmt.Errorf("%w: poll: %v", ErrFailed, err)
		}
		switch pr.Output.TaskStatus {
		case "SUCCEEDED":
			url := ""
			if len(pr.Output.Results) > 0 {
				url = pr.Output.Results[0].TranscriptionURL
			}
			text, err := p.fetchTranscription(ctx, url)
			if err != nil {
				return Transcript{}, fmt.Errorf("%w: fetch: %v", ErrFailed, err)
			}
			lang := language
			if lang == "" {
				lang = "auto"
			}
			return Transcript{Text: text, Language: lang}, nil
		case "FAILED":
			// 音频可被访问与处理，但无有效语音片段：按“识别到空文本”成功返回，而非报错。
			if pr.Output.Code == "SUCCESS_WITH_NO_VALID_FRAGMENT" ||
				(len(pr.Output.Results) > 0 && pr.Output.Results[0].Code == "SUCCESS_WITH_NO_VALID_FRAGMENT") {
				lang := language
				if lang == "" {
					lang = "auto"
				}
				return Transcript{Text: "", Language: lang}, nil
			}
			reason := pr.Output.Message
			if reason == "" && len(pr.Output.Results) > 0 {
				reason = pr.Output.Results[0].Message
			}
			return Transcript{}, fmt.Errorf("%w: 任务失败 code=%s msg=%s", ErrFailed, pr.Output.Code, reason)
		}
		// PENDING / RUNNING：继续轮询。
	}
}

func (p *Paraformer) submit(ctx context.Context, audioURL, language string) (string, error) {
	params := map[string]any{}
	if language != "" && language != "auto" {
		params["language_hints"] = []string{language}
	}
	body := map[string]any{
		"model":      p.model,
		"input":      map[string]any{"file_urls": []string{audioURL}},
		"parameters": params,
	}
	buf, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.base+"/services/audio/asr/transcription", bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.key)
	req.Header.Set("X-DashScope-Async", "enable")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http %d: %s", resp.StatusCode, trunc(string(raw)))
	}
	var sr submitResp
	if err := json.Unmarshal(raw, &sr); err != nil || sr.Output.TaskID == "" {
		return "", fmt.Errorf("无效提交响应: %s", trunc(string(raw)))
	}
	return sr.Output.TaskID, nil
}

func (p *Paraformer) poll(ctx context.Context, taskID string) (pollResp, error) {
	var pr pollResp
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.base+"/tasks/"+taskID, nil)
	if err != nil {
		return pr, err
	}
	req.Header.Set("Authorization", "Bearer "+p.key)
	resp, err := p.client.Do(req)
	if err != nil {
		return pr, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(raw, &pr); err != nil {
		return pr, fmt.Errorf("无效轮询响应: %s", trunc(string(raw)))
	}
	return pr, nil
}

func (p *Paraformer) fetchTranscription(ctx context.Context, url string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("空转写 URL")
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var doc transcriptionDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", err
	}
	var parts []string
	for _, t := range doc.Transcripts {
		if strings.TrimSpace(t.Text) != "" {
			parts = append(parts, t.Text)
		}
	}
	return strings.Join(parts, " "), nil
}

func trunc(s string) string {
	if len(s) > 300 {
		return s[:300]
	}
	return s
}
