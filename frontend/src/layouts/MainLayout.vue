<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand">英语学习助手</div>
      <nav>
        <template v-for="m in menu" :key="m.label">
          <RouterLink v-if="!m.children" :to="navTarget(m.path)" class="nav-link">
            {{ m.label }}
          </RouterLink>
          <template v-else>
            <button class="nav-link group-toggle" :class="{ open: openGroups[m.label] }"
              @click="toggleGroup(m.label)">
              <span>{{ m.label }}</span>
              <svg class="chevron" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor"
                stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="9 18 15 12 9 6" />
              </svg>
            </button>
            <div v-show="openGroups[m.label]" class="subnav">
              <RouterLink v-for="c in m.children" :key="c.path" :to="c.path" class="nav-link sub">
                {{ c.label }}
              </RouterLink>
            </div>
          </template>
        </template>
      </nav>
      <div class="spacer" />
      <div class="user-box">
        <div class="muted">{{ auth.user?.username || '...' }}</div>
        <button class="ghost small" @click="onLogout">退出登录</button>
      </div>
    </aside>
    <main class="content">
      <RouterView v-slot="{ Component }">
        <keep-alive>
          <component :is="Component" />
        </keep-alive>
      </RouterView>
    </main>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

// 记住「外刊阅读」最后停留的位置（列表或某篇文章），返回时恢复。
const lastArticlePath = ref('/articles')
watch(
  () => route.fullPath,
  (path) => {
    if (path.startsWith('/articles')) lastArticlePath.value = path
  },
  { immediate: true },
)

// 「外刊阅读」导航跳到上次阅读的文章，其余导航维持原路径。
function navTarget(path?: string) {
  return path === '/articles' ? lastArticlePath.value : path || ''
}

interface NavChild {
  path: string
  label: string
}
interface NavItem {
  path?: string
  label: string
  children?: NavChild[]
}

const menu: NavItem[] = [
  { path: '/dictionary', label: '查单词' },
  { path: '/word-plans', label: '单词书' },
  { path: '/word-learning', label: '单词学习' },
  {
    label: '收藏',
    children: [
      { path: '/vocabulary', label: '单词收藏' },
      { path: '/sentences', label: '句子收藏' },
    ],
  },
  { path: '/articles', label: '外刊阅读' },
  {
    label: '翻译与语法',
    children: [
      { path: '/translate', label: '智能翻译' },
      { path: '/grammar', label: '语法工具' },
    ],
  },
  { path: '/conversation', label: 'AI 对话' },
  { path: '/essay', label: '作文批改' },
  { path: '/training', label: '专项训练' },
  { path: '/history', label: '历史中心' },
  { path: '/settings', label: '个人设置' },
]

// 各分组的展开状态。
const openGroups = reactive<Record<string, boolean>>({})
function toggleGroup(label: string) {
  openGroups[label] = !openGroups[label]
}

// 当前路由命中某个分组的子项时，自动展开该分组。
watch(
  () => route.path,
  (path) => {
    for (const m of menu) {
      if (m.children?.some((c) => c.path === path)) openGroups[m.label] = true
    }
  },
  { immediate: true },
)

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

.group-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  background: transparent;
  border: none;
  cursor: pointer;
  font-size: 14px;
  font-family: inherit;
  text-align: left;
}
.chevron { transition: transform 0.18s; flex: 0 0 auto; }
.group-toggle.open .chevron { transform: rotate(90deg); }

.subnav { display: flex; flex-direction: column; gap: 2px; }
.nav-link.sub { padding-left: 26px; font-size: 13px; }
.user-box { display: flex; flex-direction: column; gap: 6px; padding: 10px; }
.content { flex: 1; overflow-y: auto; padding: 24px 32px; }
</style>
