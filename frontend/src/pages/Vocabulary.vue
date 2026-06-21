<template>
  <div>
    <h2 class="title">Lingua Buddy收藏单词</h2>
    <div class="card">
      <div class="row">
        <div class="search">
          <input v-model="keyword" placeholder="搜索单词" @keyup.enter="load" />
          <button class="icon-btn search-btn" title="搜索" @click="load">
            <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2"
              stroke-linecap="round" stroke-linejoin="round">
              <circle cx="11" cy="11" r="7" />
              <line x1="21" y1="21" x2="16.65" y2="16.65" />
            </svg>
          </button>
        </div>
        <div class="spacer" />
        <span class="muted">共 {{ total }} 个</span>
      </div>
    </div>

    <div class="card word-list">
      <div v-for="w in items" :key="w.id" class="word-row">
        <button class="star-btn" :class="{ lit: starred[w.id] !== false }"
          :title="starred[w.id] !== false ? '取消收藏' : '收藏'" @click="toggle(w.id)">
          <svg viewBox="0 0 24 24" width="22" height="22" stroke="currentColor" stroke-width="1.6"
            stroke-linejoin="round" :fill="starred[w.id] !== false ? 'currentColor' : 'none'">
            <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
          </svg>
        </button>
        <span class="word-text">{{ w.word }}</span>
        <span class="word-def">{{ w.definition || '暂无释义' }}</span>
      </div>
      <p v-if="!items.length" class="muted empty">收藏单词是空的，去查词页收藏单词吧。</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onActivated } from 'vue'
import { api } from '@/api/client'

interface Word {
  id: number
  word: string
  definition: string
}

const items = ref<Word[]>([])
const total = ref(0)
const keyword = ref('')
// 记录每个单词当前是否点亮（已收藏）。undefined / true 视为点亮。
const starred = reactive<Record<number, boolean>>({})

async function load() {
  const q = new URLSearchParams()
  if (keyword.value) q.set('keyword', keyword.value)
  const resp = await api.get(`/vocabulary?${q.toString()}`)
  items.value = resp.data.items || []
  total.value = resp.data.total || 0
  for (const w of items.value) starred[w.id] = true
}

async function toggle(id: number) {
  // 点击点亮的星星 = 取消收藏，调用后端移除并刷新列表。
  starred[id] = false
  await api.delete(`/vocabulary/${id}`)
  await load()
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
  width: 260px;
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
.search-btn {
  position: absolute;
  right: 4px;
  color: var(--primary);
}

.word-list { padding: 4px 0; }
.word-row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
  transition: background 0.12s;
}
.word-row:last-child { border-bottom: none; }
.word-row:hover { background: #f7f9ff; }

.star-btn {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  cursor: pointer;
  padding: 2px;
  color: #cbd2e0;
  transition: transform 0.12s, color 0.12s;
}
.star-btn.lit { color: #f5b301; }
.star-btn:hover { transform: scale(1.15); }

.word-text {
  flex: 0 0 160px;
  font-size: 16px;
  font-weight: 700;
  color: var(--text);
  word-break: break-word;
}
.word-def {
  flex: 1;
  font-size: 13px;
  line-height: 1.6;
  color: #667;
  white-space: pre-line;
}
.empty { text-align: center; padding: 24px; }
</style>
