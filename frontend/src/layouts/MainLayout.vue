<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand">
        <span class="brand-logo">
          <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.8"
            stroke-linecap="round" stroke-linejoin="round">
            <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" />
            <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z" />
          </svg>
        </span>
        英语学习助手
      </div>
      <nav>
        <template v-for="m in sortedMenu" :key="m.label">
          <RouterLink v-if="!m.children" :to="navTarget(m.path)" class="nav-link"
            :class="{ 'router-link-active': m.activePaths?.includes(route.path) }">
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
        <span class="avatar">{{ (auth.user?.username || '?').slice(0, 1).toUpperCase() }}</span>
        <span class="username">{{ auth.user?.username || '...' }}</span>
        <button class="logout-btn" title="退出登录" @click="onLogout">退出</button>
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
import { computed, onMounted, reactive, ref, watch } from 'vue'
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
  activePaths?: string[] // 额外高亮当前项的路由
}

const menu: NavItem[] = [
  { path: '/dictionary', label: '查单词' },
  { path: '/word-plans', label: '单词书', activePaths: ['/word-learning', '/review'] },
  {
    label: '我的收藏',
    children: [
      { path: '/vocabulary', label: '单词收藏' },
      { path: '/sentences', label: '句子收藏' },
    ],
  },
  { path: '/articles', label: '外刊阅读' },
  {
    label: '语言工具',
    children: [
      { path: '/translate', label: '智能翻译' },
      { path: '/grammar', label: '句子分析' },
      { path: '/essay', label: '作文批改' },
    ],
  },
  { path: '/conversation', label: 'AI 对话' },
  { path: '/training', label: '专项训练' },
  { path: '/history', label: '历史中心' },
]

// 按名字字数（不含空格）从少到多排列导航项。
const sortedMenu = computed(() =>
  [...menu].sort((a, b) => [...a.label.replace(/\s/g, '')].length - [...b.label.replace(/\s/g, '')].length),
)

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
  width: 220px;
  background: linear-gradient(180deg, #fbfcff 0%, #f4f7fd 100%);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  padding: 18px 14px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 17px;
  font-weight: 700;
  color: var(--primary);
  padding: 6px 8px 18px;
}
.brand-logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 9px;
  background: var(--primary);
  color: #fff;
  flex: 0 0 auto;
}

nav { display: flex; flex-direction: column; gap: 3px; overflow-y: auto; }
.nav-link {
  display: block;
  padding: 10px 12px;
  border-radius: 10px;
  color: #4a5160;
  font-size: 14px;
  line-height: 1.2;
  transition: background 0.14s, color 0.14s;
}
.nav-link:hover { background: #e8eefb; color: var(--primary); }
.nav-link.router-link-active {
  background: var(--primary);
  color: #fff;
  font-weight: 600;
  box-shadow: 0 4px 12px rgba(60, 110, 240, 0.28);
}

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
  color: #4a5160;
}
.group-toggle:hover { color: var(--primary); }
.chevron { transition: transform 0.18s; flex: 0 0 auto; opacity: 0.7; }
.group-toggle.open .chevron { transform: rotate(90deg); }

.subnav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 2px 0 2px 14px;
  padding-left: 8px;
  border-left: 1px solid var(--border);
}
.nav-link.sub { padding: 8px 12px; font-size: 13px; }

.user-box {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding: 10px;
  border-top: 1px solid var(--border);
}
.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--primary);
  color: #fff;
  font-size: 14px;
  font-weight: 700;
  flex: 0 0 auto;
}
.username {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  color: var(--text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.logout-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 32px;
  padding: 0 12px;
  border-radius: 8px;
  border: 1px solid var(--border);
  background: #fff;
  color: #e25555;
  font-size: 13px;
  cursor: pointer;
  flex: 0 0 auto;
  transition: background 0.14s, color 0.14s, border-color 0.14s;
}
.logout-btn:hover { background: #e25555; color: #fff; border-color: #e25555; }

.content { flex: 1; overflow-y: auto; padding: 24px 32px; }
</style>
