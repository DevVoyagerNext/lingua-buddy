<template>
  <div>
    <h2 class="title">生词本</h2>
    <div class="card">
      <div class="row">
        <input v-model="keyword" placeholder="搜索单词" @keyup.enter="load" style="max-width: 240px" />
        <select v-model="stage" @change="load" style="width: 160px">
          <option value="">全部阶段</option>
          <option value="recognition">初识</option>
          <option value="discrimination">辨认</option>
          <option value="spelling">默写</option>
          <option value="mastered">已掌握</option>
        </select>
        <button @click="load">搜索</button>
        <div class="spacer" />
        <span class="muted">共 {{ total }} 个</span>
      </div>
    </div>

    <div class="card">
      <div v-for="w in items" :key="w.id" class="list-item">
        <b>{{ w.word }}</b>
        <span class="tag">{{ stageLabel(w.stage) }}</span>
        <span class="muted">下次复习：{{ fmt(w.next_review_at) }}</span>
        <div class="spacer" />
        <button class="ghost small danger" @click="remove(w.id)">移出</button>
      </div>
      <p v-if="!items.length" class="muted">生词本是空的，去查词页收藏单词吧。</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

interface Word {
  id: number
  word: string
  stage: string
  mastery_label: string
  next_review_at: string
}

const items = ref<Word[]>([])
const total = ref(0)
const keyword = ref('')
const stage = ref('')

const stageNames: Record<string, string> = {
  recognition: '初识',
  discrimination: '辨认',
  spelling: '默写',
  mastered: '已掌握',
}
function stageLabel(s: string) {
  return stageNames[s] || s
}
function fmt(t: string) {
  return new Date(t).toLocaleString('zh-CN')
}

async function load() {
  const q = new URLSearchParams()
  if (keyword.value) q.set('keyword', keyword.value)
  if (stage.value) q.set('stage', stage.value)
  const resp = await api.get(`/vocabulary?${q.toString()}`)
  items.value = resp.data.items || []
  total.value = resp.data.total || 0
}

async function remove(id: number) {
  await api.delete(`/vocabulary/${id}`)
  await load()
}

onMounted(load)
</script>
