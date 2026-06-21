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
          <h1>开启你的英语之旅</h1>
          <p class="tagline">免费注册，立即解锁全部学习工具，<br />让每天的进步看得见。</p>
          <ul class="hero-features">
            <li><span class="fi">🎯</span>制定专属单词计划</li>
            <li><span class="fi">✍️</span>作文批改与润色</li>
            <li><span class="fi">🔊</span>语音输入与翻译</li>
          </ul>
        </div>
      </div>

      <div class="auth-form">
        <h2>创建账号 ✨</h2>
        <p class="sub">只需几秒，开始你的学习</p>

        <label class="field-label">用户名</label>
        <div class="input-group">
          <span class="ig-icon">
            <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2"
              stroke-linecap="round" stroke-linejoin="round">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
              <circle cx="12" cy="7" r="4" />
            </svg>
          </span>
          <input v-model="username" placeholder="3-30 位字母/数字/下划线" />
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
          <input v-model="password" type="password" placeholder="至少 8 位" @keyup.enter="onSubmit" />
        </div>

        <p v-if="error" class="auth-error">{{ error }}</p>
        <button class="auth-btn" :disabled="loading" @click="onSubmit">{{ loading ? '注册中…' : '注册并登录' }}</button>
        <p class="auth-alt">已有账号？<RouterLink to="/login">去登录</RouterLink></p>
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
const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function onSubmit() {
  error.value = ''
  loading.value = true
  try {
    await auth.register(username.value, password.value)
    router.push('/onboarding')
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : '注册失败'
  } finally {
    loading.value = false
  }
}
</script>
