<template>
  <div>
    <h2 class="title">外刊阅读</h2>

    <div class="card">
      <div class="row">
        <div class="search">
          <input v-model="keyword" placeholder="搜索标题" @keyup.enter="load" />
          <button class="icon-btn search-btn" title="搜索" @click="load">
            <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2"
              stroke-linecap="round" stroke-linejoin="round">
              <circle cx="11" cy="11" r="7" />
              <line x1="21" y1="21" x2="16.65" y2="16.65" />
            </svg>
          </button>
        </div>
        <button v-if="keyword" class="icon-btn" title="清除" @click="clearSearch">
          <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2"
            stroke-linecap="round" stroke-linejoin="round">
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </button>
      </div>
    </div>

    <div class="card article-list">
      <div v-for="a in items" :key="a.id" class="article-row" @click="open(a.id)">
        <span class="book-icon">
          <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="1.6"
            stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 6.5C10.5 5 8 4.2 5.5 4.5A1 1 0 0 0 4.6 5.5v11.4a1 1 0 0 0 1.1 1c2.2-.3 4.6.4 6.3 1.8 1.7-1.4 4.1-2.1 6.3-1.8a1 1 0 0 0 1.1-1V5.5a1 1 0 0 0-.9-1C16 4.2 13.5 5 12 6.5z" fill="currentColor" fill-opacity="0.12" />
            <path d="M12 6.5v12.4" />
          </svg>
        </span>
        <div class="article-main">
          <div class="article-title-row">
            <b class="article-title">{{ a.title }}</b>
            <span v-if="a.published_at" class="muted date">{{ fmtDate(a.published_at) }}</span>
          </div>
          <p v-if="a.summary" class="muted summary">{{ truncate(a.summary) }}</p>
        </div>
      </div>
      <p v-if="!items.length" class="muted empty">暂无文章，运行 cmd/article-sync 同步最新外刊。</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '@/api/client'

interface Article {
  id: number
  title: string
  summary: string | null
  source_name: string
  published_at: string | null
}

const router = useRouter()
const items = ref<Article[]>([])
const keyword = ref('')

function fmtDate(t: string) {
  return new Date(t).toLocaleDateString('zh-CN')
}
function truncate(s: string) {
  return s.length > 140 ? s.slice(0, 140) + '...' : s
}

async function load() {
  const q = new URLSearchParams()
  if (keyword.value) q.set('keyword', keyword.value)
  q.set('page_size', '100')
  const resp = await api.get(`/articles?${q.toString()}`)
  items.value = resp.data.items || []
}
function clearSearch() {
  keyword.value = ''
  load()
}
function open(id: number) {
  router.push(`/articles/${id}`)
}

onActivated(load)
</script>

<style scoped>
.search {
  position: relative;
  display: flex;
  align-items: center;
}
.search input {
  width: 280px;
  padding-right: 40px;
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

.article-list { padding: 4px 0; }
.article-row {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--border);
  cursor: pointer;
  transition: background 0.12s;
}
.article-row:last-child { border-bottom: none; }
.article-row:hover { background: #f7f9ff; }

.book-icon {
  flex: 0 0 auto;
  display: inline-flex;
  margin-top: 2px;
  color: var(--primary);
}
.article-main { flex: 1; min-width: 0; }
.article-title-row {
  display: flex;
  align-items: baseline;
  gap: 12px;
}
.article-title {
  flex: 1;
  font-size: 15px;
  color: var(--text);
  line-height: 1.5;
}
.date { flex: 0 0 auto; font-size: 12px; white-space: nowrap; }
.summary {
  margin: 6px 0 0;
  font-size: 13px;
  line-height: 1.6;
}
.empty { text-align: center; padding: 24px; }
</style>
