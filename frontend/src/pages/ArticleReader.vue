<template>
  <div v-if="article">
    <div class="top-bar">
      <RouterLink to="/articles">
        <button class="icon-btn" title="返回列表">
          <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"
            stroke-linecap="round" stroke-linejoin="round">
            <line x1="19" y1="12" x2="5" y2="12" />
            <polyline points="12 19 5 12 12 5" />
          </svg>
        </button>
      </RouterLink>
      <span class="hint muted">点击单词可查词收藏；选中一句话可翻译并分析。</span>
      <button class="icon-btn" title="标记为已读" @click="markFinished">
        <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"
          stroke-linecap="round" stroke-linejoin="round">
          <polyline points="20 6 9 17 4 12" />
        </svg>
      </button>
    </div>
    <div class="card" style="margin-top: 12px">
      <h2>{{ article.title }}</h2>
      <div class="article-body" @mouseup="onSelect">
        <span
          v-for="(tok, i) in tokens"
          :key="i"
          :class="{ word: tok.isWord }"
          @click="tok.isWord && lookup(tok.text)"
          >{{ tok.text }}</span
        >
      </div>
    </div>

    <div class="card nav-row">
      <button class="icon-btn" title="上一篇文章" :disabled="!article.prev_id" @click="goPrev">
        <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"
          stroke-linecap="round" stroke-linejoin="round">
          <polyline points="15 18 9 12 15 6" />
        </svg>
      </button>
      <div class="spacer" />
      <button class="icon-btn" title="下一篇文章" :disabled="!article.next_id" @click="goNext">
        <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"
          stroke-linecap="round" stroke-linejoin="round">
          <polyline points="9 18 15 12 9 6" />
        </svg>
      </button>
    </div>

    <!-- 选中句子/段落：翻译与句子分析 -->
    <div v-if="selectedText" class="card sel-panel">
      <div class="row">
        <b>选中文本</b>
        <div class="spacer" />
        <button class="sel-icon-btn" :class="{ lit: collected }" :title="collected ? '已收藏' : '收藏句子'"
          @click="collectSel">
          <svg viewBox="0 0 24 24" width="20" height="20" stroke="currentColor" stroke-width="1.6"
            stroke-linejoin="round" :fill="collected ? 'currentColor' : 'none'">
            <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
          </svg>
        </button>
        <button class="sel-icon-btn" title="关闭" @click="clearSel">
          <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"
            stroke-linecap="round" stroke-linejoin="round">
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </button>
      </div>
      <p class="sel-text">{{ selectedText }}</p>
      <span v-if="selMsg" :class="selMsgClass">{{ selMsg }}</span>

      <div v-if="loadingTrans" class="muted">翻译中...</div>
      <div v-if="selTranslation" class="sel-result">
        <h4>翻译</h4>
        <p style="font-size: 15px">{{ selTranslation.translated_text }}</p>
        <div v-if="selTranslation.key_expressions?.length">
          <p v-for="(k, i) in selTranslation.key_expressions" :key="i" class="muted">
            <b>{{ k.expression }}</b>：{{ k.explanation_zh }}
          </p>
        </div>
      </div>

      <div v-if="loadingAnalysis" class="muted">分析中...</div>
      <div v-if="selAnalysis" class="sel-result">
        <h4>句子分析</h4>
        <p>句型：{{ selAnalysis.sentence_type }}｜时态：{{ selAnalysis.tense }}｜语态：{{ selAnalysis.voice }}</p>
        <p class="muted">
          主干：主语「{{ selAnalysis.main_clause.subject }}」谓语「{{ selAnalysis.main_clause.predicate }}」宾语「{{ selAnalysis.main_clause.object }}」
        </p>
        <div v-if="selAnalysis.clauses?.length">
          <p v-for="(cl, i) in selAnalysis.clauses" :key="i" class="muted">从句（{{ cl.type }}）：{{ cl.text }}</p>
        </div>
        <div v-if="selAnalysis.grammar_points?.length" style="margin-top: 6px">
          <p v-for="(g, i) in selAnalysis.grammar_points" :key="i"><b>{{ g.name }}</b>：{{ g.explanation_zh }}</p>
        </div>
        <p v-if="selAnalysis.explanation_zh">{{ selAnalysis.explanation_zh }}</p>
      </div>
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
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, ApiError } from '@/api/client'

interface Article {
  id: number
  title: string
  summary: string | null
  content: string | null
  source_name: string
  source_url: string
  prev_id: number | null
  next_id: number | null
}

const route = useRoute()
const router = useRouter()
const article = ref<Article | null>(null)
const entry = ref<any>(null)
const collectMsg = ref('')
const collectMsgClass = ref('ok')

// 选中句子/段落的翻译与分析。
const selectedText = ref('')
const selTranslation = ref<any>(null)
const selAnalysis = ref<any>(null)
const loadingTrans = ref(false)
const loadingAnalysis = ref(false)
const selMsg = ref('')
const selMsgClass = ref('ok')
const collected = ref(false)

function onSelect() {
  const s = window.getSelection()?.toString().replace(/\s+/g, ' ').trim() || ''
  if (s.length >= 2) {
    selectedText.value = s
    selTranslation.value = null
    selAnalysis.value = null
    selMsg.value = ''
    collected.value = false
    entry.value = null // 与单词查词弹窗互斥
    // 选中后自动翻译并分析。
    translateSel()
    analyzeSel()
  }
}
function clearSel() {
  selectedText.value = ''
  selTranslation.value = null
  selAnalysis.value = null
  selMsg.value = ''
  collected.value = false
}
async function translateSel() {
  loadingTrans.value = true
  selTranslation.value = null
  selMsg.value = ''
  try {
    const resp = await api.post('/translations', { text: selectedText.value, target_lang: 'zh' })
    selTranslation.value = resp.data.output
  } catch (e) {
    selMsg.value = e instanceof ApiError ? e.message : '翻译失败'
    selMsgClass.value = 'error'
  } finally {
    loadingTrans.value = false
  }
}
async function analyzeSel() {
  loadingAnalysis.value = true
  selAnalysis.value = null
  selMsg.value = ''
  try {
    const resp = await api.post('/grammar/analysis', { text: selectedText.value })
    selAnalysis.value = resp.data.analysis
  } catch (e) {
    selMsg.value = e instanceof ApiError ? e.message : '分析失败'
    selMsgClass.value = 'error'
  } finally {
    loadingAnalysis.value = false
  }
}
async function collectSel() {
  selMsg.value = ''
  try {
    // 一并保存已生成的翻译与句子分析，方便在「收藏句子」中回看。
    await api.post('/sentences', {
      sentence: selectedText.value,
      translation: selTranslation.value?.translated_text || '',
      analysis: selAnalysis.value ? JSON.stringify(selAnalysis.value) : '',
    })
    collected.value = true
    selMsg.value = '已收藏句子'
    selMsgClass.value = 'ok'
  } catch (e) {
    selMsg.value = e instanceof ApiError ? e.message : '收藏失败'
    selMsgClass.value = 'error'
  }
}

const tokens = computed(() => {
  const text = article.value?.content || article.value?.summary || ''
  return text.split(/([^A-Za-z']+)/).map((t) => ({ text: t, isWord: /[A-Za-z]/.test(t) && t.length > 1 }))
})

async function loadArticle() {
  const id = route.params.id
  entry.value = null
  const resp = await api.get(`/articles/${id}`)
  article.value = resp.data
  document.querySelector('.content')?.scrollTo(0, 0)
  api.post(`/articles/${id}/read`, { finished: false }).catch(() => {})
}

// 组件被 keep-alive 缓存后，切换到不同文章时 onMounted 不再触发，靠 watch 重新加载。
watch(
  () => route.params.id,
  (id) => {
    if (id) loadArticle()
  },
)

async function lookup(word: string) {
  // 正在选中文本时，点击不触发单个单词查询。
  if ((window.getSelection()?.toString().trim().length || 0) > 0) return
  collectMsg.value = ''
  selectedText.value = '' // 与句子面板互斥
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

function goPrev() {
  if (article.value?.prev_id) router.push(`/articles/${article.value.prev_id}`)
}
function goNext() {
  if (article.value?.next_id) router.push(`/articles/${article.value.next_id}`)
}

onMounted(loadArticle)
</script>

<style scoped>
.article-body { line-height: 1.9; font-size: 15px; margin-top: 12px; white-space: pre-wrap; }
.word { cursor: pointer; border-radius: 3px; }
.word:hover { background: #fff3b0; }
.popup { position: sticky; bottom: 16px; border: 2px solid var(--primary); }
.nav-row { display: flex; align-items: center; }
.top-bar {
  display: flex;
  align-items: center;
  gap: 12px;
}
.top-bar .hint {
  flex: 1;
  text-align: center;
  font-size: 13px;
}
.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 1px solid var(--border);
  padding: 7px;
  border-radius: 8px;
  cursor: pointer;
  color: var(--text);
  transition: background 0.12s, color 0.12s, border-color 0.12s;
}
.icon-btn:hover:not(:disabled) { background: var(--primary); color: #fff; border-color: var(--primary); }
.icon-btn:disabled { opacity: 0.35; cursor: not-allowed; }
.sel-panel {
  position: sticky;
  bottom: 16px;
  border: 2px solid var(--primary);
  max-height: 55vh;
  overflow-y: auto;
}
.sel-text {
  background: #f5f7fa;
  border-left: 3px solid var(--primary);
  padding: 8px 10px;
  border-radius: 4px;
  margin: 8px 0;
}
.sel-result {
  margin-top: 10px;
  border-top: 1px solid var(--border);
  padding-top: 8px;
}
.sel-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  padding: 4px;
  border-radius: 8px;
  cursor: pointer;
  color: #97a0b3;
  transition: transform 0.12s, color 0.12s, background 0.12s;
}
.sel-icon-btn:hover { background: #eef2fb; transform: scale(1.1); }
.sel-icon-btn.lit { color: #f5b301; }
</style>
