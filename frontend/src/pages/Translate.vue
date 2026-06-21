<template>
  <div>
    <h2 class="title">智能翻译</h2>
    <div class="card">
      <textarea v-model="text" rows="5" placeholder="输入要翻译的中文或英文（自动识别方向）"></textarea>
      <div class="row" style="margin-top: 10px">
        <select v-model="tone" style="width: 140px">
          <option value="default">默认语气</option>
          <option value="daily">日常</option>
          <option value="formal">正式</option>
          <option value="business">商务</option>
          <option value="academic">学术</option>
        </select>
        <button :disabled="loading" @click="doTranslate">{{ loading ? '翻译中...' : '翻译' }}</button>
        <span v-if="error" class="error">{{ error }}</span>
      </div>
    </div>

    <div v-if="result" class="card">
      <h3>译文</h3>
      <p style="font-size: 16px">{{ result.translated_text }}</p>
      <button class="ghost small" @click="collectSentence">收藏译文句子</button>
      <span v-if="collectMsg" :class="collectMsgClass" style="margin-left: 8px">{{ collectMsg }}</span>

      <div v-if="result.key_expressions?.length" style="margin-top: 12px">
        <h4>关键表达</h4>
        <p v-for="(k, i) in result.key_expressions" :key="i" class="muted">
          <b>{{ k.expression }}</b>：{{ k.explanation_zh }}
        </p>
      </div>
      <div v-if="result.alternatives?.length" style="margin-top: 8px">
        <h4>备选译法</h4>
        <p v-for="(a, i) in result.alternatives" :key="i" class="muted">{{ a }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
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
const collectMsg = ref('')
const collectMsgClass = ref('ok')

async function doTranslate() {
  error.value = ''
  collectMsg.value = ''
  if (!text.value.trim()) {
    error.value = '请输入文本'
    return
  }
  loading.value = true
  try {
    const resp = await api.post<{ output: TransOut }>('/translations', { text: text.value, tone: tone.value })
    result.value = resp.data.output
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : '翻译失败'
  } finally {
    loading.value = false
  }
}

async function collectSentence() {
  if (!result.value) return
  collectMsg.value = ''
  try {
    await api.post('/sentences', { sentence: result.value.translated_text, translation: text.value })
    collectMsg.value = '已收藏'
    collectMsgClass.value = 'ok'
  } catch (e) {
    collectMsg.value = e instanceof ApiError ? e.message : '收藏失败'
    collectMsgClass.value = 'error'
  }
}
</script>
