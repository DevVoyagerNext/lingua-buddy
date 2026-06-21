<template>
  <div class="auth-page">
    <div class="auth-shell">
      <div class="auth-hero">
        <div class="hero-content">
          <span class="hero-logo">
            <svg viewBox="0 0 24 24" width="26" height="26" fill="none" stroke="currentColor" stroke-width="1.8"
              stroke-linecap="round" stroke-linejoin="round">
              <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" />
              <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z" />
            </svg>
          </span>
          <h1>英语学习助手</h1>
          <p class="tagline">查词、背单词、外刊阅读、AI 对话，<br />一站式陪你高效提升英语。</p>
          <ul class="hero-features">
            <li><span class="fi">📚</span>智能查词与生词本</li>
            <li><span class="fi">📰</span>外刊阅读与句子分析</li>
            <li><span class="fi">💬</span>AI 情景对话练习</li>
          </ul>
        </div>
      </div>

      <div class="auth-form">
        <h2>欢迎回来 👋</h2>
        <p class="sub">登录以继续你的学习</p>

        <label class="field-label">账号</label>
        <div class="input-group">
          <span class="ig-icon">
            <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2"
              stroke-linecap="round" stroke-linejoin="round">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
              <circle cx="12" cy="7" r="4" />
            </svg>
          </span>
          <input v-model="login" placeholder="用户名或邮箱" @keyup.enter="onSubmit" />
        </div>

        <label class="field-label">密码</label>
        <div class="input-group">
          <span class="ig-icon">
            <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2"
              stroke-linecap="round" stroke-linejoin="round">
              <rect x="3" y="11" width="18" height="11" rx="2" />
              <path d="M7 11V7a5 5 0 0 1 10 0v4" />
            </svg>
          </span>
          <input v-model="password" type="password" placeholder="密码" @keyup.enter="onSubmit" />
        </div>

        <p v-if="error" class="auth-error">{{ error }}</p>
        <button class="auth-btn" :disabled="loading" @click="onSubmit">{{ loading ? '登录中…' : '登 录' }}</button>
        <p class="auth-alt">还没有账号？<RouterLink to="/register">立即注册</RouterLink></p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ApiError } from '@/api/client'

const auth = useAuthStore()
const router = useRouter()
const login = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function onSubmit() {
  error.value = ''
  if (!login.value || !password.value) {
    error.value = '请输入账号和密码'
    return
  }
  loading.value = true
  try {
    await auth.login(login.value, password.value)
    router.push('/dictionary')
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : '登录失败'
  } finally {
    loading.value = false
  }
}
</script>
