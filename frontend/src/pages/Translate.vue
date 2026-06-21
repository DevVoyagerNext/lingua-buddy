<template>
  <div>
    <h2 class="title">智能翻译</h2>

    <!-- 翻译主面板 -->
    <div class="card trans-panel">
      <div class="panel-grid">
        <div class="pane">
          <div class="pane-head">
            <span class="pane-label">原文</span>
            <span class="muted small">
              <template v-if="recording">录音中…点击结束</template>
              <template v-else-if="transcribing">识别中…</template>
              <template v-else>自动识别中英文</template>
            </span>
            <button class="mic-btn" :class="{ recording }" :disabled="transcribing"
              :title="recording ? '点击结束录音' : '点击开始语音输入'" @click="toggleRecord">
              <svg v-if="!recording" viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor"
                stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                <rect x="9" y="2" width="6" height="11" rx="3" fill="currentColor" stroke="none" />
                <path d="M5 10.5a7 7 0 0 0 14 0" />
                <line x1="12" y1="17.5" x2="12" y2="21" />
                <line x1="8.5" y1="21" x2="15.5" y2="21" />
              </svg>
              <svg v-else viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
                <rect x="6" y="6" width="12" height="12" rx="2.5" />
              </svg>
              <span>{{ recording ? '结束录音' : '语音输入' }}</span>
            </button>
          </div>
          <textarea v-model="text" rows="7" placeholder="输入要翻译的中文或英文…"></textarea>
        </div>

        <div class="pane result-pane">
          <div class="pane-head">
            <span class="pane-label">译文</span>
            <button v-if="result" class="star-btn" :class="{ lit: collected }"
              :title="collected ? '已收藏' : '收藏译文'" @click="collectSentence">
              <svg viewBox="0 0 24 24" width="20" height="20" stroke="currentColor" stroke-width="1.6"
                stroke-linejoin="round" :fill="collected ? 'currentColor' : 'none'">
                <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
              </svg>
            </button>
          </div>
          <div class="result-box">
            <p v-if="loading" class="muted">翻译中…</p>
            <p v-else-if="result" class="result-text">{{ result.translated_text }}</p>
            <p v-else class="muted placeholder">译文将显示在这里</p>
          </div>
        </div>
      </div>

      <div class="toolbar">
        <select v-model="tone" class="tone-select">
          <option value="default">默认语气</option>
          <option value="daily">日常</option>
          <option value="formal">正式</option>
          <option value="business">商务</option>
          <option value="academic">学术</option>
        </select>
        <span v-if="error" class="error">{{ error }}</span>
        <span v-else-if="loading" class="muted small">翻译中…</span>
        <div class="spacer" />
        <span class="muted small">输入后自动翻译</span>
      </div>
    </div>

    <!-- 关键表达 / 备选译法 -->
    <div v-if="result && (result.key_expressions?.length || result.alternatives?.length)" class="extra-grid">
      <div v-if="result.key_expressions?.length" class="card">
        <h3 class="sub-title">关键表达</h3>
        <div v-for="(k, i) in result.key_expressions" :key="i" class="expr-item">
          <b class="expr">{{ k.expression }}</b>
          <span class="muted">{{ k.explanation_zh }}</span>
        </div>
      </div>
      <div v-if="result.alternatives?.length" class="card">
        <h3 class="sub-title">备选译法</h3>
        <div v-for="(a, i) in result.alternatives" :key="i" class="alt-item">
          <span class="alt-index">{{ i + 1 }}</span>
          <span>{{ a }}</span>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from 'vue'
import { api, ApiError } from '@/api/client'

interface TransOut {
  translated_text: string
  key_expressions: { expression: string; explanation_zh: string }[]
  alternatives: string[]
}

const text = ref('')
const tone = ref('default')
const result = ref<TransOut | null>(null)
const loading = ref(false)
const error = ref('')
const collected = ref(false)
const recording = ref(false)
const transcribing = ref(false)

let mediaRecorder: MediaRecorder | null = null
let chunks: Blob[] = []

// 语音输入：同一个按钮在录音/非录音状态切换。
function toggleRecord() {
  if (recording.value) {
    mediaRecorder?.stop()
    recording.value = false
  } else {
    startRecord()
  }
}

async function startRecord() {
  error.value = ''
  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    mediaRecorder = new MediaRecorder(stream)
    chunks = []
    mediaRecorder.ondataavailable = (e) => chunks.push(e.data)
    mediaRecorder.onstop = async () => {
      const blob = new Blob(chunks, { type: 'audio/webm' })
      stream.getTracks().forEach((t) => t.stop())
      await transcribe(new File([blob], 'recording.webm', { type: 'audio/webm' }))
    }
    mediaRecorder.start()
    recording.value = true
  } catch {
    error.value = '无法访问麦克风，请检查浏览器权限。'
  }
}

// 识别语音为文字并追加到原文，随后由 watch 自动翻译。
async function transcribe(file: File) {
  transcribing.value = true
  try {
    const form = new FormData()
    form.append('audio', file)
    form.append('language', 'auto')
    const resp = await api.post('/speech/transcribe', form, { headers: { 'Content-Type': 'multipart/form-data' } })
    const recognized = (resp.data.text || '').trim()
    if (recognized) {
      text.value = text.value.trim() ? `${text.value.trim()} ${recognized}` : recognized
    }
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : '语音识别失败'
  } finally {
    transcribing.value = false
  }
}

const DEBOUNCE_MS = 600
let debounceTimer: ReturnType<typeof setTimeout> | null = null
let reqSeq = 0 // 请求序号，丢弃过期响应，避免竞态

// 内容变化后防抖触发翻译。
function scheduleTranslate() {
  collected.value = false
  error.value = ''
  if (debounceTimer) clearTimeout(debounceTimer)
  if (!text.value.trim()) {
    result.value = null
    loading.value = false
    return
  }
  debounceTimer = setTimeout(doTranslate, DEBOUNCE_MS)
}

async function doTranslate() {
  if (!text.value.trim()) {
    result.value = null
    return
  }
  const seq = ++reqSeq
  loading.value = true
  try {
    const resp = await api.post<{ output: TransOut }>('/translations', { text: text.value, tone: tone.value })
    if (seq !== reqSeq) return // 已有更新的请求，丢弃过期结果
    result.value = resp.data.output
  } catch (e) {
    if (seq !== reqSeq) return
    error.value = e instanceof ApiError ? e.message : '翻译失败'
  } finally {
    if (seq === reqSeq) loading.value = false
  }
}

// 监听原文与语气变化，自动重新翻译。
watch([text, tone], scheduleTranslate)
onBeforeUnmount(() => {
  if (debounceTimer) clearTimeout(debounceTimer)
})

async function collectSentence() {
  if (!result.value) return
  try {
    await api.post('/sentences', { sentence: result.value.translated_text, translation: text.value })
    collected.value = true
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : '收藏失败'
  }
}
</script>

<style scoped>
.small { font-size: 12px; }

.trans-panel { padding: 0; overflow: hidden; }
.panel-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
}
.pane { padding: 16px; }
.result-pane { border-left: 1px solid var(--border); background: #fafbff; }
.pane-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
  min-height: 28px;
}
.pane-head .small { margin-left: auto; }

.mic-btn {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  padding: 0 14px;
  border-radius: 999px;
  border: none;
  background: var(--primary);
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: filter 0.15s, transform 0.12s;
}
.mic-btn:hover:not(:disabled) { filter: brightness(1.08); transform: translateY(-1px); }
.mic-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.mic-btn.recording {
  background: #e25555;
  animation: pulse 1.2s ease-in-out infinite;
}
@keyframes pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(226, 85, 85, 0.45); }
  50% { box-shadow: 0 0 0 7px rgba(226, 85, 85, 0); }
}
.pane-label {
  display: inline-block;
  font-size: 12px;
  font-weight: 700;
  color: var(--primary);
  background: #eef2fb;
  padding: 2px 10px;
  border-radius: 6px;
}
.pane textarea {
  width: 100%;
  border: none;
  resize: vertical;
  font-size: 15px;
  line-height: 1.7;
  padding: 0;
  background: transparent;
}
.pane textarea:focus { outline: none; box-shadow: none; }
.result-box { min-height: 130px; }
.result-text { margin: 0; font-size: 16px; line-height: 1.7; white-space: pre-wrap; }
.placeholder { margin: 0; }

.star-btn {
  display: inline-flex;
  background: transparent;
  border: none;
  cursor: pointer;
  padding: 2px;
  color: #cbd2e0;
  transition: transform 0.12s, color 0.12s;
}
.star-btn.lit { color: #f5b301; }
.star-btn:hover { transform: scale(1.15); }

.toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border-top: 1px solid var(--border);
  background: #fff;
}
.tone-select { width: 140px; }

.extra-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}
.sub-title { margin: 0 0 12px; font-size: 15px; }
.expr-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 0;
  border-bottom: 1px dashed var(--border);
}
.expr-item:last-child { border-bottom: none; }
.expr { color: var(--text); }
.alt-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 8px 0;
  font-size: 14px;
  line-height: 1.6;
  border-bottom: 1px dashed var(--border);
}
.alt-item:last-child { border-bottom: none; }
.alt-index {
  flex: 0 0 auto;
  width: 20px;
  height: 20px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 700;
  color: var(--primary);
  background: #eef2fb;
  border-radius: 50%;
}

@media (max-width: 760px) {
  .panel-grid,
  .extra-grid { grid-template-columns: 1fr; }
  .result-pane { border-left: none; border-top: 1px solid var(--border); }
}
</style>
