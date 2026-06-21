package main

// Paraformer 异步识别完整 demo：提交 → 轮询 → 下载 transcription_url → 打印识别文字。
// 运行：cd backend && go run ./cmd/asrdemo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	base := strings.TrimRight(os.Getenv("ASR_API_BASE"), "/")
	key := strings.TrimSpace(os.Getenv("ASR_API_KEY"))
	model := os.Getenv("ASR_MODEL")
	if model == "" {
		model = "paraformer-v2"
	}
	client := &http.Client{Timeout: 30 * time.Second}
	sample := "https://dashscope.oss-cn-beijing.aliyuncs.com/samples/audio/paraformer/hello_world_female2.wav"

	// 1. 提交任务
	payload, _ := json.Marshal(map[string]any{
		"model":      model,
		"input":      map[string]any{"file_urls": []string{sample}},
		"parameters": map[string]any{"language_hints": []string{"en", "zh"}},
	})
	req, _ := http.NewRequest("POST", base+"/services/audio/asr/transcription", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-DashScope-Async", "enable")
	resp, err := client.Do(req)
	must(err)
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	var sub struct {
		Output struct {
			TaskID string `json:"task_id"`
		} `json:"output"`
	}
	json.Unmarshal(b, &sub)
	fmt.Println("① 提交成功，task_id =", sub.Output.TaskID)

	// 2. 轮询状态
	var transURL string
	for i := 0; i < 15; i++ {
		time.Sleep(3 * time.Second)
		treq, _ := http.NewRequest("GET", base+"/tasks/"+sub.Output.TaskID, nil)
		treq.Header.Set("Authorization", "Bearer "+key)
		tr, err := client.Do(treq)
		must(err)
		tb, _ := io.ReadAll(tr.Body)
		tr.Body.Close()
		var st struct {
			Output struct {
				TaskStatus string `json:"task_status"`
				Results    []struct {
					TranscriptionURL string `json:"transcription_url"`
					SubtaskStatus    string `json:"subtask_status"`
				} `json:"results"`
			} `json:"output"`
		}
		json.Unmarshal(tb, &st)
		fmt.Printf("② 第%d次轮询，状态 = %s\n", i+1, st.Output.TaskStatus)
		if st.Output.TaskStatus == "SUCCEEDED" {
			if len(st.Output.Results) > 0 {
				transURL = st.Output.Results[0].TranscriptionURL
			}
			break
		}
		if st.Output.TaskStatus == "FAILED" {
			fmt.Println("任务失败:", string(tb))
			return
		}
	}
	if transURL == "" {
		fmt.Println("未拿到 transcription_url")
		return
	}
	fmt.Println("③ transcription_url =", transURL)

	// 3. 下载结果 JSON，取出识别文字
	rr, err := client.Get(transURL)
	must(err)
	rb, _ := io.ReadAll(rr.Body)
	rr.Body.Close()
	var res struct {
		Transcripts []struct {
			Text string `json:"text"`
		} `json:"transcripts"`
	}
	json.Unmarshal(rb, &res)
	fmt.Println("④ 识别结果文字：")
	for _, t := range res.Transcripts {
		fmt.Println("   →", t.Text)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
