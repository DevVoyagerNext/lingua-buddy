<template>
  <div>
    <h2 class="title">作文批改</h2>
    <div class="card input-card">
      <input v-model="title" class="title-input" placeholder="作文标题" />
      <div class="selectors">
        <label class="field">
          文体
          <select v-model="essayType">
            <option value="argumentative">议论文</option>
            <option value="narrative">记叙文</option>
            <option value="expository">说明文</option>
            <option value="letter">书信</option>
          </select>
        </label>
        <label class="field">
          目标考试
          <select v-model="targetExam">
            <option value="">不限考试</option>
            <option value="cet4">四级</option>
            <option value="cet6">六级</option>
          </select>
        </label>
      </div>
      <textarea v-model="body" rows="9" placeholder="在此输入作文正文…" @keydown.ctrl.enter="submit"></textarea>
      <div class="count-row muted small">字母 {{ letterCount }} · 单词 {{ wordCount }}</div>
      <div class="toolbar">
        <span v-if="error" class="error">{{ error }}</span>
        <span v-else-if="loading" class="muted small">AI 批改中…</span>
        <div class="spacer" />
        <button class="submit-btn" :disabled="loading || !body.trim()" title="提交批改" @click="submit">
          <svg v-if="!loading" viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor"
            stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="22" y1="2" x2="11" y2="13" />
            <polygon points="22 2 15 22 11 13 2 9 22 2" />
          </svg>
          <span v-else class="spinner" />
        </button>
      </div>
    </div>

    <div v-if="review" class="card result-card">
      <div class="result-head">
        <span class="section-label">批改结果</span>
        <span class="muted small">AI 生成，仅供学习参考</span>
      </div>

      <div class="scores">
        <div class="score-card">
          <span class="score-val">{{ review.scores.grammar }}</span>
          <span class="score-name">语法</span>
        </div>
        <div class="score-card">
          <span class="score-val">{{ review.scores.vocabulary }}</span>
          <span class="score-name">词汇</span>
        </div>
        <div class="score-card">
          <span class="score-val">{{ review.scores.structure }}</span>
          <span class="score-name">结构</span>
        </div>
        <div class="score-card">
          <span class="score-val">{{ review.scores.coherence }}</span>
          <span class="score-name">连贯</span>
        </div>
      </div>

      <p class="overall">{{ review.overall_comment }}</p>

      <div v-if="review.issues?.length" class="section">
        <span class="section-label">问题与建议</span>
        <div v-for="(it, i) in review.issues" :key="i" class="issue">
          <span class="tag">{{ it.type }}</span>
          <span class="diff">
            <span class="del">{{ it.original }}</span>
            <span class="arrow">→</span>
            <span class="add">{{ it.replacement }}</span>
          </span>
          <p class="muted issue-exp">{{ it.explanation_zh }}</p>
        </div>
      </div>

      <div class="section">
        <span class="section-label">修改后参考</span>
        <p class="revised">{{ review.revised_text }}</p>
        <p v-if="review.revision_reason" class="muted reason">{{ review.revision_reason }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { api, ApiError } from '@/api/client'

const title = ref('')
const body = ref('')
const essayType = ref('argumentative')
const targetExam = ref('')
const loading = ref(false)
const error = ref('')
const review = ref<any>(null)

// 字母数量（英文字母）与单词数量。
const letterCount = computed(() => (body.value.match(/[a-zA-Z]/g) || []).length)
const wordCount = computed(() => {
  const t = body.value.trim()
  return t ? t.split(/\s+/).length : 0
})

async function submit() {
  error.value = ''
  if (!body.value.trim()) {
    error.value = '请输入作文正文'
    return
  }
  loading.value = true
  try {
    const resp = await api.post('/essays/review', {
      submission_id: crypto.randomUUID(),
      title: title.value,
      body: body.value,
      essay_type: essayType.value,
      target_exam: targetExam.value,
    })
    review.value = resp.data.review
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : '批改失败'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.small { font-size: 12px; }

.input-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.title-input { width: 100%; font-size: 15px; font-weight: 600; }
.selectors {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}
.field {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #667;
  white-space: nowrap;
}
.field select { width: 130px; }
.count-row {
  text-align: right;
  margin-top: -4px;
}
.toolbar {
  display: flex;
  align-items: center;
  gap: 14px;
}

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

.result-card { border-left: 3px solid var(--primary); }
.result-head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.section-label {
  display: inline-block;
  font-size: 12px;
  font-weight: 700;
  color: var(--primary);
  background: #eef2fb;
  padding: 3px 12px;
  border-radius: 6px;
}

.scores {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  margin: 16px 0;
}
.score-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 14px 8px;
  background: #f7f9ff;
  border: 1px solid var(--border);
  border-radius: 12px;
}
.score-val { font-size: 22px; font-weight: 700; color: var(--primary); }
.score-name { font-size: 12px; color: #8a93a6; }

.overall { margin: 8px 0 4px; font-size: 14px; line-height: 1.7; }

.section { margin-top: 18px; }
.section-label + * { margin-top: 10px; }
.issue {
  padding: 10px 0;
  border-top: 1px dashed var(--border);
}
.diff { font-size: 14px; }
.del { color: #e25555; text-decoration: line-through; }
.arrow { margin: 0 6px; color: #8a93a6; }
.add { color: #2e9b5b; font-weight: 600; }
.issue-exp { margin: 4px 0 0; font-size: 13px; line-height: 1.5; }

.revised {
  white-space: pre-wrap;
  font-size: 15px;
  line-height: 1.8;
  background: #fafbff;
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 14px 16px;
}
.reason { margin: 8px 0 0; font-size: 13px; line-height: 1.6; }

@media (max-width: 640px) {
  .scores { grid-template-columns: repeat(2, 1fr); }
}
</style>
