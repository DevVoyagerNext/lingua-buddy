<template>
  <div>
    <h2 class="title">专项训练</h2>
    <div class="card">
      <div class="row">
        <button :class="tab === 'translation' ? '' : 'ghost'" @click="tab = 'translation'">翻译训练</button>
        <button :class="tab === 'wrong' ? '' : 'ghost'" @click="tab = 'wrong'; loadWrong()">翻译错题</button>
      </div>
    </div>

    <!-- 翻译训练 -->
    <div v-if="tab === 'translation'">
      <div class="card">
        <div class="row">
          <select v-model="direction" style="width: 160px">
            <option value="zh_to_en">中译英</option>
            <option value="en_to_zh">英译中</option>
          </select>
          <select v-model="difficulty" style="width: 140px">
            <option value="cet4">四级</option>
            <option value="cet6">六级</option>
          </select>
          <button :disabled="loadingNext" @click="nextQuestion">{{ loadingNext ? '出题中...' : '出一题' }}</button>
        </div>
      </div>

      <div v-if="source" class="card">
        <p class="muted">请翻译：</p>
        <p style="font-size: 16px">{{ source }}</p>
        <textarea v-model="answer" rows="3" placeholder="输入你的译文" :disabled="evaluating"></textarea>
        <div class="row" style="margin-top: 10px">
          <button :disabled="evaluating" @click="evaluate">{{ evaluating ? 'AI 评价中...' : '提交评价' }}</button>
          <span v-if="error" class="error">{{ error }}</span>
        </div>
      </div>

      <div v-if="evaluation" class="card">
        <h4>参考译文</h4>
        <p>{{ evaluation.reference_text }}</p>
        <p class="muted">准确性：{{ evaluation.accuracy }}</p>
        <p class="muted">语法：{{ evaluation.grammar_issues }}</p>
        <p class="muted">自然度：{{ evaluation.naturalness }}</p>
        <p class="muted">建议：{{ evaluation.suggestion }}</p>
        <button class="ghost small" @click="confirmWrong">确认我答错了，加入错题</button>
        <span v-if="wrongMsg" class="ok" style="margin-left: 8px">{{ wrongMsg }}</span>
      </div>
    </div>

    <!-- 翻译错题 -->
    <div v-else class="card">
      <div v-for="w in wrong" :key="w.id" class="list-item" style="display:block">
        <p style="margin: 4px 0"><b>{{ w.direction === 'zh_to_en' ? '中译英' : '英译中' }}：</b>{{ w.question_text }}</p>
        <p class="muted" style="margin: 2px 0">你的译文：{{ w.user_answer }}</p>
        <p class="muted" style="margin: 2px 0">参考：{{ w.reference_answer }}</p>
        <button class="ghost small" @click="resolveWrong(w.id)">已解决，移除</button>
      </div>
      <p v-if="!wrong.length" class="muted">还没有翻译错题。</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { api, ApiError } from '@/api/client'

const tab = ref<'translation' | 'wrong'>('translation')
const direction = ref('zh_to_en')
const difficulty = ref('cet4')
const source = ref('')
const answer = ref('')
const evaluation = ref<any>(null)
const recordId = ref<number | null>(null)
const loadingNext = ref(false)
const evaluating = ref(false)
const error = ref('')
const wrongMsg = ref('')
const wrong = ref<any[]>([])

async function nextQuestion() {
  error.value = ''
  evaluation.value = null
  answer.value = ''
  loadingNext.value = true
  try {
    const resp = await api.post('/training/translations/next', { direction: direction.value, difficulty: difficulty.value })
    source.value = resp.data.text
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : '出题失败'
  } finally {
    loadingNext.value = false
  }
}

async function evaluate() {
  if (!answer.value.trim()) return
  error.value = ''
  wrongMsg.value = ''
  evaluating.value = true
  try {
    const resp = await api.post('/training/translations/evaluate', {
      submission_id: crypto.randomUUID(),
      direction: direction.value,
      source_text: source.value,
      user_answer: answer.value,
    })
    evaluation.value = resp.data.evaluation
    recordId.value = resp.data.record_id
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : '评价失败'
  } finally {
    evaluating.value = false
  }
}

async function confirmWrong() {
  if (!recordId.value) return
  await api.post(`/training/answers/${recordId.value}/confirm-wrong`)
  wrongMsg.value = '已加入翻译错题'
}

async function loadWrong() {
  const resp = await api.get('/translation-wrong-questions')
  wrong.value = resp.data.items || []
}
async function resolveWrong(id: number) {
  await api.delete(`/translation-wrong-questions/${id}`)
  await loadWrong()
}
</script>
