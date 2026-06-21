<template>
  <div>
    <h2 class="title">学习首页</h2>

    <div v-if="hasPlan" class="grid">
      <div class="stat">
        <div class="num">{{ stats.totalWords }}</div>
        <div class="label">{{ planName }} · 总词数</div>
      </div>
      <div class="stat">
        <div class="num">{{ stats.learned }}/{{ stats.totalGroups }}</div>
        <div class="label">已学组 / 总组数</div>
      </div>
      <div class="stat">
        <div class="num">{{ stats.due }}</div>
        <div class="label">待复习组</div>
      </div>
      <div class="stat">
        <div class="num">{{ stats.newGroups }}</div>
        <div class="label">未学组</div>
      </div>
    </div>

    <div v-else class="card">
      <p class="muted">还没有进行中的单词书。</p>
      <RouterLink to="/word-plans"><button>去单词书选一本</button></RouterLink>
    </div>

    <div class="card" style="margin-top: 16px">
      <h3>快捷入口</h3>
      <div class="row" style="flex-wrap: wrap; gap: 8px">
        <RouterLink to="/word-learning"><button>开始学习单词</button></RouterLink>
        <RouterLink to="/word-plans"><button class="ghost">单词书</button></RouterLink>
        <RouterLink to="/dictionary"><button class="ghost">查单词</button></RouterLink>
        <RouterLink to="/articles"><button class="ghost">外刊阅读</button></RouterLink>
        <RouterLink to="/translate"><button class="ghost">智能翻译</button></RouterLink>
        <RouterLink to="/grammar"><button class="ghost">语法工具</button></RouterLink>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onActivated } from 'vue'
import { api, ApiError } from '@/api/client'

const hasPlan = ref(false)
const planName = ref('')
const stats = reactive({ totalWords: 0, totalGroups: 0, learned: 0, due: 0, newGroups: 0 })

onActivated(async () => {
  try {
    const resp = await api.get('/word-learning/groups')
    const d = resp.data
    hasPlan.value = true
    planName.value = d.plan_name
    stats.totalWords = d.total_words
    stats.totalGroups = d.total_groups
    const groups = d.groups || []
    stats.due = groups.filter((g: any) => g.status === 'due').length
    stats.newGroups = groups.filter((g: any) => g.status === 'new').length
    stats.learned = stats.totalGroups - stats.newGroups // 学过的（含待复习）
  } catch (e) {
    hasPlan.value = !(e instanceof ApiError && e.code === 'NO_ACTIVE_PLAN')
  }
})
</script>
