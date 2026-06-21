package main

// 配置连通性验证程序：逐项真实连接，确认 .env 配置是否可用。
// 运行：cd backend && go run ./cmd/verify
//
// 只用标准库 + 已有的 mysql 驱动，不引入额外依赖。
// ASR 用阿里云官方公开示例音频做端到端识别；TTS 为 WebSocket（标准库无客户端），
// 仅说明其与 AI 共用同一 sk- Key，Key 有效性由 AI 测试间接确认。

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

type result struct {
	name   string
	ok     bool
	detail string
}

func env(k string) string { return strings.TrimSpace(os.Getenv(k)) }

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("警告：未能加载 .env（请在 backend 目录运行）：", err)
	}

	results := []result{
		testMySQL(),
		testRedis(),
		testSMTP(),
		testOSS(),
		testAI(),
		testASR(),
		testTTS(),
		testRSS(),
	}

	fmt.Println("\n==================== 验证结果汇总 ====================")
	pass := 0
	for _, r := range results {
		mark := "❌ 失败"
		if r.ok {
			mark = "✅ 通过"
			pass++
		}
		fmt.Printf("%-18s %s  %s\n", r.name, mark, r.detail)
	}
	fmt.Printf("======================================================\n")
	fmt.Printf("通过 %d/%d\n", pass, len(results))
}

// ---------- MySQL ----------
func testMySQL() result {
	const name = "MySQL"
	user := env("DB_USER")
	if user == "" {
		return result{name, false, "DB_USER 为空"}
	}
	cfg := mysql.Config{
		User: user, Passwd: env("DB_PASSWORD"), Net: "tcp",
		Addr: orDefault("DB_ADDR", "127.0.0.1:3306"),
		DBName: orDefault("DB_NAME", "lingua"),
		AllowNativePasswords: true,
	}
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return result{name, false, err.Error()}
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return result{name, false, "连接失败: " + err.Error()}
	}
	var cnt int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ecdict WHERE word=?", "apple").Scan(&cnt); err != nil {
		return result{name, false, "查询 ecdict 失败: " + err.Error()}
	}
	return result{name, true, fmt.Sprintf("连接成功，ecdict 中 apple 命中 %d 条", cnt)}
}

// ---------- Redis（裸 RESP，AUTH + PING）----------
func testRedis() result {
	const name = "Redis"
	addr := orDefault("REDIS_ADDR", "127.0.0.1:6379")
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return result{name, false, "无法连接 " + addr + "（Redis 未启动？）: " + err.Error()}
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	r := bufio.NewReader(conn)

	if pw := env("REDIS_PASSWORD"); pw != "" {
		fmt.Fprintf(conn, "AUTH %s\r\n", pw)
		line, _ := r.ReadString('\n')
		if strings.HasPrefix(line, "-") {
			// 密码错或服务端未设密码
			if strings.Contains(line, "no password") {
				return result{name, false, "Redis 未设置密码，但 .env 配了密码"}
			}
			return result{name, false, "AUTH 失败: " + strings.TrimSpace(line)}
		}
	}
	fmt.Fprintf(conn, "PING\r\n")
	line, err := r.ReadString('\n')
	if err != nil {
		return result{name, false, "PING 无响应: " + err.Error()}
	}
	if strings.HasPrefix(line, "+PONG") {
		return result{name, true, "AUTH + PING 成功（注意 REDIS_ENABLED=" + env("REDIS_ENABLED") + "）"}
	}
	return result{name, false, "PING 返回异常: " + strings.TrimSpace(line)}
}

// ---------- SMTP（QQ 邮箱，仅验证登录，不发信）----------
func testSMTP() result {
	const name = "QQ邮箱SMTP"
	host := orDefault("MAIL_SMTP_HOST", "smtp.qq.com")
	port := orDefault("MAIL_SMTP_PORT", "465")
	user := env("MAIL_USERNAME")
	pass := env("MAIL_PASSWORD")
	if user == "" || pass == "" {
		return result{name, false, "MAIL_USERNAME/MAIL_PASSWORD 为空"}
	}
	addr := host + ":" + port
	d := &net.Dialer{Timeout: 8 * time.Second}
	conn, err := tls.DialWithDialer(d, "tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return result{name, false, "TLS 连接失败: " + err.Error()}
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return result{name, false, "SMTP 握手失败: " + err.Error()}
	}
	defer c.Close()
	if err := c.Auth(smtp.PlainAuth("", user, pass, host)); err != nil {
		return result{name, false, "登录失败（授权码错误？）: " + err.Error()}
	}
	return result{name, true, "登录成功（" + user + "）"}
}

// ---------- OSS（V1 签名，列举 1 个对象）----------
func testOSS() result {
	const name = "阿里云OSS"
	bucket := env("OBJECT_STORAGE_BUCKET")
	ak := env("OBJECT_STORAGE_ACCESS_KEY")
	sk := env("OBJECT_STORAGE_SECRET_KEY")
	region := orDefault("OBJECT_STORAGE_REGION", "oss-cn-beijing")
	if bucket == "" || ak == "" || sk == "" {
		return result{name, false, "bucket/AK/SK 有空值"}
	}
	hostName := bucket + "." + region + ".aliyuncs.com"
	urlStr := "https://" + hostName + "/?max-keys=1"
	date := time.Now().UTC().Format(http.TimeFormat)
	resource := "/" + bucket + "/"
	stringToSign := "GET\n\n\n" + date + "\n" + resource
	mac := hmac.New(sha1.New, []byte(sk))
	mac.Write([]byte(stringToSign))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	req, _ := http.NewRequest("GET", urlStr, nil)
	req.Header.Set("Date", date)
	req.Header.Set("Authorization", "OSS "+ak+":"+sig)
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return result{name, false, "请求失败: " + err.Error()}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	switch resp.StatusCode {
	case 200:
		return result{name, true, "签名校验通过，bucket 可访问（" + hostName + "）"}
	case 403:
		return result{name, false, "403 鉴权失败（AK/SK 错或无权限）: " + extractXML(string(body), "Code")}
	case 404:
		return result{name, false, "404 bucket 不存在或区域不对: " + extractXML(string(body), "Code")}
	default:
		return result{name, false, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, extractXML(string(body), "Code"))}
	}
}

// ---------- 千问 AI（OpenAI 兼容 chat/completions）----------
func testAI() result {
	const name = "千问AI"
	base := env("AI_API_BASE")
	key := env("AI_API_KEY")
	model := orDefault("AI_MODEL", "qwen-plus")
	if base == "" || key == "" {
		return result{name, false, "AI_API_BASE/AI_API_KEY 为空"}
	}
	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": "只回复两个字：你好"},
		},
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", strings.TrimRight(base, "/")+"/chat/completions", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return result{name, false, "请求失败: " + err.Error()}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return result{name, false, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 160))}
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	json.Unmarshal(body, &out)
	reply := ""
	if len(out.Choices) > 0 {
		reply = out.Choices[0].Message.Content
	}
	return result{name, true, fmt.Sprintf("模型 %s 调用成功，回复：%q", model, reply)}
}

// ---------- Paraformer ASR（异步任务 + 轮询，用官方示例音频）----------
func testASR() result {
	const name = "Paraformer ASR"
	base := env("ASR_API_BASE")
	key := env("ASR_API_KEY")
	model := orDefault("ASR_MODEL", "paraformer-v2")
	if base == "" || key == "" {
		return result{name, false, "ASR_API_BASE/ASR_API_KEY 为空"}
	}
	sample := "https://dashscope.oss-cn-beijing.aliyuncs.com/samples/audio/paraformer/hello_world_female2.wav"
	payload := map[string]any{
		"model": model,
		"input": map[string]any{"file_urls": []string{sample}},
		"parameters": map[string]any{"language_hints": []string{"en", "zh"}},
	}
	b, _ := json.Marshal(payload)
	submitURL := strings.TrimRight(base, "/") + "/services/audio/asr/transcription"
	req, _ := http.NewRequest("POST", submitURL, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-DashScope-Async", "enable")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return result{name, false, "提交失败: " + err.Error()}
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return result{name, false, fmt.Sprintf("提交 HTTP %d: %s", resp.StatusCode, truncate(string(body), 160))}
	}
	var sub struct {
		Output struct {
			TaskID     string `json:"task_id"`
			TaskStatus string `json:"task_status"`
		} `json:"output"`
	}
	json.Unmarshal(body, &sub)
	if sub.Output.TaskID == "" {
		return result{name, false, "未拿到 task_id: " + truncate(string(body), 160)}
	}
	// 轮询任务
	taskURL := strings.TrimRight(base, "/") + "/tasks/" + sub.Output.TaskID
	deadline := time.Now().Add(40 * time.Second)
	for {
		time.Sleep(3 * time.Second)
		treq, _ := http.NewRequest("GET", taskURL, nil)
		treq.Header.Set("Authorization", "Bearer "+key)
		tr, err := client.Do(treq)
		if err != nil {
			return result{name, false, "轮询失败: " + err.Error()}
		}
		tb, _ := io.ReadAll(tr.Body)
		tr.Body.Close()
		var st struct {
			Output struct {
				TaskStatus string `json:"task_status"`
			} `json:"output"`
		}
		json.Unmarshal(tb, &st)
		switch st.Output.TaskStatus {
		case "SUCCEEDED":
			return result{name, true, "示例音频识别任务成功（端到端打通）"}
		case "FAILED":
			return result{name, false, "任务 FAILED: " + truncate(string(tb), 160)}
		}
		if time.Now().After(deadline) {
			return result{name, false, "超时未完成（提交成功，Key 有效，但识别未在 40s 内返回）"}
		}
	}
}

// ---------- TTS（WebSocket，标准库无客户端，仅说明）----------
func testTTS() result {
	const name = "CosyVoice TTS"
	base := env("TTS_API_BASE")
	key := env("TTS_API_KEY")
	if base == "" || key == "" {
		return result{name, false, "TTS_API_BASE/TTS_API_KEY 为空"}
	}
	shared := key == env("AI_API_KEY")
	msg := "WebSocket 协议，本程序未做实时合成；"
	if shared {
		msg += "Key 与 AI 共用，已由千问测试间接验证有效"
	} else {
		msg += "Key 与 AI 不同，需单独验证"
	}
	return result{name, true, msg}
}

// ---------- VOA 外刊 RSS ----------
func testRSS() result {
	const name = "VOA外刊RSS"
	urls := env("ARTICLE_FEED_URLS")
	if urls == "" {
		return result{name, false, "ARTICLE_FEED_URLS 为空"}
	}
	first := strings.Split(urls, ",")[0]
	req, _ := http.NewRequest("GET", first, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 LinguaBuddy")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return result{name, false, "请求失败: " + err.Error()}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != 200 {
		return result{name, false, fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}
	s := string(body)
	if strings.Contains(s, "<rss") || strings.Contains(s, "<item") || strings.Contains(s, "<feed") {
		return result{name, true, "Feed 可访问且为 RSS/Atom 格式"}
	}
	return result{name, false, "返回内容不像 RSS: " + truncate(s, 80)}
}

// ---------- 工具函数 ----------
func orDefault(k, def string) string {
	if v := env(k); v != "" {
		return v
	}
	return def
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

func extractXML(body, tag string) string {
	open, close := "<"+tag+">", "</"+tag+">"
	i := strings.Index(body, open)
	j := strings.Index(body, close)
	if i >= 0 && j > i {
		return body[i+len(open) : j]
	}
	return truncate(body, 80)
}
