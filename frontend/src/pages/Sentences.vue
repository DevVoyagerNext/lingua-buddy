<template>
  <div>
    <h2 class="title">收藏句子</h2>
    <div class="card">
      <div class="row">
        <input v-model="keyword" placeholder="搜索句子内容" @keyup.enter="load" />
        <button @click="load">搜索</button>
      </div>
    </div>

    <div class="card">
      <div v-for="s in items" :key="s.id" class="sentence-item">
        <p style="font-size: 15px; margin: 4px 0">{{ s.sentence }}</p>
        <input v-model="s.translation" placeholder="中文翻译（可选）" style="margin: 4px 0" />
        <input v-model="s.note" placeholder="备注（可选）" style="margin: 4px 0" />
        <div class="row">
          <button class="small" @click="save(s)">保存</button>
          <button class="ghost small danger" @click="del(s.id)">取消收藏</button>
        </div>
      </div>
      <p v-if="!items.length" class="muted">还没有收藏句子。可在翻译、语法结果处收藏。</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

interface Sentence {
  id: number
  sentence: string
  translation: string | null
  note: string | null
}

const items = ref<Sentence[]>([])
const keyword = ref('')

async function load() {
  const q = keyword.value ? `?keyword=${encodeURIComponent(keyword.value)}` : ''
  const resp = await api.get(`/sentences${q}`)
  items.value = resp.data.items || []
}
async function save(s: Sentence) {
  await api.patch(`/sentences/${s.id}`, { translation: s.translation || '', note: s.note || '' })
}
async function del(id: number) {
  await api.delete(`/sentences/${id}`)
  await load()
}

onMounted(load)
</script>

<style scoped>
.sentence-item { padding: 12px 0; border-bottom: 1px solid var(--border); }
.sentence-item:last-child { border-bottom: none; }
</style>
