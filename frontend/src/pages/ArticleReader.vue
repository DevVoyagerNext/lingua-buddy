<template>
  <div v-if="article">
    <div class="row">
      <RouterLink to="/articles"><button class="ghost small">← 返回列表</button></RouterLink>
      <div class="spacer" />
      <button class="small" @click="markFinished">标记为已读</button>
    </div>
    <div class="card" style="margin-top: 12px">
      <h2>{{ article.title }}</h2>
      <p class="muted">
        {{ article.source_name }} ·
        <a :href="article.source_url" target="_blank">原文链接</a>
      </p>
      <div class="article-body">
        <span
          v-for="(tok, i) in tokens"
          :key="i"
          :class="{ word: tok.isWord }"
          @click="tok.isWord && lookup(tok.text)"
          >{{ tok.text }}</span
        >
      </div>
      <p class="muted" style="margin-top: 8px">点击正文中的单词可查词并收藏。</p>
    </div>

    <div v-if="entry" class="card popup">
      <div class="row">
        <h3 style="margin: 0">{{ entry.word }}</h3>
        <span v-if="entry.phonetic" class="muted">/{{ entry.phonetic }}/</span>
        <div class="spacer" />
        <button class="small" @click="collect">加入生词本</button>
        <button class="ghost small" @click="entry = null">关闭</button>
      </div>
      <p v-for="(t, i) in entry.translations" :key="i" style="margin: 2px 0">{{ t }}</p>
      <span v-if="collectMsg" :class="collectMsgClass">{{ collectMsg }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { api, ApiError } from '@/api/client'

interface Article {
  id: number
  title: string
  summary: string | null
  content: string | null
  source_name: string
  source_url: string
}

const route = useRoute()
const article = ref<Article | null>(null)
const entry = ref<any>(null)
const collectMsg = ref('')
const collectMsgClass = ref('ok')

const tokens = computed(() => {
  const text = article.value?.content || article.value?.summary || ''
  return text.split(/([^A-Za-z']+)/).map((t) => ({ text: t, isWord: /[A-Za-z]/.test(t) && t.length > 1 }))
})

async function loadArticle() {
  const id = route.params.id
  const resp = await api.get(`/articles/${id}`)
  article.value = resp.data
  api.post(`/articles/${id}/read`, { finished: false }).catch(() => {})
}

async function lookup(word: string) {
  collectMsg.value = ''
  try {
    const resp = await api.get(`/dictionary/entries/${encodeURIComponent(word.toLowerCase())}`)
    entry.value = resp.data
  } catch {
    entry.value = { word, translations: ['（词典中未找到）'] }
  }
}

async function collect() {
  if (!entry.value) return
  try {
    await api.post('/vocabulary', { word: entry.value.word, familiarity: 'unknown' })
    collectMsg.value = '已加入生词本'
    collectMsgClass.value = 'ok'
  } catch (e) {
    collectMsg.value = e instanceof ApiError ? e.message : '收藏失败'
    collectMsgClass.value = 'error'
  }
}

async function markFinished() {
  await api.post(`/articles/${route.params.id}/read`, { finished: true })
}

onMounted(loadArticle)
</script>

<style scoped>
.article-body { line-height: 1.9; font-size: 15px; margin-top: 12px; white-space: pre-wrap; }
.word { cursor: pointer; border-radius: 3px; }
.word:hover { background: #fff3b0; }
.popup { position: sticky; bottom: 16px; border: 2px solid var(--primary); }
</style>
