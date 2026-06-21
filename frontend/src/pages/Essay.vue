<template>
  <div>
    <h2 class="title">作文批改</h2>
    <div class="card">
      <input v-model="title" placeholder="作文标题" style="margin-bottom: 8px" />
      <div class="row" style="margin-bottom: 8px">
        <select v-model="essayType" style="width: 160px">
          <option value="argumentative">议论文</option>
          <option value="narrative">记叙文</option>
          <option value="expository">说明文</option>
          <option value="letter">书信</option>
        </select>
        <select v-model="targetExam" style="width: 160px">
          <option value="">不限考试</option>
          <option value="cet4">四级</option>
          <option value="cet6">六级</option>
        </select>
      </div>
      <textarea v-model="body" rows="8" placeholder="在此输入作文正文..."></textarea>
      <div class="row" style="margin-top: 10px">
        <button :disabled="loading" @click="submit">{{ loading ? 'AI 批改中...' : '提交批改' }}</button>
        <span v-if="error" class="error">{{ error }}</span>
      </div>
    </div>

    <div v-if="review" class="card">
      <h3>批改结果 <span class="muted" style="font-size:13px">（AI 生成，仅供学习参考）</span></h3>
      <p>{{ review.overall_comment }}</p>
      <div class="row" style="gap: 16px; flex-wrap: wrap">
        <span class="tag">语法 {{ review.scores.grammar }}</span>
        <span class="tag">词汇 {{ review.scores.vocabulary }}</span>
        <span class="tag">结构 {{ review.scores.structure }}</span>
        <span class="tag">连贯 {{ review.scores.coherence }}</span>
      </div>
      <div v-if="review.issues?.length" style="margin-top: 12px">
        <h4>问题与建议</h4>
        <div v-for="(it, i) in review.issues" :key="i" style="margin-bottom: 8px">
          <span class="tag">{{ it.type }}</span>
          <span class="error">{{ it.original }}</span> → <span class="ok">{{ it.replacement }}</span>
          <p class="muted" style="margin: 2px 0">{{ it.explanation_zh }}</p>
        </div>
      </div>
      <div style="margin-top: 12px">
        <h4>修改后参考</h4>
        <p style="white-space: pre-wrap">{{ review.revised_text }}</p>
        <p class="muted">{{ review.revision_reason }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { api, ApiError } from '@/api/client'

const title = ref('')
const body = ref('')
const essayType = ref('argumentative')
const targetExam = ref('')
const loading = ref(false)
const error = ref('')
const review = ref<any>(null)

async function submit() {
  error.value = ''
  if (!body.value.trim()) {
    error.value = '请输入作文正文'
    return
  }
  loading.value = true
  try {
    const resp = await api.post('/essays/review', {
      submission_id: crypto.randomUUID(),
      title: title.value,
      body: body.value,
      essay_type: essayType.value,
      target_exam: targetExam.value,
    })
    review.value = resp.data.review
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : '批改失败'
  } finally {
    loading.value = false
  }
}
</script>
