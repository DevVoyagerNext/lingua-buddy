<template>
  <div class="card" style="max-width: 520px">
    <h2 class="title">个人设置</h2>
    <div class="col">
      <label class="muted">用户名</label>
      <input :value="auth.user?.username" disabled />
      <label class="muted">邮箱</label>
      <input v-model="email" placeholder="邮箱（可选）" />
      <label class="muted">英语水平</label>
      <select v-model="level">
        <option v-for="lv in levels" :key="lv.value" :value="lv.value">{{ lv.label }}</option>
      </select>
      <p v-if="msg" :class="msgClass">{{ msg }}</p>
      <button :disabled="loading" @click="onSave">{{ loading ? '保存中...' : '保存' }}</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { ApiError } from '@/api/client'

const auth = useAuthStore()
const email = ref('')
const level = ref('intermediate')
const msg = ref('')
const msgClass = ref('ok')
const loading = ref(false)

const levels = [
  { value: 'beginner', label: '初级' },
  { value: 'intermediate', label: '中级' },
  { value: 'advanced', label: '高级' },
  { value: 'cet4', label: 'CET-4' },
  { value: 'cet6', label: 'CET-6' },
]

onMounted(async () => {
  const u = auth.user || (await auth.fetchMe())
  email.value = u.email || ''
  level.value = u.english_level || 'intermediate'
})

async function onSave() {
  msg.value = ''
  loading.value = true
  try {
    await auth.updateProfile({ email: email.value, english_level: level.value })
    msg.value = '已保存'
    msgClass.value = 'ok'
  } catch (e) {
    msg.value = e instanceof ApiError ? e.message : '保存失败'
    msgClass.value = 'error'
  } finally {
    loading.value = false
  }
}
</script>
