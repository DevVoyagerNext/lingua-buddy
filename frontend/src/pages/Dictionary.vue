<template>
  <div>
    <h2 class="title">查单词</h2>
    <div class="card">
      <div class="row">
        <input
          v-model="query"
          placeholder="输入英文单词，回车查询"
          @input="onInput"
          @keyup.enter="doLookup(query)"
        />
        <button @click="doLookup(query)">查询</button>
      </div>
      <div v-if="suggestions.length" class="suggestions">
        <span v-for="s in suggestions" :key="s.word" class="tag" style="cursor: pointer" @click="doLookup(s.word)">
          {{ s.word }}
        </span>
      </div>
    </div>

    <div v-if="notFound" class="card">
      <p class="error">未找到该单词。</p>
      <div v-if="similar.length">
        <span class="muted">你是不是想找：</span>
        <span v-for="s in similar" :key="s.word" class="tag" style="cursor: pointer" @click="doLookup(s.word)">
          {{ s.word }}
        </span>
      </div>
    </div>

    <div v-if="entry" class="card">
      <div class="row">
        <h3 style="margin: 0">{{ entry.word }}</h3>
        <span v-if="entry.phonetic" class="muted">/{{ entry.phonetic }}/</span>
        <div class="spacer" />
        <select v-model="familiarity" style="width: 140px">
          <option value="unknown">不认识</option>
          <option value="fuzzy">有点印象</option>
          <option value="known">认识不会写</option>
          <option value="mastered">已熟练</option>
        </select>
        <button class="small" @click="collect">加入生词本</button>
      </div>
      <p v-if="entry.lemma" class="muted">原形：<a @click="doLookup(entry.lemma)" style="cursor:pointer">{{ entry.lemma }}</a></p>
      <div style="margin: 8px 0">
        <span v-for="t in entry.tags" :key="t" class="tag">{{ t }}</span>
        <span v-if="entry.collins_stars" class="tag">柯林斯 {{ '★'.repeat(entry.collins_stars) }}</span>
        <span v-if="entry.oxford_core" class="tag">牛津核心</span>
      </div>
      <div v-if="entry.translations?.length">
        <p v-for="(t, i) in entry.translations" :key="i" style="margin: 4px 0">{{ t }}</p>
      </div>
      <div v-if="entry.definitions?.length" class="muted">
        <p v-for="(d, i) in entry.definitions" :key="i" style="margin: 2px 0">{{ d }}</p>
      </div>
      <div v-if="entry.word_forms?.length" style="margin-top: 8px">
        <span v-for="(w, i) in entry.word_forms" :key="i" class="tag">{{ w.type }}: {{ w.word }}</span>
      </div>
      <p v-if="collectMsg" :class="collectMsgClass">{{ collectMsg }}</p>

      <div class="examples">
        <button class="ghost small" :disabled="exLoading" @click="genExamples">
          {{ exLoading ? 'AI 生成中...' : 'AI 例句' }}
        </button>
        <div v-for="(ex, i) in examples" :key="i" style="margin-top: 8px">
          <p style="margin: 2px 0">{{ ex.english }}</p>
          <p class="muted" style="margin: 2px 0">{{ ex.chinese }}（{{ ex.word_meaning }}）</p>
        </div>
      </div>

      <div class="notes">
        <h4>单词笔记</h4>
        <div class="row">
          <input v-model="noteContent" placeholder="为这个单词添加笔记" @keyup.enter="addNote" />
          <button class="small" @click="addNote">添加</button>
        </div>
        <div v-for="n in notes" :key="n.id" class="list-item">
          <span>{{ n.content }}</span>
          <div class="spacer" />
          <button class="ghost small" @click="delNote(n.id)">删除</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { api, ApiError } from '@/api/client'

interface Entry {
  word: string
  phonetic: string
  translations: string[]
  definitions: string[]
  tags: string[]
  collins_stars: number | null
  oxford_core: boolean
  word_forms: { type: string; word: string }[]
  lemma: string
}

const query = ref('')
const entry = ref<Entry | null>(null)
const suggestions = ref<{ word: string }[]>([])
const similar = ref<{ word: string }[]>([])
const notFound = ref(false)
const familiarity = ref('unknown')
const collectMsg = ref('')
const collectMsgClass = ref('ok')
const notes = ref<{ id: number; content: string }[]>([])
const noteContent = ref('')
const examples = ref<{ english: string; chinese: string; word_meaning: string }[]>([])
const exLoading = ref(false)

async function genExamples() {
  if (!entry.value) return
  exLoading.value = true
  try {
    const resp = await api.post('/dictionary/examples', { word: entry.value.word, difficulty: 'cet4' })
    examples.value = resp.data || []
  } catch {
    examples.value = []
  } finally {
    exLoading.value = false
  }
}

let timer: any
function onInput() {
  clearTimeout(timer)
  notFound.value = false
  timer = setTimeout(async () => {
    if (query.value.trim().length < 2) {
      suggestions.value = []
      return
    }
    try {
      const resp = await api.get<{ word: string }[]>(`/dictionary/suggestions?q=${encodeURIComponent(query.value.trim())}`)
      suggestions.value = resp.data || []
    } catch {
      suggestions.value = []
    }
  }, 200)
}

async function doLookup(word: string) {
  const w = word.trim()
  if (!w) return
  query.value = w
  suggestions.value = []
  notFound.value = false
  entry.value = null
  collectMsg.value = ''
  examples.value = []
  try {
    const resp = await api.get<Entry>(`/dictionary/entries/${encodeURIComponent(w)}`)
    entry.value = resp.data
    await loadNotes(w)
  } catch (e) {
    if (e instanceof ApiError && e.code === 'NOT_FOUND') {
      notFound.value = true
      similar.value = e.data?.suggestions || []
    }
  }
}

async function collect() {
  if (!entry.value) return
  collectMsg.value = ''
  try {
    await api.post('/vocabulary', { word: entry.value.word, familiarity: familiarity.value })
    collectMsg.value = '已加入生词本'
    collectMsgClass.value = 'ok'
  } catch (e) {
    collectMsg.value = e instanceof ApiError ? e.message : '收藏失败'
    collectMsgClass.value = 'error'
  }
}

async function loadNotes(word: string) {
  try {
    const resp = await api.get<{ id: number; content: string }[]>(`/word-notes?word=${encodeURIComponent(word)}`)
    notes.value = resp.data || []
  } catch {
    notes.value = []
  }
}
async function addNote() {
  if (!entry.value || !noteContent.value.trim()) return
  await api.post('/word-notes', { word: entry.value.word, content: noteContent.value.trim() })
  noteContent.value = ''
  await loadNotes(entry.value.word)
}
async function delNote(id: number) {
  await api.delete(`/word-notes/${id}`)
  if (entry.value) await loadNotes(entry.value.word)
}
</script>

<style scoped>
.suggestions { margin-top: 8px; }
.notes { margin-top: 16px; border-top: 1px solid var(--border); padding-top: 12px; }
</style>
