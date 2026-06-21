<template>
  <div class="card" style="max-width: 520px">
    <h2 class="title">设置你的英语水平</h2>
    <p class="muted">用于调整 AI 解释深度、例句难度与练习难度。</p>
    <div class="col" style="margin-top: 16px">
      <button
        v-for="lv in levels"
        :key="lv.value"
        class="option-btn"
        :class="{ correct: selected === lv.value }"
        @click="selected = lv.value"
      >
        {{ lv.label }}
      </button>
      <p v-if="error" class="error">{{ error }}</p>
      <button :disabled="loading || !selected" @click="onSave">{{ loading ? '保存中...' : '保存并开始' }}</button>
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
const selected = ref('')
const error = ref('')
const loading = ref(false)

const levels = [
  { value: 'beginner', label: '初级' },
  { value: 'intermediate', label: '中级' },
  { value: 'advanced', label: '高级' },
  { value: 'cet4', label: 'CET-4（四级）' },
  { value: 'cet6', label: 'CET-6（六级）' },
]

async function onSave() {
  error.value = ''
  loading.value = true
  try {
    await auth.updateProfile({ english_level: selected.value })
    router.push('/dashboard')
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : '保存失败'
  } finally {
    loading.value = false
  }
}
</script>
