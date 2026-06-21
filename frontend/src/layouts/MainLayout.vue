<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand">Lingua Buddy</div>
      <nav>
        <RouterLink v-for="m in menu" :key="m.path" :to="m.path" class="nav-link">
          {{ m.label }}
        </RouterLink>
      </nav>
      <div class="spacer" />
      <div class="user-box">
        <div class="muted">{{ auth.user?.username || '...' }}</div>
        <button class="ghost small" @click="onLogout">退出登录</button>
      </div>
    </aside>
    <main class="content">
      <RouterView />
    </main>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

const menu = [
  { path: '/dashboard', label: '学习首页' },
  { path: '/dictionary', label: '查单词' },
  { path: '/word-plans', label: '词汇计划' },
  { path: '/word-learning', label: '单词学习' },
  { path: '/review', label: '到期复习' },
  { path: '/vocabulary', label: '生词本' },
  { path: '/sentences', label: '收藏句子' },
  { path: '/translate', label: '智能翻译' },
  { path: '/speech', label: '语音学习' },
  { path: '/grammar', label: '语法工具' },
  { path: '/history', label: '历史中心' },
  { path: '/settings', label: '个人设置' },
]

async function onLogout() {
  auth.logout()
  router.push('/login')
}

onMounted(async () => {
  if (!auth.user) {
    try {
      await auth.fetchMe()
    } catch {
      // 401 由拦截器处理
    }
  }
})
</script>

<style scoped>
.layout { display: flex; height: 100vh; }
.sidebar {
  width: 200px;
  background: #fff;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  padding: 16px 12px;
}
.brand { font-size: 18px; font-weight: 700; color: var(--primary); padding: 8px 10px 16px; }
nav { display: flex; flex-direction: column; gap: 2px; }
.nav-link {
  padding: 9px 12px;
  border-radius: 8px;
  color: var(--text);
  font-size: 14px;
}
.nav-link:hover { background: #eef2fb; }
.nav-link.router-link-active { background: var(--primary); color: #fff; }
.user-box { display: flex; flex-direction: column; gap: 6px; padding: 10px; }
.content { flex: 1; overflow-y: auto; padding: 24px 32px; }
</style>
