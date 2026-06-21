<template>
  <div>
    <h2 class="title">句子分析</h2>
    <div class="card input-card">
      <p class="hint muted">提交后将自动进行 <b>语法分析</b> · <b>语法纠错</b> · <b>句子润色</b></p>
      <textarea v-model="text" rows="5" placeholder="输入英文句子或短文…" @keydown.ctrl.enter="run"></textarea>
      <div class="toolbar">
        <label class="style-pick">
          润色风格
          <select v-model="style">
            <option value="natural">自然</option>
            <option value="concise">简洁</option>
            <option value="formal">正式</option>
            <option value="advanced">高级</option>
          </select>
        </label>
        <span v-if="error" class="error">{{ error }}</span>
        <span v-else-if="loading" class="muted small">处理中…</span>
        <div class="spacer" />
        <button class="submit-btn" :disabled="loading || !text.trim()" title="提交" @click="run">
          <svg v-if="!loading" viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor"
            stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <path d="M5 3l14 9-14 9V3z" />
          </svg>
          <span v-else class="spinner" />
        </button>
      </div>
    </div>

    <!-- 结构分析 -->
    <div v-if="analysis" class="card result-card">
      <span class="section-label">语法分析</span>
      <div class="chips">
        <span v-if="analysis.sentence_type" class="chip">句型：{{ analysis.sentence_type }}</span>
        <span v-if="analysis.tense" class="chip">时态：{{ analysis.tense }}</span>
        <span v-if="analysis.voice" class="chip">语态：{{ analysis.voice }}</span>
      </div>
      <p v-if="analysis.main_clause" class="muted main-clause">
        主干：主语「{{ analysis.main_clause.subject }}」谓语「{{ analysis.main_clause.predicate }}」宾语「{{ analysis.main_clause.object }}」
      </p>
      <div v-if="analysis.grammar_points?.length" class="points">
        <p v-for="(g, i) in analysis.grammar_points" :key="i" class="point">
          <b>{{ g.name }}</b><span class="muted">{{ g.explanation_zh }}</span>
        </p>
      </div>
      <p v-if="analysis.explanation_zh" class="muted explain">{{ analysis.explanation_zh }}</p>
    </div>

    <!-- 纠错 / 润色 -->
    <div v-for="c in corrections" :key="c.key" class="card result-card">
      <div class="result-head">
        <span class="section-label">{{ c.label }}</span>
        <button class="star-btn" :class="{ lit: collected[c.key] }"
          :title="collected[c.key] ? '已收藏' : '收藏句子'" @click="collect(c.key, c.data.corrected_text)">
          <svg viewBox="0 0 24 24" width="20" height="20" stroke="currentColor" stroke-width="1.6"
            stroke-linejoin="round" :fill="collected[c.key] ? 'currentColor' : 'none'">
            <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
          </svg>
        </button>
      </div>
      <p class="corrected">{{ c.data.corrected_text }}</p>
      <div v-if="c.data.issues?.length" class="issues">
        <div v-for="(it, i) in c.data.issues" :key="i" class="issue">
          <span class="tag">{{ it.type }}</span>
          <span class="diff">
            <span class="del">{{ it.original }}</span>
            <span class="arrow">→</span>
            <span class="add">{{ it.replacement }}</span>
          </span>
          <p class="muted issue-exp">{{ it.explanation_zh }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { api, ApiError } from '@/api/client'

const text = ref('')
const style = ref('natural')
const loading = ref(false)
const error = ref('')
const analysis = ref<any>(null)
const correction = ref<any>(null)
const polish = ref<any>(null)
const collected = reactive<Record<string, boolean>>({})

const corrections = computed(() =>
  [
    { key: 'correct', label: '语法纠错', data: correction.value },
    { key: 'polish', label: '句子润色', data: polish.value },
  ].filter((c) => c.data),
)

async function run() {
  error.value = ''
  analysis.value = null
  correction.value = null
  polish.value = null
  collected.correct = false
  collected.polish = false
  if (!text.value.trim()) {
    error.value = '请输入文本'
    return
  }
  loading.value = true
  try {
    const [a, c, p] = await Promise.allSettled([
      api.post('/grammar/analysis', { text: text.value }),
      api.post('/corrections', { text: text.value, mode: 'correct', style: style.value }),
      api.post('/corrections', { text: text.value, mode: 'polish', style: style.value }),
    ])
    if (a.status === 'fulfilled') analysis.value = a.value.data.analysis
    if (c.status === 'fulfilled') correction.value = c.value.data.result
    if (p.status === 'fulfilled') polish.value = p.value.data.result
    if (a.status === 'rejected' && c.status === 'rejected' && p.status === 'rejected') {
      const e = a.reason
      error.value = e instanceof ApiError ? e.message : '处理失败'
    }
  } finally {
    loading.value = false
  }
}

async function collect(key: string, sentence: string) {
  try {
    await api.post('/sentences', { sentence })
    collected[key] = true
  } catch {
    collected[key] = true // 重复收藏也视为已收藏
  }
}
</script>

<style scoped>
.small { font-size: 12px; }

.input-card .hint { margin: 0 0 10px; font-size: 13px; }
.input-card .hint b { color: var(--primary); }

.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 12px;
}
.style-pick {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #667;
}
.style-pick select { width: 110px; }

.submit-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  padding: 0;
  border-radius: 50%;
  flex: 0 0 auto;
}
.submit-btn:disabled { opacity: 0.45; cursor: not-allowed; }
.spinner {
  width: 18px;
  height: 18px;
  border: 2px solid rgba(255, 255, 255, 0.5);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

.section-label {
  display: inline-block;
  font-size: 12px;
  font-weight: 700;
  color: var(--primary);
  background: #eef2fb;
  padding: 3px 12px;
  border-radius: 6px;
}
.result-card { border-left: 3px solid var(--primary); }
.result-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
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

.chips { display: flex; flex-wrap: wrap; gap: 8px; margin: 12px 0 8px; }
.chip {
  font-size: 12px;
  color: #475;
  background: #f0f4f0;
  border: 1px solid #e0e8e0;
  padding: 2px 10px;
  border-radius: 999px;
}
.main-clause { margin: 6px 0; font-size: 13px; }
.points { margin: 8px 0; }
.point { margin: 6px 0; font-size: 13px; line-height: 1.6; }
.point b { color: var(--text); margin-right: 6px; }
.explain { margin-top: 8px; font-size: 13px; line-height: 1.6; }

.corrected {
  margin: 12px 0;
  font-size: 16px;
  line-height: 1.7;
  font-weight: 600;
  color: var(--text);
}
.issues { margin-top: 6px; }
.issue {
  padding: 10px 0;
  border-top: 1px dashed var(--border);
}
.diff { font-size: 14px; }
.del { color: #e25555; text-decoration: line-through; }
.arrow { margin: 0 6px; color: #8a93a6; }
.add { color: #2e9b5b; font-weight: 600; }
.issue-exp { margin: 4px 0 0; font-size: 13px; line-height: 1.5; }
</style>
