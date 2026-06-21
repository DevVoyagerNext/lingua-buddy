<template>
  <div class="auth-wrap">
    <div class="card auth-card">
      <h2 class="title">登录英语学习助手</h2>
      <div class="col">
        <input v-model="login" placeholder="用户名或邮箱" @keyup.enter="onSubmit" />
        <input v-model="password" type="password" placeholder="密码" @keyup.enter="onSubmit" />
        <p v-if="error" class="error">{{ error }}</p>
        <button :disabled="loading" @click="onSubmit">{{ loading ? '登录中...' : '登录' }}</button>
        <p class="muted">还没有账号？<RouterLink to="/register">去注册</RouterLink></p>
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
