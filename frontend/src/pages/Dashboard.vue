<template>
  <div>
    <h2 class="title">学习首页</h2>
    <div class="grid">
      <div class="stat">
        <div class="num">{{ due.due_count ?? 0 }}</div>
        <div class="label">今日待复习单词</div>
      </div>
      <div class="stat">
        <div class="num">{{ due.counts?.learning ?? 0 }}</div>
        <div class="label">学习中单词</div>
      </div>
      <div class="stat">
        <div class="num">{{ due.counts?.waiting ?? 0 }}</div>
        <div class="label">计划等待词</div>
      </div>
      <div class="stat">
        <div class="num">{{ due.counts?.first_mastered ?? 0 }}</div>
        <div class="label">已掌握</div>
      </div>
    </div>

    <div class="card" style="margin-top: 16px">
      <h3>快捷入口</h3>
      <div class="row" style="flex-wrap: wrap; gap: 8px">
        <RouterLink to="/word-learning"><button>开始学习单词</button></RouterLink>
        <RouterLink to="/review"><button class="ghost">到期复习</button></RouterLink>
        <RouterLink to="/dictionary"><button class="ghost">查单词</button></RouterLink>
        <RouterLink to="/translate"><button class="ghost">智能翻译</button></RouterLink>
        <RouterLink to="/grammar"><button class="ghost">语法工具</button></RouterLink>
        <RouterLink to="/speech"><button class="ghost">语音学习</button></RouterLink>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

const due = ref<any>({})

onMounted(async () => {
  try {
    const resp = await api.get('/word-learning/due')
    due.value = resp.data || {}
  } catch {
    due.value = {}
  }
})
</script>
