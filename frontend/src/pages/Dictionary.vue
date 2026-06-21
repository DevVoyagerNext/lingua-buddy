<template>
  <div>
    <h2 class="title">查单词</h2>
    <div class="card search-card">
      <div class="search">
        <input
          ref="searchInput"
          v-model="query"
          placeholder="输入英文单词查询…"
          @input="onInput"
          @focus="onFocus"
          @blur="onBlur"
          @keyup.enter="doLookup(query)"
        />
        <button class="icon-btn search-btn" title="查询" @click="doLookup(query)">
          <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2"
            stroke-linecap="round" stroke-linejoin="round">
            <circle cx="11" cy="11" r="7" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
        </button>
      </div>

      <!-- 搜索框聚焦且为空时，在同一卡片内展示搜索历史 -->
      <div v-if="showHistory && !query.trim()" class="history-panel">
        <div class="list-head">
          <span class="section-label">搜索历史</span>
        </div>
        <div v-for="h in history" :key="h.id" class="word-row" @mousedown.prevent="doLookup(h.word)">
          <b class="word-text">{{ h.word }}</b>
          <span class="muted word-def">{{ h.gloss || '—' }}</span>
          <button class="icon-btn del-btn" title="删除" @mousedown.prevent.stop="delHistory(h.id)">
            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2"
              stroke-linecap="round" stroke-linejoin="round">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>
        <p v-if="!history.length" class="muted empty">暂无搜索历史</p>
      </div>
    </div>

    <div v-if="suggestions.length && !entry" class="card word-list">
      <div v-for="s in suggestions" :key="s.word" class="word-row" @click="doLookup(s.word)">
        <b class="word-text">{{ s.word }}</b>
        <span class="muted word-def">{{ s.gloss || '—' }}</span>
      </div>
    </div>

    <div v-if="notFound" class="card">
      <p class="error" style="margin-top: 0">未找到该单词。</p>
      <div v-if="similar.length" class="word-list" style="margin-top: 4px">
        <p class="muted" style="margin: 0 0 6px">你是不是想找：</p>
        <div v-for="s in similar" :key="s.word" class="word-row" @click="doLookup(s.word)">
          <b class="word-text">{{ s.word }}</b>
          <span class="muted word-def">{{ s.gloss || '—' }}</span>
        </div>
      </div>
    </div>

    <div v-if="entry" class="card entry-card">
      <div class="entry-head">
        <div class="word-block">
          <h3 class="entry-word">{{ entry.word }}</h3>
          <span v-if="entry.phonetic" class="phonetic">/{{ entry.phonetic }}/</span>
        </div>
        <div class="spacer" />
        <select v-model="familiarity" class="fam-select">
          <option value="unknown">不认识</option>
          <option value="fuzzy">有点印象</option>
          <option value="known">认识不会写</option>
          <option value="mastered">已熟练</option>
        </select>
        <button class="collect-btn" @click="collect">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="1.8"
            stroke-linejoin="round">
            <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
          </svg>
          加入生词本
        </button>
      </div>

      <p v-if="entry.lemma" class="muted lemma">原形：<a @click="doLookup(entry.lemma)">{{ entry.lemma }}</a></p>

      <div v-if="entry.tags?.length || entry.collins_stars || entry.oxford_core" class="tag-row">
        <span v-for="t in entry.tags" :key="t" class="tag">{{ t }}</span>
        <span v-if="entry.collins_stars" class="tag">柯林斯 {{ '★'.repeat(entry.collins_stars) }}</span>
        <span v-if="entry.oxford_core" class="tag">牛津核心</span>
      </div>

      <div v-if="entry.translations?.length" class="section">
        <span class="section-label">释义</span>
        <p v-for="(t, i) in entry.translations" :key="i" class="def-line">{{ t }}</p>
      </div>
      <div v-if="entry.definitions?.length" class="section">
        <span class="section-label">英文释义</span>
        <p v-for="(d, i) in entry.definitions" :key="i" class="muted def-line">{{ d }}</p>
      </div>
      <div v-if="entry.word_forms?.length" class="tag-row">
        <span v-for="(w, i) in entry.word_forms" :key="i" class="tag">{{ w.type }}: {{ w.word }}</span>
      </div>
      <p v-if="collectMsg" :class="collectMsgClass">{{ collectMsg }}</p>

      <div class="section">
        <div class="section-bar">
          <span class="section-label">AI 例句</span>
          <button class="ghost small" :disabled="exLoading" @click="genExamples">
            {{ exLoading ? '生成中…' : '生成例句' }}
          </button>
        </div>
        <div v-for="(ex, i) in examples" :key="i" class="example">
          <p class="ex-en">{{ ex.english }}</p>
          <p class="muted ex-zh">{{ ex.chinese }}（{{ ex.word_meaning }}）</p>
        </div>
      </div>

      <div class="section">
        <span class="section-label">单词笔记</span>
        <div class="note-input">
          <input v-model="noteContent" placeholder="为这个单词添加笔记…" @keyup.enter="addNote" />
          <button class="small" @click="addNote">添加</button>
        </div>
        <div v-for="n in notes" :key="n.id" class="note-item">
          <span>{{ n.content }}</span>
          <div class="spacer" />
          <button class="icon-btn del-btn" title="删除" @click="delNote(n.id)">
            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2"
              stroke-linecap="round" stroke-linejoin="round">
              <polyline points="3 6 5 6 21 6" />
              <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6" />
              <path d="M10 11v6M14 11v6" />
              <path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, onMounted, onActivated } from 'vue'
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
const suggestions = ref<{ word: string; gloss: string }[]>([])
const similar = ref<{ word: string; gloss: string }[]>([])
const history = ref<{ id: number; word: string; query_count: number; gloss: string }[]>([])
const showHistory = ref(false)
const searchInput = ref<HTMLInputElement | null>(null)

// 进入页面时自动聚焦搜索框。
async function focusSearch() {
  await nextTick()
  searchInput.value?.focus()
}
onMounted(focusSearch)
onActivated(focusSearch)
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

// 聚焦空搜索框时展示搜索历史。
async function onFocus() {
  showHistory.value = true
  if (!query.value.trim()) await loadHistory()
}
function onBlur() {
  // 延迟隐藏，保证历史项的点击先生效。
  setTimeout(() => {
    showHistory.value = false
  }, 150)
}
async function loadHistory() {
  try {
    const resp = await api.get<{ items: { id: number; word: string; query_count: number; gloss: string }[] }>(
      '/dictionary/history?page_size=10',
    )
    history.value = resp.data.items || []
  } catch {
    history.value = []
  }
}
async function delHistory(id: number) {
  try {
    await api.delete(`/dictionary/history/${id}`)
    await loadHistory()
  } catch {
    /* 忽略 */
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
      const resp = await api.get<{ word: string; gloss: string }[]>(`/dictionary/suggestions?q=${encodeURIComponent(query.value.trim())}`)
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
.search {
  position: relative;
  display: flex;
  align-items: center;
}
.search input {
  width: 100%;
  padding-right: 42px;
}
.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  padding: 6px;
  border-radius: 8px;
  cursor: pointer;
  color: inherit;
}
.icon-btn:hover { background: #eef2fb; }
.search-btn { position: absolute; right: 4px; color: var(--primary); }

/* 搜索框聚焦时，历史面板紧贴输入框 */
.history-panel {
  margin-top: 12px;
  border-top: 1px solid var(--border);
}
.history-panel .word-row { padding-left: 0; padding-right: 0; }
.history-panel .word-row:last-child { border-bottom: none; }

/* 联想词 / 相似词列表，左词右义 */
.word-list { padding: 4px 0; }
.list-head { padding: 8px 0 4px; }
.empty { text-align: center; padding: 18px; }
.word-row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
  cursor: pointer;
  transition: background 0.12s;
}
.word-row:last-child { border-bottom: none; }
.word-row:hover { background: #f7f9ff; }
.word-text {
  flex: 0 0 150px;
  font-size: 15px;
  color: var(--text);
  word-break: break-word;
}
.word-def {
  flex: 1;
  font-size: 13px;
  line-height: 1.5;
}

/* 词条详情 */
.entry-head { display: flex; align-items: center; gap: 10px; }
.word-block { display: flex; align-items: baseline; gap: 10px; }
.entry-word { margin: 0; font-size: 24px; }
.phonetic { color: #8a93a6; font-size: 15px; }
.fam-select { width: 130px; }
.collect-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.lemma { margin: 8px 0 0; }
.lemma a { cursor: pointer; color: var(--primary); }

.tag-row { display: flex; flex-wrap: wrap; gap: 6px; margin: 12px 0; }

.section { margin-top: 16px; }
.section-label {
  display: inline-block;
  font-size: 12px;
  font-weight: 700;
  color: var(--primary);
  background: #eef2fb;
  padding: 2px 10px;
  border-radius: 6px;
  margin-bottom: 8px;
}
.section-bar { display: flex; align-items: center; gap: 10px; margin-bottom: 8px; }
.section-bar .section-label { margin-bottom: 0; }
.def-line { margin: 4px 0; font-size: 15px; line-height: 1.6; }

.example {
  padding: 8px 0;
  border-bottom: 1px dashed var(--border);
}
.example:last-child { border-bottom: none; }
.ex-en { margin: 2px 0; font-size: 14px; line-height: 1.6; }
.ex-zh { margin: 2px 0; font-size: 13px; }

.note-input { display: flex; gap: 8px; margin-bottom: 8px; }
.note-input input { flex: 1; }
.note-item {
  display: flex;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px dashed var(--border);
}
.note-item:last-child { border-bottom: none; }
.del-btn { color: #aab; }
.del-btn:hover { background: #fdecec; color: #e25555; }
</style>
