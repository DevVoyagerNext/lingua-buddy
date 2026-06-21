<template>
  <div class="auth-wrap">
    <div class="card auth-card">
      <h2 class="title">注册 Lingua Buddy</h2>
      <div class="col">
        <input v-model="username" placeholder="用户名（3-30 位字母/数字/下划线）" />
        <input v-model="email" placeholder="邮箱（可选）" />
        <input v-model="password" type="password" placeholder="密码（至少 8 位）" @keyup.enter="onSubmit" />
        <p v-if="error" class="error">{{ error }}</p>
        <button :disabled="loading" @click="onSubmit">{{ loading ? '注册中...' : '注册并登录' }}</button>
        <p class="muted">已有账号？<RouterLink to="/login">去登录</RouterLink></p>
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
const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function onSubmit() {
  error.value = ''
  loading.value = true
  try {
    await auth.register(username.value, password.value, email.value)
    router.push('/onboarding')
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : '注册失败'
  } finally {
    loading.value = false
  }
}
</script>
