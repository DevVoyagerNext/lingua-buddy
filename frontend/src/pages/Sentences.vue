<template>
  <div>
    <h2 class="title">Lingua Buddy收藏句子</h2>
    <div class="card">
      <div class="row">
        <div class="search">
          <input v-model="keyword" placeholder="搜索句子内容" @keyup.enter="load" />
          <button class="icon-btn search-btn" title="搜索" @click="load">
            <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2"
              stroke-linecap="round" stroke-linejoin="round">
              <circle cx="11" cy="11" r="7" />
              <line x1="21" y1="21" x2="16.65" y2="16.65" />
            </svg>
          </button>
        </div>
        <div class="spacer" />
        <span class="muted">共 {{ items.length }} 句</span>
      </div>
    </div>

    <div v-for="s in items" :key="s.id" class="card sentence-card">
      <div class="sentence-head">
        <button class="star-btn lit" title="取消收藏" @click="del(s.id)">
          <svg viewBox="0 0 24 24" width="22" height="22" fill="currentColor" stroke="currentColor" stroke-width="1.6"
            stroke-linejoin="round">
            <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
          </svg>
        </button>
        <p class="sentence-en">{{ s.sentence }}</p>
      </div>

      <p v-if="s.translation" class="block block-body">
        <span class="block-label">翻译</span>{{ s.translation }}
      </p>

      <button v-if="s._a" class="expand-btn" :class="{ open: expanded[s.id] }" @click="toggle(s.id)">
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2"
          stroke-linecap="round" stroke-linejoin="round" class="chevron">
          <polyline points="6 9 12 15 18 9" />
        </svg>
        <span>{{ expanded[s.id] ? '收起句子分析' : '查看句子分析' }}</span>
      </button>

      <div v-if="s._a && expanded[s.id]" class="block">
        <span class="block-label">句子分析</span>
        <div class="chips">
          <span v-if="s._a.sentence_type" class="chip">句型：{{ s._a.sentence_type }}</span>
          <span v-if="s._a.tense" class="chip">时态：{{ s._a.tense }}</span>
          <span v-if="s._a.voice" class="chip">语态：{{ s._a.voice }}</span>
        </div>
        <p v-if="s._a.main_clause" class="muted main-clause">
          主干：主语「{{ s._a.main_clause.subject }}」谓语「{{ s._a.main_clause.predicate }}」宾语「{{ s._a.main_clause.object }}」
        </p>
        <div v-if="s._a.grammar_points?.length" class="points">
          <p v-for="(g, i) in s._a.grammar_points" :key="i" class="point">
            <b>{{ g.name }}</b><span class="muted">{{ g.explanation_zh }}</span>
          </p>
        </div>
        <p v-if="s._a.explanation_zh" class="muted explain">{{ s._a.explanation_zh }}</p>
      </div>
    </div>

    <div v-if="!items.length" class="card empty muted">
      还没有收藏句子。可在外刊阅读、翻译、语法结果处点击收藏。
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onActivated } from 'vue'
import { api } from '@/api/client'

interface Analysis {
  sentence_type?: string
  tense?: string
  voice?: string
  main_clause?: { subject: string; predicate: string; object: string }
  grammar_points?: { name: string; explanation_zh: string }[]
  explanation_zh?: string
}
interface Sentence {
  id: number
  sentence: string
  translation: string | null
  analysis: string | null
  _a?: Analysis | null
}

const items = ref<Sentence[]>([])
const keyword = ref('')
// 记录每个句子的句子分析是否展开。
const expanded = reactive<Record<number, boolean>>({})

function toggle(id: number) {
  expanded[id] = !expanded[id]
}

function parseAnalysis(raw: string | null): Analysis | null {
  if (!raw) return null
  try {
    const a = JSON.parse(raw)
    return a && typeof a === 'object' ? a : null
  } catch {
    return null
  }
}

async function load() {
  const q = keyword.value ? `?keyword=${encodeURIComponent(keyword.value)}` : ''
  const resp = await api.get(`/sentences${q}`)
  const list: Sentence[] = resp.data.items || []
  for (const s of list) s._a = parseAnalysis(s.analysis)
  items.value = list
}
async function del(id: number) {
  await api.delete(`/sentences/${id}`)
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

.sentence-card {
  transition: box-shadow 0.15s;
}
.sentence-card:hover { box-shadow: 0 4px 16px rgba(0, 0, 0, 0.07); }

.sentence-head {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}
.sentence-en {
  flex: 1;
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  line-height: 1.5;
  color: var(--text);
}
.star-btn {
  flex: 0 0 auto;
  background: transparent;
  border: none;
  cursor: pointer;
  padding: 2px;
  color: #cbd2e0;
  transition: transform 0.12s;
}
.star-btn.lit { color: #f5b301; }
.star-btn:hover { transform: scale(1.15); }

.block {
  margin-top: 8px;
}
.block-label {
  display: inline-block;
  font-size: 12px;
  font-weight: 700;
  color: var(--primary);
  background: #eef2fb;
  padding: 2px 8px;
  border-radius: 6px;
  margin-right: 8px;
}
.block-body { font-size: 15px; line-height: 1.6; }

.expand-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 8px;
  padding: 4px 8px;
  background: transparent;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  color: var(--primary);
}
.expand-btn:hover { background: #eef2fb; }
.chevron { transition: transform 0.18s; }
.expand-btn.open .chevron { transform: rotate(180deg); }

.chips { display: flex; flex-wrap: wrap; gap: 8px; margin: 4px 0 8px; }
.chip {
  font-size: 12px;
  color: #475;
  background: #f0f4f0;
  border: 1px solid #e0e8e0;
  padding: 2px 10px;
  border-radius: 999px;
}
.main-clause { margin: 4px 0; font-size: 13px; }
.points { margin: 6px 0; }
.point { margin: 4px 0; font-size: 13px; line-height: 1.5; }
.point b { color: var(--text); margin-right: 6px; }
.explain { margin-top: 6px; font-size: 13px; line-height: 1.6; }
.empty { text-align: center; padding: 28px; }
</style>
