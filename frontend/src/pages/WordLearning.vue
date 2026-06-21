<template>
  <div>
    <h2 class="title">单词学习 / 到期复习</h2>

    <div v-if="status === 'NO_ACTIVE_PLAN'" class="card">
      <p>当前没有可学的单词。</p>
      <RouterLink to="/word-plans"><button>去创建词汇计划</button></RouterLink>
      <RouterLink to="/dictionary" style="margin-left: 8px"><button class="ghost">去查词收藏生词</button></RouterLink>
    </div>

    <div v-else-if="status === 'NO_DUE_WORDS'" class="card">
      <p>暂时没有到期的单词，稍后再来 👍</p>
      <button class="ghost" @click="fetchNext">刷新</button>
    </div>

    <div v-else-if="question" class="card">
      <div class="row">
        <span class="tag">{{ stageLabel(question.stage) }}</span>
        <div class="spacer" />
        <button class="ghost small" @click="fetchNext">跳过 / 下一题</button>
      </div>

      <h2 style="margin: 16px 0; font-size: 26px">{{ question.prompt }}</h2>

      <!-- 选择题 -->
      <div v-if="question.options && question.options.length">
        <button
          v-for="opt in question.options"
          :key="opt"
          class="option-btn"
          :class="optionClass(opt)"
          :disabled="!!feedback"
          @click="submit(opt)"
        >
          {{ opt }}
        </button>
      </div>

      <!-- 拼写题 -->
      <div v-else class="col">
        <input v-model="spellingAnswer" placeholder="请输入英文单词" :disabled="!!feedback" @keyup.enter="submit(spellingAnswer)" />
        <div class="row">
          <button :disabled="!!feedback" @click="submit(spellingAnswer)">提交</button>
          <button class="ghost" :disabled="!!feedback" @click="useHint = true">使用提示</button>
          <span v-if="useHint" class="muted">提示：首字母 {{ '' }}（使用提示不计入晋级）</span>
        </div>
      </div>

      <div v-if="feedback" class="feedback" :class="feedback.correct ? 'ok' : 'error'">
        <p>{{ feedback.correct ? '✅ 回答正确' : '❌ 回答错误' }}，正确答案：<b>{{ feedback.correct_answer }}</b></p>
        <p class="muted">
          阶段：{{ stageLabel(feedback.stage_before) }} → {{ stageLabel(feedback.stage_after) }}
        </p>
        <button @click="fetchNext">下一题</button>
      </div>
    </div>

    <div v-else class="card"><p class="muted">加载中...</p></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api, ApiError } from '@/api/client'

interface Question {
  question_type: string
  stage: string
  prompt: string
  options: string[] | null
  token: string
  user_word_id: number
}
interface Feedback {
  correct: boolean
  correct_answer: string
  stage_before: string
  stage_after: string
}

const question = ref<Question | null>(null)
const status = ref('')
const feedback = ref<Feedback | null>(null)
const spellingAnswer = ref('')
const useHint = ref(false)
let submissionId = ''
let lastAnswer = ''

const stageNames: Record<string, string> = {
  recognition: '初识·英文选中文',
  discrimination: '辨认·中文选英文',
  spelling: '默写·中文写英文',
  mastered: '已掌握·维护复习',
}
function stageLabel(s: string) {
  return stageNames[s] || s
}

async function fetchNext() {
  feedback.value = null
  question.value = null
  status.value = ''
  spellingAnswer.value = ''
  useHint.value = false
  try {
    const resp = await api.get<any>('/word-learning/next')
    if (resp.code === 'OK') {
      question.value = resp.data
      submissionId = crypto.randomUUID()
    } else {
      status.value = resp.code
    }
  } catch (e) {
    status.value = e instanceof ApiError ? e.code : 'ERROR'
  }
}

function optionClass(opt: string) {
  if (!feedback.value) return ''
  if (opt === feedback.value.correct_answer) return 'correct'
  if (opt === lastAnswer && !feedback.value.correct) return 'wrong'
  return ''
}

async function submit(answer: string) {
  if (!question.value || feedback.value || !answer.trim()) return
  lastAnswer = answer
  try {
    const resp = await api.post<Feedback>('/word-learning/answer', {
      submission_id: submissionId,
      token: question.value.token,
      answer,
      used_hint: useHint.value,
    })
    feedback.value = resp.data
  } catch (e) {
    if (e instanceof ApiError && (e.code === 'QUESTION_TOKEN_EXPIRED' || e.code === 'QUESTION_TOKEN_INVALID')) {
      await fetchNext()
    }
  }
}

onMounted(fetchNext)
</script>

<style scoped>
.feedback { margin-top: 16px; padding-top: 12px; border-top: 1px solid var(--border); }
</style>
