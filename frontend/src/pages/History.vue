<template>
  <div>
    <h2 class="title">历史中心</h2>
    <div class="card">
      <div class="row">
        <select v-model="type" @change="load" style="width: 180px">
          <option value="">全部类型</option>
          <option value="translation">翻译</option>
          <option value="speech">语音识别</option>
          <option value="grammar_analysis">语法分析</option>
          <option value="correction">语法纠错</option>
          <option value="essay">作文批改</option>
        </select>
        <div class="spacer" />
        <span class="muted">共 {{ total }} 条</span>
      </div>
    </div>

    <div class="card">
      <div v-for="h in items" :key="h.id" class="hist-item">
        <div class="row">
          <span class="tag">{{ typeLabel(h.record_type) }}</span>
          <span class="muted">{{ fmt(h.created_at) }}</span>
          <div class="spacer" />
          <button class="ghost small danger" @click="del(h.id)">删除</button>
        </div>
        <p style="margin: 6px 0"><b>输入：</b>{{ h.input_text }}</p>
        <p v-if="h.result_text" class="muted"><b>结果：</b>{{ truncate(h.result_text) }}</p>
      </div>
      <p v-if="!items.length" class="muted">暂无历史记录。</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

interface Hist {
  id: number
  record_type: string
  input_text: string
  result_text: string
  created_at: string
}

const items = ref<Hist[]>([])
const total = ref(0)
const type = ref('')

const labels: Record<string, string> = {
  translation: '翻译',
  speech: '语音识别',
  grammar_analysis: '语法分析',
  correction: '语法纠错',
  essay: '作文批改',
}
function typeLabel(t: string) {
  return labels[t] || t
}
function fmt(t: string) {
  return new Date(t).toLocaleString('zh-CN')
}
function truncate(s: string) {
  return s.length > 120 ? s.slice(0, 120) + '...' : s
}

async function load() {
  const q = type.value ? `?type=${type.value}` : ''
  const resp = await api.get(`/history${q}`)
  items.value = resp.data.items || []
  total.value = resp.data.total || 0
}
async function del(id: number) {
  await api.delete(`/history/${id}`)
  await load()
}

onMounted(load)
</script>

<style scoped>
.hist-item { padding: 12px 0; border-bottom: 1px solid var(--border); }
.hist-item:last-child { border-bottom: none; }
</style>
