<template>
  <div>
    <h2 class="title">外刊阅读</h2>
    <div class="card">
      <div class="row">
        <input v-model="keyword" placeholder="按标题搜索" @keyup.enter="load" />
        <select v-model="difficulty" @change="load" style="width: 160px">
          <option value="">全部难度</option>
          <option value="beginner">初级</option>
          <option value="intermediate">中级</option>
          <option value="advanced">高级</option>
        </select>
        <button @click="load">搜索</button>
        <div class="spacer" />
        <span class="muted">共 {{ total }} 篇</span>
      </div>
    </div>

    <div class="card">
      <div v-for="a in items" :key="a.id" class="list-item" style="display:block; cursor:pointer" @click="open(a.id)">
        <div class="row">
          <b>{{ a.title }}</b>
          <span class="tag">{{ a.source_name }}</span>
          <div class="spacer" />
          <span class="muted">{{ a.published_at ? fmt(a.published_at) : '' }}</span>
        </div>
        <p v-if="a.summary" class="muted" style="margin: 4px 0">{{ truncate(a.summary) }}</p>
      </div>
      <p v-if="!items.length" class="muted">暂无文章。可运行 cmd/article-sync 同步 VOA 外刊。</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
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
const total = ref(0)
const keyword = ref('')
const difficulty = ref('')

function fmt(t: string) {
  return new Date(t).toLocaleDateString('zh-CN')
}
function truncate(s: string) {
  return s.length > 140 ? s.slice(0, 140) + '...' : s
}

async function load() {
  const q = new URLSearchParams()
  if (keyword.value) q.set('keyword', keyword.value)
  if (difficulty.value) q.set('difficulty', difficulty.value)
  const resp = await api.get(`/articles?${q.toString()}`)
  items.value = resp.data.items || []
  total.value = resp.data.total || 0
}
function open(id: number) {
  router.push(`/articles/${id}`)
}

onMounted(load)
</script>
